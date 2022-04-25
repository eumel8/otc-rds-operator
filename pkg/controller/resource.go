package controller

// factory otc resources here

import (
	"context"
	"errors"
	"fmt"

	// "github.com/jinzhu/copier"
	"net/http"
	"os"
	"time"

	"github.com/gophercloud/utils/client"
	golangsdk "github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/networking/v1/subnets"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/networking/v1/vpcs"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/networking/v2/extensions/security/groups"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/rds/v3/backups"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/rds/v3/instances"

	"k8s.io/client-go/rest"

	rdsv1alpha1 "github.com/eumel8/otc-rds-operator/pkg/rds/v1alpha1"
	rdsv1alpha1clientset "github.com/eumel8/otc-rds-operator/pkg/rds/v1alpha1/apis/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// workaround https://github.com/opentelekomcloud/gophertelekomcloud/issues/342
type myRDSRestartOpts struct {
	Restart struct{} `json:"restart"`
}

func (c *Controller) secgroupGet(client *golangsdk.ServiceClient, opts *groups.ListOpts) (*groups.SecGroup, error) {
	c.logger.Debug("secgroupGet")
	pages, err := groups.List(client, *opts).AllPages()
	if err != nil {
		return nil, err
	}
	n, err := groups.ExtractGroups(pages)
	if len(n) == 0 {
		err := errors.New("no secgroup found")
		return nil, err
	}

	return &n[0], nil
}

func (c *Controller) subnetGet(client *golangsdk.ServiceClient, opts *subnets.ListOpts) (*subnets.Subnet, error) {
	c.logger.Debug("subnetGet")
	n, err := subnets.List(client, *opts)
	if err != nil {
		return nil, err
	}
	if len(n) == 0 {
		err := errors.New("no subnet found")
		return nil, err
	}

	return &n[0], nil
}

func (c *Controller) vpcGet(client *golangsdk.ServiceClient, opts *vpcs.ListOpts) (*vpcs.Vpc, error) {
	c.logger.Debug("vpcGet")
	n, err := vpcs.List(client, *opts)
	if err != nil {
		return nil, err
	}

	if len(n) == 0 {
		err := errors.New("no vpc found")
		return nil, err
	}

	return &n[0], nil
}

func (c *Controller) rdsGetById(client *golangsdk.ServiceClient, rdsId string) (*instances.RdsInstanceResponse, error) {
	c.logger.Debug("rdsGetById %s", rdsId)
	listOpts := instances.ListRdsInstanceOpts{
		Id: rdsId,
	}
	allPages, err := instances.List(client, listOpts).AllPages()
	if err != nil {
		return nil, err
	}

	n, err := instances.ExtractRdsInstances(allPages)
	if err != nil {
		return nil, err
	}
	if len(n.Instances) == 0 {
		return nil, nil
	}
	return &n.Instances[0], nil
}

func (c *Controller) rdsGetByName(client *golangsdk.ServiceClient, rdsName string) (*instances.RdsInstanceResponse, error) {
	c.logger.Debug("rdsGetByName %s", rdsName)
	listOpts := instances.ListRdsInstanceOpts{
		Name: rdsName,
	}
	allPages, err := instances.List(client, listOpts).AllPages()
	if err != nil {
		return nil, err
	}

	n, err := instances.ExtractRdsInstances(allPages)
	if err != nil {
		return nil, err
	}
	if len(n.Instances) == 0 {
		return nil, nil
	}
	return &n.Instances[0], nil
}

func (c *Controller) rdsCreate(ctx context.Context, netclient1 *golangsdk.ServiceClient, netclient2 *golangsdk.ServiceClient, client *golangsdk.ServiceClient, opts *instances.CreateRdsOpts, newRds *rdsv1alpha1.Rds) error {
	c.logger.Debug("rdsCreate ", newRds.Name)
	c.recorder.Eventf(newRds, rdsv1alpha1.EventTypeNormal, "Create", "This instance is creating.")
	rdsCheck, err := c.rdsGetByName(client, newRds.Name)
	if rdsCheck != nil {
		err := fmt.Errorf("rds already exists %s", newRds.Name)
		return err
	}
	if err != nil {
		err := fmt.Errorf("error checking rds status %v", err)
		return err
	}

	g, err := c.secgroupGet(netclient2, &groups.ListOpts{Name: newRds.Spec.Securitygroup})
	if err != nil {
		err := fmt.Errorf("error getting secgroup state: %v", err)
		return err
	}

	s, err := c.subnetGet(netclient1, &subnets.ListOpts{Name: newRds.Spec.Subnet})
	if err != nil {
		err := fmt.Errorf("error getting subnet state: %v", err)
		return err
	}

	v, err := c.vpcGet(netclient1, &vpcs.ListOpts{Name: newRds.Spec.Vpc})
	if err != nil {
		err := fmt.Errorf("error getting vpc state: %v", err)
		return err
	}

	createOpts := instances.CreateRdsOpts{}
	if newRds.Spec.Hamode == "Ha" {
		createOpts = instances.CreateRdsOpts{
			Name: newRds.Name,
			Datastore: &instances.Datastore{
				Type:    newRds.Spec.Datastoretype,
				Version: newRds.Spec.Datastoreversion,
			},
			Ha: &instances.Ha{
				Mode:            newRds.Spec.Hamode,
				ReplicationMode: newRds.Spec.Hareplicationmode,
			},
			Port:     newRds.Spec.Port,
			Password: newRds.Spec.Password,
			BackupStrategy: &instances.BackupStrategy{
				StartTime: newRds.Spec.Backupstarttime,
				KeepDays:  newRds.Spec.Backupkeepdays,
			},
			FlavorRef: newRds.Spec.Flavorref,
			Volume: &instances.Volume{
				Type: newRds.Spec.Volumetype,
				Size: newRds.Spec.Volumesize,
			},
			Region:           newRds.Spec.Region,
			AvailabilityZone: newRds.Spec.Availabilityzone,
			VpcId:            v.ID,
			SubnetId:         s.ID,
			SecurityGroupId:  g.ID,
		}
	} else {
		createOpts = instances.CreateRdsOpts{
			Name: newRds.Name,
			Datastore: &instances.Datastore{
				Type:    newRds.Spec.Datastoretype,
				Version: newRds.Spec.Datastoreversion,
			},
			Port:     newRds.Spec.Port,
			Password: newRds.Spec.Password,
			BackupStrategy: &instances.BackupStrategy{
				StartTime: newRds.Spec.Backupstarttime,
				KeepDays:  newRds.Spec.Backupkeepdays,
			},
			FlavorRef: newRds.Spec.Flavorref,
			Volume: &instances.Volume{
				Type: newRds.Spec.Volumetype,
				Size: newRds.Spec.Volumesize,
			},
			Region:           newRds.Spec.Region,
			AvailabilityZone: newRds.Spec.Availabilityzone,
			VpcId:            v.ID,
			SubnetId:         s.ID,
			SecurityGroupId:  g.ID,
		}
	}

	createResult := instances.Create(client, createOpts)
	r, err := createResult.Extract()
	if err != nil {
		err := fmt.Errorf("error creating rds instance: %v", err)
		return err
	}
	newRds.Status.Id = r.Instance.Id
	newRds.Status.Status = r.Instance.Status
	if err := c.UpdateStatus(ctx, newRds); err != nil {
		err := fmt.Errorf("error update rds create status: %v", err)
		return err
	}
	jobResponse, err := createResult.ExtractJobResponse()
	if err != nil {
		err := fmt.Errorf("error creating rds job: %v", err)
		return err
	}

	if err := instances.WaitForJobCompleted(client, int(1800), jobResponse.JobID); err != nil {
		err := fmt.Errorf("error getting rds job: %v", err)
		return err
	}

	rdsInstance, err := c.rdsGetById(client, r.Instance.Id)
	if err != nil {
		err := fmt.Errorf("error getting rds by id: %v", err)
		return err
	}
	newRds.Status.Id = rdsInstance.Id
	newRds.Status.Ip = rdsInstance.PrivateIps[0]
	newRds.Status.Status = rdsInstance.Status
	if err := c.UpdateStatus(ctx, newRds); err != nil {
		err := fmt.Errorf("error update rds status: %v", err)
		return err
	}

	return nil
}

func (c *Controller) rdsDelete(client *golangsdk.ServiceClient, newRds *rdsv1alpha1.Rds) error {
	c.logger.Debug("rdsDelete ", newRds.Name)
	c.recorder.Eventf(newRds, rdsv1alpha1.EventTypeNormal, "Create", "This instance is deleting.")
	if newRds.Status.Id != "" {
		deleteResult := instances.Delete(client, newRds.Status.Id)
		jobResponse, err := deleteResult.ExtractJobResponse()
		if err != nil {
			err := fmt.Errorf("error rds delete job: %v", err)
			return err
		}

		if err := instances.WaitForJobCompleted(client, int(1800), jobResponse.JobID); err != nil {
			err := fmt.Errorf("error getting rds delete job: %v", err)
			return err
		}
	} else {
		err := fmt.Errorf("no rds id to delete")
		return err
	}
	return nil
}

func (opts myRDSRestartOpts) ToRestartRdsInstanceMap() (map[string]interface{}, error) {
	b, err := golangsdk.BuildRequestBody(&opts, "")
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (c *Controller) rdsUpdate(ctx context.Context, client *golangsdk.ServiceClient, oldRds *rdsv1alpha1.Rds, newRds *rdsv1alpha1.Rds) error {
	c.logger.Debug("rdsUpdate ", newRds.Name)
	if newRds.Status.Id == "" {
		err := fmt.Errorf("rdsUpdate failed, Rds.Status.Id is empty")
		return err
	}
	// remove the initial root password from spec
	if newRds.Spec.Password != "" {
		newRds.Spec.Password = ""
		if err := c.UpdateStatus(ctx, newRds); err != nil {
			err := fmt.Errorf("error update rds status: %v", err)
			return err
		}
	}
	// Enlarge volume here
	if oldRds.Spec.Volumesize < newRds.Spec.Volumesize {
		c.logger.Debug("rdsUpdate: englarge volume")
		c.recorder.Eventf(newRds, rdsv1alpha1.EventTypeNormal, "Update", "This instance is enlarging.")
		enlargeOpts := instances.EnlargeVolumeRdsOpts{
			EnlargeVolume: &instances.EnlargeVolumeSize{
				Size: newRds.Spec.Volumesize,
			},
		}
		enlargeResult := instances.EnlargeVolume(client, enlargeOpts, newRds.Status.Id)
		_, err := enlargeResult.Extract()
		if err != nil {
			err := fmt.Errorf("error enlarge rds: %v", err)
			return err
		}
		jobResponse, err := enlargeResult.ExtractJobResponse()
		if err != nil {
			err := fmt.Errorf("error creating rds enlarge job: %v", err)
			return err
		}

		if err := instances.WaitForJobCompleted(client, int(1800), jobResponse.JobID); err != nil {
			err := fmt.Errorf("error getting rds enlarge job: %v", err)
			return err
		}

		rdsInstance, err := c.rdsGetById(client, newRds.Status.Id)
		if err != nil {
			err := fmt.Errorf("error getting rds by id: %v", err)
			return err
		}
		newRds.Status.Status = rdsInstance.Status
		if err := c.UpdateStatus(ctx, newRds); err != nil {
			err := fmt.Errorf("error update rds status: %v", err)
			return err
		}
	}
	// Change Flavor here
	if oldRds.Spec.Flavorref != newRds.Spec.Flavorref {
		c.logger.Debug("rdsUpdate: change flavor")
		c.recorder.Eventf(newRds, rdsv1alpha1.EventTypeNormal, "Update", "This instance is scaling.")
		resizeOpts := instances.ResizeFlavorOpts{
			ResizeFlavor: &instances.SpecCode{
				Speccode: newRds.Spec.Flavorref,
			},
		}
		resizeResult := instances.Resize(client, resizeOpts, newRds.Status.Id)
		_, err := resizeResult.Extract()
		if err != nil {
			err := fmt.Errorf("error resizing rds: %v", err)
			return err
		}
		jobResponse, err := resizeResult.ExtractJobResponse()
		if err != nil {
			err := fmt.Errorf("error creating rds resize job: %v", err)
			return err
		}

		if err := instances.WaitForJobCompleted(client, int(1800), jobResponse.JobID); err != nil {
			err := fmt.Errorf("error getting rds resize job: %v", err)
			return err
		}

		rdsInstance, err := c.rdsGetById(client, newRds.Status.Id)
		if err != nil {
			err := fmt.Errorf("error getting rds by id: %v", err)
			return err
		}
		newRds.Status.Status = rdsInstance.Status
		if err := c.UpdateStatus(ctx, newRds); err != nil {
			err := fmt.Errorf("error update rds status: %v", err)
			return err
		}
	}
	// Restart instance here
	if newRds.Status.Reboot == true {
		c.logger.Debug("rdsUpdate: restart instance")
		c.recorder.Eventf(newRds, rdsv1alpha1.EventTypeNormal, "Update", "This instance is rebooting.")
		newRds.Status.Reboot = false
		if err := c.UpdateStatus(ctx, newRds); err != nil {
			err := fmt.Errorf("error update rds status: %v", err)
			return err
		}

		restartResult, err := instances.Restart(client, myRDSRestartOpts{}, newRds.Status.Id).Extract()
		if err != nil {
			err := fmt.Errorf("error rebooting rds: %v", err)
			return err
		}
		if err := instances.WaitForJobCompleted(client, int(1800), restartResult.JobId); err != nil {
			err := fmt.Errorf("error getting rds restart job: %v", err)
			return err
		}

		rdsInstance, err := c.rdsGetById(client, newRds.Status.Id)
		if err != nil {
			err := fmt.Errorf("error getting rds by id: %v", err)
			return err
		}
		newRds.Status.Status = rdsInstance.Status
		if err := c.UpdateStatus(ctx, newRds); err != nil {
			err := fmt.Errorf("error update rds status: %v", err)
			return err
		}
	}
	// Restore backup PITR
	if newRds.Spec.Backuprestoretime != "" { // 2020-04-04T22:08:41+00:00
		c.logger.Debug("rdsUpdate: restore instance")
		c.recorder.Eventf(newRds, rdsv1alpha1.EventTypeNormal, "Update", "This instance is restoring.")
		rdsRestoredate, err := time.Parse(time.RFC3339, newRds.Spec.Backuprestoretime)
		if err != nil {
			err := fmt.Errorf("can't parse rds restore time: %v", err)
			return err
		}
		newRds.Spec.Backuprestoretime = ""
		if err := c.UpdateStatus(ctx, newRds); err != nil {
			err := fmt.Errorf("error update rds status: %v", err)
			return err
		}
		rdsRestoretime := rdsRestoredate.UnixMilli()
		restoreOpts := backups.RestorePITROpts{
			Source: backups.Source{
				InstanceID:  newRds.Status.Id,
				RestoreTime: rdsRestoretime,
				Type:        "timestamp",
			},
			Target: backups.Target{
				InstanceID: newRds.Status.Id,
			},
		}

		restoreResult := backups.RestorePITR(client, restoreOpts)
		// restoredRds, err := restoreResult.Extract()
		if err != nil {
			err := fmt.Errorf("rds restore failed: %v", err)
			return err
		}
		rdsInstance, err := c.rdsGetById(client, newRds.Status.Id)
		if err != nil {
			err := fmt.Errorf("error getting rds by id: %v", err)
			return err
		}
		newRds.Status.Status = rdsInstance.Status
		if err := c.UpdateStatus(ctx, newRds); err != nil {
			err := fmt.Errorf("error update rds status: %v", err)
			return err
		}

		jobResponse, err := restoreResult.ExtractJobResponse()
		if err != nil {
			err := fmt.Errorf("can't get rds restore job: %v", err)
			return err
		}

		if err := instances.WaitForJobCompleted(client, int(1800), jobResponse.JobID); err != nil {
			err := fmt.Errorf("error rds restore job: %v", err)
			return err
		}
		rdsInstance, err = c.rdsGetById(client, newRds.Status.Id)
		if err != nil {
			err := fmt.Errorf("error getting rds by id: %v", err)
			return err
		}
		newRds.Status.Status = rdsInstance.Status
		if err := c.UpdateStatus(ctx, newRds); err != nil {
			err := fmt.Errorf("error update rds status: %v", err)
			return err
		}
	}

	/*
		// Implementation of errorlog/slowquerylog
		// can be very long (+500 events)
		// structured data https://pkg.go.dev/github.com/opentelekomcloud/gophertelekomcloud@v0.5.9/openstack/rds/v3/instances#ErrorLogResp.ErrorLogList
		// needs the implementaion of event log handler
		// https://github.com/kubernetes/client-go/blob/master/tools/record/event.go
		// examples:
		// https://github.com/gaulzhw/learning_k8s/blob/3bab7e22958a30684bd464b0f174b3ac38d5b891/code/pkg/controllers/informer_controller.go
		// https://github.com/kaidotdev/events-logger/blob/master/main.go
		//  Error Logs https://github.com/opentelekomcloud/gophertelekomcloud/blob/devel/openstack/rds/v3/instances/requests.go#L302
		// Slow Logs https://github.com/opentelekomcloud/gophertelekomcloud/blob/devel/openstack/rds/v3/instances/requests.go#L375

		fmt.Println("doing errorlog catchup")
		sd := time.Now().AddDate(0, -1, 0)
		ed := time.Now()
		start_date := fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d+0000",
			sd.Year(), sd.Month(), sd.Day(),
			sd.Hour(), sd.Minute(), sd.Second())
		end_date := fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d+0000",
			ed.Year(), ed.Month(), ed.Day(),
			ed.Hour(), ed.Minute(), ed.Second())

		errorLogOpts := instances.DbErrorlogOpts{StartDate: start_date, EndDate: end_date}
		allPages, err := instances.ListErrorLog(client, errorLogOpts, newRds.Status.Id).AllPages()
		if err != nil {
			err := fmt.Errorf("error getting rds pages: %v", err)
			return err
		}
		errorLogs, err := instances.ExtractErrorLog(allPages)
		if err != nil {
			err := fmt.Errorf("error getting rds errorlog: %v", err)
			return err
		}

		copier.Copy(&newRds.Events.Errorlog, &errorLogs.ErrorLogList)
		fmt.Println(newRds.Events)
		if err := UpdateStatus(ctx, newRds); err != nil {
			err := fmt.Errorf("error update rds error log events: %v", err)
			return err
		}
	*/
	return nil
}

func (c *Controller) rdsUpdateStatus(ctx context.Context, client *golangsdk.ServiceClient, newRds *rdsv1alpha1.Rds) error {
	if newRds.Status.Id == "" {
		err := fmt.Errorf("rdsUpdateStatus failed, Rds.Status.Id is empty")
		return err
	}
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		err := fmt.Errorf("error init in-cluster config: %v", err)
		return err
	}
	rdsclientset, err := rdsv1alpha1clientset.NewForConfig(restConfig)
	if err != nil {
		err := fmt.Errorf("error creating rdsclientset: %v", err)
		return err
	}
	rdsInstance, err := c.rdsGetByName(client, newRds.Name)
	if len(rdsInstance.PrivateIps) > 0 {
		newRds.Status.Ip = rdsInstance.PrivateIps[0]
	} else {
		newRds.Status.Ip = ""
	}
	if rdsInstance.Status != "" {
		newRds.Status.Status = rdsInstance.Status
	}
	rdsstatus, err := rdsclientset.McspsV1alpha1().Rdss(newRds.Namespace).Update(ctx, newRds, metav1.UpdateOptions{})
	fmt.Println(rdsstatus)
	if err != nil {
		err := fmt.Errorf("error update rds: %v", err)
		return err
	}
	return nil
}

func getProvider() (*golangsdk.ProviderClient, error) {
	if os.Getenv("OS_AUTH_URL") == "" {
		os.Setenv("OS_AUTH_URL", "https://iam.eu-de.otc.t-systems.com:443/v3")
	}

	if os.Getenv("OS_IDENTITY_API_VERSION") == "" {
		os.Setenv("OS_IDENTITY_API_VERSION", "3")
	}

	if os.Getenv("OS_REGION_NAME") == "" {
		os.Setenv("OS_REGION_NAME", "eu-de")
	}

	if os.Getenv("OS_PROJECT_NAME") == "" {
		os.Setenv("OS_PROJECT_NAME", "eu-de")
	}

	opts, err := openstack.AuthOptionsFromEnv()
	if err != nil {
		return nil, fmt.Errorf("error getting auth from env: %v", err)
	}

	provider, err := openstack.AuthenticatedClient(opts)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize openstack client: %v", err)
	}

	if os.Getenv("OS_DEBUG") == "1" {
		provider.HTTPClient = http.Client{
			Transport: &client.RoundTripper{
				Rt:     &http.Transport{},
				Logger: &client.DefaultLogger{},
			},
		}
	}
	return provider, nil
}

func (c *Controller) Create(ctx context.Context, newRds *rdsv1alpha1.Rds) error {
	provider, err := getProvider()
	if err != nil {
		return fmt.Errorf("unable to initialize provider: %v", err)
	}
	network1, err := openstack.NewNetworkV1(provider, golangsdk.EndpointOpts{})
	if err != nil {
		return fmt.Errorf("unable to initialize network v1 client: %v", err)
	}
	network2, err := openstack.NewNetworkV2(provider, golangsdk.EndpointOpts{})
	if err != nil {
		return fmt.Errorf("unable to initialize network v2 client: %v", err)
	}
	rdsapi, err := openstack.NewRDSV3(provider, golangsdk.EndpointOpts{})
	if err != nil {
		return fmt.Errorf("unable to initialize rds client: %v", err)
	}

	c.rdsCreate(ctx, network1, network2, rdsapi, &instances.CreateRdsOpts{}, newRds)
	if err != nil {
		return fmt.Errorf("rds creating failed: %v", err)
	}
	return nil
}

func (c *Controller) Delete(newRds *rdsv1alpha1.Rds) error {
	provider, err := getProvider()
	if err != nil {
		return fmt.Errorf("unable to initialize provider: %v", err)
	}
	rdsapi, err := openstack.NewRDSV3(provider, golangsdk.EndpointOpts{})
	if err != nil {
		return fmt.Errorf("unable to initialize rds client: %v", err)
	}

	err = c.rdsDelete(rdsapi, newRds)
	if err != nil {
		return fmt.Errorf("rds delete failed: %v", err)
	}
	return nil
}

func (c *Controller) Update(ctx context.Context, oldRds *rdsv1alpha1.Rds, newRds *rdsv1alpha1.Rds) error {
	provider, err := getProvider()
	if err != nil {
		return fmt.Errorf("unable to initialize provider: %v", err)
	}
	rdsapi, err := openstack.NewRDSV3(provider, golangsdk.EndpointOpts{})
	if err != nil {
		return fmt.Errorf("unable to initialize rds client: %v", err)
	}

	c.rdsUpdate(ctx, rdsapi, oldRds, newRds)
	if err != nil {
		return fmt.Errorf("rds update failed: %v", err)
	}
	return nil
}

// Update K8s RDS Resource
func (c *Controller) UpdateStatus(ctx context.Context, newRds *rdsv1alpha1.Rds) error {
	c.logger.Debug("UpdateStatus ", newRds.Name)
	provider, err := getProvider()
	if err != nil {
		return fmt.Errorf("unable to initialize provider: %v", err)
	}
	rdsapi, err := openstack.NewRDSV3(provider, golangsdk.EndpointOpts{})
	if err != nil {
		return fmt.Errorf("unable to initialize rds client: %v", err)
	}

	c.logger.Debug("UpdateStatus Detail", newRds.Status)
	c.rdsUpdateStatus(ctx, rdsapi, newRds)
	if err != nil {
		return fmt.Errorf("rds update status failed: %v", err)
	}
	return nil
}

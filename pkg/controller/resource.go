package controller

// factory otc resources here

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"net/http"
	"os"
	"time"

	"github.com/gophercloud/utils/client"
	golangsdk "github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/identity/v3/tokens"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/networking/v1/subnets"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/networking/v1/vpcs"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/networking/v2/extensions/security/groups"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/rds/v3/backups"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/rds/v3/flavors"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/rds/v3/instances"

	batch "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"

	rdsv1alpha1 "github.com/eumel8/otc-rds-operator/pkg/rds/v1alpha1"
	rdsv1alpha1clientset "github.com/eumel8/otc-rds-operator/pkg/rds/v1alpha1/apis/clientset/versioned"
)

var (
	projectID = string("7c3ec0b3db5f476990043258670caf82")
)

// workaround https://github.com/opentelekomcloud/gophertelekomcloud/issues/342
type myRDSRestartOpts struct {
	Restart struct{} `json:"restart"`
}

type posflavor []struct {
	VCPUs int
	RAM   int
	Spec  string
}

func (c *Controller) secgroupGet(client *golangsdk.ServiceClient, opts *groups.ListOpts) (*groups.SecGroup, error) {
	c.logger.Debug("secgroupGet")
	pages, err := groups.List(client, *opts).AllPages()
	if err != nil {
		return nil, err
	}
	n, _ := groups.ExtractGroups(pages)
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
	c.logger.Debug("rdsGetById ", rdsId)
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

func (c *Controller) rdsGetByName(client *golangsdk.ServiceClient, rdsName string, namespace string) (*instances.RdsInstanceResponse, error) {
	c.logger.Debug("rdsGetByName ", rdsName)
	listOpts := instances.ListRdsInstanceOpts{
		Name: namespace + "_" + rdsName,
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
	rdsCheck, err := c.rdsGetByName(client, newRds.Name, newRds.Namespace)
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
			Name: newRds.Namespace + "_" + newRds.Name,
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
			Name: newRds.Namespace + "_" + newRds.Name,
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

	c.recorder.Eventf(newRds, rdsv1alpha1.EventTypeNormal, "Create", "This instance is creating.")
	createResult := instances.Create(client, createOpts)
	r, err := createResult.Extract()
	if err != nil {
		err := fmt.Errorf("error creating rds instance: %v", err)
		return err
	}
	newRds.Spec.Password = "xxxxxxxxxx"
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
	if newRds.Status.Id != "" {
		c.recorder.Eventf(newRds, rdsv1alpha1.EventTypeNormal, "Create", "This instance is deleting.")

		// make a backup before instance deleting
		backuptime := strconv.FormatInt(time.Now().Unix(), 10)
		backupOpts := backups.CreateOpts{
			InstanceID:  newRds.Status.Id,
			Name:        newRds.Namespace + "_" + newRds.Name + "_" + backuptime,
			Description: "RDS Operator Last Backup",
		}
		backupResponse, err := backups.Create(client, backupOpts).Extract()
		if err != nil {
			err := fmt.Errorf("error creating rds backup before instance deleting: %v", err)
			return err
		}
		err = backups.WaitForBackup(client, backupOpts.InstanceID, backupResponse.ID, backups.StatusCompleted)
		if err != nil {
			err := fmt.Errorf("error wait for rds backup before instance deleting: %v", err)
			return err
		}

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
	// re-check real rds spec on otc
	rdsInstance, err := c.rdsGetByName(client, newRds.Name, newRds.Namespace)
	if err != nil {
		err := fmt.Errorf("error getting rdsGetByName: %v", err)
		return err
	}
	oldRds.Spec.Volumesize = rdsInstance.Volume.Size
	oldRds.Spec.Flavorref = rdsInstance.FlavorRef
	// Enlarge volume here
	if oldRds.Spec.Volumesize < newRds.Spec.Volumesize {
		c.logger.Debug("rdsUpdate: enlarge volume")
		eventMsg := fmt.Sprint("This instance is enlarging to ", newRds.Spec.Volumesize)
		c.recorder.Eventf(newRds, rdsv1alpha1.EventTypeNormal, "Update", string(eventMsg))
		enlargeOpts := instances.EnlargeVolumeRdsOpts{
			EnlargeVolume: &instances.EnlargeVolumeSize{
				Size: newRds.Spec.Volumesize,
			},
		}
		enlargeResult := instances.EnlargeVolume(client, enlargeOpts, newRds.Status.Id)
		_, err := enlargeResult.Extract()
		if enlargeResult.Err != nil {
			err := fmt.Errorf("error rds api for enlarge: %v", enlargeResult.Err)
			return err
		}
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
		eventMsg := fmt.Sprint("This instance is scaling to ", newRds.Spec.Flavorref)
		c.recorder.Eventf(newRds, rdsv1alpha1.EventTypeNormal, "Update", string(eventMsg))
		resizeOpts := instances.ResizeFlavorOpts{
			ResizeFlavor: &instances.SpecCode{
				Speccode: newRds.Spec.Flavorref,
			},
		}
		resizeResult := instances.Resize(client, resizeOpts, newRds.Status.Id)
		if resizeResult.Err != nil {
			err := fmt.Errorf("error rds api for resize: %v", resizeResult.Err)
			return err
		}
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
	if newRds.Status.Reboot {
		c.logger.Debug("rdsUpdate: restart instance")
		newRds.Status.Reboot = false
		if err := c.UpdateStatus(ctx, newRds); err != nil {
			err := fmt.Errorf("error update rds status: %v", err)
			return err
		}

		c.recorder.Eventf(newRds, rdsv1alpha1.EventTypeNormal, "Update", "This instance is rebooting.")
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
		eventMsg := fmt.Sprint("This instance is restoring to ", newRds.Spec.Backuprestoretime)
		c.recorder.Eventf(newRds, rdsv1alpha1.EventTypeNormal, "Update", string(eventMsg))
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
		if restoreResult.Err != nil {
			err := fmt.Errorf("rds restore failed: %v", restoreResult.Err)
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

	if newRds.Status.Logs {
		c.logger.Debug("rdsUpdate: instance errorlogs")
		newRds.Status.Logs = false
		if err := c.UpdateStatus(ctx, newRds); err != nil {
			err := fmt.Errorf("error update rds status: %v", err)
			return err
		}
		c.recorder.Eventf(newRds, rdsv1alpha1.EventTypeNormal, "Update", "This instance fetch logs.")
		opts, err := openstack.AuthOptionsFromEnv()
		if err != nil {
			err := fmt.Errorf("error getting auth from env in logfetch: %v", err)
			return err
		}
		provider, err := openstack.AuthenticatedClient(opts)
		if err != nil {
			err := fmt.Errorf("error building auth client: %v", err)
			return err
		}
		client, _ := openstack.NewIdentityV3(provider, golangsdk.EndpointOpts{})

		if os.Getenv("OS_PROJEKT_ID") != "" {
			projectID = os.Getenv("OS_PROJECT_ID")
		}

		authOptions := tokens.AuthOptions{
			IdentityEndpoint: opts.IdentityEndpoint,
			Username:         opts.Username,
			Password:         opts.Password,
			Scope:            tokens.Scope{ProjectID: projectID},
			DomainName:       opts.DomainName,
		}

		token, err := tokens.Create(client, &authOptions).ExtractToken()
		if err != nil {
			err := fmt.Errorf("error getting token in logfetch: %v", err)
			return err
		}

		job := createJob(newRds, opts.IdentityEndpoint, token.ID)

		_, err = c.kubeClientSet.BatchV1().
			Jobs(newRds.Namespace).
			Create(ctx, job, metav1.CreateOptions{})
		if err != nil {
			err := fmt.Errorf("error creating logfetch job: %v", err)
			return err
		}

		watch, err := c.kubeClientSet.BatchV1().Jobs(newRds.Namespace).Watch(ctx, metav1.ListOptions{LabelSelector: "job-name=" + newRds.Name})
		if err != nil {
			err := fmt.Errorf("error create watcher for logfetch job: %v", err)
			return err
		}

		logInstance := newRds.Namespace + "_" + newRds.Name
		logjob, err := c.kubeClientSet.BatchV1().Jobs(newRds.Namespace).Get(ctx, logInstance, metav1.GetOptions{})
		if err != nil {
			err := fmt.Errorf("error getting logfetch job for watch: %v", err)
			return err
		}

		if logjob == nil {
			err := fmt.Errorf("error finding logfetch job for watch: %v", err)
			return err
		}

		events := watch.ResultChan()
		for {
			select {
			case event := <-events:
				if event.Object == nil {
					_ = tokens.Revoke(client, token.ID)
					err := fmt.Errorf("error on result channel logfetch job: %v", err)
					return err
				}
				k8sJob, ok := event.Object.(*batch.Job)
				if !ok {
					_ = tokens.Revoke(client, token.ID)
					err := fmt.Errorf("error on object logfetch job: %v", err)
					return err
				}
				conditions := k8sJob.Status.Conditions
				for _, condition := range conditions {
					if condition.Type == batch.JobComplete {
						_ = tokens.Revoke(client, token.ID)
						return nil
					} else if condition.Type == batch.JobFailed {
						_ = tokens.Revoke(client, token.ID)
						err := fmt.Errorf("logfetch job for %s failed", newRds.Name)
						return err
					}
				}
			case <-ctx.Done():
				// revoke OTC Auth Token for job
				_ = tokens.Revoke(client, token.ID)
				err := fmt.Errorf("logfetch job %s cancelled", newRds.Name)
				return err
			}
		}
	}
	// sql user handling
	if len(oldRds.Spec.Databases) != len(newRds.Spec.Databases) || oldRds.Spec.Users != newRds.Spec.Users {
		c.logger.Debug("rdsUpdate: change sql user")

		err := c.CreateSqlUser(newRds)
		if err != nil {
			err := fmt.Errorf("error CreateSqlUser: %v", err)
			return err
		}
	}

	if newRds.Spec.Endpoint != "" && !newRds.Status.Autopilot {
		c.logger.Debug("rdsUpdate: autopilot")
		c.recorder.Eventf(newRds, rdsv1alpha1.EventTypeNormal, "Update", "This instance set autopilot.")
		newRds.Status.Autopilot = true
		if err := c.UpdateStatus(ctx, newRds); err != nil {
			err := fmt.Errorf("error update rds status for autopilot: %v", err)
			return err
		}
		rdsInstance, err := c.rdsGetById(client, newRds.Status.Id)
		if err != nil {
			err := fmt.Errorf("error getting rds by id: %v", err)
			return err
		}
		err = c.CreateAlarm(rdsInstance.Nodes[0].Id, newRds.Spec.Endpoint, newRds.Name, newRds.Namespace)
		if err != nil {
			err := fmt.Errorf("error creating alarm: %v", err)
			return err
		}
		return nil
	}
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
	rdsInstance, err := c.rdsGetByName(client, newRds.Name, newRds.Namespace)
	if err != nil {
		err := fmt.Errorf("error getting rdsGetByName: %v", err)
		return err
	}
	if len(rdsInstance.PrivateIps) > 0 {
		newRds.Status.Ip = rdsInstance.PrivateIps[0]
	} else {
		newRds.Status.Ip = "0.0.0.0"
	}
	if rdsInstance.Status != "" {
		newRds.Status.Status = rdsInstance.Status
	}
	returnRds, err := rdsclientset.McspsV1alpha1().Rdss(newRds.Namespace).Update(ctx, newRds, metav1.UpdateOptions{})
	if returnRds.Status != newRds.Status {
		err := fmt.Errorf("error update rds, result empty")
		return err
	}
	if err != nil {
		err := fmt.Errorf("error update rds: %v", err)
		return err
	}
	// create service
	k8sclientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		err := fmt.Errorf("error creating k8sclientset: %v", err)
		return err
	}
	rdsService, err := k8sclientset.CoreV1().Services(newRds.Namespace).Get(context.TODO(), newRds.Name, metav1.GetOptions{})
	/*
		rdsService, err := k8sclientset.CoreV1().Services(c.namespace).Get(&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      newRds.Name,
				Namespace: newRds.Namespace,
			},
		})
	*/
	fmt.Println("GET SERVICE ", rdsService.ObjectMeta.Name)
	if err != nil {
		err := fmt.Errorf("error getting service: %v", err)
		return err
	}

	if len(rdsService.ObjectMeta.Name) == 0 {
		rdsCreatedService, err := k8sclientset.CoreV1().Services(newRds.Namespace).Create(context.TODO(), &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      newRds.Name,
				Namespace: newRds.Namespace,
				Labels: map[string]string{
					"k8s-app": "otc-rds-operatpr",
				},
			},
			Spec: corev1.ServiceSpec{
				Type:         corev1.ServiceTypeExternalName,
				ExternalName: newRds.Status.Ip,
			},
		}, metav1.CreateOptions{})
		if err != nil {
			err := fmt.Errorf("error creating service: %v", err)
			return err
		}
		fmt.Println("CREATE SERVICE ", rdsCreatedService)
	}

	return nil
}

func (c *Controller) RdsFlavorLookup(newRds *rdsv1alpha1.Rds, raisetype string) (string, error) {
	provider, err := GetProvider()
	if err != nil {
		return "", err
	}
	client, err := openstack.NewRDSV3(provider, golangsdk.EndpointOpts{})
	if err != nil {
		return "", err
	}

	azs := strings.Split(newRds.Spec.Availabilityzone, ",")
	var (
		az1 string
		az2 string
	)
	az1 = string(azs[0])
	if len(azs) > 1 {
		az2 = string(azs[1])
	} else {
		az2 = ""
	}
	curCpu := "0"
	curMem := 0
	posflavor := posflavor{}

	listOpts := flavors.ListOpts{
		VersionName: newRds.Spec.Datastoreversion,
	}
	allFlavorPages, err := flavors.List(client, listOpts, newRds.Spec.Datastoretype).AllPages()
	if err != nil {
		klog.Exitf("unable to list flavor: %v", err)
		return "", err
	}

	rdsFlavors, err := flavors.ExtractDbFlavors(allFlavorPages)
	if err != nil {
		klog.Exitf("unable to extract flavor: %v", err)
		return "", err
	}

	for _, rds := range rdsFlavors {
		if rds.SpecCode == newRds.Spec.Flavorref {
			curCpu = rds.VCPUs
			curMem = rds.RAM
		}
	}
	switch raisetype {
	case "cpu":
		for _, rds := range rdsFlavors {
			for n, az := range rds.AzStatus {
				if n == az1 && az == "normal" {
					for l, az := range rds.AzStatus {
						if az2 == "" || l == az2 && az == "normal" {
							if strings.HasSuffix(newRds.Spec.Flavorref, ".ha") && strings.HasSuffix(rds.SpecCode, ".ha") && rds.VCPUs > curCpu {
								iCpu, _ := strconv.Atoi(rds.VCPUs)
								posflavor = append(posflavor, struct {
									VCPUs int
									RAM   int
									Spec  string
								}{iCpu, rds.RAM, rds.SpecCode})
							}
							if !strings.HasSuffix(newRds.Spec.Flavorref, ".ha") && !strings.HasSuffix(rds.SpecCode, ".rr") && !strings.HasSuffix(rds.SpecCode, ".ha") {
								iCpu, _ := strconv.Atoi(rds.VCPUs)
								posflavor = append(posflavor, struct {
									VCPUs int
									RAM   int
									Spec  string
								}{iCpu, rds.RAM, rds.SpecCode})
							}
						}
					}
				}
			}
		}
		sort.Slice(posflavor, func(i, j int) bool {
			return posflavor[i].VCPUs < posflavor[j].VCPUs
		})
		if len(posflavor) > 0 {
			return posflavor[0].Spec, nil
		}

	case "mem":
		for _, rds := range rdsFlavors {
			for n, az := range rds.AzStatus {
				if n == az1 && az == "normal" {
					for l, az := range rds.AzStatus {
						if az2 == "" || l == az2 && az == "normal" {
							if strings.HasSuffix(newRds.Spec.Flavorref, ".ha") && strings.HasSuffix(rds.SpecCode, ".ha") && rds.RAM > curMem {
								iCpu, _ := strconv.Atoi(rds.VCPUs)
								posflavor = append(posflavor, struct {
									VCPUs int
									RAM   int
									Spec  string
								}{iCpu, rds.RAM, rds.SpecCode})
							}
							if !strings.HasSuffix(newRds.Spec.Flavorref, ".ha") && !strings.HasSuffix(rds.SpecCode, ".rr") && !strings.HasSuffix(rds.SpecCode, ".ha") && rds.RAM > curMem {
								iCpu, _ := strconv.Atoi(rds.VCPUs)
								posflavor = append(posflavor, struct {
									VCPUs int
									RAM   int
									Spec  string
								}{iCpu, rds.RAM, rds.SpecCode})
							}
						}

					}

				}

			}
		}
		sort.SliceStable(posflavor, func(i, j int) bool {
			return posflavor[i].RAM < posflavor[j].RAM
		})
		if len(posflavor) > 0 {
			return posflavor[0].Spec, nil
		}
	}
	return "", fmt.Errorf("no flavor found")
}

func GetProvider() (*golangsdk.ProviderClient, error) {
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
	provider, err := GetProvider()
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
	provider, err := GetProvider()
	if err != nil {
		return fmt.Errorf("unable to initialize provider: %v", err)
	}
	rdsapi, err := openstack.NewRDSV3(provider, golangsdk.EndpointOpts{})
	if err != nil {
		return fmt.Errorf("unable to initialize rds client: %v", err)
	}

	if newRds.Status.Autopilot {
		err = c.DeleteAlarm(newRds.Name, newRds.Namespace)
		if err != nil {
			err := fmt.Errorf("error deleting alarm: %v", err)
			return err
		}
	}

	err = c.rdsDelete(rdsapi, newRds)
	if err != nil {
		return fmt.Errorf("rds delete failed: %v", err)
	}
	return nil
}

func (c *Controller) Update(ctx context.Context, oldRds *rdsv1alpha1.Rds, newRds *rdsv1alpha1.Rds) error {
	provider, err := GetProvider()
	if err != nil {
		return fmt.Errorf("unable to initialize provider: %v", err)
	}
	rdsapi, err := openstack.NewRDSV3(provider, golangsdk.EndpointOpts{})
	if err != nil {
		return fmt.Errorf("unable to initialize rds client: %v", err)
	}
	err = c.rdsUpdate(ctx, rdsapi, oldRds, newRds)
	if err != nil {
		return fmt.Errorf("rds update failed: %v", err)
	}
	return nil
}

// Update K8s RDS Resource
func (c *Controller) UpdateStatus(ctx context.Context, newRds *rdsv1alpha1.Rds) error {
	c.logger.Debug("UpdateStatus ", newRds.Name)
	provider, err := GetProvider()
	if err != nil {
		return fmt.Errorf("unable to initialize provider: %v", err)
	}
	rdsapi, err := openstack.NewRDSV3(provider, golangsdk.EndpointOpts{})
	if err != nil {
		return fmt.Errorf("unable to initialize rds client: %v", err)
	}

	c.rdsUpdateStatus(ctx, rdsapi, newRds)
	if err != nil {
		return fmt.Errorf("rds update status failed: %v", err)
	}
	return nil
}

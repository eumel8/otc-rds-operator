package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/gophercloud/utils/client"
	golangsdk "github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/networking/v1/subnets"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/networking/v1/vpcs"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/networking/v2/extensions/security/groups"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/rds/v3/instances"
	"k8s.io/client-go/rest"

	rdsv1alpha1 "github.com/eumel8/otc-rds-operator/pkg/rds/v1alpha1"
	rdsv1alpha1clientset "github.com/eumel8/otc-rds-operator/pkg/rds/v1alpha1/apis/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func secgroupGet(client *golangsdk.ServiceClient, opts *groups.ListOpts) (*groups.SecGroup, error) {
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

func subnetGet(client *golangsdk.ServiceClient, opts *subnets.ListOpts) (*subnets.Subnet, error) {
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

func vpcGet(client *golangsdk.ServiceClient, opts *vpcs.ListOpts) (*vpcs.Vpc, error) {
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

func rdsGetById(client *golangsdk.ServiceClient, rdsId string) (*instances.RdsInstanceResponse, error) {
	fmt.Printf("rdsGetById lookup %s\n", rdsId)
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

func rdsGetByName(client *golangsdk.ServiceClient, rdsName string) (*instances.RdsInstanceResponse, error) {
	fmt.Printf("rdsGetByName lookup %s\n", rdsName)
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

func rdsCreate(ctx context.Context, netclient1 *golangsdk.ServiceClient, netclient2 *golangsdk.ServiceClient, client *golangsdk.ServiceClient, opts *instances.CreateRdsOpts, newRds *rdsv1alpha1.Rds, namespace string) error {

	rdsCheck, err := rdsGetByName(client, newRds.Name)
	if rdsCheck != nil {
		err := fmt.Errorf("rds already exists %s", newRds.Name)
		return err
	}
	if err != nil {
		err := fmt.Errorf("error checking rds status %v", err)
		return err
	}

	g, err := secgroupGet(netclient2, &groups.ListOpts{Name: newRds.Spec.Securitygroup})
	if err != nil {
		err := fmt.Errorf("error getting secgroup state: %v", err)
		return err
	}

	s, err := subnetGet(netclient1, &subnets.ListOpts{Name: newRds.Spec.Subnet})
	if err != nil {
		err := fmt.Errorf("error getting subnet state: %v", err)
		return err
	}

	v, err := vpcGet(netclient1, &vpcs.ListOpts{Name: newRds.Spec.Vpc})
	if err != nil {
		err := fmt.Errorf("error getting vpc state: %v", err)
		return err
	}

	createOpts := instances.CreateRdsOpts{
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

	createResult := instances.Create(client, createOpts)
	r, err := createResult.Extract()
	if err != nil {
		err := fmt.Errorf("error creating rds instance: %v", err)
		return err
	}
	fmt.Println("RDS r")
	fmt.Println(r)
	fmt.Println(r.Instance.Id)
	newRds.Status.Id = r.Instance.Id
	newRds.Status.Status = r.Instance.Status
	newObj := newRds.DeepCopy()
	fmt.Println(newRds)
	fmt.Println(newObj)
	fmt.Println("=======")
	if err := UpdateStatus(ctx, newObj, namespace); err != nil {
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

	rdsInstance, err := rdsGetById(client, r.Instance.Id)
	newRds.Status.Id = rdsInstance.Id
	newRds.Status.Ip = rdsInstance.PrivateIps[0]
	newRds.Status.Status = rdsInstance.Status
	newObj = newRds.DeepCopy()
	if err := UpdateStatus(ctx, newObj, namespace); err != nil {
		err := fmt.Errorf("error update rds status: %v", err)
		return err
	}

	return nil
}

func rdsDelete(client *golangsdk.ServiceClient, newRds *rdsv1alpha1.Rds) error {
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

func rdsUpdate(client *golangsdk.ServiceClient, oldRds *rdsv1alpha1.Rds, newRds *rdsv1alpha1.Rds) error {
	fmt.Println("enter resource update")
	/* What we have todo here:
	* Resize Flavor https://github.com/opentelekomcloud/gophertelekomcloud/blob/devel/openstack/rds/v3/instances/requests.go#L269
	* Resize Storage https://github.com/opentelekomcloud/gophertelekomcloud/blob/devel/openstack/rds/v3/instances/requests.go#L302
	* Error Logs https://github.com/opentelekomcloud/gophertelekomcloud/blob/devel/openstack/rds/v3/instances/requests.go#L302
	* Slow Logs https://github.com/opentelekomcloud/gophertelekomcloud/blob/devel/openstack/rds/v3/instances/requests.go#L375
	* Restart https://github.com/opentelekomcloud/gophertelekomcloud/blob/devel/openstack/rds/v3/instances/requests.go#L160
	* Backup PITR Restore https://github.com/opentelekomcloud/gophertelekomcloud/blob/devel/openstack/rds/v3/backups/requests.go#L217
	 */
	/*
		createOpts := instances.CreateRdsOpts{
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

		createResult := instances.Create(client, createOpts)
		r, err := createResult.Extract()
		if err != nil {
			klog.Exitf("error creating rds instance: %v", err)
		}
		jobResponse, err := createResult.ExtractJobResponse()
		if err != nil {
			klog.Exitf("error creating rds job: %v", err)
		}

		if err := instances.WaitForJobCompleted(client, int(1800), jobResponse.JobID); err != nil {
			klog.Exitf("error getting rds job: %v", err)
		}

		rdsInstance, err := rdsGet(client, r.Instance.Id)

		fmt.Println(rdsInstance.PrivateIps[0])
		if err != nil {
			klog.Exitf("error getting rds state: %v", err)
		}
	*/
	return nil
}

func rdsUpdateStatus(ctx context.Context, client *golangsdk.ServiceClient, newRds *rdsv1alpha1.Rds, namespace string) error {
	fmt.Println("enter rdsUpdateStatus:")
	fmt.Println(newRds)
	if newRds.Status.Id != "" {
		fmt.Printf("Enter rdsUpdateStatus %s\n", newRds.Status.Id)
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
		rdsInstance, err := rdsGetByName(client, newRds.Name)
		//newRds.Status.Ip = rdsInstance.PrivateIps[0]
		fmt.Println("Enter rdsInstance")
		fmt.Println(rdsInstance)
		//newRds.Status.Status = rdsInstance.Status

		newObj := newRds.DeepCopy()
		fmt.Println("Enter newObj")
		fmt.Println(newObj)
		fmt.Println("=====================")
		_, err = rdsclientset.McspsV1alpha1().Rdss(namespace).Update(ctx, newObj, metav1.UpdateOptions{})
		if err != nil {
			err := fmt.Errorf("error update rds: %v", err)
			return err
		}
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

	if os.Getenv("OS_DEBUG") != "" {
		provider.HTTPClient = http.Client{
			Transport: &client.RoundTripper{
				Rt:     &http.Transport{},
				Logger: &client.DefaultLogger{},
			},
		}
	}
	return provider, nil
}

func Create(ctx context.Context, newRds *rdsv1alpha1.Rds, namespace string) error {
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

	rdsCreate(ctx, network1, network2, rdsapi, &instances.CreateRdsOpts{}, newRds, namespace)
	if err != nil {
		return fmt.Errorf("rds creating failed: %v", err)
	}
	return nil
}

func Delete(newRds *rdsv1alpha1.Rds) error {
	provider, err := getProvider()
	if err != nil {
		return fmt.Errorf("unable to initialize provider: %v", err)
	}
	rdsapi, err := openstack.NewRDSV3(provider, golangsdk.EndpointOpts{})
	if err != nil {
		return fmt.Errorf("unable to initialize rds client: %v", err)
	}

	err = rdsDelete(rdsapi, newRds)
	if err != nil {
		return fmt.Errorf("rds delete failed: %v", err)
	}
	return nil
}

func Update(oldRds *rdsv1alpha1.Rds, newRds *rdsv1alpha1.Rds) error {
	provider, err := getProvider()
	if err != nil {
		return fmt.Errorf("unable to initialize provider: %v", err)
	}
	rdsapi, err := openstack.NewRDSV3(provider, golangsdk.EndpointOpts{})
	if err != nil {
		return fmt.Errorf("unable to initialize rds client: %v", err)
	}

	rdsUpdate(rdsapi, oldRds, newRds)
	if err != nil {
		return fmt.Errorf("rds update failed: %v", err)
	}
	return nil
}

// Update K8s RDS Resource
func UpdateStatus(ctx context.Context, newRds *rdsv1alpha1.Rds, namespace string) error {
	provider, err := getProvider()
	if err != nil {
		return fmt.Errorf("unable to initialize provider: %v", err)
	}
	rdsapi, err := openstack.NewRDSV3(provider, golangsdk.EndpointOpts{})
	if err != nil {
		return fmt.Errorf("unable to initialize rds client: %v", err)
	}

	rdsUpdateStatus(ctx, rdsapi, newRds, namespace)
	if err != nil {
		return fmt.Errorf("rds update failed: %v", err)
	}
	return nil
}

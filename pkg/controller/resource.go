package controller

import (
	"context"
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
	"k8s.io/klog/v2"

	rds "github.com/eumel8/otc-rds-operator/pkg/rds"
	rdsv1alpha1 "github.com/eumel8/otc-rds-operator/pkg/rds/v1alpha1"
	rdsv1alpha1clientset "github.com/eumel8/otc-rds-operator/pkg/rds/v1alpha1/apis/clientset/versioned"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	image      = string("ghcr.io/eumel8/busybox:latest")
	user       = int64(1000)
	privledged = bool(false)
	readonly   = bool(true)
)

func createJob(newRds *rdsv1alpha1.Rds, namespace string) *batchv1.Job {
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      newRds.ObjectMeta.Name,
			Namespace: namespace,
			Labels:    make(map[string]string),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(
					newRds,
					rdsv1alpha1.SchemeGroupVersion.WithKind(rds.RdsKind),
				),
			},
		},
		Spec: createJobSpec(newRds.Name, namespace, newRds.Spec.Flavorref),
	}
}

func createJobSpec(name, namespace, msg string) batchv1.JobSpec {
	return batchv1.JobSpec{
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: name + "-",
				Namespace:    namespace,
				Labels:       make(map[string]string),
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:            name,
						Image:           "ghcr.io/eumel8/echobusybox:latest",
						Command:         []string{"echo", msg},
						ImagePullPolicy: "Always",
						SecurityContext: &corev1.SecurityContext{
							AllowPrivilegeEscalation: &privledged,
							Privileged:               &privledged,
							ReadOnlyRootFilesystem:   &readonly,
							RunAsGroup:               &user,
							RunAsUser:                &user,
						},
					},
				},
				RestartPolicy: corev1.RestartPolicyNever,
			},
		},
	}
}

// rds resource part here
func secgroupGet(client *golangsdk.ServiceClient, opts *groups.ListOpts) (*groups.SecGroup, error) {
	pages, err := groups.List(client, *opts).AllPages()
	if err != nil {
		return nil, err
	}
	n, err := groups.ExtractGroups(pages)
	if len(n) == 0 {
		klog.Exitf("no secgroup found")
	}

	return &n[0], nil
}

func subnetGet(client *golangsdk.ServiceClient, opts *subnets.ListOpts) (*subnets.Subnet, error) {
	n, err := subnets.List(client, *opts)
	if err != nil {
		return nil, err
	}
	if len(n) == 0 {
		klog.Exitf("no subnet found")
	}

	return &n[0], nil
}

func vpcGet(client *golangsdk.ServiceClient, opts *vpcs.ListOpts) (*vpcs.Vpc, error) {
	n, err := vpcs.List(client, *opts)
	if err != nil {
		return nil, err
	}

	if len(n) == 0 {
		klog.Exitf("no vpc found")
	}

	return &n[0], nil
}

func rdsGet(client *golangsdk.ServiceClient, rdsId string) (*instances.RdsInstanceResponse, error) {
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

func rdsCreate(ctx context.Context, netclient1 *golangsdk.ServiceClient, netclient2 *golangsdk.ServiceClient, client *golangsdk.ServiceClient, opts *instances.CreateRdsOpts, newRds *rdsv1alpha1.Rds, namespace string) error {

	if newRds.Status.Id != "" {
		klog.Exitf("rds already exists %v", newRds.Status.Id)
	}

	g, err := secgroupGet(netclient2, &groups.ListOpts{Name: newRds.Spec.Securitygroup})
	if err != nil {
		klog.Exitf("error getting secgroup state: %v", err)
	}

	s, err := subnetGet(netclient1, &subnets.ListOpts{Name: newRds.Spec.Subnet})
	if err != nil {
		klog.Exitf("error getting subnet state: %v", err)
	}

	v, err := vpcGet(netclient1, &vpcs.ListOpts{Name: newRds.Spec.Vpc})
	if err != nil {
		klog.Exitf("error getting vpc state: %v", err)
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
	newRds.Status.Id = r.Instance.Id
	newRds.Status.Ip = rdsInstance.PrivateIps[0]
	newRds.Status.Status = r.Instance.Status

	restConfig, err := rest.InClusterConfig()
	if err != nil {
		klog.Exitf("error init incluster config")
	}
	rdsclientset, err := rdsv1alpha1clientset.NewForConfig(restConfig)
	if err != nil {
		klog.Exitf("error creating rdsclientset")
	}
	newObj := newRds.DeepCopy()
	// listRds, err := rdsclientset.McspsV1alpha1().Rdss("rdsoperator").List(ctx, metav1.ListOptions{})
	// updateRds, err := rdsclientset.McspsV1alpha1().Rdss("rdsoperator").Update(ctx, newObj, metav1.UpdateOptions{})
	_, err = rdsclientset.McspsV1alpha1().Rdss(namespace).Update(ctx, newObj, metav1.UpdateOptions{})
	if err != nil {
		klog.Exitf("error update rds: %v", err)
	}
	return nil
}

func rdsDelete(client *golangsdk.ServiceClient, newRds *rdsv1alpha1.Rds) error {
	fmt.Println("enter resource delete")
	/*
		createResult := instances.Create(client, createOpts)
		r, err := createResult.Extract()
		if err != nil {
			klog.Exitf("error creating rds instance: %v", err)
		}
		rdsInstance, err := rdsGet(client, r.Instance.Id)

		fmt.Println(rdsInstance.PrivateIps[0])
		if err != nil {
			klog.Exitf("error getting rds state: %v", err)
		}
	*/
	return nil
}

func rdsUpdate(client *golangsdk.ServiceClient, opts *instances.CreateRdsOpts, newRds *rdsv1alpha1.Rds) error {
	fmt.Println("enter resource update")
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

	rdsDelete(rdsapi, newRds)
	if err != nil {
		return fmt.Errorf("rds delete failed: %v", err)
	}
	return nil
}

func Update(newRds *rdsv1alpha1.Rds) error {
	provider, err := getProvider()
	if err != nil {
		return fmt.Errorf("unable to initialize provider: %v", err)
	}
	rdsapi, err := openstack.NewRDSV3(provider, golangsdk.EndpointOpts{})
	if err != nil {
		return fmt.Errorf("unable to initialize rds client: %v", err)
	}

	rdsUpdate(rdsapi, &instances.CreateRdsOpts{}, newRds)
	if err != nil {
		return fmt.Errorf("rds update failed: %v", err)
	}
	return nil
}

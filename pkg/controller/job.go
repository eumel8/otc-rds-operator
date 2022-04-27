package controller

import (
	// "github.com/opentelekomcloud/gophertelekomcloud/openstack"
	"github.com/davecgh/go-spew/spew"
	golangsdk "github.com/opentelekomcloud/gophertelekomcloud"

	rds "github.com/eumel8/otc-rds-operator/pkg/rds"
	rdsv1alpha1 "github.com/eumel8/otc-rds-operator/pkg/rds/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	image      = string("ghcr.io/eumel8/otcrdslogs:latest")
	user       = int64(1000)
	privledged = bool(false)
	readonly   = bool(true)
)

// func createJob(newRds *rdsv1alpha1.Rds, opts *openstack.AuthOptionsFromEnv) *batchv1.Job {
func createJob(newRds *rdsv1alpha1.Rds, opts golangsdk.AuthOptions) *batchv1.Job {
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      newRds.ObjectMeta.Name,
			Namespace: newRds.Namespace,
			Labels:    make(map[string]string),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(
					newRds,
					rdsv1alpha1.SchemeGroupVersion.WithKind(rds.RdsKind),
				),
			},
		},
		Spec: createJobSpec(newRds.Name, newRds.Namespace, opts),
	}
}

func createJobSpec(name string, namespace string, opts golangsdk.AuthOptions) batchv1.JobSpec {
	spew.Dump(opts)
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
						Name:            "errorlog",
						Image:           image,
						Command:         []string{"/app/rdslogs", "-errorlog"},
						ImagePullPolicy: "IfNotPresent",
						SecurityContext: &corev1.SecurityContext{
							AllowPrivilegeEscalation: &privledged,
							Privileged:               &privledged,
							ReadOnlyRootFilesystem:   &readonly,
							RunAsGroup:               &user,
							RunAsUser:                &user,
						},
						Env: []corev1.EnvVar{
							{
								Name:  "RDS_NAME",
								Value: name,
							},
							{
								Name:  "OS_USERNAME",
								Value: opts.Username,
							},
							{
								Name:  "OS_PASSWORD",
								Value: opts.Password,
							},
							{
								Name:  "OS_AUTH_URL",
								Value: opts.IdentityEndpoint,
							},
							{
								Name:  "OS_USER_DOMAIN_NAME",
								Value: opts.DomainName,
							},
							{
								Name:  "OS_PROJECT_NAME",
								Value: opts.TenantName,
							},
						},
					},
					{
						Name:            "slowlog",
						Image:           image,
						Command:         []string{"/app/rdslogs", "-slowlog"},
						ImagePullPolicy: "IfNotPresent",
						SecurityContext: &corev1.SecurityContext{
							AllowPrivilegeEscalation: &privledged,
							Privileged:               &privledged,
							ReadOnlyRootFilesystem:   &readonly,
							RunAsGroup:               &user,
							RunAsUser:                &user,
						},
						Env: []corev1.EnvVar{
							{
								Name:  "RDS_NAME",
								Value: name,
							},
							{
								Name:  "OS_USERNAME",
								Value: opts.Username,
							},
							{
								Name:  "OS_PASSWORD",
								Value: opts.Password,
							},
							{
								Name:  "OS_AUTH_URL",
								Value: opts.IdentityEndpoint,
							},
							{
								Name:  "OS_USER_DOMAIN_NAME",
								Value: opts.DomainName,
							},
							{
								Name:  "OS_PROJECT_NAME",
								Value: opts.TenantName,
							},
						},
					},
				},
				RestartPolicy: corev1.RestartPolicyNever,
			},
		},
	}
}

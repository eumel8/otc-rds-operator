package controller

import (
	// "github.com/davecgh/go-spew/spew"
	// dump structs with spew.Dump(opts)

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

func createJob(newRds *rdsv1alpha1.Rds, endpoint string, token string) *batchv1.Job {
	// func createJob(newRds *rdsv1alpha1.Rds, opts golangsdk.AuthOptions) *batchv1.Job {
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
		Spec: createJobSpec(newRds.Name, newRds.Namespace, endpoint, token),
	}
}

func createJobSpec(name string, namespace string, endpoint string, token string) batchv1.JobSpec {
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
								Value: namespace + "_" + name,
							},
							{
								Name:  "OS_TOKEN",
								Value: token,
							},
							{
								Name:  "OS_AUTH_URL",
								Value: endpoint,
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
								Value: namespace + "_" + name,
							},
							{
								Name:  "OS_TOKEN",
								Value: token,
							},
							{
								Name:  "OS_AUTH_URL",
								Value: endpoint,
							},
						},
					},
				},
				RestartPolicy: corev1.RestartPolicyNever,
			},
		},
	}
}

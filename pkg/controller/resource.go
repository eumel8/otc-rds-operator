package controller

import (
	rds "github.com/eumel8/otc-rds-operator/pkg/rds"
	rdsv1alpha1 "github.com/eumel8/otc-rds-operator/pkg/rds/v1alpha1"
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

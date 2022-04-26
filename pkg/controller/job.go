package controller

import (
	rds "github.com/eumel8/otc-rds-operator/pkg/rds"
	rdsv1alpha1 "github.com/eumel8/otc-rds-operator/pkg/rds/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		Spec: createJobSpec(newRds.Name, namespace),
	}
}

func createJobSpec(name string, namespace string) batchv1.JobSpec {
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
						Image:           "ghcr.io/eumel8/otcrdslogs:latest",
						Command:         []string{"/app/rdslogs", "-errorlogs"},
						ImagePullPolicy: "IfNotPresent",
						Env: []corev1.EnvVar{
							{
								Name:  "RDS_NAME",
								Value: name,
							},
						},
					},
				},
				RestartPolicy: corev1.RestartPolicyNever,
			},
		},
	}
}
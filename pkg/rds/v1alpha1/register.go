package v1alpha1

import (
	"github.com/eumel8/otc-rds-operator/pkg/rds"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	SchemeGroupVersion = schema.GroupVersion{
		Group:   rds.GroupName,
		Version: rds.V1alpha1,
	}
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&Rds{},
		&RdsList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)

	return nil
}

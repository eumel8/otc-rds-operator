package v1alpha1

import (
	internalinterfaces "github.com/eumel8/otc-rds-operator/pkg/rds/v1alpha1/apis/informers/externalversions/internalinterfaces"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// Rdss returns a RdsInformer.
	Rdss() RdsInformer
}

type version struct {
	factory          internalinterfaces.SharedInformerFactory
	namespace        string
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// New returns a new Interface.
func New(f internalinterfaces.SharedInformerFactory, namespace string, tweakListOptions internalinterfaces.TweakListOptionsFunc) Interface {
	return &version{factory: f, namespace: namespace, tweakListOptions: tweakListOptions}
}

// Rdss returns a RdsInformer.
func (v *version) Rdss() RdsInformer {
	return &rdsInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

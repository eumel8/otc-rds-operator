package v1alpha1

import (
	"context"
	time "time"

	rdsv1alpha1 "github.com/eumel8/otc-rds-operator/pkg/rds/v1alpha1"
	versioned "github.com/eumel8/otc-rds-operator/pkg/rds/v1alpha1/apis/clientset/versioned"
	internalinterfaces "github.com/eumel8/otc-rds-operator/pkg/rds/v1alpha1/apis/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/eumel8/otc-rds-operator/pkg/rds/v1alpha1/apis/listers/rds/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// RdsInformer provides access to a shared informer and lister for
// Rdss.
type RdsInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.RdsLister
}

type rdsInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewRdsInformer constructs a new informer for Rds type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewRdsInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredRdsInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredRdsInformer constructs a new informer for Rds type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredRdsInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.McspsV1alpha1().Rdss(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.McspsV1alpha1().Rdss(namespace).Watch(context.TODO(), options)
			},
		},
		&rdsv1alpha1.Rds{},
		resyncPeriod,
		indexers,
	)
}

func (f *rdsInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredRdsInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *rdsInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&rdsv1alpha1.Rds{}, f.defaultInformer)
}

func (f *rdsInformer) Lister() v1alpha1.RdsLister {
	return v1alpha1.NewRdsLister(f.Informer().GetIndexer())
}

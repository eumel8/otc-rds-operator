package v1alpha1

import (
	v1alpha1 "github.com/eumel8/otc-rds-operator/pkg/rds/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// RdsLister helps list Rdss.
// All objects returned here must be treated as read-only.
type RdsLister interface {
	// List lists all Rdss in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.Rds, err error)
	// Rdss returns an object that can list and get Rdss.
	Rdss(namespace string) RdsNamespaceLister
	RdsListerExpansion
}

// rdsLister implements the RdsLister interface.
type rdsLister struct {
	indexer cache.Indexer
}

// NewRdsLister returns a new RdsLister.
func NewRdsLister(indexer cache.Indexer) RdsLister {
	return &rdsLister{indexer: indexer}
}

// List lists all Rdss in the indexer.
func (s *rdsLister) List(selector labels.Selector) (ret []*v1alpha1.Rds, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.Rds))
	})
	return ret, err
}

// Rdss returns an object that can list and get Rdss.
func (s *rdsLister) Rdss(namespace string) RdsNamespaceLister {
	return rdsNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// RdsNamespaceLister helps list and get Rdss.
// All objects returned here must be treated as read-only.
type RdsNamespaceLister interface {
	// List lists all Rdss in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.Rds, err error)
	// Get retrieves the Rds from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.Rds, error)
	RdsNamespaceListerExpansion
}

// rdsNamespaceLister implements the RdsNamespaceLister
// interface.
type rdsNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all Rdss in the indexer for a given namespace.
func (s rdsNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.Rds, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.Rds))
	})
	return ret, err
}

// Get retrieves the Rds from the indexer for a given namespace and name.
func (s rdsNamespaceLister) Get(name string) (*v1alpha1.Rds, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("rds"), name)
	}
	return obj.(*v1alpha1.Rds), nil
}

package fake

import (
	v1alpha1 "github.com/eumel8/otc-rds-operator/pkg/rds/v1alpha1/apis/clientset/versioned/typed/rds/v1alpha1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeMcspsV1alpha1 struct {
	*testing.Fake
}

func (c *FakeMcspsV1alpha1) Rdss(namespace string) v1alpha1.RdsInterface {
	return &FakeRdss{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeMcspsV1alpha1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}

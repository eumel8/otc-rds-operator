package v1alpha1

import (
	"context"
	"time"

	v1alpha1 "github.com/eumel8/otc-rds-operator/pkg/rds/v1alpha1"
	scheme "github.com/eumel8/otc-rds-operator/pkg/rds/v1alpha1/apis/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// RdssGetter has a method to return a RdsInterface.
// A group's client should implement this interface.
type RdssGetter interface {
	Rdss(namespace string) RdsInterface
}

// RdsInterface has methods to work with Rds resources.
type RdsInterface interface {
	Create(ctx context.Context, rds *v1alpha1.Rds, opts v1.CreateOptions) (*v1alpha1.Rds, error)
	Update(ctx context.Context, rds *v1alpha1.Rds, opts v1.UpdateOptions) (*v1alpha1.Rds, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.Rds, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.RdsList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.Rds, err error)
	RdsExpansion
}

// rdss implements RdsInterface
type rdss struct {
	client rest.Interface
	ns     string
}

// newRdss returns a Rdss
func newRdss(c *McspsV1alpha1Client, namespace string) *rdss {
	return &rdss{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the rds, and returns the corresponding rds object, and an error if there is any.
func (c *rdss) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.Rds, err error) {
	result = &v1alpha1.Rds{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("rdss").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Rdss that match those selectors.
func (c *rdss) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.RdsList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.RdsList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("rdss").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested rdss.
func (c *rdss) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("rdss").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a rds and creates it.  Returns the server's representation of the rds, and an error, if there is any.
func (c *rdss) Create(ctx context.Context, rds *v1alpha1.Rds, opts v1.CreateOptions) (result *v1alpha1.Rds, err error) {
	result = &v1alpha1.Rds{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("rdss").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(rds).
		Do(ctx).
		Into(result)
// HERE we do the otc api calls from resource.go
	return
}

// Update takes the representation of a rds and updates it. Returns the server's representation of the rds, and an error, if there is any.
func (c *rdss) Update(ctx context.Context, rds *v1alpha1.Rds, opts v1.UpdateOptions) (result *v1alpha1.Rds, err error) {
	result = &v1alpha1.Rds{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("rdss").
		Name(rds.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(rds).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the rds and deletes it. Returns an error if one occurs.
func (c *rdss) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("rdss").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *rdss) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("rdss").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched rds.
func (c *rdss) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.Rds, err error) {
	result = &v1alpha1.Rds{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("rdss").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

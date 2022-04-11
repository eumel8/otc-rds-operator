package fake

import (
	"context"

	v1alpha1 "github.com/eumel8/otc-rds-operator/pkg/rds/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeRdss implements RdsInterface
type FakeRdss struct {
	Fake *FakeMcspsV1alpha1
	ns   string
}

var rdssResource = schema.GroupVersionResource{Group: "otc.mcsps.de", Version: "v1alpha1", Resource: "rdss"}

var rdssKind = schema.GroupVersionKind{Group: "otc.mcsps.de", Version: "v1alpha1", Kind: "Rds"}

// Get takes name of the rds, and returns the corresponding rds object, and an error if there is any.
func (c *FakeRdss) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.Rds, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(rdssResource, c.ns, name), &v1alpha1.Rds{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Rds), err
}

// List takes label and field selectors, and returns the list of Rdss that match those selectors.
func (c *FakeRdss) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.RdsList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(rdssResource, rdssKind, c.ns, opts), &v1alpha1.RdsList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.RdsList{ListMeta: obj.(*v1alpha1.RdsList).ListMeta}
	for _, item := range obj.(*v1alpha1.RdsList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested rdss.
func (c *FakeRdss) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(rdssResource, c.ns, opts))

}

// Create takes the representation of a rds and creates it.  Returns the server's representation of the rds, and an error, if there is any.
func (c *FakeRdss) Create(ctx context.Context, rds *v1alpha1.Rds, opts v1.CreateOptions) (result *v1alpha1.Rds, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(rdssResource, c.ns, rds), &v1alpha1.Rds{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Rds), err
}

// Update takes the representation of a rds and updates it. Returns the server's representation of the rds, and an error, if there is any.
func (c *FakeRdss) Update(ctx context.Context, rds *v1alpha1.Rds, opts v1.UpdateOptions) (result *v1alpha1.Rds, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(rdssResource, c.ns, rds), &v1alpha1.Rds{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Rds), err
}

// Delete takes name of the rds and deletes it. Returns an error if one occurs.
func (c *FakeRdss) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(rdssResource, c.ns, name), &v1alpha1.Rds{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeRdss) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(rdssResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.RdsList{})
	return err
}

// Patch applies the patch and returns the patched rds.
func (c *FakeRdss) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.Rds, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(rdssResource, c.ns, name, pt, data, subresources...), &v1alpha1.Rds{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Rds), err
}

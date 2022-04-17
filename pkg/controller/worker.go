package controller

import (
	"context"
	"fmt"

	rdsv1alpha1 "github.com/eumel8/otc-rds-operator/pkg/rds/v1alpha1"
	// rdsv1alpha1alpha1 "github.com/eumel8/otc-rds-operator/pkg/rds/v1alpha1/apis/clientset/versioned/typed/rds/v1alpha1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
)

const maxRetries = 3

func (c *Controller) runWorker(ctx context.Context) {
	for c.processNextItem(ctx) {
	}
}

func (c *Controller) processNextItem(ctx context.Context) bool {
	obj, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	defer c.queue.Done(obj)

	err := c.processEvent(ctx, obj)
	if err == nil {
		c.logger.Debug("processed item")
		c.queue.Forget(obj)
	} else if c.queue.NumRequeues(obj) < maxRetries {
		c.logger.Errorf("error processing event: %v, retrying", err)
		c.queue.AddRateLimited(obj)
	} else {
		c.logger.Errorf("error processing event: %v, max retries reached", err)
		c.queue.Forget(obj)
		utilruntime.HandleError(err)
	}

	return true
}

func (c *Controller) processEvent(ctx context.Context, obj interface{}) error {
	event, ok := obj.(event)
	if !ok {
		c.logger.Error("unexpected event ", obj)
		return nil
	}
	switch event.eventType {
	case addRds:
		return c.processAddRds(ctx, event.newObj.(*rdsv1alpha1.Rds))
	case delRds:
		return c.processDelRds(ctx, event.newObj.(*rdsv1alpha1.Rds))
	case updateRds:
		return c.processUpdateRds(
			ctx,
			event.oldObj.(*rdsv1alpha1.Rds),
			event.newObj.(*rdsv1alpha1.Rds),
		)
	}
	return nil
}

func (c *Controller) processAddRds(ctx context.Context, rds *rdsv1alpha1.Rds) error {
	err := Create(rds)
	return err
}

func (c *Controller) processDelRds(ctx context.Context, rds *rdsv1alpha1.Rds) error {
	err := Delete(rds)
	return err
}

func (c *Controller) processUpdateRds(
	ctx context.Context,
	oldRds, newRds *rdsv1alpha1.Rds,
) error {
	if !oldRds.HasChanged(newRds) {
		c.logger.Debug("rds has not changed, skipping")
		return nil
	}
	// rdsv1alpha1.Update(ctx, newRds, "")
	oldObj := oldRds.DeepCopy()
	newObj := newRds.DeepCopy()
	fmt.Println(oldObj)
	fmt.Println(newObj)
	err := Update(newRds)
	return err
}

func resourceExists(obj interface{}, indexer cache.Indexer) (bool, error) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		return false, fmt.Errorf("error getting key %v", err)
	}
	_, exists, err := indexer.GetByKey(key)
	return exists, err
}

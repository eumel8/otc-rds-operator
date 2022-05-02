package controller

import (
	"context"
	"fmt"
	"time"

	rdsv1alpha1 "github.com/eumel8/otc-rds-operator/pkg/rds/v1alpha1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
)

const maxRetries = 300

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
	err := c.Create(ctx, rds)
	return err
}

func (c *Controller) processDelRds(ctx context.Context, rds *rdsv1alpha1.Rds) error {
	err := c.Delete(rds)
	return err
}

func (c *Controller) processUpdateRds(
	ctx context.Context,
	oldRds, newRds *rdsv1alpha1.Rds,
) error {
	if err := c.UpdateStatus(ctx, newRds); err != nil {
		err := fmt.Errorf("error update rds status from worker: %v", err)
		return err
	}
	if !oldRds.HasChanged(newRds) {
		c.logger.Debug("rds has not changed, skipping")
		return nil
	}
	oldObj := oldRds.DeepCopy()
	newObj := newRds.DeepCopy()
	if newObj.Status.Status != "ACTIVE" {
		// c.logger.Debug("rds not in ACTIVE state", newObj.Name)
		time.Sleep(30 * time.Second)
		err := fmt.Errorf("rds not in ACTIVE state %v", newObj.Name)
		return err
	}
	err := c.Update(ctx, oldObj, newObj)
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

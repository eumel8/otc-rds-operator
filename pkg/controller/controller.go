package controller

import (
	"context"
	"errors"
	"time"

	"github.com/gotway/gotway/pkg/log"

	rdsv1alpha1 "github.com/eumel8/otc-rds-operator/pkg/rds/v1alpha1"
	rdsv1alpha1clientset "github.com/eumel8/otc-rds-operator/pkg/rds/v1alpha1/apis/clientset/versioned"
	rdsinformers "github.com/eumel8/otc-rds-operator/pkg/rds/v1alpha1/apis/informers/externalversions"

	v1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

type Controller struct {
	kubeClientSet kubernetes.Interface

	rdsInformer cache.SharedIndexInformer
	//jobInformer cache.SharedIndexInformer

	queue workqueue.RateLimitingInterface

	namespace string

	logger log.Logger

	recorder record.EventRecorder
}

func (c *Controller) Run(ctx context.Context, numWorkers int) error {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	c.logger.Info("starting controller")

	c.logger.Info("starting informers")
	for _, i := range []cache.SharedIndexInformer{
		c.rdsInformer,
		//c.jobInformer,
	} {
		go i.Run(ctx.Done())
	}

	c.logger.Info("waiting for informer caches to sync")
	if !cache.WaitForCacheSync(ctx.Done(), []cache.InformerSynced{
		c.rdsInformer.HasSynced,
		//c.jobInformer.HasSynced,
	}...) {
		err := errors.New("failed to wait for informers caches to sync")
		utilruntime.HandleError(err)
		return err
	}

	c.logger.Infof("starting %d workers", numWorkers)
	for i := 0; i < numWorkers; i++ {
		go wait.Until(func() {
			c.runWorker(ctx)
		}, time.Second, ctx.Done())
	}
	c.logger.Info("controller ready")

	<-ctx.Done()
	c.logger.Info("stopping controller")

	return nil
}

func (c *Controller) addRds(obj interface{}) {
	c.logger.Debug("adding rds")
	rds, ok := obj.(*rdsv1alpha1.Rds)
	if !ok {
		c.logger.Errorf("unexpected object %v", obj)
		return
	}
	c.queue.Add(event{
		eventType: addRds,
		newObj:    rds.DeepCopy(),
	})
}

func (c *Controller) delRds(obj interface{}) {
	c.logger.Debug("delete rds")
	rds, ok := obj.(*rdsv1alpha1.Rds)
	if !ok {
		c.logger.Errorf("unexpected object %v", obj)
		return
	}
	c.queue.Add(event{
		eventType: delRds,
		newObj:    rds.DeepCopy(),
	})
}

func (c *Controller) updateRds(oldObj, newObj interface{}) {
	c.logger.Debug("updating rds")
	oldRds, ok := oldObj.(*rdsv1alpha1.Rds)
	if !ok {
		c.logger.Errorf("unexpected new object %v", newObj)
		return
	}
	rds, ok := newObj.(*rdsv1alpha1.Rds)
	if !ok {
		c.logger.Errorf("unexpected new object %v", newObj)
		return
	}
	c.queue.Add(event{
		eventType: updateRds,
		oldObj:    oldRds.DeepCopy(),
		newObj:    rds.DeepCopy(),
	})
}

func New(
	kubeClientSet kubernetes.Interface,
	rdsClientSet rdsv1alpha1clientset.Interface,
	namespace string,
	logger log.Logger,
) *Controller {

	rdsInformerFactory := rdsinformers.NewSharedInformerFactory(
		rdsClientSet,
		10*time.Second,
	)
	rdsInformer := rdsInformerFactory.Mcsps().V1alpha1().Rdss().Informer()

	//kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClientSet, 10*time.Second)
	//jobInformer := kubeInformerFactory.Batch().V1().Jobs().Informer()

	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	eventBroadcaster := record.NewBroadcaster()
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: "rdsoperator"})
	eventBroadcaster.StartStructuredLogging(0)

	klog.Infof("Sending events to api server.")
	eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: kubeClientSet.CoreV1().Events("")})

	ctrl := &Controller{
		kubeClientSet: kubeClientSet,

		rdsInformer: rdsInformer,
		//jobInformer: jobInformer,

		queue: queue,

		namespace: namespace,

		logger: logger,

		recorder: recorder,
	}

	rdsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    ctrl.addRds,
		DeleteFunc: ctrl.delRds,
		UpdateFunc: ctrl.updateRds,
	})

	return ctrl
}

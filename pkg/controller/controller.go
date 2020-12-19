package controller

import (
	"fmt"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	awsservicesv1alpha1 "github.com/anandnilkal/aws-services/pkg/apis/awsservices/v1alpha1"
	clientset "github.com/anandnilkal/aws-services/pkg/generated/clientset/versioned"
	// awsservicesscheme "github.com/anandnilkal/aws-services/pkg/generated/clientset/versioned/scheme"
	informers "github.com/anandnilkal/aws-services/pkg/generated/informers/externalversions/awsservices/v1alpha1"
	// listers "github.com/anandnilkal/aws-services/pkg/generated/listers/awsservices/v1alpha1"
)

type operationType string

const (
	// ADD resource is added
	ADD operationType = "add"
	// DELETE resource is deleted
	DELETE operationType = "delete"
	// UPDATE resource is updated
	UPDATE operationType = "update"
)

// Factory interface for controller
type Factory interface {
	enqueueAdd(interface{})
	enqueueDelete(interface{})
	enqueueUpdate(interface{})
	runWorker()
	processNextItem() bool
	Run(int, <-chan struct{}) error
}

type controllerType struct {
	ID         string
	WorkQueue  workqueue.RateLimitingInterface
	AddFunc    func(string, string)
	DeleteFunc func(string, string)
	UpdateFunc func(string, string)
	Informer   informers.StreamInformer
	ClientSet  clientset.Interface
}

func (c *controllerType) operation(in string) (string, string) {
	out := strings.Split(in, ":")
	switch len(out) {
	case 1:
		return out[0], ""
	case 2:
		return out[0], out[1]
	}
	return "", ""
}

func (c *controllerType) allowedResource(obj interface{}) (bool, error) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			err := fmt.Errorf("error decoding object, invalid type")
			utilruntime.HandleError(err)
			return false, err
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			err := fmt.Errorf("error decoding object tombstone, invalid type")
			utilruntime.HandleError(err)
			return false, err
		}
		klog.V(4).Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}
	klog.V(4).Infof("Processing object: %s", object.GetName())
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		// If this object is not owned by a Foo, we should not do anything more
		// with it.
		if ownerRef.Kind != c.ID {
			return false, fmt.Errorf("controller can not handle the resource of kind: %s", ownerRef.Kind)
		}
	}
	return true, nil
}

func (c *controllerType) enqueueAdd(obj interface{}) {
	if ok, err := c.allowedResource(obj); !ok {
		klog.Errorf(err.Error())
		return
	}
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	key += ":add"
	c.WorkQueue.Add(key)
}

func (c *controllerType) enqueueDelete(obj interface{}) {
	if ok, err := c.allowedResource(obj); !ok {
		klog.Errorf(err.Error())
		return
	}
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	key += ":delete"
	c.WorkQueue.Add(key)
}

func (c *controllerType) enqueueUpdate(obj interface{}) {
	if ok, err := c.allowedResource(obj); !ok {
		klog.Errorf(err.Error())
		return
	}
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	key += ":update"
	c.WorkQueue.Add(key)
}

func (c *controllerType) runWorker() {
	for c.processNextItem() {
	}
}

func (c *controllerType) processNextItem() bool {
	obj, shutdown := c.WorkQueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {

		defer c.WorkQueue.Done(obj)
		var key string
		var ok bool

		if key, ok = obj.(string); !ok {

			c.WorkQueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}

		keyMetadata, operation := c.operation(key)
		namespace, name, err := cache.SplitMetaNamespaceKey(keyMetadata)
		if err != nil {
			utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
			return nil
		}

		switch operation {
		case string(ADD):
			c.AddFunc(name, namespace)
		case string(DELETE):
			c.DeleteFunc(name, namespace)
		case string(UPDATE):
			c.UpdateFunc(name, namespace)
		}

		c.WorkQueue.Forget(obj)
		klog.V(4).Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

func (c *controllerType) Run(threads int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.WorkQueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.V(4).Info("Starting %s controller", c.ID)

	// Wait for the caches to be synced before starting workers
	klog.V(4).Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	// klog.Info("Starting workers")
	// // Launch two workers to process Foo resources
	for i := 0; i < threads; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.V(4).Info("Started workers")
	<-stopCh
	klog.V(4).Info("Shutting down workers")

	return nil
}

func NewControllerFactory(id string, addFunc, deleteFunc, updateFunc func(string, string),
	resourceInformer informers.StreamInformer, resourceClientSet clientset.Interface) Factory {
	return NewControllerFactoryDefault(id, addFunc, deleteFunc, updateFunc, resourceInformer, resourceClientSet)
}

func NewControllerFactoryDefault(id string, addFunc, deleteFunc, updateFunc func(string, string), resourceInformer informers.StreamInformer, resourceClientSet clientset.Interface) Factory {
	controller := &controllerType{
		ID:         id,
		WorkQueue:  workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), id),
		AddFunc:    addFunc,
		DeleteFunc: deleteFunc,
		UpdateFunc: updateFunc,
		Informer: resourceInformer,
		ClientSet: resourceClientSet,
	}

	controller.Informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueAdd,
		UpdateFunc: func(old, new interface{}) {
				newStream := new.(*awsservicesv1alpha1.Stream)
				oldStream := old.(*awsservicesv1alpha1.Stream)
				if newStream.ResourceVersion == oldStream.ResourceVersion {
						// Periodic resync will send update events for all known Deployments.
						// Two different versions of the same Deployment will always have different RVs.
						return
				}
				controller.enqueueUpdate(new)
		},
		DeleteFunc: controller.enqueueDelete,
	})

	return controller
}

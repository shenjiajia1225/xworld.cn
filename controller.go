package main

import (
	//"context"
	"fmt"
	"time"

	//appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	//appsinformers "k8s.io/client-go/informers/apps/v1"
	extclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apiextclientv1 "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	//appslisters "k8s.io/client-go/listers/apps/v1"
	corelisterv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	xworldv1 "xworld.cn/pkg/apis/xworld/v1"
	clientset "xworld.cn/pkg/client/clientset/versioned"
	xworldscheme "xworld.cn/pkg/client/clientset/versioned/scheme"
	getterv1 "xworld.cn/pkg/client/clientset/versioned/typed/xworld/v1"
	"xworld.cn/pkg/client/informers/externalversions"
	//informers "xworld.cn/pkg/client/informers/externalversions/xworld/v1"
	listers "xworld.cn/pkg/client/listers/xworld/v1"
	"xworld.cn/pkg/utils/crd"
)

const controllerAgentName = "xworld-controller"

const (
	SuccessSynced         = "Synced"
	MessageResourceSynced = "XServer synced successfully"

	// ErrResourceExists is used as part of the Event 'reason' when a XServer fails
	// to sync due to a Deployment of the same name already existing.
	ErrResourceExists = "ErrResourceExists"

	// MessageResourceExists is the message used for Events when a resource
	// fails to sync due to a Deployment already existing
	MessageResourceExists = "Resource %q already exists and is not managed by XServer"
	// MessageResourceSynced is the message used for an Event fired when a XServer
	// is synced successfully
)

type Controller struct {
	kubeclientset   kubernetes.Interface
	extclientset    extclientset.Interface
	xworldclientset clientset.Interface

	crdGetter apiextclientv1.CustomResourceDefinitionInterface
	podGetter typedcorev1.PodsGetter
	podLister corelisterv1.PodLister
	podSynced cache.InformerSynced

	xserverGetter  getterv1.XServersGetter
	xserversLister listers.XServerLister
	xserversSynced cache.InformerSynced

	nodeLister corelisterv1.NodeLister
	nodeSynced cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the Kubernetes API.
	recorder record.EventRecorder
}

// NewController returns a new xserver controller
func NewController(
	kubeclientset kubernetes.Interface,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	extclientset extclientset.Interface,
	xworldclientset clientset.Interface,
	xworldInformerFactory externalversions.SharedInformerFactory) *Controller {

	// Create event broadcaster
	// Add xworld-controller types to the default Kubernetes Scheme so Events can be
	// logged for xworld-controller types.
	utilruntime.Must(xworldscheme.AddToScheme(scheme.Scheme))
	// event recorder. 使用kubectl get events 获取内容时会使用
	klog.Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	xserverInformer := xworldInformerFactory.Xworld().V1().XServers()
	pods := kubeInformerFactory.Core().V1().Pods()
	// 组件会先进行一次list，再持续的watch。list动作是否完成就是看各个informer的HasSynced是否为真
	controller := &Controller{
		kubeclientset:   kubeclientset,
		extclientset:    extclientset,
		xworldclientset: xworldclientset,

		crdGetter: extclientset.ApiextensionsV1().CustomResourceDefinitions(),
		podGetter: kubeclientset.CoreV1(),
		podLister: pods.Lister(),
		podSynced: pods.Informer().HasSynced,

		xserversLister: xserverInformer.Lister(),
		xserversSynced: xserverInformer.Informer().HasSynced,

		nodeLister: kubeInformerFactory.Core().V1().Nodes().Lister(),
		nodeSynced: kubeInformerFactory.Core().V1().Nodes().Informer().HasSynced,

		workqueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "XServers"),
		recorder:  recorder,
	}

	klog.Info("Setting up event handlers")
	// Set up an event handler for when XServer resources change
	xserverInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueXServer,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueXServer(new)
		},
	})
	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting XWorld controller:%v", threadiness)

	err := crd.WaitForEstablishedCRD(c.crdGetter, "xservers.agones.dev")
	if err != nil {
		return err
	}

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.xserversSynced, c.podSynced, c.nodeSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting workers")
	// Launch two workers to process XServer resources
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// XServer resource to be synced.
		if err := c.syncHandler(key); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the XServer resource
// with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the xserver resource with this namespace/name
	xserver, err := c.xserversLister.XServers(namespace).Get(name)
	if err != nil {
		// The XServer resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("xserver '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	imageName := xserver.Spec.Image
	if imageName == "" {
		// We choose to absorb the error here as the worker would requeue the
		// resource otherwise. Instead, the next time the resource is updated
		// the resource will be queued again.
		utilruntime.HandleError(fmt.Errorf("%s: image name must be specified", key))
		return nil
	}

	// Finally, we update the status block of the XServer resource to reflect the
	// current state of the world
	err = c.updateXServerStatus(xserver, imageName)
	if err != nil {
		return err
	}

	c.recorder.Event(xserver, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil
}

func (c *Controller) updateXServerStatus(xserver *xworldv1.XServer, containerName string) error {
	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	xserverCopy := xserver.DeepCopy()
	xserverCopy.Status.Address = containerName
	// If the CustomResourceSubresources feature gate is not enabled,
	// we must use Update instead of UpdateStatus to update the Status block of the XServer resource.
	// UpdateStatus will not allow changes to the Spec of the resource,
	// which is ideal for ensuring nothing other than resource status has been updated.
	_, err := c.xworldclientset.XworldV1().XServers(xserver.Namespace).Update(xserverCopy)
	return err
}

// enqueueXServer takes a XServer resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than XServer.
func (c *Controller) enqueueXServer(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

// handleObject will take any resource implementing metav1.Object and attempt
// to find the XServer resource that 'owns' it. It does this by looking at the
// objects metadata.ownerReferences field for an appropriate OwnerReference.
// It then enqueues that XServer resource to be processed. If the object does not
// have an appropriate OwnerReference, it will simply be skipped.
func (c *Controller) handleObject(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		klog.V(4).Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}
	klog.V(4).Infof("Processing object: %s", object.GetName())
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		// If this object is not owned by a XServer, we should not do anything more
		// with it.
		if ownerRef.Kind != "XServer" {
			return
		}

		xserver, err := c.xserversLister.XServers(object.GetNamespace()).Get(ownerRef.Name)
		if err != nil {
			klog.V(4).Infof("ignoring orphaned object '%s' of xserver '%s'", object.GetSelfLink(), ownerRef.Name)
			return
		}

		c.enqueueXServer(xserver)
		return
	}
}

package statefulset

import (
	"fmt"
	"time"

	constants "github.com/att-cloudnative-labs/kconfig-controller/internal/app/kconfig-controller/controller"
	kconfigv1alpha1 "github.com/att-cloudnative-labs/kconfig-controller/pkg/apis/kconfigcontroller/v1alpha1"
	clientset "github.com/att-cloudnative-labs/kconfig-controller/pkg/client/clientset/versioned"
	kcscheme "github.com/att-cloudnative-labs/kconfig-controller/pkg/client/clientset/versioned/scheme"
	informers "github.com/att-cloudnative-labs/kconfig-controller/pkg/client/informers/externalversions/kconfigcontroller/v1alpha1"
	listers "github.com/att-cloudnative-labs/kconfig-controller/pkg/client/listers/kconfigcontroller/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	event "k8s.io/client-go/kubernetes/typed/core/v1"
	appslisters "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

// Controller controller object
type Controller struct {
	recorder record.EventRecorder

	stdclient kubernetes.Interface
	kcclient  clientset.Interface

	statefulSetsLister       appslisters.StatefulSetLister
	statefulSetBindingLister listers.StatefulSetBindingLister

	statefulSetsSynced        cache.InformerSynced
	statefulSetBindingsSynced cache.InformerSynced

	workqueue workqueue.RateLimitingInterface
}

// NewController returns a new controller object
func NewController(
	stdclient kubernetes.Interface,
	kcclient clientset.Interface,
	statefulSetInformer appsinformers.StatefulSetInformer,
	statefulSetBindingInformer informers.StatefulSetBindingInformer) *Controller {

	runtime.Must(scheme.AddToScheme(kcscheme.Scheme))
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&event.EventSinkImpl{Interface: stdclient.CoreV1().Events("")})

	controller := &Controller{
		recorder:                  eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "Kconfig-StatefulSet-Controller"}),
		stdclient:                 stdclient,
		kcclient:                  kcclient,
		statefulSetsLister:        statefulSetInformer.Lister(),
		statefulSetBindingLister:  statefulSetBindingInformer.Lister(),
		statefulSetsSynced:        statefulSetInformer.Informer().HasSynced,
		statefulSetBindingsSynced: statefulSetBindingInformer.Informer().HasSynced,
		workqueue:                 workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "StatefulSet"),
	}

	klog.Info("Setting up event handlers")
	statefulSetInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueStatefulSet,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueStatefulSet(new)
		},
	})
	return controller
}

func (c *Controller) enqueueStatefulSet(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting StatefulSet controller")

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.statefulSetsSynced, c.statefulSetBindingsSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting workers")
	// Launch two workers to process Foo resources
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
			runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// Foo resource to be synced.
		if err := c.syncHandler(key); err != nil {
			return fmt.Errorf("error syncing '%s': %s", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two.
func (c *Controller) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the StatefulSet resource with this namespace/name
	statefulSet, err := c.statefulSetsLister.StatefulSets(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			runtime.HandleError(fmt.Errorf("statefulSet '%s' in work queue no longer exists", key))
			return nil
		}
		return err
	}

	// If the statefulSet doesn't have the kconfig annotation, disregard
	if statefulSet.Annotations[constants.KconfigEnabledDeploymentAnnotation] != "true" {
		return nil
	}

	if err = c.handleStatefulSet(statefulSet); err != nil {
		return err
	}

	c.recorder.Event(statefulSet, corev1.EventTypeNormal, constants.SuccessSynced, constants.MessageStatefulSetResourceSynced)
	return nil
}

func (c *Controller) handleStatefulSet(statefulSet *appsv1.StatefulSet) error {
	namespace := statefulSet.Namespace
	name := statefulSet.Name
	statefulSetBinding, err := c.statefulSetBindingLister.StatefulSetBindings(namespace).Get(name)
	if err != nil && errors.IsNotFound(err) {
		statefulSetBinding, err = c.createStatefulSetBinding(statefulSet)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	statefulSetBindingCopy := statefulSetBinding.DeepCopy()
	statefulSetBindingCopy.SetLabels(statefulSetBinding.GetLabels())
	_, err = c.kcclient.KconfigcontrollerV1alpha1().StatefulSetBindings(namespace).Update(statefulSetBindingCopy)
	return err
}

func (c *Controller) createStatefulSetBinding(statefulSet *appsv1.StatefulSet) (*kconfigv1alpha1.StatefulSetBinding, error) {
	namespace := statefulSet.Namespace
	name := statefulSet.Name
	statefulSetBinding := &kconfigv1alpha1.StatefulSetBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: kconfigv1alpha1.SchemeGroupVersion.String(),
			Kind:       "StatefulSetBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    statefulSet.Labels,
		},
		Spec: kconfigv1alpha1.KconfigBindingSpec{
			KconfigEnvsMap: make(map[string]kconfigv1alpha1.KconfigEnvs),
		},
	}
	return c.kcclient.KconfigcontrollerV1alpha1().StatefulSetBindings(namespace).Create(statefulSetBinding)
}

func (c *Controller) deleteHandler(obj interface{}) {
	d, ok := obj.(*appsv1.StatefulSet)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			runtime.HandleError(fmt.Errorf("Couldn't get object from tombstone %#v", obj))
			return
		}
		d, ok = tombstone.Obj.(*appsv1.StatefulSet)
		if !ok {
			runtime.HandleError(fmt.Errorf("Tombstone contained object that is not a StatefulSet %#v", obj))
			return
		}
	}
	klog.Infof("Deleting statefulSetBinding %s", d.Name)
	kcb, err := c.statefulSetBindingLister.StatefulSetBindings(d.Namespace).Get(d.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			return
		}
		runtime.HandleError(fmt.Errorf("Error removing corresponding statefulSetBinding: %s", err.Error()))
		return
	}
	err = c.kcclient.KconfigcontrollerV1alpha1().StatefulSetBindings(kcb.Namespace).Delete(kcb.Name, &metav1.DeleteOptions{})
	if err != nil {
		runtime.HandleError(fmt.Errorf("Error removing corresponding statefulSetBinding: %s", err.Error()))
	}
}

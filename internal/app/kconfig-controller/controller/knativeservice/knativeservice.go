package knativeservice

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	event "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"

	constants "github.com/gbraxton/kconfig/internal/app/kconfig-controller/controller"
	kcv1alpha1 "github.com/gbraxton/kconfig/pkg/apis/kconfigcontroller/v1alpha1"
	clientset "github.com/gbraxton/kconfig/pkg/client/clientset/versioned"
	kcinformers "github.com/gbraxton/kconfig/pkg/client/informers/externalversions/kconfigcontroller/v1alpha1"
	kclisters "github.com/gbraxton/kconfig/pkg/client/listers/kconfigcontroller/v1alpha1"
	knv1alpha1 "github.com/knative/serving/pkg/apis/serving/v1alpha1"
	knclientset "github.com/knative/serving/pkg/client/clientset/versioned"
	knscheme "github.com/knative/serving/pkg/client/clientset/versioned/scheme"
	kninformers "github.com/knative/serving/pkg/client/informers/externalversions/serving/v1alpha1"
	knlisters "github.com/knative/serving/pkg/client/listers/serving/v1alpha1"
)

// Controller controller object
type Controller struct {
	recorder             record.EventRecorder
	stdclient            kubernetes.Interface
	knclient             knclientset.Interface
	kcclient             clientset.Interface
	kserviceLister       knlisters.ServiceLister
	kconfigBindingLister kclisters.KconfigBindingLister
	kserviceSynced       cache.InformerSynced
	workqueue            workqueue.RateLimitingInterface
}

// NewController returns a new controller object
func NewController(
	stdclient kubernetes.Interface,
	knclient knclientset.Interface,
	kcclient clientset.Interface,
	kserviceInformer kninformers.ServiceInformer,
	kconfigBindingInformer kcinformers.KconfigBindingInformer) *Controller {

	// runtime.Must(scheme.AddToScheme(knscheme.Scheme))
	runtime.Must(knscheme.AddToScheme(scheme.Scheme))
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&event.EventSinkImpl{Interface: stdclient.CoreV1().Events("")})

	controller := &Controller{
		recorder:             eventBroadcaster.NewRecorder(knscheme.Scheme, corev1.EventSource{Component: "Kconfig-KnativeService-Controller"}),
		stdclient:            stdclient,
		knclient:             knclient,
		kcclient:             kcclient,
		kserviceLister:       kserviceInformer.Lister(),
		kconfigBindingLister: kconfigBindingInformer.Lister(),
		kserviceSynced:       kserviceInformer.Informer().HasSynced,
		workqueue:            workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Service"),
	}

	klog.Info("Setting up event handlers")
	kserviceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueKService,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueKService(new)
		},
	})
	return controller
}

func (c *Controller) enqueueKService(obj interface{}) {
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
	klog.Info("Starting KnativeService controller")

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.kserviceSynced); !ok {
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

	// Get the KnativeService resource with this namespace/name
	kservice, err := c.kserviceLister.Services(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			runtime.HandleError(fmt.Errorf("knativeservice '%s' in work queue no longer exists", key))
			return nil
		}
		return err
	}

	// If the knativeService doesn't have the kconfig env annotation, disregard
	if kservice.Annotations[constants.KconfigEnabledDeploymentAnnotation] != "true" {
		return nil
	}

	if err = c.handleKnativeService(kservice); err != nil {
		return err
	}

	c.recorder.Event(kservice, corev1.EventTypeNormal, constants.SuccessSynced, constants.MessageKnativeServiceResourceSynced)
	return nil
}

func (c *Controller) handleKnativeService(knativeService *knv1alpha1.Service) error {
	namespace := knativeService.Namespace
	name := knativeService.Name
	kconfigbinding, err := c.kconfigBindingLister.KconfigBindings(namespace).Get(name)
	if err != nil && errors.IsNotFound(err) {
		kconfigbinding, err = c.createKconfigBinding(knativeService)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	kconfigbindingCopy := kconfigbinding.DeepCopy()
	kconfigbindingCopy.SetLabels(kconfigbinding.GetLabels())
	_, err = c.kcclient.KconfigcontrollerV1alpha1().KconfigBindings(namespace).Update(kconfigbindingCopy)
	return err
}

func (c *Controller) createKconfigBinding(knativeService *knv1alpha1.Service) (*kcv1alpha1.KconfigBinding, error) {
	namespace := knativeService.Namespace
	name := knativeService.Name
	kconfigbinding := &kcv1alpha1.KconfigBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: kcv1alpha1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    knativeService.Labels,
		},
		Spec: kcv1alpha1.KconfigBindingSpec{
			KconfigEnvsMap: make(map[string]kcv1alpha1.KconfigEnvs),
		},
	}
	return c.kcclient.KconfigcontrollerV1alpha1().KconfigBindings(namespace).Create(kconfigbinding)
}

func (c *Controller) deleteHandler(obj interface{}) {
	d, ok := obj.(*knv1alpha1.Service)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			runtime.HandleError(fmt.Errorf("Couldn't get object from tombstone %#v", obj))
			return
		}
		d, ok = tombstone.Obj.(*knv1alpha1.Service)
		if !ok {
			runtime.HandleError(fmt.Errorf("Tombstone contained object that is not a KnativeService %#v", obj))
			return
		}
	}
	klog.Infof("Deleting kconfigbinding %s", d.Name)
	kcb, err := c.kconfigBindingLister.KconfigBindings(d.Namespace).Get(d.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			return
		}
		runtime.HandleError(fmt.Errorf("Error removing corresponding kconfigbinding: %s", err.Error()))
		return
	}
	err = c.kcclient.KconfigcontrollerV1alpha1().KconfigBindings(kcb.Namespace).Delete(kcb.Name, &metav1.DeleteOptions{})
	if err != nil {
		runtime.HandleError(fmt.Errorf("Error removing corresponding kconfigbinding: %s", err.Error()))
	}
}

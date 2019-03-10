package kconfigbinding

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/gbraxton/kconfig/internal/app/kconfig-controller/controller"
	kconfigv1alpha1 "github.com/gbraxton/kconfig/pkg/apis/kconfigcontroller/v1alpha1"
	clientset "github.com/gbraxton/kconfig/pkg/client/clientset/versioned"
	kcscheme "github.com/gbraxton/kconfig/pkg/client/clientset/versioned/scheme"
	informers "github.com/gbraxton/kconfig/pkg/client/informers/externalversions/kconfigcontroller/v1alpha1"
	listers "github.com/gbraxton/kconfig/pkg/client/listers/kconfigcontroller/v1alpha1"
	"github.com/gbraxton/kconfig/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	appinformers "k8s.io/client-go/informers/apps/v1"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	event "k8s.io/client-go/kubernetes/typed/core/v1"
	applisters "k8s.io/client-go/listers/apps/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

// Controller is the controller implementation for KconfigBinding resources
type Controller struct {
	recorder record.EventRecorder

	stdclient kubernetes.Interface
	kcclient  clientset.Interface

	configmaplister      corelisters.ConfigMapLister
	secretlister         corelisters.SecretLister
	deploymentslister    applisters.DeploymentLister
	kconfigBindingLister listers.KconfigBindingLister

	configmapsSynced      cache.InformerSynced
	secretsSynced         cache.InformerSynced
	deploymentsSynced     cache.InformerSynced
	kconfigBindingsSynced cache.InformerSynced

	workqueue workqueue.RateLimitingInterface
}

// NewController returns a new controller
func NewController(
	stdclient kubernetes.Interface,
	kcclient clientset.Interface,
	configmapInformer coreinformers.ConfigMapInformer,
	secretInformer coreinformers.SecretInformer,
	deploymentInformer appinformers.DeploymentInformer,
	kconfigBindingInformer informers.KconfigBindingInformer) *Controller {

	// runtime.Must(scheme.AddToScheme(kcscheme.Scheme))
	runtime.Must(kcscheme.AddToScheme(scheme.Scheme))
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&event.EventSinkImpl{Interface: stdclient.CoreV1().Events("")})

	controller := &Controller{
		recorder:              eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "KconfigBinding-Controller"}),
		stdclient:             stdclient,
		kcclient:              kcclient,
		configmaplister:       configmapInformer.Lister(),
		secretlister:          secretInformer.Lister(),
		deploymentslister:     deploymentInformer.Lister(),
		kconfigBindingLister:  kconfigBindingInformer.Lister(),
		configmapsSynced:      configmapInformer.Informer().HasSynced,
		secretsSynced:         secretInformer.Informer().HasSynced,
		deploymentsSynced:     deploymentInformer.Informer().HasSynced,
		kconfigBindingsSynced: kconfigBindingInformer.Informer().HasSynced,
		workqueue:             workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "KconfigBinding"),
	}

	klog.Info("Setting up event handlers")
	kconfigBindingInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueKconfigBinding,
		UpdateFunc: func(old, new interface{}) {
			newKcb := new.(*kconfigv1alpha1.KconfigBinding)
			oldKcb := old.(*kconfigv1alpha1.KconfigBinding)
			if newKcb.ResourceVersion == oldKcb.ResourceVersion {
				// Periodic resync will send update events for all known KconfigBindings.
				// Two different versions of the same Deployment will always have different RVs.
				return
			}
			controller.enqueueKconfigBinding(new)
		},
	})
	return controller
}

func (c *Controller) enqueueKconfigBinding(obj interface{}) {
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
	klog.Info("Starting KconfigBinding controller")

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.configmapsSynced, c.secretsSynced, c.deploymentsSynced, c.kconfigBindingsSynced); !ok {
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

	// Get the KconfigBinding resource with this namespace/name
	kconfigBinding, err := c.kconfigBindingLister.KconfigBindings(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			runtime.HandleError(fmt.Errorf("kconfigbinding '%s' in work queue no longer exists", key))
			return nil
		}
		return err
	}

	err = c.processKconfigBinding(kconfigBinding)
	if err != nil {
		return err
	}

	c.recorder.Event(kconfigBinding, corev1.EventTypeNormal, controller.SuccessSynced, controller.MessageDeploymentResourceSynced)
	return nil
}

func (c *Controller) processKconfigBinding(kconfigBinding *kconfigv1alpha1.KconfigBinding) error {
	// convert kconfigEnvs map to array of values
	kconfigEnvs := []kconfigv1alpha1.KconfigEnvs{}
	for _, kconfigEnv := range kconfigBinding.Spec.KconfigEnvsMap {
		kconfigEnvs = append(kconfigEnvs, kconfigEnv)
	}

	// Sort env array
	sort.Sort(util.ByLevel(kconfigEnvs))

	// Create final env array
	envArray := []corev1.EnvVar{}
	// envRefsVersions tracks changes to refs among various KconfigEnvs for updating deployment
	// template when reference value changes but no keys have changed. It is composed of each
	// KconfigEnv's envRefsVersion concatenated into a single string
	envRefsVersions := ""
	for _, kconfigEnv := range kconfigEnvs {
		envRefsVersions += strconv.FormatInt(kconfigEnv.EnvRefsVersion, 10)
		envArray = append(envArray, kconfigEnv.Envs...)
	}

	deployment, err := c.deploymentslister.Deployments(kconfigBinding.Namespace).Get(kconfigBinding.Name)
	if err != nil {
		return err
	}
	if deployment.GetAnnotations()[controller.KconfigEnabledDeploymentAnnotation] != "true" {
		klog.Infof("deployment %s %s does not have kconfig enabled", deployment.GetNamespace(), deployment.GetName())
		return nil
	}
	deploymentCopy := deployment.DeepCopy()
	if deploymentCopy.Spec.Template.ObjectMeta.Annotations == nil {
		deploymentCopy.Spec.Template.ObjectMeta.Annotations = make(map[string]string, 0)
	}
	deploymentCopy.Spec.Template.ObjectMeta.Annotations[controller.KconfigEnvRefVersionAnnotation] = envRefsVersions
	deploymentCopy.Spec.Template.Spec.Containers[0].Env = envArray
	if _, err = c.stdclient.Apps().Deployments(deploymentCopy.Namespace).Update(deploymentCopy); err != nil {
		return err
	}
	return nil
}

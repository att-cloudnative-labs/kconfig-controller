package knativeservicebinding

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/att-cloudnative-labs/kconfig-controller/internal/app/kconfig-controller/controller"
	kcv1alpha1 "github.com/att-cloudnative-labs/kconfig-controller/pkg/apis/kconfigcontroller/v1alpha1"
	kcclientset "github.com/att-cloudnative-labs/kconfig-controller/pkg/client/clientset/versioned"
	kcscheme "github.com/att-cloudnative-labs/kconfig-controller/pkg/client/clientset/versioned/scheme"
	kcinformers "github.com/att-cloudnative-labs/kconfig-controller/pkg/client/informers/externalversions/kconfigcontroller/v1alpha1"
	kclisters "github.com/att-cloudnative-labs/kconfig-controller/pkg/client/listers/kconfigcontroller/v1alpha1"
	"github.com/att-cloudnative-labs/kconfig-controller/pkg/util"
	knv1alpha1 "github.com/knative/serving/pkg/apis/serving/v1alpha1"
	knclientset "github.com/knative/serving/pkg/client/clientset/versioned"
	kninformers "github.com/knative/serving/pkg/client/informers/externalversions/serving/v1alpha1"
	knlisters "github.com/knative/serving/pkg/client/listers/serving/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	event "k8s.io/client-go/kubernetes/typed/core/v1"
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
	kcclient  kcclientset.Interface
	knclient  knclientset.Interface

	configmaplister             corelisters.ConfigMapLister
	secretlister                corelisters.SecretLister
	knativeServicelister        knlisters.ServiceLister
	knativeServiceBindingLister kclisters.KnativeServiceBindingLister

	configmapsSynced             cache.InformerSynced
	secretsSynced                cache.InformerSynced
	knativeServicesSynced        cache.InformerSynced
	knativeServiceBindingsSynced cache.InformerSynced

	workqueue workqueue.RateLimitingInterface
}

// NewController returns a new controller
func NewController(
	stdclient kubernetes.Interface,
	kcclient kcclientset.Interface,
	knclient knclientset.Interface,
	configmapInformer coreinformers.ConfigMapInformer,
	secretInformer coreinformers.SecretInformer,
	knativeServiceInformer kninformers.ServiceInformer,
	knativeServiceBindingInformer kcinformers.KnativeServiceBindingInformer) *Controller {

	// runtime.Must(scheme.AddToScheme(kcscheme.Scheme))
	runtime.Must(kcscheme.AddToScheme(scheme.Scheme))
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&event.EventSinkImpl{Interface: stdclient.CoreV1().Events("")})

	controller := &Controller{
		recorder:                     eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "KnativeServiceBinding-Controller"}),
		stdclient:                    stdclient,
		kcclient:                     kcclient,
		knclient:                     knclient,
		configmaplister:              configmapInformer.Lister(),
		secretlister:                 secretInformer.Lister(),
		knativeServicelister:         knativeServiceInformer.Lister(),
		knativeServiceBindingLister:  knativeServiceBindingInformer.Lister(),
		configmapsSynced:             configmapInformer.Informer().HasSynced,
		secretsSynced:                secretInformer.Informer().HasSynced,
		knativeServicesSynced:        knativeServiceInformer.Informer().HasSynced,
		knativeServiceBindingsSynced: knativeServiceBindingInformer.Informer().HasSynced,
		workqueue:                    workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "DeploymentBinding"),
	}

	klog.Info("Setting up event handlers")
	knativeServiceBindingInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueKnativeServiceBinding,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueKnativeServiceBinding(new)
		},
	})
	return controller
}

func (c *Controller) enqueueKnativeServiceBinding(obj interface{}) {
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
	klog.Info("Starting DeploymentBinding controller")

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.configmapsSynced, c.secretsSynced, c.knativeServicesSynced, c.knativeServiceBindingsSynced); !ok {
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

	// Get the KnativeServiceBinding resource with this namespace/name
	knativeServiceBinding, err := c.knativeServiceBindingLister.KnativeServiceBindings(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			runtime.HandleError(fmt.Errorf("knativeServiceBinding '%s' in work queue no longer exists", key))
			return nil
		}
		return err
	}

	err = c.processKnativeServiceBinding(knativeServiceBinding)
	if err != nil {
		return err
	}

	c.recorder.Event(knativeServiceBinding, corev1.EventTypeNormal, controller.SuccessSynced, controller.MessageDeploymentResourceSynced)
	return nil
}

func (c *Controller) processKnativeServiceBinding(knativeServiceBinding *kcv1alpha1.KnativeServiceBinding) error {
	// convert kconfigEnvs map to array of values
	kconfigEnvs := []kcv1alpha1.KconfigEnvs{}
	for _, kconfigEnv := range knativeServiceBinding.Spec.KconfigEnvsMap {
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

	knativeService, err := c.knativeServicelister.Services(knativeServiceBinding.Namespace).Get(knativeServiceBinding.Name)
	if err != nil {
		return err
	}
	if knativeService.GetAnnotations()[controller.KconfigEnabledDeploymentAnnotation] != "true" {
		klog.Infof("knativeService %s %s does not have kconfig enabled", knativeService.GetNamespace(), knativeService.GetName())
		return nil
	}
	knativeServiceCopy := knativeService.DeepCopy()

	if knativeServiceCopy.Spec.RunLatest != nil {
		knativeServiceCopy.Spec.RunLatest.Configuration = applyKconfigEnvToConfiguration(knativeServiceCopy.Spec.RunLatest.Configuration, envArray, envRefsVersions)
	}
	if knativeServiceCopy.Spec.Release != nil {
		knativeServiceCopy.Spec.Release.Configuration = applyKconfigEnvToConfiguration(knativeServiceCopy.Spec.Release.Configuration, envArray, envRefsVersions)
	}

	if _, err = c.knclient.Serving().Services(knativeServiceCopy.Namespace).Update(knativeServiceCopy); err != nil {
		return err
	}
	return nil
}

func applyKconfigEnvToConfiguration(configuration knv1alpha1.ConfigurationSpec, envArray []corev1.EnvVar, envRefsVersions string) knv1alpha1.ConfigurationSpec {
	// I dont get why this is an error when RevisionTemplate is defined as a pointer to a RevisionTemplateSpec (should be able to be nil)
	// if configuration.RevisionTemplate == nil {
	// 	configuration.RevisionTemplate = &knv1alpha1.RevisionTemplateSpec{}
	// }
	if configuration.RevisionTemplate.ObjectMeta.Annotations == nil {
		configuration.RevisionTemplate.ObjectMeta.Annotations = make(map[string]string, 0)
	}
	configuration.RevisionTemplate.ObjectMeta.Annotations[controller.KconfigEnvRefVersionAnnotation] = envRefsVersions
	// This too
	// if configuration.RevisionTemplate.Spec.Container == nil {
	// 	configuration.RevisionTemplate.Spec.Container = &corev1.Container{}
	// }
	configuration.RevisionTemplate.Spec.Container.Env = envArray
	return configuration
}

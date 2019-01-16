package kconfig

import (
	"encoding/base64"
	"fmt"
	"reflect"
	"time"

	"github.com/gbraxton/kconfig/internal/app/kconfig-controller/controller"
	"github.com/gbraxton/kconfig/pkg/apis/kconfigcontroller/v1alpha1"
	"github.com/gbraxton/kconfig/pkg/client/clientset/versioned"
	clientset "github.com/gbraxton/kconfig/pkg/client/clientset/versioned"
	kcscheme "github.com/gbraxton/kconfig/pkg/client/clientset/versioned/scheme"
	informers "github.com/gbraxton/kconfig/pkg/client/informers/externalversions/kconfigcontroller/v1alpha1"
	listers "github.com/gbraxton/kconfig/pkg/client/listers/kconfigcontroller/v1alpha1"
	"github.com/gbraxton/kconfig/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// Controller controller object
type Controller struct {
	recorder record.EventRecorder

	stdclient kubernetes.Interface
	kcclient  clientset.Interface

	configmaplister      corelisters.ConfigMapLister
	secretlister         corelisters.SecretLister
	kconfiglister        listers.KconfigLister
	kconfigbindinglister listers.KconfigBindingLister

	configmapsSynced      cache.InformerSynced
	secretsSynced         cache.InformerSynced
	kconfigsSynced        cache.InformerSynced
	kconfigBindingsSynced cache.InformerSynced

	workqueue workqueue.RateLimitingInterface
}

// NewController new controller
func NewController(
	stdclient kubernetes.Interface,
	kcclient versioned.Interface,
	configmapInformer coreinformers.ConfigMapInformer,
	secretInformer coreinformers.SecretInformer,
	kconfigInformer informers.KconfigInformer,
	kconfigBindingInformer informers.KconfigBindingInformer) *Controller {

	runtime.Must(scheme.AddToScheme(kcscheme.Scheme))
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&event.EventSinkImpl{Interface: stdclient.CoreV1().Events("")})

	controller := &Controller{
		recorder:              eventBroadcaster.NewRecorder(kcscheme.Scheme, corev1.EventSource{Component: "kconfig-controller"}),
		stdclient:             stdclient,
		kcclient:              kcclient,
		configmaplister:       configmapInformer.Lister(),
		secretlister:          secretInformer.Lister(),
		kconfiglister:         kconfigInformer.Lister(),
		kconfigbindinglister:  kconfigBindingInformer.Lister(),
		configmapsSynced:      configmapInformer.Informer().HasSynced,
		secretsSynced:         secretInformer.Informer().HasSynced,
		kconfigsSynced:        kconfigInformer.Informer().HasSynced,
		kconfigBindingsSynced: kconfigBindingInformer.Informer().HasSynced,
		workqueue:             workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Kconfig"),
	}

	klog.Info("Setting up event handlers")
	kconfigInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueKconfig,
		UpdateFunc: func(old, new interface{}) {
			newKconfig := new.(*v1alpha1.Kconfig)
			oldKconfig := old.(*v1alpha1.Kconfig)
			if newKconfig.ResourceVersion == oldKconfig.ResourceVersion {
				// Periodic resync will send update events for all known Kconfigs.
				// Two different versions of the same Deployment will always have different RVs.
				return
			}
			controller.enqueueKconfig(new)
		},
		DeleteFunc: controller.deleteHandler,
	})
	return controller
}

func (c *Controller) enqueueKconfig(obj interface{}) {
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
	klog.Info("Starting kconfig controller")

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.configmapsSynced, c.secretsSynced, c.kconfigsSynced, c.kconfigBindingsSynced); !ok {
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

	// Get the Kconfig resource with this namespace/name
	kconfig, err := c.kconfiglister.Kconfigs(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			runtime.HandleError(fmt.Errorf("kconfig '%s' in work queue no longer exists", key))
			return nil
		}
		return err
	}

	if err := c.processKconfig(kconfig); err != nil {
		klog.Errorf("Error processing kconfig: %s", err.Error())
		return err
	}

	c.recorder.Event(kconfig, corev1.EventTypeNormal, controller.SuccessSynced, controller.MessageKconfigResourceSynced)
	return nil
}

func (c *Controller) processKconfig(kconfig *v1alpha1.Kconfig) error {
	updatedEnvConfigs, actions := extractExternalResourceActions(kconfig.Spec.EnvConfigs)
	if err := c.executeExternalResourceActions(actions, kconfig.Namespace); err != nil {
		return err
	}

	if !reflect.DeepEqual(kconfig.Spec.EnvConfigs, updatedEnvConfigs) {
		updatedKconfig := kconfig.DeepCopy()
		updatedKconfig.Spec.EnvConfigs = updatedEnvConfigs
		if _, err := c.kcclient.KconfigcontrollerV1alpha1().Kconfigs(kconfig.Namespace).Update(updatedKconfig); err != nil {
			return err
		}
	}
	kconfigEnvs := buildKconfigEnv(kconfig.Spec.Level, updatedEnvConfigs)
	err := c.updateKconfigBindings(kconfig, kconfigEnvs)
	return err
}

func (c *Controller) updateKconfigBindings(kconfig *v1alpha1.Kconfig, kconfigEnvs v1alpha1.KconfigEnvs) error {
	envKey := util.GetEnvKey(kconfig.Namespace, kconfig.Name)
	selector, err := metav1.LabelSelectorAsSelector(&kconfig.Spec.Selector)
	if err != nil {
		return err
	}
	kconfigBindings, err := c.kconfigbindinglister.KconfigBindings(kconfig.Namespace).List(selector)
	if err != nil {
		return err
	}
	for _, kconfigBinding := range kconfigBindings {
		kconfigBindingCopy := kconfigBinding.DeepCopy()
		if !reflect.DeepEqual(kconfigBinding.Spec.KconfigEnvsMap[envKey], kconfigEnvs) {
			if kconfigBindingCopy.Spec.KconfigEnvsMap == nil {
				kconfigBindingCopy.Spec.KconfigEnvsMap = make(map[string]v1alpha1.KconfigEnvs)
			}
			kconfigBindingCopy.Spec.KconfigEnvsMap[envKey] = kconfigEnvs
			_, err := c.kcclient.KconfigcontrollerV1alpha1().KconfigBindings(kconfig.Namespace).Update(kconfigBindingCopy)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Controller) removeKconfigEnvsFromKconfigBindings(kconfig *v1alpha1.Kconfig) {
	envKey := util.GetEnvKey(kconfig.Namespace, kconfig.Name)
	selector, err := metav1.LabelSelectorAsSelector(&kconfig.Spec.Selector)
	if err != nil {
		runtime.HandleError(fmt.Errorf("Error removing KconfigEnvs from KconfigBinding: %+v", err.Error()))
		return
	}
	kconfigBindings, err := c.kconfigbindinglister.KconfigBindings(kconfig.Namespace).List(selector)
	if err != nil {
		runtime.HandleError(fmt.Errorf("Error removing KconfigEnvs from KconfigBinding: %+v", err.Error()))
		return
	}
	for _, kconfigBinding := range kconfigBindings {
		kconfigBindingCopy := kconfigBinding.DeepCopy()
		if kconfigBindingCopy.Spec.KconfigEnvsMap == nil {
			continue
		}
		delete(kconfigBindingCopy.Spec.KconfigEnvsMap, envKey)
		if !reflect.DeepEqual(kconfigBindingCopy.Spec, kconfigBinding.Spec) {
			_, err := c.kcclient.KconfigcontrollerV1alpha1().KconfigBindings(kconfig.Namespace).Update(kconfigBindingCopy)
			if err != nil {
				runtime.HandleError(fmt.Errorf("Error removing KconfigEnvs from KconfigBinding: %+v", err.Error()))
				continue
			}
		}
	}
}

func (c *Controller) deleteHandler(obj interface{}) {
	kc, ok := obj.(*v1alpha1.Kconfig)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			runtime.HandleError(fmt.Errorf("Couldn't get object from tombstone %#v", obj))
			return
		}
		kc, ok = tombstone.Obj.(*v1alpha1.Kconfig)
		if !ok {
			runtime.HandleError(fmt.Errorf("Tombstone contained object that is not a Kconfig %#v", obj))
			return
		}
	}
	klog.Infof("Deleting kconfig %s", kc.Name)
	c.removeKconfigEnvsFromKconfigBindings(kc)
}

func extractExternalResourceActions(origEnvConfigs []v1alpha1.EnvConfig) ([]v1alpha1.EnvConfig, []v1alpha1.ExternalResourceAction) {
	extResActions := make([]v1alpha1.ExternalResourceAction, 0)
	updatedEnvConfigs := make([]v1alpha1.EnvConfig, 0)
	for _, envConfig := range origEnvConfigs {
		switch envConfig.Type {
		case "", "Value":
			if err := validateNoActionType(envConfig.Action); err != nil {
				runtime.HandleError(fmt.Errorf("Failed to extract action for config with key %s: %s", envConfig.Key, err.Error()))
				continue
			}
			if err := validateNoExternalRef(envConfig); err != nil {
				runtime.HandleError(fmt.Errorf("Failed to extract action for config with key %s: %s", envConfig.Key, err.Error()))
				continue
			}
			updatedEnvConfig := envConfig.DeepCopy()
			updatedEnvConfig.Type = "Value"
			updatedEnvConfigs = append(updatedEnvConfigs, *updatedEnvConfig)
		case "ConfigMap":
			if err := extractConfigMapAction(envConfig, &updatedEnvConfigs, &extResActions); err != nil {
				runtime.HandleError(fmt.Errorf("Failed to extract action for config with key %s: %s", envConfig.Key, err.Error()))
				continue
			}
		case "Secret":
			if err := extractSecretAction(envConfig, &updatedEnvConfigs, &extResActions); err != nil {
				runtime.HandleError(fmt.Errorf("Failed to extract action for config with key %s: %s", envConfig.Key, err.Error()))
				continue
			}
		default:
			runtime.HandleError(fmt.Errorf("Invalid EnvConfig type %s", envConfig.Type))
		}
	}
	return updatedEnvConfigs, extResActions
}

func extractConfigMapAction(origEnvConfig v1alpha1.EnvConfig, updatedEnvConfigs *[]v1alpha1.EnvConfig, actions *[]v1alpha1.ExternalResourceAction) error {
	switch {
	case origEnvConfig.Action == nil:
		if err := validateConfigMapKeyRef(origEnvConfig.ConfigMapKeyRef); err != nil {
			return err
		}
		*updatedEnvConfigs = append(*updatedEnvConfigs, origEnvConfig)
	case origEnvConfig.Action.ActionType == "Add":
		if err := validateAddActionType(origEnvConfig.Action); err != nil {
			return err
		}
		var keyRef string
		var err error
		if keyRef, err = util.GetNewKeyReference(origEnvConfig.Key); err != nil {
			return err
		}
		updatedAction := *origEnvConfig.Action.DeepCopy()
		updatedAction.ResourceType = "ConfigMap"
		updatedAction.ReferenceKey = keyRef
		updatedAction.Data = origEnvConfig.Value
		*actions = append(*actions, updatedAction)

		optional := true
		envConfig := v1alpha1.EnvConfig{
			Key:  origEnvConfig.Key,
			Type: "ConfigMap",
			ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: origEnvConfig.Action.ResourceName,
				},
				Key:      keyRef,
				Optional: &optional,
			},
		}
		*updatedEnvConfigs = append(*updatedEnvConfigs, envConfig)
	case origEnvConfig.Action.ActionType == "Update":
		if err := validateUpdateActionType(origEnvConfig.Action); err != nil {
			return err
		}
		if err := validateConfigMapKeyRef(origEnvConfig.ConfigMapKeyRef); err != nil {
			return err
		}
		updatedAction := *origEnvConfig.Action.DeepCopy()
		updatedAction.ResourceType = "ConfigMap"
		// We ignore any ResourceName or ReferenceKey in original action
		// as the configMapKeyRef contains the true name and key
		updatedAction.ResourceName = origEnvConfig.ConfigMapKeyRef.Name
		updatedAction.ReferenceKey = origEnvConfig.ConfigMapKeyRef.Key
		updatedAction.Data = origEnvConfig.Value
		*actions = append(*actions, updatedAction)

		optional := true
		envConfig := v1alpha1.EnvConfig{
			Key:  origEnvConfig.Key,
			Type: "ConfigMap",
			ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: origEnvConfig.Action.ResourceName,
				},
				Key:      origEnvConfig.ConfigMapKeyRef.Key,
				Optional: &optional,
			},
		}
		*updatedEnvConfigs = append(*updatedEnvConfigs, envConfig)
	case origEnvConfig.Action.ActionType == "Delete":
		if err := validateDeleteActionType(origEnvConfig.Action); err != nil {
			return err
		}
		if err := validateConfigMapKeyRef(origEnvConfig.ConfigMapKeyRef); err != nil {
			return err
		}
		updatedAction := *origEnvConfig.Action.DeepCopy()
		updatedAction.ResourceType = "ConfigMap"
		// We ignore any ResourceName or ReferenceKey in original action
		// as the configMapKeyRef contains the true name and key
		updatedAction.ResourceName = origEnvConfig.ConfigMapKeyRef.Name
		updatedAction.ReferenceKey = origEnvConfig.ConfigMapKeyRef.Key
		*actions = append(*actions, updatedAction)
	default:
		return fmt.Errorf("Invalid Action Type %s", origEnvConfig.Action.ActionType)
	}
	return nil
}

func extractSecretAction(origEnvConfig v1alpha1.EnvConfig, updatedEnvConfigs *[]v1alpha1.EnvConfig, actions *[]v1alpha1.ExternalResourceAction) error {
	switch {
	case origEnvConfig.Action == nil:
		if err := validateSecretKeyRef(origEnvConfig.SecretKeyRef); err != nil {
			return err
		}
		*updatedEnvConfigs = append(*updatedEnvConfigs, origEnvConfig)
	case origEnvConfig.Action.ActionType == "Add":
		if err := validateAddActionType(origEnvConfig.Action); err != nil {
			return err
		}
		var keyRef string
		var err error
		if keyRef, err = util.GetNewKeyReference(origEnvConfig.Key); err != nil {
			return err
		}
		updatedAction := *origEnvConfig.Action.DeepCopy()
		updatedAction.ResourceType = "Secret"
		updatedAction.ReferenceKey = keyRef
		updatedAction.Data = origEnvConfig.Value
		*actions = append(*actions, updatedAction)

		optional := true
		envConfig := v1alpha1.EnvConfig{
			Key:  origEnvConfig.Key,
			Type: "Secret",
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: origEnvConfig.Action.ResourceName,
				},
				Key:      keyRef,
				Optional: &optional,
			},
		}
		*updatedEnvConfigs = append(*updatedEnvConfigs, envConfig)
	case origEnvConfig.Action.ActionType == "Update":
		if err := validateUpdateActionType(origEnvConfig.Action); err != nil {
			return err
		}
		if err := validateSecretKeyRef(origEnvConfig.SecretKeyRef); err != nil {
			return err
		}
		updatedAction := *origEnvConfig.Action.DeepCopy()
		updatedAction.ResourceType = "Secret"
		// We ignore any ResourceName or ReferenceKey in original action
		// as the configMapKeyRef contains the true name and key
		updatedAction.ResourceName = origEnvConfig.SecretKeyRef.Name
		updatedAction.ReferenceKey = origEnvConfig.SecretKeyRef.Key
		updatedAction.Data = origEnvConfig.Value
		*actions = append(*actions, updatedAction)

		optional := true
		envConfig := v1alpha1.EnvConfig{
			Key:  origEnvConfig.Key,
			Type: "Secret",
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: origEnvConfig.Action.ResourceName,
				},
				Key:      origEnvConfig.SecretKeyRef.Key,
				Optional: &optional,
			},
		}
		*updatedEnvConfigs = append(*updatedEnvConfigs, envConfig)
	case origEnvConfig.Action.ActionType == "Delete":
		if err := validateDeleteActionType(origEnvConfig.Action); err != nil {
			return err
		}
		if err := validateSecretKeyRef(origEnvConfig.SecretKeyRef); err != nil {
			return err
		}
		updatedAction := *origEnvConfig.Action.DeepCopy()
		updatedAction.ResourceType = "Secret"
		// We ignore any ResourceName or ReferenceKey in original action
		// as the configMapKeyRef contains the true name and key
		updatedAction.ResourceName = origEnvConfig.SecretKeyRef.Name
		updatedAction.ReferenceKey = origEnvConfig.SecretKeyRef.Key
		*actions = append(*actions, updatedAction)
	default:
		return fmt.Errorf("Invalid Action Type %s", origEnvConfig.Action.ActionType)
	}
	return nil
}

func validateNoActionType(action *v1alpha1.ExternalResourceAction) error {
	if action != nil {
		return fmt.Errorf("Expected no action in EnvConfig")
	}
	return nil
}

func validateAddActionType(action *v1alpha1.ExternalResourceAction) error {
	if len(action.ResourceName) == 0 {
		return fmt.Errorf("Add Action requires ResourceName")
	}
	return nil
}

func validateUpdateActionType(action *v1alpha1.ExternalResourceAction) error {
	// For now, nothing is initially required in the Action except the
	// actionType which should be check before calling this function.
	// ResourceName, Type, and ReferenceKey can all be inferred from the
	// existing ConfigMapKeyRef/SecretKeyRef
	return nil
}

func validateDeleteActionType(action *v1alpha1.ExternalResourceAction) error {
	// For now, nothing is initially required in the Action except the
	// actionType which should be check before calling this function.
	// ResourceName, Type, and ReferenceKey can all be inferred from the
	// existing ConfigMapKeyRef/SecretKeyRef
	return nil
}

func validateNoExternalRef(envConfig v1alpha1.EnvConfig) error {
	if envConfig.ConfigMapKeyRef != nil {
		return fmt.Errorf("Expected no ConfigMapKeyRef")
	}
	if envConfig.SecretKeyRef != nil {
		return fmt.Errorf("Expected no SecretKeyRef")
	}
	return nil
}

func validateConfigMapKeyRef(configMapKeyRef *corev1.ConfigMapKeySelector) error {
	if configMapKeyRef == nil {
		return fmt.Errorf("Expected ConfigMapKeyRef to not be nil")
	}
	if len(configMapKeyRef.Key) == 0 {
		return fmt.Errorf("Valid ConfigMapKeyRef must contain key")
	}
	if len(configMapKeyRef.LocalObjectReference.Name) == 0 {
		return fmt.Errorf("Valid ConfigMapKeyRef must contain name")
	}
	return nil
}

func validateSecretKeyRef(secretKeyRef *corev1.SecretKeySelector) error {
	if secretKeyRef == nil {
		return fmt.Errorf("Expected secretKeyRef to not be nil")
	}
	if len(secretKeyRef.Key) == 0 {
		return fmt.Errorf("Valid secretKeyRef must contain key")
	}
	if len(secretKeyRef.LocalObjectReference.Name) == 0 {
		return fmt.Errorf("Valid secretKeyRef must contain name")
	}
	return nil
}

// TODO: How should errors here be handled
func (c *Controller) executeExternalResourceActions(actions []v1alpha1.ExternalResourceAction, namespace string) error {
	for _, action := range actions {
		switch action.ResourceType {
		case "Secret":
			err := c.executeSecretAction(action, namespace)
			if err != nil {
				return err
			}
		case "ConfigMap":
			err := c.executeConfigMapAction(action, namespace)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Controller) executeSecretAction(action v1alpha1.ExternalResourceAction, namespace string) error {
	secret, err := c.secretlister.Secrets(namespace).Get(action.ResourceName)
	if err != nil {
		if errors.IsNotFound(err) {
			secret, err = c.createSecret(namespace, action.ResourceName)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	secretCopy := secret.DeepCopy()
	if action.ActionType != "Delete" {
		dataBytes := []byte(action.Data)
		encodedLen := base64.StdEncoding.EncodedLen(len(dataBytes))
		encoded := make([]byte, encodedLen)
		base64.StdEncoding.Encode(encoded, dataBytes)
		if secretCopy.Data == nil {
			secretCopy.Data = make(map[string][]byte)
		}
		secretCopy.Data[action.ReferenceKey] = encoded
	}
	// For now deleted configs will remain in the actual secret
	// but not referenced in the Kconfig. A process will be implemented
	// to clean such unused keys to allow a grace period for retrieving
	// or rolling back configs that were deleted.
	if !reflect.DeepEqual(secretCopy, secret) {
		_, err = c.stdclient.CoreV1().Secrets(namespace).Update(secretCopy)
	}
	return err
}

func (c *Controller) executeConfigMapAction(action v1alpha1.ExternalResourceAction, namespace string) error {
	configmap, err := c.configmaplister.ConfigMaps(namespace).Get(action.ResourceName)
	if err != nil {
		if errors.IsNotFound(err) {
			configmap, err = c.createConfigMap(namespace, action.ResourceName)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	configmapCopy := configmap.DeepCopy()
	if action.ActionType != "Delete" {
		if configmapCopy.Data == nil {
			configmapCopy.Data = make(map[string]string)
		}
		configmapCopy.Data[action.ReferenceKey] = action.Data
	}
	// For now deleted configs will remain in the actual configMap
	// but not referenced in the Kconfig. A process will be implemented
	// to clean such unused keys to allow a grace period for retrieving
	// or rolling back configs that were deleted.
	if !reflect.DeepEqual(configmapCopy, configmap) {
		_, err = c.stdclient.CoreV1().ConfigMaps(namespace).Update(configmapCopy)
	}
	return err
}

func (c *Controller) createSecret(namespace, name string) (*corev1.Secret, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Data:       map[string][]byte{},
		StringData: map[string]string{},
	}
	return c.stdclient.CoreV1().Secrets(namespace).Create(secret)
}

func (c *Controller) createConfigMap(namespace, name string) (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Data: map[string]string{},
	}
	return c.stdclient.CoreV1().ConfigMaps(namespace).Create(configMap)
}

// Assumes the EnvConfig Type is valid and all external references
// have been finalized with value fields removed where applicable
func buildKconfigEnv(level int, envConfigs []v1alpha1.EnvConfig) v1alpha1.KconfigEnvs {
	envs := make([]corev1.EnvVar, 0)
	for _, envConfig := range envConfigs {
		switch envConfig.Type {
		// Canonical key/value config case. Return existing envConfig and generic key/val env var
		case "", "Value":
			envs = append(envs, corev1.EnvVar{Name: envConfig.Key, Value: envConfig.Value})
		// Configmap Value
		case "ConfigMap":
			envs = append(envs, corev1.EnvVar{Name: envConfig.Key, ValueFrom: &corev1.EnvVarSource{ConfigMapKeyRef: envConfig.ConfigMapKeyRef}})
		// Secret Value
		case "Secret":
			envs = append(envs, corev1.EnvVar{Name: envConfig.Key, ValueFrom: &corev1.EnvVarSource{SecretKeyRef: envConfig.SecretKeyRef}})
			// Assumes type is valid so no default case
		}
	}
	return v1alpha1.KconfigEnvs{
		Level: level,
		Envs:  envs,
	}
}

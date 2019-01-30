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

// ExternalResourceConfig configs with reference to external resource (configmap/secret)
type ExternalResourceConfig struct {
	ResourceType string
	ResourceName string
	ResourceKey  string
	Value        string
}

func (c *Controller) processKconfig(kconfig *v1alpha1.Kconfig) error {
	updatedEnvConfigs, extConfigs := extractExternalResourceConfigs(kconfig.Spec.EnvConfigs)
	if err := c.processExternalResourceConfigs(extConfigs, kconfig.Namespace); err != nil {
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

func extractExternalResourceConfigs(origEnvConfigs []v1alpha1.EnvConfig) ([]v1alpha1.EnvConfig, []ExternalResourceConfig) {
	extConfigs := make([]ExternalResourceConfig, 0)
	updatedEnvConfigs := make([]v1alpha1.EnvConfig, 0)
	for _, envConfig := range origEnvConfigs {
		switch envConfig.Type {
		case "", "Value":
			if err := validateValueEnvConfig(envConfig); err != nil {
				klog.Warningf("Invalid EnvConfig: %s", err.Error())
				continue
			}
			updatedEnvConfig := envConfig.DeepCopy()
			updatedEnvConfig.Type = "Value"
			updatedEnvConfigs = append(updatedEnvConfigs, *updatedEnvConfig)
		case "ConfigMap":
			if err := validateConfigMapEnvConfig(envConfig); err != nil {
				klog.Warningf("Invalid EnvConfig: %s", err.Error())
				continue
			}
			var updatedEnvConfig *v1alpha1.EnvConfig
			if envConfig.Value != nil {
				optional := true
				refName := getConfigMapEnvConfigResourceName(envConfig)
				refKey, err := getConfigMapEnvConfigResourceKey(envConfig)
				if err != nil {
					klog.Warningf("Error processing EnvConfig: %s", err.Error())
				}
				extConfig := ExternalResourceConfig{
					ResourceType: "ConfigMap",
					ResourceName: refName,
					ResourceKey:  refKey,
					Value:        *envConfig.Value,
				}
				extConfigs = append(extConfigs, extConfig)
				updatedEnvConfig = &v1alpha1.EnvConfig{
					Type: "ConfigMap",
					Key:  envConfig.Key,
					ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: refName,
						},
						Key:      refKey,
						Optional: &optional,
					},
				}
			} else {
				updatedEnvConfig = envConfig.DeepCopy()
			}
			updatedEnvConfigs = append(updatedEnvConfigs, *updatedEnvConfig)
		case "Secret":
			if err := validateSecretEnvConfig(envConfig); err != nil {
				klog.Warningf("Invalid EnvConfig: %s", err.Error())
				continue
			}
			var updatedEnvConfig *v1alpha1.EnvConfig
			if envConfig.Value != nil {
				optional := true
				refName := getSecretEnvConfigResourceName(envConfig)
				refKey, err := getSecretEnvConfigResourceKey(envConfig)
				if err != nil {
					klog.Warningf("Error processing EnvConfig: %s", err.Error())
				}
				extConfig := ExternalResourceConfig{
					ResourceType: "Secret",
					ResourceName: refName,
					ResourceKey:  refKey,
					Value:        *envConfig.Value,
				}
				extConfigs = append(extConfigs, extConfig)
				updatedEnvConfig = &v1alpha1.EnvConfig{
					Type: "Secret",
					Key:  envConfig.Key,
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: refName,
						},
						Key:      refKey,
						Optional: &optional,
					},
				}
			} else {
				updatedEnvConfig = envConfig.DeepCopy()
			}
			updatedEnvConfigs = append(updatedEnvConfigs, *updatedEnvConfig)
		}
	}
	return updatedEnvConfigs, extConfigs
}

func getConfigMapEnvConfigResourceName(envConfig v1alpha1.EnvConfig) string {
	if envConfig.ConfigMapKeyRef != nil {
		return envConfig.ConfigMapKeyRef.LocalObjectReference.Name
	}
	return *envConfig.RefName
}

func getConfigMapEnvConfigResourceKey(envConfig v1alpha1.EnvConfig) (string, error) {
	if envConfig.ConfigMapKeyRef != nil {
		return envConfig.ConfigMapKeyRef.Key, nil
	}
	return util.GetNewKeyReference(envConfig.Key)
}

func getSecretEnvConfigResourceName(envConfig v1alpha1.EnvConfig) string {
	if envConfig.SecretKeyRef != nil {
		return envConfig.SecretKeyRef.LocalObjectReference.Name
	}
	return *envConfig.RefName
}

func getSecretEnvConfigResourceKey(envConfig v1alpha1.EnvConfig) (string, error) {
	if envConfig.SecretKeyRef != nil {
		return envConfig.SecretKeyRef.Key, nil
	}
	return util.GetNewKeyReference(envConfig.Key)
}

func validateValueEnvConfig(envConfig v1alpha1.EnvConfig) error {
	if len(envConfig.Key) == 0 {
		return fmt.Errorf("EnvConfig must have Key")
	}
	if envConfig.Value == nil {
		return fmt.Errorf("Value Type EnvConfig must have Value")
	}
	if envConfig.RefName != nil {
		return fmt.Errorf("Value Type EnvConfig should not have RefName")
	}
	if envConfig.RefKey != nil {
		return fmt.Errorf("Value Type EnvConfig should not have RefKey")
	}
	if envConfig.ConfigMapKeyRef != nil {
		return fmt.Errorf("Value Type EnvConfig should not have ConfigMapKeyRef")
	}
	if envConfig.SecretKeyRef != nil {
		return fmt.Errorf("Value Type EnvConfig should not have SecretKeyRef")
	}
	return nil
}

func validateConfigMapEnvConfig(envConfig v1alpha1.EnvConfig) error {
	if len(envConfig.Key) == 0 {
		return fmt.Errorf("EnvConfig must have Key")
	}
	if envConfig.SecretKeyRef != nil {
		return fmt.Errorf("ConfigMap Type EnvConfig should not have SecretKeyRef")
	}
	// For Pre-existing ConfigMap EnvConfig
	if envConfig.ConfigMapKeyRef != nil {
		return validateExistingConfigMapEnvConfig(envConfig)
	}
	// For Non-pre-existing ConfigMap EnvConfig
	if envConfig.RefName == nil {
		return fmt.Errorf("New ConfigMap EnvConfigs should have RefName")
	}
	return nil
}

func validateExistingConfigMapEnvConfig(envConfig v1alpha1.EnvConfig) error {
	if envConfig.RefName != nil {
		return fmt.Errorf("New ConfigMap EnvConfigs should not have RefName")
	}
	if envConfig.RefKey != nil {
		return fmt.Errorf("New ConfigMap EnvConfigs should not have RefKey")
	}
	return nil
}

// begin secret
func validateSecretEnvConfig(envConfig v1alpha1.EnvConfig) error {
	if len(envConfig.Key) == 0 {
		return fmt.Errorf("EnvConfig must have Key")
	}
	if envConfig.ConfigMapKeyRef != nil {
		return fmt.Errorf("Secret Type EnvConfig should not have ConfigMapKeyRef")
	}
	// For Pre-existing Secret EnvConfig
	if envConfig.SecretKeyRef != nil {
		return validateExistingSecretEnvConfig(envConfig)
	}
	// For Non-pre-existing Secret EnvConfig
	if envConfig.RefName == nil {
		return fmt.Errorf("New Secret EnvConfigs should have RefName")
	}
	return nil
}

func validateExistingSecretEnvConfig(envConfig v1alpha1.EnvConfig) error {
	if envConfig.RefName != nil {
		return fmt.Errorf("New Secret EnvConfigs should not have RefName")
	}
	if envConfig.RefKey != nil {
		return fmt.Errorf("New Secret EnvConfigs should not have RefKey")
	}
	return nil
}

func (c *Controller) processExternalResourceConfigs(extConfigs []ExternalResourceConfig, namespace string) error {
	for _, extConfig := range extConfigs {
		switch extConfig.ResourceType {
		case "ConfigMap":
			if err := c.processConfigMapConfig(extConfig, namespace); err != nil {
				return err
			}
		case "Secret":
			if err := c.processSecretConfig(extConfig, namespace); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Controller) processConfigMapConfig(extConfig ExternalResourceConfig, namespace string) error {
	configmap, err := c.configmaplister.ConfigMaps(namespace).Get(extConfig.ResourceName)
	if err != nil {
		if errors.IsNotFound(err) {
			configmap, err = c.createConfigMap(namespace, extConfig.ResourceName)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	configmapCopy := configmap.DeepCopy()
	if configmapCopy.Data == nil {
		configmapCopy.Data = make(map[string]string)
	}
	configmapCopy.Data[extConfig.ResourceKey] = extConfig.Value
	if !reflect.DeepEqual(configmapCopy, configmap) {
		_, err = c.stdclient.CoreV1().ConfigMaps(namespace).Update(configmapCopy)
	}
	return err
}

func (c *Controller) processSecretConfig(extConfig ExternalResourceConfig, namespace string) error {
	secret, err := c.secretlister.Secrets(namespace).Get(extConfig.ResourceName)
	if err != nil {
		if errors.IsNotFound(err) {
			secret, err = c.createSecret(namespace, extConfig.ResourceName)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	secretCopy := secret.DeepCopy()
	dataBytes := []byte(extConfig.Value)
	encodedLen := base64.StdEncoding.EncodedLen(len(dataBytes))
	encoded := make([]byte, encodedLen)
	base64.StdEncoding.Encode(encoded, dataBytes)
	if secretCopy.Data == nil {
		secretCopy.Data = make(map[string][]byte)
	}
	secretCopy.Data[extConfig.ResourceKey] = encoded
	if !reflect.DeepEqual(secretCopy, secret) {
		_, err = c.stdclient.CoreV1().Secrets(namespace).Update(secretCopy)
	}
	return err
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

// Assumes the EnvConfig Type is valid and all external references
// have been finalized with value fields removed where applicable
func buildKconfigEnv(level int, envConfigs []v1alpha1.EnvConfig) v1alpha1.KconfigEnvs {
	envs := make([]corev1.EnvVar, 0)
	for _, envConfig := range envConfigs {
		switch envConfig.Type {
		// Canonical key/value config case. Return existing envConfig and generic key/val env var
		case "", "Value":
			envs = append(envs, corev1.EnvVar{Name: envConfig.Key, Value: *envConfig.Value})
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

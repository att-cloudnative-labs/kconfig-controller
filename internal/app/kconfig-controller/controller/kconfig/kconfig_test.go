package kconfig

import (
	"reflect"
	"testing"

	"github.com/gbraxton/kconfig/pkg/apis/kconfigcontroller/v1alpha1"
	kcfake "github.com/gbraxton/kconfig/pkg/client/clientset/versioned/fake"
	kcinformers "github.com/gbraxton/kconfig/pkg/client/informers/externalversions"
	testutil "github.com/gbraxton/kconfig/test/util"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	stdinformers "k8s.io/client-go/informers"
	stdfake "k8s.io/client-go/kubernetes/fake"
	core "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

var (
	alwaysReady = func() bool { return true }
)

type fixture struct {
	t *testing.T

	stdclient *stdfake.Clientset
	kcclient  *kcfake.Clientset

	// Objects to put in the store
	configmapLister []*v1.ConfigMap
	secretLister    []*v1.Secret
	kconfigLister   []*v1alpha1.Kconfig
	kbindingLister  []*v1alpha1.KconfigBinding

	// Actions expected to happen on the client. Objects from here are also
	// preloaded into NewSimpleFake.
	stdactions []core.Action
	kcactions  []core.Action
	stdobjects []runtime.Object
	kcobjects  []runtime.Object
}

func newFixture(t *testing.T) *fixture {
	f := &fixture{}
	f.t = t
	f.stdobjects = []runtime.Object{}
	f.kcobjects = []runtime.Object{}
	return f
}

func (f *fixture) newController() (*Controller, kcinformers.SharedInformerFactory, stdinformers.SharedInformerFactory, error) {
	f.stdclient = stdfake.NewSimpleClientset(f.stdobjects...)
	f.kcclient = kcfake.NewSimpleClientset(f.kcobjects...)

	stdinformers := stdinformers.NewSharedInformerFactory(f.stdclient, 0)
	kcinformers := kcinformers.NewSharedInformerFactory(f.kcclient, 0)

	c := NewController(f.stdclient, f.kcclient, stdinformers.Core().V1().ConfigMaps(), stdinformers.Core().V1().Secrets(), kcinformers.Kconfigcontroller().V1alpha1().Kconfigs(), kcinformers.Kconfigcontroller().V1alpha1().KconfigBindings())
	c.recorder = &record.FakeRecorder{}
	c.configmapsSynced = alwaysReady
	c.secretsSynced = alwaysReady
	c.kconfigsSynced = alwaysReady
	c.kconfigBindingsSynced = alwaysReady
	for _, cm := range f.configmapLister {
		stdinformers.Core().V1().ConfigMaps().Informer().GetIndexer().Add(cm)
	}
	for _, sec := range f.secretLister {
		stdinformers.Core().V1().Secrets().Informer().GetIndexer().Add(sec)
	}
	for _, kc := range f.kconfigLister {
		kcinformers.Kconfigcontroller().V1alpha1().Kconfigs().Informer().GetIndexer().Add(kc)
	}
	for _, kcb := range f.kbindingLister {
		kcinformers.Kconfigcontroller().V1alpha1().KconfigBindings().Informer().GetIndexer().Add(kcb)
	}
	return c, kcinformers, stdinformers, nil
}

func (f *fixture) runExpectError(kconfigName string, startInformers bool) {
	f.runSync(kconfigName, startInformers, true)
}

func (f *fixture) run(kconfigName string) {
	f.runSync(kconfigName, true, false)
}

func (f *fixture) runSync(kconfigName string, startInformers bool, expectError bool) {
	c, kcinformers, stdinformers, err := f.newController()
	if err != nil {
		f.t.Fatalf("error creating Kconfig controller: %v", err)
	}
	if startInformers {
		stopCh := make(chan struct{})
		defer close(stopCh)
		kcinformers.Start(stopCh)
		stdinformers.Start(stopCh)
	}

	err = c.syncHandler(kconfigName)
	if !expectError && err != nil {
		f.t.Errorf("error syncing kconfig: %v", err)
	} else if expectError && err == nil {
		f.t.Error("expected error syncing kconfig, got nil")
	}

	f.checkActions()
}

// runDelete calls deleteKconfig instead of syncHandler
func (f *fixture) runDelete(obj interface{}) {
	startInformers := true

	c, kcinformers, stdinformers, err := f.newController()
	if err != nil {
		f.t.Fatalf("error creating Deployment controller: %v", err)
	}
	if startInformers {
		stopCh := make(chan struct{})
		defer close(stopCh)
		kcinformers.Start(stopCh)
		stdinformers.Start(stopCh)
	}

	c.deleteHandler(obj)

	f.checkActions()
}

func (f *fixture) checkActions() {
	stdactions := filterStdInformerActions(f.stdclient.Actions())
	for i, stdaction := range stdactions {
		if len(f.stdactions) < i+1 {
			f.t.Errorf("%d unexpected actions: %+v", len(stdactions)-len(f.stdactions), stdactions[i:])
			break
		}

		expectedAction := f.stdactions[i]
		if !(expectedAction.Matches(stdaction.GetVerb(), stdaction.GetResource().Resource) && stdaction.GetSubresource() == expectedAction.GetSubresource()) {
			f.t.Errorf("Expected\n\t%#v\ngot\n\t%#v", expectedAction, stdaction)
			continue
		}
		if !f.actionObjectsMatch(expectedAction, stdaction) {
			continue
		}
	}
	if len(f.stdactions) > len(stdactions) {
		f.t.Errorf("%d additional expected actions:%+v", len(f.stdactions)-len(stdactions), f.stdactions[len(stdactions):])
	}

	kcactions := filterKcInformerActions(f.kcclient.Actions())
	for i, kcaction := range kcactions {
		if len(f.kcactions) < i+1 {
			f.t.Errorf("%d unexpected actions: %+v", len(kcactions)-len(f.kcactions), kcactions[i:])
			break
		}

		expectedAction := f.kcactions[i]
		if !(expectedAction.Matches(kcaction.GetVerb(), kcaction.GetResource().Resource) && kcaction.GetSubresource() == expectedAction.GetSubresource()) {
			f.t.Errorf("Expected\n\t%#v\ngot\n\t%#v", expectedAction, kcaction)
			continue
		}
		if !f.actionObjectsMatch(expectedAction, kcaction) {
			continue
		}
	}
	if len(f.kcactions) > len(kcactions) {
		f.t.Errorf("%d additional expected actions:%+v", len(f.kcactions)-len(kcactions), f.kcactions[len(kcactions):])
	}
}

// actionObjectsMatch Assumes expectedAction and action have already had their
// verbs matched. Always returns true if params aren't Create or Update actions.
func (f *fixture) actionObjectsMatch(expectedAction, action core.Action) bool {
	if expectedCreateAction, ok := expectedAction.(core.CreateAction); ok {
		createAction, _ := action.(core.CreateAction)
		if !reflect.DeepEqual(expectedCreateAction.GetObject(), createAction.GetObject()) {
			f.t.Errorf("Expected\n\t%+v\ngot\n\t%+v", expectedCreateAction.GetObject(), createAction.GetObject())
			return false
		}
		return true
	}
	if expectedUpdateAction, ok := expectedAction.(core.UpdateAction); ok {
		updateAction, _ := action.(core.UpdateAction)
		if !reflect.DeepEqual(expectedUpdateAction.GetObject(), updateAction.GetObject()) {
			f.t.Errorf("Expected\n\t%+v\ngot\n\t%+v", expectedUpdateAction.GetObject(), updateAction.GetObject())
			return false
		}
		return true
	}
	return true
}

func filterStdInformerActions(actions []core.Action) []core.Action {
	ret := []core.Action{}
	for _, action := range actions {
		if len(action.GetNamespace()) == 0 &&
			(action.Matches("list", "configmaps") ||
				action.Matches("list", "secrets") ||
				action.Matches("watch", "configmaps") ||
				action.Matches("watch", "secrets")) {
			continue
		}
		ret = append(ret, action)
	}

	return ret
}

func filterKcInformerActions(actions []core.Action) []core.Action {
	ret := []core.Action{}
	for _, action := range actions {
		if len(action.GetNamespace()) == 0 &&
			(action.Matches("list", "kconfigs") ||
				action.Matches("list", "kconfigbindings") ||
				action.Matches("watch", "kconfigs") ||
				action.Matches("watch", "kconfigbindings")) {
			continue
		}
		ret = append(ret, action)
	}
	return ret
}

func (f *fixture) expectUpdateKconfigAction(k *v1alpha1.Kconfig) {
	resource := schema.GroupVersionResource{
		Group:    v1alpha1.SchemeGroupVersion.Group,
		Version:  v1alpha1.SchemeGroupVersion.Version,
		Resource: "kconfigs",
	}
	action := core.NewUpdateAction(resource, k.Namespace, k)
	f.kcactions = append(f.kcactions, action)
}

func (f *fixture) expectUpdateKconfigBindingAction(k *v1alpha1.KconfigBinding) {
	resource := schema.GroupVersionResource{
		Group:    v1alpha1.SchemeGroupVersion.Group,
		Version:  v1alpha1.SchemeGroupVersion.Version,
		Resource: "kconfigbindings",
	}
	action := core.NewUpdateAction(resource, k.Namespace, k)
	f.kcactions = append(f.kcactions, action)
}

func (f *fixture) expectCreateConfigMapAction(c *v1.ConfigMap) {
	resource := schema.GroupVersionResource{
		Group:    v1.SchemeGroupVersion.Group,
		Version:  v1.SchemeGroupVersion.Version,
		Resource: "configmaps",
	}
	action := core.NewCreateAction(resource, c.Namespace, c)
	f.stdactions = append(f.stdactions, action)
}

func (f *fixture) expectUpdateConfigMapAction(c *v1.ConfigMap) {
	resource := schema.GroupVersionResource{
		Group:    v1.SchemeGroupVersion.Group,
		Version:  v1.SchemeGroupVersion.Version,
		Resource: "configmaps",
	}
	action := core.NewUpdateAction(resource, c.Namespace, c)
	f.stdactions = append(f.stdactions, action)
}

func (f *fixture) expectCreateSecretAction(s *v1.Secret) {
	resource := schema.GroupVersionResource{
		Group:    v1.SchemeGroupVersion.Group,
		Version:  v1.SchemeGroupVersion.Version,
		Resource: "secrets",
	}
	action := core.NewCreateAction(resource, s.Namespace, s)
	f.stdactions = append(f.stdactions, action)
}

func (f *fixture) expectUpdateSecretAction(s *v1.Secret) {
	resource := schema.GroupVersionResource{
		Group:    v1.SchemeGroupVersion.Group,
		Version:  v1.SchemeGroupVersion.Version,
		Resource: "secrets",
	}
	action := core.NewUpdateAction(resource, s.Namespace, s)
	f.stdactions = append(f.stdactions, action)
}

func TestValueKconfig(t *testing.T) {
	f := newFixture(t)

	kc := testutil.ValueKconfig()
	kcb := testutil.KconfigBinding()
	expectedkcbupdate := testutil.ValueKconfigBinding()

	f.kconfigLister = append(f.kconfigLister, &kc)
	f.kbindingLister = append(f.kbindingLister, &kcb)
	f.kcobjects = append(f.kcobjects, &kc)
	f.kcobjects = append(f.kcobjects, &kcb)

	f.expectUpdateKconfigBindingAction(&expectedkcbupdate)

	key, _ := cache.MetaNamespaceKeyFunc(&kc.ObjectMeta)
	f.run(key)
}

func TestConfigmapKconfig(t *testing.T) {
	f := newFixture(t)

	kc := testutil.ConfigMapKconfig()
	kcb := testutil.KconfigBinding()
	expectedkcbupdate := testutil.ConfigMapKconfigBinding(0)

	f.kconfigLister = append(f.kconfigLister, &kc)
	f.kbindingLister = append(f.kbindingLister, &kcb)
	f.kcobjects = append(f.kcobjects, &kc)
	f.kcobjects = append(f.kcobjects, &kcb)

	f.expectUpdateKconfigBindingAction(&expectedkcbupdate)

	key, _ := cache.MetaNamespaceKeyFunc(&kc.ObjectMeta)
	f.run(key)
}

func TestSecretKconfig(t *testing.T) {
	f := newFixture(t)

	kc := testutil.SecretKconfig()
	kcb := testutil.KconfigBinding()
	expectedkcbupdate := testutil.SecretKconfigBinding(0)

	f.kconfigLister = append(f.kconfigLister, &kc)
	f.kbindingLister = append(f.kbindingLister, &kcb)
	f.kcobjects = append(f.kcobjects, &kc)
	f.kcobjects = append(f.kcobjects, &kcb)

	f.expectUpdateKconfigBindingAction(&expectedkcbupdate)

	key, _ := cache.MetaNamespaceKeyFunc(&kc.ObjectMeta)
	f.run(key)
}
func TestAddConfigMapKconfig(t *testing.T) {
	f := newFixture(t)

	kc := testutil.AddConfigMapKconfig()
	kcb := testutil.KconfigBinding()
	expectedkcupdate := testutil.ConfigMapKconfig()
	expectedkcupdate.Spec.EnvRefsVersion++
	expectedkcbupdate := testutil.ConfigMapKconfigBinding(1)
	expectedcmcreate := testutil.ConfigMap()
	expectedcmupdate := testutil.ConfigMapWithData()

	f.kconfigLister = append(f.kconfigLister, &kc)
	f.kbindingLister = append(f.kbindingLister, &kcb)
	f.kcobjects = append(f.kcobjects, &kc)
	f.kcobjects = append(f.kcobjects, &kcb)

	f.expectUpdateKconfigAction(&expectedkcupdate)
	f.expectCreateConfigMapAction(&expectedcmcreate)
	f.expectUpdateConfigMapAction(&expectedcmupdate)
	f.expectUpdateKconfigBindingAction(&expectedkcbupdate)

	key, _ := cache.MetaNamespaceKeyFunc(&kc.ObjectMeta)
	f.run(key)
}

func TestAddConfigMapKconfigWithoutRefName(t *testing.T) {
	f := newFixture(t)

	kc := testutil.AddConfigMapKconfig()
	kc.Spec.EnvConfigs[0].RefName = nil
	kcb := testutil.KconfigBinding()
	expectedkcupdate := testutil.ConfigMapKconfig()
	expectedkcupdate.Spec.EnvRefsVersion++
	expectedkcupdate.Spec.EnvConfigs[0].ConfigMapKeyRef.Name = testutil.DefaultName
	expectedkcbupdate := testutil.ConfigMapKconfigBinding(1)
	expectedkcbupdate.Spec.KconfigEnvsMap[testutil.DefaultKconfigEnvsKey].Envs[0].ValueFrom.ConfigMapKeyRef.Name = testutil.DefaultName
	expectedcmcreate := testutil.ConfigMap()
	expectedcmcreate.ObjectMeta.Name = testutil.DefaultName
	expectedcmupdate := testutil.ConfigMapWithData()
	expectedcmupdate.ObjectMeta.Name = testutil.DefaultName

	f.kconfigLister = append(f.kconfigLister, &kc)
	f.kbindingLister = append(f.kbindingLister, &kcb)
	f.kcobjects = append(f.kcobjects, &kc)
	f.kcobjects = append(f.kcobjects, &kcb)

	f.expectUpdateKconfigAction(&expectedkcupdate)
	f.expectCreateConfigMapAction(&expectedcmcreate)
	f.expectUpdateConfigMapAction(&expectedcmupdate)
	f.expectUpdateKconfigBindingAction(&expectedkcbupdate)

	key, _ := cache.MetaNamespaceKeyFunc(&kc.ObjectMeta)
	f.run(key)
}

func TestAddExistingConfigMapKconfig(t *testing.T) {
	f := newFixture(t)

	kc := testutil.AddConfigMapKconfig()
	kcb := testutil.KconfigBinding()
	cm := testutil.ConfigMap()
	expectedkcupdate := testutil.ConfigMapKconfig()
	expectedkcupdate.Spec.EnvRefsVersion++
	expectedkcbupdate := testutil.ConfigMapKconfigBinding(1)
	expectedcmupdate := testutil.ConfigMapWithData()

	f.kconfigLister = append(f.kconfigLister, &kc)
	f.kbindingLister = append(f.kbindingLister, &kcb)
	f.configmapLister = append(f.configmapLister, &cm)
	f.kcobjects = append(f.kcobjects, &kc)
	f.kcobjects = append(f.kcobjects, &kcb)
	f.stdobjects = append(f.stdobjects, &cm)

	f.expectUpdateKconfigAction(&expectedkcupdate)
	f.expectUpdateConfigMapAction(&expectedcmupdate)
	f.expectUpdateKconfigBindingAction(&expectedkcbupdate)

	key, _ := cache.MetaNamespaceKeyFunc(&kc.ObjectMeta)
	f.run(key)
}

func TestAddSecretKconfig(t *testing.T) {
	f := newFixture(t)

	kc := testutil.AddSecretKconfig()
	kcb := testutil.KconfigBinding()
	expectedkcupdate := testutil.SecretKconfig()
	expectedkcupdate.Spec.EnvRefsVersion++
	expectedkcbupdate := testutil.SecretKconfigBinding(1)
	expectedsecretcreate := testutil.Secret()
	expectedsecretupdate := testutil.SecretWithData()

	f.kconfigLister = append(f.kconfigLister, &kc)
	f.kbindingLister = append(f.kbindingLister, &kcb)
	f.kcobjects = append(f.kcobjects, &kc)
	f.kcobjects = append(f.kcobjects, &kcb)

	f.expectUpdateKconfigAction(&expectedkcupdate)
	f.expectCreateSecretAction(&expectedsecretcreate)
	f.expectUpdateSecretAction(&expectedsecretupdate)
	f.expectUpdateKconfigBindingAction(&expectedkcbupdate)

	key, _ := cache.MetaNamespaceKeyFunc(&kc.ObjectMeta)
	f.run(key)
}

func TestAddSecretKconfigWithoutRefName(t *testing.T) {
	f := newFixture(t)

	kc := testutil.AddSecretKconfig()
	kc.Spec.EnvConfigs[0].RefName = nil
	kcb := testutil.KconfigBinding()
	expectedkcupdate := testutil.SecretKconfig()
	expectedkcupdate.Spec.EnvRefsVersion++
	expectedkcupdate.Spec.EnvConfigs[0].SecretKeyRef.Name = testutil.DefaultName
	expectedkcbupdate := testutil.SecretKconfigBinding(1)
	expectedkcbupdate.Spec.KconfigEnvsMap[testutil.DefaultKconfigEnvsKey].Envs[0].ValueFrom.SecretKeyRef.Name = testutil.DefaultName
	expectedsecretcreate := testutil.Secret()
	expectedsecretcreate.ObjectMeta.Name = testutil.DefaultName
	expectedsecretupdate := testutil.SecretWithData()
	expectedsecretupdate.ObjectMeta.Name = testutil.DefaultName

	f.kconfigLister = append(f.kconfigLister, &kc)
	f.kbindingLister = append(f.kbindingLister, &kcb)
	f.kcobjects = append(f.kcobjects, &kc)
	f.kcobjects = append(f.kcobjects, &kcb)

	f.expectUpdateKconfigAction(&expectedkcupdate)
	f.expectCreateSecretAction(&expectedsecretcreate)
	f.expectUpdateSecretAction(&expectedsecretupdate)
	f.expectUpdateKconfigBindingAction(&expectedkcbupdate)

	key, _ := cache.MetaNamespaceKeyFunc(&kc.ObjectMeta)
	f.run(key)
}

func TestAddExistingSecretKconfig(t *testing.T) {
	f := newFixture(t)

	kc := testutil.AddSecretKconfig()
	kcb := testutil.KconfigBinding()
	secret := testutil.Secret()
	expectedkcupdate := testutil.SecretKconfig()
	expectedkcupdate.Spec.EnvRefsVersion++
	expectedkcbupdate := testutil.SecretKconfigBinding(1)
	expectedsecretupdate := testutil.SecretWithData()

	f.kconfigLister = append(f.kconfigLister, &kc)
	f.kbindingLister = append(f.kbindingLister, &kcb)
	f.secretLister = append(f.secretLister, &secret)
	f.kcobjects = append(f.kcobjects, &kc)
	f.kcobjects = append(f.kcobjects, &kcb)
	f.stdobjects = append(f.stdobjects, &secret)

	f.expectUpdateKconfigAction(&expectedkcupdate)
	f.expectUpdateSecretAction(&expectedsecretupdate)
	f.expectUpdateKconfigBindingAction(&expectedkcbupdate)

	key, _ := cache.MetaNamespaceKeyFunc(&kc.ObjectMeta)
	f.run(key)
}
func TestValueUpdateKconfig(t *testing.T) {
	f := newFixture(t)

	kc := testutil.NewValueKconfig()
	kcb := testutil.ValueKconfigBinding()
	expectedkcbupdate := testutil.NewValueKconfigBinding()

	f.kconfigLister = append(f.kconfigLister, &kc)
	f.kbindingLister = append(f.kbindingLister, &kcb)
	f.kcobjects = append(f.kcobjects, &kc)
	f.kcobjects = append(f.kcobjects, &kcb)

	f.expectUpdateKconfigBindingAction(&expectedkcbupdate)

	key, _ := cache.MetaNamespaceKeyFunc(&kc.ObjectMeta)
	f.run(key)
}

func TestUpdateConfigMapKconfig(t *testing.T) {
	f := newFixture(t)

	kc := testutil.UpdateConfigMapKconfig()
	kcb := testutil.ConfigMapKconfigBinding(0)
	cm := testutil.ConfigMapWithData()
	expectedkcupdate := testutil.ConfigMapKconfig()
	expectedkcupdate.Spec.EnvRefsVersion++
	expectedkcbupdate := testutil.ConfigMapKconfigBinding(1)
	expectedcmupdate := testutil.ConfigMapWithNewData()

	f.kconfigLister = append(f.kconfigLister, &kc)
	f.kbindingLister = append(f.kbindingLister, &kcb)
	f.configmapLister = append(f.configmapLister, &cm)
	f.kcobjects = append(f.kcobjects, &kc)
	f.kcobjects = append(f.kcobjects, &kcb)
	f.stdobjects = append(f.stdobjects, &cm)

	f.expectUpdateKconfigAction(&expectedkcupdate)
	f.expectUpdateConfigMapAction(&expectedcmupdate)
	f.expectUpdateKconfigBindingAction(&expectedkcbupdate)

	key, _ := cache.MetaNamespaceKeyFunc(&kc.ObjectMeta)
	f.run(key)
}

func TestUpdateSecretKconfig(t *testing.T) {
	f := newFixture(t)

	kc := testutil.UpdateSecretKconfig()
	kcb := testutil.SecretKconfigBinding(0)
	secret := testutil.SecretWithData()
	expectedkcupdate := testutil.SecretKconfig()
	expectedkcupdate.Spec.EnvRefsVersion++
	expectedkcbupdate := testutil.SecretKconfigBinding(1)
	expectedsecretupdate := testutil.SecretWithNewData()

	f.kconfigLister = append(f.kconfigLister, &kc)
	f.kbindingLister = append(f.kbindingLister, &kcb)
	f.secretLister = append(f.secretLister, &secret)
	f.kcobjects = append(f.kcobjects, &kc)
	f.kcobjects = append(f.kcobjects, &kcb)
	f.stdobjects = append(f.stdobjects, &secret)

	f.expectUpdateKconfigAction(&expectedkcupdate)
	f.expectUpdateSecretAction(&expectedsecretupdate)
	f.expectUpdateKconfigBindingAction(&expectedkcbupdate)

	key, _ := cache.MetaNamespaceKeyFunc(&kc.ObjectMeta)
	f.run(key)
}

func TestValueDeleteKconfig(t *testing.T) {
	f := newFixture(t)

	kc := testutil.Kconfig()
	kcb := testutil.ValueKconfigBinding()
	expectedkcbupdate := testutil.EmptyKconfigEnvsKconfigBinding()

	f.kconfigLister = append(f.kconfigLister, &kc)
	f.kbindingLister = append(f.kbindingLister, &kcb)
	f.kcobjects = append(f.kcobjects, &kc)
	f.kcobjects = append(f.kcobjects, &kcb)

	f.expectUpdateKconfigBindingAction(&expectedkcbupdate)

	key, _ := cache.MetaNamespaceKeyFunc(&kc.ObjectMeta)
	f.run(key)
}

func TestDeleteConfigMapKconfig(t *testing.T) {
	f := newFixture(t)

	kc := testutil.DeleteConfigMapKconfig()
	kcb := testutil.ConfigMapKconfigBinding(0)
	cm := testutil.ConfigMapWithData()
	expectedkcbupdate := testutil.EmptyKconfigEnvsKconfigBinding()

	f.kconfigLister = append(f.kconfigLister, &kc)
	f.kbindingLister = append(f.kbindingLister, &kcb)
	f.configmapLister = append(f.configmapLister, &cm)
	f.kcobjects = append(f.kcobjects, &kc)
	f.kcobjects = append(f.kcobjects, &kcb)
	f.stdobjects = append(f.stdobjects, &cm)

	f.expectUpdateKconfigBindingAction(&expectedkcbupdate)

	key, _ := cache.MetaNamespaceKeyFunc(&kc.ObjectMeta)
	f.run(key)
}

func TestDeleteSecretKconfig(t *testing.T) {
	f := newFixture(t)

	kc := testutil.DeleteSecretKconfig()
	kcb := testutil.SecretKconfigBinding(0)
	secret := testutil.SecretWithData()
	expectedkcbupdate := testutil.EmptyKconfigEnvsKconfigBinding()

	f.kconfigLister = append(f.kconfigLister, &kc)
	f.kbindingLister = append(f.kbindingLister, &kcb)
	f.secretLister = append(f.secretLister, &secret)
	f.kcobjects = append(f.kcobjects, &kc)
	f.kcobjects = append(f.kcobjects, &kcb)
	f.stdobjects = append(f.stdobjects, &secret)

	f.expectUpdateKconfigBindingAction(&expectedkcbupdate)

	key, _ := cache.MetaNamespaceKeyFunc(&kc.ObjectMeta)
	f.run(key)
}

func TestKconfigDeletionRemovesKconfigEnvs(t *testing.T) {
	f := newFixture(t)

	kc := testutil.ValueKconfig()
	now := metav1.Now()
	kc.ObjectMeta.DeletionTimestamp = &now

	kcb := testutil.ValueKconfigBinding()
	expectedkcbupdate := testutil.KconfigBinding()

	f.kbindingLister = append(f.kbindingLister, &kcb)
	f.kcobjects = append(f.kcobjects, &kc)
	f.kcobjects = append(f.kcobjects, &kcb)

	f.expectUpdateKconfigBindingAction(&expectedkcbupdate)

	key, _ := cache.MetaNamespaceKeyFunc(&kc.ObjectMeta)
	f.runDelete(cache.DeletedFinalStateUnknown{Key: key, Obj: &kc})
}

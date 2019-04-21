package kconfig

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/gbraxton/kconfig/pkg/apis/kconfigcontroller/v1alpha1"
	kcfake "github.com/gbraxton/kconfig/pkg/client/clientset/versioned/fake"
	kcinformers "github.com/gbraxton/kconfig/pkg/client/informers/externalversions"
	testutil "github.com/gbraxton/kconfig/test/util"
	v1 "k8s.io/api/core/v1"
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
	configmapLister         []*v1.ConfigMap
	secretLister            []*v1.Secret
	kconfigLister           []*v1alpha1.Kconfig
	deploymentBindingLister []*v1alpha1.DeploymentBinding
	statefulSetBindingLister []*v1alpha1.StatefulSetBinding
	knativeServiceBindingLister []*v1alpha1.KnativeServiceBinding

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

	stdInformers := stdinformers.NewSharedInformerFactory(f.stdclient, 0)
	kcInformers := kcinformers.NewSharedInformerFactory(f.kcclient, 0)

	c := NewController(
		f.stdclient,
		f.kcclient,
		stdInformers.Core().V1().ConfigMaps(),
		stdInformers.Core().V1().Secrets(),
		kcInformers.Kconfigcontroller().V1alpha1().Kconfigs(),
		kcInformers.Kconfigcontroller().V1alpha1().DeploymentBindings(),
		kcInformers.Kconfigcontroller().V1alpha1().StatefulSetBindings(),
		kcInformers.Kconfigcontroller().V1alpha1().KnativeServiceBindings())
	c.recorder = &record.FakeRecorder{}
	c.configmapsSynced = alwaysReady
	c.secretsSynced = alwaysReady
	c.kconfigsSynced = alwaysReady
	c.deploymentBindingsSynced = alwaysReady
	c.statefulSetBindingsSynced = alwaysReady
	c.knativeServiceBindingsSynced = alwaysReady
	for _, cm := range f.configmapLister {
		_ = stdInformers.Core().V1().ConfigMaps().Informer().GetIndexer().Add(cm)
	}
	for _, sec := range f.secretLister {
		_ = stdInformers.Core().V1().Secrets().Informer().GetIndexer().Add(sec)
	}
	for _, kc := range f.kconfigLister {
		_ = kcInformers.Kconfigcontroller().V1alpha1().Kconfigs().Informer().GetIndexer().Add(kc)
	}
	for _, db := range f.deploymentBindingLister {
		_ = kcInformers.Kconfigcontroller().V1alpha1().DeploymentBindings().Informer().GetIndexer().Add(db)
	}
	for _, db := range f.statefulSetBindingLister {
		_ = kcInformers.Kconfigcontroller().V1alpha1().StatefulSetBindings().Informer().GetIndexer().Add(db)
	}
	for _, db := range f.knativeServiceBindingLister {
		_ = kcInformers.Kconfigcontroller().V1alpha1().KnativeServiceBindings().Informer().GetIndexer().Add(db)
	}
	return c, kcInformers, stdInformers, nil
}

func (f *fixture) runExpectError(kconfigName string, startInformers bool) {
	f.runSync(kconfigName, startInformers, true)
}

func (f *fixture) run(kconfigName string) {
	f.runSync(kconfigName, true, false)
}

func (f *fixture) runSync(kconfigName string, startInformers bool, expectError bool) {
	c, kcInformers, stdInformers, err := f.newController()
	if err != nil {
		f.t.Fatalf("error creating Kconfig controller: %v", err)
	}
	if startInformers {
		stopCh := make(chan struct{})
		defer close(stopCh)
		kcInformers.Start(stopCh)
		stdInformers.Start(stopCh)
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
	c, kcInformers, stdInformers, err := f.newController()
	if err != nil {
		f.t.Fatalf("error creating Kconfig controller: %v", err)
	}

	stopCh := make(chan struct{})
	defer close(stopCh)
	kcInformers.Start(stopCh)
	stdInformers.Start(stopCh)

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
	// CreateAction and UpdateAction have the same interface methods, so either will be checked here
	if expectedCreateAction, ok := expectedAction.(core.CreateAction); ok {
		createAction, _ := action.(core.CreateAction)
		if !reflect.DeepEqual(expectedCreateAction.GetObject(), createAction.GetObject()) {
			f.t.Errorf("Expected %s\n\t%+v\ngot\n\t%+v", expectedCreateAction.GetVerb(), expectedCreateAction.GetObject(), createAction.GetObject())
			return false
		}
		return true
	}
	return true
}

func filterStdInformerActions(actions []core.Action) []core.Action {
	ret := make([]core.Action, 0)
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
	ret := make([]core.Action, 0)
	for _, action := range actions {
		if len(action.GetNamespace()) == 0 &&
			(action.Matches("list", "kconfigs") ||
				action.Matches("list", "deploymentbindings") ||
				action.Matches("list", "statefulsetbindings") ||
				action.Matches("list", "knativeservicebindings") ||
				action.Matches("watch", "kconfigs") ||
				action.Matches("watch", "deploymentbindings") ||
				action.Matches("watch", "statefulsetbindings") ||
				action.Matches("watch", "knativeservicebindings")) {
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

func (f *fixture) expectUpdateDeploymentBindingAction(k *v1alpha1.DeploymentBinding) {
	resource := schema.GroupVersionResource{
		Group:    v1alpha1.SchemeGroupVersion.Group,
		Version:  v1alpha1.SchemeGroupVersion.Version,
		Resource: "deploymentbindings",
	}
	action := core.NewUpdateAction(resource, k.Namespace, k)
	f.kcactions = append(f.kcactions, action)
}

func (f *fixture) expectUpdateStatefulSetBindingAction(k *v1alpha1.StatefulSetBinding) {
	resource := schema.GroupVersionResource{
		Group:    v1alpha1.SchemeGroupVersion.Group,
		Version:  v1alpha1.SchemeGroupVersion.Version,
		Resource: "statefulsetbindings",
	}
	action := core.NewUpdateAction(resource, k.Namespace, k)
	f.kcactions = append(f.kcactions, action)
}

func (f *fixture) expectUpdateKnativeServiceBindingAction(k *v1alpha1.KnativeServiceBinding) {
	resource := schema.GroupVersionResource{
		Group:    v1alpha1.SchemeGroupVersion.Group,
		Version:  v1alpha1.SchemeGroupVersion.Version,
		Resource: "knativeservicebindings",
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
	db := testutil.DeploymentBinding()
	ssb := testutil.StatefulSetBinding()
	ksb := testutil.KnativeServiceBinding()
	expectedDbUpdate := testutil.ValueDeploymentBinding()
	expectedSsbUpdate := testutil.ValueStatefulSetBinding()
	expectedKsbUpdate := testutil.ValueKnativeServiceBinding()

	f.kconfigLister = append(f.kconfigLister, &kc)
	f.deploymentBindingLister = append(f.deploymentBindingLister, &db)
	f.statefulSetBindingLister = append(f.statefulSetBindingLister, &ssb)
	f.knativeServiceBindingLister = append(f.knativeServiceBindingLister, &ksb)

	f.kcobjects = append(f.kcobjects, &kc)
	f.kcobjects = append(f.kcobjects, &db)
	f.kcobjects = append(f.kcobjects, &ssb)
	f.kcobjects = append(f.kcobjects, &ksb)

	f.expectUpdateDeploymentBindingAction(&expectedDbUpdate)
	f.expectUpdateStatefulSetBindingAction(&expectedSsbUpdate)
	f.expectUpdateKnativeServiceBindingAction(&expectedKsbUpdate)

	key, _ := cache.MetaNamespaceKeyFunc(&kc.ObjectMeta)
	f.run(key)
}

func TestValueKconfigWithEmptyType(t *testing.T) {
	f := newFixture(t)

	kc := testutil.ValueKconfig()
	kc.Spec.EnvConfigs[0].Type = ""
	db := testutil.DeploymentBinding()
	ssb := testutil.StatefulSetBinding()
	ksb := testutil.KnativeServiceBinding()
	expectedKcUpdate := testutil.ValueKconfig()
	expectedDbUpdate := testutil.ValueDeploymentBinding()
	expectedSsbUpdate := testutil.ValueStatefulSetBinding()
	expectedKsbUpdate := testutil.ValueKnativeServiceBinding()

	f.kconfigLister = append(f.kconfigLister, &kc)
	f.deploymentBindingLister = append(f.deploymentBindingLister, &db)
	f.statefulSetBindingLister = append(f.statefulSetBindingLister, &ssb)
	f.knativeServiceBindingLister = append(f.knativeServiceBindingLister, &ksb)
	f.kcobjects = append(f.kcobjects, &kc)
	f.kcobjects = append(f.kcobjects, &db)
	f.kcobjects = append(f.kcobjects, &ssb)
	f.kcobjects = append(f.kcobjects, &ksb)

	f.expectUpdateKconfigAction(&expectedKcUpdate)
	f.expectUpdateDeploymentBindingAction(&expectedDbUpdate)
	f.expectUpdateStatefulSetBindingAction(&expectedSsbUpdate)
	f.expectUpdateKnativeServiceBindingAction(&expectedKsbUpdate)

	key, _ := cache.MetaNamespaceKeyFunc(&kc.ObjectMeta)
	f.run(key)
}

func TestConfigmapKconfig(t *testing.T) {
	f := newFixture(t)

	kc := testutil.ConfigMapKconfig(testutil.DefaultConfigMapName)
	db := testutil.DeploymentBinding()
	ssb := testutil.StatefulSetBinding()
	ksb := testutil.KnativeServiceBinding()
	expectedKcbUpdate := testutil.ConfigMapDeploymentBinding(0, testutil.DefaultConfigMapName)
	expectedSsbUpdate := testutil.ConfigMapStatefulSetBinding(0, testutil.DefaultConfigMapName)
	expectedKsbUpdate := testutil.ConfigMapKnativeServiceBinding(0, testutil.DefaultConfigMapName)

	f.kconfigLister = append(f.kconfigLister, &kc)
	f.deploymentBindingLister = append(f.deploymentBindingLister, &db)
	f.statefulSetBindingLister = append(f.statefulSetBindingLister, &ssb)
	f.knativeServiceBindingLister = append(f.knativeServiceBindingLister, &ksb)
	f.kcobjects = append(f.kcobjects, &kc)
	f.kcobjects = append(f.kcobjects, &db)
	f.kcobjects = append(f.kcobjects, &ssb)
	f.kcobjects = append(f.kcobjects, &ksb)

	f.expectUpdateDeploymentBindingAction(&expectedKcbUpdate)
	f.expectUpdateStatefulSetBindingAction(&expectedSsbUpdate)
	f.expectUpdateKnativeServiceBindingAction(&expectedKsbUpdate)

	key, _ := cache.MetaNamespaceKeyFunc(&kc.ObjectMeta)
	f.run(key)
}

func TestSecretKconfig(t *testing.T) {
	f := newFixture(t)

	kc := testutil.SecretKconfig(testutil.DefaultSecretName)
	db := testutil.DeploymentBinding()
	ssb := testutil.StatefulSetBinding()
	ksb := testutil.KnativeServiceBinding()
	expectedKcbUpdate := testutil.SecretDeploymentBinding(0, testutil.DefaultSecretName)
	expectedSsbUpdate := testutil.SecretStatefulSetBinding(0, testutil.DefaultSecretName)
	expectedKsbUpdate := testutil.SecretKnativeServiceBinding(0, testutil.DefaultSecretName)

	f.kconfigLister = append(f.kconfigLister, &kc)
	f.deploymentBindingLister = append(f.deploymentBindingLister, &db)
	f.statefulSetBindingLister = append(f.statefulSetBindingLister, &ssb)
	f.knativeServiceBindingLister = append(f.knativeServiceBindingLister, &ksb)
	f.kcobjects = append(f.kcobjects, &kc)
	f.kcobjects = append(f.kcobjects, &db)
	f.kcobjects = append(f.kcobjects, &ssb)
	f.kcobjects = append(f.kcobjects, &ksb)

	f.expectUpdateDeploymentBindingAction(&expectedKcbUpdate)
	f.expectUpdateStatefulSetBindingAction(&expectedSsbUpdate)
	f.expectUpdateKnativeServiceBindingAction(&expectedKsbUpdate)

	key, _ := cache.MetaNamespaceKeyFunc(&kc.ObjectMeta)
	f.run(key)
}

func TestFieldRefKconfig(t *testing.T) {
	f := newFixture(t)

	kc := testutil.FieldRefKconfig()
	db := testutil.DeploymentBinding()
	ssb := testutil.StatefulSetBinding()
	ksb := testutil.KnativeServiceBinding()
	expectedDbUpdate := testutil.FieldRefDeploymentBinding(0)
	expectedSsbUpdate := testutil.FieldRefStatefulSetBinding(0)
	expectedKsbUpdate := testutil.FieldRefKnativeServiceBinding(0)

	f.kconfigLister = append(f.kconfigLister, &kc)
	f.deploymentBindingLister = append(f.deploymentBindingLister, &db)
	f.statefulSetBindingLister = append(f.statefulSetBindingLister, &ssb)
	f.knativeServiceBindingLister = append(f.knativeServiceBindingLister, &ksb)
	f.kcobjects = append(f.kcobjects, &kc)
	f.kcobjects = append(f.kcobjects, &db)
	f.kcobjects = append(f.kcobjects, &ssb)
	f.kcobjects = append(f.kcobjects, &ksb)

	f.expectUpdateDeploymentBindingAction(&expectedDbUpdate)
	f.expectUpdateStatefulSetBindingAction(&expectedSsbUpdate)
	f.expectUpdateKnativeServiceBindingAction(&expectedKsbUpdate)

	key, _ := cache.MetaNamespaceKeyFunc(&kc.ObjectMeta)
	f.run(key)
}

func TestResourceFieldRefKconfig(t *testing.T) {
	f := newFixture(t)

	kc := testutil.ResourceFieldRefKconfig()
	db := testutil.DeploymentBinding()
	ssb := testutil.StatefulSetBinding()
	ksb := testutil.KnativeServiceBinding()
	expectedDbUpdate := testutil.ResourceFieldRefDeploymentBinding(0)
	expectedSsbUpdate := testutil.ResourceFieldRefStatefulSetBinding(0)
	expectedKsbUpdate := testutil.ResourceFieldRefKnativeServiceBinding(0)

	f.kconfigLister = append(f.kconfigLister, &kc)
	f.deploymentBindingLister = append(f.deploymentBindingLister, &db)
	f.statefulSetBindingLister = append(f.statefulSetBindingLister, &ssb)
	f.knativeServiceBindingLister = append(f.knativeServiceBindingLister, &ksb)
	f.kcobjects = append(f.kcobjects, &kc)
	f.kcobjects = append(f.kcobjects, &db)
	f.kcobjects = append(f.kcobjects, &ssb)
	f.kcobjects = append(f.kcobjects, &ksb)

	f.expectUpdateDeploymentBindingAction(&expectedDbUpdate)
	f.expectUpdateStatefulSetBindingAction(&expectedSsbUpdate)
	f.expectUpdateKnativeServiceBindingAction(&expectedKsbUpdate)

	key, _ := cache.MetaNamespaceKeyFunc(&kc.ObjectMeta)
	f.run(key)
}

func TestAddConfigMapKconfig(t *testing.T) {
	f := newFixture(t)

	kc := testutil.AddConfigMapKconfig()
	db := testutil.DeploymentBinding()
	ssb := testutil.StatefulSetBinding()
	ksb := testutil.KnativeServiceBinding()
	expectedKcUpdate := testutil.ConfigMapKconfig(testutil.DefaultConfigMapName)
	expectedKcUpdate.Spec.EnvRefsVersion++
	expectedDbUpdate := testutil.ConfigMapDeploymentBinding(1, testutil.DefaultConfigMapName)
	expectedSsbUpdate := testutil.ConfigMapStatefulSetBinding(1, testutil.DefaultConfigMapName)
	expectedKsbUpdate := testutil.ConfigMapKnativeServiceBinding(1, testutil.DefaultConfigMapName)
	expectedCmCreate := testutil.ConfigMapWithData(testutil.DefaultConfigMapName)

	f.kconfigLister = append(f.kconfigLister, &kc)
	f.deploymentBindingLister = append(f.deploymentBindingLister, &db)
	f.statefulSetBindingLister = append(f.statefulSetBindingLister, &ssb)
	f.knativeServiceBindingLister = append(f.knativeServiceBindingLister, &ksb)
	f.kcobjects = append(f.kcobjects, &kc)
	f.kcobjects = append(f.kcobjects, &db)
	f.kcobjects = append(f.kcobjects, &ssb)
	f.kcobjects = append(f.kcobjects, &ksb)

	f.expectUpdateKconfigAction(&expectedKcUpdate)
	f.expectCreateConfigMapAction(&expectedCmCreate)
	f.expectUpdateDeploymentBindingAction(&expectedDbUpdate)
	f.expectUpdateStatefulSetBindingAction(&expectedSsbUpdate)
	f.expectUpdateKnativeServiceBindingAction(&expectedKsbUpdate)

	key, _ := cache.MetaNamespaceKeyFunc(&kc.ObjectMeta)
	f.run(key)
}

func TestAddConfigMapKconfigWithoutRefName(t *testing.T) {
	f := newFixture(t)

	configMapName := fmt.Sprintf("%s%s", ReferenceResourceNamePrefix, testutil.DefaultName)
	kc := testutil.AddConfigMapKconfig()
	kc.Spec.EnvConfigs[0].RefName = nil
	db := testutil.DeploymentBinding()
	ssb := testutil.StatefulSetBinding()
	ksb := testutil.KnativeServiceBinding()
	expectedKcUpdate := testutil.ConfigMapKconfig(configMapName)
	expectedKcUpdate.Spec.EnvRefsVersion++
	expectedDbUpdate := testutil.ConfigMapDeploymentBinding(1, configMapName)
	expectedSsbUpdate := testutil.ConfigMapStatefulSetBinding(1, configMapName)
	expectedKsbUpdate := testutil.ConfigMapKnativeServiceBinding(1, configMapName)
	expectedCmCreate := testutil.ConfigMapWithData(configMapName)

	f.kconfigLister = append(f.kconfigLister, &kc)
	f.deploymentBindingLister = append(f.deploymentBindingLister, &db)
	f.statefulSetBindingLister = append(f.statefulSetBindingLister, &ssb)
	f.knativeServiceBindingLister = append(f.knativeServiceBindingLister, &ksb)
	f.kcobjects = append(f.kcobjects, &kc)
	f.kcobjects = append(f.kcobjects, &db)
	f.kcobjects = append(f.kcobjects, &ssb)
	f.kcobjects = append(f.kcobjects, &ksb)

	f.expectUpdateKconfigAction(&expectedKcUpdate)
	f.expectCreateConfigMapAction(&expectedCmCreate)
	f.expectUpdateDeploymentBindingAction(&expectedDbUpdate)
	f.expectUpdateStatefulSetBindingAction(&expectedSsbUpdate)
	f.expectUpdateKnativeServiceBindingAction(&expectedKsbUpdate)

	key, _ := cache.MetaNamespaceKeyFunc(&kc.ObjectMeta)
	f.run(key)
}

func TestAddFieldRefKconfig(t *testing.T) {
	f := newFixture(t)

	kc := testutil.AddFieldRefKconfig()
	db := testutil.DeploymentBinding()
	ssb := testutil.StatefulSetBinding()
	ksb := testutil.KnativeServiceBinding()
	expectedKcUpdate := testutil.FieldRefKconfig()
	expectedDbUpdate := testutil.FieldRefDeploymentBinding(0)
	expectedSsbUpdate := testutil.FieldRefStatefulSetBinding(0)
	expectedKsbUpdate := testutil.FieldRefKnativeServiceBinding(0)

	f.kconfigLister = append(f.kconfigLister, &kc)
	f.deploymentBindingLister = append(f.deploymentBindingLister, &db)
	f.statefulSetBindingLister = append(f.statefulSetBindingLister, &ssb)
	f.knativeServiceBindingLister = append(f.knativeServiceBindingLister, &ksb)
	f.kcobjects = append(f.kcobjects, &kc)
	f.kcobjects = append(f.kcobjects, &db)
	f.kcobjects = append(f.kcobjects, &ssb)
	f.kcobjects = append(f.kcobjects, &ksb)

	f.expectUpdateKconfigAction(&expectedKcUpdate)
	f.expectUpdateDeploymentBindingAction(&expectedDbUpdate)
	f.expectUpdateStatefulSetBindingAction(&expectedSsbUpdate)
	f.expectUpdateKnativeServiceBindingAction(&expectedKsbUpdate)

	key, _ := cache.MetaNamespaceKeyFunc(&kc.ObjectMeta)
	f.run(key)
}

func TestAddResourceFieldRefKconfig(t *testing.T) {
	f := newFixture(t)

	kc := testutil.AddResourceFieldRefKconfig()
	db := testutil.DeploymentBinding()
	ssb := testutil.StatefulSetBinding()
	ksb := testutil.KnativeServiceBinding()
	expectedKcUpdate := testutil.ResourceFieldRefKconfig()
	expectedDbUpdate := testutil.ResourceFieldRefDeploymentBinding(0)
	expectedSsbUpdate := testutil.ResourceFieldRefStatefulSetBinding(0)
	expectedKsbUpdate := testutil.ResourceFieldRefKnativeServiceBinding(0)

	f.kconfigLister = append(f.kconfigLister, &kc)
	f.deploymentBindingLister = append(f.deploymentBindingLister, &db)
	f.statefulSetBindingLister = append(f.statefulSetBindingLister, &ssb)
	f.knativeServiceBindingLister = append(f.knativeServiceBindingLister, &ksb)
	f.kcobjects = append(f.kcobjects, &kc)
	f.kcobjects = append(f.kcobjects, &db)
	f.kcobjects = append(f.kcobjects, &ssb)
	f.kcobjects = append(f.kcobjects, &ksb)

	f.expectUpdateKconfigAction(&expectedKcUpdate)
	f.expectUpdateDeploymentBindingAction(&expectedDbUpdate)
	f.expectUpdateStatefulSetBindingAction(&expectedSsbUpdate)
	f.expectUpdateKnativeServiceBindingAction(&expectedKsbUpdate)

	key, _ := cache.MetaNamespaceKeyFunc(&kc.ObjectMeta)
	f.run(key)
}

func TestAddExistingConfigMapKconfig(t *testing.T) {
	f := newFixture(t)

	kc := testutil.AddConfigMapKconfig()
	db := testutil.DeploymentBinding()
	ssb := testutil.StatefulSetBinding()
	ksb := testutil.KnativeServiceBinding()
	cm := testutil.ConfigMap(testutil.DefaultConfigMapName)
	expectedKcUpdate := testutil.ConfigMapKconfig(testutil.DefaultConfigMapName)
	expectedKcUpdate.Spec.EnvRefsVersion++
	expectedDbUpdate := testutil.ConfigMapDeploymentBinding(1, testutil.DefaultConfigMapName)
	expectedSsbUpdate := testutil.ConfigMapStatefulSetBinding(1, testutil.DefaultConfigMapName)
	expectedKsbUpdate := testutil.ConfigMapKnativeServiceBinding(1, testutil.DefaultConfigMapName)
	expectedCmUpdate := testutil.ConfigMapWithData(testutil.DefaultConfigMapName)

	f.kconfigLister = append(f.kconfigLister, &kc)
	f.deploymentBindingLister = append(f.deploymentBindingLister, &db)
	f.statefulSetBindingLister = append(f.statefulSetBindingLister, &ssb)
	f.knativeServiceBindingLister = append(f.knativeServiceBindingLister, &ksb)
	f.configmapLister = append(f.configmapLister, &cm)
	f.kcobjects = append(f.kcobjects, &kc)
	f.kcobjects = append(f.kcobjects, &db)
	f.kcobjects = append(f.kcobjects, &ssb)
	f.kcobjects = append(f.kcobjects, &ksb)
	f.stdobjects = append(f.stdobjects, &cm)

	f.expectUpdateKconfigAction(&expectedKcUpdate)
	f.expectUpdateConfigMapAction(&expectedCmUpdate)
	f.expectUpdateDeploymentBindingAction(&expectedDbUpdate)
	f.expectUpdateStatefulSetBindingAction(&expectedSsbUpdate)
	f.expectUpdateKnativeServiceBindingAction(&expectedKsbUpdate)

	key, _ := cache.MetaNamespaceKeyFunc(&kc.ObjectMeta)
	f.run(key)
}

func TestAddSecretKconfig(t *testing.T) {
	f := newFixture(t)

	kc := testutil.AddSecretKconfig()
	db := testutil.DeploymentBinding()
	ssb := testutil.StatefulSetBinding()
	ksb := testutil.KnativeServiceBinding()
	expectedKcUpdate := testutil.SecretKconfig(testutil.DefaultSecretName)
	expectedKcUpdate.Spec.EnvRefsVersion++
	expectedDbUpdate := testutil.SecretDeploymentBinding(1, testutil.DefaultSecretName)
	expectedSsbUpdate := testutil.SecretStatefulSetBinding(1, testutil.DefaultSecretName)
	expectedKsbUpdate := testutil.SecretKnativeServiceBinding(1, testutil.DefaultSecretName)
	expectedSecretCreate := testutil.SecretWithData(testutil.DefaultSecretName)

	f.kconfigLister = append(f.kconfigLister, &kc)
	f.deploymentBindingLister = append(f.deploymentBindingLister, &db)
	f.statefulSetBindingLister = append(f.statefulSetBindingLister, &ssb)
	f.knativeServiceBindingLister = append(f.knativeServiceBindingLister, &ksb)
	f.kcobjects = append(f.kcobjects, &kc)
	f.kcobjects = append(f.kcobjects, &db)
	f.kcobjects = append(f.kcobjects, &ssb)
	f.kcobjects = append(f.kcobjects, &ksb)

	f.expectUpdateKconfigAction(&expectedKcUpdate)
	f.expectCreateSecretAction(&expectedSecretCreate)
	f.expectUpdateDeploymentBindingAction(&expectedDbUpdate)
	f.expectUpdateStatefulSetBindingAction(&expectedSsbUpdate)
	f.expectUpdateKnativeServiceBindingAction(&expectedKsbUpdate)

	key, _ := cache.MetaNamespaceKeyFunc(&kc.ObjectMeta)
	f.run(key)
}

func TestAddSecretKconfigWithoutRefName(t *testing.T) {
	f := newFixture(t)

	kc := testutil.AddSecretKconfig()
	kc.Spec.EnvConfigs[0].RefName = nil
	db := testutil.DeploymentBinding()
	ssb := testutil.StatefulSetBinding()
	ksb := testutil.KnativeServiceBinding()
	secretName := fmt.Sprintf("%s%s", ReferenceResourceNamePrefix, testutil.DefaultName)
	expectedKcUpdate := testutil.SecretKconfig(secretName)
	expectedKcUpdate.Spec.EnvRefsVersion++
	expectedDbUpdate := testutil.SecretDeploymentBinding(1, secretName)
	expectedSsbUpdate := testutil.SecretStatefulSetBinding(1, secretName)
	expectedKsbUpdate := testutil.SecretKnativeServiceBinding(1, secretName)
	expectedSecretCreate := testutil.SecretWithData(secretName)

	f.kconfigLister = append(f.kconfigLister, &kc)
	f.deploymentBindingLister = append(f.deploymentBindingLister, &db)
	f.statefulSetBindingLister = append(f.statefulSetBindingLister, &ssb)
	f.knativeServiceBindingLister = append(f.knativeServiceBindingLister, &ksb)
	f.kcobjects = append(f.kcobjects, &kc)
	f.kcobjects = append(f.kcobjects, &db)
	f.kcobjects = append(f.kcobjects, &ssb)
	f.kcobjects = append(f.kcobjects, &ksb)

	f.expectUpdateKconfigAction(&expectedKcUpdate)
	f.expectCreateSecretAction(&expectedSecretCreate)
	f.expectUpdateDeploymentBindingAction(&expectedDbUpdate)
	f.expectUpdateStatefulSetBindingAction(&expectedSsbUpdate)
	f.expectUpdateKnativeServiceBindingAction(&expectedKsbUpdate)

	key, _ := cache.MetaNamespaceKeyFunc(&kc.ObjectMeta)
	f.run(key)
}

func TestAddExistingSecretKconfig(t *testing.T) {
	f := newFixture(t)

	kc := testutil.AddSecretKconfig()
	db := testutil.DeploymentBinding()
	ssb := testutil.StatefulSetBinding()
	ksb := testutil.KnativeServiceBinding()
	secret := testutil.Secret(testutil.DefaultSecretName)
	expectedKcUpdate := testutil.SecretKconfig(testutil.DefaultSecretName)
	expectedKcUpdate.Spec.EnvRefsVersion++
	expectedDbUpdate := testutil.SecretDeploymentBinding(1, testutil.DefaultSecretName)
	expectedSsbUpdate := testutil.SecretStatefulSetBinding(1, testutil.DefaultSecretName)
	expectedKsbUpdate := testutil.SecretKnativeServiceBinding(1, testutil.DefaultSecretName)
	expectedSecretUpdate := testutil.SecretWithData(testutil.DefaultSecretName)

	f.kconfigLister = append(f.kconfigLister, &kc)
	f.deploymentBindingLister = append(f.deploymentBindingLister, &db)
	f.statefulSetBindingLister = append(f.statefulSetBindingLister, &ssb)
	f.knativeServiceBindingLister = append(f.knativeServiceBindingLister, &ksb)
	f.secretLister = append(f.secretLister, &secret)
	f.kcobjects = append(f.kcobjects, &kc)
	f.kcobjects = append(f.kcobjects, &db)
	f.kcobjects = append(f.kcobjects, &ssb)
	f.kcobjects = append(f.kcobjects, &ksb)
	f.stdobjects = append(f.stdobjects, &secret)

	f.expectUpdateKconfigAction(&expectedKcUpdate)
	f.expectUpdateSecretAction(&expectedSecretUpdate)
	f.expectUpdateDeploymentBindingAction(&expectedDbUpdate)
	f.expectUpdateStatefulSetBindingAction(&expectedSsbUpdate)
	f.expectUpdateKnativeServiceBindingAction(&expectedKsbUpdate)

	key, _ := cache.MetaNamespaceKeyFunc(&kc.ObjectMeta)
	f.run(key)
}
func TestValueUpdateKconfig(t *testing.T) {
	f := newFixture(t)

	kc := testutil.NewValueKconfig()
	db := testutil.ValueDeploymentBinding()
	ssb := testutil.ValueStatefulSetBinding()
	ksb := testutil.ValueKnativeServiceBinding()
	expectedDbUpdate := testutil.NewValueDeploymentBinding()
	expectedSsbUpdate := testutil.NewValueStatefulSetBinding()
	expectedKsbUpdate := testutil.NewValueKnativeServiceBinding()

	f.kconfigLister = append(f.kconfigLister, &kc)
	f.deploymentBindingLister = append(f.deploymentBindingLister, &db)
	f.statefulSetBindingLister = append(f.statefulSetBindingLister, &ssb)
	f.knativeServiceBindingLister = append(f.knativeServiceBindingLister, &ksb)
	f.kcobjects = append(f.kcobjects, &kc)
	f.kcobjects = append(f.kcobjects, &db)
	f.kcobjects = append(f.kcobjects, &ssb)
	f.kcobjects = append(f.kcobjects, &ksb)

	f.expectUpdateDeploymentBindingAction(&expectedDbUpdate)
	f.expectUpdateStatefulSetBindingAction(&expectedSsbUpdate)
	f.expectUpdateKnativeServiceBindingAction(&expectedKsbUpdate)

	key, _ := cache.MetaNamespaceKeyFunc(&kc.ObjectMeta)
	f.run(key)
}

func TestUpdateConfigMapKconfig(t *testing.T) {
	f := newFixture(t)

	kc := testutil.UpdateConfigMapKconfig()
	db := testutil.ConfigMapDeploymentBinding(0, testutil.DefaultConfigMapName)
	ssb := testutil.ConfigMapStatefulSetBinding(0, testutil.DefaultConfigMapName)
	ksb := testutil.ConfigMapKnativeServiceBinding(0, testutil.DefaultConfigMapName)
	cm := testutil.ConfigMapWithData(testutil.DefaultConfigMapName)
	expectedKcUpdate := testutil.ConfigMapKconfig(testutil.DefaultConfigMapName)
	expectedKcUpdate.Spec.EnvRefsVersion++
	expectedDbUpdate := testutil.ConfigMapDeploymentBinding(1, testutil.DefaultConfigMapName)
	expectedSsbUpdate := testutil.ConfigMapStatefulSetBinding(1, testutil.DefaultConfigMapName)
	expectedKsbUpdate := testutil.ConfigMapKnativeServiceBinding(1, testutil.DefaultConfigMapName)
	expectedCmUpdate := testutil.ConfigMapWithNewData(testutil.DefaultConfigMapName)

	f.kconfigLister = append(f.kconfigLister, &kc)
	f.deploymentBindingLister = append(f.deploymentBindingLister, &db)
	f.statefulSetBindingLister = append(f.statefulSetBindingLister, &ssb)
	f.knativeServiceBindingLister = append(f.knativeServiceBindingLister, &ksb)
	f.configmapLister = append(f.configmapLister, &cm)
	f.kcobjects = append(f.kcobjects, &kc)
	f.kcobjects = append(f.kcobjects, &db)
	f.kcobjects = append(f.kcobjects, &ssb)
	f.kcobjects = append(f.kcobjects, &ksb)
	f.stdobjects = append(f.stdobjects, &cm)

	f.expectUpdateKconfigAction(&expectedKcUpdate)
	f.expectUpdateConfigMapAction(&expectedCmUpdate)
	f.expectUpdateDeploymentBindingAction(&expectedDbUpdate)
	f.expectUpdateStatefulSetBindingAction(&expectedSsbUpdate)
	f.expectUpdateKnativeServiceBindingAction(&expectedKsbUpdate)

	key, _ := cache.MetaNamespaceKeyFunc(&kc.ObjectMeta)
	f.run(key)
}

func TestUpdateSecretKconfig(t *testing.T) {
	f := newFixture(t)

	kc := testutil.UpdateSecretKconfig()
	db := testutil.SecretDeploymentBinding(0, testutil.DefaultSecretName)
	ssb := testutil.SecretStatefulSetBinding(0, testutil.DefaultSecretName)
	ksb := testutil.SecretKnativeServiceBinding(0, testutil.DefaultSecretName)
	secret := testutil.SecretWithData(testutil.DefaultSecretName)
	expectedKcUpdate := testutil.SecretKconfig(testutil.DefaultSecretName)
	expectedKcUpdate.Spec.EnvRefsVersion++
	expectedDbUpdate := testutil.SecretDeploymentBinding(1, testutil.DefaultSecretName)
	expectedSsbUpdate := testutil.SecretStatefulSetBinding(1, testutil.DefaultSecretName)
	expectedKsbUpdate := testutil.SecretKnativeServiceBinding(1, testutil.DefaultSecretName)
	expectedSecretUpdate := testutil.SecretWithNewData(testutil.DefaultSecretName)

	f.kconfigLister = append(f.kconfigLister, &kc)
	f.deploymentBindingLister = append(f.deploymentBindingLister, &db)
	f.statefulSetBindingLister = append(f.statefulSetBindingLister, &ssb)
	f.knativeServiceBindingLister = append(f.knativeServiceBindingLister, &ksb)
	f.secretLister = append(f.secretLister, &secret)
	f.kcobjects = append(f.kcobjects, &kc)
	f.kcobjects = append(f.kcobjects, &db)
	f.kcobjects = append(f.kcobjects, &ssb)
	f.kcobjects = append(f.kcobjects, &ksb)
	f.stdobjects = append(f.stdobjects, &secret)

	f.expectUpdateKconfigAction(&expectedKcUpdate)
	f.expectUpdateSecretAction(&expectedSecretUpdate)
	f.expectUpdateDeploymentBindingAction(&expectedDbUpdate)
	f.expectUpdateStatefulSetBindingAction(&expectedSsbUpdate)
	f.expectUpdateKnativeServiceBindingAction(&expectedKsbUpdate)

	key, _ := cache.MetaNamespaceKeyFunc(&kc.ObjectMeta)
	f.run(key)
}

func TestValueDeleteKconfig(t *testing.T) {
	f := newFixture(t)

	kc := testutil.Kconfig()
	db := testutil.ValueDeploymentBinding()
	ssb := testutil.ValueStatefulSetBinding()
	ksb := testutil.ValueKnativeServiceBinding()
	expectedDbUpdate := testutil.EmptyKconfigEnvsDeploymentBinding()
	expectedSsbUpdate := testutil.EmptyKconfigEnvsStatefulSetBinding()
	expectedKsbUpdate := testutil.EmptyKconfigEnvsKnativeServiceBinding()

	f.kconfigLister = append(f.kconfigLister, &kc)
	f.deploymentBindingLister = append(f.deploymentBindingLister, &db)
	f.statefulSetBindingLister = append(f.statefulSetBindingLister, &ssb)
	f.knativeServiceBindingLister = append(f.knativeServiceBindingLister, &ksb)
	f.kcobjects = append(f.kcobjects, &kc)
	f.kcobjects = append(f.kcobjects, &db)
	f.kcobjects = append(f.kcobjects, &ssb)
	f.kcobjects = append(f.kcobjects, &ksb)

	f.expectUpdateDeploymentBindingAction(&expectedDbUpdate)
	f.expectUpdateStatefulSetBindingAction(&expectedSsbUpdate)
	f.expectUpdateKnativeServiceBindingAction(&expectedKsbUpdate)

	key, _ := cache.MetaNamespaceKeyFunc(&kc.ObjectMeta)
	f.run(key)
}

func TestDeleteConfigMapKconfig(t *testing.T) {
	f := newFixture(t)

	kc := testutil.DeleteConfigMapKconfig()
	db := testutil.ConfigMapDeploymentBinding(0, testutil.DefaultConfigMapName)
	ssb := testutil.ConfigMapStatefulSetBinding(0, testutil.DefaultConfigMapName)
	ksb := testutil.ConfigMapKnativeServiceBinding(0, testutil.DefaultConfigMapName)
	cm := testutil.ConfigMapWithData(testutil.DefaultConfigMapName)
	expectedDbUpdate := testutil.EmptyKconfigEnvsDeploymentBinding()
	expectedSsbUpdate := testutil.EmptyKconfigEnvsStatefulSetBinding()
	expectedKsbUpdate := testutil.EmptyKconfigEnvsKnativeServiceBinding()

	f.kconfigLister = append(f.kconfigLister, &kc)
	f.deploymentBindingLister = append(f.deploymentBindingLister, &db)
	f.statefulSetBindingLister = append(f.statefulSetBindingLister, &ssb)
	f.knativeServiceBindingLister = append(f.knativeServiceBindingLister, &ksb)
	f.configmapLister = append(f.configmapLister, &cm)
	f.kcobjects = append(f.kcobjects, &kc)
	f.kcobjects = append(f.kcobjects, &db)
	f.kcobjects = append(f.kcobjects, &ssb)
	f.kcobjects = append(f.kcobjects, &ksb)
	f.stdobjects = append(f.stdobjects, &cm)

	f.expectUpdateDeploymentBindingAction(&expectedDbUpdate)
	f.expectUpdateStatefulSetBindingAction(&expectedSsbUpdate)
	f.expectUpdateKnativeServiceBindingAction(&expectedKsbUpdate)

	key, _ := cache.MetaNamespaceKeyFunc(&kc.ObjectMeta)
	f.run(key)
}

func TestDeleteSecretKconfig(t *testing.T) {
	f := newFixture(t)

	kc := testutil.DeleteSecretKconfig()
	db := testutil.SecretDeploymentBinding(0, testutil.DefaultSecretName)
	ssb := testutil.SecretStatefulSetBinding(0, testutil.DefaultSecretName)
	ksb := testutil.SecretKnativeServiceBinding(0, testutil.DefaultSecretName)
	secret := testutil.SecretWithData(testutil.DefaultSecretName)
	expectedDbUpdate := testutil.EmptyKconfigEnvsDeploymentBinding()
	expectedSsbUpdate := testutil.EmptyKconfigEnvsStatefulSetBinding()
	expectedKsbUpdate := testutil.EmptyKconfigEnvsKnativeServiceBinding()

	f.kconfigLister = append(f.kconfigLister, &kc)
	f.deploymentBindingLister = append(f.deploymentBindingLister, &db)
	f.statefulSetBindingLister = append(f.statefulSetBindingLister, &ssb)
	f.knativeServiceBindingLister = append(f.knativeServiceBindingLister, &ksb)
	f.secretLister = append(f.secretLister, &secret)
	f.kcobjects = append(f.kcobjects, &kc)
	f.kcobjects = append(f.kcobjects, &db)
	f.kcobjects = append(f.kcobjects, &ssb)
	f.kcobjects = append(f.kcobjects, &ksb)
	f.stdobjects = append(f.stdobjects, &secret)

	f.expectUpdateDeploymentBindingAction(&expectedDbUpdate)
	f.expectUpdateStatefulSetBindingAction(&expectedSsbUpdate)
	f.expectUpdateKnativeServiceBindingAction(&expectedKsbUpdate)

	key, _ := cache.MetaNamespaceKeyFunc(&kc.ObjectMeta)
	f.run(key)
}

func TestKconfigDeletionRemovesKconfigEnvs(t *testing.T) {
	f := newFixture(t)

	kc := testutil.ValueKconfig()
	now := metav1.Now()
	kc.ObjectMeta.DeletionTimestamp = &now

	db := testutil.ValueDeploymentBinding()
	ssb := testutil.ValueStatefulSetBinding()
	ksb := testutil.ValueKnativeServiceBinding()
	expectedDbUpdate := testutil.DeploymentBinding()
	expectedSsbUpdate := testutil.StatefulSetBinding()
	expectedKsbUpdate := testutil.KnativeServiceBinding()

	f.deploymentBindingLister = append(f.deploymentBindingLister, &db)
	f.statefulSetBindingLister = append(f.statefulSetBindingLister, &ssb)
	f.knativeServiceBindingLister = append(f.knativeServiceBindingLister, &ksb)
	f.kcobjects = append(f.kcobjects, &kc)
	f.kcobjects = append(f.kcobjects, &db)
	f.kcobjects = append(f.kcobjects, &ssb)
	f.kcobjects = append(f.kcobjects, &ksb)

	f.expectUpdateDeploymentBindingAction(&expectedDbUpdate)
	f.expectUpdateStatefulSetBindingAction(&expectedSsbUpdate)
	f.expectUpdateKnativeServiceBindingAction(&expectedKsbUpdate)

	key, _ := cache.MetaNamespaceKeyFunc(&kc.ObjectMeta)
	f.runDelete(cache.DeletedFinalStateUnknown{Key: key, Obj: &kc})
}

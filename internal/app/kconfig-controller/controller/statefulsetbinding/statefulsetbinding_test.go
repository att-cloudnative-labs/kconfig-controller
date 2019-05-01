package statefulsetbinding

import (
"reflect"
"testing"

"github.com/att-cloudnative-labs/kconfig-controller/pkg/apis/kconfigcontroller/v1alpha1"
kcfake "github.com/att-cloudnative-labs/kconfig-controller/pkg/client/clientset/versioned/fake"
kcinformers "github.com/att-cloudnative-labs/kconfig-controller/pkg/client/informers/externalversions"
testutil "github.com/att-cloudnative-labs/kconfig-controller/test/util"
appsv1 "k8s.io/api/apps/v1"
"k8s.io/api/core/v1"
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
	configmapLister          []*v1.ConfigMap
	secretLister             []*v1.Secret
	statefulSetLister        []*appsv1.StatefulSet
	statefulsetBindingLister []*v1alpha1.StatefulSetBinding

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

	c := NewController(f.stdclient, f.kcclient, stdInformers.Core().V1().ConfigMaps(), stdInformers.Core().V1().Secrets(), stdInformers.Apps().V1().StatefulSets(), kcInformers.Kconfigcontroller().V1alpha1().StatefulSetBindings())
	c.recorder = &record.FakeRecorder{}
	c.configmapsSynced = alwaysReady
	c.secretsSynced = alwaysReady
	c.statefulsetsSynced = alwaysReady
	c.statefulsetBindingsSynced = alwaysReady
	for _, cm := range f.configmapLister {
		_ = stdInformers.Core().V1().ConfigMaps().Informer().GetIndexer().Add(cm)
	}
	for _, sec := range f.secretLister {
		_ = stdInformers.Core().V1().Secrets().Informer().GetIndexer().Add(sec)
	}
	for _, dep := range f.statefulSetLister {
		_ = stdInformers.Apps().V1().StatefulSets().Informer().GetIndexer().Add(dep)
	}
	for _, kcb := range f.statefulsetBindingLister {
		_ = kcInformers.Kconfigcontroller().V1alpha1().StatefulSetBindings().Informer().GetIndexer().Add(kcb)
	}
	return c, kcInformers, stdInformers, nil
}

func (f *fixture) runExpectError(kcbName string, startInformers bool) {
	f.runSync(kcbName, startInformers, true)
}

func (f *fixture) run(kcbName string) {
	f.runSync(kcbName, true, false)
}

func (f *fixture) runSync(kcbName string, startInformers bool, expectError bool) {
	c, kcInformers, stdInformers, err := f.newController()
	if err != nil {
		f.t.Fatalf("error creating DeploymentBinding controller: %v", err)
	}
	if startInformers {
		stopCh := make(chan struct{})
		defer close(stopCh)
		kcInformers.Start(stopCh)
		stdInformers.Start(stopCh)
	}

	err = c.syncHandler(kcbName)
	if !expectError && err != nil {
		f.t.Errorf("error syncing kconfig: %v", err)
	} else if expectError && err == nil {
		f.t.Error("expected error syncing kconfig, got nil")
	}

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
	ret := make([]core.Action, 0)
	for _, action := range actions {
		if len(action.GetNamespace()) == 0 &&
			(action.Matches("list", "configmaps") ||
				action.Matches("watch", "configmaps") ||
				action.Matches("list", "secrets") ||
				action.Matches("watch", "secrets") ||
				action.Matches("list", "deployments") ||
				action.Matches("watch", "deployments") ||
				action.Matches("list", "statefulsets") ||
				action.Matches("watch", "statefulsets")) {
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
			(action.Matches("watch", "kconfigs") ||
				action.Matches("list", "statefulsetbindings") ||
				action.Matches("watch", "statefulsetbindings")) {
			continue
		}
		ret = append(ret, action)
	}
	return ret
}

func (f *fixture) expectUpdateStatefulSetAction(d *appsv1.StatefulSet) {
	resource := schema.GroupVersionResource{
		Group:    appsv1.SchemeGroupVersion.Group,
		Version:  appsv1.SchemeGroupVersion.Version,
		Resource: "statefulsets",
	}
	action := core.NewUpdateAction(resource, d.Namespace, d)
	f.stdactions = append(f.kcactions, action)
}

func TestValueBindingUpdatesStatefulSet(t *testing.T) {
	f := newFixture(t)

	ssb := testutil.ValueStatefulSetBinding()
	ss := testutil.StatefulSet()
	sUpdate := testutil.ValueStatefulSet()

	f.statefulsetBindingLister = append(f.statefulsetBindingLister, &ssb)
	f.statefulSetLister = append(f.statefulSetLister, &ss)
	f.kcobjects = append(f.kcobjects, &ssb)
	f.stdobjects = append(f.stdobjects, &ss)

	f.expectUpdateStatefulSetAction(&sUpdate)

	key, _ := cache.MetaNamespaceKeyFunc(&ssb.ObjectMeta)
	f.run(key)
}


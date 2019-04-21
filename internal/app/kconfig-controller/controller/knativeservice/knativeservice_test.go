package knativeservice

import (
	"reflect"
	"testing"

	"github.com/att-cloudnative-labs/kconfig-controller/pkg/apis/kconfigcontroller/v1alpha1"
	kcfake "github.com/att-cloudnative-labs/kconfig-controller/pkg/client/clientset/versioned/fake"
	kcinformers "github.com/att-cloudnative-labs/kconfig-controller/pkg/client/informers/externalversions"
	knfake "github.com/att-cloudnative-labs/test/knativefakes"
	testutil "github.com/att-cloudnative-labs/test/util"
	knv1alpha1 "github.com/knative/serving/pkg/apis/serving/v1alpha1"
	kninformers "github.com/knative/serving/pkg/client/informers/externalversions"
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
	knclient *knfake.Clientset

	// Objects to put in the store
	knativeServiceLister        []*knv1alpha1.Service
	knativeServiceBindingLister []*v1alpha1.KnativeServiceBinding

	// Actions expected to happen on the client. Objects from here are also
	// preloaded into NewSimpleFake.
	stdactions []core.Action
	kcactions  []core.Action
	knactions []core.Action
	stdobjects []runtime.Object
	kcobjects  []runtime.Object
	knobjects []runtime.Object
}

func newFixture(t *testing.T) *fixture {
	f := &fixture{}
	f.t = t
	f.stdobjects = []runtime.Object{}
	f.kcobjects = []runtime.Object{}
	f.knobjects = []runtime.Object{}
	return f
}

func (f *fixture) newController() (*Controller, kcinformers.SharedInformerFactory, kninformers.SharedInformerFactory, stdinformers.SharedInformerFactory, error) {
	f.stdclient = stdfake.NewSimpleClientset(f.stdobjects...)
	f.kcclient = kcfake.NewSimpleClientset(f.kcobjects...)

	stdInformers := stdinformers.NewSharedInformerFactory(f.stdclient, 0)
	kcInformers := kcinformers.NewSharedInformerFactory(f.kcclient, 0)
	knInformers := kninformers.NewSharedInformerFactory(f.knclient, 0)

	c := NewController(f.stdclient, f.knclient, f.kcclient, knInformers.Serving().V1alpha1().Services(), kcInformers.Kconfigcontroller().V1alpha1().KnativeServiceBindings())
	c.recorder = &record.FakeRecorder{}
	c.kserviceSynced = alwaysReady
	for _, ks := range f.knativeServiceLister {
		_ = knInformers.Serving().V1alpha1().Services().Informer().GetIndexer().Add(ks)
	}
	for _, ksb := range f.knativeServiceBindingLister {
		_ = kcInformers.Kconfigcontroller().V1alpha1().KnativeServiceBindings().Informer().GetIndexer().Add(ksb)
	}
	return c, kcInformers, knInformers, stdInformers, nil
}

func (f *fixture) runExpectError(deploymentName string, startInformers bool) {
	f.runSync(deploymentName, startInformers, true)
}

func (f *fixture) run(deploymentName string) {
	f.runSync(deploymentName, true, false)
}

func (f *fixture) runSync(deploymentName string, startInformers bool, expectError bool) {
	c, kcInformers, knInformers, stdInformers, err := f.newController()
	if err != nil {
		f.t.Fatalf("error creating Kconfig controller: %v", err)
	}
	if startInformers {
		stopCh := make(chan struct{})
		defer close(stopCh)
		kcInformers.Start(stopCh)
		knInformers.Start(stopCh)
		stdInformers.Start(stopCh)
	}

	err = c.syncHandler(deploymentName)
	if !expectError && err != nil {
		f.t.Errorf("error syncing kconfig: %v", err)
	} else if expectError && err == nil {
		f.t.Error("expected error syncing kconfig, got nil")
	}

	f.checkActions()
}

// runDelete calls deleteKconfig instead of syncHandler
func (f *fixture) runDelete(obj interface{}) {
	c, kcInformers, knInformers, stdInformers, err := f.newController()
	if err != nil {
		f.t.Fatalf("error creating Deployment controller: %v", err)
	}
	stopCh := make(chan struct{})
	defer close(stopCh)
	kcInformers.Start(stopCh)
	knInformers.Start(stopCh)
	stdInformers.Start(stopCh)

	c.deleteHandler(obj)

	f.checkActions()
}

func (f *fixture) checkActions() {
	stdactions := filterKnInformerActions(f.stdclient.Actions())
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

func filterKnInformerActions(actions []core.Action) []core.Action {
	ret := make([]core.Action, 0)
	for _, action := range actions {
		if len(action.GetNamespace()) == 0 &&
			(action.Matches("watch", "services") ||
				action.Matches("list", "services")) {
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
			(action.Matches("watch", "knativeservicebindings") ||
				action.Matches("list", "knativeservicebindings")) {
			continue
		}
		ret = append(ret, action)
	}
	return ret
}

func (f *fixture) expectCreateKnativeServiceBindingAction(k *v1alpha1.KnativeServiceBinding) {
	resource := schema.GroupVersionResource{
		Group:    v1alpha1.SchemeGroupVersion.Group,
		Version:  v1alpha1.SchemeGroupVersion.Version,
		Resource: "KnativeServiceBindings",
	}
	action := core.NewCreateAction(resource, k.Namespace, k)
	f.kcactions = append(f.kcactions, action)
}

func (f *fixture) expectUpdateKnativeServiceBindingAction(k *v1alpha1.KnativeServiceBinding) {
	resource := schema.GroupVersionResource{
		Group:    v1alpha1.SchemeGroupVersion.Group,
		Version:  v1alpha1.SchemeGroupVersion.Version,
		Resource: "KnativeServiceBindings",
	}
	action := core.NewUpdateAction(resource, k.Namespace, k)
	f.kcactions = append(f.kcactions, action)
}

func (f *fixture) expectDeleteKnativeServiceBindingAction(k *v1alpha1.KnativeServiceBinding) {
	resource := schema.GroupVersionResource{
		Group:    v1alpha1.SchemeGroupVersion.Group,
		Version:  v1alpha1.SchemeGroupVersion.Version,
		Resource: "KnativeServiceBindings",
	}
	action := core.NewDeleteAction(resource, k.Namespace, k.Name)
	f.kcactions = append(f.kcactions, action)
}

func TestNewKnativeServiceCreatesKnativeServiceBinding(t *testing.T) {
	f := newFixture(t)

	ks := testutil.KnativeService()
	expectedKsbCreate := testutil.KnativeServiceBinding()
	expectedKsbUpdate := testutil.KnativeServiceBinding()

	f.knativeServiceLister = append(f.knativeServiceLister, &ks)
	f.knobjects = append(f.knobjects, &ks)

	f.expectCreateKnativeServiceBindingAction(&expectedKsbCreate)
	f.expectUpdateKnativeServiceBindingAction(&expectedKsbUpdate)

	key, _ := cache.MetaNamespaceKeyFunc(&ks.ObjectMeta)
	f.run(key)
}

func TestDeleteKnativeServiceDeletesKnativeServiceBinding(t *testing.T) {
	f := newFixture(t)

	ks := testutil.KnativeService()
	now := metav1.Now()
	ks.ObjectMeta.DeletionTimestamp = &now
	ksb := testutil.KnativeServiceBinding()

	f.knativeServiceBindingLister = append(f.knativeServiceBindingLister, &ksb)
	f.kcobjects = append(f.kcobjects, &ksb)

	f.expectDeleteKnativeServiceBindingAction(&ksb)

	key, _ := cache.MetaNamespaceKeyFunc(&ks.ObjectMeta)
	f.runDelete(cache.DeletedFinalStateUnknown{Key: key, Obj: &ks})
}

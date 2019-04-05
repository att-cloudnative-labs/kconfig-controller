package kconfig

import (
	"reflect"
	"testing"

	"gotest.tools/assert"

	testutil "github.com/gbraxton/kconfig/test/util"

	"k8s.io/apimachinery/pkg/runtime"
	stdinformers "k8s.io/client-go/informers"
	stdfake "k8s.io/client-go/kubernetes/fake"
	core "k8s.io/client-go/testing"
)

func TestPutSingleExternalAction(t *testing.T) {
	cache := NewExternalActionCache()
	cache.Put("configmap/testns/testname", ExternalAction{Key: "mykey", Value: "myvalue"})
	action := cache.Get("configmap/testns/testname")
	assert.Equal(t, 1, len(action))
	assert.Equal(t, action[0], ExternalAction{Key: "mykey", Value: "myvalue"})
}

func TestPutMultipleExternalAction(t *testing.T) {
	cache := NewExternalActionCache()
	cache.Put("configmap/testns/testname", ExternalAction{Key: "mykey", Value: "myvalue"})
	cache.Put("configmap/testns/testname", ExternalAction{Key: "anotherkey", Value: "anotherval"})
	action := cache.Get("configmap/testns/testname")
	assert.Equal(t, 2, len(action))
	assert.Equal(t, action[0], ExternalAction{Key: "mykey", Value: "myvalue"})
	assert.Equal(t, action[1], ExternalAction{Key: "anotherkey", Value: "anotherval"})
}

func TestExternalActionKeyFunc(t *testing.T) {
	externalType := "configmap"
	name := "myname"
	key := ExternalActionKeyFunc(externalType, name)
	assert.Equal(t, "configmap/myname", key)
}

func TestSplitExternalActionKey(t *testing.T) {
	key := "configmap/myname"
	eaType, name, err := SplitExternalActionKey(key)
	if err != nil {
		t.Fail()
	}
	assert.Equal(t, "configmap", eaType)
	assert.Equal(t, "myname", name)
}

func TestNewConfigMapFromExternalAction(t *testing.T) {
	configmapType := "configmap"
	namespace := testutil.DefaultNamespace
	name := testutil.DefaultConfigMapName
	key := ExternalActionKeyFunc(configmapType, name)

	action := ExternalAction{Key: testutil.DefaultReferenceKey, Value: testutil.DefaultValue}
	cache := NewExternalActionCache()
	cache.Put(key, action)

	stdclient := stdfake.NewSimpleClientset()
	stdinformers := stdinformers.NewSharedInformerFactory(stdclient, 0)

	c := Controller{
		stdclient:       stdclient,
		configmaplister: stdinformers.Core().V1().ConfigMaps().Lister(),
	}
	c.ExecuteExternalActions(namespace, cache)
	stdactions := filterStdInformerActions(stdclient.Actions())
	assert.Equal(t, 1, len(stdactions))
	assert.Equal(t, "create", stdactions[0].GetVerb())
	createAction, ok := stdactions[0].(core.CreateAction)
	assert.Assert(t, ok)
	actualCm := createAction.GetObject()
	expectedCm := testutil.ConfigMapWithData(testutil.DefaultConfigMapName)
	if !reflect.DeepEqual(&expectedCm, actualCm) {
		t.Errorf("Expected\n\t%+v\ngot\n\t%+v", &expectedCm, actualCm)
		t.Fail()
	}
}

func TestExistingConfigMapWithExternalAction(t *testing.T) {
	configmapType := "configmap"
	namespace := testutil.DefaultNamespace
	name := testutil.DefaultConfigMapName
	key := ExternalActionKeyFunc(configmapType, name)

	action1 := ExternalAction{Key: "action1key", Value: "action1value"}
	action2 := ExternalAction{Key: "action2key", Value: "action2value"}
	cache := NewExternalActionCache()
	cache.Put(key, action1)
	cache.Put(key, action2)

	existingCm := testutil.ConfigMapWithData(testutil.DefaultConfigMapName)
	stdobjects := []runtime.Object{&existingCm}
	stdclient := stdfake.NewSimpleClientset(stdobjects...)
	stdinformers := stdinformers.NewSharedInformerFactory(stdclient, 0)
	stdinformers.Core().V1().ConfigMaps().Informer().GetIndexer().Add(&existingCm)

	c := Controller{
		stdclient:       stdclient,
		configmaplister: stdinformers.Core().V1().ConfigMaps().Lister(),
	}

	stopCh := make(chan struct{})
	defer close(stopCh)
	stdinformers.Start(stopCh)

	c.ExecuteExternalActions(namespace, cache)
	stdactions := filterStdInformerActions(stdclient.Actions())
	assert.Equal(t, 1, len(stdactions))
	assert.Equal(t, "update", stdactions[0].GetVerb())
	createAction, ok := stdactions[0].(core.CreateAction)
	assert.Assert(t, ok)
	actualCm := createAction.GetObject()
	expectedCm := testutil.ConfigMapWithData(testutil.DefaultConfigMapName)
	expectedCm.Data["action1key"] = "action1value"
	expectedCm.Data["action2key"] = "action2value"
	if !reflect.DeepEqual(&expectedCm, actualCm) {
		t.Errorf("Expected\n\t%+v\ngot\n\t%+v", &expectedCm, actualCm)
		t.Fail()
	}
}

func TestNewSecretFromExternalAction(t *testing.T) {
	secretType := "secret"
	namespace := testutil.DefaultNamespace
	name := testutil.DefaultSecretName
	key := ExternalActionKeyFunc(secretType, name)

	action := ExternalAction{Key: testutil.DefaultReferenceKey, Value: testutil.DefaultValue}
	cache := NewExternalActionCache()
	cache.Put(key, action)

	stdclient := stdfake.NewSimpleClientset()
	stdinformers := stdinformers.NewSharedInformerFactory(stdclient, 0)

	c := Controller{
		stdclient:    stdclient,
		secretlister: stdinformers.Core().V1().Secrets().Lister(),
	}
	c.ExecuteExternalActions(namespace, cache)
	stdactions := filterStdInformerActions(stdclient.Actions())
	assert.Equal(t, 1, len(stdactions))
	assert.Equal(t, "create", stdactions[0].GetVerb())
	createAction, ok := stdactions[0].(core.CreateAction)
	assert.Assert(t, ok)
	actualSecret := createAction.GetObject()
	expectedSecret := testutil.SecretWithData(testutil.DefaultSecretName)
	if !reflect.DeepEqual(&expectedSecret, actualSecret) {
		t.Errorf("Expected\n\t%+v\ngot\n\t%+v", &expectedSecret, actualSecret)
		t.Fail()
	}
}

func TestExistingSecretWithExternalAction(t *testing.T) {
	secretType := "secret"
	namespace := testutil.DefaultNamespace
	name := testutil.DefaultSecretName
	key := ExternalActionKeyFunc(secretType, name)

	action1 := ExternalAction{Key: "action1key", Value: "action1value"}
	action2 := ExternalAction{Key: "action2key", Value: "action2value"}
	cache := NewExternalActionCache()
	cache.Put(key, action1)
	cache.Put(key, action2)

	existingSecret := testutil.SecretWithData(testutil.DefaultSecretName)
	stdobjects := []runtime.Object{&existingSecret}
	stdclient := stdfake.NewSimpleClientset(stdobjects...)
	stdinformers := stdinformers.NewSharedInformerFactory(stdclient, 0)
	stdinformers.Core().V1().Secrets().Informer().GetIndexer().Add(&existingSecret)

	c := Controller{
		stdclient:    stdclient,
		secretlister: stdinformers.Core().V1().Secrets().Lister(),
	}

	stopCh := make(chan struct{})
	defer close(stopCh)
	stdinformers.Start(stopCh)

	c.ExecuteExternalActions(namespace, cache)
	stdactions := filterStdInformerActions(stdclient.Actions())
	assert.Equal(t, 1, len(stdactions))
	assert.Equal(t, "update", stdactions[0].GetVerb())
	createAction, ok := stdactions[0].(core.CreateAction)
	assert.Assert(t, ok)
	actualSecret := createAction.GetObject()
	expectedSecret := testutil.SecretWithData(testutil.DefaultSecretName)
	expectedSecret.Data["action1key"] = []byte("action1value")
	expectedSecret.Data["action2key"] = []byte("action2value")
	if !reflect.DeepEqual(&expectedSecret, actualSecret) {
		t.Errorf("Expected\n\t%+v\ngot\n\t%+v", &expectedSecret, actualSecret)
		t.Fail()
	}
}

package knativefakes


import (
	fakeautoscalingv1alpha1 "github.com/att-cloudnative-labs/test/knativefakes/typed/autoscaling/v1alpha1/fake"
	fakenetworkingv1alpha1 "github.com/att-cloudnative-labs/test/knativefakes/typed/networking/v1alpha1/fake"
	fakeservingv1alpha1 "github.com/att-cloudnative-labs/test/knativefakes/typed/serving/v1alpha1/fake"
	servingscheme "github.com/knative/serving/pkg/apis/serving/v1alpha1"
	knclientset "github.com/knative/serving/pkg/client/clientset/versioned"
	autoscalingv1alpha1 "github.com/knative/serving/pkg/client/clientset/versioned/typed/autoscaling/v1alpha1"
	networkingv1alpha1 "github.com/knative/serving/pkg/client/clientset/versioned/typed/networking/v1alpha1"
	servingv1alpha1 "github.com/knative/serving/pkg/client/clientset/versioned/typed/serving/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/testing"
)

var scheme = runtime.NewScheme()
var codecs = serializer.NewCodecFactory(scheme)
var localSchemeBuilder = runtime.SchemeBuilder{
	servingscheme.AddToScheme,
}

var AddToScheme = localSchemeBuilder.AddToScheme

func init() {
	v1.AddToGroupVersion(scheme, schema.GroupVersion{Version: "v1"})
	utilruntime.Must(AddToScheme(scheme))
}


type Clientset struct {
	testing.Fake
	discovery *fakediscovery.FakeDiscovery
}



func NewSimpleClientset(objects ...runtime.Object) *Clientset {
	o := testing.NewObjectTracker(scheme, codecs.UniversalDecoder())
	for _, obj := range objects {
		if err := o.Add(obj); err != nil {
			panic(err)
		}
	}

	cs := &Clientset{}
	cs.discovery = &fakediscovery.FakeDiscovery{Fake: &cs.Fake}
	cs.AddReactor("*", "*", testing.ObjectReaction(o))
	cs.AddWatchReactor("*", func(action testing.Action) (handled bool, ret watch.Interface, err error) {
		gvr := action.GetResource()
		ns := action.GetNamespace()
		watch, err := o.Watch(gvr, ns)
		if err != nil {
			return false, nil, err
		}
		return true, watch, nil
	})

	return cs
}

func (c *Clientset) Discovery() discovery.DiscoveryInterface {
	return c.discovery
}

var _ knclientset.Interface = &Clientset{}

// AutoscalingV1alpha1 retrieves the AutoscalingV1alpha1Client
func (c *Clientset) AutoscalingV1alpha1() autoscalingv1alpha1.AutoscalingV1alpha1Interface {
	return &fakeautoscalingv1alpha1.FakeAutoscalingV1alpha1{Fake: &c.Fake}
}

// Autoscaling retrieves the AutoscalingV1alpha1Client
func (c *Clientset) Autoscaling() autoscalingv1alpha1.AutoscalingV1alpha1Interface {
	return &fakeautoscalingv1alpha1.FakeAutoscalingV1alpha1{Fake: &c.Fake}
}

// NetworkingV1alpha1 retrieves the NetworkingV1alpha1Client
func (c *Clientset) NetworkingV1alpha1() networkingv1alpha1.NetworkingV1alpha1Interface {
	return &fakenetworkingv1alpha1.FakeNetworkingV1alpha1{Fake: &c.Fake}
}

// Networking retrieves the NetworkingV1alpha1Client
func (c *Clientset) Networking() networkingv1alpha1.NetworkingV1alpha1Interface {
	return &fakenetworkingv1alpha1.FakeNetworkingV1alpha1{Fake: &c.Fake}
}

// ServingV1alpha1 retrieves the ServingV1alpha1Client
func (c *Clientset) ServingV1alpha1() servingv1alpha1.ServingV1alpha1Interface {
	return &fakeservingv1alpha1.FakeServingV1alpha1{Fake: &c.Fake}
}

// Serving retrieves the ServingV1alpha1Client
func (c *Clientset) Serving() servingv1alpha1.ServingV1alpha1Interface {
	return &fakeservingv1alpha1.FakeServingV1alpha1{Fake: &c.Fake}
}



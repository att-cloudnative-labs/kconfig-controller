/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/att-cloudnative-labs/kconfig-controller/api/v1beta1"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	InjectConfigAnnotation       = "kconfigcontroller.atteg.com/inject"
	ExclusiveEnvConfigAnnotation = "kconfigcontroller.atteg.com/exclusive-env"
)

// +kubebuilder:webhook:path=/mutate-v1-pod,mutating=true,failurePolicy=ignore,groups="",resources=pods,verbs=create,versions=v1,name=config-injector.kconfigcontroller.aeg.cloud

type PodConfigInjector struct {
	Client  client.Client
	decoder *admission.Decoder
	Log     logr.Logger
}

func (r *PodConfigInjector) InjectDecoder(d *admission.Decoder) error {
	r.decoder = d
	return nil
}

func (r *PodConfigInjector) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &v1.Pod{}

	err := r.decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	// only inject into pods with proper annotation
	if pod.Annotations == nil || strings.ToLower(pod.Annotations[InjectConfigAnnotation]) != "true" {
		return admission.Allowed("kconfig inject not indicated")
	}
	// get bindings that select this rs
	kcbs := v1beta1.KconfigBindingList{}
	if err := r.Client.List(ctx, &kcbs, client.InNamespace(req.Namespace)); err != nil {
		return admission.Errored(http.StatusInternalServerError, fmt.Errorf("could not get kconfigbininglist: %s", err.Error()))
	}

	// cleanup old pod env configs
	if pod.Annotations == nil || strings.ToLower(pod.Annotations[ExclusiveEnvConfigAnnotation]) == "true" {
		pod.Spec.Containers[0].Env = []v1.EnvVar{}
	}

	selecting := make([]v1beta1.KconfigBindingSpec, 0)
	for _, kcb := range kcbs.Items {
		ls, err := v12.LabelSelectorAsSelector(&kcb.Spec.Selector)
		if err != nil {
			r.Log.Error(err, fmt.Sprintf("couldn't get selector of kcb: %s", err.Error()))
			continue
		}

		if ls.Matches(labels.Set(pod.Labels)) {
			selecting = append(selecting, kcb.Spec)
		}
	}
	// sort by level
	sort.Sort(ByLevel(selecting))
	// add each to pod
	for _, kcb := range selecting {
		if pod.Spec.Containers[0].Env == nil {
			pod.Spec.Containers[0].Env = make([]v1.EnvVar, 0)
		}
		pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, kcb.Envs...)
	}
	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, fmt.Errorf("could not marshal pod json: %s", err.Error()))
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

// ByLevel sort function for sorting array of KconfigBindingSpecs by their level
type ByLevel []v1beta1.KconfigBindingSpec

func (c ByLevel) Len() int           { return len(c) }
func (c ByLevel) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }
func (c ByLevel) Less(i, j int) bool { return c[i].Level < c[j].Level }

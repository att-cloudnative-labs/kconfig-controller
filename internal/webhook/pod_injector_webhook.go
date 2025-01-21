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

package webhook

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sort"
	"strings"

	"github.com/att-cloudnative-labs/kconfig-controller/api/v1beta1"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var podConfigInjectorLog = logf.Log.WithName("pod-config-injector")

func SetupPodConfigInjectorWithManager(mgr ctrl.Manager, sel *v12.LabelSelector) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&v1.Pod{}).
		WithDefaulter(
			&PodConfigInjector{
				Client:                   mgr.GetClient(),
				DefaultContainerSelector: sel,
			},
		).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-v1-pod,mutating=true,failurePolicy=ignore,sideEffects=None,groups="",resources=pods,verbs=create,versions=v1,name=config-injector.kconfigcontroller.aeg.cloud,admissionReviewVersions=v1

type PodConfigInjector struct {
	Client                   client.Client
	DefaultContainerSelector *v12.LabelSelector
}

const (
	InjectConfigAnnotation       = "kconfigcontroller.atteg.com/inject"
	ExclusiveEnvConfigAnnotation = "kconfigcontroller.atteg.com/exclusive-env"
)

var _ webhook.CustomDefaulter = &PodConfigInjector{}

func (r *PodConfigInjector) Default(ctx context.Context, obj runtime.Object) error {
	pod, ok := obj.(*v1.Pod)
	if !ok {
		return fmt.Errorf("expected an Pod object but got %T", obj)
	}

	// only inject into pods with proper annotation
	if pod.Annotations == nil || strings.ToLower(pod.Annotations[InjectConfigAnnotation]) != "true" {
		podConfigInjectorLog.Info(fmt.Sprintf("skipping %s - not annotated", pod.Name))
		return nil
	}
	// get bindings that select this rs
	kcbs := v1beta1.KconfigBindingList{}
	if err := r.Client.List(ctx, &kcbs, client.InNamespace(pod.Namespace)); err != nil {
		return fmt.Errorf("could not get kconfigbininglist: %s", err.Error())
	}

	// cleanup old pod env configs
	if pod.Annotations == nil || strings.ToLower(pod.Annotations[ExclusiveEnvConfigAnnotation]) == "true" {
		pod.Spec.Containers[0].Env = []v1.EnvVar{}
	}

	selecting := make([]v1beta1.KconfigBindingSpec, 0)
	for _, kcb := range kcbs.Items {
		ls, err := v12.LabelSelectorAsSelector(&kcb.Spec.Selector)
		if err != nil {
			podConfigInjectorLog.Error(err, fmt.Sprintf("couldn't get selector of kcb: %s", err.Error()))
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
		for i, container := range pod.Spec.Containers {
			labelsForContainer := labels.Set{"name": container.Name}
			labelSelector := kcb.ContainerSelector
			if labelSelector == nil {
				labelSelector = r.DefaultContainerSelector
			}
			selector, err := v12.LabelSelectorAsSelector(labelSelector)
			if err != nil {
				podConfigInjectorLog.Error(err, fmt.Sprintf("error reading kcb containerSelector: %s", err.Error()))
				continue
			}
			if selector.Matches(labelsForContainer) {
				if pod.Spec.Containers[i].Env == nil {
					pod.Spec.Containers[i].Env = make([]v1.EnvVar, 0)
				}
				pod.Spec.Containers[i].Env = append(pod.Spec.Containers[0].Env, kcb.Envs...)
			}
		}
	}
	return nil
}

// ByLevel sort function for sorting array of KconfigBindingSpecs by their level
type ByLevel []v1beta1.KconfigBindingSpec

func (c ByLevel) Len() int           { return len(c) }
func (c ByLevel) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }
func (c ByLevel) Less(i, j int) bool { return c[i].Level < c[j].Level }

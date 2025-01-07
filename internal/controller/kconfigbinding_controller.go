/*
Copyright 2025.

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

package controller

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"strconv"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kconfigcontrollerv1beta1 "github.com/att-cloudnative-labs/kconfig-controller/api/v1beta1"
)

// KconfigBindingReconciler reconciles a KconfigBinding object
type KconfigBindingReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=kconfigcontroller.atteg.com,resources=kconfigbindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kconfigcontroller.atteg.com,resources=kconfigbindings/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kconfigcontroller.atteg.com,resources=kconfigbindings/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments;statefulsets,verbs=get;list;watch;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the KconfigBinding object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *KconfigBindingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("kconfigbinding", req.NamespacedName)

	// your logic here
	var kcb kconfigcontrollerv1beta1.KconfigBinding
	if err := r.Get(ctx, req.NamespacedName, &kcb); err != nil {
		// Not Found is disregarded and ends reconciliation
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	disableTemplateRefresh := "false"
	if val, ok := kcb.Annotations[KconfigDisableTemplateRefresh]; ok {
		disableTemplateRefresh = val
	}
	if kcb.Status.ObservedGeneration != kcb.Generation && disableTemplateRefresh != "true" {
		err := r.processKconfigBinding(ctx, kcb)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("error processing kconfigBinding: %s", err.Error())
		}
		//kcbCopy := kcb.DeepCopy()
		kcb.Status.ObservedGeneration = kcb.Generation
		if err := r.Status().Update(ctx, &kcb); err != nil {
			return ctrl.Result{}, fmt.Errorf("error updating kconfigBinding status: %s", err.Error())
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KconfigBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kconfigcontrollerv1beta1.KconfigBinding{}).
		Named("kconfigbinding").
		Complete(r)
}

func (r *KconfigBindingReconciler) processKconfigBinding(ctx context.Context, kcb kconfigcontrollerv1beta1.KconfigBinding) error {
	if err := r.updateDeployments(ctx, kcb); err != nil {
		return fmt.Errorf("error updating deployments: %s", err.Error())
	}
	if err := r.updateStatefulSets(ctx, kcb); err != nil {
		return fmt.Errorf("error updating statefulsets: %s", err.Error())
	}
	return nil
}

func (r *KconfigBindingReconciler) updateDeployments(ctx context.Context, kcb kconfigcontrollerv1beta1.KconfigBinding) error {
	var deploymentsList v1.DeploymentList
	if err := r.List(ctx, &deploymentsList, client.InNamespace(kcb.Namespace)); err != nil {
		return fmt.Errorf("error getting deploymentList: %s", err.Error())
	}
	selector, err := v12.LabelSelectorAsSelector(&kcb.Spec.Selector)
	if err != nil {
		return fmt.Errorf("couldn't get selector of kcb: %s", err.Error())
	}
	for _, deployment := range deploymentsList.Items {
		if deployment.Annotations == nil || deployment.Annotations[AllowTemplateUpdatesAnnotation] != "true" {
			continue
		}
		if !selector.Matches(labels.Set(deployment.Spec.Template.Labels)) {
			continue
		}
		deploymentCopy := deployment.DeepCopy()
		if deploymentCopy.Spec.Template.Annotations == nil {
			deploymentCopy.Spec.Template.Annotations = make(map[string]string)
		}
		generationAnnotation := fmt.Sprintf("%s%s-%s", GenerationAnnotationPrefix, kcb.Name, "generation")
		kcbGenerationString := strconv.FormatInt(kcb.Generation, 10)
		deploymentCopy.Spec.Template.Annotations[generationAnnotation] = kcbGenerationString
		if err := r.Update(ctx, deploymentCopy); err != nil {
			return fmt.Errorf("error updating deployment: %s", err.Error())
		}
	}
	return nil
}

func (r *KconfigBindingReconciler) updateStatefulSets(ctx context.Context, kcb kconfigcontrollerv1beta1.KconfigBinding) error {
	var statefulSetList v1.StatefulSetList
	if err := r.List(ctx, &statefulSetList, client.InNamespace(kcb.Namespace)); err != nil {
		return fmt.Errorf("error getting statefulSetList: %s", err.Error())
	}
	selector, err := v12.LabelSelectorAsSelector(&kcb.Spec.Selector)
	if err != nil {
		return fmt.Errorf("couldn't get selector of kcb: %s", err.Error())
	}
	for _, statefulSet := range statefulSetList.Items {
		if statefulSet.Annotations == nil || statefulSet.Annotations[AllowTemplateUpdatesAnnotation] != "true" {
			continue
		}
		if !selector.Matches(labels.Set(statefulSet.Spec.Template.Labels)) {
			continue
		}
		statefulSetCopy := statefulSet.DeepCopy()
		if statefulSetCopy.Spec.Template.Annotations == nil {
			statefulSetCopy.Spec.Template.Annotations = make(map[string]string)
		}
		generationAnnotation := fmt.Sprintf("%s%s", GenerationAnnotationPrefix, kcb.Name)
		statefulSetCopy.Spec.Template.Annotations[generationAnnotation] = fmt.Sprint(kcb.Generation)
		if err := r.Update(ctx, statefulSetCopy); err != nil {
			return fmt.Errorf("error updating statefulSet: %s", err.Error())
		}
	}
	return nil
}

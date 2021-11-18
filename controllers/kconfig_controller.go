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

package controllers

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/google/uuid"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"

	kconfigcontrollerv1beta1 "github.com/att-cloudnative-labs/kconfig-controller/api/v1beta1"
)

// KconfigReconciler reconciles a Kconfig object
type KconfigReconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	Recorder        record.EventRecorder
	ConfigMapPrefix string
	SecretPrefix    string
}

// +kubebuilder:rbac:groups=kconfigcontroller.atteg.com,resources=kconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kconfigcontroller.atteg.com,resources=kconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=configmaps;secrets,verbs=get;list;watch;create;update;patch;delete

func (r *KconfigReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	_ = r.Log.WithValues("kconfig", req.NamespacedName)

	var kc kconfigcontrollerv1beta1.Kconfig
	if err := r.Get(ctx, req.NamespacedName, &kc); err != nil {
		// Not Found is disregarded and ends reconciliation
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	return ctrl.Result{}, r.processKconfig(ctx, &kc)
}

func (r *KconfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kconfigcontrollerv1beta1.Kconfig{}).
		Complete(r)
}

// ExternalAction represents update to external resource (e.g. configMap or Secret)
type ExternalAction struct {
	Type  string
	Key   string
	Value string
}

func (r *KconfigReconciler) processKconfig(ctx context.Context, kc *kconfigcontrollerv1beta1.Kconfig) error {
	r.Log.WithValues()
	updatedEnvConfigs := make([]kconfigcontrollerv1beta1.EnvConfig, 0)
	envVars := make([]v1.EnvVar, 0)
	cmActions := make([]ExternalAction, 0)
	secActions := make([]ExternalAction, 0)

	envConfigs := kc.Spec.EnvConfigs
	for _, ec := range envConfigs {
		switch strings.ToLower(ec.Type) {
		case "value", "": // value is default type
			if err := r.processValueEnvConfig(ec, &envVars, &updatedEnvConfigs); err != nil {
				return fmt.Errorf("error processing value envConfig: %s", err.Error())
			}
		case "configmap":
			if err := r.processConfigMapEnvConfig(kc, ec, &cmActions, &envVars, &updatedEnvConfigs); err != nil {
				return fmt.Errorf("error processing configmap envConfig: %s", err.Error())
			}
		case "secret":
			if err := r.processSecretEnvConfig(kc, ec, &secActions, &envVars, &updatedEnvConfigs); err != nil {
				return fmt.Errorf("error processing secret envConfig: %s", err.Error())
			}
		case "fieldref":
			if err := r.processFieldRefEnvConfig(ec, &envVars, &updatedEnvConfigs); err != nil {
				return fmt.Errorf("error processing fieldRef envConfig: %s", err.Error())
			}
		case "resourcefieldref":
			if err := r.processResourceFieldRefEnvConfig(ec, &envVars, &updatedEnvConfigs); err != nil {
				return fmt.Errorf("error processing resourceFieldRef envConfig: %s", err.Error())
			}
		default:
			return fmt.Errorf("invalid EnvConfig type, %s", ec.Type)
		}
	}

	if err := r.executeConfigMapActions(ctx, kc, cmActions); err != nil {
		return fmt.Errorf("error executing configmap actions: %s", err.Error())
	}
	if err := r.executeSecretActions(ctx, kc, secActions); err != nil {
		return fmt.Errorf("error executing secret actions: %s", err.Error())
	}
	if err := r.updateKconfigBinding(ctx, kc, envVars); err != nil {
		return fmt.Errorf("error on update of kconfigbinding: %s", err.Error())
	}
	// update kconfig
	kcCopy := kc.DeepCopy()
	kcCopy.Spec.EnvConfigs = updatedEnvConfigs

	if err := r.Update(ctx, kcCopy); err != nil {
		return fmt.Errorf("error updating kconfig: %s", err.Error())
	}
	return nil
}

func (r *KconfigReconciler) processValueEnvConfig(ec kconfigcontrollerv1beta1.EnvConfig, envVars *[]v1.EnvVar, updatedECs *[]kconfigcontrollerv1beta1.EnvConfig) error {
	if ec.Key == "" || ec.Value == nil {
		r.Recorder.Event(&kconfigcontrollerv1beta1.Kconfig{}, WarningEventType, InvalidEnvConfigEvent, "Either key or value is empty for value type EnvConfig. This entry will be removed")
		return nil
	}
	*envVars = append(*envVars, v1.EnvVar{Name: ec.Key, Value: *ec.Value})
	*updatedECs = append(*updatedECs, kconfigcontrollerv1beta1.EnvConfig{Type: ValueEnvConfigType, Key: ec.Key, Value: ec.Value})
	return nil
}

func (r *KconfigReconciler) processConfigMapEnvConfig(kc *kconfigcontrollerv1beta1.Kconfig, ec kconfigcontrollerv1beta1.EnvConfig, actions *[]ExternalAction, envVars *[]v1.EnvVar, updatedECs *[]kconfigcontrollerv1beta1.EnvConfig) error {
	envVar := v1.EnvVar{}
	if ec.Value != nil {
		refName := fmt.Sprintf("%s%s", r.ConfigMapPrefix, kc.Name)
		refKey := uuid.New().String()
		configMapKeyRef := &v1.ConfigMapKeySelector{
			LocalObjectReference: v1.LocalObjectReference{
				Name: refName,
			},
			Key: refKey,
		}

		envVar.Name = ec.Key
		envVar.ValueFrom = &v1.EnvVarSource{
			ConfigMapKeyRef: configMapKeyRef,
		}

		*actions = append(*actions, ExternalAction{Key: refKey, Value: *ec.Value})
		*envVars = append(*envVars, envVar)
		*updatedECs = append(*updatedECs, kconfigcontrollerv1beta1.EnvConfig{
			Type:            ConfigMapEnvConfigType,
			Key:             ec.Key,
			ConfigMapKeyRef: configMapKeyRef,
		})
		return nil
	}
	*envVars = append(*envVars, v1.EnvVar{
		Name:  ec.Key,
		Value: "",
		ValueFrom: &v1.EnvVarSource{
			ConfigMapKeyRef: ec.ConfigMapKeyRef,
		},
	})
	*updatedECs = append(*updatedECs, *ec.DeepCopy())
	return nil
}

func (r *KconfigReconciler) processSecretEnvConfig(kc *kconfigcontrollerv1beta1.Kconfig, ec kconfigcontrollerv1beta1.EnvConfig, actions *[]ExternalAction, envVars *[]v1.EnvVar, updatedECs *[]kconfigcontrollerv1beta1.EnvConfig) error {
	envVar := v1.EnvVar{}
	if ec.Value != nil {
		refName := fmt.Sprintf("%s%s", r.SecretPrefix, kc.Name)
		refKey := uuid.New().String()
		secretKeyRef := &v1.SecretKeySelector{
			LocalObjectReference: v1.LocalObjectReference{
				Name: refName,
			},
			Key: refKey,
		}

		envVar.Name = ec.Key
		envVar.ValueFrom = &v1.EnvVarSource{
			SecretKeyRef: secretKeyRef,
		}
		*actions = append(*actions, ExternalAction{Key: refKey, Value: *ec.Value})
		*envVars = append(*envVars, envVar)
		*updatedECs = append(*updatedECs, kconfigcontrollerv1beta1.EnvConfig{
			Type:         SecretEnvConfigType,
			Key:          ec.Key,
			SecretKeyRef: secretKeyRef,
		})
		return nil
	}
	*envVars = append(*envVars, v1.EnvVar{
		Name: ec.Key,
		ValueFrom: &v1.EnvVarSource{
			SecretKeyRef: ec.SecretKeyRef,
		},
	})
	*updatedECs = append(*updatedECs, *ec.DeepCopy())
	return nil
}

func (r *KconfigReconciler) processFieldRefEnvConfig(ec kconfigcontrollerv1beta1.EnvConfig, envVars *[]v1.EnvVar, updatedECs *[]kconfigcontrollerv1beta1.EnvConfig) error {
	if ec.Value != nil {
		envVar := v1.EnvVar{
			Name: ec.Key,
			ValueFrom: &v1.EnvVarSource{
				FieldRef: &v1.ObjectFieldSelector{
					FieldPath: *ec.Value,
				},
			},
		}
		*envVars = append(*envVars, envVar)
		*updatedECs = append(*updatedECs, kconfigcontrollerv1beta1.EnvConfig{
			Type: FieldRefEnvConfigType,
			Key:  ec.Key,
			FieldRef: &v1.ObjectFieldSelector{
				FieldPath: *ec.Value,
			},
		})
		return nil
	}
	*envVars = append(*envVars, v1.EnvVar{
		Name: ec.Key,
		ValueFrom: &v1.EnvVarSource{
			FieldRef: ec.FieldRef,
		},
	})
	*updatedECs = append(*updatedECs, *ec.DeepCopy())
	return nil
}

func (r *KconfigReconciler) processResourceFieldRefEnvConfig(ec kconfigcontrollerv1beta1.EnvConfig, envVars *[]v1.EnvVar, updatedECs *[]kconfigcontrollerv1beta1.EnvConfig) error {
	if ec.Value != nil {
		envVar := v1.EnvVar{
			Name: ec.Key,
			ValueFrom: &v1.EnvVarSource{
				FieldRef: &v1.ObjectFieldSelector{
					FieldPath: *ec.Value,
				},
			},
		}
		*envVars = append(*envVars, envVar)
		*updatedECs = append(*updatedECs, kconfigcontrollerv1beta1.EnvConfig{
			Type: ResourceFieldRefEnvConfigType,
			Key:  ec.Key,
			ResourceFieldRef: &v1.ResourceFieldSelector{
				Resource: *ec.Value,
			},
		})
		return nil
	}
	*envVars = append(*envVars, v1.EnvVar{
		Name: ec.Key,
		ValueFrom: &v1.EnvVarSource{
			ResourceFieldRef: ec.ResourceFieldRef,
		},
	})
	*updatedECs = append(*updatedECs, *ec.DeepCopy())
	return nil
}

func (r *KconfigReconciler) executeConfigMapActions(ctx context.Context, kc *kconfigcontrollerv1beta1.Kconfig, actions []ExternalAction) error {
	if len(actions) == 0 {
		return nil
	}
	var cm v1.ConfigMap
	cmName := fmt.Sprintf("%s%s", r.ConfigMapPrefix, kc.Name)
	nn := types.NamespacedName{Namespace: kc.Namespace, Name: cmName}
	existing := true
	if err := r.Get(ctx, nn, &cm); err != nil {
		if errors.IsNotFound(err) {
			existing = false
			cm = v1.ConfigMap{
				ObjectMeta: ctrl.ObjectMeta{
					Namespace: kc.Namespace,
					Name:      cmName,
				},
				Data: make(map[string]string),
			}
		} else {
			return fmt.Errorf("error getting configmap: %s", err.Error())
		}
	}
	for _, action := range actions {
		cm.Data[action.Key] = action.Value
	}

	if existing {
		if err := r.Update(ctx, &cm); err != nil {
			return fmt.Errorf("error updating configmap: %s", err.Error())
		}
	} else {
		if err := r.Create(ctx, &cm); err != nil {
			return fmt.Errorf("error creating configmap: %s", err.Error())
		}
	}
	return nil
}

func (r *KconfigReconciler) executeSecretActions(ctx context.Context, kc *kconfigcontrollerv1beta1.Kconfig, actions []ExternalAction) error {
	if len(actions) == 0 {
		return nil
	}
	var sec v1.Secret
	secName := fmt.Sprintf("%s%s", r.SecretPrefix, kc.Name)
	nn := types.NamespacedName{Namespace: kc.Namespace, Name: secName}
	existing := true
	if err := r.Get(ctx, nn, &sec); err != nil {
		if errors.IsNotFound(err) {
			existing = false
			sec = v1.Secret{
				ObjectMeta: ctrl.ObjectMeta{
					Namespace: kc.Namespace,
					Name:      secName,
				},
				Data: make(map[string][]byte),
			}
		} else {
			return fmt.Errorf("error getting secret: %s", err.Error())
		}
	}
	for _, action := range actions {
		sec.Data[action.Key] = []byte(action.Value)
	}

	if existing {
		if err := r.Update(ctx, &sec); err != nil {
			return fmt.Errorf("error updating secret: %s", err.Error())
		}
	} else {
		if err := r.Create(ctx, &sec); err != nil {
			return fmt.Errorf("error creating secret: %s", err.Error())
		}
	}
	return nil
}

func (r *KconfigReconciler) updateKconfigBinding(ctx context.Context, kc *kconfigcontrollerv1beta1.Kconfig, envVars []v1.EnvVar) error {
	var kcb kconfigcontrollerv1beta1.KconfigBinding
	nn := types.NamespacedName{Namespace: kc.Namespace, Name: kc.Name}
	existing := true
	if err := r.Get(ctx, nn, &kcb); err != nil {
		if errors.IsNotFound(err) {
			existing = false
			kcb = kconfigcontrollerv1beta1.KconfigBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:   kc.Namespace,
					Name:        kc.Name,
					Labels:      kc.Labels,
					Annotations: kc.Annotations,
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind:       kc.Kind,
							APIVersion: kc.APIVersion,
							Name:       kc.Name,
							UID:        kc.UID,
						},
					},
				},
				Spec: kconfigcontrollerv1beta1.KconfigBindingSpec{
					Level:             0,
					Envs:              make([]v1.EnvVar, 0),
					Selector:          metav1.LabelSelector{},
					ContainerSelector: kc.Spec.ContainerSelector,
				},
			}
		} else {
			return fmt.Errorf("error getting kconfigBinding: %s", err.Error())
		}
	}

	kcb.Spec.Level = kc.Spec.Level
	kcb.Spec.Envs = envVars
	kcb.Spec.Selector = kc.Spec.Selector

	if existing {
		if err := r.Update(ctx, &kcb); err != nil {
			return fmt.Errorf("error updating kconfigBinding: %s", err.Error())
		}
	} else {
		if err := r.Create(ctx, &kcb); err != nil {
			return fmt.Errorf("error creating kconfigBinding: %s", err.Error())
		}
	}
	return nil
}

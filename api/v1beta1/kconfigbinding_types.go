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

package v1beta1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KconfigBindingSpec defines the desired state of KconfigBinding
type KconfigBindingSpec struct {
	Level    int                  `json:"level"`
	Envs     []v1.EnvVar          `json:"envs"`
	Selector metav1.LabelSelector `json:"selector"`
}

// KconfigBindingStatus defines the observed state of KconfigBinding
type KconfigBindingStatus struct {
	ObservedGeneration int64 `json:"observedGeneration"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// KconfigBinding is the Schema for the kconfigbindings API
type KconfigBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KconfigBindingSpec   `json:"spec,omitempty"`
	Status KconfigBindingStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KconfigBindingList contains a list of KconfigBinding
type KconfigBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KconfigBinding `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KconfigBinding{}, &KconfigBindingList{})
}

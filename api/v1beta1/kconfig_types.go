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

package v1beta1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KconfigSpec defines the desired state of Kconfig.
type KconfigSpec struct {
	Level int `json:"level"`
	// +kubebuilder:validation:Optional
	Selector          metav1.LabelSelector  `json:"selector"`
	EnvConfigs        []EnvConfig           `json:"envConfigs"`
	ContainerSelector *metav1.LabelSelector `json:"containerSelector"`
}

// EnvConfig represents a single environment variable configuration
type EnvConfig struct {
	// Type should be immutable
	// +kubebuilder:validation:Optional
	Type string `json:"type"`
	Key  string `json:"key"`
	// +kubebuilder:validation:Optional
	Value *string `json:"value,omitempty"`
	// +kubebuilder:validation:Optional
	ConfigMapKeyRef *v1.ConfigMapKeySelector `json:"configMapKeyRef,omitempty"`
	// +kubebuilder:validation:Optional
	SecretKeyRef *v1.SecretKeySelector `json:"secretKeyRef,omitempty" protobuf:"bytes,4,opt,name=secretKeyRef"`
	// +kubebuilder:validation:Optional
	FieldRef *v1.ObjectFieldSelector `json:"fieldRef,omitempty" protobuf:"bytes,4,opt,name=fieldRef"`
	// +kubebuilder:validation:Optional
	ResourceFieldRef *v1.ResourceFieldSelector `json:"resourceFieldRef,omitempty" protobuf:"bytes,4,opt,name=resourceFieldRef"`
}

// KconfigStatus defines the observed state of Kconfig.
type KconfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Kconfig is the Schema for the kconfigs API.
type Kconfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KconfigSpec   `json:"spec,omitempty"`
	Status KconfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KconfigList contains a list of Kconfig.
type KconfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Kconfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Kconfig{}, &KconfigList{})
}

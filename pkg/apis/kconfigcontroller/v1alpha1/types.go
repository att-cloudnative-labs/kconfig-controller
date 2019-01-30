package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Kconfig Object that defines a set of external configuration settings
// for a deployment
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Kconfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec KconfigSpec `json:"spec" protobuf:"bytes,1,opt,name=spec"`
}

// KconfigSpec Spec field for Kconfig struct
type KconfigSpec struct {
	Level      int                  `json:"level"`
	Selector   metav1.LabelSelector `json:"selector" protobuf:"bytes,1,opt,name=selector"`
	EnvConfigs []EnvConfig          `json:"envConfigs"`
}

// EnvConfig represents a single environment variable configuration
type EnvConfig struct {
	// Type should be immutable
	Type            string                   `json:"type"`
	Key             string                   `json:"key"`
	Value           *string                  `json:"value,omitempty"`
	RefName         *string                  `json:"refName,omitempty"`
	RefKey          *string                  `json:"refKey,omitempty"`
	ConfigMapKeyRef *v1.ConfigMapKeySelector `json:"configMapKeyRef,omitempty"`
	SecretKeyRef    *v1.SecretKeySelector    `json:"secretKeyRef,omitempty" protobuf:"bytes,4,opt,name=secretKeyRef"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type KconfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Kconfig `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type KconfigBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec KconfigBindingSpec `json:"spec" protobuf:"bytes,1,opt,name=spec"`
}

type KconfigBindingSpec struct {
	KconfigEnvsMap map[string]KconfigEnvs `json:"kconfigEnvs"`
}

type KconfigEnvs struct {
	Level int         `json:"level"`
	Envs  []v1.EnvVar `json:"envs"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type KconfigBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []KconfigBinding `json:"items"`
}

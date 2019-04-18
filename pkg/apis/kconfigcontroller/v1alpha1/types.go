package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Kconfig Object that defines a set of configuration settings
// for deployments matching a set of labels
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Kconfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              KconfigSpec `json:"spec" protobuf:"bytes,1,opt,name=spec"`
}

// KconfigSpec Spec field for Kconfig struct
type KconfigSpec struct {
	Level          int                  `json:"level"`
	Selector       metav1.LabelSelector `json:"selector" protobuf:"bytes,1,opt,name=selector"`
	EnvConfigs     []EnvConfig          `json:"envConfigs"`
	EnvRefsVersion int64                `json:"envRefsVersion"`
}

// EnvConfig represents a single environment variable configuration
type EnvConfig struct {
	// Type should be immutable
	Type             string                    `json:"type"`
	Key              string                    `json:"key"`
	Value            *string                   `json:"value,omitempty"`
	RefName          *string                   `json:"refName,omitempty"`
	RefKey           *string                   `json:"refKey,omitempty"`
	ConfigMapKeyRef  *v1.ConfigMapKeySelector  `json:"configMapKeyRef,omitempty"`
	SecretKeyRef     *v1.SecretKeySelector     `json:"secretKeyRef,omitempty" protobuf:"bytes,4,opt,name=secretKeyRef"`
	FieldRef         *v1.ObjectFieldSelector   `json:"fieldRef,omitempty" protobuf:"bytes,4,opt,name=fieldRef"`
	ResourceFieldRef *v1.ResourceFieldSelector `json:"resourceFieldRef,omitempty" protobuf:"bytes,4,opt,name=resourceFieldRef"`
}

// KconfigList List of Kconfigs
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type KconfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Kconfig `json:"items"`
}

// TODO: this may not be needed anymore since deployments/statefulstes/knativeservices will have their own
// KconfigBinding Holds configuration for deployment from combined Kconfigs that select its labels
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type KconfigBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec KconfigBindingSpec `json:"spec" protobuf:"bytes,1,opt,name=spec"`
}

// KconfigBindingList List of KconfigBindings
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type KconfigBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []KconfigBinding `json:"items"`
}

// DeploymentBinding Holds configuration for deployment from combined Kconfigs that select its labels
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type DeploymentBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec KconfigBindingSpec `json:"spec" protobuf:"bytes,1,opt,name=spec"`
}

// DeploymentBindingList List of DeploymentBindings
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type DeploymentBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []DeploymentBinding `json:"items"`
}

// StatefulSetBinding Holds configuration for deployment from combined Kconfigs that select its labels
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type StatefulSetBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec KconfigBindingSpec `json:"spec" protobuf:"bytes,1,opt,name=spec"`
}

// StatefulSetBindingList List of StatefulSetBindings
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type StatefulSetBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []StatefulSetBinding `json:"items"`
}

// KnativeServiceBinding Holds configuration for deployment from combined Kconfigs that select its labels
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type KnativeServiceBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec KconfigBindingSpec `json:"spec" protobuf:"bytes,1,opt,name=spec"`
}

// KnativeServiceBindingList List of KnativeServiceBindings
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type KnativeServiceBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []KnativeServiceBinding `json:"items"`
}

// KconfigBindingSpec Spec for KconfigBinding
type KconfigBindingSpec struct {
	KconfigEnvsMap map[string]KconfigEnvs `json:"kconfigEnvs"`
}

// KconfigEnvs Environment variables from a specific Kconfig
type KconfigEnvs struct {
	Level          int         `json:"level"`
	EnvRefsVersion int64       `json:"envRefsVersion"`
	Envs           []v1.EnvVar `json:"envs"`
}

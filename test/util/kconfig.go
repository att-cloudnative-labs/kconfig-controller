package util

import (
	"github.com/gbraxton/kconfig/pkg/apis/kconfigcontroller/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Kconfig returns new Kconfig
func Kconfig() v1alpha1.Kconfig {
	return v1alpha1.Kconfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: DefaultNamespace,
			Name:      DefaultName,
		},
		Spec: v1alpha1.KconfigSpec{
			Level: DefaultLevel,
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					DefaultSelectorKey: DefaultSelectorValue,
				},
			},
			EnvConfigs: []v1alpha1.EnvConfig{},
		},
	}
}

// ValueKconfig returns Kconfig with single value envConfig
func ValueKconfig() v1alpha1.Kconfig {
	defaultValue := DefaultValue
	kconfig := Kconfig()
	kconfig.Spec.EnvConfigs = append(kconfig.Spec.EnvConfigs, v1alpha1.EnvConfig{
		Type:  ValueType,
		Key:   DefaultKey,
		Value: &defaultValue,
	})
	return kconfig
}

// NewValueKconfig returns Kconfig with single value envConfig
func NewValueKconfig() v1alpha1.Kconfig {
	defaultNewValue := DefaultNewValue
	kconfig := Kconfig()
	kconfig.Spec.EnvConfigs = append(kconfig.Spec.EnvConfigs, v1alpha1.EnvConfig{
		Type:  ValueType,
		Key:   DefaultKey,
		Value: &defaultNewValue,
	})
	return kconfig
}

// ConfigMapKconfig returns Kconfig with existing ConfigMap
func ConfigMapKconfig() v1alpha1.Kconfig {
	kconfig := Kconfig()
	optional := true
	kconfig.Spec.EnvConfigs = append(kconfig.Spec.EnvConfigs, v1alpha1.EnvConfig{
		Type: ConfigMapType,
		Key:  DefaultKey,
		ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: DefaultConfigMapName},
			Key:                  DefaultReferenceKey,
			Optional:             &optional,
		},
	})
	return kconfig
}

// AddConfigMapKconfig returns Kconfig with Add action and ConfigMap type
func AddConfigMapKconfig() v1alpha1.Kconfig {
	defaultValue := DefaultValue
	defaultConfigMapName := DefaultConfigMapName
	kconfig := Kconfig()
	kconfig.Spec.EnvConfigs = append(kconfig.Spec.EnvConfigs, v1alpha1.EnvConfig{
		Type:    ConfigMapType,
		Key:     DefaultKey,
		Value:   &defaultValue,
		RefName: &defaultConfigMapName,
	})
	return kconfig
}

// UpdateConfigMapKconfig returns Kconfig with Update action and ConfigMap type
func UpdateConfigMapKconfig() v1alpha1.Kconfig {
	defaultNewValue := DefaultNewValue
	kconfig := Kconfig()
	optional := true
	kconfig.Spec.EnvConfigs = append(kconfig.Spec.EnvConfigs, v1alpha1.EnvConfig{
		Type:  ConfigMapType,
		Key:   DefaultKey,
		Value: &defaultNewValue,
		ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: DefaultConfigMapName},
			Key:                  DefaultReferenceKey,
			Optional:             &optional,
		},
	})
	return kconfig
}

// DeleteConfigMapKconfig returns empty Kconfig. Holdover from when Kconfigs had embedded actions.
func DeleteConfigMapKconfig() v1alpha1.Kconfig {
	kconfig := Kconfig()
	return kconfig
}

// SecretKconfig returns Kconfig with existing Secret
func SecretKconfig() v1alpha1.Kconfig {
	kconfig := Kconfig()
	optional := true
	kconfig.Spec.EnvConfigs = append(kconfig.Spec.EnvConfigs, v1alpha1.EnvConfig{
		Type: SecretType,
		Key:  DefaultKey,
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: DefaultSecretName},
			Key:                  DefaultReferenceKey,
			Optional:             &optional,
		},
	})
	return kconfig
}

// AddSecretKconfig returns Kconfig with Add action and Secret type
func AddSecretKconfig() v1alpha1.Kconfig {
	defaultValue := DefaultValue
	defaultSecretName := DefaultSecretName
	kconfig := Kconfig()
	kconfig.Spec.EnvConfigs = append(kconfig.Spec.EnvConfigs, v1alpha1.EnvConfig{
		Type:    SecretType,
		Key:     DefaultKey,
		Value:   &defaultValue,
		RefName: &defaultSecretName,
	})
	return kconfig
}

// UpdateSecretKconfig returns Kconfig with Update action and Secret type
func UpdateSecretKconfig() v1alpha1.Kconfig {
	defaultNewValue := DefaultNewValue
	kconfig := Kconfig()
	optional := true
	kconfig.Spec.EnvConfigs = append(kconfig.Spec.EnvConfigs, v1alpha1.EnvConfig{
		Type:  SecretType,
		Key:   DefaultKey,
		Value: &defaultNewValue,
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: DefaultSecretName},
			Key:                  DefaultReferenceKey,
			Optional:             &optional,
		},
	})
	return kconfig
}

// DeleteSecretKconfig returns empty Kconfig. Holdover from when Kconfigs had embedded actions.
func DeleteSecretKconfig() v1alpha1.Kconfig {
	kconfig := Kconfig()
	return kconfig
}

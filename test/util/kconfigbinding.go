package util

import (
	"github.com/gbraxton/kconfig/pkg/apis/kconfigcontroller/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KconfigBinding returns new KconfigBinding
func KconfigBinding() v1alpha1.KconfigBinding {
	return v1alpha1.KconfigBinding{
		TypeMeta: metav1.TypeMeta{APIVersion: v1alpha1.SchemeGroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: DefaultNamespace,
			Name:      DefaultName,
			Labels: map[string]string{
				DefaultSelectorKey: DefaultSelectorValue,
			},
		},
		Spec: v1alpha1.KconfigBindingSpec{
			KconfigEnvsMap: map[string]v1alpha1.KconfigEnvs{},
		},
	}
}

// EmptyKconfigEnvsKconfigBinding returns kcb with empty kconfigEnvs
func EmptyKconfigEnvsKconfigBinding() v1alpha1.KconfigBinding {
	kconfigEnvs := v1alpha1.KconfigEnvs{
		Level: DefaultLevel,
		Envs:  []corev1.EnvVar{},
	}
	kcb := KconfigBinding()
	kcb.Spec.KconfigEnvsMap[DefaultKconfigEnvsKey] = kconfigEnvs
	return kcb
}

// ValueKconfigBinding returns KconfigBinding with Value envVar
func ValueKconfigBinding() v1alpha1.KconfigBinding {
	kconfigEnvs := v1alpha1.KconfigEnvs{
		Level: DefaultLevel,
		Envs: []corev1.EnvVar{
			corev1.EnvVar{
				Name:  DefaultKey,
				Value: DefaultValue,
			},
		},
	}
	kcb := KconfigBinding()
	kcb.Spec.KconfigEnvsMap[DefaultKconfigEnvsKey] = kconfigEnvs
	return kcb
}

// NewValueKconfigBinding returns KconfigBinding with Value envVar
func NewValueKconfigBinding() v1alpha1.KconfigBinding {
	kconfigEnvs := v1alpha1.KconfigEnvs{
		Level: DefaultLevel,
		Envs: []corev1.EnvVar{
			corev1.EnvVar{
				Name:  DefaultKey,
				Value: DefaultNewValue,
			},
		},
	}
	kcb := KconfigBinding()
	kcb.Spec.KconfigEnvsMap[DefaultKconfigEnvsKey] = kconfigEnvs
	return kcb
}

// ConfigMapKconfigBinding returns KconfigBinding with ConfigMap envVar
func ConfigMapKconfigBinding() v1alpha1.KconfigBinding {
	optional := true
	kconfigEnvs := v1alpha1.KconfigEnvs{
		Level: DefaultLevel,
		Envs: []corev1.EnvVar{
			corev1.EnvVar{
				Name: DefaultKey,
				ValueFrom: &corev1.EnvVarSource{
					ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: DefaultConfigMapName,
						},
						Key:      DefaultReferenceKey,
						Optional: &optional,
					},
				},
			},
		},
	}
	kcb := KconfigBinding()
	kcb.Spec.KconfigEnvsMap[DefaultKconfigEnvsKey] = kconfigEnvs
	return kcb
}

// SecretKconfigBinding returns KconfigBinding with Secret envVar
func SecretKconfigBinding() v1alpha1.KconfigBinding {
	optional := true
	kconfigEnvs := v1alpha1.KconfigEnvs{
		Level: DefaultLevel,
		Envs: []corev1.EnvVar{
			corev1.EnvVar{
				Name: DefaultKey,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: DefaultSecretName,
						},
						Key:      DefaultReferenceKey,
						Optional: &optional,
					},
				},
			},
		},
	}
	kcb := KconfigBinding()
	kcb.Spec.KconfigEnvsMap[DefaultKconfigEnvsKey] = kconfigEnvs
	return kcb
}

package util

import (
	"github.com/att-cloudnative-labs/kconfig-controller/pkg/apis/kconfigcontroller/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KnativeServiceBinding returns new KnativeServiceBinding
func KnativeServiceBinding() v1alpha1.KnativeServiceBinding {
	return v1alpha1.KnativeServiceBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind: "KnativeServiceBinding",
		},
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

// EmptyKconfigEnvsKnativeServiceBinding returns kcb with empty kconfigEnvs
func EmptyKconfigEnvsKnativeServiceBinding() v1alpha1.KnativeServiceBinding {
	kconfigEnvs := v1alpha1.KconfigEnvs{
		Level: DefaultLevel,
		Envs:  []corev1.EnvVar{},
	}
	kcb := KnativeServiceBinding()
	kcb.Spec.KconfigEnvsMap[DefaultKconfigEnvsKey] = kconfigEnvs
	return kcb
}

// ValueKnativeServiceBinding returns KnativeServiceBinding with Value envVar
func ValueKnativeServiceBinding() v1alpha1.KnativeServiceBinding {
	kconfigEnvs := v1alpha1.KconfigEnvs{
		Level: DefaultLevel,
		Envs: []corev1.EnvVar{
			{
				Name:  DefaultKey,
				Value: DefaultValue,
			},
		},
	}
	kcb := KnativeServiceBinding()
	kcb.Spec.KconfigEnvsMap[DefaultKconfigEnvsKey] = kconfigEnvs
	return kcb
}

// NewValueKnativeServiceBinding returns KnativeServiceBinding with Value envVar
func NewValueKnativeServiceBinding() v1alpha1.KnativeServiceBinding {
	kconfigEnvs := v1alpha1.KconfigEnvs{
		Level: DefaultLevel,
		Envs: []corev1.EnvVar{
			{
				Name:  DefaultKey,
				Value: DefaultNewValue,
			},
		},
	}
	kcb := KnativeServiceBinding()
	kcb.Spec.KconfigEnvsMap[DefaultKconfigEnvsKey] = kconfigEnvs
	return kcb
}

// ConfigMapKnativeServiceBinding returns KnativeServiceBinding with ConfigMap envVar
func ConfigMapKnativeServiceBinding(envRefsVersion int64, configMapName string) v1alpha1.KnativeServiceBinding {
	optional := true
	kconfigEnvs := v1alpha1.KconfigEnvs{
		Level:          DefaultLevel,
		EnvRefsVersion: envRefsVersion,
		Envs: []corev1.EnvVar{
			{
				Name: DefaultKey,
				ValueFrom: &corev1.EnvVarSource{
					ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: configMapName,
						},
						Key:      DefaultReferenceKey,
						Optional: &optional,
					},
				},
			},
		},
	}
	kcb := KnativeServiceBinding()
	kcb.Spec.KconfigEnvsMap[DefaultKconfigEnvsKey] = kconfigEnvs
	return kcb
}

// SecretKnativeServiceBinding returns KnativeServiceBinding with Secret envVar
func SecretKnativeServiceBinding(envRefsVersion int64, secretName string) v1alpha1.KnativeServiceBinding {
	optional := true
	kconfigEnvs := v1alpha1.KconfigEnvs{
		Level:          DefaultLevel,
		EnvRefsVersion: envRefsVersion,
		Envs: []corev1.EnvVar{
			{
				Name: DefaultKey,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: secretName,
						},
						Key:      DefaultReferenceKey,
						Optional: &optional,
					},
				},
			},
		},
	}
	kcb := KnativeServiceBinding()
	kcb.Spec.KconfigEnvsMap[DefaultKconfigEnvsKey] = kconfigEnvs
	return kcb
}

// FieldRefKnativeServiceBinding FieldRefKnativeServiceBinding
func FieldRefKnativeServiceBinding(envRefsVersion int64) v1alpha1.KnativeServiceBinding {
	kconfigEnvs := v1alpha1.KconfigEnvs{
		Level:          DefaultLevel,
		EnvRefsVersion: envRefsVersion,
		Envs: []corev1.EnvVar{
			{
				Name: DefaultKey,
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: DefaultFieldPath,
					},
				},
			},
		},
	}
	kcb := KnativeServiceBinding()
	kcb.Spec.KconfigEnvsMap[DefaultKconfigEnvsKey] = kconfigEnvs
	return kcb
}

// ResourceFieldRefKnativeServiceBinding ResourceFieldRefKnativeServiceBinding
func ResourceFieldRefKnativeServiceBinding(envRefsVersion int64) v1alpha1.KnativeServiceBinding {
	kconfigEnvs := v1alpha1.KconfigEnvs{
		Level:          DefaultLevel,
		EnvRefsVersion: envRefsVersion,
		Envs: []corev1.EnvVar{
			{
				Name: DefaultKey,
				ValueFrom: &corev1.EnvVarSource{
					ResourceFieldRef: &corev1.ResourceFieldSelector{
						Resource: DefaultResourceFieldRefResource,
					},
				},
			},
		},
	}
	kcb := KnativeServiceBinding()
	kcb.Spec.KconfigEnvsMap[DefaultKconfigEnvsKey] = kconfigEnvs
	return kcb
}

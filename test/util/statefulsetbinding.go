package util


import (
	"github.com/att-cloudnative-labs/kconfig-controller/pkg/apis/kconfigcontroller/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StatefulSetBinding returns new StatefulSetBinding
func StatefulSetBinding() v1alpha1.StatefulSetBinding {
	return v1alpha1.StatefulSetBinding{
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

// EmptyKconfigEnvsStatefulSetBinding returns kcb with empty kconfigEnvs
func EmptyKconfigEnvsStatefulSetBinding() v1alpha1.StatefulSetBinding {
	kconfigEnvs := v1alpha1.KconfigEnvs{
		Level: DefaultLevel,
		Envs:  []corev1.EnvVar{},
	}
	kcb := StatefulSetBinding()
	kcb.Spec.KconfigEnvsMap[DefaultKconfigEnvsKey] = kconfigEnvs
	return kcb
}

// ValueStatefulSetBinding returns StatefulSetBinding with Value envVar
func ValueStatefulSetBinding() v1alpha1.StatefulSetBinding {
	kconfigEnvs := v1alpha1.KconfigEnvs{
		Level: DefaultLevel,
		Envs: []corev1.EnvVar{
			corev1.EnvVar{
				Name:  DefaultKey,
				Value: DefaultValue,
			},
		},
	}
	kcb := StatefulSetBinding()
	kcb.Spec.KconfigEnvsMap[DefaultKconfigEnvsKey] = kconfigEnvs
	return kcb
}

// NewValueStatefulSetBinding returns StatefulSetBinding with Value envVar
func NewValueStatefulSetBinding() v1alpha1.StatefulSetBinding {
	kconfigEnvs := v1alpha1.KconfigEnvs{
		Level: DefaultLevel,
		Envs: []corev1.EnvVar{
			corev1.EnvVar{
				Name:  DefaultKey,
				Value: DefaultNewValue,
			},
		},
	}
	kcb := StatefulSetBinding()
	kcb.Spec.KconfigEnvsMap[DefaultKconfigEnvsKey] = kconfigEnvs
	return kcb
}

// ConfigMapStatefulSetBinding returns StatefulSetBinding with ConfigMap envVar
func ConfigMapStatefulSetBinding(envRefsVersion int64, configMapName string) v1alpha1.StatefulSetBinding {
	optional := true
	kconfigEnvs := v1alpha1.KconfigEnvs{
		Level:          DefaultLevel,
		EnvRefsVersion: envRefsVersion,
		Envs: []corev1.EnvVar{
			corev1.EnvVar{
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
	kcb := StatefulSetBinding()
	kcb.Spec.KconfigEnvsMap[DefaultKconfigEnvsKey] = kconfigEnvs
	return kcb
}

// SecretStatefulSetBinding returns StatefulSetBinding with Secret envVar
func SecretStatefulSetBinding(envRefsVersion int64, secretName string) v1alpha1.StatefulSetBinding {
	optional := true
	kconfigEnvs := v1alpha1.KconfigEnvs{
		Level:          DefaultLevel,
		EnvRefsVersion: envRefsVersion,
		Envs: []corev1.EnvVar{
			corev1.EnvVar{
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
	kcb := StatefulSetBinding()
	kcb.Spec.KconfigEnvsMap[DefaultKconfigEnvsKey] = kconfigEnvs
	return kcb
}

// FieldRefStatefulSetBinding FieldRefStatefulSetBinding
func FieldRefStatefulSetBinding(envRefsVersion int64) v1alpha1.StatefulSetBinding {
	kconfigEnvs := v1alpha1.KconfigEnvs{
		Level:          DefaultLevel,
		EnvRefsVersion: envRefsVersion,
		Envs: []corev1.EnvVar{
			corev1.EnvVar{
				Name: DefaultKey,
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: DefaultFieldPath,
					},
				},
			},
		},
	}
	kcb := StatefulSetBinding()
	kcb.Spec.KconfigEnvsMap[DefaultKconfigEnvsKey] = kconfigEnvs
	return kcb
}

// ResourceFieldRefStatefulSetBinding ResourceFieldRefStatefulSetBinding
func ResourceFieldRefStatefulSetBinding(envRefsVersion int64) v1alpha1.StatefulSetBinding {
	kconfigEnvs := v1alpha1.KconfigEnvs{
		Level:          DefaultLevel,
		EnvRefsVersion: envRefsVersion,
		Envs: []corev1.EnvVar{
			corev1.EnvVar{
				Name: DefaultKey,
				ValueFrom: &corev1.EnvVarSource{
					ResourceFieldRef: &corev1.ResourceFieldSelector{
						Resource: DefaultResourceFieldRefResource,
					},
				},
			},
		},
	}
	kcb := StatefulSetBinding()
	kcb.Spec.KconfigEnvsMap[DefaultKconfigEnvsKey] = kconfigEnvs
	return kcb
}

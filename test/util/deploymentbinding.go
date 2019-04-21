package util

import (
	"github.com/att-cloudnative-labs/kconfig-controller/pkg/apis/kconfigcontroller/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeploymentBinding returns new DeploymentBinding
func DeploymentBinding() v1alpha1.DeploymentBinding {
	return v1alpha1.DeploymentBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind: "DeploymentBinding",
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

// EmptyKconfigEnvsDeploymentBinding returns kcb with empty kconfigEnvs
func EmptyKconfigEnvsDeploymentBinding() v1alpha1.DeploymentBinding {
	kconfigEnvs := v1alpha1.KconfigEnvs{
		Level: DefaultLevel,
		Envs:  []corev1.EnvVar{},
	}
	kcb := DeploymentBinding()
	kcb.Spec.KconfigEnvsMap[DefaultKconfigEnvsKey] = kconfigEnvs
	return kcb
}

// ValueDeploymentBinding returns DeploymentBinding with Value envVar
func ValueDeploymentBinding() v1alpha1.DeploymentBinding {
	kconfigEnvs := v1alpha1.KconfigEnvs{
		Level: DefaultLevel,
		Envs: []corev1.EnvVar{
			corev1.EnvVar{
				Name:  DefaultKey,
				Value: DefaultValue,
			},
		},
	}
	kcb := DeploymentBinding()
	kcb.Spec.KconfigEnvsMap[DefaultKconfigEnvsKey] = kconfigEnvs
	return kcb
}

// NewValueDeploymentBinding returns DeploymentBinding with Value envVar
func NewValueDeploymentBinding() v1alpha1.DeploymentBinding {
	kconfigEnvs := v1alpha1.KconfigEnvs{
		Level: DefaultLevel,
		Envs: []corev1.EnvVar{
			corev1.EnvVar{
				Name:  DefaultKey,
				Value: DefaultNewValue,
			},
		},
	}
	kcb := DeploymentBinding()
	kcb.Spec.KconfigEnvsMap[DefaultKconfigEnvsKey] = kconfigEnvs
	return kcb
}

// ConfigMapDeploymentBinding returns DeploymentBinding with ConfigMap envVar
func ConfigMapDeploymentBinding(envRefsVersion int64, configMapName string) v1alpha1.DeploymentBinding {
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
	kcb := DeploymentBinding()
	kcb.Spec.KconfigEnvsMap[DefaultKconfigEnvsKey] = kconfigEnvs
	return kcb
}

// SecretDeploymentBinding returns DeploymentBinding with Secret envVar
func SecretDeploymentBinding(envRefsVersion int64, secretName string) v1alpha1.DeploymentBinding {
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
	kcb := DeploymentBinding()
	kcb.Spec.KconfigEnvsMap[DefaultKconfigEnvsKey] = kconfigEnvs
	return kcb
}

// FieldRefDeploymentBinding FieldRefDeploymentBinding
func FieldRefDeploymentBinding(envRefsVersion int64) v1alpha1.DeploymentBinding {
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
	kcb := DeploymentBinding()
	kcb.Spec.KconfigEnvsMap[DefaultKconfigEnvsKey] = kconfigEnvs
	return kcb
}

// ResourceFieldRefDeploymentBinding ResourceFieldRefDeploymentBinding
func ResourceFieldRefDeploymentBinding(envRefsVersion int64) v1alpha1.DeploymentBinding {
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
	kcb := DeploymentBinding()
	kcb.Spec.KconfigEnvsMap[DefaultKconfigEnvsKey] = kconfigEnvs
	return kcb
}

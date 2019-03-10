package util

import (
	"github.com/gbraxton/kconfig/internal/app/kconfig-controller/controller"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TypeMeta returns Deployment TypeMeta
func TypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: appsv1.SchemeGroupVersion.String(),
		Kind:       "Deployment",
	}
}

// Deployment returns base deployment
func Deployment() appsv1.Deployment {
	return appsv1.Deployment{
		TypeMeta: TypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Namespace: DefaultNamespace,
			Name:      DefaultName,
			Labels: map[string]string{
				DefaultSelectorKey: DefaultSelectorValue,
			},
			Annotations: map[string]string{
				controller.KconfigEnabledDeploymentAnnotation: "true",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					DefaultSelectorKey: DefaultSelectorValue,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: DefaultNamespace,
					Name:      DefaultName,
					Labels: map[string]string{
						DefaultSelectorKey: DefaultSelectorValue,
					},
					Annotations: map[string]string{
						controller.KconfigEnvRefVersionAnnotation: "0",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						corev1.Container{
							Env: []corev1.EnvVar{},
						},
					},
				},
			},
		},
	}
}

// ValueDeployment returns deployment with a name/value envVar
func ValueDeployment() appsv1.Deployment {
	d := Deployment()
	envs := []corev1.EnvVar{
		corev1.EnvVar{
			Name:  DefaultKey,
			Value: DefaultValue,
		},
	}
	d.Spec.Template.Spec.Containers[0].Env = envs
	return d
}

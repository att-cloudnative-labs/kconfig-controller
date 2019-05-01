package util


import (
	"github.com/att-cloudnative-labs/kconfig-controller/internal/app/kconfig-controller/controller"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TypeMeta returns StatefulSet TypeMeta
func StatefulSetTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: appsv1.SchemeGroupVersion.String(),
		Kind:       "StatefulSet",
	}
}

// StatefulSet returns base StatefulSet
func StatefulSet() appsv1.StatefulSet {
	return appsv1.StatefulSet{
		TypeMeta: StatefulSetTypeMeta(),
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
		Spec: appsv1.StatefulSetSpec{
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

// ValueStatefulSet returns StatefulSet with a name/value envVar
func ValueStatefulSet() appsv1.StatefulSet {
	d := StatefulSet()
	envs := []corev1.EnvVar{
		corev1.EnvVar{
			Name:  DefaultKey,
			Value: DefaultValue,
		},
	}
	d.Spec.Template.Spec.Containers[0].Env = envs
	return d
}

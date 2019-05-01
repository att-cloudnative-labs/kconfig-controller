package util

import (
	"github.com/att-cloudnative-labs/kconfig-controller/internal/app/kconfig-controller/controller"
	knv1alpha1 "github.com/knative/serving/pkg/apis/serving/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KnativeServiceTypeMeta returns KnativeService TypeMeta
func KnativeServiceTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: knv1alpha1.SchemeGroupVersion.String(),
		Kind:       "KnativeService",
	}
}

// KnativeService returns base KnativeService
func KnativeService() knv1alpha1.Service {
	return knv1alpha1.Service{
		TypeMeta: KnativeServiceTypeMeta(),
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
		Spec: knv1alpha1.ServiceSpec{
			Release: &knv1alpha1.ReleaseType{
				Configuration:knv1alpha1.ConfigurationSpec{
					RevisionTemplate:knv1alpha1.RevisionTemplateSpec{

					},
				},
			},
		},
	}
}

// ValueKnativeService returns KnativeService with a name/value envVar
func ValueKnativeService() knv1alpha1.Service {
	ks := KnativeService()
	envs := []corev1.EnvVar{
		{
			Name:  DefaultKey,
			Value: DefaultValue,
		},
	}
	ks.Spec.Release.Configuration.RevisionTemplate.Spec.Container.Env = envs
	return ks
}


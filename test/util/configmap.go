package util

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigMap returns new configmap without any data
func ConfigMap() corev1.ConfigMap {
	return corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: DefaultNamespace,
			Name:      DefaultConfigMapName,
		},
		Data: map[string]string{},
	}
}

// ConfigMapWithData returns configMap with single data
func ConfigMapWithData() corev1.ConfigMap {
	configmap := ConfigMap()
	configmap.Data[DefaultReferenceKey] = DefaultValue
	return configmap
}

// ConfigMapWithNewData returns configMap with new value in data
func ConfigMapWithNewData() corev1.ConfigMap {
	configmap := ConfigMap()
	configmap.Data[DefaultReferenceKey] = DefaultNewValue
	return configmap
}

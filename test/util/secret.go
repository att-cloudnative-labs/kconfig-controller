package util

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Secret returns new secret without any data
func Secret(name string) corev1.Secret {
	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: DefaultNamespace,
			Name:      name,
		},
		Data: map[string][]byte{},
	}
}

// SecretWithData returns new secret with default data
func SecretWithData(name string) corev1.Secret {
	secret := Secret(name)
	secret.Data[DefaultReferenceKey] = []byte(DefaultValue)
	return secret
}

// SecretWithStringData returns new secret with default stringData
func SecretWithStringData(name string) corev1.Secret {
	secret := Secret(name)
	secret.Data[DefaultReferenceKey] = []byte(DefaultValue)
	return secret
}

// SecretWithNewData returns secret with new default data
func SecretWithNewData(name string) corev1.Secret {
	secret := Secret(name)
	secret.Data[DefaultReferenceKey] = []byte(DefaultNewValue)
	return secret
}

// SecretWithNewStringData returns secret with new default stringData
func SecretWithNewStringData(name string) corev1.Secret {
	secret := Secret(name)
	secret.StringData[DefaultReferenceKey] = DefaultNewValue
	return secret
}

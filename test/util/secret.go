package util

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Secret returns new secret without any data
func Secret() corev1.Secret {
	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: DefaultNamespace,
			Name:      DefaultSecretName,
		},
		Data:       map[string][]byte{},
		StringData: map[string]string{},
	}
}

// SecretWithData returns new secret with default data
func SecretWithData() corev1.Secret {
	secret := Secret()
	secret.Data[DefaultReferenceKey] = []byte(DefaultEncodedValue)
	return secret
}

// SecretWithStringData returns new secret with default stringData
func SecretWithStringData() corev1.Secret {
	secret := Secret()
	secret.StringData[DefaultReferenceKey] = DefaultValue
	return secret
}

// SecretWithNewData returns secret with new default data
func SecretWithNewData() corev1.Secret {
	secret := Secret()
	secret.Data[DefaultReferenceKey] = []byte(DefaultEncodedNewValue)
	return secret
}

// SecretWithNewStringData returns secret with new default stringData
func SecretWithNewStringData() corev1.Secret {
	secret := Secret()
	secret.StringData[DefaultReferenceKey] = DefaultNewValue
	return secret
}

package v1alpha1

import (
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestKconfigEqual(t *testing.T) {
	firstvalue := "firstvalue"
	k1 := Kconfig{
		Spec: KconfigSpec{
			Level: 1,
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"key1": "value1",
				},
			},
			EnvConfigs: []EnvConfig{
				EnvConfig{
					Key:   "firstkey",
					Value: &firstvalue,
				},
				EnvConfig{
					Key: "secondkey",
					SecretKeyRef: &v1.SecretKeySelector{
						Key:                  "firstkeyrefkey",
						LocalObjectReference: v1.LocalObjectReference{Name: "nameofsecret1"},
					},
				},
			},
		},
	}

	k2 := Kconfig{
		Spec: KconfigSpec{
			Level: 1,
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"key1": "value1",
				},
			},
			EnvConfigs: []EnvConfig{
				EnvConfig{
					Key:   "firstkey",
					Value: &firstvalue,
				},
				EnvConfig{
					Key: "secondkey",
					SecretKeyRef: &v1.SecretKeySelector{
						Key:                  "firstkeyrefkey",
						LocalObjectReference: v1.LocalObjectReference{Name: "nameofsecret1"},
					},
				},
			},
		},
	}

	if !k1.KconfigEqual(k2) {
		t.Fatalf("Kconfigs not equal")
	}
}

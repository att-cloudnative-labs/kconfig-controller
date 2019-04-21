package util

import (
	"github.com/gbraxton/kconfig/pkg/apis/kconfigcontroller/v1alpha1"
	"github.com/gbraxton/kconfig/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ValueKconfigEnvs returns KconfigEnv with single key/val envVar
func ValueKconfigEnvs(name, value string) v1alpha1.KconfigEnvs {
	return v1alpha1.KconfigEnvs{
		Level: DefaultLevel,
		Envs: []corev1.EnvVar{
			corev1.EnvVar{
				Name:  name,
				Value: value,
			},
		},
	}
}

// ConfigMapKconfigEnvs returns KconfigEnvs containing single configMap envVar
func ConfigMapKconfigEnvs(name, cmName, key string) v1alpha1.KconfigEnvs {
	optional := true
	return v1alpha1.KconfigEnvs{
		Level: DefaultLevel,
		Envs: []corev1.EnvVar{
			corev1.EnvVar{
				Name: name,
				ValueFrom: &corev1.EnvVarSource{
					ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: cmName},
						Key:                  key,
						Optional:             &optional,
					},
				},
			},
		},
	}
}

// SecretKconfigEnvs returns KconfigEnvs containing single secret envVar
func SecretKconfigEnvs(name, secName, key string) v1alpha1.KconfigEnvs {
	optional := true
	return v1alpha1.KconfigEnvs{
		Level: DefaultLevel,
		Envs: []corev1.EnvVar{
			corev1.EnvVar{
				Name: name,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: secName},
						Key:                  key,
						Optional:             &optional,
					},
				},
			},
		},
	}
}

// GetTestObjectMeta returns test metadata
func GetTestObjectMeta() v1.ObjectMeta {
	return v1.ObjectMeta{
		Namespace: TestNamespace,
		Name:      TestName,
	}
}

// GetTestObjectMetaWithLabels returns test metadata with labels
func GetTestObjectMetaWithLabels() v1.ObjectMeta {
	return v1.ObjectMeta{
		Namespace: TestNamespace,
		Name:      TestName,
		Labels: map[string]string{
			TestAppKey: TestAppName,
		},
	}
}

// GetTestSecretKeySelector1 returns test SecretKeyRef
func GetTestSecretKeySelector1() corev1.SecretKeySelector {
	key, _ := util.GetSecretRefKey(TestSecretKey1, TestSecretValue1)
	optional := false
	return corev1.SecretKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{Name: TestSecretName1},
		Key:                  key,
		Optional:             &optional,
	}
}

// GetTestEnvConfig1 returns test envconfig
func GetTestEnvConfig1() v1alpha1.EnvConfig {
	testValue1 := TestValue1
	return v1alpha1.EnvConfig{
		Key:   TestKey1,
		Value: &testValue1,
	}
}

// GetTestEnvConfig2 returns test envconfig
func GetTestEnvConfig2() v1alpha1.EnvConfig {
	testValue2 := TestValue2
	return v1alpha1.EnvConfig{
		Key:   TestKey2,
		Value: &testValue2,
	}
}

// GetTestSecretKeyRef1 test
func GetTestSecretKeyRef1() corev1.SecretKeySelector {
	optional := false
	return corev1.SecretKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{Name: TestSecretName1},
		Key:                  TestSecretKeySelectorKey1,
		Optional:             &optional,
	}
}

// GetTestAddSecretValue1 returns test secret value with add action
// func GetTestAddSecretValue1() v1alpha1.KconfigSecretValue {
// 	return v1alpha1.KconfigSecretValue{
// 		SecretName: TestSecretName1,
// 		Action:     AddAction,
// 	}
// }

// GetTestDeleteSecretValue1 returns delete action KconfigSecretValue
// func GetTestDeleteSecretValue1() v1alpha1.KconfigSecretValue {
// 	secretKeyRef := GetTestSecretKeyRef1()
// 	return v1alpha1.KconfigSecretValue{
// 		SecretName:   TestSecretName1,
// 		SecretKeyRef: &secretKeyRef,
// 		Action:       DeleteAction,
// 	}
// }

// GetTestAddSecretEnvConfig1 returns test secret envconfig
func GetTestAddSecretEnvConfig1() v1alpha1.EnvConfig {
	testSecretValue1 := TestSecretValue1
	testSecretName := "testsecret"
	return v1alpha1.EnvConfig{
		Type:    "Secret",
		Key:     TestSecretKey1,
		Value:   &testSecretValue1,
		RefName: &testSecretName,
	}
}

// GetTestDeleteSecretEnvConfig1 return test secret envconfig with delete action
func GetTestDeleteSecretEnvConfig1() v1alpha1.EnvConfig {
	return v1alpha1.EnvConfig{
		Type: "Secret",
		Key:  TestSecretKey1,
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: "testsecret",
			},
			Key: "testkey",
		},
	}
}

// GetSingleKeyTestLabelSelector returns single key test labelSelector
func GetSingleKeyTestLabelSelector() v1.LabelSelector {
	return v1.LabelSelector{
		MatchLabels: map[string]string{
			TestAppKey: TestAppName,
		},
	}
}

// GetSingleEnvTestKconfigSpec returns single env test kconfigspec
func GetSingleEnvTestKconfigSpec() v1alpha1.KconfigSpec {
	envConfig := GetTestEnvConfig1()
	selector := GetSingleKeyTestLabelSelector()
	return v1alpha1.KconfigSpec{
		EnvConfigs: []v1alpha1.EnvConfig{envConfig},
		Selector:   selector,
	}
}

// GetAddSecretSingleEnvTestKconfigSpec returns single env test kconfigspec
func GetAddSecretSingleEnvTestKconfigSpec() v1alpha1.KconfigSpec {
	envConfig := GetTestAddSecretEnvConfig1()
	selector := GetSingleKeyTestLabelSelector()
	return v1alpha1.KconfigSpec{
		EnvConfigs: []v1alpha1.EnvConfig{envConfig},
		Selector:   selector,
	}
}

// GetDeleteSecretSingleEnvTestKconfigSpec returns a single env test kconfigSpec with a delete secret actions
func GetDeleteSecretSingleEnvTestKconfigSpec() v1alpha1.KconfigSpec {
	envConfig := GetTestDeleteSecretEnvConfig1()
	selector := GetSingleKeyTestLabelSelector()
	return v1alpha1.KconfigSpec{
		EnvConfigs: []v1alpha1.EnvConfig{envConfig},
		Selector:   selector,
	}
}

// GetMultiEnvTestKconfigSpec returns single env test kconfigspec
func GetMultiEnvTestKconfigSpec() v1alpha1.KconfigSpec {
	envConfig1 := GetTestEnvConfig1()
	envConfig2 := GetTestEnvConfig2()
	selector := GetSingleKeyTestLabelSelector()
	return v1alpha1.KconfigSpec{
		EnvConfigs: []v1alpha1.EnvConfig{envConfig1, envConfig2},
		Selector:   selector,
	}
}

// GetSingleEnvTestKconfig returns kconfig with single envconfig set
func GetSingleEnvTestKconfig() v1alpha1.Kconfig {
	metadata := GetTestObjectMeta()
	spec := GetSingleEnvTestKconfigSpec()
	return v1alpha1.Kconfig{
		ObjectMeta: metadata,
		Spec:       spec,
	}
}

// GetAddSecretSingleEnvTestKconfig returns kconfig with single envconfig set
func GetAddSecretSingleEnvTestKconfig() v1alpha1.Kconfig {
	metadata := GetTestObjectMeta()
	spec := GetAddSecretSingleEnvTestKconfigSpec()
	return v1alpha1.Kconfig{
		ObjectMeta: metadata,
		Spec:       spec,
	}
}

// GetDeleteSecretSingleEnvTestKconfig returns kconfig with single envconfig set
func GetDeleteSecretSingleEnvTestKconfig() v1alpha1.Kconfig {
	metadata := GetTestObjectMeta()
	spec := GetDeleteSecretSingleEnvTestKconfigSpec()
	return v1alpha1.Kconfig{
		ObjectMeta: metadata,
		Spec:       spec,
	}
}

// GetMultiEnvTestKconfig returns kconfig with single envconfig set
func GetMultiEnvTestKconfig() v1alpha1.Kconfig {
	metadata := GetTestObjectMeta()
	spec := GetMultiEnvTestKconfigSpec()
	return v1alpha1.Kconfig{
		ObjectMeta: metadata,
		Spec:       spec,
	}
}

// GetTestEnvVar1 returns test envvar
func GetTestEnvVar1() corev1.EnvVar {
	return corev1.EnvVar{
		Name:  TestKey1,
		Value: TestValue1,
	}
}

// GetTestEnvVar2 returns test envvar
func GetTestEnvVar2() corev1.EnvVar {
	return corev1.EnvVar{
		Name:  TestKey2,
		Value: TestValue2,
	}
}

// GetTestSecretEnvVar1 returns secret env var
func GetTestSecretEnvVar1() corev1.EnvVar {
	secretKeyRef := GetTestSecretKeySelector1()
	return corev1.EnvVar{
		Name: TestSecretKey1,
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &secretKeyRef,
		},
	}
}

// GetProcessedSingleEnvTestKconfigEnvs returns single env kconfigenvs
func GetProcessedSingleEnvTestKconfigEnvs() v1alpha1.KconfigEnvs {
	envVar := GetTestEnvVar1()
	return v1alpha1.KconfigEnvs{
		Level: 0,
		Envs:  []corev1.EnvVar{envVar},
	}
}

// GetProcessedEmptyEnvTestKconfigEnvs returns kconfigenvs with kconfig map that is empty
func GetProcessedEmptyEnvTestKconfigEnvs() v1alpha1.KconfigEnvs {
	return v1alpha1.KconfigEnvs{
		Level: 0,
		Envs:  []corev1.EnvVar{},
	}
}

// GetProcessedSingleSecretEnvTestKconfigEnvs returns single env kconfigenvs
func GetProcessedSingleSecretEnvTestKconfigEnvs() v1alpha1.KconfigEnvs {
	envVar := GetTestSecretEnvVar1()
	return v1alpha1.KconfigEnvs{
		Level: 0,
		Envs:  []corev1.EnvVar{envVar},
	}
}

// GetProcessedMultiEnvTestKconfigEnvs returns single env kconfigenvs
func GetProcessedMultiEnvTestKconfigEnvs() v1alpha1.KconfigEnvs {
	envVar1 := GetTestEnvVar1()
	envVar2 := GetTestEnvVar2()
	return v1alpha1.KconfigEnvs{
		Level: 0,
		Envs:  []corev1.EnvVar{envVar1, envVar2},
	}
}

// GetProcessedSingleEnvTestKconfigBindingSpec returns kconfigbinding spec with single env
func GetProcessedSingleEnvTestKconfigBindingSpec() v1alpha1.KconfigBindingSpec {
	kconfigEnvs := GetProcessedSingleEnvTestKconfigEnvs()
	return v1alpha1.KconfigBindingSpec{
		KconfigEnvsMap: map[string]v1alpha1.KconfigEnvs{
			TestNamespace + "/" + TestName: kconfigEnvs,
		},
	}
}

// GetProcessedEmptyEnvTestKconfigBindingSpec returns kconfigbinding spec with key for kcofnig that has empty env
func GetProcessedEmptyEnvTestKconfigBindingSpec() v1alpha1.KconfigBindingSpec {
	kconfigEnvs := GetProcessedEmptyEnvTestKconfigEnvs()
	return v1alpha1.KconfigBindingSpec{
		KconfigEnvsMap: map[string]v1alpha1.KconfigEnvs{
			TestNamespace + "/" + TestName: kconfigEnvs,
		},
	}
}

// GetProcessedSingleSecretEnvTestKconfigBindingSpec returns kconfigbinding spec with single env
func GetProcessedSingleSecretEnvTestKconfigBindingSpec() v1alpha1.KconfigBindingSpec {
	kconfigEnvs := GetProcessedSingleSecretEnvTestKconfigEnvs()
	return v1alpha1.KconfigBindingSpec{
		KconfigEnvsMap: map[string]v1alpha1.KconfigEnvs{
			TestNamespace + "/" + TestName: kconfigEnvs,
		},
	}
}

// GetProcessedMultiEnvTestKconfigBindingSpec returns kconfigbinding spec with single env
func GetProcessedMultiEnvTestKconfigBindingSpec() v1alpha1.KconfigBindingSpec {
	kconfigEnvs := GetProcessedMultiEnvTestKconfigEnvs()
	return v1alpha1.KconfigBindingSpec{
		KconfigEnvsMap: map[string]v1alpha1.KconfigEnvs{
			TestNamespace + "/" + TestName: kconfigEnvs,
		},
	}
}

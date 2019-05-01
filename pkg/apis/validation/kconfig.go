package validation

import (
	"fmt"
	"strings"

	"github.com/att-cloudnative-labs/kconfig-controller/pkg/apis/kconfigcontroller/v1alpha1"
)

// ValidateEnvConfig validate envConfig
func ValidateEnvConfig(envConfig v1alpha1.EnvConfig) error {
	if err := validateEnvConfigKey(envConfig); err != nil {
		return err
	}
	if err := validateEnvConfigType(envConfig); err != nil {
		return err
	}
	if err := validateNoConflictingRefs(envConfig); err != nil {
		return err
	}
	switch strings.ToLower(envConfig.Type) {
	case "", "value":
		if err := validateValueEnvConfig(envConfig); err != nil {
			return err
		}
	case "configmap":
		if err := validateConfigMapEnvConfig(envConfig); err != nil {
			return err
		}
	case "secret":
		if err := validateSecretEnvConfig(envConfig); err != nil {
			return err
		}
	case "fieldref":
		if err := validateFieldRefEnvConfig(envConfig); err != nil {
			return err
		}
	case "resourcefieldref":
		if err := validateResourceFieldRefEnvConfig(envConfig); err != nil {
			return err
		}
	}
	return nil
}

func validateEnvConfigType(envConfig v1alpha1.EnvConfig) error {
	switch strings.ToLower(envConfig.Type) {
	case "", "value", "configmap", "secret", "fieldref", "resourcefieldref":
		return nil
	default:
		return fmt.Errorf("invalid type: %s", envConfig.Type)
	}
}

func validateEnvConfigKey(envConfig v1alpha1.EnvConfig) error {
	if len(envConfig.Key) == 0 {
		return fmt.Errorf("EnvConfig must have Key")
	}
	return nil
}

func validateNoConflictingRefs(envConfig v1alpha1.EnvConfig) error {
	if envConfig.ConfigMapKeyRef != nil && strings.ToLower(envConfig.Type) != "configmap" {
		return fmt.Errorf("%s Type EnvConfig should not have ConfigMapKeyRef", envConfig.Type)
	}
	if envConfig.SecretKeyRef != nil && strings.ToLower(envConfig.Type) != "secret" {
		return fmt.Errorf("%s Type EnvConfig should not have SecretKeyRef", envConfig.Type)
	}
	if envConfig.FieldRef != nil && strings.ToLower(envConfig.Type) != "fieldref" {
		return fmt.Errorf("%s Type EnvConfig should not have FieldRef", envConfig.Type)
	}
	if envConfig.ResourceFieldRef != nil && strings.ToLower(envConfig.Type) != "resourcefieldref" {
		return fmt.Errorf("%s Type EnvConfig should not have ResourceFieldRef", envConfig.Type)
	}
	return nil
}

func validateValueEnvConfig(envConfig v1alpha1.EnvConfig) error {
	if envConfig.Value == nil {
		return fmt.Errorf("Value Type EnvConfig must have Value")
	}
	if envConfig.RefName != nil {
		return fmt.Errorf("Value Type EnvConfig should not have RefName")
	}
	if envConfig.RefKey != nil {
		return fmt.Errorf("Value Type EnvConfig should not have RefKey")
	}
	return nil
}

func validateConfigMapEnvConfig(envConfig v1alpha1.EnvConfig) error {
	// For Pre-existing ConfigMap EnvConfig
	if envConfig.ConfigMapKeyRef != nil {
		return validateExistingConfigMapEnvConfig(envConfig)
	}
	// For Non-pre-existing ConfigMap EnvConfig
	if envConfig.Value == nil {
		return fmt.Errorf("New ConfigMap EnvConfigs should have a value")
	}
	return nil
}

func validateExistingConfigMapEnvConfig(envConfig v1alpha1.EnvConfig) error {
	if envConfig.RefName != nil {
		return fmt.Errorf("New ConfigMap EnvConfigs should not have RefName")
	}
	if envConfig.RefKey != nil {
		return fmt.Errorf("New ConfigMap EnvConfigs should not have RefKey")
	}
	return nil
}

func validateSecretEnvConfig(envConfig v1alpha1.EnvConfig) error {
	// For Pre-existing Secret EnvConfig
	if envConfig.SecretKeyRef != nil {
		return validateExistingSecretEnvConfig(envConfig)
	}
	// For Non-pre-existing Secret EnvConfig
	if envConfig.Value == nil {
		return fmt.Errorf("New Secret EnvConfigs should have a value")
	}
	return nil
}

func validateExistingSecretEnvConfig(envConfig v1alpha1.EnvConfig) error {
	if envConfig.RefName != nil {
		return fmt.Errorf("New Secret EnvConfigs should not have RefName")
	}
	if envConfig.RefKey != nil {
		return fmt.Errorf("New Secret EnvConfigs should not have RefKey")
	}
	return nil
}

func validateFieldRefEnvConfig(envConfig v1alpha1.EnvConfig) error {
	// Assumption is that if a value is present the fieldRef spec will be overwrittin anyway
	if envConfig.Value != nil {
		return nil
	}
	if envConfig.FieldRef == nil {
		return fmt.Errorf("FieldRef type EnvConfigs should have Value or valid FieldRef")
	}
	if len(envConfig.FieldRef.FieldPath) == 0 {
		return fmt.Errorf("FieldRef type EnvConfig is missing fieldPath")
	}
	return nil
}

func validateResourceFieldRefEnvConfig(envConfig v1alpha1.EnvConfig) error {
	// Assumption is that if a value is present the fieldRef spec will be overwrittin anyway
	if envConfig.Value != nil {
		return nil
	}
	if envConfig.ResourceFieldRef == nil {
		return fmt.Errorf("ResourceFieldRef type EnvConfigs should have Value or valid ResourceFieldRef")
	}
	if len(envConfig.ResourceFieldRef.Resource) == 0 {
		return fmt.Errorf("ResourceFieldRef type EnvConfig is missing resource")
	}
	return nil
}

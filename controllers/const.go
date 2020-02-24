package controllers

const (
	WarningEventType      = "Warning"
	InvalidEnvConfigEvent = "InvalidEnvConfig"

	ConfigMapEnvConfigType        = "ConfigMap"
	SecretEnvConfigType           = "Secret"
	FieldRefEnvConfigType         = "FieldRef"
	ResourceFieldRefEnvConfigType = "ResourceFieldRef"

	AllowTemplateUpdatesAnnotation = "kconfigcontroller.atteg.com/update-template"
	GenerationAnnotationPrefix     = "kconfigcontroller.atteg.com/"
)

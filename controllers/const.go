package controllers

const (
	WarningEventType      = "Warning"
	InvalidEnvConfigEvent = "InvalidEnvConfig"

	ValueEnvConfigType            = "Value"
	ConfigMapEnvConfigType        = "ConfigMap"
	SecretEnvConfigType           = "Secret"
	FieldRefEnvConfigType         = "FieldRef"
	ResourceFieldRefEnvConfigType = "ResourceFieldRef"

	AllowTemplateUpdatesAnnotation = "kconfigcontroller.atteg.com/refresh-template"
	GenerationAnnotationPrefix     = "kconfigcontroller.atteg.com/"
)

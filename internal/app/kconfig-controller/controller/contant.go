package controller

const (
	controllerAgentName = "kconfig-controller"
	// KconfigEnabledDeploymentAnnotation annotation for enabling deployment management (true/false)
	KconfigEnabledDeploymentAnnotation = "kconfigcontroller.atteg.com/env"
	// KconfigEnvRefVersionAnnotation deployment template annotation key where value equals current EnvRefVersion string to track update
	KconfigEnvRefVersionAnnotation = "kconfigcontroller.atteg.com/envrefversions"
	// SuccessSynced is used as part of the Event 'reason' when a resource is synced
	SuccessSynced = "Synced"
	// ErrResourceExists is used as part of the Event 'reason' when a resource fails
	// to sync due to a Deployment of the same name already existing.
	ErrResourceExists = "ErrResourceExists"
	// MessageResourceExists is the message used for Events when a resource
	// fails to sync due to a Deployment already existing
	MessageResourceExists = "Resource %q already exists and is not managed by Kconfig"
	// MessageKconfigResourceSynced is the message used for an Event fired when a Kconfig
	// is synced successfully
	MessageKconfigResourceSynced = "Kconfig synced successfully"
	// MessageDeploymentResourceSynced is the message used for an Event fired when a Deployment
	// is synced successfully
	MessageDeploymentResourceSynced = "Deployment synced successfully"
	// MessageStatefulSetResourceSynced is the message used for an Event fired when a StatefulSet
	// is synced successfully
	MessageStatefulSetResourceSynced = "Statefulset synced succesfully"
	// MessageKnativeServiceResourceSynced is the message used for an Event fired when a KnativeService
	// is synced successfully
	MessageKnativeServiceResourceSynced = "KnativeService synced successfully"
)

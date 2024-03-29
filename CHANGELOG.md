# Changelog

## Development release

0.9.0-BETA-1
- Add ContainerSector Field to Kconfig Spec. This indicates which container(s) should the Kconfig object apply to. This field is a label.Selector object, containing both MatchLabels and MatchExpression objects. A default-container-selector, set to everything, applies when the Kconfig Spec does not include a selector.
- Default-container-selector is overrideable through the controller-manager's command line arguments. When set, the value should be the json representation of a label.Selector.

0.8.0-BETA-1
- Full refactor to kubebuilder framework
- Environment variables not added to pods directly using admission-controller
- DeployentBindings, StatefulSetBindings, and KnativeServiceBindings removed and replaced with a single KconfigBinding resource
- The ability to specify refName and refKey has been removed. All external references (configmap, secrets) are placed in the same resource with the name, kc-(kconfig name)
- EnvRefsVersion removed. Changes now tracked using KconfigBinding generation and its status' observedGeneration

0.7.0-BETA-1

- KconfigBindings renamed to DeploymentBinding
- Support for StatefulSets with an accompanying resource StatefulSetBinding
- Support for Knative Services with an accompanying resource KnativeServiceBinding
- Pkg names are now aligned with github location
- Binary and k8s resources renamed from kconfig-controller-manager to kconfig-controller

0.6.0-BETA-1

- ExternalResources (ConfigMaps/Secrets) are now updated in one kubernetes api call per resource. Previously, multiple value changes (or adds) would result in an attempt to update the secret or configmap for each change. This would create a conflict error forcing the remaining values to be updated in retries.

0.5.1-BETA-1

- Fix issue where EnvConfig with empty type was giving error and being discarded. Empty type field now defaults to "Value" type as before.

0.5.0-BETA-1

- FieldRef/ResourceFieldRef EnvConfig types added. Value field creates a FielfRef with just a fieldPath containing the value. Value field creates a ResourceFieldPath with just a 'resource' field containing the value.
- EnvConfig Types are no longer case sensitive when modifying but will be replaced with the camel-cased type on processing. Crd removed the enum validation of type. This was to prevent case checking of types but now the api will not validate type.
- Makefile updated to build images and corrected target dependencies

0.4.1-BETA-1

- Controller informer event handlers place all updates into work queue regardless if the resource version hasn't changed. This address issue where target resources aren't updated if they were created after the source resource was modified.

0.4.0-BETA-1

- Adds prefix 'kc-' to the name of configmaps/secrets created automatically from envConfigs

0.3.2-ALPHA-1

- Fixes issue of panic when creating new secret for Kconfig envConfig with 'Secret' type

0.3.1-ALPHA-1

- Fixes issue of secret values being double base64 encoded

0.3.0-ALPHA-1

- Deployment annotation enabling Kconfig environment management changed to "kconfigcontroller.atteg.com/env"
- EnvRefsVersion field added to Kconfig.Spec and KconfigBinding.Spec.KconfigEnv to track changes to values of reference types (ConfigMap/Secret). Deployments are now correctly updated with template annotation to trigger new pods that pick up the update.

0.2.0-ALPHA-1

- Action field in EnvConfig is removed. Action to perform on Kconfig EnvConfig changes are implicit.
- RefName and RefKey fields added to EnvConfig to denote the ConfigMap or Secret name and key.
- The value field in EnvConfig is now a pointer to a string and can be null. This is used to determine when ConfigMap/Secret EnvConfig has been updated.

0.1.0-ALPHA-1

- Initial Release (ALPHA)
- Kconfig allows defining key/value environment variables
- Kconfig allows defining secret/configmap environment variables
- Kconfig controller reconciles KconfigBindings environment variables defined in Kconfigs
- KconfigBinding controller reconciles Deployments with the environment variables defined in KconfigBindings
- Deployment controller ensures annotated Deployments have a KconfigBinding created with matching labels

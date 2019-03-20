# Changelog

## Development release

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

## Current release

- 0.1.0-ALPHA-1

## Older releases

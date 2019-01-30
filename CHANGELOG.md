# Changelog

## Development release

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

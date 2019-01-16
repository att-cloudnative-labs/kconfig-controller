# Kconfig

----

Kconfig is a Kubernetes Custom-controller and CRD for externalizing configuration of Kubernetes deployments. Kconfig allows environment variables to be defined in a single resource that selects deployments based on labels, and inserts the specified environment variables into the deployment.

Multiple Kconfig resources can select a single deployment and the deployment will have the aggregation of each of those Kconfigs. In addition, Kconfigs have a level field which determines the order, in relation to other Kconfigs that select the same deployment, in which enviroment variables from multiple Kconfigs are defined in the container environment.

Aside from defining simple key/value pairs, Kconfigs can also define and reference environment variables to be stored in configmaps and/or secrets.

For a deployment to have its environment variables controlled by Kconfigs, it needs the annotation ```kconfigcontroller.atteg.com/enabled=true```.

Kconfigs have a secondary resource, KconfigBindings. These resources should not be created/manipulated directly by users and are used by the control loops. These KconfigBinding resources serve as a target for Kconfigs to update their changes whereafter, the controller can re-processed the contained enviroment variables for all Kconfigs that target a particular deployment. Note that there will always be one KconfigBinding for each Deployment that contains the kconfig enabled annotation shown above.

----

## Sample Kconfig
```
apiVersion: kconfigcontroller.atteg.com/v1alpha1
kind: Kconfig
metadata:
  name: mykconfig
  namespace: mynamespace
spec:
  envConfigs:
  - type: Value
    key: LITERALVALUEVAR
    value: firstvalue
  - type: Secret
    key: PLEASECREATETHIS
    value: shhhhh
    action:
      actionType: Add
      resourceName: samplesecret
  - type: Secret
    key: MYSECRETVAR
    secretKeyRef:
      key: mysecretvar
      name: samplesecret
      optional: true
  - type: ConfigMap
    key: MYCONFIGMAPVAR
    secretKeyRef:
      key: myconfigmapvar
      name: sameplecm
      optional: true
  level: 2
  selector:
    matchLabels:
      app: myapp
```

The first envConfig is a 'Value' type. An empty type field implies a 'Value' type envConfig. This definition would apply a simple key and value field to the target deployment's container enviroment variables. The second envConfig is a 'Secret' type. Notice the action field. This says that this key (and perhaps the secret named in the resourceName field) does not exist and should either be added to the secret if it exists or create the secret first, then add this key and value. After such an action takes place, the Kconfig is automatically updated with the secretKeyRef to the secret. The same is true with a 'ConfigMap' type. Notice the final two envConfigs that show how the envConfig appears after a Kconfig is created/updated with a envConfig action. Whenever a get Kconfig is performed, you will never see an action field, as the action is performed immediately on update and the field is automatically removed.

## Installation
```
kubectl apply -f deploy/
```
## Roadmap
* Remove the action field and make actions inferred by wether the value field is present (this will likely require additional fields for referencing the secret/configmap)
* Ability to select the container configs apply to. Currently the configs are only placed in the first container in a pod template
* Validate that all existing configmap/secret references in a Kconfig exists and if not, removed them from the Kconfig
* Support for files form and mount locations for files through Kconfigs
* Possible move to injecting the environment variables directly to pods through a custom admission controller
* Support for fieldRefs
* Support for configuring Statefulsets
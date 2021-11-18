# Kconfig

---

<p align="center">
  <a href="https://goreportcard.com/report/github.com/att-cloudnative-labs/kconfig-controller" alt="Go Report Card">
    <img src="https://goreportcard.com/badge/github.com/att-cloudnative-labs/kconfig-controller">
  </a>	
</p>
<p align="center">
    <a href="https://github.com/att-cloudnative-labs/kconfig-controller/graphs/contributors" alt="Contributors">
		<img src="https://img.shields.io/github/contributors/att-cloudnative-labs/kconfig-controller.svg">
	</a>
	<a href="https://github.com/att-cloudnative-labs/kconfig-controller/commits/master" alt="Commits">
		<img src="https://img.shields.io/github/commit-activity/m/att-cloudnative-labs/kconfig-controller.svg">
	</a>
	<a href="https://github.com/att-cloudnative-labs/kconfig-controller/pulls" alt="Open pull requests">
		<img src="https://img.shields.io/github/issues-pr-raw/att-cloudnative-labs/kconfig-controller.svg">
	</a>
	<a href="https://github.com/att-cloudnative-labs/kconfig-controller/pulls" alt="Closed pull requests">
    	<img src="https://img.shields.io/github/issues-pr-closed-raw/att-cloudnative-labs/kconfig-controller.svg">
	</a>
	<a href="https://github.com/att-cloudnative-labs/kconfig-controller/issues" alt="Issues">
		<img src="https://img.shields.io/github/issues-raw/att-cloudnative-labs/kconfig-controller.svg">
	</a>
	</p>
<p align="center">
	<a href="https://github.com/att-cloudnative-labs/kconfig-controller/stargazers" alt="Stars">
		<img src="https://img.shields.io/github/stars/att-cloudnative-labs/kconfig-controller.svg?style=social">
	</a>	
	<a href="https://github.com/att-cloudnative-labs/kconfig-controller/watchers" alt="Watchers">
		<img src="https://img.shields.io/github/watchers/att-cloudnative-labs/kconfig-controller.svg?style=social">
	</a>	
	<a href="https://github.com/att-cloudnative-labs/kconfig-controller/network/members" alt="Forks">
		<img src="https://img.shields.io/github/forks/att-cloudnative-labs/kconfig-controller.svg?style=social">
	</a>	
</p>

----

Kconfig is a Kubernetes custom-controller, admission-webhook, and custom resource definition for externalizing configuration of Kubernetes Pods. Kconfig allows environment variables to be defined in a single resource that selects the target pods based on labels, and inserts the specified environment variables into the target pods.

Multiple Kconfig resources can select the same target labels and the target pods will have the aggregation of each of those Kconfigs. In addition, Kconfigs have a level field which determines the order, in relation to other Kconfigs that select common pods, in which environment variables from multiple Kconfigs are defined in the container environment.

Aside from defining simple key/value pairs, Kconfigs can also define and reference environment variables to be stored in configmaps and/or secrets.

For a target to have its environment variables controlled by Kconfigs, it needs the annotation ```kconfigcontroller.atteg.com/inject=true```.

Add the annotation, ```kconfigcontroller.atteg.com/refresh-template=true``` to have updates to a kconfig to trigger a rolling update for deployments, statefulsets of the selected pods.

Kconfig-controller also has a secondary custom resource, KconfigBinding, that is used by the controllers and should not be created/manipulated directly by users. This resources serve as a target for Kconfigs to update their changes whereafter, the admission-controller can import the contained environment variables directly into pods. Note that there is a one-to-one mapping for each kconfig and kconfigbinding.

Build requires Kustomize (https://github.com/kubernetes-sigs/kustomize) locally and cert-manager (https://github.com/jetstack/cert-manager) installed in the kubernetes cluser for the admission-controller's TLS certificates.

----

## Sample Kconfig

```yaml
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
  - type: FieldRef
    key: MYPODIP
    value: status.podIP
  - type: ResourceFieldRef
    key: MYRESOURCE
    resourceFieldRef:
      resource: limits.memory
  level: 2
  selector:
    matchLabels:
      app: myapp
 containerSelector:
    matchLabels:
      name: myapp

```

The first envConfig is a 'Value' type. An empty type field implies a 'Value' type envConfig. This definition would apply a simple key and value field to the target pod's container environment variables. The second envConfig is a 'Secret' type. The Kconfig is automatically updated with the secretKeyRef to the secret and with the value field removed. The same is true with a 'ConfigMap' type. Notice the final two envConfigs that show how the envConfig appears after a Kconfig is created/updated with a ConfigMap or Secret type envConfig that contains a value. Whenever a get Kconfig is performed, you will never see a value field, as the action is performed immediately on update and the field is automatically removed. ContainerSelector determines which container the configs will apply to. Containers don't have labels so the selector selects on name. Name should currently be the only key used in the selector. In absence of a ContainerSelector, the default containerSelector is used, which selects everything (all containers). this can be overridden with an argument to the controller. Example ```--default-container-selector='{"matchExpressions":[{"key":"name","operator":"NotIn","values":["istio-proxy"]}]}```

## Build and Push

```bash
make docker-build IMG=your-registry.com/kconfig-controller-system/kconfig-controller:v1beta1
make docker-push IMG=your-registry.com/kconfig-controller-system/kconfig-controller:v1beta1
```

## Installation

```bash
make deploy IMG=your-registry.com/kconfig-controller-system/kconfig-controller:v1beta1
```

## Roadmap

* Validate that all existing configmap/secret references in a Kconfig exists and if not, removed them from the Kconfig
* Support for creating files and mount locations for files through Kconfigs

---

*Developed using the Kubebuilder Framework, https://github.com/kubernetes-sigs/kubebuilder
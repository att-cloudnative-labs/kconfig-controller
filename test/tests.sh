#!/bin/bash

function createSecret {

  kubectl delete secret kc-myapp-app-default --force >/dev/null 2>&1

  cat <<EOF | kubectl apply -f -
apiVersion: v1
data:
kind: Secret
metadata:
  annotations:
  creationTimestamp: "2022-12-13T04:13:48Z"
  labels:
    velero.io/backup-name: dev-int-20221219180236
    velero.io/restore-name: dev-int-20221219180236-20221219193705
  name: kc-myapp-app-default
  namespace: default
  resourceVersion: "971337234"
type: Opaque
EOF

}

function createKConfig {
  kubectl delete kconfigs.kconfigcontroller.atteg.com myapp-app-default --force >/dev/null 2>&1
  kubectl apply -f myapp-app-default-kconfig.yaml
}

make -C ../ uninstall
make -C ../ install

# given KConfig object when we add the secret entry and run kconfig-controller with any expiration time then we see only the secret reference, but not the expiration annotation

createKConfig
kubectl patch kconfigs.kconfigcontroller.atteg.com myapp-app-default --type merge -p \
  '{"spec": {"envConfigs": [{"key": "secret1", "type": "Secret", "value": "foo1"}]}}'

createSecret

timeout 20s go run ../main.go -key-removal-period-sec 10

SECRET_NAME=$(kubectl get kconfigs.kconfigcontroller.atteg.com myapp-app-default -ojson | jq ".spec.envConfigs[0].key")
SECRET_KEY=$(kubectl get kconfigs.kconfigcontroller.atteg.com myapp-app-default -ojson | jq ".spec.envConfigs[0].secretKeyRef.key")
echo "$SECRET_NAME must be secret1 with secret key: $SECRET_KEY"

SECRET_DATA=$(kubectl get secret kc-myapp-app-default -ojson | jq ".data")
ANNOTATIONS=$(kubectl get secret kc-myapp-app-default -ojson | jq ".metadata.annotations")
echo "Secret data is $SECRET_DATA with annotations $ANNOTATIONS"

#given KConfig object when we remove the secret entry and run kconfig-controller with a quite big time then we don't see the secret reference, but can see the expiration annotation and the secret data

kubectl patch kconfigs.kconfigcontroller.atteg.com myapp-app-default --type merge -p \
  '{"spec": {"envConfigs": []}}'

timeout 20s go run ../main.go -key-removal-period-sec 100000

SECRET_NAME=$(kubectl get kconfigs.kconfigcontroller.atteg.com myapp-app-default -ojson | jq ".spec.envConfigs[0].key")
SECRET_KEY=$(kubectl get kconfigs.kconfigcontroller.atteg.com myapp-app-default -ojson | jq ".spec.envConfigs[0].secretKeyRef.key")
echo "$SECRET_NAME must be null with secret key: $SECRET_KEY"

SECRET_DATA=$(kubectl get secret kc-myapp-app-default -ojson | jq ".data")
ANNOTATIONS=$(kubectl get secret kc-myapp-app-default -ojson | jq ".metadata.annotations")
echo "Secret data is $SECRET_DATA with annotations $ANNOTATIONS"

#given KConfig object when we remove the secret entry and run kconfig-controller with 1 sec expiration time time then we don't see the secret reference and don't see the expiration annotation and the secret data
kubectl patch kconfigs.kconfigcontroller.atteg.com myapp-app-default --type merge -p \
  '{"spec": {"envConfigs": [{"key": "secret1", "type": "Secret", "value": "foo1"}]}}'

timeout 20s go run ../main.go -key-removal-period-sec 10

kubectl patch kconfigs.kconfigcontroller.atteg.com myapp-app-default --type merge -p \
  '{"spec": {"envConfigs": []}}'

createSecret

timeout 20s go run ../main.go -key-removal-period-sec 1

SECRET_NAME=$(kubectl get kconfigs.kconfigcontroller.atteg.com myapp-app-default -ojson | jq ".spec.envConfigs[0].key")
SECRET_KEY=$(kubectl get kconfigs.kconfigcontroller.atteg.com myapp-app-default -ojson | jq ".spec.envConfigs[0].secretKeyRef.key")
echo "$SECRET_NAME must be null with secret key: $SECRET_KEY"

SECRET_DATA=$(kubectl get secret kc-myapp-app-default -ojson | jq ".data")
ANNOTATIONS=$(kubectl get secret kc-myapp-app-default -ojson | jq ".metadata.annotations")
echo "Secret data is $SECRET_DATA with annotations $ANNOTATIONS"

echo "Finished"

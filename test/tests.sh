#!/bin/bash

function setup {
  make -C ../ uninstall
  make -C ../ install
}

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


function assertValidUUID {
  pattern='^\{?[A-Z0-9a-z]{8}-[A-Z0-9a-z]{4}-[A-Z0-9a-z]{4}-[A-Z0-9a-z]{4}-[A-Z0-9a-z]{12}\}?$'
  uuid=$1
  if [[ "$uuid" =~ $pattern ]]; then
    echo "Assert OK"
  else
    echo "Assert Failed: $uuid is not valid"
    exit 1
  fi
}

function assertEqual() {
  actual=$1
  expected=$2
  if [ "$actual" = "$expected" ]; then
    echo "Assert OK"
  else
    echo "Assert Failed: $actual is not equal to $expected"
    exit 1
  fi
}



setup

# given KConfig object when we add the secret entry and run kconfig-controller with any expiration time then we see only the secret reference, but not the expiration annotation
createKConfig
kubectl patch kconfigs.kconfigcontroller.atteg.com myapp-app-default --type merge -p \
  '{"spec": {"envConfigs": [{"key": "secret1", "type": "Secret", "value": "foo1"}]}}'

createSecret

timeout 20s go run ../main.go -key-removal-period-sec 10

SECRET_NAME=$(kubectl get kconfigs.kconfigcontroller.atteg.com myapp-app-default -ojson | jq -r ".spec.envConfigs[0].key")
SECRET_KEY=$(kubectl get kconfigs.kconfigcontroller.atteg.com myapp-app-default -ojson | jq -r ".spec.envConfigs[0].secretKeyRef.key")
assertEqual $SECRET_NAME "secret1"
assertValidUUID $SECRET_KEY

SECRET_KEY_SECRET=$(kubectl get secret kc-myapp-app-default -ojson | jq " .data | to_entries" | jq -r ".[0].key")
SECRET_VALUE_SECRET=$(kubectl get secret kc-myapp-app-default -ojson | jq " .data | to_entries" | jq -r ".[0].value" |  base64 -d)
assertEqual $SECRET_KEY_SECRET $SECRET_KEY
assertEqual $SECRET_VALUE_SECRET "foo1"

ANNOTATION_KEY=$(kubectl get secret kc-myapp-app-default -ojson | jq -r ".metadata.annotations | to_entries | .[1].key")
ANNOTATION_VALUE_EXP_DATE=$(kubectl get secret kc-myapp-app-default -ojson | jq -r ".metadata.annotations | to_entries | .[1].value")
assertEqual $ANNOTATION_KEY "null"
assertEqual $ANNOTATION_VALUE_EXP_DATE "null"

#given KConfig object when we remove the secret entry and run kconfig-controller with a quite big expiration time then we don't see the secret reference, but can see the expiration annotation and the secret data
kubectl patch kconfigs.kconfigcontroller.atteg.com myapp-app-default --type merge -p \
  '{"spec": {"envConfigs": []}}'

timeout 20s go run ../main.go -key-removal-period-sec 100000

SECRET_NAME=$(kubectl get kconfigs.kconfigcontroller.atteg.com myapp-app-default -ojson | jq -r ".spec.envConfigs[0].key")
SECRET_KEY=$(kubectl get kconfigs.kconfigcontroller.atteg.com myapp-app-default -ojson | jq -r ".spec.envConfigs[0].secretKeyRef.key")
assertEqual $SECRET_NAME "null"
assertEqual $SECRET_KEY "null"

SECRET_KEY_SECRET=$(kubectl get secret kc-myapp-app-default -ojson | jq " .data | to_entries" | jq -r ".[0].key")
SECRET_VALUE_SECRET=$(kubectl get secret kc-myapp-app-default -ojson | jq " .data | to_entries" | jq -r ".[0].value" |  base64 -d)
assertValidUUID $SECRET_KEY_SECRET
assertEqual $SECRET_VALUE_SECRET "foo1"

ANNOTATION_KEY=$(kubectl get secret kc-myapp-app-default -ojson | jq -r ".metadata.annotations | to_entries | .[1].key")
ANNOTATION_VALUE_EXP_DATE=$(kubectl get secret kc-myapp-app-default -ojson | jq -r ".metadata.annotations | to_entries | .[1].value")
assertEqual $ANNOTATION_KEY "pendingkeyremoval/$SECRET_KEY_SECRET"

#given KConfig object when we remove the secret entry and run kconfig-controller with 1 sec expiration time time then we don't see the secret reference and don't see the expiration annotation and the secret data
kubectl patch kconfigs.kconfigcontroller.atteg.com myapp-app-default --type merge -p \
  '{"spec": {"envConfigs": [{"key": "secret1", "type": "Secret", "value": "foo1"}]}}'

timeout 20s go run ../main.go -key-removal-period-sec 10

kubectl patch kconfigs.kconfigcontroller.atteg.com myapp-app-default --type merge -p \
  '{"spec": {"envConfigs": []}}'

createSecret

timeout 20s go run ../main.go -key-removal-period-sec 1

SECRET_NAME=$(kubectl get kconfigs.kconfigcontroller.atteg.com myapp-app-default -ojson | jq -r ".spec.envConfigs[0].key")
SECRET_KEY=$(kubectl get kconfigs.kconfigcontroller.atteg.com myapp-app-default -ojson | jq -r ".spec.envConfigs[0].secretKeyRef.key")
assertEqual $SECRET_NAME "null"
assertEqual $SECRET_KEY "null"

ANNOTATION_KEY=$(kubectl get secret kc-myapp-app-default -ojson | jq -r ".metadata.annotations | to_entries | .[1].key")
ANNOTATION_VALUE_EXP_DATE=$(kubectl get secret kc-myapp-app-default -ojson | jq -r ".metadata.annotations | to_entries | .[1].value")
assertEqual $ANNOTATION_KEY "null"
assertEqual $ANNOTATION_VALUE_EXP_DATE "null"

echo "Finished"

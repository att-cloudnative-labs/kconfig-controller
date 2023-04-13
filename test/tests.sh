#!/bin/bash


make -C ../ uninstall
make -C ../ install


# given the KConfig object when we add a secret entry and run kconfig-controller with any expiration time then we see only secret references
kubectl delete  kconfigs.kconfigcontroller.atteg.com myapp-app-default --force >/dev/null 2>&1
kubectl apply -f myapp-app-default-kconfig.yaml

timeout 30s go run ../main.go -key-removal-period-sec 10

echo "Finished"


# Go parameters
VERSION=v0.5.0-beta-1
DOCKERIMAGE=docker-registry.aeg.cloud/common-system/kconfig-controller
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
KCONFIGPKG=github.com/gbraxton/kconfig/cmd
CLIENTSET=pkg/client/clientset/versioned/clientset.go
BINARY_NAME=kconfig-controller
BINARY_UNIX=$(BINARY_NAME)_unix

.PHONY: clientgen test build-docker build-local build-local-unix run push deploy clean

all: clientgen test build-docker

clientgen: $(CLIENTSET)

$(CLIENTSET): pkg/apis/kconfigcontroller/v1alpha1/types.go
	hack/update-codegen.sh

test:
	$(GOTEST) -v ./...

build-docker: test
	docker build -f build/Dockerfile -t $(DOCKERIMAGE):$(VERSION) .

build-local: $(BINARY_NAME)

build-local-unix: $(BINARY_UNIX)

$(BINARY_NAME): clientgen test **/*.go
	$(GOBUILD) -o $(BINARY_NAME) -v $(KCONFIGPKG)

$(BINARY_UNIX): clientgen test **/*.go
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME) -v $(KCONFIGPKG)

run: build-local
	./$(BINARY_NAME) -v 5 --kubeconfig ~/.kube/config --logtostderr

push:
	docker push $(DOCKERIMAGE):$(VERSION)

deploy:
	kubectl -n common-system replace -f install/

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)

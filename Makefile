# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
KCONFIGPKG=github.com/gbraxton/kconfig/cmd/kconfig-controller
BINARY_NAME=kconfig-controller
BINARY_UNIX=$(BINARY_NAME)_unix
MOCKGENCMD=$(GOPATH)/bin/mockgen

.PHONY: all build clean run
all: testall build

apis: pkg/apis/kconfigcontroller/v1alpha1/types.go
	hack/update-codegen.sh

build:
	$(GOBUILD) -o $(BINARY_NAME) -v $(KCONFIGPKG)

testall:
	$(GOTEST) -v ./...

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)
run:
	$(GOBUILD) -o $(BINARY_NAME) -v ./...
	./$(BINARY_NAME)

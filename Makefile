GO = go
GO_FLAGS =
GOFMT = gofmt
DOCKER = docker
EGRESSIP_CONTROLLER_IMAGE = static-egressip-controller
EGRESSIP_GATEWAY_MANAGER_IMAGE = static-egressip-gateway-manager
OS = linux
ARCH = amd64
BUNDLES = bundles
GO_PACKAGES = ./cmd/... ./pkg/...
GO_FILES := $(shell find $(shell $(GO) list -f '{{.Dir}}' $(GO_PACKAGES)) -name \*.go)

default: binary

controller-binary:
	CGO_ENABLED=1 ./script/controller-binary

manager-binary:
	CGO_ENABLED=1 ./script/gateway-manager-binary

update:
	./hack/update-codegen.sh

controller-container:
	docker build -t nirmata/$(EGRESSIP_CONTROLLER_IMAGE):latest -f Dockerfile.controller .

manager-container:
	docker build -t nirmata/$(EGRESSIP_GATEWAY_MANAGER_IMAGE):latest -f Dockerfile.manager .

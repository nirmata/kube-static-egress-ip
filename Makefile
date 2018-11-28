GO = go
GO_FLAGS =
GOFMT = gofmt
DOCKER = docker
EGRESSIP_CONTROLLER_IMAGE = egressip-controller-manager:latest
OS = linux
ARCH = amd64
BUNDLES = bundles
GO_PACKAGES = ./cmd/... ./pkg/...
GO_FILES := $(shell find $(shell $(GO) list -f '{{.Dir}}' $(GO_PACKAGES)) -name \*.go)

default: binary

binary:
	CGO_ENABLED=1 ./script/binary

update:
	./hack/update-codegen.sh

container:
	docker build -t nirmata/egressip-controller:latest -f Dockerfile .

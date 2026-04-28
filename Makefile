.POSIX:
PROG = hottake
VER = 0.1.0

HASH != git rev-parse --short HEAD 2>/dev/null
ifdef HASH
VERSION := $(VER)-$(HASH)
else
VERSION = $(VER)
endif

GH_ACCOUNT = lcook
GH_PROJECT = $(PROG)

GO_MODULE = github.com/$(GH_ACCOUNT)/$(GH_PROJECT)
GO_FLAGS = -v -ldflags "-s -w -X $(GO_MODULE)/internal/version.Build=$(VERSION)"

OCI_REPO ?= localhost
OCI_TAG = $(OCI_REPO)/$(GH_PROJECT):$(VERSION)
ifneq ($(OCI_REPO),localhost)
OCI_TAG = $(OCI_REPO)/$(GH_ACCOUNT)/$(GH_PROJECT)/$(PROG):$(HASH)
endif

CONFIG ?= ./config.example.yaml

default: build
build:
	go build $(GO_FLAGS) -o $(PROG) cmd/$(PROG)/$(PROG).go && strip -s $(PROG)

clean:
	rm -f $(PROG)
	go clean

container:
	podman build -t $(OCI_TAG) .

run-container:
	podman run -v $(CONFIG):/app/config.yaml $(OCI_TAG) /app/$(PROG) -V 2

publish-container: container
	podman push $(OCI_TAG)

update:
	go get -u -v ./...
	go mod tidy -v

lint:
	golangci-lint run

.PHONY: default build clean container run-container publish-container update lint
BINDIR      := $(CURDIR)/bin
CONTROLLERBINNAME ?= aws-services-stream
TARMAC     := aws-services-stream_darwin.tgz
TARAMD64   := aws-services-stream_amd64.tgz

PKG        := ./...
TAGS       :=
TESTS      := .
TESTFLAGS  :=
LDFLAGS    := -s
GOFLAGS    :=
SRC        := $(shell find . -type f -name '*.go' -print)

SHELL      = /bin/bash

NEW_TAG=$(shell cut -d'=' -f2- .release)
CONTAINER_REGISTRY = 374188005532.dkr.ecr.us-east-1.amazonaws.com
TEAM = anandnilkal
IMAGE = aws-services-stream

GIT_COMMIT = $(shell git rev-parse HEAD)
GIT_SHA    = $(shell git rev-parse --short HEAD)
GIT_TAG    = $(shell git describe --tags --abbrev=0 --exact-match 2>/dev/null)
GIT_DIRTY  = $(shell test -n "`git status --porcelain`" && echo "dirty" || echo "clean")

# ------------------------------------------------------------------------------

.PHONY: clean
clean:
	@rm -rf $(BINDIR) $(TARMAC) $(TARAMD64)

# ------------------------------------------------------------------------------

.PHONY: all
all: build release tar taramd64 container dockerhub_push

# ------------------------------------------------------------------------------
#  build

.PHONY: build
build: controller-build

# ------------------------------------------------------------------------------
#  release

.PHONY: release
release: controller-release

# ------------------------------------------------------------------------------
#  controller-build

.PHONY: controller-build
controller-build: $(BINDIR)/$(CONTROLLERBINNAME)

$(BINDIR)/$(CONTROLLERBINNAME): $(SRC)
	GO111MODULE=on go build $(GOFLAGS) -tags '$(TAGS)' -ldflags '$(LDFLAGS)' -o $(BINDIR)/$(CONTROLLERBINNAME) cmd/stream-controller/main.go

# ------------------------------------------------------------------------------
#  controller-release

.PHONY: controller-release
controller-release: $(BINDIR)/linux/$(CONTROLLERBINNAME)

$(BINDIR)/linux/$(CONTROLLERBINNAME): $(SRC)
	GO111MODULE=on GOARCH=amd64 GOOS=linux go build $(GOFLAGS) -tags '$(TAGS)' -ldflags '$(LDFLAGS)' -o $(BINDIR)/linux/$(CONTROLLERBINNAME) cmd/stream-controller/main.go

# ------------------------------------------------------------------------------
#  tar

.PHONY: tar
tar: $(TARMAC)

$(TARMAC):
	tar zcvf $(TARMAC) bin/$(CONTROLLERBINNAME)

# ------------------------------------------------------------------------------
#  taramd64

.PHONY: taramd64
taramd64: $(TARAMD64)

$(TARAMD64):
	tar zcvf $(TARAMD64) bin/linux/$(CONTROLLERBINNAME)

# ------------------------------------------------------------------------------
# git tag
.PHONY: tag
tag:
	@echo "creating tags"
	@git add .release
	@git commit -m "release $(NEW_TAG)" 
	@git tag $(NEW_TAG)
	@git push --tags origin master
	@echo "tag pushed successfully"

# ------------------------------------------------------------------------------
# container creation
.PHONE: container
container:
	@echo "creating container"
	@docker build --build-arg GOOS_VAL=linux --build-arg GOARCH_VAL=amd64 -t $(TEAM)/$(IMAGE):$(NEW_TAG) -f Dockerfile .
	@echo "docker image built successfully"

# ------------------------------------------------------------------------------
# container push
.PHONE: container_push
container_push:
	@echo "push container to remote registry"
	@aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin $(CONTAINER_REGISTRY)
	@docker push $(CONTAINER_REGISTRY)/$(TEAM)/$(IMAGE):$(NEW_TAG)
	@echo "docker container pushed successfully"

# ------------------------------------------------------------------------------
# dockerhub push
.PHONE: dockerhub_push
dockerhub_push:
	@echo "push container to remote registry"
	@docker login
	@docker push $(TEAM)/$(IMAGE):$(NEW_TAG)
	@echo "docker container pushed successfully"
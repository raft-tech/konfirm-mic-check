VERSION ?= $(shell cat VERSION)

DOCKER_CMD ?= $(shell which docker)
ifeq ($(DOCKER_CMD),)
  DOCKER_CMD = $(shell which podman)
endif

.PHONY: build
build: inspect \
 bin/konfirm-http \
 bin/konfirm-storage

.PHONY: test
export PATH := $(shell pwd)/bin:$(PATH)
test: bin/konfirm-storage bin/konfirm-http
	go test ./cmd/... ./internal/... ./pkg/... -test.v --ginkgo.github-output

.PHONY: clean
clean:
	[ ! -d bin ] || rm -rf bin
	[ ! -f cover.out ] || rm cover.out
	[ ! -f inspect ] || rm inspect


.PHONY: image
IMAGE_REPO ?= ghcr.io/raft-tech/konfirm/inspections
IMAGE_TAG ?= $(VERSION)
image:
	$(DOCKER_CMD) buildx build -t $(IMAGE_REPO):$(IMAGE_TAG) .

CACHE ?= $(shell pwd)/.cache
$(CACHE):
	mkdir -p $(CACHE)

.PHONY: kind-load
KIND_CLUSTER ?= konfirm
kind-load: image $(CACHE)
	$(DOCKER_CMD) image save $(IMAGE_REPO):$(IMAGE_TAG) > $(CACHE)/inspections.tar | kind load image-archive --name $(KIND_CLUSTER) $(CACHE)/inspections.tar

MONITORING_GATEWAY ?= gateway-prometheus-pushgateway.monitoring:9091
INSTALL_NAMESPACE ?= konfirm-inspections
install:
	helm upgrade --install --create-namespace -n $(INSTALL_NAMESPACE) inspect ./charts/inspect --set monitoring.gateway=$(MONITORING_GATEWAY)


.PHONY: inspect
inspect:
	go build -o inspect .

.PHONY: bin/konfirm-http
bin/konfirm-http:
	go test -c -o bin/konfirm-http ./inspections/http

.PHONY: bin/konfirm-storage
bin/konfirm-storage:
	go test -c -o bin/konfirm-storage ./inspections/storage

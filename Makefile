
.PHONY: build
build: inspect \
 bin/konfirm-storage

.PHONY: test
export PATH := $(shell pwd)/bin:$(PATH)
test: bin/konfirm-storage
	go test ./cmd/... ./internal/... ./pkg/... -test.v --ginkgo.github-output

.PHONY: clean
clean:
	[ ! -d bin ] || rm -rf bin
	[ ! -f cover.out ] || rm cover.out
	[ ! -f inspect ] || rm inspect


.PHONY: image
IMAGE_REPO ?= ghcr.io/raft-tech/konfirm/inspections
IMAGE_TAG ?= latest
image:
	docker build -t $(IMAGE_REPO):$(IMAGE_TAG) .

.PHONY: inspect
inspect:
	go build -o inspect .

.PHONY: bin/konfirm-storage
bin/konfirm-storage:
	go test -c -o bin/konfirm-storage ./inspections/storage

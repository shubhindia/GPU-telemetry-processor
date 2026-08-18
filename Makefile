DOCKER ?= podman
override GO := $(shell command -v go)
IMAGE_TAG ?= latest
IMAGE_REPO_PREFIX ?= shubhindia/gpu-telemetry

GO_CMD := $(shell $(GO) env GOROOT)/bin/go
GO_VERSION := $(shell $(GO_CMD) env GOVERSION)
GOENV = env -u GO -u GOROOT -u GOTOOLDIR -u GOFLAGS GOCACHE=$(CURDIR)/.gocache/$(GO_VERSION) GOMODCACHE=$(CURDIR)/.gomodcache/$(GO_VERSION)

QUEUE_IMAGE := $(IMAGE_REPO_PREFIX)-queue:$(IMAGE_TAG)
STREAMER_IMAGE := $(IMAGE_REPO_PREFIX)-streamer:$(IMAGE_TAG)
PROCESSOR_IMAGE := $(IMAGE_REPO_PREFIX)-processor:$(IMAGE_TAG)
API_IMAGE := $(IMAGE_REPO_PREFIX)-api:$(IMAGE_TAG)

.PHONY: fmt test coverage show-coverage swagger swagger-check clean-go-cache system-test build-queue build-streamer build-processor build-api build-images push-queue push-streamer push-processor push-api push-images

fmt:
	@$(GOENV) $(GO_CMD) fmt ./...

test:
	@$(GOENV) $(GO_CMD) test ./...

coverage:
	@$(GOENV) $(GO_CMD) test ./... -covermode=atomic -coverprofile=coverage.out
	@$(GOENV) $(GO_CMD) tool cover -func=coverage.out

show-coverage: coverage
	@$(GOENV) $(GO_CMD) tool cover -html=coverage.out -o coverage.html
	@open coverage.html

swagger:
	@mkdir -p internal/api/docs
	@$(GOENV) $(GO_CMD) generate ./internal/api

swagger-check: swagger
	@git diff --exit-code -- internal/api/docs/

clean-go-cache:
	@rm -rf .gocache .gomodcache coverage.out coverage.html

system-test:
	@./test/system/run.sh

build-queue:
	$(DOCKER) build --build-arg COMPONENT=queue -t $(QUEUE_IMAGE) .

build-streamer:
	$(DOCKER) build --build-arg COMPONENT=streamer -t $(STREAMER_IMAGE) .

build-processor:
	$(DOCKER) build --build-arg COMPONENT=processor -t $(PROCESSOR_IMAGE) .

build-api:
	$(DOCKER) build --build-arg COMPONENT=api -t $(API_IMAGE) .

build-images: build-queue build-streamer build-processor build-api

push-queue: build-queue
	$(DOCKER) push $(QUEUE_IMAGE)

push-streamer: build-streamer
	$(DOCKER) push $(STREAMER_IMAGE)

push-processor: build-processor
	$(DOCKER) push $(PROCESSOR_IMAGE)

push-api: build-api
	$(DOCKER) push $(API_IMAGE)

push-images: push-queue push-streamer push-processor push-api

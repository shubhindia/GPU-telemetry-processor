DOCKER ?= podman
IMAGE_TAG ?= latest
IMAGE_REPO_PREFIX ?= shubhindia/gpu-telemetry

QUEUE_IMAGE := $(IMAGE_REPO_PREFIX)-queue:$(IMAGE_TAG)
STREAMER_IMAGE := $(IMAGE_REPO_PREFIX)-streamer:$(IMAGE_TAG)
PROCESSOR_IMAGE := $(IMAGE_REPO_PREFIX)-processor:$(IMAGE_TAG)
API_IMAGE := $(IMAGE_REPO_PREFIX)-api:$(IMAGE_TAG)

.PHONY: fmt test build-queue build-streamer build-processor build-api build-images push-queue push-streamer push-processor push-api push-images

fmt:
	@go fmt ./...

test:
	@go test ./...

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

DOCKER ?= podman
IMAGE_TAG ?= latest
IMAGE_REPO_PREFIX ?= shubhindia/gpu-telemetry

QUEUE_IMAGE := $(IMAGE_REPO_PREFIX)-queue:$(IMAGE_TAG)
STREAMER_IMAGE := $(IMAGE_REPO_PREFIX)-streamer:$(IMAGE_TAG)

.PHONY: fmt test build-queue build-streamer build-images push-queue push-streamer push-images

fmt:
	@go fmt ./...

test:
	@go test ./...

build-queue:
	$(DOCKER) build --build-arg COMPONENT=queue -t $(QUEUE_IMAGE) .

build-streamer:
	$(DOCKER) build --build-arg COMPONENT=streamer -t $(STREAMER_IMAGE) .

build-images: build-queue build-streamer

push-queue: build-queue
	$(DOCKER) push $(QUEUE_IMAGE)

push-streamer: build-streamer
	$(DOCKER) push $(STREAMER_IMAGE)

push-images: push-queue push-streamer

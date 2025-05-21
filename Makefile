# Binary name
BINARY_NAME=quota-injector

# Docker image name and tag
IMAGE_NAME=ghcr.io/lengrongfu/snapshot-quota
IMAGE_TAG=latest

# Go build flags
GO_BUILD_FLAGS=CGO_ENABLED=1

# Docker buildx platform
PLATFORMS=linux/amd64

.PHONY: all build clean docker-build docker-push docker-buildx docker-pushx docker-allx

all: build

# Build binary
build:
	$(GO_BUILD_FLAGS) go build -o $(BINARY_NAME) ./cmd/quota-injector.go

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)

# Build Docker image
docker-build:
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) -f docker/Dockerfile .

# Push Docker image
docker-push:
	docker push $(IMAGE_NAME):$(IMAGE_TAG)

# Build and push Docker image
docker-all: docker-build docker-push

# Push multi-arch Docker image using buildx
docker-buildx:
	docker buildx build --platform $(PLATFORMS) \
		-t $(IMAGE_NAME):$(IMAGE_TAG) \
		-f docker/Dockerfile \
		--push .
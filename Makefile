.PHONY: build linux clean test helm integration-test integration-test-kind integration-test-image integration-test-binary kind-setup integration-test-clean kind-clean

CONTAINER_ENGINE ?= $(shell which podman >/dev/null 2>&1 && echo podman || echo docker)
CLUSTER_NAME ?= gabi-integration
IMAGE_NAME ?= gabi-integration-test:local

all: build

build:
	go build -o gabi cmd/gabi/main.go

linux:
	CGO_ENABLED=0 GOOS=linux go build -ldflags '-s -w' -o gabi cmd/gabi/main.go

clean:
	rm -f gabi integration.test

test:
	go test ./...

# Build the integration test binary locally
integration-test-binary:
	go test -c -tags integration -o integration.test ./test

# Build the container image for integration tests
integration-test-image:
	@$(CONTAINER_ENGINE) build -t $(IMAGE_NAME) -f Dockerfile.integration .

# Setup Kind cluster and load images (without running tests)
kind-setup:
	CONTAINER_ENGINE=$(CONTAINER_ENGINE) CLUSTER_NAME=$(CLUSTER_NAME) IMAGE_NAME=$(IMAGE_NAME) ./test/kind-setup.sh

# Run integration tests using oc CLI (works on Kind or OpenShift)
integration-test:
	IMAGE_URL=${IMAGE_URL} ./test/ephemeral-integration-test.sh

# Run integration tests in a local kind cluster (setup + test)
integration-test-kind: kind-setup
	IMAGE_URL=localhost/$(IMAGE_NAME) ./test/ephemeral-integration-test.sh

# Clean up integration test resources (job, pods, services)
integration-test-clean:
	@echo "Cleaning up integration test resources..."
	@oc delete job/gabi-integration-test-job --ignore-not-found=true
	@oc delete pod/test-pod --ignore-not-found=true
	@oc delete service/test-pod --ignore-not-found=true
	@oc delete configmap/wiremock-mappings --ignore-not-found=true
	@echo "✅ Cleanup complete!"

# Delete the Kind cluster
kind-clean:
	@echo "Deleting Kind cluster: $(CLUSTER_NAME)"
	@kind delete cluster --name $(CLUSTER_NAME)
	@echo "✅ Kind cluster deleted!"

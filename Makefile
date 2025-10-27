.PHONY: build linux clean test helm integration-test integration-test-kind integration-test-image integration-test-binary

all: build

build:
	go build -o gabi cmd/gabi/main.go

linux:
	CGO_ENABLED=0 GOOS=linux go build -ldflags '-s -w' -o gabi cmd/gabi/main.go

clean:
	rm -f gabi integration.test

test:
	go test ./...

integration-test:
	./ephemeral_integration_test.sh

# Build the integration test binary locally
integration-test-binary:
	go test -c -tags integration -o integration.test ./test

# Build the container image for integration tests
integration-test-image:
	podman build -t gabi-integration-test:local -f test/Dockerfile.integration .

# Run integration tests in a local kind cluster
integration-test-kind:
	./test/kind-integration-test.sh

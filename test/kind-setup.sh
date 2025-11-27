#!/usr/bin/env bash
#
# Kind cluster setup script
# Creates a Kind cluster and loads necessary images
#

set -o errexit
set -o nounset
set -o pipefail

echo "=== Kind Cluster Setup ==="
echo ""

# Configuration
CLUSTER_NAME="${CLUSTER_NAME:-gabi-integration}"
IMAGE_NAME="${IMAGE_NAME:-gabi-integration-test:local}"
NAMESPACE="${NAMESPACE:-default}"
CONTAINER_ENGINE="${CONTAINER_ENGINE:-$(which podman >/dev/null 2>&1 && echo podman || echo docker)}"

# Force Kind to use the same container runtime
# Map docker/podman to KIND_EXPERIMENTAL_PROVIDER
if [[ "$CONTAINER_ENGINE" == *"podman"* ]]; then
    export KIND_EXPERIMENTAL_PROVIDER="${KIND_EXPERIMENTAL_PROVIDER:-podman}"
fi
echo "Using container runtime: ${CONTAINER_ENGINE}"
echo ""

# Check prerequisites
if ! command -v kind &> /dev/null; then
    echo "Error: kind is not installed. Please install kind first."
    echo "Visit: https://kind.sigs.k8s.io/docs/user/quick-start/#installation"
    exit 1
fi

if ! command -v "${CONTAINER_ENGINE}" &> /dev/null; then
    echo "Error: ${CONTAINER_ENGINE} is not installed. Please install ${CONTAINER_ENGINE} first."
    exit 1
fi

# Step 1: Create kind cluster if it doesn't exist
echo "Step 1: Checking kind cluster..."
if ! kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
    echo "Creating kind cluster: ${CLUSTER_NAME}"
    kind create cluster --name "${CLUSTER_NAME}"
else
    echo "Kind cluster '${CLUSTER_NAME}' already exists"
fi

# Step 2: Build the integration test image
echo ""
echo "Step 2: Building integration test image..."
cd "$(dirname "$0")/.."
${CONTAINER_ENGINE} build -t "${IMAGE_NAME}" -f Dockerfile.integration .

# Step 3: Pull supporting service images
echo ""
echo "Step 3: Pulling supporting service images..."
echo "  - Pulling PostgreSQL image..."
${CONTAINER_ENGINE} pull registry.redhat.io/rhel9/postgresql-16:9.6

# Step 4: Load images into kind cluster
echo ""
echo "Step 4: Loading images into kind cluster..."
echo "  - Removing old integration test images from kind cluster..."
# Remove old images to prevent confusion between docker.io and localhost references
${CONTAINER_ENGINE} exec "${CLUSTER_NAME}-control-plane" crictl rmi "docker.io/library/${IMAGE_NAME}" 2>/dev/null || true
${CONTAINER_ENGINE} exec "${CLUSTER_NAME}-control-plane" crictl rmi "localhost/${IMAGE_NAME}" 2>/dev/null || true
${CONTAINER_ENGINE} exec "${CLUSTER_NAME}-control-plane" crictl rmi "${IMAGE_NAME}" 2>/dev/null || true

echo "  - Loading fresh integration test image..."
# Use save/import method which is more reliable for container->kind workflow
${CONTAINER_ENGINE} save "${IMAGE_NAME}" -o /tmp/gabi-test-image.tar
${CONTAINER_ENGINE} cp /tmp/gabi-test-image.tar "${CLUSTER_NAME}-control-plane:/gabi-test-image.tar"
${CONTAINER_ENGINE} exec "${CLUSTER_NAME}-control-plane" ctr -n k8s.io images import /gabi-test-image.tar
${CONTAINER_ENGINE} exec "${CLUSTER_NAME}-control-plane" rm /gabi-test-image.tar
rm /tmp/gabi-test-image.tar

echo "  - Loading PostgreSQL image..."
kind load docker-image registry.redhat.io/rhel9/postgresql-16:9.6 --name "${CLUSTER_NAME}"

# Verify the integration test image was loaded correctly
echo ""
echo "Verifying integration test image in cluster:"
${CONTAINER_ENGINE} exec "${CLUSTER_NAME}-control-plane" crictl images | grep gabi || echo "Warning: gabi image not found!"

echo ""
echo "✅ Kind cluster setup complete!"
echo ""
echo "Cluster name: ${CLUSTER_NAME}"
echo "Image name: localhost/${IMAGE_NAME}"
echo ""
echo "Next steps:"
echo "  - Run tests: IMAGE_URL=localhost/${IMAGE_NAME} ./test/ephemeral-integration-test.sh"
echo "  - Or use Makefile: make integration-test"
echo "  - Clean up: make integration-test-clean"
echo "  - Delete cluster: make kind-clean"
echo ""

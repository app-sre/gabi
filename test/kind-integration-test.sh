#!/usr/bin/env bash
#
# Integration test runner for Kind cluster using Podman
# Note: Kind auto-detects and works with podman when docker is not available
#

set -o errexit
set -o nounset
set -o pipefail

echo "=== Gabi Integration Test on Kind Cluster ==="
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

if ! command -v kubectl &> /dev/null; then
    echo "Error: kubectl is not installed. Please install kubectl first."
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

# Step 5: Deploy supporting services (database and mock-splunk)
echo ""
echo "Step 5: Deploying database and mock-splunk..."
# Set the mock-splunk image to the locally built image
MOCK_SPLUNK_IMAGE="localhost/${IMAGE_NAME}"
echo "Using mock-splunk image: ${MOCK_SPLUNK_IMAGE}"
sed "s|{{MOCK_SPLUNK_IMAGE}}|${MOCK_SPLUNK_IMAGE}|g" test/test-pod.yml | kubectl apply -f -

# Step 6: Wait for supporting services to be ready
echo ""
echo "Step 6: Waiting for services to be ready..."
echo "Waiting for test-pod to be ready..."
echo ""

# Show progress while waiting
(
  while kubectl get pod/test-pod -o json 2>/dev/null | jq -e '.status.conditions[] | select(.type=="Ready" and .status=="False")' > /dev/null 2>&1; do
    # Show container status
    echo -n "$(date '+%H:%M:%S') - Pod status: "
    kubectl get pod/test-pod -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown"

    # Show readiness status for each container
    echo -n "  Containers ready: "
    kubectl get pod/test-pod -o jsonpath='{range .status.containerStatuses[*]}{.name}={.ready} {end}' 2>/dev/null || echo "checking..."
    echo ""

    sleep 10
  done
) &
PROGRESS_PID=$!

# Wait for pod to be ready (increased timeout for Splunk)
if kubectl wait --for=condition=ready pod/test-pod --timeout=600s; then
    kill $PROGRESS_PID 2>/dev/null || true
    wait $PROGRESS_PID 2>/dev/null || true
    echo ""
    echo "✅ Services are ready!"
    kubectl get pod/test-pod
    kubectl get service/test-pod
else
    kill $PROGRESS_PID 2>/dev/null || true
    wait $PROGRESS_PID 2>/dev/null || true
    echo ""
    echo "❌ Error: Pod failed to become ready within 10 minutes"
    echo ""
    echo "Pod description:"
    kubectl describe pod/test-pod
    echo ""
    echo "Container logs:"
    kubectl logs test-pod --all-containers=true || true
    exit 1
fi

# Verify service endpoints are available
echo ""
echo "Verifying service endpoints..."
kubectl get endpoints test-pod

# Step 7: Create test job
echo ""
echo "Step 7: Creating test job..."
# Podman tags images with localhost/ prefix, so we need to use the full reference
FULL_IMAGE_NAME="localhost/${IMAGE_NAME}"
sed "s|{{FULL_IMAGE_NAME}}|${FULL_IMAGE_NAME}|g" test/test-job.yml | kubectl apply -f -

# Step 8: Follow test logs in real-time
echo ""
echo "Step 8: Following test execution (streaming logs)..."
echo "Waiting for test job pod to start..."

# Wait for the job pod to be created and start running
for i in {1..30}; do
    POD_NAME=$(kubectl get pods -l app=gabi-integration-test -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
    if [ -n "$POD_NAME" ]; then
        POD_STATUS=$(kubectl get pod "$POD_NAME" -o jsonpath='{.status.phase}' 2>/dev/null || echo "")
        if [ "$POD_STATUS" = "Running" ] || [ "$POD_STATUS" = "Succeeded" ] || [ "$POD_STATUS" = "Failed" ]; then
            echo "Test pod started: $POD_NAME"
            break
        fi
    fi
    if [ "$i" -eq 30 ]; then
        echo "⚠️  Warning: Test pod did not start within 60 seconds"
        echo "Attempting to show logs anyway..."
    fi
    sleep 2
done

echo ""
echo "=== Test Output (streaming) ==="
# Follow logs in real-time until the pod completes
kubectl logs -f job/gabi-integration-test-job 2>&1 || true

echo ""
echo "=== Test execution finished ==="

# Step 9: Check test results
echo ""
echo "Step 9: Checking test results..."
echo "Waiting for job status to be updated..."

# Wait for the job to have a completion status (Kubernetes needs time to update after pod finishes)
for i in {1..30}; do
    JOB_COMPLETE=$(kubectl get job gabi-integration-test-job -o jsonpath='{.status.conditions[?(@.type=="Complete")].status}' 2>/dev/null || echo "")
    JOB_FAILED=$(kubectl get job gabi-integration-test-job -o jsonpath='{.status.conditions[?(@.type=="Failed")].status}' 2>/dev/null || echo "")

    if [ "$JOB_COMPLETE" = "True" ] || [ "$JOB_FAILED" = "True" ]; then
        break
    fi

    if [ "$i" -eq 30 ]; then
        echo "⚠️  Warning: Job status not updated after 60 seconds"
        break
    fi

    sleep 2
done

if [ "$JOB_COMPLETE" = "True" ]; then
    echo "✅ Integration tests PASSED!"
    EXIT_STATUS=0
elif [ "$JOB_FAILED" = "True" ]; then
    echo "❌ Integration tests FAILED"
    EXIT_STATUS=1
else
    echo "⚠️  Integration tests did not complete normally"
    EXIT_STATUS=1
fi

# Step 10: Cleanup (optional)
echo ""
read -p "Do you want to clean up the test resources? (y/N) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "Cleaning up..."
    kubectl delete job/gabi-integration-test-job --ignore-not-found=true
    kubectl delete pod/test-pod --ignore-not-found=true
    kubectl delete service/test-pod --ignore-not-found=true
    echo "Cleanup complete!"
fi

echo ""
echo "To delete the kind cluster, run: kind delete cluster --name ${CLUSTER_NAME}"

exit $EXIT_STATUS

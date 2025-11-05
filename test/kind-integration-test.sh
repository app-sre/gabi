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

# Force Kind to use podman if KIND_EXPERIMENTAL_PROVIDER is not already set
# This is useful when both docker and podman are installed
export KIND_EXPERIMENTAL_PROVIDER="${KIND_EXPERIMENTAL_PROVIDER:-podman}"
echo "Using container runtime: ${KIND_EXPERIMENTAL_PROVIDER}"
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

if ! command -v podman &> /dev/null; then
    echo "Error: podman is not installed. Please install podman first."
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
podman build -t "${IMAGE_NAME}" -f Dockerfile.integration .

# Step 3: Pull supporting service images
echo ""
echo "Step 3: Pulling supporting service images..."
echo "  - Pulling PostgreSQL image..."
podman pull registry.redhat.io/rhel9/postgresql-16:9.6

# Step 4: Load images into kind cluster
echo ""
echo "Step 4: Loading images into kind cluster..."
echo "  - Removing old integration test images from kind cluster..."
# Remove old images to prevent confusion between docker.io and localhost references
podman exec "${CLUSTER_NAME}-control-plane" crictl rmi "docker.io/library/${IMAGE_NAME}" 2>/dev/null || true
podman exec "${CLUSTER_NAME}-control-plane" crictl rmi "localhost/${IMAGE_NAME}" 2>/dev/null || true
podman exec "${CLUSTER_NAME}-control-plane" crictl rmi "${IMAGE_NAME}" 2>/dev/null || true

echo "  - Loading fresh integration test image..."
# Use podman save/import method which is more reliable for podman->kind workflow
podman save "${IMAGE_NAME}" -o /tmp/gabi-test-image.tar
podman cp /tmp/gabi-test-image.tar "${CLUSTER_NAME}-control-plane:/gabi-test-image.tar"
podman exec "${CLUSTER_NAME}-control-plane" ctr -n k8s.io images import /gabi-test-image.tar
podman exec "${CLUSTER_NAME}-control-plane" rm /gabi-test-image.tar
rm /tmp/gabi-test-image.tar

echo "  - Loading PostgreSQL image..."
kind load docker-image registry.redhat.io/rhel9/postgresql-16:9.6 --name "${CLUSTER_NAME}"

# Verify the integration test image was loaded correctly
echo ""
echo "Verifying integration test image in cluster:"
podman exec "${CLUSTER_NAME}-control-plane" crictl images | grep gabi || echo "Warning: gabi image not found!"

# Step 5: Deploy supporting services (database and mock-splunk)
echo ""
echo "Step 5: Deploying database and mock-splunk..."
# Set the mock-splunk image to the locally built image
export MOCK_SPLUNK_IMAGE="localhost/${IMAGE_NAME}"
echo "Using mock-splunk image: ${MOCK_SPLUNK_IMAGE}"
envsubst < test/test-pod.yml | kubectl apply -f -

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

# Step 7: Create a temporary pod manifest for running tests
echo ""
echo "Step 7: Creating test runner pod..."
# Podman tags images with localhost/ prefix, so we need to use the full reference
FULL_IMAGE_NAME="localhost/${IMAGE_NAME}"
cat > /tmp/gabi-integration-test-pod.yaml <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: gabi-integration-test-runner
  labels:
    app: gabi-integration-test
spec:
  restartPolicy: Never
  containers:
  - name: test-runner
    image: ${FULL_IMAGE_NAME}
    imagePullPolicy: Never
    env:
    # Database configuration (pointing to test-pod)
    - name: DB_DRIVER
      value: "pgx"
    - name: DB_HOST
      value: "test-pod"
    - name: DB_PORT
      value: "5432"
    - name: DB_USER
      value: "gabi"
    - name: DB_PASS
      value: "passwd"
    - name: DB_NAME
      value: "mydb"
    - name: DB_WRITE
      value: "false"
    # Splunk configuration (pointing to test-pod)
    # Note: SPLUNK_TOKEN is created dynamically by tests
    - name: SPLUNK_HOST
      value: "test-pod"
    - name: SPLUNK_INDEX
      value: "main"
    # Other configuration
    - name: HOST
      value: "test"
    - name: NAMESPACE
      value: "${NAMESPACE}"
    - name: POD_NAME
      value: "gabi-integration-test-runner"
    # Test timeout
    - name: INTEGRATION_TEST_TIMEOUT
      value: "10m"
    # Use shell to add startup delay for DNS/network propagation
    # This allows pod networking/DNS to fully initialize before tests start
    # Run each test separately to avoid port 8080 conflicts between tests
    command: ["/bin/sh"]
    args:
    - "-c"
    - |
      sleep 10
      echo "Starting integration tests - running each test separately to avoid port conflicts"

      # Get list of tests (avoid pipe to preserve exit codes)
      test_list=\$(/usr/local/bin/integration.test -test.list '.*' | grep '^Test')

      # Track results
      total_tests=0
      passed_tests=0
      failed_tests=0
      failed_test_names=""

      # Run each test (continue even if one fails)
      for test in \$test_list; do
        total_tests=\$((total_tests + 1))
        echo ""
        echo "=========================================="
        echo "Running test: \${test}"
        echo "=========================================="
        if /usr/local/bin/integration.test -test.run "^\${test}\$" -test.v -test.timeout=5m; then
          echo "✓ \${test} PASSED"
          passed_tests=\$((passed_tests + 1))
        else
          echo "✗ \${test} FAILED"
          failed_tests=\$((failed_tests + 1))
          failed_test_names="\${failed_test_names}  - \${test}\n"
        fi
      done

      # Summary
      echo ""
      echo "=========================================="
      echo "TEST SUMMARY"
      echo "=========================================="
      echo "Total tests: \${total_tests}"
      echo "Passed: \${passed_tests}"
      echo "Failed: \${failed_tests}"
      echo ""

      if [ \${failed_tests} -gt 0 ]; then
        echo "Failed tests:"
        echo -e "\${failed_test_names}"
        echo "=========================================="
        exit 1
      else
        echo "All tests passed! ✅"
        echo "=========================================="
        exit 0
      fi
EOF

# Step 8: Run the integration tests
echo ""
echo "Step 8: Running integration tests..."
kubectl apply -f /tmp/gabi-integration-test-pod.yaml

# Step 9: Follow test logs
echo ""
echo "Step 9: Watching test execution..."
echo "Waiting for test pod container to start..."

# Wait for pod to be running (not just initialized)
until kubectl get pod gabi-integration-test-runner -o jsonpath='{.status.phase}' 2>/dev/null | grep -q "Running"; do
    echo "Waiting for container to start..."
    sleep 2
done

echo "Container is running, following logs..."
echo ""
echo "=== Test Output ==="
kubectl logs -f gabi-integration-test-runner || true



# Step 10: Check test results
echo ""
echo "Step 10: Checking test results..."
TEST_EXIT_CODE=$(kubectl get pod gabi-integration-test-runner -o jsonpath='{.status.containerStatuses[0].state.terminated.exitCode}' 2>/dev/null || echo "unknown")

if [ "$TEST_EXIT_CODE" = "0" ]; then
    echo "✅ Integration tests PASSED!"
    EXIT_STATUS=0
else
    echo "❌ Integration tests FAILED (exit code: ${TEST_EXIT_CODE})"
    EXIT_STATUS=1
fi

# Step 11: Cleanup (optional)
echo ""
read -p "Do you want to clean up the test resources? (y/N) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "Cleaning up..."
    kubectl delete pod/gabi-integration-test-runner --ignore-not-found=true
    envsubst < test/test-pod.yml | kubectl delete -f - --ignore-not-found=true
    echo "Cleanup complete!"
fi

echo ""
echo "To delete the kind cluster, run: kind delete cluster --name ${CLUSTER_NAME}"

exit $EXIT_STATUS

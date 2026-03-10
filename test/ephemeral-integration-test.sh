#!/usr/bin/env bash
#
# Integration test runner using oc CLI
# Works on both local Kind clusters and OpenShift clusters
#

set -o errexit
set -o nounset
set -o pipefail

echo "=== Gabi Integration Test ==="
echo ""

# Configuration
IMAGE_URL="${IMAGE_URL:-localhost/gabi-integration-test:local}"
IMAGE_DIGEST="${IMAGE_DIGEST:-}"

echo "Image URL: ${IMAGE_URL}"
if [ -n "${IMAGE_DIGEST}" ]; then
    echo "Image Digest: ${IMAGE_DIGEST}"
fi
echo ""

# Check prerequisites
if ! command -v oc &> /dev/null; then
    echo "Error: oc is not installed. Please install oc CLI first."
    exit 1
fi

# Step 1: Create wiremock mappings ConfigMap
echo ""
echo "Step 1: Creating wiremock mappings ConfigMap..."
oc create configmap wiremock-mappings --from-file=test/wiremock/mappings

# Step 2 Deploy supporting services (database and mock-splunk)
echo ""
echo "Step 2 Deploying database and mock-splunk..."
cd "$(dirname "$0")/.."
oc apply -f test/test-pods.yml

# Step 3: Wait for supporting services to be ready
echo ""
echo "Step 3: Waiting for services to be ready..."
echo "Waiting for postgres-test-pod and mock-splunk-test-pod to be ready..."
echo ""

# Show progress while waiting for both pods
(
  while true; do
    POSTGRES_READY=$(oc get pod/postgres-test-pod -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || echo "Unknown")
    SPLUNK_READY=$(oc get pod/mock-splunk-test-pod -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || echo "Unknown")

    if [ "$POSTGRES_READY" = "True" ] && [ "$SPLUNK_READY" = "True" ]; then
      break
    fi

    # Show container status
    echo "$(date '+%H:%M:%S') - Pod statuses:"
    echo -n "  postgres-test-pod: "
    oc get pod/postgres-test-pod -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown"
    echo -n " (ready: $POSTGRES_READY) - Containers: "
    oc get pod/postgres-test-pod -o jsonpath='{range .status.containerStatuses[*]}{.name}={.ready} {end}' 2>/dev/null || echo "checking..."
    echo ""

    echo -n "  mock-splunk-test-pod: "
    oc get pod/mock-splunk-test-pod -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown"
    echo -n " (ready: $SPLUNK_READY) - Containers: "
    oc get pod/mock-splunk-test-pod -o jsonpath='{range .status.containerStatuses[*]}{.name}={.ready} {end}' 2>/dev/null || echo "checking..."
    echo ""
    echo ""

    sleep 10
  done
) &
PROGRESS_PID=$!

# Wait for both pods to be ready (increased timeout for Splunk)
POSTGRES_READY=false
SPLUNK_READY=false

if oc wait --for=condition=ready pod/postgres-test-pod --timeout=60s 2>/dev/null; then
    POSTGRES_READY=true
fi

if oc wait --for=condition=ready pod/mock-splunk-test-pod --timeout=60s 2>/dev/null; then
    SPLUNK_READY=true
fi

kill $PROGRESS_PID 2>/dev/null || true
wait $PROGRESS_PID 2>/dev/null || true

if [ "$POSTGRES_READY" = "true" ] && [ "$SPLUNK_READY" = "true" ]; then
    echo ""
    echo "✅ All services are ready!"
    oc get pod/postgres-test-pod
    oc get pod/mock-splunk-test-pod
    oc get service/postgres-service
    oc get service/mock-splunk-service
else
    echo ""
    echo "❌ Error: One or more pods failed to become ready within 60 seconds"
    echo ""

    if [ "$POSTGRES_READY" = "false" ]; then
        echo "postgres-test-pod description:"
        oc describe pod/postgres-test-pod
        echo ""
        echo "postgres-test-pod logs:"
        oc logs postgres-test-pod --all-containers=true || true
        echo ""
    fi

    if [ "$SPLUNK_READY" = "false" ]; then
        echo "mock-splunk-test-pod description:"
        oc describe pod/mock-splunk-test-pod
        echo ""
        echo "mock-splunk-test-pod logs:"
        oc logs mock-splunk-test-pod --all-containers=true || true
    fi

    exit 1
fi

# Verify service endpoints are available
echo ""
echo "Verifying service endpoints..."
oc get endpoints postgres-service
oc get endpoints mock-splunk-service

# Step 4: Create test job
echo ""
echo "Step 4: Creating test job..."
# Use the full image URL
FULL_IMAGE_NAME="${IMAGE_URL}"
sed "s|{{FULL_IMAGE_NAME}}|${FULL_IMAGE_NAME}|g" test/test-job.yml | oc apply -f -

# Step 5: Follow test logs in real-time
echo ""
echo "Step 5: Following test execution (streaming logs)..."
echo "Waiting for test job pod to start..."

# Wait for the job pod to be created and start running
for i in {1..30}; do
    POD_NAME=$(oc get pods -l app=gabi-integration-test -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
    if [ -n "$POD_NAME" ]; then
        POD_STATUS=$(oc get pod "$POD_NAME" -o jsonpath='{.status.phase}' 2>/dev/null || echo "")
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
oc logs -f job/gabi-integration-test-job 2>&1 || true

echo ""
echo "=== Test execution finished ==="

# Step 6: Check test results
echo ""
echo "Step 6: Checking test results..."
echo "Waiting for job status to be updated..."

# Wait for the job to have a completion status (Kubernetes needs time to update after pod finishes)
for i in {1..30}; do
    JOB_COMPLETE=$(oc get job gabi-integration-test-job -o jsonpath='{.status.conditions[?(@.type=="Complete")].status}' 2>/dev/null || echo "")
    JOB_FAILED=$(oc get job gabi-integration-test-job -o jsonpath='{.status.conditions[?(@.type=="Failed")].status}' 2>/dev/null || echo "")

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

echo ""
echo "To clean up test resources, run: make integration-test-clean"

exit $EXIT_STATUS

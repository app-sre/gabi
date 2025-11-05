#!/usr/bin/env bash
#
# Integration test runner for Konflux ephemeral namespace
#

set -o errexit
set -o nounset
set -o pipefail

echo "=== Gabi Integration Test on Konflux ephemeral namespace ==="
echo ""

echo "Image URL: ${IMAGE_URL}"
echo "Image Digest: ${IMAGE_DIGEST}"

# Step 1: Deploy supporting services (database and mock-splunk)
echo ""
echo "Step 1: Deploying database and mock-splunk..."

# Set the mock-splunk image to the locally built image
MOCK_SPLUNK_IMAGE="${IMAGE_URL}"
echo "Using mock-splunk image: ${MOCK_SPLUNK_IMAGE}"
sed "s|{{MOCK_SPLUNK_IMAGE}}|${MOCK_SPLUNK_IMAGE}|g" test/test-pod.yml | oc apply -f -

# Step 2: Wait for supporting services to be ready
echo ""
echo "Step 2: Waiting for services to be ready..."
echo "Waiting for test-pod to be ready..."
echo ""

# Show progress while waiting
(
  while oc get pod/test-pod -o json 2>/dev/null | jq -e '.status.conditions[] | select(.type=="Ready" and .status=="False")' > /dev/null 2>&1; do
    # Show container status
    echo -n "$(date '+%H:%M:%S') - Pod status: "
    oc get pod/test-pod -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown"

    # Show readiness status for each container
    echo -n "  Containers ready: "
    oc get pod/test-pod -o jsonpath='{range .status.containerStatuses[*]}{.name}={.ready} {end}' 2>/dev/null || echo "checking..."
    echo ""

    sleep 10
  done
) &
PROGRESS_PID=$!

# Wait for pod to be ready (increased timeout for Splunk)
if oc wait --for=condition=ready pod/test-pod --timeout=60s; then
    kill $PROGRESS_PID 2>/dev/null || true
    wait $PROGRESS_PID 2>/dev/null || true
    echo ""
    echo "✅ Services are ready!"
    oc get pod/test-pod
    oc get service/test-pod
else
    kill $PROGRESS_PID 2>/dev/null || true
    wait $PROGRESS_PID 2>/dev/null || true
    echo ""
    echo "❌ Error: Pod failed to become ready within 60 seconds"
    echo ""
    echo "Pod description:"
    oc describe pod/test-pod
    echo ""
    echo "Container logs:"
    oc logs test-pod --all-containers=true || true
    exit 1
fi

# Verify service endpoints are available
echo ""
echo "Verifying service endpoints..."
oc get endpoints test-pod

# Step 3: Create a temporary pod manifest for running tests
echo ""
echo "Step 3: Creating test runner pod..."
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
    image: ${IMAGE_URL}
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
      value: "test"
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

# Step 4: Run the integration tests
echo ""
echo "Step 4: Running integration tests..."
oc apply -f /tmp/gabi-integration-test-pod.yaml

# Step 5: Follow test logs
echo ""
echo "Step 5: Watching test execution..."
echo "Waiting for test pod container to start..."

# Wait for pod to be running (not just initialized)
until oc get pod gabi-integration-test-runner -o jsonpath='{.status.phase}' 2>/dev/null | grep -q "Running"; do
    echo "Waiting for container to start..."
    sleep 2
done

echo "Container is running, following logs..."
echo ""
echo "=== Test Output ==="
oc logs -f gabi-integration-test-runner || true



# Step 6: Check test results
echo ""
echo "Step 6: Checking test results..."
TEST_EXIT_CODE=$(oc get pod gabi-integration-test-runner -o jsonpath='{.status.containerStatuses[0].state.terminated.exitCode}' 2>/dev/null || echo "unknown")

if [ "$TEST_EXIT_CODE" = "0" ]; then
    echo "✅ Integration tests PASSED!"
    EXIT_STATUS=0
else
    echo "❌ Integration tests FAILED (exit code: ${TEST_EXIT_CODE})"
    EXIT_STATUS=1
fi

echo "Cleaning up..."
oc delete pod/gabi-integration-test-runner --ignore-not-found=true
envsubst < test/test-pod.yml | oc delete -f - --ignore-not-found=true
echo "Cleanup complete!"

exit $EXIT_STATUS

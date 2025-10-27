#!/bin/bash
# Create a test runner pod that stays alive for debugging

set -e

CLUSTER_NAME="${CLUSTER_NAME:-gabi-integration}"
IMAGE_NAME="${IMAGE_NAME:-gabi-integration-test:local}"
NAMESPACE="${NAMESPACE:-default}"

echo "=== Creating Interactive Test Pod for Debugging ==="
echo ""

# Delete existing pod if present
kubectl delete pod gabi-test-interactive --ignore-not-found=true

echo "Creating test pod with same environment as integration tests..."
echo ""

# Podman tags images with localhost/ prefix, so we need to use the full reference
FULL_IMAGE_NAME="localhost/${IMAGE_NAME}"

# Create the pod
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: gabi-test-interactive
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
      value: "gabi-test-interactive"
    # Keep pod alive for debugging
    command: ["sleep"]
    args: ["infinity"]
EOF

echo "Waiting for pod to be ready..."
kubectl wait --for=condition=ready pod/gabi-test-interactive --timeout=60s

echo ""
echo "✓ Pod is ready!"
echo ""
echo "=== Environment Check ==="
echo ""

echo "Database Configuration:"
kubectl exec gabi-test-interactive -- env | grep "^DB_"

echo ""
echo "Splunk Configuration:"
kubectl exec gabi-test-interactive -- env | grep "^SPLUNK_"

echo ""
echo "=== Connectivity Tests ==="
echo ""

echo -n "DNS resolution: "
if kubectl exec gabi-test-interactive -- nslookup test-pod &>/dev/null; then
    echo "✓ OK"
else
    echo "✗ FAILED"
fi

echo -n "Port 5432: "
if kubectl exec gabi-test-interactive -- nc -zv test-pod 5432 2>&1 | grep -q "open"; then
    echo "✓ OPEN"
else
    echo "✗ CLOSED"
fi

echo -n "PostgreSQL connection: "
if kubectl exec gabi-test-interactive -- sh -c 'PGPASSWORD=passwd psql -h test-pod -U gabi -d mydb -c "SELECT 1;" 2>&1' | grep -q "1 row"; then
    echo "✓ SUCCESS"
else
    echo "✗ FAILED"
    kubectl exec gabi-test-interactive -- sh -c 'PGPASSWORD=passwd psql -h test-pod -U gabi -d mydb -c "SELECT 1;" 2>&1' || true
fi

echo ""
echo "=== Now You Can Debug ==="
echo ""
echo "Run a single test:"
echo "  kubectl exec -it gabi-test-interactive -- /usr/local/bin/integration.test -test.run TestHealthCheckOK -test.v"
echo ""
echo "Run all tests (separately to avoid port conflicts):"
echo "  kubectl exec -it gabi-test-interactive -- sh -c '"
echo "    /usr/local/bin/integration.test -test.list \".*\" | grep \"^Test\" | while read test; do"
echo "      echo \"Running: \$test\""
echo "      /usr/local/bin/integration.test -test.run \"^\$test\$\" -test.v || exit 1"
echo "    done"
echo "  '"
echo ""
echo "Open interactive shell:"
echo "  kubectl exec -it gabi-test-interactive -- /bin/sh"
echo ""
echo "View environment:"
echo "  kubectl exec gabi-test-interactive -- env"
echo ""
echo "Clean up when done:"
echo "  kubectl delete pod gabi-test-interactive"
echo ""

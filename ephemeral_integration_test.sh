#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

echo "Running ephemeral integration test"

echo "Image URL: ${IMAGE_URL}"
echo "Image Digest: ${IMAGE_DIGEST}"

echo "Getting current namespace..."
NAMESPACE=$(oc project -q)
echo "Namespace: ${NAMESPACE}"

echo "Creating pod YAML..."
oc process -f test/test-pod-template.yml -o yaml \
    -p IMAGE_URL="${IMAGE_URL}" \
    -p POD_NAME="my-test-app" \
    > /tmp/my-pod.yaml

echo ""
echo "=== Generated Pod YAML ==="
cat /tmp/my-pod.yaml

oc apply -f /tmp/my-pod.yaml

echo ""
echo "Waiting for pod to be ready..."
oc wait --for=condition=ready pod/my-test-app --timeout=300s

echo ""
echo "Getting pod information..."
oc get pods my-test-app

echo ""
echo "Getting service information..."
oc get service my-test-app-internal

echo ""
echo "Waiting a moment for DNS propagation..."
sleep 3

echo ""
echo "Testing pod with curl..."
curl --silent --show-error --fail "http://my-test-app-internal.${NAMESPACE}.svc.cluster.local:8080/healthcheck"

echo ""
echo "Cleaning up..."
oc delete pod/my-test-app
oc delete service/my-test-app-internal

exit 0
#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

echo "Running ephemeral integration test"

echo "Image URL: ${IMAGE_URL}"
echo "Image Digest: ${IMAGE_DIGEST}"

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
echo "Testing pod with curl..."
oc exec my-test-app -c app -- curl --silent --show-error --fail "http://localhost:8080/healthcheck"

echo ""
echo "Cleaning up..."
oc delete pod/my-test-app

exit 0
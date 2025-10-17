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

echo "Waiting for pod to be ready..."
oc wait --for=condition=ready pod/my-test-app --timeout=300s

echo "Getting pod information..."
oc get pods my-test-app -o wide

echo "Extracting pod IP..."
POD_IP=$(oc get pod my-test-app -o jsonpath='{.status.podIP}')
echo "Pod IP: ${POD_IP}"

echo "Testing pod with curl..."
curl -v "http://${POD_IP}:8080/healthcheck"

echo "Cleaning up..."
oc delete pod/my-test-app

exit 0
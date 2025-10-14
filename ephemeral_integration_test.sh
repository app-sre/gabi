#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

echo "Running ephemeral integration test"

echo "Image URL: ${IMAGE_URL}"
echo "Image Digest: ${IMAGE_DIGEST}"

exit 0
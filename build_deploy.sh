#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

export PATH="/opt/go/1.19.11/bin:${PATH}"

readonly QUAY_IMAGE="quay.io/app-sre/gabi"

GIT_HASH=$(git rev-parse --short=7 HEAD)

{
    set +x
    podman login quay.io -u "${QUAY_USER}" -p "${QUAY_TOKEN}"
}

podman build -t "${QUAY_IMAGE}:check" -f Dockerfile .

podman tag "${IMAGE_NAME}:latest" "${IMAGE_NAME}:${GIT_HASH}" 

podman push "${IMAGE_NAME}:latest" 
podman push "${IMAGE_NAME}:${GIT_HASH}" 

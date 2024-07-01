#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

export PATH="/opt/go/1.19.11/bin:${PATH}"

BASE_IMAGE="gabi"
QUAY_IMAGE="quay.io/app-sre/${BASE_IMAGE}"

TARGET_IMAGE="${BASE_IMAGE}:latest"

readonly BASE_IMAGE QUAY_IMAGE TARGET_IMAGE

GIT_HASH=$(git rev-parse --short=7 HEAD)

{
    set +x
    podman login quay.io -u "${QUAY_USER}" -p "${QUAY_TOKEN}"
}

BUILD_CMD="podman build" IMG="${TARGET_IMAGE}" make docker-build

{
    set +x
    skopeo copy --dest-creds "${QUAY_USER}:${QUAY_TOKEN}" \
        "${TARGET_IMAGE}" \
        "docker://${QUAY_IMAGE}:latest"

    skopeo copy --dest-creds "${QUAY_USER}:${QUAY_TOKEN}" \
        "${TARGET_IMAGE}" \
        "docker://${QUAY_IMAGE}:${GIT_HASH}"
}

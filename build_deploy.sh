#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

export PATH="/opt/go/1.19.5/bin:${PATH}"

BASE_IMG="gabi"
QUAY_IMAGE="quay.io/app-sre/${BASE_IMG}"
IMG="${BASE_IMG}:latest"

readonly BASE_IMG QUAY_IMAGE IMG

GIT_HASH=$(git rev-parse --short=7 HEAD)

{
    set +x
    docker login quay.io -u "${QUAY_USER}" -p "${QUAY_TOKEN}"
}

BUILD_CMD="docker build" IMG="${IMG}" make docker-build

{
    set +x
    skopeo copy --dest-creds "${QUAY_USER}:${QUAY_TOKEN}" \
        "docker-daemon:${IMG}" \
        "docker://${QUAY_IMAGE}:latest"

    skopeo copy --dest-creds "${QUAY_USER}:${QUAY_TOKEN}" \
        "docker-daemon:${IMG}" \
        "docker://${QUAY_IMAGE}:${GIT_HASH}"
}

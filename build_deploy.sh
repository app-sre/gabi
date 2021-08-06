#!/bin/bash

# AppSRE team CD

set -exv

BASE_IMG="gabi"
QUAY_IMAGE="quay.io/app-sre/${BASE_IMG}"
IMG="${BASE_IMG}:latest"

GIT_HASH=`git rev-parse --short=7 HEAD`

# build the image
docker login quay.io -u ${QUAY_USER} -p ${QUAY_TOKEN}

BUILD_CMD="docker build" IMG="$IMG" make docker-build

# push the image to quay
skopeo copy --dest-creds "${QUAY_USER}:${QUAY_TOKEN}" \
    "docker-daemon:${IMG}" \
    "docker://${QUAY_IMAGE}:latest"

skopeo copy --dest-creds "${QUAY_USER}:${QUAY_TOKEN}" \
    "docker-daemon:${IMG}" \
    "docker://${QUAY_IMAGE}:${GIT_HASH}"

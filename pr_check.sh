#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

export PATH="/opt/go/1.19.5/bin:${PATH}"

readonly BASE_IMG="gabi"

{
    set +x
    docker login quay.io -u "${QUAY_USER}" -p "${QUAY_TOKEN}"
}

go test ./...

./integration.sh

BUILD_CMD="docker build" IMG="${BASE_IMG}:check" make docker-build

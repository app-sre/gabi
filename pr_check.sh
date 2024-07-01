#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

export PATH="/opt/go/1.19.11/bin:${PATH}"

readonly BASE_IMG="gabi"

{
    set +x
    podman login quay.io -u "${QUAY_USER}" -p "${QUAY_TOKEN}"
}

go test ./...

podman pull quay.io/app-sre/gnomock-cleaner:latest
podman tag quay.io/app-sre/gnomock-cleaner:latest docker.io/orlangure/gnomock-cleaner:latest

podman system service -t 0 & ./integration.sh

BUILD_CMD="podman build" IMG="${BASE_IMG}:check" make docker-build

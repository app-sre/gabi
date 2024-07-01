#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

export PATH="/opt/go/1.19.11/bin:${PATH}"

readonly QUAY_IMAGE="quay.io/app-sre/gabi"

{
    set +x
    podman login quay.io -u "${QUAY_USER}" -p "${QUAY_TOKEN}"
}

go test ./...

podman pull quay.io/app-sre/gnomock-cleaner:latest
podman tag quay.io/app-sre/gnomock-cleaner:latest docker.io/orlangure/gnomock-cleaner:latest

podman system service -t 0 & ./integration.sh

podman build -t "${QUAY_IMAGE}:check" -f Dockerfile .

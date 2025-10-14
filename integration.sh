#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

export PATH="/opt/go/1.22.5/bin:${PATH}"

export QUAY_TOKEN

export DOCKER_HOST=unix:///run/user/${UID}/podman/podman.sock

# Pull these images using the auth in the jenkins nodes
podman pull registry.redhat.io/rhel9/postgresql-16:9.6
podman pull quay.io/app-sre/splunk:latest
grep -oE 'Test[A-Za-z0-9]{,}' test/integration_test.go | while read -r test; do
    echo "Running test: ${test}"
    go test -tags integration -count 1 -timeout 300s -run "^${test}$" ./test/...
done

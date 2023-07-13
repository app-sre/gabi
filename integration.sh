#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

export PATH="/opt/go/1.19.11/bin:${PATH}"

{
    set +x
    QUAY_TOKEN=$(cat <<-EOF | base64 | tr -d '\n'
{"username":"${QUAY_USER}","password":"${QUAY_TOKEN}"}
EOF
)
}

export QUAY_TOKEN

grep -oE 'Test[A-Za-z0-9]{,}' test/integration_test.go | while read -r test; do
    echo "Running test: ${test}"
    go test -tags integration -count 1 -timeout 300s -run "^${test}$" ./test/...
done

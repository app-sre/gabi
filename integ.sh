
#!/bin/bash
set -exv

export PATH=/opt/go/1.18.1/bin:$PATH

set +x
export QUAY_TOKEN=$(echo \{\"username\"\:\"${QUAY_USER}\" \, \"password\"\:\"${QUAY_TOKEN}\"\} |base64 |tr -d '\n')
set -x

for t in $(grep 'func Test' test/integration_test.go |grep -oE 'Test[A-Za-z0-9]{,}')
do
    go test -v -count=1 -timeout 500s -run ^${t}\$ ./...
done

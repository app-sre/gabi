# Integration Testing in Kind Cluster

This guide explains how to run integration tests for gabi in a local Kind cluster.

## Overview

The integration test setup consists of:
1. **Database Pod** (PostgreSQL) - provides database services
2. **Splunk Pod** - provides logging/auditing services
3. **Test Runner Pod** - runs the compiled integration tests

## Prerequisites

- Podman (or Docker)
- Kind (Kubernetes in Docker/Podman): https://kind.sigs.k8s.io/
- kubectl
- Go 1.22+ (for local development)

Install kind:
```bash
# On Linux
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
chmod +x ./kind
sudo mv ./kind /usr/local/bin/kind

# On macOS
brew install kind
```

## Quick Start

Run the automated test script:

```bash
./test/kind-integration-test.sh
```

This script will:
1. Create a kind cluster (if needed)
2. Build the integration test container image (using podman)
3. Pull supporting service images (PostgreSQL, Splunk)
4. Load all images into kind
5. Deploy database and splunk services
6. Run the integration tests
7. Show test results
8. Optionally clean up resources

## Manual Steps

### 1. Create Kind Cluster

```bash
kind create cluster --name gabi-integration
```

### 2. Build Integration Test Image

```bash
podman build -t gabi-integration-test:local -f test/Dockerfile.integration .
```

### 3. Pull Supporting Service Images

```bash
podman pull registry.redhat.io/rhel9/postgresql-16:9.6
podman pull quay.io/app-sre/splunk:latest
```

### 4. Load All Images into Kind

```bash
kind load docker-image gabi-integration-test:local --name gabi-integration
kind load docker-image registry.redhat.io/rhel9/postgresql-16:9.6 --name gabi-integration
kind load docker-image quay.io/app-sre/splunk:latest --name gabi-integration
```

### 5. Deploy Supporting Services

The `test-pod.yml` creates both a **Service** and a **Pod**:

```bash
kubectl apply -f test/test-pod.yml
kubectl wait --for=condition=ready pod/test-pod --timeout=300s

# Verify service was created
kubectl get service test-pod
kubectl get endpoints test-pod
```

**Why a Service?** Kubernetes pods need a Service for DNS-based communication. The Service named `test-pod` allows the test runner to connect using the hostname `test-pod:5432` (PostgreSQL) and `test-pod:8088` (Splunk)

### 6. Create and Run Test Pod

Create a file `test-runner-pod.yaml`:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: gabi-integration-test-runner
spec:
  restartPolicy: Never
  containers:
  - name: test-runner
    image: gabi-integration-test:local
    imagePullPolicy: Never
    env:
    - name: DB_HOST
      value: "test-pod"
    - name: DB_PORT
      value: "5432"
    - name: DB_USER
      value: "gabi"
    - name: DB_PASS
      value: "passwd"
    - name: DB_NAME
      value: "mydb"
    - name: SPLUNK_ENDPOINT
      value: "http://test-pod:8088"
```

Deploy and watch:

```bash
kubectl apply -f test-runner-pod.yaml
kubectl logs -f gabi-integration-test-runner
```

### 7. Check Results

```bash
kubectl get pod gabi-integration-test-runner -o jsonpath='{.status.containerStatuses[0].state.terminated.exitCode}'
```

Exit code 0 = tests passed

## Debugging Tools

The integration test image includes these debugging tools:

| Tool | Purpose | Example |
|------|---------|---------|
| `psql` | PostgreSQL client | `psql -h test-pod -U gabi -d mydb` |
| `nc` (ncat) | Network connectivity testing | `nc -zv test-pod 5432` |
| `nslookup` | DNS resolution testing | `nslookup test-pod` |
| `dig` | Advanced DNS queries | `dig test-pod` |
| `curl` | HTTP/HTTPS requests | `curl http://test-pod:8088` |

### Using Debugging Tools

⚠️ **Important**: If tests fail, the test runner pod stops and you can't exec into it. Create a debug pod instead:

```bash
# Create a debug pod with the same image
kubectl run gabi-debug \
  --image=gabi-integration-test:local \
  --image-pull-policy=Never \
  --restart=Never \
  --env="DB_HOST=test-pod" \
  --env="DB_PORT=5432" \
  --command -- sleep infinity

kubectl wait --for=condition=ready pod/gabi-debug --timeout=60s

# Now you can use all debugging tools:

# Test database connectivity
kubectl exec -it gabi-debug -- nc -zv test-pod 5432

# Query database
kubectl exec -it gabi-debug -- sh -c "PGPASSWORD=passwd psql -h test-pod -U gabi -d mydb -c 'SELECT 1;'"

# Check DNS resolution
kubectl exec -it gabi-debug -- nslookup test-pod

# Test Splunk endpoint
kubectl exec -it gabi-debug -- curl http://test-pod:8088/services/collector/health

# Clean up when done
kubectl delete pod/gabi-debug
```

See `test/DEBUGGING-TOOLS.md` for complete debugging guide.

## Important Notes

### ✅ Tests Are Cluster-Ready

The integration tests in `integration_test.go` have been **updated to work with deployed services**:

- ✅ No longer require **gnomock/testcontainers**
- ✅ Connect to services via **environment variables**
- ✅ Work in **Kubernetes pods** without special privileges
- ✅ Can still run **locally** with services on localhost

### How It Works

Tests now use the `getEnvOrDefault()` helper to read connection information:

```go
func TestHealthCheckOK(t *testing.T) {
    // Reads from environment or uses sensible defaults
    dbHost := getEnvOrDefault("DB_HOST", "localhost")
    dbPort := getEnvOrDefault("DB_PORT", "5432")
    splunkEndpoint := getEnvOrDefault("SPLUNK_ENDPOINT", "http://localhost:8088")
    splunkToken := getEnvOrDefault("SPLUNK_TOKEN", "test123")

    // Rest of test uses these values
    setEnvironment(configFile, dbHost, dbPort, "false", splunkToken, splunkEndpoint)
    // ...
}
```

### Local Development

For local development with gnomock, the helper functions in `helper_test.go` are still available:

```go
// Still available for local development
psql := startPostgres(t)
splunk := startSplunk(t, "password")
```

See `test/INTEGRATION-SETUP.md` for detailed information about the changes

## Environment Variables

The test runner accepts these environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `DB_DRIVER` | Database driver | `pgx` |
| `DB_HOST` | Database hostname | `localhost` |
| `DB_PORT` | Database port | `5432` |
| `DB_USER` | Database user | `gabi` |
| `DB_PASS` | Database password | `passwd` |
| `DB_NAME` | Database name | `mydb` |
| `DB_WRITE` | Enable write access | `false` |
| `SPLUNK_INDEX` | Splunk index | `main` |
| `SPLUNK_TOKEN` | Splunk HEC token | `test` |
| `SPLUNK_ENDPOINT` | Splunk HEC endpoint | `http://localhost:8088` |

## Troubleshooting

### Tests can't connect to database

```bash
# Check if test-pod is running
kubectl get pod test-pod

# Check database container logs
kubectl logs test-pod -c database

# Test connectivity from test runner
kubectl exec -it gabi-integration-test-runner -- curl test-pod:5432
```

### Image not found in kind

```bash
# Verify image is loaded (use podman or docker depending on your setup)
podman exec -it gabi-integration-control-plane crictl images | grep gabi
# OR: docker exec -it gabi-integration-control-plane crictl images | grep gabi

# Reload if needed
kind load docker-image gabi-integration-test:local --name gabi-integration
```

### Tests timeout

Increase timeout in pod spec:

```yaml
command: ["/usr/local/bin/integration.test"]
args: ["-test.v", "-test.timeout=20m"]
```

## Cleanup

```bash
# Delete test pods
kubectl delete pod/gabi-integration-test-runner
kubectl delete pod/test-pod

# Delete kind cluster
kind delete cluster --name gabi-integration
```

## Building Just the Test Binary

For local testing:

```bash
# Build test binary
go test -c -tags integration -o integration.test ./test

# Run locally (requires services)
./integration.test -test.v
```

## CI/CD Integration

Example GitHub Actions workflow:

```yaml
- name: Setup Kind
  uses: helm/kind-action@v1.8.0
  with:
    cluster_name: integration-test

- name: Run Integration Tests
  run: ./test/kind-integration-test.sh
```

## Additional Resources

- [Kind Documentation](https://kind.sigs.k8s.io/)
- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Kubernetes Pod Documentation](https://kubernetes.io/docs/concepts/workloads/pods/)

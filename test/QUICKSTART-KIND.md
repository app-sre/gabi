# Quick Start: Integration Tests on Kind

## TL;DR

```bash
# One command to rule them all
make integration-test-kind
```

## What Gets Created

This setup creates:

1. **`test/Dockerfile.integration`** - Multi-stage container image that:
   - Builds the gabi application
   - Compiles integration tests into a binary
   - Creates a minimal runtime image with both

2. **`test/kind-integration-test.sh`** - Automation script that:
   - Creates a kind cluster
   - Builds and loads the test image (using podman)
   - Pulls and loads supporting service images (PostgreSQL, Splunk)
   - Deploys database and splunk pods
   - Runs the tests
   - Reports results

3. **Makefile targets** - Convenient commands:
   - `make integration-test-binary` - Build test binary locally
   - `make integration-test-image` - Build container image (using podman)
   - `make integration-test-kind` - Full automated test run

## Step-by-Step Workflow

### Option 1: Automated (Recommended)

```bash
# Run everything automatically
./test/kind-integration-test.sh
```

### Option 2: Manual Control

```bash
# 1. Create kind cluster
kind create cluster --name gabi-integration

# 2. Build and load test image
make integration-test-image
kind load docker-image gabi-integration-test:local --name gabi-integration

# 3. Pull and load supporting service images
podman pull registry.redhat.io/rhel9/postgresql-16:9.6
podman pull quay.io/app-sre/splunk:latest
kind load docker-image registry.redhat.io/rhel9/postgresql-16:9.6 --name gabi-integration
kind load docker-image quay.io/app-sre/splunk:latest --name gabi-integration

# 4. Deploy supporting services
kubectl apply -f test/test-pod.yml
kubectl wait --for=condition=ready pod/test-pod --timeout=300s

# 5. Deploy test runner (create your own YAML or use the script's template)
kubectl run gabi-test \
  --image=gabi-integration-test:local \
  --image-pull-policy=Never \
  --restart=Never \
  --env="DB_HOST=test-pod" \
  --env="DB_PORT=5432" \
  --env="DB_USER=gabi" \
  --env="DB_PASS=passwd" \
  --env="DB_NAME=mydb" \
  --env="SPLUNK_ENDPOINT=http://test-pod:8088"

# 6. Watch logs
kubectl logs -f gabi-test

# 7. Check exit code
kubectl get pod gabi-test -o jsonpath='{.status.containerStatuses[0].state.terminated.exitCode}'

# 8. Cleanup
kubectl delete pod/gabi-test pod/test-pod
kind delete cluster --name gabi-integration
```

## Understanding the Test Image

The `test/Dockerfile.integration` creates an image with:

```
/usr/local/bin/
  ├── gabi              # Main application binary
  └── integration.test  # Compiled test binary

Debugging tools included:
  ├── psql              # PostgreSQL client
  ├── nc                # Network connectivity testing (ncat)
  ├── nslookup/dig      # DNS resolution testing
  └── curl              # HTTP testing
```

Run tests with:
```bash
podman run --rm gabi-integration-test:local
```

Run just the gabi app:
```bash
podman run --rm gabi-integration-test:local /usr/local/bin/gabi
```

Debug connectivity:
```bash
# Test database connection
podman run --rm gabi-integration-test:local psql -h test-pod -U gabi -d mydb -c "SELECT 1"

# Test network connectivity
podman run --rm gabi-integration-test:local nc -zv test-pod 5432
```

## How Tests Work

### ✅ Tests Are Now Cluster-Ready!

The integration tests in `integration_test.go` have been **updated** to work with deployed services:

- ✅ No longer need **gnomock/testcontainers** in Kubernetes
- ✅ Connect to services via **environment variables**
- ✅ Work in **Kubernetes pods** without special privileges
- ✅ Can still run **locally** with services on localhost

### Example

```go
func TestHealthCheckOK(t *testing.T) {
    // Reads from environment or uses defaults
    dbHost := getEnvOrDefault("DB_HOST", "localhost")
    dbPort := getEnvOrDefault("DB_PORT", "5432")

    // Connects to deployed services
    setEnvironment(configFile, dbHost, dbPort, ...)
    // ... test logic
}
```

For more details, see `test/INTEGRATION-SETUP.md`

## Architecture

```
┌─────────────────────────────────────────────────────┐
│         Kind Cluster                                │
│                                                     │
│  ┌──────────────────────────────────────────────┐  │
│  │  Service: test-pod (headless)                │  │
│  │  DNS: test-pod.default.svc.cluster.local    │  │
│  │  Ports: 5432 (postgres), 8088 (splunk-hec)  │  │
│  └──────────────────────────────────────────────┘  │
│         │                                           │
│         ▼                                           │
│  ┌─────────────┐                                    │
│  │  test-pod   │                                    │
│  │  ┌────────┐ │  ┌──────────────────┐            │
│  │  │database│ │  │ gabi-test-runner │            │
│  │  │:5432   │ │  │                  │            │
│  │  └────────┘ │  │ • Runs tests     │            │
│  │  ┌────────┐ │  │ • Connects to    │            │
│  │  │ splunk │◄├──┤   test-pod:5432  │            │
│  │  │:8088   │ │  │   test-pod:8088  │            │
│  │  └────────┘ │  │                  │            │
│  └─────────────┘  └──────────────────┘            │
│         ▲                 │                         │
│         └─────────────────┘                         │
│        (via Service DNS)                            │
└─────────────────────────────────────────────────────┘
```

### Pod-to-Pod Communication

The `test-pod.yml` creates:
1. **A Headless Service** (`clusterIP: None`) named `test-pod`
2. **A Pod** with database and Splunk containers

The Service enables DNS-based communication:
- Test runner connects to: `test-pod:5432` for PostgreSQL
- Test runner connects to: `test-pod:8088` for Splunk HEC

Without the Service, pod DNS resolution would fail!

## Troubleshooting

### "kind: command not found"
```bash
# Install kind
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
chmod +x ./kind
sudo mv ./kind /usr/local/bin/kind
```

### "ErrImagePull" in Kubernetes
```bash
# Reload the image
make integration-test-image
kind load docker-image gabi-integration-test:local --name gabi-integration
```

### Tests can't connect to database

**Symptom**: Tests fail with "connection refused" or "cannot resolve host test-pod"

**Solution**: Create a debug pod to troubleshoot (test runner pod stops when tests fail)

```bash
# 1. Check if service exists
kubectl get service test-pod
kubectl get endpoints test-pod

# 2. If no endpoints, verify pod labels
kubectl get pod test-pod --show-labels

# 3. Create a debug pod (can't exec into stopped test runner!)
kubectl run gabi-debug \
  --image=gabi-integration-test:local \
  --image-pull-policy=Never \
  --restart=Never \
  --env="DB_HOST=test-pod" \
  --command -- sleep infinity

kubectl wait --for=condition=ready pod/gabi-debug --timeout=60s

# 4. Test DNS resolution
kubectl exec -it gabi-debug -- nslookup test-pod

# 5. Test database connectivity with netcat
kubectl exec -it gabi-debug -- nc -zv test-pod 5432

# 6. Test actual database connection
kubectl exec -it gabi-debug -- sh -c "PGPASSWORD=passwd psql -h test-pod -U gabi -d mydb -c 'SELECT 1;'"

# 7. Check database logs
kubectl logs test-pod -c database

# 8. Clean up
kubectl delete pod/gabi-debug
```

**Common causes**:
- ❌ Service not created (check `kubectl get svc test-pod`)
- ❌ Label selector mismatch between Service and Pod
- ❌ Pod not ready yet (wait for pod to be Running)

### Tests timeout
```bash
# Increase timeout when running tests
kubectl run gabi-test \
  --image=gabi-integration-test:local \
  --image-pull-policy=Never \
  --restart=Never \
  --command -- /usr/local/bin/integration.test -test.v -test.timeout=20m
```

## Environment Variables Reference

| Variable | Purpose | Default | Example |
|----------|---------|---------|---------|
| `DB_HOST` | Database hostname | `localhost` | `test-pod` |
| `DB_PORT` | Database port | `5432` | `5432` |
| `DB_USER` | Database username | `gabi` | `gabi` |
| `DB_PASS` | Database password | `passwd` | `passwd` |
| `DB_NAME` | Database name | `mydb` | `mydb` |
| `SPLUNK_ENDPOINT` | Splunk HEC URL | `http://localhost:8088` | `http://test-pod:8088` |
| `SPLUNK_TOKEN` | Splunk HEC token | `test` | `abc123...` |

## Next Steps

1. **Adapt your tests**: See `test/k8s_integration_example_test.go` for examples
2. **Configure Splunk**: The test-pod splunk needs proper setup for HEC tokens
3. **CI/CD Integration**: Use `make integration-test-kind` in your pipeline
4. **Custom scenarios**: Modify `test/test-pod.yml` for different database versions

## Additional Commands

```bash
# Build just the test binary (no container build)
make integration-test-binary
./integration.test -test.v

# Interactive debugging in test container
kubectl run -it --rm gabi-debug \
  --image=gabi-integration-test:local \
  --image-pull-policy=Never \
  --restart=Never \
  -- /bin/sh

# View all test output with timestamps
kubectl logs gabi-test --timestamps=true

# Get detailed test pod information
kubectl describe pod/gabi-test
```

## Further Reading

- Full documentation: `test/README-INTEGRATION.md`
- Kind docs: https://kind.sigs.k8s.io/
- Test pod definition: `test/test-pod.yml`
- Integration tests: `test/integration_test.go`

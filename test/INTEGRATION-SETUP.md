# Integration Test Setup for Kubernetes

This document explains how the integration tests have been adapted to work with deployed services in a Kubernetes cluster.

## Changes from Original Tests

### Before (gnomock-based)
The original `integration_test.go` used **gnomock/testcontainers** to:
- Start PostgreSQL containers on-demand
- Start Splunk containers on-demand
- Get dynamic ports and connection info
- Clean up containers after tests

This approach worked great locally but **doesn't work in Kubernetes pods** without privileged container-in-container access.

### After (environment-based)
The updated `integration_test.go` now:
- ✅ Reads connection info from **environment variables**
- ✅ Connects to **pre-deployed services** in the cluster
- ✅ Works in Kubernetes pods without special privileges
- ✅ Uses the same test logic and assertions

## Environment Variables

All tests now use these environment variables with sensible defaults:

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | `localhost` | PostgreSQL hostname |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `gabi` | Database username (set by `setEnvironment()`) |
| `DB_PASS` | `passwd` | Database password (set by `setEnvironment()`) |
| `DB_NAME` | `mydb` | Database name (set by `setEnvironment()`) |
| `SPLUNK_ENDPOINT` | `http://localhost:8088` | Splunk HEC endpoint |
| `SPLUNK_TOKEN` | `test123` | Splunk HEC token |

## Running Tests

### In Kubernetes (Recommended)

```bash
# Automated - runs everything
make integration-test-kind

# Or use the script directly
./test/kind-integration-test.sh
```

The script will:
1. Deploy PostgreSQL and Splunk in a test pod
2. Run your integration tests against those services
3. Report results

### Locally (requires local services)

If you have PostgreSQL and Splunk running locally:

```bash
# Set up environment (optional if using defaults)
export DB_HOST=localhost
export DB_PORT=5432
export SPLUNK_ENDPOINT=http://localhost:8088
export SPLUNK_TOKEN=your-token-here

# Run tests
go test -v -tags integration ./test
```

### Local Development with gnomock

For local development, you can still use the old gnomock-based approach by using the helper functions in `helper_test.go`:

```go
// helper_test.go still has the gnomock functions
psql := startPostgres(t)
splunk := startSplunk(t, "password")
```

These are kept for backward compatibility with local development workflows.

## Splunk Configuration

⚠️ **Important**: Some tests require a valid Splunk HEC token.

### For Real Splunk Instances

If testing against a real Splunk deployment, you need to:

1. **Create an HEC token**:
   ```bash
   # Using Splunk API
   curl -k -u admin:password https://splunk-host:8089/servicesNS/admin/splunk_httpinput/data/inputs/http \
     -d name=gabi-test-token
   ```

2. **Extract the token** from the response and set it:
   ```bash
   export SPLUNK_TOKEN="your-actual-token"
   ```

### For Mock/Testing

The tests use `test123` as a default token. If you're not testing Splunk functionality specifically, you can:

1. **Skip Splunk-dependent tests** by using test filters
2. **Use a mock Splunk** that accepts any token
3. **Comment out** Splunk assertions temporarily

### Tests That Require Splunk

These tests specifically validate Splunk integration:
- `TestQueryWithRequestTimedOut`
- `TestQueryWithSplunkWrite`
- `TestQueryWithSplunkWriteFailure` (expects failure)
- `TestQueryWithDatabaseWriteAccess`
- `TestQueryWithDatabaseWriteAccessFailure`
- `TestQueryWithBase64EncodedQuery`
- `TestQueryWithBase64EncodedResults`

## Test Pod Configuration

The `test/test-pod.yml` deploys:

```yaml
containers:
- name: splunk
  image: quay.io/app-sre/splunk:latest
  # Note: Splunk needs time to start and configure HEC

- name: database
  image: registry.redhat.io/rhel9/postgresql-16:9.6
  env:
  - name: POSTGRESQL_USER
    value: gabi
  - name: POSTGRESQL_PASSWORD
    value: passwd
  - name: POSTGRESQL_DATABASE
    value: mydb
```

## Troubleshooting

### Tests can't connect to database

**Problem**: Tests fail with errors like:
- "connection refused"
- "could not resolve host test-pod"
- "no such host"

**Root Cause**: Pods need a Kubernetes **Service** for DNS-based communication.

⚠️ **Important**: If tests fail, the test runner pod stops. Create a debug pod to troubleshoot:

```bash
# Create debug pod with the same image
kubectl run gabi-debug \
  --image=gabi-integration-test:local \
  --image-pull-policy=Never \
  --restart=Never \
  --env="DB_HOST=test-pod" \
  --command -- sleep infinity

kubectl wait --for=condition=ready pod/gabi-debug --timeout=60s
```

**Solution**:

1. **Verify the Service exists**:
   ```bash
   kubectl get service test-pod
   ```

   Expected output:
   ```
   NAME       TYPE        CLUSTER-IP   EXTERNAL-IP   PORT(S)             AGE
   test-pod   ClusterIP   None         <none>        5432/TCP,8088/TCP   1m
   ```

2. **Check Service has endpoints**:
   ```bash
   kubectl get endpoints test-pod
   ```

   Expected output (should show the pod IP):
   ```
   NAME       ENDPOINTS                         AGE
   test-pod   10.244.0.5:5432,10.244.0.5:8088   1m
   ```

3. **Verify Pod labels match Service selector**:
   ```bash
   # Check pod labels
   kubectl get pod test-pod --show-labels

   # Should have: app=test-pod
   ```

4. **Test DNS from debug pod**:
   ```bash
   # DNS resolution
   kubectl exec -it gabi-debug -- nslookup test-pod

   # Should resolve to pod IP
   ```

5. **Test connectivity**:
   ```bash
   # Test PostgreSQL port with netcat
   kubectl exec -it gabi-debug -- nc -zv test-pod 5432

   # Test Splunk port with netcat
   kubectl exec -it gabi-debug -- nc -zv test-pod 8088

   # Test actual PostgreSQL connection
   kubectl exec -it gabi-debug -- sh -c "PGPASSWORD=passwd psql -h test-pod -U gabi -d mydb -c 'SELECT version();'"

   # Test Splunk HEC endpoint
   kubectl exec -it gabi-debug -- curl http://test-pod:8088/services/collector/health
   ```

6. **Check database logs**:
   ```bash
   kubectl logs test-pod -c database
   ```

7. **Clean up debug pod**:
   ```bash
   kubectl delete pod/gabi-debug
   ```

**What's in test-pod.yml**:

The file creates TWO resources:
1. A **Headless Service** (enables DNS)
2. A **Pod** (runs database and Splunk)

```yaml
---
apiVersion: v1
kind: Service
metadata:
  name: test-pod
spec:
  selector:
    app: test-pod  # Must match pod labels!
  clusterIP: None  # Headless service
  ports:
  - name: postgres
    port: 5432
  - name: splunk-hec
    port: 8088
---
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  labels:
    app: test-pod  # Matches service selector!
# ...
```

### Splunk tests failing

```bash
# Check Splunk is running
kubectl logs test-pod -c splunk

# Verify HEC endpoint is accessible
kubectl exec -it gabi-integration-test-runner -- curl http://test-pod:8088/services/collector/health
```

**Common issue**: Splunk takes 2-3 minutes to fully start. Make sure to wait for the pod to be ready:

```bash
kubectl wait --for=condition=ready pod/test-pod --timeout=300s
```

### Tests timeout

Some tests (like `TestQueryWithRequestTimedOut`) intentionally take time. Increase the test timeout:

```bash
go test -v -tags integration -timeout=20m ./test
```

## Code Example: How Tests Changed

### Before (gnomock)
```go
func TestHealthCheckOK(t *testing.T) {
    psql := startPostgres(t)  // Starts a container

    setEnvironment(configFile, psql.Host,
        strconv.Itoa(psql.DefaultPort()), ...)
    // ...
}
```

### After (environment-based)
```go
func TestHealthCheckOK(t *testing.T) {
    // Reads from environment or uses defaults
    dbHost := getEnvOrDefault("DB_HOST", "localhost")
    dbPort := getEnvOrDefault("DB_PORT", "5432")

    setEnvironment(configFile, dbHost, dbPort, ...)
    // ... same test logic
}
```

## Future Enhancements

Consider these improvements:

1. **Service readiness checks**: Add init containers to verify services are ready
2. **Splunk token automation**: Auto-create HEC token in Splunk container
3. **Test parallelization**: Run independent tests in parallel
4. **Cleanup automation**: Automatically clean up test data between runs

## Related Documentation

- Main guide: `test/README-INTEGRATION.md`
- Quick start: `test/QUICKSTART-KIND.md`
- Example tests: `test/k8s_integration_example_test.go`
- Helper script: `test/kind-integration-test.sh`

# Debug Pod Pattern: Troubleshooting Failed Tests

## The Problem

When integration tests fail in Kubernetes, the test runner pod **stops/exits**, making it impossible to exec into it for debugging:

```bash
# This FAILS after tests complete or fail:
kubectl exec -it gabi-integration-test-runner -- /bin/sh
# Error: container is in "terminated" state

# Can't debug connectivity or run diagnostic commands!
```

## The Solution: Debug Pod Pattern

Create a **separate debug pod** using the same image that stays running indefinitely:

```bash
# Create a debug pod with the same image and environment
kubectl run gabi-debug \
  --image=gabi-integration-test:local \
  --image-pull-policy=Never \
  --restart=Never \
  --env="DB_HOST=test-pod" \
  --env="DB_PORT=5432" \
  --env="SPLUNK_ENDPOINT=http://test-pod:8088" \
  --command -- sleep infinity

# Wait for it to be ready
kubectl wait --for=condition=ready pod/gabi-debug --timeout=60s
```

Now you can exec into it and debug:

```bash
# Exec into the debug pod
kubectl exec -it gabi-debug -- /bin/sh

# Run any debugging commands
$ nslookup test-pod
$ nc -zv test-pod 5432
$ PGPASSWORD=passwd psql -h test-pod -U gabi -d mydb -c 'SELECT 1;'
$ curl http://test-pod:8088/services/collector/health
$ exit

# Clean up when done
kubectl delete pod/gabi-debug
```

## Why This Works

The debug pod:
1. ✅ Uses the **same image** (same tools, same environment)
2. ✅ Has the **same network access** (can reach test-pod Service)
3. ✅ Runs **indefinitely** (`sleep infinity` keeps it alive)
4. ✅ Can be **exec'd into** for interactive debugging

## Complete Example

### Scenario: Tests fail with "connection refused"

```bash
# 1. Check test runner logs to see the error
kubectl logs gabi-integration-test-runner
# Output: Error: dial tcp 10.244.0.5:5432: connect: connection refused

# 2. Try to debug - FAILS because pod is terminated
kubectl exec -it gabi-integration-test-runner -- /bin/sh
# Error: container is in "terminated" state

# 3. Create a debug pod instead
kubectl run gabi-debug \
  --image=gabi-integration-test:local \
  --image-pull-policy=Never \
  --restart=Never \
  --env="DB_HOST=test-pod" \
  --command -- sleep infinity

kubectl wait --for=condition=ready pod/gabi-debug --timeout=60s

# 4. Test DNS resolution
kubectl exec -it gabi-debug -- nslookup test-pod
# Success: Resolves to 10.244.0.5

# 5. Test port connectivity
kubectl exec -it gabi-debug -- nc -zv test-pod 5432
# Success: test-pod [10.244.0.5] 5432 (postgresql) open

# 6. Test actual database connection
kubectl exec -it gabi-debug -- sh -c "PGPASSWORD=passwd psql -h test-pod -U gabi -d mydb -c 'SELECT 1;'"
# Success: Returns "1"

# 7. Conclusion: Database is accessible, issue must be in test code

# 8. Clean up
kubectl delete pod/gabi-debug
```

## Common Use Cases

### Use Case 1: Test DNS Resolution

```bash
kubectl run gabi-debug \
  --image=gabi-integration-test:local \
  --image-pull-policy=Never \
  --command -- sleep infinity

kubectl exec -it gabi-debug -- nslookup test-pod
kubectl exec -it gabi-debug -- dig test-pod +short

kubectl delete pod/gabi-debug
```

### Use Case 2: Test Database Connection

```bash
kubectl run gabi-debug \
  --image=gabi-integration-test:local \
  --image-pull-policy=Never \
  --env="DB_HOST=test-pod" \
  --command -- sleep infinity

kubectl exec -it gabi-debug -- nc -zv test-pod 5432
kubectl exec -it gabi-debug -- sh -c "PGPASSWORD=passwd psql -h test-pod -U gabi -d mydb -c 'SELECT version();'"

kubectl delete pod/gabi-debug
```

### Use Case 3: Interactive Debugging Session

```bash
kubectl run gabi-debug \
  --image=gabi-integration-test:local \
  --image-pull-policy=Never \
  --env="DB_HOST=test-pod" \
  --env="DB_PORT=5432" \
  --command -- sleep infinity

# Open interactive shell
kubectl exec -it gabi-debug -- /bin/sh

# Now explore interactively:
$ nslookup test-pod
$ nc -zv test-pod 5432
$ PGPASSWORD=passwd psql -h test-pod -U gabi -d mydb
mydb=> \dt
mydb=> SELECT * FROM pg_stat_activity;
mydb=> \q
$ curl http://localhost:8080/healthcheck
$ exit

kubectl delete pod/gabi-debug
```

### Use Case 4: Test Network Policies

```bash
kubectl run gabi-debug \
  --image=gabi-integration-test:local \
  --image-pull-policy=Never \
  --command -- sleep infinity

# Test if network policy allows traffic
kubectl exec -it gabi-debug -- nc -zv test-pod 5432
kubectl exec -it gabi-debug -- nc -zv test-pod 8088
kubectl exec -it gabi-debug -- curl http://test-pod:8088/services/collector/health

kubectl delete pod/gabi-debug
```

## Alternative: Using Ephemeral Containers (Kubernetes 1.23+)

If your cluster supports ephemeral containers, you can debug even terminated pods:

```bash
# Add an ephemeral debug container to the terminated pod
kubectl debug -it gabi-integration-test-runner --image=gabi-integration-test:local --target=gabi-integration-test-runner

# Note: This requires Kubernetes 1.23+ and EphemeralContainers feature gate
```

However, the debug pod pattern is:
- ✅ More portable (works on any Kubernetes version)
- ✅ More predictable (clean environment)
- ✅ Easier to script (standard `kubectl run`)

## Best Practices

### 1. Use Consistent Naming

Always name debug pods `gabi-debug` for consistency:

```bash
kubectl run gabi-debug ...
```

### 2. Set Environment Variables

Mirror the test environment:

```bash
--env="DB_HOST=test-pod" \
--env="DB_PORT=5432" \
--env="SPLUNK_ENDPOINT=http://test-pod:8088"
```

### 3. Always Clean Up

Delete debug pods when done:

```bash
kubectl delete pod/gabi-debug
```

### 4. Check Existing Debug Pods

Before creating a new one:

```bash
# Check if debug pod already exists
kubectl get pod gabi-debug

# If exists, delete it first
kubectl delete pod/gabi-debug --ignore-not-found=true

# Then create new one
kubectl run gabi-debug ...
```

### 5. Use Image Pull Policy

In kind clusters, always use `--image-pull-policy=Never`:

```bash
--image-pull-policy=Never
```

## Comparison: Exec vs Debug Pod

| Aspect | `kubectl exec` | Debug Pod |
|--------|----------------|-----------|
| **Works on stopped pods** | ❌ No | ✅ Yes |
| **Clean environment** | ❌ Shares test state | ✅ Fresh environment |
| **Easy to create** | ✅ One command | ⚠️ Two commands |
| **Cleanup needed** | ✅ Automatic | ⚠️ Manual deletion |
| **Use when** | Pod is running | Pod stopped/failed |

## Scripted Debug Pod

Save this as `debug-pod.sh` for quick debugging:

```bash
#!/bin/bash
set -e

POD_NAME="${1:-gabi-debug}"
IMAGE="${2:-gabi-integration-test:local}"

echo "Creating debug pod: ${POD_NAME}"

# Delete if exists
kubectl delete pod/${POD_NAME} --ignore-not-found=true

# Create debug pod
kubectl run ${POD_NAME} \
  --image=${IMAGE} \
  --image-pull-policy=Never \
  --restart=Never \
  --env="DB_HOST=test-pod" \
  --env="DB_PORT=5432" \
  --env="SPLUNK_ENDPOINT=http://test-pod:8088" \
  --command -- sleep infinity

# Wait for ready
echo "Waiting for pod to be ready..."
kubectl wait --for=condition=ready pod/${POD_NAME} --timeout=60s

echo "Debug pod ready! Use: kubectl exec -it ${POD_NAME} -- /bin/sh"
echo "Clean up with: kubectl delete pod/${POD_NAME}"
```

Usage:

```bash
chmod +x debug-pod.sh
./debug-pod.sh
kubectl exec -it gabi-debug -- /bin/sh
```

## Related Documentation

- Troubleshooting: `test/INTEGRATION-SETUP.md`
- Quick reference: `test/QUICKSTART-KIND.md`

## Summary

**Problem**: Can't exec into stopped/failed test runner pod
**Solution**: Create a debug pod using the same image
**Pattern**: `kubectl run gabi-debug --image=... --command -- sleep infinity`
**Result**: Interactive debugging environment that mimics the test runner ✅

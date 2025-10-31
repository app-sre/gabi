# Mock Splunk Server for Integration Testing

This is a lightweight mock HTTP server written in Go that mimics Splunk HEC (HTTP Event Collector) and Management API endpoints. It replaces the heavy Splunk container in integration tests, providing faster startup times and reduced resource requirements.

## Features

The mock server implements the following Splunk API endpoints:

### HEC Endpoints (Port 8088)
- `POST /services/collector/event` - Accepts audit log events
- `GET /services/collector/health/1.0` - Health check endpoint

### Management API Endpoints (Port 8089, HTTPS)
- `POST /servicesNS/admin/splunk_httpinput/data/inputs/http` - Create HEC tokens
- `DELETE /servicesNS/admin/splunk_httpinput/data/inputs/http/{tokenName}` - Delete HEC tokens

## How It Works

### HEC Event Ingestion
The mock server accepts events in Splunk HEC format:
```json
{
  "event": {
    "query": "SELECT 1;",
    "user": "testuser",
    "namespace": "test",
    "pod": "test-pod"
  },
  "index": "main",
  "host": "test-host",
  "source": "gabi",
  "sourcetype": "json",
  "time": 1234567890
}
```

All valid events return a success response:
```json
{
  "text": "Success",
  "code": 0
}
```

### Token Management
The mock server generates UUID-based tokens when requested via the Management API. Tokens are stored in-memory and can be created and deleted just like in real Splunk.

### HTTPS Support
The Management API (port 8089) uses a self-signed certificate generated at startup. The certificate is automatically created and stored in `/tmp/cert.pem` and `/tmp/key.pem`.

## Usage

### Standalone
```bash
go run main.go
```

The server will start on:
- HTTP on port 8088 (HEC)
- HTTPS on port 8089 (Management API)

### In Podman/Kubernetes
The mock server is built into the `gabi-integration-test` container image and can be started with:
```bash
/usr/local/bin/mock-splunk
```

## Benefits over Real Splunk

1. **Faster Startup**: Starts in milliseconds vs. 60+ seconds for Splunk
2. **Lower Resource Usage**: ~64MB RAM vs. 512MB-2GB for Splunk
3. **No Complex Configuration**: No need for license acceptance or HEC enablement
4. **Deterministic Behavior**: Predictable responses for testing
5. **Simpler Maintenance**: No version updates or security patches needed

## Implementation Details

- **Language**: Go 1.22+
- **Dependencies**:
  - `github.com/google/uuid` - Token generation
  - `github.com/gorilla/mux` - HTTP routing
- **Concurrency**: Thread-safe token storage using sync.RWMutex
- **Authentication**: Supports Basic Auth for Management API and Bearer tokens for HEC

## Compatibility

The mock server is compatible with the gabi application's Splunk integration code in `pkg/audit/splunk.go`. It implements all the endpoints and response formats expected by the integration tests.

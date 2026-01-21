# GABI - Go Auditable DB Interface

An auditable database query interface service for running SQL queries on protected databases without exposing credentials.

## Project Structure

- `cmd/gabi/` - Main application entry point
- `pkg/` - Core packages
  - `audit/` - Audit logging (console, Splunk)
  - `cmd/` - CLI command handling
  - `env/` - Environment configuration (db, splunk, user)
  - `handlers/` - HTTP request handlers (query, healthcheck, dbname)
  - `middleware/` - HTTP middleware (auth, audit, expiration, timeout, recovery)
  - `models/` - Data models
  - `version/` - Version info
- `internal/test/` - Test helpers
- `test/` - Integration tests
- `openshift/` - OpenShift deployment templates
- `.tekton/` - Tekton pipeline files

## Development Commands

```bash
make build              # Build the binary
make test               # Run unit tests
make linux              # Build Linux binary
make clean              # Clean build artifacts
make integration-test   # Run integration tests (requires k8s namespace)
make integration-test-kind  # Run integration tests locally with Kind
```

## Running Locally

1. Create `config.json` with expiration date and authorized users
2. Set environment variables (see `env.template`)
3. Run: `go run cmd/gabi/main.go`

## Environment Variables

- `DB_DRIVER` - Database driver (mysql, pgx - default: pgx)
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASS`, `DB_NAME` - Database connection
- `DB_WRITE` - Enable write access (default: false)
- `CONFIG_FILE_PATH` - Path to config.json
- `EXPIRATION_DATE` - Override expiration (YYYY-MM-DD)
- `AUTHORIZED_USERS` - Comma-separated list of authorized users

## Testing

Unit tests use `go-sqlmock` for database mocking and `testify` for assertions.

Integration tests require PostgreSQL and optionally Splunk. Run with Kind cluster using `make integration-test-kind`.


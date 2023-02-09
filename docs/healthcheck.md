# Health check

## Accessing the health check endpoint

```
$ curl -s http://localhost:8080/healthcheck
```

## Typical Responses

### Service is healthy

```
{
  "status": "OK"
}
```

### Service is unhealthy due to database connectivity issues

```
{
  "status": "Service Unavailable",
  "errors": {
    "database": "Unable to connect to the database"
  }
}
```

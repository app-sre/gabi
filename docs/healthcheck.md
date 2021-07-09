# Healthcheck

## Endpoint

`<svc_host>:<svc_port>/healthcheck`

## Return Paylaods

### OK
`
{"status":"OK"}
`

### Bad Host (Timeout)
`
{"status":"Service Unavailable","errors":{"database":"max check time exceeded"}}
`

### Bad Port
`
{"status":"Service Unavailable","errors":{"database":"dial tcp 127.0.0.1:32024: connect: connection refused"}}
`

### Bad User
`
{"status":"Service Unavailable","errors":{"database":"Error 1045: Access denied for user 'raot'@'10.42.0.1' (using password: YES)"}}
`

### Bad Password
`
{"status":"Service Unavailable","errors":{"database":"Error 1045: Access denied for user 'root'@'10.42.0.1' (using password: YES)"}}
`
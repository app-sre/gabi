# GABI: (G)o (A)uditable D(B) (I)nterface

Improving tenant quality of life for database access while following Site Reliability Engineering best practices.

**Note:** GABI is under active development and is not suitable for production use!

## Description

`GABI` is a service that provides an interface for tenants and SREs to run SQL queries on protected databases without
exposing credentials, complete with audit capabilities to comply with certifications (i.e., SOC-2). Organizations that
adopt SRE best practices are often found walking a tight line between developer happiness and full regulatory
compliance. One common area of conflict is database access. Developers are familiar with read-and-write access during
their project work but find production restrictions on databases frustrating and time-consuming. `GABI` attempts to
bridge the gap between SRE and developer needs by providing an auditable, secure, and available interface to query
databases.

### Best Practice vs Best Effort

"Best Practice" applications are applications that are fully onboarded according to SRE best practices, with the end
result of SREs taking responsibility for service health (i.e., carrying the pager).

"Best Effort" applications are run in SRE-defined runtime environments but, by choice or design, do not
follow all requirements mandated by the SRE team. These applications enjoy the benefits of SRE-led infrastructure, but
the SRE team is not tasked with carrying the pager for these services.

This stratified support model is supported in `GABI` with RBAC restrictions. Instances of `GABI` can be set to read-only
mode to support "best practice" services through read-replica databases. Optionally, read/write
mode can be enabled for primary databases, supporting "best effort" services.

### GitOps

To the joy of many SREs, `GABI` is created with GitOps in mind. In addition to allowing HTTP requests to the service
from a developer's machine, a reconciliation server can manage to interact with the API and can execute queries as part
of PR or MR hooks. This allows for a complete GitOps workflow and includes the added benefits of tracking each query
through version-controlled files.

### Secret Management

The service consumes database access credentials through environment variables. There are a multitude of secret
management techniques that can supply environment variables to Kubernetes pods, such as Vault, Kubernetes Secrets,
ConfigMaps, and more. This approach implies that one instance of `GABI` is needed for each database, as each instance of
`GABI` will only execute queries on the database defined at pod creation time.

### Supported Databases

Currently, `GABI` supports MySQL and PostgreSQL. The database interface is written with `sql.DB`, so other database
types will be easy to implement, and we welcome contributions from the community.

### Runtime Environment

The service is written in the Go programming language intended to run as a Kubernetes or OpenShift workload. In addition
to a Kubernetes workload, the application can run in a standalone Docker container or as a CLI app (not recommended).


## Quick start

`TODO`

Create a `config.json` file with the following content and set its path using `CONFIG_FILE_PATH`:

Note: the expiration date has to be set in the future.

```
{
  "expiration": "YYYY-MM-DD",
  "users": [
    "user1",
    "user2"
  ]
}
 ```

To use the deprecated users file, create a text file (historically, the file name as been `authorized-users.yaml`) and
then set its path using `USERS_FILE_PATH`:


```
user1
user2
```

Note: using the deprecated users file will disable support of the explicit expiration date in GABI, leaving only the
legacy implicit one active.

Next, start the GABI server instance:

```
$ source .env.dev
$ go run cmd/gabi/main.go
2023-02-09T11:28:48.981+0900	INFO	cmd/cmd.go:32	Starting GABI version: 0.1.0
2023-02-09T11:28:48.981+0900	INFO	cmd/cmd.go:47	Production: false, expired: false (expiration date: 2038-01-19)
2023-02-09T11:28:48.981+0900	DEBUG	cmd/cmd.go:48	Authorized users: [kwilczynski test]
2023-02-09T11:28:48.981+0900	INFO	cmd/cmd.go:55	Using database driver: pgx (write access: false)
2023-02-09T11:28:48.981+0900	DEBUG	cmd/cmd.go:62	Connected to database host: localhost (port: 5432)
2023-02-09T11:28:48.981+0900	INFO	cmd/cmd.go:71	Sending audit to Splunk endpoint: https://example.com
2023-02-09T11:28:48.981+0900	INFO	cmd/cmd.go:105	HTTP server starting on port: 8080
127.0.0.1 - - [09/Feb/2023:11:28:54 +0900] "GET /healthcheck HTTP/1.1" 200 16
2023-02-09T11:36:47.296+0900	INFO	audit/logger.go:18	AUDIT	{"Query": "select table_name from information_schema.tables where table_schema='public'", "User": "test", "Timestamp": 2943010800}
127.0.0.1 - - [09/Feb/2023:11:36:47 +0900] "POST /query HTTP/1.1" 200 39
```

An example query against a PostgreSQL to check for the existance of a specific table in the database (single quotes need
to be replaced with `'\''` in queries run with curl):

```
$ curl -s 'http://localhost:8080/query' -X POST -H 'X-Forwarded-User: test' -d '{"query":"select table_name from information_schema.tables where table_schema='\''public'\''"}' | jq
{
  "result": [
    [
      "table_name"
    ],
    [
      "persons"
    ]
  ],
  "error": ""
}
```

Using a Base64-encoded query when making a request can help alleviate some of the challenges of complex queries (SQL
statements) that include a combination of quotes, and other characters that the JSON standard considers reserved can
often be problematic, especially as ensuring that challenging parts of the SQL query have been correctly escaped can be
quite involved and error-prone. When passing a Base64-encoded query string, make sure that the `base64_query=true` query
parameter is set when making a request. For example:

```
$ echo -n "select table_name from information_schema.tables where table_schema='public'" | base64 | tr -d '\n'
c2VsZWN0IHRhYmxlX25hbWUgZnJvbSBpbmZvcm1hdGlvbl9zY2hlbWEudGFibGVzIHdoZXJlIHRhYmxlX3NjaGVtYT0ncHVibGljJw==

$ curl -s 'http://localhost:8080/query?base64_query=true' -X POST -H 'X-Forwarded-User: test' -d '{"query":"c2VsZWN0IHRhYmxlX25hbWUgZnJvbSBpbmZvcm1hdGlvbl9zY2hlbWEudGFibGVzIHdoZXJlIHRhYmxlX3NjaGVtYT0ncHVibGljJw=="}' | jq
{
  "result": [
    [
      "table_name"
    ],
    [
      "persons"
    ]
  ],
  "error": ""
}
```

A Base64-encoding can also be applied to the results. This enables rich data, such as embedded JSON documents, to be
passed without a need to escape quotes and any other special characters to be included in the response. To apply
Base64-encoding to the results, pass a `base64_results=true` query parameter when making a request. For example:

```
$ curl -s 'http://localhost:8080/query' -X POST -H 'X-Forwarded-User: test' -d '{"query":"select * from books;"}'
{"result":[["data"],["{\"title\": \"Deep Work: Rules for Focused Success in a Distracted World\", \"author\": \"Cal Newport\", \"genres\": [\"Productivity\", \"Reference\"]}"]],"error":""}

$ curl -s 'http://localhost:8080/query?base64_results=true' -X POST -H 'X-Forwarded-User: test' -d '{"query":"select * from books;"}'
{"result":[["data"],["eyJ0aXRsZSI6ICJEZWVwIFdvcms6IFJ1bGVzIGZvciBGb2N1c2VkIFN1Y2Nlc3MgaW4gYSBEaXN0cmFjdGVkIFdvcmxkIiwgImF1dGhvciI6ICJDYWwgTmV3cG9ydCIsICJnZW5yZXMiOiBbIlByb2R1Y3Rpdml0eSIsICJSZWZlcmVuY2UiXX0="]],"error":""}

$ cat - | base64 -d
eyJ0aXRsZSI6ICJEZWVwIFdvcms6IFJ1bGVzIGZvciBGb2N1c2VkIFN1Y2Nlc3MgaW4gYSBEaXN0cmFjdGVkIFdvcmxkIiwgImF1dGhvciI6ICJDYWwgTmV3cG9ydCIsICJnZW5yZXMiOiBbIlByb2R1Y3Rpdml0eSIsICJSZWZlcmVuY2UiXX0=
{"title": "Deep Work: Rules for Focused Success in a Distracted World", "author": "Cal Newport", "genres": ["Productivity", "Reference"]}
```

Note: almost every modern and well-behaved JSON parser would attempt to unescape quotes and handle reserved characters
correctly.

## Detailed Operation

`TODO`

## Environment Variables

### DB_DRIVER Options

* mysql
* pgx (default)

```
DB_DRIVER=pgx
DB_HOST=127.0.0.1
DB_PORT=5432
DB_USER=root
DB_PASS=secret123
DB_NAME=main
DB_WRITE=false
```

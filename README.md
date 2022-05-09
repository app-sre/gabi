# gabi: `go-auditable-db-interface`

Improving tenant quality of life for database access while following site reliability engineering best practices

**Note:** gabi is under active development and is not suitable for production use.

## Description

`gabi` is a service that provides an interface for tenants and SREs to run SQL queries on protected databases without exposing credentials, complete with audit capabilities to comply with certifications (i.e. SOC-2). Organizations that adopt SRE best practices are often found walking a tight line between developer happiness and full regulatory compliance. One common area of conflict is around database access. Developers are familiar with read and write access during their project work, but find production restrictions on databases to be frustrating and time consuming. `gabi` attempts to bridge the gap between SRE needs and developer needs by providing an auditable, secure, and available interface to query databases.

### Best Practice vs Best Effort

“Best Practice” applications are applications that are fully onboarded according to SRE best practices, with the end result of SREs taking responsibility for service health (i.e. carrying the pager). 

“Best Effort” applications are applications that run in SRE-defined runtime environments, but by choice or design do not follow all requirements mandated by the SRE team. These applications enjoy the benefits of SRE-led infrastructure, but the SRE team is not tasked with carrying the pager for these services. 

This stratified support model is supported in `gabi` with RBAC restrictions. Instances of `gabi` can be set to read only mode with the intention of supporting “best practice” services through read-replica databases. Optionally, read/write mode can be enabled for primary databases, supporting “best effort” services. 

### GitOps

To the joy of many SREs, `gabi` is created with GitOps in mind. In addition to allowing HTTP requests to the service from a developer’s machine, a reconciliation server can manage interacting with the API and can execute queries as part of PR or MR hooks. This allows for a full GitOps workflow, and includes the added benefits of tracking each query through version controlled files.

### Secret Management

The service consumes database access credentials through environment variables. There are a multitude of secret management techniques that can supply environment variables to Kubernetes pods, such as Vault, Kubernetes Secrets, ConfigMaps, and more. This approach implies that one instance of `gabi` is needed for each database, as each instance of `gabi` will only execute queries on the database defined at pod creation time.

### Supported Databases

Currently, `gabi` supports MySQL and PostgreSQL. The database interface is written with sql.DB, so other database types will be easy to implement, and we welcome contributions from the community.

### Runtime Environment

The service is written in golang and is intended to be run as a Kubernetes or OpenShift workload. In addition to a Kubernetes workload, the application can run in a standalone Docker container, or as a CLI app (not recommended).

## Quickstart

`TODO`

Create a authorized-users.yamls file and set its path in USERS_FILE_PATH
```
user1
user2
```

Start gabi server
```
$ source .env.dev
$ go run cmd/gabi/main.go

2022-05-09T18:20:33.972+0800    INFO    gabi/main.go:20 Starting gabi server version 0.0.1
2022-05-09T18:20:33.973+0800    INFO    cmd/cmd.go:29   Authorized Users populated.
2022-05-09T18:20:33.973+0800    INFO    cmd/cmd.go:36   Database environment variables populated.
2022-05-09T18:20:33.973+0800    INFO    cmd/cmd.go:39   Using default audit backend: stdout logger.
2022-05-09T18:20:33.973+0800    INFO    cmd/cmd.go:46   Splunk environment variables populated.
2022-05-09T18:20:33.973+0800    INFO    cmd/cmd.go:49   Using Splunk audit backend.
2022-05-09T18:20:33.973+0800    INFO    cmd/cmd.go:51   Establishing DB connection pool.
2022-05-09T18:20:33.974+0800    INFO    cmd/cmd.go:57   Database connection handle established.
2022-05-09T18:20:33.974+0800    INFO    cmd/cmd.go:58   Using pgx database driver.
2022-05-09T18:20:33.974+0800    INFO    cmd/cmd.go:67   Router initialized.
2022-05-09T18:20:33.974+0800    INFO    cmd/cmd.go:70   HTTP server starting on port 8080.
127.0.0.1 - - [09/May/2022:18:21:13 +0800] "GET /healthcheck HTTP/1.1" 200 16
127.0.0.1 - - [09/May/2022:18:25:57 +0800] "POST /query HTTP/1.1" 200 39

```

Query example to list table_name in database (single quotes need to be replaced with `'\''` in queries run with curl)
```
$ curl 'http://localhost:8080/query' -H 'X-Forwarded-User: user1' -d '{ "query": "select table_name from information_schema.tables where table_schema='\''public'\''"}' -s | jq 

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

## Detailed Operation

`TODO`

## Environment Variables

### DB_DRIVER Options

* mysql
* pgx

```
DB_DRIVER=mysql # pgx
DB_HOST=127.0.0.1
DB_PORT=32084
DB_USER=root
DB_PASS=tpate
DB_NAME=employees
DB_WRITE=false
```



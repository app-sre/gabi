FROM registry.access.redhat.com/ubi9/go-toolset:1.25.7-1771417345@sha256:799cc027d5ad58cdc156b65286eb6389993ec14c496cf748c09834b7251e78dc AS builder
ENV GOGC=off
ENV CGO_ENABLED=0
ENV GOPROXY=https://proxy.golang.org,direct

WORKDIR /build
RUN git config --global --add safe.directory /build

COPY go.mod go.sum ./

RUN set -eux && \
  go mod download

COPY . ./

RUN set -eux && \
  go build -ldflags '-s -w' -o gabi cmd/gabi/main.go

FROM registry.access.redhat.com/ubi9/ubi-minimal:9.7-1771346502@sha256:2bd144364d2cb06b08953ce5764cdbf236bbcd63cea214583c4ed011b4685453

COPY LICENSE /licenses/LICENSE

ENV DB_DRIVER=pgx
ENV DB_HOST=127.0.0.1
ENV DB_PORT=5432
ENV DB_USER=postgres
ENV DB_PASS=postgres
ENV DB_NAME=main
ENV DB_WRITE=false

EXPOSE 8080

USER 1001

COPY --from=builder /build/gabi .

CMD ["./gabi"]

FROM registry.access.redhat.com/ubi9/go-toolset:1.22.9-1744194661@sha256:e4193e71ea9f2e2504f6b4ee93cadef0fe5d7b37bba57484f4d4229801a7c063 AS builder

ENV GOGC=off
ENV CGO_ENABLED=0

WORKDIR /build
RUN git config --global --add safe.directory /build

COPY go.mod go.sum ./

RUN set -eux && \
  go mod download

COPY . ./

RUN set -eux && \
  go build -ldflags '-s -w' -o gabi cmd/gabi/main.go

FROM registry.access.redhat.com/ubi9/ubi-minimal:9.5-1745855087@sha256:e1c4703364c5cb58f5462575dc90345bcd934ddc45e6c32f9c162f2b5617681c

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

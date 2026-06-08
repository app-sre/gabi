FROM registry.access.redhat.com/ubi9/go-toolset:9.8-1780490420@sha256:d36470d5258da00f618b7aca9bdaab8e05134aa938bd6c42d9bd17d50ed45e76 AS builder
ENV GOGC=off
ENV CGO_ENABLED=0
ENV GOPROXY=https://proxy.golang.org,direct

WORKDIR /build
RUN git config --global --add safe.directory /build

COPY go.mod go.sum ./

RUN set -eux && \
  go mod download && \
  go mod tidy

COPY . ./

RUN set -eux && \
  go build -ldflags '-s -w' -o gabi cmd/gabi/main.go

FROM registry.access.redhat.com/ubi9/ubi-minimal:9.8-1780378819@sha256:ae09ecc3d754bc1726cbda3e2599cc7839e09fe1cc547ce173cf669b645be3cc

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

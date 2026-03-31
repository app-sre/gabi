FROM registry.access.redhat.com/ubi9/go-toolset:1.25.8-1774618347@sha256:0f4f6f7868962aa75dddfe4230b664bdf77071e92c43c70c824c58450e37693f AS builder
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

FROM registry.access.redhat.com/ubi9/ubi-minimal:9.7-1773939694@sha256:83006d535923fcf1345067873524a3980316f51794f01d8655be55d6e9387183

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

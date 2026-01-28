FROM registry.access.redhat.com/ubi9/go-toolset:9.7-1769430014@sha256:dc1f887fd22e4cc112f59b3173754ef97f1c6b7202a720e697b2798d8053b7be AS builder

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

FROM registry.access.redhat.com/ubi9/ubi-minimal:9.5-1742914212@sha256:ac61c96b93894b9169221e87718733354dd3765dd4a62b275893c7ff0d876869

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

FROM quay.io/app-sre/golang:1.18 as builder
WORKDIR /build
COPY . .
RUN make clean linux

FROM registry.access.redhat.com/ubi8/ubi-minimal

EXPOSE 8080
ENV DB_DRIVER=pgx
ENV DB_HOST=127.0.0.1
ENV DB_PORT=5432
ENV DB_USER=postgres
ENV DB_PASS=postgres
ENV DB_NAME=mydb
ENV DB_WRITE=false

COPY --from=builder /build/gabi .
CMD ["./gabi"]

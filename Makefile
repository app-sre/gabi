.PHONY: build linux clean test helm

all: build

build:
	go build -o gabi cmd/gabi/main.go

linux:
	CGO_ENABLED=0 GOOS=linux go build -ldflags '-s -w' -o gabi cmd/gabi/main.go

clean:
	rm -f gabi

test:
	go test ./...

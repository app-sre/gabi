.PHONY: build linux clean test

all: build

build:
	go build -o gabi cmd/gabi/main.go 

linux:
	CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -ldflags '-w' -o gabi cmd/gabi/main.go 

clean:
	rm -f gabi

test:
	go test -count=1 -v -timeout 300s ./...

docker-build:
	$(BUILD_CMD) -t ${IMG} -f Dockerfile .

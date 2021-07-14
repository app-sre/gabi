.PHONY: build linux clean

all: build

build:
	go build -o gabi cmd/gabi/main.go 

linux:
	CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -ldflags '-w' -o gabi cmd/gabi/main.go 

clean:
	rm -f gabi

docker-build:
	$(BUILD_CMD) -t ${IMG} -f Dockerfile .

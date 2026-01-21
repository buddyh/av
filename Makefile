.PHONY: build clean install test run

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.Version=$(VERSION)"

build:
	go build $(LDFLAGS) -o bin/av ./cmd/av

install:
	go install $(LDFLAGS) ./cmd/av

clean:
	rm -rf bin/

test:
	go test -v ./...

run:
	go run ./cmd/av $(ARGS)

deps:
	go mod download
	go mod tidy

build-all:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/av-darwin-amd64 ./cmd/av
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/av-darwin-arm64 ./cmd/av
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/av-linux-amd64 ./cmd/av
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/av-linux-arm64 ./cmd/av

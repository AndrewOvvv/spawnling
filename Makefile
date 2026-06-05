BINARY_CLI    := spawnling
BINARY_DAEMON := spawnlingd
MODULE        := github.com/AndrewOvvv/spawnling

.PHONY: build build-cli build-daemon test generate proto lint clean

build: build-cli build-daemon

build-cli:
	go build -o bin/$(BINARY_CLI) ./cmd/spawnling

build-daemon:
	go build -o bin/$(BINARY_DAEMON) ./cmd/spawnlingd

test:
	go test ./...

test-verbose:
	go test -v -race ./...

generate:
	go generate ./...

proto:
	protoc --go_out=. --go_opt=paths=source_relative \
	       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	       proto/spawnling.proto

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/

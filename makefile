.PHONY: server client storage server-test pkg-test

server:
	air -c .server.air.toml

client:
	air -c .client.air.toml
storage:
	air -c .storage.air.toml

build-server:
	@go build -o tmp/server cmd/server.go
build-storage:
	@go build -o tmp/storage cmd/storage.go
build-client:
	@go build -o tmp/client cmd/client.go

server-test:
	MODE=test go test ./server
pkg-test:
	MODE=test go test ./pkg
client-test:
	MODE=test go test ./client
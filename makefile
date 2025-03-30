.PHONY: server client storage server-test pkg-test

server:
	air -c .server.air.toml

client:
	air -c .client.air.toml

storage:
	air -c .storage.air.toml
server-test:
	MODE=test go test ./server
pkg-test:
	MODE=test go test ./pkg
client-test:
	MODE=test go test ./client
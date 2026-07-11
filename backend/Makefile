.PHONY: test vet fmt build run docker

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w ./cmd ./internal

build:
	CGO_ENABLED=0 go build -o bin/server ./cmd/server

run:
	go run ./cmd/server

docker:
	docker build -t broadcastplanner-backend .

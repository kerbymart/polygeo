.PHONY: build run test fmt vet

build:
	go build -o bin/polygeo ./cmd/polygeo

run:
	go run ./cmd/polygeo

test:
	go test ./...

fmt:
	gofmt -w ./cmd ./internal

vet:
	go vet ./...

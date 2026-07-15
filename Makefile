.PHONY: run build test lint fmt

run:
	go run ./cmd/rivly

build:
	CGO_ENABLED=0 go build -ldflags="-w -s" -o bin/rivly ./cmd/rivly

test:
	go test $(shell go list ./... | grep -v '/data/')

lint:
	golangci-lint run

fmt:
	go fmt $(shell go list ./... | grep -v '/data/')

.PHONY: run build build-web build-all test test-race vet lint vuln fmt

VERSION ?= dev
PKGS = $(shell go list ./... | grep -v '/data/')

run:
	go run ./cmd/rivly

build-web:
	cd web && bun install --frozen-lockfile && bun run build

build-all: build-web build

build:
	CGO_ENABLED=0 go build -ldflags="-w -s -X github.com/rivly/rivly/internal/buildinfo.version=$(VERSION)" -o bin/rivly ./cmd/rivly

test:
	go test $(PKGS)

test-race:
	go test -race $(PKGS)

vet:
	go vet $(PKGS)

lint:
	golangci-lint run

vuln:
	go run golang.org/x/vuln/cmd/govulncheck@latest $(PKGS)

fmt:
	go fmt $(PKGS)

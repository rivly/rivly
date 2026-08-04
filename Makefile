.PHONY: run build build-web build-all image packages test test-race vet lint vuln fmt

VERSION ?= dev
PKGS = $(shell go list ./... | grep -v '/data/')

run:
	go run ./cmd/rivly

build-web:
	cd web && bun install --frozen-lockfile && bun run build

build-all: build-web build

build:
	CGO_ENABLED=0 go build -ldflags="-w -s -X github.com/rivly/rivly/internal/buildinfo.version=$(VERSION)" -o bin/rivly ./cmd/rivly

image:
	docker build --build-arg VERSION=$(VERSION) -t rivly:$(VERSION) .

packages:
	@test -n "$(PKGS)" || { echo "go list found nothing, is web/dist missing? run make build-web"; exit 1; }

test: packages
	go test $(PKGS)

test-race: packages
	go test -race $(PKGS)

vet: packages
	go vet $(PKGS)

lint:
	golangci-lint run

vuln: packages
	go run golang.org/x/vuln/cmd/govulncheck@latest $(PKGS)

fmt: packages
	go fmt $(PKGS)

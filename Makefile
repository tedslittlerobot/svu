BINARY  := svu
PKG     := github.com/tedslittlerobot/svm
BIN_DIR := bin

VERSION ?= $(or $(shell git describe --tags --always --dirty 2>/dev/null),dev)
COMMIT  ?= $(or $(shell git rev-parse --verify --quiet HEAD 2>/dev/null),unknown)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
	-X $(PKG)/internal/buildinfo.version=$(VERSION) \
	-X $(PKG)/internal/buildinfo.commit=$(COMMIT) \
	-X $(PKG)/internal/buildinfo.date=$(DATE)

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ {printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: build
build: ## Build the svu binary into ./bin
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) ./cmd/svu

.PHONY: install
install: ## Install svu into $GOBIN
	go install -trimpath -ldflags "$(LDFLAGS)" ./cmd/svu

.PHONY: test
test: ## Run all tests with the race detector in random order
	go test -race -shuffle=on ./...

.PHONY: cover
cover: ## Run tests and open an HTML coverage report
	go test -race -covermode=atomic -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -n 1
	go tool cover -html=coverage.out

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run

.PHONY: fmt
fmt: ## Format all Go code
	golangci-lint fmt

.PHONY: tidy
tidy: ## Tidy and verify module dependencies
	go mod tidy
	go mod verify

.PHONY: check
check: tidy fmt lint test ## Everything CI runs; do this before pushing

.PHONY: snapshot
snapshot: ## Build release artefacts locally without publishing
	goreleaser release --snapshot --clean

.PHONY: clean
clean: ## Remove build and coverage output
	rm -rf $(BIN_DIR) dist coverage.out

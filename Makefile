BINARY  := semvu
PKG     := github.com/tedslittlerobot/semvu
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
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ {printf "  \033[36m%-17s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: build
build: ## Build the semvu binary into ./bin
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) ./cmd/semvu

.PHONY: install
install: ## Install semvu into $GOBIN
	go install -trimpath -ldflags "$(LDFLAGS)" ./cmd/semvu

.PHONY: test
test: ## Run all tests with the race detector in random order
	go test -race -shuffle=on ./...

.PHONY: cover
cover: ## Run tests and open an HTML coverage report
	go test -race -covermode=atomic -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -n 1
	go tool cover -html=coverage.out

MIN_COVERAGE ?= 95

.PHONY: cover-check
cover-check: ## Run tests with coverage, and fail if the total is below MIN_COVERAGE (95%)
	go test -race -shuffle=on -covermode=atomic -coverprofile=coverage.out ./...
	@total=$$(go tool cover -func=coverage.out | awk '/^total:/ { sub("%", "", $$3); print $$3 }'); \
	echo "Total coverage: $$total% (minimum $(MIN_COVERAGE)%)"; \
	if ! awk -v total="$$total" -v min="$(MIN_COVERAGE)" 'BEGIN { exit !(total + 0 >= min + 0) }'; then \
		echo "Coverage $$total% is below the minimum of $(MIN_COVERAGE)%. Add tests for the new code." >&2; \
		exit 1; \
	fi

FUZZ_TIME ?= 30s

.PHONY: fuzz
fuzz: ## Fuzz the version and commit parsers for FUZZ_TIME (30s) each
	go test -run='^$$' -fuzz=FuzzParse -fuzztime=$(FUZZ_TIME) ./internal/semver
	go test -run='^$$' -fuzz=FuzzParse -fuzztime=$(FUZZ_TIME) ./internal/commits

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

GO_MOD_VERSION = $(shell sed -n 's/^go //p' go.mod)

.PHONY: go-latest
go-latest: ## Move go.mod and .go-version to the latest Go release (needs network)
	go get go@latest
	go mod tidy
	sed -n 's/^go //p' go.mod > .go-version
	@echo "Now on Go $$(cat .go-version). With goenv, run: goenv install --skip-existing"

.PHONY: go-version-check
go-version-check: ## Check that .go-version matches the go line in go.mod
	@if [ "$$(cat .go-version)" != "$(GO_MOD_VERSION)" ]; then \
		echo ".go-version says $$(cat .go-version), but go.mod says $(GO_MOD_VERSION). Run make go-latest." >&2; \
		exit 1; \
	fi

.PHONY: vuln
vuln: ## Check for known vulnerabilities that semvu's code can reach (needs network)
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

.PHONY: check
check: go-version-check tidy fmt lint test ## Tidy, format, lint and test; do this before pushing

ARCHIVE_REF ?= HEAD
ARCHIVE_DIR := $(BIN_DIR)/archive-check

.PHONY: archive-check
archive-check: ## Build from git archive of ARCHIVE_REF, as Homebrew does, and check semvu knows its commit
	@mkdir -p $(ARCHIVE_DIR)
	@git archive --format=tar $(ARCHIVE_REF) | tar -x -C $(ARCHIVE_DIR)
	@# -buildvcs=false: this directory is inside the checkout, but a tarball has no git.
	@cd $(ARCHIVE_DIR) && go build -trimpath -buildvcs=false -ldflags "-X $(PKG)/internal/buildinfo.version=0.0.0-archive" -o semvu ./cmd/semvu
	@got="$$($(ARCHIVE_DIR)/semvu version)"; \
	want="semvu 0.0.0-archive (commit $$(git rev-parse '$(ARCHIVE_REF)^{commit}'), built $$(TZ=UTC git log -1 --format=%cd --date=format-local:%Y-%m-%dT%H:%M:%SZ $(ARCHIVE_REF)),"; \
	case "$$got" in \
		"$$want"*) echo "$$got" ;; \
		*) echo "A build from git archive gave: $$got" >&2; echo "want it to start: $$want" >&2; exit 1 ;; \
	esac

.PHONY: snapshot
snapshot: ## Build release artefacts locally without publishing
	goreleaser release --snapshot --clean

.PHONY: clean
clean: ## Remove build and coverage output
	rm -rf $(BIN_DIR) dist coverage.out

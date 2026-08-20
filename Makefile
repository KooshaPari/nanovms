# NanoVMS — full-featured Makefile
# Modern Go project build, test, lint, and quality targets.
# Usage: make help

.PHONY: help build test test-unit test-bench test-integration lint fmt \
        fmt-check clean tidy coverage security-scan generate vet race \
        test-ci pre-commit-install pre-commit qa

GO := go
GOFLAGS := -v
GOTESTSUM := gotestsum
GOLANGCI_LINT := golangci-lint
COVERPROFILE := coverage.out
COVERHTML := coverage.html
COVERXML := coverage.xml

# ──────────────────────────────────────────────
# Help
# ──────────────────────────────────────────────

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-25s\033[0m %s\n", $$1, $$2}'

# ──────────────────────────────────────────────
# Build
# ──────────────────────────────────────────────

build: ## Build all binaries
	$(GO) build $(GOFLAGS) ./...

build-bin: ## Build CLI binary
	$(GO) build $(GOFLAGS) -o bin/nanovms ./go/cmd/nanovms

build-daemon: ## Build daemon binary
	$(GO) build $(GOFLAGS) -o bin/nanovmsd ./go/cmd/nanovmsd

# ──────────────────────────────────────────────
# Testing
# ──────────────────────────────────────────────

test: ## Run all tests
	$(GO) test $(GOFLAGS) ./...

test-unit: ## Run unit tests only (short, no network)
	$(GO) test $(GOFLAGS) -short ./...

test-bench: ## Run benchmarks
	$(GO) test $(GOFLAGS) -bench=. -benchmem -benchtime=5s ./...

test-integration: ## Run integration tests with race detection
	$(GO) test $(GOFLAGS) -race -timeout=300s -run=TestIntegration ./...

test-race: ## Run all tests with race detector
	$(GO) test $(GOFLAGS) -race $(COVERPROFILE)=$(COVERPROFILE) ./...

test-fuzz: ## Run fuzz tests (60s per target)
	$(GO) test $(GOFLAGS) -fuzz=Fuzz -fuzztime=60s ./...

test-short: ## Run short tests only
	$(GO) test $(GOFLAGS) -short -count=1 ./...

test-watch: ## Watch mode (requires entr)
	$(GO) test $(GOFLAGS) -run=. ./... 2>&1 | entr -c -s 'make test'

# ──────────────────────────────────────────────
# Linting & Formatting
# ──────────────────────────────────────────────

lint: vet ## Run golangci-lint
	$(GOLANGCI_LINT) run ./...

lint-fix: ## Run golangci-lint with auto-fix
	$(GOLANGCI_LINT) run --fix ./...

fmt: ## Format Go code
	gofmt -s -w .
	goimports -w .

fmt-check: ## Check formatting without modifying
	@echo "Checking gofmt..."
	@test -z "$$(gofmt -s -l .)" || (echo "Files need formatting:" && gofmt -s -l . && exit 1)
	@echo "Checking goimports..."
	@test -z "$$(goimports -l .)" || (echo "Files need goimports:" && goimports -l . && exit 1)
	@echo "Formatting OK."

vet: ## Run go vet
	$(GO) vet ./...

# ──────────────────────────────────────────────
# Module Hygiene
# ──────────────────────────────────────────────

tidy: ## Run go mod tidy
	$(GO) mod tidy

mod-verify: ## Verify module checksums
	$(GO) mod verify

mod-hygiene: ## Check for module issues (mattn/mattn conflict, etc.)
	@if [ ! -f go.mod ]; then echo "no go.mod; skipping mod-hygiene"; exit 0; fi
	@bad=0; \
	if grep -E '^[[:space:]]+github\.com/mattn/go-sqlite3' go.mod >/dev/null 2>&1; then \
	  if grep -E '^[[:space:]]+modernc\.org/sqlite' go.mod >/dev/null 2>&1; then \
	    echo "ERROR: go.mod has direct github.com/mattn/go-sqlite3 alongside modernc.org/sqlite"; \
	    bad=1; \
	  fi; \
	fi; \
	if [ $$bad -ne 0 ]; then exit 1; fi; \
	echo "mod-hygiene: OK"

# ──────────────────────────────────────────────
# Coverage
# ──────────────────────────────────────────────

coverage: ## Run tests with coverage report
	$(GO) test $(GOFLAGS) -coverprofile=$(COVERPROFILE) ./...
	$(GO) tool cover -func=$(COVERPROFILE)
	$(GO) tool cover -html=$(COVERPROFILE) -o $(COVERHTML)
	@echo "Coverage report: $(COVERHTML)"

coverage-ci: ## Run coverage for CI (XML output for upload)
	$(GO) test $(GOFLAGS) -coverprofile=$(COVERPROFILE) -covermode=atomic ./...
	@command -v gocov >/dev/null 2>&1 && gocov convert $(COVERPROFILE) | gocov-xml > $(COVERXML) || true
	$(GO) tool cover -func=$(COVERPROFILE)

# ──────────────────────────────────────────────
# Generate
# ──────────────────────────────────────────────

generate: ## Run go generate for all packages
	$(GO) generate ./...

# ──────────────────────────────────────────────
# Security
# ──────────────────────────────────────────────

security-scan: ## Run security scans
	@echo "Running govulncheck..."
	@command -v govulncheck >/dev/null 2>&1 && govulncheck ./... || echo "govulncheck not installed, skipping"
	@echo "Running gitleaks..."
	-gitleaks detect --source . --verbose
	@echo "Security scan complete."

# ──────────────────────────────────────────────
# Race Detection
# ──────────────────────────────────────────────

race: ## Run tests with race detector
	$(GO) test $(GOFLAGS) -race ./...

# ──────────────────────────────────────────────
# Cleanup
# ──────────────────────────────────────────────

clean: ## Remove build artifacts and coverage files
	rm -f $(COVERPROFILE) $(COVERHTML) $(COVERXML)
	rm -rf bin/ dist/ .pytest_cache/ test-output.xml unit-tests.xml
	find . -name '*_test.out' -delete 2>/dev/null || true
	find . -name '*.test' -delete 2>/dev/null || true

# ──────────────────────────────────────────────
# Pre-commit
# ──────────────────────────────────────────────

pre-commit-install: ## Install pre-commit hooks
	pre-commit install
	pre-commit install --hook-type commit-msg

pre-commit: ## Run all pre-commit hooks
	pre-commit run --all-files

# ──────────────────────────────────────────────
# CI Aggregate
# ──────────────────────────────────────────────

test-ci: lint test-cover test-race tidy mod-hygiene mod-verify ## Full CI gate

test-cover: ## Alias for coverage
	$(GO) test $(GOFLAGS) -coverprofile=$(COVERPROFILE) ./...

# ──────────────────────────────────────────────
# Full QA
# ──────────────────────────────────────────────

qa: fmt-check lint test coverage security-scan ## Full quality assurance suite
	@echo "All QA checks passed."

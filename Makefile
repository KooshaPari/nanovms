# Go Testing Makefile
# Modern Go testing tooling

.PHONY: test test-cover test-race test-bench test-fuzz test-lint test-all test-ci tidy mod-hygiene mod-verify

# Go commands
GO := go
GOTESTSUM := gotestsum
GOLANGCI_LINT := golangci-lint
GOCOV := gocov
GOCOVXML := gocov-xml
GOMODULES := $(shell find . -name 'go.mod' -not -path '*/vendor/*')

# Test targets
test:
	$(GO) test -v ./...

test-cover:
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GOCOV) convert coverage.out | $(GOCOVXML) > coverage.xml
	$(GO) tool cover -html=coverage.out -o coverage.html

test-race:
	$(GO) test -v -race -coverprofile=coverage.out ./...

test-bench:
	$(GO) test -v -bench=. -benchmem -benchtime=5s ./...

test-short:
	$(GO) test -v -short ./...

# Fuzzing
test-fuzz:
	$(GO) test -fuzz=Fuzz -fuzztime=60s ./...

# Linting
test-lint:
	$(GOLANGCI_LINT) run ./...

test-lint-fix:
	$(GOLANGCI_LINT) run --fix ./...

# CI targets (for GitHub Actions)
test-ci: test-lint test-cover test-race tidy mod-hygiene mod-verify

# Module hygiene (DAG-T3-008: rolled out from phenodag P25)
tidy:
	$(GO) mod tidy

mod-hygiene:
	@if [ ! -f go.mod ]; then echo "no go.mod; skipping mod-hygiene"; exit 0; fi
	@bad=0; \
	if grep -E '^[[:space:]]+github\.com/mattn/go-sqlite3' go.mod >/dev/null 2>&1; then \
	  if grep -E '^[[:space:]]+modernc\.org/sqlite' go.mod >/dev/null 2>&1; then \
	    echo "ERROR: go.mod has direct github.com/mattn/go-sqlite3 alongside modernc.org/sqlite"; \
	    echo "       nanovms uses pure-Go SQLite (CGO_ENABLED=0); the mattn driver is CGO and would break the build"; \
	    bad=1; \
	  fi; \
	fi; \
	if [ $$bad -ne 0 ]; then exit 1; fi; \
	echo "mod-hygiene: OK"

mod-verify:
	$(GO) mod verify

# gotestsum targets
gotestsum:
	$(GOTESTSUM) -- -json | tee test-output.json

gotestsum-junit:
	$(GOTESTSUM) --junitfile unit-tests.xml ./...

# Watch mode (dev)
test-watch:
	$(GO) test -v -run=. ./... 2>&1 | entr -c -s 'make test'

# Module-specific tests
test-module-%:
	@echo "Testing module: $*"
	$(GO) test -v ./$*/...

# Coverage report
coverage: test-cover
	@echo "Opening coverage report..."
	@if command -v open &> /dev/null; then open coverage.html; fi

# Clean
clean-test:
	rm -f coverage.out coverage.html coverage.xml test-output.xml unit-tests.xml
	find . -name '*_test.out' -delete

## help: Show this help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

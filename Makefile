# Go Testing Makefile
# Modern Go testing tooling

.PHONY: test test-cover test-race test-bench test-fuzz test-lint test-all test-ci

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
test-ci: test-lint test-cover test-race

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

# ---------------------------------------------------------------------------
# nanovms canonical task runner (just).
#
# Mirrors Taskfile.yml so CI agents that prefer `just` can drive the same
# Go pipeline. All Go invocations pin GOCACHE to a stable, repo-scoped
# location (per the L1 audit at nanovms/STATUS_2026_06_10.md).
#
# Usage:
#   just          # show available recipes
#   just build    # `go build ./...`
#   just test     # `go test -race -coverprofile=coverage.out ./...`
#   just ci       # vet + test + lint (full CI gate)
#
# Authored by: L2 subagent #22 (V3 DAG FLEET_100TASK_DAG_V3.md task #22).
# ---------------------------------------------------------------------------

set dotenv-load := false
set positional-arguments

export GOCACHE := "/private/tmp/nanovms-gocache"
export GOFLAGS := "-mod=readonly"
export CGO_ENABLED := "0"

# Default recipe: list available commands.
default:
    @just --list
    @echo "detected languages: {{detect_languages}}"
    @echo "GOCACHE=$GOCACHE"

# ---------------------------------------------------------------------------
# L2-22 canonical Go tasks
# ---------------------------------------------------------------------------

# Build all Go packages and binaries (`go build ./...`).
build:
    go build ./...

# Run Go tests with race detection and coverage.
test:
    go test -race -coverprofile=coverage.out -covermode=atomic ./...

# Coverage report (SSOT for how to measure coverage).
coverage:
    go test -coverprofile=coverage.out -covermode=atomic ./...
    go tool cover -func=coverage.out

# Run `go vet ./...`.
vet:
    go vet ./...

# Run golangci-lint with the repo config; no-op if the binary is missing.
lint:
    @if command -v golangci-lint >/dev/null 2>&1; then \
        golangci-lint run --config=.github/golangci.yml ./...; \
    else \
        echo "golangci-lint not installed; skipping (install via 'brew install golangci-lint')"; \
    fi

# Print per-function coverage summary from coverage.out (run `just test` first).
cov:
    go tool cover -func=coverage.out

# Tidy go.mod / go.sum.
tidy:
    go mod tidy

# Full CI gate: vet + test (race + coverage) + lint.
ci: vet test lint
    @echo "ci: all gates passed"

# ---------------------------------------------------------------------------
# Polyglot helpers (mirrors Taskfile.yml's polyglot section).
# ---------------------------------------------------------------------------

# Build all Go packages and binaries (alias of `build`).
backend-build: build

# Run Go tests with race detection, verbose (alias of `test`).
backend-test:
    go test -v -race ./...

# Run gofmt + go vet + golangci-lint.
backend-lint:
    @test -z "$(gofmt -l $(find . -name '*.go' -not -path './vendor/*'))"
    go vet ./...
    @if command -v golangci-lint >/dev/null 2>&1; then \
        golangci-lint run --config=.github/golangci.yml ./...; \
    else \
        echo "golangci-lint not installed; skipping"; \
    fi

# Format Go files in place.
fmt:
    go fmt ./...

# Verify Go files are gofmt-clean (CI gate).
fmt-check:
    @test -z "$(gofmt -l $(find . -name '*.go' -not -path './vendor/*'))"

# Build the Rust SDK.
sdk-rs-build:
    cd {{rust_dir}} && cargo build --all-features

# Run Rust SDK tests.
sdk-rs-test:
    cd {{rust_dir}} && cargo test --all-features

# Run cargo fmt + clippy on the Rust SDK.
sdk-rs-lint:
    cd {{rust_dir}} && cargo fmt -- --check
    cd {{rust_dir}} && cargo clippy --all-features -- -D warnings

# Build web / docs assets.
web-build:
    @if [ -f package.json ] && grep -q '"docs:build"' package.json; then \
        {{node_pm}} run docs:build; \
    else \
        echo "No docs:build script found; skipping"; \
    fi

# Run web lint if available.
web-lint:
    @if [ -f package.json ] && grep -q '"lint"' package.json; then \
        {{node_pm}} run lint; \
    else \
        echo "No lint script found; skipping"; \
    fi

# Aggregate: build backend + Rust SDK + web.
all-build: backend-build sdk-rs-build web-build

# Aggregate: test backend + Rust SDK.
all-test: backend-test sdk-rs-test

# Aggregate: lint backend + Rust SDK + web.
all-lint: backend-lint sdk-rs-lint web-lint

# Remove build outputs and test caches.
clean:
    @if [ -f go.mod ]; then \
        go clean -cache -testcache || echo "go clean cache cleanup skipped"; \
        find . \( -name coverage.out -o -name coverage.html -o -name coverage.xml -o -name test-output.json -o -name unit-tests.xml \) -type f -delete; \
    fi
    @if [ -f package.json ]; then \
        rm -rf node_modules docs/.vitepress/dist dist build coverage; \
    fi
    @if [ -f sdk/rust/Cargo.toml ] || [ -f Cargo.toml ]; then \
        cd {{rust_dir}} && cargo clean || echo "cargo clean skipped"; \
    fi

# ---------------------------------------------------------------------------
# Helpers (expression functions used as `{{ ... }}` substitutions above).
# ---------------------------------------------------------------------------

# Project languages, space-separated (e.g. "go node"). Detected from go.mod and package.json.
detect_languages := `\
    languages=""; \
    if [ -f go.mod ]; then languages="${languages}go "; fi; \
    if [ -f package.json ]; then languages="${languages}node "; fi; \
    if [ -n "$languages" ]; then printf "%s" "$languages" | sed 's/[[:space:]]*$//'; else printf "unknown"; fi`

# Node package manager (npm | pnpm | yarn | bun), detected from lockfiles.
node_pm := `\
    if [ -f bun.lockb ] || [ -f bun.lock ]; then printf "bun"; \
    elif [ -f pnpm-lock.yaml ]; then printf "pnpm"; \
    elif [ -f yarn.lock ]; then printf "yarn"; \
    else printf "npm"; fi`

# Directory containing the Rust SDK (default: sdk/rust).
rust_dir := `if [ -d sdk/rust ]; then printf "sdk/rust"; else printf "."; fi`

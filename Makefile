# nanovms Makefile
.PHONY: all build build-rust build-go build-docs test test-rust test-go lint lint-rust lint-go format clean help

all: lint test

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "} {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: build-rust build-go build-docs

build-rust:
	cargo build --workspace

build-go:
	go build ./cmd/...
	go build ./pkg/...

build-docs:
	cd docs && npm install && npm run docs:build

test: test-rust test-go

test-rust:
	cargo test --workspace

test-go:
	go test ./... -v

lint: lint-rust lint-go

lint-rust:
	cargo clippy --workspace --all-targets -- -D warnings

lint-go:
	@which staticcheck > /dev/null 2>&1 && staticcheck ./... || echo "staticcheck not installed, skipping"

format:
	cargo fmt --all
	go fmt ./...

clean:
	cargo clean
	rm -rf docs/node_modules docs/.vitepress/dist
	go clean -cache

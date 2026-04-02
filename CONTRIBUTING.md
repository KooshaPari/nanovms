# Contributing to devenv-abstraction

Thank you for your interest in contributing!

## Development Setup

1. **Clone the repository**
   ```bash
   git clone https://github.com/KooshaPari/devenv-abstraction.git
   cd devenv-abstraction
   ```

2. **Install Go 1.22+**
   ```bash
   # On macOS
   brew install go

   # Verify
   go version
   ```

3. **Install dependencies**
   ```bash
   go mod download
   ```

4. **Build**
   ```bash
   go build ./...
   ```

5. **Run tests**
   ```bash
   go test ./...
   ```

## Project Structure

```
devenv-abstraction/
├── cmd/              # CLI entry points
├── internal/
│   ├── adapters/     # Platform adapters (mac, windows, linux, wasm)
│   ├── core/        # Core business logic
│   ├── domain/      # Domain models
│   └── ports/       # Port interfaces
├── pkg/             # Public packages
└── docs/           # VitePress documentation
```

## Code Style

- Run `go fmt` before committing
- Follow [Effective Go](https://golang.org/doc/effective_go)
- Add tests for new functionality
- Document exported functions

## Platform Adapter Guidelines

Each platform adapter must implement the `SandboxPort` interface:

```go
type SandboxPort interface {
    Info() (PlatformInfo, error)
    Create(ctx context.Context, cfg SandboxConfig) (*Sandbox, error)
    Delete(ctx context.Context, id string) error
    List(ctx context.Context) ([]Sandbox, error)
}
```

## Submitting Changes

1. Fork the repository
2. Create a feature branch: `git checkout -b feat/my-feature`
3. Make your changes
4. Run tests: `go test ./...`
5. Commit with clear messages
6. Push and open a PR

## Reporting Issues

Use GitHub Issues for bugs and feature requests. Include:
- Go version
- Operating system
- Relevant logs or error messages

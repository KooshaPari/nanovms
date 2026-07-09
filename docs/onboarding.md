# nanovms Onboarding - 5 minute quickstart

## Prerequisites
- Go 1.21+
- Git

## Clone and build
```bash
git clone https://github.com/KooshaPari/nanovms
cd nanovms
./scripts/dev-bootstrap.sh
```

## Run tests
```bash
go test ./...
```

## Try the daemon
```bash
go run ./cmd/nanovms daemon serve
```

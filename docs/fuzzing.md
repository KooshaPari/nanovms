# nanovms Fuzzing

go-fuzz targets for the most-trafficked value types and entry points.

## Tooling

- `github.com/dvyukov/go-fuzz` - coverage-guided fuzzing (Go native)
- `go-fuzz-build` - builds the fuzz target
- `go-fuzz` - runs the fuzzer
- `goccy/go-fuzz` (alternative) - simpler API

## Setup

```bash
go get -u github.com/dvyukov/go-fuzz/go-fuzz
go get -u github.com/dvyukov/go-fuzz/go-fuzz-build
go-fuzz-build ./internal/adapters/sandbox/fuzztest
```

## Targets

### FuzzAdapterCreate (sandbox_test.go)

```go
//go:build gofuzz
package fuzztest

func FuzzAdapterCreate(data []byte) int {
    a := sandbox.NewAdapter()
    cfg := domain.SandboxConfig{Name: string(data)}
    _, err := a.Create(nil, cfg)
    if err != nil { return 0 }  // invalid input is acceptable
    return 0
}
```

## Running

```bash
mkdir -p fuzzworkdir/corpus
go-fuzz -bin=./sandbox-fuzz -workdir=./fuzzworkdir -procs=4
```

The fuzzer:
1. Starts with empty corpus
2. Generates random inputs, runs `FuzzAdapterCreate`
3. Tracks coverage; mutates inputs to maximize new code paths
4. Reports any panic/crash as a finding
5. Saves interesting inputs to corpus

## Coverage-guided

Each target gets:
- Crash inputs (panic, fail) - immediate report
- New-coverage inputs (interesting) - saved to corpus
- Coverage percentage - per-function, per-package

## Crashes

```bash
# Reproduce a crash
go-fuzz -bin=./sandbox-fuzz -workdir=./fuzzworkdir -test=crash-abc123
```

## CI integration

```yaml
# .github/workflows/fuzz.yml
- name: Fuzz
  run: |
    go-fuzz-build ./internal/adapters/sandbox/fuzztest
    timeout 60 go-fuzz -bin=./sandbox-fuzz -workdir=./fuzzworkdir -procs=1
```

## Coverage target

- All public entry points (Create, Start, Stop, List, Get) have a fuzz target
- 10 minutes of fuzzing per PR finds no new panics
- Found bugs become regression tests

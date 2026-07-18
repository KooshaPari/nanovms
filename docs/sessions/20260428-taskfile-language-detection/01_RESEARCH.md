# Research

## Repo signals

- `go.mod` is present at the repository root, so the primary implementation language is Go.
- `package.json` is also present, but it serves the docs tooling (`vitepress`) rather than the core runtime.
- `README.md` documents Go build usage via `go build ./cmd/nanovms`.
- `Makefile.go` contained Go test and lint targets but used Makefile syntax, so it needed to be renamed to `Makefile`.

## Validation notes

- `task -l` recognized `build`, `test`, `lint`, and `clean`.
- `task clean` succeeded.
- `task build` exposed the root `Makefile.go` parse issue when using `./...`; renaming it to `Makefile` lets normal Go package discovery work.
- After the Makefile rename, `task build`, `task test`, and `task lint` reached repo code and exposed type drift in `internal/domain/sandbox.go` (`VMType`, `NativeSandboxConfig`, `VMinstance`, pointer-to-interface assertion) and `internal/adapters/windows/windows.go` (unused `setRuntime`).
- Those initial compile issues were fixed before PR publication.
- `task test` then reached broader existing adapter drift in `internal/adapters/mac`, `internal/adapters/sandbox`, and `internal/adapters/wasm`.
- Local `task build` was blocked once by machine disk pressure while writing Go build cache artifacts.

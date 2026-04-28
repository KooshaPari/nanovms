# Research

## Repo signals

- `go.mod` is present at the repository root, so the primary implementation language is Go.
- `package.json` is also present, but it serves the docs tooling (`vitepress`) rather than the core runtime.
- `README.md` documents Go build usage via `go build ./cmd/nanovms`.
- `Makefile.go` already contains Go test and lint targets.
- Root `Makefile.go` is a makefile, not compilable Go, so Go task package discovery needs to enumerate source directories instead of using `./...` from the root.

## Validation notes

- `task -l` recognized `build`, `test`, `lint`, and `clean`.
- `task clean` succeeded.
- `task build` exposed the root `Makefile.go` parse issue when using `./...`; package-directory discovery avoids that repo-specific helper file.
- After package discovery was fixed, `task build`, `task test`, and `task lint` reached repo code and failed on existing compile issues in `internal/domain/sandbox.go` (`VMType`, `NativeSandboxConfig`, `VMinstance`, pointer-to-interface assertion) and `internal/adapters/windows/windows.go` (unused `setRuntime`).

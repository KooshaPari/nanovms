# nanovms Release Process

`goreleaser` produces cross-platform binaries and GitHub releases.

## Workflow

1. Tag: `git tag v0.X.Y && git push --tags`
2. CI runs `goreleaser release --clean`
3. Cross-compiled binaries published to GitHub release
4. SBOM attached via cosign

## Binaries

| OS | Arch | Filename |
|----|------|----------|
| linux | amd64 | `nanovms_v0.X.Y_linux_amd64.tar.gz` |
| linux | arm64 | `nanovms_v0.X.Y_linux_arm64.tar.gz` |
| darwin | amd64 | `nanovms_v0.X.Y_darwin_amd64.tar.gz` |
| darwin | arm64 | `nanovms_v0.X.Y_darwin_arm64.tar.gz` |
| windows | amd64 | `nanovms_v0.X.Y_windows_amd64.tar.gz` |

## Conventional Commits

- `feat:` minor bump
- `fix:` patch bump
- `feat!:` major bump
- `chore:`, `docs:`, `test:`, `ci:` no bump

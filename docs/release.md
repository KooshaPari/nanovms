# nanovms Release Process

`goreleaser` produces cross-platform binaries and GitHub releases.

## Workflow

1. Tag: `git tag v0.X.Y && git push --tags`
2. CI runs `goreleaser release --clean`
3. Cross-compiled binaries published to GitHub release
4. SPDX SBOM uploaded as a workflow artifact and release asset
5. GitHub artifact attestations are generated for the release outputs and SBOM

The release workflow is the source of truth for generated evidence. A release
must not be promoted until the workflow completes successfully and its
attestation summary is visible in GitHub.

## Rollback

Do not overwrite a published release. To roll back a bad release:

1. Stop promotion and record the affected tag and attestation URL.
2. Repoint consumers to the last known-good immutable tag.
3. Remove or mark the bad release as unavailable according to the repository
   release policy; retain its assets and attestations for auditability.
4. Open a corrective change, rerun the release workflow, and verify a new
   attestation before restoring promotion.

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

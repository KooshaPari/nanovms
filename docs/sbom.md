# nanovms SBOM

SPDX JSON SBOM on every release.

## Generation

```bash
syft scan . -o spdx-json=nanovms-sbom.spdx.json
```

The release workflow uploads the SBOM as a named workflow artifact and adds it
to the GitHub release when GoReleaser creates the release. The same file is a
subject of the GitHub artifact attestation.

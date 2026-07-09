# nanovms SBOM

CycloneDX-format SBOM on every release.

## Generation

```bash
syft scan . -o cyclonedx-json=sbom.json
```

The SBOM is attached to every GitHub release.

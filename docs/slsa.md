# nanovms Supply Chain Evidence

- Hardened GitHub Actions runners
- Reproducible builds
- Pinned workflow actions
- GitHub artifact attestations for release outputs and the SPDX SBOM

The release workflow grants `id-token: write` and `attestations: write` only to
the release job. It produces an attestation bundle through
`actions/attest-build-provenance`; verify the bundle from the workflow summary
before promoting a release. This repository does not claim SLSA L3 or cosign
signatures until independently verified evidence is present.

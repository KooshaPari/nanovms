# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `Adapter.Logs` and `Adapter.Exec`: implemented with native sandbox delegation
- `gvisorAdapter.Logs` and `gvisorAdapter.Exec`: implemented via runsc log/exec commands
- `landlockAdapter`, `seccompAdapter`, `wasmtimeAdapter`: implemented Get/Logs/Exec/Metrics with runtime-appropriate mechanisms
- `nativeSandboxAdapter`: implemented Get/Logs/Exec/Metrics with nsenter and journald
- Helper functions `logsForSandbox`, `execInSandbox`, `metricsForSandbox` for per-sandbox operations

### Changed
- `generateID`: replaced timestamp-based IDs with cryptographically random UUIDs (crypto/rand)
- `checkLandlockSupport`: replaced stub with real kernel version check (/proc/sys/kernel/osrelease >= 5.13 and /sys/kernel/security/landlock)
- `Adapter.Get`: now performs proper map lookup instead of always returning "not found"
- `Adapter.Metrics`: now performs proper map lookup and returns real metrics
- `linux.execNative`: fixed hardcoded `"cmd[0]"` string literal bug (was using string literal instead of actual command)

### Fixed
- `linux/execNative`: WASM sandbox command execution now passes correct arguments instead of literal string `"cmd[0]"`

## [0.1.0] - 2026-08-19

Initial release. Captures every change merged into `main` from the
`phenotype-shared` intake through the upstream sync. Includes the AgilePlus
bootstrap (sprint tracking, pillar scorecard, quality gates, weekly CI
workflow) and the absorbtion of the `substrate` probe layer.

### Added

#### AgilePlus bootstrap (Sprint 1, S1-01..S1-05)
- `agileplus/AGILEPLUS.md` master configuration describing sprint cadence,
  roles, quality gates, and the 31-pillar scorecard.
- `agileplus/sprint-current.md` for Sprint 1 (2026-08-19 -> 2026-08-26).
- `agileplus/sprint-retrospective-template.md` retro template.
- `agileplus/backlog.md` with the top 10 prioritized backlog items.
- `agileplus/pillars/31-pillar-scorecard.json` with all 31 pillars scored
  (average 5.32, grade 5.3 D).
- `agileplus/quality-gates.yml` defining the `lint`, `test`, `security`,
  `build`, `docs-build`, and `coverage` gates.
- `agileplus/velocity.md` velocity tracker.
- `agileplus/CODEOWNERS-pillars` pillar-to-owner mapping.
- `.github/workflows/agileplus-pillar-scorecard.yml` weekly workflow that
  re-scores the pillars and posts a summary on PRs that touch `agileplus/`.
- `docs/AGILEPLUS-SETUP.md` one-page explainer.
- `CODE_OF_CONDUCT.md` confirmed as Contributor Covenant v2.1.

#### Substrate + nanovms merge
- Merge of upstream `nanovms` `main` into the substrate probe branch
  (`419f048`, "merge: integrate upstream nanovms main with local substrate
  probe work").
- Absorbed `port-adapter-shim` and `ffi_utils` from the shared substrate
  workspace (`3841c03`, PR #168).
- Phenotype-manifest intake from `phenotype-shared` (`b966c7d`, PR #140).

#### Sandbox / VM adapters
- Firecracker adapter: VM lifecycle, snapshot, memory ballooning, vCPU,
  network configuration.
- Linux adapter: unshare namespaces, cgroups v2, rootless containers, overlay
  filesystem, volume management.
- macOS adapter: Lima/Colima, Virtualization.framework, Rosetta 2 (ARM).
- Windows adapter: WSL2, Hyper-V, Cloud Hypervisor.
- gVisor adapter: runsc detection, Sentry process, Gofer filesystem proxy,
  network proxy.
- landlock adapter: Landlock LSM detection, filesystem restriction rules,
  rule merging.
- bwrap/firejail adapter: bubblewrap + firejail profile execution, namespace
  isolation.
- seccomp adapter: seccomp-bpf profile generation, syscall allowlist, error
  handling.

#### CI / quality tooling
- `.circleci/config.yml` CircleCI pipeline.
- `.github/workflows/ci.yml` main CI pipeline.
- `.github/workflows/scorecard.yml` OSSF scorecard workflow.
- `.github/workflows/trunk-check.yml` trunk-based quality check.
- `.github/workflows/audit.yml` repo audit workflow.
- `.github/workflows/cargo-deny.yml` cargo-deny gate.
- `.github/workflows/cross-compile-ci.yml` cross-compile matrix.
- `.github/workflows/release.yml` goreleaser-based release pipeline.
- `.github/workflows/infisical.yml` Infisical integration.
- `.mergify.yml` Mergify grammar-based auto-merge policy (multiple
  format upgrades via #142, #145-149, #165, #169).
- `.pre-commit-config.yaml` pre-commit hooks.
- `renovate.json` + `renovate.json5` Renovate config.
- `trunk.yaml` trunk-based config management.
- `.gitleaks.toml`, `gitleaks.toml`, `.trufflehog.yml`, `trufflehog.yml`
  secret-scanning configs.
- `.devcontainer/` reproducible dev container.
- `lefthook.yml` git-hook runner.
- `dprint.json` dprint formatter config.
- `.github/stale.yml` stale-issue rot policy.

### Changed

#### Mergify grammar upgrades (PRs #142, #145-149, #165, #169)
- Multiple `ci(mergify): upgrade configuration to current format` updates to
  keep the auto-merge policy valid against Mergify's current grammar.
- `cd8c033` `ci: make Mergify base policy valid` foundational fix.
- `2d7564b` `chore(ci): remove broken trunk-check.yml workflow` cleanup.

#### Dependency bumps
- `build(deps): bump golangci/golangci-lint-action` (#157)
- `build(deps): bump aquasecurity/trivy-action` (#156)
- `build(deps): bump dorny/paths-filter` (#155)
- `build(deps): bump Swatinem/rust-cache` (#154)
- `build(deps): bump the major group across 1 directory with 3 updates` (#153)
- `build(deps): bump the minor-and-patch group across 1 directory with 2
  updates` (#152)
- `build(deps): bump golang.org/x/sys` (#124)
- `build(deps): bump ossf/scorecard-action` (#126)
- `build(deps): bump the major group across 1 directory with 8 updates` (#130)
- `build(deps): bump the major group across 1 directory with 3 updates` (#141)

#### Docs
- `f47484b` `docs(quickstart): make Podman the supported local engine` (#132).

#### Workflow hygiene
- Stable lint/test gate names (`a90ab98`, `df128af`).
- `84f24f1` `ci: add Infisical integration workflow`.

### Fixed

- `linux/execNative`: WASM sandbox command execution now passes correct
  arguments instead of literal string `"cmd[0]"` (carried over from
  Unreleased).

### Notes

- No git tags existed prior to this release. The version string `0.1.0` is
  applied retroactively against the `main` tip at `419f048`.
- 31-pillar scorecard baseline: **average 5.32, grade 5.3 (D)**. See
  `agileplus/pillars/31-pillar-scorecard.json`.
- The 5 red-band pillars to focus next sprint are **i18n (2)**,
  **Agile PM (3)**, **Accessibility (3)**, **Mobile (3)**, **CoC (3)**.

[Unreleased]: https://github.com/KooshaPari/nanovms/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/KooshaPari/nanovms/releases/tag/v0.1.0
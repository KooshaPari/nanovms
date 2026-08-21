# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0](https://github.com/KooshaPari/nanovms/compare/v0.1.0...v0.2.0) (2026-08-21)


### Features

* add Docker/gVisor/Firecracker sandbox adapters, i18n locales, perf baselines ([384fadf](https://github.com/KooshaPari/nanovms/commit/384fadf672a91363a6cb5003c11dde1e756726f1))
* add e2e tests, release-please, SBOM workflow, and i18n locales ([5da5475](https://github.com/KooshaPari/nanovms/commit/5da5475e2fcdab286b42d623e999801171624544))
* **ci:** add Docker-in-Docker integration tests, perf trend tracking, and SLA/SLO docs ([34acb5a](https://github.com/KooshaPari/nanovms/commit/34acb5a92427ae654150e83075fc7bcfdaeb192a))
* **desktop:** M1 - AgilePlus desktop shell with spec editor, pillar scorecards, and sprint tracker ([69aeb05](https://github.com/KooshaPari/nanovms/commit/69aeb0562d9da7f7d96e334132bf3adb3a3bfc0a))
* **devex:** add ADRs, DORA metrics, Docker dev env, and incident response playbook ([b7a3113](https://github.com/KooshaPari/nanovms/commit/b7a3113c2b328a17d5421a5c63fba240723feaa5))
* **i18n:** add internationalization scaffolding with English locale ([123874b](https://github.com/KooshaPari/nanovms/commit/123874b9623f7469b70259d59ccfa643b14e3631))
* **infra:** add OpenTelemetry, chaos testing, perf dashboard, and multi-region docs ([657d7c6](https://github.com/KooshaPari/nanovms/commit/657d7c65612c7f396336e65b44797bc705aac466))
* **security:** add Trivy scan, issue templates, benchmarks, and wire i18n into agentctl ([191f8a3](https://github.com/KooshaPari/nanovms/commit/191f8a3fc17c6a7cd627a35602ab0462b4a8bdcc))
* **sre:** add chaos CI gate, Terraform IaC, SLO burn rate alerting, and OTel collector config ([f248d7f](https://github.com/KooshaPari/nanovms/commit/f248d7f2bd59d2268614f749df3a085b6e4d6f56))
* **sre:** add SLO alerting, OTel deployment scripts, terraform validate CI ([23ee367](https://github.com/KooshaPari/nanovms/commit/23ee3672ff35c1fdc9dc66cd446e33818282fa2a))
* **sre:** add SLO monitoring, Terraform CI validation, and performance dashboard ([c19e6ea](https://github.com/KooshaPari/nanovms/commit/c19e6ea5bf162deefa94ec63934850e0a1b4bbb8))
* **testing:** add adapter integration tests, fuzz harnesses, 3 new locales, and codeowners verification ([54653f9](https://github.com/KooshaPari/nanovms/commit/54653f920112e01c992736bf7fddae882f7bc30b))
* **tests:** expand test coverage for runtime, orchestrate, and sandbox registry; add metrics + observability ([e5a4f00](https://github.com/KooshaPari/nanovms/commit/e5a4f007cda428d564e9698ba1927c2dd2afaa9f))
* **validation:** M2 - Spec validation, trend charts, version history, backlog board, gate history, and pillar comparison ([4fc6f19](https://github.com/KooshaPari/nanovms/commit/4fc6f192437f2a90270744386ef2c6a1a1be5dd8))


### Bug Fixes

* **ci:** remove invalid release-please inputs (default-branch, changelog-types) ([ad020a3](https://github.com/KooshaPari/nanovms/commit/ad020a3cf4e78f7c4aa862dfa5299d482261e504))

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

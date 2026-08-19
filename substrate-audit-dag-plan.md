# Substrate Audit Remediation: NanoVMS

## Score Summary

| Metric | Value |
|---|---|
| **Total Pillars** | 140 |
| **Satisfied** | 62 (44.3%) |
| **Partial** | 35 (25.0%) |
| **Missing** | 43 (30.7%) |
| **Score** | 56.8% (C-) |

### Target Progression

| Phase | Target Grade | Target Score |
|---|---|---|
| Phase 0 | C | 65% |
| Phase 1 | C+ | 72% |
| Phase 2 | B- | 78% |
| Phase 3 | B | 82% |

---

## Phase 0: Hygiene Backbone (Config files, governance docs)

**Target: 65% → Grade C**
**Est. effort: 8h**

| ID | Task | Effort | Rationale |
|---|---|---|---|
| NV-P0-01 | Create `CONTRIBUTING.md` with dev setup | 1h | DOC-05 missing |
| NV-P0-02 | Create `docs/FAQ.md` and `docs/GLOSSARY.md` | 1h | DOC-16/17 missing |
| NV-P0-03 | Create `docs/security/DEP_POLICY.md` | 1h | SEC-16 missing |
| NV-P0-04 | Add Rust `rust-version` to all crates | 0.5h | CQ-03 partial, SC-07 partial |
| NV-P0-05 | Create root `CLAUDE.md` | 0.5h | AR-01 missing |
| NV-P0-06 | Add Rust clippy + fmt CI workflow | 1h | CI-04/05 missing |
| NV-P0-07 | Add issue templates (bug_report, feature_request) | 0.5h | CI-14 partial |
| NV-P0-08 | Create `docs/operations/runbook.md` | 1h | DOC-12 partial |
| NV-P0-09 | Create `docs/troubleshooting/known-issues.md` | 1h | DOC-13 partial |
| NV-P0-10 | Add `CODE_OF_CONDUCT.md` | 0.5h | Governance gap |

---

## Phase 1: Observability + Testing (Tracing, fuzz, coverage, OpenAPI)

**Target: 72% → Grade C+**
**Est. effort: 20h**

| ID | Task | Effort | Rationale |
|---|---|---|---|
| NV-P1-01 | Add property-based tests (Go testing/quick + Rust proptest) | 3h | TEST-03 missing |
| NV-P1-02 | Add fuzz harness for sandbox config parsing | 2h | TEST-04 missing |
| NV-P1-03 | Add mutation testing (Go mutesting + cargo-mutants) | 3h | TEST-08 missing |
| NV-P1-04 | Add coverage threshold gate in CI | 1h | TEST-05 partial |
| NV-P1-05 | Add snapshot/approval testing | 2h | TEST-07 missing |
| NV-P1-06 | Add OpenTelemetry tracing integration | 3h | OBS-04 missing |
| NV-P1-07 | Add Prometheus /metrics endpoint | 2h | OBS-05 missing |
| NV-P1-08 | Add request-id middleware | 1h | OBS-06 missing |
| NV-P1-09 | Add W3C trace context propagation | 1h | OBS-07 missing |
| NV-P1-10 | Add structured logging consistency | 1h | OBS-03 partial |
| NV-P1-11 | Add Rust test runner to CI (cargo test) | 0.5h | TEST-06 partial |
| NV-P1-12 | Add MSRV test job | 0.5h | TEST-12 missing |

---

## Phase 2: Security + Ops (SBOM, SLSA, Docker, runbook)

**Target: 78% → Grade B-**
**Est. effort: 16h**

| ID | Task | Effort | Rationale |
|---|---|---|---|
| NV-P2-01 | Add CodeQL CI workflow | 1h | SEC-07 missing |
| NV-P2-02 | Add CycloneDX SBOM generation to release | 1h | SEC-08 missing |
| NV-P2-03 | Add OWASP ZAP DAST workflow for API | 2h | SEC-13 missing |
| NV-P2-04 | Create threat model document | 2h | SEC-14 partial |
| NV-P2-05 | Add SLSA provenance to release workflow | 2h | SC-10 missing |
| NV-P2-06 | Add cosign signing for Go binaries | 2h | RE-13 missing |
| NV-P2-07 | Add release verification workflow | 1h | CI-11 missing |
| NV-P2-08 | Add post-deploy smoke test | 1h | RE-11 missing |
| NV-P2-09 | Add SLO/SLI definitions | 1h | RE-10 missing |
| NV-P2-10 | Add rollback playbook | 1h | RE-09 missing |
| NV-P2-11 | Add Homebrew formula publishing | 1h | RE-07 missing |
| NV-P2-12 | Add conventional commit lint to CI | 1h | CI-08 partial |

---

## Phase 3: Advanced (OTel, dashboards, benchmarks, mut testing)

**Target: 82% → Grade B**
**Est. effort: 18h**

| ID | Task | Effort | Rationale |
|---|---|---|---|
| NV-P3-01 | Add sccache for Rust builds in CI | 1h | DX-03 missing |
| NV-P3-02 | Create devcontainer configuration | 1h | DX-09 missing |
| NV-P3-03 | Add `rust-project.json` for rust-analyzer | 0.5h | DX-05 missing |
| NV-P3-04 | Create Justfile for common tasks | 1h | DX-10 missing |
| NV-P3-05 | Add comprehensive load/soak test suite | 3h | TEST-09 partial |
| NV-P3-06 | Add continuous benchmark dashboard | 2h | RE improvement |
| NV-P3-07 | Add cost tracking for VM/sandbox operations | 2h | AR-07 missing |
| NV-P3-08 | Add decision tracing for sandbox routing | 2h | AR-08 missing |
| NV-P3-09 | Add UX friction logging for CLI | 1h | AR-09 missing |
| NV-P3-10 | Add automated continuous audit workflow | 2h | AR-10 partial |
| NV-P3-11 | Create API SDK packages (npm, PyPI) | 2h | Distribution |
| NV-P3-12 | Update existing audit_scorecard.json to 140-pillar (this file) | 0.5h | Audit gap filled |

---

## Total Estimated Effort

| Phase | Effort | Target Grade |
|---|---|---|
| Phase 0: Hygiene | 8h | C (65%) |
| Phase 1: Observability + Testing | 20h | C+ (72%) |
| Phase 2: Security + Ops | 16h | B- (78%) |
| Phase 3: Advanced | 18h | B (82%) |
| **Total** | **62h** | **B (82%)** |

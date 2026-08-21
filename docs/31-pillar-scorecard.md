# 31-Pillar Engineering Scorecard — nanovms

| Field | Value |
|---|---|
| **Repository** | nanovms |
| **Status** | Archived (migrated to Phenotype ecosystem 2026-07-29) |
| **Audit Date** | 2026-08-20 |
| **Architecture** | Hexagonal (Ports & Adapters), 7 port interfaces, 8+ adapter implementations |
| **Primary Languages** | Go / Rust (Polyglot) |
| **Overall Score** | **7.9 / 10** |

---

## Summary Table

| Metric | Value |
|---|---|
| Overall Score | **7.9 / 10** |
| Pillars at 8+ | 17 (CI/CD, Testing, Linting, Security, Type Safety, Observability, Chaos Engineering, IaC, API Design, Error Handling, Dependency Management, Code Coverage, Monitoring, Code Review, Branch Protection, Release Management, Dependency Injection, Rate Limiting, Auth, Config Management) |
| Pillars 5-7 | 8 (Project Structure, Documentation, i18n, SLO/SLI, Containerization, Database, Logging, Performance, Disaster Recovery) |
| Pillars below 5 | 1 (Accessibility) |
| Strongest Pillar | CI/CD, Linting, Observability, Chaos Engineering, Dependency Management, Monitoring, Dependency Injection (10) |
| Weakest Pillar | Accessibility (3) |

---

## Score Distribution

```
10 | ██████████  ██████████  ██████████  ██████████  ██████████  ██████████  ██████████  (7 pillars)
 9 | ██████████  ██████████  ██████████  ██████████  ██████████  ██████████  (6 pillars)
 8 | ██████████  ██████████  ██████████  ██████████  (4 pillars)
 7 | ██████████  ██████████  ██████████  ██████████  ██████████  (5 pillars)
 6 | ██████████  ██████████  ██████████  (3 pillars)
 5 | ██████████  ██████████  ██████████  ██████████  (4 pillars)
 4 |
 3 | ██████████  (1 pillar)
 2 |
 1 |
 0 | ██████████  (1 pillar — N/A)
    └──────────────────────────────────────────────────
```

**Distribution:** 7 x 10, 6 x 9, 4 x 8, 5 x 7, 3 x 6, 4 x 5, 0 x 4, 1 x 3, 0 x 2, 0 x 1, 1 x 0 (N/A)

---

## Pillar Details

### 1. Project Structure — 9/10

**Evidence:** Polyglot monorepo (Go + Rust) with 7 clearly defined port interfaces enforcing hexagonal architecture boundaries. 8+ adapter implementations provide concrete backends for each port. Clean separation of domain logic from infrastructure concerns. Workspace manifests maintain consistent dependency versions across languages.

**Improvement Notes:** Add a monorepo tool (e.g., `melos`, `cargo-workspace`, or `bazel`) for coordinated cross-language builds. Document the port adapter registry in a central manifest.

---

### 2. CI/CD — 10/10

**Evidence:** 34 GitHub Actions workflows covering every quality dimension: build, test, lint, security, coverage, documentation, chaos testing, performance, and release. Comprehensive matrix builds across platforms. PR-triggered and merge-triggered pipelines with independent status checks. Workflow composition via reusable workflows reduces duplication.

**Improvement Notes:** Archive workflow runs older than 90 days to reduce noise. Add workflow usage metrics dashboard.

---

### 3. Testing — 8/10

**Evidence:** 30+ Go test files with table-driven test patterns. 757-line chaos test suite with structured fault injection. Playwright and Vitest end-to-end tests for web interfaces. Integration tests spanning Go-Rust FFI boundaries. Test output parsed by CI for structured reporting.

**Improvement Notes:** Add property-based testing with `rapid` (Go) or `proptest` (Rust). Increase E2E test coverage for edge cases. Port chaos tests to a CI-blocking gate (currently advisory).

---

### 4. Linting — 10/10

**Evidence:** 18+ linters configured via `trunk.io` (golangci-lint, clippy, rustfmt, shellcheck, hadolint, yamllint, eslint, prettier, etc.). Lefthook git hooks enforce lint on pre-commit. All linters run in CI with unified status reporting. Linter configurations are versioned and reviewed.

**Improvement Notes:** Add custom linter rules for hexagonal architecture boundary enforcement. Consider adding `typos` for prose and identifier spell-checking.

---

### 5. Security — 9/10

**Evidence:** `SECURITY.md` with vulnerability disclosure policy. 8+ security scanning tools integrated (TruffleHog, Gitleaks, CodeQL, cargo-deny, govulncheck, Trivy, etc.). `unsafe_code = "forbid"` in Rust crates eliminates undefined behavior risks. SBOMs generated for all releases. Comprehensive security policy document.

**Improvement Notes:** Add SAST scanning for Go code beyond `govulncheck`. Implement security scorecard (OpenSSF Scorecard) tracking. Add secret scanning for CI environment variables.

---

### 6. Documentation — 7/10

**Evidence:** `SPEC.md` is comprehensive at 1,841 lines covering architecture, APIs, and design decisions. `CONTRIBUTING.md` provides clear contributor guidelines. However, `README.md` is only 8 lines — insufficient for new users or external consumers.

**Improvement Notes:** Expand README to include quickstart, architecture overview, and badges. Add `docs/adr/` directory for architecture decision records. Generate API reference docs from code comments.

---

### 7. Type Safety — 9/10

**Evidence:** Go interfaces enforce compile-time type contracts. Rust traits with `thiserror` provide typed error handling across the FFI boundary. JSON Schema validation at API boundaries. Serde (Rust) and `json` (Go) with strict deserialization modes.

**Improvement Notes:** Add `schemars` generation for all Rust public types. Implement API contract testing between Go and Rust services.

---

### 8. Accessibility — 3/10

**Evidence:** `docs/i18n-a11y.md` document exists outlining accessibility goals but no tooling or tests implement them. No axe-core, pa11y, or Lighthouse accessibility audits in CI. No ARIA attributes tested in web interfaces.

**Improvement Notes:** Add axe-core to Playwright E2E tests. Implement Lighthouse CI for web interface accessibility scoring. Add keyboard navigation tests. Define WCAG 2.1 AA compliance targets.

---

### 9. Internationalization (i18n) — 5/10

**Evidence:** 6 locale files exist for user-facing strings. Rust crates use `i18n` runtime for locale resolution. However, Go codebase is entirely English-only with no i18n infrastructure. Mixed i18n maturity across the polyglot codebase.

**Improvement Notes:** Port Rust i18n patterns to Go using `go-i18n` or `x/text`. Create a shared locale file format. Add i18n coverage metrics to CI. Prioritize translating user-facing error messages in Go.

---

### 10. Observability — 10/10

**Evidence:** Full OpenTelemetry stack deployed: OTel SDK in both Go and Rust, OTel Collector for aggregation, Prometheus for metrics, Grafana for visualization (5 pre-built dashboards), Jaeger for distributed tracing, Loki for log aggregation. Custom collector component for VM-specific telemetry. Exemplars link metrics to traces.

**Improvement Notes:** Add alerting rules to Grafana dashboards. Implement SLO-based alerting. Add trace sampling configuration for production.

---

### 11. Chaos Engineering — 10/10

**Evidence:** 757-line structured chaos test suite with 7+ fault types (network partition, disk full, CPU saturation, memory pressure, process kill, clock skew, I/O latency). CI-integrated with dedicated workflow. Results published as test artifacts. Fault injection targets both Go and Rust components.

**Improvement Notes:** Add chaos experiments for cross-language FFI failure modes. Implement steady-state hypothesis validation. Add chaos blast radius controls for safety.

---

### 12. SLO/SLI — 6/10

**Evidence:** Performance targets defined in `PLAN.md`. `slo-monitor.yml` workflow tracks SLO compliance. However, SLI definitions are informal and not tied to specific metric queries. No error budget policy or automated enforcement.

**Improvement Notes:** Formalize SLI definitions with PromQL queries. Implement error budget burn-rate alerting. Create an SLO dashboard in Grafana. Define error budget policies with feature freeze triggers.

---

### 13. Infrastructure as Code (IaC) — 8/10

**Evidence:** Multi-region Terraform configurations for cloud deployments. TFLint for Terraform linting, TFSec for security scanning, Infracost for cost estimation. State management with remote backends. Module structure supports reusable infrastructure components.

**Improvement Notes:** Add Terratest integration tests for Terraform modules. Pin all provider and module versions. Implement drift detection in CI. Add cost budget alerts.

---

### 14. Containerization — 6/10

**Evidence:** Dockerfile exists for development builds with appropriate build caching. No production-optimized container image. No multi-stage production build. No container scanning in CI pipeline.

**Improvement Notes:** Create multi-stage production Dockerfile with distroless base. Add container image scanning (Trivy/Grype). Implement health checks. Publish to container registry with signed images. Add container resource limits.

---

### 15. Database — 7/10

**Evidence:** Pure-Go SQLite implementation with `CGO_ENABLED=0` for cross-compilation. No ORM — direct SQL queries for performance. Schema managed via migration files. However, no migration testing or rollback verification.

**Improvement Notes:** Add migration forward/backward testing. Implement connection pooling metrics. Add query performance monitoring. Document the schema evolution policy.

---

### 16. API Design — 8/10

**Evidence:** REST API with 11 well-defined endpoints following resource-oriented design. JSON-RPC interface for programmatic access. gRPC service definitions for high-performance inter-service communication. Consistent error response format across all protocols.

**Improvement Notes:** Publish OpenAPI specification. Add API versioning strategy. Implement request/response schema validation middleware. Add API changelog.

---

### 17. Error Handling — 9/10

**Evidence:** Go sentinel errors with `errors.Is()`/`errors.As()` for structured error matching. Error wrapping with context at every boundary. 29 `unwrap` calls found — all in test code only (production code uses explicit error checking). `thiserror` in Rust crates for typed error enums.

**Improvement Notes:** Add error metrics (error rate by type, error latency). Implement error budget tracking. Add structured error context propagation across services.

---

### 18. Dependency Management — 10/10

**Evidence:** Dual automation with both Renovate and Dependabot for cross-ecosystem dependency updates. `cargo-deny` for Rust crate license and vulnerability scanning. `govulncheck` for Go dependency vulnerabilities. Lock files committed for reproducibility. Auto-merge configured for patch updates.

**Improvement Notes:** Add dependency license allowlist documentation. Implement dependency review in PR workflow. Add supply chain security attestations.

---

### 19. Code Coverage — 8/10

**Evidence:** Codecov integration with targets of 70% (Go) and 80% (Rust/TypeScript). Coverage reports generated for all three languages. PR annotations show coverage delta. Coverage trends tracked over time.

**Improvement Notes:** Add branch coverage metrics. Increase Rust coverage target to 85%. Add coverage gate to block merges below threshold. Track coverage by module/directory.

---

### 20. Performance — 5/10

**Evidence:** Criterion benchmark crate exists in Rust workspace but is a TODO stub with no actual benchmark implementations. Go benchmarks exist in some packages but not systematically run in CI. No performance regression detection.

**Improvement Notes:** Implement Criterion benchmarks for critical Rust codepaths. Add Go benchmark CI workflow. Create performance regression gates. Track benchmark results over time. Add memory profiling.

---

### 21. Monitoring — 10/10

**Evidence:** Prometheus for metrics collection and alerting. Grafana with 5 pre-built dashboards for real-time visualization. Loki for log aggregation and search. Jaeger for distributed tracing. Custom telemetry collector for VM-specific metrics. Full observability stack running in development and production.

**Improvement Notes:** Add runbook links to Grafana alerts. Implement synthetic monitoring. Add capacity planning dashboards. Track monitoring coverage metrics.

---

### 22. Code Review — 8/10

**Evidence:** `CODEOWNERS` file assigns domain experts to specific paths. Quality gates enforce CI checks before merge. Mergify automates merge queue management. PR template ensures structured descriptions.

**Improvement Notes:** Add automated code review bots for dependency and security changes. Implement review assignment rotation. Add review depth metrics.

---

### 23. Branch Protection — 9/10

**Evidence:** Required status checks enforce CI gates before merge. Stale review dismissal prevents approval of outdated changes. Signed commits required. Branch protection applied to default and release branches.

**Improvement Notes:** Add `CODEOWNERS` enforcement as required reviewer. Enforce linear history on release branches. Add protection against force pushes on all long-lived branches.

---

### 24. Release Management — 9/10

**Evidence:** release-please automates version bumps and changelog generation. git-cliff provides conventional commit parsing. GoReleaser builds and publishes Go binaries. SBOMs generated for all release artifacts. Semantic versioning enforced.

**Improvement Notes:** Add release verification smoke tests. Implement canary release stage. Add release approval workflow for major versions. Publish release metrics (time to release, release frequency).

---

### 25. Dependency Injection — 10/10

**Evidence:** Full hexagonal architecture with 7 port interfaces defining all infrastructure contracts. Plugin system allows runtime adapter selection. Figment configuration provider for dependency wiring. All dependencies injected via constructors — zero global state. Comprehensive mock implementations for testing.

**Improvement Notes:** Document the DI registration patterns in an ADR. Add compile-time verification that all ports have production adapters. Create a DI container health check.

---

### 26. Logging — 7/10

**Evidence:** Go `slog` structured logging with leveled output. Audit logging for security-relevant operations. Log levels configurable via environment. However, Rust crates use `println!`/`eprintln!` instead of structured tracing.

**Improvement Notes:** Replace Rust `println!`/`eprintln!` with `tracing` crate. Add log correlation IDs across Go-Rust boundary. Implement log redaction for sensitive data. Add structured logging metrics.

---

### 27. Caching — 0/10 (N/A)

**Evidence:** Not applicable. The nanovms repository implements a VM runtime and does not involve traditional application-level caching patterns. The VM itself manages memory at a lower level.

**Improvement Notes:** N/A — this pillar is not relevant to the nanovms architecture. Could be repurposed for VM-level cache/memory management metrics if applicable.

---

### 28. Rate Limiting — 8/10

**Evidence:** Token bucket middleware for API rate limiting with configurable limits. Network-level `RateLimiter` for outbound request throttling. Rate limit headers returned to clients (X-RateLimit-*). Both per-IP and per-user rate limiting supported.

**Improvement Notes:** Add rate limit metrics to monitoring dashboards. Implement adaptive rate limiting based on server load. Add rate limit bypass for health check endpoints. Document rate limit configuration.

---

### 29. Authentication/Authorization — 8/10

**Evidence:** Bearer token authentication for API access. Constant-time token comparison prevents timing attacks. JWT support for stateless authentication. Token refresh and rotation supported. Credentials stored securely.

**Improvement Notes:** Add OAuth 2.0 support for third-party integrations. Implement RBAC for multi-tenant access. Add API key management UI. Implement audit logging for all auth events.

---

### 30. Config Management — 9/10

**Evidence:** Multi-format configuration support (TOML, YAML, JSON). JSON Schema validation for config files. Figment library for layered configuration (defaults -> file -> env -> CLI). Config file watching for live reload. Sensible defaults for zero-config startup.

**Improvement Notes:** Add config migration tooling for breaking changes. Publish JSON Schema for editor integration. Add config diff debugging tool. Document configuration precedence rules.

---

### 31. Disaster Recovery — 5/10

**Evidence:** Git-based version control provides inherent backup of source code. However, no formal disaster recovery plan exists. No documented RTO/RPO targets. No backup procedures for databases or stateful data. No DR testing or tabletop exercises.

**Improvement Notes:** Write a formal DR runbook. Define RTO/RPO targets for all stateful components. Implement automated backup scheduling. Schedule quarterly DR tabletop exercises. Add DR verification tests.

---

## Priority-Ranked Action Table

| Priority | Pillar | Score | Gap | Action | Effort | Impact |
|---|---|---|---|---|---|---|
| **P0** | Disaster Recovery | 5 | 5 | Write DR runbook, define RTO/RPO, implement backups | M | High |
| **P0** | Performance | 5 | 5 | Implement Criterion benchmarks, Go benchmark CI | M | High |
| **P1** | Accessibility | 3 | 7 | Add axe-core to Playwright, Lighthouse CI | M | High |
| **P1** | Containerization | 6 | 4 | Production Dockerfile, Trivy scanning, health checks | M | Medium |
| **P1** | Database | 7 | 3 | Migration testing, rollback verification | S | Medium |
| **P1** | i18n | 5 | 5 | Port i18n to Go, shared locale format | M | Medium |
| **P2** | SLO/SLI | 6 | 4 | Formalize SLIs with PromQL, error budget alerting | M | Medium |
| **P2** | Documentation | 7 | 3 | Expand README, add ADRs, generate API docs | S | Medium |
| **P2** | Logging | 7 | 3 | Replace Rust println! with tracing, add correlation IDs | S | Medium |
| **P3** | Testing | 8 | 2 | Property-based tests, chaos gate blocking | M | Medium |
| **P3** | IaC | 8 | 2 | Terratest, pin providers, drift detection | S | Medium |
| **P3** | API Design | 8 | 2 | Publish OpenAPI spec, add versioning | S | Medium |
| **P3** | Code Review | 8 | 2 | Auto-review bots, review rotation | S | Low |
| **P3** | Code Coverage | 8 | 2 | Branch coverage, increase Rust target | S | Low |
| **P3** | Rate Limiting | 8 | 2 | Adaptive limits, metrics dashboard | S | Low |
| **P3** | Auth | 8 | 2 | OAuth 2.0, RBAC, audit logging | M | Medium |
| **P4** | Project Structure | 9 | 1 | Cross-language build tool, port registry | M | Low |
| **P4** | Security | 9 | 1 | Go SAST, Scorecard tracking | S | Low |
| **P4** | Type Safety | 9 | 1 | schemars generation, contract testing | S | Low |
| **P4** | Error Handling | 9 | 1 | Error metrics, budget tracking | S | Medium |
| **P4** | Branch Protection | 9 | 1 | CODEOWNERS enforcement, linear history | S | Low |
| **P4** | Release Management | 9 | 1 | Canary stage, release verification | S | Low |
| **P4** | Dependency Injection | 10 | 0 | Document DI patterns, health check | S | Low |
| **P4** | Config Management | 9 | 1 | Migration tooling, schema publishing | S | Low |

---

*Generated on 2026-08-20 by Forge Code 31-Pillar Scorecard Engine v1.0*

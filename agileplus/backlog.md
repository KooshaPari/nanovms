# Backlog

Backlog is ordered by **pillar weakness × business impact**. Each item carries
a `pillar:` label matching `agileplus/pillars/31-pillar-scorecard.json`.

## Priority key

- **P0** — must be done this quarter; pillar is red (score 0–3)
- **P1** — next sprint; pillar is yellow (score 4–5)
- **P2** — within two sprints; pillar is yellow (score 4–6)
- **P3** — opportunistic, no pillar SLA

## P0 — red pillars (score 0–3)

| ID    | Pillar        | Score | Title                                                            | Sprint |
|-------|---------------|-------|------------------------------------------------------------------|--------|
| B-01  | i18n          | 2     | Audit `locales/*.json` coverage; add missing locale fallbacks    | S2     |
| B-02  | Agile PM      | 3     | Bootstrap AgilePlus scaffolding (this sprint)                    | S1     |
| B-03  | Accessibility | 3     | Add axe-core scan in CI for VitePress docs; fix top 5 violations | S2     |
| B-04  | Mobile        | 3     | Decide mobile parity scope (none / wrapper / first-class)        | S3     |
| B-05  | CoC           | 3     | Add CoC violation reporting email + link from `CONTRIBUTING.md`  | S2     |

## P1 — yellow pillars (score 4–5)

| ID    | Pillar         | Score | Title                                                            | Sprint |
|-------|----------------|-------|------------------------------------------------------------------|--------|
| B-06  | Branch Mgmt    | 4     | Adopt trunk-based dev with short-lived feature branches          | S2     |
| B-07  | Tests          | 4     | Push coverage to 60 % floor on `internal/` and `crates/`         | S2     |
| B-08  | DB             | 4     | Document storage adapter contract; add SQLite-backed fixture     | S3     |
| B-09  | Releases       | 4     | Cut `v0.1.0` with goreleaser; signed attestations                | S1     |

## P2 — yellow pillars (score 5–6)

| ID    | Pillar         | Score | Title                                                            | Sprint |
|-------|----------------|-------|------------------------------------------------------------------|--------|
| B-10  | Reviews        | 5     | Add `pillar:` owners to all 31 pillars via `CODEOWNERS-pillars`  | S1     |

## P3 — opportunistic

- Enable Renovate minor-and-patch auto-merge (`deps`).
- Add `cargo-nextest` to the test gate (`tests`).
- Adopt `cargo deny` license allow-list audit in CI (`security`).
- Add `mizu` snapshot tests on JSON schemas (`api`).
- Promote existing `locales/en.json` into a typed `i18n` Rust crate (`i18n`).

## How items get added

1. Open an issue with the `pillar:` label and a one-paragraph description.
2. Triage assigns P0–P3 and a target sprint (or "backlog" if undecided).
3. A line is added to this file in the matching section.
4. At sprint planning, items move from `Backlog` to `sprint-current.md`.
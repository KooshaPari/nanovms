# Sprint 1 — 2026-08-19 → 2026-08-26

> First sprint under AgilePlus. Goal: stand up the framework itself and
> surface the lowest-scoring pillars so the next sprint can target them.

## Sprint goal

Bootstrap the AgilePlus scaffolding end-to-end (this directory, the weekly
pillar-scorecard workflow, the docs one-pager, the changelog) and lock in a
concrete plan to lift the five pillars currently in the red band.

## Capacity

- 1 active engineer (@KooshaPari), 0.6 FTE this sprint.

## Committed (P0 / P1)

| ID     | Pillar    | Title                                                        | Owner       | State |
|--------|-----------|--------------------------------------------------------------|-------------|-------|
| S1-01  | Agile PM  | Bootstrap `agileplus/` directory with all standard files     | @KooshaPari | done  |
| S1-02  | Agile PM  | Wire weekly pillar-scorecard GitHub Actions workflow         | @KooshaPari | done  |
| S1-03  | Agile PM  | Publish `docs/AGILEPLUS-SETUP.md` one-pager                  | @KooshaPari | done  |
| S1-04  | CoC       | Confirm `CODE_OF_CONDUCT.md` is Contributor Covenant v2.1    | @KooshaPari | done  |
| S1-05  | Releases  | Author `CHANGELOG.md` `v0.1.0` entry summarising merged work | @KooshaPari | done  |
| S1-06  | Releases  | Cut `v0.1.0` git tag and attach signed attestation           | @KooshaPari | todo  |

## Done

- `agileplus/AGILEPLUS.md`, `sprint-current.md`, `sprint-retrospective-template.md`
- `agileplus/backlog.md` (top 10 items, P0–P2)
- `agileplus/pillars/31-pillar-scorecard.json` (current state, 5.3 grade)
- `agileplus/quality-gates.yml`, `velocity.md`, `CODEOWNERS-pillars`
- `.github/workflows/agileplus-pillar-scorecard.yml` (weekly, Mon 06:00 UTC)
- `docs/AGILEPLUS-SETUP.md`
- `CHANGELOG.md` updated with `v0.1.0` release notes

## Risks / blockers

- **Single-engineer sprint.** Anything that slips takes one item with it.
- **Scorecard workflow needs an audit binary.** Until
  `vars.AGILEPLUS_AUDIT_BIN` is wired to a real audit binary, the workflow
  emits a warning and skips re-scoring. The JSON is the source of truth for
  now and is updated manually via PRs.
- **i18n (score 2)** is the weakest pillar and is *not* addressed this
  sprint — it's on top of Sprint 2.

## Definition of done

- [x] All committed items merged to `main`.
- [ ] Sprint retro written to `agileplus/sprint-1-retro.md`.
- [ ] Row appended to `agileplus/velocity.md` with the final grade.
- [ ] `agileplus/pillars/31-pillar-scorecard.json` re-scored by the workflow
      or by manual PR.

## Carry-over candidates for Sprint 2

| Pillar      | Score | Carry-over candidate                              |
|-------------|-------|---------------------------------------------------|
| i18n        | 2     | Audit `locales/*.json` coverage (B-01)            |
| Accessibility | 3   | axe-core scan in docs CI (B-03)                   |
| Mobile      | 3     | Decide mobile parity scope (B-04)                 |
| Branch Mgmt | 4     | Trunk-based dev with short-lived branches (B-06)  |
| Tests       | 4     | Push coverage to 60 % floor on `internal/` (B-09) |
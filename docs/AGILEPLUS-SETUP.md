---
title: AgilePlus Setup
---

# AgilePlus Setup

One-page explainer for the AgilePlus framework used in this repository.

## What is AgilePlus?

AgilePlus is a **file-based** project-management framework that pairs
**1-week sprints** with a **31-pillar engineering quality scorecard**.
Everything lives in `agileplus/` at the repo root and is version-controlled
alongside the code it tracks.

## Files

| File | Purpose |
|------|---------|
| `agileplus/AGILEPLUS.md` | Master configuration |
| `agileplus/sprint-current.md` | Active sprint goals |
| `agileplus/sprint-retrospective-template.md` | Retro template |
| `agileplus/backlog.md` | Prioritized backlog (P0-P3) |
| `agileplus/pillars/31-pillar-scorecard.json` | Current pillar scores |
| `agileplus/quality-gates.yml` | CI gate definitions |
| `agileplus/velocity.md` | Sprint velocity history |
| `agileplus/CODEOWNERS-pillars` | Pillar-to-owner mapping |

## Sprint flow

1. **Plan** (last day of prior sprint). Pull items from `backlog.md`, copy the
   goals into `sprint-current.md`.
2. **Build** (the sprint). Commit messages reference the sprint item
   (`(#S<n>-<id>)`) and PRs carry a `pillar:` label.
3. **Demo + retro** (final day). Copy
   `sprint-retrospective-template.md` to `sprint-<n>-retro.md`, fill it in,
   append a row to `velocity.md`.
4. **Recompute** (weekly, Mon 06:00 UTC). The
   `agileplus-pillar-scorecard` workflow re-scores all 31 pillars and posts a
   summary.

## Pillar scorecard

Each pillar is scored 0-10 and bucketed:

| Band   | Range | Meaning                          |
|--------|-------|----------------------------------|
| red    | 0-3   | Pillar failing; address this Q    |
| yellow | 4-6   | Pillar under target; next sprint  |
| green  | 7-8   | On-track                         |
| gold   | 9-10  | Gold standard                    |

The aggregate grade is the rounded average. **Current state: average 5.32,
grade 5.3 (D)**. Full per-pillar breakdown lives in
`agileplus/pillars/31-pillar-scorecard.json`.

## Quality gates

Defined in `agileplus/quality-gates.yml` and enforced by GitHub branch
protection: `lint`, `test`, `security`, `build` must all be green before a PR
can merge to `main`. `docs-build` and `coverage` are tracked but not blocking.

## Where to start

- **Joining a sprint?** Read `sprint-current.md` first.
- **Have a new idea?** Open an issue with the `pillar:` label and add a row
  to `backlog.md`.
- **Disagreement on a pillar score?** Open a PR that edits the JSON and tag
  `@KooshaPari`.
- **Need the weekly audit?** Trigger the `agileplus-pillar-scorecard`
  workflow manually via `workflow_dispatch`.

## Conventions

- One-line PR titles; body references the sprint item, the pillar, and links
  the issue.
- Commit messages follow Conventional Commits (`feat:`, `fix:`, `chore:`,
  `docs:`, `refactor:`, `test:`, `build:`, `ci:`).
- Backlog items get `pillar:<id>` labels that match
  `agileplus/pillars/31-pillar-scorecard.json`.
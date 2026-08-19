# AgilePlus Configuration

Master configuration for the **AgilePlus** project-management framework on the
`nanovms` unikernel repository. AgilePlus is intentionally lightweight: every
artefact is plain text in `agileplus/` and version-controlled alongside code.

## 1. Cadence

| Field           | Value                                                    |
|-----------------|----------------------------------------------------------|
| Sprint length   | **1 week** (Monday → Monday)                             |
| Sprint ID       | `S<n>` (`S1`, `S2`, ...) starting at `S1` on 2026-08-19  |
| Item ID         | `S<n>-<seq>` (e.g. `S1-03`)                              |
| Planning        | Final day of prior sprint                                |
| Retro + demo    | Final day of sprint                                      |
| Branch model    | Trunk-based; short-lived feature branches off `main`     |
| Release cadence | Ad-hoc, gated by `quality-gates.yml` + signed attestations|

Sprint IDs appear in commit messages (`(#S1-03)`) and PR titles to make backlog
provenance traceable from `git log`.

## 2. Roles

| Role             | Default owner | Responsibility                                |
|------------------|---------------|-----------------------------------------------|
| Scrum master     | rotating      | Runs ceremonies, removes blockers             |
| Tech lead        | @KooshaPari   | Architecture, pillar ownership, releases      |
| Reviewer pool    | CODEOWNERS    | PR review per pillar                          |
| Release manager  | @KooshaPari   | Tags, signing, attestations                   |

## 3. Quality gates

Defined in [`quality-gates.yml`](quality-gates.yml). Four gates are required for
merge to `main`:

1. `lint`   — Go (`golangci-lint`), Rust (`cargo clippy -D warnings`),
   YAML (`yamllint`), Markdown (`markdownlint`), dprint check.
2. `test`   — `go test ./... -race`, `cargo test --workspace`.
3. `security` — `cargo deny`, `trivy fs`, `gitleaks detect`.
4. `build`  — `cargo build --release`, `go build ./cmd/nanovms`.

Two gates are tracked but not blocking: `docs-build`, `coverage`.

## 4. 31-pillar scorecard

Defined in [`pillars/31-pillar-scorecard.json`](pillars/31-pillar-scorecard.json).
Each pillar is scored 0–10 and bucketed:

| Band    | Range | Meaning                          |
|---------|-------|----------------------------------|
| red     | 0–3   | Pillar failing; address this Q    |
| yellow  | 4–6   | Pillar under target; next sprint  |
| green   | 7–8   | On-track                         |
| gold    | 9–10  | Gold standard                    |

The aggregate grade is the rounded average:

| Avg     | Grade |
|---------|-------|
| ≥ 9.0   | A+    |
| ≥ 8.0   | A     |
| ≥ 7.0   | B     |
| ≥ 6.0   | C     |
| ≥ 5.0   | D     |
| < 5.0   | F     |

**Current state**: average **5.32**, grade **5.3 (D)**. See the scorecard JSON
for the per-pillar breakdown and the top/weak pillars.

The scorecard is re-computed weekly by
[`.github/workflows/agileplus-pillar-scorecard.yml`](../.github/workflows/agileplus-pillar-scorecard.yml).

## 5. Process

1. New work enters [`backlog.md`](backlog.md) with the affected `pillar:` and
   priority (`P0`–`P3`).
2. Sprint planning pulls the highest-priority items, biased toward weak
   pillars.
3. PRs reference their sprint item (`(#S<n>-<id>)`) and carry a `pillar:`
   label.
4. At sprint close: copy
   [`sprint-retrospective-template.md`](sprint-retrospective-template.md) →
   `sprint-<n>-retro.md`, append a row to [`velocity.md`](velocity.md), update
   the scorecard if a pillar moved.
5. Weekly: the pillar-scorecard workflow runs and posts a summary to the PR or
   the workflow run summary.

## 6. File map

| Path                                                | Purpose                |
|-----------------------------------------------------|------------------------|
| `agileplus/AGILEPLUS.md`                            | This file              |
| `agileplus/sprint-current.md`                       | Active sprint goals    |
| `agileplus/sprint-retrospective-template.md`        | Retro template         |
| `agileplus/backlog.md`                              | Prioritized backlog    |
| `agileplus/pillars/31-pillar-scorecard.json`        | Pillar scores          |
| `agileplus/quality-gates.yml`                       | CI gate definitions    |
| `agileplus/velocity.md`                             | Velocity history       |
| `agileplus/CODEOWNERS-pillars`                      | Pillar → owner mapping |
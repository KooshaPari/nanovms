# Velocity

Velocity is updated by appending one row per closed sprint. "Points" map to
backlog items (P0 = 3 pts, P1 = 2 pts, P2 = 1 pt, P3 = 0.5 pt) so the running
average converges on a stable per-sprint capacity number.

| Sprint | Window                     | Committed | Completed | Carry-over | Avg pillar | Grade |
|--------|----------------------------|-----------|-----------|------------|------------|-------|
| 1      | 2026-08-19 → 2026-08-26     | 6 (15 pt) | 5 (in-progress) | 1   | 5.32       | D     |

> Sprint 1 is in progress as of `2026-08-19`. The row is finalised at the
> retro on `2026-08-26`.

## Chart (ASCII)

```
Points completed per sprint (target line at running avg)
────┬────────┬────────┬────────┬────────┬────────
 S1 │■■■■■░░ │        │        │        │
────┴────────┴────────┴────────┴────────┴────────
   0        5       10       15       20
```

## Capacity notes

- S1 had a single engineer at 0.6 FTE. Future sprints assume 0.6 FTE until
  another maintainer joins.
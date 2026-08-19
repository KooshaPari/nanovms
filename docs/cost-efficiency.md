# nanovms Cost Efficiency

Per-adapter + per-tenant cost tracking.

## Cost dimensions

| Dimension | Metric | Unit |
|-----------|--------|------|
| Compute | vCPU-second | $/vCPU-s |
| Memory | GB-second | $/GB-s |
| Storage | GB-month | $/GB-month |
| Network egress | GB | $/GB |

## Cost report

```go
type CostReport struct {
    SandboxID        string
    VCpUPerSecond    float64
    MemoryGBPerSec   float64
    StorageGB        float64
    NetworkEgressGB  float64
    EstimatedUSD     float64
}

type CostTracker interface {
    Record(ctx context.Context, e CostEvent) error
    Report(ctx context.Context, period Period) (CostReport, error)
}
```

## Real-time tracking

```go
tracker := NewPostgresCostTracker(db)
tracker.Record(ctx, CostEvent{
    Type:      EventSandboxStarted,
    SandboxID: sb.ID,
    VCpU:      2,
    MemoryGB:  1.0,
})
tracker.Record(ctx, CostEvent{
    Type:  EventNetworkEgress,
    Bytes: 1024,
})
```

## Per-tenant billing

```go
report, _ := tracker.Report(ctx, PeriodMonth(2026, 7)).ByTenant("user-123")
// CostReport{VCpUPerSecond: 86400, MemoryGBPerSec: 43200, EstimatedUSD: 12.34}
```

## Budget enforcement

```go
tracker.SetBudget(ctx, "user-123", 100.00)
if err := tracker.TryAcquire(ctx, "user-123", CostEventSandboxStarted{...}); err != nil {
    if errors.Is(err, ErrBudgetExceeded) {
        // reject
    }
}
```

## Optimization targets

- P50 cost per sandbox: < $0.10/hour
- P99 cost per sandbox: < $1.00/hour
- Idle cost (stopped but not reclaimed): < $0.01/hour
- Cost per GB-second: < $0.0001

## CI integration

- Cost report on every PR
- Alert on >20% cost regression
- Weekly cost reports emailed to admins

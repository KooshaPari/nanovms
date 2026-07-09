# nanovms Monitoring

OTel-based observability pipeline.

## Pipeline

```
   ┌─────────────┐      ┌─────────┐      ┌──────────────┐
   │ nanovms     │ ───> │ OTel    │ ───> │ Grafana Cloud│
   │ daemon+cli  │      │ Collector│      │ + Prometheus│
   │             │      │ (OTLP)   │      │ + Tempo      │
   └─────────────┘      └─────────┘      └──────────────┘
```

- **Metrics**: Prometheus exporter on port 9090
- **Traces**: OTLP to Tempo, sampled 10% in prod
- **Logs**: structured slog to Loki via Promtail

## SLOs

| Service | SLO | Target | Error budget |
|---------|-----|--------|--------------|
| daemon serve | p99 latency | < 50ms | 99.95% monthly |
| sandbox create | p99 latency | < 500ms | 99.9% monthly |
| sandbox exec | p99 latency | < 1s | 99.5% monthly |
| secret store | p99 latency | < 100ms | 99.9% monthly |

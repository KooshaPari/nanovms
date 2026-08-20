# Multi-Region Deployment

This document describes the multi-region deployment architecture for nanovms,
covering region selection, data replication, failover, latency considerations,
and compliance requirements.

## Architecture Overview

nanovms uses an active-active multi-region deployment model to provide
low-latency access to users worldwide while maintaining high availability.

```
                     +------------------------------+
                     |        Global DNS / GSLB      |
                     |    (latency-based routing)    |
                     +----------+----------+---------+
                                |          |
                +---------------+          +----------------+
                |                                             |
    +-----------+-----------+            +--------------------+-----------+
    |    us-east-1         |            |       eu-west-1               |
    |  +---------------+   |   <sync>   |  +------------------------+   |
    |  | API Gateway   |   |            |  | API Gateway            |   |
    |  | Orchestrator  |   |            |  | Orchestrator           |   |
    |  | Sandbox Pool  |   |            |  | Sandbox Pool           |   |
    |  | D1 (primary)  |   |            |  | D1 (replica)           |   |
    |  | KV Store      |   |            |  | KV Store               |   |
    |  | R2 Bucket     |   |            |  | R2 Bucket              |   |
    |  +---------------+   |            |  +------------------------+   |
    +-----------------------+            +-------------------------------+
```

### Key Design Principles

1. **Active-Active**: All regions can serve read and write traffic.
2. **Eventual Consistency**: Non-critical data uses eventual consistency
   with conflict resolution via last-writer-wins.
3. **Strong Consistency for Critical Paths**: Sandbox lifecycle operations
   are strongly consistent within a single region.
4. **Graceful Degradation**: A region failure degrades functionality
   rather than failing completely.

## Region Selection Strategy

### DNS-Based Routing

Global load balancing uses latency-based DNS routing:

| Region        | Code           | Deployment Target         |
|---------------|----------------|---------------------------|
| US East       | `us-east-1`    | North/South America       |
| US West       | `us-west-2`    | West Coast Americas       |
| EU West       | `eu-west-1`    | Europe, Africa            |
| EU Central    | `eu-central-1` | Central/Eastern Europe    |
| AP Southeast  | `ap-southeast-1`| Southeast Asia, Oceania  |
| AP Northeast  | `ap-northeast-1`| East Asia                |

### Region Selection Algorithm

```go
func SelectRegion(clientIP string, availableRegions []Region) Region {
    // 1. Resolve client geolocation from IP
    geo := geolocate(clientIP)

    // 2. Rank regions by latency (probes from client subnet)
    ranked := rankByLatency(geo, availableRegions)

    // 3. Filter out regions with degraded health
    healthy := filterHealthy(ranked)

    // 4. Return lowest-latency healthy region
    return healthy[0]
}
```

### Selection Criteria

| Criterion              | Weight | Description                                    |
|------------------------|--------|------------------------------------------------|
| Network latency        | 40%    | Round-trip time from client to region          |
| Region health          | 25%    | Current health status and error rates          |
| Capacity               | 20%    | Available sandbox capacity in the region       |
| Data locality          | 15%    | Whether user data is replicated to the region  |

## Data Replication

### Replication Tiers

| Tier | Data Type              | Strategy           | RPO       | RTO     |
|------|------------------------|--------------------|-----------|---------|
| 1    | Sandbox state          | Async (per-second) | 1s        | 30s     |
| 2    | User configs           | Async (near-real-time) | 5s   | 1m      |
| 3    | Audit logs             | Async (batched)    | 5m        | 10m     |
| 4    | Telemetry / metrics    | Local-only + export | N/A      | N/A     |

### Replication Flow

1. **Write Path**: Client writes to the nearest region.
2. **Local Commit**: Region acknowledges write immediately after local commit.
3. **Async Replication**: Change is replicated to other regions asynchronously.
4. **Conflict Resolution**: Last-writer-wins with vector clocks for ordering.

### Cross-Region Sync

```go
type ReplicationConfig struct {
    Mode            ReplicationMode `json:"mode"`            // "sync", "async", "bidirectional"
    Interval        time.Duration   `json:"interval"`        // Replication frequency
    ConflictPolicy  string          `json:"conflict_policy"` // "last-writer-wins", "source-wins"
    RetryAttempts   int             `json:"retry_attempts"`
    RetryBackoff    time.Duration   `json:"retry_backoff"`
    MaxBatchSize    int             `json:"max_batch_size"`
    CompressionEnabled bool         `json:"compression_enabled"`
}
```

## Failover Procedures

### Automatic Failover

When a region becomes unhealthy:

1. **Detection** (0-30s): Health checks fail for 3 consecutive intervals.
2. **Traffic Shift** (30-60s): DNS weight shifts to healthy regions.
3. **State Recovery** (1-5min): In-flight operations complete or timeout.
4. **Stabilization** (5-10min): New steady state with reduced capacity.

### Failover States

```
HEALTHY -> DEGRADED -> UNHEALTHY -> FAILED -> RECOVERING -> HEALTHY
```

| State        | Description                                     |
|--------------|-------------------------------------------------|
| HEALTHY      | All subsystems operational                       |
| DEGRADED     | Partial failure, reduced functionality           |
| UNHEALTHY    | Majority of subsystems failing                   |
| FAILED       | Complete region failure, traffic drained          |
| RECOVERING   | Region coming back, traffic being restored       |

### Manual Failover

```bash
# Force traffic away from a region
nanovms region drain --region=us-east-1 --grace-period=300s

# Verify failover completed
nanovms region status --region=us-east-1

# Restore traffic to recovered region
nanovms region restore --region=us-east-1 --ramp=5m
```

## Latency Considerations

### SLO Targets

| Metric              | Target    | Measurement                    |
|---------------------|-----------|--------------------------------|
| API latency (p50)   | < 50ms    | Same-region request            |
| API latency (p99)   | < 200ms   | Same-region request            |
| Cross-region latency| < 500ms   | Worst-case inter-region        |
| Sandbox boot time   | < 2s      | Time from API to sandbox ready |
| Failover RTO        | < 60s     | Time from detection to traffic shift |

### Latency Optimization

1. **Connection Pooling**: Maintain persistent connections to regional backends.
2. **Request Hedging**: Send parallel requests to the nearest two regions
   for latency-critical operations.
3. **Local Caching**: Read-heavy data cached locally with TTL-based invalidation.
4. **Edge Computing**: Static assets and non-critical API responses served
   from edge locations via Cloudflare CDN.

### Network Topology

```
Client -> Edge (CDN) -> Regional API -> Sandbox Pool
                    + (async replication)
              Other Regions
```

## Compliance Requirements

### Data Residency

| Regulation | Requirement                                          | Implementation               |
|------------|------------------------------------------------------|-------------------------------|
| GDPR       | EU user data must stay in EU                          | EU-West, EU-Central regions   |
| CCPA       | California data access requests                      | US regions with data export   |
| SOC 2      | Audit trail for all data access                      | Immutable audit logs per region|
| ISO 27001  | Information security management                      | Region-isolated security controls |

### Data Sovereignty Implementation

```go
type DataSovereigntyPolicy struct {
    AllowedRegions    []string          `json:"allowed_regions"`
    DataClassifications map[string]RegionRule `json:"data_classifications"`
    AuditRequired     bool              `json:"audit_required"`
    EncryptionAtRest  bool              `json:"encryption_at_rest"`
    EncryptionInTransit bool            `json:"encryption_in_transit"`
}

type RegionRule struct {
    AllowedRegions []string `json:"allowed_regions"`
    RequiresConsent bool   `json:"requires_consent"`
    RetentionDays  int      `json:"retention_days"`
}
```

### Encryption

- **At Rest**: All data encrypted with region-specific keys (AWS KMS / Cloudflare).
- **In Transit**: TLS 1.3 for all inter-region communication.
- **Key Management**: Keys are region-scoped; never cross region boundaries.

### Audit Logging

All cross-region operations generate audit events with:

- `source_region` / `destination_region`
- `operation_type` (read, write, replicate)
- `data_classification` (public, internal, confidential, restricted)
- `user_id` / `session_id` for traceability
- `timestamp` with nanosecond precision

## Monitoring and Alerting

### Multi-Region Health Dashboard

Monitor via the performance dashboard (`docs/dashboards/performance.html`):

- Per-region request latency (p50, p95, p99)
- Cross-region replication lag
- Error rates by region
- Capacity utilization per region
- Failover events and recovery status

### Alert Thresholds

| Alert                     | Condition                   | Severity | Window |
|---------------------------|-----------------------------|----------|--------|
| High error rate           | > 5% for 5 minutes         | Critical | 5m     |
| Replication lag           | > 30s for 2 minutes        | Warning  | 2m     |
| Region capacity           | > 80% for 10 minutes       | Warning  | 10m    |
| Sandbox boot latency      | > 3s p99 for 5 minutes     | Warning  | 5m     |
| Cross-region latency      | > 1s p99 for 3 minutes     | Critical | 3m     |
| Region unhealthy          | Health check failing        | Critical | 1m     |

# nanovms Observability

Audit log for compliance + debugging, append-only storage.

## Event types

```go
type AuditEvent struct {
    Type      string                 `json:"type"`
    UserID    string                 `json:"user_id"`
    Timestamp time.Time              `json:"ts"`
    IP        string                 `json:"ip,omitempty"`
    Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

const (
    EventUserAuthenticated = "user.authenticated"
    EventSecretAccessed    = "secret.accessed"
    EventSandboxCreated    = "sandbox.created"
    EventConfigChanged     = "config.changed"
    EventAdminAction       = "admin.action"
)
```

## Append-only storage

```go
type AuditLog interface {
    Append(ctx context.Context, e AuditEvent) error
    Query(ctx context.Context, filter QueryFilter) ([]AuditEvent, error)
}

// Postgres impl: WORM (no UPDATE, no DELETE)
type pgLog struct{ db *sql.DB }
```

## Rotation

- Daily: `audit_2026-07-09.log` -> `audit_2026-07-10.log`
- Compression: gzip after 7 days
- Retention: 7 years
- Cold storage: S3 (Glacier) after 90 days

## Usage

```go
log.Append(ctx, AuditEvent{
    Type: EventSandboxCreated,
    UserID: user.ID,
    IP: r.RemoteAddr,
    Metadata: map[string]interface{}{"sandbox_id": sb.ID, "image": sb.Image},
})
```

## Querying

```sql
SELECT type, count(*) FROM audit_log
WHERE created_at > NOW() - INTERVAL '7 days'
GROUP BY type;
```

## Retention policies

- Hot (Postgres): 90 days
- Warm (S3 IA): 1 year
- Cold (S3 Glacier): 7 years
- Tamper-evident: hash-chain each entry

# ADR-012: Multi-Tenancy Architecture

## Status

Proposed

## Context

NanoVMS serves multiple users/teams sharing infrastructure:
- Team workspaces with isolated VMs
- Project-level resource quotas
- Organization-wide policies
- Shared hardware, isolated tenants

## Decision

### Multi-Tenancy Model

```
┌────────────────────────────────────────────────────────────────────┐
│                         NanoVMS Multi-Tenant Architecture               │
├────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                     Organization (Tenant)                         │  │
│  │                                                                     │  │
│  │   ┌────────────────────────────────────────────────────────┐  │  │
│  │   │                     User Pool                               │  │  │
│  │   │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐  │  │  │
│  │   │  │ User A  │ │ User B  │ │ User C  │ │ User D  │  │  │  │
│  │   │  └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘  │  │  │
│  │   │       │            │            │            │         │  │  │
│  │   │       ▼            ▼            ▼            ▼         │  │  │
│  │   │  ┌───────────────────────────────────────────────┐  │  │  │
│  │   │  │              Team Pool                           │  │  │  │
│  │   │  │  ┌─────────────┐  ┌─────────────┐            │  │  │  │
│  │   │  │  │ Team Alpha  │  │ Team Beta   │            │  │  │  │
│  │   │  │  │ 5 VMs max  │  │ 10 VMs max  │            │  │  │  │
│  │   │  │  └─────────────┘  └─────────────┘            │  │  │  │
│  │   │  └───────────────────────────────────────────────┘  │  │  │
│  │   └────────────────────────────────────────────────────────┘  │  │
│  │                                                                     │  │
│  └────────────────────────────────────────────────────────────────┘  │
│                                │                                         │
│                                ▼                                         │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                     Shared Infrastructure                        │  │
│  │                                                                     │  │
│  │   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐      │  │
│  │   │  Host Node  │  │  Host Node  │  │  Host Node  │      │  │
│  │   │  CPU: 16   │  │  CPU: 16   │  │  CPU: 16   │      │  │
│  │   │  RAM: 64GB │  │  RAM: 64GB │  │  RAM: 64GB │      │  │
│  │   │  VMs: 20   │  │  VMs: 15   │  │  VMs: 18   │      │  │
│  │   └──────┬──────┘  └──────┬──────┘  └──────┬──────┘      │  │
│  │          │                 │                 │               │  │
│  │          └─────────────────┼─────────────────┘               │  │
│  │                            ▼                                 │  │
│  │               ┌──────────────────────┐                   │  │
│  │               │   Network Isolation   │                   │  │
│  │               │   (VLAN, WireGuard) │                   │  │
│  │               └──────────────────────┘                   │  │
│  │                            ▼                                 │  │
│  │               ┌──────────────────────┐                   │  │
│  │               │   Storage Pool       │                   │  │
│  │               │   (ZFS, Ceph)       │                   │  │
│  │               └──────────────────────┘                   │  │
│  │                                                                     │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                                                     │
└────────────────────────────────────────────────────────────────────┘
```

### Resource Quotas

```yaml
# quotas.yaml
quota_types:
  - name: vm_count
    description: "Maximum number of VMs per entity"
    scopes: [user, team, organization]
    default:
      user: 5
      team: 50
      organization: 500

  - name: cpu_cores
    description: "Maximum CPU cores allocated"
    scopes: [user, team, organization]
    default:
      user: 8
      team: 64
      organization: 256

  - name: memory_bytes
    description: "Maximum memory in bytes"
    scopes: [user, team, organization]
    default:
      user: 16_GB
      team: 128_GB
      organization: 512_GB

  - name: storage_bytes
    description: "Maximum storage in bytes"
    scopes: [user, team, organization]
    default:
      user: 100_GB
      team: 1_TB
      organization: 10_TB

  - name: gpu_count
    description: "Maximum number of GPUs"
    scopes: [user, team, organization]
    default:
      user: 0
      team: 1
      organization: 4

  - name: snapshot_count
    description: "Maximum number of snapshots"
    scopes: [user, team, organization]
    default:
      user: 10
      team: 50
      organization: 200
```

### Isolation Model

```go
// pkg/tenant/tenant.go
package tenant

type Tenant struct {
    ID            uuid.UUID
    Name          string
    Organization  uuid.UUID
    Team          *uuid.UUID  // nil if org-level
    Quota         Quota
    Policies      []Policy
    CreatedAt     time.Time
    UpdatedAt     time.Time
}

type Quota struct {
    VMCount       int
    CPUCores     int
    MemoryBytes  int64
    StorageBytes int64
    GPUCount     int
    Snapshots    int
}

type Policy struct {
    Name       string
    Effect     string  // "allow" or "deny"
    Actions    []string
    Resources  []string
    Conditions map[string]string
}

// CheckQuota checks if a resource allocation would exceed quota
func (t *Tenant) CheckQuota(resource string, requested int64) error {
    used, limit := t.GetUsage(resource)
    if used + requested > limit {
        return &QuotaExceededError{
            Resource: resource,
            Used:     used,
            Limit:    limit,
            Tenant:   t.ID,
        }
    }
    return nil
}

// Example policies
var DefaultPolicies = []Policy{
    {
        Name:    "deny-dangerous-ops",
        Effect:  "deny",
        Actions: []string{"vm:exec", "vm:attach-gpu"},
        Conditions: map[string]string{
            "user.role": "developer",
        },
    },
    {
        Name:    "allow-read-only-snapshots",
        Effect:  "allow",
        Actions: []string{"vm:snapshot:create"},
        Conditions: map[string]string{
            "vm.backup": "true",
        },
    },
}
```

### Network Isolation

```go
// pkg/tenant/network.go
package tenant

type NetworkIsolation struct {
    VLANID       uint16
    WireGuardKey string  // Per-tenant WireGuard key
    Subnet       string  // 10.{tenant_id}.0.0/16
    Gateway      string
    DNS          []string
}

// CreateTenantNetwork creates isolated network for tenant
func CreateTenantNetwork(ctx context.Context, tenant *Tenant) (*NetworkIsolation, error) {
    // Allocate VLAN
    vlanID, err := AllocateVLAN()
    if err != nil {
        return nil, err
    }

    // Generate WireGuard keys
    privKey, pubKey, err := GenerateWireGuardKeys()
    if err != nil {
        return nil, err
    }

    // Calculate subnet based on tenant ID
    subnet := fmt.Sprintf("10.%d.0.0/16", tenant.ID[0]%256)
    gateway := fmt.Sprintf("10.%d.0.1", tenant.ID[0]%256)

    return &NetworkIsolation{
        VLANID:       vlanID,
        WireGuardKey: pubKey,
        Subnet:       subnet,
        Gateway:      gateway,
        DNS:          []string{"10.0.0.53", "10.0.0.54"},
    }, nil
}

// VLAN configuration for switch
const vlanConfig = `
# /etc/network/interfaces.d/tenant-{tenant_id}
auto vlan{vlan_id}
iface vlan{vlan_id} inet static
    address {gateway}/24
    vlan_raw_device {physical_interface}
`
```

### Storage Isolation

```go
// pkg/tenant/storage.go
package tenant

type StoragePool struct {
    Name       string
    QuotaBytes int64
    UsedBytes  int64
    Datasets   []Dataset
}

type Dataset struct {
    Name       string  // tenant-{id}/{dataset}
    MountPoint string  // /tank/tenant-{id}/{dataset}
    QuotaBytes int64
}

// ZFS dataset per tenant
const zfsCommands = `
# Create tenant dataset with quota
zfs create -o quota={quota_gb}G \
           -o recordsize=128K \
           -o compression=lz4 \
           tank/tenant-{tenant_id}

# Create sub-datasets
zfs create tank/tenant-{tenant_id}/vms
zfs create tank/tenant-{tenant_id}/snapshots
zfs create tank/tenant-{tenant_id}/backups

# Set reservation to prevent overcommit
zfs set reservation={reservation_gb}G tank/tenant-{tenant_id}

# Enable atime for performance
zfs set atime=on tank/tenant-{tenant_id}
`
```

### API Authentication

```go
// pkg/tenant/auth.go
package tenant

type AuthConfig struct {
    TokenIssuer    string
    TokenAudience string
    JWKSURL       string
}

// TokenClaims represents JWT claims
type TokenClaims struct {
    jwt.RegisteredClaims
    TenantID    uuid.UUID  `json:"tenant_id"`
    TeamID      *uuid.UUID `json:"team_id,omitempty"`
    UserID      uuid.UUID  `json:"user_id"`
    Roles       []string   `json:"roles"`
    Scopes      []string   `json:"scopes"`
    QuotaLimits Quota      `json:"quota_limits"`
}

// Middleware extracts tenant from JWT
func TenantMiddleware(cfg *AuthConfig) gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        if token == "" {
            c.AbortWithStatusJSON(401, Error{Message: "no token"})
            return
        }

        claims, err := ValidateToken(token, cfg)
        if err != nil {
            c.AbortWithStatusJSON(401, Error{Message: err.Error()})
            return
        }

        // Set tenant context
        c.Set("tenant_id", claims.TenantID)
        c.Set("team_id", claims.TeamID)
        c.Set("user_id", claims.UserID)
        c.Set("roles", claims.Roles)
        c.Set("scopes", claims.Scopes)
        c.Set("quota", claims.QuotaLimits)

        c.Next()
    }
}

// ScopedClient creates API client with tenant context
func ScopedClient(tenantID uuid.UUID) *Client {
    return &Client{
        baseURL:  Config.APIURL,
        tenantID: tenantID,
        auth:     Config.APIToken,
    }
}
```

### RBAC Model

```go
// pkg/tenant/rbac.go
package tenant

type Role string

const (
    RoleAdmin    Role = "admin"      // Full access
    RoleOperator Role = "operator"    // VM lifecycle
    RoleDeveloper Role = "developer"   // Limited VM access
    RoleViewer   Role = "viewer"     // Read-only
)

// Role permissions matrix
var RolePermissions = map[Role][]string{
    RoleAdmin: {
        "tenant:*",
        "vm:*",
        "network:*",
        "storage:*",
        "quota:*",
        "policy:*",
    },
    RoleOperator: {
        "vm:create",
        "vm:start",
        "vm:stop",
        "vm:delete",
        "vm:snapshot:*",
        "network:list",
        "storage:list",
    },
    RoleDeveloper: {
        "vm:create",
        "vm:start",
        "vm:stop",
        "network:list",
    },
    RoleViewer: {
        "vm:list",
        "vm:info",
        "network:list",
        "storage:list",
    },
}

// CheckPermission validates user has required permission
func CheckPermission(roles []Role, required string) bool {
    for _, role := range roles {
        perms := RolePermissions[role]
        for _, perm := range perms {
            if perm == required || perm == "*" {
                return true
            }
            // Wildcard match (e.g., "vm:*" matches "vm:create")
            if strings.HasSuffix(perm, ":*") {
                prefix := strings.TrimSuffix(perm, ":*")
                if strings.HasPrefix(required, prefix+":") {
                    return true
                }
            }
        }
    }
    return false
}
```

### Resource Tracking

```go
// pkg/tenant/usage.go
package tenant

type UsageTracker struct {
    tenantID uuid.UUID
    redis   *redis.Client
}

func (u *UsageTracker) IncrementUsage(resource string, delta int64) error {
    key := fmt.Sprintf("tenant:%s:usage:%s", u.tenantID, resource)
    return u.redis.IncrBy(ctx, key, delta).Err()
}

func (u *UsageTracker) GetUsage(resource string) (int64, error) {
    key := fmt.Sprintf("tenant:%s:usage:%s", u.tenantID, resource)
    val, err := u.redis.Get(ctx, key).Int64()
    if err == redis.Nil {
        return 0, nil
    }
    return val, err
}

// Usage report
type UsageReport struct {
    TenantID     uuid.UUID
    Period       string
    Resources    []ResourceUsage
    QuotaUsage   Quota
    QuotaLimit   Quota
    ProjectedCost float64
}

type ResourceUsage struct {
    Resource    string
    Used        int64
    Limit       int64
    Unit        string
    PercentUsed float64
}
```

### Consequences

### Positive
- Complete isolation between tenants
- Resource guarantees via quotas
- Fine-grained RBAC
- Cost attribution per tenant
- Compliance-ready

### Negative
- Complexity in resource allocation
- Network configuration overhead
- Storage management complexity
- Performance isolation challenges

# ADR-010: Security Model

## Status

Accepted

## Context

NanoVMS must provide strong isolation for:
- Multi-tenant workloads
- Untrusted code execution
- GPU passthrough
- Network isolation

## Decision

### Security Architecture

```
┌────────────────────────────────────────────────────────────────────┐
│                     NanoVMS Security Layers                           │
├────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  Layer 1: VM Isolation                                                │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                                                               │  │
│  │   ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌─────────┐ │  │
│  │   │ VFIO    │  │ KVM      │  │ Firecracker│ │ Kata   │ │  │
│  │   │ (PCI)   │  │ (CPU)    │  │ (jailer) │  │ (VM)   │ │  │
│  │   └──────────┘  └──────────┘  └──────────┘  └─────────┘ │  │
│  │                                                               │  │
│  │   IOMMU ───► VT-d / AMD-Vi ───► Device isolation             │  │
│  │                                                               │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                              │                                        │
│                              ▼                                        │
│  Layer 2: Container Isolation                                          │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                                                               │  │
│  │   namespaces ───► PID, net, mount, IPC, UTS, user            │  │
│  │   cgroups ───► CPU, memory, I/O limits                       │  │
│  │   seccomp ───► Syscall allowlisting                          │  │
│  │   AppArmor ───► Profile enforcement                           │  │
│  │                                                               │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                              │                                        │
│                              ▼                                        │
│  Layer 3: User-Space Kernel                                          │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                                                               │  │
│  │   gVisor (runsc) ───► Sentry + Gofer                        │  │
│  │   • Syscall interception                                      │  │
│  │   • Address space randomization                              │  │
│  │   • Minimal device model                                    │  │
│  │                                                               │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                              │                                        │
│                              ▼                                        │
│  Layer 4: Sandboxing                                                 │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                                                               │  │
│  │   bwrap ───► Linux namespaces (unprivileged)                  │  │
│  │   landlock ───► Filesystem restrictions                       │  │
│  │   seccomp ───► Syscall filtering                            │  │
│  │   firejail ───► AppArmor profiles                            │  │
│  │                                                               │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                                                     │
└────────────────────────────────────────────────────────────────────┘
```

### Security Profiles

#### MicroVM Profile (Firecracker)

```json
{
  "name": "microvm-secure",
  "hypervisor": "firecracker",
  "isolation": {
    "type": "kvm",
    "jailer": {
      "uid": 65534,
      "gid": 65534,
      "chroot": true,
      "newuidmap": true,
      "newgidmap": true
    }
  },
  "resources": {
    "memory": {
      "limit": "256M"
    },
    "vcpu": {
      "count": 1
    }
  },
  "capabilities": {
    "drop": ["all"],
    "keep": []
  },
  "seccomp": {
    "mode": "notify",
    "allowlist": ["read", "write", "exit", "sigreturn"]
  }
}
```

#### Container Profile (gVisor)

```json
{
  "name": "container-secure",
  "runtime": "gvisor",
  "isolation": {
    "type": "ptrace",
    "platform": "ptrace"
  },
  "resources": {
    "memory": {
      "limit": "1G"
    }
  },
  "capabilities": {
    "drop": ["all"]
  },
  "seccomp": {
    "mode": "trap",
    "errno_ret": 1
  },
  "apparmor": {
    "profile": "nanovms-container"
  }
}
```

#### Sandbox Profile (bwrap)

```bash
#!/bin/bash
# bwrap sandbox profile
bwrap \
  --unshare-user \
  --unshare-pid \
  --unshare-net \
  --unshare-uts \
  --unshare-ipc \
  --ro-bind /usr /usr \
  --ro-bind /lib /lib \
  --ro-bind /bin /bin \
  --tmpfs /tmp \
  --tmpfs /run \
  --proc /proc \
  --dev /dev \
  --hostname nanovms-sandbox \
  /bin/sh
```

### Threat Model

| Threat | Mitigation | Layer |
|--------|------------|-------|
| VM escape | KVM isolation, IOMMU | Hypervisor |
| Container breakout | namespaces, seccomp | OS |
| Syscall exploitation | gVisor, landlock | User-space |
| Privilege escalation | Drop capabilities | Capability |
| Resource exhaustion | cgroups limits | Resource |
| Network attack | Network namespaces | Network |
| File system access | Read-only binds | Filesystem |
| Hardware access | VFIO isolation | PCI |

### Compliance

```go
// pkg/security/compliance.go
package security

// Compliance standards supported
const (
    StandardSOC2 = "soc2"
    StandardHIPAA = "hipaa"
    StandardPCI = "pci-dss"
    StandardFedRAMP = "fedramp"
)

// ComplianceChecker validates security posture
type ComplianceChecker struct {
    standard string
}

func (c *ComplianceChecker) Validate(vm *VM) error {
    switch c.standard {
    case StandardSOC2:
        return c.validateSOC2(vm)
    case StandardHIPAA:
        return c.validateHIPAA(vm)
    case StandardPCI:
        return c.validatePCI(vm)
    }
    return nil
}

func (c *ComplianceChecker) validateSOC2(vm *VM) error {
    checks := []struct {
        name string
        fn   func(*VM) bool
    }{
        {"encrypted_transport", func(v *VM) bool { return v.TLSEnabled }},
        {"audit_logging", func(v *VM) bool { return v.AuditEnabled }},
        {"access_controls", func(v *VM) bool { return v.AuthRequired }},
        {"data_at_rest_encryption", func(v *VM) bool { return v.EncryptionEnabled }},
    }

    for _, check := range checks {
        if !check.fn(vm) {
            return fmt.Errorf("SOC2 check failed: %s", check.name)
        }
    }
    return nil
}
```

### Audit Logging

```go
// pkg/security/audit.go
package security

type AuditEvent struct {
    Timestamp   time.Time
    VMID       string
    UserID     string
    Action     string
    Resource   string
    Result     string
    IPAddress  string
    UserAgent  string
    Metadata   map[string]interface{}
}

func (a *AuditLogger) Log(event *AuditEvent) error {
    // Log to multiple destinations
    go a.logToFile(event)
    go a.logToSyslog(event)
    go a.logToCloudwatch(event)
    return nil
}

// Audit events to track
var auditedActions = map[string]bool{
    "vm.create":        true,
    "vm.start":          true,
    "vm.stop":           true,
    "vm.delete":         true,
    "vm.exec":           true,
    "vm.snapshot":       true,
    "vm.restore":        true,
    "gpu.attach":        true,
    "gpu.detach":        true,
    "network.attach":    true,
    "network.detach":    true,
    "storage.attach":    true,
    "storage.detach":    true,
    "admin.user.create": true,
    "admin.user.delete": true,
    "admin.policy.update": true,
}
```

## Consequences

### Positive
- Defense in depth
- Multiple isolation layers
- Compliance-ready
- Audit trail

### Negative
- Performance overhead from security layers
- Complexity in configuration
- Potential for security misconfiguration

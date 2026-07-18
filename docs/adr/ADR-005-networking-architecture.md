# ADR-005: Networking Architecture

## Status

Accepted

## Context

NanoVMS needs networking capabilities for:
- VM-to-VM communication
- Host-to-VM access
- NAT / bridge networking
- Port forwarding
- MacVTap / Linux bridges
- VFIO networking for GPU workloads

## Decision Drivers

- Performance (throughput, latency)
- Isolation between VMs
- Cross-platform support
- Security

## Options Considered

### Option 1: Linux Bridge + TAP (Selected)

**Pros:**
- Works everywhere
- Good performance
- Simple configuration
- NAT and bridge modes

**Cons:**
- Limited to Linux

### Option 2: MacVTap

**Pros:**
- Direct connection to host NIC
- Good performance
- Simple

**Cons:**
- Linux/macOS only

### Option 3: Slirp (User-mode)

**Pros:**
- No root required
- Works everywhere

**Cons:**
- Poor performance
- High latency

### Option 4: DPDK (for NFV workloads)

**Pros:**
- 10Gbps+ throughput
- Kernel bypass

**Cons:**
- Requires hugepages
- Complex setup
- Not for general use

## Decision

We implement a **tiered approach**:

1. **Slirp** (default): No root, works everywhere
2. **Bridge** (recommended): Best performance/isolation balance
3. **MacVTap**: Direct host connection
4. **DPDK** (future): For NFV workloads

## Consequences

### Positive
- Works on all platforms
- Progressive performance improvement
- Security through isolation

### Negative
- Bridge requires root on Linux
- Different behavior per mode

## Implementation

```go
type NetworkMode string

const (
    ModeNAT     NetworkMode = "nat"
    ModeBridge  NetworkMode = "bridge"
    ModeTap     NetworkMode = "tap"
    ModeSlirp   NetworkMode = "slirp"
)

type NetworkManager struct {
    mode NetworkMode
    bridge string
}

func (n *NetworkManager) CreateVMNetwork(ctx context.Context, vmID string) (*NetworkConfig, error) {
    switch n.mode {
    case ModeBridge:
        return n.createBridgeNetwork(vmID)
    case ModeTap:
        return n.createTapNetwork(vmID)
    case ModeSlirp:
        return n.createSlirpNetwork(vmID)
    default:
        return n.createNATNetwork(vmID)
    }
}
```

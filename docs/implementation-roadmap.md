# Implementation Roadmap

## Overview

This roadmap outlines the implementation phases for NanoVMS, a high-performance virtualization platform targeting SOTA cloud infrastructure on consumer hardware.

## Timeline

```
Q1 2026          Q2 2026          Q3 2026          Q4 2026
├─────────────────┼─────────────────┼─────────────────┤
│ Phase 1        │ Phase 2        │ Phase 3        │ Phase 4
│ Core Foundation │ Game Automation │ Consumer HW    │ Production
│                 │                 │ Optimization   │ Hardening
```

## Phase 1: Core Foundation (Q1 2026)

### Goals
- Establish core hypervisor abstraction layer
- Implement Tier 2-4 VM flavors (MicroVM, Container, Sandbox)
- Basic CLI with essential commands

### Milestones

| Week | Deliverable | Status |
|------|-------------|--------|
| 1-2 | Project scaffolding, CI/CD setup | |
| 3-4 | Hypervisor abstraction (Firecracker, QEMU, containerd) | |
| 5-6 | MicroVM implementation (Tier 2) | |
| 7-8 | Container runtime (Tier 3) | |
| 9-10 | Sandbox isolation (Tier 4) | |
| 11-12 | Basic CLI, integration tests | |
| 13 | Phase 1 release | |

### Implementation Details

#### Week 1-2: Project Scaffolding

```bash
# Initialize Go module
go mod init github.com/KooshaPari/nanovms

# Create directory structure
mkdir -p cmd/nanovms \
         pkg/{hypervisor,vm,container,sandbox,network,storage} \
         internal/{adapters,ports,domain} \
         test/{integration,benchmarks} \
         docs/{adr,guide,reference}

# Setup CI/CD
gh workflow create --name ci
gh workflow create --name release
```

#### Week 3-4: Hypervisor Abstraction

```go
// pkg/hypervisor/manager.go
type Manager struct {
    hypervisors map[string]Hypervisor
    default     Hypervisor
}

type Hypervisor interface {
    CreateVM(ctx context.Context, cfg VMConfig) (*VM, error)
    StartVM(ctx context.Context, id string) error
    StopVM(ctx context.Context, id string) error
    DeleteVM(ctx context.Context, id string) error
    ListVMs(ctx context.Context) ([]*VM, error)
    GetVMInfo(ctx context.Context, id string) (*VMInfo, error)
}

// Implementations:
// - FirecrackerHypervisor (Rust FFI)
// - QEMUMypervisor (CLI wrapper)
// - ContainerdHypervisor (containerd API)
```

#### Week 5-6: MicroVM (Tier 2)

```go
// pkg/vm/microvm.go
type MicroVM struct {
    id        string
    flavor    VMFlavor
    resources ResourceConfig
    state     VMState
    snapshot  []byte
}

func (m *MicroVM) Create(ctx context.Context) error {
    // Use Firecracker for <125ms startup
    // Pre-allocate memory and CPU
    // Setup virtio-net and virtio-blk
}

func (m *MicroVM) Snapshot(ctx context.Context) error {
    // Pause VM
    // Serialize memory state
    // Store to disk
    // Resume VM
}

func (m *MicroVM) Restore(ctx context.Context) error {
    // Allocate resources
    // Load memory state
    // Resume execution
}
```

### Phase 1 Exit Criteria
- [ ] Can create/start/stop/delete VMs
- [ ] Firecracker microVM starts in <125ms
- [ ] Basic CLI commands work
- [ ] Unit tests pass with >70% coverage
- [ ] Integration tests pass

---

## Phase 2: Game Automation Testing (Q2 2026)

### Goals
- Implement game automation testing infrastructure
- Steam integration (headless mode)
- Parallel VM orchestration
- <10s game startup target

### Milestones

| Week | Deliverable | Status |
|------|-------------|--------|
| 14-15 | VM pool management | |
| 16-17 | Steam headless integration | |
| 18-19 | Game automation runner | |
| 20-21 | Parallel test execution | |
| 22-23 | Result aggregation | |
| 24 | Phase 2 release | |

### Implementation Details

#### Week 14-15: VM Pool Management

```go
// pkg/vm/pool.go
type Pool struct {
    vms       []*MicroVM
    available chan *MicroVM
    active    map[string]*MicroVM
    maxSize   int
    factory   VMFactory
}

func (p *Pool) Acquire(ctx context.Context) (*MicroVM, error) {
    select {
    case vm := <-p.available:
        p.active[vm.id] = vm
        return vm, nil
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
        if len(p.vms) < p.maxSize {
            vm := p.factory.Create()
            p.vms = append(p.vms, vm)
            p.active[vm.id] = vm
            return vm, nil
        }
        // Wait for available VM
        return p.Acquire(ctx)
    }
}

func (p *Pool) Release(vm *MicroVM) {
    delete(p.active, vm.id)
    vm.Reset()  // Reset to base state
    p.available <- vm
}
```

#### Week 16-17: Steam Integration

```go
// pkg/game/steam/headless.go
type HeadlessSteam struct {
    vm      *MicroVM
    token   string
    appID   string
}

func (h *HeadlessSteam) Launch(ctx context.Context, appID string, token string) error {
    // Install Steam if not present
    if !h.isSteamInstalled() {
        if err := h.installSteam(ctx); err != nil {
            return err
        }
    }

    // Start Steam in headless mode
    cmd := []string{
        "steam",
        "-headless",
        "-applaunch", appID,
        "-steampath", "/usr/games/steam",
    }

    if token != "" {
        cmd = append(cmd, "-token", token)
    }

    return h.vm.Exec(ctx, cmd)
}

func (h *HeadlessSteam) WaitForReady(ctx context.Context, timeout time.Duration) error {
    deadline := time.Now().Add(timeout)

    for time.Now().Before(deadline) {
        if h.isGameRunning() {
            return nil
        }
        time.Sleep(500 * time.Millisecond)
    }

    return fmt.Errorf("game not ready after %v", timeout)
}
```

#### Week 18-19: Game Automation Runner

```go
// pkg/game/runner/runner.go
type Runner struct {
    pool      *Pool
    steam     *HeadlessSteam
    recorder  *ActionRecorder
    validator *StateValidator
}

type TestCase struct {
    Name        string
    Setup       func(*GameContext) error
    Steps       []TestStep
    Assertions  []Assertion
}

type TestStep struct {
    Name     string
    Action   func(*GameContext) error
    Screenshot bool
}

type GameContext struct {
    VM      *MicroVM
    Steam   *HeadlessSteam
    Mods    []string
    ScreenshotDir string
}

func (r *Runner) Run(ctx context.Context, tc *TestCase) (*TestResult, error) {
    vm, err := r.pool.Acquire(ctx)
    if err != nil {
        return nil, err
    }
    defer r.pool.Release(vm)

    gc := &GameContext{
        VM:            vm,
        Steam:         r.steam,
        ScreenshotDir: "/tmp/screenshots",
    }

    // Setup
    if tc.Setup != nil {
        if err := tc.Setup(gc); err != nil {
            return &TestResult{Success: false, Error: err}, nil
        }
    }

    // Execute steps
    for _, step := range tc.Steps {
        if step.Screenshot {
            r.recorder.Capture(gc)
        }
        if err := step.Action(gc); err != nil {
            return &TestResult{Success: false, Error: err}, nil
        }
    }

    // Validate assertions
    for _, assertion := range tc.Assertions {
        if err := r.validator.Validate(gc, assertion); err != nil {
            return &TestResult{Success: false, Error: err}, nil
        }
    }

    return &TestResult{Success: true}, nil
}
```

### Phase 2 Exit Criteria
- [ ] VM pool with <500ms acquire time
- [ ] Steam headless launches game
- [ ] Can run parallel tests
- [ ] <10s game startup achieved
- [ ] Test results aggregated

---

## Phase 3: Consumer Hardware Optimization (Q3 2026)

### Goals
- Implement consumer hardware profiles
- VFIO/GPU passthrough support
- Performance optimization
- Looking Glass integration

### Milestones

| Week | Deliverable | Status |
|------|-------------|--------|
| 25-26 | Hardware detection | |
| 27-28 | VFIO integration | |
| 29-30 | Performance tuning | |
| 31-32 | Looking Glass setup | |
| 33-34 | Resource optimization | |
| 35 | Phase 3 release | |

### Implementation Details

#### Week 25-26: Hardware Detection

```go
// pkg/hardware/detect.go
type HardwareProfile struct {
    CPU       CPUInfo
    Memory    MemoryInfo
    GPU       GPUInfo
    Storage   []StorageInfo
    Network   []NetworkInfo
    IOMMU     bool
    SR_IOV    bool
}

type CPUInfo struct {
    Vendor   string
    Model    string
    Cores    int
    Threads   int
    Features  []string  // "vmx", "svm", "aes", etc.
}

func Detect() (*HardwareProfile, error) {
    profile := &HardwareProfile{}

    // Detect CPU
    profile.CPU = detectCPU()

    // Detect memory
    profile.Memory = detectMemory()

    // Detect GPU
    profile.GPU = detectGPU()

    // Detect storage
    profile.Storage = detectStorage()

    // Detect network
    profile.Network = detectNetwork()

    // Check virtualization support
    profile.IOMMU = checkIOMMU()
    profile.SR_IOV = checkSR_IOV()

    return profile, nil
}

func detectCPU() CPUInfo {
    // Read /proc/cpuinfo
    // Parse vendor, model, flags
    // Check for VM features (vmx/svm)
}

func (h *HardwareProfile) RecommendTier() VMFlavor {
    if !h.IOMMU {
        return Tier4Sandbox  // No GPU passthrough
    }

    if h.GPU.IsNvidia() && h.GPU.IsModern() {
        return Tier1VFIO  // Full GPU passthrough
    }

    if h.GPU.IsAMD() || h.GPU.IsIntel() {
        return Tier1VFIO  // AMDVi/GVT-g
    }

    return Tier2MicroVM  // Software rendering
}
```

#### Week 27-28: VFIO Integration

```go
// pkg/vfio/manager.go
type Manager struct {
    iommuGroups map[int][]string  // group ID -> devices
    boundDevices map[string]string // device -> driver
}

func (m *Manager) BindGPU(pciAddr string) error {
    // Unbind from current driver
    if err := m.unbind(pciAddr); err != nil {
        return err
    }

    // Get IOMMU group
    group, err := m.getIOMMUGroup(pciAddr)
    if err != nil {
        return err
    }

    // Bind all devices in group to vfio-pci
    for _, device := range group.Devices {
        if err := m.bindToVFIO(device); err != nil {
            return err
        }
    }

    return nil
}

func (m *Manager) AttachToVM(vmID string, pciAddr string) error {
    // Configure VM to use VFIO device
    vm := vm.Get(vmID)

    qemuArgs := []string{
        "-device", fmt.Sprintf("vfio-pci,host=%s", pciAddr),
    }

    return vm.UpdateArgs(qemuArgs)
}
```

### Phase 3 Exit Criteria
- [ ] Auto-detects hardware capabilities
- [ ] VFIO binds GPU successfully
- [ ] GPU passthrough works
- [ ] Looking Glass displays VM
- [ ] Performance benchmarks pass

---

## Phase 4: Production Hardening (Q4 2026)

### Goals
- Security hardening
- Multi-tenancy
- Production deployment
- Documentation complete

### Milestones

| Week | Deliverable | Status |
|------|-------------|--------|
| 36-37 | Security audit | |
| 38-39 | Multi-tenancy | |
| 40-41 | Monitoring/Observability | |
| 42-43 | Documentation | |
| 44 | Release candidate | |
| 45-46 | GA release | |
| 47-48 | Post-release | |

### Phase 4 Exit Criteria
- [ ] Security audit complete
- [ ] Multi-tenant isolation verified
- [ ] Monitoring operational
- [ ] Docs complete
- [ ] GA release

---

## Resource Requirements

### Team Structure

| Role | Q1 | Q2 | Q3 | Q4 |
|------|----|----|----|----|
| Core Developers | 2 | 2 | 2 | 1 |
| Security Engineer | 0 | 0 | 1 | 1 |
| DevOps | 1 | 1 | 1 | 1 |
| Technical Writer | 0 | 0 | 0 | 1 |

### Infrastructure

| Resource | Q1 | Q2 | Q3 | Q4 |
|----------|----|----|----|----|
| Build agents | 2 | 4 | 4 | 2 |
| Test VMs | 4 | 16 | 8 | 4 |
| Storage (TB) | 1 | 4 | 2 | 1 |

---

## Risk Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| Firecracker FFI complexity | High | Use existing Go bindings |
| GPU passthrough issues | Medium | Multiple fallback paths |
| Steam anti-cheat conflicts | Medium | Focus on modding community |
| Performance targets missed | Medium | Iterate on optimization |
| Resource constraints | Low | Prioritize critical paths |

---

## Success Metrics

| Metric | Phase 1 | Phase 2 | Phase 3 | Phase 4 |
|--------|---------|---------|---------|---------|
| VM startup | <125ms | <100ms | <50ms | <50ms |
| CLI coverage | 60% | 80% | 90% | 100% |
| Test coverage | 70% | 80% | 85% | 90% |
| Documentation | 50% | 70% | 85% | 100% |
| Concurrent VMs | 10 | 50 | 100 | 100 |
| Game startup | - | <10s | <8s | <8s |

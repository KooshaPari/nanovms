# ADR-001: Optimal Language Selection for NanoVMS

**Date**: 2026-04-02
**Status**: Proposed
**Deciders**: KooshaPari

## Context

NanoVMS (Nano Virtual Machine Services) provides lightweight, headless VM abstraction for agent-driven development workflows. We need to choose the optimal language(s) for implementing:

1. **VM Adapters** (Lima, WSL, Native, Firecracker)
2. **Sandbox Isolation** (bwrap, firejail, gVisor)
3. **WASM Runtime** support
4. **CLI and orchestration**

The project currently uses Go, but we need to evaluate if a language switch or multi-language approach is warranted.

## Decision Drivers

- **Performance**: VM startup time, memory footprint, container creation rate
- **Security**: Memory safety is critical for sandbox isolation
- **Ecosystem**: Existing libraries and tooling for virtualization
- **Maintainability**: Team expertise and code longevity
- **Startup Speed**: Critical for agent-driven ephemeral workloads

## Options Considered

### Option A: Go (Current)

**Pros**:
- Industry standard for containers (runc, containerd, Docker)
- Large ecosystem for orchestration and cloud-native tools
- Easy deployment and cross-compilation
- Strong async runtime (goroutines)

**Cons**:
- GC pauses affect latency-sensitive operations
- 2x slower container creation than Rust youki
- Higher memory overhead than Rust/C

### Option B: Rust (Recommended)

**Pros**:
- Dominates VMM space (Firecracker, Cloud Hypervisor, crosvm)
- Leading WASM runtime (Wasmtime)
- Memory safety without GC (critical for security)
- 2x faster than Go for container operations
- rust-vmm project provides shared components

**Cons**:
- Steeper learning curve
- Longer compile times
- Smaller talent pool than Go

### Option C: C

**Pros**:
- Fastest container runtime (crun: 47ms vs youki: 111ms vs runc: 225ms)
- Lowest memory footprint
- Direct system access

**Cons**:
- Memory safety vulnerabilities
- Manual memory management overhead
- Higher development time

### Option D: Multi-Language (Hybrid)

**Pros**:
- Best tool for each job
- Leverage existing battle-tested implementations

**Cons**:
- Complexity in build system
- Cross-language dependencies
- Testing complexity

### Option E: Zig / Carbon / Mojo

**Decision**: Rejected for production use in 2026.

| Language | Reason |
|----------|--------|
| Zig | Ecosystem not mature enough for isolation workloads |
| Carbon | Early development, years from production |
| Mojo | Focuses on GPU kernels, stdlib not stable |

## Decision

**Adopt a hybrid approach using Rust as the primary language with strategic Go integration:**

1. **New VM adapters** → Rust (using rust-vmm components)
2. **CLI and orchestration** → Go (leverage existing code)
3. **Sandbox adapters** → Rust via FFI or Go bindings
4. **WASM support** → Rust (Wasmtime bindings)

### Rationale

| Workload | Language | Justification |
|----------|----------|---------------|
| **VMM layer** | Rust | Firecracker, Cloud Hypervisor use Rust; rust-vmm provides components |
| **Container runtime** | Rust (youki) | 2x faster than runc, memory safe |
| **CLI/Tooling** | Go | Existing code, large ecosystem |
| **WASM runtime** | Rust (Wasmtime) | Bytecode Alliance standard |
| **Integration adapters** | Go → Rust FFI | Bridge existing Go with new Rust components |

## Performance Benchmarks

| Component | Language | Benchmark | Source |
|-----------|----------|-----------|--------|
| Firecracker | Rust | <125ms startup, <5MB RAM | AWS production |
| youki (containerd) | Rust | 111ms create/start/delete | youki benchmarks |
| runc | Go | 225ms create/start/delete | youki benchmarks |
| crun | C | 47ms create/start/delete | youki benchmarks |
| Wasmtime | Rust | ~1ms startup, ~1MB RAM | Bytecode Alliance |

## Implementation Plan

### Phase 1: Research & PoC
- [ ] Evaluate rust-vmm component library
- [ ] Benchmark youki vs runc in NanoVMS workloads
- [ ] Proof-of-concept Rust VM adapter for Lima

### Phase 2: Incremental Migration
- [ ] Add Rust-based Firecracker adapter
- [ ] Add Rust-based WASM runtime adapter
- [ ] Migrate sandbox adapters to Rust

### Phase 3: Full Integration
- [ ] Rust CLI scaffolding for VM operations
- [ ] Go/Rust FFI for orchestration layer
- [ ] Deprecate Go-only VM adapters

## Consequences

### Positive
- Improved VM startup times (<125ms vs current)
- Reduced memory footprint
- Stronger security guarantees (memory safety)
- Alignment with industry leaders (AWS, Google, Bytecode Alliance)

### Negative
- Increased complexity (multi-language)
- Learning curve for Rust
- Longer initial development time
- Build system complexity

## References

- [rust-vmm project](https://github.com/rust-vmm)
- [Firecracker](https://github.com/firecracker-microvm/firecracker)
- [youki benchmarks](https://github.com/youki-dev/youki)
- [Wasmtime](https://github.com/bytecodealliance/wasmtime)
- [Phenotype 2026 Tech Radar](./TECH_RADAR.md)

---

*This ADR will be updated as the implementation progresses.*

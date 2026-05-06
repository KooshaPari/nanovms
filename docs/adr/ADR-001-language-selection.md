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
- Higher memory overhead than Rust/Zig

### Option B: Rust

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

### Option C: Zig (Preferred for Low-Level)

**Pros**:
- **No hidden control flow** — explicit error handling, no hidden allocations
- ** comptime** — compile-time code execution for zero-cost abstractions
- ** comptime generics** — pattern matching at compile time
- **Better C interop** — drop-in replacement for C, easy syscall access
- **Simpler than Rust** — easier learning curve, no borrow checker
- **Debuggable binaries** — no LLVM overhead in debug builds
- **Minimal runtime** — perfect for embedded/kernel-level work
- **Explicit memory** — you know exactly when/where allocations happen

**Cons**:
- Ecosystem younger than Rust (but growing rapidly)
- Smaller standard library — needs more external dependencies
- Less production VMM adoption (but that means opportunity)

**Why Zig for VM/Sandbox work**:
- Direct syscall access without FFI overhead
- Comptime makes code generation zero-cost
- Better than C for systems programming without Rust's complexity
- Emerging VMM projects (Kobold,unikraft-zig) show viability

### Option D: C

**Pros**:
- Fastest container runtime (crun: 47ms vs youki: 111ms vs runc: 225ms)
- Lowest memory footprint
- Direct system access

**Cons**:
- Memory safety vulnerabilities
- Manual memory management overhead
- Higher development time

### Option E: Carbon / Mojo

**Decision**: Rejected for production use in 2026.

| Language | Reason |
|----------|--------|
| Carbon | Early development, years from production |
| Mojo | Focuses on GPU kernels, stdlib not stable |

## Decision

**Adopt a multi-language approach with Zig as the primary language for low-level components and Rust for VMM/WASM:**

### Language Assignments

| Component | Language | Justification |
|----------|----------|---------------|
| **VMM core** | Rust | Firecracker, Cloud Hypervisor; rust-vmm components |
| **Hypervisor adapters** | Zig | Direct syscall access, comptime code gen |
| **Sandbox isolation** | Zig | No hidden allocations, explicit memory, debuggable |
| **WASM runtime** | Rust | Wasmtime, Bytecode Alliance standard |
| **CLI/Tooling** | Go | Existing code, large ecosystem |
| **Systems utilities** | Zig | shims, loaders, low-level tooling |

### Why Zig + Rust?

| Aspect | Rust | Zig |
|--------|------|-----|
| Memory safety | ✅ | ✅ (explicit) |
| Compile time | Slower | Faster |
| Error handling | `Result<T, E>` | `!` operator |
| Generics | Complex trait bounds | comptime |
| Learning curve | Steep (borrow checker) | Moderate |
| C interop | Good | Excellent |
| Ecosystem | Mature | Growing |
| Debug builds | LLVM overhead | Minimal |

**Zig wins for**: Sandbox loaders, VM shims, low-level init, comptime-heavy code
**Rust wins for**: Long-running services, complex async, WASM, established VMM

## Performance Benchmarks

| Component | Language | Benchmark | Source |
|-----------|----------|-----------|--------|
| Firecracker | Rust | <125ms startup, <5MB RAM | AWS production |
| youki (containerd) | Rust | 111ms create/start/delete | youki benchmarks |
| runc | Go | 225ms create/start/delete | youki benchmarks |
| crun | C | 47ms create/start/delete | youki benchmarks |
| Wasmtime | Rust | ~1ms startup, ~1MB RAM | Bytecode Alliance |
| Zig std lib | Zig | Comparable to C | Zig benchmarks |

## Implementation Plan

### Phase 1: Research & PoC
- [ ] Evaluate rust-vmm component library
- [ ] Benchmark youki vs runc in NanoVMS workloads
- [ ] Proof-of-concept Zig VM adapter for Lima
- [ ] Test Zig comptime code generation for VM config

### Phase 2: Zig Integration
- [ ] Add Zig-based sandbox loader
- [ ] Add Zig hypervisor shims
- [ ] Integrate Zig build via `build.zig`

### Phase 3: Rust VMM Components
- [ ] Add Rust-based Firecracker adapter
- [ ] Add Rust-based WASM runtime adapter
- [ ] Migrate sandbox adapters

### Phase 4: Full Integration
- [ ] Go CLI orchestration with Zig/Rust FFI
- [ ] Unified build system (Go + Cargo + zig build)
- [ ] Deprecate Go-only VM adapters

## Consequences

### Positive
- **Zig advantages**: Explicit memory, no hidden allocations, comptime power, better C interop
- **Rust advantages**: VMM battle-tested, WASM standard, large ecosystem
- **Combined**: Best tool for each job, clear boundaries

### Negative
- Multi-language complexity
- Learning curve for Zig
- Build system coordination (Go + Cargo + Zig)
- Smaller Zig ecosystem than Rust

## References

- [rust-vmm project](https://github.com/rust-vmm)
- [Firecracker](https://github.com/firecracker-microvm/firecracker)
- [youki benchmarks](https://github.com/youki-dev/youki)
- [Wasmtime](https://github.com/bytecodealliance/wasmtime)
- [Zig language](https://ziglang.org/)
- [zigvm (experimental)](https://github.com/AndreaPuca/zigvm)
- [Kobold VMM](https://github.com/joshke小人/kobold)

---

*This ADR will be updated as the implementation progresses.*

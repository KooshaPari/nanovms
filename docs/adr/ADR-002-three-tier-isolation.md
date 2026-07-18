# ADR-002: Three-Tier Isolation Architecture

**Date**: 2026-04-02
**Status**: Proposed
**Deciders**: KooshaPari

## Context

NanoVMS provides VM abstraction for AI agent workloads. We need an architecture that balances:
- **Speed**: Agents need fast iteration (<100ms for tool execution)
- **Security**: Agents may execute untrusted LLM-generated code
- **Compatibility**: Must run OCI images and existing containers
- **Efficiency**: Memory footprint critical for multi-agent workloads

## Decision

**Adopt a three-tier isolation architecture based on trust level:**

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Agent Controller                               │
├─────────────────────────────────────────────────────────────────────┤
│  Tier 1: WASM Sandboxes (~1ms startup, ~1MB memory)              │
│  └── Fast tool execution, WASI sandbox, no syscalls                │
├─────────────────────────────────────────────────────────────────────┤
│  Tier 2: gVisor Containers (~90ms startup, ~20MB memory)         │
│  └── Syscall filtering, network isolation                           │
├─────────────────────────────────────────────────────────────────────┤
│  Tier 3: MicroVMs (~125ms startup, <5MB memory)                  │
│  └── Firecracker, OCI compatible, full hardware isolation            │
└─────────────────────────────────────────────────────────────────────┘
```

## Tier Selection Framework

| Trust Level | Code Source | Tier | Technology |
|-------------|-------------|------|------------|
| **Trusted** | Agent-native tools, first-party | 1 - WASM | Wasmtime |
| **Semi-trusted** | Third-party tools, scripts | 2 - gVisor | runsc |
| **Untrusted** | LLM-generated code, external | 3 - MicroVM | Firecracker |

## Architecture Details

### Tier 1: WASM Sandboxes

**Use Cases**:
- Agent tool execution (code formatters, linters, compilers)
- Plugin systems with language-agnostic execution
- Edge computing and serverless functions
- Zero-install execution

**Technology**: Wasmtime (Bytecode Alliance standard)

**Benefits**:
- ~1ms startup time
- ~1MB memory footprint
- Language-agnostic (Rust, Go, C, Python, etc.)
- Strong sandboxing via WASI
- No syscalls to host kernel

**Limitations**:
- Cannot run arbitrary Linux binaries
- Limited OS access (WASI only)
- No GPU access

### Tier 2: gVisor Containers

**Use Cases**:
- Third-party tool execution with network access
- Semi-trusted scripts and utilities
- Development containers with full OS
- Containerized development environments

**Technology**: gVisor (Google's userspace kernel)

**Benefits**:
- ~90ms startup time
- ~20MB memory footprint
- Runs OCI container images
- Full syscall filtering (seccomp)
- Network namespace isolation

**Limitations**:
- Performance overhead vs native containers
- Some syscalls may not be supported
- Larger than WASM

### Tier 3: MicroVMs

**Use Cases**:
- Untrusted LLM-generated code
- Full Linux environment requirements
- Compliance/multi-tenant isolation
- Running arbitrary Docker images

**Technology**: Firecracker (AWS)

**Benefits**:
- ~125ms startup time
- <5MB memory per microVM
- Full hardware virtualization (VT-x/AMD-V)
- OCI image compatibility
- Minimal attack surface (5 devices only)
- Production proven (AWS Lambda, Fargate)

**Limitations**:
- Larger than Tier 1/2
- Requires kernel access
- More resource overhead

## Performance Comparison

| Tier | Technology | Startup | Memory | Security |
|------|------------|---------|--------|----------|
| 1 | Wasmtime | ~1ms | ~1MB | WASI sandbox |
| 2 | gVisor | ~90ms | ~20MB | Userspace kernel |
| 3 | Firecracker | ~125ms | <5MB | Hardware VM |

## Implementation

### CLI Interface

```bash
# Tier 1: WASM (trusted tools)
nanovms sandbox create tool --tier wasm --wasm module.wasm
nanovms sandbox exec tool -- analyze-code ./src

# Tier 2: gVisor (semi-trusted)
nanovms sandbox create dev --tier gvisor --image ubuntu:22.04
nanovms sandbox exec dev -- npm test

# Tier 3: MicroVM (untrusted)
nanovms sandbox create untrusted --tier microvm --image debian:sid
nanovms sandbox exec untrusted -- ./generated-binary
```

### Agent Integration

```go
// Tier selection based on trust
func (e *Executor) Execute(ctx context.Context, code *Code, trust TrustLevel) error {
    switch trust {
    case TrustTrusted:
        return e.executeWasm(ctx, code)
    case TrustSemiTrusted:
        return e.executeGvisor(ctx, code)
    case TrustUntrusted:
        return e.executeMicroVM(ctx, code)
    }
}
```

## Consequences

### Positive
- Clear trust-based tier selection
- Optimal performance for each use case
- Defense in depth with stacking capability
- Industry-proven technologies

### Negative
- More complex architecture than single-tier
- Multiple runtime dependencies
- Testing complexity across tiers

## References

- [Firecracker](https://github.com/firecracker-microvm/firecracker) — AWS microVM
- [gVisor](https://github.com/google/gvisor) — Google sandbox
- [Wasmtime](https://github.com/bytecodealliance/wasmtime) — Bytecode Alliance
- [WASI](https://github.com/WebAssembly/WASI) — WebAssembly System Interface

---

*This ADR defines the three-tier isolation architecture for NanoVMS.*

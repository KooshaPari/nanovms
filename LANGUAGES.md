# Language Policy for Unified NVMS

## Document Information

| Field | Value |
|-------|-------|
| **ID** | lang-policy-nvms-001 |
| **Title** | Language Selection Policy - Unified NVMS Stack |
| **Created** | 2026-04-06 |
| **Status** | approved |
| **Scope** | KooshaPari/nvms, PhenoCompose, all Phenotype polyglot infra |

---

## Executive Summary

This policy establishes a **tiered language selection strategy** for the unified NVMS project:

```
┌─────────────────────────────────────────────────────────────────┐
│                    LANGUAGE TIER HIERARCHY                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  TIER 1 (Optimal) - Use by default                              │
│  ┌─────────┬─────────┬─────────┬─────────┐                      │
│  │   Go    │  Mojo   │  Zig    │  Rust   │                      │
│  └─────────┴─────────┴─────────┴─────────┘                      │
│                                                                  │
│  TIER 2 (Fallback) - Only when Tier 1 unsuitable                │
│  ┌─────────┬─────────┬─────────┬─────────┐                      │
│  │   C#    │ Python  │   TS    │  Swift  │  Kotlin             │
│  └─────────┴─────────┴─────────┴─────────┘                      │
│                                                                  │
│  ⚠️  TIER 2 = WORST CASE ONLY - Requires documented justification │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 1. Language Selection Matrix

### 1.1 Tier 1: Optimal Languages

| Language | Primary Use Case | Why Optimal | Performance | Concurrency |
|----------|-----------------|-------------|-------------|-------------|
| **Go** | Core orchestration, CLI, networking | Best cloud SDKs, simple concurrency, fast compile | High | Excellent (goroutines) |
| **Rust** | Performance-critical, FFI, safety-critical | Memory safety without GC, zero-cost abstractions | Highest | Good (async/await) |
| **Zig** | Low-level, embedded, build systems | No hidden control flow, perfect C interop | Highest | Manual only |
| **Mojo** | ML/AI workloads, Python alternative | Python ergonomics + Systems performance | Very High | Good (async) |

#### Go Selection Criteria

**Use Go when:**
- ✅ Building CLI tools and orchestration layers
- ✅ Network services, HTTP/gRPC servers
- ✅ Cloud infrastructure (AWS/GCP/Azure SDKs)
- ✅ Cross-platform support needed (macOS/Linux/Windows)
- ✅ Team familiarity (easy to onboard)

**Avoid Go when:**
- ❌ Extremely latency-sensitive (<1ms p99 required)
- ❌ Heavy numerical computation (ML/AI)
- ❌ Memory-constrained embedded systems
- ❌ Real-time systems with hard deadlines

#### Rust Selection Criteria

**Use Rust when:**
- ✅ Performance-critical paths (hot loops, data processing)
- ✅ FFI with C/Zig libraries
- ✅ Memory-constrained environments
- ✅ Safety-critical components
- ✅ Building reusable libraries

**Avoid Rust when:**
- ❌ Rapid prototyping needed
- ❌ Team lacks Rust experience
- ❌ Simple scripts or automation
- ❌ Long compile times are unacceptable

#### Zig Selection Criteria

**Use Zig when:**
- ✅ Building C-compatible libraries
- ✅ Embedded systems (no_std)
- ✅ Custom memory allocators
- ✅ Build system / package management
- ✅ Replacing C in performance-critical code

**Avoid Zig when:**
- ❌ Ecosystem immaturity is unacceptable
- ❌ Large team collaboration needed
- ❌ Complex async / networking
- ❌ Stable toolchain required

#### Mojo Selection Criteria

**Use Mojo when:**
- ✅ ML/AI model inference or training
- ✅ Python migration path needed
- ✅ SIMD/vectorization beneficial
- ✅ Prototype to production path

**Avoid Mojo when:**
- ❌ Not yet stable (Mojo is still in development)
- ❌ Pure backend services without ML
- ❌ Team unfamiliar with Python-like syntax

---

### 1.2 Tier 2: Fallback Languages

> **⚠️ WARNING**: Tier 2 languages are WORST CASE ONLY. Documented justification required.

| Language | Acceptable Fallback For | When Justified |
|----------|------------------------|----------------|
| **Python** | ML/AI (if Mojo unavailable), scripting | Legacy integration, rapid prototyping only |
| **C#** | .NET ecosystem integration | Windows-specific tooling only |
| **TypeScript** | Web UI, browser tooling | PhenoCompose web dashboard only |
| **Swift** | macOS/iOS native tooling | Apple platform tooling only |
| **Kotlin** | JVM ecosystem integration | Android tooling only |

#### Python Fallback Criteria

**Only use Python when:**
- ✅ Explicitly integrating with Python ML stack (PyTorch, TensorFlow)
- ✅ Legacy automation scripts that cannot be rewritten
- ✅ Single-use scripting (never for long-lived services)

**Never use Python for:**
- ❌ Core orchestration or services
- ❌ Performance-critical paths
- ❌ New feature development

#### C# Fallback Criteria

**Only use C# when:**
- ✅ Windows-specific tooling that cannot be Go/Rust
- ✅ .NET library integration required
- ✅ Team has C# expertise and no Go/Rust

**Never use C# for:**
- ❌ Cross-platform core services
- ❌ Performance-critical paths
- ❌ New feature development

#### TypeScript Fallback Criteria

**Only use TypeScript when:**
- ✅ Web UI for PhenoCompose dashboard
- ✅ Browser-based tooling

**Never use TypeScript for:**
- ❌ Backend services
- ❌ CLI tools
- ❌ Performance-critical paths

#### Swift/Kotlin Fallback Criteria

**Only use Swift/Kotlin when:**
- ✅ Native macOS/iOS PhenoCompose companion apps
- ✅ Android PhenoCompose companion apps
- ✅ Platform-specific tooling only

**Never use Swift/Kotlin for:**
- ❌ Core services
- ❌ Cross-platform tooling
- ❌ Performance-critical paths

---

## 2. Component Language Assignment

### 2.1 Current Unified NVMS Stack

```
┌─────────────────────────────────────────────────────────────────┐
│                    UNIFIED NVMS ARCHITECTURE                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  PhenoCompose (Rust) ─────┐                                     │
│                            │                                     │
│  ┌────────────────────────▼────────────────────────┐            │
│  │              NVMS Core (Go)                      │            │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────────────┐   │            │
│  │  │ WASM    │ │ gVisor  │ │ Firecracker     │   │            │
│  │  │ Runtime │ │ Runtime │ │ Orchestrator    │   │            │
│  │  │ (Go)    │ │ (Go)    │ │ (Go)           │   │            │
│  │  └─────────┘ └─────────┘ └─────────────────┘   │            │
│  └─────────────────────────────────────────────────┘            │
│                                                                  │
│  Platform Adapters (Tier 1)                                     │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐               │
│  │ macOS  │ │ Windows │ │ Linux   │ │  AWS    │               │
│  │ (Go)   │ │ (Go)    │ │ (Go)    │ │ (Go)    │               │
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘               │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 Language Assignment by Component

| Component | Language | Rationale | Tier |
|-----------|----------|-----------|------|
| **NVMS Core Orchestrator** | Go | Concurrency, cloud SDKs, simplicity | 1 |
| **WASM Runtime Adapter** | Go | Tight integration with Go core | 1 |
| **gVisor Runtime Adapter** | Go | Tight integration with Go core | 1 |
| **Firecracker Orchestrator** | Go/Rust | AWS SDK (Go), performance (Rust) | 1 |
| **Platform Adapters (macOS/Linux/Windows)** | Go | Cross-platform, FFI if needed | 1 |
| **PhenoCompose CLI** | Rust | Type safety, performance, CLI ergonomics | 1 |
| **PhenoCompose NVMS Driver** | Rust | FFI with NVMS Go library | 1 |
| **ML/AI Integration** | Mojo | Python-like + systems performance | 1 |
| **Low-level Memory/Build** | Zig | C interop, no hidden allocations | 1 |
| **Web Dashboard** | TypeScript | Browser UI only | 2 |
| **macOS Companion** | Swift | Native Apple tooling only | 2 |
| **Android Companion** | Kotlin | Native Android tooling only | 2 |

---

## 3. FFI and Binding Strategies

### 3.1 Go ↔ Rust Bindings

```rust
// Rust library exposed via C ABI
// nvms-core/src/lib.rs

#[repr(C)]
pub struct NvmsInstance {
    pub id: u64,
    pub tier: NvmsTier,
    pub status: NvmsStatus,
}

#[no_mangle]
pub extern "C" fn nvms_create(
    config: *const c_char,
) -> *mut NvmsInstance {
    // Safe Rust implementation
}
```

```go
// Go binding using cgo
// nvms/binding.go

/*
#cgo LDFLAGS: -lnvms_core
#cgo CFLAGS: -I${SRCDIR}/include
#include "nvms_core.h"
*/
import "C"

type Instance struct {
    ptr C.NvmsInstance
}

func Create(config string) (*Instance, error) {
    cConfig := C.CString(config)
    defer C.free(unsafe.Pointer(cConfig))

    ptr := C.nvms_create(cConfig)
    if ptr == nil {
        return nil, errors.New("failed to create instance")
    }
    return &Instance{ptr: ptr}, nil
}
```

### 3.2 Go ↔ Zig Bindings

```zig
// Zig library for low-level operations
// nvms-zig/src/memory.zig

pub const NvmsMemory = struct {
    pub fn allocate(size: u64) ?[*]u8 {
        return std.heap.page_allocator.alloc(u8, size) catch null;
    }

    pub fn deallocate(ptr: [*]u8, size: u64) void {
        std.heap.page_allocator.free(ptr[0..size]);
    }
};
```

```go
// Go binding using cgo + Zig C export
/*
#cgo LDFLAGS: -lnvms_zig_memory
*/
import "C"
```

### 3.3 Go ↔ Mojo Bindings

```python
# Mojo ML integration
# nvms-ml/inference.mojo

from tensor import Tensor
from memory import UnsafePointer

@register_passable("trivial")
pub struct MlModel {
    weights: UnsafePointer[Float32]
    bias: UnsafePointer[Float32]

    pub fn forward(self, input: Tensor) -> Tensor:
        # Forward pass implementation
        return output
}
```

```python
# Python/Go via NumPy FFI
import numpy as np

def inference_go(input_data: np.ndarray) -> np.ndarray:
    # Call Mojo via subprocess or shared library
    pass
```

---

## 4. Performance Benchmarks

### 4.1 Language Performance Comparison

| Operation | Go | Rust | Zig | Python | C# |
|-----------|-----|------|-----|--------|-----|
| **Startup Time** | ~50ms | ~100ms | ~20ms | ~500ms | ~200ms |
| **Memory Baseline** | ~10MB | ~5MB | ~2MB | ~50MB | ~30MB |
| **HTTP Latency (p99)** | <5ms | <1ms | <1ms | >50ms | ~20ms |
| **Compute Throughput** | High | Highest | Highest | Low | Medium |
| **Concurrency Model** | Goroutines | async/await | Manual | GIL | Threads |

### 4.2 Decision Framework

```
                    ┌─────────────────────────────┐
                    │    LANGUAGE SELECTION FLOW    │
                    └─────────────────────────────┘
                                       │
                                       ▼
                    ┌─────────────────────────────────┐
                    │ Is this performance-critical?   │
                    │   (<1ms latency, high compute)  │
                    └─────────────────────────────────┘
                              │                    │
                             YES                   NO
                              │                    │
                              ▼                    ▼
               ┌──────────────────┐    ┌─────────────────────────┐
               │ Need memory      │    │ Is this cloud/infra?   │
               │ safety + no GC?  │    │ (AWS/GCP/Azure SDK)    │
               └──────────────────┘    └─────────────────────────┘
                        │                        │
                       YES                      YES
                        │                        │
                        ▼                        ▼
                    ┌────────┐              ┌────────┐
                    │  Rust  │              │   Go   │
                    │ (Tier1)│              │ (Tier1)│
                    └────────┘              └────────┘

               ┌─────────────────────────────────────┐
               │ Need C interop or embedded?          │
               └─────────────────────────────────────┘
                              │
                             YES
                              │
                              ▼
                        ┌────────┐
                        │  Zig   │
                        │ (Tier1)│
                        └────────┘

               ┌─────────────────────────────────────┐
               │ Need ML/AI?                          │
               └─────────────────────────────────────┘
                              │
                             YES
                              │
                              ▼
                        ┌────────┐
                        │  Mojo  │
                        │ (Tier1)│
                        └────────┘

               ┌─────────────────────────────────────┐
               │ TIER 1 not suitable?                │
               │ (Document justification required)    │
               └─────────────────────────────────────┘
                              │
                             YES
                              │
                              ▼
                    ┌─────────────────┐
                    │ TIER 2 FALLBACK │
                    │ (Worst case)    │
                    └─────────────────┘
```

---

## 5. Team and Ecosystem Considerations

### 5.1 Team Skill Matrix

| Language | Phenotype Team Expertise | Learning Curve | Productivity |
|----------|-------------------------|----------------|--------------|
| **Go** | High | Low | High |
| **Rust** | Medium | High | Medium |
| **Zig** | Low | Very High | Low (early) |
| **Mojo** | Low | Medium | TBD |
| **Python** | High | Low | High |
| **TypeScript** | High | Low | High |
| **C#** | Low | Medium | Medium |

### 5.2 Ecosystem Maturity

| Language | Libraries | Tooling | Stability | Recommendation |
|----------|-----------|---------|-----------|----------------|
| **Go** | Excellent | Excellent | Stable | **Primary** |
| **Rust** | Excellent | Excellent | Stable | Performance-critical |
| **Zig** | Growing | Good | Pre-1.0 | Low-level only |
| **Mojo** | Emerging | Limited | Beta | ML only |
| **Python** | Excellent | Excellent | Stable | Legacy only |
| **TypeScript** | Excellent | Excellent | Stable | Web only |

---

## 6. Anti-Patterns

### 6.1 Language Selection Anti-Patterns

| Anti-Pattern | Why Bad | Correct Approach |
|--------------|---------|-----------------|
| "Python is faster to write" | Technical debt, performance | Use Go (fast to write + fast runtime) |
| "We'll rewrite in Rust later" | Rarely happens | Start with correct language |
| "C# is fine for Windows" | Lock-in | Use Go + CGO if needed |
| "Everyone knows Python" | Maintenance burden | Invest in Go training |
| "Zig is trendy" | Ecosystem immaturity | Use only for C interop |

### 6.2 Justification Required Triggers

**Document justification when:**
1. Adding Python beyond ML integration
2. Using C# for anything beyond Windows tooling
3. Using TypeScript for backend services
4. Using Swift/Kotlin beyond companion apps
5. Adding any new language not in Tier 1

---

## 7. Migration Paths

### 7.1 Python → Go

| Python | Go Equivalent |
|--------|--------------|
| FastAPI | net/http + chi |
| Celery | NATS JetStream |
| NumPy/Pandas | gonum, dataframe-go |
| PyTorch | Mojo (eventually) |

### 7.2 C# → Go/Rust

| C# | Go | Rust |
|----|----|------|
| ASP.NET Core | net/http | axum |
| Entity Framework | sqlx, GORM | Diesel |
| .NET gRPC | google.golang.org/grpc | tonic |

---

## 8. Enforcement

### 8.1 PR Checklist

```markdown
## Language Policy Compliance

- [ ] New code uses Tier 1 language (Go/Rust/Zig/Mojo)
- [ ] If Tier 2 used, documented justification provided
- [ ] FFI bindings follow established patterns
- [ ] Performance benchmarks attached (if performance-critical)
- [ ] Team skill matrix updated (if new language)
```

### 8.2 Code Review Questions

1. **Why this language?** - Must be justified
2. **Why not Go/Rust/Zig/Mojo?** - Required if Tier 2
3. **FFI necessary?** - Minimize cross-language calls
4. **Performance validated?** - Benchmarks for hot paths

---

## 9. References

- [Go Performance](https://go.dev/doc/articles/debugging_perf_go.pdf)
- [Rust vs Go](https://notehub.org/8rqhy-tpmyi)
- [Zig C Interop](https://ziglang.org/documentation/master/#C-Exporting)
- [Mojo Performance](https://docs.modular.com/mojo/)
- [Language Benchmark Game](https://benchmarksgame-team.pages.debian.net/benchmarksgame/)

---

## Appendix A: Quick Reference Card

```
┌─────────────────────────────────────────────────────────────────┐
│                    LANGUAGE QUICK REFERENCE                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  CORE NVMS:              Go (always)                             │
│                                                                  │
│  PERFORMANCE CRITICAL:    Rust (first) or Zig (C interop)        │
│                                                                  │
│  ML/AI:                  Mojo (first) or Python (fallback)       │
│                                                                  │
│  LOW-LEVEL/EMBEDDED:     Zig (first) or Rust (fallback)         │
│                                                                  │
│  WEB UI:                 TypeScript (always)                     │
│                                                                  │
│  MAC/IOS:                Swift (only)                           │
│                                                                  │
│  ANDROID:                Kotlin (only)                          │
│                                                                  │
│  .NET ECOSYSTEM:         C# (only if .NET required)             │
│                                                                  │
│  QUICK SCRIPTS:          Go (never Python for scripts)          │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

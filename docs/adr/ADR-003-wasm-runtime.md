# ADR-003: WASM Runtime Selection

## Status

Accepted

## Context

NanoVMS needs to support WebAssembly as a Tier 5 isolation mechanism for lightweight sandboxing of untrusted code. We evaluated multiple WASM runtimes for production use.

## Decision Drivers

- Cold start latency (<1ms target)
- Security and sandboxing capabilities
- WASI support
- Production maturity
- Language bindings (Go)
- Memory efficiency

## Options Considered

### Option 1: Wasmtime (Selected)

**Pros:**
- Production-grade (used by Fastly, Shopify)
- Cranelift JIT compilation
- Excellent cold start (<1ms)
- Full WASI support
- Bytecode Alliance project
- Well-maintained

**Cons:**
- Larger binary size (~10MB)
- Rust-based (harder to extend in Go)

### Option 2: WAMR (WebAssembly Micro Runtime)

**Pros:**
- Designed for embedded/IoT
- Very small footprint
- AOT and interpreter modes
- Intel-maintained

**Cons:**
- Less production-tested
- Smaller community
- Limited WASI support

### Option 3: Wasmer

**Pros:**
- Universal runtime
- Multiple compiler backends
- Good Go bindings

**Cons:**
- Larger overhead
- Less focused on serverless

### Option 4: WasmEdge

**Pros:**
- Cloud-native focused
- TensorFlow support
- Good for AI workloads

**Cons:**
- Larger footprint
- Less mature

## Decision

We select **Wasmtime** as the primary WASM runtime for the following reasons:

1. Production proven at scale (Fastly Compute, Shopify)
2. Best cold start performance (<1ms)
3. Full WASI support for system interfaces
4. Active development by Bytecode Alliance
5. Security-first design

We will also support WAMR as an alternative for memory-constrained environments.

## Consequences

### Positive
- Fastest cold start of any runtime
- Production-grade security
- Ecosystem support from major vendors

### Negative
- Larger binary size
- Rust-based (harder to extend)

## Implementation

```go
type WasmRuntime struct {
    engine  *wasmtime.Engine
    store   *wasmtime.Store
    linker  *wasmtime.Linker
}

func (r *WasmRuntime) Instantiate(ctx context.Context, wasm []byte) (*Instance, error) {
    module, err := wasmtime.NewModule(r.engine, wasm)
    if err != nil {
        return nil, err
    }

    instance, err := r.linker.Instantiate(r.store, module)
    if err != nil {
        return nil, err
    }

    return &Instance{instance: instance}, nil
}
```

## References

- [Wasmtime](https://github.com/bytecodealliance/wasmtime)
- [WASI Specification](https://github.com/WebAssembly/WASI)
- Fastly's WASM in Production Talk

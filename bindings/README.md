# NVMS Bindings

This directory contains FFI/binding implementations for the unified NVMS project.

## Directory Structure

```
bindings/
├── go-c-export/          # Go C-export layer (build with cgo)
├── rust-ffi/             # Rust FFI bindings to Go library
├── zig/                  # Zig memory allocator module
├── mojo/                 # Mojo ML integration (stub)
└── README.md            # This file
```

## Building

### Go C-Export

```bash
cd go-c-export
go build -buildmode=c-archive -o nvms_core.a .
```

This produces:
- `nvms_core.a` - Static library
- `nvms_core.h` - C header for Rust bindings

### Rust FFI

```bash
cd rust-ffi
cargo build
```

Requires `nvms_core.h` from Go C-export build.

### Zig

```bash
cd zig
zig build
```

## Usage

### From Rust (PhenoCompose Driver)

```rust
use pheno_compose_driver::NvmsDriver;

let driver = NvmsDriver::new()?;
let instance = driver.create_instance(Tier::Wasm, "my-service")?;
instance.start()?;
```

### From Go (Direct)

```go
import "C"

func main() {
    C.nvms_init()
    inst := C.nvms_instance_create(C.NVMS_TIER_WASM, C.CString("my-service"))
    C.nvms_instance_start(inst)
}
```

### From Zig (Memory)

```zig
const nvms = @import("nvms");

pub fn main() void {
    const ptr = nvms.nvms_zig_alloc(1024);
    defer nvms.nvms_zig_free(ptr, 1024);
}
```

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    NVMS BINDINGS ARCHITECTURE                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────┐                                           │
│  │  PhenoCompose   │                                           │
│  │    (Rust)       │                                           │
│  └────────┬────────┘                                           │
│           │                                                      │
│  ┌────────▼────────┐                                           │
│  │ Rust FFI Bindings │                                        │
│  │  (nvms-ffi)     │                                        │
│  └────────┬────────┘                                           │
│           │ CGO                                                 │
│  ┌────────▼────────┐                                           │
│  │  Go C-Export     │                                        │
│  │ (nvms_core.go)  │                                        │
│  └────────┬────────┘                                           │
│           │                                                      │
│  ┌────────▼────────┐                                           │
│  │  NVMS Core (Go)  │                                        │
│  │  WASM/gVisor/FC │                                        │
│  └─────────────────┘                                           │
│                                                                  │
│  ┌─────────────────┐                                           │
│  │  Zig Memory     │◄── Direct C ABI                          │
│  │ (memory.zig)   │                                           │
│  └─────────────────┘                                           │
│                                                                  │
│  ┌─────────────────┐                                           │
│  │  Mojo ML        │◄── Python/NumPy FFI                      │
│  │ (nvms_ml.py)   │                                           │
│  └─────────────────┘                                           │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## Language Policy

See [LANGUAGES.md](../LANGUAGES.md) for the tiered language selection policy.

| Language | Use Case | Status |
|----------|----------|--------|
| Go | Core orchestration | Primary |
| Rust | FFI, performance | Tier 1 |
| Zig | Memory, low-level | Tier 1 |
| Mojo | ML inference | Stub (pending stable) |
| Python | ML fallback | Tier 2 |

## GPU Support

Full GPU acceleration support for:
- **Apple Silicon**: Metal Performance Shaders (MPS) + NEON SIMD
- **NVIDIA**: CUDA + cuDNN + TF32
- **AMD**: ROCm + MIOpen
- **CPU Fallback**: Auto-detection

```python
from nvms_ml import get_gpu_backend, VectorEmbedding

backend = get_gpu_backend()  # auto-detect
emb = VectorEmbedding(dim=768, gpu_backend=backend)
```

## Cross-Platform Build

```bash
# Build all bindings (auto-detect platform)
python3 bindings/build_cross_platform.py

# Platform-specific
python3 bindings/build_cross_platform.py --verbose
```

## TODO

- [x] Complete Go C-export with full NVMS API
- [x] Add more Rust bindings for configuration management
- [x] Implement Zig memory pool with better performance
- [x] Replace Python ML stub with Mojo when stable (pending Mojo stable)
- [x] Add CUDA/ROCm support for GPU acceleration
- [x] Benchmark all bindings (see benchmark script)

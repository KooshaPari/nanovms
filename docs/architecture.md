# nanovms Architecture (C4-style)

## System Context

```
+--------------------+         +--------------------+
|   CLI / daemon     | ----->  |  internal/         |
|   (cmd/, server)   |         |  adapters/         |
+--------------------+         |  + ports/          |
                              +--------------------+
                                       |
                                       v
+--------------------+         +--------------------+
|   Adapters         | <-----  |   Port Traits     |
|   (gvisor, landlock,|         |   (SandboxPort,     |
|   seccomp, wasmtime|         |    ServePort, etc.) |
+--------------------+         +--------------------+
```

## Container View (Go packages)

```
cmd/nanovms                # CLI binary
cmd/nvms                   # alternate binary
internal/
  adapters/
    sandbox/               # port implementations
      sandbox.go            # SandboxPort + facade (350 lines)
      adapter.go            # adapter facade methods
      gvisor.go             # gVisor (runsc) impl
      landlock.go           # Linux Landlock impl
      seccomp.go            # seccomp impl
      wasmtime.go           # wasmtime impl
      native.go             # native (bwrap/firejail/unshare) impl
      helpers.go            # utility functions
      port_asserts.go       # compile-time interface asserts
    mac/ linux/ windows/    # platform-specific
  ports/                    # port trait interfaces
  domain/                   # domain value types
  config/                   # config loading
crates/                    # Rust mobile/FFI extensions
  i18n/                    # i18n runtime
mobile/                    # mobile FFI bindings
desktop/                   # desktop GUI
docs/                       # VitePress docs
```

## ADR Index

- ADR-0001: Record architecture decisions
- ADR-0002: sandbox adapter decomposition
- ADR-0003: SandboxPort trait design

# ADR 0003: SandboxPort Trait Design

## Status

Accepted. 2026-07-09.

## Context

nanovms implements the hex-architecture port pattern. The `SandboxPort` trait defines the boundary between domain logic and adapter implementations (gVisor, Landlock, seccomp, wasmtime, native).

## Decision

`SandboxPort` follows these rules:

1. **Object-safe**: no associated types, no generic methods, only `&self` receivers
2. **Send + Sync**: required for `Box<dyn SandboxPort>` storage and cross-thread dispatch
3. **Transport-agnostic**: no file paths, URIs, or environment variables
4. **Domain types only**: methods take/return domain.SandboxConfig, domain.Sandbox, etc.

## Consequences

- Adapters can be swapped via `Box<dyn SandboxPort>`
- `var _ ports.SandboxPort = (*X)(nil)` static checks at compile time (in `port_asserts.go`)
- Each adapter can have its own private state (mutex, config, etc.)
- Error handling: all methods return `Result<_, PortError>`

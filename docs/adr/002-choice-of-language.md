# ADR 002: Choice of Languages

## Status
Accepted

## Context
NanovMS is a high-performance unikernel and cloud platform. It requires a mix of systems-level performance, robust CLI tooling, and a modern web-based control plane.

## Decision
We will use a polyglot approach: **Go** for the core unikernel tooling and CLI, **Rust** for performance-critical subsystems and the new SDK, and **TypeScript** for the web-based management interface.

- **Go**: Chosen for the primary CLI and many internal packages due to its simplicity, fast compilation, and excellent concurrency primitives. It is the traditional choice for cloud-native tooling.
- **Rust**: Chosen for new subsystems where memory safety and raw performance are paramount, such as high-throughput networking components and the new Rust-based SDK.
- **TypeScript**: Chosen for the web management console and dashboard to provide a responsive, type-safe frontend experience.

## Consequences
- **Pros**:
    - Leveraging the strengths of each language: Go's developer experience, Rust's performance, and TypeScript's web ecosystem.
    - Ability to incrementally migrate components to Rust where performance is critical.
- **Cons**:
    - Maintaining multiple language ecosystems and toolchains.
    - Potential for inconsistency in API styles between Go and Rust interfaces.

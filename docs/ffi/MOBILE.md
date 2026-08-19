# Cross-Platform FFI — Mobile (PR-A)

This document tracks the mobile scaffolding introduced by PR-A
(`feat/nvms-mobile-ffi-bootstrap`). It is a scaffold only — neither crate is
a working mobile app yet. Both targets compile under `cargo check
--workspace` on a non-mobile host because the platform-specific FFI crates
are gated behind Cargo features that are **off** by default.

## Audit context

`nanovms-audit.json` recorded iOS FFI = 0/100 and Android FFI = 0/100 in the
cross-platform FFI category (avg 32/100). The audit's `top_5_ffi_priorities`
flags "Wire swift-rs into nvms-cli for macOS menu-bar AppKit tray" and
"Add ndk + android-activity Android FFI surface" as the two highest-leverage
gaps. This PR scaffolds the minimum viable crate layout to start closing
those gaps.

## What PR-A adds

| Path | Purpose |
|---|---|
| `mobile/macos-tray/` | swift-rs-backed NSStatusBar tray (L121 + L129 surface for macOS) |
| `mobile/android-companion/` | ndk + jni-backed observability companion (L125 surface for Android) |
| `Cargo.toml` (new top-level) | workspace manifest grouping the FFI crates |

The two crates compile under `cargo check --workspace` on linux-x86_64.
Cross-compile (`aarch64-linux-android`, `aarch64-apple-darwin`) is
**documented but not gated in CI** — adding it is the next step (B3 in
PLAN.md §2.5.2) once the cross-compile matrix is wired up.

## Feature flags

| Crate | Feature | Default | Meaning |
|---|---|---|---|
| `nvms-macos-tray` | `native-bridge` | OFF | Links swift-rs; requires `swiftc` |
| `nvms-android-companion` | `jni-bridge` | OFF | Links ndk + jni; requires the Android NDK |

Both crates expose a `*_STATUS_STUB()` function whose return value reports
whether the corresponding feature flag is active. Downstream `nvms-cli`
subcommands can pattern-match on this without taking a hard dependency on
AppKit or the Android platform headers.

## Out of scope (deliberately deferred)

- Real `NSStatusItem` registration in Swift (needs `swiftc` in CI).
- Real `extern "system" fn Java_io_kooshapari_nanovms_companion_*` symbols
  (needs Android NDK in CI).
- Cross-compile matrix in `.github/workflows/ci.yml` (follow-up PR).
- Re-export through `sdk/rust/` (sdk/rust/ is out of scope for this PR
  per the constraints).

## Acceptance

- [x] `cargo check --workspace --all-targets` passes on linux-x86_64 (host).
- [x] Cross-compile target is documented here but not gated in CI.
- [x] No vendor source trees added.

Refs: nanovms-audit.json#L121-L130 (FFI category)
// SPDX-License-Identifier: MIT OR Apache-2.0
// Copyright (c) 2026 Phenotype org (heliosCLI)

//! `ffi_utils` — small FFI-friendly helpers shared across Phenotype crates.
//!
//! Currently exposes [`FfiMutex`], a thin alias for [`parking_lot::Mutex`]
//! used by `helios_cli::harness_pyo3` for a process-wide Python cache.
//! Kept as a separate crate so FFI consumers don't need to depend on
//! `parking_lot` directly when they're already using PyO3 / etc.

#![deny(unsafe_code)]
#![warn(missing_docs)]

/// FFI-friendly `Mutex` alias. Backed by `parking_lot::Mutex` for non-blocking
/// access patterns that are safe across `cdylib` FFI boundaries.
pub type FfiMutex<T> = parking_lot::Mutex<T>;

/// Re-export of `parking_lot::MutexGuard` for downstream use.
pub use parking_lot::MutexGuard;

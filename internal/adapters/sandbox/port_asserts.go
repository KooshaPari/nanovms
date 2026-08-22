// SPDX-License-Identifier: MIT OR Apache-2.0
//! Compile-time interface assertions.
//!
//! This file exists solely to ensure at compile time that the various
//! adapter implementations in this package satisfy the [`ports.SandboxPort`]
//! interface contract. Each `var _ ports.SandboxPort = (*X)(nil)` line
//! would fail to compile if `X` did not implement the trait.
//!
//! This is a zero-cost static check: the variables are never read at runtime.

package sandbox

import "github.com/kooshapari/nanovms/internal/ports"

var _ ports.SandboxPort = (*Adapter)(nil)
var _ ports.SandboxPort = (*landlockAdapter)(nil)
var _ ports.SandboxPort = (*seccompAdapter)(nil)
var _ ports.SandboxPort = (*wasmtimeAdapter)(nil)
var _ ports.SandboxPort = (*nativeSandboxAdapter)(nil)

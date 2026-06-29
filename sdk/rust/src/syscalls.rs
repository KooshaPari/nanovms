// SPDX-License-Identifier: MIT OR Apache-2.0
//! Generated FFI bindings for the NanoVMS syscall ABI.

use std::ffi::c_char;
use std::os::raw::{c_int, c_uint};

#[allow(non_camel_case_types, non_snake_case, non_upper_case_globals, clippy::all)]
mod bindings {
    use super::{c_char, c_int, c_uint};
    include!("generated/nvms_syscalls_bindings.rs");
}

pub use bindings::{nvms_exec, nvms_exec_args, nvms_exit, nvms_focus};

/// Convenience constructor for the generated exec argument payload.
pub fn exec_args(path: *const c_char, argv: *const *const c_char, envp: *const *const c_char) -> nvms_exec_args {
    nvms_exec_args { path, argv, envp }
}

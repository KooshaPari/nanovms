// SPDX-License-Identifier: MIT OR Apache-2.0
use std::path::Path;
use std::process::Command;

fn main() {
    let header = "include/nvms_syscalls.h";
    let generated = "src/generated/nvms_syscalls_bindings.rs";
    println!("cargo:rerun-if-changed={header}");
    println!("cargo:rerun-if-changed={generated}");

    if Path::new(generated).exists() {
        return;
    }

    let status = Command::new("bindgen")
        .args([
            header,
            "--allowlist-function",
            "nvms_(focus|exit|exec)",
            "--allowlist-type",
            "nvms_exec_args",
            "--output",
            generated,
        ])
        .status();

    match status {
        Ok(status) if status.success() => {}
        Ok(status) => panic!("bindgen failed with status {status} and no checked-in bindings"),
        Err(error) => panic!("bindgen unavailable and no checked-in bindings: {error}"),
    }
}

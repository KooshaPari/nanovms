// SPDX-License-Identifier: MIT OR Apache-2.0
//
// nvms-macos-tray — build.rs
//
// The full implementation will shell out to `swiftc` to compile
// `swift/NvmsTray.swift` into a static archive that swift-rs links.
// For now this stub only declares the env vars downstream crates will need.

fn main() {
    // swift-rs embeds the Swift sources via `SwiftLinker::new(...)` from the
    // caller's `build.rs`. We only print rerun-if-changed hints so the build
    // is reproducible without a Swift toolchain.
    println!("cargo:rerun-if-changed=swift/NvmsTray.swift");
    println!("cargo:rerun-if-changed=Cargo.toml");

    // Surface the macOS deployment target to downstream crates that may want
    // to match it (e.g. an ObjC shim around Security.framework).
    println!("cargo:rustc-env=MACOS_DEPLOYMENT_TARGET=12.0");

    // When the `native-bridge` feature is enabled, this is where
    // swift_rs::SwiftLinker would compile the Swift sources:
    //
    //   #[cfg(feature = "native-bridge")]
    //   swift_rs::SwiftLinker::new("12.0")
    //       .with_bridge("swift/NvmsTray.swift")
    //       .link();
    //
    // The stub is gated out so linux/windows CI does not require `swiftc`.
}
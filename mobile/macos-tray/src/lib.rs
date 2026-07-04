// SPDX-License-Identifier: MIT OR Apache-2.0
//
// nvms-macos-tray — stub lib.rs
//
// Scaffold for the macOS menu-bar tray backing `nvms-cli`'s `tray install` and
// `tray status` subcommands. The real implementation will:
//
//   1. Use swift-rs to register an NSStatusBar item backed by AppKit
//      (NSStatusBar.system.statusItem(withLength:)).
//   2. Wire menu actions to nvms-cli subcommands via an embedded gRPC client
//      (see sdk/rust/src/client.rs).
//   3. Persist a LaunchAgent at ~/Library/LaunchAgents/io.kooshapari.nanovms.plist
//      so the tray auto-starts at user login.
//   4. Read API keys from Keychain via Security.framework (resolved in a future PR).
//
// For now this scaffold only verifies that the feature-flag plumbing compiles
// on a non-macOS host so `cargo check --workspace` stays green in CI.

#![cfg_attr(docsrs, feature(doc_cfg))]

#[cfg(target_os = "macos")]
pub const TRAY_DOMAIN: &str = "io.kooshapari.nanovms.tray";

#[cfg(target_os = "macos")]
pub const LAUNCH_AGENT_LABEL: &str = "io.kooshapari.nanovms.tray";

/// Compile-time marker so a downstream crate can detect that the swift-rs
/// feature is wired in without taking a hard dependency on AppKit.
#[cfg(all(target_os = "macos", feature = "native-bridge"))]
pub const NATIVE_BRIDGE_ENABLED: bool = true;

#[cfg(not(all(target_os = "macos", feature = "native-bridge")))]
pub const NATIVE_BRIDGE_ENABLED: bool = false;

/// Minimum macOS deployment target — kept here so the Swift build script
/// (see `build.rs`) can read it via `cargo:rustc-env`.
#[cfg(target_os = "macos")]
pub const MACOS_DEPLOYMENT_TARGET: &str = "12.0";

/// Stub entry point — the real implementation will return a Swift-backed
/// `TrayHandle` from `swift_rs::SwiftObj`. Until then this returns a
/// placeholder so consumers can pattern-match on the feature flag.
pub fn tray_status_stub() -> &'static str {
    if NATIVE_BRIDGE_ENABLED {
        "[nvms-macos-tray] native AppKit bridge ENABLED (feature flag set)"
    } else {
        "[nvms-macos-tray] stub mode (no native-bridge feature — host or build did not opt in)"
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn tray_status_stub_is_stable() {
        let s = tray_status_stub();
        assert!(s.starts_with("[nvms-macos-tray]"));
    }

    #[test]
    fn native_bridge_flag_is_false_on_linux() {
        if cfg!(not(target_os = "macos")) {
            assert!(!NATIVE_BRIDGE_ENABLED);
        }
    }
}
// SPDX-License-Identifier: MIT OR Apache-2.0
//
// nvms-macos-tray — Swift source stub.
//
// This file is the placeholder for the real NSStatusBar-driven menu-bar tray.
// The full implementation will:
//
//   - Create an NSStatusItem in the system menu bar with a tier icon (WASM /
//     gVisor / Firecracker) chosen at runtime from a base64-encoded PNG.
//   - Build an NSMenu with: Open Dashboard, View Logs, Pause / Resume, Quit.
//   - Forward clicks back to Rust via swift-rs callbacks (`@_cdecl` shims).
//
// swift-rs generates a `NvmsTrayBridge.h` from this file at build time when
// `cargo build -p nvms-macos-tray --features native-bridge` is run on macOS.
//
// Until then this stub documents the public surface so reviewers can reason
// about the FFI boundary without a Swift toolchain.

import Foundation
#if canImport(AppKit)
import AppKit
#endif

/// Swift-side configuration mirror of `nvms_macos_tray::TRAY_DOMAIN`.
public enum NvmsTrayConfig {
    public static let domain: String = "io.kooshapari.nanovms.tray"
    public static let launchAgentLabel: String = "io.kooshapari.nanovms.tray"
}

/// Stub — the real implementation returns an NSStatusItem and keeps a strong
/// reference alive for the lifetime of the tray process.
@_cdecl("nvms_tray_install_stub")
public func nvms_tray_install_stub() -> Int32 {
    #if canImport(AppKit)
    // Real code would do:
    //   let item = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)
    //   item.button?.title = "nvms"
    //   item.menu = NvmsTrayMenuBuilder.build()
    //   return Unmanaged.passUnretained(item).toOpaque().hashValue
    return 0
    #else
    return -1
    #endif
}
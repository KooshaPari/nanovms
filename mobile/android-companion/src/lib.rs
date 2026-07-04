// SPDX-License-Identifier: MIT OR Apache-2.0
//
// nvms-android-companion — stub lib.rs
//
// Scaffold for the Android observability companion. The real implementation
// will expose a JNI surface that a Kotlin `MainActivity` can call to:
//
//   1. Stream per-tier VM telemetry (WASM / gVisor / Firecracker) into the
//      Android `Notification` channel at low frequency (≤ 1 Hz).
//   2. Subscribe to `nvms-cli` deploy events via a local Unix domain socket
//      (resolved in a future PR once the desktop tray stabilises).
//   3. Push launch intents back to nvms-cli via `am start`.
//
// For now this stub verifies the feature-flag plumbing and the JNI shim
// signatures are syntactically valid. The `jni-bridge` feature is OFF by
// default so `cargo check --workspace` on a linux/windows/macos host stays
// green without the Android NDK installed.

#![cfg_attr(docsrs, feature(doc_cfg))]

mod jni_bridge;

pub use jni_bridge::nvms_android_companion_stub;

/// Java/Kotlin class that hosts our JNI entry points. Matches the
/// `applicationId` + class name declared in `aar/AndroidManifest.xml`.
pub const COMPANION_CLASS: &str = "io.kooshapari.nanovms.companion.MainActivity";

/// Compile-time marker so a downstream consumer can detect that the ndk
/// feature is wired in without taking a hard dependency on the Android
/// platform headers.
#[cfg(all(target_os = "android", feature = "jni-bridge"))]
pub const JNI_BRIDGE_ENABLED: bool = true;

#[cfg(not(all(target_os = "android", feature = "jni-bridge")))]
pub const JNI_BRIDGE_ENABLED: bool = false;

/// Stub entry point — the real implementation will be a `#[no_mangle]`
/// `extern "system" fn Java_io_kooshapari_nanovms_companion_MainActivity_*`
/// pair that uses `jni::JNIEnv` to marshal telemetry into a Kotlin
/// `Flow<List<VmStatus>>`.
pub fn companion_status_stub() -> &'static str {
    if JNI_BRIDGE_ENABLED {
        "[nvms-android-companion] JNI bridge ENABLED (ndk + jni crates linked)"
    } else {
        "[nvms-android-companion] stub mode (no jni-bridge feature — host or build did not opt in)"
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn companion_status_stub_is_stable() {
        let s = companion_status_stub();
        assert!(s.starts_with("[nvms-android-companion]"));
    }

    #[test]
    fn jni_bridge_flag_is_false_off_android() {
        if cfg!(not(target_os = "android")) {
            assert!(!JNI_BRIDGE_ENABLED);
        }
    }
}
// SPDX-License-Identifier: MIT OR Apache-2.0
//
// nvms-android-companion — JNI bridge stub.
//
// The full implementation will be gated on `#[cfg(all(target_os = "android",
// feature = "jni-bridge"))]` and expose:
//
//   - `Java_io_kooshapari_nanovms_companion_MainActivity_nativeInit`
//   - `Java_io_kooshapari_nanovms_companion_MainActivity_pollTelemetry`
//   - `Java_io_kooshapari_nanovms_companion_MainActivity_subscribeDeployEvents`
//
// We deliberately do NOT pull `jni::JNIEnv` here even when the feature is
// enabled — the stub is shipped without the feature so `cargo check` on a
// non-Android host can verify the public Rust surface compiles. The real
// `extern "system"` symbols will be added in a follow-up PR once the JNI
// contract is agreed with the Kotlin side.

#![cfg_attr(docsrs, feature(doc_cfg))]

/// Stub function whose symbol will be exported on Android targets. We use
/// `extern "C"` rather than `extern "system"` here so the symbol remains
/// callable from a non-Android Rust unit test (which is what runs in CI).
#[no_mangle]
pub extern "C" fn nvms_android_companion_stub() -> i32 {
    0
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn stub_returns_zero() {
        assert_eq!(nvms_android_companion_stub(), 0);
    }
}
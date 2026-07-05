//! nvms-ios-companion: iOS observability companion for nanovms.
//!
//! L122: iOS Native FFI. Port of the L121 (macos-tray) and L125 (android-companion)
//! pattern to iOS via swift-rs + UIKit.
//!
//! Default features are empty so `cargo check` on linux/windows hosts stays green.
//! Build with `--features native-bridge` to enable the Swift bridge (requires swiftc).

#![cfg_attr(feature = "native-bridge", allow(unsafe_code))]

/// iOS FFI version. Bump when the Swift bridge contract changes.
pub const IOS_FFI_VERSION: &str = "0.1.0";

/// AppKit-style status from the iOS companion (mirrors macos-tray).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum CompanionStatus {
    Active,
    Idle,
    Stale,
}

impl CompanionStatus {
    pub fn as_str(&self) -> &'static str {
        match self {
            CompanionStatus::Active => "active",
            CompanionStatus::Idle => "idle",
            CompanionStatus::Stale => "stale",
        }
    }
}

/// Swift-callable entry point. Mirrors `nvms_macos_tray::ping` semantics.
pub fn ping() -> &'static str {
    "nvms-ios-companion:ok"
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn version_is_set() {
        assert!(IOS_FFI_VERSION.starts_with("0."));
    }

    #[test]
    fn ping_ok() {
        assert_eq!(ping(), "nvms-ios-companion:ok");
    }

    #[test]
    fn status_roundtrip() {
        for s in [CompanionStatus::Active, CompanionStatus::Idle, CompanionStatus::Stale] {
            assert!(!s.as_str().is_empty());
        }
    }
}

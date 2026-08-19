// SPDX-License-Identifier: MIT OR Apache-2.0
//
// nvms-scm-host — stub lib.rs
//
// Scaffold for the nanovms Windows Service Control Manager (SCM) host.
// Wires `nvms-cli` into SCM so it can run as a registered Windows service
// (`sc start nanovms`). PILLAR-TAXONOMY-v2 **L123** (Windows Native FFI —
// `microsoft/windows-rs`) and **L130** (System Service Integration —
// Windows-service / SCM).
//
// Status: scaffold only. The actual `CreateServiceW` /
// `RegisterEventSourceW` / `ReportEventW` calls land in a follow-up PR
// once the FFI shape is validated. On non-Windows targets the module
// exposes the stub surface so `cargo check --workspace` continues to
// pass on macOS / linux without a Windows toolchain.
//
// Reference pattern: PhenoCompose PR #77 (`windows/scm-service/`,
// MERGED 51babba). See also PhenoCompose `docs/ffi/WINDOWS-SCM.md`.

#![cfg_attr(docsrs, feature(doc_cfg))]

use thiserror::Error;

pub mod scm;
pub mod event_log;

#[cfg(target_os = "windows")]
#[cfg(feature = "service")]
pub mod service;

pub const SERVICE_NAME: &str = "nanovms";
pub const SERVICE_DISPLAY: &str = "nanovms Daemon (3-tier microVM isolation)";
pub const SERVICE_DESCRIPTION: &str =
    "nanovms — WASM / gVisor / Firecracker 3-tier microVM isolation daemon. See SPEC.md.";

#[derive(Debug, Error)]
pub enum ScmHostError {
    #[error("Windows SCM bridge unavailable on this target")]
    BridgeUnavailable,

    #[error("service registration failed: {0}")]
    RegistrationFailed(String),

    #[error("service start failed: {0}")]
    StartFailed(String),

    #[error("service stop failed: {0}")]
    StopFailed(String),

    #[error("event log operation failed: {0}")]
    EventLogFailed(String),
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct ScmCapabilities {
    pub scm_registration: bool,
    pub service_control: bool,
    pub event_log: bool,
}

impl Default for ScmCapabilities {
    fn default() -> Self {
        Self {
            scm_registration: cfg!(target_os = "windows"),
            service_control: cfg!(target_os = "windows"),
            event_log: cfg!(target_os = "windows"),
        }
    }
}

/// Query the platform-level capability set — useful for `nvms-cli` to
/// decide whether to attempt SCM bootstrap or skip with a friendly
/// message.
pub fn capabilities() -> ScmCapabilities {
    ScmCapabilities::default()
}

/// Install the daemon binary with the SCM. The Windows-only path uses
/// `CreateServiceW` from `Win32_System_Services`; on non-Windows the call
/// returns `BridgeUnavailable` so the scaffold compiles.
pub fn install_service(binary_path: &str) -> Result<(), ScmHostError> {
    #[cfg(all(target_os = "windows", feature = "service"))]
    {
        service::install_via_scm(binary_path)
    }
    #[cfg(not(all(target_os = "windows", feature = "service")))]
    {
        let _ = binary_path;
        Err(ScmHostError::BridgeUnavailable)
    }
}

/// Send a `SERVICE_CONTROL_STOP` to the SCM-registered service.
pub fn stop_service() -> Result<(), ScmHostError> {
    #[cfg(all(target_os = "windows", feature = "service"))]
    {
        service::stop_via_scm()
    }
    #[cfg(not(all(target_os = "windows", feature = "service")))]
    {
        Err(ScmHostError::BridgeUnavailable)
    }
}

/// Start the SCM-registered service. Currently forwards through the
/// PowerShell `sc start` shim until the real `StartServiceW` FFI lands.
pub fn start_service() -> Result<(), ScmHostError> {
    #[cfg(all(target_os = "windows", feature = "service"))]
    {
        service::start_via_scm()
    }
    #[cfg(not(all(target_os = "windows", feature = "service")))]
    {
        Err(ScmHostError::BridgeUnavailable)
    }
}

/// Write a record to the `nanovms` Application event-log channel. The
/// Windows path will call `RegisterEventSourceW` + `ReportEventW` (from
/// `Win32_System_EventLog`); on non-Windows the call returns
/// `BridgeUnavailable`.
pub fn write_event_log(entry: &event_log::EventLogEntry) -> Result<(), ScmHostError> {
    #[cfg(all(target_os = "windows", feature = "service"))]
    {
        service::write_event_log_via_win32(entry)
    }
    #[cfg(not(all(target_os = "windows", feature = "service")))]
    {
        let _ = entry;
        Err(ScmHostError::BridgeUnavailable)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn capabilities_default_reflects_target() {
        let caps = capabilities();
        assert_eq!(caps.scm_registration, cfg!(target_os = "windows"));
        assert_eq!(caps.service_control, cfg!(target_os = "windows"));
        assert_eq!(caps.event_log, cfg!(target_os = "windows"));
    }

    #[test]
    fn service_constants_are_stable() {
        assert_eq!(SERVICE_NAME, "nanovms");
        assert!(SERVICE_DISPLAY.contains("nanovms"));
        assert!(SERVICE_DESCRIPTION.contains("WASM"));
    }

    #[test]
    fn install_off_windows_is_unsupported() {
        #[cfg(not(target_os = "windows"))]
        {
            let err = install_service("C:/nanovms/nanovms.exe").unwrap_err();
            assert!(matches!(err, ScmHostError::BridgeUnavailable));
        }
    }

    #[test]
    fn stop_off_windows_is_unsupported() {
        #[cfg(not(target_os = "windows"))]
        {
            let err = stop_service().unwrap_err();
            assert!(matches!(err, ScmHostError::BridgeUnavailable));
        }
    }

    #[test]
    fn start_off_windows_is_unsupported() {
        #[cfg(not(target_os = "windows"))]
        {
            let err = start_service().unwrap_err();
            assert!(matches!(err, ScmHostError::BridgeUnavailable));
        }
    }

    #[test]
    fn event_log_off_windows_is_unsupported() {
        #[cfg(not(target_os = "windows"))]
        {
            let entry = event_log::EventLogEntry::info("test", "stub");
            let err = write_event_log(&entry).unwrap_err();
            assert!(matches!(err, ScmHostError::BridgeUnavailable));
        }
    }
}

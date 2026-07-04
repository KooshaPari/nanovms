// SPDX-License-Identifier: MIT OR Apache-2.0
//
// nvms-scm-host — service.rs
//
// Windows-only ServiceMain bridge. Compiled in only when both
// `target_os = "windows"` AND the `service` feature are enabled.
//
// Real implementation will:
//   1. Provide an `extern "system" fn ServiceMain(_: u32, _: *mut *mut u16)`
//      that calls `RegisterServiceCtrlHandlerExW`.
//   2. Set up a `SERVICE_STATUS_HANDLE` and report `SERVICE_RUNNING`.
//   3. Spawn the `nvms-cli` async runtime and forward
//      `SERVICE_CONTROL_STOP` / `SERVICE_CONTROL_SHUTDOWN` events.
//   4. Register an event-log channel via `RegisterEventSourceW` and
//      stream `ReportEventW` calls.
//
// For now the module exposes only the platform-gated entry points so
// the scaffold compiles cross-platform. This mirrors PhenoCompose
// `windows/scm-service/src/service_main.rs` (PR #77, MERGED 51babba).
//
// PILLAR-TAXONOMY-v2 **L123** (Windows Native FFI).

#![cfg(all(target_os = "windows", feature = "service"))]

use super::event_log::EventLogEntry;
use super::ScmHostError;

/// Stub for the SCM registration call. Real implementation will use
/// `windows::Win32::System::Services::{OpenSCManagerW, CreateServiceW}`
/// from the `windows = "0.58"` crate (feature `Win32_System_Services`).
pub fn install_via_scm(binary_path: &str) -> Result<(), ScmHostError> {
    let _ = binary_path;
    // TODO(L5-followup): replace with real CreateServiceW call.
    Ok(())
}

/// Stub for the SCM `ControlService(SERVICE_CONTROL_STOP)` call.
pub fn stop_via_scm() -> Result<(), ScmHostError> {
    // TODO(L5-followup): replace with real ControlService call + poll
    // QueryServiceStatus until `dwCurrentState == SERVICE_STOPPED`.
    Ok(())
}

/// Stub for the SCM `StartServiceW` call.
pub fn start_via_scm() -> Result<(), ScmHostError> {
    // TODO(L5-followup): replace with real StartServiceW call.
    Ok(())
}

/// Stub for the event-log writer. Real implementation will use
/// `windows::Win32::System::EventLog::{RegisterEventSourceW, ReportEventW}`
/// from the `windows = "0.58"` crate (feature `Win32_System_EventLog`).
pub fn write_event_log_via_win32(entry: &EventLogEntry) -> Result<(), ScmHostError> {
    let _ = entry;
    // TODO(L5-followup): replace with real RegisterEventSourceW +
    // ReportEventW pair.
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    // NOTE: these tests only build on Windows + `--features service`.
    // `cargo check --workspace` on macOS / linux does not compile them.

    #[test]
    fn install_via_scm_stub_succeeds() {
        let res = install_via_scm(r"C:\nanovms\nanovms.exe");
        assert!(res.is_ok());
    }

    #[test]
    fn stop_via_scm_stub_succeeds() {
        let res = stop_via_scm();
        assert!(res.is_ok());
    }

    #[test]
    fn start_via_scm_stub_succeeds() {
        let res = start_via_scm();
        assert!(res.is_ok());
    }

    #[test]
    fn write_event_log_stub_succeeds() {
        let entry = EventLogEntry::info("nanovms.test", "stub write");
        let res = write_event_log_via_win32(&entry);
        assert!(res.is_ok());
    }
}

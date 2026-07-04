// SPDX-License-Identifier: MIT OR Apache-2.0
//
// nvms-scm-host — event_log.rs
//
// Cross-platform surface that exposes an `EventLogEntry` value type
// and the Win32 severity constants from `Win32_System_EventLog`. The
// actual `RegisterEventSourceW` / `ReportEventW` FFI calls land in
// `service::write_event_log_via_win32` (Windows-only, follow-up PR).
//
// PILLAR-TAXONOMY-v2 **L123** (Windows Native FFI — `microsoft/windows-rs`)
// and **L130** (System Service Integration — Windows-service / SCM).
//
// Reference: PhenoCompose `windows/scm-service/docs/WINDOWS-SCM.md`
// "Event-log channel (event-log.mc)" section.

/// Event-log severity. Mirrors the values from
/// `Win32_System_EventLog::EVENTLOG_*_TYPE` in `windows = "0.58"`.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Severity {
    Success,
    Information,
    Warning,
    Error,
}

impl Severity {
    /// Map to the Win32 event-type bit constant. Useful for the
    /// `ReportEventW` call that lands in `service::write_event_log_via_win32`.
    pub const fn win32_type(self) -> u16 {
        match self {
            // EVENTLOG_SUCCESS = 0x0000
            Severity::Success => 0x0000,
            // EVENTLOG_EVENT_TYPE_SUCCESS (we re-use the success-word for clarity)
            // EVENTLOG_INFORMATION_TYPE = 0x0004
            Severity::Information => 0x0004,
            // EVENTLOG_WARNING_TYPE = 0x0002
            Severity::Warning => 0x0002,
            // EVENTLOG_ERROR_TYPE = 0x0001
            Severity::Error => 0x0001,
        }
    }

    /// Lower-case ASCII label used by `eventcreate.exe` and PowerShell
    /// `New-EventLog` / `Write-EventLog` cmdlets.
    pub const fn as_str(self) -> &'static str {
        match self {
            Severity::Success => "success",
            Severity::Information => "information",
            Severity::Warning => "warning",
            Severity::Error => "error",
        }
    }
}

/// A single event-log row destined for the `nanovms` Application
/// channel. Stays purely a value type (no `unsafe`) so it can be
/// constructed from any thread / context.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct EventLogEntry {
    pub severity: Severity,
    pub category: &'static str,
    pub message: String,
}

impl EventLogEntry {
    /// Construct an `Information`-severity entry.
    pub fn info(category: &'static str, message: impl Into<String>) -> Self {
        Self {
            severity: Severity::Information,
            category,
            message: message.into(),
        }
    }

    /// Construct a `Warning`-severity entry.
    pub fn warning(category: &'static str, message: impl Into<String>) -> Self {
        Self {
            severity: Severity::Warning,
            category,
            message: message.into(),
        }
    }

    /// Construct an `Error`-severity entry.
    pub fn error(category: &'static str, message: impl Into<String>) -> Self {
        Self {
            severity: Severity::Error,
            category,
            message: message.into(),
        }
    }
}

/// Compose the `eventcreate.exe` command for an operator-driven write.
/// Useful for the PowerShell install script. Real implementation will
/// call `RegisterEventSourceW` + `ReportEventW` (from
/// `Win32_System_EventLog`) on every service-mode event.
pub fn compose_eventcreate_command(entry: &EventLogEntry) -> String {
    format!(
        "eventcreate /L Application /SO {source} /T {ty} /ID 100 /D \"{msg}\"",
        source = super::SERVICE_NAME,
        ty = entry.severity.as_str(),
        msg = entry.message.replace('"', "'"),
    )
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn severity_win32_constants_match_win32_docs() {
        // EVENTLOG_INFORMATION_TYPE = 0x0004
        assert_eq!(Severity::Information.win32_type(), 0x0004);
        // EVENTLOG_WARNING_TYPE = 0x0002
        assert_eq!(Severity::Warning.win32_type(), 0x0002);
        // EVENTLOG_ERROR_TYPE = 0x0001
        assert_eq!(Severity::Error.win32_type(), 0x0001);
    }

    #[test]
    fn severity_as_str_matches_eventcreate_syntax() {
        assert_eq!(Severity::Information.as_str(), "information");
        assert_eq!(Severity::Warning.as_str(), "warning");
        assert_eq!(Severity::Error.as_str(), "error");
    }

    #[test]
    fn event_log_entry_constructors_set_severity() {
        let i = EventLogEntry::info("nanovms.boot", "started");
        assert_eq!(i.severity, Severity::Information);
        assert_eq!(i.category, "nanovms.boot");
        assert_eq!(i.message, "started");

        let w = EventLogEntry::warning("nanovms.tier", "gVisor degraded");
        assert_eq!(w.severity, Severity::Warning);

        let e = EventLogEntry::error("nanovms.fatal", "firecracker shutdown");
        assert_eq!(e.severity, Severity::Error);
    }

    #[test]
    fn compose_eventcreate_command_uses_eventcreate_syntax() {
        let entry = EventLogEntry::info("nanovms.test", "hello world");
        let cmd = compose_eventcreate_command(&entry);
        assert!(cmd.contains("/L Application"));
        assert!(cmd.contains("/SO nanovms"));
        assert!(cmd.contains("/T information"));
        assert!(cmd.contains("/ID 100"));
        assert!(cmd.contains("hello world"));
    }

    #[test]
    fn compose_eventcreate_command_sanitises_double_quotes() {
        let entry = EventLogEntry::warning("nanovms.test", r#"path "C:\nvms" missing"#);
        let cmd = compose_eventcreate_command(&entry);
        // Embedded double-quotes must be replaced with single-quotes so the
        // command line parses correctly under cmd.exe.
        assert!(!cmd.contains(r#"path "C:\nvms" missing"#));
        assert!(cmd.contains(r#"path 'C:\nvms' missing"#));
    }
}

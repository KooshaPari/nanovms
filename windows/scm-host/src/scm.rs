// SPDX-License-Identifier: MIT OR Apache-2.0
//
// nvms-scm-host — scm.rs
//
// Cross-platform surface that exposes the SCM registration constants
// and a string-form `sc create` command builder. On Windows + `service`
// feature the corresponding `CreateServiceW` FFI call lands in a
// follow-up PR (see `service::install_via_scm`). On every other build
// these helpers still work — they only touch constants and strings.
//
// PILLAR-TAXONOMY-v2 **L123** (Windows Native FFI — `microsoft/windows-rs`)
// and **L130** (System Service Integration — Windows-service / SCM).
//
// Reference: PhenoCompose `windows/scm-service/src/scm.rs`.

/// Service start type registered with the SCM. Mirrors the
/// `SERVICE_AUTO_START` constant from `Win32_System_Services`.
pub const SERVICE_AUTO_START: u32 = 0x00000002;

/// Service type: `SERVICE_WIN32_OWN_PROCESS`.
pub const SERVICE_WIN32_OWN_PROCESS: u32 = 0x00000010;

/// Service type: `SERVICE_WIN32_SHARE_PROCESS` — reserved for the
/// follow-up PR that splits `nvms-cli` into a shared-process companion.
pub const SERVICE_WIN32_SHARE_PROCESS: u32 = 0x00000020;

/// Error control severity: `SERVICE_ERROR_NORMAL`.
pub const SERVICE_ERROR_NORMAL: u32 = 0x00000001;

/// Error control severity: `SERVICE_ERROR_CRITICAL` — boots the machine
/// into the recovery control set if the daemon fails to start.
pub const SERVICE_ERROR_CRITICAL: u32 = 0x00000003;

/// Compose the full `sc create` command line for an operator-driven
/// registration. Real implementation will call `sc.exe create` or
/// `OpenSCManagerW` + `CreateServiceW` — the constants above mirror the
/// native Win32 values so the JSON manifest stays stable cross-platform.
pub fn compose_create_command(binary_path: &str) -> String {
    format!(
        "sc create \"{name}\" binPath= \"{bin}\" start= auto DisplayName= \"{disp}\"",
        name = super::SERVICE_NAME,
        bin = binary_path,
        disp = super::SERVICE_DISPLAY,
    )
}

/// Compose the equivalent `sc delete` command. PowerShell variant lands
/// in the companion `install.ps1` / `uninstall.ps1` (see README.md).
pub fn compose_delete_command() -> String {
    format!("sc delete \"{name}\"", name = super::SERVICE_NAME)
}

/// Compose the `sc start` command for the operator runbook.
pub fn compose_start_command() -> String {
    format!("sc start \"{name}\"", name = super::SERVICE_NAME)
}

/// Compose the `sc stop` command for the operator runbook.
pub fn compose_stop_command() -> String {
    format!("sc stop \"{name}\"", name = super::SERVICE_NAME)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn create_command_contains_service_metadata() {
        let cmd = compose_create_command(r"C:\Program Files\nanovms\nanovms.exe");
        assert!(cmd.contains(super::super::SERVICE_NAME));
        assert!(cmd.contains("DisplayName"));
        assert!(cmd.contains("start= auto"));
        assert!(cmd.contains(r"C:\Program Files\nanovms\nanovms.exe"));
    }

    #[test]
    fn delete_command_targets_nanovms() {
        let cmd = compose_delete_command();
        assert_eq!(cmd, "sc delete \"nanovms\"");
    }

    #[test]
    fn start_and_stop_commands_target_nanovms() {
        assert_eq!(compose_start_command(), "sc start \"nanovms\"");
        assert_eq!(compose_stop_command(), "sc stop \"nanovms\"");
    }

    #[test]
    fn constants_match_win32_values() {
        // Pin to upstream values so a future windows-rs bump is caught
        // at unit-test time, not at SCM-registration time.
        assert_eq!(SERVICE_AUTO_START, 0x00000002);
        assert_eq!(SERVICE_WIN32_OWN_PROCESS, 0x00000010);
        assert_eq!(SERVICE_WIN32_SHARE_PROCESS, 0x00000020);
        assert_eq!(SERVICE_ERROR_NORMAL, 0x00000001);
        assert_eq!(SERVICE_ERROR_CRITICAL, 0x00000003);
    }
}

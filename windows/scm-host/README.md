# nvms-scm-host

nanovms Windows Service Control Manager (SCM) host — wires `nvms-cli`
into the Windows service infrastructure so the 3-tier microVM isolation
daemon can run as a registered Windows service (`sc start nanovms`).

PILLAR-TAXONOMY-v2 **L123** (Windows Native FFI — `microsoft/windows-rs`)
and **L130** (System Service Integration — Windows-service / SCM).

## Crate shape

| Path | Purpose |
| --- | --- |
| `src/lib.rs` | Public API: `capabilities()`, `install_service()`, `stop_service()`, `start_service()`, `write_event_log()`; error type `ScmHostError` |
| `src/scm.rs` | Win32 SCM constants (`SERVICE_AUTO_START`, `SERVICE_WIN32_OWN_PROCESS`, `SERVICE_ERROR_NORMAL`, `SERVICE_ERROR_CRITICAL`, `SERVICE_WIN32_SHARE_PROCESS`) + `sc create` / `sc delete` / `sc start` / `sc stop` command composers |
| `src/service.rs` | Windows-only `ServiceMain` skeleton (cfg + feature gated) — defer real `CreateServiceW` to follow-up PR |
| `src/event_log.rs` | `EventLogEntry` value type + Win32 severity constants + `eventcreate.exe` command composer for the `Win32_System_EventLog` channel |

## Feature flags

| Flag | Default | Pulls in |
| --- | --- | --- |
| (none) | ✓ | nothing extra — scaffold compiles cross-platform |
| `service` |  | `windows = "0.58"` (Win32_System_Services, Win32_System_EventLog, Win32_Foundation, Win32_Security) + `windows-sys = "0.59"` |

The `service` feature is additionally gated to `target_os = "windows"`
in `[target.'cfg(target_os = "windows")'.dependencies]`. Result:
`cargo check --workspace` continues to pass on macOS / linux without a
Windows toolchain.

## Design rationale

The daemon logic itself is OS-agnostic Rust in `pkg/`. To run on
Windows we need a binary that:

1. **Registers** itself with the SCM so it survives reboots and crashes.
2. **Reports status** (`SERVICE_RUNNING`, `SERVICE_STOP_PENDING`, …)
   back to the SCM so `sc query` / `Get-Service` reflects truth.
3. **Logs** lifecycle and tier events to the `nanovms` Application
   channel so operators can audit tier transitions.

All three interactions are pure Win32 FFI — `CreateServiceW`,
`SetServiceStatus`, `RegisterEventSourceW`, `ReportEventW` from
`Win32_System_Services` / `Win32_System_EventLog`.

This crate wraps that FFI surface in safe **stubs** first. The scope of
this PR is the FFI shape + dependency wiring + command composers so that
`cargo check --workspace` stays green on macOS / linux and the
follow-up PR can land the real unsafe blocks without touching the
workspace lint.

## Cross-platform guarantees

| Target | `cargo check` | `cargo build --features service` |
| --- | --- | --- |
| `x86_64-apple-darwin` (macOS dev) | ✅ | ⛔ feature gated off (`cfg(target_os = "windows")`) |
| `x86_64-unknown-linux-gnu` | ✅ | ⛔ same |
| `x86_64-pc-windows-msvc` | ✅ (no feature) | 🔄 needs MSVC + Windows SDK |

## Operator runbook (post-merge follow-up)

### Install

```powershell
# 1. Build the daemon + the SCM shim
cargo build --release --target x86_64-pc-windows-msvc -p nvms-cli
cargo build --release --target x86_64-pc-windows-msvc -p nvms-scm-host --features service

# 2. Install (operator wrapper around the lib's `install_service()`)
$env:NANOVMS_PATH = "C:\Program Files\nanovms\nanovms.exe"
sc create nanovms binPath= "$env:NANOVMS_PATH" start= auto `
    DisplayName= "nanovms Daemon (3-tier microVM isolation)"

# 3. Start
sc start nanovms

# 4. Verify
sc query nanovms
Get-EventLog -LogName Application -Source nanovms -Newest 10
```

### Uninstall

```powershell
# 1. Stop the service (poll until SERVICE_STOPPED)
sc stop nanovms
sc query nanovms   # confirm STATE = STOPPED

# 2. Delete the registration
sc delete nanovms

# 3. (Optional) Remove the event-log channel source
Remove-EventLog -Source nanovms -LogName Application -ErrorAction SilentlyContinue
```

## PowerShell install helper (planned, follow-up PR)

```powershell
# install.ps1
[CmdletBinding()]
param(
    [Parameter(Mandatory=$true)][string]$BinaryPath,
    [string]$ServiceName = "nanovms"
)
$ErrorActionPreference = "Stop"

if (-not (Test-Path $BinaryPath)) {
    throw "nanovms binary not found at $BinaryPath"
}

$existing = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if ($existing) {
    Write-Warning "Service '$ServiceName' already registered; stopping and removing first."
    Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
    sc.exe delete $ServiceName | Out-Null
    Start-Sleep -Seconds 1
}

sc.exe create $ServiceName binPath= "`"$BinaryPath`"" start= auto `
    DisplayName= "nanovms Daemon (3-tier microVM isolation)" | Out-Null

# Optional: descriptive service description
sc.exe description $ServiceName "nanovms — WASM / gVisor / Firecracker 3-tier microVM isolation daemon. See SPEC.md." | Out-Null

sc.exe start $ServiceName | Out-Null
Write-Host "nanovms service installed and started."
```

```powershell
# uninstall.ps1
[CmdletBinding()]
param(
    [string]$ServiceName = "nanovms"
)
$ErrorActionPreference = "Stop"

$svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if (-not $svc) {
    Write-Host "Service '$ServiceName' not registered; nothing to do."
    return
}

if ($svc.Status -ne 'Stopped') {
    Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
    $svc.WaitForStatus('Stopped', '00:00:30')
}

sc.exe delete $ServiceName | Out-Null
Remove-EventLog -Source $ServiceName -LogName Application -ErrorAction SilentlyContinue
Write-Host "nanovms service uninstalled."
```

## Reference

- PhenoCompose PR #77 (`windows/scm-service/`, MERGED 51babba) — same scaffold pattern
- PhenoCompose `docs/ffi/WINDOWS-SCM.md` — companion design doc
- PILLAR-TAXONOMY-v2.md v2.2 § L123 / § L130

## Scaffold status

This PR ships the **FFI shape** only:

- ✅ `windows = "0.58"` + `windows-sys = "0.59"` deps wired under
  `[target.'cfg(target_os = "windows")'.dependencies]`
- ✅ `ScmCapabilities` / `ScmHostError` / public API shape
- ✅ Win32 constants exposed (`SERVICE_AUTO_START`,
  `SERVICE_WIN32_OWN_PROCESS`, etc.)
- ✅ `event_log` value type + severity constants
- ✅ `cargo check --workspace` passes on macOS
- 🔄 Real `CreateServiceW` / `RegisterEventSourceW` FFI calls land in
  the follow-up PR once the windows-rs version pin (0.58 → 0.59+) is
  finalised.

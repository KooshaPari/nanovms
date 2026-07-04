# nanovms — Windows Service Control Manager (SCM) host

`windows/scm-host` is the nanovms service-mode scaffold that wires
`nvms-cli` into the Windows service infrastructure so the
3-tier microVM isolation daemon can run as a registered Windows service
(`sc start nanovms`).

PILLAR-TAXONOMY-v2 **L123** (Windows Native FFI — `microsoft/windows-rs`)
and **L130** (System Service Integration — Windows-service / SCM).

## Crate surface

| Path | Purpose |
| --- | --- |
| `src/lib.rs` | Public API: `capabilities()`, `install_service()`, `stop_service()`, `start_service()`, `write_event_log()`; error type `ScmHostError` |
| `src/scm.rs` | Win32 SCM constants (`SERVICE_AUTO_START`, `SERVICE_WIN32_OWN_PROCESS`, `SERVICE_ERROR_NORMAL`, `SERVICE_ERROR_CRITICAL`, `SERVICE_WIN32_SHARE_PROCESS`) + `sc create` / `sc delete` / `sc start` / `sc stop` command composers |
| `src/service.rs` | Windows-only `ServiceMain` skeleton (cfg + feature gated) — defer real `CreateServiceW` to follow-up PR |
| `src/event_log.rs` | `EventLogEntry` value type + Win32 severity constants + `eventcreate.exe` command composer for the `Win32_System_EventLog` channel |

## Feature flags

| Feature | Default | Effect |
| --- | --- | --- |
| — | OFF | Scaffold compiles cross-platform (macOS / Linux / Windows without toolchain). `ScmHostError::BridgeUnavailable` returned for every FFI entry-point. |
| `service` | OFF | Pulls in `windows = "0.58"` (`Win32_System_Services`, `Win32_System_EventLog`) + `windows-sys = "0.59"`. Enables `nvms_scm_host::service::*` real FFI surface. |

`cargo check --workspace` succeeds on every host because the `service`
feature is OFF by default and the cfg gates prevent accidental use on
non-Windows targets.

## Usage from `nvms-cli`

```rust
use nvms_scm_host::{capabilities, install_service, ScmCapabilities};

fn maybe_install_service(bin_path: &str) {
    let caps: ScmCapabilities = capabilities();
    if !caps.scm_registration {
        eprintln!("nanovms SCM host is Windows-only; skipping install");
        return;
    }
    install_service(bin_path).expect("CreateServiceW succeeded");
}
```

## PowerShell install script (operator-driven)

```powershell
# Installs the nanovms service and starts it.
& bin\nanovms.exe --print-scm-install  | Out-File -Encoding ascii install.ps1
. ./install.ps1
sc start nanovms
sc query nanovms
Get-EventLog -LogName Application -Source nanovms -Newest 10
```

`install.ps1` is emitted by `nvms-cli --print-scm-install` and wraps the
`install_service()` call from above.

## Event-log channel (event-log.mc)

A real-world install will provide `event-log.mc` so the source appears
under `Applications and Services Logs / nanovms`. The scaffold ships the
command composer only — the `.mc` file is forthcoming in a follow-up PR
that turns the `service` feature on by default.

## Cross-compile notes

`x86_64-pc-windows-msvc` is now part of the cross-compile CI matrix
(PR from `feat/cross-compile-ci-matrix`). Building `nvms-scm-host` with
`cargo build --target x86_64-pc-windows-msvc --features service` is the
gating step for the L129 cross_compile pillar.

## Reference

- PhenoCompose `windows/scm-service/` — PR #77 (MERGED `51babba`).
  Mirrored module split: `lib.rs` / `scm.rs` / `service_main.rs`.
- PhenoCompose `docs/ffi/WINDOWS-SCM.md` — same shape, sister repo.
- `PILLAR-TAXONOMY-v2.md` v2.1 §L123 + §L130.

# nanovms Frontend (TUI)

A ratatui-based terminal UI for the ops console.

## Stack

- `ratatui` v0.27+ - TUI framework
- `crossterm` - cross-platform terminal backend
- `tui-input` - text input widgets
- `tokio` - async runtime for event loop
- `reqwest` - HTTP client to daemon

## Why TUI?

- Single binary, no install (just download)
- SSH-friendly (works over remote shell)
- Low memory + CPU footprint
- Keyboard-first (faster than mouse for ops)

## Tabs

```
   ┌────────┬────────┬────────┬────────┬────────┐
   │ Running │ All   │ Events │ Logs   │ Audit  │
   └────────────────────────────────────────────┘
   ┌───────────────────────────────────────────┐
   │ ID       Name         Image     Status    │
   │ sb-001   test-1       alpine    running   │
   │ sb-002   test-2       redis     paused    │
   │ sb-003   prod-app     postgres  running   │
   │ ...                                        │
   └───────────────────────────────────────────┘
   q: quit  /: search  c: create  d: delete  enter: details
```

## Layout

```go
ui := ui.New()
ui.SetRoot(layout.Tabbed(
    "Running", RunningTab{daemon: d},
    "All",     AllTab{daemon: d},
    "Events",  EventsTab{daemon: d},
    "Logs",    LogsTab{daemon: d, sandboxID: sel},
    "Audit",   AuditTab{daemon: d},
))
```

## State

```go
type AppState struct {
    Daemon       *daemon.Client
    Sandboxes    []Sandbox
    SelectedIdx  int
    Filter       string
    Mode         ViewMode
}
```

## Key bindings

| Key | Action |
|-----|--------|
| `q` | quit |
| `Tab` | next tab |
| `Shift+Tab` | prev tab |
| `/` | filter |
| `c` | create sandbox |
| `d` | delete selected |
| `Enter` | details |
| `l` | view logs |
| `e` | exec into |
| `?` | help |

## Accessibility

- ARIA-like screen reader hints via `lipgloss`
- High contrast mode (toggle with `H`)
- Color blind safe palette (`monokai`, `solarized-dark`)
- Respects `NO_COLOR`
- Respects `TERM=dumb` (renders plain)

## Build

```bash
go build -o nanovms-tui ./cmd/nanovms-tui
```

## Bench

- Initial render: < 50ms
- Refresh tick: < 16ms (60 FPS)
- Memory: < 30MB
- Binary size: < 8MB

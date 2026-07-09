# nanovms Logging

Structured logging via Go's standard `log/slog` package (Go 1.21+).

## Stack

- `log/slog` - structured logging (stdlib since Go 1.21)
- JSON handler for production
- Text handler for dev

## Setup

```go
import (
    "log/slog"
    "os"
)

func init() {
    handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelInfo,
        AddSource: false,
    })
    logger := slog.New(handler)
    slog.SetDefault(logger)
}
```

## Usage

```go
slog.Info("sandbox created", "id", id, "name", cfg.Name)
slog.Error("sandbox create failed", "err", err, "name", cfg.Name)
```

## Context-aware logging

```go
ctx = WithLogger(ctx, slog.With("trace_id", traceID))
log := LoggerFrom(ctx)
log.Info("handling request", "method", r.Method)
```

## Redaction

```go
// Use a custom ReplaceAttr to redact sensitive fields
slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
        if a.Key == "token" || a.Key == "password" {
            return slog.String(a.Key, "REDACTED")
        }
        return a
    },
})
```

## Log levels

- `ERROR` - 4xx/5xx, panics
- `WARN` - retries, deprecated API
- `INFO` - lifecycle (start, stop, create)
- `DEBUG` - per-request details

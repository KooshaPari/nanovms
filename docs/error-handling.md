# nanovms Error Handling

Wrap errors with context using `fmt.Errorf("...: %w", err)`. Document the error catalog in code comments + this doc.

## Library errors (sentinel)

```go
package sandbox

import "errors"

var (
    ErrNotFound       = errors.New("sandbox not found")
    ErrAlreadyExists  = errors.New("sandbox already exists")
    ErrInvalidConfig  = errors.New("invalid config")
    ErrBackendOffline = errors.New("sandbox backend offline")
)
```

## Application errors (wrapping)

```go
func (a *Adapter) Get(ctx context.Context, id string) (*Sandbox, error) {
    sb, ok := a.sandboxes[id]
    if !ok {
        return nil, fmt.Errorf("sandbox %s: %w", id, ErrNotFound)
    }
    return sb, nil
}
```

## Error catalog

| Sentinel | HTTP | Retry? | User message |
|----------|------|--------|--------------|
| `ErrNotFound` | 404 | no | "sandbox not found" |
| `ErrAlreadyExists` | 409 | no | "sandbox already exists" |
| `ErrInvalidConfig` | 400 | no | "invalid config" |
| `ErrBackendOffline` | 503 | yes | "sandbox backend offline" |

## Conversion (sentinel -> http)

```go
func httpStatusFor(err error) int {
    switch {
    case errors.Is(err, ErrNotFound):       return http.StatusNotFound
    case errors.Is(err, ErrAlreadyExists):  return http.StatusConflict
    case errors.Is(err, ErrInvalidConfig):  return http.StatusBadRequest
    case errors.Is(err, ErrBackendOffline): return http.StatusServiceUnavailable
    default:                                 return http.StatusInternalServerError
    }
}
```

## Tests

```go
func TestGetNotFound(t *testing.T) {
    a := NewAdapter()
    _, err := a.Get(context.Background(), "missing")
    if !errors.Is(err, ErrNotFound) {
        t.Fatalf("got %v, want ErrNotFound", err)
    }
}
```

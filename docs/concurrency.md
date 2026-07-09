# nanovms Concurrency

Structured concurrency for parallel sandbox operations + port dispatch.

## Stack (Go 1.21+)

- `sync` (stdlib: Mutex, RWMutex, WaitGroup, Once)
- `golang.org/x/sync/errgroup` - bounded + cancellation
- `golang.org/x/sync/semaphore` - weight-based limits
- `github.com/knz/boundedparallel` - alternative

## Patterns

### Bounded parallel with errgroup

```go
import "golang.org/x/sync/errgroup"

g := new(errgroup.Group)
g.SetLimit(32)

for _, sb := range sandboxes {
    sb := sb
    g.Go(func() error {
        return start(ctx, sb)
    })
}
if err := g.Wait(); err != nil { return err }
```

### Semaphore (weight-based)

```go
sem := semaphore.NewWeighted(64)
for _, req := range requests {
    weight := int64(req.Weight)
    if err := sem.Acquire(ctx, weight); err != nil { return err }
    go func() {
        defer sem.Release(weight)
        handle(req)
    }()
}
```

### Bounded parallel + first error wins

```go
import "sync/atomic"
var wg sync.WaitGroup
var firstErr atomic.Value
for _, item := range items {
    item := item
    wg.Add(1)
    go func() {
        defer wg.Done()
        if err := handle(ctx, item); err != nil {
            firstErr.CompareAndSwap(nil, err)
        }
    }()
}
wg.Wait()
if v := firstErr.Load(); v != nil { return v.(error) }
```

## Send/Sync audit

```go
// All types passed across goroutines must satisfy Send/Sync.
// Use `go vet -copylocks` + `errcheck -blank` to verify.
// For struct holding a non-Sync field: use a mutex or atomic.
```

## Bench

| Pattern | Target |
|---------|--------|
| 100 parallel sandbox start (errgroup, limit=32) | < 5s |
| 1000 parallel ops (semaphore, weight=1) | < 30s |
| Channel close propagation | < 1us |

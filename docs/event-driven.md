# nanovms Event-Driven

Event bus for cross-component signaling + outbox pattern for reliable delivery.

## Stack (Go)

- `sync/atomic` + channels - in-process pub/sub
- Optional NATS / Kafka for cross-process

## In-process pub/sub

```go
type EventBus struct {
    subscribers []chan Event
    mu          sync.RWMutex
}

func (b *EventBus) Subscribe() <-chan Event {
    ch := make(chan Event, 64)
    b.mu.Lock()
    b.subscribers = append(b.subscribers, ch)
    b.mu.Unlock()
    return ch
}

func (b *EventBus) Publish(e Event) {
    b.mu.RLock()
    defer b.mu.RUnlock()
    for _, ch := range b.subscribers {
        select {
        case ch <- e:
        default: // drop if subscriber is slow
        }
    }
}
```

## Event types

```go
type Event interface {
    eventTag()
}

type SandboxCreated struct {
    ID   string
    Time time.Time
}
func (SandboxCreated) eventTag() {}

type SecretRotated struct {
    Ref string
}
func (SecretRotated) eventTag() {}
```

## Outbox pattern

For durable, exactly-once delivery to external systems:

```
   ┌──────────────┐   ┌────────────┐   ┌──────────────┐
   │ Application │ ─>│  Outbox    │ ─>│  Publisher   │
   │             │   │  (table)   │   │  (NATS/etc)  │
   └──────────────┘   └────────────┘   └──────────────┘
```

1. App writes event + business data in same DB transaction
2. Background publisher reads outbox rows, dispatches
3. On success, marks outbox row as published
4. On failure, retries with exponential backoff

## Library layout

- `internal/eventbus/` - in-process pub/sub (above)
- `internal/outbox/` - durable outbox

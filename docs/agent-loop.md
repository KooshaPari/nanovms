# nanovms Agent Loop

`agentctl` is a single-binary CLI that exposes nanovms port-trait operations
as JSON-in/JSON-out RPC. It's wire-compatible with the Omniroute dispatcher
schema so agents can drive nanovms through the same channel as any other
service.

## Schema

Request:
```json
{"method": "sandbox.create", "params": {"name": "hello"}}
```

Response:
```json
{"ok": true, "result": {"id": "sb-12345", "name": "hello"}}
```

## Methods

| Method         | Params                | Returns                |
|----------------|----------------------|------------------------|
| sandbox.create | `{name: string}`     | `{id, name, createdAt}`|
| sandbox.list   | `{}`                  | `[SandboxRuntime]`     |
| sandbox.get    | `{id: string}`       | `{id, status, ...}`    |

## Usage

```bash
echo '{"method":"sandbox.create","params":{"name":"hello"}}' \
  | nanovms-agentctl
```

Output:
```json
{"ok":true,"result":{"createdAt":"2026-07-09T19:00:00Z","id":"sb-1234567890","name":"hello"}}
```

## Implementation

`cmd/agentctl/main.go` — pure Go, no external deps, 30s timeout per call.

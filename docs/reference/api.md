# API Reference

REST API documentation for NanoVMS HTTP server.

## Base URL

```
http://localhost:8080/api/v1
```

## Authentication

Currently no authentication in local mode. For production, use `--auth` flag with JWT tokens.

## Common Headers

```http
Content-Type: application/json
Accept: application/json
Authorization: Bearer <token>  # If auth enabled
```

## Response Format

All responses follow this format:

```json
{
  "success": true,
  "data": { ... },
  "error": null,
  "meta": {
    "request_id": "req-123",
    "duration_ms": 5
  }
}
```

Error response:

```json
{
  "success": false,
  "data": null,
  "error": {
    "code": "VM_NOT_FOUND",
    "message": "VM 'my-vm' not found"
  },
  "meta": {
    "request_id": "req-456",
    "duration_ms": 2
  }
}
```

---

## VMs

### List VMs

```http
GET /vms
```

Query parameters:
- `flavor` - Filter by flavor
- `status` - Filter by status

Response:

```json
{
  "vms": [
    {
      "id": "vm-abc123",
      "name": "my-vm",
      "flavor": "microvm",
      "status": "running",
      "cpu": 2,
      "memory": 1073741824,
      "created_at": "2026-01-01T00:00:00Z"
    }
  ],
  "total": 1
}
```

### Create VM

```http
POST /vms
```

Request body:

```json
{
  "name": "my-vm",
  "flavor": "microvm",
  "cpu": 2,
  "memory": "1G",
  "disk": "10G",
  "image": "ubuntu-22.04",
  "network": "nat",
  "tags": ["test", "development"]
}
```

Response: `201 Created`

```json
{
  "id": "vm-abc123",
  "name": "my-vm",
  "flavor": "microvm",
  "status": "created",
  "created_at": "2026-01-01T00:00:00Z"
}
```

### Get VM

```http
GET /vms/{id}
```

Response:

```json
{
  "id": "vm-abc123",
  "name": "my-vm",
  "flavor": "microvm",
  "status": "running",
  "cpu": 2,
  "memory": 1073741824,
  "disk": {
    "size": 10737418240,
    "used": 2147483648
  },
  "network": {
    "type": "nat",
    "ip": "192.168.100.2",
    "mac": "52:54:00:12:34:56"
  },
  "created_at": "2026-01-01T00:00:00Z",
  "started_at": "2026-01-01T00:01:00Z",
  "tags": ["test"]
}
```

### Start VM

```http
POST /vms/{id}/start
```

Optional body:

```json
{
  "snapshot": "base-state",
  "debug": false
}
```

Response: `202 Accepted`

```json
{
  "status": "starting",
  "started_at": "2026-01-01T00:02:00Z"
}
```

### Stop VM

```http
POST /vms/{id}/stop
```

Optional body:

```json
{
  "force": false,
  "timeout": 30
}
```

Response: `202 Accepted`

```json
{
  "status": "stopping"
}
```

### Restart VM

```http
POST /vms/{id}/restart
```

Response: `202 Accepted`

```json
{
  "status": "restarting"
}
```

### Delete VM

```http
DELETE /vms/{id}
```

Query parameters:
- `force` - Force delete (default: false)

Response: `204 No Content`

### Execute Command

```http
POST /vms/{id}/exec
```

Request body:

```json
{
  "command": ["ls", "-la"],
  "user": "root",
  "cwd": "/home",
  "env": {
    "HOME": "/root"
  },
  "timeout": 30
}
```

Response:

```json
{
  "exit_code": 0,
  "stdout": "total 64\ndrwxr-xr-x 24 root root 4096 Jan  1 00:00 .\n...",
  "stderr": "",
  "duration_ms": 45
}
```

### Get VM Stats

```http
GET /vms/{id}/stats
```

Response:

```json
{
  "cpu": {
    "usage_percent": 12.5,
    "user_percent": 8.0,
    "system_percent": 4.5
  },
  "memory": {
    "total": 1073741824,
    "used": 536870912,
    "usage_percent": 50.0
  },
  "network": {
    "rx_bytes": 1024000,
    "tx_bytes": 512000,
    "rx_packets": 1000,
    "tx_packets": 500
  },
  "disk": {
    "read_bytes": 104857600,
    "write_bytes": 52428800,
    "read_iops": 100,
    "write_iops": 50
  },
  "timestamp": "2026-01-01T00:05:00Z"
}
```

### Stream VM Stats (WebSocket)

```http
WS /vms/{id}/stats/stream
```

Messages:

```json
{
  "type": "stats",
  "data": { ...stats object... }
}
```

---

## Snapshots

### List Snapshots

```http
GET /vms/{id}/snapshots
```

Response:

```json
{
  "snapshots": [
    {
      "id": "snap-xyz789",
      "name": "base-install",
      "size": 5368709120,
      "created_at": "2026-01-01T00:00:00Z"
    }
  ]
}
```

### Create Snapshot

```http
POST /vms/{id}/snapshots
```

Request body:

```json
{
  "name": "base-install",
  "description": "Base Ubuntu installation"
}
```

Response: `201 Created`

```json
{
  "id": "snap-xyz789",
  "name": "base-install",
  "size": 5368709120,
  "created_at": "2026-01-01T00:10:00Z"
}
```

### Restore Snapshot

```http
POST /vms/{id}/snapshots/{snapshot_id}/restore
```

Response: `202 Accepted`

### Delete Snapshot

```http
DELETE /vms/{id}/snapshots/{snapshot_id}
```

Response: `204 No Content`

---

## Sandboxes

### List Sandboxes

```http
GET /sandboxes
```

Query parameters:
- `tier` - Filter by tier

Response:

```json
{
  "sandboxes": [
    {
      "id": "sand-abc123",
      "name": "my-sandbox",
      "tier": "gvisor",
      "vm_id": "vm-abc123",
      "created_at": "2026-01-01T00:00:00Z"
    }
  ]
}
```

### Create Sandbox

```http
POST /sandboxes
```

Request body:

```json
{
  "name": "my-sandbox",
  "tier": "gvisor",
  "vm_id": "vm-abc123"
}
```

Response: `201 Created`

### Apply Sandbox

```http
POST /vms/{id}/sandbox
```

Request body:

```json
{
  "tier": "gvisor"
}
```

Response: `200 OK`

### Delete Sandbox

```http
DELETE /sandboxes/{id}
```

Response: `204 No Content`

---

## Networks

### List Networks

```http
GET /networks
```

Response:

```json
{
  "networks": [
    {
      "id": "net-abc123",
      "name": "default",
      "type": "nat",
      "subnet": "192.168.100.0/24",
      "gateway": "192.168.100.1",
      "dhcp_enabled": true
    }
  ]
}
```

### Create Network

```http
POST /networks
```

Request body:

```json
{
  "name": "my-network",
  "type": "nat",
  "subnet": "10.0.0.0/24",
  "dhcp": true
}
```

Response: `201 Created`

### Delete Network

```http
DELETE /networks/{id}
```

Response: `204 No Content`

---

## Images

### List Images

```http
GET /images
```

Response:

```json
{
  "images": [
    {
      "id": "img-abc123",
      "name": "ubuntu-22.04",
      "size": 536870912,
      "format": "qcow2",
      "created_at": "2026-01-01T00:00:00Z"
    }
  ]
}
```

### Pull Image

```http
POST /images/pull
```

Request body:

```json
{
  "name": "ubuntu-22.04",
  "url": "https://cloud-images.ubuntu.com/releases/22.04/release/ubuntu-22.04.qcow2"
}
```

Response: `202 Accepted`

```json
{
  "job_id": "job-xyz789",
  "status": "pulling"
}
```

### Delete Image

```http
DELETE /images/{id}
```

Response: `204 No Content`

---

## Storage

### List Storage Pools

```http
GET /storage/pools
```

Response:

```json
{
  "pools": [
    {
      "id": "pool-abc123",
      "name": "default",
      "type": "directory",
      "path": "/var/lib/nanovms",
      "total": 107374182400,
      "used": 53687091200,
      "available": 53687091200
    }
  ]
}
```

---

## Metrics

### Get Metrics (Prometheus)

```http
GET /metrics
```

Returns Prometheus-formatted metrics.

### Get Metrics (JSON)

```http
GET /metrics/json
```

Response:

```json
{
  "nanovms_vms_total": 10,
  "nanovms_vms_running": 5,
  "nanovms_vms_stopped": 5,
  "nanovms_memory_bytes_total": 10737418240,
  "nanovms_memory_bytes_used": 5368709120,
  "nanovms_cpu_percent": 12.5,
  "nanovms_sandboxes_total": 8
}
```

---

## Health

### Health Check

```http
GET /health
```

Response:

```json
{
  "status": "healthy",
  "version": "0.1.0",
  "uptime_seconds": 3600
}
```

---

## Events (WebSocket)

Subscribe to events:

```http
WS /events
```

Event types:

```json
{
  "type": "vm.created",
  "data": {
    "id": "vm-abc123",
    "name": "my-vm"
  },
  "timestamp": "2026-01-01T00:00:00Z"
}
```

```json
{
  "type": "vm.started",
  "data": {
    "id": "vm-abc123"
  },
  "timestamp": "2026-01-01T00:01:00Z"
}
```

```json
{
  "type": "vm.stopped",
  "data": {
    "id": "vm-abc123"
  },
  "timestamp": "2026-01-01T00:05:00Z"
}
```

```json
{
  "type": "vm.error",
  "data": {
    "id": "vm-abc123",
    "error": "VM crashed"
  },
  "timestamp": "2026-01-01T00:10:00Z"
}
```

---

## Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `VM_NOT_FOUND` | 404 | VM does not exist |
| `VM_ALREADY_EXISTS` | 409 | VM with name already exists |
| `VM_ALREADY_RUNNING` | 409 | VM is already running |
| `VM_NOT_RUNNING` | 409 | VM is not running |
| `SNAPSHOT_NOT_FOUND` | 404 | Snapshot does not exist |
| `NETWORK_NOT_FOUND` | 404 | Network does not exist |
| `IMAGE_NOT_FOUND` | 404 | Image does not exist |
| `INVALID_CONFIG` | 400 | Invalid configuration |
| `PERMISSION_DENIED` | 403 | Permission denied |
| `RESOURCE_EXHAUSTED` | 507 | Out of resources |
| `INTERNAL_ERROR` | 500 | Internal server error |

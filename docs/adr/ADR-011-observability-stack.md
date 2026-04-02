# ADR-011: Observability Stack

## Status

Accepted

## Context

NanoVMS requires comprehensive observability for:
- Performance debugging
- Capacity planning
- Incident response
- SLA monitoring
- Cost attribution

## Decision

### Observability Stack

```
┌────────────────────────────────────────────────────────────────────┐
│                     NanoVMS Observability Stack                        │
├────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                     Metrics (Prometheus)                        │  │
│  │                                                                │  │
│  │   vm_start_seconds{flavor="firecracker"} 0.045                │  │
│  │   vm_memory_bytes{vm="test-vm"} 268435456                    │  │
│  │   sandbox_active_count{type="gvisor"} 42                       │  │
│  │   cpu_seconds_total{core="0-15"} 12345.67                      │  │
│  │   disk_io_bytes_total{device="vda"} 9876543210                  │  │
│  │   network_bytes_total{interface="eth0"} 1234567890              │  │
│  │                                                                │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                              │                                        │
│                              ▼                                        │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                     Traces (OpenTelemetry)                      │  │
│  │                                                                │  │
│  │   Trace: vm.create                                              │  │
│  │     ├─ validate_config (10ms)                                 │  │
│  │     ├─ allocate_resources (5ms)                                │  │
│  │     ├─ create_hypervisor (20ms)                               │  │
│  │     ├─ setup_networking (15ms)                                │  │
│  │     └─ start_vm (45ms)                                        │  │
│  │                                                                │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                              │                                        │
│                              ▼                                        │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                     Logs (Structured JSON)                       │  │
│  │                                                                │  │
│  │   {                                                           │  │
│  │     "level": "INFO",                                          │  │
│  │     "ts": "2026-04-02T12:34:56.789Z",                        │  │
│  │     "msg": "VM started",                                       │  │
│  │     "vm_id": "vm-abc123",                                      │  │
│  │     "flavor": "firecracker",                                   │  │
│  │     "duration_ms": 45,                                         │  │
│  │     "trace_id": "abc123...",                                   │  │
│  │     "span_id": "def456..."                                    │  │
│  │   }                                                           │  │
│  │                                                                │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                                                     │
└────────────────────────────────────────────────────────────────────┘
```

### Metrics Specification

```yaml
# prometheus/nanovms.yml
metrics:
  - name: vm_created_total
    type: counter
    help: "Total number of VMs created"
    labels:
      - flavor
      - result
      - host

  - name: vm_start_seconds
    type: histogram
    help: "VM start time in seconds"
    buckets: [0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0]
    labels:
      - flavor

  - name: vm_memory_bytes
    type: gauge
    help: "Current VM memory usage in bytes"
    labels:
      - vm_id
      - flavor

  - name: vm_cpu_seconds_total
    type: counter
    help: "Total CPU time consumed by VMs"
    labels:
      - vm_id
      - core

  - name: sandbox_active_count
    type: gauge
    help: "Number of active sandboxes"
    labels:
      - type
      - host

  - name: disk_io_bytes_total
    type: counter
    help: "Total disk I/O in bytes"
    labels:
      - device
      - direction  # read, write
      - vm_id

  - name: network_bytes_total
    type: counter
    help: "Total network traffic in bytes"
    labels:
      - interface
      - direction  # rx, tx
      - vm_id

  - name: storage_bytes_total
    type: gauge
    help: "Total storage allocated in bytes"
    labels:
      - pool
      - tier  # ssd, nvme, hdd

  - name: gpu_utilization_percent
    type: gauge
    help: "GPU utilization percentage"
    labels:
      - gpu_id
      - vm_id

  - name: request_duration_seconds
    type: histogram
    help: "API request duration"
    buckets: [0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0]
    labels:
      - method
      - endpoint
      - status_code
```

### OpenTelemetry Integration

```go
// pkg/observability/tracing.go
package observability

import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace"
    "go.opentelemetry.io/otel/sdk/trace/provider"
)

func initTracing(serviceName string) (*trace.TracerProvider, error) {
    exporter, err := otlptrace.New(
        context.Background(),
        otlptrace.WithEndpoint("otel-collector:4317"),
    )
    if err != nil {
        return nil, err
    }

    tp := provider.NewTracerProvider(
        provider.WithBatcher(exporter),
        provider.WithSampler(
            sampler.ParentBased(
                sampler.TraceIDRatioBased(0.1), // 10% sampling
            ),
        ),
    )

    otel.SetTracerProvider(tp)
    return tp, nil
}

// Span names for VM operations
const (
    SpanValidateConfig   = "vm.validate_config"
    SpanAllocateResource = "vm.allocate_resource"
    SpanCreateHypervisor = "vm.create_hypervisor"
    SpanSetupNetworking  = "vm.setup_networking"
    SpanSetupStorage     = "vm.setup_storage"
    SpanStartVM          = "vm.start"
    SpanStopVM           = "vm.stop"
    SpanDeleteVM         = "vm.delete"
    SpanCreateSnapshot    = "vm.create_snapshot"
    SpanRestoreSnapshot   = "vm.restore_snapshot"
    SpanExecCommand       = "vm.exec_command"
)
```

### Structured Logging

```go
// pkg/observability/logger.go
package observability

import (
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

func NewLogger(service string, level zapcore.Level) (*zap.Logger, error) {
    config := zap.Config{
        Level:       zap.NewAtomicLevelAt(level),
        Development: false,
        Encoding:    "json",
        EncoderConfig: zapcore.EncoderConfig{
            TimeKey:        "ts",
            LevelKey:       "level",
            NameKey:        "logger",
            CallerKey:      "caller",
            FunctionKey:    zapcore.OmitKey,
            MessageKey:     "msg",
            StacktraceKey:  "stacktrace",
            LineEnding:     zapcore.DefaultLineEnding,
            EncodeLevel:    zapcore.LowercaseLevelEncoder,
            EncodeTime:     zapcore.ISO8601TimeEncoder,
            EncodeDuration: zapcore.MillisDurationEncoder,
            EncodeCaller:   zapcore.ShortCallerEncoder,
        },
        OutputPaths:      []string{"stdout", "/var/log/nanovms/app.log"},
        ErrorOutputPaths: []string{"stderr", "/var/log/nanovms/error.log"},
    }

    logger, err := config.Build()
    if err != nil {
        return nil, err
    }

    return logger, nil
}

// Log fields
var (
    FieldVMID       = zap.String("vm_id", "")
    FieldVMFlavor   = zap.String("vm_flavor", "")
    FieldUserID     = zap.String("user_id", "")
    FieldTraceID    = zap.String("trace_id", "")
    FieldSpanID     = zap.String("span_id", "")
    FieldDuration    = zap.Duration("duration", 0)
    FieldHost       = zap.String("host", "")
    FieldRegion     = zap.String("region", "")
    FieldError      = zap.Error(errors.New(""))
    FieldRequestID  = zap.String("request_id", "")
)

// Usage
func logVMStart(vm *VM, span trace.Span) {
    logger.Info("VM started",
        FieldVMID,
        FieldVMFlavor,
        FieldDuration,
        zap.String("trace_id", span.SpanContext().TraceID().String()),
    )
}
```

### Alerting Rules

```yaml
# prometheus/alerts/nanovms.yml
groups:
  - name: nanovms.alerts
    rules:
      # High VM failure rate
      - alert: VMCreateFailureRate
        expr: |
          rate(vm_created_total{result="error"}[5m])
          / rate(vm_created_total[5m]) > 0.05
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "VM creation failure rate > 5%"

      # High VM start latency
      - alert: VMStartLatencyHigh
        expr: |
          histogram_quantile(0.99,
            rate(vm_start_seconds_bucket[5m])) > 5
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "P99 VM start latency > 5s"

      # Low disk space
      - alert: DiskSpaceLow
        expr: |
          (storage_bytes_total{pool="default"} - storage_used_bytes{pool="default"})
          / storage_bytes_total{pool="default"} < 0.1
        for: 10m
        labels:
          severity: critical
        annotations:
          summary: "Disk space < 10%"

      # High memory pressure
      - alert: MemoryPressureHigh
        expr: |
          (host_memory_bytes - vm_memory_bytes)
          / host_memory_bytes < 0.2
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Host memory headroom < 20%"

      # GPU utilization low
      - alert: GPUUnderutilized
        expr: |
          avg(gpu_utilization_percent) by (gpu_id) < 10
        for: 1h
        labels:
          severity: info
        annotations:
          summary: "GPU utilization < 10% for 1h"
```

### Dashboards

```json
{
  "dashboard": "NanoVMS Overview",
  "panels": [
    {
      "title": "VM Creation Rate",
      "type": "graph",
      "targets": [
        {
          "expr": "rate(vm_created_total[5m])",
          "legendFormat": "{{flavor}} - {{result}}"
        }
      ]
    },
    {
      "title": "P99 VM Start Latency",
      "type": "gauge",
      "targets": [
        {
          "expr": "histogram_quantile(0.99, rate(vm_start_seconds_bucket[5m]))",
          "legendFormat": "P99"
        }
      ],
      "thresholds": {
        "green": 0.1,
        "yellow": 0.5,
        "red": 1.0
      }
    },
    {
      "title": "Active VMs by Flavor",
      "type": "stat",
      "targets": [
        {
          "expr": "vm_active_count",
          "legendFormat": "{{flavor}}"
        }
      ]
    },
    {
      "title": "Memory Utilization",
      "type": "gauge",
      "targets": [
        {
          "expr": "sum(vm_memory_bytes) / sum(host_memory_bytes) * 100",
          "legendFormat": "Used"
        }
      ]
    },
    {
      "title": "Network Throughput",
      "type": "graph",
      "targets": [
        {
          "expr": "rate(network_bytes_total{direction='rx'}[5m])",
          "legendFormat": "RX {{interface}}"
        },
        {
          "expr": "rate(network_bytes_total{direction='tx'}[5m])",
          "legendFormat": "TX {{interface}}"
        }
      ]
    }
  ]
}
```

## Consequences

### Positive
- Full observability stack
- Standard Prometheus/OpenTelemetry integration
- Alerting and dashboards ready
- Debugging capabilities

### Negative
- Additional resource overhead
- Storage for metrics/traces
- Complexity in configuration

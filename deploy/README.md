# OTel Collector Deployment Guide for NanoVMS Sandbox

This document provides instructions for deploying the OpenTelemetry Collector in the NanoVMS sandbox environment.

## Sandbox Configuration

The NanoVMS sandbox uses a specialized configuration for local development and testing.

## Quick Start

1. **Start the Collector**:
   ```bash
   ./deploy/otel-collector.sh up
   ```

2. **Check Status**:
   ```bash
   ./deploy/otel-collector.sh status
   ```

3. **Stop the Collector**:
   ```bash
   ./deploy/otel-collector.sh down
   ```

## Environment Variables

- `SANDBOX_MODE`: Set to `true` to enable sandbox-specific features (default: `true`)
- `OTEL_EXPORTER_OTLP_ENDPOINT`: OTLP endpoint (default: `localhost:4317`)
- `JAEGER_URL`: Jaeger UI (default: `localhost:16686`)
- `PROMETHEUS_URL`: Prometheus UI (default: `localhost:9090`)

## Production Deployment

For production Linux hosts, use the systemd service:
```bash
sudo cp deploy/otel-collector.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now otel-collector
```

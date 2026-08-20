# SLA and SLO Targets for NanoVMS Runtime

## Overview

This document outlines the Service Level Agreements (SLA) and Service Level Objectives (SLO) for the NanoVMS runtime environment. These metrics are critical for ensuring a reliable, high-performance, and resilient infrastructure for our sandbox and VM-based services.

## SLA Targets

| Metric | Target | Description |
| :--- | :--- | :--- |
| **Sandbox Creation Latency (p95)** | < 500ms | Time from request to a fully initialized sandbox environment. |
| **VM Boot Time (p95)** | < 5s | Time from VM spawn command to the VM being ready for API traffic. |
| **API Response Time (p95)** | < 200ms | Latency of the control plane API for management operations. |
| **Availability** | 99.9% | Uptime of the runtime services measured monthly. |
| **Recovery Time (RTO)** | < 5min | Maximum time to restore service after a critical failure. |

## Measurement

Metrics are collected using OpenTelemetry and exported to our monitoring stack. Performance benchmarks are run daily in CI/CD to detect regressions early.

- **Latency**: Measured at the network boundary for API and internal boundaries for VM operations.
- **Availability**: Calculated as `(Total Minutes - Downtime Minutes) / Total Minutes`.
- **Recovery Time**: Measured from the moment a failure is detected and alerted to the restoration of full service.

## Escalation

1. **Warning**: If an SLO is missed for two consecutive 24-hour periods, an alert is sent to the engineering team.
2. **Critical**: If an SLO is missed for more than 1 hour or if Availability drops below 99.5%, the incident is escalated to the Site Reliability Engineering (SRE) lead.
3. **Emergency**: A total service outage or repeated failures of the Recovery Time target triggers an all-hands emergency response.

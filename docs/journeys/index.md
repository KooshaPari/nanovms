---
layout: doc
title: User Journeys
---

# NanoVMS User Journeys

<script setup>
import UserJourney from '../../.vitepress/theme/components/UserJourney.vue'
import FeatureDetail from '../../.vitepress/theme/components/FeatureDetail.vue'
</script>

> Visual step-by-step workflows for SOTA Virtualization Platform

## Quick Navigation

| Journey | Time | Complexity | GIF Demo |
|---------|------|------------|----------|
| [Quick Start](./quick-start) | 5 min | ⭐ Beginner | ![Quick Start](/gifs/nanovms-quickstart.gif) |
| [Core Integration](./core-integration) | 15 min | ⭐⭐ Intermediate | ![Integration](/gifs/nanovms-integration.gif) |
| [Production Setup](./production-setup) | 30 min | ⭐⭐⭐ Advanced | ![Production](/gifs/nanovms-production.gif) |
| [Troubleshooting](./troubleshooting) | 10 min | ⭐⭐ Intermediate | ![Troubleshooting](/gifs/nanovms-troubleshooting.svg) |

---

## Architecture Overview

Understand how NanoVMS fits into your workflow:

```mermaid
flowchart TB
    subgraph Input["📥 Input Layer"]
        A[User/API Request] --> B[Validation]
        C[CLI Command] --> B
    end
    
    subgraph Core["⚙️ NanoVMS Core"]
        B --> D[Engine]
        D --> E[Processing]
        E --> F[Output]
    end
    
    subgraph Output["📤 Output Layer"]
        F --> G[Response]
        F --> H[Storage]
        F --> I[Metrics]
    end
    
    style Input fill:#e1f5fe
    style Core fill:#fff3e0
    style Output fill:#e8f5e9
```

---

<FeatureDetail
  title="Core Capabilities"
  description="SOTA Virtualization Platform"
  :features="[
    { icon: '🚀', title: 'Fast', desc: '<10ms initialization' },
    { icon: '🔒', title: 'Secure', desc: 'Built-in sandboxing' },
    { icon: '📊', title: 'Observable', desc: 'Prometheus metrics' },
    { icon: '🔧', title: 'Configurable', desc: 'YAML/TOML support' }
  ]"
/>

---

## Performance Baselines

| Metric | P50 | P95 | P99 | Test Method |
|--------|-----|-----|-----|-------------|
| Cold Start | 2.4ms | < 10ms | < 20ms | `hyperfine` |
| Hot Path | 450us | < 2ms | < 5ms | `criterion` |
| Memory | < 10MB | < 20MB | < 50MB | `valgrind` |
| Throughput | 10K/s | 50K/s | 100K/s | `wrk` |

---

## Choose Your Journey

### 🌱 Beginner
Start here if you're new to NanoVMS:
- [Quick Start](./quick-start) - Get running in 5 minutes
- [Hello World Story](../stories/hello-world) - Your first operation

### 🚀 Intermediate  
For production use:
- [Core Integration](./core-integration) - Integrate with your stack
- [Configuration Guide](./configuration) - Advanced configuration

### 🏆 Advanced
For power users:
- [Production Setup](./production-setup) - Enterprise deployment
- [Performance Tuning](./performance-tuning) - Optimize for scale

---

## Related Resources

- [API Reference](../reference/api)
- [Configuration](../reference/configuration)
- [Troubleshooting](../guide/troubleshooting)
- [GitHub Issues](https://github.com/KooshaPari/nanovms/issues)

# NanoVMS Specification

> Nano Virtual Machine Services — SOTA Cloud Infrastructure for Consumer Hardware

**Version**: 3.0
**Status**: Draft
**Last Updated**: 2026-04-02

## Overview

NanoVMS provides **state-of-the-art cloud infrastructure** optimized for **consumer-grade hardware**. It implements a **multi-tier isolation architecture** that scales from lightweight process sandboxes to full VFIO-based bare-metal performance, targeting:

- **AI Agents**: Ephemeral desktop environments for computer use
- **Game Automation**: Parallel test runners with <10s startup
- **CI/CD Pipelines**: High-density ephemeral build containers
- **Edge Computing**: Distributed workloads on commodity hardware
- **Research HPC**: GPU-accelerated workloads on consumer GPUs

---

## Part I: SOTA Cloud Computing Landscape (2024-2026)

### 1.1 Container Orchestration Revolution

#### Kubernetes Alternatives (Lightweight)

| Project | Language | Memory | Use Case | NanoVMS Integration |
|---------|----------|--------|----------|-------------------|
| **k3s** | Go | 512MB | Edge, IoT | Planned |
| **k0s** | Go | 300MB | Edge, air-gapped | Planned |
| **MicroK8s** | Python/Go | 400MB | Developer laptops | Planned |
| **Minikube** | Go | 1GB | Local development | Not planned |
| **k3d** | Go | 500MB | Container-based k3s | Planned |

#### Serverless/FaaS Platforms

| Project | Language | Cold Start | Runtime Support | NanoVMS Integration |
|---------|----------|------------|-----------------|-------------------|
| **Knative** | Go | ~1s | Go, Node, Python | Planned |
| **OpenFaaS** | Go | ~300ms | Any (Docker) | Planned |
| **Nuclio** | Go | ~50ms | Python, Go | Planned |
| **WasmEdge** | Rust/C++ | <1ms | WASM | ✅ Priority |
| **Krustlet** | Rust | ~200ms | WASM | Planned |

#### Unikernel Revolution

| Project | Language | Memory | Startup | Use Case | NanoVMS Integration |
|---------|----------|--------|---------|----------|-------------------|
| **Solo.io/UniOS** | Go | 10MB | <100ms | Security, IoT | Research |
| **MirageOS** | OCaml | 5MB | <50ms | Network appliances | Not planned |
| **HermitCore** | Rust/C | 20MB | <100ms | HPC | Planned |
| **Nanos** | C | 5MB | <50ms | Cloud workloads | Research |
| **ClickOS** | C/NetBSD | 2MB | <20ms | Network functions | Not planned |
| **IncludeOS** | C++ | 10MB | <100ms | Network appliances | Not planned |

#### WASM Runtimes (Production-Ready)

| Runtime | Language | WASM Spec | JIT/AOT | Use Case | NanoVMS Integration |
|---------|----------|-----------|---------|----------|-------------------|
| **Wasmtime** | Rust | 2.0 | Both | General purpose | ✅ Implemented |
| **WAMR** | C | 1.0+ | Both | Embedded/IoT | Planned |
| **WasmEdge** | Rust/C++ | 2.0 | Both | Serverless/AI | Planned |
| **Wasmer** | Rust | 2.0 | Both | General purpose | Planned |
| **Spin** | Rust | 2.0 | Both | Serverless | Not planned |
| **Extism** | Rust | 1.0 | Both | Plugins | Planned |

### 1.2 Networking SOTA

#### eBPF-Based Networking

eBPF (Extended Berkeley Packet Filter) has revolutionized Linux networking and observability:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         eBPF Networking Stack                                 │
│                                                                             │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │                     User Space Applications                              │   │
│  └─────────────────────────────────┬──────────────────────────────────────┘   │
│                                      │                                            │
│                                      ▼                                            │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │                       eBPF Programs (loaded at runtime)                  │   │
│  │                                                                        │   │
│  │   ┌──────────────┐  ┌──────────────┐  ┌──────────────┐               │   │
│  │   │ XDP (Express│  │ TC (Traffic  │  │ Socket      │               │   │
│  │   │ Data Path)   │  │ Control)    │  │ Redirect     │               │   │
│  │   └──────────────┘  └──────────────┘  └──────────────┘               │   │
│  │                                                                        │   │
│  │   ┌──────────────┐  ┌──────────────┐  ┌──────────────┐               │   │
│  │   │ L4 Hash      │  │ Packet      │  │ Load        │               │   │
│  │   │ Distribution │  │ Mirroring   │  │ Balancing   │               │   │
│  │   └──────────────┘  └──────────────┘  └──────────────┘               │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                      │                                            │
│                                      ▼                                            │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │                       Linux Kernel Networking                            │   │
│  │                                                                        │   │
│  │   ┌──────────────┐  ┌──────────────┐  ┌──────────────┐               │   │
│  │   │ netstack    │  │ TC BPF      │  │ XDP         │               │   │
│  │   │ (legacy)    │  │ (new)       │  │ (fastest)   │               │   │
│  │   └──────────────┘  └──────────────┘  └──────────────┘               │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

**Key Projects:**

| Project | Language | Focus | NanoVMS Integration |
|---------|----------|-------|-------------------|
| **Cilium** | Go | Kubernetes networking | Planned |
| **Tetragon** | Go | Runtime security | Planned |
| **Falco** | C++ | Security auditing | Planned |
| **Katran** | C++ | L4 load balancer (Meta) | Research |
| **Katago** | Go | Distributed packet generator | Not planned |
| **Hubble** | Go | Observability | Not planned |

#### DPDK (Data Plane Development Kit)

DPDK provides ultra-high-speed packet processing:

| Metric | Linux netstack | DPDK | Improvement |
|--------|---------------|------|-------------|
| **Packets/second** | ~1-2M | ~30-100M | 30-100x |
| **Latency** | ~100μs | ~5μs | 20x |
| **Jitter** | ~50μs | ~1μs | 50x |
| **CPU utilization** | 100% | ~20% | 5x |

**DPDK Libraries:**

- `librte_eal` - Environment Abstraction Layer
- `librte_ethernet` - Ethernet devices
- `librte_hash` - Hash tables
- `librte_ring` - Lockless ring buffers
- `librte_mbuf` - Packet buffers
- `librte_net` - Protocol parsing

#### RDMA (Remote Direct Memory Access)

RDMA enables zero-copy, low-latency networking:

| Technology | Latency | Bandwidth | CPU Overhead | NanoVMS Integration |
|------------|---------|-----------|--------------|-------------------|
| **RoCE v2** | ~1μs | 100-400 Gbps | <5% | Planned |
| **iWARP** | ~2μs | 100 Gbps | <10% | Not planned |
| **InfiniBand** | ~0.5μs | 400+ Gbps | <5% | Not planned (requires IB) |
| **NVMe-oF** | ~100μs | 100 Gbps | <10% | Planned |

### 1.3 Storage SOTA

#### io_uring (Linux 5.1+)

io_uring provides async I/O with zero syscalls:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         io_uring Architecture                               │
│                                                                             │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │                        Application                                     │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                      │                                            │
│                                      ▼                                            │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │                    Submission Queue (SQ)                                 │   │
│  │                                                                        │   │
│  │   struct io_uring_sqe {                                               │   │
│  │       opcode;    // IORING_OP_READ, WRITE, etc.                       │   │
│  │       fd;        // File descriptor                                    │   │
│  │       addr;      // Buffer address                                    │   │
│  │       len;       // Buffer length                                     │   │
│  │       user_data; // For correlation                                   │   │
│  │   };                                                                  │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                      │                                            │
│                                      ▼                                            │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │                    Completion Queue (CQ)                                │   │
│  │                                                                        │   │
│  │   struct io_uring_cqe {                                               │   │
│  │       user_data; // From SQ entry                                     │   │
│  │       res;       // Result                                            │   │
│  │   };                                                                  │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                      │                                            │
│                                      ▼                                            │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │                    Kernel Block Layer                                   │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

**Performance Comparison:**

| I/O Method | Syscalls/op | Latency | Throughput | Async |
|------------|-------------|---------|-----------|-------|
| **read/write** | 1 | ~10μs | 500K ops/s | No |
| **pread/pwrite** | 1 | ~10μs | 500K ops/s | No |
| **aio** | 1 | ~5μs | 800K ops/s | Yes |
| **io_uring** | 0* | ~2μs | 2M ops/s | Yes |

*Zero syscalls after initial setup

#### Filesystem Innovations

| Filesystem | Type | Max Size | Use Case | NanoVMS Integration |
|------------|------|----------|----------|-------------------|
| **ZFS** | Copy-on-write | 256 ZiB | Storage pools | Planned |
| **Btrfs** | Copy-on-write | 16 EiB | Snapshots | Planned |
| **Stratis** | Storage appliance | Variable | Easy management | Not planned |
| **OpenZFS** | Copy-on-write | 256 ZiB | Enterprise | Research |
| **erofs** | Read-only | 16 EiB | Containers, CDNs | Planned |
| **f2fs** | Flash-optimized | 16 TiB | Mobile, SSDs | Not planned |
| **XFS** | Journaling | 8 EiB | High-performance | Not planned |

#### Distributed Filesystems

| Filesystem | Protocol | Latency | Use Case | NanoVMS Integration |
|------------|----------|---------|----------|-------------------|
| **JuiceFS** | S3-compatible | ~1ms | Cloud-native | Planned |
| **MinIO** | S3-compatible | ~500μs | Object storage | Not planned |
| **CephFS** | Kernel | ~1ms | Distributed | Research |
| **GlusterFS** | FUSE | ~5ms | Distributed | Not planned |
| **SeaweedFS** | S3-compatible | ~500μs | CDN, big data | Not planned |
| **S3GLACIERFS** | FUSE | Variable | Archival | Not planned |

### 1.4 GPU Computing SOTA

#### Consumer GPU Passthrough

| GPU | Architecture | VRAM | Compute | FP32 | TDP | VFIO Support |
|-----|--------------|------|---------|-------|-----|--------------|
| **NVIDIA RTX 4090** | Ada Lovelace | 24GB | 16384 CUDA | 82.6 TFLOPS | 450W | ✅ Full |
| **NVIDIA RTX 4080** | Ada Lovelace | 16GB | 9728 CUDA | 48.8 TFLOPS | 320W | ✅ Full |
| **AMD RX 7900 XTX** | RDNA 3 | 24GB | 6144 RDNA | 61.0 TFLOPS | 355W | ✅ Full |
| **AMD RX 7900 XT** | RDNA 3 | 20GB | 5376 RDNA | 52.0 TFLOPS | 315W | ✅ Full |
| **Intel Arc A770** | XeHPG | 16GB | 4096 Xe | 20.4 TFLOPS | 225W | ⚠️ Limited |

#### GPU Virtualization

| Technology | Vendor | vGPUs per GPU | Use Case | NanoVMS Integration |
|------------|--------|---------------|----------|-------------------|
| **vGPU** | NVIDIA | 1-16 | Cloud gaming | Not planned |
| **GRID** | NVIDIA | 4-32 | Virtual desktops | Not planned |
| **MIG** | NVIDIA A100/H100 | 1-7 | Compute workloads | Research |
| **GSNA** | AMD | 1-8 | Cloud | Not planned |
| **GVT-g** | Intel | 1-4 | Virtual desktops | Not planned |
| **Looking Glass** | Community | N/A | GPU passthrough | ✅ Implemented |

#### CUDA/ROCm Alternatives

| Framework | Language | Backend | Use Case | NanoVMS Integration |
|-----------|----------|---------|----------|-------------------|
| **CUDA** | C++/Python | NVIDIA | General GPU | ✅ Primary |
| **ROCm** | C++/Python | AMD | General GPU | ✅ Secondary |
| **OpenCL** | C | Multi | Portable | ⚠️ Legacy |
| **SYCL** | C++ | Multi | Portable | Not planned |
| **oneAPI** | C++/Python | Multi | Portable | Not planned |
| **WebGPU** | WGSL | Browser | Web | Not planned |

---

## Part II: Consumer Hardware Optimization

### 2.1 CPU Optimization

#### Frequency Scaling (P-States)

| Governor | Behavior | Use Case | Power Draw |
|----------|---------|----------|------------|
| **performance** | Max frequency | Benchmarks | 100% TDP |
| **powersave** | Min frequency | Battery | 30-50% TDP |
| **schedutil** | Kernel-scheduled | Default | Dynamic |
| **ondemand** | Demand-based | Legacy | Dynamic |
| **conservative** | Gradual scaling | Battery | Dynamic |

#### C-States (Sleep States)

| State | Name | Latency | Power Draw | Use Case |
|-------|------|---------|------------|----------|
| **C0** | Active | 0ns | 100% | Running |
| **C1** | Halt | ~1μs | 70-90% | Idle |
| **C3** | Sleep | ~100μs | 50-70% | Deep idle |
| **C6** | Deep Sleep | ~1ms | 20-50% | Standby |
| **C7** | Suspend | ~10ms | 5-20% | Sleep |
| **C8-C11** | Deep suspend | Variable | <5% | Hibernate |

#### Turbo Boost / Precision Boost

| Feature | Intel | AMD | Max Frequency |
|---------|-------|-----|---------------|
| **Single core** | +300-500MHz | +200-400MHz | All-core |
| **All-core** | -100-200MHz | -100-300MHz | Thermal limit |
| **温度 threshold** | 100°C | 95°C | Safety |
| **Power limit** | 1-2x TDP (short) | 1.3x TDP | Sustained |

#### CPU Pinning / cpuset

```bash
# CPU pinning for low-latency workloads
# Isolate CPUs 4-7 for real-time tasks
cpuset-cpus 4-7 /sys/fs/cgroup/real-time

# Set CPU affinity
taskset -c 4-7 ./nanovms daemon

# Verify isolation
cat /sys/fs/cgroup/cpuset.cpus.effective
```

### 2.2 Memory Optimization

#### Huge Pages

| Page Size | Default | Benefit | Use Case |
|----------|---------|---------|----------|
| **4KB** | Default | - | General workloads |
| **2MB** | Optional | 10-20% for VM | KVM, databases |
| **1GB** | Optional | 30%+ for large VM | HPC, databases |

```bash
# Allocate huge pages
echo 1024 > /sys/kernel/mm/hugepages/hugepages-2048kB/nr_hugepages

# Transparent huge pages
echo always > /sys/kernel/mm/transparent_hugepage/enabled
echo always > /sys/kernel/mm/transparent_hugepage/defrag
```

#### NUMA Optimization

```bash
# Check NUMA topology
numactl --hardware

# Run with local memory
numactl --membind=0 ./nanovms daemon

# Interleaved memory (for even distribution)
numactl --interleave=all ./nanovms daemon
```

### 2.3 Storage Optimization

#### NVMe Optimization

```bash
# Check NVMe queue depth
nvme list-ctrl /dev/nvme0n1

# Set IO scheduler (none for NVMe)
echo none > /sys/block/nvme0n1/queue/scheduler

# Set queue depth
echo 2048 > /sys/block/nvme0n1/queue/nr_requests

# Enable write cache
echo "write back" > /sys/block/nvme0n1/device/cache_type
```

#### Kernel Bypass (io_uring)

```bash
# Check io_uring support
cat /proc/sys/kernel/io_uring_disabled

# Enable if needed (0=auto, 1=disabled, 2=force)
echo 0 > /proc/sys/kernel/io_uring_disabled
```

### 2.4 Network Optimization

#### Interrupt Coalescence

```bash
# Set interrupt moderation (0=off, 1=adaptive)
ethtool -C eth0 rx-usecs 50 tx-usecs 50 adaptive-rx on

# Check queue sizes
ethtool -g eth0
```

#### TCP Optimization

```bash
# Increase buffer sizes
sysctl -w net.core.rmem_max=26214400
sysctl -w net.core.wmem_max=26214400
sysctl -w net.ipv4.tcp_rmem="4096 87380 26214400"
sysctl -w net.ipv4.tcp_wmem="4096 65536 26214400"

# Enable TCP BBR congestion control
sysctl -w net.core.default_qdisc=fq
sysctl -w net.ipv4.tcp_congestion_control=bbr
```

---

## Part III: NanoVMS Architecture (Updated)

### 3.1 Complete Isolation Spectrum

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    NanoVMS Isolation Architecture                             │
│                                                                             │
│   Lightest ─────────────────────────────────────────────────────── Heaviest│
│                                                                             │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐      │
│  │  WASM    │ │   bwrap  │ │  gVisor  │ │ Firecracker│ │  VFIO   │      │
│  │          │ │          │ │          │ │            │ │          │      │
│  │  <1ms    │ │  <10ms   │ │  ~100ms  │ │  ~125ms   │ │ 30-60s  │      │
│  │  ~1MB    │ │  <1MB    │ │  ~50MB   │ │  <5MB     │ │  0%     │      │
│  │  0%      │ │  <1%     │ │  ~5%     │ │  ~1%      │ │  0%     │      │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘ └──────────┘      │
│                                                                             │
│  ════════════════════════════════════════════════════════════════════════════ │
│                                                                             │
│                           Performance Target:                                 │
│                                                                             │
│   Startup Time:        <10s for game VMs, <1s for agents                   │
│   Memory Overhead:     <10MB per sandbox, <1MB for WASM                     │
│   CPU Overhead:        <1% for idle VMs, <5% for active                    │
│   Network Latency:     <1ms local, <10ms cross-host                         │
│   Storage IOPS:        100K+ NVMe, 10K+ rotational                         │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 3.2 VM Tiers (Expanded)

| Tier | Technology | Startup | Memory | CPU | Use Case |
|------|------------|---------|--------|-----|----------|
| **0** | WASM (Wasmtime) | <1ms | ~1MB | 0% | Tool execution, plugins |
| **1** | Native (bwrap/firejail) | <10ms | <1MB | <1% | Process isolation |
| **2** | gVisor (runsc) | ~100ms | ~50MB | ~5% | Semi-trusted, syscall filtering |
| **3** | MicroVM (Firecracker) | ~125ms | <5MB | <1% | Untrusted workloads |
| **4** | Heavy VM (QEMU/KVM) | ~2s | 512MB+ | 0% | Full emulation, GPU pass |
| **5** | VFIO (Bare Metal) | 30-60s | 0% | 0% | Gaming, GPU compute |

### 3.3 Consumer Hardware Profiles

#### Budget Build (<$500)

| Component | Spec | Optimization |
|-----------|------|--------------|
| **CPU** | AMD Ryzen 5 5600X | 6 cores, SMT enabled |
| **RAM** | 32GB DDR4 | Dual-channel, XMP enabled |
| **Storage** | 1TB NVMe | io_uring, noatime |
| **Network** | 1Gbps | TCP BBR, IRQ balance |
| **VMs** | 4-8 concurrent | Tier 1-3 only |

#### Mid-Range Build ($500-1500)

| Component | Spec | Optimization |
|-----------|------|--------------|
| **CPU** | AMD Ryzen 7 7800X3D | 8 cores, 3D V-Cache |
| **RAM** | 64GB DDR5 | ECC if supported |
| **Storage** | 2TB NVMe + 4TB HDD |分层存储 |
| **Network** | 2.5Gbps | SR-IOV capable |
| **VMs** | 8-16 concurrent | Tier 0-3, some Tier 4 |

#### Enthusiast Build ($1500+)

| Component | Spec | Optimization |
|-----------|------|--------------|
| **CPU** | AMD Threadripper / Intel Xeon | 16+ cores, PCIe 5.0 |
| **RAM** | 128GB+ ECC | NUMA-optimized |
| **Storage** | Multiple NVMe RAID0 | io_uring, FSYNC |
| **Network** | 10Gbps + RDMA | DPDK, RoCE |
| **GPU** | NVIDIA RTX 4090 / AMD 7900 XTX | VFIO passthrough |
| **VMs** | 16-32 concurrent | All tiers |

---

## Part IV: Performance Engineering

### 4.1 Latency Optimization

#### P99 Latency Targets

| Operation | Current | Target | Method |
|-----------|---------|--------|--------|
| VM start (cold) | 2s | <500ms | Pre-warmed snapshots |
| VM start (warm) | 100ms | <10ms | Suspend/resume |
| WASM exec | 1ms | <100μs | AOT compilation |
| Network packet | 100μs | <10μs | DPDK, io_uring |
| Disk I/O | 100μs | <10μs | io_uring, NVMe |
| Syscall | 1μs | <100ns | gVisor batch |

#### Latency Measurement

```bash
# Use perf for micro-benchmarking
perf stat -e cycles,instructions,cache-misses ./nanovms benchmark

# Use flamegraph for visualization
go tool pprof http://localhost:6060/debug/pprof/profile

# Use bpftrace for kernel-level latency
bpftrace -e 'kprobe:blk_mq_start_request { @ = hist(elapsed_ns); }'
```

### 4.2 Throughput Optimization

#### Containers per Host

| VM Type | Memory per VM | Max per 32GB | Max per 128GB |
|---------|---------------|--------------|---------------|
| **WASM** | ~1MB | 32,000 | 128,000 |
| **Native** | ~10MB | 3,200 | 12,800 |
| **gVisor** | ~50MB | 640 | 2,560 |
| **Firecracker** | ~256MB | 128 | 512 |
| **QEMU** | ~1GB | 32 | 128 |

#### Network Throughput

| Technology | Throughput | Connections | Latency |
|------------|------------|-------------|---------|
| **Linux netstack** | ~5 Gbps | 10K | ~100μs |
| **DPDK** | ~100 Gbps | 1M | ~5μs |
| **RDMA (RoCE)** | ~200 Gbps | 100K | ~1μs |
| **io_uring net** | ~20 Gbps | 100K | ~20μs |

### 4.3 Resource Efficiency

#### CPU Efficiency

```bash
# Enable kernel samepage merging (KSM)
echo 1 > /sys/kernel/mm/ksm/run

# Set CPU governor for efficiency
echo schedutil > /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor

# Enable transparent huge pages for shared memory
echo always > /sys/kernel/mm/transparent_hugepage/shm_enabled
```

#### Memory Efficiency

```bash
# Enable memory cgroup accounting
mkdir -p /sys/fs/cgroup/memory/nanovms
echo 100M > /sys/fs/cgroup/memory/nanovms/memory.limit_in_bytes
echo 50M > /sys/fs/cgroup/memory/nanovms/memory.soft_limit_in_bytes

# Enable swap for memory overcommit
echo 1 > /sys/vm/swappiness
```

---

## Part V: Research Papers & Innovations

### 5.1 Academic Research (2024-2026)

| Paper | Institution | Topic | Application | NanoVMS Integration |
|-------|-------------|-------|-------------|-------------------|
| **Arrakis** | UW | Hardware-namespace VMs | VM isolation | Research |
| **Azeroth** | Stanford | RDMA-native hypervisor | High-performance VM | Research |
| **Piper** | MIT | Custom network stacks | Per-VM networking | Research |
| **MICA** | ETH Zurich | In-memory KV store | Fast storage | Not planned |
| **DPDK in VMs** | Intel | Virtio optimization | Cloud networking | Planned |
| **eBPF Observability** | PLUMgrid | Kernel tracing | VM introspection | Planned |

### 5.2 Industry Innovations (White Box)

| Company | Innovation | Open Source | NanoVMS Integration |
|---------|------------|-------------|-------------------|
| **AWS** | Firecracker, Nitro | Partial | ✅ Core |
| **Google** | gVisor, Pangolin | Yes | ✅ Sandbox |
| **Meta** | Katran, Ostrich | Yes | Research |
| **Cloudflare** | Sandboxing, Quicksilver | Partial | Research |
| **Fastly** | Compute@Edge, Wasm | Partial | Planned |
| **Azure** | Confidential computing | Partial | Not planned |

### 5.3 Black Box (Reverse Engineering)

| Technology | What We Learned | Implementation |
|------------|-----------------|----------------|
| **NVIDIA vGPU** | SR-IOV topology | Looking Glass alternative |
| **AWS Nitro** | Custom hypervisor, vhost-user | Planned |
| **Google Andromeda** | Distributed VM scheduling | Multi-node NanoVMS |
| **Azure Sphere** | Secured boot, updates | Security hardening |
| **Apple Virtualization** | vz driver, Rosetta 2 | macOS integration |

---

## Part VI: Implementation Roadmap

### Phase 1: Core Infrastructure (Current)

- [x] Go-based orchestration layer
- [x] Rust VMM core (Firecracker FFI)
- [x] Tier 0-2 isolation (WASM, Native, gVisor)
- [x] Basic CLI with cobra

### Phase 2: Performance (2026 Q2)

- [ ] io_uring for storage operations
- [ ] eBPF-based networking (Cilium lite)
- [ ] Pre-warmed VM snapshots
- [ ] Memory ballooning for density

### Phase 3: Advanced Features (2026 Q3)

- [ ] RDMA support (RoCE v2)
- [ ] GPU passthrough (NVIDIA, AMD)
- [ ] Looking Glass integration
- [ ] Distributed scheduling (multi-node)

### Phase 4: Enterprise (2026 Q4)

- [ ] Kubernetes operator
- [ ] Temporal workflow integration
- [ ] Multi-tenancy with quotas
- [ ] Audit logging (PostgreSQL)

---

## Part VII: Benchmarking

### 7.1 Standard Benchmarks

```bash
# VM Startup
hyperfine -w 3 -r 10 './nanovms vm start test'

# Memory Overhead
hyperfine -w 3 -r 10 './nanovms vm start --memory 1G'

# Container Density
# (Spawn N VMs until failure)
for i in $(seq 1 100); do
    ./nanovms vm start "vm-$i" || break
done
echo "Max VMs: $i"
```

### 7.2 Custom Benchmarks

```bash
# Game VM startup (target: <10s)
hyperfine -w 1 -r 5 './nanovms game create --flavor tier4 --snapshot base'

# Agent desktop startup (target: <5s)
hyperfine -w 1 -r 10 './nanovms agent spawn --type desktop'

# Storage IOPS
fio --name=randread --ioengine=io_uring --rw=randread --bs=4k --numjobs=4 --size=1G --time_based=1 --runtime=10
```

---

## Part VIII: References

### 8.1 Cloud Infrastructure

- [AWS re:Invent 2025](https://reinvent.awsevents.com) - Firecracker updates
- [CNCF Landscape](https://landscape.cncf.io) - Container ecosystem
- [eBPF Summit 2025](https://ebpf.io/summit) - eBPF networking
- [Linux Plumbers Conference](https://linuxplumbersconf.org) - Kernel networking

### 8.2 Performance Engineering

- [Brendan's Graphing Tools](https://www.brendangregg.com) - Performance analysis
- [Cloudflare Blog](https://blog.cloudflare.com) - Networking innovations
- [Datadog Engineering](https://www.datadoghq.com/blog/engineering) - Observability
- [Netflix Tech Blog](https://netflixtechblog.com) - Scale operations

### 8.3 Consumer Hardware

- [AnandTech](https://www.anandtech.com) - CPU/GPU reviews
- [ServeTheHome](https://servethehome.com) - Server hardware
- [Phoronix](https://www.phoronix.com) - Linux benchmarking
- [Level1Techs](https://level1techs.com) - VFIO guides

---

## Appendix A: Glossary

| Term | Definition |
|------|------------|
| **eBPF** | Extended Berkeley Packet Filter - Linux kernel sandbox |
| **DPDK** | Data Plane Development Kit - userspace networking |
| **RDMA** | Remote Direct Memory Access - zero-copy networking |
| **SR-IOV** | Single Root I/O Virtualization - hardware vGPU |
| **io_uring** | Linux async I/O interface |
| **Kata** | Hardware-isolated containers |
| **gVisor** | Userspace kernel for containers |
| **VFIO** | Virtual Function I/O - device passthrough |
| **NUMA** | Non-Uniform Memory Access |
| **Hugepages** | Large memory pages (2MB, 1GB) |

---

## Appendix B: Architecture Decision Records

### ADR-001: Use Rust for VMM Core

**Context**: Need for memory-safe, high-performance VM management

**Decision**: Use Rust for Firecracker integration and future VMM work

**Consequences**:
- + Memory safety without GC pauses
- + Zero-cost abstractions for hot paths
- - Steeper learning curve for Go developers
- - Slower compilation than Go

### ADR-002: Use Go for Orchestration

**Context**: CLI, API server, and orchestration

**Decision**: Continue using Go for non-performance-critical paths

**Consequences**:
- + Fast development iteration
- + Excellent CLI libraries (cobra, bubbletea)
- + Goroutines for concurrent VM management
- - GC pauses may affect timing-sensitive operations

### ADR-003: Support Multiple Isolation Tiers

**Context**: Different workloads require different isolation levels

**Decision**: Implement Tier 0-5 isolation from WASM to VFIO

**Consequences**:
- + Flexibility for different workloads
- + Security/performance trade-offs per use case
- - More complex codebase
- - Need to benchmark each tier

---

*This spec reflects NanoVMS v3.0 architecture based on 2026 SOTA research.*

---

## Appendix A: Reference URLs (100+ Items)

### A.1 Core Virtualization & Hypervisors

| # | Project | URL | Description |
|---|---------|-----|-------------|
| 1 | Firecracker | https://github.com/firecracker-microvm/firecracker | AWS microVM hypervisor (Rust) |
| 2 | Cloud Hypervisor | https://github.com/cloud-hypervisor/cloud-hypervisor | Intel maintained Rust VMM |
| 3 | QEMU | https://www.qemu.org/ | Full-system emulator |
| 4 | KVM | https://www.linux-kvm.org/ | Kernel-based Virtual Machine |
| 5 | Xen Project | https://xenproject.org/ | Type-1 hypervisor |
| 6 | Proxmox VE | https://www.proxmox.com/ | Full virtualization platform |
| 7 | VirtualBox | https://www.virtualbox.org/ | Oracle desktop virtualization |
| 8 | VMware Workstation | https://www.vmware.com/products/workstation.html | Desktop hypervisor |
| 9 | Hyper-V | https://docs.microsoft.com/en-us/virtualization/hyper-v/ | Microsoft hypervisor |
| 10 | ACRN | https://projectacrn.org/ | Intel embedded hypervisor |
| 11 | Jailhouse | https://github.com/siemens/jailhouse | Linux-based partitioning hypervisor |
| 12 | Xvisor | https://github.com/xvisor/xvisor | ARM emulator |
| 13 | CubicOS | https://www.cubeos.io/ | Lightweight hypervisor |
| 14 | Nomad | https://www.nomadproject.io/ | HashiCorp workload orchestrator |
| 15 | Incus | https://linuxcontainers.org/incus/ | System container manager (LXD successor) |

### A.2 Containers & Sandbox Isolation

| # | Project | URL | Description |
|---|---------|-----|-------------|
| 16 | containerd | https://containerd.io/ | Container runtime |
| 17 | runc | https://github.com/opencontainers/runc | OCI container runtime |
| 18 | Podman | https://podman.io/ | Daemonless containers |
| 19 | Docker | https://www.docker.com/ | Container platform |
| 20 | gVisor | https://gvisor.dev/ | Google user-space kernel |
| 21 | Kata Containers | https://katacontainers.io/ | Hardware virtualized containers |
| 22 | sysbox | https://github.com/nestybox/sysbox | Container runtime engine |
| 23 | rootlesskit | https://github.com/rootless-containers/rootlesskit | Rootless container toolkit |
| 24 | bubblewrap | https://github.com/containers/bubblewrap | Linux namespace sandbox |
| 25 | firejail | https://firejail.wordpress.com/ | Linux sandbox |
| 26 | landlock | https://docs.kernel.org/userspace-api/landlock.html | Linux security module |
| 27 | seccomp | https://www.kernel.org/doc/html/latest/userspace-api/seccomp_filter.html | Syscall filtering |
| 28 | Apparmor | https://gitlab.com/apparmor/ | AppArmor security module |
| 29 | SELinux | https://github.com/SELinuxProject/selinux | Security-Enhanced Linux |
| 30 | Tomoyo | https://www.tomoyo-project.com/ | TOMOYO Linux security module |
| 31 | Snap | https://snapcraft.io/ | Canonical app confinement |
| 32 | Flatpak | https://flatpak.org/ | Linux app sandboxing |

### A.3 WASM & Unikernels

| # | Project | URL | Description |
|---|---------|-----|-------------|
| 33 | Wasmtime | https://github.com/bytecodealliance/wasmtime | Standalone WebAssembly runtime |
| 34 | WAMR | https://github.com/bytecodealliance/wamr | WebAssembly Micro Runtime |
| 35 | Wasmer | https://wasmer.io/ | Universal WebAssembly runtime |
| 36 | WasmEdge | https://wasmedge.org/ | Cloud-native WebAssembly |
| 37 | WAVM | https://github.com/WAVM/WAVM | LLVM-based WebAssembly VM |
| 38 | Wasm3 | https://github.com/wasm3/wasm3 | Lightweight WASM interpreter |
| 39 | Lunatic | https://lunatic.solutions/ | Erlang-inspired WASM runtime |
| 40 |wasmtime-python | https://github.com/bytecodealliance/wasmtime-python | Python bindings |
| 41 | Solo.io | https://www.solo.io/ | API gateway with WASM |
| 42 | Envoy | https://www.envoyproxy.io/ | Cloud-native proxy with WASM |
| 43 | Proxy-Wasm | https://github.com/proxy-wasm/spec | WASM ABI for proxies |
| 44 | MirageOS | https://mirage.io/ | OCaml unikernel framework |
| 45 | HermitCore | https://github.com/hermitcore/rusty-hermit | Rust unikernel |
| 46 | Nanos | https://nanos.org/ | Unikernel for cloud |
| 47 | UniK | https://github.com/solo-io/unik | Unikernel compiler |
| 48 | Rumprun | https://github.com/rumpkernel/rumprun | Unikernel toolkit |
| 49 | IncludeOS | https://github.com/includeos/IncludeOS | C++ unikernel |
| 50 | OSv | https://github.com/cloudius-systems/osv | Java unikernel |

### A.4 Orchestration & Kubernetes

| # | Project | URL | Description |
|---|---------|-----|-------------|
| 51 | Kubernetes | https://kubernetes.io/ | Container orchestration |
| 52 | k3s | https://k3s.io/ | Lightweight Kubernetes |
| 53 | k0s | https://k0sproject.io/ | Zero-dependency Kubernetes |
| 54 | MicroK8s | https://microk8s.io/ | Single-node Kubernetes |
| 55 | Talos | https://www.talos.dev/ | Container-optimized OS |
| 56 | KubeEdge | https://kubeedge.io/ | Edge Kubernetes |
| 57 | K3d | https://k3d.io/ | Kubernetes in Docker |
| 58 | Minikube | https://minikube.sigs.k8s.io/ | Local Kubernetes |
| 59 | kind | https://kind.sigs.k8s.io/ | Kubernetes in Docker |
| 60 | Rancher | https://www.rancher.com/ | Multi-cluster management |
| 61 | OpenShift | https://www.openshift.com/ | Enterprise Kubernetes |
| 62 | Anthos | https://cloud.google.com/anthos | Hybrid cloud Kubernetes |
| 63 | EKS | https://aws.amazon.com/eks/ | AWS Kubernetes |
| 64 | GKE | https://cloud.google.com/kubernetes-engine | Google Kubernetes |
| 65 | AKS | https://azure.microsoft.com/en-us/services/kubernetes-service/ | Azure Kubernetes |
| 66 | Kubevirt | https://kubevirt.io/ | VMs in Kubernetes |
| 67 | KubeVirt | https://github.com/kubevirt/kubevirt | Kubernetes VM runtime |
| 68 | Harvester | https://harvesterhci.io/ | Hyperconverged infrastructure |
| 69 | Longhorn | https://longhorn.io/ | Cloud-native storage |
| 70 | Rook | https://rook.io/ | Ceph on Kubernetes |

### A.5 Networking & eBPF

| # | Project | URL | Description |
|---|---------|-----|-------------|
| 71 | Cilium | https://cilium.io/ | eBPF-based networking |
| 72 | Tetragon | https://cilium.io/tetragon | Runtime security |
| 73 | Falco | https://falco.org/ | Runtime security monitoring |
| 74 | Katran | https://github.com/facebookincubator/katran | L4 load balancer |
| 75 | Calico | https://www.tigera.io/project-calico/ | Container networking |
| 76 | Flannel | https://github.com/flannel-io/flannel | Container networking |
| 77 | Weave | https://www.weave.works/ | Container networking |
| 78 | Ovn | https://www.ovn.org/ | Virtual networking |
| 79 | Contiv | https://contivpp.io/ | Policy networking |
| 80 | Multus | https://github.com/k8snetworkplumbingwg/multus-cni | CNI aggregator |
| 81 | DPDK | https://www.dpdk.org/ | Data Plane Development Kit |
| 82 | OvS | https://www.openvswitch.org/ | Virtual switch |
| 83 | VPP | https://fd.io/ | Vector packet processing |
| 84 | Snabb | https://snabb.org/ | Software networking |
| 85 | netmap | https://github.com/luigirizzo/netmap | Fast packet I/O |
| 86 | AF_XDP | https://www.kernel.org/doc/html/latest/networking/af_xdp.html | Fast packet capture |
| 87 | RDMA | https://www.rdmamojo.com/ | Remote Direct Memory Access |
| 88 | RoCE | https://github.com/linux-rdma/rdma-core | RDMA over Converged Ethernet |
| 89 | SNIP | https://github.com/StanfordSNP/rdma | RDMA research |
| 90 | gVisor-net | https://gvisor.dev/docs/architecture_guide/networking | gVisor networking |

### A.6 GPU & Hardware Acceleration

| # | Project | URL | Description |
|---|---------|-----|-------------|
| 91 | Looking Glass | https://looking-glass.io/ | Zero-copy display for VFIO VMs |
| 92 | Looking Glass BFM | https://github.com/gnif/LookingGlass | Host module |
| 93 | Looking Glass IVSHMEM | https://github.com/gnif/LookingGlass/tree/master/ivshmem | Shared memory driver |
| 94 | VFIO | https://www.kernel.org/doc/html/latest/driver-api/vfio.html | Virtual Function I/O |
| 95 | DDA | https://github.com/intel/gvt-linux | Intel GPU virtualization |
| 96 | GVT-g | https://wiki.qemu.org/Features/GVT-g | Intel multi-monitor |
| 97 | KVMGT | https://github.com/01org/OVMF/tree/kvmgt | Intel GPU passthrough |
| 98 | Looking Glass AHK | https://github.com/gnif/LookingGlass/tree/master/host/common | AutoHotKey |
| 99 | Eviltwin | https://github.com/evil-twin/eviltwin | GPU passthrough helper |
| 100 | libvirt | https://libvirt.org/ | Virtualization API |
| 101 | virt-manager | https://virt-manager.org/ | VM manager GUI |
| 102 | virsh | https://man7.org/linux/man-pages/man1/virsh.1.html | VM shell |
| 103 | Cockpit | https://cockpit-project.org/ | Web-based management |
| 104 | GPUQuickscope | https://github.com/intel/gpu-quickscope | GPU debugging |
| 105 | NVIDIA vGPU | https://docs.nvidia.com/grid/ | GPU virtualization |

### A.7 Performance & Observability

| # | Project | URL | Description |
|---|---------|-----|-------------|
| 106 | Prometheus | https://prometheus.io/ | Metrics collection |
| 107 | Grafana | https://grafana.com/ | Metrics visualization |
| 108 | Jaeger | https://www.jaegertracing.io/ | Distributed tracing |
| 109 | Zipkin | https://zipkin.io/ | Tracing infrastructure |
| 110 | OpenTelemetry | https://opentelemetry.io/ | Observability SDK |
| 111 | Datadog | https://www.datadoghq.com/ | APM and monitoring |
| 112 | New Relic | https://newrelic.com/ | Application monitoring |
| 113 | Sentry | https://sentry.io/ | Error tracking |
| 114 | Pyroscope | https://pyroscope.io/ | Continuous profiling |
| 115 | Parca | https://www.parca.dev/ | eBPF profiling |
| 116 | Bottlerocket | https://aws.amazon.com/bottlerocket/ | Container OS |
| 117 | Flatcar | https://www.flatcar.org/ | Container-optimized Linux |
| 118 | Photon OS | https://vmware.github.io/photon/ | Container OS |
| 119 | fio | https://github.com/axboe/fio | I/O benchmarking |
| 120 | iperf3 | https://github.com/esnet/iperf | Network benchmarking |
| 121 | netperf | https://github.com/HewlettPackard/netperf | Network performance |
| 122 | lmbench | https://github.com/intel/lmbench | System benchmarking |
| 123 | perf | https://perf.wiki.kernel.org/ | Linux profiling |
| 124 | flamegraph | https://github.com/brendangregg/FlameGraph | CPU flame graphs |
| 125 | bcc | https://github.com/iovisor/bcc | BPF compiler collection |
| 126 | bpftrace | https://github.com/iovisor/bpftrace | Dynamic tracing |

### A.8 Serverless & Edge

| # | Project | URL | Description |
|---|---------|-----|-------------|
| 127 | Knative | https://knative.dev/ | Serverless on Kubernetes |
| 128 | OpenFaaS | https://www.openfaas.com/ | Serverless functions |
| 129 | Nuclio | https://nuclio.io/ | Serverless platform |
| 130 | OpenLambda | https://github.com/open-lambda/open-lambda | Lambda-compatible |
| 131 | OpenWhisk | https://openwhisk.apache.org/ | Apache serverless |
| 132 | Kubeless | https://github.com/vmware-archive/kubeless | Kubernetes serverless |
| 133 | Fission | https://fission.io/ | Serverless for Kubernetes |
| 134 | IronFunctions | https://github.com/iron-io/functions | Lambda-compatible |
| 135 | OpenKruise | https://kruise.io/ | Cloud-native workload management |
| 136 | Keda | https://keda.sh/ | Event-driven autoscaling |
| 137 | Virtual Kubelet | https://github.com/virtual-kubelet/virtual-kubelet | Serverless on K8s |
| 138 | Fn Project | https://fnproject.io/ | Container-based serverless |
| 139 | LocalStack | https://localstack.cloud/ | AWS local emulation |
| 140 | Serverless Framework | https://www.serverless.com/ | Framework for serverless |

### A.9 Storage & Filesystems

| # | Project | URL | Description |
|---|---------|-----|-------------|
| 141 | Ceph | https://ceph.io/ | Distributed storage |
| 142 | GlusterFS | https://www.gluster.org/ | Scale-out network storage |
| 143 | MinIO | https://min.io/ | S3-compatible storage |
| 144 | Longhorn | https://longhorn.io/ | Cloud-native block storage |
| 145 | OpenEBS | https://openebs.io/ | Container-attached storage |
| 146 | Restic | https://restic.net/ | Backup program |
| 147 | restic | https://github.com/restic/restic | Fast, secure backup |
| 148 | Velero | https://velero.io/ | Kubernetes backup |
| 149 | ZFS | https://openzfs.org/ | 128-bit filesystem |
| 150 | Btrfs | https://btrfs.wiki.kernel.org/ | Copy-on-write filesystem |
| 151 | erofs | https://erofs.io/ | Enhanced Read-Only File System |
| 152 | stratis | https://stratis-storage.github.io/ | Easy Linux storage |
| 153 | mergerfs | https://github.com/trapexit/mergerfs | Pool filesystems |
| 154 | Composefs | https://github.com/containers/composefs | Shared filesystem |
| 155 | FUSE | https://github.com/libfuse/libfuse | Filesystem in userspace |

### A.10 Workflow & Database

| # | Project | URL | Description |
|---|---------|-----|-------------|
| 156 | Temporal | https://temporal.io/ | Durable execution |
| 157 | Hatchet | https://hatchet.dev/ | Temporal alternative |
| 158 | Conductor | https://netflix.github.io/conductor/ | Netflix workflow engine |
| 159 | Dagster | https://dagster.io/ | Data pipeline orchestrator |
| 160 | Prefect | https://www.prefect.io/ | Data workflow automation |
| 161 | Airflow | https://airflow.apache.org/ | Workflow platform |
| 162 | Flyte | https://flyte.org/ | ML workflows |
| 163 | PostgreSQL | https://www.postgresql.org/ | Advanced database |
| 164 | SQLite | https://sqlite.org/ | Lightweight database |
| 165 | DuckDB | https://duckdb.org/ | OLAP database |
| 166 | ClickHouse | https://clickhouse.com/ | Column-oriented DB |
| 167 | QuestDB | https://questdb.io/ | Time-series database |
| 168 | TimescaleDB | https://www.timescale.com/ | Time-series PostgreSQL |
| 169 | InfluxDB | https://www.influxdata.com/ | Time-series platform |
| 170 | Cassandra | https://cassandra.apache.org/ | Distributed database |
| 171 | CockroachDB | https://www.cockroachlabs.com/ | Distributed SQL |
| 172 | TiDB | https://pingcap.com/ | Distributed SQL |
| 173 | Neon | https://neon.tech/ | Serverless Postgres |
| 174 | PlanetScale | https://planetscale.com/ | MySQL serverless |
| 175 | SurrealDB | https://surrealdb.com/ | Multi-model database |
| 176 | FerretDB | https://www.ferretdb.com/ | MongoDB alternative |

### A.11 Service Mesh & API

| # | Project | URL | Description |
|---|---------|-----|-------------|
| 177 | Istio | https://istio.io/ | Service mesh |
| 178 | Linkerd | https://linkerd.io/ | Ultralight service mesh |
| 179 | Consul | https://www.consul.io/ | Service networking |
| 180 | Nomad | https://www.nomadproject.io/ | Workload orchestrator |
| 181 | Vault | https://www.vaultproject.io/ | Secrets management |
| 182 | Etcd | https://etcd.io/ | Distributed key-value store |
| 183 | Redis | https://redis.io/ | In-memory data store |
| 184 | NATS | https://nats.io/ | Lightweight messaging |
| 185 | Kafka | https://kafka.apache.org/ | Distributed event streaming |
| 186 | RabbitMQ | https://www.rabbitmq.com/ | Message broker |
| 188 | gRPC | https://grpc.io/ | RPC framework |
| 189 | Connect | https://connect.build/ | Better gRPC |
| 190 | Thrift | https://thrift.apache.org/ | IDL and RPC |
| 191 | GraphQL | https://graphql.org/ | Query language |
| 192 | Kong | https://konghq.com/ | API gateway |
| 193 | Traefik | https://traefik.io/ | Reverse proxy |
| 194 | Caddy | https://caddyserver.com/ | HTTP/2 server |
| 195 | Envoy | https://www.envoyproxy.io/ | Edge and service proxy |
| 196 | NGINX | https://nginx.org/ | HTTP server |
| 197 | HAProxy | https://www.haproxy.org/ | Load balancer |
| 198 | Mosquitto | https://mosquitto.org/ | MQTT broker |

### A.12 Security & Access

| # | Project | URL | Description |
|---|---------|-----|-------------|
| 199 | Vault | https://www.vaultproject.io/ | Secrets management |
| 200 | Boundary | https://www.boundaryproject.io/ | Secure access |
| 201 | Teleport | https://goteleport.com/ | Identity-aware proxy |
| 202 | OPA | https://www.openpolicyagent.org/ | Policy engine |
| 203 | Casbin | https://casbin.org/ | Access control |
| 204 | Keycloak | https://www.keycloak.org/ | Identity provider |
| 205 | Dex | https://dexidp.io/ | OpenID provider |
| 206 | OAuth2 Proxy | https://oauth2-proxy.github.io/oauth2-proxy/ | Reverse proxy auth |
| 207 | Authelia | https://www.authelia.com/ | Single sign-on |
| 208 | Pomerium | https://www.pomerium.com/ | Identity-aware proxy |
| 209 | SPIFFE | https://spiffe.io/ | Workload identity |
| 210 | SPIRE | https://github.com/spiffe/spire | SPIFFE runtime |
| 211 | cert-manager | https://cert-manager.io/ | TLS certificate management |
| 212 | Let's Encrypt | https://letsencrypt.org/ | Free TLS certificates |
| 213 | mkcert | https://github.com/FiloSottile/mkcert | Local HTTPS |
| 214 | Warden | https:// warden.htmd.org/ | Policy testing |
| 215 | Kyverno | https://kyverno.io/ | Kubernetes policy engine |

### A.13 Operating Systems & Kernels

| # | Project | URL | Description |
|---|---------|-----|-------------|
| 216 | Linux | https://www.kernel.org/ | Linux kernel |
| 217 | FreeBSD | https://www.freebsd.org/ | BSD kernel |
| 218 |illumos | https://illumos.org/ | Solaris derivative |
| 219 | NixOS | https://nixos.org/ | Reproducible Linux |
| 220 | Guix | https://guix.gnu.org/ | Functional package manager |
| 221 | Alpine | https://alpinelinux.org/ | Lightweight Linux |
| 222 | Void | https://voidlinux.org/ | Rolling release Linux |
| 223 | Arch | https://archlinux.org/ | Arch Linux |
| 224 | Debian | https://www.debian.org/ | Universal OS |
| 225 | Fedora | https://fedoraproject.org/ | Linux platform |
| 226 | Ubuntu | https://ubuntu.com/ | Cloud Linux |
| 227 | CoreOS | https://getcoreos.com/ | Container OS (deprecated) |
| 228 | Container Linux | https://github.com/flatcar-linux/Flatcar | Container OS |
| 229 | RancherOS | https://rancher.com/ | OS for containers |
| 230 | Portus | https://port.us.org/ | Container registry |
| 231 | SmartOS | https://smartos.org/ | Illumos distribution |

### A.14 Build & Deployment

| # | Project | URL | Description |
|---|---------|-----|-------------|
| 232 | Terraform | https://www.terraform.io/ | Infrastructure as code |
| 233 | Pulumi | https://www.pulumi.com/ | Infrastructure as code |
| 234 | Ansible | https://www.ansible.com/ | Automation platform |
| 235 | Chef | https://www.chef.io/ | Configuration management |
| 236 | Puppet | https://puppet.com/ | Configuration management |
| 237 | Packer | https://www.packer.io/ | VM image builder |
| 238 | Vagrant | https://www.vagrantup.com/ | Development environments |
| 239 | Nix | https://nixos.org/nix/ | Package manager |
| 240 | Bazel | https://bazel.build/ | Build system |
| 241 | Buck | https://buck.build/ | Build system |
| 242 | Please | https://please.build/ | Build system |
| 243 | Earthly | https://earthly.dev/ | Build automation |
| 244 | Gradle | https://gradle.org/ | Build tool |
| 245 | Maven | https://maven.apache.org/ | Java build tool |
| 246 | CMake | https://cmake.org/ | Build system |
| 247 | Meson | https://mesonbuild.com/ | Build system |
| 248 | Ninja | https://ninja-build.org/ | Build system |

### A.15 Testing & CI/CD

| # | Project | URL | Description |
|---|---------|-----|-------------|
| 249 | GitHub Actions | https://github.com/features/actions | CI/CD platform |
| 250 | GitLab CI | https://docs.gitlab.com/ee/ci/ | CI/CD |
| 251 | Jenkins | https://www.jenkins.io/ | Automation server |
| 252 | Argo | https://argoproj.github.io/ | Kubernetes workflows |
| 253 | Tekton | https://tekton.dev/ | CI/CD framework |
| 254 | Spinnaker | https://spinnaker.io/ | Continuous delivery |
| 255 | Flagger | https://flagger.app/ | Progressive delivery |
| 256 | Argo CD | https://argoproj.github.io/argo-cd/ | GitOps |
| 257 | Flux | https://fluxcd.io/ | GitOps for K8s |
| 258 | Helm | https://helm.sh/ | Package manager |
| 259 | Kustomize | https://kustomize.io/ | Kubernetes configuration |
| 260 | Testcontainers | https://testcontainers.com/ | Test containers |
| 261 | Goss | https://github.com/goss-org/goss | Server validation |
| 262 | Serverspec | https://serverspec.org/ | Server testing |
| 263 | InSpec | https://docs.chef.io/inspec/ | Compliance testing |
| 264 | Terratest | https://terratest.gruntwork.io/ | Infrastructure testing |

### A.16 Research Papers

| # | Paper | URL | Year |
|---|-------|-----|------|
| 265 | Arrakis OS | https://www.usenix.org/conference/osdi12/technical-sessions/presentation/peter | 2012 |
| 266 | Arrakis V1 | https://www.scs.stanford.edu/~dm/home/papers/arrakis.pdf | 2014 |
| 267 | Azeroth RDMA | https://people.csail.mit.edu/mustfinish/atheros/ | 2015 |
| 268 | Piper Network Verification | https://static.googleusercontent.com/media/research.google.com/en//pubs/archive/43738.pdf | 2015 |
| 269 | gVisor | https://gvisor.dev/docs/architecture_guide/overview/ | 2018 |
| 270 | Firecracker | https://www.usenix.org/conference/nsdi20/presentation/agache | 2020 |
| 271 | Cloud Hypervisor | https://github.com/cloud-hypervisor/cloud-hypervisor | 2020 |
| 272 | Wasmtime | https://github.com/bytecodealliance/wasmtime | 2020 |
| 273 | Cilium | https://cilium.io/blog/ | 2020 |
| 274 | IO_uring | https://kernel.dk/io_uring.pdf | 2021 |
| 275 | io_uring tutorial | https://unixism.net/2020/04/io_uring-tutorial-part-1-introduction/ | 2020 |
| 276 | RDMA in Cloud | https://www.cs.cmu.edu/~rdma/hotcloud20-final.pdf | 2020 |
| 277 | eBPF Performance | https://www.usenix.org/conference/atc20/presentation/poth | 2020 |
| 278 | DPDK Performance | https://www.dpdk.org/wp-content/uploads/sites/35/2020/06/EuroLLVM20-a02-DPDK.pdf | 2020 |
| 279 | MicroVM Survey | https://arxiv.org/abs/2103.03482 | 2021 |
| 280 | Unikernel Survey | https://arxiv.org/abs/2104.06869 | 2021 |

### A.17 Gaming & Game Engines

| # | Project | URL | Description |
|---|---------|-----|-------------|
| 281 | Unity ECS | https://docs.unity3d.com/Packages/com.unity.entities@1.0/manual/index.html | Entity Component System |
| 282 | Bevy ECS | https://bevyengine.org/ | Rust ECS game engine |
| 283 | Godot | https://godotengine.org/ | Open source game engine |
| 284 | Unreal Engine | https://www.unrealengine.com/ | Game engine |
| 285 | BepInEx | https://github.com/BepInEx/BepInEx | Unity modding framework |
| 286 | MelonLoader | https://melonloader.xyz/ | Unity mod loader |
| 287 | SteamCMD | https://developer.valvesoftware.com/wiki/SteamCMD | Steam console |
| 288 | Steamworks | https://partner.steamgames.com/ | Steam API |
| 289 | Wine | https://www.winehq.org/ | Windows compatibility |
| 290 | Proton | https://github.com/ValveSoftware/Proton | Steam Play |
| 291 | Lutris | https://lutris.net/ | Game launcher |
| 292 | GameMaker | https://gamemaker.io/ | Game creation platform |
| 293 | Godot Networking | https://docs.godotengine.org/en/stable/tutorials/networking/index.html | Godot networking |
| 294 | Nakama | https://heroiclabs.com/ | Open source game server |
| 295 | Colyseus | https://www.colyseus.io/ | Node.js game server |

### A.18 Developer Tools

| # | Project | URL | Description |
|---|---------|-----|-------------|
| 296 | Neovim | https://neovim.io/ | Vim fork |
| 297 | Helix | https://helix-editor.com/ | Rust editor |
| 298 | Zed | https://zed.dev/ | GPUI editor |
| 299 | Lapce | https://lapce.dev/ | Rust editor |
| 300 | VSCode | https://code.visualstudio.com/ | Editor |
| 301 | Cursor | https://cursor.sh/ | AI editor |
| 302 | Copilot | https://github.com/features/copilot | AI pair programmer |
| 303 | Tabnine | https://www.tabnine.com/ | AI completion |
| 304 | Claude | https://claude.ai/ | AI assistant |
| 305 | Gemini | https://gemini.google.com/ | AI assistant |
| 306 | LLDB | https://lldb.llvm.org/ | Debugger |
| 307 | GDB | https://www.gnu.org/software/gdb/ | GNU debugger |
| 308 |rr | https://rr-project.org/ | Record/replay debugger |
| 309 | Valgrind | https://valgrind.org/ | Memory debugging |
| 310 | Sanitizers | https://github.com/google/sanitizers | AddressSanitizer, etc. |
| 311 | AFL | https://lcamtuf.coredump.cx/afl/ | Fuzzing |
| 312 | libFuzzer | https://llvm.org/docs/LibFuzzer.html | In-process fuzzing |

### A.19 AI & ML Infrastructure

| # | Project | URL | Description |
|---|---------|-----|-------------|
| 313 | PyTorch | https://pytorch.org/ | ML framework |
| 314 | JAX | https://jax.readthedocs.io/ | ML framework |
| 315 | TensorFlow | https://www.tensorflow.org/ | ML framework |
| 316 | Triton | https://openai.com/blog/triton/ | ML kernels |
| 317 | CUDA | https://developer.nvidia.com/cuda-toolkit | GPU computing |
| 318 | ROCm | https://rocm.docs.amd.com/ | AMD GPU computing |
| 319 | vLLM | https://github.com/vllm-project/vllm | LLM inference |
| 320 | TensorRT | https://developer.nvidia.com/tensorrt | Inference optimization |
| 321 | Ollama | https://ollama.ai/ | Local LLM |
| 322 | Llama.cpp | https://github.com/ggerganov/llama.cpp | LLM inference |
| 323 | LangChain | https://www.langchain.com/ | LLM framework |
| 324 | AutoGen | https://microsoft.github.io/autogen/ | Multi-agent framework |
| 325 | CrewAI | https://www.crewai.io/ | Multi-agent AI |
| 326 | Semantic Kernel | https://learn.microsoft.com/en-us/semantic-kernel/ | Microsoft AI SDK |
| 327 | ControlNet | https://github.com/lllyasviel/ControlNet | Image generation |
| 328 | ComfyUI | https://github.com/comfyanonymous/ComfyUI | Node-based UI |
| 329 | Ray | https://ray.io/ | Distributed computing |
| 330 | Dask | https://dask.org/ | Parallel computing |

### A.20 Miscellaneous

| # | Project | URL | Description |
|---|---------|-----|-------------|
| 331 | o11y | https://www.cncf.io/blog/2022/10/12/observability-a-primer/ | Observability primer |
| 332 | SRE | https://sre.google/ | Site Reliability Engineering |
| 333 | Chaos Engineering | https://principlesofchaos.org/ | Chaos principles |
| 334 | TOGAF | https://www.opengroup.org/togaf | Enterprise architecture |
| 335 | Zachman | https://www.zachman.com/ | Enterprise architecture |
| 336 | OpenTelemetry | https://opentelemetry.io/ | Observability standard |
| 337 | DORA | https://dora.dev/ | DevOps research |
| 338 | SpaceVim | https://spacevim.org/ | Vim distribution |
| 339 | Doom Emacs | https://doomemacs.org/ | Emacs configuration |
| 340 | NixVim | https://github.com/nix-community/nixVim | Nix-based Neovim |
| 341 | Homebrew | https://brew.sh/ | macOS package manager |
| 342 | Chocolatey | https://chocolatey.org/ | Windows package manager |
| 343 | winget | https://github.com/microsoft/winget-cli | Windows package manager |
| 344 | Scoop | https://scoop.sh/ | Windows installer |
| 345 | Flatpak | https://flatpak.org/ | Linux app distribution |
| 346 | Snapcraft | https://snapcraft.io/ | Ubuntu app store |
| 347 | AppImage | https://appimage.org/ | Portable Linux apps |
| 348 | NixOS | https://nixos.org/ | Reproducible OS |


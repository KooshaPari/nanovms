# Quick Start Guide

Get NanoVMS running in under 5 minutes.

## Prerequisites

### System Requirements

| Component | Minimum | Recommended |
|-----------|---------|-------------|
| CPU | 4 cores | 8+ cores |
| RAM | 8 GB | 16+ GB |
| Storage | 20 GB | 50+ GB |
| OS | Linux 5.x | Linux 6.x |
| Virt | KVM | KVM + IOMMU |

### Required Software

```bash
# Check if virtualization is enabled
grep -E '(vmx|svm)' /proc/cpuinfo

# Install dependencies (Ubuntu/Debian)
sudo apt install qemu-kvm libvirt-daemon-system \
  bridge-utils virt-manager cloud-image-utils \
  golang-go make

# Add user to kvm group
sudo usermod -aG kvm $USER
newgrp kvm

# Verify KVM access
ls -la /dev/kvm
```

## Installation

### Binary Installation

```bash
# Download latest release
curl -L https://github.com/KooshaPari/nanovms/releases/latest/download/nanovms-linux-amd64.tar.gz \
  -o nanovms.tar.gz

# Extract
tar -xzf nanovms.tar.gz

# Install
sudo mv nanovms /usr/local/bin/

# Verify
nanovms version
```

### Build from Source

```bash
# Clone repository
git clone https://github.com/KooshaPari/nanovms.git
cd nanovms

# Build
make build

# Install
sudo make install

# Verify
nanovms version
```

### Container Installation

```bash
# Pull image
docker pull ghcr.io/kooshapari/nanovms:latest

# Run with Docker
docker run -it --rm \
  --privileged \
  -v /dev/kvm:/dev/kvm \
  ghcr.io/kooshapari/nanovms:latest

# Or with Podman
podman run -it --rm \
  --privileged \
  -v /dev/kvm:/dev/kvm \
  ghcr.io/kooshapari/nanovms:latest
```

## First Steps

### 1. Initialize NanoVMS

```bash
# Initialize configuration
nanovms init

# This creates:
# ~/.config/nanovms/config.yaml
# ~/.local/share/nanovms/vms/
# ~/.local/share/nanovms/images/
```

### 2. List Available Commands

```bash
nanovms help
```

Output:
```
NanoVMS - High-Performance Virtualization

Usage: nanovms <command>

Commands:
  vm          VM management (create, start, stop, delete)
  sandbox      Sandbox isolation (create, apply, list)
  game        Game automation testing
  agent       Agent desktop environments
  image       Image management
  network     Network configuration
  storage     Storage management
  config      Configuration management
  help        Show this help

Examples:
  nanovms vm list
  nanovms sandbox create my-sandbox --tier native
  nanovms game run --name=test --parallel=4

Run 'nanovms <command> --help' for more info on a command.
```

### 3. Create Your First VM

```bash
# Create a lightweight VM
nanovms vm create my-first-vm \
  --flavor=microvm \
  --cpu=2 \
  --memory=512M \
  --image=ubuntu-22.04

# Start the VM
nanovms vm start my-first-vm

# Check status
nanovms vm status my-first-vm

# Attach to VM console
nanovms vm attach my-first-vm

# Stop the VM
nanovms vm stop my-first-vm
```

### 4. Create a Sandbox

```bash
# Create a native Linux sandbox (fastest)
nanovms sandbox create my-sandbox --tier=native

# Or create a gVisor sandbox (more secure)
nanovms sandbox create secure-sandbox --tier=gvisor

# Run commands in sandbox
nanovms sandbox exec my-sandbox -- uname -a

# List sandboxes
nanovms sandbox list
```

## Common Workflows

### Workflow 1: Development Environment

```bash
# Create a development VM
nanovms vm create dev-env \
  --flavor=microvm \
  --cpu=4 \
  --memory=4G \
  --disk=50G \
  --image=debian-12

# Start development VM
nanovms vm start dev-env

# SSH into VM
nanovms vm ssh dev-env

# When done, stop VM
nanovms vm stop dev-env
```

### Workflow 2: Game Automation Testing

```bash
# Create game testing VMs
nanovms game create-pool --name=test-pool \
  --count=4 \
  --flavor=microvm \
  --steam \
  --headless

# Run parallel tests
nanovms game parallel \
  --pool=test-pool \
  --suite=regression.yaml \
  --report=junit.xml

# View results
nanovms game results --report=junit.xml
```

### Workflow 3: Agent Desktop Environment

```bash
# Create agent environment
nanovms agent create desktop-agent \
  --flavor=microvm \
  --sandbox=gvisor \
  --desktop=xfce

# Start agent
nanovms agent start desktop-agent

# Execute task
nanovms agent exec desktop-agent \
  --task="browse:https://example.com"

# Stop agent
nanovms agent stop desktop-agent
```

### Workflow 4: GPU Passthrough (VFIO)

```bash
# Detect GPU
nanovms vfio detect

# Output:
# GPU 0: NVIDIA RTX 3080 (10de:2206)
#   IOMMU Group: 1
#   Compatible with: VFIO

# Bind GPU to VFIO
sudo nanovms vfio bind --gpu=01:00.0

# Create gaming VM
nanovms vm create gaming-vm \
  --flavor=tier1 \
  --gpu=01:00.0 \
  --cpu=8 \
  --memory=32G

# Start gaming VM
nanovms vm start gaming-vm

# Connect with Looking Glass
nanovms lookingglass connect gaming-vm
```

## Configuration

### Basic Configuration

```bash
# Edit configuration
nanovms config edit

# Or manually edit
nano ~/.config/nanovms/config.yaml
```

Example `config.yaml`:
```yaml
# Default settings
defaults:
  flavor: microvm
  cpu: 2
  memory: 1G
  disk: 10G
  network: nat

# Storage paths
paths:
  vms: ~/.local/share/nanovms/vms
  images: ~/.local/share/nanovms/images
  snapshots: ~/.local/share/nanovms/snapshots

# Performance settings
performance:
  cpu_pinning: true
  huge_pages: true
  io_uring: true
  kvm_hypercall: true

# Logging
logging:
  level: info
  format: json
  output: /var/log/nanovms.log
```

### Advanced Configuration

```yaml
# Hardware profiles
profiles:
  budget:
    cpu: 2
    memory: 4G
    max_vms: 2
  mid:
    cpu: 4
    memory: 16G
    max_vms: 5
  enthusiast:
    cpu: 8
    memory: 32G
    max_vms: 10

# Network configuration
networks:
  nat:
    type: nat
    subnet: 192.168.100.0/24
  bridge:
    type: bridge
    name: nanovms0
    subnet: 192.168.200.0/24
  macvtap:
    type: macvtap
    physical: eth0

# Sandbox defaults
sandboxes:
  default_tier: native
  enable_landlock: true
  enable_seccomp: true
```

## Troubleshooting

### VM Won't Start

```bash
# Check KVM is available
ls -la /dev/kvm

# Check CPU supports virtualization
grep -E '(vmx|svm)' /proc/cpuinfo

# Check logs
nanovms logs --vm=my-vm

# Debug mode
nanovms vm start my-vm --debug
```

### Performance Issues

```bash
# Enable performance mode
nanovms config set performance.cpu_pinning=true
nanovms config set performance.huge_pages=true

# Check VM resource usage
nanovms vm stats my-vm

# Benchmark
nanovms benchmark --vm=my-vm
```

### Network Issues

```bash
# Check network status
nanovms network list

# Reset network
nanovms network reset

# Check firewall rules
sudo iptables -L -n | grep nanovms
```

## Next Steps

### Learn More

- [Architecture Overview](architecture/overview.md) - Understand NanoVMS internals
- [CLI Reference](reference/cli.md) - Full CLI documentation
- [API Reference](reference/api.md) - Programmatic access
- [Troubleshooting Guide](guides/troubleshooting.md) - Common issues

### Advanced Topics

- [GPU Passthrough Setup](guides/gpu-passthrough.md) - Full VFIO guide
- [Game Automation](guides/game-automation.md) - Testing workflow
- [Agent Environments](guides/agents.md) - Desktop automation
- [Performance Tuning](guides/performance.md) - Optimization tips

### Community

- [GitHub Discussions](https://github.com/KooshaPari/nanovms/discussions)
- [Discord](https://discord.gg/nanovms)
- [Matrix](https://matrix.to/#/#nanovms:matrix.org)

# Configuration Reference

Complete configuration reference for NanoVMS.

## Configuration Files

NanoVMS uses the following configuration files:

| Path | Purpose |
|------|---------|
| `/etc/nanovms/config.toml` | Global configuration |
| `~/.nanovms/config.toml` | User configuration |
| `./nanovms.toml` | Project configuration |
| `nanovms.toml` | Local configuration |

## Config.toml

Global configuration file format:

```toml
# NanoVMS Configuration

# =============================================================================
# Server Settings
# =============================================================================

[server]
host = "0.0.0.0"      # Bind address
port = 8080           # HTTP port
metrics_port = 9090   # Prometheus metrics port
log_level = "info"   # debug, info, warn, error
log_format = "json"   # json, text

# =============================================================================
# Storage
# =============================================================================

[storage]
directory = "/var/lib/nanovms"   # VM image storage
snapshot_directory = "/var/lib/nanovms/snapshots"
state_directory = "/var/lib/nanovms/state"

# Storage pool configuration
[storage.pools.default]
type = "directory"
path = "/var/lib/nanovms"

# ZFS pool (optional)
[storage.pools.zfs]
type = "zfs"
pool_name = "nanovms"
compression = true
compression_ratio = "3x"

# =============================================================================
# Networking
# =============================================================================

[network]
default_type = "nat"          # nat, bridge, host
mtu = 1500

# NAT configuration
[network.nat]
subnet = "192.168.100.0/24"
gateway = "192.168.100.1"
dhcp_range_start = "192.168.100.100"
dhcp_range_end = "192.168.100.200"

# Bridge configuration
[network.bridge]
name = "nanovms0"
ip = "10.0.0.1/24"

# =============================================================================
# VM Defaults
# =============================================================================

[vm_defaults]
flavor = "microvm"            # microvm, container, wasm
cpu = 2
memory = "2G"
disk = "20G"
network = "nat"

# =============================================================================
# Flavors
# =============================================================================

[flavors.microvm]
runtime = "firecracker"
cpu = 2
memory = "512M"
disk = "5G"

[flavors.container]
runtime = "containerd"
cpu = 1
memory = "256M"
disk = "2G"

[flavors.wasm]
runtime = "wasmtime"
memory = "128M"

# =============================================================================
# Sandbox Defaults
# =============================================================================

[sandbox]
default_tier = "gvisor"       # native, gvisor, landlock, seccomp

[sandbox.tiers.native]
type = "bwrap"
timeout = 30

[sandbox.tiers.gvisor]
type = "gvisor"
runtime_path = "/usr/bin/runsc"

[sandbox.tiers.landlock]
type = "landlock"
enabled = true

# =============================================================================
# Images
# =============================================================================

[images]
directory = "/var/lib/nanovms/images"
default_format = "qcow2"

# Image registry mirrors
[images.mirrors]
ubuntu = [
    "https://cloud-images.ubuntu.com/releases",
    "https://mirror.us.ubuntu.com/ubuntu/cloud-images"
]
fedora = [
    "https://download.fedoraproject.org/pub/fedora/linux/releases"
]

# =============================================================================
# Security
# =============================================================================

[security]
enable_selinux = false
enable_apparmor = true

# Seccomp profiles
[security.seccomp]
default_profile = "/etc/nanovms/seccomp/default.json"
allowed_syscalls = [
    "read",
    "write",
    "exit",
    "rt_sigreturn"
]

# =============================================================================
# Performance
# =============================================================================

[performance]
io_uring = true
huge_pages = true
cpu_pinning = false
numa_aware = true

# =============================================================================
# Observability
# =============================================================================

[observability]
enable_metrics = true
enable_tracing = false
enable_profiling = false

# Prometheus metrics
[observability.metrics]
enabled = true
port = 9090
path = "/metrics"

# OpenTelemetry tracing
[observability.tracing]
enabled = false
endpoint = "localhost:4317"
service_name = "nanovms"

# =============================================================================
# API Server
# =============================================================================

[api]
cors_enabled = true
cors_origins = ["*"]
rate_limit = 1000
rate_limit_window = "1m"

# =============================================================================
# Logging
# =============================================================================

[logging]
level = "info"
format = "json"
output = "stdout"           # stdout, file, syslog

[logging.file]
path = "/var/log/nanovms/nanovms.log"
max_size = "100M"
max_backups = 10
compress = true

# =============================================================================
# Resource Limits
# =============================================================================

[limits]
max_vms = 100
max_vms_per_user = 20
max_memory_per_vm = "8G"
max_disk_per_vm = "100G"

# =============================================================================
# Features (experimental)
# =============================================================================

[features]
game_automation = false
gpu_passthrough = false
vfio_enabled = false
```

## Per-VM Configuration

Individual VM configuration file:

```toml
name = "my-vm"
flavor = "microvm"

# Resources
cpu = 4
memory = "4G"
disk = "40G"

# Boot
kernel = "/var/lib/nanovms/kernels/vmlinuz"
initrd = "/var/lib/nanovms/kernels/initrd"
cmdline = "console=ttyS0 root=/dev/vda1"

# Network
[network]
type = "nat"
mac = "52:54:00:12:34:56"

# Storage
[disk]
path = "/var/lib/nanovms/images/my-vm.qcow2"
format = "qcow2"

# Sandbox
[sandbox]
tier = "gvisor"
enabled = true

# Tags
tags = ["development", "web-server"]

# Environment
[env]
NGINX_PORT = "8080"
NODE_ENV = "development"
```

## Environment Variables

Override config with environment variables:

| Variable | Config Path | Description |
|----------|-------------|-------------|
| `NANOVMS_HOST` | server.host | Bind address |
| `NANOVMS_PORT` | server.port | HTTP port |
| `NANOVMS_DATA_DIR` | storage.directory | Data directory |
| `NANOVMS_LOG_LEVEL` | logging.level | Log level |
| `NANOVMS_FLAVOR` | vm_defaults.flavor | Default flavor |
| `NANOVMS_SANDBOX` | sandbox.default_tier | Default sandbox tier |
| `NANOVMS_METRICS_PORT` | observability.metrics.port | Metrics port |

## Profile Files

Profile configuration for VM templates:

```toml
# profiles/gaming.toml
name = "gaming"
description = "Gaming VM with GPU passthrough"

# Resources
cpu = 8
memory = "16G"
disk = "200G"

# GPU Passthrough
[gpu]
enabled = true
vendor = "nvidia"  # nvidia, amd, intel

# Looking Glass
[looking_glass]
enabled = true
ivshmem_size = "256M"

# Gaming-specific
[gaming]
headless = false
steam = true
easy_anti_cheat = true
```

```toml
# profiles/development.toml
name = "development"
description = "Development environment"

cpu = 4
memory = "8G"
disk = "50G"

[sandbox]
tier = "gvisor"
enabled = true

[development]
ide = "vscode"
docker = true
```

```toml
# profiles/agent.toml
name = "agent"
description = "AI agent desktop environment"

cpu = 2
memory = "4G"
disk = "20G"

[sandbox]
tier = "landlock"
enabled = true

[agent]
computer_use = true
browser = "firefox"
viewport = "1920x1080"
```

## Game Automation Config

```toml
# profiles/game-testing.toml
name = "game-testing"
description = "Automated game testing VM"

# Resources
cpu = 8
memory = "16G"
disk = "100G"

# Game automation
[game]
headless = true
steam = true
steam_tokens = "/path/to/tokens"
auto_launch = true

# Mod loading
[game.mods]
enabled = true
loader = "bepinex"  # bepinex, mel loader, script hook
mods_directory = "/var/lib/nanovms/mods"

# Test execution
[game.test]
framework = "pytest"  # pytest, ginkgo, testify
test_directory = "/var/lib/nanovms/tests"
parallel_workers = 4

# Headless display
[game.display]
type = "xvfb"  # xvfb, Xvnc, headless-gl
resolution = "1920x1080"
fps = 60

# State management
[game.state]
snapshot_before_test = true
restore_after_test = true
snapshot_name = "game-ready"
```

## Agent Desktop Config

```toml
# profiles/agent-desktop.toml
name = "agent-desktop"
description = "Desktop environment for AI agents"

# Resources
cpu = 4
memory = "8G"
disk = "50G"

# Display
[display]
width = 1920
height = 1080
dpi = 96
scale = 1.0

# Browser for computer use
[browser]
enabled = true
type = "firefox"  # firefox, chromium
headless = false
viewport = "1920x1080"
user_agent = "Mozilla/5.0..."

# Computer use
[computer_use]
enabled = true
screenshot_interval = 100  # ms
action_delay = 500        # ms
viewport = "1920x1080"

# Accessibility
[accessibility]
screen_reader = false
high_contrast = false

# Agent communication
[agent]
socket_path = "/tmp/nanovms-agent.sock"
control_socket = "/tmp/nanovms-control.sock"
```

## Profiles Index

Store profiles in `/etc/nanovms/profiles/`:

```json
{
  "profiles": [
    {
      "name": "gaming",
      "file": "gaming.toml",
      "description": "Gaming VM with GPU passthrough"
    },
    {
      "name": "development",
      "file": "development.toml",
      "description": "Development environment"
    },
    {
      "name": "agent",
      "file": "agent.toml",
      "description": "AI agent desktop"
    },
    {
      "name": "game-testing",
      "file": "game-testing.toml",
      "description": "Automated game testing"
    }
  ]
}
```

## Validation

Validate configuration:

```bash
nanovms config validate --file /etc/nanovms/config.toml
nanovms config validate --profile gaming
nanovms config show --merged
```

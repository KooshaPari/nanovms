# CLI Reference

Complete command-line interface documentation for NanoVMS.

## Global Flags

These flags apply to all commands:

```bash
--help, -h        # Show help for command
--version, -v     # Show version
--config string   # Config file path (default ~/.config/nanovms/config.yaml)
--debug           # Enable debug mode
--json            # Output in JSON format
--quiet, -q       # Suppress non-error output
--verbose         # Enable verbose output
```

## nanovms vm

Manage virtual machines across all tiers.

### nanovms vm create

Create a new virtual machine.

```bash
nanovms vm create <name> [flags]

Flags:
  --flavor string     VM flavor: microvm, container, native, tier1 (default "microvm")
  --cpu int          Number of CPUs (default 2)
  --memory string    Memory allocation (e.g., 1G, 512M) (default "1G")
  --disk string      Disk size (e.g., 10G, 50G) (default "10G")
  --image string     Base image to use
  --network string   Network type: nat, bridge, none (default "nat")
  --gpu string       GPU device for passthrough (e.g., 01:00.0)
  --snapshot string  Create from snapshot
  --profile string  Hardware profile: budget, mid, enthusiast

Examples:
  # Create a microVM
  nanovms vm create my-vm --flavor=microvm --cpu=2 --memory=512M

  # Create with GPU passthrough
  nanovms vm create gaming --flavor=tier1 --gpu=01:00.0 --cpu=8 --memory=32G

  # Create from snapshot
  nanovms vm create dev --snapshot=base-ubuntu-22.04
```

### nanovms vm start

Start a virtual machine.

```bash
nanovms vm start <name> [flags]

Flags:
  --snapshot string  Start from snapshot
  --debug            Enable VM debug mode
  --timeout int      Startup timeout in seconds (default 60)

Examples:
  nanovms vm start my-vm
  nanovms vm start gaming --snapshot=game-ready
```

### nanovms vm stop

Stop a virtual machine.

```bash
nanovms vm stop <name> [flags]

Flags:
  --force, -f     Force stop (SIGKILL)
  --timeout int    Shutdown timeout in seconds (default 30)

Examples:
  nanovms vm stop my-vm
  nanovms vm stop my-vm --force
```

### nanovms vm restart

Restart a virtual machine.

```bash
nanovms vm restart <name> [flags]

Flags:
  --force           Force restart
  --timeout int     Restart timeout in seconds (default 60)

Examples:
  nanovms vm restart my-vm
```

### nanovms vm delete

Delete a virtual machine.

```bash
nanovms vm delete <name> [flags]

Flags:
  --force, -f     Force delete (don't prompt)
  --snapshots      Also delete associated snapshots

Examples:
  nanovms vm delete old-vm
  nanovms vm delete old-vm --force --snapshots
```

### nanovms vm list

List all virtual machines.

```bash
nanovms vm list [flags]

Flags:
  --flavor string   Filter by flavor
  --status string    Filter by status: running, stopped, error
  --format string    Output format: table, json, yaml (default "table")

Examples:
  nanovms vm list
  nanovms vm list --flavor=microvm --status=running --format=json
```

### nanovms vm status

Get VM status.

```bash
nanovms vm status <name> [flags]

Flags:
  --watch, -w       Watch status continuously
  --interval int     Watch interval in seconds (default 5)

Examples:
  nanovms vm status my-vm
  nanovms vm status my-vm --watch
```

### nanovms vm attach

Attach to VM console.

```bash
nanovms vm attach <name> [flags]

Flags:
  --escape string   Escape key sequence (default "^]")
  --width int       Terminal width
  --height int      Terminal height

Examples:
  nanovms vm attach my-vm
```

### nanovms vm exec

Execute command in VM.

```bash
nanovms vm exec <name> -- <command> [flags]

Flags:
  --user string     Execute as user (default "root")
  --cwd string      Working directory
  --env string      Environment variable (can be repeated)
  --timeout int     Command timeout in seconds

Examples:
  nanovms vm exec my-vm -- uname -a
  nanovms vm exec my-vm -- --user=user -- apt update
  nanovms vm exec my-vm -- --env=HOME=/home/user -- pwd
```

### nanovms vm ssh

SSH into VM (if configured).

```bash
nanovms vm ssh <name> [flags]

Flags:
  --user string     SSH user (default "root")
  --port int        SSH port (default 22)
  --key string      SSH private key path

Examples:
  nanovms vm ssh my-vm
  nanovms vm ssh my-vm --user=admin
```

### nanovms vm snapshot

Manage VM snapshots.

```bash
nanovms vm snapshot <name> <action> [flags]

Actions:
  create <snapshot-name>    Create new snapshot
  list                      List snapshots
  delete <snapshot-name>    Delete snapshot
  restore <snapshot-name>   Restore from snapshot

Flags:
  --description string   Snapshot description

Examples:
  nanovms vm snapshot my-vm create base-install
  nanovms vm snapshot my-vm list
  nanovms vm snapshot my-vm restore base-install
```

### nanovms vm clone

Clone a virtual machine.

```bash
nanovms vm clone <source> <destination> [flags]

Flags:
  --snapshot string  Clone from specific snapshot
  --flavor string   Override flavor

Examples:
  nanovms vm clone my-vm my-vm-copy
  nanovms vm clone my-vm dev-env --flavor=microvm
```

### nanovms vm stats

Get VM resource statistics.

```bash
nanovms vm stats <name> [flags]

Flags:
  --watch, -w       Watch stats continuously
  --interval int     Stats interval in seconds (default 5)
  --format string    Output format: table, json (default "table")

Examples:
  nanovms vm stats my-vm
  nanovms vm stats my-vm --watch --interval=1
```

---

## nanovms sandbox

Manage sandbox isolation layers.

### nanovms sandbox create

Create a sandbox.

```bash
nanovms sandbox create <name> [flags]

Flags:
  --tier string     Sandbox tier: native, gvisor, landlock, seccomp, wasm (default "native")
  --root string     Root filesystem path
  --readonly        Mount root as read-only
  --network         Enable network namespace

Examples:
  # Create native sandbox (bwrap)
  nanovms sandbox create my-sandbox --tier=native

  # Create gVisor sandbox
  nanovms sandbox create secure-sandbox --tier=gvisor

  # Create WASM sandbox
  nanovms sandbox create wasm-sandbox --tier=wasm
```

### nanovms sandbox apply

Apply sandbox to a VM.

```bash
nanovms sandbox apply <vm-name> --tier=<tier> [flags]

Flags:
  --tier string     Sandbox tier to apply
  --force           Force apply (stop VM first)

Examples:
  nanovms sandbox apply my-vm --tier=gvisor
  nanovms sandbox apply my-vm --tier=landlock --force
```

### nanovms sandbox exec

Execute command in sandbox.

```bash
nanovms sandbox exec <name> -- <command> [flags]

Examples:
  nanovms sandbox exec my-sandbox -- ls -la
  nanovms sandbox exec my-sandbox -- --user=nobody whoami
```

### nanovms sandbox list

List all sandboxes.

```bash
nanovms sandbox list [flags]

Flags:
  --tier string    Filter by tier
  --format string  Output format: table, json (default "table")

Examples:
  nanovms sandbox list
  nanovms sandbox list --tier=gvisor --format=json
```

---

## nanovms game

Game automation testing commands.

### nanovms game create-pool

Create a pool of game testing VMs.

```bash
nanovms game create-pool <name> [flags]

Flags:
  --count int       Number of VMs in pool (default 4)
  --flavor string   VM flavor (default "microvm")
  --steam          Install Steam
  --headless       Run in headless mode
  --mod string      Game mod framework (e.g., bepinex, melons)

Examples:
  nanovms game create-pool test-pool --count=8 --steam --headless
  nanovms game create-pool mod-pool --count=4 --mod=bepinex
```

### nanovms game run

Run a single game test.

```bash
nanovms game run <vm-name> [flags]

Flags:
  --game string      Game to launch
  --script string   Automation script
  --args string     Game arguments
  --timeout int     Test timeout in seconds

Examples:
  nanovms game run test-vm --game=unity --script=automation.lua
```

### nanovms game parallel

Run parallel game tests.

```bash
nanovms game parallel [flags]

Flags:
  --pool string      VM pool name
  --count int        Number of parallel tests (default 4)
  --suite string     Test suite file
  --report string    Report output file (JUnit XML)
  --timeout int      Total timeout in seconds

Examples:
  nanovms game parallel --pool=test-pool --count=4 --suite=regression.yaml
  nanovms game parallel --pool=test-pool --count=8 --report=results.xml
```

### nanovms game snapshot

Create game-ready snapshot.

```bash
nanovms game snapshot <vm-name> <snapshot-name> [flags]

Flags:
  --game string      Installed game path
  --mods string     Mods directory
  --settings string  Game settings file

Examples:
  nanovms game snapshot test-vm game-ready --game=/opt/game
```

### nanovms game results

View test results.

```bash
nanovms game results [flags]

Flags:
  --report string   Report file to parse
  --format string   Output format: table, json (default "table")
  --failed-only     Show only failed tests

Examples:
  nanovms game results --report=results.xml
  nanovms game results --report=results.xml --failed-only
```

---

## nanovms agent

Agent desktop environment management.

### nanovms agent create

Create an agent environment.

```bash
nanovms agent create <name> [flags]

Flags:
  --flavor string    VM flavor (default "microvm")
  --sandbox string   Sandbox tier (default "gvisor")
  --desktop string   Desktop environment: xfce, kde, gnome (default "xfce")
  --resolution string Desktop resolution (default "1920x1080")

Examples:
  nanovms agent create desktop-agent --desktop=xfce
  nanovms agent create kde-agent --desktop=kde
```

### nanovms agent start

Start an agent.

```bash
nanovms agent start <name> [flags]

Examples:
  nanovms agent start desktop-agent
```

### nanovms agent stop

Stop an agent.

```bash
nanovms agent stop <name> [flags]

Flags:
  --force           Force stop

Examples:
  nanovms agent stop desktop-agent
```

### nanovms agent exec

Execute task in agent.

```bash
nanovms agent exec <name> --task=<task> [flags]

Flags:
  --task string      Task description
  --timeout int      Task timeout in seconds
  --desktop          Use desktop environment

Examples:
  nanovms agent exec desktop-agent --task="browse:https://example.com"
  nanovms agent exec desktop-agent --task="screenshot" --desktop
```

### nanovms agent computer-use

Run computer-use agent task.

```bash
nanovms agent computer-use <name> [flags]

Flags:
  --task string      Task to perform
  --model string     AI model to use
  --max-steps int    Maximum agent steps (default 10)

Examples:
  nanovms agent computer-use desktop-agent --task="book a flight"
```

---

## nanovms vfio

GPU passthrough management.

### nanovms vfio detect

Detect available GPUs.

```bash
nanovms vfio detect [flags]

Flags:
  --verbose         Show detailed info

Examples:
  nanovms vfio detect
  nanovms vfio detect --verbose
```

### nanovms vfio bind

Bind GPU to VFIO driver.

```bash
nanovms vfio bind <pci-address> [flags]

Flags:
  --driver string   Driver to use (default "vfio-pci")

Examples:
  nanovms vfio bind 01:00.0
```

### nanovms vfio unbind

Unbind GPU from VFIO driver.

```bash
nanovms vfio unbind <pci-address> [flags]

Examples:
  nanovms vfio unbind 01:00.0
```

### nanovms vfio status

Show VFIO status.

```bash
nanovms vfio status [flags]

Examples:
  nanovms vfio status
```

### nanovms lookingglass

Looking Glass management.

```bash
nanovms lookingglass connect <vm-name> [flags]

Flags:
  --resize          Allow window resize
  --fullscreen     Start in fullscreen

Examples:
  nanovms lookingglass connect gaming-vm
```

---

## nanovms network

Network configuration.

### nanovms network list

List networks.

```bash
nanovms network list [flags]

Flags:
  --format string   Output format: table, json (default "table")

Examples:
  nanovms network list
```

### nanovms network create

Create network.

```bash
nanovms network create <name> [flags]

Flags:
  --type string     Network type: nat, bridge, macvtap (default "nat")
  --subnet string   Subnet CIDR (default "192.168.100.0/24")
  --dhcp            Enable DHCP
  --gateway string  Gateway IP

Examples:
  nanovms network create my-net --type=bridge
  nanovms network create custom --subnet=10.0.0.0/24 --dhcp
```

### nanovms network delete

Delete network.

```bash
nanovms network delete <name> [flags]

Examples:
  nanovms network delete my-net
```

---

## nanovms storage

Storage management.

### nanovms storage list

List storage pools.

```bash
nanovms storage list [flags]

Examples:
  nanovms storage list
```

### nanovms storage create

Create storage pool.

```bash
nanovms storage create <name> --path=<path> [flags]

Flags:
  --path string     Storage path (required)
  --type string     Pool type: directory, lvm, zfs (default "directory")

Examples:
  nanovms storage create fast-pool --path=/mnt/nvme --type=directory
```

---

## nanovms image

Image management.

### nanovms image list

List available images.

```bash
nanovms image list [flags]

Flags:
  --source string   Image source: local, remote
  --format string   Output format: table, json (default "table")

Examples:
  nanovms image list
  nanovms image list --source=remote --format=json
```

### nanovms image pull

Pull an image.

```bash
nanovms image pull <image> [flags]

Flags:
  --source string   Image source URL
  --format string   Image format: qcow2, raw, vmdk

Examples:
  nanovms image pull ubuntu-22.04 --source=https://cloud-images.ubuntu.com
```

### nanovms image delete

Delete an image.

```bash
nanovms image delete <image> [flags]

Examples:
  nanovms image delete old-image
```

---

## nanovms config

Configuration management.

### nanovms config get

Get configuration value.

```bash
nanovms config get <key> [flags]

Examples:
  nanovms config get defaults.flavor
  nanovms config get paths.vms
```

### nanovms config set

Set configuration value.

```bash
nanovms config set <key>=<value> [flags]

Examples:
  nanovms config set defaults.flavor=microvm
  nanovms config set performance.io_uring=true
```

### nanovms config edit

Open config file in editor.

```bash
nanovms config edit [flags]

Examples:
  nanovms config edit
```

### nanovms config show

Show full configuration.

```bash
nanovms config show [flags]

Flags:
  --format string   Output format: yaml, json (default "yaml")

Examples:
  nanovms config show
  nanovms config show --format=json
```

---

## nanovms completion

Shell completion.

### Generate Completion

```bash
nanovms completion <shell> [flags]

Shells:
  bash, zsh, fish, powershell

Flags:
  --output string   Output file (default: stdout)

Examples:
  # Bash
  nanovms completion bash > /etc/bash_completion.d/nanovms

  # Zsh
  nanovms completion zsh > ~/.zsh/completions/_nanovms

  # Fish
  nanovms completion fish > ~/.config/fish/completions/nanovms.fish
```

---

## nanovms benchmark

Run performance benchmarks.

### nanovms benchmark run

Run benchmarks.

```bash
nanovms benchmark run [flags]

Flags:
  --suite string     Benchmark suite: standard, full, quick (default "standard")
  --vm string        VM to benchmark
  --output string    Output file
  --format string    Output format: text, json (default "text")

Examples:
  nanovms benchmark run
  nanovms benchmark run --suite=full --vm=my-vm
```

---

## nanovms logs

View logs.

### nanovms logs

```bash
nanovms logs [flags]

Flags:
  --vm string        VM to show logs for
  --sandbox string   Sandbox to show logs for
  --follow, -f      Follow log output
  --lines int        Number of lines to show (default 100)
  --level string     Log level filter: debug, info, warn, error

Examples:
  nanovms logs
  nanovms logs --vm=my-vm --follow
  nanovms logs --level=error
```

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Invalid arguments |
| 3 | VM not found |
| 4 | VM already running |
| 5 | VM not running |
| 6 | Permission denied |
| 7 | Resource not available |
| 8 | Timeout |
| 9 | Configuration error |

---

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `NANOVMS_CONFIG` | Config file path | `~/.config/nanovms/config.yaml` |
| `NANOVMS_DATA_DIR` | Data directory | `~/.local/share/nanovms` |
| `NANOVMS_LOG_LEVEL` | Log level | `info` |
| `NANOVMS_NO_COLOR` | Disable colors | `false` |

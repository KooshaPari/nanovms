# ADR-006: CLI Design

## Status

Accepted

## Context

NanoVMS needs a user-friendly CLI for:
- VM lifecycle management
- Sandbox creation
- Agent spawning
- Game automation
- VFIO management

## Decision Drivers

- Simplicity
- Discoverability
- Composability
- Cross-platform consistency

## Commands

### VM Commands

```bash
# Create a VM
nanovms vm create --name=test --flavor=firecracker

# List VMs
nanovms vm list

# Start/stop/delete
nanovms vm start test
nanovms vm stop test
nanovms vm delete test

# Get VM info
nanovms vm info test

# Console access
nanovms vm console test

# Snapshot management
nanovms vm snapshot test --name=base
nanovms vm restore test --snapshot=base
```

### Sandbox Commands

```bash
# Create sandbox
nanovms sandbox create --name=dev --tier=gvisor

# Apply sandbox to running VM
nanovms sandbox apply test --tier=landlock

# List sandboxes
nanovms sandbox list
```

### Agent Commands

```bash
# Spawn agent
nanovms agent spawn --name=desktop --vm=test --sandbox=gvisor

# Execute task
nanovms agent exec desktop --task="browse:https://example.com"

# Computer use mode
nanovms agent computer-use desktop
```

### Game Commands

```bash
# Create game VM
nanovms game create --name=test-game --flavor=firecracker --headless

# Run with mods
nanovms game run test-game --mod=bepinex --script=automation.lua

# Parallel test execution
nanovms game parallel --count=8 --suite=regression

# Snapshot for fast restore
nanovms game snapshot test-game --name=level-1
```

### VFIO Commands

```bash
# Bind GPU
nanovms vfio bind --gpu=01:00.0 --driver=nvidia

# Status
nanovms vfio status

# Looking Glass connection
nanovms lookingglass connect gaming-vm
```

## Subcommands Structure

```
nanovms
├── vm          # VM management
├── sandbox     # Sandbox isolation
├── agent       # Agent workloads
├── game        # Game automation
├── vfio        # GPU passthrough
├── image       # Image management
├── network     # Network config
├── profile     # VM/sandbox profiles
├── completion  # Shell completion
├── config      # Configuration
└── help        # Help
```

## Flag Conventions

- `--name, -n`: Resource name
- `--flavor, -f`: Flavor/type
- `--output, -o`: Output format (text, json, yaml)
- `--verbose, -v`: Verbose output
- `--dry-run`: Show what would be done

## Output Format

```bash
# Default (human-readable)
$ nanovms vm list
NAME      STATUS    FLAVOR      IP
test-vm   running   firecracker  192.168.1.100
dev-vm    stopped   gvisor      -

# JSON output
$ nanovms vm list --output=json
[{"name":"test-vm","status":"running","flavor":"firecracker","ip":"192.168.1.100"}]
```

## Shell Completion

Generated for: bash, zsh, fish, PowerShell

```bash
# Install completion
nanovms completion bash > /etc/bash_completion.d/nanovms
nanovms completion zsh > ~/.zsh/completions/_nanovms
nanovms completion fish > ~/.config/fish/completions/nanovms.fish
```

## Consequences

### Positive
- Intuitive command structure
- Easy to discover
- Consistent flag naming
- Shell completion for discoverability

### Negative
- Large command surface
- May need to learn new patterns

## References

- [cobra CLI library](https://github.com/spf13/cobra)
- [Cli style guide](https://clig.dev/)

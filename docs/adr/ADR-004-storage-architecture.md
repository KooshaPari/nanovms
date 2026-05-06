# ADR-004: Storage Architecture

## Status

Accepted

## Context

NanoVMS needs to manage storage for VMs and containers with requirements for:
- Fast I/O (NVMe + io_uring)
- Snapshots and versioning
- CoW (Copy-on-Write) for fast cloning
- Compression for pre-copied images
- Cross-platform support

## Decision Drivers

- Performance (IOPS, latency)
- Space efficiency
- Snapshot capability
- Cross-platform compatibility
- Simplicity

## Options Considered

### Option 1: QCOW2 + QEMU (Selected)

**Pros:**
- Native QEMU/KVM support
- Live snapshots
- Compression
- Copy-on-write
- Widely tested

**Cons:**
- QEMU-specific
- Complex on other platforms

### Option 2: ZFS

**Pros:**
- Best-in-class snapshots
- Compression
- Data integrity
- Copy-on-write

**Cons:**
- Not native on macOS/Windows
- Heavy memory usage
- Complex setup

### Option 3: Btrfs

**Pros:**
- Native Linux snapshots
- Compression
- Fast CoW

**Cons:**
- Not stable on some distributions
- Limited cross-platform

### Option 4: erofs + overlay

**Pros:**
- Fast read-only filesystem
- Excellent for container images
- Compression

**Cons:**
- Read-only base
- Needs overlay for writes

## Decision

We select a **layered approach**:

1. **Base images**: erofs with compression (fast distribution)
2. **Writable layer**: overlayfs or QCOW2 snapshot
3. **Snapshots**: QCOW2 backing files or native snapshots

For macOS/Windows, we use:
- APFS + Time Machine for snapshots
- Proprietary CoW format for VMs

## Consequences

### Positive
- Best performance per platform
- Snapshot support
- Compression for fast distribution

### Negative
- Different formats per platform
- Complexity in migration

## Implementation

```go
type StorageManager struct {
    basePath    string
    snapshotPath string
    compressor  Compressor
}

func (s *StorageManager) CreateSnapshot(ctx context.Context, vmID string, parent string) (string, error) {
    if parent != "" {
        // QCOW2 backing file
        return s.createQCowSnapshot(vmID, parent)
    }
    // Full copy
    return s.createFullCopy(vmID)
}

func (s *StorageManager) CompressImage(ctx context.Context, path string) error {
    return s.compressor.Compress(path)
}
```

---
head:
  - - meta
    - name: description
      content: "Setup VFIO GPU passthrough for near-bare-metal gaming performance"
---

# Setup GPU Passthrough

> Configure VFIO for 99% bare-metal GPU performance

**Time**: 30 minutes | **Level**: Advanced | **Prerequisites**: Linux host, dedicated GPU

## Goal

Configure a Windows VM with GPU passthrough for gaming or GPU-intensive workloads.

## Prerequisites

### Hardware Requirements

| Component | Minimum | Recommended |
|-----------|---------|-------------|
| CPU | Intel VT-d or AMD-Vi | Latest gen with IOMMU |
| GPU | GTX 1060 / RX 580 | RTX 3080 / RX 6800 XT |
| RAM | 16GB | 32GB+ |
| Storage | 100GB SSD | 500GB NVMe |

### Check IOMMU Support

```bash
# Check if IOMMU is enabled
dmesg | grep -i -e DMAR -e IOMMU

# Should see:
# [    0.000000] DMAR: IOMMU enabled

# Check IOMMU groups
find /sys/kernel/iommu_groups/ -type l | sort -V
```

## Step 1: Kernel Parameters

```bash
# Edit GRUB configuration
sudo nano /etc/default/grub

# Add to GRUB_CMDLINE_LINUX_DEFAULT:
# intel_iommu=on iommu=pt rd.driver.pre=vfio-pci

# For AMD:
# amd_iommu=on iommu=pt rd.driver.pre=vfio-pci

# Update GRUB
sudo grub-mkconfig -o /boot/grub/grub.cfg
```

## Step 2: VFIO Configuration

```bash
# Create VFIO configuration
sudo nano /etc/modprobe.d/vfio.conf

# Add:
options vfio-pci ids=10de:1b80,10de:10f0 disable_vga=1

# Blacklist NVIDIA driver
sudo nano /etc/modprobe.d/blacklist-nvidia.conf

# Add:
blacklist nouveau
blacklist nvidia
blacklist nvidia_drm
blacklist nvidia_uvm
blacklist nvidia_modeset
```

## Step 3: Create VM with GPU

```bash
# Create VFIO-enabled VM
nanovms vm create gaming-vm \
  --flavor vfio \
  --image windows-11 \
  --gpu 01:00.0 \
  --gpu-audio 01:00.1 \
  --memory 16g \
  --vcpus 8
```

## Step 4: Looking Glass Setup

```bash
# Install Looking Glass (host)
git clone https://github.com/gnif/LookingGlass.git
cd LookingGlass
mkdir build && cd build
cmake ../
make -j$(nproc)
sudo make install

# Start Looking Glass client
looking-glass-client -S -F
```

## Step 5: VM Configuration

```bash
# Enable hugepages
echo 8192 | sudo tee /sys/kernel/mm/hugepages/hugepages-2048kB/nr_hugepages

# Configure CPU pinning
nanovms vm config gaming-vm --cpu-pinning 4-7,12-15

# Start VM
nanovms vm start gaming-vm
```

## Verification

```bash
# Verify GPU passthrough
nanovms vfio status

# Check VM GPU
nanovms vm exec gaming-vm -- nvidia-smi
```

## Performance Benchmarks

| Metric | Bare Metal | VM (VFIO) | Overhead |
|--------|------------|-----------|----------|
| 3DMark Score | 25,000 | 24,500 | 2% |
| CS2 FPS | 450 | 445 | 1.1% |
| Memory Latency | 60ns | 62ns | 3.3% |

## Troubleshooting

### Code 43 (NVIDIA)

```bash
# Hide KVM from guest
nanovms vm config gaming-vm --kvm-hidden true
```

### Audio Issues

```bash
# Use Scream for audio
nanovms vm config gaming-vm --audio scream
```

## Success!

Your GPU is now passed through with near-bare-metal performance.

## Next Steps

- [Optimize VM Performance](./optimize-vm.md)
- [Configure USB Passthrough](./usb-passthrough.md)
- [Setup Looking Glass B6](./looking-glass-b6.md)

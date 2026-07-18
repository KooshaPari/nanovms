---
head:
  - - meta
    - name: description
      content: "Step-by-step guide to creating your first NanoVMS VM in 5 minutes"
---

# Create Your First VM

> Get a VM running in under 60 seconds with NanoVMS

**Time**: 5 minutes | **Level**: Beginner | **Prerequisites**: None

## Goal

Create and start your first Firecracker microVM with NanoVMS.

## Step-by-Step

### 1. Install NanoVMS

```bash
curl -fsSL https://nanovms.dev/install.sh | sh
```

### 2. Create Your VM

```bash
nanovms vm create my-first-vm --flavor microvm --image alpine:3.19
```

### 3. Start the VM

```bash
nanovms vm start my-first-vm
```

### 4. Verify

```bash
nanovms vm list
```

## Success! 

You've created your first NanoVMS VM.

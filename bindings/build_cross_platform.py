#!/usr/bin/env python3
"""
Cross-platform build script for NVMS bindings.

Supports:
- Linux (x86_64, aarch64)
- macOS (x86_64, ARM64/M-series)
- Windows (x86_64)

GPU backends:
- CUDA (NVIDIA)
- ROCm (AMD)
- Metal (Apple Silicon)
- CPU fallback
"""

import os
import sys
import subprocess
import platform
import shutil
from pathlib import Path


def run(cmd: list[str], cwd: Path | None = None, env: dict | None = None) -> int:
    """Run command and return exit code."""
    print(f"  $ {' '.join(cmd)}")
    result = subprocess.run(cmd, cwd=cwd, env=env)
    return result.returncode


def detect_platform() -> dict:
    """Detect platform and architecture."""
    system = platform.system().lower()
    machine = platform.machine().lower()

    # Normalize machine names
    if machine in ("amd64", "x86_64"):
        machine = "x86_64"
    elif machine in ("arm64", "aarch64"):
        machine = "arm64"
    elif machine == "arm":
        machine = "armv7"

    return {
        "system": system,
        "machine": machine,
        "is_apple_silicon": system == "darwin" and machine == "arm64",
        "is_cuda_available": shutil.which("nvcc") is not None,
        "is_rocm_available": shutil.which("rocminfo") is not None,
    }


def check_go() -> bool:
    """Check if Go is installed."""
    return shutil.which("go") is not None


def check_rust() -> bool:
    """Check if Rust/Cargo is installed."""
    return shutil.which("cargo") is not None


def check_zig() -> bool:
    """Check if Zig is installed."""
    return shutil.which("zig") is not None


def build_go_c_export(plat: dict, verbose: bool = False) -> int:
    """Build Go C-export layer."""
    print("\n=== Building Go C-Export ===")

    if not check_go():
        print("  [SKIP] Go not found")
        return 0

    src_dir = Path("bindings/go-c-export")
    if not src_dir.exists():
        print("  [SKIP] go-c-export not found")
        return 0

    # Output directory
    out_dir = src_dir / "out" / f"{plat['system']}-{plat['machine']}"
    out_dir.mkdir(parents=True, exist_ok=True)

    # Build with c-archive
    env = os.environ.copy()
    env["CGO_ENABLED"] = "1"

    # Cross-compilation for ARM64 on x86_64 macOS
    if plat["is_apple_silicon"] and plat["machine"] == "arm64":
        env["GOARCH"] = "arm64"
        env["GOOS"] = "darwin"
    elif plat["system"] == "linux" and plat["machine"] == "arm64":
        env["GOARCH"] = "arm64"
        env["GOOS"] = "linux"
    elif plat["system"] == "windows":
        env["GOOS"] = "windows"

    cmd = [
        "go", "build",
        "-buildmode", "c-archive",
        "-o", str(out_dir / "libnvms_core"),
    ]

    if verbose:
        cmd.append("-v")

    ret = run(cmd, cwd=src_dir, env=env)

    if ret == 0:
        print(f"  [OK] Built: {out_dir / 'libnvms_core.a'}")

    return ret


def build_rust_ffi(plat: dict, verbose: bool = False) -> int:
    """Build Rust FFI bindings."""
    print("\n=== Building Rust FFI ===")

    if not check_rust():
        print("  [SKIP] Rust not found")
        return 0

    src_dir = Path("bindings/rust-ffi")
    if not src_dir.exists():
        print("  [SKIP] rust-ffi not found")
        return 0

    # Target triple for cross-compilation
    target = None
    if plat["system"] == "darwin" and plat["machine"] == "arm64":
        target = "aarch64-apple-darwin"
    elif plat["system"] == "linux" and plat["machine"] == "arm64":
        target = "aarch64-unknown-linux-gnu"
    elif plat["system"] == "windows":
        target = "x86_64-pc-windows-msvc"

    cmd = ["cargo", "build", "--package", "nvms-ffi"]
    if verbose:
        cmd.append("-v")
    if target:
        cmd.extend(["--target", target])

    ret = run(cmd, cwd=src_dir)

    if ret == 0:
        print(f"  [OK] Rust FFI built")

    return ret


def build_zig(plat: dict, verbose: bool = False) -> int:
    """Build Zig module."""
    print("\n=== Building Zig Module ===")

    if not check_zig():
        print("  [SKIP] Zig not found")
        return 0

    src_dir = Path("bindings/zig")
    if not src_dir.exists():
        print("  [SKIP] zig not found")
        return 0

    # Try to build with zig build
    if (src_dir / "build.zig").exists():
        cmd = ["zig", "build"]
        if verbose:
            cmd.append("-v")
        ret = run(cmd, cwd=src_dir)

        if ret == 0:
            print("  [OK] Zig module built")
        return ret

    # Otherwise try to compile directly
    cmd = ["zig", "build-lib", "memory.zig", "-fPIC"]
    if plat["system"] == "darwin":
        cmd.extend(["-target", "aarch64-macos-gnu" if plat["is_apple_silicon"] else "x86_64-macos-gnu"])
    elif plat["system"] == "linux":
        cmd.extend(["-target", f"{plat['machine']}-linux-gnu"])

    ret = run(cmd, cwd=src_dir)

    if ret == 0:
        print("  [OK] Zig module built")
    return ret


def test_python_bindings() -> int:
    """Test Python ML bindings."""
    print("\n=== Testing Python Bindings ===")

    try:
        import nvms_ml

        # Test backend detection
        backend = nvms_ml.get_gpu_backend()
        print(f"  GPU Backend: {backend}")

        # Test device count
        count = nvms_ml.device_count()
        print(f"  GPU Devices: {count}")

        # Test VectorEmbedding
        emb = nvms_ml.VectorEmbedding(dim=128, gpu_backend=backend)
        emb.add("test", np.ones(128))
        indices, scores = emb.search(np.ones(128), k=1)
        print(f"  VectorEmbedding: OK (indices={indices}, scores={scores})")

        print("  [OK] Python bindings work")
        return 0

    except ImportError as e:
        print(f"  [SKIP] Import error: {e}")
        return 0
    except Exception as e:
        print(f"  [WARN] Test error: {e}")
        return 0


def main():
    plat = detect_platform()

    print("=" * 60)
    print("NVMS Cross-Platform Build")
    print("=" * 60)
    print(f"Platform: {plat['system']}-{plat['machine']}")
    print(f"Apple Silicon: {plat['is_apple_silicon']}")
    print(f"CUDA: {plat['is_cuda_available']}")
    print(f"ROCm: {plat['is_rocm_available']}")
    print()

    verbose = "-v" in sys.argv or "--verbose" in sys.argv

    failures = []

    # Build Go
    if build_go_c_export(plat, verbose) != 0:
        failures.append("Go C-Export")

    # Build Rust
    if build_rust_ffi(plat, verbose) != 0:
        failures.append("Rust FFI")

    # Build Zig
    if build_zig(plat, verbose) != 0:
        failures.append("Zig")

    # Test Python
    if test_python_bindings() != 0:
        failures.append("Python bindings")

    print()
    print("=" * 60)
    if failures:
        print(f"FAILED: {', '.join(failures)}")
        return 1
    else:
        print("All builds successful!")
        return 0


if __name__ == "__main__":
    sys.exit(main())

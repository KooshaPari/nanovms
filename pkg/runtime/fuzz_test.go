// SPDX-License-Identifier: MIT OR Apache-2.0
package runtime

import (
	"testing"
)

// ---------------------------------------------------------------------------
// Fuzz: ParseBackend — Go native fuzzing (testing.F)
// ---------------------------------------------------------------------------

func FuzzParseBackend(f *testing.F) {
	// Seed corpus: all valid backends + known invalid strings
	for _, seed := range []string{
		"nanovms",
		"podman",
		"apple-containers",
		"wsl-containers",
		"docker",
		"",
		" ",
		"PODMAN",
		"Podman",
		"nanovms ",
		" nanovms",
		"nanovms\tpodman",
		"../../etc/passwd",
		"SELECT * FROM backends",
		string([]byte{0x00, 0xFF, 0xFE}),
		"<script>alert(1)</script>",
		"\x00",
		"nanovms; rm -rf /",
		"podman\ninjection",
		"very-long-backend-name-that-exceeds-all-reasonable-limits-and-keeps-going-forever",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		b, err := ParseBackend(input)
		if err != nil {
			// Invalid input should always return an error with empty Backend
			if b != "" {
				t.Errorf("expected empty Backend on error, got %q for input %q", b, input)
			}
			return
		}

		// If no error, the returned Backend must match the canonical constants
		switch b {
		case BackendNanoVMS, BackendPodman, BackendAppleContainers, BackendWSLContainers:
			// Valid — round-trip check
			again, err2 := ParseBackend(string(b))
			if err2 != nil {
				t.Errorf("round-trip ParseBackend(%q) failed: %v", b, err2)
			}
			if again != b {
				t.Errorf("round-trip mismatch: %q -> %q", b, again)
			}
		default:
			t.Errorf("ParseBackend(%q) returned unexpected backend %q", input, b)
		}
	})
}

// ---------------------------------------------------------------------------
// Fuzz: Registry.Get — Go native fuzzing (testing.F)
// ---------------------------------------------------------------------------

func FuzzRegistryGet(f *testing.F) {
	// Seed corpus: valid tiers + edge cases
	for _, seed := range []int{
		0,
		1,
		2,
		3,
		4,
		-1,
		-100,
		99,
		1000,
		2147483647,  // max int32
		-2147483648, // min int32
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, tier int) {
		r := NewRegistry()

		rt, err := r.Get(tier)

		if err != nil {
			// Error path: should always return nil runtime
			if rt != nil {
				t.Errorf("expected nil Runtime on error for tier %d, got %v", tier, rt)
			}
			return
		}

		// Success path: only tiers 1, 2, 3 are registered
		switch tier {
		case 1:
			if rt.Name() != "wasm" {
				t.Errorf("tier 1: expected wasm, got %s", rt.Name())
			}
			if rt.Tier() != 1 {
				t.Errorf("tier 1: expected Tier()=1, got %d", rt.Tier())
			}
		case 2:
			if rt.Name() != "gvisor" {
				t.Errorf("tier 2: expected gvisor, got %s", rt.Name())
			}
		case 3:
			if rt.Name() != "firecracker" {
				t.Errorf("tier 3: expected firecracker, got %s", rt.Name())
			}
		default:
			t.Errorf("unexpected success for tier %d — only 1,2,3 should succeed", tier)
		}

		// Interface contract: StartupTime must be non-negative
		if rt.StartupTime() < 0 {
			t.Errorf("negative StartupTime for tier %d: %v", tier, rt.StartupTime())
		}
	})
}

// ---------------------------------------------------------------------------
// Fuzz: BackendRegistry.Resolve — Go native fuzzing (testing.F)
// ---------------------------------------------------------------------------

func FuzzBackendRegistryResolve(f *testing.F) {
	// Seed corpus
	for _, seed := range []string{
		"nanovms",
		"podman",
		"apple-containers",
		"wsl-containers",
		"docker",
		"",
		"\x00",
		"PODMAN",
		"nanovms-podman",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		r := NewBackendRegistry()
		meta, err := r.Resolve(BackendID(input))

		if err != nil {
			// Error path
			return
		}

		// Success: Tier must be 1, 2, or 3
		if meta.Tier < 1 || meta.Tier > 3 {
			t.Errorf("invalid tier %d for backend %q", meta.Tier, input)
		}

		// Success: ID must match
		if meta.ID != BackendID(input) {
			t.Errorf("ID mismatch: got %q, want %q", meta.ID, input)
		}

		// Resolve twice should give same result
		meta2, err2 := r.Resolve(BackendID(input))
		if err2 != nil {
			t.Errorf("second Resolve(%q) failed: %v", input, err2)
		}
		if meta != meta2 {
			t.Errorf("non-deterministic Resolve(%q): %+v vs %+v", input, meta, meta2)
		}
	})
}

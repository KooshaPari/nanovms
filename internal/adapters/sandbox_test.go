package adapters

import (
	"testing"
)

func TestNewSandboxPort(t *testing.T) {
	tests := []struct {
		tier    int
		wantErr bool
	}{
		{1, true},  // WASM not yet implemented as SandboxPort
		{2, false}, // gVisor
		{3, false}, // Firecracker
		{0, true},  // invalid
		{4, true},  // invalid
	}
	for _, tt := range tests {
		port, err := NewSandboxPort(tt.tier)
		if tt.wantErr {
			if err == nil {
				t.Errorf("tier %d: expected error, got port=%v", tt.tier, port)
			}
			continue
		}
		if err != nil {
			t.Errorf("tier %d: unexpected error: %v", tt.tier, err)
		}
		if port == nil {
			t.Errorf("tier %d: expected non-nil port", tt.tier)
		}
	}
}

func TestNewProviderKeepsTierDefaultsAndAddsOptInPodman(t *testing.T) {
	port, err := NewProvider("tier", 2)
	if err != nil || port == nil {
		t.Fatalf("tier provider: port=%v err=%v", port, err)
	}
	port, err = NewProvider("", 3)
	if err != nil || port == nil {
		t.Fatalf("empty provider: port=%v err=%v", port, err)
	}
	port, err = NewProvider("podman", 1)
	if err != nil || port == nil {
		t.Fatalf("podman provider: port=%v err=%v", port, err)
	}
	if _, err := NewProvider("unknown", 2); err == nil {
		t.Fatal("unknown provider unexpectedly accepted")
	}
}

func TestNewProviderAddsNativeContainerAdapters(t *testing.T) {
	for _, name := range []string{"apple-containers", "wsl-containers"} {
		port, err := NewProvider(name, 2)
		if err != nil || port == nil {
			t.Fatalf("%s provider: port=%v err=%v", name, port, err)
		}
	}
}

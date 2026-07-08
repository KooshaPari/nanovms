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

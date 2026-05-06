package domain

import (
	"testing"
)

func TestSandboxString(t *testing.T) {
	sb := &Sandbox{ID: "abc123", Name: "test-sb", Status: SandboxStatusRunning}
	s := sb.String()
	if s == "" {
		t.Fatal("String() returned empty")
	}
}

func TestSandboxIsRunning(t *testing.T) {
	tests := []struct {
		status   SandboxStatus
		expected bool
	}{
		{SandboxStatusPending, false},
		{SandboxStatusRunning, true},
		{SandboxStatusStopped, false},
		{SandboxStatusFailed, false},
		{SandboxStatusDeleting, false},
	}
	for _, tc := range tests {
		sb := &Sandbox{Status: tc.status}
		if got := sb.IsRunning(); got != tc.expected {
			t.Errorf("IsRunning() for status %s: got %v, want %v", tc.status, got, tc.expected)
		}
	}
}

func TestGenerateID(t *testing.T) {
	id := GenerateID()
	if id == "" {
		t.Fatal("GenerateID() returned empty string")
	}
	// Should be a number string
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		ids[GenerateID()] = true
	}
	if len(ids) < 50 {
		t.Fatalf("GenerateID() produced too few unique IDs: %d/100", len(ids))
	}
}

func TestSandboxStatus(t *testing.T) {
	tests := []SandboxStatus{
		SandboxStatusPending,
		SandboxStatusRunning,
		SandboxStatusStopped,
		SandboxStatusFailed,
		SandboxStatusDeleting,
	}
	for _, s := range tests {
		if s == "" {
			t.Errorf("SandboxStatus constant is empty")
		}
	}
}

func TestSandboxType(t *testing.T) {
	types := []SandboxType{
		SandboxTypeVM,
		SandboxTypeContainer,
		SandboxTypeWasm,
		SandboxTypeProcess,
		SandboxTypeNative,
	}
	for _, st := range types {
		if st == "" {
			t.Error("SandboxType constant is empty")
		}
	}
}

func TestVMFlavor(t *testing.T) {
	flavors := []VMFlavor{
		VMFlavorNative,
		VMFlavorLima,
		VMFlavorMicroVM,
	}
	for _, f := range flavors {
		if f == "" {
			t.Errorf("VMFlavor constant is empty")
		}
	}
}

func TestNativeSandboxType(t *testing.T) {
	types := []NativeSandboxType{
		NativeSandboxBwrap,
		NativeSandboxFirejail,
		NativeSandboxUnshare,
		NativeSandboxChroot,
	}
	for _, nst := range types {
		if nst == "" {
			t.Error("NativeSandboxType constant is empty")
		}
	}
}

func TestWASMState(t *testing.T) {
	states := []WASMState{
		WASMStateInstantiated,
		WASMStateRunning,
		WASMStateTerminated,
	}
	for _, s := range states {
		if s == "" {
			t.Errorf("WASMState constant is empty")
		}
	}
}

func TestSandboxConfigDefaults(t *testing.T) {
	cfg := SandboxConfig{Name: "my-sandbox", VMType: VMFlavorNative}
	if cfg.Name != "my-sandbox" {
		t.Errorf("expected Name=my-sandbox, got %s", cfg.Name)
	}
	if cfg.VMType != VMFlavorNative {
		t.Errorf("expected VMType=native, got %s", cfg.VMType)
	}
}

func TestResourceConfig(t *testing.T) {
	rc := ResourceConfig{CPU: 4, MemoryMB: 8192, DiskMB: 50000}
	if rc.CPU != 4 {
		t.Errorf("expected CPU=4, got %d", rc.CPU)
	}
	if rc.MemoryMB != 8192 {
		t.Errorf("expected MemoryMB=8192, got %d", rc.MemoryMB)
	}
}

func TestSandboxMetricsZero(t *testing.T) {
	m := &SandboxMetrics{}
	if m.CPUUsage != 0 {
		t.Errorf("expected CPUUsage=0, got %f", m.CPUUsage)
	}
	if m.MemoryUsage != 0 {
		t.Errorf("expected MemoryUsage=0, got %d", m.MemoryUsage)
	}
	if m.DiskUsage != 0 {
		t.Errorf("expected DiskUsage=0, got %d", m.DiskUsage)
	}
}

func TestFilesystemMount(t *testing.T) {
	m := FilesystemMount{Source: "/data", Target: "/mnt", ReadOnly: true}
	if m.Source != "/data" {
		t.Errorf("expected Source=/data, got %s", m.Source)
	}
	if !m.ReadOnly {
		t.Error("expected ReadOnly=true")
	}
}

func TestPortMapping(t *testing.T) {
	pm := PortMapping{HostPort: 8080, ContainerPort: 80, Protocol: "tcp"}
	if pm.HostPort != 8080 {
		t.Errorf("expected HostPort=8080, got %d", pm.HostPort)
	}
	if pm.Protocol != "tcp" {
		t.Errorf("expected Protocol=tcp, got %s", pm.Protocol)
	}
}

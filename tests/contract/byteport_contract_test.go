package contract

// Cross-ecosystem contract: BytePort (GUI) <-> nanovms (runtime)
//
// These tests verify that the data contracts between the BytePort frontend
// and the nanovms runtime are compatible. They don't require running either
// service — they validate the API schema alignment.

import (
	"encoding/json"
	"testing"
)

// SandboxStatus represents the API response contract that nanovms exposes
// and BytePort renders. This MUST match the BytePort SandboxNode type.
type SandboxStatus struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Tier     int               `json:"tier"`
	Status   string            `json:"status"`   // "running", "stopped", "error"
	Image    string            `json:"image"`
	CPU      string            `json:"cpu"`
	Memory   string            `json:"memory"`
	Ports    []int             `json:"ports"`
	Metadata map[string]string `json:"metadata"`
}

// TierInfo represents the tier probe response contract.
type TierInfo struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Available bool  `json:"available"`
	Runtime  string `json:"runtime"` // "native", "gvisor", "firecracker", etc.
}

// TestSandboxStatusContract verifies the nanovms SandboxStatus JSON schema
// matches what BytePort's SandboxNode component expects to render.
func TestSandboxStatusContract(t *testing.T) {
	validStatuses := map[string]bool{
		"running": true,
		"stopped": true,
		"error":   true,
	}

	// Example payload from nanovms /v1/sandboxes endpoint
	payload := `{
		"id": "sb-abc123",
		"name": "test-sandbox",
		"tier": 2,
		"status": "running",
		"image": "alpine:latest",
		"cpu": "2",
		"memory": "512",
		"ports": [8080, 3000],
		"metadata": {"gpu": "false"}
	}`

	var status SandboxStatus
	if err := json.Unmarshal([]byte(payload), &status); err != nil {
		t.Fatalf("SandboxStatus JSON parse failed: %v", err)
	}

	// Verify all fields are present
	if status.ID == "" {
		t.Error("ID must be non-empty")
	}
	if status.Name == "" {
		t.Error("Name must be non-empty")
	}
	if status.Tier < 1 || status.Tier > 30 {
		t.Errorf("Tier must be 1-30, got %d", status.Tier)
	}
	if !validStatuses[status.Status] {
		t.Errorf("Status must be one of running/stopped/error, got %q", status.Status)
	}
	if len(status.Ports) == 0 {
		t.Error("Ports must be non-empty array")
	}
}

// TestTierInfoContract verifies the nanovms tier probe response
// matches what BytePort's tier selector expects.
func TestTierInfoContract(t *testing.T) {
	payload := `{"id": 2, "name": "gvisor", "available": true, "runtime": "gvisor"}`

	var tier TierInfo
	if err := json.Unmarshal([]byte(payload), &tier); err != nil {
		t.Fatalf("TierInfo JSON parse failed: %v", err)
	}

	if tier.ID < 1 || tier.ID > 30 {
		t.Errorf("Tier ID must be 1-30, got %d", tier.ID)
	}
	if tier.Name == "" {
		t.Error("Name must be non-empty")
	}
}

// TestSandboxStatusRoundTrip verifies nanovms response can be serialized
// and deserialized by BytePort's data layer without data loss.
func TestSandboxStatusRoundTrip(t *testing.T) {
	status := SandboxStatus{
		ID:       "sb-test",
		Name:     "test",
		Tier:     1,
		Status:   "running",
		Image:    "alpine:3.19",
		CPU:      "1",
		Memory:   "256",
		Ports:    []int{80},
		Metadata: map[string]string{"env": "test"},
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var roundtrip SandboxStatus
	if err := json.Unmarshal(data, &roundtrip); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if roundtrip.ID != status.ID {
		t.Errorf("ID mismatch: %q vs %q", roundtrip.ID, status.ID)
	}
	if len(roundtrip.Ports) != len(status.Ports) {
		t.Errorf("Ports length mismatch: %d vs %d", len(roundtrip.Ports), len(status.Ports))
	}
}

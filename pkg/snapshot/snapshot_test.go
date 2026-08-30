// SPDX-License-Identifier: MIT OR Apache-2.0
package snapshot

import (
	"strings"
	"testing"
	"time"
)

func TestCreateAndGetSnapshot(t *testing.T) {
	store := NewMemorySnapshotStore()

	id, err := store.Create("sandbox-1")
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}
	if id == "" {
		t.Fatal("Create: returned empty ID")
	}

	got, err := store.Get(id)
	if err != nil {
		t.Fatalf("Get: unexpected error: %v", err)
	}
	if got.SandboxID != "sandbox-1" {
		t.Errorf("Get: SandboxID = %q, want %q", got.SandboxID, "sandbox-1")
	}
	if got.SizeBytes != 0 {
		t.Errorf("Get: SizeBytes = %d, want 0", got.SizeBytes)
	}
	if got.TierID != 0 {
		t.Errorf("Get: TierID = %d, want 0", got.TierID)
	}
	if got.CreatedAt.IsZero() {
		t.Error("Get: CreatedAt is zero")
	}
}

func TestListSnapshotsForSandbox(t *testing.T) {
	store := NewMemorySnapshotStore()

	// Create snapshots for two different sandboxes.
	id1, _ := store.Create("sandbox-a")
	time.Sleep(2 * time.Millisecond) // ensure distinct timestamps
	id2, _ := store.Create("sandbox-a")
	_, _ = store.Create("sandbox-b")

	list, err := store.List("sandbox-a")
	if err != nil {
		t.Fatalf("List: unexpected error: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("List: got %d snapshots, want 2", len(list))
	}

	// Verify ordering: id1 should come before id2.
	if list[0].ID != id1 || list[1].ID != id2 {
		t.Errorf("List: order mismatch: got [%s, %s], want [%s, %s]",
			list[0].ID, list[1].ID, id1, id2)
	}
}

func TestDeleteSnapshot(t *testing.T) {
	store := NewMemorySnapshotStore()

	id, _ := store.Create("sandbox-1")

	if err := store.Delete(id); err != nil {
		t.Fatalf("Delete: unexpected error: %v", err)
	}

	// Verify it's gone.
	_, err := store.Get(id)
	if err == nil {
		t.Fatal("Get after Delete: expected error, got nil")
	}

	// List should be empty now.
	list, _ := store.List("sandbox-1")
	if len(list) != 0 {
		t.Errorf("List after Delete: got %d snapshots, want 0", len(list))
	}
}

func TestGetNonexistentReturnsError(t *testing.T) {
	store := NewMemorySnapshotStore()

	_, err := store.Get(SnapshotID("does-not-exist"))
	if err == nil {
		t.Fatal("Get nonexistent: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Get nonexistent: error = %q, want substring 'not found'", err.Error())
	}
}

func TestDeleteNonexistentReturnsError(t *testing.T) {
	store := NewMemorySnapshotStore()

	err := store.Delete(SnapshotID("does-not-exist"))
	if err == nil {
		t.Fatal("Delete nonexistent: expected error, got nil")
	}
}

func TestMultipleSandboxesIsolated(t *testing.T) {
	store := NewMemorySnapshotStore()

	_, _ = store.Create("sandbox-x")
	_, _ = store.Create("sandbox-x")
	_, _ = store.Create("sandbox-y")

	listX, _ := store.List("sandbox-x")
	listY, _ := store.List("sandbox-y")
	listZ, _ := store.List("sandbox-z") // empty sandbox

	if len(listX) != 2 {
		t.Errorf("List sandbox-x: got %d, want 2", len(listX))
	}
	if len(listY) != 1 {
		t.Errorf("List sandbox-y: got %d, want 1", len(listY))
	}
	if len(listZ) != 0 {
		t.Errorf("List sandbox-z: got %d, want 0", len(listZ))
	}
}

func TestRestoreNoop(t *testing.T) {
	store := NewMemorySnapshotStore()

	id, _ := store.Create("sandbox-1")

	err := store.Restore(id, "sandbox-2")
	if err != nil {
		t.Fatalf("Restore: unexpected error: %v", err)
	}
}

func TestSnapshotIDIsUUID(t *testing.T) {
	store := NewMemorySnapshotStore()

	id, _ := store.Create("sandbox-1")

	// UUID v4 format: 8-4-4-4-12 hex chars.
	s := string(id)
	parts := strings.Split(s, "-")
	if len(parts) != 5 {
		t.Fatalf("ID format: got %q, expected 5 dash-separated parts", s)
	}
	expectedLens := []int{8, 4, 4, 4, 12}
	for i, l := range expectedLens {
		if len(parts[i]) != l {
			t.Errorf("ID part %d: len=%d, want %d (part=%q)", i, len(parts[i]), l, parts[i])
		}
	}
}

// Compile-time check that MemorySnapshotStore satisfies SnapshotStore.
var _ SnapshotStore = (*MemorySnapshotStore)(nil)

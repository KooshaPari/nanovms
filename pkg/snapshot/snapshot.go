// SPDX-License-Identifier: MIT OR Apache-2.0
// Package snapshot defines the API for saving and restoring Firecracker VM
// state. It provides a SnapshotStore interface with an in-memory
// implementation suitable for unit tests.
package snapshot

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"
)

// SnapshotID is a unique identifier for a snapshot.
type SnapshotID string

// Snapshot represents a point-in-time snapshot of a VM's state.
type Snapshot struct {
	// ID is the unique identifier for this snapshot.
	ID SnapshotID `json:"id"`
	// SandboxID is the sandbox this snapshot belongs to.
	SandboxID string `json:"sandbox_id"`
	// CreatedAt is the timestamp when the snapshot was taken.
	CreatedAt time.Time `json:"created_at"`
	// SizeBytes is the size of the snapshot in bytes.
	SizeBytes int64 `json:"size_bytes"`
	// TierID indicates the storage tier (e.g. 0=hot, 1=warm, 2=cold).
	TierID int `json:"tier_id"`
}

// SnapshotStore defines the interface for snapshot persistence operations.
type SnapshotStore interface {
	// Create takes a new snapshot of the given sandbox and returns its ID.
	Create(sandboxID string) (SnapshotID, error)

	// Restore restores a previously-created snapshot into the given sandbox.
	Restore(id SnapshotID, sandboxID string) error

	// List returns all snapshots for the given sandbox, ordered by creation
	// time ascending.
	List(sandboxID string) ([]Snapshot, error)

	// Delete removes a snapshot by ID.
	Delete(id SnapshotID) error

	// Get returns the snapshot metadata for the given ID.
	Get(id SnapshotID) (*Snapshot, error)
}

// ---------------------------------------------------------------------------
// In-memory implementation for testing
// ---------------------------------------------------------------------------

// MemorySnapshotStore is a thread-safe in-memory SnapshotStore useful for
// unit tests. It does not persist data across process restarts.
type MemorySnapshotStore struct {
	mu        sync.RWMutex
	snapshots map[SnapshotID]Snapshot
}

// NewMemorySnapshotStore returns an initialised MemorySnapshotStore.
func NewMemorySnapshotStore() *MemorySnapshotStore {
	return &MemorySnapshotStore{
		snapshots: make(map[SnapshotID]Snapshot),
	}
}

// Create takes a new snapshot, assigns a random UUID as the snapshot ID,
// and stores it in memory.
func (m *MemorySnapshotStore) Create(sandboxID string) (SnapshotID, error) {
	id, err := generateUUID()
	if err != nil {
		return "", fmt.Errorf("snapshot: generate uuid: %w", err)
	}

	s := Snapshot{
		ID:        SnapshotID(id),
		SandboxID: sandboxID,
		CreatedAt: time.Now(),
		SizeBytes: 0, // stub — real impl will populate
		TierID:    0,
	}

	m.mu.Lock()
	m.snapshots[s.ID] = s
	m.mu.Unlock()

	return s.ID, nil
}

// Restore is a no-op stub. The real implementation will replay Firecracker
// snapshot state into the target sandbox.
func (m *MemorySnapshotStore) Restore(_ SnapshotID, _ string) error {
	return nil
}

// List returns all snapshots for the given sandbox, ordered by creation
// time ascending.
func (m *MemorySnapshotStore) List(sandboxID string) ([]Snapshot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []Snapshot
	for _, s := range m.snapshots {
		if s.SandboxID == sandboxID {
			result = append(result, s)
		}
	}

	// Sort by CreatedAt ascending for deterministic output.
	sortSnapshotsByTime(result)
	return result, nil
}

// Delete removes the snapshot with the given ID. Returns an error if the
// snapshot does not exist.
func (m *MemorySnapshotStore) Delete(id SnapshotID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.snapshots[id]; !ok {
		return fmt.Errorf("snapshot: %s not found", id)
	}
	delete(m.snapshots, id)
	return nil
}

// Get returns a pointer to the snapshot with the given ID, or an error if
// not found.
func (m *MemorySnapshotStore) Get(id SnapshotID) (*Snapshot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	s, ok := m.snapshots[id]
	if !ok {
		return nil, fmt.Errorf("snapshot: %s not found", id)
	}
	return &s, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// generateUUID produces a v4-style UUID string (36 chars, no external deps).
func generateUUID() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	// Set version 4 and variant bits per RFC 4122.
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:16]), nil
}

// sortSnapshotsByTime sorts a slice of Snapshots in place by CreatedAt
// ascending using insertion sort (fine for small slices).
func sortSnapshotsByTime(ss []Snapshot) {
	for i := 1; i < len(ss); i++ {
		for j := i; j > 0 && ss[j].CreatedAt.Before(ss[j-1].CreatedAt); j-- {
			ss[j], ss[j-1] = ss[j-1], ss[j]
		}
	}
}

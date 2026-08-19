// SPDX-License-Identifier: MIT OR Apache-2.0
package environment

import "sync"

// StateStore records applied environment contract digests for idempotent apply.
type StateStore interface {
	AppliedDigest(ProfileID) (string, bool)
	RecordApplied(ProfileID, string) error
}

// MemoryStateStore is an in-memory StateStore for tests and local use.
type MemoryStateStore struct {
	mu      sync.Mutex
	applied map[ProfileID]string
}

// AppliedDigest returns the last applied contract digest for one profile.
func (store *MemoryStateStore) AppliedDigest(profile ProfileID) (string, bool) {
	if store == nil {
		return "", false
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	digest, ok := store.applied[profile]
	return digest, ok
}

// RecordApplied stores one applied contract digest.
func (store *MemoryStateStore) RecordApplied(profile ProfileID, digest string) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.applied == nil {
		store.applied = make(map[ProfileID]string)
	}
	store.applied[profile] = digest
	return nil
}

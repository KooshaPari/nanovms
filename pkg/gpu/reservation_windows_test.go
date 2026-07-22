// SPDX-License-Identifier: MIT OR Apache-2.0
//go:build windows

package gpu

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestReservationStoreReserveReleaseReserveCycle(t *testing.T) {
	store := &ReservationStore{
		Path: t.TempDir() + string(os.PathSeparator) + "reservations.json",
	}
	ctx := context.Background()

	lease, err := store.Reserve(ctx, []UUID{testUUIDA}, "owner-a", time.Minute)
	if err != nil {
		t.Fatalf("first reserve: %v", err)
	}
	if err := store.Release(ctx, lease); err != nil {
		t.Fatalf("release: %v", err)
	}
	if _, err := store.Reserve(ctx, []UUID{testUUIDA}, "owner-b", time.Minute); err != nil {
		t.Fatalf("second reserve after release: %v", err)
	}
}

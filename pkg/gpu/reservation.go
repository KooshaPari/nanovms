// SPDX-License-Identifier: MIT OR Apache-2.0
package gpu

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// ReservationStore is a process-safe, file-backed GPU reservation store.
type ReservationStore struct {
	Path string
	Now  func() time.Time
}

// ReservationLease proves ownership of an atomic reservation set.
type ReservationLease struct {
	Token     string    `json:"token"`
	Owner     string    `json:"owner"`
	UUIDs     []UUID    `json:"uuids"`
	ExpiresAt time.Time `json:"expires_at"`
}

type reservationState struct {
	Version      int                        `json:"version"`
	Reservations map[UUID]reservationRecord `json:"reservations"`
}

type reservationRecord struct {
	Token     string    `json:"token"`
	Owner     string    `json:"owner"`
	ExpiresAt time.Time `json:"expires_at"`
}

// BoundReservationPath returns the store path used for GPU leases.
func (store *ReservationStore) BoundReservationPath() string {
	if store == nil {
		return ""
	}
	return store.Path
}

// Reserve atomically reserves every UUID or none of them.
func (store *ReservationStore) Reserve(ctx context.Context, uuids []UUID, owner string, ttl time.Duration) (ReservationLease, error) {
	if store == nil || store.Path == "" {
		return ReservationLease{}, fmt.Errorf("reservation store path is required")
	}
	if owner == "" || ttl <= 0 || len(uuids) == 0 {
		return ReservationLease{}, fmt.Errorf("reservation owner, positive expiry, and at least one GPU are required")
	}
	canonical := append([]UUID(nil), uuids...)
	sort.Slice(canonical, func(i, j int) bool { return canonical[i] < canonical[j] })
	for i, uuid := range canonical {
		if err := uuid.Validate(); err != nil {
			return ReservationLease{}, err
		}
		if i > 0 && canonical[i-1] == uuid {
			return ReservationLease{}, fmt.Errorf("duplicate GPU UUID %q in reservation request", uuid)
		}
	}

	lock, err := store.lock(ctx)
	if err != nil {
		return ReservationLease{}, err
	}
	defer unlockReservationFile(lock)
	state, err := store.readState()
	if err != nil {
		return ReservationLease{}, err
	}
	now := store.now()
	removeExpired(state.Reservations, now)
	for _, uuid := range canonical {
		if current, reserved := state.Reservations[uuid]; reserved {
			return ReservationLease{}, fmt.Errorf("GPU %s is reserved by %q until %s", uuid, current.Owner, current.ExpiresAt.Format(time.RFC3339Nano))
		}
	}
	token, err := randomToken()
	if err != nil {
		return ReservationLease{}, err
	}
	lease := ReservationLease{Token: token, Owner: owner, UUIDs: canonical, ExpiresAt: now.Add(ttl).UTC()}
	for _, uuid := range canonical {
		state.Reservations[uuid] = reservationRecord{Token: token, Owner: owner, ExpiresAt: lease.ExpiresAt}
	}
	if err := store.writeState(state); err != nil {
		return ReservationLease{}, err
	}
	return lease, nil
}

// Release removes only reservations owned by the supplied unexpired token.
func (store *ReservationStore) Release(ctx context.Context, lease ReservationLease) error {
	if store == nil || store.Path == "" {
		return fmt.Errorf("reservation store path is required")
	}
	if lease.Token == "" || lease.Owner == "" || len(lease.UUIDs) == 0 {
		return fmt.Errorf("complete reservation lease is required")
	}
	lock, err := store.lock(ctx)
	if err != nil {
		return err
	}
	defer unlockReservationFile(lock)
	state, err := store.readState()
	if err != nil {
		return err
	}
	removeExpired(state.Reservations, store.now())
	for _, uuid := range lease.UUIDs {
		current, exists := state.Reservations[uuid]
		if !exists {
			continue
		}
		if current.Token != lease.Token || current.Owner != lease.Owner {
			return fmt.Errorf("reservation ownership mismatch for GPU %s", uuid)
		}
	}
	for _, uuid := range lease.UUIDs {
		delete(state.Reservations, uuid)
	}
	return store.writeState(state)
}

// Active returns unexpired reservations in deterministic UUID order.
func (store *ReservationStore) Active(ctx context.Context) ([]ReservationLease, error) {
	if store == nil || store.Path == "" {
		return nil, fmt.Errorf("reservation store path is required")
	}
	lock, err := store.lock(ctx)
	if err != nil {
		return nil, err
	}
	defer unlockReservationFile(lock)
	state, err := store.readState()
	if err != nil {
		return nil, err
	}
	removeExpired(state.Reservations, store.now())
	result := make([]ReservationLease, 0, len(state.Reservations))
	for uuid, record := range state.Reservations {
		result = append(result, ReservationLease{Token: record.Token, Owner: record.Owner, UUIDs: []UUID{uuid}, ExpiresAt: record.ExpiresAt})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].UUIDs[0] < result[j].UUIDs[0] })
	return result, nil
}

func (store *ReservationStore) lock(ctx context.Context) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(store.Path), 0o700); err != nil {
		return nil, fmt.Errorf("create reservation directory: %w", err)
	}
	file, err := os.OpenFile(store.Path+".lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open reservation lock: %w", err)
	}
	var lockWait *time.Timer
	defer func() {
		if lockWait != nil {
			lockWait.Stop()
		}
	}()
	for attempt := 0; ; attempt++ {
		acquired, lockErr := tryLockReservationFile(file)
		if lockErr != nil {
			_ = file.Close()
			return nil, fmt.Errorf("lock reservation store: %w", lockErr)
		}
		if acquired {
			return file, nil
		}
		if err := waitForContendedReservationLock(ctx, &lockWait, attempt); err != nil {
			_ = file.Close()
			return nil, fmt.Errorf("lock reservation store: %w", err)
		}
	}
}

func waitForContendedReservationLock(ctx context.Context, timer **time.Timer, attempt int) error {
	delays := [...]time.Duration{
		time.Millisecond,
		2 * time.Millisecond,
		5 * time.Millisecond,
		10 * time.Millisecond,
		20 * time.Millisecond,
		50 * time.Millisecond,
	}
	delay := delays[len(delays)-1]
	if attempt >= 0 && attempt < len(delays) {
		delay = delays[attempt]
	}
	if *timer == nil {
		*timer = time.NewTimer(delay)
	} else {
		if !(*timer).Stop() {
			select {
			case <-(*timer).C:
			default:
			}
		}
		(*timer).Reset(delay)
	}
	select {
	case <-ctx.Done():
		if !(*timer).Stop() {
			<-(*timer).C
		}
		return ctx.Err()
	case <-(*timer).C:
		return nil
	}
}

func (store *ReservationStore) readState() (*reservationState, error) {
	data, err := os.ReadFile(store.Path)
	if errors.Is(err, os.ErrNotExist) {
		return &reservationState{Version: 1, Reservations: make(map[UUID]reservationRecord)}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read reservation store: %w", err)
	}
	var state reservationState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("decode reservation store: %w", err)
	}
	if state.Version != 1 || state.Reservations == nil {
		return nil, fmt.Errorf("unsupported or malformed reservation store")
	}
	return &state, nil
}

func (store *ReservationStore) writeState(state *reservationState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("encode reservation store: %w", err)
	}
	temp, err := os.CreateTemp(filepath.Dir(store.Path), ".gpu-reservations-*")
	if err != nil {
		return fmt.Errorf("create reservation transaction: %w", err)
	}
	tempName := temp.Name()
	defer func() { _ = os.Remove(tempName) }()
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return fmt.Errorf("write reservation transaction: %w", err)
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return fmt.Errorf("sync reservation transaction: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close reservation transaction: %w", err)
	}
	if err := replaceFile(tempName, store.Path); err != nil {
		return fmt.Errorf("commit reservation transaction: %w", err)
	}
	return nil
}

func (store *ReservationStore) now() time.Time {
	if store.Now != nil {
		return store.Now().UTC()
	}
	return time.Now().UTC()
}

func removeExpired(reservations map[UUID]reservationRecord, now time.Time) {
	for uuid, reservation := range reservations {
		if !reservation.ExpiresAt.After(now) {
			delete(reservations, uuid)
		}
	}
}

func randomToken() (string, error) {
	var bytes [32]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", fmt.Errorf("generate reservation ownership token: %w", err)
	}
	return hex.EncodeToString(bytes[:]), nil
}

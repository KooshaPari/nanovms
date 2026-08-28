package listen

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNewUDSDefault(t *testing.T) {
	dir := t.TempDir()
	ln, err := NewUDS(context.Background(), "routed.sock", dir)
	if err != nil {
		t.Fatalf("NewUDS: %v", err)
	}
	defer func() { _ = ln.Close() }()

	sock := filepath.Join(dir, "routed.sock")
	if _, err := os.Stat(sock); err != nil {
		t.Fatalf("socket not created: %v", err)
	}
	fi, err := os.Stat(sock)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	wantPerm := os.FileMode(0o660)
	if runtime.GOOS == "windows" {
		// Windows reports Unix socket permissions as the platform default; the
		// chmod request is not reflected in Mode().Perm().
		wantPerm = 0o666
	}
	if fi.Mode().Perm() != wantPerm {
		t.Fatalf("socket perms = %o, want %o", fi.Mode().Perm(), wantPerm)
	}
}

func TestNewUDSStaleRemoval(t *testing.T) {
	dir := t.TempDir()
	sock := filepath.Join(dir, "stale.sock")
	if err := os.WriteFile(sock, []byte("x"), 0o600); err != nil {
		t.Fatalf("seed stale: %v", err)
	}
	ln, err := NewUDS(context.Background(), sock, dir)
	if err != nil {
		t.Fatalf("NewUDS over stale: %v", err)
	}
	defer func() { _ = ln.Close() }()

	if _, err := os.Stat(sock); err != nil {
		t.Fatalf("stale socket not replaced: %v", err)
	}
}

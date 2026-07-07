package listen

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNewUDSDefault(t *testing.T) {
	dir := t.TempDir()
	ln, err := NewUDS(context.Background(), "routed.sock", dir)
	if err != nil {
		t.Fatalf("NewUDS: %v", err)
	}
	defer ln.Close()

	sock := filepath.Join(dir, "routed.sock")
	if _, err := os.Stat(sock); err != nil {
		t.Fatalf("socket not created: %v", err)
	}
	fi, err := os.Stat(sock)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if fi.Mode().Perm() != 0o660 {
		t.Fatalf("socket perms = %o, want 660", fi.Mode().Perm())
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
	defer ln.Close()

	if _, err := os.Stat(sock); err != nil {
		t.Fatalf("stale socket not replaced: %v", err)
	}
}

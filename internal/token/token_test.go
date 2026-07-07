package token

import (
	"strings"
	"testing"
)

func TestManager_CheckValid(t *testing.T) {
	m, err := NewFromHex([]string{"deadbeef"})
	if err != nil {
		t.Fatalf("NewFromHex: %v", err)
	}
	if err := m.Check("deadbeef"); err != nil {
		t.Fatalf("Check(valid) = %v, want nil", err)
	}
}

func TestManager_CheckInvalid(t *testing.T) {
	m, err := NewFromHex([]string{"deadbeef"})
	if err != nil {
		t.Fatalf("NewFromHex: %v", err)
	}
	if err := m.Check(""); err == nil {
		t.Fatal("Check(empty) = nil, want error")
	}
	if err := m.Check("cafebabe"); err == nil {
		t.Fatal("Check(wrong) = nil, want error")
	}
}

func TestMintToken(t *testing.T) {
	tok, err := MintToken()
	if err != nil {
		t.Fatalf("MintToken: %v", err)
	}
	if len(tok) != 64 {
		t.Fatalf("MintToken len = %d, want 64 (32 bytes hex)", len(tok))
	}
	if strings.Trim(tok, "0123456789abcdef") != "" {
		t.Fatalf("MintToken not hex: %q", tok)
	}
}

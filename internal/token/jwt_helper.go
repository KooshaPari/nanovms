package token

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// ── JWT helper functions (no external deps, stdlib only) ──────────

// splitToken splits "a.b.c" into three parts.
func splitToken(raw string) []string {
	return strings.SplitN(raw, ".", 3)
}

// decodeSegment decodes a URL-safe base64-encoded JWT segment.
func decodeSegment(seg string) ([]byte, error) {
	// Restore padding
	switch len(seg) % 4 {
	case 2:
		seg += "=="
	case 3:
		seg += "="
	}
	return base64.URLEncoding.DecodeString(seg)
}

// decodeSig decodes the signature segment (no padding fix needed).
func decodeSig(seg string) []byte {
	switch len(seg) % 4 {
	case 2:
		seg += "=="
	case 3:
		seg += "="
	}
	b, _ := base64.URLEncoding.DecodeString(seg)
	return b
}

type jwtHeader struct {
	Alg string `json:"alg"`
	KID string `json:"kid,omitempty"`
}

func parseHeader(raw []byte) (alg, kid string, err error) {
	var h jwtHeader
	if err := json.Unmarshal(raw, &h); err != nil {
		return "", "", fmt.Errorf("json: %w", err)
	}
	if h.Alg == "" {
		return "", "", fmt.Errorf("missing 'alg' in header")
	}
	return h.Alg, h.KID, nil
}

func parseClaims(raw []byte) (*JWTClaims, error) {
	var c JWTClaims
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("json: %w", err)
	}
	return &c, nil
}

func sha256Hash(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

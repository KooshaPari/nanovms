// SPDX-License-Identifier: MIT OR Apache-2.0
package phenointegration

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// newRequestID returns a canonical RFC-4122 version-4 UUID rendered as
// the 36-character 8-4-4-4-12 hex form, e.g.
//
//	550e8400-e29b-41d4-a716-446655440000
//
// We implement UUID v4 directly (no third-party dep) because the
// generation is 16 bytes of crypto/rand plus a few bit-twiddles. The
// returned string is suitable for the X-Request-Id header and for log
// correlation across services.
//
// crypto/rand should never fail on a healthy system. If it does, we
// return a deterministic, well-formed non-empty marker so the
// middleware still attaches *some* id and downstream handlers don't
// observe an empty value.
func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "00000000-0000-4000-8000-000000000000"
	}
	// Set version (4) and variant (10xx) nibbles per RFC 4122 §4.4.
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10xx
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(b[0:4]),
		hex.EncodeToString(b[4:6]),
		hex.EncodeToString(b[6:8]),
		hex.EncodeToString(b[8:10]),
		hex.EncodeToString(b[10:16]),
	)
}

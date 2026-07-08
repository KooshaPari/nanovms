package token

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"
)

// JWTClaims represents the standard claims we care about.
type JWTClaims struct {
	Subject   string `json:"sub"`
	Issuer    string `json:"iss"`
	Audience  string `json:"aud"`
	Expiry    int64  `json:"exp"`
	NotBefore int64  `json:"nbf"`
	IssuedAt  int64  `json:"iat"`
}

// JWTVerifier verifies RS256/ES256 JWTs against an OIDC issuer.
type JWTVerifier struct {
	issuer   string
	audience string
	jwkSet   *JWKSet
}

// JWKSet holds a single cached RSA or ECDSA public key from the OIDC issuer.
type JWKSet struct {
	key any // *rsa.PublicKey or *ecdsa.PublicKey
	kid string
}

// NewJWTVerifier creates a verifier that validates tokens signed by issuer for audience.
// The key argument is a PEM-encoded RSA or ECDSA public key.
func NewJWTVerifier(issuer, audience string, pemKey []byte) (*JWTVerifier, error) {
	block, _ := pem.Decode(pemKey)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found in public key")
	}

	var key any
	var err error

	switch block.Type {
	case "PUBLIC KEY":
		key, err = x509.ParsePKIXPublicKey(block.Bytes)
	case "RSA PUBLIC KEY":
		key, err = x509.ParsePKCS1PublicKey(block.Bytes)
	case "EC PRIVATE KEY", "RSA PRIVATE KEY":
		// For testing purposes, accept private keys too
		key, err = parsePrivateKey(block)
	default:
		return nil, fmt.Errorf("unsupported PEM block type: %s", block.Type)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	return &JWTVerifier{
		issuer:   issuer,
		audience: audience,
		jwkSet:   &JWKSet{key: key, kid: ""},
	}, nil
}

func parsePrivateKey(block *pem.Block) (any, error) {
	switch block.Type {
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	case "EC PRIVATE KEY":
		return x509.ParseECPrivateKey(block.Bytes)
	}
	return nil, fmt.Errorf("unknown private key type: %s", block.Type)
}

// Verify parses and validates a JWT. It returns the claims on success.
// Supported: RS256 (RSASSA-PKCS1-v1_5 with SHA-256) and ES256 (ECDSA with P-256 and SHA-256).
func (v *JWTVerifier) Verify(tokenStr string) (*JWTClaims, error) {
	// Quick length sanity
	if len(tokenStr) < 20 || len(tokenStr) > 32768 {
		return nil, fmt.Errorf("invalid token length")
	}

	// Split into parts
	parts := splitToken(tokenStr)
	if len(parts) != 3 {
		return nil, fmt.Errorf("token must have 3 parts, got %d", len(parts))
	}

	headerJSON, err := decodeSegment(parts[0])
	if err != nil {
		return nil, fmt.Errorf("header decode: %w", err)
	}
	payloadJSON, err := decodeSegment(parts[1])
	if err != nil {
		return nil, fmt.Errorf("payload decode: %w", err)
	}
	sig := decodeSig(parts[2])

	// Parse header for algorithm + kid
	alg, _, err := parseHeader(headerJSON)
	if err != nil {
		return nil, fmt.Errorf("header parse: %w", err)
	}

	// Parse claims from payload
	claims, err := parseClaims(payloadJSON)
	if err != nil {
		return nil, fmt.Errorf("claims parse: %w", err)
	}

	// Verify issuer
	if claims.Issuer != v.issuer {
		return nil, fmt.Errorf("unexpected issuer: %q (expected %q)", claims.Issuer, v.issuer)
	}

	// Verify audience
	if claims.Audience != v.audience {
		return nil, fmt.Errorf("unexpected audience: %q (expected %q)", claims.Audience, v.audience)
	}

	// Verify expiry
	now := time.Now().Unix()
	if claims.Expiry > 0 && now >= claims.Expiry {
		return nil, fmt.Errorf("token expired at %d (now %d)", claims.Expiry, now)
	}

	// Verify not-before
	if claims.NotBefore > 0 && now < claims.NotBefore {
		return nil, fmt.Errorf("token not yet valid until %d (now %d)", claims.NotBefore, now)
	}

	// Verify signature
	signingInput := parts[0] + "." + parts[1]
	if err := verifySignature(v.jwkSet.key, alg, signingInput, sig); err != nil {
		return nil, fmt.Errorf("signature verification: %w", err)
	}

	return claims, nil
}

// verifySignature dispatches to the correct verifier based on key type.
func verifySignature(pub any, alg string, data string, sig []byte) error {
	switch k := pub.(type) {
	case *rsa.PublicKey:
		if alg != "RS256" {
			return fmt.Errorf("unsupported RSA algorithm: %s", alg)
		}
		hashed := sha256Hash([]byte(data))
		return rsa.VerifyPKCS1v15(k, crypto.SHA256, hashed, sig)

	case *ecdsa.PublicKey:
		if alg != "ES256" {
			return fmt.Errorf("unsupported ECDSA algorithm: %s", alg)
		}
		hashed := sha256Hash([]byte(data))
		if !ecdsa.VerifyASN1(k, hashed, sig) {
			return fmt.Errorf("ECDSA signature invalid")
		}
		return nil

	default:
		return fmt.Errorf("unsupported key type: %T", pub)
	}
}

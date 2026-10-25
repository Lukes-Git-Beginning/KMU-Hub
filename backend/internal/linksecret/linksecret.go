// Package linksecret draws and hashes the secrets behind public share links.
//
// A share token is the whole credential of an unauthenticated route, so it is
// drawn the same way everywhere (32 bytes of crypto/rand, base64url-encoded to
// 43 URL-safe characters) and stored only as its SHA-256. A database read --
// a backup, a wrong join -- then yields no usable link. The plain token leaves
// the server exactly once, in the response that mints it.
package linksecret

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// tokenBytes is the entropy of one link secret.
const tokenBytes = 32

// New draws a link secret. base64url keeps it usable as a URL path segment
// without escaping.
func New() (string, error) {
	buf := make([]byte, tokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate link secret: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// Hash is the stored form of a link secret: hex-encoded SHA-256 of its UTF-8
// bytes. It matches `encode(sha256(convert_to(token, 'UTF8')), 'hex')`, which
// is how the migrations backfilled tokens minted before the hash existed. A
// plain hash, no salt: the secret has 256 bits of entropy, so there is nothing
// to brute-force, and the lookup must stay an indexed equality match.
func Hash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

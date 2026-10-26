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
	"errors"
	"fmt"
	"time"
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

// MaxLifetime bounds how long a share link may stay valid (decision
// 2026-10-25): an external path that outlives anyone's memory of opening it is
// the failure mode, so no link is minted without an end.
const MaxLifetime = 365 * 24 * time.Hour

// ErrLifetimeTooLong refuses a requested expiry past MaxLifetime.
var ErrLifetimeTooLong = errors.New("share link may stay valid for at most 365 days")

// Expiry resolves the end of a link minted at now: no request means
// now+MaxLifetime, a request past that is refused. Whether a requested expiry
// lies in the future stays the caller's check, each service words it itself.
func Expiry(requested *time.Time, now time.Time) (time.Time, error) {
	limit := now.Add(MaxLifetime)
	if requested == nil {
		return limit, nil
	}
	if requested.After(limit) {
		return time.Time{}, ErrLifetimeTooLong
	}
	return *requested, nil
}

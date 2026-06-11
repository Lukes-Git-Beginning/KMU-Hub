package gateway

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
)

const (
	// bexioStateExpiry is the maximum lifetime of a Bexio OAuth state token.
	// Bexio redirects back within seconds; 10 minutes is generous but bounded.
	bexioStateExpiry = 10 * time.Minute

	// bexioStateNonceLen is the byte length of the random nonce embedded in the token.
	bexioStateNonceLen = 16
)

var (
	// ErrBexioStateInvalid is returned when the state token fails signature verification,
	// has an invalid format, or is expired.
	ErrBexioStateInvalid = errors.New("bexio: invalid or expired state token")
)

// encodeBexioState creates a signed, expiring state token for the Bexio OAuth flow.
//
// Token format (before base64):
//
//	{tenant_id}|{nonce_hex}|{expiry_unix}
//
// The HMAC-SHA256 signature covers the payload and is appended after a "." separator:
//
//	base64url(payload) + "." + base64url(sig)
//
// Note: This provides HMAC+Expiry protection. It does not implement Nonce-Replay
// prevention (no server-side nonce store). The 10-minute window and HMAC binding
// to tenant_id limit the practical replay risk to the same tenant within the expiry
// window; a full nonce store (Redis/DB) can be added when available in the gateway.
func encodeBexioState(secret, tenantID string) (string, error) {
	nonceBytes := make([]byte, bexioStateNonceLen)
	if _, err := rand.Read(nonceBytes); err != nil {
		return "", fmt.Errorf("bexio state: generate nonce: %w", err)
	}
	nonce := base64.RawURLEncoding.EncodeToString(nonceBytes)

	expiry := time.Now().Add(bexioStateExpiry).Unix()
	payload := fmt.Sprintf("%s|%s|%d", tenantID, nonce, expiry)

	sig := computeBexioHMAC(secret, payload)
	encoded := base64.RawURLEncoding.EncodeToString([]byte(payload))
	return encoded + "." + sig, nil
}

// decodeBexioState verifies and decodes a Bexio OAuth state token.
// Returns the tenant_id extracted from the token.
func decodeBexioState(secret, token string) (tenantID string, err error) {
	dot := strings.LastIndex(token, ".")
	if dot < 0 {
		slog.Warn("bexio: state token missing signature separator")
		return "", ErrBexioStateInvalid
	}

	encodedPayload := token[:dot]
	givenSig := token[dot+1:]

	payloadBytes, err := base64.RawURLEncoding.DecodeString(encodedPayload)
	if err != nil {
		slog.Warn("bexio: state token payload base64 decode failed", "error", err)
		return "", ErrBexioStateInvalid
	}
	payload := string(payloadBytes)

	// Constant-time HMAC verification
	expectedSig := computeBexioHMAC(secret, payload)
	if !hmac.Equal([]byte(givenSig), []byte(expectedSig)) {
		slog.Warn("bexio: state token HMAC mismatch — possible CSRF or tampering")
		return "", ErrBexioStateInvalid
	}

	parts := strings.Split(payload, "|")
	if len(parts) != 3 {
		slog.Warn("bexio: state token malformed payload", "parts", len(parts))
		return "", ErrBexioStateInvalid
	}

	tenantID = parts[0]
	expiryUnix, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		slog.Warn("bexio: state token expiry parse failed", "error", err)
		return "", ErrBexioStateInvalid
	}

	if time.Now().Unix() > expiryUnix {
		slog.Warn("bexio: state token expired", "expiry_unix", expiryUnix)
		return "", ErrBexioStateInvalid
	}

	return tenantID, nil
}

// computeBexioHMAC returns the base64url-encoded HMAC-SHA256 of payload under secret.
func computeBexioHMAC(secret, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

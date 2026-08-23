package gateway

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

const testStateSecret = "test-secret-minimum-32-characters-long"

func TestEncodeBexioState_RoundTrip(t *testing.T) {
	tenantID := "123e4567-e89b-12d3-a456-426614174000"

	token, err := encodeBexioState(testStateSecret, tenantID)
	if err != nil {
		t.Fatalf("encodeBexioState: %v", err)
	}

	// Token must contain exactly one "." separator
	if !strings.Contains(token, ".") {
		t.Fatalf("token missing signature separator: %q", token)
	}

	got, err := decodeBexioState(testStateSecret, token)
	if err != nil {
		t.Fatalf("decodeBexioState: %v", err)
	}
	if got != tenantID {
		t.Errorf("tenant_id mismatch: got %q, want %q", got, tenantID)
	}
}

func TestDecodeBexioState_ManipulatedSignature(t *testing.T) {
	tenantID := "123e4567-e89b-12d3-a456-426614174000"

	token, err := encodeBexioState(testStateSecret, tenantID)
	if err != nil {
		t.Fatalf("encodeBexioState: %v", err)
	}

	// Flip a byte in the signature part. The replacement char must differ from the
	// original — the nonce is random per call, so the original byte at this position
	// is uniformly distributed over the base64url alphabet and could already be "X".
	dot := strings.LastIndex(token, ".")
	replacement := byte('X')
	if token[dot+1] == replacement {
		replacement = 'Y'
	}
	tampered := token[:dot+1] + string(replacement) + token[dot+2:]

	_, err = decodeBexioState(testStateSecret, tampered)
	if err == nil {
		t.Fatal("expected error for tampered signature, got nil")
	}
	if err != ErrBexioStateInvalid {
		t.Errorf("expected ErrBexioStateInvalid, got %v", err)
	}
}

func TestDecodeBexioState_Expired(t *testing.T) {
	// Forge a token with a valid HMAC but an expired timestamp (unix=1).
	tenantID := "123e4567-e89b-12d3-a456-426614174000"
	payload := tenantID + "|deadbeefnonce|1" // unix timestamp 1 = Jan 1970

	sig := computeBexioHMAC(testStateSecret, payload)
	encodedPayload := base64.RawURLEncoding.EncodeToString([]byte(payload))
	token := encodedPayload + "." + sig

	_, err := decodeBexioState(testStateSecret, token)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
	if err != ErrBexioStateInvalid {
		t.Errorf("expected ErrBexioStateInvalid, got %v", err)
	}
}

func TestDecodeBexioState_Garbage(t *testing.T) {
	cases := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"no dot separator", "aGVsbG8"},
		{"bad base64 payload", "!!!.sig"},
		{"wrong number of parts", base64.RawURLEncoding.EncodeToString([]byte("onlyone")) + ".sig"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := decodeBexioState(testStateSecret, tc.token)
			if err == nil {
				t.Fatalf("expected error for token %q, got nil", tc.token)
			}
		})
	}
}

func TestDecodeBexioState_WrongSecret(t *testing.T) {
	tenantID := "123e4567-e89b-12d3-a456-426614174000"

	token, err := encodeBexioState(testStateSecret, tenantID)
	if err != nil {
		t.Fatalf("encodeBexioState: %v", err)
	}

	_, err = decodeBexioState("different-secret-also-long-enough-here", token)
	if err == nil {
		t.Fatal("expected error for wrong secret, got nil")
	}
	if err != ErrBexioStateInvalid {
		t.Errorf("expected ErrBexioStateInvalid, got %v", err)
	}
}

func TestEncodeBexioState_Uniqueness(t *testing.T) {
	// Two tokens for the same tenant must differ (nonce randomness)
	tenantID := "123e4567-e89b-12d3-a456-426614174000"

	t1, err := encodeBexioState(testStateSecret, tenantID)
	if err != nil {
		t.Fatalf("first encode: %v", err)
	}
	t2, err := encodeBexioState(testStateSecret, tenantID)
	if err != nil {
		t.Fatalf("second encode: %v", err)
	}
	if t1 == t2 {
		t.Error("two tokens for the same tenant must differ due to nonce randomness")
	}
}

func TestBexioStateExpiry_Constant(t *testing.T) {
	if bexioStateExpiry != 10*time.Minute {
		t.Errorf("unexpected expiry constant: %v", bexioStateExpiry)
	}
}

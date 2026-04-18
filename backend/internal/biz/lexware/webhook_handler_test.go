package lexware

import (
	"testing"
)

func TestValidateHMAC(t *testing.T) {
	const secret = "test-webhook-secret-abc123"
	body := []byte(`{"eventType":"contact.created","resourceId":"abc"}`)

	t.Run("valid signature accepted", func(t *testing.T) {
		sig := ComputeHMAC(body, secret)
		if !ValidateHMAC(body, sig, secret) {
			t.Error("ValidateHMAC() = false for a correctly signed payload, want true")
		}
	})

	t.Run("invalid signature rejected", func(t *testing.T) {
		// Tamper with the payload digest.
		sig := hmacSignaturePrefix + "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
		if ValidateHMAC(body, sig, secret) {
			t.Error("ValidateHMAC() = true for an invalid signature, want false")
		}
	})

	t.Run("missing prefix rejected", func(t *testing.T) {
		// Provide a raw hex digest without the required prefix.
		sig := "abc123noprefixhere"
		if ValidateHMAC(body, sig, secret) {
			t.Error("ValidateHMAC() = true for a signature without the hmac-sha256= prefix, want false")
		}
	})

	t.Run("wrong secret rejected", func(t *testing.T) {
		sig := ComputeHMAC(body, secret)
		if ValidateHMAC(body, sig, "wrong-secret") {
			t.Error("ValidateHMAC() = true when validated with a different secret, want false")
		}
	})

	t.Run("tampered body rejected", func(t *testing.T) {
		sig := ComputeHMAC(body, secret)
		tamperedBody := []byte(`{"eventType":"contact.changed","resourceId":"abc"}`)
		if ValidateHMAC(tamperedBody, sig, secret) {
			t.Error("ValidateHMAC() = true for a tampered body, want false")
		}
	})

	t.Run("empty body with matching signature accepted", func(t *testing.T) {
		emptyBody := []byte{}
		sig := ComputeHMAC(emptyBody, secret)
		if !ValidateHMAC(emptyBody, sig, secret) {
			t.Error("ValidateHMAC() = false for empty body with correct signature, want true")
		}
	})

	t.Run("malformed hex in signature rejected", func(t *testing.T) {
		sig := hmacSignaturePrefix + "not-valid-hex!"
		if ValidateHMAC(body, sig, secret) {
			t.Error("ValidateHMAC() = true for a signature with invalid hex, want false")
		}
	})
}

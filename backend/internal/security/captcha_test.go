package security

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCaptchaVerifier_Enabled(t *testing.T) {
	t.Run("disabled when secret is empty", func(t *testing.T) {
		v := NewCaptchaVerifier("", "https://example.com/siteverify")
		if v.Enabled() {
			t.Error("Enabled() = true, want false for empty secret")
		}
	})

	t.Run("enabled when secret is set", func(t *testing.T) {
		v := NewCaptchaVerifier("my-secret", "https://example.com/siteverify")
		if !v.Enabled() {
			t.Error("Enabled() = false, want true for non-empty secret")
		}
	})
}

func TestCaptchaVerifier_Verify_Disabled(t *testing.T) {
	v := NewCaptchaVerifier("", "https://example.com/siteverify")
	ok, err := v.Verify(context.Background(), "any-token", "1.2.3.4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("Verify() = false, want true when verifier is disabled")
	}
}

func TestCaptchaVerifier_Verify_EmptyToken(t *testing.T) {
	// Secret set, but token is empty → immediate false without network call.
	v := NewCaptchaVerifier("secret", "https://example.com/siteverify")
	ok, err := v.Verify(context.Background(), "", "1.2.3.4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("Verify() = true, want false for empty token with enabled verifier")
	}
}

func TestCaptchaVerifier_Verify_UpstreamSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}))
	defer srv.Close()

	v := NewCaptchaVerifier("test-secret", srv.URL)
	ok, err := v.Verify(context.Background(), "valid-token", "1.2.3.4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("Verify() = false, want true when upstream returns success:true")
	}
}

func TestCaptchaVerifier_Verify_UpstreamFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]bool{"success": false})
	}))
	defer srv.Close()

	v := NewCaptchaVerifier("test-secret", srv.URL)
	ok, err := v.Verify(context.Background(), "invalid-token", "1.2.3.4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("Verify() = true, want false when upstream returns success:false")
	}
}

package middleware

import (
	"net/http/httptest"
	"testing"
)

// The audit entry stores the client IP in an inet column: host:port is refused there and the
// whole entry is lost, so the port has to go when no proxy header names the client.
func TestAuditClientIP_DropsThePortFromRemoteAddr(t *testing.T) {
	for remote, want := range map[string]string{
		"[::1]:52502":      "::1",
		"203.0.113.7:4711": "203.0.113.7",
		"203.0.113.7":      "203.0.113.7",
	} {
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = remote
		if got := auditClientIP(r); got != want {
			t.Errorf("auditClientIP(%q) = %q, want %q", remote, got, want)
		}
	}

	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "[::1]:52502"
	r.Header.Set("X-Forwarded-For", "198.51.100.1, 10.0.0.1")
	if got := auditClientIP(r); got != "198.51.100.1" {
		t.Errorf("X-Forwarded-For first hop = %q, want 198.51.100.1", got)
	}
}

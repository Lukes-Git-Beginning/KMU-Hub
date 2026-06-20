package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRateLimiter_InMemoryFallback(t *testing.T) {
	// nil redis client triggers in-memory fallback
	rl := NewRateLimiter(nil, 3, false)

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code, "request %d should be allowed", i+1)
	}

	// 4th request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Equal(t, "1", rec.Header().Get("Retry-After"))
}

// TestRateLimiterWithPrefix_IndependentCounters verifies that two limiters
// created with different prefixes maintain separate in-memory counters for the
// same client IP. This is the core property required by the global vs. public
// rate-limiter separation (WP-F).
func TestRateLimiterWithPrefix_IndependentCounters(t *testing.T) {
	// RPS=1 so the second request from the same IP triggers the limit.
	globalLimiter := NewRateLimiterWithPrefix(nil, 1, "ratelimit", false)
	publicLimiter := NewRateLimiterWithPrefix(nil, 1, "ratelimit:public", false)

	ip := "10.1.2.3:5000"
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	globalHandler := globalLimiter.Middleware(okHandler)
	publicHandler := publicLimiter.Middleware(okHandler)

	// First request on globalLimiter — should pass.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = ip
	rec := httptest.NewRecorder()
	globalHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("global limiter first request: got %d, want 200", rec.Code)
	}

	// Second request on globalLimiter — exhausted (RPS=1).
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = ip
	rec = httptest.NewRecorder()
	globalHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("global limiter second request: got %d, want 429 (counter exhausted)", rec.Code)
	}

	// First request on publicLimiter with same IP — counter is independent, must pass.
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = ip
	rec = httptest.NewRecorder()
	publicHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("public limiter first request: got %d, want 200 (independent counter)", rec.Code)
	}
}

func TestRateLimiter_DifferentIPs(t *testing.T) {
	rl := NewRateLimiter(nil, 1, false)

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First IP
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.RemoteAddr = "10.0.0.1:1234"
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	assert.Equal(t, http.StatusOK, rec1.Code)

	// Second IP should also be allowed
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "10.0.0.2:1234"
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusOK, rec2.Code)
}

func TestClientIPTrusted(t *testing.T) {
	cases := []struct {
		name        string
		behindProxy bool
		xff         string
		remoteAddr  string
		want        string
	}{
		{"direct exposure ignores XFF", false, "1.2.3.4", "203.0.113.9:5000", "203.0.113.9"},
		{"direct exposure no XFF uses RemoteAddr", false, "", "203.0.113.9:5000", "203.0.113.9"},
		{"behind proxy single XFF entry", true, "198.51.100.7", "10.0.0.5:5000", "198.51.100.7"},
		{"behind proxy uses last (proxy-appended) entry", true, "1.2.3.4, 198.51.100.7", "10.0.0.5:5000", "198.51.100.7"},
		{"behind proxy spoofed leftmost is ignored", true, "evil-spoof, 9.9.9.9, 198.51.100.7", "10.0.0.5:5000", "198.51.100.7"},
		{"behind proxy no XFF falls back to RemoteAddr", true, "", "10.0.0.5:5000", "10.0.0.5"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tc.remoteAddr
			if tc.xff != "" {
				req.Header.Set("X-Forwarded-For", tc.xff)
			}
			assert.Equal(t, tc.want, ClientIPTrusted(req, tc.behindProxy))
		})
	}
}

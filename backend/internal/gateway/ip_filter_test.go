package gateway

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestIPFilter_ColdStartFailsOpen verifies the cold-start decision directly:
// when rules have never loaded, a fetch failure fails OPEN (no error, no rules)
// so the gateway does not 503 every request — including /auth/login — while auth
// is still coming up. Once rules have loaded, a failure that leaves the cache
// stale beyond the max staleness window fails CLOSED.
func TestIPFilter_ColdStartFailsOpen(t *testing.T) {
	f := &IPFilterMiddleware{} // rulesEverLoaded == false

	rules, err := f.handleFetchFailure(errors.New("auth unreachable"))
	if err != nil {
		t.Fatalf("cold start must fail open (nil error), got %v", err)
	}
	if rules != nil {
		t.Fatalf("cold start must return no rules, got %d", len(rules))
	}

	// After a successful load, a failure beyond max staleness must fail closed.
	f.rulesEverLoaded = true
	f.lastLoad = time.Now().Add(-2 * ipRuleMaxStaleness)
	if _, err := f.handleFetchFailure(errors.New("auth unreachable")); err == nil {
		t.Fatal("stale-beyond-max must fail closed (return error)")
	}
}

// TestIPFilterMiddleware_ColdStartAllowsRequest exercises the full middleware
// path on a cold start: the auth service is not reachable (not registered), so
// rule loading fails, and the request — including a login — must still pass.
func TestIPFilterMiddleware_ColdStartAllowsRequest(t *testing.T) {
	reg := NewServiceRegistry(nil) // no "auth" service → GetConnection fails
	f := NewIPFilterMiddleware(reg, false)

	nextCalled := false
	h := f.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req.RemoteAddr = "203.0.113.5:443"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if !nextCalled || rec.Code != http.StatusOK {
		t.Fatalf("cold start must allow the request (fail open): code=%d nextCalled=%v", rec.Code, nextCalled)
	}
}

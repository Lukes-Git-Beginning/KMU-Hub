package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/featureflag"
	"github.com/kmuhub/kmuhub/internal/middleware"
)

// testTenantID is a stable test tenant UUID used by all handler unit tests.
var testTenantID = uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

// emptyRegistry returns a ServiceRegistry with no registered services.
// Any handler that calls getXClient will get a "service not registered" error.
func emptyRegistry() *ServiceRegistry {
	return NewServiceRegistry(nil)
}

// registryWithService returns a ServiceRegistry with a single service registered.
// The address is a dummy localhost address — gRPC calls will fail at RPC time,
// but getXClient will succeed (grpc.NewClient is non-blocking).
func registryWithService(name string) *ServiceRegistry {
	reg := NewServiceRegistry(nil)
	reg.Register(name, "localhost:0")
	return reg
}

// noFlags returns a featureflag.Registry with every flag at its default value
// (no env overrides). Use for handler tests that need a flags dependency but
// don't exercise flag-gated behaviour.
func noFlags() *featureflag.Registry {
	return featureflag.NewRegistry().Load(func(string) string { return "" })
}

// withUserID creates a request with a user ID set in the context (as the auth middleware would).
func withUserID(r *http.Request, userID string) *http.Request {
	ctx := context.WithValue(r.Context(), middleware.UserIDKey, userID)
	return r.WithContext(ctx)
}

// withTenantID creates a request with a tenant ID set in the context (as the auth middleware would).
// Handlers that call middleware.GetTenantID require this in unit tests.
func withTenantID(r *http.Request, tenantID uuid.UUID) *http.Request {
	ctx := context.WithValue(r.Context(), middleware.TenantIDKey, tenantID.String())
	return r.WithContext(ctx)
}

// withAuth creates a request with both user ID and tenant ID in the context,
// mirroring what the Auth middleware injects in production.
func withAuth(r *http.Request, userID string, tenantID uuid.UUID) *http.Request {
	return withTenantID(withUserID(r, userID), tenantID)
}

// withChiURLParam sets a chi URL parameter on the request context.
// withChiURLParam adds a path parameter, reusing one already on the request
// instead of replacing it: a route with two ids (/users/{id}/roles/{roleId})
// needs both, and a fresh route context per call would drop the first.
func withChiURLParam(r *http.Request, key, value string) *http.Request {
	rctx, ok := r.Context().Value(chi.RouteCtxKey).(*chi.Context)
	if !ok {
		rctx = chi.NewRouteContext()
	}
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// jsonBody creates a reader from a JSON-serializable value.
func jsonBody(t *testing.T, v interface{}) *bytes.Reader {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal json: %v", err)
	}
	return bytes.NewReader(data)
}

// invalidJSON returns a reader with invalid JSON content.
func invalidJSON() *strings.Reader {
	return strings.NewReader("{invalid json")
}

// assertStatus checks the HTTP response status code.
func assertStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Errorf("status = %d, want %d; body = %s", rec.Code, want, rec.Body.String())
	}
}

// assertErrorContains checks that the response contains a JSON error message with the given substring.
func assertErrorContains(t *testing.T, rec *httptest.ResponseRecorder, substr string) {
	t.Helper()
	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if !strings.Contains(resp["error"], substr) {
		t.Errorf("error = %q, want to contain %q", resp["error"], substr)
	}
}

// validationErrorBody mirrors the structured validation error response shape
// produced by decodeAndValidate (see internal/validation.ErrorBody).
type validationErrorBody struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Details []struct {
		Field string `json:"field"`
		Rule  string `json:"rule"`
		Param string `json:"param"`
	} `json:"details"`
}

// assertValidationError checks that the response is a structured validation
// failure (HTTP 400, code "validation_failed") whose details include the given
// field name. Use this for inputs rejected by decodeAndValidate; for malformed
// JSON (which yields the plain {"error": "invalid request body"} shape) use
// assertErrorContains instead.
func assertValidationError(t *testing.T, rec *httptest.ResponseRecorder, wantField string) {
	t.Helper()
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	var body validationErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode validation error response: %v; body = %s", err, rec.Body.String())
	}
	if body.Code != "validation_failed" {
		t.Errorf("code = %q, want %q; body = %s", body.Code, "validation_failed", rec.Body.String())
	}
	if body.Error == "" {
		t.Errorf("error string must be non-empty; body = %s", rec.Body.String())
	}
	for _, d := range body.Details {
		if d.Field == wantField {
			return
		}
	}
	t.Errorf("validation details missing field %q; body = %s", wantField, rec.Body.String())
}

// testServiceUnavailable is a generic test for any handler that gets a gRPC client first.
// It verifies that when the service is not registered, the handler returns 503.
// A valid TenantID is set in the context so that the tenant check does not interfere.
func testServiceUnavailable(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/", strings.NewReader("{}"))
	req = withTenantID(req, testTenantID)
	handler(rec, req)
	assertStatus(t, rec, http.StatusServiceUnavailable)
}

// withAuthRequired wraps a handler with the RequireAuthenticated middleware.
// Use this in _NoUserID tests that call handlers directly (bypassing the chi router).
// The wrapped handler returns 401 when no user ID is in the context, mirroring
// what RequireAuthenticated does at the route group level in production.
func withAuthRequired(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		RequireAuthenticated(h).ServeHTTP(w, r)
	}
}

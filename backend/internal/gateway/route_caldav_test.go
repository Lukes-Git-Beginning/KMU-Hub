package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/middleware"
)

// fakeCalDAVPasswordEntry is one password tracked by fakeCalDAVPasswordService.
type fakeCalDAVPasswordEntry struct {
	userID   uuid.UUID
	password string
	revoked  bool
}

// fakeCalDAVPasswordService is an in-memory stand-in for the real
// caldav.AppPasswordService, used so handleTestConnection can be exercised
// without a database.
type fakeCalDAVPasswordService struct {
	mu           sync.Mutex
	passwords    map[uuid.UUID]*fakeCalDAVPasswordEntry
	orgEnabled   bool
	forceInvalid bool // when true, Validate always rejects (simulates an auth failure)
}

func newFakeCalDAVPasswordService() *fakeCalDAVPasswordService {
	return &fakeCalDAVPasswordService{
		passwords:  make(map[uuid.UUID]*fakeCalDAVPasswordEntry),
		orgEnabled: true,
	}
}

func (f *fakeCalDAVPasswordService) Create(_ context.Context, userID, _ uuid.UUID, label string) (string, *CalDAVPasswordInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	id := uuid.New()
	plaintext := uuid.NewString()
	f.passwords[id] = &fakeCalDAVPasswordEntry{userID: userID, password: plaintext}
	return plaintext, &CalDAVPasswordInfo{ID: id, Label: label, PasswordPrefix: plaintext[:4]}, nil
}

func (f *fakeCalDAVPasswordService) Validate(_ context.Context, username, password string) (uuid.UUID, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.orgEnabled || f.forceInvalid {
		return uuid.Nil, errors.New("caldav disabled")
	}
	userID, err := uuid.Parse(username)
	if err != nil {
		return uuid.Nil, errors.New("invalid username")
	}
	for _, pw := range f.passwords {
		if pw.userID == userID && !pw.revoked && pw.password == password {
			return userID, nil
		}
	}
	return uuid.Nil, errors.New("invalid credentials")
}

func (f *fakeCalDAVPasswordService) List(_ context.Context, _ uuid.UUID) ([]*CalDAVPasswordInfo, error) {
	return nil, nil
}

func (f *fakeCalDAVPasswordService) Revoke(_ context.Context, id, _ uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	pw, ok := f.passwords[id]
	if !ok {
		return errors.New("not found")
	}
	pw.revoked = true
	return nil
}

func (f *fakeCalDAVPasswordService) IsOrgEnabled(_ context.Context) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.orgEnabled, nil
}

func (f *fakeCalDAVPasswordService) SetOrgEnabled(_ context.Context, enabled bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.orgEnabled = enabled
	return nil
}

// activePasswordCount returns the number of non-revoked passwords, used to
// prove the ephemeral test password was actually revoked after the probe.
func (f *fakeCalDAVPasswordService) activePasswordCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, pw := range f.passwords {
		if !pw.revoked {
			n++
		}
	}
	return n
}

// noopCtxInjector stands in for caldavpkg.NewCtxInjector(pool) on the tests
// that are not about tenant resolution.
func noopCtxInjector(ctx context.Context, _ uuid.UUID) (context.Context, error) { return ctx, nil }

// withCalDAVAuth injects a user/tenant context the way the real JWT middleware
// does, standing in for authMiddleware on /api/v1/caldav/*.
func withCalDAVAuth(userID, tenantID uuid.UUID) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), middleware.UserIDKey, userID.String())
			ctx = context.WithValue(ctx, middleware.TenantIDKey, tenantID.String())
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// passthroughCalDAVMiddleware stands in for middleware.Idempotency in the
// tests that are not about idempotency. The real chain is exercised in
// route_caldav_idempotency_db_test.go against a Postgres-backed repository.
func passthroughCalDAVMiddleware(next http.Handler) http.Handler { return next }

// noContentHandler stands in for the real caldav.Handler/carddav.Handler: it
// answers any request that makes it past basicAuthMiddleware with 204, the
// same status the real OPTIONS handler returns for the DAV root. This keeps
// the test focused on handleTestConnection/probeSelf/basicAuthMiddleware --
// the code this unit actually adds -- rather than re-verifying the
// third-party go-webdav library's OPTIONS handling.
var noContentHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
})

func TestHandleTestConnection_Success(t *testing.T) {
	userID, tenantID := uuid.New(), uuid.New()
	pwSvc := newFakeCalDAVPasswordService()

	routes := NewCalDAVRoutes(noContentHandler, noContentHandler, pwSvc, nil, noopCtxInjector, withCalDAVAuth(userID, tenantID), passthroughCalDAVMiddleware, "")

	r := chi.NewRouter()
	routes.RegisterRoutes(r)
	ts := httptest.NewServer(r)
	defer ts.Close()
	routes.selfBaseURL = ts.URL

	resp, err := http.Post(ts.URL+"/api/v1/caldav/test", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /api/v1/caldav/test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var result caldavTestResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if !result.Success || !result.CalDAVReachable || !result.CardDAVReachable {
		t.Fatalf("got %+v, want success with both protocols reachable", result)
	}
	if result.Message != "" {
		t.Fatalf("message = %q, want empty on success", result.Message)
	}

	if n := pwSvc.activePasswordCount(); n != 0 {
		t.Fatalf("active passwords after test = %d, want 0 (ephemeral password must be revoked)", n)
	}
}

func TestHandleTestConnection_AuthFailure(t *testing.T) {
	userID, tenantID := uuid.New(), uuid.New()
	pwSvc := newFakeCalDAVPasswordService()
	pwSvc.forceInvalid = true // Validate rejects every request, including the freshly created password

	routes := NewCalDAVRoutes(noContentHandler, noContentHandler, pwSvc, nil, noopCtxInjector, withCalDAVAuth(userID, tenantID), passthroughCalDAVMiddleware, "")

	r := chi.NewRouter()
	routes.RegisterRoutes(r)
	ts := httptest.NewServer(r)
	defer ts.Close()
	routes.selfBaseURL = ts.URL

	resp, err := http.Post(ts.URL+"/api/v1/caldav/test", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /api/v1/caldav/test: %v", err)
	}
	defer resp.Body.Close()

	var result caldavTestResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if result.Success || result.CalDAVReachable || result.CardDAVReachable {
		t.Fatalf("got %+v, want failure on both protocols", result)
	}
	if !strings.Contains(result.Message, "authentication failed") {
		t.Fatalf("message = %q, want it to name the auth failure", result.Message)
	}
}

// TestBasicAuthMiddleware_FailedLogin_DoesNotLogRawUsername belongs to
// fix-gateway-caldav-basic-auth-username-log-leakage: CalDAV clients
// conventionally send an email address as the Basic-Auth username (the
// expected value is a user UUID), and /caldav/ is reachable without prior
// authentication -- a failed attempt must not put that email in the log
// verbatim.
func TestBasicAuthMiddleware_FailedLogin_DoesNotLogRawUsername(t *testing.T) {
	userID, tenantID := uuid.New(), uuid.New()
	pwSvc := newFakeCalDAVPasswordService()
	pwSvc.forceInvalid = true

	routes := NewCalDAVRoutes(noContentHandler, noContentHandler, pwSvc, nil, noopCtxInjector, withCalDAVAuth(userID, tenantID), passthroughCalDAVMiddleware, "")

	r := chi.NewRouter()
	routes.RegisterRoutes(r)
	ts := httptest.NewServer(r)
	defer ts.Close()

	var logBuf bytes.Buffer
	prevLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logBuf, nil)))
	defer slog.SetDefault(prevLogger)

	const email = "leaks@example.com"
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/caldav/", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.SetBasicAuth(email, "wrong-password")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /caldav/: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}

	logged := logBuf.String()
	if strings.Contains(logged, email) {
		t.Fatalf("log output leaked the raw Basic-Auth username: %s", logged)
	}
	if !strings.Contains(logged, fingerprintCaldavUsername(email)) {
		t.Fatalf("log output missing fingerprint for anomaly/rate-limit analysis: %s", logged)
	}
}

func TestHandleTestConnection_NetworkUnreachable(t *testing.T) {
	userID, tenantID := uuid.New(), uuid.New()
	pwSvc := newFakeCalDAVPasswordService()

	// Reserve a loopback port and free it immediately: nothing listens there,
	// so a connection attempt fails deterministically with "connection refused"
	// instead of depending on an external unreachable host.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	deadAddr := l.Addr().String()
	l.Close()

	routes := NewCalDAVRoutes(noContentHandler, noContentHandler, pwSvc, nil, noopCtxInjector, withCalDAVAuth(userID, tenantID), passthroughCalDAVMiddleware, "")

	r := chi.NewRouter()
	routes.RegisterRoutes(r)
	ts := httptest.NewServer(r)
	defer ts.Close()
	routes.selfBaseURL = "http://" + deadAddr

	resp, err := http.Post(ts.URL+"/api/v1/caldav/test", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /api/v1/caldav/test: %v", err)
	}
	defer resp.Body.Close()

	var result caldavTestResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if result.Success {
		t.Fatalf("got %+v, want failure when the DAV port is unreachable", result)
	}
	if !strings.Contains(result.Message, "network unreachable") {
		t.Fatalf("message = %q, want it to name the network failure", result.Message)
	}

	// The ephemeral password must still be revoked even though both probes failed.
	if n := pwSvc.activePasswordCount(); n != 0 {
		t.Fatalf("active passwords after test = %d, want 0", n)
	}
}

// TestBasicAuthMiddleware_PassesInjectedTenantContextDownstream pins the root
// cause of the CardDAV/CalDAV tenant gap: the protocol routes authenticate via
// Basic Auth and never see a JWT, so unless basicAuthMiddleware hands the
// injector's context (user AND tenant) to the DAV handler, every gRPC call the
// backends make is rejected with codes.Unauthenticated and every direct pool
// query is filtered empty by RLS.
func TestBasicAuthMiddleware_PassesInjectedTenantContextDownstream(t *testing.T) {
	userID, tenantID := uuid.New(), uuid.New()
	pwSvc := newFakeCalDAVPasswordService()
	plaintext, _, err := pwSvc.Create(context.Background(), userID, tenantID, "davx5")
	if err != nil {
		t.Fatalf("create app password: %v", err)
	}

	var gotTenant, gotUser string
	probe := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if tid, tErr := middleware.GetTenantID(r.Context()); tErr == nil {
			gotTenant = tid.String()
		}
		gotUser = middleware.GetUserID(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})

	injector := func(ctx context.Context, uid uuid.UUID) (context.Context, error) {
		ctx = context.WithValue(ctx, middleware.TenantIDKey, tenantID.String())
		return context.WithValue(ctx, middleware.UserIDKey, uid.String()), nil
	}

	routes := NewCalDAVRoutes(probe, probe, pwSvc, nil, injector, withCalDAVAuth(userID, tenantID), passthroughCalDAVMiddleware, "")

	r := chi.NewRouter()
	routes.RegisterRoutes(r)
	ts := httptest.NewServer(r)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/carddav/", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.SetBasicAuth(userID.String(), plaintext)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /carddav/: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
	if gotTenant != tenantID.String() {
		t.Fatalf("tenant in handler context = %q, want %q", gotTenant, tenantID)
	}
	if gotUser != userID.String() {
		t.Fatalf("user in handler context = %q, want %q", gotUser, userID)
	}
}

// TestBasicAuthMiddleware_ContextInjectionFailure_Returns500 covers the other
// half: the credential was valid, so a failed tenant lookup is an
// infrastructure fault, not a rejected login. Answering 401 would be actively
// harmful -- DAV clients disable an account after repeated 401s, so a
// transient database outage would log every CalDAV user out for good.
func TestBasicAuthMiddleware_ContextInjectionFailure_Returns500(t *testing.T) {
	userID, tenantID := uuid.New(), uuid.New()
	pwSvc := newFakeCalDAVPasswordService()
	plaintext, _, err := pwSvc.Create(context.Background(), userID, tenantID, "davx5")
	if err != nil {
		t.Fatalf("create app password: %v", err)
	}

	failingInjector := func(context.Context, uuid.UUID) (context.Context, error) {
		return nil, errors.New("resolve tenant for caldav user: connection refused")
	}

	routes := NewCalDAVRoutes(noContentHandler, noContentHandler, pwSvc, nil, failingInjector, withCalDAVAuth(userID, tenantID), passthroughCalDAVMiddleware, "")

	r := chi.NewRouter()
	routes.RegisterRoutes(r)
	ts := httptest.NewServer(r)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/carddav/", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.SetBasicAuth(userID.String(), plaintext)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /carddav/: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", resp.StatusCode)
	}
	if resp.Header.Get("WWW-Authenticate") != "" {
		t.Fatalf("WWW-Authenticate set on an infrastructure failure: clients would drop their credentials")
	}
}

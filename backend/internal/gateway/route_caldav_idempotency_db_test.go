package gateway

// route_caldav_idempotency_db_test.go belongs to
// harden-caldav-rest-routes-missing-idempotency (BACKLOG.yml). Until this
// unit, cmd/gateway/main.go handed setupCalDAV the BARE authMiddleware while
// every other route registrar got authWithIdempotency
// (authMiddleware(idempotencyMW(next))). The five mutating routes under
// /api/v1/caldav and the two under /api/v1/admin/caldav therefore never saw
// middleware.Idempotency at all -- a double-submitted POST /passwords created
// TWO valid app-specific passwords, of which the caller sees only the second.
//
// These tests register the real RegisterRoutes wiring on a real chi router
// with a real Postgres-backed idempotency.Repository, so they fail if the
// idempotencyMW is ever dropped from the /api/v1 chain again. The password
// service stays a fake on purpose: the claim under test is about the
// middleware chain, not about caldav.AppPasswordService.

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/idempotency"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// caldavIdempotencyFixture mirrors cmd/gateway/main.go's wiring: the same
// middleware.Idempotency instance production builds at main.go:204, handed to
// NewCalDAVRoutes alongside the auth middleware and applied by RegisterRoutes.
type caldavIdempotencyFixture struct {
	pool      *pgxpool.Pool
	tenantID  uuid.UUID
	userID    uuid.UUID
	pwSvc     *fakeCalDAVPasswordService
	idempRepo idempotency.Repository
	router    chi.Router
}

func newCalDAVIdempotencyFixture(t *testing.T) *caldavIdempotencyFixture {
	t.Helper()
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "CalDAV Idempotency")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenantID) })

	userID := uuid.New()
	pwSvc := newFakeCalDAVPasswordService()
	idempRepo := idempotency.NewPostgresRepository(pool)

	routes := NewCalDAVRoutes(
		noContentHandler, noContentHandler, pwSvc, nil, noopCtxInjector,
		withCalDAVAuth(userID, tenantID),
		middleware.Idempotency(idempRepo, middleware.WarnMode),
		"",
	)
	r := chi.NewRouter()
	routes.RegisterRoutes(r)

	return &caldavIdempotencyFixture{
		pool:      pool,
		tenantID:  tenantID,
		userID:    userID,
		pwSvc:     pwSvc,
		idempRepo: idempRepo,
		router:    r,
	}
}

func (f *caldavIdempotencyFixture) createPassword(t *testing.T, key, label string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/caldav/passwords",
		bytes.NewReader([]byte(`{"label":"`+label+`"}`)))
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)
	return rec
}

// waitForCompletion polls until middleware.Idempotency's async Complete
// goroutine has stored the response. Without it a second request would race
// the first and see an in-flight reservation (409) instead of the replay.
func (f *caldavIdempotencyFixture) waitForCompletion(t *testing.T, key string) {
	t.Helper()
	ctx := testutil.WithTenantCtx(context.Background(), f.tenantID)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		rec, err := f.idempRepo.Get(ctx, f.tenantID, key)
		if err == nil && rec.CompletedAt != nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("idempotency key %q did not complete within deadline", key)
}

// decodeCalDAVBody parses a create-password response body into a map.
func decodeCalDAVBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response body %q: %v", rec.Body.String(), err)
	}
	return out
}

func TestCalDAVCreatePassword_SameIdempotencyKey_CreatesExactlyOnePassword(t *testing.T) {
	f := newCalDAVIdempotencyFixture(t)
	key := "caldav-pw-" + uuid.New().String()

	first := f.createPassword(t, key, "Thunderbird")
	assertStatus(t, first, http.StatusCreated)
	if got := first.Header().Get("Idempotency-Replayed"); got != "" {
		t.Fatalf("first request must not be marked replayed, got header %q", got)
	}

	f.waitForCompletion(t, key)

	second := f.createPassword(t, key, "Thunderbird")
	assertStatus(t, second, http.StatusCreated)
	if got := second.Header().Get("Idempotency-Replayed"); got != "true" {
		t.Fatalf("second request Idempotency-Replayed header = %q, want %q", got, "true")
	}
	// Compared field-wise, not byte-wise: the cached body makes a round trip
	// through a jsonb column, which re-serialises it (key order and spacing
	// differ from the handler's original output).
	if firstJSON, secondJSON := decodeCalDAVBody(t, first), decodeCalDAVBody(t, second); firstJSON["password"] != secondJSON["password"] || firstJSON["id"] != secondJSON["id"] {
		t.Fatalf("replay must return the cached password; first=%v second=%v", firstJSON, secondJSON)
	}
	if got := f.pwSvc.activePasswordCount(); got != 1 {
		t.Fatalf("app password count = %d, want 1 (a replayed POST must not mint a second password)", got)
	}
}

func TestCalDAVCreatePassword_DifferentIdempotencyKeys_CreateTwoPasswords(t *testing.T) {
	f := newCalDAVIdempotencyFixture(t)

	first := f.createPassword(t, "caldav-pw-a-"+uuid.New().String(), "Phone")
	assertStatus(t, first, http.StatusCreated)

	second := f.createPassword(t, "caldav-pw-b-"+uuid.New().String(), "Laptop")
	assertStatus(t, second, http.StatusCreated)
	if got := second.Header().Get("Idempotency-Replayed"); got != "" {
		t.Fatalf("a different key must not replay, got header %q", got)
	}
	if got := f.pwSvc.activePasswordCount(); got != 2 {
		t.Fatalf("app password count = %d, want 2 (different keys must not be deduplicated)", got)
	}
}

// TestCalDAVProtocolRoutes_StayFreeOfIdempotency proves the /caldav and
// /carddav protocol trees are untouched by this unit: a WebDAV client speaks
// Basic Auth and mutating verbs (PUT, DELETE) and knows nothing about an
// Idempotency-Key header. Both must still answer without one.
func TestCalDAVProtocolRoutes_StayFreeOfIdempotency(t *testing.T) {
	f := newCalDAVIdempotencyFixture(t)

	plaintext, _, err := f.pwSvc.Create(context.Background(), f.userID, f.tenantID, "WebDAV client")
	if err != nil {
		t.Fatalf("seed app password: %v", err)
	}

	for _, path := range []string{"/caldav/calendars/event.ics", "/carddav/contacts/card.vcf"} {
		req := httptest.NewRequest(http.MethodPut, path, bytes.NewReader([]byte("BEGIN:VCALENDAR\r\nEND:VCALENDAR\r\n")))
		req.SetBasicAuth(f.userID.String(), plaintext)
		rec := httptest.NewRecorder()
		f.router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("PUT %s without Idempotency-Key = %d, want %d (protocol routes must stay key-free)",
				path, rec.Code, http.StatusNoContent)
		}
	}
}

package middleware

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/idempotency"
)

// --- In-memory mock repo ---

type memIdempotencyRepo struct {
	records map[string]*idempotency.Record
	// injectErr forces Reserve to return this error on the next call.
	injectErr error
}

func newMemIdempotencyRepo() *memIdempotencyRepo {
	return &memIdempotencyRepo{records: make(map[string]*idempotency.Record)}
}

func (m *memIdempotencyRepo) Reserve(
	ctx context.Context,
	key string,
	tenantID, userID uuid.UUID,
	method, path, hash string,
) (*idempotency.Record, error) {
	if m.injectErr != nil {
		err := m.injectErr
		m.injectErr = nil
		return nil, err
	}

	existing, ok := m.records[key]
	if !ok {
		now := time.Now()
		m.records[key] = &idempotency.Record{
			Key:         key,
			TenantID:    tenantID,
			UserID:      userID,
			Method:      method,
			Path:        path,
			RequestHash: hash,
			CreatedAt:   now,
			ExpiresAt:   now.Add(24 * time.Hour),
		}
		return nil, nil
	}

	if existing.RequestHash != hash {
		return nil, idempotency.ErrConflict
	}
	if existing.CompletedAt == nil {
		return nil, idempotency.ErrInFlight
	}
	return existing, nil
}

func (m *memIdempotencyRepo) Get(ctx context.Context, tenantID uuid.UUID, key string) (*idempotency.Record, error) {
	rec, ok := m.records[key]
	if !ok {
		return nil, pgx.ErrNoRows
	}
	return rec, nil
}

func (m *memIdempotencyRepo) Complete(ctx context.Context, tenantID uuid.UUID, key string, status int, body []byte) error {
	rec, ok := m.records[key]
	if !ok {
		return errors.New("key not found")
	}
	// Enforce tenant isolation in mock: refuse to complete for wrong tenant.
	if rec.TenantID != tenantID {
		return idempotency.ErrKeyMissing
	}
	rec.ResponseStatus = &status
	rec.ResponseBody = body
	now := time.Now()
	rec.CompletedAt = &now
	return nil
}

func (m *memIdempotencyRepo) Cleanup(ctx context.Context) (int, error) {
	return 0, nil
}

func (m *memIdempotencyRepo) CleanupWithLock(ctx context.Context, lockKey int64) (int, error) {
	return 0, nil
}

// --- Helpers ---

// requestWithAuth builds an *http.Request with tenant/user context pre-set.
func requestWithAuth(method, path string, body string) *http.Request {
	var bodyReader *strings.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	} else {
		bodyReader = strings.NewReader("")
	}
	r := httptest.NewRequest(method, path, bodyReader)
	tenantID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	userID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	ctx := context.WithValue(r.Context(), TenantIDKey, tenantID.String())
	ctx = context.WithValue(ctx, UserIDKey, userID.String())
	return r.WithContext(ctx)
}

func okHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write([]byte(`{"id":"new-resource"}`))
}

// --- Tests ---

func TestIdempotency_Skip_GETRequest(t *testing.T) {
	repo := newMemIdempotencyRepo()
	mw := Idempotency(repo, WarnMode)

	r := requestWithAuth(http.MethodGet, "/api/v1/crm/contacts", "")
	rec := httptest.NewRecorder()
	mw(http.HandlerFunc(okHandler)).ServeHTTP(rec, r)

	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Empty(t, repo.records, "GET should not create idempotency records")
}

func TestIdempotency_FreshKey_StoresAndReturns(t *testing.T) {
	repo := newMemIdempotencyRepo()
	mw := Idempotency(repo, WarnMode)

	r := requestWithAuth(http.MethodPost, "/api/v1/crm/contacts", `{"name":"Test"}`)
	r.Header.Set("Idempotency-Key", "key-fresh-001")
	rec := httptest.NewRecorder()
	mw(http.HandlerFunc(okHandler)).ServeHTTP(rec, r)

	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, `{"id":"new-resource"}`, rec.Body.String())
}

func TestIdempotency_Replay_ReturnsCached(t *testing.T) {
	repo := newMemIdempotencyRepo()
	mw := Idempotency(repo, WarnMode)

	// First call — handler runs
	r1 := requestWithAuth(http.MethodPost, "/api/v1/crm/contacts", `{"name":"Test"}`)
	r1.Header.Set("Idempotency-Key", "key-replay-001")
	rec1 := httptest.NewRecorder()
	mw(http.HandlerFunc(okHandler)).ServeHTTP(rec1, r1)
	require.Equal(t, http.StatusCreated, rec1.Code)

	// Wait briefly for async Complete goroutine
	time.Sleep(20 * time.Millisecond)

	// Second call — should be replayed from cache
	r2 := requestWithAuth(http.MethodPost, "/api/v1/crm/contacts", `{"name":"Test"}`)
	r2.Header.Set("Idempotency-Key", "key-replay-001")
	rec2 := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called on replay")
	})).ServeHTTP(rec2, r2)

	assert.Equal(t, http.StatusCreated, rec2.Code)
	assert.Equal(t, "true", rec2.Header().Get("Idempotency-Replayed"))
	assert.Equal(t, `{"id":"new-resource"}`, rec2.Body.String())
}

func TestIdempotency_InFlight_Returns409(t *testing.T) {
	repo := newMemIdempotencyRepo()
	// Pre-populate a non-completed record (simulates in-flight)
	repo.records["key-inflight-001"] = &idempotency.Record{
		Key:         "key-inflight-001",
		TenantID:    uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		UserID:      uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		Method:      "POST",
		Path:        "/api/v1/crm/contacts",
		RequestHash: computeRequestHash("11111111-1111-1111-1111-111111111111", "22222222-2222-2222-2222-222222222222", "POST", "/api/v1/crm/contacts", []byte(`{"name":"Test"}`)),
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	}

	mw := Idempotency(repo, WarnMode)
	r := requestWithAuth(http.MethodPost, "/api/v1/crm/contacts", `{"name":"Test"}`)
	r.Header.Set("Idempotency-Key", "key-inflight-001")
	rec := httptest.NewRecorder()
	mw(http.HandlerFunc(okHandler)).ServeHTTP(rec, r)

	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.Equal(t, "2", rec.Header().Get("Retry-After"))
}

func TestIdempotency_Conflict_Returns422(t *testing.T) {
	repo := newMemIdempotencyRepo()
	// Pre-populate with a different hash
	status := 201
	body := []byte(`{"id":"old"}`)
	now := time.Now()
	repo.records["key-conflict-001"] = &idempotency.Record{
		Key:            "key-conflict-001",
		TenantID:       uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		UserID:         uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		Method:         "POST",
		Path:           "/api/v1/crm/contacts",
		RequestHash:    "different-hash-from-first-request",
		ResponseStatus: &status,
		ResponseBody:   body,
		CreatedAt:      now,
		CompletedAt:    &now,
		ExpiresAt:      now.Add(24 * time.Hour),
	}

	mw := Idempotency(repo, WarnMode)
	r := requestWithAuth(http.MethodPost, "/api/v1/crm/contacts", `{"name":"DifferentPayload"}`)
	r.Header.Set("Idempotency-Key", "key-conflict-001")
	rec := httptest.NewRecorder()
	mw(http.HandlerFunc(okHandler)).ServeHTTP(rec, r)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestIdempotency_MissingKey_WarnMode_Passes(t *testing.T) {
	repo := newMemIdempotencyRepo()
	mw := Idempotency(repo, WarnMode)

	r := requestWithAuth(http.MethodPost, "/api/v1/crm/contacts", `{"name":"Test"}`)
	// No Idempotency-Key header
	rec := httptest.NewRecorder()
	mw(http.HandlerFunc(okHandler)).ServeHTTP(rec, r)

	assert.Equal(t, http.StatusCreated, rec.Code, "WarnMode should pass through missing key")
}

func TestIdempotency_MissingKey_HardMode_Blocks(t *testing.T) {
	repo := newMemIdempotencyRepo()
	mw := Idempotency(repo, HardMode)

	r := requestWithAuth(http.MethodPost, "/api/v1/crm/contacts", `{"name":"Test"}`)
	rec := httptest.NewRecorder()
	mw(http.HandlerFunc(okHandler)).ServeHTTP(rec, r)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// TestHardMode_MissingKey_Returns400 verifies that HardMode rejects mutation
// requests without an Idempotency-Key header with 400 Bad Request.
// Note: TestIdempotency_MissingKey_HardMode_Blocks covers the same path;
// this alias documents the HardMode contract explicitly for regression clarity.
func TestHardMode_MissingKey_Returns400(t *testing.T) {
	repo := newMemIdempotencyRepo()
	mw := Idempotency(repo, HardMode)

	r := requestWithAuth(http.MethodPost, "/api/v1/crm/contacts", `{"name":"Test"}`)
	// Intentionally no Idempotency-Key header.
	rec := httptest.NewRecorder()
	mw(http.HandlerFunc(okHandler)).ServeHTTP(rec, r)

	assert.Equal(t, http.StatusBadRequest, rec.Code,
		"HardMode must return 400 when Idempotency-Key header is absent")
}

// TestHardMode_CrossTenantKeyRejected verifies that a completed idempotency record
// belonging to Tenant A is not replayed for Tenant B when they share the same key string.
// This tests the composite-PK cross-tenant isolation at the middleware level.
func TestHardMode_CrossTenantKeyRejected(t *testing.T) {
	repo := newMemIdempotencyRepo()
	mw := Idempotency(repo, HardMode)

	tenantAID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	tenantBID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	userID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	const sharedKey = "cross-tenant-key-001"

	// Helper that builds a request carrying a specific tenantID in context.
	buildRequest := func(tenantID uuid.UUID) *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/api/v1/crm/contacts", strings.NewReader(`{"name":"Test"}`))
		r.Header.Set("Idempotency-Key", sharedKey)
		ctx := context.WithValue(r.Context(), TenantIDKey, tenantID.String())
		ctx = context.WithValue(ctx, UserIDKey, userID.String())
		return r.WithContext(ctx)
	}

	// Tenant A sends request — handler runs, record reserved.
	rec1 := httptest.NewRecorder()
	mw(http.HandlerFunc(okHandler)).ServeHTTP(rec1, buildRequest(tenantAID))
	require.Equal(t, http.StatusCreated, rec1.Code)

	// Wait briefly for async Complete goroutine to finish.
	time.Sleep(20 * time.Millisecond)

	// Validate Tenant A's record is present under the correct tenant.
	storedRec, ok := repo.records[sharedKey]
	require.True(t, ok, "record must be stored after Tenant A's request")
	require.Equal(t, tenantAID, storedRec.TenantID, "stored record must belong to Tenant A")

	// Tenant B sends request with the same Idempotency-Key string.
	// Expected: middleware calls Reserve → key found but tenantID differs → ErrConflict OR
	// the repo mock scopes lookup by tenantID so B gets a fresh insert (no cross-tenant replay).
	// Either way Tenant B's response must NOT be Tenant A's cached body.
	rec2 := httptest.NewRecorder()
	handlerCalledByB := false
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalledByB = true
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"tenant-b-resource"}`))
	})).ServeHTTP(rec2, buildRequest(tenantBID))

	// The mock's Reserve uses records[key] without tenant scoping — so it will detect
	// a hash conflict (same hash, different tenantID embedded in hash) and return ErrConflict,
	// OR proceed fresh if hashes differ. Either way Tenant B must NOT get Tenant A's response.
	assert.NotEqual(t, `{"id":"new-resource"}`, rec2.Body.String(),
		"Tenant B must never receive Tenant A's cached idempotency response")
	// In any non-replay path the status code must not be the replayed 201 with Replayed header.
	assert.Empty(t, rec2.Header().Get("Idempotency-Replayed"),
		"Idempotency-Replayed header must not be set for Tenant B (no cross-tenant replay)")
	_ = handlerCalledByB // Both 422 (conflict) and fresh handler call are acceptable outcomes.
}

func TestIdempotency_WhitelistedPath_Skip(t *testing.T) {
	repo := newMemIdempotencyRepo()
	mw := Idempotency(repo, HardMode)

	// /auth/login is whitelisted — no Idempotency-Key required even in HardMode
	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader([]byte(`{"email":"x@x.com","password":"pw"}`)))
	rec := httptest.NewRecorder()
	mw(http.HandlerFunc(okHandler)).ServeHTTP(rec, r)

	assert.Equal(t, http.StatusCreated, rec.Code)
}

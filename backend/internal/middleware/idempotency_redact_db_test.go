package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/idempotency"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

const redactTestSecret = "s3cret-one-time-token"

// secretHandler answers 201 with a one-time secret. explicitHeader=false takes
// the implicit-200-path of Write (no WriteHeader call) — the redact marker must
// be consumed there too.
func secretHandler(redact, explicitHeader bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if redact {
			middleware.RedactIdempotentResponse(w)
		}
		if explicitHeader {
			w.WriteHeader(http.StatusCreated)
		}
		_, _ = w.Write([]byte(`{"password":"` + redactTestSecret + `"}`))
	}
}

func storedResponse(t *testing.T, tenantID uuid.UUID, key string) (status int, body string) {
	t.Helper()
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)
	ctx := testutil.WithSystemCtx(context.Background())
	require.Eventually(t, func() bool {
		var st *int
		var b []byte
		err := pool.QueryRow(ctx,
			"SELECT response_status, response_body FROM idempotency_keys WHERE tenant_id=$1 AND key=$2",
			tenantID, key).Scan(&st, &b)
		if err != nil || st == nil {
			return false
		}
		status, body = *st, string(b)
		return true
	}, 3*time.Second, 10*time.Millisecond, "response must be completed in idempotency_keys")
	return status, body
}

func secretRequest(tenantID, userID uuid.UUID, key, payload string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/caldav/passwords", strings.NewReader(payload))
	r.Header.Set("Idempotency-Key", key)
	ctx := testutil.WithTenantCtx(r.Context(), tenantID)
	ctx = context.WithValue(ctx, middleware.UserIDKey, userID.String())
	return r.WithContext(ctx)
}

func TestIdempotency_RedactedResponse_NotStoredButDeduplicated(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Idempotency Redact")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	userID := uuid.New()
	mw := middleware.Idempotency(idempotency.NewPostgresRepository(pool), middleware.WarnMode)

	for name, explicitHeader := range map[string]bool{"explicit WriteHeader": true, "implicit header via Write": false} {
		t.Run(name, func(t *testing.T) {
			key := "redact-" + uuid.New().String()
			r := secretRequest(tenantID, userID, key, `{"label":"iPhone"}`)
			rec := httptest.NewRecorder()
			mw(secretHandler(true, explicitHeader)).ServeHTTP(rec, r)

			// The client gets the secret and never sees the internal marker.
			assert.Contains(t, rec.Body.String(), redactTestSecret)
			assert.Empty(t, rec.Header().Get(middleware.RedactIdempotencyHeader))
			wantStatus := http.StatusCreated
			if !explicitHeader {
				wantStatus = http.StatusOK
			}
			assert.Equal(t, wantStatus, rec.Code)

			// The store holds status, but not the secret.
			status, body := storedResponse(t, tenantID, key)
			assert.Equal(t, wantStatus, status)
			assert.NotContains(t, body, redactTestSecret)

			// Replay: same key + body → handler is skipped, no second secret.
			rec2 := httptest.NewRecorder()
			mw(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				t.Fatal("handler must not run on replay")
			})).ServeHTTP(rec2, secretRequest(tenantID, userID, key, `{"label":"iPhone"}`))
			assert.Equal(t, wantStatus, rec2.Code)
			assert.Equal(t, "true", rec2.Header().Get("Idempotency-Replayed"))
			assert.NotContains(t, rec2.Body.String(), redactTestSecret)

			// Same key, other body stays a payload conflict.
			rec3 := httptest.NewRecorder()
			mw(secretHandler(true, explicitHeader)).ServeHTTP(rec3, secretRequest(tenantID, userID, key, `{"label":"Pixel"}`))
			assert.Equal(t, http.StatusUnprocessableEntity, rec3.Code)
		})
	}
}

// Control: without the marker the body is stored as before — proves the
// assertions above look at the real column and not at an always-empty one.
func TestIdempotency_UnmarkedResponse_StillStoredVerbatim(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()
	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Idempotency Unmarked")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	mw := middleware.Idempotency(idempotency.NewPostgresRepository(pool), middleware.WarnMode)
	key := "plain-" + uuid.New().String()
	rec := httptest.NewRecorder()
	mw(secretHandler(false, true)).ServeHTTP(rec, secretRequest(tenantID, uuid.New(), key, `{"label":"iPhone"}`))
	require.Equal(t, http.StatusCreated, rec.Code)

	_, body := storedResponse(t, tenantID, key)
	assert.Contains(t, body, redactTestSecret)
}

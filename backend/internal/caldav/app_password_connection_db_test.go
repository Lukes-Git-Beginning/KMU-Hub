package caldav

// DB-backed proof for fix-caldav-test-revoke-runs-without-tenant-context:
// POST /api/v1/caldav/test mints an ephemeral app password and revokes it in a
// defer. app_specific_passwords is under RLS, so a revoke on a context without
// the tenant matches zero rows and leaves a permanently valid credential
// behind. The fake password service in the gateway tests cannot see that --
// this one runs the real AppPasswordService against Postgres as kmuhub_app.
// It lives in package caldav because gateway's own tests cannot import caldav
// (caldav already imports gateway).

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/gateway"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// gatewayPasswordAdapter mirrors cmd/gateway's caldavPasswordAdapter, reduced
// to the methods handleTestConnection and its Basic-Auth probes touch.
type gatewayPasswordAdapter struct {
	*AppPasswordService
}

func (a gatewayPasswordAdapter) Create(ctx context.Context, userID, tenantID uuid.UUID, label string) (string, *gateway.CalDAVPasswordInfo, error) {
	plaintext, pw, err := a.AppPasswordService.Create(ctx, userID, tenantID, label)
	if err != nil {
		return "", nil, err
	}
	return plaintext, &gateway.CalDAVPasswordInfo{ID: pw.ID, Label: pw.Label, PasswordPrefix: pw.PasswordPrefix}, nil
}

func (a gatewayPasswordAdapter) List(ctx context.Context, userID uuid.UUID) ([]*gateway.CalDAVPasswordInfo, error) {
	pws, err := a.AppPasswordService.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]*gateway.CalDAVPasswordInfo, len(pws))
	for i, pw := range pws {
		out[i] = &gateway.CalDAVPasswordInfo{ID: pw.ID, Label: pw.Label, RevokedAt: pw.RevokedAt}
	}
	return out, nil
}

func TestHandleTestConnection_RevokesEphemeralPasswordUnderRLS(t *testing.T) {
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	svc := NewAppPasswordService(NewPostgresAppPasswordRepository(pool), pool)
	withOrgEnabledRestored(t, svc)
	require.NoError(t, svc.SetOrgEnabled(context.Background(), true))

	ctx, tenantID, userID := seedCalDAVUser(t, pool)

	davOK := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	auth := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c := context.WithValue(r.Context(), middleware.UserIDKey, userID.String())
			c = context.WithValue(c, middleware.TenantIDKey, tenantID.String())
			next.ServeHTTP(w, r.WithContext(c))
		})
	}
	passthrough := func(next http.Handler) http.Handler { return next }
	noopInjector := func(ctx context.Context, _ uuid.UUID) (context.Context, error) { return ctx, nil }

	// The probes call back into this same server, so its address has to be
	// known before the routes are built: bind first, serve after.
	ts := httptest.NewUnstartedServer(nil)
	defer ts.Close()
	routes := gateway.NewCalDAVRoutes(davOK, davOK, gatewayPasswordAdapter{svc}, nil, noopInjector, auth, passthrough,
		"http://"+ts.Listener.Addr().String())
	r := chi.NewRouter()
	routes.RegisterRoutes(r)
	ts.Config.Handler = r
	ts.Start()

	resp, err := http.Post(ts.URL+"/api/v1/caldav/test", "application/json", nil)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Success bool `json:"success"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	require.True(t, result.Success, "probes must pass, otherwise the revoke path proves nothing")

	pws, err := svc.List(ctx, userID)
	require.NoError(t, err)
	require.Len(t, pws, 1, "the test run mints exactly one ephemeral password")
	assert.NotNil(t, pws[0].RevokedAt, "ephemeral password must be revoked after the connection test")

	active, err := NewPostgresAppPasswordRepository(pool).FindActiveByUser(ctx, userID)
	require.NoError(t, err)
	assert.Empty(t, active, "no active app password may survive a connection test")
}

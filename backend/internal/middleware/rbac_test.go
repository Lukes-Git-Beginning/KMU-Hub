package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/kmuhub/kmuhub/internal/auth"
)

func withUserContext(r *http.Request, userID string, roles, perms []string) *http.Request {
	ctx := r.Context()
	ctx = context.WithValue(ctx, UserIDKey, userID)
	ctx = context.WithValue(ctx, UserRolesKey, roles)
	ctx = context.WithValue(ctx, UserPermsKey, perms)
	return r.WithContext(ctx)
}

func TestRequireRole(t *testing.T) {
	handler := RequireRole("admin", "manager")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name       string
		roles      []string
		wantStatus int
	}{
		{"admin allowed", []string{"admin"}, http.StatusOK},
		{"manager allowed", []string{"manager"}, http.StatusOK},
		{"member denied", []string{"member"}, http.StatusForbidden},
		{"no roles denied", nil, http.StatusForbidden},
		{"multiple roles with one match", []string{"member", "admin"}, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req = withUserContext(req, "user-123", tt.roles, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)
			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

// TestRequirePermissionAny covers the compatibility guarantee the helper
// exists for: a token minted before a guard was tightened (coarse key only)
// must still pass, and so must a freshly minted one (fine-grained key).
func TestRequirePermissionAny(t *testing.T) {
	handler := RequirePermissionAny(
		[2]string{"files", "write"},
		[2]string{"documents:file", "upload"},
	)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name       string
		perms      []string
		wantStatus int
	}{
		{"legacy token, coarse key only", []string{"files:write"}, http.StatusOK},
		{"fresh token, fine-grained key only", []string{"documents:file:upload"}, http.StatusOK},
		{"both keys", []string{"files:write", "documents:file:upload"}, http.StatusOK},
		{"neither key", []string{"files:read", "documents:file:read"}, http.StatusForbidden},
		{"no permissions", nil, http.StatusForbidden},
		{"resource matches but action does not", []string{"documents:file:delete"}, http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req = withUserContext(req, "user-123", nil, tt.perms)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)
			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

func TestRequirePermission(t *testing.T) {
	handler := RequirePermission("contacts", "write")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name       string
		perms      []string
		wantStatus int
	}{
		{"has permission", []string{"contacts:read", "contacts:write"}, http.StatusOK},
		{"missing permission", []string{"contacts:read"}, http.StatusForbidden},
		{"no permissions", nil, http.StatusForbidden},
		{"wrong resource", []string{"deals:write"}, http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req = withUserContext(req, "user-123", nil, tt.perms)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)
			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

// withDenied is withUserContext plus the deny list a per-user override puts in
// the token.
func withDenied(r *http.Request, perms, denied []string) *http.Request {
	r = withUserContext(r, "user-123", nil, perms)
	return r.WithContext(context.WithValue(r.Context(), UserDeniedKey, denied))
}

// The case the deny claim exists for: an administrator denies the fine-grained
// capability, but the account still holds the coarse legacy key its role
// grants. Without the claim the coarse key would keep the door open — in 154 of
// the 164 call sites of this guard.
func TestRequirePermissionAny_DeniedFineKeyClosesTheCoarseStandIn(t *testing.T) {
	handler := RequirePermissionAny(
		[2]string{"files", "write"},
		[2]string{"documents:file", "upload"},
	)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name       string
		perms      []string
		denied     []string
		wantStatus int
	}{
		{
			name:       "coarse key alone passes while nothing is denied",
			perms:      []string{"files:write"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "coarse key stops standing in for the denied fine key",
			perms:      []string{"files:write"},
			denied:     []string{"documents:file:upload"},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "a deny on an unrelated key leaves this guard alone",
			perms:      []string{"files:write"},
			denied:     []string{"crm:contact:delete"},
			wantStatus: http.StatusOK,
		},
		{
			// The allow set already dropped the fine key; this only proves the
			// coarse one does not quietly take over.
			name:       "both keys held, fine one denied",
			perms:      []string{"files:write"},
			denied:     []string{"documents:file:upload"},
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := withDenied(httptest.NewRequest(http.MethodGet, "/", nil), tt.perms, tt.denied)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

// Two fine keys in one guard are two different rights that happen to open the
// same route. Denying one must not take the other away — the coarse-key rule
// does not generalise to them.
func TestRequirePermissionAny_FineKeysAreJudgedIndividually(t *testing.T) {
	handler := RequirePermissionAny(
		[2]string{"berichte:reports", "read"},
		[2]string{"berichte:export", "run"},
	)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := withDenied(httptest.NewRequest(http.MethodGet, "/", nil),
		[]string{"berichte:reports:read"}, []string{"berichte:export:run"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	// …and denying the one the account actually holds does close the door.
	req = withDenied(httptest.NewRequest(http.MethodGet, "/", nil),
		nil, []string{"berichte:reports:read"})
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// A token minted before the deny claim existed carries no list at all. It has
// to behave exactly as it did — otherwise every still-valid session would
// change behaviour the moment this shipped.
func TestRequirePermissionAny_TokenWithoutDenyClaimIsUnchanged(t *testing.T) {
	handler := RequirePermissionAny(
		[2]string{"files", "write"},
		[2]string{"documents:file", "upload"},
	)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := withUserContext(httptest.NewRequest(http.MethodGet, "/", nil),
		"user-123", nil, []string{"files:write"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// gateFromToken runs one request through Auth and the guard, using a really
// signed token rather than a hand-built context — the deny claim has to survive
// the JWT round trip to be worth anything.
func gateFromToken(t *testing.T, perms *auth.TokenPermissions) int {
	t.Helper()

	tm := auth.NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour)
	token, err := tm.CreateAccessToken(uuid.New(), uuid.New().String(), []string{"member"}, perms)
	if err != nil {
		t.Fatalf("CreateAccessToken: %v", err)
	}

	guard := RequirePermissionAny(
		[2]string{"files", "write"},
		[2]string{"documents:file", "upload"},
	)
	handler := Auth(auth.NewService(nil, tm))(guard(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec.Code
}

// End to end: what Service.ResolveTokenPermissions produces for a denied and an
// allowed key closes and opens this gate through a signed token.
func TestRequirePermissionAny_DenyClaimSurvivesTheTokenRoundTrip(t *testing.T) {
	// What the role union alone yields — the coarse key opens the door.
	assert.Equal(t, http.StatusOK, gateFromToken(t, &auth.TokenPermissions{
		Permissions: []string{"files:write", "contacts:read"},
	}))

	// The same account after an administrator denied the fine capability. The
	// key is gone from the allow set AND named in the deny list, which is what
	// stops "files:write" from standing in for it.
	assert.Equal(t, http.StatusForbidden, gateFromToken(t, &auth.TokenPermissions{
		Permissions: []string{"files:write", "contacts:read"},
		Denied:      []string{"documents:file:upload"},
		Scopes:      map[string]string{"documents:file:upload": auth.ScopeOwn},
	}))

	// An allow override opens the gate for an account whose roles grant neither
	// key.
	assert.Equal(t, http.StatusForbidden, gateFromToken(t, &auth.TokenPermissions{
		Permissions: []string{"contacts:read"},
	}))
	assert.Equal(t, http.StatusOK, gateFromToken(t, &auth.TokenPermissions{
		Permissions: []string{"contacts:read", "documents:file:upload"},
	}))
}

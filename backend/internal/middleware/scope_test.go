package middleware

// PermissionScope decides how many rows an endpoint may return, so the two
// failure directions are asymmetric and both bad: reading "all" where "own" was
// granted leaks the tenant's data, reading "own" where nothing was granted
// empties the lists of every user still holding a token minted before the
// scopes claim existed. These tests pin both, end to end through a real signed
// token rather than a hand-built context.

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/kmuhub/kmuhub/internal/auth"
)

// scopeFromToken runs one request through Auth with the given scopes claim and
// reports what PermissionScope answers inside the handler.
func scopeFromToken(t *testing.T, scopes map[string]string, resource, action string) string {
	t.Helper()

	tm := newTestTokenMaker()
	token, err := tm.CreateAccessToken(uuid.New(), uuid.New().String(),
		[]string{"member"}, []string{resource + ":" + action}, scopes)
	if err != nil {
		t.Fatalf("CreateAccessToken: %v", err)
	}

	var got string
	handler := Auth(newTestAuthService())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = PermissionScope(r.Context(), resource, action)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Auth rejected the request: %d", rec.Code)
	}
	return got
}

func TestPermissionScope(t *testing.T) {
	tests := []struct {
		name   string
		scopes map[string]string
		want   string
	}{
		{
			name:   "narrowed key reports its scope",
			scopes: map[string]string{"rapporte:report:read": auth.ScopeOwn},
			want:   auth.ScopeOwn,
		},
		{
			name:   "team travels the same way",
			scopes: map[string]string{"rapporte:report:read": auth.ScopeTeam},
			want:   auth.ScopeTeam,
		},
		{
			// A key the user holds but that nobody narrowed is absent from the
			// map — that is what keeps the claim small, and it must not read as
			// a restriction.
			name:   "key absent from a populated map reaches all",
			scopes: map[string]string{"helpdesk:ticket:read": auth.ScopeOwn},
			want:   auth.ScopeAll,
		},
		{
			// A token minted before the claim existed. Reading "own" here would
			// shrink every list for every still-valid session.
			name:   "legacy token without the claim reaches all",
			scopes: nil,
			want:   auth.ScopeAll,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, scopeFromToken(t, tc.scopes, "rapporte:report", "read"))
		})
	}
}

// Without the Auth middleware there is no scopes value in the context at all —
// the type assertion has to miss cleanly instead of panicking.
func TestPermissionScope_WithoutAuthMiddleware(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	assert.Equal(t, auth.ScopeAll, PermissionScope(req.Context(), "rapporte:report", "read"))
}

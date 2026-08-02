package gateway

// ownerFilterForScope is the single place where a resolved data scope turns
// into a list filter. Every list endpoint that honours "own" goes through it,
// so a wrong answer here is a leak in all of them at once.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/middleware"
)

func requestWithScope(userID string, scopes map[string]string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/rapporte/reports?author_id="+uuid.NewString(), nil)
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	ctx = context.WithValue(ctx, middleware.UserScopesKey, scopes)
	return req.WithContext(ctx)
}

func TestOwnerFilterForScope_OwnReturnsCaller(t *testing.T) {
	userID := uuid.NewString()
	rec := httptest.NewRecorder()

	got, ok := ownerFilterForScope(rec, requestWithScope(userID, map[string]string{
		"rapporte:report:read": auth.ScopeOwn,
	}), "rapporte:report", "read")

	require.True(t, ok)
	require.NotNil(t, got)
	// The caller's own id, never the author_id they asked for in the query.
	assert.Equal(t, userID, *got)
}

func TestOwnerFilterForScope_WiderScopesDoNotFilter(t *testing.T) {
	for name, scopes := range map[string]map[string]string{
		"all":                 {"rapporte:report:read": auth.ScopeAll},
		"team behaves as all": {"rapporte:report:read": auth.ScopeTeam},
		"key not narrowed":    {"helpdesk:ticket:read": auth.ScopeOwn},
		"legacy token":        nil,
	} {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			got, ok := ownerFilterForScope(rec, requestWithScope(uuid.NewString(), scopes), "rapporte:report", "read")

			require.True(t, ok)
			assert.Nil(t, got, "a scope wider than own must not narrow the list")
		})
	}
}

// At scope "own" without a user id there is no honest answer: returning nil
// would hand back the unfiltered list, and an empty id would reach the service
// as an unparsable filter.
func TestOwnerFilterForScope_OwnWithoutUserIsRejected(t *testing.T) {
	rec := httptest.NewRecorder()

	got, ok := ownerFilterForScope(rec, requestWithScope("", map[string]string{
		"rapporte:report:read": auth.ScopeOwn,
	}), "rapporte:report", "read")

	assert.False(t, ok)
	assert.Nil(t, got)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

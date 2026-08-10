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

// ownerFilterForScopeAny backs routes guarded by middleware.RequirePermissionAny
// across a legacy coarse key ("tasks", "read") and a fine key
// ("work:task", "read") — see route_work_tasks.go HandleListTasks. The guard
// grants access if EITHER key is present, so a caller narrowed under one key
// must not escape that narrowing just because the other key was never
// assigned to them (an unassigned key resolves to auth.ScopeAll, not
// "unknown" — see middleware.PermissionScope).

var taskPermissionPairs = [][2]string{{"tasks", "read"}, {"work:task", "read"}}

func TestOwnerFilterForScopeAny_OwnOnFirstKeyNarrows(t *testing.T) {
	userID := uuid.NewString()
	rec := httptest.NewRecorder()

	got, ok := ownerFilterForScopeAny(rec, requestWithScope(userID, map[string]string{
		"tasks:read": auth.ScopeOwn,
		// "work:task:read" intentionally absent -> resolves to ScopeAll alone.
	}), taskPermissionPairs...)

	require.True(t, ok)
	require.NotNil(t, got)
	assert.Equal(t, userID, *got)
}

func TestOwnerFilterForScopeAny_OwnOnSecondKeyNarrows(t *testing.T) {
	userID := uuid.NewString()
	rec := httptest.NewRecorder()

	got, ok := ownerFilterForScopeAny(rec, requestWithScope(userID, map[string]string{
		"work:task:read": auth.ScopeOwn,
		// "tasks:read" intentionally absent -> resolves to ScopeAll alone.
	}), taskPermissionPairs...)

	require.True(t, ok)
	require.NotNil(t, got)
	assert.Equal(t, userID, *got)
}

func TestOwnerFilterForScopeAny_WiderScopesDoNotFilter(t *testing.T) {
	for name, scopes := range map[string]map[string]string{
		"both all":     {"tasks:read": auth.ScopeAll, "work:task:read": auth.ScopeAll},
		"both absent":  nil,
		"team on fine": {"work:task:read": auth.ScopeTeam},
	} {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			got, ok := ownerFilterForScopeAny(rec, requestWithScope(uuid.NewString(), scopes), taskPermissionPairs...)

			require.True(t, ok)
			assert.Nil(t, got, "no pair at own must not narrow the list")
		})
	}
}

func TestOwnerFilterForScopeAny_OwnWithoutUserIsRejected(t *testing.T) {
	rec := httptest.NewRecorder()

	got, ok := ownerFilterForScopeAny(rec, requestWithScope("", map[string]string{
		"work:task:read": auth.ScopeOwn,
	}), taskPermissionPairs...)

	assert.False(t, ok)
	assert.Nil(t, got)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

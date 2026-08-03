package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tokenPermsFixture seeds one account with the two permission worlds the token
// carries side by side: coarse legacy keys and fine catalogue keys.
func tokenPermsFixture(t *testing.T) (*mockRepository, *Service, uuid.UUID) {
	t.Helper()
	repo := newMockRepository()
	svc := NewService(repo, nil)

	userID := uuid.New()
	role := uuid.New()
	repo.userPerms[userID] = []string{
		"files:write", "contacts:read", // coarse, what most guards read
		"documents:file:upload", "crm:contact:read", // fine, what the catalogue knows
	}
	repo.effectiveGrants[userID] = []EffectiveGrantRow{
		grantRow(role, "member", "documents:file:upload", "all"),
		grantRow(role, "member", "crm:contact:read", "own"),
	}
	return repo, svc, userID
}

// The claims of an account without overrides must be byte-identical to what was
// minted before overrides existed. Overrides are opt-in and rare; a regression
// here would move every token in the system.
func TestResolveTokenPermissions_NoOverridesIsUnchanged(t *testing.T) {
	repo, svc, userID := tokenPermsFixture(t)

	got, err := svc.ResolveTokenPermissions(context.Background(), uuid.New(), userID)
	require.NoError(t, err)

	assert.Equal(t, repo.userPerms[userID], got.Permissions)
	assert.Nil(t, got.Denied)
	assert.Equal(t, map[string]string{"crm:contact:read": ScopeOwn}, got.Scopes)
}

func TestResolveTokenPermissions_DenyDropsTheKeyAndRecordsIt(t *testing.T) {
	repo, svc, userID := tokenPermsFixture(t)
	repo.userOverrides[userID] = []CapabilityOverride{
		{Key: "documents:file:upload", Mode: OverrideModeDeny, Scope: ScopeAll},
	}

	got, err := svc.ResolveTokenPermissions(context.Background(), uuid.New(), userID)
	require.NoError(t, err)

	assert.NotContains(t, got.Permissions, "documents:file:upload")
	assert.Equal(t, []string{"documents:file:upload"}, got.Denied)

	// The coarse keys are the currency of most RequirePermission gates. Losing
	// them would 403 an account the moment it gets its first override.
	assert.Contains(t, got.Permissions, "files:write")
	assert.Contains(t, got.Permissions, "contacts:read")
	assert.Contains(t, got.Permissions, "crm:contact:read")

	// Written in at the narrowest scope, not dropped: absence reads as "all".
	assert.Equal(t, ScopeOwn, got.Scopes["documents:file:upload"])
}

func TestResolveTokenPermissions_AllowAddsTheKeyAtItsScope(t *testing.T) {
	repo, svc, userID := tokenPermsFixture(t)
	repo.userOverrides[userID] = []CapabilityOverride{
		{Key: "hr:employee:read", Mode: OverrideModeAllow, Scope: ScopeTeam},
	}

	got, err := svc.ResolveTokenPermissions(context.Background(), uuid.New(), userID)
	require.NoError(t, err)

	assert.Contains(t, got.Permissions, "hr:employee:read")
	assert.Equal(t, ScopeTeam, got.Scopes["hr:employee:read"])
	assert.Empty(t, got.Denied)
}

// An allow sets the scope rather than raising it, so it can also narrow one
// person without cloning a role — and an allow at "all" has to remove the
// narrower entry the roles put there instead of leaving it behind.
func TestResolveTokenPermissions_AllowSetsScopeInBothDirections(t *testing.T) {
	t.Run("widen to all clears the narrowed entry", func(t *testing.T) {
		repo, svc, userID := tokenPermsFixture(t)
		repo.userOverrides[userID] = []CapabilityOverride{
			{Key: "crm:contact:read", Mode: OverrideModeAllow, Scope: ScopeAll},
		}

		got, err := svc.ResolveTokenPermissions(context.Background(), uuid.New(), userID)
		require.NoError(t, err)
		assert.NotContains(t, got.Scopes, "crm:contact:read", "absence is how the claim spells all")
		assert.Contains(t, got.Permissions, "crm:contact:read")
	})

	t.Run("narrow a key the roles grant at all", func(t *testing.T) {
		repo, svc, userID := tokenPermsFixture(t)
		repo.userOverrides[userID] = []CapabilityOverride{
			{Key: "documents:file:upload", Mode: OverrideModeAllow, Scope: ScopeOwn},
		}

		got, err := svc.ResolveTokenPermissions(context.Background(), uuid.New(), userID)
		require.NoError(t, err)
		assert.Equal(t, ScopeOwn, got.Scopes["documents:file:upload"])
	})
}

// An allow for a key the roles already grant must not duplicate it: the claim
// is read as a set, but a doubled entry would grow every request header.
func TestResolveTokenPermissions_AllowOfAHeldKeyDoesNotDuplicate(t *testing.T) {
	repo, svc, userID := tokenPermsFixture(t)
	repo.userOverrides[userID] = []CapabilityOverride{
		{Key: "documents:file:upload", Mode: OverrideModeAllow, Scope: ScopeAll},
	}

	got, err := svc.ResolveTokenPermissions(context.Background(), uuid.New(), userID)
	require.NoError(t, err)

	seen := 0
	for _, p := range got.Permissions {
		if p == "documents:file:upload" {
			seen++
		}
	}
	assert.Equal(t, 1, seen)
}

// A mode the CHECK constraint rules out today must not be read as an allow.
// Handing out a right nobody meant to grant is the failure we cannot afford, so
// anything that is not a literal allow takes the key away.
func TestResolveTokenPermissions_UnknownModeDenies(t *testing.T) {
	repo, svc, userID := tokenPermsFixture(t)
	repo.userOverrides[userID] = []CapabilityOverride{
		{Key: "documents:file:upload", Mode: "maybe", Scope: ScopeAll},
	}

	got, err := svc.ResolveTokenPermissions(context.Background(), uuid.New(), userID)
	require.NoError(t, err)
	assert.NotContains(t, got.Permissions, "documents:file:upload")
	assert.Contains(t, got.Denied, "documents:file:upload")
}

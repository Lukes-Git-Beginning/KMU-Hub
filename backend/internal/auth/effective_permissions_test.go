package auth

import (
	"context"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sourceIDs builds the expected Sources slice for a set of contributing roles:
// role IDs as strings, sorted — the frontend looks each entry up in the Roles
// slice of the same response, so names would never match.
func sourceIDs(ids ...uuid.UUID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id.String()
	}
	slices.Sort(out)
	return out
}

// grantRow keeps the table-driven fixtures below readable — the resolver takes
// one row per (role, capability) pair and the interesting cases are the ones
// where the same key arrives twice.
func grantRow(roleID uuid.UUID, roleName, key, scope string) EffectiveGrantRow {
	return EffectiveGrantRow{
		RoleID:       roleID,
		RoleName:     roleName,
		RoleColor:    "hsl(217 91% 60%)",
		RoleIsSystem: true,
		Key:          key,
		Scope:        scope,
	}
}

func TestGetEffectivePermissions_WidestScopeWinsAndSourcesAccumulate(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, nil)

	userID := uuid.New()
	manager := uuid.New()
	hrAdmin := uuid.New()

	// documents:file:read arrives through both roles with different scopes,
	// work:task:edit only through manager. The role order is deliberately not
	// alphabetical to prove the resolver does not rely on the query's ORDER BY.
	repo.effectiveGrants[userID] = []EffectiveGrantRow{
		grantRow(manager, "manager", "documents:file:read", "team"),
		grantRow(manager, "manager", "work:task:edit", "own"),
		grantRow(hrAdmin, "hr_admin", "documents:file:read", "all"),
		grantRow(hrAdmin, "hr_admin", "hr:employee:read", "all"),
	}

	got, err := svc.GetEffectivePermissions(context.Background(), userID)
	require.NoError(t, err)

	caps := make(map[string]EffectiveCapability, len(got.Capabilities))
	for _, c := range got.Capabilities {
		caps[c.Key] = c
	}
	require.Len(t, caps, 3)

	// team + all → all: a second role can only widen reach, never narrow it.
	assert.Equal(t, "all", caps["documents:file:read"].Scope)
	// Sources lists every contributing role, including the one that lost the
	// scope comparison — the admin UI has to show where a right comes from.
	assert.Equal(t, sourceIDs(manager, hrAdmin), caps["documents:file:read"].Sources)

	assert.Equal(t, "own", caps["work:task:edit"].Scope)
	assert.Equal(t, sourceIDs(manager), caps["work:task:edit"].Sources)
	assert.Equal(t, sourceIDs(hrAdmin), caps["hr:employee:read"].Sources)

	// Roles are distinct and sorted, no matter how often a role contributed.
	require.Len(t, got.Roles, 2)
	assert.Equal(t, "hr_admin", got.Roles[0].Name)
	assert.Equal(t, "manager", got.Roles[1].Name)
	assert.True(t, got.Roles[0].IsSystem)
}

func TestGetEffectivePermissions_ScopeOrderIsOwnTeamAll(t *testing.T) {
	userID := uuid.New()
	roleA, roleB := uuid.New(), uuid.New()

	cases := []struct {
		name  string
		first string
		other string
		want  string
	}{
		{"own loses to team", "own", "team", "team"},
		{"team loses to all", "team", "all", "all"},
		{"own loses to all", "own", "all", "all"},
		{"widest first still wins", "all", "own", "all"},
		{"identical scopes stay", "team", "team", "team"},
		// The CHECK constraint rules this out in the database; if it ever gets
		// through, the narrow reading is the safe one.
		{"unknown scope collapses to own", "supervisor", "own", "own"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newMockRepository()
			svc := NewService(repo, nil)
			repo.effectiveGrants[userID] = []EffectiveGrantRow{
				grantRow(roleA, "role_a", "crm:contact:read", tc.first),
				grantRow(roleB, "role_b", "crm:contact:read", tc.other),
			}

			got, err := svc.GetEffectivePermissions(context.Background(), userID)
			require.NoError(t, err)
			require.Len(t, got.Capabilities, 1)
			assert.Equal(t, tc.want, got.Capabilities[0].Scope)
			assert.Equal(t, sourceIDs(roleA, roleB), got.Capabilities[0].Sources)
		})
	}
}

func TestGetEffectivePermissions_RoleWithoutFineGrainedGrantStillListed(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, nil)

	userID := uuid.New()
	bare := uuid.New()
	// Empty Key is how the repository reports a role whose grants are all
	// coarse legacy keys — the role exists and must show up, but it
	// contributes no capability.
	repo.effectiveGrants[userID] = []EffectiveGrantRow{grantRow(bare, "platform_admin", "", "")}

	got, err := svc.GetEffectivePermissions(context.Background(), userID)
	require.NoError(t, err)

	require.Len(t, got.Roles, 1)
	assert.Equal(t, "platform_admin", got.Roles[0].Name)
	assert.Empty(t, got.Capabilities)
}

func TestGetEffectivePermissions_NoRolesYieldsEmptySlicesNotNil(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, nil)

	got, err := svc.GetEffectivePermissions(context.Background(), uuid.New())
	require.NoError(t, err)

	// Empty, not nil: the response marshals to [] and the frontend's
	// default-deny lookup needs a container, not null.
	require.NotNil(t, got)
	assert.Empty(t, got.Roles)
	assert.Empty(t, got.Capabilities)
	assert.NotNil(t, got.Roles)
	assert.NotNil(t, got.Capabilities)
}

func TestGetEffectivePermissions_CapabilitiesSortedByKey(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, nil)

	userID := uuid.New()
	role := uuid.New()
	repo.effectiveGrants[userID] = []EffectiveGrantRow{
		grantRow(role, "member", "work:task:read", "all"),
		grantRow(role, "member", "crm:contact:read", "all"),
		grantRow(role, "member", "documents:file:read", "all"),
	}

	got, err := svc.GetEffectivePermissions(context.Background(), userID)
	require.NoError(t, err)

	keys := make([]string, 0, len(got.Capabilities))
	for _, c := range got.Capabilities {
		keys = append(keys, c.Key)
	}
	assert.Equal(t, []string{"crm:contact:read", "documents:file:read", "work:task:read"}, keys)
}

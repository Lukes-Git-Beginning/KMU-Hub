package settings_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/settings"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// TestTenantWrites_MemberAndManagerRolesRejectedAdminAllowed is the proof the
// gateway relies on for PUT /admin/branding and PUT /settings/{m}/tenant: both
// routes only check settings:write, which the system roles member and manager
// hold for their personal settings. The tenant-wide write is stopped here, by
// the real role checker against the seeded system roles — not by a fake.
func TestTenantWrites_MemberAndManagerRolesRejectedAdminAllowed(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	// t.Cleanup throughout, LIFO: users, then the tenant (users.tenant_id has
	// no cascade), then the pool.
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Settings Role Guard Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenantID) })

	sysCtx := testutil.WithSystemCtx(context.Background())
	userWithRole := func(role string) uuid.UUID {
		id := testutil.SeedRow(t, pool, "users", map[string]any{
			"tenant_id":     tenantID,
			"email":         fmt.Sprintf("settings-%s-%s@tenanta.local", role, uuid.New().String()[:8]),
			"password_hash": "$argon2id$v=19$test",
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", id) })
		tag, err := pool.Exec(sysCtx,
			`INSERT INTO user_roles (user_id, role_id, tenant_id)
			 SELECT $1, id, $2 FROM roles WHERE name = $3 AND tenant_id IS NULL`,
			id, tenantID, role)
		require.NoError(t, err)
		require.EqualValues(t, 1, tag.RowsAffected(), "system role %q must exist", role)
		return id
	}

	member := userWithRole("member")
	manager := userWithRole("manager")
	admin := userWithRole("admin")
	lead := userWithRole("member")

	svc := settings.NewService(settings.NewPostgresRepository(pool), settings.NewPostgresRoleChecker(pool))
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	_, err := svc.GrantModuleLead(ctx, tenantID, admin, lead, "work")
	require.NoError(t, err)

	branding := func() *settings.Branding {
		return &settings.Branding{Name: "Fremd GmbH", AccentColor: "#3B82F6"}
	}
	entries := func() []*settings.SettingEntry {
		return []*settings.SettingEntry{{Key: "billableByDefault", Value: rawJSON(t, false)}}
	}

	for name, caller := range map[string]uuid.UUID{"member": member, "manager": manager} {
		_, err := svc.PutBranding(ctx, tenantID, caller, branding())
		assert.ErrorIs(t, err, settings.ErrNotModuleLead, "%s must not overwrite the tenant branding", name)

		_, err = svc.PutTenantSettings(ctx, tenantID, caller, "work", entries())
		assert.ErrorIs(t, err, settings.ErrNotModuleLead, "%s must not write tenant settings", name)
	}

	// A module-lead is usually a member: the gateway must keep letting
	// settings:write through so this path stays reachable.
	_, err = svc.PutTenantSettings(ctx, tenantID, lead, "work", entries())
	require.NoError(t, err, "module-lead (role member) writes its own module's tenant settings")
	_, err = svc.PutBranding(ctx, tenantID, lead, branding())
	assert.ErrorIs(t, err, settings.ErrNotModuleLead, "a lead of another module must not write branding")

	got, err := svc.PutBranding(ctx, tenantID, admin, branding())
	require.NoError(t, err, "admin writes the tenant branding")
	assert.Equal(t, "Fremd GmbH", got.Name)
}

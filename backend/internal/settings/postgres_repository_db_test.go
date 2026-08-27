package settings_test

// Coverage for the PostgresRepository methods that tenant_write_test.go and
// valueset_test.go do not touch: role checking, module-lead/grant reads,
// licensing counts, and the tenant-subscription read. All against a real
// Postgres so the queries (including the users/roles joins) actually run,
// not a fakeRepo standing in for them.
//
// Also covers scope point (3) from BACKLOG.yml: whether concurrent writes to
// the same user/module pair can silently drop a field. ReplaceUserSettings'
// delete-then-insert is checked with two genuinely held connections, not a
// single warm one — a single connection would only prove statement order,
// not what Postgres actually does when two transactions overlap.

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/settings"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestIsAdmin_TrueForAdminRoleFalseOtherwise(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "IsAdmin Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	adminUser := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("isadmin-admin-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", adminUser)

	plainUser := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("isadmin-plain-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", plainUser)

	ctx := testutil.WithSystemCtx(context.Background())
	_, err := pool.Exec(ctx,
		`INSERT INTO user_roles (user_id, role_id, tenant_id)
		 SELECT $1, id, $2 FROM roles WHERE name = 'admin' AND tenant_id IS NULL`,
		adminUser, tenantID)
	require.NoError(t, err)

	checker := settings.NewPostgresRoleChecker(pool)

	// user_roles is forced-RLS: a query without a tenant-scoped (or system)
	// context sees zero rows regardless of the tenantID argument, which would
	// make IsAdmin look permanently false. Production callers always pass a
	// ctx carrying the caller's tenant (see service.go), so the test does too.
	tenantCtx := testutil.WithTenantCtx(context.Background(), tenantID)

	isAdmin, err := checker.IsAdmin(tenantCtx, tenantID, adminUser)
	require.NoError(t, err)
	assert.True(t, isAdmin, "user with the admin role must read as admin")

	isAdmin, err = checker.IsAdmin(tenantCtx, tenantID, plainUser)
	require.NoError(t, err)
	assert.False(t, isAdmin, "user without any role must not read as admin")

	// A user's admin role in a FOREIGN tenant must not leak: IsAdmin's join
	// pins u.tenant_id = $2, so asking about adminUser under tenantOther must
	// come back false even though the same user id holds the role somewhere.
	tenantOther := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOther, "IsAdmin Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	isAdmin, err = checker.IsAdmin(testutil.WithTenantCtx(context.Background(), tenantOther), tenantOther, adminUser)
	require.NoError(t, err)
	assert.False(t, isAdmin, "admin role must not be visible from a foreign tenant_id")
}

func TestModuleLeadReads_ListGetIsLeadAndListForUser(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "ModuleLead Read Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	user := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("lead-read-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", user)

	repo := settings.NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	if _, err := repo.GrantModuleLead(ctx, tenantID, user, "crm", nil); err != nil {
		t.Fatalf("GrantModuleLead(crm): %v", err)
	}
	if _, err := repo.GrantModuleLead(ctx, tenantID, user, "finance", nil); err != nil {
		t.Fatalf("GrantModuleLead(finance): %v", err)
	}

	// GetModuleLead — found and not-found.
	ml, err := repo.GetModuleLead(ctx, tenantID, user, "crm")
	require.NoError(t, err)
	assert.Equal(t, "crm", ml.ModuleID)

	_, err = repo.GetModuleLead(ctx, tenantID, user, "hr")
	assert.ErrorIs(t, err, settings.ErrNotFound)

	// IsModuleLead.
	isLead, err := repo.IsModuleLead(ctx, tenantID, user, "crm")
	require.NoError(t, err)
	assert.True(t, isLead)
	isLead, err = repo.IsModuleLead(ctx, tenantID, user, "hr")
	require.NoError(t, err)
	assert.False(t, isLead)

	// ListLeadModulesForUser.
	mods, err := repo.ListLeadModulesForUser(ctx, tenantID, user)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"crm", "finance"}, mods)

	// ListModuleLeads with the moduleID-only filter (no userID) — the branch
	// that builds "$2" instead of "$3" in the dynamic WHERE clause.
	leads, err := repo.ListModuleLeads(ctx, tenantID, nil, ptrString("crm"))
	require.NoError(t, err)
	require.Len(t, leads, 1)
	assert.Equal(t, user, leads[0].UserID)

	// ListModuleLeads with both filters — the "$3" branch.
	uid := user
	leads, err = repo.ListModuleLeads(ctx, tenantID, &uid, ptrString("finance"))
	require.NoError(t, err)
	require.Len(t, leads, 1)
	assert.Equal(t, "finance", leads[0].ModuleID)

	// Cross-tenant: none of these reads may see the lead from a foreign tenant.
	tenantOther := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOther, "ModuleLead Read Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	leadsOther, err := repo.ListModuleLeads(ctxOther, tenantID, nil, nil)
	require.NoError(t, err)
	assert.Empty(t, leadsOther, "a foreign tenant context must not see another tenant's leads")

	isLead, err = repo.IsModuleLead(ctxOther, tenantID, user, "crm")
	require.NoError(t, err)
	assert.False(t, isLead)
}

func TestListModuleGrants_JoinsUserNameAndScopesToTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "ModuleGrants Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	user := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"first_name":    "Ada",
		"last_name":     "Lovelace",
		"email":         fmt.Sprintf("grants-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", user)

	repo := settings.NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	if _, err := repo.GrantModuleAccess(ctx, tenantID, user, "crm", nil); err != nil {
		t.Fatalf("GrantModuleAccess: %v", err)
	}

	grants, err := repo.ListModuleGrants(ctx, tenantID, nil, nil)
	require.NoError(t, err)
	require.Len(t, grants, 1)
	assert.Equal(t, "Ada Lovelace", grants[0].UserName, "user_name must come from the users join")
	assert.Equal(t, "crm", grants[0].ModuleID)

	// Filtered by userID only (the "$2" branch) and by both (the "$3" branch).
	uid := user
	grants, err = repo.ListModuleGrants(ctx, tenantID, &uid, nil)
	require.NoError(t, err)
	require.Len(t, grants, 1)

	grants, err = repo.ListModuleGrants(ctx, tenantID, &uid, ptrString("crm"))
	require.NoError(t, err)
	require.Len(t, grants, 1)

	grants, err = repo.ListModuleGrants(ctx, tenantID, &uid, ptrString("hr"))
	require.NoError(t, err)
	assert.Empty(t, grants)

	// Cross-tenant isolation.
	tenantOther := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOther, "ModuleGrants Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	grantsOther, err := repo.ListModuleGrants(ctxOther, tenantID, nil, nil)
	require.NoError(t, err)
	assert.Empty(t, grantsOther, "a foreign tenant context must not see another tenant's grants")
}

func TestCountGrantsByModule_CountsPerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "CountGrants Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	userOne := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("count-1-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userOne)
	userTwo := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("count-2-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userTwo)

	repo := settings.NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	for _, u := range []uuid.UUID{userOne, userTwo} {
		if _, err := repo.GrantModuleAccess(ctx, tenantID, u, "crm", nil); err != nil {
			t.Fatalf("GrantModuleAccess(%s, crm): %v", u, err)
		}
	}
	if _, err := repo.GrantModuleAccess(ctx, tenantID, userOne, "finance", nil); err != nil {
		t.Fatalf("GrantModuleAccess(finance): %v", err)
	}

	counts, err := repo.CountGrantsByModule(ctx, tenantID)
	require.NoError(t, err)
	assert.Equal(t, int32(2), counts["crm"])
	assert.Equal(t, int32(1), counts["finance"])
	assert.NotContains(t, counts, "hr", "a module nobody holds must be absent, not zero")

	tenantOther := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOther, "CountGrants Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	countsOther, err := repo.CountGrantsByModule(ctxOther, tenantID)
	require.NoError(t, err)
	assert.Empty(t, countsOther, "a foreign tenant context must not see another tenant's grant counts")
}

func TestGetTenantSubscription_OwnTenantVsForeign(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Subscription Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := settings.NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	sub, err := repo.GetTenantSubscription(ctx, tenantID)
	require.NoError(t, err)
	assert.Equal(t, "cosmi", sub.PlanType)
	assert.Equal(t, "standard", sub.SupportTier)
	assert.Equal(t, "active", sub.Status)
	assert.Nil(t, sub.BillingPeriodEnd)
	assert.Nil(t, sub.TotalSeats)

	// RLS on `tenants` only exposes the caller's own tenant: reading a foreign
	// tenant's subscription from tenantID's own context must miss entirely,
	// not return the foreign tenant's data.
	tenantOther := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOther, "Subscription Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	_, err = repo.GetTenantSubscription(ctx, tenantOther)
	assert.ErrorIs(t, err, settings.ErrNotFound, "a foreign tenant's subscription row must read as not-found under RLS")
}

// ============================================================================
// Concurrency — scope point (3): does a concurrent write lose a field?
// ============================================================================

// TestReplaceUserSettings_GenuinelyConcurrentAdditionSurvives holds two real
// transactions open at once (not one warm connection executing statements
// back to back) to check whether ReplaceUserSettings' unconditional
// "DELETE everything for this module, then INSERT what I was given" can
// clobber a *different* key added by a truly concurrent writer.
//
// It cannot: ReplaceUserSettings' DELETE only matches rows that are visible
// (i.e. already committed) at the moment the statement runs. A concurrent
// INSERT of an unrelated key does not conflict on any row lock (different
// key = different row) and is invisible to a DELETE that already executed,
// so it survives regardless of which transaction happens to commit last.
func TestReplaceUserSettings_GenuinelyConcurrentAdditionSurvives(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Concurrent Replace Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	user := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("concurrent-replace-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", user)

	repo := settings.NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	const module = "profile"

	// Seed the pre-existing keys the replace call will delete.
	_, err := repo.PutUserSettings(ctx, tenantID, user, module, []*settings.SettingEntry{
		{Key: "language", Value: json.RawMessage(`"de"`)},
		{Key: "region", Value: json.RawMessage(`"DE"`)},
	})
	require.NoError(t, err)

	// tx1 mirrors ReplaceUserSettings' own delete-then-insert, but held open
	// manually so we control when the second connection's write lands
	// relative to tx1's DELETE and its eventual commit.
	tx1, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer tx1.Rollback(ctx) //nolint:errcheck

	_, err = tx1.Exec(ctx,
		`DELETE FROM user_settings WHERE tenant_id = $1 AND user_id = $2 AND module_id = $3`,
		tenantID, user, module)
	require.NoError(t, err, "tx1 delete")

	// While tx1 is still open (uncommitted), a second, fully independent
	// connection runs a real PutUserSettings call adding a new key. This is
	// what a genuinely concurrent request looks like: its own transaction,
	// its own connection, committed on its own schedule.
	done := make(chan error, 1)
	go func() {
		_, addErr := repo.PutUserSettings(ctx, tenantID, user, module, []*settings.SettingEntry{
			{Key: "timezone", Value: json.RawMessage(`"Europe/Berlin"`)},
		})
		done <- addErr
	}()

	select {
	case err := <-done:
		require.NoError(t, err, "concurrent PutUserSettings must not be blocked by tx1's delete on unrelated rows")
	case <-time.After(5 * time.Second):
		t.Fatal("concurrent PutUserSettings did not complete — it must not block on tx1's delete of different rows")
	}

	// tx1 now performs its own insert (as ReplaceUserSettings would) and commits.
	_, err = tx1.Exec(ctx,
		`INSERT INTO user_settings (tenant_id, user_id, module_id, key, value, updated_at)
		 VALUES ($1, $2, $3, $4, $5, now())`,
		tenantID, user, module, "language", json.RawMessage(`"en"`))
	require.NoError(t, err, "tx1 insert")
	require.NoError(t, tx1.Commit(ctx))

	got, err := repo.GetUserSettings(ctx, tenantID, user, module)
	require.NoError(t, err)
	byKey := make(map[string]json.RawMessage, len(got))
	for _, e := range got {
		byKey[e.Key] = e.Value
	}
	assert.Contains(t, byKey, "timezone", "a key added by a genuinely concurrent writer must survive a replace that started before it")
	assert.Contains(t, byKey, "language")
	assert.NotContains(t, byKey, "region", "region was not re-inserted by the replace and must stay gone")
}

// TestReplaceUserSettings_FullReplaceDropsKeysAddedAfterItsSnapshot documents
// the other half of scope point (3): if the sequence is not interleaved but
// strictly sequential — a key is added and committed, and only afterwards
// does a full replace run — the replace's unconditional DELETE (no key
// filter) removes it too, because it has no way to know the key exists
// unless it was told about it. This is not a lost-update RACE (no two
// statements touch the same row concurrently); it is the documented
// full-replace contract already called out in ReplaceUserSettings' doc
// comment and mirrored from auth.PostgresRepository.SetUserOverrides. A
// client that fetches settings, then PUTs its own idea of the complete set,
// will always drop anything added in between — the same as any REST PUT
// without an ETag/If-Match guard. No fix filed: this is intended PUT
// semantics, not a defect introduced by this repository.
func TestReplaceUserSettings_FullReplaceDropsKeysAddedAfterItsSnapshot(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Sequential Replace Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	user := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("sequential-replace-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", user)

	repo := settings.NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	const module = "profile"

	_, err := repo.PutUserSettings(ctx, tenantID, user, module, []*settings.SettingEntry{
		{Key: "language", Value: json.RawMessage(`"de"`)},
	})
	require.NoError(t, err)

	// Fully committed before the replace call even starts.
	_, err = repo.PutUserSettings(ctx, tenantID, user, module, []*settings.SettingEntry{
		{Key: "timezone", Value: json.RawMessage(`"Europe/Berlin"`)},
	})
	require.NoError(t, err)

	replaced, err := repo.ReplaceUserSettings(ctx, tenantID, user, module, []*settings.SettingEntry{
		{Key: "language", Value: json.RawMessage(`"en"`)},
	})
	require.NoError(t, err)
	require.Len(t, replaced, 1)

	got, err := repo.GetUserSettings(ctx, tenantID, user, module)
	require.NoError(t, err)
	require.Len(t, got, 1, "a full replace drops every key not in its payload, including ones added before it ran")
	assert.Equal(t, "language", got[0].Key)
}

func ptrString(s string) *string { return &s }

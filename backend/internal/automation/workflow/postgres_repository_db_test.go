package workflow

// Backlog unit e-cov-automation-workflow-repo. tenant_write_test.go and
// time_trigger_db_test.go already cover Create, Update, GetByID,
// CreateExecution, GetExecution, UpdateExecution, ListActiveTimeBased and
// ClaimTimeTrigger against a real pool. This file covers the remaining
// PostgresRepository methods that were previously exercised only through
// mocks in service_test.go/engine_test.go/poller_test.go: Delete, the
// non-AutomationID List filter branches plus its Limit/Offset
// normalization, SetActive, UpdateLastTriggered, ClaimTimeTriggerFire,
// ListActiveByTriggerType, CleanupOldExecutions and the full
// TemplateRepository.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestPostgresRepository_Delete(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Delete Own Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Delete Other Tenant")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("delete-%s@example.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)

	auto := &models.Automation{
		TenantID:      tenantOwn,
		Name:          "Delete Test Automation",
		Scope:         models.AutomationScopePersonal,
		OwnerID:       userID,
		TriggerType:   "contact.created",
		TriggerConfig: []byte(`{}`),
		Conditions:    []byte(`{}`),
		Actions:       []byte(`[]`),
		IsActive:      true,
	}
	require.NoError(t, repo.Create(ctxOwn, auto))
	defer testutil.CleanupRow(t, pool, "automations", auto.ID)

	// A foreign tenant_id in the WHERE clause must not find the row at all —
	// it must not be deleted and must report ErrAutomationNotFound.
	err := repo.Delete(ctxOther, auto.ID, tenantOther)
	assert.ErrorIs(t, err, ErrAutomationNotFound)
	testutil.AssertRowCount(t, pool, ctxOwn, "automations", auto.ID, 1)

	require.NoError(t, repo.Delete(ctxOwn, auto.ID, tenantOwn))
	testutil.AssertRowCount(t, pool, ctxOwn, "automations", auto.ID, 0)

	// Deleting again (already gone) must also report ErrAutomationNotFound.
	err = repo.Delete(ctxOwn, auto.ID, tenantOwn)
	assert.ErrorIs(t, err, ErrAutomationNotFound)
}

func TestPostgresRepository_List_Filters(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "List Filters Tenant")

	userA := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("list-a-%s@example.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userA)
	userB := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("list-b-%s@example.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userB)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	sharedTrigger := fmt.Sprintf("test.trigger.%s", uuid.New().String()[:8])

	auto1 := &models.Automation{
		TenantID: tenantID, Name: "A1", Scope: models.AutomationScopePersonal,
		OwnerID: userA, TriggerType: sharedTrigger,
		TriggerConfig: []byte(`{}`), Conditions: []byte(`{}`), Actions: []byte(`[]`),
		IsActive: true,
	}
	require.NoError(t, repo.Create(ctx, auto1))
	defer testutil.CleanupRow(t, pool, "automations", auto1.ID)

	auto2 := &models.Automation{
		TenantID: tenantID, Name: "A2", Scope: "team",
		OwnerID: userB, TriggerType: "other.trigger",
		TriggerConfig: []byte(`{}`), Conditions: []byte(`{}`), Actions: []byte(`[]`),
		IsActive: false,
	}
	require.NoError(t, repo.Create(ctx, auto2))
	defer testutil.CleanupRow(t, pool, "automations", auto2.ID)

	auto3 := &models.Automation{
		TenantID: tenantID, Name: "A3", Scope: "organization",
		OwnerID: userA, TriggerType: sharedTrigger,
		TriggerConfig: []byte(`{}`), Conditions: []byte(`{}`), Actions: []byte(`[]`),
		IsActive: true,
	}
	require.NoError(t, repo.Create(ctx, auto3))
	defer testutil.CleanupRow(t, pool, "automations", auto3.ID)

	t.Run("OwnerID", func(t *testing.T) {
		got, total, err := repo.List(ctx, ListFilter{TenantID: tenantID, OwnerID: &userA})
		require.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, got, 2)
		for _, a := range got {
			assert.Equal(t, userA, a.OwnerID)
		}
	})

	t.Run("Scope", func(t *testing.T) {
		scope := "team"
		got, total, err := repo.List(ctx, ListFilter{TenantID: tenantID, Scope: &scope})
		require.NoError(t, err)
		require.Equal(t, 1, total)
		require.Len(t, got, 1)
		assert.Equal(t, auto2.ID, got[0].ID)
	})

	t.Run("TriggerType", func(t *testing.T) {
		got, total, err := repo.List(ctx, ListFilter{TenantID: tenantID, TriggerType: &sharedTrigger})
		require.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, got, 2)
		for _, a := range got {
			assert.Equal(t, sharedTrigger, a.TriggerType)
		}
	})

	t.Run("IsActive", func(t *testing.T) {
		inactive := false
		got, total, err := repo.List(ctx, ListFilter{TenantID: tenantID, IsActive: &inactive})
		require.NoError(t, err)
		require.Equal(t, 1, total)
		require.Len(t, got, 1)
		assert.Equal(t, auto2.ID, got[0].ID)
	})

	// Limit <= 0 must fall back to the default of 50 rather than LIMIT 0 —
	// with only 3 fixtures in this tenant, a broken default would return an
	// empty slice here.
	t.Run("LimitDefault", func(t *testing.T) {
		got, total, err := repo.List(ctx, ListFilter{TenantID: tenantID, Limit: 0})
		require.NoError(t, err)
		assert.Equal(t, 3, total)
		assert.Len(t, got, 3)
	})

	// A negative Offset must normalize to 0. Postgres itself rejects a
	// literal negative OFFSET, so a broken normalization would surface as a
	// query error here, not just a wrong result.
	t.Run("NegativeOffsetNormalizes", func(t *testing.T) {
		got, total, err := repo.List(ctx, ListFilter{TenantID: tenantID, Offset: -5})
		require.NoError(t, err)
		assert.Equal(t, 3, total)
		assert.Len(t, got, 3)
	})
}

func TestPostgresRepository_SetActive(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "SetActive Tenant")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("setactive-%s@example.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	auto := &models.Automation{
		TenantID: tenantID, Name: "SetActive Test", Scope: models.AutomationScopePersonal,
		OwnerID: userID, TriggerType: "contact.created",
		TriggerConfig: []byte(`{}`), Conditions: []byte(`{}`), Actions: []byte(`[]`),
		IsActive: false,
	}
	require.NoError(t, repo.Create(ctx, auto))
	defer testutil.CleanupRow(t, pool, "automations", auto.ID)

	require.NoError(t, repo.SetActive(ctx, auto.ID, tenantID, true))
	got, err := repo.GetByID(ctx, auto.ID, tenantID)
	require.NoError(t, err)
	assert.True(t, got.IsActive)

	err = repo.SetActive(ctx, uuid.New(), tenantID, true)
	assert.ErrorIs(t, err, ErrAutomationNotFound)
}

func TestPostgresRepository_UpdateLastTriggered(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "UpdateLastTriggered Tenant")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("lasttrig-%s@example.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	auto := &models.Automation{
		TenantID: tenantID, Name: "UpdateLastTriggered Test", Scope: models.AutomationScopePersonal,
		OwnerID: userID, TriggerType: "contact.created",
		TriggerConfig: []byte(`{}`), Conditions: []byte(`{}`), Actions: []byte(`[]`),
		IsActive: true,
	}
	require.NoError(t, repo.Create(ctx, auto))
	defer testutil.CleanupRow(t, pool, "automations", auto.ID)

	at := time.Now().UTC().Truncate(time.Microsecond)
	require.NoError(t, repo.UpdateLastTriggered(ctx, auto.ID, at))

	got, err := repo.GetByID(ctx, auto.ID, tenantID)
	require.NoError(t, err)
	require.NotNil(t, got.LastTriggeredAt)
	assert.WithinDuration(t, at, *got.LastTriggeredAt, time.Second)

	err = repo.UpdateLastTriggered(ctx, uuid.New(), at)
	assert.ErrorIs(t, err, ErrAutomationNotFound)
}

func TestPostgresRepository_ClaimTimeTriggerFire(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "ClaimTimeTriggerFire Tenant")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("firetrig-%s@example.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	autoID := testutil.SeedRow(t, pool, "automations", map[string]any{
		"tenant_id":    tenantID,
		"name":         "ClaimTimeTriggerFire Test",
		"description":  "",
		"owner_id":     userID,
		"trigger_type": "biz.invoice.overdue",
		"is_active":    true,
	})
	defer testutil.CleanupRow(t, pool, "automations", autoID)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithSystemCtx(context.Background())

	entityKey := fmt.Sprintf("invoice:%s:day1", uuid.New().String()[:8])
	now := time.Now()

	claimedFirst, err := repo.ClaimTimeTriggerFire(ctx, tenantID, autoID, entityKey, now)
	require.NoError(t, err)
	assert.True(t, claimedFirst, "first claim for a fresh (automation_id, entity_key) pair must win")

	claimedSecond, err := repo.ClaimTimeTriggerFire(ctx, tenantID, autoID, entityKey, now.Add(time.Minute))
	require.NoError(t, err)
	assert.False(t, claimedSecond, "a second claim for the same (automation_id, entity_key) pair must lose")

	// A different entity_key for the same automation must be independent.
	otherKey := fmt.Sprintf("invoice:%s:day1", uuid.New().String()[:8])
	claimedOther, err := repo.ClaimTimeTriggerFire(ctx, tenantID, autoID, otherKey, now)
	require.NoError(t, err)
	assert.True(t, claimedOther, "a different entity_key must not be blocked by an unrelated claim")
}

func TestPostgresRepository_ListActiveByTriggerType(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "ListActiveByTriggerType Tenant")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("bytrigger-%s@example.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	// Random per-run trigger_type so no other parallel test's fixed literal
	// trigger types (this method carries no tenant_id predicate) can leak in.
	matchType := fmt.Sprintf("test.trigger.%s", uuid.New().String()[:8])

	activeMatch := testutil.SeedRow(t, pool, "automations", map[string]any{
		"tenant_id": tenantID, "name": "Active Match", "description": "",
		"owner_id": userID, "trigger_type": matchType, "is_active": true,
	})
	defer testutil.CleanupRow(t, pool, "automations", activeMatch)

	inactiveMatch := testutil.SeedRow(t, pool, "automations", map[string]any{
		"tenant_id": tenantID, "name": "Inactive Match", "description": "",
		"owner_id": userID, "trigger_type": matchType, "is_active": false,
	})
	defer testutil.CleanupRow(t, pool, "automations", inactiveMatch)

	activeOther := testutil.SeedRow(t, pool, "automations", map[string]any{
		"tenant_id": tenantID, "name": "Active Other Type", "description": "",
		"owner_id": userID, "trigger_type": "other.trigger", "is_active": true,
	})
	defer testutil.CleanupRow(t, pool, "automations", activeOther)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithSystemCtx(context.Background())

	got, err := repo.ListActiveByTriggerType(ctx, matchType)
	require.NoError(t, err)
	require.Len(t, got, 1, "only the active automation matching trigger_type must be returned")
	assert.Equal(t, activeMatch, got[0].ID)
}

func TestPostgresRepository_CleanupOldExecutions(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "CleanupOldExecutions Tenant")

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("cleanup-%s@example.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	autoID := testutil.SeedRow(t, pool, "automations", map[string]any{
		"tenant_id": tenantID, "name": "CleanupOldExecutions Test", "description": "",
		"owner_id": userID, "trigger_type": "contact.created", "is_active": true,
	})
	defer testutil.CleanupRow(t, pool, "automations", autoID)

	now := time.Now()

	// CreateExecution always overwrites StartedAt with time.Now(), so old
	// fixtures have to be seeded directly to get a started_at in the past.
	oldCompleted := testutil.SeedRow(t, pool, "automation_executions", map[string]any{
		"tenant_id": tenantID, "automation_id": autoID, "chain_id": uuid.New(),
		"trigger_event": []byte(`{}`), "condition_result": true, "status": "completed",
		"steps": []byte(`[]`), "started_at": now.Add(-48 * time.Hour), "completed_at": now.Add(-47 * time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "automation_executions", oldCompleted)

	recentCompleted := testutil.SeedRow(t, pool, "automation_executions", map[string]any{
		"tenant_id": tenantID, "automation_id": autoID, "chain_id": uuid.New(),
		"trigger_event": []byte(`{}`), "condition_result": true, "status": "completed",
		"steps": []byte(`[]`), "started_at": now.Add(-1 * time.Hour), "completed_at": now.Add(-30 * time.Minute),
	})
	defer testutil.CleanupRow(t, pool, "automation_executions", recentCompleted)

	oldRunning := testutil.SeedRow(t, pool, "automation_executions", map[string]any{
		"tenant_id": tenantID, "automation_id": autoID, "chain_id": uuid.New(),
		"trigger_event": []byte(`{}`), "condition_result": true, "status": "running",
		"steps": []byte(`[]`), "started_at": now.Add(-48 * time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "automation_executions", oldRunning)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithSystemCtx(context.Background())

	deleted, err := repo.CleanupOldExecutions(ctx, 24*time.Hour)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, deleted, int64(1))

	testutil.AssertRowCount(t, pool, ctx, "automation_executions", oldCompleted, 0)
	testutil.AssertRowCount(t, pool, ctx, "automation_executions", recentCompleted, 1)
	testutil.AssertRowCount(t, pool, ctx, "automation_executions", oldRunning, 1)
}

func TestPostgresRepository_TemplateRepository(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithSystemCtx(context.Background())

	categoryA := fmt.Sprintf("test-category-a-%s", uuid.New().String()[:8])
	categoryB := fmt.Sprintf("test-category-b-%s", uuid.New().String()[:8])

	tplA := &models.AutomationTemplate{
		ID: fmt.Sprintf("test-tpl-a-%s", uuid.New().String()[:8]), Name: "Template A",
		Description: "First template", Category: categoryA, Complexity: "einfach",
		TriggerType: "contact.created", TriggerConfig: []byte(`{}`),
		Conditions: []byte(`{}`), Actions: []byte(`[]`),
	}
	require.NoError(t, repo.UpsertTemplate(ctx, tplA))
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM automation_templates WHERE id = $1", tplA.ID)
	}()

	tplB := &models.AutomationTemplate{
		ID: fmt.Sprintf("test-tpl-b-%s", uuid.New().String()[:8]), Name: "Template B",
		Description: "Second template", Category: categoryB, Complexity: "mittel",
		TriggerType: "deal.created", TriggerConfig: []byte(`{}`),
		Conditions: []byte(`{}`), Actions: []byte(`[]`),
	}
	require.NoError(t, repo.UpsertTemplate(ctx, tplB))
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM automation_templates WHERE id = $1", tplB.ID)
	}()

	t.Run("GetTemplate_found", func(t *testing.T) {
		got, err := repo.GetTemplate(ctx, tplA.ID)
		require.NoError(t, err)
		assert.Equal(t, tplA.Name, got.Name)
		assert.Equal(t, tplA.Category, got.Category)
	})

	t.Run("GetTemplate_not_found", func(t *testing.T) {
		_, err := repo.GetTemplate(ctx, "does-not-exist-"+uuid.New().String())
		assert.ErrorIs(t, err, ErrTemplateNotFound)
	})

	t.Run("ListTemplates_no_category_filter", func(t *testing.T) {
		got, err := repo.ListTemplates(ctx, nil)
		require.NoError(t, err)
		var foundA, foundB bool
		for _, tpl := range got {
			if tpl.ID == tplA.ID {
				foundA = true
			}
			if tpl.ID == tplB.ID {
				foundB = true
			}
		}
		assert.True(t, foundA, "ListTemplates(nil) must include template A")
		assert.True(t, foundB, "ListTemplates(nil) must include template B")
	})

	t.Run("ListTemplates_category_filter", func(t *testing.T) {
		got, err := repo.ListTemplates(ctx, &categoryA)
		require.NoError(t, err)
		require.Len(t, got, 1, "category filter must exclude template B's category")
		assert.Equal(t, tplA.ID, got[0].ID)
	})

	// UpsertTemplate must UPDATE on a conflicting id, not fail or duplicate.
	t.Run("UpsertTemplate_updates_on_conflict", func(t *testing.T) {
		updated := &models.AutomationTemplate{
			ID: tplA.ID, Name: "Template A Renamed", Description: "Updated description",
			Category: categoryA, Complexity: "fortgeschritten", TriggerType: "contact.created",
			TriggerConfig: []byte(`{}`), Conditions: []byte(`{}`), Actions: []byte(`[]`),
		}
		require.NoError(t, repo.UpsertTemplate(ctx, updated))

		got, err := repo.GetTemplate(ctx, tplA.ID)
		require.NoError(t, err)
		assert.Equal(t, "Template A Renamed", got.Name)
		assert.Equal(t, "fortgeschritten", got.Complexity)

		all, err := repo.ListTemplates(ctx, &categoryA)
		require.NoError(t, err)
		assert.Len(t, all, 1, "the conflicting upsert must not have created a duplicate row")
	})
}

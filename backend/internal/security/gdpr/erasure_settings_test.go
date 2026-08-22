package gdpr

// Integration coverage for SettingsErasureHandler -- the eighth erasure
// handler, covering four personal-settings tables that CASCADE on users(id)
// but belong to no other domain handler (auth, crm, chat, work, calendar,
// notifications, audit): user_settings, user_dashboard_layouts,
// user_project_preferences and saved_filters. Without this handler the
// tables never fire their CASCADE, because AuthErasureHandler anonymizes the
// user row in place instead of deleting it.

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestSettingsErasureHandler_ModuleName(t *testing.T) {
	h := NewSettingsErasureHandler(nil)
	assert.Equal(t, "settings", h.ModuleName())
}

func TestSettingsErasureHandler_ExecuteErasure_Integration(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Settings Erasure Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	userID := seedExportUser(t, pool, tenantOwn, "settings-erasure-subject")
	defer testutil.CleanupRow(t, pool, "users", userID)
	colleagueID := seedExportUser(t, pool, tenantOwn, "settings-erasure-colleague")
	defer testutil.CleanupRow(t, pool, "users", colleagueID)

	ctx := testutil.WithSystemCtx(context.Background())
	if _, err := pool.Exec(ctx,
		`INSERT INTO user_settings (tenant_id, user_id, module_id, key, value) VALUES ($1, $2, $3, $4, $5)`,
		tenantOwn, userID, "crm", "theme", `"dark"`,
	); err != nil {
		t.Fatalf("seed user_settings: %v", err)
	}

	dashboardID := testutil.SeedRow(t, pool, "user_dashboard_layouts", map[string]any{
		"tenant_id": tenantOwn,
		"user_id":   userID,
	})
	defer testutil.CleanupRow(t, pool, "user_dashboard_layouts", dashboardID)

	projectID := testutil.SeedRow(t, pool, "projects", map[string]any{
		"tenant_id":   tenantOwn,
		"name":        "Projekt Settings-Loeschtest",
		"project_key": fmt.Sprintf("SL%d", uuid.New().ID()%1000),
		"created_by":  colleagueID,
	})
	defer testutil.CleanupRow(t, pool, "projects", projectID)
	if _, err := pool.Exec(ctx,
		`INSERT INTO user_project_preferences (tenant_id, user_id, project_id, view_type) VALUES ($1, $2, $3, $4)`,
		tenantOwn, userID, projectID, "kanban",
	); err != nil {
		t.Fatalf("seed user_project_preferences: %v", err)
	}

	filterID := testutil.SeedRow(t, pool, "saved_filters", map[string]any{
		"tenant_id":   tenantOwn,
		"name":        "Meine Kontakte",
		"entity_type": "contact",
		"filter_json": `{"status":"active"}`,
		"created_by":  userID,
	})
	defer testutil.CleanupRow(t, pool, "saved_filters", filterID)

	// A colleague's own settings must survive the subject's erasure untouched.
	colleagueFilterID := testutil.SeedRow(t, pool, "saved_filters", map[string]any{
		"tenant_id":   tenantOwn,
		"name":        "Kollegen-Filter",
		"entity_type": "deal",
		"filter_json": `{"stage":"won"}`,
		"created_by":  colleagueID,
	})
	defer testutil.CleanupRow(t, pool, "saved_filters", colleagueFilterID)

	tenantCtx := testutil.WithTenantCtx(context.Background(), tenantOwn)

	h := NewSettingsErasureHandler(pool)

	preview, err := h.PreviewErasure(tenantCtx, userID)
	require.NoError(t, err)

	affected, err := h.ExecuteErasure(tenantCtx, userID, erasureLabel, ErasureDelete)
	require.NoError(t, err)

	// 1 user_settings row + 1 dashboard layout + 1 project preference + 1 saved
	// filter of the subject = 4. The colleague's filter is not counted.
	const wantAffected = 4
	assert.Equal(t, wantAffected, affected, "affected count must match PreviewErasure exactly")
	assert.Equal(t, wantAffected, preview.RecordCount, "PreviewErasure must report the same total ExecuteErasure later affects")

	var settingsRows int
	require.NoError(t, pool.QueryRow(tenantCtx,
		`SELECT COUNT(*) FROM user_settings WHERE user_id = $1`, userID,
	).Scan(&settingsRows))
	assert.Equal(t, 0, settingsRows)

	testutil.AssertRowCount(t, pool, tenantCtx, "user_dashboard_layouts", dashboardID, 0)

	var prefRows int
	require.NoError(t, pool.QueryRow(tenantCtx,
		`SELECT COUNT(*) FROM user_project_preferences WHERE user_id = $1 AND project_id = $2`, userID, projectID,
	).Scan(&prefRows))
	assert.Equal(t, 0, prefRows)

	testutil.AssertRowCount(t, pool, tenantCtx, "saved_filters", filterID, 0)

	// The colleague's filter survives untouched.
	testutil.AssertRowCount(t, pool, tenantCtx, "saved_filters", colleagueFilterID, 1)
}

func TestSettingsErasureHandler_ExecuteErasure_DeadPool(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	pool.Close()

	h := NewSettingsErasureHandler(pool)
	ctx := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	userID := uuid.New()

	affected, err := h.ExecuteErasure(ctx, userID, erasureLabel, ErasureDelete)
	require.Error(t, err, "a dead pool must surface as an error, not as a silent no-op erasure")
	assert.Equal(t, 0, affected)
}

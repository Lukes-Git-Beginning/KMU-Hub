package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/plugin/repository"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// TestExecutionLog_ListWithInstallationFilterAndLimit covers Create (whose
// tenant_id is derived from the owning installation via subquery, same
// pattern as KVStoreRepository.Set) and List's two independent knobs: the
// optional installation_id filter and the LIMIT/ordering behavior.
func TestExecutionLog_ListWithInstallationFilterAndLimit(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "ExecutionLog Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenant)

	manifestOneID := testutil.SeedRow(t, pool, "plugin_manifests", map[string]any{
		"slug": "execlog-manifest-one", "name": "ExecLog Manifest One",
	})
	defer testutil.CleanupRow(t, pool, "plugin_manifests", manifestOneID)
	manifestTwoID := testutil.SeedRow(t, pool, "plugin_manifests", map[string]any{
		"slug": "execlog-manifest-two", "name": "ExecLog Manifest Two",
	})
	defer testutil.CleanupRow(t, pool, "plugin_manifests", manifestTwoID)

	installOneID := testutil.SeedRow(t, pool, "plugin_installations", map[string]any{
		"tenant_id": tenant, "manifest_id": manifestOneID, "installed_by": uuid.New(),
	})
	defer testutil.CleanupRow(t, pool, "plugin_installations", installOneID)
	installTwoID := testutil.SeedRow(t, pool, "plugin_installations", map[string]any{
		"tenant_id": tenant, "manifest_id": manifestTwoID, "installed_by": uuid.New(),
	})
	defer testutil.CleanupRow(t, pool, "plugin_installations", installTwoID)

	repo := repository.NewExecutionLogRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	base := time.Now()
	entryOneID := uuid.New()
	logOne := &models.PluginExecutionLog{
		ID: entryOneID, InstallationID: installOneID, HookType: "before_save",
		Module: "crm", EntityType: "contact", DurationMs: 12,
		Status: models.ExecutionStatusSuccess, CreatedAt: base,
	}
	if err := repo.Create(ctx, logOne); err != nil {
		t.Fatalf("create log one: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "plugin_execution_log", entryOneID)

	entryTwoID := uuid.New()
	logTwo := &models.PluginExecutionLog{
		ID: entryTwoID, InstallationID: installTwoID, HookType: "after_save",
		Module: "crm", EntityType: "contact", DurationMs: 30,
		Status: models.ExecutionStatusError, CreatedAt: base.Add(time.Second),
	}
	if err := repo.Create(ctx, logTwo); err != nil {
		t.Fatalf("create log two: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "plugin_execution_log", entryTwoID)

	// Create against an installation the RLS-scoped subquery cannot see (made
	// up id) returns ErrInstallationNotFound instead of silently inserting
	// nothing — same fix and shape as KVStoreRepository.Set for the identical
	// subquery pattern (fix-plugin-kvset-silent-noop-unknown-installation).
	unknownErr := repo.Create(ctx, &models.PluginExecutionLog{
		ID: uuid.New(), InstallationID: uuid.New(), HookType: "x", Module: "y", EntityType: "z", CreatedAt: base,
	})
	if !errors.Is(unknownErr, repository.ErrInstallationNotFound) {
		t.Fatalf("expected ErrInstallationNotFound for an unresolvable installation, got %v", unknownErr)
	}

	all, err := repo.List(ctx, tenant, nil, 0)
	if err != nil || len(all) != 2 {
		t.Fatalf("list all: got %d, err %v", len(all), err)
	}

	filtered, err := repo.List(ctx, tenant, &installOneID, 0)
	if err != nil || len(filtered) != 1 || filtered[0].ID != entryOneID {
		t.Fatalf("list filtered by installation: got %+v, err %v", filtered, err)
	}

	// ORDER BY created_at DESC — the newer entry (logTwo) must win under
	// LIMIT 1.
	limited, err := repo.List(ctx, tenant, nil, 1)
	if err != nil || len(limited) != 1 || limited[0].ID != entryTwoID {
		t.Fatalf("list with limit=1: got %+v, err %v", limited, err)
	}
}

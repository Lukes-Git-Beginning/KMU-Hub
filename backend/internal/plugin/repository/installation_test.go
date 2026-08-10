package repository_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/plugin/repository"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// TestInstallation_Lifecycle covers Create, the (tenant, manifest) unique
// constraint, both lookup paths, status/settings filtering in List, and both
// mutators — the full CRUD surface of InstallationRepository.
func TestInstallation_Lifecycle(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Installation Lifecycle Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenant)

	manifestID := testutil.SeedRow(t, pool, "plugin_manifests", map[string]any{
		"id":   uuid.New(),
		"slug": "installation-lifecycle-manifest",
		"name": "Installation Lifecycle Manifest",
	})
	defer testutil.CleanupRow(t, pool, "plugin_manifests", manifestID)

	repo := repository.NewInstallationRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)
	installerID := uuid.New()

	inst := &models.PluginInstallation{
		ID:          uuid.New(),
		TenantID:    tenant,
		ManifestID:  manifestID,
		Status:      models.InstallationStatusPendingApproval,
		Settings:    json.RawMessage(`{}`),
		InstalledBy: installerID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := repo.Create(ctx, inst); err != nil {
		t.Fatalf("create installation: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "plugin_installations", inst.ID)

	// A second installation for the same (tenant, manifest) pair must be
	// rejected by the UNIQUE(tenant_id, manifest_id) constraint.
	dup := &models.PluginInstallation{
		ID: uuid.New(), TenantID: tenant, ManifestID: manifestID,
		Status: models.InstallationStatusPendingApproval, Settings: json.RawMessage(`{}`),
		InstalledBy: installerID, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := repo.Create(ctx, dup); err == nil {
		testutil.CleanupRow(t, pool, "plugin_installations", dup.ID)
		t.Fatal("created a second installation for the same (tenant, manifest) pair")
	}

	fetched, err := repo.GetByID(ctx, inst.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if fetched == nil || fetched.Status != models.InstallationStatusPendingApproval || fetched.ManifestID != manifestID {
		t.Fatalf("unexpected installation: %+v", fetched)
	}

	if missing, err := repo.GetByID(ctx, uuid.New()); err != nil || missing != nil {
		t.Fatalf("get by unknown id: got %+v, err %v", missing, err)
	}

	byTM, err := repo.GetByTenantAndManifest(ctx, tenant, manifestID)
	if err != nil || byTM == nil || byTM.ID != inst.ID {
		t.Fatalf("get by tenant+manifest: got %+v, err %v", byTM, err)
	}
	if wrongPair, err := repo.GetByTenantAndManifest(ctx, tenant, uuid.New()); err != nil || wrongPair != nil {
		t.Fatalf("get by tenant+unknown manifest: got %+v, err %v", wrongPair, err)
	}

	all, err := repo.List(ctx, tenant, "")
	if err != nil || len(all) != 1 {
		t.Fatalf("list all: got %d, err %v", len(all), err)
	}
	if all[0].ManifestSlug != "installation-lifecycle-manifest" {
		t.Fatalf("list did not join manifest data: %+v", all[0])
	}

	if activeOnly, err := repo.List(ctx, tenant, "active"); err != nil || len(activeOnly) != 0 {
		t.Fatalf("list status=active before activation: got %d, err %v", len(activeOnly), err)
	}

	errMsg := "installation failed"
	if err := repo.UpdateStatus(ctx, inst.ID, models.InstallationStatusError, &errMsg); err != nil {
		t.Fatalf("update status to error: %v", err)
	}
	afterError, err := repo.GetByID(ctx, inst.ID)
	if err != nil || afterError.Status != models.InstallationStatusError || afterError.ErrorMessage == nil || *afterError.ErrorMessage != errMsg {
		t.Fatalf("expected status=error with message, got %+v (err %v)", afterError, err)
	}

	if err := repo.UpdateStatus(ctx, inst.ID, models.InstallationStatusActive, nil); err != nil {
		t.Fatalf("update status to active: %v", err)
	}
	afterActive, err := repo.GetByID(ctx, inst.ID)
	if err != nil || afterActive.Status != models.InstallationStatusActive || afterActive.ErrorMessage != nil {
		t.Fatalf("expected status=active with cleared error, got %+v (err %v)", afterActive, err)
	}

	if activeList, err := repo.List(ctx, tenant, "active"); err != nil || len(activeList) != 1 {
		t.Fatalf("list status=active after activation: got %d, err %v", len(activeList), err)
	}

	newSettings := json.RawMessage(`{"theme":"dark"}`)
	if err := repo.UpdateSettings(ctx, inst.ID, newSettings); err != nil {
		t.Fatalf("update settings: %v", err)
	}
	afterSettings, err := repo.GetByID(ctx, inst.ID)
	if err != nil {
		t.Fatalf("get after update settings: %v", err)
	}
	var gotSettings map[string]any
	if jsonErr := json.Unmarshal(afterSettings.Settings, &gotSettings); jsonErr != nil || gotSettings["theme"] != "dark" {
		t.Fatalf("expected updated settings {theme: dark}, got %s (err %v)", afterSettings.Settings, jsonErr)
	}
}

// TestInstallation_ListActiveByHook proves ListActiveByHook filters on BOTH
// the hook registration match AND installation status — two installations
// share the identical hook registration, only the active one is returned.
func TestInstallation_ListActiveByHook(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Installation Hook Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenant)

	hooks := `[{"hook_type":"before_save","module":"crm","entity_type":"contact","priority":10}]`
	activeManifestID := testutil.SeedRow(t, pool, "plugin_manifests", map[string]any{
		"id": uuid.New(), "slug": "installation-hook-active-manifest", "name": "Active Hook Manifest",
		"hook_registrations": hooks,
	})
	defer testutil.CleanupRow(t, pool, "plugin_manifests", activeManifestID)
	disabledManifestID := testutil.SeedRow(t, pool, "plugin_manifests", map[string]any{
		"id": uuid.New(), "slug": "installation-hook-disabled-manifest", "name": "Disabled Hook Manifest",
		"hook_registrations": hooks,
	})
	defer testutil.CleanupRow(t, pool, "plugin_manifests", disabledManifestID)

	repo := repository.NewInstallationRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	active := &models.PluginInstallation{
		ID: uuid.New(), TenantID: tenant, ManifestID: activeManifestID,
		Status: models.InstallationStatusActive, Settings: json.RawMessage(`{}`),
		InstalledBy: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := repo.Create(ctx, active); err != nil {
		t.Fatalf("create active installation: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "plugin_installations", active.ID)

	disabled := &models.PluginInstallation{
		ID: uuid.New(), TenantID: tenant, ManifestID: disabledManifestID,
		Status: models.InstallationStatusDisabled, Settings: json.RawMessage(`{}`),
		InstalledBy: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := repo.Create(ctx, disabled); err != nil {
		t.Fatalf("create disabled installation: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "plugin_installations", disabled.ID)

	matches, err := repo.ListActiveByHook(ctx, tenant, "before_save", "crm", "contact")
	if err != nil {
		t.Fatalf("list active by hook: %v", err)
	}
	if len(matches) != 1 || matches[0].ID != active.ID {
		t.Fatalf("expected exactly the active installation, got %+v", matches)
	}

	if noMatch, err := repo.ListActiveByHook(ctx, tenant, "after_save", "crm", "contact"); err != nil || len(noMatch) != 0 {
		t.Fatalf("hook type that no manifest registers: got %d, err %v", len(noMatch), err)
	}
}

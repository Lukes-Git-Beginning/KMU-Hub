package repository_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/plugin/repository"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// TestKVStore_IsolatedByInstallation proves entries are keyed by
// installation_id, not just by key name: two installations under the same
// tenant hold different values under the identical key "config:theme" and
// must never see each other's value or count.
func TestKVStore_IsolatedByInstallation(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "KVStore Isolation Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenant)

	manifestOneID := testutil.SeedRow(t, pool, "plugin_manifests", map[string]any{
		"slug": "kv-manifest-one", "name": "KV Manifest One",
	})
	defer testutil.CleanupRow(t, pool, "plugin_manifests", manifestOneID)
	manifestTwoID := testutil.SeedRow(t, pool, "plugin_manifests", map[string]any{
		"slug": "kv-manifest-two", "name": "KV Manifest Two",
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

	repo := repository.NewKVStoreRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	// plugin_kv_store rows cascade-delete with their owning installation
	// (FK ON DELETE CASCADE), so no separate cleanup is needed here.
	if err := repo.Set(ctx, installOneID, "config:theme", json.RawMessage(`"dark"`)); err != nil {
		t.Fatalf("set installation one: %v", err)
	}
	if err := repo.Set(ctx, installTwoID, "config:theme", json.RawMessage(`"light"`)); err != nil {
		t.Fatalf("set installation two: %v", err)
	}

	valOne, ok, err := repo.Get(ctx, installOneID, "config:theme")
	if err != nil || !ok || string(valOne) != `"dark"` {
		t.Fatalf("get installation one: value=%s ok=%v err=%v", valOne, ok, err)
	}
	valTwo, ok, err := repo.Get(ctx, installTwoID, "config:theme")
	if err != nil || !ok || string(valTwo) != `"light"` {
		t.Fatalf("get installation two: value=%s ok=%v err=%v", valTwo, ok, err)
	}

	if _, ok, err := repo.Get(ctx, installOneID, "does-not-exist"); err != nil || ok {
		t.Fatalf("get missing key: ok=%v err=%v", ok, err)
	}

	// Upsert on the same (installation_id, key) overwrites rather than
	// duplicating (ON CONFLICT DO UPDATE).
	if err := repo.Set(ctx, installOneID, "config:theme", json.RawMessage(`"contrast"`)); err != nil {
		t.Fatalf("overwrite installation one: %v", err)
	}
	if updated, _, err := repo.Get(ctx, installOneID, "config:theme"); err != nil || string(updated) != `"contrast"` {
		t.Fatalf("expected overwritten value, got %s (err %v)", updated, err)
	}
	if err := repo.Set(ctx, installOneID, "config:locale", json.RawMessage(`"de-DE"`)); err != nil {
		t.Fatalf("set second key: %v", err)
	}

	all, err := repo.List(ctx, installOneID, "")
	if err != nil || len(all) != 2 {
		t.Fatalf("list all for installation one: got %d, err %v", len(all), err)
	}

	prefixed, err := repo.List(ctx, installOneID, "config:theme")
	if err != nil || len(prefixed) != 1 {
		t.Fatalf("list with prefix filter: got %d, err %v", len(prefixed), err)
	}

	// Installation two must only ever see its own single entry — proves
	// isolation is keyed by installation_id, not merely by key name.
	otherList, err := repo.List(ctx, installTwoID, "")
	if err != nil || len(otherList) != 1 {
		t.Fatalf("list for installation two: got %d, err %v", len(otherList), err)
	}

	if err := repo.Delete(ctx, installOneID, "config:locale"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if afterDelete, err := repo.List(ctx, installOneID, ""); err != nil || len(afterDelete) != 1 {
		t.Fatalf("list after delete: got %d, err %v", len(afterDelete), err)
	}
}

// TestKVStore_Set_UnknownInstallation_ReturnsError: Set derives tenant_id via
// a subquery on plugin_installations (see kv_store.go comment). When the
// installation_id does not resolve under the caller's RLS scope (deleted,
// foreign tenant, or simply made up), the subquery returns zero rows, the
// INSERT affects zero rows, and Set now returns ErrInstallationNotFound
// instead of silently reporting success (fixed in
// fix-plugin-kvset-silent-noop-unknown-installation — previously this
// documented the gap: Set returned nil and the gRPC handler answered
// Success: true even though nothing was stored).
func TestKVStore_Set_UnknownInstallation_ReturnsError(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "KVStore Unknown Installation Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenant)

	repo := repository.NewKVStoreRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	unknownInstallationID := uuid.New()
	err := repo.Set(ctx, unknownInstallationID, "x", json.RawMessage(`1`))
	if !errors.Is(err, repository.ErrInstallationNotFound) {
		t.Fatalf("expected ErrInstallationNotFound for an unresolvable installation, got %v", err)
	}
	if _, ok, err := repo.Get(ctx, unknownInstallationID, "x"); err != nil || ok {
		t.Fatalf("expected no row stored for the unknown installation, got ok=%v err=%v", ok, err)
	}
}

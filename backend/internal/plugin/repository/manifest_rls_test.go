package repository_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/plugin/repository"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// Migration 000276 gave plugin_manifests a nullable tenant_id and the
// asymmetric policy pair roles has carried since 000256: a NULL row is part of
// the catalogue shipped with the product and readable by everyone, a row with a
// tenant belongs to that tenant alone, and writes only ever admit the caller's
// own tenant.
//
// These tests mint their own tenants instead of sharing testutil.TenantA/B:
// two of them create manifests under the same slug, which collides on the
// per-tenant unique index as soon as another test in the package reuses the
// shared fixtures.

func manifestFixture(tenantID *uuid.UUID, slug string) *models.PluginManifest {
	now := time.Now()
	return &models.PluginManifest{
		ID:                uuid.New(),
		TenantID:          tenantID,
		Slug:              slug,
		Name:              slug,
		Version:           "1.0.0",
		SettingsSchema:    json.RawMessage("{}"),
		Permissions:       []string{},
		HookRegistrations: []models.HookRegistration{},
		PluginType:        models.PluginTypeConfig,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

// TestRLS_PluginManifests_TenantSeesOwnAndCatalogueOnly covers both halves of
// the read policy at once: a tenant's own manifest is invisible to another
// tenant, and the catalogue row is visible to both.
func TestRLS_PluginManifests_TenantSeesOwnAndCatalogueOnly(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOne, tenantTwo := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOne, "Manifest RLS Tenant One")
	testutil.EnsureTenant(t, pool, tenantTwo, "Manifest RLS Tenant Two")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOne)
	defer testutil.CleanupRow(t, pool, "tenants", tenantTwo)

	catalogueID := testutil.SeedRow(t, pool, "plugin_manifests", map[string]any{
		"id":   uuid.New(),
		"slug": "rls-test-catalogue-manifest",
		"name": "Catalogue Manifest",
	})
	defer testutil.CleanupRow(t, pool, "plugin_manifests", catalogueID)

	repo := repository.NewManifestRepository(pool)
	ctxOne := testutil.WithTenantCtx(context.Background(), tenantOne)
	ctxTwo := testutil.WithTenantCtx(context.Background(), tenantTwo)

	own := manifestFixture(&tenantOne, "rls-test-manifest-tenant-one")
	if err := repo.Create(ctxOne, own); err != nil {
		t.Fatalf("create own manifest: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "plugin_manifests", own.ID)

	testutil.AssertRowCount(t, pool, ctxOne, "plugin_manifests", own.ID, 1)
	testutil.AssertRowCount(t, pool, ctxTwo, "plugin_manifests", own.ID, 0)

	testutil.AssertRowCount(t, pool, ctxOne, "plugin_manifests", catalogueID, 1)
	testutil.AssertRowCount(t, pool, ctxTwo, "plugin_manifests", catalogueID, 1)

	// The repository's own read path has to agree with the raw counts — a
	// SELECT that forgot the tenant column would still pass AssertRowCount.
	fetched, err := repo.GetByID(ctxOne, own.ID)
	if err != nil {
		t.Fatalf("get own manifest: %v", err)
	}
	if fetched == nil || fetched.TenantID == nil || *fetched.TenantID != tenantOne {
		t.Fatalf("expected manifest owned by %s, got %+v", tenantOne, fetched)
	}
	if foreign, ferr := repo.GetByID(ctxTwo, own.ID); ferr != nil || foreign != nil {
		t.Fatalf("foreign tenant read own manifest: %+v (err %v)", foreign, ferr)
	}
	catalogue, err := repo.GetByID(ctxTwo, catalogueID)
	if err != nil || catalogue == nil {
		t.Fatalf("catalogue manifest not readable by tenant two: %+v (err %v)", catalogue, err)
	}
	if catalogue.TenantID != nil {
		t.Fatalf("catalogue manifest carries a tenant: %s", *catalogue.TenantID)
	}
}

// TestPluginManifests_CreateForForeignTenant_Rejected exercises the real write
// path: stamping another tenant's id on a manifest must fail the policy's
// WITH CHECK rather than land in that tenant's catalogue.
func TestPluginManifests_CreateForForeignTenant_Rejected(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOne, tenantTwo := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOne, "Manifest Write Tenant One")
	testutil.EnsureTenant(t, pool, tenantTwo, "Manifest Write Tenant Two")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOne)
	defer testutil.CleanupRow(t, pool, "tenants", tenantTwo)

	repo := repository.NewManifestRepository(pool)
	foreign := manifestFixture(&tenantTwo, "rls-test-manifest-foreign")

	err := repo.Create(testutil.WithTenantCtx(context.Background(), tenantOne), foreign)
	if err == nil {
		testutil.CleanupRow(t, pool, "plugin_manifests", foreign.ID)
		t.Fatal("created a manifest for a foreign tenant; the write policy is not in force")
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "42501" {
		t.Fatalf("expected SQLSTATE 42501, got: %v", err)
	}

	// A NULL tenant_id — a catalogue row — is equally out of reach over the
	// API: only migrations write those.
	catalogue := manifestFixture(nil, "rls-test-manifest-catalogue-attempt")
	err = repo.Create(testutil.WithTenantCtx(context.Background(), tenantOne), catalogue)
	if err == nil {
		testutil.CleanupRow(t, pool, "plugin_manifests", catalogue.ID)
		t.Fatal("a tenant created a catalogue manifest; the write policy admits NULL tenant_id")
	}
}

// TestPluginManifests_SameSlugAcrossTenants_Allowed pins the reason the unique
// constraint moved to an expression index: the slug is a per-tenant name, and
// GetBySlug must still answer with the caller's own manifest.
func TestPluginManifests_SameSlugAcrossTenants_Allowed(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOne, tenantTwo := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOne, "Manifest Slug Tenant One")
	testutil.EnsureTenant(t, pool, tenantTwo, "Manifest Slug Tenant Two")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOne)
	defer testutil.CleanupRow(t, pool, "tenants", tenantTwo)

	const slug = "rls-test-shared-slug"
	repo := repository.NewManifestRepository(pool)

	one := manifestFixture(&tenantOne, slug)
	ctxOne := testutil.WithTenantCtx(context.Background(), tenantOne)
	if err := repo.Create(ctxOne, one); err != nil {
		t.Fatalf("create manifest for tenant one: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "plugin_manifests", one.ID)

	two := manifestFixture(&tenantTwo, slug)
	ctxTwo := testutil.WithTenantCtx(context.Background(), tenantTwo)
	if err := repo.Create(ctxTwo, two); err != nil {
		t.Fatalf("second tenant could not reuse the slug: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "plugin_manifests", two.ID)

	found, err := repo.GetBySlug(ctxOne, slug)
	if err != nil {
		t.Fatalf("get by slug: %v", err)
	}
	if found == nil || found.ID != one.ID {
		t.Fatalf("GetBySlug returned %+v; want the caller's own manifest %s", found, one.ID)
	}
}

// TestPluginManifests_DeleteCatalogue_SilentlyMatchesNothing is why
// Service.DeleteManifest refuses catalogue manifests up front instead of
// leaving the policy to it: the DELETE itself succeeds and removes nothing, so
// the caller would be told the manifest is gone while it is still there.
func TestPluginManifests_DeleteCatalogue_SilentlyMatchesNothing(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOne := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOne, "Manifest Delete Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOne)

	catalogueID := testutil.SeedRow(t, pool, "plugin_manifests", map[string]any{
		"id":   uuid.New(),
		"slug": "rls-test-catalogue-undeletable",
		"name": "Catalogue Manifest",
	})
	defer testutil.CleanupRow(t, pool, "plugin_manifests", catalogueID)

	repo := repository.NewManifestRepository(pool)
	ctxOne := testutil.WithTenantCtx(context.Background(), tenantOne)
	if err := repo.Delete(ctxOne, catalogueID); err != nil {
		t.Fatalf("delete returned an error rather than matching no rows: %v", err)
	}

	testutil.AssertRowCount(t, pool, testutil.WithSystemCtx(context.Background()),
		"plugin_manifests", catalogueID, 1)
}

package customfield

// wp-crm-meta: closes the write-surface gap for custom_field_definitions the
// same way wp-crm-core did for the core CRM entities — the package had no
// RLS test at all beforehand.

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestCustomFieldWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "CustomField Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "CustomField Write Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("customfield-write-%s@tenantown.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())
	now := time.Now().UTC()

	field := &models.CustomFieldDefinition{
		ID:         uuid.New(),
		TenantID:   tenantOwn,
		EntityType: models.EntityType("contact"),
		FieldName:  "write_test_" + uuid.New().String()[:8],
		FieldLabel: "Write Test",
		FieldType:  models.FieldType("text"),
		CreatedBy:  userID,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// Create with the victim tenant's real TenantID, called from the attacker
	// session — only RLS's WITH CHECK can be stopping this insert.
	if err := repo.Create(ctxOther, field); err == nil {
		t.Fatalf("Create (foreign ctx): expected an RLS error, got nil")
	}
	testutil.AssertRowCount(t, pool, sysCtx, "custom_field_definitions", field.ID, 0)

	if err := repo.Create(ctxOwn, field); err != nil {
		t.Fatalf("Create (own ctx): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "custom_field_definitions", field.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "custom_field_definitions", field.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "custom_field_definitions", field.ID, 0)

	// Update carries an explicit tenant_id predicate (field.TenantID) and
	// treats zero affected rows as not-found.
	foreign := *field
	foreign.FieldLabel = "Hacked"
	foreign.UpdatedAt = time.Now().UTC()
	if err := repo.Update(ctxOther, &foreign); !errors.Is(err, ErrFieldNotFound) {
		t.Fatalf("Update (foreign ctx): expected ErrFieldNotFound, got %v", err)
	}
	got, err := repo.GetByID(ctxOwn, field.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.FieldLabel != field.FieldLabel {
		t.Fatalf("a foreign-tenant write reached the custom field: label=%q", got.FieldLabel)
	}

	foreign.FieldLabel = "Renamed-" + uuid.New().String()[:8]
	if err := repo.Update(ctxOwn, &foreign); err != nil {
		t.Fatalf("Update (own ctx): %v", err)
	}
	got, err = repo.GetByID(ctxOwn, field.ID, tenantOwn)
	if err != nil {
		t.Fatalf("GetByID (own ctx): %v", err)
	}
	if got.FieldLabel != foreign.FieldLabel {
		t.Fatalf("own-tenant write did not land: label=%q", got.FieldLabel)
	}

	// Delete carries the same explicit predicate + not-found-on-zero-rows.
	if err := repo.Delete(ctxOther, field.ID, tenantOwn); !errors.Is(err, ErrFieldNotFound) {
		t.Fatalf("Delete (foreign ctx): expected ErrFieldNotFound, got %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "custom_field_definitions", field.ID, 1)

	if err := repo.Delete(ctxOwn, field.ID, tenantOwn); err != nil {
		t.Fatalf("Delete (own ctx): %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "custom_field_definitions", field.ID, 0)
}

// TestCustomFieldNames_UniquePerTenantNotGlobally is a regression test for a
// pre-existing bug found while writing the write-surface test above:
// idx_custom_field_definitions_entity_name (migration 000005) was never
// re-scoped by tenant_id when tenant_id was retrofitted (migration 000106),
// so a field_name was unique per entity_type GLOBALLY — the second tenant to
// define e.g. a "budget" custom field on contacts got a raw
// unique-violation 500. Fixed in migration 000255
// (tenant_id, entity_type, field_name).
func TestCustomFieldNames_UniquePerTenantNotGlobally(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "CustomField Unique Tenant A")
	testutil.EnsureTenant(t, pool, tenantB, "CustomField Unique Tenant B")
	defer testutil.CleanupRow(t, pool, "tenants", tenantA)
	defer testutil.CleanupRow(t, pool, "tenants", tenantB)

	userA := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantA,
		"email":         fmt.Sprintf("customfield-unique-a-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userA)
	userB := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantB,
		"email":         fmt.Sprintf("customfield-unique-b-%s@tenantb.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userB)

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)
	now := time.Now().UTC()
	fieldName := "budget_" + uuid.New().String()[:8]

	fieldA := &models.CustomFieldDefinition{
		ID:         uuid.New(),
		TenantID:   tenantA,
		EntityType: models.EntityType("contact"),
		FieldName:  fieldName,
		FieldLabel: "Budget",
		FieldType:  models.FieldType("text"),
		CreatedBy:  userA,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := repo.Create(ctxA, fieldA); err != nil {
		t.Fatalf("Create tenant A: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "custom_field_definitions", fieldA.ID)

	fieldB := &models.CustomFieldDefinition{
		ID:         uuid.New(),
		TenantID:   tenantB,
		EntityType: models.EntityType("contact"),
		FieldName:  fieldName,
		FieldLabel: "Budget",
		FieldType:  models.FieldType("text"),
		CreatedBy:  userB,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := repo.Create(ctxB, fieldB); err != nil {
		t.Fatalf("Create tenant B with the same field_name: %v (idx_custom_field_definitions_entity_name is not tenant-scoped)", err)
	}
	defer testutil.CleanupRow(t, pool, "custom_field_definitions", fieldB.ID)
}

package repository

// Cross-tenant isolation for validation_rules and workflow_rules after
// migration 000269 put both under RLS.
//
// These tests deliberately go through the real repository methods rather than
// testutil.SeedRow: the point is not that a hand-written INSERT respects the
// policy, it is that GetByID, Update and Delete — all of which run
// `WHERE id = $1` with no tenant predicate — stop reaching another tenant's
// row. That was the actual hole; a seed-based test would never have shown it.
//
// Each test mints its own tenants instead of reusing testutil.TenantA/B, which
// several packages seed in parallel.

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// twoTenants mints and registers two fresh tenants for a single test.
func twoTenants(t *testing.T, pool *pgxpool.Pool) (uuid.UUID, uuid.UUID) {
	t.Helper()
	owner, other := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, owner, "rules-isolation owner")
	testutil.EnsureTenant(t, pool, other, "rules-isolation other")
	return owner, other
}

func TestTenantIsolation_ValidationRules_DB(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	owner, other := twoTenants(t, pool)
	repo := NewValidationRuleRepository(pool)

	now := time.Now().UTC()
	rule := &models.ValidationRule{
		ID:           uuid.New(),
		TenantID:     owner,
		Name:         "email format",
		Description:  "contact email must look like an address",
		EntityType:   "contact",
		FieldName:    "email",
		RuleType:     models.ValidationRuleType("format"),
		RuleConfig:   json.RawMessage(`{}`),
		ErrorMessage: "email is malformed",
		Priority:     10,
		Enabled:      true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	ownerCtx := testutil.WithTenantCtx(context.Background(), owner)
	otherCtx := testutil.WithTenantCtx(context.Background(), other)

	if err := repo.Create(ownerCtx, rule); err != nil {
		t.Fatalf("create as owner: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "validation_rules", rule.ID)

	// Owner sees its own row through the unscoped lookup.
	got, err := repo.GetByID(ownerCtx, rule.ID)
	if err != nil {
		t.Fatalf("get as owner: %v", err)
	}
	if got == nil {
		t.Fatal("owner cannot read back its own validation rule")
	}

	// The other tenant does not, even holding the exact UUID.
	foreign, err := repo.GetByID(otherCtx, rule.ID)
	if err != nil {
		t.Fatalf("get as other tenant: %v", err)
	}
	if foreign != nil {
		t.Fatalf("cross-tenant read leaked validation rule %s", rule.ID)
	}

	// Nor can it delete it — Delete has no tenant predicate, so without RLS
	// this call would have removed another tenant's rule.
	if err := repo.Delete(otherCtx, rule.ID); err != nil {
		t.Fatalf("cross-tenant delete errored: %v", err)
	}
	testutil.AssertRowCount(t, pool, ownerCtx, "validation_rules", rule.ID, 1)

	// A write naming a foreign tenant is rejected by the policy's WITH CHECK
	// rather than silently landing in the other tenant's data.
	stray := *rule
	stray.ID = uuid.New()
	stray.TenantID = other
	if err := repo.Create(ownerCtx, &stray); err == nil {
		testutil.CleanupRow(t, pool, "validation_rules", stray.ID)
		t.Fatal("insert naming a foreign tenant succeeded, expected RLS rejection")
	}
}

func TestTenantIsolation_WorkflowRules_DB(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	owner, other := twoTenants(t, pool)
	repo := NewWorkflowRuleRepository(pool)

	now := time.Now().UTC()
	rule := &models.WorkflowRule{
		ID:           uuid.New(),
		TenantID:     owner,
		Name:         "notify on deal won",
		Description:  "send a mail when a deal closes",
		TriggerEvent: "deal.won",
		Conditions:   json.RawMessage(`{}`),
		Actions:      json.RawMessage(`[]`),
		Priority:     5,
		Enabled:      true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	ownerCtx := testutil.WithTenantCtx(context.Background(), owner)
	otherCtx := testutil.WithTenantCtx(context.Background(), other)

	if err := repo.Create(ownerCtx, rule); err != nil {
		t.Fatalf("create as owner: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "workflow_rules", rule.ID)

	got, err := repo.GetByID(ownerCtx, rule.ID)
	if err != nil {
		t.Fatalf("get as owner: %v", err)
	}
	if got == nil {
		t.Fatal("owner cannot read back its own workflow rule")
	}

	foreign, err := repo.GetByID(otherCtx, rule.ID)
	if err != nil {
		t.Fatalf("get as other tenant: %v", err)
	}
	if foreign != nil {
		t.Fatalf("cross-tenant read leaked workflow rule %s", rule.ID)
	}

	if err := repo.Delete(otherCtx, rule.ID); err != nil {
		t.Fatalf("cross-tenant delete errored: %v", err)
	}
	testutil.AssertRowCount(t, pool, ownerCtx, "workflow_rules", rule.ID, 1)
}

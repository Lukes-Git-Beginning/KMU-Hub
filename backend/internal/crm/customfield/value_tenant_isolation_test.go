package customfield_test

// Cross-tenant isolation for the four CRM custom-field value tables after
// migration 000270 put them under RLS via enable_tenant_rls_via_join().
//
// The test lives in an external test package because it drives the contact,
// company, deal and activity repositories, which the customfield package
// itself does not import.
//
// It goes through the real repository methods rather than testutil.SeedRow for
// a specific reason: those methods filter on the parent id alone
// (`WHERE cfv.contact_id = $1`) with no tenant predicate anywhere. That is the
// gap migration 000270 closes, and a seed-based test would only prove that a
// hand-written INSERT respects the policy — not that the unscoped read and the
// unscoped upsert stopped reaching across tenants.

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/crm/activity"
	"github.com/kmuhub/kmuhub/internal/crm/company"
	"github.com/kmuhub/kmuhub/internal/crm/contact"
	"github.com/kmuhub/kmuhub/internal/crm/deal"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// valueRepo is the slice of each entity repository this test needs. All four
// expose the same two method signatures, so one table-driven case covers them.
type valueRepo struct {
	get func(ctx context.Context, parentID uuid.UUID) ([]*models.CustomFieldValueRow, error)
	set func(ctx context.Context, parentID uuid.UUID, values map[uuid.UUID]any) error
}

type valueCase struct {
	name       string
	table      string // child table, for the raw visibility counts
	parentCol  string // its FK column back to the parent entity
	entityType string // custom_field_definitions.entity_type
	// seedParent creates the parent entity for the given tenant and returns its
	// id, registering its own cleanup.
	seedParent func(t *testing.T, pool *pgxpool.Pool, tenantID, userID uuid.UUID) uuid.UUID
	newRepo    func(pool *pgxpool.Pool) valueRepo
}

func valueCases() []valueCase {
	return []valueCase{
		{
			name:       "contact",
			table:      "contact_custom_field_values",
			parentCol:  "contact_id",
			entityType: "contact",
			seedParent: func(t *testing.T, pool *pgxpool.Pool, tenantID, userID uuid.UUID) uuid.UUID {
				t.Helper()
				id := testutil.SeedRow(t, pool, "contacts", map[string]any{
					"tenant_id":  tenantID,
					"first_name": "Custom",
					"last_name":  "Field",
					"created_by": userID,
				})
				t.Cleanup(func() { testutil.CleanupRow(t, pool, "contacts", id) })
				return id
			},
			newRepo: func(pool *pgxpool.Pool) valueRepo {
				r := contact.NewPostgresRepository(pool)
				return valueRepo{get: r.GetCustomFieldValues, set: r.SetCustomFieldValues}
			},
		},
		{
			name:       "company",
			table:      "company_custom_field_values",
			parentCol:  "company_id",
			entityType: "company",
			seedParent: func(t *testing.T, pool *pgxpool.Pool, tenantID, userID uuid.UUID) uuid.UUID {
				t.Helper()
				id := testutil.SeedRow(t, pool, "companies", map[string]any{
					"tenant_id":  tenantID,
					"name":       fmt.Sprintf("CFV Co %s", uuid.New().String()[:8]),
					"created_by": userID,
				})
				t.Cleanup(func() { testutil.CleanupRow(t, pool, "companies", id) })
				return id
			},
			newRepo: func(pool *pgxpool.Pool) valueRepo {
				r := company.NewPostgresRepository(pool)
				return valueRepo{get: r.GetCustomFieldValues, set: r.SetCustomFieldValues}
			},
		},
		{
			name:       "deal",
			table:      "deal_custom_field_values",
			parentCol:  "deal_id",
			entityType: "deal",
			seedParent: func(t *testing.T, pool *pgxpool.Pool, tenantID, userID uuid.UUID) uuid.UUID {
				t.Helper()
				stageID := testutil.SeedRow(t, pool, "pipeline_stages", map[string]any{
					"tenant_id": tenantID,
					"name":      fmt.Sprintf("CFV Stage %s", uuid.New().String()[:8]),
				})
				t.Cleanup(func() { testutil.CleanupRow(t, pool, "pipeline_stages", stageID) })

				id := testutil.SeedRow(t, pool, "deals", map[string]any{
					"tenant_id":  tenantID,
					"name":       fmt.Sprintf("CFV Deal %s", uuid.New().String()[:8]),
					"stage_id":   stageID,
					"created_by": userID,
				})
				t.Cleanup(func() { testutil.CleanupRow(t, pool, "deals", id) })
				return id
			},
			newRepo: func(pool *pgxpool.Pool) valueRepo {
				r := deal.NewPostgresRepository(pool)
				return valueRepo{get: r.GetCustomFieldValues, set: r.SetCustomFieldValues}
			},
		},
		{
			name:       "activity",
			table:      "activity_custom_field_values",
			parentCol:  "activity_id",
			entityType: "activity",
			seedParent: func(t *testing.T, pool *pgxpool.Pool, tenantID, userID uuid.UUID) uuid.UUID {
				t.Helper()
				id := testutil.SeedRow(t, pool, "activities", map[string]any{
					"tenant_id":     tenantID,
					"activity_type": "note",
					"subject":       "CFV activity",
					"created_by":    userID,
				})
				t.Cleanup(func() { testutil.CleanupRow(t, pool, "activities", id) })
				return id
			},
			newRepo: func(pool *pgxpool.Pool) valueRepo {
				r := activity.NewPostgresRepository(pool)
				return valueRepo{get: r.GetCustomFieldValues, set: r.SetCustomFieldValues}
			},
		},
	}
}

// countValues counts rows of a child table under the visibility of ctx. The
// value tables have a composite primary key and no id column, so
// testutil.AssertRowCount does not apply. table and parentCol come from the
// case table above, never from input.
func countValues(t *testing.T, pool *pgxpool.Pool, ctx context.Context, table, parentCol string, parentID uuid.UUID) int {
	t.Helper()
	var n int
	query := fmt.Sprintf("SELECT count(*) FROM %s WHERE %s = $1", table, parentCol)
	if err := pool.QueryRow(ctx, query, parentID).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

func TestTenantIsolation_CRMCustomFieldValues_DB(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	// Registered before any fixture so it runs last: t.Cleanup is LIFO and the
	// fixture cleanups still need a live pool. A plain `defer pool.Close()`
	// would close it first and leave the whole FK chain in the database.
	t.Cleanup(func() { pool.Close() })

	owner, other := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, owner, "custom field owner")
	testutil.EnsureTenant(t, pool, other, "custom field other")

	ownerCtx := testutil.WithTenantCtx(context.Background(), owner)
	otherCtx := testutil.WithTenantCtx(context.Background(), other)

	ownerUser := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     owner,
		"email":         fmt.Sprintf("cfv-owner-%s@example.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", ownerUser) })

	for _, tc := range valueCases() {
		t.Run(tc.name, func(t *testing.T) {
			parentID := tc.seedParent(t, pool, owner, ownerUser)

			fieldID := testutil.SeedRow(t, pool, "custom_field_definitions", map[string]any{
				"tenant_id":   owner,
				"entity_type": tc.entityType,
				"field_name":  fmt.Sprintf("cfv_%s", uuid.New().String()[:8]),
				"field_label": "CFV probe",
				"field_type":  "text",
				"created_by":  ownerUser,
			})
			t.Cleanup(func() { testutil.CleanupRow(t, pool, "custom_field_definitions", fieldID) })

			repo := tc.newRepo(pool)

			if err := repo.set(ownerCtx, parentID, map[uuid.UUID]any{fieldID: "owner value"}); err != nil {
				t.Fatalf("set as owner: %v", err)
			}

			// Visible to its own tenant...
			if n := countValues(t, pool, ownerCtx, tc.table, tc.parentCol, parentID); n != 1 {
				t.Fatalf("owner expected 1 row in %s, got %d", tc.table, n)
			}
			// ...and to nobody else.
			if n := countValues(t, pool, otherCtx, tc.table, tc.parentCol, parentID); n != 0 {
				t.Fatalf("cross-tenant read of %s leaked %d row(s)", tc.table, n)
			}

			// The repository read is unscoped by design — it filters on the
			// parent id only. Before 000270 the foreign tenant got the value
			// back simply by knowing the parent UUID.
			ownerValues, err := repo.get(ownerCtx, parentID)
			if err != nil {
				t.Fatalf("get as owner: %v", err)
			}
			if len(ownerValues) != 1 || ownerValues[0].Value != "owner value" {
				t.Fatalf("owner expected its own value back, got %+v", ownerValues)
			}

			foreignValues, err := repo.get(otherCtx, parentID)
			if err != nil {
				t.Fatalf("get as foreign tenant: %v", err)
			}
			if len(foreignValues) != 0 {
				t.Fatalf("cross-tenant read via repository leaked %d value(s)", len(foreignValues))
			}

			// The upsert is unscoped the same way. Whether the policy rejects
			// it outright or the statement matches nothing, the owner's value
			// must survive — that is the property worth asserting, and it does
			// not depend on which error Postgres picks for a conflict against
			// an invisible row.
			_ = repo.set(otherCtx, parentID, map[uuid.UUID]any{fieldID: "hijacked"})

			after, err := repo.get(ownerCtx, parentID)
			if err != nil {
				t.Fatalf("get as owner after foreign write: %v", err)
			}
			if len(after) != 1 || after[0].Value != "owner value" {
				t.Fatalf("foreign write changed the owner's data: %+v", after)
			}
		})
	}
}

// TestMergeInto_CarriesCustomFieldValues_DB guards the one path migration
// 000270 could plausibly have broken: contact MergeInto copies custom field
// values with `INSERT INTO contact_custom_field_values ... SELECT ... FROM
// contact_custom_field_values` inside a transaction, so the new policy now
// filters both the source read and the target write. Both contacts belong to
// the same tenant, so it must still copy — a silently empty merge would be the
// phantom-404 shape this repository has produced before.
func TestMergeInto_CarriesCustomFieldValues_DB(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "custom field merge")
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	user := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenant,
		"email":         fmt.Sprintf("cfv-merge-%s@example.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", user) })

	newContact := func(first string) uuid.UUID {
		id := testutil.SeedRow(t, pool, "contacts", map[string]any{
			"tenant_id":  tenant,
			"first_name": first,
			"last_name":  "Merge",
			"created_by": user,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "contacts", id) })
		return id
	}
	primaryID, duplicateID := newContact("Primary"), newContact("Duplicate")

	fieldID := testutil.SeedRow(t, pool, "custom_field_definitions", map[string]any{
		"tenant_id":   tenant,
		"entity_type": "contact",
		"field_name":  fmt.Sprintf("cfv_%s", uuid.New().String()[:8]),
		"field_label": "CFV merge probe",
		"field_type":  "text",
		"created_by":  user,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "custom_field_definitions", fieldID) })

	repo := contact.NewPostgresRepository(pool)
	if err := repo.SetCustomFieldValues(ctx, duplicateID, map[uuid.UUID]any{fieldID: "carried over"}); err != nil {
		t.Fatalf("set on duplicate: %v", err)
	}

	if err := repo.MergeInto(ctx, primaryID, duplicateID, tenant); err != nil {
		t.Fatalf("merge: %v", err)
	}

	merged, err := repo.GetCustomFieldValues(ctx, primaryID)
	if err != nil {
		t.Fatalf("read primary after merge: %v", err)
	}
	if len(merged) != 1 || merged[0].Value != "carried over" {
		t.Fatalf("merge dropped the custom field value under RLS: %+v", merged)
	}
}

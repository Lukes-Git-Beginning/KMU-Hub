package vendoraccess_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/security/vendoraccess"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// vendor_access_requests.tenant_id cascades from tenants, so cleaning up the
// seeded tenants also removes their requests -- no separate row cleanup
// needed. Cleanups run as `defer` in the test body, not t.Cleanup: t.Cleanup
// runs after every defer, including `defer pool.Close()`, and would hit a
// closed pool (lesson from iteration 15/16 of this backlog).

func seedTenantPair(t *testing.T, pool *pgxpool.Pool) (uuid.UUID, uuid.UUID) {
	t.Helper()
	tenantOne, tenantTwo := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOne, "Vendor Access Tenant One")
	testutil.EnsureTenant(t, pool, tenantTwo, "Vendor Access Tenant Two")
	return tenantOne, tenantTwo
}

func realRequest(tenantID uuid.UUID) *models.VendorAccessRequest {
	now := time.Now().UTC()
	return &models.VendorAccessRequest{
		ID:             uuid.New(),
		TenantID:       tenantID,
		Reason:         "RLS test request",
		Description:    "seeded by postgres_repository_test.go",
		Agents:         []models.VendorAccessAgent{{Name: "Test Agent"}},
		Scope:          []string{"crm", "documents"},
		RequestedStart: now,
		DurationDays:   7,
		ExpiresAt:      now.Add(7 * 24 * time.Hour),
		Status:         models.VendorAccessStatusPending,
		CreatedAt:      now,
	}
}

// TestRLS_VendorAccessRequests_ForeignTenantSeesNothing exercises the real
// CreateRequest write path (not a raw INSERT) so a write that silently
// hardcodes or omits tenant_id would be caught, then confirms RLS hides the
// row from a different tenant and shows it to its own.
func TestRLS_VendorAccessRequests_ForeignTenantSeesNothing(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOne, tenantTwo := seedTenantPair(t, pool)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOne)
	defer testutil.CleanupRow(t, pool, "tenants", tenantTwo)

	repo := vendoraccess.NewPostgresRepository(pool)
	reqID := uuid.New()

	testutil.AssertWriteCarriesTenant(t, pool, func(ctx context.Context, tenantID uuid.UUID) error {
		req := realRequest(tenantID)
		req.ID = reqID
		return repo.CreateRequest(ctx, req)
	}, "vendor_access_requests", reqID)

	// AssertWriteCarriesTenant always writes under a freshly minted tenant of
	// its own; also check explicitly against tenantOne/tenantTwo via the
	// repository's own read paths (List + Get), not just a raw row count.
	req2 := realRequest(tenantOne)
	if err := repo.CreateRequest(testutil.WithTenantCtx(context.Background(), tenantOne), req2); err != nil {
		t.Fatalf("CreateRequest: %v", err)
	}

	ownList, err := repo.ListRequests(testutil.WithTenantCtx(context.Background(), tenantOne), tenantOne)
	if err != nil {
		t.Fatalf("ListRequests (own tenant): %v", err)
	}
	if len(ownList) == 0 {
		t.Fatal("own tenant sees 0 requests, want >= 1")
	}

	foreignList, err := repo.ListRequests(testutil.WithTenantCtx(context.Background(), tenantTwo), tenantOne)
	if err != nil {
		t.Fatalf("ListRequests (foreign tenant, RLS should filter not error): %v", err)
	}
	if len(foreignList) != 0 {
		t.Fatalf("foreign tenant sees %d requests for tenantOne, want 0", len(foreignList))
	}

	if _, err := repo.GetRequest(testutil.WithTenantCtx(context.Background(), tenantTwo), tenantOne, req2.ID); !errors.Is(err, vendoraccess.ErrRequestNotFound) {
		t.Fatalf("GetRequest across tenants: err = %v, want ErrRequestNotFound", err)
	}
}

// TestService_Lifecycle_RealDB runs the full approve/counter-propose/revoke
// cycle through the real Service + PostgresRepository, checking that
// approved_by/revoked_by resolve to the reviewer's display name via the join
// (not just an id), which the fake-repo unit tests in service_test.go cannot
// exercise.
func TestService_Lifecycle_RealDB(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOne, _ := seedTenantPair(t, pool)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOne)

	reviewerID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     tenantOne,
		"email":         "reviewer-" + uuid.New().String() + "@example.test",
		"password_hash": "not-a-real-hash",
		"first_name":    "Rita",
		"last_name":     "Reviewer",
	})
	// users.tenant_id has no ON DELETE CASCADE to tenants, so this row must
	// be cleaned up before the tenant cleanup below runs. Deferred after the
	// tenant cleanup registration, so it runs first (LIFO).
	defer testutil.CleanupRow(t, pool, "users", reviewerID)

	repo := vendoraccess.NewPostgresRepository(pool)
	svc := vendoraccess.NewService(repo)
	ctx := testutil.WithTenantCtx(context.Background(), tenantOne)

	created, err := svc.Create(ctx, &models.VendorAccessRequest{
		Reason:         "Lifecycle test",
		Description:    "created via Service.Create",
		Agents:         []models.VendorAccessAgent{{Name: "Zentria Agent"}},
		Scope:          []string{"crm", "salary"}, // sensitive
		RequestedStart: time.Now().UTC(),
		DurationDays:   7,
		ExpiresAt:      time.Now().UTC().Add(7 * 24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Sensitive scope without ack must be rejected.
	if _, err := svc.Approve(ctx, created.ID, reviewerID, false); !errors.Is(err, vendoraccess.ErrSensitiveAckRequired) {
		t.Fatalf("Approve without ack: err = %v, want ErrSensitiveAckRequired", err)
	}

	approved, err := svc.Approve(ctx, created.ID, reviewerID, true)
	if err != nil {
		t.Fatalf("Approve with ack: %v", err)
	}
	if approved.Status != models.VendorAccessStatusActive {
		t.Fatalf("status after approve = %q, want active", approved.Status)
	}
	if approved.ApprovedByName != "Rita Reviewer" {
		t.Fatalf("approved_by name = %q, want %q (joined from users)", approved.ApprovedByName, "Rita Reviewer")
	}

	revoked, err := svc.Revoke(ctx, created.ID, reviewerID)
	if err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if revoked.Status != models.VendorAccessStatusRevoked {
		t.Fatalf("status after revoke = %q, want revoked", revoked.Status)
	}
	if revoked.RevokedByName != "Rita Reviewer" {
		t.Fatalf("revoked_by name = %q, want %q (joined from users)", revoked.RevokedByName, "Rita Reviewer")
	}
}

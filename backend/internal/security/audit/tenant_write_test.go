package audit_test

// The RLS tests in rls_test.go seed rows via testutil.SeedRow (raw SQL with
// tenant_id set directly) and never exercise the real PostgresRepository.Create
// path — the hash-chain write with the advisory lock. This test goes through
// the actual repo instead, using the new testutil.AssertWriteCarriesTenant
// helper (built in this same unit) so the write and both read-back checks are
// one call instead of hand-rolled per package.

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/security/audit"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestAuditWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	repo := audit.NewPostgresRepository(pool)
	id := uuid.New()

	testutil.AssertWriteCarriesTenant(t, pool, func(ctx context.Context, tenantID uuid.UUID) error {
		entry := &models.AuditEntry{
			ID:       id,
			TenantID: tenantID,
			Action:   fmt.Sprintf("tenant-write-test-%s", id),
			Result:   audit.ResultSuccess,
			Details:  "{}",
		}
		return repo.Create(ctx, entry)
	}, "audit_log", id)
}

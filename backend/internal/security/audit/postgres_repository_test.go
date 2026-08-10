package audit_test

// List() and VerifyChain() both Scan the raw ip_address column into
// models.AuditEntry.IPAddress (a plain string, not *string):
//
//	SELECT ..., ip_address, ... FROM audit_log ...
//	rows.Scan(..., &e.IPAddress, ...)
//
// ip_address is INET. pgx v5 cannot decode an INET value -- NULL or set --
// into a string destination; both cases were reproduced directly against
// the local Postgres while writing this test ("cannot scan NULL into
// *string" for an empty IP, "cannot scan inet (OID 869) in binary format
// into *string" for a real one). Every other repository in this codebase
// that reads ip_address casts it first -- internal/auth/postgres_repository.go
// uses COALESCE(host(ip_address), ''), internal/crm/consent/postgres_repository.go
// and internal/formulare/postgres_repository.go use ip_address::text.
// audit/postgres_repository.go (List at line ~170, VerifyChain at line
// ~224) is missing that cast, so both functions error on every call that
// would return at least one row -- unconditionally, regardless of whether
// that row happens to carry a real IP or none. Verified by temporarily
// adding the missing cast locally: the two integration tests below then
// pass with their full intended assertions (filter matching, pagination
// defaults, chain verification); reverted, since fixing this is a behavior
// change out of scope for a coverage-only unit. Flagged for a Block A
// fix-unit in the next run -- this likely means the audit log viewer,
// export, and VerifyAuditChain RPC have never worked against a tenant with
// any recorded audit activity.
//
// Consequently this file can only pin the bug (TestPostgresRepository_
// List_IPAddressScanBug) rather than exercise the filter/pagination logic
// the unit originally set out to cover, and does not attempt a VerifyChain
// integration test at all: VerifyChain has no tenant scope by design (it
// walks the one global chain), and this shared dev database's global chain
// already contains links seeded by rls_test.go via testutil.SeedRow with
// hardcoded dummy previous_hash/entry_hash values that don't reflect the
// real preceding row -- so even a "verify an intact range" test cannot be
// made reliably green here without picking apart pre-existing, unrelated
// chain state. GetLastHash is unaffected (it selects only entry_hash) and
// is covered on its own below.
//
// GetLastHash's "no rows" branch (returns "") is also not exercised: audit_log
// is a single global table, not just non-empty but structurally impossible to
// make empty for a moment (it is append-only by DB trigger -- migration
// 000222 -- so DELETE is blocked even under system context, and truncating a
// shared table other processes may be writing to is not an option).

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/security/audit"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// TestPostgresRepository_List_NoMatch_PaginationDefaultsDontError proves the
// parts of List() that are safely testable without ever scanning a row: a
// filter matching nothing returns an empty result without error, and the
// Limit<=0/Offset<0 defaults reach the SQL without producing a syntax or
// range error (a literal negative OFFSET is rejected by Postgres outright).
func TestPostgresRepository_List_NoMatch_PaginationDefaultsDontError(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Audit List No-Match Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := audit.NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	entries, total, err := repo.List(ctx, &models.AuditFilter{
		TenantID: tenantID,
		Action:   "no-such-action-" + uuid.New().String(),
		Limit:    0,
		Offset:   -100,
	})
	if err != nil {
		t.Fatalf("List(no match, defaulted pagination): %v", err)
	}
	if total != 0 || len(entries) != 0 {
		t.Fatalf("expected an empty result for a non-matching action, got total=%d len=%d", total, len(entries))
	}
}

// TestPostgresRepository_List_IPAddressScanBug pins the bug described in the
// file header: List() errors as soon as its result set is non-empty, because
// it scans the raw INET ip_address column into a plain string. Seeds exactly
// one entry via the real Create() path (a normal, correctly hash-chained
// row -- safe to add permanently, same as every other write test in this
// package) and confirms List() fails with the specific pgx scan error rather
// than returning that entry. If this test starts failing because List()
// unexpectedly succeeds, the missing ip_address cast has been fixed --
// replace this test with real filter/pagination assertions at that point.
func TestPostgresRepository_List_IPAddressScanBug(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Audit List IP Scan Bug Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := audit.NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	run := uuid.New().String()
	id := uuid.New()
	entry := &models.AuditEntry{
		ID:        id,
		TenantID:  tenantID,
		Action:    "ip-scan-bug-" + run,
		Result:    audit.ResultSuccess,
		Details:   "{}",
		IPAddress: "10.0.0.1",
	}
	if err := repo.Create(ctx, entry); err != nil {
		t.Fatalf("seed Create: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "audit_log", id)

	_, _, err := repo.List(ctx, &models.AuditFilter{
		TenantID: tenantID,
		Action:   "ip-scan-bug-" + run,
		Limit:    50,
	})
	if err == nil {
		t.Fatal("expected List() to fail scanning ip_address (see file header) -- it returned successfully, which means the bug is fixed; replace this test with real assertions")
	}
	if !strings.Contains(err.Error(), "ip_address") {
		t.Fatalf("expected a scan error mentioning ip_address, got: %v", err)
	}
}

// TestPostgresRepository_GetLastHash covers GetLastHash's non-empty branch --
// unlike List/VerifyChain it selects only entry_hash, so it is unaffected by
// the ip_address scan bug documented above.
func TestPostgresRepository_GetLastHash(t *testing.T) {
	testutil.SkipIfNoDB(t)
	// Deliberately not t.Parallel(): GetLastHash reads the tail of the single
	// global audit_log chain with no tenant or range filter. Running serially
	// (Go never overlaps non-t.Parallel tests with each other) keeps the
	// window between the seed Create() and the GetLastHash read free of
	// interleaved writes from other tests in this package.
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	repo := audit.NewPostgresRepository(pool)
	ctx := testutil.WithSystemCtx(context.Background())

	entry := &models.AuditEntry{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Action:   fmt.Sprintf("last-hash-%s", uuid.New()),
		Result:   audit.ResultSuccess,
		Details:  "{}",
	}
	if err := repo.Create(ctx, entry); err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "audit_log", entry.ID)

	lastHash, err := repo.GetLastHash(ctx)
	if err != nil {
		t.Fatalf("GetLastHash: %v", err)
	}
	if lastHash != entry.EntryHash {
		t.Fatalf("GetLastHash = %q, want the most recently created entry's hash %q", lastHash, entry.EntryHash)
	}
}

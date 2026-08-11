package audit_test

// List() and VerifyChain() both scan the ip_address column -- INET on
// audit_log -- via COALESCE(host(ip_address), '') into models.AuditEntry.IPAddress
// (a plain string). Until fix-audit-list-verifychain-ip-address-scan, both
// queries selected ip_address raw instead of casting it first: pgx v5 cannot
// decode an INET value -- NULL or set -- into a string destination, so every
// call that would return at least one row failed unconditionally ("cannot
// scan NULL into *string" for an empty IP, "cannot scan inet (OID 869) in
// binary format into *string" for a real one), both reproduced directly
// against the local Postgres. That made the audit log viewer, the
// CSV/JSON export (Service.ExportEntries -> repo.List), and the
// VerifyAuditChain RPC (-> repo.VerifyChain) unusable for any tenant with
// real audit activity. internal/auth/postgres_repository.go was already
// using this exact COALESCE(host(ip_address), '') cast for the analogous
// user_sessions.ip_address column; internal/crm/consent and
// internal/formulare use ip_address::text for the same reason.
//
// VerifyChain has no tenant scope by design (it walks the one global
// chain), and this shared dev database's global chain contains links
// seeded by rls_test.go via testutil.SeedRow with hardcoded dummy
// previous_hash/entry_hash values that don't reflect the real preceding
// row. A test asserting "an arbitrary multi-row range verifies as intact"
// would therefore be flaky depending on what other tests happen to have
// written into that range. The two VerifyChain tests below sidestep that
// by using single-row ranges (fromSequence == toSequence): the row's own
// PreviousHash field was captured by Create() from the actual last hash at
// insert time, so a single-row range's validity depends only on that row
// itself, not on the surrounding chain state.
//
// GetLastHash's "no rows" branch (returns "") is not exercised: audit_log
// is a single global table, not just non-empty but structurally impossible
// to make empty for a moment (it is append-only by DB trigger -- migration
// 000222 -- so DELETE is blocked even under system context, and truncating
// a shared table other processes may be writing to is not an option).

import (
	"context"
	"fmt"
	"testing"
	"time"

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

// TestPostgresRepository_List_ReturnsSeededEntry_WithIPAddress proves the
// ip_address cast fix: a seeded entry carrying a real IP is returned by
// List() -- not a scan error -- with IPAddress round-tripped correctly.
func TestPostgresRepository_List_ReturnsSeededEntry_WithIPAddress(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Audit List IP Address Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := audit.NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	run := uuid.New().String()
	id := uuid.New()
	entry := &models.AuditEntry{
		ID:        id,
		TenantID:  tenantID,
		Action:    "ip-address-" + run,
		Result:    audit.ResultSuccess,
		Details:   "{}",
		IPAddress: "10.0.0.1",
	}
	if err := repo.Create(ctx, entry); err != nil {
		t.Fatalf("seed Create: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "audit_log", id)

	entries, total, err := repo.List(ctx, &models.AuditFilter{
		TenantID: tenantID,
		Action:   "ip-address-" + run,
		Limit:    50,
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 1 || len(entries) != 1 {
		t.Fatalf("expected exactly one matching entry, got total=%d len=%d", total, len(entries))
	}
	if entries[0].ID != id {
		t.Fatalf("entries[0].ID = %s, want %s", entries[0].ID, id)
	}
	if entries[0].IPAddress != "10.0.0.1" {
		t.Fatalf("entries[0].IPAddress = %q, want %q", entries[0].IPAddress, "10.0.0.1")
	}
}

// TestPostgresRepository_List_ReturnsSeededEntry_NullIPAddress proves the
// other half of the cast fix: a row with no IP at all (SQL NULL, the path
// Create() takes when IPAddress == "") still scans cleanly, as "".
func TestPostgresRepository_List_ReturnsSeededEntry_NullIPAddress(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Audit List Null IP Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := audit.NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	run := uuid.New().String()
	id := uuid.New()
	entry := &models.AuditEntry{
		ID:       id,
		TenantID: tenantID,
		Action:   "null-ip-" + run,
		Result:   audit.ResultSuccess,
		Details:  "{}",
	}
	if err := repo.Create(ctx, entry); err != nil {
		t.Fatalf("seed Create: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "audit_log", id)

	entries, total, err := repo.List(ctx, &models.AuditFilter{
		TenantID: tenantID,
		Action:   "null-ip-" + run,
		Limit:    50,
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 1 || len(entries) != 1 {
		t.Fatalf("expected exactly one matching entry, got total=%d len=%d", total, len(entries))
	}
	if entries[0].IPAddress != "" {
		t.Fatalf("entries[0].IPAddress = %q, want empty string for a NULL ip_address", entries[0].IPAddress)
	}
}

// TestPostgresRepository_List_FiltersByResult proves the Result filter
// narrows within a shared Action, now that a matching row can actually be
// scanned back.
func TestPostgresRepository_List_FiltersByResult(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Audit List Filter Result Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := audit.NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	run := uuid.New().String()
	action := "filter-result-" + run

	successID := uuid.New()
	if err := repo.Create(ctx, &models.AuditEntry{
		ID: successID, TenantID: tenantID, Action: action,
		Result: audit.ResultSuccess, Details: "{}",
	}); err != nil {
		t.Fatalf("seed success Create: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "audit_log", successID)

	failureID := uuid.New()
	if err := repo.Create(ctx, &models.AuditEntry{
		ID: failureID, TenantID: tenantID, Action: action,
		Result: audit.ResultFailure, Details: "{}",
	}); err != nil {
		t.Fatalf("seed failure Create: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "audit_log", failureID)

	entries, total, err := repo.List(ctx, &models.AuditFilter{
		TenantID: tenantID,
		Action:   action,
		Result:   audit.ResultFailure,
		Limit:    50,
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 1 || len(entries) != 1 {
		t.Fatalf("expected exactly one failure-result entry, got total=%d len=%d", total, len(entries))
	}
	if entries[0].ID != failureID {
		t.Fatalf("entries[0].ID = %s, want the failure entry %s", entries[0].ID, failureID)
	}
}

// TestPostgresRepository_List_PaginationLimitAndOffset proves Limit/Offset
// slice a shared Action's entries correctly, newest first.
func TestPostgresRepository_List_PaginationLimitAndOffset(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Audit List Pagination Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)

	repo := audit.NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	run := uuid.New().String()
	action := "pagination-" + run

	ids := make([]uuid.UUID, 3)
	for i := range ids {
		id := uuid.New()
		if err := repo.Create(ctx, &models.AuditEntry{
			ID: id, TenantID: tenantID, Action: action,
			Result: audit.ResultSuccess, Details: "{}",
		}); err != nil {
			t.Fatalf("seed entry %d Create: %v", i, err)
		}
		ids[i] = id
		defer testutil.CleanupRow(t, pool, "audit_log", id)
	}
	// List orders by sequence_num DESC, so the most recently created entry
	// (ids[2]) sorts first.
	newestFirst := []uuid.UUID{ids[2], ids[1], ids[0]}

	page1, total, err := repo.List(ctx, &models.AuditFilter{
		TenantID: tenantID, Action: action, Limit: 2, Offset: 0,
	})
	if err != nil {
		t.Fatalf("List page 1: %v", err)
	}
	if total != 3 {
		t.Fatalf("total = %d, want 3", total)
	}
	if len(page1) != 2 || page1[0].ID != newestFirst[0] || page1[1].ID != newestFirst[1] {
		t.Fatalf("page1 = %+v, want first two of %v", entryIDs(page1), newestFirst[:2])
	}

	page2, total, err := repo.List(ctx, &models.AuditFilter{
		TenantID: tenantID, Action: action, Limit: 2, Offset: 2,
	})
	if err != nil {
		t.Fatalf("List page 2: %v", err)
	}
	if total != 3 {
		t.Fatalf("total = %d, want 3", total)
	}
	if len(page2) != 1 || page2[0].ID != newestFirst[2] {
		t.Fatalf("page2 = %+v, want [%s]", entryIDs(page2), newestFirst[2])
	}
}

func entryIDs(entries []*models.AuditEntry) []uuid.UUID {
	out := make([]uuid.UUID, len(entries))
	for i, e := range entries {
		out[i] = e.ID
	}
	return out
}

// TestPostgresRepository_VerifyChain_ValidSingleEntryRange proves the
// ip_address cast fix on the VerifyChain path: a freshly created entry
// verifies as valid over a single-row range (fromSequence == toSequence),
// which depends only on that row's own PreviousHash/EntryHash, not on the
// state of the surrounding shared chain.
func TestPostgresRepository_VerifyChain_ValidSingleEntryRange(t *testing.T) {
	testutil.SkipIfNoDB(t)
	// Deliberately not t.Parallel(): reads its own row's sequence_num right
	// after the seed write, no shared fixture beyond the global audit_log
	// table.
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	repo := audit.NewPostgresRepository(pool)
	ctx := testutil.WithSystemCtx(context.Background())

	id := uuid.New()
	if err := repo.Create(ctx, &models.AuditEntry{
		ID: id, TenantID: uuid.New(), Action: fmt.Sprintf("verify-chain-valid-%s", uuid.New()),
		Result: audit.ResultSuccess, Details: "{}",
		// Truncated to microsecond: audit_log.timestamp is TIMESTAMPTZ, which
		// only stores microsecond precision. computeEntryHash formats with
		// RFC3339Nano, so an untruncated time.Now().UTC() (this machine
		// regularly has non-zero sub-microsecond digits) would hash
		// differently before insert than after a round trip through
		// Postgres, and VerifyChain's recompute would spuriously report a
		// valid chain as broken. That mismatch is real and reachable via the
		// normal Create() path -- separate bug, tracked as
		// fix-audit-verifychain-timestamp-precision-mismatch. Truncating
		// here isolates this test to the link/recompute logic this unit is
		// responsible for.
		Timestamp: time.Now().UTC().Truncate(time.Microsecond),
	}); err != nil {
		t.Fatalf("seed Create: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "audit_log", id)

	var seq int64
	if err := pool.QueryRow(ctx, "SELECT sequence_num FROM audit_log WHERE id = $1", id).Scan(&seq); err != nil {
		t.Fatalf("read seeded sequence_num: %v", err)
	}

	valid, brokenSeq, err := repo.VerifyChain(ctx, seq, seq)
	if err != nil {
		t.Fatalf("VerifyChain: %v", err)
	}
	if !valid {
		t.Fatalf("VerifyChain(%d, %d) = invalid at sequence %d, want valid", seq, seq, brokenSeq)
	}
}

// TestPostgresRepository_VerifyChain_DetectsTamperedEntry proves
// VerifyChain still detects a broken link once it can scan the row at all:
// a row seeded with a hash that doesn't match its own fields must verify
// as invalid, pointing at its own sequence number.
func TestPostgresRepository_VerifyChain_DetectsTamperedEntry(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	repo := audit.NewPostgresRepository(pool)

	// target/target_type/details are included explicitly (not left to
	// column defaults) so this test exercises only the tamper-detection
	// path -- omitting them would leave those nullable columns NULL, which
	// Create() never does for a real entry (it always writes "" or a JSON
	// string), so a NULL there would be a SeedRow test artifact, not
	// production-reachable behavior.
	id := testutil.SeedRow(t, pool, "audit_log", map[string]any{
		"tenant_id":     uuid.New(),
		"action":        fmt.Sprintf("verify-chain-tampered-%s", uuid.New()),
		"target":        "",
		"target_type":   "",
		"details":       "{}",
		"user_agent":    "",
		"result":        audit.ResultSuccess,
		"entry_hash":    "tampered-hash-does-not-match-fields",
		"previous_hash": "",
	})
	defer testutil.CleanupRow(t, pool, "audit_log", id)

	ctx := testutil.WithSystemCtx(context.Background())
	var seq int64
	if err := pool.QueryRow(ctx, "SELECT sequence_num FROM audit_log WHERE id = $1", id).Scan(&seq); err != nil {
		t.Fatalf("read seeded sequence_num: %v", err)
	}

	valid, brokenSeq, err := repo.VerifyChain(ctx, seq, seq)
	if err != nil {
		t.Fatalf("VerifyChain: %v", err)
	}
	if valid {
		t.Fatalf("VerifyChain(%d, %d) = valid, want invalid for a tampered entry_hash", seq, seq)
	}
	if brokenSeq != seq {
		t.Fatalf("brokenSeq = %d, want %d", brokenSeq, seq)
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

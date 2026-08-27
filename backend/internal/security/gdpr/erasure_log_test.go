package gdpr

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func seedErasureLogUser(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("wp-security-erasure-%s@test.invalid", uuid.New()),
		"password_hash": "x",
	})
}

func TestCreateErasureLog_TenantDerivedFromUser(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "GDPR Erasure Log Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "GDPR Erasure Log Other Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)
	defer testutil.CleanupRow(t, pool, "tenants", tenantOther)

	userOwn := seedErasureLogUser(t, pool, tenantOwn)
	defer testutil.CleanupRow(t, pool, "users", userOwn)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())

	// CreateErasureLog derives tenant_id from the target user's row (not from
	// the caller's ctx), so a caller from a different tenant logging an
	// erasure of userOwn must be rejected by RLS's WITH CHECK.
	foreignEntry := &models.GDPRErasureLog{
		ID:               uuid.New(),
		OriginalUserID:   userOwn,
		AnonymizedLabel:  "Geloeschter Benutzer #999",
		ExecutedBy:       userOwn,
		ExecutedAt:       time.Now().UTC(),
		ModulesAffected:  map[string]string{"crm": "anonymized"},
		ConfirmationHash: "foreign-hash",
	}
	if err := repo.CreateErasureLog(ctxOther, foreignEntry); err == nil {
		t.Fatalf("CreateErasureLog (foreign ctx): expected an RLS error, got nil")
	}
	testutil.AssertRowCount(t, pool, sysCtx, "gdpr_erasure_log", foreignEntry.ID, 0)

	entry := &models.GDPRErasureLog{
		ID:               uuid.New(),
		OriginalUserID:   userOwn,
		AnonymizedLabel:  "Geloeschter Benutzer #1",
		ExecutedBy:       userOwn,
		ExecutedAt:       time.Now().UTC(),
		ModulesAffected:  map[string]string{"crm": "anonymized", "chat": "anonymized"},
		ConfirmationHash: "own-hash",
	}
	if err := repo.CreateErasureLog(ctxOwn, entry); err != nil {
		t.Fatalf("CreateErasureLog (own ctx): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "gdpr_erasure_log", entry.ID)

	testutil.AssertRowCount(t, pool, ctxOwn, "gdpr_erasure_log", entry.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "gdpr_erasure_log", entry.ID, 0)

	var label, hash string
	var modulesJSON []byte
	if err := pool.QueryRow(ctxOwn,
		`SELECT anonymized_label, confirmation_hash, modules_affected FROM gdpr_erasure_log WHERE id = $1`,
		entry.ID,
	).Scan(&label, &hash, &modulesJSON); err != nil {
		t.Fatalf("read back gdpr_erasure_log row: %v", err)
	}
	if label != entry.AnonymizedLabel {
		t.Fatalf("anonymized_label: got %q, want %q", label, entry.AnonymizedLabel)
	}
	if hash != entry.ConfirmationHash {
		t.Fatalf("confirmation_hash: got %q, want %q", hash, entry.ConfirmationHash)
	}
	if !strings.Contains(string(modulesJSON), `"crm"`) || !strings.Contains(string(modulesJSON), `"chat"`) {
		t.Fatalf("modules_affected did not round-trip: %s", modulesJSON)
	}
}

func TestCreateErasureLog_UnknownUser(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "GDPR Erasure Log Unknown-User Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenant)

	// executor must be a real user (executed_by has an FK) so the error below
	// is attributable to the unresolved original_user_id, not a second,
	// unrelated FK violation.
	executor := seedErasureLogUser(t, pool, tenant)
	defer testutil.CleanupRow(t, pool, "users", executor)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	// The tenant_id subselect resolves to NULL when original_user_id matches
	// no user; the NOT NULL constraint on tenant_id must reject the insert.
	entry := &models.GDPRErasureLog{
		ID:               uuid.New(),
		OriginalUserID:   uuid.New(),
		AnonymizedLabel:  "Geloeschter Benutzer #1",
		ExecutedBy:       executor,
		ExecutedAt:       time.Now().UTC(),
		ModulesAffected:  map[string]string{"crm": "anonymized"},
		ConfirmationHash: "orphan-hash",
	}
	if err := repo.CreateErasureLog(ctx, entry); err == nil {
		t.Fatalf("CreateErasureLog (unknown user): expected an error, got nil")
	}
}

func TestGetNextAnonymizedLabel_IncrementsPerCall(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "GDPR Label Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenant)

	user := seedErasureLogUser(t, pool, tenant)
	defer testutil.CleanupRow(t, pool, "users", user)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	label1, err := repo.GetNextAnonymizedLabel(ctx)
	if err != nil {
		t.Fatalf("GetNextAnonymizedLabel (1st): %v", err)
	}
	if label1 != "Geloeschter Benutzer #1" {
		t.Fatalf("GetNextAnonymizedLabel (1st): got %q, want %q", label1, "Geloeschter Benutzer #1")
	}

	entry1 := &models.GDPRErasureLog{
		ID:               uuid.New(),
		OriginalUserID:   user,
		AnonymizedLabel:  label1,
		ExecutedBy:       user,
		ExecutedAt:       time.Now().UTC(),
		ModulesAffected:  map[string]string{"crm": "anonymized"},
		ConfirmationHash: "hash-1",
	}
	if err := repo.CreateErasureLog(ctx, entry1); err != nil {
		t.Fatalf("CreateErasureLog (1st): %v", err)
	}
	defer testutil.CleanupRow(t, pool, "gdpr_erasure_log", entry1.ID)

	label2, err := repo.GetNextAnonymizedLabel(ctx)
	if err != nil {
		t.Fatalf("GetNextAnonymizedLabel (2nd): %v", err)
	}
	if label2 != "Geloeschter Benutzer #2" {
		t.Fatalf("GetNextAnonymizedLabel (2nd): got %q, want %q", label2, "Geloeschter Benutzer #2")
	}
	if label1 == label2 {
		t.Fatalf("expected distinct labels across calls, got %q both times", label1)
	}
}

// TestGetNextAnonymizedLabel_ConcurrentCallersCollide is the counter-race
// scope question from cov-security-gdpr-remaining-paths: "what happens on two
// simultaneous anonymizations?" GetNextAnonymizedLabel (postgres_repository.go:
// 197) is a bare `SELECT COUNT(*) FROM gdpr_erasure_log` with no lock and no
// sequence behind it -- two callers that both read before either writes get
// the identical "next" number.
//
// The repo method itself always goes through the pool (one query, one
// implicit connection), so it cannot be driven from inside a caller-held
// transaction. This reproduces its exact query on two explicitly held
// REPEATABLE READ transactions instead -- the "hold a second connection"
// shape every per-resource race test in this package uses, here proving the
// collision deterministically rather than via goroutine timing: each
// transaction's snapshot is pinned at its first statement, so as long as
// neither has committed an INSERT yet (true here -- both read before either
// writes), they are guaranteed to observe the identical count regardless of
// unrelated activity elsewhere in the shared table.
func TestGetNextAnonymizedLabel_ConcurrentCallersCollide(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "GDPR Label Race Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenant)

	userA := seedErasureLogUser(t, pool, tenant)
	defer testutil.CleanupRow(t, pool, "users", userA)
	userB := seedErasureLogUser(t, pool, tenant)
	defer testutil.CleanupRow(t, pool, "users", userB)

	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	txA, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		t.Fatalf("begin txA: %v", err)
	}
	defer func() { _ = txA.Rollback(ctx) }()
	var countA int
	if err := txA.QueryRow(ctx, `SELECT COUNT(*) FROM gdpr_erasure_log`).Scan(&countA); err != nil {
		t.Fatalf("txA count: %v", err)
	}

	txB, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		t.Fatalf("begin txB: %v", err)
	}
	defer func() { _ = txB.Rollback(ctx) }()
	var countB int
	if err := txB.QueryRow(ctx, `SELECT COUNT(*) FROM gdpr_erasure_log`).Scan(&countB); err != nil {
		t.Fatalf("txB count: %v", err)
	}

	if countA != countB {
		t.Fatalf("txA and txB must observe the identical count (got %d vs %d) -- "+
			"that identical read is exactly the race window GetNextAnonymizedLabel has no lock against",
			countA, countB)
	}

	labelA := fmt.Sprintf("Geloeschter Benutzer #%d", countA+1)
	labelB := fmt.Sprintf("Geloeschter Benutzer #%d", countB+1)

	idA := uuid.New()
	if _, err := txA.Exec(ctx,
		`INSERT INTO gdpr_erasure_log (id, tenant_id, original_user_id, anonymized_label, executed_by, executed_at, modules_affected, confirmation_hash)
		 VALUES ($1, $2, $3, $4, $3, now(), '{}', 'race-a')`,
		idA, tenant, userA, labelA); err != nil {
		t.Fatalf("txA insert: %v", err)
	}
	if err := txA.Commit(ctx); err != nil {
		t.Fatalf("txA commit: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "gdpr_erasure_log", idA)

	idB := uuid.New()
	if _, err := txB.Exec(ctx,
		`INSERT INTO gdpr_erasure_log (id, tenant_id, original_user_id, anonymized_label, executed_by, executed_at, modules_affected, confirmation_hash)
		 VALUES ($1, $2, $3, $4, $3, now(), '{}', 'race-b')`,
		idB, tenant, userB, labelB); err != nil {
		t.Fatalf("txB insert: %v", err)
	}
	if err := txB.Commit(ctx); err != nil {
		t.Fatalf("txB commit: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "gdpr_erasure_log", idB)

	if labelA != labelB {
		t.Fatalf("expected the race to produce IDENTICAL labels, got %q and %q -- "+
			"if this fails, GetNextAnonymizedLabel gained a lock and this test (and its filed follow-up unit) is obsolete",
			labelA, labelB)
	}
}

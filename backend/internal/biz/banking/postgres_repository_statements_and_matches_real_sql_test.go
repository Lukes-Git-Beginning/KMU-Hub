package banking

// Real-SQL coverage for the repository methods integration_transactions_test.go
// and integration_accounts_test.go leave untouched: GetStatementByHash /
// GetStatement / ListStatements / ListTransactionsByStatement /
// UpdateTransactionMatch, plus the one question that decides whether a
// re-imported file is safe: does the unique constraint on (tenant_id,
// content_hash) actually stop a second row from landing.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestPostgresRepository_GetStatementByHash_FoundNotFoundAndTenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	mine, theirs := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, mine, "Bank Hash Tenant Mine")
	testutil.EnsureTenant(t, pool, theirs, "Bank Hash Tenant Theirs")

	repo := NewPostgresRepository(pool)
	myCtx := testutil.WithTenantCtx(context.Background(), mine)
	theirCtx := testutil.WithTenantCtx(context.Background(), theirs)

	// Both tenants import a file that happens to hash the same (e.g. two banks
	// that both sent an empty CAMT.053 skeleton) — the constraint is per tenant,
	// so both rows must exist, and each tenant's lookup must land on its own.
	sharedHash := uuid.NewString()
	opening := decimal.RequireFromString("1000.00")
	closing := decimal.RequireFromString("-250.50")
	mineStmt := &models.BankStatement{
		ID: uuid.New(), TenantID: mine, Format: "camt053", Filename: "mine.xml",
		ContentHash: sharedHash, AccountIBAN: testIBAN, Currency: "EUR",
		OpeningBalance: &opening, ClosingBalance: &closing, CreatedAt: time.Now().UTC(),
	}
	if err := repo.CreateStatement(myCtx, mineStmt, nil); err != nil {
		t.Fatalf("CreateStatement (mine): %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_bank_statements", mineStmt.ID) })

	theirStmt := &models.BankStatement{
		ID: uuid.New(), TenantID: theirs, Format: "mt940", Filename: "theirs.sta",
		ContentHash: sharedHash, AccountIBAN: testIBAN, Currency: "EUR", CreatedAt: time.Now().UTC(),
	}
	if err := repo.CreateStatement(theirCtx, theirStmt, nil); err != nil {
		t.Fatalf("CreateStatement (theirs): %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_bank_statements", theirStmt.ID) })

	t.Run("own tenant's row comes back, with the balance surviving the nullable roundtrip", func(t *testing.T) {
		got, err := repo.GetStatementByHash(myCtx, mine, sharedHash)
		if err != nil {
			t.Fatalf("GetStatementByHash: %v", err)
		}
		if got.ID != mineStmt.ID {
			t.Errorf("id = %s, want %s (their statement with the same hash must not leak)", got.ID, mineStmt.ID)
		}
		if got.OpeningBalance == nil || !got.OpeningBalance.Equal(opening) {
			t.Errorf("opening balance = %v, want %s", got.OpeningBalance, opening)
		}
		if got.ClosingBalance == nil || !got.ClosingBalance.Equal(closing) {
			t.Errorf("closing balance = %v, want %s (a debit-side closing balance)", got.ClosingBalance, closing)
		}
	})

	t.Run("their tenant's lookup lands on their own row, not mine", func(t *testing.T) {
		got, err := repo.GetStatementByHash(theirCtx, theirs, sharedHash)
		if err != nil {
			t.Fatalf("GetStatementByHash: %v", err)
		}
		if got.ID != theirStmt.ID {
			t.Errorf("id = %s, want %s", got.ID, theirStmt.ID)
		}
		if got.OpeningBalance != nil || got.ClosingBalance != nil {
			t.Errorf("balances = %v / %v, want both nil (never set)", got.OpeningBalance, got.ClosingBalance)
		}
	})

	t.Run("an unknown hash is not found", func(t *testing.T) {
		if _, err := repo.GetStatementByHash(myCtx, mine, uuid.NewString()); !errors.Is(err, ErrStatementNotFound) {
			t.Fatalf("err = %v, want ErrStatementNotFound", err)
		}
	})
}

func TestPostgresRepository_GetStatement_ForeignIDIsNotFound(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	mine, theirs := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, mine, "Bank GetByID Tenant Mine")
	testutil.EnsureTenant(t, pool, theirs, "Bank GetByID Tenant Theirs")

	repo := NewPostgresRepository(pool)
	theirCtx := testutil.WithTenantCtx(context.Background(), theirs)

	theirStmt := &models.BankStatement{
		ID: uuid.New(), TenantID: theirs, Format: "camt053", Filename: "theirs.xml",
		ContentHash: uuid.NewString(), AccountIBAN: testIBAN, Currency: "EUR", CreatedAt: time.Now().UTC(),
	}
	if err := repo.CreateStatement(theirCtx, theirStmt, nil); err != nil {
		t.Fatalf("CreateStatement: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_bank_statements", theirStmt.ID) })

	myCtx := testutil.WithTenantCtx(context.Background(), mine)
	if _, err := repo.GetStatement(myCtx, mine, theirStmt.ID); !errors.Is(err, ErrStatementNotFound) {
		t.Fatalf("GetStatement across tenants: err = %v, want ErrStatementNotFound", err)
	}
}

func TestPostgresRepository_ListStatements_PaginationTotalAndTenantScoping(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	mine, theirs := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, mine, "Bank List Tenant Mine")
	testutil.EnsureTenant(t, pool, theirs, "Bank List Tenant Theirs")

	repo := NewPostgresRepository(pool)
	myCtx := testutil.WithTenantCtx(context.Background(), mine)
	theirCtx := testutil.WithTenantCtx(context.Background(), theirs)

	base := time.Now().UTC()
	var ids []uuid.UUID
	for i := range 3 {
		stmt := &models.BankStatement{
			ID: uuid.New(), TenantID: mine, Format: "camt053", Filename: "mine.xml",
			ContentHash: uuid.NewString(), AccountIBAN: testIBAN, Currency: "EUR",
			// Strictly increasing so DESC order is unambiguous even at second resolution.
			CreatedAt: base.Add(time.Duration(i) * time.Minute),
		}
		if err := repo.CreateStatement(myCtx, stmt, nil); err != nil {
			t.Fatalf("CreateStatement %d: %v", i, err)
		}
		t.Cleanup(func(id uuid.UUID) func() {
			return func() { testutil.CleanupRow(t, pool, "finance_bank_statements", id) }
		}(stmt.ID))
		ids = append(ids, stmt.ID)
	}
	foreign := &models.BankStatement{
		ID: uuid.New(), TenantID: theirs, Format: "camt053", Filename: "theirs.xml",
		ContentHash: uuid.NewString(), AccountIBAN: testIBAN, Currency: "EUR", CreatedAt: base,
	}
	if err := repo.CreateStatement(theirCtx, foreign, nil); err != nil {
		t.Fatalf("CreateStatement (foreign): %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_bank_statements", foreign.ID) })

	t.Run("first page, newest first, total counts only mine", func(t *testing.T) {
		rows, total, err := repo.ListStatements(myCtx, mine, 2, 0)
		if err != nil {
			t.Fatalf("ListStatements: %v", err)
		}
		if total != 3 {
			t.Fatalf("total = %d, want 3 (the foreign statement must not be counted)", total)
		}
		if len(rows) != 2 || rows[0].ID != ids[2] || rows[1].ID != ids[1] {
			t.Fatalf("page 1 = %v, want [ids[2], ids[1]] (newest first)", rowIDs(rows))
		}
	})

	t.Run("second page carries the remainder", func(t *testing.T) {
		rows, total, err := repo.ListStatements(myCtx, mine, 2, 2)
		if err != nil {
			t.Fatalf("ListStatements: %v", err)
		}
		if total != 3 || len(rows) != 1 || rows[0].ID != ids[0] {
			t.Fatalf("page 2 = %v (total %d), want [ids[0]] (total 3)", rowIDs(rows), total)
		}
	})

	t.Run("a tenant with nothing imported gets an empty slice, not null", func(t *testing.T) {
		empty := uuid.New()
		testutil.EnsureTenant(t, pool, empty, "Bank List Tenant Empty")
		rows, total, err := repo.ListStatements(testutil.WithTenantCtx(context.Background(), empty), empty, 10, 0)
		if err != nil {
			t.Fatalf("ListStatements: %v", err)
		}
		if total != 0 || rows == nil || len(rows) != 0 {
			t.Fatalf("rows = %v (nil=%v), total = %d, want empty non-nil slice / 0", rows, rows == nil, total)
		}
	})
}

func rowIDs(rows []*models.BankStatement) []uuid.UUID {
	out := make([]uuid.UUID, len(rows))
	for i, r := range rows {
		out[i] = r.ID
	}
	return out
}

func TestPostgresRepository_ListTransactionsByStatement_OrderedAndTenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	mine, theirs := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, mine, "Bank ListByStatement Tenant Mine")
	testutil.EnsureTenant(t, pool, theirs, "Bank ListByStatement Tenant Theirs")

	repo := NewPostgresRepository(pool)
	myCtx := testutil.WithTenantCtx(context.Background(), mine)
	theirCtx := testutil.WithTenantCtx(context.Background(), theirs)

	// value_date is a DATE column — anchor on midnight UTC so the value read
	// back compares equal without a time-of-day component to trip over.
	now := time.Now().UTC().Truncate(24 * time.Hour)
	stmt := &models.BankStatement{
		ID: uuid.New(), TenantID: mine, Format: "camt053", Filename: "mine.xml",
		ContentHash: uuid.NewString(), AccountIBAN: testIBAN, Currency: "EUR", CreatedAt: now,
	}
	later := now.AddDate(0, 0, 1)
	earlier := now.AddDate(0, 0, -1)
	txs := []*models.BankTransaction{
		{ID: uuid.New(), TenantID: mine, StatementID: stmt.ID, ValueDate: now, Amount: decimal.RequireFromString("10.00"), Currency: "EUR", MatchStatus: models.BankMatchUnmatched, CreatedAt: now},
		{ID: uuid.New(), TenantID: mine, StatementID: stmt.ID, ValueDate: earlier, Amount: decimal.RequireFromString("-5.00"), Currency: "EUR", MatchStatus: models.BankMatchUnmatched, CreatedAt: now},
		{ID: uuid.New(), TenantID: mine, StatementID: stmt.ID, ValueDate: later, Amount: decimal.RequireFromString("20.00"), Currency: "EUR", MatchStatus: models.BankMatchUnmatched, CreatedAt: now},
	}
	if err := repo.CreateStatement(myCtx, stmt, txs); err != nil {
		t.Fatalf("CreateStatement: %v", err)
	}
	t.Cleanup(func() {
		for _, tx := range txs {
			testutil.CleanupRow(t, pool, "finance_bank_transactions", tx.ID)
		}
		testutil.CleanupRow(t, pool, "finance_bank_statements", stmt.ID)
	})

	t.Run("returned oldest value date first", func(t *testing.T) {
		rows, err := repo.ListTransactionsByStatement(myCtx, mine, stmt.ID)
		if err != nil {
			t.Fatalf("ListTransactionsByStatement: %v", err)
		}
		if len(rows) != 3 {
			t.Fatalf("got %d rows, want 3", len(rows))
		}
		if !rows[0].ValueDate.Equal(earlier) || !rows[1].ValueDate.Equal(now) || !rows[2].ValueDate.Equal(later) {
			t.Fatalf("order = %v/%v/%v, want earlier/now/later", rows[0].ValueDate, rows[1].ValueDate, rows[2].ValueDate)
		}
		if !rows[1].Amount.Equal(decimal.RequireFromString("10.00")) {
			t.Errorf("amount = %s, want 10.00 (sign must survive the roundtrip)", rows[1].Amount)
		}
	})

	t.Run("a foreign tenant reading this statement id gets nothing, not an error", func(t *testing.T) {
		rows, err := repo.ListTransactionsByStatement(theirCtx, theirs, stmt.ID)
		if err != nil {
			t.Fatalf("ListTransactionsByStatement: %v", err)
		}
		if len(rows) != 0 {
			t.Fatalf("got %d rows for a statement outside the tenant, want 0", len(rows))
		}
	})
}

func TestPostgresRepository_UpdateTransactionMatch_PersistsAndForeignIsNoop(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	mine, theirs := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, mine, "Bank Update Match Tenant Mine")
	testutil.EnsureTenant(t, pool, theirs, "Bank Update Match Tenant Theirs")

	repo := NewPostgresRepository(pool)
	myCtx := testutil.WithTenantCtx(context.Background(), mine)

	invoiceID := seedInvoice(t, pool, myCtx, mine, "RE-MATCH-0001")
	tx := seedTransaction(t, myCtx, repo, mine, models.BankMatchUnmatched, nil, "119.00")

	t.Run("the reconciliation columns persist and the bank-reported columns stay put", func(t *testing.T) {
		reconciledAt := time.Now().UTC().Truncate(time.Second)
		update := &models.BankTransaction{
			TenantID: mine, ID: tx.ID, MatchStatus: models.BankMatchMatched,
			MatchReason: "manual", MatchedInvoiceID: &invoiceID,
			ReconciledAt: &reconciledAt,
			// reconciled_by left nil: it FKs to users, and no user fixture is
			// worth seeding just to prove this column round-trips a uuid.
		}
		if err := repo.UpdateTransactionMatch(myCtx, update); err != nil {
			t.Fatalf("UpdateTransactionMatch: %v", err)
		}

		got, err := repo.GetTransaction(myCtx, mine, tx.ID)
		if err != nil {
			t.Fatalf("GetTransaction: %v", err)
		}
		if got.MatchStatus != models.BankMatchMatched || got.MatchReason != "manual" {
			t.Errorf("match_status/reason = %q/%q, want matched/manual", got.MatchStatus, got.MatchReason)
		}
		if got.MatchedInvoiceID == nil || *got.MatchedInvoiceID != invoiceID {
			t.Errorf("matched_invoice_id = %v, want %s", got.MatchedInvoiceID, invoiceID)
		}
		if got.ReconciledAt == nil || !got.ReconciledAt.Equal(reconciledAt) {
			t.Errorf("reconciled_at = %v, want %v", got.ReconciledAt, reconciledAt)
		}
		if got.MatchedInvoiceNumber != "RE-MATCH-0001" {
			t.Errorf("matched invoice number = %q, want RE-MATCH-0001", got.MatchedInvoiceNumber)
		}
		// What the bank reported must never move under a reconciliation write.
		if !got.Amount.Equal(decimal.RequireFromString("119.00")) || got.CounterpartyName != "Gruber Maschinenbau GmbH" {
			t.Errorf("bank-reported columns changed: amount=%s counterparty=%q", got.Amount, got.CounterpartyName)
		}
	})

	t.Run("a foreign tenant id touches nothing and reports not found", func(t *testing.T) {
		update := &models.BankTransaction{
			TenantID: theirs, ID: tx.ID, MatchStatus: models.BankMatchIgnored,
		}
		if err := repo.UpdateTransactionMatch(testutil.WithTenantCtx(context.Background(), theirs), update); !errors.Is(err, ErrTransactionNotFound) {
			t.Fatalf("err = %v, want ErrTransactionNotFound", err)
		}

		// The row from the previous subtest must still read as matched: a
		// cross-tenant call must be a pure no-op, not a partial write.
		got, err := repo.GetTransaction(myCtx, mine, tx.ID)
		if err != nil {
			t.Fatalf("GetTransaction after the foreign call: %v", err)
		}
		if got.MatchStatus != models.BankMatchMatched {
			t.Errorf("match_status = %q after a foreign-tenant update, want it unchanged at matched", got.MatchStatus)
		}
	})
}

// The re-import question the unit exists to answer: does the unique
// constraint on (tenant_id, content_hash) actually stop a second row, and
// what does CreateStatement hand back when it fires. Service.Import avoids
// this path on the sequential no-op case by checking GetStatementByHash
// first (see service.go) — this test drives CreateStatement directly to
// prove the constraint itself, which is what the two-requests-at-once case in
// that comment ultimately falls back on.
func TestPostgresRepository_CreateStatement_DuplicateContentHashIsRejectedNotDuplicated(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Bank Duplicate Hash Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	hash := uuid.NewString()
	first := &models.BankStatement{
		ID: uuid.New(), TenantID: tenant, Format: "camt053", Filename: "first.xml",
		ContentHash: hash, AccountIBAN: testIBAN, Currency: "EUR",
		TransactionCount: 1, CreatedAt: time.Now().UTC(),
	}
	firstTx := &models.BankTransaction{
		ID: uuid.New(), TenantID: tenant, StatementID: first.ID, ValueDate: time.Now().UTC(),
		Amount: decimal.RequireFromString("50.00"), Currency: "EUR",
		MatchStatus: models.BankMatchUnmatched, CreatedAt: time.Now().UTC(),
	}
	if err := repo.CreateStatement(ctx, first, []*models.BankTransaction{firstTx}); err != nil {
		t.Fatalf("CreateStatement (first): %v", err)
	}
	t.Cleanup(func() {
		testutil.CleanupRow(t, pool, "finance_bank_transactions", firstTx.ID)
		testutil.CleanupRow(t, pool, "finance_bank_statements", first.ID)
	})

	// Same tenant, same hash, different id and a different transaction — the
	// shape a re-upload of the identical file would take if it ever reached
	// CreateStatement instead of being short-circuited by the hash lookup.
	second := &models.BankStatement{
		ID: uuid.New(), TenantID: tenant, Format: "camt053", Filename: "second.xml",
		ContentHash: hash, AccountIBAN: testIBAN, Currency: "EUR",
		TransactionCount: 1, CreatedAt: time.Now().UTC(),
	}
	secondTx := &models.BankTransaction{
		ID: uuid.New(), TenantID: tenant, StatementID: second.ID, ValueDate: time.Now().UTC(),
		Amount: decimal.RequireFromString("999.00"), Currency: "EUR",
		MatchStatus: models.BankMatchUnmatched, CreatedAt: time.Now().UTC(),
	}
	err := repo.CreateStatement(ctx, second, []*models.BankTransaction{secondTx})
	if err == nil {
		t.Fatal("CreateStatement with a duplicate content_hash succeeded, want the unique constraint to fire")
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		t.Fatalf("err = %v, want a wrapped unique-violation (23505) — unlike CreateAccount, CreateStatement maps no sentinel for this case", err)
	}

	// The failed second attempt must not have left a dangling transaction row —
	// CreateStatement runs inside one transaction, so the rollback must take
	// both writes down together.
	var count int
	if scanErr := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM finance_bank_statements WHERE tenant_id = $1 AND content_hash = $2`,
		tenant, hash,
	).Scan(&count); scanErr != nil {
		t.Fatalf("count statements: %v", scanErr)
	}
	if count != 1 {
		t.Fatalf("statements with this hash = %d, want 1 (the constraint must not allow a second row)", count)
	}
	var txCount int
	if scanErr := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM finance_bank_transactions WHERE tenant_id = $1 AND statement_id = $2`,
		tenant, second.ID,
	).Scan(&txCount); scanErr != nil {
		t.Fatalf("count orphaned transactions: %v", scanErr)
	}
	if txCount != 0 {
		t.Fatalf("orphaned transactions for the rejected statement = %d, want 0 (rollback must be atomic)", txCount)
	}
}

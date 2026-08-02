package banking

// These tests run against a real database because the three things they prove
// cannot be proven against a fake: that the invoice number really comes out of
// a join rather than out of a column on the transaction, that the queue filter
// keeps set-aside entries out without hiding them from a caller who asks for
// them, and that both stay inside the tenant under kmuhub_app (NOSUPERUSER
// NOBYPASSRLS).

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// seedInvoice writes a minimal invoice so a transaction has something to match
// against. The number is what the join is expected to bring back.
func seedInvoice(t *testing.T, pool *pgxpool.Pool, ctx context.Context, tenantID uuid.UUID, number string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	now := time.Now().UTC()
	_, err := pool.Exec(ctx,
		`INSERT INTO finance_invoices (
			id, tenant_id, invoice_number, status,
			customer_name, customer_address, customer_email, customer_ust_id_nr,
			tax_mode, subtotal, total_tax, gross_total,
			invoice_date, due_date, payment_terms, created_by, created_at, updated_at
		) VALUES ($1,$2,$3,'sent','Bank Kunde GmbH','Addr','bank@example.com','',
			'standard','100','19','119',$4,$5,'30 days',$6,$7,$7)`,
		id, tenantID, number, now, now.Add(30*24*time.Hour), uuid.New(), now,
	)
	if err != nil {
		t.Fatalf("seed invoice %s: %v", number, err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_invoices", id) })
	return id
}

// seedTransaction writes one statement with one transaction in the given match
// state. CreateStatement is the real write path, so the fixture cannot drift
// from what an import produces.
func seedTransaction(
	t *testing.T, ctx context.Context, repo *PostgresRepository, tenantID uuid.UUID,
	matchStatus string, matchedInvoiceID *uuid.UUID, amount string,
) *models.BankTransaction {
	t.Helper()
	now := time.Now().UTC()
	stmt := &models.BankStatement{
		ID:            uuid.New(),
		TenantID:      tenantID,
		Format:        "camt053",
		Filename:      "queue-test.xml",
		ContentHash:   uuid.NewString(),
		AccountIBAN:   testIBAN,
		Currency:      "EUR",
		StatementDate: &now,
		CreatedAt:     now,
	}
	tx := &models.BankTransaction{
		ID:               uuid.New(),
		TenantID:         tenantID,
		StatementID:      stmt.ID,
		EntryRef:         fmt.Sprintf("REF-%s", uuid.NewString()[:8]),
		ValueDate:        now,
		Amount:           decimal.RequireFromString(amount),
		Currency:         "EUR",
		CounterpartyName: "Gruber Maschinenbau GmbH",
		RemittanceInfo:   "Abschlag 3",
		MatchStatus:      matchStatus,
		MatchedInvoiceID: matchedInvoiceID,
		CreatedAt:        now,
	}
	if err := repo.CreateStatement(ctx, stmt, []*models.BankTransaction{tx}); err != nil {
		t.Fatalf("CreateStatement: %v", err)
	}
	t.Cleanup(func() {
		testutil.CleanupRow(t, repo.pool, "finance_bank_transactions", tx.ID)
		testutil.CleanupRow(t, repo.pool, "finance_bank_statements", stmt.ID)
	})
	return tx
}

func TestPostgresRepository_TransactionCarriesTheInvoiceNumber(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	// t.Cleanup rather than defer: the row cleanups below register later and run
	// first (LIFO), so the pool is still open when they fire.
	t.Cleanup(pool.Close)

	// Private tenants rather than the shared TenantA/TenantB constants: these
	// tests count every transaction of their tenant, so a neighbour's fixture
	// under -parallel would make them flap.
	mine, theirs := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, mine, "Bank Queue Tenant Mine")
	testutil.EnsureTenant(t, pool, theirs, "Bank Queue Tenant Theirs")

	repo := NewPostgresRepository(pool)
	myCtx := testutil.WithTenantCtx(context.Background(), mine)
	theirCtx := testutil.WithTenantCtx(context.Background(), theirs)

	invoiceID := seedInvoice(t, pool, myCtx, mine, "RE-QUEUE-0001")
	matched := seedTransaction(t, myCtx, repo, mine, models.BankMatchMatched, &invoiceID, "119.00")
	unmatched := seedTransaction(t, myCtx, repo, mine, models.BankMatchUnmatched, nil, "-42.00")
	ignored := seedTransaction(t, myCtx, repo, mine, models.BankMatchIgnored, nil, "-9.90")
	foreign := seedTransaction(t, theirCtx, repo, theirs, models.BankMatchUnmatched, nil, "500.00")

	t.Run("the number comes back from the join", func(t *testing.T) {
		got, err := repo.GetTransaction(myCtx, mine, matched.ID)
		if err != nil {
			t.Fatalf("GetTransaction: %v", err)
		}
		if got.MatchedInvoiceNumber != "RE-QUEUE-0001" {
			t.Errorf("matched invoice number = %q, want RE-QUEUE-0001", got.MatchedInvoiceNumber)
		}
	})

	t.Run("nothing matched leaves the number empty", func(t *testing.T) {
		got, err := repo.GetTransaction(myCtx, mine, unmatched.ID)
		if err != nil {
			t.Fatalf("GetTransaction: %v", err)
		}
		if got.MatchedInvoiceNumber != "" {
			t.Errorf("matched invoice number = %q, want empty", got.MatchedInvoiceNumber)
		}
	})

	t.Run("the queue leaves out what was set aside", func(t *testing.T) {
		rows, total, err := repo.ListTransactions(myCtx, mine, models.BankTransactionFilter{
			ExcludeMatchStatus: []string{models.BankMatchIgnored},
			Limit:              50,
		})
		if err != nil {
			t.Fatalf("ListTransactions: %v", err)
		}
		if total != 2 {
			t.Fatalf("total = %d, want the two entries still needing a decision", total)
		}
		for _, row := range rows {
			if row.ID == ignored.ID {
				t.Error("an ignored entry surfaced in the queue")
			}
			if row.ID == foreign.ID {
				t.Error("another tenant's transaction surfaced")
			}
			if row.ID == matched.ID && row.MatchedInvoiceNumber != "RE-QUEUE-0001" {
				t.Errorf("list dropped the joined number: %q", row.MatchedInvoiceNumber)
			}
		}
	})

	t.Run("asking for ignored by name still returns it", func(t *testing.T) {
		rows, total, err := repo.ListTransactions(myCtx, mine, models.BankTransactionFilter{
			MatchStatus: models.BankMatchIgnored,
			Limit:       50,
		})
		if err != nil {
			t.Fatalf("ListTransactions: %v", err)
		}
		if total != 1 || len(rows) != 1 || rows[0].ID != ignored.ID {
			t.Fatalf("total = %d, rows = %d, want exactly the ignored entry", total, len(rows))
		}
	})

	t.Run("a foreign transaction is not readable", func(t *testing.T) {
		if _, err := repo.GetTransaction(myCtx, mine, foreign.ID); !errors.Is(err, ErrTransactionNotFound) {
			t.Fatalf("err = %v, want ErrTransactionNotFound", err)
		}
	})
}

func TestPostgresRepository_FindInvoiceIDByNumberStaysInTheTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	mine, theirs := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, mine, "Bank Lookup Tenant Mine")
	testutil.EnsureTenant(t, pool, theirs, "Bank Lookup Tenant Theirs")

	repo := NewPostgresRepository(pool)
	myCtx := testutil.WithTenantCtx(context.Background(), mine)
	theirCtx := testutil.WithTenantCtx(context.Background(), theirs)

	wanted := seedInvoice(t, pool, myCtx, mine, "RE-LOOKUP-0001")
	seedInvoice(t, pool, theirCtx, theirs, "RE-LOOKUP-FOREIGN")

	got, err := repo.FindInvoiceIDByNumber(myCtx, mine, "RE-LOOKUP-0001")
	if err != nil {
		t.Fatalf("FindInvoiceIDByNumber: %v", err)
	}
	if got != wanted {
		t.Errorf("id = %s, want %s", got, wanted)
	}

	// Another tenant's number must not resolve — booking a payment against it
	// would cross the tenant boundary with money.
	if _, err := repo.FindInvoiceIDByNumber(myCtx, mine, "RE-LOOKUP-FOREIGN"); !errors.Is(err, ErrInvoiceNotFound) {
		t.Fatalf("err = %v, want ErrInvoiceNotFound", err)
	}
}

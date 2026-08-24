package banking

// These tests run against a real database because the three things they prove
// cannot be proven against a fake: that every account query carries the tenant,
// that the RLS policy on finance_bank_accounts holds under kmuhub_app
// (NOSUPERUSER NOBYPASSRLS), and that the derived balance really comes out of
// the newest imported statement rather than out of a stored column.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

const testIBAN = "DE89370400440532013000"

func newAccountRow(t *testing.T, ctx context.Context, repo *PostgresRepository, tenantID uuid.UUID, bankName, iban string) *models.BankAccount {
	t.Helper()
	acc := &models.BankAccount{
		TenantID: tenantID,
		BankName: bankName,
		IBAN:     iban,
		BIC:      "COBADEFFXXX",
		Currency: "EUR",
	}
	if err := repo.CreateAccount(ctx, acc); err != nil {
		t.Fatalf("CreateAccount(%s): %v", bankName, err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, repo.pool, "finance_bank_accounts", acc.ID) })
	return acc
}

// seedStatement writes a bank statement for one IBAN. The balance an account
// reports is read from here, so this is what the derivation test drives.
func seedStatement(t *testing.T, pool *pgxpool.Pool, ctx context.Context, tenantID uuid.UUID, iban, closing string, date time.Time) {
	t.Helper()
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
		INSERT INTO finance_bank_statements
			(tenant_id, format, filename, content_hash, account_iban, currency,
			 closing_balance, statement_date)
		VALUES ($1, 'camt053', 'test.xml', $2, $3, 'EUR', $4::numeric, $5)
		RETURNING id`,
		tenantID, uuid.NewString(), iban, closing, date,
	).Scan(&id)
	if err != nil {
		t.Fatalf("seed statement (%s, %s): %v", iban, closing, err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_bank_statements", id) })
}

func TestPostgresRepository_AccountsAreTenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	// t.Cleanup rather than defer: the row cleanups below register later and run
	// first (LIFO), so the pool is still open when they fire.
	t.Cleanup(pool.Close)

	// Private tenants rather than the shared TenantA/TenantB constants: these
	// tests count every account of their tenant, and (tenant_id, iban) is
	// unique, so a neighbour's fixture under -parallel would make them flap.
	mine, theirs := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, mine, "Bank Account Tenant Mine")
	testutil.EnsureTenant(t, pool, theirs, "Bank Account Tenant Theirs")

	repo := NewPostgresRepository(pool)
	myCtx := testutil.WithTenantCtx(context.Background(), mine)
	theirCtx := testutil.WithTenantCtx(context.Background(), theirs)

	ours := newAccountRow(t, myCtx, repo, mine, "Commerzbank", testIBAN)
	// The same IBAN under another tenant: allowed, and it must stay invisible.
	foreign := newAccountRow(t, theirCtx, repo, theirs, "Commerzbank", testIBAN)

	t.Run("list stays inside the tenant", func(t *testing.T) {
		rows, err := repo.ListAccounts(myCtx, mine)
		if err != nil {
			t.Fatalf("ListAccounts: %v", err)
		}
		if len(rows) != 1 {
			t.Fatalf("listed %d accounts, want 1", len(rows))
		}
		if rows[0].ID != ours.ID {
			t.Errorf("listed %s, want %s", rows[0].ID, ours.ID)
		}
	})

	t.Run("a foreign id reads as not found", func(t *testing.T) {
		if _, err := repo.GetAccount(myCtx, mine, foreign.ID); !errors.Is(err, ErrAccountNotFound) {
			t.Errorf("GetAccount across tenants: err = %v, want ErrAccountNotFound", err)
		}
	})

	t.Run("a foreign id cannot be deleted", func(t *testing.T) {
		if err := repo.DeleteAccount(myCtx, mine, foreign.ID); !errors.Is(err, ErrAccountNotFound) {
			t.Errorf("DeleteAccount across tenants: err = %v, want ErrAccountNotFound", err)
		}
		if _, err := repo.GetAccount(theirCtx, theirs, foreign.ID); err != nil {
			t.Errorf("the foreign row went missing after a cross-tenant delete: %v", err)
		}
	})

	// No cleanup registered below: on the wanted path nothing is written.
	t.Run("a duplicate IBAN inside the tenant is refused", func(t *testing.T) {
		dup := &models.BankAccount{
			TenantID: mine, BankName: "Commerzbank Zweitkonto",
			IBAN: testIBAN, Currency: "EUR",
		}
		if err := repo.CreateAccount(myCtx, dup); !errors.Is(err, ErrAccountExists) {
			t.Fatalf("duplicate CreateAccount: err = %v, want ErrAccountExists", err)
		}
	})
}

// The balance is not a column. This drives the derivation end to end: no
// statement means zero, the newest statement wins, and a statement of another
// tenant is invisible even though it carries the same IBAN.
func TestPostgresRepository_AccountBalanceComesFromTheNewestStatement(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	// t.Cleanup rather than defer: the row cleanups below register later and run
	// first (LIFO), so the pool is still open when they fire.
	t.Cleanup(pool.Close)

	mine, theirs := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, mine, "Bank Balance Tenant Mine")
	testutil.EnsureTenant(t, pool, theirs, "Bank Balance Tenant Theirs")

	repo := NewPostgresRepository(pool)
	myCtx := testutil.WithTenantCtx(context.Background(), mine)
	theirCtx := testutil.WithTenantCtx(context.Background(), theirs)

	acc := newAccountRow(t, myCtx, repo, mine, "Commerzbank", testIBAN)

	read, err := repo.GetAccount(myCtx, mine, acc.ID)
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if !read.Balance.IsZero() || read.LastSync != nil {
		t.Fatalf("without an import: balance %s / lastSync %v, want 0 / nil",
			read.Balance, read.LastSync)
	}

	older := time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC)
	seedStatement(t, pool, myCtx, mine, testIBAN, "1000.00", older)
	seedStatement(t, pool, myCtx, mine, testIBAN, "142850.75", newer)
	// Same IBAN, other tenant, far larger figure: RLS and the tenant predicate
	// must both keep it out.
	seedStatement(t, pool, theirCtx, theirs, testIBAN, "999999.99", newer.AddDate(0, 1, 0))

	read, err = repo.GetAccount(myCtx, mine, acc.ID)
	if err != nil {
		t.Fatalf("GetAccount after import: %v", err)
	}
	if read.Balance.String() != "142850.75" {
		t.Errorf("balance = %s, want 142850.75", read.Balance)
	}
	if read.LastSync == nil || !read.LastSync.Equal(newer) {
		t.Errorf("lastSync = %v, want %v", read.LastSync, newer)
	}

	// The list must derive it the same way as the single read; the two run
	// through different queries and are exactly where they could drift apart.
	rows, err := repo.ListAccounts(myCtx, mine)
	if err != nil {
		t.Fatalf("ListAccounts: %v", err)
	}
	if len(rows) != 1 || rows[0].Balance.String() != "142850.75" {
		t.Fatalf("list reported %d rows with balance %v, want 1 / 142850.75",
			len(rows), rows)
	}
}

// UpdateAccount at the repository level has two error branches
// service_accounts_test.go's fake repo cannot exercise, because the fake never
// touches SQL: a row that plain does not exist, and a write that would collide
// with another row's unique IBAN. Both must be reported, not silently dropped.
func TestPostgresRepository_UpdateAccount_NotFoundAndIBANConflict(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Bank Account Update Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	first := newAccountRow(t, ctx, repo, tenant, "Commerzbank", testIBAN)
	secondIBAN := "DE02120300000000202051"
	second := newAccountRow(t, ctx, repo, tenant, "DKB", secondIBAN)

	t.Run("a row that does not exist is reported, not silently ignored", func(t *testing.T) {
		ghost := &models.BankAccount{
			TenantID: tenant, ID: uuid.New(), BankName: "Ghost",
			IBAN: "DE89370400440532013099", Currency: "EUR",
		}
		if err := repo.UpdateAccount(ctx, ghost); !errors.Is(err, ErrAccountNotFound) {
			t.Fatalf("UpdateAccount on a nonexistent id: err = %v, want ErrAccountNotFound", err)
		}
	})

	t.Run("moving onto another account's IBAN is refused, not silently merged", func(t *testing.T) {
		clash := *second
		clash.IBAN = first.IBAN
		if err := repo.UpdateAccount(ctx, &clash); !errors.Is(err, ErrAccountExists) {
			t.Fatalf("UpdateAccount onto a taken IBAN: err = %v, want ErrAccountExists", err)
		}
		reread, err := repo.GetAccount(ctx, tenant, second.ID)
		if err != nil {
			t.Fatalf("GetAccount after the refused update: %v", err)
		}
		if reread.IBAN != secondIBAN {
			t.Errorf("iban after a refused update = %s, want unchanged %s", reread.IBAN, secondIBAN)
		}
	})
}

// DeleteAccount's success path: the row is gone afterward, not just reported as
// deleted. integration_accounts_test.go's tenant-scoping test only exercises
// the not-found branch (a foreign id), never the row actually vanishing.
func TestPostgresRepository_DeleteAccount_RemovesTheRow(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Bank Account Delete Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	acc := &models.BankAccount{
		TenantID: tenant, BankName: "Sparkasse", IBAN: testIBAN, Currency: "EUR",
	}
	if err := repo.CreateAccount(ctx, acc); err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}

	if err := repo.DeleteAccount(ctx, tenant, acc.ID); err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}
	if _, err := repo.GetAccount(ctx, tenant, acc.ID); !errors.Is(err, ErrAccountNotFound) {
		t.Fatalf("GetAccount after delete: err = %v, want ErrAccountNotFound", err)
	}
	// The IBAN is free again: a delete that left a phantom unique-index entry
	// would refuse this and nobody could re-add the same account.
	again := &models.BankAccount{
		TenantID: tenant, BankName: "Sparkasse (erneut)", IBAN: testIBAN, Currency: "EUR",
	}
	if err := repo.CreateAccount(ctx, again); err != nil {
		t.Fatalf("re-creating the same IBAN after delete: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_bank_accounts", again.ID) })
}

// Service.ListAccounts itself is a one-line passthrough, but until this test it
// had zero coverage — every existing test read through the repository or the
// fake directly.
func TestService_ListAccounts(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Bank Account Service List Tenant")

	repo := NewPostgresRepository(pool)
	svc := NewService(repo, nil, nil, nil)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	acc := newAccountRow(t, ctx, repo, tenant, "Commerzbank", testIBAN)

	rows, err := svc.ListAccounts(ctx, tenant)
	if err != nil {
		t.Fatalf("ListAccounts: %v", err)
	}
	if len(rows) != 1 || rows[0].ID != acc.ID {
		t.Fatalf("ListAccounts = %+v, want exactly %s", rows, acc.ID)
	}
}

// The service path over real rows: connect persists, and a corrected IBAN moves
// the account onto that IBAN's statements instead of keeping the old balance.
func TestService_AccountConnectAndIBANCorrectionPersist(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	// t.Cleanup rather than defer: the row cleanups below register later and run
	// first (LIFO), so the pool is still open when they fire.
	t.Cleanup(pool.Close)

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Bank Account Service Tenant")

	repo := NewPostgresRepository(pool)
	svc := NewService(repo, nil, nil, nil)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	// Entered with a typo'd bank name and the wrong (but valid) IBAN.
	wrongIBAN := "DE02120300000000202051"
	acc, err := svc.CreateAccount(ctx, CreateAccountInput{
		TenantID: tenant, BankName: "Commerzbnak", IBAN: wrongIBAN,
	})
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "finance_bank_accounts", acc.ID) })

	seedStatement(t, pool, ctx, tenant, testIBAN, "500.25",
		time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC))

	if _, err := svc.ConnectAccount(ctx, tenant, acc.ID); err != nil {
		t.Fatalf("ConnectAccount: %v", err)
	}

	name, iban := "Commerzbank", testIBAN
	updated, err := svc.UpdateAccount(ctx, UpdateAccountInput{
		TenantID: tenant, ID: acc.ID, BankName: &name, IBAN: &iban,
	})
	if err != nil {
		t.Fatalf("UpdateAccount: %v", err)
	}
	if updated.Balance.String() != "500.25" {
		t.Errorf("balance after the IBAN correction = %s, want 500.25", updated.Balance)
	}
	// The connect survived an unrelated edit rather than being reset by it.
	if !updated.Connected || updated.ConnectedAt == nil {
		t.Errorf("connected = %v / %v after a master-data edit, want true / set",
			updated.Connected, updated.ConnectedAt)
	}

	reread, err := svc.GetAccount(ctx, tenant, acc.ID)
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if reread.BankName != name || reread.IBAN != testIBAN || !reread.Connected {
		t.Errorf("the edit did not persist: %+v", reread)
	}
}

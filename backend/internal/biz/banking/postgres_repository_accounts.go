package banking

// Bank accounts (Migration 000258) — the master data the imported statements
// attach to.

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

// accountColumns selects the stored fields plus the two derived ones.
//
// Balance and last sync come from the newest statement carrying this IBAN. The
// lateral join is what keeps them honest: nothing writes a balance into the
// account row, so the figure shown can only ever be one this system imported.
// A NULL closing balance (MT940 files sometimes omit it) reads as zero rather
// than dropping the account from the list.
const accountColumns = `
	a.id, a.tenant_id, a.bank_name, a.iban, a.bic, a.currency, a.connected,
	a.connected_at, a.created_at, a.updated_at,
	COALESCE(s.closing_balance, 0)::text, s.statement_date`

const accountFrom = `
	FROM finance_bank_accounts a
	LEFT JOIN LATERAL (
		SELECT closing_balance, statement_date
		FROM finance_bank_statements s
		WHERE s.tenant_id = a.tenant_id AND s.account_iban = a.iban
		ORDER BY s.statement_date DESC NULLS LAST, s.created_at DESC
		LIMIT 1
	) s ON TRUE`

func (r *PostgresRepository) CreateAccount(ctx context.Context, acc *models.BankAccount) error {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO finance_bank_accounts (
			tenant_id, bank_name, iban, bic, currency, connected, connected_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`,
		acc.TenantID, acc.BankName, acc.IBAN, acc.BIC, acc.Currency,
		acc.Connected, acc.ConnectedAt,
	).Scan(&acc.ID, &acc.CreatedAt, &acc.UpdatedAt)
	if err != nil {
		if isAccountIBANConflict(err) {
			return ErrAccountExists
		}
		return fmt.Errorf("insert bank account: %w", err)
	}
	// A brand-new account has no statement yet; both derived fields stay at
	// their zero value rather than being read back in a second query.
	acc.Balance = decimal.Zero
	acc.LastSync = nil
	return nil
}

func (r *PostgresRepository) GetAccount(ctx context.Context, tenantID, id uuid.UUID) (*models.BankAccount, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+accountColumns+accountFrom+`
		WHERE a.tenant_id = $1 AND a.id = $2`, tenantID, id)
	return scanAccount(row)
}

func (r *PostgresRepository) ListAccounts(ctx context.Context, tenantID uuid.UUID) ([]*models.BankAccount, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+accountColumns+accountFrom+`
		WHERE a.tenant_id = $1
		ORDER BY a.bank_name, a.iban`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list bank accounts: %w", err)
	}
	defer rows.Close()

	accounts := make([]*models.BankAccount, 0)
	for rows.Next() {
		acc, err := scanAccount(rows)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, acc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list bank accounts: %w", err)
	}
	return accounts, nil
}

func (r *PostgresRepository) UpdateAccount(ctx context.Context, acc *models.BankAccount) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE finance_bank_accounts
		SET bank_name = $3, iban = $4, bic = $5, currency = $6,
		    connected = $7, connected_at = $8, updated_at = NOW()
		WHERE tenant_id = $1 AND id = $2`,
		acc.TenantID, acc.ID, acc.BankName, acc.IBAN, acc.BIC, acc.Currency,
		acc.Connected, acc.ConnectedAt)
	if err != nil {
		if isAccountIBANConflict(err) {
			return ErrAccountExists
		}
		return fmt.Errorf("update bank account: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrAccountNotFound
	}
	return nil
}

func (r *PostgresRepository) DeleteAccount(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM finance_bank_accounts WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return fmt.Errorf("delete bank account: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrAccountNotFound
	}
	return nil
}

// isAccountIBANConflict reports whether err is the unique violation on
// (tenant_id, iban). Matching the constraint name rather than the bare SQLSTATE
// keeps a future second unique index from being reported as a duplicate IBAN.
func isAccountIBANConflict(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) &&
		pgErr.Code == "23505" &&
		pgErr.ConstraintName == "finance_bank_accounts_iban_unique"
}

func scanAccount(row pgx.Row) (*models.BankAccount, error) {
	var (
		acc     models.BankAccount
		balance string
	)
	err := row.Scan(
		&acc.ID, &acc.TenantID, &acc.BankName, &acc.IBAN, &acc.BIC, &acc.Currency,
		&acc.Connected, &acc.ConnectedAt, &acc.CreatedAt, &acc.UpdatedAt,
		&balance, &acc.LastSync,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAccountNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan bank account: %w", err)
	}
	acc.Balance, err = decimal.NewFromString(balance)
	if err != nil {
		return nil, fmt.Errorf("scan bank account balance: %w", err)
	}
	return &acc, nil
}

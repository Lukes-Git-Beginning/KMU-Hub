package banking

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

// PostgresRepository stores bank statements in finance_bank_statements and
// finance_bank_transactions (Migration 000247).
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates the repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

const statementColumns = `
	id, tenant_id, format, filename, content_hash, account_iban, statement_ref,
	currency, opening_balance::text, closing_balance::text, statement_date,
	transaction_count, imported_by, created_at`

const transactionColumns = `
	id, tenant_id, statement_id, entry_ref, end_to_end_id, value_date,
	booking_date, amount::text, currency, counterparty_name, counterparty_iban,
	remittance_info, match_status, match_reason, matched_invoice_id, payment_id,
	reconciled_at, reconciled_by, created_at`

func (r *PostgresRepository) GetStatementByHash(ctx context.Context, tenantID uuid.UUID, hash string) (*models.BankStatement, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT `+statementColumns+`
		FROM finance_bank_statements
		WHERE tenant_id = $1 AND content_hash = $2`, tenantID, hash)
	return scanStatement(row)
}

func (r *PostgresRepository) GetStatement(ctx context.Context, tenantID, id uuid.UUID) (*models.BankStatement, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT `+statementColumns+`
		FROM finance_bank_statements
		WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	return scanStatement(row)
}

func (r *PostgresRepository) ListStatements(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*models.BankStatement, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM finance_bank_statements WHERE tenant_id = $1`, tenantID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count statements: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT `+statementColumns+`
		FROM finance_bank_statements
		WHERE tenant_id = $1
		ORDER BY created_at DESC, id
		LIMIT $2 OFFSET $3`, tenantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list statements: %w", err)
	}
	defer rows.Close()

	// Non-nil slice: an empty list must serialise as [] and not as null.
	out := make([]*models.BankStatement, 0, limit)
	for rows.Next() {
		stmt, scanErr := scanStatement(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		out = append(out, stmt)
	}
	return out, total, rows.Err()
}

// CreateStatement writes the statement and its transactions atomically. Without
// the transaction a crash between the two writes would leave a statement whose
// transaction_count promises entries nobody can read.
func (r *PostgresRepository) CreateStatement(ctx context.Context, stmt *models.BankStatement, txs []*models.BankTransaction) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err = tx.Exec(ctx, `
		INSERT INTO finance_bank_statements (
			id, tenant_id, format, filename, content_hash, account_iban,
			statement_ref, currency, opening_balance, closing_balance,
			statement_date, transaction_count, imported_by, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		stmt.ID, stmt.TenantID, stmt.Format, stmt.Filename, stmt.ContentHash,
		stmt.AccountIBAN, stmt.StatementRef, stmt.Currency,
		decimalPtrText(stmt.OpeningBalance), decimalPtrText(stmt.ClosingBalance),
		stmt.StatementDate, stmt.TransactionCount, stmt.ImportedBy, stmt.CreatedAt,
	); err != nil {
		return fmt.Errorf("insert statement: %w", err)
	}

	for _, t := range txs {
		if _, err = tx.Exec(ctx, `
			INSERT INTO finance_bank_transactions (
				id, tenant_id, statement_id, entry_ref, end_to_end_id, value_date,
				booking_date, amount, currency, counterparty_name,
				counterparty_iban, remittance_info, match_status, match_reason,
				matched_invoice_id, created_at
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
			t.ID, t.TenantID, t.StatementID, t.EntryRef, t.EndToEndID, t.ValueDate,
			t.BookingDate, t.Amount.String(), t.Currency, t.CounterpartyName,
			t.CounterpartyIBAN, t.RemittanceInfo, t.MatchStatus, t.MatchReason,
			t.MatchedInvoiceID, t.CreatedAt,
		); err != nil {
			return fmt.Errorf("insert transaction: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepository) GetTransaction(ctx context.Context, tenantID, id uuid.UUID) (*models.BankTransaction, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT `+transactionColumns+`
		FROM finance_bank_transactions
		WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	return scanTransaction(row)
}

func (r *PostgresRepository) ListTransactions(ctx context.Context, tenantID uuid.UUID, filter models.BankTransactionFilter) ([]*models.BankTransaction, int, error) {
	args := []any{tenantID}
	var conditions []string
	if filter.StatementID != nil {
		args = append(args, *filter.StatementID)
		conditions = append(conditions, fmt.Sprintf("statement_id = $%d", len(args)))
	}
	if filter.MatchStatus != "" {
		args = append(args, filter.MatchStatus)
		conditions = append(conditions, fmt.Sprintf("match_status = $%d", len(args)))
	}
	where := "WHERE tenant_id = $1"
	if len(conditions) > 0 {
		where += " AND " + strings.Join(conditions, " AND ")
	}

	var total int
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM finance_bank_transactions `+where, args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count transactions: %w", err)
	}

	args = append(args, filter.Limit, filter.Offset)
	rows, err := r.pool.Query(ctx, `
		SELECT `+transactionColumns+`
		FROM finance_bank_transactions `+where+`
		ORDER BY value_date DESC, created_at DESC, id
		LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list transactions: %w", err)
	}
	defer rows.Close()

	out := make([]*models.BankTransaction, 0, filter.Limit)
	for rows.Next() {
		t, scanErr := scanTransaction(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		out = append(out, t)
	}
	return out, total, rows.Err()
}

func (r *PostgresRepository) ListTransactionsByStatement(ctx context.Context, tenantID, statementID uuid.UUID) ([]*models.BankTransaction, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+transactionColumns+`
		FROM finance_bank_transactions
		WHERE tenant_id = $1 AND statement_id = $2
		ORDER BY value_date, created_at, id`, tenantID, statementID)
	if err != nil {
		return nil, fmt.Errorf("list transactions of statement: %w", err)
	}
	defer rows.Close()

	out := make([]*models.BankTransaction, 0)
	for rows.Next() {
		t, scanErr := scanTransaction(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// UpdateTransactionMatch writes only the reconciliation columns: what the bank
// reported is never rewritten after import.
func (r *PostgresRepository) UpdateTransactionMatch(ctx context.Context, t *models.BankTransaction) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE finance_bank_transactions
		SET match_status = $3, match_reason = $4, matched_invoice_id = $5,
		    payment_id = $6, reconciled_at = $7, reconciled_by = $8
		WHERE tenant_id = $1 AND id = $2`,
		t.TenantID, t.ID, t.MatchStatus, t.MatchReason, t.MatchedInvoiceID,
		t.PaymentID, t.ReconciledAt, t.ReconciledBy)
	if err != nil {
		return fmt.Errorf("update transaction match: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrTransactionNotFound
	}
	return nil
}

// rowScanner is satisfied by both pgx.Row and pgx.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanStatement(row rowScanner) (*models.BankStatement, error) {
	var (
		s              models.BankStatement
		openingBalance *string
		closingBalance *string
	)
	err := row.Scan(
		&s.ID, &s.TenantID, &s.Format, &s.Filename, &s.ContentHash, &s.AccountIBAN,
		&s.StatementRef, &s.Currency, &openingBalance, &closingBalance,
		&s.StatementDate, &s.TransactionCount, &s.ImportedBy, &s.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrStatementNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan statement: %w", err)
	}
	if s.OpeningBalance, err = parseDecimalPtr(openingBalance); err != nil {
		return nil, err
	}
	if s.ClosingBalance, err = parseDecimalPtr(closingBalance); err != nil {
		return nil, err
	}
	return &s, nil
}

func scanTransaction(row rowScanner) (*models.BankTransaction, error) {
	var (
		t      models.BankTransaction
		amount string
	)
	err := row.Scan(
		&t.ID, &t.TenantID, &t.StatementID, &t.EntryRef, &t.EndToEndID, &t.ValueDate,
		&t.BookingDate, &amount, &t.Currency, &t.CounterpartyName, &t.CounterpartyIBAN,
		&t.RemittanceInfo, &t.MatchStatus, &t.MatchReason, &t.MatchedInvoiceID,
		&t.PaymentID, &t.ReconciledAt, &t.ReconciledBy, &t.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTransactionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan transaction: %w", err)
	}
	parsed, err := parseDecimal(amount)
	if err != nil {
		return nil, err
	}
	t.Amount = parsed
	return &t, nil
}

// parseDecimal reads a NUMERIC that was selected as text. An empty column is a
// zero; anything else that fails to parse is an error, never a silent zero — a
// misread amount is the one thing this whole feature must not produce.
func parseDecimal(s string) (decimal.Decimal, error) {
	if strings.TrimSpace(s) == "" {
		return decimal.Zero, nil
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero, fmt.Errorf("parse decimal %q: %w", s, err)
	}
	return d, nil
}

// parseDecimalPtr is parseDecimal for a nullable column.
func parseDecimalPtr(s *string) (*decimal.Decimal, error) {
	if s == nil {
		return nil, nil
	}
	d, err := parseDecimal(*s)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// decimalPtrText renders a nullable decimal for the driver. NUMERIC is fed a
// string so the value never passes through a float.
func decimalPtrText(d *decimal.Decimal) *string {
	if d == nil {
		return nil
	}
	s := d.String()
	return &s
}

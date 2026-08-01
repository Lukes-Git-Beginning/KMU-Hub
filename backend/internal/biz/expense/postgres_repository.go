package expense

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

// PostgresRepository stores expenses in finance_expenses (Migration 000257).
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates the repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// amount is read as text so the NUMERIC(15,2) never passes through a float.
const expenseColumns = `
	id, tenant_id, description, amount::text, expense_date, category, supplier,
	project, account, receipt_name, status, submitted_by, decided_by, decided_at,
	created_at, updated_at`

type rowScanner interface {
	Scan(dest ...any) error
}

func (r *PostgresRepository) Create(ctx context.Context, e *models.Expense) error {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO finance_expenses (
			tenant_id, description, amount, expense_date, category, supplier,
			project, account, receipt_name, status, submitted_by
		) VALUES ($1, $2, $3::numeric, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING `+expenseColumns,
		e.TenantID, e.Description, e.Amount.StringFixed(2), e.ExpenseDate,
		e.Category, e.Supplier, e.Project, e.Account, e.ReceiptName,
		e.Status, e.SubmittedBy)

	created, err := scanExpense(row)
	if err != nil {
		return err
	}
	*e = *created
	return nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Expense, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT `+expenseColumns+`
		FROM finance_expenses
		WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	return scanExpense(row)
}

// Update writes the mutable columns. The tenant is part of the predicate, so an
// expense of another tenant is a no-op that surfaces as ErrNotFound rather than
// as a silent cross-tenant write.
func (r *PostgresRepository) Update(ctx context.Context, e *models.Expense) error {
	row := r.pool.QueryRow(ctx, `
		UPDATE finance_expenses SET
			description  = $3,
			amount       = $4::numeric,
			expense_date = $5,
			category     = $6,
			supplier     = $7,
			project      = $8,
			account      = $9,
			receipt_name = $10,
			status       = $11,
			decided_by   = $12,
			decided_at   = $13,
			updated_at   = NOW()
		WHERE tenant_id = $1 AND id = $2
		RETURNING `+expenseColumns,
		e.TenantID, e.ID, e.Description, e.Amount.StringFixed(2), e.ExpenseDate,
		e.Category, e.Supplier, e.Project, e.Account, e.ReceiptName, e.Status,
		e.DecidedBy, e.DecidedAt)

	updated, err := scanExpense(row)
	if err != nil {
		return err
	}
	*e = *updated
	return nil
}

func (r *PostgresRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM finance_expenses WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return fmt.Errorf("delete expense: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// List returns one page plus the total number of rows matching the same
// filter -- the count runs against the identical WHERE clause so paging and
// total describe the same set.
func (r *PostgresRepository) List(ctx context.Context, tenantID uuid.UUID, filter ListFilter) ([]*models.Expense, int, error) {
	var (
		where strings.Builder
		args  = []any{tenantID}
	)
	where.WriteString("tenant_id = $1")
	if filter.Status != "" {
		args = append(args, filter.Status)
		fmt.Fprintf(&where, " AND status = $%d", len(args))
	}
	if filter.SubmittedBy != nil {
		args = append(args, *filter.SubmittedBy)
		fmt.Fprintf(&where, " AND submitted_by = $%d", len(args))
	}

	var total int
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM finance_expenses WHERE `+where.String(), args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count expenses: %w", err)
	}

	query := `SELECT ` + expenseColumns + ` FROM finance_expenses WHERE ` + where.String() +
		` ORDER BY expense_date DESC, created_at DESC`
	if filter.Limit > 0 {
		args = append(args, filter.Limit)
		query += fmt.Sprintf(" LIMIT $%d", len(args))
	}
	if filter.Offset > 0 {
		args = append(args, filter.Offset)
		query += fmt.Sprintf(" OFFSET $%d", len(args))
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list expenses: %w", err)
	}
	defer rows.Close()

	expenses := make([]*models.Expense, 0, filter.Limit)
	for rows.Next() {
		e, scanErr := scanExpense(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		expenses = append(expenses, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate expenses: %w", err)
	}
	return expenses, total, nil
}

func scanExpense(row rowScanner) (*models.Expense, error) {
	var (
		e      models.Expense
		amount string
	)
	err := row.Scan(
		&e.ID, &e.TenantID, &e.Description, &amount, &e.ExpenseDate, &e.Category,
		&e.Supplier, &e.Project, &e.Account, &e.ReceiptName, &e.Status,
		&e.SubmittedBy, &e.DecidedBy, &e.DecidedAt, &e.CreatedAt, &e.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan expense: %w", err)
	}
	if e.Amount, err = decimal.NewFromString(amount); err != nil {
		return nil, fmt.Errorf("parse expense amount %q: %w", amount, err)
	}
	return &e, nil
}

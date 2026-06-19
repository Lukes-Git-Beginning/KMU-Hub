package payment

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL payment repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, p *models.Payment) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO finance_payments (
			id, tenant_id, invoice_id, amount, payment_date,
			method, reference, notes, created_by, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		p.ID, p.TenantID, p.InvoiceID, p.Amount, p.PaymentDate,
		p.Method, p.Reference, p.Notes, p.CreatedBy, p.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) List(ctx context.Context, tenantID, invoiceID uuid.UUID) ([]*models.Payment, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, invoice_id, amount, payment_date,
			method, reference, notes, created_by, created_at
		FROM finance_payments
		WHERE tenant_id = $1 AND invoice_id = $2
		ORDER BY payment_date DESC, created_at DESC`,
		tenantID, invoiceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var payments []*models.Payment
	for rows.Next() {
		p, scanErr := r.scanPaymentFromRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		payments = append(payments, p)
	}
	return payments, rows.Err()
}

func (r *PostgresRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM finance_payments WHERE tenant_id = $1 AND id = $2`,
		tenantID, id,
	)
	return err
}

// SumByInvoiceID returns the total of all payments for a given invoice.
// The sum is scanned as a string and parsed via decimal to avoid float64
// precision loss on monetary values (ADR-0007). COALESCE keeps the result
// non-NULL when no payments exist.
func (r *PostgresRepository) SumByInvoiceID(ctx context.Context, tenantID, invoiceID uuid.UUID) (decimal.Decimal, error) {
	var sumStr string
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount), 0)::text FROM finance_payments WHERE tenant_id = $1 AND invoice_id = $2`,
		tenantID, invoiceID,
	).Scan(&sumStr)
	if err != nil {
		return decimal.Zero, err
	}
	sum, parseErr := decimal.NewFromString(sumStr)
	if parseErr != nil {
		return decimal.Zero, fmt.Errorf("parse payment sum %q: %w", sumStr, parseErr)
	}
	return sum, nil
}

// GetByID retrieves a single payment by ID.
func (r *PostgresRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Payment, error) {
	var p models.Payment
	var amountStr string
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, invoice_id, amount, payment_date,
			method, reference, notes, created_by, created_at
		FROM finance_payments
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id,
	).Scan(
		&p.ID, &p.TenantID, &p.InvoiceID, &amountStr, &p.PaymentDate,
		&p.Method, &p.Reference, &p.Notes, &p.CreatedBy, &p.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("payment not found")
	}
	if err != nil {
		return nil, err
	}
	// Scan amount as string + decimal to avoid float64 precision loss (ADR-0007).
	p.Amount, err = decimal.NewFromString(amountStr)
	if err != nil {
		return nil, fmt.Errorf("parse payment amount %q: %w", amountStr, err)
	}
	return &p, nil
}

func (r *PostgresRepository) scanPaymentFromRows(rows pgx.Rows) (*models.Payment, error) {
	var p models.Payment
	var amountStr string
	err := rows.Scan(
		&p.ID, &p.TenantID, &p.InvoiceID, &amountStr, &p.PaymentDate,
		&p.Method, &p.Reference, &p.Notes, &p.CreatedBy, &p.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	// Scan amount as string + decimal to avoid float64 precision loss (ADR-0007).
	p.Amount, err = decimal.NewFromString(amountStr)
	if err != nil {
		return nil, fmt.Errorf("parse payment amount %q: %w", amountStr, err)
	}
	return &p, nil
}

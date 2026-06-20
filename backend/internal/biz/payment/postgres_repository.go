package payment

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

// pgxExecutor is satisfied by both *pgxpool.Pool and pgx.Tx, letting the write
// helpers run either standalone (pool) or inside a caller-owned transaction.
type pgxExecutor interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL payment repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, p *models.Payment) error {
	return createPayment(ctx, r.pool, p)
}

// CreateInTx records a payment within the caller's transaction (atomic with the
// coupled invoice status transition).
func (r *PostgresRepository) CreateInTx(ctx context.Context, tx pgx.Tx, p *models.Payment) error {
	return createPayment(ctx, tx, p)
}

func createPayment(ctx context.Context, db pgxExecutor, p *models.Payment) error {
	// Persist the idempotency key as NULL when absent so payments without a key
	// never collide on the partial unique index (F5).
	var keyArg any
	if p.IdempotencyKey != "" {
		keyArg = p.IdempotencyKey
	}
	tag, err := db.Exec(ctx,
		`INSERT INTO finance_payments (
			id, tenant_id, invoice_id, amount, payment_date,
			method, reference, notes, created_by, created_at, idempotency_key
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (tenant_id, idempotency_key) WHERE idempotency_key IS NOT NULL DO NOTHING`,
		p.ID, p.TenantID, p.InvoiceID, p.Amount, p.PaymentDate,
		p.Method, p.Reference, p.Notes, p.CreatedBy, p.CreatedAt, keyArg,
	)
	if err != nil {
		return err
	}
	// A keyed insert that affected no row hit the dedup guard: a payment with this
	// key already exists. Signal the caller to return the existing one.
	if p.IdempotencyKey != "" && tag.RowsAffected() == 0 {
		return ErrDuplicatePayment
	}
	return nil
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

// DeleteInTx removes a payment within the caller's transaction (atomic with the
// coupled invoice status re-evaluation).
func (r *PostgresRepository) DeleteInTx(ctx context.Context, tx pgx.Tx, tenantID, id uuid.UUID) error {
	_, err := tx.Exec(ctx,
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
	return sumByInvoiceID(ctx, r.pool, tenantID, invoiceID)
}

// SumByInvoiceIDInTx totals the invoice payments within the caller's transaction so
// the paid/revert decision sees the same snapshot as the payment write.
func (r *PostgresRepository) SumByInvoiceIDInTx(ctx context.Context, tx pgx.Tx, tenantID, invoiceID uuid.UUID) (decimal.Decimal, error) {
	return sumByInvoiceID(ctx, tx, tenantID, invoiceID)
}

func sumByInvoiceID(ctx context.Context, db pgxExecutor, tenantID, invoiceID uuid.UUID) (decimal.Decimal, error) {
	var sumStr string
	err := db.QueryRow(ctx,
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

// GetByIdempotencyKey returns the payment recorded with the given idempotency
// key for the tenant, or (nil, nil) if none exists.
func (r *PostgresRepository) GetByIdempotencyKey(ctx context.Context, tenantID uuid.UUID, key string) (*models.Payment, error) {
	if key == "" {
		return nil, nil
	}
	var p models.Payment
	var amountStr string
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, invoice_id, amount, payment_date,
			method, reference, notes, created_by, created_at
		FROM finance_payments
		WHERE tenant_id = $1 AND idempotency_key = $2`,
		tenantID, key,
	).Scan(
		&p.ID, &p.TenantID, &p.InvoiceID, &amountStr, &p.PaymentDate,
		&p.Method, &p.Reference, &p.Notes, &p.CreatedBy, &p.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.Amount, err = decimal.NewFromString(amountStr)
	if err != nil {
		return nil, fmt.Errorf("parse payment amount %q: %w", amountStr, err)
	}
	p.IdempotencyKey = key
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

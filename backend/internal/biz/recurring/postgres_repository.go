package recurring

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// recurringColumns is the shared projection, kept in one place so it never drifts
// from the scan order below.
const recurringColumns = "id, tenant_id, title, " +
	"customer_name, customer_address, customer_email, customer_ust_id_nr, " +
	"tax_mode, line_items, tax_breakdown, currency, " +
	"recurrence_interval, status, start_date, end_date, next_run, " +
	"payment_terms_days, generated_count, last_generated_at, notes, " +
	"created_by, created_at, updated_at"

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL recurring invoice repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, rec *models.RecurringInvoice) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO finance_recurring_invoices (
			id, tenant_id, title,
			customer_name, customer_address, customer_email, customer_ust_id_nr,
			tax_mode, line_items, tax_breakdown, currency,
			recurrence_interval, status, start_date, end_date, next_run,
			payment_terms_days, generated_count, last_generated_at, notes,
			created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3,
			$4, $5, $6, $7,
			$8, $9, $10, $11,
			$12, $13, $14, $15, $16,
			$17, $18, $19, $20,
			$21, $22, $23
		)`,
		rec.ID, rec.TenantID, rec.Title,
		rec.CustomerName, rec.CustomerAddress, rec.CustomerEmail, rec.CustomerUStIDNr,
		rec.TaxMode, rec.LineItems, rec.TaxBreakdownRaw, rec.Currency,
		rec.Interval, rec.Status, rec.StartDate, rec.EndDate, rec.NextRun,
		rec.PaymentTermsDays, rec.GeneratedCount, rec.LastGeneratedAt, rec.Notes,
		rec.CreatedBy, rec.CreatedAt, rec.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert finance_recurring_invoices: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.RecurringInvoice, error) {
	row := r.pool.QueryRow(ctx,
		"SELECT "+recurringColumns+" FROM finance_recurring_invoices WHERE tenant_id = $1 AND id = $2",
		tenantID, id,
	)
	rec, err := scanRecurring(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return rec, err
}

func (r *PostgresRepository) List(ctx context.Context, tenantID uuid.UUID, filter ListFilter) ([]*models.RecurringInvoice, int, error) {
	conditions := []string{"tenant_id = $1"}
	args := []any{tenantID}
	argNum := 2

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argNum))
		args = append(args, filter.Status)
		argNum++
	}
	where := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM finance_recurring_invoices "+where, args...,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	offset := max(filter.Offset, 0)

	query := fmt.Sprintf(
		"SELECT %s FROM finance_recurring_invoices %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
		recurringColumns, where, argNum, argNum+1,
	)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	// Empty result must stay an empty slice: the wire shape is a JSON array, never null.
	items := make([]*models.RecurringInvoice, 0, limit)
	for rows.Next() {
		rec, scanErr := scanRecurring(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		items = append(items, rec)
	}
	return items, total, rows.Err()
}

func (r *PostgresRepository) Update(ctx context.Context, rec *models.RecurringInvoice) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE finance_recurring_invoices SET
			title = $3,
			customer_name = $4, customer_address = $5, customer_email = $6, customer_ust_id_nr = $7,
			tax_mode = $8, line_items = $9, tax_breakdown = $10, currency = $11,
			recurrence_interval = $12, status = $13, start_date = $14, end_date = $15, next_run = $16,
			payment_terms_days = $17, generated_count = $18, last_generated_at = $19, notes = $20,
			updated_at = $21
		WHERE tenant_id = $1 AND id = $2`,
		rec.TenantID, rec.ID, rec.Title,
		rec.CustomerName, rec.CustomerAddress, rec.CustomerEmail, rec.CustomerUStIDNr,
		rec.TaxMode, rec.LineItems, rec.TaxBreakdownRaw, rec.Currency,
		rec.Interval, rec.Status, rec.StartDate, rec.EndDate, rec.NextRun,
		rec.PaymentTermsDays, rec.GeneratedCount, rec.LastGeneratedAt, rec.Notes,
		rec.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update finance_recurring_invoices: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		"DELETE FROM finance_recurring_invoices WHERE tenant_id = $1 AND id = $2",
		tenantID, id,
	)
	if err != nil {
		return fmt.Errorf("delete finance_recurring_invoices: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) ClaimPeriod(ctx context.Context, tenantID, recurringID uuid.UUID, period time.Time) (*Run, bool, error) {
	var run Run
	err := r.pool.QueryRow(ctx,
		`INSERT INTO finance_recurring_runs (tenant_id, recurring_id, period_date)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (tenant_id, recurring_id, period_date) DO NOTHING
		 RETURNING id, tenant_id, recurring_id, period_date, invoice_id, created_at`,
		tenantID, recurringID, period,
	).Scan(&run.ID, &run.TenantID, &run.RecurringID, &run.PeriodDate, &run.InvoiceID, &run.CreatedAt)
	if err == nil {
		return &run, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, false, fmt.Errorf("claim recurring period: %w", err)
	}

	// DO NOTHING swallowed the insert: this period was already claimed. Return the
	// existing run so the caller can answer with the invoice that already exists.
	err = r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, recurring_id, period_date, invoice_id, created_at
		 FROM finance_recurring_runs
		 WHERE tenant_id = $1 AND recurring_id = $2 AND period_date = $3`,
		tenantID, recurringID, period,
	).Scan(&run.ID, &run.TenantID, &run.RecurringID, &run.PeriodDate, &run.InvoiceID, &run.CreatedAt)
	if err != nil {
		return nil, false, fmt.Errorf("load existing recurring run: %w", err)
	}
	return &run, false, nil
}

func (r *PostgresRepository) AttachInvoice(ctx context.Context, tenantID, runID, invoiceID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE finance_recurring_runs SET invoice_id = $3 WHERE tenant_id = $1 AND id = $2",
		tenantID, runID, invoiceID,
	)
	if err != nil {
		return fmt.Errorf("attach invoice to recurring run: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ReleasePeriod(ctx context.Context, tenantID, runID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		"DELETE FROM finance_recurring_runs WHERE tenant_id = $1 AND id = $2 AND invoice_id IS NULL",
		tenantID, runID,
	)
	if err != nil {
		return fmt.Errorf("release recurring run: %w", err)
	}
	return nil
}

// rowScanner is satisfied by both pgx.Row and pgx.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanRecurring(row rowScanner) (*models.RecurringInvoice, error) {
	var rec models.RecurringInvoice
	err := row.Scan(
		&rec.ID, &rec.TenantID, &rec.Title,
		&rec.CustomerName, &rec.CustomerAddress, &rec.CustomerEmail, &rec.CustomerUStIDNr,
		&rec.TaxMode, &rec.LineItems, &rec.TaxBreakdownRaw, &rec.Currency,
		&rec.Interval, &rec.Status, &rec.StartDate, &rec.EndDate, &rec.NextRun,
		&rec.PaymentTermsDays, &rec.GeneratedCount, &rec.LastGeneratedAt, &rec.Notes,
		&rec.CreatedBy, &rec.CreatedAt, &rec.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

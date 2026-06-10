package einvoice

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL incoming invoice repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// Create inserts a new incoming invoice record.
func (r *PostgresRepository) Create(ctx context.Context, inv *models.IncomingInvoice) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO finance_incoming_invoices (
			id, tenant_id,
			supplier_name, supplier_address, supplier_vat_id,
			invoice_number, invoice_date, due_date,
			currency, line_items, tax_breakdown,
			subtotal, total_tax, gross_total,
			source_format, original_xml, original_filename,
			status, created_by, created_at, updated_at
		) VALUES (
			$1, $2,
			$3, $4, $5,
			$6, $7, $8,
			$9, $10, $11,
			$12, $13, $14,
			$15, $16, $17,
			$18, $19, $20, $21
		)`,
		inv.ID, inv.TenantID,
		inv.SupplierName, inv.SupplierAddress, inv.SupplierVATID,
		inv.InvoiceNumber, inv.InvoiceDate, inv.DueDate,
		inv.Currency, inv.LineItems, inv.TaxBreakdownRaw,
		inv.Subtotal, inv.TotalTax, inv.GrossTotal,
		inv.SourceFormat, inv.OriginalXML, inv.OriginalFilename,
		inv.Status, inv.CreatedBy, inv.CreatedAt, inv.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrDuplicateInvoice
		}
		return fmt.Errorf("insert finance_incoming_invoices: %w", err)
	}
	return nil
}

// GetByID retrieves a single incoming invoice scoped to the tenant.
func (r *PostgresRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.IncomingInvoice, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT
			id, tenant_id,
			supplier_name, supplier_address, supplier_vat_id,
			invoice_number, invoice_date, due_date,
			currency, line_items, tax_breakdown,
			subtotal, total_tax, gross_total,
			source_format, original_xml, original_filename,
			status, created_by, created_at, updated_at
		FROM finance_incoming_invoices
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id,
	)
	return scanIncomingInvoice(row)
}

// List retrieves incoming invoices with optional filters, returning the page and total count.
func (r *PostgresRepository) List(ctx context.Context, tenantID uuid.UUID, filter ListFilter) ([]*models.IncomingInvoice, int, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}

	args := []any{tenantID}
	conditions := []string{"tenant_id = $1"}
	argIdx := 2

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}
	if filter.DateFrom != nil {
		conditions = append(conditions, fmt.Sprintf("invoice_date >= $%d", argIdx))
		args = append(args, filter.DateFrom)
		argIdx++
	}
	if filter.DateTo != nil {
		conditions = append(conditions, fmt.Sprintf("invoice_date <= $%d", argIdx))
		args = append(args, filter.DateTo)
		argIdx++
	}

	where := strings.Join(conditions, " AND ")

	// Total count
	var total int
	if err := r.pool.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM finance_incoming_invoices WHERE %s", where),
		args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count finance_incoming_invoices: %w", err)
	}

	// Data query
	dataArgs := append(args, limit, filter.Offset)
	rows, err := r.pool.Query(ctx,
		fmt.Sprintf(`SELECT
			id, tenant_id,
			supplier_name, supplier_address, supplier_vat_id,
			invoice_number, invoice_date, due_date,
			currency, line_items, tax_breakdown,
			subtotal, total_tax, gross_total,
			source_format, original_xml, original_filename,
			status, created_by, created_at, updated_at
		FROM finance_incoming_invoices
		WHERE %s
		ORDER BY invoice_date DESC, created_at DESC
		LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1),
		dataArgs...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list finance_incoming_invoices: %w", err)
	}
	defer rows.Close()

	var result []*models.IncomingInvoice
	for rows.Next() {
		inv, scanErr := scanIncomingInvoiceRow(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		result = append(result, inv)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate finance_incoming_invoices: %w", err)
	}
	return result, total, nil
}

// UpdateStatus updates the status and updated_at for a single incoming invoice.
func (r *PostgresRepository) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status string, updatedAt time.Time) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE finance_incoming_invoices
		 SET status = $3, updated_at = $4
		 WHERE tenant_id = $1 AND id = $2`,
		tenantID, id, status, updatedAt,
	)
	if err != nil {
		return fmt.Errorf("update status finance_incoming_invoices: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ============================================================================
// Scan helpers
// ============================================================================

// rowScanner is implemented by both pgx.Row and pgx.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanIncomingInvoice(row rowScanner) (*models.IncomingInvoice, error) {
	inv, err := scanIncomingInvoiceRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return inv, nil
}

func scanIncomingInvoiceRow(row rowScanner) (*models.IncomingInvoice, error) {
	var inv models.IncomingInvoice
	err := row.Scan(
		&inv.ID, &inv.TenantID,
		&inv.SupplierName, &inv.SupplierAddress, &inv.SupplierVATID,
		&inv.InvoiceNumber, &inv.InvoiceDate, &inv.DueDate,
		&inv.Currency, &inv.LineItems, &inv.TaxBreakdownRaw,
		&inv.Subtotal, &inv.TotalTax, &inv.GrossTotal,
		&inv.SourceFormat, &inv.OriginalXML, &inv.OriginalFilename,
		&inv.Status, &inv.CreatedBy, &inv.CreatedAt, &inv.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan incoming invoice: %w", err)
	}
	return &inv, nil
}

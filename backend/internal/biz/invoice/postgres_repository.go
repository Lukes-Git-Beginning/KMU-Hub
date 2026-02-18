package invoice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

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

// NewPostgresRepository creates a new PostgreSQL invoice repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, inv *models.Invoice) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO finance_invoices (
			id, tenant_id, invoice_number, status,
			customer_name, customer_address, customer_email, customer_ust_id_nr,
			company_snapshot, tax_mode, line_items, tax_breakdown,
			subtotal, total_tax, gross_total,
			invoice_date, delivery_date, due_date, payment_terms,
			snapshot_data, source_quote_id, notes,
			created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8,
			$9, $10, $11, $12,
			$13, $14, $15,
			$16, $17, $18, $19,
			$20, $21, $22,
			$23, $24, $25
		)`,
		inv.ID, inv.TenantID, inv.InvoiceNumber, inv.Status,
		inv.CustomerName, inv.CustomerAddress, inv.CustomerEmail, inv.CustomerUStIDNr,
		inv.CompanySnapshotRaw, inv.TaxMode, inv.LineItems, inv.TaxBreakdownRaw,
		inv.Subtotal, inv.TotalTax, inv.GrossTotal,
		inv.InvoiceDate, inv.DeliveryDate, inv.DueDate, inv.PaymentTerms,
		inv.SnapshotData, inv.SourceQuoteID, inv.Notes,
		inv.CreatedBy, inv.CreatedAt, inv.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Invoice, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, invoice_number, status,
			customer_name, customer_address, customer_email, customer_ust_id_nr,
			company_snapshot, tax_mode, line_items, tax_breakdown,
			subtotal, total_tax, gross_total,
			invoice_date, delivery_date, due_date, payment_terms,
			snapshot_data, source_quote_id, notes,
			created_by, created_at, updated_at
		FROM finance_invoices
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id,
	)
	return r.scanInvoice(row)
}

func (r *PostgresRepository) List(ctx context.Context, tenantID uuid.UUID, filter ListFilter) ([]*models.Invoice, int, error) {
	var conditions []string
	var args []any
	argNum := 1

	// Always filter by tenant
	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argNum))
	args = append(args, tenantID)
	argNum++

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argNum))
		args = append(args, filter.Status)
		argNum++
	}

	if filter.DateFrom != nil {
		conditions = append(conditions, fmt.Sprintf("invoice_date >= $%d", argNum))
		args = append(args, *filter.DateFrom)
		argNum++
	}

	if filter.DateTo != nil {
		conditions = append(conditions, fmt.Sprintf("invoice_date <= $%d", argNum))
		args = append(args, *filter.DateTo)
		argNum++
	}

	if filter.Overdue {
		conditions = append(conditions, "status = 'sent'")
		conditions = append(conditions, fmt.Sprintf("due_date < $%d", argNum))
		args = append(args, "now()")
		argNum++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM finance_invoices %s", whereClause)
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Apply defaults for pagination
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	// Query with pagination
	query := fmt.Sprintf(`
		SELECT id, tenant_id, invoice_number, status,
			customer_name, customer_address, customer_email, customer_ust_id_nr,
			company_snapshot, tax_mode, line_items, tax_breakdown,
			subtotal, total_tax, gross_total,
			invoice_date, delivery_date, due_date, payment_terms,
			snapshot_data, source_quote_id, notes,
			created_by, created_at, updated_at
		FROM finance_invoices %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)

	args = append(args, limit, filter.Offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var invoices []*models.Invoice
	for rows.Next() {
		inv, scanErr := r.scanInvoiceFromRows(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		invoices = append(invoices, inv)
	}

	return invoices, total, rows.Err()
}

func (r *PostgresRepository) Update(ctx context.Context, inv *models.Invoice) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE finance_invoices SET
			invoice_number = $1, status = $2,
			customer_name = $3, customer_address = $4, customer_email = $5, customer_ust_id_nr = $6,
			company_snapshot = $7, tax_mode = $8, line_items = $9, tax_breakdown = $10,
			subtotal = $11, total_tax = $12, gross_total = $13,
			invoice_date = $14, delivery_date = $15, due_date = $16, payment_terms = $17,
			snapshot_data = $18, notes = $19,
			updated_at = $20
		WHERE tenant_id = $21 AND id = $22`,
		inv.InvoiceNumber, inv.Status,
		inv.CustomerName, inv.CustomerAddress, inv.CustomerEmail, inv.CustomerUStIDNr,
		inv.CompanySnapshotRaw, inv.TaxMode, inv.LineItems, inv.TaxBreakdownRaw,
		inv.Subtotal, inv.TotalTax, inv.GrossTotal,
		inv.InvoiceDate, inv.DeliveryDate, inv.DueDate, inv.PaymentTerms,
		inv.SnapshotData, inv.Notes,
		inv.UpdatedAt, inv.TenantID, inv.ID,
	)
	return err
}

func (r *PostgresRepository) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE finance_invoices SET status = $1, updated_at = NOW() WHERE tenant_id = $2 AND id = $3`,
		status, tenantID, id,
	)
	return err
}

// GetOverdue returns all sent invoices past their due date for a tenant.
func (r *PostgresRepository) GetOverdue(ctx context.Context, tenantID uuid.UUID) ([]*models.Invoice, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, invoice_number, status,
			customer_name, customer_address, customer_email, customer_ust_id_nr,
			company_snapshot, tax_mode, line_items, tax_breakdown,
			subtotal, total_tax, gross_total,
			invoice_date, delivery_date, due_date, payment_terms,
			snapshot_data, source_quote_id, notes,
			created_by, created_at, updated_at
		FROM finance_invoices
		WHERE tenant_id = $1 AND status = 'sent' AND due_date < NOW()
		ORDER BY due_date ASC`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invoices []*models.Invoice
	for rows.Next() {
		inv, scanErr := r.scanInvoiceFromRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		invoices = append(invoices, inv)
	}
	return invoices, rows.Err()
}

func (r *PostgresRepository) GetByQuoteID(ctx context.Context, tenantID, quoteID uuid.UUID) (*models.Invoice, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, invoice_number, status,
			customer_name, customer_address, customer_email, customer_ust_id_nr,
			company_snapshot, tax_mode, line_items, tax_breakdown,
			subtotal, total_tax, gross_total,
			invoice_date, delivery_date, due_date, payment_terms,
			snapshot_data, source_quote_id, notes,
			created_by, created_at, updated_at
		FROM finance_invoices
		WHERE tenant_id = $1 AND source_quote_id = $2`,
		tenantID, quoteID,
	)
	return r.scanInvoice(row)
}

// NextInvoiceNumber returns the next gap-free invoice number using SELECT FOR UPDATE.
// This is the CRITICAL GoBD compliance function -- gap-free sequential numbering.
// Format: RE-{year}-{padded_number} e.g., RE-2026-0001
//
// IMPORTANT: This is delegated to the shared NumberSequenceRepo implementation
// (PostgresNumberSequenceRepo in the quote package) which handles the transaction
// and INSERT-if-missing logic.

// scanInvoice scans a single row into an Invoice model.
func (r *PostgresRepository) scanInvoice(row pgx.Row) (*models.Invoice, error) {
	var inv models.Invoice
	var subtotalFloat, totalTaxFloat, grossTotalFloat float64
	err := row.Scan(
		&inv.ID, &inv.TenantID, &inv.InvoiceNumber, &inv.Status,
		&inv.CustomerName, &inv.CustomerAddress, &inv.CustomerEmail, &inv.CustomerUStIDNr,
		&inv.CompanySnapshotRaw, &inv.TaxMode, &inv.LineItems, &inv.TaxBreakdownRaw,
		&subtotalFloat, &totalTaxFloat, &grossTotalFloat,
		&inv.InvoiceDate, &inv.DeliveryDate, &inv.DueDate, &inv.PaymentTerms,
		&inv.SnapshotData, &inv.SourceQuoteID, &inv.Notes,
		&inv.CreatedBy, &inv.CreatedAt, &inv.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvoiceNotFound
	}
	if err != nil {
		return nil, err
	}
	inv.Subtotal = decimal.NewFromFloat(subtotalFloat)
	inv.TotalTax = decimal.NewFromFloat(totalTaxFloat)
	inv.GrossTotal = decimal.NewFromFloat(grossTotalFloat)
	return &inv, nil
}

// scanInvoiceFromRows scans from a rows iterator.
func (r *PostgresRepository) scanInvoiceFromRows(rows pgx.Rows) (*models.Invoice, error) {
	var inv models.Invoice
	var subtotalFloat, totalTaxFloat, grossTotalFloat float64
	err := rows.Scan(
		&inv.ID, &inv.TenantID, &inv.InvoiceNumber, &inv.Status,
		&inv.CustomerName, &inv.CustomerAddress, &inv.CustomerEmail, &inv.CustomerUStIDNr,
		&inv.CompanySnapshotRaw, &inv.TaxMode, &inv.LineItems, &inv.TaxBreakdownRaw,
		&subtotalFloat, &totalTaxFloat, &grossTotalFloat,
		&inv.InvoiceDate, &inv.DeliveryDate, &inv.DueDate, &inv.PaymentTerms,
		&inv.SnapshotData, &inv.SourceQuoteID, &inv.Notes,
		&inv.CreatedBy, &inv.CreatedAt, &inv.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	inv.Subtotal = decimal.NewFromFloat(subtotalFloat)
	inv.TotalTax = decimal.NewFromFloat(totalTaxFloat)
	inv.GrossTotal = decimal.NewFromFloat(grossTotalFloat)
	return &inv, nil
}

// marshalLineItems converts model line items to JSON for storage.
func marshalLineItems(items []models.LineItem) (json.RawMessage, error) {
	data, err := json.Marshal(items)
	if err != nil {
		return nil, fmt.Errorf("marshal line items: %w", err)
	}
	return data, nil
}

// unmarshalLineItems parses JSONB line items from storage.
func unmarshalLineItems(raw json.RawMessage) ([]models.LineItem, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var items []models.LineItem
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("unmarshal line items: %w", err)
	}
	return items, nil
}

// marshalTaxBreakdown converts a TaxBreakdown to JSON for storage.
func marshalTaxBreakdown(tb *models.TaxBreakdown) (json.RawMessage, error) {
	if tb == nil {
		return nil, nil
	}
	data, err := json.Marshal(tb)
	if err != nil {
		return nil, fmt.Errorf("marshal tax breakdown: %w", err)
	}
	return data, nil
}

// marshalCompanySnapshot converts company settings to a snapshot for storage.
func marshalCompanySnapshot(cs *models.CompanySettings) (json.RawMessage, error) {
	if cs == nil {
		return nil, nil
	}
	snapshot := models.CompanySnapshot{
		Name:            cs.Name,
		Street:          cs.Street,
		PLZ:             cs.PLZ,
		City:            cs.City,
		Country:         cs.Country,
		Steuernummer:    cs.Steuernummer,
		UStIDNr:         cs.UStIDNr,
		Handelsregister: cs.Handelsregister,
		BankName:        cs.BankName,
		IBAN:            cs.IBAN,
		BIC:             cs.BIC,
		LogoURL:         cs.LogoURL,
		AccentColor:     cs.AccentColor,
	}
	data, err := json.Marshal(snapshot)
	if err != nil {
		return nil, fmt.Errorf("marshal company snapshot: %w", err)
	}
	return data, nil
}

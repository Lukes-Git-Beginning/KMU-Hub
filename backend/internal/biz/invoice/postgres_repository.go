package invoice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

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

// querier is satisfied by both *pgxpool.Pool and pgx.Tx — used by loadInvoiceLines.
type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// invoiceColumns is the shared finance_invoices projection for all read queries, kept
// in one place so it never drifts from the scanInvoice/scanInvoiceFromRows order. Ends
// with the provenance columns from Migration 000243 (source/external_id/external_number).
const invoiceColumns = "id, tenant_id, invoice_number, status, " +
	"customer_name, customer_address, customer_email, customer_ust_id_nr, " +
	"company_snapshot, tax_mode, tax_breakdown, " +
	"subtotal, total_tax, gross_total, " +
	"invoice_date, delivery_date, due_date, payment_terms, " +
	"snapshot_data, source_quote_id, notes, " +
	"zugferd_profile, time_tracking_source, locked_at, locked_by, " +
	"contact_id, " +
	"created_by, created_at, updated_at, currency, " +
	"source, external_id, external_number, recurring_id"

func (r *PostgresRepository) Create(ctx context.Context, inv *models.Invoice) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = tx.Exec(ctx,
		`INSERT INTO finance_invoices (
			id, tenant_id, invoice_number, status,
			customer_name, customer_address, customer_email, customer_ust_id_nr,
			company_snapshot, tax_mode, tax_breakdown,
			subtotal, total_tax, gross_total,
			invoice_date, delivery_date, due_date, payment_terms,
			snapshot_data, source_quote_id, notes,
			contact_id,
			created_by, created_at, updated_at,
			currency, recurring_id
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8,
			$9, $10, $11,
			$12, $13, $14,
			$15, $16, $17, $18,
			$19, $20, $21,
			$22,
			$23, $24, $25,
			$26, $27
		)`,
		inv.ID, inv.TenantID, inv.InvoiceNumber, inv.Status,
		inv.CustomerName, inv.CustomerAddress, inv.CustomerEmail, inv.CustomerUStIDNr,
		inv.CompanySnapshotRaw, inv.TaxMode, inv.TaxBreakdownRaw,
		inv.Subtotal, inv.TotalTax, inv.GrossTotal,
		inv.InvoiceDate, inv.DeliveryDate, inv.DueDate, inv.PaymentTerms,
		inv.SnapshotData, inv.SourceQuoteID, inv.Notes,
		inv.ContactID,
		inv.CreatedBy, inv.CreatedAt, inv.UpdatedAt,
		inv.Currency, inv.RecurringID,
	)
	if err != nil {
		return fmt.Errorf("insert finance_invoices: %w", err)
	}

	if err := r.insertInvoiceLines(ctx, tx, inv); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// UpsertImported inserts or updates a read-only imported invoice keyed by
// (tenant_id, source, external_id). It never touches finance_number_sequences — the
// invoice keeps its external number. Line items are replaced (delete + re-insert),
// mirroring UpdateInTx. inv.ID is overwritten with the canonical row id so the line
// insert targets the right invoice on conflict.
func (r *PostgresRepository) UpsertImported(ctx context.Context, inv *models.Invoice) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var rowID uuid.UUID
	err = tx.QueryRow(ctx,
		`INSERT INTO finance_invoices (
			id, tenant_id, invoice_number, status,
			customer_name, customer_address, customer_email,
			tax_mode, subtotal, total_tax, gross_total,
			invoice_date, due_date, payment_terms, notes,
			contact_id, created_by, created_at, updated_at, currency,
			source, external_id, external_number
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7,
			$8, $9, $10, $11,
			$12, $13, $14, $15,
			$16, $17, $18, $19, $20,
			$21, $22, $23
		)
		ON CONFLICT (tenant_id, source, external_id) WHERE external_id IS NOT NULL
		DO UPDATE SET
			invoice_number = EXCLUDED.invoice_number,
			status = EXCLUDED.status,
			customer_name = EXCLUDED.customer_name,
			customer_address = EXCLUDED.customer_address,
			customer_email = EXCLUDED.customer_email,
			subtotal = EXCLUDED.subtotal,
			total_tax = EXCLUDED.total_tax,
			gross_total = EXCLUDED.gross_total,
			invoice_date = EXCLUDED.invoice_date,
			due_date = EXCLUDED.due_date,
			contact_id = EXCLUDED.contact_id,
			currency = EXCLUDED.currency,
			external_number = EXCLUDED.external_number,
			updated_at = EXCLUDED.updated_at
		RETURNING id`,
		inv.ID, inv.TenantID, inv.InvoiceNumber, inv.Status,
		inv.CustomerName, inv.CustomerAddress, inv.CustomerEmail,
		inv.TaxMode, inv.Subtotal, inv.TotalTax, inv.GrossTotal,
		inv.InvoiceDate, inv.DueDate, inv.PaymentTerms, inv.Notes,
		inv.ContactID, inv.CreatedBy, inv.CreatedAt, inv.UpdatedAt, inv.Currency,
		inv.Source, inv.ExternalID, inv.ExternalNumber,
	).Scan(&rowID)
	if err != nil {
		return fmt.Errorf("upsert imported invoice: %w", err)
	}
	inv.ID = rowID

	// Replace line items for the canonical row (delete + re-insert, like UpdateInTx).
	if _, err = tx.Exec(ctx,
		`DELETE FROM finance_invoice_lines WHERE invoice_id = $1 AND tenant_id = $2`,
		inv.ID, inv.TenantID,
	); err != nil {
		return fmt.Errorf("delete imported invoice lines: %w", err)
	}
	if err = r.insertInvoiceLines(ctx, tx, inv); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Invoice, error) {
	row := r.pool.QueryRow(ctx,
		"SELECT "+invoiceColumns+" FROM finance_invoices WHERE tenant_id = $1 AND id = $2",
		tenantID, id,
	)
	inv, err := r.scanInvoice(row)
	if err != nil {
		return nil, err
	}

	linesByID, err := loadInvoiceLines(ctx, r.pool, []uuid.UUID{inv.ID})
	if err != nil {
		return nil, err
	}
	if lines, ok := linesByID[inv.ID]; ok {
		raw, marshalErr := marshalLineItems(lines)
		if marshalErr != nil {
			return nil, marshalErr
		}
		inv.LineItems = raw
	}

	return inv, nil
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

	// Both bounds are inclusive — an invoice dated exactly on DateFrom or DateTo
	// is included. Belegt durch TestPostgresRepository_List_DateRangeInclusiveBoundaries.
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
		conditions = append(conditions, "due_date < NOW()")
	}

	if filter.ContactID != nil {
		conditions = append(conditions, fmt.Sprintf("contact_id = $%d", argNum))
		args = append(args, *filter.ContactID)
		argNum++
	}

	if filter.RecurringID != nil {
		conditions = append(conditions, fmt.Sprintf("recurring_id = $%d", argNum))
		args = append(args, *filter.RecurringID)
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
	query := fmt.Sprintf(
		"SELECT %s FROM finance_invoices %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
		invoiceColumns, whereClause, argNum, argNum+1,
	)

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
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	// Bulk-load line items — one query for all invoices in the page (avoids N+1).
	if len(invoices) > 0 {
		ids := make([]uuid.UUID, len(invoices))
		for i, inv := range invoices {
			ids[i] = inv.ID
		}
		linesByID, linesErr := loadInvoiceLines(ctx, r.pool, ids)
		if linesErr != nil {
			return nil, 0, linesErr
		}
		for _, inv := range invoices {
			if lines, ok := linesByID[inv.ID]; ok {
				raw, marshalErr := marshalLineItems(lines)
				if marshalErr != nil {
					return nil, 0, marshalErr
				}
				inv.LineItems = raw
			}
		}
	}

	return invoices, total, nil
}

func (r *PostgresRepository) Update(ctx context.Context, inv *models.Invoice) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := r.UpdateInTx(ctx, tx, inv); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// UpdateInTx performs the invoice update (header + replace-strategy line items)
// within the provided transaction. The caller owns the transaction. Used by
// invoice.Service.Send to couple number assignment and the status/number update
// atomically (GoBD: a failed update must not burn a sequence number).
func (r *PostgresRepository) UpdateInTx(ctx context.Context, tx pgx.Tx, inv *models.Invoice) error {
	_, err := tx.Exec(ctx,
		`UPDATE finance_invoices SET
			invoice_number = $1, status = $2,
			customer_name = $3, customer_address = $4, customer_email = $5, customer_ust_id_nr = $6,
			company_snapshot = $7, tax_mode = $8, tax_breakdown = $9,
			subtotal = $10, total_tax = $11, gross_total = $12,
			invoice_date = $13, delivery_date = $14, due_date = $15, payment_terms = $16,
			snapshot_data = $17, notes = $18,
			locked_at = $19, locked_by = $20,
			updated_at = $21
		WHERE tenant_id = $22 AND id = $23`,
		inv.InvoiceNumber, inv.Status,
		inv.CustomerName, inv.CustomerAddress, inv.CustomerEmail, inv.CustomerUStIDNr,
		inv.CompanySnapshotRaw, inv.TaxMode, inv.TaxBreakdownRaw,
		inv.Subtotal, inv.TotalTax, inv.GrossTotal,
		inv.InvoiceDate, inv.DeliveryDate, inv.DueDate, inv.PaymentTerms,
		inv.SnapshotData, inv.Notes,
		inv.LockedAt, inv.LockedBy,
		inv.UpdatedAt, inv.TenantID, inv.ID,
	)
	if err != nil {
		return fmt.Errorf("update finance_invoices: %w", err)
	}

	// Delete and re-insert lines (replace strategy — no partial-update complexity).
	_, err = tx.Exec(ctx,
		`DELETE FROM finance_invoice_lines WHERE invoice_id = $1 AND tenant_id = $2`,
		inv.ID, inv.TenantID,
	)
	if err != nil {
		return fmt.Errorf("delete invoice lines: %w", err)
	}

	return r.insertInvoiceLines(ctx, tx, inv)
}

func (r *PostgresRepository) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE finance_invoices SET status = $1, updated_at = NOW() WHERE tenant_id = $2 AND id = $3`,
		status, tenantID, id,
	)
	return err
}

// UpdateStatusInTx updates the invoice status within the caller's transaction so a
// payment write and the coupled status transition commit (or roll back) atomically.
func (r *PostgresRepository) UpdateStatusInTx(ctx context.Context, tx pgx.Tx, tenantID, id uuid.UUID, status string) error {
	_, err := tx.Exec(ctx,
		`UPDATE finance_invoices SET status = $1, updated_at = NOW() WHERE tenant_id = $2 AND id = $3`,
		status, tenantID, id,
	)
	return err
}

// SetLock sets locked_at and locked_by on the invoice without triggering a
// full Update (which would unnecessarily re-write all line items).
// Used by LockInvoice (service_gobd.go).
func (r *PostgresRepository) SetLock(ctx context.Context, tenantID, id uuid.UUID, lockedAt time.Time, lockedBy uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE finance_invoices SET locked_at = $1, locked_by = $2, updated_at = $3
		WHERE id = $4 AND tenant_id = $5`,
		lockedAt, lockedBy, lockedAt, id, tenantID,
	)
	if err != nil {
		return fmt.Errorf("set invoice lock: %w", err)
	}
	return nil
}

// GetOverdue returns all sent invoices past their due date for a tenant.
func (r *PostgresRepository) GetOverdue(ctx context.Context, tenantID uuid.UUID) ([]*models.Invoice, error) {
	rows, err := r.pool.Query(ctx,
		"SELECT "+invoiceColumns+" FROM finance_invoices "+
			"WHERE tenant_id = $1 AND status = 'sent' AND due_date < NOW() AND source = 'cosmi' "+
			"ORDER BY due_date ASC",
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
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(invoices) > 0 {
		ids := make([]uuid.UUID, len(invoices))
		for i, inv := range invoices {
			ids[i] = inv.ID
		}
		linesByID, linesErr := loadInvoiceLines(ctx, r.pool, ids)
		if linesErr != nil {
			return nil, linesErr
		}
		for _, inv := range invoices {
			if lines, ok := linesByID[inv.ID]; ok {
				raw, marshalErr := marshalLineItems(lines)
				if marshalErr != nil {
					return nil, marshalErr
				}
				inv.LineItems = raw
			}
		}
	}

	return invoices, nil
}

// GetByQuoteID returns the invoice a quote was converted into. The schema does
// not enforce one invoice per quote (a storno leaves the cancelled invoice in
// place and the quote may be invoiced again), so the ordering is explicit: a
// live invoice wins over a cancelled one, the newest wins among equals. Service
// callers use this as the duplicate-conversion check and must not depend on
// whichever row the planner happens to return first.
func (r *PostgresRepository) GetByQuoteID(ctx context.Context, tenantID, quoteID uuid.UUID) (*models.Invoice, error) {
	row := r.pool.QueryRow(ctx,
		"SELECT "+invoiceColumns+" FROM finance_invoices "+
			"WHERE tenant_id = $1 AND source_quote_id = $2 "+
			"ORDER BY (status = 'cancelled'), created_at DESC LIMIT 1",
		tenantID, quoteID,
	)
	inv, err := r.scanInvoice(row)
	if err != nil {
		return nil, err
	}

	linesByID, err := loadInvoiceLines(ctx, r.pool, []uuid.UUID{inv.ID})
	if err != nil {
		return nil, err
	}
	if lines, ok := linesByID[inv.ID]; ok {
		raw, marshalErr := marshalLineItems(lines)
		if marshalErr != nil {
			return nil, marshalErr
		}
		inv.LineItems = raw
	}

	return inv, nil
}

// LinkTimeTracking persists the time_tracking_source JSONB column on an invoice.
func (r *PostgresRepository) LinkTimeTracking(ctx context.Context, tenantID, invoiceID uuid.UUID, src json.RawMessage) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE finance_invoices SET time_tracking_source = $1 WHERE id = $2 AND tenant_id = $3`,
		src, invoiceID, tenantID,
	)
	return err
}

// NextInvoiceNumber returns the next gap-free invoice number using SELECT FOR UPDATE.
// This is the CRITICAL GoBD compliance function -- gap-free sequential numbering.
// Format: RE-{year}-{padded_number} e.g., RE-2026-0001
//
// IMPORTANT: This is delegated to the shared NumberSequenceRepo implementation
// (PostgresNumberSequenceRepo in the quote package) which handles the transaction
// and INSERT-if-missing logic.

// ============================================================================
// GoBD-completion repository methods (Sprint 2 / Wave 1.B)
// ============================================================================

// InvoiceNumberExists returns true if the given invoice number is already
// assigned to an invoice belonging to the tenant.
func (r *PostgresRepository) InvoiceNumberExists(ctx context.Context, tenantID uuid.UUID, invoiceNumber string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM finance_invoices
			WHERE tenant_id = $1 AND invoice_number = $2
		)`,
		tenantID, invoiceNumber,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check invoice number exists: %w", err)
	}
	return exists, nil
}

// CountByFiscalYear counts non-draft invoices that have an invoice_number assigned
// and whose invoice_date falls within the given calendar year.
func (r *PostgresRepository) CountByFiscalYear(ctx context.Context, tenantID uuid.UUID, year int) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM finance_invoices
		WHERE tenant_id = $1
		  AND status != 'draft'
		  AND invoice_number != ''
		  AND source = 'cosmi'
		  AND EXTRACT(YEAR FROM invoice_date) = $2`,
		tenantID, year,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count invoices by fiscal year: %w", err)
	}
	return count, nil
}

// AggregatePaymentStats returns aggregated payment statistics for invoices
// with invoice_date within [fromDate, toDate]. total_outstanding_amount and
// total_paid_amount are netted against finance_payments the same way
// postgres_open_items.go's openItemsBase nets the Open-Items view: a partial
// payment lowers the outstanding amount, and an overpayment shows up in the
// paid amount. Where no payment row exists for a 'paid' invoice (marked paid
// without a tracked payment), gross_total is used as the fallback so a
// manually-closed invoice still counts as fully paid.
func (r *PostgresRepository) AggregatePaymentStats(ctx context.Context, tenantID uuid.UUID, fromDate, toDate time.Time) (PaymentStats, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT
			COUNT(*) AS total_invoices,
			COUNT(*) FILTER (WHERE i.status = 'paid') AS total_paid,
			COUNT(*) FILTER (WHERE i.status NOT IN ('paid', 'cancelled')) AS total_outstanding,
			COALESCE(SUM(COALESCE(p.paid, i.gross_total)) FILTER (WHERE i.status = 'paid'), 0) AS total_paid_amount,
			COALESCE(SUM(i.gross_total - COALESCE(p.paid, 0)) FILTER (WHERE i.status NOT IN ('paid', 'cancelled')), 0) AS total_outstanding_amount,
			COALESCE(
				AVG(
					EXTRACT(EPOCH FROM (i.updated_at - i.invoice_date)) / 86400.0
				) FILTER (WHERE i.status = 'paid'),
				0
			) AS avg_days_to_pay
		FROM finance_invoices i
		LEFT JOIN (
		    SELECT invoice_id, SUM(amount) AS paid
		    FROM finance_payments
		    WHERE tenant_id = $1
		    GROUP BY invoice_id
		) p ON p.invoice_id = i.id
		WHERE i.tenant_id = $1
		  AND i.invoice_date >= $2
		  AND i.invoice_date <= $3`,
		tenantID, fromDate, toDate,
	)

	var totalInvoices, totalPaid, totalOutstanding int
	// Money sums are scanned as strings and parsed into decimal (ADR-0007) to avoid
	// float rounding; avg_days_to_pay is a derived duration, not currency.
	var totalPaidAmtStr, totalOutstandingAmtStr string
	var avgDays float64
	if err := row.Scan(&totalInvoices, &totalPaid, &totalOutstanding, &totalPaidAmtStr, &totalOutstandingAmtStr, &avgDays); err != nil {
		return PaymentStats{}, fmt.Errorf("aggregate payment stats: %w", err)
	}
	totalPaidAmt, _ := decimal.NewFromString(totalPaidAmtStr)
	totalOutstandingAmt, _ := decimal.NewFromString(totalOutstandingAmtStr)

	return PaymentStats{
		TotalInvoices:          totalInvoices,
		TotalPaid:              totalPaid,
		TotalOutstanding:       totalOutstanding,
		TotalPaidAmount:        totalPaidAmt,
		TotalOutstandingAmount: totalOutstandingAmt,
		AverageDaysToPay:       decimal.NewFromFloat(avgDays),
	}, nil
}

// ListForDATEVExport returns sent/paid/overdue invoices in [fromDate, toDate],
// keyset-paged by (invoice_date, id) so a DATEV export can stream pages without
// holding the whole result set in memory. afterDate/afterID are the cursor from
// the previous page (nil for the first page). Ordered by (invoice_date, id) ASC.
// This is also the path GenerateGoBDExport (biz_grpc.go) builds its CSV from,
// combined with creditNoteService.ListForDATEVExport as negative storno rows —
// there is no separate GoBD-specific export method.
func (r *PostgresRepository) ListForDATEVExport(ctx context.Context, tenantID uuid.UUID, fromDate, toDate time.Time, afterDate *time.Time, afterID *uuid.UUID, limit int) ([]*models.Invoice, error) {
	args := []any{tenantID, fromDate, toDate}
	cursorClause := ""
	if afterDate != nil && afterID != nil {
		cursorClause = fmt.Sprintf(" AND (invoice_date, id) > ($%d, $%d)", len(args)+1, len(args)+2)
		args = append(args, *afterDate, *afterID)
	}
	args = append(args, limit)

	query := fmt.Sprintf(
		"SELECT %s FROM finance_invoices "+
			"WHERE tenant_id = $1 AND status IN ('sent', 'paid', 'overdue') AND source = 'cosmi' "+
			"AND invoice_date >= $2 AND invoice_date <= $3%s "+
			"ORDER BY invoice_date ASC, id ASC LIMIT $%d",
		invoiceColumns, cursorClause, len(args),
	)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list invoices for DATEV export: %w", err)
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
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate DATEV export rows: %w", err)
	}

	if len(invoices) > 0 {
		ids := make([]uuid.UUID, len(invoices))
		for i, inv := range invoices {
			ids[i] = inv.ID
		}
		linesByID, linesErr := loadInvoiceLines(ctx, r.pool, ids)
		if linesErr != nil {
			return nil, linesErr
		}
		for _, inv := range invoices {
			if lines, ok := linesByID[inv.ID]; ok {
				raw, marshalErr := marshalLineItems(lines)
				if marshalErr != nil {
					return nil, marshalErr
				}
				inv.LineItems = raw
			}
		}
	}

	return invoices, nil
}

// ============================================================================
// Scan helpers
// ============================================================================

// scanInvoice scans a single pgx.Row into an Invoice model.
// Scans all 30 columns including contact_id (Migration 000141), zugferd_profile,
// time_tracking_source, locked_at, locked_by (ADR-0007 lock columns) and
// currency (Migration 000216, B6). Does NOT scan line_items (dropped in 000217).
func (r *PostgresRepository) scanInvoice(row pgx.Row) (*models.Invoice, error) {
	var inv models.Invoice
	var subtotalStr, totalTaxStr, grossTotalStr string
	err := row.Scan(
		&inv.ID, &inv.TenantID, &inv.InvoiceNumber, &inv.Status,
		&inv.CustomerName, &inv.CustomerAddress, &inv.CustomerEmail, &inv.CustomerUStIDNr,
		&inv.CompanySnapshotRaw, &inv.TaxMode, &inv.TaxBreakdownRaw,
		&subtotalStr, &totalTaxStr, &grossTotalStr,
		&inv.InvoiceDate, &inv.DeliveryDate, &inv.DueDate, &inv.PaymentTerms,
		&inv.SnapshotData, &inv.SourceQuoteID, &inv.Notes,
		&inv.ZUGFeRDProfile, &inv.TimeTrackingSource, &inv.LockedAt, &inv.LockedBy,
		&inv.ContactID,
		&inv.CreatedBy, &inv.CreatedAt, &inv.UpdatedAt, &inv.Currency,
		&inv.Source, &inv.ExternalID, &inv.ExternalNumber, &inv.RecurringID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvoiceNotFound
	}
	if err != nil {
		return nil, err
	}
	inv.Subtotal, _ = decimal.NewFromString(subtotalStr)
	inv.TotalTax, _ = decimal.NewFromString(totalTaxStr)
	inv.GrossTotal, _ = decimal.NewFromString(grossTotalStr)
	return &inv, nil
}

// scanInvoiceFromRows scans from a pgx.Rows iterator.
func (r *PostgresRepository) scanInvoiceFromRows(rows pgx.Rows) (*models.Invoice, error) {
	var inv models.Invoice
	var subtotalStr, totalTaxStr, grossTotalStr string
	err := rows.Scan(
		&inv.ID, &inv.TenantID, &inv.InvoiceNumber, &inv.Status,
		&inv.CustomerName, &inv.CustomerAddress, &inv.CustomerEmail, &inv.CustomerUStIDNr,
		&inv.CompanySnapshotRaw, &inv.TaxMode, &inv.TaxBreakdownRaw,
		&subtotalStr, &totalTaxStr, &grossTotalStr,
		&inv.InvoiceDate, &inv.DeliveryDate, &inv.DueDate, &inv.PaymentTerms,
		&inv.SnapshotData, &inv.SourceQuoteID, &inv.Notes,
		&inv.ZUGFeRDProfile, &inv.TimeTrackingSource, &inv.LockedAt, &inv.LockedBy,
		&inv.ContactID,
		&inv.CreatedBy, &inv.CreatedAt, &inv.UpdatedAt, &inv.Currency,
		&inv.Source, &inv.ExternalID, &inv.ExternalNumber, &inv.RecurringID,
	)
	if err != nil {
		return nil, err
	}
	inv.Subtotal, _ = decimal.NewFromString(subtotalStr)
	inv.TotalTax, _ = decimal.NewFromString(totalTaxStr)
	inv.GrossTotal, _ = decimal.NewFromString(grossTotalStr)
	return &inv, nil
}

// ============================================================================
// Line-item helpers
// ============================================================================

// insertInvoiceLines inserts all line items for inv into finance_invoice_lines within tx.
// Called from Create and Update (after DELETE). Decimal values are written as
// their string representation — pgx maps these to NUMERIC correctly.
func (r *PostgresRepository) insertInvoiceLines(ctx context.Context, tx pgx.Tx, inv *models.Invoice) error {
	items, err := unmarshalLineItems(inv.LineItems)
	if err != nil {
		return fmt.Errorf("unmarshal line items for insert: %w", err)
	}
	for i, item := range items {
		position := item.Position
		if position < 1 {
			position = i + 1
		}
		_, execErr := tx.Exec(ctx,
			`INSERT INTO finance_invoice_lines (
				invoice_id, tenant_id, position, description,
				quantity, unit_price, tax_rate, line_total,
				created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
			inv.ID, inv.TenantID, position, item.Description,
			item.Quantity.String(), item.UnitPrice.String(),
			item.TaxRate.String(), item.LineTotal.String(),
			inv.CreatedAt, inv.UpdatedAt,
		)
		if execErr != nil {
			return fmt.Errorf("insert invoice line %d: %w", position, execErr)
		}
	}
	return nil
}

// loadInvoiceLines loads all line items for the given invoice IDs in a single query
// and returns them grouped by invoice_id. Avoids N+1 for List operations.
// The row id is mapped to LineItem.ID so callers can identify individual lines.
func loadInvoiceLines(ctx context.Context, q querier, invoiceIDs []uuid.UUID) (map[uuid.UUID][]models.LineItem, error) {
	if len(invoiceIDs) == 0 {
		return nil, nil
	}

	rows, err := q.Query(ctx,
		`SELECT id, invoice_id, position, description,
			quantity, unit_price, tax_rate, line_total
		FROM finance_invoice_lines
		WHERE invoice_id = ANY($1)
		ORDER BY invoice_id, position`,
		invoiceIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("load invoice lines: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]models.LineItem)
	for rows.Next() {
		var lineID, invoiceID uuid.UUID
		var quantityStr, unitPriceStr, taxRateStr, lineTotalStr string
		var item models.LineItem
		if scanErr := rows.Scan(
			&lineID, &invoiceID, &item.Position, &item.Description,
			&quantityStr, &unitPriceStr, &taxRateStr, &lineTotalStr,
		); scanErr != nil {
			return nil, fmt.Errorf("scan invoice line: %w", scanErr)
		}
		item.ID = lineID.String()
		item.Quantity, _ = decimal.NewFromString(quantityStr)
		item.UnitPrice, _ = decimal.NewFromString(unitPriceStr)
		item.TaxRate, _ = decimal.NewFromString(taxRateStr)
		item.LineTotal, _ = decimal.NewFromString(lineTotalStr)
		result[invoiceID] = append(result[invoiceID], item)
	}
	return result, rows.Err()
}

// ============================================================================
// JSON helpers (shared with service.go)
// ============================================================================

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

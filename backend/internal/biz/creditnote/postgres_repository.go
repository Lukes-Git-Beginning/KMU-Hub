package creditnote

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

// NewPostgresRepository creates a new PostgreSQL credit note repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// querier is satisfied by both *pgxpool.Pool and pgx.Tx — used by loadCreditNoteLines.
type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func (r *PostgresRepository) Create(ctx context.Context, cn *models.CreditNote) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = tx.Exec(ctx,
		`INSERT INTO finance_credit_notes (
			id, tenant_id, credit_note_number, status,
			original_invoice_id,
			customer_name, customer_address, customer_email, customer_ust_id_nr,
			tax_mode, line_items, tax_breakdown,
			subtotal, total_tax, gross_total,
			reason, created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4,
			$5,
			$6, $7, $8, $9,
			$10, $11, $12,
			$13, $14, $15,
			$16, $17, $18, $19
		)`,
		cn.ID, cn.TenantID, cn.CreditNoteNumber, cn.Status,
		cn.OriginalInvoiceID,
		cn.CustomerName, cn.CustomerAddress, cn.CustomerEmail, cn.CustomerUStIDNr,
		cn.TaxMode, cn.LineItems, cn.TaxBreakdownRaw,
		cn.Subtotal.String(), cn.TotalTax.String(), cn.GrossTotal.String(),
		cn.Reason, cn.CreatedBy, cn.CreatedAt, cn.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert finance_credit_notes: %w", err)
	}

	if err := r.insertCreditNoteLines(ctx, tx, cn); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.CreditNote, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, credit_note_number, status,
			original_invoice_id,
			customer_name, customer_address, customer_email, customer_ust_id_nr,
			tax_mode, line_items, tax_breakdown,
			subtotal, total_tax, gross_total,
			reason, created_by, created_at, updated_at
		FROM finance_credit_notes
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id,
	)
	cn, err := r.scanCreditNote(row)
	if err != nil {
		return nil, err
	}

	linesByID, err := loadCreditNoteLines(ctx, r.pool, []uuid.UUID{cn.ID})
	if err != nil {
		return nil, err
	}
	if lines, ok := linesByID[cn.ID]; ok {
		raw, marshalErr := marshalLineItems(lines)
		if marshalErr != nil {
			return nil, marshalErr
		}
		cn.LineItems = raw
	}

	return cn, nil
}

func (r *PostgresRepository) Update(ctx context.Context, cn *models.CreditNote) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = tx.Exec(ctx,
		`UPDATE finance_credit_notes SET
			credit_note_number = $1, status = $2,
			customer_name = $3, customer_address = $4, customer_email = $5, customer_ust_id_nr = $6,
			tax_mode = $7, line_items = $8, tax_breakdown = $9,
			subtotal = $10, total_tax = $11, gross_total = $12,
			reason = $13, updated_at = $14
		WHERE tenant_id = $15 AND id = $16`,
		cn.CreditNoteNumber, cn.Status,
		cn.CustomerName, cn.CustomerAddress, cn.CustomerEmail, cn.CustomerUStIDNr,
		cn.TaxMode, cn.LineItems, cn.TaxBreakdownRaw,
		cn.Subtotal.String(), cn.TotalTax.String(), cn.GrossTotal.String(),
		cn.Reason, cn.UpdatedAt,
		cn.TenantID, cn.ID,
	)
	if err != nil {
		return fmt.Errorf("update finance_credit_notes: %w", err)
	}

	// Delete and re-insert lines (replace strategy — no partial-update complexity).
	_, err = tx.Exec(ctx,
		`DELETE FROM finance_credit_note_lines WHERE credit_note_id = $1 AND tenant_id = $2`,
		cn.ID, cn.TenantID,
	)
	if err != nil {
		return fmt.Errorf("delete credit note lines: %w", err)
	}

	if err := r.insertCreditNoteLines(ctx, tx, cn); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepository) List(ctx context.Context, tenantID uuid.UUID, filter ListFilter) ([]*models.CreditNote, int, error) {
	var conditions []string
	var args []any
	argNum := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argNum))
	args = append(args, tenantID)
	argNum++

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argNum))
		args = append(args, filter.Status)
		argNum++
	}

	if filter.OriginalInvID != nil {
		conditions = append(conditions, fmt.Sprintf("original_invoice_id = $%d", argNum))
		args = append(args, *filter.OriginalInvID)
		argNum++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM finance_credit_notes %s", whereClause)
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, credit_note_number, status,
			original_invoice_id,
			customer_name, customer_address, customer_email, customer_ust_id_nr,
			tax_mode, line_items, tax_breakdown,
			subtotal, total_tax, gross_total,
			reason, created_by, created_at, updated_at
		FROM finance_credit_notes %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)

	args = append(args, limit, filter.Offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var creditNotes []*models.CreditNote
	for rows.Next() {
		cn, scanErr := r.scanCreditNoteFromRows(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		creditNotes = append(creditNotes, cn)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	// Bulk-load line items — one query for all credit notes in the page (avoids N+1).
	if len(creditNotes) > 0 {
		ids := make([]uuid.UUID, len(creditNotes))
		for i, cn := range creditNotes {
			ids[i] = cn.ID
		}
		linesByID, linesErr := loadCreditNoteLines(ctx, r.pool, ids)
		if linesErr != nil {
			return nil, 0, linesErr
		}
		for _, cn := range creditNotes {
			if lines, ok := linesByID[cn.ID]; ok {
				raw, marshalErr := marshalLineItems(lines)
				if marshalErr != nil {
					return nil, 0, marshalErr
				}
				cn.LineItems = raw
			}
		}
	}

	return creditNotes, total, nil
}

func (r *PostgresRepository) GetByInvoiceID(ctx context.Context, tenantID, invoiceID uuid.UUID) ([]*models.CreditNote, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, credit_note_number, status,
			original_invoice_id,
			customer_name, customer_address, customer_email, customer_ust_id_nr,
			tax_mode, line_items, tax_breakdown,
			subtotal, total_tax, gross_total,
			reason, created_by, created_at, updated_at
		FROM finance_credit_notes
		WHERE tenant_id = $1 AND original_invoice_id = $2
		ORDER BY created_at DESC`,
		tenantID, invoiceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var creditNotes []*models.CreditNote
	for rows.Next() {
		cn, scanErr := r.scanCreditNoteFromRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		creditNotes = append(creditNotes, cn)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Bulk-load line items for all credit notes in the result (avoids N+1).
	if len(creditNotes) > 0 {
		ids := make([]uuid.UUID, len(creditNotes))
		for i, cn := range creditNotes {
			ids[i] = cn.ID
		}
		linesByID, linesErr := loadCreditNoteLines(ctx, r.pool, ids)
		if linesErr != nil {
			return nil, linesErr
		}
		for _, cn := range creditNotes {
			if lines, ok := linesByID[cn.ID]; ok {
				raw, marshalErr := marshalLineItems(lines)
				if marshalErr != nil {
					return nil, marshalErr
				}
				cn.LineItems = raw
			}
		}
	}

	return creditNotes, nil
}

// ============================================================================
// Scan helpers
// ============================================================================

func (r *PostgresRepository) scanCreditNote(row pgx.Row) (*models.CreditNote, error) {
	var cn models.CreditNote
	var subtotalStr, totalTaxStr, grossTotalStr string
	err := row.Scan(
		&cn.ID, &cn.TenantID, &cn.CreditNoteNumber, &cn.Status,
		&cn.OriginalInvoiceID,
		&cn.CustomerName, &cn.CustomerAddress, &cn.CustomerEmail, &cn.CustomerUStIDNr,
		&cn.TaxMode, &cn.LineItems, &cn.TaxBreakdownRaw,
		&subtotalStr, &totalTaxStr, &grossTotalStr,
		&cn.Reason, &cn.CreatedBy, &cn.CreatedAt, &cn.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCreditNoteNotFound
	}
	if err != nil {
		return nil, err
	}
	cn.Subtotal, _ = decimal.NewFromString(subtotalStr)
	cn.TotalTax, _ = decimal.NewFromString(totalTaxStr)
	cn.GrossTotal, _ = decimal.NewFromString(grossTotalStr)
	return &cn, nil
}

func (r *PostgresRepository) scanCreditNoteFromRows(rows pgx.Rows) (*models.CreditNote, error) {
	var cn models.CreditNote
	var subtotalStr, totalTaxStr, grossTotalStr string
	err := rows.Scan(
		&cn.ID, &cn.TenantID, &cn.CreditNoteNumber, &cn.Status,
		&cn.OriginalInvoiceID,
		&cn.CustomerName, &cn.CustomerAddress, &cn.CustomerEmail, &cn.CustomerUStIDNr,
		&cn.TaxMode, &cn.LineItems, &cn.TaxBreakdownRaw,
		&subtotalStr, &totalTaxStr, &grossTotalStr,
		&cn.Reason, &cn.CreatedBy, &cn.CreatedAt, &cn.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	cn.Subtotal, _ = decimal.NewFromString(subtotalStr)
	cn.TotalTax, _ = decimal.NewFromString(totalTaxStr)
	cn.GrossTotal, _ = decimal.NewFromString(grossTotalStr)
	return &cn, nil
}

// ============================================================================
// Line-item helpers
// ============================================================================

// insertCreditNoteLines inserts all line items for cn into finance_credit_note_lines within tx.
// Called from Create and Update (after DELETE). Decimal values are written as
// their string representation — pgx maps these to NUMERIC correctly.
func (r *PostgresRepository) insertCreditNoteLines(ctx context.Context, tx pgx.Tx, cn *models.CreditNote) error {
	items, err := unmarshalLineItems(cn.LineItems)
	if err != nil {
		return fmt.Errorf("unmarshal line items for insert: %w", err)
	}
	for i, item := range items {
		position := item.Position
		if position < 1 {
			position = i + 1
		}
		_, execErr := tx.Exec(ctx,
			`INSERT INTO finance_credit_note_lines (
				credit_note_id, tenant_id, position, description,
				quantity, unit_price, tax_rate, line_total,
				created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
			cn.ID, cn.TenantID, position, item.Description,
			item.Quantity.String(), item.UnitPrice.String(),
			item.TaxRate.String(), item.LineTotal.String(),
			cn.CreatedAt, cn.UpdatedAt,
		)
		if execErr != nil {
			return fmt.Errorf("insert credit note line %d: %w", position, execErr)
		}
	}
	return nil
}

// loadCreditNoteLines loads all line items for the given credit note IDs in a single query
// and returns them grouped by credit_note_id. Avoids N+1 for List and GetByInvoiceID.
// The row id is mapped to LineItem.ID so callers can identify individual lines.
func loadCreditNoteLines(ctx context.Context, q querier, creditNoteIDs []uuid.UUID) (map[uuid.UUID][]models.LineItem, error) {
	if len(creditNoteIDs) == 0 {
		return nil, nil
	}

	rows, err := q.Query(ctx,
		`SELECT id, credit_note_id, position, description,
			quantity, unit_price, tax_rate, line_total
		FROM finance_credit_note_lines
		WHERE credit_note_id = ANY($1)
		ORDER BY credit_note_id, position`,
		creditNoteIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("load credit note lines: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]models.LineItem)
	for rows.Next() {
		var lineID, creditNoteID uuid.UUID
		var quantityStr, unitPriceStr, taxRateStr, lineTotalStr string
		var item models.LineItem
		if scanErr := rows.Scan(
			&lineID, &creditNoteID, &item.Position, &item.Description,
			&quantityStr, &unitPriceStr, &taxRateStr, &lineTotalStr,
		); scanErr != nil {
			return nil, fmt.Errorf("scan credit note line: %w", scanErr)
		}
		item.ID = lineID.String()
		item.Quantity, _ = decimal.NewFromString(quantityStr)
		item.UnitPrice, _ = decimal.NewFromString(unitPriceStr)
		item.TaxRate, _ = decimal.NewFromString(taxRateStr)
		item.LineTotal, _ = decimal.NewFromString(lineTotalStr)
		result[creditNoteID] = append(result[creditNoteID], item)
	}
	return result, rows.Err()
}

// ============================================================================
// JSON helpers (local to repository — marshal helpers live in service.go)
// ============================================================================

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

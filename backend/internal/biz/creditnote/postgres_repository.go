package creditnote

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

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL credit note repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, cn *models.CreditNote) error {
	_, err := r.pool.Exec(ctx,
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
		cn.Subtotal, cn.TotalTax, cn.GrossTotal,
		cn.Reason, cn.CreatedBy, cn.CreatedAt, cn.UpdatedAt,
	)
	return err
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
	return r.scanCreditNote(row)
}

func (r *PostgresRepository) Update(ctx context.Context, cn *models.CreditNote) error {
	_, err := r.pool.Exec(ctx,
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
		cn.Subtotal, cn.TotalTax, cn.GrossTotal,
		cn.Reason, cn.UpdatedAt,
		cn.TenantID, cn.ID,
	)
	return err
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

	return creditNotes, total, rows.Err()
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
	return creditNotes, rows.Err()
}

func (r *PostgresRepository) scanCreditNote(row pgx.Row) (*models.CreditNote, error) {
	var cn models.CreditNote
	var subtotalFloat, totalTaxFloat, grossTotalFloat float64
	err := row.Scan(
		&cn.ID, &cn.TenantID, &cn.CreditNoteNumber, &cn.Status,
		&cn.OriginalInvoiceID,
		&cn.CustomerName, &cn.CustomerAddress, &cn.CustomerEmail, &cn.CustomerUStIDNr,
		&cn.TaxMode, &cn.LineItems, &cn.TaxBreakdownRaw,
		&subtotalFloat, &totalTaxFloat, &grossTotalFloat,
		&cn.Reason, &cn.CreatedBy, &cn.CreatedAt, &cn.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCreditNoteNotFound
	}
	if err != nil {
		return nil, err
	}
	cn.Subtotal = decimal.NewFromFloat(subtotalFloat)
	cn.TotalTax = decimal.NewFromFloat(totalTaxFloat)
	cn.GrossTotal = decimal.NewFromFloat(grossTotalFloat)
	return &cn, nil
}

func (r *PostgresRepository) scanCreditNoteFromRows(rows pgx.Rows) (*models.CreditNote, error) {
	var cn models.CreditNote
	var subtotalFloat, totalTaxFloat, grossTotalFloat float64
	err := rows.Scan(
		&cn.ID, &cn.TenantID, &cn.CreditNoteNumber, &cn.Status,
		&cn.OriginalInvoiceID,
		&cn.CustomerName, &cn.CustomerAddress, &cn.CustomerEmail, &cn.CustomerUStIDNr,
		&cn.TaxMode, &cn.LineItems, &cn.TaxBreakdownRaw,
		&subtotalFloat, &totalTaxFloat, &grossTotalFloat,
		&cn.Reason, &cn.CreatedBy, &cn.CreatedAt, &cn.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	cn.Subtotal = decimal.NewFromFloat(subtotalFloat)
	cn.TotalTax = decimal.NewFromFloat(totalTaxFloat)
	cn.GrossTotal = decimal.NewFromFloat(grossTotalFloat)
	return &cn, nil
}


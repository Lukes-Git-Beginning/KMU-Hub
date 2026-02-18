package quote

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

// NewPostgresRepository creates a new PostgreSQL quote repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, q *models.Quote) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO finance_quotes (
			id, tenant_id, quote_number, status,
			customer_name, customer_address, customer_email, customer_ust_id_nr,
			tax_mode, line_items, tax_breakdown,
			subtotal, total_tax, gross_total,
			valid_until, notes, deal_id, source_quote_id,
			created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8,
			$9, $10, $11,
			$12, $13, $14,
			$15, $16, $17, $18,
			$19, $20, $21
		)`,
		q.ID, q.TenantID, q.QuoteNumber, q.Status,
		q.CustomerName, q.CustomerAddress, q.CustomerEmail, q.CustomerUStIDNr,
		q.TaxMode, q.LineItems, q.TaxBreakdownRaw,
		q.Subtotal, q.TotalTax, q.GrossTotal,
		q.ValidUntil, q.Notes, q.DealID, q.SourceQuoteID,
		q.CreatedBy, q.CreatedAt, q.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Quote, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, quote_number, status,
			customer_name, customer_address, customer_email, customer_ust_id_nr,
			tax_mode, line_items, tax_breakdown,
			subtotal, total_tax, gross_total,
			valid_until, notes, deal_id, source_quote_id,
			created_by, created_at, updated_at
		FROM finance_quotes
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id,
	)
	return r.scanQuote(row)
}

func (r *PostgresRepository) List(ctx context.Context, tenantID uuid.UUID, filter ListFilter) ([]*models.Quote, int, error) {
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

	if filter.DealID != nil {
		conditions = append(conditions, fmt.Sprintf("deal_id = $%d", argNum))
		args = append(args, *filter.DealID)
		argNum++
	}

	if filter.DateFrom != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argNum))
		args = append(args, *filter.DateFrom)
		argNum++
	}

	if filter.DateTo != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argNum))
		args = append(args, *filter.DateTo)
		argNum++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM finance_quotes %s", whereClause)
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
		SELECT id, tenant_id, quote_number, status,
			customer_name, customer_address, customer_email, customer_ust_id_nr,
			tax_mode, line_items, tax_breakdown,
			subtotal, total_tax, gross_total,
			valid_until, notes, deal_id, source_quote_id,
			created_by, created_at, updated_at
		FROM finance_quotes %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)

	args = append(args, limit, filter.Offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var quotes []*models.Quote
	for rows.Next() {
		q, scanErr := r.scanQuoteFromRows(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		quotes = append(quotes, q)
	}

	return quotes, total, rows.Err()
}

func (r *PostgresRepository) Update(ctx context.Context, q *models.Quote) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE finance_quotes SET
			quote_number = $1, status = $2,
			customer_name = $3, customer_address = $4, customer_email = $5, customer_ust_id_nr = $6,
			tax_mode = $7, line_items = $8, tax_breakdown = $9,
			subtotal = $10, total_tax = $11, gross_total = $12,
			valid_until = $13, notes = $14, deal_id = $15,
			updated_at = $16
		WHERE tenant_id = $17 AND id = $18`,
		q.QuoteNumber, q.Status,
		q.CustomerName, q.CustomerAddress, q.CustomerEmail, q.CustomerUStIDNr,
		q.TaxMode, q.LineItems, q.TaxBreakdownRaw,
		q.Subtotal, q.TotalTax, q.GrossTotal,
		q.ValidUntil, q.Notes, q.DealID,
		q.UpdatedAt, q.TenantID, q.ID,
	)
	return err
}

func (r *PostgresRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM finance_quotes WHERE tenant_id = $1 AND id = $2`,
		tenantID, id,
	)
	return err
}

func (r *PostgresRepository) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE finance_quotes SET status = $1, updated_at = NOW() WHERE tenant_id = $2 AND id = $3`,
		status, tenantID, id,
	)
	return err
}

func (r *PostgresRepository) GetByDealID(ctx context.Context, tenantID, dealID uuid.UUID) ([]*models.Quote, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, quote_number, status,
			customer_name, customer_address, customer_email, customer_ust_id_nr,
			tax_mode, line_items, tax_breakdown,
			subtotal, total_tax, gross_total,
			valid_until, notes, deal_id, source_quote_id,
			created_by, created_at, updated_at
		FROM finance_quotes
		WHERE tenant_id = $1 AND deal_id = $2
		ORDER BY created_at DESC`,
		tenantID, dealID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var quotes []*models.Quote
	for rows.Next() {
		q, scanErr := r.scanQuoteFromRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		quotes = append(quotes, q)
	}
	return quotes, rows.Err()
}

// PostgresNumberSequenceRepo implements NumberSequenceRepo using PostgreSQL with
// SELECT FOR UPDATE for gap-free numbering.
type PostgresNumberSequenceRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresNumberSequenceRepo creates a new number sequence repository.
func NewPostgresNumberSequenceRepo(pool *pgxpool.Pool) *PostgresNumberSequenceRepo {
	return &PostgresNumberSequenceRepo{pool: pool}
}

// NextNumber returns the next gap-free document number for the given type and fiscal year.
// Uses SELECT FOR UPDATE to serialize access. If no sequence row exists for the fiscal year,
// it inserts one within the same transaction.
func (r *PostgresNumberSequenceRepo) NextNumber(ctx context.Context, tenantID uuid.UUID, documentType string, fiscalYear int, prefix string) (string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var currentNumber int
	err = tx.QueryRow(ctx,
		`SELECT current_number FROM finance_number_sequences
		WHERE tenant_id = $1 AND document_type = $2 AND fiscal_year = $3
		FOR UPDATE`,
		tenantID, documentType, fiscalYear,
	).Scan(&currentNumber)

	if errors.Is(err, pgx.ErrNoRows) {
		// First document of this type for this fiscal year -- insert with current_number=1
		currentNumber = 1
		_, insertErr := tx.Exec(ctx,
			`INSERT INTO finance_number_sequences (id, tenant_id, document_type, prefix, current_number, fiscal_year)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			uuid.New(), tenantID, documentType, prefix, 1, fiscalYear,
		)
		if insertErr != nil {
			return "", fmt.Errorf("insert sequence: %w", insertErr)
		}
	} else if err != nil {
		return "", fmt.Errorf("select sequence: %w", err)
	} else {
		// Increment existing sequence
		currentNumber++
		_, updateErr := tx.Exec(ctx,
			`UPDATE finance_number_sequences SET current_number = $1, updated_at = NOW()
			WHERE tenant_id = $2 AND document_type = $3 AND fiscal_year = $4`,
			currentNumber, tenantID, documentType, fiscalYear,
		)
		if updateErr != nil {
			return "", fmt.Errorf("update sequence: %w", updateErr)
		}
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		return "", fmt.Errorf("commit tx: %w", commitErr)
	}

	// Format: PREFIX-YEAR-PADDED e.g., AN-2026-0001
	return fmt.Sprintf("%s-%d-%04d", prefix, fiscalYear, currentNumber), nil
}

// PostgresCompanySettingsRepo implements CompanySettingsRepo using PostgreSQL.
type PostgresCompanySettingsRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresCompanySettingsRepo creates a new company settings repository.
func NewPostgresCompanySettingsRepo(pool *pgxpool.Pool) *PostgresCompanySettingsRepo {
	return &PostgresCompanySettingsRepo{pool: pool}
}

func (r *PostgresCompanySettingsRepo) GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*models.CompanySettings, error) {
	var cs models.CompanySettings
	var basiszinssatzFloat float64
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, street, plz, city, country,
			steuernummer, ust_id_nr, handelsregister,
			bank_name, iban, bic,
			logo_url, accent_color,
			is_kleinunternehmer, default_payment_terms_days, default_quote_validity_days,
			basiszinssatz, created_at, updated_at
		FROM company_settings
		WHERE tenant_id = $1`,
		tenantID,
	).Scan(
		&cs.ID, &cs.TenantID, &cs.Name, &cs.Street, &cs.PLZ, &cs.City, &cs.Country,
		&cs.Steuernummer, &cs.UStIDNr, &cs.Handelsregister,
		&cs.BankName, &cs.IBAN, &cs.BIC,
		&cs.LogoURL, &cs.AccentColor,
		&cs.IsKleinunternehmer, &cs.DefaultPaymentTermsDays, &cs.DefaultQuoteValidityDays,
		&basiszinssatzFloat, &cs.CreatedAt, &cs.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // No settings configured yet
	}
	if err != nil {
		return nil, err
	}
	cs.Basiszinssatz = decimal.NewFromFloat(basiszinssatzFloat)
	return &cs, nil
}

// scanQuote scans a single row into a Quote model.
func (r *PostgresRepository) scanQuote(row pgx.Row) (*models.Quote, error) {
	var q models.Quote
	var subtotalFloat, totalTaxFloat, grossTotalFloat float64
	err := row.Scan(
		&q.ID, &q.TenantID, &q.QuoteNumber, &q.Status,
		&q.CustomerName, &q.CustomerAddress, &q.CustomerEmail, &q.CustomerUStIDNr,
		&q.TaxMode, &q.LineItems, &q.TaxBreakdownRaw,
		&subtotalFloat, &totalTaxFloat, &grossTotalFloat,
		&q.ValidUntil, &q.Notes, &q.DealID, &q.SourceQuoteID,
		&q.CreatedBy, &q.CreatedAt, &q.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrQuoteNotFound
	}
	if err != nil {
		return nil, err
	}
	q.Subtotal = decimal.NewFromFloat(subtotalFloat)
	q.TotalTax = decimal.NewFromFloat(totalTaxFloat)
	q.GrossTotal = decimal.NewFromFloat(grossTotalFloat)
	return &q, nil
}

// scanQuoteFromRows scans from a rows iterator.
func (r *PostgresRepository) scanQuoteFromRows(rows pgx.Rows) (*models.Quote, error) {
	var q models.Quote
	var subtotalFloat, totalTaxFloat, grossTotalFloat float64
	err := rows.Scan(
		&q.ID, &q.TenantID, &q.QuoteNumber, &q.Status,
		&q.CustomerName, &q.CustomerAddress, &q.CustomerEmail, &q.CustomerUStIDNr,
		&q.TaxMode, &q.LineItems, &q.TaxBreakdownRaw,
		&subtotalFloat, &totalTaxFloat, &grossTotalFloat,
		&q.ValidUntil, &q.Notes, &q.DealID, &q.SourceQuoteID,
		&q.CreatedBy, &q.CreatedAt, &q.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	q.Subtotal = decimal.NewFromFloat(subtotalFloat)
	q.TotalTax = decimal.NewFromFloat(totalTaxFloat)
	q.GrossTotal = decimal.NewFromFloat(grossTotalFloat)
	return &q, nil
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

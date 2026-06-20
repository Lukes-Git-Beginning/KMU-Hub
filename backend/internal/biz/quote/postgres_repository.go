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

	invoicebiz "github.com/kmuhub/kmuhub/internal/biz/invoice"
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

// querier is satisfied by both *pgxpool.Pool and pgx.Tx — used by loadQuoteLines.
type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func (r *PostgresRepository) Create(ctx context.Context, q *models.Quote) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = tx.Exec(ctx,
		`INSERT INTO finance_quotes (
			id, tenant_id, quote_number, status,
			customer_name, customer_address, customer_email, customer_ust_id_nr,
			tax_mode, line_items, tax_breakdown,
			subtotal, total_tax, gross_total,
			valid_until, notes, deal_id, source_quote_id,
			created_by, created_at, updated_at,
			currency
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8,
			$9, $10, $11,
			$12, $13, $14,
			$15, $16, $17, $18,
			$19, $20, $21,
			$22
		)`,
		q.ID, q.TenantID, q.QuoteNumber, q.Status,
		q.CustomerName, q.CustomerAddress, q.CustomerEmail, q.CustomerUStIDNr,
		q.TaxMode, q.LineItems, q.TaxBreakdownRaw,
		q.Subtotal, q.TotalTax, q.GrossTotal,
		q.ValidUntil, q.Notes, q.DealID, q.SourceQuoteID,
		q.CreatedBy, q.CreatedAt, q.UpdatedAt,
		q.Currency,
	)
	if err != nil {
		return fmt.Errorf("insert finance_quotes: %w", err)
	}

	if err := r.insertQuoteLines(ctx, tx, q); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Quote, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, quote_number, status,
			customer_name, customer_address, customer_email, customer_ust_id_nr,
			tax_mode, line_items, tax_breakdown,
			subtotal, total_tax, gross_total,
			valid_until, notes, deal_id, source_quote_id,
			created_by, created_at, updated_at, currency
		FROM finance_quotes
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id,
	)
	q, err := r.scanQuote(row)
	if err != nil {
		return nil, err
	}

	linesByID, err := loadQuoteLines(ctx, r.pool, []uuid.UUID{q.ID})
	if err != nil {
		return nil, err
	}
	if lines, ok := linesByID[q.ID]; ok {
		raw, marshalErr := marshalLineItems(lines)
		if marshalErr != nil {
			return nil, marshalErr
		}
		q.LineItems = raw
	}

	return q, nil
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
			created_by, created_at, updated_at, currency
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
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	// Bulk-load line items — one query for all quotes in the page (avoids N+1).
	if len(quotes) > 0 {
		ids := make([]uuid.UUID, len(quotes))
		for i, q := range quotes {
			ids[i] = q.ID
		}
		linesByID, linesErr := loadQuoteLines(ctx, r.pool, ids)
		if linesErr != nil {
			return nil, 0, linesErr
		}
		for _, q := range quotes {
			if lines, ok := linesByID[q.ID]; ok {
				raw, marshalErr := marshalLineItems(lines)
				if marshalErr != nil {
					return nil, 0, marshalErr
				}
				q.LineItems = raw
			}
		}
	}

	return quotes, total, nil
}

func (r *PostgresRepository) Update(ctx context.Context, q *models.Quote) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := r.UpdateInTx(ctx, tx, q); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// UpdateInTx performs the quote update within the caller's transaction. It is used by
// Send to assign the number and persist the sent state atomically (no separate commit
// per step), preventing a consumed-but-unassigned number on partial failure.
func (r *PostgresRepository) UpdateInTx(ctx context.Context, tx pgx.Tx, q *models.Quote) error {
	_, err := tx.Exec(ctx,
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
	if err != nil {
		return fmt.Errorf("update finance_quotes: %w", err)
	}

	// Delete and re-insert lines (replace strategy — no partial-update complexity).
	_, err = tx.Exec(ctx,
		`DELETE FROM finance_quote_lines WHERE quote_id = $1 AND tenant_id = $2`,
		q.ID, q.TenantID,
	)
	if err != nil {
		return fmt.Errorf("delete quote lines: %w", err)
	}

	return r.insertQuoteLines(ctx, tx, q)
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
			created_by, created_at, updated_at, currency
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
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(quotes) > 0 {
		ids := make([]uuid.UUID, len(quotes))
		for i, q := range quotes {
			ids[i] = q.ID
		}
		linesByID, linesErr := loadQuoteLines(ctx, r.pool, ids)
		if linesErr != nil {
			return nil, linesErr
		}
		for _, q := range quotes {
			if lines, ok := linesByID[q.ID]; ok {
				raw, marshalErr := marshalLineItems(lines)
				if marshalErr != nil {
					return nil, marshalErr
				}
				q.LineItems = raw
			}
		}
	}

	return quotes, nil
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

// NextNumber returns the next gap-free document number for the given type and fiscal year,
// running in its own transaction. Callers that must atomically couple numbering with another
// write (invoice/credit-note Send) should use NextNumberInTx within their own transaction so a
// failed status/number update rolls the sequence increment back instead of burning a number.
func (r *PostgresNumberSequenceRepo) NextNumber(ctx context.Context, tenantID uuid.UUID, documentType string, fiscalYear int, prefix string) (string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	number, err := r.NextNumberInTx(ctx, tx, tenantID, documentType, fiscalYear, prefix)
	if err != nil {
		return "", err
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		return "", fmt.Errorf("commit tx: %w", commitErr)
	}
	return number, nil
}

// NextNumberInTx performs the gap-free sequence increment within the provided transaction.
// Uses SELECT FOR UPDATE to serialize access; inserts the sequence row on first use. The
// caller owns the transaction (Begin/Commit/Rollback).
func (r *PostgresNumberSequenceRepo) NextNumberInTx(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, documentType string, fiscalYear int, prefix string) (string, error) {
	var currentNumber int
	err := tx.QueryRow(ctx,
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

	// Format: PREFIX-YEAR-PADDED e.g., AN-2026-0001
	return fmt.Sprintf("%s-%d-%04d", prefix, fiscalYear, currentNumber), nil
}

// GetSequenceInfo returns the current state of the number sequence for the given
// document type and fiscal year. Returns nil if no sequence exists yet.
// Used by invoice.GetJournalSummary for GoBD gap detection.
// The return type is *invoicebiz.SequenceInfo to satisfy the invoice.NumberSequenceRepo interface.
func (r *PostgresNumberSequenceRepo) GetSequenceInfo(ctx context.Context, tenantID uuid.UUID, documentType string, fiscalYear int) (*invoicebiz.SequenceInfo, error) {
	var currentNumber int
	err := r.pool.QueryRow(ctx,
		`SELECT current_number FROM finance_number_sequences
		WHERE tenant_id = $1 AND document_type = $2 AND fiscal_year = $3`,
		tenantID, documentType, fiscalYear,
	).Scan(&currentNumber)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get sequence info: %w", err)
	}
	return &invoicebiz.SequenceInfo{
		CurrentNumber: currentNumber,
		FiscalYear:    fiscalYear,
	}, nil
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
			basiszinssatz, default_currency, created_at, updated_at
		FROM company_settings
		WHERE tenant_id = $1`,
		tenantID,
	).Scan(
		&cs.ID, &cs.TenantID, &cs.Name, &cs.Street, &cs.PLZ, &cs.City, &cs.Country,
		&cs.Steuernummer, &cs.UStIDNr, &cs.Handelsregister,
		&cs.BankName, &cs.IBAN, &cs.BIC,
		&cs.LogoURL, &cs.AccentColor,
		&cs.IsKleinunternehmer, &cs.DefaultPaymentTermsDays, &cs.DefaultQuoteValidityDays,
		&basiszinssatzFloat, &cs.DefaultCurrency, &cs.CreatedAt, &cs.UpdatedAt,
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

// Upsert creates or updates company settings for a tenant.
func (r *PostgresCompanySettingsRepo) Upsert(ctx context.Context, settings *models.CompanySettings) error {
	if settings.ID == uuid.Nil {
		settings.ID = uuid.New()
	}
	// Never persist an empty currency over the column default — callers that don't
	// yet set DefaultCurrency would otherwise blank it out (B6).
	defaultCurrency := settings.DefaultCurrency
	if defaultCurrency == "" {
		defaultCurrency = models.DefaultCurrency
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO company_settings (
			id, tenant_id, name, street, plz, city, country,
			steuernummer, ust_id_nr, handelsregister,
			bank_name, iban, bic,
			logo_url, accent_color,
			is_kleinunternehmer, default_payment_terms_days, default_quote_validity_days,
			basiszinssatz, default_currency, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, NOW(), $21)
		ON CONFLICT (tenant_id)
		DO UPDATE SET
			name = EXCLUDED.name,
			street = EXCLUDED.street,
			plz = EXCLUDED.plz,
			city = EXCLUDED.city,
			country = EXCLUDED.country,
			steuernummer = EXCLUDED.steuernummer,
			ust_id_nr = EXCLUDED.ust_id_nr,
			handelsregister = EXCLUDED.handelsregister,
			bank_name = EXCLUDED.bank_name,
			iban = EXCLUDED.iban,
			bic = EXCLUDED.bic,
			logo_url = EXCLUDED.logo_url,
			accent_color = EXCLUDED.accent_color,
			is_kleinunternehmer = EXCLUDED.is_kleinunternehmer,
			default_payment_terms_days = EXCLUDED.default_payment_terms_days,
			default_quote_validity_days = EXCLUDED.default_quote_validity_days,
			basiszinssatz = EXCLUDED.basiszinssatz,
			default_currency = EXCLUDED.default_currency,
			updated_at = EXCLUDED.updated_at`,
		settings.ID, settings.TenantID,
		settings.Name, settings.Street, settings.PLZ, settings.City, settings.Country,
		settings.Steuernummer, settings.UStIDNr, settings.Handelsregister,
		settings.BankName, settings.IBAN, settings.BIC,
		settings.LogoURL, settings.AccentColor,
		settings.IsKleinunternehmer, settings.DefaultPaymentTermsDays, settings.DefaultQuoteValidityDays,
		settings.Basiszinssatz.InexactFloat64(), defaultCurrency, settings.UpdatedAt,
	)
	return err
}

// ============================================================================
// Scan helpers
// ============================================================================

// scanQuote scans a single pgx.Row into a Quote model.
// Decimals are scanned as strings to avoid float64 precision loss (ADR-0007).
func (r *PostgresRepository) scanQuote(row pgx.Row) (*models.Quote, error) {
	var q models.Quote
	var subtotalStr, totalTaxStr, grossTotalStr string
	err := row.Scan(
		&q.ID, &q.TenantID, &q.QuoteNumber, &q.Status,
		&q.CustomerName, &q.CustomerAddress, &q.CustomerEmail, &q.CustomerUStIDNr,
		&q.TaxMode, &q.LineItems, &q.TaxBreakdownRaw,
		&subtotalStr, &totalTaxStr, &grossTotalStr,
		&q.ValidUntil, &q.Notes, &q.DealID, &q.SourceQuoteID,
		&q.CreatedBy, &q.CreatedAt, &q.UpdatedAt, &q.Currency,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrQuoteNotFound
	}
	if err != nil {
		return nil, err
	}
	q.Subtotal, _ = decimal.NewFromString(subtotalStr)
	q.TotalTax, _ = decimal.NewFromString(totalTaxStr)
	q.GrossTotal, _ = decimal.NewFromString(grossTotalStr)
	return &q, nil
}

// scanQuoteFromRows scans from a pgx.Rows iterator.
// Decimals are scanned as strings to avoid float64 precision loss (ADR-0007).
func (r *PostgresRepository) scanQuoteFromRows(rows pgx.Rows) (*models.Quote, error) {
	var q models.Quote
	var subtotalStr, totalTaxStr, grossTotalStr string
	err := rows.Scan(
		&q.ID, &q.TenantID, &q.QuoteNumber, &q.Status,
		&q.CustomerName, &q.CustomerAddress, &q.CustomerEmail, &q.CustomerUStIDNr,
		&q.TaxMode, &q.LineItems, &q.TaxBreakdownRaw,
		&subtotalStr, &totalTaxStr, &grossTotalStr,
		&q.ValidUntil, &q.Notes, &q.DealID, &q.SourceQuoteID,
		&q.CreatedBy, &q.CreatedAt, &q.UpdatedAt, &q.Currency,
	)
	if err != nil {
		return nil, err
	}
	q.Subtotal, _ = decimal.NewFromString(subtotalStr)
	q.TotalTax, _ = decimal.NewFromString(totalTaxStr)
	q.GrossTotal, _ = decimal.NewFromString(grossTotalStr)
	return &q, nil
}

// ============================================================================
// Line-item helpers
// ============================================================================

// insertQuoteLines inserts all line items for q into finance_quote_lines within tx.
// Called from Create and Update (after DELETE). Decimal values are written as
// their string representation — pgx maps these to NUMERIC correctly.
func (r *PostgresRepository) insertQuoteLines(ctx context.Context, tx pgx.Tx, q *models.Quote) error {
	items, err := unmarshalLineItems(q.LineItems)
	if err != nil {
		return fmt.Errorf("unmarshal line items for insert: %w", err)
	}
	for i, item := range items {
		position := item.Position
		if position < 1 {
			position = i + 1
		}
		_, execErr := tx.Exec(ctx,
			`INSERT INTO finance_quote_lines (
				quote_id, tenant_id, position, description,
				quantity, unit_price, tax_rate, line_total,
				created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
			q.ID, q.TenantID, position, item.Description,
			item.Quantity.String(), item.UnitPrice.String(),
			item.TaxRate.String(), item.LineTotal.String(),
			q.CreatedAt, q.UpdatedAt,
		)
		if execErr != nil {
			return fmt.Errorf("insert quote line %d: %w", position, execErr)
		}
	}
	return nil
}

// loadQuoteLines loads all line items for the given quote IDs in a single query
// and returns them grouped by quote_id. Avoids N+1 for List operations.
// The row id is mapped to LineItem.ID so callers can identify individual lines.
func loadQuoteLines(ctx context.Context, q querier, quoteIDs []uuid.UUID) (map[uuid.UUID][]models.LineItem, error) {
	if len(quoteIDs) == 0 {
		return nil, nil
	}

	rows, err := q.Query(ctx,
		`SELECT id, quote_id, position, description,
			quantity, unit_price, tax_rate, line_total
		FROM finance_quote_lines
		WHERE quote_id = ANY($1)
		ORDER BY quote_id, position`,
		quoteIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("load quote lines: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]models.LineItem)
	for rows.Next() {
		var lineID, quoteID uuid.UUID
		var quantityStr, unitPriceStr, taxRateStr, lineTotalStr string
		var item models.LineItem
		if scanErr := rows.Scan(
			&lineID, &quoteID, &item.Position, &item.Description,
			&quantityStr, &unitPriceStr, &taxRateStr, &lineTotalStr,
		); scanErr != nil {
			return nil, fmt.Errorf("scan quote line: %w", scanErr)
		}
		item.ID = lineID.String()
		item.Quantity, _ = decimal.NewFromString(quantityStr)
		item.UnitPrice, _ = decimal.NewFromString(unitPriceStr)
		item.TaxRate, _ = decimal.NewFromString(taxRateStr)
		item.LineTotal, _ = decimal.NewFromString(lineTotalStr)
		result[quoteID] = append(result[quoteID], item)
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

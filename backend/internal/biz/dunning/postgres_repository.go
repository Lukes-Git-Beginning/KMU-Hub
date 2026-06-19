package dunning

import (
	"context"
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

// NewPostgresRepository creates a new PostgreSQL dunning repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, record *models.DunningRecord) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO finance_dunning_records (
			id, tenant_id, invoice_id, level, status,
			fee, interest, sent_at, created_by, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		record.ID, record.TenantID, record.InvoiceID, record.Level, record.Status,
		record.Fee, record.Interest, record.SentAt, record.CreatedBy, record.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.DunningRecord, error) {
	var record models.DunningRecord
	var feeStr, interestStr string
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, invoice_id, level, status,
			fee, interest, sent_at, created_by, created_at
		FROM finance_dunning_records
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id,
	).Scan(
		&record.ID, &record.TenantID, &record.InvoiceID, &record.Level, &record.Status,
		&feeStr, &interestStr, &record.SentAt, &record.CreatedBy, &record.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrDunningNotFound
	}
	if err != nil {
		return nil, err
	}
	// Scan monetary values as string + decimal to avoid float64 precision loss (ADR-0007).
	if record.Fee, err = decimal.NewFromString(feeStr); err != nil {
		return nil, fmt.Errorf("parse dunning fee %q: %w", feeStr, err)
	}
	if record.Interest, err = decimal.NewFromString(interestStr); err != nil {
		return nil, fmt.Errorf("parse dunning interest %q: %w", interestStr, err)
	}
	return &record, nil
}

func (r *PostgresRepository) List(ctx context.Context, tenantID uuid.UUID, filter ListFilter) ([]*models.DunningRecord, int, error) {
	var conditions []string
	var args []any
	argNum := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argNum))
	args = append(args, tenantID)
	argNum++

	if filter.InvoiceID != nil {
		conditions = append(conditions, fmt.Sprintf("invoice_id = $%d", argNum))
		args = append(args, *filter.InvoiceID)
		argNum++
	}

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argNum))
		args = append(args, filter.Status)
		argNum++
	}

	if filter.Level > 0 {
		conditions = append(conditions, fmt.Sprintf("level = $%d", argNum))
		args = append(args, filter.Level)
		argNum++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM finance_dunning_records %s", whereClause)
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, invoice_id, level, status,
			fee, interest, sent_at, created_by, created_at
		FROM finance_dunning_records %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)

	args = append(args, limit, filter.Offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var records []*models.DunningRecord
	for rows.Next() {
		record, scanErr := r.scanDunningFromRows(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		records = append(records, record)
	}

	return records, total, rows.Err()
}

func (r *PostgresRepository) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status string, sentAt *time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE finance_dunning_records SET status = $1, sent_at = $2
		WHERE tenant_id = $3 AND id = $4`,
		status, sentAt, tenantID, id,
	)
	return err
}

func (r *PostgresRepository) GetByInvoiceID(ctx context.Context, tenantID, invoiceID uuid.UUID) ([]*models.DunningRecord, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, invoice_id, level, status,
			fee, interest, sent_at, created_by, created_at
		FROM finance_dunning_records
		WHERE tenant_id = $1 AND invoice_id = $2
		ORDER BY level ASC, created_at ASC`,
		tenantID, invoiceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*models.DunningRecord
	for rows.Next() {
		record, scanErr := r.scanDunningFromRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

// GetByInvoiceIDs returns dunning records for multiple invoices in a single
// query, grouped by invoice ID. This avoids the N+1 that DetectAndCreateDunnings
// would otherwise incur by fetching existing records one invoice at a time.
func (r *PostgresRepository) GetByInvoiceIDs(ctx context.Context, tenantID uuid.UUID, invoiceIDs []uuid.UUID) (map[uuid.UUID][]*models.DunningRecord, error) {
	result := make(map[uuid.UUID][]*models.DunningRecord)
	if len(invoiceIDs) == 0 {
		return result, nil
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, invoice_id, level, status,
			fee, interest, sent_at, created_by, created_at
		FROM finance_dunning_records
		WHERE tenant_id = $1 AND invoice_id = ANY($2)
		ORDER BY invoice_id, level ASC, created_at ASC`,
		tenantID, invoiceIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		record, scanErr := r.scanDunningFromRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result[record.InvoiceID] = append(result[record.InvoiceID], record)
	}
	return result, rows.Err()
}

// GetHighestLevelByInvoiceID returns the dunning record with the highest level for an invoice.
func (r *PostgresRepository) GetHighestLevelByInvoiceID(ctx context.Context, tenantID, invoiceID uuid.UUID) (*models.DunningRecord, error) {
	var record models.DunningRecord
	var feeStr, interestStr string
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, invoice_id, level, status,
			fee, interest, sent_at, created_by, created_at
		FROM finance_dunning_records
		WHERE tenant_id = $1 AND invoice_id = $2
		ORDER BY level DESC
		LIMIT 1`,
		tenantID, invoiceID,
	).Scan(
		&record.ID, &record.TenantID, &record.InvoiceID, &record.Level, &record.Status,
		&feeStr, &interestStr, &record.SentAt, &record.CreatedBy, &record.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // No dunning records exist yet
	}
	if err != nil {
		return nil, err
	}
	// Scan monetary values as string + decimal to avoid float64 precision loss (ADR-0007).
	if record.Fee, err = decimal.NewFromString(feeStr); err != nil {
		return nil, fmt.Errorf("parse dunning fee %q: %w", feeStr, err)
	}
	if record.Interest, err = decimal.NewFromString(interestStr); err != nil {
		return nil, fmt.Errorf("parse dunning interest %q: %w", interestStr, err)
	}
	return &record, nil
}

func (r *PostgresRepository) scanDunningFromRows(rows pgx.Rows) (*models.DunningRecord, error) {
	var record models.DunningRecord
	var feeStr, interestStr string
	err := rows.Scan(
		&record.ID, &record.TenantID, &record.InvoiceID, &record.Level, &record.Status,
		&feeStr, &interestStr, &record.SentAt, &record.CreatedBy, &record.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	// Scan monetary values as string + decimal to avoid float64 precision loss (ADR-0007).
	if record.Fee, err = decimal.NewFromString(feeStr); err != nil {
		return nil, fmt.Errorf("parse dunning fee %q: %w", feeStr, err)
	}
	if record.Interest, err = decimal.NewFromString(interestStr); err != nil {
		return nil, fmt.Errorf("parse dunning interest %q: %w", interestStr, err)
	}
	return &record, nil
}

// PostgresConfigRepository implements ConfigRepository using PostgreSQL.
type PostgresConfigRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresConfigRepository creates a new PostgreSQL dunning config repository.
func NewPostgresConfigRepository(pool *pgxpool.Pool) *PostgresConfigRepository {
	return &PostgresConfigRepository{pool: pool}
}

func (r *PostgresConfigRepository) Get(ctx context.Context, tenantID uuid.UUID) (*models.DunningConfig, error) {
	var config models.DunningConfig
	var l1Fee, l2Fee, l3Fee string
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id,
			level1_days_after_due, level2_days_after_level1, level3_days_after_level2,
			level1_fee, level2_fee, level3_fee,
			updated_at
		FROM finance_dunning_config
		WHERE tenant_id = $1`,
		tenantID,
	).Scan(
		&config.ID, &config.TenantID,
		&config.Level1DaysAfterDue, &config.Level2DaysAfterLevel1, &config.Level3DaysAfterLevel2,
		&l1Fee, &l2Fee, &l3Fee,
		&config.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // No config exists yet
	}
	if err != nil {
		return nil, err
	}
	// Scan fee values as string + decimal to avoid float64 precision loss (ADR-0007).
	if config.Level1Fee, err = decimal.NewFromString(l1Fee); err != nil {
		return nil, fmt.Errorf("parse level1_fee %q: %w", l1Fee, err)
	}
	if config.Level2Fee, err = decimal.NewFromString(l2Fee); err != nil {
		return nil, fmt.Errorf("parse level2_fee %q: %w", l2Fee, err)
	}
	if config.Level3Fee, err = decimal.NewFromString(l3Fee); err != nil {
		return nil, fmt.Errorf("parse level3_fee %q: %w", l3Fee, err)
	}
	return &config, nil
}

func (r *PostgresConfigRepository) Upsert(ctx context.Context, config *models.DunningConfig) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO finance_dunning_config (
			id, tenant_id,
			level1_days_after_due, level2_days_after_level1, level3_days_after_level2,
			level1_fee, level2_fee, level3_fee,
			updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (tenant_id) DO UPDATE SET
			level1_days_after_due = EXCLUDED.level1_days_after_due,
			level2_days_after_level1 = EXCLUDED.level2_days_after_level1,
			level3_days_after_level2 = EXCLUDED.level3_days_after_level2,
			level1_fee = EXCLUDED.level1_fee,
			level2_fee = EXCLUDED.level2_fee,
			level3_fee = EXCLUDED.level3_fee,
			updated_at = EXCLUDED.updated_at`,
		config.ID, config.TenantID,
		config.Level1DaysAfterDue, config.Level2DaysAfterLevel1, config.Level3DaysAfterLevel2,
		config.Level1Fee, config.Level2Fee, config.Level3Fee,
		config.UpdatedAt,
	)
	return err
}

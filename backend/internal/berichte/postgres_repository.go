package berichte

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

	"github.com/kmuhub/kmuhub/internal/database"
)

// PostgresRepository implements Repository using PostgreSQL (pgx v5).
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// ============================================================================
// Definitions
// ============================================================================

func (r *PostgresRepository) CreateDefinition(ctx context.Context, def *Definition) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO report_definitions
		    (id, tenant_id, name, description, module, kind, query_config,
		     default_format, created_by, is_published, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		def.ID, def.TenantID, def.Name, def.Description, def.Module, def.Kind,
		def.QueryConfig, def.DefaultFormat, def.CreatedBy, def.IsPublished,
		def.CreatedAt, def.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) UpdateDefinition(ctx context.Context, def *Definition) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE report_definitions
		 SET name = $1, description = $2, module = $3, kind = $4,
		     query_config = $5, default_format = $6, is_published = $7, updated_at = $8
		 WHERE id = $9 AND tenant_id = $10`,
		def.Name, def.Description, def.Module, def.Kind, def.QueryConfig,
		def.DefaultFormat, def.IsPublished, def.UpdatedAt,
		def.ID, def.TenantID,
	)
	if err != nil {
		return fmt.Errorf("update definition: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrDefinitionNotFound
	}
	return nil
}

func (r *PostgresRepository) DeleteDefinition(ctx context.Context, tenantID, definitionID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM report_definitions WHERE id = $1 AND tenant_id = $2`,
		definitionID, tenantID,
	)
	if err != nil {
		return fmt.Errorf("delete definition: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrDefinitionNotFound
	}
	return nil
}

func (r *PostgresRepository) GetDefinition(ctx context.Context, tenantID, definitionID uuid.UUID) (*Definition, error) {
	return r.scanDefinition(r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, description, module, kind, query_config,
		        default_format, created_by, is_published, created_at, updated_at
		 FROM report_definitions WHERE id = $1 AND tenant_id = $2`,
		definitionID, tenantID,
	))
}

func (r *PostgresRepository) ListDefinitions(ctx context.Context, tenantID uuid.UUID, filter ListDefinitionsFilter, offset, limit int) ([]*Definition, int, error) {
	conditions := []string{"tenant_id = $1"}
	args := []any{tenantID}
	argNum := 2

	if filter.Module != nil {
		conditions = append(conditions, fmt.Sprintf("module = $%d", argNum))
		args = append(args, *filter.Module)
		argNum++
	}
	if filter.Kind != nil {
		conditions = append(conditions, fmt.Sprintf("kind = $%d", argNum))
		args = append(args, *filter.Kind)
		argNum++
	}
	if filter.IsPublished != nil {
		conditions = append(conditions, fmt.Sprintf("is_published = $%d", argNum))
		args = append(args, *filter.IsPublished)
		argNum++
	}
	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("LOWER(name) LIKE $%d", argNum))
		args = append(args, "%"+strings.ToLower(filter.Search)+"%")
		argNum++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM report_definitions %s", whereClause),
		args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count definitions: %w", err)
	}

	orderBy := "created_at"
	switch filter.SortBy {
	case "name":
		orderBy = "LOWER(name)"
	case "updated_at":
		orderBy = "updated_at"
	}
	direction := "DESC"
	if !filter.SortDesc {
		direction = "ASC"
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, name, description, module, kind, query_config,
		       default_format, created_by, is_published, created_at, updated_at
		FROM report_definitions %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderBy, direction, argNum, argNum+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list definitions: %w", err)
	}
	defer rows.Close()

	var out []*Definition
	for rows.Next() {
		d, scanErr := r.scanDefinitionFromRows(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		out = append(out, d)
	}
	return out, total, rows.Err()
}

// ============================================================================
// Cache
// ============================================================================

func (r *PostgresRepository) GetCacheEntry(ctx context.Context, tenantID, definitionID uuid.UUID, paramsHash string) (*CacheEntry, error) {
	var e CacheEntry
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, definition_id, params_hash, result, row_count, computed_at, expires_at
		 FROM report_cache
		 WHERE tenant_id = $1 AND definition_id = $2 AND params_hash = $3`,
		tenantID, definitionID, paramsHash,
	).Scan(&e.ID, &e.TenantID, &e.DefinitionID, &e.ParamsHash, &e.Result, &e.RowCount, &e.ComputedAt, &e.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCacheMiss
	}
	if err != nil {
		return nil, fmt.Errorf("get cache entry: %w", err)
	}
	return &e, nil
}

func (r *PostgresRepository) UpsertCacheEntry(ctx context.Context, entry *CacheEntry) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO report_cache
		    (id, tenant_id, definition_id, params_hash, result, row_count, computed_at, expires_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		 ON CONFLICT (definition_id, params_hash) DO UPDATE
		 SET result = EXCLUDED.result,
		     row_count = EXCLUDED.row_count,
		     computed_at = EXCLUDED.computed_at,
		     expires_at = EXCLUDED.expires_at`,
		entry.ID, entry.TenantID, entry.DefinitionID, entry.ParamsHash,
		entry.Result, entry.RowCount, entry.ComputedAt, entry.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("upsert cache entry: %w", err)
	}
	return nil
}

func (r *PostgresRepository) InvalidateCache(ctx context.Context, tenantID, definitionID uuid.UUID) (int, error) {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM report_cache WHERE tenant_id = $1 AND definition_id = $2`,
		tenantID, definitionID,
	)
	if err != nil {
		return 0, fmt.Errorf("invalidate cache: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

func (r *PostgresRepository) DeleteExpiredCacheEntries(ctx context.Context, before time.Time) (int, error) {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM report_cache WHERE expires_at < $1`,
		before,
	)
	if err != nil {
		return 0, fmt.Errorf("delete expired cache entries: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

// ============================================================================
// Schedules
// ============================================================================

func (r *PostgresRepository) CreateSchedule(ctx context.Context, sch *Schedule) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO report_schedules
		    (id, tenant_id, definition_id, name, cron_expression, recipients, format,
		     params, active, last_run_at, last_run_status, last_run_error,
		     created_by, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		sch.ID, sch.TenantID, sch.DefinitionID, sch.Name, sch.CronExpression,
		sch.Recipients, sch.Format, sch.Params, sch.Active,
		sch.LastRunAt, sch.LastRunStatus, sch.LastRunError,
		sch.CreatedBy, sch.CreatedAt, sch.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) UpdateSchedule(ctx context.Context, sch *Schedule) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE report_schedules
		 SET name = $1, cron_expression = $2, recipients = $3, format = $4,
		     params = $5, active = $6, updated_at = $7
		 WHERE id = $8 AND tenant_id = $9`,
		sch.Name, sch.CronExpression, sch.Recipients, sch.Format, sch.Params,
		sch.Active, sch.UpdatedAt, sch.ID, sch.TenantID,
	)
	if err != nil {
		return fmt.Errorf("update schedule: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrScheduleNotFound
	}
	return nil
}

func (r *PostgresRepository) DeleteSchedule(ctx context.Context, tenantID, scheduleID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM report_schedules WHERE id = $1 AND tenant_id = $2`,
		scheduleID, tenantID,
	)
	if err != nil {
		return fmt.Errorf("delete schedule: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrScheduleNotFound
	}
	return nil
}

func (r *PostgresRepository) GetSchedule(ctx context.Context, tenantID, scheduleID uuid.UUID) (*Schedule, error) {
	return r.scanSchedule(r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, definition_id, name, cron_expression, recipients, format,
		        params, active, last_run_at, last_run_status, last_run_error,
		        created_by, created_at, updated_at
		 FROM report_schedules WHERE id = $1 AND tenant_id = $2`,
		scheduleID, tenantID,
	))
}

func (r *PostgresRepository) ListSchedules(ctx context.Context, tenantID uuid.UUID, filter ListSchedulesFilter, offset, limit int) ([]*Schedule, int, error) {
	conditions := []string{"tenant_id = $1"}
	args := []any{tenantID}
	argNum := 2

	if filter.DefinitionID != nil {
		conditions = append(conditions, fmt.Sprintf("definition_id = $%d", argNum))
		args = append(args, *filter.DefinitionID)
		argNum++
	}
	if filter.Active != nil {
		conditions = append(conditions, fmt.Sprintf("active = $%d", argNum))
		args = append(args, *filter.Active)
		argNum++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM report_schedules %s", whereClause),
		args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count schedules: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, definition_id, name, cron_expression, recipients, format,
		       params, active, last_run_at, last_run_status, last_run_error,
		       created_by, created_at, updated_at
		FROM report_schedules %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list schedules: %w", err)
	}
	defer rows.Close()

	var out []*Schedule
	for rows.Next() {
		s, scanErr := r.scanScheduleFromRows(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		out = append(out, s)
	}
	return out, total, rows.Err()
}

func (r *PostgresRepository) ListDueSchedules(ctx context.Context, now time.Time) ([]*Schedule, error) {
	// Cron-next-fire decision happens in the scheduler; the repo returns all
	// active schedules so the scheduler can decide per-schedule.
	_ = now
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, definition_id, name, cron_expression, recipients, format,
		        params, active, last_run_at, last_run_status, last_run_error,
		        created_by, created_at, updated_at
		 FROM report_schedules WHERE active = TRUE
		 ORDER BY COALESCE(last_run_at, created_at) ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list due schedules: %w", err)
	}
	defer rows.Close()

	var out []*Schedule
	for rows.Next() {
		s, scanErr := r.scanScheduleFromRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ClaimSchedule atomically updates last_run_at iff it still matches the
// previously observed value (or was NULL when previousLastRunAt is nil).
// Returns true if the claim succeeded. Prevents double-execution when two
// scheduler ticks overlap.
func (r *PostgresRepository) ClaimSchedule(ctx context.Context, scheduleID uuid.UUID, previousLastRunAt *time.Time, now time.Time) (bool, error) {
	var tag pgconn.CommandTag
	var err error
	if previousLastRunAt == nil {
		tag, err = r.pool.Exec(ctx,
			`UPDATE report_schedules SET last_run_at = $1, updated_at = $1
			 WHERE id = $2 AND last_run_at IS NULL`,
			now, scheduleID,
		)
	} else {
		tag, err = r.pool.Exec(ctx,
			`UPDATE report_schedules SET last_run_at = $1, updated_at = $1
			 WHERE id = $2 AND last_run_at = $3`,
			now, scheduleID, *previousLastRunAt,
		)
	}
	if err != nil {
		return false, fmt.Errorf("claim schedule: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *PostgresRepository) UpdateScheduleLastRun(ctx context.Context, scheduleID uuid.UUID, status string, runErr *string, at time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE report_schedules
		 SET last_run_status = $1, last_run_error = $2, last_run_at = $3, updated_at = $3
		 WHERE id = $4`,
		status, runErr, at, scheduleID,
	)
	if err != nil {
		return fmt.Errorf("update schedule last run: %w", err)
	}
	return nil
}

// ============================================================================
// Runs
// ============================================================================

func (r *PostgresRepository) InsertRun(ctx context.Context, run *Run) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO report_runs
		    (id, tenant_id, definition_id, schedule_id, trigger, params,
		     duration_ms, row_count, status, error, started_at, completed_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		run.ID, run.TenantID, run.DefinitionID, run.ScheduleID, run.Trigger, run.Params,
		run.DurationMs, run.RowCount, run.Status, run.Error, run.StartedAt, run.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("insert run: %w", err)
	}
	return nil
}

// ============================================================================
// Scan helpers
// ============================================================================

func (r *PostgresRepository) scanDefinition(row pgx.Row) (*Definition, error) {
	var d Definition
	err := row.Scan(
		&d.ID, &d.TenantID, &d.Name, &d.Description, &d.Module, &d.Kind,
		&d.QueryConfig, &d.DefaultFormat, &d.CreatedBy, &d.IsPublished,
		&d.CreatedAt, &d.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrDefinitionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan definition: %w", err)
	}
	return &d, nil
}

func (r *PostgresRepository) scanDefinitionFromRows(rows pgx.Rows) (*Definition, error) {
	var d Definition
	err := rows.Scan(
		&d.ID, &d.TenantID, &d.Name, &d.Description, &d.Module, &d.Kind,
		&d.QueryConfig, &d.DefaultFormat, &d.CreatedBy, &d.IsPublished,
		&d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan definition row: %w", err)
	}
	return &d, nil
}

func (r *PostgresRepository) scanSchedule(row pgx.Row) (*Schedule, error) {
	var s Schedule
	err := row.Scan(
		&s.ID, &s.TenantID, &s.DefinitionID, &s.Name, &s.CronExpression,
		&s.Recipients, &s.Format, &s.Params, &s.Active,
		&s.LastRunAt, &s.LastRunStatus, &s.LastRunError,
		&s.CreatedBy, &s.CreatedAt, &s.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrScheduleNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan schedule: %w", err)
	}
	return &s, nil
}

func (r *PostgresRepository) scanScheduleFromRows(rows pgx.Rows) (*Schedule, error) {
	var s Schedule
	err := rows.Scan(
		&s.ID, &s.TenantID, &s.DefinitionID, &s.Name, &s.CronExpression,
		&s.Recipients, &s.Format, &s.Params, &s.Active,
		&s.LastRunAt, &s.LastRunStatus, &s.LastRunError,
		&s.CreatedBy, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan schedule row: %w", err)
	}
	return &s, nil
}

// ============================================================================
// Documents
// ============================================================================

const documentColumns = `id, tenant_id, title, description, module, status,
	        "rows", settings, template_id, created_by, created_at, updated_at, released_at`

func (r *PostgresRepository) CreateDocument(ctx context.Context, doc *Document) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO report_documents
		    (id, tenant_id, title, description, module, status, "rows", settings,
		     template_id, created_by, created_at, updated_at, released_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		doc.ID, doc.TenantID, doc.Title, doc.Description, doc.Module, doc.Status,
		doc.Rows, doc.Settings, doc.TemplateID, doc.CreatedBy,
		doc.CreatedAt, doc.UpdatedAt, doc.ReleasedAt,
	)
	if err != nil {
		return fmt.Errorf("create document: %w", err)
	}
	return nil
}

func (r *PostgresRepository) UpdateDocument(ctx context.Context, doc *Document) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE report_documents
		 SET title = $1, description = $2, module = $3, status = $4,
		     "rows" = $5, settings = $6, updated_at = $7, released_at = $8
		 WHERE id = $9 AND tenant_id = $10`,
		doc.Title, doc.Description, doc.Module, doc.Status,
		doc.Rows, doc.Settings, doc.UpdatedAt, doc.ReleasedAt,
		doc.ID, doc.TenantID,
	)
	if err != nil {
		return fmt.Errorf("update document: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrDocumentNotFound
	}
	return nil
}

func (r *PostgresRepository) DeleteDocument(ctx context.Context, tenantID, documentID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM report_documents WHERE id = $1 AND tenant_id = $2`,
		documentID, tenantID,
	)
	if err != nil {
		return fmt.Errorf("delete document: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrDocumentNotFound
	}
	return nil
}

func (r *PostgresRepository) GetDocument(ctx context.Context, tenantID, documentID uuid.UUID) (*Document, error) {
	return scanDocument(r.pool.QueryRow(ctx,
		`SELECT `+documentColumns+`
		 FROM report_documents WHERE id = $1 AND tenant_id = $2`,
		documentID, tenantID,
	))
}

func (r *PostgresRepository) ListDocuments(ctx context.Context, tenantID uuid.UUID, filter ListDocumentsFilter, offset, limit int) ([]*Document, int, error) {
	conditions := []string{"tenant_id = $1"}
	args := []any{tenantID}
	argNum := 2

	if filter.Module != nil {
		conditions = append(conditions, fmt.Sprintf("module = $%d", argNum))
		args = append(args, *filter.Module)
		argNum++
	}
	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argNum))
		args = append(args, *filter.Status)
		argNum++
	}
	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("LOWER(title) LIKE $%d", argNum))
		args = append(args, "%"+strings.ToLower(filter.Search)+"%")
		argNum++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM report_documents %s", whereClause),
		args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count documents: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT `+documentColumns+`
		FROM report_documents %s
		ORDER BY updated_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list documents: %w", err)
	}
	defer rows.Close()

	docs := make([]*Document, 0, limit)
	for rows.Next() {
		doc, scanErr := scanDocument(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		docs = append(docs, doc)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate documents: %w", err)
	}
	return docs, total, nil
}

// scanDocument reads one report_documents row. pgx.Row and pgx.Rows share the
// Scan signature, so the same helper serves single-row and list reads.
func scanDocument(row pgx.Row) (*Document, error) {
	var d Document
	err := row.Scan(
		&d.ID, &d.TenantID, &d.Title, &d.Description, &d.Module, &d.Status,
		&d.Rows, &d.Settings, &d.TemplateID, &d.CreatedBy,
		&d.CreatedAt, &d.UpdatedAt, &d.ReleasedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrDocumentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan document: %w", err)
	}
	return &d, nil
}

// ============================================================================
// Share tokens
// ============================================================================

const shareTokenColumns = `id, tenant_id, document_id, token, password_hash,
	        expires_at, revoked_at, view_count, created_by, created_at`

func (r *PostgresRepository) CreateShareToken(ctx context.Context, t *ShareToken) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO report_share_tokens
		    (id, tenant_id, document_id, token, password_hash, expires_at,
		     revoked_at, view_count, created_by, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		t.ID, t.TenantID, t.DocumentID, t.Token, t.PasswordHash, t.ExpiresAt,
		t.RevokedAt, t.ViewCount, t.CreatedBy, t.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create share token: %w", err)
	}
	return nil
}

// ListShareTokens returns the still-usable links of one document. Revoked rows
// stay in the table for the audit trail but are not listed — the frontend
// renders this list as "these links are live right now".
func (r *PostgresRepository) ListShareTokens(ctx context.Context, tenantID, documentID uuid.UUID) ([]*ShareToken, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+shareTokenColumns+`
		 FROM report_share_tokens
		 WHERE tenant_id = $1 AND document_id = $2 AND revoked_at IS NULL
		 ORDER BY created_at DESC`,
		tenantID, documentID,
	)
	if err != nil {
		return nil, fmt.Errorf("list share tokens: %w", err)
	}
	defer rows.Close()

	tokens := make([]*ShareToken, 0, 4)
	for rows.Next() {
		t, scanErr := scanShareToken(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		tokens = append(tokens, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate share tokens: %w", err)
	}
	return tokens, nil
}

// RevokeShareToken cuts a live link. Revoking an already-revoked one reports
// ErrShareNotFound rather than succeeding silently, so a double revoke from a
// stale UI does not read as a fresh one.
func (r *PostgresRepository) RevokeShareToken(ctx context.Context, tenantID, shareID uuid.UUID, at time.Time) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE report_share_tokens SET revoked_at = $1
		 WHERE id = $2 AND tenant_id = $3 AND revoked_at IS NULL`,
		at, shareID, tenantID,
	)
	if err != nil {
		return fmt.Errorf("revoke share token: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrShareNotFound
	}
	return nil
}

// GetShareTokenBySecret resolves a link — and with it its tenant — from the
// secret alone.
//
// This is the one read in this service that runs without a tenant in the
// session, because the public request has none: RLS would filter away the
// single row that answers which tenant the caller may see. The query is
// therefore pinned to an exact match on the unique token column and returns
// everything the service needs to decide; it is never a listing and never
// takes a filter. The revoked/expired verdict is the service's, not this
// query's, so that a revoked link still resolves far enough to be logged.
func (r *PostgresRepository) GetShareTokenBySecret(ctx context.Context, secret string) (*ShareToken, error) {
	if secret == "" {
		return nil, ErrShareNotFound
	}
	return scanShareToken(r.pool.QueryRow(database.WithSystemContext(ctx),
		`SELECT `+shareTokenColumns+`
		 FROM report_share_tokens WHERE token = $1`,
		secret,
	))
}

func (r *PostgresRepository) IncrementShareView(ctx context.Context, tenantID, shareID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE report_share_tokens SET view_count = view_count + 1
		 WHERE id = $1 AND tenant_id = $2`,
		shareID, tenantID,
	)
	if err != nil {
		return fmt.Errorf("increment share view: %w", err)
	}
	return nil
}

func scanShareToken(row pgx.Row) (*ShareToken, error) {
	var t ShareToken
	err := row.Scan(
		&t.ID, &t.TenantID, &t.DocumentID, &t.Token, &t.PasswordHash,
		&t.ExpiresAt, &t.RevokedAt, &t.ViewCount, &t.CreatedBy, &t.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrShareNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan share token: %w", err)
	}
	return &t, nil
}

// compile-time interface check
var _ Repository = (*PostgresRepository)(nil)

package formulare

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
)

// PostgresRepository implements Repository using PostgreSQL (pgx v5).
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL-backed formulare repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// ============================================================================
// Schema
// ============================================================================

func (r *PostgresRepository) CreateSchema(ctx context.Context, schema *FormSchema) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO form_schemas
		    (id, tenant_id, title, description, fields, status, is_template,
		     is_public, page_count, submission_count, created_by, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		schema.ID, schema.TenantID, schema.Title, schema.Description,
		schema.Fields, string(schema.Status), schema.IsTemplate, schema.IsPublic,
		schema.PageCount, schema.SubmissionCount, schema.CreatedBy,
		schema.CreatedAt, schema.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create form schema: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetSchema(ctx context.Context, id, tenantID uuid.UUID) (*FormSchema, error) {
	return r.scanSchema(r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, title, description, fields, status, is_template,
		        is_public, page_count, submission_count, created_by, created_at,
		        updated_at, deleted_at
		 FROM form_schemas
		 WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`,
		id, tenantID,
	))
}

func (r *PostgresRepository) UpdateSchema(ctx context.Context, schema *FormSchema) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE form_schemas
		 SET title = $1, description = $2, fields = $3, status = $4,
		     is_template = $5, is_public = $6, page_count = $7, updated_at = $8
		 WHERE id = $9 AND tenant_id = $10 AND deleted_at IS NULL`,
		schema.Title, schema.Description, schema.Fields, string(schema.Status),
		schema.IsTemplate, schema.IsPublic, schema.PageCount, schema.UpdatedAt,
		schema.ID, schema.TenantID,
	)
	if err != nil {
		return fmt.Errorf("update form schema: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrSchemaNotFound
	}
	return nil
}

func (r *PostgresRepository) SoftDeleteSchema(ctx context.Context, id, tenantID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE form_schemas SET deleted_at = NOW()
		 WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`,
		id, tenantID,
	)
	if err != nil {
		return fmt.Errorf("soft delete form schema: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrSchemaNotFound
	}
	return nil
}

func (r *PostgresRepository) ListSchemas(ctx context.Context, tenantID uuid.UUID, filter ListSchemasFilter) ([]*FormSchema, int, error) {
	conditions := []string{"tenant_id = $1", "deleted_at IS NULL"}
	args := []any{tenantID}
	argNum := 2

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argNum))
		args = append(args, string(*filter.Status))
		argNum++
	}
	if filter.IsTemplate != nil {
		conditions = append(conditions, fmt.Sprintf("is_template = $%d", argNum))
		args = append(args, *filter.IsTemplate)
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
		fmt.Sprintf("SELECT COUNT(*) FROM form_schemas %s", whereClause),
		args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count form schemas: %w", err)
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := max(filter.Offset, 0)

	query := fmt.Sprintf(`
		SELECT id, tenant_id, title, description, fields, status, is_template,
		       is_public, page_count, submission_count, created_by, created_at,
		       updated_at, deleted_at
		FROM form_schemas %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list form schemas: %w", err)
	}
	defer rows.Close()

	var out []*FormSchema
	for rows.Next() {
		s, scanErr := r.scanSchemaFromRows(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		out = append(out, s)
	}
	return out, total, rows.Err()
}

func (r *PostgresRepository) DuplicateSchema(ctx context.Context, id, tenantID uuid.UUID, newTitle string) (*FormSchema, error) {
	original, err := r.GetSchema(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	newID := uuid.New()
	title := newTitle
	if title == "" {
		title = original.Title + " (Copy)"
	}

	newSchema := &FormSchema{
		ID:          newID,
		TenantID:    tenantID,
		Title:       title,
		Description: original.Description,
		Fields:      append([]byte(nil), original.Fields...),
		Status:      FormSchemaStatusDraft,
		IsTemplate:  original.IsTemplate,
		IsPublic:    false,
		PageCount:   original.PageCount,
		CreatedBy:   original.CreatedBy,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := r.CreateSchema(ctx, newSchema); err != nil {
		return nil, fmt.Errorf("duplicate form schema: %w", err)
	}
	return newSchema, nil
}

// ============================================================================
// Submission
// ============================================================================

func (r *PostgresRepository) CreateSubmission(ctx context.Context, submission *FormSubmission, webhookIDs []uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = tx.Exec(ctx,
		`INSERT INTO form_submissions
		    (id, form_schema_id, tenant_id, answers, status, submitted_by, ip_address, submitted_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		submission.ID, submission.FormSchemaID, submission.TenantID,
		submission.Answers, string(submission.Status), submission.SubmittedBy,
		submission.IPAddress, submission.SubmittedAt,
	)
	if err != nil {
		return fmt.Errorf("insert form submission: %w", err)
	}

	// Enqueue one delivery row per active webhook.
	for _, wID := range webhookIDs {
		payload, marshalErr := json.Marshal(map[string]any{
			"submission_id":   submission.ID,
			"form_schema_id":  submission.FormSchemaID,
			"answers":         json.RawMessage(submission.Answers),
		})
		if marshalErr != nil {
			return fmt.Errorf("marshal delivery payload: %w", marshalErr)
		}

		_, err = tx.Exec(ctx,
			`INSERT INTO form_webhook_deliveries
			    (id, webhook_id, submission_id, tenant_id, payload, status,
			     attempt_count, max_attempts, next_attempt_at, created_at)
			 VALUES ($1,$2,$3,$4,$5,'pending',0,5,NOW(),NOW())`,
			uuid.New(), wID, submission.ID, submission.TenantID, payload,
		)
		if err != nil {
			return fmt.Errorf("enqueue webhook delivery: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepository) GetSubmission(ctx context.Context, id, tenantID uuid.UUID) (*FormSubmission, error) {
	return r.scanSubmission(r.pool.QueryRow(ctx,
		`SELECT id, form_schema_id, tenant_id, answers, status, submitted_by,
		        ip_address::text, submitted_at
		 FROM form_submissions
		 WHERE id = $1 AND tenant_id = $2`,
		id, tenantID,
	))
}

func (r *PostgresRepository) ListSubmissions(ctx context.Context, tenantID uuid.UUID, filter ListSubmissionsFilter) ([]*FormSubmission, int, error) {
	conditions := []string{"tenant_id = $1"}
	args := []any{tenantID}
	argNum := 2

	if filter.FormSchemaID != nil {
		conditions = append(conditions, fmt.Sprintf("form_schema_id = $%d", argNum))
		args = append(args, *filter.FormSchemaID)
		argNum++
	}
	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argNum))
		args = append(args, string(*filter.Status))
		argNum++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM form_submissions %s", whereClause),
		args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count form submissions: %w", err)
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := max(filter.Offset, 0)

	query := fmt.Sprintf(`
		SELECT id, form_schema_id, tenant_id, answers, status, submitted_by,
		       ip_address::text, submitted_at
		FROM form_submissions %s
		ORDER BY submitted_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list form submissions: %w", err)
	}
	defer rows.Close()

	var out []*FormSubmission
	for rows.Next() {
		s, scanErr := r.scanSubmissionFromRows(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		out = append(out, s)
	}
	return out, total, rows.Err()
}

func (r *PostgresRepository) UpdateSubmissionStatus(ctx context.Context, id, tenantID uuid.UUID, status FormSubmissionStatus) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE form_submissions SET status = $1
		 WHERE id = $2 AND tenant_id = $3`,
		string(status), id, tenantID,
	)
	if err != nil {
		return fmt.Errorf("update submission status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrSubmissionNotFound
	}
	return nil
}

func (r *PostgresRepository) ListSubmissionsForExport(ctx context.Context, formSchemaID, tenantID uuid.UUID, filter ExportFilter) ([]*FormSubmission, error) {
	conditions := []string{"form_schema_id = $1", "tenant_id = $2"}
	args := []any{formSchemaID, tenantID}
	argNum := 3

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argNum))
		args = append(args, string(*filter.Status))
		argNum++
	}
	if filter.SubmittedAfter != nil {
		conditions = append(conditions, fmt.Sprintf("submitted_at >= $%d", argNum))
		args = append(args, *filter.SubmittedAfter)
		argNum++
	}
	if filter.SubmittedBefore != nil {
		conditions = append(conditions, fmt.Sprintf("submitted_at <= $%d", argNum))
		args = append(args, *filter.SubmittedBefore)
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")
	query := fmt.Sprintf(`
		SELECT id, form_schema_id, tenant_id, answers, status, submitted_by,
		       ip_address::text, submitted_at
		FROM form_submissions %s
		ORDER BY submitted_at ASC
	`, whereClause)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list submissions for export: %w", err)
	}
	defer rows.Close()

	var out []*FormSubmission
	for rows.Next() {
		s, scanErr := r.scanSubmissionFromRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ============================================================================
// Webhook
// ============================================================================

func (r *PostgresRepository) CreateWebhook(ctx context.Context, webhook *FormWebhook) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO form_webhooks
		    (id, form_schema_id, tenant_id, url, secret, events, active, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		webhook.ID, webhook.FormSchemaID, webhook.TenantID,
		webhook.URL, webhook.Secret, webhook.Events, webhook.Active,
		webhook.CreatedAt, webhook.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create form webhook: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetWebhook(ctx context.Context, id, tenantID uuid.UUID) (*FormWebhook, error) {
	return r.scanWebhook(r.pool.QueryRow(ctx,
		`SELECT id, form_schema_id, tenant_id, url, secret, events, active,
		        last_triggered_at, last_status, created_at, updated_at
		 FROM form_webhooks
		 WHERE id = $1 AND tenant_id = $2`,
		id, tenantID,
	))
}

func (r *PostgresRepository) UpdateWebhook(ctx context.Context, webhook *FormWebhook) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE form_webhooks
		 SET url = $1, secret = $2, events = $3, active = $4, updated_at = $5
		 WHERE id = $6 AND tenant_id = $7`,
		webhook.URL, webhook.Secret, webhook.Events, webhook.Active, webhook.UpdatedAt,
		webhook.ID, webhook.TenantID,
	)
	if err != nil {
		return fmt.Errorf("update form webhook: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrWebhookNotFound
	}
	return nil
}

func (r *PostgresRepository) DeleteWebhook(ctx context.Context, id, tenantID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM form_webhooks WHERE id = $1 AND tenant_id = $2`,
		id, tenantID,
	)
	if err != nil {
		return fmt.Errorf("delete form webhook: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrWebhookNotFound
	}
	return nil
}

func (r *PostgresRepository) ListWebhooks(ctx context.Context, formSchemaID, tenantID uuid.UUID) ([]*FormWebhook, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, form_schema_id, tenant_id, url, secret, events, active,
		        last_triggered_at, last_status, created_at, updated_at
		 FROM form_webhooks
		 WHERE form_schema_id = $1 AND tenant_id = $2
		 ORDER BY created_at ASC`,
		formSchemaID, tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("list form webhooks: %w", err)
	}
	defer rows.Close()

	var out []*FormWebhook
	for rows.Next() {
		w, scanErr := r.scanWebhookFromRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) ListActiveWebhooksForSchema(ctx context.Context, formSchemaID uuid.UUID) ([]*FormWebhook, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, form_schema_id, tenant_id, url, secret, events, active,
		        last_triggered_at, last_status, created_at, updated_at
		 FROM form_webhooks
		 WHERE form_schema_id = $1 AND active = TRUE
		 ORDER BY created_at ASC`,
		formSchemaID,
	)
	if err != nil {
		return nil, fmt.Errorf("list active webhooks for schema: %w", err)
	}
	defer rows.Close()

	var out []*FormWebhook
	for rows.Next() {
		w, scanErr := r.scanWebhookFromRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// ============================================================================
// Delivery
// ============================================================================

// ClaimPendingDeliveries returns up to batchSize pending deliveries whose
// next_attempt_at is in the past, using SKIP LOCKED so parallel workers
// don't race. Callers are responsible for updating status after the HTTP call.
func (r *PostgresRepository) ClaimPendingDeliveries(ctx context.Context, batchSize int) ([]*WebhookDelivery, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, webhook_id, submission_id, tenant_id, payload, status,
		        attempt_count, max_attempts, next_attempt_at, last_error,
		        last_response_code, created_at, delivered_at
		 FROM form_webhook_deliveries
		 WHERE status = 'pending' AND next_attempt_at <= NOW()
		 ORDER BY next_attempt_at ASC
		 LIMIT $1
		 FOR UPDATE SKIP LOCKED`,
		batchSize,
	)
	if err != nil {
		return nil, fmt.Errorf("claim pending deliveries: %w", err)
	}
	defer rows.Close()

	var out []*WebhookDelivery
	for rows.Next() {
		d, scanErr := r.scanDeliveryFromRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) MarkDeliveryResult(ctx context.Context, id uuid.UUID, status WebhookDeliveryStatus, attemptCount int, nextAttemptAt *time.Time, lastError string, lastCode int) error {
	var deliveredAt *time.Time
	if status == WebhookDeliveryStatusDelivered {
		now := time.Now().UTC()
		deliveredAt = &now
	}

	var lastErrPtr *string
	if lastError != "" {
		lastErrPtr = &lastError
	}
	var lastCodePtr *int
	if lastCode != 0 {
		lastCodePtr = &lastCode
	}

	tag, err := r.pool.Exec(ctx,
		`UPDATE form_webhook_deliveries
		 SET status = $1, attempt_count = $2, next_attempt_at = $3,
		     last_error = $4, last_response_code = $5, delivered_at = $6
		 WHERE id = $7`,
		string(status), attemptCount, nextAttemptAt,
		lastErrPtr, lastCodePtr, deliveredAt,
		id,
	)
	if err != nil {
		return fmt.Errorf("mark delivery result: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrDeliveryNotFound
	}
	return nil
}

func (r *PostgresRepository) ListWebhookDeliveries(ctx context.Context, tenantID uuid.UUID, filter ListDeliveriesFilter) ([]*WebhookDelivery, int, error) {
	conditions := []string{"tenant_id = $1"}
	args := []any{tenantID}
	argNum := 2

	if filter.WebhookID != nil {
		conditions = append(conditions, fmt.Sprintf("webhook_id = $%d", argNum))
		args = append(args, *filter.WebhookID)
		argNum++
	}
	if filter.SubmissionID != nil {
		conditions = append(conditions, fmt.Sprintf("submission_id = $%d", argNum))
		args = append(args, *filter.SubmissionID)
		argNum++
	}
	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argNum))
		args = append(args, string(*filter.Status))
		argNum++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM form_webhook_deliveries %s", whereClause),
		args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count webhook deliveries: %w", err)
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := max(filter.Offset, 0)

	query := fmt.Sprintf(`
		SELECT id, webhook_id, submission_id, tenant_id, payload, status,
		       attempt_count, max_attempts, next_attempt_at, last_error,
		       last_response_code, created_at, delivered_at
		FROM form_webhook_deliveries %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list webhook deliveries: %w", err)
	}
	defer rows.Close()

	var out []*WebhookDelivery
	for rows.Next() {
		d, scanErr := r.scanDeliveryFromRows(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		out = append(out, d)
	}
	return out, total, rows.Err()
}

func (r *PostgresRepository) GetWebhookDelivery(ctx context.Context, id, tenantID uuid.UUID) (*WebhookDelivery, error) {
	return r.scanDelivery(r.pool.QueryRow(ctx,
		`SELECT id, webhook_id, submission_id, tenant_id, payload, status,
		        attempt_count, max_attempts, next_attempt_at, last_error,
		        last_response_code, created_at, delivered_at
		 FROM form_webhook_deliveries
		 WHERE id = $1 AND tenant_id = $2`,
		id, tenantID,
	))
}

// ============================================================================
// Scan helpers — FormSchema
// ============================================================================

func (r *PostgresRepository) scanSchema(row pgx.Row) (*FormSchema, error) {
	var s FormSchema
	var status string
	err := row.Scan(
		&s.ID, &s.TenantID, &s.Title, &s.Description, &s.Fields,
		&status, &s.IsTemplate, &s.IsPublic, &s.PageCount, &s.SubmissionCount,
		&s.CreatedBy, &s.CreatedAt, &s.UpdatedAt, &s.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSchemaNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan form schema: %w", err)
	}
	s.Status = FormSchemaStatus(status)
	return &s, nil
}

func (r *PostgresRepository) scanSchemaFromRows(rows pgx.Rows) (*FormSchema, error) {
	var s FormSchema
	var status string
	err := rows.Scan(
		&s.ID, &s.TenantID, &s.Title, &s.Description, &s.Fields,
		&status, &s.IsTemplate, &s.IsPublic, &s.PageCount, &s.SubmissionCount,
		&s.CreatedBy, &s.CreatedAt, &s.UpdatedAt, &s.DeletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan form schema row: %w", err)
	}
	s.Status = FormSchemaStatus(status)
	return &s, nil
}

// ============================================================================
// Scan helpers — FormSubmission
// ============================================================================

func (r *PostgresRepository) scanSubmission(row pgx.Row) (*FormSubmission, error) {
	var s FormSubmission
	var status string
	err := row.Scan(
		&s.ID, &s.FormSchemaID, &s.TenantID, &s.Answers,
		&status, &s.SubmittedBy, &s.IPAddress, &s.SubmittedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSubmissionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan form submission: %w", err)
	}
	s.Status = FormSubmissionStatus(status)
	return &s, nil
}

func (r *PostgresRepository) scanSubmissionFromRows(rows pgx.Rows) (*FormSubmission, error) {
	var s FormSubmission
	var status string
	err := rows.Scan(
		&s.ID, &s.FormSchemaID, &s.TenantID, &s.Answers,
		&status, &s.SubmittedBy, &s.IPAddress, &s.SubmittedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan form submission row: %w", err)
	}
	s.Status = FormSubmissionStatus(status)
	return &s, nil
}

// ============================================================================
// Scan helpers — FormWebhook
// ============================================================================

func (r *PostgresRepository) scanWebhook(row pgx.Row) (*FormWebhook, error) {
	var w FormWebhook
	err := row.Scan(
		&w.ID, &w.FormSchemaID, &w.TenantID, &w.URL, &w.Secret,
		&w.Events, &w.Active, &w.LastTriggeredAt, &w.LastStatus,
		&w.CreatedAt, &w.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrWebhookNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan form webhook: %w", err)
	}
	return &w, nil
}

func (r *PostgresRepository) scanWebhookFromRows(rows pgx.Rows) (*FormWebhook, error) {
	var w FormWebhook
	err := rows.Scan(
		&w.ID, &w.FormSchemaID, &w.TenantID, &w.URL, &w.Secret,
		&w.Events, &w.Active, &w.LastTriggeredAt, &w.LastStatus,
		&w.CreatedAt, &w.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan form webhook row: %w", err)
	}
	return &w, nil
}

// ============================================================================
// Scan helpers — WebhookDelivery
// ============================================================================

func (r *PostgresRepository) scanDelivery(row pgx.Row) (*WebhookDelivery, error) {
	var d WebhookDelivery
	var status string
	err := row.Scan(
		&d.ID, &d.WebhookID, &d.SubmissionID, &d.TenantID, &d.Payload,
		&status, &d.AttemptCount, &d.MaxAttempts, &d.NextAttemptAt,
		&d.LastError, &d.LastResponseCode, &d.CreatedAt, &d.DeliveredAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrDeliveryNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan webhook delivery: %w", err)
	}
	d.Status = WebhookDeliveryStatus(status)
	return &d, nil
}

func (r *PostgresRepository) scanDeliveryFromRows(rows pgx.Rows) (*WebhookDelivery, error) {
	var d WebhookDelivery
	var status string
	err := rows.Scan(
		&d.ID, &d.WebhookID, &d.SubmissionID, &d.TenantID, &d.Payload,
		&status, &d.AttemptCount, &d.MaxAttempts, &d.NextAttemptAt,
		&d.LastError, &d.LastResponseCode, &d.CreatedAt, &d.DeliveredAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan webhook delivery row: %w", err)
	}
	d.Status = WebhookDeliveryStatus(status)
	return &d, nil
}

// ============================================================================
// Stats
// ============================================================================

func (r *PostgresRepository) GetFormStats(ctx context.Context, formSchemaID, tenantID uuid.UUID) (*FormStats, error) {
	var total, newCount, last7d, last30d int
	err := r.pool.QueryRow(ctx,
		`SELECT
		  COUNT(*) AS total,
		  COUNT(*) FILTER (WHERE status = 'new') AS new_count,
		  COUNT(*) FILTER (WHERE submitted_at >= NOW() - INTERVAL '7 days') AS last_7d,
		  COUNT(*) FILTER (WHERE submitted_at >= NOW() - INTERVAL '30 days') AS last_30d
		FROM form_submissions
		WHERE form_schema_id = $1 AND tenant_id = $2`,
		formSchemaID, tenantID,
	).Scan(&total, &newCount, &last7d, &last30d)
	if err != nil {
		return nil, fmt.Errorf("get form stats: %w", err)
	}
	return &FormStats{
		FormSchemaID: formSchemaID,
		TotalCount:   total,
		NewCount:     newCount,
		Last7dCount:  last7d,
		Last30dCount: last30d,
	}, nil
}

// compile-time interface check
var _ Repository = (*PostgresRepository)(nil)

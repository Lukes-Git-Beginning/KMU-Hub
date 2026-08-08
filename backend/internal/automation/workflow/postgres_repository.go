package workflow

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

// PostgresRepository implements Repository, ExecutionRepository, and
// TemplateRepository using PostgreSQL with pgxpool.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL-backed automation repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// Compile-time interface compliance checks.
var (
	_ Repository          = (*PostgresRepository)(nil)
	_ ExecutionRepository = (*PostgresRepository)(nil)
	_ TemplateRepository  = (*PostgresRepository)(nil)
)

// ============================================================================
// Repository (automations)
// ============================================================================

func (r *PostgresRepository) Create(ctx context.Context, a *models.Automation) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	now := time.Now()
	a.CreatedAt = now
	a.UpdatedAt = now

	query := `
		INSERT INTO automations (
			id, tenant_id, name, description, scope, owner_id, trigger_type,
			trigger_config, conditions, actions, is_active, max_steps,
			template_id, last_triggered_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`

	_, err := r.pool.Exec(ctx, query,
		a.ID, a.TenantID, a.Name, a.Description, a.Scope, a.OwnerID, a.TriggerType,
		a.TriggerConfig, a.Conditions, a.Actions, a.IsActive, a.MaxSteps,
		a.TemplateID, a.LastTriggeredAt, a.CreatedAt, a.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) Update(ctx context.Context, a *models.Automation) error {
	a.UpdatedAt = time.Now()

	query := `
		UPDATE automations SET
			name = $3, description = $4, scope = $5, trigger_type = $6,
			trigger_config = $7, conditions = $8, actions = $9,
			is_active = $10, max_steps = $11, template_id = $12,
			updated_at = $13
		WHERE id = $1 AND tenant_id = $2`

	tag, err := r.pool.Exec(ctx, query,
		a.ID, a.TenantID, a.Name, a.Description, a.Scope, a.TriggerType,
		a.TriggerConfig, a.Conditions, a.Actions,
		a.IsActive, a.MaxSteps, a.TemplateID,
		a.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrAutomationNotFound
	}
	return nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	query := `DELETE FROM automations WHERE id = $1 AND tenant_id = $2`

	tag, err := r.pool.Exec(ctx, query, id, tenantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrAutomationNotFound
	}
	return nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.Automation, error) {
	query := `
		SELECT id, tenant_id, name, description, scope, owner_id, trigger_type,
			trigger_config, conditions, actions, is_active, max_steps,
			template_id, last_triggered_at, created_at, updated_at
		FROM automations WHERE id = $1 AND tenant_id = $2`

	a := &models.Automation{}
	err := r.pool.QueryRow(ctx, query, id, tenantID).Scan(
		&a.ID, &a.TenantID, &a.Name, &a.Description, &a.Scope, &a.OwnerID, &a.TriggerType,
		&a.TriggerConfig, &a.Conditions, &a.Actions, &a.IsActive, &a.MaxSteps,
		&a.TemplateID, &a.LastTriggeredAt, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAutomationNotFound
		}
		return nil, err
	}
	return a, nil
}

// GetByIDUnscoped retrieves an automation by ID without a tenant_id filter.
// See the Repository interface doc comment: callers must wrap ctx with
// sysctx.With and must not skip the caller-side trigger-type/active checks.
func (r *PostgresRepository) GetByIDUnscoped(ctx context.Context, id uuid.UUID) (*models.Automation, error) {
	query := `
		SELECT id, tenant_id, name, description, scope, owner_id, trigger_type,
			trigger_config, conditions, actions, is_active, max_steps,
			template_id, last_triggered_at, created_at, updated_at
		FROM automations WHERE id = $1`

	a := &models.Automation{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&a.ID, &a.TenantID, &a.Name, &a.Description, &a.Scope, &a.OwnerID, &a.TriggerType,
		&a.TriggerConfig, &a.Conditions, &a.Actions, &a.IsActive, &a.MaxSteps,
		&a.TemplateID, &a.LastTriggeredAt, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAutomationNotFound
		}
		return nil, err
	}
	return a, nil
}

func (r *PostgresRepository) List(ctx context.Context, filter ListFilter) ([]*models.Automation, int, error) {
	// tenant_id is always the first condition for isolation
	conditions := []string{fmt.Sprintf("tenant_id = $%d", 1)}
	args := []any{filter.TenantID}
	argIndex := 2

	if filter.OwnerID != nil {
		conditions = append(conditions, fmt.Sprintf("owner_id = $%d", argIndex))
		args = append(args, *filter.OwnerID)
		argIndex++
	}
	if filter.Scope != nil {
		conditions = append(conditions, fmt.Sprintf("scope = $%d", argIndex))
		args = append(args, *filter.Scope)
		argIndex++
	}
	if filter.TriggerType != nil {
		conditions = append(conditions, fmt.Sprintf("trigger_type = $%d", argIndex))
		args = append(args, *filter.TriggerType)
		argIndex++
	}
	if filter.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", argIndex))
		args = append(args, *filter.IsActive)
		argIndex++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, name, description, scope, owner_id, trigger_type,
			trigger_config, conditions, actions, is_active, max_steps,
			template_id, last_triggered_at, created_at, updated_at,
			COUNT(*) OVER() AS total_count
		FROM automations %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, where, argIndex, argIndex+1)

	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var automations []*models.Automation
	var totalCount int

	for rows.Next() {
		a := &models.Automation{}
		err := rows.Scan(
			&a.ID, &a.TenantID, &a.Name, &a.Description, &a.Scope, &a.OwnerID, &a.TriggerType,
			&a.TriggerConfig, &a.Conditions, &a.Actions, &a.IsActive, &a.MaxSteps,
			&a.TemplateID, &a.LastTriggeredAt, &a.CreatedAt, &a.UpdatedAt,
			&totalCount,
		)
		if err != nil {
			return nil, 0, err
		}
		automations = append(automations, a)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return automations, totalCount, nil
}

func (r *PostgresRepository) ListActiveByTriggerType(ctx context.Context, triggerType string) ([]*models.Automation, error) {
	query := `
		SELECT id, tenant_id, name, description, scope, owner_id, trigger_type,
			trigger_config, conditions, actions, is_active, max_steps,
			template_id, last_triggered_at, created_at, updated_at
		FROM automations
		WHERE trigger_type = $1 AND is_active = true
		ORDER BY created_at ASC`

	rows, err := r.pool.Query(ctx, query, triggerType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanAutomations(rows)
}

// ListActiveTimeBased returns active automations whose trigger_type is in
// triggerTypes. Unlike scanAutomations' callers, this also selects
// last_polled_at (needed by the caller to attempt an optimistic-concurrency
// claim via ClaimTimeTrigger), so it scans its own rows rather than sharing
// scanAutomations' column set.
func (r *PostgresRepository) ListActiveTimeBased(ctx context.Context, triggerTypes []string) ([]*models.Automation, error) {
	if len(triggerTypes) == 0 {
		return nil, nil
	}

	query := `
		SELECT id, tenant_id, name, description, scope, owner_id, trigger_type,
			trigger_config, conditions, actions, is_active, max_steps,
			template_id, last_triggered_at, last_polled_at, created_at, updated_at
		FROM automations
		WHERE trigger_type = ANY($1) AND is_active = true
		ORDER BY created_at ASC`

	rows, err := r.pool.Query(ctx, query, triggerTypes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var automations []*models.Automation
	for rows.Next() {
		a := &models.Automation{}
		if scanErr := rows.Scan(
			&a.ID, &a.TenantID, &a.Name, &a.Description, &a.Scope, &a.OwnerID, &a.TriggerType,
			&a.TriggerConfig, &a.Conditions, &a.Actions, &a.IsActive, &a.MaxSteps,
			&a.TemplateID, &a.LastTriggeredAt, &a.LastPolledAt, &a.CreatedAt, &a.UpdatedAt,
		); scanErr != nil {
			return nil, scanErr
		}
		automations = append(automations, a)
	}
	return automations, rows.Err()
}

// ClaimTimeTrigger atomically advances last_polled_at iff it still matches
// previousLastPolledAt, exactly mirroring
// internal/berichte/scheduler.PostgresRepository.ClaimSchedule's
// optimistic-concurrency pattern on report_schedules.last_run_at.
func (r *PostgresRepository) ClaimTimeTrigger(ctx context.Context, id uuid.UUID, previousLastPolledAt *time.Time, now time.Time) (bool, error) {
	var tag pgconn.CommandTag
	var err error
	if previousLastPolledAt == nil {
		tag, err = r.pool.Exec(ctx,
			`UPDATE automations SET last_polled_at = $1 WHERE id = $2 AND last_polled_at IS NULL`,
			now, id,
		)
	} else {
		tag, err = r.pool.Exec(ctx,
			`UPDATE automations SET last_polled_at = $1 WHERE id = $2 AND last_polled_at = $3`,
			now, id, *previousLastPolledAt,
		)
	}
	if err != nil {
		return false, fmt.Errorf("claim time trigger: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

// ClaimTimeTriggerFire inserts the (automation_id, entity_key) fire record and
// reports whether this caller was the one that inserted it. ON CONFLICT DO
// NOTHING makes the check-and-record a single atomic statement, so two poller
// instances resolving the same due entity in the same tick cannot both fire:
// the loser's INSERT affects zero rows.
func (r *PostgresRepository) ClaimTimeTriggerFire(ctx context.Context, tenantID, automationID uuid.UUID, entityKey string, now time.Time) (bool, error) {
	tag, err := r.pool.Exec(ctx,
		`INSERT INTO automation_time_trigger_fires (tenant_id, automation_id, entity_key, fired_at)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (automation_id, entity_key) DO NOTHING`,
		tenantID, automationID, entityKey, now,
	)
	if err != nil {
		return false, fmt.Errorf("claim time trigger fire: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *PostgresRepository) SetActive(ctx context.Context, id uuid.UUID, tenantID uuid.UUID, active bool) error {
	query := `UPDATE automations SET is_active = $3, updated_at = NOW() WHERE id = $1 AND tenant_id = $2`

	tag, err := r.pool.Exec(ctx, query, id, tenantID, active)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrAutomationNotFound
	}
	return nil
}

func (r *PostgresRepository) UpdateLastTriggered(ctx context.Context, id uuid.UUID, at time.Time) error {
	// Internal-only: called by automation engine, not user-facing. No tenant filter needed.
	query := `UPDATE automations SET last_triggered_at = $2, updated_at = NOW() WHERE id = $1`

	tag, err := r.pool.Exec(ctx, query, id, at)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrAutomationNotFound
	}
	return nil
}

// ============================================================================
// ExecutionRepository
// ============================================================================

func (r *PostgresRepository) CreateExecution(ctx context.Context, e *models.AutomationExecution) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	e.StartedAt = time.Now()

	query := `
		INSERT INTO automation_executions (
			id, tenant_id, automation_id, chain_id, trigger_event, condition_result,
			status, steps, error_message, started_at, completed_at, duration_ms
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	_, err := r.pool.Exec(ctx, query,
		e.ID, e.TenantID, e.AutomationID, e.ChainID, e.TriggerEvent, e.ConditionResult,
		e.Status, e.Steps, e.ErrorMessage, e.StartedAt, e.CompletedAt, e.DurationMs,
	)
	return err
}

func (r *PostgresRepository) UpdateExecution(ctx context.Context, e *models.AutomationExecution) error {
	query := `
		UPDATE automation_executions SET
			condition_result = $3, status = $4, steps = $5,
			error_message = $6, completed_at = $7, duration_ms = $8
		WHERE id = $1 AND tenant_id = $2`

	tag, err := r.pool.Exec(ctx, query,
		e.ID, e.TenantID, e.ConditionResult, e.Status, e.Steps,
		e.ErrorMessage, e.CompletedAt, e.DurationMs,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrExecutionNotFound
	}
	return nil
}

func (r *PostgresRepository) GetExecution(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.AutomationExecution, error) {
	query := `
		SELECT id, tenant_id, automation_id, chain_id, trigger_event, condition_result,
			status, steps, error_message, started_at, completed_at, duration_ms
		FROM automation_executions WHERE id = $1 AND tenant_id = $2`

	e := &models.AutomationExecution{}
	err := r.pool.QueryRow(ctx, query, id, tenantID).Scan(
		&e.ID, &e.TenantID, &e.AutomationID, &e.ChainID, &e.TriggerEvent, &e.ConditionResult,
		&e.Status, &e.Steps, &e.ErrorMessage, &e.StartedAt, &e.CompletedAt, &e.DurationMs,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrExecutionNotFound
		}
		return nil, err
	}
	return e, nil
}

func (r *PostgresRepository) ListExecutions(ctx context.Context, filter ExecutionFilter) ([]*models.AutomationExecution, int, error) {
	// tenant_id is always the first condition for isolation
	conditions := []string{fmt.Sprintf("tenant_id = $%d", 1)}
	args := []any{filter.TenantID}
	argIndex := 2

	if filter.AutomationID != nil {
		conditions = append(conditions, fmt.Sprintf("automation_id = $%d", argIndex))
		args = append(args, *filter.AutomationID)
		argIndex++
	}
	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, *filter.Status)
		argIndex++
	}
	if filter.StartedAfter != nil {
		conditions = append(conditions, fmt.Sprintf("started_at >= $%d", argIndex))
		args = append(args, *filter.StartedAfter)
		argIndex++
	}
	if filter.StartedBefore != nil {
		conditions = append(conditions, fmt.Sprintf("started_at <= $%d", argIndex))
		args = append(args, *filter.StartedBefore)
		argIndex++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, automation_id, chain_id, trigger_event, condition_result,
			status, steps, error_message, started_at, completed_at, duration_ms,
			COUNT(*) OVER() AS total_count
		FROM automation_executions %s
		ORDER BY started_at DESC
		LIMIT $%d OFFSET $%d`, where, argIndex, argIndex+1)

	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var executions []*models.AutomationExecution
	var totalCount int

	for rows.Next() {
		e := &models.AutomationExecution{}
		err := rows.Scan(
			&e.ID, &e.TenantID, &e.AutomationID, &e.ChainID, &e.TriggerEvent, &e.ConditionResult,
			&e.Status, &e.Steps, &e.ErrorMessage, &e.StartedAt, &e.CompletedAt, &e.DurationMs,
			&totalCount,
		)
		if err != nil {
			return nil, 0, err
		}
		executions = append(executions, e)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return executions, totalCount, nil
}

func (r *PostgresRepository) CleanupOldExecutions(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	query := `DELETE FROM automation_executions WHERE completed_at IS NOT NULL AND started_at < $1`

	tag, err := r.pool.Exec(ctx, query, cutoff)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// ============================================================================
// TemplateRepository
// ============================================================================

func (r *PostgresRepository) ListTemplates(ctx context.Context, category *string) ([]*models.AutomationTemplate, error) {
	var query string
	var args []any

	if category != nil {
		query = `
			SELECT id, name, description, category, complexity, trigger_type,
				trigger_config, conditions, actions, created_at
			FROM automation_templates
			WHERE category = $1
			ORDER BY name ASC`
		args = append(args, *category)
	} else {
		query = `
			SELECT id, name, description, category, complexity, trigger_type,
				trigger_config, conditions, actions, created_at
			FROM automation_templates
			ORDER BY category, name ASC`
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []*models.AutomationTemplate
	for rows.Next() {
		t := &models.AutomationTemplate{}
		err := rows.Scan(
			&t.ID, &t.Name, &t.Description, &t.Category, &t.Complexity, &t.TriggerType,
			&t.TriggerConfig, &t.Conditions, &t.Actions, &t.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}

	return templates, rows.Err()
}

func (r *PostgresRepository) GetTemplate(ctx context.Context, id string) (*models.AutomationTemplate, error) {
	query := `
		SELECT id, name, description, category, complexity, trigger_type,
			trigger_config, conditions, actions, created_at
		FROM automation_templates WHERE id = $1`

	t := &models.AutomationTemplate{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.Name, &t.Description, &t.Category, &t.Complexity, &t.TriggerType,
		&t.TriggerConfig, &t.Conditions, &t.Actions, &t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTemplateNotFound
		}
		return nil, err
	}
	return t, nil
}

func (r *PostgresRepository) UpsertTemplate(ctx context.Context, t *models.AutomationTemplate) error {
	t.CreatedAt = time.Now()

	query := `
		INSERT INTO automation_templates (
			id, name, description, category, complexity, trigger_type,
			trigger_config, conditions, actions, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			category = EXCLUDED.category,
			complexity = EXCLUDED.complexity,
			trigger_type = EXCLUDED.trigger_type,
			trigger_config = EXCLUDED.trigger_config,
			conditions = EXCLUDED.conditions,
			actions = EXCLUDED.actions`

	_, err := r.pool.Exec(ctx, query,
		t.ID, t.Name, t.Description, t.Category, t.Complexity, t.TriggerType,
		t.TriggerConfig, t.Conditions, t.Actions, t.CreatedAt,
	)
	return err
}

// ============================================================================
// Helpers
// ============================================================================

func scanAutomations(rows pgx.Rows) ([]*models.Automation, error) {
	var automations []*models.Automation
	for rows.Next() {
		a := &models.Automation{}
		err := rows.Scan(
			&a.ID, &a.TenantID, &a.Name, &a.Description, &a.Scope, &a.OwnerID, &a.TriggerType,
			&a.TriggerConfig, &a.Conditions, &a.Actions, &a.IsActive, &a.MaxSteps,
			&a.TemplateID, &a.LastTriggeredAt, &a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		automations = append(automations, a)
	}
	return automations, rows.Err()
}

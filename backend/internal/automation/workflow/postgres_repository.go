package workflow

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
			id, name, description, scope, owner_id, trigger_type,
			trigger_config, conditions, actions, is_active, max_steps,
			template_id, last_triggered_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`

	_, err := r.pool.Exec(ctx, query,
		a.ID, a.Name, a.Description, a.Scope, a.OwnerID, a.TriggerType,
		a.TriggerConfig, a.Conditions, a.Actions, a.IsActive, a.MaxSteps,
		a.TemplateID, a.LastTriggeredAt, a.CreatedAt, a.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) Update(ctx context.Context, a *models.Automation) error {
	a.UpdatedAt = time.Now()

	query := `
		UPDATE automations SET
			name = $2, description = $3, scope = $4, trigger_type = $5,
			trigger_config = $6, conditions = $7, actions = $8,
			is_active = $9, max_steps = $10, template_id = $11,
			updated_at = $12
		WHERE id = $1`

	tag, err := r.pool.Exec(ctx, query,
		a.ID, a.Name, a.Description, a.Scope, a.TriggerType,
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

func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM automations WHERE id = $1`

	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrAutomationNotFound
	}
	return nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Automation, error) {
	query := `
		SELECT id, name, description, scope, owner_id, trigger_type,
			trigger_config, conditions, actions, is_active, max_steps,
			template_id, last_triggered_at, created_at, updated_at
		FROM automations WHERE id = $1`

	a := &models.Automation{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&a.ID, &a.Name, &a.Description, &a.Scope, &a.OwnerID, &a.TriggerType,
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
	var conditions []string
	var args []any
	argIndex := 1

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

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(`
		SELECT id, name, description, scope, owner_id, trigger_type,
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
			&a.ID, &a.Name, &a.Description, &a.Scope, &a.OwnerID, &a.TriggerType,
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
		SELECT id, name, description, scope, owner_id, trigger_type,
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

func (r *PostgresRepository) ListActiveTimeBased(ctx context.Context) ([]*models.Automation, error) {
	query := `
		SELECT id, name, description, scope, owner_id, trigger_type,
			trigger_config, conditions, actions, is_active, max_steps,
			template_id, last_triggered_at, created_at, updated_at
		FROM automations
		WHERE trigger_type = 'time_based' AND is_active = true
		ORDER BY created_at ASC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanAutomations(rows)
}

func (r *PostgresRepository) SetActive(ctx context.Context, id uuid.UUID, active bool) error {
	query := `UPDATE automations SET is_active = $2, updated_at = NOW() WHERE id = $1`

	tag, err := r.pool.Exec(ctx, query, id, active)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrAutomationNotFound
	}
	return nil
}

func (r *PostgresRepository) UpdateLastTriggered(ctx context.Context, id uuid.UUID, at time.Time) error {
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
			id, automation_id, chain_id, trigger_event, condition_result,
			status, steps, error_message, started_at, completed_at, duration_ms
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err := r.pool.Exec(ctx, query,
		e.ID, e.AutomationID, e.ChainID, e.TriggerEvent, e.ConditionResult,
		e.Status, e.Steps, e.ErrorMessage, e.StartedAt, e.CompletedAt, e.DurationMs,
	)
	return err
}

func (r *PostgresRepository) UpdateExecution(ctx context.Context, e *models.AutomationExecution) error {
	query := `
		UPDATE automation_executions SET
			condition_result = $2, status = $3, steps = $4,
			error_message = $5, completed_at = $6, duration_ms = $7
		WHERE id = $1`

	tag, err := r.pool.Exec(ctx, query,
		e.ID, e.ConditionResult, e.Status, e.Steps,
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

func (r *PostgresRepository) GetExecution(ctx context.Context, id uuid.UUID) (*models.AutomationExecution, error) {
	query := `
		SELECT id, automation_id, chain_id, trigger_event, condition_result,
			status, steps, error_message, started_at, completed_at, duration_ms
		FROM automation_executions WHERE id = $1`

	e := &models.AutomationExecution{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&e.ID, &e.AutomationID, &e.ChainID, &e.TriggerEvent, &e.ConditionResult,
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
	var conditions []string
	var args []any
	argIndex := 1

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

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(`
		SELECT id, automation_id, chain_id, trigger_event, condition_result,
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
			&e.ID, &e.AutomationID, &e.ChainID, &e.TriggerEvent, &e.ConditionResult,
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
			&a.ID, &a.Name, &a.Description, &a.Scope, &a.OwnerID, &a.TriggerType,
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

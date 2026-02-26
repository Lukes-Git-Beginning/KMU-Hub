package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// WorkflowRuleRepository handles workflow rule persistence
type WorkflowRuleRepository struct {
	pool *pgxpool.Pool
}

// NewWorkflowRuleRepository creates a new workflow rule repository
func NewWorkflowRuleRepository(pool *pgxpool.Pool) *WorkflowRuleRepository {
	return &WorkflowRuleRepository{pool: pool}
}

// Create inserts a new workflow rule
func (r *WorkflowRuleRepository) Create(ctx context.Context, rule *models.WorkflowRule) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO workflow_rules (id, tenant_id, installation_id, name, description, trigger_event, conditions, actions, priority, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		rule.ID, rule.TenantID, rule.InstallationID, rule.Name, rule.Description,
		rule.TriggerEvent, rule.Conditions, rule.Actions,
		rule.Priority, rule.Enabled, rule.CreatedAt, rule.UpdatedAt,
	)
	return err
}

// GetByID retrieves a workflow rule by ID
func (r *WorkflowRuleRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.WorkflowRule, error) {
	rule := &models.WorkflowRule{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, installation_id, name, description, trigger_event, conditions, actions, priority, enabled, created_at, updated_at
		FROM workflow_rules WHERE id = $1`, id,
	).Scan(&rule.ID, &rule.TenantID, &rule.InstallationID, &rule.Name, &rule.Description,
		&rule.TriggerEvent, &rule.Conditions, &rule.Actions,
		&rule.Priority, &rule.Enabled, &rule.CreatedAt, &rule.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return rule, err
}

// List retrieves workflow rules for a tenant with optional trigger event filter
func (r *WorkflowRuleRepository) List(ctx context.Context, tenantID uuid.UUID, triggerEvent string) ([]*models.WorkflowRule, error) {
	query := `
		SELECT id, tenant_id, installation_id, name, description, trigger_event, conditions, actions, priority, enabled, created_at, updated_at
		FROM workflow_rules WHERE tenant_id = $1`
	args := []any{tenantID}

	if triggerEvent != "" {
		query += ` AND trigger_event = $2`
		args = append(args, triggerEvent)
	}
	query += ` ORDER BY priority ASC, name ASC`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*models.WorkflowRule
	for rows.Next() {
		rule := &models.WorkflowRule{}
		if scanErr := rows.Scan(&rule.ID, &rule.TenantID, &rule.InstallationID, &rule.Name, &rule.Description,
			&rule.TriggerEvent, &rule.Conditions, &rule.Actions,
			&rule.Priority, &rule.Enabled, &rule.CreatedAt, &rule.UpdatedAt,
		); scanErr != nil {
			return nil, scanErr
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

// ListEnabled retrieves only enabled workflow rules for a tenant and trigger event
func (r *WorkflowRuleRepository) ListEnabled(ctx context.Context, tenantID uuid.UUID, triggerEvent string) ([]*models.WorkflowRule, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, installation_id, name, description, trigger_event, conditions, actions, priority, enabled, created_at, updated_at
		FROM workflow_rules
		WHERE tenant_id = $1 AND trigger_event = $2 AND enabled = true
		ORDER BY priority ASC`, tenantID, triggerEvent,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*models.WorkflowRule
	for rows.Next() {
		rule := &models.WorkflowRule{}
		if scanErr := rows.Scan(&rule.ID, &rule.TenantID, &rule.InstallationID, &rule.Name, &rule.Description,
			&rule.TriggerEvent, &rule.Conditions, &rule.Actions,
			&rule.Priority, &rule.Enabled, &rule.CreatedAt, &rule.UpdatedAt,
		); scanErr != nil {
			return nil, scanErr
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

// Update modifies a workflow rule
func (r *WorkflowRuleRepository) Update(ctx context.Context, rule *models.WorkflowRule) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE workflow_rules
		SET name = $2, description = $3, conditions = $4, actions = $5, priority = $6, enabled = $7, updated_at = NOW()
		WHERE id = $1`,
		rule.ID, rule.Name, rule.Description, rule.Conditions, rule.Actions, rule.Priority, rule.Enabled,
	)
	return err
}

// Delete removes a workflow rule
func (r *WorkflowRuleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM workflow_rules WHERE id = $1`, id)
	return err
}

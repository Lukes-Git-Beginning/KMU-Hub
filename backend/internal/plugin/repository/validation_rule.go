package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ValidationRuleRepository handles validation rule persistence
type ValidationRuleRepository struct {
	pool *pgxpool.Pool
}

// NewValidationRuleRepository creates a new validation rule repository
func NewValidationRuleRepository(pool *pgxpool.Pool) *ValidationRuleRepository {
	return &ValidationRuleRepository{pool: pool}
}

// Create inserts a new validation rule
func (r *ValidationRuleRepository) Create(ctx context.Context, rule *models.ValidationRule) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO validation_rules (id, tenant_id, installation_id, name, description, entity_type, field_name, rule_type, rule_config, error_message, priority, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		rule.ID, rule.TenantID, rule.InstallationID, rule.Name, rule.Description,
		rule.EntityType, rule.FieldName, rule.RuleType, rule.RuleConfig,
		rule.ErrorMessage, rule.Priority, rule.Enabled, rule.CreatedAt, rule.UpdatedAt,
	)
	return err
}

// GetByID retrieves a validation rule by ID
func (r *ValidationRuleRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.ValidationRule, error) {
	rule := &models.ValidationRule{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, installation_id, name, description, entity_type, field_name, rule_type, rule_config, error_message, priority, enabled, created_at, updated_at
		FROM validation_rules WHERE id = $1`, id,
	).Scan(&rule.ID, &rule.TenantID, &rule.InstallationID, &rule.Name, &rule.Description,
		&rule.EntityType, &rule.FieldName, &rule.RuleType, &rule.RuleConfig,
		&rule.ErrorMessage, &rule.Priority, &rule.Enabled, &rule.CreatedAt, &rule.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return rule, err
}

// List retrieves validation rules for a tenant with optional entity type filter
func (r *ValidationRuleRepository) List(ctx context.Context, tenantID uuid.UUID, entityType string) ([]*models.ValidationRule, error) {
	query := `
		SELECT id, tenant_id, installation_id, name, description, entity_type, field_name, rule_type, rule_config, error_message, priority, enabled, created_at, updated_at
		FROM validation_rules WHERE tenant_id = $1`
	args := []any{tenantID}

	if entityType != "" {
		query += ` AND entity_type = $2`
		args = append(args, entityType)
	}
	query += ` ORDER BY priority ASC, name ASC`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*models.ValidationRule
	for rows.Next() {
		rule := &models.ValidationRule{}
		if scanErr := rows.Scan(&rule.ID, &rule.TenantID, &rule.InstallationID, &rule.Name, &rule.Description,
			&rule.EntityType, &rule.FieldName, &rule.RuleType, &rule.RuleConfig,
			&rule.ErrorMessage, &rule.Priority, &rule.Enabled, &rule.CreatedAt, &rule.UpdatedAt,
		); scanErr != nil {
			return nil, scanErr
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

// ListEnabled retrieves only enabled validation rules for a tenant and entity type
func (r *ValidationRuleRepository) ListEnabled(ctx context.Context, tenantID uuid.UUID, entityType string) ([]*models.ValidationRule, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, installation_id, name, description, entity_type, field_name, rule_type, rule_config, error_message, priority, enabled, created_at, updated_at
		FROM validation_rules
		WHERE tenant_id = $1 AND entity_type = $2 AND enabled = true
		ORDER BY priority ASC`, tenantID, entityType,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*models.ValidationRule
	for rows.Next() {
		rule := &models.ValidationRule{}
		if scanErr := rows.Scan(&rule.ID, &rule.TenantID, &rule.InstallationID, &rule.Name, &rule.Description,
			&rule.EntityType, &rule.FieldName, &rule.RuleType, &rule.RuleConfig,
			&rule.ErrorMessage, &rule.Priority, &rule.Enabled, &rule.CreatedAt, &rule.UpdatedAt,
		); scanErr != nil {
			return nil, scanErr
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

// Update modifies a validation rule
func (r *ValidationRuleRepository) Update(ctx context.Context, rule *models.ValidationRule) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE validation_rules
		SET name = $2, description = $3, rule_config = $4, error_message = $5, priority = $6, enabled = $7, updated_at = NOW()
		WHERE id = $1`,
		rule.ID, rule.Name, rule.Description, rule.RuleConfig, rule.ErrorMessage, rule.Priority, rule.Enabled,
	)
	return err
}

// Delete removes a validation rule
func (r *ValidationRuleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM validation_rules WHERE id = $1`, id)
	return err
}

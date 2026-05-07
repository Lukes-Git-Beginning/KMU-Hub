package routing

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL routing rule repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, rule *models.RoutingRule) error {
	query := `
		INSERT INTO routing_rules (id, tenant_id, name, channel, conditions, actions, priority,
			is_active, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())`

	_, err := r.pool.Exec(ctx, query,
		rule.ID, rule.TenantID, rule.Name, rule.Channel, rule.Conditions, rule.Actions,
		rule.Priority, rule.IsActive, rule.CreatedBy,
	)
	return err
}

func (r *PostgresRepository) Update(ctx context.Context, rule *models.RoutingRule) error {
	query := `
		UPDATE routing_rules SET
			name = $3, channel = $4, conditions = $5, actions = $6,
			priority = $7, is_active = $8, updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2`

	tag, err := r.pool.Exec(ctx, query,
		rule.ID, rule.TenantID, rule.Name, rule.Channel, rule.Conditions, rule.Actions,
		rule.Priority, rule.IsActive,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRuleNotFound
	}
	return nil
}

func (r *PostgresRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM routing_rules WHERE id = $1 AND tenant_id = $2", id, tenantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRuleNotFound
	}
	return nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.RoutingRule, error) {
	query := `
		SELECT id, tenant_id, name, channel, conditions, actions, priority,
			is_active, created_by, created_at, updated_at
		FROM routing_rules WHERE id = $1 AND tenant_id = $2`

	rule := &models.RoutingRule{}
	err := r.pool.QueryRow(ctx, query, id, tenantID).Scan(
		&rule.ID, &rule.TenantID, &rule.Name, &rule.Channel, &rule.Conditions, &rule.Actions,
		&rule.Priority, &rule.IsActive, &rule.CreatedBy, &rule.CreatedAt, &rule.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrRuleNotFound
	}
	return rule, err
}

func (r *PostgresRepository) ListActive(ctx context.Context, tenantID uuid.UUID, channel *string) ([]*models.RoutingRule, error) {
	where := "WHERE tenant_id = $1 AND is_active = true"
	args := []interface{}{tenantID}
	argIdx := 2

	if channel != nil {
		where += fmt.Sprintf(" AND (channel IS NULL OR channel = $%d)", argIdx)
		args = append(args, *channel)
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, name, channel, conditions, actions, priority,
			is_active, created_by, created_at, updated_at
		FROM routing_rules %s
		ORDER BY priority ASC`, where)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*models.RoutingRule
	for rows.Next() {
		rule := &models.RoutingRule{}
		if err := rows.Scan(
			&rule.ID, &rule.TenantID, &rule.Name, &rule.Channel, &rule.Conditions, &rule.Actions,
			&rule.Priority, &rule.IsActive, &rule.CreatedBy, &rule.CreatedAt, &rule.UpdatedAt,
		); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}

	return rules, rows.Err()
}

func (r *PostgresRepository) ListAll(ctx context.Context, tenantID uuid.UUID) ([]*models.RoutingRule, error) {
	query := `
		SELECT id, tenant_id, name, channel, conditions, actions, priority,
			is_active, created_by, created_at, updated_at
		FROM routing_rules
		WHERE tenant_id = $1
		ORDER BY priority ASC, created_at ASC`

	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*models.RoutingRule
	for rows.Next() {
		rule := &models.RoutingRule{}
		if err := rows.Scan(
			&rule.ID, &rule.TenantID, &rule.Name, &rule.Channel, &rule.Conditions, &rule.Actions,
			&rule.Priority, &rule.IsActive, &rule.CreatedBy, &rule.CreatedAt, &rule.UpdatedAt,
		); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}

	return rules, rows.Err()
}

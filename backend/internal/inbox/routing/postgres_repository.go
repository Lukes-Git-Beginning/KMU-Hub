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
		INSERT INTO routing_rules (id, name, channel, conditions, actions, priority,
			is_active, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())`

	_, err := r.pool.Exec(ctx, query,
		rule.ID, rule.Name, rule.Channel, rule.Conditions, rule.Actions,
		rule.Priority, rule.IsActive, rule.CreatedBy,
	)
	return err
}

func (r *PostgresRepository) Update(ctx context.Context, rule *models.RoutingRule) error {
	query := `
		UPDATE routing_rules SET
			name = $2, channel = $3, conditions = $4, actions = $5,
			priority = $6, is_active = $7, updated_at = NOW()
		WHERE id = $1`

	tag, err := r.pool.Exec(ctx, query,
		rule.ID, rule.Name, rule.Channel, rule.Conditions, rule.Actions,
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

func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM routing_rules WHERE id = $1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRuleNotFound
	}
	return nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.RoutingRule, error) {
	query := `
		SELECT id, name, channel, conditions, actions, priority,
			is_active, created_by, created_at, updated_at
		FROM routing_rules WHERE id = $1`

	rule := &models.RoutingRule{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&rule.ID, &rule.Name, &rule.Channel, &rule.Conditions, &rule.Actions,
		&rule.Priority, &rule.IsActive, &rule.CreatedBy, &rule.CreatedAt, &rule.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrRuleNotFound
	}
	return rule, err
}

func (r *PostgresRepository) ListActive(ctx context.Context, channel *string) ([]*models.RoutingRule, error) {
	where := "WHERE is_active = true"
	args := []interface{}{}
	argIdx := 1

	if channel != nil {
		where += fmt.Sprintf(" AND (channel IS NULL OR channel = $%d)", argIdx)
		args = append(args, *channel)
	}

	query := fmt.Sprintf(`
		SELECT id, name, channel, conditions, actions, priority,
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
			&rule.ID, &rule.Name, &rule.Channel, &rule.Conditions, &rule.Actions,
			&rule.Priority, &rule.IsActive, &rule.CreatedBy, &rule.CreatedAt, &rule.UpdatedAt,
		); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}

	return rules, rows.Err()
}

func (r *PostgresRepository) ListAll(ctx context.Context) ([]*models.RoutingRule, error) {
	query := `
		SELECT id, name, channel, conditions, actions, priority,
			is_active, created_by, created_at, updated_at
		FROM routing_rules
		ORDER BY priority ASC, created_at ASC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*models.RoutingRule
	for rows.Next() {
		rule := &models.RoutingRule{}
		if err := rows.Scan(
			&rule.ID, &rule.Name, &rule.Channel, &rule.Conditions, &rule.Actions,
			&rule.Priority, &rule.IsActive, &rule.CreatedBy, &rule.CreatedAt, &rule.UpdatedAt,
		); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}

	return rules, rows.Err()
}

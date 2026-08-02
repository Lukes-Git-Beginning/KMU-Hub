package rule

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

const ruleColumns = `id, tenant_id, name, field, op, value, action_type, action_target, created_at, updated_at`

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL email rule repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func scanRule(row pgx.Row) (*models.EmailRule, error) {
	var r models.EmailRule
	if err := row.Scan(&r.ID, &r.TenantID, &r.Name, &r.Field, &r.Op, &r.Value,
		&r.ActionType, &r.ActionTarget, &r.CreatedAt, &r.UpdatedAt); err != nil {
		return nil, err
	}
	return &r, nil
}

func (p *PostgresRepository) Create(ctx context.Context, r *models.EmailRule) error {
	_, err := p.pool.Exec(ctx,
		`INSERT INTO email_rules (`+ruleColumns+`)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		r.ID, r.TenantID, r.Name, r.Field, r.Op, r.Value, r.ActionType, r.ActionTarget,
		r.CreatedAt, r.UpdatedAt,
	)
	return err
}

func (p *PostgresRepository) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*models.EmailRule, error) {
	r, err := scanRule(p.pool.QueryRow(ctx,
		`SELECT `+ruleColumns+` FROM email_rules WHERE id = $1 AND tenant_id = $2`, id, tenantID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRuleNotFound
		}
		return nil, err
	}
	return r, nil
}

func (p *PostgresRepository) List(ctx context.Context, tenantID uuid.UUID) ([]*models.EmailRule, error) {
	rows, err := p.pool.Query(ctx,
		`SELECT `+ruleColumns+` FROM email_rules WHERE tenant_id = $1 ORDER BY created_at, id`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rules := make([]*models.EmailRule, 0)
	for rows.Next() {
		r, err := scanRule(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}
	return rules, rows.Err()
}

func (p *PostgresRepository) Update(ctx context.Context, r *models.EmailRule) error {
	r.UpdatedAt = time.Now().UTC()
	tag, err := p.pool.Exec(ctx,
		`UPDATE email_rules
		    SET name = $3, field = $4, op = $5, value = $6,
		        action_type = $7, action_target = $8, updated_at = $9
		  WHERE id = $1 AND tenant_id = $2`,
		r.ID, r.TenantID, r.Name, r.Field, r.Op, r.Value, r.ActionType, r.ActionTarget, r.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRuleNotFound
	}
	return nil
}

func (p *PostgresRepository) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	tag, err := p.pool.Exec(ctx, `DELETE FROM email_rules WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRuleNotFound
	}
	return nil
}

// FolderBelongsToTenant checks the move target against email_folders.tenant_id,
// which is NOT NULL since the Option-B retrofit -- no join through the account
// needed.
func (p *PostgresRepository) FolderBelongsToTenant(ctx context.Context, folderID, tenantID uuid.UUID) (bool, error) {
	var exists bool
	err := p.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM email_folders WHERE id = $1 AND tenant_id = $2)`,
		folderID, tenantID).Scan(&exists)
	return exists, err
}

// LabelBelongsToTenant checks the label action's target against
// email_labels.tenant_id.
func (p *PostgresRepository) LabelBelongsToTenant(ctx context.Context, labelID, tenantID uuid.UUID) (bool, error) {
	var exists bool
	err := p.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM email_labels WHERE id = $1 AND tenant_id = $2)`,
		labelID, tenantID).Scan(&exists)
	return exists, err
}

func (p *PostgresRepository) ListApplyCandidates(ctx context.Context, tenantID uuid.UUID, limit int) ([]*models.EmailRuleCandidate, error) {
	rows, err := p.pool.Query(ctx,
		`SELECT m.id, COALESCE(m.from_name, ''), m.from_email, COALESCE(m.subject, ''),
		        m.folder_id, m.label_ids
		   FROM email_messages m
		   LEFT JOIN email_folders f ON f.id = m.folder_id
		  WHERE m.tenant_id = $1
		    AND COALESCE(f.folder_type, '') <> 'trash'
		  ORDER BY m.date DESC
		  LIMIT $2`, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]*models.EmailRuleCandidate, 0)
	for rows.Next() {
		var c models.EmailRuleCandidate
		if err := rows.Scan(&c.ID, &c.FromName, &c.FromEmail, &c.Subject, &c.FolderID, &c.LabelIDs); err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

func (p *PostgresRepository) ApplyToMessage(ctx context.Context, tenantID, messageID, folderID uuid.UUID, labelIDs []uuid.UUID) error {
	if labelIDs == nil {
		labelIDs = []uuid.UUID{}
	}
	_, err := p.pool.Exec(ctx,
		`UPDATE email_messages
		    SET folder_id = $3, label_ids = $4, updated_at = NOW()
		  WHERE id = $1 AND tenant_id = $2`,
		messageID, tenantID, folderID, labelIDs,
	)
	return err
}

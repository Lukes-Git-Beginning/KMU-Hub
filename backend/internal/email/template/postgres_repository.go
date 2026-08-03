package template

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL template repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// A row is visible if it is shared, owned by the caller, or the caller is an
// admin. tenant_id is additionally enforced by email_templates' own RLS
// policy (migration 000287); the explicit tenant_id predicate in both
// queries below is defense in depth for callers running under
// sysctx.With(), same reasoning as user_roles.

func (r *PostgresRepository) Create(ctx context.Context, tpl *models.EmailTemplate) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO email_templates (id, tenant_id, owner_id, visibility, name, subject, body_html, body_text, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		tpl.ID, tpl.TenantID, tpl.OwnerID, tpl.Visibility, tpl.Name, tpl.Subject, tpl.BodyHTML, tpl.BodyText,
		tpl.CreatedAt, tpl.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetByID(ctx context.Context, id, tenantID, userID uuid.UUID, isAdmin bool) (*models.EmailTemplate, error) {
	var tpl models.EmailTemplate
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, owner_id, visibility, name, subject, body_html, body_text, created_at, updated_at
		 FROM email_templates
		 WHERE id = $1 AND tenant_id = $2 AND (visibility = 'shared' OR owner_id = $3 OR $4 = true)`,
		id, tenantID, userID, isAdmin,
	).Scan(&tpl.ID, &tpl.TenantID, &tpl.OwnerID, &tpl.Visibility, &tpl.Name, &tpl.Subject, &tpl.BodyHTML, &tpl.BodyText,
		&tpl.CreatedAt, &tpl.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTemplateNotFound
		}
		return nil, err
	}
	return &tpl, nil
}

func (r *PostgresRepository) ListVisible(ctx context.Context, tenantID, userID uuid.UUID, isAdmin bool) ([]*models.EmailTemplate, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, owner_id, visibility, name, subject, body_html, body_text, created_at, updated_at
		 FROM email_templates
		 WHERE tenant_id = $1 AND (visibility = 'shared' OR owner_id = $2 OR $3 = true)
		 ORDER BY name ASC`,
		tenantID, userID, isAdmin,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []*models.EmailTemplate
	for rows.Next() {
		var tpl models.EmailTemplate
		if err := rows.Scan(&tpl.ID, &tpl.TenantID, &tpl.OwnerID, &tpl.Visibility, &tpl.Name, &tpl.Subject, &tpl.BodyHTML, &tpl.BodyText,
			&tpl.CreatedAt, &tpl.UpdatedAt); err != nil {
			return nil, err
		}
		templates = append(templates, &tpl)
	}
	return templates, rows.Err()
}

func (r *PostgresRepository) Update(ctx context.Context, tpl *models.EmailTemplate) error {
	tpl.UpdatedAt = time.Now().UTC()
	_, err := r.pool.Exec(ctx,
		`UPDATE email_templates SET owner_id = $2, visibility = $3, name = $4, subject = $5, body_html = $6, body_text = $7, updated_at = $8
		 WHERE id = $1 AND tenant_id = $9`,
		tpl.ID, tpl.OwnerID, tpl.Visibility, tpl.Name, tpl.Subject, tpl.BodyHTML, tpl.BodyText, tpl.UpdatedAt, tpl.TenantID,
	)
	return err
}

func (r *PostgresRepository) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM email_templates WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	return err
}

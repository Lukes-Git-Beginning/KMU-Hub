package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// IndustryTemplateRepository handles industry template persistence
type IndustryTemplateRepository struct {
	pool *pgxpool.Pool
}

// NewIndustryTemplateRepository creates a new industry template repository
func NewIndustryTemplateRepository(pool *pgxpool.Pool) *IndustryTemplateRepository {
	return &IndustryTemplateRepository{pool: pool}
}

// Create inserts a new industry template
func (r *IndustryTemplateRepository) Create(ctx context.Context, tmpl *models.IndustryTemplate) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO industry_templates (id, slug, name, description, industry, icon, custom_fields, validation_rules, workflow_rules, default_settings, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (slug) DO UPDATE SET
			name = EXCLUDED.name, description = EXCLUDED.description, industry = EXCLUDED.industry,
			icon = EXCLUDED.icon, custom_fields = EXCLUDED.custom_fields,
			validation_rules = EXCLUDED.validation_rules, workflow_rules = EXCLUDED.workflow_rules,
			default_settings = EXCLUDED.default_settings, updated_at = NOW()`,
		tmpl.ID, tmpl.Slug, tmpl.Name, tmpl.Description, tmpl.Industry, tmpl.Icon,
		tmpl.CustomFields, tmpl.ValidationRules, tmpl.WorkflowRules, tmpl.DefaultSettings,
		tmpl.CreatedAt, tmpl.UpdatedAt,
	)
	return err
}

// GetByID retrieves a template by ID
func (r *IndustryTemplateRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.IndustryTemplate, error) {
	tmpl := &models.IndustryTemplate{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, slug, name, description, industry, icon, custom_fields, validation_rules, workflow_rules, default_settings, created_at, updated_at
		FROM industry_templates WHERE id = $1`, id,
	).Scan(&tmpl.ID, &tmpl.Slug, &tmpl.Name, &tmpl.Description, &tmpl.Industry, &tmpl.Icon,
		&tmpl.CustomFields, &tmpl.ValidationRules, &tmpl.WorkflowRules, &tmpl.DefaultSettings,
		&tmpl.CreatedAt, &tmpl.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return tmpl, err
}

// GetBySlug retrieves a template by slug
func (r *IndustryTemplateRepository) GetBySlug(ctx context.Context, slug string) (*models.IndustryTemplate, error) {
	tmpl := &models.IndustryTemplate{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, slug, name, description, industry, icon, custom_fields, validation_rules, workflow_rules, default_settings, created_at, updated_at
		FROM industry_templates WHERE slug = $1`, slug,
	).Scan(&tmpl.ID, &tmpl.Slug, &tmpl.Name, &tmpl.Description, &tmpl.Industry, &tmpl.Icon,
		&tmpl.CustomFields, &tmpl.ValidationRules, &tmpl.WorkflowRules, &tmpl.DefaultSettings,
		&tmpl.CreatedAt, &tmpl.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return tmpl, err
}

// List retrieves all templates with optional industry filter
func (r *IndustryTemplateRepository) List(ctx context.Context, industry string) ([]*models.IndustryTemplate, error) {
	query := `SELECT id, slug, name, description, industry, icon, custom_fields, validation_rules, workflow_rules, default_settings, created_at, updated_at FROM industry_templates`
	args := []any{}

	if industry != "" {
		query += ` WHERE industry = $1`
		args = append(args, industry)
	}
	query += ` ORDER BY industry, name ASC`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []*models.IndustryTemplate
	for rows.Next() {
		tmpl := &models.IndustryTemplate{}
		if scanErr := rows.Scan(&tmpl.ID, &tmpl.Slug, &tmpl.Name, &tmpl.Description, &tmpl.Industry, &tmpl.Icon,
			&tmpl.CustomFields, &tmpl.ValidationRules, &tmpl.WorkflowRules, &tmpl.DefaultSettings,
			&tmpl.CreatedAt, &tmpl.UpdatedAt,
		); scanErr != nil {
			return nil, scanErr
		}
		templates = append(templates, tmpl)
	}
	return templates, nil
}

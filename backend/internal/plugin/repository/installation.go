package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// InstallationRepository handles plugin installation persistence
type InstallationRepository struct {
	pool *pgxpool.Pool
}

// NewInstallationRepository creates a new installation repository
func NewInstallationRepository(pool *pgxpool.Pool) *InstallationRepository {
	return &InstallationRepository{pool: pool}
}

// Create inserts a new plugin installation
func (r *InstallationRepository) Create(ctx context.Context, inst *models.PluginInstallation) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO plugin_installations (id, tenant_id, manifest_id, status, settings, error_message, installed_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		inst.ID, inst.TenantID, inst.ManifestID, inst.Status, inst.Settings,
		inst.ErrorMessage, inst.InstalledBy, inst.CreatedAt, inst.UpdatedAt,
	)
	return err
}

// GetByID retrieves an installation by ID
func (r *InstallationRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.PluginInstallation, error) {
	inst := &models.PluginInstallation{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, manifest_id, status, settings, error_message, installed_by, created_at, updated_at
		FROM plugin_installations WHERE id = $1`, id,
	).Scan(&inst.ID, &inst.TenantID, &inst.ManifestID, &inst.Status, &inst.Settings,
		&inst.ErrorMessage, &inst.InstalledBy, &inst.CreatedAt, &inst.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return inst, err
}

// GetByTenantAndManifest retrieves an installation by tenant and manifest
func (r *InstallationRepository) GetByTenantAndManifest(ctx context.Context, tenantID, manifestID uuid.UUID) (*models.PluginInstallation, error) {
	inst := &models.PluginInstallation{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, manifest_id, status, settings, error_message, installed_by, created_at, updated_at
		FROM plugin_installations WHERE tenant_id = $1 AND manifest_id = $2`, tenantID, manifestID,
	).Scan(&inst.ID, &inst.TenantID, &inst.ManifestID, &inst.Status, &inst.Settings,
		&inst.ErrorMessage, &inst.InstalledBy, &inst.CreatedAt, &inst.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return inst, err
}

// List retrieves installations for a tenant with optional status filter
func (r *InstallationRepository) List(ctx context.Context, tenantID uuid.UUID, status string) ([]*models.PluginInstallationWithManifest, error) {
	query := `
		SELECT i.id, i.tenant_id, i.manifest_id, i.status, i.settings, i.error_message, i.installed_by, i.created_at, i.updated_at,
		       m.slug, m.name, m.version, m.plugin_type, m.permissions, m.settings_schema
		FROM plugin_installations i
		JOIN plugin_manifests m ON m.id = i.manifest_id
		WHERE i.tenant_id = $1`
	args := []any{tenantID}

	if status != "" {
		query += ` AND i.status = $2`
		args = append(args, status)
	}
	query += ` ORDER BY m.name ASC`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var installations []*models.PluginInstallationWithManifest
	for rows.Next() {
		inst := &models.PluginInstallationWithManifest{}
		if scanErr := rows.Scan(
			&inst.ID, &inst.TenantID, &inst.ManifestID, &inst.Status, &inst.Settings,
			&inst.ErrorMessage, &inst.InstalledBy, &inst.CreatedAt, &inst.UpdatedAt,
			&inst.ManifestSlug, &inst.ManifestName, &inst.ManifestVersion,
			&inst.PluginType, &inst.Permissions, &inst.SettingsSchema,
		); scanErr != nil {
			return nil, scanErr
		}
		installations = append(installations, inst)
	}
	return installations, nil
}

// UpdateStatus updates the status of an installation
func (r *InstallationRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status models.InstallationStatus, errorMsg *string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE plugin_installations SET status = $2, error_message = $3, updated_at = NOW()
		WHERE id = $1`, id, status, errorMsg,
	)
	return err
}

// UpdateSettings updates the settings of an installation
func (r *InstallationRepository) UpdateSettings(ctx context.Context, id uuid.UUID, settings []byte) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE plugin_installations SET settings = $2, updated_at = NOW()
		WHERE id = $1`, id, settings,
	)
	return err
}

// ListActiveByHook retrieves active installations that registered for a specific hook
func (r *InstallationRepository) ListActiveByHook(ctx context.Context, tenantID uuid.UUID, hookType, module, entityType string) ([]*models.PluginInstallation, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT i.id, i.tenant_id, i.manifest_id, i.status, i.settings, i.error_message, i.installed_by, i.created_at, i.updated_at
		FROM plugin_installations i
		JOIN plugin_manifests m ON m.id = i.manifest_id
		WHERE i.tenant_id = $1 AND i.status = 'active'
		  AND EXISTS (
		    SELECT 1 FROM jsonb_array_elements(m.hook_registrations) AS hr
		    WHERE hr->>'hook_type' = $2 AND hr->>'module' = $3 AND hr->>'entity_type' = $4
		  )
		ORDER BY (
		  SELECT (hr->>'priority')::int FROM jsonb_array_elements(m.hook_registrations) AS hr
		  WHERE hr->>'hook_type' = $2 AND hr->>'module' = $3 AND hr->>'entity_type' = $4
		  LIMIT 1
		) ASC`, tenantID, hookType, module, entityType,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var installations []*models.PluginInstallation
	for rows.Next() {
		inst := &models.PluginInstallation{}
		if scanErr := rows.Scan(&inst.ID, &inst.TenantID, &inst.ManifestID, &inst.Status, &inst.Settings,
			&inst.ErrorMessage, &inst.InstalledBy, &inst.CreatedAt, &inst.UpdatedAt,
		); scanErr != nil {
			return nil, scanErr
		}
		installations = append(installations, inst)
	}
	return installations, nil
}

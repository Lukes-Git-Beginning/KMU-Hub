package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// PermissionRepository handles plugin permission persistence
type PermissionRepository struct {
	pool *pgxpool.Pool
}

// NewPermissionRepository creates a new permission repository
func NewPermissionRepository(pool *pgxpool.Pool) *PermissionRepository {
	return &PermissionRepository{pool: pool}
}

// Grant inserts a new permission grant
func (r *PermissionRepository) Grant(ctx context.Context, perm *models.PluginPermission) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO plugin_permissions (id, installation_id, permission, granted_by, granted_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (installation_id, permission) DO NOTHING`,
		perm.ID, perm.InstallationID, perm.Permission, perm.GrantedBy, perm.GrantedAt,
	)
	return err
}

// ListByInstallation retrieves all granted permissions for an installation
func (r *PermissionRepository) ListByInstallation(ctx context.Context, installationID uuid.UUID) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT permission FROM plugin_permissions
		WHERE installation_id = $1 ORDER BY permission`, installationID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	permissions := make([]string, 0)
	for rows.Next() {
		var perm string
		if scanErr := rows.Scan(&perm); scanErr != nil {
			return nil, scanErr
		}
		permissions = append(permissions, perm)
	}
	return permissions, nil
}

// HasPermission checks if an installation has a specific permission
func (r *PermissionRepository) HasPermission(ctx context.Context, installationID uuid.UUID, permission string) (bool, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM plugin_permissions
		WHERE installation_id = $1 AND permission = $2`, installationID, permission,
	).Scan(&count)
	return count > 0, err
}

// RevokeAll revokes all permissions for an installation
func (r *PermissionRepository) RevokeAll(ctx context.Context, installationID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM plugin_permissions WHERE installation_id = $1`, installationID)
	return err
}

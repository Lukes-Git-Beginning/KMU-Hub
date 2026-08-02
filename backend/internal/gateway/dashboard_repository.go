package gateway

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ErrDashboardNotFound is returned when no dashboard layout is found.
var ErrDashboardNotFound = errors.New("dashboard layout not found")

// DashboardRepository defines the interface for dashboard layout persistence.
type DashboardRepository interface {
	// GetDefaultLayout returns the tenant's default dashboard layout for a role.
	GetDefaultLayout(ctx context.Context, tenantID uuid.UUID, role string) (*models.DashboardDefault, error)
	// UpsertDefaultLayout creates or updates the tenant's default layout for a role.
	UpsertDefaultLayout(ctx context.Context, tenantID uuid.UUID, def *models.DashboardDefault) (*models.DashboardDefault, error)
	// GetUserLayout returns a user's personal layout override.
	GetUserLayout(ctx context.Context, tenantID uuid.UUID, userID string) (*models.UserDashboardLayout, error)
	// UpsertUserLayout creates or updates a user's personal layout.
	UpsertUserLayout(ctx context.Context, layout *models.UserDashboardLayout) (*models.UserDashboardLayout, error)
	// DeleteUserLayout removes a user's personal layout override (reset to defaults).
	DeleteUserLayout(ctx context.Context, tenantID uuid.UUID, userID string) error
}

// PostgresDashboardRepository implements DashboardRepository using PostgreSQL.
type PostgresDashboardRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresDashboardRepository creates a new PostgresDashboardRepository.
func NewPostgresDashboardRepository(pool *pgxpool.Pool) *PostgresDashboardRepository {
	return &PostgresDashboardRepository{pool: pool}
}

// GetDefaultLayout retrieves the tenant's default layout for a given role.
// The tenant is filtered explicitly rather than left to the RLS session GUC,
// so a caller running under a system context cannot read a foreign tenant's
// preset — the same reason GetUserLayout below takes it as a parameter.
func (r *PostgresDashboardRepository) GetDefaultLayout(ctx context.Context, tenantID uuid.UUID, role string) (*models.DashboardDefault, error) {
	var d models.DashboardDefault
	err := r.pool.QueryRow(ctx,
		`SELECT id, role, layout, active_widgets, created_at, updated_at
		 FROM dashboard_defaults
		 WHERE tenant_id = $1 AND role = $2`, tenantID, role,
	).Scan(&d.ID, &d.Role, &d.Layout, &d.ActiveWidgets, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDashboardNotFound
		}
		return nil, err
	}
	return &d, nil
}

// UpsertDefaultLayout inserts or updates the tenant's role default layout.
func (r *PostgresDashboardRepository) UpsertDefaultLayout(ctx context.Context, tenantID uuid.UUID, def *models.DashboardDefault) (*models.DashboardDefault, error) {
	var d models.DashboardDefault
	err := r.pool.QueryRow(ctx,
		`INSERT INTO dashboard_defaults (tenant_id, role, layout, active_widgets, updated_at)
		 VALUES ($1, $2, $3, $4, NOW())
		 ON CONFLICT (tenant_id, role) DO UPDATE
		 SET layout = EXCLUDED.layout,
		     active_widgets = EXCLUDED.active_widgets,
		     updated_at = NOW()
		 RETURNING id, role, layout, active_widgets, created_at, updated_at`,
		tenantID, def.Role, def.Layout, def.ActiveWidgets,
	).Scan(&d.ID, &d.Role, &d.Layout, &d.ActiveWidgets, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// GetUserLayout retrieves a user's personal dashboard layout.
func (r *PostgresDashboardRepository) GetUserLayout(ctx context.Context, tenantID uuid.UUID, userID string) (*models.UserDashboardLayout, error) {
	var l models.UserDashboardLayout
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, layout, active_widgets, created_at, updated_at
		 FROM user_dashboard_layouts
		 WHERE tenant_id = $1 AND user_id = $2`, tenantID, userID,
	).Scan(&l.ID, &l.UserID, &l.Layout, &l.ActiveWidgets, &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDashboardNotFound
		}
		return nil, err
	}
	return &l, nil
}

// UpsertUserLayout inserts or updates a user's personal layout.
func (r *PostgresDashboardRepository) UpsertUserLayout(ctx context.Context, layout *models.UserDashboardLayout) (*models.UserDashboardLayout, error) {
	var l models.UserDashboardLayout
	err := r.pool.QueryRow(ctx,
		`INSERT INTO user_dashboard_layouts (tenant_id, user_id, layout, active_widgets, updated_at)
		 VALUES ($1, $2, $3, $4, NOW())
		 ON CONFLICT ON CONSTRAINT uq_user_dashboard_layouts_user_id DO UPDATE
		 SET layout = EXCLUDED.layout,
		     active_widgets = EXCLUDED.active_widgets,
		     updated_at = NOW()
		 RETURNING id, user_id, layout, active_widgets, created_at, updated_at`,
		layout.TenantID, layout.UserID, layout.Layout, layout.ActiveWidgets,
	).Scan(&l.ID, &l.UserID, &l.Layout, &l.ActiveWidgets, &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// DeleteUserLayout removes a user's personal layout override.
func (r *PostgresDashboardRepository) DeleteUserLayout(ctx context.Context, tenantID uuid.UUID, userID string) error {
	result, err := r.pool.Exec(ctx,
		`DELETE FROM user_dashboard_layouts WHERE tenant_id = $1 AND user_id = $2`, tenantID, userID,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrDashboardNotFound
	}
	return nil
}

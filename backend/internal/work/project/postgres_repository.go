package project

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, project *models.Project) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO projects (id, tenant_id, name, description, project_key, next_task_number, is_template,
		 template_source_id, created_by, created_at, updated_at, archived_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		project.ID, project.TenantID, project.Name, project.Description, project.ProjectKey,
		project.NextTaskNumber, project.IsTemplate, project.TemplateSourceID,
		project.CreatedBy, project.CreatedAt, project.UpdatedAt, project.ArchivedAt,
	)
	return err
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.Project, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, description, project_key, next_task_number, is_template,
		 template_source_id, created_by, created_at, updated_at, archived_at
		 FROM projects WHERE id = $1 AND tenant_id = $2`, id, tenantID,
	)
	return r.scanProject(row)
}

func (r *PostgresRepository) List(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, isAdmin bool, includeArchived bool) ([]models.ProjectWithDetails, error) {
	var query string
	var args []any

	if isAdmin {
		if includeArchived {
			query = `SELECT p.id, p.tenant_id, p.name, p.description, p.project_key, p.next_task_number,
				p.is_template, p.template_source_id, p.created_by, p.created_at, p.updated_at, p.archived_at,
				(SELECT COUNT(*) FROM project_members pm WHERE pm.project_id = p.id) AS member_count,
				(SELECT COUNT(*) FROM tasks t WHERE t.project_id = p.id) AS task_count,
				COALESCE((SELECT u.first_name || ' ' || u.last_name FROM users u WHERE u.id = p.created_by), '') AS owner_name
				FROM projects p
				WHERE p.tenant_id = $1
				ORDER BY p.created_at DESC`
			args = append(args, tenantID)
		} else {
			query = `SELECT p.id, p.tenant_id, p.name, p.description, p.project_key, p.next_task_number,
				p.is_template, p.template_source_id, p.created_by, p.created_at, p.updated_at, p.archived_at,
				(SELECT COUNT(*) FROM project_members pm WHERE pm.project_id = p.id) AS member_count,
				(SELECT COUNT(*) FROM tasks t WHERE t.project_id = p.id) AS task_count,
				COALESCE((SELECT u.first_name || ' ' || u.last_name FROM users u WHERE u.id = p.created_by), '') AS owner_name
				FROM projects p
				WHERE p.tenant_id = $1 AND p.archived_at IS NULL
				ORDER BY p.created_at DESC`
			args = append(args, tenantID)
		}
	} else {
		if includeArchived {
			query = `SELECT p.id, p.tenant_id, p.name, p.description, p.project_key, p.next_task_number,
				p.is_template, p.template_source_id, p.created_by, p.created_at, p.updated_at, p.archived_at,
				(SELECT COUNT(*) FROM project_members pm WHERE pm.project_id = p.id) AS member_count,
				(SELECT COUNT(*) FROM tasks t WHERE t.project_id = p.id) AS task_count,
				COALESCE((SELECT u.first_name || ' ' || u.last_name FROM users u WHERE u.id = p.created_by), '') AS owner_name
				FROM projects p
				JOIN project_members pm ON p.id = pm.project_id AND pm.user_id = $2
				WHERE p.tenant_id = $1
				ORDER BY p.created_at DESC`
			args = append(args, tenantID, userID)
		} else {
			query = `SELECT p.id, p.tenant_id, p.name, p.description, p.project_key, p.next_task_number,
				p.is_template, p.template_source_id, p.created_by, p.created_at, p.updated_at, p.archived_at,
				(SELECT COUNT(*) FROM project_members pm WHERE pm.project_id = p.id) AS member_count,
				(SELECT COUNT(*) FROM tasks t WHERE t.project_id = p.id) AS task_count,
				COALESCE((SELECT u.first_name || ' ' || u.last_name FROM users u WHERE u.id = p.created_by), '') AS owner_name
				FROM projects p
				JOIN project_members pm ON p.id = pm.project_id AND pm.user_id = $2
				WHERE p.tenant_id = $1 AND p.archived_at IS NULL
				ORDER BY p.created_at DESC`
			args = append(args, tenantID, userID)
		}
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []models.ProjectWithDetails
	for rows.Next() {
		var p models.ProjectWithDetails
		scanErr := rows.Scan(
			&p.ID, &p.TenantID, &p.Name, &p.Description, &p.ProjectKey, &p.NextTaskNumber,
			&p.IsTemplate, &p.TemplateSourceID, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt, &p.ArchivedAt,
			&p.MemberCount, &p.TaskCount, &p.OwnerName,
		)
		if scanErr != nil {
			return nil, scanErr
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func (r *PostgresRepository) Update(ctx context.Context, project *models.Project) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE projects SET name = $1, description = $2, updated_at = $3
		 WHERE id = $4 AND tenant_id = $5`,
		project.Name, project.Description, project.UpdatedAt, project.ID, project.TenantID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) Archive(ctx context.Context, projectID uuid.UUID, tenantID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE projects SET archived_at = $1, updated_at = $1 WHERE id = $2 AND tenant_id = $3`,
		time.Now(), projectID, tenantID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) GetProjectKey(ctx context.Context, projectID uuid.UUID, tenantID uuid.UUID) (string, error) {
	var key string
	err := r.pool.QueryRow(ctx,
		`SELECT project_key FROM projects WHERE id = $1 AND tenant_id = $2`, projectID, tenantID,
	).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return key, err
}

func (r *PostgresRepository) KeyExists(ctx context.Context, tenantID uuid.UUID, key string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM projects WHERE project_key = $1 AND tenant_id = $2 AND archived_at IS NULL)`, key, tenantID,
	).Scan(&exists)
	return exists, err
}

// Member management

func (r *PostgresRepository) AddMember(ctx context.Context, projectID, userID uuid.UUID, role string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO project_members (project_id, user_id, role, added_at)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (project_id, user_id) DO NOTHING`,
		projectID, userID, role, time.Now(),
	)
	return err
}

func (r *PostgresRepository) RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM project_members WHERE project_id = $1 AND user_id = $2`,
		projectID, userID,
	)
	return err
}

func (r *PostgresRepository) UpdateMemberRole(ctx context.Context, projectID, userID uuid.UUID, role string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE project_members SET role = $1 WHERE project_id = $2 AND user_id = $3`,
		role, projectID, userID,
	)
	return err
}

func (r *PostgresRepository) GetMember(ctx context.Context, projectID, userID uuid.UUID) (*models.ProjectMember, error) {
	var m models.ProjectMember
	err := r.pool.QueryRow(ctx,
		`SELECT pm.project_id, pm.user_id, pm.role, pm.added_at,
		 COALESCE(u.first_name, '') AS first_name, COALESCE(u.last_name, '') AS last_name
		 FROM project_members pm
		 JOIN users u ON pm.user_id = u.id
		 WHERE pm.project_id = $1 AND pm.user_id = $2`,
		projectID, userID,
	).Scan(&m.ProjectID, &m.UserID, &m.Role, &m.AddedAt, &m.FirstName, &m.LastName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotProjectMember
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *PostgresRepository) ListMembers(ctx context.Context, projectID uuid.UUID) ([]models.ProjectMember, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT pm.project_id, pm.user_id, pm.role, pm.added_at,
		 COALESCE(u.first_name, '') AS first_name, COALESCE(u.last_name, '') AS last_name
		 FROM project_members pm
		 JOIN users u ON pm.user_id = u.id
		 WHERE pm.project_id = $1
		 ORDER BY pm.added_at ASC`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []models.ProjectMember
	for rows.Next() {
		var m models.ProjectMember
		if scanErr := rows.Scan(&m.ProjectID, &m.UserID, &m.Role, &m.AddedAt, &m.FirstName, &m.LastName); scanErr != nil {
			return nil, scanErr
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

func (r *PostgresRepository) IsMember(ctx context.Context, projectID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM project_members WHERE project_id = $1 AND user_id = $2)`,
		projectID, userID,
	).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) CountOwners(ctx context.Context, projectID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM project_members WHERE project_id = $1 AND role = 'owner'`,
		projectID,
	).Scan(&count)
	return count, err
}

// Template operations

func (r *PostgresRepository) SaveAsTemplate(ctx context.Context, tenantID uuid.UUID, sourceID uuid.UUID, newName string, newKey string, createdBy uuid.UUID) (*models.Project, error) {
	now := time.Now()
	id := uuid.New()

	_, err := r.pool.Exec(ctx,
		`INSERT INTO projects (id, tenant_id, name, description, project_key, next_task_number, is_template,
		 template_source_id, created_by, created_at, updated_at)
		 SELECT $1, tenant_id, $2, description, $3, 1, true, $4, $5, $6, $6
		 FROM projects WHERE id = $4 AND tenant_id = $7`,
		id, newName, newKey, sourceID, createdBy, now, tenantID,
	)
	if err != nil {
		return nil, err
	}

	return r.GetByID(ctx, id, tenantID)
}

func (r *PostgresRepository) GetForTemplate(ctx context.Context, projectID uuid.UUID, tenantID uuid.UUID) (*models.Project, []models.ProjectStatus, error) {
	project, err := r.GetByID(ctx, projectID, tenantID)
	if err != nil {
		return nil, nil, err
	}

	rows, queryErr := r.pool.Query(ctx,
		`SELECT id, project_id, name, color, sort_order, is_default, is_closed, created_at
		 FROM project_statuses WHERE project_id = $1 ORDER BY sort_order ASC`,
		projectID,
	)
	if queryErr != nil {
		return nil, nil, queryErr
	}
	defer rows.Close()

	var statuses []models.ProjectStatus
	for rows.Next() {
		var s models.ProjectStatus
		if scanErr := rows.Scan(&s.ID, &s.ProjectID, &s.Name, &s.Color, &s.SortOrder, &s.IsDefault, &s.IsClosed, &s.CreatedAt); scanErr != nil {
			return nil, nil, scanErr
		}
		statuses = append(statuses, s)
	}
	if rowErr := rows.Err(); rowErr != nil {
		return nil, nil, rowErr
	}

	return project, statuses, nil
}

// User preferences

func (r *PostgresRepository) GetUserPreference(ctx context.Context, userID, projectID uuid.UUID) (*models.UserProjectPreference, error) {
	var pref models.UserProjectPreference
	err := r.pool.QueryRow(ctx,
		`SELECT tenant_id, user_id, project_id, view_type, list_group_by, list_sort_by, list_sort_desc, updated_at
		 FROM user_project_preferences WHERE user_id = $1 AND project_id = $2`,
		userID, projectID,
	).Scan(&pref.TenantID, &pref.UserID, &pref.ProjectID, &pref.ViewType, &pref.ListGroupBy, &pref.ListSortBy, &pref.ListSortDesc, &pref.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // No preference set, return nil (caller uses defaults)
	}
	if err != nil {
		return nil, err
	}
	return &pref, nil
}

func (r *PostgresRepository) SetUserPreference(ctx context.Context, pref *models.UserProjectPreference) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO user_project_preferences (tenant_id, user_id, project_id, view_type, list_group_by, list_sort_by, list_sort_desc, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (user_id, project_id) DO UPDATE SET
		 tenant_id = $1, view_type = $4, list_group_by = $5, list_sort_by = $6, list_sort_desc = $7, updated_at = $8`,
		pref.TenantID, pref.UserID, pref.ProjectID, pref.ViewType, pref.ListGroupBy, pref.ListSortBy, pref.ListSortDesc, pref.UpdatedAt,
	)
	return err
}

// Helper to scan a single row into Project
func (r *PostgresRepository) scanProject(row pgx.Row) (*models.Project, error) {
	var p models.Project
	err := row.Scan(
		&p.ID, &p.TenantID, &p.Name, &p.Description, &p.ProjectKey, &p.NextTaskNumber,
		&p.IsTemplate, &p.TemplateSourceID, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt, &p.ArchivedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

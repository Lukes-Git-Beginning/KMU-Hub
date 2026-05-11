package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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

func (r *PostgresRepository) Create(ctx context.Context, task *models.Task) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO tasks (id, tenant_id, project_id, task_number, title, description, status_id, priority,
		  assignee_id, parent_task_id, depth, sort_order, due_date, created_by, created_at, updated_at, completed_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`,
		task.ID, task.TenantID, task.ProjectID, task.TaskNumber, task.Title, task.Description,
		task.StatusID, task.Priority, task.AssigneeID, task.ParentTaskID,
		task.Depth, task.SortOrder, task.DueDate, task.CreatedBy,
		task.CreatedAt, task.UpdatedAt, task.CompletedAt,
	)
	return err
}

func (r *PostgresRepository) GetNextTaskNumber(ctx context.Context, projectID uuid.UUID) (int, error) {
	var num int
	err := r.pool.QueryRow(ctx,
		`UPDATE projects SET next_task_number = next_task_number + 1 WHERE id = $1
		 RETURNING next_task_number - 1`,
		projectID,
	).Scan(&num)
	if err != nil {
		return 0, fmt.Errorf("get next task number: %w", err)
	}
	return num, nil
}

const getByIDQuery = `
SELECT t.id, t.project_id, t.task_number, t.title, t.description, t.status_id, t.priority,
       t.assignee_id, t.parent_task_id, t.depth, t.sort_order, t.due_date,
       t.created_by, t.created_at, t.updated_at, t.completed_at,
       ps.name AS status_name, ps.color AS status_color,
       u.first_name || ' ' || u.last_name AS assignee_name,
       p.project_key,
       (SELECT COUNT(*) FROM tasks WHERE parent_task_id = t.id) AS subtask_count,
       EXISTS(SELECT 1 FROM task_dependencies WHERE target_task_id = t.id AND dependency_type = 'blocks') AS has_blocked_deps
FROM tasks t
LEFT JOIN project_statuses ps ON t.status_id = ps.id
LEFT JOIN users u ON t.assignee_id = u.id
LEFT JOIN projects p ON t.project_id = p.id
WHERE t.id = $1 AND t.tenant_id = $2
`

func (r *PostgresRepository) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*models.TaskWithRelations, error) {
	var tw models.TaskWithRelations
	err := r.pool.QueryRow(ctx, getByIDQuery, id, tenantID).Scan(
		&tw.ID, &tw.ProjectID, &tw.TaskNumber, &tw.Title, &tw.Description,
		&tw.StatusID, &tw.Priority, &tw.AssigneeID, &tw.ParentTaskID,
		&tw.Depth, &tw.SortOrder, &tw.DueDate, &tw.CreatedBy,
		&tw.CreatedAt, &tw.UpdatedAt, &tw.CompletedAt,
		&tw.StatusName, &tw.StatusColor, &tw.AssigneeName, &tw.ProjectKey,
		&tw.SubtaskCount, &tw.HasBlockedDeps,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &tw, nil
}

// validSortColumns maps allowed sort field names to SQL expressions
var validSortColumns = map[string]string{
	"created_at":  "t.created_at",
	"due_date":    "t.due_date",
	"priority":    "t.priority",
	"title":       "t.title",
	"task_number": "t.task_number",
	"updated_at":  "t.updated_at",
	"sort_order":  "t.sort_order",
}

func (r *PostgresRepository) List(ctx context.Context, tenantID uuid.UUID, filters TaskFilters) ([]models.TaskWithRelations, int, error) {
	conditions := []string{fmt.Sprintf("t.tenant_id = $%d", 1)}
	args := []any{tenantID}
	argIdx := 2

	if filters.ProjectID != nil {
		conditions = append(conditions, fmt.Sprintf("t.project_id = $%d", argIdx))
		args = append(args, *filters.ProjectID)
		argIdx++
	}
	if filters.AssigneeID != nil {
		conditions = append(conditions, fmt.Sprintf("t.assignee_id = $%d", argIdx))
		args = append(args, *filters.AssigneeID)
		argIdx++
	}
	if filters.StatusID != nil {
		conditions = append(conditions, fmt.Sprintf("t.status_id = $%d", argIdx))
		args = append(args, *filters.StatusID)
		argIdx++
	}
	if filters.Priority != nil {
		conditions = append(conditions, fmt.Sprintf("t.priority = $%d", argIdx))
		args = append(args, *filters.Priority)
		argIdx++
	}
	if filters.ParentTaskID != nil {
		conditions = append(conditions, fmt.Sprintf("t.parent_task_id = $%d", argIdx))
		args = append(args, *filters.ParentTaskID)
		argIdx++
	}
	if filters.CreatedBy != nil {
		conditions = append(conditions, fmt.Sprintf("t.created_by = $%d", argIdx))
		args = append(args, *filters.CreatedBy)
		argIdx++
	}
	if filters.DueDateFrom != nil {
		conditions = append(conditions, fmt.Sprintf("t.due_date >= $%d", argIdx))
		args = append(args, *filters.DueDateFrom)
		argIdx++
	}
	if filters.DueDateTo != nil {
		conditions = append(conditions, fmt.Sprintf("t.due_date <= $%d", argIdx))
		args = append(args, *filters.DueDateTo)
		argIdx++
	}
	if filters.Search != "" {
		conditions = append(conditions, fmt.Sprintf("t.title ILIKE '%%' || $%d || '%%'", argIdx))
		args = append(args, filters.Search)
		argIdx++
	}
	if filters.IsStandalone != nil && *filters.IsStandalone {
		conditions = append(conditions, "t.project_id IS NULL")
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM tasks t %s", where)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Sort
	sortCol := "t.created_at"
	if col, ok := validSortColumns[filters.SortBy]; ok {
		sortCol = col
	}
	sortDir := "ASC"
	if filters.SortDesc {
		sortDir = "DESC"
	}

	// Pagination
	page := filters.Page
	if page < 1 {
		page = 1
	}
	pageSize := filters.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	listQuery := fmt.Sprintf(`
		SELECT t.id, t.project_id, t.task_number, t.title, t.description, t.status_id, t.priority,
		       t.assignee_id, t.parent_task_id, t.depth, t.sort_order, t.due_date,
		       t.created_by, t.created_at, t.updated_at, t.completed_at,
		       ps.name AS status_name, ps.color AS status_color,
		       u.first_name || ' ' || u.last_name AS assignee_name,
		       p.project_key,
		       (SELECT COUNT(*) FROM tasks WHERE parent_task_id = t.id) AS subtask_count,
		       EXISTS(SELECT 1 FROM task_dependencies WHERE target_task_id = t.id AND dependency_type = 'blocks') AS has_blocked_deps
		FROM tasks t
		LEFT JOIN project_statuses ps ON t.status_id = ps.id
		LEFT JOIN users u ON t.assignee_id = u.id
		LEFT JOIN projects p ON t.project_id = p.id
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, where, sortCol, sortDir, argIdx, argIdx+1)

	args = append(args, pageSize, offset)

	rows, err := r.pool.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tasks []models.TaskWithRelations
	for rows.Next() {
		var tw models.TaskWithRelations
		if scanErr := rows.Scan(
			&tw.ID, &tw.ProjectID, &tw.TaskNumber, &tw.Title, &tw.Description,
			&tw.StatusID, &tw.Priority, &tw.AssigneeID, &tw.ParentTaskID,
			&tw.Depth, &tw.SortOrder, &tw.DueDate, &tw.CreatedBy,
			&tw.CreatedAt, &tw.UpdatedAt, &tw.CompletedAt,
			&tw.StatusName, &tw.StatusColor, &tw.AssigneeName, &tw.ProjectKey,
			&tw.SubtaskCount, &tw.HasBlockedDeps,
		); scanErr != nil {
			return nil, 0, scanErr
		}
		tasks = append(tasks, tw)
	}
	return tasks, total, rows.Err()
}

func (r *PostgresRepository) Update(ctx context.Context, task *models.Task) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE tasks SET title = $1, description = $2, status_id = $3, priority = $4,
		  assignee_id = $5, parent_task_id = $6, depth = $7, sort_order = $8,
		  due_date = $9, updated_at = $10, completed_at = $11
		 WHERE id = $12 AND tenant_id = $13`,
		task.Title, task.Description, task.StatusID, task.Priority,
		task.AssigneeID, task.ParentTaskID, task.Depth, task.SortOrder,
		task.DueDate, task.UpdatedAt, task.CompletedAt, task.ID, task.TenantID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM tasks WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) MoveTask(ctx context.Context, taskID uuid.UUID, newStatusID uuid.UUID, newSortOrder float64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE tasks SET status_id = $1, sort_order = $2, updated_at = $3 WHERE id = $4`,
		newStatusID, newSortOrder, time.Now(), taskID,
	)
	return err
}

func (r *PostgresRepository) GetSubtasks(ctx context.Context, parentID uuid.UUID, maxDepth int) ([]models.TaskWithRelations, error) {
	rows, err := r.pool.Query(ctx, `
		WITH RECURSIVE subtree AS (
			SELECT id, project_id, task_number, title, description, status_id, priority,
			       assignee_id, parent_task_id, depth, sort_order, due_date,
			       created_by, created_at, updated_at, completed_at
			FROM tasks
			WHERE parent_task_id = $1

			UNION ALL

			SELECT t.id, t.project_id, t.task_number, t.title, t.description, t.status_id, t.priority,
			       t.assignee_id, t.parent_task_id, t.depth, t.sort_order, t.due_date,
			       t.created_by, t.created_at, t.updated_at, t.completed_at
			FROM tasks t
			INNER JOIN subtree s ON t.parent_task_id = s.id
			WHERE t.depth <= $2
		)
		SELECT st.id, st.project_id, st.task_number, st.title, st.description, st.status_id, st.priority,
		       st.assignee_id, st.parent_task_id, st.depth, st.sort_order, st.due_date,
		       st.created_by, st.created_at, st.updated_at, st.completed_at,
		       ps.name AS status_name, ps.color AS status_color,
		       u.first_name || ' ' || u.last_name AS assignee_name,
		       p.project_key,
		       (SELECT COUNT(*) FROM tasks WHERE parent_task_id = st.id) AS subtask_count,
		       EXISTS(SELECT 1 FROM task_dependencies WHERE target_task_id = st.id AND dependency_type = 'blocks') AS has_blocked_deps
		FROM subtree st
		LEFT JOIN project_statuses ps ON st.status_id = ps.id
		LEFT JOIN users u ON st.assignee_id = u.id
		LEFT JOIN projects p ON st.project_id = p.id
		ORDER BY st.depth, st.sort_order
	`, parentID, maxDepth)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTasksWithRelations(rows)
}

func (r *PostgresRepository) GetParentChain(ctx context.Context, taskID uuid.UUID) ([]models.Task, error) {
	rows, err := r.pool.Query(ctx, `
		WITH RECURSIVE chain AS (
			SELECT id, project_id, task_number, title, description, status_id, priority,
			       assignee_id, parent_task_id, depth, sort_order, due_date,
			       created_by, created_at, updated_at, completed_at
			FROM tasks
			WHERE id = $1

			UNION ALL

			SELECT t.id, t.project_id, t.task_number, t.title, t.description, t.status_id, t.priority,
			       t.assignee_id, t.parent_task_id, t.depth, t.sort_order, t.due_date,
			       t.created_by, t.created_at, t.updated_at, t.completed_at
			FROM tasks t
			INNER JOIN chain c ON t.id = c.parent_task_id
		)
		SELECT id, project_id, task_number, title, description, status_id, priority,
		       assignee_id, parent_task_id, depth, sort_order, due_date,
		       created_by, created_at, updated_at, completed_at
		FROM chain
		ORDER BY depth ASC
	`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		var t models.Task
		if scanErr := rows.Scan(
			&t.ID, &t.ProjectID, &t.TaskNumber, &t.Title, &t.Description,
			&t.StatusID, &t.Priority, &t.AssigneeID, &t.ParentTaskID,
			&t.Depth, &t.SortOrder, &t.DueDate, &t.CreatedBy,
			&t.CreatedAt, &t.UpdatedAt, &t.CompletedAt,
		); scanErr != nil {
			return nil, scanErr
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (r *PostgresRepository) GetDepth(ctx context.Context, taskID uuid.UUID) (int, error) {
	var depth int
	err := r.pool.QueryRow(ctx, `SELECT depth FROM tasks WHERE id = $1`, taskID).Scan(&depth)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	return depth, err
}

func (r *PostgresRepository) CreateDependency(ctx context.Context, dep *models.TaskDependency) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO task_dependencies (id, source_task_id, target_task_id, dependency_type, created_by, created_at, tenant_id)
		 VALUES ($1, $2, $3, $4, $5, $6,
		         (SELECT p.tenant_id FROM tasks t JOIN projects p ON t.project_id = p.id WHERE t.id = $2))`,
		dep.ID, dep.SourceTaskID, dep.TargetTaskID, dep.DependencyType, dep.CreatedBy, dep.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) DeleteDependency(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM task_dependencies WHERE id = $1`, id)
	return err
}

func (r *PostgresRepository) ListDependencies(ctx context.Context, taskID uuid.UUID) ([]models.TaskDependency, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, source_task_id, target_task_id, dependency_type, created_by, created_at
		 FROM task_dependencies
		 WHERE source_task_id = $1 OR target_task_id = $1
		 ORDER BY created_at ASC`,
		taskID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deps []models.TaskDependency
	for rows.Next() {
		var d models.TaskDependency
		if scanErr := rows.Scan(&d.ID, &d.SourceTaskID, &d.TargetTaskID, &d.DependencyType, &d.CreatedBy, &d.CreatedAt); scanErr != nil {
			return nil, scanErr
		}
		deps = append(deps, d)
	}
	return deps, rows.Err()
}

func (r *PostgresRepository) HasCycle(ctx context.Context, sourceID, targetID uuid.UUID) (bool, error) {
	var hasCycle bool
	err := r.pool.QueryRow(ctx, `
		WITH RECURSIVE chain AS (
			SELECT target_task_id AS task_id
			FROM task_dependencies
			WHERE source_task_id = $2
			AND dependency_type IN ('blocks', 'blocked_by')

			UNION ALL

			SELECT td.target_task_id
			FROM task_dependencies td
			INNER JOIN chain c ON td.source_task_id = c.task_id
			WHERE td.dependency_type IN ('blocks', 'blocked_by')
		)
		SELECT EXISTS (SELECT 1 FROM chain WHERE task_id = $1)
	`, sourceID, targetID).Scan(&hasCycle)
	return hasCycle, err
}

func (r *PostgresRepository) CreateActivity(ctx context.Context, activity *models.TaskActivity) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO task_activities (id, task_id, actor_id, action, field_name, old_value, new_value, created_at, tenant_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8,
		         (SELECT p.tenant_id FROM tasks t JOIN projects p ON t.project_id = p.id WHERE t.id = $2))`,
		activity.ID, activity.TaskID, activity.ActorID, activity.Action,
		activity.FieldName, activity.OldValue, activity.NewValue, activity.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) ListActivities(ctx context.Context, taskID uuid.UUID, page, pageSize int) ([]models.TaskActivity, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM task_activities WHERE task_id = $1`, taskID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT ta.id, ta.task_id, ta.actor_id,
		        COALESCE(u.first_name || ' ' || u.last_name, '') AS actor_name,
		        ta.action, ta.field_name, ta.old_value, ta.new_value, ta.created_at
		 FROM task_activities ta
		 LEFT JOIN users u ON ta.actor_id = u.id
		 WHERE ta.task_id = $1
		 ORDER BY ta.created_at DESC
		 LIMIT $2 OFFSET $3`,
		taskID, pageSize, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var activities []models.TaskActivity
	for rows.Next() {
		var a models.TaskActivity
		if scanErr := rows.Scan(&a.ID, &a.TaskID, &a.ActorID, &a.ActorName, &a.Action, &a.FieldName, &a.OldValue, &a.NewValue, &a.CreatedAt); scanErr != nil {
			return nil, 0, scanErr
		}
		activities = append(activities, a)
	}
	return activities, total, rows.Err()
}

func (r *PostgresRepository) LinkEntity(ctx context.Context, link *models.TaskEntityLink) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO task_entity_links (id, tenant_id, task_id, entity_type, entity_id, created_by, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		link.ID, link.TenantID, link.TaskID, link.EntityType, link.EntityID, link.CreatedBy, link.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) UnlinkEntity(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM task_entity_links WHERE id = $1`, id)
	return err
}

func (r *PostgresRepository) ListEntityLinks(ctx context.Context, taskID uuid.UUID) ([]models.TaskEntityLink, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, task_id, entity_type, entity_id, entity_display_name, created_by, created_at
		 FROM task_entity_links WHERE task_id = $1 ORDER BY created_at ASC`,
		taskID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []models.TaskEntityLink
	for rows.Next() {
		var l models.TaskEntityLink
		if scanErr := rows.Scan(&l.ID, &l.TaskID, &l.EntityType, &l.EntityID, &l.EntityDisplayName, &l.CreatedBy, &l.CreatedAt); scanErr != nil {
			return nil, scanErr
		}
		links = append(links, l)
	}
	return links, rows.Err()
}

func (r *PostgresRepository) ListTasksForEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]models.TaskWithRelations, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT t.id, t.project_id, t.task_number, t.title, t.description, t.status_id, t.priority,
		       t.assignee_id, t.parent_task_id, t.depth, t.sort_order, t.due_date,
		       t.created_by, t.created_at, t.updated_at, t.completed_at,
		       ps.name AS status_name, ps.color AS status_color,
		       u.first_name || ' ' || u.last_name AS assignee_name,
		       p.project_key,
		       (SELECT COUNT(*) FROM tasks WHERE parent_task_id = t.id) AS subtask_count,
		       EXISTS(SELECT 1 FROM task_dependencies WHERE target_task_id = t.id AND dependency_type = 'blocks') AS has_blocked_deps
		FROM tasks t
		INNER JOIN task_entity_links tel ON t.id = tel.task_id
		LEFT JOIN project_statuses ps ON t.status_id = ps.id
		LEFT JOIN users u ON t.assignee_id = u.id
		LEFT JOIN projects p ON t.project_id = p.id
		WHERE tel.entity_type = $1 AND tel.entity_id = $2
		ORDER BY t.created_at DESC
	`, entityType, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTasksWithRelations(rows)
}

func (r *PostgresRepository) AttachFile(ctx context.Context, file *models.TaskFile) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO task_files (id, task_id, filename, mime_type, file_size, storage_key, thumbnail_key, uploaded_by, created_at, tenant_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,
		         (SELECT p.tenant_id FROM tasks t JOIN projects p ON t.project_id = p.id WHERE t.id = $2))`,
		file.ID, file.TaskID, file.Filename, file.MimeType, file.FileSize,
		file.StorageKey, file.ThumbnailKey, file.UploadedBy, file.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) RemoveFile(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM task_files WHERE id = $1`, id)
	return err
}

func (r *PostgresRepository) ListFiles(ctx context.Context, taskID uuid.UUID) ([]models.TaskFile, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT tf.id, tf.task_id, tf.filename, tf.mime_type, tf.file_size, tf.storage_key, tf.thumbnail_key,
		        tf.uploaded_by, COALESCE(u.first_name || ' ' || u.last_name, '') AS uploaded_by_name, tf.created_at
		 FROM task_files tf
		 LEFT JOIN users u ON tf.uploaded_by = u.id
		 WHERE tf.task_id = $1 ORDER BY tf.created_at ASC`,
		taskID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []models.TaskFile
	for rows.Next() {
		var f models.TaskFile
		if scanErr := rows.Scan(&f.ID, &f.TaskID, &f.Filename, &f.MimeType, &f.FileSize,
			&f.StorageKey, &f.ThumbnailKey, &f.UploadedBy, &f.UploadedByName, &f.CreatedAt); scanErr != nil {
			return nil, scanErr
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

func (r *PostgresRepository) SetCustomFieldValues(ctx context.Context, taskID uuid.UUID, tenantID uuid.UUID, values map[uuid.UUID]any) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	now := time.Now()
	for fieldID, value := range values {
		jsonVal, marshalErr := json.Marshal(value)
		if marshalErr != nil {
			return marshalErr
		}
		_, execErr := tx.Exec(ctx,
			`INSERT INTO task_custom_field_values (task_id, tenant_id, field_id, value, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $5)
			 ON CONFLICT (task_id, field_id) DO UPDATE SET value = $4, updated_at = $5`,
			taskID, tenantID, fieldID, jsonVal, now,
		)
		if execErr != nil {
			return execErr
		}
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepository) GetCustomFieldValues(ctx context.Context, taskID uuid.UUID) (map[string]any, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT cfd.field_name, tcfv.value
		 FROM task_custom_field_values tcfv
		 JOIN custom_field_definitions cfd ON tcfv.field_id = cfd.id
		 WHERE tcfv.task_id = $1`,
		taskID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]any)
	for rows.Next() {
		var fieldName string
		var value json.RawMessage
		if scanErr := rows.Scan(&fieldName, &value); scanErr != nil {
			return nil, scanErr
		}
		var parsed any
		if unmarshalErr := json.Unmarshal(value, &parsed); unmarshalErr != nil {
			return nil, unmarshalErr
		}
		result[fieldName] = parsed
	}
	return result, rows.Err()
}

func (r *PostgresRepository) Search(ctx context.Context, tenantID uuid.UUID, query string, filters TaskSearchFilters) ([]models.TaskWithRelations, int, error) {
	var conditions []string
	var args []any
	argIdx := 1

	// Full-text search condition
	conditions = append(conditions, fmt.Sprintf("t.search_vector @@ plainto_tsquery('german', $%d)", argIdx))
	args = append(args, query)
	argIdx++

	// Tenant isolation — mandatory filter; cross-tenant full-text search is not allowed.
	conditions = append(conditions, fmt.Sprintf("t.tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	if len(filters.ProjectIDs) > 0 {
		conditions = append(conditions, fmt.Sprintf("t.project_id = ANY($%d)", argIdx))
		args = append(args, filters.ProjectIDs)
		argIdx++
	}
	if len(filters.AssigneeIDs) > 0 {
		conditions = append(conditions, fmt.Sprintf("t.assignee_id = ANY($%d)", argIdx))
		args = append(args, filters.AssigneeIDs)
		argIdx++
	}
	if filters.Priority != nil {
		conditions = append(conditions, fmt.Sprintf("t.priority = $%d", argIdx))
		args = append(args, *filters.Priority)
		argIdx++
	}

	where := strings.Join(conditions, " AND ")

	// Count
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM tasks t WHERE %s", where)
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Pagination
	page := filters.Page
	if page < 1 {
		page = 1
	}
	pageSize := filters.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	searchQuery := fmt.Sprintf(`
		SELECT t.id, t.project_id, t.task_number, t.title, t.description, t.status_id, t.priority,
		       t.assignee_id, t.parent_task_id, t.depth, t.sort_order, t.due_date,
		       t.created_by, t.created_at, t.updated_at, t.completed_at,
		       ps.name AS status_name, ps.color AS status_color,
		       u.first_name || ' ' || u.last_name AS assignee_name,
		       p.project_key,
		       (SELECT COUNT(*) FROM tasks WHERE parent_task_id = t.id) AS subtask_count,
		       EXISTS(SELECT 1 FROM task_dependencies WHERE target_task_id = t.id AND dependency_type = 'blocks') AS has_blocked_deps
		FROM tasks t
		LEFT JOIN project_statuses ps ON t.status_id = ps.id
		LEFT JOIN users u ON t.assignee_id = u.id
		LEFT JOIN projects p ON t.project_id = p.id
		WHERE %s
		ORDER BY ts_rank(t.search_vector, plainto_tsquery('german', $1)) DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)

	args = append(args, pageSize, offset)

	rows, err := r.pool.Query(ctx, searchQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	tasks, scanErr := scanTasksWithRelations(rows)
	if scanErr != nil {
		return nil, 0, scanErr
	}
	return tasks, total, nil
}

func (r *PostgresRepository) ListByProject(ctx context.Context, projectID uuid.UUID) ([]models.Task, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, project_id, task_number, title, description, status_id, priority,
		        assignee_id, parent_task_id, depth, sort_order, due_date,
		        created_by, created_at, updated_at, completed_at
		 FROM tasks WHERE project_id = $1 ORDER BY depth ASC, sort_order ASC`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		var t models.Task
		if scanErr := rows.Scan(
			&t.ID, &t.ProjectID, &t.TaskNumber, &t.Title, &t.Description,
			&t.StatusID, &t.Priority, &t.AssigneeID, &t.ParentTaskID,
			&t.Depth, &t.SortOrder, &t.DueDate, &t.CreatedBy,
			&t.CreatedAt, &t.UpdatedAt, &t.CompletedAt,
		); scanErr != nil {
			return nil, scanErr
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (r *PostgresRepository) ListDependenciesByProject(ctx context.Context, projectID uuid.UUID) ([]models.TaskDependency, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT td.id, td.source_task_id, td.target_task_id, td.dependency_type, td.created_by, td.created_at
		 FROM task_dependencies td
		 JOIN tasks t ON td.source_task_id = t.id
		 WHERE t.project_id = $1`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deps []models.TaskDependency
	for rows.Next() {
		var d models.TaskDependency
		if scanErr := rows.Scan(&d.ID, &d.SourceTaskID, &d.TargetTaskID, &d.DependencyType, &d.CreatedBy, &d.CreatedAt); scanErr != nil {
			return nil, scanErr
		}
		deps = append(deps, d)
	}
	return deps, rows.Err()
}

func (r *PostgresRepository) GetStatusByID(ctx context.Context, statusID uuid.UUID) (string, bool, error) {
	var name string
	var isClosed bool
	err := r.pool.QueryRow(ctx,
		`SELECT name, is_closed FROM project_statuses WHERE id = $1`, statusID,
	).Scan(&name, &isClosed)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, ErrNotFound
	}
	return name, isClosed, err
}

// scanTasksWithRelations is a helper to scan task rows with relation columns
func scanTasksWithRelations(rows pgx.Rows) ([]models.TaskWithRelations, error) {
	var tasks []models.TaskWithRelations
	for rows.Next() {
		var tw models.TaskWithRelations
		if scanErr := rows.Scan(
			&tw.ID, &tw.ProjectID, &tw.TaskNumber, &tw.Title, &tw.Description,
			&tw.StatusID, &tw.Priority, &tw.AssigneeID, &tw.ParentTaskID,
			&tw.Depth, &tw.SortOrder, &tw.DueDate, &tw.CreatedBy,
			&tw.CreatedAt, &tw.UpdatedAt, &tw.CompletedAt,
			&tw.StatusName, &tw.StatusColor, &tw.AssigneeName, &tw.ProjectKey,
			&tw.SubtaskCount, &tw.HasBlockedDeps,
		); scanErr != nil {
			return nil, scanErr
		}
		tasks = append(tasks, tw)
	}
	return tasks, rows.Err()
}

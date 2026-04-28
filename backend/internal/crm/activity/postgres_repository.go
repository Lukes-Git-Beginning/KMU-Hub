package activity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

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

func (r *PostgresRepository) Create(ctx context.Context, activity *models.Activity) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO activities (id, tenant_id, activity_type, subject, description, contact_id, company_id,
		 deal_id, assigned_to, due_date, is_completed, completed_at, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
		activity.ID, activity.TenantID, activity.ActivityType, activity.Subject, activity.Description,
		activity.ContactID, activity.CompanyID, activity.DealID, activity.AssignedTo,
		activity.DueDate, activity.IsCompleted, activity.CompletedAt,
		activity.CreatedBy, activity.CreatedAt, activity.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*models.Activity, error) {
	return r.scanActivity(r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, activity_type, subject, description, contact_id, company_id, deal_id,
		 assigned_to, due_date, is_completed, completed_at, created_by, created_at, updated_at
		 FROM activities WHERE tenant_id = $1 AND id = $2`, tenantID, id,
	))
}

func (r *PostgresRepository) List(ctx context.Context, tenantID uuid.UUID, filter ListFilter, offset, limit int) ([]*models.Activity, int, error) {
	// Build WHERE clause — tenant_id is always the first filter
	conditions := []string{fmt.Sprintf("tenant_id = $%d", 1)}
	args := []any{tenantID}
	argNum := 2

	if filter.ActivityType != nil {
		conditions = append(conditions, fmt.Sprintf("activity_type = $%d", argNum))
		args = append(args, *filter.ActivityType)
		argNum++
	}

	if filter.ContactID != nil {
		conditions = append(conditions, fmt.Sprintf("contact_id = $%d", argNum))
		args = append(args, *filter.ContactID)
		argNum++
	}

	if filter.CompanyID != nil {
		conditions = append(conditions, fmt.Sprintf("company_id = $%d", argNum))
		args = append(args, *filter.CompanyID)
		argNum++
	}

	if filter.DealID != nil {
		conditions = append(conditions, fmt.Sprintf("deal_id = $%d", argNum))
		args = append(args, *filter.DealID)
		argNum++
	}

	if filter.AssignedTo != nil {
		conditions = append(conditions, fmt.Sprintf("assigned_to = $%d", argNum))
		args = append(args, *filter.AssignedTo)
		argNum++
	}

	if filter.CreatedBy != nil {
		conditions = append(conditions, fmt.Sprintf("created_by = $%d", argNum))
		args = append(args, *filter.CreatedBy)
		argNum++
	}

	if filter.IsCompleted != nil {
		conditions = append(conditions, fmt.Sprintf("is_completed = $%d", argNum))
		args = append(args, *filter.IsCompleted)
		argNum++
	}

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("LOWER(subject) LIKE $%d", argNum))
		args = append(args, "%"+strings.ToLower(filter.Search)+"%")
		argNum++
	}

	if len(filter.TagIDs) > 0 {
		conditions = append(conditions, fmt.Sprintf(`id IN (
			SELECT activity_id FROM activity_tags WHERE tag_id = ANY($%d)
			GROUP BY activity_id HAVING COUNT(DISTINCT tag_id) = $%d
		)`, argNum, argNum+1))
		args = append(args, filter.TagIDs, len(filter.TagIDs))
		argNum += 2
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM activities %s", whereClause)
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Build ORDER BY
	orderBy := "created_at"
	switch filter.SortBy {
	case "subject":
		orderBy = "LOWER(subject)"
	case "due_date":
		orderBy = "due_date"
	case "activity_type":
		orderBy = "activity_type"
	}
	direction := "DESC"
	if !filter.SortDesc {
		direction = "ASC"
	}

	// Query with pagination
	query := fmt.Sprintf(`
		SELECT id, tenant_id, activity_type, subject, description, contact_id, company_id, deal_id,
		       assigned_to, due_date, is_completed, completed_at, created_by, created_at, updated_at
		FROM activities %s
		ORDER BY %s %s NULLS LAST
		LIMIT $%d OFFSET $%d
	`, whereClause, orderBy, direction, argNum, argNum+1)

	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var activities []*models.Activity
	for rows.Next() {
		activity, scanErr := r.scanActivityFromRows(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		activities = append(activities, activity)
	}

	return activities, total, rows.Err()
}

func (r *PostgresRepository) Update(ctx context.Context, activity *models.Activity) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE activities SET subject = $1, description = $2, contact_id = $3, company_id = $4,
		 deal_id = $5, assigned_to = $6, due_date = $7, is_completed = $8, completed_at = $9,
		 updated_at = $10
		 WHERE id = $11 AND tenant_id = $12`,
		activity.Subject, activity.Description, activity.ContactID, activity.CompanyID,
		activity.DealID, activity.AssignedTo, activity.DueDate,
		activity.IsCompleted, activity.CompletedAt, activity.UpdatedAt, activity.ID, activity.TenantID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrActivityNotFound
	}
	return nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM activities WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	return err
}

func (r *PostgresRepository) GetContactName(ctx context.Context, contactID uuid.UUID) (string, error) {
	var firstName, lastName string
	err := r.pool.QueryRow(ctx,
		`SELECT first_name, last_name FROM contacts WHERE id = $1`, contactID,
	).Scan(&firstName, &lastName)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return firstName + " " + lastName, err
}

func (r *PostgresRepository) GetCompanyName(ctx context.Context, companyID uuid.UUID) (string, error) {
	var name string
	err := r.pool.QueryRow(ctx,
		`SELECT name FROM companies WHERE id = $1`, companyID,
	).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return name, err
}

func (r *PostgresRepository) GetDealName(ctx context.Context, dealID uuid.UUID) (string, error) {
	var name string
	err := r.pool.QueryRow(ctx,
		`SELECT name FROM deals WHERE id = $1`, dealID,
	).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return name, err
}

func (r *PostgresRepository) GetUserName(ctx context.Context, userID uuid.UUID) (string, error) {
	var firstName, lastName string
	err := r.pool.QueryRow(ctx,
		`SELECT first_name, last_name FROM users WHERE id = $1`, userID,
	).Scan(&firstName, &lastName)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return firstName + " " + lastName, err
}

func (r *PostgresRepository) GetTags(ctx context.Context, activityID uuid.UUID) ([]*models.Tag, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT t.id, t.name, t.color, t.entity_type, t.created_at
		 FROM tags t
		 JOIN activity_tags at ON t.id = at.tag_id
		 WHERE at.activity_id = $1
		 ORDER BY t.name`, activityID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []*models.Tag
	for rows.Next() {
		var tag models.Tag
		if scanErr := rows.Scan(&tag.ID, &tag.Name, &tag.Color, &tag.EntityType, &tag.CreatedAt); scanErr != nil {
			return nil, scanErr
		}
		tags = append(tags, &tag)
	}
	return tags, rows.Err()
}

func (r *PostgresRepository) AddTags(ctx context.Context, activityID uuid.UUID, tagIDs []uuid.UUID) error {
	for _, tagID := range tagIDs {
		_, err := r.pool.Exec(ctx,
			`INSERT INTO activity_tags (activity_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			activityID, tagID,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *PostgresRepository) RemoveTags(ctx context.Context, activityID uuid.UUID, tagIDs []uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM activity_tags WHERE activity_id = $1 AND tag_id = ANY($2)`,
		activityID, tagIDs,
	)
	return err
}

func (r *PostgresRepository) GetCustomFieldValues(ctx context.Context, activityID uuid.UUID) ([]*models.CustomFieldValueRow, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT cfv.field_id, cfd.field_name, cfv.value
		 FROM activity_custom_field_values cfv
		 JOIN custom_field_definitions cfd ON cfv.field_id = cfd.id
		 WHERE cfv.activity_id = $1`, activityID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var values []*models.CustomFieldValueRow
	for rows.Next() {
		var v models.CustomFieldValueRow
		var valueJSON []byte
		if scanErr := rows.Scan(&v.FieldID, &v.FieldName, &valueJSON); scanErr != nil {
			return nil, scanErr
		}
		if unmarshalErr := json.Unmarshal(valueJSON, &v.Value); unmarshalErr != nil {
			return nil, unmarshalErr
		}
		values = append(values, &v)
	}
	return values, rows.Err()
}

func (r *PostgresRepository) SetCustomFieldValues(ctx context.Context, activityID uuid.UUID, values map[uuid.UUID]any) error {
	for fieldID, value := range values {
		valueJSON, err := json.Marshal(value)
		if err != nil {
			return err
		}
		_, execErr := r.pool.Exec(ctx,
			`INSERT INTO activity_custom_field_values (activity_id, field_id, value, created_at, updated_at)
			 VALUES ($1, $2, $3, NOW(), NOW())
			 ON CONFLICT (activity_id, field_id) DO UPDATE SET value = $3, updated_at = NOW()`,
			activityID, fieldID, valueJSON,
		)
		if execErr != nil {
			return execErr
		}
	}
	return nil
}

func (r *PostgresRepository) ContactExists(ctx context.Context, contactID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM contacts WHERE id = $1)`, contactID,
	).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) CompanyExists(ctx context.Context, companyID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM companies WHERE id = $1)`, companyID,
	).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) DealExists(ctx context.Context, dealID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM deals WHERE id = $1)`, dealID,
	).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) UserExists(ctx context.Context, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`, userID,
	).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) TagExists(ctx context.Context, tagID uuid.UUID, entityType models.EntityType) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM tags WHERE id = $1 AND entity_type = $2)`,
		tagID, entityType,
	).Scan(&exists)
	return exists, err
}

// GetContactTimeline returns a unified timeline of activities and deal links for a contact,
// scoped to the given tenant.
func (r *PostgresRepository) GetContactTimeline(ctx context.Context, contactID, tenantID uuid.UUID, offset, limit int) ([]*TimelineEvent, int, error) {
	// Count total — both tables filtered by tenant_id
	var total int
	countQuery := `
		SELECT COUNT(*) FROM (
			SELECT id FROM activities WHERE contact_id = $1 AND tenant_id = $2
			UNION ALL
			SELECT d.id FROM deals d WHERE d.contact_id = $1 AND d.tenant_id = $2
		) sub
	`
	if err := r.pool.QueryRow(ctx, countQuery, contactID, tenantID).Scan(&total); err != nil {
		return nil, 0, err
	}

	// UNION query: activities + deal links — both arms filtered by tenant_id
	query := `
		SELECT id, 'activity' AS event_type, created_at AS occurred_at,
		       subject AS title, description,
		       COALESCE((SELECT first_name || ' ' || last_name FROM users WHERE id = created_by), '') AS created_by_name
		FROM activities
		WHERE contact_id = $1 AND tenant_id = $2

		UNION ALL

		SELECT d.id, 'deal_linked' AS event_type, d.created_at AS occurred_at,
		       'Deal verknüpft: ' || d.name AS title, NULL::text AS description,
		       COALESCE((SELECT first_name || ' ' || last_name FROM users WHERE id = d.created_by), '') AS created_by_name
		FROM deals d
		WHERE d.contact_id = $1 AND d.tenant_id = $2

		ORDER BY occurred_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.pool.Query(ctx, query, contactID, tenantID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []*TimelineEvent
	for rows.Next() {
		var e TimelineEvent
		var desc *string
		if scanErr := rows.Scan(&e.ID, &e.EventType, &e.OccurredAt, &e.Title, &desc, &e.CreatedByName); scanErr != nil {
			return nil, 0, scanErr
		}
		e.Description = desc
		events = append(events, &e)
	}

	return events, total, rows.Err()
}

// Helper to scan a single row into Activity
func (r *PostgresRepository) scanActivity(row pgx.Row) (*models.Activity, error) {
	var a models.Activity
	err := row.Scan(
		&a.ID, &a.TenantID, &a.ActivityType, &a.Subject, &a.Description,
		&a.ContactID, &a.CompanyID, &a.DealID, &a.AssignedTo,
		&a.DueDate, &a.IsCompleted, &a.CompletedAt,
		&a.CreatedBy, &a.CreatedAt, &a.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrActivityNotFound
	}
	return &a, err
}

// Helper to scan rows iterator
func (r *PostgresRepository) scanActivityFromRows(rows pgx.Rows) (*models.Activity, error) {
	var a models.Activity
	err := rows.Scan(
		&a.ID, &a.TenantID, &a.ActivityType, &a.Subject, &a.Description,
		&a.ContactID, &a.CompanyID, &a.DealID, &a.AssignedTo,
		&a.DueDate, &a.IsCompleted, &a.CompletedAt,
		&a.CreatedBy, &a.CreatedAt, &a.UpdatedAt,
	)
	return &a, err
}

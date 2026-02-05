package deal

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
	"github.com/shopspring/decimal"

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

func (r *PostgresRepository) Create(ctx context.Context, deal *models.Deal) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO deals (id, name, value, currency, stage_id, contact_id, company_id, owner_id,
		 expected_close_date, notes, created_by, created_at, updated_at, closed_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		deal.ID, deal.Name, deal.Value, deal.Currency, deal.StageID,
		deal.ContactID, deal.CompanyID, deal.OwnerID,
		deal.ExpectedCloseDate, deal.Notes, deal.CreatedBy,
		deal.CreatedAt, deal.UpdatedAt, deal.ClosedAt,
	)
	return err
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Deal, error) {
	return r.scanDeal(r.pool.QueryRow(ctx,
		`SELECT id, name, value, currency, stage_id, contact_id, company_id, owner_id,
		 expected_close_date, notes, created_by, created_at, updated_at, closed_at
		 FROM deals WHERE id = $1`, id,
	))
}

func (r *PostgresRepository) List(ctx context.Context, filter ListFilter, offset, limit int) ([]*models.Deal, int, error) {
	// Build WHERE clause
	var conditions []string
	var args []any
	argNum := 1

	if filter.StageID != nil {
		conditions = append(conditions, fmt.Sprintf("stage_id = $%d", argNum))
		args = append(args, *filter.StageID)
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

	if filter.OwnerID != nil {
		conditions = append(conditions, fmt.Sprintf("owner_id = $%d", argNum))
		args = append(args, *filter.OwnerID)
		argNum++
	}

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("LOWER(name) LIKE $%d", argNum))
		args = append(args, "%"+strings.ToLower(filter.Search)+"%")
		argNum++
	}

	if len(filter.TagIDs) > 0 {
		conditions = append(conditions, fmt.Sprintf(`id IN (
			SELECT deal_id FROM deal_tags WHERE tag_id = ANY($%d)
			GROUP BY deal_id HAVING COUNT(DISTINCT tag_id) = $%d
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
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM deals %s", whereClause)
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Build ORDER BY
	orderBy := "created_at"
	switch filter.SortBy {
	case "name":
		orderBy = "LOWER(name)"
	case "value":
		orderBy = "value"
	case "expected_close_date":
		orderBy = "expected_close_date"
	}
	direction := "DESC"
	if !filter.SortDesc {
		direction = "ASC"
	}

	// Query with pagination
	query := fmt.Sprintf(`
		SELECT id, name, value, currency, stage_id, contact_id, company_id, owner_id,
		       expected_close_date, notes, created_by, created_at, updated_at, closed_at
		FROM deals %s
		ORDER BY %s %s NULLS LAST
		LIMIT $%d OFFSET $%d
	`, whereClause, orderBy, direction, argNum, argNum+1)

	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var deals []*models.Deal
	for rows.Next() {
		deal, scanErr := r.scanDealFromRows(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		deals = append(deals, deal)
	}

	return deals, total, rows.Err()
}

func (r *PostgresRepository) Update(ctx context.Context, deal *models.Deal) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE deals SET name = $1, value = $2, currency = $3, stage_id = $4,
		 contact_id = $5, company_id = $6, owner_id = $7, expected_close_date = $8,
		 notes = $9, updated_at = $10
		 WHERE id = $11`,
		deal.Name, deal.Value, deal.Currency, deal.StageID,
		deal.ContactID, deal.CompanyID, deal.OwnerID, deal.ExpectedCloseDate,
		deal.Notes, deal.UpdatedAt, deal.ID,
	)
	return err
}

func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM deals WHERE id = $1`, id)
	return err
}

func (r *PostgresRepository) GetStageName(ctx context.Context, stageID uuid.UUID) (string, error) {
	var name string
	err := r.pool.QueryRow(ctx,
		`SELECT name FROM pipeline_stages WHERE id = $1`, stageID,
	).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return name, err
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

func (r *PostgresRepository) GetOwnerName(ctx context.Context, ownerID uuid.UUID) (string, error) {
	var firstName, lastName string
	err := r.pool.QueryRow(ctx,
		`SELECT first_name, last_name FROM users WHERE id = $1`, ownerID,
	).Scan(&firstName, &lastName)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return firstName + " " + lastName, err
}

func (r *PostgresRepository) GetTags(ctx context.Context, dealID uuid.UUID) ([]*models.Tag, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT t.id, t.name, t.color, t.entity_type, t.created_at
		 FROM tags t
		 JOIN deal_tags dt ON t.id = dt.tag_id
		 WHERE dt.deal_id = $1
		 ORDER BY t.name`, dealID,
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

func (r *PostgresRepository) AddTags(ctx context.Context, dealID uuid.UUID, tagIDs []uuid.UUID) error {
	for _, tagID := range tagIDs {
		_, err := r.pool.Exec(ctx,
			`INSERT INTO deal_tags (deal_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			dealID, tagID,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *PostgresRepository) RemoveTags(ctx context.Context, dealID uuid.UUID, tagIDs []uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM deal_tags WHERE deal_id = $1 AND tag_id = ANY($2)`,
		dealID, tagIDs,
	)
	return err
}

func (r *PostgresRepository) GetCustomFieldValues(ctx context.Context, dealID uuid.UUID) ([]*models.CustomFieldValueRow, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT cfv.field_id, cfd.field_name, cfv.value
		 FROM deal_custom_field_values cfv
		 JOIN custom_field_definitions cfd ON cfv.field_id = cfd.id
		 WHERE cfv.deal_id = $1`, dealID,
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

func (r *PostgresRepository) SetCustomFieldValues(ctx context.Context, dealID uuid.UUID, values map[uuid.UUID]any) error {
	for fieldID, value := range values {
		valueJSON, err := json.Marshal(value)
		if err != nil {
			return err
		}
		_, execErr := r.pool.Exec(ctx,
			`INSERT INTO deal_custom_field_values (deal_id, field_id, value, created_at, updated_at)
			 VALUES ($1, $2, $3, NOW(), NOW())
			 ON CONFLICT (deal_id, field_id) DO UPDATE SET value = $3, updated_at = NOW()`,
			dealID, fieldID, valueJSON,
		)
		if execErr != nil {
			return execErr
		}
	}
	return nil
}

func (r *PostgresRepository) StageExists(ctx context.Context, stageID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM pipeline_stages WHERE id = $1)`, stageID,
	).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) GetStage(ctx context.Context, stageID uuid.UUID) (*models.PipelineStage, error) {
	var s models.PipelineStage
	var probFloat float64
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, color, sort_order, is_won, is_lost, probability, created_at, updated_at
		 FROM pipeline_stages WHERE id = $1`, stageID,
	).Scan(&s.ID, &s.Name, &s.Color, &s.SortOrder, &s.IsWon, &s.IsLost, &probFloat, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrStageNotFound
	}
	if err != nil {
		return nil, err
	}
	s.Probability = decimal.NewFromFloat(probFloat)
	return &s, nil
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

func (r *PostgresRepository) OwnerExists(ctx context.Context, ownerID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`, ownerID,
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

func (r *PostgresRepository) SetClosedAt(ctx context.Context, dealID uuid.UUID, closedAt *time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE deals SET closed_at = $1, updated_at = NOW() WHERE id = $2`,
		closedAt, dealID,
	)
	return err
}

// Helper to scan a single row into Deal
func (r *PostgresRepository) scanDeal(row pgx.Row) (*models.Deal, error) {
	var d models.Deal
	var valueFloat float64
	err := row.Scan(
		&d.ID, &d.Name, &valueFloat, &d.Currency, &d.StageID,
		&d.ContactID, &d.CompanyID, &d.OwnerID,
		&d.ExpectedCloseDate, &d.Notes, &d.CreatedBy,
		&d.CreatedAt, &d.UpdatedAt, &d.ClosedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrDealNotFound
	}
	if err != nil {
		return nil, err
	}
	d.Value = decimal.NewFromFloat(valueFloat)
	return &d, nil
}

// Helper to scan rows iterator
func (r *PostgresRepository) scanDealFromRows(rows pgx.Rows) (*models.Deal, error) {
	var d models.Deal
	var valueFloat float64
	err := rows.Scan(
		&d.ID, &d.Name, &valueFloat, &d.Currency, &d.StageID,
		&d.ContactID, &d.CompanyID, &d.OwnerID,
		&d.ExpectedCloseDate, &d.Notes, &d.CreatedBy,
		&d.CreatedAt, &d.UpdatedAt, &d.ClosedAt,
	)
	if err != nil {
		return nil, err
	}
	d.Value = decimal.NewFromFloat(valueFloat)
	return &d, nil
}

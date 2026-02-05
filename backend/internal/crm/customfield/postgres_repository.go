package customfield

import (
	"context"
	"encoding/json"
	"errors"

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

func (r *PostgresRepository) Create(ctx context.Context, field *models.CustomFieldDefinition) error {
	optionsJSON, err := json.Marshal(field.Options)
	if err != nil {
		return err
	}

	_, err = r.pool.Exec(ctx,
		`INSERT INTO custom_field_definitions
		 (id, entity_type, field_name, field_label, field_type, options, default_value, is_required, sort_order, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		field.ID, field.EntityType, field.FieldName, field.FieldLabel, field.FieldType,
		optionsJSON, field.DefaultValue, field.IsRequired, field.SortOrder,
		field.CreatedBy, field.CreatedAt, field.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.CustomFieldDefinition, error) {
	return r.scanField(r.pool.QueryRow(ctx,
		`SELECT id, entity_type, field_name, field_label, field_type, options, default_value, is_required, sort_order, created_by, created_at, updated_at
		 FROM custom_field_definitions WHERE id = $1`, id,
	))
}

func (r *PostgresRepository) GetByEntityAndName(ctx context.Context, entityType models.EntityType, fieldName string) (*models.CustomFieldDefinition, error) {
	return r.scanField(r.pool.QueryRow(ctx,
		`SELECT id, entity_type, field_name, field_label, field_type, options, default_value, is_required, sort_order, created_by, created_at, updated_at
		 FROM custom_field_definitions WHERE entity_type = $1 AND field_name = $2`, entityType, fieldName,
	))
}

func (r *PostgresRepository) List(ctx context.Context, entityType *models.EntityType, offset, limit int) ([]*models.CustomFieldDefinition, int, error) {
	var total int
	var countErr error

	if entityType != nil {
		countErr = r.pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM custom_field_definitions WHERE entity_type = $1`, *entityType,
		).Scan(&total)
	} else {
		countErr = r.pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM custom_field_definitions`,
		).Scan(&total)
	}
	if countErr != nil {
		return nil, 0, countErr
	}

	var rows pgx.Rows
	var err error

	if entityType != nil {
		rows, err = r.pool.Query(ctx,
			`SELECT id, entity_type, field_name, field_label, field_type, options, default_value, is_required, sort_order, created_by, created_at, updated_at
			 FROM custom_field_definitions
			 WHERE entity_type = $1
			 ORDER BY sort_order ASC, created_at ASC
			 LIMIT $2 OFFSET $3`, *entityType, limit, offset,
		)
	} else {
		rows, err = r.pool.Query(ctx,
			`SELECT id, entity_type, field_name, field_label, field_type, options, default_value, is_required, sort_order, created_by, created_at, updated_at
			 FROM custom_field_definitions
			 ORDER BY entity_type ASC, sort_order ASC, created_at ASC
			 LIMIT $1 OFFSET $2`, limit, offset,
		)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var fields []*models.CustomFieldDefinition
	for rows.Next() {
		field, scanErr := r.scanFieldFromRows(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		fields = append(fields, field)
	}

	return fields, total, rows.Err()
}

func (r *PostgresRepository) Update(ctx context.Context, field *models.CustomFieldDefinition) error {
	optionsJSON, err := json.Marshal(field.Options)
	if err != nil {
		return err
	}

	_, err = r.pool.Exec(ctx,
		`UPDATE custom_field_definitions
		 SET field_label = $1, options = $2, default_value = $3, is_required = $4, sort_order = $5, updated_at = $6
		 WHERE id = $7`,
		field.FieldLabel, optionsJSON, field.DefaultValue, field.IsRequired, field.SortOrder, field.UpdatedAt, field.ID,
	)
	return err
}

func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM custom_field_definitions WHERE id = $1`, id,
	)
	return err
}

// Helper to scan a single row into CustomFieldDefinition
func (r *PostgresRepository) scanField(row pgx.Row) (*models.CustomFieldDefinition, error) {
	var field models.CustomFieldDefinition
	var optionsJSON []byte

	err := row.Scan(
		&field.ID, &field.EntityType, &field.FieldName, &field.FieldLabel, &field.FieldType,
		&optionsJSON, &field.DefaultValue, &field.IsRequired, &field.SortOrder,
		&field.CreatedBy, &field.CreatedAt, &field.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrFieldNotFound
	}
	if err != nil {
		return nil, err
	}

	if optionsJSON != nil {
		if unmarshalErr := json.Unmarshal(optionsJSON, &field.Options); unmarshalErr != nil {
			return nil, unmarshalErr
		}
	}

	return &field, nil
}

// Helper to scan rows iterator
func (r *PostgresRepository) scanFieldFromRows(rows pgx.Rows) (*models.CustomFieldDefinition, error) {
	var field models.CustomFieldDefinition
	var optionsJSON []byte

	err := rows.Scan(
		&field.ID, &field.EntityType, &field.FieldName, &field.FieldLabel, &field.FieldType,
		&optionsJSON, &field.DefaultValue, &field.IsRequired, &field.SortOrder,
		&field.CreatedBy, &field.CreatedAt, &field.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if optionsJSON != nil {
		if unmarshalErr := json.Unmarshal(optionsJSON, &field.Options); unmarshalErr != nil {
			return nil, unmarshalErr
		}
	}

	return &field, nil
}

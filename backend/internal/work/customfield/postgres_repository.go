package customfield

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL custom field repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, tenantID, name, fieldType string, options []string, position int) (*Definition, error) {
	opts, err := marshalOptions(options)
	if err != nil {
		return nil, err
	}

	d := &Definition{}
	var rawOpts []byte
	err = r.pool.QueryRow(ctx,
		`INSERT INTO work_custom_field_definitions
		     (tenant_id, name, field_type, options, position, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $6)
		 RETURNING id, tenant_id, name, field_type, options, position, created_at, updated_at`,
		tenantID, name, fieldType, opts, position, time.Now(),
	).Scan(&d.ID, &d.TenantID, &d.Name, &d.FieldType, &rawOpts, &d.Position, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		if isDuplicateError(err) {
			return nil, ErrDuplicateName
		}
		return nil, err
	}
	d.Options = unmarshalOptions(rawOpts)
	return d, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, tenantID, id string) (*Definition, error) {
	d := &Definition{}
	var rawOpts []byte
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, field_type, options, position, created_at, updated_at
		 FROM work_custom_field_definitions
		 WHERE id = $1 AND tenant_id = $2`,
		id, tenantID,
	).Scan(&d.ID, &d.TenantID, &d.Name, &d.FieldType, &rawOpts, &d.Position, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	d.Options = unmarshalOptions(rawOpts)
	return d, nil
}

func (r *PostgresRepository) List(ctx context.Context, tenantID string) ([]*Definition, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, field_type, options, position, created_at, updated_at
		 FROM work_custom_field_definitions
		 WHERE tenant_id = $1
		 ORDER BY position ASC, name ASC`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var defs []*Definition
	for rows.Next() {
		d := &Definition{}
		var rawOpts []byte
		if scanErr := rows.Scan(&d.ID, &d.TenantID, &d.Name, &d.FieldType, &rawOpts, &d.Position, &d.CreatedAt, &d.UpdatedAt); scanErr != nil {
			return nil, scanErr
		}
		d.Options = unmarshalOptions(rawOpts)
		defs = append(defs, d)
	}
	return defs, rows.Err()
}

func (r *PostgresRepository) Update(ctx context.Context, tenantID, id, name, fieldType string, options []string, position int) (*Definition, error) {
	opts, err := marshalOptions(options)
	if err != nil {
		return nil, err
	}

	d := &Definition{}
	var rawOpts []byte
	err = r.pool.QueryRow(ctx,
		`UPDATE work_custom_field_definitions
		 SET name = $3, field_type = $4, options = $5, position = $6, updated_at = $7
		 WHERE id = $1 AND tenant_id = $2
		 RETURNING id, tenant_id, name, field_type, options, position, created_at, updated_at`,
		id, tenantID, name, fieldType, opts, position, time.Now(),
	).Scan(&d.ID, &d.TenantID, &d.Name, &d.FieldType, &rawOpts, &d.Position, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		if isDuplicateError(err) {
			return nil, ErrDuplicateName
		}
		return nil, err
	}
	d.Options = unmarshalOptions(rawOpts)
	return d, nil
}

func (r *PostgresRepository) Delete(ctx context.Context, tenantID, id string) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM work_custom_field_definitions WHERE id = $1 AND tenant_id = $2`,
		id, tenantID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// marshalOptions serialises a string slice to JSON.
func marshalOptions(opts []string) ([]byte, error) {
	if opts == nil {
		opts = []string{}
	}
	return json.Marshal(opts)
}

// unmarshalOptions deserialises JSONB bytes to a string slice.
func unmarshalOptions(raw []byte) []string {
	if len(raw) == 0 {
		return []string{}
	}
	var opts []string
	if err := json.Unmarshal(raw, &opts); err != nil {
		return []string{}
	}
	return opts
}

// isDuplicateError checks whether a pgx error is a unique-constraint violation.
func isDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	type pgErr interface {
		SQLState() string
	}
	var pe pgErr
	if errors.As(err, &pe) {
		return pe.SQLState() == "23505"
	}
	return false
}

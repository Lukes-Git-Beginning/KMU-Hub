package share

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL share repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, share *models.DocumentShare) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO document_shares (id, entity_type, entity_id, shared_with_user_id, permission, shared_by, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		share.ID, share.EntityType, share.EntityID, share.SharedWithUserID,
		share.Permission, share.SharedBy, share.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM document_shares WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrShareNotFound
	}
	return nil
}

func (r *PostgresRepository) ListByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]*models.DocumentShare, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT ds.id, ds.entity_type, ds.entity_id, ds.shared_with_user_id,
		        COALESCE(u_with.first_name || ' ' || u_with.last_name, '') AS shared_with_user_name,
		        ds.permission, ds.shared_by,
		        COALESCE(u_by.first_name || ' ' || u_by.last_name, '') AS shared_by_name,
		        ds.created_at
		 FROM document_shares ds
		 LEFT JOIN users u_with ON ds.shared_with_user_id = u_with.id
		 LEFT JOIN users u_by ON ds.shared_by = u_by.id
		 WHERE ds.entity_type = $1 AND ds.entity_id = $2
		 ORDER BY ds.created_at DESC`, entityType, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanShares(rows)
}

func (r *PostgresRepository) ListSharedWithUser(ctx context.Context, userID uuid.UUID, entityType string) ([]*models.DocumentShare, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT ds.id, ds.entity_type, ds.entity_id, ds.shared_with_user_id,
		        COALESCE(u_with.first_name || ' ' || u_with.last_name, '') AS shared_with_user_name,
		        ds.permission, ds.shared_by,
		        COALESCE(u_by.first_name || ' ' || u_by.last_name, '') AS shared_by_name,
		        ds.created_at
		 FROM document_shares ds
		 LEFT JOIN users u_with ON ds.shared_with_user_id = u_with.id
		 LEFT JOIN users u_by ON ds.shared_by = u_by.id
		 WHERE ds.shared_with_user_id = $1 AND ds.entity_type = $2
		 ORDER BY ds.created_at DESC`, userID, entityType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanShares(rows)
}

func (r *PostgresRepository) GetUserPermission(ctx context.Context, entityType string, entityID uuid.UUID, userID uuid.UUID) (*models.DocumentShare, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT ds.id, ds.entity_type, ds.entity_id, ds.shared_with_user_id,
		        COALESCE(u_with.first_name || ' ' || u_with.last_name, '') AS shared_with_user_name,
		        ds.permission, ds.shared_by,
		        COALESCE(u_by.first_name || ' ' || u_by.last_name, '') AS shared_by_name,
		        ds.created_at
		 FROM document_shares ds
		 LEFT JOIN users u_with ON ds.shared_with_user_id = u_with.id
		 LEFT JOIN users u_by ON ds.shared_by = u_by.id
		 WHERE ds.entity_type = $1 AND ds.entity_id = $2 AND ds.shared_with_user_id = $3`,
		entityType, entityID, userID)

	var s models.DocumentShare
	err := row.Scan(
		&s.ID, &s.EntityType, &s.EntityID, &s.SharedWithUserID, &s.SharedWithUserName,
		&s.Permission, &s.SharedBy, &s.SharedByName, &s.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrShareNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// HasAccess checks if a user has access to an entity and returns the permission level.
func (r *PostgresRepository) HasAccess(ctx context.Context, entityType string, entityID uuid.UUID, userID uuid.UUID) (bool, string, error) {
	var permission string
	err := r.pool.QueryRow(ctx,
		`SELECT permission FROM document_shares
		 WHERE entity_type = $1 AND entity_id = $2 AND shared_with_user_id = $3`,
		entityType, entityID, userID,
	).Scan(&permission)

	if errors.Is(err, pgx.ErrNoRows) {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}
	return true, permission, nil
}

// scanShares scans multiple share rows.
func (r *PostgresRepository) scanShares(rows pgx.Rows) ([]*models.DocumentShare, error) {
	var shares []*models.DocumentShare
	for rows.Next() {
		var s models.DocumentShare
		if err := rows.Scan(
			&s.ID, &s.EntityType, &s.EntityID, &s.SharedWithUserID, &s.SharedWithUserName,
			&s.Permission, &s.SharedBy, &s.SharedByName, &s.CreatedAt,
		); err != nil {
			return nil, err
		}
		shares = append(shares, &s)
	}
	return shares, rows.Err()
}

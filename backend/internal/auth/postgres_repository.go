package auth

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) CreateUser(ctx context.Context, user *models.User) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, first_name, last_name, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		user.ID, user.Email, user.PasswordHash, user.FirstName, user.LastName,
		user.IsActive, user.CreatedAt, user.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, first_name, last_name, is_active, created_at, updated_at
		 FROM users WHERE email = $1`, email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName,
		&user.IsActive, &user.CreatedAt, &user.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	return &user, err
}

func (r *PostgresRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, first_name, last_name, is_active, created_at, updated_at
		 FROM users WHERE id = $1`, id,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName,
		&user.IsActive, &user.CreatedAt, &user.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	return &user, err
}

func (r *PostgresRepository) UpdateUser(ctx context.Context, user *models.User) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET first_name = $1, last_name = $2, is_active = $3, updated_at = $4 WHERE id = $5`,
		user.FirstName, user.LastName, user.IsActive, user.UpdatedAt, user.ID,
	)
	return err
}

func (r *PostgresRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`,
		passwordHash, userID,
	)
	return err
}

func (r *PostgresRepository) ListUsers(ctx context.Context, offset, limit int) ([]*models.User, int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, email, password_hash, first_name, last_name, is_active, created_at, updated_at
		 FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName,
			&u.IsActive, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, err
		}
		users = append(users, &u)
	}

	return users, total, rows.Err()
}

func (r *PostgresRepository) StoreRefreshToken(ctx context.Context, token *models.RefreshToken) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, revoked, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		token.ID, token.UserID, token.TokenHash, token.ExpiresAt, token.Revoked, token.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) GetRefreshTokenByHash(ctx context.Context, hash string) (*models.RefreshToken, error) {
	var token models.RefreshToken
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, token_hash, expires_at, revoked, created_at
		 FROM refresh_tokens WHERE token_hash = $1`, hash,
	).Scan(&token.ID, &token.UserID, &token.TokenHash, &token.ExpiresAt, &token.Revoked, &token.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTokenInvalid
	}
	return &token, err
}

func (r *PostgresRepository) RevokeRefreshToken(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked = true WHERE id = $1`, id,
	)
	return err
}

func (r *PostgresRepository) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked = true WHERE user_id = $1 AND revoked = false`, userID,
	)
	return err
}

func (r *PostgresRepository) AssignRole(ctx context.Context, userID uuid.UUID, roleName string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO user_roles (user_id, role_id)
		 SELECT $1, r.id FROM roles r WHERE r.name = $2
		 ON CONFLICT DO NOTHING`, userID, roleName,
	)
	return err
}

func (r *PostgresRepository) RemoveRole(ctx context.Context, userID uuid.UUID, roleName string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM user_roles WHERE user_id = $1 AND role_id = (SELECT id FROM roles WHERE name = $2)`,
		userID, roleName,
	)
	return err
}

func (r *PostgresRepository) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]string, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT r.name FROM roles r
		 JOIN user_roles ur ON ur.role_id = r.id
		 WHERE ur.user_id = $1`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		roles = append(roles, name)
	}
	return roles, rows.Err()
}

func (r *PostgresRepository) GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]string, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT DISTINCT p.name FROM permissions p
		 JOIN role_permissions rp ON rp.permission_id = p.id
		 JOIN user_roles ur ON ur.role_id = rp.role_id
		 WHERE ur.user_id = $1`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		permissions = append(permissions, name)
	}
	return permissions, rows.Err()
}

func (r *PostgresRepository) UserHasPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM permissions p
			JOIN role_permissions rp ON rp.permission_id = p.id
			JOIN user_roles ur ON ur.role_id = rp.role_id
			WHERE ur.user_id = $1 AND p.resource = $2 AND p.action = $3
		)`, userID, resource, action,
	).Scan(&exists)
	return exists, err
}

// Invitation methods

func (r *PostgresRepository) CreateInvitation(ctx context.Context, inv *models.Invitation) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO invitations (id, email, role, token_hash, created_by, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		inv.ID, inv.Email, inv.Role, inv.TokenHash, inv.CreatedBy, inv.ExpiresAt, inv.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) GetInvitationByToken(ctx context.Context, tokenHash string) (*models.Invitation, error) {
	var inv models.Invitation
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, role, token_hash, created_by, expires_at, accepted_at, created_at
		 FROM invitations WHERE token_hash = $1`, tokenHash,
	).Scan(&inv.ID, &inv.Email, &inv.Role, &inv.TokenHash, &inv.CreatedBy,
		&inv.ExpiresAt, &inv.AcceptedAt, &inv.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvitationNotFound
	}
	return &inv, err
}

func (r *PostgresRepository) GetInvitationByID(ctx context.Context, id uuid.UUID) (*models.Invitation, error) {
	var inv models.Invitation
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, role, token_hash, created_by, expires_at, accepted_at, created_at
		 FROM invitations WHERE id = $1`, id,
	).Scan(&inv.ID, &inv.Email, &inv.Role, &inv.TokenHash, &inv.CreatedBy,
		&inv.ExpiresAt, &inv.AcceptedAt, &inv.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvitationNotFound
	}
	return &inv, err
}

func (r *PostgresRepository) ListPendingInvitations(ctx context.Context) ([]*models.Invitation, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, email, role, token_hash, created_by, expires_at, accepted_at, created_at
		 FROM invitations
		 WHERE accepted_at IS NULL
		 ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invitations []*models.Invitation
	for rows.Next() {
		var inv models.Invitation
		if err := rows.Scan(&inv.ID, &inv.Email, &inv.Role, &inv.TokenHash, &inv.CreatedBy,
			&inv.ExpiresAt, &inv.AcceptedAt, &inv.CreatedAt); err != nil {
			return nil, err
		}
		invitations = append(invitations, &inv)
	}
	return invitations, rows.Err()
}

func (r *PostgresRepository) MarkInvitationAccepted(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE invitations SET accepted_at = NOW() WHERE id = $1`, id,
	)
	return err
}

func (r *PostgresRepository) DeleteInvitation(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM invitations WHERE id = $1`, id,
	)
	return err
}

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
		`INSERT INTO users (id, tenant_id, email, password_hash, first_name, last_name, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		user.ID, user.TenantID, user.Email, user.PasswordHash, user.FirstName, user.LastName,
		user.IsActive, user.CreatedAt, user.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	// lower(email) match is case-insensitive and index-backed (idx_users_email
	// on lower(email), migration 000148). Callers normalize the input, but this
	// also matches any legacy mixed-case row not yet backfilled.
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, email, password_hash, first_name, last_name, is_active, created_at, updated_at,
		        two_factor_enabled, locale, avatar_url
		 FROM users WHERE lower(email) = $1`, email,
	).Scan(&user.ID, &user.TenantID, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName,
		&user.IsActive, &user.CreatedAt, &user.UpdatedAt, &user.TwoFactorEnabled, &user.Locale, &user.AvatarURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	return &user, err
}

func (r *PostgresRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, email, password_hash, first_name, last_name, is_active, created_at, updated_at,
		        two_factor_enabled, locale, avatar_url
		 FROM users WHERE id = $1`, id,
	).Scan(&user.ID, &user.TenantID, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName,
		&user.IsActive, &user.CreatedAt, &user.UpdatedAt, &user.TwoFactorEnabled, &user.Locale, &user.AvatarURL)
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

// UpdateProfile writes the self-service profile fields. Unlike UpdateUser it
// never touches is_active (that stays admin-only).
func (r *PostgresRepository) UpdateProfile(ctx context.Context, user *models.User) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET first_name = $1, last_name = $2, avatar_url = $3, updated_at = $4 WHERE id = $5`,
		user.FirstName, user.LastName, user.AvatarURL, user.UpdatedAt, user.ID,
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
		`SELECT id, tenant_id, email, password_hash, first_name, last_name, is_active, created_at, updated_at,
		        two_factor_enabled, locale, avatar_url
		 FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.TenantID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName,
			&u.IsActive, &u.CreatedAt, &u.UpdatedAt, &u.TwoFactorEnabled, &u.Locale, &u.AvatarURL); err != nil {
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

// GetEffectivePermissions returns the raw (role × fine-grained grant) rows a
// user holds. A role without a single fine-grained grant still produces one row
// with empty Key/Scope, so it survives into the roles list — dropping it would
// hide a role the user demonstrably has.
//
// The `p.resource LIKE '%:%'` filter separates the two permission worlds that
// coexist since migration 000256: the three-segment catalogue keys
// ("work:task:edit" → resource "work:task") belong in this response, the coarse
// legacy keys ("files:write" → resource "files") are the currency of the
// existing RequirePermission gates and do not.
func (r *PostgresRepository) GetEffectivePermissions(ctx context.Context, userID uuid.UUID) ([]EffectiveGrantRow, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT r.id, r.name, r.color, (r.tenant_id IS NULL) AS is_system,
		        COALESCE(g.key, ''), COALESCE(g.scope, '')
		 FROM user_roles ur
		 JOIN roles r ON r.id = ur.role_id
		 LEFT JOIN (
		     SELECT rp.role_id, p.name AS key, rp.scope
		     FROM role_permissions rp
		     JOIN permissions p ON p.id = rp.permission_id
		     WHERE p.resource LIKE '%:%'
		 ) g ON g.role_id = r.id
		 WHERE ur.user_id = $1
		 ORDER BY r.name, g.key`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var grants []EffectiveGrantRow
	for rows.Next() {
		var g EffectiveGrantRow
		if err := rows.Scan(&g.RoleID, &g.RoleName, &g.RoleColor, &g.RoleIsSystem, &g.Key, &g.Scope); err != nil {
			return nil, err
		}
		grants = append(grants, g)
	}
	return grants, rows.Err()
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

// invitationColumns is the read shape of an invitation; every invitation query
// selects exactly these, in this order, and scans through scanInvitation.
const invitationColumns = `id, tenant_id, email, role, token_hash, created_by, expires_at, accepted_at, created_at`

func scanInvitation(row pgx.Row) (*models.Invitation, error) {
	var inv models.Invitation
	err := row.Scan(&inv.ID, &inv.TenantID, &inv.Email, &inv.Role, &inv.TokenHash,
		&inv.CreatedBy, &inv.ExpiresAt, &inv.AcceptedAt, &inv.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvitationNotFound
	}
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

func (r *PostgresRepository) CreateInvitation(ctx context.Context, inv *models.Invitation) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO invitations (id, tenant_id, email, role, token_hash, created_by, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		inv.ID, inv.TenantID, inv.Email, inv.Role, inv.TokenHash, inv.CreatedBy, inv.ExpiresAt, inv.CreatedAt,
	)
	return err
}

// GetInvitationByToken looks the invitation up by token hash alone. The caller
// has no tenant yet — the token is what carries the tenant, and the row it
// finds is what determines which tenant the account is created in.
func (r *PostgresRepository) GetInvitationByToken(ctx context.Context, tokenHash string) (*models.Invitation, error) {
	return scanInvitation(r.pool.QueryRow(ctx,
		`SELECT `+invitationColumns+` FROM invitations WHERE token_hash = $1`, tokenHash,
	))
}

// GetPendingInvitationByEmail finds an unaccepted, unexpired invitation for
// the address in any tenant. Deliberately not tenant-scoped: users.email is
// globally unique, so an address invited into one tenant cannot become an
// account in another. Only the provisioning path calls it, under system
// context — RLS would otherwise hide exactly the row it needs to see.
func (r *PostgresRepository) GetPendingInvitationByEmail(ctx context.Context, email string) (*models.Invitation, error) {
	return scanInvitation(r.pool.QueryRow(ctx,
		`SELECT `+invitationColumns+`
		 FROM invitations
		 WHERE email = $1 AND accepted_at IS NULL AND expires_at > NOW()
		 ORDER BY created_at DESC
		 LIMIT 1`, email,
	))
}

func (r *PostgresRepository) GetInvitationByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Invitation, error) {
	return scanInvitation(r.pool.QueryRow(ctx,
		`SELECT `+invitationColumns+` FROM invitations WHERE id = $1 AND tenant_id = $2`, id, tenantID,
	))
}

func (r *PostgresRepository) ListPendingInvitations(ctx context.Context, tenantID uuid.UUID) ([]*models.Invitation, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+invitationColumns+`
		 FROM invitations
		 WHERE tenant_id = $1 AND accepted_at IS NULL
		 ORDER BY created_at DESC`, tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Non-nil so an empty list marshals as [] rather than null.
	invitations := []*models.Invitation{}
	for rows.Next() {
		inv, scanErr := scanInvitation(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		invitations = append(invitations, inv)
	}
	return invitations, rows.Err()
}

// AcceptInvitation claims the invitation, creates the account and assigns the
// invited role in one transaction.
//
// The claim is the conditional UPDATE, not a preceding read: two accepts
// arriving together would both pass a read-then-write check and create two
// accounts from one invitation. The UPDATE takes the row lock, so the second
// transaction sees accepted_at already set and gets ErrInvitationAlreadyUsed.
// Everything after it shares the transaction — a failure while writing the
// user rolls the claim back instead of burning the invitation.
func (r *PostgresRepository) AcceptInvitation(ctx context.Context, inv *models.Invitation, user *models.User) error {
	// Plain Begin: the pool's PrepareConn hook already stamps the session GUCs
	// from ctx, and the accept path runs under sysctx.With — the connection
	// carries app.role='system' before the first statement. Importing
	// internal/database here is not an option (database → middleware → auth).
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	claim, err := tx.Exec(ctx,
		`UPDATE invitations SET accepted_at = NOW() WHERE id = $1 AND accepted_at IS NULL`, inv.ID,
	)
	if err != nil {
		return err
	}
	if claim.RowsAffected() == 0 {
		return ErrInvitationAlreadyUsed
	}

	if _, err = tx.Exec(ctx,
		`INSERT INTO users (id, tenant_id, email, password_hash, first_name, last_name, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		user.ID, user.TenantID, user.Email, user.PasswordHash, user.FirstName, user.LastName,
		user.IsActive, user.CreatedAt, user.UpdatedAt,
	); err != nil {
		return err
	}

	// The account is new, so nothing can conflict here: zero rows means the
	// invited role does not exist. Letting that pass would create an account
	// with no role at all, which is a support case, not a login.
	assigned, err := tx.Exec(ctx,
		`INSERT INTO user_roles (user_id, role_id)
		 SELECT $1, r.id FROM roles r WHERE r.name = $2`, user.ID, inv.Role,
	)
	if err != nil {
		return err
	}
	if assigned.RowsAffected() == 0 {
		return ErrRoleNotFound
	}

	return tx.Commit(ctx)
}

// ProvisionTenant creates the tenant row, its module activations and the first
// administrator's invitation in a single transaction.
//
// Plain Begin, like AcceptInvitation: the pool's PrepareConn hook stamps the
// session GUCs from ctx, and the caller runs under sysctx.With, so the
// connection carries app.role='system' before the first statement. Without it
// the INSERT into tenants would fail its WITH CHECK — the row being created is
// not the caller's tenant.
func (r *PostgresRepository) ProvisionTenant(ctx context.Context, tenant *models.Tenant, moduleIDs []string, inv *models.Invitation) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err = tx.Exec(ctx,
		`INSERT INTO tenants (id, name, plan_type, support_tier, subscription_status,
		                      billing_period_end, seat_limit, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)`,
		tenant.ID, tenant.Name, tenant.PlanType, tenant.SupportTier, tenant.SubscriptionStatus,
		tenant.BillingPeriodEnd, tenant.SeatLimit, tenant.CreatedAt,
	); err != nil {
		return err
	}

	// updated_by stays NULL: the activation comes out of provisioning, not out
	// of an administrator toggling a module, and the platform operator is not
	// a user of this tenant.
	if len(moduleIDs) > 0 {
		if _, err = tx.Exec(ctx,
			`INSERT INTO tenant_module_activations (tenant_id, module_id, active)
			 SELECT $1, m, TRUE FROM unnest($2::text[]) AS m`,
			tenant.ID, moduleIDs,
		); err != nil {
			return err
		}
	}

	if _, err = tx.Exec(ctx,
		`INSERT INTO invitations (id, tenant_id, email, role, token_hash, created_by, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		inv.ID, inv.TenantID, inv.Email, inv.Role, inv.TokenHash, inv.CreatedBy, inv.ExpiresAt, inv.CreatedAt,
	); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// CountSeatsInUse returns the tenant's booked seats and how many are taken. An
// active user and a pending, unexpired invitation each take one — an expired
// invitation does not, so a seat comes back on its own once the invite lapses.
// A nil limit means the tenant has no seat cap.
func (r *PostgresRepository) CountSeatsInUse(ctx context.Context, tenantID uuid.UUID, excludeInvitation *uuid.UUID) (used int, limit *int, err error) {
	err = r.pool.QueryRow(ctx,
		`SELECT t.seat_limit,
		        (SELECT COUNT(*) FROM users u
		          WHERE u.tenant_id = t.id AND u.is_active)
		      + (SELECT COUNT(*) FROM invitations i
		          WHERE i.tenant_id = t.id
		            AND i.accepted_at IS NULL
		            AND i.expires_at > NOW()
		            AND ($2::uuid IS NULL OR i.id <> $2))
		 FROM tenants t WHERE t.id = $1`, tenantID, excludeInvitation,
	).Scan(&limit, &used)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil, ErrTenantNotFound
	}
	return used, limit, err
}

func (r *PostgresRepository) DeleteInvitation(ctx context.Context, tenantID, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM invitations WHERE id = $1 AND tenant_id = $2`, id, tenantID,
	)
	return err
}

// Two-factor authentication methods

func (r *PostgresRepository) StorePending2FASecret(ctx context.Context, userID uuid.UUID, encryptedSecret string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET two_factor_pending_secret = $1, updated_at = NOW() WHERE id = $2`,
		encryptedSecret, userID,
	)
	return err
}

func (r *PostgresRepository) GetPending2FASecret(ctx context.Context, userID uuid.UUID) (string, error) {
	var secret string
	err := r.pool.QueryRow(ctx,
		`SELECT two_factor_pending_secret FROM users WHERE id = $1`, userID,
	).Scan(&secret)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrUserNotFound
	}
	if err != nil {
		return "", err
	}
	if secret == "" {
		return "", ErrNo2FASetupPending
	}
	return secret, nil
}

func (r *PostgresRepository) Enable2FA(ctx context.Context, userID uuid.UUID, encryptedSecret string, recoveryCodes []*models.RecoveryCode) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Update user: enable 2FA, move secret from pending to permanent, clear pending
	_, err = tx.Exec(ctx,
		`UPDATE users SET
			two_factor_enabled = true,
			two_factor_secret_encrypted = $1,
			two_factor_pending_secret = '',
			two_factor_enabled_at = NOW(),
			updated_at = NOW()
		 WHERE id = $2`,
		encryptedSecret, userID,
	)
	if err != nil {
		return err
	}

	// Delete any existing recovery codes for this user
	_, err = tx.Exec(ctx,
		`DELETE FROM recovery_codes WHERE user_id = $1`, userID,
	)
	if err != nil {
		return err
	}

	// Insert new recovery codes
	for _, code := range recoveryCodes {
		_, err = tx.Exec(ctx,
			`INSERT INTO recovery_codes (id, tenant_id, user_id, code_hash, created_at)
			 VALUES ($1, $2, $3, $4, $5)`,
			code.ID, code.TenantID, code.UserID, code.CodeHash, code.CreatedAt,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepository) Disable2FA(ctx context.Context, userID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Disable 2FA on user record
	_, err = tx.Exec(ctx,
		`UPDATE users SET
			two_factor_enabled = false,
			two_factor_secret_encrypted = '',
			two_factor_pending_secret = '',
			two_factor_enabled_at = NULL,
			updated_at = NOW()
		 WHERE id = $1`,
		userID,
	)
	if err != nil {
		return err
	}

	// Delete all recovery codes
	_, err = tx.Exec(ctx,
		`DELETE FROM recovery_codes WHERE user_id = $1`, userID,
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepository) GetRecoveryCodes(ctx context.Context, userID uuid.UUID) ([]*models.RecoveryCode, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, code_hash, used_at, created_at
		 FROM recovery_codes WHERE user_id = $1
		 ORDER BY created_at`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var codes []*models.RecoveryCode
	for rows.Next() {
		var c models.RecoveryCode
		if err := rows.Scan(&c.ID, &c.UserID, &c.CodeHash, &c.UsedAt, &c.CreatedAt); err != nil {
			return nil, err
		}
		codes = append(codes, &c)
	}
	return codes, rows.Err()
}

func (r *PostgresRepository) UseRecoveryCode(ctx context.Context, codeID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE recovery_codes SET used_at = NOW() WHERE id = $1`, codeID,
	)
	return err
}

func (r *PostgresRepository) ReplaceRecoveryCodes(ctx context.Context, userID uuid.UUID, codes []*models.RecoveryCode) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Delete old codes
	_, err = tx.Exec(ctx,
		`DELETE FROM recovery_codes WHERE user_id = $1`, userID,
	)
	if err != nil {
		return err
	}

	// Insert new codes
	for _, code := range codes {
		_, err = tx.Exec(ctx,
			`INSERT INTO recovery_codes (id, tenant_id, user_id, code_hash, created_at)
			 VALUES ($1, $2, $3, $4, $5)`,
			code.ID, code.TenantID, code.UserID, code.CodeHash, code.CreatedAt,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// Password reset token methods

func (r *PostgresRepository) CreatePasswordResetToken(ctx context.Context, token *models.PasswordResetToken) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO password_reset_tokens (id, tenant_id, user_id, token_hash, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		token.ID, token.TenantID, token.UserID, token.TokenHash, token.ExpiresAt, token.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) GetPasswordResetToken(ctx context.Context, tokenHash string) (*models.PasswordResetToken, error) {
	var t models.PasswordResetToken
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, user_id, token_hash, expires_at, used_at, created_at
		 FROM password_reset_tokens WHERE token_hash = $1`, tokenHash,
	).Scan(&t.ID, &t.TenantID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.UsedAt, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrResetTokenInvalid
	}
	return &t, err
}

func (r *PostgresRepository) MarkPasswordResetTokenUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE password_reset_tokens SET used_at = NOW() WHERE id = $1`, id,
	)
	return err
}

// Two-factor policy methods

func (r *PostgresRepository) GetTwoFactorPolicy(ctx context.Context, roleName string) (*models.TwoFactorPolicy, error) {
	var p models.TwoFactorPolicy
	err := r.pool.QueryRow(ctx,
		`SELECT id, role_name, enforced, grace_period_days, updated_at, updated_by
		 FROM two_factor_policy WHERE role_name = $1`, roleName,
	).Scan(&p.ID, &p.RoleName, &p.Enforced, &p.GracePeriodDays, &p.UpdatedAt, &p.UpdatedBy)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &p, err
}

func (r *PostgresRepository) ListTwoFactorPolicies(ctx context.Context) ([]*models.TwoFactorPolicy, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, role_name, enforced, grace_period_days, updated_at, updated_by
		 FROM two_factor_policy ORDER BY role_name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []*models.TwoFactorPolicy
	for rows.Next() {
		var p models.TwoFactorPolicy
		if err := rows.Scan(&p.ID, &p.RoleName, &p.Enforced, &p.GracePeriodDays, &p.UpdatedAt, &p.UpdatedBy); err != nil {
			return nil, err
		}
		policies = append(policies, &p)
	}
	return policies, rows.Err()
}

func (r *PostgresRepository) UpsertTwoFactorPolicy(ctx context.Context, policy *models.TwoFactorPolicy) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO two_factor_policy (id, role_name, enforced, grace_period_days, updated_at, updated_by)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (role_name) DO UPDATE SET
			enforced = EXCLUDED.enforced,
			grace_period_days = EXCLUDED.grace_period_days,
			updated_at = EXCLUDED.updated_at,
			updated_by = EXCLUDED.updated_by`,
		policy.ID, policy.RoleName, policy.Enforced, policy.GracePeriodDays, policy.UpdatedAt, policy.UpdatedBy,
	)
	return err
}

// Session management methods

func (r *PostgresRepository) CreateSession(ctx context.Context, session *models.UserSession) error {
	_, err := r.pool.Exec(ctx,
		// ip_address is INET: binding the empty string fails the whole insert
		// with SQLSTATE 22P02, so an unknown address has to become SQL NULL.
		// A login must not fail because the peer address could not be read.
		`INSERT INTO user_sessions (id, tenant_id, user_id, refresh_token_id, device_name, device_type, ip_address, location, user_agent, last_active_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, '')::inet, $8, $9, $10, $11)`,
		session.ID, session.TenantID, session.UserID, session.RefreshTokenID,
		session.DeviceName, session.DeviceType, session.IPAddress,
		session.Location, session.UserAgent,
		session.LastActiveAt, session.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) GetSession(ctx context.Context, id uuid.UUID) (*models.UserSession, error) {
	var s models.UserSession
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, refresh_token_id, device_name, device_type, COALESCE(host(ip_address), ''), location, user_agent, last_active_at, created_at
		 FROM user_sessions WHERE id = $1`, id,
	).Scan(&s.ID, &s.UserID, &s.RefreshTokenID, &s.DeviceName, &s.DeviceType,
		&s.IPAddress, &s.Location, &s.UserAgent, &s.LastActiveAt, &s.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSessionNotFound
	}
	return &s, err
}

func (r *PostgresRepository) ListUserSessions(ctx context.Context, userID uuid.UUID) ([]*models.UserSession, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, refresh_token_id, device_name, device_type, COALESCE(host(ip_address), ''), location, user_agent, last_active_at, created_at
		 FROM user_sessions WHERE user_id = $1
		 ORDER BY last_active_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*models.UserSession
	for rows.Next() {
		var s models.UserSession
		if err := rows.Scan(&s.ID, &s.UserID, &s.RefreshTokenID, &s.DeviceName, &s.DeviceType,
			&s.IPAddress, &s.Location, &s.UserAgent, &s.LastActiveAt, &s.CreatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, &s)
	}
	return sessions, rows.Err()
}

func (r *PostgresRepository) ListAllSessions(ctx context.Context, offset, limit int) ([]*models.UserSession, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM user_sessions").Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, refresh_token_id, device_name, device_type, COALESCE(host(ip_address), ''), location, user_agent, last_active_at, created_at
		 FROM user_sessions
		 ORDER BY last_active_at DESC
		 LIMIT $1 OFFSET $2`, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var sessions []*models.UserSession
	for rows.Next() {
		var s models.UserSession
		if err := rows.Scan(&s.ID, &s.UserID, &s.RefreshTokenID, &s.DeviceName, &s.DeviceType,
			&s.IPAddress, &s.Location, &s.UserAgent, &s.LastActiveAt, &s.CreatedAt); err != nil {
			return nil, 0, err
		}
		sessions = append(sessions, &s)
	}
	return sessions, total, rows.Err()
}

func (r *PostgresRepository) UpdateSessionActivity(ctx context.Context, sessionID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE user_sessions SET last_active_at = NOW() WHERE id = $1`, sessionID,
	)
	return err
}

func (r *PostgresRepository) DeleteSession(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM user_sessions WHERE id = $1`, id,
	)
	return err
}

func (r *PostgresRepository) DeleteAllUserSessions(ctx context.Context, userID uuid.UUID, exceptSessionID *uuid.UUID) error {
	if exceptSessionID != nil {
		_, err := r.pool.Exec(ctx,
			`DELETE FROM user_sessions WHERE user_id = $1 AND id != $2`,
			userID, *exceptSessionID,
		)
		return err
	}
	_, err := r.pool.Exec(ctx,
		`DELETE FROM user_sessions WHERE user_id = $1`, userID,
	)
	return err
}

func (r *PostgresRepository) RotateSessionRefreshToken(ctx context.Context, oldTokenID, newTokenID uuid.UUID, ipAddress, userAgent string) (bool, error) {
	// COALESCE keeps the previously recorded device when the caller has no
	// value this time round — an empty user agent is missing information, not
	// the statement that the device changed.
	tag, err := r.pool.Exec(ctx,
		`UPDATE user_sessions
		    SET refresh_token_id = $2,
		        ip_address = COALESCE(NULLIF($3, '')::inet, ip_address),
		        user_agent = COALESCE(NULLIF($4, ''), user_agent),
		        last_active_at = NOW()
		  WHERE refresh_token_id = $1`,
		oldTokenID, newTokenID, ipAddress, userAgent,
	)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (r *PostgresRepository) DeleteSessionByRefreshTokenID(ctx context.Context, refreshTokenID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM user_sessions WHERE refresh_token_id = $1`, refreshTokenID,
	)
	return err
}

func (r *PostgresRepository) DeleteStaleUserSessions(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM user_sessions
		  WHERE user_id = $1
		    AND (refresh_token_id IS NULL
		         OR refresh_token_id IN (
		             SELECT id FROM refresh_tokens
		              WHERE revoked = TRUE OR expires_at < NOW()
		         ))`,
		userID,
	)
	return err
}

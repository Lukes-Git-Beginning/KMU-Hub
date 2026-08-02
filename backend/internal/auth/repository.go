package auth

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

type Repository interface {
	CreateUser(ctx context.Context, user *models.User) error
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	UpdateUser(ctx context.Context, user *models.User) error
	UpdateProfile(ctx context.Context, user *models.User) error
	UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error
	ListUsers(ctx context.Context, offset, limit int) ([]*models.User, int, error)

	StoreRefreshToken(ctx context.Context, token *models.RefreshToken) error
	GetRefreshTokenByHash(ctx context.Context, hash string) (*models.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, id uuid.UUID) error
	RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error

	AssignRole(ctx context.Context, userID uuid.UUID, roleName string) error
	RemoveRole(ctx context.Context, userID uuid.UUID, roleName string) error
	GetUserRoles(ctx context.Context, userID uuid.UUID) ([]string, error)
	GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]string, error)
	UserHasPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error)
	// GetEffectivePermissions returns one row per (role, fine-grained
	// capability) pair; the union across roles happens in the service.
	GetEffectivePermissions(ctx context.Context, userID uuid.UUID) ([]EffectiveGrantRow, error)

	// ListRoles returns the system presets plus the calling tenant's custom
	// roles, relying entirely on the roles table's RLS read policy for the
	// tenant scoping.
	ListRoles(ctx context.Context) ([]Role, error)

	// CountCustomRoles counts the roles owned by the calling tenant (presets
	// excluded) for the CustomRoleLimit check.
	CountCustomRoles(ctx context.Context) (int, error)

	// RoleNameExists reports whether a role visible to the caller — a preset
	// or one of its own — already carries this name, compared
	// case-insensitively. exceptID skips one role so a rename onto its own
	// name is not a collision; pass uuid.Nil when creating.
	RoleNameExists(ctx context.Context, name string, exceptID uuid.UUID) (bool, error)

	// CreateRole inserts a tenant-owned role and clones every grant of
	// in.BasedOn onto it in one transaction.
	CreateRole(ctx context.Context, tenantID uuid.UUID, in CreateRoleInput) (*Role, error)

	// GetRoleByID resolves a single role through the roles table's RLS read
	// policy — a preset or the caller's own tenant. A foreign tenant's role is
	// indistinguishable from an unknown id, both ErrBaseRoleNotFound.
	GetRoleByID(ctx context.Context, id uuid.UUID) (*Role, error)

	// UpdateRole applies the provided fields (nil = unchanged) to a
	// tenant-owned role and returns it with fresh member/capability counts.
	// Callers must reject presets first: the write policy confines the
	// UPDATE to the caller's own tenant, so a preset id simply touches zero
	// rows here.
	UpdateRole(ctx context.Context, roleID uuid.UUID, in UpdateRoleInput) (*Role, error)

	// RoleHasMembers reports whether any account currently carries roleID,
	// scoped to the calling tenant through the users join — user_roles itself
	// carries neither tenant_id nor RLS.
	RoleHasMembers(ctx context.Context, roleID uuid.UUID) (bool, error)

	// DeleteRole removes a tenant-owned role. role_permissions and user_roles
	// cascade on role_id.
	DeleteRole(ctx context.Context, roleID uuid.UUID) error

	// Invitation methods. Everything but the token lookup is tenant-scoped:
	// the token lookup is the one call whose caller has no tenant yet.
	CreateInvitation(ctx context.Context, inv *models.Invitation) error
	GetInvitationByToken(ctx context.Context, tokenHash string) (*models.Invitation, error)
	// GetPendingInvitationByEmail looks the address up across tenants and is
	// therefore only meaningful under system context — provisioning uses it to
	// refuse an address that is already invited somewhere else.
	GetPendingInvitationByEmail(ctx context.Context, email string) (*models.Invitation, error)
	GetInvitationByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Invitation, error)
	ListPendingInvitations(ctx context.Context, tenantID uuid.UUID) ([]*models.Invitation, error)
	AcceptInvitation(ctx context.Context, inv *models.Invitation, user *models.User) error
	CountSeatsInUse(ctx context.Context, tenantID uuid.UUID, excludeInvitation *uuid.UUID) (used int, limit *int, err error)
	DeleteInvitation(ctx context.Context, tenantID, id uuid.UUID) error

	// ProvisionTenant writes the tenant, its module activations and the first
	// administrator's invitation in one transaction. A partial failure must
	// leave nothing behind — a tenant nobody can log into is worse than none.
	ProvisionTenant(ctx context.Context, tenant *models.Tenant, moduleIDs []string, inv *models.Invitation) error

	// Two-factor authentication methods
	StorePending2FASecret(ctx context.Context, userID uuid.UUID, encryptedSecret string) error
	GetPending2FASecret(ctx context.Context, userID uuid.UUID) (string, error)
	Enable2FA(ctx context.Context, userID uuid.UUID, encryptedSecret string, recoveryCodes []*models.RecoveryCode) error
	Disable2FA(ctx context.Context, userID uuid.UUID) error
	GetRecoveryCodes(ctx context.Context, userID uuid.UUID) ([]*models.RecoveryCode, error)
	UseRecoveryCode(ctx context.Context, codeID uuid.UUID) error
	ReplaceRecoveryCodes(ctx context.Context, userID uuid.UUID, codes []*models.RecoveryCode) error

	// Two-factor policy methods
	GetTwoFactorPolicy(ctx context.Context, roleName string) (*models.TwoFactorPolicy, error)
	ListTwoFactorPolicies(ctx context.Context) ([]*models.TwoFactorPolicy, error)
	UpsertTwoFactorPolicy(ctx context.Context, policy *models.TwoFactorPolicy) error

	// Password reset token methods
	CreatePasswordResetToken(ctx context.Context, token *models.PasswordResetToken) error
	GetPasswordResetToken(ctx context.Context, tokenHash string) (*models.PasswordResetToken, error)
	MarkPasswordResetTokenUsed(ctx context.Context, id uuid.UUID) error

	// Session management methods
	CreateSession(ctx context.Context, session *models.UserSession) error
	GetSession(ctx context.Context, id uuid.UUID) (*models.UserSession, error)
	ListUserSessions(ctx context.Context, userID uuid.UUID) ([]*models.UserSession, error)
	ListAllSessions(ctx context.Context, offset, limit int) ([]*models.UserSession, int, error)
	UpdateSessionActivity(ctx context.Context, sessionID uuid.UUID) error
	DeleteSession(ctx context.Context, id uuid.UUID) error
	DeleteAllUserSessions(ctx context.Context, userID uuid.UUID, exceptSessionID *uuid.UUID) error
	// RotateSessionRefreshToken re-points the session behind oldTokenID at
	// newTokenID and refreshes its device metadata. Reports whether a session
	// was found: a refresh whose session predates session tracking has none,
	// and the caller then creates one instead of losing the device entry.
	RotateSessionRefreshToken(ctx context.Context, oldTokenID, newTokenID uuid.UUID, ipAddress, userAgent string) (bool, error)
	DeleteSessionByRefreshTokenID(ctx context.Context, refreshTokenID uuid.UUID) error
	// DeleteStaleUserSessions removes the user's sessions whose refresh token
	// is gone, revoked or expired — those can never be reached again, and the
	// row carries personal data (IP, user agent) worth not keeping.
	DeleteStaleUserSessions(ctx context.Context, userID uuid.UUID) error
}

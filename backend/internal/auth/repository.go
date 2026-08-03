package auth

import (
	"context"
	"time"

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

	// GetUserGrants returns the same rows as GetEffectivePermissions without
	// the catalogue filter — the coarse legacy keys ("files:write") included.
	// The guardrails need that wider set: a grant set may name any permission
	// the catalogue has, and the coarse keys are the currency of the
	// RequirePermission gates, so leaving them out of the escalation check
	// would let a caller hand out exactly the rights those gates protect.
	GetUserGrants(ctx context.Context, userID uuid.UUID) ([]EffectiveGrantRow, error)

	// CountUnknownPermissionKeys reports how many of keys the permissions
	// catalogue does not contain. SetRolePermissions repeats the same check
	// inside its transaction as a backstop; this one exists so the service can
	// answer a typo with 422 before the escalation check turns it into a 403 —
	// a key that does not exist is never within anyone's reach.
	CountUnknownPermissionKeys(ctx context.Context, keys []string) (int, error)

	// CountRoleAdminsExcluding counts the active accounts of the calling
	// tenant that hold at least one of keys, ignoring the single assignment
	// (ignoreUserID, ignoreRoleID) — "who would still be able to administer
	// roles if this revoke went through". Tenant scoping comes from the users
	// join, since user_roles carries neither tenant_id nor RLS.
	CountRoleAdminsExcluding(ctx context.Context, keys []string, ignoreUserID, ignoreRoleID uuid.UUID) (int, error)

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

	// GetRolePermissions returns the grant set of roleID. role_permissions'
	// own RLS read policy (derived through the owning role) does the tenant
	// scoping; callers must first resolve the role through GetRoleByID to
	// tell "role invisible" apart from "role has zero grants".
	GetRolePermissions(ctx context.Context, roleID uuid.UUID) ([]RoleGrant, error)

	// SetRolePermissions replaces the entire grant set of roleID in one
	// transaction — delete then insert, so a failure midway leaves the
	// previous grants standing. Every key must resolve against the
	// permissions catalogue; an unresolved key aborts the whole write with
	// ErrCapabilityKeyUnknown. Callers must reject presets first.
	SetRolePermissions(ctx context.Context, roleID uuid.UUID, grants []RoleGrant) ([]RoleGrant, error)

	// AssignUserRole grants roleID to userID, idempotently. Both ids are
	// re-resolved through users and roles inside the statement: user_roles
	// carries neither tenant_id nor RLS, so those two joins are the tenant
	// boundary of the write, not a convenience.
	AssignUserRole(ctx context.Context, userID, roleID uuid.UUID) error

	// RevokeUserRole takes roleID off userID. Removing a role the account does
	// not hold is a no-op, not an error — the caller asked for a state, not for
	// a transition.
	RevokeUserRole(ctx context.Context, userID, roleID uuid.UUID) error

	// GetUserRoleIDs returns the role ids an account holds, oldest assignment
	// first (the frontend renders them in assignment order). Scoped to the
	// calling tenant through the users join, for the same reason as
	// AssignUserRole.
	GetUserRoleIDs(ctx context.Context, userID uuid.UUID) ([]string, error)

	// GetUserOverrides returns the per-user permission overrides of an
	// account, ordered by key. user_permission_overrides carries tenant_id and
	// an RLS policy, so this needs no tenant filter of its own.
	GetUserOverrides(ctx context.Context, userID uuid.UUID) ([]CapabilityOverride, error)

	// GetUserOverridesForTenant is the same read for callers that run under
	// sysctx.With(), where RLS does not filter and the tenant has to be named.
	GetUserOverridesForTenant(ctx context.Context, tenantID, userID uuid.UUID) ([]CapabilityOverride, error)

	// SetUserOverrides replaces the whole override map of an account in one
	// transaction: everything currently stored is deleted, the given list is
	// inserted. An empty list therefore clears the account. createdBy is
	// recorded per row so the personnel-style question "who gave them this"
	// survives even after the audit log is rotated.
	SetUserOverrides(ctx context.Context, tenantID, createdBy, userID uuid.UUID, overrides []CapabilityOverride) ([]CapabilityOverride, error)

	// ClearUserOverrides drops every override of an account. Clearing an
	// account that has none is a no-op, not an error — same reasoning as
	// RevokeUserRole.
	ClearUserOverrides(ctx context.Context, userID uuid.UUID) error

	// CountEffectiveRoleAdminsExcluding counts the active accounts of the
	// calling tenant, excluding excludeUserID, that hold at least one of keys
	// AFTER their own overrides are applied — a deny takes the key away, an
	// allow hands it over even without a role that grants it.
	//
	// It is the override-aware sibling of CountRoleAdminsExcluding, which
	// looks at roles alone. The two stay separate because they answer
	// different questions: that one asks what a role revoke would leave
	// behind, this one what an override would.
	CountEffectiveRoleAdminsExcluding(ctx context.Context, keys []string, excludeUserID uuid.UUID) (int, error)

	// ListAdminUsers returns the tenant's account roster: every real account
	// (users, RLS-scoped) plus every still-open invitation (invitations,
	// RLS-scoped), merged into the one list the account admin surface shows.
	// Roles and last-login load once each, not once per account.
	ListAdminUsers(ctx context.Context) ([]AdminUser, error)

	// GetAdminUser and GetInvitationAsAdminUser return a single roster row —
	// the answer the three writing roster routes give, in the shape the list
	// returns. GetAdminUser answers ErrUserNotFound and
	// GetInvitationAsAdminUser ErrInvitationNotFound for a row RLS hides,
	// since a foreign row is invisible rather than forbidden.
	GetAdminUser(ctx context.Context, id uuid.UUID) (*AdminUser, error)
	GetInvitationAsAdminUser(ctx context.Context, id uuid.UUID) (*AdminUser, error)

	// CountActiveRoleAdminsExcludingUser counts the tenant's active accounts
	// carrying role administration, ignoring one account whole. Unlike
	// CountRoleAdminsExcluding (which ignores one user/role PAIR) this is what
	// "may this account be deactivated" needs.
	CountActiveRoleAdminsExcludingUser(ctx context.Context, keys []string, ignoreUserID uuid.UUID) (int, error)

	// GetPresetRoleIDByName resolves a legacy preset name to its role id.
	// Presets only — a name does not identify a custom role uniquely.
	GetPresetRoleIDByName(ctx context.Context, name string) (uuid.UUID, error)

	// Invitation methods. Everything but the token lookup is tenant-scoped:
	// the token lookup is the one call whose caller has no tenant yet.
	CreateInvitation(ctx context.Context, inv *models.Invitation) error
	GetInvitationByToken(ctx context.Context, tokenHash string) (*models.Invitation, error)
	// GetPendingInvitationByEmail looks the address up across tenants and is
	// therefore only meaningful under system context — provisioning uses it to
	// refuse an address that is already invited somewhere else.
	GetPendingInvitationByEmail(ctx context.Context, email string) (*models.Invitation, error)
	GetInvitationByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Invitation, error)
	// RefreshInvitationToken re-issues a still-pending invitation with a new
	// token hash and expiry, invalidating the old token by overwriting it.
	// Answers ErrInvitationNotFound when the row is gone or already accepted.
	RefreshInvitationToken(ctx context.Context, tenantID, id uuid.UUID, tokenHash string, expiresAt time.Time) (*models.Invitation, error)
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

	// Two-factor policy methods. The tenant is a parameter rather than a
	// context read because the login path queries the policy under
	// sysctx.With, where RLS admits every tenant's rows.
	GetTwoFactorPolicy(ctx context.Context, tenantID uuid.UUID, roleName string) (*models.TwoFactorPolicy, error)
	ListTwoFactorPolicies(ctx context.Context, tenantID uuid.UUID) ([]*models.TwoFactorPolicy, error)
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

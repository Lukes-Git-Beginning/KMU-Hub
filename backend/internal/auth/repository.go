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

	// Invitation methods. Everything but the token lookup is tenant-scoped:
	// the token lookup is the one call whose caller has no tenant yet.
	CreateInvitation(ctx context.Context, inv *models.Invitation) error
	GetInvitationByToken(ctx context.Context, tokenHash string) (*models.Invitation, error)
	GetInvitationByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Invitation, error)
	ListPendingInvitations(ctx context.Context, tenantID uuid.UUID) ([]*models.Invitation, error)
	AcceptInvitation(ctx context.Context, inv *models.Invitation, user *models.User) error
	CountSeatsInUse(ctx context.Context, tenantID uuid.UUID, excludeInvitation *uuid.UUID) (used int, limit *int, err error)
	DeleteInvitation(ctx context.Context, tenantID, id uuid.UUID) error

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
}

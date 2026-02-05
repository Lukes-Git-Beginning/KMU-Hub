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

	// Invitation methods
	CreateInvitation(ctx context.Context, inv *models.Invitation) error
	GetInvitationByToken(ctx context.Context, tokenHash string) (*models.Invitation, error)
	GetInvitationByID(ctx context.Context, id uuid.UUID) (*models.Invitation, error)
	ListPendingInvitations(ctx context.Context) ([]*models.Invitation, error)
	MarkInvitationAccepted(ctx context.Context, id uuid.UUID) error
	DeleteInvitation(ctx context.Context, id uuid.UUID) error
}

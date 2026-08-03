package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// AdminUserStatus mirrors AdminUserStatus in admin-types.ts.
type AdminUserStatus string

const (
	AdminUserStatusActive      AdminUserStatus = "active"
	AdminUserStatusInvited     AdminUserStatus = "invited"
	AdminUserStatusDeactivated AdminUserStatus = "deactivated"
)

// AdminUser is one row of the tenant's account roster (admin-types.ts,
// AdminUser), merged from two sources that can never overlap for the same
// email: users (real accounts, active/deactivated) and invitations (still
// pending — no user row exists yet, so FirstName/LastName are empty and
// RoleIDs holds at most the one preset the invite named).
type AdminUser struct {
	ID          uuid.UUID
	FirstName   string
	LastName    string
	Email       string
	RoleIDs     []string
	Status      AdminUserStatus
	LastLoginAt *time.Time
	InvitedAt   *time.Time
}

// ListAdminUsers returns the calling tenant's account roster. Both sources
// are RLS-scoped (users, invitations), so no tenant id travels as a
// parameter — the same shape as ListUsers and ListRoles.
func (s *Service) ListAdminUsers(ctx context.Context) ([]AdminUser, error) {
	return s.repo.ListAdminUsers(ctx)
}

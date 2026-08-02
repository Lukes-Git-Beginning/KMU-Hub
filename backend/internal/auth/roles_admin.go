package auth

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

// CustomRoleLimit caps how many roles a tenant may own. It mirrors
// CUSTOM_ROLE_LIMIT in the frontend (mocks/data/rbac.ts) and counts only
// tenant-owned roles — the shared system presets never eat into it.
const CustomRoleLimit = 20

// Role is a role as the admin surface sees it: either a Zentria-maintained
// system preset (TenantID nil, shared by every tenant, immutable) or a
// tenant-owned custom role. BasedOn names the preset a custom role was cloned
// from and is nil on presets themselves.
type Role struct {
	ID              uuid.UUID
	Name            string
	Description     string
	TenantID        *uuid.UUID
	BasedOn         *uuid.UUID
	Color           string
	IsSystem        bool
	MemberCount     int
	CapabilityCount int
}

// ListRoles returns the system presets together with the custom roles of the
// calling tenant — exactly the rows the roles table's RLS read policy admits,
// so no tenant filter is needed here.
func (s *Service) ListRoles(ctx context.Context) ([]Role, error) {
	return s.repo.ListRoles(ctx)
}

// CreateRoleInput describes a new tenant-owned role. BasedOn is mandatory:
// creating a role is always a clone, never a blank slate.
type CreateRoleInput struct {
	Name        string
	Description string
	Color       string
	BasedOn     uuid.UUID
}

// CreateRole clones BasedOn into a new role owned by tenantID, copying every
// grant of the source role including its scope. The tenant arrives as a
// parameter rather than being read from the context because this package must
// not import middleware (import cycle) — the gRPC layer resolves it, exactly
// like CreateInvitation.
//
// The three business rules live here rather than in the repository: the custom
// role budget, the name collision, and the requirement that the clone source
// is visible to the caller. The name check is deliberately case-insensitive
// and also covers the presets, matching the frontend contract — the unique
// index cannot do either (it is case-sensitive, and presets sit in their own
// COALESCE bucket, so "Admin" beside the "admin" preset would pass it).
// The repository still maps a unique violation back onto ErrRoleNameExists as
// a backstop for the race between this check and the insert.
func (s *Service) CreateRole(ctx context.Context, tenantID uuid.UUID, in CreateRoleInput) (*Role, error) {
	name := strings.TrimSpace(in.Name)

	count, err := s.repo.CountCustomRoles(ctx)
	if err != nil {
		return nil, err
	}
	if count >= CustomRoleLimit {
		return nil, ErrRoleLimitReached
	}

	exists, err := s.repo.RoleNameExists(ctx, name, uuid.Nil)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrRoleNameExists
	}

	return s.repo.CreateRole(ctx, tenantID, CreateRoleInput{
		Name:        name,
		Description: strings.TrimSpace(in.Description),
		Color:       in.Color,
		BasedOn:     in.BasedOn,
	})
}

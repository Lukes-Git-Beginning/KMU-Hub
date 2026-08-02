package auth

import (
	"context"

	"github.com/google/uuid"
)

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

package models

import "github.com/google/uuid"

// DefaultTenantID is the bootstrap tenant created by migration 000114.
// Users registered via public sign-up or accepted invitations land here
// until multi-tenant provisioning is wired in (Sprint 5+).
var DefaultTenantID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

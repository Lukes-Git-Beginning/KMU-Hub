package models

import (
	"time"

	"github.com/google/uuid"
)

type Invitation struct {
	ID         uuid.UUID  `json:"id"`
	TenantID   uuid.UUID  `json:"tenant_id"`
	Email      string     `json:"email"`
	FirstName  string     `json:"first_name"`
	LastName   string     `json:"last_name"`
	// Role is the legacy preset name. Display only since migration 000280 —
	// what an accepted invitation actually grants is RoleIDs.
	Role    string      `json:"role"`
	RoleIDs []uuid.UUID `json:"role_ids"`
	TokenHash  string     `json:"-"`
	CreatedBy  uuid.UUID  `json:"created_by"`
	ExpiresAt  time.Time  `json:"expires_at"`
	AcceptedAt *time.Time `json:"accepted_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

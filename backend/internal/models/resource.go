package models

import (
	"time"

	"github.com/google/uuid"
)

// Resource type constants
const (
	ResourceTypeRoom      = "room"
	ResourceTypeEquipment = "equipment"
	ResourceTypeVehicle   = "vehicle"
)

// ValidResourceTypes contains all valid resource types
var ValidResourceTypes = map[string]bool{
	ResourceTypeRoom:      true,
	ResourceTypeEquipment: true,
	ResourceTypeVehicle:   true,
}

// Resource represents a bookable resource (room, equipment, vehicle)
type Resource struct {
	ID           uuid.UUID `json:"id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	Name         string    `json:"name"`
	ResourceType string    `json:"resource_type"`
	Capacity     *int      `json:"capacity,omitempty"`
	Floor        *string   `json:"floor,omitempty"`
	Location     *string   `json:"location,omitempty"`
	Description  *string   `json:"description,omitempty"`
	IsActive     bool      `json:"is_active"`
	CreatedBy    uuid.UUID `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Tags         []string  `json:"tags,omitempty"` // populated from resource_tags table
}

// ResourceBooking represents a time-bounded reservation of a resource
type ResourceBooking struct {
	ID           uuid.UUID  `json:"id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	ResourceID   uuid.UUID  `json:"resource_id"`
	EventID      uuid.UUID  `json:"event_id"`
	BookedBy     uuid.UUID  `json:"booked_by"`
	StartTime    time.Time  `json:"start_time"`
	EndTime      time.Time  `json:"end_time"`
	CancelledAt  *time.Time `json:"cancelled_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	ResourceName string     `json:"resource_name"` // denormalized for display
	EventTitle   string     `json:"event_title"`   // denormalized for display
}

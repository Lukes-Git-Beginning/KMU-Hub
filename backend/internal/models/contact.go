package models

import (
	"time"

	"github.com/google/uuid"
)

// Contact represents a CRM contact
type Contact struct {
	ID           uuid.UUID  `json:"id"`
	FirstName    string     `json:"first_name"`
	LastName     string     `json:"last_name"`
	Email        *string    `json:"email,omitempty"`
	Phone        *string    `json:"phone,omitempty"`
	CompanyID    *uuid.UUID `json:"company_id,omitempty"`
	Position     *string    `json:"position,omitempty"`
	Notes        *string    `json:"notes,omitempty"`
	Visibility   string     `json:"visibility"` // shared, personal (default: shared)
	OwnerID      *uuid.UUID `json:"owner_id,omitempty"`
	MergedIntoID *uuid.UUID `json:"merged_into_id,omitempty"`
	CreatedBy    uuid.UUID  `json:"created_by"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// ContactWithRelations includes associated data for API responses
type ContactWithRelations struct {
	Contact
	CompanyName  *string        `json:"company_name,omitempty"`
	Tags         []*Tag         `json:"tags,omitempty"`
	CustomFields map[string]any `json:"custom_fields,omitempty"` // field_name -> value
}

// CustomFieldValueRow represents a custom field value from DB query
type CustomFieldValueRow struct {
	FieldID   uuid.UUID `json:"field_id"`
	FieldName string    `json:"field_name"`
	Value     any       `json:"value"`
}

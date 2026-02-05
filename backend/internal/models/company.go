package models

import (
	"time"

	"github.com/google/uuid"
)

// Company represents a CRM company
type Company struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Domain        *string   `json:"domain,omitempty"`
	Industry      *string   `json:"industry,omitempty"`
	EmployeeCount *int      `json:"employee_count,omitempty"`
	Address       *string   `json:"address,omitempty"`
	City          *string   `json:"city,omitempty"`
	Country       *string   `json:"country,omitempty"`
	Notes         *string   `json:"notes,omitempty"`
	CreatedBy     uuid.UUID `json:"created_by"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// CompanyWithRelations includes associated data for API responses
type CompanyWithRelations struct {
	Company
	ContactCount int            `json:"contact_count"`
	Tags         []*Tag         `json:"tags,omitempty"`
	CustomFields map[string]any `json:"custom_fields,omitempty"` // field_name -> value
}

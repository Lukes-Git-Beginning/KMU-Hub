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
	TenantID     uuid.UUID  `json:"tenant_id"`
	CreatedBy    uuid.UUID  `json:"created_by"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`

	// Lead lifecycle (migration 000259). A lead is this same row at an earlier
	// stage, never a separate table -- two rows for one person would break
	// duplicate detection. Contacts that never came through the lead inbox
	// carry LifecycleStage "customer" and NULL in every Lead* field.
	LifecycleStage string  `json:"lifecycle_stage"`
	LeadSource     *string `json:"lead_source,omitempty"` // manual, csv, dialer
	LeadScore      *int16  `json:"lead_score,omitempty"`  // 0-100, computed server-side
	// LeadTemperature is the manual override only; NULL means "derive from LeadScore".
	LeadTemperature *string `json:"lead_temperature,omitempty"` // hot, warm, cold
	LeadStatus      *string `json:"lead_status,omitempty"`      // new, contacted, qualified, disqualified
	// LeadCompany names the employer before a companies row exists for it.
	LeadCompany *string `json:"lead_company,omitempty"`
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

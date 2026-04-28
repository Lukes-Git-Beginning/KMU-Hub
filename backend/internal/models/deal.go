package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Deal represents a sales opportunity in the CRM
type Deal struct {
	ID                uuid.UUID       `json:"id"`
	TenantID          uuid.UUID       `json:"tenant_id"`
	Name              string          `json:"name"`
	Value             decimal.Decimal `json:"value"`
	Currency          string          `json:"currency"`
	StageID           uuid.UUID       `json:"stage_id"`
	ContactID         *uuid.UUID      `json:"contact_id,omitempty"`
	CompanyID         *uuid.UUID      `json:"company_id,omitempty"`
	OwnerID           *uuid.UUID      `json:"owner_id,omitempty"`
	ExpectedCloseDate *time.Time      `json:"expected_close_date,omitempty"`
	Notes             *string         `json:"notes,omitempty"`
	CreatedBy         uuid.UUID       `json:"created_by"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
	ClosedAt          *time.Time      `json:"closed_at,omitempty"`
}

// DealWithRelations includes associated data for API responses
type DealWithRelations struct {
	Deal
	StageName    string         `json:"stage_name"`
	ContactName  *string        `json:"contact_name,omitempty"`
	CompanyName  *string        `json:"company_name,omitempty"`
	OwnerName    *string        `json:"owner_name,omitempty"`
	Tags         []*Tag         `json:"tags,omitempty"`
	CustomFields map[string]any `json:"custom_fields,omitempty"` // field_name -> value
}

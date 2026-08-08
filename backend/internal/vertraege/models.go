package vertraege

import (
	"time"

	"github.com/google/uuid"
)

// ContractType classifies the kind of contract.
type ContractType string

const (
	ContractTypeRental     ContractType = "rental"
	ContractTypeService    ContractType = "service"
	ContractTypeEmployment ContractType = "employment"
	ContractTypeNDA        ContractType = "nda"
	ContractTypeOther      ContractType = "other"
)

// ContractStatus represents the lifecycle state of a contract.
type ContractStatus string

const (
	ContractStatusDraft      ContractStatus = "draft"
	ContractStatusActive     ContractStatus = "active"
	ContractStatusExpired    ContractStatus = "expired"
	ContractStatusTerminated ContractStatus = "terminated"
)

// PartyType identifies who the contracting party is.
type PartyType string

const (
	PartyTypeContact  PartyType = "contact"
	PartyTypeCompany  PartyType = "company"
	PartyTypeExternal PartyType = "external"
)

// ReminderType classifies the purpose of a reminder.
type ReminderType string

const (
	ReminderTypeRenewal  ReminderType = "renewal"
	ReminderTypeExpiry   ReminderType = "expiry"
	ReminderTypePayment  ReminderType = "payment"
	ReminderTypeCustom   ReminderType = "custom"
)

// ReminderStatus tracks whether a reminder has been sent.
type ReminderStatus string

const (
	ReminderStatusPending   ReminderStatus = "pending"
	ReminderStatusSent      ReminderStatus = "sent"
	ReminderStatusCancelled ReminderStatus = "cancelled"
)

// ContractEventAction names the kind of change recorded on a contract. The
// column is TEXT, so this constant set — not the database — is what keeps the
// vocabulary closed; never write a literal at a call site.
type ContractEventAction string

const (
	ContractEventCreated      ContractEventAction = "created"
	ContractEventUpdated      ContractEventAction = "updated"
	ContractEventTerminated   ContractEventAction = "terminated"
	ContractEventSigned       ContractEventAction = "signed"
	ContractEventPartyAdded   ContractEventAction = "party_added"
	ContractEventPartyRemoved ContractEventAction = "party_removed"
)

// ContractEvent is one entry of a contract's append-only audit trail.
// UserID is nil for changes made without a caller (reminder worker,
// auto-expiry) or after the acting account was deleted.
type ContractEvent struct {
	ID         uuid.UUID           `json:"id"`
	TenantID   uuid.UUID           `json:"tenant_id"`
	ContractID uuid.UUID           `json:"contract_id"`
	Action     ContractEventAction `json:"action"`
	UserID     *uuid.UUID          `json:"user_id,omitempty"`
	Payload    map[string]any      `json:"payload"`
	CreatedAt  time.Time           `json:"created_at"`
}

// Contract is the central aggregate for a Vertrag (contract).
type Contract struct {
	ID                uuid.UUID      `json:"id"`
	TenantID          uuid.UUID      `json:"tenant_id"`
	ContractNumber    string         `json:"contract_number"`
	Title             string         `json:"title"`
	ContractType      ContractType   `json:"contract_type"`
	Status            ContractStatus `json:"status"`
	StartsOn          time.Time      `json:"starts_on"`
	EndsOn            *time.Time     `json:"ends_on,omitempty"`          // nil = open-ended
	DocumentURL       *string        `json:"document_url,omitempty"`     // MinIO path
	Notes             string         `json:"notes"`
	CreatedBy         *uuid.UUID     `json:"created_by,omitempty"`
	SignatureProvider *string        `json:"signature_provider,omitempty"` // Phase D: Skribble
	SignatureData     *string        `json:"signature_data,omitempty"`     // EES inline: base64 PNG or SVG
	SignedAt          *time.Time     `json:"signed_at,omitempty"`
	SignedBy          *string        `json:"signed_by,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`

	// Populated only when GetContract is called
	Parties   []*ContractParty    `json:"parties,omitempty"`
	Reminders []*ContractReminder `json:"reminders,omitempty"`
}

// ContractParty represents one party (signatory / stakeholder) on a contract.
type ContractParty struct {
	ID              uuid.UUID  `json:"id"`
	TenantID        uuid.UUID  `json:"tenant_id"`
	ContractID      uuid.UUID  `json:"contract_id"`
	PartyType       PartyType  `json:"party_type"`
	ContactID       *uuid.UUID `json:"contact_id,omitempty"`
	CompanyID       *uuid.UUID `json:"company_id,omitempty"`
	ExternalName    *string    `json:"external_name,omitempty"`
	RoleInContract  string     `json:"role_in_contract"`
	SignedOn        *time.Time `json:"signed_on,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// ContractReminder is a time-based reminder attached to a contract.
type ContractReminder struct {
	ID           uuid.UUID      `json:"id"`
	TenantID     uuid.UUID      `json:"tenant_id"`
	ContractID   uuid.UUID      `json:"contract_id"`
	RemindAt     time.Time      `json:"remind_at"`
	ReminderType ReminderType   `json:"reminder_type"`
	Subject      string         `json:"subject"`
	Message      string         `json:"message"`
	Status       ReminderStatus `json:"status"`
	CreatedAt    time.Time      `json:"created_at"`
	SentAt       *time.Time     `json:"sent_at,omitempty"`
}

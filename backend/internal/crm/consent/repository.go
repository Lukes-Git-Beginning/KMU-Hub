package consent

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ConsentRecord represents a single consent grant or revocation.
type ConsentRecord struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	ContactID   uuid.UUID  `json:"contact_id"`
	ConsentType string     `json:"consent_type"`
	Granted     bool       `json:"granted"`
	LegalBasis  string     `json:"legal_basis"`
	Source      string     `json:"source,omitempty"`
	IPAddress   *string    `json:"ip_address,omitempty"`
	Notes       string     `json:"notes,omitempty"`
	GrantedAt   *time.Time `json:"granted_at,omitempty"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
	CreatedBy   *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// ConsentSummary holds the current consent status per type for a contact.
type ConsentSummary struct {
	ContactID uuid.UUID                      `json:"contact_id"`
	Consents  map[string]*ConsentRecord      `json:"consents"` // consent_type -> latest record
}

// GDPRDeletionRequest represents a GDPR Art. 17 erasure request.
//
// ContactID is nullable and set to NULL if the contact is later hard-deleted
// (migration 000322, ON DELETE SET NULL) -- the request row is Art. 5(2)
// accountability evidence and must outlive the contact it was about.
// OriginalContactID has no foreign key and is populated once at creation, so
// it identifies the (possibly since-deleted) contact even after ContactID
// goes NULL.
type GDPRDeletionRequest struct {
	ID                uuid.UUID  `json:"id"`
	TenantID          uuid.UUID  `json:"tenant_id"`
	ContactID         *uuid.UUID `json:"contact_id"`
	OriginalContactID uuid.UUID  `json:"original_contact_id"`
	RequestedBy       *uuid.UUID `json:"requested_by,omitempty"`
	Reason            string     `json:"reason,omitempty"`
	Status            string     `json:"status"` // pending, processing, completed
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

// Repository defines the interface for consent persistence.
type Repository interface {
	// Consent records
	CreateConsentRecord(ctx context.Context, record *ConsentRecord) error
	GetConsentHistory(ctx context.Context, tenantID, contactID uuid.UUID, consentType string) ([]*ConsentRecord, error)
	GetLatestConsents(ctx context.Context, tenantID, contactID uuid.UUID) ([]*ConsentRecord, error)

	// GDPR deletion
	CreateDeletionRequest(ctx context.Context, req *GDPRDeletionRequest) error
	GetDeletionRequest(ctx context.Context, id, tenantID uuid.UUID) (*GDPRDeletionRequest, error)
	UpdateDeletionRequest(ctx context.Context, req *GDPRDeletionRequest) error

	// Contact anonymization (GDPR Art. 17)
	AnonymizeContact(ctx context.Context, contactID, tenantID uuid.UUID) error

	// Checks
	ContactExists(ctx context.Context, contactID, tenantID uuid.UUID) (bool, error)
}

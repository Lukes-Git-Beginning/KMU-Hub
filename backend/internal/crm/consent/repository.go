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
type GDPRDeletionRequest struct {
	ID          uuid.UUID  `json:"id"`
	ContactID   uuid.UUID  `json:"contact_id"`
	RequestedBy *uuid.UUID `json:"requested_by,omitempty"`
	Reason      string     `json:"reason,omitempty"`
	Status      string     `json:"status"` // pending, processing, completed
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// Repository defines the interface for consent persistence.
type Repository interface {
	// Consent records
	CreateConsentRecord(ctx context.Context, record *ConsentRecord) error
	GetConsentHistory(ctx context.Context, tenantID, contactID uuid.UUID, consentType string) ([]*ConsentRecord, error)
	GetLatestConsents(ctx context.Context, tenantID, contactID uuid.UUID) ([]*ConsentRecord, error)

	// GDPR deletion
	CreateDeletionRequest(ctx context.Context, req *GDPRDeletionRequest) error
	GetDeletionRequest(ctx context.Context, id uuid.UUID) (*GDPRDeletionRequest, error)
	UpdateDeletionRequest(ctx context.Context, req *GDPRDeletionRequest) error

	// Contact anonymization (GDPR Art. 17)
	AnonymizeContact(ctx context.Context, contactID uuid.UUID) error

	// Checks
	ContactExists(ctx context.Context, contactID uuid.UUID) (bool, error)
}

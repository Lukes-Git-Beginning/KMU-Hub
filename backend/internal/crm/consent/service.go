package consent

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// Service handles GDPR consent management business logic.
type Service struct {
	repo *PostgresRepository
}

// NewService creates a new consent service.
func NewService(repo *PostgresRepository) *Service {
	return &Service{repo: repo}
}

// GrantConsent records a consent grant for a contact.
func (s *Service) GrantConsent(ctx context.Context, contactID uuid.UUID, consentType, source, legalBasis string, ip *string, createdBy *uuid.UUID) (*ConsentRecord, error) {
	if !ValidConsentTypes[consentType] {
		return nil, ErrInvalidConsentType
	}
	if !ValidLegalBases[legalBasis] {
		return nil, ErrInvalidLegalBasis
	}

	exists, err := s.repo.ContactExists(ctx, contactID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrContactNotFound
	}

	now := time.Now()
	record := &ConsentRecord{
		ID:          uuid.New(),
		ContactID:   contactID,
		ConsentType: consentType,
		Granted:     true,
		LegalBasis:  legalBasis,
		Source:      source,
		IPAddress:   ip,
		GrantedAt:   &now,
		CreatedBy:   createdBy,
		CreatedAt:   now,
	}

	if err := s.repo.CreateConsentRecord(ctx, record); err != nil {
		return nil, err
	}

	slog.Info("consent.granted",
		"contact_id", contactID,
		"consent_type", consentType,
		"legal_basis", legalBasis,
	)

	return record, nil
}

// RevokeConsent records a consent revocation for a contact.
func (s *Service) RevokeConsent(ctx context.Context, contactID uuid.UUID, consentType, notes string, revokedBy *uuid.UUID) (*ConsentRecord, error) {
	if !ValidConsentTypes[consentType] {
		return nil, ErrInvalidConsentType
	}

	exists, err := s.repo.ContactExists(ctx, contactID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrContactNotFound
	}

	now := time.Now()
	record := &ConsentRecord{
		ID:          uuid.New(),
		ContactID:   contactID,
		ConsentType: consentType,
		Granted:     false,
		LegalBasis:  "consent",
		Notes:       notes,
		RevokedAt:   &now,
		CreatedBy:   revokedBy,
		CreatedAt:   now,
	}

	if err := s.repo.CreateConsentRecord(ctx, record); err != nil {
		return nil, err
	}

	slog.Info("consent.revoked",
		"contact_id", contactID,
		"consent_type", consentType,
	)

	return record, nil
}

// GetContactConsents returns the current consent status for all types for a contact.
func (s *Service) GetContactConsents(ctx context.Context, contactID uuid.UUID) (*ConsentSummary, error) {
	records, err := s.repo.GetLatestConsents(ctx, contactID)
	if err != nil {
		return nil, err
	}

	summary := &ConsentSummary{
		ContactID: contactID,
		Consents:  make(map[string]*ConsentRecord),
	}
	for _, r := range records {
		summary.Consents[r.ConsentType] = r
	}
	return summary, nil
}

// GetConsentHistory returns the full consent history for a contact and consent type.
func (s *Service) GetConsentHistory(ctx context.Context, contactID uuid.UUID, consentType string) ([]*ConsentRecord, error) {
	if !ValidConsentTypes[consentType] {
		return nil, ErrInvalidConsentType
	}
	return s.repo.GetConsentHistory(ctx, contactID, consentType)
}

// RequestDeletion creates a GDPR Art. 17 erasure request.
func (s *Service) RequestDeletion(ctx context.Context, contactID uuid.UUID, reason string, requestedBy *uuid.UUID) (*GDPRDeletionRequest, error) {
	exists, err := s.repo.ContactExists(ctx, contactID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrContactNotFound
	}

	now := time.Now()
	req := &GDPRDeletionRequest{
		ID:          uuid.New(),
		ContactID:   contactID,
		RequestedBy: requestedBy,
		Reason:      reason,
		Status:      "pending",
		CreatedAt:   now,
	}

	if err := s.repo.CreateDeletionRequest(ctx, req); err != nil {
		return nil, err
	}

	slog.Info("gdpr.deletion_request",
		"request_id", req.ID,
		"contact_id", contactID,
	)

	return req, nil
}

// ProcessDeletion executes a GDPR erasure request by anonymizing the contact.
func (s *Service) ProcessDeletion(ctx context.Context, requestID uuid.UUID) error {
	req, err := s.repo.GetDeletionRequest(ctx, requestID)
	if err != nil {
		return err
	}
	if req.Status == "completed" {
		return ErrDeletionAlreadyComplete
	}

	if err := s.repo.AnonymizeContact(ctx, req.ContactID); err != nil {
		return err
	}

	now := time.Now()
	req.Status = "completed"
	req.CompletedAt = &now
	if err := s.repo.UpdateDeletionRequest(ctx, req); err != nil {
		return err
	}

	slog.Info("gdpr.deletion_processed",
		"request_id", requestID,
		"contact_id", req.ContactID,
	)

	return nil
}

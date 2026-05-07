package signature

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Service handles email signature business logic.
type Service struct {
	repo Repository
}

// NewService creates a new signature service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Create creates a new email signature.
// If this is the first signature for the user within the tenant, it is set as default.
func (s *Service) Create(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID, name, htmlContent string) (*models.EmailSignature, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("signature name is required")
	}

	// Check if this is the first signature for this user within this tenant
	count, err := s.repo.CountByUser(ctx, userID, tenantID)
	if err != nil {
		return nil, err
	}
	isDefault := count == 0

	now := time.Now().UTC()
	sig := &models.EmailSignature{
		ID:          uuid.New(),
		TenantID:    tenantID,
		UserID:      userID,
		Name:        name,
		HTMLContent: htmlContent,
		IsDefault:   isDefault,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.Create(ctx, sig); err != nil {
		return nil, fmt.Errorf("failed to create signature: %w", err)
	}

	slog.Info("signature created",
		"signature_id", sig.ID,
		"user_id", userID,
		"tenant_id", tenantID,
		"is_default", isDefault,
	)

	return sig, nil
}

// GetByID returns a signature by ID scoped to a tenant.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.EmailSignature, error) {
	return s.repo.GetByID(ctx, id, tenantID)
}

// GetDefault returns the default signature for a user within a tenant.
func (s *Service) GetDefault(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) (*models.EmailSignature, error) {
	return s.repo.GetDefault(ctx, userID, tenantID)
}

// ListByUser returns all signatures for a user within a tenant.
func (s *Service) ListByUser(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) ([]*models.EmailSignature, error) {
	return s.repo.ListByUser(ctx, userID, tenantID)
}

// Update updates a signature's name and content.
func (s *Service) Update(ctx context.Context, id uuid.UUID, tenantID uuid.UUID, name, htmlContent string) (*models.EmailSignature, error) {
	sig, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	if name != "" {
		sig.Name = strings.TrimSpace(name)
	}
	sig.HTMLContent = htmlContent

	if err := s.repo.Update(ctx, sig); err != nil {
		return nil, fmt.Errorf("failed to update signature: %w", err)
	}

	slog.Info("signature updated", "signature_id", id, "tenant_id", tenantID)
	return sig, nil
}

// Delete removes a signature.
func (s *Service) Delete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	if err := s.repo.Delete(ctx, id, tenantID); err != nil {
		return fmt.Errorf("failed to delete signature: %w", err)
	}

	slog.Info("signature deleted", "signature_id", id, "tenant_id", tenantID)
	return nil
}

// SetDefault sets a signature as the default for its user, clearing other defaults.
func (s *Service) SetDefault(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	sig, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return err
	}

	// Clear all defaults for this user within this tenant
	if err := s.repo.ClearDefaultForUser(ctx, sig.UserID, tenantID); err != nil {
		return fmt.Errorf("failed to clear defaults: %w", err)
	}

	// Set the new default
	sig.IsDefault = true
	if err := s.repo.Update(ctx, sig); err != nil {
		return fmt.Errorf("failed to set default: %w", err)
	}

	slog.Info("signature set as default",
		"signature_id", id,
		"user_id", sig.UserID,
		"tenant_id", tenantID,
	)

	return nil
}

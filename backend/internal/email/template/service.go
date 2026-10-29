package template

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

const (
	// VisibilityPersonal templates are only visible to their owner (and
	// admins). VisibilityShared templates are visible to the whole tenant.
	VisibilityPersonal = "personal"
	VisibilityShared   = "shared"
)

// AllowedPlaceholders is the fixed, exhaustive set of tokens Render will
// substitute (as literal "{{key}}" occurrences). This is deliberately not a
// general template engine — arbitrary field access on a caller-supplied
// struct would let a template exfiltrate data through fields nobody
// intended to expose. Extending the feature means adding a key here, not
// widening the substitution mechanism.
var AllowedPlaceholders = []string{
	"contact_first_name",
	"contact_last_name",
	"contact_email",
	"company_name",
	"sender_name",
	"today",
}

func isValidVisibility(v string) bool {
	return v == VisibilityPersonal || v == VisibilityShared
}

// Service handles email template business logic.
type Service struct {
	repo Repository
}

// NewService creates a new template service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Create creates a new email template. An empty visibility defaults to
// personal. A personal template is always owned by its creator; a shared
// template has no owner. Only a maintainer (coarse email key, admin) creates
// a shared template; everyone else is confined to personal ones, the same
// line Update and Delete draw.
func (s *Service) Create(ctx context.Context, tenantID, ownerID uuid.UUID, maintainer bool, name, subject, bodyHTML, bodyText, visibility string) (*models.EmailTemplate, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrNameRequired
	}
	if visibility == "" {
		visibility = VisibilityPersonal
	}
	if !isValidVisibility(visibility) {
		return nil, ErrInvalidVisibility
	}
	if visibility == VisibilityShared && !maintainer {
		return nil, ErrSharedTemplate
	}

	var owner *uuid.UUID
	if visibility == VisibilityPersonal {
		owner = &ownerID
	}

	now := time.Now().UTC()
	tpl := &models.EmailTemplate{
		ID:         uuid.New(),
		TenantID:   tenantID,
		OwnerID:    owner,
		Visibility: visibility,
		Name:       name,
		Subject:    subject,
		BodyHTML:   bodyHTML,
		BodyText:   bodyText,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.repo.Create(ctx, tpl); err != nil {
		return nil, fmt.Errorf("failed to create email template: %w", err)
	}

	slog.Info("email template created", "template_id", tpl.ID, "tenant_id", tenantID, "visibility", visibility)
	return tpl, nil
}

// GetByID returns a template visible to userID (shared, own, or isAdmin).
func (s *Service) GetByID(ctx context.Context, id, tenantID, userID uuid.UUID, isAdmin bool) (*models.EmailTemplate, error) {
	return s.repo.GetByID(ctx, id, tenantID, userID, isAdmin)
}

// List returns every template visible to userID within tenantID.
func (s *Service) List(ctx context.Context, tenantID, userID uuid.UUID, isAdmin bool) ([]*models.EmailTemplate, error) {
	return s.repo.ListVisible(ctx, tenantID, userID, isAdmin)
}

// maintainable checks that the caller may change or remove tpl. A
// maintainer (coarse email key, admin) keeps every template it can see;
// everyone else only their own personal templates. A foreign personal
// template stays NotFound, as it is for the visibility filter.
func maintainable(tpl *models.EmailTemplate, userID uuid.UUID, maintainer bool) error {
	if maintainer {
		return nil
	}
	if tpl.Visibility == VisibilityShared {
		return ErrSharedTemplate
	}
	if tpl.OwnerID == nil || *tpl.OwnerID != userID {
		return ErrTemplateNotFound
	}
	return nil
}

// Update applies the given optional fields to a template visible to userID.
// Switching to shared clears the owner; switching to personal on a
// previously shared template makes the caller its new owner. Only a
// maintainer touches shared templates or changes the visibility (decision
// 2026-10-25: own personal templates free, shared ones admin only).
func (s *Service) Update(ctx context.Context, id, tenantID, userID uuid.UUID, isAdmin, maintainer bool, name, subject, bodyHTML, bodyText, visibility *string) (*models.EmailTemplate, error) {
	tpl, err := s.repo.GetByID(ctx, id, tenantID, userID, isAdmin)
	if err != nil {
		return nil, err
	}
	if err := maintainable(tpl, userID, maintainer); err != nil {
		return nil, err
	}
	if !maintainer && visibility != nil && *visibility != tpl.Visibility {
		return nil, ErrSharedTemplate
	}

	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" {
			return nil, ErrNameRequired
		}
		tpl.Name = trimmed
	}
	if subject != nil {
		tpl.Subject = *subject
	}
	if bodyHTML != nil {
		tpl.BodyHTML = *bodyHTML
	}
	if bodyText != nil {
		tpl.BodyText = *bodyText
	}
	if visibility != nil {
		if !isValidVisibility(*visibility) {
			return nil, ErrInvalidVisibility
		}
		tpl.Visibility = *visibility
		if tpl.Visibility == VisibilityShared {
			tpl.OwnerID = nil
		} else if tpl.OwnerID == nil {
			tpl.OwnerID = &userID
		}
	}

	if err := s.repo.Update(ctx, tpl); err != nil {
		return nil, fmt.Errorf("failed to update email template: %w", err)
	}

	slog.Info("email template updated", "template_id", id, "tenant_id", tenantID)
	return tpl, nil
}

// Delete removes a template visible to userID; without maintainer only the
// caller's own personal template.
func (s *Service) Delete(ctx context.Context, id, tenantID, userID uuid.UUID, isAdmin, maintainer bool) error {
	tpl, err := s.repo.GetByID(ctx, id, tenantID, userID, isAdmin)
	if err != nil {
		return err
	}
	if err := maintainable(tpl, userID, maintainer); err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id, tenantID); err != nil {
		return fmt.Errorf("failed to delete email template: %w", err)
	}

	slog.Info("email template deleted", "template_id", id, "tenant_id", tenantID)
	return nil
}

// Render substitutes AllowedPlaceholders tokens in a template visible to
// userID. Keys in values that are not in AllowedPlaceholders are ignored —
// only the fixed allow-list is ever looked up, so an unexpected key can
// never reach the output.
func (s *Service) Render(ctx context.Context, id, tenantID, userID uuid.UUID, isAdmin bool, values map[string]string) (subject, bodyHTML, bodyText string, err error) {
	tpl, err := s.repo.GetByID(ctx, id, tenantID, userID, isAdmin)
	if err != nil {
		return "", "", "", err
	}

	subject, bodyHTML, bodyText = tpl.Subject, tpl.BodyHTML, tpl.BodyText
	for _, key := range AllowedPlaceholders {
		val, ok := values[key]
		if !ok {
			continue
		}
		token := "{{" + key + "}}"
		subject = strings.ReplaceAll(subject, token, val)
		bodyHTML = strings.ReplaceAll(bodyHTML, token, val)
		bodyText = strings.ReplaceAll(bodyText, token, val)
	}
	return subject, bodyHTML, bodyText, nil
}

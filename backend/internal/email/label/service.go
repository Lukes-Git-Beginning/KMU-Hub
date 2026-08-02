package label

import (
	"context"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

var colorHexRe = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

const defaultColor = "#6B7280"

// MessageReader is the minimal view of the message service this package needs
// to answer "what does this message look like after a label assignment"
// without depending on the whole message package. *message.Service satisfies
// it structurally (the same pattern send.Service uses for MessageCreator).
type MessageReader interface {
	GetByID(ctx context.Context, id, tenantID uuid.UUID) (*models.EmailMessage, error)
}

// Service handles label business logic: validation, persistence and the
// message assignment endpoint.
type Service struct {
	repo     Repository
	messages MessageReader
}

// NewService creates a new label service.
func NewService(repo Repository, messages MessageReader) *Service {
	return &Service{repo: repo, messages: messages}
}

// List returns all labels of a tenant, name-ordered.
func (s *Service) List(ctx context.Context, tenantID uuid.UUID) ([]*models.EmailLabel, error) {
	return s.repo.List(ctx, tenantID)
}

// Create validates and stores a new label.
func (s *Service) Create(ctx context.Context, tenantID uuid.UUID, name, color string) (*models.EmailLabel, error) {
	name = strings.TrimSpace(name)
	if color == "" {
		color = defaultColor
	}
	if err := validate(name, color); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	l := &models.EmailLabel{
		ID: uuid.New(), TenantID: tenantID, Name: name, Color: color,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.repo.Create(ctx, l); err != nil {
		return nil, err
	}
	slog.Info("email label created", "label_id", l.ID, "tenant_id", tenantID)
	return l, nil
}

// Update applies a partial change (nil fields keep their stored value).
func (s *Service) Update(ctx context.Context, id, tenantID uuid.UUID, name, color *string) (*models.EmailLabel, error) {
	l, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}
	if name != nil {
		l.Name = strings.TrimSpace(*name)
	}
	if color != nil && strings.TrimSpace(*color) != "" {
		l.Color = strings.TrimSpace(*color)
	}
	if err := validate(l.Name, l.Color); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, l); err != nil {
		return nil, err
	}
	slog.Info("email label updated", "label_id", id, "tenant_id", tenantID)
	return l, nil
}

// Delete removes a label of the tenant.
func (s *Service) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	if err := s.repo.Delete(ctx, id, tenantID); err != nil {
		return err
	}
	slog.Info("email label deleted", "label_id", id, "tenant_id", tenantID)
	return nil
}

// AssignToMessage replaces the label set of a message and returns the message
// as it looks afterwards. Every id in labelIDs must already be a label of the
// tenant -- without that check an assignment could plant a foreign or
// nonexistent label id in label_ids, which has no foreign key to catch it.
func (s *Service) AssignToMessage(ctx context.Context, tenantID, messageID uuid.UUID, labelIDs []uuid.UUID) (*models.EmailMessage, error) {
	ok, err := s.repo.LabelIDsBelongToTenant(ctx, tenantID, labelIDs)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrTargetInvalid
	}

	if err := s.repo.AssignToMessage(ctx, tenantID, messageID, labelIDs); err != nil {
		return nil, err
	}
	return s.messages.GetByID(ctx, messageID, tenantID)
}

func validate(name, color string) error {
	if name == "" {
		return ErrNameRequired
	}
	if !colorHexRe.MatchString(color) {
		return ErrColorInvalid
	}
	return nil
}

package thread

import (
	"context"

	"github.com/google/uuid"
)

// Service is the thin service layer for inbox threads and canned responses.
type Service struct {
	repo Repository
}

// NewService creates a Service wired to the given Repository.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ListThread returns the conversation thread for an inbox message. If the thread
// is still empty it first materializes the inbound original from seed, so the
// originating message always renders as the first bubble.
//
// lean: the inbound seed body is the inbox preview (full inbound body is not
// stored). Upgrade to the full body when channel adapters push inbound content.
func (s *Service) ListThread(ctx context.Context, tenantID, messageID uuid.UUID, seed InboundSeed) ([]*ThreadMessage, error) {
	if seed.Body != "" {
		if err := s.repo.SeedInboundIfEmpty(ctx, tenantID, messageID, seed.SenderName, seed.Body, seed.CreatedAt); err != nil {
			return nil, err
		}
	}
	return s.repo.ListThreadMessages(ctx, tenantID, messageID)
}

// AppendReply persists an outbound reply row after the reply was delegated to
// the channel adapter.
func (s *Service) AppendReply(ctx context.Context, tenantID, messageID, authorID uuid.UUID, body string) error {
	aid := authorID
	return s.repo.AppendThreadMessage(ctx, &ThreadMessage{
		TenantID:  tenantID,
		MessageID: messageID,
		AuthorID:  &aid,
		Direction: DirectionOutbound,
		Body:      body,
	})
}

// CreateCanned creates a tenant-scoped canned response.
func (s *Service) CreateCanned(ctx context.Context, tenantID uuid.UUID, name, body string) (*CannedResponse, error) {
	c := &CannedResponse{TenantID: tenantID, Name: name, Body: body}
	if err := s.repo.CreateCannedResponse(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

// ListCanned returns all canned responses for the tenant.
func (s *Service) ListCanned(ctx context.Context, tenantID uuid.UUID) ([]*CannedResponse, error) {
	return s.repo.ListCannedResponses(ctx, tenantID)
}

// UpdateCanned applies the given name/body changes to a canned response.
func (s *Service) UpdateCanned(ctx context.Context, tenantID, id uuid.UUID, name, body *string) (*CannedResponse, error) {
	c, err := s.repo.GetCannedResponse(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if name != nil {
		c.Name = *name
	}
	if body != nil {
		c.Body = *body
	}
	if err := s.repo.UpdateCannedResponse(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

// DeleteCanned removes a canned response.
func (s *Service) DeleteCanned(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.repo.DeleteCannedResponse(ctx, tenantID, id)
}

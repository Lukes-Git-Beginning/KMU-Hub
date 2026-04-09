package dialer

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// CampaignRepository handles persistence for campaigns and their contact queues.
type CampaignRepository interface {
	// Campaign CRUD
	Create(ctx context.Context, c *Campaign) error
	GetByID(ctx context.Context, id uuid.UUID) (*Campaign, error)
	List(ctx context.Context, tenantID uuid.UUID, statusFilter *string, page, pageSize int) ([]*Campaign, int, error)
	Update(ctx context.Context, c *Campaign) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Campaign contact queue management
	AddContacts(ctx context.Context, campaignID uuid.UUID, contacts []CampaignContact) (added int, skipped int, err error)
	GetNextPendingContact(ctx context.Context, campaignID uuid.UUID) (*CampaignContact, error)
	ListContacts(ctx context.Context, campaignID uuid.UUID, statusFilter *string, page, pageSize int) ([]*CampaignContact, int, error)
	UpdateContactStatus(ctx context.Context, id uuid.UUID, status string, outcomeID *uuid.UUID) error
	SetContactCallback(ctx context.Context, id uuid.UUID, callbackAt time.Time) error
	SkipContact(ctx context.Context, id uuid.UUID) error
	RequeueContact(ctx context.Context, id uuid.UUID) error
	IncrementContactCallCount(ctx context.Context, id uuid.UUID) error

	// Aggregate stats and denormalized counts
	GetCampaignStats(ctx context.Context, campaignID uuid.UUID) (*CampaignStats, error)
	GetAgentStats(ctx context.Context, agentID uuid.UUID) (*AgentStats, error)
	UpdateCampaignCounts(ctx context.Context, campaignID uuid.UUID) error
}

// CallRepository handles persistence for call sessions and their events.
type CallRepository interface {
	CreateSession(ctx context.Context, s *CallSession) error
	GetSessionByID(ctx context.Context, id uuid.UUID) (*CallSession, error)
	UpdateSession(ctx context.Context, s *CallSession) error
	AppendEvent(ctx context.Context, e *CallEvent) error
	ListEventsBySession(ctx context.Context, sessionID uuid.UUID) ([]*CallEvent, error)
}

// OutcomeRepository handles persistence for call outcomes (per-tenant configurable).
type OutcomeRepository interface {
	Create(ctx context.Context, o *CallOutcome) error
	GetByID(ctx context.Context, id uuid.UUID) (*CallOutcome, error)
	List(ctx context.Context, tenantID uuid.UUID, includeInactive bool) ([]*CallOutcome, error)
	Update(ctx context.Context, o *CallOutcome) error
	Delete(ctx context.Context, id uuid.UUID) error
	// EnsureDefaults creates the 4 standard outcomes for a tenant if none exist yet.
	EnsureDefaults(ctx context.Context, tenantID uuid.UUID) error
}

// AgentStatusRepository handles persistence for agent status transitions.
type AgentStatusRepository interface {
	LogStatusChange(ctx context.Context, entry *AgentStatusLogEntry) error
}

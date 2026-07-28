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
	GetByIDForTenant(ctx context.Context, id, tenantID uuid.UUID) (*Campaign, error)
	List(ctx context.Context, tenantID uuid.UUID, statusFilter *string, page, pageSize int) ([]*Campaign, int, error)
	// Update, UpdateStatus and Delete take an explicit tenantID and filter on it
	// in the WHERE clause — not just relying on RLS — so a caller with a
	// campaign ID from the wrong tenant fails closed (0 rows affected) instead
	// of depending solely on the session's app.tenant_id being set correctly.
	Update(ctx context.Context, c *Campaign, tenantID uuid.UUID) error
	UpdateStatus(ctx context.Context, id, tenantID uuid.UUID, status string) error
	Delete(ctx context.Context, id, tenantID uuid.UUID) error

	// Campaign contact queue management. Writes take an explicit tenantID for
	// the same defense-in-depth reason as the campaign writes above.
	AddContacts(ctx context.Context, campaignID uuid.UUID, contacts []CampaignContact) (added int, skipped int, err error)
	GetNextPendingContact(ctx context.Context, campaignID uuid.UUID) (*CampaignContact, error)
	ListContacts(ctx context.Context, campaignID uuid.UUID, statusFilter *string, page, pageSize int) ([]*CampaignContact, int, error)
	UpdateContactStatus(ctx context.Context, id, tenantID uuid.UUID, status string, outcomeID *uuid.UUID) error
	SetContactCallback(ctx context.Context, id, tenantID uuid.UUID, callbackAt time.Time) error
	SkipContact(ctx context.Context, id, tenantID uuid.UUID) error
	RequeueContact(ctx context.Context, id, tenantID uuid.UUID) error
	IncrementContactCallCount(ctx context.Context, id, tenantID uuid.UUID) error
	GetCampaignContactByID(ctx context.Context, id, tenantID uuid.UUID) (*CampaignContact, error)

	// Aggregate stats and denormalized counts
	GetCampaignStats(ctx context.Context, campaignID uuid.UUID) (*CampaignStats, error)
	GetAgentStats(ctx context.Context, agentID uuid.UUID) (*AgentStats, error)
	UpdateCampaignCounts(ctx context.Context, campaignID uuid.UUID) error
}

// CallRepository handles persistence for call sessions and their events.
type CallRepository interface {
	CreateSession(ctx context.Context, s *CallSession) error
	GetSessionByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*CallSession, error)
	UpdateSession(ctx context.Context, s *CallSession) error
	AppendEvent(ctx context.Context, e *CallEvent) error
	ListEventsBySession(ctx context.Context, sessionID uuid.UUID) ([]*CallEvent, error)
	// UpdateSessionWithEventAndContact atomically updates a call session, appends
	// an outcome event, and updates the campaign contact's queue status in a
	// single database transaction. This prevents partial writes when
	// LogCallOutcome is interrupted between the session/event persist and the
	// contact-status update — which would otherwise strand the contact in an
	// inconsistent status (e.g. in_progress instead of completed/callback).
	//
	// contactID is the campaign-contact join row to update; contactStatus is the
	// resulting status; outcomeID is recorded on the contact; callbackAt, when
	// non-nil, sets the contact's callback_at and is otherwise left unchanged.
	UpdateSessionWithEventAndContact(ctx context.Context, s *CallSession, e *CallEvent, contactID uuid.UUID, contactStatus string, outcomeID *uuid.UUID, callbackAt *time.Time) error

	// GetRecentCallsForTenant returns the most recent calls across all campaigns
	// for the given tenant, enriched with contact and outcome display data.
	GetRecentCallsForTenant(ctx context.Context, tenantID uuid.UUID, limit int) ([]RecentCallRow, error)

	// ListCallsByContact returns all call sessions for a campaign contact,
	// enriched with outcome and agent display data.
	ListCallsByContact(ctx context.Context, campaignContactID uuid.UUID) ([]ContactCallRow, error)

	// GetTenantCallsTodayCount returns the total number of calls made today for
	// all agents within the given tenant.
	GetTenantCallsTodayCount(ctx context.Context, tenantID uuid.UUID) (int, error)

	// GetTenantAppointmentsTodayCount returns the number of appointments
	// (calls with is_appointment outcome) made today across the tenant.
	GetTenantAppointmentsTodayCount(ctx context.Context, tenantID uuid.UUID) (int, error)

	// GetAgentCallsTodayCount returns the number of calls made today by a single agent.
	GetAgentCallsTodayCount(ctx context.Context, agentID uuid.UUID) (int, error)

	// GetAgentAvgDurationToday returns the average call duration in seconds for
	// an agent's calls today.
	GetAgentAvgDurationToday(ctx context.Context, agentID uuid.UUID) (float64, error)
}

// OutcomeRepository handles persistence for call outcomes (per-tenant configurable).
type OutcomeRepository interface {
	Create(ctx context.Context, o *CallOutcome) error
	GetByID(ctx context.Context, id, tenantID uuid.UUID) (*CallOutcome, error)
	List(ctx context.Context, tenantID uuid.UUID, includeInactive bool) ([]*CallOutcome, error)
	// Update trusts o.TenantID (populated by a tenant-scoped GetByID beforehand)
	// as the predicate; Delete takes tenantID explicitly since it has no
	// preceding object load.
	Update(ctx context.Context, o *CallOutcome) error
	Delete(ctx context.Context, id, tenantID uuid.UUID) error
	// EnsureDefaults creates the 4 standard outcomes for a tenant if none exist yet.
	EnsureDefaults(ctx context.Context, tenantID uuid.UUID) error
}

// AgentStatusRepository handles persistence for agent status transitions.
type AgentStatusRepository interface {
	LogStatusChange(ctx context.Context, entry *AgentStatusLogEntry) error

	// GetActiveAgentIDsForTenant returns the distinct user IDs of agents who
	// logged a status change today for the given tenant. Used by the supervisor
	// overview to enumerate agents whose live status should be fetched from Redis.
	GetActiveAgentIDsForTenant(ctx context.Context, tenantID uuid.UUID) ([]uuid.UUID, error)

	// GetUserDisplayNames returns a map of user ID → (first_name, last_name) for
	// the given set of user IDs. Used to enrich supervisor agent rows with names.
	GetUserDisplayNames(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID][2]string, error)
}

package helpdesk

import (
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Ticket status constants
// ---------------------------------------------------------------------------

const (
	TicketStatusOpen    = "open"
	TicketStatusPending = "pending"
	TicketStatusSolved  = "solved"
	TicketStatusClosed  = "closed"
	TicketStatusMerged  = "merged"
)

// ValidTicketStatuses lists all accepted ticket status values.
var ValidTicketStatuses = map[string]bool{
	TicketStatusOpen:    true,
	TicketStatusPending: true,
	TicketStatusSolved:  true,
	TicketStatusClosed:  true,
	TicketStatusMerged:  true,
}

// ---------------------------------------------------------------------------
// Ticket priority constants
// ---------------------------------------------------------------------------

const (
	TicketPriorityLow    = "low"
	TicketPriorityNormal = "normal"
	TicketPriorityHigh   = "high"
	TicketPriorityUrgent = "urgent"
)

// ValidTicketPriorities lists all accepted ticket priority values.
var ValidTicketPriorities = map[string]bool{
	TicketPriorityLow:    true,
	TicketPriorityNormal: true,
	TicketPriorityHigh:   true,
	TicketPriorityUrgent: true,
}

// ---------------------------------------------------------------------------
// SLA status constants
// ---------------------------------------------------------------------------

// SLAStatus represents the current SLA health of a ticket.
type SLAStatus string

const (
	SLAStatusOnTrack SLAStatus = "on_track"
	SLAStatusAtRisk  SLAStatus = "at_risk"
	SLAStatusBreached SLAStatus = "breached"
)

// ---------------------------------------------------------------------------
// Domain models
// ---------------------------------------------------------------------------

// Ticket is the central helpdesk work item.
type Ticket struct {
	ID              uuid.UUID  `json:"id"`
	TenantID        uuid.UUID  `json:"tenant_id"`
	Subject         string     `json:"subject"`
	Status          string     `json:"status"`
	Priority        string     `json:"priority"`
	AssigneeID      *uuid.UUID `json:"assignee_id,omitempty"`
	RequesterID     uuid.UUID  `json:"requester_id"`
	QueueID         *uuid.UUID `json:"queue_id,omitempty"`
	DueAt           *time.Time `json:"due_at,omitempty"`
	MergedIntoID    *uuid.UUID `json:"merged_into_id,omitempty"`
	FirstResponseAt *time.Time `json:"first_response_at,omitempty"`
	ResolvedAt      *time.Time `json:"resolved_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// TicketMessage is a single reply or internal note on a ticket.
type TicketMessage struct {
	ID          uuid.UUID   `json:"id"`
	TenantID    uuid.UUID   `json:"tenant_id"`
	TicketID    uuid.UUID   `json:"ticket_id"`
	AuthorID    uuid.UUID   `json:"author_id"`
	Body        string      `json:"body"`
	Internal    bool        `json:"internal"`
	Attachments []string    `json:"attachments"`
	CreatedAt   time.Time   `json:"created_at"`
}

// TicketQueue is a named bucket into which tickets are routed.
type TicketQueue struct {
	ID               uuid.UUID  `json:"id"`
	TenantID         uuid.UUID  `json:"tenant_id"`
	Name             string     `json:"name"`
	DefaultAssigneeID *uuid.UUID `json:"default_assignee_id,omitempty"`
	SLAPolicyID      *uuid.UUID `json:"sla_policy_id,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// CannedResponse is a pre-written reply template.
type CannedResponse struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Name      string    `json:"name"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SLAPolicy defines first-response and resolution time targets for a queue.
type SLAPolicy struct {
	ID                uuid.UUID      `json:"id"`
	TenantID          uuid.UUID      `json:"tenant_id"`
	Name              string         `json:"name"`
	FirstResponseMins int            `json:"first_response_mins"`
	ResolutionMins    int            `json:"resolution_mins"`
	BusinessHours     map[string]any `json:"business_hours,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

// KBArticleStatus represents the publication state of a knowledge-base article.
type KBArticleStatus string

const (
	KBArticleStatusDraft     KBArticleStatus = "draft"
	KBArticleStatusPublished KBArticleStatus = "published"
)

// KBArticle is a knowledge-base article.
type KBArticle struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Category  string    `json:"category"`
	Status    string    `json:"status"`
	AuthorID  uuid.UUID `json:"author_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// RoutingRule auto-assigns incoming tickets to a target queue based on conditions.
type RoutingRule struct {
	ID            uuid.UUID      `json:"id"`
	TenantID      uuid.UUID      `json:"tenant_id"`
	Name          string         `json:"name"`
	Conditions    map[string]any `json:"conditions"`
	TargetQueueID *uuid.UUID     `json:"target_queue_id,omitempty"`
	Priority      int            `json:"priority"`
	Enabled       bool           `json:"enabled"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

// HelpdeskStats aggregates key helpdesk metrics for the statistics dashboard.
type HelpdeskStats struct {
	OpenTickets          int              `json:"open_tickets"`
	AvgResponseTime      string           `json:"avg_response_time"`
	ResolvedThisWeek     int              `json:"resolved_this_week"`
	CustomerSatisfaction string           `json:"customer_satisfaction"`
	WeeklyBreakdown      []WeeklyDayCount `json:"weekly_breakdown"`
}

// WeeklyDayCount is a single bar in the weekly-tickets chart.
type WeeklyDayCount struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

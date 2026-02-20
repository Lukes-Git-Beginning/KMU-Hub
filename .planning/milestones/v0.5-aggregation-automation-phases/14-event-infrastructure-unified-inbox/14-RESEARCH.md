# Phase 14: Event Infrastructure + Unified Inbox - Research

**Researched:** 2026-02-19
**Domain:** Event-driven architecture, message aggregation, routing engine
**Confidence:** HIGH

## Summary

Phase 14 has two tightly coupled deliverables: (1) retrofitting all existing services to emit structured events through the existing PostgreSQL `events` table + `pg_notify` infrastructure, and (2) building a Unified Inbox ("Kommunikation") that aggregates Email, Chat DMs/@mentions, and Notifications into a single triage-oriented view with team inboxes and routing rules.

The good news is that the event infrastructure is already partially built. Phase 4 created the `events` durability table, `event_types` registry, `pg_notify('events', ...)` emit pattern, and `EventBus` consumer framework. Three services (Chat, CRM, Work/Calendar) already emit events. The gap is that Email, Document, Finance (Biz), and HR services do NOT emit events yet. Extending them follows the exact same `PGEventEmitter` + `SetEventEmitter()` pattern already established in Chat/CRM/Work.

The Unified Inbox is a new service with its own proto, migration, and gateway routes. It introduces a `Channel Adapter` pattern (EmailAdapter, ChatAdapter, NotificationAdapter) that normalizes messages from different sources into a unified `inbox_messages` table. The inbox UI follows the three-column layout (sidebar | list | detail) already used in the MailsPage. Team inboxes, snooze, routing rules, and inline reply are the key features. The routing rule engine should use a simple JSON condition evaluator built in-house -- this is simple enough (AND/OR with field comparisons) that bringing in a full rule engine library like Grule would be overkill, and importantly, the condition evaluator should be designed for reuse by the Phase 16 Automation Engine.

**Primary recommendation:** Extend the existing event infrastructure to all services using the established PGEventEmitter pattern, then build the Unified Inbox as a new service in the notification binary (shared gRPC port :50054) since it is tightly coupled to the event/notification subsystem.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **Inbox layout:** Three-column layout (channel/filter sidebar | message list | detail/preview pane, Outlook/Gmail desktop pattern)
- **Channel badges:** Colored icon badges (blue envelope for email, green chat bubble, orange bell for notifications)
- **Message list items:** Two-line format (Line 1 = sender + timestamp, Line 2 = subject/preview snippet)
- **Left sidebar sections:** Smart views (All, Unread, Starred, Assigned to me) | Channel filters (Email, Chat, Notifications) | Team inboxes (dynamic per membership)
- **Triage workflow:** GTD-style inbox-zero (reply, delegate, snooze, archive)
- **Snooze:** Presets (1h, tomorrow morning, next week) AND custom date/time picker; item disappears and reappears at chosen time
- **Quick actions on hover:** Reply, snooze, archive, assign, star -- handle most items without opening detail pane
- **Inline reply:** Adapts per channel (email composer for email, chat input for chat, action buttons for notifications)
- **Team inbox modes:** Manual-claim OR round-robin auto-assign (team admin chooses)
- **Team inbox visibility:** Open (everyone sees all) or private (only unassigned queue + own assigned)
- **Team inbox membership:** Unlimited per user; each appears in sidebar
- **Team inbox creation:** Restricted to admin + manager roles
- **Routing rules:** AND/OR condition builder with rich condition fields (sender, subject, body keywords, CRM link status, tags)
- **Routing actions:** Route to team inbox, auto-assign, add tags, auto-reply
- **Rule scope:** Global rules (all channels) and per-channel rules
- **Module naming:** "Kommunikation" = external/cross-module comms; "Chat" = internal team messaging
- **NOT in scope:** Automation triggers/actions (Phase 16), calendar sync (Phase 15), Teams/Slack (Phase 17)

### Claude's Discretion
- Event infrastructure architecture (events table schema, pg_notify channel naming, consumer framework design)
- Channel adapter implementation patterns (normalization schema, adapter interfaces)
- Materialized inbox table design and update strategy
- Loading states, empty states, error handling
- Exact color values for channel badges (fit existing desk theme system)
- Keyboard shortcuts for power-user triage

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| EVENT-01 | All existing services emit structured events via PostgreSQL events table + pg_notify | Event emitter pattern already established in 3 services; 4 services (Email, Document, Biz, HR) need retrofitting using same PGEventEmitter pattern. New event types need seeding in event_types table. |
| INBOX-01 | User sees unified inbox aggregating Email, Chat DMs/@mentions, and Notifications in a single view | Channel adapter pattern normalizes into inbox_messages table; three-column UI with TanStack Query hooks; WebSocket push for real-time updates |
| INBOX-02 | User can reply, mark-read, and triage items from unified inbox without switching modules | Inline reply adapts per channel type; backend proxy routes reply to originating service via gRPC; snooze uses scheduled_at field with background requeue |
| INBOX-03 | Team inboxes with assignment and routing rules | team_inboxes + team_inbox_members tables; routing_rules table with JSONB conditions; condition evaluator with AND/OR logic; round-robin counter for auto-assign |
</phase_requirements>

## Standard Stack

### Core (Already in Project)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| pgx/v5 | Latest | PostgreSQL LISTEN/NOTIFY, event emitting | Already used across all services for pg_notify |
| chi/v5 | Latest | HTTP routing for gateway routes | Established gateway router |
| gRPC + protobuf | Latest | Inter-service communication | All services use gRPC proto definitions |
| TanStack Query | v5 | Frontend data fetching + caching | All modules use TanStack Query hooks |
| Zustand | Latest | UI-only state (selected item, filters) | Consistent with existing store patterns |

### Supporting (Already in Project)
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| slog | stdlib | Structured logging | All event processing, rule evaluation |
| encoding/json | stdlib | JSON condition evaluation | Routing rule condition parsing |
| google/uuid | Latest | ID generation | All new table rows |
| lucide-react | Latest | Icons for channel badges and UI | Sidebar icons, quick actions |
| sonner | Latest | Toast notifications for actions | Snooze/archive/assign confirmations |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Custom condition evaluator | Grule / expr-lang | Full rule engine is overkill for AND/OR field comparisons; custom is ~100 LOC, reusable in Phase 16 |
| Materialized view | Regular table with triggers | Regular table gives more control over update timing; triggers would couple to source tables across services |
| Separate inbox binary | Co-hosted in notification binary | Inbox is tightly coupled to event/notification subsystem; separate binary would add deployment complexity for little benefit |
| Redis for snooze scheduling | PostgreSQL + background goroutine | Redis adds dependency for a simple delayed re-queue; pg-based polling every 60s is sufficient for snooze precision |

**Installation:**
No new dependencies needed. All required libraries are already in the project.

## Architecture Patterns

### Recommended Project Structure

```
backend/
├── proto/inbox/v1/inbox.proto              # InboxService proto (~25 RPCs)
├── internal/inbox/
│   ├── adapter/
│   │   ├── adapter.go                      # ChannelAdapter interface
│   │   ├── email_adapter.go                # EmailAdapter (fetches from email service)
│   │   ├── chat_adapter.go                 # ChatAdapter (fetches from chat service)
│   │   └── notification_adapter.go         # NotificationAdapter (fetches from notification tables)
│   ├── message/
│   │   ├── repository.go                   # InboxMessage CRUD interface
│   │   ├── postgres_repository.go          # PostgreSQL implementation
│   │   └── service.go                      # Message normalization, CRUD, snooze
│   ├── team/
│   │   ├── repository.go                   # TeamInbox + membership CRUD
│   │   ├── postgres_repository.go
│   │   └── service.go                      # Team inbox management, assignment
│   └── routing/
│       ├── evaluator.go                    # Condition evaluator (AND/OR/field matchers)
│       ├── evaluator_test.go               # TDD: condition evaluation
│       ├── repository.go                   # RoutingRule CRUD
│       ├── postgres_repository.go
│       └── service.go                      # Rule matching, action execution
├── internal/notification/event/
│   ├── types.go                            # Extended with new event type constants
│   └── ... (existing files unchanged)
├── internal/email/                         # Add event emitting (retrofit)
├── internal/document/                      # Add event emitting (retrofit)
├── internal/biz/                           # Add event emitting (retrofit)
├── internal/gateway/
│   └── route_inbox.go                      # InboxRoutes (gateway HTTP routes)
└── migrations/
    ├── 000047_create_inbox_tables.up.sql   # inbox_messages, team_inboxes, routing_rules
    └── 000047_create_inbox_tables.down.sql
    ├── 000048_seed_inbox_event_types.up.sql
    └── 000048_seed_inbox_event_types.down.sql

desktop/src/renderer/src/
├── api/
│   ├── inbox-client.ts                     # Inbox API client (fetch wrapper)
│   ├── inbox-types.ts                      # TypeScript types
│   └── hooks/
│       └── useInbox.ts                     # TanStack Query hooks
├── modules/kommunikation/
│   ├── KommunikationPage.tsx               # Main three-column layout
│   ├── InboxSidebar.tsx                    # Smart views + channel filters + team inboxes
│   ├── MessageList.tsx                     # Two-line message items with quick actions
│   ├── MessageDetail.tsx                   # Detail pane with channel-adaptive reply
│   ├── SnoozePopover.tsx                   # Snooze preset + custom picker
│   ├── TeamInboxSettings.tsx               # Team inbox config (mode, visibility)
│   └── RoutingRulesEditor.tsx              # AND/OR condition builder UI
└── stores/kommunikation.ts                 # UI state (selected item, active view)
```

### Pattern 1: Event Emitter Retrofit (Extending Existing Services)

**What:** Add event emitting to Email, Document, Finance, and HR services using the established PGEventEmitter pattern.
**When to use:** Every service that performs CRUD operations that other modules might care about.

**Current Pattern (from chat/message/event_emitter.go):**
```go
// 1. Define the EventEmitter interface in the service package
type EventEmitter interface {
    EmitEmailEvent(ctx context.Context, payload models.EventPayload) error
}

// 2. Implement with PGEventEmitter
type PGEventEmitter struct {
    pool *pgxpool.Pool
}

func NewPGEventEmitter(pool *pgxpool.Pool) *PGEventEmitter {
    return &PGEventEmitter{pool: pool}
}

func (e *PGEventEmitter) EmitEmailEvent(ctx context.Context, payload models.EventPayload) error {
    return event.EmitEvent(ctx, e.pool, payload)
}

// 3. Optional setter on the service (nil-safe, same as calendar/task)
func (s *Service) SetEventEmitter(emitter EventEmitter) {
    s.emitter = emitter
}

// 4. Emit in service methods (nil-check emitter before calling)
func (s *Service) emitEvent(ctx context.Context, eventType string, ...) {
    if s.emitter == nil {
        return
    }
    payload := models.EventPayload{
        Type:     eventType,
        ModuleID: "email",
        // ...
    }
    s.emitter.EmitEmailEvent(ctx, payload)
}
```

**Key principle:** The emitter is optional and nil-safe. Services work without it. This decouples the service from the notification subsystem.

### Pattern 2: Channel Adapter for Message Normalization

**What:** Each source channel (Email, Chat, Notifications) has an adapter that converts source-specific messages into a normalized `InboxMessage` format.
**When to use:** When aggregating heterogeneous message sources into a unified view.

```go
// ChannelAdapter normalizes messages from a specific channel into InboxMessages.
type ChannelAdapter interface {
    // Channel returns the channel identifier (e.g., "email", "chat", "notification")
    Channel() string

    // FetchNewMessages fetches messages newer than the given cursor for a user.
    // Returns normalized InboxMessages ready for storage.
    FetchNewMessages(ctx context.Context, userID uuid.UUID, since time.Time) ([]InboxMessage, error)

    // HandleReply routes a reply back through the original channel.
    HandleReply(ctx context.Context, messageID uuid.UUID, userID uuid.UUID, body string) error

    // MarkReadOnSource syncs read status back to the original source.
    MarkReadOnSource(ctx context.Context, messageID uuid.UUID, userID uuid.UUID) error
}

// InboxMessage is the normalized message stored in inbox_messages.
type InboxMessage struct {
    ID             uuid.UUID
    UserID         uuid.UUID       // owner of this inbox item
    Channel        string          // "email", "chat", "notification"
    SourceID       string          // original message ID in source system
    SenderName     string          // display name of sender
    SenderID       *uuid.UUID      // internal user ID if applicable
    SenderEmail    *string         // email address if applicable
    Subject        string          // subject line or first line
    Preview        string          // body preview snippet (max ~200 chars)
    IsRead         bool
    IsStarred      bool
    IsArchived     bool
    SnoozedUntil   *time.Time      // nil = not snoozed
    AssignedTo     *uuid.UUID      // for team inbox assignment
    TeamInboxID    *uuid.UUID      // nil = personal inbox
    Tags           []string        // user-applied or rule-applied tags
    DeepLink       string          // route to open in source module
    CRMContactID   *uuid.UUID      // linked CRM contact if any
    Metadata       json.RawMessage // channel-specific extra data
    ReceivedAt     time.Time       // original message timestamp
    CreatedAt      time.Time       // when added to inbox
}
```

### Pattern 3: Routing Rule Condition Evaluator

**What:** JSON-based AND/OR condition tree that evaluates against message fields.
**When to use:** For inbox routing rules and as a reusable foundation for Phase 16 Automation.

```go
// Condition represents a rule condition (AND/OR tree with leaf comparisons).
type Condition struct {
    And      []Condition `json:"and,omitempty"`
    Or       []Condition `json:"or,omitempty"`
    Field    string      `json:"field,omitempty"`    // e.g., "channel", "sender", "subject", "body", "tags"
    Operator string      `json:"operator,omitempty"` // "equals", "contains", "starts_with", "in", "not_equals"
    Value    interface{} `json:"value,omitempty"`
}

// Action represents what happens when a rule matches.
type Action struct {
    Type   string          `json:"type"`   // "route_to_team", "assign_to", "add_tags", "auto_reply"
    Config json.RawMessage `json:"config"` // type-specific config
}

// RoutingRule ties conditions to actions.
type RoutingRule struct {
    ID         uuid.UUID
    Name       string
    Channel    *string     // nil = global, "email"/"chat"/"notification" = per-channel
    Conditions Condition   // root condition (AND/OR tree)
    Actions    []Action
    Priority   int         // lower = higher priority; rules evaluated in order
    IsActive   bool
    CreatedBy  uuid.UUID
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

// Evaluate returns true if the message matches the condition tree.
func (c *Condition) Evaluate(msg *InboxMessage) bool {
    if len(c.And) > 0 {
        for _, sub := range c.And {
            if !sub.Evaluate(msg) {
                return false
            }
        }
        return true
    }
    if len(c.Or) > 0 {
        for _, sub := range c.Or {
            if sub.Evaluate(msg) {
                return true
            }
        }
        return false
    }
    return c.evaluateLeaf(msg)
}
```

### Pattern 4: Event-Driven Inbox Population

**What:** When events arrive via the existing EventBus, the inbox consumer creates/updates inbox_messages.
**When to use:** For real-time inbox population without polling.

```
Event Flow:
1. Email service receives new email -> emits "email.message.received" via pg_notify
2. EventBus in notification service dispatches to all handlers
3. InboxConsumer handler (new) runs EmailAdapter.FetchNewMessages()
4. Normalized InboxMessage stored in inbox_messages table
5. pg_notify('inbox_delivery', ...) signals gateway for WebSocket push
6. Frontend receives WebSocket event -> invalidates inbox query cache
```

### Anti-Patterns to Avoid

- **Polling source services for changes:** Use events, not periodic polling. The EventBus already provides real-time event delivery. The inbox should react to events, not scan for changes.
- **Storing full message bodies in inbox_messages:** Store only preview snippets. Full content is fetched from the source service on demand (via the adapter) when the user opens a message. This avoids data duplication and keeps the inbox table lean.
- **Separate pg_notify channel per inbox feature:** Reuse the existing `events` channel for all event types. Add new event type constants (e.g., `email.message.received`, `biz.invoice.created`). The EventBus wildcard handler already dispatches all events.
- **Building separate inbox binary:** The inbox service is tightly coupled to notifications/events. Co-hosting in the notification binary avoids another port and Docker container.
- **Implementing undo for archive/delete:** Archive moves to archived state; delete is soft-delete. No undo queue needed in v1.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Event emission | New event transport | Existing `event.EmitEvent()` + `pg_notify` | Already built in Phase 4, battle-tested across 3 services |
| Event consumption | New consumer framework | Existing `EventBus.RegisterHandler()` | Wildcard handler + type-specific handlers already work |
| Event durability | Custom WAL | Existing `events` table + `ProcessBacklog()` | Dual-write pattern (table + notify) already handles service restarts |
| Email threading | Custom threading | Use existing `thread_id` from email service | Email already implements JWZ threading |
| WebSocket delivery | New WS protocol | Existing `wsHub.SendNotificationToUser()` + `pg_notify('notification_delivery')` | Gateway notification delivery listener already pushes to users |
| Date/time for snooze | Custom scheduler | Background goroutine with pg polling | `SELECT ... WHERE snoozed_until <= NOW() AND is_snoozed = true` every 60s |

**Key insight:** Phase 14 builds ON TOP of existing infrastructure (EventBus, pg_notify, events table, WebSocket hub). The core plumbing exists. The work is: (a) extending it to more services, (b) adding a new consumer that feeds the inbox, and (c) building the inbox service and UI.

## Common Pitfalls

### Pitfall 1: pg_notify 8000-byte Payload Limit
**What goes wrong:** Event payloads exceed 8000 bytes when including full message content, causing silent truncation or errors.
**Why it happens:** Developers include too much data in the NOTIFY payload (e.g., full email body).
**How to avoid:** The existing pattern is correct: send only metadata (event type, actor ID, resource ID, target users) in the NOTIFY payload. Full data is in the `events` durability table or fetched from the source service. The `EmitEvent()` function already warns at 7500 bytes.
**Warning signs:** slog warnings about payload size approaching limit.

### Pitfall 2: N+1 Queries in Inbox Population
**What goes wrong:** Each incoming event triggers individual queries to populate inbox_messages for each target user.
**Why it happens:** Naive event handler that processes one user at a time.
**How to avoid:** Batch operations. When an event arrives with multiple `target_user_ids`, create all inbox_messages in a single batch INSERT. Channel adapters should also support batch fetching.
**Warning signs:** Slow event processing, high DB connection count during bulk operations.

### Pitfall 3: Stale Inbox After Source Deletion
**What goes wrong:** A chat message is deleted, but the inbox still shows it. User clicks and gets a 404.
**Why it happens:** Inbox stores a copy/reference but doesn't handle source deletion events.
**How to avoid:** Register handlers for delete events (e.g., `chat.message.deleted`, `email.message.deleted`) that mark corresponding inbox_messages as deleted/hidden. DeepLink clicks should gracefully handle missing source items.
**Warning signs:** 404 errors when clicking inbox items.

### Pitfall 4: Circular Event Loops
**What goes wrong:** Inbox actions (mark-read, archive) emit events that trigger inbox consumers, creating infinite loops.
**Why it happens:** Inbox service emits events + listens to events on the same bus.
**How to avoid:** Use a separate "origin" field in events. Inbox-originated events should have `module_id: "inbox"` and the inbox consumer should skip events with that module. Alternatively, inbox internal operations should NOT emit to the events bus -- only source-affecting actions (like replying) should emit.
**Warning signs:** Exponential event growth, CPU spike on notification service.

### Pitfall 5: Snooze Timer Drift
**What goes wrong:** Snoozed items reappear late or not at all.
**Why it happens:** Background polling interval too long, or service restart loses in-memory timers.
**How to avoid:** Snooze is purely database-driven: `snoozed_until` column. A background goroutine polls every 60 seconds: `SELECT ... WHERE snoozed_until <= NOW() AND is_archived = false`. On un-snooze, set `snoozed_until = NULL` and `is_read = false`. This survives restarts.
**Warning signs:** Items staying invisible after snooze time, user complaints about missing items.

### Pitfall 6: Team Inbox Race Conditions on Claim
**What goes wrong:** Two team members claim the same item simultaneously.
**Why it happens:** No locking on the assignment operation.
**How to avoid:** Use `UPDATE inbox_messages SET assigned_to = $1 WHERE id = $2 AND assigned_to IS NULL RETURNING id`. If no rows returned, the item was already claimed. Return a "already assigned" error. Round-robin uses a `next_assignee_index` counter with atomic increment.
**Warning signs:** Duplicate processing of team inbox items.

### Pitfall 7: Routing Rule Evaluation Performance
**What goes wrong:** Complex rules with body content matching slow down inbox population.
**Why it happens:** Evaluating regex or keyword matches on every incoming message body.
**How to avoid:** Evaluate rules in priority order and stop at first match. Keep condition evaluation simple (string contains, not regex). Limit body keyword matching to the preview snippet (first 200 chars), not the full body. Cache compiled rules in memory with a 60s refresh.
**Warning signs:** High latency on inbox message creation, slow event processing.

## Code Examples

### Event Type Constants Extension

```go
// types.go - Add to existing constants in internal/notification/event/types.go

// Email events
const (
    EventEmailReceived     = "email.message.received"
    EventEmailSent         = "email.message.sent"
    EventEmailDeleted      = "email.message.deleted"
)

// Document events
const (
    EventDocumentUploaded  = "document.file.uploaded"
    EventDocumentShared    = "document.file.shared"
    EventDocumentVersioned = "document.file.versioned"
)

// Finance events
const (
    EventInvoiceCreated    = "biz.invoice.created"
    EventInvoiceSent       = "biz.invoice.sent"
    EventInvoiceOverdue    = "biz.invoice.overdue"
    EventPaymentReceived   = "biz.payment.received"
    EventQuoteCreated      = "biz.quote.created"
    EventDunningCreated    = "biz.dunning.created"
)

// HR events
const (
    EventLeaveRequested    = "hr.leave.requested"
    EventLeaveApproved     = "hr.leave.approved"
    EventLeaveRejected     = "hr.leave.rejected"
    EventShiftStarted      = "hr.shift.started"
    EventShiftEnded        = "hr.shift.ended"
)

// Inbox events (internal)
const (
    EventInboxItemCreated  = "inbox.item.created"
    EventInboxItemAssigned = "inbox.item.assigned"
)

// Module IDs
const (
    ModuleEmail    = "email"
    ModuleDocument = "document"
    ModuleBiz      = "biz"
    ModuleHR       = "hr"
    ModuleInbox    = "inbox"
)
```

### Migration: inbox_messages Table

```sql
-- 000047_create_inbox_tables.up.sql

-- Unified inbox messages (normalized from all channels)
CREATE TABLE inbox_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel VARCHAR(20) NOT NULL CHECK (channel IN ('email', 'chat', 'notification')),
    source_id VARCHAR(255) NOT NULL,
    sender_name VARCHAR(255) NOT NULL DEFAULT '',
    sender_id UUID,
    sender_email VARCHAR(255),
    subject VARCHAR(500) NOT NULL DEFAULT '',
    preview TEXT NOT NULL DEFAULT '',
    is_read BOOLEAN NOT NULL DEFAULT false,
    is_starred BOOLEAN NOT NULL DEFAULT false,
    is_archived BOOLEAN NOT NULL DEFAULT false,
    snoozed_until TIMESTAMPTZ,
    assigned_to UUID REFERENCES users(id),
    team_inbox_id UUID,
    tags TEXT[] NOT NULL DEFAULT '{}',
    deep_link VARCHAR(1000) NOT NULL DEFAULT '',
    crm_contact_id UUID,
    metadata JSONB,
    received_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Fast lookup: user's unread inbox
CREATE INDEX idx_inbox_messages_user_unread ON inbox_messages(user_id, received_at DESC)
    WHERE is_read = false AND is_archived = false AND snoozed_until IS NULL;

-- Fast lookup: user's inbox by channel
CREATE INDEX idx_inbox_messages_user_channel ON inbox_messages(user_id, channel, received_at DESC)
    WHERE is_archived = false;

-- Fast lookup: team inbox unassigned
CREATE INDEX idx_inbox_messages_team_unassigned ON inbox_messages(team_inbox_id, received_at DESC)
    WHERE assigned_to IS NULL AND is_archived = false;

-- Fast lookup: snoozed items due for requeue
CREATE INDEX idx_inbox_messages_snoozed ON inbox_messages(snoozed_until)
    WHERE snoozed_until IS NOT NULL AND is_archived = false;

-- Unique constraint: prevent duplicate source items per user
CREATE UNIQUE INDEX idx_inbox_messages_user_source ON inbox_messages(user_id, channel, source_id);

-- Starred items lookup
CREATE INDEX idx_inbox_messages_user_starred ON inbox_messages(user_id, received_at DESC)
    WHERE is_starred = true AND is_archived = false;

-- Team inboxes
CREATE TABLE team_inboxes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    description TEXT,
    assignment_mode VARCHAR(20) NOT NULL DEFAULT 'manual'
        CHECK (assignment_mode IN ('manual', 'round_robin')),
    visibility VARCHAR(20) NOT NULL DEFAULT 'open'
        CHECK (visibility IN ('open', 'private')),
    next_assignee_index INTEGER NOT NULL DEFAULT 0,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Team inbox members
CREATE TABLE team_inbox_members (
    team_inbox_id UUID NOT NULL REFERENCES team_inboxes(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL DEFAULT 'member'
        CHECK (role IN ('admin', 'member')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (team_inbox_id, user_id)
);

-- FK for inbox_messages.team_inbox_id
ALTER TABLE inbox_messages
    ADD CONSTRAINT fk_inbox_messages_team_inbox
    FOREIGN KEY (team_inbox_id) REFERENCES team_inboxes(id) ON DELETE SET NULL;

-- Routing rules
CREATE TABLE routing_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    channel VARCHAR(20),
    conditions JSONB NOT NULL,
    actions JSONB NOT NULL,
    priority INTEGER NOT NULL DEFAULT 100,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_routing_rules_active ON routing_rules(priority ASC)
    WHERE is_active = true;
```

### InboxService Proto Sketch

```protobuf
service InboxService {
    // Messages
    rpc ListMessages(ListMessagesRequest) returns (ListMessagesResponse);
    rpc GetMessage(GetMessageRequest) returns (GetMessageResponse);
    rpc MarkRead(MarkReadRequest) returns (MarkReadResponse);
    rpc MarkUnread(MarkUnreadRequest) returns (MarkUnreadResponse);
    rpc ToggleStar(ToggleStarRequest) returns (ToggleStarResponse);
    rpc ArchiveMessage(ArchiveMessageRequest) returns (ArchiveMessageResponse);
    rpc UnarchiveMessage(UnarchiveMessageRequest) returns (UnarchiveMessageResponse);
    rpc SnoozeMessage(SnoozeMessageRequest) returns (SnoozeMessageResponse);
    rpc UnsnoozeMessage(UnsnoozeMessageRequest) returns (UnsnoozeMessageResponse);
    rpc ReplyToMessage(ReplyToMessageRequest) returns (ReplyToMessageResponse);
    rpc AssignMessage(AssignMessageRequest) returns (AssignMessageResponse);
    rpc GetUnreadCount(GetUnreadCountRequest) returns (GetUnreadCountResponse);
    rpc BulkMarkRead(BulkMarkReadRequest) returns (BulkMarkReadResponse);
    rpc BulkArchive(BulkArchiveRequest) returns (BulkArchiveResponse);

    // Team Inboxes
    rpc CreateTeamInbox(CreateTeamInboxRequest) returns (CreateTeamInboxResponse);
    rpc UpdateTeamInbox(UpdateTeamInboxRequest) returns (UpdateTeamInboxResponse);
    rpc DeleteTeamInbox(DeleteTeamInboxRequest) returns (DeleteTeamInboxResponse);
    rpc ListTeamInboxes(ListTeamInboxesRequest) returns (ListTeamInboxesResponse);
    rpc AddTeamMember(AddTeamMemberRequest) returns (AddTeamMemberResponse);
    rpc RemoveTeamMember(RemoveTeamMemberRequest) returns (RemoveTeamMemberResponse);
    rpc ListTeamMembers(ListTeamMembersRequest) returns (ListTeamMembersResponse);
    rpc ClaimMessage(ClaimMessageRequest) returns (ClaimMessageResponse);

    // Routing Rules
    rpc CreateRoutingRule(CreateRoutingRuleRequest) returns (CreateRoutingRuleResponse);
    rpc UpdateRoutingRule(UpdateRoutingRuleRequest) returns (UpdateRoutingRuleResponse);
    rpc DeleteRoutingRule(DeleteRoutingRuleRequest) returns (DeleteRoutingRuleResponse);
    rpc ListRoutingRules(ListRoutingRulesRequest) returns (ListRoutingRulesResponse);
    rpc TestRoutingRule(TestRoutingRuleRequest) returns (TestRoutingRuleResponse);
}
```

### Snooze Background Worker

```go
// StartSnoozeWorker runs a background loop that un-snoozes due items.
func StartSnoozeWorker(ctx context.Context, repo MessageRepository, interval time.Duration) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            count, err := repo.UnsnoozeExpired(ctx)
            if err != nil {
                slog.Error("snooze worker failed", "error", err)
                continue
            }
            if count > 0 {
                slog.Info("unsnoozed expired items", "count", count)
                // Emit events for each unsnoozed item so gateway pushes to WebSocket
            }
        }
    }
}

// UnsnoozeExpired marks snoozed items as visible again.
// SQL: UPDATE inbox_messages SET snoozed_until = NULL, is_read = false, updated_at = NOW()
//      WHERE snoozed_until <= NOW() AND is_archived = false
//      RETURNING id, user_id
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Polling each service for changes | Event-driven via pg_notify | Phase 4 (2026-02-07) | Real-time updates without polling overhead |
| Full message body in NOTIFY payload | Metadata-only payload + events table | Phase 4 design decision | Avoids 8KB limit, clean separation |
| Per-service notification channels | Single "events" channel with type routing | Phase 4 design decision | Simpler listener management |
| Separate event emitter per service | Shared `event.EmitEvent()` function | Phase 4 | DRY, consistent across all services |

**Already established patterns in this project:**
- Dual-write: events table + pg_notify (durability + real-time)
- EventBus with wildcard + type-specific handlers
- PGEventEmitter per service package with nil-safe SetEventEmitter()
- notification_delivery channel for WebSocket gateway push
- ProcessBacklog for catch-up after service restart

## Open Questions

1. **Should inbox_messages store the full thread or just individual messages?**
   - What we know: Email already has threading via JWZ (References/In-Reply-To). Chat DMs are individual messages. Notifications are single items.
   - What's unclear: Whether to show threaded email conversations as one inbox item or separate items per message.
   - Recommendation: One inbox item per top-level thread/conversation. The `source_id` for email should be the `thread_id`, not the individual `message_id`. The preview shows the latest message in the thread. This matches Gmail behavior and reduces inbox clutter.

2. **How should the inbox handle high-volume chat channels?**
   - What we know: The user decision specifies "Chat DMs/@mentions", not all channel messages.
   - What's unclear: Whether channel messages (non-DM, non-mention) should appear in the inbox at all.
   - Recommendation: Only DMs and @mentions appear in the unified inbox. Regular channel messages stay in the Chat module only. This matches the "Kommunikation = external/cross-module" vs "Chat = internal" distinction.

3. **Where should the InboxService run?**
   - What we know: Notification service on :50054 already handles events. Inbox is tightly coupled.
   - What's unclear: Whether adding ~25 RPCs to the notification binary is acceptable.
   - Recommendation: Co-host InboxService in the notification binary. The notification binary already has the EventBus, event handlers, and pg_notify listeners. Adding inbox makes it a unified "communication hub" service. The alternative (separate binary on :50059) adds deployment overhead for little benefit.

4. **Auto-reply: should it happen in the inbox service or the email service?**
   - What we know: Auto-reply is a routing rule action. The inbox service evaluates rules.
   - What's unclear: Whether the inbox should call the email service to send the reply, or handle it itself.
   - Recommendation: Inbox evaluates the rule and calls the email service's Send RPC via gRPC to actually send the auto-reply. Inbox should not have SMTP capabilities. This keeps email sending centralized.

## Sources

### Primary (HIGH confidence)
- Project codebase analysis: `backend/internal/notification/event/` -- existing EventBus, EmitEvent, EventTypeRegistry
- Project codebase analysis: `backend/internal/notification/notification/service.go` -- ProcessEvent handler pattern
- Project codebase analysis: `backend/internal/chat/message/event_emitter.go`, `backend/internal/crm/deal/event_emitter.go`, `backend/internal/work/task/event_emitter.go` -- established PGEventEmitter pattern
- Project codebase analysis: `backend/migrations/000020_create_event_types.up.sql`, `000021_create_notifications.up.sql` -- existing event/notification schema
- Project codebase analysis: `backend/cmd/notification/main.go` -- EventBus wiring in notification binary
- Project codebase analysis: `backend/cmd/gateway/main.go` -- notification delivery listener, WebSocket push

### Secondary (MEDIUM confidence)
- [PostgreSQL NOTIFY documentation](https://www.postgresql.org/docs/current/sql-notify.html) -- 8000 byte payload limit, transactional semantics
- [Enterprise Integration Patterns: Channel Adapter](https://www.enterpriseintegrationpatterns.com/patterns/messaging/ChannelAdapter.html) -- adapter pattern for heterogeneous sources
- [PostgreSQL as message bus](https://thinhdanggroup.github.io/postgres-as-a-message-bus/) -- event table + pg_notify dual-write pattern validation
- [Go and Postgres LISTEN/NOTIFY](https://brojonat.com/posts/go-postgres-listen-notify/) -- Go implementation patterns with pgx

### Tertiary (LOW confidence)
- [Unified inbox architecture survey 2025](https://www.deemerge.ai/post/best-unified-inbox-apps-in-2025) -- market survey of unified inbox features
- [Grule Rule Engine](https://github.com/hyperjumptech/grule-rule-engine) -- Go rule engine (considered but rejected as overkill)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all libraries already in use, no new dependencies needed
- Architecture: HIGH -- event infrastructure already exists, extending it follows established patterns
- Pitfalls: HIGH -- based on analysis of existing codebase patterns and common event-driven architecture issues
- Routing rules: MEDIUM -- custom condition evaluator is straightforward but needs careful Phase 16 reuse design

**Research date:** 2026-02-19
**Valid until:** 2026-03-19 (stable domain, no fast-moving dependencies)

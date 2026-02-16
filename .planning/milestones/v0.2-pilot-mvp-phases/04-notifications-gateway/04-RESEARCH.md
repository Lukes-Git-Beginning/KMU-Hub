# Phase 4: Notifications + Gateway Modernization - Research

**Researched:** 2026-02-07
**Domain:** Event-driven notification system + API gateway architecture (Go microservices)
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Notification UX:**
- Dropdown panel accessed via bell icon (like GitHub/Slack) -- stays on current page
- Smart grouping: collapse similar events (e.g., "5 new messages in #general") with expand/collapse
- Capped badge count: exact number up to threshold, then "99+" style cap
- Click marks individual notification as read and navigates to source; "Mark all as read" button available; viewing the panel does NOT auto-mark as read

**Notification Preferences:**
- Two-level granularity: module-level defaults + per-event-type overrides
- Quiet hours: scheduled recurring DND (e.g., 18:00-08:00) as default + manual toggle override for ad-hoc situations
- Per-resource muting: users can mute specific chat channels, CRM pipelines, or any future module resource -- overrides event-type settings for that resource

**Event Taxonomy:**
- Chat events: default to mentions + DMs only; users can opt into "all messages" per channel
- 3-tier priority system:
  - **Urgent**: always delivered, even during DND (e.g., system alerts, direct escalations)
  - **Normal**: delivered per user preferences (e.g., @mentions, deal assignments)
  - **Low**: in-app only, batched/collapsed (e.g., informational updates)
- CRM events: notify on assignment/ownership changes and deal stage transitions by default
- Generic event bus: module-agnostic event type registry -- any module registers event types + default preferences; future modules plug in without notification service changes

**Desktop Push Behavior:**
- Click-to-navigate: clicking a desktop push notification opens the Hub and deep-links directly to the source item (message, deal, task)
- Quick action buttons: 1-2 action buttons on push notifications (e.g., "Reply" on messages, "Mark as read")
- Notification sounds: enabled by default with preset options; different sounds per priority tier; user can disable or choose presets
- System tray: dot indicator (colored dot when unread notifications exist) -- subtle, not a badge count

### Claude's Discretion

- Delivery channels selection (in-app bell + desktop push baseline; email digest if it makes sense for desktop-first app)
- Gateway refactoring technical approach (lazy gRPC connections, per-service route handlers, graceful degradation patterns)
- Event bus implementation (PostgreSQL LISTEN/NOTIFY vs other patterns)
- Notification storage schema and retention policy
- Smart grouping algorithm details
- Sound preset selection

### Deferred Ideas (OUT OF SCOPE)

None -- discussion stayed within phase scope
</user_constraints>

## Summary

This phase has two orthogonal but strategically linked workstreams: (1) refactoring the gateway from a monolithic handler with eager gRPC connections into a modular, lazy-connecting router that gracefully degrades per-service, and (2) building a centralized notification service with event bus, preference engine, and real-time delivery over the existing WebSocket infrastructure.

The gateway currently hardcodes all three gRPC clients (auth, CRM, chat) in `main.go`, creates them eagerly at startup (crashing if any is unavailable), and routes everything through a single `GatewayHandler` struct in `http.go`. The refactoring introduces a `ServiceRegistry` pattern with lazy gRPC connection factories, per-service route handler modules, and a 503 fallback when a backend service is unreachable. This is critical because Phases 5-13 will add up to 4 more backend services (notification, work, biz, automation).

For notifications, PostgreSQL LISTEN/NOTIFY is the right event bus for this scale (single-instance, all services share one PostgreSQL). The notification service listens on a single `events` channel, receives event payloads from any service that does `pg_notify('events', ...)`, evaluates user preferences and quiet hours, stores notifications, and pushes them to connected clients via WebSocket. The existing `WebSocketHub` already supports `sendToUser()` -- it just needs new message types for notification events.

**Primary recommendation:** Build the notification service as a new gRPC microservice (port 50054) that owns the event bus listener, notification storage, and preference engine. The gateway refactoring should happen first since the notification service will be the first new service to plug into the modernized gateway.

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| jackc/pgx/v5 | v5.8.0 | PostgreSQL driver (already in use) | Project standard, supports LISTEN/NOTIFY natively |
| jackc/pgxlisten | v0.0.x (pre-v1) | Higher-level LISTEN/NOTIFY wrapper | Built by pgx author, handles reconnection, single-connection pattern |
| google.golang.org/grpc | v1.78.0 | gRPC framework (already in use) | Project standard, `grpc.NewClient` is lazy by default |
| go-chi/chi/v5 | v5.2.4 | HTTP router (already in use) | Project standard, `Mount()` for modular route registration |
| coder/websocket | v1.8.14 | WebSocket (already in use) | Already powering chat real-time; extend for notifications |
| google.golang.org/protobuf | v1.36.11 | Protobuf (already in use) | Project standard |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| jackc/pgx/v5 (pgconn) | v5.8.0 | Low-level notification access | Fallback if pgxlisten proves too immature |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| pgxlisten | Raw pgx conn.WaitForNotification | pgxlisten adds reconnection + backlog handling; raw pgx is simpler but needs manual reconnection logic |
| PostgreSQL LISTEN/NOTIFY | NATS or Redis PubSub | Overkill for single-instance; adds operational dependency; PG is already there |
| PostgreSQL LISTEN/NOTIFY | Redis Streams | Redis already in stack but only for rate limiting; LISTEN/NOTIFY keeps single source of truth in PG |

**Recommendation on pgxlisten vs raw pgx:** Use raw pgx `conn.WaitForNotification()` with a custom reconnection wrapper. pgxlisten is pre-v1 with no tagged releases (latest: commit hash from Aug 2025). The reconnection logic is straightforward (10-20 lines) and avoids depending on an unstable library. The Notifier Pattern (Brandur) provides an excellent reference implementation.

**Installation:**
```bash
# No new dependencies needed -- all libraries are already in go.mod
# pgxlisten NOT recommended due to pre-v1 instability
```

## Architecture Patterns

### Recommended Project Structure

```
backend/
├── cmd/
│   └── notification/
│       └── main.go                    # Notification service entry point
├── internal/
│   ├── notification/
│   │   ├── event/
│   │   │   ├── bus.go                 # PostgreSQL LISTEN/NOTIFY event bus
│   │   │   ├── bus_test.go
│   │   │   ├── registry.go           # Event type registry (module-agnostic)
│   │   │   ├── registry_test.go
│   │   │   └── types.go              # Event type constants + payload structs
│   │   ├── preference/
│   │   │   ├── service.go            # Preference evaluation engine
│   │   │   ├── service_test.go
│   │   │   ├── repository.go         # Interface
│   │   │   ├── postgres_repository.go
│   │   │   └── errors.go
│   │   ├── notification/
│   │   │   ├── service.go            # Notification CRUD + grouping
│   │   │   ├── service_test.go
│   │   │   ├── repository.go
│   │   │   ├── postgres_repository.go
│   │   │   ├── grouper.go            # Smart grouping algorithm
│   │   │   ├── grouper_test.go
│   │   │   └── errors.go
│   │   └── delivery/
│   │       ├── dispatcher.go         # Routes notifications to channels
│   │       └── dispatcher_test.go
│   ├── gateway/
│   │   ├── registry.go               # ServiceRegistry: lazy gRPC connections
│   │   ├── registry_test.go
│   │   ├── route_auth.go             # Auth route handler module
│   │   ├── route_crm.go              # CRM route handler module
│   │   ├── route_chat.go             # Chat route handler module
│   │   ├── route_notification.go     # Notification route handler module
│   │   └── route_health.go           # Health/metrics routes
│   ├── server/
│   │   ├── notification_grpc.go      # Notification gRPC server
│   │   └── websocket.go              # Extended with notification events
│   └── models/
│       ├── notification.go            # Notification, NotificationPreference models
│       └── event.go                   # Event type models
├── proto/
│   └── notification/
│       └── v1/
│           └── notification.proto     # Notification service protobuf
└── migrations/
    ├── 000020_create_event_types.up.sql
    ├── 000020_create_event_types.down.sql
    ├── 000021_create_notifications.up.sql
    ├── 000021_create_notifications.down.sql
    ├── 000022_create_notification_preferences.up.sql
    └── 000022_create_notification_preferences.down.sql
```

### Pattern 1: Service Registry with Lazy gRPC Connections

**What:** A `ServiceRegistry` that creates gRPC client connections on-demand (lazy), caches them, and returns a 503 error if a service is unreachable rather than crashing the gateway.

**When to use:** Gateway initialization, replacing the current eager connection pattern in `cmd/gateway/main.go`.

**Example:**
```go
// Source: grpc-go docs -- grpc.NewClient is lazy by default
// backend/internal/gateway/registry.go

type ServiceConnection struct {
    conn    *grpc.ClientConn
    address string
    mu      sync.RWMutex
    healthy bool
}

type ServiceRegistry struct {
    services map[string]*ServiceConnection
    mu       sync.RWMutex
}

func NewServiceRegistry() *ServiceRegistry {
    return &ServiceRegistry{
        services: make(map[string]*ServiceConnection),
    }
}

// Register adds a service with its address but does NOT connect yet
func (r *ServiceRegistry) Register(name, address string) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.services[name] = &ServiceConnection{address: address}
}

// GetConnection returns a lazy gRPC connection, creating it on first use
func (r *ServiceRegistry) GetConnection(name string) (*grpc.ClientConn, error) {
    r.mu.RLock()
    svc, ok := r.services[name]
    r.mu.RUnlock()
    if !ok {
        return nil, fmt.Errorf("service %s not registered", name)
    }

    svc.mu.RLock()
    if svc.conn != nil {
        svc.mu.RUnlock()
        return svc.conn, nil
    }
    svc.mu.RUnlock()

    svc.mu.Lock()
    defer svc.mu.Unlock()
    // Double-check after acquiring write lock
    if svc.conn != nil {
        return svc.conn, nil
    }

    // grpc.NewClient does NOT perform I/O -- connection happens on first RPC
    conn, err := grpc.NewClient(
        svc.address,
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create client for %s: %w", name, err)
    }
    svc.conn = conn
    return conn, nil
}

// Close cleans up all connections
func (r *ServiceRegistry) Close() {
    r.mu.RLock()
    defer r.mu.RUnlock()
    for _, svc := range r.services {
        svc.mu.Lock()
        if svc.conn != nil {
            _ = svc.conn.Close()
        }
        svc.mu.Unlock()
    }
}
```

### Pattern 2: Modular Route Handler Registration

**What:** Each backend service gets its own route handler file that registers routes independently, replacing the monolithic `GatewayHandler` struct.

**When to use:** Breaking up the 900+ line `http.go` into service-specific modules.

**Example:**
```go
// backend/internal/gateway/route_crm.go

// RouteRegistrar is the interface each service module implements
type RouteRegistrar interface {
    RegisterRoutes(r chi.Router, auth func(http.Handler) http.Handler)
    ServiceName() string
}

type CRMRoutes struct {
    registry *ServiceRegistry
}

func NewCRMRoutes(registry *ServiceRegistry) *CRMRoutes {
    return &CRMRoutes{registry: registry}
}

func (cr *CRMRoutes) ServiceName() string { return "crm" }

func (cr *CRMRoutes) RegisterRoutes(r chi.Router, auth func(http.Handler) http.Handler) {
    r.Route("/api/v1/contacts", func(r chi.Router) {
        r.Use(auth)
        r.Get("/", cr.handleListContacts)
        r.Post("/", cr.handleCreateContact)
        // ...
    })
}

func (cr *CRMRoutes) getCRMClient() (crmv1.CRMServiceClient, error) {
    conn, err := cr.registry.GetConnection("crm")
    if err != nil {
        return nil, err
    }
    return crmv1.NewCRMServiceClient(conn), nil
}

func (cr *CRMRoutes) handleListContacts(w http.ResponseWriter, r *http.Request) {
    client, err := cr.getCRMClient()
    if err != nil {
        response.Error(w, http.StatusServiceUnavailable, "crm service unavailable")
        return
    }
    // ... proceed with gRPC call
}
```

### Pattern 3: PostgreSQL Event Bus (Notifier Pattern)

**What:** A single dedicated PostgreSQL connection listens on an `events` channel. When any service writes an event (via `pg_notify`), the bus distributes it to registered handlers. Services emit events by calling `pg_notify('events', '...')` in their existing transactions.

**When to use:** Notification service startup; any service that needs to emit events.

**Example:**
```go
// backend/internal/notification/event/bus.go

type EventBus struct {
    connString    string
    handlers      map[string][]EventHandler
    mu            sync.RWMutex
    reconnectWait time.Duration
}

type Event struct {
    Type       string          `json:"type"`       // e.g., "chat.mention", "crm.deal.stage_changed"
    Priority   string          `json:"priority"`   // "urgent", "normal", "low"
    ActorID    string          `json:"actor_id"`
    ResourceID string          `json:"resource_id"`
    ModuleID   string          `json:"module_id"`  // "chat", "crm", etc.
    Payload    json.RawMessage `json:"payload"`
    Timestamp  time.Time       `json:"timestamp"`
}

type EventHandler func(ctx context.Context, event Event) error

func (bus *EventBus) Listen(ctx context.Context) error {
    for {
        err := bus.listenLoop(ctx)
        if ctx.Err() != nil {
            return ctx.Err()
        }
        slog.Warn("event bus connection lost, reconnecting",
            "error", err,
            "retry_in", bus.reconnectWait,
        )
        time.Sleep(bus.reconnectWait)
    }
}

func (bus *EventBus) listenLoop(ctx context.Context) error {
    conn, err := pgx.Connect(ctx, bus.connString)
    if err != nil {
        return fmt.Errorf("connect: %w", err)
    }
    defer conn.Close(ctx)

    _, err = conn.Exec(ctx, "LISTEN events")
    if err != nil {
        return fmt.Errorf("listen: %w", err)
    }

    for {
        notification, err := conn.WaitForNotification(ctx)
        if err != nil {
            return fmt.Errorf("wait: %w", err)
        }

        var event Event
        if err := json.Unmarshal([]byte(notification.Payload), &event); err != nil {
            slog.Error("failed to unmarshal event", "error", err, "payload", notification.Payload)
            continue
        }

        bus.dispatch(ctx, event)
    }
}

// Emit is called by services within their existing transactions
// Example SQL: SELECT pg_notify('events', $1)
func EmitEventSQL(eventType, priority, actorID, resourceID, moduleID string, payload json.RawMessage) string {
    // Returns SQL for embedding in service transactions
    return "SELECT pg_notify('events', $1)"
}
```

### Pattern 4: Notification Preference Evaluation Pipeline

**What:** A pipeline that determines whether a notification should be delivered to a user, checking: (1) resource muting, (2) event type preference, (3) module defaults, (4) quiet hours, (5) priority override.

**When to use:** Before creating/delivering each notification.

**Example:**
```go
// backend/internal/notification/preference/service.go

type DeliveryDecision struct {
    Deliver      bool
    InApp        bool
    DesktopPush  bool
    Sound        string // "" = no sound, "default", "urgent", etc.
    Reason       string // for debugging
}

func (s *Service) Evaluate(ctx context.Context, userID string, event Event) (*DeliveryDecision, error) {
    // 1. Check priority -- Urgent always delivers
    if event.Priority == PriorityUrgent {
        return &DeliveryDecision{
            Deliver: true, InApp: true, DesktopPush: true,
            Sound: "urgent", Reason: "urgent priority bypasses all filters",
        }, nil
    }

    // 2. Check resource muting (e.g., muted channel)
    muted, err := s.repo.IsResourceMuted(ctx, userID, event.ModuleID, event.ResourceID)
    if err != nil {
        return nil, err
    }
    if muted {
        return &DeliveryDecision{Deliver: false, Reason: "resource muted"}, nil
    }

    // 3. Check event-type preference
    pref, err := s.repo.GetEventTypePreference(ctx, userID, event.Type)
    if err != nil && !errors.Is(err, ErrNotFound) {
        return nil, err
    }

    // 4. Fall back to module default if no event-type override
    if pref == nil {
        pref, err = s.repo.GetModuleDefault(ctx, userID, event.ModuleID)
        // ... handle fallback to system default
    }

    // 5. Check quiet hours (only for Normal priority)
    if event.Priority == PriorityNormal {
        inQuietHours, err := s.isInQuietHours(ctx, userID)
        if err != nil {
            return nil, err
        }
        if inQuietHours {
            return &DeliveryDecision{
                Deliver: true, InApp: true, DesktopPush: false,
                Reason: "quiet hours -- in-app only",
            }, nil
        }
    }

    // 6. Low priority: in-app only, no desktop push
    if event.Priority == PriorityLow {
        return &DeliveryDecision{
            Deliver: true, InApp: true, DesktopPush: false,
            Reason: "low priority -- in-app only",
        }, nil
    }

    return &DeliveryDecision{
        Deliver: true, InApp: pref.InApp, DesktopPush: pref.DesktopPush,
        Sound: pref.Sound,
    }, nil
}
```

### Anti-Patterns to Avoid

- **Eager gRPC connections at gateway startup:** The current pattern crashes the gateway if any backend service is down. Use lazy connections instead.
- **Monolithic handler file:** The current 900+ line `http.go` makes adding services painful. Split into per-service route modules.
- **Direct WebSocket coupling in services:** Services should emit events to the bus, not directly push to WebSocket. The notification service owns delivery.
- **N+1 preference lookups:** Pre-fetch user preferences in batch when evaluating multiple notifications, not one-by-one.
- **Storing full event payload in NOTIFY:** PostgreSQL NOTIFY payload is limited to 8000 bytes. Send only event metadata (type + IDs), let the notification service fetch full details from the source service if needed.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| LISTEN reconnection | Custom TCP reconnect logic | Simple reconnect loop around pgx (see Pattern 3) | The pattern is well-established: connect, listen, loop, reconnect on error |
| Event serialization | Custom binary protocol | JSON in NOTIFY payload | 8000 byte limit is fine for event metadata; JSON is debuggable |
| Notification grouping | Complex real-time aggregation | Time-window batching with simple key-based grouping | Group by (event_type + resource_id) within a configurable window (e.g., 30s) |
| Preference inheritance | Custom tree traversal | Simple fallback chain: resource mute > event-type > module default > system default | Flat hierarchy, 3-4 lookups max |
| gRPC health checking | Custom ping/pong | grpc.NewClient lazy connect + WaitForReady per-call | grpc-go handles connection state internally |

**Key insight:** The complexity here is in the preference evaluation pipeline and the gateway refactoring, not in the event transport. PostgreSQL LISTEN/NOTIFY handles the transport; the real work is deciding what to deliver to whom and making the gateway modular.

## Common Pitfalls

### Pitfall 1: NOTIFY Payload Size Overflow

**What goes wrong:** Attempting to send full event data (including message content, user details) in the NOTIFY payload exceeds the 8000-byte limit.
**Why it happens:** Developers treat NOTIFY like a message queue instead of a signal mechanism.
**How to avoid:** Send only event metadata (type, IDs, priority, timestamp) in the NOTIFY payload. The notification service fetches rich data from the source service via gRPC when needed.
**Warning signs:** Events with large payloads (file uploads, long messages) silently fail or get truncated.

### Pitfall 2: NOTIFY is Not a Durable Queue

**What goes wrong:** Events are lost when the notification service is down because LISTEN/NOTIFY is ephemeral -- only connected listeners receive notifications.
**Why it happens:** PostgreSQL NOTIFY does not queue events for disconnected listeners.
**How to avoid:** Dual approach: (1) services write events to an `events` table AND call `pg_notify()` in the same transaction, (2) the notification service has a catch-up mechanism that processes unprocessed events from the table on reconnection. The NOTIFY is just a "wake up, there's work" signal.
**Warning signs:** Notifications missing after notification service restarts.

### Pitfall 3: Gateway Service Unavailability Cascade

**What goes wrong:** One unavailable backend service makes the entire gateway unresponsive because all routes share a single handler.
**Why it happens:** Current monolithic `GatewayHandler` couples all services together.
**How to avoid:** Per-service route handlers that independently check service availability and return 503 only for their own routes.
**Warning signs:** All API endpoints return errors when only one backend service is down.

### Pitfall 4: Notification Storm from Batch Operations

**What goes wrong:** Bulk operations (e.g., importing 1000 contacts, reassigning 50 deals) generate thousands of notifications simultaneously, overwhelming users and the system.
**Why it happens:** Each entity change triggers an event, which triggers a notification, with no batching.
**How to avoid:** (1) Event deduplication in the bus (same type + resource within time window), (2) smart grouping collapses similar notifications ("50 contacts imported"), (3) Low priority events are batched by default.
**Warning signs:** Notification panel shows hundreds of identical notifications after a bulk operation.

### Pitfall 5: Quiet Hours Timezone Confusion

**What goes wrong:** Quiet hours calculated in server time instead of user's local time, causing DND at wrong hours.
**Why it happens:** Server stores UTC but evaluates DND without converting to user timezone.
**How to avoid:** Store user's timezone in preferences (IANA timezone, e.g., "Europe/Berlin"). Evaluate DND by converting current UTC to user's local time. DACH users are typically in CET/CEST.
**Warning signs:** Users in different timezones get notifications during their quiet hours.

### Pitfall 6: WebSocket Hub Becomes Notification Bottleneck

**What goes wrong:** The existing WebSocket hub's channel-based subscription model doesn't map cleanly to user-targeted notifications.
**Why it happens:** The hub was designed for chat channels (subscribe to channel -> get channel messages). Notifications are per-user, not per-channel.
**How to avoid:** Extend the hub with a direct user notification channel. Notifications are delivered via `sendToUser()` (already exists) with a new `notification.new` message type. No need to subscribe to anything -- user-targeted by default.
**Warning signs:** Having to create fake "notification channels" to use the subscription model.

## Code Examples

### Emitting Events from Existing Services (CRM Example)

```go
// Source: PostgreSQL docs -- pg_notify() function
// backend/internal/crm/deal/postgres_repository.go

// In the deal stage transition transaction, add event emission:
func (r *PostgresRepository) MoveDealToStage(ctx context.Context, dealID, stageID, userID uuid.UUID) error {
    tx, err := r.pool.Begin(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback(ctx)

    // ... existing stage transition logic ...

    // Emit event in the same transaction (atomicity guaranteed)
    eventPayload := map[string]interface{}{
        "type":        "crm.deal.stage_changed",
        "priority":    "normal",
        "actor_id":    userID.String(),
        "resource_id": dealID.String(),
        "module_id":   "crm",
        "payload": map[string]interface{}{
            "deal_id":      dealID.String(),
            "new_stage_id": stageID.String(),
            "owner_id":     deal.OwnerID.String(),
        },
        "timestamp": time.Now().UTC(),
    }
    payloadJSON, _ := json.Marshal(eventPayload)
    _, err = tx.Exec(ctx, "SELECT pg_notify('events', $1)", string(payloadJSON))
    if err != nil {
        return err
    }

    return tx.Commit(ctx)
}
```

### Notification Storage Schema

```sql
-- Source: Research-derived schema based on project conventions
-- backend/migrations/000020_create_event_types.up.sql

CREATE TABLE event_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    module_id VARCHAR(50) NOT NULL,           -- 'chat', 'crm', etc.
    event_key VARCHAR(100) NOT NULL UNIQUE,   -- 'chat.mention', 'crm.deal.stage_changed'
    display_name VARCHAR(200) NOT NULL,       -- 'Mentioned in chat'
    description TEXT,
    default_priority VARCHAR(10) NOT NULL DEFAULT 'normal'
        CHECK (default_priority IN ('urgent', 'normal', 'low')),
    default_in_app BOOLEAN NOT NULL DEFAULT true,
    default_desktop_push BOOLEAN NOT NULL DEFAULT true,
    default_sound VARCHAR(50) DEFAULT 'default',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_event_types_module_id ON event_types(module_id);

-- Seed initial event types
INSERT INTO event_types (module_id, event_key, display_name, default_priority) VALUES
    ('chat', 'chat.mention', 'Mentioned in message', 'normal'),
    ('chat', 'chat.dm.new', 'New direct message', 'normal'),
    ('chat', 'chat.channel.message', 'New channel message', 'low'),
    ('crm', 'crm.deal.stage_changed', 'Deal stage changed', 'normal'),
    ('crm', 'crm.deal.assigned', 'Deal assigned to you', 'normal'),
    ('crm', 'crm.contact.assigned', 'Contact assigned to you', 'normal'),
    ('system', 'system.alert', 'System alert', 'urgent');
```

```sql
-- backend/migrations/000021_create_notifications.up.sql

CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_type_key VARCHAR(100) NOT NULL,     -- denormalized for fast queries
    module_id VARCHAR(50) NOT NULL,
    priority VARCHAR(10) NOT NULL DEFAULT 'normal'
        CHECK (priority IN ('urgent', 'normal', 'low')),
    actor_id UUID,                            -- who triggered this
    resource_id VARCHAR(255),                 -- what was affected (UUID or composite key)
    title VARCHAR(500) NOT NULL,
    body TEXT,
    deep_link VARCHAR(1000),                  -- e.g., '/chat/channels/{id}', '/crm/deals/{id}'
    group_key VARCHAR(255),                   -- for smart grouping (e.g., 'chat.channel.{channel_id}')
    is_read BOOLEAN NOT NULL DEFAULT false,
    read_at TIMESTAMPTZ,
    delivered_desktop BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notifications_user_unread ON notifications(user_id, is_read, created_at DESC)
    WHERE is_read = false;
CREATE INDEX idx_notifications_user_created ON notifications(user_id, created_at DESC);
CREATE INDEX idx_notifications_group_key ON notifications(user_id, group_key, created_at DESC);

-- Events table for durability (catch-up on reconnect)
CREATE TABLE events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type_key VARCHAR(100) NOT NULL,
    module_id VARCHAR(50) NOT NULL,
    priority VARCHAR(10) NOT NULL DEFAULT 'normal',
    actor_id UUID,
    resource_id VARCHAR(255),
    payload JSONB,
    processed BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_events_unprocessed ON events(created_at ASC) WHERE processed = false;
```

```sql
-- backend/migrations/000022_create_notification_preferences.up.sql

CREATE TABLE notification_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    -- Scope: event-type level or module level
    event_type_key VARCHAR(100),              -- NULL = module-level default
    module_id VARCHAR(50),                    -- NULL only for global preferences
    -- Delivery settings
    in_app BOOLEAN NOT NULL DEFAULT true,
    desktop_push BOOLEAN NOT NULL DEFAULT true,
    sound VARCHAR(50) DEFAULT 'default',
    -- Constraints
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, event_type_key),          -- one preference per event type per user
    UNIQUE(user_id, module_id) WHERE event_type_key IS NULL  -- one module default per user
);

CREATE INDEX idx_notification_preferences_user ON notification_preferences(user_id);

-- Resource muting (per-channel, per-pipeline, etc.)
CREATE TABLE notification_mutes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    module_id VARCHAR(50) NOT NULL,           -- 'chat', 'crm', etc.
    resource_id VARCHAR(255) NOT NULL,        -- channel UUID, pipeline UUID, etc.
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, module_id, resource_id)
);

CREATE INDEX idx_notification_mutes_user ON notification_mutes(user_id, module_id);

-- Quiet hours configuration
CREATE TABLE notification_quiet_hours (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    -- Scheduled quiet hours
    start_time TIME NOT NULL DEFAULT '18:00',  -- local time
    end_time TIME NOT NULL DEFAULT '08:00',    -- local time (next day if end < start)
    timezone VARCHAR(50) NOT NULL DEFAULT 'Europe/Berlin',
    days_of_week INTEGER[] NOT NULL DEFAULT '{1,2,3,4,5,6,7}', -- 1=Monday, 7=Sunday
    enabled BOOLEAN NOT NULL DEFAULT false,
    -- Manual DND override
    manual_dnd BOOLEAN NOT NULL DEFAULT false,
    manual_dnd_until TIMESTAMPTZ,             -- NULL = indefinite until manually toggled
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id)
);

-- Retention: auto-delete notifications older than 90 days
-- Implement via scheduled job or pg_cron
```

### WebSocket Notification Message Types

```go
// Source: Extending existing WebSocket hub pattern
// backend/internal/server/websocket.go (additions)

const (
    // Notification events (Server -> Client)
    WSNotificationNew     = "notification.new"
    WSNotificationRead    = "notification.read"
    WSNotificationReadAll = "notification.read_all"
    WSUnreadCountUpdate   = "notification.unread_count"

    // Notification actions (Client -> Server)
    WSNotificationMarkRead    = "notification.mark_read"
    WSNotificationMarkAllRead = "notification.mark_all_read"
)

// NotificationMessage sent to connected clients
type NotificationWSMessage struct {
    Type         string          `json:"type"`
    Notification json.RawMessage `json:"notification,omitempty"`
    UnreadCount  int             `json:"unread_count,omitempty"`
    IDs          []string        `json:"ids,omitempty"`
}
```

## Discretion Recommendations

### Delivery Channels

**Recommendation:** In-app bell + desktop push only. Skip email digest for v1.

**Rationale:** This is a desktop-first application. Email digest adds significant complexity (email template system, scheduling, batching) for minimal value when users are already in the app. The desktop push notification via Electron provides background awareness. Email digest can be added later if customers request it (Phase 12 automation engine could handle this as a workflow).

### Event Bus Implementation

**Recommendation:** PostgreSQL LISTEN/NOTIFY with an events table for durability.

**Rationale:** (1) PostgreSQL is already in the stack, no new dependencies. (2) Single-instance deployment means no need for distributed message bus. (3) The events table provides durability that LISTEN/NOTIFY alone lacks. (4) All services already have PostgreSQL connections. The dual-write pattern (events table + pg_notify in same transaction) gives atomicity. NATS/RabbitMQ would be overkill and add operational complexity for a solo developer.

### Notification Retention Policy

**Recommendation:** 90-day retention, auto-cleanup via scheduled PostgreSQL job.

**Rationale:** Notifications are ephemeral by nature. 90 days covers quarterly review cycles common in DACH KMUs. Implement cleanup as a simple SQL DELETE with a PostgreSQL-native scheduled mechanism (pg_cron in production, or a goroutine in the notification service for development).

### Smart Grouping Algorithm

**Recommendation:** Time-window key-based grouping.

**Details:**
1. Each notification has a `group_key` (e.g., `chat.channel.{channel_id}`, `crm.deal.{deal_id}`)
2. When a new notification arrives with a group_key that matches an unread notification from the last 30 seconds, increment the group count instead of creating a new notification
3. The frontend displays grouped notifications as "5 new messages in #general" with expand/collapse
4. Group collapse is calculated at query time, not stored (keeps schema simple)

### Sound Presets

**Recommendation:** 3 presets mapped to priority tiers.

| Priority | Sound | Description |
|----------|-------|-------------|
| Urgent | `alert` | Short, attention-grabbing tone |
| Normal | `default` | Soft chime |
| Low | (none) | Silent, in-app only |

Sound files are bundled with the Electron app (Phase 5). The backend stores the sound name in preferences; the frontend resolves it to the actual audio file. For Phase 4, the backend only stores the preference -- actual playback is Phase 5 Electron implementation.

### Gateway Refactoring Approach

**Recommendation:** Extract to modular route handlers with lazy ServiceRegistry.

**Approach:**
1. Create `internal/gateway/registry.go` with `ServiceRegistry` (lazy gRPC connections)
2. Define `RouteRegistrar` interface
3. Create per-service route files: `route_auth.go`, `route_crm.go`, `route_chat.go`, `route_notification.go`
4. Move handler methods from monolithic `http.go` to per-service files
5. Update `cmd/gateway/main.go` to use registry + route registrars
6. Gateway `depends_on` in docker-compose changes from `condition: service_healthy` to just `condition: service_started` (gateway no longer needs all services up to start)

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `grpc.Dial()` (eager, blocking) | `grpc.NewClient()` (lazy, non-blocking) | grpc-go v1.63+ (2024) | Gateway can start without all services up |
| Separate message broker (NATS/Rabbit) for events | PostgreSQL LISTEN/NOTIFY for small-scale | Always available, but ecosystem recognition grew 2024-2025 | No new infrastructure needed |
| Manual LISTEN reconnection | pgxlisten library | 2022-2025 | Still pre-v1; raw pgx recommended for production |
| WebSocket per-channel model only | WebSocket per-user + per-channel | N/A (design pattern) | Notifications are user-targeted, not channel-targeted |

**Deprecated/outdated:**
- `grpc.Dial()` and `grpc.DialContext()`: Deprecated in grpc-go. Use `grpc.NewClient()` instead. The project already uses `grpc.NewClient()`.
- `grpc.WithBlock()`: Deprecated. Not needed because `grpc.NewClient()` is lazy by default.
- `lib/pq` for LISTEN/NOTIFY: The project uses `jackc/pgx` which has native LISTEN/NOTIFY support. Do not use `lib/pq`.

## Open Questions

1. **Notification service as separate microservice vs. embedded in gateway?**
   - What we know: The notification service needs a dedicated PostgreSQL connection for LISTEN, plus its own service logic (preferences, grouping, storage). Other services (auth, CRM, chat) are separate microservices.
   - What's unclear: Whether the LISTEN connection should live in the notification service (clean separation) or in the gateway (closer to WebSocket hub).
   - Recommendation: Separate microservice for consistency with existing architecture. The gateway calls the notification service via gRPC for preference CRUD and notification listing. The notification service pushes to the WebSocket hub via a callback or by being co-located with the hub in the gateway process. Practical approach: notification service logic runs as a library within the gateway process (avoiding an extra gRPC hop for real-time delivery) while exposing a gRPC interface for other services to query notifications and preferences.

2. **Event emission: service-side trigger function vs. database trigger?**
   - What we know: Services can call `pg_notify()` in their transactions. Alternatively, PostgreSQL triggers on key tables could auto-emit events.
   - What's unclear: Which approach is more maintainable as modules grow.
   - Recommendation: Service-side `pg_notify()` calls. Database triggers are harder to debug, test, and version. Service code is explicit about what events it emits.

3. **Notification for offline users: store-and-forward or drop?**
   - What we know: Notifications are stored in the database regardless of online status. The question is whether desktop push should be queued for delivery when the user comes back online.
   - What's unclear: How long to queue, and whether stale desktop pushes are useful.
   - Recommendation: Store in database always (in-app notifications). Desktop push is fire-and-forget -- if the user's Electron app is not connected, they see the notification in the bell panel when they return. No queuing of desktop pushes.

## Sources

### Primary (HIGH confidence)
- [PostgreSQL NOTIFY documentation](https://www.postgresql.org/docs/current/sql-notify.html) - Payload limits (8000 bytes), transactional semantics, queue behavior
- [grpc-go v1.78.0 documentation](https://pkg.go.dev/google.golang.org/grpc) - `grpc.NewClient` lazy behavior, WaitForReady, connection management
- [Electron Notification API](https://www.electronjs.org/docs/latest/api/notification) - Platform support matrix, action buttons (macOS only), sounds
- [Electron Tray API](https://www.electronjs.org/docs/latest/api/tray) - System tray icon, badge/indicator via image swap
- Existing codebase analysis: `backend/cmd/gateway/main.go`, `backend/internal/server/http.go`, `backend/internal/server/websocket.go`

### Secondary (MEDIUM confidence)
- [The Notifier Pattern (Brandur)](https://brandur.org/notifier) - Single-connection LISTEN pattern, buffered channel distribution, reconnection strategy
- [pgxlisten documentation](https://pkg.go.dev/github.com/jackc/pgxlisten) - Handler interface, BacklogHandler, reconnection (pre-v1, commit-based versioning)
- [jackc/pgx GitHub](https://github.com/jackc/pgx) - LISTEN/NOTIFY native support in pgx v5

### Tertiary (LOW confidence)
- [Scalable Notification System Design (dev.to)](https://dev.to/ndohjapan/scalable-notification-system-design-for-50-million-users-database-design-4cl) - Schema patterns for notifications, receivers, templates
- [Building Real-Time Notification System in Go (finly.ch)](https://www.finly.ch/engineering-blog/436253-building-a-real-time-notification-system-in-go-with-postgresql) - Go + PostgreSQL notification patterns

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All libraries already in use, no new dependencies needed
- Architecture: HIGH - Patterns derived from existing codebase analysis + official documentation
- Gateway refactoring: HIGH - grpc.NewClient lazy behavior verified via official docs; chi Mount pattern well-documented
- Event bus: HIGH - PostgreSQL LISTEN/NOTIFY thoroughly documented; limitations well-understood
- Notification schema: MEDIUM - Schema designed from first principles + community patterns; not copied from a proven reference implementation
- Electron push: MEDIUM - API documented but action button support varies by platform; Windows support for quick actions needs validation during Phase 5
- Pitfalls: HIGH - Based on PostgreSQL documentation (payload limits, durability) and codebase analysis (gateway coupling)

**Research date:** 2026-02-07
**Valid until:** 2026-03-07 (30 days - stable domain, no fast-moving dependencies)

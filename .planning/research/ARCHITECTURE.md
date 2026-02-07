# Architecture Patterns

**Domain:** All-in-one workplace platform (CRM + Chat + Project Management + Calendar + Email + Video + HR + Finance + Automation + Plugins)
**Researched:** 2026-02-07
**Confidence:** HIGH (based on thorough analysis of existing 4-service codebase and established Go microservice patterns)

---

## Executive Summary

KMU Hub currently runs 4 Go microservices (Gateway, Auth, CRM, Chat) with a clean gateway-to-gRPC pattern. Adding 8 more modules as 8 separate gRPC services would result in 12+ services, which is unmaintainable for a solo developer. The recommendation is to consolidate the 8 new modules into 3-4 new gRPC services grouped by domain affinity and coupling. An internal event bus (PostgreSQL LISTEN/NOTIFY + an in-process event dispatcher) enables cross-service coordination without introducing message broker infrastructure. The WASM plugin system hooks into well-defined extension points within existing services rather than running as a separate service.

---

## Recommended Architecture

### Service Topology: 7 Services Total (Current 4 + 3 New)

```
                    +------------------+
                    |     Desktop      |
                    |  Electron+React  |
                    +--------+---------+
                             |
                    HTTP/REST + WebSocket
                             |
                    +--------+---------+
                    |     Gateway      |  Port 8080 (HTTP), 9090 (metrics)
                    |   chi/v5 router  |
                    |   + WS Hub       |
                    +--+--+--+--+--+--+
                       |  |  |  |  |
            +----------+  |  |  |  +----------+
            |             |  |  |             |
      gRPC :50051   :50052 :50053 :50054   :50055
      +-----+  +-----+ +-----+ +-------+ +--------+
      | Auth | | CRM | | Chat| | Work  | | Biz    |
      +------+ +-----+ +-----+ +-------+ +--------+
                                | Proj  | | HR     |
                                | Cal   | | Finance|
                                | Email | +--------+
                                +-------+
                                    |
                              +-----+------+
                              | Automation |  Port :50056
                              |  Engine    |  (gRPC + internal event consumer)
                              +------------+
```

**Why 7, not 12:**
- 1 developer cannot maintain 12 separate Dockerfiles, health checks, config entries, proto files, gRPC servers, and gateway client connections
- Current pattern already groups related domains: CRM service contains contacts + companies + deals + activities + tags + custom fields + pipeline stages + reports + filters (10 sub-domains in 1 service)
- Domain affinity: Project Management, Calendar, and Email share user/scheduling concepts and frequently cross-reference each other
- HR and Finance are administrative modules with minimal real-time requirements

### Service Grouping Recommendation

| Service | Port | Contains | Rationale |
|---------|------|----------|-----------|
| **Auth** (existing) | :50051 | Users, roles, permissions, invitations, JWT | Identity is foundational, stays separate |
| **CRM** (existing) | :50052 | Contacts, companies, deals, activities, tags, custom fields, pipeline, reports, filters | Already cohesive, no changes needed |
| **Chat** (existing) | :50053 | Channels, messages, DMs, threads, mentions, files, search, **notifications** | Add notifications here -- they are primarily chat-driven |
| **Work** (new) | :50054 | Project management, calendar, email integration | These three share scheduling, assignments, and timeline concepts. A task has a due date (calendar), an email can create a task, a calendar event can link to a project. Tight coupling justifies grouping. |
| **Biz** (new) | :50055 | HR module, finance module | Administrative back-office functions. Low coupling to real-time features. Both relate to employee/company data. Separate from CRM because CRM is customer-facing, Biz is internal operations. |
| **Automation** (new) | :50056 | Workflow engine, triggers, actions, WASM plugin runtime | Must observe events from ALL services. Separate because it is cross-cutting and has unique runtime requirements (WASM sandbox). |
| **Gateway** (existing) | :8080 | HTTP routing, WebSocket hub, rate limiting, auth middleware | Unchanged role. Adds gRPC clients for Work, Biz, Automation. |

### Why NOT Keep Them All Separate

The temptation is "clean boundaries = separate services." But consider the operational cost per service for a solo developer:

| Per-Service Cost | Items |
|-----------------|-------|
| Dockerfile | 1 file, ~20 lines |
| docker-compose entry | 1 entry, ~30 lines, env vars, health check, port |
| Config struct fields | 2 fields (port + address) |
| Gateway gRPC connection | ~15 lines of connection setup |
| Proto file | 1 file, grows with RPCs |
| gRPC server file | 1 file, 500-2000 lines |
| cmd/main.go | 1 file, ~150 lines |
| Health check endpoint | 1 check + port |
| CI adjustments | Service container if needed |
| Gateway handler methods | N methods, each ~20 lines |

With 8 separate services: 8 Dockerfiles + 8 compose entries + 16 config fields + 8 gateway connections + 8 proto files + 8 gRPC servers + 8 main.go files = massive surface area for 1 developer.

With 3 grouped services: 3 Dockerfiles + 3 compose entries + 6 config fields + 3 gateway connections + 3 proto files + 3 gRPC servers + 3 main.go files = manageable.

### Why NOT a Monolith

Going the other direction -- merging everything into 1-2 services -- is also wrong:
- The existing 4-service architecture works well and is proven
- Graceful degradation requires service isolation (CRM works when Chat is down)
- Different scaling profiles: Chat needs WebSocket concurrency, CRM needs query throughput, Biz needs almost nothing
- Proto files would become unmanageably large (CRM proto is already 828 lines)

---

## Component Boundaries

### Work Service (Project Management + Calendar + Email)

| Sub-Domain | Responsibility | Internal Dependencies |
|-----------|---------------|----------------------|
| **Project** | Projects, tasks, boards, timelines, assignments, dependencies | Calendar (task due dates appear on calendar) |
| **Calendar** | Events, shared calendars, availability, scheduling | Project (project milestones on calendar), Email (meeting invitations) |
| **Email** | Send/receive via IMAP/SMTP, email-to-task conversion, per-customer mail config | Project (email creates task), CRM contacts (email linked to contact) |

**Internal communication:** These sub-domains share a database schema within the Work service. They can call each other's service layer directly (like CRM's contact service calling tag service). No gRPC between sub-domains.

**External communication:** Work service exposes a single gRPC service (`WorkService`) with RPCs grouped by sub-domain. Gateway routes to this one gRPC endpoint.

```
backend/
  cmd/work/main.go
  internal/work/
    project/         # service.go, repository.go, postgres_repository.go, errors.go
      task/          # sub-package for task-specific logic
      board/         # Kanban/list views
    calendar/        # service.go, repository.go, ...
      event/
      availability/
    email/           # service.go, repository.go, ...
      imap/          # IMAP client
      smtp/          # SMTP client
  proto/work/v1/work.proto
```

**Cross-service reads (Work -> CRM):**
Work service needs CRM data (e.g., link a task to a contact, show contact name on a calendar event). Two approaches:

- **Option A: gRPC call to CRM.** Work service is a gRPC client of CRM service. This creates a service dependency but keeps data ownership clear.
- **Option B: Shared database read.** Work service reads from CRM tables directly (read-only). Violates pure microservice boundaries but is pragmatic for a single database.

**Recommendation: Option B (shared database read) for reads, gRPC for writes.** All services already share one PostgreSQL database. Having Work service read `contacts.first_name` via a JOIN is simpler and faster than a gRPC round-trip. The rule: services OWN their tables (only they write) but may READ other services' tables. This is safe because PostgreSQL is the single source of truth and there is no database-per-service split.

### Biz Service (HR + Finance)

| Sub-Domain | Responsibility | Internal Dependencies |
|-----------|---------------|----------------------|
| **HR** | Leave requests, time tracking, employee profiles, onboarding checklists | Finance (time tracking feeds payroll) |
| **Finance** | Quotes, invoices, payment tracking, accounting system export (DATEV) | CRM (invoices linked to deals/companies), HR (payroll data) |

```
backend/
  cmd/biz/main.go
  internal/biz/
    hr/
      leave/
      timetracking/
      onboarding/
    finance/
      quote/
      invoice/
      accounting/     # DATEV export, accounting system integration
  proto/biz/v1/biz.proto
```

**Cross-service reads (Biz -> CRM, Biz -> Auth):**
Same pattern as Work: direct SQL reads for CRM/Auth data, gRPC calls only for mutations that belong to other services.

### Automation Service (Workflow Engine + WASM Plugin Runtime)

This is the most architecturally complex new service. It must:
1. React to events from ALL other services
2. Execute user-defined workflow rules
3. Run WASM plugins in sandboxed environments
4. Trigger actions in ANY service

```
backend/
  cmd/automation/main.go
  internal/automation/
    engine/           # Workflow execution engine
      trigger.go      # Event matching, condition evaluation
      action.go       # Action execution (calls other services)
      workflow.go     # Workflow definition, versioning
    plugin/           # WASM plugin system
      runtime.go      # wazero WASM runtime manager
      host.go         # Host functions exposed to WASM
      registry.go     # Plugin registration, lifecycle
      sandbox.go      # Resource limits, timeouts
    event/            # Event consumption
      consumer.go     # PostgreSQL LISTEN/NOTIFY consumer
      dispatcher.go   # Routes events to workflows and plugins
  proto/automation/v1/automation.proto
```

---

## Inter-Service Communication Patterns

### Pattern 1: Synchronous gRPC (Request/Response)

**What:** Gateway calls backend services via gRPC. Already established pattern.

**When:** Client requests that need immediate responses. CRUD operations. Queries.

**Data flow:**
```
Desktop -> HTTP -> Gateway -> gRPC -> Service -> PostgreSQL
                                                     |
Desktop <- HTTP <- Gateway <- gRPC <- Service <------+
```

**For new services:** Gateway adds 3 new gRPC client connections (Work, Biz, Automation). Config adds 6 new fields (3 ports + 3 addresses).

### Pattern 2: Cross-Service Reads via Shared Database

**What:** Services read (but never write) tables owned by other services via direct SQL JOINs.

**When:** Denormalizing data for API responses (e.g., Work service showing contact name on a task).

**Rules:**
- Each service OWNS its tables (only it writes to them)
- Services MAY read other services' tables via SQL JOINs
- No cross-service write access, ever
- If a write is needed in another service's domain, use gRPC

**Why this is safe for KMU Hub:**
- Single PostgreSQL database (no database-per-service)
- No plans to split databases (SMB scale, not enterprise scale)
- PostgreSQL handles concurrent reads efficiently
- The alternative (gRPC calls for every cross-reference) adds latency, complexity, and failure modes

**Ownership table:**

| Table Prefix | Owner Service | May Read |
|-------------|---------------|----------|
| `users`, `roles`, `permissions`, `refresh_tokens`, `invitations` | Auth | All |
| `contacts`, `companies`, `deals`, `activities`, `tags`, `custom_field_*`, `pipeline_stages`, `saved_filters` | CRM | Work, Biz, Automation |
| `channels`, `messages`, `channel_memberships`, `chat_files`, `mentions` | Chat | Automation |
| `projects`, `tasks`, `calendar_events`, `emails` | Work | CRM (activity linking), Biz, Automation |
| `leave_requests`, `time_entries`, `quotes`, `invoices` | Biz | Automation |
| `workflows`, `workflow_runs`, `plugins` | Automation | None (internal only) |

### Pattern 3: Event Bus via PostgreSQL LISTEN/NOTIFY

**What:** Services emit domain events after successful writes. The Automation service (and optionally others) listens for these events.

**Why PostgreSQL LISTEN/NOTIFY instead of a message broker (NATS, RabbitMQ, Kafka):**
- Zero additional infrastructure (PostgreSQL already running)
- Sufficient for SMB scale (100-200 concurrent users, not millions)
- No new operational dependency for self-hosted customers
- Simple Go implementation with `pgx` (which is already a dependency)
- If scale demands it later, NATS can replace LISTEN/NOTIFY with minimal code changes

**How it works:**

```sql
-- Service emits event after successful write (in same transaction)
-- Using a PostgreSQL function + trigger OR explicit NOTIFY in repository

-- Option A: Explicit NOTIFY in repository (recommended -- simpler, more control)
NOTIFY kmuhub_events, '{"type":"deal.stage_changed","deal_id":"...","from_stage":"...","to_stage":"...","user_id":"...","timestamp":"..."}';

-- Option B: Trigger-based (automatic but harder to debug)
-- Not recommended for this use case
```

```go
// Event consumer in Automation service
func (c *EventConsumer) Start(ctx context.Context, pool *pgxpool.Pool) error {
    conn, err := pool.Acquire(ctx)
    if err != nil {
        return err
    }
    defer conn.Release()

    _, err = conn.Exec(ctx, "LISTEN kmuhub_events")
    if err != nil {
        return err
    }

    for {
        notification, err := conn.Conn().WaitForNotification(ctx)
        if err != nil {
            return err
        }
        c.dispatch(ctx, notification.Payload)
    }
}
```

**Event schema:**

```json
{
    "type": "deal.stage_changed",
    "source": "crm",
    "entity_id": "uuid",
    "entity_type": "deal",
    "user_id": "uuid",
    "timestamp": "2026-02-07T10:00:00Z",
    "data": {
        "from_stage": "qualified",
        "to_stage": "proposal"
    }
}
```

**Event types by service:**

| Service | Events Emitted |
|---------|---------------|
| Auth | `user.created`, `user.updated`, `user.deactivated`, `role.assigned`, `role.removed` |
| CRM | `contact.created`, `contact.updated`, `contact.deleted`, `deal.created`, `deal.stage_changed`, `deal.won`, `deal.lost`, `activity.created`, `activity.completed` |
| Chat | `message.sent`, `channel.created`, `file.uploaded`, `mention.created` |
| Work | `task.created`, `task.completed`, `task.overdue`, `event.created`, `email.received`, `email.sent` |
| Biz | `leave.requested`, `leave.approved`, `invoice.created`, `invoice.paid`, `time.logged` |

**Event persistence (optional, recommended):**
Write events to an `events` table before NOTIFY. This provides:
- Audit trail
- Replay capability if Automation service was down
- Debugging (query events table)

```sql
CREATE TABLE events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type VARCHAR(100) NOT NULL,
    source VARCHAR(50) NOT NULL,
    entity_id UUID,
    entity_type VARCHAR(50),
    user_id UUID,
    data JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_events_type ON events(type);
CREATE INDEX idx_events_created_at ON events(created_at);
-- Auto-delete events older than 90 days via pg_cron or application cleanup
```

### Pattern 4: WebSocket Event Broadcasting (Gateway Hub)

**What:** The existing WebSocket hub broadcasts real-time events to connected desktop clients.

**Current state:** Hub only handles Chat events (messages, typing, read receipts).

**Extension for new services:** The hub becomes a general-purpose event broadcaster. Each service can publish events that the gateway broadcasts to relevant users.

```go
// Extended WebSocket message types
const (
    // Existing Chat types...
    WSMessageNew      = "message.new"
    WSTypingIndicator = "typing"

    // New: Work events
    WSTaskAssigned    = "task.assigned"
    WSTaskCompleted   = "task.completed"
    WSCalendarEvent   = "calendar.event"
    WSEmailReceived   = "email.received"

    // New: Biz events
    WSLeaveApproved   = "leave.approved"
    WSInvoicePaid     = "invoice.paid"

    // New: Automation events
    WSWorkflowTriggered = "workflow.triggered"
    WSWorkflowCompleted = "workflow.completed"

    // New: Generic notification
    WSNotification    = "notification"
)
```

**How gateway learns about events from non-chat services:**
Two options, both viable:

**Option A: Services call gateway via reverse gRPC (not recommended).** Adds bidirectional gRPC complexity.

**Option B: Gateway also listens to PostgreSQL LISTEN/NOTIFY (recommended).** Gateway has a database connection (already established for file uploads). Add a second NOTIFY channel (`kmuhub_realtime`) specifically for events that should reach the desktop client. Services emit to this channel when they want real-time push.

```go
// In gateway main.go
realtimeConsumer := server.NewRealtimeConsumer(pool, wsHub)
go realtimeConsumer.Start(ctx)

// In RealtimeConsumer
func (c *RealtimeConsumer) dispatch(ctx context.Context, payload string) {
    var event RealtimeEvent
    json.Unmarshal([]byte(payload), &event)

    switch {
    case event.TargetUserIDs != nil:
        // Send to specific users
        for _, userID := range event.TargetUserIDs {
            c.hub.sendToUser(ctx, userID, WSMessage{Type: event.Type, Message: event.Data})
        }
    case event.TargetChannelID != "":
        // Broadcast to channel members (existing pattern)
        c.hub.broadcastToChannel(ctx, event.TargetChannelID, WSMessage{Type: event.Type, Message: event.Data})
    default:
        // Broadcast to all connected users (rare, e.g., system announcements)
        c.hub.broadcastToAll(ctx, WSMessage{Type: event.Type, Message: event.Data})
    }
}
```

---

## Automation Engine Architecture

The automation engine is the most complex new component. It must be designed carefully.

### Workflow Model

```
Workflow
  ├── name: "When deal won, create invoice"
  ├── trigger: { type: "event", event: "deal.won" }
  ├── conditions: [
  │     { field: "data.value", operator: ">=", value: 1000 }
  │   ]
  ├── actions: [
  │     { type: "create_entity", service: "biz", entity: "invoice", mapping: {...} },
  │     { type: "send_notification", target: "deal.owner_id", message: "..." },
  │     { type: "run_plugin", plugin_id: "...", input: {...} }
  │   ]
  └── is_active: true
```

### How the Automation Engine Crosses Service Boundaries

The engine needs to both READ from and WRITE to other services. The pattern:

**Reading:** Direct SQL queries against the shared database. The engine can read any table to evaluate conditions.

**Writing:** gRPC calls to the owning service. The engine is a gRPC client of ALL other services.

```go
// Automation service main.go
func main() {
    // ... standard setup ...

    // gRPC clients to all services (for executing actions)
    authClient := authv1.NewAuthServiceClient(authConn)
    crmClient := crmv1.NewCRMServiceClient(crmConn)
    chatClient := chatv1.NewChatServiceClient(chatConn)
    workClient := workv1.NewWorkServiceClient(workConn)
    bizClient := bizv1.NewBizServiceClient(bizConn)

    // Action executor with all clients
    executor := automation.NewActionExecutor(authClient, crmClient, chatClient, workClient, bizClient)

    // Event consumer (PostgreSQL LISTEN/NOTIFY)
    consumer := event.NewConsumer(pool)

    // Workflow engine
    engine := automation.NewEngine(workflowService, executor, consumer)
    go engine.Start(ctx)
}
```

**Security constraint:** Automation executes actions AS the user who created the workflow (or a system account). The engine passes `user_id` in gRPC requests, and target services enforce RBAC. The automation engine does NOT bypass authorization.

### Workflow Execution Model

```
Event arrives via LISTEN/NOTIFY
  |
  v
Match against active workflow triggers
  |
  v
Evaluate conditions (SQL reads + in-memory logic)
  |
  v
Execute actions sequentially
  |--- gRPC call to target service (create invoice, send message, etc.)
  |--- WASM plugin call (sandboxed, with host function access)
  |--- Notification (WebSocket via NOTIFY to kmuhub_realtime channel)
  |
  v
Log execution result to workflow_runs table
```

---

## Plugin/WASM System Architecture

### Two-Tier Extension System (Confirmed from ADR-004)

**Tier 1: Config-based (80% of customizations)**
- Custom fields (already built in CRM)
- Workflow rules (automation engine)
- UI layout configuration (desktop app)
- Report templates
- Validation rules (JSON schema)
- No code required, managed via admin UI

**Tier 2: WASM plugins (20% of customizations)**
- Custom business logic (e.g., industry-specific calculations)
- External integrations (API connectors)
- Custom data transformations
- Complex validation beyond JSON schema

### WASM Runtime (wazero)

**Why wazero:** Pure Go WASM runtime, no CGO dependency, sandbox isolation, deterministic execution. Already decided in ADR-004.

**Where WASM runs:** Inside the Automation service. Plugins do NOT run inside CRM, Chat, or other services. The automation engine is the plugin host.

```
Plugin Lifecycle:
  1. Admin uploads .wasm file via Gateway -> Automation service
  2. Automation service validates, stores in MinIO, registers in DB
  3. When workflow action type is "run_plugin", engine loads WASM module
  4. WASM module executes with defined inputs, resource limits, timeout
  5. Output returned to workflow engine for next action
```

### Host Functions (What Plugins Can Do)

WASM plugins are sandboxed. They cannot access the filesystem, network, or database directly. They interact with the system through host functions:

```go
// Host functions exposed to WASM via wazero
type PluginHost struct {
    crmClient  crmv1.CRMServiceClient
    chatClient chatv1.ChatServiceClient
    // ... other clients
}

// Read operations (plugins can query data)
func (h *PluginHost) GetContact(contactID string) (Contact, error)
func (h *PluginHost) GetDeal(dealID string) (Deal, error)
func (h *PluginHost) QueryContacts(filter string) ([]Contact, error)

// Write operations (plugins can create/update entities)
func (h *PluginHost) CreateActivity(activity Activity) (string, error)
func (h *PluginHost) SendChatMessage(channelID, message string) error
func (h *PluginHost) CreateTask(task Task) (string, error)

// Utility functions
func (h *PluginHost) Log(level, message string)
func (h *PluginHost) HTTPGet(url string) ([]byte, error)  // Allowlisted domains only
func (h *PluginHost) Now() time.Time
```

**Security boundaries:**
- All host function calls go through the automation service, which enforces RBAC
- HTTP access is restricted to an allowlist configured per plugin
- Resource limits: max memory (16MB default), max execution time (5s default), max host function calls (100 per invocation)
- Plugins cannot access raw SQL, environment variables, or filesystem

### How Plugins Hook Into Existing Services

Plugins do NOT directly modify existing services. Instead:

1. **Event-driven:** Plugins react to events via the automation engine workflow system
2. **Action-driven:** Plugin outputs trigger standard gRPC actions (create contact, send message, etc.)
3. **Transform-driven:** Plugins can transform data in-flight (e.g., custom field validation, data enrichment before save)

For transform-driven plugins, the pattern is:

```
Client -> Gateway -> Service -> [Pre-save hook: run WASM transform] -> Repository -> DB
```

The pre-save hook is implemented in the service layer:

```go
func (s *ContactService) Create(ctx context.Context, req CreateContactRequest) (*Contact, error) {
    // Standard validation
    if err := s.validate(req); err != nil {
        return nil, err
    }

    // Plugin hook: pre-create transform
    if s.pluginHooks != nil {
        transformed, err := s.pluginHooks.OnContactPreCreate(ctx, req)
        if err != nil {
            slog.Warn("plugin pre-create hook failed", "error", err)
            // Graceful degradation: continue with original request
        } else {
            req = transformed
        }
    }

    return s.repo.Create(ctx, req)
}
```

**Confidence note:** The plugin hook architecture is MEDIUM confidence. The exact wazero API for host functions and the plugin SDK design will need deeper research during the automation/plugin phase. The high-level architecture (plugins run in automation service, communicate via host functions, triggered via workflow engine) is HIGH confidence.

---

## Desktop App Architecture for Multiple Modules

### Module-Based UI Architecture

```
desktop/src/
  main/                    # Electron main process
    index.ts               # Window management, IPC, system tray
    ipc.ts                 # IPC handlers
    updater.ts             # Auto-update logic
  renderer/                # React renderer process
    App.tsx                # Root with router + providers
    layouts/
      WorkspaceLayout.tsx  # Shell: sidebar + content area
      ModuleLayout.tsx     # Per-module chrome (tabs, breadcrumbs)
    modules/               # Feature modules (lazy-loaded)
      crm/
        index.tsx          # Module entry + routes
        pages/             # Full pages
        components/        # Module-specific components
      chat/
        index.tsx
        pages/
        components/
      projects/
        index.tsx
        pages/
        components/
      calendar/
        index.tsx
      email/
        index.tsx
      hr/
        index.tsx
      finance/
        index.tsx
      settings/
        index.tsx
        automation/        # Workflow builder UI
        plugins/           # Plugin management UI
    shared/                # Cross-module shared code
      components/          # Design system (buttons, forms, tables, modals)
      hooks/               # Shared React hooks (useAuth, useWebSocket, useAPI)
      api/                 # API client layer
        client.ts          # HTTP client with auth interceptor
        websocket.ts       # WebSocket connection manager
        types.ts           # Shared TypeScript types
      stores/              # State management (Zustand or React Context)
        auth.ts
        notifications.ts
        workspace.ts
```

### Module Registration Pattern

Each module registers itself with the workspace shell:

```typescript
// modules/crm/index.tsx
export const crmModule: WorkspaceModule = {
    id: 'crm',
    name: 'CRM',
    icon: 'users',
    routes: [
        { path: '/crm/contacts', component: lazy(() => import('./pages/Contacts')) },
        { path: '/crm/deals', component: lazy(() => import('./pages/Deals')) },
        // ...
    ],
    sidebarItems: [
        { label: 'Contacts', path: '/crm/contacts', icon: 'user' },
        { label: 'Deals', path: '/crm/deals', icon: 'dollar' },
    ],
    permissions: ['contacts:read', 'deals:read'],
};
```

This pattern allows:
- Lazy loading (only load module JS when user navigates to it)
- Role-based module visibility (hide HR from non-HR users)
- Per-customer module enablement (config-based: "this customer uses CRM + Chat + HR, not Finance")

### State Management

**Recommendation: Zustand** for client-side state. Lightweight, TypeScript-friendly, no boilerplate. Each module gets its own store slice.

```
Global stores:        auth, notifications, workspace, websocket
Module stores:        crm, chat, projects, calendar, email, hr, finance
Cross-module:         entity-linking (contact picker, deal selector used across modules)
```

**API layer:** A thin HTTP client that wraps `fetch` with JWT auth header injection, refresh token rotation, and error handling. Each module defines its own API functions calling this client.

---

## Build Order (Dependencies Between New Services)

### Phase Dependencies

```
Phase 3 Complete (Chat)
  |
  v
Phase 4: Desktop App Shell + CRM/Chat UI
  |  (No new backend services -- pure frontend)
  |
  v
Phase 5: Work Service (Project Management first, then Calendar, then Email)
  |  Depends on: Auth (user assignment), CRM (entity linking)
  |  Work service reads CRM tables for contact/deal references
  |
  v
Phase 6: Video/Voice (LiveKit integration)
  |  Depends on: Chat (video calls initiated from chat), Auth (room access tokens)
  |  LiveKit is external infrastructure, not a custom gRPC service
  |  Integration code lives in Chat service (or a thin video sub-package)
  |
  v
Phase 7: Biz Service (HR first, then Finance)
  |  Depends on: Auth (employee = user), CRM (finance links to deals/companies)
  |  HR has minimal dependencies, good starting point
  |  Finance needs CRM data for invoices linked to deals
  |
  v
Phase 8: Automation Engine + Plugin System
  |  Depends on: ALL other services (it observes and acts on everything)
  |  Must be built LAST because it needs stable APIs to hook into
  |  Event emission (NOTIFY) should be added to earlier services incrementally
  |
  v
Beta
```

### Incremental Event Emission

Do NOT wait until Phase 8 to add events. Instead, add `NOTIFY` calls to services as they are built:

- **Phase 3 (current):** Add events to Chat service (message.sent, file.uploaded)
- **Phase 5:** Add events to CRM service (deal.stage_changed, contact.created) and Work service
- **Phase 7:** Add events to Biz service
- **Phase 8:** Automation engine consumes all accumulated events

This way, by Phase 8, all services already emit events and the automation engine just needs to consume and process them.

---

## Patterns to Follow

### Pattern: Consistent Sub-Domain Package Structure

Every sub-domain follows the same file structure (already established in CRM/Chat):

```
internal/{service}/{subdomain}/
  service.go              # Business logic (thick)
  service_test.go         # Unit tests with mock repo
  repository.go           # Repository interface
  postgres_repository.go  # PostgreSQL implementation
  errors.go               # Domain-specific sentinel errors
```

**Why:** Consistency means any developer (or AI) can navigate unfamiliar modules by pattern recognition. "Where is the deal creation logic?" -> `internal/crm/deal/service.go`. "Where is the task creation logic?" -> `internal/work/project/task/service.go`.

### Pattern: Event Emission in Repository Layer

```go
// In postgres_repository.go
func (r *PostgresRepository) Create(ctx context.Context, deal *models.Deal) error {
    tx, err := r.pool.Begin(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback(ctx)

    // Insert deal
    _, err = tx.Exec(ctx, "INSERT INTO deals ...")
    if err != nil {
        return err
    }

    // Emit event (same transaction -- atomic with the write)
    eventPayload, _ := json.Marshal(map[string]interface{}{
        "type":      "deal.created",
        "source":    "crm",
        "entity_id": deal.ID,
        "user_id":   deal.CreatedBy,
        "data":      deal,
    })
    _, err = tx.Exec(ctx, "SELECT pg_notify('kmuhub_events', $1)", string(eventPayload))
    if err != nil {
        return err  // Event emission failure rolls back the entire transaction
    }

    return tx.Commit(ctx)
}
```

**Why in repository, not service:** Events should be atomic with the database write. If the INSERT succeeds but the NOTIFY fails (unlikely with same-transaction), we want to roll back. Putting NOTIFY in the service layer risks the event being emitted without the data being committed (or vice versa).

### Pattern: Service-to-Service gRPC Call with Circuit Breaker

```go
// For automation engine calling other services
func (e *ActionExecutor) CreateInvoice(ctx context.Context, req *bizv1.CreateInvoiceRequest) error {
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    _, err := e.bizClient.CreateInvoice(ctx, req)
    if err != nil {
        st, ok := status.FromError(err)
        if ok && st.Code() == codes.Unavailable {
            slog.Warn("biz service unavailable, queuing action for retry",
                "action", "create_invoice",
                "error", err,
            )
            return e.queueRetry(ctx, "create_invoice", req)
        }
        return err
    }
    return nil
}
```

---

## Anti-Patterns to Avoid

### Anti-Pattern: Service-per-Module

**What:** Creating a separate gRPC service for every logical module (Project, Calendar, Email, HR, Finance, Automation, Plugin, Notification = 8 new services).

**Why bad:** Operational overhead scales linearly. 12+ services for 1 developer means more time on infrastructure than features. Docker Compose becomes a nightmare. Config file grows to 50+ env vars. Gateway main.go becomes 300+ lines of just gRPC connection setup.

**Instead:** Group by domain affinity into 3 services (Work, Biz, Automation).

### Anti-Pattern: Database-per-Service

**What:** Giving each service its own PostgreSQL database for "true" microservice isolation.

**Why bad:** Eliminates the ability to JOIN across domains (need for every cross-reference becomes a gRPC call). Distributed transactions become necessary for operations spanning services. Migration management multiplies. For a solo dev targeting SMBs, the complexity is unjustifiable.

**Instead:** Single database, table-level ownership. Services write only to their tables, may read from others.

### Anti-Pattern: Message Broker for Internal Events

**What:** Adding NATS, RabbitMQ, or Kafka for inter-service events.

**Why bad at this stage:** Additional infrastructure component that self-hosted customers must deploy and maintain. PostgreSQL LISTEN/NOTIFY handles the event volume for 5-200 user companies. Message brokers add operational complexity (monitoring, persistence, dead letter queues, partitioning) without proportional benefit at SMB scale.

**Instead:** PostgreSQL LISTEN/NOTIFY with an events table for persistence. Upgrade to NATS if/when event volume exceeds PostgreSQL's notification capacity (~10,000 events/second is the practical limit for LISTEN/NOTIFY).

### Anti-Pattern: Direct Service-to-Service Database Writes

**What:** The Work service writes directly to the `activities` table (owned by CRM) to auto-create an activity when a task is completed.

**Why bad:** Bypasses CRM business logic (validation, event emission, custom field handling). Creates hidden coupling. CRM cannot trust its own data if other services write to its tables.

**Instead:** Work service calls `crmClient.CreateActivity()` via gRPC. CRM service handles all write-path logic.

### Anti-Pattern: WASM Plugins with Direct DB Access

**What:** Giving WASM plugins a database connection to read/write directly.

**Why bad:** Bypasses all authorization, validation, and business logic. Plugins could corrupt data, cause injection, or leak sensitive information. Plugin authors must understand the database schema.

**Instead:** Host functions that expose a controlled API. Plugins call `GetContact(id)` which internally does a gRPC call through the automation service.

---

## Scalability Considerations

| Concern | At 100 users (SMB) | At 10K users (multi-tenant SaaS) | At 100K users (future) |
|---------|---------------------|----------------------------------|------------------------|
| **Service instances** | 1 per service (Docker Compose) | 2-3 replicas per service (K8s) | Auto-scale per service |
| **Database** | Single PostgreSQL, 25 connections | Read replicas, pgBouncer (200 connections) | Aurora PostgreSQL or Citus |
| **Events** | PostgreSQL LISTEN/NOTIFY | PostgreSQL LISTEN/NOTIFY still fine | Migrate to NATS JetStream |
| **WebSocket** | Single gateway, in-memory hub | Multiple gateways, Redis pubsub for hub | Multiple gateways, NATS for hub |
| **File storage** | Single MinIO node | MinIO distributed (4+ nodes) | S3-compatible cloud storage |
| **WASM plugins** | In-process wazero | Dedicated automation replicas | Worker pool with queue |
| **Video** | Single LiveKit server | LiveKit cloud or multi-node | LiveKit cloud |

---

## Docker Compose Additions (Self-Hosted Compatibility)

```yaml
# New services added to docker-compose.yml

  work:
    build:
      context: ../../backend
      dockerfile: Dockerfile.work
    depends_on:
      postgres:
        condition: service_healthy
      migrate:
        condition: service_completed_successfully
    environment:
      DATABASE_URL: postgres://kmuhub:kmuhub_dev@postgres:5432/kmuhub?sslmode=disable
      JWT_SECRET: docker-dev-secret-minimum-32-characters
      WORK_GRPC_PORT: ":50054"
      WORK_HEALTH_PORT: ":9094"
      CRM_GRPC_ADDRESS: crm:50052    # For cross-service writes
      CHAT_GRPC_ADDRESS: chat:50053
      # Email IMAP/SMTP config (per-tenant, loaded from DB)
    ports:
      - "50054:50054"
      - "9094:9094"

  biz:
    build:
      context: ../../backend
      dockerfile: Dockerfile.biz
    depends_on:
      postgres:
        condition: service_healthy
      migrate:
        condition: service_completed_successfully
    environment:
      DATABASE_URL: postgres://kmuhub:kmuhub_dev@postgres:5432/kmuhub?sslmode=disable
      JWT_SECRET: docker-dev-secret-minimum-32-characters
      BIZ_GRPC_PORT: ":50055"
      BIZ_HEALTH_PORT: ":9095"
      CRM_GRPC_ADDRESS: crm:50052
    ports:
      - "50055:50055"
      - "9095:9095"

  automation:
    build:
      context: ../../backend
      dockerfile: Dockerfile.automation
    depends_on:
      postgres:
        condition: service_healthy
      migrate:
        condition: service_completed_successfully
      auth:
        condition: service_healthy
      crm:
        condition: service_healthy
      chat:
        condition: service_healthy
      work:
        condition: service_healthy
      biz:
        condition: service_healthy
    environment:
      DATABASE_URL: postgres://kmuhub:kmuhub_dev@postgres:5432/kmuhub?sslmode=disable
      JWT_SECRET: docker-dev-secret-minimum-32-characters
      AUTOMATION_GRPC_PORT: ":50056"
      AUTOMATION_HEALTH_PORT: ":9096"
      AUTH_GRPC_ADDRESS: auth:50051
      CRM_GRPC_ADDRESS: crm:50052
      CHAT_GRPC_ADDRESS: chat:50053
      WORK_GRPC_ADDRESS: work:50054
      BIZ_GRPC_ADDRESS: biz:50055
      WASM_MAX_MEMORY_MB: 16
      WASM_EXEC_TIMEOUT_SEC: 5
    ports:
      - "50056:50056"
      - "9096:9096"

  # LiveKit (video/voice calls)
  livekit:
    image: livekit/livekit-server:latest
    command: --config /etc/livekit.yaml
    volumes:
      - ./livekit.yaml:/etc/livekit.yaml
    ports:
      - "7880:7880"   # HTTP
      - "7881:7881"   # gRPC (internal)
      - "7882:7882/udp"  # WebRTC UDP
    depends_on:
      redis:
        condition: service_healthy
```

**Total Docker Compose services (final state):** postgres, redis, minio, createbucket, migrate, auth, crm, chat, work, biz, automation, gateway, livekit = **13 containers**. This is manageable. Self-hosted customers run `docker-compose up -d` and get the complete platform.

---

## LiveKit Integration Pattern

LiveKit is NOT a custom gRPC service. It is external infrastructure (like PostgreSQL or Redis). The integration pattern:

```
Desktop App ---- WebRTC ----> LiveKit Server
                                   |
Desktop App <--- WebRTC ---- LiveKit Server

Gateway ---- LiveKit Server SDK ----> LiveKit Server (room creation, token generation)
```

**Where LiveKit integration code lives:** In the Chat service (video calls are initiated from chat channels/DMs). The Chat service uses the LiveKit Server SDK (Go) to:
1. Create rooms
2. Generate participant tokens (with permissions)
3. List active rooms
4. Manage recordings

```go
// internal/chat/video/livekit.go
type LiveKitService struct {
    roomClient *lksdk.RoomServiceClient
    apiKey     string
    apiSecret  string
}

func (s *LiveKitService) CreateRoom(ctx context.Context, channelID string) (*Room, error) { ... }
func (s *LiveKitService) GenerateToken(userID, roomName string, canPublish bool) (string, error) { ... }
```

The desktop app uses the LiveKit JavaScript SDK to join rooms, publish/subscribe to tracks, and handle WebRTC.

---

## Config Struct Extension

```go
// Additions to backend/internal/config/config.go
type Config struct {
    // ... existing fields ...

    // Work Service
    WorkGRPCPort    string `env:"WORK_GRPC_PORT,default=:50054"`
    WorkGRPCAddress string `env:"WORK_GRPC_ADDRESS,default=localhost:50054"`
    WorkHealthPort  string `env:"WORK_HEALTH_PORT,default=:9094"`

    // Biz Service
    BizGRPCPort    string `env:"BIZ_GRPC_PORT,default=:50055"`
    BizGRPCAddress string `env:"BIZ_GRPC_ADDRESS,default=localhost:50055"`
    BizHealthPort  string `env:"BIZ_HEALTH_PORT,default=:9095"`

    // Automation Service
    AutomationGRPCPort    string `env:"AUTOMATION_GRPC_PORT,default=:50056"`
    AutomationGRPCAddress string `env:"AUTOMATION_GRPC_ADDRESS,default=localhost:50056"`
    AutomationHealthPort  string `env:"AUTOMATION_HEALTH_PORT,default=:9096"`
    WASMMaxMemoryMB       int    `env:"WASM_MAX_MEMORY_MB,default=16"`
    WASMExecTimeoutSec    int    `env:"WASM_EXEC_TIMEOUT_SEC,default=5"`

    // LiveKit
    LiveKitURL       string `env:"LIVEKIT_URL,default=ws://localhost:7880"`
    LiveKitAPIKey    string `env:"LIVEKIT_API_KEY"`
    LiveKitAPISecret string `env:"LIVEKIT_API_SECRET"`
}
```

---

## Key Architecture Decisions Summary

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Service count | 7 total (4 existing + 3 new) | Manageable for 1 dev, clear domain boundaries |
| Service grouping | Work (PM+Cal+Email), Biz (HR+Finance), Automation (Engine+WASM) | Domain affinity, coupling, operational cost |
| Cross-service reads | Shared database (read), gRPC (write) | Pragmatic for single-DB architecture, avoids gRPC overhead for reads |
| Event bus | PostgreSQL LISTEN/NOTIFY + events table | Zero new infrastructure, sufficient for SMB scale |
| Real-time push | Extend existing WebSocket hub via NOTIFY channel | Reuses proven pattern, no new technology |
| WASM runtime | wazero in automation service | ADR-004 confirmed, pure Go, sandboxed |
| Plugin access | Host functions only, no direct DB | Security boundary, controlled API surface |
| Video/voice | LiveKit external server, SDK in Chat service | Not a custom service, proven infrastructure |
| Desktop modules | Lazy-loaded React modules with shared shell | Performance, role-based visibility, per-customer configuration |
| Build order | Desktop -> Work -> Video -> Biz -> Automation | Dependencies flow left to right |

---

## Sources

- **HIGH confidence:** All findings based on direct analysis of the existing codebase (4 service implementations, docker-compose, proto files, config structure, WebSocket hub, migration patterns)
- **HIGH confidence (PostgreSQL LISTEN/NOTIFY):** Standard PostgreSQL feature, well-documented, used by pgx library that KMU Hub already depends on
- **HIGH confidence (wazero):** Decided in ADR-004, pure Go WASM runtime, well-established in Go ecosystem
- **MEDIUM confidence (LiveKit integration pattern):** Based on LiveKit's documented architecture (SFU + Go Server SDK). Exact API calls need verification against current LiveKit SDK version during implementation phase.
- **MEDIUM confidence (Desktop module architecture):** Based on standard React patterns. Exact Zustand vs. Context decision and module registration API need validation during desktop phase.
- **LOW confidence (Event volume limits):** PostgreSQL LISTEN/NOTIFY limit of ~10,000 events/second is an estimate from community benchmarks. Actual limits depend on payload size and PostgreSQL configuration. Needs load testing during automation phase.

---

*Architecture research: 2026-02-07*

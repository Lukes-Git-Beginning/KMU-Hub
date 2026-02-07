# Technology Stack: New Modules for KMU Hub

**Project:** KMU Hub - All-in-One Workplace for DACH SMBs
**Researched:** 2026-02-07
**Scope:** Stack additions for 8 new modules (PM, Calendar, Email, Video, HR, Finance, Automation, Plugins)
**Research limitations:** WebSearch and WebFetch were unavailable. All recommendations are based on training data (cutoff ~May 2025). Versions MUST be verified against official sources before adoption. Confidence levels reflect this limitation.

---

## Existing Stack (Verified from Codebase)

| Component | Technology | Version | Status |
|-----------|-----------|---------|--------|
| Language | Go | 1.25.6 | In use |
| HTTP Router | chi/v5 | 5.2.4 | In use |
| gRPC | google.golang.org/grpc | 1.78.0 | In use |
| Protobuf | google.golang.org/protobuf | 1.36.11 | In use |
| Database driver | pgx/v5 | 5.8.0 | In use |
| Redis client | go-redis/v9 | 9.17.3 | In use |
| JWT | golang-jwt/v5 | 5.3.1 | In use |
| Object storage | minio-go/v7 | 7.0.98 | In use |
| WebSocket | coder/websocket | 1.8.14 | In use |
| Image processing | disintegration/imaging | 1.6.2 | In use |
| Language detection | pemistahl/lingua-go | 1.4.0 | In use |
| Decimal math | shopspring/decimal | 1.4.0 | In use |
| Metrics | prometheus/client_golang | 1.23.2 | In use |
| Testing | stretchr/testify | 1.11.1 | In use |
| Config | sethvargo/go-envconfig | 1.3.0 | In use |
| UUID | google/uuid | 1.6.0 | In use |
| Crypto | golang.org/x/crypto | 0.47.0 | In use |
| Frontend | React 19 + TypeScript 5.7 | ^19.0.0 | In use |
| Desktop | Electron 33 | ^33.0.0 | In use |
| Build | electron-vite 2.4 | ^2.4.0 | In use |
| CSS | Tailwind CSS 4 | ^4.0.0 | In use |
| Routing (FE) | react-router-dom 7 | ^7.0.0 | In use |
| Testing (FE) | Vitest 2.1 | ^2.1.0 | In use |
| Database | PostgreSQL 16 Alpine | 16-alpine | Docker |
| Cache | Redis 7 Alpine | 7-alpine | Docker |
| File Storage | MinIO | latest | Docker |

---

## Module 1: Project Management

### Backend Libraries (Go)

No external Go libraries needed -- project management is pure domain logic built on PostgreSQL.

| Component | Approach | Why |
|-----------|----------|-----|
| Task CRUD | Custom service layer | Tasks are domain objects with custom fields support, same pattern as CRM entities |
| Kanban boards | Custom service layer + PostgreSQL | Board = pipeline stages pattern already proven in CRM Deals Pipeline |
| Gantt/Timeline | Custom service layer | Timeline rendering is frontend-only; backend provides date ranges + dependencies |
| Dependencies | PostgreSQL `task_dependencies` table | Directed acyclic graph (DAG) — topological sort in Go stdlib |
| Recurring tasks | `robfig/cron/v3` | Cron expression parsing for recurring task schedules |

**Specific recommendation:**

| Library | Version | Purpose | Confidence |
|---------|---------|---------|------------|
| `github.com/robfig/cron/v3` | v3.0.1 | Cron expression parsing for recurring tasks and automation schedules | MEDIUM -- widely used, version from training data |

**Why no Gantt library on the backend?** Gantt rendering is 100% a frontend concern. The backend stores `start_date`, `end_date`, `duration`, `dependencies[]`, and `progress_percent`. The frontend calculates layout. This keeps the backend lean.

**Architecture note:** Reuse the exact same patterns from CRM Deals Pipeline. Kanban boards ARE pipeline stages. Tasks ARE deals with different fields. The custom fields engine already exists. Do NOT build a separate entity framework -- extend the existing one.

### Frontend Libraries

| Library | Purpose | Why This One | Confidence |
|---------|---------|-------------|------------|
| `@dnd-kit/core` + `@dnd-kit/sortable` | Drag-and-drop for Kanban boards | React 18/19 compatible, accessible, lightweight. Successor to react-beautiful-dnd which is deprecated. | MEDIUM |
| `frappe-gantt` OR custom SVG | Gantt chart rendering | frappe-gantt is lightweight (no heavy dependencies), renders to SVG. Alternative: build custom with SVG since Gantt is relatively straightforward geometry. | LOW -- verify current maintenance status |
| `date-fns` | Date manipulation for timelines | Tree-shakeable, no mutation, TypeScript-first. Do NOT use moment.js (deprecated, massive bundle). | MEDIUM |

**Why NOT react-beautiful-dnd?** Atlassian deprecated it. @dnd-kit is the community successor -- accessible, composable, framework-agnostic core with React bindings.

**Why NOT dhtmlxGantt or similar?** Commercial licenses, large bundle sizes, and overkill for SMB project management. For v1, a lightweight Gantt or even a custom SVG timeline view is sufficient. Full-featured Gantt can come in v2 if customers demand it.

---

## Module 2: Calendar & Scheduling

### Backend Libraries (Go)

| Library | Version | Purpose | Confidence |
|---------|---------|---------|------------|
| `github.com/emersion/go-webdav` | latest | CalDAV server implementation -- serves calendars to external clients (Outlook, Apple Calendar, Thunderbird) | MEDIUM |
| `github.com/emersion/go-ical` | latest (dependency of go-webdav) | iCal parsing/generation (RFC 5545) -- import/export .ics files, handle VEVENT/VTODO/VFREEBUSY | MEDIUM |

**Architecture decision: Build a CalDAV server, not just an API.**

Why: DACH SMBs use Outlook and Thunderbird. If your calendar only has a REST API, users must duplicate events. A CalDAV server lets Outlook sync bidirectionally with KMU Hub's calendar. This is a table-stakes feature for the DACH market where Outlook dominance is absolute.

**Implementation approach:**
- `go-webdav` provides the CalDAV protocol layer (HTTP-based, integrates with chi/v5 router)
- Storage backend is PostgreSQL (calendars, events, attendees tables)
- `go-ical` handles iCal format parsing/serialization
- Availability/free-busy is computed server-side from event data
- Recurring events: store RRULE in PostgreSQL, expand occurrences on query

**What NOT to use:**
- Do NOT use Radicale/Baikal as external CalDAV server -- adds infrastructure complexity for self-hosted customers. Embed the CalDAV protocol directly in the calendar microservice.
- Do NOT use Google Calendar API as primary -- violates EU data sovereignty. Support import/sync as optional connector later.

### Frontend Libraries

| Library | Purpose | Why This One | Confidence |
|---------|---------|-------------|------------|
| `@schedule-x/react` OR `@fullcalendar/react` | Calendar UI (day/week/month/agenda views) | Both are actively maintained. FullCalendar has better ecosystem but is larger. schedule-x is lighter and newer. | LOW -- verify current versions |
| `date-fns` | Date math (shared with PM module) | Already recommended above, single date library across modules | MEDIUM |
| `rrule` (npm) | Recurring event rule UI | Standard implementation of iCal RRULE in JS -- for building the recurrence picker UI | MEDIUM |

**Why FullCalendar over custom?** Calendar UI is deceptively complex (timezone handling, recurring events, drag-to-resize, multi-day events, week/month layout). Building custom is a multi-month effort. FullCalendar handles this and has Electron compatibility.

---

## Module 3: Email Integration

### Backend Libraries (Go)

| Library | Version | Purpose | Confidence |
|---------|---------|---------|------------|
| `github.com/emersion/go-imap/v2` | v2.x | IMAP client -- fetch emails from customer mail servers (per-customer IMAP config) | MEDIUM |
| `github.com/emersion/go-smtp` | latest | SMTP client -- send emails through customer mail servers | MEDIUM |
| `github.com/emersion/go-message` | latest | Email message parsing (MIME, attachments, HTML/plain text) | MEDIUM |
| `github.com/emersion/go-sasl` | latest | SASL authentication for IMAP/SMTP (PLAIN, LOGIN, OAUTH2) | MEDIUM |
| `github.com/jhillyerd/enmime` | latest | Alternative/supplement for complex MIME parsing (better at handling malformed emails) | MEDIUM |
| `golang.org/x/net/html` | stdlib | HTML sanitization for email display (prevent XSS from HTML emails) | HIGH |
| `github.com/microcosm-cc/bluemonday` | latest | HTML sanitization policy engine -- whitelist safe HTML for email rendering | MEDIUM |

**Architecture: Email is a PROXY, not a mail server.**

Critical decision: KMU Hub does NOT become a mail server. It connects to each customer's existing IMAP/SMTP server (Outlook 365, Gmail Workspace, local Exchange, etc.) and provides a unified inbox within the Hub. Each user configures their mail credentials (encrypted at rest).

**Why this matters for DACH:**
- German SMBs often use hosted Exchange or local mail servers
- They will NOT migrate email to a new provider
- They WILL use an integrated view if it syncs with their existing setup
- IMAP IDLE (push notifications) is essential for real-time inbox updates

**Implementation approach:**
- `go-imap/v2` handles IMAP connection pooling, IDLE monitoring, and folder sync
- `go-smtp` sends through customer's SMTP relay (preserving their domain/DKIM/SPF)
- Emails cached in PostgreSQL for search + offline access
- Attachments stored in MinIO (reuse existing file infrastructure)
- Per-user encrypted IMAP/SMTP credentials (AES-256-GCM, key from env)
- OAuth2 support for Microsoft 365 and Google Workspace via `go-sasl`

**What NOT to build:**
- Do NOT build a full MTA (Mail Transfer Agent) -- enormous security surface, deliverability nightmare
- Do NOT use a third-party email API (SendGrid, Mailgun) for reading customer mail -- violates data sovereignty
- Do NOT store emails only in cache -- PostgreSQL is source of truth for synced emails (consistent with no-dual-write rule)

### Frontend Libraries

| Library | Purpose | Why This One | Confidence |
|---------|---------|-------------|------------|
| `dompurify` | Sanitize HTML email for safe rendering in Electron | Standard for HTML sanitization in browser/Electron contexts | MEDIUM |
| `quill` or `tiptap` | Rich text email composer | Quill is simpler and lighter. tiptap (ProseMirror-based) is more extensible. For email composition, Quill suffices. | MEDIUM |

**Security warning for Electron:** Rendering HTML emails in Electron is a known attack vector. Emails MUST be rendered in a sandboxed iframe/webview with `sandbox` attribute, `Content-Security-Policy`, and all scripts stripped. Use `bluemonday` server-side AND `dompurify` client-side as defense in depth.

---

## Module 4: Video & Voice Calls

### Backend Libraries (Go)

| Library | Version | Purpose | Confidence |
|---------|---------|---------|------------|
| `github.com/livekit/server-sdk-go/v2` | v2.x | LiveKit server SDK -- room management, token generation, webhook handling | MEDIUM |
| `github.com/livekit/protocol` | latest (dependency) | LiveKit protocol types, room/participant models | MEDIUM |

**Architecture: LiveKit is a separate infrastructure service, not embedded.**

LiveKit server runs as its own Docker container alongside PostgreSQL/Redis/MinIO. The KMU Hub video microservice is a thin orchestration layer:
- Creates rooms via LiveKit API
- Generates participant tokens (JWT signed with LiveKit API key)
- Handles webhooks (participant joined/left, recording ready)
- Stores call history in PostgreSQL

**Docker Compose addition:**

```yaml
livekit:
  image: livekit/livekit-server:latest
  ports:
    - "7880:7880"   # HTTP
    - "7881:7881"   # WebRTC/TCP
    - "7882:7882/udp" # WebRTC/UDP
  environment:
    LIVEKIT_KEYS: "devkey: secret"
  volumes:
    - ./livekit.yaml:/etc/livekit.yaml
  command: ["--config", "/etc/livekit.yaml"]

# Optional: LiveKit egress for recording
livekit-egress:
  image: livekit/egress:latest
  environment:
    EGRESS_CONFIG_FILE: /etc/egress.yaml
  volumes:
    - ./egress.yaml:/etc/egress.yaml
```

**What NOT to do:**
- Do NOT embed WebRTC directly using pion/webrtc -- LiveKit already handles SFU, simulcast, bandwidth estimation, TURN, etc. Reimplementing this is years of work.
- Do NOT use Jitsi -- Java-based, harder to integrate with Go stack, heavier resource requirements for self-hosted
- Do NOT skip TURN server -- required for ~15% of users behind symmetric NATs. LiveKit has built-in TURN.

### Frontend Libraries

| Library | Purpose | Why This One | Confidence |
|---------|---------|-------------|------------|
| `@livekit/components-react` | Pre-built React components for video UI (participant tiles, controls, screen share) | Official LiveKit React SDK -- handles all WebRTC complexity | MEDIUM |
| `livekit-client` | Low-level LiveKit client SDK (room connection, track management) | Required by components-react, also useful for custom UI | MEDIUM |

**Screen sharing in Electron:** Electron has `desktopCapturer` API for screen/window capture. LiveKit's client SDK supports custom media sources, so you pipe Electron's desktopCapturer stream into LiveKit. This is well-documented in LiveKit's Electron guide.

---

## Module 5: HR Module

### Backend Libraries (Go)

No additional Go libraries needed beyond what already exists. HR module is pure domain logic.

| Component | Approach | Why |
|-----------|----------|-----|
| Leave requests | Custom service + PostgreSQL | Approval workflows, balance tracking, calendar integration |
| Time tracking | Custom service + PostgreSQL | Start/stop timers, manual entry, project/task association |
| Employee profiles | Extend existing user model | Add HR-specific fields (department, position, start_date, manager_id) |
| Onboarding | Custom service + workflow engine | Checklist templates, task assignments, deadline tracking |
| Work schedule | Custom service + PostgreSQL | Template-based schedules, supports part-time/flexible for German Arbeitszeitgesetz |

**Architecture note:** The HR module is a new microservice (`cmd/hr`, `internal/hr/`) following the same gRPC pattern. It connects to the auth service for user data and to the calendar service for leave visualization.

**DACH-specific considerations:**
- German labor law requires precise time tracking (Arbeitszeiterfassungspflicht since 2022 BAG ruling)
- Leave types: Urlaub (vacation), Krankheit (sick), Sonderurlaub (special leave), Elternzeit (parental)
- Minimum 20 Urlaubstage (24 Werktage) by BUrlG
- Overtime tracking with monthly caps (ArbZG: max 8h/day, extendable to 10h with compensation)

**What NOT to build for v1:**
- Payroll calculation -- extremely complex, country-specific, heavily regulated. Integrate with DATEV/Lexware instead.
- Full HRIS (Human Resource Information System) -- focus on daily operations (leave, time, onboarding), not strategic HR

### Frontend Libraries

No new libraries needed. Time tracking UI uses existing form components. Calendar integration renders in the shared calendar module.

---

## Module 6: Finance Module

### Backend Libraries (Go)

| Library | Version | Purpose | Confidence |
|---------|---------|---------|------------|
| `github.com/signintech/gopdf` OR `github.com/jung-kurt/gofpdf` | latest | PDF generation for quotes and invoices | MEDIUM -- verify which is better maintained in 2026 |
| `github.com/pdfcpu/pdfcpu` | latest | PDF manipulation (merge, watermark) -- useful for attaching terms & conditions | MEDIUM |
| `github.com/shopspring/decimal` | 1.4.0 | Already in use -- financial calculations MUST use decimal, never float64 | HIGH (in use) |
| `encoding/xml` (stdlib) | stdlib | DATEV XML export generation | HIGH |

**DATEV Integration -- Critical for DACH:**

DATEV is the dominant accounting software in DACH (used by ~50% of German tax advisors). Integration approach:

| Method | Description | Complexity | Recommendation |
|--------|-------------|------------|----------------|
| DATEV CSV/XML Export | Generate DATEV-compatible export files (Buchungsstapel format) | LOW | **v1 -- do this first** |
| DATEV Unternehmen Online API | REST API for direct data push | MEDIUM | v2 -- requires DATEV partner registration |
| DATEV Connect Online | OAuth2-based cloud API | HIGH | v3 -- newest API, most capability |

**v1 recommendation: DATEV CSV/XML export.** Generate files in DATEV's "Buchungsstapel" format that accountants can import directly into DATEV. This covers 80% of use cases without API registration overhead. The format is documented in DATEV's public specification (DATEV-Format Beschreibung).

**Invoice generation approach:**
- Quote/Invoice data model in PostgreSQL (line items, tax rates, terms, numbering)
- PDF generation via Go library (company letterhead as template)
- Sequential invoice numbering (German GoBD requirement -- no gaps allowed)
- PDF/A format for archiving (GoBD compliance)
- XRechnung/ZUGFeRD for B2G (Behorden) invoicing

**ZUGFeRD -- important for DACH:**

| Library | Purpose | Confidence |
|---------|---------|------------|
| Custom XML generation (encoding/xml) | ZUGFeRD/Factur-X XML embedding in PDF | MEDIUM |

ZUGFeRD is a German standard for electronic invoices (PDF + embedded XML). Since November 2020, B2G invoices in Germany must be electronic (XRechnung). ZUGFeRD 2.1+ is compatible with Factur-X (French standard) and EN 16931 (EU standard). This is a differentiator for DACH.

**What NOT to build:**
- Full double-entry bookkeeping -- leave this to DATEV/Lexware/sevDesk. KMU Hub generates documents and exports data.
- Payment processing -- integrate with existing bank/payment setup, do not handle money movement
- Tax calculation engine -- too jurisdiction-specific. Store tax rates as configurable values.

### Frontend Libraries

| Library | Purpose | Why | Confidence |
|---------|---------|-----|------------|
| Existing React components | Invoice/quote forms | Standard form UIs, no special library needed | HIGH |
| `recharts` or `victory` | Financial dashboards/charts | Revenue charts, payment status, overdue tracking. recharts is lighter. | MEDIUM |

---

## Module 7: Automation Engine

### Backend Libraries (Go)

| Library | Version | Purpose | Confidence |
|---------|---------|---------|------------|
| `github.com/robfig/cron/v3` | v3.0.1 | Scheduled trigger execution (same as PM module) | MEDIUM |
| `github.com/expr-lang/expr` | latest | Expression evaluation engine for workflow conditions ("deal.value > 10000 AND deal.stage == 'Won'") | MEDIUM |
| `github.com/dop251/goja` | latest | JavaScript runtime for complex automation scripts (alternative to expr for power users) | LOW -- evaluate need |

**Architecture: Event-driven workflow engine.**

The automation engine is the nervous system of KMU Hub. It connects all modules through an event bus.

**Core components:**
1. **Event Bus** -- Redis Pub/Sub (already available) for real-time events across services
2. **Trigger Evaluator** -- Listens to events, evaluates conditions
3. **Action Executor** -- Executes actions (create task, send email, update field, notify user)
4. **Workflow Store** -- PostgreSQL stores workflow definitions (JSON-based DSL)

**Implementation approach:**

```
Event Flow:
Service emits event -> Redis Pub/Sub -> Automation Engine subscribes
-> Evaluate trigger conditions (expr-lang/expr) -> Execute actions -> Emit result events
```

**Why `expr-lang/expr`?**
- Pure Go, no CGO
- Safe by default (no access to filesystem/network)
- Type-checked expressions
- Fast compilation + evaluation
- Supports custom functions
- Perfect for "if condition then action" workflows

**Workflow definition format (JSON/YAML DSL, NOT a code language):**

```json
{
  "trigger": { "event": "deal.stage_changed", "condition": "deal.stage == 'Won'" },
  "actions": [
    { "type": "create_task", "params": { "title": "Send welcome package", "assignee": "deal.owner_id" } },
    { "type": "send_email", "params": { "template": "deal_won", "to": "deal.contact.email" } }
  ]
}
```

**Why NOT a full BPMN engine (Camunda/Temporal)?**
- Massive overkill for SMB automation
- Heavy infrastructure requirements (Temporal needs its own DB + multiple services)
- 1-dev team cannot maintain a BPMN engine
- Config-based DSL covers 95% of SMB workflows
- WASM plugins handle the other 5%

**What NOT to use:**
- Do NOT use Temporal/Cadence -- distributed workflow engines designed for microservice orchestration at scale. Wrong tool for "when deal closes, create a task."
- Do NOT use Camunda -- Java-based BPMN engine, way too heavy
- Do NOT use Lua scripting -- WASM is already chosen for complex extensibility, adding Lua creates two plugin systems
- Do NOT build a visual flow editor for v1 -- config-based YAML/JSON first, visual editor is a v2 feature

### Frontend Libraries

| Library | Purpose | Confidence |
|---------|---------|------------|
| `reactflow` | Visual workflow builder (v2 feature, not v1) | MEDIUM |

For v1, workflow creation is a form-based UI: select trigger, set condition, choose actions. No visual drag-and-drop flow editor needed yet.

---

## Module 8: Plugin System

### Backend Libraries (Go)

| Library | Version | Purpose | Confidence |
|---------|---------|---------|------------|
| `github.com/tetratelabs/wazero` | v1.8.x or latest v1 | WASM runtime -- pure Go, zero CGO, runs WASM plugins in sandbox | MEDIUM -- verify latest v1.x |

**Why wazero over alternatives?**

| Runtime | Language | CGO Required | Maturity | Decision |
|---------|----------|-------------|----------|----------|
| **wazero** | Pure Go | No | Production | **USE THIS** |
| wasmtime-go | Go bindings to Rust | Yes (CGO) | Production | Reject -- CGO breaks cross-compilation, complicates Docker builds |
| wasmer-go | Go bindings to C | Yes (CGO) | Declining | Reject -- less maintained, CGO dependency |

wazero is the clear winner for a Go project. It is the only production-ready WASM runtime that is pure Go. No CGO means:
- Simple `go build` produces a static binary
- Alpine Docker images work without extra libraries
- Cross-compilation works trivially
- No Rust/C toolchain in CI

**Plugin system architecture (two tiers):**

**Tier 1: Config-Based Plugins (80% of customization)**
- Custom fields (already built)
- Workflow automations (Module 7)
- Dashboard layouts (JSON config)
- Report templates (JSON config)
- Form layouts (JSON config)
- No new library needed -- this is pure application logic

**Tier 2: WASM Plugins (20% of customization)**
- Custom validation logic
- External system connectors
- Complex business rules
- Data transformation pipelines

**WASM Plugin Host Interface:**
- Define a stable ABI (Application Binary Interface) using protobuf for data exchange
- Plugin receives serialized protobuf input, returns serialized protobuf output
- Host provides functions: `kv_get`, `kv_set`, `http_fetch` (proxied through host), `log`
- Plugin CANNOT access: filesystem, network directly, database, other plugins' data
- Resource limits: memory cap, execution timeout, CPU quota

**Plugin SDK languages (for plugin authors):**
- Rust (first-class, smallest WASM output)
- Go (TinyGo compiles to WASM)
- AssemblyScript (TypeScript-like, compiles to WASM)
- C/C++ (via Emscripten)

**What NOT to do:**
- Do NOT use Lua/JavaScript as plugin runtime -- WASM is already decided (ADR-004), adding another runtime creates maintenance burden
- Do NOT expose raw database access to plugins -- all data access through host functions
- Do NOT allow plugins to modify core UI -- plugins extend through defined extension points only

---

## Shared Infrastructure Additions

### New Docker Compose Services

| Service | Image | Purpose | Required For |
|---------|-------|---------|-------------|
| LiveKit Server | `livekit/livekit-server:latest` | WebRTC SFU for video/voice calls | Video module |
| LiveKit Egress | `livekit/egress:latest` | Call recording + streaming | Video module (optional for v1) |
| TURN Server | Built into LiveKit | NAT traversal for WebRTC | Video module (included) |

**No other new infrastructure services needed.** All other modules use existing PostgreSQL + Redis + MinIO.

### New Shared Go Libraries (Cross-Module)

| Library | Version | Purpose | Used By | Confidence |
|---------|---------|---------|---------|------------|
| `github.com/robfig/cron/v3` | v3.0.1 | Cron scheduling | PM (recurring tasks), Automation (scheduled triggers), Calendar (recurring events backend) | MEDIUM |
| `github.com/expr-lang/expr` | latest | Expression evaluation | Automation (conditions), Plugin system (config expressions) | MEDIUM |
| `github.com/microcosm-cc/bluemonday` | latest | HTML sanitization | Email (sanitize incoming HTML), Chat (rich text messages future) | MEDIUM |

### New Shared Frontend Libraries

| Library | Purpose | Used By | Confidence |
|---------|---------|---------|------------|
| `date-fns` | Date manipulation | PM, Calendar, HR, Finance | MEDIUM |
| `@dnd-kit/core` + `@dnd-kit/sortable` | Drag-and-drop | PM (Kanban), Calendar (event drag) | MEDIUM |
| `dompurify` | Client-side HTML sanitization | Email rendering | MEDIUM |
| `recharts` | Charts and dashboards | Finance, CRM reports, HR dashboards | MEDIUM |
| `@livekit/components-react` | Video call UI | Video module | MEDIUM |
| `livekit-client` | LiveKit client SDK | Video module | MEDIUM |

---

## New Microservice Map

Following the existing pattern (each module = gRPC service + Dockerfile + health check):

| Service | gRPC Port | Health Port | Proto File | Dependencies |
|---------|-----------|-------------|------------|-------------|
| `cmd/pm` | :50054 | :9094 | `proto/pm/v1/pm.proto` | PostgreSQL |
| `cmd/calendar` | :50055 | :9095 | `proto/calendar/v1/calendar.proto` | PostgreSQL, Redis (event reminders) |
| `cmd/email` | :50056 | :9096 | `proto/email/v1/email.proto` | PostgreSQL, MinIO (attachments), Redis (sync status) |
| `cmd/video` | :50057 | :9097 | `proto/video/v1/video.proto` | PostgreSQL, Redis, LiveKit Server |
| `cmd/hr` | :50058 | :9098 | `proto/hr/v1/hr.proto` | PostgreSQL |
| `cmd/finance` | :50059 | :9099 | `proto/finance/v1/finance.proto` | PostgreSQL, MinIO (PDF storage) |
| `cmd/automation` | :50060 | :9100 | `proto/automation/v1/automation.proto` | PostgreSQL, Redis (event bus) |

**Plugin system is NOT a separate service.** It is a library (`internal/plugin/`) loaded by each service that supports plugins. The plugin registry (which plugins are installed, their configs) lives in PostgreSQL, managed by the gateway.

---

## Alternatives Rejected

| Category | Recommended | Rejected | Why Rejected |
|----------|-------------|----------|-------------|
| WASM Runtime | wazero | wasmtime-go, wasmer-go | CGO requirement breaks build simplicity, cross-compilation |
| Workflow Engine | Custom + expr-lang | Temporal, Camunda, n8n | Massive infrastructure overhead for simple SMB workflows |
| Calendar Protocol | go-webdav (embedded CalDAV) | External Radicale/Baikal | Extra service to manage, harder self-hosted deployment |
| Email Approach | IMAP/SMTP proxy | Built-in MTA, SendGrid | MTA is security nightmare; external APIs violate EU sovereignty |
| Gantt Library (BE) | None (frontend-only) | Any server-side Gantt | Gantt is a rendering concern, not a data concern |
| PDF Generation | gopdf/gofpdf | wkhtmltopdf, Chromium headless | External binary dependencies, harder Docker builds, overkill |
| Expression Engine | expr-lang/expr | goja (full JS), Lua | expr is simpler, safer, faster for condition evaluation |
| Video | LiveKit (dedicated service) | pion/webrtc (DIY), Jitsi | Reimplementing SFU is years of work; Jitsi is Java, heavy |
| Chart Library (FE) | recharts | Chart.js, D3, Victory | recharts is React-native, lightweight, sufficient for dashboards |
| Date Library (FE) | date-fns | moment.js, dayjs, Luxon | Tree-shakeable, immutable, TypeScript-first. moment is deprecated. |
| DnD Library (FE) | @dnd-kit | react-beautiful-dnd, react-dnd | react-beautiful-dnd is deprecated; @dnd-kit is accessible, maintained |
| Rich Text Editor | Quill (for email) | TinyMCE, CKEditor, Slate | Quill is lightweight, sufficient for email composition. Slate is too low-level. |
| Calendar UI (FE) | FullCalendar or schedule-x | Custom build | Calendar layout is deceptively complex -- use a battle-tested library |

---

## Version Verification Needed

**IMPORTANT:** All versions below are from training data (cutoff ~May 2025). Before adding to go.mod or package.json, verify the current latest version.

| Library | Training Data Version | Verify At |
|---------|----------------------|-----------|
| `tetratelabs/wazero` | v1.8.x | github.com/tetratelabs/wazero/releases |
| `livekit/server-sdk-go` | v2.x | github.com/livekit/server-sdk-go/releases |
| `livekit/livekit-server` (Docker) | latest | hub.docker.com/r/livekit/livekit-server |
| `emersion/go-imap` | v2.x | github.com/emersion/go-imap/releases |
| `emersion/go-webdav` | check | github.com/emersion/go-webdav/releases |
| `emersion/go-smtp` | check | github.com/emersion/go-smtp/releases |
| `robfig/cron` | v3.0.1 | github.com/robfig/cron/releases |
| `expr-lang/expr` | check | github.com/expr-lang/expr/releases |
| `microcosm-cc/bluemonday` | check | github.com/microcosm-cc/bluemonday/releases |
| `@dnd-kit/core` (npm) | check | npmjs.com/package/@dnd-kit/core |
| `@livekit/components-react` (npm) | check | npmjs.com/package/@livekit/components-react |
| `recharts` (npm) | check | npmjs.com/package/recharts |
| `date-fns` (npm) | check | npmjs.com/package/date-fns |
| `@fullcalendar/react` (npm) | check | npmjs.com/package/@fullcalendar/react |

---

## Dependency Budget

Current Go dependencies: 16 direct. Proposed additions: 8 new direct dependencies.

| New Dependency | Module | Justification |
|----------------|--------|---------------|
| `robfig/cron/v3` | PM, Automation, Calendar | Cron scheduling -- stdlib has no cron parser |
| `expr-lang/expr` | Automation | Expression evaluation -- safe, fast, pure Go |
| `tetratelabs/wazero` | Plugin system | WASM runtime -- only pure-Go option |
| `livekit/server-sdk-go/v2` | Video | LiveKit integration -- required for video |
| `emersion/go-imap/v2` | Email | IMAP client -- no stdlib alternative |
| `emersion/go-smtp` | Email | SMTP client -- stdlib net/smtp is too basic |
| `emersion/go-webdav` | Calendar | CalDAV server -- required for Outlook sync |
| `microcosm-cc/bluemonday` | Email | HTML sanitization -- security requirement |

Current npm dependencies: 3 production. Proposed additions: ~8 new production dependencies.

| New Dependency | Module | Justification |
|----------------|--------|---------------|
| `date-fns` | PM, Calendar, HR | Date manipulation across modules |
| `@dnd-kit/core` | PM | Kanban drag-and-drop |
| `@dnd-kit/sortable` | PM | Sortable lists |
| `dompurify` | Email | HTML sanitization for email rendering |
| `recharts` | Finance, Reports | Charts for dashboards |
| `livekit-client` | Video | LiveKit WebRTC client |
| `@livekit/components-react` | Video | Pre-built video UI components |
| `@fullcalendar/react` | Calendar | Calendar UI views |

**Total after expansion:** ~24 Go direct deps, ~11 npm production deps. This is lean for an all-in-one platform.

---

## Self-Hosted Compatibility Check

Every recommendation verified against self-hosted Docker Compose constraint:

| Component | Self-Hosted Compatible | Notes |
|-----------|----------------------|-------|
| LiveKit Server | YES | Docker image available, requires UDP port exposure |
| LiveKit Egress | YES | Docker image, needs ffmpeg (included in image) |
| wazero | YES | Pure Go, compiled into service binary |
| go-imap/go-smtp | YES | Client libraries, connect to customer's mail server |
| go-webdav (CalDAV) | YES | Embedded in calendar service binary |
| expr-lang/expr | YES | Pure Go, compiled into service binary |
| All frontend libs | YES | Bundled into Electron app |
| PostgreSQL | YES | Already in Docker Compose |
| Redis | YES | Already in Docker Compose |
| MinIO | YES | Already in Docker Compose |

**No SaaS-only dependencies introduced.** Every component runs in Docker Compose.

---

## Installation Commands (When Ready)

```bash
# Backend - new Go dependencies
cd backend
go get github.com/robfig/cron/v3@latest
go get github.com/expr-lang/expr@latest
go get github.com/tetratelabs/wazero@latest
go get github.com/livekit/server-sdk-go/v2@latest
go get github.com/emersion/go-imap/v2@latest
go get github.com/emersion/go-smtp@latest
go get github.com/emersion/go-webdav@latest
go get github.com/microcosm-cc/bluemonday@latest

# Frontend - new npm dependencies
cd desktop
npm install date-fns @dnd-kit/core @dnd-kit/sortable dompurify recharts livekit-client @livekit/components-react @fullcalendar/react @fullcalendar/daygrid @fullcalendar/timegrid @fullcalendar/interaction
npm install -D @types/dompurify
```

**Do NOT install all at once.** Install per-module as each phase begins. This keeps go.mod and package.json clean and avoids unused dependencies in intermediate builds.

---

## Confidence Assessment

| Area | Confidence | Reason |
|------|------------|--------|
| Project Management stack | HIGH | No new libraries needed, reuses existing patterns |
| Calendar/CalDAV stack | MEDIUM | emersion/go-webdav is known but versions unverified |
| Email (IMAP/SMTP) stack | MEDIUM | emersion/go-imap v2 is known but API changes unverified |
| Video (LiveKit) stack | MEDIUM | LiveKit is well-established, Go SDK version unverified |
| HR module stack | HIGH | Pure domain logic, no new dependencies |
| Finance stack | MEDIUM | PDF generation library choice needs verification |
| Automation engine stack | MEDIUM | expr-lang/expr is well-known, version unverified |
| Plugin system (wazero) stack | MEDIUM | wazero v1.x is established, exact latest version unverified |
| Frontend libraries | LOW-MEDIUM | npm ecosystem moves fast, all versions need verification |
| Self-hosted compatibility | HIGH | Architecture verified against Docker Compose constraint |
| DACH-specific requirements (DATEV, ZUGFeRD, CalDAV) | MEDIUM | Domain knowledge solid, specific API versions unverified |

---

## Sources

All recommendations based on training data (cutoff ~May 2025). WebSearch and WebFetch were unavailable during research.

- Codebase analysis: `backend/go.mod`, `desktop/package.json`, `deploy/docker/docker-compose.yml`
- Architecture decisions: `docs/ARCHITECTURE.md` (ADR-001 through ADR-006)
- Project context: `.planning/PROJECT.md`
- Historical roadmap: `docs/ROADMAP.md`

**Verification required before implementation:** All library versions should be checked against their GitHub releases pages and official documentation. This is flagged as a pre-implementation task for each phase.

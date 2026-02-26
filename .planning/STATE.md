# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-08)

**Core value:** Every employee completes their entire workday without opening another program
**Current focus:** Phase 20 (Plugin System + Industry Templates) -- COMPLETE
**Recent strategy changes:** Phase 19 pivoted from Abacus+RmA to DATEV API + Lexware Office (full DACH coverage: Bexio CH + Lexware DE + DATEV DE/AT)

## Current Position

Phase: 20 of 20 (Plugin System + Industry Templates) -- COMPLETE
Plan: 4 of 4 complete
Status: Phase 20 COMPLETE. All plans implemented.
Last activity: 2026-02-26 -- Phase 20 complete (all 20 phases done!)
Next: Beta release preparation

Progress: [████████████████████████████████████] 100% (103/103 plans across phases 4-20)

## Performance Metrics

**Velocity:**
- Total plans completed: 73
- Average duration: ~7 minutes
- Total execution time: ~8h 8min

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 04 | 3/3 | ~46min | ~15min |
| 05 | 7/7 | ~66min | ~9min |
| 06 | 10/10 | ~88min | ~8.8min |
| 07 | 9/9 | ~48min | ~5min |
| 08 | 9/9 | ~45min | ~5min |

| 09 | 9/9 | ~62min | ~6.9min |
| 10 | 7/7 | ~80min | ~11min |
| 11 | 6/6 | ~57min | ~9.5min |

| 12 | 7/7 | ~75min | ~10.7min |
| 13 | 4/4 | ~64min | ~16min |

| 14 | 4/4 | ~33min | ~8min |
| 15 | 3/3 | ~19min | ~6min |
| 16 | 3/3 | ~48min | ~16min |
| 17 | 3/3 | ~16min | ~5min |
| 17.5 | 3/3 | ~32min | ~11min |
| 18 | 4/4 | ~65min | ~16min |
| 19 | 2/2 | ~50min | ~25min |

**Recent Trend:**
- Phases 9-11 (Compliance & Comms milestone) all complete
- Phase 12 (Rechnungen & Finanzen) COMPLETE -- all 7 plans done (incl. 2 gap closure)
- Phase 13 (HR & Zeiterfassung) COMPLETE -- all 4 plans done (proto, services, gRPC+gateway, frontend)
- Phase 14 (Event Infrastructure + Unified Inbox) COMPLETE -- all 4 plans done (proto, services, gRPC+gateway, frontend)
- Phase 15 (CalDAV/CardDAV Integration) COMPLETE -- all 3 plans done (data foundation, backend adapters, gateway+frontend+push)
- Phase 16 (Automation Engine) COMPLETE -- all 3 plans done (data foundation + workflow engine + frontend)
- Phase 17 (Teams & Slack Integration) COMPLETE -- all 3 plans done (data foundation, forwarder + adapters, frontend)
- Phase 17.5 (Gast-Chat) COMPLETE -- all 3 plans done (data foundation, services+gateway, frontend SPA)
- Phase 18 (Bexio Integration) COMPLETE -- all 4 plans done (data foundation + sync engine + gRPC/gateway + frontend)
- Phase 19 (DATEV API + Lexware Office) COMPLETE -- all 2 plans done (backend + frontend, 54 files, 8338 insertions)
- 99/99 plans done across Phases 4-19

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- [Roadmap]: Service consolidation -- 3 new backend services (Work, Biz, Automation) instead of 8 separate ones
- [Roadmap]: Gateway refactoring bundled with Phase 4 (before adding new services)
- [Roadmap]: Notifications first (unblocks all future modules)
- [Roadmap]: Full IMAP+SMTP email in v1 (user decision despite research suggesting deferral)
- [Roadmap]: Automation and Plugins last (need stable APIs from all other modules)
- [Roadmap]: Feature gap analysis expansion -- 13 to 18 phases, Meeting Management merged into Phase 8, Security & Compliance as Phase 9 gatekeeper, Documents & Files as Phase 11, 3 integration mini-phases (14-16)
- [Strategy]: Phase reorder -- Unified Inbox as Phase 14 (NEW), CalDAV shifted to 15, Automation vorgezogen to 16 (was 19), Teams/Slack to 17, Bexio to 18, Abacus+RmA merged into 19 (were 17+18), Plugins stay 20
- [Strategy]: Buchhaltung → Finanzen rename -- "Rechnungen & Finanzen" scope (invoices, quotes, dunning, DATEV), NOT full FiBu/Buchhaltung
- [Strategy]: Payroll anti-feature -- Lohnabrechnung NEVER built, integration-only via Bexio/Abacus/RmA. 8 endpoints struck from audit.
- [Strategy]: Industry modules (Fuhrpark, Produktion, Rapporte, etc.) are Phase 20 plugin candidates, NOT core endpoints
- [Strategy]: WOPI/OnlyOffice added to Phase 11 (Documents) for collaborative document editing
- [Strategy]: Event infrastructure (pg_notify + events table) built in Phase 14, prerequisite for Automation Engine
- [04-01]: WebSocket hub stays in main.go (cross-cutting, needs both chat + auth clients)
- [04-02]: Raw pgx over pgxlisten for event bus (pgxlisten pre-v1, unstable)
- [04-02]: Dual write (events table + pg_notify) for event durability
- [04-02]: DeliveryCallback pattern decouples notification service from WebSocket delivery
- [04-03]: Dual pg_notify channels: 'events' for notification processing, 'notification_delivery' for gateway WebSocket push
- [05-01]: electron-vite v5 with build.externalizeDeps (deprecated plugin replaced)
- [05-01]: TSconfig split: node (bundler resolution) + web (DOM, react-jsx, path aliases)
- [05-01]: CSP unsafe-inline for dev only (Vite HMR), production uses self only
- [05-02]: createHashRouter for Electron file:// protocol compatibility
- [05-02]: Auth init before render; GuestRoute guard on login page
- [05-03]: Routes/Route for CRM sub-navigation (module-level routing inside AppShell Outlet)
- [05-04]: WebSocket cache sync via queryClient.setQueryData with invalidation fallback
- [05-04]: Native push only when document.hasFocus() === false
- [05-05]: Widget registry pattern -- centralized definitions with lazy-loaded components
- [05-05]: Per-widget ErrorBoundary for crash isolation
- [05-06]: Dashboard service in gateway with direct DB access (not gRPC)
- [05-06]: localStorage as offline cache, server as source of truth
- [05-07]: 24h maxAge for TanStack Query cache with 5min staleTime
- [05-07]: Mutations blocked when offline via OfflineError in API client
- [05-07]: CORS origins include localhost:5173 for Electron dev
- [06-01]: Task constants prefixed with Task* to avoid collision with notification model priority constants
- [06-02]: Project key auto-normalizes to uppercase; validation rejects non-alphanumeric only
- [06-02]: Status service trusts caller for authorization (gRPC server checks membership)
- [06-02]: GetUserPreference returns nil when no preference set (caller applies defaults)
- [06-03]: Standalone tasks get task_number=0 (no project counter increment)
- [06-03]: Comment service depends on taskRepo.CreateActivity for activity logging
- [06-03]: @mention pattern uses @{uuid} format for deterministic user resolution
- [06-03]: Cycle detection only for blocking deps (blocks/blocked_by), not relates_to/duplicates
- [06-03]: MoveTask handles completed_at setting/clearing based on status is_closed flag
- [06-04]: gRPC server uses uuid.Nil + isAdmin=true (gateway handles auth)
- [06-04]: Template key auto-generated from name prefix + UUID suffix
- [06-04]: Work routes follow exact same RouteRegistrar pattern as CRM/Chat/Notification
- [06-05]: API types regenerated to include work/project/task endpoints from OpenAPI spec
- [06-05]: Project create dialog bundles status creation as sequential API calls after project POST
- [06-05]: My Tasks hooks auto-set assignee_id from auth store (user sees only own tasks)
- [06-06]: Client-side grouping for instant group switching without network roundtrip
- [06-06]: closestCorners collision detection for multi-container Kanban DnD
- [06-06]: Max 3 visual nesting levels on Kanban to keep board clean
- [06-06]: Subtask cards not independently draggable on Kanban (use list view or detail panel)
- [06-07]: Fixed overlay panel (CSS transform) for task detail slide-over, no Radix Sheet dependency
- [06-07]: Two-step file upload: multipart to MinIO via /files/upload, then JSON metadata to task files
- [06-07]: Nested Routes in ProjectDetailPage for board view vs task detail page
- [06-07]: Tab-based activity/comments view (Alle/Kommentare/Aktivitaet) for user control
- [06-08]: Context-aware auto-suggest shows banner but never auto-applies links
- [06-08]: Standalone tasks use task_number=0 with TASK system key
- [06-08]: CRM search API reused for entity linking (no new backend search endpoint)
- [06-08]: Custom fields reuse CRM engine with entity_type=task
- [06-08]: Move-to-project updates project_id via existing task update API
- [06-09]: Migration renumbered to 000031 to avoid collision with uncommitted time_entries migration
- [06-09]: Batch dependency fetching via useQueries for tasks with has_blocked_deps flag
- [06-09]: Gantt view is read-only in v1 (bars clickable, not draggable)
- [06-09]: Critical path uses forward/backward pass CPM with Kahn's topological sort
- [06-10]: Separate timeentry package under internal/work/ for clean separation of concerns
- [06-10]: Auto-stop previous timer in service layer ensures single-timer invariant at DB level
- [06-10]: Partial index idx_time_entries_active for O(1) active timer lookup
- [06-10]: requestAnimationFrame for timer display (smoother, auto-pauses in background tabs)
- [06-10]: Migration 000030 for time_entries (06-09 used 000031 for gantt view type)
- [07-01]: Separate calendar.proto file rather than extending work.proto (cleaner separation, same binary)
- [07-01]: Deferred FK constraints: resource_id FK added via ALTER TABLE in migration 000034 after resources table exists
- [07-01]: Calendar-prefixed model naming (CalendarEvent, EventCategory) to avoid collision with notification Event model
- [07-01]: 40 RPCs in CalendarService covering calendars, events, resources, bookings, holidays, preferences, LiveKit
- [07-02]: Separate calendar-types.ts instead of modifying auto-generated types.ts (openapi-typescript)
- [07-02]: calendar-client.ts fetch wrapper mirrors openapi-fetch auth/error patterns for pre-OpenAPI hooks
- [07-02]: Set<string> serialized as array in Zustand persist localStorage adapter
- [07-02]: CalendarLayout uses internal Routes/Route pattern (same as WorkLayout)
- [07-03]: Calendar permission hierarchy: view < edit < admin numeric levels, owner implicit admin
- [07-03]: EnsurePersonalCalendar on every ListByUser for auto-creation with DACH defaults
- [07-03]: Three-way recurring edit: this=exception, this_and_future=split with SetUntil, all=update master
- [07-03]: Event emitter optional via SetEventEmitter (same pattern as task service)
- [07-03]: rrule-go v1.8.2 re-added to go.mod (was missing from working tree despite 07-01 commit)
- [07-04]: HolidayFetcher interface abstracts Nager client for testability
- [07-04]: LiveKit disabled-by-default: empty config values = feature off
- [07-04]: BookingConflictError carries alternative resource suggestions
- [07-04]: Resource delete is soft-delete (is_active=false), bookings preserved
- [07-05]: CalendarService registered in same binary as WorkService (shared gRPC port :50055)
- [07-05]: CalendarRoutes uses ServiceName "work" to reuse existing gRPC connection from gateway
- [07-05]: Proto fields Date/DueDate are strings (YYYY-MM-DD), not Timestamps
- [07-UI]: Design integration: Darien's KalenderPage.tsx (1613 lines) merged from design/brainstorm
- [07-UI]: Adapter layer (adapters.ts) transforms backend types to UI types bidirectionally
- [07-UI]: Sonner toast system added to App.tsx (was pending todo)
- [07-UI]: Plans 07-06 to 07-09 made obsolete by design integration (saved ~4 plans)
- [07-UI]: D2 color system (globals.css) merged from design/brainstorm
- [07-UI]: New Radix UI components: alert-dialog, checkbox, dropdown-menu, sheet, switch
- [08-01]: tools.go with //go:build tools constraint to retain server-sdk-go/v2 in go.mod before app code imports it
- [08-01]: Domain-scoped model packages (internal/work/video/, meeting/, etc.) for Phase 8 models
- [08-01]: 31 RPCs in single VideoService covering calls, recording, meetings, notes/actions, presence
- [08-01]: Presence runtime state in Redis; only admin config (away_timeout_seconds) persisted in PostgreSQL
- [08-02]: errors.go added for domain errors (ErrEmojiRequired, ErrEmojiTooLong) following comment package pattern
- [08-02]: Empty batch returns early in service layer (no DB call) for efficiency
- [08-02]: Service returns empty slice (not nil) for reactions when none exist
- [08-03]: RoomManager/EgressManager interfaces with nil=disabled pattern for graceful LiveKit-off mode
- [08-03]: DSGVO consent checked at StartRecording time (all must respond before Egress begins)
- [08-03]: 30-day retention on every recording via RetentionExpiresAt
- [08-03]: Phase 11 integration via ListRecordingsWithAccess (participant-only access via JOIN)
- [08-03]: Call auto-ends when last participant leaves (HandleParticipantLeft webhook)
- [08-04]: Lazy away detection on read instead of background worker for simplicity
- [08-04]: Config cache with 60s refresh avoids DB hit on every heartbeat/presence check
- [08-04]: Heartbeat respects manual DND/away and InCall - does not override
- [08-04]: Notes saveable during in_progress AND completed meetings (post-meeting notes)
- [08-04]: ConvertActionItemsToTasks returns unconverted items for caller to create tasks
- [08-05]: VideoRoutes shares gRPC connection with WorkRoutes via ServiceName "work" (same binary)
- [08-05]: Reaction HTTP endpoints return 501; reactions handled via WebSocket events only
- [08-05]: WSPresenceService/WSVideoService interfaces injected post-construction to avoid circular imports
- [08-05]: LiveKit SDK types in github.com/livekit/protocol/livekit, not in server-sdk-go/v2 package
- [08-06]: video-client.ts mirrors calendar-client.ts fetch wrapper pattern (not openapi-fetch)
- [08-06]: Video store ephemeral (no persist); presence store persists only myStatus
- [08-06]: 33 hooks across 4 files (10 video, 15 meetings, 5 presence, 3 reactions)
- [08-06]: Presence queries use 10s staleTime for near-real-time updates
- [08-06]: Reaction toggle uses optimistic update via setQueryData
- [08-07]: LiveKitRoom with GridLayout (gallery) + FocusLayout (speaker) view modes
- [08-07]: Electron desktopCapturer for screen/window sharing via navigator.mediaDevices override
- [08-07]: RecordingConsentDialog with DSGVO-compliant blur/mute for declined participants
- [08-07]: FloatingCallBar fixed bottom-right z-50, IncomingCallOverlay fullscreen z-[100]
- [08-08]: Darien's meeting components cherry-picked from design/brainstorm + wired to API hooks
- [08-08]: MeetingNotesPanel with 30s debounced auto-save and visual indicator
- [08-08]: MeetingActionItems batch conversion to tasks with project picker
- [08-08]: MeetingLobby shows "Letzte Notizen" for recurring meetings
- [08-09]: frimousse emoji picker per-message via Radix Popover (no global singleton)
- [08-09]: 5-color presence: green (online), yellow (away), red (DND), purple (in_call), gray (offline)
- [08-09]: PresenceProvider sends heartbeat every 30s, handles visibility-based away detection
- [08-09]: Call-from-chat button in ChannelHeader for DM and group calls
- [09-01]: Separate security.v1.SecurityService proto (audit/vault/GDPR/password/IP) while 2FA/sessions stay in auth.v1
- [09-01]: VaultSecret never exposes encrypted_value -- Get returns decrypted_value only
- [09-01]: BIGSERIAL sequence_num for audit hash chain ordering (more reliable than timestamp)
- [09-01]: tools/security_deps.go retains otp+validator in go.mod before service code exists
- [09-01]: gdpr_erasure_log.original_user_id has no FK (user row gets anonymized)
- [09-07]: react-intl over i18next for native ICU message format (no plugin needed)
- [09-07]: Static imports for all 4 locale bundles (small JSON, no async loading complexity)
- [09-07]: Zustand persist for locale store (consistent with existing store patterns)
- [09-07]: Fallback chain: user choice -> navigator.language -> DE default
- [09-07]: MISSING_TRANSLATION errors suppressed in dev mode only
- [09-02]: HKDF-SHA256 with nil salt and context-string for vault/TOTP key separation
- [09-02]: AES-256-GCM nonce prepended to ciphertext, base64 encoded for storage
- [09-02]: Vault dual-key derivation from single master secret (min 32 chars)
- [09-02]: Password history uses bcrypt.CompareHashAndPassword for reuse checking
- [09-02]: go-password-validator GetEntropy for custom threshold entropy checking
- [09-05]: Handler registry pattern for modular per-service GDPR export/erasure operations
- [09-05]: Continue-on-failure erasure: partial erasure across modules better than aborting
- [09-05]: Audit logs retained per DSGVO Art. 17(3)(e) -- AuditErasureHandler is no-op
- [09-05]: 7-day download expiration on export ZIP files
- [09-05]: Anonymized label "Geloeschter Benutzer #NNN" via sequential counter from erasure log
- [09-03]: Advisory lock ID 8675309 for serializing audit log writes
- [09-03]: Audit LogEvent never returns error to caller (fire-and-forget, logs internally)
- [09-03]: CSV export includes UTF-8 BOM for Excel compatibility
- [09-03]: User-agent parser detects Electron, major browsers, OS for session device metadata
- [09-04]: VaultEncryptor interface for at-rest TOTP encryption (nil = dev fallback)
- [09-04]: Login returns LoginResult instead of (User, TokenPair) for 2FA pending flow
- [09-04]: PendingToken is 5-min JWT with type=2fa_pending claim
- [09-04]: Recovery codes: 8 codes, 10 hex chars, SHA-256 hashed at rest
- [09-04]: 2FA enforcement grace period calculated from user.CreatedAt
- [09-04]: AdminReset2FA requires non-empty reason for audit trail
- [09-06]: SecurityService runs in same binary as AuthService (shared gRPC port)
- [09-06]: VaultAdapter bridges vault.Service (no ctx) to auth.VaultEncryptor (with ctx)
- [09-06]: IP filter applied globally before rate limiter, fail-open when rules unavailable
- [09-06]: IP rules use direct pgx in security_grpc.go (no separate service layer)
- [09-06]: 2FA validate is public (pending_token), all other 2FA endpoints require JWT auth
- [09-08]: Auth store login throws '2FA_REQUIRED' error for LoginPage state machine transition
- [09-08]: Vault secret reveal uses mutation (not query) to avoid caching decrypted values in TanStack Query
- [09-08]: 30-second auto-hide timer for revealed vault secrets using setInterval cleanup
- [09-08]: Security hooks placed in api/hooks/ directory (existing pattern) not hooks/ root
- [09-08]: LoginPage 2FA flow uses auth store complete2FALogin instead of direct API hook
- [09-09]: Settings sidebar navigation pattern adapted from design/brainstorm branch
- [09-09]: /settings for all users, /admin/security/* prefix for admin-only security pages
- [09-09]: Settings link visible to all users; separate admin-only Shield icon for security
- [09-09]: GDPR erasure two-step confirmation with admin password input
- [09-09]: Hook imports corrected to @/api/hooks/ paths matching 09-08 output

- [10-01]: Selective file checkout from design/brainstorm (not full merge) to preserve main's backend connectivity
- [10-01]: 5-layer desk theme system with OKLCH color tokens in globals.css
- [10-01]: New sidebar (collapsible, grouped nav) replaces old Sidebar.tsx
- [10-01]: Header widgets: Clock, SearchBar, DailyPlanner, TimeTracker, LanguageSwitcher, ProfileMenu
- [10-01]: DeskEnvironment wraps AppShell for themed workspace
- [10-02]: 25 new module pages with Zustand mock stores (ready for TanStack Query migration when backend built)
- [10-02]: App.tsx: 25+ lazy imports + routes added, I18nProvider/VideoPage/security routes preserved
- [10-02]: SettingsPage merged: 8 tabs (general, security, language, privacy, calendar, finance, mail, team)
- [10-02]: 77 desk PNG assets across 6 themes (cozy, dreamy, industrial, nature, raumstation, minimal)
- [10-03]: Umlaut normalization (ae/oe/ue/ss -> proper Unicode) across all existing modules
- [10-03]: KalenderPage: design's mock rewrite skipped, only umlaut fixes + missing constants added
- [10-03]: EmptyState prop mismatch fixed: design's actionLabel/onAction -> main's action:{label,onClick}
- [10-03]: ActionItem type narrowing: variant string -> 'default'|'destructive' union
- [10-03]: 12 industry-specific modules from design (not in current roadmap, candidate for Phase 20 Plugins)

- [10-04]: 39 RPCs in email.v1.EmailService covering accounts, folders, messages, send/compose, signatures, CRM linking, sync, attachments, import/export
- [10-04]: Email service on gRPC :50056, health/metrics on :9096 (following existing service port pattern)
- [10-04]: tools/email_deps.go in backend/tools/ to retain email Go deps before service code imports them
- [10-04]: tsvector search with German config on email_messages (consistent with CRM contact search)
- [10-04]: Contact visibility CHECK constraint (shared, personal) with owner_id FK to users

- [11-01]: Document service as separate binary (not shared with work service) due to OS-level docconv/poppler dependencies
- [11-01]: German tsvector config for document search (consistent with email/CRM search)
- [11-01]: Service ports :50057 (gRPC) and :9097 (health) following sequential port pattern
- [11-01]: 34 RPCs in DocumentService covering folders, files, shares, tags, entity links, search, WOPI

- [11-02]: File service reuses chatfile.FileStore interface (no new MinIO client)
- [11-02]: Version revert creates new version with old content (append-only version history)
- [11-02]: File move updates folder_id only, MinIO storage key unchanged (reference by file ID)
- [11-02]: DACH default folders: personal=Dokumente/Bilder/Vorlagen, team=Allgemein/Projekte/Vorlagen

- [11-04]: WOPI handler runs in gateway with gRPC-backed WOPIFileAdapter, not in document microservice
- [11-04]: Global search fans out to CRM and documents; email returns empty until SearchMessages RPC exists
- [11-04]: WOPI routes at root level (/wopi/files/) per WOPI spec, outside RouteRegistrar loop, no standard auth
- [11-04]: OnlyOffice runs with JWT_ENABLED=false in dev, production will use separate ONLYOFFICE_JWT_SECRET

- [11-06]: Global search types defined inline in useGlobalSearch.ts (API response shape differs from document-types.ts)
- [11-06]: 300ms debounce + 2-char minimum for global search to prevent excessive API calls
- [11-06]: OnlyOffice editor as full-screen fixed overlay (z-50) with WOPI iframe URL construction
- [11-06]: File editability via both MIME-type and extension-based helpers for flexibility

- [11-03]: FileRepo interface in search package avoids circular import with file package (gRPC server bridges them)
- [11-03]: Virtual folder ListAll uses UNION ALL with per-source delegation for filtered requests
- [11-03]: Extractor returns empty string on error rather than failing (search gracefully degrades)

- [12-01]: 34 RPCs in FinanceService covering quotes, invoices, credit notes, payments, dunning, dashboard, DATEV export
- [12-01]: Biz service on gRPC :50058, health/metrics on :9098 (following sequential port pattern)
- [12-01]: All monetary values as string in proto (no native decimal), decimal.Decimal in Go models
- [12-01]: JSONB for line_items, tax_breakdown, snapshot_data, company_snapshot (document flexibility)
- [12-01]: Per-line rounding to 2dp in tax calculator prevents cent discrepancies
- [12-01]: TaxByRate keys use truncated rate strings (e.g., "19" not "19.00") for clean aggregation
- [12-01]: maroto/v2 added via tools/biz_deps.go for PDF generation in subsequent plans

- [12-02]: DealValueUpdater interface for decoupled CRM deal sync (nil-safe, graceful degradation on failure)
- [12-02]: Shared NumberSequenceRepo (SELECT FOR UPDATE) for gap-free numbering across document types
- [12-02]: GoBD immutability enforced at service layer: ErrInvoiceImmutable for any non-draft modification
- [12-02]: Invoice Send() builds complete JSONB snapshot (customer, company, line items, tax, metadata)
- [12-02]: QuoteReader interface in invoice package avoids circular dependency with quote package
- [12-02]: CompanySettings fallback chain: explicit input > company_settings table > hardcoded 30-day default

- [12-03]: InvoiceReader/InvoiceStatusUpdater as separate interfaces for cross-service payment dependencies
- [12-03]: ConfigRepository with upsert and lazy default creation for dunning config (14/14/14, 0/5/10 EUR)
- [12-03]: Dashboard forecast: avg monthly revenue * remaining months (simple, no ML for v1)
- [12-03]: PDF uses maroto/v2 with registered footer for Pflichtangaben on every page
- [12-03]: Dunning tone escalation: Zahlungserinnerung -> 1. Mahnung -> 2. Mahnung (threatens Inkasso)

- [12-04]: DealValueUpdater uses InexactFloat64() for CRM proto compatibility (Value is *float64)
- [12-04]: Tenant ID passed through gateway as user ID (single-tenant mode, multi-tenant via JWT claims later)
- [12-04]: PDF endpoints return 501 from gateway; biz gRPC service serves PDFs directly
- [12-04]: CreateDunning/EscalateDunning RPCs map to DetectAndCreateDunnings service method
- [12-04]: PostgresCompanySettingsRepo.Upsert added (INSERT ON CONFLICT DO UPDATE) for settings CRUD

- [12-05]: Finance store (Zustand) holds only UI state; all server data via TanStack Query hooks
- [12-05]: formatEUR centralized in stores/finance.ts for consistent EUR/de-DE formatting
- [12-05]: requestBlob helper for PDF/CSV binary downloads via URL.createObjectURL
- [12-05]: ExpenseFormDialog replaced with null stub (expenses not in Phase 12 scope)
- [12-05]: DunningConfigDialog inline within DunningPanel (no separate file)
- [12-05]: Finance query keys: ['finance', domain, ...params] for granular cache invalidation
- [12-06]: Fresh pdf.Generator per request with latest company settings from DB (not reusing startup instance)
- [12-06]: respondPDF gateway helper consolidates Content-Type/Disposition/Length for PDF binary streaming
- [12-06]: Dunning PDF filename varies by level: Zahlungserinnerung (level 1), 1_Mahnung (2), 2_Mahnung (3)
- [12-07]: Cross-service gRPC: BizRoutes.getCRMClient() enables CRM data enrichment in finance gateway
- [12-07]: Customer name prefers company over contact (B2B DACH invoicing norm)
- [12-07]: Contact/company fetch errors handled gracefully (quote created with partial data)
- [12-07]: Tax mode defaults to standard 19% MWSt, user adjusts in quote form

- [13-01]: 29 RPCs in single HRService proto (leave, time, absences, employees, settings)
- [13-01]: System leave types and doc categories seeded with zero UUID tenant_id for per-tenant copy pattern
- [13-01]: Partial unique index on active shifts ensures single active shift per employee at DB level
- [13-01]: Pure compliance functions use shopspring/decimal throughout for half-day precision
- [13-01]: BUrlG carryover expires after March 31 (inclusive) with CarryoverExpired flag
- [13-01]: ArbZG severity: 8h=Info, 9h=Warning, >10h=Error matching "warns at 8h, warns harder at 9h" requirement
- [13-02]: Leave service EmployeeRepository interface for cross-package manager lookup (avoids circular import)
- [13-02]: Overlap detection warns but allows approval (overlaps returned in ApproveResult for gRPC to surface)
- [13-02]: HR fallback: when no manager assigned, service allows approval; gRPC layer enforces HR role
- [13-02]: Leave balance auto-created on first access using BUrlG compliance engine with previous year carryover
- [13-02]: Employee self-service uses hasRestrictedFields() check for cleaner role-based field restrictions
- [13-02]: Absence calendar masks leave types to "Abwesend" with neutral gray (#9ca3af) when ShowAbsenceReason false
- [13-03]: HR services registered on same gRPC server as finance (biz binary), sharing port :50058
- [13-03]: GetWorkTimeStatus composed in gateway from GetActiveShift + GetDailySummary RPCs (no dedicated proto RPC)
- [13-03]: ArbZG severity at exactly 600 min returns "warning" not "error" (CheckWorkTime uses > 600 for error)
- [13-03]: HRRoutes ServiceName="biz" reuses existing gateway connection to biz gRPC server
- [13-04]: requestAnimationFrame for ClockInButton live timer (matching Phase 6 pattern, smoother than setInterval)
- [13-04]: Payroll/training tabs kept on Zustand mock stores (payroll = anti-feature, training not in Phase 13 scope)
- [13-04]: MemberDetailPanel changed to memberId-based API lookup instead of full TeamMember prop
- [13-04]: AbsenceCalendar changed to self-fetching (no props) instead of parent-provided data
- [13-04]: Deutschland-First locale: EUR formatting, de-DE date locale throughout all HR pages
- [13-04]: 30s polling for WorkTimeStatus in header ClockInButton for near-real-time display

- [14-01]: HR timetracking event emitter in biz/hr/timetracking/ (plan said timeentry/ which doesn't exist)
- [14-01]: Document shared event emitted on LinkToEntity (entity linking = sharing semantics)
- [14-01]: 27 RPCs in InboxService proto (14 messages, 8 team inboxes, 5 routing rules)
- [14-01]: Condition/Action JSON tree model designed for Phase 16 Automation reuse
- [14-02]: Empty AND=true (vacuous truth), empty OR=false -- standard logic for condition tree evaluator
- [14-02]: Routing rule cache stores all active rules, filters by channel at read time (simpler invalidation)
- [14-02]: Auto-reply failure is non-fatal in routing actions (logs warning, continues processing)
- [14-02]: GetBySourceID returns nil (not error) for missing entries to simplify dedup flow in message Create
- [14-03]: InboxRoutes ServiceName returns "notification" to reuse existing gRPC connection (co-hosted service)
- [14-03]: InboxConsumer uses messageRepo directly for NotifyDelivery instead of exposing repo through service
- [14-03]: Page token format is RFC3339Nano|UUID for cursor-based pagination
- [14-03]: Docker Compose unchanged -- inbox co-hosted in notification container requires no new service
- [14-04]: Channel-adaptive inline reply: email=textarea, chat=single-line, notification=no-reply
- [14-04]: Keyboard shortcuts j/k/e/s/r for GTD-style inbox triage without mouse
- [14-04]: 30s staleTime for messages, 15s for unread counts for near-real-time inbox
- [14-04]: Routing rules test panel embedded in editor dialog (not separate page)
- [14-04]: Channel badge colors: blue=email, green=chat, orange=notification (consistent across components)
- [14-04]: Optimistic triage mutations: mark read/star/archive update cache immediately, rollback on error

- [15-01]: User UUID as CalDAV username for v1 simplicity (avoids email resolution via auth gRPC)
- [15-01]: Bcrypt cost 12 for app-specific passwords (balance of security and validation speed)
- [15-01]: Sync token format "sync-token-{N}" for human-readable debugging
- [15-01]: caldav_settings key-value table for org-level feature toggles with upsert semantics
- [15-01]: go-webdav v0.7.0 and go-ical added via tools/caldav_deps.go build tag pattern
- [15-02]: CalDAVBackend queries event_exceptions directly from DB (no dedicated gRPC RPC)
- [15-02]: CardDAV two fixed address books per user: personal (Meine Kontakte) and company (Firmenkontakte)
- [15-02]: VTIMEZONE cache via sync.Map; DACH CET/CEST hardcoded for Europe/Berlin, Europe/Zurich, Europe/Vienna
- [15-02]: Compile-time interface compliance checks via var _ caldav.Backend = (*CalDAVBackend)(nil)
- [15-02]: Sync collection ID for address books via uuid.NewSHA1(userID, bookType) for deterministic tracking
- [15-03]: CalDAVPasswordService interface in gateway breaks import cycle (caldav->gateway->caldav)
- [15-03]: Adapter pattern in main.go bridges AppPasswordService to CalDAVPasswordService interface
- [15-03]: Variadic PushNotifier parameter on NewSyncTokenService for backward-compatible injection
- [15-03]: Push notifications fire-and-forget in goroutines, never blocking CalDAV writes
- [15-03]: Auto-unsubscribe on 410 Gone from push endpoints per WebDAV-Push draft spec
- [15-03]: Pool() accessor on AppPasswordService for direct DB queries in route handlers

- [16-01]: expr-lang/expr v1.17.8 for dual condition evaluation (simple AND/OR tree + expression mode)
- [16-01]: sync.Map cache for compiled expr programs (concurrent safe, no TTL for immutable programs)
- [16-01]: Automation service as standalone binary on :50059 (gRPC) and :9099 (health/metrics)
- [16-01]: ExprEnv typed environments per module for compile-time field validation in expressions
- [16-01]: Dotted field path resolution in simple mode conditions (e.g., deal.value)
- [16-02]: Function-reference adapter pattern to avoid workflow->trigger->workflow import cycle
- [16-02]: Notification action standalone (slog-based) since notification service lacks CreateNotification RPC
- [16-02]: Calendar action reuses work gRPC connection (co-hosted services)
- [16-02]: Semaphore of 20 concurrent executions + circuit breaker at 100/hour for resource protection
- [16-02]: 30s TTL cache on TriggerMatcher for active automations
- [16-02]: Loop prevention: events with module_id "automation" skipped by consumer
- [16-03]: Single Zustand draft workflow as source of truth for both wizard and react-flow editor
- [16-03]: @xyflow/react ^12.0.0 for visual node editor
- [16-03]: Recursive AND/OR condition tree with arbitrarily nested groups in simple mode
- [16-03]: Template variable {{key}} insertion in action parameter inputs
- [16-03]: Module-grouped trigger/action catalogs with search filtering
- [16-03]: Optimistic enable/disable mutations in TanStack Query hooks

- [17-01]: msbotbuilder-go/core sub-package import (root package has no Go files)
- [17-01]: Upsert semantics on CreateAccountLink (ON CONFLICT DO UPDATE) per research pitfall #4
- [17-01]: JSONB @> operator for module-level channel mapping filtering
- [17-01]: Credentials vault key reference in config, never exposed in proto responses
- [17-02]: PlatformPoster interface decouples forwarder from concrete Teams/Slack clients
- [17-02]: Proto codegen regenerated (was missing from 17-01, blocking server/gateway compilation)
- [17-02]: WithIntegration functional option pattern for backward-compatible gRPC server extension
- [17-02]: Nil-safe platform initialization: missing env vars = platform disabled, not crash
- [17-02]: Inbound webhook routes bypass JWT auth but verify platform-specific signatures
- [17.5-01]: Message.CreatedBy changed to *uuid.UUID -- nullable for guest messages, CHECK constraint enforces exactly one sender
- [17.5-01]: Guest token: UUID v4 stored as SHA-256 hash (64 hex chars), plain token returned once on creation
- [17.5-01]: In-memory sliding window rate limiter (30 msgs/min), resets on restart
- [17.5-01]: Guest channel config defaults: 7-day expiry, 10MB file limit, image+PDF, blue primary color, German welcome message
- [17.5-02]: GuestSessionValidator interface in server package for circular import avoidance
- [17.5-02]: uuid.Nil as guest sentinel in GetMessages gRPC, server checks is_guest_enabled
- [17.5-02]: LEFT JOIN users + LEFT JOIN guest_sessions for mixed user/guest message listing
- [17.5-02]: SkipMembershipCheck flag on ListInput for guest-enabled channel access
- [17.5-02]: Guest routes public (no JWT) with X-Guest-Token header auth middleware
- [17.5-03]: Standalone Vite SPA (no Tailwind/TanStack/Zustand) for minimal bundle size (~66KB gzipped)
- [17.5-03]: CSS custom properties for theming, primary color overridden by channel config
- [17.5-03]: Gateway serves SPA with /guest/assets/* for static files and /guest/* SPA fallback
- [17.5-03]: Graceful degradation: if guest-chat/dist/ doesn't exist, guest chat is simply disabled
- [17.5-03]: useRef<T | null>(null) pattern for React 19 strict mode compatibility

- [18-01]: VaultService interface in types.go matches vault.Service signature (ctx + createdBy uuid) for direct implementation
- [18-01]: Token cache uses 30s safety margin before expiry to prevent edge-case token usage during refresh
- [18-01]: Rate limiter adaptive: starts with defaults (50/10s), adjusts from X-RateLimit-* response headers
- [18-01]: Client.do() handles 429 with exponential backoff (1s, 2s, 4s) and 401 with single token refresh retry
- [18-01]: Bexio API quirks: POST for updates (not PATCH), Content-Length:0 for GET, salutation_id 0 = none
- [18-01]: GetFieldMappings returns nil (not error) when no mapping exists
- [18-01]: UpdateLastSyncTime uses column name switch (not dynamic SQL) for safety
- [18-02]: ContactService/InvoiceReader/QuoteReader interfaces in bexio package to avoid circular imports
- [18-02]: ContactSyncData intermediate struct decouples Bexio API from CRM service
- [18-02]: Last-write-wins uses bexio_updated_at vs kmuhub_updated_at from entity mapping
- [18-02]: IntegrationConfigRepo interface for shared integration_configs table (same as Teams/Slack)
- [18-02]: Scheduler per-tenant goroutines with context cancellation for graceful shutdown
- [18-02]: Payment poller skips already-paid invoices for efficiency
- [18-02]: noopEmitter default for EventEmitter (same pattern as invoice/quote services)
- [18-03]: BexioGRPCServer in internal/server/ (not internal/biz/server/) following existing BizGRPCServer pattern
- [18-03]: BexioRoutes ServiceName returns "biz" to reuse existing gRPC connection (co-hosted service)
- [18-03]: OAuth callback route public (no auth middleware), all admin routes use RequireRole("admin")
- [18-03]: Bexio service optional: only initialized when BEXIO_CLIENT_ID env var is set
- [18-03]: PostgresIntegrationConfigRepo in bexio package (same table as notification, avoids import cycle)
- [18-03]: Vault initialized per biz binary for OAuth token storage when VAULT_MASTER_SECRET set
- [18-03]: Bexio scheduler shutdown before gRPC graceful stop in shutdown sequence
- [18-04]: bexio-client.ts follows integration-client.ts fetch wrapper pattern (typed fetch + auth + 401 retry)
- [18-04]: Separate useBexio.ts hooks (not merged into useIntegration.ts) for clean separation
- [18-04]: BexioSetupWizard 4 steps: OAuth → Sync Config → Field Mapping → Initial Sync
- [18-04]: BexioSyncDashboard as Dialog (same pattern as Teams/Slack wizards)
- [18-04]: Field mapping editor compact prop for wizard-embedded vs standalone mode
- [18-04]: Components in modules/settings/integrations/ with re-exports in components/settings/

### Pending Todos

- CRUD action buttons are placeholder only -- need proper create/edit/delete dialogs in future plan

### Blockers/Concerns

- [Phase 12]: GoBD compliance requires Steuerberater consultation before data model design
- [Phase 13]: ArbZG/BUrlG implementation details need labor law expert review
- [Phase 12]: DATEV Buchungsstapel format spec not publicly detailed -- may need DATEV partner access

## Session Continuity

Last session: 2026-02-26
Stopped at: Phase 19 (DATEV API + Lexware Office) -- COMPLETE (committed 7eb74d9)
Resume file: N/A
Next: Phase 20 (Plugin System + Industry Templates)

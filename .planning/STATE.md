# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-08)

**Core value:** Every employee completes their entire workday without opening another program
**Current focus:** Phase 11 in progress (Documents & Files + WOPI/OnlyOffice)
**Recent strategy changes:** Phases 11-20 reordered, buchhaltung→finanzen rename, payroll anti-feature confirmed

## Current Position

Phase: 11 of 20 (Documents & Files + WOPI)
Plan: 2 of 6 in current phase (2 complete)
Status: In progress
Last activity: 2026-02-17 -- 11-02 document service core business logic complete

Progress: [███████████████████████████░] 88% (58/66 plans across phases 4-20)

## Performance Metrics

**Velocity:**
- Total plans completed: 57
- Average duration: ~7 minutes
- Total execution time: ~6h 13min

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
| 11 | 2/6 | ~10min | ~5min |

**Recent Trend:**
- Last 5 plans: 10-06 (~10min), 10-07 (~8min), 11-01 (~5min), 11-02 (~5min)
- Trend: Document service plans consistently fast (~5min each)

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

### Pending Todos

- CRUD action buttons are placeholder only -- need proper create/edit/delete dialogs in future plan

### Blockers/Concerns

- [Phase 12]: GoBD compliance requires Steuerberater consultation before data model design
- [Phase 13]: ArbZG/BUrlG implementation details need labor law expert review
- [Phase 12]: DATEV Buchungsstapel format spec not publicly detailed -- may need DATEV partner access

## Session Continuity

Last session: 2026-02-17
Stopped at: Completed 11-02-PLAN.md
Resume file: .planning/phases/11-documents-files-wopi-onlyoffice/11-02-SUMMARY.md
Next: Execute 11-03-PLAN.md (document gRPC server or additional services)

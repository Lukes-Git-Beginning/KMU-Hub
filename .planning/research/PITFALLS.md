# Domain Pitfalls

**Domain:** All-in-one workplace platform for DACH SMBs (scaling from 4 services to 12+)
**Researched:** 2026-02-07
**Confidence:** HIGH (based on codebase inspection, team context, prior project learnings)

---

## Critical Pitfalls (Project Killers)

### Pitfall 1: Gateway God Object — The Single Point of Coupling Death

**What goes wrong:** The API gateway (`cmd/gateway/main.go`) currently instantiates gRPC connections to auth, CRM, and chat. `GatewayHandler` holds all three clients. Adding 8 more modules means the gateway imports every proto, creates 10+ gRPC connections at startup, and `RegisterRoutes` becomes a 1000+ line monster. One failing service blocks gateway startup entirely (current pattern: `os.Exit(1)` on connection failure).

**Why it happens:** The current pattern works for 3 services. At 10+ services, the gateway becomes the coupling point for the entire system. Every new module requires gateway code changes, gateway redeployment, and gateway restart.

**Consequences:**
- Gateway startup becomes slow and fragile (10+ gRPC dials)
- Any service failure at startup kills the gateway (no partial availability)
- Gateway binary grows with every proto import
- Every module change requires gateway redeployment
- Testing the gateway requires mocking 10+ services
- Docker-compose `depends_on` chain becomes a startup nightmare (currently 6 dependencies)

**Warning signs:**
- Gateway main.go exceeds 300 lines
- Gateway handler file exceeds 500 lines
- Startup takes more than 10 seconds
- One offline service prevents all API access

**Prevention:**
- **Phase 4 (before adding module 5+):** Refactor gateway to lazy-connect gRPC clients. Use `grpc.NewClient` with blocking disabled (already partially the case since `grpc.NewClient` is lazy by default, but the error handling pattern suggests the intent is to fail fast). Remove `os.Exit(1)` on connection failure; instead, return 503 for routes whose backing service is unavailable.
- **Split `GatewayHandler` into per-module handlers** registered via a plugin-style pattern: each module registers its own routes on the chi router via an interface like `RouteRegistrar`. The gateway main.go stays thin.
- **Consider moving to a config-driven routing layer** where proto-to-HTTP mapping is declarative rather than hand-coded in `http.go`.

**Detection:** Monitor gateway binary size, startup time, and the number of imports in `cmd/gateway/main.go`.

**Severity:** CRITICAL. If not addressed before module 5, every new module addition becomes exponentially harder.

---

### Pitfall 2: Shared Database, Hidden Coupling — The Distributed Monolith

**What goes wrong:** All 4 current services share one PostgreSQL database (`kmuhub`). All 19 migrations run in sequence. When you add HR (employee records, time tracking, leave), Finance (invoices, payments, tax), and Calendar (events, recurring rules), you get 50+ tables in one database. Service A modifies a table that Service B reads. Migrations become ordering nightmares. A bad migration in the finance service takes down CRM.

**Why it happens:** Shared database is the pragmatic choice for a solo dev. But "shared database" without "shared schema ownership rules" silently creates coupling. The CRM service already accesses `users` (owned by auth) via foreign keys. This gets worse with every module.

**Consequences:**
- Migration conflicts (two modules add migrations with overlapping sequence numbers)
- Cascading failures (bad migration in one module corrupts shared tables)
- Schema coupling (HR needs employee records, which are really user records, which auth owns)
- Cannot deploy modules independently (shared migration must run first)
- Self-hosted customers cannot selectively disable modules (tables are all in one DB)

**Warning signs:**
- Migration numbering conflicts during development
- Two services writing to the same table
- Foreign keys crossing service boundaries (e.g., `deals.owner_id` -> `users.id`)
- Rollback of one migration breaks another service's tables

**Prevention:**
- **Keep shared database** (splitting databases for a solo dev is premature optimization) BUT establish strict ownership rules:
  - Each service owns specific tables (documented in a SCHEMA_OWNERS.md)
  - Cross-service references use only the owning service's API, NOT direct JOINs (exception: user_id foreign keys are acceptable as they are stable)
  - Migration files are prefixed by module: `000020_chat_xxx`, `000021_hr_xxx`
- **Add a migration CI check** that verifies no two pending migrations have the same sequence number
- **Test migrations in both directions** (up AND down) in CI
- **Never share mutable state across services through the database** — if HR needs to update a user's department, it calls auth service, not writes to users table directly

**Detection:** Grep for cross-service table access in repository files. Any repository outside `internal/auth/` accessing the `users` table directly is a red flag.

**Severity:** CRITICAL. Causes data corruption and cascading failures that are extremely hard to debug.

---

### Pitfall 3: DACH Regulatory Compliance as Afterthought

**What goes wrong:** Building HR and Finance modules without upfront regulatory research leads to fundamental data model mistakes that require complete rewrites. German labor law (Arbeitszeitgesetz, Bundesurlaubsgesetz, Entgeltfortzahlungsgesetz), Austrian/Swiss variants, GDPR Article 17 (right to deletion vs. retention requirements), and German tax law (GoBD, AO) impose contradictory requirements that cannot be bolted on later.

**Why it happens:** Developers build the "obvious" features first (employee CRUD, invoice generation) and discover regulatory requirements late. By then, the data model is wrong.

**Consequences — HR module:**
- **Arbeitszeitgesetz (ArbZG):** Maximum 8h/day (10h with compensation), mandatory 11h rest between shifts, mandatory break after 6h. If your time tracking model does not enforce these FROM DAY ONE, data retroactively becomes non-compliant. Employer is liable, not the software.
- **Bundesurlaubsgesetz (BUrlG):** Minimum 24 workdays (6-day week) or 20 workdays (5-day week). Carryover rules are complex — must be taken by March 31 of following year (federal labor court rulings have nuanced this). Part-time workers get proportional leave. If the leave model does not handle part-time, Minijob, and varying work-week patterns, it is useless for German SMBs.
- **Mutterschutzgesetz / Elternzeit:** Maternity/parental leave has strict employer obligations. The absence management system must handle protected periods.
- **Datenschutz (GDPR + BDSG):** Employee data is special category data. Purpose limitation means HR data stored for payroll cannot be used for performance analytics without separate consent. Deletion requirements conflict with retention requirements (7 years for tax, 3 years for legal claims).

**Consequences — Finance module:**
- **GoBD (Grundsaetze ordnungsmaessiger Buchfuehrung und DV):** Digital records must be immutable, traceable, and auditable. Invoice data CANNOT be updated — only corrected via credit notes (Gutschriften/Stornierungen). If the data model allows `UPDATE` on invoices, it violates GoBD.
- **UStG (Umsatzsteuergesetz):** VAT handling in DACH is complex — standard (19%/20%/8.1%), reduced (7%/10%/2.6%), exempt, reverse charge, intra-EU, third-country. Switzerland is not EU. The tax calculation engine must handle all of these from the start.
- **DATEV compatibility:** 80%+ of German SMB accountants use DATEV. If KMU Hub cannot export DATEV-compatible data (Buchungsstapel format, SKR03/SKR04 chart of accounts), the finance module is DOA for the DACH market.
- **Aufbewahrungspflicht:** Invoices must be retained 10 years (AO ss147), business correspondence 6 years. "Delete my data" (GDPR) collides with "keep records 10 years" (tax law). Must implement purpose-based retention with selective anonymization.

**Warning signs:**
- Building invoice CRUD with UPDATE capability
- Time tracking without rest period / max hours validation
- Leave management without part-time / Minijob handling
- No DATEV export discussed in requirements
- Employee deletion that removes records instead of anonymizing

**Prevention:**
- **Before writing a single line of HR code:** Research and document the exact regulatory requirements in a dedicated ADR. Key laws: ArbZG, BUrlG, MuSchG, BEEG, TzBfG, EntgFG, BDSG.
- **Before writing a single line of Finance code:** Research GoBD requirements. The invoice model MUST be append-only (no updates, only cancellations + corrections). Research DATEV Buchungsstapel format.
- **Implement a retention policy engine** early: data has a purpose, a retention period, and an anonymization strategy. This is NOT a feature to add later — it must be the foundation of the data model.
- **Hire a Steuerberater (tax advisor) for a 2-hour consultation** on GoBD compliance before building the finance data model. This is not optional.
- **Consider making HR and Finance "compliance-first" modules** — build the regulatory constraints first, then add the UX.

**Detection:** If any migration contains `UPDATE` triggers on invoice tables, or if time tracking has no max-hours validation, these are hard compliance failures.

**Severity:** CRITICAL. Non-compliance with GoBD can result in tax authority rejecting the entire bookkeeping. Non-compliance with ArbZG makes the employer liable. This is not "nice to have" — it is legal requirement.

---

### Pitfall 4: Scope Paralysis — 12 Modules for 1 Developer

**What goes wrong:** The project plans auth, CRM, chat, project management, calendar, email, video, HR, finance, automation, plugin system, and desktop/mobile apps. That is 12+ major modules. At the current pace (roughly 1 module per month with AI assistance), this is 12+ months of backend work alone, PLUS desktop, PLUS mobile. The pilot customer (Zentrum fuer finanzielle Aufklaerung) needs specific features NOW, not all 12 modules eventually.

**Why it happens:** "All-in-one" is the vision, and it is a good vision. But building all of it before any of it ships means the pilot customer waits, feedback loops do not exist, and motivation dies.

**Consequences:**
- Pilot customer churns before seeing value
- 12 months of building without user feedback leads to wrong assumptions
- Solo dev burnout (the most underestimated risk in the project)
- Technical debt compounds across 12 modules without users to validate priorities
- "Almost done" syndrome — always 2 months from launch

**Warning signs:**
- Phase plan shows all 12 modules before first customer deployment
- No module is "shippable" individually
- Pilot customer asks "when can I use it?" more than twice
- Developer works more than 10 hours/day for more than 2 weeks straight

**Prevention:**
- **Define the "Pilot MVP" explicitly:** Which 3-4 modules does Zentrum fuer finanzielle Aufklaerung actually need? Almost certainly: Auth + CRM + Chat + Calendar. NOT HR, NOT Finance (they probably have DATEV already).
- **Ship the Pilot MVP first, then iterate.** Real users finding real bugs is more valuable than building module 7.
- **Phase modules by customer demand, not by technical elegance.** If nobody asks for the automation engine, do not build it.
- **Set hard time-boxes per module** (2-3 weeks max). If a module is not shippable in 3 weeks, it is scoped too large. Split it.
- **Accept that some modules will be "good enough" not "perfect."** The first version of project management does not need Gantt charts.

**Detection:** If the roadmap has more than 6 modules before first customer deployment, it is too ambitious.

**Severity:** CRITICAL. This is the most likely cause of project failure — not technology, not architecture, but scope.

---

## High Pitfalls (Major Rework)

### Pitfall 5: Config Struct Explosion

**What goes wrong:** The current `Config` struct in `internal/config/config.go` has 22 fields for 4 services. Adding 8 more modules (each with their own ports, addresses, credentials) pushes this to 60+ fields. Every service loads the same Config struct even though it only needs 5-6 fields. Adding LiveKit config, email SMTP config, CalDAV config, DATEV config, etc. makes this unmanageable.

**Why it happens:** Single config struct is the simplest approach at 4 services. It does not scale.

**Warning signs:**
- Config struct exceeds 40 fields
- Services loading config fields they never use
- Environment variable naming conflicts
- Config validation becomes a function longer than the struct itself

**Prevention:**
- **Refactor to per-service config structs** before adding module 5. Each service has its own config package that embeds a shared `CommonConfig` (database, redis, JWT) and adds service-specific fields.
- **Use a config registry pattern:** `config.LoadAuth()`, `config.LoadCRM()`, `config.LoadCalendar()` etc.
- **Validate required vs. optional fields per service** — the calendar service should not fail because MINIO_ACCESS_KEY is missing.

**Severity:** HIGH. Not a project killer, but causes significant friction and weird bugs (service fails because unrelated env var is missing).

---

### Pitfall 6: Proto File and gRPC Sprawl

**What goes wrong:** Each module gets its own `.proto` file and gRPC service. Currently: `auth/v1/auth.proto`, `crm/v1/crm.proto`, `chat/v1/chat.proto`. CRM proto already has many RPCs covering custom fields, tags, contacts, companies, deals, pipeline stages, activities, search, saved filters, and reports — all in ONE proto file. Adding project management, calendar, HR, finance means either (a) mega-proto files with 50+ RPCs or (b) many small protos that proliferate.

**Why it happens:** Proto-per-service is natural but each service grows. The CRM proto will be enormous.

**Warning signs:**
- Single proto file exceeds 500 lines
- Proto compile time exceeds 10 seconds
- Difficulty finding RPCs in proto files
- Proto import cycles between services

**Prevention:**
- **Split CRM proto NOW** (before it grows further): `crm/v1/contacts.proto`, `crm/v1/deals.proto`, `crm/v1/reports.proto` etc.
- **Establish proto organization convention:** `proto/{domain}/v1/{entity}.proto`
- **Shared types in `proto/common/v1/`:** Pagination, timestamps, money types, error details
- **NEVER import across service protos** (no `crm.proto` importing `auth.proto`). Duplicate shared types via common protos instead.
- **Consider whether new modules even need gRPC.** Some modules (calendar, project management) might be better as packages within an existing service rather than separate gRPC services.

**Severity:** HIGH. Proto sprawl causes compile-time pain, code generation bloat, and makes the codebase hard to navigate.

---

### Pitfall 7: Cross-Module Feature Interactions Nobody Planned For

**What goes wrong:** Modules are built in isolation, then users expect them to work together. "I want to create a deal from a chat message." "I want to see calendar events on the project timeline." "When I close a deal, auto-create an invoice." "Link an email thread to a contact." These cross-module interactions are where 80% of the value lies in an "all-in-one" platform, but they are never in the initial module spec.

**Why it happens:** Each module is scoped independently. Cross-module features feel like "Phase 2" but users expect them immediately.

**Consequences:**
- Modules feel like separate apps stapled together, not an integrated platform
- Users compare unfavorably to Slack + Salesforce (which at least have integrations)
- Retrofitting cross-module interactions requires changing multiple services, migrations, and protos
- The "all-in-one" value proposition is undermined

**Warning signs:**
- Users say "I have to switch between modules all the time"
- No module references any other module's data
- The demo feels like 5 separate apps
- No entity linking or cross-module search

**Prevention:**
- **Design a "linking" system early:** A generic `entity_links` table that connects any entity to any other entity (contact <-> deal, message <-> task, email <-> contact). This is a foundation, not a feature.
- **Define the top 10 cross-module interactions before building** modules 5-8. Prioritize the ones the pilot customer would use.
- **Build a unified activity feed** that aggregates events across modules. This forces integration thinking early.
- **For each new module, ask: "What 3 existing entities does this connect to?"** If the answer is "none," the module is too isolated.

**Severity:** HIGH. The difference between "collection of tools" and "integrated platform" is cross-module interactions. Without them, KMU Hub has no advantage over best-of-breed tools.

---

### Pitfall 8: Self-Hosted Deployment Complexity at Scale

**What goes wrong:** The current docker-compose has 7 services (postgres, redis, minio, migrate, auth, crm, chat, gateway). Adding 8 more backend services means 15+ containers. Self-hosted customers (the EU-sovereignty selling point) need to run this on a single server or small cluster. Docker-compose with 15+ services is not "easy self-hosted deployment" — it is ops complexity that SMBs cannot handle.

**Why it happens:** Each microservice = one container. This is the natural extension of the current architecture. But it ignores the operational reality of DACH SMBs that do not have DevOps teams.

**Consequences:**
- Self-hosted customers need 32GB+ RAM just for the platform
- Container orchestration failures multiply with service count
- Customers call support because "the HR module is offline" (container crashed, nobody restarted it)
- Update process requires pulling and restarting 15+ containers
- Log management across 15+ containers is a nightmare without Grafana/Loki

**Warning signs:**
- Docker-compose file exceeds 300 lines
- Total container memory exceeds 8GB
- Self-hosted customer needs help restarting services weekly
- Self-hosted health check dashboard shows 3+ red services regularly

**Prevention:**
- **Consolidate services for self-hosted deployment.** Not every module needs its own container. Consider a "monolith binary" option: one Go binary that runs all services in-process (each as a goroutine), sharing one database connection pool. The SaaS deployment can still use separate containers.
- **Build the consolidated binary NOW** while there are only 4 services. It gets harder to retrofit.
- **Target: self-hosted = 4 containers max** (app, postgres, redis, minio). SaaS = separate containers for independent scaling.
- **Add a self-hosted admin dashboard** (simple web UI showing service health, disk usage, backup status).

**Severity:** HIGH. The self-hosted deployment model is a core USP. If it is too complex, SMBs will not adopt it, and the EU-sovereignty differentiator is lost.

---

### Pitfall 9: Email Integration is Deceptively Hard

**What goes wrong:** Email seems simple ("just IMAP/SMTP") but is actually one of the hardest modules to build correctly. Email threading (RFC 5322 References/In-Reply-To), MIME parsing (multipart, attachments, inline images, encoding chaos), HTML sanitization (email HTML is 1999-era table layouts with CSS inline), sync reliability (IMAP IDLE, connection drops, quota handling), and provider-specific quirks (Gmail, Outlook, Exchange, Postfix all behave differently) make this a 3-month module, not a 3-week module.

**Why it happens:** Email has been around since 1971. Everyone uses it. So it "should be simple." It is not.

**Consequences:**
- MIME parsing bugs cause data loss (attachments silently dropped, encoding mangled)
- Email threading breaks (replies appear as new threads)
- IMAP sync falls out of sync after connection drop, causing duplicate or missing emails
- HTML email rendering is a security nightmare (XSS via email content)
- OAuth2 for Gmail/Outlook is a separate authentication flow to implement and maintain
- Provider rate limits cause sync delays that users blame on KMU Hub

**Warning signs:**
- Estimating email module at less than 4 weeks
- Not planning for MIME multipart parsing edge cases
- No HTML sanitization strategy for email content display
- Testing only with Gmail (works) and assuming Outlook works the same (it does not)

**Prevention:**
- **Use a well-tested Go email library** for MIME parsing (do NOT hand-roll). Evaluate `emersion/go-imap`, `emersion/go-message`, `emersion/go-smtp`.
- **Scope email integration aggressively for V1:** Send-only via SMTP + link to external inbox. Full two-way sync is a Phase 2 feature.
- **If building full sync:** Plan for eventual consistency, idempotent sync operations, and a message-ID based dedup system.
- **HTML email display:** Render in a sandboxed iframe with CSP restrictions. Never render raw HTML in the Electron app context.
- **Consider deferring email entirely** to post-pilot. The pilot customer almost certainly already has email. What they need is CRM + email linking (BCC to KMU Hub, auto-associate with contact), not a full email client.

**Severity:** HIGH. Email is the module most likely to be underestimated by a factor of 3x.

---

### Pitfall 10: Calendar Recurrence Rules (RFC 5545 RRULE)

**What goes wrong:** Recurring events seem simple until you implement them. RFC 5545 RRULE supports "every third Wednesday of every other month, except holidays, until December 2027, with timezone-aware DST transitions." PostgreSQL cannot natively expand RRULEs into occurrences. Every calendar implementation must choose between: (a) store expanded occurrences (wastes space, hard to modify series), (b) compute on the fly (complex, slow for "show me all events in March"), or (c) hybrid (complex but correct).

**Why it happens:** Developers implement "repeat daily" and "repeat weekly" and think they are done. Real users need complex recurrence patterns.

**Consequences:**
- "Edit this and all future events" requires splitting a recurrence series — complex data model
- Timezone handling with DST transitions causes events to shift by 1 hour
- Performance degrades when computing occurrences for a month view with many recurring series
- Exception dates (cancelled single occurrence) add another layer of complexity
- CalDAV compatibility requires full RRULE support

**Warning signs:**
- Calendar data model has no recurrence-related columns
- Tests only cover "daily" and "weekly" recurrence
- No timezone-awareness in event storage (storing local time without offset)
- No plan for "edit this occurrence only" vs. "edit all future"

**Prevention:**
- **Store events in UTC with explicit timezone** (IANA timezone name, not offset). Use Go's `time.LoadLocation`.
- **Use the hybrid approach:** Store the RRULE rule in the event, AND pre-expand occurrences for the next N months into an `event_occurrences` table. Re-expand on rule change.
- **Use an RRULE library** — do NOT implement RFC 5545 yourself. Evaluate Go RRULE libraries. If none are mature, consider porting a well-tested JS library (rrule.js).
- **Start with CalDAV compatibility in mind** even if CalDAV sync is a later feature. The data model must support VEVENT/VTODO mapping.

**Severity:** HIGH. Wrong data model means rewrite. Get it right first.

---

## Medium Pitfalls (Significant Delay)

### Pitfall 11: Test Infrastructure Does Not Scale to 10+ Services

**What goes wrong:** Currently, each service has unit tests with mocks. CI runs tests against real Postgres + Redis. As services multiply, CI time grows linearly. E2E tests require starting 10+ services — flaky, slow, and resource-intensive. Test data setup becomes complex (creating a deal requires auth user + CRM contact + pipeline stage).

**Why it happens:** Test infrastructure that works for 4 services does not scale to 12.

**Warning signs:**
- CI pipeline exceeds 15 minutes
- E2E tests are flaky (>5% failure rate)
- Test helper functions grow to hundreds of lines
- Developers skip running tests locally because it is too slow

**Prevention:**
- **Invest in a test data factory** (similar to FactoryBot/Faker): `testutil.CreateUser()`, `testutil.CreateDeal()`, `testutil.CreateChannel()`. Each factory handles its own prerequisites.
- **Use `testcontainers-go`** for integration tests (already mentioned in LEARNINGS.md but not yet implemented). Each service's integration tests spin up their own isolated database.
- **Parallelize CI:** Run each service's tests in a separate CI job. Use a matrix strategy.
- **Keep E2E tests minimal:** Test critical cross-service flows only (auth -> create contact -> send message). Resist the urge to E2E test everything.

**Severity:** MEDIUM. Slow tests lead to skipped tests lead to bugs.

---

### Pitfall 12: Notification Fatigue Across Modules

**What goes wrong:** Each module generates notifications independently. Chat message -> notification. CRM deal update -> notification. Calendar reminder -> notification. HR leave approval -> notification. Finance invoice due -> notification. Users get 50+ notifications per hour and disable all of them.

**Why it happens:** Each module thinks its notifications are important. Nobody designs the cross-module notification strategy.

**Warning signs:**
- Each module has its own notification system
- No unified notification preferences
- No notification batching / digest options
- Users ask how to turn off notifications

**Prevention:**
- **Build a centralized Notification Service** (not per-module notification code). All modules emit events; the notification service decides what to surface, when, and how.
- **Implement notification channels:** in-app, desktop push, email digest. Let users configure per-channel, per-module.
- **Batch notifications:** "3 new messages in #general" not 3 separate notifications.
- **Smart defaults:** Do not notify on events the user triggered themselves.
- **Build the notification service BEFORE building more modules** (Sprint 5 of Chat is the right time, as Chat notifications is the use case that forces the design).

**Severity:** MEDIUM. Does not break anything but severely degrades user experience.

---

### Pitfall 13: DATEV Integration is a Business Gatekeeper

**What goes wrong:** The DATEV ecosystem is the dominant accounting software platform in the DACH market. Over 80% of German tax advisors (Steuerberater) use DATEV. If KMU Hub's finance module cannot export data in a format the customer's Steuerberater can import into DATEV, the finance module is useless for the target market.

**Why it happens:** Developers build "generic" invoicing and assume export formats are a later feature. But DATEV compatibility is not a feature — it is table stakes for the DACH market.

**Warning signs:**
- Finance module design discussions never mention DATEV
- Invoice model does not include Kontenrahmen (SKR03/SKR04) account numbers
- No research on DATEV Buchungsstapel CSV format
- No plan for DATEV Unternehmen Online API access (requires DATEV partnership)

**Prevention:**
- **Research DATEV export format BEFORE building the finance data model.** The data model must accommodate DATEV's expectations: Belegfeld 1/2, Kontonummer, Gegenkonto, Steuerschluessel, Buchungstext.
- **Start with CSV export** in DATEV Buchungsstapel format. This covers 90% of use cases (Steuerberater imports CSV into DATEV).
- **Do NOT attempt full DATEV API integration for V1.** The DATEV Unternehmen Online API requires partnership application and certification. This is a 6+ month process.
- **Use SKR03 as default chart of accounts** (most common in Germany), with SKR04 as alternative.
- **Consider partnering with a DATEV-experienced freelancer** for the finance module data model review.

**Severity:** MEDIUM (because the finance module itself is likely post-pilot, but critical once it is built).

---

### Pitfall 14: Automation Engine Premature Abstraction

**What goes wrong:** Building a "workflow automation engine" (if/then rules, triggers, actions) is essentially building a domain-specific programming language. Developers either (a) build it too simple (only supports fixed trigger->action pairs, users hit limits immediately) or (b) build it too complex (visual programming environment, Turing-complete, 6 months of work before anyone uses it).

**Why it happens:** The vision of "customizable workflows" is exciting. The implementation is one of the hardest things in software engineering.

**Warning signs:**
- Automation engine is planned before any manual workflows exist
- Designing a generic "rule engine" without concrete use cases
- Using words like "any trigger" and "any action" in the spec
- No user research on what automations customers actually want

**Prevention:**
- **Do NOT build a generic automation engine initially.** Instead, build specific, hardcoded automations that solve real problems:
  - "When deal moves to Won, create invoice draft" (if finance exists)
  - "When new contact is created, assign to owner based on region"
  - "When message mentions @channel, send notification to all members"
- **Collect 20 concrete automation requests** from the pilot customer before designing the engine.
- **When building the engine:** Start with a simple trigger->condition->action model. Use a pipeline pattern, not a DAG (directed acyclic graph). Support only typed triggers (deal.stage.changed, contact.created) and typed actions (send.notification, create.task), not generic scripting.
- **Defer the visual builder.** API/config-based automation is sufficient for V1.

**Severity:** MEDIUM. Building it too early wastes months. Building it wrong means rebuilding it.

---

### Pitfall 15: Video (LiveKit) Resource Management for Self-Hosted

**What goes wrong:** LiveKit server requires significant resources for video processing (SFU, recording, egress). Self-hosted customers on a single server may not have enough CPU/RAM/bandwidth. Video quality degrades silently (pixelation, audio drops) rather than failing explicitly. Customers blame KMU Hub.

**Why it happens:** Video works perfectly in development (localhost, no network constraints). Production on a 4-core VPS with 8GB RAM and 50Mbps bandwidth is a different story.

**Warning signs:**
- No minimum hardware requirements documented for self-hosted
- Video tested only on local network
- No TURN/STUN server setup documented
- No bandwidth estimation or quality adaptation

**Prevention:**
- **Document minimum hardware requirements** explicitly: "Video calls require X CPU cores and Y GB RAM per concurrent call."
- **Implement quality detection:** If server resources are low, automatically reduce video resolution. Notify the admin.
- **Make video optional** in the self-hosted package. Not all SMBs need video.
- **TURN server setup is mandatory** for production — NAT traversal will fail without it. Include TURN setup in the self-hosted deployment guide.
- **Consider offering LiveKit Cloud as a managed option** for self-hosted customers who do not want to run their own video infrastructure.

**Severity:** MEDIUM. Video is a differentiator but not table stakes for the pilot.

---

### Pitfall 16: Multi-Tenancy Bolted On

**What goes wrong:** The current schema does not show explicit multi-tenancy (no `tenant_id` column, no row-level security). For SaaS deployment, every query needs to be scoped to a tenant. Adding `tenant_id` to 50+ tables after they are built is a massive migration and a guaranteed source of data leaks (forgetting the WHERE clause on one query exposes another tenant's data).

**Why it happens:** Single-tenant (self-hosted) is the initial focus. Multi-tenancy seems like a "SaaS-only" concern. But the project explicitly targets both SaaS and self-hosted.

**Warning signs:**
- No `tenant_id` or `organization_id` in the data model
- Queries do not include tenant scoping
- No row-level security in PostgreSQL
- SaaS launch is planned but multi-tenancy is not in any phase

**Prevention:**
- **Add `organization_id` to every tenant-scoped table NOW** (or in the next available migration window). For self-hosted, there is exactly one organization. For SaaS, this is the tenant boundary.
- **Use PostgreSQL Row Level Security (RLS)** as a safety net: even if application code forgets the WHERE clause, RLS prevents cross-tenant data access.
- **Set `organization_id` via a middleware context value** so repositories automatically scope queries.
- **This is far easier to do with 19 migrations than with 50.** Every week this is deferred makes it harder.

**Severity:** MEDIUM (for now — becomes CRITICAL at SaaS launch if not addressed).

---

## Low Pitfalls (Minor Friction)

### Pitfall 17: Inconsistent Error Handling Across Services

**What goes wrong:** Each service (auth, CRM, chat) has its own error types and error handling patterns. When the gateway translates gRPC errors to HTTP errors, the mapping is inconsistent. Users get different error response shapes from different endpoints.

**Prevention:**
- Establish a shared `proto/common/v1/errors.proto` with standard error details.
- Create a single `grpcToHTTP` error mapper in the gateway used by all handlers.
- Ensure every error response has the same JSON shape: `{"error": {"code": "...", "message": "...", "details": {}}}`.

**Severity:** LOW. Causes developer friction and inconsistent API experience.

---

### Pitfall 18: Electron App Memory Bloat

**What goes wrong:** Electron + React + 12 modules loaded simultaneously = 500MB+ RAM. Users with 8GB machines (common in DACH SMBs) feel the pain. Each module adds components, state management, and background processes.

**Prevention:**
- **Lazy-load modules:** Only load the module the user is currently viewing. React.lazy + code splitting.
- **Unload inactive modules** after 5 minutes of inactivity (unmount components, clear state).
- **Set a memory budget** (300MB max) and measure in CI.
- **Use Electron's BrowserView** for heavy modules (video) to isolate memory.

**Severity:** LOW (initially — becomes MEDIUM as modules accumulate).

---

### Pitfall 19: Localization Assumptions

**What goes wrong:** Building everything in German first, then adding English and French as afterthoughts. Or building in English and assuming German localization is "just translation." Date formats (DD.MM.YYYY vs. MM/DD/YYYY), number formats (1.234,56 vs. 1,234.56), currency formats, address formats, and legal terminology all differ between DE, AT, and CH.

**Prevention:**
- **Use i18n from day 1** in the frontend. All user-facing strings go through a translation function.
- **Store dates in UTC, display in user's timezone.** Store numbers as integers/decimals, format for display.
- **German is the primary locale** (DACH target market). English is secondary. Do not build English-first.
- **Swiss German, Austrian German, and German German are different** in legal/financial terminology (Rechnung vs. Faktura, MwSt vs. USt).

**Severity:** LOW. Easy to fix early, painful to fix late.

---

## Phase-Specific Warnings

| Phase/Module | Likely Pitfall | Severity | Mitigation |
|---|---|---|---|
| Gateway refactor (before module 5+) | #1 Gateway God Object | CRITICAL | Split into per-module route registrars, lazy gRPC connections |
| Database schema (ongoing) | #2 Shared DB coupling | CRITICAL | Document table ownership, forbid cross-service writes |
| HR module | #3 German labor law compliance | CRITICAL | Research ArbZG/BUrlG BEFORE writing data model |
| Finance module | #3 GoBD compliance, #13 DATEV | CRITICAL | Immutable invoice model, DATEV CSV export from day 1 |
| Overall scope | #4 Scope paralysis | CRITICAL | Ship Pilot MVP with 4 modules, defer rest |
| Config system | #5 Config explosion | HIGH | Per-service config structs before module 5 |
| Proto files | #6 Proto sprawl | HIGH | Split CRM proto, establish naming convention |
| Cross-module UX | #7 Module isolation | HIGH | Entity linking system, unified activity feed |
| Self-hosted deployment | #8 Container sprawl | HIGH | Build consolidated binary option |
| Email module | #9 MIME/IMAP complexity | HIGH | Use go-imap, scope V1 to send-only |
| Calendar module | #10 RRULE recurrence | HIGH | Hybrid occurrence expansion, RRULE library |
| CI/CD | #11 Test scaling | MEDIUM | Test factories, parallel CI jobs |
| Notifications | #12 Notification fatigue | MEDIUM | Centralized notification service |
| Automation | #14 Premature abstraction | MEDIUM | Hardcoded automations first, engine later |
| Video (LiveKit) | #15 Self-hosted resources | MEDIUM | Optional video, TURN docs, min hardware reqs |
| SaaS launch | #16 Multi-tenancy | MEDIUM->CRITICAL | Add organization_id to all tables NOW |
| API consistency | #17 Error handling | LOW | Shared error proto, gateway mapper |
| Desktop performance | #18 Memory bloat | LOW | Lazy-load modules, memory budget |
| Internationalization | #19 Localization | LOW | i18n from day 1 in frontend |

---

## Solo Developer-Specific Warnings

These pitfalls are amplified by being a solo developer:

1. **No code review safety net.** Every mistake ships. Mitigation: AI-assisted code review, strong linting, comprehensive tests. But acknowledge that subtle bugs (wrong business logic, missed compliance requirement) will not be caught by any automated tool.

2. **Context switching cost is devastating.** Working on 3 modules in parallel (finishing chat notifications, starting project management, fixing CRM bug) means nothing gets done well. Mitigation: Finish one module completely before starting the next. No parallel module development.

3. **Bus factor of 1.** If the developer is sick for 2 weeks, everything stops. There is no "hand off to colleague." Mitigation: Obsessive documentation of decisions (ADRs), comprehensive tests as documentation, clean code over clever code.

4. **Bias toward building over buying/integrating.** As a solo dev with AI tooling, there is a temptation to build everything custom. But integrating an existing CalDAV server (Radicale, Baikal) might be smarter than building calendar from scratch. Mitigation: For every module, ask "Is there an open-source tool I can embed or wrap?" before writing code.

5. **AI-generated code compounds tech debt faster.** AI assistance accelerates development but also accelerates the accumulation of slightly-wrong patterns, duplicated logic, and untested edge cases. The 134 Go source files already built should be periodically reviewed for pattern consistency. Mitigation: Periodic "refactoring sprints" (every 4th sprint is cleanup/consolidation).

---

## Sources

- Codebase inspection: `backend/cmd/gateway/main.go`, `internal/config/config.go`, `internal/server/http.go`, `deploy/docker/docker-compose.yml`
- Historical learnings: `docs/LEARNINGS.md` (10 documented pitfalls from previous project)
- Architecture decisions: `docs/ARCHITECTURE.md` (ADR-001 through ADR-006)
- Existing roadmap: `docs/ROADMAP.md` (phase structure and progress log)
- Project rules: `CLAUDE.md` (architecture rules, common mistakes list)
- DACH regulatory knowledge: Based on training data (MEDIUM confidence). Key laws cited (ArbZG, BUrlG, GoBD, UStG, AO) are well-established German federal law. Specific thresholds and requirements should be verified with a legal/tax professional before implementation.
- Microservice scaling patterns: Based on training data and industry best practices (HIGH confidence for general patterns, MEDIUM for Go-specific recommendations).

**Note on confidence for DACH regulatory pitfalls (#3, #13):** The specific laws and their general requirements are HIGH confidence (these are fundamental, long-standing German regulations). However, exact compliance implementation details (specific DATEV format fields, exact ArbZG exception rules for specific industries) are MEDIUM confidence and MUST be verified with domain experts before implementation. WebSearch was unavailable to verify 2025/2026 regulatory changes.

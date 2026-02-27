# KMU Hub

## What This Is

An all-in-one desktop-first workplace platform for DACH SMBs (5-200 employees) that replaces the daily tool chaos. Every employee — from CEO to office worker — gets a personalizable workspace combining CRM, chat, project management, calendar, email, video calls, HR, finance, and automation in a single Electron app. Tailored per company via onsite process analysis and extended through a Config/WASM plugin system. EU data sovereignty is a core selling point — customers choose between self-hosted and managed SaaS on EU servers.

## Core Value

**Every employee completes their entire workday without opening another program.** The Hub is the only window they need — a digital home that adapts to their role, their workflow, and their company's processes.

## Requirements

### Validated

- ✓ **Authentication & Authorization** — JWT access/refresh tokens, RBAC (admin/manager/member), user management, invitation system, password change with token revocation — *Phase 1*
- ✓ **API Gateway** — HTTP/REST ingress, gRPC proxy, rate limiting (Redis + in-memory fallback), CORS, request logging, Prometheus metrics, health checks — *Phase 1*
- ✓ **Infrastructure** — PostgreSQL + Redis, Docker Compose local dev, CI/CD (lint, test, build, openapi-validate, e2e), structured logging (slog), database migrations (golang-migrate) — *Phase 1*
- ✓ **CRM: Contacts & Companies** — CRUD, custom fields engine, tags system, company-contact linking, full-text search (PostgreSQL tsvector, German config) — *Phase 2*
- ✓ **CRM: Deals Pipeline** — Pipeline stages (CRUD, reorder, defaults), deals (CRUD, stage transitions, tags, custom fields) — *Phase 2*
- ✓ **CRM: Activities** — Call, meeting, note, email, task types, complete/uncomplete — *Phase 2*
- ✓ **CRM: Filters & Reports** — Saved filters per entity type, default filters, pipeline/conversion/activity reports — *Phase 2*
- ✓ **Chat: Channels & Messages** — Channel CRUD, membership, real-time messaging via WebSocket, message edit/delete — *Phase 3*
- ✓ **Chat: DMs & Threads** — Direct messages, thread replies with reply counts — *Phase 3*
- ✓ **Chat: Mentions, Read Receipts, Typing** — @mentions with notifications, per-channel read tracking, typing indicators — *Phase 3*
- ✓ **Chat: File Sharing & Search** — MinIO file store, upload/download/thumbnails, multi-language full-text search (DE/EN/FR/IT/ES) — *Phase 3*

### Completed (Feature Development — Phasen 1-20)

- ✓ **Project management** (tasks, boards, Gantt chart, timer, subtasks, dependencies, templates, CRM linking) — *Phase 6*
- ✓ **Calendar & scheduling** (shared calendars, availability, event management, DACH holidays) — *Phase 7*
- ✓ **Video, voice calls & meetings** (LiveKit, screen sharing, meeting management, emoji reactions, presence) — *Phase 8*
- ✓ **Security & compliance** (2FA, audit log, DSGVO export/deletion, session management, vault, i18n) — *Phase 9*
- ✓ **Email integration** (send/receive within Hub, CRM auto-linking, contact import/export, two-level contacts) — *Phase 10*
- ✓ **Documents & files** (file browser, upload, preview, versioning, sharing, search, tags, global search, OnlyOffice WOPI) — *Phase 11*
- ✓ **Rechnungen & Finanzen** (quotes, invoices, GoBD compliance, DATEV export, dunning — NO full accounting, NO payroll) — *Phase 12*
- ✓ **HR & Zeiterfassung** (leave requests, time tracking, employee profiles, ArbZG/BUrlG compliance — payroll via integration only) — *Phase 13*
- ✓ **Event infrastructure + Unified Inbox** (aggregated view across Email/Chat/Notifications) — *Phase 14*
- ✓ **CalDAV/CardDAV sync** (bidirectional calendar and contact sync with Outlook/Thunderbird/macOS) — *Phase 15*
- ✓ **Automation engine** (workflows, triggers, automatic actions — killer feature of all-in-one) — *Phase 16*
- ✓ **External integrations** (Teams/Slack notifications, Bexio CH sync, DATEV API + Lexware Office DE) — *Phases 17-19*
- ✓ **Plugin system** (Config-based modules, WASM wazero runtime, extension points, industry templates) — *Phase 20*
- ✓ **Desktop app** (Electron + React, personalizable workspace, role-based dashboards) — *Phase 5*
- ✓ **Guest Chat / Kundenportal** (external visitor chat, unified inbox integration) — *Phase 17.5*

### Active — Beta Preparation

- [ ] **Frontend API-Wiring** (18 Stores auf Mock-Daten → echtes Backend, Phasen A-B)
- [ ] **Legal & Compliance** (Anwalt, AGB, AVV/DPA, DSGVO-Prüfung)
- [ ] **Production Infrastructure** (Hetzner Server, Docker Compose, SSL, Domain)
- [ ] **QA & E2E-Tests** (kritische Pfade, Fehler-Monitoring, Backup-Strategie)
- [ ] **Pilot-Onboarding** (Zentrum für finanzielle Aufklärung, erster Kunde)

### Out of Scope

- **Full document editor / Wiki** — Building a Notion/Google Docs competitor is a separate product. Will add simple doc linking and a knowledge base later, not a rich editor. Focus on workflow tools first.
- **Mobile app** — React Native is planned long-term, but desktop-first. Mobile comes after the desktop daily-driver is proven.
- **Office suite replacement** — We integrate with Word/Excel/Google Docs, not replace them. If a customer insists on a specific tool, we build a connector.
- **Real-time collaboration editing** — OT/CRDT-based document co-editing is out of scope. Leave this to specialized tools.
- **AI features** — Not in v1 scope. The development uses AI, but the product doesn't expose AI features to end users yet.

## Context

**Team:** 3 people (1 developer + 2 business), all with full-time jobs. Revenue is not the primary goal — building something that feels like home is. This means we can take quality-focused decisions without VC pressure.

**Pilot customer:** Zentrum für finanzielle Aufklärung (Center for Financial Education) has expressed interest. Their pain: tool chaos across Excel, email, Teams, and misc. tools. Additional pilot possible via team member's employer.

**Market position:** DACH SMBs underserved by enterprise tools (too complex, too expensive, US-hosted) and consumer tools (too simple, no customization). KMU Hub fills the gap with EU-sovereign, tailored, affordable all-in-one.

**Existing codebase:** Go microservices backend (12 services: gateway + auth + CRM + chat + notification + work + document + biz + automation + caldav + plugin + email), Electron + React + TypeScript desktop shell, PostgreSQL + Redis + MinIO, 61 Migrationen, comprehensive CI/CD pipeline. Alle 20 Feature-Phasen abgeschlossen (2026-02-26). Jetzt: Beta Preparation.

**Business model:** Onsite 1-week process analysis → custom Hub configuration → ongoing support. Pricing sensitive to SMB budgets (SaaS subscription + self-hosted license option).

## Constraints

- **Tech stack**: Go backend (microservices + gRPC), Electron + React + TypeScript desktop, PostgreSQL + Redis — established, do not change
- **EU data sovereignty**: All hosting EU-only (Hetzner/OVH). No US cloud providers for customer data. Non-negotiable for DACH market trust.
- **Team size**: 1 developer + AI tooling. Architecture must stay manageable for solo dev — clear service boundaries, good testing, automated deployment.
- **Self-hosted compatibility**: Every feature must work in Docker Compose deployment. No SaaS-only dependencies.
- **Desktop-first**: Electron app is the primary interface. Web access is secondary. All features designed for desktop UX patterns.
- **No dual-write**: PostgreSQL is source of truth. Redis is cache only. MinIO is file storage only. No split-brain architectures.
- **Plugin isolation**: WASM plugins run sandboxed. Config plugins operate within defined extension points. Neither can bypass auth or access raw DB.

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Go microservices + gRPC | Performance, type safety, single-binary deployment, strong concurrency model | ✓ Good — clean service boundaries established |
| Electron + React desktop | Cross-platform, web tech reuse, rich UI capabilities, familiar ecosystem | ✓ Good — desktop shell functional |
| PostgreSQL full-text search over Elasticsearch | Simpler ops, good enough for SMB scale, one less service to manage | ✓ Good — multi-language FTS working well |
| MinIO over cloud S3 | Self-hosted compatible, S3 API means easy migration, EU sovereign | ✓ Good — file sharing working |
| LiveKit for video | Self-hostable, WebRTC-based, open source, EU-deployable | — Pending |
| Config + WASM plugin system | Config for 80% of customization, WASM for complex cases — avoids full scripting engine security nightmare | ✓ Good — wazero runtime, hook dispatcher, Fuhrpark reference plugin |
| Case-by-case integrations | No generic integration framework upfront. Build connectors as customers need them. Pragmatic over comprehensive. | ✓ Good — Bexio, DATEV, Lexware, Teams, Slack |
| Role-based dashboards + personalizable workspace | Every employee sees what they need. CEO dashboard ≠ office worker dashboard. Onsite analysis determines defaults. | ✓ Good — dashboard system working |
| Feature gap analysis expansion | Security before business modules, meetings merged into video, integrations as mini-phases, documents as standalone module. Roadmap from 13 to 20 phases. | ✓ Applied — all 20 phases complete |
| Buchhaltung → Finanzen rename | "Rechnungen & Finanzen" (invoices + finance), NOT full accounting. No doppelte Buchfuehrung, no payroll. | ✓ Applied |
| Payroll as anti-feature | Lohnabrechnung NEVER built in-house. Integration only via Bexio/Abacus/RmA. 8 planned endpoints struck. | ✓ Applied |
| Phase reorder (strategy session) | Unified Inbox as Phase 14, Automation vorgezogen to 16, Abacus+RmA merged into 19. Industry modules as Phase 20 plugins. | ✓ Applied |
| Beta scope: Mock-vs-Real | Industry-specific modules (Fuhrpark, Inventar, Produktion etc.) bleiben Demo-Daten für v1. Erst bei realem Kundenbedarf verdrahten. Qualität vor Quantität. | ✓ Entschieden 2026-02-27 |
| Frontend API-Wiring strategy | Stores nach Priorität verdrahten: CRM (Kontakte ✅, Firmen, Deals) → Work → Kalender → Finanzen → Dashboard → Team → Chat → Dokumente. | ✓ Entschieden 2026-02-27 |

---
*Last updated: 2026-02-27 — Beta Preparation started (all 20 phases complete)*

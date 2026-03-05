# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-27)

**Core value:** Every employee completes their entire workday without opening another program
**Current focus:** Beta Preparation — Frontend API-Wiring + Legal + Business
**Recent strategy changes:** Alle 20 Feature-Phasen abgeschlossen. Scope-Entscheidung: 11 industry-spezifische Stores bleiben Demo-Daten fuer v1 (Plugin-Roadmap).

## Current Position

Mode: Beta Preparation
Status: Phase A (Core Wiring) — COMPLETE. All technical items done.
Last activity: 2026-03-05 -- Store cleanup (kommunikation.ts dead code removed), ROADMAP + STATE updated
Next: Phase B (Beta Hardening) — DokumentePage wiring, D9 Design-Merge, E2E tests

Feature Development: [████████████████████████████████████] 100% (103/103 plans across phases 4-20)
Beta Preparation:    [████████████░░░░░░░░░░░░░░░░░░░░░░░░] ~30% (Phase A Track 1 done, 9/10 core modules wired, Legal/Business 0%)

## Phase A Completion Summary (2026-03-05)

| Item | Commit/Date | Status |
|------|-------------|--------|
| KontaktePage → Backend | `be1f0b3` (2026-02-27) | Done |
| FirmenPage + DealsPage + CRM | `3d587b6` (2026-02-27) | Done |
| WorkPage + KalenderPage | `15005ac` (2026-02-27) | Done |
| FinanzenPage + Dashboard | `15005ac` (2026-02-27) | Done |
| Dashboard TeamStatus + TeamChat | `3fb4856` (2026-03-04) | Done |
| TeamPage → hr-hooks.ts | `e5ff87e` (2026-03-04) | Done |
| Kommunikation → inbox API hooks | `34c2f23` (2026-03-04) | Done |
| Work mock TODOs | `e2aaf7f` (2026-03-05) | Done |
| Finance mock TODOs | `0def695` (2026-03-05) | Done |
| Store cleanup (kommunikation.ts) | 2026-03-05 | Done |

## Beta Roadmap

Drei Phasen, drei parallele Tracks. Details: .planning/ROADMAP.md

| Phase | Zeitraum | Technisch | Legal | Business |
|-------|----------|-----------|-------|----------|
| A — Core Wiring | Maerz 2026 | ✅ DONE (9 modules wired) | Anwalt, Unternehmensform | Kundengespraech, Hetzner |
| B — Beta Hardening | April 2026 | DokumentePage, D9 Design, E2E | AGB, DSGVO, AVV/DPA | Website, Preisliste |
| C — Beta Launch | Mai 2026 | Performance, Self-Hosted-Paket | Rechtstexte live, Impressum | Pilot-Onboarding |

## Scope-Entscheidungen (2026-02-27)

**Mock bis v2 (nicht verdrahtet fuer Beta):**
einkauf.ts, inventar.ts, produktion.ts, vermietung.ts, fuhrpark.ts,
rapporte.ts, schichten.ts, vertraege.ts, wiki.ts, formulare.ts, berichte.ts

**Begruendung:** Industry-Templates (Phase 20 Plugins), erst bei realem Kundenbedarf verdrahten.

**Kritische Blocker:**
1. Legal (AVV/DPA) → Blocker fuer Pilot-Onboarding mit echten Kundendaten
2. Produktionsserver (Hetzner) → Blocker fuer Beta-Deployment
3. D9 Design-Merge → vor erstem Pilot-Kunden-Demo

## Performance Metrics

**Velocity:**
- Total plans completed: 103
- Average duration: ~7 minutes
- Total execution time: ~11h 30min

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
| 20 | 4/4 | ~45min | ~11min |

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
- [Beta-Prep]: Team + Chat modules pulled forward from Phase B to Phase A (completed early)
- [Beta-Prep]: team.ts members store NOT removable -- still used by MentionAutocomplete + training participation dialog (no API yet)

### Blockers/Concerns

- [Phase 12]: GoBD compliance requires Steuerberater consultation before data model design
- [Phase 13]: ArbZG/BUrlG implementation details need labor law expert review
- [Phase 12]: DATEV Buchungsstapel format spec not publicly detailed -- may need DATEV partner access

## Session Continuity

Last session: 2026-03-05
Stopped at: Phase A audit complete -- store cleanup done, ROADMAP + STATE updated
Resume file: N/A
Next: Phase B -- DokumentePage wiring OR D9 Design-Merge

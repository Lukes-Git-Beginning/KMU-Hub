---
tags: [fortschritt, milestones]
updated: 2026-04-19
---
# Milestones

## Sprint 1 Session 2026-04-18 — R2-P0 Batch A + Wiki + Helpdesk end-to-end
| Schritt | Details |
|---------|---------|
| R2-P0.2 LiveKit-Secrets Startup-Assertion | `310c803` — Prod crasht bei devkey/devsecret |
| R2-P0.5 Egress-Webhook `egress_ended` | `d8f89d4` — setzt `recordings.status=completed` + `file_url` |
| R2-P0.6 Lexware Webhook HMAC | `787c327` — HMAC-SHA256 Verifikation auf eingehenden Webhooks |
| R2-P0.3 Recording-Consent-Bug | `efd752a` — `StartRecording` pruft Consent fuer alle Call-Teilnehmer |
| R2-P0.1 coturn-Prep (flag-off) | `a9749fa` — `livekit.yaml` Overlay + Go-TURN-Credential-Propagation, Deploy offen |
| S1.1 Wiki Backend-Modul | `601a815` — 15 RPCs, Postgres-FTS (tsvector+GIN, deutsch), 5 Tabellen, Coverage 38.2% |
| S1.4 Helpdesk Backend-Modul | `c2d179e` — 22 RPCs, SLA-Engine + Ticket-Merge (ILIKE-Prefix), Coverage 39.3% |
| S1.1 Wiki Wiring | `75c783e` — Proto, gRPC-Server, `cmd/wiki` Binary, `route_wiki.go` hinter `modules.wiki`-Flag |
| S1.4 Helpdesk Wiring | `2d8f6d3` — Proto, gRPC-Server, `cmd/helpdesk` Binary, `route_helpdesk.go` hinter `modules.helpdesk`-Flag |
| Frontend Clients + Hooks | `eed1329` — `wiki-client.ts`/`useWiki.ts` (21 Hooks), `helpdesk-client.ts`/`useHelpdesk.ts` (28 Hooks), `useRecordingStatus` Polling |
| Gateway + Docker-Compose Activation | `0ac916c` — Registry-Register aktiv, `Dockerfile.wiki`/`Dockerfile.helpdesk`, Services in compose |
| Knowledge-Base Update | `7349ba3` — `_index.md`/`api.md`/`datenbank.md` |

**Gesamt:** 13 Commits auf main, `go build ./...` + `go test ./...` gruen. Wiki + Helpdesk default-OFF via Feature-Flags. Offen: `.env.example` uncommitted (Hook-Whitelist blockiert), TURN-Deploy wartet auf CPX11-Provisionierung.

## Sprint 1 Session 2026-04-19 — S1.2 Berichte Completion
3-Wellen-Subagent-Pipeline fuer die verbleibenden 5 Work-Packages (WP-3/5/6/7/11). Plan: `~/.claude/plans/sodele-was-steht-als-structured-raccoon.md`. Ports 50063/9103 (Luecke zwischen wiki und helpdesk gefuellt).
| Schritt | Details |
|---------|---------|
| WP-3 Export-Layer | `5039f79` — `internal/berichte/export/` mit PDF (maroto v2) + CSV (strings.Builder, UTF-8-BOM + Semikolon fuer DATEV) + XLSX (excelize v2.8.1). Coverage 80.2% ueber Golden-File-Tests |
| WP-5 gRPC-Server + cmd | `a742b9e` — `server/berichte_grpc.go` (14 RPCs, UUID-Validation, `mapBerichteError`), `cmd/berichte/main.go` mit Scheduler-Goroutine + Graceful-Shutdown, `Dockerfile.berichte`, Config-Felder (`BerichteGRPCPort/Address/HealthPort`). Coverage 77.6% |
| go.mod tidy | `22fe40f` — cron/v3 + excelize/v2 von indirect → direct (nach Welle 1) |
| WP-6 Gateway-Routes | `e76441a` — `gateway/route_berichte.go` mit 14 HTTP-Endpoints, `modules.berichte`-Gate, RBAC-Middleware `RequirePermission("berichte:reports", read|write)`, Export-Response via `Content-Disposition`. Coverage 57%. Migration 000080 seed_berichte_permissions |
| WP-7 Docker-Compose | `98d60c3` — `berichte`-Service-Block (dev + prod), Gateway `depends_on: berichte {service_healthy}` + `BERICHTE_GRPC_ADDRESS=berichte:50063` |
| WP-11 Final-Wire + Smoke | `a4b2cc9` — Exporter-Stub in `cmd/berichte/main.go` durch `export.NewExporter`-Adapter ersetzt, `smoke.sh` um 3 Berichte-Checks (Definitions/Run/Export-MIME) erweitert — Flag-OFF gracefully als Pass. ROADMAP S1.2 ✅ Done |

**Gesamt:** 6 Commits auf main. Gate S1.2 erfuellt. Berichte default-OFF via `modules.berichte`-Flag. Tenant-ID bleibt Placeholder `00000000-…-000001` bis JWT-Claim-Extraktion in Sprint 2 (Option-B Phase 1).

## Abgeschlossene Meilensteine
| Meilenstein | Phasen | Abgeschlossen |
|------------|--------|--------------|
| Foundation | 1-3 | vor GSD-Adoption |
| Pilot MVP | 4-8 | 2026-02-11 |
| Compliance & Comms | 9-11 | 2026-02-17 |
| Business Suite | 12-13 | 2026-02-19 |
| Aggregation & Automation | 14-16 | 2026-02-20 |
| Integrations | 17-19 (+17.5 Guest Chat) | 2026-02-26 |
| Extensibility | 20 (Plugin System + WASM) | 2026-02-26 |

## Beta Phase A — Core Wiring (abgeschlossen)
| Schritt | Abgeschlossen |
|---------|--------------|
| 9 Core-Module auf echte API-Hooks migriert | 2026-03-05 |
| D9 Design-Merge (Waves 15-20) | 2026-03-07 |
| Lint-Cleanup (347 ESLint-Probleme auf 0) | 2026-03-07 |
| Phase A Dead-Code Audit | 2026-03-05 |

## Beta Phase B9 — Crash-Fixes & Playwright Testing (2026-04-01)
| Schritt | Details |
|---------|---------|
| Playwright MCP eingerichtet | Chrome CDP Port 9222, .mcp.json konfiguriert |
| MSW durch Fetch-Interceptor ersetzt | demo-mode.ts, sauberer als MSW Service Worker |
| RichTextEditor entfernt | Ungenutzte shared component |
| Business Roadmap erstellt | docs/BUSINESS-ROADMAP.md |
| ErrorBoundary: Route-Reset | ModuleShell key={location.pathname}, kein Reload nötig |
| 9 Modul-Crashes gefixt | Inventar (duplicate import), Einkauf (null guard), Formulare (null guard), Vermietung (objectName/currency), Dashboard Widgets (camelCase, activities, pipeline) |
| 5 weitere Null-Guards | CalendarUpcoming (today/dd scope), MyCalendar (now scope), EinkaufPage (showWareneingangDialog?.id), FormularePage (showShareDialog?.name, editingForm), ZustandsprotokollDialog (reservation?.objectName) |
| Projekte Mock-Daten | project_key, is_template, Handler pagination |
| Playwright-Verifikation | Alle Module crash-frei verifiziert (0 JS-Errors) |

## Beta Phase B10 — Design Audit & Rebrand (2026-04-01)
| Schritt | Details |
|---------|---------|
| Design Audit (36 Screenshots) | Playwright-Session, alle Module visuell geprüft, Score ~6.6/10 |
| Auth Redesign | AuthLayout mit Split-Layout + Brand-Panel für Login/Register |
| Empty States normalisiert | Projekte (EmptyGeneric), Zeiterfassung (EmptyCalendar), Buchhaltung (Props gefixt) |
| Team Tabs Overflow | Fade-Mask + scrollbar-hide für 11-Tab-Leiste |
| Wiki Selection Highlight | Full-row bg → Left-border Accent |
| Kalender Headers | text-[10px] → text-xs font-medium (Wochen- und Monatsansicht) |
| Rebrand: KMU Hub → Cosmi | 36 Dateien, alle user-sichtbaren Texte |
| Locale: de-CH → de-DE | 104 Dateien, Default-Währung CHF → EUR |
| Umlaut-Normalisierung | ae/oe/ue → ä/ö/ü in ~255 Dateien (nur Display-Text, nicht Code) |

## i18n Migration Sprint (2026-04-06)
| Schritt | Details |
|---------|---------|
| Library-Migration | react-intl → i18next v26 + react-i18next v17 + i18next-icu v2 |
| Wave 1: Module | 32 Module instrumentiert (useTranslation + t() Calls) |
| Wave 1: Komponenten | 9 Komponentengruppen instrumentiert (46 Dateien) |
| Additions-JSONs | 41 JSON-Dateien in `i18n/additions/` — 4.500+ Schluessel |
| Merge-System | `mergedDE` in `i18n.ts` — alle Additions statisch in de.json integriert |
| Verbleibend (Wave 2+3) | ~47 Dateien (Settings, Integrations, Sub-Pages, Dialoge) |
| Verbleibend | Keys in de.json konsolidieren, EN/FR/IT-Uebersetzungen, Strict Types |

## Performance-Optimierung (2026-04-08)
| Phase | Inhalt | Status |
|-------|--------|--------|
| Phase 1 — Quick Wins | Bundle-Analyzer, Fonts self-host, Demo Dead Code, motion entfernt | ✅ |
| Phase 2 — Frontend | Chunk-Splitting, Async Persister, HR Polling, React Compiler, List Virtualization | ✅ |
| Phase 3 — Backend | N+1 Queries (Contact 61→4, Deal 121→7), Batch-Inserts, owner_id Index, Pool Fix, PG Tuning | ✅ |
| Phase 4 — Electron | V8 Compile Cache, modulePreload, Skeleton Screen | ✅ |
| Phase 5 — Gateway | Audit Logger Worker Pool, gRPC Keep-Alive, pprof | ✅ (Redis Caching offen) |

5 parallele Agenten (Worktree-Isolation), 6 Commits.
Detaillierter Plan: `docs/PERFORMANCE-PLAN.md`

## Dialer-Modul Phase 1 (2026-04-09)
| Sub-Phase | Inhalt | Status |
|-----------|--------|--------|
| 1A — Foundation | Proto (27 RPCs), 5 Migrations (063-067), Service-Skeleton, Docker | ✅ |
| 1B — Backend Core | service.go (24 Methoden), 4 Repos, Redis Agent-Status, CRM-Bridge, gRPC-Server | ✅ |
| 1C — Gateway + Permissions | 25 REST-Endpoints, Permission-Migration (068), route_dialer.go (1014 LoC) | ✅ |
| 1D — Frontend | 26 Dateien, DialerWorkspace (4-Phasen Call-Flow), Campaigns, Dashboard, Settings, Mock-Handler, i18n (4 Sprachen) | ✅ |
| 1E — Integration | CRM-Timeline live, Callback-Notifications, Filter-Import, Bug Fixes (ContactID, wrap_up, skip), EventEmitter, Unit/Gateway/E2E Tests | ✅ |

Strategische Roadmap: `docs/DIALER-ROADMAP.md`

## Rigorosum Runde 1 + 2 (2026-04-18)
- **Runde 1 (wild-wren):** Gesamtnote 3.3 — 7 P0-Launch-Blocker + 8 P1 + 7 P2 + 9 P3 identifiziert (Backend/Frontend/Ops)
- **Runde 2 Vertiefung (functional-seahorse):** Gesamtnote 4.1 — 9 neue P0-Blocker in Integrationen, Realtime-Kern, DB-Schema + 12 P1 + 15 P2 + 6 P3
- Kombinierte Launch-Reife **3.7** → Launch auf 2026-07-01 verschoben (+4 Wochen)
- Strategische Entscheidungen: Option-B-Full Multi-Tenancy, coturn self-hosted, Join-with-Consent, WASM Feature-Flag OFF, `finance_invoices.line_items` vor Launch normalisieren
- Details: siehe `docs/ROADMAP.md` (Single Source of Truth) und MEMORY `project_rigorosum_april.md`, `project_rigorosum_runde2.md`

## Sprint 0 — Launch-Blocker abgeraeumt (2026-04-18) ✅
Alle 7 R1-P0-Blocker + R2-P1.2 (WASM-OFF) + Cleanup + Modul-Scope-Matrix in drei parallelen Wellen.
| # | Task | PR |
|---|---|---|
| S0.1 | Migration 000075: `consent_records.contact_id` ON DELETE SET NULL | #5 |
| S0.2 | `AssertConsent`-Wrapper vor SendEmail + DialerCall (`crm/consent/`) | #10 |
| S0.3 | Prod-Secrets Startup-Assertion (`JWT`, `VAULT`, `WOPI_JWT`, `MINIO`) | #6 |
| S0.4 | DOMPurify `lib/sanitize.ts` fuer 5 Call-Sites | #9 |
| S0.5 | OnlyOffice `JWT_ENABLED: true` in Prod-Override | #7 |
| S0.6 | Feature-Flag-Registry (16 Flags, API, useFeatureFlags, FeatureGate) + WASM Build-Tag `!no_wasm` | #11 |
| S0.7 | ICU-Plural-Klammer-Fix (18 Strings × 4 Sprachen) | #3 |
| S0.8 | `mobile/`-Ordner entfernt, Pitch auf PWA | #4 |
| S0.9 | `docs/MODULES_SCOPE_MATRIX.md` (14 Module) | #8 |

Gate S0 bestanden. Sprint 1 startet 2026-04-28 mit 7 Modulen + TURN-Server + LiveKit-Secrets + Recording-Consent-Fix + Egress-Webhook + Lexware-HMAC (R2-P0 Batch A).

## Code Review Hardening (2026-04-09)
| Finding | Status |
|---------|--------|
| tenant_id auf contacts/companies (Migration 070) | ✅ |
| Desktop Tests reparieren (62/62 grün) | ✅ |
| IP Filter Fail-Close mit 5min TTL | ✅ |
| gRPC mTLS (optional, env-var-gesteuert) | ✅ |
| Gateway Bloat reduzieren (route_crm_ext.go) | ✅ |

Commits (cdaeefd–8f13465): 14 neue gRPC RPCs (11 CRM + 3 Biz), pgxpool aus Gateway entfernt, ~570 Zeilen Boilerplate reduziert, 3 große Dateien gesplittet, main.go cleanup.
Vorherige Commits (d136ea6, 96740e8, 927dbcf): HR tenant_id, SQL aus Gateway in Repos, Security/CI/Coverage Fixes.

## Verwandte Notes
- [[i18n]] — i18n-Architektur & Schluessel-Konventionen
- [[design]] — Frontend Wiring Progress
- [[architektur]] — Technischer Kontext, Performance-Patterns

---
tags: [fortschritt, milestones]
updated: 2026-04-09
---
# Milestones

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

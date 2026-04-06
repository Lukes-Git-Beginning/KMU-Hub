---
tags: [fortschritt, milestones]
updated: 2026-04-06
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

## Verwandte Notes
- [[i18n]] — i18n-Architektur & Schluessel-Konventionen
- [[design]] — Frontend Wiring Progress
- [[architektur]] — Technischer Kontext

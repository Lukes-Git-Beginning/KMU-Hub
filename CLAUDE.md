# CLAUDE.md - KMU Hub CRM

> Schlanker Quick-Reference. Detailwissen liegt im Knowledge-Vault unter `.knowledge/` (Single Source of Truth) — siehe Pointer-Block am Ende.

## Projekt-Kontext

- **Mission:** All-in-One CRM fuer DACH-KMUs mit EU-Datensouveraenitaet
- **USP:** Massanfertigung durch 1-Woche-Onsite-Prozessanalyse + Config/WASM-Plugin-System
- **Zielgruppe:** Branchenunabhaengige KMUs (5-200 Mitarbeiter)
- **Team:** 1 Dev + 2 Business, AI-First Development
- **Branding:** Software="Cosmi", Firma="Zentria"
- **Ziel:** **Produkt 1.0.0** nach Reifegrad-Gates, **kein Kalenderdatum** (das Launch-Datum 2026-09-01 ist seit 2026-08-12 entwertet). Definition und Sequenz: `.planning/launch-lagebild-2026-08-12.md` §3 und §6
- **Version:** 0.1.0

## Tech-Stack

| Komponente | Technologie |
|-----------|-------------|
| Backend | Go (API-Gateway + 23 gRPC-Microservices = 24 `backend/cmd/*`-Dirs) |
| Desktop | Electron + React 19 + TypeScript |
| Mobile | PWA (Phase E, auf Desktop-Basis) |
| Datenbank | PostgreSQL 16 + Redis 7 (Cache only) |
| Video | LiveKit (self-hostable) |
| Plugins | Config-basiert + WASM (aktuell Feature-Flag OFF) |
| Hosting | EU-only (Hetzner), SaaS + Self-Hosted Option |

## Architektur-Regeln (KRITISCH)

Bullet-Liste — Code-Beispiele und Detail-Erlaeuterungen siehe `[[architektur#architektur-regeln-detail]]`.

1. **Thick Services, Thin Handlers** — Business-Logik in Services, Handler nur Parse/Call/Respond
2. **Centralized Service Registry** — explizite DI in `cmd/gateway/main.go`, kein `init()`, keine Globals
3. **Structured Logging (slog)** — kein `fmt.Println`, kein `console.log`
4. **API-First** — OpenAPI-Spec vor Implementation (`backend/api/openapi.yaml`)
5. **Migrations via golang-migrate** — `make migrate-create name=xxx`, NIE manuell SQL
6. **Test-Coverage** — 15%+ gesamt (CI-enforced), 60%+ kritische Pfade (Auth, Payments, Data)
7. **Security First** — Auth + Rate Limit + Input Validation + CSRF + Prepared Statements + CORS-Allowlist + Env-Secrets
8. **Graceful Degradation** — Services muessen unabhaengig ausfallen koennen (CRM ohne Chat etc.)
9. **Config via Env-Vars** — `.env` NIE committen (Pre-Commit-Hook blockt), Production-Secrets-Assertion beim Start
10. **Idempotente Operationen** — Idempotency-Keys fuer POST (Dialer-Outcomes, Finance-Postings, Webhooks)
11. **Tenant-Modell** — Option-B-Full: alle Tabellen `tenant_id UUID NOT NULL` + Row-Level-Security. RLS **produktiv erzwungen** (`COSMI_ENV=production` scharf seit 2026-06-05); App-Services laufen als `kmuhub_app` (NOSUPERUSER NOBYPASSRLS), DDL-Migrations als `kmuhub`. Daten aktuell Single-Tenant, Code Multi-Tenant-fähig. Neue Tabellen MUESSEN `tenant_id` + RLS-Policy haben — oder explizit in die System-Global-Liste (ADR-006, `docs/ARCHITECTURE.md`) eingetragen werden

**Entwicklungs-Kommandos** (Backend / Desktop / Docker): `[[architektur#entwicklungs-kommandos]]`

## UI/UX

Aesthetik: **"Premium SaaS mit Editorial Touch"** — kein generisches Dashboard-Look.

**Designphilosophie:** Cosmi-Identitaet bleibt eigen. Apple-Linse (Reduktion, Hierarchie, Daily-Use-Disziplin) fuer alles was 100x/Tag passiert. Discord-Linse (Waerme, Personality, Joy-Moments) fuer Empty-States, Onboarding, Success. **Keine visuellen Kopien** (kein Mac-Chrome, keine Discord-Sidebar, keine Mascots von draussen).

- **Font-Bans:** NIEMALS Inter, Roboto, Arial, Space Grotesk, Helvetica, Open Sans. Erlaubt: Plus Jakarta Sans, Clash Display, Satoshi, JetBrains Mono, Playfair Display
- **Anti-Patterns:** Card-in-Card, AI-Slop-Aesthetik (lila Gradienten), symmetrische Bootstrap-Grids, Ueber-Animation
- **Motion-Hardrule:** Nur `transform`/`opacity` animieren (GPU). Nie `width/height/margin/padding`. Tokens in `lib/motion.ts` + `styles/animations.css` — keine magic numbers in Komponenten.
- **Keine Emojis in UI** (Personality via Custom-SVG, Motion, Wording).
- **Skills:** `frontend-design` (auto-loaded), `impeccable` (`/audit`, `/critique`, `/polish`, `/animate` etc.). On-demand: `emilkowalski/skill`, `kylezantos/design-motion-principles`.

Vollstaendige Direktiven (Joy-Matrix, Personality-Guidelines, Motion-Tokens, Workflow): `[[design]]`

## Haeufige Fehler

Vermeide die Fehler aus dem Vorgaenger-Projekt (slot_booking_webapp): Dual-Write, Business-Logik in Handlern, manuelle Migrations, fehlende Test-Isolation, Komponenten kopieren statt wiederverwenden, Tailwind-JIT-Runtime mit CSP, Deployment ohne Backup.

Vollstaendige Liste mit Beispielen: `[[troubleshooting]]`

## Deployment-Reihenfolge (KRITISCH)

1. Backup erstellen
2. Assets bauen (CSS, JS, Electron)
3. Config aktualisieren
4. Code deployen
5. Migrations ausfuehren
6. Services restarten
7. Health Check
8. Rollback-Plan bereithalten

Self-Hosted-/SaaS-Details, Hetzner-Setup, CI/CD-Workflows: `[[deployment]]`

## Git-Regeln

- **Conventional Commits:** `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`
- **Branch-Strategie:** Ab Sprint 1 (2026-04-18) **direct-to-main ist Default**. Keine Feature-Branches, keine PRs — ausser explizit gefordert
- **CI-Rot-Recovery:** `git revert <sha>`. **NIE** `reset --hard`, **NIE** Force-Push
- **Keine AI-Attribution** (kein `Co-Authored-By`, kein "Generated by")
- **Commit-Messages:** Englisch, imperativ ("Add contact endpoint", nicht "Added...")
- **Push-Rhythmus:** Am Ende jeder Session, um Divergenz zu vermeiden

## Modul-Arbeit: Build-+-Verify-Standard (verbindlich, fuer JEDE Phase)

Gilt fuer alle, die Frontend-Module bauen (Haupt-Team + delegierte Mitarbeit). Pro Phase IMMER dieselbe Schleife: bauen → i18n ×4 (`{var}`, nie `{{var}}`; Plural als ICU `{count, plural, …}`, nie `_one`/`_other`) → Demo-Handler falls noetig → **gescopter Typecheck** (nur geaenderte Dateien, nie Full-tsc als Gate) → **Playwright-Screenshot-QA + die Screenshots wirklich ansehen** (Raw-Keys/Emojis/Layout/leere Zustaende) → iterieren bis gruen → ein Commit + Push. „Kompiliert ja" oder „Script lief gruen" allein reicht NICHT — die Bilder muessen angeschaut werden.

**Exakter Prozess + kopierbare Vorlagen:** `.planning/nico-block/WORKFLOW.md`. Delegations-Paket (Runbook, Repo-Map, Pilot-Specs): `.planning/nico-block/`.

## Knowledge-Vault (.knowledge/)

Single Source of Truth fuer projektspezifisches Wissen. Notes haben YAML-Frontmatter (`tags`, `updated`), verlinken via `[[note-name]]`. Lesen via MCP-Filesystem-Tools (`mcp__knowledge__read_text_file`).

| Note | Inhalt |
|------|--------|
| `_index.md` | Master-Index, 6-Sprint-Roadmap bis Launch 01.09 |
| `architektur.md` | Services, Routes, Architektur-Regeln (Detail mit Code), Entwicklungs-Kommandos, Feature-Flags, Consent-Wrapper, WASM-OFF |
| `stack.md` | Strategy Decisions, Frontend-Bibliotheken, dompurify, Mobile=PWA |
| `i18n.md` | i18next-Architektur, Schluessel-Konventionen, ICU-Plural-Bug |
| `design.md` | Design-System, Themes, UI/UX-Direktiven, Magic UI, LanguageSwitcher |
| `datenbank.md` | Schema, Migrationskopf 000213 (Prod 209) |
| `api.md` | 28 Endpoint-Domains, Auth-Flow |
| `security.md` | JWT, RBAC, Consent-Enforcement, Prod-Secrets-Assertion, DOMPurify |
| `integrationen.md` | Bexio, Lexware, DATEV, LiveKit, OnlyOffice JWT, Plugin-WASM (off) |
| `deployment.md` | Docker, CI/CD, Hetzner, `make build-prod -tags no_wasm` |
| `testing.md` | Test-Strategie, Sprint-0-Neuzugaenge |
| `pricing.md` | Modul-x-User Preismodell (COSMI + ORBIT) |
| `troubleshooting.md` | Bekannte Probleme, Git-Workflow, Tailwind/CSP, ICU-Plural, Radix-Null |
| `tooling-graphify.md` | Graphify-Eval (vertagt auf Sprint 2/3) |
| `milestones.md` | Meilensteine inkl. Rigorosum Runde 1+2 + Sprint 0 Closure |

## Intel-System (zentria-intel)

Paralleles Markt-Intelligence-Repo (`~/Documents/zentria-intel/`, github.com/Lukes-Git-Beginning/zentria-intel) — taeglicher CRM-/KMU-Markt-Scan, Friday-Synthese, Discord-Pick-Mechanik, Recall in Cosmi-Sessions. Memory-Pointer in `MEMORY.md > ## Intel-System`.

| Slash-Command | Zweck |
|---|---|
| `/intel-bootstrap` | Smoke-Test, Discord-Server-Setup, Routinen-Registrierung (drei Sub-Modi `--smoke`/`--discord`/`--routines`) |
| `/intel-friday` | Manueller Trigger der Friday-Synthese (Routine-Backup, optional `--week=YYYY-Wxx`) |
| `/intel-pick <id> <action> [tags] [note]` | Backup zur Discord-Bot-Pick-Mechanik (Stable-ID `W..-T..-i..` aus Friday-Report) |
| `/intel-promote <modul> [--dry-run]` | Synthetisiert ≥5 Keepers über ≥3 Wochen zu `.knowledge/intel-<modul>.md`-Note |
| `/intel-recall <modul-oder-thema> [--days=90]` | Laedt Keepers + promoted Notes nach Modul/Thema in den Context |

## Weiterfuehrende Dokumentation (Repo-Root `docs/`)

| Thema | Datei |
|-------|-------|
| **Lage und Sequenz (Single Source of Truth)** | `.planning/launch-lagebild-2026-08-12.md` |
| Architektur-Entscheidungen (ADRs) | `docs/ARCHITECTURE.md` |
| Learnings aus Vorgaenger-Projekt | `docs/LEARNINGS.md` |
| Pricing-Modell | `docs/PRICING.md` |
| Roadmap (⚠ Kalendermodell überholt 2026-08-12, Messwerte gelten) | `docs/ROADMAP.md` |
| Modul-Scope-Matrix | `docs/MODULES_SCOPE_MATRIX.md` |
| Live-Reifegrad (deskriptiver Snapshot) | `.planning/status-overview.md` |

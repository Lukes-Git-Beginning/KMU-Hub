# CLAUDE.md - KMU Hub CRM

> Schlanker Quick-Reference. Detailwissen liegt im Knowledge-Vault unter `.knowledge/` (Single Source of Truth) — siehe Pointer-Block am Ende.

## Projekt-Kontext

- **Mission:** All-in-One CRM fuer DACH-KMUs mit EU-Datensouveraenitaet
- **USP:** Massanfertigung durch 1-Woche-Onsite-Prozessanalyse + Config/WASM-Plugin-System
- **Zielgruppe:** Branchenunabhaengige KMUs (5-200 Mitarbeiter)
- **Team:** 1 Dev + 2 Business, AI-First Development
- **Branding:** Software="Cosmi", Firma="Zentria"
- **Ziel:** **Produkt 1.0.0** nach Reifegrad-Gates, **kein Kalenderdatum**. Definition und Sequenz: `.planning/launch-lagebild-2026-08-12.md` §3 und §6

## Tech-Stack

| Komponente | Technologie |
|-----------|-------------|
| Backend | Go — API-Gateway + gRPC-Microservices unter `backend/cmd/*` |
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
11. **Tenant-Modell** — Option-B-Full: alle Tabellen `tenant_id UUID NOT NULL` + Row-Level-Security. RLS **produktiv erzwungen**; App-Services laufen als `kmuhub_app` (NOSUPERUSER NOBYPASSRLS), DDL-Migrations als `kmuhub`. Daten aktuell Single-Tenant, Code Multi-Tenant-fähig. Neue Tabellen MUESSEN `tenant_id` + RLS-Policy haben — oder explizit in die System-Global-Liste (ADR-006, `docs/ARCHITECTURE.md`) eingetragen werden

**Entwicklungs-Kommandos** (Backend / Desktop / Docker): `[[architektur#entwicklungs-kommandos]]`

## UI/UX

Aesthetik: **"Premium SaaS mit Editorial Touch"** — kein generisches Dashboard-Look.

**Designphilosophie:** Cosmi-Identitaet bleibt eigen. Apple-Linse (Reduktion, Hierarchie, Daily-Use-Disziplin) fuer alles was 100x/Tag passiert. Discord-Linse (Waerme, Personality, Joy-Moments) fuer Empty-States, Onboarding, Success. **Keine visuellen Kopien** (kein Mac-Chrome, keine Discord-Sidebar, keine Mascots von draussen).

- **Font-Bans:** NIEMALS Inter, Roboto, Arial, Space Grotesk, Helvetica, Open Sans. Erlaubt: Plus Jakarta Sans, Clash Display, Satoshi, JetBrains Mono, Playfair Display
- **Anti-Patterns:** Card-in-Card, AI-Slop-Aesthetik (lila Gradienten), symmetrische Bootstrap-Grids, Ueber-Animation
- **Motion-Hardrule:** Nur `transform`/`opacity` animieren (GPU). Nie `width/height/margin/padding`. Tokens in `lib/motion.ts` + `styles/animations.css` — keine magic numbers in Komponenten.
- **Keine Emojis in UI** (Personality via Custom-SVG, Motion, Wording).
- **Skills:** `frontend-design`, `impeccable` (`/audit`, `/critique`, `/polish`, `/animate` etc.). On-demand: `emil-design-eng`, `design-motion-principles`.

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

Conventional Commits und "keine AI-Attribution" sind per Hook erzwungen
(`.claude/hooks/check-commit-message.sh`, `check-no-attribution.sh`) — nicht als Bitte hier.

- **Branch-Strategie:** **direct-to-main ist Default**. Keine Feature-Branches, keine PRs — ausser explizit gefordert
- **CI-Rot-Recovery:** `git revert <sha>`. **NIE** `reset --hard`, **NIE** Force-Push
- **Push-Rhythmus:** Am Ende jeder Session, um Divergenz zu vermeiden

## Modul-Arbeit: Build-+-Verify-Standard (verbindlich, fuer JEDE Phase)

Gilt fuer alle, die Frontend-Module bauen (Haupt-Team + delegierte Mitarbeit). Pro Phase IMMER dieselbe Schleife: bauen → i18n ×4 (`{var}`, nie `{{var}}`; Plural als ICU `{count, plural, …}`, nie `_one`/`_other`) → Demo-Handler falls noetig → **gescopter Typecheck** (nur geaenderte Dateien, nie Full-tsc als Gate) → **Playwright-Screenshot-QA + die Screenshots wirklich ansehen** (Raw-Keys/Emojis/Layout/leere Zustaende) → iterieren bis gruen → ein Commit + Push. „Kompiliert ja" oder „Script lief gruen" allein reicht NICHT — die Bilder muessen angeschaut werden.

**Exakter Prozess + kopierbare Vorlagen:** `.planning/nico-block/WORKFLOW.md`. Delegations-Paket (Runbook, Repo-Map, Pilot-Specs): `.planning/nico-block/`.

## Knowledge-Vault (.knowledge/)

Single Source of Truth fuer projektspezifisches Wissen. Notes haben YAML-Frontmatter, verlinken via `[[note-name]]`.
Wird **nicht** automatisch geladen — bei Bedarf mit Read/Grep nachschlagen.
Vollstaendige Notes-Liste und Einstieg: `.knowledge/_index.md`.

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
| Intel-System (Markt-Scan, eigenes Repo) | `github.com/Lukes-Git-Beginning/zentria-intel` |
| Was die Setup-Kur 2026-08 gestrichen hat | `docs/claude-config-archiv-2026-08.md` |

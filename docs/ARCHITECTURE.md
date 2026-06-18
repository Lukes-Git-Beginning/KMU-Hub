# Architecture Decision Records (ADRs)

---

## ADR-001: Electron statt Tauri fuer Desktop App

**Status:** Akzeptiert

**Kontext:**
Desktop-App benoetigt nativen Zugriff (Dateisystem, System Tray, Notifications) und soll CRM + Chat + Video in einer Anwendung vereinen.

**Optionen:**
1. **Electron** — Chromium + Node.js, groesstes Ecosystem
2. **Tauri** — Rust-basiert, kleiner, schneller, aber juengeres Ecosystem

**Entscheidung:** Electron

**Begruendung:**
- Deutlich groesseres Library-Ecosystem (insb. fuer Video/WebRTC)
- Mehr Dokumentation und Community-Support
- React-Integration ausgereift und gut dokumentiert
- Team hat bereits TypeScript/React Erfahrung
- Tauri's Rust-Requirement wuerde Onboarding erschweren
- Bundle-Groesse ist bei einem CRM weniger kritisch als bei Consumer-Apps

**Konsequenzen:**
- Groessere App-Groesse (~150MB vs ~10MB bei Tauri)
- Hoeherer RAM-Verbrauch
- Muss Chromium-Updates im Auge behalten (Security)

---

## ADR-002: Go statt Node.js/Python fuer Backend

**Status:** Akzeptiert

**Kontext:**
Backend muss Microservices, WebSocket-Connections (Chat), und API Gateway performant handeln. Team-Groesse: 1 Entwickler.

**Optionen:**
1. **Go** — Statisch typisiert, kompiliert, exzellente Concurrency
2. **Node.js** — JavaScript/TypeScript, Event-Loop, groesstes Ecosystem
3. **Python** — Bekannt vom Vorgaenger-Projekt, Flask/FastAPI

**Entscheidung:** Go

**Begruendung:**
- Natuerliche Concurrency (Goroutines) fuer Chat/WebSocket
- Kompiliert zu einzelnem Binary — einfaches Deployment
- Geringer Speicherverbrauch (wichtig fuer Self-Hosted bei Kunden)
- Statische Typisierung verhindert Laufzeitfehler
- Stdlib deckt HTTP, JSON, Crypto ab — wenige externe Dependencies
- LiveKit ist Go-basiert — gleiche Sprache fuer Plugins/Extensions
- Python's GIL waere Bottleneck fuer Concurrency

**Konsequenzen:**
- Steilere Lernkurve als Node.js
- Weniger Web-spezifische Libraries als Node.js
- Generics (seit Go 1.18) nutzen wo sinnvoll

---

## ADR-003: API Gateway Pattern

**Status:** Akzeptiert

**Kontext:**
Mehrere Backend-Services (CRM, Chat, Auth, Video) muessen ueber eine einheitliche API erreichbar sein.

**Entscheidung:** Eigener API Gateway Service

**Begruendung:**
- Zentrales Rate Limiting und Auth
- Request Routing zu internen Services
- API Versioning an einer Stelle
- Kein externer Gateway (Kong, Traefik) noetig fuer MVP
- Spaeter austauschbar gegen Production-Gateway

**Konsequenzen:**
- Zusaetzlicher Service zu maintainen
- Single Point of Failure (muss hochverfuegbar sein)
- Muss Health Checks fuer alle Backend-Services implementieren

---

## ADR-004: WASM Plugins statt Native Plugins

**Status:** Akzeptiert

**Kontext:**
Kunden brauchen individuelle Erweiterungen (Branchenlogik, Integrationen). System muss sicher und wartbar erweiterbar sein.

**Zwei-Stufen-System:**
1. **Config-basierte Anpassung** (80% der Faelle) — Custom Fields, Workflows, Validierungsregeln ueber JSON/YAML Config
2. **WASM Plugins** (20% der Faelle) — Komplexe Logik, externe Integrationen

**Begruendung fuer WASM:**
- Sandbox-Sicherheit: Plugin kann nicht auf Host-System zugreifen
- Sprachunabhaengig: Kunden/Partner koennen in Rust, Go, C, AssemblyScript etc. schreiben
- Deterministische Ausfuehrung: Gleicher Input = Gleicher Output
- Performance: Near-Native Speed
- Go hat gute WASM-Runtime Support (wazero — pure Go, keine CGO)

**Konsequenzen:**
- Plugin-API muss stabil sein (Breaking Changes betreffen alle Plugins)
- WASM-Runtime (wazero) als Dependency
- Plugin-Entwicklung ist komplexer als Scripting (z.B. Lua)
- Braucht gute Plugin-SDK Dokumentation

---

## ADR-005: LiveKit fuer Video/Audio

**Status:** Akzeptiert

**Kontext:**
CRM braucht integrierte Video-Calls (1:1 und Gruppen), Screen Sharing, und Recording.

**Optionen:**
1. **LiveKit** — Open Source, Go-basiert, self-hostable
2. **Jitsi** — Open Source, Java-basiert, bewahrt
3. **Twilio/Vonage** — Cloud-only, Pay-per-Use

**Entscheidung:** LiveKit

**Begruendung:**
- Go-basiert (passt zum Stack)
- Self-hostable (EU-Datensouveraenitaet!)
- Modernes SFU-Design (Selective Forwarding Unit)
- WebRTC-basiert, funktioniert in Electron (Desktop) und modernen Browsern (PWA, Phase E)
- Gute SDKs fuer alle Plattformen
- Recording und Egress built-in
- Cloud-Option verfuegbar als Fallback

**Konsequenzen:**
- LiveKit Server muss gehostet und gewartet werden
- TURN/STUN Server fuer NAT Traversal
- Bandbreiten-Kosten fuer Video

---

## ADR-006: PostgreSQL + Redis

**Status:** Akzeptiert

**Kontext:**
Datenbank muss relational (CRM-Daten), performant (Chat), und self-hostable sein.

**Entscheidung:** PostgreSQL als Primary Store + Redis als Cache/PubSub

**Begruendung PostgreSQL:**
- Bewaehrt fuer CRM-Daten (relationale Strukturen, JOINs, ACID)
- JSONB fuer flexible Custom Fields
- Full-Text Search built-in
- Self-hostable, keine Lizenzkosten
- Erfahrung aus Vorgaenger-Projekt

**Begruendung Redis:**
- Session Storage
- Cache-Layer fuer haeufige Queries
- PubSub fuer Realtime-Events (Chat-Notifications)
- Rate Limiting Counters

**Wichtig:** Redis ist NUR Cache, NICHT Source of Truth. Bei Redis-Ausfall muss das System weiter funktionieren (langsamer, aber korrekt).

**Learning aus Vorgaenger:**
- KEIN Dual-Write Pattern
- PostgreSQL ist die einzige Datenquelle
- Redis-Daten sind jederzeit aus PostgreSQL rekonstruierbar

## ADR-007: Finance-Line-Items-Normalisierung (JSONB → relationale Tabellen)

**Status:** Akzeptiert · Umgesetzt 2026-06-08 (Sprint 4, Migrationen 000132/000133)

**Volltext:** [`docs/adr/0007-finance-line-items-normalization.md`](adr/0007-finance-line-items-normalization.md)

**Kontext:** `finance_invoices.line_items` (und Quotes/Credit-Notes) lagen als JSONB-Array vor — nicht GoBD-/ZUGFeRD-revisionssicher, keine FKs, keine Constraints, N+1-anfaellig bei Aggregation.

**Entscheidung:** Relationaler Cutover auf eigene Tabellen `finance_invoice_lines` / `finance_quote_lines` / `finance_credit_note_lines` (FK CASCADE, RLS, `tax_rate`-CHECK 0–100 DACH-sicher, `locked_at`/`locked_by` auf `finance_invoices`). Sauberer Cutover ohne Dual-Write/Feature-Flag (keine Prod-Finance-Daten). Die JSONB-Spalte bleibt **synchron befuellt** → gRPC/PDF/DATEV/Dashboard unveraendert (kein API-Bruch, Proto war schon `repeated LineItem`). **JSONB-Drop deferred auf Sprint 5.**

**Konsequenz:** Finance-Test-Coverage via testcontainers-go (echtes PG16, `-tags=integration`): invoice 69.6 % · quote 63.7 % · creditnote 51.3 %. Schliesst R2-P1.12.

## System-Global Tables (No RLS)

Folgende Tabellen sind bewusst NICHT RLS-aktiviert. Sie sind system-globaler Natur (Schema-Metadata, kontext-unabhaengige Konfiguration, Seed-Daten). Alle anderen Tabellen sind ab Sprint 4 Welle 4 RLS-pflichtig.

| Tabelle | Grund |
|---|---|
| `schema_migrations` | golang-migrate State-Tabelle, kein Tenant-Kontext |
| `caldav_settings` | Server-globale CalDAV-Defaults, key-value-Store ohne Tenant-Bezug |
| `industry_templates` | System-Seed-Daten (Branchen-Templates), shared read-only |

Jede neue Tabelle in `backend/migrations/` muss entweder `tenant_id UUID NOT NULL` + RLS-Policy haben oder explizit hier eingetragen werden mit Begruendung.

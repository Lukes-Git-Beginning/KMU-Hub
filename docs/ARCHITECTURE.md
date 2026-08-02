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

## ADR-008: Orbit Appliance-Architektur — Mini-PC + Standard-Linux, kein eigenes OS

**Status:** Akzeptiert (Richtungsentscheidung 2026-06-21, Luke + Darien) · Umsetzung Phase E / Q4 2026 · ⚠ Integrator-/CRA-Rechtslage durch Produktrecht-Anwalt zu bestaetigen VOR Bau

**Kontext:**
Orbit ist die Self-Hosted-Variante von Cosmi: dieselbe Software auf Hardware, die Zentria physisch liefert, einrichtet und remote wartet (Updates/Backup/Monitoring) — Managed-Appliance, kein "bring your own server". Zielgruppe: datenschutz-sensible Branchen (Aerzte, Anwaelte, Steuerberater, Behoerden) + Betriebe mit schlechter Konnektivitaet / No-Cloud-Wunsch. Orbit existiert im Code noch nicht (Phase E).

Ausloeser war der Wunsch nach einem "eigenen System". Geklaert: gemeint ist ein produkt-eigenes, gebrandetes Appliance-Erlebnis — kein OS von Grund auf. Recherche (2026-06-21) ergab: der urspruenglich geplante Synology-NAS-Pfad (DSM umhuellen) bringt vermeidbare Last (Docker-Engine 24.0.2 EOL auf DSM, Port-80/443-Patch, "DSM verstecken", Synology-Plattformrisiko via Drive-Lock-Praezedenz). Ein neutraler Mini-PC mit nacktem Standard-Linux loest diese Probleme und schafft Paritaet mit der Hetzner-Cloud.

**Optionen:**
1. Synology-NAS + DSM umhuellen (Reverse-Proxy davor)
2. Neutraler Mini-PC + Standard-Linux (Ubuntu Server) + Docker
3. Eigenes OS / eigene Distro / Appliance-Image (Sisyphus, fuer 1-Dev nicht tragbar)

**Entscheidung:** Option 2.

**Grundsatz — Integrator, kein Hardware-Hersteller:** Fertige CE-zertifizierte Geraete kaufen + vom Hersteller vorgesehenes RAM/SSD einsetzen + unsere Software. Branding ausschliesslich in Software/Login/Domain/Verpackung — nichts aufs Gehaeuse, kein Eigenbau aus Losteilen (sonst eigene CE-Messung, ElektroG/stiftung-ear, Produkthaftung). ⚠ Die CRA-Security-Update-Pflicht trifft die Software Cosmi unabhaengig vom Hardware-Status.

**Kern-Entscheidungen:**
- **Hardware:** neutraler Mini-PC (Pod/Station), echter Tower/Rack-Server fuer Command-Tier (80–200+ User). Auswahlkriterien: 2× NVMe (RAID-1), 24/7-Eignung, ECC (Station/Command), bestaetigte CE. Exaktes Modell nach Ressourcen-Spike.
- **Plattform-Paritaet Cloud↔Orbit:** identisches Linux+Docker-Setup → eine Deploy-Pipeline (private Registry + `docker compose pull`), ein Betriebsmodell, eine Test-Matrix.
- **Schlankes Orbit-Profil je Tier** (Feature-Flags + Compose-Overlays); OnlyOffice/LiveKit-Egress auf Pod aus; Video gestaffelt (Pod=SFU-light/Cloud, Recording ab Station).
- **Branding/UX:** Orbit-Console als gebrandetes Admin-Modul IN Cosmi (+ Host-Agent fuer System-Daten); Zero-Touch-Provisionierung; zwei Onboardings (Geraet + Modul); Domain pro Kunde via Split-DNS + Let's-Encrypt DNS-01; transparenter/automatischer VPN-Zugang.
- **Betrieb:** Remote-Shell via Headscale + Tailscale, Fleet via Portainer Business + Edge (beide self-hosted auf Hetzner); Monitoring lokal schlank + Remote-Aggregation (Opt-in); Backup lokal + verschluesseltes Cloud-Offsite + Restore-Test-Pflicht.
- **Security/Kommerz:** volle Haertung (LUKS-Disk-Encryption, Firewall, offizielle Image-Quellen); DSGVO/AVV + §203-StGB-/§43e-BRAO-Vertraege + TOMs; Offline-Ed25519-Lizenz (kein Default-Phone-Home); Erloes ueber Recurring (Kauf + optionales Service-Abo, **kritische Security-Patches auch ohne Abo**, Feature-Updates im Abo); offener Standard-Export (pg_dump + MinIO-mirror, kein Lock-in).

**Begruendung:**
- Ein Codepfad/Pipeline/Betriebsmodell fuer ein 1-Dev-Team (groesster Hebel).
- Eliminiert DSM-spezifische Risiken (Engine-EOL, Port-Patch-Fragilitaet) und Synology-Plattformabhaengigkeit (Drive-Lock-Praezedenz, Modell-Lifecycle).
- Hardware-agnostisches Image → Modellwechsel/RMA trivial (Ersatzgeraet in Minuten neu bespielt).
- Integrator-Linie haelt CE-/Produkthaftungs-Last gering.

**Konsequenzen:**
- "DSM unsichtbar"/Port-Patch/Caddy-Trick entfallen; Disk-Encryption via LUKS.
- Neue Zentria-Infra: private Registry, Headscale-Control, Portainer-Server, zentrales Monitoring, CI-Multi-Image-Pipeline, Lizenz-Signing (alles EU/Hetzner).
- Registry-Pipeline ersetzt "Build-on-Host" auch in der Cloud (Nebengewinn).
- Storage-Redundanz NICHT geschenkt (kein NAS-RAID) → selbst via RAID-1 + verschluesseltem Offsite-Backup.
- Command-Tier ggf. echter Tower/Rack-Server statt Mini-PC.

**Offene Spikes vor Bau:** (1) Ressourcen-Spike als HARTES Gate (ausgewachsener Stack auf Kandidaten-Mini-PC → Modell/RAM fix), (2) Produktrecht-Anwalt (Integrator + CRA + AVV/§203/§43e), (3) Registry/CI-Design, (4) Headscale/Portainer-Konsolidierung, (5) RAID-1/LUKS + DNS-01 Proof.

**Verweis:** Roadmap (Spikes + Epics + Ownership) in `docs/ROADMAP.md` §Phase E.

---

## System-Global Tables (No RLS)

Folgende Tabellen sind bewusst NICHT RLS-aktiviert. Sie sind system-globaler Natur (Schema-Metadata, kontext-unabhaengige Konfiguration, Seed-Daten). Alle anderen Tabellen sind ab Sprint 4 Welle 4 RLS-pflichtig.

| Tabelle | Grund |
|---|---|
| `schema_migrations` | golang-migrate State-Tabelle, kein Tenant-Kontext |
| `caldav_settings` | Server-globale CalDAV-Defaults, key-value-Store ohne Tenant-Bezug |
| `industry_templates` | System-Seed-Daten (Branchen-Templates), shared read-only |
| `permissions` | Capability-Katalog, reine Seed-Daten ohne Tenant-Bezug (jeder Tenant kennt dieselben Keys) |
| `automation_templates` | Katalog vorgefertigter Automations-Vorlagen, ausschliesslich von `TemplateRegistry.SeedToDatabase()` beim Start geschrieben (`internal/automation/template/registry.go`) — keine HTTP-Route schreibt hierher |
| `event_types` | Katalog moeglicher Benachrichtigungs-Event-Typen, ausschliesslich per Migrations-Seed gefuellt (000027, 000048) — kein Laufzeit-Schreibpfad existiert |
| `public_holidays` | Feiertagskalender, befuellt durch `HolidayService.SeedHolidays()` aus der externen Nager.Date-API (`POST /holidays/seed`, `calendars:admin`-gescopt). Der Inhalt ist deterministische, laenderspezifische Realdaten (Upsert auf `date, country_code, name`) — jeder Tenant, der den Sync ausloest, schreibt dieselben Zeilen, keine Tenant-Divergenz moeglich |

Jede neue Tabelle in `backend/migrations/` muss entweder `tenant_id UUID NOT NULL` + RLS-Policy haben oder explizit hier eingetragen werden mit Begruendung.

**Sonderfall `roles` / `role_permissions` (seit Migration 000256):** RLS-aktiviert, aber `tenant_id` ist NULLable — NULL bedeutet "System-Preset, fuer jeden Tenant sichtbar". Deshalb greift hier NICHT `enable_tenant_rls()`, sondern ein Paar aus Lese- und Schreib-Policy: Lesen sieht `tenant_id IS NULL OR tenant_id = current_tenant_id()`, Schreiben nur den eigenen Tenant. Die Trennung ist noetig, weil DELETE ausschliesslich die USING-Klausel auswertet — eine einzelne permissive Policy wuerde einem Tenant erlauben, die Presets zu loeschen.

**Nicht mehr system-global — `events` (seit Migration 000271):** Die Event-Bus-Durability-Tabelle galt seit 000021 als system-global und wurde in 000242 mit dieser Begruendung partitioniert. Die Praemisse trug nicht: `models.EventPayload` fuehrt eine `TenantID`, jeder Emitter fuellt sie, und der Schreibpfad verwarf sie nur. Dadurch konnte `EventBus.ProcessBacklog` den Tenant nach einem Neustart nicht wiederherstellen — jedes nachgeholte Event erreichte die Handler mit `uuid.Nil`. 000271 ergaenzt `tenant_id NOT NULL` (Backfill ueber `actor_id` -> `users.tenant_id`, Rest nur auf einer Single-Tenant-DB zuordenbar) und `enable_tenant_rls('events')`. Die beiden tenant-uebergreifenden Worker-Pfade (Catch-up-Read, `processed`-Flag) laufen bewusst als System-Kontext; der Schreibpfad nicht, damit die Policy den Insert wirklich prueft.

**Offene Luecke:** `user_roles` hat weder `tenant_id` noch RLS. Heute ist das ungefaehrlich, weil alle Lesepfade ueber eine `user_id` aus dem JWT filtern; ein direkter, ungefilterter SELECT waere aber tenant-uebergreifend. Backlog-Unit `g-user-roles-rls`.

**Geschlossen — `refresh_tokens` und `plugin_permissions` (seit Migration 000272):** Teil des Allowlist-Audits (Backlog-Unit `g-rls-allowlist-audit`). `refresh_tokens` bekam `tenant_id NOT NULL` (Backfill ueber `user_id` -> `users.tenant_id`, FK garantiert keine Waisen) + `enable_tenant_rls()` — die vorbestehenden `sysctx.With()`-Aufrufe in `RefreshToken`/`Logout` (auth/service.go) decken den Pre-JWT-Lesepfad bereits ab, analog zu `password_reset_tokens`. `plugin_permissions` bekam `enable_tenant_rls_via_join()` ueber `installation_id` -> `plugin_installations` (dieselbe Form wie die CRM-Custom-Field-Value-Tabellen aus 000270), da die Tabelle selbst kein `tenant_id` traegt.

**Offener Befund — tenant-uebergreifende Schreibpfade unter einem tenant-gescopten Guard:** fuenf Tabellen ohne `tenant_id` werden ueber eine mit `RequireRole("admin")`/`RequirePermission(..., "write")` geschuetzte Route geschrieben — der Guard prueft nur "ist Admin/hat Recht", nicht "in welchem Tenant". Da keine der Tabellen `tenant_id` fuehrt, kann JEDER Tenant-Admin den einzigen globalen Datensatz fuer ALLE Tenants ueberschreiben:
- `dashboard_defaults` — `PUT /admin/dashboard/defaults/{role}`, `UNIQUE(role)`, ueberschreibt das Rollen-Default-Layout platformweit.
- `presence_config` — `PUT /presence/config` (`settings:write`), `UPDATE ... ` ohne WHERE auf der einzigen Zeile.
- `two_factor_policy` — `PUT /2fa/policy` (`RequireRole("admin")`), `UNIQUE(role_name)` — ein Tenant-Admin kann die 2FA-Pflicht fuer die `admin`-Rolle aller Tenants abschalten.
- `storage_quotas` — kein HTTP-Guard direkt, aber `IncrementUsedBytes`/`DecrementUsedBytes` ohne WHERE auf der einzigen Zeile — jeder Tenant zaehlt gegen dasselbe globale `max_bytes`-Kontingent.
- `plugin_manifests` — `POST /api/v1/plugins/manifests` (`RequireRole("admin")`), kein `tenant_id`, `GET /manifests` ohne Filter — ein `plugin_type=config`-Manifest, das ein Tenant anlegt, ist fuer alle Tenants sichtbar (WASM-Manifeste sind durch den `plugins.wasm`-Flag-Check gesondert blockiert, Config-Manifeste nicht).

Das ist kein RLS-Luecken-Fall im Sinne dieses Abschnitts (RLS wuerde ohne `tenant_id` nichts trennen) und keine reine Katalog-/Seed-Tabelle (der Inhalt IST admin-mutable und divergiert potenziell pro Tenant) — beide Antworten dieses Abschnitts passen nicht. Die ehrliche Loesung braucht `tenant_id` + Backfill + ein Provisioning-Schritt fuer neue Tenants (aktuell holt `GetConfig`/`GetDefaults`/`GetTwoFactorPolicy` genau eine globale Zeile per `LIMIT 1` bzw. `UNIQUE`-Lookup — ohne eigene Zeile pro Tenant faellt der Read nach RLS auf `ErrNoRows`). Das ist groesser als eine Iteration und bewusst nicht in 000272 mit hineingezogen. Backlog-Unit `g-rls-tenant-scoped-admin-writes`.

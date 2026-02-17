# 11 — Backend-Implementierungsplan Teil 1: Architektur, Features, Migrationen, Sicherheit

**Datum:** 2026-02-17
**Grundlage:** `00-SYNTHESE.md`, `08-datenbankmodelle.md`, `13-vision-ergaenzungen.md`, Luke's `ROADMAP.md`, `CLAUDE.md`
**Zielgruppe:** Luke (Backend-Dev), Darien (Design-Kontext), Nico (Business-Review)

---

## 1. Aktuelle Architektur-Uebersicht

### 1.1 Backend-Services (6 Binaerdateien)

| Service | Binary | Port | Proto | Packages (internal/) |
|---------|--------|------|-------|---------------------|
| **API Gateway** | `cmd/gateway` | 8080 | HTTP | `gateway/` (registry, route_auth, route_crm, route_chat, route_notification, route_work, route_calendar, route_dashboard, route_health) |
| **Auth** | `cmd/auth` | gRPC | `auth/v1` | `auth/` (service, repository, token) |
| **CRM** | `cmd/crm` | gRPC | `crm/v1` | `crm/contact`, `crm/company`, `crm/deal`, `crm/activity`, `crm/pipelinestage`, `crm/tag`, `crm/customfield`, `crm/savedfilter`, `crm/search`, `crm/report` |
| **Chat** | `cmd/chat` | gRPC | `chat/v1` | `chat/channel`, `chat/message`, `chat/file`, `chat/search`, `chat/langdetect` |
| **Notification** | `cmd/notification` | gRPC | `notification/v1` | `notification/notification`, `notification/preference`, `notification/delivery`, `notification/event` |
| **Work** | `cmd/work` | gRPC | `work/v1` + `calendar/v1` | `work/project`, `work/task`, `work/comment`, `work/status`, `work/timeentry`, `work/calendar`, `work/event`, `work/resource`, `work/holiday`, `work/livekit` |

### 1.2 Shared Packages

| Package | Zweck |
|---------|-------|
| `internal/database/` | PostgreSQL + Redis Connection-Factory |
| `internal/middleware/` | Auth-Middleware (JWT), CORS, Rate Limiting, Logging |
| `internal/health/` | Health-Check Endpoints |
| `internal/metrics/` | Prometheus-Metriken |
| `internal/models/` | Shared Domain-Modelle |

### 1.3 Kommunikationsmuster

```
Browser/Electron
      |
      v  (HTTP/WebSocket)
  [API Gateway :8080]
      |
      v  (gRPC, lazy connections via ServiceRegistry)
  +-------+------+---------+----------+-------+
  | Auth  | CRM  |  Chat   | Notif.   | Work  |
  +-------+------+---------+----------+-------+
      |      |       |          |         |
      v      v       v          v         v
  [PostgreSQL]    [Redis]    [MinIO]  [LiveKit]
```

- Gateway -> Services: **gRPC** (lazy, pro Route-Handler)
- Services -> Notifications: **PostgreSQL LISTEN/NOTIFY** Event-Bus
- Services -> Services: **kein direkter Aufruf** (Graceful Degradation)
- Realtime: Gateway haelt **WebSocket-Hub**, pushed Notifications an Clients

### 1.4 Bestehende Migrationen (000001-000035)

| Nr. | Name | Tabellen |
|-----|------|----------|
| 000001 | `create_users_table` | `users` |
| 000002 | `create_roles_and_permissions` | `roles`, `permissions`, `role_permissions`, `user_roles` |
| 000003 | `create_refresh_tokens` | `refresh_tokens` |
| 000004 | `create_invitations` | `invitations` |
| 000005 | `create_custom_field_definitions` | `custom_field_definitions` |
| 000006 | `create_tags` | `tags`, Junction-Tables |
| 000007 | `create_contacts_companies` | `contacts`, `companies`, `*_custom_field_values` |
| 000008 | `create_pipeline_stages` | `pipeline_stages` |
| 000009 | `create_deals` | `deals`, `deal_custom_field_values` |
| 000010 | `create_activities` | `activities` |
| 000011 | `add_fulltext_search` | FTS `search_vector` |
| 000012 | `create_saved_filters` | `saved_filters` |
| 000013 | `create_deal_stage_history` | `deal_stage_history` |
| 000014 | `create_chat_channels` | `chat_channels`, `channel_members` |
| 000015 | `create_chat_messages` | `chat_messages` |
| 000016 | `add_dm_and_threads` | DM + Thread-Erweiterungen |
| 000017 | `add_mentions` | `mentions` |
| 000018 | `create_chat_files` | `chat_files`, `storage_quotas` |
| 000019 | `add_chat_search` | Chat-FTS |
| 000020 | `create_event_types` | `event_types` |
| 000021 | `create_notifications` | `notifications` |
| 000022 | `create_notification_preferences` | `notification_preferences` |
| 000023 | `create_dashboard_layouts` | `dashboard_layouts` |
| 000024 | `create_projects` | `projects`, `project_members`, `project_statuses` |
| 000025 | `create_tasks` | `tasks` |
| 000026 | `create_task_collaboration` | Kommentare, Dateien, Abhaengigkeiten |
| 000027 | `seed_work_event_types` | Seed-Daten |
| 000028 | `add_entity_display_name` | Darstellungsformat |
| 000029 | `add_notification_permissions` | RBAC fuer Notifications |
| 000030 | `create_time_entries` | `time_entries` |
| 000031 | `add_gantt_view_type` | View-Erweiterung |
| 000032 | `create_calendars` | `calendars`, `calendar_members`, `event_categories` |
| 000033 | `create_events` | `calendar_events`, `event_attendees`, `event_exceptions`, `event_reminders`, `user_calendar_preferences` |
| 000034 | `create_resources` | `resources` |
| 000035 | `create_holidays` | `holidays` |

**Wichtig:** Kein `tenant_id` in bestehenden Tabellen. Aktuell single-tenant.

---

## 2. Neue Features — Priorisierte Reihenfolge

### Abgleich: Luke's Roadmap vs. Research-Ergebnisse

Luke's Roadmap deckt Phasen 8-20 ab. Die Research-Dokumente (00-SYNTHESE, 13-vision-ergaenzungen) identifizieren Features die entweder **gar nicht in der Roadmap sind** oder **tiefer gehen als Luke geplant hat**.

### 2.1 Features die Luke BEREITS geplant hat

| Feature | Luke's Phase | Research-Ergaenzung |
|---------|-------------|---------------------|
| LiveKit Video/Voice | Phase 8 | Zoom-Fallback fuer Starter-Tier (NEU) |
| 2FA + Audit-Log | Phase 9 | Consent-Mgmt, Retention-Policies (TIEFER) |
| IMAP/SMTP E-Mail | Phase 10 | CRM-Auto-Linking, Threading (GLEICH) |
| Dokumente + Files | Phase 11 | OnlyOffice WOPI, File-Sharing-Links (TIEFER) |
| Finance (GoBD, DATEV) | Phase 12 | Belegkette, QR-Rechnung, ZUGFeRD (TIEFER) |
| HR (Urlaub, Zeit) | Phase 13 | GLEICH |
| CalDAV/CardDAV | Phase 14 | GLEICH |
| Teams/Slack Bridge | Phase 15 | Unified Inbox-Konzept (TIEFER) |
| Bexio-Integration | Phase 16 | GLEICH |
| Automation Engine | Phase 19 | GLEICH |
| Plugin System | Phase 20 | GLEICH |

### 2.2 Features die NEU sind (nicht in Luke's Roadmap)

Sortiert nach Business Impact. Fuer jedes Feature: Beschreibung, betroffener Service, geschaetzter Aufwand.

---

#### P1: Unified Inbox / Omnichannel (KRITISCH)

**Was:** Alle externen Kommunikationskanaele (E-Mail, Teams, WhatsApp Business, Website-Chat-Widget, Kundenportal) in EINER Inbox. Automatische CRM-Kontakt-Zuordnung, Kontext-Panel mit offenen Deals/Tickets.

**Service:** NEUER Service `cmd/inbox` oder Erweiterung von `cmd/chat`
- Unified Message Model (channel, direction, contact_id, content, metadata)
- Channel-Adapter-Interface pro Kanal
- WebSocket fuer Echtzeit
- Queue fuer ausgehende Nachrichten (Retry, Rate Limiting)
- OAuth2-Flows fuer Teams/Slack, WhatsApp Business API Webhook-Handler

**Aufwand:** 8-12 Wochen (alle Kanaele), davon:
- E-Mail-Kanal: kommt mit Phase 10 (bereits geplant)
- Website-Chat-Widget: 2-3 Wochen (JWT-Token-Auth, embeddable JS-Snippet)
- WhatsApp Business: 2-3 Wochen (Meta Cloud API)
- Teams Bridge: 3-4 Wochen (Graph API, Bot Framework)
- Kundenportal: 3-4 Wochen (eigener Auth-Flow, eingeschraenkte Ansicht)

**Empfehlung:** Widget + WhatsApp nach Phase 10 (E-Mail). Teams nach Phase 15 integrieren.

---

#### P2: OnlyOffice WOPI-Integration

**Was:** .docx/.xlsx/.pptx direkt in KMU Hub bearbeiten. OnlyOffice Document Server als Docker-Container. WOPI-Protokoll-Endpoints in Go. Vorlagen-System mit CRM-Platzhaltern.

**Service:** Erweiterung von Phase 11 (Documents) oder eigenes Package `internal/documents/wopi`
- WOPI-Discovery-Endpoint
- `CheckFileInfo`, `GetFile`, `PutFile` Endpoints
- Lock-Management (Co-Editing)
- Vorlagen-Storage in MinIO, Template-Engine fuer Platzhalter

**Aufwand:** 2-4 Wochen

**Empfehlung:** In Phase 11 integrieren, nicht separat.

---

#### P3: Belegkette (Angebot -> Auftrag -> Lieferschein -> Rechnung -> Gutschrift)

**Was:** State-Machine fuer Belegtypen. Jeder Beleg kann aus dem vorherigen erzeugt werden. Lueckenlose Nummerierung. GoBD-konforme Unveraenderbarkeit nach Versand.

**Service:** NEUER Service `cmd/biz` (Phase 12 geplant) oder Package `internal/biz/document`
- `document_chain_items` mit `derived_from_id`
- `document_line_items` mit Steuerberechnung
- `number_sequences` fuer Nummernkreise
- `payments` fuer Teilzahlungen
- `dunnings` fuer Mahnwesen (3 Stufen)
- GoBD-Trigger (UPDATE/DELETE verhindert nach Lock)

**Aufwand:** 3-4 Wochen (Kernlogik), +2 Wochen (PDF-Generierung, Mahnwesen)

**Empfehlung:** IST Phase 12, muss aber tiefer gehen als Luke geplant hat. Belegkette + GoBD-Trigger sind essentiell.

---

#### P4: QR-Rechnung (CH) + ZUGFeRD/XRechnung (DE)

**Was:**
- Swiss QR-Code auf Rechnungen (seit 2022 PFLICHT in CH)
- ZUGFeRD: PDF/A-3 mit eingebettetem XML (EN 16931, ab 2027 Versand-Pflicht DE B2B)

**Service:** Package in `internal/biz/invoice` oder `internal/biz/qr`
- Go-Library fuer Swiss QR-Code (SIX-Spezifikation)
- Go-Library `invopop/gobl` fuer ZUGFeRD-XML
- PDF/A-3-Generierung mit eingebettetem XML

**Aufwand:** QR-Rechnung: 1-2 Wochen. ZUGFeRD: 2-3 Wochen.

**Empfehlung:** QR-Rechnung sofort mit Phase 12. ZUGFeRD kann 2-3 Monate spaeter kommen (erst 2027 Pflicht).

---

#### P5: Custom Fields Erweiterung (ueber CRM hinaus)

**Was:** Custom Fields existieren bereits fuer `contact/company/deal/activity` (Migration 000005). Muessen erweitert werden auf: `ticket`, `project`, `task`, `invoice`, `quote`, `order`, `vehicle`, `article`, `contract`, `rental_object`, `field_report`, `form`.

**Service:** Erweiterung von `internal/crm/customfield`
- CHECK-Constraint auf `entity_type` erweitern
- Neue `*_custom_field_values` Junction-Tables
- API-Endpoints pro neuer Entity

**Aufwand:** 1-2 Wochen (DB + API), Frontend pro Modul je 2-3 Tage

**Empfehlung:** Migration 000037. Kann frueh gemacht werden (keine grosse Abhaengigkeit).

---

#### P6: Firma als eigene Entity (Hierarchie + Kontaktrollen)

**Was:** `companies`-Tabelle erweitern: `parent_company_id` (Hierarchie), N:M `contact_company_roles` (ein Kontakt kann bei mehreren Firmen in verschiedenen Rollen sein). Zusatzfelder: USt-IdNr, Handelsregister, Rechtsform, Branche.

**Service:** Erweiterung von `internal/crm/company`
- Migration 000038: ALTER TABLE + neue Junction-Table
- API: Rollen-CRUD, Hierarchie-Navigation
- Kontakte: Akadem. Titel, Anrede (Sie/Du), bevorzugte Sprache

**Aufwand:** 2-3 Wochen

**Empfehlung:** Vor Phase 10 (E-Mail), weil E-Mail-zu-Kontakt-Zuordnung bessere Kontaktdaten braucht.

---

#### P7: Duplikaterkennung (CRM)

**Was:** Automatische Erkennung von Duplikaten bei Kontakten und Firmen. Matching per Name (Fuzzy), E-Mail (Exact), Telefon (Partial). Merge-Workflow mit Feld-Auswahl.

**Service:** Neues Package `internal/crm/duplicate`
- `duplicate_candidates` Tabelle
- `merge_history` (Snapshot + Feld-Entscheidungen)
- pg_trgm-Extension fuer Fuzzy-Matching
- Batch-Job fuer periodischen Scan, Plus On-Create-Check

**Aufwand:** 1-2 Wochen

**Empfehlung:** Nach Phase 10 (E-Mail) oder parallel, wenn CRM-Daten wachsen.

---

#### P8: Canned Responses + Private Notes (Helpdesk)

**Was:** Textbausteine mit Platzhaltern (`{{kunde_name}}`) und Shortcut-Codes (`/danke`). Interne Notizen auf Tickets/Deals/Kontakten die der Kunde nicht sieht.

**Service:** Neues Package `internal/helpdesk/cannedresponse` + `internal/shared/internalnote`
- `canned_responses` mit FTS-Suche
- `internal_notes` (polymorphe Verknuepfung: entity_type + entity_id)

**Aufwand:** 3-5 Tage (sehr einfach, CRUD + FTS)

**Empfehlung:** Sofort machbar, niedriger Aufwand, hoher Nutzen. Kann jederzeit eingebaut werden.

---

#### P9: E-Signatur (Skribble)

**Was:** Vertraege digital signieren ueber Skribble (Schweizer Firma, ZertES + eIDAS). REST-API-Integration. EES/FES/QES-Stufen.

**Service:** Package `internal/integrations/skribble`
- OAuth2-Flow
- Dokument hochladen, Signatur-Request erstellen
- Webhook fuer Signatur-Status-Updates
- Verknuepfung mit Vertraege-Modul

**Aufwand:** 2-3 Wochen

**Empfehlung:** Nach Phase 12 (Finance), passt gut ins Vertraege-Modul.

---

#### P10: Newsletter (Brevo / CleverReach)

**Was:** Kontaktlisten an Newsletter-Provider syncen. Kampagnen erstellen/senden. Opt-in/Opt-out-Status per CRM-Kontakt tracken.

**Service:** Package `internal/integrations/newsletter`
- Brevo REST-API (EU-Firma, Transactional + Marketing)
- CleverReach REST-API v3 (Deutsche Firma)
- Sync-Engine: CRM-Kontakte -> Newsletter-Listen
- Consent-Verknuepfung (DSGVO Art. 6/7)

**Aufwand:** 2-3 Wochen pro Provider

**Empfehlung:** Nach Phase 16 (Bexio), da Integrations-Infrastruktur dann steht.

---

#### P11: Banking (FinAPI)

**Was:** Automatischer Bankabgleich fuer 4.000+ Banken in DACH. Zahlungseingaenge automatisch Rechnungen zuordnen.

**Service:** Package `internal/integrations/finapi`
- PSD2-kompatible FinAPI REST-API
- Konten verknuepfen, Transaktionen abrufen
- Matching-Engine: Verwendungszweck -> Rechnungsnummer
- Dashboard-Widget fuer Kontostatus

**Aufwand:** 3-4 Wochen. FinAPI-Account ab ~500 EUR/Mo.

**Empfehlung:** Phase 2 post-launch (teuer, geringere Prioritaet als Kernfeatures).

---

#### P12: KI-Features

**Was:**
- Meeting-Zusammenfassungen, Ticket-Verlaeufe, CRM-Aktivitaeten
- E-Mail-/Ticket-Antwort-Entwuerfe, Angebotstexte
- Semantische Suche ueber Wiki/Docs/Tickets/CRM
- Auto-Klassifizierung (oeffentlich/intern/vertraulich)
- KI-Governance: Opt-out pro Modul, kein Training auf Kundendaten, Logging

**Service:** Eigenes Package `internal/ai` oder separater Service
- OpenAI/Anthropic API-Adapter (abstrahiert, Provider-unabhaengig)
- Embedding-Generierung fuer semantische Suche (pgvector-Extension)
- Rate-Limiting pro Tenant
- Opt-out-Flags pro Modul in Tenant-Settings

**Aufwand:** 4-8 Wochen (je nach Scope)

**Empfehlung:** Nach Phase 11 (Dokumente/Suche), da semantische Suche auf bestehender Suchinfrastruktur aufbaut.

---

#### P13: Zoom-Fallback fuer Starter-Tier

**Was:** Gestufte Video-Strategie: Starter = Zoom/Google Meet Links, Business = LiveKit, Enterprise = LiveKit + Recording.

**Service:** Erweiterung von Phase 8 (Video), Package `internal/work/livekit` erweitern zu `internal/work/video`
- VideoProvider-Interface (Zoom, Google Meet, LiveKit)
- Zoom OAuth2 + Meeting-Create API
- Google Calendar API fuer Meet-Links

**Aufwand:** 2-3 Wochen (Zoom), 1-2 Wochen (Google Meet)

**Empfehlung:** Phase 8 mit dem Interface bauen, Zoom/Google Meet spaeter nachruestbar.

---

### 2.3 Priorisierte Umsetzungsreihenfolge (Feature-Ergaenzungen)

Features die **nicht** in Luke's Phasen 8-20 stehen, sortiert nach Einbau-Zeitpunkt:

| Prioritaet | Feature | Einbau nach Phase | Aufwand | Begruendung |
|-----------|---------|-------------------|---------|-------------|
| **KRITISCH** | Firma als eigene Entity | vor Phase 10 | 2-3 Wo | E-Mail braucht saubere Kontaktdaten |
| **KRITISCH** | Canned Responses + Private Notes | jederzeit | 3-5 Tage | Trivialer Aufwand, sofortiger Nutzen |
| **KRITISCH** | Custom Fields Erweiterung | jederzeit | 1-2 Wo | Blockiert Modul-Anpassungen |
| **HOCH** | Belegkette (tiefer als Phase 12) | mit Phase 12 | 5-6 Wo | Phase 12 MUSS Belegkette beinhalten |
| **HOCH** | QR-Rechnung (CH) | mit Phase 12 | 1-2 Wo | Ohne das kein Schweizer Kunde |
| **HOCH** | OnlyOffice WOPI | mit Phase 11 | 2-4 Wo | "Dann brauche ich trotzdem Office" |
| **HOCH** | Unified Inbox (Widget + WhatsApp) | nach Phase 10 | 4-6 Wo | Differentiator, nach E-Mail logisch |
| **MITTEL** | Duplikaterkennung | nach Phase 10 | 1-2 Wo | Relevant wenn Daten wachsen |
| **MITTEL** | E-Signatur (Skribble) | nach Phase 12 | 2-3 Wo | Vertraege-Modul ergaenzen |
| **MITTEL** | ZUGFeRD/XRechnung | 2-3 Mo nach Phase 12 | 2-3 Wo | Erst 2027 Versand-Pflicht |
| **MITTEL** | Newsletter (Brevo/CleverReach) | nach Phase 16 | 2-3 Wo | Integrations-Infra muss stehen |
| **MITTEL** | KI-Features (Basis) | nach Phase 11 | 4-8 Wo | Semantische Suche auf bestehendem Index |
| **NIEDRIG** | Banking (FinAPI) | post-launch | 3-4 Wo | Teuer (500+/Mo), Nice-to-have |
| **NIEDRIG** | Zoom-Fallback | nach Phase 8 | 2-3 Wo | Interface in Phase 8, Impl spaeter |

---

## 3. Datenbank-Migrationen

### 3.1 Migrationsplan (000036-000050)

Basierend auf `08-datenbankmodelle.md`. Abhaengigkeiten beachtet.

```
000036  tenants + tenant_members                              [optional, Luke entscheidet]
000037  Custom Fields erweitern (entity_types, neue CFV-Tables) [keine Abhaengigkeit]
000038  Companies erweitern + contact_company_roles + Kontaktfelder [abh: companies, contacts]
000039  audit_entries                                          [abh: users]
000040  tax_rates (MWSt multi-country DACH)                    [keine Abhaengigkeit]
000041  Belegkette (number_sequences, document_chain_items,    [abh: contacts, companies, users,
        document_line_items, payments, dunnings)                      tax_rates]
000042  GoBD-Trigger (lock_check, delete_check, line_item_check) [abh: 000041]
000043  E-Mail (email_accounts, email_folders,                 [abh: users, contacts, companies,
        email_messages, email_attachments)                            deals]
000044  Helpdesk (canned_responses, internal_notes)            [abh: users]
000045  Duplikaterkennung (duplicate_candidates, merge_history) [abh: users]
000046  File-Sharing (shared_links, shared_link_access_log)    [abh: users]
000047  Consent-Management (consent_purposes, consents)        [abh: contacts, users]
000048  Retention-Policies (retention_rules)                   [keine Abhaengigkeit]
000049  DATEV-Export (datev_export_batches, datev_export_entries) [abh: users]
000050  Integration-Configs (integration_connections,          [abh: users]
        integration_sync_log, integration_entity_mappings)
```

### 3.2 Neue Tabellen (Zusammenfassung)

| Migration | Tabellen | Spalten (ca.) | Indexes |
|-----------|----------|---------------|---------|
| 000036 | `tenants`, `tenant_members` | 12 + 4 | 3 |
| 000037 | `ticket_custom_field_values`, `project_custom_field_values` + ALTER | 5 + 5 | 2 |
| 000038 | ALTER `companies` (+14 Spalten), ALTER `contacts` (+18 Spalten), `contact_company_roles` | 14 + 18 + 10 | 10 |
| 000039 | `audit_entries` | 12 | 7 |
| 000040 | `tax_rates` | 11 | 4 |
| 000041 | `number_sequences`, `document_chain_items`, `document_line_items`, `payments`, `dunnings` | 8 + 28 + 18 + 10 + 12 | 22 |
| 000042 | 3 Trigger-Funktionen | — | — |
| 000043 | `email_accounts`, `email_folders`, `email_messages`, `email_attachments` | 20 + 14 + 28 + 8 | 23 |
| 000044 | `canned_responses`, `internal_notes` | 11 + 10 | 6 |
| 000045 | `duplicate_candidates`, `merge_history` | 11 + 9 | 7 |
| 000046 | `shared_links`, `shared_link_access_log` | 14 + 7 | 5 |
| 000047 | `consent_purposes`, `consents` | 10 + 14 | 7 |
| 000048 | `retention_rules` | 10 | 3 |
| 000049 | `datev_export_batches`, `datev_export_entries` | 14 + 12 | 5 |
| 000050 | `integration_connections`, `integration_sync_log`, `integration_entity_mappings` | 18 + 12 + 10 | 9 |

**Total: 30 neue Tabellen, 3 Trigger, ~32 ALTER TABLE Spalten, ~113 Indexes**

### 3.3 Abhaengigkeitsgraph

```
000036 (tenants)        --- optional, unabhaengig
000037 (custom_fields)  --- unabhaengig
000038 (companies/contacts) --- abh: 000007
000039 (audit)          --- abh: 000001
000040 (tax_rates)      --- unabhaengig
000041 (belegkette)     --- abh: 000007, 000040
000042 (gobd)           --- abh: 000041
000043 (email)          --- abh: 000001, 000007, 000009
000044 (helpdesk)       --- abh: 000001
000045 (duplikate)      --- abh: 000001
000046 (file-sharing)   --- abh: 000001
000047 (consent)        --- abh: 000007
000048 (retention)      --- unabhaengig
000049 (datev)          --- abh: 000001
000050 (integrations)   --- abh: 000001
```

### 3.4 Empfohlene Reihenfolge fuer Luke

**Sofort (vor Phase 8):**
- 000037 (Custom Fields erweitern) — blockt nichts, hilft allen Modulen
- 000038 (Companies/Contacts erweitern) — DACH-Grunderwartung
- 000039 (Audit-Log) — Basis fuer Phase 9 (Security)
- 000040 (MWSt) — Konfigurationstabelle, blockt Finance
- 000044 (Canned Responses + Notes) — 3-5 Tage, sofortiger Nutzen

**Mit Phase 10 (E-Mail):**
- 000043 (E-Mail-System)

**Mit Phase 12 (Finance):**
- 000041 (Belegkette)
- 000042 (GoBD-Trigger)
- 000049 (DATEV-Export)

**Mit Phase 9 (Security):**
- 000047 (Consent)
- 000048 (Retention)

**Mit Phase 11 (Documents):**
- 000046 (File-Sharing Links)

**Mit Phase 16+ (Integrationen):**
- 000050 (Integration-Configs)

**Nach Bedarf:**
- 000036 (Tenants) — Luke entscheidet Zeitpunkt
- 000045 (Duplikate) — wenn CRM-Daten wachsen

---

## 4. Sicherheit & Compliance

### 4.1 Row-Level Security (RLS)

**Status:** Aktuell single-tenant (kein `tenant_id` in bestehenden Tabellen).

**Plan:**
1. Neue Tabellen (000036-000050) enthalten `tenant_id UUID NOT NULL` als Platzhalter
2. Bestehende Tabellen (000001-000035) werden per ALTER TABLE nachgeruestet wenn Multi-Tenancy aktiviert wird
3. RLS-Activation per Tabelle:

```sql
-- Pro Tabelle (nach Multi-Tenancy-Einrichtung):
ALTER TABLE {tabelle} ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON {tabelle}
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);
```

**Implementierung in Go:**
```go
// Pro Request den Tenant-Context setzen:
func SetTenantContext(ctx context.Context, db *sql.DB, tenantID uuid.UUID) error {
    _, err := db.ExecContext(ctx,
        "SET LOCAL app.current_tenant_id = $1", tenantID.String())
    return err
}
```

**Empfehlung:** RLS erst mit erstem zahlenden Multi-Tenant-Kunden aktivieren. Vorher: Application-Level Tenant-Filterung per `WHERE tenant_id = ?`.

### 4.2 Audit-Logging

**Tabelle:** `audit_entries` (Migration 000039)

**Was wird geloggt:**

| Aktion | Entity-Types | Details |
|--------|-------------|---------|
| `create` | Alle Business-Entities | Neuer Record |
| `update` | Alle | Aenderungsdiff: `{"field": {"old": "X", "new": "Y"}}` |
| `delete` | Alle | Snapshot des geloeschten Records |
| `login` | users | IP, User-Agent |
| `export` | contacts, invoices, time_entries | Was exportiert wurde |
| `view` | Sensible Entities (Payroll, HR) | Wer hat was eingesehen |
| `lock` | document_chain_items | GoBD: Beleg gesperrt |
| `send` | invoices, dunnings | Dokument versendet |
| `merge` | contacts, companies | Duplikat-Merge |

**Implementierung in Go:**

```go
// Audit-Service als Middleware/Decorator:
type AuditService struct {
    repo AuditRepository
}

func (s *AuditService) Log(ctx context.Context, entry AuditEntry) error {
    entry.UserID = auth.UserIDFromContext(ctx)
    entry.IPAddress = middleware.IPFromContext(ctx)
    entry.UserAgent = middleware.UserAgentFromContext(ctx)
    entry.CreatedAt = time.Now()
    return s.repo.Insert(ctx, entry)
}
```

**GoBD-Anforderung:** Audit-Eintraege sind IMMUTABLE (kein UPDATE, kein DELETE). Sichergestellt durch:
1. Kein `updated_at` auf `audit_entries`
2. Application-Level: Kein Update/Delete-Endpoint
3. Optional: DB-Trigger der UPDATE/DELETE auf `audit_entries` verhindert

**Retention:** Audit-Logs 10 Jahre aufbewahren (GoBD DE). Partitionierung nach Monat empfohlen ab >10M Rows.

### 4.3 Verschluesselung

#### At Rest (Storage)

| Schicht | Methode |
|---------|---------|
| PostgreSQL | `pgcrypto`-Extension fuer feldspezifische Verschluesselung. Full-Disk-Encryption auf Hetzner (Standard). |
| MinIO/S3 | Server-Side Encryption (SSE-S3 oder SSE-KMS) |
| Backups | AES-256-GCM Verschluesselung pro Tenant |
| Redis | Keine sensiblen Daten im Cache. TTL fuer alle Keys. |

#### In Transit

| Verbindung | Methode |
|------------|---------|
| Client -> Gateway | TLS 1.3 (Let's Encrypt, HSTS) |
| Gateway -> Services | gRPC mit TLS (intern). Oder mTLS in Produktion. |
| Services -> PostgreSQL | `sslmode=require` in Connection-String |
| Services -> Redis | TLS (`rediss://`) |
| Services -> MinIO | HTTPS |
| Services -> LiveKit | WebRTC (DTLS + SRTP, von LiveKit gehandhabt) |

#### Field-Level Encryption

Sensible Felder die feldbasiert verschluesselt werden muessen:

| Tabelle | Feld | Methode |
|---------|------|---------|
| `email_accounts` | `imap_password_encrypted`, `smtp_password_encrypted` | AES-256-GCM (BYTEA) |
| `integration_connections` | `credentials_encrypted` | AES-256-GCM (JSONB -> BYTEA) |
| `shared_links` | `password_hash` | Bcrypt (One-Way) |
| `users` | `password_hash` | Argon2id (bereits implementiert) |

**Key Management:**

```go
// Encryption-Key aus Environment-Variable:
// ENCRYPTION_KEY=base64-encoded-32-byte-key
// Pro Tenant eigener Key (spaeter, mit Key-Rotation)

type FieldEncryptor struct {
    key []byte // 32 Bytes fuer AES-256
}

func (e *FieldEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
    block, _ := aes.NewCipher(e.key)
    gcm, _ := cipher.NewGCM(block)
    nonce := make([]byte, gcm.NonceSize())
    io.ReadFull(rand.Reader, nonce)
    return gcm.Seal(nonce, nonce, plaintext, nil), nil
}
```

### 4.4 DSGVO-Tools

#### Art. 15: Auskunftsrecht (Datenexport)

**Ablauf:**
1. Admin klickt "DSGVO-Auskunft" fuer einen Kontakt
2. System durchsucht ALLE Tabellen nach Daten zu diesem Kontakt:
   - `contacts` (Stammdaten)
   - `contact_company_roles` (Firmen-Zuordnungen)
   - `activities` (Interaktionen)
   - `deals` (Geschaeftsbezuege)
   - `email_messages` (E-Mail-Verlauf)
   - `consents` (Einwilligungen)
   - `audit_entries` (Zugriffe auf diesen Kontakt)
   - `internal_notes` (interne Notizen)
   - `document_chain_items` (Belege)
3. JSON/CSV-Export als ZIP-Paket

**Implementierung:** `internal/compliance/gdpr/export.go`
- Interface `DataCollector` pro Modul
- Registry die alle Collector zusammenfuehrt
- Async-Job (kann bei grossen Datenmengen Minuten dauern)
- Download-Link per E-Mail an Admin

#### Art. 17: Recht auf Loeschung

**ACHTUNG:** GoBD-Rechnungen duerfen NICHT geloescht werden (10-Jahres-Frist). Loesung: Anonymisierung.

**Ablauf:**
1. Admin klickt "DSGVO-Loeschung" fuer einen Kontakt
2. System prueft Aufbewahrungsfristen (Retention-Rules):
   - Rechnungen mit `is_locked = TRUE`: NUR Kontaktdaten anonymisieren (Name -> "GELOESCHT", E-Mail -> NULL)
   - Aktive Vertraege: Warnung, Loeschung erst nach Vertragsende
   - Alles andere: Kaskadierte Loeschung
3. Audit-Log-Eintrag: "DSGVO-Loeschung durchgefuehrt" (DARF NICHT geloescht werden!)

**Implementierung:** `internal/compliance/gdpr/erasure.go`
- Interface `DataAnonymizer` pro Modul
- Transaktionale Ausfuehrung (alles oder nichts)
- Trocken-Lauf-Modus (zeigt was geloescht wuerde)

#### Art. 20: Datenportabilitaet

- Gleicher Export wie Art. 15, aber in maschinenlesbarem Format (JSON, CSV)
- ZIP-Paket mit einem Ordner pro Modul

### 4.5 GoBD-Compliance

**Anforderungen fuer Rechnungen/Belege in Deutschland:**

| Anforderung | Implementierung |
|-------------|----------------|
| **Unveraenderbarkeit** | `is_locked` Flag + DB-Trigger verhindert UPDATE nach Versand (Migration 000042) |
| **Lueckenlose Nummerierung** | `number_sequences` Tabelle mit `SELECT ... FOR UPDATE` (Race-Condition-sicher) |
| **Storno statt Loeschung** | Gutschrift (`credit_note`) statt DELETE. DELETE-Trigger verhindert Loeschung gesperrter Belege. |
| **Aenderungsprotokoll** | `audit_entries` fuer jede Aenderung VOR dem Lock |
| **Pflichtangaben** | Validierung in Service-Layer: Rechnungsnummer, Datum, Absender (inkl. USt-IdNr), Empfaenger, Einzelpositionen mit Menge+Preis, Steuersatz, Netto/Brutto |
| **Aufbewahrung 10 Jahre** | `retention_rules` + automatische Warnung vor Ablauf |
| **Digitale Archivierung** | PDF + DB-Record. Original-PDF nicht veraenderbar (MinIO Object-Lock) |

**Kritische Go-Implementierung:**

```go
// Nummer generieren (race-condition-sicher):
func (s *NumberService) NextNumber(ctx context.Context, tx *sql.Tx, tenantID uuid.UUID, docType string) (string, error) {
    var seq NumberSequence
    // SELECT FOR UPDATE sperrt die Zeile fuer andere Transaktionen
    err := tx.QueryRowContext(ctx,
        `SELECT id, prefix, current_number, format_pattern
         FROM number_sequences
         WHERE tenant_id = $1 AND sequence_type = $2
         FOR UPDATE`,
        tenantID, docType).Scan(&seq.ID, &seq.Prefix, &seq.CurrentNumber, &seq.FormatPattern)
    if err != nil {
        return "", err
    }

    seq.CurrentNumber++
    _, err = tx.ExecContext(ctx,
        `UPDATE number_sequences SET current_number = $1, updated_at = NOW() WHERE id = $2`,
        seq.CurrentNumber, seq.ID)
    if err != nil {
        return "", err
    }

    return formatNumber(seq), nil
}
```

### 4.6 Schweiz-spezifisch (nDSG)

| Aspekt | Unterschied zu DSGVO | Implikation fuer KMU Hub |
|--------|---------------------|--------------------------|
| Schutzobjekt | Nur natuerliche Personen (nicht juristische) | Firmen-Daten fallen NICHT unter nDSG |
| Bussen | Bis 250.000 CHF, treffen natuerliche Person (GF!) | Severity in Verkaufsmaterialien betonen |
| DSB | Freiwillig ("Datenschutzberater") | Optional in Settings anbieten |
| Datenfluss DE <-> CH | Unproblematisch (beidseitige Angemessenheit) | Kein spezieller Transfer-Mechanismus noetig |
| Datenresidenz | Kein Pflicht, aber Erwartung | Schweizer Cluster auf Exoscale anbieten |

### 4.7 Compliance-Roadmap (Timeline)

| Wann | Was | Aufwand | Kosten (extern) |
|------|-----|---------|-----------------|
| **Vor Beta** | AVV-Vorlage (Rechtsanwalt) | — | 2.000-4.000 EUR |
| **Vor Beta** | TOMs dokumentieren | 2-3 Tage | — |
| **Vor Beta** | Verarbeitungsverzeichnis | 1-2 Tage | — |
| **Vor Beta** | RLS aktivieren ODER App-Level Tenant-Filterung | 1-2 Wochen | — |
| **Vor Beta** | Audit-Log (Migration 000039 + Service) | 1-2 Wochen | — |
| **Vor Beta** | 2FA (TOTP) — Phase 9 | 1-2 Wochen | — |
| **Vor Launch** | DSGVO-Auskunft-Tool (Art. 15) | 2-3 Wochen | — |
| **Vor Launch** | DSGVO-Loeschung (Art. 17) | 2-3 Wochen | — |
| **Vor Launch** | Consent-Management (Migration 000047) | 1-2 Wochen | — |
| **Vor Launch** | Penetrationstest | — | 3.000-8.000 EUR |
| **Vor Launch** | Externer DSB | — | 300-500 EUR/Mo |
| **Nach Launch** | Retention-Policy-Automation | 1-2 Wochen | — |
| **12+ Monate** | ISO 27001 Vorbereitung | — | 30.000-70.000 EUR |

**Geschaetzte Gesamtkosten Compliance bis Launch: ~12.000-24.000 EUR** (+ interne Entwicklungszeit)

---

## Zusammenfassung: Was Luke jetzt wissen muss

### 5 Dinge die sich aendern gegenueber der aktuellen Roadmap:

1. **Phase 12 (Finance) muss tiefer gehen:** Belegkette als State-Machine, GoBD-Trigger, QR-Rechnung (CH), Nummernkreise. Nicht nur "Quotes + Invoices".

2. **Phase 11 (Documents) sollte OnlyOffice WOPI enthalten:** Ohne das brauchen Kunden weiterhin Office 365. WOPI-Endpoints sind 2-4 Wochen Aufwand.

3. **Unified Inbox als Phase 10.5:** Nach E-Mail (Phase 10) sofort Website-Chat-Widget und WhatsApp-Adapter bauen. Das ist DER Differentiator gegenueber Zoho/Odoo.

4. **CRM-Erweiterungen VOR Phase 10:** Firma als eigene Entity (000038) und Kontaktfelder (Titel, Anrede) muessen stehen bevor E-Mail-zu-Kontakt-Zuordnung gebaut wird.

5. **Audit-Log + Consent frueh einbauen:** 000039 (Audit) und 000047 (Consent) blockieren Phase 9 (Security). Besser vorher als Migration laufen lassen.

### Empfohlene Service-Erweiterung:

```
Bestehend:        Neu (wenn noetig):
- gateway         - biz (Finance, Belegkette, DATEV)
- auth            - inbox (Unified Inbox, nach Phase 10)
- crm
- chat
- notification
- work
```

Maximal 2 neue Services. `biz` fuer Finance/HR (wie in Luke's Roadmap Phase 12-13 geplant). `inbox` nur wenn Unified Inbox zu komplex fuer Chat-Service wird. Alles andere als Packages in bestehende Services integrieren.

---

*Erstellt: 2026-02-17 | Grundlage: 00-SYNTHESE, 08-datenbankmodelle, 13-vision-ergaenzungen, ROADMAP, CLAUDE.md*
*Teil 2 behandelt: Detaillierte API-Endpoints, gRPC-Proto-Erweiterungen, Go-Package-Strukturen, Test-Strategie*

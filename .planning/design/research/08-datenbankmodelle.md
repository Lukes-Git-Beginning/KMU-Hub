# 08 — PostgreSQL-Datenbankmodelle fuer fehlende Features

**Datum:** 2026-02-17
**Grundlage:** `00-SYNTHESE.md` (Feature-Luecken), `BACKEND-REQUIREMENTS-AUDIT.md` (332 Endpoints)
**Bestehende Migrations:** 000001–000035 (users, roles, CRM, chat, notifications, dashboard, projects, tasks, time_entries, calendars, events, resources, holidays)
**Naechste Migration:** 000036+

---

## Inhaltsverzeichnis

1. [Bestandsanalyse — Was existiert bereits](#1-bestandsanalyse)
2. [Architektur-Entscheidungen](#2-architektur-entscheidungen)
3. [Migrations-Reihenfolge](#3-migrations-reihenfolge)
4. [Schemas — Querschnittsfunktionen](#4-querschnittsfunktionen)
5. [Schemas — CRM-Erweiterungen](#5-crm-erweiterungen)
6. [Schemas — E-Mail (IMAP-Cache)](#6-e-mail)
7. [Schemas — Belegkette / Finance](#7-belegkette)
8. [Schemas — Helpdesk-Erweiterungen](#8-helpdesk)
9. [Schemas — Dokumente / File-Sharing](#9-dokumente)
10. [Schemas — Compliance (DSGVO/GoBD)](#10-compliance)
11. [Schemas — Integrationen](#11-integrationen)
12. [Entity-Relationship-Uebersicht](#12-er-uebersicht)
13. [PostgreSQL-spezifische Features](#13-pg-features)

---

## 1. Bestandsanalyse

### Bereits vorhanden (Migrations 000001–000035)

| Tabelle | Migration | Anmerkung |
|---------|-----------|-----------|
| `users` | 000001 | Kein `tenant_id` — single-tenant aktuell |
| `roles`, `permissions`, `role_permissions`, `user_roles` | 000002 | RBAC-System |
| `refresh_tokens` | 000003 | JWT Auth |
| `invitations` | 000004 | Team-Einladungen |
| `custom_field_definitions` | 000005 | Nur fuer CRM-Entities (contact/company/deal/activity) |
| `tags`, `*_tags` Junction-Tables | 000006 | Tag-System |
| `companies`, `contacts`, `*_custom_field_values` | 000007 | Basis-CRM |
| `pipeline_stages` | 000008 | Sales-Pipeline |
| `deals`, `deal_custom_field_values` | 000009 | Deals |
| `activities` | 000010 | CRM-Aktivitaeten |
| FTS (`search_vector`) | 000011 | Volltextsuche DE |
| `saved_filters` | 000012 | Gespeicherte Filter |
| `deal_stage_history` | 000013 | Pipeline-History |
| `chat_channels`, `channel_members` | 000014 | Chat |
| `chat_messages` | 000015 | Nachrichten |
| DM + Threads | 000016 | Direct Messages |
| `mentions` | 000017 | @-Mentions |
| `chat_files`, `storage_quotas` | 000018 | Datei-Uploads |
| Chat-Suche | 000019 | FTS fuer Chat |
| `event_types` | 000020 | Notification Event-Types |
| `notifications` | 000021 | Benachrichtigungen |
| `notification_preferences` | 000022 | Notification-Settings |
| `dashboard_layouts` | 000023 | Dashboard |
| `projects`, `project_members`, `project_statuses` | 000024 | Projekte |
| `tasks` | 000025 | Aufgaben |
| Task Collaboration | 000026 | Kommentare, Dateien, Abhaengigkeiten |
| Work Event Type Seeds | 000027 | Seed-Daten |
| Entity Display Names | 000028 | Darstellungsformat |
| Notification Permissions | 000029 | RBAC |
| `time_entries` | 000030 | Zeiterfassung (an Tasks) |
| Gantt View Type | 000031 | View-Erweiterung |
| `calendars`, `calendar_members`, `event_categories` | 000032 | Kalender |
| `calendar_events`, `event_attendees`, `event_exceptions`, `event_reminders`, `user_calendar_preferences` | 000033 | Events |
| `resources` | 000034 | Raum/Geraete-Buchung |
| `holidays` | 000035 | Feiertage |

### Wichtig: Kein `tenant_id`

Die bestehenden Migrations haben KEIN `tenant_id`. Das Projekt ist aktuell single-tenant (vgl. Kommentar in 000018). **Multi-Tenancy wird spaeter per ALTER TABLE + RLS nachgeruestet.**

Fuer die neuen Schemas verwenden wir trotzdem `tenant_id` als Platzhalter, damit die Modelle zukunftssicher sind. Bei der tatsaechlichen Migration entscheidet Luke, ob `tenant_id` sofort oder spaeter eingefuehrt wird.

---

## 2. Architektur-Entscheidungen

### ADR-01: JSONB fuer Custom Fields (nicht EAV)

**Entscheidung:** Custom-Field-Werte werden als JSONB in Junction-Tables gespeichert (1 Row pro Entity x Field), nicht als Spalte auf der Entity-Tabelle.

**Begruendung:**
- Bereits implementiert in 000005/000007/000009 (`*_custom_field_values` Tables)
- GIN-Index auf `value` JSONB-Spalte ermoeglicht performante Suche
- Kein Schema-Lock bei neuen Feldern
- EAV waere flexibler aber langsamer bei Queries

### ADR-02: Immutable Records fuer GoBD

**Entscheidung:** `invoices` und `invoice_line_items` haben eine `is_locked` Flag. Nach Versand: `is_locked = TRUE`, UPDATE wird per Trigger verhindert. Storno statt Loeschung.

### ADR-03: Audit-Log als eigene Tabelle (nicht CDC)

**Entscheidung:** Application-Level Audit-Log per INSERT in `audit_entries`. Kein Logical Replication / Change Data Capture.

**Begruendung:** Einfacher zu implementieren, ausreichend fuer GoBD/DSGVO, kein Infra-Overhead.

### ADR-04: IMAP-Cache (nicht -Mirror)

**Entscheidung:** E-Mails werden als IMAP-Cache in PostgreSQL gespeichert. Der IMAP-Server bleibt Source of Truth. Sync per `UIDVALIDITY` + `HIGHESTMODSEQ`.

### ADR-05: Belegkette als State-Machine

**Entscheidung:** `document_chain_items` mit `document_type` (quote/order/delivery_note/invoice/credit_note) und `derived_from_id` FK auf sich selbst. Jeder Beleg kann aus dem vorherigen erzeugt werden.

### ADR-06: Multi-Tenant Ready (Vorwaertskompatibel)

**Entscheidung:** Neue Tabellen enthalten `tenant_id UUID NOT NULL`. Bei der tatsaechlichen Implementation wird entweder:
- Eine `tenants`-Tabelle erstellt und FKs nachgezogen, oder
- RLS-Policies direkt auf `tenant_id` gesetzt

Da `tenants` noch NICHT existiert, werden die FKs in den Schemas als Kommentar markiert (`-- REFERENCES tenants(id) when table exists`). Luke entscheidet den Zeitpunkt.

---

## 3. Migrations-Reihenfolge

Abhaengigkeiten beachten: Tabellen die per FK referenziert werden, muessen ZUERST erstellt werden.

```
000036  tenants + tenant_id Vorbereitung (optional, entscheidet Luke)
000037  Custom Fields erweitern (entity_types fuer neue Module)
000038  Companies erweitern (Hierarchie, contact_company_roles)
000039  Audit-Log
000040  Tax Rates (MWSt multi-country)
000041  Belegkette (document_chain_items, line_items)
000042  GoBD (invoice_lock Trigger, Nummernkreise)
000043  E-Mail (email_accounts, email_folders, email_messages, email_attachments)
000044  Helpdesk-Erweiterungen (canned_responses, internal_notes)
000045  Duplikaterkennung (duplicate_candidates, merge_history)
000046  File-Sharing (shared_links, access_log)
000047  Consent-Management (consents)
000048  Retention-Policies (retention_rules)
000049  DATEV-Export (export_batches, export_entries)
000050  Integration-Configs (integration_connections)
```

---

## 4. Querschnittsfunktionen

### 4.1 Tenants (Vorbereitung fuer Multi-Tenancy)

```sql
-- Migration 000036: tenants
-- HINWEIS: Optional. Luke entscheidet ob sofort oder spaeter.
-- Wenn spaeter: tenant_id in neuen Tabellen wird als NOT NULL ohne FK erstellt
-- und per ALTER TABLE + FK nachgeruestet.

CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL,           -- URL-tauglicher Identifier
    plan VARCHAR(50) NOT NULL DEFAULT 'trial',  -- 'trial', 'starter', 'professional', 'enterprise'
    country VARCHAR(2) NOT NULL DEFAULT 'DE',   -- ISO 3166-1 alpha-2
    timezone VARCHAR(50) NOT NULL DEFAULT 'Europe/Berlin',
    locale VARCHAR(10) NOT NULL DEFAULT 'de-DE',
    settings JSONB NOT NULL DEFAULT '{}',  -- Tenant-spezifische Einstellungen
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    trial_ends_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_tenants_slug ON tenants(slug);
CREATE INDEX idx_tenants_active ON tenants(is_active) WHERE is_active = TRUE;

-- Tenant-Membership (welcher User gehoert zu welchem Tenant)
CREATE TABLE tenant_members (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL DEFAULT 'member',  -- 'owner', 'admin', 'member'
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, user_id)
);

CREATE INDEX idx_tenant_members_user ON tenant_members(user_id);
```

### 4.2 Audit-Log

```sql
-- Migration 000039: audit_entries
-- Zentrale Aenderungsverfolgung fuer DSGVO (Art. 30) und GoBD

CREATE TABLE audit_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,  -- REFERENCES tenants(id) when table exists
    -- Wer
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    user_email VARCHAR(255),              -- Snapshot fuer den Fall dass User geloescht wird
    ip_address INET,
    user_agent TEXT,
    -- Was
    action VARCHAR(50) NOT NULL,          -- 'create', 'update', 'delete', 'login', 'export', 'view', 'lock', 'unlock'
    entity_type VARCHAR(100) NOT NULL,    -- 'contact', 'invoice', 'deal', etc.
    entity_id UUID,                       -- ID des betroffenen Records
    entity_display_name VARCHAR(500),     -- Lesbarer Name zum Zeitpunkt der Aenderung
    -- Details
    changes JSONB,                        -- {"field": {"old": "X", "new": "Y"}, ...}
    metadata JSONB,                       -- Zusaetzliche Kontextdaten
    -- Wann
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    -- KEIN updated_at: Audit-Eintraege sind IMMUTABLE
);

-- Partitionierung nach Monat empfohlen fuer grosse Tenants (spaeter)
-- CREATE TABLE audit_entries (...) PARTITION BY RANGE (created_at);

CREATE INDEX idx_audit_entries_tenant ON audit_entries(tenant_id);
CREATE INDEX idx_audit_entries_user ON audit_entries(user_id);
CREATE INDEX idx_audit_entries_entity ON audit_entries(entity_type, entity_id);
CREATE INDEX idx_audit_entries_action ON audit_entries(action);
CREATE INDEX idx_audit_entries_created ON audit_entries(created_at DESC);
CREATE INDEX idx_audit_entries_tenant_created ON audit_entries(tenant_id, created_at DESC);

-- GIN-Index fuer JSONB-Suche in changes
CREATE INDEX idx_audit_entries_changes ON audit_entries USING GIN (changes jsonb_path_ops);
```

### 4.3 Custom Fields erweitern

```sql
-- Migration 000037: Custom Fields fuer neue Module
-- Bestehende custom_field_definitions hat CHECK constraint auf entity_type.
-- Diesen erweitern fuer neue Module.

ALTER TABLE custom_field_definitions
    DROP CONSTRAINT valid_entity_type;

ALTER TABLE custom_field_definitions
    ADD CONSTRAINT valid_entity_type CHECK (
        entity_type IN (
            'contact', 'company', 'deal', 'activity',           -- bestehend
            'ticket', 'project', 'task',                         -- Helpdesk + PM
            'invoice', 'quote', 'order',                         -- Finance
            'vehicle', 'article', 'contract',                    -- Industry
            'rental_object', 'field_report', 'form'              -- Erweiterungen
        )
    );

-- Neue Custom-Field-Value-Tables fuer Module die Custom Fields brauchen
-- (Contacts, Companies, Deals haben bereits *_custom_field_values)

CREATE TABLE ticket_custom_field_values (
    ticket_id UUID NOT NULL,  -- FK wird in Helpdesk-Migration gesetzt
    field_id UUID NOT NULL REFERENCES custom_field_definitions(id) ON DELETE CASCADE,
    value JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (ticket_id, field_id)
);

CREATE INDEX idx_ticket_cfv_field ON ticket_custom_field_values(field_id);

CREATE TABLE project_custom_field_values (
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    field_id UUID NOT NULL REFERENCES custom_field_definitions(id) ON DELETE CASCADE,
    value JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (project_id, field_id)
);

CREATE INDEX idx_project_cfv_field ON project_custom_field_values(field_id);

-- tenant_id zu custom_field_definitions hinzufuegen (Vorbereitung Multi-Tenancy)
-- ALTER TABLE custom_field_definitions ADD COLUMN tenant_id UUID;
-- Wird aktiviert wenn tenants-Tabelle existiert
```

### 4.4 MWSt Multi-Country (Steuersaetze)

```sql
-- Migration 000040: tax_rates
-- Multi-Country MWSt: DE (19%/7%), CH (8.1%/2.6%/3.8%), AT (20%/10%/13%)

CREATE TABLE tax_rates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,  -- REFERENCES tenants(id) when table exists
    country VARCHAR(2) NOT NULL,          -- ISO 3166-1: 'DE', 'CH', 'AT'
    name VARCHAR(100) NOT NULL,           -- z.B. "Normalsatz", "Ermaessigter Satz", "Sondersatz"
    rate DECIMAL(5,2) NOT NULL,           -- z.B. 19.00, 7.00, 8.10, 2.60
    category VARCHAR(50) NOT NULL,        -- 'standard', 'reduced', 'super_reduced', 'zero', 'exempt'
    description TEXT,                     -- z.B. "Grundnahrungsmittel, Buecher"
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    valid_from DATE NOT NULL DEFAULT '2024-01-01',
    valid_to DATE,                        -- NULL = unbegrenzt gueltig
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_rate CHECK (rate >= 0 AND rate < 100),
    CONSTRAINT valid_category CHECK (category IN ('standard', 'reduced', 'super_reduced', 'zero', 'exempt')),
    CONSTRAINT valid_date_range CHECK (valid_to IS NULL OR valid_to > valid_from)
);

CREATE INDEX idx_tax_rates_tenant ON tax_rates(tenant_id);
CREATE INDEX idx_tax_rates_country ON tax_rates(country);
CREATE INDEX idx_tax_rates_active ON tax_rates(country, valid_from, valid_to)
    WHERE valid_to IS NULL OR valid_to >= CURRENT_DATE;
CREATE UNIQUE INDEX idx_tax_rates_default ON tax_rates(tenant_id, country)
    WHERE is_default = TRUE;

-- Seed: Standard-Steuersaetze DACH
-- Diese werden per Tenant-Setup eingefuegt, hier als Referenz:
/*
INSERT INTO tax_rates (tenant_id, country, name, rate, category, is_default) VALUES
    -- Deutschland
    ('{tid}', 'DE', 'Normalsatz', 19.00, 'standard', TRUE),
    ('{tid}', 'DE', 'Ermaessigter Satz', 7.00, 'reduced', FALSE),
    ('{tid}', 'DE', 'Steuerfrei', 0.00, 'exempt', FALSE),
    -- Schweiz (seit 2024-01-01)
    ('{tid}', 'CH', 'Normalsatz', 8.10, 'standard', TRUE),
    ('{tid}', 'CH', 'Reduzierter Satz', 2.60, 'reduced', FALSE),
    ('{tid}', 'CH', 'Sondersatz Beherbergung', 3.80, 'super_reduced', FALSE),
    ('{tid}', 'CH', 'Steuerfrei', 0.00, 'exempt', FALSE),
    -- Oesterreich
    ('{tid}', 'AT', 'Normalsatz', 20.00, 'standard', TRUE),
    ('{tid}', 'AT', 'Ermaessigter Satz', 10.00, 'reduced', FALSE),
    ('{tid}', 'AT', 'Zwischensteuersatz', 13.00, 'super_reduced', FALSE),
    ('{tid}', 'AT', 'Steuerfrei', 0.00, 'exempt', FALSE);
*/
```

---

## 5. CRM-Erweiterungen

### 5.1 Companies erweitern (Hierarchie + Kontaktrollen)

```sql
-- Migration 000038: Companies erweitern
-- Firma als eigene Entity mit Hierarchie und Kontaktrollen

-- Neue Spalten fuer companies
ALTER TABLE companies ADD COLUMN parent_company_id UUID REFERENCES companies(id) ON DELETE SET NULL;
ALTER TABLE companies ADD COLUMN website VARCHAR(500);
ALTER TABLE companies ADD COLUMN phone VARCHAR(50);
ALTER TABLE companies ADD COLUMN email VARCHAR(255);
ALTER TABLE companies ADD COLUMN tax_id VARCHAR(50);              -- USt-IdNr / UID / MWST-Nr
ALTER TABLE companies ADD COLUMN registration_number VARCHAR(100); -- HRB / CHE / FN
ALTER TABLE companies ADD COLUMN registration_authority VARCHAR(100); -- Amtsgericht / Zefix
ALTER TABLE companies ADD COLUMN legal_form VARCHAR(100);         -- GmbH, AG, e.K., Einzelfirma
ALTER TABLE companies ADD COLUMN annual_revenue DECIMAL(15,2);
ALTER TABLE companies ADD COLUMN postal_code VARCHAR(20);
ALTER TABLE companies ADD COLUMN state_province VARCHAR(100);     -- Bundesland / Kanton
ALTER TABLE companies ADD COLUMN street VARCHAR(500);
ALTER TABLE companies ADD COLUMN logo_url TEXT;
ALTER TABLE companies ADD COLUMN category VARCHAR(50) DEFAULT 'customer';  -- customer, supplier, partner, prospect
ALTER TABLE companies ADD COLUMN status VARCHAR(50) DEFAULT 'active';      -- active, inactive, archived
ALTER TABLE companies ADD COLUMN custom_data JSONB DEFAULT '{}';  -- Zusaetzliche unstrukturierte Daten

CREATE INDEX idx_companies_parent ON companies(parent_company_id) WHERE parent_company_id IS NOT NULL;
CREATE INDEX idx_companies_tax_id ON companies(tax_id) WHERE tax_id IS NOT NULL;
CREATE INDEX idx_companies_category ON companies(category);
CREATE INDEX idx_companies_status ON companies(status);

-- Kontaktrollen: N:M zwischen contacts und companies MIT Rolle
-- Ersetzt das einfache company_id FK auf contacts
CREATE TABLE contact_company_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contact_id UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    role VARCHAR(100) NOT NULL,           -- 'Geschaeftsfuehrer', 'Ansprechpartner', 'Techniker', etc.
    department VARCHAR(100),              -- Abteilung
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,  -- Hauptfirma des Kontakts
    started_at DATE,
    ended_at DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT unique_contact_company_role UNIQUE (contact_id, company_id, role)
);

CREATE INDEX idx_ccr_contact ON contact_company_roles(contact_id);
CREATE INDEX idx_ccr_company ON contact_company_roles(company_id);
CREATE INDEX idx_ccr_primary ON contact_company_roles(contact_id) WHERE is_primary = TRUE;
```

### 5.2 Kontakte erweitern (Akadem. Titel, Anrede)

```sql
-- In Migration 000038 (zusammen mit Companies)
-- Kontakte um DACH-spezifische Felder erweitern

ALTER TABLE contacts ADD COLUMN salutation VARCHAR(20);           -- 'Herr', 'Frau', 'Divers', ''
ALTER TABLE contacts ADD COLUMN academic_title VARCHAR(100);      -- 'Prof. Dr.', 'Dr.', 'Dipl.-Ing.', etc.
ALTER TABLE contacts ADD COLUMN mobile VARCHAR(50);
ALTER TABLE contacts ADD COLUMN department VARCHAR(100);
ALTER TABLE contacts ADD COLUMN website VARCHAR(500);
ALTER TABLE contacts ADD COLUMN date_of_birth DATE;
ALTER TABLE contacts ADD COLUMN preferred_language VARCHAR(5);    -- 'de', 'en', 'fr', 'it'
ALTER TABLE contacts ADD COLUMN preferred_communication VARCHAR(20) DEFAULT 'email'; -- 'email', 'phone', 'chat'
ALTER TABLE contacts ADD COLUMN formal_address BOOLEAN NOT NULL DEFAULT TRUE;  -- Sie (true) vs. Du (false)
ALTER TABLE contacts ADD COLUMN category VARCHAR(50) DEFAULT 'customer';  -- customer, supplier, partner, employee, lead
ALTER TABLE contacts ADD COLUMN status VARCHAR(50) DEFAULT 'active';       -- active, inactive, archived
ALTER TABLE contacts ADD COLUMN street VARCHAR(500);
ALTER TABLE contacts ADD COLUMN postal_code VARCHAR(20);
ALTER TABLE contacts ADD COLUMN city VARCHAR(100);                -- Erweitert bestehende Felder
ALTER TABLE contacts ADD COLUMN state_province VARCHAR(100);
ALTER TABLE contacts ADD COLUMN country VARCHAR(100);
ALTER TABLE contacts ADD COLUMN social_media JSONB DEFAULT '{}';  -- {"linkedin": "url", "xing": "url", ...}
ALTER TABLE contacts ADD COLUMN is_favorite BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE contacts ADD COLUMN last_contacted_at TIMESTAMPTZ;
ALTER TABLE contacts ADD COLUMN source VARCHAR(100);              -- 'website', 'referral', 'cold_call', 'import'

CREATE INDEX idx_contacts_category ON contacts(category);
CREATE INDEX idx_contacts_status ON contacts(status);
CREATE INDEX idx_contacts_favorite ON contacts(is_favorite) WHERE is_favorite = TRUE;
CREATE INDEX idx_contacts_mobile ON contacts(mobile) WHERE mobile IS NOT NULL;
```

### 5.3 Duplikaterkennung

```sql
-- Migration 000045: Duplikaterkennung
-- Kandidaten + Merge-History

CREATE TABLE duplicate_candidates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,  -- REFERENCES tenants(id) when table exists
    entity_type VARCHAR(50) NOT NULL,     -- 'contact', 'company'
    entity_id_a UUID NOT NULL,            -- Erster Kandidat
    entity_id_b UUID NOT NULL,            -- Zweiter Kandidat
    confidence_score DECIMAL(3,2) NOT NULL,  -- 0.00-1.00
    match_fields JSONB NOT NULL,          -- {"email": "exact", "name": "fuzzy", "phone": "partial"}
    status VARCHAR(20) NOT NULL DEFAULT 'pending',  -- 'pending', 'confirmed', 'rejected', 'merged'
    reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_confidence CHECK (confidence_score >= 0 AND confidence_score <= 1),
    CONSTRAINT valid_status CHECK (status IN ('pending', 'confirmed', 'rejected', 'merged')),
    CONSTRAINT different_entities CHECK (entity_id_a != entity_id_b),
    -- Sicherstellen dass Paar nicht doppelt vorkommt (A,B = B,A)
    CONSTRAINT ordered_pair CHECK (entity_id_a < entity_id_b)
);

CREATE INDEX idx_duplicate_candidates_tenant ON duplicate_candidates(tenant_id);
CREATE INDEX idx_duplicate_candidates_status ON duplicate_candidates(status) WHERE status = 'pending';
CREATE INDEX idx_duplicate_candidates_entity_a ON duplicate_candidates(entity_type, entity_id_a);
CREATE INDEX idx_duplicate_candidates_entity_b ON duplicate_candidates(entity_type, entity_id_b);
CREATE UNIQUE INDEX idx_duplicate_candidates_pair ON duplicate_candidates(entity_type, entity_id_a, entity_id_b)
    WHERE status = 'pending';

-- Merge-History: Welcher Record wurde in welchen gemerged
CREATE TABLE merge_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,  -- REFERENCES tenants(id) when table exists
    entity_type VARCHAR(50) NOT NULL,
    surviving_id UUID NOT NULL,           -- Record der bestehen bleibt
    merged_id UUID NOT NULL,              -- Record der aufgeloest wurde
    merged_data JSONB NOT NULL,           -- Snapshot des merged Records VOR dem Merge
    field_decisions JSONB NOT NULL,       -- {"email": "surviving", "phone": "merged", ...}
    merged_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    -- KEIN updated_at: Immutable
);

CREATE INDEX idx_merge_history_tenant ON merge_history(tenant_id);
CREATE INDEX idx_merge_history_surviving ON merge_history(entity_type, surviving_id);
CREATE INDEX idx_merge_history_merged ON merge_history(entity_type, merged_id);
```

---

## 6. E-Mail (IMAP-Cache)

### 6.1 E-Mail-Konten

```sql
-- Migration 000043: E-Mail-System

-- E-Mail-Konten (IMAP/SMTP-Konfiguration pro User)
CREATE TABLE email_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,  -- REFERENCES tenants(id) when table exists
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    -- Anzeigename
    display_name VARCHAR(255) NOT NULL,
    email_address VARCHAR(255) NOT NULL,
    -- IMAP-Konfiguration
    imap_host VARCHAR(255) NOT NULL,
    imap_port INTEGER NOT NULL DEFAULT 993,
    imap_encryption VARCHAR(10) NOT NULL DEFAULT 'ssl',  -- 'ssl', 'starttls', 'none'
    imap_username VARCHAR(255) NOT NULL,
    imap_password_encrypted BYTEA NOT NULL,  -- AES-256-GCM verschluesselt
    -- SMTP-Konfiguration
    smtp_host VARCHAR(255) NOT NULL,
    smtp_port INTEGER NOT NULL DEFAULT 587,
    smtp_encryption VARCHAR(10) NOT NULL DEFAULT 'starttls',
    smtp_username VARCHAR(255) NOT NULL,
    smtp_password_encrypted BYTEA NOT NULL,
    -- Sync-Status
    last_sync_at TIMESTAMPTZ,
    last_sync_error TEXT,
    sync_state JSONB DEFAULT '{}',        -- IMAP sync state: UIDVALIDITY, HIGHESTMODSEQ per folder
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    -- Einstellungen
    signature_html TEXT,
    signature_text TEXT,
    sync_period_days INTEGER NOT NULL DEFAULT 30,  -- Wie weit zurueck syncen
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_imap_encryption CHECK (imap_encryption IN ('ssl', 'starttls', 'none')),
    CONSTRAINT valid_smtp_encryption CHECK (smtp_encryption IN ('ssl', 'starttls', 'none'))
);

CREATE INDEX idx_email_accounts_tenant ON email_accounts(tenant_id);
CREATE INDEX idx_email_accounts_user ON email_accounts(user_id);
CREATE UNIQUE INDEX idx_email_accounts_default ON email_accounts(user_id) WHERE is_default = TRUE;
```

### 6.2 E-Mail-Ordner

```sql
-- E-Mail-Ordner (IMAP-Folder-Cache)
CREATE TABLE email_folders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES email_accounts(id) ON DELETE CASCADE,
    -- IMAP-Daten
    imap_name VARCHAR(500) NOT NULL,      -- Vollstaendiger IMAP-Pfad (z.B. "INBOX.Archiv")
    imap_separator VARCHAR(5) NOT NULL DEFAULT '/',
    imap_attributes JSONB DEFAULT '[]',   -- IMAP folder flags: \Sent, \Drafts, etc.
    uidvalidity BIGINT,                   -- IMAP UIDVALIDITY
    highest_modseq BIGINT,                -- Fuer CONDSTORE-Sync
    -- Anzeige
    display_name VARCHAR(255) NOT NULL,
    folder_type VARCHAR(20) NOT NULL DEFAULT 'custom',  -- 'inbox', 'sent', 'drafts', 'trash', 'spam', 'archive', 'custom'
    parent_id UUID REFERENCES email_folders(id) ON DELETE CASCADE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    -- Statistiken (gecached)
    total_count INTEGER NOT NULL DEFAULT 0,
    unread_count INTEGER NOT NULL DEFAULT 0,
    -- Sync
    last_sync_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_folder_type CHECK (folder_type IN ('inbox', 'sent', 'drafts', 'trash', 'spam', 'archive', 'custom'))
);

CREATE INDEX idx_email_folders_account ON email_folders(account_id);
CREATE INDEX idx_email_folders_parent ON email_folders(parent_id) WHERE parent_id IS NOT NULL;
CREATE UNIQUE INDEX idx_email_folders_imap ON email_folders(account_id, imap_name);
CREATE INDEX idx_email_folders_type ON email_folders(account_id, folder_type);
```

### 6.3 E-Mail-Nachrichten

```sql
-- E-Mail-Nachrichten (IMAP-Message-Cache)
CREATE TABLE email_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES email_accounts(id) ON DELETE CASCADE,
    folder_id UUID NOT NULL REFERENCES email_folders(id) ON DELETE CASCADE,
    -- IMAP-Identifikatoren
    imap_uid BIGINT NOT NULL,             -- IMAP UID innerhalb des Folders
    message_id VARCHAR(500),              -- RFC 2822 Message-ID Header
    in_reply_to VARCHAR(500),             -- Fuer Thread-Zuordnung
    references_header TEXT,               -- References Header (Thread-Chain)
    thread_id UUID,                       -- Berechneter Thread-Identifier (Gruppe)
    -- Header-Daten
    subject TEXT,
    from_address VARCHAR(500) NOT NULL,
    from_name VARCHAR(255),
    to_addresses JSONB NOT NULL DEFAULT '[]',   -- [{"address": "a@b.com", "name": "A B"}, ...]
    cc_addresses JSONB DEFAULT '[]',
    bcc_addresses JSONB DEFAULT '[]',
    reply_to_address VARCHAR(500),
    -- Body
    body_text TEXT,                        -- Plain-Text-Version
    body_html TEXT,                        -- HTML-Version
    snippet TEXT,                          -- Vorschau-Text (max 200 Zeichen)
    -- Flags
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    is_starred BOOLEAN NOT NULL DEFAULT FALSE,
    is_draft BOOLEAN NOT NULL DEFAULT FALSE,
    is_answered BOOLEAN NOT NULL DEFAULT FALSE,
    is_forwarded BOOLEAN NOT NULL DEFAULT FALSE,
    has_attachments BOOLEAN NOT NULL DEFAULT FALSE,
    -- Groesse
    size_bytes INTEGER,
    -- CRM-Verknuepfung
    contact_id UUID REFERENCES contacts(id) ON DELETE SET NULL,  -- Automatisch zugeordnet per E-Mail
    company_id UUID REFERENCES companies(id) ON DELETE SET NULL,
    deal_id UUID REFERENCES deals(id) ON DELETE SET NULL,
    -- Zeitstempel
    sent_at TIMESTAMPTZ,                  -- Date-Header der E-Mail
    received_at TIMESTAMPTZ NOT NULL,     -- IMAP INTERNALDATE
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_email_messages_account ON email_messages(account_id);
CREATE INDEX idx_email_messages_folder ON email_messages(folder_id);
CREATE UNIQUE INDEX idx_email_messages_uid ON email_messages(folder_id, imap_uid);
CREATE INDEX idx_email_messages_message_id ON email_messages(message_id) WHERE message_id IS NOT NULL;
CREATE INDEX idx_email_messages_thread ON email_messages(thread_id) WHERE thread_id IS NOT NULL;
CREATE INDEX idx_email_messages_contact ON email_messages(contact_id) WHERE contact_id IS NOT NULL;
CREATE INDEX idx_email_messages_company ON email_messages(company_id) WHERE company_id IS NOT NULL;
CREATE INDEX idx_email_messages_deal ON email_messages(deal_id) WHERE deal_id IS NOT NULL;
CREATE INDEX idx_email_messages_received ON email_messages(account_id, received_at DESC);
CREATE INDEX idx_email_messages_unread ON email_messages(folder_id, is_read) WHERE is_read = FALSE;
CREATE INDEX idx_email_messages_starred ON email_messages(account_id, is_starred) WHERE is_starred = TRUE;
CREATE INDEX idx_email_messages_from ON email_messages(from_address);

-- Volltextsuche fuer E-Mails
ALTER TABLE email_messages ADD COLUMN search_vector TSVECTOR;
CREATE INDEX idx_email_messages_search ON email_messages USING GIN (search_vector);

CREATE OR REPLACE FUNCTION email_messages_search_vector_update() RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('german', COALESCE(NEW.subject, '')), 'A') ||
        setweight(to_tsvector('german', COALESCE(NEW.from_name, '')), 'B') ||
        setweight(to_tsvector('german', COALESCE(NEW.from_address, '')), 'B') ||
        setweight(to_tsvector('german', COALESCE(NEW.body_text, '')), 'C');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER email_messages_search_update
    BEFORE INSERT OR UPDATE OF subject, from_name, from_address, body_text ON email_messages
    FOR EACH ROW EXECUTE FUNCTION email_messages_search_vector_update();
```

### 6.4 E-Mail-Anhaenge

```sql
-- E-Mail-Anhaenge (Metadaten + optionaler lokaler Cache)
CREATE TABLE email_attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES email_messages(id) ON DELETE CASCADE,
    -- Metadaten
    filename VARCHAR(500) NOT NULL,
    content_type VARCHAR(255) NOT NULL,
    size_bytes INTEGER NOT NULL,
    content_id VARCHAR(255),              -- Fuer Inline-Bilder (CID)
    is_inline BOOLEAN NOT NULL DEFAULT FALSE,
    -- Storage (optional lokal gecached)
    storage_path TEXT,                    -- Pfad im lokalen Storage (S3/MinIO)
    is_cached BOOLEAN NOT NULL DEFAULT FALSE,  -- Ob der Inhalt lokal liegt
    -- IMAP-Referenz
    imap_part_id VARCHAR(50),             -- IMAP BODYSTRUCTURE Part-ID fuer On-Demand-Fetch
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_email_attachments_message ON email_attachments(message_id);
CREATE INDEX idx_email_attachments_cid ON email_attachments(content_id) WHERE content_id IS NOT NULL;
```

---

## 7. Belegkette / Finance

### 7.1 Belegkette (Document Chain)

```sql
-- Migration 000041: Belegkette
-- Angebot -> Auftrag -> Lieferschein -> Rechnung -> Gutschrift
-- State Machine mit Ableitung (derived_from_id)

-- Nummernkreise (fuer lueckenlose Nummerierung)
CREATE TABLE number_sequences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,  -- REFERENCES tenants(id) when table exists
    sequence_type VARCHAR(50) NOT NULL,   -- 'quote', 'order', 'delivery_note', 'invoice', 'credit_note'
    prefix VARCHAR(20) NOT NULL DEFAULT '',  -- z.B. 'RE-', 'AN-', 'LS-'
    current_number BIGINT NOT NULL DEFAULT 0,
    fiscal_year INTEGER,                  -- NULL = fortlaufend, oder Jahr fuer jaehrlichen Reset
    format_pattern VARCHAR(100) NOT NULL DEFAULT '{prefix}{year}-{number:05}',  -- z.B. "RE-2026-00001"
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_sequence_type CHECK (sequence_type IN ('quote', 'order', 'delivery_note', 'invoice', 'credit_note'))
);

CREATE UNIQUE INDEX idx_number_sequences_type ON number_sequences(tenant_id, sequence_type, fiscal_year);

-- Belegkette-Haupttabelle
CREATE TABLE document_chain_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,  -- REFERENCES tenants(id) when table exists
    -- Typ + Nummer
    document_type VARCHAR(20) NOT NULL,   -- 'quote', 'order', 'delivery_note', 'invoice', 'credit_note'
    document_number VARCHAR(50) NOT NULL, -- Formatierte Nummer aus number_sequences
    -- Ableitung (Belegkette)
    derived_from_id UUID REFERENCES document_chain_items(id) ON DELETE RESTRICT,
    root_document_id UUID REFERENCES document_chain_items(id) ON DELETE RESTRICT,  -- Erster Beleg der Kette
    -- Status
    status VARCHAR(30) NOT NULL DEFAULT 'draft',
    -- quote:   draft -> sent -> accepted -> rejected -> expired -> cancelled
    -- order:   draft -> confirmed -> in_progress -> completed -> cancelled
    -- delivery_note: draft -> shipped -> delivered -> cancelled
    -- invoice: draft -> sent -> partially_paid -> paid -> overdue -> cancelled
    -- credit_note: draft -> sent -> applied
    -- Partner
    contact_id UUID REFERENCES contacts(id) ON DELETE SET NULL,
    company_id UUID REFERENCES companies(id) ON DELETE SET NULL,
    -- Adressdaten (Snapshot zum Zeitpunkt der Erstellung)
    billing_address JSONB,                -- {"name": "...", "street": "...", "postal_code": "...", ...}
    shipping_address JSONB,
    -- Betraege (Summen werden aus line_items berechnet und hier gecached)
    subtotal DECIMAL(15,2) NOT NULL DEFAULT 0.00,       -- Netto
    tax_total DECIMAL(15,2) NOT NULL DEFAULT 0.00,      -- MWSt-Summe
    discount_total DECIMAL(15,2) NOT NULL DEFAULT 0.00, -- Rabatt-Summe
    total DECIMAL(15,2) NOT NULL DEFAULT 0.00,           -- Brutto
    currency VARCHAR(3) NOT NULL DEFAULT 'EUR',
    -- Zahlungsbedingungen (nur fuer Rechnungen relevant)
    payment_terms_days INTEGER DEFAULT 30,
    due_date DATE,
    -- Daten
    issued_date DATE,                     -- Belegdatum
    delivered_date DATE,                  -- Lieferdatum
    valid_until DATE,                     -- Gueltig bis (fuer Angebote)
    -- Zusatzinfos
    reference_text VARCHAR(500),          -- Referenz/Bestellnummer des Kunden
    header_text TEXT,                     -- Freitext oben
    footer_text TEXT,                     -- Freitext unten
    internal_notes TEXT,                  -- Interne Notizen (nicht auf dem Beleg)
    -- Metadaten
    assigned_to UUID REFERENCES users(id) ON DELETE SET NULL,
    tags JSONB DEFAULT '[]',
    custom_data JSONB DEFAULT '{}',       -- Fuer Custom Fields
    -- GoBD (siehe Migration 000042)
    is_locked BOOLEAN NOT NULL DEFAULT FALSE,  -- TRUE nach Versand = unveraenderbar
    locked_at TIMESTAMPTZ,
    -- PDF
    pdf_storage_path TEXT,                -- Pfad zur generierten PDF
    pdf_generated_at TIMESTAMPTZ,
    -- Zeitstempel
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sent_at TIMESTAMPTZ,                  -- Wann versendet
    cancelled_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,

    CONSTRAINT valid_document_type CHECK (document_type IN ('quote', 'order', 'delivery_note', 'invoice', 'credit_note')),
    CONSTRAINT valid_status CHECK (status IN (
        'draft', 'sent', 'accepted', 'rejected', 'expired', 'cancelled',
        'confirmed', 'in_progress', 'completed',
        'shipped', 'delivered',
        'partially_paid', 'paid', 'overdue',
        'applied'
    ))
);

CREATE INDEX idx_dci_tenant ON document_chain_items(tenant_id);
CREATE UNIQUE INDEX idx_dci_number ON document_chain_items(tenant_id, document_type, document_number);
CREATE INDEX idx_dci_type_status ON document_chain_items(tenant_id, document_type, status);
CREATE INDEX idx_dci_derived_from ON document_chain_items(derived_from_id) WHERE derived_from_id IS NOT NULL;
CREATE INDEX idx_dci_root ON document_chain_items(root_document_id) WHERE root_document_id IS NOT NULL;
CREATE INDEX idx_dci_contact ON document_chain_items(contact_id) WHERE contact_id IS NOT NULL;
CREATE INDEX idx_dci_company ON document_chain_items(company_id) WHERE company_id IS NOT NULL;
CREATE INDEX idx_dci_assigned ON document_chain_items(assigned_to) WHERE assigned_to IS NOT NULL;
CREATE INDEX idx_dci_due_date ON document_chain_items(due_date) WHERE due_date IS NOT NULL;
CREATE INDEX idx_dci_overdue ON document_chain_items(tenant_id, due_date, status)
    WHERE document_type = 'invoice' AND status = 'sent';
CREATE INDEX idx_dci_created ON document_chain_items(tenant_id, created_at DESC);
```

### 7.2 Belegpositionen (Line Items)

```sql
-- Einzelpositionen auf Belegen
CREATE TABLE document_line_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL REFERENCES document_chain_items(id) ON DELETE CASCADE,
    -- Position
    sort_order INTEGER NOT NULL DEFAULT 0,
    item_type VARCHAR(20) NOT NULL DEFAULT 'item',  -- 'item', 'service', 'text', 'subtotal', 'discount'
    -- Inhalt
    description TEXT NOT NULL,
    detailed_description TEXT,            -- Langtext unter der Position
    -- Artikel-Referenz (optional)
    article_id UUID,                      -- FK zu articles wenn vorhanden (Inventar-Modul)
    article_sku VARCHAR(100),             -- SKU-Snapshot
    -- Menge + Preis
    quantity DECIMAL(12,4) NOT NULL DEFAULT 1,
    unit VARCHAR(20) DEFAULT 'Stk.',      -- 'Stk.', 'h', 'kg', 'm', 'm2', 'm3', 'pauschal'
    unit_price DECIMAL(15,4) NOT NULL DEFAULT 0,  -- Einzelpreis netto
    -- Rabatt
    discount_percent DECIMAL(5,2) DEFAULT 0,
    discount_amount DECIMAL(15,2) DEFAULT 0,   -- Absoluter Rabatt (alternativ zu Prozent)
    -- Steuer
    tax_rate_id UUID REFERENCES tax_rates(id) ON DELETE RESTRICT,
    tax_rate_percent DECIMAL(5,2) NOT NULL DEFAULT 0,  -- Snapshot des Steuersatzes
    tax_amount DECIMAL(15,2) NOT NULL DEFAULT 0,       -- Berechneter Steuerbetrag
    -- Summen
    line_total_net DECIMAL(15,2) NOT NULL DEFAULT 0,   -- Netto (quantity * unit_price - discount)
    line_total_gross DECIMAL(15,2) NOT NULL DEFAULT 0, -- Brutto (net + tax)
    -- Zeiterfassungs-Referenz (fuer Stunden-zu-Rechnung Workflow)
    time_entry_ids JSONB DEFAULT '[]',    -- Array von time_entry UUIDs
    -- Metadaten
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_item_type CHECK (item_type IN ('item', 'service', 'text', 'subtotal', 'discount')),
    CONSTRAINT valid_quantity CHECK (quantity > 0 OR item_type IN ('text', 'subtotal')),
    CONSTRAINT valid_discount CHECK (discount_percent >= 0 AND discount_percent <= 100)
);

CREATE INDEX idx_dli_document ON document_line_items(document_id, sort_order);
CREATE INDEX idx_dli_article ON document_line_items(article_id) WHERE article_id IS NOT NULL;
CREATE INDEX idx_dli_tax_rate ON document_line_items(tax_rate_id) WHERE tax_rate_id IS NOT NULL;
```

### 7.3 Zahlungen

```sql
-- Zahlungseingaenge zu Rechnungen (Teilzahlungen moeglich)
CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,  -- REFERENCES tenants(id) when table exists
    document_id UUID NOT NULL REFERENCES document_chain_items(id) ON DELETE RESTRICT,
    -- Zahlungsdetails
    amount DECIMAL(15,2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'EUR',
    payment_method VARCHAR(30) NOT NULL,  -- 'bank_transfer', 'cash', 'credit_card', 'paypal', 'direct_debit', 'other'
    payment_reference VARCHAR(255),       -- Verwendungszweck / Transaktions-ID
    -- Datum
    payment_date DATE NOT NULL,
    booked_at TIMESTAMPTZ,                -- Wann verbucht
    -- Bank-Zuordnung (fuer FinAPI-Integration spaeter)
    bank_transaction_id VARCHAR(255),
    -- Metadaten
    notes TEXT,
    recorded_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_payment_method CHECK (payment_method IN (
        'bank_transfer', 'cash', 'credit_card', 'paypal', 'direct_debit', 'other'
    )),
    CONSTRAINT positive_amount CHECK (amount > 0)
);

CREATE INDEX idx_payments_tenant ON payments(tenant_id);
CREATE INDEX idx_payments_document ON payments(document_id);
CREATE INDEX idx_payments_date ON payments(payment_date);
CREATE INDEX idx_payments_bank_tx ON payments(bank_transaction_id) WHERE bank_transaction_id IS NOT NULL;
```

### 7.4 Mahnwesen (Dunning)

```sql
-- Mahnwesen: Mahnstufen 1-3 mit Eskalation
CREATE TABLE dunnings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,  -- REFERENCES tenants(id) when table exists
    invoice_id UUID NOT NULL REFERENCES document_chain_items(id) ON DELETE RESTRICT,
    dunning_level INTEGER NOT NULL DEFAULT 1,  -- 1, 2, 3
    dunning_number VARCHAR(50) NOT NULL,
    -- Betraege
    outstanding_amount DECIMAL(15,2) NOT NULL,
    dunning_fee DECIMAL(15,2) NOT NULL DEFAULT 0,     -- Mahngebuehr
    interest_amount DECIMAL(15,2) NOT NULL DEFAULT 0,  -- Verzugszinsen
    total_amount DECIMAL(15,2) NOT NULL,               -- outstanding + fee + interest
    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'draft',  -- 'draft', 'sent', 'paid', 'escalated', 'cancelled'
    -- Fristen
    due_date DATE NOT NULL,
    -- Zeitstempel
    sent_at TIMESTAMPTZ,
    sent_via VARCHAR(20),                 -- 'email', 'letter'
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_dunning_level CHECK (dunning_level BETWEEN 1 AND 3),
    CONSTRAINT valid_dunning_status CHECK (status IN ('draft', 'sent', 'paid', 'escalated', 'cancelled'))
);

CREATE INDEX idx_dunnings_tenant ON dunnings(tenant_id);
CREATE INDEX idx_dunnings_invoice ON dunnings(invoice_id);
CREATE INDEX idx_dunnings_status ON dunnings(tenant_id, status) WHERE status NOT IN ('paid', 'cancelled');
CREATE INDEX idx_dunnings_due ON dunnings(due_date) WHERE status = 'sent';
```

### 7.5 GoBD-Compliance

```sql
-- Migration 000042: GoBD-Compliance
-- Unveraenderbare Rechnungen nach Versand, Trigger verhindert UPDATE

-- Trigger-Funktion: Verhindert Aenderungen an gesperrten Belegen
CREATE OR REPLACE FUNCTION prevent_locked_document_update() RETURNS TRIGGER AS $$
BEGIN
    -- Erlaubt: Nur is_locked, locked_at, pdf_storage_path, pdf_generated_at aendern
    -- (fuer das initiale Sperren und PDF-Update)
    IF OLD.is_locked = TRUE THEN
        -- Pruefen ob nur erlaubte Felder geaendert werden
        IF (
            NEW.document_type != OLD.document_type OR
            NEW.document_number != OLD.document_number OR
            NEW.subtotal != OLD.subtotal OR
            NEW.tax_total != OLD.tax_total OR
            NEW.total != OLD.total OR
            NEW.currency != OLD.currency OR
            NEW.issued_date IS DISTINCT FROM OLD.issued_date OR
            NEW.contact_id IS DISTINCT FROM OLD.contact_id OR
            NEW.company_id IS DISTINCT FROM OLD.company_id OR
            NEW.billing_address IS DISTINCT FROM OLD.billing_address OR
            NEW.header_text IS DISTINCT FROM OLD.header_text OR
            NEW.footer_text IS DISTINCT FROM OLD.footer_text
        ) THEN
            RAISE EXCEPTION 'GoBD: Gesperrter Beleg (%) kann nicht geaendert werden. Storno erstellen statt aendern.',
                OLD.document_number;
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER gobd_lock_check
    BEFORE UPDATE ON document_chain_items
    FOR EACH ROW EXECUTE FUNCTION prevent_locked_document_update();

-- Trigger: Verhindert Loeschung von gesperrten Belegen
CREATE OR REPLACE FUNCTION prevent_locked_document_delete() RETURNS TRIGGER AS $$
BEGIN
    IF OLD.is_locked = TRUE THEN
        RAISE EXCEPTION 'GoBD: Gesperrter Beleg (%) kann nicht geloescht werden. Storno erstellen statt loeschen.',
            OLD.document_number;
    END IF;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER gobd_delete_check
    BEFORE DELETE ON document_chain_items
    FOR EACH ROW EXECUTE FUNCTION prevent_locked_document_delete();

-- Trigger: Verhindert Aenderung von Positionen auf gesperrten Belegen
CREATE OR REPLACE FUNCTION prevent_locked_line_item_change() RETURNS TRIGGER AS $$
DECLARE
    doc_locked BOOLEAN;
BEGIN
    IF TG_OP = 'DELETE' THEN
        SELECT is_locked INTO doc_locked FROM document_chain_items WHERE id = OLD.document_id;
    ELSE
        SELECT is_locked INTO doc_locked FROM document_chain_items WHERE id = NEW.document_id;
    END IF;

    IF doc_locked = TRUE THEN
        RAISE EXCEPTION 'GoBD: Positionen auf gesperrtem Beleg koennen nicht geaendert werden.';
    END IF;

    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER gobd_line_item_check
    BEFORE INSERT OR UPDATE OR DELETE ON document_line_items
    FOR EACH ROW EXECUTE FUNCTION prevent_locked_line_item_change();
```

---

## 8. Helpdesk-Erweiterungen

### 8.1 Canned Responses (Textbausteine)

```sql
-- Migration 000044: Helpdesk-Erweiterungen

CREATE TABLE canned_responses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,  -- REFERENCES tenants(id) when table exists
    -- Inhalt
    title VARCHAR(255) NOT NULL,
    content_text TEXT NOT NULL,            -- Plain-Text-Version
    content_html TEXT,                     -- HTML-Version (fuer Rich-Text)
    -- Kategorisierung
    category VARCHAR(100),                -- z.B. "Begruessung", "Technisch", "Abschluss"
    shortcut VARCHAR(50),                 -- Kuerzel fuer Schnellzugriff: z.B. "/danke"
    -- Variablen (Platzhalter)
    variables JSONB DEFAULT '[]',         -- [{"name": "{{kunde_name}}", "description": "Name des Kunden"}]
    -- Berechtigungen
    is_shared BOOLEAN NOT NULL DEFAULT TRUE,   -- Fuer alle sichtbar oder nur Ersteller
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_canned_responses_tenant ON canned_responses(tenant_id);
CREATE INDEX idx_canned_responses_category ON canned_responses(tenant_id, category);
CREATE UNIQUE INDEX idx_canned_responses_shortcut ON canned_responses(tenant_id, shortcut)
    WHERE shortcut IS NOT NULL;

-- Volltextsuche fuer Textbausteine
ALTER TABLE canned_responses ADD COLUMN search_vector TSVECTOR;
CREATE INDEX idx_canned_responses_search ON canned_responses USING GIN (search_vector);

CREATE OR REPLACE FUNCTION canned_responses_search_update() RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('german', COALESCE(NEW.title, '')), 'A') ||
        setweight(to_tsvector('german', COALESCE(NEW.content_text, '')), 'B') ||
        setweight(to_tsvector('german', COALESCE(NEW.category, '')), 'C');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER canned_responses_search_trigger
    BEFORE INSERT OR UPDATE OF title, content_text, category ON canned_responses
    FOR EACH ROW EXECUTE FUNCTION canned_responses_search_update();
```

### 8.2 Interne Notizen (Private Notes)

```sql
-- Interne Notizen auf Tickets (nicht fuer Kunden sichtbar)
-- Universell: Kann auch fuer andere Entities verwendet werden (deals, contacts, etc.)
CREATE TABLE internal_notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,  -- REFERENCES tenants(id) when table exists
    -- Polymorphe Verknuepfung
    entity_type VARCHAR(50) NOT NULL,     -- 'ticket', 'deal', 'contact', 'company', 'invoice', 'contract'
    entity_id UUID NOT NULL,
    -- Inhalt
    content TEXT NOT NULL,
    content_html TEXT,                    -- Optional: Rich-Text
    -- Metadaten
    is_pinned BOOLEAN NOT NULL DEFAULT FALSE,
    mentioned_user_ids JSONB DEFAULT '[]',  -- User-IDs die @-erwaehnt werden
    -- Zeitstempel
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_internal_notes_tenant ON internal_notes(tenant_id);
CREATE INDEX idx_internal_notes_entity ON internal_notes(entity_type, entity_id);
CREATE INDEX idx_internal_notes_created_by ON internal_notes(created_by);
CREATE INDEX idx_internal_notes_pinned ON internal_notes(entity_type, entity_id)
    WHERE is_pinned = TRUE;
```

---

## 9. Dokumente / File-Sharing

### 9.1 Shared Links (Externer Link-Share)

```sql
-- Migration 000046: File-Sharing

-- Teilbare Links fuer Dateien (aehnlich Dropbox/Nextcloud Share-Links)
CREATE TABLE shared_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,  -- REFERENCES tenants(id) when table exists
    -- Was wird geteilt
    entity_type VARCHAR(50) NOT NULL,     -- 'document', 'folder', 'report', 'form'
    entity_id UUID NOT NULL,
    -- Link-Details
    token VARCHAR(64) NOT NULL,           -- Kryptographisch sicheres Token (URL-Parameter)
    -- Berechtigungen
    permission VARCHAR(10) NOT NULL DEFAULT 'view',  -- 'view', 'download', 'edit'
    -- Schutz
    password_hash VARCHAR(255),           -- Bcrypt-Hash, NULL = kein Passwort
    -- Einschraenkungen
    expires_at TIMESTAMPTZ,               -- NULL = kein Ablauf
    max_downloads INTEGER,                -- NULL = unbegrenzt
    download_count INTEGER NOT NULL DEFAULT 0,
    -- Zugriffsbeschraenkung
    allowed_email_domains JSONB DEFAULT '[]',  -- ["firma.de", "partner.ch"] oder leer = alle
    -- Status
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    -- Metadaten
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_accessed_at TIMESTAMPTZ,

    CONSTRAINT valid_permission CHECK (permission IN ('view', 'download', 'edit'))
);

CREATE UNIQUE INDEX idx_shared_links_token ON shared_links(token);
CREATE INDEX idx_shared_links_tenant ON shared_links(tenant_id);
CREATE INDEX idx_shared_links_entity ON shared_links(entity_type, entity_id);
CREATE INDEX idx_shared_links_active ON shared_links(is_active, expires_at)
    WHERE is_active = TRUE;

-- Zugriffs-Log fuer Shared Links
CREATE TABLE shared_link_access_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shared_link_id UUID NOT NULL REFERENCES shared_links(id) ON DELETE CASCADE,
    -- Zugriffs-Details
    accessed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ip_address INET,
    user_agent TEXT,
    action VARCHAR(20) NOT NULL,          -- 'view', 'download'
    -- Optional: Wer hat zugegriffen (wenn bekannt)
    accessed_by_email VARCHAR(255),
    -- Geo (optional, per IP-Lookup)
    country_code VARCHAR(2)
);

CREATE INDEX idx_sla_log_link ON shared_link_access_log(shared_link_id);
CREATE INDEX idx_sla_log_accessed ON shared_link_access_log(accessed_at DESC);
```

---

## 10. Compliance (DSGVO / GoBD)

### 10.1 Consent-Management

```sql
-- Migration 000047: Consent-Management
-- Einwilligungsflags pro Kontakt pro Zweck (Art. 6/7 DSGVO)

CREATE TABLE consent_purposes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,  -- REFERENCES tenants(id) when table exists
    -- Definition des Zwecks
    name VARCHAR(255) NOT NULL,           -- z.B. "Newsletter", "Telefonmarketing", "Profilbildung"
    slug VARCHAR(100) NOT NULL,           -- URL-tauglicher Identifier
    description TEXT NOT NULL,            -- Erklaerungstext fuer den Kontakt
    legal_basis VARCHAR(30) NOT NULL,     -- 'consent', 'contract', 'legal_obligation', 'legitimate_interest', 'vital_interest', 'public_interest'
    -- Konfiguration
    is_required BOOLEAN NOT NULL DEFAULT FALSE,  -- Pflicht-Einwilligung (z.B. AGB)
    requires_double_opt_in BOOLEAN NOT NULL DEFAULT FALSE,
    -- Status
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_legal_basis CHECK (legal_basis IN (
        'consent', 'contract', 'legal_obligation', 'legitimate_interest', 'vital_interest', 'public_interest'
    ))
);

CREATE UNIQUE INDEX idx_consent_purposes_slug ON consent_purposes(tenant_id, slug);
CREATE INDEX idx_consent_purposes_tenant ON consent_purposes(tenant_id);

-- Einwilligungen pro Kontakt pro Zweck (mit History)
CREATE TABLE consents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,  -- REFERENCES tenants(id) when table exists
    contact_id UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    purpose_id UUID NOT NULL REFERENCES consent_purposes(id) ON DELETE RESTRICT,
    -- Status
    status VARCHAR(20) NOT NULL,          -- 'granted', 'revoked', 'expired', 'pending_double_opt_in'
    -- Quelle
    source VARCHAR(50) NOT NULL,          -- 'web_form', 'email', 'phone', 'in_person', 'import', 'api'
    source_details TEXT,                  -- z.B. URL des Formulars, Name des Anrufers
    ip_address INET,                      -- IP bei Web-Einwilligung
    -- Zeitstempel
    granted_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,               -- Optional: Automatischer Ablauf
    -- Double Opt-In
    doi_token VARCHAR(100),               -- Token fuer Double-Opt-In-Bestaetigung
    doi_confirmed_at TIMESTAMPTZ,
    -- Metadaten
    recorded_by UUID REFERENCES users(id) ON DELETE SET NULL,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- KEIN updated_at: Jede Aenderung ist ein NEUER Record (History)

    CONSTRAINT valid_consent_status CHECK (status IN ('granted', 'revoked', 'expired', 'pending_double_opt_in'))
);

CREATE INDEX idx_consents_tenant ON consents(tenant_id);
CREATE INDEX idx_consents_contact ON consents(contact_id);
CREATE INDEX idx_consents_purpose ON consents(purpose_id);
CREATE INDEX idx_consents_contact_purpose ON consents(contact_id, purpose_id, created_at DESC);
CREATE INDEX idx_consents_status ON consents(tenant_id, status) WHERE status = 'granted';
CREATE INDEX idx_consents_doi_token ON consents(doi_token) WHERE doi_token IS NOT NULL;
```

### 10.2 Retention Policies (Aufbewahrungsfristen)

```sql
-- Migration 000048: Retention Policies
-- Automatische Berechnung von Aufbewahrungsfristen nach Land und Dokumenttyp

CREATE TABLE retention_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,  -- REFERENCES tenants(id) when table exists
    -- Regel-Definition
    country VARCHAR(2) NOT NULL,          -- 'DE', 'CH', 'AT'
    document_category VARCHAR(50) NOT NULL,  -- 'invoice', 'business_letter', 'personnel_file', 'time_record', 'contract'
    entity_type VARCHAR(50) NOT NULL,     -- 'document_chain_item', 'email_message', 'contact', 'time_entry', 'contract'
    -- Fristen
    retention_years INTEGER NOT NULL,     -- Aufbewahrungsfrist in Jahren
    retention_start VARCHAR(30) NOT NULL DEFAULT 'end_of_fiscal_year',  -- Ab wann zaehlt die Frist
    -- 'end_of_fiscal_year' = Ende des Geschaeftsjahres in dem erstellt
    -- 'document_date' = Ab Belegdatum
    -- 'employment_end' = Ab Austritt des Mitarbeiters
    -- Aktion nach Ablauf
    action_after_expiry VARCHAR(20) NOT NULL DEFAULT 'flag',  -- 'flag', 'anonymize', 'delete'
    -- Gesetzliche Grundlage
    legal_reference VARCHAR(255),         -- z.B. "HGB §257", "AO §147", "OR Art. 958f"
    description TEXT,
    -- Status
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_retention_start CHECK (retention_start IN ('end_of_fiscal_year', 'document_date', 'employment_end')),
    CONSTRAINT valid_expiry_action CHECK (action_after_expiry IN ('flag', 'anonymize', 'delete'))
);

CREATE INDEX idx_retention_rules_tenant ON retention_rules(tenant_id);
CREATE INDEX idx_retention_rules_country ON retention_rules(country);
CREATE UNIQUE INDEX idx_retention_rules_unique ON retention_rules(tenant_id, country, document_category, entity_type);

-- Seed: Standard-Aufbewahrungsfristen DACH
/*
-- Deutschland
('DE', 'invoice', 'document_chain_item', 10, 'end_of_fiscal_year', 'flag', 'HGB §257 Abs. 4, AO §147 Abs. 3')
('DE', 'business_letter', 'email_message', 6, 'end_of_fiscal_year', 'flag', 'HGB §257 Abs. 4')
('DE', 'time_record', 'time_entry', 2, 'end_of_fiscal_year', 'flag', 'ArbZG §16 Abs. 2')
('DE', 'personnel_file', 'contact', 9, 'employment_end', 'anonymize', 'Allgemein: 3J + 6J Lohn')
('DE', 'contract', 'document_chain_item', 10, 'end_of_fiscal_year', 'flag', 'HGB §257')

-- Schweiz
('CH', 'invoice', 'document_chain_item', 10, 'end_of_fiscal_year', 'flag', 'OR Art. 958f')
('CH', 'business_letter', 'email_message', 10, 'end_of_fiscal_year', 'flag', 'OR Art. 958f')
('CH', 'personnel_file', 'contact', 15, 'employment_end', 'anonymize', '5J + 10J Lohnausweise')
('CH', 'contract', 'document_chain_item', 10, 'end_of_fiscal_year', 'flag', 'OR Art. 958f')

-- Oesterreich
('AT', 'invoice', 'document_chain_item', 7, 'end_of_fiscal_year', 'flag', 'BAO §132')
('AT', 'business_letter', 'email_message', 7, 'end_of_fiscal_year', 'flag', 'BAO §132')
('AT', 'personnel_file', 'contact', 10, 'employment_end', 'anonymize', '3J + 7J')
('AT', 'contract', 'document_chain_item', 7, 'end_of_fiscal_year', 'flag', 'BAO §132')
*/
```

---

## 11. Integrationen

### 11.1 DATEV-Export

```sql
-- Migration 000049: DATEV-Export

-- Export-Batches (ein Batch = ein DATEV-Export-Lauf)
CREATE TABLE datev_export_batches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,  -- REFERENCES tenants(id) when table exists
    -- Export-Konfiguration
    export_type VARCHAR(30) NOT NULL,     -- 'buchungsstapel', 'debitoren_kreditoren', 'sachkonten'
    date_from DATE NOT NULL,
    date_to DATE NOT NULL,
    fiscal_year INTEGER NOT NULL,
    -- DATEV-spezifisch
    berater_nummer VARCHAR(10),           -- DATEV Beraternummer
    mandanten_nummer VARCHAR(10),         -- DATEV Mandantennummer
    wirtschaftsjahr_beginn DATE,          -- Beginn Wirtschaftsjahr
    sachkonten_laenge INTEGER NOT NULL DEFAULT 4,  -- 4-8 Stellen
    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'pending',  -- 'pending', 'generating', 'completed', 'failed'
    error_message TEXT,
    -- Ergebnis
    file_path TEXT,                       -- Pfad zur generierten CSV-Datei
    file_size_bytes INTEGER,
    entry_count INTEGER DEFAULT 0,        -- Anzahl Buchungssaetze
    -- Encoding: DATEV erfordert Windows-1252!
    encoding VARCHAR(20) NOT NULL DEFAULT 'windows-1252',
    -- Zeitstempel
    generated_by UUID NOT NULL REFERENCES users(id),
    generated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_export_type CHECK (export_type IN ('buchungsstapel', 'debitoren_kreditoren', 'sachkonten')),
    CONSTRAINT valid_export_status CHECK (status IN ('pending', 'generating', 'completed', 'failed')),
    CONSTRAINT valid_date_range CHECK (date_to >= date_from)
);

CREATE INDEX idx_datev_batches_tenant ON datev_export_batches(tenant_id);
CREATE INDEX idx_datev_batches_status ON datev_export_batches(status);
CREATE INDEX idx_datev_batches_date ON datev_export_batches(tenant_id, date_from, date_to);

-- Einzelne Buchungssaetze in einem Export-Batch
CREATE TABLE datev_export_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id UUID NOT NULL REFERENCES datev_export_batches(id) ON DELETE CASCADE,
    -- Quell-Referenz
    source_type VARCHAR(50) NOT NULL,     -- 'invoice', 'credit_note', 'payment', 'expense', 'time_entry'
    source_id UUID NOT NULL,              -- ID des Quell-Records
    -- DATEV-Buchungssatz-Felder (Buchungsstapel-Format)
    umsatz DECIMAL(15,2) NOT NULL,        -- Betrag
    soll_haben VARCHAR(1) NOT NULL,       -- 'S' oder 'H'
    konto VARCHAR(10) NOT NULL,           -- Sachkonto
    gegenkonto VARCHAR(10) NOT NULL,      -- Gegenkonto
    belegdatum DATE NOT NULL,
    buchungstext VARCHAR(60),             -- Max 60 Zeichen (DATEV-Limit!)
    belegnummer VARCHAR(36),              -- Max 36 Zeichen
    bu_schluessel VARCHAR(4),             -- Buchungsschluessel (USt-Kennzeichen)
    -- Status
    is_exported BOOLEAN NOT NULL DEFAULT FALSE,
    export_line_number INTEGER,           -- Zeilennummer in der CSV
    -- Zeitstempel
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_datev_entries_batch ON datev_export_entries(batch_id);
CREATE INDEX idx_datev_entries_source ON datev_export_entries(source_type, source_id);
```

### 11.2 Integration Connections

```sql
-- Migration 000050: Integration-Konfigurationen

-- Generische Integrations-Konfiguration pro Tenant
CREATE TABLE integration_connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,  -- REFERENCES tenants(id) when table exists
    -- Integration
    provider VARCHAR(50) NOT NULL,        -- 'bexio', 'skribble', 'brevo', 'cleverreach', 'onlyoffice',
                                          -- 'nextcloud', 'finapi', 'zefix', 'datev'
    display_name VARCHAR(255) NOT NULL,   -- Benutzerdefinierter Name
    -- Authentifizierung (verschluesselt)
    auth_type VARCHAR(20) NOT NULL,       -- 'oauth2', 'api_key', 'basic', 'token', 'webhook'
    credentials_encrypted JSONB NOT NULL, -- AES-256-GCM verschluesselt: {"access_token": "...", "refresh_token": "...", ...}
    -- OAuth2-spezifisch
    oauth_token_url TEXT,
    oauth_scopes TEXT,
    oauth_expires_at TIMESTAMPTZ,
    -- Konfiguration
    settings JSONB NOT NULL DEFAULT '{}', -- Provider-spezifische Einstellungen
    -- z.B. Bexio: {"default_account_id": "...", "sync_contacts": true}
    -- z.B. Nextcloud: {"base_url": "https://...", "default_folder": "/KMU Hub"}
    -- Sync-Status
    last_sync_at TIMESTAMPTZ,
    last_sync_status VARCHAR(20),         -- 'success', 'partial', 'failed'
    last_sync_error TEXT,
    sync_cursor JSONB DEFAULT '{}',       -- Cursor/Offset fuer inkrementellen Sync
    -- Webhook (fuer eingehende Events)
    webhook_secret VARCHAR(255),
    webhook_url TEXT,                     -- Generierte URL fuer diesen Tenant
    -- Status
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    -- Zeitstempel
    connected_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_auth_type CHECK (auth_type IN ('oauth2', 'api_key', 'basic', 'token', 'webhook'))
);

CREATE INDEX idx_integration_connections_tenant ON integration_connections(tenant_id);
CREATE INDEX idx_integration_connections_provider ON integration_connections(tenant_id, provider);
CREATE UNIQUE INDEX idx_integration_connections_active ON integration_connections(tenant_id, provider)
    WHERE is_active = TRUE;

-- Sync-Log: Protokolliert jeden Sync-Lauf
CREATE TABLE integration_sync_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    connection_id UUID NOT NULL REFERENCES integration_connections(id) ON DELETE CASCADE,
    -- Sync-Details
    sync_type VARCHAR(30) NOT NULL,       -- 'full', 'incremental', 'webhook', 'manual'
    direction VARCHAR(10) NOT NULL,       -- 'inbound', 'outbound', 'bidirectional'
    entity_type VARCHAR(50),              -- 'contact', 'invoice', etc.
    -- Ergebnis
    status VARCHAR(20) NOT NULL,          -- 'running', 'success', 'partial', 'failed'
    records_processed INTEGER DEFAULT 0,
    records_created INTEGER DEFAULT 0,
    records_updated INTEGER DEFAULT 0,
    records_failed INTEGER DEFAULT 0,
    error_details JSONB,                  -- Array von Fehlern: [{"record_id": "...", "error": "..."}]
    -- Timing
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    duration_ms INTEGER
);

CREATE INDEX idx_sync_log_connection ON integration_sync_log(connection_id);
CREATE INDEX idx_sync_log_started ON integration_sync_log(started_at DESC);
CREATE INDEX idx_sync_log_status ON integration_sync_log(status) WHERE status IN ('running', 'failed');

-- Entity-Mapping: Zuordnung externer IDs zu internen IDs
CREATE TABLE integration_entity_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    connection_id UUID NOT NULL REFERENCES integration_connections(id) ON DELETE CASCADE,
    -- Intern
    internal_entity_type VARCHAR(50) NOT NULL,
    internal_entity_id UUID NOT NULL,
    -- Extern
    external_entity_type VARCHAR(100) NOT NULL,  -- Provider-spezifischer Typ
    external_entity_id VARCHAR(255) NOT NULL,    -- ID im externen System
    -- Sync-State
    last_synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    external_updated_at TIMESTAMPTZ,      -- Letzter Aenderungszeitpunkt im externen System
    sync_hash VARCHAR(64),                -- SHA-256 Hash fuer Change Detection
    -- Metadaten
    metadata JSONB DEFAULT '{}',          -- Zusaetzliche Mapping-Daten
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_entity_mappings_internal ON integration_entity_mappings(
    connection_id, internal_entity_type, internal_entity_id
);
CREATE UNIQUE INDEX idx_entity_mappings_external ON integration_entity_mappings(
    connection_id, external_entity_type, external_entity_id
);
```

---

## 12. Entity-Relationship-Uebersicht

### Querschnitts-Beziehungen

```
tenants 1---N users (ueber tenant_members)
tenants 1---N [alle Business-Tabellen via tenant_id]

users 1---N audit_entries
users 1---N email_accounts
users 1---N canned_responses
users 1---N internal_notes
users 1---N shared_links
users 1---N integration_connections
```

### CRM-Beziehungen

```
companies 1---N contacts (bestehend: contact.company_id)
companies 1---N companies (Self-Reference: parent_company_id)
contacts N---M companies (NEU: contact_company_roles mit Rolle)
contacts 1---N consents
contacts 1---N email_messages (Zuordnung per E-Mail-Adresse)

duplicate_candidates: Verknuepft je 2 contacts ODER 2 companies
merge_history: Dokumentiert welcher Record in welchen gemerged wurde
```

### E-Mail-Beziehungen

```
email_accounts 1---N email_folders
email_folders 1---N email_messages
email_messages 1---N email_attachments
email_messages N---1 contacts (auto-Zuordnung)
email_messages N---1 companies (auto-Zuordnung)
email_messages N---1 deals (manuelle Zuordnung)
```

### Belegkette-Beziehungen

```
document_chain_items ---self--- document_chain_items (derived_from_id, root_document_id)
document_chain_items 1---N document_line_items
document_chain_items 1---N payments
document_chain_items 1---N dunnings
document_chain_items N---1 contacts
document_chain_items N---1 companies
document_line_items N---1 tax_rates
document_line_items N---0..1 articles (Inventar-Referenz)
number_sequences: Pro Tenant + document_type
```

### Compliance-Beziehungen

```
consent_purposes 1---N consents
contacts 1---N consents
retention_rules: Pro Tenant x Country x document_category
audit_entries: Universelle Verknuepfung ueber entity_type + entity_id
```

### Integration-Beziehungen

```
integration_connections 1---N integration_sync_log
integration_connections 1---N integration_entity_mappings
datev_export_batches 1---N datev_export_entries
datev_export_entries ---ref--- document_chain_items / payments / etc. (via source_type + source_id)
```

---

## 13. PostgreSQL-spezifische Features

### 13.1 Verwendete Features

| Feature | Wo | Warum |
|---------|-----|-------|
| **JSONB** | `custom_field_values.value`, `email_messages.to_addresses`, `audit_entries.changes`, `integration_connections.credentials_encrypted`, `document_chain_items.billing_address` | Flexible Schema-lose Daten. GIN-Indexierbar fuer Suche. |
| **GIN-Index** | `search_vector` (TSVECTOR), `audit_entries.changes` (JSONB) | Performante Volltextsuche und JSONB-Queries. |
| **Partial Indexes** | `idx_email_messages_unread WHERE is_read = FALSE`, `idx_dci_overdue WHERE status = 'sent'` | Reduziert Index-Groesse, beschleunigt haeufige Queries. |
| **TSVECTOR** | E-Mail-Suche, Canned-Response-Suche | Deutsche Stemming-Analyse fuer Volltextsuche. |
| **TIMESTAMPTZ** | Alle Zeitstempel | Timezone-aware. Essentiell fuer DACH (CET/CEST). |
| **INET** | `audit_entries.ip_address`, `consents.ip_address` | Nativer PostgreSQL-Typ fuer IP-Adressen (IPv4+IPv6). |
| **BYTEA** | `email_accounts.*_password_encrypted` | Verschluesselte Binaerdaten (AES-256-GCM). |
| **CHECK Constraints** | Enum-Validierung (status, types) | Application-Level + DB-Level Validierung. |
| **UNIQUE WHERE** | `idx_email_accounts_default WHERE is_default = TRUE` | Stellt sicher: maximal 1 Default pro User. |
| **Trigger** | GoBD: `prevent_locked_document_update`, FTS: `*_search_update` | Datenkonsistenz auf DB-Level (nicht nur App-Level). |
| **DECIMAL(15,2)** / **DECIMAL(15,4)** | Alle Geldbetraege / Einzelpreise | Exakte Dezimalarithmetik (kein FLOAT fuer Geld!). |
| **Self-Referencing FK** | `companies.parent_company_id`, `document_chain_items.derived_from_id` | Hierarchien und Ketten innerhalb einer Tabelle. |
| **ON DELETE RESTRICT** | `tax_rates`, `consent_purposes`, `document_chain_items` | Verhindert versehentliches Loeschen von referenzierten Stammdaten. |

### 13.2 RLS (Row-Level Security) — Spaetere Migration

```sql
-- NICHT in den initialen Migrations, sondern wenn Multi-Tenancy aktiviert wird.
-- Hier als Referenz fuer Luke:

-- Aktivierung
ALTER TABLE document_chain_items ENABLE ROW LEVEL SECURITY;

-- Policy: User sieht nur eigenen Tenant
CREATE POLICY tenant_isolation ON document_chain_items
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);

-- Fuer jede neue Tabelle mit tenant_id analog:
-- ALTER TABLE {table} ENABLE ROW LEVEL SECURITY;
-- CREATE POLICY tenant_isolation ON {table}
--     USING (tenant_id = current_setting('app.current_tenant_id')::UUID);
```

### 13.3 Empfohlene PostgreSQL-Extensions

```sql
-- Bereits vorhanden:
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";  -- uuid_generate_v4() (Migration 000001)
-- ALTERNATIV: gen_random_uuid() (PG 13+, keine Extension noetig)

-- Empfohlen fuer spaeter:
CREATE EXTENSION IF NOT EXISTS "pg_trgm";    -- Trigram-Index fuer Fuzzy-Suche (Duplikaterkennung!)
CREATE EXTENSION IF NOT EXISTS "btree_gin";  -- Combined GIN+BTree Indexes
CREATE EXTENSION IF NOT EXISTS "pgcrypto";   -- Encryption-Funktionen
```

### 13.4 Partitionierung (fuer grosse Tabellen — spaeter)

```sql
-- Empfohlen ab >10M Rows:
-- audit_entries: Partitionierung nach created_at (monatlich)
-- email_messages: Partitionierung nach received_at (monatlich)
-- integration_sync_log: Partitionierung nach started_at (monatlich)

-- Beispiel:
-- CREATE TABLE audit_entries (...)
--     PARTITION BY RANGE (created_at);
-- CREATE TABLE audit_entries_2026_01 PARTITION OF audit_entries
--     FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');
```

---

## Zusammenfassung: Neue Tabellen

| # | Tabelle | Migrations-Nr | Abhaengigkeiten |
|---|---------|---------------|-----------------|
| 1 | `tenants` | 000036 | Keine (optional) |
| 2 | `tenant_members` | 000036 | `tenants`, `users` |
| 3 | `ticket_custom_field_values` | 000037 | `custom_field_definitions` |
| 4 | `project_custom_field_values` | 000037 | `projects`, `custom_field_definitions` |
| 5 | `contact_company_roles` | 000038 | `contacts`, `companies` |
| 6 | `audit_entries` | 000039 | `users` |
| 7 | `tax_rates` | 000040 | Keine |
| 8 | `number_sequences` | 000041 | Keine |
| 9 | `document_chain_items` | 000041 | `contacts`, `companies`, `users` |
| 10 | `document_line_items` | 000041 | `document_chain_items`, `tax_rates` |
| 11 | `payments` | 000041 | `document_chain_items`, `users` |
| 12 | `dunnings` | 000041 | `document_chain_items`, `users` |
| 13 | `email_accounts` | 000043 | `users` |
| 14 | `email_folders` | 000043 | `email_accounts` |
| 15 | `email_messages` | 000043 | `email_folders`, `email_accounts`, `contacts`, `companies`, `deals` |
| 16 | `email_attachments` | 000043 | `email_messages` |
| 17 | `canned_responses` | 000044 | `users` |
| 18 | `internal_notes` | 000044 | `users` |
| 19 | `duplicate_candidates` | 000045 | `users` |
| 20 | `merge_history` | 000045 | `users` |
| 21 | `shared_links` | 000046 | `users` |
| 22 | `shared_link_access_log` | 000046 | `shared_links` |
| 23 | `consent_purposes` | 000047 | Keine |
| 24 | `consents` | 000047 | `contacts`, `consent_purposes`, `users` |
| 25 | `retention_rules` | 000048 | Keine |
| 26 | `datev_export_batches` | 000049 | `users` |
| 27 | `datev_export_entries` | 000049 | `datev_export_batches` |
| 28 | `integration_connections` | 000050 | `users` |
| 29 | `integration_sync_log` | 000050 | `integration_connections` |
| 30 | `integration_entity_mappings` | 000050 | `integration_connections` |

**Total: 30 neue Tabellen** + ALTER TABLE fuer `contacts` (9 Spalten) + `companies` (14 Spalten) + `custom_field_definitions` (Constraint) + GoBD-Trigger (3 Trigger-Funktionen)

---

## Nicht-abgedeckt (werden in separaten Dokumenten behandelt)

Die ~45 DB-Tabellen aus dem Backend-Audit (Inventar, Schichten, Einkauf, Helpdesk-Basis, Fuhrpark, Produktion, Berichte, Vertraege, Formulare, Vermietung, Rapporte, Lohn, Schulungen, Wiki) sind im `BACKEND-REQUIREMENTS-AUDIT.md` Abschnitt "DB Tables Needed" aufgelistet. Diese sind modul-spezifisch und werden in den jeweiligen Modul-Migrations entworfen, wenn Luke die Services baut.

Dieses Dokument fokussiert auf die **Querschnittsfunktionen und kritischen Feature-Luecken** aus der Synthese (`00-SYNTHESE.md`, Abschnitt 2: "Was FEHLT").

---

*Erstellt: 2026-02-17 | Grundlage: 00-SYNTHESE.md + BACKEND-REQUIREMENTS-AUDIT.md + Bestehende Migrations 000001-000035*

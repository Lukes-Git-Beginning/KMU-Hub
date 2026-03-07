# Backend-Plan Wave 3: CRM + Finanzen

> **Von:** Darien (Frontend) | **Fuer:** Luke (Backend)
> **Datum:** 2026-02-21 | **Status:** Frontend DONE (Commit `6ee1921`)
> **Geschaetzter Backend-Aufwand:** ~3.100 LOC Go

---

## Uebersicht

Wave 3 ist der groesste Backend-Brocken. CRM braucht vollstaendige CRUD-Operationen,
Custom Fields und Cross-Modul-Queries. Finanzen braucht GoBD-Compliance, PDF-Generierung,
DATEV-Export und Swiss QR-Bill.

**Was das Frontend aktuell macht:** Alles mit Zustand Stores (Mock-Daten). Beim Merge
werden `useContactStore`, `useFinanceStore` etc. durch TanStack Query Hooks ersetzt.

---

## A. CRM Backend-Anforderungen

### A1. CRM CRUD (Item 3.1) — PRIO HOCH

Das Frontend hat Create/Edit-Dialoge fuer Kontakte, Firmen und Deals gebaut.
Aktuell schreiben sie in den Zustand Store. Luke braucht vollstaendige CRUD-Endpoints.

**Endpoints:**

```
POST   /api/v1/contacts              — Kontakt erstellen
PUT    /api/v1/contacts/:id          — Kontakt aktualisieren
DELETE /api/v1/contacts/:id          — Kontakt loeschen (Soft-Delete wegen DSGVO)
GET    /api/v1/contacts              — Liste (existiert bereits)
GET    /api/v1/contacts/:id          — Detail (existiert bereits)

POST   /api/v1/companies             — Firma erstellen
PUT    /api/v1/companies/:id         — Firma aktualisieren
DELETE /api/v1/companies/:id         — Firma loeschen
GET    /api/v1/companies             — Liste
GET    /api/v1/companies/:id         — Detail

POST   /api/v1/deals                 — Deal erstellen
PUT    /api/v1/deals/:id             — Deal aktualisieren
DELETE /api/v1/deals/:id             — Deal loeschen
PATCH  /api/v1/deals/:id/stage       — Deal-Phase aendern (Kanban-Drag)
GET    /api/v1/deals                 — Liste (mit Pipeline-Filter)
GET    /api/v1/deals/:id             — Detail
```

**Request Body — Contact Create/Update:**
```json
{
  "first_name": "Max",
  "last_name": "Mustermann",
  "email": "max@firma.de",
  "phone": "+49 170 1234567",
  "mobile": "+49 170 9876543",
  "company_id": "uuid-or-null",
  "position": "Geschaeftsfuehrer",
  "salutation": "Herr",
  "academic_title": "Dr.",
  "preferred_language": "de",
  "address": {
    "street": "Musterstrasse 1",
    "zip": "80331",
    "city": "Muenchen",
    "country": "DE"
  },
  "tags": ["VIP", "Partner"],
  "source": "website",
  "notes": "Kennt uns von der Messe",
  "custom_fields": {
    "field_def_id_1": "Wert 1",
    "field_def_id_2": 42
  }
}
```

**Response Body — Contact:**
```json
{
  "id": "uuid",
  "first_name": "Max",
  "last_name": "Mustermann",
  "email": "max@firma.de",
  "phone": "+49 170 1234567",
  "mobile": "+49 170 9876543",
  "company_id": "uuid-or-null",
  "company_name": "Firma GmbH",
  "position": "Geschaeftsfuehrer",
  "salutation": "Herr",
  "academic_title": "Dr.",
  "preferred_language": "de",
  "address": { "street": "...", "zip": "...", "city": "...", "country": "DE" },
  "tags": ["VIP", "Partner"],
  "source": "website",
  "notes": "...",
  "custom_fields": { "field_def_id_1": "Wert 1" },
  "created_at": "2026-02-21T10:00:00Z",
  "updated_at": "2026-02-21T10:00:00Z"
}
```

**Request Body — Deal Create:**
```json
{
  "title": "Website Redesign",
  "value": 15000.00,
  "currency": "EUR",
  "stage": "qualification",
  "probability": 60,
  "contact_id": "uuid",
  "company_id": "uuid-or-null",
  "expected_close_date": "2026-04-15",
  "assigned_to": "user-uuid",
  "tags": ["Website"],
  "notes": "Erstgespraech war positiv"
}
```

**Deal Stages (Pipeline):**
```
lead → qualification → proposal → negotiation → closed_won | closed_lost
```

---

### A2. Custom Fields (Item 3.2) — PRIO HOCH

Das Frontend hat einen Admin-UI fuer Feld-Definitionen + dynamisches Rendering.
Backend braucht JSONB-Spalte + Definition-Management.

**DB-Schema:**
```sql
CREATE TABLE custom_field_definitions (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id   UUID NOT NULL REFERENCES tenants(id),
  entity_type VARCHAR(50) NOT NULL,  -- 'contact', 'company', 'deal', 'ticket'
  field_name  VARCHAR(100) NOT NULL,
  field_type  VARCHAR(20) NOT NULL,  -- 'text', 'number', 'date', 'dropdown', 'checkbox', 'url'
  field_label VARCHAR(200) NOT NULL,
  options     JSONB,                  -- fuer dropdown: ["Option A", "Option B"]
  required    BOOLEAN DEFAULT FALSE,
  sort_order  INTEGER DEFAULT 0,
  created_at  TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE(tenant_id, entity_type, field_name)
);

-- JSONB-Spalte in contacts, companies, deals:
ALTER TABLE contacts ADD COLUMN custom_fields JSONB DEFAULT '{}';
ALTER TABLE companies ADD COLUMN custom_fields JSONB DEFAULT '{}';
ALTER TABLE deals ADD COLUMN custom_fields JSONB DEFAULT '{}';
```

**Endpoints:**
```
GET    /api/v1/custom-fields?entity_type=contact    — Alle Definitionen
POST   /api/v1/custom-fields                         — Definition erstellen
PUT    /api/v1/custom-fields/:id                     — Definition aendern
DELETE /api/v1/custom-fields/:id                     — Definition loeschen
```

**Validierung:** Backend muss Custom-Field-Werte gegen Definitionen validieren
bei Contact/Company/Deal Create/Update.

---

### A3. Firma als eigene Entity (Item 3.3) — PRIO MITTEL

Falls `companies`-Tabelle noch nicht existiert:

**DB-Schema:**
```sql
CREATE TABLE companies (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id    UUID NOT NULL REFERENCES tenants(id),
  name         VARCHAR(300) NOT NULL,
  industry     VARCHAR(100),
  website      VARCHAR(500),
  logo_url     VARCHAR(500),
  address      JSONB,
  phone        VARCHAR(50),
  email        VARCHAR(200),
  employee_count INTEGER,
  annual_revenue DECIMAL(15,2),
  tags         TEXT[],
  custom_fields JSONB DEFAULT '{}',
  notes        TEXT,
  created_at   TIMESTAMPTZ DEFAULT NOW(),
  updated_at   TIMESTAMPTZ DEFAULT NOW()
);

-- Kontakt-Firma-Beziehung:
ALTER TABLE contacts ADD COLUMN company_id UUID REFERENCES companies(id);
CREATE INDEX idx_contacts_company ON contacts(company_id);
```

---

### A4. Duplikaterkennung (Item 3.4) — PRIO NIEDRIG

**Endpoint:**
```
POST /api/v1/contacts/check-duplicates
```

**Request:**
```json
{
  "email": "max@firma.de",
  "phone": "+49 170 1234567",
  "first_name": "Max",
  "last_name": "Mustermann"
}
```

**Response:**
```json
{
  "duplicates": [
    {
      "contact_id": "uuid",
      "contact_name": "Max Mustermann",
      "match_type": "email_exact",
      "confidence": 0.95,
      "matched_fields": ["email"]
    }
  ]
}
```

**Matching-Logik:**
1. E-Mail: Exakter Match (Confidence 0.95)
2. Telefon: Normalisierter Match (nur Ziffern vergleichen, Confidence 0.85)
3. Name: Trigram-Similarity (`pg_trgm` Extension, Threshold 0.4, Confidence = Similarity)

---

### A5. Kontakt-Timeline (Item 3.5) — PRIO MITTEL

**Endpoint:**
```
GET /api/v1/contacts/:id/timeline?page=1&page_size=20&type=all
```

**Response:**
```json
{
  "items": [
    {
      "id": "uuid",
      "type": "email",
      "title": "E-Mail gesendet: Angebot Website",
      "description": "An max@firma.de",
      "timestamp": "2026-02-20T14:30:00Z",
      "metadata": { "email_id": "uuid" }
    },
    {
      "type": "deal",
      "title": "Deal erstellt: Website Redesign",
      "description": "EUR 15.000, Phase: Qualifikation",
      "timestamp": "2026-02-19T10:00:00Z",
      "metadata": { "deal_id": "uuid" }
    }
  ],
  "total": 47,
  "page": 1,
  "page_size": 20
}
```

**Types:** `email`, `deal`, `ticket`, `meeting`, `note`, `call`, `task`, `invoice`

**Implementation:** Cross-Service Query. Entweder:
- Option A: Jeder Service hat `GET /api/v1/{service}/by-contact/:contactId` und das Gateway aggregiert
- Option B: Materialized View / Event-Sourced Timeline-Tabelle

---

### A6. Consent-Management (Item 3.7) — PRIO NIEDRIG

**DB-Schema:**
```sql
CREATE TABLE consent_records (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id  UUID NOT NULL,
  contact_id UUID NOT NULL REFERENCES contacts(id),
  purpose    VARCHAR(50) NOT NULL,  -- 'email_marketing', 'phone', 'post', 'profiling'
  granted    BOOLEAN NOT NULL,
  source     VARCHAR(100),          -- 'form', 'verbal', 'email', 'import'
  granted_at TIMESTAMPTZ,
  revoked_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_consent_contact ON consent_records(contact_id);
```

**Endpoints:**
```
GET  /api/v1/contacts/:id/consents        — Alle Einwilligungen eines Kontakts
POST /api/v1/contacts/:id/consents        — Einwilligung erfassen
PUT  /api/v1/contacts/:id/consents/:cid   — Einwilligung widerrufen
```

---

### A7. Import/Export (Item 3.9) — PRIO MITTEL

**Endpoints:**
```
POST /api/v1/contacts/import          — CSV/vCard Bulk-Import
GET  /api/v1/contacts/export?format=csv|vcf  — Export
```

**Import Request:** Multipart form mit:
- `file`: CSV oder .vcf Datei
- `field_mapping`: JSON mit Spalten-Mapping (z.B. `{"Vorname": "first_name", "E-Mail": "email"}`)
- `duplicate_action`: `skip` | `update` | `create_anyway`

**Import Response:**
```json
{
  "imported": 142,
  "skipped": 3,
  "errors": [
    { "row": 17, "error": "Ungueltige E-Mail-Adresse" }
  ]
}
```

---

## B. Finanzen Backend-Anforderungen

### B1. GoBD-Compliance (Item 3.17) — PRIO HOCH

**Kritisch fuer deutsche Kunden!** Grundsaetze ordnungsmaessiger digitaler Buchfuehrung.

**Anforderungen:**
1. **Unveraenderbarkeit:** Gesendete Rechnungen duerfen NICHT geaendert oder geloescht werden
2. **Storno statt Delete:** `DELETE /api/v1/invoices/:id` gibt 403 zurueck. Stattdessen: `POST /api/v1/invoices/:id/cancel` erstellt Storno-Rechnung
3. **Lueckenlose Nummern:** Rechnungsnummern muessen fortlaufend sein (keine Luecken!)
4. **Audit-Log:** Jede Aktion auf einer Rechnung wird protokolliert

**DB-Erweiterungen:**
```sql
-- Rechnungs-Status erweitern:
-- 'draft' → 'sent' → 'paid' | 'overdue' | 'cancelled'
-- Nur 'draft' darf bearbeitet werden!

ALTER TABLE invoices ADD COLUMN is_locked BOOLEAN DEFAULT FALSE;
ALTER TABLE invoices ADD COLUMN cancelled_at TIMESTAMPTZ;
ALTER TABLE invoices ADD COLUMN cancel_reason TEXT;
ALTER TABLE invoices ADD COLUMN original_invoice_id UUID REFERENCES invoices(id);

-- Audit-Log:
CREATE TABLE invoice_audit_log (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id   UUID NOT NULL,
  invoice_id  UUID NOT NULL REFERENCES invoices(id),
  action      VARCHAR(50) NOT NULL,  -- 'created', 'sent', 'paid', 'cancelled', 'reminded'
  performed_by UUID NOT NULL,
  details     JSONB,
  created_at  TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_invoice_audit ON invoice_audit_log(invoice_id);

-- Nummern-Generator (DB-Sequence fuer lueckenlose Nummern):
CREATE SEQUENCE invoice_number_seq START 1;
```

**Neue Endpoints:**
```
POST /api/v1/invoices/:id/cancel     — Rechnung stornieren (erstellt Storno-Beleg)
POST /api/v1/invoices/:id/send       — Rechnung als "gesendet" markieren (sperrt Bearbeitung)
GET  /api/v1/invoices/:id/audit-log  — Aenderungsprotokoll abrufen
```

**Cancel Request:**
```json
{
  "reason": "Falsche Artikelpreise"
}
```

**Cancel Response:**
```json
{
  "cancellation_invoice": {
    "id": "new-uuid",
    "number": "2026-S-0042",
    "original_invoice_id": "original-uuid",
    "type": "cancellation",
    "total": -1500.00
  }
}
```

---

### B2. PDF-Generierung (Item 3.18) — PRIO HOCH

**Endpoint:**
```
GET /api/v1/invoices/:id/pdf         — PDF herunterladen
GET /api/v1/invoices/:id/pdf/preview — PDF als Base64 fuer Inline-Vorschau
```

**Empfohlene Libraries (Go):**
- `chromedp` (Headless Chrome) — bestes Layout, aber schwer
- `wkhtmltopdf` — gut, braucht Binary
- `go-pdf` / `gofpdf` — rein Go, aber manuelles Layout
- `maroto` — Go PDF mit flexiblem Layout (empfohlen)

**Template-Felder:**
- Firmen-Logo + Adresse (Absender)
- Kunden-Adresse
- Rechnungsnummer, Datum, Faelligkeitsdatum
- Positionen (Beschreibung, Menge, Einzelpreis, MWSt-Satz, Gesamtpreis)
- Zwischensumme, MWSt-Betrag, Gesamtbetrag
- Zahlungsbedingungen
- Bankverbindung (IBAN, BIC)
- Optional: Swiss QR-Code (siehe B3)
- Footer mit Handelsregister/USt-ID

---

### B3. Swiss QR-Bill (Item 3.13) — PRIO MITTEL

Schweizer QR-Rechnung nach SIX-Standard (ISO 20022). Pflicht seit 2022 fuer CH-Rechnungen.

**Generierung:** Als Teil der PDF (B2) oder separat:
```
GET /api/v1/invoices/:id/qr-bill     — QR-Bill Daten + QR-Code Image
```

**Response:**
```json
{
  "qr_code_base64": "data:image/png;base64,...",
  "creditor": {
    "iban": "CH93 0076 2011 6238 5295 7",
    "name": "Firma GmbH",
    "address": "Musterstrasse 1, 8000 Zuerich"
  },
  "amount": 1500.00,
  "currency": "CHF",
  "reference_type": "QRR",
  "reference": "21 00000 00003 13947 14300 09017",
  "additional_info": "Rechnung 2026-0042"
}
```

**Go Library:** `github.com/nicovince/qr-bill` oder `github.com/krepost/swiss-qr-bill`

---

### B4. DATEV-Export (Item 3.12) — PRIO HOCH

**Endpoint:**
```
GET /api/v1/finance/export/datev?from=2026-01-01&to=2026-03-31&type=invoices
```

**Response:** CSV-Datei (Windows-1252 Encoding, Semikolon-Separator)

**DATEV-Format-Spezifikation:**
- Header-Zeile mit DATEV-Metadaten (Format-Version, Berater-Nr, Mandanten-Nr)
- Buchungssaetze: Umsatz, Soll/Haben-Konto, BU-Schluessel, Belegdatum, Belegfeld1, Buchungstext
- Konten-Mapping: Erloes-Konto (z.B. 8400), Debitor-Konto (z.B. 10000-69999)

**Wichtig:** DATEV-Format ist streng spezifiziert. Empfehlung: Referenz-Dateien von
einem Steuerberater holen und dagegen testen.

---

### B5. Belegkette (Item 3.11) — PRIO MITTEL

Verknuepfung: Angebot → Auftrag → Lieferschein → Rechnung → Mahnung

**DB-Schema:**
```sql
-- Dokument-Typ-Erweiterung:
-- invoices.type: 'invoice', 'quote', 'order', 'delivery_note', 'credit_note', 'reminder'

ALTER TABLE invoices ADD COLUMN parent_document_id UUID REFERENCES invoices(id);
ALTER TABLE invoices ADD COLUMN document_type VARCHAR(30) DEFAULT 'invoice';
CREATE INDEX idx_invoices_parent ON invoices(parent_document_id);
```

**Konvertierungs-Endpoint:**
```
POST /api/v1/documents/:id/convert
```

**Request:**
```json
{
  "target_type": "invoice",
  "copy_positions": true,
  "adjust_date": true
}
```

**Response:** Das neu erstellte Dokument (Rechnung basierend auf Angebot).

---

### B6. ZUGFeRD/XRechnung (Item 3.14) — PRIO NIEDRIG

Ab 2025 muessen DE-Unternehmen E-Rechnungen empfangen koennen (EN 16931).
Ab 2027/2028 Versandpflicht.

**Was gebraucht wird:**
- XML-Attachment im Factur-X/ZUGFeRD-Format in Rechnungs-PDF einbetten
- Profile: Minimum, Basic, Comfort, Extended
- Library: `github.com/nicovince/factur-x` oder manuell XML generieren

Kann spaeter kommen, aber das Schema sollte schon die noetige Info speichern:
```sql
ALTER TABLE invoices ADD COLUMN e_invoice_profile VARCHAR(20);  -- 'minimum', 'basic', 'comfort', 'extended'
ALTER TABLE invoices ADD COLUMN e_invoice_xml TEXT;
```

---

### B7. Stunden-zu-Rechnung (Item 3.19) — PRIO MITTEL

Cross-Modul: Zeiteintraege aus Zeiterfassung → Rechnungspositionen.

**Endpoint:**
```
POST /api/v1/invoices/from-timeentries
```

**Request:**
```json
{
  "time_entry_ids": ["uuid1", "uuid2", "uuid3"],
  "hourly_rate": 120.00,
  "currency": "EUR",
  "contact_id": "uuid",
  "group_by": "task"  // 'task' | 'date' | 'none'
}
```

**Response:** Neu erstellte Rechnung mit Positionen.

**Logik:**
- Jeder Zeiteintrag wird zu einer Rechnungsposition
- `group_by: "task"` → Eintraege pro Task zusammenfassen (Gesamtstunden x Satz)
- Zeiteintraege werden als "abgerechnet" markiert (`invoiced_at`, `invoice_id`)

---

### B8. FinAPI Banking (Item 3.20) — PRIO NIEDRIG

Automatischer Bankabgleich ueber FinAPI (oder Open Banking API).

**Was spaeter gebraucht wird:**
- FinAPI-Account Registration
- Bank-Connection aufbauen (BankID + Credentials)
- Transaktionen abrufen
- Matching-Algorithmus: Rechnung ↔ Transaktion (Betrag, Verwendungszweck, Referenz)

**Fuer jetzt:** Nur Platzhalter im Frontend. Backend kann spaeter kommen.

---

## C. Zusammenfassung: Empfohlene Reihenfolge

| Prio | Item | Was | Aufwand |
|------|------|-----|---------|
| 1 | B1 | GoBD-Compliance (unveraenderbare Records, Storno, Audit-Log) | ~400 LOC |
| 2 | A1 | CRM CRUD (POST/PUT/DELETE fuer Kontakte, Firmen, Deals) | ~500 LOC |
| 3 | B2 | PDF-Generierung (Rechnungs-PDF) | ~400 LOC |
| 4 | B4 | DATEV-Export | ~300 LOC |
| 5 | A2 | Custom Fields (JSONB + Definitionen) | ~300 LOC |
| 6 | B5 | Belegkette (Konvertierung Angebot→Rechnung) | ~250 LOC |
| 7 | B3 | Swiss QR-Bill | ~200 LOC |
| 8 | A5 | Kontakt-Timeline (Cross-Modul-Query) | ~300 LOC |
| 9 | B7 | Stunden-zu-Rechnung | ~200 LOC |
| 10 | A7 | Import/Export (CSV, vCard) | ~200 LOC |
| 11 | A4 | Duplikaterkennung | ~150 LOC |
| 12 | A6 | Consent-Management | ~100 LOC |
| 13 | B6 | ZUGFeRD/XRechnung | ~200 LOC |
| 14 | B8 | FinAPI Banking | ~300 LOC |

**Total: ~3.800 LOC Go** (etwas mehr als die urspruengliche Schaetzung)

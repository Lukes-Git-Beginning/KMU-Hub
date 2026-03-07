# Wave 13 API Contract: DSGVO + KI Features

> **From:** Darien (Frontend) | **For:** Luke (Backend)
> **Date:** 2026-02-22 | **Status:** Entwurf
> **Frontend branch:** `design/brainstorm`
> **Estimated backend effort:** ~2,200 LOC Go

---

## Overview

Wave 13 adds two major feature areas: **DSGVO-Compliance** (GDPR) and **KI-Assistenz** (AI). The DSGVO features ensure legal compliance for DACH-KMUs including Art. 15 data subject access requests, retention policies, and consent tracking. The KI features add AI-powered productivity tools across modules with a governance framework for per-tenant control.

**Frontend will build:**
- DSGVO: DSAR search interface (cross-module person search), data export (JSON/CSV), retention policy dashboard (read-only), consent log viewer
- KI: AI draft generation (emails, documents), meeting summarization, ticket suggestion (auto-categorize), semantic search (embedding-based), document classification, AI activity log
- KI Governance: Per-tenant AI toggle, per-module opt-out, activity logging dashboard

---

## A. DSGVO — DSAR (Data Subject Access Request, Art. 15)

### 13.1 Person Search (Cross-Module)

```
GET    /api/v1/dsgvo/person-search                        -> Personen-Suche ueber alle Module
```

Query params: `?query=max+mustermann&email=max@example.com&phone=+491701234567`

Response:
```json
{
  "results": [
    {
      "personId": "uuid",
      "displayName": "Max Mustermann",
      "email": "max@example.com",
      "phone": "+49 170 1234567",
      "sources": [
        {
          "module": "crm",
          "entityType": "contact",
          "entityId": "uuid",
          "lastUpdated": "2026-02-10T14:30:00Z"
        },
        {
          "module": "helpdesk",
          "entityType": "ticket_requester",
          "entityId": "uuid",
          "lastUpdated": "2026-01-20T10:00:00Z"
        },
        {
          "module": "chat",
          "entityType": "external_participant",
          "entityId": "uuid",
          "lastUpdated": "2026-02-05T09:00:00Z"
        },
        {
          "module": "finanzen",
          "entityType": "invoice_recipient",
          "entityId": "uuid",
          "lastUpdated": "2026-02-18T11:00:00Z"
        }
      ]
    }
  ],
  "total": 1
}
```

**Backend-Logik:**
- Sucht ueber CRM-Kontakte, Helpdesk-Kunden, Chat-Teilnehmer, Rechnungsempfaenger, Vertragpartner, Booking-Gaeste
- Suche nach Name (fuzzy), E-Mail (exakt), Telefon (normalisiert)
- `sources` zeigt alle Module in denen Daten dieser Person existieren
- `personId` ist eine interne Referenz (kann ein CRM-Kontakt-UUID sein oder ein generierter Key)
- **Recht:** Nur Admins und Datenschutzbeauftragte (neue Rolle: `privacy_officer`)

---

### 13.2 Data Aggregation (DSAR-Export)

```
POST   /api/v1/dsgvo/dsar                                -> DSAR starten
GET    /api/v1/dsgvo/dsar/{dsarId}                        -> DSAR-Status abfragen
GET    /api/v1/dsgvo/dsar/{dsarId}/download                -> Export herunterladen
```

DSAR Request:
```json
{
  "personId": "uuid",
  "requestedBy": "max@example.com",
  "requestDate": "2026-02-22",
  "modules": ["crm", "helpdesk", "finanzen", "chat"],
  "format": "json|csv",
  "includeMetadata": true
}
```

DSAR Status Response:
```json
{
  "dsarId": "uuid",
  "status": "pending|processing|completed|error",
  "personName": "Max Mustermann",
  "requestDate": "2026-02-22",
  "completedAt": "2026-02-22T15:00:00Z|null",
  "moduleStatuses": {
    "crm": "completed",
    "helpdesk": "completed",
    "finanzen": "processing",
    "chat": "pending"
  },
  "downloadUrl": "/api/v1/dsgvo/dsar/uuid/download|null",
  "expiresAt": "2026-03-22T00:00:00Z"
}
```

Download Response:
- `format=json`: `Content-Type: application/json`
- `format=csv`: `Content-Type: application/zip` (ZIP mit einer CSV pro Modul)
- `Content-Disposition: attachment; filename="DSAR_Max_Mustermann_2026-02-22.json"`

JSON Export Structure:
```json
{
  "metadata": {
    "exportDate": "2026-02-22T15:00:00Z",
    "requestedBy": "max@example.com",
    "personName": "Max Mustermann",
    "personEmail": "max@example.com"
  },
  "crm": {
    "contacts": [
      {
        "id": "uuid",
        "name": "Max Mustermann",
        "email": "max@example.com",
        "phone": "+49 170 1234567",
        "company": "Mustermann GmbH",
        "tags": ["Bestandskunde"],
        "createdAt": "2025-06-15",
        "customFields": { "branche": "IT" }
      }
    ],
    "deals": [
      {
        "id": "uuid",
        "title": "CRM-Projekt",
        "value": 25000,
        "stage": "Verhandlung",
        "contactRef": "uuid"
      }
    ],
    "activities": [
      {
        "id": "uuid",
        "type": "email",
        "subject": "Angebot CRM-Projekt",
        "date": "2026-01-15T10:00:00Z"
      }
    ]
  },
  "helpdesk": {
    "tickets": [
      {
        "id": "uuid",
        "subject": "Login-Problem",
        "status": "closed",
        "createdAt": "2026-01-20T10:00:00Z",
        "messages": [
          {
            "sender": "max@example.com",
            "body": "Ich kann mich nicht einloggen.",
            "sentAt": "2026-01-20T10:00:00Z"
          }
        ]
      }
    ]
  },
  "finanzen": {
    "invoices": [
      {
        "id": "uuid",
        "number": "2026-048",
        "amount": 12500,
        "currency": "EUR",
        "status": "paid",
        "date": "2026-02-14"
      }
    ]
  },
  "chat": {
    "messages": [
      {
        "channelName": "Projekt Alpha",
        "messageCount": 23,
        "firstMessage": "2026-01-05T09:00:00Z",
        "lastMessage": "2026-02-05T14:00:00Z"
      }
    ]
  }
}
```

**Backend-Logik:**
- DSAR ist ein asynchroner Prozess (kann Minuten dauern bei grossen Datenmengen)
- Cross-Service-Queries an alle relevanten Module
- `modules` Array bestimmt welche Module abgefragt werden
- Chat-Nachrichten: Nur Metadaten (Channel, Anzahl, Zeitraum), nicht den vollstaendigen Inhalt (Datenschutz Dritter)
- Export-Datei temporaer speichern (30 Tage Aufbewahrung)
- `expiresAt` nach 30 Tagen — dann wird die Datei geloescht
- DSAR-Request wird im Audit-Log protokolliert
- **Frist:** Art. 15 DSGVO: Antwort innerhalb 1 Monat

---

### 13.3 Person Data Deletion (Art. 17 — Recht auf Loeschung)

```
POST   /api/v1/dsgvo/delete-request                       -> Loeschantrag stellen
GET    /api/v1/dsgvo/delete-request/{id}                  -> Status abfragen
POST   /api/v1/dsgvo/delete-request/{id}/execute           -> Loeschung durchfuehren
```

Delete Request:
```json
{
  "personId": "uuid",
  "reason": "Widerruf der Einwilligung",
  "modules": ["crm", "helpdesk", "chat"],
  "excludeModules": ["finanzen"],
  "requestedBy": "datenschutz@firma.de"
}
```

Status Response:
```json
{
  "id": "uuid",
  "personName": "Max Mustermann",
  "status": "pending_approval|approved|executing|completed|rejected",
  "requestDate": "2026-02-22",
  "modules": {
    "crm": { "status": "pending", "recordCount": 3 },
    "helpdesk": { "status": "pending", "recordCount": 1 },
    "chat": { "status": "pending", "recordCount": 23 }
  },
  "excludedModules": {
    "finanzen": { "reason": "Gesetzliche Aufbewahrungspflicht (GoBD, 10 Jahre)" }
  },
  "approvedBy": "admin-uuid|null",
  "executedAt": "ISO-8601|null"
}
```

**Backend-Logik:**
- Loeschung benoetigt Genehmigung durch Admin/Privacy Officer
- `excludeModules` fuer Module mit gesetzlicher Aufbewahrungspflicht (z.B. Finanzen: 10 Jahre GoBD)
- Bei Ausfuehrung: Soft-Delete mit Anonymisierung (Name -> "Geloeschter Kontakt", E-Mail -> Hash)
- Referenzen in anderen Modulen: Anonymisieren statt Loeschen (z.B. Deal-Historie bleibt, aber Kontaktname wird anonymisiert)
- Audit-Log: Vollstaendiges Protokoll der Loeschung (wer, wann, was, warum)
- Chat-Nachrichten: Inhalte loeschen, Metadaten behalten ("Nachricht geloescht")

---

### 13.4 Retention Policies (Aufbewahrungsfristen)

```
GET    /api/v1/dsgvo/retention-policies                   -> Alle Aufbewahrungsfristen
PUT    /api/v1/dsgvo/retention-policies/{id}              -> Frist aktualisieren (Admin)
```

Response:
```json
{
  "policies": [
    {
      "id": "uuid",
      "module": "finanzen",
      "dataCategory": "Rechnungen",
      "retentionDays": 3650,
      "legalBasis": "GoBD § 147 AO — 10 Jahre Aufbewahrungspflicht",
      "autoDeleteEnabled": false,
      "description": "Steuerlich relevante Belege muessen 10 Jahre aufbewahrt werden"
    },
    {
      "id": "uuid",
      "module": "crm",
      "dataCategory": "Kontaktdaten",
      "retentionDays": 1095,
      "legalBasis": "Art. 6 Abs. 1 lit. f DSGVO — Berechtigtes Interesse",
      "autoDeleteEnabled": true,
      "description": "Kontaktdaten werden nach 3 Jahren Inaktivitaet automatisch geloescht"
    },
    {
      "id": "uuid",
      "module": "helpdesk",
      "dataCategory": "Support-Tickets",
      "retentionDays": 730,
      "legalBasis": "Art. 6 Abs. 1 lit. b DSGVO — Vertragserfuellung",
      "autoDeleteEnabled": true,
      "description": "Geschlossene Tickets werden nach 2 Jahren geloescht"
    },
    {
      "id": "uuid",
      "module": "chat",
      "dataCategory": "Chat-Nachrichten",
      "retentionDays": 365,
      "legalBasis": "Art. 6 Abs. 1 lit. f DSGVO — Berechtigtes Interesse",
      "autoDeleteEnabled": false,
      "description": "Chat-Nachrichten werden nach 1 Jahr archiviert"
    },
    {
      "id": "uuid",
      "module": "team",
      "dataCategory": "Personalakten",
      "retentionDays": 3650,
      "legalBasis": "§ 257 HGB, § 147 AO",
      "autoDeleteEnabled": false,
      "description": "Personalunterlagen 10 Jahre nach Austritt"
    }
  ]
}
```

**Backend-Logik:**
- Seed-Daten mit deutschen Aufbewahrungsfristen
- `autoDeleteEnabled`: Wenn true, loescht ein Cron-Job automatisch abgelaufene Daten
- PUT nur fuer Admins (Fristen anpassen, z.B. autoDelete ein/aus, Tage aendern)
- Automatische Loeschung: Cron-Job (taeglich) prueft Daten aelter als `retentionDays` und loescht/archiviert

---

### 13.5 Consent Log (Einwilligungsprotokoll)

```
GET    /api/v1/dsgvo/consents                             -> Alle Einwilligungen
POST   /api/v1/dsgvo/consents                             -> Einwilligung erfassen
DELETE /api/v1/dsgvo/consents/{id}                        -> Einwilligung widerrufen
```

Query params (GET): `?personId=uuid&purpose=marketing|analytics|profiling&status=active|revoked&page=1&pageSize=20`

Consent model:
```json
{
  "id": "uuid",
  "personId": "uuid",
  "personName": "Max Mustermann",
  "personEmail": "max@example.com",
  "purpose": "marketing|analytics|profiling|data_processing|newsletter",
  "legalBasis": "Art. 6 Abs. 1 lit. a DSGVO — Einwilligung",
  "status": "active|revoked",
  "givenAt": "2026-01-15T10:00:00Z",
  "revokedAt": "ISO-8601|null",
  "source": "website_form|email|verbal|contract",
  "ipAddress": "192.168.1.1|null",
  "notes": "Einwilligung zum Newsletter-Versand"
}
```

**Backend-Logik:**
- Einwilligungen sind immutable (nicht editierbar, nur widerrufen)
- Widerruf: `status` auf `revoked`, `revokedAt` setzen — alter Eintrag bleibt bestehen (Audit-Trail)
- Bei oeffentlichen Formularen (Wave 10): Automatisch Consent-Eintrag erstellen
- `source` dokumentiert wie die Einwilligung eingeholt wurde
- `ipAddress` nur bei Online-Einwilligungen speichern

---

### 13.6 DSGVO Audit Log

```
GET    /api/v1/dsgvo/audit-log                            -> Protokoll aller DSGVO-Aktionen
```

Query params: `?action=dsar_created|data_deleted|consent_given|consent_revoked|policy_changed&dateFrom=&dateTo=&page=1&pageSize=50`

Response:
```json
{
  "entries": [
    {
      "id": "uuid",
      "action": "dsar_created|dsar_completed|data_deleted|consent_given|consent_revoked|policy_changed|person_searched",
      "actor": "user-uuid",
      "actorName": "Admin Mueller",
      "personId": "uuid|null",
      "personName": "Max Mustermann|null",
      "details": "DSAR fuer Max Mustermann gestartet (Module: crm, helpdesk, finanzen)",
      "timestamp": "2026-02-22T14:30:00Z"
    }
  ],
  "total": 89
}
```

**Backend-Logik:**
- Jede DSGVO-relevante Aktion wird automatisch protokolliert
- Audit-Log ist immutable (keine Loeschung moeglich)
- Retention: 6 Jahre (Art. 5 Abs. 2 DSGVO Rechenschaftspflicht)

---

## B. KI-Assistenz (AI Proxy)

### 13.7 AI Draft Generation

```
POST   /api/v1/ai/generate                               -> KI-Text generieren
```

Request:
```json
{
  "type": "email_reply|email_compose|document_draft|ticket_response|meeting_agenda",
  "context": {
    "moduleId": "kommunikation|dokumente|helpdesk|meetings",
    "entityId": "uuid|null",
    "language": "de|en",
    "tone": "formal|neutral|casual",
    "maxLength": 500
  },
  "prompt": "Antworte auf diese Kundenanfrage zum Liefertermin",
  "previousMessages": [
    {
      "role": "customer",
      "content": "Wann wird meine Bestellung geliefert?"
    }
  ]
}
```

Response:
```json
{
  "id": "uuid",
  "content": "Sehr geehrte/r Herr/Frau Mustermann,\n\nvielen Dank fuer Ihre Anfrage...",
  "type": "email_reply",
  "tokensUsed": 245,
  "model": "gpt-4o|claude-3-opus",
  "createdAt": "ISO-8601"
}
```

**Backend-Logik:**
- **AI Proxy:** Backend leitet Anfragen an den konfigurierten AI-Provider weiter (OpenAI, Anthropic, oder Self-Hosted)
- `context.entityId` laedt relevanten Kontext aus der DB (z.B. vorherige E-Mails, Ticket-Historie)
- `previousMessages` fuer Konversationskontext
- `tokensUsed` fuer Usage-Tracking und Abrechnung
- **EU-Datensouveraenitaet:** AI-Anfragen duerfen NUR an EU-Endpoints gehen (Azure OpenAI EU, oder Self-Hosted LLM)
- System-Prompt wird serverseitig hinzugefuegt (nicht im Request, damit der User ihn nicht manipulieren kann)
- Rate Limiting: Pro Tenant, z.B. 100 Anfragen pro Stunde (konfigurierbar)

---

### 13.8 Meeting Summarization

```
POST   /api/v1/ai/summarize-meeting                       -> Meeting zusammenfassen
```

Request:
```json
{
  "meetingId": "uuid",
  "transcriptSource": "notes|recording",
  "outputFormat": "summary|action_items|both",
  "language": "de"
}
```

Response:
```json
{
  "meetingId": "uuid",
  "summary": "Im woechentlichen Standup wurden folgende Punkte besprochen:\n1. Sprint-Review: ...",
  "actionItems": [
    {
      "description": "Angebot fuer Projekt Alpha fertigstellen",
      "assignee": "Lisa Mueller",
      "deadline": "2026-02-28"
    },
    {
      "description": "Bug im Login-Formular fixen",
      "assignee": "Thomas Keller",
      "deadline": "2026-02-25"
    }
  ],
  "tokensUsed": 580,
  "createdAt": "ISO-8601"
}
```

**Backend-Logik:**
- `transcriptSource: notes`: Nutzt die Meeting-Notes (HTML/Text aus TipTap, Wave 10)
- `transcriptSource: recording`: Nutzt eine Transkription der Aufnahme (erfordert Speech-to-Text Integration)
- Action Items werden als strukturierte Daten extrahiert (AI Structured Output)
- Optional: Action Items automatisch als Tasks im Projektmodul erstellen

---

### 13.9 Ticket Suggestion (Auto-Kategorisierung)

```
POST   /api/v1/ai/suggest-ticket                          -> Ticket-Kategorisierung vorschlagen
```

Request:
```json
{
  "ticketId": "uuid",
  "subject": "Drucker druckt nicht mehr",
  "body": "Seit heute Morgen funktioniert der Drucker im 2. Stock nicht mehr. Error Code E-04."
}
```

Response:
```json
{
  "ticketId": "uuid",
  "suggestions": {
    "category": "Hardware",
    "priority": "high",
    "tags": ["drucker", "hardware-defekt"],
    "assigneeGroupId": "uuid",
    "assigneeGroupName": "IT-Support",
    "suggestedResponse": "Vielen Dank fuer Ihre Meldung. Der Fehlercode E-04 deutet auf einen Papierstau hin. Bitte pruefen Sie..."
  },
  "confidence": 0.87,
  "tokensUsed": 120
}
```

**Backend-Logik:**
- Analysiert Subject + Body und schlaegt Kategorie, Prioritaet, Tags und Zustaendigkeitsgruppe vor
- `confidence` Score (0-1) gibt die Zuversicht der KI an
- Vorschlaege werden NICHT automatisch angewendet — der Helpdesk-Agent entscheidet
- Training: Basiert auf historischen Ticket-Daten des Tenants (optional: Fine-Tuning)

---

### 13.10 Semantic Search (Embedding-basiert)

```
POST   /api/v1/ai/search                                 -> Semantische Suche
```

Request:
```json
{
  "query": "Wann haben wir zuletzt ueber Preiserhoehungen gesprochen?",
  "modules": ["chat", "email", "dokumente", "crm"],
  "limit": 10,
  "dateFrom": "2026-01-01",
  "dateTo": "2026-02-22"
}
```

Response:
```json
{
  "results": [
    {
      "id": "uuid",
      "module": "chat",
      "entityType": "message",
      "title": "Nachricht in #vertrieb",
      "snippet": "...wir sollten die Preise im Q2 um 5% erhoehen, besonders bei...",
      "relevanceScore": 0.92,
      "date": "2026-02-10T14:30:00Z",
      "deepLink": "/chat/channel-uuid?msg=uuid",
      "author": "Lisa Mueller"
    },
    {
      "id": "uuid",
      "module": "dokumente",
      "entityType": "file",
      "title": "Preisliste_2026_v2.xlsx",
      "snippet": "Aktualisierte Preisliste mit 5% Erhoehung ab April 2026",
      "relevanceScore": 0.85,
      "date": "2026-02-08T10:00:00Z",
      "deepLink": "/dokumente?file=uuid",
      "author": "Thomas Keller"
    }
  ],
  "tokensUsed": 80,
  "searchTime": 340
}
```

**BACKEND-DEP: Embedding-Pipeline + Vektor-DB.**

**Backend-Logik:**
- **Embedding-Pipeline:** Bei neuen/aktualisierten Dokumenten, Nachrichten, E-Mails: Text embedden und in Vektor-DB speichern
- **Vektor-DB:** pgvector (PostgreSQL Extension) oder Qdrant/Milvus
- **Query:** User-Query embedden, Nearest-Neighbor-Suche in Vektor-DB, Ergebnisse nach Relevanz sortieren
- `modules` Filter: Nur in bestimmten Modulen suchen
- `snippet`: Relevanter Textausschnitt (Kontext um den Match-Punkt)
- `searchTime` in Millisekunden
- **Datenschutz:** Nur Dokumente/Nachrichten indexieren, auf die der suchende User Zugriff hat (ACL-Check)

---

### 13.11 Document Classification

```
POST   /api/v1/ai/classify-document                       -> Dokument klassifizieren
```

Request:
```json
{
  "fileId": "uuid",
  "fileName": "Arbeitsvertrag_Mueller.pdf"
}
```

Response:
```json
{
  "fileId": "uuid",
  "classification": "vertraulich|intern|oeffentlich",
  "confidence": 0.94,
  "suggestedTags": ["vertrag", "personal", "HR"],
  "containsPII": true,
  "piiTypes": ["name", "address", "salary", "social_security_number"],
  "tokensUsed": 200
}
```

**Backend-Logik:**
- Analysiert Dateiname + Inhalt (bei Text/PDF-Dateien) oder nur Dateiname (bei Bildern/Videos)
- `classification` Stufen:
  - `oeffentlich`: Keine sensiblen Daten
  - `intern`: Geschaeftlich, aber nicht personenbezogen
  - `vertraulich`: Personenbezogene oder geschaeftskritische Daten
- `containsPII`: True wenn personenbezogene Daten erkannt werden
- `piiTypes`: Welche Arten von PII gefunden wurden
- Ergebnis wird auf dem Dokument gespeichert (neues Feld `classificationLevel` auf `document_files`)
- **Batch-Klassifizierung:** Optional ein Cron-Job der alle nicht-klassifizierten Dokumente durchgeht

---

## C. KI Governance

### 13.12 Tenant AI Settings

```
GET    /api/v1/ai/settings                                -> KI-Einstellungen des Tenants
PUT    /api/v1/ai/settings                                -> KI-Einstellungen aktualisieren
```

Settings model:
```json
{
  "aiEnabled": true,
  "provider": "openai_eu|anthropic_eu|self_hosted",
  "modelPreference": "fast|balanced|quality",
  "monthlyBudgetEur": 50.00,
  "currentMonthUsageEur": 12.45,
  "moduleSettings": {
    "chat": { "enabled": true, "features": ["draft", "search"] },
    "helpdesk": { "enabled": true, "features": ["suggest", "draft", "search"] },
    "dokumente": { "enabled": true, "features": ["classify", "search"] },
    "meetings": { "enabled": false, "features": [] },
    "email": { "enabled": true, "features": ["draft", "search"] }
  },
  "dataProcessingConsent": true,
  "consentDate": "2026-01-15T10:00:00Z"
}
```

**Backend-Logik:**
- `aiEnabled: false` deaktiviert ALLE KI-Features fuer den Tenant
- `moduleSettings` ermoeglicht granulare Steuerung pro Modul
- `provider`: Bestimmt welcher AI-Provider genutzt wird
  - `openai_eu`: Azure OpenAI in EU-Region
  - `anthropic_eu`: Anthropic EU-Endpoint
  - `self_hosted`: Eigenes LLM (z.B. Llama, Mistral) auf eigenem Server
- `modelPreference`: Steuert Modellwahl (fast = guenstiger/schneller, quality = besser/teurer)
- `monthlyBudgetEur`: Limit fuer AI-Kosten pro Monat (0 = unlimited)
- `currentMonthUsageEur`: Berechnet aus Token-Usage (automatisch)
- `dataProcessingConsent`: Muss true sein bevor AI genutzt werden kann (DSGVO Art. 6)
- Nur Admins koennen Settings aendern

---

### 13.13 AI Activity Log

```
GET    /api/v1/ai/activity-log                            -> KI-Aktivitaetsprotokoll
```

Query params: `?userId=&module=&type=generate|summarize|suggest|search|classify&dateFrom=&dateTo=&page=1&pageSize=50`

Response:
```json
{
  "activities": [
    {
      "id": "uuid",
      "userId": "uuid",
      "userName": "Lisa Mueller",
      "type": "generate|summarize|suggest|search|classify",
      "module": "helpdesk",
      "inputPreview": "Ticket-Antwort generieren: Drucker-Problem...",
      "outputPreview": "Vielen Dank fuer Ihre Meldung. Der Fehlercode...",
      "tokensUsed": 245,
      "costEur": 0.003,
      "model": "gpt-4o",
      "createdAt": "2026-02-22T14:30:00Z"
    }
  ],
  "total": 234,
  "totalTokens": 45680,
  "totalCostEur": 12.45
}
```

**Backend-Logik:**
- Jeder AI-API-Call wird protokolliert
- `inputPreview` / `outputPreview`: Gekuerzt auf 200 Zeichen (Datenschutz)
- `costEur` berechnet aus Tokens + Provider-Preisliste
- Aggregation: `totalTokens` und `totalCostEur` fuer den gefilterten Zeitraum
- Retention: 90 Tage (danach nur aggregierte Statistiken)

---

### 13.14 AI Usage Statistics

```
GET    /api/v1/ai/usage-stats                             -> KI-Nutzungsstatistiken
```

Query params: `?period=current_month|last_month|quarter|year`

Response:
```json
{
  "period": "current_month",
  "totalRequests": 234,
  "totalTokens": 45680,
  "totalCostEur": 12.45,
  "budgetEur": 50.00,
  "budgetUsedPercent": 24.9,
  "byModule": {
    "helpdesk": { "requests": 89, "tokens": 18200, "costEur": 4.55 },
    "chat": { "requests": 67, "tokens": 12400, "costEur": 3.10 },
    "dokumente": { "requests": 45, "tokens": 8900, "costEur": 2.23 },
    "email": { "requests": 33, "tokens": 6180, "costEur": 2.57 }
  },
  "byType": {
    "generate": { "requests": 120, "tokens": 28000 },
    "search": { "requests": 65, "tokens": 5200 },
    "suggest": { "requests": 30, "tokens": 7500 },
    "classify": { "requests": 15, "tokens": 3000 },
    "summarize": { "requests": 4, "tokens": 1980 }
  },
  "dailyTrend": [
    { "date": "2026-02-01", "requests": 12, "costEur": 0.65 },
    { "date": "2026-02-02", "requests": 8, "costEur": 0.42 }
  ]
}
```

---

## D. DB Schema Suggestions

### DSGVO

```sql
CREATE TABLE dsar_requests (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id       UUID NOT NULL,
  person_id       UUID NOT NULL,
  person_name     VARCHAR(300),
  person_email    VARCHAR(300),
  requested_by    VARCHAR(300) NOT NULL,
  request_date    DATE NOT NULL,
  modules         TEXT[] NOT NULL DEFAULT '{}',
  format          VARCHAR(10) NOT NULL DEFAULT 'json',
  status          VARCHAR(15) DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'error')),
  file_path       VARCHAR(500),
  completed_at    TIMESTAMPTZ,
  expires_at      TIMESTAMPTZ,
  created_at      TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_dsar_tenant ON dsar_requests(tenant_id);
CREATE INDEX idx_dsar_person ON dsar_requests(person_id);
CREATE INDEX idx_dsar_expires ON dsar_requests(expires_at);

CREATE TABLE dsar_module_status (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  dsar_id         UUID NOT NULL REFERENCES dsar_requests(id) ON DELETE CASCADE,
  module_id       VARCHAR(50) NOT NULL,
  status          VARCHAR(15) DEFAULT 'pending',
  record_count    INTEGER DEFAULT 0,
  completed_at    TIMESTAMPTZ
);
CREATE INDEX idx_dsar_module_dsar ON dsar_module_status(dsar_id);

CREATE TABLE deletion_requests (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id       UUID NOT NULL,
  person_id       UUID NOT NULL,
  person_name     VARCHAR(300),
  reason          TEXT,
  modules         TEXT[] NOT NULL DEFAULT '{}',
  excluded_modules TEXT[] DEFAULT '{}',
  status          VARCHAR(20) DEFAULT 'pending_approval' CHECK (status IN ('pending_approval', 'approved', 'executing', 'completed', 'rejected')),
  requested_by    UUID NOT NULL,
  approved_by     UUID,
  executed_at     TIMESTAMPTZ,
  created_at      TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_deletion_tenant ON deletion_requests(tenant_id);

CREATE TABLE retention_policies (
  id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id           UUID NOT NULL,
  module_id           VARCHAR(50) NOT NULL,
  data_category       VARCHAR(200) NOT NULL,
  retention_days      INTEGER NOT NULL,
  legal_basis         TEXT,
  auto_delete_enabled BOOLEAN DEFAULT FALSE,
  description         TEXT,
  UNIQUE(tenant_id, module_id, data_category)
);
CREATE INDEX idx_retention_tenant ON retention_policies(tenant_id);

CREATE TABLE consents (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     UUID NOT NULL,
  person_id     UUID,
  person_name   VARCHAR(300),
  person_email  VARCHAR(300),
  purpose       VARCHAR(50) NOT NULL,
  legal_basis   TEXT,
  status        VARCHAR(10) DEFAULT 'active' CHECK (status IN ('active', 'revoked')),
  given_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  revoked_at    TIMESTAMPTZ,
  source        VARCHAR(30),
  ip_address    INET,
  notes         TEXT,
  created_at    TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_consents_tenant ON consents(tenant_id);
CREATE INDEX idx_consents_person ON consents(person_id);
CREATE INDEX idx_consents_purpose ON consents(purpose);

CREATE TABLE dsgvo_audit_log (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     UUID NOT NULL,
  action        VARCHAR(50) NOT NULL,
  actor_id      UUID NOT NULL,
  actor_name    VARCHAR(200),
  person_id     UUID,
  person_name   VARCHAR(200),
  details       TEXT,
  created_at    TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_dsgvo_audit_tenant ON dsgvo_audit_log(tenant_id);
CREATE INDEX idx_dsgvo_audit_action ON dsgvo_audit_log(action);
CREATE INDEX idx_dsgvo_audit_date ON dsgvo_audit_log(created_at);
```

### KI

```sql
CREATE TABLE ai_tenant_settings (
  tenant_id               UUID PRIMARY KEY,
  ai_enabled              BOOLEAN DEFAULT FALSE,
  provider                VARCHAR(30) DEFAULT 'openai_eu',
  model_preference        VARCHAR(20) DEFAULT 'balanced',
  monthly_budget_eur      DECIMAL(10,2) DEFAULT 50.00,
  module_settings         JSONB NOT NULL DEFAULT '{}',
  data_processing_consent BOOLEAN DEFAULT FALSE,
  consent_date            TIMESTAMPTZ,
  updated_at              TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE ai_activity_log (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id       UUID NOT NULL,
  user_id         UUID NOT NULL,
  request_type    VARCHAR(20) NOT NULL CHECK (request_type IN ('generate', 'summarize', 'suggest', 'search', 'classify')),
  module_id       VARCHAR(50),
  input_preview   VARCHAR(200),
  output_preview  VARCHAR(200),
  tokens_used     INTEGER NOT NULL DEFAULT 0,
  cost_eur        DECIMAL(10,6) DEFAULT 0,
  model           VARCHAR(50),
  created_at      TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_ai_log_tenant ON ai_activity_log(tenant_id);
CREATE INDEX idx_ai_log_user ON ai_activity_log(user_id);
CREATE INDEX idx_ai_log_date ON ai_activity_log(created_at);
CREATE INDEX idx_ai_log_type ON ai_activity_log(request_type);

-- Embedding-Tabelle (fuer Semantic Search mit pgvector)
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE ai_embeddings (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id       UUID NOT NULL,
  module_id       VARCHAR(50) NOT NULL,
  entity_type     VARCHAR(50) NOT NULL,
  entity_id       UUID NOT NULL,
  content_preview VARCHAR(500),
  embedding       vector(1536),
  author          VARCHAR(200),
  created_at      TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE(tenant_id, module_id, entity_id)
);
CREATE INDEX idx_embeddings_tenant ON ai_embeddings(tenant_id);
CREATE INDEX idx_embeddings_vector ON ai_embeddings USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- Document classification cache
ALTER TABLE document_files ADD COLUMN IF NOT EXISTS classification_level VARCHAR(20);
ALTER TABLE document_files ADD COLUMN IF NOT EXISTS classification_confidence DECIMAL(3,2);
ALTER TABLE document_files ADD COLUMN IF NOT EXISTS contains_pii BOOLEAN;
ALTER TABLE document_files ADD COLUMN IF NOT EXISTS pii_types TEXT[];
ALTER TABLE document_files ADD COLUMN IF NOT EXISTS classified_at TIMESTAMPTZ;
```

---

## E. Summary: Recommended Build Order

| Prio | Item | What | Effort |
|------|------|------|--------|
| 1 | AI Tenant Settings | Basis-Config CRUD mit Module-Toggles | ~60 LOC |
| 2 | AI Proxy (Generate) | Zentraler AI-Dispatcher mit Provider-Abstraction | ~200 LOC |
| 3 | AI Activity Log | Logging + Usage-Statistiken | ~100 LOC |
| 4 | DSGVO Person Search | Cross-Module Suche | ~150 LOC |
| 5 | DSAR Export | Asynchroner Cross-Module Daten-Export | ~200 LOC |
| 6 | Retention Policies | Seed-Daten + CRUD + Auto-Delete Cron | ~100 LOC |
| 7 | Consent Log | CRUD + Audit-Trail | ~80 LOC |
| 8 | DSGVO Audit Log | Automatisches Protokoll | ~60 LOC |
| 9 | Deletion Requests | Art. 17 Workflow mit Approval | ~150 LOC |
| 10 | Ticket Suggestion | AI-basierte Kategorisierung | ~80 LOC |
| 11 | Meeting Summarization | AI-Zusammenfassung aus Notes | ~80 LOC |
| 12 | Document Classification | AI-Klassifizierung + PII-Erkennung | ~100 LOC |
| 13 | Embedding Pipeline | Background-Worker fuer Embeddings | ~200 LOC |
| 14 | Semantic Search | Vektor-Suche mit pgvector | ~150 LOC |
| 15 | AI Budget Enforcement | Monatliches Limit pruefen + Warnung | ~40 LOC |

**Total: ~1,750 LOC Go** (ohne externe Provider-Connector, die kommen on-demand)

---

## F. Cross-Module Dependencies

- **13.1 Person Search -> Alle Module:** Cross-Service-Queries an CRM, Helpdesk, Chat, Finanzen, etc.
- **13.2 DSAR -> Alle Module:** Daten-Export aus jedem Modul fuer eine bestimmte Person
- **13.3 Deletion -> Alle Module:** Anonymisierung/Loeschung in jedem Modul
- **13.5 Consents -> Formulare (Wave 10):** Automatischer Consent bei oeffentlichen Formularen
- **13.7 AI Proxy -> External AI API:** OpenAI (Azure EU), Anthropic, oder Self-Hosted LLM
- **13.8 Meeting Summary -> Meetings (Wave 10):** Notes-Feld aus Meeting lesen
- **13.9 Ticket Suggestion -> Helpdesk:** Ticket-Daten lesen, Vorschlaege zurueckgeben
- **13.10 Semantic Search -> pgvector:** PostgreSQL-Extension erforderlich
- **13.10 Embedding Pipeline -> Alle Content-Module:** Chat, E-Mail, Dokumente, CRM-Notizen
- **13.11 Document Classification -> Dokumente:** Classification-Felder auf document_files

---

## G. Notes for Luke

- **DSGVO ist KEIN Nice-to-Have:** Fuer DACH-KMUs ist DSGVO-Compliance ein Muss-Feature. Kunden werden danach fragen. Die DSAR-Funktion (Art. 15) muss funktionieren bevor wir live gehen.
- **Person Search ist die Grundlage:** Fast alle DSGVO-Features bauen auf der Cross-Module-Person-Search auf. Implementiere das zuerst. Idealerweise gibt es eine zentrale `persons`-Tabelle die alle externen Kontakte referenziert (CRM-Kontakt, Ticket-Requester, etc. zeigen auf dieselbe Person).
- **AI Provider Abstraction:** Baue eine Provider-Schnittstelle (`AIProvider` Interface mit `Generate`, `Embed`, `Classify` Methoden). Dann Implementierungen fuer OpenAI und Anthropic. So koennen wir spaeter leicht wechseln.
- **EU-Only fuer AI:** Azure OpenAI bietet EU-Regionen (West Europe, France Central). Die API-Calls muessen IMMER an EU-Endpoints gehen. Das ist ein Selling-Point gegenueber amerikanischen CRM-Anbietern.
- **pgvector fuer Semantic Search:** `CREATE EXTENSION vector;` in PostgreSQL. 1536 Dimensionen fuer OpenAI `text-embedding-3-small`. IVFFlat-Index fuer schnelle Nearest-Neighbor-Suche. Alternatives: Qdrant als separater Service (besser fuer grosse Datasets).
- **Embedding Pipeline — Async:** Embeddings bei jedem INSERT/UPDATE von Text-Content erstellen. Als Background-Worker (Message Queue), NICHT synchron im Request-Path. Batch-Processing fuer initiale Indexierung.
- **AI Budget — Kosten:** OpenAI Pricing (Stand 2025): GPT-4o ~$2.50/1M input tokens, $10/1M output tokens. Embeddings: ~$0.02/1M tokens. Bei 50 EUR/Monat Budget kann ein KMU ~5M Tokens verbrauchen. Das reicht fuer ~500-1000 AI-Anfragen.
- **Consent Log — Beweislast:** Die DSGVO legt dem Verantwortlichen die Beweislast fuer Einwilligungen auf. Deshalb: Consent-Eintraege NIEMALS loeschen, nur widerrufen. IP-Adresse + Timestamp + Source dokumentieren.
- **Deletion Request — Soft Delete + Anonymisierung:** Nie "hard delete" bei DSGVO-Loeschungen. Stattdessen: Personenbezogene Felder anonymisieren (Name -> "Geloeschte Person", E-Mail -> SHA256-Hash). So bleiben Statistiken und Referenzen intakt.
- **Rollen:** Neuen Rollen-Typ `privacy_officer` einfuehren. Nur Admins + Privacy Officers haben Zugriff auf DSGVO-Endpoints. AI-Settings nur fuer Admins.

# Backend-Plan Wave 6: Work/Projekte + Zeiterfassung

> **Von:** Darien (Frontend) | **Fuer:** Luke (Backend)
> **Datum:** 2026-02-21 | **Status:** Frontend IN PROGRESS
> **Geschaetzter Backend-Aufwand:** ~1.000 LOC Go

---

## Uebersicht

Wave 6 hat 11 Items (5 Work, 6 Zeiterfassung), davon sind 4 Backend-abhaengig.
Der Rest ist FE-ONLY (Gantt-Drag, Budget-Tracking, Ueberstunden, Abwesenheits-Integration, GPS-Mock).

**Was das Frontend aktuell macht:**
- Work: Gantt-Balken sind jetzt draggable (Move/Resize), Budget-Section und Auslastungs-Report mit Mock-Daten
- Zeiterfassung: Projekt/Task-Dropdown im Timer, Export-Dialog, Ueberstunden-Berechnung, Genehmigungs-Banner — alles Zustand Store

---

## A. Work Backend-Anforderungen

### A1. Stunden-zu-Rechnung Cross-Modul (Item 6.2) — PRIO MITTEL

Gleiche Anforderung wie Wave 3.19, aber aus einer anderen Richtung:
Wave 3.19 = Zeiterfassungs-Seite → Rechnung. Wave 6.2 = Projekt-Detail → Rechnung.

**Der Backend-Endpoint ist identisch:**
```
POST /api/v1/invoices/from-timeentries
```

(Siehe `wave-03-crm-finanzen.md`, Abschnitt B7 fuer Details)

**Zusaetzlich benoetigt fuer Projekt-Kontext:**
```
GET /api/v1/projects/:id/time-entries?invoiced=false
```

**Response:**
```json
{
  "time_entries": [
    {
      "id": "uuid",
      "task_id": "uuid",
      "task_title": "Homepage Design",
      "user_id": "uuid",
      "user_name": "Max Mustermann",
      "date": "2026-02-20",
      "duration_minutes": 180,
      "description": "Wireframes erstellt",
      "hourly_rate": 120.00,
      "is_invoiced": false
    }
  ],
  "total_uninvoiced_hours": 42.5,
  "total_uninvoiced_amount": 5100.00
}
```

**Frontend-Flow:**
1. User klickt "Rechnung erstellen" im Projekt-Detail
2. Dialog zeigt alle nicht-abgerechneten Zeiteintraege des Projekts
3. User waehlt aus, setzt Stundensatz, waehlt Kunde
4. `POST /api/v1/invoices/from-timeentries` erstellt Rechnung
5. Zeiteintraege werden als `invoiced` markiert

---

### A2. Gaeste-Zugang (Item 6.5) — PRIO NIEDRIG

Read-only Projektansicht fuer externe Personen (Kunden, Partner) ohne KMU Hub Account.

**Auth-Erweiterung:**

Neuer Token-Typ: `guest_access_token`

```sql
CREATE TABLE guest_access_tokens (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id    UUID NOT NULL,
  project_id   UUID NOT NULL REFERENCES projects(id),
  token        VARCHAR(100) UNIQUE NOT NULL,
  label        VARCHAR(200),           -- "Fuer Kunde Firma GmbH"
  created_by   UUID NOT NULL,          -- Wer hat den Link erstellt
  permissions  JSONB DEFAULT '{}',     -- {"view_tasks": true, "view_milestones": true, "view_files": false}
  expires_at   TIMESTAMPTZ,            -- NULL = kein Ablauf
  last_used_at TIMESTAMPTZ,
  is_active    BOOLEAN DEFAULT TRUE,
  created_at   TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_guest_token ON guest_access_tokens(token);
```

**Endpoints:**

Intern (Auth):
```
POST   /api/v1/projects/:id/guest-links          — Gaeste-Link erstellen
GET    /api/v1/projects/:id/guest-links          — Alle Links auflisten
DELETE /api/v1/projects/:id/guest-links/:linkId  — Link deaktivieren
```

**Create Request:**
```json
{
  "label": "Fuer Firma GmbH",
  "permissions": {
    "view_tasks": true,
    "view_milestones": true,
    "view_files": false,
    "view_budget": false
  },
  "expires_at": "2026-06-01T00:00:00Z"
}
```

**Create Response:**
```json
{
  "id": "uuid",
  "token": "abc123def456...",
  "url": "https://app.kmuhub.de/guest/abc123def456",
  "label": "Fuer Firma GmbH",
  "expires_at": "2026-06-01T00:00:00Z"
}
```

Oeffentlich (kein Auth, Token im URL):
```
GET /api/v1/public/projects/:token              — Projekt-Ueberblick (nur freigegebene Felder)
GET /api/v1/public/projects/:token/tasks        — Aufgaben-Liste (wenn erlaubt)
GET /api/v1/public/projects/:token/milestones   — Meilensteine (wenn erlaubt)
```

**Public Project Response:**
```json
{
  "project_name": "Website Redesign",
  "progress_percent": 65,
  "status": "In Arbeit",
  "milestones": [
    {
      "title": "Design Phase",
      "due_date": "2026-03-15",
      "completed": true
    },
    {
      "title": "Development",
      "due_date": "2026-04-30",
      "completed": false
    }
  ],
  "task_summary": {
    "total": 42,
    "completed": 27,
    "in_progress": 8,
    "open": 7
  }
}
```

**Sicherheit:**
- Token mindestens 32 Bytes random
- Rate Limiting auf Public Endpoints (10 req/min pro IP)
- Keine sensiblen Daten (Kommentare, interne Notizen, Kosten) in der Public API

---

## B. Zeiterfassung Backend-Anforderungen

### B1. DATEV Zeitexport (Item 6.7) — PRIO MITTEL

Export von Zeiteintraegen im DATEV-Lohn-Format fuer den Steuerberater.

**Endpoint:**
```
GET /api/v1/timeentries/export/datev?from=2026-01-01&to=2026-01-31&format=datev
GET /api/v1/timeentries/export/csv?from=2026-01-01&to=2026-01-31
GET /api/v1/timeentries/export/excel?from=2026-01-01&to=2026-01-31
```

**DATEV-Lohn-Format:**
- CSV mit Windows-1252 Encoding, Semikolon-Separator
- Spalten: Personal-Nr, Lohnart, Datum, Stunden, Zuschlag (%), Projekt-Nr, Kostentraeger
- Header-Zeile mit DATEV-Metadaten

**CSV-Format:**
```csv
Mitarbeiter;Datum;Projekt;Aufgabe;Start;Ende;Dauer (h);Kategorie;Beschreibung
Max Mustermann;20.02.2026;Website Redesign;Homepage Design;09:00;12:30;3.50;Entwicklung;Wireframes erstellt
```

**Excel-Format:** Gleiche Daten als `.xlsx` mit formatierter Tabelle.

**Response Header:**
```
Content-Type: text/csv; charset=windows-1252     (DATEV)
Content-Type: text/csv; charset=utf-8            (CSV)
Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet  (Excel)
Content-Disposition: attachment; filename="zeiterfassung_2026-01.csv"
```

---

### B2. Genehmigungs-Workflow (Item 6.9) — PRIO NIEDRIG

Wochenrapport-Genehmigung durch Vorgesetzten.

**DB-Schema:**
```sql
CREATE TABLE timesheet_approvals (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     UUID NOT NULL,
  user_id       UUID NOT NULL,           -- Mitarbeiter
  week_start    DATE NOT NULL,           -- Montag der Woche
  status        VARCHAR(20) DEFAULT 'pending',  -- 'pending', 'submitted', 'approved', 'rejected'
  submitted_at  TIMESTAMPTZ,
  reviewed_by   UUID,                    -- Vorgesetzter
  reviewed_at   TIMESTAMPTZ,
  rejection_reason TEXT,
  total_hours   DECIMAL(5,2),
  created_at    TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE(tenant_id, user_id, week_start)
);
CREATE INDEX idx_timesheet_approvals_user ON timesheet_approvals(user_id, week_start);
```

**Endpoints:**
```
-- Mitarbeiter:
POST /api/v1/timesheets/submit              — Wochenrapport einreichen
GET  /api/v1/timesheets/my?week=2026-W08    — Eigenen Status abrufen

-- Vorgesetzter:
GET  /api/v1/timesheets/pending              — Alle offenen Genehmigungen
POST /api/v1/timesheets/:id/approve          — Genehmigen
POST /api/v1/timesheets/:id/reject           — Ablehnen
```

**Submit Request:**
```json
{
  "week_start": "2026-02-17",
  "total_hours": 42.5,
  "comment": "Ueberstunden wegen Projektdeadline"
}
```

**Reject Request:**
```json
{
  "reason": "Bitte Projektzeiten fuer Montag nachtragen"
}
```

**Pending Response (fuer Vorgesetzten):**
```json
{
  "pending_approvals": [
    {
      "id": "uuid",
      "user_id": "uuid",
      "user_name": "Max Mustermann",
      "week_start": "2026-02-17",
      "total_hours": 42.5,
      "submitted_at": "2026-02-21T17:00:00Z",
      "entries_count": 12,
      "comment": "Ueberstunden wegen Projektdeadline"
    }
  ]
}
```

**Workflow:**
1. Mitarbeiter traegt Zeiten ein (normal, existiert schon)
2. Am Ende der Woche: "Woche einreichen" Button → `POST /api/v1/timesheets/submit`
3. Vorgesetzter sieht Banner "3 offene Genehmigungen" → `GET /api/v1/timesheets/pending`
4. Vorgesetzter klickt "Genehmigen" oder "Ablehnen"
5. Bei Ablehnung: Mitarbeiter sieht Banner "Woche abgelehnt: Bitte nachbessern"

---

## C. Zusammenfassung: Empfohlene Reihenfolge

| Prio | Item | Was | Aufwand |
|------|------|-----|---------|
| 1 | A1 | Stunden-zu-Rechnung (Cross-Modul, gleich wie Wave 3.19) | ~200 LOC |
| 2 | B1 | DATEV/CSV/Excel Zeitexport | ~300 LOC |
| 3 | A2 | Gaeste-Zugang (Public Token Auth) | ~350 LOC |
| 4 | B2 | Genehmigungs-Workflow (Timesheet Approval) | ~200 LOC |

**Total: ~1.050 LOC Go**

---

## D. Abhaengigkeiten zu anderen Waves

- **A1 (Stunden-zu-Rechnung)** haengt von Wave 3 B7 ab (gleicher Endpoint)
- **B1 (DATEV Export)** kann unabhaengig gebaut werden
- **A2 (Gaeste-Zugang)** kann unabhaengig gebaut werden (eigenes Auth-System)
- **B2 (Genehmigung)** kann unabhaengig gebaut werden

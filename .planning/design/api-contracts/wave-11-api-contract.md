# Wave 11 API Contract: Video + Notifications + Dashboard + Berichte + Dokumente

> **From:** Darien (Frontend) | **For:** Luke (Backend)
> **Date:** 2026-02-22 | **Status:** Entwurf
> **Frontend branch:** `design/brainstorm`
> **Estimated backend effort:** ~1,800 LOC Go

---

## Overview

Wave 11 extends five existing modules: **Video** (calls page), **Notifications** (full system), **Dashboard** (widgets), **Berichte** (reports), and **Dokumente** (documents). The frontend is fully built with a mix of React Query hooks (notifications, documents) and Zustand mock stores (video, berichte, dashboard).

**What the frontend currently does:**
- Video: Full page with active calls list (from meetings store), call history with search/grouping, device settings (audio/video/background)
- Notifications: Paginated list with all/unread filter, mark-all-as-read, per-event-type preferences (in_app/desktop_push toggles), snooze with timestamp, action buttons on toasts, WebSocket real-time push, quiet hours, DND mode, resource muting
- Dashboard: Quick actions bar, profile widget suggestions, customizable widget grid, alerts section
- Berichte: KPI cards with drill-down detail table, period comparison charts, DATEV tab (BWA + SuSa tables), report generation, scheduled reports with email recipients
- Dokumente: Template gallery (12 templates across 5 categories), share links (with expiry, password, permission), OnlyOffice editor integration, context menu with "open in Office"

---

## A. Video & Calls Endpoints

### Aktive Anrufe (Active Calls / Live Meetings)

```
GET    /api/v1/meetings/active                            -> Alle laufenden Meetings/Calls
```

Response:
```json
{
  "activeCalls": [
    {
      "id": "uuid",
      "title": "Team-Standup",
      "room": "meeting-room-slug",
      "startTime": "2026-02-22T09:00:00Z",
      "duration": 30,
      "color": "#6366f1",
      "isVideoCall": true,
      "participants": [
        {
          "id": "user-uuid",
          "name": "Anna Mueller",
          "initials": "AM"
        }
      ]
    }
  ]
}
```

**Backend-Logik:**
- Filtert alle Meetings mit `status: 'live'` (oder aktive LiveKit-Rooms)
- `duration` ist die aktuelle Laufzeit in Minuten (berechnet aus `startTime`)
- `room` ist der LiveKit-Room-Slug
- `participants` aus aktueller Room-Teilnehmerliste (LiveKit API) oder aus Meeting-Teilnehmerliste

---

### 11.1 Anrufverlauf (Call History)

```
GET    /api/v1/calls/history                              -> Anrufverlauf (paginiert)
```

Query params: `?search=&direction=incoming|outgoing|missed&type=audio|video&page=1&pageSize=50`

Response:
```json
{
  "entries": [
    {
      "id": "uuid",
      "contactId": "user-uuid|null",
      "contactName": "Thomas Keller",
      "contactInitials": "TK",
      "direction": "incoming|outgoing|missed",
      "type": "audio|video",
      "date": "2026-02-22",
      "startTime": "14:30",
      "duration": 12,
      "meetingId": "uuid|null"
    }
  ],
  "total": 142,
  "page": 1,
  "pageSize": 50
}
```

**Backend-Logik:**
- `duration` in Minuten (0 fuer verpasste Anrufe)
- `contactId` kann null sein bei externen Anrufern
- `meetingId` verknuepft mit einem Meeting falls der Anruf aus einem geplanten Meeting kam
- `search` durchsucht `contactName`
- Sortierung: neueste zuerst (absteigend nach `date` + `startTime`)

---

### 11.2 Video-Einstellungen (Device Preferences)

```
GET    /api/v1/users/me/video-settings                    -> Aktuelle Video-Einstellungen
PUT    /api/v1/users/me/video-settings                    -> Einstellungen aktualisieren
```

VideoSettings model:
```json
{
  "audioInputDeviceId": "default",
  "audioOutputDeviceId": "default",
  "videoInputDeviceId": "default",
  "virtualBackground": "none|blur|office|nature",
  "noiseSuppression": true,
  "echoCancellation": true
}
```

**Backend-Logik:**
- Pro-User-Einstellung (JSONB auf User-Tabelle oder eigene Tabelle)
- Device-IDs sind Browser/Electron MediaDeviceInfo.deviceId — opak fuer das Backend
- `virtualBackground` wird clientseitig angewendet, Backend speichert nur die Praeferenz
- Default-Werte wenn noch keine Einstellungen gespeichert

---

## B. Notifications Endpoints

### Notifications CRUD (Basis)

```
GET    /api/v1/notifications                              -> Paginierte Benachrichtigungen
GET    /api/v1/notifications/unread-count                  -> Ungelesene Anzahl
POST   /api/v1/notifications/{id}/read                     -> Als gelesen markieren
POST   /api/v1/notifications/read-all                      -> Alle als gelesen markieren
```

Query params (GET list): `?page=1&page_size=20&is_read=true|false&module_id=crm|chat|hr|...`

Notification model:
```json
{
  "id": "uuid",
  "tenant_id": "uuid",
  "user_id": "uuid",
  "title": "Neue Nachricht von Anna Mueller",
  "body": "Hey, hast du die Entwuerfe gesehen?",
  "module_id": "chat|crm|hr|finanzen|kalender|projekte|helpdesk|system",
  "priority": "low|normal|high|urgent",
  "is_read": false,
  "deep_link": "/chat/channel-uuid",
  "group_key": "chat-channel-uuid|null",
  "group_count": 3,
  "created_at": "ISO-8601",
  "snoozed_until": "ISO-8601|null"
}
```

List Response:
```json
{
  "notifications": [...],
  "total": 47,
  "page": 1,
  "page_size": 20
}
```

Unread Count Response:
```json
{
  "count": 12
}
```

Mark All Read Request:
```json
{
  "module_id": "chat|null"
}
```

**Backend-Logik:**
- `module_id` optional im read-all Request (nur Notifications dieses Moduls als gelesen markieren)
- `group_key` fasst aehnliche Notifications zusammen (z.B. mehrere Nachrichten im selben Channel)
- `group_count` zeigt die Anzahl der gruppierten Notifications an
- `deep_link` ist ein relativer Pfad in der App (z.B. `/chat/channel-uuid`, `/crm/contacts/uuid`)
- Notification-Erstellung erfolgt serverseitig wenn Events eintreten (neue Nachricht, Task zugewiesen, etc.)
- `snoozed_until` wenn gesetzt: Notification wird bis zu diesem Zeitpunkt nicht in der Liste angezeigt (Frontend filtert, aber Backend speichert)

---

### 11.3 Notification Snooze

```
POST   /api/v1/notifications/{id}/snooze                  -> Notification zurueckstellen
```

Request:
```json
{
  "snoozed_until": "2026-02-22T16:00:00Z"
}
```

Response:
```json
{
  "id": "uuid",
  "snoozed_until": "2026-02-22T16:00:00Z"
}
```

**Backend-Logik:**
- `snoozed_until` wird auf der Notification gespeichert
- GET-Liste filtert snoozed Notifications aus (optional: Query-Param `include_snoozed=true`)
- Scheduler/Cron: Wenn `snoozed_until <= now()`, Snooze-Feld auf null setzen und WebSocket-Event `notification.unsnooze` senden
- Frontend-Snooze-Optionen: 30 Min, 1 Std, Morgen (24h) — berechneter Timestamp wird im Request gesendet

---

### 11.4 Notification Preferences

```
GET    /api/v1/notifications/preferences                  -> Alle Praeferenzen
PUT    /api/v1/notifications/preferences                  -> Praeferenz aktualisieren
```

Query params (GET): `?module_id=chat|crm|...` (optional, filtert nach Modul)

GET Response:
```json
{
  "preferences": [
    {
      "event_type_key": "chat.new_message",
      "in_app": true,
      "desktop_push": true,
      "email": false
    }
  ]
}
```

PUT Request (einzelne Praeferenz):
```json
{
  "event_type_key": "chat.new_message",
  "in_app": true,
  "desktop_push": false
}
```

**Backend-Logik:**
- Pro User + pro Event-Typ gespeichert
- Defaults kommen aus der Event-Type-Definition (siehe 11.5)
- Wenn fuer einen Event-Typ keine Praeferenz existiert, gelten die Defaults
- `email` Kanal optional (spaetere Erweiterung), fuer jetzt `in_app` + `desktop_push`

---

### 11.5 Notification Event Types

```
GET    /api/v1/notifications/event-types                  -> Verfuegbare Event-Typen
```

Query params: `?module_id=chat|crm|...` (optional)

Response:
```json
{
  "event_types": [
    {
      "id": "uuid",
      "event_key": "chat.new_message",
      "display_name": "Neue Nachricht",
      "description": "Benachrichtigung bei neuen Chat-Nachrichten",
      "module_id": "chat",
      "default_in_app": true,
      "default_desktop_push": true
    },
    {
      "id": "uuid",
      "event_key": "crm.deal_stage_changed",
      "display_name": "Deal-Status geaendert",
      "description": "Ein Deal wurde in eine andere Pipeline-Phase verschoben",
      "module_id": "crm",
      "default_in_app": true,
      "default_desktop_push": false
    }
  ]
}
```

**Backend-Logik:**
- Event-Typen sind vom System definiert (Seed-Daten, nicht user-editierbar)
- Gruppierung nach `module_id` im Frontend fuer die Einstellungs-UI
- Empfohlene Event-Typen (Minimum):
  - `chat.new_message`, `chat.mention`
  - `crm.deal_stage_changed`, `crm.contact_assigned`
  - `hr.leave_approved`, `hr.leave_rejected`
  - `calendar.event_reminder`, `calendar.event_cancelled`
  - `helpdesk.ticket_new`, `helpdesk.ticket_reply`
  - `finanzen.invoice_overdue`, `finanzen.payment_received`
  - `system.update_available`, `system.maintenance`
  - `contract.expiry_reminder`

---

### 11.6 Quiet Hours + DND

```
GET    /api/v1/notifications/quiet-hours                  -> Ruhezeiten lesen
PUT    /api/v1/notifications/quiet-hours                  -> Ruhezeiten aktualisieren
GET    /api/v1/notifications/dnd                          -> DND-Status lesen
POST   /api/v1/notifications/dnd/enable                   -> DND aktivieren
POST   /api/v1/notifications/dnd/disable                  -> DND deaktivieren
```

QuietHours model:
```json
{
  "quiet_hours": {
    "enabled": true,
    "start_time": "22:00",
    "end_time": "07:00",
    "timezone": "Europe/Berlin"
  }
}
```

DND Status:
```json
{
  "active": true,
  "expires_at": "2026-02-22T17:00:00Z|null"
}
```

DND Enable Request:
```json
{
  "expires_at": "2026-02-22T17:00:00Z"
}
```

**Backend-Logik:**
- Quiet Hours: Waehrend der Ruhezeit werden keine Push-Notifications gesendet (In-App werden trotzdem gespeichert)
- DND: Keine Notifications bis `expires_at` (null = manuell deaktivieren)
- Beide sind pro-User-Einstellungen
- Scheduler: DND automatisch deaktivieren wenn `expires_at <= now()`

---

### 11.7 Resource Muting

```
GET    /api/v1/notifications/mutes                        -> Stummschaltungen lesen
POST   /api/v1/notifications/mutes                        -> Ressource stummschalten
DELETE /api/v1/notifications/mutes/{muteId}               -> Stummschaltung aufheben
```

Mute model:
```json
{
  "id": "uuid",
  "resource_type": "channel|thread|ticket|project",
  "resource_id": "uuid",
  "created_at": "ISO-8601"
}
```

**Backend-Logik:**
- Stummgeschaltete Ressourcen generieren keine Notifications fuer diesen User
- Bei Notification-Erstellung pruefen: Ist der User fuer diese Ressource stummgeschaltet?
- Typische Anwendung: Chat-Channel stummschalten, einzelnes Ticket stummschalten

---

### 11.8 WebSocket Events (Notifications)

**BACKEND-DEP: WebSocket-Events fuer Echtzeit-Benachrichtigungen.**

Events:
```
notification.new          -> Neue Notification erstellt (Payload = Notification-Objekt)
notification.unread_count -> Unread Count geaendert (Payload: { count: number })
notification.unsnooze     -> Snooze abgelaufen (Payload: { id: "uuid" })
```

**Backend-Logik:**
- Bei jeder Notification-Erstellung: WebSocket-Event an den Ziel-User senden
- Pruefen: Quiet Hours aktiv? DND aktiv? Ressource stummgeschaltet? Praeferenz `in_app` deaktiviert? -> Dann NICHT senden
- `notification.unread_count` kann per Polling (30s Fallback) oder bei jeder Aenderung gesendet werden

---

## C. Dashboard Endpoints

### 11.9 Quick Stats (Dashboard-Widgets)

```
GET    /api/v1/dashboard/stats                            -> Schnell-Statistiken
```

Response:
```json
{
  "stats": {
    "unread_messages": 5,
    "open_tasks": 12,
    "upcoming_events": 3,
    "open_tickets": 8,
    "overdue_invoices": 2,
    "active_deals": 7,
    "pending_approvals": 1
  }
}
```

**Backend-Logik:**
- Aggregation ueber mehrere Module (Cross-Service-Queries oder Redis-Cache)
- Caching empfohlen: 60s TTL, da die Werte sich nicht staendig aendern
- Nur Daten fuer Module zaehlen, die der Tenant aktiviert hat

---

### 11.10 Activity Feed

```
GET    /api/v1/dashboard/activity-feed                    -> Letzte Aktivitaeten
```

Query params: `?limit=20`

Response:
```json
{
  "activities": [
    {
      "id": "uuid",
      "type": "deal_created|task_completed|message_sent|invoice_paid|contact_added|meeting_scheduled",
      "title": "Deal 'Projekt Alpha' erstellt",
      "actor_name": "Lisa Mueller",
      "actor_initials": "LM",
      "module_id": "crm",
      "timestamp": "ISO-8601",
      "deep_link": "/crm/deals/uuid"
    }
  ]
}
```

**Backend-Logik:**
- Letzte N Aktivitaeten ueber alle Module aggregiert
- Sortierung: neueste zuerst
- Nicht dasselbe wie Notifications: Activity Feed zeigt alle Aktivitaeten, Notifications nur relevante Events fuer den User
- Optional: Activity-Log-Tabelle die bei jeder Aktion gefuellt wird

---

### 11.11 Upcoming Events

```
GET    /api/v1/dashboard/upcoming-events                  -> Naechste Termine
```

Query params: `?limit=5`

Response:
```json
{
  "events": [
    {
      "id": "uuid",
      "title": "Team-Standup",
      "start_time": "2026-02-22T09:00:00Z",
      "duration": 30,
      "type": "meeting|task|reminder|blocker",
      "is_video_call": true,
      "attendee_count": 4
    }
  ]
}
```

**Backend-Logik:**
- Naechste N Events ab jetzt aus dem Kalender-Modul
- Nur Events an denen der User teilnimmt oder die er erstellt hat
- `attendee_count` statt voller Teilnehmerliste (Performance)

---

## D. Berichte (Reports) Endpoints

### 11.12 KPI-Daten

```
GET    /api/v1/reports/kpis                               -> Alle KPI-Werte
```

Query params: `?module_id=crm|finanzen|helpdesk|...&period=current_month|last_month|quarter|year`

Response:
```json
{
  "kpis": [
    {
      "id": "kpi-1",
      "label": "Umsatz",
      "value": "48.500",
      "unit": "EUR",
      "changePercent": 12.3,
      "moduleId": "finanzen",
      "period": "current_month"
    },
    {
      "id": "kpi-2",
      "label": "Offene Deals",
      "value": "23",
      "unit": null,
      "changePercent": -5.2,
      "moduleId": "crm",
      "period": "current_month"
    }
  ]
}
```

**Backend-Logik:**
- `changePercent` wird serverseitig berechnet (aktueller Zeitraum vs. Vorperiode)
- `value` ist ein formatierter String (fuer flexible Darstellung: EUR, Prozent, Stueck, Zeit)
- `unit` optional (null wenn `value` bereits formatiert)
- KPIs pro Modul:
  - Finanzen: Umsatz, Offene Rechnungen, Zahlungseingaenge
  - CRM: Offene Deals, Conversion Rate, Neukontakte
  - Helpdesk: Offene Tickets, Reaktionszeit, Kundenzufriedenheit
  - HR: Mitarbeiteranzahl, Krankentage, Ueberstunden
  - Produktion: Ausschussquote, Auslastung

---

### 11.13 KPI Drill-Down

```
GET    /api/v1/reports/kpis/{kpiId}/details               -> Detail-Daten zu einem KPI
```

Query params: `?period=current_month&limit=20`

Response:
```json
{
  "kpiId": "kpi-1",
  "details": [
    {
      "date": "14.02.2026",
      "label": "Rechnung #2026-048 an Meier AG",
      "amount": "12.500 EUR",
      "status": "Bezahlt"
    },
    {
      "date": "12.02.2026",
      "label": "Rechnung #2026-045 an Schmidt GmbH",
      "amount": "8.200 EUR",
      "status": "Offen"
    }
  ]
}
```

**Backend-Logik:**
- Detail-Tabelle zu einem bestimmten KPI
- Die Felder haengen vom KPI-Typ ab (Finanzen: Rechnungen, CRM: Deals, Helpdesk: Tickets)
- `amount` ist ein formatierter String
- `status` ist ein beschreibender Text

---

### 11.14 Periodenvergleich (Period Comparison)

```
GET    /api/v1/reports/comparison                         -> Vergleichsdaten fuer Charts
```

Query params: `?metric=revenue|deals|tickets&current_period=2026-01|2026-06&compare_period=2025-01|2025-06`

Response:
```json
{
  "metric": "revenue",
  "data": [
    { "label": "Jan", "current": 42, "previous": 38 },
    { "label": "Feb", "current": 48, "previous": 41 },
    { "label": "Mrz", "current": 45, "previous": 39 },
    { "label": "Apr", "current": 51, "previous": 44 },
    { "label": "Mai", "current": 47, "previous": 42 },
    { "label": "Jun", "current": 53, "previous": 46 }
  ],
  "summary": {
    "current_total": 286,
    "previous_total": 250,
    "difference": 36,
    "difference_percent": 14.4
  }
}
```

**Backend-Logik:**
- `current` und `previous` sind Werte in Tsd. EUR (oder andere Einheit je nach Metrik)
- Perioden als Monatsbereich: z.B. `2026-01|2026-06` (Januar bis Juni 2026)
- `summary` wird serverseitig berechnet

---

### 11.15 DATEV BWA + SuSa Export-Daten

```
GET    /api/v1/reports/datev/bwa                          -> BWA-Daten
GET    /api/v1/reports/datev/susa                         -> Summen & Salden
GET    /api/v1/reports/datev/export                       -> DATEV-Export als Datei
```

Query params (BWA/SuSa): `?period=2026-02`

BWA Response:
```json
{
  "period": "2026-02",
  "rows": [
    {
      "position": "1",
      "label": "Umsatzerloese",
      "currentMonth": 85400,
      "previousMonth": 78200,
      "yearToDate": 163600
    },
    {
      "position": "2",
      "label": "Materialaufwand",
      "currentMonth": -24500,
      "previousMonth": -22100,
      "yearToDate": -46600
    },
    {
      "position": "3",
      "label": "Rohertrag",
      "currentMonth": 60900,
      "previousMonth": 56100,
      "yearToDate": 117000
    }
  ]
}
```

SuSa Response:
```json
{
  "period": "2026-02",
  "rows": [
    {
      "position": "1400",
      "label": "Forderungen aus Lieferungen",
      "currentMonth": 34200,
      "previousMonth": 31800,
      "yearToDate": 34200
    }
  ]
}
```

Export Query params: `?type=bwa|susa&period=2026-02&format=csv|datev`

Export Response:
- `format=csv`: `Content-Type: text/csv; charset=utf-8` mit BOM
- `format=datev`: DATEV-kompatibles Exportformat (ASCII mit Header)
- `Content-Disposition: attachment; filename="BWA_2026-02.csv"`

**Backend-Logik:**
- BWA-Positionen nach Standard-SKR03/SKR04 Kontenrahmen
- `position` ist die BWA-Zeile (1-11 Standard-Positionen)
- SuSa: Position ist die Kontonummer (SKR03/04)
- Negative Werte = Aufwand/Kosten
- `yearToDate` ist der kumulierte Wert seit Jahresbeginn
- DATEV-Export: Beraternummer + Mandantennummer aus Integration-Settings (Wave 12)

---

### 11.16 Geplante Berichte (Scheduled Reports)

```
GET    /api/v1/reports/scheduled                          -> Alle geplanten Berichte
POST   /api/v1/reports/scheduled                          -> Neuen geplanten Bericht erstellen
PUT    /api/v1/reports/scheduled/{id}                     -> Geplanten Bericht bearbeiten
DELETE /api/v1/reports/scheduled/{id}                     -> Geplanten Bericht loeschen
PATCH  /api/v1/reports/scheduled/{id}/toggle              -> Aktivieren/Deaktivieren
```

ScheduledReport model:
```json
{
  "id": "uuid",
  "tenantId": "uuid",
  "name": "Monatsbericht CRM",
  "reportId": "uuid",
  "schedule": "daily|weekly|monthly|quarterly",
  "active": true,
  "recipients": ["chef@firma.de", "controlling@firma.de"],
  "lastRun": "2026-02-15T08:00:00Z|null",
  "nextRun": "2026-02-22T08:00:00Z",
  "createdAt": "ISO-8601"
}
```

Create Request:
```json
{
  "reportId": "uuid",
  "schedule": "weekly",
  "recipients": ["chef@firma.de"]
}
```

Toggle Request:
```json
{
  "active": true
}
```

**BACKEND-DEP: Cron-Job fuer automatischen Berichtsversand.**

**Backend-Logik:**
- Cron-Job prueft taeglich (z.B. 08:00 UTC): Welche Berichte muessen laufen?
- Bei Faelligkeit: Bericht generieren (PDF/Excel), per E-Mail an alle `recipients` senden
- `lastRun` aktualisieren, `nextRun` berechnen
- `active: false` ueberspringt den Bericht

---

### 11.17 Berichts-Generierung

```
POST   /api/v1/reports/generate                           -> Bericht ad-hoc generieren
```

Request:
```json
{
  "name": "Monatsbericht Februar",
  "moduleId": "finanzen",
  "dateFrom": "2026-02-01",
  "dateTo": "2026-02-28",
  "format": "pdf|excel"
}
```

Response:
```json
{
  "downloadUrl": "/api/v1/reports/download/uuid",
  "expiresAt": "2026-02-22T16:00:00Z"
}
```

Download:
```
GET    /api/v1/reports/download/{reportId}                -> Datei herunterladen
```

**Backend-Logik:**
- Asynchron: Report-Generierung kann dauern, deshalb `downloadUrl` in der Response
- Datei wird temporaer gespeichert (z.B. 24h) und dann geloescht
- Format-spezifisch: PDF braucht ein PDF-Template/Generator, Excel braucht eine XLSX-Library
- Report-Inhalte haengen vom Modul ab (Finanzen: Umsatz/Kosten, CRM: Pipeline, etc.)

---

## E. Dokumente (Documents) Enhancements

### 11.18 Template Gallery

```
GET    /api/v1/documents/templates                        -> Alle Dokumentvorlagen
GET    /api/v1/documents/templates/{id}                   -> Einzelne Vorlage
POST   /api/v1/documents/templates/{id}/create             -> Dokument aus Vorlage erstellen
```

Query params (GET list): `?category=vertraege|briefe|formulare|rechnungen|berichte&search=`

Template model:
```json
{
  "id": "uuid",
  "name": "Arbeitsvertrag",
  "description": "Standard-Arbeitsvertrag nach deutschem Recht",
  "category": "vertraege",
  "format": ".docx",
  "tenantId": "uuid|null",
  "isSystem": true,
  "createdAt": "ISO-8601"
}
```

Create from Template Request:
```json
{
  "folderId": "uuid|null",
  "fileName": "Arbeitsvertrag_Mueller.docx"
}
```

Create from Template Response:
```json
{
  "fileId": "uuid",
  "fileName": "Arbeitsvertrag_Mueller.docx",
  "folderId": "uuid|null"
}
```

**Backend-Logik:**
- `isSystem: true` fuer vordefinierte Vorlagen (nicht loeschbar), `isSystem: false` fuer benutzerdefinierte
- `tenantId: null` fuer System-Vorlagen, gesetzt fuer Tenant-spezifische
- Create: Vorlage-Datei kopieren, neuen Dateinamen vergeben, in Zielordner ablegen
- Kategorien: `vertraege`, `briefe`, `formulare`, `rechnungen`, `berichte`
- System liefert ~12 Standard-Vorlagen mit (Seed-Daten)

---

### 11.19 Share Links (Freigabe-Links)

```
POST   /api/v1/documents/files/{fileId}/share-links       -> Share Link erstellen
GET    /api/v1/documents/files/{fileId}/share-links        -> Share Links fuer Datei auflisten
DELETE /api/v1/documents/share-links/{linkId}              -> Share Link widerrufen
```

Create Request:
```json
{
  "expiryDays": 7,
  "permission": "view|download",
  "password": "optional-password|null"
}
```

ShareLink model:
```json
{
  "id": "uuid",
  "fileId": "uuid",
  "token": "random-slug",
  "url": "https://kmuhub.app/s/abc12345-xyz",
  "permission": "view|download",
  "hasPassword": true,
  "expiresAt": "2026-03-01T00:00:00Z|null",
  "accessCount": 5,
  "createdAt": "ISO-8601",
  "createdBy": "user-uuid"
}
```

**Oeffentlicher Zugang (OHNE Auth):**

```
GET    /api/v1/public/share/{token}                       -> Datei-Metadaten abrufen
GET    /api/v1/public/share/{token}/download               -> Datei herunterladen
POST   /api/v1/public/share/{token}/verify-password        -> Passwort pruefen
```

Share Page Response:
```json
{
  "fileName": "Vertrag_Meier_AG.pdf",
  "fileSize": 245760,
  "mimeType": "application/pdf",
  "permission": "view|download",
  "requiresPassword": true,
  "isExpired": false
}
```

**Backend-Logik:**
- `token` ist ein zufaelliger Slug (URL-safe, z.B. 16 Zeichen)
- `expiresAt` berechnet aus `expiryDays` (0 = kein Ablauf)
- `password` wird serverseitig gehasht gespeichert (bcrypt)
- `accessCount` wird bei jedem Zugriff inkrementiert
- Oeffentliche Endpoints: Kein JWT noetig, Rate Limiting (30 Zugriffe pro IP pro Minute)
- Wenn `requiresPassword: true`, muss zuerst `/verify-password` aufgerufen werden (gibt ein temporaeres Token zurueck)
- `permission: view` erlaubt Vorschau aber keinen Download; `download` erlaubt beides

---

### 11.20 Office Document Opener Metadata

```
GET    /api/v1/documents/files/{fileId}/office-url         -> Office-Oeffnungs-URL fuer Electron
```

Response:
```json
{
  "fileId": "uuid",
  "fileName": "Bericht_Q1.docx",
  "localPath": "/tmp/kmuhub-cache/uuid-Bericht_Q1.docx",
  "protocol": "ms-word:ofe|u|",
  "downloadUrl": "/api/v1/documents/files/uuid/content"
}
```

**Backend-Logik:**
- Fuer Electron-Desktop-App: Datei lokal cachen, dann ueber MS Office Protocol Handler oeffnen
- `protocol` basiert auf MIME-Typ (Word: `ms-word:ofe|u|`, Excel: `ms-excel:ofe|u|`, PowerPoint: `ms-powerpoint:ofe|u|`)
- Fallback: Datei herunterladen und normal oeffnen

---

## F. DB Schema Suggestions

### Call History

```sql
CREATE TABLE call_history (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     UUID NOT NULL,
  user_id       UUID NOT NULL,
  contact_id    UUID,
  contact_name  VARCHAR(300) NOT NULL,
  direction     VARCHAR(10) NOT NULL CHECK (direction IN ('incoming', 'outgoing', 'missed')),
  call_type     VARCHAR(10) NOT NULL CHECK (call_type IN ('audio', 'video')),
  meeting_id    UUID,
  started_at    TIMESTAMPTZ NOT NULL,
  duration      SMALLINT DEFAULT 0,
  created_at    TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_call_history_user ON call_history(user_id);
CREATE INDEX idx_call_history_tenant ON call_history(tenant_id);
CREATE INDEX idx_call_history_date ON call_history(started_at);
```

### Notifications

```sql
CREATE TABLE notifications (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     UUID NOT NULL,
  user_id       UUID NOT NULL,
  title         VARCHAR(500) NOT NULL,
  body          TEXT,
  module_id     VARCHAR(50),
  priority      VARCHAR(10) DEFAULT 'normal' CHECK (priority IN ('low', 'normal', 'high', 'urgent')),
  is_read       BOOLEAN DEFAULT FALSE,
  deep_link     VARCHAR(500),
  group_key     VARCHAR(200),
  snoozed_until TIMESTAMPTZ,
  created_at    TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_notifications_user ON notifications(user_id);
CREATE INDEX idx_notifications_tenant ON notifications(tenant_id);
CREATE INDEX idx_notifications_unread ON notifications(user_id, is_read) WHERE is_read = FALSE;
CREATE INDEX idx_notifications_created ON notifications(created_at);
CREATE INDEX idx_notifications_snoozed ON notifications(snoozed_until) WHERE snoozed_until IS NOT NULL;

CREATE TABLE notification_event_types (
  id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  event_key             VARCHAR(100) NOT NULL UNIQUE,
  display_name          VARCHAR(200) NOT NULL,
  description           TEXT,
  module_id             VARCHAR(50),
  default_in_app        BOOLEAN DEFAULT TRUE,
  default_desktop_push  BOOLEAN DEFAULT TRUE
);

CREATE TABLE notification_preferences (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id         UUID NOT NULL,
  event_type_key  VARCHAR(100) NOT NULL,
  in_app          BOOLEAN DEFAULT TRUE,
  desktop_push    BOOLEAN DEFAULT TRUE,
  email           BOOLEAN DEFAULT FALSE,
  UNIQUE(user_id, event_type_key)
);
CREATE INDEX idx_notif_prefs_user ON notification_preferences(user_id);

CREATE TABLE notification_quiet_hours (
  user_id     UUID PRIMARY KEY,
  enabled     BOOLEAN DEFAULT FALSE,
  start_time  TIME NOT NULL DEFAULT '22:00',
  end_time    TIME NOT NULL DEFAULT '07:00',
  timezone    VARCHAR(50) DEFAULT 'Europe/Berlin'
);

CREATE TABLE notification_dnd (
  user_id     UUID PRIMARY KEY,
  active      BOOLEAN DEFAULT FALSE,
  expires_at  TIMESTAMPTZ
);

CREATE TABLE notification_mutes (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id       UUID NOT NULL,
  resource_type VARCHAR(50) NOT NULL,
  resource_id   UUID NOT NULL,
  created_at    TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE(user_id, resource_type, resource_id)
);
CREATE INDEX idx_notif_mutes_user ON notification_mutes(user_id);
```

### Video Settings

```sql
CREATE TABLE user_video_settings (
  user_id                UUID PRIMARY KEY,
  audio_input_device_id  VARCHAR(200) DEFAULT 'default',
  audio_output_device_id VARCHAR(200) DEFAULT 'default',
  video_input_device_id  VARCHAR(200) DEFAULT 'default',
  virtual_background     VARCHAR(20) DEFAULT 'none',
  noise_suppression      BOOLEAN DEFAULT TRUE,
  echo_cancellation      BOOLEAN DEFAULT TRUE,
  updated_at             TIMESTAMPTZ DEFAULT NOW()
);
```

### Reports

```sql
CREATE TABLE scheduled_reports (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id   UUID NOT NULL,
  name        VARCHAR(300) NOT NULL,
  report_id   UUID,
  schedule    VARCHAR(15) NOT NULL CHECK (schedule IN ('daily', 'weekly', 'monthly', 'quarterly')),
  active      BOOLEAN DEFAULT TRUE,
  recipients  TEXT[] NOT NULL DEFAULT '{}',
  last_run    TIMESTAMPTZ,
  next_run    TIMESTAMPTZ,
  created_by  UUID NOT NULL,
  created_at  TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_scheduled_reports_tenant ON scheduled_reports(tenant_id);
CREATE INDEX idx_scheduled_reports_next ON scheduled_reports(next_run) WHERE active = TRUE;

CREATE TABLE generated_reports (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id   UUID NOT NULL,
  name        VARCHAR(300) NOT NULL,
  module_id   VARCHAR(50),
  format      VARCHAR(10) NOT NULL,
  file_path   VARCHAR(500) NOT NULL,
  expires_at  TIMESTAMPTZ NOT NULL,
  created_by  UUID NOT NULL,
  created_at  TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_generated_reports_expires ON generated_reports(expires_at);
```

### Document Share Links

```sql
CREATE TABLE document_share_links (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  file_id       UUID NOT NULL REFERENCES document_files(id) ON DELETE CASCADE,
  token         VARCHAR(50) NOT NULL UNIQUE,
  permission    VARCHAR(10) NOT NULL CHECK (permission IN ('view', 'download')),
  password_hash VARCHAR(200),
  expires_at    TIMESTAMPTZ,
  access_count  INTEGER DEFAULT 0,
  created_by    UUID NOT NULL,
  created_at    TIMESTAMPTZ DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_share_links_token ON document_share_links(token);
CREATE INDEX idx_share_links_file ON document_share_links(file_id);

CREATE TABLE document_templates (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id   UUID,
  name        VARCHAR(300) NOT NULL,
  description TEXT,
  category    VARCHAR(30) NOT NULL,
  format      VARCHAR(10) NOT NULL,
  file_path   VARCHAR(500) NOT NULL,
  is_system   BOOLEAN DEFAULT FALSE,
  created_at  TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_doc_templates_tenant ON document_templates(tenant_id);
CREATE INDEX idx_doc_templates_category ON document_templates(category);
```

---

## G. Summary: Recommended Build Order

| Prio | Item | What | Effort |
|------|------|------|--------|
| 1 | Notifications CRUD | Basis-Entity mit Pagination, Read/ReadAll | ~200 LOC |
| 2 | Event Types + Preferences | Seed-Daten + Praeferenz CRUD | ~120 LOC |
| 3 | Notification WebSocket | Echtzeit-Push ueber bestehenden WS-Channel | ~80 LOC |
| 4 | Snooze + Quiet Hours + DND | Pro-User Einstellungen mit Scheduler | ~120 LOC |
| 5 | Resource Muting | Stummschaltungs-CRUD + Check bei Notification-Erstellung | ~60 LOC |
| 6 | Dashboard Stats | Cross-Module Aggregation mit Caching | ~100 LOC |
| 7 | Activity Feed + Upcoming | Activity-Log-Tabelle + Kalender-Query | ~80 LOC |
| 8 | Call History | CRUD + Paginierung + Search | ~100 LOC |
| 9 | Video Settings | Pro-User JSONB Settings | ~40 LOC |
| 10 | Active Calls | LiveKit Room-Status API + Teilnehmer | ~60 LOC |
| 11 | KPI Daten + Drill-Down | Modul-spezifische Aggregation | ~150 LOC |
| 12 | Period Comparison | Vergleichs-Berechnung zweier Perioden | ~80 LOC |
| 13 | DATEV BWA + SuSa | SKR03/04 Kontenrahmen-Aggregation | ~120 LOC |
| 14 | Scheduled Reports | CRUD + Cron-Job + E-Mail-Versand | ~120 LOC |
| 15 | Report Generation | PDF/Excel-Generierung + temporaere Dateien | ~100 LOC |
| 16 | Document Templates | Template CRUD + File-Kopie-Logik | ~80 LOC |
| 17 | Share Links | Link-Generierung, Password-Hash, oeffentlicher Zugang | ~120 LOC |
| 18 | Office Opener | Protocol-URL-Generierung + File-Download | ~40 LOC |

**Total: ~1,850 LOC Go**

---

## H. Cross-Module Dependencies

- **11.1 Active Calls -> LiveKit:** LiveKit Room List API fuer aktive Meetings
- **11.3 Snooze -> Scheduler:** Cron-Job der abgelaufene Snooze-Eintraege aufhebt
- **11.6 Quiet Hours -> Notification Service:** Quiet Hours pruefen vor Push-Versand
- **11.8 WebSocket -> Auth:** WebSocket-Verbindung muss authentifiziert sein
- **11.9 Dashboard Stats -> Alle Module:** Cross-Service-Queries fuer Aggregation
- **11.15 DATEV -> Finanzen:** BWA/SuSa-Daten aus dem Finanz-Modul
- **11.15 DATEV Export -> Wave 12 (Integration Settings):** Beraternummer + Mandantennummer
- **11.16 Scheduled Reports -> E-Mail Service:** E-Mail-Versand mit Attachment
- **11.19 Share Links -> Auth System:** Auth-Ausnahme fuer `/api/v1/public/share/*`
- **11.20 Office Opener -> Electron IPC:** Desktop-spezifisch, nur relevant fuer Electron-Builds

---

## I. Notes for Luke

- **Notifications sind das Herzstück:** Das Notification-System wird von ALLEN anderen Modulen genutzt. Jedes Modul wird serverseitig Notifications erstellen (Chat-Nachricht -> Notification, Task zugewiesen -> Notification, Rechnung ueberfaellig -> Notification, etc.). Design es als zentralen Service.
- **WebSocket-Architektur:** Die Frontend-Hooks nutzen bereits `wsManager.on('notification.new', ...)`. Der bestehende WebSocket-Channel kann einfach um den `notification.*` Event-Namespace erweitert werden.
- **Notification Preferences pruefen BEVOR eine Notification erstellt wird:** Wenn der User `desktop_push: false` fuer `chat.new_message` gesetzt hat, soll kein WebSocket-Push gesendet werden. Die In-App-Notification wird trotzdem erstellt (falls `in_app: true`).
- **DATEV BWA-Positionen:** Standard BWA hat 11 Positionen (Umsatzerloese, Materialaufwand, Rohertrag, Personalaufwand, Raumkosten, Betriebliche Steuern, Versicherungen, Abschreibungen, Sonstige Kosten, Betriebsergebnis, Vorläufiges Ergebnis). Wird spaeter mit DATEV-Integration (Wave 12) verknuepft.
- **Share Links — Sicherheit:** Token muss kryptographisch zufaellig sein (crypto/rand, nicht math/rand). Password mit bcrypt hashen. Rate Limiting auf oeffentliche Endpoints.
- **Template Gallery — Seed-Daten:** Die 12 Templates aus dem Frontend koennen als System-Seed-Daten eingefuegt werden. Die eigentlichen Template-Dateien (.docx, .xlsx) muessen im Backend-Repository liegen.
- **Call History — Events:** Anrufe werden nicht manuell erstellt, sondern automatisch bei LiveKit-Events (Room created, Participant joined/left, Room ended). Ein LiveKit-Webhook kann diese Events abfangen und in die `call_history`-Tabelle schreiben.
- **Dashboard Stats — Caching:** Redis-Cache mit 60s TTL empfohlen. Bei jedem relevanten Event (neue Nachricht, Task erstellt, etc.) den Cache invalidieren oder einfach TTL ablaufen lassen.
- **Report Generation — Libraries:** PDF: `github.com/jung-kurt/gofpdf` oder `wkhtmltopdf`. Excel: `github.com/xuri/excelize`. DATEV-Export: Eigenes Format (ASCII mit Headerzeilen).

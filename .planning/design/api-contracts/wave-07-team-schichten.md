# Wave 7 API Contract: Team/HR + Schichtplanung

## Overview
Wave 7 adds HR integrations, digital personnel files, onboarding checklists, org chart,
employee self-service, and enhanced shift planning with ArbZG validation.

---

## Team / HR Endpoints

### 7.1 HR Integrations

```
GET    /api/v1/hr/integrations                  → List connected HR integrations
POST   /api/v1/hr/integrations/:id/connect      → Initiate OAuth for integration
POST   /api/v1/hr/integrations/:id/sync         → Trigger manual sync
DELETE /api/v1/hr/integrations/:id/disconnect    → Disconnect integration
GET    /api/v1/hr/integrations/:id/status        → Sync status + last sync timestamp
```

Response: `{ id, name, status: 'connected'|'disconnected'|'syncing'|'error', lastSync, syncCount }`

### 7.2 Multi-Country Deductions (Read-only display)

```
GET    /api/v1/hr/deductions/preview?country=DE&gross=4200  → Preview deductions by country
```

Response: `{ country, currency, grossSalary, deductions: [{ label, rate, amount, type }], netSalary }`

Supported countries: `DE`, `CH`, `AT`

### 7.3 Digital Personnel Documents

```
GET    /api/v1/hr/documents                      → List all documents (filterable)
GET    /api/v1/hr/documents/:id                  → Document metadata
POST   /api/v1/hr/documents                      → Upload document (multipart)
DELETE /api/v1/hr/documents/:id                  → Delete document
GET    /api/v1/hr/documents/:id/download         → Download file
```

Query params: `?employeeId=&category=vertrag|zeugnis|zertifikat|ausweis|sonstiges&status=aktuell|bald_ablaufend|abgelaufen`

Document model:
```json
{
  "id": "string",
  "employeeId": "string",
  "title": "string",
  "category": "vertrag|zeugnis|zertifikat|ausweis|sonstiges",
  "fileName": "string",
  "fileSize": "number",
  "mimeType": "string",
  "uploadedAt": "ISO-8601",
  "uploadedBy": "string",
  "expiresAt": "ISO-8601|null",
  "status": "aktuell|bald_ablaufend|abgelaufen",
  "notes": "string|null"
}
```

### 7.4 Onboarding Checklists

```
GET    /api/v1/hr/onboarding/templates           → List onboarding templates
POST   /api/v1/hr/onboarding/templates           → Create template
PUT    /api/v1/hr/onboarding/templates/:id       → Update template
DELETE /api/v1/hr/onboarding/templates/:id       → Delete template

GET    /api/v1/hr/onboarding                     → List active onboardings
POST   /api/v1/hr/onboarding                     → Start onboarding for employee
PATCH  /api/v1/hr/onboarding/:id/items/:itemId   → Toggle checklist item
DELETE /api/v1/hr/onboarding/:id                 → Cancel onboarding
```

Template model:
```json
{
  "id": "string",
  "name": "string",
  "items": [{ "id": "string", "label": "string", "assignee": "string|null", "dueInDays": "number|null" }]
}
```

Active onboarding model:
```json
{
  "id": "string",
  "employeeId": "string",
  "templateId": "string",
  "startDate": "ISO-8601",
  "items": [{ "id": "string", "label": "string", "completed": "boolean", "assignee": "string|null" }]
}
```

### 7.5 Org Chart

```
GET    /api/v1/hr/org-chart                      → Org tree (nested JSON)
```

Response: Recursive tree of `{ id, name, role, department, email, managerId, children: [...] }`

### 7.6 Self-Service

```
GET    /api/v1/hr/self-service/profile           → Current user profile (scoped)
GET    /api/v1/hr/self-service/leave-balance      → Leave balances
GET    /api/v1/hr/self-service/requests           → Own leave requests
POST   /api/v1/hr/self-service/requests           → Submit leave request
GET    /api/v1/hr/self-service/salary-statements  → List salary statement PDFs
GET    /api/v1/hr/self-service/salary-statements/:id/download → Download PDF
GET    /api/v1/hr/self-service/time-account       → Overtime + Gleitzeit balances
```

---

## Schichtplanung Endpoints

### 7.7 Surcharges (Zuschlaege)

```
GET    /api/v1/shifts/surcharge-rules             → List surcharge rules
POST   /api/v1/shifts/surcharge-rules             → Create rule
PUT    /api/v1/shifts/surcharge-rules/:id         → Update rule
DELETE /api/v1/shifts/surcharge-rules/:id         → Delete rule
```

Surcharge rule model:
```json
{
  "id": "string",
  "name": "string",
  "condition": "night|weekend|holiday",
  "ratePercent": "number",
  "templateIds": ["string"]
}
```

### 7.8 + 7.9 ArbZG Validation + Conflict Detection

```
GET    /api/v1/shifts/validate?weekStart=2026-02-16  → Validate week's assignments
```

Response:
```json
{
  "violations": [{
    "employeeId": "string",
    "type": "max_hours|rest_period|break_missing|consecutive_days|double_booking",
    "severity": "warning|error",
    "message": "string",
    "affectedDates": ["ISO-8601"]
  }]
}
```

### 7.10 Holiday Calendar

```
GET    /api/v1/shifts/holidays?year=2026&country=DE&state=BY  → Holidays for year/country/state
```

Response: `{ holidays: [{ date: "ISO-8601", name: "string", type: "public|regional" }] }`

### 7.11 Drag-and-Drop (existing assignment endpoints)

```
PATCH  /api/v1/shifts/assignments/:id             → Update assignment (move to new date/employee)
```

Body: `{ userId?: "string", date?: "ISO-8601" }`

### 7.12 Self-Service Availability

```
GET    /api/v1/shifts/availability/:userId         → Get availability for user
PUT    /api/v1/shifts/availability/:userId         → Update availability
```

Availability model:
```json
{
  "userId": "string",
  "weekdays": {
    "0": "green|yellow|red",
    "1": "green|yellow|red",
    "2": "green|yellow|red",
    "3": "green|yellow|red",
    "4": "green|yellow|red",
    "5": "green|yellow|red",
    "6": "green|yellow|red"
  }
}
```

### 7.13 PDF Export

```
GET    /api/v1/shifts/export/pdf?weekStart=2026-02-16  → Generate PDF of weekly plan
```

Response: Binary PDF file (Content-Type: application/pdf)

---

## Notes for Luke

- **Deutschland first:** Holiday calendar defaults to DE. Country/state configurable.
- **ArbZG rules hardcoded for DE:** §3 (48h max), §5 (11h Ruhezeit), §9 (6 Tage max). Configurable per country later.
- **HR Integrations:** DATEV Lohn is the primary integration. OAuth flow needed. Others are placeholder.
- **Personnel documents:** File storage (S3/MinIO). Max 10MB per file. Categories are enum.
- **Onboarding templates:** Seeded with 4 default templates (IT, HR, Fach, Compliance).
- **Org chart:** Derived from employee.managerId relationships. No separate data model needed.
- **Self-service:** Scoped to authenticated user. Read-only for personal data (change requests via HR).
- **Surcharges:** Display-only in frontend. Actual calculation happens in payroll integration (DATEV).

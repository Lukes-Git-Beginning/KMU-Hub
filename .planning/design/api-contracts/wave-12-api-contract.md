# Wave 12 API Contract: Integration Panels + Settings

> **From:** Darien (Frontend) | **For:** Luke (Backend)
> **Date:** 2026-02-22 | **Status:** Entwurf
> **Frontend branch:** `design/brainstorm`
> **Estimated backend effort:** ~800 LOC Go

---

## Overview

Wave 12 adds a unified **Integration Management** system with 12 integrations across 5 categories. The frontend has a grid of integration cards, drill-down config panels, and a Zustand store for connection state. Three integrations (Slack, Teams Webhook, Custom Webhook) already have API-driven backends from earlier waves; the remaining 9 are new and need backend support.

**What the frontend currently does:**
- Grid of 12 integration cards organized by category (Buchhaltung, Kommunikation, Dokumente, Video, Marketing)
- Per-integration connection status tracking (connected/disconnected/syncing/error)
- GenericIntegrationPanel: field-driven config (text/password/url/select/switch/readonly), connect/disconnect/sync/test buttons
- DATEVConfigPanel: Custom panel with Beraternummer, Mandantennummer, Kontenrahmen (SKR03/04), export settings
- BexioConfigPanel: Custom panel with OAuth2 flow mock, sync options (Kontakte/Rechnungen/Produkte)
- Integration store persists field values + connection state per integration ID in localStorage

**What already exists (from earlier waves):**
- `POST /api/v1/integrations` — Create Slack/Teams/Webhook integration
- `GET /api/v1/integrations` — List integration configs
- `DELETE /api/v1/integrations/{id}` — Delete integration
- `POST /api/v1/integrations/{id}/test` — Test connection
- `GET/POST/DELETE /api/v1/integrations/{id}/channel-mappings` — Channel mappings
- `GET/POST/DELETE /api/v1/account-links` — Account linking for Slack/Teams

---

## A. Generic Integration Config Endpoints

These endpoints cover all 9 new integrations that are NOT Slack/Teams/Webhook:
- DATEV Rechnungswesen, Bexio, MS Teams (Graph), WhatsApp Business, Skribble, Collabora Online, Zoom, LiveKit, Brevo

### Integration Config CRUD

```
GET    /api/v1/integration-configs                        -> Alle Konfigurationen des Tenants
GET    /api/v1/integration-configs/{integrationId}        -> Konfiguration fuer eine Integration
PUT    /api/v1/integration-configs/{integrationId}        -> Konfiguration speichern/aktualisieren
DELETE /api/v1/integration-configs/{integrationId}        -> Konfiguration loeschen (disconnect)
```

`integrationId` ist der String-Schluessel aus der Integration Registry (z.B. `datev-rechnungswesen`, `bexio`, `zoom`, `livekit`, etc.)

GET All Response:
```json
{
  "configs": [
    {
      "integrationId": "datev-rechnungswesen",
      "status": "connected|disconnected|syncing|error",
      "connectedAt": "ISO-8601|null",
      "lastSync": "ISO-8601|null",
      "fieldValues": {
        "beraternummer": "12345",
        "mandantennummer": "00001",
        "kontenrahmen": "skr03",
        "exportFormat": "datev_connect"
      }
    },
    {
      "integrationId": "zoom",
      "status": "connected",
      "connectedAt": "2026-02-20T10:00:00Z",
      "lastSync": "2026-02-22T08:00:00Z",
      "fieldValues": {
        "autoRecord": false,
        "waitingRoom": true,
        "defaultDuration": "60"
      }
    }
  ]
}
```

GET Single Response:
```json
{
  "integrationId": "datev-rechnungswesen",
  "status": "connected",
  "connectedAt": "2026-02-20T10:00:00Z",
  "lastSync": "2026-02-22T08:00:00Z",
  "fieldValues": {
    "beraternummer": "12345",
    "mandantennummer": "00001",
    "kontenrahmen": "skr03",
    "exportFormat": "datev_connect",
    "autoExport": true,
    "exportInterval": "monthly"
  }
}
```

PUT Request (erstellt oder aktualisiert):
```json
{
  "fieldValues": {
    "beraternummer": "12345",
    "mandantennummer": "00001",
    "kontenrahmen": "skr03",
    "exportFormat": "datev_connect",
    "autoExport": true,
    "exportInterval": "monthly"
  }
}
```

**Backend-Logik:**
- `integrationId` ist ein String-Key (NICHT UUID), referenziert die Integration-Registry
- `fieldValues` ist ein JSONB-Objekt mit beliebigen Key-Value-Paaren (String oder Boolean)
- `status` wird nur serverseitig gesetzt (nicht im PUT-Body)
- DELETE setzt den Status auf `disconnected` und loescht die `fieldValues` (oder Soft-Delete)
- Felder mit `type: password` muessen serverseitig verschluesselt gespeichert werden (AES-256 oder Vault)
- Bei GET: Password-Felder als maskiert zurueckgeben (z.B. `"***"` statt Klartext), AUSSER der User hat explizit "Anzeigen" geklickt (optional: separater Endpoint)

---

### 12.1 Integration Connect

```
POST   /api/v1/integration-configs/{integrationId}/connect  -> Integration verbinden
```

Request (fuer API-Key-basierte Integrationen):
```json
{}
```

Request (fuer OAuth2-basierte Integrationen wie Bexio, Zoom):
```json
{
  "authorizationCode": "oauth-code-from-redirect"
}
```

Response:
```json
{
  "integrationId": "datev-rechnungswesen",
  "status": "connected",
  "connectedAt": "2026-02-22T14:30:00Z"
}
```

**Backend-Logik:**
- **API-Key-Integrationen** (DATEV, Skribble, Teams Graph, WhatsApp, Brevo): Validiert die gespeicherten Credentials durch einen Test-Call an die externe API
- **OAuth2-Integrationen** (Bexio, Zoom): Fuehrt OAuth2 Token-Exchange durch (Authorization Code -> Access Token + Refresh Token)
- **Server-Integrationen** (Collabora, LiveKit): Prueft die Server-Erreichbarkeit (Health-Check/Ping)
- Setzt `status` auf `connected` bei Erfolg, `error` bei Fehler
- Speichert OAuth2-Tokens verschluesselt (NICHT in `fieldValues`)

---

### 12.2 Integration Disconnect

```
POST   /api/v1/integration-configs/{integrationId}/disconnect -> Integration trennen
```

Response:
```json
{
  "integrationId": "zoom",
  "status": "disconnected"
}
```

**Backend-Logik:**
- Setzt `status` auf `disconnected`
- Bei OAuth2: Refresh Token widerrufen (Token Revocation)
- `fieldValues` bleiben erhalten (nur Status aendert sich)
- `connectedAt` und `lastSync` auf null setzen

---

### 12.3 Integration Sync

```
POST   /api/v1/integration-configs/{integrationId}/sync      -> Synchronisation ausloesen
```

Response:
```json
{
  "integrationId": "bexio",
  "status": "syncing",
  "syncStartedAt": "2026-02-22T14:30:00Z"
}
```

**BACKEND-DEP: Asynchroner Sync-Worker.**

**Backend-Logik:**
- Setzt `status` auf `syncing` sofort
- Startet asynchronen Job (Message-Queue oder Go-Routine):
  - Bexio: Kontakte/Rechnungen/Produkte synchronisieren
  - DATEV: Export generieren und uebermitteln
  - Brevo: Kontaktlisten synchronisieren
  - Zoom: Meeting-Daten aktualisieren
- Bei Abschluss: `status` auf `connected`, `lastSync` aktualisieren
- Bei Fehler: `status` auf `error`, Fehler loggen
- WebSocket-Event: `integration.sync_complete` an den aufrufenden User

---

### 12.4 Integration Test

```
POST   /api/v1/integration-configs/{integrationId}/test      -> Verbindung testen
```

Response:
```json
{
  "success": true,
  "message": "Verbindung erfolgreich",
  "latencyMs": 120
}
```

Fehler-Response:
```json
{
  "success": false,
  "message": "API-Schluessel ungueltig",
  "errorCode": "auth_failed"
}
```

**Backend-Logik:**
- Fuehrt einen leichtgewichtigen API-Call gegen den externen Service aus
- Misst Latenz
- Typische Tests:
  - DATEV: `/api/v1/ping` oder Mandanten-Abfrage
  - Bexio: GET `/2.0/contacts?limit=1`
  - Zoom: GET `/v2/users/me`
  - LiveKit: Room-List-Abfrage
  - Skribble: API-Status-Check
  - Brevo: GET `/v3/account`
  - WhatsApp: `/v18.0/me`
  - Collabora: WOPI Discovery Endpoint
  - Teams Graph: GET `/v1.0/me`

---

## B. DATEV-Spezifische Konfiguration

DATEV nutzt den generischen Config-Endpoint mit folgenden `fieldValues`:

```json
{
  "beraternummer": "12345",
  "mandantennummer": "00001",
  "kontenrahmen": "skr03|skr04",
  "exportFormat": "datev_connect|csv_export",
  "autoExport": true,
  "exportInterval": "monthly|quarterly",
  "buchungsstapelName": "KMU Hub",
  "sachkontenlaenge": "4",
  "wirtschaftsjahrBeginn": "01"
}
```

### DATEV Export Trigger

```
POST   /api/v1/integration-configs/datev-rechnungswesen/export -> DATEV-Export ausloesen
```

Request:
```json
{
  "period": "2026-02",
  "type": "buchungsstapel|stammdaten|bwa|susa",
  "format": "datev_connect|csv"
}
```

Response:
```json
{
  "exportId": "uuid",
  "status": "generating",
  "downloadUrl": "/api/v1/integration-configs/datev-rechnungswesen/export/uuid/download"
}
```

Download:
```
GET    /api/v1/integration-configs/datev-rechnungswesen/export/{exportId}/download
```

**Backend-Logik:**
- `buchungsstapel`: Alle Buchungen des Zeitraums im DATEV-Format (ASCII mit Header)
- `stammdaten`: Konten, Kunden, Lieferanten
- `bwa`: Betriebswirtschaftliche Auswertung (siehe Wave 11.15)
- `susa`: Summen und Salden (siehe Wave 11.15)
- `datev_connect`: Direkte Uebermittlung an DATEV (benoetigt DATEV Connect Online API)
- `csv`: Lokaler Download als CSV-Datei
- `beraternummer` + `mandantennummer` werden in den Export-Header geschrieben
- `kontenrahmen` bestimmt die Kontonummern (SKR03 vs. SKR04)

---

## C. Bexio-Spezifische Konfiguration

Bexio nutzt OAuth2 und den generischen Config-Endpoint mit folgenden `fieldValues`:

```json
{
  "syncContacts": true,
  "syncInvoices": true,
  "syncProducts": true,
  "syncDirection": "bidirectional|kmuhub_to_bexio|bexio_to_kmuhub",
  "defaultAccount": "1100"
}
```

### Bexio OAuth2 Flow

```
GET    /api/v1/integration-configs/bexio/auth-url          -> OAuth2 Redirect-URL generieren
POST   /api/v1/integration-configs/bexio/callback           -> OAuth2 Callback verarbeiten
```

Auth URL Response:
```json
{
  "authUrl": "https://idp.bexio.com/authorize?client_id=xxx&redirect_uri=xxx&scope=xxx&state=xxx"
}
```

Callback Request:
```json
{
  "code": "authorization-code",
  "state": "csrf-state-token"
}
```

Callback Response:
```json
{
  "integrationId": "bexio",
  "status": "connected",
  "connectedAt": "2026-02-22T14:30:00Z",
  "companyName": "Muster GmbH"
}
```

**Backend-Logik:**
- OAuth2 Credentials: `BEXIO_CLIENT_ID`, `BEXIO_CLIENT_SECRET` in ENV
- Redirect URI: `${APP_URL}/api/v1/integration-configs/bexio/callback`
- Scopes: `openid profile email accounting`
- Access Token + Refresh Token verschluesselt speichern
- Token-Refresh automatisch bei API-Calls (wenn Access Token abgelaufen)
- `companyName` aus Bexio-API lesen nach erfolgreichem Login

### Bexio Sync Details

```
GET    /api/v1/integration-configs/bexio/sync-status       -> Sync-Status pro Entity
```

Response:
```json
{
  "contacts": { "synced": 142, "lastSync": "2026-02-22T08:00:00Z", "errors": 0 },
  "invoices": { "synced": 87, "lastSync": "2026-02-22T08:00:00Z", "errors": 2 },
  "products": { "synced": 34, "lastSync": "2026-02-22T08:00:00Z", "errors": 0 }
}
```

---

## D. OAuth2 Flow (Generisch fuer Bexio + Zoom)

Fuer alle OAuth2-Integrationen:

```
GET    /api/v1/integration-configs/{integrationId}/auth-url  -> Redirect-URL
POST   /api/v1/integration-configs/{integrationId}/callback   -> OAuth2 Callback
```

**Backend-Logik:**
- Pro Integration: Client-ID, Client-Secret, Scopes, Auth-URL, Token-URL in einer Config-Map
- State-Parameter mit CSRF-Schutz (Signed JWT oder Random Token in Redis)
- Nach erfolgreichem Callback: `status` auf `connected`, Tokens speichern
- Token-Refresh-Middleware fuer alle API-Calls an den externen Service

### OAuth2 Config Map

| Integration | Auth URL | Token URL | Scopes |
|------------|----------|-----------|--------|
| Bexio | `https://idp.bexio.com/authorize` | `https://idp.bexio.com/token` | `openid profile accounting` |
| Zoom | `https://zoom.us/oauth/authorize` | `https://zoom.us/oauth/token` | `meeting:write user:read` |

ENV Variables:
```bash
BEXIO_CLIENT_ID=...
BEXIO_CLIENT_SECRET=...
ZOOM_CLIENT_ID=...
ZOOM_CLIENT_SECRET=...
```

---

## E. WebSocket Events

```
integration.status_changed    -> { integrationId: string, status: string, connectedAt?: string }
integration.sync_complete     -> { integrationId: string, lastSync: string, errors: number }
integration.sync_error        -> { integrationId: string, error: string }
```

---

## F. DB Schema Suggestions

```sql
CREATE TABLE integration_configs (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id       UUID NOT NULL,
  integration_id  VARCHAR(50) NOT NULL,
  status          VARCHAR(15) DEFAULT 'disconnected' CHECK (status IN ('connected', 'disconnected', 'syncing', 'error')),
  field_values    JSONB NOT NULL DEFAULT '{}',
  encrypted_secrets JSONB,
  connected_at    TIMESTAMPTZ,
  last_sync       TIMESTAMPTZ,
  created_at      TIMESTAMPTZ DEFAULT NOW(),
  updated_at      TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE(tenant_id, integration_id)
);
CREATE INDEX idx_integration_configs_tenant ON integration_configs(tenant_id);
CREATE INDEX idx_integration_configs_status ON integration_configs(status);

CREATE TABLE integration_oauth_tokens (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id       UUID NOT NULL,
  integration_id  VARCHAR(50) NOT NULL,
  access_token    TEXT NOT NULL,
  refresh_token   TEXT,
  token_type      VARCHAR(20) DEFAULT 'Bearer',
  expires_at      TIMESTAMPTZ,
  scopes          TEXT,
  created_at      TIMESTAMPTZ DEFAULT NOW(),
  updated_at      TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE(tenant_id, integration_id)
);
CREATE INDEX idx_oauth_tokens_tenant ON integration_oauth_tokens(tenant_id);

CREATE TABLE integration_sync_log (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id       UUID NOT NULL,
  integration_id  VARCHAR(50) NOT NULL,
  entity_type     VARCHAR(50),
  synced_count    INTEGER DEFAULT 0,
  error_count     INTEGER DEFAULT 0,
  started_at      TIMESTAMPTZ DEFAULT NOW(),
  completed_at    TIMESTAMPTZ,
  status          VARCHAR(15) DEFAULT 'running' CHECK (status IN ('running', 'completed', 'error')),
  error_details   TEXT
);
CREATE INDEX idx_sync_log_tenant ON integration_sync_log(tenant_id);
CREATE INDEX idx_sync_log_integration ON integration_sync_log(integration_id);

-- DATEV-spezifisch: Export-History
CREATE TABLE datev_exports (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id       UUID NOT NULL,
  period          VARCHAR(7) NOT NULL,
  export_type     VARCHAR(20) NOT NULL,
  format          VARCHAR(20) NOT NULL,
  file_path       VARCHAR(500),
  status          VARCHAR(15) DEFAULT 'generating' CHECK (status IN ('generating', 'completed', 'error')),
  beraternummer   VARCHAR(20),
  mandantennummer VARCHAR(20),
  created_at      TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_datev_exports_tenant ON datev_exports(tenant_id);
```

---

## G. Summary: Recommended Build Order

| Prio | Item | What | Effort |
|------|------|------|--------|
| 1 | Integration Config CRUD | GET/PUT/DELETE fuer generische Configs | ~100 LOC |
| 2 | Connect/Disconnect | Status-Wechsel + Credential-Validation | ~80 LOC |
| 3 | Test Endpoint | Pro-Integration Test-Call Dispatcher | ~100 LOC |
| 4 | Sync Trigger | Async Sync-Job starten + Status-Updates | ~80 LOC |
| 5 | OAuth2 Flow (Bexio + Zoom) | Auth URL + Callback + Token Storage | ~120 LOC |
| 6 | DATEV Export | Buchungsstapel/Stammdaten/BWA/SuSa Export | ~120 LOC |
| 7 | Bexio Sync | Contact/Invoice/Product Sync-Worker | ~100 LOC |
| 8 | Secret Encryption | AES-256 Verschluesselung fuer API-Keys/Passwords | ~50 LOC |
| 9 | WebSocket Events | Status-Change + Sync-Complete Events | ~30 LOC |

**Total: ~780 LOC Go**

---

## H. Cross-Module Dependencies

- **12.1 Connect -> External APIs:** Jede Integration braucht einen externen API-Connector
- **12.3 Sync -> Message Queue:** Async Sync-Jobs sollten ueber eine Queue laufen (oder zumindest Go-Routines mit Timeout)
- **12.5 OAuth2 -> Auth System:** OAuth2 State-Tokens brauchen CSRF-Schutz
- **12.6 DATEV Export -> Finanzen-Modul:** BWA/SuSa-Daten aus dem Finanz-Service (gleich wie Wave 11.15)
- **12.6 DATEV Export -> Wave 11 Berichte:** Beraternummer/Mandantennummer fuer DATEV-Header
- **12.7 Bexio Sync -> CRM + Finanzen:** Kontakt-Sync mit CRM-Kontakten, Rechnungs-Sync mit Finanzen

---

## I. Notes for Luke

- **Generischer Ansatz:** Alle 9 neuen Integrationen teilen sich das gleiche `integration_configs`-Schema. Die `fieldValues` sind ein JSONB-Blob. Das Frontend weiss ueber die `IntegrationDefinition` welche Felder existieren, aber das Backend speichert einfach Key-Value-Paare.
- **Secrets NIEMALS in `fieldValues` im Klartext:** Felder mit `type: password` (API-Keys, Client-Secrets, Access-Tokens) muessen verschluesselt in `encrypted_secrets` gespeichert werden. Bei GET zurueckgeben als `"***"`. Encryption Key aus ENV: `INTEGRATION_ENCRYPTION_KEY`.
- **OAuth2 Token Management:** Access Tokens haben ein TTL. Refresh automatisch vor Ablauf. Wenn Refresh fehlschlaegt: `status` auf `error` setzen und User informieren (WebSocket-Event).
- **DATEV hat hoehere Prio als Bexio:** Deutschland first. DATEV ist der primaere Steuerberater-Export.
- **Bexio-Sync Richtung:** `bidirectional` ist komplex (Konflikte!). Starte mit `bexio_to_kmuhub` (Import-only), dann `kmuhub_to_bexio` (Export), dann bidirektional.
- **Test-Endpoint ist ein Dispatcher:** Je nach `integrationId` wird ein anderer externer Endpunkt aufgerufen. Implementiere als Switch/Map mit Integration-spezifischen Testern.
- **Bestehende Slack/Teams/Webhook-Integrationen bleiben unveraendert.** Sie nutzen weiterhin die alten Endpoints (`/api/v1/integrations`). Die neuen 9 Integrationen nutzen `/api/v1/integration-configs`. Das Frontend erkennt den Typ ueber `authMethod: 'existing'` vs. andere.
- **LiveKit-Integration:** Hier wird nur die Server-URL + API-Key gespeichert. Die eigentliche LiveKit-Nutzung (Rooms erstellen, Tokens generieren) bleibt in den bestehenden Video/Meeting-Endpoints.
- **Collabora-Integration:** Speichert Server-URL + WOPI-Discovery-URL. Die eigentliche WOPI-Integration (Token-Exchange, Document-Editing) bleibt in den Dokumente-Endpoints.

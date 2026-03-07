# Luke Antwort — 2026-02-25

> Antworten auf Backend-Koordination + Gast-Chat/Kundenportal

---

## 🎉 Überraschungen (Mehr fertig als erwartet!)

1. **CRM CRUD** - KOMPLETT FERTIG (voller CRUD für Contacts, Companies, Deals, Activities, Pipeline Stages, Custom Fields, Tags, Import/Export)
2. **DATEV-Export** - VOLL IMPLEMENTIERT (EXTF, SKR03, UTF-8 BOM)
3. **Phase 17.5 (Gast-Chat)** - KOMPLETT FERTIG (alle 3 Plans: Data Foundation, Services+Gateway, Frontend SPA)
4. **Email-Modul** - KOMPLETT FERTIG (39 RPCs, IMAP/SMTP, Threading, CRM-Linking, Attachments, Volltext-Suche)
5. **Gap-Analyse Wave 3** - ~60% bereits gebaut! (~7.200 LOC statt 15.300 LOC noch zu bauen)

---

## Backend-Status (Was steht schon)

### 1. CRM CRUD
**Status:** KOMPLETT FERTIG

- Contacts: Create, Get, List, Update, Delete
- Companies: Create, Get, List, Update, Delete
- Deals: Create, Get, List, Update, Delete, Move (Pipeline-Stage wechseln)
- Activities: Create, Get, List, Update, Delete, Complete
- Pipeline Stages: Create, Get, List, Update, Delete, Reorder
- Custom Fields: Create, Get, List, Update, Delete (pro Entity-Typ)
- Tags: Create, List, Delete, Assign, Remove
- Import/Export: CSV-Import, CSV-Export, vCard-Import, vCard-Export

**Code:**
- Proto: `backend/proto/crm/v1/crm.proto`
- Gateway: `backend/internal/gateway/route_crm.go` (Zeilen 62-114)

**Für Frontend:** TanStack Query Hooks direkt gegen diese Endpoints schreiben. Alle RESTful, alle mit Pagination, alle mit Tenant-Isolation.

### 2. File-Storage
**Status:** MinIO (S3-kompatibel), voll implementiert

**Config:** `backend/internal/config/config.go` (StorageConfig struct)

**Genutzt von:**
- Chat: `backend/internal/biz/chat/file/minio_store.go` - Dateianhänge
- Documents: `backend/internal/biz/document/file/service.go` - mit Versionierung
- Email: Attachments

**Limits:** 50MB (adjustable), Auto-Bucket-Creation

**Für Frontend:** Upload via `POST /api/v1/files/upload` (multipart/form-data), Download via presigned URLs

### 3. PDF-Generierung
**Status:** maroto v2, voll implementiert

**Library:** `github.com/johnfercher/maroto/v2 v2.3.3`

**Code:** `backend/internal/biz/pdf/generator.go`

**4 PDF-Typen:**
- Angebot (Quote)
- Rechnung (Invoice)
- Gutschrift (Credit Note)
- Mahnung (Dunning)

**Endpoints:** `GET /api/v1/finance/{type}/{id}/pdf`

**Features:** Logo, Absender, Empfänger, Positionen-Tabelle, Summen (Netto/USt/Brutto), Zahlungsbedingungen, Footer. Deutsche Formatierung.

**Für Frontend:** PDF-Vorschau via `iframe` oder `react-pdf`

### 4. LiveKit
**Status:** Komplett, feature-flagged

**Services:**
- Docker: `livekit:7880` + egress
- Token: `backend/internal/work/livekit/service.go`
- Room: `backend/internal/work/livekit/room_manager.go`

**Features:**
- Room erstellen/schließen
- Token mit Grants (publish, subscribe, room admin)
- Participant-Management
- Egress (Recording) vorbereitet

**Graceful Degradation:** Wenn `LIVEKIT_URL`, `LIVEKIT_API_KEY`, `LIVEKIT_API_SECRET` leer → kein Crash, nur Log

**Für Frontend:** Token holen via `POST /api/v1/video/token`, dann an `livekit-react-components` übergeben

### 5. WebSocket
**Status:** EIN Hub für alles

**Code:** `backend/internal/server/websocket.go` (823 LOC)

**Handelt:**
- Chat-Nachrichten (neue Nachricht, Edit, Delete, Reactions)
- Notifications (Echtzeit-Push an Client)
- Presence (Online/Offline/Away Status)
- Call-Signaling (Ring, Accept, Decline, End)
- Typing-Indicators

**Notification-Flow:** PostgreSQL LISTEN/NOTIFY → Gateway → WS Hub → Client

**Für Frontend:** Eine einzige WebSocket-Verbindung pro Client, Messages als typisierte JSON-Events (`{"type": "chat.message", "payload": {...}}`)

### 6. DATEV-Export
**Status:** VOLL IMPLEMENTIERT

**Code:** `backend/internal/biz/datev/exporter.go`

**Features:**
- EXTF Buchungsstapel CSV (Steuerberater-Import)
- SKR03 Kontenrahmen-Mapping (Standard für KMUs)
- UTF-8 BOM (Excel-Umlaute)
- Semikolon als Delimiter (DATEV-Standard)
- Tests existieren

**Endpoint:** `POST /api/v1/finance/export/datev`

**Für Frontend:** Button "DATEV-Export" → Zeitraum-Dialog (von/bis) → POST → CSV-Download

### 7. Phase 10 Email
**Status:** KOMPLETT FERTIG, 39 RPCs

**Features:**
- IMAP/SMTP Anbindung (beliebiger Mail-Provider)
- AES-verschlüsselte Passwörter in DB
- Folder Sync (Inbox, Sent, Drafts, Trash, Custom Folders)
- Message CRUD (List, Get, Search, Move, Delete, MarkRead)
- Threading (References + In-Reply-To Header Parsing)
- Send / Reply / Forward mit HTML-Body
- Drafts (Auto-Save, manuelles Speichern)
- Signaturen (CRUD, pro Account, HTML-basiert)
- CRM-Linking (Email an Contact/Deal/Company verknüpfen)
- Attachments via MinIO
- Volltext-Suche mit deutschem Stemmer (tsvector)

**Code:**
- Binary: `backend/cmd/email/main.go`
- Proto: `backend/proto/email/v1/email.proto`

**Für Frontend:** Mail-Module-Seite direkt gegen echte APIs bauen, kein Mock-Modus nötig

---

## Phase 17.5 Gast-Chat (KOMPLETT FERTIG)

### Alle 3 Plans ausgeführt:

1. **17.5-01:** Guest Session Management + DB Migration + Token-Generierung + Rate Limiter ✓
2. **17.5-02:** Chat-Service-Erweiterung + Gateway-Routes + WebSocket Hub Extension ✓
3. **17.5-03:** Frontend SPA (Standalone Vite+React) + Gateway Static Serving ✓

### Backend Details

**Guest Session Management (~300 LOC):**
- Token-Generierung (UUID v4)
- SHA-256 Hash in DB
- Konfigurierbarer Expiry (Default 7 Tage)
- Rate Limiting (30 msg/min pro Session)

**Chat-Modell-Erweiterung (~100 LOC):**
- Migration 000054
- `guest_sessions` Tabelle
- `chat_channels.is_guest_enabled` Flag
- `chat_messages.guest_session_id` (nullable)
- `chat_messages.created_by` nullable gemacht
- Constraint: Message muss entweder `created_by` ODER `guest_session_id` haben

**API-Endpoints (~400 LOC):**
- 5 Guest-Routes:
  - Session Create/Validate
  - Messages List/Get
  - Send Message
  - Get Channel Config

**WebSocket Extension:**
- Bestehenden Hub erweitert (NICHT neuer `/ws/guest` Endpoint)
- `GuestSessionValidator` Interface (bricht Import-Zyklen)
- Auth-Pfad: `?guest_token=xxx` statt JWT
- Routing: Gast-Messages NUR an verlinkten Channel
- Kein Zugriff: Presence anderer User, Channel-Liste, Suche, History anderer Channels
- Typing-Indicators funktionieren automatisch
- Rate Limiting: In-Memory Sliding Window (30 msg/min)

### Frontend SPA (~700 LOC TypeScript + 350 LOC CSS)

**Standalone Vite + React App:**
- Pfad: `guest-chat/`
- Route: `/guest/:token` (Gateway served als Static Files)
- Bundle: ~66KB gzipped

**8 Komponenten:**
1. `PreChatForm` - Name/Email eingeben
2. `ChatWindow` - Hauptfenster
3. `MessageList` - Nachrichten-Timeline
4. `MessageBubble` - Einzelne Nachricht
5. `MessageInput` - Eingabefeld + Send
6. `FileUpload` - Datei-Upload
7. `TypingIndicator` - "X schreibt..."
8. `ConnectionStatus` - Online/Offline/Reconnecting

**WebSocket Hook:**
- Auto-Reconnect mit Exponential Backoff + Jitter
- 30s Heartbeat
- Typed Events

**Theming:**
- CSS Custom Properties
- Primärfarbe + Logo aus Channel-Config
- Deutsch-only UI (keine i18n nötig)

**Kein Auth-Provider:**
- Token aus URL
- `X-Guest-Token` Header für API-Calls

---

## Security (Gast-Chat)

**Implementiert:**
- ✅ In-Memory Rate Limiter (Sliding Window, 30 Nachrichten/Minute)
- ✅ Token-basierte Sessions (UUID v4, SHA-256 gehasht)
- ✅ IP-basiertes Rate Limiting am Gateway
- ✅ Input-Sanitization (XSS-Schutz)
- ✅ Channel-Isolation (Gast nur eigener Channel)
- ✅ `X-Guest-Token` Header (eigener Auth-Pfad)
- ✅ File-Upload-Limits (10MB statt 50MB, nur Bilder+PDF)

**Noch offen (nicht kritisch für Beta):**
- CAPTCHA nach X Nachrichten bei Verdacht
- Token-Rotation (optional)
- Abuse-Reporting (Mitarbeiter sperrt Gast-Session)

---

## Gap-Analyse: Was schon existiert (Wave 3)

**FERTIG (Backend existiert, Frontend kann sofort anbinden):**

| Item | Was | Wo |
|------|-----|-----|
| 3.1 | CRM CRUD (Contacts, Companies, Deals) | `proto/crm/v1/crm.proto`, `route_crm.go` |
| 3.2 | Custom Fields (pro Entity-Typ) | CRM Service, voller CRUD |
| 3.3 | Firmen-Management | Companies CRUD inkl. Zuordnung zu Contacts |
| 3.9 | Import/Export (CSV + vCard) | CRM Service, 4 Endpoints |
| 3.12 | DATEV-Export | `biz/datev/exporter.go`, EXTF + SKR03 |
| 3.17 | GoBD-Konformität | Audit-Trail, Unveränderbarkeit, Versionierung |
| 3.18 | PDF-Generierung (4 Typen) | `biz/pdf/generator.go`, maroto v2 |

**TEILWEISE (Basis da, Erweiterung nötig):**

| Item | Was | Status |
|------|-----|--------|
| 3.11 | Belegkette | Quote→Invoice Konvertierung existiert, Gutschrift→Mahnung fehlt |

**FEHLT (muss noch gebaut werden):**

| Item | Was | Geschätzter Aufwand |
|------|-----|---------------------|
| 3.4 | Duplikaterkennung | ~400 LOC (Levenshtein + Email-Match) |
| 3.5 | Kontakt-Timeline (Activity Stream) | ~300 LOC (Aggregation-Query) |
| 3.7 | Consent-Management (DSGVO) | ~500 LOC (eigenes Modul) |
| 3.8 | Newsletter/Brevo-Integration | ~600 LOC (API-Client + Sync) |
| 3.13 | QR-Rechnung (CH) | ~400 LOC (Swiss QR Bill Standard) |
| 3.14 | ZUGFeRD (XML in PDF) | ~500 LOC (Factur-X/ZUGFeRD 2.1) |
| 3.19 | Stunden→Rechnung | ~300 LOC (Aggregation + Invoice-Generation) |
| 3.20 | FinAPI (Bankkonto) | ~800 LOC (OAuth + Transaction-Sync) |

---

## Korrigierte Backend-Schätzung

| Wave | Deine Schätzung | Korrigiert | Grund |
|------|-----------------|------------|-------|
| Wave 3 (CRM+Finanzen) | 3.100 LOC | ~1.200 LOC | 7 von 15 Items fertig |
| Wave 5 (Kalender) | 750 LOC | ~500 LOC | CalDAV-Basis existiert |
| Wave 10 (Dokumente) | 1.600 LOC | ~1.200 LOC | WOPI + MinIO + Versionierung fertig |
| Wave 11 (Email) | 1.950 LOC | ~1.000 LOC | 39 RPCs fertig, nur Frontend-spezifische Ergänzungen |
| Wave 13 (DSGVO+KI) | 2.300 LOC | ~2.300 LOC | Muss komplett neu gebaut werden |
| **Gesamt** | **15.300 LOC** | **~7.200 LOC** | **53% weniger** |

---

## 🎯 Action Items für Darien (Design)

### 1. Gast-Chat Design-Polish ⚡ Quick Win

**Was tun:**
- `guest-chat/src/globals.css` upgraden (aktuell plain CSS, kein Tailwind)
- Komponenten redesignen (siehe oben)
- Theme anpassen (Cozy-Style? Oder neues Design?)

**Wie starten:**
```bash
git pull origin main --rebase
cd guest-chat
npm install
npm run dev
```

### 2. Admin-UI für Gast-Channels 🆕 Neues Feature

**Was fehlt:**
- Toggle "Gast-Chat aktivieren" pro Channel
- Konfiguration:
  - Logo hochladen
  - Primärfarbe wählen
  - Willkommensnachricht ("Hallo! Wie können wir helfen?")
  - File-Upload-Limit (10MB default)

**Wo platzieren:**
- Chat-Modul (`modules/chat/`)?
- Oder Settings → Integrations?

**Deine Aufgabe:**
- Design-Mockup (Figma oder direkt Code)
- UI bauen, Luke macht Backend

### 3. Notification-Präferenzen ⚙️ Offene Frage

**Luke fragt:**
> "Wie wird der Mitarbeiter benachrichtigt wenn ein Gast schreibt?"

**Optionen:**
- WS-Push + Email?
- Nur WS-Push?
- Sound abspielen?
- Desktop-Notification?

**Deine Aufgabe:**
- Design für Notification-Settings
- Pro Channel oder global?

### 4. Von Mocks zu echten APIs 🔌 Migration

**Was fertig ist:**
- CRM, Email, DATEV, PDF, Custom Fields, Import/Export

**Deine Aufgabe:**
- TanStack Query Hooks schreiben
- Mock-Daten durch API-Calls ersetzen
- Testing gegen echtes Backend

**Luke sagt explizit:**
> "Du kannst die Mail-Module-Seite direkt gegen echte APIs bauen. Kein Mock-Modus nötig."

---

## Nächste Schritte

**Luke:**
1. Phase 18 (Bexio-Integration) - nächste Phase
2. Phase 19 (Abacus + RmA)
3. Phase 20 (Plugin System)
4. Wave-3-Lücken nach Beta-Scope-Entscheidung

**Darien:**
1. Gast-Chat Design-Upgrade? (Quick Win, sichtbar)
2. Admin-UI designen? (Neues Feature)
3. Echte APIs anbinden? (Technisch)
4. Weiteres Design-Feature?

---

## Gepusht auf design/brainstorm (2026-02-25)

1. `FEATURE-BRAINSTORM.md` - 103 Features, alle bewertet
2. `STRATEGY.md` - Digitale Souveränität + MS Office Koexistenz
3. `PRICING.md v2` - Role-Based Pricing + Einmalkauf + Branchenpakete

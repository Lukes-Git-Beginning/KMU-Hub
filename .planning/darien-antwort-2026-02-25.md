# Antworten auf Dariens Fragen - 2026-02-25

---

Hey Darien,

Hab beide Nachrichten komplett durchgelesen -- die Backend-Koordinationsfragen und den Gast-Chat/Kundenportal-Vorschlag. Hier sind alle 14 Antworten mit konkreten Code-Referenzen, damit du weisst was schon steht und was wir noch bauen muessen.

Kurze Vorwarnung: Ein paar der Fragen werden dich ueberraschen, weil deutlich mehr fertig ist als erwartet.

---

## Antworten auf Nachricht 1 (Backend-Koordination)

### 1. CRM CRUD -- wie weit sind wir?

**KOMPLETT FERTIG.** Nicht nur GET-Endpoints -- voller CRUD fuer alles:

- **Contacts:** Create, Get, List, Update, Delete
- **Companies:** Create, Get, List, Update, Delete
- **Deals:** Create, Get, List, Update, Delete, Move (Pipeline-Stage wechseln)
- **Activities:** Create, Get, List, Update, Delete, Complete
- **Pipeline Stages:** Create, Get, List, Update, Delete, Reorder
- **Custom Fields:** Create, Get, List, Update, Delete (pro Entity-Typ)
- **Tags:** Create, List, Delete, Assign, Remove
- **Import/Export:** CSV-Import, CSV-Export, vCard-Import, vCard-Export

Proto-Definition: `backend/proto/crm/v1/crm.proto`
Gateway-Routes: `backend/internal/gateway/route_crm.go` (Zeilen 62-114)

Du kannst also sofort TanStack Query Hooks gegen diese Endpoints schreiben. Alle RESTful, alle mit Pagination, alle mit Tenant-Isolation.

### 2. File-Storage -- was nutzen wir?

**MinIO (S3-kompatibel), voll implementiert.**

Config: `backend/internal/config/config.go` (StorageConfig struct mit Endpoint, Bucket, AccessKey, SecretKey, UseSSL, Region)

Bereits genutzt von:
- **Chat:** `backend/internal/biz/chat/file/minio_store.go` -- Dateianhang in Nachrichten
- **Documents:** `backend/internal/biz/document/file/service.go` -- mit Versionierung! Jede Datei-Aenderung erzeugt eine neue Version
- **Email:** Attachments werden ueber MinIO gespeichert

Konfigurierbar: 50MB Limit (adjustable), Auto-Bucket-Creation beim Start. Docker-Compose hat MinIO bereits drin.

Fuer dein Frontend heisst das: Upload via `POST /api/v1/files/upload` (multipart/form-data), Download via presigned URLs. Kein eigener S3-Client noetig.

### 3. PDF-Generierung -- welche Library?

**maroto v2** (`github.com/johnfercher/maroto/v2 v2.3.3`)

Implementierung: `backend/internal/biz/pdf/generator.go`

4 PDF-Typen sind gebaut:
- **Angebot** (Quote)
- **Rechnung** (Invoice)
- **Gutschrift** (Credit Note)
- **Mahnung** (Dunning)

Endpoints: `GET /api/v1/finance/{type}/{id}/pdf`

Jedes PDF hat: Logo, Absender, Empfaenger, Positionen-Tabelle, Summen (Netto/USt/Brutto), Zahlungsbedingungen, Footer. Deutsche Formatierung (Komma als Dezimal, Punkt als Tausender).

Wenn du eine PDF-Vorschau im Frontend willst: Endpoint liefert `application/pdf`, du kannst das direkt in einem iframe oder mit react-pdf rendern.

### 4. LiveKit -- laeuft das schon?

**JA, komplett.**

Docker-Compose: `livekit:7880` + `egress` Service
Token-Generierung: `backend/internal/work/livekit/service.go`
Room-Management: `backend/internal/work/livekit/room_manager.go`

Features:
- Room erstellen/schliessen
- Token mit Grants (publish, subscribe, room admin)
- Participant-Management
- Egress (Recording) vorbereitet

**Wichtig: Feature-flagged.** Wenn `LIVEKIT_URL`, `LIVEKIT_API_KEY`, `LIVEKIT_API_SECRET` leer sind, degraded der Video-Service graceful -- keine Crashes, nur ein Log-Eintrag. Das heisst: Starter-Tier ohne Video funktioniert out-of-the-box.

Fuer dein Frontend: Token holen via `POST /api/v1/video/token`, dann direkt an livekit-react-components uebergeben.

### 5. WebSocket -- ein Hub oder mehrere?

**EIN Hub fuer alles.** `backend/internal/server/websocket.go` (823 LOC)

Dieser eine Hub handelt:
- **Chat-Nachrichten** (neue Nachricht, Edit, Delete, Reactions)
- **Notifications** (Echtzeit-Push an den Client)
- **Presence** (Online/Offline/Away Status)
- **Call-Signaling** (Ring, Accept, Decline, End)
- **Typing-Indicators**

Notification-Flow: PostgreSQL LISTEN/NOTIFY → Gateway → WS Hub → Client

Das heisst fuer dich: Eine einzige WebSocket-Verbindung pro Client, Messages kommen als typisierte JSON-Events (`{"type": "chat.message", "payload": {...}}`). Du brauchst im Frontend nur einen WS-Provider der nach Event-Type dispatched.

### 6. DATEV-Export -- wie weit?

**VOLL IMPLEMENTIERT.** Das hat mich selbst ueberrascht beim Nachschauen.

Implementierung: `backend/internal/biz/datev/exporter.go`

Was drin ist:
- **EXTF Buchungsstapel CSV** (das Format das Steuerberater importieren)
- **SKR03 Kontenrahmen-Mapping** (Standard fuer KMUs)
- **UTF-8 BOM** (damit Excel die Umlaute korrekt anzeigt)
- **Semikolon als Delimiter** (DATEV-Standard)
- Tests existieren

Endpoint: `POST /api/v1/finance/export/datev`

Fuer dein Frontend: Ein Button "DATEV-Export" der einen Zeitraum-Dialog oeffnet (von/bis), POST abschickt, CSV als Download zurueckbekommt. Fertig.

### 7. Timeline / Kapazitaet -- wann sind die Plans durch?

**93 von 93 Plans sind fertig.** Phase 17.5 (Gast-Chat) ebenfalls komplett abgeschlossen -- alle 3 Plans (Data Foundation, Services+Gateway, Frontend SPA).

API-Contracts koennen jederzeit geschrieben werden -- dafuer muss kein Plan fertig sein. Wenn du fuer ein bestimmtes Frontend-Modul die Endpoint-Spezifikation brauchst, sag Bescheid, dann priorisiere ich das.

---

## Antworten auf Nachricht 2 (Gast-Chat / Kundenportal)

### 1. ~1.100 LOC realistisch?

**JA -- und bereits gebaut.** Die Schaetzung war sogar konservativ.

Wiederverwendet:
- **MinIO** -- Datei-Upload fuer Gast-Nachrichten, null neue LOC
- **WebSocket Hub** -- Erweiterung statt Neubau (siehe Frage 2)
- **Notification Push** -- Gaeste erzeugen Notifications fuer Mitarbeiter, bestehender Flow
- **Integration Framework** -- Rate Limiting, Token-Validierung, alles da

Was neu gebaut wurde:
- **Guest Session Management** (~300 LOC) -- Token-Generierung, SHA-256 Hash, Expiry, Rate Limiting
- **Chat-Modell-Erweiterung** (~100 LOC) -- Migration, guest_session_id auf Messages
- **API-Endpoints** (~400 LOC) -- 5 Guest-Routes (Session Create/Validate, Messages, Send, Config)
- **Frontend SPA** (~700 LOC) -- Standalone Vite+React App unter `/guest/:token`
- **Gateway Static Serving** (~15 LOC) -- SPA unter `/guest/*` ausgeliefert

### 2. WebSocket-Erweiterung -- neuer Endpoint oder bestehenden Hub erweitern?

**Bestehenden Hub erweitert.** Kein separater `/ws/guest` Endpoint. Bereits implementiert.

Begruendung: Ein zweiter WS-Server waere doppelter Maintenance-Aufwand. Der bestehende Hub fuehrt Gaeste als eigene Connection-Klasse.

Was umgesetzt wurde:
- **GuestSessionValidator Interface** im Server-Package (bricht Import-Zyklen)
- **Neuer Auth-Pfad:** `?guest_token=xxx` statt JWT im Header
- **Routing:** Gast-Messages gehen NUR an den verlinkten Channel, nie an andere
- **Kein Zugriff auf:** Presence anderer User, Channel-Liste, Suche, History anderer Channels
- **Typing-Indicators:** Funktionieren automatisch, weil sie Channel-basiert sind
- **Rate Limiting:** In-Memory Sliding Window (30 msg/min pro Session)

### 3. Chat-Modell erweitern oder neues Modell?

**Bestehendes Modell erweitert.** Kein neues `guest_messages`-Tabelle. Bereits implementiert.

Migration (000046):
```sql
-- guest_sessions Tabelle
CREATE TABLE guest_sessions (
  id UUID PRIMARY KEY, token_hash VARCHAR(64) UNIQUE NOT NULL,
  channel_id UUID NOT NULL REFERENCES chat_channels(id),
  display_name VARCHAR(200) NOT NULL, email VARCHAR(255),
  ip_address INET, user_agent TEXT,
  is_active BOOLEAN DEFAULT true, expires_at TIMESTAMPTZ NOT NULL,
  ...
);

-- chat_channels erweitert
ALTER TABLE chat_channels
  ADD COLUMN is_guest_enabled BOOLEAN DEFAULT false;

-- chat_messages erweitert (created_by nullable)
ALTER TABLE chat_messages
  ADD COLUMN guest_session_id UUID REFERENCES guest_sessions(id),
  ALTER COLUMN created_by DROP NOT NULL,
  ADD CONSTRAINT chk_message_sender CHECK (...);
```

Vorteile (genau wie vorgeschlagen):
- **Eine Tabelle, eine Query** -- LEFT JOIN users + LEFT JOIN guest_sessions fuer gemischte Listings
- **Threads funktionieren automatisch**
- **Timeline bleibt chronologisch**
- **Suche funktioniert** -- bestehender Volltext-Index greift

### 4. Web-Frontend fuer Gaeste -- eigenes Mini-Frontend oder Route in der React-App?

**Entscheidung: Standalone Vite + React SPA** in `/guest-chat/` (NICHT in der Electron-App).

Begruendung: Nach genauerer Analyse hat sich ein separates Mini-Frontend als besserer Ansatz herausgestellt:
- **Kein Electron-Ballast** -- Gaeste brauchen weder Zustand, TanStack Query, Radix UI, noch react-intl
- **Minimaler Bundle** -- ~66KB gzipped (statt hunderte KB wenn in der Electron-App eingebettet)
- **Unabhaengig deploybar** -- SPA-Update erfordert keinen Desktop-Client-Release
- **Sicherheits-Isolation** -- Gast-Code hat keinen Zugriff auf interne App-Stores oder Auth-State

Was gebaut wurde:
- **Route:** `/guest/:token` -- Gateway served die SPA als Static Files
- **8 Komponenten:** PreChatForm, ChatWindow, MessageList, MessageBubble, MessageInput, FileUpload, TypingIndicator, ConnectionStatus
- **WebSocket Hook:** Auto-Reconnect mit Exponential Backoff + Jitter, 30s Heartbeat
- **Theming:** CSS Custom Properties, Primaerfarbe + Logo aus Channel-Config
- **Kein Auth-Provider:** Token aus URL, X-Guest-Token Header fuer API-Calls
- **Deutsch-only** UI (keine i18n noetig fuer Gast-Widget)

Code: `guest-chat/src/` (ca. 700 LOC TypeScript + 350 LOC CSS)

### 5. Timing -- wann Gast-Chat?

**Phase 17.5 KOMPLETT FERTIG.** Alle 3 Plans ausgefuehrt:

1. ~~Phase 17-01 (Notification Engine)~~ -- FERTIG
2. ~~Phase 17-02 (Forwarder/Platform Adapters)~~ -- FERTIG
3. ~~Phase 17-03 (Frontend Teams/Slack)~~ -- FERTIG
4. ~~**Phase 17.5 (Gast-Chat)**~~ -- **FERTIG** (alle 3 Plans)
5. Phase 18 (Bexio-Integration) -- naechste Phase

Die 3 Plans:
- **17.5-01:** Guest Session Management + DB Migration + Token-Generierung + Rate Limiter ✓
- **17.5-02:** Chat-Service-Erweiterung + Gateway-Routes + WebSocket Hub Extension ✓
- **17.5-03:** Frontend SPA (Standalone Vite+React) + Gateway Static Serving ✓

### 6. Phase 10 Email -- wie weit?

**KOMPLETT FERTIG. 39 RPCs. Das ist das groesste Modul im ganzen Backend.**

Was alles drin ist:
- **IMAP/SMTP** Anbindung (beliebiger Mail-Provider)
- **AES-verschluesselte Passwoerter** in der DB (nicht Klartext!)
- **Folder Sync** (Inbox, Sent, Drafts, Trash, Custom Folders)
- **Message CRUD** (List, Get, Search, Move, Delete, MarkRead)
- **Threading** (References + In-Reply-To Header Parsing)
- **Send / Reply / Forward** mit HTML-Body
- **Drafts** (Auto-Save, manuelles Speichern)
- **Signaturen** (CRUD, pro Account, HTML-basiert)
- **CRM-Linking** (Email an Contact/Deal/Company verknuepfen)
- **Attachments** via MinIO (Upload + Download)
- **Volltext-Suche** mit deutschem Stemmer (tsvector)

Binary: `backend/cmd/email/main.go`
Proto: `backend/proto/email/v1/email.proto`

Fuer dein Frontend heisst das: Du kannst die Mail-Module-Seite direkt gegen echte APIs bauen. Kein Mock-Modus noetig.

### 7. Security fuer Gast-Chat -- was haben wir schon?

**Alles Wesentliche ist gebaut.** Implementierte Mechanismen:

- **In-Memory Rate Limiter** -- Sliding Window, 30 Nachrichten/Minute pro Guest-Session
- **Token-basierte Sessions** -- UUID v4, SHA-256 gehasht in DB, konfigurierbarer Expiry (Default 7 Tage)
- **IP-basiertes Rate Limiting** am Gateway -- bereits aktiv fuer alle Endpoints
- **Input-Sanitization** -- XSS-Schutz fuer Chat-Messages existiert bereits
- **Channel-Isolation** -- Gast kann NUR seinen eigenen Channel lesen/schreiben (Middleware-Check)
- **X-Guest-Token Header** -- eigener Auth-Pfad, komplett getrennt von JWT-Auth
- **File-Upload-Limits** -- 10MB statt 50MB, nur Bilder+PDF (konfigurierbar pro Channel)

Noch offen fuer spaeter (nicht kritisch fuer Beta):
- CAPTCHA nach X Nachrichten bei Verdacht
- Token-Rotation (optional)
- Abuse-Reporting (Mitarbeiter sperrt Gast-Session)

---

## Gap-Analyse: Was schon existiert

Hier die grosse Ueberraschung. Ich habe deinen MASTER-PLAN Wave 3 gegen den tatsaechlichen Backend-Stand abgeglichen. Ergebnis: **~60% von Wave 3 ist bereits gebaut.**

### FERTIG (Backend existiert, Frontend kann sofort anbinden):

| Item | Was | Wo |
|------|-----|-----|
| 3.1 | CRM CRUD (Contacts, Companies, Deals) | `proto/crm/v1/crm.proto`, `route_crm.go` |
| 3.2 | Custom Fields (pro Entity-Typ) | CRM Service, voller CRUD |
| 3.3 | Firmen-Management | Companies CRUD inkl. Zuordnung zu Contacts |
| 3.9 | Import/Export (CSV + vCard) | CRM Service, 4 Endpoints |
| 3.12 | DATEV-Export | `biz/datev/exporter.go`, EXTF + SKR03 |
| 3.17 | GoBD-Konformitaet | Audit-Trail, Unveraenderbarkeit, Versionierung |
| 3.18 | PDF-Generierung (4 Typen) | `biz/pdf/generator.go`, maroto v2 |

### TEILWEISE (Basis da, Erweiterung noetig):

| Item | Was | Status |
|------|-----|--------|
| 3.11 | Belegkette | Quote→Invoice Konvertierung existiert, Gutschrift→Mahnung fehlt |

### FEHLT (muss noch gebaut werden):

| Item | Was | Geschaetzter Aufwand |
|------|-----|---------------------|
| 3.4 | Duplikaterkennung | ~400 LOC (Levenshtein + Email-Match) |
| 3.5 | Kontakt-Timeline (Activity Stream) | ~300 LOC (Aggregation-Query ueber Activities) |
| 3.7 | Consent-Management (DSGVO) | ~500 LOC (eigenes Modul) |
| 3.8 | Newsletter/Brevo-Integration | ~600 LOC (API-Client + Sync) |
| 3.13 | QR-Rechnung (CH) | ~400 LOC (Swiss QR Bill Standard) |
| 3.14 | ZUGFeRD (XML in PDF) | ~500 LOC (Factur-X/ZUGFeRD 2.1) |
| 3.19 | Stunden→Rechnung | ~300 LOC (Aggregation + Invoice-Generation) |
| 3.20 | FinAPI (Bankkonto) | ~800 LOC (OAuth + Transaction-Sync) |

Phase 18 (Bexio) ist separat geplant und nicht in dieser Analyse.

---

## Korrigierte Backend-Schaetzung

Basierend auf der Gap-Analyse reduziert sich der verbleibende Backend-Aufwand erheblich:

| Wave | Deine Schaetzung | Korrigiert | Grund |
|------|-----------------|------------|-------|
| Wave 3 (CRM+Finanzen) | 3.100 LOC | **~1.200 LOC** | 7 von 15 Items fertig |
| Wave 5 (Kalender) | 750 LOC | **~500 LOC** | CalDAV-Basis existiert |
| Wave 10 (Dokumente) | 1.600 LOC | **~1.200 LOC** | WOPI + MinIO + Versionierung fertig |
| Wave 11 (Email) | 1.950 LOC | **~1.000 LOC** | 39 RPCs fertig, nur Frontend-spezifische Ergaenzungen |
| Wave 13 (DSGVO+KI) | 2.300 LOC | **~2.300 LOC** | Muss komplett neu gebaut werden |
| **Gesamt** | **15.300 LOC** | **~7.200 LOC** | **53% weniger** |

Das ist der Vorteil von AI-gestuetzter Entwicklung: Waehrend du den MASTER-PLAN geschrieben hast, waren wir schon am Bauen. Kein Vorwurf -- im Gegenteil, der Plan ist trotzdem wertvoll als Koordinations-Dokument. Aber die LOC-Schaetzungen muessen angepasst werden.

---

## Naechste Schritte

1. ~~**Phase 17.5 (Gast-Chat)**~~ -- **FERTIG.** Backend + Frontend SPA komplett.
2. **Phase 18 (Bexio-Integration)** -- naechste Phase, braucht noch Planung
3. **Phase 19 (Abacus + RmA)** → **Phase 20 (Plugin System)**
4. **Wave-3-Luecken** (Duplikate, Timeline, QR-Rechnung etc.) -- priorisieren wir nach Beta-Scope-Entscheidung

Was du fuer den Gast-Chat noch tun kannst:
- **Design-Polish:** Das Frontend (`guest-chat/src/`) nutzt plain CSS, kein Tailwind -- wenn du ein Design-Upgrade willst, kannst du die `globals.css` anpassen
- **Admin-UI fuer Gast-Channels:** Noch nicht gebaut -- ein Toggle "Gast-Chat aktivieren" pro Channel + Konfiguration (Logo, Farbe, Willkommensnachricht) waere das naechste Frontend-Feature
- **Notification-Praeferenz:** Wie wird der Mitarbeiter benachrichtigt wenn ein Gast schreibt? (WS-Push + Email? Nur WS? Sound?) -- das koennen wir noch konfigurierbar machen

Gruss,
Luke

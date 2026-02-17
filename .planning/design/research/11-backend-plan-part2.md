# Backend-Implementierungsplan Part 2: API-Endpoints, Integrationen, Sprint-Plan

**Datum:** 2026-02-17
**Referenzen:** `00-SYNTHESE.md`, `10-integrations-guide.md`, `13-vision-ergaenzungen.md`, `BACKEND-REQUIREMENTS-AUDIT.md`
**Zielgruppe:** Luke (Backend-Dev)
**Kontext:** Ergaenzt Luke's Phase 8-20 Roadmap um NEUE Features aus der Marktrecherche

---

## 5. API-Endpoints (Neue, gruppiert nach Feature)

Format: `[METHOD] /api/v1/path -- Beschreibung -- Auth Role -- Rate Limit`

### 5.1 Unified Inbox (Conversations + Messages + Channels)

Zentrale Omnichannel-Inbox fuer E-Mail, Teams, WhatsApp, Widget.

```
# Conversations
GET    /api/v1/inbox/conversations              -- Liste aller Konversationen (filter: channel, status, contact_id)  -- member  -- 60/min
GET    /api/v1/inbox/conversations/{id}          -- Konversation mit Messages                                       -- member  -- 60/min
POST   /api/v1/inbox/conversations               -- Neue Konversation starten (outbound)                            -- member  -- 30/min
PATCH  /api/v1/inbox/conversations/{id}          -- Status aendern (open/snoozed/closed), Assignee zuweisen         -- member  -- 30/min
DELETE /api/v1/inbox/conversations/{id}          -- Archivieren (soft delete)                                       -- manager -- 20/min

# Messages
GET    /api/v1/inbox/conversations/{id}/messages -- Messages einer Konversation (pagination)                        -- member  -- 60/min
POST   /api/v1/inbox/conversations/{id}/messages -- Nachricht senden (outbound via Channel Adapter)                 -- member  -- 30/min
POST   /api/v1/inbox/conversations/{id}/notes    -- Interne Notiz (nicht sichtbar fuer Kunden)                      -- member  -- 30/min

# Channels (Admin-Konfiguration)
GET    /api/v1/inbox/channels                    -- Konfigurierte Kanaele auflisten                                 -- admin   -- 30/min
POST   /api/v1/inbox/channels                    -- Kanal verbinden (email/teams/whatsapp/widget)                   -- admin   -- 10/min
PATCH  /api/v1/inbox/channels/{id}               -- Kanal-Config aktualisieren                                      -- admin   -- 10/min
DELETE /api/v1/inbox/channels/{id}               -- Kanal trennen                                                   -- admin   -- 10/min
POST   /api/v1/inbox/channels/{id}/test          -- Verbindung testen                                               -- admin   -- 5/min

# Webhooks (eingehend von externen Diensten)
POST   /api/v1/webhooks/whatsapp                 -- WhatsApp Cloud API Webhook                                      -- public (HMAC)  -- 120/min
POST   /api/v1/webhooks/teams                    -- Microsoft Teams Bot Webhook                                     -- public (token) -- 120/min
WS     /api/v1/inbox/widget/{token}              -- Website-Widget WebSocket                                        -- JWT token      -- N/A
```

### 5.2 Belegkette (Angebot -> Auftrag -> Lieferschein -> Rechnung)

```
# Dokument-Kette
GET    /api/v1/documents/chain                   -- Alle Belegketten auflisten (filter: contact, status)            -- member  -- 60/min
GET    /api/v1/documents/chain/{id}              -- Belegkette mit allen verknuepften Dokumenten                    -- member  -- 60/min
POST   /api/v1/documents/chain                   -- Neue Kette starten (beginnt als Angebot)                       -- member  -- 20/min

# Konvertierungen
POST   /api/v1/finance/quotes/{id}/convert       -- Angebot -> Auftragsbestaetigung                                -- member  -- 20/min
POST   /api/v1/finance/orders/{id}/convert        -- Auftrag -> Lieferschein                                       -- member  -- 20/min
POST   /api/v1/finance/orders/{id}/invoice        -- Auftrag -> Rechnung (mit Positionsauswahl)                    -- member  -- 20/min
POST   /api/v1/finance/delivery-notes/{id}/invoice -- Lieferschein -> Rechnung                                     -- member  -- 20/min

# Auftraege (NEU -- zwischen Angebot und Rechnung)
GET    /api/v1/finance/orders                    -- Auftraege auflisten                                             -- member  -- 60/min
GET    /api/v1/finance/orders/{id}               -- Auftrag Detail                                                  -- member  -- 60/min
POST   /api/v1/finance/orders                    -- Auftrag erstellen                                               -- member  -- 20/min
PATCH  /api/v1/finance/orders/{id}               -- Auftrag bearbeiten                                              -- member  -- 20/min

# Lieferscheine (NEU)
GET    /api/v1/finance/delivery-notes            -- Lieferscheine auflisten                                         -- member  -- 60/min
POST   /api/v1/finance/delivery-notes            -- Lieferschein erstellen                                          -- member  -- 20/min
GET    /api/v1/finance/delivery-notes/{id}/pdf   -- Lieferschein als PDF                                            -- member  -- 30/min
```

### 5.3 Custom Fields

EAV-Modell mit JSONB fuer Flexibilitaet. Gilt fuer: contacts, companies, deals, tickets, projects.

```
# Field-Definitionen (Admin)
GET    /api/v1/custom-fields                     -- Alle Definitionen (filter: entity_type)                         -- admin   -- 30/min
POST   /api/v1/custom-fields                     -- Feld erstellen (name, type, entity_type, options, required)     -- admin   -- 20/min
PATCH  /api/v1/custom-fields/{id}                -- Feld bearbeiten (label, optionen, required)                     -- admin   -- 20/min
DELETE /api/v1/custom-fields/{id}                -- Feld loeschen (soft delete, Werte bleiben)                      -- admin   -- 10/min
PATCH  /api/v1/custom-fields/reorder             -- Reihenfolge aendern                                             -- admin   -- 10/min

# Field-Werte (werden via Entity-Endpoints gelesen/geschrieben)
# Eingebettet in bestehende Entity-Responses als `custom_fields: {}`
# Kein separater Endpoint noetig -- PATCH /contacts/{id} akzeptiert custom_fields
```

**Field-Types:** `text`, `number`, `date`, `select`, `multiselect`, `checkbox`, `url`, `email`, `phone`, `currency`, `rating`

### 5.4 Company Entity (Firma als eigene Entity)

```
# Firmen (erweitert bestehende CRM-Endpoints)
GET    /api/v1/companies                         -- Firmen auflisten (search, filter, pagination)                   -- member  -- 60/min
GET    /api/v1/companies/{id}                    -- Firma Detail (inkl. Kontakte, Deals, Aktivitaeten)              -- member  -- 60/min
POST   /api/v1/companies                         -- Firma erstellen                                                 -- member  -- 20/min
PATCH  /api/v1/companies/{id}                    -- Firma bearbeiten                                                -- member  -- 30/min
DELETE /api/v1/companies/{id}                    -- Firma loeschen (soft delete)                                    -- manager -- 10/min

# Beziehungen
POST   /api/v1/companies/{id}/contacts           -- Kontakt zu Firma zuordnen (role: Geschaeftsfuehrer, etc.)      -- member  -- 30/min
DELETE /api/v1/companies/{id}/contacts/{cid}     -- Zuordnung entfernen                                             -- member  -- 30/min
GET    /api/v1/companies/{id}/contacts           -- Kontakte einer Firma                                            -- member  -- 60/min
GET    /api/v1/companies/{id}/deals              -- Deals einer Firma                                               -- member  -- 60/min

# Duplikaterkennung
POST   /api/v1/companies/duplicates/check        -- Pruefen ob Duplikat existiert (name, domain, uid)              -- member  -- 20/min
POST   /api/v1/companies/duplicates/merge        -- Zwei Firmen zusammenfuehren                                    -- manager -- 5/min
```

### 5.5 Canned Responses (Helpdesk Textbausteine)

```
GET    /api/v1/helpdesk/canned-responses         -- Alle Textbausteine (filter: category, search)                  -- member  -- 60/min
POST   /api/v1/helpdesk/canned-responses         -- Neuen Textbaustein erstellen                                   -- member  -- 20/min
PATCH  /api/v1/helpdesk/canned-responses/{id}    -- Textbaustein bearbeiten                                        -- member  -- 20/min
DELETE /api/v1/helpdesk/canned-responses/{id}    -- Textbaustein loeschen                                          -- member  -- 20/min

# Private Notizen (Helpdesk intern)
POST   /api/v1/helpdesk/tickets/{id}/notes       -- Interne Notiz hinzufuegen                                      -- member  -- 30/min
GET    /api/v1/helpdesk/tickets/{id}/notes       -- Interne Notizen eines Tickets                                  -- member  -- 60/min
```

### 5.6 Shared Links (Externer Datei-Link-Share)

```
POST   /api/v1/files/{id}/share                  -- Shared Link erstellen (expiry, password, download_limit)       -- member  -- 20/min
GET    /api/v1/files/shares                      -- Eigene Shared Links auflisten                                  -- member  -- 30/min
DELETE /api/v1/files/shares/{token}              -- Shared Link widerrufen                                         -- member  -- 20/min
PATCH  /api/v1/files/shares/{token}              -- Link aktualisieren (Ablaufdatum, Passwort)                     -- member  -- 20/min

# Oeffentlicher Zugang (kein Auth noetig)
GET    /api/v1/public/files/{token}              -- Datei-Info anzeigen (Name, Groesse, Vorschau)                  -- public  -- 60/min
GET    /api/v1/public/files/{token}/download     -- Datei herunterladen (Passwort-Check wenn gesetzt)              -- public  -- 30/min
```

### 5.7 Consent Management (DSGVO-Einwilligungen)

```
GET    /api/v1/contacts/{id}/consents            -- Einwilligungen eines Kontakts                                  -- member  -- 60/min
POST   /api/v1/contacts/{id}/consents            -- Einwilligung erfassen (purpose, source, timestamp)             -- member  -- 20/min
DELETE /api/v1/contacts/{id}/consents/{cid}      -- Einwilligung widerrufen (mit Timestamp)                        -- member  -- 20/min

# Consent-Zwecke (Admin)
GET    /api/v1/consent-purposes                  -- Alle Zwecke (newsletter, marketing, profiling, etc.)           -- admin   -- 30/min
POST   /api/v1/consent-purposes                  -- Neuen Zweck definieren                                         -- admin   -- 10/min
```

### 5.8 DATEV Export

```
POST   /api/v1/export/datev/bookings             -- Buchungsstapel exportieren (zeitraum, kontenrahmen)            -- manager -- 5/min
POST   /api/v1/export/datev/timetracking         -- Zeiterfassung fuer Lohnbuchhaltung exportieren                 -- manager -- 5/min
GET    /api/v1/export/datev/preview               -- Vorschau (erste 20 Zeilen)                                    -- manager -- 10/min
GET    /api/v1/export/datev/accounts              -- Konten-Zuordnungstabelle                                      -- manager -- 30/min
PATCH  /api/v1/export/datev/accounts              -- Konten-Zuordnung speichern                                    -- admin   -- 10/min
```

### 5.9 Integration Connections (generisch)

```
# Verbindungs-Management
GET    /api/v1/integrations                      -- Alle verfuegbaren Integrationen + Status                       -- admin   -- 30/min
GET    /api/v1/integrations/{provider}            -- Status einer Integration (connected, last_sync, errors)        -- admin   -- 30/min
POST   /api/v1/integrations/{provider}/connect    -- OAuth2-Flow starten oder API-Key speichern                    -- admin   -- 10/min
DELETE /api/v1/integrations/{provider}/disconnect -- Verbindung trennen                                            -- admin   -- 5/min
POST   /api/v1/integrations/{provider}/sync       -- Manuellen Sync ausloesen                                     -- admin   -- 5/min
GET    /api/v1/integrations/{provider}/logs       -- Sync-Logs (Fehler, Konflikte)                                 -- admin   -- 30/min

# OAuth2 Callbacks
GET    /api/v1/integrations/bexio/callback        -- Bexio OAuth2 Redirect                                        -- session -- N/A
GET    /api/v1/integrations/zoom/callback          -- Zoom OAuth2 Redirect                                        -- session -- N/A
GET    /api/v1/integrations/teams/callback         -- Teams OAuth2 Redirect                                       -- session -- N/A

# Bexio-spezifisch
POST   /api/v1/integrations/bexio/sync/contacts   -- Kontakt-Sync manuell                                        -- admin   -- 5/min
POST   /api/v1/integrations/bexio/sync/invoices   -- Rechnungs-Sync manuell                                      -- admin   -- 5/min
GET    /api/v1/integrations/bexio/conflicts        -- Offene Konflikte anzeigen                                   -- admin   -- 30/min
POST   /api/v1/integrations/bexio/conflicts/{id}   -- Konflikt loesen (keep_local / keep_remote)                  -- admin   -- 20/min

# Skribble
POST   /api/v1/integrations/skribble/sign          -- Signatur-Request erstellen                                  -- member  -- 10/min
GET    /api/v1/integrations/skribble/requests       -- Offene Signatur-Requests                                   -- member  -- 30/min
POST   /api/v1/webhooks/skribble                    -- Skribble Status-Webhook                                    -- public (HMAC) -- 60/min

# Newsletter (Brevo/CleverReach)
POST   /api/v1/integrations/newsletter/sync         -- Kontakte zum Newsletter-Provider syncen                    -- admin   -- 5/min
POST   /api/v1/integrations/newsletter/campaigns    -- Kampagne erstellen                                         -- manager -- 10/min
GET    /api/v1/integrations/newsletter/stats         -- Oeffnungsraten, Klicks, Bounces                           -- member  -- 30/min
```

---

## 6. Integrations-Implementierung (Backend-Seite)

### 6.1 Unified Inbox Channel Adapters

**Interface-Definition:**

```go
// internal/inbox/adapter.go
type ChannelAdapter interface {
    // Typ des Kanals (email, teams, whatsapp, widget)
    Type() string

    // Eingehende Nachricht verarbeiten (Webhook/Polling)
    HandleIncoming(ctx context.Context, raw json.RawMessage) (*IncomingMessage, error)

    // Ausgehende Nachricht senden
    SendMessage(ctx context.Context, conv *Conversation, msg *OutboundMessage) error

    // Verbindung testen
    TestConnection(ctx context.Context, config json.RawMessage) error

    // Verbindung herstellen
    Connect(ctx context.Context, config json.RawMessage) error

    // Verbindung trennen
    Disconnect(ctx context.Context) error
}

type IncomingMessage struct {
    ExternalID  string
    ChannelType string
    SenderName  string
    SenderEmail string
    Content     string
    ContentType string // text, html, image, file
    Attachments []Attachment
    Metadata    json.RawMessage
    ReceivedAt  time.Time
}
```

**IMAP Adapter** (bereits von Luke in Phase 10 geplant):
- `emersion/go-imap` v2 fuer IMAP-Zugriff
- IDLE-Support fuer Push-Benachrichtigungen
- Thread-Erkennung via `In-Reply-To` / `References` Header
- Automatische CRM-Kontakt-Zuordnung ueber E-Mail-Adresse

**Teams Adapter** (Microsoft Graph API):
- OAuth2 Authorization Code Flow (`golang.org/x/oauth2`)
- Bot Framework: Subscription auf `/teams/{id}/channels/{id}/messages`
- Change Notifications (Webhook) fuer eingehende Nachrichten
- Outbound via `POST /v1.0/teams/{id}/channels/{id}/messages`
- Token-Refresh in Background-Goroutine

**WhatsApp Adapter** (Meta Cloud API):
- Webhook-Verifikation (Challenge-Response)
- HMAC-SHA256 Signatur-Validierung (`X-Hub-Signature-256`)
- Message-Templates fuer erste Nachricht (24h-Fenster-Regel beachten!)
- Medien-Download via `GET /{media-id}` + Auth-Header
- Outbound: `POST /v18.0/{phone-id}/messages`

**Widget Adapter** (WebSocket):
- JWT-Token pro Besucher (anonym oder identifiziert)
- WebSocket-Upgrade auf `/api/v1/inbox/widget/{token}`
- Rate-Limiting: max 10 Messages/Minute pro Widget-Session
- Embeddable JS-Snippet generiert Token via REST-Call

**Message Queue (Outbound):**
- Redis-basierte Queue fuer ausgehende Nachrichten
- Retry mit exponentiellem Backoff (3 Versuche)
- Dead-Letter-Queue fuer fehlgeschlagene Zustellungen
- Worker-Pool: 1 Goroutine pro Kanal-Typ

### 6.2 OnlyOffice (Document Server API)

Wir nutzen die **Document Server API** (nicht WOPI) -- einfacher, besser dokumentiert.

**Go-Endpoints:**

```go
// GET /api/v1/onlyoffice/config/{fileID}
// Generiert Editor-Config mit JWT-signiertem Token
func (h *OnlyOfficeHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
    fileID := chi.URLParam(r, "fileID")
    file, _ := h.fileService.GetByID(r.Context(), fileID)

    config := onlyoffice.EditorConfig{
        Document: onlyoffice.Document{
            FileType: filepath.Ext(file.Name)[1:],
            Key:      fmt.Sprintf("%s-v%d", file.ID, file.Version),
            Title:    file.Name,
            URL:      fmt.Sprintf("%s/api/v1/files/%s/download", h.baseURL, file.ID),
        },
        EditorConfig: onlyoffice.Editor{
            CallbackURL: fmt.Sprintf("%s/api/v1/onlyoffice/callback", h.baseURL),
            User:        onlyoffice.User{ID: userID, Name: userName},
            Lang:        "de",
        },
    }

    token := h.signJWT(config) // HMAC-SHA256 mit ONLYOFFICE_JWT_SECRET
    config.Token = token
    respondJSON(w, http.StatusOK, config)
}

// POST /api/v1/onlyoffice/callback
// OnlyOffice ruft dies auf wenn Dokument gespeichert wird
func (h *OnlyOfficeHandler) Callback(w http.ResponseWriter, r *http.Request) {
    // Status 2 = Dokument fertig bearbeitet
    // Status 6 = Dokument wird bearbeitet (Force-Save)
    // Neue Version von callback.url herunterladen und in S3 speichern
}
```

**File-Locking:**
- Optimistic Locking via `document.key` (Key aendert sich bei jeder Version)
- Co-Editing: OnlyOffice verwaltet Locks intern
- Callback-Status 1 = Dokument wird bearbeitet (Lock aktiv)
- Callback-Status 4 = Dokument geschlossen ohne Aenderung (Lock freigeben)

**JWT-Validierung:**
- `golang-jwt/jwt/v5` fuer Token-Erstellung
- Shared Secret zwischen Go-Backend und OnlyOffice Container (`ONLYOFFICE_JWT_SECRET`)

### 6.3 DATEV Export

```go
// internal/export/datev/exporter.go
func (e *Exporter) ExportBookings(bookings []Booking, period Period, w io.Writer) error {
    encoder := charmap.Windows1252.NewEncoder()     // NICHT UTF-8!
    tw := transform.NewWriter(w, encoder)
    cw := csv.NewWriter(tw)
    cw.Comma = ';'        // Semikolon-Trennung
    cw.UseCRLF = true     // CR+LF Pflicht

    cw.Write(e.formatHeader(period))  // Zeile 1: EXTF;700;21;"Buchungsstapel";12;...
    cw.Write(fieldNames)               // Zeile 2: Feldnamen (121 Felder, ~14 relevant)
    for _, b := range bookings {
        cw.Write(e.bookingToRecord(b)) // Umsatz(Komma!);S/H;EUR;Konto;Gegenkonto;TTMM;Beleg;Text
    }
    cw.Flush()
    return cw.Error()
}
// formatDecimalDE: 1234.56 -> "1234,56" (deutsches Dezimalformat)
```

**Kernpunkte:**
- Windows-1252 Encoding (`golang.org/x/text/encoding/charmap`)
- Semikolon-Trennung, Komma als Dezimalzeichen, Datumsformat TTMM (4-stellig)
- ~200-300 LOC fuer kompletten Exporter

### 6.4 Bexio API

**Kernaspekte:**
- OAuth2 Authorization Code Flow (`golang.org/x/oauth2`)
- Auth-URL: `https://idp.bexio.com/authorize`, Token-URL: `https://idp.bexio.com/token`
- Scopes: `openid contact_edit kb_invoice_edit kb_offer_edit`
- Polling alle 5 Min (Bexio hat KEINE Webhooks), `modified_since` fuer Delta-Sync
- Konfliktstrategie: Last-Write-Wins mit manueller Konfliktanzeige
- `sync_metadata` Tabelle: `bexio_id`, `kmuhub_id`, `entity_type`, `last_synced_at`, `etag`
- Token-Refresh via Background-Goroutine, bei Fehler -> Integration als `disconnected` markieren

### 6.5 Skribble

**Flow:**
1. `POST /v2/signature-requests` mit Base64-PDF, Signer-Email, Quality (SES/AES/QES)
2. Skribble sendet E-Mail an Unterzeichner
3. Webhook an KMU Hub: `signature_request.signed` -> signiertes PDF herunterladen
4. Status-Tracking in DB: `pending` -> `opened` -> `signed` -> `completed` / `declined`
5. HMAC-Validierung aller eingehenden Webhooks

### 6.6 Brevo / CleverReach

**Newsletter-Provider Interface:**

```go
// internal/integrations/newsletter/provider.go
type NewsletterProvider interface {
    // Kontakt anlegen/aktualisieren
    UpsertContact(ctx context.Context, contact Contact) error
    // Kontakt loeschen
    DeleteContact(ctx context.Context, email string) error
    // Double-Opt-In starten
    StartDoubleOptIn(ctx context.Context, email string, listID string) error
    // Kampagne erstellen
    CreateCampaign(ctx context.Context, campaign Campaign) (string, error)
    // Kampagne senden
    SendCampaign(ctx context.Context, campaignID string) error
    // Statistiken abrufen
    GetCampaignStats(ctx context.Context, campaignID string) (*Stats, error)
    // Transactional E-Mail senden (nur Brevo)
    SendTransactional(ctx context.Context, msg TransactionalEmail) error
}
```

- **Brevo:** REST API direkt mit `net/http` (kein SDK -- offizielles Go-SDK veraltet)
- **CleverReach:** OAuth2 + REST API v3
- Kunde waehlt in Settings welchen Provider er nutzt
- Kontakt-Sync: CRM-Kontakte mit Newsletter-Flag -> Provider
- DOI-Flow: Brevo/CleverReach uebernimmt Double-Opt-In-Mail

### 6.7 FinAPI

**Flow:**
1. WebForm-URL via `POST https://webform.finapi.io/api/webforms` -> Frontend oeffnet Iframe
2. Nutzer gibt Banking-Credentials direkt bei FinAPI ein (NIE bei KMU Hub!)
3. Transaktionen abrufen: `GET /api/v2/transactions?accountIds=...&minDate=...` (Pagination, max 500/Seite)
4. Auto-Matching: Exakter Betrag + Referenz im Verwendungszweck -> offene Rechnung zuordnen
5. OAuth2 Client Credentials Flow (Backend-zu-Backend), Transaktionen in eigener Tabelle zwischenspeichern

### 6.8 Swiss QR-Code + ZUGFeRD

**QR-Rechnung:**

```go
// internal/finance/qr/swiss.go
type SwissQRData struct {
    IBAN         string  // CH-IBAN oder QR-IBAN
    CreditorName string
    CreditorAddr Address
    Amount       *float64 // nil = ohne Betrag
    Currency     string   // CHF oder EUR
    DebtorName   string
    DebtorAddr   Address
    RefType      string   // QRR, SCOR, NON
    Reference    string
    Message      string   // max 140 Zeichen
}

func (q *SwissQRData) GenerateQRImage() ([]byte, error) {
    payload := q.buildPayload() // Zeilenweiser Aufbau, \r\n getrennt
    qr, _ := qrcode.New(payload, qrcode.Medium) // Error-Correction M fuer Schweizer Kreuz
    png := qr.Image(256)

    // Schweizer Kreuz in der Mitte ueberlagern (7mm x 7mm)
    return overlaySwissCross(png)
}
```

- `skip2/go-qrcode` fuer QR-Generierung
- Schweizer Kreuz als Overlay via `image/draw`
- Zahlteil-Layout: exakte Masse (210x105mm) als PDF-Element
- Validierung: Modulo-10-rekursiv fuer QR-Referenz-Pruefsumme

**ZUGFeRD:**

```go
// internal/finance/einvoice/zugferd.go
func GenerateZUGFeRD(invoice *Invoice) ([]byte, error) {
    // 1. Invoice -> GOBL Envelope
    env, _ := gobl.Envelop(toGOBLInvoice(invoice))

    // 2. GOBL -> ZUGFeRD XML (EN 16931 Profil)
    xml, _ := zugferd.Generate(env, zugferd.ProfileEN16931)

    // 3. PDF generieren (wkhtmltopdf oder unipdf)
    pdf := generateInvoicePDF(invoice)

    // 4. XML als Attachment in PDF/A-3 einbetten
    return embedXMLInPDFA3(pdf, xml, "factur-x.xml")
}
```

- `invopop/gobl` + `invopop/gobl.zugferd` fuer XML-Generierung
- PDF/A-3: `unidoc/unipdf` (kommerziell, ~500 EUR/Jahr) oder wkhtmltopdf + PDF/A-Konvertierung
- EN 16931 (Comfort) als Default-Profil

### 6.9 LiveKit (Video)

Bereits in Luke's Phase 8 geplant. Ergaenzungen:

- Go SDK: `github.com/livekit/server-sdk-go` fuer Room-Management + Token-Generierung
- `auth.NewAccessToken(apiKey, apiSecret)` mit `VideoGrant{RoomJoin: true, Room: name}`
- Rooms mit `EmptyTimeout: 300` (5 Min nach letztem Teilnehmer loeschen)

**Recording mit DSGVO-Consent:**
- Vor Recording-Start: alle Teilnehmer muessen zustimmen (Consent-Dialog im Frontend)
- Consent in `meeting_consents` Tabelle protokollieren (participant_id, meeting_id, timestamp)
- Recording via LiveKit Egress API, Speicherung in S3/MinIO (AES-256 verschluesselt)

### 6.10 Zoom Fallback

Fuer KMUs ohne eigenen LiveKit-Server (Starter-Tier):

- OAuth2 Authorization Code Flow, `POST /users/me/meetings` fuer Meeting-Erstellung
- `join_url` in Meeting-Objekt speichern, Frontend oeffnet Zoom-Link

**VideoProvider Interface:**

```go
type VideoProvider interface {
    CreateMeeting(ctx context.Context, req MeetingRequest) (*Meeting, error)
    JoinURL(meetingID string) string
    CancelMeeting(ctx context.Context, meetingID string) error
}
// Implementierungen: LiveKitProvider, ZoomProvider
// Auswahl via Tenant-Config: tenant.video_provider = "livekit" | "zoom"
```

---

## 7. Sprint-Plan (Backend, 2-Wochen-Sprints)

### Legende

| Symbol | Bedeutung |
|--------|-----------|
| **NEU** | Nicht in Lukes Phase 8-20 enthalten, kommt aus Marktrecherche |
| **ERWEITERUNG** | Luke hat es geplant, braucht aber Ergaenzungen |
| **PLANNED** | Bereits in Lukes Roadmap, hier nur referenziert |

### Sprint 1-2: Foundation Gaps (Wochen 1-4)

> **Fokus:** Schnelle Wins die sofort Business Impact haben.

| Task | Typ | Aufwand | Abhaengigkeit |
|------|-----|---------|---------------|
| MWSt multi-country (DE 19%/7%, AT 20%/10%/13%, CH 8.1%/2.6%/3.8%) | **NEU** | 2-3 Tage | Finance-Modul |
| Akadem. Titel + Anrede-Logik (`salutation`, `title`, `preferred_language`, `formal`) | **NEU** | 2-3 Tage | Contact-Model |
| Canned Responses + Private Notizen (Helpdesk) | **NEU** | 3-5 Tage | Helpdesk-Modul |
| Firma als eigene Entity (company_contacts Join-Tabelle, Migration) | **NEU** | 5-7 Tage | CRM DB-Schema |
| Consent-Management (consents Tabelle, CRUD) | **NEU** | 3-4 Tage | Contact-Model |

### Sprint 3-4: DATEV + QR-Rechnung + PDF (Wochen 5-8)

> **Fokus:** KRITISCH fuer DE + CH Markt. Ohne das kein Verkauf.

| Task | Typ | Aufwand | Abhaengigkeit |
|------|-----|---------|---------------|
| DATEV Buchungsstapel-Exporter (Go, Windows-1252, CSV) | **NEU** | 5-7 Tage | Finance-Modul |
| Swiss QR-Code Generierung (Payload + QR-Image + Schweizer Kreuz) | **NEU** | 5-7 Tage | — |
| PDF-Generierung fuer Rechnungen/Angebote (wkhtmltopdf oder unipdf) | **NEU** | 5-7 Tage | Finance-Modul |
| QR-Rechnung in PDF-Layout einbetten (Zahlteil 210x105mm) | **NEU** | 3-4 Tage | QR-Code + PDF |

### Sprint 5-6: Custom Fields + Belegkette (Wochen 9-12)

> **Fokus:** Custom Fields = CRM-Pflicht. Belegkette = Handwerker-Pflicht.

| Task | Typ | Aufwand | Abhaengigkeit |
|------|-----|---------|---------------|
| Custom Fields: Schema (JSONB), Definitionen-CRUD, Validierung | **NEU** | 7-10 Tage | DB-Migration |
| Custom Fields: Integration in Contacts, Companies, Deals, Tickets | **NEU** | 5-7 Tage | Custom Fields Schema |
| Belegkette: Auftraege + Lieferscheine (neue Entities) | **NEU** | 5-7 Tage | Finance-Modul |
| Belegkette: Konvertierungs-Endpoints (Quote->Order->Invoice) | **NEU** | 3-5 Tage | Auftraege |

### Sprint 7-8: Phase 8 -- Video/Meetings (Wochen 13-16)

> **Fokus:** Luke's Phase 8. Ergaenzt um Zoom-Fallback.

| Task | Typ | Aufwand | Abhaengigkeit |
|------|-----|---------|---------------|
| LiveKit Server Setup + Go-SDK Integration | **PLANNED** | 5-7 Tage | Docker |
| Token-Generierung, Room-Management, Presence | **PLANNED** | 5-7 Tage | LiveKit |
| Zoom OAuth2 + Meeting-Creation Fallback | **NEU** | 3-5 Tage | — |
| VideoProvider Interface (LiveKit / Zoom) | **NEU** | 2-3 Tage | — |
| Recording mit DSGVO-Consent-Flow | **ERWEITERUNG** | 3-4 Tage | LiveKit |

### Sprint 9-10: Phase 9 -- Security & Compliance (Wochen 17-20)

> **Fokus:** Luke's Phase 9. Ergaenzt um DSGVO-Tools.

| Task | Typ | Aufwand | Abhaengigkeit |
|------|-----|---------|---------------|
| 2FA (TOTP), Session-Management, Audit-Log | **PLANNED** | 7-10 Tage | Auth-Service |
| DSGVO-Auskunft (Art. 15): Globale Suche + JSON/CSV-Export | **ERWEITERUNG** | 5-7 Tage | Audit-Log |
| DSGVO-Loeschung (Art. 17): Kaskadierte Anonymisierung | **ERWEITERUNG** | 5-7 Tage | Alle Module |
| i18n (DE/FR/IT/EN) | **PLANNED** | 5-7 Tage | — |
| GoBD-konforme Rechnungen (unveraenderbar, lueckenlose Nummern) | **NEU** | 3-5 Tage | Finance |

### Sprint 11-13: Phase 10 -- E-Mail + Unified Inbox (Wochen 21-26)

> **Fokus:** IMAP/SMTP (Luke) + Unified Inbox Architektur (NEU).

| Task | Typ | Aufwand | Abhaengigkeit |
|------|-----|---------|---------------|
| IMAP-Client (`emersion/go-imap` v2), Folder-Sync, IDLE-Push | **PLANNED** | 10-14 Tage | — |
| SMTP-Versand, Threading, Attachments | **PLANNED** | 5-7 Tage | IMAP |
| CRM-Auto-Linking (E-Mail <-> Kontakt via Adresse) | **PLANNED** | 3-5 Tage | CRM + E-Mail |
| Unified Inbox: ChannelAdapter Interface + Message-Model | **NEU** | 5-7 Tage | E-Mail |
| Unified Inbox: E-Mail als erster Adapter | **NEU** | 3-5 Tage | ChannelAdapter |
| Unified Inbox: Conversation-Management (Status, Assignment) | **NEU** | 3-5 Tage | Message-Model |

### Sprint 14-15: Phase 11 -- Documents + OnlyOffice (Wochen 27-30)

> **Fokus:** File-Management (Luke) + OnlyOffice-Integration (NEU).

| Task | Typ | Aufwand | Abhaengigkeit |
|------|-----|---------|---------------|
| File-Browser, Upload (S3/MinIO), Versionierung, Tags | **PLANNED** | 10-14 Tage | S3-Setup |
| Full-Text Search, Sharing, Permissions | **PLANNED** | 5-7 Tage | Files |
| Shared Links (signierte URLs, Passwort, Ablaufdatum) | **NEU** | 3-5 Tage | Files |
| OnlyOffice: Callback-Endpoint + JWT-Signing + Editor-Config | **NEU** | 5-7 Tage | Files + Docker |
| OnlyOffice: Co-Editing, Versionierung bei Speichern | **NEU** | 3-5 Tage | OnlyOffice |

### Sprint 16-17: Phase 12 -- Finance Erweiterung (Wochen 31-34)

> **Fokus:** Lukes Finance-Phase + ZUGFeRD + Duplikaterkennung.

| Task | Typ | Aufwand | Abhaengigkeit |
|------|-----|---------|---------------|
| GoBD-konforme Quotes/Invoices, Tax Calculation | **PLANNED** | 7-10 Tage | Finance |
| ZUGFeRD XML-Generierung (`invopop/gobl`) | **NEU** | 5-7 Tage | Finance |
| PDF/A-3 mit eingebettetem XML | **NEU** | 3-5 Tage | ZUGFeRD + PDF |
| XRechnung-Support (UBL-Syntax) | **NEU** | 3-4 Tage | ZUGFeRD |
| Duplikaterkennung CRM (Name + E-Mail + Domain Matching) | **NEU** | 5-7 Tage | Companies |
| Stunden-zu-Rechnung Workflow | **NEU** | 3-5 Tage | Zeiterfassung + Finance |

### Sprint 18-19: Integrationen Batch 1 (Wochen 35-38)

> **Fokus:** Bexio (CH-Pflicht) + Brevo (Newsletter).

| Task | Typ | Aufwand | Abhaengigkeit |
|------|-----|---------|---------------|
| Integration Connection Framework (generisch: connect/disconnect/sync) | **NEU** | 5-7 Tage | — |
| Bexio OAuth2 + Token-Management | **ERWEITERUNG** | 3-5 Tage | Framework |
| Bexio Kontakt-Sync (bidirektional, Polling, Delta) | **ERWEITERUNG** | 5-7 Tage | Bexio OAuth2 |
| Bexio Rechnungs-Export + Zahlungs-Import | **ERWEITERUNG** | 5-7 Tage | Bexio Sync |
| Brevo Newsletter-Integration (Kontakt-Sync, DOI, Transactional) | **NEU** | 5-7 Tage | Newsletter Interface |

### Sprint 20-21: Integrationen Batch 2 (Wochen 39-42)

> **Fokus:** Skribble + Teams-Adapter + WhatsApp-Adapter.

| Task | Typ | Aufwand | Abhaengigkeit |
|------|-----|---------|---------------|
| Skribble API: Signatur-Request + Webhook + PDF-Download | **NEU** | 5-7 Tage | Files |
| Teams Channel Adapter (MS Graph API, Bot Framework) | **NEU** | 7-10 Tage | Unified Inbox |
| WhatsApp Channel Adapter (Meta Cloud API, Webhooks) | **NEU** | 5-7 Tage | Unified Inbox |
| Website Chat-Widget (WebSocket Adapter + JWT) | **NEU** | 5-7 Tage | Unified Inbox |

### Sprint 22-23: Integrationen Batch 3 (Wochen 43-46)

> **Fokus:** FinAPI + Nextcloud + CleverReach.

| Task | Typ | Aufwand | Abhaengigkeit |
|------|-----|---------|---------------|
| FinAPI Partner-Onboarding + WebForm-Einbettung | **NEU** | 5-7 Tage | — |
| FinAPI Transaktions-Abruf + Auto-Matching mit Rechnungen | **NEU** | 7-10 Tage | FinAPI + Finance |
| Nextcloud WebDAV (Datei-Browse + Upload/Download) | **ERWEITERUNG** | 5-7 Tage | Files |
| CleverReach als alternativer Newsletter-Provider | **NEU** | 3-5 Tage | Newsletter Interface |

### Sprint 24-25: Phase 19-20 -- Automation + Plugins (Wochen 47-50)

> **Fokus:** Luke's letzte Phasen. Ergaenzt um Widget-System.

| Task | Typ | Aufwand | Abhaengigkeit |
|------|-----|---------|---------------|
| Automation Engine (Trigger -> Condition -> Action) | **PLANNED** | 10-14 Tage | Alle Module |
| Plugin System (Config-basiert + WASM Runtime) | **PLANNED** | 10-14 Tage | — |
| Gaeste-Zugang fuer Projekte (eingeschraenkte Auth) | **NEU** | 5-7 Tage | Auth + Work |

---

### Zusammenfassung: NEU vs. PLANNED

| Kategorie | Sprints | Aufwand (Wochen) |
|-----------|---------|------------------|
| **NEU** (aus Marktrecherche) | ~60% der Arbeit | ~28-32 Wochen |
| **PLANNED** (Luke's Roadmap) | ~30% der Arbeit | ~16-20 Wochen |
| **ERWEITERUNG** (Geplant + Ergaenzung) | ~10% der Arbeit | ~6-8 Wochen |
| **GESAMT** | 25 Sprints | ~50 Wochen |

### Kritischer Pfad

```
Sprint 1-2:  Foundation Gaps (MWSt, Titel, Firma, Consents)
    |
Sprint 3-4:  DATEV + QR + PDF  ← Ohne das kein DE/CH-Verkauf
    |
Sprint 5-6:  Custom Fields + Belegkette  ← Ohne das kein echtes CRM
    |
Sprint 11-13: E-Mail + Unified Inbox  ← Ohne das ist die App halb leer
    |
Sprint 16-17: Finance + ZUGFeRD  ← Pflicht ab 2027
    |
Sprint 18-19: Bexio + Newsletter  ← CH-Markt + Marketing
```

**Empfehlung:** Sprints 1-6 (12 Wochen) als naechstes nach Phase 8 einplanen. Das schliesst die kritischsten Luecken fuer einen Beta-Launch bei Dienstleistern und Handwerkern.

---

*Hinweis: Aufwandsschaetzungen basieren auf 1 Backend-Entwickler (Luke). Bei AI-gestuetzter Entwicklung koennen die Zeiten um ~30-40% reduziert werden. Alle API-Pfade sind Vorschlaege und koennen an Lukes bestehende Konventionen angepasst werden.*

# Backend-Plan Wave 5: Chat + Helpdesk

> **Von:** Darien (Frontend) | **Fuer:** Luke (Backend)
> **Datum:** 2026-02-21 | **Status:** Frontend DONE (Commit `47ad258`)
> **Geschaetzter Backend-Aufwand:** ~750 LOC Go

---

## Uebersicht

Wave 5 hat 14 Items, davon sind nur 4 Backend-abhaengig. Der Rest ist FE-ONLY
(Emoji-Reactions, @Mentions, Canned Responses, SLA-Badges, etc. — alles mit Mock-Daten).

**Was das Frontend aktuell macht:**
- Chat: Emoji-Reactions, File-Drop, @Mentions, Thread-Replies — alles im Zustand Store
- Helpdesk: Canned Responses, Private Notes, Business Hours, SLA, Categories, Routing Rules, CSAT — alles Mock

---

## A. Chat Backend-Anforderungen

### A1. File Sharing / Upload (Item 5.4) — PRIO MITTEL

Das Frontend hat Drag-and-Drop File Sharing in Chat gebaut. Dateien werden als
MessageBubble mit Thumbnail/Icon dargestellt. Backend braucht Upload-Endpoint.

**Endpoint:**
```
POST /api/v1/files/upload
Content-Type: multipart/form-data
```

**Request Fields:**
- `file`: Die Datei (Binary)
- `context`: `chat` | `helpdesk` | `document` | `avatar`
- `context_id`: Channel-ID oder Ticket-ID (optional)
- `thumbnail`: Auto-generieren fuer Bilder (true/false)

**Response:**
```json
{
  "id": "uuid",
  "filename": "screenshot.png",
  "mime_type": "image/png",
  "size_bytes": 245760,
  "url": "/api/v1/files/uuid/download",
  "thumbnail_url": "/api/v1/files/uuid/thumbnail",
  "uploaded_at": "2026-02-21T10:00:00Z",
  "uploaded_by": "user-uuid"
}
```

**Anforderungen:**
- Max-Dateigroesse: 50 MB (konfigurierbar)
- Thumbnail-Generierung fuer Bilder (max 200x200px)
- Virus-Scan (optional, z.B. ClamAV)
- Storage: MinIO/S3-kompatibel (bereits fuer Chat-Anhaenge vorhanden?)
- Access-Control: Nur Channel-Mitglieder duerfen Dateien sehen

**Download:**
```
GET /api/v1/files/:id/download       — Original-Datei
GET /api/v1/files/:id/thumbnail      — Thumbnail (nur Bilder)
```

---

## B. Helpdesk Backend-Anforderungen

### B1. Ticket-Routing / Auto-Zuweisung (Item 5.11) — PRIO NIEDRIG

Das Frontend hat eine Routing-Konfiguration gebaut (Regel-Editor: Kategorie → Agent/Team).
Backend muss Regeln speichern und bei Ticket-Erstellung automatisch anwenden.

**DB-Schema:**
```sql
CREATE TABLE ticket_routing_rules (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     UUID NOT NULL,
  rule_name     VARCHAR(200) NOT NULL,
  priority      INTEGER DEFAULT 0,      -- hoehere Zahl = hoehere Prio
  conditions    JSONB NOT NULL,          -- {"category": "billing", "priority": "high"}
  action        JSONB NOT NULL,          -- {"assign_to": "user-uuid"} oder {"assign_to_team": "team-uuid"}
  is_active     BOOLEAN DEFAULT TRUE,
  created_at    TIMESTAMPTZ DEFAULT NOW()
);
```

**Endpoints:**
```
GET    /api/v1/helpdesk/routing-rules         — Alle Regeln
POST   /api/v1/helpdesk/routing-rules         — Regel erstellen
PUT    /api/v1/helpdesk/routing-rules/:id     — Regel aendern
DELETE /api/v1/helpdesk/routing-rules/:id     — Regel loeschen
```

**Auto-Routing Logik:**
Bei `POST /api/v1/helpdesk/tickets` (Ticket-Erstellung):
1. Alle aktiven Regeln nach Prioritaet sortiert laden
2. Conditions gegen Ticket-Felder matchen
3. Erste passende Regel: Action ausfuehren (Agent zuweisen)
4. Keine Regel passt: Ticket bleibt unassigned

---

### B2. CSAT — Kundenzufriedenheit (Item 5.12) — PRIO NIEDRIG

Nach Ticket-Schliessung bekommt der Kunde einen Bewertungs-Link.
Oeffentlicher Endpoint (kein Login noetig!).

**DB-Schema:**
```sql
CREATE TABLE ticket_ratings (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id  UUID NOT NULL,
  ticket_id  UUID NOT NULL REFERENCES tickets(id),
  rating     SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
  comment    TEXT,
  token      VARCHAR(100) UNIQUE NOT NULL,  -- Token im Bewertungs-Link
  rated_at   TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_ticket_ratings_ticket ON ticket_ratings(ticket_id);
```

**Endpoints:**
```
-- Oeffentlich (kein Auth!):
GET  /api/v1/public/rate/:token        — Bewertungs-Formular Daten laden
POST /api/v1/public/rate/:token        — Bewertung abgeben

-- Intern (Auth):
GET  /api/v1/helpdesk/csat/stats       — Aggregierte CSAT-Statistiken
```

**Oeffentliche Bewertungs-Seite Request:**
```json
{
  "rating": 4,
  "comment": "Schnelle Hilfe, danke!"
}
```

**Stats Response:**
```json
{
  "average_rating": 4.2,
  "total_ratings": 87,
  "distribution": { "1": 3, "2": 5, "3": 12, "4": 35, "5": 32 },
  "period": "last_30_days"
}
```

**Token-Generierung:**
- Bei Ticket-Schliessung: Token generieren (z.B. `crypto/rand` Base64, 32 Bytes)
- Token per E-Mail an Kunden senden (Link: `https://app.kmuhub.de/rate/{token}`)
- Token ist 7 Tage gueltig, einmalig nutzbar

---

### B3. E-Mail-zu-Ticket Konvertierung (Item 5.15) — PRIO MITTEL

Eingehende E-Mails automatisch zu Helpdesk-Tickets konvertieren.

**Was gebraucht wird:**
1. **IMAP-Listener:** Polling oder IDLE auf einer konfigurierten Mailbox
2. **Parser:** E-Mail → Ticket-Felder mappen (Subject → Titel, Body → Beschreibung, From → Kontakt)
3. **Kategorie-Zuordnung:** Absender-Domain oder Betreff-Keywords → Kategorie
4. **Thread-Erkennung:** Reply auf existierendes Ticket erkennt In-Reply-To/References Header

**Konfiguration:**
```json
{
  "email": "support@firma.de",
  "imap_server": "imap.gmail.com",
  "imap_port": 993,
  "username": "support@firma.de",
  "password": "***",  // verschluesselt speichern!
  "poll_interval_seconds": 60,
  "default_category": "general",
  "auto_reply": true,
  "auto_reply_template": "Vielen Dank fuer Ihre Anfrage. Ticket-Nr: {{ticket_number}}"
}
```

**Endpoints:**
```
GET  /api/v1/helpdesk/email-config     — Konfiguration lesen
PUT  /api/v1/helpdesk/email-config     — Konfiguration speichern
POST /api/v1/helpdesk/email-config/test — Verbindung testen
```

**Thread-Erkennung:**
- Custom Header `X-KMUHub-Ticket-ID` in ausgehenden Mails
- Fallback: Subject-Matching `[Ticket #1234]`

---

## C. Zusammenfassung: Empfohlene Reihenfolge

| Prio | Item | Was | Aufwand |
|------|------|-----|---------|
| 1 | A1 | File Upload (generisch, wird ueberall wiederverwendet) | ~200 LOC |
| 2 | B3 | E-Mail-zu-Ticket (IMAP-Listener + Parser) | ~300 LOC |
| 3 | B1 | Ticket-Routing (Regel-Engine) | ~150 LOC |
| 4 | B2 | CSAT (Oeffentlicher Bewertungs-Endpoint) | ~100 LOC |

**Total: ~750 LOC Go**

**Hinweis:** Der File-Upload-Endpoint (A1) ist generisch und wird in Wave 7 (Personalakte),
Wave 9 (Fahrzeug-Fotos, Rapport-Fotos) und Wave 11 (Dokumente) wiederverwendet.
Falls du ihn schon hast fuer Chat-Anhaenge, muss er nur erweitert werden.

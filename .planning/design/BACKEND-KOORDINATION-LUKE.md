# Backend-Koordination: Frontend-Masterplan × Backend

> **Von:** Darien (Design/Frontend) | **Fuer:** Luke (Backend/Go)
> **Datum:** 2026-02-21 | **Branch:** `design/brainstorm`
> **Referenz:** `.planning/design/MASTER-PLAN.md` (20-Wave-Gesamtplan, ~80K LOC)

---

## Detaillierte API-Vertraege

Fuer jede Wave mit Backend-Bedarf gibt es einen detaillierten Plan mit exakten
Endpoints, DB-Schemas, Request/Response Bodies:

| Wave | Datei | Status |
|------|-------|--------|
| Wave 3 | `api-contracts/wave-03-crm-finanzen.md` | Fertig |
| Wave 5 | `api-contracts/wave-05-chat-helpdesk.md` | Fertig |
| Wave 6 | `api-contracts/wave-06-work-zeiterfassung.md` | Fertig |
| Wave 7 | `api-contracts/wave-07-team-schichten.md` | Fertig |
| Wave 8 | `api-contracts/wave-08-api-contract.md` | Fertig |
| Wave 9 | `api-contracts/wave-09-api-contract.md` | Fertig |
| Wave 10 | `api-contracts/wave-10-api-contract.md` | Fertig |
| Wave 11 | `api-contracts/wave-11-api-contract.md` | Fertig |
| Wave 12 | `api-contracts/wave-12-api-contract.md` | Fertig |
| Wave 13 | `api-contracts/wave-13-api-contract.md` | Fertig |

---

## TL;DR

Der Frontend-Masterplan hat **20 Waves** (13 Feature + 7 Design). Davon sind **~80% rein Frontend** (Mock-Daten, Zustand Stores, UI) und **~20% brauchen Backend-Endpoints**. Design-Waves (14-20) sind komplett FE-ONLY. Ich baue alles zuerst mit Mock-Daten — du wirst nie von mir blockiert. Aber damit wir die Mock-Daten spaeter sauber gegen echte API-Calls austauschen koennen, brauchen wir fuer bestimmte Waves frueh einen **API-Vertrag** (Endpoint + Request/Response Shape).

**Gesamtuebersicht Backend-Aufwand (nur Feature-Waves 1-13):**

| Wave | Beschreibung | Frontend LOC | Backend LOC | Dringlichkeit | API-Vertrag |
|------|-------------|-------------|-------------|---------------|-------------|
| 1 | Foundation | 3.890 | 0 | — | — |
| 2 | Kommunikation + Wiki | 4.000 | 0 | — | — |
| 3 | **CRM + Finanzen** | 4.740 | **3.100** | **HOCH** | **Fertig** |
| 4 | E-Mail | 1.200 | 0 | — | — |
| 5 | Chat + Helpdesk | 2.670 | 750 | Mittel | **Fertig** |
| 6 | Projekte + Zeiterfassung | 2.800 | 1.000 | Mittel | **Fertig** |
| 7 | Team/HR + Schichten | 3.840 | 400 | Niedrig | **Fertig** |
| 8 | Einkauf + Inventar + Produktion | 3.880 | 2.500 | Niedrig | **Fertig** |
| 9 | Fuhrpark + Rapporte + Vermietung | 3.580 | 2.200 | Niedrig | **Fertig** |
| 10 | Formulare + Vertraege + Kalender + Meetings | 3.590 | 1.600 | Mittel | **Fertig** |
| 11 | Video + Notifications + Dokumente | 3.530 | 1.950 | Mittel | **Fertig** |
| 12 | Integrations-Settings | 1.730 | 120 | Niedrig | **Fertig** |
| 13 | **DSGVO + KI** | 2.200 | **2.300** | **HOCH** | **Fertig** |
| 14-20 | Design-Ueberarbeitung | 24.600 | 0 | — | — |
| | **Gesamt** | **~66.000** | **~18.100** | | |

---

## So funktioniert die Koordination

### Prinzip: Mock-First, API-Vertrag, Parallele Arbeit

```
Darien (Frontend)                    Luke (Backend)
─────────────────                    ──────────────
1. Baut UI mit Mock-Daten
2. Definiert API-Vertrag ──────────→ 3. Baut Endpoints gegen Vertrag
   (Endpoint, Request/Response)
4. Tauscht Mock gegen               ← Endpoints fertig
   TanStack Query Hooks
```

**Was heisst "API-Vertrag"?** Ein kurzes Markdown pro Wave mit:
- Endpoint (Method + Path)
- Request Body (JSON Shape)
- Response Body (JSON Shape)
- Error Cases

Ablage: `.planning/api-contracts/wave-XX.md`

**Was heisst "Austauschen"?** Deine TanStack Query Hooks + API-Client Architektur existiert bereits (`api/hooks/`, `api/*-client.ts`). Der Umbau pro Modul ist minimal:

```ts
// VORHER (Mock, design/brainstorm):
const contacts = useContactStore(s => s.contacts)
const addContact = useContactStore(s => s.addContact)

// NACHHER (Backend, nach Merge):
const { data: contacts } = useContacts()           // TanStack Query
const addContact = useCreateContact()               // Mutation
```

### Timeline-Vorschlag

```
Woche 1-4:   Darien baut Wave 1-2 (alles FE-ONLY)     ← Wave 1 DONE
Woche 2:     API-Vertrag fuer Wave 3 schreiben (zusammen)
Woche 3-8:   Darien baut Wave 3-4 (Mock)
             Luke baut Wave 3 Backend parallel
Woche 6:     API-Vertrag fuer Wave 5-6 schreiben
Woche 8-12:  Darien baut Wave 5-6 (Mock)
             Luke baut Wave 5-6 Backend parallel
...          (Pattern wiederholt sich)
```

**Kern-Idee:** Ich bin dir immer 1-2 Waves voraus mit der UI. Du baust die Endpoints wenn du Zeit hast. Niemand blockiert niemanden.

---

## Backend-Anforderungen pro Wave (Detail)

---

### Wave 1: Foundation — Kein Backend

Alles FE-ONLY. Stores, Shared Components, TipTap Editor, Global Search, Currency Utility.

**Frontend-Status:** DONE (Commit `2342beb`, 38 Dateien, +4.816 LOC)

---

### Wave 2: Kommunikation + Wiki — Kein Backend

Zwei neue Module mit Mock-Daten. Kommunikation (Unified Inbox fuer E-Mail/WhatsApp/Teams/Widget/Portal) und Wiki (Knowledge Base mit TipTap).

**Frontend-Status:** DONE (Commit `071ca9f`)

**Backend spaeter noetig fuer:**
- Echte Kanal-Anbindung (IMAP, WhatsApp Business API, Microsoft Graph, Widget-Webhook)
- Wiki-Persistenz (Artikel in DB statt Zustand Store)
- Aber: Das ist erst relevant wenn wir live gehen, nicht jetzt

---

### Wave 3: CRM + Finanzen — ~3.100 LOC Backend (GRÖSSTER BROCKEN)

**Frontend-Status:** DONE (Commit `6ee1921`)
**Detaillierter API-Vertrag:** `api-contracts/wave-03-crm-finanzen.md`

Das ist die Wave mit dem meisten Backend-Bedarf. Hier die einzelnen Punkte:

#### CRM

| # | Feature | Was Luke bauen muss | Prio |
|---|---------|-------------------|------|
| 3.1 | CRM CRUD-Formulare | `POST/PUT/DELETE /api/contacts`, `/api/companies`, `/api/deals` — aktuell nur GET. Standard-CRUD mit Validierung. | HOCH |
| 3.2 | Custom Fields | JSONB-Spalte `custom_fields` in `contacts`-Tabelle. CRUD-Endpoints fuer Feld-Definitionen (`/api/custom-field-definitions`). Feld-Typen: text, number, date, dropdown, checkbox, url. | HOCH |
| 3.3 | Firma als eigene Entity | DB-Tabelle `companies` mit Relationen zu `contacts`. Falls im CRM-Service schon vorhanden: nur sicherstellen dass die Endpoints komplett sind. | MITTEL |
| 3.4 | Duplikaterkennung | `POST /api/contacts/check-duplicates` — Fuzzy-Matching auf E-Mail (exakt), Telefon (normalisiert), Name (Levenshtein oder trigram). Response: Liste potentieller Duplikate mit Confidence-Score. | NIEDRIG |
| 3.5 | Kontakt-Timeline | `GET /api/contacts/:id/timeline` — Cross-Modul-Query: E-Mails, Deals, Tickets, Meetings, Notizen zu einem Kontakt. Sortiert nach Datum. Pagination. | MITTEL |
| 3.7 | Consent-Management | `consent_records`-Tabelle (contact_id, purpose, granted, source, timestamp). CRUD-Endpoints. Zwecke: email_marketing, phone, post, profiling. | NIEDRIG |
| 3.8 | Newsletter (Brevo) | API-Connector zu Brevo/CleverReach. OAuth2 oder API-Key. Subscriber-Listen sync. | NIEDRIG |
| 3.9 | Import/Export | `POST /api/contacts/import` (CSV/vCard Bulk-Import), `GET /api/contacts/export?format=csv|vcf`. | MITTEL |

#### Finanzen

| # | Feature | Was Luke bauen muss | Prio |
|---|---------|-------------------|------|
| 3.11 | Belegkette | Relationen zwischen Angebot→Auftrag→Lieferschein→Rechnung→Mahnung. Konvertierungs-Endpoint: `POST /api/invoices/:id/convert?to=rechnung`. | MITTEL |
| 3.12 | DATEV-Export | `GET /api/finance/export/datev?from=...&to=...` — CSV im DATEV-Format (Windows-1252, Semikolon, spezifische Spalten). | HOCH |
| 3.12 | Bexio-Sync | OAuth2-Flow fuer Bexio, Sync-Endpoints, Konflikt-Handling. | NIEDRIG |
| 3.13 | QR-Rechnung | Swiss QR-bill generieren (IBAN, Referenz, Betrag, QR-Code). Library: z.B. `go-qrbill`. Oder als Teil der PDF-Generierung. | MITTEL |
| 3.14 | ZUGFeRD/XRechnung | XML-Attachment an Rechnungs-PDF (Factur-X). Compliance-Level: Basic/Comfort/Extended. | NIEDRIG |
| 3.17 | GoBD | Unveraenderbare Invoice-Records (kein DELETE, nur Storno). Lueckenlose Nummern. Audit-Log-Tabelle. | HOCH |
| 3.18 | PDF-Generierung | `GET /api/invoices/:id/pdf` — Rechnungs-PDF serverseitig rendern. Template-basiert. | HOCH |
| 3.19 | Stunden→Rechnung | Cross-Modul: Zeiteintraege auswaehlen → Rechnungspositionen generieren. `POST /api/invoices/from-timeentries`. | MITTEL |
| 3.20 | FinAPI (Banking) | FinAPI-Connector fuer automatischen Bankabgleich. Transaktions-Matching. | NIEDRIG |

---

### Wave 4: E-Mail — Kein Backend

TipTap in ComposeModal, E-Mail-Vorlagen, Signatur-Editor, Kontakt-Chips. Alles FE-ONLY.

**Frontend-Status:** DONE (Commit `c148305`)

---

### Wave 5: Chat + Helpdesk — ~750 LOC Backend

**Frontend-Status:** DONE (Commit `47ad258`)
**Detaillierter API-Vertrag:** `api-contracts/wave-05-chat-helpdesk.md`

| # | Feature | Was Luke bauen muss | Prio |
|---|---------|-------------------|------|
| 5.4 | File Sharing | `POST /api/files/upload` — Multipart Upload fuer Chat-Dateien. Thumbnail-Generierung fuer Bilder. | MITTEL |
| 5.11 | Ticket-Routing | Regel-Engine: Kategorie → Team/Agent Zuweisung. Config-Endpoint + automatische Anwendung bei Ticket-Erstellung. | NIEDRIG |
| 5.12 | CSAT | Oeffentlicher Endpoint fuer Kundenbewertung nach Ticket-Schliessung. Token-basiert (kein Login noetig). | NIEDRIG |
| 5.15 | E-Mail→Ticket | IMAP-Listener der eingehende Mails zu Helpdesk-Tickets konvertiert. Kategorie-Zuordnung nach Absender/Betreff. | MITTEL |

---

### Wave 6: Projekte + Zeiterfassung — ~1.000 LOC Backend

**Frontend-Status:** IN PROGRESS
**Detaillierter API-Vertrag:** `api-contracts/wave-06-work-zeiterfassung.md`

| # | Feature | Was Luke bauen muss | Prio |
|---|---------|-------------------|------|
| 6.2 | Stunden→Rechnung | Cross-Modul: Zeiteintraege → Rechnungspositionen (gleich wie 3.19). | MITTEL |
| 6.5 | Gaeste-Zugang | Auth-System erweitern: Read-only Token ohne Login. Separater Endpoint der nur Projektfortschritt/Meilensteine liefert. | NIEDRIG |
| 6.7 | DATEV Zeitexport | `GET /api/timeentries/export/datev` — CSV im DATEV-Lohn-Format. | MITTEL |
| 6.9 | Genehmigungs-Banner | Wochenrapport-Approval Workflow serverseitig. Status: pending→approved/rejected. | NIEDRIG |

---

### Wave 7: Team/HR + Schichten — ~400 LOC Backend

| # | Feature | Was Luke bauen muss | Prio |
|---|---------|-------------------|------|
| 7.3 | Digitale Personalakte | File Storage fuer Mitarbeiter-Dokumente (Vertrag, Zeugnis, Zertifikate). Kategorien + Ablauf-Tracking. | MITTEL |
| 7.13 | PDF-Export Schichtplan | Wochenplan als druckbares PDF serverseitig rendern. | NIEDRIG |

---

### Wave 8: Einkauf + Inventar + Produktion — ~950 LOC Backend

| # | Feature | Was Luke bauen muss | Prio |
|---|---------|-------------------|------|
| 8.2 | Belegkette→Buchhaltung | Cross-Modul: Bestellung→Lieferschein→Eingangsrechnung Verknuepfung. | MITTEL |
| 8.4 | Genehmigungsworkflow | Bestellungen ab Betrag X brauchen Approval. | NIEDRIG |
| 8.6 | Inventar-Integration | Wareneingang aus Einkauf erzeugt automatisch Inventar-Bewegung. | MITTEL |
| 8.8 | Barcode-Scanner | Kein Go-Backend — Electron IPC fuer Kamera-Zugriff. Aber: Artikel-Lookup per Barcode braucht `GET /api/inventory/barcode/:code`. | NIEDRIG |
| 8.14 | Materialverfuegbarkeit | Produktion fragt Inventar-Bestaende ab bevor Auftrag startet. | NIEDRIG |

---

### Wave 9: Fuhrpark + Rapporte + Vermietung — ~2.200 LOC Backend

**Frontend-Status:** DONE
**Detaillierter API-Vertrag:** `api-contracts/wave-09-api-contract.md`

| # | Feature | Was Luke bauen muss | Prio |
|---|---------|-------------------|------|
| 9.1 | Fahrtenbuch | LogbookEntry CRUD mit km-Berechnung, Geschaefts-/Privat-Filter. | MITTEL |
| 9.2 | TCO-Dashboard | Kostenaggregate pro Fahrzeug (Maintenance + Fuel + Insurance + Depreciation). | NIEDRIG |
| 9.3 | Fahrzeug-Dokumente | File Storage fuer Fahrzeugschein, Versicherungspolice, TUeV-Berichte. Ablaufdatum-Tracking. | NIEDRIG |
| 9.4 | Schadensmeldung | DamageReport CRUD mit Foto-Upload und Status-Workflow (open/in_progress/resolved). | MITTEL |
| 9.5 | Reifenwechsel-Erinnerung | Config-Endpoint pro Fahrzeug + optionaler Cron-Job. | NIEDRIG |
| 9.6 | Unterschrift (Rapporte) | signatureDataUrl-Feld auf FieldReport (Base64 PNG oder File-Upload). | NIEDRIG |
| 9.7 | Rapport Foto-Upload | Photo-Attachment CRUD auf FieldReport (multipart/form-data). | MITTEL |
| 9.8 | Aufmass-Skizze | sketchDataUrl-Feld auf Measurement (Base64 PNG oder File-Upload). | NIEDRIG |
| 9.9 | PDF Tagesbericht | Druckfaehigen Tagesbericht als PDF generieren (Wetter, Arbeitszeit, Material, Fotos, Unterschrift). | MITTEL |
| 9.10 | Rapport-Genehmigung | Submit/Approve/Reject Workflow mit Status-Uebergaengen. | MITTEL |
| 9.11 | Wetter-Daten | Optionaler externer API-Call (OpenWeatherMap) mit Cache. | NIEDRIG |
| 9.12 | Preisberechnung (Vermietung) | Tages-/Wochen-Logik fuer Mietpreis. | NIEDRIG |
| 9.13 | Kautionsverwaltung | depositStatus-Feld auf Reservation (none/collected/returned). | NIEDRIG |
| 9.14 | Zustandsprotokoll | CRUD mit JSONB-Checkliste, Foto-Upload, Unterschrift. | MITTEL |
| 9.15 | Multi-Waehrung | currency-Feld auf RentalObject + Reservation (EUR/CHF/USD). | NIEDRIG |

---

### Wave 10: Formulare + Vertraege + Kalender + Meetings — ~1.600 LOC Backend

| # | Feature | Was Luke bauen muss | Prio |
|---|---------|-------------------|------|
| 10.3 | Formular-Aktionen | Server-Actions bei Formular-Einreichung (E-Mail senden, Task erstellen, CRM-Kontakt anlegen). | MITTEL |
| 10.4 | Oeffentliche Formulare | Auth-Ausnahme: Formulare ohne Login ausfuellbar. Oeffentlicher Endpoint mit Token. | MITTEL |
| 10.6 | E-Signatur (Skribble) | Skribble API-Connector. Signer-Management, Signing-Order, Status-Webhooks. | NIEDRIG |
| 10.8 | Vertrags-Erinnerungen | Cron-Job: X Tage vor Kuendigungsfrist Notification pushen. | NIEDRIG |
| 10.12 | Video-Meeting Button | LiveKit Token generieren: `POST /api/meetings/:id/token`. | HOCH |
| 10.13 | Push-Erinnerungen | Reminder X Minuten vor Event via WebSocket/Push. | MITTEL |
| 10.14 | Externer Buchungslink | Oeffentlicher Endpoint: Verfuegbarkeit anzeigen + Termin buchen ohne Login. | NIEDRIG |
| 10.19 | Einladungs-E-Mails | .ics Kalender-Attachment generieren + per E-Mail versenden bei Meeting-Erstellung. | MITTEL |

---

### Wave 11: Video + Notifications + Dokumente — ~1.950 LOC Backend

| # | Feature | Was Luke bauen muss | Prio |
|---|---------|-------------------|------|
| 11.1 | Video Meeting Room | LiveKit Token (gleich wie 10.12). Room-Management. | HOCH |
| 11.2 | Notification Center | WebSocket-Channel fuer Push-Notifications an Client. Notification-Persistenz in DB. | HOCH |
| 11.9 | DATEV-Auswertungen | Steuerberater-konforme Reports im DATEV-Format. | NIEDRIG |
| 11.10b | Collabora Viewer | **WOPI-Protokoll:** 3 Endpoints (`CheckFileInfo`, `GetFile`, `PutFile`). Collabora Docker-Container Setup. Token-basierte Auth pro User+Datei. | MITTEL |
| 11.12 | Share Links | Signierte URLs mit Ablaufdatum + optionalem Passwort fuer Dokumente. | NIEDRIG |
| 11.13 | Datei-Versionierung | Jede Datei-Aenderung = neue Version. Versions-Liste Endpoint. Wiederherstellen-Endpoint. | MITTEL |
| 11.15 | Nextcloud | WebDAV-Proxy zu Nextcloud. Dateien listen/lesen/schreiben ueber KMU Hub. | NIEDRIG |

---

### Wave 12: Integrations-Settings — ~120 LOC Backend

| # | Feature | Was Luke bauen muss | Prio |
|---|---------|-------------------|------|
| 12.4 | Bexio OAuth2 | OAuth2-Flow fuer Bexio (Token-Exchange, Refresh). | NIEDRIG |

Alle anderen Integration-Panels sind FE-ONLY Mocks mit Config-Formularen.

---

### Wave 13: DSGVO + KI — ~2.300 LOC Backend (ZWEITGRÖSSTER BROCKEN)

#### DSGVO

| # | Feature | Was Luke bauen muss | Prio |
|---|---------|-------------------|------|
| 13.1 | Consent-Management | Consent-Records persistieren (erweitert Wave 3.7). | MITTEL |
| 13.2 | DSGVO-Auskunft (Art. 15) | Cross-Modul-Suche: Alle Daten einer Person aus CRM, Helpdesk, Mails, Projekte sammeln. JSON/CSV-Export. | HOCH |
| 13.3 | DSGVO-Loeschung (Art. 17) | Kaskadierte Anonymisierung ueber alle Module. GoBD-Ausnahmen beachten (Rechnungen 10 Jahre behalten, nur Kontaktdaten anonymisieren). | HOCH |
| 13.4 | Datenexport (Art. 20) | Strukturiertes ZIP-Paket aller Daten einer Person generieren. | MITTEL |

#### KI

| # | Feature | Was Luke bauen muss | Prio |
|---|---------|-------------------|------|
| 13.6-8 | KI E-Mail/Meeting/Ticket | OpenAI-Proxy: API-Key serverseitig, nicht im Client. `POST /api/ai/generate` mit Context. | MITTEL |
| 13.9 | Semantische Suche | Embedding-Service (z.B. pgvector). Dokumente/Wiki/Tickets vektorisieren. Query-Endpoint. | NIEDRIG |
| 13.10 | Auto-Klassifizierung | KI-Service der Dokumente als oeffentlich/intern/vertraulich klassifiziert. | NIEDRIG |

---

### Waves 14-20: UI-Design-Ueberarbeitung — Kein Backend

7 Design-Waves (Farbpaletten, Navigation-Layouts, Component-Redesign, Animationen).
Komplett FE-ONLY, kein Backend-Bedarf.

---

## Empfohlene Reihenfolge fuer Luke

### Phase A: Jetzt anfangen (parallel zu meinen Waves 1-4)

1. **CRM CRUD** (3.1) — POST/PUT/DELETE fuer Kontakte, Firmen, Deals
2. **GoBD** (3.17) — Unveraenderbare Records, Storno, Audit-Log
3. **PDF-Generierung** (3.18) — Rechnungs-PDF
4. **DATEV-Export** (3.12) — CSV im DATEV-Format

→ Das sind die Grundlagen fuer "erwachsene" Finanzen.

### Phase B: Wenn Wave 3 Frontend fertig ist

5. **Custom Fields** (3.2) — JSONB-Spalte + Definitions-CRUD
6. **QR-Rechnung** (3.13) — Swiss QR-bill
7. **Kontakt-Timeline** (3.5) — Cross-Modul-Query
8. **LiveKit Token** (10.12/11.1) — fuer Video-Meetings

### Phase C: Wenn Waves 5-6 Frontend fertig sind

9. **File Upload** (5.4) — generischer Upload-Endpoint (wird ueberall wiederverwendet)
10. **E-Mail→Ticket** (5.15) — IMAP-Listener
11. **WebSocket Notifications** (11.2) — Push an Client
12. **Stunden→Rechnung** (3.19/6.2) — Cross-Modul

### Phase D: Vor dem Launch

13. **DSGVO-Tools** (13.2-4) — Pflicht vor Go-Live
14. **OpenAI-Proxy** (13.6-8) — KI-Features
15. **Collabora/WOPI** (11.10b) — Enterprise Tier
16. **Externe Integrationen** (Bexio, Skribble, Brevo, FinAPI)

---

## Was schon existiert (kein Neubau noetig)

Luke hat bereits gebaut und gemerged (Phases 6+7):

- TanStack Query Hooks Architektur (`api/hooks/`)
- API-Client Layer (`api/*-client.ts`, `api/*-types.ts`)
- Presence Store + WebSocket Provider
- Auth-System (JWT, Login/Logout)
- CalDAV-Integration
- Basic CRM-Service (GET Endpoints)

Diese Architektur wird 1:1 weiterverwendet. Neue Endpoints folgen dem gleichen Pattern.

---

## Offene Fragen an Luke

1. **CRM-Service:** Wie weit sind die CRUD-Endpoints fuer Kontakte/Firmen/Deals? Nur GET oder schon POST/PUT/DELETE?
2. **File Storage:** Welches System ist geplant? Lokales Filesystem? S3-kompatibel (MinIO)? Brauchen wir das fuer mehrere Waves.
3. **PDF-Generierung:** Welche Library? `wkhtmltopdf`? `chromedp`? `go-pdf`? Brauchen wir fuer Rechnungen, Schichtplaene, Rapporte.
4. **LiveKit:** Ist der LiveKit-Server schon aufgesetzt? Brauchen wir fuer Video-Meetings (Wave 10+11).
5. **WebSocket:** Laeuft schon fuer Presence. Koennen wir den gleichen Channel fuer Notifications erweitern?
6. **DATEV:** Hast du Erfahrung mit dem DATEV-Format? Oder brauchen wir einen Steuerberater der das validiert?
7. **Timeline:** Wann hast du kapazitaet fuer die Backend-Arbeit? Damit ich die API-Vertraege rechtzeitig schreibe.

---

## Zusammenfassung

- **~75% Frontend** kann ich komplett allein bauen (Mock-Daten)
- **~25% Backend** braucht Luke, aber **nie blockierend** — alles Mock-First
- **Groesste Brocken:** Wave 3 (CRM+Finanzen, ~3.1K) und Wave 13 (DSGVO+KI, ~2.3K)
- **Koordination:** API-Vertrag schreiben → parallel arbeiten → Mock gegen Hooks tauschen
- **Reihenfolge:** CRM CRUD + GoBD + PDF + DATEV zuerst, Rest nach Kapazitaet

# Backend-Gaps für Luke

> Was im Backend fehlt oder nicht „ready zum Verknüpfen" ist, damit das Frontend zu Feature-Parität andocken kann.
> Claude sammelt, Darien reicht an Luke weiter. Stand: Welle 1 (2026-06-01).
> Priorität: 🔴 ZFA-Pilot-kritisch · 🟠 wichtig · ⚪ später/Post-Launch.

## 🔴 ZFA-Pilot-kritisch

### kalender — Terminbuchungs-Link (Online-Terminbuchung)
ZFA-Akquise hängt an Online-Terminbuchung. FE-Flow existiert komplett als Mock, BE fehlt ganz.
- `GET/POST/PUT/DELETE /api/v1/calendar/booking-pages` — Buchungsseiten (Slug, Services, Verfügbarkeitsregeln)
- `GET /api/v1/public/book/:slug` — **öffentlich/unauthenticated** (Kunde bucht ohne Login)
- `GET .../availability?date=&service=` — freie Slots aus Kalender-Belegung berechnen
- `POST /api/v1/public/bookings` — öffentliche Terminanlage → erzeugt Event + Bestätigung

### dialer — DSGVO-Consent-Absicherung (⚠ Risiko, nicht nur Feature)
`consentAsserter` ist im Standard-`NewService`-Konstruktor `nil` — nur `NewServiceWithConsent` verdrahtet den Einwilligungs-Check. Prüfen, ob der Standard-Konstruktor irgendwo aktiv ist → sonst Anrufe ohne Consent möglich. Für Finanzberatung heikel.

## 🟠 Wichtig (Kern-Module, ZFA-relevant)

### kontakte
- XLSX/Excel-Import-Endpoint (CSV/vCard existieren, XLSX fehlt)

#### Kontakte/360° — fehlende Hooks/Endpoints für Kunden-360°-Ansicht
Folgende Verknüpfungen konnten im ContactDetailPanel NICHT gebaut werden, weil Hooks + Endpoints fehlen:
- **Verträge am Kontakt**: kein `GET /api/v1/contracts?contact_id={id}` und kein Frontend-Hook `useContactContracts(contactId)`. Vertragsservice hat nur generisches CRUD; die Filterung nach `contact_id` fehlt im Modell und der Route.
- **Rechnungen am Kontakt**: kein `GET /api/v1/finance/invoices?contact_id={id}` und kein Frontend-Hook `useContactInvoices(contactId)`. finance_line_items hat keinen direkten Kontakt-FK; Normalisierung (Sprint 4) Voraussetzung.
- Sobald Luke diese Endpoints + contact_id-Felder ergänzt, kann das FE die beiden Sektionen in ContactDetailPanel nachziehen (Muster: analog useDeals mit contact_id-Filter).

### crm
- Lead-Scoring: Score-Feld im Contact-Modell + Berechnungsservice (Regelwerk) + Endpoint
- Umsatz-Forecasting: dedizierter `/api/v1/reports/forecast`-Endpoint (Zeitreihe auf Deal-Wahrscheinlichkeit)
- E-Mail-Marketing/Kampagnen: kompletter Service fehlt (`/api/v1/campaigns`)

### vertraege
- `UploadDocument` vervollständigen (aktuell Stub, TODO „Sprint 3 — MinIO-Integration")
- Audit-Log: `contract_events`-Tabelle (action/user/timestamp/payload) + `GET /contracts/{id}/events`
- Digitale Signatur-Workflow (Phase D, Skribble/DocuSign): `POST /contracts/{id}/send-for-signing`, `/sign`, Status-Endpoint + Webhook-Receiver

### kommunikation (chat + inbox, werden zusammengeführt)
- Reaction-Endpoint: `POST /api/v1/messages/{id}/reactions` (Toggle)
- Chat-Datei-Upload-Route: `POST /api/v1/channels/{id}/files` (Multipart) — Service `Upload()` existiert, Route fehlt
- **Externe Kanal-Verknüpfungen verwaltbar machen** (für Modul-Merge): Settings/CRUD um nicht-interne Kanäle (Mail/WhatsApp/Widget) anzubinden — Routing-Rules-Infra im inbox-Service ist Basis

### dokumente
- Datei-Kommentare: Comment-Tabelle + Endpoints
- Externe Share-Links: Token-Store + `GET/POST /api/v1/documents/share-links` (Ablauf, Passwort-Hash) + öffentliches Resolve-Endpoint

### mails
- Multi-Account: Tabelle + `ListEmailAccounts` (aktuell 1 Account/User)
- Vorlagen/Quicktext: Template-CRUD (`email/template/`)
- Regeln & Filter: `email/rule/` + Endpoints

### helpdesk
- `contact_id`/`org_id` ins Ticket-Modell (CRM-Verknüpfung)
- `source_channel` ins Ticket-Modell (Multi-Channel) + Inbox→Ticket-Adapter-Endpoint
- Knowledge-Base-Endpoint (FE-Tab existiert)
- `time_spent`-Feld (Ticket-Zeiterfassung)

### berichte
- Query-Builder: BE-Executor liest `query_config` schon — Editor-Contract festzurren
- `ExecuteKindCross`-Methode im Executor (datenquellen-übergreifend)
- Breakout/Pivot-Schema in `RunReportRequest.Params`

### team
- Onboarding-Workflow-API (Template + Checklist)
- DATEV-**HR-Lohn**-Endpoint (bestehender `route_datev_upload.go` ist nur Buchungsdaten)
- **Lohnvorbereitung / Lohnlauf (P-team, 2026-06-07)** — siehe `team-datev-lohn-spec.md`. FE mock-first gebaut (`PayrollPrepPanel` + `payrollRuns`/`payrollSettings`-Stores). Backend: `payroll_runs` (period, group, status locked/exported, exported_at, employee_count) + **DATEV-Datei-Generierung** (LODAS / Lohn&Gehalt-Format mit Lohnarten + Abwesenheitsschlüssel) bzw. **Lohnimportdatenservice** (DATEVconnect, Akkreditierung). Bewegungsdaten-Aggregation aus Zeiterfassung+Abwesenheiten pro Periode/Gruppe. tenant_settings (module_id='team', key='payroll.*') für Berater-/Mandanten-Nr + Mappings.
- **Lohnauswertungsdatenservice** (Phase 2): Abrechnungen/Auswertungen zurück nach Cosmi importieren.
- ⚠ **Demo-Daten-Lücke (modulweit, vorbestehend):** `/api/v1/hr/employees` liefert im Demo-Mode `userName` leer → ganzes team-Modul zeigt „Unbekannt" (Members + Lohnvorbereitung). Demo-Fixtures sollten `userName` befüllen.

## ⚪ Später / Post-Launch / Architektur
- dialer: Gesprächsaufzeichnung (recording_url, an Video-Infra gekoppelt), AMD, Predictive (Phase 3 — bewusst)
- crm: Mobile App (PWA-Architekturentscheidung)
- mails: Exchange/EWS, PGP/S-MIME
- formulare: öffentlicher Submit-Endpoint (IsPublic-Flag da), File-Upload-Feldtyp, Submission-Mail

---

# Welle 2 — System, Produktivität, Finanzen, Automatisierung, Video

## 🔴 Pre-Launch wichtig
- **security — „Passwort vergessen"-Flow**: Login hat keinen Reset-Link; BE-Endpoint (Mail-Token) fehlt. Vor Launch nötig.
- **profil/settings — User-Preferences-Persistenz**: Sprache/Theme/Region nur client-seitig (Store/localStorage). Für Multi-Device BE-Endpoint `GET/PUT /users/preferences`. (Für Electron-Single-Device tolerierbar.)

## 🟠 Wichtig
### admin
- Tenant-Provisioning-Endpoint (`POST /api/v1/tenants`) + Onboarding-Flow
- Super-Admin/System-Level-Rolle (über Tenant hinaus)
- Billing/License-Service (`/api/v1/billing`) — aktuell nur statische Mock-Daten
- Tenant-Ressourcen-Monitoring-API (Metrics intern da, kein HTTP-Endpoint)

### profil
- Avatar-Upload-Endpoint (MinIO) — Camera-Button im FE wartet darauf

### settings
- Workspace-Branding-Persistenz (`/api/v1/tenant/branding`) — aktuell nur localStorage
- Modul-Aktivierungs-Toggle exponieren (Flag-Registry existiert)

#### Settings-Fundament (Scope-Hierarchie, FE seit 2026-06-07 mock-first)
3-Ebenen-Modell: **Tenant-Default → Modul-Leiter-Override (tenant-weit) → User-Override (persönlich)**. FE komplett gebaut (`ModuleSettingsShell`, `useIsModuleLead`, `useModuleLeadsStore`), persistiert aktuell nur in localStorage.
- **`tenant_module_leads`-Tabelle** (`tenant_id`, `user_id`, `module_id`, `granted_by`, `granted_at`) — wer ist Modul-Leiter für welches Modul. Admin setzt das im Team-Modul (MemberDetailPanel → „Erweiterte Moduleinstellungen"). Endpoints: `GET /api/v1/tenant/module-leads?user_id=`, `PUT/DELETE .../module-leads/{user_id}/{module_id}`.
- **Settings-Scope-Persistenz**: zwei Ebenen statt einem Blob. `tenant_settings` (`tenant_id`, `module_id`, `key`, `value`) für tenant-weite Defaults (nur Modul-Leiter/Admin schreibbar) + `user_settings` (`user_id`, `module_id`, `key`, `value`) für persönliche Overrides. Resolve-Reihenfolge serverseitig erzwingen (RBAC: tenant-Writes nur mit module-lead/admin).
- Beispiel CalendarSettings: `defaultView/weekStartsOn/defaultReminder` = user-scope; `workStartHour/workEndHour/holidayRegion` = tenant-scope.

### notifications
- E-Mail- + SMS-Kanal im Gateway exponieren (Dispatcher existiert intern)

### work
- `start_date`-Feld im Task-Modell + Proto (für vollwertigen Gantt; aktuell nur due_date)
- Projekt-Portfolio-Entität + Aggregations-Endpoint

### zeiterfassung
- Stundenkonto-Saldo-Endpoint (kumuliert, Perioden-Übertrag)
- Export-API (CSV / DATEV-Lohn)
- HR-Worktime-Entry um `project_id`/`customer_id`/`service_code` erweitern

### wiki
- Share-Token-Routes in `route_wiki.go` registrieren (Repo-Methoden existieren) + öffentl. Read-Endpoint
- Artikel-Templates-Endpoint (FE-Dialog existiert)

### finanzen (Symbiose-Ziel — NICHT Vollersatz, siehe finanzen-buchhaltung-strategy.md)
Strategie: Cosmi macht Vorkette (Angebot→Zahlungseingang) eigenständig, übergibt an DATEV/Bexio. Steuerberater macht Kontierung/Bilanz/USt/Lohn.
- **DATEV EXTF-Export (DE, Launch-kritisch):** Buchungsstapel (EXTF ASCII/CSV, Windows-1252) + Belegbilder-ZIP. Settings: Berater-Nr., Mandanten-Nr., Sachkonto-Länge, Steuerkennzeichen→Konto-Mapping. EXTF-Spec öffentlich, kein DATEV-Marktplatz-Partner nötig.
- **Bexio-API (CH, Launch-kritisch):** OAuth2, Rechnungen/Kontakte bidirektional sync.
- **E-Rechnung (Launch-Blocker):** XRechnung-UBL + ZUGFeRD 2.x (EN-16931) Generierung Ausgang + Empfang/XML-Extraktion Eingang. Empfangspflicht DE seit 01.01.2025.
- **GoBD-Belegarchiv (Launch-Blocker):** unveränderbar, Änderungshistorie, 8 J. Retention.
- Wiederkehrende Rechnungen (Tabelle + Scheduler + CRUD) · OP-Liste · mehrstufiges Mahnwesen
- Zahlungsabgleich: CAMT.053/MT940-Import + Matching · später finAPI/HBCI-Banking
- `currency`-Feld + Wechselkurslogik (aktuell EUR hardcoded; CHF/USD)
- BMD-Export (AT) + Lexware/lexoffice-Anbindung (Selbstbucher) — Post-Launch

### kontakte — Beratungsprotokoll (Finanzberatung, P8) — siehe kontakte-p8-beratungsprotokoll-spec.md
- `advisory_protocols`-Tabelle (contact_id, ~40 Felder über 8 Abschnitte, **immutable nach Aushändigung**, 10-Jahre-Retention, DSGVO Art.6(1)(c)). Endpoints CRUD + PDF-Export (Aushändigung).
- „Empfohlen von"-Feld am Contact (Self-Referenz) + Empfehler-Report-Aggregation.
- Mandanten-Segment A/B/C (regelbasiert nach Umsatzpotenzial) — Feld + Berechnungsregel.

### automatisierung
- Branch-/Merge-Step im Workflow-Modell + Engine (aktuell sequenziell)
- `http_request`-Action + inbound `webhook.received`-Trigger
- Zeitbasierte/Cron-Trigger: Poller-Integration verifizieren/aktivieren

### video / meetings
- Breakout-Räume (LiveKit kann es technisch)
- Recording-Download/List-UI-Endpoint exponieren
- Meeting-Recurrence-Logik

## ⚪ Später / Post-Launch
- buchhaltung (Brücke): automatische Kontierung (Kontenplan SKR03/04), EÜR-Endpoint, Steuerberater-Rolle (read-only)
- security: SSO (SAML/OAuth2/OIDC), Federation (LDAP/AD), WebAuthn/Passkeys
- admin/dashboard: zentraler modul-übergreifender Activity-Feed-Aggregator

---

# Welle 3 — Branchen-Module (Post-Launch / Solar-Pilot)

## ⭐ Cross-cutting (einmal bauen → viele Module profitieren)
- **S3/MinIO-Foto-Upload-Service**: generischer Upload-Endpoint für Foto-Anhänge — gebraucht von fuhrpark (Schaden), inventar (Bewegung), rapporte (Doku), vermietung (Protokoll), chat, profil (Avatar). Aktuell überall Mock.
- **Signatur-Persistenz**: `signature`-Feld/-Endpoint — gebraucht von rapporte, vermietung, vertraege. SignatureCanvas existiert im FE, BE nimmt es nirgends an.
- **Einkauf↔Inventar-Sync**: Wareneingang (`einkauf.ReceiveGoods`) muss `inventar.RecordMovement` triggern (Code-Kommentar „Sprint-3-Item").

## 🟠 Branchen-BE-Lücken (für Solar-Pilot ab Nov relevant)
### fuhrpark
- Führerscheinkontrolle-Modell (`LicenseCheck`: Fahrer, Ablauf, letzter Check) + Route
- Fahrtenbuch-Modell (`LogbookEntry`, finanzamtkonform) + Route `/fuhrpark/logbook` + PDF-Export
- Fahrzeugbuchung/Pool (`VehicleBooking` + Conflict-Check)
- `FuelRecord`-Modell (Tankprotokoll) + Tankkartenverwaltung
- GPS/Telematik: Provider-Integration + Webhook-Endpoint

### inventar
- `batch_number`/`serial_numbers` im Item-Modell (Chargen/Seriennummern)
- Inventur-Modell (`InventurSession`: open/counting/review/completed, Soll/Ist) + Routes
- Kommissionierung/Picklisten-Modell

### vermietung
- Strukturiertes Checklist-Format für Zustandsprotokolle (BE nur notes+photo_urls)
- `signature_url` im Inspection-Modell
- Online-Buchungsportal (öffentl. Endpoint/Embed)
- Tarif-Erweiterung (Wochensatz, Staffeln, Saison)

### einkauf
- `SupplierRating`-Modell (quality/delivery/price)
- `FrameworkContract`-Modell (Rahmenverträge) + Katalogartikel
- 2-stufiger Bestellfreigabe-Workflow (`approved_by`, `/approve`-Endpoint)
- Automatische Bestellvorschläge (Inventar-MinQty → PO)

### produktion (Brücke — MRP-Tiefe bewusst begrenzen)
- BOM-Modell (`bom` + `bom_items`) + CRUD
- `progress`/`work_steps`/`scrap`-Felder auf Order
- Maschinen-Stammdaten-Register (aktuell nur String-ID)
- Material-Verfügbarkeit: Inventar-Abgleich (FE nutzt Fake-Hash)
- QualityCheck-Modell (FE-UI existiert)
- Kalkulation (Soll/Ist-Kosten)

### schichten (Self-Service Pilot-kritisch)
- `shift_swap_requests`-Modell + approve/reject (FE-Tab da)
- Availability-Tabelle (employee × weekday) + Qualifikations-Tabelle
- Echter regelbasierter Auto-Planer (ApplyTemplate ist nur Datums-Kopie)
- `is_minor`-Flag + JArbSchG-Compliance

### rapporte (Solar-Außendienst)
- Signatur-Persistenz (s. cross-cutting)
- Aufmaß-Modell (`Measurement`/`MeasurementPosition`) — FE komplett, BE null
- `weather`-Feld auf WorkReport (Bautagesbericht)
- ReportLine: Material-vs-Leistung-Unterscheidung schärfen

## 🟠 Mobile/Offline (Solar-Pilot)
- PWA/Mobile-Zugangsweg (App ist Electron-Desktop) — rapporte + schichten brauchen Außendienst-Zugang
- `rapporte-client.ts` + `schichten-client` auf offline-queue-fähigen Basis-`client.ts` umstellen (offline-queue.ts existiert)

## 🟡 Leads (CRM, Phase 4 — 2026-06-06)
- **Architektur-Entscheidung:** Lead als **Kontakt-Lifecycle-Status** modellieren, nicht als separate Tabelle. Empfehlung: `contacts.lifecycle_stage` (`lead` → `qualified` → `customer`). Frontend baut die Inbox als gefilterte Sicht.
- Lead-Metadaten am Kontakt: `lead_source` (manual/csv/dialer), `lead_score` (0–100, auto), `lead_temperature_override` (hot/warm/cold, sticky), `lead_status` (new/contacted/qualified/disqualified).
- Endpoints: `GET /api/v1/leads` (= Kontakte mit lifecycle=lead), Status-/Temperatur-Patch, Convert (lead → contact + opt. company + opt. deal). FE nutzt aktuell In-Memory-Mock (`api/hooks/useLeads.ts`) — swapbar.
- Scoring-Regel serverseitig spiegeln (FE-Logik in `computeLeadScore`): Quelle-Basis + Vollständigkeits-Boni.
- Quelle „Dialer": Dialer-Outcomes mit Rückruf-Wunsch sollten automatisch Leads erzeugen (Verknüpfung Dialer↔Leads).

---

## 🟠 kommunikation (Team-Chat + Posteingang, vereintes Modul, Stand 2026-06-08)

> Beim Scharfschalten: echte gRPC-Endpoints existieren überwiegend, aber im **Demo-Mode (MSW)** fehlen Handler — FE-seitig nachgebaut wo nötig. Echtes Backend / Verkabelung durch Luke:

- **🔴 Chat-Reactions NICHT verdrahtet (Luke):** chat-proto deklariert `ToggleReaction/ListReactions/GetReactionSummary`, aber der **chat-Service implementiert sie nicht** (kein `reaction`-Ordner in `internal/chat/`) und das **Gateway exponiert keine chat-Reactions-Route** (route_chat.go hat keine `/reactions`). Einzige echte Impl = `internal/work/reaction` (video/work). `MessageInfo` hat zudem kein `reactions`-Feld (weder proto noch OpenAPI). → To-do Luke: (a) Reactions am chat-Service implementieren ODER work-reaction-Service mitnutzen, (b) Gateway-Route z.B. `POST/GET /api/v1/messages/{id}/reactions`, (c) `reactions` in `MessageInfo` + OpenAPI aufnehmen. **FE-Status:** MessageBubble nutzt jetzt einen session-persistenten Zustand-Store `stores/chatReactions.ts` als Demo-Backing (seedet deterministisch, persistiert Toggles, überlebt Virtualisierung). Wiring-ready: Store-Reads/Writes gegen echten Hook tauschen, sobald (a)–(c) stehen.
- **Chat-Volltextsuche:** `GET /api/v1/chat/search?q=&channel_id=` (SearchChat-RPC) existiert im Backend; FE-Hook `useChatSearch` + Demo-Handler (durchsucht Mock-Messages) jetzt gebaut. Realer Index/Ranking + File-Treffer = Luke.
- **✅ File-Upload im Chat (FE verdrahtet):** `POST /api/v1/files/upload` (multipart, nimmt `channel_id` + optional `message_id`) + `GET /{id}/files` + download/thumbnail existieren, `MessageInfo.files` (FileInfo) ist in proto+OpenAPI. FE lädt jetzt beim Senden via `useChatFileUpload` mit `message_id` hoch und rendert `message.files`. **Luke (optional/nice):** echter Storage/Virus-Scan/Thumbnail-Gen statt Stub; `GetFileThumbnailURL` für Bild-Previews verdrahten.
- **Gruppen-DMs, Pin/Lesezeichen, Channel-Notification-Settings:** im chat-proto nicht vorhanden — Neubau.
- **✅ Mentions-Inbox (FE verdrahtet):** `GET /api/v1/messages/mentions` (`GetUserMentions`) + `UserMentionsResponse` in OpenAPI. FE-Hook `useUserMentions` + `MentionsPanel` + Demo-Handler gebaut. Real out-of-the-box.
- **Posteingang (Inbox):** Demo-Handler fehlen für `snooze`, `claim`, `teams` (CRUD), `rules` (CRUD) — echte Endpoints existieren. Zusätzlich Backend-Neubau: Conversation-**Status** (offen/wartend/gelöst/geschlossen), echtes **Threading** (mehrere Msg/Conversation), **Tags-CRUD**, **Forward**, **SLA-Tracking**.
- **Audio/Video aus Chat + Bots/Webhooks/Slash-Commands:** keine RPCs — kompletter Neubau (Phase 5 baut UI-Shell/Bridge).

# Hinweis zur Arbeitsweise (Claude = FE, Luke = BE)
Die meisten 🟡-Punkte (FE-Page von Mock-Store auf fertige TanStack-Hooks umstellen) brauchen KEIN Backend von Luke — die Hooks + Endpoints existieren bereits. Luke-Bedarf = nur die 🔴- und cross-cutting-Punkte oben.

# Backend-Handover für Luke

> **Zweck:** Ein konsolidierter, nach Launch-Impact priorisierter Plan dessen, was im Backend fehlt, damit das Frontend zu Feature-Parität andocken kann. Ersetzt als Lesefassung die organisch gewachsene `backend-gaps.md` (die bleibt als Detail-/Arbeitsnotiz bestehen).
> **Stand:** 2026-06-08 · **Quelle:** `backend-gaps.md` (Wellen 1–3 + work/kommunikation-Batches)
> **Arbeitsteilung:** Claude = Frontend, Luke = Backend. Viele FE-Module sind bereits gebaut und laufen *mock-first* — sie brauchen nur das echte Backend, dann wird der Mock-Store gegen den fertigen TanStack-Hook getauscht.

## Wie dieser Plan zu lesen ist

- **Priorität = Launch-Impact**, nicht Modul-Wichtigkeit:
  - **P0 — Launch-Blocker** (Recht/Compliance/Pilot, muss vor 01.09 stehen)
  - **P1 — Feature-Parität** (schaltet bereits gebaute FE-Module scharf)
  - **P2 — Später / Post-Launch** (Branchen-Module, Komfort, Architektur)
- **FE-Status** sagt dir, wie viel Arbeit nach dem Backend noch im Frontend anfällt:
  - 🟢 **wiring-ready** — FE komplett, nur Mock-Store → echten Hook tauschen
  - 🟡 **teilweise** — FE-Overlay/Shell da, Verkabelung + ggf. UI-Feinschliff offen
  - 🔴 **FE wartet** — ohne Backend kein sinnvolles FE möglich / kompletter Neubau

---

# P0 — Launch-Blocker

## finanzen (Buchhaltung) — DE/CH gesetzlich + Steuerberater-Übergabe
Strategie: Cosmi macht die Vorkette (Angebot → Zahlungseingang) eigenständig und übergibt an DATEV/Bexio. Modul-Code ist zu ~90 % gebaut (Faktura-Kette/Mahnwesen/Belegkette/Banking/Export-Gerüst), das Folgende sind die echten Backend-Lücken.

- **E-Rechnung (Pflicht):** XRechnung-UBL + ZUGFeRD 2.x (EN-16931) erzeugen (Ausgang) + Empfang/XML-Extraktion (Eingang). Empfangspflicht DE seit 01.01.2025.
  - *Vorschlag:* Generator-Service + `POST /api/v1/finance/invoices/{id}/erechnung` (Format-Param) · Eingangs-Parser-Endpoint.
  - *FE-Status:* 🟢 Rechnungskette steht; FE braucht nur Download/Status.
- **GoBD-Belegarchiv:** unveränderbar, mit Änderungshistorie, 8 Jahre Retention.
  - *Vorschlag:* WORM-Storage + `document_events`-Audit + Retention-Policy.
  - *FE-Status:* 🟢 Belegkette gebaut.
- **DATEV EXTF-Export (DE):** Buchungsstapel (EXTF ASCII/CSV, Windows-1252) + Belegbilder-ZIP. Spec öffentlich, kein DATEV-Marktplatz-Partner nötig.
  - *Vorschlag:* Export-Job + Settings (Berater-Nr., Mandanten-Nr., Sachkonto-Länge, Steuerkennzeichen→Konto-Mapping).
  - *FE-Status:* 🟢 Export-UI + Settings-Panel (FinanceSettingsPanel) vorhanden.
- **Bexio-API (CH):** OAuth2, Rechnungen/Kontakte bidirektional.
  - *Vorschlag:* OAuth2-Connect + Sync-Service + Mapping-Settings.
  - *FE-Status:* 🟢 Integrations-Karte vorhanden.

## kalender — Online-Terminbuchung (🔴 ZFA-Pilot-kritisch)
Die ZFA-Akquise hängt daran. FE-Flow existiert komplett als Mock, Backend fehlt ganz.
- *Vorschlag:*
  - `GET/POST/PUT/DELETE /api/v1/calendar/booking-pages` (Slug, Services, Verfügbarkeitsregeln)
  - `GET /api/v1/public/book/:slug` — **öffentlich/unauthenticated** (Kunde bucht ohne Login)
  - `GET .../availability?date=&service=` — freie Slots aus Kalender-Belegung
  - `POST /api/v1/public/bookings` — öffentliche Terminanlage → Event + Bestätigung
- *FE-Status:* 🟢 Buchungs-Flow als Mock fertig.

## dialer — DSGVO-Consent-Absicherung (⚠ Risiko, nicht nur Feature)
`consentAsserter` ist im Standard-`NewService`-Konstruktor `nil`; nur `NewServiceWithConsent` verdrahtet den Einwilligungs-Check.
- *To-do:* Prüfen, ob der Standard-Konstruktor irgendwo aktiv ist → sonst Anrufe **ohne Consent** möglich. Für Finanzberatung heikel.
- *FE-Status:* n/a (reines Backend-/Risiko-Thema).

## security — „Passwort vergessen"-Flow
Login hat keinen Reset-Link.
- *Vorschlag:* `POST /api/v1/auth/forgot-password` (Mail-Token) + `POST /reset-password`.
- *FE-Status:* 🔴 Login-Screen-Redesign wartet u. a. darauf (siehe `pre-launch-todos.md`).

---

# P1 — Feature-Parität (gebaute FE-Module scharfschalten)

## kommunikation (Team-Chat + Posteingang, vereintes Modul)
Die meisten gRPC-Endpoints existieren; im **Demo-Mode (MSW)** wurden Handler nachgebaut. Echtes Backend / Verkabelung:

- **🔴 Chat-Reactions:** chat-proto deklariert `ToggleReaction/ListReactions/GetReactionSummary`, aber der chat-Service implementiert sie nicht (kein `reaction`-Ordner) und das Gateway hat keine Route. `MessageInfo` hat kein `reactions`-Feld.
  - *Vorschlag:* (a) Reactions am chat-Service implementieren **oder** `internal/work/reaction` mitnutzen, (b) `POST/GET /api/v1/messages/{id}/reactions`, (c) `reactions` in `MessageInfo` + OpenAPI.
  - *FE-Status:* 🟢 session-persistenter `stores/chatReactions.ts` als Demo-Backing — Reads/Writes gegen echten Hook tauschen.
- **Posteingang-Erweiterungen** (FE läuft mock-first als verdrahtungs-bereites Overlay):
  - **Inbox-Status** (offen/wartend/gelöst/geschlossen): kein Feld in `InboxMessage`. → `status`-Feld + Filter in `ListMessagesRequest` + Set-Status-RPC. *FE:* 🟢 `stores/inboxStatus.ts`.
  - **Threading** (mehrere Msg/Conversation): kein Thread-RPC. → `ListThreadMessages`/Conversation-Modell. *FE:* 🟡 `stores/inboxThread.ts` synthetisiert Seed + persistiert Replies/Notizen.
  - **Tags-CRUD:** `repeated string tags` existiert, aber kein Add/Remove-RPC. → `AddTag`/`RemoveTag`. *FE:* 🟢 `stores/inboxTags.ts`.
  - **Forward:** kein RPC. → `ForwardMessage`. *FE:* 🟢 `ForwardDialog`.
  - **SLA-Tracking:** noch nicht modelliert. *FE:* 🔴.
- **Canned Responses:** CRUD-Endpoints fehlen. *FE:* 🟢 `CannedResponseManager` auf Store-Backing.
- **Channels-Connect** (E-Mail/WhatsApp/Widget): echte OAuth/Connect-Flows. *FE:* 🟡 `ChannelSettingsDialog` (Mock-Toast). Routing-Rules-Infra im inbox-Service ist die Basis.
- **Interne Notizen + @Mention im Kunden-Thread:** Persistenz + echtes @Mention-Notify. *FE:* 🟡 als `direction:'internal'` im Thread-Overlay.
- **Collision-Hinweis „X bearbeitet gerade":** Live-Presence-pro-Conversation (Viewers-Event). *FE:* 🟡 `lib/inbox-collision.ts` (deterministischer Mock).
- **Call-Bridge aus Posteingang:** `createCall`-Verkabelung + Kunde→user_id-Mapping. *FE:* 🟡 Buttons → `useMeetingsStore().startCall`.
- **Per-Channel-Mute / eigener Status:** serverseitige Notification-Routing-Beachtung. *FE:* 🟢 `kommunikationPrefs.mutedChannels` / `presence.myStatus`.
- **Slash-Commands + Webhooks + Bots:** Bot/Command-Runtime + Webhook-CRUD/-Delivery — kompletter Neubau. *FE:* 🟡 Mock-Shell (`SlashCommandPalette`, `WebhookConfig`).
- **Gruppen-DMs, Pin/Lesezeichen, Channel-Notification-Settings:** im chat-proto nicht vorhanden — Neubau. *FE:* 🔴.
- **✅ bereits backend-fertig (FE schon verdrahtet, nur Hinweis):** File-Upload (`POST /files/upload` mit `message_id`), Mentions-Inbox (`GetUserMentions`), Chat-Volltextsuche (`SearchChat`). Optional/nice: echter Storage/Virus-Scan/Thumbnail statt Stub.

## work (Projekte/Aufgaben)
- **🔴 Label-Taxonomie + Task-Labels:** Tasks haben `tags` (string[]), aber keine strukturierten Labels (Name+Farbe, tenant-verwaltet).
  - *Vorschlag:* Label-CRUD `/api/v1/work/labels` (GET/POST/PUT/DELETE) + `label_ids` an Task + Filter im `listTasks`-Query.
  - *FE-Status:* 🟢 `stores/workSettings.labels` + `stores/taskLabels.ts` (Chips/Picker/Filter gebaut).
- **🔴 Custom-Field-Definitionen (Task):** Werte pro Task gibt es (`GET/PUT /tasks/{id}/custom-fields`), aber keine Definitions-Verwaltung (Feld anlegen/Typ/Pflicht, tenant-weit).
  - *Vorschlag:* `/api/v1/work/custom-fields` (Definitions-CRUD) analog CRM.
  - *FE-Status:* 🟢 `stores/workSettings.customFields`.
- **Default-Status-Set:** Status-Set für neue Projekte. *Vorschlag:* tenant-Setting `default_project_statuses` + Anwendung in `createProject`. *FE:* 🟢 `stores/workSettings.defaultStatuses`.
- **Projekt-Vorlagen löschen:** Liste/Umbenennen/„aus Vorlage" laufen echt; Löschen fehlt. *Vorschlag:* `DELETE /projects/{id}` oder Template-Archivierung. *FE:* 🟢.
- **Zeit-Regeln (billable-Default, Stundensatz):** *Vorschlag:* tenant-Setting; Anwendung beim Anlegen von Time-Entries + Stunden→Rechnung. *FE:* 🟢 `stores/workSettings.billableByDefault/defaultHourlyRate`.
- **Portfolio + Auslastung/Budget:** Portfolio-Entität + Aggregations-Endpoint; per-User-Auslastungs-Aggregat (FE zeigt „Vorschau mit Beispieldaten"). *Vorschlag:* `start_date` am Task (vollwertiger Gantt), Portfolio-Aggregat, Auslastungs-Aggregat. *FE:* 🟡 Portfolio + abgeleitetes Budget echt, Auslastung Mock.
- **✅ Kalender-Sicht (P5):** KEIN neuer Bedarf — nutzt `PUT /tasks/{id}` (due_date). Latentes Komfort-Feld: `due_from`/`due_to`-Filter im `listTasks`-Query (heute irrelevant bei page_size 500).

## kontakte (CRM) — 360°-Verknüpfungen + Finanzberatung
- **Verträge am Kontakt:** kein `GET /api/v1/contracts?contact_id={id}`. *Vorschlag:* contact_id-Filter in Modell + Route → FE-Hook `useContactContracts`. *FE:* 🔴 Sektion in ContactDetailPanel wartet.
- **Rechnungen am Kontakt:** kein `GET /api/v1/finance/invoices?contact_id={id}`; finance_line_items hat keinen Kontakt-FK (Normalisierung Sprint 4 Voraussetzung). *FE:* 🔴 Sektion wartet.
- **Beratungsprotokoll (P8, Finanzberatung):** `advisory_protocols`-Tabelle (contact_id, ~40 Felder/8 Abschnitte, **immutable nach Aushändigung**, 10-Jahre-Retention, DSGVO Art. 6(1)(c)) + CRUD + PDF-Export. *FE:* 🟢 Editor-Route + Historie gebaut (mock-first).
- **„Empfohlen von" + Mandanten-Segmente:** Self-Referenz am Contact + Empfehler-Report-Aggregation; Segment A/B/C (regelbasiert) als Feld + Berechnungsregel. *FE:* 🟢 gebaut (`referrals`/`segmentSettings`-Stores).
- **Leads (Phase 4):** Lead als **Kontakt-Lifecycle-Status** modellieren (`contacts.lifecycle_stage`: lead → qualified → customer), nicht als separate Tabelle. Metadaten: `lead_source`, `lead_score` (0–100 auto), `lead_temperature_override`, `lead_status`. Endpoints: `GET /api/v1/leads` (= Kontakte mit lifecycle=lead), Status-/Temperatur-Patch, Convert. Scoring-Regel serverseitig spiegeln (`computeLeadScore`). Dialer-Outcomes mit Rückrufwunsch → autom. Lead. *FE:* 🟢 In-Memory-Mock `useLeads.ts` swapbar.
- **XLSX-Import:** CSV/vCard existieren, XLSX fehlt. *FE:* 🟡.

## team — Lohnvorbereitung / DATEV-HR
- **Lohnvorbereitung / Lohnlauf:** `payroll_runs`-Tabelle (period, group, status locked/exported, exported_at, employee_count) + **DATEV-HR-Datei-Generierung** (LODAS / Lohn&Gehalt-Format mit Lohnarten + Abwesenheitsschlüssel) bzw. Lohnimportdatenservice (DATEVconnect, Akkreditierung). Bewegungsdaten-Aggregation aus Zeiterfassung+Abwesenheiten pro Periode/Gruppe. `tenant_settings` (module_id='team', key='payroll.*') für Berater-/Mandanten-Nr + Mappings. *FE:* 🟢 `PayrollPrepPanel` + `payrollRuns`/`payrollSettings`-Stores (mock-first). Spec: `team-datev-lohn-spec.md`.
- **Lohnauswertungsdatenservice (Phase 2):** Abrechnungen/Auswertungen zurück nach Cosmi.
- **Onboarding-Workflow-API:** Template + Checklist. *FE:* 🟡.
- **⚠ Demo-Daten-Lücke (modulweit, schnell):** `/api/v1/hr/employees` liefert im Demo-Mode `userName` leer → team-Modul + Mention/Assignee-Listen zeigen „Unbekannt". *To-do:* Demo-Fixtures `userName` befüllen. **Quick Win, entsperrt mehrere Module visuell.**

## Settings-Fundament (Scope-Hierarchie) — querliegend zu allen Modulen
3-Ebenen-Modell **Tenant-Default → Modul-Leiter-Override → User-Override**. FE komplett (`ModuleSettingsShell`, `useIsModuleLead`), persistiert nur in localStorage.
- **`tenant_module_leads`-Tabelle** (tenant_id, user_id, module_id, granted_by, granted_at) + `GET /api/v1/tenant/module-leads?user_id=`, `PUT/DELETE .../module-leads/{user_id}/{module_id}`.
- **Settings-Scope-Persistenz:** `tenant_settings` (tenant_id, module_id, key, value; nur Modul-Leiter/Admin schreibbar) + `user_settings` (user_id, module_id, key, value). Resolve-Reihenfolge + RBAC serverseitig erzwingen.
- *FE-Status:* 🟢 alle Modul-Settings-Panels bauen darauf — sobald die zwei Tabellen + Resolve stehen, wird modulweit von localStorage auf echte Persistenz umgestellt.

## Cross-cutting Quick Wins (einmal bauen → viele Module profitieren)
- **S3/MinIO-Upload-Service:** generischer Foto-/Datei-Upload — gebraucht von profil (Avatar), kontakte, chat, später fuhrpark/inventar/rapporte/vermietung. *FE:* überall Mock/Camera-Button wartet.
- **Signatur-Persistenz:** `signature`-Feld/-Endpoint — gebraucht von vertraege, rapporte, vermietung. *FE:* 🟢 SignatureCanvas existiert.
- **User-Preferences-Persistenz:** `GET/PUT /users/preferences` (Sprache/Theme/Region) für Multi-Device. Für Electron-Single-Device tolerierbar. *FE:* 🟢 Store/localStorage.

---

# P2 — Später / Post-Launch

## Produktivität / System
- **vertraege:** `UploadDocument` (MinIO) vervollständigen · `contract_events`-Audit + `GET /contracts/{id}/events` · Signatur-Workflow (Skribble/DocuSign): `send-for-signing`/`sign`/Status + Webhook.
- **dokumente:** Datei-Kommentare (Tabelle+Endpoints) · externe Share-Links (Token-Store + öffentliches Resolve).
- **mails:** Multi-Account (`ListEmailAccounts`) · Vorlagen/Quicktext · Regeln & Filter · (Post-Launch: Exchange/EWS, PGP/S-MIME).
- **helpdesk:** `contact_id`/`org_id` + `source_channel` ins Ticket-Modell · Inbox→Ticket-Adapter · Knowledge-Base-Endpoint · `time_spent`.
- **berichte:** Query-Builder-Contract festzurren · `ExecuteKindCross` (datenquellen-übergreifend) · Breakout/Pivot-Schema.
- **wiki:** Share-Token-Routes registrieren + öffentl. Read · Artikel-Templates-Endpoint.
- **notifications:** E-Mail- + SMS-Kanal im Gateway exponieren (Dispatcher existiert intern).
- **automatisierung:** Branch-/Merge-Step in Workflow-Engine · `http_request`-Action + `webhook.received`-Trigger · Cron-Trigger aktivieren.
- **video/meetings:** Breakout-Räume · Recording-Download/List-UI-Endpoint · Recurrence-Logik.
- **zeiterfassung:** Stundenkonto-Saldo-Endpoint · Export-API (CSV/DATEV-Lohn) · Worktime-Entry um project_id/customer_id/service_code.

## admin / settings / finanzen-Brücke
- **admin:** Tenant-Provisioning (`POST /api/v1/tenants`) + Onboarding · Super-Admin-Rolle · Billing/License-Service (`/api/v1/billing`, aktuell Mock) · Tenant-Ressourcen-Monitoring-HTTP-Endpoint.
- **settings:** Workspace-Branding-Persistenz (`/api/v1/tenant/branding`) · Modul-Aktivierungs-Toggle exponieren (Flag-Registry existiert).
- **finanzen (Komfort):** Wiederkehrende Rechnungen + Scheduler · OP-Liste · mehrstufiges Mahnwesen · CAMT.053/MT940-Import + Matching · `currency`-Feld + Wechselkurs (CHF/USD) · BMD (AT) + Lexware/lexoffice (Post-Launch).
- **buchhaltung-Brücke:** autom. Kontierung (SKR03/04), EÜR-Endpoint, Steuerberater-Rolle (read-only).
- **security (Post-Launch):** SSO (SAML/OIDC), LDAP/AD-Federation, WebAuthn/Passkeys.

## Branchen-Module (Welle 3, Solar-Pilot ab Nov)
> Erst nach Launch relevant. Detail-Liste in `backend-gaps.md` (Welle 3).
- **fuhrpark:** Führerscheinkontrolle · Fahrtenbuch (finanzamtkonform) + PDF · Fahrzeugbuchung/Pool · Tankprotokoll/-karten · GPS/Telematik-Webhook.
- **inventar:** Chargen/Seriennummern · Inventur-Session · Kommissionierung/Picklisten.
- **vermietung:** Checklist-Zustandsprotokoll · `signature_url` · Online-Buchungsportal · Tarif-Staffeln.
- **einkauf:** SupplierRating · Rahmenverträge + Katalog · 2-stufige Bestellfreigabe · autom. Bestellvorschläge (Inventar-MinQty).
- **produktion:** BOM-Modell · progress/work_steps/scrap · Maschinen-Register · Material-Verfügbarkeit (Inventar-Abgleich) · QualityCheck · Kalkulation.
- **schichten:** `shift_swap_requests` + approve/reject · Availability + Qualifikationen · echter Auto-Planer · `is_minor` + JArbSchG.
- **rapporte:** Signatur-Persistenz · Aufmaß-Modell · `weather`-Feld · Material/Leistung-Unterscheidung.
- **Mobile/Offline:** PWA-Zugangsweg (App ist Electron-Desktop) · `rapporte-client`/`schichten-client` auf offline-queue-fähigen `client.ts` umstellen.
- **Einkauf↔Inventar-Sync:** `einkauf.ReceiveGoods` muss `inventar.RecordMovement` triggern.

---

## Empfohlene Reihenfolge für Luke

1. **P0 zuerst** — finanzen (E-Rechnung/GoBD/DATEV/Bexio), kalender-Terminbuchung (ZFA-Pilot), dialer-Consent-Fix, Passwort-vergessen.
2. **Quick Wins parallel** — Demo-`userName`-Fix, S3/MinIO-Upload, Settings-Scope-Tabellen (entsperren je mehrere Module auf einmal).
3. **P1 modulweise scharfschalten** — pro Modul die 🟢-Punkte, weil dort nach dem Backend nur ein Hook-Tausch im FE bleibt: kommunikation → work → kontakte → team.
4. **P2 nach Launch.**

> Bei 🟢-Punkten genügt oft schon der Endpoint — das FE zieht ohne weitere Abstimmung nach (Mock-Store → Hook). Bei 🔴-Punkten bitte kurz mit Darien/Claude die Contract-Form (Felder/Shape) abstimmen, bevor du baust.

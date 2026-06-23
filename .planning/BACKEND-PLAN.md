# Cosmi 1.0 — Backend-Plan (Luke-Track)

> **Parallel-Plan zu `MASTER-PLAN.md`** (Frontend). Zeigt: was im Backend noch fehlt + **welches Frontend auf welches Backend wartet**.
> **Quelle:** `backend-gaps.md` (Detail-Arbeitsnotiz) + `archiv/backend-handover-luke.md` + Ist-Abgleich 23.06.
> **Ziel:** Cosmi 1.0. Kein ZFA-/Launch-Blocker-Treiber mehr — Priorität = **„entsperrt die meisten FE-Module".**

---

## 0 · Methodik & Legende

- **Das Kern-Backend steht** (24 Services, alle Module mit Basis-RPCs, RLS/Option-B produktiv). „Nahezu fertig" stimmt **fürs Fundament** — die Liste hier sind die **Feature-Lücken** obendrauf.
- **FE-Bereitschaft** (wie viel FE-Arbeit nach dem Backend bleibt):
  - 🟢 **wiring-ready** — FE komplett, nur Mock-Store → echten Hook tauschen (Endpoint genügt, FE zieht ohne Abstimmung nach)
  - 🟡 **teilweise** — FE-Overlay/Shell da, Verkabelung + Feinschliff offen
  - 🔴 **FE wartet** — ohne Backend kein sinnvolles FE / Contract-Form vorab mit FE abstimmen
- **Priorität (1.0-Sicht):** `B-P0` Fundament (entsperrt mehrere Module) · `B-P1` Modul-Scharfschaltung · `B-P2` Vertiefung/Post-1.0 · `📱` Post-1.0 (Handy-App).

---

## 1 · Bereits erledigt (Backend steht — nur noch FE-Echt-Schaltung)

> Diese sind **fertig im Backend** — das FE muss nur noch von Mock auf den echten Hook umstellen (🟢). Das ist Batch 1 im MASTER-PLAN.

- ✅ **kalender** Terminbuchung (Booking-Pages + öffentliche Buchung, Migr. 135/136)
- ✅ **finanzen** E-Rechnung (XRechnung/ZUGFeRD + Eingang, Migr. 140) · GoBD-Archiv (Migr. 139) · DATEV-EXTF-Export (existiert komplett)
- ✅ **security** Passwort-vergessen (Migr. 134) · GDPR-Export/-Erasure echt (`47d210d9`)
- ✅ **settings** Settings-Fundament: `tenant_settings`/`user_settings`/`tenant_module_leads` + 3-Ebenen-Resolve (Migr. 138)
- ✅ **kontakte** Beratungsprotokoll (Migr. 137, 7 RPCs) · 360°-Endpoints `contracts?contact_id` + `invoices?contact_id` (Migr. 141)
- ✅ **kommunikation** Chat-Reactions (`c9c19380`) · File-Upload · Mentions-Inbox · Volltextsuche-RPC
- ✅ **work** Label-Taxonomie + Task-Labels (Migr. 145/147) · Custom-Field-Definitionen (Migr. 146/147)
- ✅ **team** CreateEmployee-Endpoint (Teilschnitt) · HR-Read-Layer-Bug (`display_name`) gefixt
- ✅ **dialer** DSGVO-Consent-Asserter verdrahtet · **dokumente** Presign-Upload-Code (Prod-Rollout offen)

→ **FE-Aufgabe:** für all diese die 🟢-Hooks von Mock auf echt umstellen (MASTER-PLAN Batch 1).

---

## 2 · Backend-Lücken nach Priorität

### B-P0 · Fundament-Bausteine (entsperren mehrere Module auf einmal)
| # | Baustein | Was | Entsperrt | FE |
|---|---|---|---|---|
| F-1 | **S3/MinIO-Upload-Service** | Generischer Foto-/Datei-Upload-Endpoint | profil(Avatar), chat, fuhrpark, inventar, rapporte, vermietung | 🟢 (überall Camera-Button/Mock wartet) |
| F-2 | **Signatur-Persistenz-Service** | `signature`-Feld/-Endpoint (SignatureCanvas existiert) | vertraege, rapporte, vermietung | 🟢 |
| F-3 | **dokumente Presign Prod-Rollout** | DNS `s3.zentria.tech` + Env + CORS-Origin (Code da) | dokumente, alle Uploads | 🟢 |
| F-4 | **Demo-`userName`-Fix** | `/hr/employees` Demo liefert `userName` leer → „Unbekannt" überall | team, Mentions, Assignee-Listen | 🟢 Quick Win |
| F-5 | **OpenAPI-Specs nachholen** | formulare, dialer, inventar, vermietung, vertraege, mails (untypisierte Clients) | diese 6 Module | 🟡 |

### B-P1 · Modul-Scharfschaltung (gebaute FE-Module echt schalten)
| # | Modul | Backend-Lücke | FE |
|---|---|---|---|
| M-1 | **kommunikation/Inbox** | Inbox-`status`-Feld + Set-RPC · Threading (`ListThreadMessages`) · Tags Add/Remove-RPC · `ForwardMessage` · Canned-CRUD · interne-Notiz-Persistenz + @Mention-Notify | 🟢/🟡 (Overlays gebaut) |
| M-2 | **kommunikation** | Channels-Connect (OAuth E-Mail/WhatsApp/Widget) · Collision-Presence · Call-Bridge (createCall + Kunde→user_id) · Slash/Bots/Webhooks · Gruppen-DMs/Pin | 🟡/🔴 |
| M-3 | **work** | `start_date` am Task (Gantt) · Portfolio-Entität + Aggregat · Auslastungs-Aggregat (per-User) · `DELETE /projects/{id}` · Default-Status-Set + Zeit-Regeln als tenant_settings | 🟢/🟡 |
| M-4 | **kontakte/crm** | Lead als `lifecycle_stage` (lead→qualified→customer) + `GET /leads` + Convert · Lead-Scoring-Engine · Segment-Override-Feld · CustomFields-CRUD · `/tags` in OpenAPI · XLSX-Import · Beratungsprotokoll-Server-PDF (revisionssicher) | 🟢/🟡 |
| M-5 | **zeiterfassung** | `GET /hr/time/balance` · `POST /entries` · `GET /projects` · `GET /analytics` · Wochen-Freigabe (`submit/approve/reject` + `time_week_submissions`) · `GET /team` · `project_id/customer_id/billable` an `hr_work_time_entries` | 🟢 (alle mock-first verdrahtet) |
| M-6 | **dokumente** | Datei-Kommentare · externe Share-Links (Token+Resolve) · Versions-Download-Endpoint · Template-Storage · `document_activities` + Activity-Endpoint · Thumbnail-Rendering | 🟡 (FE mock-first) |
| M-7 | **team** | Auth-User-Invite-Flow (email+temp-pw+roles, transaktional) · DATEV-HR-Lohn (LODAS) + `payroll_runs` · Onboarding-Workflow-API | 🟢/🟡 |
| M-8 | **dashboard** | KPI-Service echt (Werte + `change_percent` pro Modul) · KPI-Zeitreihe für Sparkline (8 Perioden) · Widget-Layout-Persistenz | 🟢 |
| M-9 | **settings/profil** | FE localStorage→`tenant_settings`/`user_settings` (Persistenz steht, FE-Wiring) · Workspace-Branding-Persistenz · Modul-Leiter-CRUD-UI-Endpoints · Avatar-Upload (=F-1) · User-Preferences-Persistenz | 🟢 |
| M-10 | **wiki** | Share-Token-Routes registrieren + öffentl. Read · Artikel-Templates-Endpoint | 🟡 |
| M-11 | **notifications** | E-Mail-/SMS-Kanal im Gateway exponieren (Dispatcher intern da) · `priority`-Enum + Spalten `is_pinned/is_dismissed/actor_name` · Real-Time WebSocket | 🟡 |
| M-12 | **berichte** | Server-PDF-Service (Playwright `page.pdf`) · Cron-Executor+Mailer + `report_runs` · `ExecuteKindCross` (datenquellen-übergreifend) · Breakout/Pivot-Schema · öffentl. Share-Token-Seite | 🟡 |
| M-13 | **helpdesk** | `contact_id`/`org_id` + `source_channel` ins Ticket · KB-Endpoint · `time_spent` · Inbox→Ticket-Adapter | 🟡 |
| M-14 | **vertraege** | `contract_events`-Audit + `GET /{id}/events` · (UploadDocument ✅) · E-Signatur-Workflow → P2 | 🟡 |
| M-15 | **automatisierung** | echtes CRUD/Execution-Log · Branch/Merge-Step · `http_request`-Action + `webhook.received`-Trigger · Cron-Trigger | 🔴 (Engine = großer Block) |
| M-16 | **mails** | IMAP/SMTP-Sync · Multi-Account (`ListEmailAccounts`) · Templates · Regeln/Filter | 🔴 (Neubau-nah) |
| M-17 | **video** | Recording-Download/List-Endpoint · Breakout-Räume · Recurrence-Logik | 🟡 |
| M-18 | **finanzen** | Bexio-OAuth-Sync (CH) · wiederkehrende Rechnungen · OP-Liste · mehrstufiges Mahnwesen · CAMT/MT940-Matching · `currency`-Feld (CHF/USD) | 🟢/🟡 |

### B-P2 · Branchen-Vertiefung (nach den Kern-Modulen)
- **rapporte:** Aufmaß-Modell · PDF-Export · `weather` · Material/Leistung · Approval-Backend · Signatur (=F-2)
- **schichten:** `shift_swap_requests`+approve/reject · Availability/Qualifikation · echter Auto-Planer · `is_minor`/JArbSchG
- **fuhrpark:** Fahrtenbuch (§7 EStG)+PDF · Führerschein-OCR · Buchungs-Pool · Tankprotokoll · GPS-Webhook
- **inventar:** Chargen/Seriennummern · Inventur-Session · Picklisten · **Einkauf↔Inventar-Sync** (`ReceiveGoods`→`RecordMovement`)
- **vermietung:** Checklist-Zustandsprotokoll · `signature_url` · Online-Buchungsportal · Tarif-Staffeln
- **einkauf:** SupplierRating · Rahmenverträge+Katalog · 2-stufige Bestellfreigabe · Bestellvorschläge
- **produktion:** BOM · progress/work_steps/scrap · Maschinen-Register · MRP (Inventar-Abgleich) · QualityCheck · Kalkulation

### B-P2 · Plattform / später
- Tenant-Provisioning (`POST /tenants`) + Super-Admin-Rolle · **Billing/License-Service** (`/billing`, aktuell Mock) · Tenant-Ressourcen-Monitoring-Endpoint
- Auto-Update (electron-updater) + Code-Signing · DB-Partitionierung (R2-P1.10) · Migrations-Drift prod(209)↔repo(213+) · buchhaltung-Brücke (SKR03/04, EÜR, Steuerberater-Rolle) · SSO/SAML/OIDC/WebAuthn

### 📱 Post-1.0 (Handy-App-Phase)
PWA-Zugangsweg (App ist Electron-Desktop) · `rapporte-client`/`schichten-client` auf offline-queue-fähig · GPS-Stempel · Barcode/QR-Scan · mobile Self-Service-Portale (schichten/vermietung).

---

## 3 · FE↔BE-Warte-Mapping (Herzstück: was wartet worauf)

> Sortiert nach FE-Bereitschaft. **🟢 = sobald Luke den Endpoint baut, schaltet das FE ohne weitere Abstimmung scharf.**

| Frontend-Feature (fertig/mock) | wartet auf Backend | FE | Prio |
|---|---|---|---|
| Avatar-Bild (profil), Foto-Anhänge (chat/Branchen) | **F-1 S3-Upload** | 🟢 | B-P0 |
| Unterschriften speichern (vertraege/rapporte/vermietung) | **F-2 Signatur-Service** | 🟢 | B-P0 |
| Alle Modul-Einstellungen dauerhaft/tenant-weit | **M-9 Settings-Wiring** (Tabellen da) | 🟢 | B-P0 |
| team-Namen, @Mentions, Assignee-Listen | **F-4 Demo-userName + Invite-Flow** | 🟢 | B-P0 |
| Stundenkonto, Zeit-Analytics, Wochen-Freigabe | **M-5 hr/time/\*-Endpoints** | 🟢 | B-P1 |
| Dashboard-KPIs + Sparklines echt | **M-8 KPI-Service** | 🟢 | B-P1 |
| Kontakt→Verträge/Rechnungen (360°) | ✅ BE da (Migr. 141) → nur Hooks | 🟢 | B-P1 |
| Leads-Inbox, Lead-Scoring | **M-4 lifecycle_stage + /leads** | 🟢 | B-P1 |
| Chat-Reactions, Mentions, File-Upload | ✅ BE da → nur Echt-Schaltung | 🟢 | — |
| work Labels/Custom-Fields/Portfolio | **M-3** (Labels/CF ✅ da, Portfolio offen) | 🟢/🟡 | B-P1 |
| Posteingang Status/Threading/Tags/Forward | **M-1 Inbox-RPCs** | 🟡 | B-P1 |
| Berichte als echtes PDF + geplanter Versand | **M-12 PDF-Service + Cron** | 🟡 | B-P1 |
| Dokumente Share-Links/Versions-Download/Activity | **M-6** | 🟡 | B-P1 |
| Lohnvorbereitung/Lohnlauf (DATEV-HR) | **M-7 payroll_runs + LODAS** | 🟡 | B-P1 |
| Channels verbinden (Mail/WhatsApp), Bots, Call-Bridge | **M-2** | 🟡/🔴 | B-P1 |
| Automatisierungen wirklich ausführen | **M-15 Engine** | 🔴 | B-P1/2 |
| Echte E-Mails senden/empfangen | **M-16 IMAP/SMTP** | 🔴 | B-P1/2 |
| Branchen-Tiefe (Aufmaß, Fahrtenbuch, BOM, Inventur …) | **B-P2 Branchen-Endpoints** | 🟡/🔴 | B-P2 |

---

## 4 · Empfohlene Reihenfolge für Luke (parallel zum FE)

1. **B-P0 Fundament zuerst** — F-1 S3-Upload · F-2 Signatur · F-3 Presign-Prod-Rollout · F-4 Demo-userName · F-5 OpenAPI-Specs. Jeder Punkt entsperrt mehrere FE-Module gleichzeitig → koordiniert mit MASTER-PLAN Batch 1 (Echt-Schaltung).
2. **B-P1 modulweise** — pro Modul die 🟢-Endpoints zuerst (FE zieht sofort nach, kein Feinschliff): zeiterfassung → dashboard → kontakte/leads → work-Portfolio → settings-Wiring → dann die 🟡-Module (Inbox, berichte-PDF, dokumente, team-Lohn).
3. **Große Blöcke planen** — M-15 Automatisierungs-Engine + M-16 Mail-IMAP/SMTP sind eigenständige Bauten (🔴) → früh terminieren, da FE dort wirklich wartet.
4. **B-P2 Branchen + Plattform** nach den Kern-Modulen.

**Arbeitsregel:** Bei 🟢 genügt der Endpoint — FE schaltet ohne Abstimmung scharf. Bei 🔴 vorab Contract-Form (Felder/Shape) mit Darien/FE abstimmen. Sync-Punkt: `backend-gaps.md` bleibt die Detail-Notiz, dieser Plan die Übersicht.

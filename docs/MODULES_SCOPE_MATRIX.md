# Modules Scope Matrix — 14 Cosmi-Module

**Stand (Planungs-Snapshot):** 2026-05-10 — Tabellen/RPCs/Flag-Keys sind die Scope-Planung aus Sprint 0.
**Purpose:** Basis fuer Feature-Flag-Registry (S0.6), Sprint-1/2-Planung, Pilot-Segmentierung
**Single Source of Truth:** [docs/ROADMAP.md §5](./ROADMAP.md) — diese Matrix extrahiert operative Details.

> ⚠ **Live-Status statt Plan-Snapshot:** Diese Matrix beschreibt den *geplanten* Scope (Stand 2026-05-10) — die
> „Mock-Store"-Vermerke pro Modul sind teils ueberholt. Den **aktuellen FE↔Backend-Wiring-Reifegrad** (welche
> Module store→API verdrahtet sind) fuehrt [`.planning/status-overview.md`](../.planning/status-overview.md)
> (Modul-Reifegrad-Matrix, laufend gepflegt). Backend-RPCs sind ueber alle 14 Module in Sprint 1+2 gebaut.

> Alle 14 Module hatten initial Mock-Frontend (Stores unter `desktop/src/renderer/src/stores/<modul>.ts`) und werden bis Launch 2026-09-01 mit echtem Backend ausgestattet. `buchhaltung` und `video` sind Completion-Taetigkeiten (Backend teilweise vorhanden), alle anderen sind Neubau.

---

## Uebersicht

| Modul | Sprint | Pilot-Prio | Flag-Key | FE-LOC | Backend-Pkg | Status | Tabellen | Migrations |
|---|---|---|---|---|---|---|---|---|
| wiki | S1 | Dienstleister | `modules.wiki` | 1297 | `backend/internal/wiki/` ✅ | Live (Sprint 1) | 3 | 1 (000076) |
| berichte | S1 | Dienstleister | `modules.berichte` | 1186 | `backend/internal/berichte/` ✅ | Live (Sprint 1) | 2 | 2 (000079–080) |
| formulare | S1 | Cross | `modules.formulare` | 2238 | `backend/internal/formulare/` ✅ | Live (Sprint 1) | 2 | 1 (000081) |
| helpdesk | S1 | Dienstleister | `modules.helpdesk` | 2041 | `backend/internal/helpdesk/` ✅ | Live (Sprint 1) | 4 | 1 (000077) |
| vertraege | S1 | Dienstleister | `modules.vertraege` | 1899 | `backend/internal/vertraege/` ✅ | Live (Sprint 1) | 3 | 2 (000089–090) |
| buchhaltung | S1 | Cross | `modules.buchhaltung` | 1524 | `backend/internal/biz/` ✅ | Completion | 4 (vorhanden) | — |
| video | S1 | Cross | `modules.video` | 459 | `backend/internal/work/` ✅ | Completion | 3 (vorhanden) | — |
| rapporte | S2 | Handwerk | `modules.rapporte` | 2686 | `backend/internal/rapporte/` ✅ | Live (Sprint 2) | 3 | 3 (000092–093, 100) |
| schichten | S2 | Handwerk | `modules.schichten` | 1406 | `backend/internal/schichten/` ✅ | Live (Sprint 2) | 3 | 4 (000094–095, 102–103) |
| fuhrpark | S2 | Handwerk | `modules.fuhrpark` | 2299 | `backend/internal/fuhrpark/` ✅ | Live (Sprint 2) | 3 | 2 (000096–097) |
| vermietung | S2 | Handwerk | `modules.vermietung` | 2028 | `backend/internal/vermietung/` ✅ | Live (Sprint 2) | 3 | 3 (000098–099, 101) |
| inventar | S2 | Cross | `modules.inventar` | 1460 | `backend/internal/inventar/` ✅ | Live (Sprint 2) | 3 | 2 (000083–084) |
| einkauf | S2 | Cross | `modules.einkauf` | 1724 | `backend/internal/einkauf/` ✅ | Live (Sprint 2) | 3 | 2 (000085–086) |
| produktion | S2 | Handwerk | `modules.produktion` | 1674 | `backend/internal/produktion/` ✅ | Live (Sprint 2) | 3 | 2 (000087–088) |

---

## Detailsektionen

### wiki (S1, Dienstleister)

- **Tabellen (geplant):** `wiki_articles`, `wiki_versions`, `wiki_attachments`
- **RPCs (~14):** CreateArticle, UpdateArticle, DeleteArticle, GetArticle, ListArticles, SearchArticles, ListVersions, GetVersion, RestoreVersion, UploadAttachment, ListAttachments, DeleteAttachment, ListCategories, CreateCategory
- **Frontend-Hook (geplant):** `desktop/src/renderer/src/api/hooks/useWiki.ts`
- **Flag-Key:** `modules.wiki`
- **Frontend-Stand:** 1297 LOC in `desktop/src/renderer/src/modules/wiki/` (12 Files), Mock-Store in `stores/wiki.ts`
- **Backend-Stand:** ✅ Package vorhanden (`backend/internal/wiki/`), 1 Migration (000076)
- **Besonderheiten:** PostgreSQL Full-Text-Search (FTS) auf `wiki_articles.content`, TipTap-JSON-Format fuer Rich-Content, Share-Links via kurzem Token, Versionierung automatisch bei jedem Save
- **Status Sprint 1:** Live — Package, Migration 000076, FTS-Index `idx_wiki_articles_fts` (GIN) vorhanden

---

### berichte (S1, Dienstleister)

- **Tabellen (geplant):** `report_definitions`, `report_cache`
- **RPCs (~10):** CreateDefinition, UpdateDefinition, DeleteDefinition, GetDefinition, ListDefinitions, RunReport, GetCachedResult, InvalidateCache, ExportPDF, ExportCSV
- **Frontend-Hook (geplant):** `desktop/src/renderer/src/api/hooks/useBerichte.ts`
- **Flag-Key:** `modules.berichte`
- **Frontend-Stand:** 1186 LOC, 1 File, Mock-Store in `stores/berichte.ts`
- **Backend-Stand:** ✅ Package vorhanden (`backend/internal/berichte/`), 2 Migrations (000079 create, 000080 seed_permissions)
- **Besonderheiten:** `report_definitions.query_config` als JSONB (Aggregations-Konfiguration), `report_cache` mit TTL-Spalte fuer automatischen Verfall, PDF/CSV/XLSX-Export, In-Process-Cron-Scheduler
- **Status Sprint 1:** Live — Package, Migrations 000079–080, Export-Layer (80.2% Cov), gRPC-Server Ports 50063/9103

---

### formulare (S1, Cross)

- **Tabellen (geplant):** `form_schemas`, `form_submissions`
- **RPCs (~16):** CreateSchema, UpdateSchema, DeleteSchema, GetSchema, ListSchemas, PublishSchema, UnpublishSchema, SubmitForm, GetSubmission, ListSubmissions, DeleteSubmission, ExportSubmissions, TriggerWebhook, ListWebhooks, AddWebhook, RemoveWebhook
- **Frontend-Hook (geplant):** `desktop/src/renderer/src/api/hooks/useFormulare.ts`
- **Flag-Key:** `modules.formulare`
- **Frontend-Stand:** 2238 LOC, 1 File, Mock-Store in `stores/formulare.ts`
- **Backend-Stand:** ✅ Package vorhanden (`backend/internal/formulare/`), 1 Migration (000081)
- **Besonderheiten:** `form_schemas.schema` als JSONB (JSON Schema Draft-7), `form_submissions.data` als JSONB, Webhook-Worker (HMAC-SHA256, Exp-Backoff, Dead-Letter), CSV+XLSX Export
- **Status Sprint 1:** Live — Package, Migration 000081, Webhook-Delivery-Queue vorhanden

---

### helpdesk (S1, Dienstleister)

- **Tabellen (geplant):** `tickets`, `ticket_messages`, `ticket_queues`, `canned_responses`
- **RPCs (~22):** CreateTicket, UpdateTicket, DeleteTicket, GetTicket, ListTickets, AssignTicket, CloseTicket, ReopenTicket, MergeTickets, EscalateTicket, AddMessage, EditMessage, DeleteMessage, ListMessages, CreateQueue, UpdateQueue, DeleteQueue, ListQueues, CreateCannedResponse, UpdateCannedResponse, DeleteCannedResponse, ListCannedResponses
- **Frontend-Hook (geplant):** `desktop/src/renderer/src/api/hooks/useHelpdesk.ts`
- **Flag-Key:** `modules.helpdesk`
- **Frontend-Stand:** 2041 LOC in `desktop/src/renderer/src/modules/helpdesk/` (7 Files), Mock-Store
- **Backend-Stand:** ✅ Package vorhanden (`backend/internal/helpdesk/`), 1 Migration (000077)
- **Besonderheiten:** Ticket-Merge (Duplikat-Erkennung), SLA-Tracking via `due_at`-Spalte, Canned Responses fuer schnelle Antworten, Queue-basiertes Routing, Agenten-Zuweisung
- **Status Sprint 1:** Live — Package, Migration 000077, 22 RPCs, SLA + Merge vorhanden

---

### vertraege (S1, Dienstleister)

- **Tabellen (geplant):** `contracts`, `contract_parties`, `contract_reminders`
- **RPCs (~14):** CreateContract, UpdateContract, DeleteContract, GetContract, ListContracts, AddParty, RemoveParty, ListParties, CreateReminder, UpdateReminder, DeleteReminder, ListReminders, UploadDocument, ExportContract
- **Frontend-Hook (geplant):** `desktop/src/renderer/src/api/hooks/useVertraege.ts`
- **Flag-Key:** `modules.vertraege`
- **Frontend-Stand:** 1899 LOC in `desktop/src/renderer/src/modules/vertraege/` (2 Files), Mock-Store
- **Backend-Stand:** ✅ Package vorhanden (`backend/internal/vertraege/`), 2 Migrations (000089 create, 000090 seed_permissions)
- **Besonderheiten:** Laufzeit-Engine berechnet `expires_at` und triggert Erinnerungen N Tage vor Ablauf, `contract_parties` linked zu `contacts`/`companies`, advisory-lock-claim, 5+60min Ticker, Vertrags-PDF-Upload via MinIO
- **Status Sprint 1:** Live — Package, Migrations 000089–090, Cron-basierter Reminder-Check vorhanden

---

### buchhaltung (S1, Cross) — Completion

- **Tabellen (vorhanden):** `finance_invoices`, `finance_quotes`, `finance_payments`, `finance_dunning_records` — **alle seit Migration 000045**
- **Completion-Scope:** GoBD-konforme Journal-Nummerierung + FinanzenHook-Luecken schliessen. Tabellen existieren seit Migration 000045.
  - Laufende Journal-Nummer (`invoice_number`) sicher generieren (kein Gap, kein Rollback)
  - `useFinanzen`-Hook-Gaps schliessen: fehlende RPC-Bindings fuer Dunning-Flow
  - Vorbereitung fuer Sprint-4-Normalisierung (`finance_invoice_lines`-Tabelle)
- **RPCs (Completion, ~8 neue Gaps):** GetJournalSummary, ListDunningRecords, UpdateDunningStatus, SendDunningNotice, GenerateGoBDExport, ValidateInvoiceNumber, LockInvoice, GetPaymentStats
- **Frontend-Hook (geplant):** `desktop/src/renderer/src/api/hooks/useFinanzen.ts` (erweitern)
- **Flag-Key:** `modules.buchhaltung`
- **Frontend-Stand:** 1524 LOC in `desktop/src/renderer/src/modules/buchhaltung/` (6 Files)
- **Backend-Stand:** `backend/internal/biz/` vorhanden, Dunning-RPCs und GoBD-Journal fehlen
- **Aktion Sprint 1:** `biz`-Package um fehlende RPCs erweitern, Journal-Nummerierungs-Service implementieren, Hook-Gaps schliessen

---

### video (S1, Cross) — Completion

- **Tabellen (vorhanden):** `meetings`, `recordings`, `recording_consents` — **alle seit Migration 000037**
- **Completion-Scope:** Recording-Consent-Tagging + useVideo-Hook-Ergaenzungen. Tabellen existieren seit Migration 000037.
  - `recordings`-Tabelle um `started_by`, `consent_snapshot` (JSONB) ergaenzen (Sprint 1, Vorlauf fuer R2-P0.3-Fix)
  - `useVideo`-Hook-Luecken: Recording-Status-Polling, Consent-Modal-Integration
  - Vorbereitung fuer R2-P0.3 (Recording-Consent-Bug: alle aktiven Teilnehmer als participantIDs)
- **RPCs (Completion, ~6 neue Gaps):** GetRecordingConsents, TagRecordingWithConsents, UpdateRecordingMetadata, ListRecordingsByMeeting, GetRecordingStatus, CleanupExpiredRecording
- **Frontend-Hook (geplant):** `desktop/src/renderer/src/api/hooks/useVideo.ts` (erweitern)
- **Flag-Key:** `modules.video`
- **Frontend-Stand:** 459 LOC, 1 File
- **Backend-Stand:** `backend/internal/work/` vorhanden, Recording-Tagging und Consent-Snapshot fehlen
- **Aktion Sprint 1:** `work`-Package um Recording-Tagging erweitern, `recordings`-Migration fuer neue Spalten, Hook-Gaps schliessen

---

### rapporte (S2, Handwerk)

- **Tabellen (geplant):** `work_reports`, `report_lines`, `report_attachments`
- **RPCs (~18):** CreateReport, UpdateReport, DeleteReport, GetReport, ListReports, SubmitReport, ApproveReport, RejectReport, AddLine, UpdateLine, DeleteLine, ListLines, UploadAttachment, ListAttachments, DeleteAttachment, ExportPDF, GetReportStats, ListPendingApprovals
- **Frontend-Hook (geplant):** `desktop/src/renderer/src/api/hooks/useRapporte.ts`
- **Flag-Key:** `modules.rapporte`
- **Frontend-Stand:** 2686 LOC in `desktop/src/renderer/src/modules/rapporte/` (3 Files), Mock-Store
- **Backend-Stand:** ✅ Package vorhanden (`backend/internal/rapporte/`), 3 Migrations (000092 create, 000093 seed_permissions, 000100 rapporte_approve_permission)
- **Besonderheiten:** Foto-Uploads pro Rapport-Zeile via MinIO, GPS-Tag optional (`lat`, `lon` Spalten auf `work_reports`), mehrstufiger Approval-Flow (Submitted → Approved/Rejected), Export als PDF fuer Kundenunterschrift
- **Status Sprint 2:** Live — Package, Migrations 000092–093+100, MinIO-Attachment-Upload vorhanden

---

### schichten (S2, Handwerk)

- **Tabellen (geplant):** `shifts`, `shift_assignments`, `shift_templates`
- **RPCs (~16):** CreateShift, UpdateShift, DeleteShift, GetShift, ListShifts, PublishShifts, AssignEmployee, UnassignEmployee, ListAssignments, CreateTemplate, UpdateTemplate, DeleteTemplate, ListTemplates, ApplyTemplate, CheckArbzgCompliance, GetShiftStats
- **Frontend-Hook (geplant):** `desktop/src/renderer/src/api/hooks/useSchichten.ts`
- **Flag-Key:** `modules.schichten`
- **Frontend-Stand:** 1406 LOC, 1 File, Mock-Store
- **Backend-Stand:** ✅ Package vorhanden (`backend/internal/schichten/`), 4 Migrations (000094 create, 000095 seed_permissions, 000102 shift_assignments_tenant_unique, 000103 shift_capacity)
- **Besonderheiten:** ArbZG §5 Backend-Check (min. 11h Ruhezeit zwischen Schichten, DST-aware — Warning wenn verletzt), Schicht-Templates fuer Wochenmuster, Publish-Flow (Entwurf → Veroeffentlicht → Mitarbeiter sehen)
- **Status Sprint 2:** Live — Package, Migrations 000094–095+102–103, ArbZG-Pruef-Service vorhanden

---

### fuhrpark (S2, Handwerk)

- **Tabellen (geplant):** `vehicles`, `vehicle_services`, `vehicle_damages`
- **RPCs (~18):** CreateVehicle, UpdateVehicle, DeleteVehicle, GetVehicle, ListVehicles, ScheduleService, UpdateService, DeleteService, CompleteService, ListServices, ReportDamage, UpdateDamage, ResolveDamage, ListDamages, GetVehicleHistory, CheckTuvDue, ListUpcomingServices, ExportVehicleReport
- **Frontend-Hook (geplant):** `desktop/src/renderer/src/api/hooks/useFuhrpark.ts`
- **Flag-Key:** `modules.fuhrpark`
- **Frontend-Stand:** 2299 LOC in `desktop/src/renderer/src/modules/fuhrpark/` (2 Files), Mock-Store
- **Backend-Stand:** ✅ Package vorhanden (`backend/internal/fuhrpark/`), 2 Migrations (000096 create, 000097 seed_permissions)
- **Besonderheiten:** TUeV-Reminder via `tuev_due_date` + Cron-basierter Check via advisory-lock (7d/1d vor Faelligkeit), Fahrzeug-Service-History vollstaendig protokolliert, Schaden-Meldungen mit Foto-Upload (MinIO), Fuel-Tracking optional
- **Status Sprint 2:** Live — Package, Migrations 000096–097, TUeV-Reminder-Cron vorhanden

---

### vermietung (S2, Handwerk)

- **Tabellen (geplant):** `rental_objects`, `rentals`, `rental_inspections`
- **RPCs (~20):** CreateObject, UpdateObject, DeleteObject, GetObject, ListObjects, CheckAvailability, CreateRental, UpdateRental, DeleteRental, GetRental, ListRentals, StartRental, EndRental, CreateInspection, UpdateInspection, GetInspection, ListInspections, UploadInspectionPhoto, GetRentalCalendar, ExportRentalReport
- **Frontend-Hook (geplant):** `desktop/src/renderer/src/api/hooks/useVermietung.ts`
- **Flag-Key:** `modules.vermietung`
- **Frontend-Stand:** 2028 LOC in `desktop/src/renderer/src/modules/vermietung/` (2 Files), Mock-Store
- **Backend-Stand:** ✅ Package vorhanden (`backend/internal/vermietung/`), 3 Migrations (000098 create, 000099 seed_permissions, 000101 gist_overlap_unique_inspection)
- **Besonderheiten:** Zustandsprotokolle (Uebergabe-/Ruecknahme-Inspektion) mit Foto-Uploads, Verfuegbarkeits-Check via GIST tstzrange-Overlap-Index, Kalender-View fuer Auslastung
- **Status Sprint 2:** Live — Package, Migrations 000098–099+101, Verfuegbarkeits-Index vorhanden

---

### inventar (S2, Cross)

- **Tabellen (geplant):** `inventory_items`, `inventory_movements`, `stock_warnings`
- **RPCs (~16):** CreateItem, UpdateItem, DeleteItem, GetItem, ListItems, AdjustStock, TransferStock, RecordMovement, ListMovements, GetStockHistory, CreateWarning, UpdateWarning, AcknowledgeWarning, ListWarnings, GetStockReport, ExportInventory
- **Frontend-Hook (geplant):** `desktop/src/renderer/src/api/hooks/useInventar.ts`
- **Flag-Key:** `modules.inventar`
- **Frontend-Stand:** 1460 LOC, 1 File, Mock-Store
- **Backend-Stand:** ✅ Package vorhanden (`backend/internal/inventar/`), 2 Migrations (000083 create, 000084 seed_permissions)
- **Besonderheiten:** Bestands-Alarm via `stock_warnings` — automatisch generiert wenn `inventory_items.quantity <= min_quantity`, doppelte Buchfuehrung via `inventory_movements` (jede Bestandsaenderung trackt Bewegungstyp + User), Barcode-Feld optional
- **Status Sprint 2:** Live — Package, Migrations 000083–084, Bestands-Alarm-Trigger vorhanden

---

### einkauf (S2, Cross)

- **Tabellen (geplant):** `purchase_orders`, `suppliers`, `po_lines`
- **RPCs (~18):** CreateSupplier, UpdateSupplier, DeleteSupplier, GetSupplier, ListSuppliers, CreatePO, UpdatePO, DeletePO, GetPO, ListPOs, AddPOLine, UpdatePOLine, DeletePOLine, ListPOLines, SubmitPO, ReceiveGoods, PartialReceive, ExportPO
- **Frontend-Hook (geplant):** `desktop/src/renderer/src/api/hooks/useEinkauf.ts`
- **Flag-Key:** `modules.einkauf`
- **Frontend-Stand:** 1724 LOC, 1 File, Mock-Store
- **Backend-Stand:** ✅ Package vorhanden (`backend/internal/einkauf/`), 2 Migrations (000085 create, 000086 seed_permissions)
- **Besonderheiten:** Wareneingangs-Flow: `ReceiveGoods` bucht automatisch in `inventory_movements` (Cross-Modul-Abhaengigkeit zu `inventar`), PO-Lifecycle Submit → Sent → PartiallyReceived → Received → Closed, Lieferanten-Katalog mit Kontakt-Link zu `contacts`
- **Status Sprint 2:** Live — Package, Migrations 000085–086, Wareneingangs-Flow vorhanden

---

### produktion (S2, Handwerk)

- **Tabellen (geplant):** `production_orders`, `machine_bookings`, `production_plans`
- **RPCs (~16):** CreateOrder, UpdateOrder, DeleteOrder, GetOrder, ListOrders, StartOrder, CompleteOrder, CancelOrder, CreateMachineBooking, UpdateMachineBooking, DeleteMachineBooking, ListMachineBookings, CreatePlan, UpdatePlan, GetPlan, GetCapacityOverview
- **Frontend-Hook (geplant):** `desktop/src/renderer/src/api/hooks/useProduktion.ts`
- **Flag-Key:** `modules.produktion`
- **Frontend-Stand:** 1674 LOC in `desktop/src/renderer/src/modules/produktion/` (2 Files), Mock-Store
- **Backend-Stand:** ✅ Package vorhanden (`backend/internal/produktion/`), 2 Migrations (000087 create, 000088 seed_permissions)
- **Besonderheiten:** Maschinenbelegungs-Konflikt-Check via advisory-lock (Ueberschneidung in `machine_bookings`), Produktionsplan verknuepft Orders mit Zeitfenstern, Kapazitaets-Uebersicht per Maschine/Woche
- **Status Sprint 2:** Live — Package, Migrations 000087–088, Belegungs-Konflikt-Pruefung vorhanden

---

## Stand (2026-05-10)

Alle 14 Module haben funktionierende Backend-Packages und Migrations. Die "Neubau"-Eintraege aus Sprint-0-Planung sind obsolet — alle Module sind live (Migrationskopf 000213 / Prod 209). Feature-Flag-Registry aktiv (17 Flags). Option-B-Full-Retrofit + RLS produktiv (Migrations 000104–000127, `COSMI_ENV=production` scharf seit 2026-06-05).

Naechste offene Punkte: Sprint 4 (`finance_invoice_lines`-Normalisierung nach ADR-0007), Sprint 5 (Peer-Review, Rigorosum Runde 3).

## Aenderungshistorie

- 2026-04-18: Initial — Sprint-0 Task S0.9.
- 2026-05-10: Refresh nach Sprint 1+2+3 — alle 12 "Neubau"-Module auf "Live" aktualisiert, reale Package-/Migration-Counts eingetragen, Spalte "Migrations" hinzugefuegt.

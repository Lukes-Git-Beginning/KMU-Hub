# Modules Scope Matrix — 14 Cosmi-Module

**Stand:** 2026-04-18
**Purpose:** Basis fuer Feature-Flag-Registry (S0.6), Sprint-1/2-Planung, Pilot-Segmentierung
**Single Source of Truth:** [docs/ROADMAP.md §5](./ROADMAP.md) — diese Matrix extrahiert operative Details.

> Alle 14 Module haben Mock-Frontend (Stores unter `desktop/src/renderer/src/stores/<modul>.ts`) und werden bis Launch 2026-07-01 mit echtem Backend ausgestattet. `buchhaltung` und `video` sind Completion-Taetigkeiten (Backend teilweise vorhanden), alle anderen sind Neubau.

---

## Uebersicht

| Modul | Sprint | Pilot-Prio | Flag-Key | FE-LOC | Backend-Pkg | Status | Tabellen |
|---|---|---|---|---|---|---|---|
| wiki | S1 | Dienstleister | `modules.wiki` | 1297 | `backend/internal/wiki/` (neu) | Neubau | 3 |
| berichte | S1 | Dienstleister | `modules.berichte` | 1186 | `backend/internal/berichte/` (neu) | Neubau | 2 |
| formulare | S1 | Cross | `modules.formulare` | 2238 | `backend/internal/formulare/` (neu) | Neubau | 2 |
| helpdesk | S1 | Dienstleister | `modules.helpdesk` | 2041 | `backend/internal/helpdesk/` (neu) | Neubau | 4 |
| vertraege | S1 | Dienstleister | `modules.vertraege` | 1899 | `backend/internal/vertraege/` (neu) | Neubau | 3 |
| buchhaltung | S1 | Cross | `modules.buchhaltung` | 1524 | `backend/internal/biz/` (erweitern) | Completion | 4 (vorhanden) |
| video | S1 | Cross | `modules.video` | 459 | `backend/internal/work/` (erweitern) | Completion | 3 (vorhanden) |
| rapporte | S2 | Handwerk | `modules.rapporte` | 2686 | `backend/internal/rapporte/` (neu) | Neubau | 3 |
| schichten | S2 | Handwerk | `modules.schichten` | 1406 | `backend/internal/schichten/` (neu) | Neubau | 3 |
| fuhrpark | S2 | Handwerk | `modules.fuhrpark` | 2299 | `backend/internal/fuhrpark/` (neu) | Neubau | 3 |
| vermietung | S2 | Handwerk | `modules.vermietung` | 2028 | `backend/internal/vermietung/` (neu) | Neubau | 3 |
| inventar | S2 | Cross | `modules.inventar` | 1460 | `backend/internal/inventar/` (neu) | Neubau | 3 |
| einkauf | S2 | Cross | `modules.einkauf` | 1724 | `backend/internal/einkauf/` (neu) | Neubau | 3 |
| produktion | S2 | Handwerk | `modules.produktion` | 1674 | `backend/internal/produktion/` (neu) | Neubau | 3 |

---

## Detailsektionen

### wiki (S1, Dienstleister)

- **Tabellen (geplant):** `wiki_articles`, `wiki_versions`, `wiki_attachments`
- **RPCs (~14):** CreateArticle, UpdateArticle, DeleteArticle, GetArticle, ListArticles, SearchArticles, ListVersions, GetVersion, RestoreVersion, UploadAttachment, ListAttachments, DeleteAttachment, ListCategories, CreateCategory
- **Frontend-Hook (geplant):** `desktop/src/renderer/src/api/hooks/useWiki.ts`
- **Flag-Key:** `modules.wiki`
- **Frontend-Stand:** 1297 LOC in `desktop/src/renderer/src/modules/wiki/` (12 Files), Mock-Store in `stores/wiki.ts`
- **Backend-Stand:** kein Package, keine Migration
- **Besonderheiten:** PostgreSQL Full-Text-Search (FTS) auf `wiki_articles.content`, TipTap-JSON-Format fuer Rich-Content, Share-Links via kurzem Token, Versionierung automatisch bei jedem Save
- **Aktion Sprint 1:** Neues `backend/internal/wiki/`-Package anlegen, Migration fuer 3 Tabellen (inkl. `tenant_id`), Hook ersetzt Mock-Store, FTS-Index `idx_wiki_articles_fts` (GIN)

---

### berichte (S1, Dienstleister)

- **Tabellen (geplant):** `report_definitions`, `report_cache`
- **RPCs (~10):** CreateDefinition, UpdateDefinition, DeleteDefinition, GetDefinition, ListDefinitions, RunReport, GetCachedResult, InvalidateCache, ExportPDF, ExportCSV
- **Frontend-Hook (geplant):** `desktop/src/renderer/src/api/hooks/useBerichte.ts`
- **Flag-Key:** `modules.berichte`
- **Frontend-Stand:** 1186 LOC, 1 File, Mock-Store in `stores/berichte.ts`
- **Backend-Stand:** kein Package, keine Migration
- **Besonderheiten:** `report_definitions.query_config` als JSONB (Aggregations-Konfiguration), `report_cache` mit TTL-Spalte fuer automatischen Verfall, PDF/CSV-Export als Stream
- **Aktion Sprint 1:** Neues `backend/internal/berichte/`-Package, Migration fuer 2 Tabellen (inkl. `tenant_id`), DB-Views fuer Aggregationen

---

### formulare (S1, Cross)

- **Tabellen (geplant):** `form_schemas`, `form_submissions`
- **RPCs (~16):** CreateSchema, UpdateSchema, DeleteSchema, GetSchema, ListSchemas, PublishSchema, UnpublishSchema, SubmitForm, GetSubmission, ListSubmissions, DeleteSubmission, ExportSubmissions, TriggerWebhook, ListWebhooks, AddWebhook, RemoveWebhook
- **Frontend-Hook (geplant):** `desktop/src/renderer/src/api/hooks/useFormulare.ts`
- **Flag-Key:** `modules.formulare`
- **Frontend-Stand:** 2238 LOC, 1 File, Mock-Store in `stores/formulare.ts`
- **Backend-Stand:** kein Package, keine Migration
- **Besonderheiten:** `form_schemas.schema` als JSONB (JSON Schema Draft-7), `form_submissions.data` als JSONB, Webhook-Trigger bei neuer Submission, Public-URL fuer externe Einsendungen
- **Aktion Sprint 1:** Neues `backend/internal/formulare/`-Package, Migration fuer 2 Tabellen (inkl. `tenant_id`), Webhook-Delivery-Queue

---

### helpdesk (S1, Dienstleister)

- **Tabellen (geplant):** `tickets`, `ticket_messages`, `ticket_queues`, `canned_responses`
- **RPCs (~22):** CreateTicket, UpdateTicket, DeleteTicket, GetTicket, ListTickets, AssignTicket, CloseTicket, ReopenTicket, MergeTickets, EscalateTicket, AddMessage, EditMessage, DeleteMessage, ListMessages, CreateQueue, UpdateQueue, DeleteQueue, ListQueues, CreateCannedResponse, UpdateCannedResponse, DeleteCannedResponse, ListCannedResponses
- **Frontend-Hook (geplant):** `desktop/src/renderer/src/api/hooks/useHelpdesk.ts`
- **Flag-Key:** `modules.helpdesk`
- **Frontend-Stand:** 2041 LOC in `desktop/src/renderer/src/modules/helpdesk/` (7 Files), Mock-Store
- **Backend-Stand:** kein Package, keine Migration
- **Besonderheiten:** Ticket-Merge (Duplikat-Erkennung), SLA-Tracking via `due_at`-Spalte, Canned Responses fuer schnelle Antworten, Queue-basiertes Routing, Agenten-Zuweisung
- **Aktion Sprint 1:** Neues `backend/internal/helpdesk/`-Package, Migration fuer 4 Tabellen (inkl. `tenant_id`), SLA-Warning-Trigger

---

### vertraege (S1, Dienstleister)

- **Tabellen (geplant):** `contracts`, `contract_parties`, `contract_reminders`
- **RPCs (~14):** CreateContract, UpdateContract, DeleteContract, GetContract, ListContracts, AddParty, RemoveParty, ListParties, CreateReminder, UpdateReminder, DeleteReminder, ListReminders, UploadDocument, ExportContract
- **Frontend-Hook (geplant):** `desktop/src/renderer/src/api/hooks/useVertraege.ts`
- **Flag-Key:** `modules.vertraege`
- **Frontend-Stand:** 1899 LOC in `desktop/src/renderer/src/modules/vertraege/` (2 Files), Mock-Store
- **Backend-Stand:** kein Package, keine Migration
- **Besonderheiten:** Laufzeit-Engine berechnet `expires_at` und triggert Erinnerungen N Tage vor Ablauf, `contract_parties` linked zu `contacts`/`companies`, Skribble-Placeholder fuer e-Signatur (Phase D), Vertrags-PDF-Upload via MinIO
- **Aktion Sprint 1:** Neues `backend/internal/vertraege/`-Package, Migration fuer 3 Tabellen (inkl. `tenant_id`), Cron-basierter Reminder-Check

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
- **Backend-Stand:** kein Package, keine Migration
- **Besonderheiten:** Foto-Uploads pro Rapport-Zeile via MinIO, GPS-Tag optional (`lat`, `lon` Spalten auf `work_reports`), mehrstufiger Approval-Flow (Submitted → Approved/Rejected), Export als PDF fuer Kundenunterschrift
- **Aktion Sprint 2:** Neues `backend/internal/rapporte/`-Package, Migration fuer 3 Tabellen (inkl. `tenant_id`), MinIO-Attachment-Upload

---

### schichten (S2, Handwerk)

- **Tabellen (geplant):** `shifts`, `shift_assignments`, `shift_templates`
- **RPCs (~16):** CreateShift, UpdateShift, DeleteShift, GetShift, ListShifts, PublishShifts, AssignEmployee, UnassignEmployee, ListAssignments, CreateTemplate, UpdateTemplate, DeleteTemplate, ListTemplates, ApplyTemplate, CheckArbzgCompliance, GetShiftStats
- **Frontend-Hook (geplant):** `desktop/src/renderer/src/api/hooks/useSchichten.ts`
- **Flag-Key:** `modules.schichten`
- **Frontend-Stand:** 1406 LOC, 1 File, Mock-Store
- **Backend-Stand:** kein Package, keine Migration
- **Besonderheiten:** ArbZG §5 Backend-Check (min. 11h Ruhezeit zwischen Schichten — Warning wenn verletzt), Schicht-Templates fuer Wochenmuster, Publish-Flow (Entwurf → Veroeffentlicht → Mitarbeiter sehen)
- **Aktion Sprint 2:** Neues `backend/internal/schichten/`-Package, Migration fuer 3 Tabellen (inkl. `tenant_id`), ArbZG-Pruef-Service

---

### fuhrpark (S2, Handwerk)

- **Tabellen (geplant):** `vehicles`, `vehicle_services`, `vehicle_damages`
- **RPCs (~18):** CreateVehicle, UpdateVehicle, DeleteVehicle, GetVehicle, ListVehicles, ScheduleService, UpdateService, DeleteService, CompleteService, ListServices, ReportDamage, UpdateDamage, ResolveDamage, ListDamages, GetVehicleHistory, CheckTuvDue, ListUpcomingServices, ExportVehicleReport
- **Frontend-Hook (geplant):** `desktop/src/renderer/src/api/hooks/useFuhrpark.ts`
- **Flag-Key:** `modules.fuhrpark`
- **Frontend-Stand:** 2299 LOC in `desktop/src/renderer/src/modules/fuhrpark/` (2 Files), Mock-Store
- **Backend-Stand:** kein Package, keine Migration
- **Besonderheiten:** TUeV-Reminder via `tuev_due_date` + Cron-basierter Check (7 Tage / 1 Tag vor Faelligkeit), Fahrzeug-Service-History vollstaendig protokolliert, Schaden-Meldungen mit Foto-Upload (MinIO), Fuel-Tracking optional
- **Aktion Sprint 2:** Neues `backend/internal/fuhrpark/`-Package, Migration fuer 3 Tabellen (inkl. `tenant_id`), TUeV-Reminder-Cron

---

### vermietung (S2, Handwerk)

- **Tabellen (geplant):** `rental_objects`, `rentals`, `rental_inspections`
- **RPCs (~20):** CreateObject, UpdateObject, DeleteObject, GetObject, ListObjects, CheckAvailability, CreateRental, UpdateRental, DeleteRental, GetRental, ListRentals, StartRental, EndRental, CreateInspection, UpdateInspection, GetInspection, ListInspections, UploadInspectionPhoto, GetRentalCalendar, ExportRentalReport
- **Frontend-Hook (geplant):** `desktop/src/renderer/src/api/hooks/useVermietung.ts`
- **Flag-Key:** `modules.vermietung`
- **Frontend-Stand:** 2028 LOC in `desktop/src/renderer/src/modules/vermietung/` (2 Files), Mock-Store
- **Backend-Stand:** kein Package, keine Migration
- **Besonderheiten:** Zustandsprotokolle (Uebergabe-/Ruecknahme-Inspektion) mit Foto-Uploads, Verfuegbarkeits-Check via `rentals.start_date`/`end_date` Ueberschneidungs-Query, Kalender-View fuer Auslastung
- **Aktion Sprint 2:** Neues `backend/internal/vermietung/`-Package, Migration fuer 3 Tabellen (inkl. `tenant_id`), Verfuegbarkeits-Index `idx_rentals_object_dates`

---

### inventar (S2, Cross)

- **Tabellen (geplant):** `inventory_items`, `inventory_movements`, `stock_warnings`
- **RPCs (~16):** CreateItem, UpdateItem, DeleteItem, GetItem, ListItems, AdjustStock, TransferStock, RecordMovement, ListMovements, GetStockHistory, CreateWarning, UpdateWarning, AcknowledgeWarning, ListWarnings, GetStockReport, ExportInventory
- **Frontend-Hook (geplant):** `desktop/src/renderer/src/api/hooks/useInventar.ts`
- **Flag-Key:** `modules.inventar`
- **Frontend-Stand:** 1460 LOC, 1 File, Mock-Store
- **Backend-Stand:** kein Package, keine Migration
- **Besonderheiten:** Bestands-Alarm via `stock_warnings` — automatisch generiert wenn `inventory_items.quantity <= min_quantity`, doppelte Buchfuehrung via `inventory_movements` (jede Bestandsaenderung trackt Bewegungstyp + User), Barcode-Feld optional
- **Aktion Sprint 2:** Neues `backend/internal/inventar/`-Package, Migration fuer 3 Tabellen (inkl. `tenant_id`), Bestands-Alarm-Trigger

---

### einkauf (S2, Cross)

- **Tabellen (geplant):** `purchase_orders`, `suppliers`, `po_lines`
- **RPCs (~18):** CreateSupplier, UpdateSupplier, DeleteSupplier, GetSupplier, ListSuppliers, CreatePO, UpdatePO, DeletePO, GetPO, ListPOs, AddPOLine, UpdatePOLine, DeletePOLine, ListPOLines, SubmitPO, ReceiveGoods, PartialReceive, ExportPO
- **Frontend-Hook (geplant):** `desktop/src/renderer/src/api/hooks/useEinkauf.ts`
- **Flag-Key:** `modules.einkauf`
- **Frontend-Stand:** 1724 LOC, 1 File, Mock-Store
- **Backend-Stand:** kein Package, keine Migration
- **Besonderheiten:** Wareneingangs-Flow: `ReceiveGoods` bucht automatisch in `inventory_movements` (Cross-Modul-Abhaengigkeit zu `inventar`), PO-Lifecycle Submit → Sent → PartiallyReceived → Received → Closed, Lieferanten-Katalog mit Kontakt-Link zu `contacts`
- **Aktion Sprint 2:** Neues `backend/internal/einkauf/`-Package, Migration fuer 3 Tabellen (inkl. `tenant_id`), Wareneingangs-Event triggert Inventar-Bewegung

---

### produktion (S2, Handwerk)

- **Tabellen (geplant):** `production_orders`, `machine_bookings`, `production_plans`
- **RPCs (~16):** CreateOrder, UpdateOrder, DeleteOrder, GetOrder, ListOrders, StartOrder, CompleteOrder, CancelOrder, CreateMachineBooking, UpdateMachineBooking, DeleteMachineBooking, ListMachineBookings, CreatePlan, UpdatePlan, GetPlan, GetCapacityOverview
- **Frontend-Hook (geplant):** `desktop/src/renderer/src/api/hooks/useProduktion.ts`
- **Flag-Key:** `modules.produktion`
- **Frontend-Stand:** 1674 LOC in `desktop/src/renderer/src/modules/produktion/` (2 Files), Mock-Store
- **Backend-Stand:** kein Package, keine Migration
- **Besonderheiten:** Maschinenbelegungs-Konflikt-Check (Ueberschneidung in `machine_bookings`), Produktionsplan verknuepft Orders mit Zeitfenstern, Kapazitaets-Uebersicht per Maschine/Woche
- **Aktion Sprint 2:** Neues `backend/internal/produktion/`-Package, Migration fuer 3 Tabellen (inkl. `tenant_id`), Belegungs-Konflikt-Pruefung

---

## Naechste Schritte

1. **S0.6 (Feature-Flag-Registry):** Nutzt alle 14 Flag-Keys oben plus `plugins.wasm=false` (Default OFF). Registry in `backend/internal/featureflag/registry.go`.
2. **Sprint 1 (2026-04-28 bis 2026-05-11):** S1-Module implementieren (wiki, berichte, formulare, helpdesk, vertraege, buchhaltung-Completion, video-Completion).
3. **Sprint 2 (2026-05-12 bis 2026-05-25):** S2-Module (rapporte, schichten, fuhrpark, vermietung, inventar, einkauf, produktion).
4. **Multi-Tenancy:** Alle neuen Modul-Tabellen brauchen `tenant_id UUID NOT NULL` von Anfang an — Option-B-Full-Retrofit (siehe `.knowledge/` project_multi_tenancy).
5. **Feature-Flag-Default:** Alle Modul-Flags starten OFF. Pilot-Aktivierung pro Pilot-Deployment via Env-Var.

## Aenderungshistorie

- 2026-04-18: Initial — Sprint-0 Task S0.9.

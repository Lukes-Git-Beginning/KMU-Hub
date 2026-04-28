# Sprint 2 Welle 2 — Known Issues & TODOs

## fuhrpark

### TUEV Notification Delivery (TODO Sprint 3)

**Status:** Worker is live and scans vehicles, but actual notification delivery is a stub.

**Problem:** The `notifications` table exists (`backend/migrations/000021_create_notifications.up.sql`)
but requires `user_id UUID NOT NULL` — a specific user to notify. The TUEV cron worker operates
at the vehicle/tenant level and does not know which user(s) to notify.

**Current behavior:** The worker logs "fuhrpark TUEV reminder triggered (delivery stub)" and
stamps `tuev_reminder_sent_at` on the vehicle row. No notification row is inserted.

**Required Sprint 3 wiring:**
1. Add a `notification_recipients` or `tenant_admins` query to find relevant users for a tenant.
2. Call the `notification` gRPC service (`NotificationService.CreateNotification`) for each recipient.
3. Or: add a tenant-level notification channel (e.g. `tenant_notifications` table without user_id).

**Idempotency:** Already guarded — `tuev_reminder_sent_at` is stamped on delivery attempt,
and the worker skips vehicles where the stamp is < 23 hours old.

### Photo Upload (MinIO Stub)

**Status:** `photo_keys` field on `vehicle_damages` stores MinIO object keys as TEXT[]. The upload
itself (multipart → MinIO) must be handled by the existing file-upload handler in the gateway
(`POST /api/v1/files/upload`). The `photo_keys` are passed in from the client after upload.

No additional work needed here — pattern matches `helpdesk` attachments.

### assigned_driver_id (Sprint 3 team-wiring)

`vehicles.assigned_driver_id` is a nullable UUID stub. No FK constraint (avoids circular dep with
team/user tables). Sprint 3: add FK to `users.id` and wire to team module.

---

## Cross-Module Tech-Debt — JWT Claim Extraction (P0 Sicherheit, vor Pilot-1)

**Status:** Hardcoded Placeholder-TenantID in 7 Gateway-Route-Files — Welle-1-Altlast, von Welle-2A-Subagents wiederholt. Cross-Tenant-Isolation auf HTTP-Ebene **funktioniert nicht**.

**Betroffene Files:**
- `backend/internal/gateway/route_inventar.go` (Welle 1) — Comment auf Line 38: "tenantID is a temporary placeholder until JWT claim extraction is implemented"
- `backend/internal/gateway/route_einkauf.go` (Welle 1)
- `backend/internal/gateway/route_produktion.go` (Welle 1)
- `backend/internal/gateway/route_rapporte.go` (Welle 2A) — Line 38
- `backend/internal/gateway/route_schichten.go` (Welle 2A) — Line 41
- `backend/internal/gateway/route_fuhrpark.go` (Welle 2A) — Line 37
- `backend/internal/gateway/route_vermietung.go` (Welle 2A) — Line 42

Jede Datei hat `const <modul>PlaceholderTenantID = "00000000-0000-0000-0000-000000000001"` und nutzt diesen Wert in jedem gRPC-Aufruf. Die echte Tenant-ID aus dem JWT-Auth-Context wird nie extrahiert.

**Konsequenz:** Jeder authentifizierte User aller Tenants liest und schreibt Daten unter Tenant-ID `...000001`. Multi-Tenant-Isolation ist effektiv aus.

**Fix (eigene Sprint-2-Task):** JWT-Claim-Extraction-Refactor fuer alle 7 Module gemeinsam. Pattern aus existierenden Routes uebernehmen (z.B. `route_crm_contacts.go`, `route_chat.go`) — entweder via `middleware.TenantIDFromContext(r.Context())` oder analoge Helper-Funktion.

**Aufwand:** ~0.5 Tag (mechanisches Pattern-Anwenden in 7 Files × ~15-20 Stellen pro File). Keine API-Aenderungen.

**Pflicht-Tests:** 2-Tenant-Integration-Test pro Modul (User aus Tenant A liest, sieht KEINE Daten von Tenant B).

**Prioritaet:** Vor jedem Pilot-Deployment. **Blocker fuer Gate S2** (Tenant-Isolation-Test).

---

## Welle 2C Bugfix-Sweep (2026-04-28) — abgeschlossen

23 von 27 Findings gefixt im Commit `a4d189e`. Restliche 4 sind Cross-Module Tech-Debt (siehe oben).

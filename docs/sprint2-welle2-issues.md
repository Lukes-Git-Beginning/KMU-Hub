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

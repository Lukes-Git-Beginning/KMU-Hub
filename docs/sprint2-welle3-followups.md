# Sprint 2 Welle 3.5 — Follow-Ups (P2/P3)

> **Closure 2026-05-09:** Pre-Sprint-4-Cleanup hat alle hier gelisteten Items abgeklaert (entweder gefixt, integral in spaeteren Wellen mitgeschlossen, oder explizit deferred). Details siehe MEMORY `project_followup_cleanup_20260509.md`.

> Findings aus dem Welle-3.5-Bugfix-Sweep, die ausserhalb des konsolidierten Fix-Commits bleiben. P0/P1 sind im Fix-Commit selbst geschlossen — siehe `~/.claude/projects/.../memory/project_sprint2_welle3_5_findings.md` fuer die volle Aufstellung.

---

## P2 — UX / DX

### useVideo: globale Recordings-Invalidierung
- **Datei:** `desktop/src/renderer/src/api/hooks/useVideo.ts:110-117`
- **Status (2026-05-09): Done** — `useConfirmInitiatorConsent` invalidiert `['recordings', recordingId]` UND `['recordings']` global (Z.117-118). Vermutlich integral in Welle-4-Frontend-Sweep gefixt.

### Recording-Service `service_test.go` — Test-Aussagekraft
- **Datei:** `backend/internal/work/recording/service_test.go:801-846`
- **Status (2026-05-09): Partial done — verifiziert** — Test prueft Sentinel-Existenz UND echten `svc.ConfirmInitiatorConsent(...)`-Aufruf mit GetPreConsentStatus-Roundtrip. Service-Layer-Defense-in-Depth (echter `svc.StartRecording`-Pre-Consent-Check) ist per aktueller Architektur Gateway-Verantwortung (siehe service.go:87 Doc-Note); ein Service-Layer-Pre-Consent-Gate als zusaetzliche Verteidigung wurde auf Sprint 4 als Hardening-Item geschoben.

### useOnlineStatus: Drain → Invalidate-Reihenfolge
- **Datei:** `desktop/src/renderer/src/hooks/useOnlineStatus.ts:38-43`
- **Status (2026-05-09): Done** (Pre-Sprint-4-Cleanup, commit `c790c4a`) — `drain(...).finally(() => queryClient.invalidateQueries())` sequenziert.

### OfflineBanner: Dead-Letter-Recovery
- **Datei:** `desktop/src/renderer/src/components/ui/OfflineBanner.tsx`
- **Status (2026-05-09): Done** — `retryDeadLetter(idempotencyKey)` in `offline-queue.ts:231` plus per-Item `Erneut senden`-Button im Banner (Z.166-187). Vermutlich integral in Welle-4-Frontend-Sweep gefixt.

### TeamInbox-Modell ohne `TenantID`
- **Datei:** `backend/internal/models/inbox.go:42`
- **Status (2026-05-09): Done** — `TeamInbox.TenantID uuid.UUID` ist gesetzt (Z.43). Vermutlich integral in Welle-4-Backend-Sweep gefixt.

### chat/message Cursor-Lookup ohne tenant_id
- **Datei:** `backend/internal/chat/message/postgres_repository.go:68-89`
- **Status (2026-05-09): Done** — `SELECT created_at FROM messages WHERE id = $1 AND tenant_id = $2` plus expliziter Doc-Comment "scope cursor lookup to tenant to prevent cross-tenant time leaks" (Z.66). Vermutlich integral in Welle-4-Backend-Sweep gefixt.

### tenant_isolation_test.go-Coverage
- **Datei:** `backend/internal/gateway/tenant_isolation_test.go`
- **Status (2026-05-09): Partial done** — 12 neue Welle-4B-Sub-Tests + 4 echte DB-Backed Tests aus Sprint 3 Welle 1A bereits existent. Pre-Sprint-4-Cleanup hat zusaetzlich 4 Welle-2-Module ergaenzt: schichten/fuhrpark/einkauf/produktion mit je 4 Tests (commit `39780a9`). Vier weitere DB-Backed Tests fuer Pipeline_Stages/Automations/AuditLog/Dialer_Campaigns auf **Sprint 5** verschoben (Stretch nicht ausgefuehrt).

### video_grpc.go::UpdatePresenceConfig mit `uuid.Nil`
- **Datei:** `backend/internal/server/video_grpc.go:1144-1148`
- **Status (2026-05-09): Done** — `tenantID, tenantErr := middleware.GetTenantID(ctx)` plus `Unauthenticated`-Return bei Fehler. Vermutlich integral in Welle-4-Backend-Sweep gefixt.

---

## P3 — Style / Polish

### GetRecordingConsent — leere participantIDs
- **Datei:** `backend/internal/server/video_grpc.go:307` plus `backend/internal/work/recording/service.go:246`
- **Status (2026-05-09): Done** (Pre-Sprint-4-Cleanup, commit `c790c4a`) — Service short-circuited `len(participantIDs) == 0` zu `allResponded=false` plus Doc-Comment der die nil-Semantik erklaert. gRPC-Handler dokumentiert warum nil sicher ist.

### Migration 000107 — `responded_at` NULLABLE vs. NOT NULL
- **Datei:** `backend/migrations/000107_recordings_pre_consent_audit.up.sql`
- **Status (2026-05-09): Done** (Pre-Sprint-4-Cleanup, commit `599ebb1`) — Migration `000116_recording_consents_responded_at_not_null` setzt NOT NULL nach idempotentem Backfill.

### Tote Zeile in idempotency.go — Erledigt in Welle-3.5-Fix.

### Migration 000106 down.sql — Erledigt in Welle-3.5-Fix.

### activity AddTags — Loop statt Batch
- **Datei:** `backend/internal/crm/activity/postgres_repository.go:259-272`
- **Status (2026-05-09): Done** — Z.265 `INSERT INTO activity_tags ... SELECT $1, unnest($2::uuid[]), (SELECT tenant_id FROM activities WHERE id = $1)` plus expliziter `P3-3: Single batch-INSERT using unnest — N roundtrips → 1`-Comment. Integral in Welle-4-Sweep gefixt.

---

## Hinweise zur Welle-4-Planung (historisch)

- **Migration-Slot:** Naechste freie Nummer ist `000109` (000108 war Welle-3.5 idempotency_keys Composite-PK). **Update 2026-05-09:** Migrations-Head ist jetzt **000116** (000109-000113 Welle 4B, 000114-000115 Sprint 3, 000116 Pre-Sprint-4-Cleanup).
- **15 nicht-gewireten Top-20-Repos** — alle in Welle 4B (000109-000113) gewired worden.
- **Idempotency HardMode:** Switch durchgefuehrt in Welle 4B (Default WarnMode in Prod, Dev-Default Hard via docker-compose).
- **Server-Deploy-Drift:** abgeschlossen in Sprint 3 Welle 1 (Marathon-Deploy 81 → 115).

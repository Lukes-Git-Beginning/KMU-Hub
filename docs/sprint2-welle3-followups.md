# Sprint 2 Welle 3.5 — Follow-Ups (P2/P3)

> Findings aus dem Welle-3.5-Bugfix-Sweep, die ausserhalb des konsolidierten Fix-Commits bleiben. P0/P1 sind im Fix-Commit selbst geschlossen — siehe `~/.claude/projects/.../memory/project_sprint2_welle3_5_findings.md` fuer die volle Aufstellung.
>
> Pattern analog `docs/sprint2-welle2-issues.md`: Issue benannt, Status, Required Action, Sprint-Ziel.

---

## P2 — UX / DX

### useVideo: globale Recordings-Invalidierung

**Datei:** `desktop/src/renderer/src/api/hooks/useVideo.ts:110-117`

**Status:** `useConfirmInitiatorConsent` invalidiert nur `['recordings', recordingId]`, nicht `['recordings']` global. Status-Poll kann bis zum naechsten Intervall (3 s) stale sein.

**Required action (Sprint 3):** Zusaetzlicher `queryClient.invalidateQueries({ queryKey: ['recordings'] })` im `onSuccess`-Handler.

**Impact:** Nur kosmetisch — UI aktualisiert sich um eine Poll-Periode verzoegert. Kein Datenverlust.

### Recording-Service `service_test.go` — Test-Aussagekraft

**Datei:** `backend/internal/work/recording/service_test.go:801-808`

**Status:** `TestStartRecording_RequiresPreConsent` testet nur die Existenz und Stringinhalt von `ErrPreConsentMissing`, nicht die tatsaechliche Service-Layer-Pruefung.

**Required action (Sprint 3):** Echten Service-Aufruf `StartRecording` mit Mock-Repository, das `pre_recording_consent_at IS NULL` liefert; assert auf `errors.Is(err, ErrPreConsentMissing)` und `mockEgressManager.StartRoomCompositeEgressCalls() == 0`.

### useOnlineStatus: Drain → Invalidate-Reihenfolge

**Datei:** `desktop/src/renderer/src/hooks/useOnlineStatus.ts:38-43`

**Status:** `queryClient.invalidateQueries()` und `drain()` laufen parallel. Drain-Responses triggern keine eigenen Invalidations, also kann frisch ge-drainete Daten erst beim naechsten Polling sichtbar werden.

**Required action (Sprint 3):** `drain(...).then(() => queryClient.invalidateQueries())` sequenzieren.

### OfflineBanner: Dead-Letter-Recovery

**Datei:** `desktop/src/renderer/src/components/ui/OfflineBanner.tsx`

**Status:** Dead-Letter-Counter wird angezeigt, aber kein Button zum manuellen Retry oder Loeschen einzelner Items — nur `clear()` (loescht alles).

**Required action (Sprint 3):** `retryDeadLetter(id)`-Aktion in `offline-queue.ts` ergaenzen + UI-Button pro Item. Optional: Auto-Retry ein einzelnes Mal mit Reset von `retryCount`.

### TeamInbox-Modell ohne `TenantID`

**Datei:** `backend/internal/models/inbox.go:42`

**Status:** `TeamInbox`-Struct hat kein `TenantID`-Feld, obwohl `InboxMessage` es bekommen hat (Welle 3 Migration 000106). Queries in `route_inbox.go` waeren cross-tenant.

**Required action (Welle 4):** `team_inboxes` zur Top-30+-Tabellen-Liste hinzufuegen. `TeamInbox.TenantID` ergaenzen, Repo-Filter umstellen.

### chat/message Cursor-Lookup ohne tenant_id

**Datei:** `backend/internal/chat/message/postgres_repository.go:68-89`

**Status:** Cursor-Lookup `SELECT created_at FROM messages WHERE id = $1` ohne tenant_id-Filter (TOCTOU-Leak: Cursor-Zeit aus fremdem Tenant verwendbar). Im Welle-3.5-Fix-Commit nur die Haupt-Queries gefixt; Cursor-Helper wurde uebersehen.

**Required action (Welle 4):** `id = $1 AND tenant_id = $2` im Cursor-Lookup.

### tenant_isolation_test.go-Coverage

**Datei:** `backend/internal/gateway/tenant_isolation_test.go`

**Status:** Welle-3.5-Fix hat 4 neue Tests fuer Recording-Initiator-Consent ergaenzt. Es fehlen weiterhin Tests fuer Pipeline-Stages, Channels (List), Projects, CalendarEvents, TimeEntries, Automations, SavedFilters, CustomFields, EmailMessages, InboxMessages, Dialer-CampaignList, AuditLog, Recordings (Top-Level-Listen).

**Required action (Welle 4):** 12 neue Sub-Tests im selben Pattern (`No-Tenant`, `Empty-Tid`, `Valid-Tid`).

### video_grpc.go::UpdatePresenceConfig mit `uuid.Nil`

**Datei:** `backend/internal/server/video_grpc.go:1059`

**Status:** Methode uebergibt `uuid.Nil` als tenantID an `presenceService.UpdateConfig`. Falls Service den Wert nutzt, schreibt Operation ohne Tenant-Kontext.

**Required action (Welle 4):** Aus Context lesen via `middleware.GetTenantID`.

---

## P3 — Style / Polish

### GetRecordingConsent — leere participantIDs

**Datei:** `backend/internal/server/video_grpc.go:307`

**Status:** `nil` als participantIDs uebergeben. `CountPendingConsents` mit leerer ID-Liste liefert COUNT=0 (alle responded), was semantisch inkonsistent ist.

**Required action:** Doku-Kommentar oder explizite Validation der Participant-IDs.

### Migration 000107 — `responded_at` NULLABLE vs. NOT NULL

**Datei:** `backend/migrations/000107_recordings_pre_consent_audit.up.sql`

**Status:** Spalte `responded_at` als NULLABLE hinzugefuegt, business logic schreibt aber immer `time.Now()` (de-facto NOT NULL).

**Required action (Welle 4):** Migration ergaenzen die `responded_at SET NOT NULL` setzt nach Backfill aller bestehenden Rows.

### Tote Zeile in idempotency.go

**Datei:** `backend/internal/middleware/idempotency.go:240`

**Status:** Wurde im Welle-3.5-Fix entfernt. Erledigt.

### Migration 000106 down.sql — fehlender FK-Kommentar

**Datei:** `backend/migrations/000106_tenant_id_retrofit_phase1.down.sql`

**Status:** Im Welle-3.5-Fix wurde Doku-Kommentar ergaenzt, dass FK `tenant_id → tenants(id)` absichtlich fehlt (Backfill-Kompatibilitaet). Erledigt.

### activity AddTags — Loop statt Batch

**Datei:** `backend/internal/crm/activity/postgres_repository.go:253-264`

**Status:** Einzel-INSERTs in Loop statt unnest-Batch (wie `deal/postgres_repository.go:237-247`).

**Required action:** Refactor auf unnest-Batch fuer Konsistenz.

---

## Hinweise zur Welle-4-Planung

- **Migration-Slot:** Naechste freie Nummer ist `000109` (000108 war Welle-3.5 idempotency_keys Composite-PK).
- **15 nicht-gewireten Top-20-Repos** (projects, channels-Liste, calendar_events, email_messages, inbox_messages, deal_stage_history, pipeline_stages-Listen, saved_filters, custom_field_definitions, automations, document_files, recordings-Top-Level, dialer_call_sessions, audit_log, time_entries) gehen in Welle 4.
- **Idempotency HardMode:** Switch von WarnMode auf HardMode (fehlende Idempotency-Key → 400) erst NACH vollstaendigem Frontend-Rollout. Frontend setzt Key bereits via `client.ts`-Wrapper, also Welle 4 kann den Switch durchfuehren.
- **Server-Deploy-Drift:** Migration-Head Prod ist 81, lokal ist nun 108. 27 Migrations Drift fuer Pilot-1 — eigener Sprint-Task vor Welle 4.

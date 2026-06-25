# Meeting-Parität Wave 5 — Restlücken (Followup-Tickets)

**Stand:** 2026-06-25 (Session federated-clarke). **Wave 5 ist vollständig gebaut & live.** Diese Datei hält
die *einzigen* echten Restlücken fest, die bei der Code-Re-Verifikation auffielen. Keine davon ist
launch-kritisch. Jede ist self-contained als isolierte Welle ziehbar.

> Kontext: Der ursprüngliche „Wave 5 bauen"-Task war ein No-op — Lock-DTO, Recording-Stop für Host/Co-Host,
> Playback und Retention waren bereits durch eine frühere Session (06-23, BE `eae99273`) implementiert und
> deployt. Beleg u.a. Header-Kommentar „Wave 5 / B4" in `RecordingPlayerModal.tsx`. Memory-Note
> `project_meeting_parity_20260623.md` enthält die volle Wave-5-Historie.

---

## Ticket 1 — Retention policy-driven machen (Backend, mittel)

**Problem:** Recording-Retention ist hartkodiert auf 30 Tage. Die generische, admin-konfigurierbare
`retention_policies`-Tabelle (Migr.233) wird vom Recording-Cleanup **ignoriert**.

**IST:**
- `backend/internal/work/recording/service.go:18` — `const RetentionDays = 30`
- `service.go:141` — `retentionExpires := now.Add(RetentionDays * 24 * time.Hour)` (bei CreateRecording gesetzt)
- `backend/cmd/work/main.go:256-286` — 24h-Cleanup-Goroutine ruft `CleanupExpiredRecordings`
- `backend/internal/work/recording/postgres_repository.go:225` — `WHERE retention_expires_at < $1 AND status='completed'`
- `retention_policies` (Migr.233): `tenant_id`, `resource_type VARCHAR(64)`, `retention_days INT CHECK(>0)`,
  `action VARCHAR(16) CHECK IN ('delete','anonymize')`, `enabled BOOLEAN`, UNIQUE`(tenant_id,resource_type)`.
  Aktuell nur an `backend/internal/server/security_grpc.go` (CRUD) angebunden, **kein Sweeper liest sie.**

**Fix-Muster:** Beim Setzen von `retention_expires_at` (bzw. im Cleanup) die Policy für
`resource_type='recording'` des Tenants konsultieren; Fallback auf `RetentionDays=30`, wenn keine/`enabled=false`.
Tenant-Loop im Sweeper via `database.WithSystemContext(ctx)` (RLS-Bypass, Muster aus `startMeetingSweeper`,
`cmd/work/main.go:329`). `action='anonymize'` für Recordings vorerst als „wie delete" behandeln oder ablehnen.

**Akzeptanz:** Admin setzt `retention_policies(resource_type='recording', retention_days=7)` → neue/abgelaufene
Recordings werden nach 7 statt 30 Tagen gelöscht; ohne Policy bleibt 30d-Default. Tenant-isoliert.

---

## Ticket 2 — `ListRecordings`-Authz härten (Backend, klein, Security)

**Problem:** Nur die Download-URL ist participant-ACL-gated. Die Listen-Endpoints prüfen keine
Teilnehmerschaft → jeder authentifizierte User kann Recording-**Metadaten** enumerieren (nicht den Inhalt),
sofern er eine `meeting_id`/`call_id` kennt.

**IST:**
- `backend/internal/gateway/route_video.go` — `HandleListRecordings`, `HandleListRecordingsByMeeting`,
  `HandleGetRecordingStatus` ohne Authz-Check (Thin-Proxy).
- Gegenbeispiel mit korrektem ACL: `recording.Service.GetRecordingDownloadURL`
  (`backend/internal/work/recording/service.go:615-679`) — prüft `call_participants` ODER `meeting_attendees`.

**Fix-Muster:** Participant-Check aus `GetRecordingDownloadURL` extrahieren/wiederverwenden und in den
List-/Status-Pfad ziehen (Service-Layer, nicht Handler — thick service). Für Listen evtl. Ergebnis auf
Recordings filtern, an denen der Caller teilnahm.

**Akzeptanz:** Nicht-Teilnehmer bekommt 403 (oder leere Liste) auf `GET /video/meetings/{id}/recordings`;
Teilnehmer/Host sieht seine Recordings. Kein RBAC-Guard/Seed nötig (meeting-scoped).

---

## Ticket 3 — Lock-Live-Propagation (Frontend, klein, UX)

**Problem:** Der Lock-Indikator zeigt server-State, wird aber nicht live aktualisiert. Ändert *ein anderer*
Host den Lock, bleibt der Indikator bei den übrigen Clients stale bis Remount. (Das *Enforcement* ist live —
ein Nicht-Host, der einen gesperrten Raum betreten will, bekommt server-seitig `ErrMeetingLocked`.)

**IST:**
- `desktop/src/renderer/src/api/hooks/useMeetings.ts:49` — `useMeeting` ohne `refetchInterval`; invalidiert nur
  via `useSetMeetingLock.onSuccess` (nur beim *handelnden* Client).
- `desktop/src/renderer/src/features/video/VideoCallView.tsx:1060` — `const isLocked = meeting?.locked ?? false`
- Indikator `MeetingLockIndicator` (`features/video/HostControls.tsx:343`), nur host-sichtbar.

**Fix-Muster:** Lock-Change über den bestehenden LiveKit-Data-Channel broadcasten (wie Chat/Hand/Reactions in
W2) und im Empfänger den `['meetings', meetingId]`-Query invalidieren — sauberer als Polling. Alternativ
minimal: `refetchInterval` auf `useMeeting`, solange ein Call aktiv ist.

**Akzeptanz:** Host A sperrt → Host B sieht den Indikator innerhalb weniger Sekunden ohne Remount. Nur
`transform`/`opacity`-Animation (Motion-Hardrule).

---

## Manuell / user-getrieben — Live-2-Personen-Smoke W1–W5

Nicht durch eine Coding-Session allein fahrbar (braucht 2 Teilnehmer/2 Netze). Tooling vorhanden:
`.planning/videocall-test-runbook.md`, `.planning/create-test-meeting.py`. Blocker M1 (Hetzner-Firewall
UDP 7882) + neuer Electron-Installer-Build für FE-Features — s. Memory-Note „Manuelle Schritte".

**Smoke-Checkliste (Wave 5):**
1. Host sperrt Meeting (Lock-Toggle) → 2. Client kann **nicht** joinen (`ErrMeetingLocked`), Co-Host schon.
2. Recording per Co-Host (≠ Initiator) **starten und stoppen** → kein PermissionDenied.
3. Nach Stop: Aufnahmen-Tab im `MeetingDetailPanel` listet das Recording → `RecordingPlayerModal` spielt via
   Presigned-URL ab, Download funktioniert, Retention-Countdown sichtbar.
4. Consent-Dialog überspringt den Initiator (started_by).

# Backend-Nachtloop — Journal Lauf 17

Append-only. Ein Eintrag je Iteration, **ans Dateiende**, nie einsortieren. Form und Pflichtzeilen
stehen in `ITERATION.md` Schritt 6.

Frühere Läufe liegen vollständig im Archiv:
`archive/lauf-1-2/` (58 Units) · `archive/lauf-3/` (61) · `archive/lauf-4/` (54) ·
`archive/lauf-5/` (41) · `archive/lauf-6/` (46) · `archive/lauf-7/` (71) ·
`archive/lauf-8/` (94) · `archive/lauf-9/` (37) · `archive/lauf-10/` (93) ·
`archive/lauf-11/` (131 Einträge, davon 121 done) · `archive/lauf-12/` (99, davon 41 done) ·
`archive/lauf-13/` (120, davon 21 done — abgestürzt) · `archive/lauf-14/` (110, alle done —
Backlog leergelaufen, gemergt als `cadf4bf4`) · `archive/lauf-15/` (74, alle done — Backlog
leergelaufen, PR #27) · `archive/lauf-16/` (82, alle done — Backlog leergelaufen, PR #28).
Alle inkl. Laufbilanz.

---

## Laufkontext

- **Ausgangspunkt:** Lauf 16 gemergt als `b88d7eac` (PR #28) und deployt (CD 2026-10-25 13:27,
  Kopf 345 clean, 32 Container healthy). `main` = `6001c1a9`. `backend-loop` = `main` plus
  Staging-Commits (Archiv, Entscheidungsrunde, Treiber-Pflege, lokaler E2E-Stack, dieser Lauf).
- **Migrationen:** Repo-Kopf = lokaler DB-Kopf = Produktionskopf = **345**. Nummer immer zur
  Laufzeit ermitteln: `ls backend/migrations | grep -E "^[0-9]{6}" | sort | tail -1`
- **Coverage-Start:** **74,5 %** gesamt (Run 40196109073 auf `b88d7eac`). Paketwerte je Unit.
- **Queue:** 107 Units, roter Faden „Rechte gegen den echten Seed — und jedes Modul einmal
  Ende-zu-Ende“. Rechte-Scans (2), Lecks und Grenzen (15), Reste (5), E2E (12), entschiedene
  Features (34), Coverage (17 + 16 Füllung), Reviews (6). Begründung, Regeln und Sperren im Kopf
  von `BACKLOG.yml`.
- **Neu im Gate:** lokaler E2E-Stack `hooks/e2e-local.sh` (Regel 1 im Backlog-Kopf). Der Treiber
  setzt `GOFLAGS=-p=2` und `GOMAXPROCS=2`. Die Kopfzeilen-Zeit ist die Startzeit aus dem
  Laufkontext, wörtlich.

## Entscheidungsrunde 2026-10-25 (Luke, sieben Runden) — Ergebnis

Je Unit steht die Entscheidung im Feld `decision` von `BACKLOG.yml`. Kurzfassung:

- **Inbox:** sichtbar = Eigentümer, zugewiesen oder Mitglied der Team-Inbox. Danach bekommt member
  die Team-Inbox.
- **Abwesenheiten:** member sieht eine neutrale Projektion, der manager sieht die Art für seine
  Berichtslinie.
- **E-Mail:** Self-Setup mit Dial-Filter und Allowlist per Env. Regeln pro Nutzer, Labels geteilt,
  eigene Vorlagen änderbar, im Kontakt-Mailverlauf sieht member nur die eigenen Mails.
- **Rollenmatrix:**
  - member: Fuhrpark-Verfügbarkeitsroute, Schäden lesen, Inventar-Umbuchung, eigene Schichten,
    zugewiesene Helpdesk-Tickets, eigene Aufgaben löschen, Deals (Team), eigene Wiki-Seiten
    teilen/löschen.
  - manager: Finanz-Kachel, Personalakte und Vollfelder der Berichtslinie.
  - NICHT: Tags anlegen, CRM-Merge, Dokument-Freigaben, Kontakte importieren/löschen.
- **Datenschutz:**
  - E-Mail-Absendername beim Anonymisieren scrubben, Body bleibt als Restrisiko (HGB).
  - Meeting-Notizen/-Chat: DSAR-Export und 2 Jahre Frist.
  - Inspektionsfotos werden mit gelöscht, Lieferanten gelten als HGB-Daten.
  - Fristen: dialer_agent_status_log 12 Monate, call_sessions 24 Monate, public_bookings 24 Monate.
- **Finanzen:** Verzugszinsen streichen, Eingangsrechnung prüfen und warnen, `finance_payments.currency`
  als eigene Spalte, Ad-hoc-Rechnung nur in Tenant-Währung.
- **Schichten/Fuhrpark/Work/HR:**
  - Überlappende Schichten werden mit 409 abgelehnt, ein Tausch ist eine Übergabe.
  - Führerschein-Löschen mit Audit-Event, Privatfahrten maskiert (Fahrtenbuch bleibt lückenlos).
  - Aufgabe ohne Projekt bekommt `completed_at`.
  - Deaktivieren läuft über das Offboarding, dazu ein neuer Reaktivieren-RPC.
- **Sonstiges:**
  - Gruppen-DMs bauen. Egress startet erst nach dem Initiator-Stempel (Prod-Egress ist heute
    nicht konfiguriert).
  - Tokens: Share-Tokens als Hash, Share-Links höchstens 365 Tage, Join-Token nur für Teilnehmer,
    Offboarding widerruft CalDAV.
  - Alarm: jetzt nur die Regel auf `audit_events_dropped_total`, der Backup-Wächter folgt tagsüber.
  - Geister-Routen: team-worktime wird gebaut, der Rest wird im FE ausgeblendet.
  - Rahmenvertrag weich: buchen und warnen.

## Was aus Lauf 16 gelernt wurde — drei Dinge, die dich direkt betreffen

1. **Die Rollenmatrix bildet den Ist-Seed ab, nicht die Absicht.** 000068 gab member
   `dialer:agent:manage`, und die Matrix nannte das „ok“. Prüfe Grants gegen die fachliche
   Absicht (Block S).
2. **Handgebaute Tokens sehen keine Seed-Fehler.** `hr:self` (`resource = 'hr'`) fiel durch
   `resource LIKE '%:%'`, der Scope kam nie ins JWT, und ein fehlender Key gilt als `all`. Rechte
   nur mit echtem Token prüfen (Regel 2), neue Keys dreiteilig.
3. **Der Morgen-PR ist nicht automatisch grün.** Fünf E2E-Flows fielen erst in CI. Seit diesem
   Lauf läuft E2E lokal (Regel 1): Ein roter Flow ist ein Fund für diese Nacht, nicht für morgen.

---

## Iteration 1 — scan-legacy-seed-grants-for-member-and-manager — done — 2026-10-25 16:29
- commit: siehe Commit `docs(loop): scan legacy member and manager grants` (dieser Eintrag)
- gebaut: Scan, keine Code-Änderung. Lokale DB auf Kopf 345 (`schema_migrations` 345, nicht dirty). Abfrage
  `role_permissions` × `roles` (tenant_id IS NULL, member/manager) × `permissions` auf `*:manage|*:delete|*:approve|admin:*`,
  zusätzlich alle groben member-Keys (resource ohne Doppelpunkt) mit Aktion ≠ read/view, weil `*:admin` dieselbe Klasse ist.
  Herkunft per Grep auf den Key-Literal in `backend/migrations/*.up.sql`.
  - **member (11 Grants):**
    - `calendars:delete` own (000129, 000336), `files:delete` own (000018, 000336), `messages:delete` own (000015, 000336),
      `email:mailbox:delete` own (000337): Lauf-16-Entscheidungen, own-Scope. ok.
    - `kommunikation:channel:manage` all (000256): öffnet zusätzlich nur `DELETE /channels/{id}` (Create/Update/Archive/Rolle
      gehen schon über `channels:write`). `chat/channel/service.go` `Delete` verlangt Kanal-Owner, `Update`/`Archive`
      `CanModerate()`. ok.
    - `rapporte:measurement:manage` all (000256): Schreibwege über `rapporteMeasurementScopeKeys` mit `rapporte:report:edit`
      (own) begrenzt, narrower wins (`ownerFilterForScopeAny`). ok. **Aber Lesewege:** `rapporte:measurement:read` all neben
      `rapporte:report:read` own, `HandleListMeasurements`/`HandleGetMeasurement` reichen keine Einschränkung durch → Leck
      innerhalb des Tenants (Fix-Unit).
    - `settings:personal:manage` all (000256): kein Gateway-Guard liest den Key, nur FE-Katalog. ok.
    - `calendars:admin` all (000035): Kalender-Mitglieder, `work/calendar/service.go` `AddMember`/`RemoveMember` prüfen
      Owner oder Kalender-Admin; Holiday-Seed zusätzlich `settings:tenant:manage`. ok.
    - `resources:admin` all (000035): kein Guard liest den Key. ok.
    - `settings:write` all: `PUT /admin/branding` und `PUT /settings/{module}/tenant` nur mit diesem Guard, aber
      `settings_grpc.go` `PutBranding` ("only admins") und `PutTenantSettings` ("only module-leads or admins") prüfen im
      Server; Presence-Config hat `settings:tenant:manage`. ok.
    - `dialer:agent:manage` (000068) ist seit 000344 weg. Bestätigt.
  - **manager (50 Grants):** 43 aus dem 000256-Preset (Supervisor-Rolle per Design: Approve-/Manage-/Delete-Keys der
    Module, Scope all bzw. team bei `crm:deal:delete`, `work:task:delete`, `team:absence:approve` (000335)). Vor 000256:
    `projects:delete`/`tasks:delete` all (000024), `work_labels:delete`/`work_custom_fields:delete` all (000147),
    `booking-pages:delete` all (000136), `dialer:outcomes:manage` all (000068), `rapporte:report:approve` (000100),
    `schichten:swap:approve` (000161). `tasks:delete` all neben `work:task:delete` team: `DeleteTask` prüft im Service die
    Projektmitgliedschaft, Zeiteintrag-/Kommentar-Löschen prüfen den Akteur. Kein `admin:*` bei manager oder member.
    Kein Fund, der über die Supervisor-Absicht hinausgeht.
  - **Nebenbefund Totalausfall:** member hält `tasks:write`, nicht `tasks:delete` → `DELETE /task-comments/{id}` und
    `DELETE /time-entries/{id}` (hart `RequirePermission("tasks","delete")`, exakter Vergleich `middleware/rbac.go:38`)
    geben jedem member 403, obwohl das FE den Löschknopf über `isOwn` zeigt und der Service den Akteur prüft.
- gate: n.a. (Scan, kein Code) | backlog-check --preflight ok
- coverage: n.a. (kein Coverage-Ziel)
- mutations-probe: n.a.
- verify vorgaenger: erste Iteration des Laufs, kein Vorgänger-Commit aus dem Loop (Kopf = Staging `65ebf725`)
- neue-units: `fix-work-member-own-comment-and-time-entry-delete` (Totalausfall) und
  `fix-rapporte-measurement-read-follows-report-scope` (Leck im Tenant) — beide VOR die erste todo-Unit gesetzt
- desktop-aufrufer: `DELETE /task-comments/{id}` -> `api/hooks/useTaskComments.ts:121` -> bricht für member
  (fix-work-member-own-comment-and-time-entry-delete); `DELETE /time-entries/{id}` -> `api/hooks/useTimeEntries.ts:196`
  -> bricht für member (dieselbe Unit); `GET /rapporte/measurements` -> nur `types.ts`, kein Aufrufer
- offen: Herkunft ist dateigenau per Key-Literal; Grants, die ältere Migrationen per Muster (`p.action IN (...)`) vergeben
  haben, tragen als Herkunft die erste Migration mit Literal. Bewertet wurde der Ist-Seed der lokalen DB (= Repo-Kopf 345).

## Iteration 2 — fix-work-member-own-comment-and-time-entry-delete — done — 2026-10-25 16:34
- commit: siehe Commit `fix(work): let writers delete their own task comments and time entries` (dieser Eintrag)
- gebaut: `DELETE /task-comments/{id}` und `DELETE /time-entries/{id}` laufen jetzt über `ownDelete =
  RequirePermissionAny({tasks,delete},{tasks,write})` (`gateway/route_work.go`), additiv, kein neuer Key, kein Seed.
  Vorab geprüft: `comment.Service.Delete` (Autor oder `IsAdmin`) und `timeentry.Service.DeleteEntry` (Eigentümer) vergleichen
  den Akteur; `IsAdmin` kommt aus `middleware.IsAdmin(ctx)` (Token-Rollen), nicht aus dem Request-Body.
  Neuer DB-Test `route_work_own_delete_db_test.go`: echter Seed → `ResolveTokenPermissions` → echtes Token → echte
  `middleware.Auth` + `WorkRoutes.RegisterRoutes` → Fake-gRPC-Work-Server. member/manager/admin 200 (RPC erreicht),
  readonly/extern 403 (RPC nicht erreicht); `IsAdmin` im RPC false für member/manager, true für admin.
- gate: build ok | vet ok | lint ok | test ok (gateway voll, inkl. TestOpenAPIRouteDrift; DB-Test lief real, 5 Subtests, 0 Skips) | migration n.a. | rls-smoke n.a.
- coverage: n.a. (kein Coverage-Ziel)
- mutations-probe: `ownDelete` zurück auf `RequirePermission("tasks","delete")` → member-Subtest rot (403 statt 200, RPC nicht
  erreicht); zurückgedreht, Diff sauber.
- verify vorgaenger: sauber (Iteration 1 war reiner Docs-/Backlog-Commit `b1ec2498`, kein Code)
- neue-units: keine
- desktop-aufrufer: `DELETE /task-comments/{id}` -> `api/hooks/useTaskComments.ts:121` -> passt (Antwort unverändert, member kommt jetzt durch);
  `DELETE /time-entries/{id}` -> `api/hooks/useTimeEntries.ts:196` -> passt
- offen: Wirkt ohne Re-Login, weil member `tasks:write` schon im Alt-Token hat. Das Fremdlöschen bleibt Service-Sache:
  `DeleteTimeEntry` im gRPC-Server übergibt `isAdmin=false` fest, ein Admin kann also keine fremden Zeiteinträge löschen
  (Verhalten unverändert, Produktfrage, keine Unit).

## Iteration 3 — fix-rapporte-measurement-read-follows-report-scope — done — 2026-10-25 16:40
- commit: siehe Commit `fix(rapporte): narrow measurement reads to the caller's own reports` (dieser Eintrag)
- gebaut: `restrict_to_author` in `GetMeasurementRequest` (3) und `ListMeasurementsRequest` (5), Proto regeneriert. Gateway
  setzt es über `rapporteMeasurementReadScopeKeys` (`measurement:read` + `report:read`, narrower wins). gRPC `GetMeasurement`
  prüft bei restrict den Rapport über `authorizeMeasurement` (fremder Rapport → 404 als Aufmaß nicht gefunden),
  `ListMeasurements` reicht den Aufrufer als `authorID` ans Repository, das per Subselect auf `work_reports.author_id`
  filtert. Aufmaße ohne Rapport bleiben sichtbar — gleiche Regel wie die Schreibwege (`authorizeMeasurement`-Kommentar).
  Neuer DB-Test `route_rapporte_measurement_read_db_test.go`: echter Seed → `ResolveTokenPermissions` → echte Auth +
  Routen → echter rapporte-gRPC-Server (TenantInbound-Interceptor, Postgres-Repo).
- gate: build ok | vet ok | lint ok | test ok (rapporte 95 Tests 0 Skips, server Rapporte-Tests, gateway voll inkl.
  TestOpenAPIRouteDrift; neuer DB-Test lief real, 0 Skips) | migration n.a. | rls-smoke n.a. (keine Tabelle/Policy) | e2e n.a. (kein Flow prüft Aufmaße)
- coverage: n.a. (kein Coverage-Ziel)
- mutations-probe: (1) Read-Keys zurück auf nur `rapporte:measurement:read` (= Vorzustand) → member-Subtest rot: Liste
  enthält das fremde Aufmaß, `?report_id=<fremd>` liefert 1 statt 0, GET fremd 200 statt 404 — damit ist das Leck auch
  reproduziert. (2) Autor-Prüfung in gRPC `GetMeasurement` neutralisiert → GET fremd 200 statt 404. Beide zurückgedreht, Diff sauber.
- verify vorgaenger: sauber (`bea77419`: Guard additiv per `RequirePermissionAny`, kein neuer Key, kein Proto, keine Route)
- neue-units: keine
- desktop-aufrufer: `GET /rapporte/measurements[/{id}]` -> nur `api/types.ts`, kein Client/Hook -> passt (Antwortform unverändert)
- offen: Aufmaße ohne Rapport (`report_id` NULL, auch nach `ON DELETE SET NULL` eines gelöschten Rapports) bleiben für
  member les- und schreibbar — bewusst konsistent mit der bestehenden Schreibregel, aber eine Produktfrage, falls verwaiste
  Aufmaße fremder Rapporte privat sein sollen. Kein Re-Login nötig (kein neuer Key).

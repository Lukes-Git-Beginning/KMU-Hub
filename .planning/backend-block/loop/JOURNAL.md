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

## Iteration 4 — scan-coarse-keys-with-non-all-scope — done — 2026-10-25 16:49
- commit: siehe Commit `docs(loop): scan coarse keys with a narrowed scope` (dieser Eintrag)
- gebaut: Scan, kein Code. Preset-Grants (`tenant_id IS NULL`, lokale DB = Kopf 345) mit grobem Key und Scope != all:
  genau vier Keys, alle `own`, je member und manager — `calendars:delete`, `files:delete`, `messages:delete`, `hr:self`.
  Kein grober Key mit `team`. Liste Key -> Scope -> Handler -> vorher/nachher:
  - `calendars:delete` own -> `DELETE /calendar/calendars/{id}`, `DELETE /calendar/events/{id}`, `DELETE /calendar/categories/{id}`
    (Any mit `kalender:category:manage`). Scope wird nirgends gelesen (kein `PermissionScope`-Leser), Guard prüft nur Präsenz
    -> vorher = nachher. Grenze sitzt im Service: Kalender `OwnerID == actor`, Event `CreatedBy == actor`, Kategorie
    `DELETE ... WHERE user_id = actor`. Kein Fund.
  - `files:delete` own -> `DELETE /chat/files/{id}`; `messages:delete` own -> `DELETE /chat/messages/{id}`. Scope ungelesen,
    vorher = nachher. Service: Uploader/Absender oder Kanal-Admin/Owner, Akteur aus dem Token (`UserId` vom Gateway). Kein Fund.
  - `hr:self` own -> 24 Routen unter `/hr/leave` und `/hr/time` (selfRead/selfWrite) plus `restrictToOwnEmployee`/
    `ownerFilterForScopeAny`. 21 Handler nehmen die ID ohnehin aus dem Token; Liste/Detail Urlaubsanträge, Saldo je Nutzer
    und Zeiteinträge-Liste werden seit `b00ed131` auf den Aufrufer verengt (vorher: `all` für jeden member — das Leck ist zu).
    Klasse (b) dabei: Manager hält nur `hr:self` own (kein `hr:read`) und verliert damit zwei Team-Sichten; die
    Urlaubsfreigabe-Liste ist über `callerMayDecideLeave` bereits ausgenommen.
  Router-Probe (temporärer DB-Test, nicht committet): echter Seed -> `ResolveTokenPermissions` -> echtes Token -> echte
  `middleware.Auth` + `HRRoutes` -> Fake-HR-gRPC. Ergebnis:
  `GET /hr/time/entries?status=correction_pending`: member 200 gepinnt auf sich (richtig), **manager 200 gepinnt auf sich**,
  **hr_admin 403**, admin 200 ungefiltert. `GET /hr/leave/balance/{fremd}`: member 403 (richtig), **manager 403**,
  **hr_admin 403**, admin 200.
- gate: n.a. (Scan, kein Code) | backlog-check --preflight ok | Probe lief real gegen lokale DB (kmuhub_app), 0 Skips
- coverage: n.a. (kein Coverage-Ziel)
- mutations-probe: n.a.
- verify vorgaenger: sauber (`a69fc2e6`: Handler über `client.GetMeasurement/ListMeasurements`, `.proto` mit `.pb.go`
  regeneriert, kein neuer Guard/Seed, keine neue Route, Antwortform unverändert)
- neue-units: `fix-hr-correction-list-for-approvers` (Totalausfall: Korrektur-Tab für jeden Manager nur eigene Einträge,
  hr_admin 403) und `fix-hr-employee-leave-balance-for-team-viewers` (Saldo im Team-Profil für Manager/hr_admin 403) —
  beide VOR die erste todo-Unit gesetzt, in dieser Reihenfolge
- desktop-aufrufer: `GET /hr/time/entries` -> `modules/team/TimeCorrectionPanel.tsx:33` (über `useWorkTimeEntries`) -> bricht
  für manager/hr_admin (fix-hr-correction-list-for-approvers); `GET /hr/leave/balance/{userId}` ->
  `modules/team/MemberProfileContent.tsx:87` -> bricht für manager/hr_admin (fix-hr-employee-leave-balance-for-team-viewers);
  `GET /hr/leave/requests/{id}` -> `useLeaveRequest` ohne Verbraucher -> n.a.
- offen: hr_admin hält überhaupt keinen groben `hr:*`-Key; jede Route hinter `selfRead`/`selfWrite` ist für einen reinen
  hr_admin 403 (auch eigenes Stempeln) — ob hr_admin je allein vergeben wird oder immer neben member, ist eine Produktfrage;
  die zwei Fix-Units decken nur die beiden Team-Sichten ab. Tenant-eigene Rollen und Per-User-Overrides mit groben Keys
  wurden nicht gescannt (Unit-Scope = Presets).

## Iteration 5 — fix-hr-correction-list-for-approvers — done — 2026-10-25 16:56
- commit: siehe Commit `fix(hr): list pending corrections at the approver's reach` (dieser Eintrag)
- gebaut: `GET /hr/time/entries?status=correction_pending` ohne `employee_id` folgt jetzt der Reichweite des Prüfers
  (`correctionReviewScope` = engster Scope aus `zeiterfassung:corrections:approve` und `zeiterfassung:team:view`):
  manager (approve all, team:view team) -> eigene + Berichtslinie, hr_admin/admin -> Tenant, member -> eigene.
  Neues Proto-Feld `restrict_to_reporting_line` (8) in `ListWorkTimeEntriesReq`, regeneriert; gRPC setzt
  `WorkTimeFilter.ReportingLineOf` = Aufrufer aus dem Kontext, das Repository filtert im WHERE
  (`employee_id = caller OR IN (hr_employee_profiles.manager_user_id = caller)`, tenant-gebunden) — kein Post-Filter.
  Guard: `requireSelfReadOrCorrectionReview` = `selfRead` ODER Review-Keys als ZWEI getrennte Prüfungen (in einem
  gemeinsamen `RequirePermissionAny` würde ein per Override verweigerter Review-Key auch `hr:read`/`hr:self`
  entwerten und den Nutzer aus seinen eigenen Einträgen aussperren). Wer nur über die Review-Keys hereinkam
  (hr_admin), bekommt außerhalb der Queue 403. Alle anderen Anfragen laufen unverändert über `restrictToOwnEmployee`.
  Neuer DB-Test `route_hr_correction_queue_db_test.go`: echter Seed -> `ResolveTokenPermissions` -> echte Auth + HR-Routen
  -> echter HR-gRPC-Server (TenantInbound-Interceptor) -> Postgres-Repo; Mitarbeiter außerhalb der Berichtslinie geprüft.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (gateway 5513 PASS 0 Skips inkl. TestOpenAPIRouteDrift und
  neuer DB-Test real gelaufen; timetracking ok 0 Skips; server HR-Tests 185 PASS 0 Skips) | migration n.a. |
  rls-smoke n.a. (keine Tabelle/Policy) | e2e ok (lokal, 2:39, `TestMemberHRSelfServiceFlow` + `TestHREmployeeEditDesktopBodyFlow`)
- coverage: n.a. (kein Coverage-Ziel)
- mutations-probe: (1) Berichtslinien-Subselect auf `tenant_id = ... OR manager_user_id = ...` aufgeweicht -> manager
  sieht 3 statt 2, "outside the reporting line" rot. (2) Queue-Pfad deaktiviert (= Vorzustand) -> manager sieht nur sich,
  hr_admin 403 — damit auch der Totalausfall reproduziert. (3) Handler-Sperre "Review-Keys nur für die Queue"
  neutralisiert -> hr_admin ungefiltert 200 statt 403. Alle zurückgedreht, Diff sauber.
- verify vorgaenger: sauber (`c6bfb10b`: nur BACKLOG.yml + JOURNAL.md, kein Code)
- neue-units: `fe-time-entries-own-views-send-employee-id` nach BACKLOG-NEXT.yml (FE, Desktop tabu)
- desktop-aufrufer: `GET /hr/time/entries?status=correction_pending` -> `modules/team/TimeCorrectionPanel.tsx:33` -> passt
  (zeigt jetzt die Queue); dieselbe Anfrage aus `modules/profil/tabs/ZeiterfassungTab.tsx:134` ("meine Korrekturen") ->
  zeigt manager/hr_admin/admin jetzt die Team-Queue (Kosmetik, fe-time-entries-own-views-send-employee-id);
  ungefiltert aus `components/header/WorkClockWidget.tsx:50` und `ExportDialog.tsx:69` -> für manager/member unverändert
  (gepinnt), für admin wie bisher Tenant (Altfehler, gleiche NEXT-Unit). Antwortform unverändert.
- offen: Der Queue-Pfad hängt am Statusfilter (`lean:`-Marker), weil das FE dieselbe ungefilterte Anfrage für eigene
  Einträge sendet. Kein neuer Key, kein Re-Login nötig. `ApproveCorrection` selbst prüft keine Berichtslinie
  (manager hält approve `all`) — nicht Teil dieser Unit.

## Iteration 6 — fix-hr-employee-leave-balance-for-team-viewers — done — 2026-10-25 17:09
- commit: siehe Commit `fix(hr): show leave balances at the viewer's absence-data reach` (dieser Eintrag)
- gebaut: `GET /hr/leave/balance/{userId}` folgt der Reichweite aus `team:absence_data:view` (`leaveBalanceReach`):
  grober `hr:read`/`hr:self` über own -> Tenant (wie bisher), sonst absence_data team -> sich + Berichtslinie,
  all -> Tenant, sonst nur sich. Neues Proto-Feld `restrict_to_reporting_line` (4) in `GetEmployeeLeaveBalanceReq`,
  regeneriert; gRPC nimmt den Aufrufer aus dem Kontext, `leave.Service.GetReportingLineLeaveBalance` prüft
  `hr_employee_profiles.manager_user_id` (und Tenant) und liefert sonst `ErrNotInReportingLine` -> PermissionDenied
  -> 403; eine ID ohne Profil antwortet ebenso 403, nicht 404. Guard additiv: `selfRead` ODER `team:absence_data:view`
  als zwei getrennte Prüfungen — der Guard aus Iteration 5 ist dafür zu `requireSelfReadOr(view...)` verallgemeinert.
  Neuer DB-Test `route_hr_leave_balance_reach_db_test.go` (echter Seed -> ResolveTokenPermissions -> echte Auth +
  HR-Routen -> echter HR-gRPC-Server -> Postgres-Leave/Employee-Repos).
- gate: build ok | vet ok | lint ok (0 issues) | test ok (gateway 5518 PASS 0 Skips inkl. TestOpenAPIRouteDrift und
  neuer DB-Test; biz/hr/... 7 Pakete ok 0 Skips; server HR/Leave 202 PASS 0 Skips) | migration n.a. |
  rls-smoke n.a. (keine Tabelle/Policy) | e2e ok (lokal, 2:31, `TestMemberHRSelfServiceFlow`)
- coverage: n.a. (kein Coverage-Ziel)
- mutations-probe: (1) Berichtslinien-Bedingung im Service entfernt -> manager liest Außenstehenden 200 statt 403, rot.
  (2) Guard zurück auf `selfRead` (Vorzustand) -> hr_admin 403 statt 200, rot. (3) team-Zweig ohne
  `RestrictToReportingLine` -> manager liest Außenstehenden 200, rot. Alle zurückgedreht, Diff sauber.
- verify vorgaenger: sauber (`494ec76c`: Handler über `client.ListWorkTimeEntries`, `.proto` mit `.pb.go` regeneriert,
  Guard additiv, keine neue Route/Tabelle/Permission, Antwortform unverändert)
- neue-units: keine
- desktop-aufrufer: `GET /hr/leave/balance/{userId}` -> `api/hr-client.ts:407` (`getEmployeeBalance`) ->
  `hooks/hr-hooks.ts:151` (`useEmployeeLeaveBalance`) -> `modules/team/MemberProfileContent.tsx:87` -> passt
  (manager/hr_admin bekommen den Saldo jetzt, Antwortform unverändert `LeaveBalance`)
- offen: Unit-Test `TestHandleGetEmployeeLeaveBalance_ReachesRPC` ruft den Handler ohne Permissions auf und lebte
  davon, dass ein fehlender Scope als `all` galt; er fragt jetzt den eigenen Saldo ab. `GET /hr/leave/requests/{id}`
  (in der Unit "erwägen") nicht angefasst: kein Desktop-Aufrufer (`useLeaveRequest` unbenutzt), manager/hr_admin
  bekommen dort für fremde Anträge weiter 403 — kein Ausfall, deshalb keine Unit. Kein neuer Key, kein Re-Login nötig.

## Iteration 7 — harden-inbox-message-user-scoping — done — 2026-10-25 17:21
- commit: siehe Commit `fix(inbox): limit single-message actions to owner, assignee and team members` (dieser Eintrag)
- gebaut: Sichtbarkeit nach Lukes Entscheidung = Eigentümer ODER zugewiesen ODER Mitglied der Team-Inbox der Nachricht.
  SQL-Prädikat `visibleToCaller` in `message.PostgresRepository.GetByID` (neuer Parameter `userID`) und in
  `BulkMarkRead`/`BulkArchive` (unsichtbare IDs werden übersprungen). `message.Service`: GetByID, MarkRead/Unread,
  ToggleStar, SetStatus, Add/RemoveTag, Archive/Unarchive, Snooze, AssignMessage (Akteur), Reply, Forward und Bulk*
  prüfen den Aufrufer vor dem Schreiben (`requireVisible`, `lean:`-Marker). `InboxGRPCServer` nimmt den Aufrufer für
  alle Einzelnachrichten-RPCs plus ListThreadMessages/Unsnooze/Bulk aus dem Kontext (`callerID`, x-user-id) →
  fehlend = Unauthenticated, unsichtbar = NotFound; Reply/Forward/Claim nutzen weiter `req.UserId` (Gateway setzt
  beides aus dem Token). Helpdesk `CreateTicketFromMessage` braucht keine Änderung: der Outbound-Interceptor reicht
  x-user-id durch, d. h. ein Ticket entsteht nur aus einer Nachricht, die der Anfragende sieht.
  Kein Guard, keine Route, kein Proto, keine Migration angefasst.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (server + inbox/... + gateway: 10702 PASS, 0 Skips, 0 FAIL;
  inbox/message 41 PASS inkl. neuem DB-Test `TestGetByID_VisibleToOwnerAssigneeAndTeamMemberOnly` real gelaufen als
  kmuhub_app) | migration n.a. | rls-smoke n.a. (keine Tabelle/Policy) | e2e ok (lokal, 2:36,
  `TestInboxTeamAndRoutingFlow` als Smoke, kein Flow ruft die Einzelnachrichten-RPCs)
- coverage: n.a. (kein Coverage-Ziel; `internal/inbox/message` nach der Änderung 78,8 %)
- mutations-probe: (1) `visibleToCaller` auf `... OR $3 IS NOT NULL` aufgeweicht -> DB-Test rot (Kollege sieht Team- und
  Privatnachricht, Bulk-Schreiben landet). (2) Team-Zweig aus dem Prädikat entfernt -> DB-Test rot (Team-Mitglied
  NotFound, Bulk-Archiv landet nicht). (3) `requireVisible` auf `return nil` -> `TestSingleMessageWrites_RejectInvisibleCaller`
  und `TestInboxMessageRPCs_VisibleToOwnerAssigneeAndTeamMember` rot. Alle zurückgedreht, Diff sauber.
- verify vorgaenger: sauber (`2d60e1fb`: Handler über `client.GetEmployeeLeaveBalance`, `.proto` mit `.pb.go`
  regeneriert, Guard additiv `requireSelfReadOr`, keine Route/Tabelle/Permission neu, Antwortform unverändert)
- neue-units: keine
- desktop-aufrufer: `/inbox/messages/{id}/*` + `/messages/bulk/*` -> `api/inbox-client.ts:105-160` (über
  `api/hooks/useInbox.ts`) -> passt: das FE öffnet Nachrichten nur aus `GET /inbox/messages`, das schon auf
  `user_id = Aufrufer` filtert; Antwortformen unverändert, nur Fremd-IDs bekommen jetzt 404.
- offen: Änderung in Testerwartungen: ein Nicht-Mitglied, das NICHT Eigentümer ist, bekommt bei Claim jetzt 404 statt
  403 (Sichtbarkeit vor Mitgliedschaft); die 403-Pfade testen jetzt den Eigentümer ohne Mitgliedschaft. Bulk meldet
  `updated_count` weiterhin = Anzahl angefragter IDs, auch wenn unsichtbare übersprungen wurden (Altverhalten, kein
  FE-Leser). Die Team-Inbox-Liste selbst (`ListMessages` mit `team_inbox_id`) filtert weiter auf `user_id` — das ist
  `feat-team-inbox-for-members`.

## Iteration 8 — feat-team-inbox-for-members — done — 2026-10-25 17:36
- commit: siehe Commit `feat(inbox): let members read the team inboxes they belong to` (dieser Eintrag)
- gebaut: Neuer feiner Key `kommunikation:team_inbox:read` (Migration 000346: member/manager `own`, admin `all`).
  Gateway nimmt ihn additiv (`inboxTeamInboxRead` = Any(`inbox:read`, neu)) an GET `/messages`, `/messages/unread-count`,
  `/unread-count`, `/messages/{id}`, `/messages/{id}/thread`, `/teams`, `/teams/{id}/members` — nicht an Canned-
  Responses und `/rules`. Datengrenze vor dem Grant: `message.PostgresRepository.List` mit `team_inbox_id` liefert
  eigene Nachrichten ODER (bei Mitgliedschaft des Aufrufers) alle Nachrichten des Teams; ohne Teamfilter weiter nur
  eigene. Team-Liste war schon mitgliedschaftsbasiert (`ListByUser`), Einzelnachricht seit Iteration 7.
  E2E `TestInboxTeamAndRoutingFlow` um member-Schritte erweitert (eigenes Postfach sichtbar, nach Entfernen weg).
- gate: build ok | vet ok (inkl. `-tags=e2e`) | lint ok (0 issues) | test ok (inbox/... + server + gateway: 10710 PASS,
  0 Skips, 0 FAIL, inkl. TestOpenAPIRouteDrift; neue DB-Tests `TestList_TeamInboxFilterFollowsMembership` und
  `TestInboxTeamInbox_MemberReadsOwnTeamsOnly` real als kmuhub_app gelaufen, Token über ResolveTokenPermissions) |
  migration ok (up, down, up, zweites Anwenden INSERT 0 0; Kopf 346) | rls-smoke n.a. (keine Tabelle/Policy) |
  e2e ok (lokal, 2:41, `TestInboxTeamAndRoutingFlow`, beide neuen Subtests PASS)
- coverage: n.a. (kein Coverage-Ziel; `internal/inbox/message` nach der Änderung 81,1 %)
- mutations-probe: (1) Mitgliedschafts-Zweig in `List` entfernt -> Repo-DB-Test und Router-Test rot (Team-Mitglied
  sieht 0 statt 2). (2) `/teams` zurück auf `RequirePermission("inbox","read")` -> Router-Test rot (member 403).
  (3) `/messages` ebenso -> Router-Test rot. Alle zurückgedreht, Diff sauber.
- verify vorgaenger: sauber (`1c5b1e64`: kein Gateway-Code, kein Proto, keine Route/Tabelle/Permission;
  Sichtbarkeitsprädikat tenant-gescoped, Aufrufer aus dem Kontext, keine Stubs)
- neue-units: `fe-capability-catalog-team-inbox-read` (BACKLOG-NEXT.yml, Frontend: Rollen-Editor kennt den neuen Key nicht)
- desktop-aufrufer: GET `/inbox/teams`, `/teams/{id}/members`, `/messages` -> `api/inbox-client.ts:84,167,183` über
  `api/hooks/useInbox.ts:41,57,69` (`KommunikationPage`, `ConversationList`, `KommunikationSettingsPanel`) -> passt:
  Antwortformen unverändert, member bekommt jetzt 200 statt 403. Das FE gated die Seite nicht auf `inbox:read`.
- offen: Neuer Key wirkt erst nach Re-Login (Produktion). manager bekommt den Key mit (`own`), obwohl die Entscheidung
  member nennt — manager hielt `kommunikation:team_inbox:manage`, konnte die Postfächer aber nicht lesen; bitte
  bestätigen. Mit dem Key sieht member auch sein persönliches Postfach (GET `/messages` ohne Filter = eigene
  Nachrichten) und den eigenen Ungelesen-Zähler — beides war vorher 403. Schreibrouten (lesen markieren, Claim,
  Antworten) bleiben `inbox:write` = admin-only; ob member Team-Nachrichten bearbeiten darf, ist eine eigene Entscheidung.
  Mitgliederliste offener Team-Postfächer bleibt für Nicht-Mitglieder lesbar (Altverhalten seit Lauf 14, `visibility`).

## Iteration 9 — harden-email-dial-guard-for-account-hosts — done — 2026-10-25 17:48
- commit: siehe Commit `fix(email): refuse internal mail server addresses unless allowlisted` (dieser Eintrag)
- gebaut: Neues Paket `internal/security/safedial` (Guard): `Validate` löst den Host auf und lehnt jede Adresse ab, die
  `safehttp.CheckAddr` (neu exportiert, dieselbe Klassifikation wie der HTTP-Client) nicht als öffentlich durchlässt;
  `Dialer` hängt denselben Check in den `Control`-Hook (DNS-Rebinding). Allowlist aus neuem optionalem Env
  `EMAIL_DIAL_ALLOWLIST` (Host/IP/CIDR, komma-getrennt, Default leer, kein RequireX; ungültiger Eintrag bricht den
  email-Start ab). Nil-Guard = nur öffentlich (fail closed). Verdrahtet in `account.Create`/`Update` (beide Hosts,
  vor der Zuweisung), `TestConnection` (Check beider Hosts + Dial über `imapclient.Options.Dialer`), Sync-Dial
  (`IMAPClient.guard`, vom Worker aus `accountService.DialGuard()`) und SMTP-Versand (beide Transporte,
  `send.Service.WithDialGuard`). Neuer Fehler `account.ErrHostNotAllowed` -> `codes.InvalidArgument` (400).
- gate: build ok | vet ok | lint ok (0 issues: email/..., safedial, safehttp, config, cmd/email, server/...) |
  test ok (email/... + safedial + safehttp + config: 486 PASS, 0 SKIP; `internal/server` voll: 4998 PASS, 0 SKIP,
  0 FAIL, DATABASE_URL als kmuhub_app) | migration n.a. | rls-smoke n.a. (keine Tabelle/Policy) | e2e n.a. (kein
  Flow legt ein Mailkonto an); `internal/gateway` nicht gelaufen (keine Route, kein Gateway-Code angefasst)
- coverage: n.a. (kein Coverage-Ziel; nachher `internal/security/safedial` 93,2 %, `internal/email/sync` 84,5 %,
  `internal/email/account` 56,8 %)
- mutations-probe: (1) `checkAddr` -> `return nil`: safedial-Tests (4) und Account-Create/Update-Tests rot. (2) SMTP-Check
  in `Create` entfernt -> `TestCreate_RejectsInternalMailHosts` rot. (3) Sync-`Connect` mit plain `net.Dialer` ->
  `TestConnect_RefusesInternalAddressWithoutAllowlist` rot. (4) STARTTLS-Versand mit plain `net.Dialer` ->
  `TestSendSMTP_RefusesInternalAddressWithoutAllowlist` rot. (5) `TestConnection` ohne Guard-Dialer -> ÜBERLEBT:
  `Validate` fängt jede interne Adresse vorher, der Dial-Hook greift dort nur bei Rebinding (aus dem account-Paket
  ohne Resolver-Stub nicht erzeugbar; der Hook selbst ist in safedial gegen echten Listener getestet). Alle
  zurückgedreht, finaler Tree erneut grün.
- verify vorgaenger: sauber (`8f67a1ff`: Guards additiv `RequirePermissionAny(inbox:read, team_inbox:read)`, Seed 000346
  mit Up/Down für member/manager/admin, Repo-Reichweite tenant-gescoped plus Mitgliedschaft, keine neue Route/Tabelle)
- neue-units: keine
- desktop-aufrufer: POST `/email/accounts/`, PUT `/email/accounts/{id}`, POST `/email/accounts/test` ->
  `api/email-client.ts:87,102,115` über `api/hooks/useEmail.ts:107` -> passt: Antwortformen unverändert; interne Hosts
  bekommen bei Anlegen/Ändern 400 statt Erfolg, beim Verbindungstest `imap_ok=false` mit Meldung in `imap_error`
  (wie jeder andere Verbindungsfehler).
- offen: Compose-Passthrough für `EMAIL_DIAL_ALLOWLIST` an den email-Service fehlt (deploy/ gesperrt) — nötig erst, wenn
  ein Self-Hosted-Kunde einen internen Mailserver hat. VERHALTENSÄNDERUNG für Bestandskonten: ein schon gespeichertes
  Konto mit privatem/Loopback-Host synct und sendet ab dem Deploy nicht mehr (Sync loggt `IMAP connection lost ...
  not a public address`), bis der Host auf die Allowlist kommt — vor Deploy auf Prod prüfen:
  `SELECT imap_host, smtp_host FROM email_accounts` gegen private Ranges. Nicht auflösbare Hosts passieren `Validate`
  bewusst (der Dial scheitert ohnehin, die Grenze ist der Dial-Hook). Der Verbindungstest meldet eine gesperrte
  SMTP-Adresse im Feld `imap_error` (Proto kennt nur das). `systemmail` (Operator-SMTP aus Env) bleibt ungefiltert.

## Iteration 10 — feat-email-account-self-setup-for-members — done — 2026-10-25 17:59
- commit: siehe Commit `feat(email): let members set up and manage their own mail account` (dieser Eintrag)
- gebaut: Migration 000347 `email:account:write|delete` (member/manager `own`, admin `all`, zweimal anwendbar,
  Down entfernt genau diese Keys). Gateway: Anlegen/Ändern/Verbindungstest `RequirePermissionAny(email:write,
  email:account:write)`, Löschen `(email:delete, email:account:delete)` — additiv. Anlegen pinnt `user_id` bei
  Scope `own` über `ownerFilterForScopeAny` (leer -> Aufrufer, fremd -> 403). Email-Server: `mayManageAccountOf`
  (Create, fremde user_id -> PermissionDenied) und `mayManageAccount` (Update/Delete über `ownAccount`, fremd ->
  NotFound), Ausnahme nur mit weitergereichtem groben Key: `email:write`/`email:delete` neu in
  `forwardedPermissions` (`PermEmailWrite`/`PermEmailDelete`, Whitelist-Test mitgezogen). Spec: 403 an POST
  `/email/accounts` dokumentiert (swagger-cli valid).
- gate: build ok | vet ok | lint ok (0 issues: server, gateway, middleware) | test ok (`internal/gateway` 5533 PASS,
  `internal/server` 5002 PASS, middleware/auth/email/... ok, 0 SKIP, DATABASE_URL als kmuhub_app) | migration ok
  (347 up/down/up, Kopf 347 clean) | rls-smoke n.a. (keine Tabelle/Policy) | e2e n.a. (kein Flow berührt
  `/email/accounts`)
- coverage: n.a. (kein Coverage-Ziel)
- mutations-probe: (1) Besitzprüfung in `UpdateEmailAccount` entfernt -> `TestUpdateEmailAccount_SomeoneElsesAccountIsNotFound`
  und der DB-Kettentest (member PUT fremdes Konto = 200 statt 404) rot. (2) `mayManageAccountOf` lässt fremde
  user_id durch -> `TestCreateEmailAccount_ForSomeoneElseNeedsTheProvisioningKey` rot. (3) grobe Email-Keys aus
  `forwardedPermissions` entfernt -> DB-Kettentest (admin legt für anderen an = 403) rot. Alle zurückgedreht,
  finaler Tree erneut grün.
- verify vorgaenger: sauber (`7f7faf03`: kein Proto, keine Route, kein neuer Guard, keine Tabelle; Dial-Guard in
  account/send/sync/cmd verdrahtet, kein Stub)
- neue-units: keine
- desktop-aufrufer: POST `/email/accounts/`, PUT/DELETE `/email/accounts/{id}`, POST `/email/accounts/test` ->
  `api/email-client.ts:86-119` über `api/hooks/useEmail.ts:75-108` -> passt: Antwortformen unverändert, der FE-Typ
  `CreateEmailAccountRequest` sendet `user_id` immer mit (eigene ID = 201). Kein Konto-Dialog ruft die Hooks
  derzeit auf (kein TSX-Aufrufer von `useCreateEmailAccount`/`useUpdateEmailAccount`/`useDeleteEmailAccount`/
  `useTestEmailConnection`) — die Self-Setup-Oberfläche fehlt im FE.
- offen: Neue Keys wirken erst nach Re-Login (Produktion). Die groben Keys `email:write`/`email:delete` reisen jetzt
  über den gRPC-Hop zu ALLEN Services (nur der Email-Server liest sie); der Hop trägt keine Scopes — eine Custom-Rolle
  mit grobem `email:write` bei Scope `own` könnte fremde Konten ändern (`lean:`-Marker in `grpc_tenant.go`; heute
  hält nur admin die groben Keys, Scope `all`). manager bekommt den Key mit (`own`), wie bei 000337/000346 — bitte
  bestätigen. Ein FE-Konto-Dialog für member fehlt; als FE-Thema für NEXT einordnen, falls gewünscht.

## Iteration 11 — feat-email-rules-per-user — done — 2026-10-25 18:12
- commit: siehe Commit `feat(email): scope email rules to their owner` (dieser Eintrag)
- gebaut: Migration 000348 `email_rules.user_id` (nullable, FK users ON DELETE CASCADE, Index tenant+user; Bestand
  bleibt NULL = tenantweit) plus `email:rule:read|write|delete` (member/manager `own`, admin `all`, zweimal
  anwendbar; Down löscht erst die Nutzerregeln, damit sie nicht tenantweit werden). `rule.Service` mit `Actor`:
  Create stempelt immer den Aufrufer (auch admin); List = tenantweite + eigene (fremde Nutzerregeln unsichtbar,
  auch für admin); Update/Delete über `maintainable` (fremde -> NotFound, tenantweit ohne groben Key ->
  `ErrTenantWideRule`/PermissionDenied); Move-Ziel einer Nutzerregel muss ein Ordner eines EIGENEN Kontos sein
  (`FolderBelongsToUser`), sonst landete eigene Post im Postfach eines Kollegen; `ApplyAll` wendet je Nachricht
  nur tenantweite Regeln plus die des Kontoinhabers an, Kandidaten bei admin (grobes `email:write` über den Hop)
  tenantweit, sonst nur eigene Postfächer. Gateway-Guards additiv `RequirePermissionAny(email:*, email:rule:*)`.
  Spec-Beschreibungen nachgezogen (Statuscodes standen schon, swagger-cli valid).
- gate: build ok | vet ok | lint ok (0 issues: email/rule, server, gateway, models) | test ok (`internal/email/...`
  422 PASS, `internal/server` 5004 PASS, `internal/gateway` 5542 PASS, 0 SKIP, 0 FAIL, DATABASE_URL als
  kmuhub_app; DB-Tests real gelaufen: rule-Repo 6 inkl. neuem Service-auf-Postgres-Test, Gateway-Kettentest mit
  echtem Seed über `ResolveTokenPermissions`) | migration ok (348 up/down/up, Kopf 348 clean, Up per psql ein
  zweites Mal: no-op) | rls-smoke ok (`email_rules` als kmuhub_app: eigener Tenant 1, fremder 0) | e2e n.a. (kein
  Flow berührt `/email/rules`)
- coverage: `internal/email/rule` n.a. -> 82,3 % (kein Coverage-Ziel, Vorher nicht gemessen)
- mutations-probe: (1) Owner-Filter in `evaluateMessage` entfernt -> `TestApplyAll_PersonalRulesOnlyTouchTheOwnersMailbox`
  und DB-Test `TestService_PersonalRuleLeavesAColleaguesMailboxAlone` rot. (2) Fremd-Zweig in `maintainable`
  entfernt -> `TestUpdateDelete_OwnershipHoldsWithoutTheGateway` und Gateway-Kettentest (colleague's rule) rot.
  (3) Nutzerregel prüft Move-Ziel nur gegen Tenant -> `TestCreate_PersonalMoveTargetMustBeTheOwnersFolder` und
  DB-Test rot. (4) List-Filter `return false` -> `TestList_ShowsTenantWideAndOwnRulesOnly` rot. (5) Kandidaten-
  Einschränkung für Nicht-admin aus -> `TestApplyAll_PersonalRulesOnlyTouchTheOwnersMailbox/bob` rot.
  (6) `ruleActor` Admin immer true -> `TestUpdateDeleteEmailRule_TenantWideRuleNeedsTheCoarseKey` und
  Gateway-Kettentest (tenant-wide 403) rot. Alle zurückgedreht, finaler Tree erneut grün.
- verify vorgaenger: sauber (`d6c12f0b`: Guards additiv, Seed 000347 mit Up/Down für member/manager/admin,
  Besitzprüfung im Email-Server, Whitelist-Test mitgezogen, keine neue Route/Tabelle, kein Proto)
- neue-units: `fe-email-rules-per-user-catalog-and-tenant-wide-marker` (BACKLOG-NEXT.yml, FE-Thema, blocked)
- desktop-aufrufer: GET/POST `/email/rules`, DELETE `/email/rules/{id}`, POST `/email/rules/apply` ->
  `api/email-client.ts:337-349` über `api/hooks/useEmail.ts:492` und `modules/mails/RulesDialog.tsx` -> passt:
  Antwortformen unverändert (`{rules:[...]}`, `{rule:{...}}`). member sieht im Dialog tenantweite Regeln mit
  Löschknopf, der Klick gibt 403 — FE kann tenantweit nicht erkennen (Proto ohne Feld), siehe neue NEXT-Unit.
- offen: Neue Keys wirken erst nach Re-Login. Verhaltensänderung: jede ab dem Deploy angelegte Regel ist eine
  Nutzerregel — auch die von admin; neue tenantweite Regeln lassen sich nicht mehr anlegen (Entscheidung wörtlich
  "Neue Regeln bekommen den Aufrufer"), bitte bestätigen. admin sieht und ändert fremde Nutzerregeln nicht
  (Postfach-Logik wie `ownAccount`). Der Admin-Zweig im Server hängt am weitergereichten `email:write`, auch für
  DELETE (eine Custom-Rolle mit nur `email:delete` löscht tenantweite Regeln nicht). Bestehende tenantweite
  Move-Regeln dürfen weiter in jeden Ordner des Tenants verschieben (unverändert, nicht Teil der Unit).
  manager bekommt die Keys mit (`own`), wie bei 000337/000346/000347.

## Iteration 12 — feat-email-templates-own-personal-edit — done — 2026-10-25 18:27
- commit: siehe Commit `feat(email): let members maintain their own personal templates` (dieser Eintrag)
- gebaut: Migration 000349 seedet `email:template:write|delete` (member/manager `own`, admin `all`, zweimal
  anwendbar, Down entfernt genau diese Keys). `template.Service.Update/Delete` bekommen `maintainer` (grobes
  `email:write` bzw. `email:delete`, über den Hop weitergereicht): ohne ihn nur eigene persönliche Vorlagen,
  geteilte -> `ErrSharedTemplate` (PermissionDenied/403), fremde persönliche -> NotFound (auch wenn `is_admin`
  sie sichtbar macht), jeder echte Sichtbarkeitswechsel -> 403 (unveränderte `visibility` im Body ist erlaubt).
  Gateway-Guards PUT/DELETE additiv `RequirePermissionAny(email:*, email:template:*)`. Spec: 403 plus
  Beschreibung an PUT/DELETE, swagger-cli valid.
- gate: build ok | vet ok | lint ok (0 issues: email/template, server, gateway, middleware) | test ok
  (`internal/email/...`, `internal/middleware`, `internal/server`, `internal/gateway`: 11083 PASS, 0 SKIP, 0 FAIL,
  DATABASE_URL als kmuhub_app; DB-Tests real gelaufen: Template-Repo 2, Gateway-Kettentest mit echtem Seed über
  `ResolveTokenPermissions` und Postgres-Template-Repo, 8 Subtests) | migration ok (349 up/down/up, Kopf 349
  clean, Up per psql ein zweites Mal: no-op) | rls-smoke n.a. (keine Tabelle/Policy, nur `permissions`/
  `role_permissions`) | e2e n.a. (kein Flow berührt `/email/templates`)
- coverage: `internal/email/template` 79,1 % -> 87,6 %
- mutations-probe: (1) Shared-Zweig in `maintainable` entfernt -> `TestUpdateDelete_SharedTemplateNeedsTheCoarseKey`
  und Gateway-Kettentest (shared 403) rot. (2) Besitzprüfung in `maintainable` entfernt ->
  `TestUpdateDelete_ForeignPersonalTemplateStaysHiddenWithoutTheCoarseKey` rot. (3) Sichtbarkeitswechsel-Prüfung
  aus -> `TestUpdate_SharingAPersonalTemplateNeedsTheCoarseKey` und Gateway-Kettentest (personal->shared, Zeile
  unverändert) rot. (4) Delete prüft `email:write` statt `email:delete` ->
  `TestDeleteEmailTemplate_SharedNeedsTheCoarseDeleteKey` rot. (5) PUT-Guard wieder nur grob -> Gateway-Kettentest
  (own 200, fremde 404) rot. Alle zurückgedreht, finaler Tree erneut grün.
- verify vorgaenger: sauber (`a8351f01`: Guards additiv, Seed 000348 mit Up/Down für member/manager/admin,
  INSERT setzt `user_id`, List/Apply-Kandidaten owner-gefiltert, kein Proto, keine neue Route)
- neue-units: `decide-email-template-create-shared-by-member` (BACKLOG-NEXT.yml, Entscheidung Luke, blocked)
- desktop-aufrufer: PUT/DELETE `/email/templates/{id}` -> kein Aufrufer in `desktop/src/renderer/src/api/`
  (Clients und hooks); die Mail-Vorlagen laufen über `stores/mailTemplates.ts` rein lokal ("no backend") -> passt
  (nichts bricht), die neuen Rechte wirken erst mit einer FE-Anbindung.
- offen: Neue Keys wirken erst nach Re-Login. Das ANLEGEN geteilter Vorlagen bleibt für member offen (POST mit
  `email:mailbox:write`) — member kann eine geteilte Vorlage anlegen, die er danach nicht mehr ändern kann;
  Entscheidung in der neuen NEXT-Unit. admin sieht und ändert fremde persönliche Vorlagen weiter (unverändertes
  Verhalten, `is_admin` plus grober Key). manager bekommt die Keys mit (`own`), wie bei 000347/000348.
  "Admin" heißt im Server: grober Key über den Hop (wie bei den Regeln), nicht die Rolle.

## Iteration 13 — feat-email-labels-read-assign-for-members — done — 2026-10-25 18:38
- commit: siehe Commit `test(email): prove members label their own mail and leave the catalogue to admin` (dieser Eintrag)
- gebaut: Kein Produktionscode nötig — Grenze und Grants standen schon: `GET /email/labels` und
  `POST /email/messages/{id}/labels` hängen seit 000337 an `boxRead`/`boxWrite` (`email:mailbox:*`, member/manager
  `own`), `AssignMessageLabels` prüft `ownMessage` vor der Zuweisung, Katalog-POST/PATCH/DELETE bleiben auf
  `email:write|delete` (admin). Neu: Gateway-Kettentest `route_email_labels_member_db_test.go` mit echtem Seed über
  `ResolveTokenPermissions`, echtem Auth, gRPC-Server hinter `TenantInboundUnaryInterceptor` und Postgres-Repos für
  Account/Message/Label; Router-Kommentar nennt den geteilten Katalog.
- gate: build ok | vet ok | lint ok (0 issues: gateway, email/label) | test ok (`internal/gateway` 5558 PASS,
  0 SKIP, 0 FAIL, DATABASE_URL als kmuhub_app; DB-Tests real gelaufen: neuer Kettentest mit 6 Subtests; `email/label`
  ok) | migration n.a. (keine) | rls-smoke n.a. (keine Tabelle/Policy) | e2e n.a. (kein Flow berührt Labels)
- coverage: n.a. (kein Coverage-Ziel; nur Gateway-Test, `internal/email/label` unverändert 83,1 %)
- mutations-probe: (1) `ownMessage`-Prüfung in `AssignMessageLabels` entfernt -> Subtest "colleague's message" rot.
  (2) Assign-Guard auf grobes `email:write` -> "own message" und "colleague's message" rot. (3) Katalog-POST auf
  `boxWrite` -> "cannot change the catalogue" rot. (4) Katalog-GET auf grobes `email:read` -> "reads the catalogue"
  rot. Alle per `git checkout` zurückgedreht, finaler Tree erneut grün.
- verify vorgaenger: sauber (`c1eacccd`: Guards PUT/DELETE additiv, Seed 000349 mit Up/Down für member/manager/
  admin, `maintainer` aus weitergereichtem grobem Key, ErrSharedTemplate -> PermissionDenied, kein Proto, keine
  neue Route)
- neue-units: keine
- desktop-aufrufer: n.a. (keine Validierungs- oder Antwortform-Änderung); zur Kontrolle `api/email-client.ts:310-330`
  liest `{labels:[...]}` und `{message:{...}}` und sendet `{label_ids:[...]}` -> passt.
- offen: Die Unit war vor dem Lauf schon erfüllt (000337 + `ownMessage`), dieser Commit belegt sie nur. Testbefund
  ohne Produktionswirkung: `email_messages.message_id/preview/body_*/raw_headers` und `email_accounts.display_name`
  sind nullable, die Scans lesen sie als `string` — eine per SQL eingefügte Zeile mit NULL gibt 500. Die App
  schreibt diese Spalten immer als String (Sync/Create), deshalb keine Unit; Seeds im Test setzen sie explizit.

## Iteration 14 — feat-email-contact-history-own-for-members — done — 2026-10-25 18:46
- commit: siehe Commit `feat(email): show members the contact mail history of their own mailboxes` (dieser Eintrag)
- gebaut: Migration 000350 seedet `email:contact_history:read` (member `own`, manager/admin `all`).
  `GET /email/links/contact/{contactId}` nimmt `RequirePermissionAny(email:read, email:contact_history:read)`
  (additiv); der Handler setzt bei Scope `own` das neue Proto-Feld `own_mailboxes_only`, der Server reicht dann
  den Aufrufer (`x-user-id`) als Owner an `contactlink.GetByContactID`, das in Liste UND Count per
  `m.account_id IN (SELECT id FROM email_accounts WHERE user_id = $3)` filtert. Proto regeneriert
  (protoc 33.4, gen-go 1.36.11), Spec mit 400/403 und Scope-Beschreibung.
- gate: build ok | vet ok | lint ok (0 issues: email/contactlink, server, gateway) | test ok (`internal/gateway`
  5564 PASS, 0 SKIP; `internal/server -run Email|GetContact` 165 PASS, 0 SKIP; `internal/email/...` ok;
  DATABASE_URL als kmuhub_app; DB-Tests real: Gateway-Kettentest 5 Subtests, Repo-Test 1) | migration ok
  (up/down/up plus zweites Anwenden der Up-Datei, Grants member own / manager all / admin all) | rls-smoke n.a.
  (keine Tabelle/Policy; Repo-Test prüft fremden Tenant = 0) | e2e n.a. (kein Flow berührt die Route)
- coverage: internal/email/contactlink 35,0 % -> 78,0 %
- mutations-probe: (1) Owner-Klausel im Repo zu `OR TRUE` -> Repo-Test (alice total 3 statt 2) und Gateway-Subtests
  "member sees ... own mailbox only" und "total stays narrowed" rot. (2) Server ignoriert `OwnMailboxesOnly` ->
  beide member-Subtests rot. (3) Scope-Paar im Handler nur grob (`email:read`) -> member-Subtests rot. (4) Guard
  wieder nur grob -> member UND manager rot. Alle per Backup zurückgedreht, finaler Tree erneut grün.
- verify vorgaenger: sauber (`b35523cd`: nur Test plus Router-Kommentar, keine Guard-, Seed- oder Proto-Änderung)
- neue-units: keine
- desktop-aufrufer: GET `/email/links/contact/{contactId}` -> `api/email-client.ts:297` (`getContactEmails`, liest
  `GetContactEmailsResponse` `{messages,total}`) -> passt, Antwortform unverändert; member bekommt statt 403
  jetzt 200 mit den eigenen Mails.
- offen: Neuer Key wirkt erst nach Re-Login. Ein Custom-Role mit `email:contact_history:read` auf `team` wird
  wie `all` behandelt (`ownerFilterForScopeAny` engt nur bei `own` ein) — heute hat keine Systemrolle `team`.
  Lokal fehlt `make` in Git Bash; Migration per `migrate create -seq -digits 6` angelegt.

## Iteration 15 — fix-fuhrpark-trip-logs-private-flag-not-filtered — done — 2026-10-25 18:56
- commit: siehe Commit `fix(fuhrpark): mask private trips for everyone but their driver` (dieser Eintrag)
- gebaut: `fuhrpark.maskPrivateTrips` im Service leert bei `is_private` für jeden außer dem Fahrer
  (`driver_user_id` == Aufrufer) Start-/Zielort, Geschäftspartner und Notizen und setzt den Zweck auf das Label
  „Privatfahrt“; Datum, km-Stände, Strecke, Fahrzeug und Fahrer bleiben. Greift in `ListTripLogs` (Liste und
  Fahrzeugliste) und `ExportTripLogs` (CSV/PDF, Zeile bleibt drin). Neues Feld `ViewerID` in beiden Params, der
  gRPC-Server setzt es aus `x-user-id` (`tripViewer`, fehlend -> uuid.Nil -> alles maskiert). Privatfahrt ohne
  Fahrerkonto ist für alle maskiert. Kein Proto, keine Route, keine Migration.
- gate: build ok | vet ok | lint ok (0 issues: fuhrpark, server, gateway) | test ok (`internal/fuhrpark` 189 PASS,
  0 SKIP; `internal/server` 5011 PASS, 0 SKIP; `internal/gateway` 5564 PASS, 0 SKIP; DATABASE_URL als kmuhub_app;
  DB-Tests real: neuer Schema-Test mit 5 Subtests) | migration n.a. | rls-smoke n.a. (keine Tabelle/Policy) |
  e2e n.a. (kein Flow berührt Fahrtenbuch)
- coverage: n.a. (kein Coverage-Ziel); `internal/fuhrpark` nachher 83,4 %
- mutations-probe: (1) Fahrer-Ausnahme entfernt (alles maskiert) -> `TestMaskPrivateTrips/for the driver...`,
  Schema-Subtests "driver lists ... in full" und "driver's export" sowie Server-Subtests rot. (2) `ViewerID` in
  `ListTripLogs` des Servers weggelassen -> Server-Subtest "the driver lists the full trip" rot. (3) Maskierung im
  Export entfernt -> Schema-Subtest "manager's export ... masked" und Server-Subtest "colleague exports" rot.
  (4) Geschäftspartner nicht geleert -> Unit- und beide Manager-Schema-Subtests rot. Alle per Backup
  zurückgedreht, finaler Tree erneut grün.
- verify vorgaenger: sauber (`f897ceb2`: Guard additiv per `RequirePermissionAny`, Seed 000350 mit Up/Down für
  member own / manager+admin all, Proto mit `.pb.go` regeneriert, Owner-Filter in Liste UND Count der Query)
- neue-units: `fix-fuhrpark-concurrent-booking-move-deadlock` (Backlog-Ende, Zwilling der produktion-Unit:
  `TestVehicleBookingOverlapConstraint` fiel lokal einmal mit 40P01, 3x danach grün); in BACKLOG-NEXT
  `decide-fuhrpark-gps-during-private-trips` (GPS-Punkte lassen sich Privatfahrten heute nicht zuordnen)
- desktop-aufrufer: GET `/fuhrpark/trip-logs` und `/fuhrpark/vehicles/{id}/trip-logs` -> `api/fuhrpark-client.ts`
  (`TripLog`), `FuhrparkPage.tsx:115 adaptTripLog` -> passt, Form unverändert; maskierte Zeile zeigt leere Orte und
  Zweck „Privatfahrt“. Export liefert Datei, Form unverändert.
- offen: (1) `UpdateTripLog` schreibt alle Felder voll (`postgres_repository.go`, SET aller Spalten): ein
  Nicht-Fahrer, der eine maskierte Privatfahrt aus der Liste editiert und zurückschickt, würde Ziel/Zweck mit
  „“/„Privatfahrt“ überschreiben. Heute ohne Wirkung, das Desktop hat keinen Edit-Dialog für Fahrten
  (`updateTripLog` wird nirgends aufgerufen); beim Anbinden beachten. (2) „ohne GPS“ aus der Entscheidung betrifft
  die GPS-Routen, siehe NEXT-Unit. (3) Fahrername bleibt in der maskierten Zeile sichtbar (Fahrtenbuch braucht ihn).

## Iteration 16 — harden-share-tokens-hashed-wiki-formulare — done — 2026-10-25 19:04
- commit: siehe Commit `fix(wiki,formulare): store share tokens only as SHA-256` (dieser Eintrag)
- gebaut: Neues Paket `internal/linksecret` (`New`: 32 Byte crypto/rand, base64url; `Hash`: hex-SHA-256) ersetzt die
  wortgleichen `newShareToken`-Kopien in `wiki/share.go` und `formulare/form_share.go`. Migration 000351 legt in
  `wiki_share_tokens` und `form_share_tokens` `token_hash` an, füllt sie per `encode(sha256(convert_to(token,'UTF8')),'hex')`
  (PG-Core, kein pgcrypto), setzt NOT NULL + UNIQUE und droppt die Klartextspalte samt Index. Die Repos schreiben
  und suchen nur noch per Hash; Listen und Lookups liefern `Token` leer, nur die Create-Antwort trägt das Token einmal.
- gate: build ok | vet ok | lint ok (0 issues: linksecret, wiki, formulare, server) | test ok (`internal/wiki` +
  `internal/formulare` 196 RUN, 0 SKIP; `internal/server` + `internal/gateway` 10575 PASS, 0 SKIP; DATABASE_URL als
  kmuhub_app; DB-Tests real: 2 neue Tests mit 5 Subtests) | migration ok (up/down/up auf 351; Backfill-Probe: Klartext-
  Zeile bei 350 eingefügt, nach up `token_hash` == `sha256sum` des Klartexts, Klartext nicht mehr in `row_to_json`) |
  rls-smoke ok (form_share_tokens als kmuhub_app: fremd 0, eigen 1; Policy unverändert) | e2e n.a. (kein Flow berührt
  Share-Links)
- coverage: n.a. (kein Coverage-Ziel); neues Paket `internal/linksecret` 83,3 %
- mutations-probe: (1) Wiki-Lookup ohne Hash -> `TestWikiShareRedeem_ResolvesTenantWithoutOne`, `..._RevokedTokenIsRefused`,
  `TestWikiShareToken_StoredOnlyAsHash` rot. (2) Wiki-Insert speichert Klartext -> dieselben drei rot ("stored row contains
  the clear secret"). (3) Formular-Lookup ohne Hash -> `TestSubmitByShareToken_IdempotencyKeyDedupesAgainstPostgres`,
  `..._TwoTenantsRedemptionStaysScoped`, `TestFormShareLink_StoredOnlyAsHash` rot. (4) Formular-Insert Klartext ->
  `TestFormShareLink_StoredOnlyAsHash` rot. Alle per Backup zurückgedreht, finaler Tree erneut grün.
- verify vorgaenger: sauber (`944d87c4`: kein Guard, kein Proto, keine Migration; `tripViewer` liest `x-user-id`, das
  `grpc_tenant.go` aus den Metadaten in `UserIDKey` legt; Maskierung in Liste und Export)
- neue-units: keine
- desktop-aufrufer: GET `/wiki/articles/{id}/share` -> `api/wiki-client.ts:216` / `WikiShareDialog.tsx` liest nur `id`,
  `permissions`, `created_at` (kopiert wird der interne App-Link, nie `token`) -> passt, `token` kommt jetzt leer.
  GET `/formulare/schemas/{id}/share-links` -> `FormularePage.tsx:980 shareValueOf` kopiert `url` (Gateway sendet heute
  `""`), liest `token` nirgends -> passt. POST beider Routen liefert das Token unverändert.
- offen: (1) Down-Migration kann Klartext nicht zurückholen: sie füllt `token` mit dem Hash, danach sind alle vorher
  erzeugten Links tot (im Down-Kommentar). (2) Kommt FD-4 (öffentliche Ausfüllseite), muss die URL beim Erstellen
  gezeigt werden — aus der Liste lässt sie sich nicht mehr bauen (`decide-formulare-share-link-url-channel-toggle`).
  (3) `report_share_tokens` (000252) bleibt bewusst Klartext, nicht Teil dieser Unit. (4) Lokal kein `make`, Migration
  per `migrate create -seq -digits 6`.

## Iteration 17 — harden-share-links-max-365-days — done — 2026-10-25 19:13
- commit: siehe Commit `fix(wiki,formulare,berichte): expire every share link within 365 days` (dieser Eintrag)
- vorweg blockiert: `harden-share-tokens-hashed-berichte-csat` (erste baubare Unit) -> `blocked`, im selben Commit nach
  BACKLOG-NEXT.yml verschoben. Die Entscheidung setzt voraus, dass das Token nur in der Create-Antwort gebraucht wird; das
  stimmt für beide Tabellen nicht. Berichte: `ShareLinkDialog.tsx:134/138` baut den kopierbaren Link aus `s.token` der
  LISTE und verwirft die Create-Antwort, Hash-only macht „Link kopieren“ tot (Desktop für den Loop tabu). CSAT: das Token
  wird beim Schließen gemintet und vom Dispatcher verzögert verschickt, der es per `ListDueCsatSurveys`
  (`SELECT r.token`) aus der DB liest (`csat_dispatch.go:288`). FRAGE AN LUKE steht als (a)/(b)/(c) im `blocked_reason`.
  Die nur ordnende dep von `harden-share-links-max-365-days` auf die blockierte Unit ist gelöst (Kommentar in der Unit).
- gebaut: `linksecret.MaxLifetime` (365 Tage) + `linksecret.Expiry(requested, now)`: ohne Wunsch now+365 Tage, darüber
  `ErrLifetimeTooLong`. Wiki und Formulare nutzen es (Fehler in `ErrInvalidShareLink` gewickelt -> InvalidArgument -> 400),
  Berichte setzt `expires_in_days` null/0 auf `maxShareExpiryDays` statt „nie“. Migration 000352: Bestand ohne Ablauf auf
  `created_at + 365 days`, dann `expires_at SET NOT NULL` an wiki_share_tokens, form_share_tokens, report_share_tokens.
  Spec-Texte der drei Create-Routen nachgezogen (keine neue Route).
- gate: build ok | vet ok | lint ok (0 issues: linksecret, wiki, formulare, berichte, server) | test ok (linksecret, wiki,
  formulare, berichte, server: 5304 RUN, 0 SKIP, 0 FAIL; gateway ok; DATABASE_URL als kmuhub_app; DB-Tests real: 2 neue
  Subtests „expires after a year“ plus 6 angepasste DB-Tests) | migration ok (up/down/up auf 352; Backfill-Probe bei 351:
  Zeile 10 Tage alt -> lebt, 2 Jahre alt -> abgelaufen, beide `expires_at - created_at` = 365 days; down macht NULLable)
  | rls-smoke ok (report_share_tokens als kmuhub_app: eigen 1, fremd 0; Policy unverändert) | swagger-cli valid |
  e2e n.a. (kein Flow berührt Share-Links, grep `test/e2e/`)
- coverage: n.a. (kein Coverage-Ziel); `internal/linksecret` 83,3 % -> 91,7 %
- mutations-probe: (1) Kappung in `Expiry` aus -> `TestExpiry`, `TestCreateFormShareLink_DefaultsToAYearAndRefusesMore`,
  `TestCreateShareToken_DefaultsToAYearAndRefusesMore` rot. (2) Wiki ohne Default (Ablauf +100 Jahre) -> DB-Subtest
  „a link expires after a year …“ und Server-Test rot. (3) Formulare dito -> DB-Subtest und Server-Test rot. (4) Berichte
  Default 0 -> `TestCreateShareToken_SecretIsFullEntropyAndUnique`, `..._ZeroExpiryMeansTheCap` rot. Alle per Backup
  zurückgedreht, finaler Tree erneut grün.
- verify vorgaenger: sauber (`9fe78eb6`: kein Guard, kein Proto, keine neue Tabelle/Route; Hash nur in INSERT und Lookup,
  Liste ohne Token-Spalte, Backfill in SQL identisch zu `linksecret.Hash`)
- neue-units: `harden-document-share-links-max-365-days` (Backlog-Ende: `document_share_links.expires_at` aus 000266 ist
  ebenso optional); in BACKLOG-NEXT `fe-share-link-expiry-max-365-days` und die blockierte
  `harden-share-tokens-hashed-berichte-csat`
- desktop-aufrufer: POST `/wiki/articles/{id}/share` -> `WikiShareDialog.tsx` sendet kein `expires_at` -> passt (365 Tage).
  POST `/formulare/schemas/{id}/share-links` -> `FormularePage.tsx:1034` sendet `expiresAt` aus Datumsfeld oder `null` ->
  passt bei leer (365 Tage); Datum > 365 Tage gibt 400-Toast, Feld hat kein `max` (FE-Unit). POST
  `/berichte/documents/{id}/shares` -> `ShareLinkDialog.tsx:66` sendet `null` für „nie“ -> Link läuft nach 365 Tagen ab,
  Liste zeigt dann das Datum statt „läuft nie ab“ (FE-Unit, Option umbenennen). Antwortformen unverändert.
- offen: (1) Lokal null Bestandszeilen, in Produktion Anzahl sofort abgelaufener Links (älter als ein Jahr, ohne Ablauf)
  vor dem Deploy zählen: `SELECT count(*) FROM <tabelle> WHERE expires_at IS NULL AND created_at < now() - interval
  '365 days'` für alle drei Tabellen. (2) Bestand mit Ablauf > 1 Jahr (alter Wiki/Formular-Pfad) bleibt bewusst stehen,
  steht im Migrationskommentar. (3) Down setzt die Backfill-Werte nicht auf NULL zurück. (4) Done-when „Router-Tests“
  als gRPC-Server-Tests umgesetzt: der Gateway-Handler reicht durch, die 400 entsteht über `respondGRPCError` aus
  InvalidArgument; ein Gateway-Test mit Mock-Client hätte die Service-Regel nicht geprüft.

## Iteration 18 — harden-calendar-join-token-participants-only — done — 2026-10-25 19:26
- commit: siehe Commit `fix(calendar): issue video join tokens only to organiser and attendees` (dieser Eintrag)
- gebaut: `event.Service.RequireParticipant` (Organisator = `created_by` oder Zeile in `event_attendees`, sonst
  `ErrEventNotFound` wie bei fehlendem Event -> NotFound/404, kein Existenz-Orakel). `GenerateJoinToken` prüft damit den
  Aufrufer aus dem Kontext (`callerID`) statt nur die Tenant-Existenz und stellt das LiveKit-Token auf die geprüfte
  Aufrufer-ID aus statt auf `req.UserId` (sonst hätte ein Teilnehmer ein Token in fremdem Namen ziehen können; der
  Gateway setzt das Feld heute ohnehin aus dem JWT). Spec-Beschreibung der Route ergänzt, 404 war schon dokumentiert.
- gate: build ok (event, server, gateway, cmd/work, cmd/gateway) | vet ok | lint ok (0 issues) | test ok (event + server:
  5090 RUN, 0 SKIP, 0 FAIL; gateway ok inkl. TestOpenAPIRouteDrift; DATABASE_URL als kmuhub_app; neuer DB-Test
  `TestRequireParticipant_OnlyOrganiserAndAttendees` real gelaufen) | migration n.a. | rls-smoke n.a. (keine Tabelle,
  keine Policy) | swagger-cli valid | e2e n.a. (kein Flow nutzt join-token, grep `test/e2e/`)
- coverage: `internal/work/event` 80,6 % -> 80,9 % (kein Coverage-Ziel, `coverage_start: n.a.`)
- mutations-probe: (1) Teilnehmer-Schleife ausgeschaltet -> DB-Test „attendee: expected access“ und gRPC-Subtest
  „attendee gets a token in their own name“ rot. (2) Schluss-Return gewährt Zugriff -> DB-Test „colleague: expected
  ErrEventNotFound“ und gRPC-Subtest „colleague without invitation gets NotFound“ rot. (3) Token-Identität zurück auf
  `req.UserId` -> gRPC-Subtest „attendee gets a token in their own name“ rot (Identity per `lkauth.ParseAPIToken`). Alle
  per Backup zurückgedreht, finaler Tree erneut grün.
- verify vorgaenger: sauber (`65f196a4`: kein Guard, kein Proto, keine neue Tabelle/Route; Migration 000352 mit up/down,
  Backfill + NOT NULL; `linksecret.Expiry` ohne Stub, Berichte-Default auf 365 Tage)
- neue-units: keine
- desktop-aufrufer: POST `/calendar/events/{id}/join-token` -> kein Desktop-Aufrufer, nur der generierte Typ in
  `api/types.ts:2952` (grep `join-?token` in `desktop/src/renderer/src` ohne mocks). Der Video-Beitritt im Desktop läuft
  nicht über diese Route -> nichts bricht. Antwortform unverändert.
- offen: (1) „Organisator“ = `created_by` des Events; ein Kalender-Besitzer oder -Editor, der das Event nicht angelegt
  hat und nicht eingeladen ist, bekommt jetzt 404 — so in der Entscheidung („nur Organisator und Teilnehmer“), aber
  falls Kalender-Delegation gemeint war, ist das eine Erweiterung. (2) Recurring Events: Teilnehmer hängen an der Serie,
  das Token auch (Raumname aus Serien-ID) — unverändert.

## Iteration 19 — fix-offboarding-revokes-caldav-app-passwords — done — 2026-10-25 19:33
- commit: siehe Commit `fix(hr): revoke CalDAV app passwords when an employee is offboarded` (dieser Eintrag)
- gebaut: `PostgresEmployeeRepo.Offboard` setzt in derselben Transaktion `revoked_at = NOW()` auf alle noch aktiven
  `app_specific_passwords` des Leavers (tenant-gescoped). Vorprüfung: `FindActiveByUser` joint schon auf `users.is_active`,
  die Basic-Auth lehnte Offboardete also bereits ab (Unit `fix-caldav-basic-auth-ignores-user-state` war erledigt); der
  Widerruf ist die zweite Schicht, damit eine versehentliche Reaktivierung keine alten Passwörter wiederbelebt.
- gate: build ok (hr, caldav, cmd/gateway) | vet ok | lint ok (0 issues) | test ok (`internal/biz/hr/...` alle ok,
  employee 50 RUN, 0 SKIP; DATABASE_URL als kmuhub_app) | migration n.a. | rls-smoke n.a. (keine Tabelle, keine Policy;
  Test läuft unter kmuhub_app mit Tenant-Kontext) | gateway-Test n.a. (keine Route berührt) | e2e n.a. (kein Flow prüft Offboarding, grep `test/e2e/`)
- coverage: `internal/biz/hr/employee` 76,2 % -> 76,0 % (kein Coverage-Ziel; ein zusätzlicher Fehlerzweig im Repo ohne Test)
- mutations-probe: Bedingung `revoked_at IS NULL` zu `IS NOT NULL` verdreht -> `TestOffboard_RevokesLeaversAppPasswordsOnly`
  rot („leaver still has 2 active app passwords“ und „Basic Auth lookup returns 2 passwords for a re-activated leaver“).
  Zurückgedreht, Diff nur die gewollten 12 Zeilen, finaler Tree erneut grün.
- verify vorgaenger: sauber (`788f475e`: kein Guard, kein Proto, keine Tabelle/Route neu; Spec-Text ergänzt, Teilnehmerprüfung
  über `RequireParticipant` ohne Stub, Tests decken Organisator/Teilnehmer/Fremder)
- neue-units: keine
- desktop-aufrufer: n.a. (keine Validierungs- oder Antwortform-Änderung; Route `POST /hr/employees/{id}/offboard` unverändert)
- offen: (1) Bestand: bereits offboardete Nutzer behalten ihre unwiderrufenen Passwörter in der Tabelle (wirkungslos wegen
  `is_active`-Join, aber nicht widerrufen); keine Backfill-Migration gebaut, da die Unit sie nicht verlangt. (2) Der Widerruf
  ist in der Transaktion, nicht als Folgeschritt: scheitert er, rollt das ganze Offboarding zurück.

## Iteration 20 — fix-video-egress-after-initiator-consent — done — 2026-10-25 19:36
- commit: siehe Commit `fix(video): start the recording egress only after the initiator confirms` (dieser Eintrag)
- gebaut: Variante (a) laut Entscheidung. `StartRecording` legt die Aufnahme nur noch als `pending` an (neuer Status,
  Proto `RECORDING_STATUS_PENDING = 6`, regeneriert) und speichert den Raumnamen (`recordings.room_name`), startet
  aber keinen Egress. `ConfirmInitiatorConsent` claimt per `UPDATE … WHERE status = 'pending' RETURNING room_name`
  (Row-Lock serialisiert parallele Stempel) und startet genau dann den Egress; ein zweiter Stempel claimt nichts und
  startet nichts (200, idempotent). Egress-Fehler -> Aufnahme `failed`, HTTP 500. Toter `PreConsentChecker` samt
  drei Tests entfernt (nie verdrahtet, widersprach dem neuen Ablauf). Migration 000353 (CHECK um `pending`,
  Spalte `room_name`), Spec-Texte für start/initiator-consent ergänzt, 500 dokumentiert.
- gate: build ok (work, server, gateway, cmd/work, cmd/gateway) | vet ok | lint ok (0 issues) | test ok (recording
  87 RUN, 0 SKIP, davon 15 DB-/RLS-Tests real gelaufen; internal/work/... alle ok; server 5015 RUN, 0 SKIP;
  gateway ok inkl. TestOpenAPIRouteDrift; DATABASE_URL als kmuhub_app) | migration ok (353 up/down/up) |
  rls-smoke n.a. (keine neue Tabelle/Policy, nur Spalte + CHECK; Repo-Test läuft unter kmuhub_app mit Tenant-Kontext
  und prüft fremden Tenant -> ErrNotFound) | swagger-cli valid | e2e n.a. (kein Flow nutzt Aufnahmen, grep `test/e2e/`)
- coverage: `internal/work/recording` 80,3 % -> 81,1 % (`coverage_start: ungemessen`)
- mutations-probe: (1) Service ignoriert `claimed` -> `TestConfirmInitiatorConsent_StartsEgressExactlyOnce` rot.
  (2) SQL-Claim ohne `status = 'pending'` -> RealSQL-Test rot („repeat = (room-pending, true)“). (3) StartRecording
  wieder mit Status `active` -> vier Tests rot (u. a. SucceedsWhenAllConsented, StartsEgressExactlyOnce). Alle per
  Backup zurückgedreht, finaler Tree erneut grün.
- verify vorgaenger: sauber (`977046ac`: kein Guard, kein Proto, keine Route/Tabelle; Widerruf tenant-gescoped in der
  Offboard-Transaktion, DB-Test vorhanden)
- neue-units: `decide-recording-participant-consent-before-egress` in BACKLOG-NEXT.yml (blocked, Entscheidung Luke):
  das Teilnehmer-Gate in `StartRecording` ist unerfüllbar (Einwilligungen für eine gerade erzeugte Recording-ID kann
  es nicht geben -> immer ErrConsentPending), und das Desktop erkennt eine aktive Aufnahme nie (`status === 'active'`
  gegen Enum-Zahl auf dem Wire) -> Teilnehmer-Dialog öffnet nie. Auf Prod latent (Egress nicht konfiguriert, 503
  vorher); deshalb nicht an den Kopf von BACKLOG.yml, aber VOR dem Scharfschalten von Egress zu klären.
- desktop-aufrufer: POST `/video/recordings/start` -> `video-client.ts:107` / `CallControls.tsx:129` liest nur
  `recording.id` -> passt. POST `/video/recordings/{id}/initiator-consent` -> `video-client.ts:105`, liest `stamped`
  (unverändert true) -> passt; Ablauf start -> Dialog -> bestätigen startet jetzt den Egress, „Abbrechen“ lässt eine
  `pending`-Zeile ohne Aufnahme. `useActiveMeetingRecording` (`useVideo.ts:284`) sucht `'active'` — pending taucht
  dort korrekt nicht auf (der Vergleich ist unabhängig davon kaputt, siehe neue Unit).
- offen: (1) Abgebrochene `pending`-Zeilen bleiben liegen (keine Aufnahme, kein Medium); Retention-Cleanup greift
  nur `completed`, ein Aufräumen ist nicht gebaut. (2) Down-Migration setzt `pending` auf `failed`; lokal ohne
  Meetings nicht mit echter pending-Zeile durchgespielt. (3) Stop auf `pending` bleibt 412/FailedPrecondition.

## Iteration 21 — review-run17-r1 — done — 2026-10-25 19:51
- commit: siehe Commit `docs(loop): review run 17 rights commits r1` (dieser Eintrag; kein Code-Commit, kein Fund)
- gebaut: Review der Commits seit dem Staging-Commit `65ebf725`, End-SHA **`85d1d7e1`** (r2 beginnt dort). Ergebnis je SHA:
  - `b1ec2498`, `c6bfb10b` (Scans, nur Doku): n.a.
  - `bea77419` work own delete: Guard `Any(tasks:delete, tasks:write)`, Alt-Key erhalten; Besitzprüfung im Service
    (Kommentar: Autor oder `IsAdmin` aus `middleware.IsAdmin`, Zeiteintrag: nur Besitzer), `task_comments`/`time_entries`
    RLS forced; Gateway-Test mit Seed-Token -> ok.
  - `a69fc2e6` rapporte, `494ec76c` + `2d60e1fb` hr: Tests mit Seed-Token decken member nur selbst, manager eigene
    Berichtslinie, nicht fremde Teams, hr_admin/admin Tenant -> ok. Neue Guards additiv (`requireSelfReadOr(...)`).
  - `1c5b1e64` + `8f67a1ff` inbox: `visibleToCaller` (Besitzer/Zugewiesener/Team-Mitglied) in GetByID, List mit
    Team-Filter nur bei Mitgliedschaft, ListMembers bei privatem Team nur für Mitglieder, Guards `Any(inbox:read, …)`;
    Seed 346 geprüft; E2E `TestInboxTeamAndRoutingFlow` (member liest Team-Inbox, entfernter member verliert sie) per
    echtem Login grün -> ok.
  - `7f7faf03` safedial: kein Grant/Guard -> n.a. (Dial-Guard ist Voraussetzung für 347).
  - `d6c12f0b` (347), `a8351f01` (348), `c1eacccd` (349), `b35523cd`, `f897ceb2` (350) E-Mail: alle Guards
    `Any(grob, fein)`, grobe Keys über `forwardedPermissions` an den Hop (lean-Marker zum fehlenden Scope im Hop
    vorhanden); Tests mit Seed-Token decken zweiten member (fremdes Konto/Regel/Vorlage/Nachricht -> 403/404,
    Zeilen unverändert), manager/admin `all` bei Kontakt-Historie laut Entscheidung -> ok.
  - `944d87c4` fuhrpark: Maske in Liste und Export, Viewer aus `x-user-id`, `uuid.Nil` maskiert alles; Update antwortet
    nur mit dem, was der Aufrufer selbst schrieb (Vollersatz); GPS bereits als `decide-…` in NEXT -> ok.
  - `9fe78eb6`, `65f196a4` Share-Tokens, `788f475e` Kalender-Join, `977046ac` CalDAV, `85d1d7e1` Video: keine Grants,
    Sichtbarkeitsgrenzen mit DB-Tests -> ok.
- gate: Seeds 346/347/348/349/350 in einer ROLLBACK-Transaktion gegen die lokale DB (Kopf 353): zweites Up ändert
  0 Zeilen, Down entfernt genau 18 + 9 eigene Grants und die eigenen Keys, 0 fremde, Re-Up stellt sie wieder her;
  Token-Scopes laut DB: member/manager `own`, admin `all`, manager `email:contact_history:read` `all`.
  gateway 5564 RUN, 0 SKIP, ok (inkl. TestOpenAPIRouteDrift; DATABASE_URL als kmuhub_app). Service-Pakete
  inbox/message, email/{rule,template,contactlink}, fuhrpark, work/{event,recording}, biz/hr/employee, wiki, formulare,
  rapporte, linksecret: 804 RUN, 0 SKIP, alle ok bis auf einen fuhrpark-Lauf (s. offen). RLS forced auf allen
  berührten Tabellen geprüft. e2e ok (lokal, 158 s; Inbox, MemberDailyWork, WorkProjectMembers, Calendar,
  MemberHRSelfService; 76 RUN, 0 SKIP).
- coverage: n.a. (Review-Unit, kein Coverage-Ziel)
- mutations-probe: n.a. (kein Code geändert)
- verify vorgaenger: sauber (`85d1d7e1`: Proto mit Regen, Migration 353 up/down, keine neue Route/Guard, Claim per
  `status = 'pending'`)
- neue-units: keine
- desktop-aufrufer: n.a. (keine Validierungs- oder Antwortform-Änderung)
- offen: (1) `TestVehicleBookingOverlapConstraint/two_concurrent_moves…` fiel einmal mit 40P01, Wiederholung grün —
  bekannter Flake, Unit `fix-fuhrpark-concurrent-booking-move-deadlock` steht schon im Backlog. (2) Zeiteinträge
  kann auch admin nicht fremd löschen (`DeleteEntry(..., false)`), Altbestand, kein Leck. (3) Alle neuen Keys aus
  346–350 wirken in Produktion erst nach Re-Login.

## Bilanz Lauf (2026-10-25 20:00)
- iterationen: 21 im Bereich 1-21, davon 21 done, 0 blocked, 0 ohne auswertbare Kopfzeile
- units nach praefix: fix 7 · feat 6 · harden 5 · scan 2 · review 1
- commits nach typ: fix 12 · feat 5 · docs 3 · test 1 (21 seit 65ebf725)
- coverage-delta: internal/email/contactlink 35,0 -> 78,0 · internal/email/template 79,1 -> 87,6 · internal/work/recording 80,3 -> 81,1 · (2 weitere)
- offen mit entscheidungsbedarf: 5 von 21 nicht-leeren offen:-Zeilen (Treffer auf "Luke"/"Entscheidung")
- minuten je iteration: 9,8 (205 gesamt)

## Iteration 22 — seed-leave-types-on-tenant-provisioning — done — 2026-10-25 21:45
- commit: (siehe git log)
- gebaut: `ProvisionTenant` kopiert in derselben Transaktion die zehn Zero-UUID-Urlaubsarten (`hr_leave_types`, `ON CONFLICT DO NOTHING`)
  in den neuen Tenant; DB-Test mit frisch provisioniertem Tenant (urlaub/krank is_system, Nachbar-Tenant sieht 0).
- gate: build ok | vet ok | lint ok | test ok | migration n.a. | rls-smoke ok (Test prüft Nachbar-Tenant als kmuhub_app: 0 Zeilen)
  auth 313 RUN, 0 SKIP; gateway ok (inkl. TestOpenAPIRouteDrift)
- coverage: internal/auth 77,9 % (Bugfix-Unit, Vorher-Wert nicht gemessen) | n.a. (kein Coverage-Ziel)
- mutations-probe: Quell-SELECT im Seed auf `WHERE 1=0 AND ...` gebrochen -> TestProvisionTenant_SeedsSystemLeaveTypes rot (0 statt 10), zurückgedreht, Diff sauber.
- verify vorgaenger: sauber (`ddee3bbb` nur Journal)
- neue-units: keine
- desktop-aufrufer: n.a. (keine Validierungs- oder Antwortform-Änderung)
- offen: `hr_document_categories` nicht angefasst — `ListByTenant` liest die Zero-UUID-Vorlagen laut Unit mit, kein Bruch erwartet (nicht separat geprüft).

## Iteration 23 — fix-produktion-concurrent-booking-move-deadlock — done — 2026-10-25 21:49
- commit: (siehe git log)
- gebaut: `asBookingConflict` bildet neben 23P01 jetzt auch 40P01 (Deadlock zweier gleichzeitiger Umbuchungen) auf `ErrBookingConflict` ab;
  der Verlierer bekommt damit 409 (`produktion_grpc.go:629` -> AlreadyExists) statt 500. `CreateBookingWithLock` nutzt dieselbe Funktion für den INSERT.
  Neuer deterministischer Mapping-Test `TestAsBookingConflict` (23P01, 40P01, fremder Code, nil).
- gate: build ok | vet ok | lint ok | test ok | migration n.a. | rls-smoke n.a.
  produktion 78,9 % Paket-Coverage; DB-Tests real gelaufen, 0 SKIP; Overlap-Test 20x mit `-count=20` grün (ohne -race, lokal nicht möglich);
  gateway ok (inkl. TestOpenAPIRouteDrift).
- coverage: internal/produktion n.a. (Bugfix-Unit, Vorher-Wert nicht gemessen) -> 78,9 %
- mutations-probe: `|| pgErr.Code == pgErrCodeDeadlockDetected` entfernt -> TestAsBookingConflict rot (code 40P01), zurückgedreht, Diff sauber.
- verify vorgaenger: sauber (`db5c75b4`: nur Repo-Insert in derselben Tx plus DB-Test, keine Route/Proto/Guard)
- neue-units: keine
- desktop-aufrufer: n.a. (keine Validierungs- oder Antwortform-Änderung; 409 existierte für 23P01 bereits)
- offen: Der echte Deadlock lässt sich lokal nicht erzwingen; das Mapping ist per Unit-Test fixiert, der Concurrent-Test bleibt der CI-Flake-Wächter (`-race` nur in CI). Dasselbe Muster in `fuhrpark/postgres_repository.go:1340` ist durch `fix-fuhrpark-concurrent-booking-move-deadlock` abgedeckt.

## Iteration 24 — harden-einkauf-po-line-tax-rate-range — done — 2026-10-25 21:52
- commit: (siehe git log)
- gebaut: neuer Validator `rate_fraction` (0..1, höchstens 4 Nachkommastellen = NUMERIC(5,4)) in `internal/validation`; `tax_rate` bei
  AddPOLine und UpdatePOLine trägt `omitempty,rate_fraction`. "19" ergibt jetzt 400 `validation_failed` statt 500 (22003); "0.19" läuft bis zum RPC.
  OpenAPI: Beschreibung an beiden Request-Schemas (Bruch, 0.19 = 19 %).
- gate: build ok | vet ok | lint ok | test ok | migration n.a. | rls-smoke n.a.
  gateway 9652 RUN, 0 SKIP, ok (inkl. TestOpenAPIRouteDrift; DATABASE_URL als kmuhub_app); validation ok; swagger-cli validate ok
- coverage: n.a. (Härtung, kein Coverage-Ziel)
- mutations-probe: `LessThanOrEqual(1)` aus `rate_fraction` entfernt -> TestPOLineTaxRate_FractionOnly rot (19, 1.0001 bei add und update), zurückgedreht, Diff sauber.
- verify vorgaenger: sauber (`eab9ca11`: nur Fehler-Mapping 40P01 -> ErrBookingConflict plus Mapping-Test, keine Route/Proto/Guard/Migration)
- neue-units: keine
- desktop-aufrufer: POST/PATCH `/einkauf/pos/{id}/lines` -> `api/einkauf-types.ts:142,151` (optionales `tax_rate`), kein Aufrufer sendet es (EinkaufDialogs), `einkauf-export.ts:173` liest als Bruch -> passt
- offen: PATCH mit `tax_rate: ""` ist jetzt 400 (vorher durchgereicht und an der DB-Umwandlung gescheitert); `LineItem.tax_rate` im Spec bleibt "19.00"-Beschreibung (generisches Schema, nicht angefasst).

## Iteration 25 — fix-pagination-max-200-for-vermietung-and-inventar-lists — done — 2026-10-25 21:55
- commit: (siehe git log)
- gebaut: Alle Listen von vermietung, inventar, fuhrpark, rapporte, schichten (Service-Ceiling 200) parsen `page_size` jetzt mit `parsePaginationMax(..., listPageSizeMax)`;
  produktion, vertraege, einkauf (Service-Ceiling 100) mit neuer Konstante `listPageSizeMaxSmall` — zu große Werte werden geklemmt statt still auf den Default zurückzufallen.
  OpenAPI: `maximum` 100 -> 200 an 31 Inline-Parametern der 200er-Module (Ceilings je Service-Code geprüft).
  Tests: Stub-gRPC für Vermietung `ListObjects` (""/25/200/500) und Produktion `ListOrders` (""/100/500) sowie Source-Guard gegen `parsePagination(r,` in allen Branchen-Route-Dateien.
- gate: build ok | vet ok | lint ok (0 issues) | test ok | migration n.a. | rls-smoke n.a.
  gateway-Paket komplett grün inkl. TestOpenAPIRouteDrift; swagger-cli validate ok; kein Service-Paket angefasst.
- coverage: n.a. (kein Coverage-Ziel; internal/gateway Start 77,8 %, nicht neu gemessen)
- mutations-probe: Vermietung `HandleListObjects` auf `parsePagination(r, 1, 50)` zurückgedreht -> TestBranchPagination_VermietungObjectsClampToTwoHundred und ..._NoPlainParsePaginationInBranchRoutes rot, zurückgedreht, Diff sauber.
- verify vorgaenger: sauber (`12a0add8`: Validator + Spec-Beschreibung, keine neue Route/Proto/Guard/Migration)
- neue-units: keine
- desktop-aufrufer: GET `/vermietung/objects|rentals`, `/inventar/items|locations` -> `VermietungPage.tsx:643/644`, `InventarPage.tsx:796/797` (page_size 200/500/100) -> passt jetzt (200 statt 50 Zeilen; 500 wird auf 200 geklemmt, Rentals bleibt bei >200 Treffern abgeschnitten)
- offen: `VermietungPage.tsx:644` fragt 500 Mieten an, bekommt höchstens 200 — echte Abhilfe wäre ein Desktop-Paging oder höheres Service-Ceiling (Entscheidung Luke). Die Produktion-Listen werden jetzt auf 100 geklemmt (vorher Default 20 bei >100).

## Iteration 26 — fix-recurring-concurrent-generate-test-flake — done — 2026-10-25 21:58
- commit: (siehe git log)
- gebaut: `TestGenerate_RealSQL_ConcurrentRunsProduceExactlyOneInvoice` nutzt einen `barrierRepo` (GetByID als Rendezvous, 10 s Timeout): beide Racer laden den Plan,
  bevor einer `Update` ausführt, sehen also dieselbe Periode. Kein Produktcode geändert. Ursache der CI-Flake (2 Rechnungen) war der legitime Claim der Folgeperiode, nicht ein ON-CONFLICT-Bruch.
- gate: build n.a. (nur Testdatei) | vet ok | lint ok (0 issues) | test ok | migration n.a. | rls-smoke n.a.
  Test 30x mit `-count=30` grün (ohne -race, lokal nicht möglich); Paket komplett grün, 0 SKIP.
- coverage: n.a. (Test-Flake, Paket biz/recurring 88,5 % nachher, Vorher nicht gemessen)
- mutations-probe: Barriere auf 1 Partei gestellt und Racer 2 um 500 ms verzögert (= die Flake-Reihenfolge) -> Test rot (zwei Rechnungen), zurückgedreht, Diff sauber.
- verify vorgaenger: sauber (`371684e5`: Pagination-Klemmung in Route-Dateien plus Test und Spec-`maximum`, keine neue Route/Proto/Guard/Migration)
- neue-units: decide-recurring-generate-future-period (BACKLOG-NEXT.yml, blocked — Produktfrage "Folgeperiode vor Fälligkeit abrechenbar?")
- desktop-aufrufer: n.a. (keine Validierungs- oder Antwortform-Änderung)
- offen: `-race` nur in CI; Barriere ist rein testseitig. Produktfrage siehe decide-Unit.

## Iteration 27 — e2e-local-gate-check — done — 2026-10-25 22:00
- commit: (siehe git log)
- gebaut: nichts am Code — lokales E2E-Gate in der Loop-Umgebung (`claude -p`, bypassPermissions) abgenommen. Volle Suite
  `bash .planning/backend-block/loop/hooks/e2e-local.sh` ohne Regex: Exit 0, 26 Top-Level-Tests (258 RUN inkl. Subtests), 26 PASS, 0 SKIP, 0 FAIL.
  Laufzeit gesamt 195 s: Infra 3 s, Migration 3 s, Build der 21 Binaries 150 s (Stamp neu, HEAD hatte sich geändert), Service-Start ~10 s, `go test` 34,5 s.
  Speicherspitze der 21 Service-Prozesse (tasklist-Summe, 2-s-Takt): ~912 MB (934 220 KB); Docker-Container und `go test` nicht mitgezählt.
- gate: build ok (im Skript) | vet n.a. | lint n.a. | test n.a. | e2e ok (lokal, 195 s, davon 150 s Build) | migration ok (frische DB, Kopf ohne Fehler) | rls-smoke n.a.
- coverage: n.a. (Gate-Abnahme)
- mutations-probe: n.a. (kein Code)
- verify vorgaenger: sauber (`e8ff1134`: nur `service_db_test.go` plus Backlog/NEXT/Journal, kein Produktcode, keine Route/Proto/Guard/Migration)
- neue-units: keine
- desktop-aufrufer: n.a. (keine Validierungs- oder Antwortform-Änderung)
- offen: Aufräumen belegt — nach dem Lauf keine `kmuhub-e2e-*`-Prozesse (tasklist leer) und keine `kmuhub-e2e`-Container (`docker ps -a` leer).
  Regel 1 gilt damit für den Rest des Laufs; ein Lauf mit gültigem Stamp spart die 150 s Build. Services laufen weiter als Superuser `kmuhub_test` (lean-Marker im Skript), RLS wird im E2E also nicht bewiesen.

## Iteration 28 — review-run17-r2 — done — 2026-10-25 22:05
- commit: siehe Commit `docs(loop): review run 17 rights commits r2` (dieser Eintrag; kein Code-Commit, kein Fund)
- gebaut: Review der Commits seit r1-End-SHA `85d1d7e1`, End-SHA **`a1c8f05e`** (r3 beginnt dort). Ergebnis je SHA:
  - `5a57e13e`, `ddee3bbb`, `a1c8f05e` (nur Loop-Doku): n.a.
  - `db5c75b4` Urlaubsarten beim Provisioning: kein Grant/Guard/Scope; Insert in derselben Tx mit `tenant.ID`, Quelle nur
    Zero-UUID-Vorlagen. RLS `hr_leave_types` enabled+forced, Policy liest eigener Tenant ODER Zero-UUID, schreibt nur eigenen
    Tenant. Smoke als `kmuhub_app`: fremder Tenant sieht 10 Zeilen (nur Vorlagen, 1 Tenant-ID), eigener Tenant 20 (eigene Kopie
    plus Vorlagen) -> ok. `TestProvisionTenant_SeedsSystemLeaveTypes` und `TestRLS_ProvisionedTenant_IsIsolatedFromExistingData`
    als `kmuhub_app` grün -> ok.
  - `eab9ca11` produktion 40P01 -> 409, `12a0add8` tax_rate 0..1, `371684e5` Seitengrößen-Klemmung: Diff gegen
    `RequirePermission|role_permissions|scope|tenant_id|GetTenantID` gegrept, null Treffer außerhalb von db5c75b4 -> keine
    Rechte- oder Sichtbarkeitsänderung; Pagination klemmt nur `page_size`, Filter/Owner-Scopes unverändert -> n.a.
  - `e8ff1134` recurring: nur Testdatei -> n.a.
- gate: auth (2 Provisioning-Tests, 0 SKIP) ok; gateway `TestBranchPagination*|TaxRate*|TestOpenAPIRouteDrift` ok (24 PASS-Zeilen
  inkl. Subtests); DATABASE_URL als `kmuhub_app`. Keine Seed-Migration im Prüfbereich -> Up/Down-Prüfung n.a. | rls-smoke ok (hr_leave_types)
- coverage: n.a. (Review)
- mutations-probe: n.a. (kein Code)
- verify vorgaenger: sauber (`a1c8f05e`: nur BACKLOG/JOURNAL, kein Code)
- neue-units: keine
- desktop-aufrufer: n.a. (keine Validierungs- oder Antwortform-Änderung)
- offen: Block B hat keine Rechte-Commits erzeugt; der Prüfbereich von r2 war deshalb dünn. Die Scan-Fix-Units sind in r1 geprüft.

## Iteration 29 — e2e-produktion-member-flow — done — 2026-10-25 22:06
- commit: (siehe git log)
- gebaut: `backend/test/e2e/produktion_test.go`, `TestProduktionMemberFlow` mit sieben Subtests gegen den echten Seed (admin, manager, zwei member):
  manager legt Auftrag (Duplikat -> 409) und Maschine an, partielles PATCH; admin legt Plan und Arbeitsschritte an (manager dort 403, IST: `plan:write`/`workstep:write` nur admin);
  member liest Aufträge/Maschinen/Schritte, meldet Schritt `in_progress` -> `completed`, startet und schließt den Auftrag (Doppelstart 409), legt eine Qualitätsprüfung an;
  elf member-Verbote (Auftrag anlegen/ändern/löschen/abbrechen, Maschine, Schritte, Plan, alle Buchungen) für beide member 403 und Auftrag danach unverändert;
  manager storniert/löscht geplante Aufträge, abgeschlossener Auftrag löschen -> 409; admin: Überschneidung beim Anlegen 409, zwei parallele Umbuchungen auf dasselbe Fenster -> genau 200 und 409.
  Bodies aus `produktion-client.ts`/`produktion-types.ts`. Buchungen laufen als admin, weil kein anderer Preset `booking:read/write` hält (Seed, nicht Auslassung).
- gate: build n.a. (nur Testdatei) | vet ok (`go vet -tags=e2e ./test/e2e/`) | lint ok für die neue Datei (4 Altfunde in auth_test.go/helpers_test.go, nicht meine) | test n.a. | e2e ok (lokal, `TestProduktionMemberFlow` 2,9 s Test, 58 s Lauf mit Cache, Erstlauf 154 s mit Build) | migration n.a. | rls-smoke n.a.
- coverage: n.a. (E2E-Unit, kein Coverage-Ziel)
- mutations-probe: Guard `orderComplete` in `gateway/route_produktion.go` von `complete` auf `delete` gestellt -> Test rot (member-Abschluss 403), zurückgedreht (`git checkout`), Diff sauber, danach wieder grün.
- verify vorgaenger: sauber (`3bbcd950`: nur BACKLOG/JOURNAL, kein Code)
- neue-units: fix-produktion-workstep-order-path-check (Ende BACKLOG.yml; kein Tenant-Leck, Pfad `{orderId}` wird bei PATCH/DELETE eines Schritts ignoriert, im E2E als `IST`-Log festgehalten); decide-desktop-produktion-booking-plan-envelope (BACKLOG-NEXT.yml, blocked: Gateway `{booking}`/`{plan}`/`{overview}` gegen flache FE-Typen, kein Screen ruft es heute)
- desktop-aufrufer: n.a. (keine Validierungs- oder Antwortform-Änderung; Abgleich der Bodies und der Antwortformen siehe neue-units)
- offen: Rechte-Lücken sind der Ist-Seed und gehören zu `decide-member-manager-module-defaults` (member ohne Buchungen, manager ohne Plan/Buchungen/Schritte-Anlage). Kein Totalausfall gefunden. Der CI-Job `e2e` muss die Datei im Morgen-PR bestätigen (Flag `modules.produktion` ist laut Skript gesetzt).

## Iteration 30 — e2e-rapporte-member-fill-manager-approve-flow — done — 2026-10-25 22:14
- commit: (siehe git log)
- gebaut: `backend/test/e2e/rapporte_test.go`, `TestRapporteMemberFlow` mit sieben Subtests gegen den echten Seed (admin, manager, zwei member):
  admin legt Vorlage an (member 403 auf Anlegen/Löschen, liest sie); member A legt Rapport an (fremdes `author_id` wird auf den Aufrufer umgeschrieben), PATCH, zwei Positionen aus der Vorlage, Position ändern, Signatur, PDF-Export, Stats;
  member B sieht/ändert nichts von A (Liste, `author_id`-Query, GET/PATCH/Submit/Lines/Export 404, Delete 403, Rapport danach unverändert, Pending-Liste ohne A);
  A reicht ein (Doppel-Submit, Edit, neue Position danach 409; Approve/Reject als member 403); manager sieht ihn in `/pending`, Reject ohne Notiz 400, Reject mit Notiz (gespeicherter Reviewer = Aufrufer, nicht `reviewer_id` im Body), Approve auf rejected/Entwurf 409;
  A korrigiert, reicht erneut ein (atomares Submit), manager genehmigt, zweites Approve 409, Edit/Delete-Line nach Freigabe 409, Lesen bleibt. Bodies aus `rapporte-client.ts`/`rapporte-types.ts`.
- gate: build n.a. (nur Testdatei) | vet ok (`go vet -tags=e2e ./test/e2e/`) | lint ok (golangci-lint `--build-tags e2e`, null Funde in der neuen Datei) | test n.a. | e2e ok (lokal, `TestRapporteMemberFlow` 2,6 s Test; Erstlauf 150 s mit Build, dann 6 s) | migration n.a. | rls-smoke n.a.
- coverage: n.a. (E2E-Unit, kein Coverage-Ziel)
- mutations-probe: Approve-Guard in `gateway/route_rapporte.go` von `approve` auf `read` gestellt -> Test rot (member-Approve 200 statt 403, Folgefehler in vier Subtests), zurückgedreht (`git checkout`), Diff sauber.
- verify vorgaenger: sauber (`b472f6b4`: nur `produktion_test.go` plus Backlog/NEXT/Journal, kein Produktcode, keine Route/Proto/Guard/Migration)
- neue-units: fix-rapporte-empty-list-wire-shape (Ende BACKLOG.yml; leere Listen sind `{}` statt `{reports:[],total:0}`, FE fängt es mit `?? []` ab, kein Totalausfall)
- desktop-aufrufer: n.a. (keine Änderung an Validierung oder Antwortform; Bodies und Antwortformen im Test gegen `rapporte-client.ts` abgeglichen, Abweichung nur die leere Liste, siehe neue-units)
- offen: IST des Seeds, kein Fund: Approve/Reject sind nicht an die Berichtslinie gebunden (jeder manager des Tenants prüft jeden eingereichten Rapport); member hat kein `report:delete` (kein Entwurf löschbar); `/stats` liefert tenantweite Zähler auch an member (own-Scope wirkt nicht); Vorlagen-Anlage nur admin, das Desktop hat keinen Vorlagen-Aufrufer. Das gehört zu `decide-member-manager-module-defaults`. CI-Job `e2e` muss die Datei im Morgen-PR bestätigen.

## Iteration 31 — e2e-fuhrpark-member-booking-license-damage-flow — done — 2026-10-25 22:22
- commit: (siehe git log)
- gebaut: `backend/test/e2e/fuhrpark_test.go`, `TestFuhrparkMemberFlow` mit neun Subtests gegen den echten Seed (admin, manager, zwei member):
  manager legt Poolfahrzeug an (member 403, liest es); member A bucht (Body-`user_id` = member B wird auf den Aufrufer umgeschrieben), Doppelbuchung identisch/Überlappung Ende/Anfang 409, umgekehrtes Fenster 400, Anschlussbuchung von B 201;
  B sieht trotz `user_id=A` im Query nur die eigene Buchung und kann fremde per PATCH/Status/DELETE nicht ändern (404, Buchung danach unverändert); A verschiebt/storniert/löscht eigene (Kollision 409, unbekannter Status 400, storniertes Fenster wird frei, zweites DELETE 404);
  Schaden: Body-`reported_by` wird auf den Aufrufer gesetzt, nur Melder und admin sehen ihn in beiden Listen (B und manager nicht), Update/Resolve für member und manager 403, admin 200; Führerscheinkontrolle: `driver_id` wird auf den Aufrufer gesetzt, Pflichtfelder 400, B sieht/ändert/löscht fremde nicht (404), A ändert und löscht eigene.
  Bodies: Schaden aus `fuhrpark-types.ts`, Buchung und Führerschein aus den Gateway-Request-Structs (das Desktop hat dafür keinen Aufrufer, nur die generierte `types.ts` listet die Routen).
- gate: build n.a. (nur Testdatei) | vet ok (`go vet -tags=e2e ./test/e2e/`) | lint ok (golangci-lint `--build-tags e2e`, null Funde in der neuen Datei) | test n.a. | e2e ok (lokal, `TestFuhrparkMemberFlow` 2,6 s Test, 9 Subtests PASS; Erstlauf 155 s mit Build, dann ~25 s) | migration n.a. | rls-smoke n.a.
- coverage: n.a. (E2E-Unit, kein Coverage-Ziel)
- mutations-probe: in `gateway/route_fuhrpark.go` (`HandleCreateVehicleBooking`) die Umschreibung `userID = *owner` entfernt -> Test rot (Buchung landet bei B statt A, drei Subtests rot), zurückgedreht (`git checkout`), Diff sauber, danach wieder grün.
- verify vorgaenger: sauber (`8eea0a06`: nur `rapporte_test.go` plus Backlog/Journal, kein Produktcode, keine Route/Proto/Guard/Migration)
- neue-units: keine
- desktop-aufrufer: n.a. (keine Validierungs- oder Antwortform-Änderung; Bodies und Antwortformen gegen `fuhrpark-types.ts` abgeglichen, passt; Buchungen/Führerscheine haben kein Desktop-Gegenstück)
- offen: IST des Seeds, kein Fund: der manager ist bei Schäden auf own begrenzt (sieht fremde Schäden nur als admin), Update/Resolve von Schäden nur admin; Fahrzeuganlage `vehicle:manage` (manager, admin). Listen-Antworten waren in allen Fällen mit Zeilen gefüllt, die Leerform `{}` wurde hier nicht geprüft (siehe `fix-rapporte-empty-list-wire-shape`). Der CI-Job `e2e` muss die Datei im Morgen-PR bestätigen (das Flag `modules.fuhrpark` lief lokal; in `ci.yml` nicht per Grep auffindbar, der Dienst ist im Start-Loop enthalten).

## Iteration 32 — e2e-fuhrpark-trip-log-private-masking-flow — done — 2026-10-25 22:30
- commit: (siehe git log)
- gebaut: `backend/test/e2e/fuhrpark_triplog_test.go`, `TestFuhrparkTripLogPrivateMaskingFlow` mit neun Subtests gegen den echten Seed (admin, manager, zwei member):
  manager legt Fahrzeug an; Fahrer (member) legt Geschäftsfahrt und zwei Privatfahrten an (Desktop-Body ohne `driver_user_id`, und mit verknüpftem Konto); Validierung (Geschäftsfahrt ohne Zweck 400, km rückwärts 400, fremdes Konto 400, keine Restzeile);
  Fahrer sieht Geschäfts- und verknüpfte Privatfahrt voll (Fahrzeugliste und Tenant-Liste); anderer member, manager und admin sehen beide Privatfahrten nur als Label „Privatfahrt“ (Ziel, Notiz, Partner leer, km und Fahrername bleiben) in beiden Listen;
  CSV-Export für member/manager/admin: kein Ziel und keine Notiz, beide Privatfahrten als Label, Geschäftsfahrt voll, Kopfzeile plus drei Zeilen, alle km-Stände vorhanden; Format `xml` 400; PATCH/DELETE für Fahrer, member und manager 403, Zeile danach unverändert; admin-Korrektur mit vollem Body 200.
  Bodies aus `fuhrpark-client.ts` (`CreateTripLogRequest`, `UpdateTripLogRequest`).
- gate: build n.a. (nur Testdatei) | vet ok (`go vet -tags=e2e ./test/e2e/`) | lint ok (golangci-lint `--build-tags e2e`, null Funde in der neuen Datei) | test n.a. | e2e ok (lokal, `TestFuhrparkTripLogPrivateMaskingFlow` 2,4 s Test, 10 Zeilen PASS; Erstlauf 147 s mit Build, dann 27 s) | migration n.a. | rls-smoke n.a.
- coverage: n.a. (E2E-Unit, kein Coverage-Ziel)
- mutations-probe: in `fuhrpark/service.go` (`ExportTripLogs`) den Aufruf `maskPrivateTrips(...)` auskommentiert -> Subtest „export keeps every trip …“ rot (Ziel im CSV), zurückgedreht (`git checkout`), Diff sauber.
- verify vorgaenger: sauber (`6129ca59`: nur `fuhrpark_test.go` plus Backlog/Journal, kein Produktcode, keine Route/Proto/Guard/Migration)
- neue-units: fix-fuhrpark-trip-log-driver-defaults-to-caller (Ende BACKLOG.yml; das Desktop-Formular sendet kein `driver_user_id`, die Privatfahrt ist damit auch für den eigenen Fahrer maskiert); fix-fuhrpark-trip-log-partial-patch-overwrites-row (Ende BACKLOG.yml; PATCH mit nur `purpose` setzt Datum, km, Orte und `is_private` zurück, nur admin, im Desktop kein Aufrufer in einer Komponente). Beide im E2E als IST-Subtest festgehalten; kein Leck über Tenants, kein Totalausfall, daher nicht an den Kopf.
- desktop-aufrufer: n.a. (keine Validierungs- oder Antwortform-Änderung; Bodies und Antwortformen gegen `fuhrpark-client.ts` abgeglichen: Create/List passen, `driver_user_id` fehlt im Desktop-Typ, siehe neue-units)
- offen: IST des Seeds, kein Fund: Fahrtenbuch-Liste ist tenantweit für jede Rolle (kein own-Scope), member und manager haben `fuhrpark:trip:create|read`, nie `:write` (Korrektur nur admin). Die Antwort von Create/Update ist nicht maskiert (der Aufrufer bekommt die Zeile, die er selbst geschrieben hat). Frage an Luke zu `fix-fuhrpark-trip-log-driver-defaults-to-caller`: Manager, der für andere Fahrer erfasst. Der CI-Job `e2e` muss die Datei im Morgen-PR bestätigen (`modules.fuhrpark` lief lokal).

## Iteration 33 — e2e-vermietung-member-inspection-flow — done — 2026-10-25 22:39
- commit: (siehe git log)
- gebaut: `backend/test/e2e/vermietung_test.go`, `TestVermietungMemberInspectionFlow` mit neun Subtests gegen den echten Seed (admin, manager, zwei member):
  manager legt Mietobjekt an (member 403 auf Anlegen/PATCH/DELETE, liest es), manager und member A buchen je ein Fenster, Überlappung (Anfang, Ende, umschließend) 409, rückwärts 400, unbekanntes Objekt 404, Verfügbarkeitsprobe meldet belegt;
  member A legt das Übergabeprotokoll mit dem Desktop-Body an (`kind`, `notes` mit `[by:…]`, `photo_urls`), doppelte Art 409, unbekannte Art 400, unbekannte Vermietung 404; Protokoll ist für member und manager schreibgeschützt (PATCH 403, Inhalt unverändert), admin korrigiert 200;
  member B liest das Protokoll und legt die Rückgabe der fremden Vermietung an; Start/Ende durch member (Reihenfolge erzwungen: Ende auf reserviert 409, doppelter Start 409, B beendet die Vermietung von A); member ändert eine fremde Buchung (Notiz 200, Überlappung 409, Fenster rückwärts 400), Signatur-PUT für member und manager 403, Storno member 403 und manager 204, freigewordenes Fenster buchbar. Bodies aus `vermietung-client.ts` / `ZustandsprotokollDialog.tsx`.
- gate: build n.a. (nur Testdatei) | vet ok (`go vet -tags=e2e ./test/e2e/`) | lint ok für die neue Datei (golangci-lint `--build-tags e2e`: vier Altfunde in `auth_test.go`/`helpers_test.go`, null in `vermietung_test.go`) | test n.a. | e2e ok (lokal, `TestVermietungMemberInspectionFlow` 2,6 s Test, 10 Zeilen PASS; Erstlauf 143 s mit Build, dann ~6 s) | migration n.a. | rls-smoke n.a.
- coverage: n.a. (E2E-Unit, kein Coverage-Ziel)
- mutations-probe: in `gateway/route_vermietung.go` Zeile 108 den Guard von `PATCH /inspections/{id}` von `write` auf `read` gestellt -> Test rot (member-PATCH nicht mehr 403), zurückgedreht (`git checkout`), Diff sauber, danach wieder grün.
- verify vorgaenger: sauber (`f20b2a02`: nur `fuhrpark_triplog_test.go` plus Backlog/Journal, kein Produktcode, keine Route/Proto/Guard/Migration)
- neue-units: keine
- desktop-aufrufer: n.a. (keine Validierungs- oder Antwortform-Änderung; Bodies und Antwortformen gegen `vermietung-client.ts`/`vermietung-types.ts` abgeglichen: Objekte, Buchungen, Protokolle, Start/Ende passen; `performed_by`/`checklist` sendet das Desktop nicht)
- offen: IST des Seeds, kein Fund: vermietung hat keinen own-Scope (jeder member liest und fährt jede Vermietung, legt die fremde Protokoll-Art an); `performed_by` wird nicht auf den Aufrufer umgeschrieben (member kann einen anderen als Prüfer eintragen, das Desktop sendet das Feld nicht); Protokoll-PATCH und Signatur-PUT nur admin (`inspection:write`, `rental:write` Altkeys, das Desktop hat dafür keinen Aufrufer in einer Komponente); Storno (`DELETE`) manager/admin. Gehört zu `decide-member-manager-module-defaults`. Die Leerform der Listen (`{}`) wurde nur für die Protokollliste der unberührten Vermietung berührt und vom Test toleriert. CI-Job `e2e` muss die Datei im Morgen-PR bestätigen (`COSMI_MODULE_VERMIETUNG_ENABLED` steht in `ci.yml`).

## Iteration 34 — e2e-vermietung-rental-lists-and-calendar-flow — done — 2026-10-25 22:46
- commit: (siehe git log)
- gebaut: `backend/test/e2e/vermietung_lists_test.go`, `TestVermietungListsAndCalendarFlow` mit sieben Subtests gegen den echten Seed (manager, zwei member): manager legt 55 Objekte an, die beiden member buchen abwechselnd je eine Vermietung (März eines Zufallsjahres ab 2200, damit Wiederholungen nichts verschieben);
  Desktop-Aufrufe wie in `VermietungPage.tsx:643/644`: Objektliste `page_size=200` liefert 51–200 Zeilen, Suche auf den Lauf findet genau 55; Mietliste `page_size=500` wird auf 200 geklemmt, `total` bleibt voll, Fenster `from`/`to` liefert genau 55, member B liest die Buchungen von member A; Paging 20/20/15, keine Doppelten, Seite 4 leer mit erhaltenem Total;
  Monatskalender (`month`, `year` wie `getRentalCalendar`): 55 Einträge mit `object_name` und genau einer Buchung je Objekt, Nachbarmonate leer als Array (nicht `null`), `object_id` des Desktop-Params wird ignoriert; Storno: member 403, manager 204, danach weg aus Kalender, Liste und Total, GET 404.
- gate: build n.a. (nur Testdatei) | vet ok (`go vet -tags=e2e ./test/e2e/`) | lint ok (golangci-lint `--build-tags e2e`, null Funde in der neuen Datei) | test n.a. | e2e ok (lokal, `TestVermietungListsAndCalendarFlow` 2,5 s Test; Erstlauf 3 min mit Build, danach ~25 s) | migration n.a. | rls-smoke n.a.
- coverage: n.a. (E2E-Unit, kein Coverage-Ziel)
- mutations-probe: in `gateway/route_vermietung.go` Zeile 439 (`HandleListRentals`) `listPageSizeMax` durch `50` ersetzt -> Subtest „rental list with the desktop's page_size 500" rot (50 statt 51..200 Zeilen), zurückgedreht (`git checkout`), Diff sauber, danach wieder grün.
- verify vorgaenger: sauber (`d0000132`: nur `vermietung_test.go` plus Backlog/Journal, kein Produktcode, keine Route/Proto/Guard/Migration)
- neue-units: keine
- desktop-aufrufer: n.a. (keine Validierungs- oder Antwortform-Änderung; Listen und Kalender gegen `vermietung-types.ts` abgeglichen: `{objects,total}`, `{rentals,total}`, `{entries:[{object_id,object_name,rentals}]}` passen; `useRentalCalendar` hat keine Komponente als Aufrufer)
- offen: IST des Seeds, kein Fund: „Stornieren" ist ein Hard-Delete (`DELETE /rentals/{id}` löscht die Zeile, kein Status `cancelled`; die Route kommentiert das als „soft label"), der Kalender-Filter auf `cancelled` ist damit toter Code; der Kalender ignoriert `object_id`; die Leerform der Listen (`{}` statt `[]`) wurde nicht erzwungen, das Desktop liest mit `?? []`. Fremder Mandant nicht geprüft: der E2E-Stack hat keinen zweiten Tenant. member sieht alle Buchungen des Tenants (kein own-Scope), gehört zu `decide-member-manager-module-defaults`. CI-Job `e2e` muss die Datei im Morgen-PR bestätigen.

## Iteration 35 — e2e-vertraege-flow — done — 2026-10-25 22:55
- commit: (siehe git log)
- gebaut: `backend/test/e2e/vertraege_test.go`, `TestVertraegeMemberFlow` mit zehn Subtests gegen den echten Seed (manager, zwei member): Anlegen mit dem Body des Anlege-Dialogs (member 403, doppelte Nummer 409, Titel/Nummer/Typ/Datum fehlerhaft 400), Lesen als member (Get, gefilterte Liste, leere Kind-Listen als `[]`, 400/404 auf Fremd-ID);
  Bearbeiten wie der Dialog (Titel/Typ/Daten, `clear_ends_on`, `document_url`-only-Patch lässt den Rest), Partei und Erinnerung anlegen/abbrechen/löschen (member überall 403, Zeile unverändert), Export und Verlauf als member, Signatur-PUT (member 403, falsches Präfix 400, doppelt 409), Kündigung `{status:"terminated"}` legt genau einen Verlaufseintrag an, Löschen nur im Entwurf (409 sonst), Liste ohne `page_size` = 20 Zeilen mit Gesamtzahl, Seite 2 ohne Dubletten, `page_size=500` auf 100 geklemmt. Bodies aus `vertraege-client.ts` / `VertraegePage.tsx`.
- gate: build n.a. (nur Testdatei) | vet ok (`go vet -tags=e2e ./test/e2e/`) | lint ok (golangci-lint `--build-tags e2e`, null Funde in der neuen Datei) | test n.a. | e2e ok (lokal, `TestVertraegeMemberFlow` ~2,2 s Test, 10 Subtests PASS; Erstlauf 2 min 34 s mit Build) | migration n.a. | rls-smoke n.a.
- coverage: n.a. (E2E-Unit, kein Coverage-Ziel)
- mutations-probe: in `gateway/route_vertraege.go` die Definition von `contractCreate` auf `RequirePermission("vertraege:contract","read")` gestellt -> Test rot (member-POST gab 201 statt 403), zurückgedreht (`git checkout`), Diff sauber, danach wieder grün.
- verify vorgaenger: sauber (`9c0a881a`: nur `vermietung_lists_test.go` plus Backlog/Journal, kein Produktcode, keine Route/Proto/Guard/Migration)
- neue-units: fix-vertraege-enum-and-reference-validation-500 (Ende BACKLOG.yml; Enum-Werte und unbekannte Referenzen geben 500 statt 400/404, als IST-Subtest festgehalten); fe-vertraege-list-asks-for-first-page-only (Ende BACKLOG-NEXT.yml; Desktop-Liste sendet kein `page_size`, ab dem 21. Vertrag fehlen die übrigen, Desktop-Code gesperrt). Kein Leck über Tenants, kein Totalausfall, daher nicht an den Kopf.
- desktop-aufrufer: n.a. (keine Validierungs- oder Antwortform-Änderung; Bodies und Antwortformen gegen `vertraege-client.ts`/`vertraege-types.ts` abgeglichen: `{contract}`, `{contracts,total}`, `{party}`, `{parties}`, `{reminder}`, `{reminders}` passen; Verlauf `{items,total}` und Signatur-PUT haben keinen Aufrufer im Client)
- offen: IST des Seeds, kein Fund: member hat nur die drei `:read`-Keys, alles Schreibende manager/admin; Lesen ist mandantenweit (kein own-Scope), gehört zu `decide-member-manager-module-defaults`. Der Anlege-Dialog sendet keinen Status, neue Verträge bleiben im Backend `draft` (lokaler Store zeigt `active`), ein Statuswechsel `draft -> active` hat keinen Aufrufer; Statuswechsel sind frei (kein Übergangsautomat). Fremder Mandant nicht geprüft (der E2E-Stack hat nur einen). Kündigungsfrist/Verlängerung des Desktops (`noticePeriodDays`, `renewal`) liegen nur im lokalen Store, das Backend hat dafür keine Felder. CI-Job `e2e` muss die Datei im Morgen-PR bestätigen (`COSMI_MODULE_VERTRAEGE_ENABLED` steht in `ci.yml`).

## Iteration 36 — e2e-formulare-flow — done — 2026-10-25 23:05
- commit: (siehe git log)
- gebaut: `backend/test/e2e/formulare_test.go`, `TestFormulareMemberFlow` mit zwölf Subtests gegen den echten Seed (manager, zwei member): Formular mit Desktop-Body anlegen (member 403, Titel/`closed` 400, Felder als Array zurück), Liste/Suche/Lesen als member (404/400 auf Fremd-ID), member 403 auf PATCH/Duplizieren/Löschen/Share-Link/Webhook-Schreiben; manager veröffentlicht, dupliziert, löscht die Kopie; Webhook: member liest ihn mit maskiertem Secret, Schreiben 403;
  member A reicht ein, member B und manager sehen die Einreichung, B setzt den Status und exportiert CSV; Share-Link: manager legt an (Ablauf immer gesetzt, Vergangenheit/`maxSubmissions:-1` 400, privates Formular 409, member 403), öffentliches Ausfüllen ohne Token-Login (Basis64-Body 201, nur `submission_id` auf der Leitung, unbekanntes Token 404, Zähler steigt, `submittedBy` leer), Kontingent 1 stoppt, Widerruf (member 403, doppelt 200, fremde ID 404, Token danach 404, Liste zeigt `disabled`), `isPublic:false` tötet ausgegebene Links. Bodies aus `formulare-client.ts` / `formulare-types.ts`.
- gate: build n.a. (nur Testdatei) | vet ok (`go vet -tags=e2e ./test/e2e/`) | lint ok (golangci-lint `--build-tags e2e`, null Funde in der neuen Datei) | test n.a. | e2e ok (lokal, `TestFormulareMemberFlow` 1,9 s Test; Erstlauf 2 min 33 s mit Build, dann ~15 s) | migration n.a. | rls-smoke n.a.
- coverage: n.a. (E2E-Unit, kein Coverage-Ziel)
- mutations-probe: in `gateway/route_formulare.go` den Guard `shareManage` auf `RequirePermission("formulare:submissions","read")` gestellt -> Test rot (member-GET der Share-Links 200 statt 403, member-Widerruf 200 statt 403), zurückgedreht (`git checkout`), Diff sauber, danach wieder grün (Neubau).
- verify vorgaenger: sauber (`95309497`: nur `vertraege_test.go` plus Backlog/Journal, kein Produktcode, keine Route/Proto/Guard/Migration)
- neue-units: fix-formulare-create-submission-checks-schema (Ende BACKLOG.yml; `Service.CreateSubmission` lädt das Formular nie: unbekannte ID 500, gelöschtes Formular nimmt die Einreichung an und stößt Webhooks an; `submittedBy` wird ungeprüft übernommen); fix-formulare-public-submit-accepts-json-answers (Ende BACKLOG.yml; `answers` der öffentlichen Route ist `[]byte`, ein JSON-Objekt gibt 400, nur base64 geht). Beide als IST im E2E festgehalten; kein Leck über Tenants, kein Totalausfall (die öffentliche Ausfüllseite existiert nicht, die angemeldete Route hat im Desktop keinen Aufrufer), daher nicht an den Kopf.
- desktop-aufrufer: n.a. (keine Validierungs- oder Antwortform-Änderung; Bodies und Antwortformen gegen `formulare-client.ts`/`formulare-types.ts` abgeglichen: Schema `{items,total}` und flach camelCase, Share-Links/Webhooks als bare Arrays, Einreichung flach — passt; `createSubmission` hat keinen Aufrufer in einer Komponente, das Desktop sendet `submittedBy` nie; der Toggle `PATCH /share-links/{id}` (`useUpdateShareLink`, `FormularePage.tsx:988`) trifft keine Route)
- offen: IST des Seeds, kein Fund: member hat `schemas|submissions:read`, `submissions:write` und das Alt-Recht `webhooks:read` (000234); Lesen ist mandantenweit (kein own-Scope): ein zweiter member sieht, exportiert und setzt den Status fremder Einreichungen, Entwürfe sind für member sichtbar. Gehört zu `decide-member-manager-module-defaults`. Der Toggle `PATCH /share-links/{id}` des Desktops hat keine Gateway-Route (404/405), gehört zu `decide-formulare-share-link-url-channel-toggle`. Die Frage zu `submittedBy` (Fremdvergabe durch den Admin erlaubt?) steht in der neuen Unit. Fremder Mandant nicht geprüft (der E2E-Stack hat nur einen). CI-Job `e2e` muss die Datei im Morgen-PR bestätigen (`COSMI_MODULE_FORMULARE_ENABLED` steht in `ci.yml:233`).

## Iteration 37 — e2e-helpdesk-member-ticket-flow — done — 2026-10-25 23:14
- commit: (siehe git log)
- gebaut: `backend/test/e2e/helpdesk_test.go`, `TestHelpdeskMemberTicketFlow` mit acht Subtests gegen den echten Seed (manager, zwei member): manager legt ein Ticket mit dem Desktop-Body an (`requester_id` im Body wird verworfen, Requester ist der Aufrufer) und weist es member A zu (member-Zuweisen 403, ungültige ID 400); beide member legen eigene Tickets an (Validierung: Betreff leer/zu lang, unbekannte Priorität, Nicht-UUIDs 400);
  own-Scope: Liste von A zeigt genau eigenes + zugewiesenes (Zeilen und Total 2), B nur das eigene, Status-Filter behält Einschränkung und Zähler, manager sieht alle; A liest das zugewiesene Ticket und antwortet (Leernachricht 400), manager antwortet, beide sehen zwei Nachrichten; member 403 auf PUT, close, reopen, csat, merge und alle Konfigurations-Schreibzugriffe (Queue, Canned, KB, SLA, Routing, Geschäftszeiten), alle Lese-Endpunkte 200; manager PUT/close/reopen/stats. Bodies aus `helpdesk-client.ts` / `helpdesk-types.ts`.
- gate: build n.a. (nur Testdatei) | vet ok (`go vet -tags=e2e ./test/e2e/`) | lint ok (golangci-lint `--build-tags e2e`, null Funde in der neuen Datei) | test n.a. | e2e ok (lokal, `TestHelpdeskMemberTicketFlow` 1,8 s Test, 8 Subtests PASS; Erstlauf 2 min 40 s mit Build) | migration n.a. | rls-smoke n.a.
- coverage: n.a. (E2E-Unit, kein Coverage-Ziel)
- mutations-probe: in `gateway/route_helpdesk.go` (`HandleListTickets`) `grpcReq.ParticipantId = ownerID` auf `nil` gestellt -> Test rot (member A sah 3 statt 2 Zeilen, offene Tickets 2 statt 1), zurückgedreht (`git checkout`), Diff sauber.
- verify vorgaenger: sauber (`aaa66c86`: nur `formulare_test.go` plus Backlog/Journal, kein Produktcode, keine Route/Proto/Guard/Migration)
- neue-units: fix-helpdesk-own-scope-ticket-detail-and-messages (VOR `e2e-wiki-member-page-flow`, also an den Kopf der todo-Units gesetzt: VERIFIZIERTER Datenabfluss innerhalb des Mandanten. `helpdesk:ticket:read` ist für member `own`, aber nur die Liste ist eingeschränkt; `GET /tickets/{id}`, `GET|POST /tickets/{id}/messages` nehmen jede ID des Mandanten, ein zweiter member liest fremde Tickets samt internen Notizen und antwortet darauf. Fremd-Mandant nicht betroffen.)
- desktop-aufrufer: n.a. (keine Validierungs- oder Antwortform-Änderung; Antwortformen gegen `helpdesk-types.ts` abgeglichen: Liste `{tickets,total}`, Nachrichten `{messages}`, Einzelticket flach snake_case passt; das Desktop liest `attachments` der Nachrichtenliste nicht, ein leeres Array fehlt auf der Leitung, `internal: false` ebenso)
- offen: IST des Seeds, kein weiterer Fund: `GET /tickets/{id}/messages` mit unbekannter ID gibt 200 `{}` statt 404 (steht in der Fix-Unit); member sieht interne Notizen (`internal: true`) in der Nachrichtenliste, ob das gewollt ist, steht in der Fix-Unit als Frage; das Bearbeiten zugewiesener Tickets (403) deckt `feat-helpdesk-assigned-tickets-edit-for-members`, der Flow pinnt den IST-Zustand und muss dort umgestellt werden. Fremder Mandant nicht geprüft (der E2E-Stack hat nur einen). CI-Job `e2e` muss die Datei im Morgen-PR bestätigen (`helpdesk` steht in `ci.yml`).

## Iteration 38 — fix-helpdesk-own-scope-ticket-detail-and-messages — done — 2026-10-25 23:22
- commit: (siehe git log)
- gebaut: own-Scope von `helpdesk:ticket:read` gilt jetzt auch für `GET /tickets/{id}`, `GET /tickets/{id}/messages` und `POST /tickets/{id}/messages`: Gateway setzt `participant_id` (Antwort über `ownerFilterForScopeAny` auf read+reply), neue optionale Proto-Felder an `GetTicketRequest`/`ListMessagesRequest`/`AddMessageRequest` (regeneriert), Service prüft in `loadVisibleTicket` Requester/Assignee und antwortet sonst 404 (`ErrTicketNotFound`, kein 403).
  `ListMessages` lädt das Ticket zuerst (unbekannte ID 404 statt 200 `{}`). Interne Notizen: ohne `helpdesk:ticket:edit`/`helpdesk:write` blendet die Liste sie aus (`hide_internal`), ein Schreibversuch mit `internal: true` ist 403 (OpenAPI um 403 ergänzt, `swagger-cli validate` grün). E2E-Subtest auf die Sperre umgestellt, der IST-Pin „unbekannte ID = leere Liste" auf 404.
- gate: build ok | vet ok | lint ok (0 issues, helpdesk/gateway/server) | test ok (`internal/helpdesk` 177 Runs inkl. Subtests, 0 SKIP, DATABASE_URL als kmuhub_app; `internal/server` Helpdesk ok; `internal/gateway` ok inkl. `TestOpenAPIRouteDrift`/`TestOpenAPIStatusCodeDrift`) | e2e ok (lokal, `TestHelpdeskMemberTicketFlow` 2,8 s; Erstlauf 2 min 20 s Build) | migration n.a. | rls-smoke n.a.
- coverage: internal/helpdesk 82,6 % -> 83,0 %
- mutations-probe: in `helpdesk/service.go` (`loadVisibleTicket`) die Teilnehmerprüfung mit `if false &&` abgeschaltet -> `TestOwnScope_TicketDetailAndMessages/a_stranger_at_own_scope_gets_not_found_everywhere` rot (GetTicket, ListMessages, AddMessage je `<nil>` statt ErrTicketNotFound), zurückgedreht, Diff sauber.
- verify vorgaenger: sauber (`bf2fa8bc`: nur `helpdesk_test.go` plus Backlog/Journal, kein Produktcode)
- neue-units: fix-helpdesk-sla-status-own-scope (Ende BACKLOG.yml; `GET /tickets/{id}/sla-status` hat keinen own-Scope und bestätigt einem member fremde Ticket-IDs samt Fristen; nicht reproduziert, daher ans Ende)
- desktop-aufrufer: `GET /tickets/{id}`, `GET|POST /tickets/{id}/messages` -> `helpdesk-client.ts:96/149/156`, `HelpdeskPage.tsx` -> passt (member öffnet nur Tickets aus der eigenen, schon gefilterten Liste; der Interne-Notiz-Toggle erscheint nur mit `helpdesk:ticket:edit` (`canEditInternal`, Z. 1338/1444), ein member sendet `internal: true` also nie; die Nachrichtenliste zeigt `internal`-Einträge ungefiltert an, bekommt sie für member jetzt aber nicht mehr)
- offen: Entscheidung getroffen, bitte bestätigen: interne Notizen sind für alle ohne `helpdesk:ticket:edit` unsichtbar, auch für einen member, dem das Ticket ZUGEWIESEN ist (die Spec beschreibt `internal` als „not visible to requester", das Desktop gibt den Toggle nur Agenten). Soll ein zugewiesener member sie sehen, gehört das zu `feat-helpdesk-assigned-tickets-edit-for-members`. Keine neuen Keys, kein Re-Login nötig. CI-Job `e2e` muss den umgestellten Flow im Morgen-PR bestätigen.

## Iteration 39 — e2e-wiki-member-page-flow — done — 2026-10-25 23:34
- commit: (siehe git log)
- gebaut: `backend/test/e2e/wiki_test.go`, `TestWikiMemberFlow` mit neun Subtests gegen den echten Seed (manager, zwei member): Kategorien (member liest, Schreiben 403; manager legt an), Artikel mit dem Desktop-Body (base64-TipTap, `tags`/`icon` im Body, Titel leer/Kategorie-ID/kaputtes JSON 400, doppelter Slug 409, Entwurf), zweiter member liest Detail, Liste (Kategorie-, Such-, Autor-, `published`-Filter) und Suche, 400/404 auf Fremd-ID;
  Autor bearbeitet (Titel, Inhalt, Kategorie leeren), genau eine Version, Version lesen, Restore bringt den alten Inhalt zurück; Anhänge: member legt an und liest, Löschen 403 (beide Routen); Löschen und Share-Link für member 403, Listen der Tokens 200; manager legt Share-Token an, öffentliche Route ohne Login liefert nur `title/content/updated_at` (kein `id`/`tenant_id`/`author_id`), unbekanntes Token 404, Widerruf (member 403, manager 204, Token danach 404); manager räumt Anhang, Entwurf und Kategorie ab.
- gate: build n.a. (nur Testdatei) | vet ok (`go vet -tags=e2e ./test/e2e/`) | lint ok (golangci-lint `--build-tags e2e`, null Funde in der neuen Datei; vier Altfunde in `helpers_test.go`) | test n.a. | e2e ok (lokal, `TestWikiMemberFlow` 1,8 s Test, 9 Subtests PASS; Erstlauf 2 min 30 s mit Build) | migration n.a. | rls-smoke n.a.
- coverage: n.a. (E2E-Unit, kein Coverage-Ziel)
- mutations-probe: in `gateway/route_wiki.go` den Guard `wikiDelete` auf `wiki:article:read` gestellt -> Test rot (member-DELETE gab 204 statt 403, Folgefehler beim Share), zurückgedreht (`git checkout`), Diff sauber, danach wieder grün (Neubau).
- verify vorgaenger: sauber (`aa041842`: `.proto` und `.pb.go` gemeinsam, kein Guard ersetzt, Handler gehen über `helpdeskClient`, Route-Änderung hat OpenAPI-Eintrag)
- neue-units: fix-wiki-own-scope-article-edit (VOR `e2e-berichte-schedule-run-now-flow`, also an den Kopf der todo-Units gesetzt: VERIFIZIERT, `wiki:article:edit` ist für member scope `own`, aber `PATCH /articles/{id}`, Restore und Anhang-Upload prüfen nur den Key; ein zweiter member ändert fremde Artikel. Innerhalb des Mandanten, Fremd-Mandant nicht betroffen). Der IST-Zustand steht im Flow als eigener Subtest.
- desktop-aufrufer: n.a. (keine Validierungs- oder Antwortform-Änderung; gegen `wiki-client.ts`/`wiki-types.ts` abgeglichen: Listen `{articles}` (bei leerer Seite fehlt das Feld, der Client fängt das ab), Kategorien/Versionen/Anhänge/Tokens gewrappt und von `unwrapList` gelesen, passt)
- offen: IST des Seeds, kein weiterer Fund: Create/Get/Update/Restore antworten `{article:{…}}` (Version `{version}`, Anhang `{attachment}`, Token `{token}`), der Client liest sie flach; harmlos, weil `WikiPage.tsx` Artikel aus der (flachen) Liste zeigt und die Mutations-Antworten nicht nutzt, aber `useArticle` (Detail-GET nur für `view_count`) bekommt die verschachtelte Form. `tags`, `icon`, `cover_url`, `change_note` aus dem Desktop-Body sind kein Teil des Gateway-Requests und werden ohne Fehler verworfen: Tags/Icon/Cover der Oberfläche überleben kein Neuladen, das Backend hat dafür keine Felder (Feature-Lücke, keine Unit angelegt, gehört in die Wiki-Planung). manager darf löschen und teilen, weil `wiki:articles:write` (000234) beides aufmacht; member liest Entwürfe fremder Autoren (Liste und Detail), ob gewollt steht als Frage in `fix-wiki-own-scope-article-edit`. Das Templates-Gerüst des Desktops (`/wiki/templates`) hat keine Gateway-Route (`decide-wiki-templates-routes`). Fremder Mandant nicht geprüft (der E2E-Stack hat nur einen). CI-Job `e2e` muss die Datei im Morgen-PR bestätigen (`COSMI_MODULE_WIKI_ENABLED` im `ci.yml` prüfen).

## Iteration 40 — fix-wiki-own-scope-article-edit — done — 2026-10-25 23:43
- commit: (siehe git log)
- gebaut: own-Scope von `wiki:article:edit` gilt jetzt für `PATCH /wiki/articles/{id}`, `POST …/versions/{versionId}/restore` und `POST …/attachments`: neues optionales Proto-Feld `owner_id` an `UpdateArticleRequest`/`RestoreVersionRequest`/`UploadAttachmentRequest` (regeneriert), Service prüft in `loadEditableArticle` `author_id` und antwortet bei fremdem Artikel 404 (`ErrArticleNotFound`); Restore prüft den Artikel vor der Version, Upload lädt den Artikel nur bei gesetztem Owner.
  Gateway-Helper `wikiEditOwnerFilter`: wer `wiki:articles:write` (Alt-Key, nicht own) hält, bleibt mandantenweit — bewusst NICHT `ownerFilterForScopeAny`, weil der Seed manager beide Keys gibt (`wiki:article:edit` own + `wiki:articles:write` all) und die engere Regel manager sonst eingeschränkt hätte. OpenAPI: 404 am Anhang-Upload ergänzt. E2E-Subtest von IST (200) auf Sperre umgestellt, plus manager/admin-Bearbeitung fremder Artikel und Autor bearbeitet weiter.
- gate: build ok | vet ok | lint ok (0 issues, wiki/gateway/server) | test ok (`internal/wiki` 74 Runs, 0 SKIP, DATABASE_URL als kmuhub_app; `internal/server -run Wiki` ok; `internal/gateway` ok inkl. `TestOpenAPIRouteDrift`; `swagger-cli validate` ok) | e2e ok (lokal, `TestWikiMemberFlow` 3,5 s Test; Lauf 2 min 31 s mit Build) | migration n.a. | rls-smoke n.a.
- coverage: internal/wiki 78,5 % -> 79,1 %
- mutations-probe: in `wiki/service.go` (`loadEditableArticle`) die Autorprüfung mit `if false &&` abgeschaltet -> `TestService_OwnScopeEdit/a_stranger_at_own_scope_gets_not_found_on_update,_restore_and_upload` rot, zurückgedreht, Diff sauber, wieder grün.
- verify vorgaenger: sauber (`f4b91813`: nur `wiki_test.go` plus Backlog/Journal, kein Produktcode)
- neue-units: keine
- desktop-aufrufer: `PATCH /wiki/articles/{id}`, Restore, Anhang-Upload -> `wiki-client.ts:118/154/168`, `WikiArticle.tsx:42`, `WikiArticleHeader.tsx:85`, `WikiAttachments.tsx:66` -> passt (das Desktop blendet Bearbeiten/Upload schon per `useScopedCapability('wiki:article:edit', authorId)` aus, ein member erreicht die 404 also nicht über die Oberfläche)
- offen: (1) Abweichung Desktop ↔ Backend, kein Bruch: `useScopedCapability` kennt den Alt-Key nicht, manager (`wiki:article:edit` own) sieht im Desktop bei fremden Artikeln keinen Bearbeiten-Knopf, obwohl das Backend ihn per `wiki:articles:write` lässt. Soll manager fremde Artikel bearbeiten, gehört `wiki:article:edit` für manager auf scope all (Seed-Entscheidung, `decide-member-manager-module-defaults`). (2) Frage aus der Unit: member liest fremde ENTWÜRFE (Liste mit `published=false` und Detail) mandantenweit — gewollt? Nicht gesperrt, weil das Desktop Entwürfe in der Liste zeigt. (3) Nebenbefund ohne Unit: Restore prüft nicht, dass `versionId` zum Artikel gehört (Inhalt einer fremden Version lässt sich in den eigenen Artikel holen; Inhalt ist für member ohnehin lesbar). Kein Re-Login nötig, keine neuen Keys. CI-Job `e2e` muss den umgestellten Flow im Morgen-PR bestätigen.

## Iteration 41 — e2e-berichte-schedule-run-now-flow — done — 2026-10-25 23:53
- commit: (siehe git log)
- gebaut: `backend/test/e2e/berichte_test.go`, `TestBerichteMemberFlow` mit elf Subtests gegen den echten Seed (manager, zwei member): member liest Definitionen, Zeitpläne, Dokumente, Vorlagen und KPIs; legt eine Definition mit dem Desktop-Body an (Validierung Name/Modul/Format 400), liest/startet/exportiert (csv) sie als zweiter member, Löschen und Cache-Invalidierung 403; Dokument anlegen, Statuswechsel, Share-Links 403 (Liste 200); manager legt Zeitplan an (Cron/Adresse/Format/Name 400, unbekannte Definition 404), `POST …/schedules/{id}/run` 200 mit dem Zeitplan zurück (unbekannte ID 404, Nicht-UUID 400), member 403 auf run/toggle/PATCH/DELETE; manager toggelt, ändert, löscht; Aufräumen.
- gate: build n.a. (nur Testdatei) | vet ok (`go vet -tags=e2e ./test/e2e/`) | lint ok (golangci-lint `--build-tags e2e`, null Funde in der neuen Datei) | test n.a. | e2e ok (lokal, `TestBerichteMemberFlow` 1,9 s Test, 11 Subtests PASS; Erstlauf 2 min mit Build) | migration n.a. | rls-smoke n.a.
- coverage: n.a. (E2E-Unit, kein Coverage-Ziel)
- mutations-probe: in `gateway/route_berichte.go` den Guard von `POST /schedules/{id}/run` von `scheduleManage` auf `berichte:reports:read` gestellt -> Subtest „member sees the schedule but changes nothing on it" rot (member-run gab 200 statt 403), zurückgedreht (`git checkout`), Diff sauber.
- verify vorgaenger: sauber (`be5fb432`: `.proto` und `.pb.go` gemeinsam, Guards unverändert, OpenAPI-Eintrag ergänzt, kein Tabellen-/Seed-Bezug)
- neue-units: fix-berichte-definition-module-allowlist-vs-builder-sources und fix-berichte-own-scope-definition-and-document-edit (beide VOR `feat-schichten-own-assignments-for-members`, also an den Kopf der todo-Units gesetzt. (1) VERIFIZIERT, Totalausfall: der Builder speichert `module: source.module`, der Service (und zwei DB-Constraints) kennen nur sechs Module, `POST /definitions` gibt für acht von elf Quellen 400 für jeden Nutzer. (2) VERIFIZIERT, in-Mandant: `berichte:reports:edit` ist für member own, aber PATCH auf Definition und Dokument prüft den Autor nie; member setzt zudem ein Dokument auf `released`, obwohl ihm `publish` fehlt. Fremd-Mandant nicht betroffen.) Beide IST-Zustände stehen im Flow als eigene Subtests.
- desktop-aufrufer: n.a. (keine Validierungs- oder Antwortform-Änderung; gegen `berichte-client.ts`/`berichte-types.ts` abgeglichen: Listen `{definitions,total}`/`{schedules,total}`/`{documents,total}`, Einzelentitäten `{definition}`/`{schedule}`/`{document}`, Lauf `{result,run}`, `rows` nie `null` — passt; die Schedule-Tabs blendet das Desktop per `berichte:schedule:manage` aus, der 403 des member erreicht die Oberfläche nicht)
- offen: IST des Seeds, kein weiterer Fund: `POST /berichte/preview` (Builder-Live-Vorschau, `berichte-client.ts:168`) hat keine Gateway-Route und antwortet 404, das steckt schon in `fe-berichte-hide-live-preview` (BACKLOG-NEXT); im E2E-Stack sind die Downstream-Dienste des Report-Executors nicht verdrahtet, ein Lauf antwortet 200 mit `meta.warning = downstream_not_available` und leeren Zeilen, der Flow prüft darum nur Form, keine Zahlen; member sieht Zeitpläne samt Empfängeradressen (`berichte:reports:read` reicht), ob gewollt ist offen, nicht gesperrt. Fremder Mandant nicht geprüft (der E2E-Stack hat nur einen). CI-Job `e2e` muss die Datei im Morgen-PR bestätigen (`berichte` steht in `ci.yml`).

## Iteration 42 — fix-berichte-definition-module-allowlist-vs-builder-sources — done — 2026-10-26 00:03
- commit: (siehe git log)
- gebaut: Modul-Allowlist der Berichte auf die 14 Werte von `ReportModule` (`berichte-types.ts`) erweitert: `validModules` in `berichte/service.go`, beide `oneof` in `route_berichte.go`, Migration 000354 (beide CHECK-Constraints neu, `.down.sql` faltet die neuen Module auf `cross` zurück und stellt die alten sechs her), OpenAPI-Beschreibungen (10 Stellen). KPIs bleiben auf die sechs Executor-Module beschränkt (eigenes Set `kpiModules` für `sanitizeModules`, `kpiModuleVisibility` unverändert).
  DB-Test `TestService_EveryDesktopModuleIsStored` (Service + echte Constraints, je Modul Definition und Dokument, Modulwechsel per Update, `bogus` bleibt `ErrInvalidModule`) plus `TestService_GetDashboardKPIs_DropsBuilderOnlyModules`. E2E: die Schleife über die acht Builder-Module von 400 auf 201 umgestellt (Modul im Body geprüft, Modulfilter findet `fuhrpark`), Dokument mit `hr`, Builder-Lauf festgeschrieben, Aufräumen per manager.
- gate: build ok | vet ok (inkl. `-tags=e2e`) | lint ok (0 issues berichte/gateway, 0 Funde in `berichte_test.go`) | test ok (`internal/berichte` 108 Runs, 0 SKIP, DATABASE_URL als kmuhub_app; Unterpakete ok; `internal/server -run Berichte` ok; `internal/gateway` ok inkl. `TestOpenAPIRouteDrift`; `swagger-cli validate` ok) | migration ok (354 up/down/up lokal) | e2e ok (lokal, `TestBerichteMemberFlow` 1,9 s Test, 11 Subtests PASS) | rls-smoke n.a. (nur CHECK-Constraint, keine Tabelle/Policy)
- coverage: internal/berichte 86,6 % -> 86,7 %
- mutations-probe: (1) in `validModules` `fuhrpark` gestrichen und in `sanitizeModules` wieder `isValidModule` gesetzt -> `TestService_EveryDesktopModuleIsStored/fuhrpark`, `…/an_existing_definition_moves_to_a_builder_module` und `TestService_GetDashboardKPIs_DropsBuilderOnlyModules` rot, zurückgedreht, Diff sauber, grün. (2) Migration 354 lokal zurückgenommen -> `TestService_EveryDesktopModuleIsStored/work` u. a. rot mit `violates check constraint "report_definitions_module_check"` (der Test beweist also auch die DB-Seite), wieder hochgezogen.
- verify vorgaenger: sauber (`e55b835f`: nur `berichte_test.go` plus Backlog/Journal, kein Produktcode)
- neue-units: keine
- desktop-aufrufer: `POST|PATCH /berichte/definitions` (`module`) -> `ReportBuilderShell.tsx:139/146`, `ChartBlockPicker.tsx:352` -> passt (senden `source.module`, jetzt 201); `GET /berichte/definitions` -> `MyReportsLibrary.tsx:62/67` (Modulfilter aus den gelieferten Werten) -> passt; `POST /documents` (`module`) -> unverändert lesend, passt. Die Builder-Komponenten rufen `POST /definitions/{id}/run` nicht auf (nur `DashboardGrid`/`DatevView` mit Systemberichten).
- offen: Ein gespeicherter Builder-Bericht (`query_config.kind = builder`) antwortet auf `POST /definitions/{id}/run` mit 400 `execute report: berichte query config is invalid` (kein 500, im E2E festgeschrieben) — der Server-Executor kennt die Builder-Abfrage nicht; ein Zeitplan auf einen Builder-Bericht würde damit jedes Mal scheitern (keine Unit angelegt, gehört zur Builder-Planung bzw. `fe-berichte-hide-live-preview`). `templates_data.go`: `tpl-projekt` bleibt auf `cross` (lean-Marker), obwohl `work` jetzt gültig ist. Coverage-Vorher gemessen mit schon vorhandener neuer Testdatei gegen den alten Service. Migration 354 muss beim nächsten Deploy laufen (nur Constraint-Tausch, keine Datenänderung im Up). CI-Job `e2e` muss den umgestellten Flow im Morgen-PR bestätigen.

## Iteration 43 — fix-berichte-own-scope-definition-and-document-edit — done — 2026-10-26 00:13
- commit: (siehe git log)
- gebaut: own-Scope von `berichte:reports:edit` gilt jetzt für `PATCH /berichte/definitions/{id}` und `PATCH /berichte/documents/{id}`: neues optionales Proto-Feld `owner_id` an `UpdateDefinitionRequest`/`UpdateDocumentRequest` (regeneriert), Service prüft per `ownedBy` den `created_by` und antwortet bei fremder (oder erstellerloser) Zeile 404. Gateway: `wikiEditOwnerFilter` zum gemeinsamen Helper `ownerFilterUnlessAltKey` (helpers.go) verallgemeinert, `berichteEditOwnerFilter` nutzt ihn mit Alt-Key `berichte:reports:write` (manager/admin bleiben mandantenweit).
  Statuswechsel am Dokument (jedes gesetzte `status`) verlangt `berichte:reports:publish` oder `write`, sonst 403; reine Inhalts-PATCHes bleiben mit `edit` erlaubt. Guards unverändert (additiv). OpenAPI: Beschreibungen beider PATCH-Routen (403/404 waren schon dokumentiert). DB-Test `TestService_OwnScopeEdit` (echte User als Ersteller), E2E: beide IST-Subtests auf die Sperre umgestellt, plus manager/admin-Bearbeitung und Lifecycle per manager/admin.
- gate: build ok | vet ok (inkl. `-tags=e2e`) | lint ok (0 issues berichte/gateway/server) | test ok (`internal/berichte` 113 Runs, 0 SKIP, DATABASE_URL als kmuhub_app; Unterpakete ok; `internal/server -run Berichte|Wiki` ok; `internal/wiki` ok; `internal/gateway` ok inkl. `TestOpenAPIRouteDrift` auf finalem Tree; `swagger-cli validate` ok) | e2e ok (lokal, `TestBerichteMemberFlow` 3,5 s Test, 12 Subtests PASS, 2 min 39 s mit Build; dazu `TestWikiMemberFlow|TestBerichteMemberFlow` zusammen grün wegen des Helper-Umbaus) | migration n.a. | rls-smoke n.a.
- coverage: internal/berichte 86,7 % -> 86,8 %
- mutations-probe: (1) `ownedBy` in `berichte/service.go` mit `return true ||` abgeschaltet -> `TestService_OwnScopeEdit/a_stranger_at_own_scope_…` und `…/a_row_without_creator_…` rot, zurückgedreht, Diff sauber. (2) Status-Sperre in `route_berichte.go` mit `if false &&` abgeschaltet -> E2E `TestBerichteMemberFlow/member_creates_a_report_document,_edits_but_does_not_release_it,…` rot (member-PATCH `status: final` gab 200 statt 403), zurückgedreht, Diff sauber.
- verify vorgaenger: sauber (`86ea1a73`: Allowlist in Service, beiden `oneof` und Migration 354 deckungsgleich, Up/Down gefüllt, kein Guard/Proto/Route neu)
- neue-units: keine
- desktop-aufrufer: `PATCH /berichte/documents/{id}` -> `ReportDocumentEditor.tsx:86/87/103/135/160` -> passt (Bearbeiten per `useScopedCapability('berichte:reports:edit', doc.created_by)`, Lifecycle-Knöpfe nur mit `berichte:reports:publish`, Inhalts-Speichern sendet kein `status`); `PATCH /berichte/definitions/{id}` -> `ReportBuilderShell.tsx:138` (Fehler-Toast bei 404) passt; `MyReportsLibrary.tsx:96/128` (Pin = `is_published`-Toggle) -> passt formal, aber das Desktop blendet „Zum Dashboard" für fremde Custom-Berichte nicht aus: member bekommt dort jetzt 404 statt den geteilten Bericht für alle umzupinnen (vorher das Leck), es fehlt nur der FE-Gate.
- offen: (1) FE-Hinweis: Pin/Unpin in `MyReportsLibrary.tsx` sollte per `useScopedCapability('berichte:reports:edit', def.created_by)` gegatet werden (Desktop-Code, nicht Loop-Fläche, keine Unit angelegt). (2) Frage aus der Unit: verlangt schon `final` `publish`? Umgesetzt wie das Desktop: JEDER Statuswechsel braucht `publish`/`write` (member kann ein eigenes Dokument also nicht auf `final` setzen). Wenn member eigene Entwürfe abschließen soll, `final` freigeben (Lukes Entscheidung). (3) Coverage-Vorher aus Iteration 42 übernommen (dasselbe Paket, seitdem unverändert). (4) Erstellerlose Zeilen (System-/Altdaten mit `created_by NULL`) sind für own-Scope-Bearbeiter fremd. Kein Re-Login nötig, keine neuen Keys. CI-Job `e2e` muss den umgestellten Flow im Morgen-PR bestätigen.

## Iteration 44 — feat-schichten-own-assignments-for-members — done — 2026-10-26 00:26
- commit: (siehe git log)
- gebaut: Datengrenze zuerst: `GET /schichten/shifts/{id}/assignments` filtert bei Scope `own` von `schichten:assignment:read` per `ownerFilterForScope` auf die eigenen Zuweisungen (neues optionales Proto-Feld `own_user_id` an `ListAssignmentsRequest`, regeneriert; Repo-Signatur `ListAssignments(…, ownUserID)`). Own zählt die User-ID UND die HR-Profil-ID des Aufrufers (Subquery auf `hr_employee_profiles`, lean-Marker), weil das Desktop-Raster per Profil-ID zuweist, API/E2E per User-ID. Danach Migration 000355: member bekommt `schichten:assignment:read` own und `schichten:template:read` all (Down entfernt genau diese zwei Zeilen). OpenAPI-Beschreibung der Route ergänzt.
  Tests: `TestListAssignments_OwnScopeMatchesUserAndProfileID` (Service+DB), Router-Test `TestSchichtenAssignments_MemberSeesOwnOnly` mit echtem Seed-Token über `ResolveTokenPermissions` (member sieht eigene Zeile per User-ID und per Profil-ID, nicht die des Kollegen; fremde Schicht 200 `{"assignments":[]}`; manager alles; member Vorlagen 200). E2E: `assignments`/`templates` aus der 403-Liste genommen, member A sieht genau seine Zuweisung, member C eine leere Liste, member liest die Vorlage.
- gate: build ok | vet ok (inkl. `-tags=e2e`) | lint ok (0 issues schichten/gateway/server, 0 Funde in `schichten_test.go`) | test ok (`internal/schichten` 103 Runs, 0 SKIP, DATABASE_URL als kmuhub_app; `internal/gateway` ok inkl. `TestOpenAPIRouteDrift` auf finalem Tree; `internal/server -run Schichten` ok; `internal/auth` ok; `swagger-cli validate` ok) | migration ok (355 up/down/up lokal) | e2e ok (lokal, `TestSchichtenMemberFlow` 2,3 s Test, 8 Subtests PASS, 2 min 56 s mit Build) | rls-smoke n.a. (keine Tabelle/Policy; Subquery läuft unter RLS als kmuhub_app, im DB-Test belegt)
- coverage: internal/schichten 79,9 % -> 80,0 %
- mutations-probe: (1) Own-Filter in `ListAssignments` per `OR TRUE` abgeschaltet -> `TestListAssignments_OwnScopeMatchesUserAndProfileID` (3 Subtests) und `TestSchichtenAssignments_MemberSeesOwnOnly` (2 Subtests) rot, zurückgedreht, Diff sauber. (2) Gateway reicht `OwnUserId: nil` statt `ownerID` -> Router-Test (2 Subtests) rot, zurückgedreht, Diff sauber.
- verify vorgaenger: sauber (`c1871b30`: `.proto` + `.pb.go` gemeinsam, Handler über `client.UpdateDefinition/UpdateDocument`, Guards unverändert additiv, keine neue Route/Tabelle)
- neue-units: fix-schichten-swap-own-scope-employee-identity (VOR `feat-helpdesk-assigned-tickets-edit-for-members`, also an den Kopf der todo-Units gesetzt — Totalausfall, per Code-Kette belegt: der Desktop-Tauschantrag sendet die HR-Profil-ID als `requested_by_employee_id`, der own-Check vergleicht mit der User-ID -> 403 für jeden member mit Raster-Zuweisung; seit dieser Unit erreichbar)
- desktop-aufrufer: `GET /schichten/shifts/{id}/assignments` -> `useSchichten.ts:83/139` (`useShiftAssignments`, `useShiftsWithAssignments`) -> passt (`{assignments:[…]}`, leer `[]`, member-Raster zeigt nun die eigene Zeile über `employees.find(e.id === employeeId)`); `GET /schichten/templates` -> `useSchichten.ts:96`, Tab-Gate `SchichtenPage.tsx:534` (`schichten:template:read`) -> passt; Folgefund Tauschantrag `SchichtenPage.tsx:632` -> bricht (fix-schichten-swap-own-scope-employee-identity)
- offen: (1) Grants wirken erst nach Re-Login (keine neuen Keys, nur neue role_permissions-Zeilen). (2) Datenmodell-Frage für Luke: `employee_id` trägt zwei Identitäten (Profil-ID aus dem Raster, User-ID aus API/E2E/Tauschanträgen) — vereinheitlichen? Steht als notes in der neuen Fix-Unit. (3) `GET /schichten/compliance?employee_id=` und `/stats` bleiben mit `shift:read` mandantenweit (member kann ArbZG-Ruhezeiten eines Kollegen abfragen) — nicht Teil der Entscheidung, keine Unit angelegt. Migration 355 muss beim nächsten Deploy laufen. CI-Job `e2e` muss den umgestellten Flow im Morgen-PR bestätigen.

## Iteration 45 — fix-schichten-swap-own-scope-employee-identity — done — 2026-10-26 00:41
- commit: (siehe git log)
- gebaut: own-Prüfung beim Tauschantrag vom Gateway in den Service verlegt: neues optionales Proto-Feld `owner_user_id` an `CreateSwapRequestRequest` (regeneriert), Gateway reicht bei Scope `own` von `schichten:swap:create` die User-ID durch (Stringvergleich im Handler gestrichen), Service prüft per neuer Repo-Methode `IsOwnEmployeeID` (User-ID selbst ODER HR-Profil-ID des Users im Mandanten) und antwortet mit `ErrSwapRequesterNotCaller` -> PermissionDenied -> 403. Own-Filter der Liste (`ListSwapRequests`) matcht `requested_by`/`swap_with` gegen beide Identitäten (Subquery auf `hr_employee_profiles`, lean-Marker wie `ListAssignments`).
  Tests: Setup des Vorgänger-DB-Tests als `schichtenSeedHarness` herausgezogen (echter Seed über `ResolveTokenPermissions`, echte Auth-Middleware, echter gRPC-Server, Postgres-Repo), neuer `TestSchichtenSwapRequests_MemberActsUnderUserAndProfileID` (member legt per Profil-ID und per User-ID an -> 201, im Namen des Kollegen per Profil- und User-ID -> 403; own-Liste enthält eigene Anträge unter beiden Identitäten inkl. Partner-Rolle, den fremden nicht, `total` passt; manager sieht alle). Service-Test `TestService_CreateSwapRequest_OwnerMustBeRequester`, Gateway-Stubtest auf Durchreichen von `owner_user_id` umgestellt. E2E: member A im Namen von B -> 403 ergänzt.
- gate: build ok | vet ok (inkl. `-tags=e2e`) | lint ok (0 issues schichten/gateway/server) | test ok (`internal/schichten` 104 Runs, 0 SKIP, DATABASE_URL als kmuhub_app; Unterpakete ok; `internal/server -run Schichten` ok; `internal/gateway` ok inkl. `TestOpenAPIRouteDrift`, beide DB-Tests liefen real) | e2e ok (lokal, `TestSchichtenMemberFlow` 3,2 s Test, 2 min 34 s mit Build) | migration n.a. | rls-smoke n.a. (keine Tabelle/Policy; Subqueries laufen unter RLS als kmuhub_app, im DB-Test belegt)
- coverage: internal/schichten 80,0 % -> 80,0 %
- mutations-probe: (1) Profil-Zweig der Listen-Subquery per `WHERE false AND` abgeschaltet -> `TestSchichtenSwapRequests_…/member's_list_holds_their_swaps_under_both_identities…` rot, zurückgedreht, Diff sauber. (2) Owner-Prüfung im Service per `if false &&` abgeschaltet -> `…/member_cannot_file_in_a_colleague's_name…` (plus Folge-Subtests) und `TestService_CreateSwapRequest_OwnerMustBeRequester` rot, zurückgedreht, Diff sauber.
- verify vorgaenger: sauber (`a4def05c`: `.proto` + `.pb.go` gemeinsam, Handler über `client.ListAssignments`, Migration 355 Up/Down deckungsgleich mit Seed-Form, Guard unverändert; kosmetisch: lean-Marker in `ListAssignments` nennt `fix-schichten-employee-id-identity-for-own-scope`, die Unit heißt `fix-schichten-swap-own-scope-employee-identity`)
- neue-units: fe-schichten-swap-list-drop-client-own-filter (BACKLOG-NEXT, Desktop-Fläche)
- desktop-aufrufer: `POST /schichten/swap-requests` -> `useSchichten.ts` (`SWAP_BASE`), `SchichtenPage.tsx:632` (sendet Profil-ID als `requested_by_employee_id`) -> passt jetzt (201 statt 403, Antwortform unverändert `{swap_request}`); `GET /schichten/swap-requests` -> `useSchichten.ts:370`, Form `{swap_requests,total}` unverändert -> passt, ABER der Clientfilter `SchichtenPage.tsx:518-527` vergleicht nur mit `currentUserId` und blendet Profil-ID-Anträge im UI aus (fe-schichten-swap-list-drop-client-own-filter)
- offen: (1) Der Totalausfall ist backendseitig behoben, im UI aber erst vollständig mit der FE-Unit (der Antrag wird angelegt, Toast erscheint, nur die Liste blendet ihn für member aus). (2) Datenmodell-Frage aus der Unit bleibt bei Luke: `employee_id` auf EINE Identität vereinheitlichen? Dann fallen beide Subqueries weg. (3) Genehmigung (`SwapAssignmentsForRequest`) mit gemischten Identitäten (Antrag per Profil-ID, Partner per User-ID) nicht geprüft — manager-Pfad, nicht Teil der Unit. (4) Coverage-Vorher aus Iteration 44 übernommen (Paket seitdem unverändert). Kein neuer Key, kein Re-Login nötig. CI-Job `e2e` muss den ergänzten Flow im Morgen-PR bestätigen.

## Iteration 46 — feat-helpdesk-assigned-tickets-edit-for-members — done — 2026-10-26 00:53
- commit: (siehe git log)
- gebaut: Datengrenze zuerst: neues optionales Proto-Feld `editor_id` an `UpdateTicketRequest`/`CloseTicketRequest`/`ReopenTicketRequest`/`SubmitCsatRequest` (regeneriert). Gateway setzt es per `ticketEditorScope` (= `ownerFilterUnlessAltKey` mit Alt-Key `helpdesk:write`) bei Scope `own` von `helpdesk:ticket:edit`; Service prüft in `loadEditableTicket` den ASSIGNEE (nicht den Requester) und antwortet sonst 404 (`ErrTicketNotFound`); ein `assignee_id` im PUT bei own ist `ErrReassignNotAllowed` -> 403. `POST /assign` und `POST /merge` verlangen `isTenantWideTicketAgent` (edit/write mit Scope ≠ own), sonst 403. `canWriteInternalNotes` läuft jetzt über denselben Helper, damit der neue own-Grant interne Notizen NICHT freischaltet (member ist auch Requester).
  Danach Migration 000356: member bekommt `helpdesk:ticket:edit` own (Down entfernt genau diese Zeile). OpenAPI: Beschreibungen an PUT/close/reopen/csat/assign/merge, 403 an PUT/assign/merge. Tests: `TestOwnScope_EditAssignedTicket` (Service), Router-DB-Test `TestHelpdeskAssignedTicketEdit_MemberAgainstRealSeed` (echter Seed über `ResolveTokenPermissions`, echter gRPC-Server + Postgres: zugewiesen 200 für PUT/close/csat/reopen, nur erstellt/fremd/unbekannt 404, assign/merge/PUT-assignee 403, Kollege bearbeitet seins 200, interne Notiz member 403 + unsichtbar). E2E-Subtest vom IST-Pin auf das neue Verhalten umgestellt.
- gate: build ok | vet ok (inkl. `-tags=e2e`) | lint ok (0 issues helpdesk/gateway/server) | test ok (`internal/helpdesk` 181 Runs, 0 SKIP, DATABASE_URL als kmuhub_app; Unterpakete ok; `internal/server -run Helpdesk` ok; `internal/gateway` ok inkl. `TestOpenAPIRouteDrift` und `TestOpenAPIStatusCodeDrift` auf finalem Tree, 0 SKIP, Router-DB-Test lief real; `swagger-cli validate` ok) | migration ok (356 up/down/up lokal) | e2e ok (lokal, `TestHelpdeskMemberTicketFlow` 2,9 s Test, 2 min 33 s mit Build) | rls-smoke n.a. (keine Tabelle/Policy)
- coverage: internal/helpdesk 83,0 % -> 83,5 %
- mutations-probe: (1) Assignee-Prüfung in `loadEditableTicket` per `if false &&` abgeschaltet -> `TestOwnScope_EditAssignedTicket` (2 Subtests) und `TestHelpdeskAssignedTicketEdit_…/a_ticket_the_member_only_raised_or_does_not_know_answers_404` rot, zurückgedreht, Diff sauber. (2) In `isTenantWideTicketAgent` `&&` durch `||` ersetzt (Scope ignoriert) -> `…/handing_the_ticket_on_stays_with_tenant-wide_agents` und `…/internal_notes_stay_with_agents_beyond_own_scope` rot, zurückgedreht, Diff sauber.
- verify vorgaenger: sauber (`0c92e75c`: `.proto` + `.pb.go` gemeinsam, Handler über `client.CreateSwapRequest`, own-Prüfung im Service, kein Guard ersetzt, keine Route/Tabelle neu)
- neue-units: fe-helpdesk-own-scope-hide-agent-actions (BACKLOG-NEXT, Desktop-Fläche, mit Entscheidungsfrage interne Notizen)
- desktop-aufrufer: `PUT /helpdesk/tickets/{id}` -> `helpdesk-client.ts:103`, `HelpdeskPage.tsx:538/667/1402` (Status, Zusatzfelder, Eskalieren; senden nie `assignee_id`) -> passt (member am zugewiesenen Ticket 200, Antwort unverändert flaches Ticket); `POST /close|/reopen` -> `helpdesk-client.ts:107/111`, `HelpdeskPage.tsx:525/533` -> passt; `POST /csat` -> `useSaveCsat` (CSATWidget) -> passt; `POST /assign|/merge` -> `HelpdeskPage.tsx:1389/1408`, hinter `canEditTicket = useScopedCapability('helpdesk:ticket:edit', assigneeId)` -> bricht für member (403, fe-helpdesk-own-scope-hide-agent-actions); Interne-Notiz-Toggle `canEditInternal = useHasCapability('helpdesk:ticket:edit')` erscheint jetzt für member, Senden 403 (gleiche FE-Unit)
- offen: (1) Grant wirkt erst nach Re-Login (kein neuer Key, nur eine role_permissions-Zeile). Migration 356 muss beim nächsten Deploy laufen. (2) Entscheidung umgesetzt, bitte bestätigen: interne Notizen bleiben Agenten mit Scope ≠ own vorbehalten, auch für den zugewiesenen member (Stand Iteration 38); Frage steht in der FE-Unit. (3) Ein member am zugewiesenen Ticket darf `queue_id`, `contact_id`, `org_id` und Zusatzfelder ändern, nur `assignee_id` ist gesperrt — wenn Queue-Wechsel als „Umzuweisen" gelten soll, ergänzen. (4) Ein vom member selbst erstelltes, aber nicht zugewiesenes Ticket antwortet beim Bearbeiten 404 (nicht 403), obwohl er es lesen kann — gleiches Muster wie Wiki/Berichte own-Edit. (5) member kann am zugewiesenen Ticket CSAT erfassen (Route hängt am Edit-Guard). CI-Job `e2e` muss den umgestellten Flow im Morgen-PR bestätigen.

## Iteration 47 — feat-work-own-task-delete-for-members — done — 2026-10-26 01:08
- commit: (siehe git log)
- gebaut: Datengrenze zuerst: neues optionales Proto-Feld `owner_id` an `DeleteTaskRequest` (regeneriert). Gateway setzt es per `ownerFilterUnlessAltKey` (Alt-Key `tasks:delete`) bei Scope `own` von `work:task:delete`; `task.Service.Delete` bekommt `ownerID *uuid.UUID` und antwortet bei fremdem `created_by` mit `ErrNotFound` (404), vor der Projekt-Editor-Prüfung. Guard `taskDelete` unverändert (nimmt `work:task:delete` schon an). Danach Migration 000357: member bekommt `work:task:delete` own (Down entfernt genau diese Zeile). Projekte löschen bleibt bei `projects:delete`/`work:project:delete` (member hält keins).
  OpenAPI: Beschreibung plus 401/403/404 an `DELETE /tasks/{id}` (403 Viewer und 404 kamen schon vorher, 401 neu über den Owner-Filter). Tests: Service `TestService_DeleteOwnScopeOnlyCreator`; Router-DB-Test `TestWorkOwnTaskDelete_MemberAgainstRealSeed` (echter Seed über `ResolveTokenPermissions`, echter Work-gRPC-Server + Postgres: eigene Aufgabe standalone und im eigenen Projekt 200 und danach weg, Aufgabe von Kollege/Manager 404 und bleibt, eigenes Projekt löschen 403, manager löscht fremde Aufgabe 200). E2E `TestMemberDailyWorkFlow`: IST-Pin 403 auf eigene Aufgabe 200 umgestellt, plus Admin-Aufgabe -> member 404.
- gate: build ok | vet ok (inkl. `-tags=e2e`) | lint ok (0 issues work/task, gateway, server) | test ok (`internal/work/task` 83 Runs, 0 SKIP, DATABASE_URL als kmuhub_app; `internal/work/...` ok; `internal/server -run Work` ok; `internal/gateway` ok inkl. `TestOpenAPIRouteDrift` und `TestOpenAPIStatusCodeDrift` auf finalem Tree, 0 SKIP; `swagger-cli validate` ok) | migration ok (357 up/down/up lokal) | e2e ok (lokal, `TestMemberDailyWorkFlow` 3,4 s Test, 2 min 23 s mit Build) | rls-smoke n.a. (keine Tabelle/Policy)
- coverage: internal/work/task 85,1 % -> 85,1 %
- mutations-probe: (1) Owner-Prüfung in `task.Service.Delete` per `if false &&` abgeschaltet -> `TestService_DeleteOwnScopeOnlyCreator` und `TestWorkOwnTaskDelete_…/member_cannot_delete_a_colleague's_or_a_manager's_task` rot, zurückgedreht, Diff sauber. (2) Gateway reicht `OwnerId: nil` statt `ownerID` -> derselbe Router-Subtest rot (member löschte die Kollegen-Aufgabe), zurückgedreht, Diff sauber. Ein erster Anlauf von Probe 2 erzeugte nur einen Syntaxfehler und galt nicht; wiederholt.
- verify vorgaenger: sauber (`26fe03b9`: `.proto` + `.pb.go` gemeinsam, Handler über `client.UpdateTicket/CloseTicket/ReopenTicket/SubmitCsat`, Prüfung im Service, Guards unverändert additiv, Migration 356 Up/Down deckungsgleich mit Seed-Form)
- neue-units: fe-work-task-delete-own-means-creator (BACKLOG-NEXT, Desktop-Fläche)
- desktop-aufrufer: `DELETE /tasks/{id}` -> `hooks/useTasks.ts:222` (`useDeleteTask`, Antwort `{status}` unverändert) -> passt; Gate `modules/work/useWorkPermissions.ts:32` `useScopedCapability('work:task:delete', assignee_id, created_by, reporter_id)` -> bricht teilweise: member sieht Löschen jetzt auch an nur zugewiesenen Aufgaben und bekommt 404 (fe-work-task-delete-own-means-creator)
- offen: (1) Grant wirkt erst nach Re-Login (kein neuer Key, nur eine role_permissions-Zeile). Migration 357 muss beim nächsten Deploy laufen. (2) `done_when` nennt 204, die Route liefert seit jeher 200 `{"status":"task deleted"}` — Antwortform nicht geändert (FE liest sie so). (3) Erstellerlose Aufgaben gibt es nicht (`created_by` NOT NULL im Modell); Aufgaben aus Vorlagen/Automationen gehören ihrem anlegenden User. (4) Subtasks: das Löschen der eigenen Elternaufgabe räumt per DB-Kaskade auch fremde Unteraufgaben ab, falls ein Kollege dort welche angelegt hat — nicht Teil der Entscheidung, keine Unit angelegt. (5) Coverage-Vorher per `git stash` auf demselben Paket gemessen. CI-Job `e2e` muss den umgestellten Flow im Morgen-PR bestätigen.

## Iteration 48 — feat-crm-deals-write-team-scope-for-members — done — 2026-10-26 01:23
- commit: (siehe git log)
- gebaut: Datengrenze zuerst — es gab KEINE Team-Grenze für Deals (und auch nicht für Kontakte: `crm:contact:edit|team` wirkt seit 000256 wie `all`, `PermissionScope` kennt `team` nur in HR). Gebaut mit der einzigen Team-Definition im Backend (Reporting-Line wie HR 334/335, timetracking `ReportingLineOf`): neue Proto-Nachricht `DealEditor{user_id, scope}` als Feld an Create/Update/MoveToStage/AddTags/RemoveTags (regeneriert). Gateway `dealEditor()` setzt sie, wenn der Aufrufer `deals:write` nicht jenseits von own hält und `crm:deal:<create|edit>` nicht `all` ist; Service `checkEditor` lässt nur Deals durch, deren Owner (`owner_id`, sonst `created_by`) der Aufrufer oder (bei team) ein direkter Bericht ist (`IsDirectReport`, `hr_employee_profiles.manager_user_id`), sonst `ErrOutsideEditorScope` -> 403. Update prüft vorher UND nach einem Owner-Wechsel (kein Herausgeben aus dem Team), Create prüft das Ziel-Owner. Danach Migration 000358: member bekommt `crm:deal:create` + `crm:deal:edit` mit Scope team (Down entfernt genau diese Zeilen). Löschen bleibt manager/admin. Deal-Tag-Routen bleiben bei `deals:write` (kein Desktop-Aufrufer, `TestCapabilityGuards_AdditiveWiring` pinnt das), die Service-Grenze sitzt dort trotzdem.
  OpenAPI: Beschreibungen an POST/PUT/stage, 403 an stage und beide Tag-Routen (der Guard lieferte 403 schon vorher). Tests: Service `TestService_EditorScope` (5 Subtests), Router-DB-Test `TestCRMDealTeamWrite_MemberAgainstRealSeed` (echter Seed über `ResolveTokenPermissions`, echter CRM-gRPC-Server + Postgres, HR-Profil mit `manager_user_id`: eigene und Bericht-Deals PUT/stage 200, Kollegen-Deal PUT/stage/tags 403 und unverändert, Bericht -> Vorgesetzten-Deal 403, Owner an Kollegen 403, Anlegen für Kollegen 403 / Manager 201, eigener Deal löschen 403, manager schreibt Kollegen-Deal 200). E2E `TestContactAndDealDesktopBodyFlow`: IST-Pin 403 auf member legt Deal an 201, pflegt und verschiebt ihn 200, löscht 403, Admin-Deal weiter 403.
- gate: build ok | vet ok (inkl. `-tags=e2e`) | lint ok (0 issues crm/gateway/server) | test ok (`internal/crm/deal` 80 Runs, 0 SKIP, DATABASE_URL als kmuhub_app; `internal/crm/...` ok; `internal/server -run CRM|Deal` ok; `internal/gateway` 5627 Runs, 0 SKIP, inkl. `TestOpenAPIRouteDrift`/`TestOpenAPIStatusCodeDrift` auf finalem Tree; `swagger-cli validate` ok) | migration ok (358 up/down/up lokal) | e2e ok (lokal, `TestContactAndDealDesktopBodyFlow` 2,4 s Test, 2 min 33 s mit Build) | rls-smoke n.a. (keine Tabelle/Policy; `IsDirectReport` filtert zusätzlich explizit auf tenant_id und läuft unter RLS als kmuhub_app)
- coverage: internal/crm/deal 88,2 % -> 88,2 %
- mutations-probe: (1) `checkEditor` antwortet `nil` statt `ErrOutsideEditorScope` -> `TestService_EditorScope` (4 Subtests) und `TestCRMDealTeamWrite_…` (4 Subtests: Kollegen-Anlage, Kollegen-Deal, nach oben, Herausgeben) rot, zurückgedreht, Diff sauber. (2) Gateway `dealEditor` liefert immer nil (`|| true`) -> dieselben 4 Router-Subtests rot, zurückgedreht, Diff sauber. Beim ersten Lauf von Probe 1 riefen die Router-Helper `t.Fatalf` des Elterntests und brachen nach dem ersten Subtest ab; Helper nehmen jetzt das Subtest-`t`, Probe wiederholt.
- verify vorgaenger: sauber (`0aa773e7`: `.proto` + `.pb.go` gemeinsam, Handler über `client.DeleteTask` mit `OwnerId`, Prüfung im Service, Guard unverändert, Migration 357 Up/Down deckungsgleich mit Seed-Form)
- neue-units: fe-crm-deal-edit-team-scope-and-team-definition (BACKLOG-NEXT, Desktop-Fläche plus Entscheidungsfrage Team-Definition)
- desktop-aufrufer: `POST /deals` -> `hooks/useDeals.ts:98`, Gate `DealsListPage.tsx:102` `useHasCapability('crm:deal:create')` -> passt (member 201, Antwort `{deal}` unverändert); `PUT /deals/{id}` -> `useDeals.ts:132`, `POST /deals/{id}/stage` -> `useDeals.ts:166`, Gates `DealDetailPage.tsx:90` und `DealPipelineView.tsx:276` `useScopedCapability('crm:deal:edit')` ohne Owner -> bricht teilweise: member sieht Bearbeiten/Ziehen an jedem Deal, fremde antworten 403 (fe-crm-deal-edit-team-scope-and-team-definition); `/deals/{id}/tags` -> kein Desktop-Aufrufer
- offen: (1) ENTSCHEIDUNG bitte bestätigen: "Team" = Owner ist der Aufrufer oder ein DIREKTER Bericht (Reporting-Line wie HR). Ein member ohne Berichte pflegt damit praktisch nur eigene Deals, nicht die der Kollegen mit demselben Vorgesetzten. Soll "Pipeline seines Teams" Geschwister/`department` heißen, ändert sich nur `IsDirectReport` (Frage steht in der NEXT-Unit). (2) manager hält weiter `deals:write|all` und schreibt damit tenant-weit, obwohl sein `crm:deal:edit` `team` ist — unverändert, wie vor dieser Unit. (3) Grants wirken erst nach Re-Login (keine neuen Keys, nur zwei role_permissions-Zeilen). Migration 358 muss beim nächsten Deploy laufen. (4) Kontakte haben dieselbe Lücke (`crm:contact:edit|team` = all) — nicht Teil der Entscheidung, keine Unit angelegt; gehört in die Team-Definitionsfrage. (5) `crm_grpc.pb.go` wurde beim Regen nur in den Zeilenenden berührt und ist nicht gestaged. CI-Job `e2e` muss den umgestellten Flow im Morgen-PR bestätigen.

## Iteration 49 — feat-wiki-own-page-share-delete-for-members — done — 2026-10-26 01:44
- commit: (siehe git log)
- gebaut: Datengrenze zuerst: neues optionales Proto-Feld `owner_id` an `DeleteArticleRequest`/`DeleteAttachmentRequest`/`CreateShareTokenRequest`/`RevokeShareTokenRequest` (regeneriert). Gateway setzt es per `wikiDeleteOwnerFilter` (`wiki:article:delete`) bzw. `wikiShareOwnerFilter` (`wiki:share_token:create`), beide `ownerFilterUnlessAltKey` mit Alt-Key `wiki:articles:write` (manager bleibt mandantenweit). Service prüft über `loadEditableArticle` den `author_id`: Löschen und Teilen direkt am Artikel, Anhang löschen und Widerruf lösen vorher den Artikel über zwei neue tenant-gescopte Repo-Lookups auf (`GetAttachmentArticleID`, `GetShareTokenArticleID`); fremd -> 404 (`ErrArticleNotFound`/`ErrAttachmentNotFound`/`ErrShareTokenNotFound`).
  Danach Migration 000359: member bekommt `wiki:article:delete` + `wiki:share_token:create` mit Scope own (Down entfernt genau diese Zeilen). OpenAPI: Beschreibungen plus 403 an DELETE Artikel, DELETE Artikel-Anhang, POST share, DELETE share. Tests: Service `TestService_OwnScopeDeleteAndShare` (4 Subtests), Repo-Lookups im RLS-DB-Test `TestWikiWrites_LandInCallerTenant` (eigener/fremder Mandant), Router-DB-Test `TestWikiOwnDeleteShare_MemberAgainstRealSeed` (echter Seed über `ResolveTokenPermissions`, echter Wiki-gRPC-Server + Postgres: eigene Seite teilen 201/widerrufen 204, fremde teilen/widerrufen 404 und Link bleibt, fremden Anhang 404 und bleibt, eigenen 204, fremde Seiten löschen 404 und bleiben, eigene 204, manager teilt/widerruft/löscht fremde 201/204/204). E2E `TestWikiMemberFlow`: IST-Pins 403 auf das neue Verhalten umgestellt (zweiter member 404 an fremder Seite/Anhang/Link; Autor teilt, öffentlicher Abruf 200, widerruft, Abruf 404; Autor löscht eigene Seite 204).
- gate: build ok | vet ok (inkl. `-tags=e2e`) | lint ok (0 issues wiki/gateway/server) | test ok (`internal/wiki` 79 Runs, 0 SKIP, DATABASE_URL als kmuhub_app; `internal/server -run Wiki` ok; `internal/gateway` 5634 Runs, 0 SKIP, inkl. `TestOpenAPIRouteDrift`/`TestOpenAPIStatusCodeDrift` auf finalem Tree; `swagger-cli validate` ok) | migration ok (359 up/down/up lokal) | e2e ok (lokal, `TestWikiMemberFlow` 3,6 s Test, 10 PASS, 2 min 34 s mit Build) | rls-smoke n.a. (keine Tabelle/Policy; die neuen Lookups sind im RLS-DB-Test als kmuhub_app mit fremdem Mandanten belegt: 0 Treffer -> NotFound)
- coverage: internal/wiki 79,1 % -> 80,6 %
- mutations-probe: (1) Autorprüfung in `loadEditableArticle` per `if false &&` abgeschaltet -> `TestService_OwnScopeDeleteAndShare/a_stranger…` und 4 Subtests von `TestWikiOwnDeleteShare_…` (fremd teilen, fremd widerrufen, fremder Anhang, fremde Seite löschen) rot, zurückgedreht, Diff sauber. (2) `wikiShareOwnerFilter` liefert immer nil -> `…/member_cannot_share_a_colleague's…` und `…/member_cannot_revoke_a_link…` rot, zurückgedreht (`cmp` gegen Backup), wieder grün.
- verify vorgaenger: sauber (`2748042e`: `.proto` + `.pb.go` gemeinsam, Handler über `client.CreateDeal/UpdateDeal/MoveDealToStage/AddDealTags/RemoveDealTags`, Prüfung im Service, Guards unverändert, Migration 358 Up/Down deckungsgleich mit Seed-Form)
- neue-units: fe-wiki-share-dialog-own-scope (BACKLOG-NEXT, Desktop-Fläche)
- desktop-aufrufer: `DELETE /wiki/articles/{id}` -> `wiki-client.ts:124`, `useWiki.ts:158`, Gate `WikiArticleHeader.tsx:87` `useScopedCapability('wiki:article:delete', authorId)` -> passt (member sieht Löschen nur an eigenen, 204); `DELETE /wiki/attachments/{id}` -> `wiki-client.ts:184`, Gate `WikiAttachments.tsx:66` (`wiki:article:edit` own) -> passt (jetzt 204 statt 403); `POST /wiki/articles/{id}/share` und `DELETE /wiki/share/{id}` -> `wiki-client.ts:223/231`, Gate `WikiShareDialog.tsx:51` `useHasCapability('wiki:share_token:create')` ohne Owner -> bricht teilweise: member sieht Teilen/Widerruf an fremden Seiten und bekommt 404 (fe-wiki-share-dialog-own-scope). Antwortformen unverändert.
- offen: (1) Grants wirken erst nach Re-Login (keine neuen Keys, nur zwei role_permissions-Zeilen). Migration 359 muss beim nächsten Deploy laufen. (2) Annahme: "eigene Seite löschen" schließt Anhänge der eigenen Seite ein — die Anhang-Routen hängen am selben Guard (`wikiDelete`), eine Grenze war ohnehin nötig. (3) Widerruf eines Links gehört zum Teilen (Guard `wikiShare`, own über den Artikel-Autor, nicht über `created_by` des Links): manager-Links an der eigenen Seite kann der Autor also widerrufen. (4) manager hält `wiki:share_token:create` nicht, nur den Alt-Key — das Desktop blendet Teilen für manager aus (steht in der FE-Unit). (5) `done_when` sagt "200/204": Teilen antwortet 201, Löschen/Widerruf 204 (unverändert). CI-Job `e2e` muss den umgestellten Flow im Morgen-PR bestätigen.

## Iteration 50 — feat-fuhrpark-vehicle-availability-route — done — 2026-10-26 01:59
- commit: (siehe git log)
- gebaut: Neue Route `GET /api/v1/fuhrpark/vehicles/{id}/availability?from&to` (beide Pflicht, RFC3339) über neue RPC `GetVehicleAvailability` (Proto erweitert, `.pb.go`/`_grpc.pb.go` regeneriert, kein Service-Shortcut im Gateway). Antwort `{vehicle_id, from, to, items: [{starts_at, ends_at}], total}` per `ProtoFull` (leere Liste `[]`). Service prüft `to > from`, Fenster ≤ 93 Tage (`MaxAvailabilityWindow`, sonst 400) und dass das Fahrzeug im Mandanten existiert (sonst 404); Repo `ListBusySlots` liest NUR `starts_at, ends_at` aktiver Buchungen (booked, in_use), halboffen wie die Konfliktprüfung (`bookingOverlapPredicate`), tenant-gescoped. Guard `RequirePermission("fuhrpark:booking","read")` — member hält ihn (own, 000339), der Scope wird bewusst nicht angewendet, weil die Antwort nichts Personenbezogenes trägt. Keine Migration, kein neuer Key.
  OpenAPI-Pfad mit 200/400/401/403/404/503. Tests: Service `TestService_GetVehicleAvailabilityValidatesWindowAndVehicle` (5 Fälle), Repo-DB `TestListBusySlots_ActiveOnlyHalfOpenTenantScoped` (storniert/abgeschlossen raus, Randberührung raus, anderes Fahrzeug raus, fremder Mandant 0), Router-DB `TestFuhrparkVehicleAvailability_MemberAgainstRealSeed` (echter Seed über `ResolveTokenPermissions`, echter Fuhrpark-gRPC-Server + Postgres: member sieht das Fenster eines Kollegen 09–11 ohne `user_id`/`purpose`/Kollegen-ID im Body, die Buchungsliste bleibt own, freies Fenster `items:[]`/`total:0`, fehlendes/umgedrehtes/zu langes/kaputtes Fenster 400, unbekanntes Fahrzeug 404, readonly 403).
- gate: build ok | vet ok | lint ok (0 issues fuhrpark/gateway/server) | test ok (`internal/fuhrpark` 198 Runs, 0 SKIP, DATABASE_URL als kmuhub_app; `internal/server -run Fuhrpark` ok; `internal/gateway` 5640 Runs, 0 SKIP, inkl. `TestOpenAPIRouteDrift`/`TestOpenAPIStatusCodeDrift`; finaler Tree nach Stash/Pop erneut grün; `swagger-cli validate` ok) | migration n.a. | e2e n.a. (kein bestehender Flow geändert oder betroffen) | rls-smoke n.a. (keine Tabelle/Policy; `ListBusySlots` filtert explizit auf tenant_id und ist im Repo-DB-Test als kmuhub_app mit fremdem Mandanten belegt: 0 Treffer)
- coverage: internal/fuhrpark 83,3 % -> 83,4 %
- mutations-probe: (1) Statusfilter `status IN ('booked','in_use')` aus `ListBusySlots` entfernt -> `TestListBusySlots_…` und `TestFuhrparkVehicleAvailability_…/member_sees_a_colleague's_booked_window…` rot, Datei aus Backup zurück, Diff sauber. (2) Fensterlimit im Service per `if false &&` abgeschaltet -> `TestService_GetVehicleAvailability…/window_longer_than_limit` und `TestFuhrparkVehicleAvailability_…/bad_input_and_unknown_vehicles` (200 statt 400) rot, zurückgedreht, Diff sauber.
- verify vorgaenger: sauber (`fd7b3ed0`: `.proto` + `.pb.go` gemeinsam, Handler über `client.DeleteArticle/DeleteAttachment/CreateShareToken/RevokeShareToken` mit `OwnerId`, Prüfung im Service, neue Repo-Lookups tenant-gescoped, Guards unverändert, Migration 359 Up/Down deckungsgleich mit Seed-Form)
- neue-units: fe-fuhrpark-pool-booking-with-availability (BACKLOG-NEXT, Desktop-Fläche)
- desktop-aufrufer: `GET /fuhrpark/vehicles/{id}/availability` -> kein Desktop-Aufrufer (neue Route); auch `/fuhrpark/bookings` hat keinen Aufrufer im Modul (`modules/fuhrpark` kennt keine Buchungen, nur `api/types.ts`) -> n.a. für bestehende Aufrufer, Lücke als fe-fuhrpark-pool-booking-with-availability
- offen: (1) Die Zeiten kommen wie bei `/fuhrpark/bookings` in der Server-Zeitzone formatiert (RFC3339 mit Offset, z. B. `+01:00`), `from`/`to` im Echo so, wie sie gesendet wurden — gültiges RFC3339, aber nicht einheitlich UTC. (2) Annahme: auch manager (booking:read own) und admin nutzen die Route; readonly hält keinen Booking-Key und bekommt 403. (3) Fahrzeugstatus (`in_service`, `inactive`) fließt nicht in die Verfügbarkeit ein — nur Buchungen. Soll ein Fahrzeug in der Werkstatt als belegt erscheinen, wäre das eine eigene Entscheidung. (4) Desktop kann weder buchen noch die Route nutzen (NEXT-Unit).

## Iteration 51 — seed-fuhrpark-damage-read-all-for-members — done — 2026-10-26 02:09
- commit: (siehe git log)
- gebaut: Datengrenze zuerst: `HandleReportDamage` band den Melder bisher über den LESE-Scope (`ownerFilterForScope(damage, read)`) an den Aufrufer — mit read all hätte ein member Schäden unter einem Kollegen melden können (der Kollege steht dann als Melder in der Liste). Neu hängt die Bindung an `fuhrpark:damage:write` jenseits von own (`damageEditorTenantWide`, nur admin): alle anderen melden unter sich selbst, admin weiter unter dem genannten Melder. Verhalten für member/manager/admin damit unverändert. Danach Migration 000360: `fuhrpark:damage:read` für member per UPDATE von own auf all (Down setzt own zurück). PATCH/resolve bleiben bei `damage:write` (admin), manager bleibt own.
  Tests: Router-DB-Test `TestFuhrparkDamageRead_MemberAgainstRealSeed` (echter Seed über `ResolveTokenPermissions`, echter Fuhrpark-gRPC-Server + Postgres: member sieht Kollegen-Schaden in beiden Listen, manager nicht, member PATCH/resolve 403 und Status bleibt, member meldet mit Kollegen-ID im Body unter sich selbst, admin unter dem genannten Kollegen). Stub-Test `TestFuhrparkRoutes_MemberSelfServiceIsNarrowedToTheCaller`: Melder bei Scope all ohne write jetzt gepinnt, neuer Subtest write all/own. E2E `TestFuhrparkMemberFlow`: IST-Pin "member B sieht den Schaden nicht" auf "sieht ihn" umgestellt, Kopfkommentar nachgezogen.
- gate: build ok | vet ok (inkl. `-tags=e2e`) | lint ok (0 issues gateway) | test ok (`internal/gateway` 5647 Runs, 0 SKIP, DATABASE_URL als kmuhub_app, inkl. `TestOpenAPIRouteDrift`/`TestOpenAPIStatusCodeDrift`) | migration ok (360 up/down/up lokal, Seed-Abfrage nach jedem Schritt: member all/own/all, manager own, admin all) | e2e ok (lokal, `TestFuhrparkMemberFlow` 3,5 s Test, 2 min 41 s mit Build) | rls-smoke n.a. (keine Tabelle/Policy)
- coverage: n.a. (kein Coverage-Ziel; `internal/fuhrpark` nicht angefasst)
- mutations-probe: (1) Melder-Bindung per `if false &&` abgeschaltet -> `TestFuhrparkDamageRead_…/member_files_a_damage_under_themselves…` und drei Subtests von `TestFuhrparkRoutes_MemberSelfServiceIsNarrowedToTheCaller` rot, Datei aus Backup zurück, Diff sauber. (2) Migration 360 lokal down -> `TestFuhrparkDamageRead_…/member_reads_a_colleague's_damage_in_both_lists` rot, wieder up, grün.
- verify vorgaenger: sauber (`88d43a8a`: `.proto` + `.pb.go`/`_grpc.pb.go` gemeinsam, Handler über `client.GetVehicleAvailability`, Guard `fuhrpark:booking:read` mit Seed aus 000300/000339, Route in `openapi.yaml`, Antwort ohne Nutzer/Zweck)
- neue-units: decide-fuhrpark-damage-read-scope-for-managers (BACKLOG-NEXT, Frage an Luke)
- desktop-aufrufer: `GET /fuhrpark/damages` und `/vehicles/{id}/damages` -> `fuhrpark-client.ts:196/216`, `useFuhrpark.ts:163` (Antwort `{damages}` unverändert, member sieht nur mehr Zeilen) -> passt; `POST /vehicles/{id}/damages` -> `FuhrparkPage.tsx:2811` sendet `reported_by` als Freitext-Namen (landet über `damageReporter` in den Notizen, Melder = Aufrufer) -> passt, für member/manager/admin unverändert
- offen: (1) Grant wirkt erst nach Re-Login (Scope-Änderung einer bestehenden role_permissions-Zeile). Migration 360 muss beim nächsten Deploy laufen. (2) manager sieht jetzt weniger Schäden als member (own vs. all) — Entscheidungsfrage in `decide-fuhrpark-damage-read-scope-for-managers`. (3) Die Schadensliste trägt `reported_by` (User-ID) und Notizen, also auch "Gemeldet von: <Name>" — für member nun mandantenweit sichtbar; laut Entscheidung gewollt (stand schon in `/vehicles/{id}/history`). (4) Eigene Rollen eines Mandanten mit `damage:read|all` ohne `damage:write` melden jetzt unter sich selbst statt unter dem genannten Melder — das war vorher möglich. CI-Job `e2e` muss den umgestellten Flow im Morgen-PR bestätigen.

## Iteration 52 — feat-inventar-transfer-for-movement-create — done — 2026-10-26 02:19
- commit: (siehe git log)
- gebaut: Guard an `POST /api/v1/inventar/transfer` additiv von `RequirePermission(inventar:item, write)` auf `RequirePermissionAny(inventar:item:write, inventar:movement:create)` erweitert (Alt-Key bleibt). Datengrenze: member hält `movement:create` bereits mit Scope all (000224/000256, lokal geprüft), Artikel sind für member mandantenweit lesbar, die Umbuchung bucht im Service beide Beine in einer Transaktion (`ApplyStockMovementsTx`, tenant-gescoped), der Akteur war schon per `callerIDPtr` gepinnt. Keine Migration, kein neuer Key.
  OpenAPI: Beschreibung an der Route (beide Keys, Akteur = Aufrufer, `performed_by` ignoriert). Tests: Router-DB `TestInventarTransfer_MemberAgainstRealSeed` (echter Seed über `ResolveTokenPermissions`, echter Inventar-gRPC-Server + Postgres: member bucht 2 Stück um -> 204, Bestände 5/1 -> 3/3, zwei `transfer`-Bewegungen mit `performed_by` = Aufrufer, obwohl der Body einen Kollegen nennt; Überziehen 4xx und beide Bestände unverändert; readonly 403 und Bestand unverändert). `TestCapabilityGuards_AdditiveWiring`: Zeile "still requires item:write only" ersetzt durch "is not an adjustment" (movement:adjust bleibt 403) plus neue Zeile movement:create erlaubt. E2E `TestInventarEinkaufFlow`: IST-Pin "member Umbuchung 403" auf "member bucht um 204, Bestände 1/2" umgestellt (vorher bewies der Admin denselben Schritt), Subtest-Name nachgezogen.
- gate: build ok | vet ok (inkl. `-tags=e2e`) | lint ok (0 issues gateway) | test ok (`internal/gateway` 5652 Runs, 0 SKIP, DATABASE_URL als kmuhub_app, inkl. `TestOpenAPIRouteDrift`/`TestOpenAPIStatusCodeDrift`; `swagger-cli validate` ok) | migration n.a. | e2e ok (lokal, `TestInventarEinkaufFlow` 2,7 s Test, 2 min 28 s mit Build) | rls-smoke n.a. (keine Tabelle/Policy)
- coverage: n.a. (kein Coverage-Ziel; `internal/inventar` nicht angefasst)
- mutations-probe: (1) Guard zurück auf `RequirePermission(inventar:item, write)` -> `TestInventarTransfer_…/member_transfers_stock_and_is_the_actor_of_both_legs` rot, Datei aus Backup zurück. (2) Handler reicht `req.PerformedBy` statt `callerIDPtr(r)` durch -> derselbe Subtest rot (Akteur = Kollege), zurück, `cmp` gegen Backup sauber.
- verify vorgaenger: sauber (`0981823b`: keine Proto-/Routen-Änderung, Melder-Bindung über `damageEditorTenantWide` im Handler, Migration 360 Up/Down deckungsgleich als Scope-Update, keine Tabelle)
- neue-units: fe-inventar-transfer-dialog (BACKLOG-NEXT, Desktop-Fläche)
- desktop-aufrufer: `POST /inventar/transfer` -> `inventar-client.ts:99` `transferStock`, `hooks/useInventar.ts:192` `useTransferStock`; Body `TransferStockInput` (`inventar-types.ts:123`) passt, Antwort `void`/204 unverändert -> passt. Keine Komponente ruft den Hook auf (Umbuchung im UI nicht erreichbar) -> fe-inventar-transfer-dialog
- offen: (1) `done_when` sagt "Umbuchung 200": die Route antwortet seit jeher 204 (Spec und Desktop `request<void>`), unverändert gelassen. (2) Keine neuen Keys, kein Re-Login nötig — member trägt `inventar:movement:create` schon im Token. (3) manager hält `item:write` ohnehin, für ihn ändert sich nichts. CI-Job `e2e` muss den umgestellten Flow im Morgen-PR bestätigen.

## Iteration 53 — seed-finance-module-view-for-managers — done — 2026-10-26 02:27
- commit: (siehe git log)
- gebaut: Migration 000361: manager bekommt `finance:module:view` (scope all, eine Zeile, `ON CONFLICT DO NOTHING`; Down löscht genau diese Zeile). Keine Route, kein Proto, kein neuer Key (existiert seit 000256, admin/readonly halten ihn). Test `TestResolveTokenPermissions_DB_ManagerHoldsFinanceModuleView` (echter Seed über `ResolveTokenPermissions`: manager-Token trägt den Key, ohne Scope-Einengung; member trägt ihn nicht).
  Nebenbei gefixt: `TestPresetRoles_DB_MemberAndManagerHoldBranchenSelfServiceKeys` stand seit Iteration 51 (Migration 360, member `damage:read` own -> all) rot, weil Iteration 51 nur Gateway-Tests gefahren hat; Erwartung für member/`fuhrpark:damage:read` auf `all` nachgezogen, manager bleibt `own`.
- gate: build n.a. (nur Tests/SQL) | vet ok | lint ok (0 issues auth) | test ok (`internal/auth` komplett grün, 0 SKIP, DATABASE_URL als kmuhub_app; `internal/gateway` ok) | migration ok (361 up/down/up lokal, Seed-Abfrage nach jedem Schritt: admin+readonly / +manager / +manager) | e2e n.a. (kein Flow prüft manager-Finanzkachel oder berichte-kpis als manager) | rls-smoke n.a. (keine Tabelle/Policy)
- coverage: internal/auth 77,9 % -> 77,9 % (ein Seed-Test, kein neuer Code)
- mutations-probe: Migration 361 lokal down -> `TestResolveTokenPermissions_DB_ManagerHoldsFinanceModuleView` rot (Token ohne `finance:module:view`), wieder up, grün.
- verify vorgaenger: Befund klein — `39c56bbb` selbst sauber (additiver Guard `RequirePermissionAny(item:write, movement:create)`, Akteur gepinnt, Spec-Text, keine Proto-/Tabellenänderung). Aber `0981823b`/Iteration 51 ließ `preset_branchen_self_service_db_test.go` rot (siehe oben); in diesem Commit behoben, keine eigene Fix-Unit nötig.
- neue-units: keine
- desktop-aufrufer: n.a. (keine Validierungs- oder Antwortform-Änderung); Desktop gated Modulkacheln generisch über `<module>:module:view` (`rbac-types.ts:12`, `App.tsx:206`), manager sieht die Finanzkachel nach Re-Login.
- offen: (1) Grant wirkt erst nach Re-Login (JWT). Migration 361 muss beim nächsten Deploy laufen. (2) Nebenwirkung: `GET /berichte/kpis` schneidet Module nach `<module>:module:view` — manager sieht damit auch die Finanz-/Umsatzkacheln im Berichte-Dashboard (`kpiModuleVisibility["finanzen"]`); manager liest Rechnungen ohnehin (`finance:read` all), also kein neuer Datenzugang, aber sichtbar anders. (3) manager hält weiter nur die groben Keys `finance:read`/`finance:write`, keine feinen (`finance:amounts:view` o. Ä.) — nicht Teil der Entscheidung. (4) Lehre: Seed-Änderungen an bestehenden Keys brauchen auch `go test ./internal/auth/` (Preset-Pin-Tests), nicht nur Gateway.

## Iteration 54 — feat-hr-team-documents-for-managers — blocked — 2026-10-26 02:33
- commit: (siehe git log)
- gebaut: nichts Bleibendes (Code zurückgesetzt). Gebaut und getestet war: Guard an `GET /hr/employees/{id}/documents` additiv `RequirePermissionAny(hr:read, team:documents:view)`, Reichweite `employeeDocumentsReach` im Gateway (hr:read/all -> alles, team -> Proto-Flag `restrict_to_reporting_line = 4` an `ListEmployeeDocumentsReq` (regeneriert), own -> nur eigene ID sonst 403), Prüfung `ListReportingLineEmployeeDocuments` im employee-Service nach Muster `GetReportingLineLeaveBalance` mit neuem `ErrNotInReportingLine` -> PermissionDenied, Spec 400/401/403. Der Router-DB-Test (echter Seed, echte Auth, echter HR-gRPC-Server hinter `TenantInboundUnaryInterceptor`, Postgres als kmuhub_app) war in den Grenzfällen grün (manager Fremder/unbekannte ID 403, member fremd 403), aber manager bekam für den Bericht `[]` statt der manager-/employee-Dokumente — und **hr_admin ebenso `[]`** für eine Akte mit Dokumenten in allen drei Stufen.
  Ursache: `hr_document_access` prüft die Rolle über `app.user_roles`, und die Rollen überqueren den gRPC-Hop nicht (`grpc_tenant.go` trägt nur Tenant, User, Perms, Client-IP/UA). Im biz-Service sieht damit jeder nur seine eigenen employee-Dokumente — Totalausfall der Personalakte für HR/admin, Upload laut Code-Lesung per `WITH CHECK` ebenso.
- gate: build ok (vor dem Zurücksetzen) | vet n.a. | lint n.a. | test rot (`TestHREmployeeDocuments_FollowsTeamDocumentsReach`: 2 von 4 Subtests rot, Ursache oben, nicht in dieser Unit behebbar ohne Entscheidung) | migration n.a. | e2e n.a. | rls-smoke n.a.
- coverage: n.a. (kein Coverage-Ziel, nichts committet)
- mutations-probe: n.a. (nichts committet)
- verify vorgaenger: sauber (`3d4b4c8f`: nur Seed-Migration 361 Up/Down deckungsgleich, DB-Test, Fix des Preset-Pin-Tests; keine Route, kein Proto, keine Tabelle)
- neue-units: fix-hr-document-rls-roles-over-grpc (Totalausfall, VOR die erste todo-Unit gesetzt); feat-hr-team-documents-for-managers selbst nach BACKLOG-NEXT verschoben (blocked, deps auf die Fix-Unit)
- desktop-aufrufer: n.a. (keine Validierungs- oder Antwortform-Änderung committet)
- offen: (1) Frage an Luke: Nach dem Rollen-Fix sieht ein manager laut Policy die manager- UND employee-Stufe; `gehaltsabrechnung` (000310) ist employee-Stufe, manager hält `team:salary:view` aber nicht. Sieht der manager die Gehaltsabrechnungen seiner Berichte — ja, oder Kategorie für Aufrufer ohne `team:salary:view` ausfiltern? (2) Die Fix-Unit muss `/hr/personnel-documents` für manager/member im selben Commit auf die eigenen Dokumente einengen, sonst öffnet der Rollen-Fix allen managern die Akten des ganzen Mandanten. (3) Upload-500 nur per Code-Lesung, nicht reproduziert — die Fix-Unit belegt es mit einem Test. (4) Produktion vermutlich betroffen, seit HR über gRPC läuft; ob dort je ein HR-Dokument hochgeladen wurde, morgens per `SELECT count(*) FROM hr_employee_documents` prüfbar (lesend).

## Iteration 55 — fix-hr-document-rls-roles-over-grpc — done — 2026-10-26 02:39
- commit: (siehe git log)
- gebaut: Neuer Metadata-Header `x-user-roles` in `middleware/grpc_tenant.go`: Outbound schickt die Rollen der Whitelist `forwardedRoles` = {admin, hr_admin}, Inbound filtert erneut und legt sie unter `UserRolesKey` ab. Damit setzt der biz-Pool `app.user_roles`, und `hr_document_access` (000128) greift wieder: admin/hr_admin lesen alle drei Stufen, Uploads bestehen `WITH CHECK`. `manager` fehlt mit Absicht (`lean:`-Kommentar): mit der Rolle öffnet die Policy jedem manager die manager- und employee-Stufe des ganzen Mandanten (inkl. `gehaltsabrechnung`) über `/hr/personnel-documents`. So bleibt für manager/member alles wie heute, ohne Handler-Filter. Die Einengung, die die Unit für den Fall „manager-Rolle kommt an“ verlangt hat, war damit nicht mehr nötig. `forwardedOf` nutzt jetzt den gemeinsamen `allowedOf`.
  Tests: Router-DB `TestHRDocuments_RolesReachTheDocumentPolicy` (echter Seed, echte Auth, echter HR-gRPC-Server hinter `TenantInboundUnaryInterceptor`, Postgres als kmuhub_app; drei Kategorien je Stufe): admin liest `/{id}/documents` mit allen drei Stufen; admin und hr_admin lesen `/personnel-documents` komplett; manager und member sehen dort nur ihre eigenen employee-Dokumente; admin lädt über `/{id}/documents` hoch (201), hr_admin über `/personnel-documents` (201); manager/member Upload 403. Unit-Tests `TestRoleForwarding_*`: admin/hr_admin kommen an, manager und nicht-ASCII-Rollen nicht, ein handgebauter Header schmuggelt keine Rolle hinein, member/manager senden keinen Header.
- gate: build ok (middleware, gateway, cmd/gateway, cmd/biz) | vet ok | lint ok (0 issues) | test ok (`internal/middleware`; `internal/biz/hr/...` 7 Pakete; `internal/gateway` 5658 Runs, 0 SKIP, DATABASE_URL als kmuhub_app, inkl. `TestOpenAPIRouteDrift`) | migration n.a. | e2e n.a. (kein Flow prüft HR-Dokumente) | rls-smoke n.a. (keine Tabelle/Policy angefasst, Policy-Wirkung über den Router-DB-Test als kmuhub_app belegt)
- coverage: internal/middleware 72,3 % -> 72,4 %
- mutations-probe: (1) `forwardedRoles` leer -> drei Subtests des Router-DB-Tests rot (admin `[]`, Upload scheitert) und `TestRoleForwarding_AdminAndHRAdminCrossTheHop` rot. (2) `manager` in `forwardedRoles` ergänzt -> „manager and member see only their own …“ und die Upload-Gegenprobe rot (manager sieht die Akte des Kollegen). Beide Male aus Backup zurückgespielt, `cmp` sauber.
- verify vorgaenger: sauber (`ec662dfc`: nur BACKLOG/NEXT/JOURNAL, Code wie notiert zurückgesetzt)
- neue-units: keine (`feat-hr-team-documents-for-managers` in BACKLOG-NEXT um den Hinweis ergänzt: `manager` erst zusammen mit Berichtslinien-Grenze und Einengung von `/personnel-documents` in `forwardedRoles` aufnehmen)
- desktop-aufrufer: `/hr/employees/{id}/documents` (GET/POST) -> `hr-client.ts:924/930`; Antwortform unverändert (GET `{documents}`, POST `{document}`), admin bekommt jetzt Zeilen statt `[]` -> passt. `/hr/personnel-documents` hat keinen Desktop-Aufrufer außerhalb von `types.ts` -> n.a.
- offen: (1) Wirkt ohne Re-Login: Die Rollen stehen schon im JWT, nur der Hop war neu. Nach dem Deploy sehen admin/hr_admin in Produktion die Akten, und Uploads gehen. (2) Die beiden Upload-Routen antworten unterschiedlich (`/{id}/documents` -> `{document}`, `/personnel-documents` -> nackt). Bestand, nicht angefasst. (3) Die Rollen gehen an jeden gRPC-Service. Außer `hr_document_access` liest kein Service `app.user_roles` oder `IsAdmin` (grep). (4) Prod-Check weiterhin sinnvoll: `SELECT count(*) FROM hr_employee_documents` (lesend). Er zeigt, ob dort je etwas hochgeladen werden konnte.

## Iteration 56 — feat-hr-team-profile-full-fields-for-managers — done — 2026-10-26 02:48
- commit: (siehe git log)
- gebaut: `GET /hr/employees/{id}` liefert ohne `hr:read` das volle Profil, wenn `team:data_personal:view` UND `team:data_job:view` den Mitarbeiter erreichen (`fullProfileReaches` in `route_hr.go`): Scope all -> jeder, Scope team -> direkte Berichte (`manager_user_id` = Aufrufer, dieselbe Berichtslinie wie `GetReportingLineLeaveBalance`). Sonst bleibt die Verzeichnisprojektion. Kein Proto, keine Migration, kein neuer Key: manager hält beide Keys seit 000256 auf team, member auf own (-> Verzeichnis), hr_admin auf all. Feldliste: `EmployeeProfile` trägt weder Gehalt noch Bank (`hourly_rate` liegt nur in der Tabelle, nicht im Proto), also gilt Vollprofil = Personal- und Vertragsdaten. `lean:`: beide Keys oder Verzeichnis, Aufteilung je Key erst, wenn eine Custom-Rolle nur einen hält. OpenAPI: Beschreibung der Operation.
  Test `TestHREmployee_ManagerSeesFullProfileOfReportsOnly` (echter Seed über `ResolveTokenPermissions`, echte Auth, echter HR-gRPC-Server hinter `TenantInboundUnaryInterceptor`, Postgres als kmuhub_app): manager sieht Vollfelder des eigenen Berichts, nur Verzeichnisfelder des Berichts eines anderen managers; member nur Verzeichnis; hr_admin Vollprofil eines beliebigen Mitarbeiters.
- gate: build ok | vet ok | lint ok (0 issues gateway) | test ok (`internal/gateway` 5663 Runs pass, 0 SKIP, 0 FAIL, DATABASE_URL als kmuhub_app, inkl. `TestOpenAPIRouteDrift`; `swagger-cli validate` ok) | migration n.a. | e2e n.a. (kein Flow prüft das Profilfenster als manager) | rls-smoke n.a. (keine Tabelle/Policy)
- coverage: internal/gateway 78,4 % -> 78,4 % (eine kleine Funktion, durch den neuen Test gedeckt)
- mutations-probe: (1) Team-Zweig auf `caller != "" && emp == nil` (nie wahr) -> Subtest "manager sees the full profile of a direct report" rot. (2) Team-Zweig ohne `manager_user_id`-Vergleich -> Subtest "manager sees only the directory fields of someone else's report" rot. Beide Male aus Backup zurück, `cmp` sauber.
- verify vorgaenger: sauber (`0ec52e25`: Rollen-Whitelist admin/hr_admin über den Hop, Inbound filtert erneut; außer `hr_document_access` liest kein Service die Rollen aus dem Kontext — `settings` prüft `IsAdmin` per DB; keine Proto-/Routen-/Tabellenänderung)
- neue-units: keine
- desktop-aufrufer: `GET /hr/employees/{id}` -> `hr-client.ts:909` (`adaptEmployee`) -> `MemberProfileContent.tsx:138ff` rendert Vertrag, Startdatum, Adresse schon bedingt (wie bisher für admin) -> passt; Antwortform `{employee}` unverändert.
- offen: (1) Neu auch für hr_admin: hält kein `hr:read`, sah im Profilfenster bisher nur das Verzeichnis, jetzt über Scope all das Vollprofil — gewollte Folge der Keys, aber eine sichtbare Änderung. (2) Kein Re-Login nötig, die Keys stehen schon im Token. (3) Beobachtung ohne Produktionswirkung: Profilzeilen mit `NULL` in den Text-Spalten (z. B. `emergency_contact_phone`) lassen `GetByUserID` mit 500 scheitern; lokal haben alle 14 Zeilen (Test-Altlasten vom 25.10.) solche NULLs. Der einzige Schreibpfad (`PostgresEmployeeRepo.Create`) schreibt Leerstrings, Produktion daher vermutlich nicht betroffen — per `SELECT count(*) FROM hr_employee_profiles WHERE emergency_contact_phone IS NULL` lesend prüfbar.

## Iteration 57 — feat-absence-calendar-projection-for-members — done — 2026-10-26 02:57
- commit: (siehe git log)
- gebaut: `GET /hr/absences/calendar` additiv `RequirePermissionAny(hr:read, team:absence:read)`. Die Art folgt `leaveBalanceReach` (hr:read bzw. `team:absence_data:view`): Reichweite all (admin, hr_admin) -> volle `AbsenceEntry` wie bisher; darunter Projektion `absenceCalendarEntry` für jede Zeile (Name, Abteilung, Zeitraum, `absence_class: "absent"`, neutrale Farbe `#9ca3af`, kein `total_days`), `leave_type_name` plus echte Farbe nur für eigene Abwesenheiten und bei Scope team (manager) für direkte Berichte. Dafür neues Proto-Feld `AbsenceEntry.manager_user_id = 9` (regeneriert), Repo liest `ep.manager_user_id`, gRPC-Server reicht es durch (gofmt hat dabei zwei Strukturliterale in `hr_grpc.go` neu ausgerichtet). OpenAPI: Beschreibung, `oneOf` mit neuem Schema `AbsenceCalendarEntry`, 400/403 ergänzt.
  Test `TestHRAbsenceCalendar_ProjectsTheLeaveTypeByReach` (echter Seed über `ResolveTokenPermissions`, echte Auth, echter HR-gRPC-Server hinter `TenantInboundUnaryInterceptor`, Postgres als kmuhub_app, Art „Krankheit“): member sieht die Art nur bei sich selbst, manager nur beim eigenen Bericht, admin und hr_admin voll.
- gate: build ok | vet ok | lint ok (0 issues gateway/server/absence) | test ok (`internal/gateway` 5667 pass, 0 SKIP, 0 FAIL, DATABASE_URL als kmuhub_app, inkl. `TestOpenAPIRouteDrift`; `internal/biz/hr/absence`, `internal/server` ok; `swagger-cli validate` ok) | migration n.a. | e2e n.a. (kein Flow ruft den Kalender) | rls-smoke n.a. (keine Tabelle/Policy)
- coverage: internal/gateway 78,4 % -> 78,4 %
- mutations-probe: (1) Team-Zweig abgeschaltet -> manager-Subtest rot. (2) Eigene-Zeile-Zweig abgeschaltet -> member-Subtest rot. (3) Team ohne `manager_user_id`-Vergleich -> manager-Subtest rot (sieht Art des Fremden). (4) Voller Kalender für alle -> member- und manager-Subtest rot. Jeweils aus Backup zurück, `cmp` sauber.
- verify vorgaenger: sauber (`352fc52c`: `fullProfileReaches` verlangt beide Keys, Scope own -> Verzeichnis, Team nur über `manager_user_id`; keine Proto-/Tabellen-/Routenänderung, Spec-Text nachgezogen)
- neue-units: keine
- desktop-aufrufer: `GET /hr/absences/calendar` -> `hr-client.ts:857` (`adaptAbsenceEntry`: fehlendes `leave_type_name` -> `''`, Farbe aus `leave_type_color`) -> `team/AbsenceCalendar.tsx:219/232` zeigt bei leerem Namen `team.absence.absent`, `dashboard/widgets/Absences.tsx:70` mappt `leaveTypeKey` (kam schon bisher nie vom Backend) -> passt. Hülle `{entries}` unverändert. Die Legende (`AbsenceCalendar.tsx:255`) gruppiert nach `leaveTypeKey`, der immer `''` ist — Bestand, nicht neu.
- offen: (1) Auslegung: `done_when` sagt „member sieht keinen leave_type_name“ — umgesetzt für fremde Zeilen; die eigene Abwesenheit zeigt dem member seine Art (eigene Daten, `team:absence_data:view` own). Soll auch das weg, ist es eine Zeile. (2) Wirkt ohne Re-Login: `team:absence:read` steht schon im Token; member/readonly/manager kommen jetzt durch den Guard, vorher 403. (3) Der Tenant-Schalter `ShowAbsenceReason` maskiert weiterhin zusätzlich für alle. (4) Projektion ist tenantweit (wer abwesend ist) — so entschieden, der `department`-Filter bleibt frei wählbar.

## Iteration 58 — feat-dashboard-team-worktime-route — done — 2026-10-26 03:06
- commit: (siehe git log)
- gebaut: `GET /api/v1/dashboard/team-worktime` in `route_dashboard.go` (`HandleGetTeamWorktime`), gRPC-Hop `GetTeamTime` über die biz-Verbindung; `DashboardRoutes.WithHR(registry)` reicht sie herein (`cmd/gateway/main.go`), ohne Registry 503, Layout-Routen unberührt. Reichweite wie `/hr/time/team`: `narrowestScopeAny(hr:read, zeiterfassung:team:view)` -> Service listet bei all den Mandanten, sonst die Berichtslinie. Wer keinen der beiden Keys hält (member), bekommt `{members: []}` mit 200 ohne Aufruf — Pflicht, weil `PermissionScope` für einen nicht gehaltenen Key `all` liefert. Antwort `{members: [{id, name, minutes}]}`, Minuten = `week_minutes` (net) der laufenden Woche. OpenAPI-Eintrag mit 200/401/503.
  Test `TestDashboardTeamWorktime_FollowsTheTeamTimeReach` (echter Seed über `ResolveTokenPermissions`, echte Auth, echter HR-gRPC-Server mit timetracking-Service hinter `TenantInboundUnaryInterceptor`, Postgres als kmuhub_app, ein Zeiteintrag 420 min): manager sieht nur den eigenen Bericht mit 420, member leere Liste 200, admin und hr_admin den ganzen Mandanten.
- gate: build ok (gateway, cmd/gateway) | vet ok | lint ok (0 issues) | test ok (`internal/gateway` 5671 pass, 0 SKIP, 0 FAIL, DATABASE_URL als kmuhub_app, inkl. `TestOpenAPIRouteDrift`; `swagger-cli validate` ok) | migration n.a. | e2e n.a. (kein Flow ruft das Widget) | rls-smoke n.a. (keine Tabelle/Policy)
- coverage: internal/gateway 78,4 % -> 78,4 %
- mutations-probe: (1) Scope fest auf `"all"` -> manager-Subtest rot (sieht den Fremden). (2) Kurzschluss für Aufrufer ohne Key abgeschaltet -> member-Subtest rot (member sieht über das `all` des nicht gehaltenen Keys den ganzen Mandanten). Beide Male aus Backup zurück, `cmp` sauber.
- verify vorgaenger: sauber (`ed69d499`: additiver Guard `RequirePermissionAny(hr:read, team:absence:read)`, Proto-Feld 9 mit regeneriertem `.pb.go`, Projektion verbirgt Art und Farbe außerhalb der Reichweite, Spec nachgezogen; keine Tabelle, kein neuer Key)
- neue-units: keine
- desktop-aufrufer: `GET /dashboard/team-worktime` -> `api/hooks/useTeamWorktime.ts:21` liest `json.members ?? []` mit `{id, name, minutes}` -> Form passt; `TeamWorktime.tsx` zeigt bei leerer Liste den Empty-State. ABER der Hook ruft rohes `fetch` ohne Auth-Header -> heute 401, das Widget zeigt erst nach `fe-raw-fetch-without-auth-header` (BACKLOG-NEXT) Daten.
- offen: (1) Widget bleibt bis zum FE-Fix leer (401 -> Hook wirft -> Empty-State). (2) Keine `week_start`-Option — das Widget fragt nur die laufende Woche (YAGNI). (3) Testprofile mussten die Textspalten als `""` setzen wie `PostgresEmployeeRepo.Create`; mit NULL in `position_title` scheitert die Mitarbeiterliste mit 500 (dieselbe Beobachtung wie Iteration 56, Produktion vermutlich nicht betroffen).

## Iteration 59 — review-run17-r3 — done — 2026-10-26 03:14
- commit: siehe Commit `docs(loop): review run 17 rights commits r3` (dieser Eintrag; kein Code-Commit, kein Fund)
- gebaut: Review der Commits seit r2-End-SHA `a1c8f05e`, End-SHA **`edec275a`** (r4 beginnt dort). 30 Commits, Ergebnis je SHA:
  - Seed-Migrationen 354–361 (`86ea1a73`, `a4def05c`, `26fe03b9`, `0aa773e7`, `2748042e`, `fd7b3ed0`, `0981823b`, `3d4b4c8f`): in EINER Transaktion mit ROLLBACK lokal als `kmuhub` alle Downs rückwärts, dann jedes Up zweimal. Downs entfernen genau die zehn eigenen Zeilen (member: crm:deal:create/edit team, helpdesk:ticket:edit own, schichten:assignment:read own, schichten:template:read all, wiki:article:delete own, wiki:share_token:create own, work:task:delete own; manager: finance:module:view all) und setzen fuhrpark:damage:read member all -> own zurück; 354-Down stellt die enge CHECK-Liste her. Up×2 fehlerfrei, Endstand = Ausgangsstand (0 Zeilen Diff, 0 Constraint-Diff). Kein Key davon hielt member/manager schon vor der Migration (grep 000001–000353: nur Key-Definition 000256/000095, Grants an andere Rollen) -> Down löscht keinen Altbestand -> ok.
  - Code-Commits mit Grant/Guard/Scope/Sichtbarkeit (`edec275a`, `ed69d499`, `352fc52c`, `0ec52e25`, `39c56bbb`, `0981823b`, `88d43a8a`, `fd7b3ed0`, `2748042e`, `0aa773e7`, `26fe03b9`, `0c92e75c`, `a4def05c`): jeder bringt einen Router-DB-Test mit Token aus dem echten Seed (`ResolveTokenPermissions` bzw. `schichtenSeedHarness.token` = Preset-Rolle aus `roles`, keine Handbau-Permissions), jeweils mit Negativfall zweiter member/fremder Bericht/manager-Linie. 13 Tests plus `TestOpenAPIRouteDrift` als `kmuhub_app`: 74 PASS, 0 SKIP -> ok.
  - Guard-Diffs (Klasse 8): `ed69d499` `hr:read` -> `RequirePermissionAny(hr:read, team:absence:read)`, `39c56bbb` `inventar:item:write` -> `Any(inventar:item:write, inventar:movement:create)` -> beide additiv. `88d43a8a` neuer Guard `fuhrpark:booking:read` auf neuer Route: Seed admin all, manager own, member own vorhanden -> ok.
  - `c1871b30`, `be5fb432`, `aa041842` (own scope berichte/wiki/helpdesk) und die E2E-Commits `e55b835f`, `f4b91813`, `bf2fa8bc`, `39c56bbb`, `0981823b`, `0aa773e7` (Flow-Anpassung): lokaler E2E-Stack mit Berichte-, Fuhrpark-, Fuhrpark-Fahrtenbuch-, Helpdesk-, Inventar/Einkauf-, Member-Tagesarbeit- und Wiki-Flow: 7/7 Top-Level PASS, 66 PASS gesamt, 0 SKIP -> ok.
  - Reine Test-Commits ohne Rechteänderung (`aaa66c86`, `95309497`, `9c0a881a`, `d0000132`, `f20b2a02`, `6129ca59`, `8eea0a06`, `b472f6b4`) und Loop-Doku (`ec662dfc`, `3bbcd950`): n.a.
- gate: `internal/auth` 314 PASS, 0 SKIP; `internal/gateway` (13 Router-DB-Tests + Drift) 74 PASS, 0 SKIP; DATABASE_URL als `kmuhub_app`; E2E lokal 66 PASS, 0 SKIP | migration ok (Up×2/Down in Rollback-Tx) | rls-smoke n.a. (keine Tabelle/Policy im Prüfbereich)
- coverage: n.a. (kein Coverage-Ziel)
- mutations-probe: n.a. (Review-Unit; die Mutations-Proben der geprüften Commits stehen in deren Journal-Einträgen)
- verify vorgaenger: sauber (`edec275a`: Handler geht über `hrv1.NewHRServiceClient(conn).GetTeamTime`, kein Stub, kein Proto, kein neuer Guard, Spec-Eintrag vorhanden, member-Kurzschluss vor dem Aufruf)
- neue-units: keine
- desktop-aufrufer: n.a. (keine Validierungs- oder Antwortform-Änderung)
- offen: (1) 000360-Down setzt member auf `own` zurück, was dem Stand aus 000339 entspricht — korrekt, aber nur solange 339 nicht selbst zurückgerollt ist. (2) Alle Seed-Änderungen wirken erst nach Re-Login (Permissions im JWT).

## Iteration 60 — feat-scrub-email-sender-names-on-contact-anonymize — done — 2026-10-26 03:21
- commit: (siehe git log)
- gebaut: `ScrubDependentPII` (`crm/consent/scrub.go`) setzt bei Anonymisierung und Hard-Delete `email_messages.from_name = NULL` und `from_email = AnonymizedRequesterEmail` (NOT NULL-Spalte) für alle über `email_contact_links` verknüpften Mails des Mandanten. Eigene Ausgangsmails des Postfachbesitzers (`from_email` = `email_accounts.email_address`) bleiben unberührt, weil der Link auch für Empfänger-Kontakte existiert und dort der Mitarbeiter der Absender ist. Godoc: neuer Scrub-Absatz plus Eintrag unter „Deliberately NOT scrubbed“ (Betreff, Body, HTML, Header, Empfängerlisten; §257 HGB).
  Test `TestAnonymizeContact_ScrubsEmailSenderIdentity`: verknüpfte Eingangsmail gescrubbt (Betreff/Body bleiben), eigene Ausgangsmail, unverknüpfte Mail, Mail eines Geschwister-Kontakts und Mail eines fremden Mandanten bleiben.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (`internal/crm/...` alle ok; `consent` 35 PASS, 0 SKIP, DATABASE_URL als kmuhub_app) | migration n.a. | rls-smoke n.a. (keine Tabelle/Policy; Test läuft als kmuhub_app mit Fremd-Mandant-Zeile) | gateway n.a. (keine Route)
- coverage: internal/crm/consent 65,3 % -> 65,4 %
- mutations-probe: (1) Own-Mailbox-Ausschluss auf `false` gesetzt -> Test rot („owner's own outbound mail must keep its sender“). (2) Link-Subselect ohne `contact_id`-Filter -> Test rot (Geschwister-Mail gescrubbt). Beide Male aus Backup zurück, `cmp` sauber.
- verify vorgaenger: sauber (`5f8fee21` reiner Doku-Commit der Review-Unit r3, kein Code)
- neue-units: keine
- desktop-aufrufer: n.a. (keine Validierungs- oder Antwortform-Änderung)
- offen: (1) `search_vector` von `email_messages` wird per Trigger beim UPDATE neu berechnet, enthält aber weiterhin Betreff und Body (bewusst nicht gescrubbt). (2) Der Platzhalter `geloescht@deleted.invalid` ist derselbe wie bei Tickets. (3) Nur Mails mit vorhandenem `email_contact_links`-Eintrag werden erfasst; nicht verknüpfte Mails desselben Absenders bleiben (Verknüpfung entsteht beim Sync automatisch). (4) Beim Verifizieren hinterließen die Mutations-Probe-Läufe lokal Testdaten (Cleanup-Reihenfolge der ersten Testfassung), manuell entfernt; die endgültige Fassung räumt vollständig auf.

## Iteration 61 — feat-dsar-meeting-notes-and-chat — done — 2026-10-26 03:24
- commit: (siehe git log)
- gebaut: Zwei neue DSAR-Module in `gdpr/dsar_search_video_dialer.go`: `meetingNotesModule` („Meeting-Notizen“: Meeting, Notiz, Privat, Angelegt; `author_id` = Betroffener, eigene private Notizen inklusive, es sind seine Daten) und `meetingChatModule` („Meeting-Chat“: `meeting_chat.sender_id` = Betroffener). Beide über `userRowsModule`, tenant-gescoped, auf `dsarMaxRows` gedeckelt, in `videoDialerModules` eingehängt. Dateikommentar nachgezogen (kein „undecided“ mehr).
  Test im bestehenden `TestSearchByQuery_UserVideoAndDialerParticipation_Integration`: eigene geteilte + eigene private Notiz sichtbar, geteilte und private Notiz des Kollegen nicht, nur die eigene Chatzeile; Fremd-Mandant sieht nur seine eigene Notiz/Chatzeile.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (`internal/security/...` alle ok; `gdpr` 234 PASS, 0 SKIP, DATABASE_URL als kmuhub_app) | migration n.a. | rls-smoke n.a. (keine Tabelle/Policy) | gateway n.a. (keine Route)
- coverage: internal/security/gdpr 76,0 % -> 76,1 %
- mutations-probe: (1) Autorenfilter der Notizen entfernt -> Test rot (Notizen des Kollegen, privat wie geteilt, tauchen auf). (2) Senderfilter des Chats entfernt -> Test rot (2 statt 1 Zeile). Beide aus Backup zurück, `cmp` sauber. (Ein erster Versuch mit ersatzlos gestrichenem `$2` war ungültig — SQLSTATE 42P18 —, deshalb mit `$2::uuid IS NOT NULL` wiederholt.) Der Tenant-Filter selbst ist durch RLS doppelt gesichert; seine Mutation bleibt ohne tenant-fremde Daten im sichtbaren Kontext unsichtbar, der Fremd-Mandant-Fall prüft die Isolation von der anderen Seite.
- verify vorgaenger: sauber (`7129522a`: `ScrubDependentPII` setzt `from_name`/`from_email` nur über `email_contact_links`, kein Stub, keine Route/Proto/Tabelle, Test mit Geschwister- und Fremd-Mandant-Zeile)
- neue-units: keine
- desktop-aufrufer: n.a. (keine Validierungs- oder Antwortform-Änderung; DSAR-Module sind generische `{module, columns, records}`-Einträge)
- offen: (1) Der Export (`export.go`/Erasure) behandelt Notizen/Chat nicht — Löschung steht in separaten Units (Retention-Folgeunit `feat-retention-meeting-notes-and-chat-2y`). (2) `sender_name` im Chat und der Notiztext können Dritte nennen (Art. 15 Abs. 4) — wie bei den anderen Freitextmodulen unmaskiert. (3) Meeting-Notizen von Gästen ohne Benutzerkonto sind nicht erfasst (`author_id` ist FK auf `users`).

## Iteration 62 — feat-retention-meeting-notes-and-chat-2y — done — 2026-10-26 03:28
- commit: (siehe git log)
- gebaut: `MeetingNoteRetentionHandler` (`meeting_notes`) und `MeetingChatRetentionHandler` (`meeting_chat`) in `gdpr/retention_meeting_notes_chat.go`, beide im Registry-Block von `cmd/auth/main.go` verdrahtet. Uhr = Ende des zugehörigen Meetings, `COALESCE(actual_end, scheduled_end)` per Join auf `meetings` (tenant-gleich); Delete-only (`SupportsAction(anonymize)` = false), `meetings`-Zeile bleibt. Die 2 Jahre sind der `retention_days`-Wert der Policy, keine Konstante; Dry-Run ist der Worker-Default (`RETENTION_MODE`), `Plan` schreibt nie.
  Tests (`retention_meeting_notes_chat_test.go`, echte DB als kmuhub_app): alt (actual_end) und alt (nur scheduled_end) fällig, jünger nicht, `actual_end` schlägt altes `scheduled_end`, Fremd-Mandant nicht im Plan; Apply löscht idempotent, Meeting bleibt, Fremd-Mandant-Zeilen bleiben; Plan/Apply zusätzlich unter System-Kontext (RLS sieht alle), damit der Code-Filter selbst geprüft wird.
- gate: build ok | vet ok | lint ok (0 issues security/cmd/auth) | test ok (`internal/security/...` alle ok; `gdpr` 0 SKIP, DATABASE_URL als kmuhub_app) | migration n.a. | rls-smoke n.a. (keine Tabelle/Policy) | gateway n.a. (keine Route) | e2e n.a.
- coverage: internal/security/gdpr 76,1 % -> 76,2 %
- mutations-probe: (1) `actual_end` ignoriert (nur `scheduled_end`) -> Plan- und Apply-Test rot. (2) Tenant-Filter in Plan (WHERE und Join) entfernt -> Plan-Test rot (nur dank System-Kontext-Fall; unter RLS allein bleibt er unsichtbar). (3) Tenant-Filter im DELETE entfernt -> Apply-Test rot (ebenso). Jeweils aus Backup zurück, `cmp` sauber.
- verify vorgaenger: sauber (`e6f81999`: zwei Module über `userRowsModule`, tenant-gescoped, Test mit Kollegen- und Fremd-Mandant-Zeilen, kein Stub, keine Route/Proto/Tabelle)
- neue-units: keine
- desktop-aufrufer: n.a. (keine Validierungs- oder Antwortform-Änderung; `grep desktop/src/renderer/src` nach den Resource-Types ohne Treffer — die Policy-UI kennt keine feste Liste)
- offen: (1) Wirkt erst, wenn ein Admin eine Policy `meeting_notes` bzw. `meeting_chat` mit 730 Tagen anlegt — es gibt keinen Default-Seed, wie bei den anderen Handlern; im Dry-Run (Default) löscht nichts. (2) Notizen/Chat von Meetings ohne `actual_end` und mit `scheduled_end` in der Zukunft sind nie fällig, auch nicht, wenn das Meeting nie stattfand — erst nach Ablauf des geplanten Endes plus Frist. (3) Aufnahmen/Action-Items eines Meetings sind nicht Teil der Unit. (4) Erster Testlauf ließ Daten liegen (`defer pool.Close()` vor `t.Cleanup`), im Test auf `t.Cleanup(pool.Close)` korrigiert und die Reste lokal gelöscht.

## Iteration 63 — feat-erasure-deletes-rental-inspection-photos — done — 2026-10-26 03:35
- commit: (siehe git log)
- gebaut: `consent.PurgeInspectionPhotos` (`crm/consent/inspection_photos.go`) löscht vor `ScrubDependentPII` die Fotoobjekte hinter `rental_inspections.photo_urls` aller Mieten des Kontakts, in derselben Transaktion VOR dem Commit; Fehler → Rollback, der Aufrufer bekommt den Fehler, die Schlüssel bleiben referenziert und ein Retry löscht erneut (MinIO-Delete ist idempotent). Gelöscht wird nur, was als Schlüssel unter `{eigener tenant}/vermietung/` liegt und kein `..` enthält — `photo_urls` ist Client-Text, sonst ließen sich über eine Inspektion fremde Objekte löschen. Beide Pfade verdrahtet: `consent.AnonymizeContact` und `contact.Delete` über `SetObjectDeleter`; `cmd/crm` und `cmd/auth` (Contact-Retention-Handler) bauen den Store über neues `chatfile.NewMinIOStoreOrUnavailable` (nicht fatal bei MinIO-Ausfall beim Start, Delete liefert dann Fehler). `docker-compose.yml`: MINIO_*-Passthrough für `auth` und `crm` (gleiches `${VAR:-minioadmin}`-Muster wie die anderen Dienste).
  Tests (echte DB als kmuhub_app, Recording-Deleter): Fehler-Store → Erasure scheitert, nichts committet, Kontakt unverändert; danach genau a/b/c je einmal gelöscht (Duplikat über zwei Inspektionen dedupliziert; Fremd-Mandant-Schlüssel, externe URL, `..`-Pfad, anderer Scope bleiben unberührt); Geschwister-Kontakt und Fremd-Mandant behalten ihre Fotos; zweiter Lauf löscht nichts mehr. Dazu ein Hard-Delete-Test in `contact`.
- gate: build ok | vet ok | lint ok (0 issues crm/chat-file/cmd auth+crm) | test ok (`internal/crm/...`, `chat/file`, `security/gdpr` alle ok, 0 SKIP, DATABASE_URL als kmuhub_app) | migration n.a. | rls-smoke n.a. (keine Tabelle/Policy) | gateway n.a. (keine Route) | e2e n.a.
- coverage: internal/crm/consent 64,6 % -> 66,7 % (`PurgeInspectionPhotos` 76,5 %)
- mutations-probe: (1) Präfixprüfung auf `prefix[:1]` aufgeweicht -> Test rot. (2) `..`-Prüfung entfernt -> Test rot (`vermietung/../documents/secret.pdf` wird gelöscht). (3) `contact_id`-Filter in der Mietenauswahl entfernt -> Test rot (Geschwisterfotos gelöscht). (4) Delete-Fehler nur geloggt statt zurückgegeben -> Test rot (Erasure scheitert nicht). Jeweils aus Backup zurück, `cmp` sauber.
- verify vorgaenger: sauber (`183c18f2`: zwei Retention-Handler, Delete-only, tenant-gescoped, in `cmd/auth` verdrahtet, Test mit Fremd-Mandant und System-Kontext, kein Stub, keine Route/Proto/Tabelle)
- neue-units: keine
- desktop-aufrufer: n.a. (keine Validierungs- oder Antwortform-Änderung; Löschen läuft hinter bestehenden Routen)
- offen: (1) Deploy: `crm` und `auth` bekommen erstmals MINIO_*; die Werte laufen über die Produktions-Env wie bei den anderen Diensten, kein `RequireX`. Ohne erreichbares MinIO scheitert nur die Löschung eines Kontakts MIT Inspektionsfotos (Fehler, Retry), alles andere läuft. (2) Fotos mit externer URL oder fremdem Schlüssel werden bewusst nicht gelöscht (Warnung im Log), die Referenz wird weiterhin geleert. (3) Fehler landen als Rückgabefehler (Erasure scheitert und wird wiederholt), nicht im Erasure-Log — die Kontakt-Anonymisierung hat keinen. (4) Lieferantendaten sind laut Entscheidung bewusst nicht Teil der Unit.

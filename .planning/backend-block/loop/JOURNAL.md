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

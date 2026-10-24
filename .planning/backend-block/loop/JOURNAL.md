# Backend-Nachtloop — Journal Lauf 16

Append-only. Ein Eintrag je Iteration, **ans Dateiende**, nie einsortieren. Form und Pflichtzeilen
stehen in `ITERATION.md` Schritt 6.

Frühere Läufe liegen vollständig im Archiv:
`archive/lauf-1-2/` (58 Units) · `archive/lauf-3/` (61) · `archive/lauf-4/` (54) ·
`archive/lauf-5/` (41) · `archive/lauf-6/` (46) · `archive/lauf-7/` (71) ·
`archive/lauf-8/` (94) · `archive/lauf-9/` (37) · `archive/lauf-10/` (93) ·
`archive/lauf-11/` (131 Einträge, davon 121 done) · `archive/lauf-12/` (99, davon 41 done) ·
`archive/lauf-13/` (120, davon 21 done — abgestürzt) · `archive/lauf-14/` (110, alle done —
Backlog leergelaufen, gemergt als `cadf4bf4`) · `archive/lauf-15/` (74, alle done — Backlog
leergelaufen, PR #27). Alle inkl. Laufbilanz.

---

## Laufkontext

- **Ausgangspunkt:** Lauf 15 gemergt als `da9fc602` (PR #27) und deployt (CD 2026-10-24
  10:44 UTC, Smoke 27/27). `backend-loop` = `main` plus Staging-Commits (Entscheidungsrunde in
  NEXT, E2E-Stack-Erweiterung, dieser Lauf). CI grün auf `da9fc602` (Run 40085600528, alle
  fünf Jobs inkl. E2E, plus CI Desktop).
- **Migrationen:** Repo-Kopf = lokaler DB-Kopf = Produktionskopf = **332**, `dirty = f`. Lauf 16
  bringt mehrere Seed-Migrationen. Nummer immer zur Laufzeit ermitteln:
  `ls backend/migrations | grep -E "^[0-9]{6}" | sort | tail -1`
- **Coverage-Start:** **73,7 %** gesamt bei Gate 15 % (Run 40085600528). Bezugswerte je Paket im
  Kopf von `BACKLOG.yml`.
- **Queue:** 52 Units, roter Faden „Echte Nutzer, Teil 2: der Mitarbeiter, nicht der Admin —
  und jede Eingabe kommt an“. Sicherheit (1), Totalausfälle für member/manager (16),
  entschiedene Einzelfixes (11), Geister-Routen und Scans (14), E2E als member (8),
  Coverage (2). Begründung, sechs neue Regeln und Sperren im Kopf von `BACKLOG.yml`.
- **CI-E2E erweitert:** der E2E-Job startet seit dem Staging zusätzlich `inventar` (:9110),
  `einkauf` (:9111) und `schichten` (:9115) mit `COSMI_MODULE_*_ENABLED=true` (nur auf
  `backend-loop`). Lokal geprüft bis zum Port-Bind (Config und DB ok, `bind` verbietet die
  Sandbox) — abgenommen erst im Morgen-PR.
- **Nicht im Lauf:** sieben Desktop-Hooks mit rohem `fetch` ohne Auth-Header
  (`fe-raw-fetch-without-auth-header` in NEXT). Ein Gateway-Fix hilft dort nicht.

## Was aus Lauf 15 gelernt wurde — vier Dinge, die dich direkt betreffen

1. **Admin überdeckt alles.** Jede Prüfung in Lauf 15 lief als Admin (462 von 463 Keys). Die
   Vorbereitung fand danach ganze Module, die für member 403 liefern. Teste als member
   (Regel 6), und prüfe vor jedem Grant die Datengrenze (Regel 1).
2. **Neue E2E-Flows finden Altbugs.** Der Kalender-Flow fand `include_inactive`, das schon
   vor Lauf 15 kaputt war. Scheitert ein Flow-Schritt, ist das ein Fund — Schritt nicht
   weglassen, Fix-Unit an den Kopf.
3. **Read-Modify-Write ist Datenverlust.** `UpdateItem` schrieb den Bestand aus einem
   ungesperrten Read zurück und löschte parallele Wareneingänge (`8cc079ce`). Zähler, Salden,
   Summen nur in der gesperrten Transaktion ändern.
4. **CI läuft erst im Morgen-PR.** Lokal fehlen `-race` und der E2E-Stack. Tests mit
   Goroutinen und jede E2E-Iteration in `offen:` vermerken.

---

## Iteration 1 — fix-member-can-write-tenant-settings-and-branding — done — 2026-10-24 17:00
- commit: (siehe git log, Subject `fix(settings): keep members off tenant-wide presence config and holiday seeding`)
- gebaut: `PUT /video/presence/config` (tenantweiter Away-Timeout, Service ohne Admin-Prüfung) und `POST /calendar/holidays/seed` (Nager-Fetch, Service ohne Admin-Prüfung) zusätzlich hinter `settings:tenant:manage` (Katalog-Key aus 000256, nur admin + it_admin; it_admin fehlt `settings:write` bzw. `calendars:admin`, also effektiv admin wie vorher). Key statt `RequireRole("admin")`, weil er existiert, semantisch passt (tenantweite Einstellung) und auch tenant-eigene Admin-Rollen trägt; Admin-Tokens haben ihn seit 000256, kein Re-Login. `PUT /admin/branding` und `PUT /settings/{m}/tenant` belegt sicher: `callerMayWriteTenantSettings` lässt nur admin oder Modul-Leiter des Moduls durch — Gateway bewusst NICHT verengt, sonst bricht der Modul-Leiter (meist Rolle member). `calendars/{id}/members*` sicher (Service prüft Besitzer bzw. Kalender-Admin-Mitgliedschaft), `resources:admin` schützt keine Route. Kein Grant entzogen.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (settings 59 Tests, davon 10 DB-Tests real als kmuhub_app, 0 Skips; gateway ok inkl. TestOpenAPIRouteDrift) | migration n.a. | rls-smoke n.a.
- coverage: internal/settings 82,8 % -> 82,8 % · internal/gateway 76,6 % -> 76,6 % (neue Tests belegen Verhalten bereits abgedeckter Zeilen: Guard-Kette und echter Role-Checker mit Seed-Rollen statt Fake)
- mutations-probe: (1) `callerMayWriteTenantSettings` auf `return true` -> `TestTenantWrites_MemberAndManagerRolesRejectedAdminAllowed` rot (5 Assertions: member/manager auf branding+tenant, fremder Modul-Leiter auf branding); (2) `settings:tenant:manage` an presence/config entfernt -> `TestTenantWideSettingsGuards_MemberVsAdmin` rot (member 503 statt 403). Beide zurückgedreht, Diff sauber.
- verify vorgaenger: n.a. (erste Iteration von Lauf 16, kein Vorgänger-Commit im Journal)
- neue-units: keine
- desktop-aufrufer: `PUT /video/presence/config` -> `api/video-client.ts:243` + `hooks/usePresence.ts:69` (`useUpdatePresenceConfig`) -> kein .tsx-Konsument, passt · `POST /calendar/holidays/seed` -> `hooks/useHolidays.ts:44` (`useSeedHolidays`, als „Admin action“ kommentiert) -> kein Konsument, passt · branding/tenant/user/preferences unverändert
- offen: Real-Token-Prüfung der Rollen nur über Router-Test mit den aus `role_permissions` rekonstruierten Key-Mengen; E2E als member für diese Routen gibt es noch nicht. Die Spec-Änderung ist nur der `summary`-Text (YAML parst, Drift-Test grün); `desktop/src/renderer/src/api/types.ts` nicht regeneriert (Desktop tabu, nur Summary betroffen).

## Iteration 2 — feat-hr-self-service-key-for-members — done — 2026-10-24 17:10
- commit: (siehe git log, Subject `feat(hr): let members and managers use the HR self-service routes`)
- gebaut: Migration 000333 sät `hr:self` (`hr`,`self`) und gibt ihn member/manager mit Scope `own`, admin mit `all` (zweimal anwendbar, Down entfernt Grants und Permission). In `route_hr.go` nehmen 20 Self-Service-Routen (Urlaub anlegen/lesen/zurückziehen/Saldo/Typen; Stempeluhr, Pause, active/status, POST /entries, summary daily/weekly, corrections anlegen, balance, analytics, weeks/status, weeks/submit) additiv `RequirePermissionAny(hr:read|write, hr:self)`. `HandleListLeaveRequests`/`HandleGetLeaveRequest` nutzen jetzt `ownerFilterForScopeAny(hr:read, hr:self)`: ein Token mit nur `hr:self` (own) sieht keine fremden Anträge, `employee_id`/`user_id` in der Query erweitern nichts. Alle umgestellten Handler nehmen die Identität aus dem JWT (geprüft); `GET /hr/time/entries` liest `employee_id` aus der Query und bleibt bewusst auf `hr:read` (Folge-Unit `fix-hr-own-scope-foreign-balance-and-time-entries`). approve/reject, `/sick`, `/balance/{userId}`, `/employees`, `/settings` bleiben zu.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (gateway inkl. TestOpenAPIRouteDrift; testutil `TestEveryRouteGuardHasAUsablePermission`, settings, auth, middleware) | migration ok (up/down/up lokal, zweites up No-op, `schema_migrations` 333 clean) | rls-smoke n.a. (keine Tabelle angefasst; `role_permissions`-Policies unverändert)
- sql-probe: `hr:self` -> admin|all, manager|own, member|own
- coverage: internal/gateway 76,6 % -> 76,6 % (neue Tests belegen Guard-Verdrahtung und Own-Narrowing)
- mutations-probe: (1) Guard von `POST /hr/leave/requests` zurück auf `RequirePermission(hr,write)` -> `TestHRSelfKey_OpensOnlyTheSelfServiceRoutes` rot; (2) `ownerFilterForScopeAny` zurück auf `ownerFilterForScope(hr,read)` -> `TestHRSelfKey_OwnScopeNarrowsLeaveReadsLikeTheCoarseKey` rot. Beide zurückgedreht, Diff sauber.
- verify vorgaenger: sauber (Commit 27594bfb: Guards additiv ergänzt, kein Alt-Key verloren, Tests mit echtem Role-Checker)
- neue-units: keine
- desktop-aufrufer: `hr-client.ts` Leave (`/hr/leave/requests|balance|types|cancel`) und Zeit (`clock-in|out`, `break/*`, `active`, `status`, `summary/*`, `balance`, `analytics`, `weeks/*`, `corrections`, `POST entries`) -> Antwortformen unverändert, nur Zugriff wird erweitert -> passt. Offen im FE-Pfad: `GET /hr/time/entries` und `/hr/time/projects` bleiben für member 403 (Folge-Units).
- offen: Neuer Key wirkt in Produktion erst nach Re-Login der betroffenen Nutzer (JWT-gebackene Permissions). Migration 333 noch nicht in Produktion. Echte Token-Prüfung als member nur über Router-Test mit rekonstruierten Key-Mengen; E2E als member folgt in Block D.

## Iteration 3 — fix-hr-own-scope-foreign-balance-and-time-entries — done — 2026-10-24 17:25
- commit: (siehe git log, Subject `fix(hr): pin hr:self at scope own to the caller on balance and time entries`)
- gebaut: Helper `restrictToOwnEmployee` (`helpers.go`, baut auf `ownerFilterForScopeAny`): Scope own + fremde ID -> 403, Scope own ohne ID -> eigene ID, breitere Scopes unverändert. `GET /hr/leave/balance/{userId}` und `GET /hr/time/entries` nutzen ihn und stehen jetzt auf `RequirePermissionAny(hr:read, hr:self)`. Spec: 403 an beiden Operationen ergänzt (StatusCodeDrift-Test hat es erzwungen), `swagger-cli validate` ok.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (gateway komplett, inkl. TestOpenAPIRouteDrift und StatusCodeDrift; 0 Skips) | migration n.a. | rls-smoke n.a.
- coverage: internal/gateway 76,6 % -> 76,6 % (neue Tests belegen Verhalten, keine neue Fläche)
- mutations-probe: Pin in `restrictToOwnEmployee` (`if ownerID == nil` -> `if true`) -> `TestHROwnScope_ForeignBalanceAndTimeEntries` rot; zurückgedreht, Diff sauber.
- verify vorgaenger: sauber (Commit 175160fb: Guards additiv, Alt-Keys erhalten, Migration 333 mit Down, kein Stub)
- neue-units: keine
- desktop-aufrufer: `hr-client.ts:407` `/hr/leave/balance/${userId}` -> `MemberProfileContent.tsx:87` fragt Saldo des angesehenen Teammitglieds: für member künftig 403 bei fremden Profilen (gewollt), eigenes passt · `hr-client.ts:674` `/hr/time/entries` -> `ZeiterfassungTab.tsx:131` sendet `employee_id=<eigene id>` -> passt; `:134` `status=correction_pending` ohne employee_id liefert für member nur die eigenen (vorher 403, bei hr:read/all weiter alle) · `ExportDialog.tsx:69` ohne Filter -> für member eigene Einträge. Antwortformen unverändert.
- offen: Wirkt für member/manager erst nach Re-Login (hr:self aus Migration 333). Die Team-Korrekturliste (`correction_pending` ohne employee_id) zeigt einem manager mit nur hr:self nur eigene Einträge; Team-Sicht folgt mit `feat-hr-reporting-line-and-team-time-scope`.

## Iteration 4 — feat-hr-reporting-line-and-team-time-scope — done — 2026-10-24 17:44
- commit: (siehe git log, Subject `feat(hr): scope the team time overview to the caller's reporting line`)
- gebaut: `GET /hr/time/team` listet nur noch bei Scope `all` den ganzen Tenant. Der Gateway löst den Scope über neues `narrowestScopeAny(hr:read, zeiterfassung:team:view)` auf (der engere gewinnt, unbekannter Wert = `own`) und gibt ihn im neuen Proto-Feld `GetTeamTimeReq.scope` weiter (`hr.pb.go` regeneriert). `HRGRPCServer.GetTeamTime` filtert fail-closed: nur exakt `all` lässt den Filter weg, alles andere (leer, `team`, `own`, Unsinn) setzt `manager_user_id = callerID(ctx)`; ohne Aufrufer `Unauthenticated`. Filter sitzt in der SQL-Query (`EmployeeFilter.ManagerUserID` gab es schon, kein neuer Helper nötig). Migration 000334 stellt `zeiterfassung:team:view` für den System-Preset `manager` auf `team` (admin/hr_admin bleiben `all`). Stale Kommentar in `middleware/auth.go` nachgezogen; der Testkommentar mit der nie angelegten Unit ist ersetzt.
- entscheidung: Proto-Feld statt weitergereichter Permission, weil `UserScopesKey` nicht im gRPC-Kontext ankommt (nur der Gateway-Kontext trägt `claims.Scopes`); der Scope-String ist ein Gateway-Befund an einen vertrauenswürdigen internen Aufrufer, die Identität kommt weiter aus dem Kontext, nie aus der Nachricht.
- nebenbei behoben: `GetTeamTime` rief `employeeRepo.List` mit `PerPage: 500` auf; das Repo kappt >100 auf 20. Die Team-Zeitliste zeigte also in jedem Tenant mit >20 Mitarbeitern nur 20. Jetzt `listAllEmployees` (Seiten à 100 bis `total` erreicht). `grep PerPage: 3-stellig` fand keinen zweiten Fall.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (hr/..., middleware, server, gateway inkl. TestOpenAPIRouteDrift; DB-Tests real als kmuhub_app, 0 Skips) | migration ok (up/down/up, zweites up No-op, Kopf 334 clean; SQL-Probe: admin|all, hr_admin|all, manager|team) | rls-smoke n.a. (keine Tabelle/Policy angefasst)
- coverage: internal/biz/hr/timetracking 65,6 % -> 65,9 %
- mutations-probe: (1) `ManagerUserID` aus dem Service-Filter entfernt -> `TestGetTeamTime_ScopeNarrowsToReportingLine` (DB) und `TestGetTeamTime_ListsEveryEmployeeAcrossPages` rot; (2) Server fail-open (Scope-Vergleich nie wahr) -> `TestGetTeamTime_ScopeNarrowsToReportingLine` und `TestGetTeamTime_NonAllScopeNeedsACaller` rot; (3) unbekannter Scope in `narrowestScopeAny` -> `all` -> `TestNarrowestScopeAny` rot. Alle zurückgedreht, Diff sauber.
- verify vorgaenger: sauber (Commit e40284c6: `restrictToOwnEmployee` additiv, Alt-Keys erhalten, Spec 403 ergänzt, kein Stub)
- neue-units: keine
- desktop-aufrufer: `GET /hr/time/team` -> `hr-client.ts:731` (`getTeamTime`) liest `raw.team ?? raw.rows ?? []` -> Antwortform unverändert (`{team: []}`, nie null); für Manager nur weniger Zeilen -> passt. Spec: nur der `summary`-Text, `swagger-cli validate` ok.
- offen: Wirkt für Manager erst nach Re-Login (Scope im JWT); Alt-Token lösen `zeiterfassung:team:view` weiter zu `all` auf. Tenant-eigene Rollen, die vom Manager-Preset abgeleitet wurden, behalten `all` (Migration trifft nur System-Presets). Der Berichtslinien-Helper für `feat-hr-leave-approve-by-manager-for-reporting-line` ist `EmployeeFilter.ManagerUserID` (Liste) bzw. `changerequest.approveScopeAllows`/`ManagerOf` (Einzelprüfung). Kein Goroutinen-Code, daher kein `-race`-Vorbehalt.

## Iteration 5 — feat-hr-leave-approve-by-manager-for-reporting-line — done — 2026-10-24 17:25
- commit: (siehe git log, Subject `feat(hr): let managers decide leave requests of their reporting line`)
- gebaut: `POST /hr/leave/requests/{id}/approve|reject` nehmen additiv `team:absence:approve` (`RequirePermissionAny(hr:write, team:absence:approve)`). Der Gateway gibt den engeren Scope beider Keys (`leaveDecisionScope`, `narrowestScopeAny`) im neuen Proto-Feld `scope` weiter (`hr.pb.go` regeneriert); `HRGRPCServer` setzt fail-closed `leave.InReportingLineOnly()` für alles außer exakt `all` — dann entscheidet nur der direkte Vorgesetzte (`ManagerUserID == approver`), der HR-Fallback „Mitarbeiter ohne Vorgesetzten“ entfällt, Self-Approval bleibt verboten. `GET /hr/leave/requests`: ein Aufrufer mit `team:absence:approve` jenseits `own` (Manager mit `hr:self` own) bekommt ohne Mitarbeiterfilter die Anträge seiner Berichtslinie (`ListLeaveRequestsReq.reporting_line_only`, Repo-Filter `EXISTS` auf `hr_employee_profiles.manager_user_id`, Identität aus dem Kontext); fragt er nach der eigenen ID (Selbstservice, Profil-Tab), bleibt es die eigene Liste. Migration 000335 setzt `team:absence:approve` und `team:absence:read` für den Preset `manager` auf `team` (admin/hr_admin bleiben `all`; member-`read` nicht angefasst, wird von keiner Route genutzt). Spec: `summary` und 403 an approve/reject.
- entscheidung: Listen-Guard bleibt `hr:read|hr:self` (der Manager hat `hr:self`); der Berichtslinien-Zweig hängt am `approve`-Key, nicht an `team:absence:read`, weil member `read` auf `all` hält und sonst den ganzen Tenant lesen würde. Scope `all` (admin, hr_admin, Alt-Token) behält den bisherigen `verifyApprover`-Pfad unverändert.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (hr/..., server, gateway inkl. TestOpenAPIRouteDrift/StatusCodeDrift, middleware, testutil; DB-Tests real als kmuhub_app, 0 Skips; `swagger-cli validate` ok) | migration ok (up/down/up, zweites up No-op, Kopf 335 clean; SQL-Probe: manager approve|team, read|team) | rls-smoke n.a. (keine Tabelle/Policy angefasst)
- coverage: internal/biz/hr/leave 85,7 % -> 86,5 % (Start laut Backlog 85,7; selbst vorher nicht separat gemessen)
- mutations-probe: (1) Berichtslinien-Zweig in `verifyApprover` abgeschaltet -> `TestApproveLeaveRequest_ReportingLineOnly` rot; (2) `leaveDecisionScope` immer `all` -> `TestLeaveDecide_ManagerReachesApproveAndRejectWithScopeTeam` rot; (3) Repo-Filter `ManagerUserID` abgeschaltet -> `TestPostgresLeaveRequestRepo_List_ManagerUserIDLimitsToDirectReports` (DB) rot; (4) Listen-Zweig `reportingLine = true` -> `false` -> `TestLeaveList_ManagerSeesTheReportingLineAndStillTheirOwnRequests` rot; (5) `leaveApprovalOptions` fail-open -> `TestLeaveApprovalOptions_FailClosed` rot. Alle zurückgedreht, Diff sauber.
- verify vorgaenger: sauber (Commit d252e520: Proto-Feld mit pb.go, Guard unverändert, Server fail-closed, Migration 334 mit Down, kein Stub)
- neue-units: keine
- desktop-aufrufer: `hr-client.ts` `approveRequest/rejectRequest` (Body `{comment}`, Antwort ungelesen) -> passt · `listRequests` -> `TeamPage.tsx:177` (`status=pending`, kein Mitarbeiterfilter, Genehmigungs-Tab; `canApprove={has('team:absence:approve')}` Z. 515 deckt sich mit dem neuen Guard) zeigt für Manager jetzt die Berichtslinie statt der eigenen Anträge · `SelfServiceView.tsx:188` (`user_id=<self>`) und `AbwesenheitenTab.tsx:122` (`employee_id=<self>`) -> eigene Liste unverändert · `HRApprovalDialog.tsx:60` (`status=approved`, ohne Filter) zeigt dem Manager ebenfalls nur die Berichtslinie. Antwortform `{leave_requests, total}` unverändert.
- offen: Wirkt für Manager erst nach Re-Login; ein vor der Migration gebackener Manager-Token trägt `team:absence:approve` weiter auf `all` und entscheidet bis zum Ablauf tenantweit (gleicher Vorbehalt wie 000334). Admin/hr_admin mit Scope `all` laufen weiter durch `verifyApprover`: Hat der Mitarbeiter einen anderen Vorgesetzten, bekommt auch der Admin 403 — bestehendes Verhalten, nicht angefasst (Produktfrage, ob `all` den Manager-Check überspringen soll). `GET /hr/leave/requests/{id}` bleibt für den Manager auf eigene Anträge verengt (kein FE-Aufrufer). Kein Goroutinen-Code, kein `-race`-Vorbehalt.

## Iteration 6 — fix-hr-employee-directory-for-members — done — 2026-10-24 17:34
- commit: (siehe git log, Subject `fix(hr): open the team directory to members with a narrow projection`)
- gebaut: `GET /hr/employees` und `GET /hr/employees/me` additiv auf `RequirePermissionAny(hr:read, team:employee:read)` (Key seit 000256 bei member/all, manager/team, admin — kein Seed nötig). Ohne `hr:read` liefert die Liste den eigenen Wire-Typ `employeeDirectoryEntry` (id, user_id, user_name, user_email, department, position_title, manager_user_id, manager_name, status; omitempty wie protojson), kein geleertes Proto. `/me` liefert das volle eigene Profil (ID nur aus dem JWT). `GET /hr/employees/{id}` und `/hr/absences/calendar` bleiben auf `hr:read`. Spec: description + 403 an beiden Operationen, `swagger-cli validate` ok.
- entscheidung: Scope `team` beim manager bewusst ignoriert (tenantweites Verzeichnis, `lean:`-Marker am Code): Mention-Picker, Helpdesk-Zuweiser und `useTeamPermissions`/OrgChart (bauen den Berichtsbaum aus der vollen Liste) brauchen alle Kollegen; die Verzeichnisfelder sind im KMU nicht vertraulich. `/hr/absences/calendar` NICHT umgestellt: die Route liefert `leave_type_name` im Klartext, inkl. Leave-Type `krank` (000046) — Gesundheitsdatum, nach NEXT.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (gateway komplett inkl. TestOpenAPIRouteDrift/StatusCodeDrift, 0 Skips; testutil inkl. TestEveryRouteGuardHasAUsablePermission) | migration n.a. | rls-smoke n.a.
- coverage: internal/gateway 76,6 % -> 76,7 % (Vorher-Wert aus Iteration 4/5, nicht separat neu gemessen)
- mutations-probe: (1) Projektion abgeschaltet (`if false && !callerHasPermission…`, volles Proto an member) -> `TestEmployeeDirectory_TeamEmployeeReadGetsOnlyTheDirectoryFields` rot; (2) Listen-Guard zurück auf `RequirePermission(hr,read)` -> derselbe Test rot (403). Beide zurückgedreht, Diff sauber.
- verify vorgaenger: sauber (Commit c6497cca: approve/reject additiv `hr:write|team:absence:approve`, `hr.pb.go` regeneriert, Server fail-closed, Migration 335 mit Down, kein Stub)
- neue-units: `decide-absence-calendar-for-members`, `fe-team-card-contract-type-default-full-time` (beide BACKLOG-NEXT.yml: Entscheidung bzw. Desktop-Territorium)
- desktop-aufrufer: `GET /hr/employees` -> `hr-client.ts:873` `adaptEmployee` (alle Felder mit `??`, fehlende -> undefined) -> `MentionAutocomplete.tsx` (id, userId, userName, positionTitle) passt · `TeamStatus.tsx` (userId, userName, department, positionTitle) passt · `HelpdeskPage.tsx` (userId, userName, userEmail) passt · `ConversationThreadHeader.tsx` (userId, userName, userEmail) passt · `SchichtenPage.tsx` (id, userName, department, positionTitle, initials — `initials` liefert auch die volle Form nicht) passt · `useWikiUsers.ts`/`WikiSuggest.tsx` (userId, userName, positionTitle) passt · `OrgChart.tsx`/`useTeamPermissions.ts` (managerUserId, userId) passt · `AbsenceCalendar.tsx` (department; `startDate` gehört zu Kalender-Einträgen, nicht zum Profil) passt · `TeamPage.tsx` passt bis auf Z. 990 `contractType`: fehlt -> `normaliseContractType` -> `full_time`, Karte zeigt „Vollzeit“ für jeden (`fe-team-card-contract-type-default-full-time`) · Admin-Dialoge (CreateEmployeeWizard, EditMember, InviteMember, Offboard, PayrollPrep, PersonnelDocuments) laufen mit `hr:read` und bekommen weiter die volle Form · `GET /hr/employees/me` -> `hr-client.ts:894` `getSelf` unverändert volle Form.
- offen: Kein Re-Login nötig (`team:employee:read` ist seit 000256 in den Tokens). Ein Token mit `hr:read` aber verweigertem `team:employee:read` (Per-User-Deny) bekommt durch die Deny-Semantik von `RequirePermissionAny` 403 statt der vollen Liste — bestehendes Verhalten des Any-Guards, nicht angefasst. Abwesenheitskalender bleibt für member leer bis zur Entscheidung.

## Iteration 7 — fix-documents-helper-routes-accept-fine-keys — done — 2026-10-24 17:55
- commit: (siehe git log, Subject `fix(document): let members reach the document helper routes through the file keys`)
- gebaut: Elf Dokument-Routen auf additive Any-Guards (grob + `documents:file:*`) umgestellt, member hat nur die feinen Keys. Entscheidungstabelle: `GET /documents/virtual`, `GET /documents/search`, `GET /documents/tags`, `GET /documents/files/{id}/links`, `GET /documents/wopi/discovery` -> `docRead` (`documents:read | file:read`) · `POST /documents/tags`, `POST|DELETE /documents/tags/file`, `POST|DELETE /documents/files/{id}/links`, `DELETE /documents/links/{id}` -> `docEdit` (Metadaten der Datei) · `POST /documents/wopi/token` -> `docEdit` (der gRPC-Server stellt den Token immer schreibfähig aus, `GenerateWOPIToken` -> `Generate(..., true)`; das FE ruft ihn nur bei „In OnlyOffice bearbeiten“) · `POST /documents/files/{id}/copy` -> `docUpload` (Kopie = neue Datei; member hat `file:upload` all). Unverändert: `DELETE /tags/{id}` (`documents:delete`), `initialize-user|team`.
- entscheidung: Kein Seed nötig, alle Keys existieren (member: file:read all, upload all, edit own). `file:edit` ist bei member Scope own, aber das Gateway hat ihn auf diesen Routen nie verengt (`UpdateFile`, Kommentare, Move sind identisch) und der Service kennt auf Tag/Link/WOPI keinen Besitz — die Routen wirken wie bisher tenantweit; Dokumente sind laut Entscheidung 2026-10-24 tenantweit sichtbar, ein Zugriffsmodell ist nicht Teil des Laufs. `initialize-user` nicht geöffnet: nimmt `user_id` aus dem Body (ein member könnte fremde Räume anlegen), `useInitializeUserSpace` hat keinen Aufrufer.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (gateway komplett inkl. TestOpenAPIRouteDrift/StatusCodeDrift; middleware, testutil) | migration n.a. | rls-smoke n.a. (keine Tabelle angefasst; Gateway-Tests brauchen keine DB)
- coverage: internal/gateway 76,6 % -> 76,7 % (keine neue Fläche, Tests belegen Guards; Vorher-Wert aus Backlog/Iteration 6)
- mutations-probe: `GET /documents/virtual` zurück auf `RequirePermission(documents,read)` -> `TestCapabilityGuards_AdditiveWiring/virtual_files,_catalogue_read_key_only` rot; zurückgedreht, Diff sauber. Alte Pin-Tests „tag create / entity link delete / copy bleiben coarse-only“ umgeschrieben (sie hielten genau den Totalausfall fest).
- verify vorgaenger: sauber (Commit aec34774: Guard additiv `hr:read | team:employee:read`, eigener schmaler Wire-Typ, Spec-403 ergänzt, kein Stub, kein neuer Key)
- neue-units: keine
- desktop-aufrufer: nur Zugriff erweitert, Routen/Antwortformen unverändert · `document-client.ts:336` copy, `:443-468` tags, `:482-499` links, `:522` search, `:528` virtual, `:539` wopi/token -> `DokumentePage.tsx:234-240` (virtual beim Öffnen), `:602` (WOPI nur beim Bearbeiten), `FilePreviewModal.tsx:166`, `FileDetailPanel.tsx:114` (Tags) -> passt.
- offen: Keine Migration, kein Re-Login nötig (member trägt die `documents:file:*`-Keys schon). Ein member mit nur `file:read` (z. B. eigene Tenant-Rolle) bekommt für Tags setzen/Link/Kopie weiter 403, wie gewollt. Der Besitz-Scope `own` von `file:edit` wird auf Tag/Link/WOPI weiter nicht durchgesetzt (Teil von `decide-document-access-baseline` in NEXT). Kein Goroutinen-Code.

## Iteration 8 — fix-calendar-and-chat-own-delete-for-members — done — 2026-10-24 17:50
- commit: (siehe git log, Subject `fix(auth): grant members and managers the own-delete keys for events, calendars, chat messages and files`)
- gebaut: Migration 000336 gibt den System-Presets `member` und `manager` `calendars:delete`, `messages:delete`, `files:delete` mit Scope `own` (admin hatte sie schon auf `all`). Kein Gateway-Code geändert: die Guards bleiben die groben Keys. Regel 1 vorab belegt, Besitzprüfung sitzt in allen vier Pfaden im Service (Scope `own` wird dort durchgesetzt, nicht im Gateway): Termin `event.Service.DeleteEvent` (`work/event/service.go:413`, `CreatedBy != actor` -> `ErrNotEventCreator`), Kalender `calendar.Service.Delete` (`work/calendar/service.go:174`, `OwnerID != actor` -> `ErrNotCalendarOwner`, Default-Kalender gesperrt), Nachricht `message.Service.Delete` (`chat/message/service.go:366`, Autor oder Kanal-Moderator, sonst `ErrNotAuthorized`/`ErrNotChannelMember`), Chat-Datei `file.Service.Delete` (`chat/file/service.go:293`, Uploader oder Kanal-Moderator). Der Gateway reicht die Akteur-ID aus dem JWT durch (`middleware.GetUserID`), nie aus dem Body. `calendars:delete` deckt zusätzlich `DELETE /calendar/categories/{id}` ab; `DeleteCategory` filtert im SQL auf `user_id` (`work/calendar/postgres_repository.go:337`) -> ebenfalls nur eigene.
- tests: neuer DB-Test `TestPresetRoles_DB_MemberAndManagerHoldTheOwnDeleteKeys` (liest über `GetUserPermissions`, die Abfrage fürs JWT; member/manager/admin halten die Keys, extern nicht) und Router-Test `TestOwnDeleteRoutes_AcceptTheDeleteKeyAndNothingElse` (vier Routen: Delete-Key -> Guard offen, Write-Keys -> 403). Fremd-Fälle decken die bestehenden Service-Tests ab (`calendar/service_test.go:655/841/1090`, `chat/message/service_test.go:698`, `chat/file/service_test.go:666`).
- gate: build ok | vet ok | lint ok (0 issues) | test ok (gateway inkl. TestOpenAPIRouteDrift, auth, testutil inkl. TestEveryRouteGuardHasAUsablePermission, chat, work, middleware, server; DB-Tests real als kmuhub_app, 0 Skips; ein erster Lauf ohne `-p 2` riss die Connection-Slots der lokalen DB, Wiederholung mit `-p 2` grün) | migration ok (up/down/up, zweites up No-op, Kopf 336 clean; SQL-Probe: member/manager `calendars:delete|messages:delete|files:delete = own`, admin `all`, down entfernt genau die sechs Zeilen) | rls-smoke n.a. (keine Tabelle/Policy angefasst)
- coverage: internal/gateway 76,7 % -> 76,7 % (nur Guard-Tests, keine neue Fläche; Vorher-Wert aus Iteration 6/7)
- mutations-probe: (1) Grant `member`/`files:delete` per SQL entfernt -> `TestPresetRoles_DB_…` rot, wiederhergestellt; (2) Autor-Prüfung in `message.Service.Delete` abgeschaltet -> `TestService_Delete` (und `TestService_Update`, derselbe Ausdruck) rot, zurückgedreht; (3) Guard `DELETE /messages/{id}` auf `messages:write` -> `TestOwnDeleteRoutes_…` rot, zurückgedreht. Diff sauber.
- verify vorgaenger: sauber (Commit bae8889e: alle elf Routen additiv, `docEdit`/`docUpload`/`docRead` behalten die groben Keys, kein Seed nötig, kein Stub)
- neue-units: keine
- desktop-aufrufer: n.a. (nur Rollen-Grants, keine Validierungs- oder Antwortform-Änderung; die Aufrufer `KalenderPage.tsx:566-569`, `MessageBubble.tsx:306-328` sind die, die bisher 403 bekamen)
- offen: Wirkt für member/manager erst nach Re-Login (Berechtigungen im JWT gebacken). Ein Kanal-Moderator (Kanalrolle admin/owner) darf fremde Nachrichten und Dateien in seinem Kanal löschen — das Modell kannte das schon, es war nur für Nicht-Admins unerreichbar; Scope `own` beschreibt die Absicht, das Gateway verengt nicht. Tenant-eigene Rollen, die vom member-Preset kopiert wurden, bekommen die Keys nicht (Migration trifft nur System-Presets). Kein Goroutinen-Code.

## Iteration 9 — fix-dialer-outcomes-readable-by-agents — done — 2026-10-24 17:51
- commit: (siehe git log, Subject `fix(dialer): let agents read the call outcome list through calls:write`)
- gebaut: `GET /dialer/outcomes` additiv auf `RequirePermissionAny(dialer:outcomes:manage, dialer:calls:write)` (`route_dialer.go`, Variable `outcomesRead`). POST/PUT/DELETE der Outcomes bleiben `outcomes:manage`. Kein Seed nötig (member hält `dialer:calls:write`, Migration 000256). Sechs Router-Tests in `TestCapabilityGuards_AdditiveWiring`: GET mit `calls:write` offen, GET mit Alt-Key offen, GET ohne beide 403, POST/PUT/DELETE mit `calls:write` 403.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (gateway komplett inkl. TestOpenAPIRouteDrift, middleware, testutil; gateway-Tests brauchen keine DB) | migration n.a. | rls-smoke n.a.
- coverage: internal/gateway 76,7 % -> 76,7 % (nur Guard-Tests; Vorher-Wert aus Iteration 6-8, nicht separat neu gemessen)
- mutations-probe: `calls:write` aus dem GET-Guard entfernt (`RequirePermission(outcomes, manage)`) -> `TestCapabilityGuards_AdditiveWiring/outcomes_list,_calls:write_grants_the_read` rot; zurückgedreht, Gateway-Suite wieder grün, Diff sauber.
- verify vorgaenger: sauber (Commit 6be4a33f: reine Seed-Migration 000336 mit Down, keine Guard-Änderung, kein Stub, Tests vorhanden)
- neue-units: `fix-dialer-agent-status-for-members` (VOR die erste `todo`-Unit gesetzt: Totalausfall — `GET`/`PUT /dialer/agent-status` verlangen `dialer:agent:manage`, das hat member nicht; Workspace lädt und setzt den eigenen Status, beides 403)
- desktop-aufrufer: `GET /dialer/outcomes` -> `dialer-client.ts:416-419` -> `DialerWorkspacePage.tsx:68` `useOutcomes()` — Antwortform unverändert, nur der Zugriff wird erweitert -> passt.
- offen: Kein Re-Login nötig (`dialer:calls:write` steckt bei member schon im Token). Der Spec-Eintrag an GET /dialer/outcomes wurde nicht angefasst (Route und Statuscodes unverändert).

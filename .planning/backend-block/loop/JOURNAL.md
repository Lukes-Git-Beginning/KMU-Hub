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

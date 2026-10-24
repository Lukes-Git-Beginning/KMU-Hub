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

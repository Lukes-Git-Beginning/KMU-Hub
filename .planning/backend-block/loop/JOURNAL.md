# Backend-Nachtloop — Journal Lauf 14

Append-only. Ein Eintrag je Iteration, **ans Dateiende**, nie einsortieren. Form und Pflichtzeilen
stehen in `ITERATION.md` Schritt 6.

Frühere Läufe liegen vollständig im Archiv:
`archive/lauf-1-2/` (58 Units) · `archive/lauf-3/` (61) · `archive/lauf-4/` (54) ·
`archive/lauf-5/` (41) · `archive/lauf-6/` (46) · `archive/lauf-7/` (71) ·
`archive/lauf-8/` (94) · `archive/lauf-9/` (37) · `archive/lauf-10/` (93) ·
`archive/lauf-11/` (131 Einträge, davon 121 done) · `archive/lauf-12/` (99, davon 41 done) ·
`archive/lauf-13/` (120, davon 21 done — abgestürzt, siehe unten). Alle inkl. Laufbilanz.

---

## Laufkontext

- **Ausgangspunkt:** Lauf 13 gemergt als `572cb6b0` und deployt. `main` = `backend-loop`,
  Fast-Forward, **nicht** rebased. CI grün auf `572cb6b0` (Run 33073076446, alle fünf Jobs
  inkl. E2E, 57 Steps).
- **Migrationen:** Repo-Kopf = lokaler DB-Kopf = Produktionskopf = **325**, `schema_migrations`
  clean. Lauf 13 hat keine Migration angelegt. Nächste freie Nummer wäre 326 — aber immer zur
  Laufzeit ermitteln:
  `ls backend/migrations | grep -E "^[0-9]{6}" | sort | tail -1`
- **Coverage-Start:** **71,6 %** gesamt bei Gate 15 % (CI-Artefakt auf `572cb6b0`).
  Lauf 13 hat 69,6 → 71,6 gehoben, in 21 Iterationen.
- **Queue:** 58 Units, die dreizehn verifizierten Lecks am Kopf. Begründung im Kommentarblock
  über den Units in `BACKLOG.yml`.

## Was aus Lauf 13 gelernt wurde — drei Dinge, die dich direkt betreffen

1. **Ein Listeneintrag darf nicht mit einem Backtick beginnen.** YAML verbietet das als erstes
   Zeichen eines Plain-Skalars. Genau daran ist Lauf 13 um 04:37 gestorben, nach 21 von 130
   Iterationen: `BACKLOG.yml` wurde unparsebar, der nächste `--state` des Treibers scheiterte,
   der Lauf war vorbei. Schreib `- >-` und den Text eingerückt darunter. Seit `7d59d43e` blockt
   `hooks/loop-guard.sh` einen Commit, der den Backlog kaputt hinterlässt — du merkst es also
   sofort und reparierst es, statt die Nacht zu beenden.

2. **Ein verifizierter Tenant- oder Auth-Befund gehört an den KOPF von `BACKLOG.yml`, nicht ans
   Ende.** Lauf 13 hat zwei über REST erreichbare Cross-Tenant-Lecks gefunden und regelkonform
   angehängt — Position 44 und 45 von 58. Bei Dateireihenfolge als einziger Priorität heißt das
   „nie". Die Regel steht jetzt in `ITERATION.md`.

3. **`go test -race` läuft auf dieser Maschine nicht** (kein C-Compiler, `-race` braucht cgo).
   Dein Gate kann Data Races grundsätzlich nicht sehen. Lauf 13 hat zwei eingebaut, die lokal in
   jedem Gate grün waren und CI rot machten: ein ungeschützter `append` in einem Mock, der erst
   nebenläufig erreicht wurde, nachdem ein No-op-Stub verdrahtet war, und ein `t.Cleanup`, das
   paketweite Variablen zurückschrieb, während eine Worker-Goroutine sie noch las. Schreibst du
   einen Test, der Goroutinen startet oder gemeinsamen Zustand nebenläufig anfasst, **schreib das
   in `offen:`** — abgenommen ist er erst nach CI.

---

## Iteration 1 — fix-websocket-chat-rpcs-missing-tenant-context — done — 2026-10-20 19:38
- commit: dieser Commit, `fix(server): carry the JWT tenant into websocket chat RPCs` (SHA via `git log`)
- gebaut: `HandleWebSocket` stempelt `TenantIDKey`/`UserIDKey` aus den JWT-Claims einmal auf den Context (Variante a), bevor `cacheUserName` und `handleConnection` laufen — GetChannel/MarkChannelRead/SendMessage aus der WS-Loop tragen jetzt `x-tenant-id`/`x-user-id`. `handleSendMessage` setzt zusätzlich `TenantId` aus dem Context (chat_grpc.go `SendMessage` bleibt unverändert, liest das Feld). Docstring von `TenantInboundUnaryInterceptor` korrigiert (lehnt nie selbst ab).
- gate: build ok | vet ok | lint ok (0 issues) | test ok (internal/server 1934 PASS, 0 SKIP, 21 DB-Testfunktionen real gelaufen; internal/server/... + middleware ok; internal/gateway ok inkl. TestOpenAPIRouteDrift, 0 SKIP) | migration n.a. | rls-smoke n.a.
- coverage: internal/server 72,6 % -> 72,7 % (kein Coverage-Ziel, Bugfix)
- mutations-probe: (1) Context-Anreicherung in `HandleWebSocket` entfernt -> alle drei neuen Tests rot (Subscribe/SendMessage/MarkRead); (2) `TenantId` in `handleSendMessage` geleert -> `TestHandleSendMessage_RequestPassesChatServerGuards` rot ("failed to send message"); beides zurückgedreht, Diff sauber
- verify vorgaenger: n.a. — erste Iteration von Lauf 14, kein Loop-Commit davor (main-Commits seit 08-27 stammen von Luke)
- neue-units: keine
- offen: (1) Der authentifizierte WS-Sendepfad aus `fix-websocket-chat-send-message-missing-tenant-id` ist hiermit mit erledigt; dort bleiben nur die Gast-Pfade (`handleGuestSendMessage`, `route_guest.go`). (2) Die neuen Tests starten Server-Goroutinen (Hub über echten WS-Dial); Fakes melden nur über gepufferte Channels zurück, `channelMembers` wird unter Lock gelesen — abgenommen erst nach CI mit `-race`. (3) `cacheUserName` lief bisher ebenfalls ohne Tenant und dürfte unter RLS nie getroffen haben; läuft jetzt mit Tenant-Context (Nebeneffekt, gewollt).

## Iteration 2 — fix-websocket-chat-send-message-missing-tenant-id — done — 2026-10-20 19:50
- commit: `fix(server): carry the guest session tenant into chat sends` (SHA via `git log`)
- gebaut: `GuestSessionInfo` hat jetzt `TenantID` (Adapter in `cmd/gateway/adapters.go` füllt es aus `guest.GuestSession`); `handleGuestSendMessage` (WS) und `GuestRoutes.HandleSendMessage` (REST) setzen `SendMessageRequest.TenantId` aus der Session. Der authentifizierte WS-Pfad war schon in Iteration 1 erledigt. Kein Channel-Lookup nötig: die Guest-Session trägt die Tenant-ID bereits (Spalte `tenant_id`, aus dem Channel beim Anlegen befüllt).
- gate: build ok | vet ok | lint ok (0 issues) | test ok (internal/server/... + internal/gateway + cmd/gateway ok, DATABASE_URL als kmuhub_app gesetzt) | migration n.a. | rls-smoke n.a.
- coverage: internal/server 72,7 % -> 72,8 % (kein Coverage-Ziel, Bugfix)
- mutations-probe: `TenantId:`-Zeile in beiden Gast-Pfaden entfernt -> `TestHandleGuestSendMessage_RequestCarriesSessionTenant` (WS) und `TestGuestHandleSendMessage_CarriesSessionTenant` (REST, gegen echten loopback-gRPC-Server) beide rot; zurückgedreht, Diff sauber
- verify vorgaenger: sauber (Iteration 1, 675408f2: WS-Context gestempelt, TenantId gesetzt, keine Route/Proto/Migration/Guard berührt, Tests mit Mutations-Probe)
- neue-units: keine
- offen: (1) Der neue WS-Gast-Test und der REST-Test starten Server-Goroutinen (http.Server bzw. grpc.Server); Fake meldet nur über gepufferten Channel zurück — abgenommen erst nach CI mit `-race`. (2) Der Gast-WS-Kontext trägt weiterhin keinen Tenant in der Message-Loop (nur das Request-Feld); reicht für `SendMessage`, würde aber bei einem künftigen Gast-RPC, der `GetTenantID(ctx)` liest, wieder fail-closed ausfallen.

## Iteration 3 — fix-createevent-never-sets-tenant-id — done — 2026-10-20 19:47
- commit: `fix(calendar): scope event creation to the caller's tenant` (SHA via `git log`)
- gebaut: `CalendarGRPCServer.CreateEvent` liest den Tenant per `middleware.GetTenantID(ctx)` (fehlt er: `codes.InvalidArgument "missing tenant context"`, wie GetEvent) und setzt `event.CreateInput.TenantID`. Kalender-Lookup und `calendar_events.tenant_id` laufen damit auf dem echten Tenant statt der Null-UUID. `TestCreateEvent` umgedreht: fehlender Tenant -> InvalidArgument, fremder Tenant -> NotFound, eigener Tenant -> Event angelegt, Event trägt den Tenant.
- klaerung (1): `CreateCalendar` (calendar_grpc.go:72-73) liest `GetTenantID` und schreibt den echten Tenant; lokale DB hat 0 Kalender mit Null-UUID. Das Paar passte also nie zusammen — CreateEvent konnte für keinen echten Tenant gelingen.
- klaerung (2): `calendars` und `calendar_events` haben RLS aktiv + FORCE, Policy `tenant_isolation`: `tenant_id = current_tenant_id() OR is_system_context()` (pg_policies, lokale DB). Der Repo-Filter `tenant_id=$2` und RLS tragen beide.
- uebrige RPCs ohne `GetTenantID` in calendar_grpc.go: `ListEventReminders`, `GetCalendarPreferences`, `UpdateCalendarPreferences`, `ListTaskDeadlinesInRange` — beabsichtigt: Tabellen `event_reminders`/`user_calendar_preferences`/`tasks` haben RLS+FORCE, der Pool stempelt den ctx-Tenant, INSERTs leiten `tenant_id` aus Eltern-Zeile ab (event_reminders <- calendar_events, prefs <- users). `ListHolidays`/`SeedHolidays` — beabsichtigt, `public_holidays` ist system-global (kein tenant_id). `GetPublicBookingPage`/`GetAvailability`/`CreatePublicBooking` — beabsichtigt, öffentlich, Tenant kommt aus der Slug-Auflösung der Buchungsseite (`grpcCalClientAdapter` setzt `in.TenantID`). Kein weiterer Befund in dieser Datei.
- gate: build ok (internal/server, internal/gateway, ./cmd/...) | vet ok | lint ok (0 issues) | test ok (internal/server 4605 RUN-Zeilen, 0 SKIP; internal/server/..., internal/work/event, internal/caldav ok; internal/gateway ok inkl. TestOpenAPIRouteDrift, 0 SKIP; DATABASE_URL als kmuhub_app) | migration n.a. | rls-smoke n.a. (keine Tabelle/Policy angefasst)
- coverage: internal/server 72,8 % -> 72,8 % (kein Coverage-Ziel, Bugfix)
- mutations-probe: (1) `TenantID: tenantID` aus CreateInput entfernt -> `TestCreateEvent/happy_path_in_the_caller's_tenant` rot (NotFound); (2) Tenant-Check zu `tenantID, _ :=` entschärft -> `TestCreateEvent/missing_tenant_context` rot; beides zurückgedreht, Diff sauber
- verify vorgaenger: sauber (Iteration 2, 64f9ab93: TenantID aus der Guest-Session in WS- und REST-Gastpfad, Adapter füllt das Feld, keine Route/Proto/Migration/Guard)
- neue-units: `fix-automation-grpc-actions-missing-tenant-context` (ans Ende — Funktionsausfall, kein Leck)
- offen: (1) Aufrufer von CreateEvent: Gateway (Auth-ctx), CalDAV (stempelt Tenant, caldav_backend.go:62-66) — beide tragen den Tenant. Automation (`action/calendar_actions.go:70`) NICHT: `cmd/automation` wählt ohne Outbound-Interceptor und führt mit `context.Background()` aus; dort scheiterte CreateEvent vorher mit NotFound, jetzt mit InvalidArgument — kein Rückschritt, aber weiter kaputt, siehe neue Unit. (2) `calendar_grpc.go` hat eine vorbestehende gofmt-Abweichung in `bookingPageToProto` (~Z. 1980), nicht angefasst.

## Iteration 4 — fix-caldav-test-revoke-runs-without-tenant-context — done — 2026-10-20 19:56
- commit: `fix(gateway): revoke the caldav test password with the caller's tenant` (SHA via `git log`)
- gebaut: `handleTestConnection` widerruft das Ephemeral-Passwort jetzt mit `context.WithTimeout(context.WithoutCancel(r.Context()), 5s)` statt `context.Background()` — von der Request-Absage entkoppelt, Tenant-/User-Werte bleiben. Neuer DB-Test `TestHandleTestConnection_RevokesEphemeralPasswordUnderRLS` (Paket `caldav`, weil `gateway`-Tests `caldav` nicht importieren können: Zyklus) fährt den echten `AppPasswordService` als `kmuhub_app` durch die echte Route; danach `revoked_at IS NOT NULL` und `FindActiveByUser` leer.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (internal/caldav 161 RUN, 0 SKIP; internal/gateway ok inkl. TestOpenAPIRouteDrift) | migration n.a. | rls-smoke n.a. (keine Tabelle/Policy angefasst)
- coverage: internal/caldav 68,6 % -> 68,6 % (kein Coverage-Ziel, Bugfix; geändert ist gateway-Code, der Test deckt keine neuen caldav-Statements)
- mutations-probe: `WithoutCancel(r.Context())` -> `context.Background()` -> der neue Test rot ("ephemeral password must be revoked", aktives Passwort bleibt); zurückgedreht, Diff sauber
- verify vorgaenger: sauber (Iteration 3, 556287da: nur `calendar_grpc.go` + Test, keine Route/Proto/Migration/Guard)
- neue-units: keine
- offen: (1) Alt-Zeilen: lokale DB hat 0 `app_specific_passwords` insgesamt (0 dangling `connection-test`) — Produktion muss Luke selbst prüfen: `SELECT count(*) FROM app_specific_passwords WHERE label='connection-test' AND revoked_at IS NULL` (bei Fund per UPDATE `revoked_at = NOW()` widerrufen). (2) Der neue Test startet einen httptest-Server (Goroutinen), kein geteilter Schreibzustand — trotzdem erst nach CI mit `-race` abgenommen. (3) Der Test flippt die globale `caldav_settings`-Org-Flag (mit Restore via `withOrgEnabledRestored`) und läuft deshalb nicht parallel, wie die Nachbartests.

## Iteration 5 — fix-getpresence-rpc-missing-tenant-check — done — 2026-10-20 20:20
- commit: `fix(presence): drop presence targets outside the caller's tenant` (SHA via `git log`)
- gebaut: Grenze im Service (`presence.Service.visibleTargets`), nicht im Handler: fehlender Tenant -> `ErrMissingTenant` (Unauthenticated), eigener User (`middleware.GetUserID`) immer erlaubt, alle übrigen nur, wenn der neue `presence.UserDirectory` sie im Aufrufer-Tenant findet (`PostgresUserDirectory`: `SELECT id FROM users WHERE tenant_id = $1 AND id = ANY($2)`, Tenant explizit wie beim ConfigRepository). `GetPresence` auf fremden/unbekannten User -> neues `ErrUserNotVisible` -> NotFound; `GetBulkPresence` entfernt fremde IDs vor dem Redis-Zugriff (auch kein Offline-Eintrag). Verdrahtung in `cmd/work/main.go` über den vorhandenen Pool — keine neue Abhängigkeit, kein Auth-Client. `openapi.yaml`: 404 an `getPresence` dokumentiert.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (internal/work/presence 48 RUN, 0 SKIP, 5 DB-Tests real gelaufen inkl. neuem `TestPostgresUserDirectory_OnlyCallerTenantVisible`; internal/server/... 0 SKIP; internal/gateway ok inkl. TestOpenAPIRouteDrift; swagger-cli validate ok; DATABASE_URL als kmuhub_app) | migration n.a. | rls-smoke n.a. (keine Tabelle/Policy angefasst; der DB-Test prüft den users-Filter unter Tenant- und System-Context)
- coverage: internal/work/presence 81,2 % -> 83,8 % (kein Coverage-Ziel, Sicherheits-Fix)
- mutations-probe: (1) Filter in `visibleTargets` übersprungen -> 4 presence- + 2 server-Tests rot (Foreign/Bulk/Invalid/OwnUserWithoutDirectory); (2) Tenant-Check fail-open -> 3 MissingTenant-Tests rot; (3) `tenant_id = $1` aus dem Directory-SQL entfernt -> DB-Test rot im System-Context-Fall ("got 2 entries"; im Tenant-Context hält RLS als zweites Netz); alles per cp-Sicherung zurückgedreht, cmp sauber
- verify vorgaenger: sauber (Iteration 4, dee9d832: nur `route_caldav.go` + caldav-DB-Test, keine Route/Proto/Migration/Guard)
- neue-units: keine
- offen: (1) Verhaltensänderung: `GET /api/v1/video/presence/{userId}` liefert für eine UUID, die im eigenen Tenant kein User ist, jetzt 404 statt "offline" — das FE fragt nur Tenant-Kollegen ab, aber der Presence-Hook im Desktop sollte 404 nicht als Fehler-Toast zeigen (nicht geprüft, FE gesperrt). (2) Jede Presence-Abfrage kostet jetzt eine indexierte users-Query (PK `id`); bei Bulk eine einzige `ANY`-Query. (3) Kein Goroutinen-Test hinzugefügt, `-race` nicht relevant.

## Iteration 6 — fix-dialer-agent-status-missing-tenant-check — done — 2026-10-20 20:05
- commit: `fix(dialer): hide agent status entries outside the caller's tenant` (SHA via `git log`)
- gebaut: `AgentStatusEntry` trägt jetzt den mitgespeicherten Tenant (`parseAgentStatusData` parst `tenant_id`, nicht parsebar -> `uuid.Nil`, damit `SetStatus` seine Transition weiter prüfen kann). `dialer.Service.GetAgentStatus`/`GetCampaignAgents` nehmen `tenantID` und filtern über `visibleAgentEntry` (fremd oder ohne Tenant = nicht vorhanden, fail-closed); der Kampagnen-Filter läuft pro Einzeleintrag, Ergebnis ist eine leere nicht-nil Liste. Auch `GetSupervisorOverview` filtert (fremder Eintrag -> Zeile offline). gRPC `GetAgentStatus`/`GetCampaignAgents` lesen `middleware.GetTenantID` (fehlt -> Unauthenticated) und liefern für unbekannte UND fremde UUID identisch `{user_id, status: OFFLINE}` ohne since/campaign.
- klaerung: Vorher führte eine unbekannte UUID zu `agentStatusToProto(nil)` -> nil-Deref -> Panik, von `RecoveryUnaryInterceptor` als Internal/500 aufgefangen. Auch der eigene Status eines Agenten ohne Redis-Eintrag (`GET /dialer/agent-status` ohne `user_id`) lief so in 500. Jetzt 200 + offline — derselbe Zustand, den der Store einem fehlenden Schlüssel ohnehin zuschreibt.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (internal/dialer 165 RUN, 0 SKIP, DB-Tests in rls_test/tenant_write_test/queue_and_list_test real gelaufen; internal/server 4612 RUN, 0 SKIP; internal/server/..., internal/dialer/..., internal/gateway ok inkl. TestOpenAPIRouteDrift; DATABASE_URL als kmuhub_app) | migration n.a. | rls-smoke n.a. (Redis, keine Tabelle/Policy)
- coverage: internal/dialer 77,5 % -> 77,8 %
- mutations-probe: (1) Tenant-Vergleich in `visibleAgentEntry` entfernt -> 4 dialer- + 2 server-Tests rot; (2) Tenant-Parse in `parseAgentStatusData` entfernt -> 4 dialer- + 3 server-Tests rot; (3) Pro-Eintrag-Filter in `GetCampaignAgents` entfernt -> beide Zwei-Tenant-Tests rot; (4) Filter in `GetSupervisorOverview` entfernt -> `TestGetSupervisorOverview_ForeignTenantEntryReadsOffline` rot; (5) gRPC-Tenant-Check fail-open -> `TestGetAgentStatus_MissingTenant` rot; alles per cp-Sicherung zurückgedreht, cmp sauber
- verify vorgaenger: sauber (Iteration 5, 01c79ed7: Grenze im presence-Service, einzige Caller sind die zwei gRPC-RPCs, Spec-404 dokumentiert, kein Proto/Guard/Migration)
- neue-units: `fix-dialer-set-agent-status-foreign-user-overwrite` (an den KOPF gesetzt — Cross-Tenant-Write per Wegwerf-Test reproduziert: Tenant A überschreibt den Live-Status eines Agenten aus Tenant B, B sieht ihn danach offline) · `fix-flaky-gdpr-export-rpc-test` (ans Ende — einmaliger roter Lauf, isoliert 45/45 grün)
- offen: (1) Deploy-Effekt wie in der Unit vorhergesagt: laufende Redis-Einträge MIT `tenant_id` bleiben sichtbar; Einträge ohne verwertbaren Tenant sind ab Deploy unsichtbar, bis der Agent seinen Status neu setzt (TTL `agentStatusTTL` = 8 h). Das Feld wird seit dem ersten SetStatus-Schreibpfad mitgeschrieben, Altbestand ist also unwahrscheinlich. (2) Verhaltensänderung: `GET /dialer/agent-status` liefert für Agenten ohne Status 200/offline statt 500 — FE-Typ `AgentStatusEntry` deckt das (status als Zahl, since optional?) nicht geprüft, FE gesperrt. (3) `TestGDPRExportRPCs_HappyPathAndDomainErrors` fiel einmal in einem Mutationslauf rot (unabhängig von dieser Änderung), siehe neue Unit. (4) Keine Goroutinen in den neuen Tests (miniredis läuft im Hintergrund, aber ohne geteilten Testzustand).

## Iteration 7 — fix-dialer-set-agent-status-foreign-user-overwrite — done — 2026-10-20 20:14
- commit: `fix(dialer): refuse agent status writes outside the caller's tenant` (SHA via `git log`)
- gebaut: `dialer.Service.SetAgentStatus` prüft vor jedem Redis-Schreibzugriff, dass der Ziel-User im Aufrufer-Tenant liegt (neue Repo-Methode `AgentStatusRepository.UserInTenant`, `SELECT EXISTS … WHERE id = $1 AND tenant_id = $2`, Tenant explizit wie `presence.PostgresUserDirectory`); fremd und unbekannt -> neues `ErrAgentNotFound` -> NotFound, identische Meldung. Eine übergebene `campaign_id` wird über `campaigns.GetByIDForTenant` geladen, fremd -> `ErrCampaignNotFound` -> NotFound, bevor das SET angefasst wird. Lookup-Fehler -> nichts geschrieben (fail-closed). `openapi.yaml`: 404 an `setDialerAgentStatus` dokumentiert.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (internal/dialer 169 RUN, 0 SKIP, neuer DB-Test `TestAgentStatusRepository_UserInTenant` real gelaufen; internal/server 4614 RUN, 0 SKIP, 0 FAIL; internal/server/... ok; internal/gateway ok inkl. TestOpenAPIRouteDrift, 0 SKIP; swagger-cli validate ok; DATABASE_URL als kmuhub_app; Gate auf finalem Tree wiederholt) | migration n.a. | rls-smoke n.a. (keine Tabelle/Policy angefasst)
- coverage: internal/dialer 77,8 % -> 78,0 % (Vorher-Wert im Wegwerf-Worktree auf HEAD gemessen)
- mutations-probe: (1) `if !inTenant`-Zweig entfernt -> `TestSetAgentStatus_ForeignUserRejectedLikeUnknown` (dialer) + `TestSetAgentStatus_ForeignUserMatchesUnknown` (server) rot; (2) Kampagnen-Check auf `if false` -> `TestSetAgentStatus_ForeignCampaignRejected` + `TestSetAgentStatus_ForeignCampaignNotFound` rot; (3) `tenant_id = $2` im SQL entfernt -> DB-Test rot im System-Context-Fall (im Tenant-Context hält RLS als zweites Netz); (4) `ErrAgentNotFound`-Case aus `mapDialerError` entfernt -> server-Test rot (Internal statt NotFound); alles per cp-Sicherung zurückgedreht, cmp sauber
- verify vorgaenger: sauber (Iteration 6, 3392d7d0: Lesepfade gefiltert, Signaturänderung in allen Callern nachgezogen, keine Route/Proto/Migration/Guard)
- neue-units: `fix-dialer-agent-invalid-transition-maps-to-internal` (ans Ende — `ErrInvalidTransition` des Agent-Stores fehlt in `mapDialerError`, unzulässiger Statuswechsel liefert 500 statt FailedPrecondition; kein Leck)
- offen: (1) Test-Fakes `mockCampaignRepo`/`stubCampaignRepo.GetByIDForTenant` prüfen jetzt den Tenant, wenn die gesäte Kampagne einen trägt (ohne Tenant: passt zu jedem Aufrufer, alte Tests unverändert); `mockAgentStatusRepo`/`stubAgentStatusRepo.UserInTenant` sind ohne gesetzte `userTenants`-Map permissiv — nur die neuen Grenztests pinnen User an Tenants. (2) Die zwei Read-Filter-Tests mit gemischtem Kampagnen-SET säen jetzt am Store vorbei bzw. mit tenantloser Kampagne, weil der RPC das gemischte SET nicht mehr erzeugen kann. (3) Verhaltensänderung: `PUT /dialer/agent-status` liefert 404 für einen `user_id` außerhalb des Tenants (vorher 200) und für eine fremde/unbekannte `campaign_id` (vorher 200 + Eintrag ins fremde SET). Der eigene Default-Pfad (kein `user_id` -> JWT-User) trifft immer den eigenen Tenant. (4) Jeder Statuswechsel kostet eine PK-Query auf `users`, mit Kampagne eine zweite. (5) Keine Goroutinen in neuen Tests, `-race` nicht relevant.

## Iteration 8 — fix-rapporte-worker-and-measurement-parent-report-tenant-check — done — 2026-10-20 20:24
- commit: `fix(rapporte): reject workers and measurements attached to another tenant's report` (SHA via `git log`)
- gebaut: `AddWorker` und `CreateMeasurement` (nur wenn `report_id` gesetzt) fügen jetzt per `INSERT … SELECT … WHERE EXISTS (work_reports mit id, tenant_id, deleted_at IS NULL)` ein, 0 Zeilen -> `ErrReportNotFound` (bereits auf NotFound gemappt). Guard im Repository, nicht im Handler; NIL-`report_id` bleibt erlaubt. Nebenbefund `UploadAttachment`: eine übergebene `line_id` wird im Service über `GetLine` (tenant-gescopt) geladen und muss zum selben Report gehören, sonst `ErrLineNotFound`.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (internal/rapporte/... 0 SKIP, neue DB-Tests `TestAddWorker_RejectsReportIDFromAnotherTenant` + `TestCreateMeasurement_ReportIDMustBelongToCallerTenant` real gelaufen; internal/server ok; internal/gateway ok inkl. TestOpenAPIRouteDrift; DATABASE_URL als kmuhub_app) | migration n.a. | rls-smoke n.a. (keine Tabelle/Policy angefasst)
- coverage: internal/rapporte 76,9 % -> 77,3 % (kein Coverage-Ziel, Sicherheits-Fix; Vorher-Wert per stash auf HEAD gemessen)
- mutations-probe: (1) AddWorker-Guard komplett entfernt (zurück auf VALUES) -> `TestAddWorker_RejectsReportIDFromAnotherTenant` rot; (2) CreateMeasurement-Guard entfernt -> `TestCreateMeasurement_ReportIDMustBelongToCallerTenant` rot; (3) Line-Report-Vergleich im Service deaktiviert -> `TestService_UploadAttachment_LineOfAnotherReport_Rejected` rot; (4) nur der `tenant_id`-Filter im EXISTS entfernt -> Tests bleiben grün, weil die RLS-Policy der Elterntabelle als zweites Netz hält (wie in der Unit vorhergesagt); alles per cp-Sicherung zurückgedreht, cmp sauber
- verify vorgaenger: sauber (Iteration 7, 2fdf18cd: Tenant-Check vor jedem Redis-Write, `ErrAgentNotFound` gemappt, 404 in der Spec, kein Proto/Guard/Migration)
- neue-units: keine
- offen: (1) Verhaltensänderung: `AddWorker`/`CreateMeasurement` mit fremder oder unbekannter `report_id` liefern jetzt 404 statt 500 (FK-Fehler bei unbekannter ID) bzw. stilles Anhängen an den fremden Report; ein bereits soft-gelöschter eigener Report wird ebenfalls abgelehnt. (2) Die REST-Route für `UploadAttachment` mit fremder/fehlender `line_id` liefert jetzt 404 (`ErrLineNotFound`); die Spec dokumentiert 404 dort ggf. nicht — nicht geprüft. (3) Keine Goroutinen in neuen Tests, `-race` nicht relevant.

## Iteration 9 — fix-chat-search-channel-filter-bypasses-membership — done — 2026-10-20 20:41
- commit: `fix(chat): ...` (SHA via `git log`)
- gebaut: `search.Service.Search` holt jetzt immer `GetUserChannelIDs`; ein explizites `channel_id` engt nur auf diese Liste ein (`narrowToChannel`). Kein Treffer in der Liste -> leere Antwort ohne Repo-Suche (`0`, kein Fehler), damit nicht verraten wird, ob der Channel existiert. Der BUG-Test ist zum Regressionstest umgebaut; neuer Test belegt den Mitglieds-Fall inkl. exaktem Suchscope.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (internal/chat/... und internal/server ok mit -p 2, internal/chat/search 0 SKIP; internal/gateway ok inkl. TestOpenAPIRouteDrift; DATABASE_URL als kmuhub_app) | migration n.a. | rls-smoke n.a. (keine Tabelle/Policy angefasst)
- coverage: internal/chat/search 86,7 % -> 87,0 % (Vorher-Wert per stash auf HEAD gemessen)
- mutations-probe: (1) `narrowToChannel(...)` durch `[]uuid.UUID{*input.ChannelID}` ersetzt (alter Bug) -> `explicit_channel_id_outside_the_caller's_memberships_returns_nothing` rot; (2) `narrowToChannel` gibt alle Mitgliedschaften statt nur den Channel zurück -> `explicit_channel_id_of_a_member_channel_searches_only_that_channel` rot; beides per cp-Sicherung zurückgedreht, cmp sauber
- verify vorgaenger: sauber (Iteration 8, ec983629: EXISTS-Guard im Repository, ErrReportNotFound gemappt, 404 in der Spec für den Worker-/Measurement-Pfad, kein Proto/Guard/Migration; offener Punkt aus Iteration 8: Spec-404 an UploadAttachment ungeprüft)
- neue-units: keine
- offen: (1) Verhaltensänderung: `GET /api/v1/chat/search?channel_id=<fremder Channel>` liefert jetzt 200 mit leerer Liste statt Treffer. (2) Beim ersten Lauf von `go test ./internal/chat/... ./internal/gateway/` ohne `-p 2` ging der lokalen Postgres die Verbindungsslots aus ("too many clients") und `chat/message` fiel rot — mit `-p 2` grün; reines Lokal-Phänomen, kein Codebefund. (3) Jede Suche mit `channel_id` kostet jetzt zusätzlich eine `channel_memberships`-Abfrage (indexiert über user_id).

## Iteration 10 — harden-livekit-room-name-truncation-collides-across-tenants — done — 2026-10-20 20:52
- commit: `fix(calendar): derive livekit room names from tenant and event id` (SHA via `git log`)
- gebaut: Neue `livekit.RoomName(tenantID, eventID)` = `"cal-" + hex(SHA-256(tenant ":" event)[:16])` (128 Bit statt 32 Bit, kein Roh-ID-Fragment). `Service.GenerateRoomName` nimmt jetzt `tenantID` und delegiert; `CalendarGRPCServer.GenerateJoinToken` übergibt den bereits geprüften Tenant. Die zwei weiteren, bisher handgebauten `fmt.Sprintf("cal-%s", id[:8])`-Stellen in `event.Service` (Create und Backfill bei `HasVideoCall` am Update) nutzen dieselbe Funktion — vorher gab es drei Kopien des Formats.
- fuehrender Weg: **Neuberechnung** (Tenant+Event) ist führend, nicht die Spalte. Grund: Token-Pfad rechnete schon bisher neu und las die Spalte nie; die Spalte ist rein informativ (proto/CalDAV-Anzeige). Neue Events und der Update-Backfill schreiben jetzt exakt denselben Namen in die Spalte, beide Wege laufen also für alles Neue zusammen.
- bestandszeilen: Altzeilen behalten ihren gespeicherten `cal-xxxxxxxx` (keine Migration, bewusst — informativ, und Migrationen an `calendar_events` sind hier nicht nötig). Der Token-Pfad liefert für sie ab Deploy den neuen Namen; gespeicherter und tatsächlicher Raum weichen bei Altzeilen also ab. Prod ist Single-Tenant, die Kollision bleibt dort ohnehin hypothetisch.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (internal/work/..., internal/caldav/..., internal/server, internal/gateway ok inkl. TestOpenAPIRouteDrift; DATABASE_URL als kmuhub_app) | migration n.a. | rls-smoke n.a.
- coverage: internal/work/livekit 38,4 % -> 38,9 % (Haertung, kein Coverage-Ziel; Vorher-Wert per stash auf HEAD)
- mutations-probe: (1) Tenant aus dem Hash entfernt -> `TestRoomName_SameEventPrefixDifferentTenants` rot; (2) `uuid.Nil` statt Tenant im Event-Service (Create + Backfill) -> `TestService_Create_WithVideoCall` + `TestService_UpdateEvent_EnablingVideoCallDerivesTenantRoomName` rot; (3) `uuid.Nil` in `GenerateJoinToken` -> `TestGenerateJoinToken` rot; alles per Sicherung zurückgedreht. Ein erster Versuch für (2) brach am Build (unused import) ab und zählt nicht.
- verify vorgaenger: sauber (Iteration 9, c93d1939: nur chat/search-Service, kein Proto/Route/Guard/Migration)
- neue-units: keine
- offen: (1) Verhaltensänderung: bereits laufende Meetings zu bestehenden Events — wer nach dem Deploy neu beitritt, bekommt einen anderen Raum als schon verbundene Teilnehmer, bis das Meeting endet (Token 24 h). (2) Altzeilen: gespeicherter Name ≠ Token-Raum (siehe oben); wer das bereinigen will, braucht eine Datenmigration mit demselben SHA-256 (`sha256(tenant_id::text || ':' || id::text)`, erste 32 Hex-Zeichen) — nicht gebaut. (3) Keine Goroutinen in neuen Tests.

## Iteration 11 — fix-caldav-write-and-exceptions-blocked-by-missing-tenant-ctx — done — 2026-10-20 21:05
- commit: `test(caldav): prove writes and exceptions work under the real basic-auth context` (SHA via `git log`)
- gebaut: Keine Code-Änderung nötig — der Root Cause (`NewCtxInjector`, Lauf 13) wirkt: unter dem echten Kontext darf der Besitzer schreiben und `listEventExceptions` liefert die Zeilen. Die zwei `_RealCalDAVContext_`-Tests, die den alten Fehler dokumentierten, bauen ihren Kontext jetzt über den Produktions-Injector und sind umgedreht (`_OwnerAllowed`, `_ReturnsExceptions`). Zwei neue Grenztests (`_OtherTenantSeesNoCalendar` -> 404, `_OtherTenantSeesNothing` -> leer) belegen, dass die Tenant-Grenze steht. Kopfkommentar der Datei nachgezogen (kein "CONFIRMED PRODUCTION BUG" mehr); gemeinsame Helfer `realCalDAVCtx`, `seedForeignTenantUser`, `seedEventWithException`.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (internal/caldav/... 0 SKIP, DATABASE_URL als kmuhub_app, -p 2) | migration n.a. | rls-smoke n.a. (keine Tabelle/Policy angefasst; die DB-Tests laufen unter RLS als kmuhub_app) — gateway-Test nicht nötig, keine Route berührt
- coverage: internal/caldav 68,6 % -> 68,6 % (Bugfix-/Beweis-Unit, kein Coverage-Ziel; Vorher-Wert per stash auf HEAD)
- mutations-probe: (1) Tenant-Stempel in `NewCtxInjector` entfernt -> `_OwnerAllowed` + `_ReturnsExceptions` rot; (2) `sysctx.With(ctx)` vor beide Direkt-Pool-Abfragen gesetzt (pauschaler Bypass) -> `_OtherTenantSeesNoCalendar` + `_OtherTenantSeesNothing` rot; beides per cp-Sicherung zurückgedreht, `git status` zeigt nur Test + Backlog
- verify vorgaenger: sauber (Iteration 10, be134921: `RoomName` einzige Quelle, alle drei Aufrufer nachgezogen, kein weiteres `"cal-"`-Literal im Produktivcode, kein Proto/Route/Guard/Migration)
- neue-units: keine
- offen: (1) `checkCalendarWritePermission`/`listEventExceptions` filtern nicht selbst nach Tenant, sondern verlassen sich auf RLS plus den vom Injector gestempelten Pool-Tenant — das ist das gleiche Muster wie im übrigen Paket, hier bewusst nicht geändert. (2) `gofmt -l` listet im Paket jede Datei (CRLF im Arbeitsbaum), kein Befund dieser Unit.

## Iteration 12 — fix-login-inactive-account-is-an-enumeration-oracle — done — 2026-10-20 21:20
- commit: `fix(auth): answer a deactivated login exactly like a wrong password` (SHA via `git log`)
- gebaut: `Service.Login` vergleicht jetzt das Passwort VOR dem Aktiv-Check: ein deaktiviertes Konto kostet dieselbe bcrypt-Runde wie ein falsches Passwort, und ein falsches Passwort darauf liefert `ErrInvalidCredentials`. `ErrUserInactive` kommt nur noch bei RICHTIGEM Passwort (Log-Grund `user_inactive` bleibt). Der gRPC-Handler `AuthGRPCServer.Login` (nicht `mapError` global) faltet `ErrUserInactive` in `ErrInvalidCredentials` -> `Unauthenticated`/401 mit identischer Meldung.
- frontend-pruefung: `loginErrorKey` (`stores/auth.ts:147`) mappt 403 nur auf die Meldung `auth.accountDisabled`; kein Kontrollfluss haengt daran. Die Meldung erscheint künftig nie mehr aus dem Login (Nutzer sieht „ungültige Zugangsdaten“) — FE nicht angefasst.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (internal/auth, internal/server, internal/gateway inkl. TestOpenAPIRouteDrift; Login-/MapError-Tests 0 SKIP; DATABASE_URL als kmuhub_app) | migration n.a. | rls-smoke n.a.
- coverage: internal/auth 77,3 % -> 77,3 % (Bugfix-Unit, kein Coverage-Ziel)
- mutations-probe: (1) Faltung im gRPC-Handler entfernt -> `TestAuthGRPC_Login` rot; (2) Aktiv-Check zurück vor den bcrypt-Vergleich -> `TestLogin_FailurePaths_ShareOneAnswer` + `TestService_Login/inactive_user,_wrong_password` rot; beides per cp-Sicherung zurückgedreht
- verify vorgaenger: sauber (Iteration 11, 0c674a45: nur Tests/Kommentar in internal/caldav, kein Proto/Route/Guard/Migration)
- neue-units: keine
- offen: (1) Verhaltensänderung: `POST /auth/login` für ein deaktiviertes Konto liefert 401 statt 403. (2) Rest-Orakel, bewusst nicht angefasst: ein UNBEKANNTER Nutzer überspringt bcrypt komplett (Zeitunterschied zu bekanntem Nutzer, unabhängig vom Aktiv-Status); `Refresh` und `CompleteTwoFactorLogin` geben weiter 403 für Inaktive, hängen aber an gültigen Tokens/Pending-Token. (3) Der tote i18n-Key `auth.accountDisabled` und der 403-Test in `auth-login-errors.test.ts` bleiben stehen. (4) Keine Goroutinen in neuen Tests.

## Iteration 13 — harden-password-reset-invalidate-previous-tokens — done — 2026-10-20 21:35
- commit: `fix(auth): invalidate open password reset tokens on a new request` (SHA via `git log`)
- gebaut: Neue Repository-Methode `InvalidateOpenPasswordResetTokens(ctx, tenantID, userID)` (`UPDATE ... SET used_at = NOW() WHERE tenant_id AND user_id AND used_at IS NULL`, Tenant explizit in der WHERE-Klausel, weil der Pfad unter `sysctx` läuft). `RequestPasswordReset` ruft sie vor dem Anlegen des neuen Tokens; schlägt sie fehl, wird nichts angelegt und keine Mail verschickt (Antwort bleibt `nil`, kein Enumerations-Leck). Der Pinning-Test ist umgedreht (`..._ASecondRequestKillsTheFirstToken`: älterer Link -> `ErrResetTokenUsed`, neuester funktioniert). Neuer DB-Test gegen echte RLS-Rolle.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (internal/auth/... 0 SKIP, internal/server, internal/gateway inkl. TestOpenAPIRouteDrift; DATABASE_URL als kmuhub_app, -p 2) | migration n.a. | rls-smoke n.a. (keine Tabelle/Policy angefasst; der DB-Test läuft als kmuhub_app)
- coverage: internal/auth 77,3 % -> 77,5 % (Härtung, kein Coverage-Ziel; Vorher-Wert aus Iteration 12)
- mutations-probe: (1) Service-Aufruf entfernt -> `TestRequestPasswordReset_ASecondRequestKillsTheFirstToken` rot; (2) Tenant-Filter im SQL entfernt -> `TestInvalidateOpenPasswordResetTokens_...` rot; (3) User-Filter entfernt -> derselbe DB-Test rot; alles per cp-Sicherung zurückgedreht, Diff auf service.go/postgres_repository.go nur die beabsichtigten +8/+10
- verify vorgaenger: sauber (Iteration 12, f168151f: bcrypt-Vergleich vor Aktiv-Check, Faltung nur im gRPC-Login-Handler, kein Proto/Route/Guard/Migration)
- neue-units: keine
- offen: (1) Verhaltensänderung: ein zweiter Reset-Antrag macht den Link der ersten Mail ungültig (`ErrResetTokenUsed`) — wer beide Mails parallel offen hat, kann nur noch die neuere nutzen. (2) Beim ersten Gate-Lauf fiel `internal/server` einmal rot (mit `-p 2` kombiniert mit dem Paralleltest davor), der Wiederholungslauf war grün — nicht reproduzierbar, vermutlich die lokale Postgres-Verbindungsgrenze wie in Iteration 9. (3) Keine Goroutinen in neuen Tests.

## Iteration 14 — fix-gobd-archive-hash-never-verified — done — 2026-10-20 20:51
- commit: `fix(gobdarchive): verify archived document bytes against their recorded sha256` (SHA via `git log`)
- gebaut: `gobdarchive.Service.VerifyIntegrity(ctx, tenantID, id, checkedBy)` lädt den Beleg tenant-gescoped (`repo.GetByID`), streamt die Bytes über `store.Download`, rechnet SHA-256 neu und vergleicht mit `gobd_documents.sha256`. Jede abgeschlossene Prüfung schreibt ein `integrity_check`-Event mit `metadata = {expected_sha256, actual_sha256, match}`; Abweichung -> neuer Sentinel `ErrIntegrityMismatch` + `slog.Error`. Download-/Lesefehler -> Fehler ohne Event und ohne Mismatch (unlesbar ≠ manipuliert). Schlägt das Event fehl, ist das ein Fehler (ungeloggte Prüfung beweist nichts); ein Mismatch bleibt dabei per `errors.Is` erkennbar (`errors.Join`). Keine Route, kein Proto, keine Migration, kein Flag.
- gate: build ok (gobdarchive, server, cmd/biz, gateway, cmd/gateway, -p 2) | vet ok | lint ok (0 issues) | test ok (internal/biz/gobdarchive 48 PASS / 0 SKIP inkl. neuem DB-Test als kmuhub_app; internal/server grün, siehe offen (2)) | migration n.a. | rls-smoke n.a. (keine Tabelle/Policy angefasst; der neue DB-Test belegt den Fremd-Tenant als kmuhub_app)
- coverage: internal/biz/gobdarchive 89,9 % -> 90,1 % (Bugfix-Unit, kein Coverage-Ziel; Vorher-Wert per stash auf HEAD)
- mutations-probe: (1) Vergleich durch `Match = true` ersetzt -> `_TamperedBytes_…`, `_EventFailureOnMismatch_…`, `_DB_…` rot; (2) Event-Schreiben entfernt -> alle fünf Event-Tests inkl. DB-Test rot; (3) Event-Fehler verschluckt -> `_EventFailureOnMatch_IsAnError` + `_EventFailureOnMismatch_…` rot; (4) NotFound aus `GetByID` ignoriert -> `_ForeignTenant_…` + `_DB_…` rot; jeweils per cp-Sicherung zurückgedreht, Diff danach nur die beabsichtigten +64 Zeilen
- verify vorgaenger: sauber (Iteration 13, 5a82d317: Repo-Methode mit explizitem Tenant-Filter, Mock in server/testhelpers nachgezogen, kein Proto/Route/Guard/Migration)
- neue-units: `fix-ws-connect-test-reads-tenant-before-it-is-registered` (Backlog-Ende; kein Leck, Test-Flake)
- offen: (1) **Entscheidung für Luke — der Aufrufer fehlt bewusst:** `VerifyIntegrity` hat heute keinen Aufrufer. Optionen: periodischer Worker mit Advisory-Lock (Muster `gdpr/retention_scheduler.go`, Vollscan des Buckets = Download jedes Belegs, bis 50 MiB je Stück), RPC/Route auf Zuruf je Beleg, oder Prüfung beim GoBD-Export. `checkedBy` ist `created_by NOT NULL` ohne FK — ein Worker bräuchte eine feste System-UUID. (2) `internal/server` fiel im ersten Gate-Lauf an `TestHandleWebSocket_ValidToken_RegistersConnectionAndTenant` (leerer Tenant) — auf HEAD ohne diese Änderung ebenfalls rot reproduziert, Ursache benannt (Register-Reihenfolge websocket.go:376/377), drei Wiederholungsläufe mit dieser Änderung grün; als Unit angelegt. (3) `go test ./internal/gateway/` nicht gelaufen — keine Route berührt. (4) Keine Goroutinen in neuen Tests.

## Iteration 15 — fix-fuhrpark-update-damage-missing-enum-validation — done — 2026-10-20 21:10
- commit: `fix(fuhrpark): validate severity and status on damage updates` (SHA via `git log`)
- gebaut: `updateDamageRequest.Severity`/`.Status` tragen jetzt `validate:"omitempty,oneof=..."` (dieselbe Menge wie beim Erzeugen bzw. `DamageStatus`-Konstanten) -> 400 statt RPC-Durchlauf. Zusätzlich prüft `Service.UpdateDamage` beide Felder (`ErrInvalidInput` -> `InvalidArgument`), damit der Wertebereich nicht nur am REST-Rand hängt. Der alte Pinning-Test ist zum Regressionstest umgebaut (400 + Feldname), neuer Service-Test deckt Fehl- und Gutfall.
- vorab: `fix-fuhrpark-delete-driver-license-no-last-check-guard` verlangt in der Unit selbst eine Vorab-Entscheidung Luke (harter 409 vs. Audit-Event) -> `blocked`, im selben Commit nach `BACKLOG-NEXT.yml` verschoben.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (internal/fuhrpark/..., internal/gateway inkl. TestOpenAPIRouteDrift, internal/server; DATABASE_URL als kmuhub_app, -p 2; 0 SKIP) | migration n.a. | rls-smoke n.a.
- coverage: internal/fuhrpark 81,5 % -> 81,9 % (Bugfix-Unit, kein Coverage-Ziel; Vorher-Wert per stash auf HEAD)
- mutations-probe: (1) Gateway-Tags entfernt -> `TestHandleUpdateDamage_RejectsValuesOutsideTheEnumSets` rot; (2) Service-Checks auf `if false` -> `TestService_UpdateDamage_RejectsValuesOutsideTheEnumSets` rot; beides per cp-Sicherung zurückgedreht, Diff nur die beabsichtigten Zeilen
- verify vorgaenger: sauber (Iteration 14, 8cea8ead: nur gobdarchive-Service/Errors/Tests, tenant-gescopetes `GetByID`, kein Proto/Route/Guard/Migration)
- neue-units: keine
- offen: (1) Verhaltensänderung: `PATCH /fuhrpark/damages/{id}` mit unbekanntem severity/status liefert 400. (2) Rest-Lücke, nicht gebaut: `PATCH` mit `status: resolved` umgeht den Übergangs-Guard von `ResolveDamage` (kein `resolved_at`/`resolved_by`) — die Unit verlangt ausdrücklich alle drei Status-Werte. (3) `ReportDamage` prüft Severity im Service weiterhin nicht (nur Gateway-Tag). (4) Keine Goroutinen in neuen Tests.

## Iteration 16 — fix-hr-approve-correction-self-approval — done — 2026-10-20 21:25
- commit: `fix(hr): refuse self-approval of time corrections` (SHA via `git log`)
- vorab: `fix-fuhrpark-trip-logs-private-flag-not-filtered` verlangt in der Unit selbst eine Vorab-Entscheidung Luke (private Fahrten per Default ausblenden vs. `include_private`-Parameter) -> `blocked`, im selben Commit nach `BACKLOG-NEXT.yml` verschoben.
- gebaut: `Service.ApproveTimeCorrection` lehnt `approverID == correction.EmployeeID` mit dem neuen Sentinel `ErrCannotApproveOwnCorrection` ab (nach dem Status-Check, vor Wochensperre und Write — kein Statuswechsel, kein `CorrectionApprovedBy`). `mapHRError` mappt ihn auf `PermissionDenied` -> 403 über `respondGRPCError`. Beide Seiten sind user_id-basiert (`SubmitTimeCorrection` bekommt die user_id als employeeID), derselbe Vergleich wie bei `leave.ErrCannotApproveOwnRequest`. Pinning-Test umgedreht zu `..._RejectsSelfApproval` (Fehler, Status bleibt pending, kein Approver/Zeitstempel); Sentinel in `TestMapHRError_AllSentinels` ergänzt. Keine Route, kein Proto, keine Migration, keine Guards geändert.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (internal/biz/hr/..., internal/server, internal/gateway inkl. TestOpenAPIRouteDrift; 0 SKIP; DATABASE_URL als kmuhub_app, -p 2) | migration n.a. | rls-smoke n.a.
- coverage: internal/biz/hr/timetracking 63,7 % -> 63,8 % (Bugfix-Unit, kein Coverage-Ziel; Vorher-Wert per stash auf HEAD)
- mutations-probe: (1) Guard auf `if false && ...` -> `TestApproveTimeCorrection_RejectsSelfApproval` rot; (2) Mapping-Case auf anderen Sentinel umgebogen -> `TestMapHRError_AllSentinels/CannotApproveOwnCorrection` rot; beides per cp-Sicherung zurückgedreht, Diff nur die beabsichtigten Zeilen
- verify vorgaenger: sauber (Iteration 15, d9912f13: nur validate-Tags/Service-Check/Tests in fuhrpark+gateway, keine neue Route, kein Proto/Guard/Migration)
- neue-units: keine
- offen: (1) Verhaltensänderung: `POST` der Korrektur-Genehmigung durch den Antragsteller selbst liefert 403 statt 200 — Ein-Personen-Tenants (ein Admin, der nur eigene Korrekturen hat) können eigene Korrekturen damit nicht mehr freigeben; das ist die gewollte Vier-Augen-Regel, aber im Pilot ohne zweiten Genehmiger relevant. Das FE zeigt den Approve-Button evtl. weiter für eigene Korrekturen (nicht geprüft). (2) Ungenutzter Sentinel `ErrNotAuthorizedToApproveCorrection` bleibt stehen. (3) Keine Goroutinen in neuen Tests.

## Iteration 17 — fix-hr-settings-never-consumed-by-timetracking — done — 2026-10-20 21:04
- commit: `fix(hr): apply tenant work-hour settings to time tracking` (SHA via `git log`)
- gebaut: Neuer Helfer `Service.workHoursPolicy(ctx, tenantID)` liest `settingsRepo.GetByTenant` (pro Aufruf, wie `employeeRepo.GetByUserID`, kein Cache) und liefert `(workHoursPerDay, maxDailyHours)`. Fallback 8h/10h bei nil-Repo, `ErrSettingsNotFound`, Lesefehler (`slog.Warn`) und Werten <= 0 (`UpdateHRSettings` speichert weggelassene Felder als 0). `GetTimeBalance`, `GetTimeAnalytics` (inkl. DayTrend-Tagesziel), `GetTeamTime` (einmal vor der Schleife) und `GetMyWeekStatus` rechnen mit `WorkHoursPerDay` statt fester 8h; der `ClockIn`-Block nutzt `MaxDailyHours`, aber nur nach unten — Werte > 10 werden auf die ArbZG-§3-Grenze gedeckelt. Fehlertext von `ErrMaxDailyHoursExceeded` ohne feste „10 hours" (FE matcht nicht darauf, geprüft). Keine Route, kein Proto, keine Migration.
- gate: build ok (biz/hr, server, gateway, cmd/biz, cmd/gateway, -p 2) | vet ok | lint ok (0 issues) | test ok (internal/biz/hr/... inkl. 10 DB-Tests in timetracking, 0 SKIP; internal/server; internal/gateway inkl. TestOpenAPIRouteDrift; DATABASE_URL als kmuhub_app) | migration n.a. | rls-smoke n.a.
- coverage: internal/biz/hr/timetracking 63,8 % -> 64,6 % (Bugfix-Unit, kein Coverage-Ziel; Vorher-Wert per stash auf HEAD)
- mutations-probe: (1) Balance auf 8h zurück -> `TestGetTimeBalance_UsesTenantWorkHoursPerDay` rot; (2) Analytics auf 8h -> `TestGetTimeAnalytics_UsesTenantWorkHoursPerDay` rot; (3) Team auf 8h -> `TestGetTeamTime_UsesTenantWorkHoursPerDay` rot; (4) ClockIn auf feste 600 -> `TestClockIn_UsesTenantMaxDailyHours` rot; (5) Deckel entfernt -> `TestClockIn_MaxDailyHoursAboveArbZGIsCappedAtTen` rot; (6) `> 0` durch `!= 0` -> `TestWorkHoursPolicy_FallsBackToDefaults` rot; jeweils per cp-Sicherung zurückgedreht, `cmp` bestätigt
- verify vorgaenger: sauber (Iteration 16, 471a6a12: Sentinel + Service-Check + Mapping auf PermissionDenied, kein Proto/Route/Guard/Migration)
- neue-units: keine
- offen: (1) **Annahme zur Prüfung:** `max_daily_hours` > 10 wird auf 10 gedeckelt (ArbZG §3; Verlängerung nur per Tarifvertrag §7, den die Einstellung nicht abbildet). Das Gateway erlaubt weiter bis 24 — die UI speichert also z. B. 12 ohne Fehler, wirksam bleibt 10. Will Luke Tarif-Tenants >10h erlauben, braucht es eine bewusste Entscheidung. (2) Verhaltensänderung: Tenants mit gespeichertem `work_hours_per_day` ≠ 8 sehen ab Deploy andere Salden/Soll-Werte (auch rückwirkend für die laufende Woche); Prod hat laut Lage 1 Tenant, Wert nicht geprüft (kein Prod-Zugriff). (3) `BreakAfterHours` und `AUThresholdDays` bewusst nicht verdrahtet: AU läuft bereits in `leave`, die Pausenregel ist gesetzliches Minimum (ArbZG §4) und hängt an `compliance`, nicht an diesem done_when. (4) `GetMyWeekStatus` nutzt den Helfer ebenfalls, hat aber keinen eigenen Settings-Test. (5) Keine Goroutinen in neuen Tests.

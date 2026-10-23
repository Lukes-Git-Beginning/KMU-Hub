# Backend-Nachtloop — Journal Lauf 15

Append-only. Ein Eintrag je Iteration, **ans Dateiende**, nie einsortieren. Form und Pflichtzeilen
stehen in `ITERATION.md` Schritt 6.

Frühere Läufe liegen vollständig im Archiv:
`archive/lauf-1-2/` (58 Units) · `archive/lauf-3/` (61) · `archive/lauf-4/` (54) ·
`archive/lauf-5/` (41) · `archive/lauf-6/` (46) · `archive/lauf-7/` (71) ·
`archive/lauf-8/` (94) · `archive/lauf-9/` (37) · `archive/lauf-10/` (93) ·
`archive/lauf-11/` (131 Einträge, davon 121 done) · `archive/lauf-12/` (99, davon 41 done) ·
`archive/lauf-13/` (120, davon 21 done — abgestürzt) · `archive/lauf-14/` (110, alle done —
Backlog leergelaufen, gemergt als `cadf4bf4`). Alle inkl. Laufbilanz.

---

## Laufkontext

- **Ausgangspunkt:** Lauf 14 gemergt als `cadf4bf4` und deployt, danach sechs Fixes aus der
  Nachbearbeitung, zuletzt `ea5a39a9` (Kalender-Akteur). `main` = `backend-loop` = `ea5a39a9`
  beim Staging. CI grün auf `ea5a39a9` (Run 39957283803, alle fünf Jobs inkl. E2E).
- **Migrationen:** Repo-Kopf = lokaler DB-Kopf = Produktionskopf = **332**, `dirty = f`. Immer
  zur Laufzeit ermitteln: `ls backend/migrations | grep -E "^[0-9]{6}" | sort | tail -1`
- **Coverage-Start:** **72,6 %** gesamt bei Gate 15 % (Run 39957283803). Bezugswerte je Paket im
  Kopf von `BACKLOG.yml`.
- **Queue:** 26 Units, roter Faden „Funktioniert es für einen echten Nutzer?“ —
  Totalausfälle und entschiedene Fixes (9), Scans (8), E2E als echter Nutzer (6),
  Coverage-Füllung (3). Begründung und Blockfolge im Kopf von `BACKLOG.yml`.
- **CI-E2E erweitert:** der E2E-Job startet seit dem Staging zusätzlich `biz` und
  `notification` (nur auf `backend-loop`, abgenommen im Morgen-PR).

## Was aus Lauf 14 gelernt wurde — vier Dinge, die dich direkt betreffen

1. **Ein Totalausfall ist nie „nur dokumentiert“.** `calendar_grpc.go` gab an acht Stellen
   jedem Nutzer 403, weil `uuid.Nil` als Akteur durchgereicht wurde. Ein früherer Lauf hatte
   das gefunden, in einen Testkommentar geschrieben und die Tests mit `uuid.Nil`-Ownern grün
   gemacht. Findest du so etwas: Fix-Unit an den Kopf, im selben Commit (`ITERATION.md`
   Schritt 6).
2. **Prüf gegen das Desktop-FE, nicht nur gegen Backend-Aufrufer.** Neue Pflichtzeile
   `desktop-aufrufer:` im Journal. MSW-Mocks sind kein Beleg.
3. **`-race` sieht nur die CI.** Lauf 13 baute zwei Data Races, Lauf 14 zwei flaky Tests
   (bcrypt-Kosten unter `-race`, Goroutine-Baseline im Teardown). Lokal grün heißt hier nichts —
   Tests mit Goroutinen in `offen:` vermerken.
4. **E2E läuft lokal nicht.** Gate ist `go vet -tags=e2e ./test/e2e/`, abgenommen erst in CI
   (`GATE-COMMANDS.md`, Abschnitt „E2E“).

---

## Iteration 1 — fix-document-virtual-files-nil-user — done — 2026-10-23 17:30
- commit: (siehe git log, Subject `fix(document): list virtual files as the calling user`)
- gebaut: `DocumentGRPCServer.ListVirtualFiles` löst den Aufrufer mit `callerID(ctx)` auf statt `uuid.Nil` (fehlt er: `Unauthenticated`). Gateway akzeptiert zusätzlich `source_type` als Alias für `source` (so sendet es der Desktop; vorher war der Filter wirkungslos), Spec-Parameter ergänzt.
- gate: build ok | vet ok | lint ok | test ok | migration n.a. | rls-smoke n.a. (keine Tabelle/Policy angefasst; der neue DB-Test läuft als `kmuhub_app` unter RLS)
- coverage: internal/server 74,8 % -> 74,8 % (neuer Test deckt nur noch die geänderte Handlerzeile) · internal/document/virtual 81,7 % -> 81,7 % (unverändert, nicht angefasst). 0 Skips, 4783 PASS in server + virtual.
- mutations-probe: `uuid.Nil` statt `userID` zurückgedreht -> `channel_member_sees_the_chat_file` rot (expected 1, actual 0); zurückgedreht, Diff sauber.
- verify vorgaenger: sauber (`ea5a39a9`: nur calendar_grpc.go + Tests, kein Proto/Route/Guard/Tabelle; CI laut Backlog-Kopf grün).
- neue-units: `fe-virtual-files-view-wire-shape` (in BACKLOG-NEXT.yml, FE-Fund, kein Backend-Totalausfall)
- desktop-aufrufer: `GET /documents/virtual` -> `document-client.ts:526` / `useDocuments.ts:469` -> bricht teilweise: (a) FE sendet `source_type` statt `source` (am Gateway behoben, Alias), (b) FE sendet `limit`/`offset`, Gateway liest `page`/`page_size` (FE-Unit), (c) `source_type` im FE als String, im Wire als Zahl (FE-Unit), (d) `DokumentePage.tsx:234-242` verwirft die Daten, die Ansicht ist nicht verdrahtet (FE-Unit). Antwort `{files,total}` passt.
- offen: Gateway-Test prüft nur den Param-Helfer, nicht den RPC-Aufruf (kein gRPC-Client-Stub im Gateway-Testaufbau). Stale Kommentar in `document_grpc_test.go:25,527` nennt `ListVirtualFiles` noch „untestable“ — nicht angefasst. Goroutinen: keine, kein Race-Risiko.

## Iteration 2 — fix-team-inbox-manage-permission-bypass — done — 2026-10-23 17:50
- commit: (siehe git log, Subject `fix(inbox): let team inbox managers edit members of any team inbox`)
- gebaut: `kommunikation:team_inbox:manage` erreichte den Service nie (Permissions liegen nur im HTTP-Kontext). Der gRPC-Client-Interceptor schickt jetzt genau die Allowlist-Keys als `x-user-perms` (nur dieser eine Key), der Server-Interceptor filtert erneut. `InboxGRPCServer` setzt daraus `team.WithManageAll(ctx)` für Update/AddMember/RemoveMember; `team.Service` überspringt dann den Inbox-Admin-Check, nachdem die Inbox im Aufrufer-Tenant gefunden wurde. Letzter-Admin-Schutz bleibt.
- gate: build ok | vet ok | lint ok (0 issues) | test ok | migration n.a. | rls-smoke n.a. (keine Tabelle/Policy; DB-Test läuft als `kmuhub_app`)
- coverage: internal/inbox/team 66,5 % -> 68,4 % (Service-Tests + DB-Test). 0 Skips in inbox/middleware.
- mutations-probe: `canManageAll`-Zweig in `requireTeamAdmin` entfernt -> `ManageAllWaivesInboxAdminCheck`, `ManageAllStillKeepsLastAdmin` und der DB-Test rot; zurückgedreht, Diff sauber.
- verify vorgaenger: sauber (`290e6ac9`: kein Proto, Route nur Query-Alias mit Spec-Eintrag, kein Guard, keine Tabelle; Fix über `callerID`).
- neue-units: keine
- desktop-aufrufer: n.a. (keine Validierungs- oder Antwortform-Änderung; `POST/DELETE /inbox/teams/{id}/members` liefert nur bei vorher 403 jetzt Erfolg)
- offen: (1) `DeleteTeamInbox` bleibt `RequireRole("admin","manager")` + Inbox-Admin im Service: die Manage-Permission wirkt dort bewusst nicht (Entscheidung betraf Mitglieder; Update ist mitgezogen, weil derselbe Guard). (2) Die drei Zeilen Glue in `inbox_grpc.go` (`withTeamManage`) sind nur indirekt belegt (Interceptor-Test + Service-Test), kein Test auf Server-Ebene. (3) Permissions im Service-Kontext sind jetzt eine Teilmenge (nur Allowlist) — wer dort `GetUserPermissions` nutzt, bekommt keine volle Liste. (4) Keine Goroutinen, kein Race-Risiko.

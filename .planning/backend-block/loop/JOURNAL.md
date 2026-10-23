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

## Iteration 3 — wire-biz-event-emitters-for-finance-triggers — done — 2026-10-23 17:35
- commit: (siehe git log, Subject `feat(biz): wire invoice and quote event emitters for automation triggers`)
- gebaut: `cmd/biz/event_wiring.go` (`wireFinanceEventEmitters`) setzt `SetEventEmitter(<pkg>.NewPGEventEmitter(pool))` für invoiceSvc und quoteSvc, aufgerufen direkt nach der Konstruktion in `main.go`. Damit feuern `biz.invoice.sent` und `biz.quote.created` wirklich. Lexware bleibt auf `noopEmitter` (lean:-Marker unverändert). Zwei DB-Tests (invoice `Send`, quote `Create`) hören mit `LISTEN events` auf dem echten Kanal und prüfen Typ UND `tenant_id`.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (invoice, quote, automation/..., cmd/biz) | migration n.a. | rls-smoke n.a. (keine Tabelle/Policy)
- coverage: internal/biz/invoice 61,3 % -> 61,5 % · internal/biz/quote 49,6 % -> 50,1 % (Vorher-Werte aus dem Backlog-Kopf, nicht selbst gemessen; nur Testcode hinzugekommen). 0 Skips in beiden Paketen.
- mutations-probe: (1) `invoiceSvc.SetEventEmitter`-Zeile im Helper entfernt -> `TestWireFinanceEventEmitters_SetsBothEmitters` rot (`was <nil>`). (2) `emitEvent` in invoice zum No-op gemacht -> `TestService_Send_PGEmitterPublishesTenantScopedEvent` rot (kein Event). Beide zurückgedreht, Diff sauber.
- verify vorgaenger: sauber (`ec495841`: kein Proto, keine Route, kein neuer Guard, keine Tabelle; `UpdateTeamInbox` im Server holt die Inbox vorher tenant-gescoped, WithManageAll greift erst danach).
- neue-units: keine
- desktop-aufrufer: n.a. (keine Validierungs- oder Antwortform-Änderung; nur Event-Emission)
- offen: (1) Der Test belegt den Helper, nicht den einen Aufruf in `main.go` — fällt dieser Aufruf weg, wird kein Test rot (bewusst, `main()` ist nicht testbar). (2) Mit dem Deploy beginnen die Events zu fließen; Prod-Prüfung vom 2026-10-23: `automations` 0 Zeilen, es feuert also nichts Bestehendes. (3) Keine Goroutinen im neuen Code, kein Race-Risiko.

## Iteration 4 — feat-gobd-verify-integrity-on-export — done — 2026-10-23 18:05
- commit: (siehe git log, Subject `feat(biz): verify GoBD document integrity before handing out a download`)
- gebaut: "Export" = Ausgang des Archivs. Der CSV-Export (`GenerateGoBDExport`) liest keine Archivdokumente, einziger Weg, auf dem Archivbytes das System verlassen, ist `GetDownloadURL`. Dort läuft jetzt `VerifyIntegrity` vor dem Presign (gemeinsame Funktion, einziger Aufrufer `DownloadGobdDocument`; `checkedBy` = echter Aufrufer). Bruch -> `ErrIntegrityMismatch`, keine URL, kein Access-Event, der Prüfvermerk bleibt im Audit-Trail. Fehler-Mapping: `FailedPrecondition` (-> HTTP 409 mit lesbarer Meldung, vorher `Internal`/500); 409 in `openapi.yaml` ergänzt. Entscheidung Abbruch statt Vermerk: ein Vermerk in einer Presigned-URL ist nicht transportierbar, und die Download-Antwort hat kein Feld dafür. Nicht lesbares Objekt oder nicht schreibbares Prüf-Event bricht den Download ebenfalls ab (fail-closed).
- gate: build ok | vet ok | lint ok (0 issues) | test ok (gobdarchive, server, gateway inkl. TestOpenAPIRouteDrift) | migration n.a. | rls-smoke n.a. (keine Tabelle/Policy; DB-Test läuft als `kmuhub_app`)
- coverage: internal/biz/gobdarchive 90,1 % -> 90,2 % (Vorher-Wert aus dem Backlog-Kopf). 0 Skips; DB-Tests `TestGetDownloadURL_DB_VerifiesBeforeHandingOut` und `TestVerifyIntegrity_DB_*` real gelaufen.
- mutations-probe: `VerifyIntegrity`-Aufruf in `GetDownloadURL` entfernt -> `TestGetDownloadURL_DB_VerifiesBeforeHandingOut`, `..._TamperedBytes_NoURLNoAccessEvent`, `..._Success_AppendsAccessEvent`, `..._UnreadableObject_NoURL` rot; zurückgedreht, Diff sauber.
- verify vorgaenger: sauber (`3e7a9300`: kein Proto, keine Route, kein Guard, keine Tabelle; Emitter-Wiring in cmd/biz, Tests prüfen Typ und tenant_id).
- neue-units: keine
- desktop-aufrufer: `GET /finance/gobd-archive/{id}/download` -> in `desktop/src/renderer/src/api/` kein Aufrufer (nur generierte `types.ts`); Antwortform unverändert, neu ist nur der 409-Fall -> passt.
- offen: (1) Jeder Download schreibt jetzt zusätzlich einen `integrity_check`-Event und lädt das Objekt einmal vollständig (Archivgröße ist begrenzt; bei Durchsatzproblemen Prüfung cachen). (2) Hat jemand ein Dokument manipuliert, ist es per Download nicht mehr abrufbar, bis der Befund geklärt ist: gewollt, aber Luke sollte wissen, dass das Support-Anfragen erzeugt. (3) Keine Goroutinen, kein Race-Risiko.

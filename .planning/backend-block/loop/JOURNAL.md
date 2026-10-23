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

## Iteration 5 — fix-biz-actor-attribution-uuid-nil — done — 2026-10-23 17:26
- commit: (siehe git log, Subject `fix(biz): act as the calling user in finance send, cancel and convert RPCs`)
- gebaut: Sieben Stellen in `biz_grpc.go` lösen den Akteur mit `callerID(ctx)` auf statt `uuid.Nil` (fehlt der Aufrufer: `Unauthenticated`, nach der ID-Validierung). Wohin der Akteur geht: `ConvertQuoteToInvoice` -> `finance_invoices.created_by` · `SendInvoice` -> `sent_by` im unveränderlichen GoBD-Snapshot (`snapshot_data`) und `ActorID` des Events `biz.invoice.sent` · `CancelInvoice` (versendet) -> `created_by` der Storno-Gutschrift + Log `cancelled_by` · `SendDunning`/`SendDunningNotice` -> Log `sent_by` · `SendQuote` und `SendCreditNote`: das Servicemodell ignoriert den Parameter, nichts persistiert (Fix trotzdem einheitlich). Die "Proto doesn't include user_id"-Kommentare sind weg.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (server, biz/..., gateway inkl. TestOpenAPIRouteDrift) | migration n.a. | rls-smoke n.a. (keine Tabelle/Policy; DB-Test läuft als `kmuhub_app`)
- coverage: internal/server 74,8 % -> 74,9 % (gemessen vor: 74,8 %, nach: 74,9 %). 0 Skips in `internal/server`.
- mutations-probe: `Cancel`, `SendInvoice`, `CreateFromQuote` und `SendDunningNotice` auf `uuid.Nil` zurückgedreht -> `..._DB/cancelling_a_sent_invoice...` (created_by der Storno-Gutschrift), `..._DB/sending_an_invoice...` (snapshot `sent_by`) und `SendDunning_LogsTheCallerAsSender/SendDunningNotice` rot; zurückgedreht, Diff sauber. (Die Convert-Probe lief mit, der Ausgabe-Ausschnitt zeigte sie nicht einzeln.)
- verify vorgaenger: sauber (`ee63b341`: kein Proto, keine neue Route, kein Guard, keine Tabelle; 409 in der Spec ergänzt, `VerifyIntegrity` vor Presign).
- neue-units: keine
- desktop-aufrufer: `POST /finance/invoices/{id}/send|cancel`, `/quotes/{id}/send|convert`, `/credit-notes/{id}/send`, `/dunning/.../send` -> Antwortform und Validierung unverändert; neu ist nur `Unauthenticated`, wenn kein Aufrufer im Kontext steht, und der Gateway reicht die User-ID über den Outbound-Interceptor immer durch -> passt.
- offen: (1) `go test ./internal/biz/...` ohne `-p 2` scheitert lokal mit "remaining connection slots are reserved" (zu viele parallele Pakete gegen die Dev-DB); mit `-p 2` alles grün — Umgebung, nicht der Fix. (2) Der Dunning-Test fängt `slog` über `slog.SetDefault` ab (Log ist der einzige Ort des Akteurs); parallele Tests in `internal/server`, die ebenfalls loggen, stören nicht, weil nach Nachricht + `dunning_id` gefiltert wird. (3) Bereits persistierte Datensätze mit `created_by`/`sent_by` = 00000000-… bleiben unverändert (GoBD-Snapshots sind unveränderlich, kein Backfill). (4) Keine Goroutinen, kein Race-Risiko.

## Iteration 6 — fix-dsar-third-party-matching — done — 2026-10-23 17:35
- commit: (siehe git log, Subject `fix(gdpr): stop disclosing namesakes and the employer in DSAR matches`)
- gebaut: (1) `guestVisitorKey` gruppiert Gast-Sitzungen ohne E-Mail nicht mehr über den kleingeschriebenen Anzeigenamen, sondern als `'session:' || gs.id` — jede solche Sitzung ist ein eigenes Subjekt. (2) Eingangsrechnungen und Bankumsätze werden nur noch über den vollen Namen des Kontakts abgeglichen, nie über `c.company` (Firma = Dritter/Arbeitgeber). (3) Das Fahrtenbuch fällt aus dem Nutzerzweig: `trip_logs` hat keine User-Verknüpfung, nur Freitext `driver_name`; `fuhrparkModules` + `driverNameVariants` ersetzt durch `vehicleDamagesModule` (nur FK-Treffer), `lean:`-Marker mit Upgrade-Trigger (Fahrer-FK).
- gate: build ok (security/..., cmd/auth, cmd/gateway) | vet ok | lint ok (0 issues) | test ok (security/... komplett) | migration n.a. | rls-smoke n.a. (keine Tabelle/Policy; DB-Tests laufen als `kmuhub_app`)
- coverage: internal/security/gdpr 76,1 % -> 76,0 % (gemessen; gedeckter Fahrtenbuch-Code ist entfernt worden, deshalb leicht runter). 0 Skips, 67 Integrationstests real gelaufen.
- mutations-probe: (1) Gast-Schlüssel zurück auf Anzeigename -> `TestSearchByQuery_ExternalVisitors_Integration` rot (1 statt 2 Subjekte). (2) Finanzdatei auf HEAD (Firmenabgleich) -> `..._ContactFinanceBeyondInvoices_Integration` rot (ER-1 und Bankumsatz „Finanz AG" tauchen auf). (3) Fuhrpark + Aufrufer auf HEAD (Namensabgleich) -> `..._UserFuhrparkDamagesNotLogbook_Integration` rot (Fahrtenbuch-Modul, Fahrten des Namensvetters). Alle zurückgedreht, Diff sauber.
- verify vorgaenger: sauber (`5294eba3`: kein Proto, keine Route, kein Guard, keine Tabelle; sieben `uuid.Nil` -> `callerID(ctx)`, DB-Tests für Storno/Snapshot/Mahnung).
- neue-units: `decide-trip-logs-driver-user-fk` (BACKLOG-NEXT.yml, Datenmodell-Entscheidung Luke)
- desktop-aufrufer: `GET /security/dsar/search` -> `modules/security/DSARSearchPage.tsx` (kein Client in `api/`/`hooks/`, Aufruf direkt in der Seite) -> passt: Person-ID und Modulnamen werden nur opak/als Anzeige genutzt, Modul-Icons fallen auf `FileText` zurück; Gast-IDs heißen jetzt `guest:session:<uuid>` statt `guest:name:<name>`.
- offen: (1) Folge der Entscheidung: die eigenen Fahrten eines Fahrers fehlen jetzt in seiner Art.-15-Auskunft, bis `trip_logs` eine Fahrer-FK hat (Unit oben). (2) Ein Kontakt, der nur eine Firma ist (ohne Vor-/Nachname), bekommt keine Eingangsrechnungen/Bankumsätze mehr — juristische Person, kein DSGVO-Subjekt. (3) Namensabgleich über den vollen Namen bei Eingangsrechnungen/Bankumsätzen bleibt (als „Namensabgleich" markiert) — dokumentiertes Restrisiko laut Entscheidung. (4) Keine Goroutinen, kein Race-Risiko.

## Iteration 7 — fix-produktion-capacity-week-bounds-europe-berlin — done — 2026-10-23 17:50
- commit: (siehe git log, Subject `fix(produktion): read capacity weeks in Europe/Berlin instead of UTC`)
- gebaut: `isoWeekBounds` rechnet Wochengrenzen in `planningLocation` (Europe/Berlin, einmal beim Paketstart geladen, `lean:`-Marker mit Upgrade-Trigger „erster Mandant außerhalb Europe/Berlin“) statt in UTC. Eine Buchung Montag 00:30 Ortszeit zählt damit zur Woche, die an diesem Montag beginnt. `_ "time/tzdata"` im Paket, weil das Alpine-Runtime-Image kein zoneinfo hat und `LoadLocation` sonst still auf UTC zurückfiele (der Fix wäre in Produktion wirkungslos).
- gate: build ok | vet ok | lint ok (0 issues) | test ok (produktion/...) | migration n.a. | rls-smoke n.a. (keine Tabelle/Policy; DB-Tests laufen als `kmuhub_app`)
- coverage: internal/produktion 79,0 % -> 79,0 % (gemessen vor/nach; nur Testcode und eine Zeile Initialisierung hinzugekommen). 0 Skips.
- mutations-probe: `isoWeekBounds` zurück auf `time.UTC` -> `TestGetCapacityOverview_WeekBoundaryUsesBerlinTime_RealDB` (booked = 0 statt 2) und `TestIsoWeekBounds_BerlinWallClock` rot; zurückgedreht, Diff sauber.
- verify vorgaenger: sauber (`ebe97670`: kein Proto, keine Route, kein Guard, keine Tabelle; nur DSAR-Matching verengt, DB-Tests mit Mutations-Proben laut Journal).
- neue-units: keine
- desktop-aufrufer: n.a. (keine Validierungs- oder Antwortform-Änderung; `GET .../capacity` liefert dieselbe Form, nur die Wochengrenze verschiebt sich um 1-2 h)
- offen: (1) `schichten` (`jarbschgLocation`, gleicher `LoadLocation("Europe/Berlin")` mit stillem UTC-Fallback) importiert `time/tzdata` im eigenen Paket nicht; ob `cmd/schichten` es transitiv linkt, habe ich nicht geprüft — falls nicht, rechnet der Jarbschg-Check im Container in UTC. (2) Bestehende Buchungen werden nicht umgebucht, nur die Summierung ändert sich. (3) Keine Goroutinen, kein Race-Risiko.

## Iteration 8 — chore-work-project-actor-and-admin-bypass-lean — done — 2026-10-23 18:20
- commit: (siehe git log, Subject `chore(work): pass the calling user through project RPCs and mark the open-project bypass`)
- gebaut: Alle 15 `uuid.Nil`-Akteur-Stellen in `work_grpc.go` (der Backlog nannte 16; Grep `projectService\.\w+\(.*uuid\.Nil, true` ergab 15) lösen den Akteur per `callerID(ctx)` auf (`Unauthenticated` ohne Aufrufer, nach der ID-Validierung). `isAdmin=true` bleibt, hinter der Konstante `projectsOpenToTenant` mit `lean:`-Marker und Upgrade-Trigger (erster Kunde, der nach Mitgliedschaft trennen will); dasselbe für die drei Kalender-Ressourcen-Stellen (`resourcesOpenToTenant`, Marker; Kommentar "Gateway handles auth" ersetzt, tatsächlicher Schutz = `RequirePermission("resources", ...)` in `route_calendar.go`). Neu: die drei Mitglieder-Logzeilen im Projekt-Service tragen jetzt `actor_id`.
- akteur-tabelle: `AddMember`/`RemoveMember`/`UpdateMemberRole` -> `actorID` wird nur für die (bei isAdmin=true übersprungene) Rollenprüfung gelesen, nirgends persistiert (kein `added_by`, kein Audit-Event) -> einziger Ort ist jetzt die slog-Zeile. `Get/List/Update/Archive/Delete/ListMembers` -> `userID` nur für die übersprungene Mitgliedschaftsprüfung / im Repo-List ignoriert, kommt nirgends an. Der Beitrag der Unit ist daher der Marker plus die festgeschriebenen Tests, nicht mehr.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (server, work/..., gateway inkl. TestOpenAPIRouteDrift) | migration n.a. | rls-smoke n.a. (keine Tabelle/Policy)
- coverage: internal/server 74,8 % -> 75,0 % (Vorher-Wert aus dem Backlog-Kopf, nicht selbst gemessen). 0 Skips in `internal/server` und `internal/work/project`.
- mutations-probe: (1) `AddMember`-Aufruf auf `uuid.Nil` zurück -> `TestWorkGRPCServer_MemberMutationsLogTheCaller` rot. (2) Guard in `ListProjectTeamUtilization` entfernt -> `TestWorkGRPCServer_ProjectRPCsRefuseMissingCaller/ListProjectTeamUtilization` rot. Zurückgedreht, Diff sauber.
- verify vorgaenger: sauber (`cc12efe9`: kein Proto, keine Route, kein Guard, keine Tabelle; nur `isoWeekBounds` + Test, `time/tzdata` im Paket).
- neue-units: keine
- desktop-aufrufer: `/projects/*` und `/projects/{id}/members/*` -> Antwortformen unverändert; neu ist nur `Unauthenticated`, wenn kein Aufrufer im Kontext steht, und der Gateway reicht die User-ID immer durch. Einziger andere WorkServiceClient-Nutzer (`automation/action/work_actions.go`) ruft nur `CreateTask` -> unberührt -> passt.
- offen: (1) Test-Helper `ctxWithWorkTenant` trägt jetzt einen Nutzer (bestehende Erfolgspfad-Tests brauchten das); `work_grpc_nil_slice_test.go` auf diesen Helper umgestellt. (2) Keine Goroutinen, kein Race-Risiko.

## Iteration 9 — chore-dialer-stale-contact-default-240 — done — 2026-10-23 18:30
- commit: (siehe git log, Subject `chore(deploy): default the dialer stale-contact reaper to 240 minutes`)
- gebaut: Compose-Default `DIALER_STALE_CONTACT_MINUTES` in `deploy/docker/docker-compose.yml:544` von `:-0` auf `:-240`; `docker compose config` zeigt `"240"`. Unter `deploy/` genau diese eine Zeile geändert. Der Go-Default in `config.go:128` bleibt 0 (ohne Compose keine Reaper-Aktivierung).
- gate: build n.a. | vet n.a. | lint n.a. | test n.a. (keine Go-Änderung) | migration n.a. | rls-smoke n.a.
- coverage: n.a. (kein Coverage-Ziel)
- mutations-probe: n.a.
- verify vorgaenger: sauber (`c3734957`: kein Proto, keine Route, kein Guard, keine Tabelle; `callerID(ctx)` statt `uuid.Nil`, Konstanten mit lean:-Marker, Tests mit Mutations-Proben laut Journal).
- neue-units: keine
- desktop-aufrufer: n.a. (keine Validierungs- oder Antwortform-Änderung)
- offen: (1) Ein Commit, der nur Deploy-Dateien berührt, löst weder CI noch CD aus: beim Merge zusammen mit Code-Commits ausrollen oder den CD-Workflow manuell starten (Luke). (2) Wirkt erst nach dem Deploy; ein per Env-Datei gesetzter Wert hat weiter Vorrang. (3) Ab dann räumt der Reaper Dialer-Kontakte, die länger als 240 Minuten hängen.

## Iteration 10 — scan-grpc-hardcoded-actor-and-admin-remaining — done — 2026-10-23 17:50
- commit: (siehe git log, Subject `chore(loop): record actor scan findings and queue the GoBD archive PDF fix`)
- gebaut: Scan, kein Code. Ein Totalausfall gefunden und als Fix-Unit VOR die erste todo-Unit gesetzt: `fix-biz-gobd-archive-invoice-startup-pdf-settings` (`POST /finance/gobd-archive/from-invoice/{id}` -> 500 für jeden Mandanten, weil `s.pdfGenerator` beim Start mit `GetByTenantID(ctx, uuid.Nil)` gebaut wird; per Wegwerf-Test reproduziert: `GenerateInvoicePDF` -> "missing name, street, plz, city, steuernummer or ust_id_nr"; Test wieder gelöscht). Zwei weitere Units ans Ende.
- grep-muster: (1) `uuid\.Nil` in `internal/server/*_grpc.go` (ohne Tests); (2) `(OwnerID|ActorID|CreatedBy|UserID|AuthorID|AgentID|UploadedBy|SenderID|AssigneeID|ApproverID|ReviewerID|ResolvedBy|CreatedByID)\s*[:=]+\s*uuid\.Nil` in `internal cmd`; (3) `uuid\.Nil, true|, true, uuid\.Nil|isAdmin\w* *:?= *true|IsAdmin: *true|OpenToTenant`; (4) `\((ctx|r\.Context\(\)|context\.Background\(\)), [^)]*uuid\.Nil` außerhalb `internal/server`; (5) Kommentare `gateway (handles|does|already)|handled by (the )?gateway`.
- befunde:
  | Stelle | Befund | Klasse | Unit |
  |---|---|---|---|
  | `cmd/biz/main.go:185` -> `server/biz_grpc_gobd_archive.go:94` | Startup-Generator mit Nil-Mandant, Archiv-PDF scheitert für jeden | a (Totalausfall) | `fix-biz-gobd-archive-invoice-startup-pdf-settings` (Kopf) |
  | `server/helpdesk_grpc.go:911` | KB-Artikel mit `author_id = 0000…` (NOT NULL, kein FK) | b | `fix-helpdesk-kb-article-author-id-uuid-nil` (Ende) |
  | `biz/invoice/service.go:327`, `biz/dunning/service.go:262` | System-Akteur Nil, kommentiert, kein `lean:` | c | `chore-mark-system-actor-uuid-nil-lean` |
  | `biz/bexio/contact_sync.go:218`, `biz/lexware/contact_sync.go:258` | Sync-Import mit Nil-Ersteller | c | dito |
  | `server/biz_grpc.go:2108`, `:2274` | unparsbares `created_by` -> Nil (gateway setzt es immer: `route_biz_*.go` `CreatedBy: userID`) | c | dito |
  | `server/work_grpc.go:915` | DeleteTask ohne Aufrufer -> Nil statt Unauthenticated | c | dito |
  | `crm_grpc.go:594/1372/1384/1396/1738…`, `einkauf_grpc.go:90`, `wiki_grpc.go:117/449`, `work_grpc.go:849` | `uuid.Nil` als Lösch-Sentinel für „Feld leeren" | unkritisch | – |
  | `chat_grpc.go:446` | Nil = Gast, mit Gast-Freigabe-Prüfung | unkritisch | – |
  | `email_grpc.go:1092` | Upload vor dem Versand, keine Nachricht | unkritisch | – |
  | `cmd/biz/crm_contact_adapter.go:47/134` | `IsAdmin: true` im Sync-Adapter (Hintergrundjob) | c, ohne Marker | in der chore-Unit nicht gesondert; Sync-Kontext, kein Nutzer-Pfad |
  | `calendar_grpc.go:46`, `work_grpc.go:47` | `*OpenToTenant`-Konstanten | bereits markiert (Iteration 8) | – |
- gate: build n.a. | vet n.a. | lint n.a. | test n.a. (nur Lesearbeit; ein Wegwerf-Test lief und wurde gelöscht, Diff sauber) | migration n.a. | rls-smoke n.a.
- coverage: n.a. (Scan)
- mutations-probe: n.a.
- verify vorgaenger: sauber (`71b6c296`: nur eine Compose-Zeile `:-0` -> `:-240`, keine Go-Änderung).
- neue-units: `fix-biz-gobd-archive-invoice-startup-pdf-settings` (VOR die erste todo-Unit gesetzt: verifizierter Totalausfall), `fix-helpdesk-kb-article-author-id-uuid-nil`, `chore-mark-system-actor-uuid-nil-lean` (beide ans Ende)
- desktop-aufrufer: n.a. (keine Validierungs- oder Antwortform-Änderung)
- offen: (1) Der Scan hat `uuid.Nil` nur im Go-Code der Handler/Services gesucht, nicht in SQL-Funktionen oder Consumern mit anderem Bezeichner. (2) Der Totalausfall ist per Generator-Test und Codelesen belegt, nicht per HTTP-Aufruf gegen den Stack; ob das FE den Endpunkt überhaupt aufruft, prüft die Fix-Unit.

## Iteration 11 — fix-biz-gobd-archive-invoice-startup-pdf-settings — done — 2026-10-23 18:55
- commit: (siehe git log, Subject `fix(biz): render the GoBD archive invoice PDF from the tenant's company settings`)
- gebaut: `ArchiveInvoiceDocument` lädt die Stammdaten pro Anfrage (`requireCompanySettings`, Muster der Geschwister-RPCs), prüft `ValidateCompanySettingsForPDF` vorab und liefert bei fehlenden/unvollständigen Stammdaten `FailedPrecondition` (Gateway: 409, in der Spec dokumentiert) statt `Internal`/500. Totes Feld `pdfGenerator`, Konstruktor-Parameter und die Startup-Abfrage mit `uuid.Nil` in `cmd/biz/main.go` sind entfernt (samt toter Imports); Aufruf in `einvoice/roundtrip_outbound_test.go` angepasst.
- tests: Die alten Stub-Tests setzten `srv.pdfGenerator` direkt und hielten den Bug damit grün; sie laufen jetzt über ein Settings-Repo (Fälle: kein Settings-Row, unvollständig, Lookup-Fehler). Neu `biz_grpc_gobd_archive_db_test.go` (echte Rechnung + echte `company_settings` als `kmuhub_app`): Mandant mit Stammdaten archiviert Bytes mit `%PDF`, Mandant ohne wird mit FailedPrecondition abgewiesen und archiviert nichts.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (server, biz/einvoice, gateway inkl. TestOpenAPIRouteDrift) | migration n.a. | rls-smoke n.a. (keine Tabelle/Policy; DB-Tests laufen als `kmuhub_app`)
- coverage: internal/server 75,0 % -> 75,0 % (Vorher-Wert aus dem Backlog-Kopf, nicht selbst gemessen; Zuwachs unter Rundung). 0 Skips.
- mutations-probe: `requireCompanySettings(ctx, tenantID)` im Archiv-Handler auf `uuid.Nil` -> `..._UsesTenantCompanySettings_DB/tenant_with_company_settings_archives_a_real_PDF` rot; zurückgedreht, Diff sauber.
- verify vorgaenger: sauber (`363f9cfb`: nur BACKLOG/JOURNAL, kein Code).
- neue-units: keine
- desktop-aufrufer: `POST /finance/gobd-archive/from-invoice/{invoiceId}` -> nur generierte `api/types.ts`, kein Client/Hook ruft die Route auf -> kein FE-Pfad betroffen; neue Antwort bei fehlenden Stammdaten ist 409 statt 500 -> passt.
- offen: (1) Kein HTTP-Aufruf gegen den Stack; Beleg ist der DB-Test auf Handler-Ebene. (2) Der Fix wirkt erst nach Deploy von `cmd/biz`. (3) Keine Goroutinen, kein Race-Risiko.

## Iteration 12 — scan-fe-gateway-shape-crm-kommunikation — done — 2026-10-23 18:30
- commit: (siehe git log, Subject `chore(loop): record CRM and communication shape scan findings`)
- gebaut: Scan, kein Produktivcode. Geprüft: 79 Inbox/Email-Endpunkte, 55 Kalender/CalDAV/Booking-Hooks, 70 Notification/Video-Clients + 76 Hooks, 60 CRM-Methoden/Hooks. Vier Teilscans liefen als Recherche-Agenten; die Totalausfälle wurden danach von mir am Code nachgeprüft (Kalender-Events, Feiertage, Mail-Empfänger, Inbox-`tags`/Teams, Meeting-`scheduled_start`, Aktivitäts-`due_date`, Timeline-Pfad, Prioritäts-Tabelle). Kontrollfall `GET /inbox/rules`: passt (ProtoList, `3c13fd66`). Sechs Fix-Units VOR die erste todo-Unit, eine Teilfehler-Unit ans Ende, ein FE-Sammelpaket nach BACKLOG-NEXT.yml.
- befunde (Stelle -> Befund -> Klasse -> Unit):
  | Domäne | Befund | Klasse | Unit |
  |---|---|---|---|
  | Kalender | `GET /calendar/events` nested `{event:{…}}`, FE liest flach -> kein Termin sichtbar, Widgets `NaN:NaN` | Totalausfall | `fix-calendar-events-holidays-recurring-wire` |
  | Kalender | Datum-only `end` exklusiv -> Tagesansicht leer, Sonntag fehlt | Totalausfall | dito |
  | Kalender | `GET /holidays` ignoriert `year` -> immer leer | Totalausfall | dito |
  | Kalender | Serien-Edit: flacher Body, Gateway will `changes` -> 200 ohne Wirkung; `reminder_minutes`/`rrule` verworfen | Funktionsausfall | dito |
  | E-Mail | send/forward/draft: `to` als `[{name,email}]` gegen `[]string` -> 400 für jede Mail | Totalausfall | `fix-email-compose-addresses-and-empty-arrays` |
  | E-Mail | leere `cc`/`to`/`attachments` fehlen -> "Allen antworten", .eml-Export, Thread werfen | Teilausfall | dito |
  | E-Mail | "Alle Eingänge"/Label/Filter -> `folder_id=''` -> 400 | Funktionsausfall (Proto nötig) | dito, Punkt (3) |
  | Inbox | `tags` fehlt bei leer -> `msg.tags.slice` wirft in der Konversationsliste | Totalausfall | `fix-inbox-tags-teams-bulk-wire` |
  | Inbox | Team-Inboxen: `{team_inboxes}`/`{team_inbox}`/`{members}` statt bare, Enums als Int, `user_id` vs `member_user_id` | Totalausfall der Team-Einstellungen | dito |
  | Inbox | Bulk-Pfade `/bulk/read` vs `/bulk-read`, `counts` vs `by_channel`, Regel-Test `message` vs `test_message` | Funktionsausfall | dito |
  | Notification | Quiet-Hours `days`/`is_active` vs `days_of_week`/`enabled`, Wochentag 0=So vs 1=Mo; `false`-Werte fehlen -> Checkbox springt zurück; PUT überschreibt email/sms/sound | Funktionsausfall | `fix-notification-quiet-hours-prefs-wire` |
  | Notification | `PRIORITY_BY_INT` im FE vertauscht (urgent <-> low) | FE-Fehler | `fe-handle-crm-kommunikation-shape-findings` (NEXT) |
  | Video | Meeting anlegen/ändern: `scheduled_start` ohne Zone -> 400 für jeden; `contact_id`/`deal_id` verworfen | Totalausfall | `fix-video-meetings-recordings-actionitems-wire` |
  | Video | Aufnahmen-Listen falsche Form (Einwilligungsbanner sieht nie eine Aufnahme, DSGVO), Action-Items `{}` wirft, convert `task_ids`, Presence-Bulk, Co-Hosts, Breakout-Assignment | Teil-/Totalausfall | dito |
  | CRM | Aktivität mit `due_date=YYYY-MM-DD` -> 400, auch jedes Bearbeiten | Totalausfall | `fix-crm-activity-due-date-and-timeline-route` |
  | CRM | Timeline-Pfad `/api/v1/crm/contacts/…` existiert nicht -> 404 für jeden Kontakt | Totalausfall | dito |
  | CRM/FE | rohes `fetch` ohne Auth-Header (Tags, Custom Fields, Birthdays), Leads-Mock, `completed` vs `is_completed`, camelCase-Reads, `page_size` 200 -> 20, 20-Kontakte-Deckel | FE | `fe-handle-crm-kommunikation-shape-findings` (NEXT) |
  | Rest | leere Booking-Liste, Mutes, Meeting-Liste, Notes, Logo-Clear u. a. | Randfall | `fix-kommunikation-shape-partial-failures` (Ende) |
  | CalDAV | 11/11 Methoden passen | – | – |
- gate: build n.a. | vet n.a. | lint n.a. | test n.a. (nur Lesearbeit, keine Go-Änderung) | migration n.a. | rls-smoke n.a. Preflight `backlog-check.py --preflight`: valide.
- coverage: n.a. (Scan)
- mutations-probe: n.a.
- verify vorgaenger: sauber (`5db02375`: kein Proto, keine Route, kein Guard, keine Tabelle; `ArchiveInvoiceDocument` über Settings-Repo, 409 in der Spec, DB-Test als `kmuhub_app` mit Mutations-Probe laut Journal).
- neue-units: `fix-calendar-events-holidays-recurring-wire`, `fix-email-compose-addresses-and-empty-arrays`, `fix-inbox-tags-teams-bulk-wire`, `fix-notification-quiet-hours-prefs-wire`, `fix-video-meetings-recordings-actionitems-wire`, `fix-crm-activity-due-date-and-timeline-route` (alle sechs VOR die erste todo-Unit: verifizierte Totalausfälle), `fix-kommunikation-shape-partial-failures` (Ende), `fe-handle-crm-kommunikation-shape-findings` (BACKLOG-NEXT.yml, blocked)
- desktop-aufrufer: n.a. (Scan, keine Validierungs- oder Antwortform-Änderung am Gateway); die Fix-Units tragen die Aufrufer-Prüfung selbst.
- offen: (1) Nicht nachgeprüft, nur vom Teilscan übernommen: Inbox B6/B8, E-Mail E2-E5, Video V6-V10/V16, CRM A3-A10, B11-B18, Kalender-Randfälle — die Belege stehen als Datei:Zeile in den Unit-Beschreibungen, vor dem Fix gegen den Code prüfen. (2) Die Fixes liegen am Gateway; ein Teil (Enum-Strings, Wochentag-Mapping, Zeitzone zonenloser Meeting-Zeiten) braucht Entscheidungen, die in den Units benannt sind. (3) Nicht gescannt: Request-Seite von Aktivitäts-/Deal-Tags, `/search`, Duplikaterkennung/Merge, `/reports/*`. (4) `useFeed`-Links und tote CRM-Seiten (`CRMLayout`) sind im FE-Paket vermerkt. (5) Die Recherche-Agenten haben ca. 8 USD verbraucht; für die folgenden Shape-Scans (Geld, Branchen, Inhalte, Admin) kleinere Teilscans einplanen.

## Iteration 13 — fix-calendar-events-holidays-recurring-wire — done — 2026-10-23 18:40
- commit: (siehe git log, Subject `fix(gateway): flatten calendar events and honour year, inclusive end dates and flat recurring bodies`)
- gebaut: (1) `GET /calendar/events` und `GET /calendar/events/{id}` schreiben die Event-Felder flach auf Elementebene (neu `response.ProtoFlatten`, Zusatzfelder `calendar_name`/`display_color` bleiben, leere Liste `[]`). (2) Bares Datum als `end` ist inklusiv (+1 Tag, `parseRangeEnd`). (3) `GET /calendar/holidays` wertet `year` aus (1.1.–31.12.), ohne `year` und ohne `start`+`end` jetzt 400 statt stiller 1970-Abfrage. (4) `PUT /events/{id}/recurring` nimmt die Änderungen flach oder unter `changes`. (5) `reminder_minutes` wird beim Anlegen im Service validiert (max. 3, >0, vor dem Schreiben) und gespeichert (der gRPC-Server ignorierte das Feld bisher); `rrule` auf `PUT /events/{id}` wird validiert und gesetzt (leer = entfernen). Spec nachgezogen (year, end, reminder_minutes, rrule, flacher Recurring-Body, 500).
- bewusste Abweichung: `rrule` wird auf der Recurring-Route NICHT durchgereicht (`lean:`-Marker im Code). Das FE rechnet die Regel aus dem Anzeigetext zurück (`displayToRrule`: jede `FREQ=WEEKLY…` wird `FREQ=WEEKLY`), ein Titel-Edit würde BYDAY/INTERVAL der Serie zerstören. Freischalten, sobald das FE die Original-RRULE sendet.
- tests: Handler-Tests mit echtem gRPC-Server auf Loopback (`route_calendar_wire_handler_test.go`, prüft das tatsächliche Wire-JSON und die weitergereichten RPC-Felder), Helfer-Tests, Service-Tests (Reminder, RRule setzen/validieren/löschen). Der bestehende Test `HandleListHolidays_ReachesRPC` hing am alten 1970-Verhalten und sendet jetzt `year`.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (gateway inkl. TestOpenAPIRouteDrift/Status-Drift, work/..., server/...; `swagger-cli validate` ok) | migration n.a. | rls-smoke n.a. 0 Skips in work/event und gateway.
- coverage: internal/gateway 73,6 % -> 73,8 %
- mutations-probe: fünf Brüche einzeln, jeweils rot und zurückgedreht: Flatten abgeschaltet (Handler-Tests Events/Get + Helper), `AddDate(0,0,1)` -> 0, Jahresende -> 1.1., `changeSet` ignoriert den flachen Body, `repo.SetReminders` im Create entfernt. Diff sauber.
- verify vorgaenger: sauber (`a6041d5d`: nur BACKLOG/JOURNAL).
- neue-units: keine
- desktop-aufrufer: `GET /calendar/events` -> `useEvents.ts` `useEventsInRange` + `adapters.ts expandedEventToUI` liest flach -> passt jetzt; `GET /events/{id}` -> `useEvent` (`EventResponse`) -> passt; `GET /holidays` -> `useHolidays.ts` sendet `year`/`country_code`/`subdivision_code`, liest `holidays[]` -> passt; `PUT .../recurring` -> `useUpdateRecurringEvent` sendet flach -> passt; `POST /events` `reminder_minutes` -> `uiEventToCreateRequest` -> kommt an. Der neue 400 bei Feiertagen ohne `year` trifft keinen FE-Aufruf (sendet immer `year`).
- offen: (1) Zeitzone: `start`/`end` als bares Datum bleiben UTC-Mitternacht; Termine zwischen 00:00 und 02:00 Berliner Zeit am ersten/letzten Tag können am Rand falsch einsortiert werden. (2) `rrule` auf der Recurring-Route bleibt aus (s. o.). (3) Kein Aufruf gegen den laufenden Stack; Beleg sind Handler-Tests gegen einen echten gRPC-Server. (4) Wirkt erst nach Deploy von Gateway und `cmd/work`. (5) Keine Goroutinen im Produktivcode; der Test-gRPC-Server läuft in einer Goroutine, Race erst in CI.

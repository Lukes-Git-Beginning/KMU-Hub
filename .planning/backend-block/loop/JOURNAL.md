# Backend-Nachtloop — Journal Lauf 8

Append-only. Ein Eintrag je Iteration, **ans Dateiende**, nie einsortieren. Form und Pflichtzeilen
stehen in `ITERATION.md` Schritt 6.

Frühere Läufe liegen vollständig im Archiv:
`archive/lauf-1-2/` (58 Units) · `archive/lauf-3/` (61) · `archive/lauf-4/` (54) ·
`archive/lauf-5/` (41) · `archive/lauf-6/` (46) · `archive/lauf-7/` (71, inkl. `logs/`).

---

## Laufkontext

- **Ausgangspunkt:** `backend-loop` auf `origin/main` gemergt (nicht rebased). `main` = `c6d1c972`
  (PR #20, Lauf 7). Produktion: Migrationskopf **309 clean**, 30 Container healthy, `/health` mit
  23 Services.
- **Migrationen:** Repo-Kopf = lokaler Kopf = Produktionskopf = **309**. Nächste freie **310** —
  aber immer zur Laufzeit ermitteln.
- **Lokale DB:** `docker-postgres-1` healthy, Rolle `kmuhub_app` mit Passwort `app_dev` verifiziert.
  `DATABASE_URL` ist damit kein Alibi.
- **Backlog:** `BACKLOG.yml`, Blöcke A–F. Null `blocked`-Units zum Laufbeginn — die neun aus Lauf 7
  sind am 2026-08-10 entschieden worden (zwei wurden Lauf-8-Units, vier geparkt, zwei gestrichen,
  siehe `BACKLOG-PARKED.yml`).
- **Fenster:** 15:00–19:30 und 23:00–09:00, über das einmalige Pausenfenster des Treibers als **ein**
  Lauf gefahren (`-UntilTime "09:00" -PauseFrom "19:30" -PauseTo "23:00"`). Ein Prozess, ein Push,
  ein CI-Lauf.
- **Workflow-Zustand beim Start:** `Claude PR Review` `disabled_manually`, `Security Review` vor dem
  Anlegen des Draft-PRs disabled (beide haben kein Draft-Gate und würden beim `opened`-Event zünden).

### Neu in diesem Lauf

- **`coverage:` ist Pflichtzeile** im Journal-Eintrag, mit `coverage_start:` je Unit als festem
  Bezugswert. In Lauf 7 haben nur 8 von 71 Iterationen eine Zahl notiert, obwohl Coverage das
  erklärte Laufziel war — der Endstand musste aus dem CI-Artefakt rekonstruiert werden. Die
  Messmethode ist in `GATE-COMMANDS.md` unter „Coverage messen" festgeschrieben.
- **`mutations-probe:` steht jetzt ebenfalls im Template**, nicht mehr nur im Backlog-Kopf. Sie kam
  in Lauf 7 71/71 und hat zwölf reale Produktionsbugs zutage gefördert; sie bleibt unverändert
  Pflicht.
- **Der Treiber warnt bei Drift.** Stimmt die Nummer der neuen Überschrift nicht mit seiner
  Iterationszählung überein oder fehlt der gelieferte Zeitstempel, schreibt er eine gelbe
  `DRIFT:`-Zeile ins `run.log`. Der Lauf bricht deswegen nicht ab.
- **Zeitstempel sind Pflichtwerte**, keine Formatvorschläge: exakt die Zeichenkette aus dem
  Laufkontext-Block. „(Lauf 8)" oder „(siehe Commit-Zeit)" ist ein Fehler.
- **Der Treiber liest `ITERATION.md` jetzt als UTF-8.** Bis Lauf 7 kam jeder Gedankenstrich als
  `â€”` beim Modell an (PS 5.1 liest UTF-8 ohne BOM sonst als ANSI).

---

## Iteration 1 — fix-work-tasks-listtasks-no-own-scope — done — 2026-08-10 13:27
- commit: (siehe unten, wird nach diesem Journal-Eintrag committet)
- gebaut: `ownerFilterForScopeAny` (helpers.go) als Pendant zu `ownerFilterForScope` für Routen
  hinter `RequirePermissionAny` über zwei Schlüssel — narrower key wins (ein Treffer auf `own`
  filtert, egal was der andere Schlüssel liefert, weil ein nicht vergebener Schlüssel per
  `middleware.PermissionScope` auf `ScopeAll` fällt, nicht "unbekannt"). `HandleListTasks`
  (route_work_tasks.go) ruft ihn jetzt für `tasks:read`/`work:task:read` auf und überschreibt
  ein client-seitiges `?assignee_id=` unconditionally, wenn die Scope-Prüfung auf `own` trifft.
- gate: build ok | vet ok | lint ok (0 issues) | test ok | migration n.a. (keine Migration) | rls-smoke n.a.
- coverage: internal/gateway 34,9 % -> 35,0 %
- mutations-probe: zwei Proben, beide gefangen. (1) `if !ok { return }` in HandleListTasks auf
  `if ok { return }` gedreht -> 12 Tests in TestHandleListTasks_FilterCombinations rot (alle
  erwarteten 503, bekamen 200), zurückgedreht, Diff sauber. (2) in `ownerFilterForScopeAny` die
  Schleife auf `pairs[:1]` verengt (nur erster Schlüssel zählt) -> TestOwnerFilterForScopeAny_
  OwnOnSecondKeyNarrows und TestOwnerFilterForScopeAny_OwnWithoutUserIsRejected rot, zurückgedreht,
  `git diff` zeigt exakt die beabsichtigten Zeilen.
- verify vorgaenger: n.a. — erste Iteration dieses Laufs, kein Vorgänger-Commit im Journal.
- offen: `ownerFilterForScopeAny` ist jetzt verfügbar für weitere `RequirePermissionAny`-Routen mit
  demselben Muster (nicht gesucht — Scope der Unit war ausdrücklich nur `HandleListTasks`). DB-Gate
  lief real (lokale `docker-postgres-1`, `DATABASE_URL` gesetzt, `kmuhub_app`); keine Migration in
  dieser Unit, daher kein RLS-Smoke nötig.

## Iteration 2 — fix-crm-reports-invalid-date-not-rejected — done — 2026-08-10 13:41
- commit: 755dadf4
- gebaut: `validateDateParam` (helpers.go) prüft einen bereits extrahierten Query-Wert gegen
  `dateParamFormats` (`YYYY-MM-DD`, `time.RFC3339` — dieselben Layouts wie `parseDate` in
  `internal/server/crm_grpc.go`, dort im Kommentar so verlinkt) und schreibt bei Fehlschlag ein
  400 mit dem erwarteten Format in der Meldung. `HandleGetPipelineReport`,
  `HandleGetConversionReport` und `HandleGetActivityReport` rufen ihn nach der bestehenden
  Required-Prüfung für beide Felder auf — ein unbrauchbares Datum erreicht die RPC jetzt nicht
  mehr. Die drei alten `TestHandle*Report_MalformedDateNoPanic`-Tests sind auf
  `TestHandle*Report_MalformedDateRejected` umgestellt (erwarten jetzt 400 + Format-Meldung statt
  503), plus je ein neuer `ValidDatesReachClient`-Test, der belegt, dass gültige Daten
  unverändert bis zum (in den Tests absichtlich nicht erreichbaren) gRPC-Client durchlaufen.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (0 skipped) | migration n.a. (keine
  Migration) | rls-smoke n.a.
- coverage: internal/gateway 34,9 % -> 35,0 %
- mutations-probe: `validateDateParam` auf `return true` nach der Format-Schleife gedreht (jedes
  Datum gilt als gültig) -> `TestHandleGetPipelineReport_MalformedDateRejected`,
  `TestHandleGetConversionReport_MalformedDateRejected` und
  `TestHandleGetActivityReport_MalformedDateRejected` alle drei rot (503 statt erwarteter 400,
  weil der Request jetzt bis zum nicht erreichbaren gRPC-Client durchläuft), zurückgedreht,
  `git diff --stat` zeigt nur die beabsichtigten 22 Zeilen in helpers.go.
- verify vorgaenger: sauber — Commit 7e932ec1 geprüft: Handler geht über `grpcReq`/den
  gRPC-Client (kein Bypass), keine neue Route, kein neuer `RequirePermission`-Key, kein `.proto`
  geändert, keine neue Tabelle.
- offen: `route_hr.go:298-299, 791-792, 969-970` (`HandleListLeaveRequests` und zwei weitere)
  reichen `start_date`/`end_date` ebenfalls ungeprüft an den gRPC-Request durch — dort aber als
  **optionale** Listenfilter, nicht als Pflichtfeld wie bei den CRM-Reports, also ein anderes
  Risikoprofil (leerer Filter statt 400 auf Garbage). Nicht in dieser Unit gefixt (Scope war
  ausdrücklich die drei Report-Handler); Kandidat für eine eigene Lauf-9-Unit, falls das
  gewünscht ist. Kein PR/Push, lokal committet.

## CI nach Lauf (2026-08-10 13:53)
- run: 31384710707
- sha: 668e4f0ef301d0dd418c3a57a7621bc8e5708dcf
- ergebnis: success
- commits: 2

## Iteration 3 — fix-einkauf-contract-call-cap — done — 2026-08-10 15:00
- commit: de885fc4
- gebaut: `PostgresRepository.CreateContractCall` ist jetzt transaktional und der einzige
  Schreiber von `used_value`: Contract-Zeile `FOR UPDATE` sperren → Status prüfen (nur `active`,
  sonst `ErrContractNotActive` mit Status im Text) → Restwert aus **frisch gerechnetem
  `SUM(amount)`** gegen `total_value` in **numeric** vergleichen (nicht gegen die gecachte
  `used_value`-Spalte, nicht in Go-float) → bei Überschreitung `ErrContractBudgetExceeded` mit
  Restwert in der Meldung → INSERT → `used_value`-Recompute, alles in derselben Transaktion.
  Der Service prüft nur noch die Betragsform; der Vorab-`GetFrameworkContract` ist raus (die
  Existenz liefert jetzt die Transaktion als `ErrContractNotFound`). `mapEinkaufError` bildet
  beide neuen Fehler auf `FailedPrecondition` ab → Gateway 409, in `openapi.yaml` beim POST
  `/api/v1/einkauf/contracts/{id}/calls` dokumentiert.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (einkauf 0 skipped, 73 PASS inkl. aller
  DB-Tests; server + gateway grün) | migration n.a. (keine Migration — der Cap sitzt im Code,
  ein CHECK könnte den Cross-Table-Vergleich nicht ausdrücken) | rls-smoke n.a. (keine Tabelle/
  Policy angefasst; die neue Transaktion erbt die Session-GUCs aus `PrepareConn`, die DB-Tests
  laufen unverändert unter `kmuhub_app`) | swagger-cli validate ok
- coverage: internal/einkauf 63,3 % -> 63,9 %
- mutations-probe: zwei Proben am Produktionscode, beide gefangen. (1) im SQL `$2::numeric <=`
  auf `>=` gedreht -> `TestFrameworkContract_CreateContractCall_EnforcesRemainingValue` und
  `..._AccumulatesUsedValue` rot ("call-off of 600.0000, remaining 1000.0000" statt Annahme),
  zurückgedreht. (2) `if contractStatus != string(ContractStatusActive)` auf `if false && …`
  -> `TestFrameworkContract_CreateContractCall_RejectsInactiveAndUnknown` rot (draft-Vertrag
  lieferte `<nil>` statt `ErrContractNotActive`), zurückgedreht. `git diff` danach exakt die
  beabsichtigten Zeilen (geprüft: kein `TRUE`/`false &&`-Rest im Diff). Erste Fassung von
  Probe 1 (`TRUE` statt der Vergleichsspalte) war unbrauchbar — sie nahm `$2` aus dem Statement
  und Postgres brach mit 42P18 ab, statt die Schranke wirklich zu öffnen; deshalb der
  Operator-Flip.
- verify vorgaenger: sauber — `668e4f0e` geprüft: `validateDateParam` + drei Report-Handler,
  kein gRPC-Bypass (die Handler laufen weiter über `client.Get*Report`), kein Stub, kein
  `.proto`, keine neue Route, kein neuer Permission-Key, keine Tabelle, Wire-Shape unverändert
  (400 über `response.Error` wie bei den Nachbarn).
- offen: **Eine Entscheidung gehört Luke.** `total_value` ist beim Anlegen optional und
  defaultet auf `0` (Migration 000207 Z. 14, `createFrameworkContractRequest.TotalValue` ist
  `omitempty`). Mit der strikten Variante ist ein Rahmenvertrag ohne eingetragenen Gesamtwert ab
  sofort **gar nicht mehr abrufbar** (409, "remaining 0.0000"). Das ist die wörtliche Umsetzung
  der Entscheidung vom 2026-08-10 und lässt nirgends Budget durch; die Alternative wäre
  „`total_value = 0` heißt unbegrenzt" — ein Einzeiler in `CreateContractCall`
  (`if totalValue != "0.0000" { … }`), aber eine implizite Sonderregel. Bewusst nicht selbst
  entschieden. — Weiter: `UpdateContractUsedValue` ist ganz raus (Interface, Postgres-Impl,
  Mock, Server-Stub), weil sie nach dem Umbau keinen Produktions-Aufrufer mehr hatte und genau
  der non-fatale Aufruf war, über den die Spalte nach unten driften konnte; der DB-Test dazu
  heißt jetzt `TestFrameworkContract_CreateContractCall_AccumulatesUsedValue` und beweist
  dieselben Summen über den neuen Pfad. Kein Push, lokal committet. Für das FE: 409 ist auf
  dieser Route neu.


## Iteration 4 — fix-plugin-createmanifest-missing-tenant-wrong-code — done — 2026-08-10 15:19
- commit: 1f476fda
- gebaut: `mapPluginError` (plugin_grpc.go) hat jetzt einen expliziten `errors.Is(err,
  middleware.ErrMissingTenantID)`-Fall -> `codes.InvalidArgument`, statt den von
  `Service.CreateManifest` per `fmt.Errorf("create manifest: %w", err)` gewrappten Sentinel
  ungefangen in den `default`-Zweig (`codes.Internal`) durchfallen zu lassen. Einziger Aufrufer
  von `middleware.GetTenantID` im Paket `internal/plugin` ist `Service.CreateManifest`
  (service.go:90) — keine weitere RPC im Paket reicht denselben Sentinel durch. Der
  dokumentierende Test in plugin_grpc_test.go ("missing tenant context maps to Internal, not
  InvalidArgument (documents current gap)") ist auf das korrekte Verhalten umgestellt
  ("... is rejected as InvalidArgument", erwartet jetzt `codes.InvalidArgument`), nicht
  gelöscht; sein alter Kommentar behauptete zusätzlich fälschlich, `mapPluginError` vergleiche
  mit `==` und verwies auf ein nicht existierendes "TestMapPluginError gap #2" — beides beim
  Verify-First nicht bestätigt, der Kommentar ist entsprechend korrigiert. `TestMapPluginError`
  um zwei Fälle ergänzt (roher Sentinel + gewrappt wie in CreateManifest).
- gate: build ok | vet ok | lint ok (0 issues) | test ok (0 skipped, DATABASE_URL gesetzt,
  kmuhub_app) | migration n.a. (keine Migration) | rls-smoke n.a. (keine Tabelle/Policy
  angefasst) | route n.a. (keine Route angefasst, TestOpenAPIRouteDrift nicht betroffen)
- coverage: internal/server 47,7 % -> 47,7 % (Fix ist ein einzelner neuer `case`, kein
  Coverage-Ziel dieser Unit)
- mutations-probe: neuen `case`-Zweig von `codes.InvalidArgument` auf `codes.Internal` gedreht
  -> `TestMapPluginError/missing_tenant_id`, `.../wrapped_missing_tenant_id_...` und
  `TestPluginCreateManifest/missing_tenant_context_is_rejected_as_InvalidArgument` alle drei
  rot, zurückgedreht, `git diff` zeigt exakt die eine beabsichtigte Zeile.
- verify vorgaenger: sauber — `de885fc4` geprüft: `mapEinkaufError` bildet beide neuen
  Sentinels über `errors.Is` ab (kein `==`-Vergleich), kein gRPC-Bypass (Repository-Methode
  läuft weiter über den bestehenden Service-Aufruf), kein Stub, kein `.proto` geändert, kein
  neuer `RequirePermission`-Key, keine neue Tabelle, keine neue Route (openapi.yaml ergänzt nur
  den 409-Code einer bestehenden Route), Wire-Shape unverändert.
- offen: Der Doc-Kommentar auf `middleware.GetTenantID` sagt explizit "Callers must respond
  with 401 on error — never substitute a default tenant" — der gesamte Rest des Repos
  (automation_grpc.go, calendar_grpc.go, `ruleTenant()` in plugin_grpc.go selbst, u. v. a.)
  bildet den fehlenden Tenant aber durchgängig auf `InvalidArgument` (400) ab, nicht auf
  `Unauthenticated`/401. Diese Unit folgt der real gelebten Konvention (und dem `done_when` der
  Unit), nicht dem Kommentar — der Kommentar ist vermutlich selbst veraltet. Nicht in dieser
  Unit korrigiert (Scope war ausdrücklich nur `CreateManifest`/`mapPluginError`); falls das
  gewünscht ist, wäre ein Repo-weiter 401-vs-400-Entscheid eine eigene Unit für Lauf 9.

## Iteration 5 — fix-automation-scope-filter-silent-default — done — 2026-08-10 15:26
- commit: (siehe unten, wird nach diesem Journal-Eintrag committet)
- gebaut: `parseAutomationScope` (route_automation.go) liefert jetzt `(automationv1.AutomationScope,
  bool)` statt eines nackten Enums. Leer bleibt beim historischen Default `SCOPE_PERSONAL` (kein
  Verhaltenswechsel fuer Aufrufer, die das Feld nicht schicken), ein NICHT-leerer unbekannter Wert
  liefert `ok=false` — Variante (a) aus der Unit-Notiz, vertragserhaltend. Alle drei Call-Sites
  (`HandleCreateAutomation`, `HandleListAutomations`-Filter, `HandleUpdateAutomation`) pruefen `ok`
  und schreiben bei `false` ein 400 mit den erlaubten Werten in der Meldung, statt den Request mit
  einem still auf `personal` verengten Scope weiterzureichen. `HandleUpdateAutomation` war der
  eigentlich scharfe Fund: es dekodiert den Body ueber ein rohes `json.NewDecoder` ohne
  `decodeAndValidate`/`oneof`-Tag, ein Tippfehler haette dort einen bestehenden Automation-Scope
  (z. B. `organization`) still auf `personal` zurueckgestuft. `HandleCreateAutomation` hat den
  `oneof=personal team organization`-Tag bereits vorher gehabt (`TestHandleCreateAutomation_
  InvalidScope` bestand schon vor dieser Unit) — der neue `ok`-Check dort ist Verteidigung an
  derselben gemeinsamen Stelle, nicht der eigentliche Fund.
  `parseExecutionStatus` direkt darunter geprueft (Notiz-Auflage): KEIN gleiches Muster. Ein
  unbekannter Status faellt auf `EXECUTION_STATUS_UNSPECIFIED`, und der Server
  (`automation_grpc.go:287`) behandelt `UNSPECIFIED` explizit als "kein Filter gesetzt" (skip),
  nicht als einen spezifischen falschen Status wie bei Scope. Ein Tippfehler zeigt dort also alle
  Ausfuehrungen statt einer plausibel falschen Teilmenge — anderes Risikoprofil, nicht angefasst.
  Tests: `TestParseAutomationScope` um die `ok`-Dimension erweitert (inkl. `organisation`-Tippfehler
  als eigener Fall), `TestHandleListAutomations_FilterCombinations` verliert den
  `scope_unknown_defaults_personal`-Fall (Verhalten geaendert), dafuer neuer
  `TestHandleListAutomations_UnknownScopeRejected` (400). Neuer
  `TestHandleUpdateAutomation_InvalidScope` (400) fuer den zuvor ungeschuetzten Pfad.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (0 skipped, DATABASE_URL gesetzt,
  kmuhub_app) | migration n.a. (keine Migration) | rls-smoke n.a. (keine Tabelle/Policy angefasst) |
  route n.a. (keine neue Route, nur bestehende Handler geaendert)
- coverage: internal/gateway 34,9 % -> 35,0 %
- mutations-probe: `return automationv1.AutomationScope_SCOPE_PERSONAL, false` im `default`-Zweig
  auf `, true` gedreht -> `TestParseAutomationScope/unknown-garbage`,
  `TestParseAutomationScope/organisation`, `TestHandleListAutomations_UnknownScopeRejected` und
  `TestHandleUpdateAutomation_InvalidScope` alle vier rot (Scope wurde wieder still auf
  `SCOPE_PERSONAL` mit `ok=true` verengt, Requests erreichten 503 statt 400), zurueckgedreht,
  `git diff` zeigt exakt die urspruenglich beabsichtigten Zeilen.
- verify vorgaenger: sauber — `1f476fda` geprueft: neuer `errors.Is`-Case in `mapPluginError`
  (kein `==`-Vergleich), kein gRPC-Bypass, kein Stub, kein `.proto`, kein neuer
  `RequirePermission`-Key, keine Tabelle, keine Route, dokumentierender Test auf korrektes
  Verhalten umgestellt statt geloescht.
- offen: `route_automation.go` hat an mehreren Stellen (Zeile ~590, ~621) `interface{}` statt `any`
  im bestehenden Code — golangci-lint meldet das als reinen Stil-Hinweis (kein Fehler, `0 issues`
  im Lint-Gate), nicht in dieser Unit angefasst, da ausserhalb ihres Scopes. Laufkontext-Block
  (Iterationsnummer/Zeitstempel) war in diesem Prompt nicht sichtbar mitgeliefert — Nummer aus der
  letzten Journal-Ueberschrift (Iteration 4) fortgezaehlt, Zeitstempel per `date` auf dem
  Loop-Rechner ermittelt statt aus dem Block uebernommen; falls das vom Treiber-Mechanismus
  abweicht, bitte pruefen.

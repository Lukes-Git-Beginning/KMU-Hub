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

## Iteration 6 — fix-gateway-id-validation-consistency — done — 2026-08-10 15:28
- commit: (siehe unten, wird nach diesem Journal-Eintrag committet)
- gebaut: Alle elf `chi.URLParam(r, "id")`-Stellen in `route_biz_billing.go`
  (HandleGetCreditNote, HandleSendCreditNote, HandleGenerateCreditNotePDF, HandleRecordPayment,
  HandleListPayments, HandleDeletePayment, HandleSendDunning, HandleGenerateDunningPDF,
  HandleLockInvoice, HandleUpdateDunningStatus, HandleSendDunningNotice) laufen jetzt über
  `validateUUIDParam` statt den rohen chi-Wert direkt ins Proto zu reichen — eine
  strukturell/syntaktisch unbrauchbare ID liefert jetzt 400 an der Grenze statt eines
  Downstream-Fehlers vom (unerreichbaren Dummy-)Backend. Ungenutzter `chi`-Import in der Datei
  entfernt. Ebenso `HandleDeleteTask` und `HandleMoveTask` in `route_work_tasks.go` (Task-ID ist
  laut `moveTaskRequest.StatusID` (`validate:"required,uuid"`) im selben Handler-Set durchgängig
  UUID). Alle elf Handler in route_biz_billing.go geprüft: jede der IDs ist eine
  Entity-Primary-Key-UUID (Credit-Note-, Payment-, Dunning-, Invoice-ID), keine Slugs/Composite-Keys
  — keine Ausnahme nötig.
  Test-Fallout: vier Bestandstests (`TestHandleRecordPayment_InvalidJSON/MissingAmount/
  InvalidPaymentDate/InvalidMethod` in route_biz_test.go, zwei weitere in
  route_biz_billing_test.go, zwei in route_biz_billing_test.go für UpdateDunningStatus) riefen die
  Handler ohne `withChiURLParam` auf und hätten die Body-Validierung jetzt nie erreicht (leere ID
  bricht zuerst ab) — allen eine gültige `withChiURLParam(req, "id", "550e8400-…")` ergänzt.
  `TestHandleDeleteTask_InvalidIDReachesRPCNotLocalValidation` in route_work_tasks_test.go
  dokumentierte explizit die alte Lücke ("no local UUID check... reaches RPC, 503") — auf
  `TestHandleDeleteTask_InvalidUUID` (400 + "invalid id") umgestellt, der erklärende
  Kommentarblock darüber (der die Lücke als "stilistische Inkonsistenz, keine Sicherheitslücke"
  beschrieb) entfernt, da er die neue Realität nicht mehr beschreibt. Neuer
  `TestBizBillingRoutes_InvalidUUID` (Table-Test, alle elf Handler, ein Aufruf pro Handler) und
  `TestHandleMoveTask_InvalidUUID` ergänzt.
  Gateway-weite Bestandsaufnahme (Auflage aus der Unit-Notiz, Schwelle "~20"): `grep -rn
  'chi\.URLParam(r, "id")' backend/internal/gateway/*.go` (ohne Tests) findet **174** Rohstellen
  über **26** Dateien — deutlich über der Schwelle, deshalb bewusst NICHT alles in dieser Unit
  umgebaut (Budget-Deckel-Risiko). Nach dieser Unit bleiben **161** Rohstellen über 24 Dateien
  offen: route_auth.go, route_biz_bank_accounts.go, route_biz_bank_transactions.go,
  route_biz_banking.go, route_biz_einvoice.go, route_biz_expenses.go,
  route_biz_gobd_archive.go, route_biz_invoices.go, route_biz_quotes.go,
  route_biz_recurring.go, route_biz_transactions.go, route_caldav.go, route_calendar.go,
  route_crm_ext.go, route_customization.go, route_document.go, route_email.go, route_hr.go,
  route_hr_changerequest.go, route_lexware.go, route_security.go, route_work_labels.go,
  route_work_projects.go, route_work_time.go (route_auth.go und route_work_tasks.go haben
  daneben bereits validierte Stellen — die 161 zählen nur die verbleibenden rohen). Kandidat für
  eine eigene Folge-Unit (oder mehrere, nach Datei/Domäne gebündelt) in Lauf 9 — nicht selbst
  angelegt, siehe `offen:` unten.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (0 skipped, DATABASE_URL gesetzt,
  kmuhub_app) | migration n.a. (keine Migration) | rls-smoke n.a. (keine Tabelle/Policy
  angefasst) | route n.a. (keine neue Route, TestOpenAPIRouteDrift +
  TestOpenAPIRouteDriftParserSanity grün)
- coverage: internal/gateway 34,9 % -> 35,1 %
- mutations-probe: `if !ok { return }` in `HandleDeleteTask` (route_work_tasks.go, neuer Zweig)
  auf `if ok { return }` gedreht -> `TestHandleDeleteTask_InvalidUUID` rot (ServiceUnavailable-
  und übrige Tasks-Tests blieben grün), zurückgedreht, `git diff --stat` auf die Datei zeigt exakt
  wieder 8 Insertions/2 Deletions (die ursprünglich beabsichtigte Änderung, kein Rest).
- verify vorgaenger: sauber — `be847c52` geprüft: `parseAutomationScope` liefert `(scope, bool)`,
  alle drei Call-Sites (Create/List/Update) prüfen `ok` und antworten 400 bei unbekanntem,
  nicht-leerem Wert; leerer Wert bleibt beim historischen `SCOPE_PERSONAL`-Default (kein
  Verhaltensbruch für bestehende Aufrufer); kein gRPC-Bypass, kein Stub, kein `.proto` geändert,
  kein neuer `RequirePermission`-Key, keine Tabelle, keine neue Route (nur bestehende Handler
  geändert), `parseExecutionStatus` wie in der Notiz gefordert geprüft und im Commit-Text
  begründet, warum es NICHT denselben Fix braucht (UNSPECIFIED wird vom Server explizit als
  "kein Filter" behandelt, kein plausibel-falscher Zustand wie bei Scope).
- offen: Laufkontext-Block (Iterationsnummer/Zeitstempel) war in diesem Prompt wieder nicht
  sichtbar mitgeliefert (wie schon in Iteration 5 vermerkt) — Nummer aus der letzten
  Journal-Überschrift (Iteration 5) fortgezählt, Zeitstempel per `date` auf dem Loop-Rechner
  ermittelt. Falls der Treiber-Mechanismus das abweichend behandelt, bitte prüfen.
  Die 161 verbleibenden rohen `chi.URLParam(r, "id")`-Stellen aus 24 Dateien (Liste oben unter
  "gebaut") sind NICHT als Backlog-Unit angelegt — analog zum Vorgehen in Iteration 5 nur hier im
  Journal dokumentiert, damit Luke entscheidet, ob/wie das für Lauf 9 gebündelt wird (eine Unit
  je Datei wäre zu granular, ein Rundumschlag zu groß für eine Iteration).
  `.planning/backend-block/loop/run-loop.ps1` hat im Arbeitsverzeichnis einen unstaged Diff
  (neuer `-StartNotBefore`-Parameter), der schon zu Beginn dieser Iteration vorhanden war und zu
  keiner Backlog-Unit gehört — nicht angefasst, nicht committet (nicht meine Datei für diese
  Unit).

## Iteration 7 — fix-email-sync-imap-connect-no-timeout — done — 2026-08-10 15:52
- commit: (siehe unten, wird nach diesem Journal-Eintrag committet)
- gebaut: `IMAPClient.Connect` (imap_client.go) macht jetzt den TCP-Dial (und beim direkten
  TLS-Pfad Port 993 den TLS-Handshake) selbst über `net.Dialer{Timeout: imapDialTimeout}` bzw.
  `tls.DialWithDialer` statt `imapclient.DialTLS`/`DialStartTLS(addr, nil)`. Neuer Helper
  `newIMAPProtocolClient(conn, direct)`: fürs direkte TLS reicht `imapclient.New(conn, nil)`
  (blockiert laut Quellcode nicht — spawnt nur die interne Read-Goroutine), für STARTTLS läuft
  `imapclient.NewStartTLS(conn, nil)` in einer eigenen Goroutine, geract gegen
  `time.After(imapHandshakeDeadline)`; bei Timeout schließt der Aufrufer `conn`, was den
  blockierten Read in der verwaisten Goroutine entriegelt statt sie fuer immer haengen zu lassen.
  `imapDialTimeout = 10s`, `imapHandshakeDeadline = 30s` — dieselben Werte wie
  `email/send`s `smtpDialTimeout`/`smtpExchangeDeadline` (package-lokale Vars, nicht importiert —
  `internal/email/systemmail` hat exakt dasselbe Muster kopiert, das ist hier Konvention, kein
  Versehen). `addr` jetzt über `net.JoinHostPort` statt `fmt.Sprintf("%s:%d", ...)` (Linter
  `hostport`, IPv6-Adressen brechen sonst).
  **Wichtiger Fund gegen die Backlog-Notiz:** die Vermutung "go-imap v2 hat gar keinen
  Read-Timeout" stimmt nicht ganz — es gibt einen internen `respReadTimeout` (30s, hartkodiert),
  der aber bei jedem Leseversuch neu gesetzt wird und daher gegen einen Server, der komplett
  schweigt, nie wirklich ablaeuft (empirisch verifiziert: `imapclient.DialStartTLS(addr, nil)`
  gegen einen Fake-Server, der annimmt und dann für immer schweigt, kehrte auch nach >35s nicht
  zurück). Ein zusätzliches `conn.SetDeadline(...)` von außen half NICHT — es wird vom
  internen Read-Loop sofort wieder überschrieben (das war mein erster, verworfener Versuch,
  siehe unten). Nur das Race mit eigenem Timer + `conn.Close()` bei Ablauf entriegelt den
  blockierten Aufruf zuverlässig. `worker.go:148` ruft `client.Connect(...)` ohne eigenen
  Kontext/Deadline auf (wie in der Notiz zu prüfen aufgetragen) — der Fix hier ist die einzige
  Absicherung, es gibt keine doppelte.
  Neuer Test `imap_timeout_test.go` (`TestConnect_HandshakeDeadline`), nach dem Muster von
  `smtp_timeout_test.go`: TCP-Fake-Server nimmt an und schweigt für immer, Timeouts für den Test
  auf 2s/200ms geschrumpft, `Connect` muss `ErrIMAPConnectionLost` liefern statt zu haengen.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (0 skipped, DATABASE_URL gesetzt,
  kmuhub_app) | migration n.a. (keine Migration) | rls-smoke n.a. (keine Tabelle/Policy
  angefasst) | route n.a. (keine Route angefasst, `go test ./internal/gateway/` daher nicht
  Pflicht und nicht gelaufen)
- coverage: internal/email/sync 10,9 % -> 16,0 %
- mutations-probe: `time.After(imapHandshakeDeadline)` in `newIMAPProtocolClient` auf
  `time.After(imapHandshakeDeadline * 100)` gedreht (simuliert "Timeout wirkt nicht") ->
  `TestConnect_HandshakeDeadline` rot (Connect kehrte nicht innerhalb der 3s-Testgrenze zurück),
  zurückgedreht, `gofmt`-normalisierter Diff zeigt exakt die urspruenglich beabsichtigte Aenderung
  (73 Insertions/5 Deletions in imap_client.go, reiner Neubau in imap_timeout_test.go).
- verify vorgaenger: sauber — `e545d33c` geprüft: alle 13 Handler nutzen `validateUUIDParam`
  statt rohem `chi.URLParam`, kein gRPC-Bypass, kein Stub, kein `.proto`, kein neuer
  `RequirePermission`-Key, keine Tabelle, keine neue Route (nur bestehende Handler geändert,
  `TestOpenAPIRouteDrift` unberührt), Testfallout (fehlende `withChiURLParam`-IDs in
  Bestandstests) korrekt nachgezogen statt Assertions geschwächt.
- offen: Der Fix deckt laut Scope nur `Connect()` ab. `Login()`/`Select()`/`Fetch()` usw. hängen
  weiterhin am selben potenziell nie ablaufenden internen `respReadTimeout` — ein Postfach-Server,
  der nach erfolgreichem Connect+Login beim ersten `SELECT` verstummt, würde denselben
  unbegrenzten Hang zeigen. Nicht in dieser Unit behoben (Scope war ausdrücklich nur Connect),
  aber real genug, um als eigene Folge-Unit für Lauf 9 vorzumerken (`newIMAPProtocolClient`s
  Race-Pattern ließe sich grundsätzlich auf die anderen blockierenden Calls übertragen, wäre aber
  eine größere Änderung — jeder Call bräuchte eine eigene Goroutine+Race oder einen zentralen
  Wrapper). Für Port 993 (direktes TLS) bleibt das Restrisiko kleiner, da `imapclient.New`
  nicht blockiert und der TLS-Handshake selbst über `net.Dialer.Timeout` bereits gebunden ist
  (per `crypto/tls`-Quellcode verifiziert: `tls.DialWithDialer` spannt einen `context.WithTimeout`
  über Dial UND Handshake).

## Iteration 8 — hr-salary-document-category — done — 2026-08-10 15:58
- commit: (siehe unten, wird nach diesem Journal-Eintrag committet)
- gebaut: Migration `000310_hr_salary_document_category` seedet die System-Dokumentenkategorie
  `gehaltsabrechnung` (`hr_document_categories`, `tenant_id` = Zero-UUID, `visibility='employee'`,
  `is_system=TRUE`, `sort_order=15`) nach exakt dem Muster von 000046/000294 —
  `ON CONFLICT (tenant_id, key) DO NOTHING`, idempotent gegen `uq_hr_doc_category_key`. `.down.sql`
  entfernt genau diese eine Zeile (`DELETE ... WHERE tenant_id = zero AND key = 'gehaltsabrechnung'`),
  nichts sonst. Kein Payroll-Datenmodell, kein Brutto/Netto — Gehaltsabrechnungen laufen als PDF
  über die bestehende `hr_employee_documents`-Infrastruktur, wie von Luke am 2026-08-10 entschieden.
  Neuer Test `salary_document_category_test.go` (Package `hr`, wie `personnel_documents_read_test.go`
  danebenliegend): `TestSalaryDocumentCategory_SeedRow_DB` liest die reale Seed-Zeile und prüft
  `visibility='employee'`/`is_system=true`; `TestSalaryDocumentCategory_VisibleToEmployee_NotHROnly_DB`
  seedet ein Dokument unter der echten `gehaltsabrechnung`-Kategorie plus eines unter einer ad-hoc
  `hr_only`-Kategorie und belegt über `employee.PostgresEmployeeDocRepo.ListByTenant`: der Mitarbeiter
  sieht seine eigene Gehaltsabrechnung, nicht das hr_only-Dokument; ein Kollege ohne HR-Rolle sieht
  keines von beiden; `hr_admin` sieht beide. Kein neuer RLS-Test nötig — die Sichtbarkeitslogik selbst
  ist bereits durch `hr_document_access` (Migration 000127/000128) und `TestHRRoleBased_DocumentAccess_DB`
  bewiesen, hier wird nur die neue Zeile gegen genau diese Logik durchgespielt.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (0 skipped, DATABASE_URL gesetzt, kmuhub_app,
  `internal/biz/hr/...` + `internal/gateway/` grün) | migration ok (000310 angewendet, Kopf 310, dirty=false) |
  rls-smoke n.a. (keine neue Tabelle/Policy, nur eine Seed-Zeile in einer bereits RLS-gesicherten Tabelle) |
  route n.a. (keine Route angefasst)
- coverage: internal/biz/hr 72,8 % -> 72,8 % (aggregiert über `internal/biz/hr/...`, siehe `offen:` —
  das Zielpaket `internal/biz/hr` selbst hat keine Nicht-Test-Datei/keine Statements, der neue Test
  deckt bereits getestete Zeilen in `internal/biz/hr/employee` erneut ab statt neue aufzudecken)
- mutations-probe: `visibility` der Seed-Zeile in der laufenden DB per `UPDATE hr_document_categories
  SET visibility='hr_only' WHERE key='gehaltsabrechnung' ...` gedreht (Migration selbst ist idempotent
  und würde eine SQL-Änderung nicht neu anwenden, deshalb direkt am DB-Zustand simuliert) ->
  `TestSalaryDocumentCategory_SeedRow_DB` und der Subtest
  `.../the_employee_sees_their_own_salary_statement` beide rot, zurückgedreht (`UPDATE ... SET
  visibility='employee' ...`), `go test ./internal/biz/hr/...` wieder vollständig grün.
- verify vorgaenger: sauber — `d5b71104` geprüft: `Connect` dialt jetzt über `net.Dialer{Timeout:
  imapDialTimeout}`/`tls.DialWithDialer`, STARTTLS-Handshake läuft in eigener Goroutine gegen
  `imapHandshakeDeadline` gerannt, kein gRPC-Bypass, kein Stub, kein `.proto`, keine Tabelle, keine
  Route, kein neuer `RequirePermission`-Key, dokumentierter Restrisiko-Hinweis (Login/Select/Fetch
  bleiben ungebunden) korrekt unter `offen:` vermerkt statt verschwiegen.
- offen: `coverage_start: "internal/biz/hr 72,8 %"` in der Unit ist nur über `./internal/biz/hr/...`
  (den ganzen Unterbaum) reproduzierbar, nicht über das nackte `./internal/biz/hr/`-Paket — das hat
  keine Nicht-Test-Datei und meldet `[no statements]`. Damit weicht diese Unit von der in
  GATE-COMMANDS.md verbindlich gemachten Regel "genau ein Paket, ohne `...`" begründet ab; für Pakete
  mit Unterpaketen und ohne eigenen Code gehört das dort ergänzt, ist hier nicht nachgezogen (nicht
  Scope dieser Unit).
  `hr-salary-self-service-route` (deps: [hr-salary-document-category]) ist jetzt entsperrt und die
  nächste Unit im Backlog — ihre eigene Vorpruefung entscheidet, ob sie ueberhaupt eine neue Route
  braucht.
  `.planning/backend-block/loop/run-loop.ps1` hat weiterhin denselben unstaged `-StartNotBefore`-Diff
  wie in Iteration 6/7 vermerkt — nicht meine Datei, nicht angefasst, nicht committet.
  Laufkontext-Block (Iterationsnummer/Zeitstempel) war in diesem Prompt nicht sichtbar mitgeliefert —
  Nummer aus der letzten Journal-Ueberschrift (Iteration 7) fortgezaehlt, Zeitstempel per `date` auf
  dem Loop-Rechner ermittelt, wie in den vorigen Iterationen.

## Iteration 9 — hr-salary-self-service-route — done — 2026-08-10 16:00
- commit: ef8dec17
- gebaut: VORPRUEFUNG ERGAB "NEIN" (Belege unter `offen:`), deshalb Schritt 2 der Unit gebaut:
  neue Route `GET /api/v1/hr/employees/me/documents` (`route_hr.go`, registriert VOR
  `/{id}/documents`, Guard ist der bestehende `hrDocumentCategoriesGuard` =
  `RequirePermissionAny(hr:read, team:documents:view)` — kein neuer Permission-Key, keine
  Seed-Migration). Handler `HandleListOwnDocuments` nimmt die Employee-ID ausschliesslich aus
  dem JWT (`middleware.GetUserID`), nie aus dem Request: `hr_employee_documents.employee_id`
  referenziert `users(id)` (Migration 000046:178), die User-ID IST also die Employee-ID, und
  ein Aufrufer kann ueber diese Route keine fremde Akte adressieren. Kein Proto-Change und
  kein Service-Change noetig — die RPC `ListEmployeeDocuments(employee_id)` existiert bereits
  und geht regulaer ueber den gRPC-Client. Optionaler `?category=<key>`-Filter ueber die neue
  Hilfsfunktion `filterDocumentsByCategory` (mit `lean:`-Marker: filtert die Antwort, weil die
  Liste eines Mitarbeiters vollstaendig und unpaginiert ist; Upgrade-Trigger = sobald diese
  Liste Pagination bekommt, gehoert der Key in die RPC). Antwort gewrappt als
  `{documents: [...]}` wie `/hr/personnel-documents`, nicht als nacktes Array wie
  `/{id}/documents` — bewusst, weil das FE ohnehin neu geschrieben wird und die
  Architektur-Regel gewrappte Listen fordert. Spec-Eintrag in `backend/api/openapi.yaml` im
  selben Commit, inkl. 401/403/503. Neue Testdatei `route_hr_self_documents_test.go` mit
  6 Tests: Pfad-Match mit `team:documents:view` (503 = Handler erreicht, nicht 404),
  Legacy-Key `hr:read` erreicht die Route ebenfalls, ohne Key 403, fehlender User-Kontext 401,
  Tabellentest fuer den Kategorie-Filter (6 Faelle inkl. Whitespace-Trim, Teilstring-Nichttreffer,
  unbekannter Key) und ein Wire-Shape-Test (leeres Ergebnis ist `[]`, nicht `nil`/`null`).
  DAZU der wichtigste Test: `TestListEmployeeDocuments_StillAdminOnly` haelt fest, dass die
  neue Route die ALTE nicht aufgeweicht hat — `team:documents:view` allein bekommt auf einer
  fremden `{id}` weiterhin 403.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (`go test -count=1 ./internal/gateway/`
  gruen, 2164 PASS / **0 SKIP** bei gesetzter `DATABASE_URL` als `kmuhub_app`) |
  `TestOpenAPIRouteDrift` gruen (im Paket enthalten) | `swagger-cli validate` gruen (Pflicht nach
  jeder `openapi.yaml`-Aenderung, nicht nur der Drift-Test) | migration n.a. (keine) |
  rls-smoke n.a. (keine Tabelle/Policy angefasst)
- coverage: internal/gateway 34,9 % -> 35,1 % (`go tool cover -func`, genau das Paket
  `./internal/gateway/`)
- mutations-probe: drei Proben, alle rot, alle zurueckgedreht, Diff danach restfrei
  (`git diff` gegen den finalen Tree geprueft):
  (1) Guard der neuen Route von `hrDocumentCategoriesGuard` auf
  `RequirePermission("hr","read")` verengt -> `TestHandleListOwnDocuments_MatchesFrontendPath`
  rot (403 statt 503). Das ist die Probe, die zaehlt: sie beweist, dass der Test genau den
  Fehler faengt, der die Route fuer jeden Nicht-Admin nutzlos machen wuerde.
  (2) `if userID == ""`-Guard entfernt -> `TestHandleListOwnDocuments_MissingUserContext` rot
  (503 statt 401).
  (3) `d.GetCategoryKey() == key` durch `strings.HasPrefix(...)` ersetzt ->
  `TestFilterDocumentsByCategory/partial_key_does_not_match` rot (2 statt 0 Dokumente).
- verify vorgaenger: sauber — `6b42ef16` geprueft: Migration 000310 legt keine Tabelle an,
  sondern seedet eine Zeile in der bereits RLS-gesicherten `hr_document_categories`, idempotent
  per `ON CONFLICT (tenant_id, key) DO NOTHING`; `.down.sql` loescht exakt diese eine Zeile
  (tenant_id + key im WHERE), nichts sonst. Kein gRPC-Bypass, kein Stub, kein `.proto`, keine
  Route, kein neuer `RequirePermission`-Key, keine bereits ausgerollte Migration angefasst.
- offen: VORPRUEFUNGS-BELEGE (Schritt 1 der Unit, das war die eigentliche Frage):
  (a) `GET /api/v1/hr/employees/{id}/documents` ist mit `RequirePermission("hr","read")`
      geguardet (`route_hr.go:168`). `hr:read` wird in Migration 000129 **ausschliesslich an
      `admin`** vergeben (Block ab Z. 49 `WHERE r.name = 'admin'`, Key in Z. 63) und taucht in
      Migration 000256 gar nicht auf. Ein `member` kann seine eigene Akte ueber diese Route
      also NICHT lesen — er bekommt 403, unabhaengig von der employee_id. Antwort auf Schritt 1
      ist damit klar NEIN.
  (b) `GET /api/v1/hr/personnel-documents` waere fuer einen `member` zwar erreichbar
      (`team:documents:view` auf Scope `own`, Migration 000256 Z. 465) und die RLS-Policy
      `hr_document_access` (000128) zeigt ihm dort tatsaechlich nur seine eigenen
      employee-sichtbaren Dokumente. Als Self-Service-Quelle ist die Route trotzdem falsch:
      dieselbe Policy gibt einem `manager` alle Dokumente der Stufen `manager` UND `employee`
      **aller** Mitarbeiter — ein Manager, der seine eigene Self-Service-Ansicht oeffnet,
      bekaeme also die Gehaltsabrechnungen seiner Kollegen mitgeliefert und muesste im FE
      wieder herausgefiltert werden. Serverseitig auf die eigene employee_id einzuschraenken
      ist die einzige Variante, bei der fremde Gehaltsdaten den Prozess gar nicht erst
      verlassen.
  FRONTEND-FOLGEARBEIT ist als `fe-salary-statements-endpoint` in `BACKLOG-PARKED.yml`
  eingetragen (Ziel: `.planning/nico-block/`): `SelfServiceView.tsx:130-141` ruft weiterhin
  `/me/salary-statements` (existiert nicht, nur MSW-Mock) und muss auf
  `/me/documents?category=gehaltsabrechnung` mit `{documents: [...]}` umgestellt werden; der
  FE-Typ `SalaryStatement` (Brutto/Netto) hat keine Backend-Entsprechung und wird durch die
  Dokument-Felder ersetzt. Bis dahin ist die neue Route im Produkt unbenutzt — sie ist
  getestet und spezifiziert, aber ohne den FE-Teil sieht der Mitarbeiter nichts.
  NICHT VERIFIZIERBAR IM UNIT-TEST: dass der HR-Service die Rollen des Aufrufers ueber die
  gRPC-Grenze in die `app.user_roles`-GUC bekommt (`database/postgres.go:74-85` setzt sie aus
  `middleware.GetUserRoles(ctx)`). Fuer diese Route ist das unkritisch — sie filtert per
  `WHERE d.employee_id = $1` auf die eigene ID, RLS ist hier zweite Verteidigungslinie, nicht
  die erste. Fuer `/hr/personnel-documents` ist es dagegen die einzige Grenze; ein
  End-to-End-Beleg dafuer fehlt im Repo weiterhin und waere eine eigene Unit wert.
  `.planning/backend-block/loop/run-loop.ps1` hat weiterhin denselben unstaged
  `-StartNotBefore`-Diff wie in Iteration 6-8 vermerkt — nicht meine Datei, nicht angefasst.
  Laufkontext-Block (Iterationsnummer/Zeitstempel) war auch in diesem Prompt nicht sichtbar
  mitgeliefert — Nummer aus der letzten Journal-Ueberschrift (Iteration 8) fortgezaehlt,
  Zeitstempel per `date` auf dem Loop-Rechner ermittelt.
- nachtrag: Beim Eintragen des Commit-Hashs hat ein `sed`-Ersetzen ohne Zeilenbegrenzung den
  Platzhalter `- commit: (siehe unten, ...)` in FUENF aelteren Eintraegen (Zeilen 50, 194, 241,
  311, 367) mitersetzt und ihnen faelschlich `ef8dec17` gegeben. Sofort zurueckgedreht
  (`git diff HEAD~1 -- JOURNAL.md` zeigt jetzt nur noch die eine beabsichtigte Zeile 418), der
  fehlerhafte Zwischenstand steckt in Commit `861d9c6c` und ist mit dem Folge-Commit
  vollstaendig korrigiert — kein Nachziehen noetig. LEHRE fuer folgende Iterationen: der
  Platzhaltertext ist NICHT eindeutig, mehrere frueherere Eintraege tragen ihn bis heute (sie
  haben ihren Hash nie nachgetragen). Den Hash mit dem Edit-Tool oder einer zeilengebundenen
  Ersetzung eintragen, nie mit einem globalen `sed`.

## Iteration 10 — b-cov-server-crm-fields-tags-contacts — done — 2026-08-10 16:26
- commit: 1fde576f
- gebaut: neue Datei `backend/internal/server/crm_grpc_fields_tags_contacts_test.go` mit
  Server-Package-Kopien der Repository-Stubs fuer `customfield`, `tag`, `contact`
  (`stubCustomFieldRepo`, `stubTagRepo`, `stubContactRepo` — die echten `MockRepository`-Typen
  liegen in den `_test.go`-Dateien der jeweiligen internal-Pakete und sind von hier aus nicht
  importierbar, gleiches Muster wie `stubFormulareRepo` in `formulare_grpc_test.go`) plus
  `newTestCRMServer()`/`newCRMServerWith<X>Repo()`-Konstruktoren. 51 neue Tests decken alle 17
  genannten Methoden ab: CustomFields (Create/Get/List/Update/Delete), Tags
  (Create/Get/List/Update/Delete, inkl. `DeleteTag_InUse`) und Contacts
  (Create/Get/List/Update/Delete/AddContactTags/RemoveContactTags, inkl. `..._InUse` und
  `AddContactTags_UnknownTag`). Jede Methode hat mindestens einen Validierungs- oder
  Happy-Path-Test UND mindestens einen Fehlerpfad (ungueltige UUID, fehlender Tenant,
  Downstream-Fehler wie NotFound/FailedPrecondition). `mapCRMError` selbst blieb unangetastet
  (laut Scope bereits vollstaendig getestet).
- gate: build ok | vet ok | lint ok (0 issues) | test ok (`go test -count=1 ./internal/server/`
  gruen, 1006 PASS / **0 SKIP** bei gesetzter `DATABASE_URL` als `kmuhub_app`; zusaetzlich
  `./internal/server/...` inkl. `response`-Unterpaket gruen) | migration n.a. (keine) |
  rls-smoke n.a. (keine Tabelle/Policy angefasst, reine gRPC-Server-Schicht mit In-Memory-Stubs)
- coverage: internal/server 47,7 % -> 48,8 % (`go tool cover -func=/tmp/cov.out | tail -1`,
  Paket `./internal/server/` exakt wie in GATE-COMMANDS.md)
- mutations-probe: drei Proben, alle rot, alle zurueckgedreht, `git diff` gegen den finalen
  Baum restfrei:
  (1) `backend/internal/server/crm_grpc.go`, `CreateCustomField`: den `uuid.Parse(req.CreatedBy)`-
  Fehlerpfad auf `_` verkuerzt (Check komplett entfernt) -> `TestCreateCustomField_InvalidCreatedBy`
  wurde NICHT rot, weil der Test urspruenglich ein leeres `EntityType` mitschickte und die
  Service-eigene `ErrInvalidEntityType`-Pruefung denselben `codes.InvalidArgument` zufaellig
  reproduzierte. Test korrigiert: jetzt ueber einen Repo-gestuetzten Server mit vollstaendig
  gueltigen Uebrigen Feldern, damit ausschliesslich der `created_by`-Pfad getroffen wird. Danach
  bei derselben Mutation tatsaechlich rot (Feld wurde erstellt statt `InvalidArgument`),
  zurueckgedreht, gruen. Das ist die Probe, die zaehlt: sie hat eine echte Testluecke in der
  ersten Fassung gefunden, nicht nur eine Zeile im Produktcode.
  (2) `backend/internal/crm/tag/service.go`, `Delete`: `if inUse` zu `if false && inUse`
  -> `TestDeleteTag_InUse` rot (Delete erfolgreich statt `FailedPrecondition`), zurueckgedreht.
  (3) `backend/internal/crm/contact/service.go`, `AddTags`: `if !exists` zu `if false && !exists`
  -> `TestAddContactTags_UnknownTag` rot (kein Fehler statt `NotFound`), zurueckgedreht.
- verify vorgaenger: sauber — `ef8dec17` geprueft gegen alle acht Fehlerklassen: Handler geht
  ueber `client.ListEmployeeDocuments` (gRPC-Client, kein Service-Bypass), keine
  `codes.Unimplemented`/Stub-Rueckgabe, kein `.proto` angefasst, kein neuer
  `RequirePermission`-Key (additive Wiederverwendung von `hrDocumentCategoriesGuard`), keine
  neue Tabelle, Wire-Shape gewrappt (`{documents: [...]}`), `backend/api/openapi.yaml` im
  selben Commit ergaenzt, kein Alt-Guard ersetzt. Details siehe `openapi.yaml`-Diff im Commit.
- offen: ECHTER BEFUND, NICHT GEFIXT (Coverage-Units bauen laut Backlog-Kopf keine
  Verhaltensaenderungen): `ListContacts`, `ListCustomFields` und `ListTags` in
  `backend/internal/server/crm_grpc.go` lassen `var infos []*crmv1.<X>Info` bei einem leeren
  Ergebnis als `nil`-Slice stehen statt mit `make([]*T, 0, len(...))` vorzubelegen — exakt die
  Wire-Shape-Klasse, die fuer `document_grpc.go`s `toProtoFile` in Lauf 7 (Commit `c3f0c46f`)
  bereits als Bug gefunden UND direkt gefixt wurde (2-Zeilen-Diff). Fuer `ListContacts` liegt
  die eigentliche Nil-Quelle sogar noch eine Ebene tiefer:
  `contact.Service.enrichWithRelationsBatch` gibt bei `len(contacts) == 0` explizit `nil, nil`
  zurueck (`internal/crm/contact/service.go:401-403), bevor `crm_grpc.go` ueberhaupt eine Slice
  zum Iterieren bekommt. Alle drei Faelle sind mit
  `TestList<X>Fields_EmptyIsNilNotEmptySlice`-Tests dokumentiert (assert `!= nil` wuerde aktuell
  fehlschlagen; die Tests pruefen bewusst die HEUTIGE, nicht die gewuenschte Form und tragen
  einen Kommentar, der beim Fix umzudrehen ist). Empfehlung fuer Lauf 9: eine kleine,
  risikoarme Fix-Unit analog zu `c3f0c46f`, die alle drei Stellen in einem Commit vorbelegt und
  die drei Tests auf `require.NotNil` umstellt. `swagger-cli`/`TestOpenAPIRouteDrift` sind hier
  nicht betroffen (kein Routen- oder Spec-Aenderungsbedarf, reine JSON-Feldform).
  Sonst nichts offen — kein DB-Gate noetig (reine In-Memory-Stubs), kein Proto-Regen, keine
  Route registriert.
  `.planning/backend-block/loop/run-loop.ps1` traegt weiterhin denselben unstaged
  `-StartNotBefore`-Diff wie in den Iterationen 6-9 vermerkt — nicht meine Datei, nicht
  angefasst, nicht committet.
  Laufkontext-Block (Iterationsnummer/Zeitstempel) war in diesem Prompt nicht sichtbar
  mitgeliefert — Nummer aus der letzten Journal-Ueberschrift (Iteration 9) fortgezaehlt,
  Zeitstempel per `date` auf dem Loop-Rechner ermittelt, wie in den vorigen Iterationen.

## Iteration 11 — b-cov-server-crm-companies-dedupe-import — done — 2026-08-10 16:28
- commit: 73dba136
- gebaut: neue Datei `backend/internal/server/crm_grpc_companies_dedupe_import_test.go`
  mit `stubCompanyRepo` (Server-Package-Kopie, gleiches Muster wie `stubContactRepo` aus
  Iteration 10) plus `newCRMServerWithCompanyRepo`/`newCRMServerWithContactAndCompanyRepo`/
  `newCRMServerWithImportExport`-Konstruktoren. 60 neue Tests decken alle 17 genannten
  Methoden ab: Companies CRUD (Create/Get/List/Update/Delete/GetCompanyContacts),
  UpdateContactVisibility (inkl. dediziertem Test fuer den `visibilityService==nil`-
  Fallback-Zweig, der direkt `contactService.UpdateVisibility` aufruft — das ist der in
  Produktion heute aktive Pfad, solange Welle 1b `visibilityService` fuer einen Tenant
  nicht verdrahtet), Dedupe/Merge (FindContactDuplicates, MergeContacts,
  FindCompanyDuplicates, MergeCompanies — je mit Self-Merge- UND Already-Merged-
  Fehlerpfad wie im `done_when` gefordert) sowie Import/Export
  (ImportContactsCSV/VCard/XLSX, ExportContactsCSV/VCard, PreviewImportCSV). Fuer die
  Import/Export-Happy-Paths laufen echte Datei-Bytes durch den Handler (CSV-Text,
  eine per `excelize.NewFile()` erzeugte echte XLSX-Datei, ein minimales vCard) statt
  gemockter Parser-Ergebnisse — der Handler baut fuer den eigentlichen Import/Export
  ohnehin einen lokalen `emailcontact`-Service aus `contactService`/`companyService`
  (`NewTenantScopedAdapter`), das injizierte `importService`/`exportService`-Feld dient
  nur als "konfiguriert"-Flag (Ausnahme: `PreviewImportCSV` ruft es direkt auf, dort
  reicht ein `nil`-Provider, weil `PreviewCSV` nur CSV-Bytes parst, keinen Provider
  beruehrt). `stubContactRepo.MergeInto` (aus der Iteration-10-Datei) war bislang ein
  No-Op und liess `MergedIntoID` unveraendert — `TestMergeContacts_HappyPath` deckte das
  auf (Assertion auf die gesetzte `MergedIntoID` schlug fehl, obwohl der Handler `nil`
  als Fehler zurueckgab); auf das gleiche Set-Pattern wie `stubCompanyRepo.MergeInto`
  umgestellt, damit der Stub den echten Repository-Vertrag (Duplicate wird auf den
  Primary gemerged) tatsaechlich abbildet.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (`go test -count=1
  ./internal/server/` gruen, 1188 PASS / **0 SKIP** bei gesetzter `DATABASE_URL` als
  `kmuhub_app`; zusaetzlich `./internal/server/...` inkl. `response`-Unterpaket gruen) |
  migration n.a. (keine) | rls-smoke n.a. (keine Tabelle/Policy angefasst, reine
  gRPC-Server-Schicht mit In-Memory-Stubs) | route n.a. (keine Route angefasst,
  `go test ./internal/gateway/` daher nicht Pflicht und nicht gelaufen)
- coverage: internal/server 47,7 % -> 50,0 % (`go tool cover -func=/tmp/cov.out | tail -1`,
  Paket `./internal/server/` exakt wie in GATE-COMMANDS.md)
- mutations-probe: zwei Proben in `backend/internal/crm/company/service.go`, beide rot,
  beide zurueckgedreht, `git diff` gegen den finalen Baum restfrei (leer):
  (1) `Delete`: `if hasContacts {` auf `if false && hasContacts {` gedreht ->
  `TestDeleteCompany_InUse` rot ("An error is expected but got nil" statt
  `FailedPrecondition`), zurueckgedreht.
  (2) `MergeCompanies`: `if dup.MergedIntoID != nil {` (Already-Merged-Guard) auf
  `if false && dup.MergedIntoID != nil {` gedreht -> `TestMergeCompanies_AlreadyMerged`
  rot (Merge lief durch statt `FailedPrecondition`), zurueckgedreht.
- verify vorgaenger: sauber — `1fde576f` geprueft gegen alle acht Fehlerklassen: reine
  neue Testdatei (`crm_grpc_fields_tags_contacts_test.go`), kein `.proto` angefasst,
  keine Route, kein neuer `RequirePermission`-Key, keine neue Tabelle, kein
  gRPC-Bypass, kein Stub im Produktionscode, keine bestehende Migration angefasst.
- offen: ECHTER BEFUND, NICHT GEFIXT (Coverage-Units bauen laut Backlog-Kopf keine
  Verhaltensaenderungen) — erweitert den Iteration-10-Fund um vier weitere Stellen mit
  derselben Wire-Shape-Klasse (`var x []*T` bleibt bei leerem Ergebnis `nil` statt `[]`):
  `ListCompanies` (crm_grpc.go, `var infos []*crmv1.CompanyInfo`), `GetCompanyContacts`
  (`var infos []*crmv1.ContactInfo`), `FindContactDuplicates` und
  `FindCompanyDuplicates` (`var results []*crmv1.Duplicate*Candidate`). Zusammen mit den
  drei Stellen aus Iteration 10 (ListContacts/ListCustomFields/ListTags) sind das jetzt
  sieben bekannte Stellen fuer dieselbe Lauf-9-Kandidat-Fix-Unit (analog `c3f0c46f` aus
  Lauf 7) — nicht einzeln neu vermerkt, siehe Iteration-10-Eintrag fuer die ersten drei.
  `stubContactRepo.MergeInto` wurde in dieser Iteration geaendert (siehe `gebaut:` oben)
  — das ist eine Aenderung an einer Datei aus Iteration 10, nicht an meiner eigenen neuen
  Datei; sie gehoert trotzdem in diesen Commit, weil sie der Stub-Korrektheit dient, die
  mein neuer Test aufgedeckt hat, und ohne sie `TestMergeContacts_HappyPath` nicht gruen
  wird.
  `.planning/backend-block/loop/run-loop.ps1` traegt weiterhin denselben unstaged
  `-StartNotBefore`-Diff wie in den Iterationen 6-10 vermerkt — nicht meine Datei, nicht
  angefasst, nicht committet.
  Laufkontext-Block (Iterationsnummer/Zeitstempel) war in diesem Prompt nicht sichtbar
  mitgeliefert — Nummer aus der letzten Journal-Ueberschrift (Iteration 10) fortgezaehlt,
  Zeitstempel per `date` auf dem Loop-Rechner ermittelt (2026-08-10 16:28), wie in den
  vorigen Iterationen.

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

## Iteration 12 — b-cov-server-crm-pipelines-deals — done — 2026-08-10 16:47
- commit: 23c06ac4
- gebaut: neue Datei `backend/internal/server/crm_grpc_pipelines_deals_test.go` mit
  `stubPipelineStageRepo` (implementiert `pipelinestage.Repository`) und `stubDealRepo`
  (implementiert `deal.Repository`), je über `newCRMServerWithPipelineStageRepo`/
  `newCRMServerWithDealRepo` mit der echten `*Service` verdrahtet (gleiches Muster wie
  `stubCompanyRepo`/`newCRMServerWithCompanyRepo` aus Iteration 11). 49 neue Tests decken
  alle 12 in scope genannten Methoden ab: PipelineStages CRUD (Create/Get/List/Update/
  Delete) je mit MissingTenant + mindestens einem Validierungs- oder Fehlerpfad
  (NameRequired, InvalidID, NotFound, InvalidColor, StageHasDeals) plus Happy Path,
  ReorderPipelineStages mit MissingTenant, ungueltiger Stage-ID in der Liste und
  Anzahl-Mismatch (`ErrInvalidReorder`) sowie Happy Path mit tatsaechlich vertauschtem
  `SortOrder`; Deals CRUD (Create/Get/List/Update/Delete) je mit MissingTenant +
  Validierungs-/Fehlerpfad (InvalidCreatedBy, InvalidStageID, StageNotFound, InvalidID,
  NotFound, InvalidCurrency) plus Happy Path, dazu ein dedizierter Test fuer den
  Uuid-Nil-Clear-Pfad bei `UpdateDeal` (leerer `contact_id`-String loescht die Relation)
  und MoveDealToStage mit MissingTenant/InvalidDealID/InvalidStageID/StageNotFound sowie
  Happy Path, der zusaetzlich prueft, dass `closed_at` beim Wechsel in eine Won-Stage
  gesetzt wird (Service-Logik in `deal/service.go:407-411`).
- gate: build ok | vet ok | lint ok (0 issues) | test ok (`go test -count=1
  ./internal/server/` gruen, 0 SKIP bei gesetzter `DATABASE_URL` als `kmuhub_app`;
  zusaetzlich `./internal/server/...` inkl. `response`-Unterpaket gruen) | migration n.a.
  (keine) | rls-smoke n.a. (keine Tabelle/Policy angefasst, reine gRPC-Server-Schicht mit
  In-Memory-Stubs) | route n.a. (keine Route angefasst, `go test ./internal/gateway/`
  daher nicht Pflicht — trotzdem lokal `go build`/`go vet` gegen `internal/gateway/`
  mitlaufen lassen, beides gruen)
- coverage: internal/server 47,7 % -> 50,9 % (`go tool cover -func=/tmp/cov.out | tail -1`,
  Paket `./internal/server/` exakt wie in GATE-COMMANDS.md; Ausgangswert aus
  `coverage_start:` der Unit, nicht aus dem tatsaechlichen Vorgaengerstand 50,0 % aus
  Iteration 11 — wie in ITERATION.md Schritt 6 vorgeschrieben)
- mutations-probe: zwei Proben in den echten Service-Paketen, beide rot, beide
  zurueckgedreht, `git diff` gegen den finalen Baum restfrei (leer):
  (1) `internal/crm/pipelinestage/service.go` Delete: `if hasDeals {` auf
  `if false && hasDeals {` gedreht -> `TestDeletePipelineStage_HasDeals` rot ("An error
  is expected but got nil" statt `FailedPrecondition`), zurueckgedreht.
  (2) `internal/crm/deal/service.go` Update: `if !validCurrencies[currency] {` auf
  `if false && !validCurrencies[currency] {` gedreht -> `TestUpdateDeal_InvalidCurrency`
  rot (Update lief durch statt `InvalidArgument`), zurueckgedreht.
- verify vorgaenger: sauber — `73dba136` geprueft gegen alle acht Fehlerklassen: reine
  neue Testdatei plus ein Fix an einem in Iteration 11 selbst neu eingefuehrten Stub
  (`stubContactRepo.MergeInto`), kein `.proto` angefasst, keine Route, kein neuer
  `RequirePermission`-Key, keine neue Tabelle, kein gRPC-Bypass, kein Stub im
  Produktionscode, keine bestehende Migration angefasst.
- offen: `stubDealRepo.RemoveTags` ist implementiert (Interface-Pflicht), aber von keinem
  Test in dieser Datei direkt aufgerufen — `AddTags`/`RemoveTags` auf `CRMGRPCServer`
  existieren als eigene RPCs nicht im Scope dieser Unit (nur `Create/Get/List/Update/
  Delete/MoveToStage` fuer Deals stehen im `crm_grpc.go`-Handler); falls eine kuenftige
  Unit `AddTags`/`RemoveTags`-Handler auf `CRMGRPCServer` findet, kann der Stub
  wiederverwendet werden. `ReorderPipelineStages` prueft in dieser Unit nur den
  Anzahl-Mismatch als Reorder-Validierungsfehler (`ErrInvalidReorder` deckt sowohl
  Anzahl- als auch Duplikat-Faelle ab, siehe `pipelinestage/service.go:238-261`) —
  Duplikat-Fall nicht separat getestet, gleicher Codepfad, geringes Zusatzrisiko.
  `.planning/backend-block/loop/run-loop.ps1` traegt weiterhin denselben unstaged
  `-StartNotBefore`-Diff wie in den Iterationen 6-11 vermerkt — nicht meine Datei, nicht
  angefasst, nicht committet.
  Laufkontext-Block (Iterationsnummer/Zeitstempel) war in diesem Prompt nicht sichtbar
  mitgeliefert — Nummer aus der letzten Journal-Ueberschrift (Iteration 11) fortgezaehlt,
  Zeitstempel per `date` auf dem Loop-Rechner ermittelt (2026-08-10 16:47), wie in den
  vorigen Iterationen.

## Iteration 13 — b-cov-server-crm-activities-reports-consent — done — 2026-08-10 16:58
- commit: 0dc8bd4b
- gebaut: neue Datei `backend/internal/server/crm_grpc_activities_reports_consent_test.go`
  mit fünf Stub-Repositories (`stubActivityRepo`, `stubSavedFilterRepo`, `stubSearchRepo`,
  `stubReportRepo`, `stubConsentRepo`, je über `newCRMServerWithActivityRepo`/
  `newCRMServerWithSavedFilterRepo`/`newCRMServerWithSearchRepo`/`newCRMServerWithReportRepo`/
  `newCRMServerWithConsentRepo` verdrahtet, gleiches Muster wie `stubPipelineStageRepo` aus
  Iteration 12). 89 neue Tests decken alle 22 Methoden im Scope ab: Activities CRUD +
  CompleteActivity (inkl. Clear-Contact-ID-Pfad wie bei UpdateDeal in Iteration 12), Search
  (QueryTooShort, InvalidEntityType, Happy Path mit Entity-Type-Filter), SavedFilters CRUD,
  Reports (GetPipelineReport/GetConversionReport/GetActivityReport je mit ungültigem Datum
  bzw. ungültiger Owner-/User-ID sowie Happy Path gegen einen fest bestückten Report-Stub),
  GetContactTimeline (Reihenfolge InvalidContactID vor MissingTenant beachtet — der Handler
  parst `contact_id` vor dem Tenant-Check, anders als alle übrigen Methoden in dieser Datei)
  sowie GDPR-Consent (GetContactConsents/GrantConsent/RevokeConsent/GetConsentHistory/
  RequestDeletion/ProcessDeletion). GrantConsent/RevokeConsent/ProcessDeletion decken je den
  geforderten DSGVO-Fehlerpfad ab (InvalidConsentType/InvalidLegalBasis bzw. ContactNotFound
  bzw. AlreadyComplete); `TestProcessDeletion_HappyPath` prüft zusätzlich, dass
  `AnonymizeContact` real aufgerufen wurde (`repo.anonymized[contactID]`) und der
  Deletion-Request-Status im Stub auf `completed` steht.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (`go test -count=1
  ./internal/server/` grün, 1277 PASS / **0 SKIP** bei gesetzter `DATABASE_URL` als
  `kmuhub_app`; zusätzlich `./internal/server/...` inkl. `response`-Unterpaket grün) |
  migration n.a. (keine) | rls-smoke n.a. (keine Tabelle/Policy angefasst, reine
  gRPC-Server-Schicht mit In-Memory-Stubs) | route n.a. (keine Route angefasst,
  `go test ./internal/gateway/` daher nicht Pflicht und nicht gelaufen)
- coverage: internal/server 47,7 % -> 52,7 % (`go tool cover -func=/tmp/cov.out | tail -1`,
  Paket `./internal/server/` exakt wie in GATE-COMMANDS.md; Ausgangswert aus
  `coverage_start:` der Unit, nicht aus dem tatsächlichen Vorgängerstand 50,9 % aus
  Iteration 12 — wie in ITERATION.md Schritt 6 vorgeschrieben)
- mutations-probe: zwei Proben in den echten Service-Paketen, beide rot, beide
  zurückgedreht, `git diff` gegen den finalen Baum restfrei (leer):
  (1) `internal/crm/activity/service.go` Create: `if !input.ActivityType.IsValid() {` auf
  `if false && !input.ActivityType.IsValid() {` gedreht -> `TestCreateActivity_InvalidActivityType`
  rot ("An error is expected but got nil" statt `InvalidArgument`), zurückgedreht.
  (2) `internal/crm/consent/service.go` ProcessDeletion: `if req.Status == "completed" {` auf
  `if false && req.Status == "completed" {` gedreht -> `TestProcessDeletion_AlreadyComplete`
  rot (Deletion lief durch statt `AlreadyExists`), zurückgedreht.
- verify vorgaenger: sauber — `23c06ac4` geprüft gegen alle acht Fehlerklassen: reine neue
  Testdatei (`crm_grpc_pipelines_deals_test.go`), kein `.proto` angefasst, keine Route, kein
  neuer `RequirePermission`-Key, keine neue Tabelle, kein gRPC-Bypass, kein Stub im
  Produktionscode, keine bestehende Migration angefasst.
- offen: ZWEI ECHTE BEFUNDE, NICHT GEFIXT (Coverage-Units bauen laut Backlog-Kopf keine
  Verhaltensänderungen):
  (1) `UpdateSavedFilter`/`DeleteSavedFilter` in `crm_grpc.go` laden den Filter selbst
  (`existingFilter, err := s.savedFilterService.GetByID(...)`) und reichen dann
  `existingFilter.CreatedBy` als `userID`-Parameter an `Update`/`Delete` durch. Der
  Ownership-Check in `savedfilter/service.go` (`if filter.CreatedBy != userID`) vergleicht
  damit denselben Wert mit sich selbst — `ErrFilterNotOwned` (mapCRMError:
  `codes.PermissionDenied`) kann über diese beiden RPCs nie auslösen, unabhängig davon, wer
  den Request schickt. Der Service selbst ist korrekt (siehe `savedfilter/service_test.go`),
  der Bug sitzt im Handler-Wiring. Hätte production-Auswirkung: jeder Nutzer mit gültigem
  Tenant-Token kann fremde Saved Filters umbenennen/löschen, solange er die ID kennt.
  Empfehlung Lauf 9: kleine Fix-Unit, die den echten Caller (aus dem Request bzw. Auth-
  Context) statt `existingFilter.CreatedBy` durchreicht.
  (2) `GetPipelineReport`/`GetConversionReport`/`GetActivityReport`: die Service-Sentinels
  `ErrStartDateRequired`/`ErrEndDateRequired` (`internal/crm/report/errors.go`,
  gemappt in `mapCRMError` auf `InvalidArgument`) sind über diese drei RPCs unerreichbar,
  weil der Handler `parseDate(req.StartDate)`/`parseDate(req.EndDate)` bereits vorher
  aufruft und bei leerem oder ungültigem String selbst mit `InvalidArgument "invalid
  start_date/end_date format"` abbricht — `time.Time{}` (Zero-Value) erreicht die
  Service-Methoden nie. Kein Bug (das Endergebnis für den Client ist in beiden Fällen
  `InvalidArgument`), aber toter Code auf Service-Ebene — nicht als eigene Fix-Unit nötig,
  nur zur Kenntnis für Lauf 9, falls dort an `report/service.go` gearbeitet wird.
  `.planning/backend-block/loop/run-loop.ps1` trägt weiterhin denselben unstaged
  `-StartNotBefore`-Diff wie in den Iterationen 6-12 vermerkt — nicht meine Datei, nicht
  angefasst, nicht committet.
  Laufkontext-Block (Iterationsnummer/Zeitstempel) war in diesem Prompt nicht sichtbar
  mitgeliefert — Nummer aus der letzten Journal-Überschrift (Iteration 12) fortgezählt,
  Zeitstempel per `date` auf dem Loop-Rechner ermittelt (2026-08-10 16:58), wie in den
  vorigen Iterationen.

## Iteration 14 — b-cov-server-biz-error-map-settings-quotes — done — 2026-08-10 17:07
- commit: 0a5e5e55
- gebaut: neue Datei `backend/internal/server/biz_grpc_errormap_settings_quotes_test.go`.
  `TestMapBizError` deckt jedes der 32 Sentinels in `mapBizError` (biz_grpc.go:2611)
  einzeln gegen den erwarteten gRPC-Code ab (Quote, Recurring, Offene-Posten,
  Invoice, CreditNote, E-Invoice, Payment, Dunning) plus `nil`- und
  Fallback-Pfad (`unknown error` -> `Internal`, separat `TestMapBizError_Nil`).
  Dazu je mindestens ein Validierungs- und ein Happy-Path-Test für die 12
  genannten Quote/Settings-Methoden: GetCompanySettings, UpdateCompanySettings
  (inkl. `settings is required` und `invalid basiszinssatz`), CreateQuote
  (invalid tenant_id/created_by/valid_until/deal_id, `ErrNoLineItems` ->
  InvalidArgument, Happy Path), GetQuote (invalid id, NotFound-Mapping, Happy
  Path), ListQuotes, UpdateQuote (`ErrQuoteNotDraft` -> FailedPrecondition),
  DeleteQuote (dito), SendQuote (atomare Nummernvergabe über einen
  `fakeTxBeginner`/`fakeTx` — echtes `Send` inkl. Statuswechsel auf `sent` und
  zugewiesener Quote-Nummer), AcceptQuote/RejectQuote (`ErrQuoteNotSent` ->
  FailedPrecondition), ExpireQuote (invalid tenant_id, leerer Bulk-Lauf),
  ConvertQuoteToInvoice (Validierungspfad — kein voller `invoice.Service`
  aufgebaut, siehe offen), CreateQuoteFromDeal (deal_id required, invalid
  tenant_id, `crmClient == nil` -> Unavailable). Zusätzlich
  `TestGenerateQuotePDF_QuoteNotFound` als kleiner Bonus für den PDF-Pfad
  (nicht in den 12 genannten Methoden, aber selber Fehlerpfad). Neue Stubs:
  `stubCompanySettingsRepo` (erfüllt sowohl `server.CompanySettingsRepository`
  als auch `quote.CompanySettingsRepo` strukturell), `stubQuoteRepo` (volles
  `quote.Repository`), `stubNumberSeqRepo`, `fakeTxBeginner`/`fakeTx` (eigene
  Kopie des unexported `quote.txBeginner`-Musters aus `service_test.go`, weil
  der Typ dort nicht exportiert ist).
- gate: build ok | vet ok | lint ok (0 issues) | test ok (`go test -count=1
  ./internal/server/` grün, 0 SKIP bei gesetzter `DATABASE_URL` als
  `kmuhub_app`; zusätzlich `./internal/server/...` inkl. `response`-Unterpaket
  grün) | migration n.a. (keine) | rls-smoke n.a. (keine Tabelle/Policy
  angefasst — reine gRPC-Server-Schicht mit In-Memory-Stubs) | route n.a.
  (keine Route angefasst, `go test ./internal/gateway/` daher nicht Pflicht
  und nicht gelaufen)
- coverage: internal/server 47,7 % -> 53,7 % (`go tool cover -func=/tmp/cov.out
  | tail -1`, Paket `./internal/server/` exakt wie in GATE-COMMANDS.md;
  Ausgangswert aus `coverage_start:` der Unit, nicht aus dem tatsächlichen
  Vorgängerstand 52,7 % aus Iteration 13 — wie in ITERATION.md Schritt 6
  vorgeschrieben)
- mutations-probe: zwei Proben, beide rot, beide zurückgedreht, `git diff`
  gegen den finalen Baum restfrei (leer):
  (1) `mapBizError` Case `quote.ErrQuoteNotDraft`: `codes.FailedPrecondition`
  auf `codes.Internal` gedreht -> `TestMapBizError/quote_not_draft` rot
  (erwartet FailedPrecondition, bekam Internal), zurückgedreht.
  (2) `UpdateCompanySettings`: `if ps == nil {` auf `if false && ps == nil {`
  gedreht -> `TestUpdateCompanySettings/settings_required` rot ("An error is
  expected but got nil"), zurückgedreht.
- verify vorgaenger: sauber — `0dc8bd4b` geprüft gegen alle acht
  Fehlerklassen: reine neue Testdatei
  (`crm_grpc_activities_reports_consent_test.go`), kein `.proto` angefasst,
  keine Route, kein neuer `RequirePermission`-Key, keine neue Tabelle, kein
  gRPC-Bypass, kein Stub im Produktionscode, keine bestehende Migration
  angefasst.
- offen: `ConvertQuoteToInvoice` ist nur mit dem Validierungspfad
  (`invalid tenant_id`/`invalid id` über `parseTenantAndID`) abgedeckt, nicht
  mit einem Happy Path — ein voller `invoice.Service` (5 Konstruktor-Parameter,
  eigener `quoteReader`/`numberSeqRepo`/Pool) hätte den Scope dieser Iteration
  gesprengt. `mapBizError` deckt `invoice.ErrQuoteNotAccepted` bereits einzeln
  ab, das ist der einzige service-eigene Fehlerpfad dieser Methode. Ebenso
  `CreateQuoteFromDeal` nur mit drei Validierungsfällen (deal_id required,
  invalid tenant_id, `crmClient == nil`), kein Happy Path — bräuchte einen
  vollständigen Fake für `crmv1.CRMServiceClient` (großes Interface), den es
  im `server`-Package noch nicht gibt. Für Lauf 9, falls an
  `CreateQuoteFromDeal` weitergearbeitet wird: ein minimaler
  `fakeCRMServiceClient` (nur `GetDeal`/`GetContact`/`GetCompany` implementiert,
  Rest `Unimplemented`) würde den Happy Path und die Enrichment-Logik
  (Firma/Kontakt-Fallback für `customerName`) freischalten.
  `.planning/backend-block/loop/run-loop.ps1` trägt weiterhin denselben
  unstaged `-StartNotBefore`-Diff wie in den Iterationen 6-13 vermerkt — nicht
  meine Datei, nicht angefasst, nicht committet.
  Laufkontext-Block (Iterationsnummer/Zeitstempel) war in diesem Prompt nicht
  sichtbar mitgeliefert — Nummer aus der letzten Journal-Überschrift
  (Iteration 13) fortgezählt, Zeitstempel per `date` auf dem Loop-Rechner
  ermittelt (2026-08-10 17:07), wie in den vorigen Iterationen.

## Iteration 15 — b-cov-server-biz-invoices-creditnotes-payments — done — 2026-08-10 17:22
- commit: e994ddf5
- gebaut: `biz_grpc_invoices_creditnotes_payments_test.go` — Validierungs-,
  Fehler- und Happy-Path-Tests für alle 23 im Backlog genannten Methoden
  (Rechnungen: CreateInvoice, GetInvoice, ListInvoices, UpdateInvoice,
  SendInvoice, MarkInvoicePaid, CancelInvoice, ValidateInvoiceNumber,
  LockInvoice, GenerateInvoicePDF, GenerateZUGFeRDInvoicePDF,
  GenerateEInvoice, CreateInvoiceFromTimeEntries; Gutschriften:
  CreateCreditNote, GetCreditNote, ListCreditNotes, SendCreditNote,
  GenerateCreditNotePDF; Zahlungen: RecordPayment, ListPayments,
  DeletePayment; GetPaymentStats, GetJournalSummary). CancelInvoice und
  MarkInvoicePaid decken den FailedPrecondition-Pfad für bereits bezahlte
  Rechnungen ab (`ErrInvoiceAlreadyPaid`); CancelInvoice zusätzlich den
  bislang ungetesteten `ErrStornoUnavailable`-Pfad (versendete Rechnung ohne
  gewirten StornoCreator -> Internal). Neue Stubs: `stubInvoiceRepo` (volles
  `invoice.Repository`, 18 Methoden), `stubCreditNoteRepo`
  (`creditnote.Repository`), `stubPaymentRepo` (`payment.Repository`,
  inkl. In-Tx-Varianten über `fakeTx`), `stubInvoiceNumberSeqRepo` (eigener
  Typ statt Erweiterung des bestehenden `stubNumberSeqRepo`, weil
  `invoice.NumberSequenceRepo` zusätzlich `GetSequenceInfo` braucht —
  `creditnote.NumberSequenceRepo` bleibt beim bestehenden Zwei-Methoden-Stub).
  `newFinanceTestServer` verdrahtet alle drei Services auf denselben
  `stubInvoiceRepo`, der strukturell sowohl `creditnote.InvoiceReader` als
  auch `payment.InvoiceReader`/`InvoiceStatusUpdater` erfüllt.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (`go test -count=1
  ./internal/server/` grün, 0 SKIP bei gesetzter `DATABASE_URL` als
  `kmuhub_app`; `./internal/server/...` inkl. `response`-Unterpaket grün) |
  migration n.a. (keine) | rls-smoke n.a. (keine Tabelle/Policy angefasst —
  reine gRPC-Server-Schicht mit In-Memory-Stubs) | route n.a. (keine Route
  angefasst, `go test ./internal/gateway/` daher nicht Pflicht und nicht
  gelaufen)
- coverage: internal/server 47,7 % -> 55,0 % (`go tool cover -func=/tmp/cov.out
  | tail -1`, Paket `./internal/server/` exakt wie in GATE-COMMANDS.md;
  Ausgangswert aus `coverage_start:` der Unit, tatsächlicher Vorgängerstand aus
  Iteration 14 war 53,7 %)
- mutations-probe: zwei Proben, beide rot, beide zurückgedreht, `git diff`
  gegen `internal/biz/invoice/service_gobd.go` restfrei (leer):
  (1) `GetJournalSummary`: `gaps := max(seq.CurrentNumber-invoiceCount, 0)`
  auf `gaps := 0` gedreht -> `TestGetJournalSummary/gap_detected_when_issued_
  count_trails_the_sequence` rot (erwartete 2, bekam 0), zurückgedreht.
  (2) `ValidateInvoiceNumber`: `AlreadyUsed: used` auf `AlreadyUsed: !used`
  gedreht -> sowohl `already_used` als auch `happy_path_canonicalizes_the_
  number` rot (invertierte Erwartung in beide Richtungen), zurückgedreht.
- verify vorgaenger: sauber — `0a5e5e55` geprüft gegen alle acht
  Fehlerklassen: reine neue Testdatei
  (`biz_grpc_errormap_settings_quotes_test.go`), kein `.proto` angefasst,
  keine Route, kein neuer `RequirePermission`-Key, keine neue Tabelle, kein
  gRPC-Bypass, kein Stub im Produktionscode, keine bestehende Migration
  angefasst. `origin/main` war bereits vollständig in `backend-loop`
  gemergt (kein neuer Merge nötig).
- offen: `CreateInvoiceFromTimeEntries` ist nur mit den sieben
  Validierungspfaden (inkl. `s.timetrackingRepo == nil` -> Unavailable)
  abgedeckt, kein Happy Path — bräuchte einen vollständigen Fake für
  `timetracking.WorkTimeRepository` (12 Methoden), den es im `server`-Package
  noch nicht gibt. Für Lauf 9: ein minimaler `stubWorkTimeRepo` (nur
  `AggregateWorkTimeForInvoice` sinnvoll befüllt, Rest `Unimplemented`/Zero)
  würde den Happy Path (Stundenaggregation -> Rechnungsposition) freischalten.
  `GenerateInvoicePDF`/`GenerateZUGFeRDInvoicePDF`/`GenerateCreditNotePDF`
  sind nur mit ihren Fehlerpfaden (NotFound, fehlende Company-Settings)
  getestet, kein Happy Path mit echter PDF-Generierung (maroto/v2) — gleiches
  Muster wie `TestGenerateQuotePDF_QuoteNotFound` in Iteration 1/13, damit
  konsistent zum Rest der Datei. `RecordPayment` mit einer Rechnung im
  `draft`-Status mappt aktuell auf `codes.Internal` (plain `fmt.Errorf`, kein
  Sentinel) statt `FailedPrecondition` — echtes, unverändertes Verhalten, nur
  als Beobachtung notiert, nicht Teil des Fix-Scopes dieser Unit.
  `.planning/backend-block/loop/run-loop.ps1` trägt weiterhin denselben
  unstaged `-StartNotBefore`-Diff wie in den Iterationen 6-14 vermerkt — nicht
  meine Datei, nicht angefasst, nicht committet.
  Laufkontext-Block (Iterationsnummer/Zeitstempel) war in diesem Prompt nicht
  sichtbar mitgeliefert — Nummer aus der letzten Journal-Überschrift
  (Iteration 14) fortgezählt, Zeitstempel per `date` auf dem Loop-Rechner
  ermittelt (2026-08-10 17:22).

## Iteration 16 — b-cov-server-biz-dunning-dashboard-exports — done — 2026-08-10 17:33
- commit: 1b4039f5
- gebaut: neue Datei `backend/internal/server/biz_grpc_dunning_dashboard_exports_test.go`.
  Deckt alle 11 im Backlog genannten Methoden ab: Mahnwesen (ListDunnings,
  CreateDunning, EscalateDunning, GetDunningConfig, UpdateDunningConfig,
  UpdateDunningStatus, SendDunningNotice, GenerateDunningPDF), Dashboard
  (GetFinanceDashboard) und Exporte (ExportDATEV, GenerateGoBDExport).
  EscalateDunning prüft explizit die Eskalations-Obergrenze (bestehender
  Level-3-sent-Datensatz -> `EscalateDunning` liefert FailedPrecondition, kein
  Level-4-Draft). Neue Stubs: `stubDunningRecordRepo` (volles
  `dunning.Repository`, konfigurierbar für List/Create/UpdateStatus-Fehlerpfade,
  bewusst getrennt vom minimalen `stubDunningRepo` aus
  `biz_grpc_dunning_test.go`, der nur SendDunning bedient), `stubDunningConfigRepo`
  (`dunning.ConfigRepository`), `stubDunningInvoiceReader` (`dunning.InvoiceReader`,
  nur GetOverdue/GetByID real befüllt), `stubDashboardRepo` (`dashboard.Repository`,
  ein Methode), `stubInvoicePager`/`stubCreditNotePager`
  (`datev.InvoicePager`/`datev.CreditNotePager` für den echten
  `datev.BuchungsstapelBuilder` mit `datev.NewExporter()`, kein Mock des
  Exporters selbst). `stubCompanySettingsRepo` aus der Quote/Settings-Iteration
  wiederverwendet (implementiert zusätzlich `datev.CompanySettingsReader`).
  GenerateGoBDExport nur mit den drei Validierungspfaden (invalid tenant_id/
  from_date/to_date) — Details siehe „offen".
- gate: build ok | vet ok | lint ok (0 issues) | test ok (`go test -count=1
  ./internal/server/` grün, 0 SKIP bei gesetzter `DATABASE_URL` als
  `kmuhub_app`; `./internal/server/...` inkl. `response`-Unterpaket grün) |
  migration n.a. (keine) | rls-smoke n.a. (keine Tabelle/Policy angefasst —
  reine gRPC-Server-Schicht mit In-Memory-Stubs) | route n.a. (keine Route
  angefasst, `go test ./internal/gateway/` daher nicht Pflicht und nicht
  gelaufen)
- coverage: internal/server 47,7 % -> 55,8 % (`go tool cover -func=/tmp/cov.out
  | tail -1`, Paket `./internal/server/` exakt wie in GATE-COMMANDS.md;
  Ausgangswert aus `coverage_start:` der Unit, tatsächlicher Vorgängerstand aus
  Iteration 15 war 55,0 %)
- mutations-probe: zwei Proben, beide rot, beide zurückgedreht, `git diff`
  gegen `backend/internal/biz/dunning/service.go` restfrei (leer):
  (1) `UpdateDunningConfig`: `input.Level1DaysAfterDue = &days` durch `_ = days`
  ersetzt -> `TestUpdateDunningConfig/happy_path_updates_level_days_and_fees`
  rot (erwartete 21, bekam 14 aus dem unveränderten Default-Config), zurückgedreht.
  (2) `sendAndNotify` (`service.go`): `if record.Status != models.DunningStatusDraft`
  auf `if false && record.Status != models.DunningStatusDraft` gedreht ->
  `TestSendDunningNotice/not_draft` rot ("An error is expected but got nil"),
  zurückgedreht.
- verify vorgaenger: sauber — `e994ddf5` geprüft gegen alle acht
  Fehlerklassen: reine neue Testdatei
  (`biz_grpc_invoices_creditnotes_payments_test.go`), kein `.proto` angefasst,
  keine Route, kein neuer `RequirePermission`-Key, keine neue Tabelle, kein
  gRPC-Bypass, kein Stub im Produktionscode, keine bestehende Migration
  angefasst. `origin/main` war bereits vollständig in `backend-loop` gemergt
  (kein neuer Merge nötig).
- offen: `GenerateGoBDExport` ist nur mit den drei Validierungspfaden
  (invalid tenant_id/from_date/to_date) abgedeckt, kein Happy Path — der
  Handler ruft direkt `s.invoiceService.ListForDATEVExport` und
  `s.creditNoteService.ListForDATEVExport` auf konkreten
  `*invoice.Service`/`*creditnote.Service`-Feldern (keine Interfaces), nicht
  über die schlanken `datev.InvoicePager`/`CreditNotePager`-Interfaces wie
  `ExportDATEV`. Ein voller Test bräuchte dieselbe schwere
  Service-Konstruktion, die bereits bei `CreateInvoiceFromTimeEntries` und
  `CreateQuoteFromDeal` als out-of-scope markiert wurde. Für Lauf 9: entweder
  ein minimaler `stubInvoiceService`/`stubCreditNoteService`, der nur
  `ListForDATEVExport` implementiert (falls die Felder testweise auf
  Interfaces umgestellt würden — das wäre selbst ein Produktivcode-Umbau,
  keine reine Test-Unit), oder Akzeptanz, dass GoBD-Export-Happy-Path nur via
  Integrationstest mit echter DB abgedeckt wird.
  `GetDunningConfig`/`UpdateDunningConfig` testen nur den expliziten
  Fehlerfall über `configRepo.getErr`; den impliziten Auto-Create-Default-Pfad
  (configRepo.Get liefert `nil, nil` -> Service legt Default an und ruft
  Upsert) habe ich nicht separat getestet, weil `GetDunningConfig`/
  `TestCreateDunning`/`TestEscalateDunning` diesen Pfad bereits indirekt über
  `defaultDunningConfig` umgehen (Config immer vorbelegt) — echte
  Auto-Create-Assertion wäre ein zusätzlicher, in dieser Iteration nicht
  eingeplanter Testfall.
  `.planning/backend-block/loop/run-loop.ps1` trägt weiterhin denselben
  unstaged `-StartNotBefore`-Diff wie in den Iterationen 6-15 vermerkt — nicht
  meine Datei, nicht angefasst, nicht committet.
  Laufkontext-Block (Iterationsnummer/Zeitstempel) war in diesem Prompt nicht
  sichtbar mitgeliefert — Nummer aus der letzten Journal-Überschrift
  (Iteration 15) fortgezählt, Zeitstempel per `date` auf dem Loop-Rechner
  ermittelt (2026-08-10 17:33).

## Iteration 17 — b-cov-server-calendar-error-map-calendars-members — done — 2026-08-10 17:36
- commit: 52e07376
- gebaut: neue Testdatei
  `calendar_grpc_errormap_calendars_members_test.go` mit `TestMapCalendarError`
  (alle 52 Sentinel-Fälle aus `mapCalendarError` — calendar/event/resource/
  holiday/livekit/booking-page — einzeln gegen den erwarteten gRPC-Code, plus
  Fallback auf `codes.Internal`) und `stubCalendarRepo` (implementiert
  `calendar.Repository` vollständig; Category/Preference/Visibility-Methoden
  sind No-Ops, weil sie zu einer späteren Unit gehören und hier nie erreicht
  werden). Je ein Happy-Path- und ein Fehlerpfad-Test für alle 12 im Scope
  genannten RPCs: CreateCalendar, GetCalendar, ListCalendars, UpdateCalendar,
  DeleteCalendar, AddCalendarMember, RemoveCalendarMember, ListCalendarMembers,
  UpdateCalendarMemberPermission, ListBrowsableCalendars, SubscribeToCalendar,
  UnsubscribeFromCalendar.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (`go test -count=1
  ./internal/server/` grün, 0 SKIP bei gesetzter `DATABASE_URL` als
  `kmuhub_app`; `./internal/server/...` inkl. `response`-Unterpaket grün) |
  migration n.a. (keine) | rls-smoke n.a. (keine Tabelle/Policy angefasst —
  reine gRPC-Server-Schicht mit In-Memory-Stub) | route n.a. (keine Route
  angefasst, `go test ./internal/gateway/` daher nicht Pflicht und nicht
  gelaufen)
- coverage: internal/server 47,7 % -> 56,6 % (`go tool cover -func=/tmp/cov.out
  | tail -1`, Paket `./internal/server/` exakt wie in GATE-COMMANDS.md;
  Ausgangswert aus `coverage_start:` der Unit, tatsächlicher Vorgängerstand aus
  Iteration 16 war 55,8 %)
- mutations-probe: zwei Proben, beide rot, beide zurückgedreht, `git diff`
  gegen beide betroffenen Dateien restfrei (leer):
  (1) `mapCalendarError` (`calendar_grpc.go`): `calendar.ErrCannotDeleteDefaultCalendar`
  von `codes.FailedPrecondition` auf `codes.Internal` gedreht ->
  `TestMapCalendarError/cannot_delete_default_calendar` UND
  `TestDeleteCalendar/cannot_delete_default_calendar` beide rot (0x9 erwartet,
  0xd bekommen), zurückgedreht.
  (2) `calendar.Service.AddMember` (`internal/work/calendar/service.go`):
  `if cal.OwnerID == targetUserID` auf `if false && cal.OwnerID == targetUserID`
  gedreht -> `TestAddCalendarMember/owner_cannot_be_added` rot (InvalidArgument
  erwartet, PermissionDenied bekommen — der Owner rutschte in den
  Admin-Permission-Check statt der Guard-Klausel), zurückgedreht.
- verify vorgaenger: sauber — `1b4039f5` geprüft gegen alle acht
  Fehlerklassen: reine neue Testdatei
  (`biz_grpc_dunning_dashboard_exports_test.go`), kein `.proto` angefasst,
  keine Route, kein neuer `RequirePermission`-Key, keine neue Tabelle, kein
  gRPC-Bypass, kein Stub im Produktionscode, keine bestehende Migration
  angefasst. `origin/main` war bereits vollständig in `backend-loop` gemergt
  (kein neuer Merge nötig).
- offen: `mapCalendarError` behandelt einige im Code definierte Sentinels
  nicht (z. B. `event.ErrExceptionNotFound`, `event.ErrExceptionAlreadyExists`,
  `holiday.ErrHolidayNotFound`, `holiday.ErrSeedFailed`) — das ist keine Lücke
  dieser Unit, diese Fehler laufen laut `errors.go`-Kommentaren über andere
  Pfade oder werden dort noch gar nicht produziert; nicht weiter untersucht,
  da außerhalb des Scopes „jedes Sentinel IN mapCalendarError".
  Events/Categories/Reminders (`b-cov-server-calendar-events-categories-
  reminders`) und Resources/BookingPages/Public
  (`b-cov-server-calendar-resources-bookingpages-public`) bleiben unverändert
  bei ihrem alten Coverage-Stand — dieselbe `stubCalendarRepo` kann für beide
  wiederverwendet werden (No-Op-Methoden für Category/Preferences ggf. dann
  ausbauen).
  `.planning/backend-block/loop/run-loop.ps1` trägt weiterhin denselben
  unstaged `-StartNotBefore`-Diff wie in den Iterationen 6-16 vermerkt — nicht
  meine Datei, nicht angefasst, nicht committet.
  Laufkontext-Block (Iterationsnummer/Zeitstempel) war in diesem Prompt nicht
  sichtbar mitgeliefert — Nummer aus der letzten Journal-Überschrift
  (Iteration 16) fortgezählt, Zeitstempel per `date` auf dem Loop-Rechner
  ermittelt (2026-08-10 17:36).

## Iteration 18 — b-cov-server-calendar-events-categories-reminders — done — 2026-08-10 17:59
- commit: 78685fc7
- gebaut: neue Testdatei calendar_grpc_events_categories_reminders_test.go
  mit stubEventRepo (implementiert event.Repository vollstaendig) und
  newTestCalendarServerWithEvents/disabledLiveKit/enabledLiveKit-Helfern.
  Je ein Validierungs- und/oder Happy-Path-Test plus mindestens ein Fehlerpfad
  fuer alle 15 Scope-Methoden: CreateEvent, GetEvent, ListEventsInRange,
  UpdateEvent, DeleteEvent, UpdateRecurringEvent (inkl. Scope-Fehlerpfad
  "event not recurring"), RSVPToEvent, ListEventAttendees, CreateEventCategory,
  ListEventCategories, DeleteEventCategory, SetEventReminders,
  ListEventReminders, ListTaskDeadlinesInRange, GenerateJoinToken.
  Zusaetzlich stubCalendarRepo (aus Iteration 17,
  calendar_grpc_errormap_calendars_members_test.go) erweitert: die
  Category-Methoden (CreateCategory/ListCategories/DeleteCategory) waren dort
  bewusst No-Ops "fuer eine spaetere Unit" -- jetzt echte Map-basierte
  Implementierung inkl. DeleteCategory-Semantik, die exakt
  postgres_repository.go:337-349 nachbildet (0 rows affected ->
  ErrCategoryNotFound). Iteration 17 selbst bleibt unangetastet gruen, da ihre
  Tests die Category-Methoden nie erreichen.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (go test -count=1
  ./internal/server/ gruen, 0 SKIP bei gesetzter DATABASE_URL als
  kmuhub_app) | ./internal/server/... gruen in zwei separaten Laeufen; ein
  dritter Lauf waehrend der Entwicklung zeigte ein einzelnes FAIL in einem
  Timer/Duration-Test (zeitabhaengig, nicht in meinen Dateien, in zwei
  Wiederholungen nicht reproduzierbar -- vorbestehende Flakiness, nicht
  Gegenstand dieser Unit) | migration n.a. (keine) | rls-smoke n.a. (keine
  Tabelle/Policy angefasst) | route n.a. (keine Route angefasst, go test
  ./internal/gateway/ daher nicht Pflicht und nicht gelaufen)
- coverage: internal/server 47,7 % -> 57,8 % (go tool cover -func=/tmp/cov.out
  | tail -1, Paket ./internal/server/ exakt wie in GATE-COMMANDS.md;
  Ausgangswert aus coverage_start: der Unit, tatsaechlicher Vorgaengerstand aus
  Iteration 17 war 56,6 %)
- mutations-probe: zwei Proben, beide rot, beide zurueckgedreht, git diff
  gegen internal/work/event/service.go restfrei (leer):
  (1) Service.SetReminders (event/service.go:487): Limit-Check von
  len(minutesBefore) > 3 auf > 4 gedreht ->
  TestSetEventReminders/reminder_limit_exceeded rot (InvalidArgument
  erwartet, NotFound bekommen -- der 4-Item-Request rutschte durch den
  entschaerften Limit-Check und traf danach auf ein nicht existierendes
  Event), zurueckgedreht.
  (2) Service.UpdateRecurringEvent (event/service.go:231): die
  Not-Recurring-Pruefung evt.RRule == nil || *evt.RRule == "" mit
  false && (...) deaktiviert ->
  TestUpdateRecurringEvent/event_not_recurring_(scope_error_path) rot (Fehler
  erwartet, nil bekommen -- der Edit lief auf einem nicht-wiederkehrenden
  Event unbemerkt durch), zurueckgedreht.
- verify vorgaenger: sauber -- 52e07376 geprueft gegen alle acht
  Fehlerklassen: reine neue Testdatei
  (calendar_grpc_errormap_calendars_members_test.go), kein .proto
  angefasst, keine Route, kein neuer RequirePermission-Key, keine neue
  Tabelle, kein gRPC-Bypass, kein Stub im Produktionscode, keine bestehende
  Migration angefasst. origin/main war bereits vollstaendig in backend-loop
  gemergt (kein neuer Merge noetig, verifiziert per
  git merge-base --is-ancestor origin/main HEAD).
- offen: WICHTIGER FUND, NICHT Teil dieser Unit -- Kandidat fuer eine
  Fix-Unit in Lauf 9, hohe Prioritaet. Zwei getrennte, jeweils verifizierte
  Bugs in backend/internal/server/calendar_grpc.go:
  (A) CreateEvent threadet den Tenant nie durch. CreateEvent
  (calendar_grpc.go:391-459) ruft middleware.GetTenantID(ctx) an keiner
  Stelle auf -- anders als praktisch jeder andere Handler in dieser Datei.
  event.CreateInput.TenantID bleibt dadurch die Nullwert-UUID, und
  event.Service.Create (service.go:77) laedt den Kalender ueber
  s.calendarRepo.GetByID(ctx, input.CalendarID, input.TenantID) -- die
  Postgres-Query filtert explizit WHERE id=$1 AND tenant_id=$2
  (calendar/postgres_repository.go:41). Fuer jeden echten (Nicht-Null-)Tenant
  kann das nie einen Treffer liefern. TestCreateEvent/"calendar not found
  (missing tenant scoping)" belegt das direkt: derselbe Tenant im ctx und auf
  dem Kalender-Datensatz fuehrt trotzdem zu NotFound. Der Gateway-Handler
  (route_calendar.go:649, HandleCreateEvent) reicht r.Context() unveraendert
  durch -- es fehlt schlicht der middleware.GetTenantID(ctx)-Aufruf plus
  input.TenantID = tenantID in CreateEvent. Wirkung in Produktion: JEDES
  Erstellen eines Kalenderereignisses ueber den normalen API-Pfad scheitert an
  "calendar not found", ausser ein Kalender traegt zufaellig die Tenant-ID
  00000000-0000-0000-0000-000000000000.
  (B) Acht RPCs mit hart verdrahtetem uuid.Nil-Actor, alle vom selben
  Muster. Kommentar "Gateway handles auth; use uuid.Nil as actorID/userID"
  an calendar_grpc.go:165, 201, 229, 253, 303 (UpdateCalendar, DeleteCalendar,
  AddCalendarMember, RemoveCalendarMember, UpdateCalendarMemberPermission --
  alle in Iteration 17 getestet) sowie :598, :791, :820 (DeleteEvent,
  DeleteEventCategory, SetEventReminders -- in dieser Unit getestet). Die
  jeweilige Service-Methode prueft aber cal.OwnerID != actorID bzw.
  evt.CreatedBy != actorID und verlangt bei Ungleichheit eine
  Member-/Edit-Berechtigung fuer GENAU diesen Actor
  (calendar/service.go:133-137, 180-182, 215-218, 257-260, 293-296;
  event/service.go:398-400, 503-507; calendar/postgres_repository.go:337-349
  fuer die category-Delete-Query). Da der Actor immer uuid.Nil ist und ein
  echter Owner/Creator/User nie diese UUID hat, schlagen alle acht RPCs fuer
  jeden echten Datensatz fehl (PermissionDenied bzw. NotFound je nach
  Pruefpfad) -- ausser der Aufrufer setzt den Owner/Creator explizit auf
  uuid.Nil, was in echten Daten nie vorkommt.
  middleware.GetUserID(ctx) liefert den echten Actor bereits serverseitig
  (middleware/auth.go:66, vom gRPC-Tenant-Interceptor gesetzt,
  middleware/grpc_tenant.go:83) und wird an keiner der acht Stellen gelesen.
  TestDeleteEvent/"real creator denied (uuid.Nil actor bug)",
  TestDeleteEventCategory/"real owner denied (uuid.Nil actor bug)" und
  TestSetEventReminders/"real creator denied (uuid.Nil actor bug)" belegen
  das direkt. Iteration 17s Happy-Path-Tests fuer die fuenf Calendar/Member-RPCs
  setzen durchweg OwnerID: uuid.Nil auf den Test-Kalendern -- das ist ein
  Workaround um genau diesen Bug, kein Beleg, dass die RPCs fuer echte Owner
  funktionieren; das war beim Schreiben dieser Unit nicht sofort ersichtlich
  und sollte bei kuenftigen Coverage-Units auf derselben Datei ebenfalls
  geprueft werden, bevor "OwnerID: uuid.Nil" unkommentiert als Fixture-Default
  weiterkopiert wird.
  Root Cause fuer beide Funde ist strukturell identisch: die
  gRPC-Handler-Schicht sollte den Actor/Tenant aus dem Kontext lesen (wie es
  die Mehrheit der anderen Handler in dieser Datei tut) statt ihn zu ignorieren
  bzw. hart auf uuid.Nil zu setzen. Bewusst NICHT in dieser Unit gefixt
  (Coverage-Units bauen keine Verhaltensaenderungen); beide Funde sind reale,
  mit Tests reproduzierte Befunde, keine Vermutungen.
  .planning/backend-block/loop/run-loop.ps1 traegt weiterhin denselben
  unstaged -StartNotBefore-Diff wie in den Iterationen 6-17 vermerkt -- nicht
  meine Datei, nicht angefasst, nicht committet.
  Laufkontext-Block (Iterationsnummer/Zeitstempel) war in diesem Prompt nicht
  sichtbar mitgeliefert -- Nummer aus der letzten Journal-Ueberschrift
  (Iteration 17) fortgezaehlt, Zeitstempel per date auf dem Loop-Rechner
  ermittelt (2026-08-10 17:59).

## Iteration 19 — b-cov-server-calendar-resources-bookingpages-public — done — 2026-08-10 18:13
- commit: 9dff16aa
- gebaut: neue Testdatei calendar_grpc_resources_bookingpages_test.go mit
  stubResourceRepo (implementiert resource.Repository vollstaendig),
  stubHolidayRepo (implementiert holiday.Repository) und stubBookingRepo
  (implementiert calendar.BookingRepository ueber Interface-Embedding —
  GetCalendarEventsInRange/GetBookedSlotsForPage bleiben unimplementiert,
  weil ihr Rueckgabetyp calendar.eventSlot unexported ist und von diesem
  Package aus nicht benennbar ist; alle Testfaelle kehren vor dem ersten
  Aufruf dieser beiden Methoden zurueck, siehe Kommentar im Code). Je ein
  Happy-Path- und mindestens ein Fehlerpfad-Test fuer alle 20 Scope-Methoden:
  CreateResource, GetResource, ListResources, UpdateResource, DeleteResource,
  ListResourceAvailability, BookResource (inkl. Konflikt-mit-Alternativen-Pfad),
  ListResourceBookings, ListHolidays, SeedHolidays, GetCalendarPreferences,
  UpdateCalendarPreferences, CreateBookingPage (inkl. bookingService==nil ->
  Unimplemented), GetBookingPage, ListBookingPages, UpdateBookingPage,
  DeleteBookingPage, GetPublicBookingPage, GetAvailability (Validierungs- und
  Page-not-found-Pfad, bewusst ohne Happy-Path wegen der eventSlot-Sperre),
  CreatePublicBooking (Customer-fehlt-, Page-not-found- und der geforderte
  Zeitfenster-Fehlerpfad ErrBookingSlotInPast). stubCalendarRepo (aus
  Iteration 17/18) um ein `prefs`-Feld plus echte GetPreferences/
  UpsertPreferences-Implementierung erweitert (war zuvor No-Op `return nil, nil`
  bzw. `return nil`).
- gate: build ok | vet ok | lint ok (0 issues) | test ok | migration n.a.
  (keine Migration) | rls-smoke n.a. (keine Tabelle/Policy angefasst)
- coverage: internal/server 47,7 % -> 59,1 % (kumulativ ueber alle Iterationen
  dieses Laufs, nicht allein durch diese Unit)
- mutations-probe: `!models.ValidResourceTypes[input.ResourceType]` in
  internal/work/resource/service.go:50 auf `models.ValidResourceTypes[...]`
  gedreht (Negation entfernt) -> TestCreateResource_Success UND
  TestCreateResource_InvalidResourceType beide rot (erster lehnt einen
  gueltigen Typ ab, zweiter akzeptiert einen ungueltigen). Zurueckgedreht,
  `git status` zeigt keinen Diff mehr auf der Datei, Tests wieder gruen.
- verify vorgaenger: sauber. Commit 78685fc7 (Iteration 18) aendert nur
  Testdateien plus BACKLOG.yml/JOURNAL.md — keine Produktionscode-Datei,
  kein neues Proto, keine neue Route, kein neuer RequirePermission-Guard,
  keine neue Tabelle. Keine der acht Fehlerklassen einschlaegig.
- offen: DB-Gate lief mit lokaler kmuhub_app-DB (DATABASE_URL gesetzt),
  aber diese Unit ist reine In-Memory-Stub-Coverage ohne echte DB-Queries —
  nichts, was Luke morgens nachfahren muss. `go test ./internal/gateway/`
  bewusst nicht gelaufen, da keine Route/kein Gateway-Code angefasst wurde.
  .planning/backend-block/loop/run-loop.ps1 traegt weiterhin denselben
  unstaged -StartNotBefore-Diff wie in den Iterationen 6-18 vermerkt --
  nicht meine Datei, nicht angefasst, nicht committet.
  Laufkontext-Block (Iterationsnummer/Zeitstempel) war in diesem Prompt nicht
  sichtbar mitgeliefert -- Nummer aus der letzten Journal-Ueberschrift
  (Iteration 18) fortgezaehlt, Zeitstempel per date auf dem Loop-Rechner
  ermittelt (2026-08-10 18:13).

## Iteration 20 — b-cov-server-email-error-map-accounts-sync — done — 2026-08-10 18:15
- commit: 89b98a85
- gebaut: neue Testdatei email_grpc_accounts_sync_test.go mit stubAccountRepo
  (implementiert account.Repository), fakeVaultEncryptor (implementiert
  account.VaultEncryptor reversibel ohne echte Kryptografie) und
  stubFolderRepo (implementiert message.FolderRepository). newTestEmailAccounts-
  Server() verdrahtet einen echten account.Service und message.Service gegen
  diese Stubs sowie einen echten emailsync.Engine (MessageSyncer/FolderSyncer/
  AttachmentStorer bewusst nil, weil TriggerSync/GetStatus/StopWorker -- die
  einzigen in dieser Unit erreichten Engine-Methoden -- sie nie anfassen;
  StartWorker/Run wird in keinem Testfall ausgeloest, SyncEnabled steht in den
  Fixtures durchweg auf false). mapEmailError vollstaendig tabellengetrieben
  getestet, jedes der 33 Sentinels einzeln gegen den erwarteten gRPC-Code, plus
  nil- und Default-Fall. Je ein Happy-Path- und mindestens ein Fehlerpfad-Test
  fuer alle 12 Scope-Methoden: CreateEmailAccount, GetEmailAccount,
  ListEmailAccounts (inkl. Wire-Shape-Check leere Liste [] statt null),
  UpdateEmailAccount, DeleteEmailAccount, SetDefaultEmailAccount,
  TestEmailConnection (Fehlerpfad ueber echten TCP-Connect-Refused auf
  127.0.0.1:1, kein Netzwerk-Mock noetig), ListFolders, GetFolder, SyncFolders,
  TriggerSync, GetSyncStatus.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (1352 PASS, 0 SKIP) |
  migration n.a. (keine Migration) | rls-smoke n.a. (keine Tabelle/Policy
  angefasst) | go test ./internal/gateway/ bewusst nicht gelaufen (keine
  Route/kein Gateway-Code angefasst)
- coverage: internal/server 47,7 % -> 59,8 % (kumulativ ueber alle Iterationen
  dieses Laufs, nicht allein durch diese Unit)
- mutations-probe: `codes.NotFound` fuer `account.ErrAccountNotFound` in
  mapEmailError (internal/server/email_grpc.go:1432-1433) auf `codes.Internal`
  gedreht -> TestMapEmailError, TestGetEmailAccount_NotFound,
  TestDeleteEmailAccount, TestDeleteEmailAccount_NotFound und
  TestSetDefaultEmailAccount_NotFound alle rot. Zurueckgedreht, `git diff`
  zeigt keinen Rest-Diff mehr auf der Datei, Tests wieder gruen.
- verify vorgaenger: sauber. Commit 9dff16aa (Iteration 19) aendert nur die
  neue Testdatei calendar_grpc_resources_bookingpages_test.go, eine bestehende
  Testdatei und BACKLOG.yml/JOURNAL.md — keine Produktionscode-Datei, kein
  neues Proto, keine neue Route, kein neuer RequirePermission-Guard, keine neue
  Tabelle. Keine der acht Fehlerklassen einschlaegig.
- offen: DB-Gate lief mit lokaler kmuhub_app-DB (DATABASE_URL gesetzt), aber
  diese Unit ist reine In-Memory-Stub-Coverage ohne echte DB-Queries — nichts,
  was Luke morgens nachfahren muss. Waehrend der Recherche aufgefallen:
  ListFolders/GetFolder/SyncFolders in email_grpc.go lesen accountID/folderID
  ohne middleware.GetTenantID(ctx)-Aufruf und ohne tenantID an den Service
  durchzureichen -- anders als fast jeder andere Handler in dieser Datei. Der
  zugrundeliegende PostgresFolderRepository filtert in GetByID/ListByAccount
  ebenfalls nicht nach tenant_id in der WHERE-Klausel (postgres_repository.go:
  344-393) und verlaesst sich vollstaendig auf RLS. Da `knownRLSGaps` leer ist
  und der Standing-Guard das haelt, ist das kein Fund, sondern das erwartete
  Muster dieses Repos -- trotzdem hier vermerkt, falls eine spaetere Iteration
  denselben Code liest und sich wundert, warum kein tenantID-Parameter da ist.
  .planning/backend-block/loop/run-loop.ps1 traegt weiterhin denselben
  unstaged -StartNotBefore-Diff wie in den Iterationen 6-19 vermerkt -- nicht
  meine Datei, nicht angefasst, nicht committet.
  Laufkontext-Block (Iterationsnummer/Zeitstempel) war in diesem Prompt nicht
  sichtbar mitgeliefert -- Nummer aus der letzten Journal-Ueberschrift
  (Iteration 19) fortgezaehlt, Zeitstempel per date auf dem Loop-Rechner
  ermittelt (2026-08-10 18:15).

## Iteration 21 — b-cov-server-email-messages-send-attachments — done — 2026-08-10 18:39
- commit: ee2d0cab
- gebaut: neue Testdatei email_grpc_messages_send_test.go mit stubEmailMessageRepo
  (implementiert message.Repository), stubAttachmentRepo/stubObjectStore (implementieren
  attachment.Repository/attachment.ObjectStore), stubLinkRepo (implementiert
  EmailContactLinkRepository) und stubSendAccountProvider (implementiert
  send.AccountProvider). newEmailMessagesFixture() verdrahtet einen echten message.Service,
  send.Service und attachment.Service gegen diese Stubs. Fuer SendEmail/ReplyEmail/
  ForwardEmail zusaetzlich ein minimaler Fake-SMTP-Server (startFakeSMTPServer/
  serveFakeSMTP, 127.0.0.1:0, keine STARTTLS-Ankuendigung, AUTH PLAIN pauschal akzeptiert),
  damit die Happy-Path-Tests wirklich durch MIME-Bau + SMTP-Dial laufen statt an der
  Credential-Abfrage zu stoppen.
  Je ein Validierungs- oder Happy-Path-Test PLUS mindestens ein Fehlerpfad fuer alle 20
  Methoden aus dem Scope: ListMessages, GetMessage, GetThreadMessages, MarkRead, MarkUnread,
  ToggleStar, MoveToFolder, DeleteMessage, BulkMessageAction, SendEmail, SaveDraft,
  ReplyEmail, ForwardEmail, UploadAttachment, GetAttachmentDownloadURL,
  GetEmailContactLinks, LinkEmailToContact, UnlinkEmailFromContact, GetContactEmails,
  SetReadFlag. BulkMessageAction deckt den geforderten Teilfehler-Fall ab (eine von drei
  IDs existiert nicht, die anderen beiden werden verarbeitet, Affected=2). Wire-Shape
  (leere Liste [] statt null) fuer ListMessages/GetEmailContactLinks/GetContactEmails
  geprueft.
  ZWEI echte Produktionsbugs gefunden und NICHT gefixt (Coverage-Units aendern kein
  Verhalten), stattdessen als neue Fix-Units fuer Lauf 9 ans Ende von BACKLOG.yml gehaengt
  (fix-email-send-missing-tenant-id, fix-email-attachment-download-metadata-wrong-message-id)
  und mit je einem dokumentierenden Test hier belegt:
  1) send.Service.Send/SaveDraft (internal/email/send/service.go, ~Z. 203-215 bzw.
     305-318) setzen TenantID nie auf dem gespeicherten models.EmailMessage, obwohl
     SendInput.TenantID/DraftInput.TenantID vorhanden sind. message.PostgresRepository.Create
     schreibt msg.TenantID direkt in die INSERT-Query (postgres_repository.go:41) -- jede
     gesendete/entworfene Mail landet mit tenant_id = Nulluuid in der DB. Bei scharfer RLS
     vermutlich ein fehlschlagender INSERT, sonst ein fuer den Absender unauffindbarer
     Datensatz.
  2) EmailGRPCServer.GetAttachmentDownloadURL (email_grpc.go, ~Z. 1131) laedt die
     Metadaten (filename/content_type/size_bytes) ueber
     attachmentService.GetByMessage(ctx, uuid.Nil, tenantID) -- fest verdrahtetes uuid.Nil
     statt der echten MessageID des Anhangs. Fuer jeden Anhang an einer echten Nachricht
     bleiben die Metadaten leer; nur Pre-Send-Uploads (die UploadAttachment absichtlich mit
     MessageID=uuid.Nil anlegt) funktionieren zufaellig.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (1431 PASS, 0 SKIP in
  internal/server, inkl. internal/server/response) | migration n.a. (keine Migration) |
  rls-smoke n.a. (keine Tabelle/Policy angefasst) | go test ./internal/gateway/ bewusst
  nicht gelaufen (keine Route/kein Gateway-Code angefasst)
- coverage: internal/server 47,7 % -> 61,2 % (kumulativ ueber alle Iterationen dieses
  Laufs, nicht allein durch diese Unit)
- mutations-probe: `!bulkActions[action]` in internal/email/message/service.go:161
  (BulkAction) auf `bulkActions[action]` gedreht (Negation entfernt) ->
  TestBulkMessageAction_UnknownAction, TestBulkMessageAction_MoveRequiresTarget und
  TestBulkMessageAction_PartialFailureSkipsMissing alle drei rot (der erste akzeptiert
  jetzt eine unbekannte Action, die anderen beiden lehnen gueltige Actions ab).
  Zurueckgedreht, `git diff --stat` zeigt keinen Rest-Diff mehr auf der Datei, alle fuenf
  BulkMessageAction-Tests wieder gruen.
- verify vorgaenger: sauber. Commit 89b98a85 (Iteration 20) aendert nur die neue Testdatei
  email_grpc_accounts_sync_test.go plus BACKLOG.yml/JOURNAL.md -- keine
  Produktionscode-Datei, kein neues Proto, keine neue Route, kein neuer
  RequirePermission-Guard, keine neue Tabelle. Keine der acht Fehlerklassen einschlaegig.
- offen: DB-Gate lief mit lokaler kmuhub_app-DB (DATABASE_URL gesetzt), aber diese Unit ist
  reine In-Memory-Stub-Coverage ohne echte DB-Queries -- nichts, was Luke morgens
  nachfahren muss (ein breiter `go test ./internal/...`-Lauf zeigte zwei FAILs in
  internal/biz/recurring wegen "too many clients"/erschoepftem Connection-Pool durch die
  hohe Parallelitaet des Gesamtlaufs -- isoliert mit `go test ./internal/biz/recurring/...`
  sofort gruen, also kein Fund dieser Iteration und nicht mein Paket).
  ZWEI neue Fix-Units fuer Lauf 9 ans Ende von BACKLOG.yml gehaengt, siehe oben unter
  "gebaut" -- beide mit Fundstelle, Reproduktion (Test) und done_when versehen.
  hr-salary-document-category und hr-salary-self-service-route (Block A) stehen im
  Backlog weiterhin auf `done`, aber ohne dass ich sie in dieser Iteration angefasst
  haette -- nur zur Einordnung erwaehnt, kein Handlungsbedarf.
  .planning/backend-block/loop/run-loop.ps1 traegt weiterhin einen unstaged
  -StartNotBefore-Diff (wie in den Iterationen 6-20 vermerkt) -- nicht meine Datei, nicht
  angefasst, nicht committet.
  Laufkontext-Block (Iterationsnummer/Zeitstempel) war in diesem Prompt nicht sichtbar
  mitgeliefert -- Nummer aus der letzten Journal-Ueberschrift (Iteration 20) fortgezaehlt,
  Zeitstempel per `date` auf dem Loop-Rechner ermittelt (2026-08-10 18:39).

## Iteration 22 — b-cov-server-email-signatures-rules-labels-templates — done — 2026-08-10 18:41
- commit: cfb6bd47
- gebaut: neue Testdatei email_grpc_signatures_rules_labels_templates_test.go mit
  stubSignatureRepo (signature.Repository), stubRuleRepo (rule.Repository, folder/label
  Membership tenant-gescopt wie das Repo-Vorbild internal/email/rule/service_test.go),
  stubLabelRepo (label.Repository) + stubLabelMessageReader (label.MessageReader) und
  stubTemplateRepo (template.Repository, Visibility-Filter wie die echte SQL-Query).
  newEmailSigRuleLabelTplFixture() verdrahtet echte signature.Service/rule.Service/
  label.Service/template.Service dagegen. mapEmailError deckt alle Sentinels dieser vier
  Pakete bereits vollstaendig ab (TestMapEmailError in email_grpc_accounts_sync_test.go,
  gegengeprueft) -- hier ausschliesslich Validierungs-/Happy-Path pro Methode, wie im
  Scope verlangt.
  Je ein Validierungs- oder Happy-Path-Test PLUS mindestens ein Fehlerpfad fuer alle 22
  Methoden aus dem Scope: CreateSignature, GetSignature, ListSignatures, UpdateSignature,
  DeleteSignature, SetDefaultSignature, ListEmailRules, CreateEmailRule, UpdateEmailRule,
  DeleteEmailRule, ApplyEmailRules, ListEmailLabels, CreateEmailLabel, UpdateEmailLabel,
  DeleteEmailLabel, AssignMessageLabels, ListEmailTemplates, GetEmailTemplate,
  CreateEmailTemplate, UpdateEmailTemplate, DeleteEmailTemplate, RenderEmailTemplate.
  Wire-Shape (leere Liste [] statt null) fuer ListSignatures/ListEmailRules/ListEmailLabels
  geprueft. ApplyEmailRules-Happy-Test folgt dem verifizierten Muster aus
  rule/service_test.go (Affected/Scanned-Zaehlung ueber zwei Kandidaten, nur einer matcht).
  BEFUND zu done_when "RenderEmailTemplate prueft einen Fehlerpfad fuer ein unbekanntes
  Platzhalter-Feld": es gibt keinen solchen Fehlerpfad im Code. template.Service.Render
  (internal/email/template/service.go:172-181) iteriert NUR ueber AllowedPlaceholders und
  liest values[key] optional -- ein values-Schluessel ausserhalb der Allowlist wird
  stillschweigend ignoriert, kein Fehler, kein Panic, per Kommentar im Quellcode so gewollt
  ("only the fixed allow-list is ever looked up"). Kein Bug, keine neue Fix-Unit noetig --
  stattdessen TestRenderEmailTemplate_UnknownPlaceholderKeyIgnored geschrieben, das dieses
  tatsaechliche Verhalten dokumentiert (unbekannter Key hat keine Wirkung, {{token}} bleibt
  im Output stehen), plus TestRenderEmailTemplate_InvalidID als der eigentliche Fehlerpfad
  (ungueltige Template-ID -> InvalidArgument).
- gate: build ok (internal/server, internal/gateway, cmd/email, cmd/gateway -- cmd/server
  existiert nicht, internal/server wird von mehreren cmd/*-Binaries importiert) | vet ok |
  lint ok (0 issues) | test ok (1477 PASS, 0 SKIP, 0 FAIL in internal/server, inkl.
  internal/server/response) | migration n.a. (keine Migration) | rls-smoke n.a. (keine
  Tabelle/Policy angefasst) | go test ./internal/gateway/ bewusst nicht gelaufen (keine
  Route/kein Gateway-Code angefasst)
- coverage: internal/server 47,7 % -> 62,2 % (kumulativ ueber alle Iterationen dieses
  Laufs, nicht allein durch diese Unit)
- mutations-probe: in internal/email/template/service.go:130-131 die Zeile
  `tpl.OwnerID = nil` (Owner-Clearing beim Wechsel auf visibility=shared) durch einen
  Kommentar ersetzt -> TestUpdateEmailTemplate_SwitchToSharedClearsOwner rot (erwartete
  leere OwnerId, bekam die alte User-ID). Zurueckgedreht, `git diff --stat` auf der Datei
  zeigt keinen Rest-Diff mehr, Test wieder gruen.
- verify vorgaenger: sauber. Commit ee2d0cab (Iteration 21) aendert nur die neue Testdatei
  email_grpc_messages_send_test.go plus BACKLOG.yml/JOURNAL.md -- keine
  Produktionscode-Datei, kein neues Proto, keine neue Route, kein neuer
  RequirePermission-Guard, keine neue Tabelle. Keine der acht Fehlerklassen einschlaegig.
  (cc649891 direkt danach ist nur der Chore-Commit, der den Commit-Hash im Journal
  nachtraegt.)
- offen: DB-Gate lief mit lokaler kmuhub_app-DB (DATABASE_URL gesetzt), aber diese Unit ist
  reine In-Memory-Stub-Coverage ohne echte DB-Queries -- nichts, was Luke morgens
  nachfahren muss.
  .planning/backend-block/loop/run-loop.ps1 traegt weiterhin einen unstaged
  -StartNotBefore-Diff (wie in den Iterationen 6-21 vermerkt) -- nicht meine Datei, nicht
  angefasst, nicht committet.
  Laufkontext-Block (Iterationsnummer/Zeitstempel) war in diesem Prompt nicht sichtbar
  mitgeliefert -- Nummer aus der letzten Journal-Ueberschrift (Iteration 21) fortgezaehlt,
  Zeitstempel per `date` auf dem Loop-Rechner ermittelt (2026-08-10 18:41).

## Iteration 23 — b-cov-server-chat-channels-messages — done — 2026-08-10 19:00
- commit: -
- gebaut: neue Testdatei chat_grpc_channels_messages_test.go mit stubChannelRepo
  (channel.Repository) und stubChatMessageRepo (message.Repository, umbenannt gegen
  Kollision mit dem bereits existierenden stubMessageRepo aus inbox_grpc_test.go),
  beide adaptiert von den geprueften MockRepository-Implementierungen aus
  internal/chat/channel/service_test.go bzw. internal/chat/message/service_test.go.
  chatChannelsMessagesFixture verdrahtet echte channel.Service/message.Service dagegen
  (NewChatGRPCServer mit nil fileService/searchService/reactionService/bookmarkService,
  da ausserhalb des Scopes). seedChannel/addMemberBoth/addUserBoth halten beide Repos
  konsistent, weil Channel- und Message-Service in Produktion unabhaengige Repos sind.
  mapChatError war laut Scope bereits vollstaendig getestet (TestMapChatError) -- hier
  ausschliesslich Validierungs-/Happy-Path plus mindestens ein Fehlerpfad pro Methode
  fuer alle 20 Methoden aus dem Scope: CreateChannel, GetChannel, ListChannels,
  UpdateChannel, DeleteChannel, ArchiveChannel, JoinChannel, LeaveChannel,
  GetChannelMembers, UpdateMemberRole (inkl. done_when "Demotion des letzten Owners":
  UpdateMemberRole blockt JEDE Rollenaenderung eines Owner-Targets ueber
  ErrCannotChangeOwner/FailedPrecondition, unabhaengig davon ob weitere Owner existieren
  -- der Test macht das im Kommentar explizit, da der Service keine "letzter Owner"-
  Zaehlung kennt), SendMessage, GetMessages, UpdateMessage, DeleteMessage, GetOrCreateDM,
  ListDMs, GetThreadReplies, MarkChannelRead, GetUnreadCounts, GetUserMentions.
  Testfunktionen mit "Chat"-Praefix benannt (TestChatCreateChannel etc.), weil
  TestDeleteMessage bereits in email_grpc_messages_send_test.go existiert.
- gate: build ok (internal/server, cmd/gateway) | vet ok | lint ok (0 issues) | test ok
  (1497 PASS, 0 SKIP, 0 FAIL in internal/server; internal/chat/... komplett gruen) |
  migration n.a. (keine Migration) | rls-smoke n.a. (keine Tabelle/Policy angefasst) |
  go test ./internal/gateway/ bewusst nicht gelaufen (keine Route/kein Gateway-Code
  angefasst)
- coverage: internal/server 47,7 % -> 63,5 % (kumulativ ueber alle Iterationen dieses
  Laufs, nicht allein durch diese Unit)
- mutations-probe: in internal/chat/channel/service.go:349 die Owner-Leave-Sperre
  `if membership.Role == models.ChannelRoleOwner` durch `if false && ...` entschaerft ->
  TestChatLeaveChannel/owner_cannot_leave rot (erwartete FailedPrecondition, bekam nil).
  Zurueckgedreht, `git diff --stat` auf der Datei zeigt keinen Rest-Diff mehr, Test
  wieder gruen.
- verify vorgaenger: sauber. Commit cfb6bd47 (Iteration 22) aendert nur die neue
  Testdatei email_grpc_signatures_rules_labels_templates_test.go plus
  BACKLOG.yml/JOURNAL.md -- keine Produktionscode-Datei, kein neues Proto, keine neue
  Route, kein neuer RequirePermission-Guard, keine neue Tabelle. Keine der acht
  Fehlerklassen einschlaegig. (56e64e49 direkt danach ist nur der Chore-Commit, der den
  Commit-Hash im Journal nachtraegt.)
- offen: DB-Gate lief mit lokaler kmuhub_app-DB (DATABASE_URL gesetzt), aber diese Unit
  ist reine In-Memory-Stub-Coverage ohne echte DB-Queries -- nichts, was Luke morgens
  nachfahren muss.
  .planning/backend-block/loop/run-loop.ps1 traegt weiterhin einen unstaged
  -StartNotBefore-Diff (wie in den Iterationen 6-22 vermerkt) -- nicht meine Datei, nicht
  angefasst, nicht committet.
  Laufkontext-Block (Iterationsnummer/Zeitstempel) war in diesem Prompt nicht sichtbar
  mitgeliefert -- Nummer aus der letzten Journal-Ueberschrift (Iteration 22)
  fortgezaehlt, Zeitstempel per `date` auf dem Loop-Rechner ermittelt (2026-08-10 19:00).

## Iteration 24 — b-cov-server-chat-files-search-reactions — done — 2026-08-10 19:10
- commit: 1193d990
- gebaut: neue Testdatei chat_grpc_files_search_reactions_test.go. Anders als bei den
  vorigen Chat-Units sind fileService/searchService/reactionService/bookmarkService als
  konkrete `*file.Service`/`*search.Service`/`*reaction.Service`/`*bookmark.Service`
  typisiert (keine Interfaces am ChatGRPCServer) -- also vier neue In-Memory-Stubs gegen
  die jeweiligen Repository-Interfaces gebaut: stubChatSearchRepo (chatsearch.Repository,
  umbenannt gegen Kollision mit dem bereits existierenden stubSearchRepo aus
  crm_grpc_activities_reports_consent_test.go) + stubChatDetector (search.Detector),
  stubReactionRepo (reaction.Repository) mit einer summarizeReactions-Hilfsfunktion fuer
  die Aggregation, stubMessageReader (bookmark.MessageReader, structural interface das
  *message.Service strukturell erfuellt) und stubBookmarkRepo (bookmark.Repository).
  fileService laeuft ueber den bereits vorhandenen fileMockRepo aus testhelpers_test.go
  (bislang nur fuer den Upload-Handler-Test genutzt) -- dessen ListChannelFiles-Stub gab
  bislang immer nil,0,nil zurueck; erweitert auf eine echte Filterung nach ChannelID plus
  Uploader-Lookup, weil ListChannelFiles sonst nie eine gefuellte Antwort haette liefern
  koennen. Kein bestehender Test ruft ListChannelFiles auf (nur der Upload-Test nutzt
  fileMockRepo), also kein Rueckwirkungsrisiko.
  Alle 10 Methoden aus dem Scope abgedeckt: GetFileDownloadURL, GetFileThumbnailURL,
  ListChannelFiles, DeleteFile (uploader-darf-loeschen, Nicht-Uploader-ohne-Moderationsrolle
  wird abgelehnt, Channel-Admin-darf-fremde-Datei-loeschen), SearchChat (inkl.
  ErrQueryTooShort -> InvalidArgument -- dieser Sentinel fehlte bislang auch in
  TestMapChatError, ist jetzt indirekt mitgetestet), ToggleReaction (inkl. leeres Emoji ->
  reaction.ErrEmojiRequired -> InvalidArgument, ebenfalls bislang ungetesteter
  mapChatError-Zweig), ListReactions, GetReactionSummary, ToggleBookmark,
  ListBookmarks (inkl. Skip-Verhalten fuer ein Bookmark, dessen Message durch
  bookmark.Service.List uebersprungen wird, wenn der MessageReader
  message.ErrNotChannelMember liefert -- reales Verhalten bei entzogener
  Channel-Mitgliedschaft nach dem Bookmarken).
  toChatSearchResultProto direkt getestet: "nil-Eingabe" aus dem done_when waere ein
  Nil-Pointer-Panic (keine Nil-Guard in der Funktion, kein Aufrufer uebergibt je nil) --
  stattdessen ein volltstaendig gefuellter File-Treffer und ein Message-Treffer mit
  ausschliesslich Pflichtfeldern getestet, um die optionalen Zeiger-Felder in beiden
  Richtungen zu pruefen. toProtoMentionType direkt mit allen drei MentionType-Werten plus
  unbekanntem Fallback-Wert getestet.
- gate: build ok (internal/server, cmd/gateway) | vet ok | lint ok (0 issues) | test ok
  (1521 PASS, 0 SKIP, 0 FAIL in internal/server) | migration n.a. (keine Migration) |
  rls-smoke n.a. (keine Tabelle/Policy angefasst) | go test ./internal/gateway/ bewusst
  nicht gelaufen (keine Route/kein Gateway-Code angefasst)
- coverage: internal/server 47,7 % -> 63,5 % (kumulativ ueber alle Iterationen dieses
  Laufs; `go tool cover -func` zeigt denselben Ein-Dezimalstellen-Wert wie Iteration 23,
  da der Zuwachs dieser Unit innerhalb der Rundungsschwelle von 0,1 Prozentpunkten liegt)
- mutations-probe: in internal/chat/file/service.go:309 die Moderations-Pruefung
  `if !role.CanModerate()` zu `if role.CanModerate()` invertiert ->
  TestChatDeleteFile/non-uploader_without_moderate_role_is_denied UND
  TestChatDeleteFile/channel_admin_can_delete_another_member's_file beide rot (erste
  erwartete PermissionDenied, bekam nil; zweite erwartete nil, bekam PermissionDenied).
  Zurueckgedreht, `git diff --stat` auf der Datei zeigt keinen Rest-Diff mehr, beide Tests
  wieder gruen.
- verify vorgaenger: sauber. Commit de864446 (Iteration 23) aendert nur die neue
  Testdatei chat_grpc_channels_messages_test.go plus BACKLOG.yml/JOURNAL.md -- keine
  Produktionscode-Datei, kein neues Proto, keine neue Route, kein neuer
  RequirePermission-Guard, keine neue Tabelle. Keine der acht Fehlerklassen einschlaegig.
  (56e64e49 direkt davor ist nur der Chore-Commit, der den Commit-Hash im Journal
  nachtraegt.)
- offen: DB-Gate lief mit lokaler kmuhub_app-DB (DATABASE_URL gesetzt), aber diese Unit
  ist reine In-Memory-Stub-Coverage ohne echte DB-Queries -- nichts, was Luke morgens
  nachfahren muss.
  .planning/backend-block/loop/run-loop.ps1 traegt weiterhin einen unstaged
  -StartNotBefore-Diff (wie in den Iterationen 6-23 vermerkt) -- nicht meine Datei, nicht
  angefasst, nicht committet.
  Laufkontext-Block (Iterationsnummer/Zeitstempel) war in diesem Prompt nicht sichtbar
  mitgeliefert -- Nummer aus der letzten Journal-Ueberschrift (Iteration 23)
  fortgezaehlt, Zeitstempel per `date` auf dem Loop-Rechner ermittelt (2026-08-10 19:10).

## Iteration 25 — b-cov-server-dialer-campaigns-queue — done — 2026-08-10 19:18
- commit: e1c6ab1f
- gebaut: neue Testdatei dialer_grpc_campaigns_queue_test.go deckt alle 12
  Methoden aus dem Scope ab (CreateCampaign, GetCampaign, ListCampaigns,
  UpdateCampaign, StartCampaign, PauseCampaign, ArchiveCampaign,
  AddContactsToCampaign, GetNextContact, SkipContact, RequeueContact,
  ListCampaignContacts) je mit Happy-Path- und mindestens einem Fehlerpfad-
  Test (ungueltige UUID, fehlender Tenant/Caller, ungueltiger Status-
  Uebergang). StartCampaign/PauseCampaign decken den ErrCampaignNotDraft/
  ErrInvalidStatusTransition-Pfad ab, ArchiveCampaign den
  ErrInvalidStatusTransition-Pfad von draft aus. Wire-Shape von
  ListCampaigns/ListCampaignContacts explizit geprueft (leere Liste [],
  nicht nil).
  stubCampaignRepo (dialer_grpc_test.go) um ein `nextPending`-Feld erweitert,
  damit GetNextPendingContact einen seedbaren Happy-Path liefert statt immer
  ErrNoContactsAvailable — Muster aus Iteration 24 (bestehenden Stub
  erweitern statt neu bauen) fortgesetzt.
- gate: build ok (internal/server, internal/gateway, cmd/gateway) | vet ok |
  lint ok (0 issues) | test ok (1552 PASS, 0 SKIP, 0 FAIL in internal/server;
  internal/server/response ebenfalls ok) | migration n.a. (keine Migration) |
  rls-smoke n.a. (keine Tabelle/Policy angefasst) | go test
  ./internal/gateway/ bewusst nicht separat gelaufen (keine Route/kein
  Gateway-Code angefasst, aber Build oben deckt ihn ab)
- coverage: internal/server 47,7 % -> 65,1 %
- mutations-probe: in internal/dialer/service.go:226 die Statusprüfung in
  StartCampaign von `if c.Status != CampaignStatusDraft` zu
  `if c.Status == CampaignStatusDraft` invertiert ->
  TestStartCampaign_Happy UND TestStartCampaign_NotDraft beide rot (erste
  erwartete nil, bekam FailedPrecondition; zweite erwartete
  FailedPrecondition, bekam nil). Zurueckgedreht, `git diff --stat` auf der
  Datei zeigt keinen Rest-Diff mehr, beide Tests wieder gruen.
- verify vorgaenger: sauber. Commit 1193d990 (Iteration 24) aendert nur die
  neue Testdatei chat_grpc_files_search_reactions_test.go plus
  testhelpers_test.go (Stub-Erweiterung) und BACKLOG.yml/JOURNAL.md — keine
  Produktionscode-Datei, kein neues Proto, keine neue Route, kein neuer
  RequirePermission-Guard, keine neue Tabelle. Keine der acht Fehlerklassen
  einschlaegig.
- offen: DB-Gate lief mit lokaler kmuhub_app-DB (DATABASE_URL gesetzt), aber
  diese Unit ist reine In-Memory-Stub-Coverage ohne echte DB-Queries —
  nichts, was Luke morgens nachfahren muss.
  .planning/backend-block/loop/run-loop.ps1 traegt weiterhin einen unstaged
  -StartNotBefore-Diff (wie in den Iterationen 6-24 vermerkt) -- nicht meine
  Datei, nicht angefasst, nicht committet.
  Laufkontext-Block (Iterationsnummer/Zeitstempel) war in diesem Prompt nicht
  sichtbar mitgeliefert -- Nummer aus der letzten Journal-Ueberschrift
  (Iteration 24) fortgezaehlt, Zeitstempel per `date` auf dem Loop-Rechner
  ermittelt (2026-08-10 19:18).

## Iteration 26 — b-cov-server-dialer-agents-outcomes-dashboards — done — 2026-08-10 23:11
- commit: e9978680
- gebaut: neue Testdatei dialer_grpc_agents_outcomes_dashboards_test.go deckt
  alle 13 Methoden aus dem Scope ab (SaveCallNotes, CompleteWrapUp,
  SetAgentStatus, GetAgentStatus, GetCampaignAgents, ListCallOutcomes,
  CreateCallOutcome, UpdateCallOutcome, DeleteCallOutcome,
  GetCampaignDashboard, GetAgentDashboard, GetSupervisorOverview,
  GetContactCalls) je mit Happy-Path- und mindestens einem Fehlerpfad-Test.
  Neuer Helper newDialerTestServerWithAgentStore(t) baut den Service mit
  einem echten *dialer.AgentStatusStore auf miniredis (Muster aus
  internal/cache/cache_test.go) statt des bisherigen nil-Stores, weil
  agentStore ein konkreter Typ ist (kein Interface) und
  SetAgentStatus/GetAgentStatus/GetCampaignAgents ihn direkt dereferenzieren
  -- gegen nil waere das ein Panic gewesen. GetSupervisorOverview_Happy laeuft
  bewusst weiter ueber den alten nil-Store-Helper: die
  Active-Agents-Liste ist dort leer, die Schleife, die sonst den Store
  dereferenziert, laeuft also gar nicht an.
  stubOutcomeRepo.Delete (dialer_grpc_test.go) erweitert: loeschte bisher
  nichts wirklich aus der Map, jetzt echtes delete() plus ErrOutcomeNotFound
  bei fehlendem Eintrag -- Grundlage fuer den Happy-Test und den echten
  Fehlerpfad von DeleteCallOutcome.
  ZWEI Abweichungen vom Scope-Text dokumentiert statt stillschweigend
  geaendert:
  (1) "DeleteCallOutcome prueft den Fehlerpfad fuer ein noch referenziertes
      Outcome" trifft nicht mehr zu: Migration 000130 hat fk_dcc_outcome und
      fk_dcs_outcome bewusst auf ON DELETE SET NULL gesetzt (Kommentar in der
      Migration: Outcome-Labels sind tenant-konfigurierbar und muessen loeschbar
      bleiben, ohne Business-/Audit-Daten zu kaskadieren). Ein referenziertes
      Outcome zu loeschen ist also by design erfolgreich, kein Fehlerpfad.
      Getestet ist stattdessen der reale Fehlerpfad: fehlendes/bereits
      geloeschtes Outcome -> NotFound.
  (2) Beim Bauen der SetAgentStatus-Tests gefunden: dialer.ErrInvalidTransition
      (redis_agent_store.go) hat keinen Case in mapDialerError und faellt auf
      codes.Internal mit der generischen Meldung "internal error" -- ein Agent,
      der z. B. von offline direkt auf wrap_up springt, bekommt also einen
      Server-Fehler statt einer FailedPrecondition mit Klartext. Analog zu den
      Fix-Units aus Block A (falscher Code fuer einen unbehandelten Sentinel).
      TestSetAgentStatus_InvalidTransition dokumentiert das aktuelle (falsche)
      Verhalten mit Kommentar im Test -- nicht gefixt, Coverage-Units aendern
      kein Verhalten. Kandidat fuer eine Fix-Unit in Lauf 9.
- gate: build ok (internal/server, internal/gateway, cmd/dialer, cmd/gateway)
  | vet ok | lint ok (0 issues) | test ok (kompletter internal/server-Lauf
  gruen, 0 SKIP, nach Neustart von docker-postgres-1/docker-redis-1 -- siehe
  offen) | internal/server/response ok | migration n.a. (keine Migration) |
  rls-smoke n.a. (keine Tabelle/Policy angefasst) | go test
  ./internal/gateway/ bewusst nicht separat gelaufen (keine Route/kein
  Gateway-Code angefasst, Build oben deckt ihn ab)
- coverage: internal/server 47,7 % -> 65,8 %
- mutations-probe: in dialer_grpc_test.go stubOutcomeRepo.Delete das
  `delete(r.outcomes, id)` entfernt (Stub raeumt dann nichts mehr aus der
  Map) -> TestDeleteCallOutcome_Happy rot (erwartete Outcome entfernt,
  bekam es weiterhin im Store), alle anderen Delete-Tests blieben gruen.
  Zurueckgedreht, `git diff --stat` auf der Datei zeigt nur den
  beabsichtigten Netto-Diff (echtes Delete + ErrOutcomeNotFound-Zweig),
  kein Rest.
- verify vorgaenger: sauber. Commit e1c6ab1f (Iteration 25) aendert nur
  dialer_grpc_campaigns_queue_test.go (neue Testdatei) plus
  dialer_grpc_test.go (Stub-Erweiterung um nextPending) und
  BACKLOG.yml/JOURNAL.md -- keine Produktionscode-Datei, kein neues Proto,
  keine neue Route, kein neuer RequirePermission-Guard, keine neue Tabelle.
  Keine der acht Fehlerklassen einschlaegig.
- offen: docker-postgres-1 und docker-redis-1 liefen zu Beginn dieser
  Iteration NICHT (Docker Desktop war komplett aus) -- Docker Desktop
  gestartet, beide Container per `docker start` reaktiviert, Postgres-Health
  vor dem Gate abgewartet. Danach lief das komplette Gate wie gewohnt gegen
  die lokale kmuhub_app-DB. Falls das oefter passiert: pruefen, ob Docker
  Desktop bei diesem Rechner automatisch mit Windows startet.
  Finding (2) oben (dialer.ErrInvalidTransition -> codes.Internal statt
  FailedPrecondition) ist ein Kandidat fuer eine Fix-Unit in Lauf 9, analog
  zu fix-plugin-createmanifest-missing-tenant-wrong-code aus Block A.
  Ausserdem beim Lesen von GetAgentStatus aufgefallen (nicht getestet, um
  keinen Panic im Testbinary auszuloesen): AgentStatusStore.GetStatus gibt
  bei fehlendem Redis-Key (nil, nil) zurueck, und agentStatusToProto(nil)
  dereferenziert a.UserID ungeprueft -- ein Agent, der noch nie einen Status
  gesetzt hat, loest also einen Nil-Pointer-Panic aus. In Produktion faengt
  der RecoveryUnaryInterceptor das ab (codes.Internal statt Prozessabsturz),
  trotzdem ein zweiter Kandidat fuer Lauf 9.
  .planning/backend-block/loop/run-loop.ps1 traegt weiterhin einen unstaged
  -StartNotBefore-Diff (wie in den Iterationen 6-25 vermerkt) -- nicht meine
  Datei, nicht angefasst, nicht committet.
  Laufkontext-Block (Iterationsnummer/Zeitstempel) war in diesem Prompt nicht
  sichtbar mitgeliefert -- Nummer aus der letzten Journal-Ueberschrift
  (Iteration 25) fortgezaehlt, Zeitstempel per `date` auf dem Loop-Rechner
  ermittelt (2026-08-10 23:11).

## Iteration 27 — b-cov-server-websocket-helpers-ratelimit — done — 2026-08-10 23:21
- commit: 96664cc1
- unit: b-cov-server-websocket-helpers-ratelimit (Block B, Coverage
  internal/server)
- scope: die zehn im Backlog genannten reinen Helfer-/
  Verbindungsverwaltungs-Funktionen in backend/internal/server/websocket.go
  ohne vollen Hub-Aufbau (chatClient, tokenMaker): msgRateLimiter.allow,
  extractToken, ValidateChannelID, wsSubscriptionKey, mustMarshal,
  cacheUserName/getUserName, cleanupPresenceSubscriptions,
  registerConnection/unregisterConnection,
  registerGuestConnection/unregisterGuestConnection.
- was: neue Datei internal/server/websocket_helpers_test.go, 26 Testfaelle,
  je Funktion Normalfall + mindestens ein Grenzfall (Rate-Limit erschoepft,
  Header nur mit Marker ohne Query-Fallback-Wert, ungueltige Channel-ID,
  nicht marshalbarer Kanal-Wert, unbekannter User bei getUserName, User ohne
  jegliche Presence-Subscription, zweite von zwei Verbindungen bleibt nach
  Teil-Unregister erhalten). newTestHub aus websocket_redis_test.go
  wiederverwendet statt eines eigenen Konstruktors.
  Neuer Helper newConnectedWSConnPair(t) baut ein echtes *websocket.Conn-Paar
  ueber httptest.NewServer + coder/websocket (Muster aus
  websocket_revalidate_test.go), weil registerConnection/unregisterConnection
  und die Guest-Pendants echte *websocket.Conn als Map-Key brauchen und
  unregisterConnection intern conn.Close() aufruft. Beide Seiten fahren einen
  passiven Read-Loop (Server: mirrorartig zu handleConnection; Client:
  gleiches Muster) und Cleanup schliesst ueber CloseNow() -- sonst blockiert
  entweder der server- oder der clientseitige Close-Aufruf bis zu
  coder/websockets Default-Close-Handshake-Timeout (in einer ersten Fassung
  ohne Client-Read-Loop 5-10s pro Test, insgesamt ~40s fuer die
  Register/Unregister-Tests; mit Read-Loop + CloseNow < 1s).
- gate: build n.a. (vollstaendiges `go build ./...` scheitert auf diesem
  Windows-Rechner an einem Linker-OOM beim cmd/auth-Binary -- reproduzierbar,
  unabhaengig von dieser Aenderung, siehe offen; stattdessen `go build
  ./internal/server/...` ok + `go vet ./...` (deckt das gesamte Modul ohne
  Linking ab) ok) | vet ok | lint ok (golangci-lint run ./internal/server/...
  -- 0 issues) | test ok (go test -count=1 ./internal/server/... gruen,
  internal/server + internal/server/response, 0 SKIP) | migration n.a.
  (keine Migration) | rls-smoke n.a. (keine Tabelle/Policy angefasst) |
  go test ./internal/gateway/ bewusst nicht separat gelaufen (keine
  Route/kein Gateway-Code angefasst)
- coverage: internal/server 47,7 % -> 66,2 % (lokal gemessen per
  `go test -coverprofile`; Iteration 26 hatte 65,8 % notiert, die kleine
  Differenz stammt allein aus dieser Unit)
- mutations-probe: in websocket.go msgRateLimiter.allow() die Bedingung
  `rl.tokens < 1` auf `rl.tokens < 0` geaendert (Off-by-one: ein Aufruf mit
  exakt 0 Tokens waere danach faelschlich noch erlaubt) ->
  TestMsgRateLimiter_Allow_ExceedsLimit rot (erwartete false, bekam true),
  TestMsgRateLimiter_Allow_WithinLimit und
  TestMsgRateLimiter_Allow_RefillsOverElapsedTime blieben gruen. Zurueckgedreht,
  `git diff --stat internal/server/websocket.go` zeigt keinen Rest.
- verify vorgaenger: sauber. Commit 08836376 (Iteration 26) aendert nur den
  BACKLOG.yml-Status und den Journal-Eintrag von Iteration 26 selbst (Meta-
  Commit "record commit hash") -- keine Produktionscode-Datei, kein neues
  Proto, keine neue Route, kein neuer RequirePermission-Guard, keine neue
  Tabelle. Keine der acht Fehlerklassen einschlaegig.
- offen: `go build ./...` bricht auf diesem Rechner reproduzierbar mit
  einem Linker-Fehler ab ("runtime: cannot allocate memory" waehrend
  cmd/link DWARF-Generierung fuer cmd/auth) -- kein Go-Compile-Fehler,
  sondern ein Speicher-/Ressourcenlimit des Linker-Prozesses auf diesem
  Windows-Host. `go vet ./...` (kein Linking) und paketweise `go build
  ./internal/...`-Aufrufe laufen dagegen sauber durch. Falls kuenftige
  Iterationen einen echten Full-Binary-Build brauchen: pruefen, ob mehr
  Swap/RAM hilft oder ob einzelne cmd/*-Pakete separat gebaut werden
  koennen, um den Linker-Speicherbedarf zu verteilen.
  .planning/backend-block/loop/run-loop.ps1 traegt weiterhin einen unstaged
  -StartNotBefore-Diff (wie in den Iterationen 6-26 vermerkt) -- nicht meine
  Datei, nicht angefasst, nicht committet.
  Laufkontext-Block (Iterationsnummer/Zeitstempel) war in diesem Prompt nicht
  sichtbar mitgeliefert -- Nummer aus der letzten Journal-Ueberschrift
  (Iteration 26) fortgezaehlt, Zeitstempel per `date` auf dem Loop-Rechner
  ermittelt (2026-08-10 23:21).

## Iteration 28 — b-cov-server-websocket-broadcast-notifications — done — 2026-08-10 23:28
- commit: 02c6fc61
- unit: b-cov-server-websocket-broadcast-notifications (Block B, Coverage
  internal/server)
- scope: die 13 im Backlog genannten Broadcast-/Notification-Methoden von
  WebSocketHub in backend/internal/server/websocket.go: sendToUser,
  broadcastToChannel, broadcastToChannelExcept, BroadcastMessageCreated/
  Updated/Deleted, SendNotificationToUser/SendNotificationRead/
  SendNotificationReadAll/SendNotificationUnreadCount,
  BroadcastPresenceUpdate, BroadcastCallIncoming/BroadcastCallEnded,
  BroadcastRecordingStarted, BroadcastReactionToggled und sendError.
- was: neue Datei internal/server/websocket_broadcast_test.go, 20 Testfaelle
  ueber eine echte *websocket.Conn. Neuer Helfer newBroadcastConnPair(t) ist
  eine Variante von newConnectedWSConnPair aus websocket_helpers_test.go
  (Iteration 27) OHNE den client-seitigen Hintergrund-Read-Loop -- der wuerde
  genau die Nachrichten wegkonsumieren, die diese Tests pruefen muessen. Der
  server-seitige Read-Loop bleibt (haelt den httptest-Handler und damit die
  registrierte serverConn am Leben, bis der Client schliesst); geschrieben
  wird in diesen Tests ausschliesslich vom Hub, nie vom Client. Wire-Shape
  fuer drei unterschiedliche Message-Typen geprueft (chatv1.MessageInfo-JSON
  bei BroadcastMessageCreated/-Updated: id/content snake_case aus den
  protoc-gen-go-Struct-Tags; Notification-Payload bei
  SendNotificationToUser: verschachteltes notification-Feld + desktop_push +
  sound; Presence-Payload bei BroadcastPresenceUpdate: user_id + status).
  Vier Grenzfaelle ohne Panic: sendToUser an einen nicht verbundenen User,
  BroadcastPresenceUpdate ohne Abonnenten, BroadcastRecordingStarted ohne
  verbundene User, broadcastToChannelExcept schliesst den ausgeschlossenen
  User aktiv aus (assertNoWSMessage mit 100ms-Timeout-Read).
- gate: build ok (go build -p 2 ./internal/server/... ./internal/gateway/...)
  | vet ok | lint ok (golangci-lint run ./internal/server/... ./internal/
  gateway/... -- 0 issues) | test ok (go test -count=1 ./internal/server/...
  gruen, internal/server + internal/server/response, 0 SKIP) | migration
  n.a. (keine Migration) | rls-smoke n.a. (keine Tabelle/Policy angefasst) |
  go test ./internal/gateway/ bewusst nicht separat gelaufen (keine Route/
  kein Gateway-Code angefasst)
- coverage: internal/server 47,7 % -> 66,6 % (lokal gemessen per
  `go test -coverprofile`; Iteration 27 hatte 66,2 % notiert)
- mutations-probe: in websocket.go broadcastToChannelExcept() die
  Ausschluss-Bedingung `if userID != excludeUserID` entfernt (der
  ausgeschlossene User waere dann faelschlich Empfaenger) ->
  TestBroadcastToChannelExcept_ExcludesGivenUser rot (assertNoWSMessage
  erhielt eine Nachricht statt eines Timeout-Fehlers). Zurueckgedreht,
  `git diff --stat internal/server/websocket.go` zeigt keinen Rest.
- verify vorgaenger: sauber. Commit 96664cc1 (Iteration 27) fuegt
  ausschliesslich internal/server/websocket_helpers_test.go plus
  Journal/Backlog-Metadaten hinzu -- keine Produktionscode-Datei, kein neues
  Proto, keine neue Route, kein neuer RequirePermission-Guard, keine neue
  Tabelle. Keine der acht Fehlerklassen einschlaegig.
- offen: `go build ./...` bricht auf diesem Rechner weiterhin reproduzierbar
  mit einem Linker-OOM ab (siehe Iterationen 25-27) -- unabhaengig von dieser
  Aenderung, stattdessen paketweise `go build ./internal/.../...` genutzt.
  .planning/backend-block/loop/run-loop.ps1 traegt weiterhin einen unstaged
  -StartNotBefore-Diff (wie in den Iterationen 6-27 vermerkt) -- nicht meine
  Datei, nicht angefasst, nicht committet.
  Laufkontext-Block (Iterationsnummer/Zeitstempel) war in diesem Prompt nicht
  sichtbar mitgeliefert -- Nummer aus der letzten Journal-Ueberschrift
  (Iteration 27) fortgezaehlt, Zeitstempel per `date` auf dem Loop-Rechner
  ermittelt (2026-08-10 23:28).

## Iteration 29 — b-cov-server-work-search-links-preferences — done — 2026-08-10 23:39
- commit: c9a33570
- unit: b-cov-server-work-search-links-preferences (Block B, Coverage
  internal/server)
- scope: die 12 im Backlog genannten ungetesteten work_grpc.go-Methoden:
  SearchTasks, GetUserProjectPreference, SetUserProjectPreference,
  AttachFileToTask, RemoveTaskFile, ListTaskFiles, LinkEntityToTask,
  UnlinkEntityFromTask, ListTaskEntityLinks, ListEntityTasks,
  SetTaskCustomFieldValues, GetTaskCustomFieldValues, ListTaskActivities.
- was: `workTaskMockRepo` in work_label_test.go von reinen No-Op-Stubs auf
  echte In-Memory-Implementierungen umgebaut (entityLinks/files/
  customFieldVals/activities als Maps, Search filtert jetzt tatsaechlich
  nach ProjectIDs/AssigneeIDs statt leer zurueckzugeben) und um ein
  `forceErr`-Feld ergaenzt (Muster aus wiki_grpc_test.go/
  errStubWikiRepoFailure uebernommen), damit sowohl die mapWorkError-Pfade
  (Link/Unlink/AttachFile/RemoveFile/SetCustomFieldValues/Preferences) als
  auch die direkten status.Error(Internal)-Pfade (List*, GetCustomFieldValues)
  echte Fehlerfaelle durchlaufen statt nur Happy-Path-Stubs zu treffen. Neue
  Datei work_search_links_test.go, 44 Testfaelle: pro Methode mindestens ein
  Validierungs-/Fehlerpfad (ungueltige UUID, fehlender Tenant, Repo-Fehler,
  Not-Found bei Unlink/RemoveFile) plus Happy Path. SearchTasks deckt die
  geforderte Filterkombination ProjectIds+AssigneeIds (drei Tasks geseedet:
  Treffer, falsches Projekt, kein Assignee -> genau 1 Ergebnis) sowie beide
  Fehlerpfade fuer ungueltige IDs in der Liste. SetUserProjectPreference
  deckt zusaetzlich den Backfill-Zweig (bestehende Preference mit
  TenantID==uuid.Nil bekommt die Tenant-ID aus dem Context nachgetragen).
  golangci-lint schlug fuer die Mock-Erweiterung zwei Simplify-Hinweise vor
  (maps.Copy statt Kopier-Loop, slices.Contains statt Vergleichs-Loop in
  Search) -- beide uebernommen, dadurch 0 Lint-Issues.
- gate: build ok (`go build -p 2 ./internal/server/... ./internal/gateway/...`)
  | vet ok (`go vet ./internal/server/... ./internal/gateway/...`) | lint ok
  (golangci-lint run --config .golangci.yml ./internal/server/... -- 0
  issues) | test ok (`go test -count=1 ./internal/server/` gruen, 0 SKIP) |
  migration n.a. (keine Migration) | rls-smoke n.a. (keine Tabelle/Policy
  angefasst) | go test ./internal/gateway/ bewusst nicht separat gelaufen
  (keine Route/kein Gateway-Code angefasst, nur die bereits erlaubte
  Ausnahme hr-salary-self-service-route betraf Block A, nicht diese Unit)
- coverage: internal/server 47,7 % -> 67,5 % (lokal gemessen per
  `go test -coverprofile=/tmp/cov.out ./internal/server/` +
  `go tool cover -func`; Iteration 28 hatte 66,6 % notiert)
- mutations-probe: in work_grpc.go SearchTasks() die Zeile
  `filters.ProjectIDs = append(filters.ProjectIDs, id)` durch `_ = id`
  ersetzt (Project-ID-Filter waere dann wirkungslos, SearchTasks liefert
  ungefiltert nach Projekt) -> TestSearchTasks_FiltersByProjectAndAssignee
  rot (2 statt 1 Ergebnis: der Task aus dem falschen Projekt blieb drin).
  Zurueckgedreht, `git diff --stat internal/server/work_grpc.go` zeigt
  keinen Rest.
- verify vorgaenger: sauber. Commit 02c6fc61 (Iteration 28) fuegt
  ausschliesslich internal/server/websocket_broadcast_test.go plus
  Journal/Backlog-Metadaten hinzu -- keine Produktionscode-Datei, kein neues
  Proto, keine neue Route, kein neuer RequirePermission-Guard, keine neue
  Tabelle. Keine der acht Fehlerklassen einschlaegig.
- offen: waehrend eines vollen `go test -count=1 ./internal/server/ -v`-Laufs
  (8 Wiederholungen zur Flakiness-Pruefung dieser Unit) ist EINMAL
  `TestGDPRExportRPCs_HappyPathAndDomainErrors` in security_grpc_test.go mit
  "stub: export request not found" -> codes.Internal statt NoError
  fehlgeschlagen (7/8 Laeufe gruen). Auf dem unveraenderten Basis-Commit
  3efc3da6 liefen 3/3 Wiederholungen durch, das ist also keine Regression
  dieser Unit, sondern ein vorbestehender Flake in der GDPR-Export-Testsuite
  (vermutlich zeit-/reihenfolgeabhaengiger Zustand im dortigen Stub-Repo).
  Nicht Teil dieser Unit (kein GDPR-/security-Code angefasst) -- fuer Lauf 9
  als eigene Beobachtung vormerken, falls es sich wiederholt.
  `go build ./...` bricht auf diesem Rechner weiterhin reproduzierbar mit
  einem Linker-OOM ab (siehe Iterationen 25-28) -- unabhaengig von dieser
  Aenderung, stattdessen paketweise `go build ./internal/.../...` genutzt.
  .planning/backend-block/loop/run-loop.ps1 traegt weiterhin einen unstaged
  -StartNotBefore-Diff (wie in den Iterationen 6-28 vermerkt) -- nicht meine
  Datei, nicht angefasst, nicht committet.
  Laufkontext-Block (Iterationsnummer/Zeitstempel) war in diesem Prompt nicht
  sichtbar mitgeliefert -- Nummer aus der letzten Journal-Ueberschrift
  (Iteration 28) fortgezaehlt, Zeitstempel per `date` auf dem Loop-Rechner
  ermittelt (2026-08-10 23:39).

## Iteration 30 — b-cov-server-inbox-message-state-transitions — done — 2026-08-10 23:45
- commit: a8fc540d
- unit: b-cov-server-inbox-message-state-transitions (Block B, Coverage
  internal/server)
- scope: die 12 im Backlog genannten ungetesteten inbox_grpc.go-Methoden:
  MarkRead, MarkUnread, ToggleStar, ArchiveMessage, UnarchiveMessage,
  SnoozeMessage, UnsnoozeMessage, ClaimMessage, AssignMessage,
  GetUnreadCount, BulkMarkRead, BulkArchive.
- gebaut: neue Datei internal/server/inbox_message_state_test.go, 40
  Testfaelle (Praefix `TestInbox*` gewaehlt, weil `TestMarkRead_NotFound`
  bereits in email_grpc_messages_send_test.go fuer die Email-RPC
  gleichen Namens existierte -- Compiler-Fehler DuplicateDecl, per Rename
  behoben). Pro Methode mindestens ein Happy-Path- und ein Fehlerpfad-Test:
  NotFound fuer die einfachen Status-Uebergaenge, SnoozeMessage zusaetzlich
  fehlendes snooze_until und eine Vergangenheits-Zeit (ErrInvalidSnoozeTime),
  UnsnoozeMessage NotFound und ungueltige MessageId. ClaimMessage deckt alle
  vier team.Err*-Sentinels ab (NotTeamMember, ManualClaimInRoundRobin ueber
  round_robin-Assignment-Mode, message.ErrAlreadyAssigned) plus den
  Gateway-eigenen FailedPrecondition-Zweig fuer TeamInboxId==nil. AssignMessage
  deckt AlreadyExists (message.ErrAlreadyAssigned) und ungueltige
  Assignee-/Message-ID. stubMessageRepo.GetUnreadCounts war bislang ein reiner
  `return nil, nil`-Stub (nie fuer echte Zahlen genutzt) -- auf eine
  In-Memory-Aggregation umgebaut, die die Produktionsquery in
  postgres_repository.go:401-421 spiegelt (unread, nicht archiviert, nicht
  aktuell snoozed, gruppiert nach Channel), damit
  TestInboxGetUnreadCount_AggregatesByChannel echte Filterlogik prueft statt
  eines Leerergebnisses. BulkMarkRead/BulkArchive pruefen Happy-Path (2
  Nachrichten, UpdatedCount korrekt, Repo-Zustand tatsaechlich geaendert) und
  den Fehlerpfad fuer eine ungueltige ID in der Liste.
- gate: build ok (`go build -p 2 ./internal/server/... ./internal/gateway/...`)
  | vet ok (`go vet ./internal/server/... ./internal/gateway/...`) | lint ok
  (golangci-lint run --config .golangci.yml ./internal/server/... -- 0
  issues) | test ok (`go test -count=1 ./internal/server/` gruen, `-v` zeigt
  1724 PASS / 0 SKIP / 0 FAIL) | test ok (`go test -count=1
  ./internal/server/...` inkl. response-Unterpaket gruen) | migration n.a.
  (keine Migration) | rls-smoke n.a. (keine Tabelle/Policy angefasst) | go
  test ./internal/gateway/ zusaetzlich gelaufen (keine Route angefasst, aber
  zur Sicherheit geprueft) -- gruen
- coverage: internal/server 47,7 % -> 68,1 % (lokal gemessen per
  `go test -coverprofile=/tmp/cov.out ./internal/server/` +
  `go tool cover -func`; Iteration 29 hatte 67,5 % notiert)
- mutations-probe: in inbox_grpc.go ClaimMessage() die Bedingung
  `if msg.TeamInboxID == nil` durch `if false` ersetzt (der Guard fuer
  Nachrichten ausserhalb einer Team-Inbox waere dann wirkungslos) ->
  TestInboxClaimMessage_NotInTeamInbox rot: statt des erwarteten
  FailedPrecondition-Fehlers ein Nil-Pointer-Panic bei
  `*msg.TeamInboxID` zwei Zeilen weiter (Test schlaegt trotzdem sauber als
  FAIL fehl, testing faengt den Panic als Testfehler ab). Zurueckgedreht,
  `git diff --stat internal/server/inbox_grpc.go` zeigt keinen Rest.
- verify vorgaenger: sauber. Commit c9a33570 (Iteration 29) fuegt
  ausschliesslich internal/server/work_label_test.go,
  internal/server/work_search_links_test.go plus Journal/Backlog-Metadaten
  hinzu -- keine Produktionscode-Datei, kein neues Proto, keine neue Route,
  kein neuer RequirePermission-Guard, keine neue Tabelle. Keine der acht
  Fehlerklassen einschlaegig.
- offen: `go build ./...` bricht auf diesem Rechner weiterhin reproduzierbar
  mit einem Linker-OOM ab (siehe Iterationen 25-29) -- unabhaengig von dieser
  Aenderung, stattdessen paketweise `go build ./internal/.../...` genutzt.
  .planning/backend-block/loop/run-loop.ps1 traegt weiterhin einen unstaged
  -StartNotBefore-Diff (wie in den Iterationen 6-29 vermerkt) -- nicht meine
  Datei, nicht angefasst, nicht committet.
  Laufkontext-Block (Iterationsnummer/Zeitstempel) war in diesem Prompt nicht
  sichtbar mitgeliefert -- Nummer aus der letzten Journal-Ueberschrift
  (Iteration 29) fortgezaehlt, Zeitstempel per `date` auf dem Loop-Rechner
  ermittelt (2026-08-10 23:45).

## Iteration 31 — b-cov-server-hr-leave-lifecycle — done — 2026-08-10 23:53
- commit: 73690839
- unit: b-cov-server-hr-leave-lifecycle (Block B, Coverage internal/server)
- scope: die 10 im Backlog genannten Happy-Path-Luecken des
  Urlaubsantrag-Lebenszyklus in hr_grpc.go: CreateLeaveRequest,
  GetLeaveRequest, ListLeaveRequests, ApproveLeaveRequest,
  RejectLeaveRequest, CancelLeaveRequest, GetLeaveBalance,
  GetEmployeeLeaveBalance, ListLeaveTypes, RecordSickLeave.
- was: neue Datei internal/server/hr_leave_lifecycle_test.go. Da fuer
  leave.Service noch kein Stub-Repository existierte, fuenf neue Stubs nach
  dem Muster von stubFormulareRepo/newFormulareServerWithRepo gebaut:
  stubLeaveRequestRepo (leave.LeaveRequestRepository), stubLeaveBalanceRepo
  (leave.LeaveBalanceRepository -- GetByEmployeeYear liefert bewusst
  (nil, nil) bei Miss, das spiegelt PostgresLeaveBalanceRepo.GetByEmployeeYear
  exakt, dort ist "kein Datensatz" explizit kein Fehler), stubLeaveTypeRepo
  (leave.LeaveTypeRepository), stubLeaveEmployeeRepo (leave.EmployeeRepository,
  liefert employee.ErrEmployeeNotFound bei Miss). Alle vier plus ein
  leave.NewService(...) mit settingsRepo=nil (kein einziger im Test gesetzter
  LeaveType setzt RequiresAUDocument, der Zweig der settingsRepo braucht, wird
  also nie erreicht) sind in newLeaveTestFixtures() gebuendelt, das einen
  echten NewHRGRPCServer(svc, nil, nil, nil, nil, nil) liefert -- die anderen
  vier HR-Services bleiben nil wie in newTestHRServer(), da diese Unit nur den
  Leave-Cluster testet. seedEmployee() setzt StartDate zwei Jahre in die
  Vergangenheit, damit BUrlG-ProRata immer den vollen Jahresanspruch liefert
  (30 Tage) und Balance-Assertions ohne Pro-Rata-Sonderfaelle auskommen.
  22 Testfaelle: pro Methode mindestens ein Happy-Path, zusaetzlich
  CreateLeaveRequest/InvalidDateRange, ListLeaveRequests/MissingTenant,
  ApproveLeaveRequest+RejectLeaveRequest je /AlreadyDecided (bereits
  entschiedener Antrag -> FailedPrecondition, wie im Backlog gefordert),
  CancelLeaveRequest/NotOwner, GetLeaveBalance/InvalidUserID,
  RecordSickLeave/LeaveTypeNotFound. Beim Schreiben zwei falsche Annahmen
  in den Testerwartungen korrigiert, nachdem der erste Testlauf sie aufdeckte:
  ErrInvalidDateRange mappt in mapHRError auf InvalidArgument, nicht
  FailedPrecondition; und ErrLeaveTypeNotFound hat in mapHRError ueberhaupt
  keinen eigenen case (faellt auf den default codes.Internal-Zweig) --
  RecordSickLeave_LeaveTypeNotFound erwartet jetzt Internal, mit Kommentar im
  Test, dass das eine bestehende Luecke in mapHRError ist, keine Regression
  dieser Unit.
- gate: build ok (`go build -p 2 ./internal/server/... ./internal/gateway/...`)
  | vet ok (`go vet ./internal/server/...`) | lint ok (golangci-lint run
  --config .golangci.yml ./internal/server/... -- 0 issues) | test ok
  (`go test -count=1 ./internal/server/` gruen) | migration n.a. (keine
  Migration) | rls-smoke n.a. (keine Tabelle/Policy angefasst) | keine neue
  Route, kein neuer RequirePermission-Guard, keine neue config.RequireX-
  Assertion
- coverage: internal/server 67,4 % -> 67,6 % (lokal gemessen per
  `go test -coverprofile=/tmp/cov.out ./internal/server/` +
  `go tool cover -func`; Basiswert selbst nachgemessen durch kurzzeitiges
  Verschieben der neuen Testdatei, da der in Iteration 30 notierte Wert
  68,1 % beim Nachmessen auf demselben Basis-Commit a8fc540d nicht
  reproduzierbar war -- 67,4 % gemessen. Vermutlich Mess-Rauschen zwischen
  Iterationen, kein Befund dieser Unit. Der Zugewinn faellt klein aus, weil
  hr_grpc.go mit 2000+ Zeilen einer der groessten Handler ist und diese Unit
  gezielt nur den Leave-Cluster (10 von >60 Methoden) abdeckt.)
- mutations-probe: in internal/biz/hr/leave/service.go ApproveLeaveRequest()
  die Bedingung `if req.Status != models.LeaveStatusPending` durch `if false`
  ersetzt (der Statuscheck vor der Genehmigung waere dann wirkungslos, ein
  bereits entschiedener Antrag liesse sich beliebig oft erneut genehmigen) ->
  TestApproveLeaveRequest_AlreadyDecided rot ("expected error with code
  FailedPrecondition, got nil"). Zurueckgedreht, `git diff --stat
  internal/biz/hr/leave/service.go` zeigt keinen Rest.
- verify vorgaenger: sauber. Commit a8fc540d (Iteration 30) fuegt
  ausschliesslich internal/server/inbox_grpc_test.go (Erweiterung),
  internal/server/inbox_message_state_test.go plus Journal/Backlog-Metadaten
  hinzu -- keine Produktionscode-Datei, kein neues Proto, keine neue Route,
  kein neuer RequirePermission-Guard, keine neue Tabelle. Keine der acht
  Fehlerklassen einschlaegig.
- offen: `go build ./...` wurde in dieser Iteration nicht erneut probiert
  (siehe Iterationen 25-30, reproduzierbarer Linker-OOM auf diesem Rechner,
  unabhaengig von dieser Aenderung) -- stattdessen paketweise
  `go build ./internal/.../...` genutzt. mapHRError kennt weder
  leave.ErrLeaveTypeNotFound noch leave.ErrSettingsNotFound (beide fallen auf
  den default Internal-Zweig statt NotFound) -- fuer Lauf 9 als kleine
  Nacharbeit vormerken, kein RequirePermission-/Route-/Migrations-Thema, also
  kein Blocker fuer diesen Lauf. .planning/backend-block/loop/run-loop.ps1
  traegt weiterhin einen unstaged -StartNotBefore-Diff (wie in den
  Iterationen 6-30 vermerkt) -- nicht meine Datei, nicht angefasst, nicht
  committet. Laufkontext-Block (Iterationsnummer/Zeitstempel) war in diesem
  Prompt nicht sichtbar mitgeliefert -- Nummer aus der letzten
  Journal-Ueberschrift (Iteration 30) fortgezaehlt, Zeitstempel per `date`
  auf dem Loop-Rechner ermittelt (2026-08-10 23:53).

## Iteration 32 — b-cov-server-document-folders — done — 2026-08-11 00:02
- commit: 069e8ed9
- unit: b-cov-server-document-folders (Block B, Coverage internal/server)
- scope: die 8 im Backlog genannten Happy-Path-Luecken der Ordner-Operationen
  in document_grpc.go: CreateFolder, GetFolder, ListFolders, UpdateFolder,
  DeleteFolder, GetFolderPath, InitializeUserSpace, InitializeTeamSpace.
- was: neue Datei internal/server/document_folders_test.go. Da fuer
  folder.Service noch kein Stub-Repository existierte, stubDocumentFolderRepo
  (Interface folder.Repository, benannt mit Document-Praefix -- ein
  stubFolderRepo/newStubFolderRepo existiert bereits fuer
  email/message.FolderRepository in email_grpc_accounts_sync_test.go, Namens-
  kollision beim ersten Draft vom Compiler gefunden und korrigiert) nach dem
  Muster von stubFormulareRepo/stubLeaveRequestRepo gebaut. List() spiegelt
  bewusst die Filter-Semantik von PostgresRepository.List
  (postgres_repository.go:46-109): ohne ParentID werden Root-Ordner nur dann
  gefiltert, wenn ein SpaceType-Filter gesetzt ist -- das ist der Pfad, den
  folder.Service.Create fuer seinen Sibling-Namenskonflikt-Check nutzt, ein
  naiverer Stub haette dort falsch-positive Konflikte oder falsche Treffer
  geliefert. newFolderTestServer(repo) baut einen echten folder.NewService(repo)
  in NewDocumentGRPCServer, die anderen fuenf Document-Services bleiben nil
  wie in newTestDocumentServer(). 16 Testfaelle: pro Methode Happy-Path plus
  Fehlerpfad -- CreateFolder/DuplicateNameInSameParent (AlreadyExists),
  GetFolder/WrongTenantNotFound, ListFolders/RepositoryErrorMapsToInternal
  (plus ein Leerlisten-Test, der [] statt null belegt), UpdateFolder+
  DeleteFolder je /SystemFolderFails (FailedPrecondition),
  GetFolderPath/RepositoryErrorMapsToInternal, InitializeUserSpace+
  InitializeTeamSpace je /RepositoryErrorPropagates (Internal, da der
  Sentinel-Fehler unverpackt aus dem Service durchgereicht und im default-
  Zweig von mapDocumentError landet).
- abweichung vom backlog: DeleteFolder hat KEINEN Fehlerpfad fuer "Ordner mit
  noch enthaltenen Dateien/Unterordnern", wie das done_when unterstellte.
  folder.Service.Delete (service.go:199-219) prueft ausschliesslich IsSystem;
  PostgresRepository.Delete (postgres_repository.go:154-172) soft-deleted
  enthaltene Dateien und verlaesst sich fuer Unterordner auf ein FK
  ON DELETE CASCADE -- beides laeuft klaglos durch, kein Guard. Repository
  hat dafuer CountFiles/GetChildren/IsDescendant im Interface, aber
  Service.Delete ruft keine der drei auf (per grep in service.go verifiziert).
  Nach der "Coverage-Units bauen keine Verhaltensaenderungen"-Regel keinen
  Guard nachgezogen, sondern den TATSAECHLICH implementierten Fehlerpfad
  (System-Ordner) getestet und die Abweichung hier + als Kommentarblock direkt
  ueber den beiden Delete-Tests im Code dokumentiert. Fuer Lauf 9: falls das
  fachlich gewollt ist (ein geloeschter Ordner mit Dateien verliert deren
  Sichtbarkeit fuer den User, der sie an keinem "geloescht"-Status mehr findet)
  waere das eine Fix-Unit, keine Coverage-Unit -- Luke muss entscheiden, ob
  Kaskadieren das gewuenschte Verhalten ist.
- weitere beobachtung, nicht gefixt: InitializeUserSpaceResponse/
  InitializeTeamSpaceResponse tragen laut Proto ein root_folder-Feld
  (document.proto:310-321), beide Handler geben aber immer eine leere
  Response zurueck (document_grpc.go:256, :274) -- ein FE-Aufrufer, der den
  neu angelegten Root-Ordner direkt aus der Response lesen wollte, bekommt
  nichts und muesste separat ListFolders aufrufen. Kein RequirePermission-,
  Route- oder Migrations-Thema, daher kein Blocker fuer diesen Lauf; als
  kleine Nacharbeit fuer Lauf 9 vormerken.
- gate: build ok (`go build -p 2 ./internal/server/... ./internal/gateway/...`)
  | vet ok (`go vet ./internal/server/... ./internal/gateway/...`) | lint ok
  (golangci-lint run --config .golangci.yml ./internal/server/...
  ./internal/gateway/... -- 0 issues) | test ok (`go test -count=1
  ./internal/server/` und `./internal/server/...` und `./internal/gateway/`
  alle gruen, 0 uebersprungene Tests per -v-Grep auf SKIP verifiziert) |
  migration n.a. (keine Migration) | rls-smoke n.a. (keine Tabelle/Policy
  angefasst) | keine neue Route, kein neuer RequirePermission-Guard, keine
  neue config.RequireX-Assertion
- coverage: internal/server 47,7 % -> 68,5 % (lokal gemessen per
  `go test -coverprofile=/tmp/cov.out ./internal/server/` +
  `go tool cover -func`; der Bezugswert 47,7 % ist der Lauf-Startwert aus
  `coverage_start:` der Unit, nicht der Zwischenwert 67,4-67,6 % aus
  Iteration 31 -- beide Zahlen sind paket-eigen und damit vergleichbar, der
  grosse Sprung seit Lauf-Start ist die Summe aller Iterationen seit
  coverage_start, nicht allein dieser Unit.)
- mutations-probe: in internal/document/folder/service.go Delete() die
  Bedingung `if folder.IsSystem` durch `if false` ersetzt (der Schutz gegen
  das Loeschen eines System-Ordners waere dann wirkungslos) ->
  TestDeleteFolder_SystemFolderFails rot ("An error is expected but got
  nil"). Zurueckgedreht, `git diff --stat
  internal/document/folder/service.go` zeigt keinen Rest.
- verify vorgaenger: sauber. Commit 73690839 (Iteration 31) fuegt
  ausschliesslich internal/server/hr_leave_lifecycle_test.go plus
  Journal/Backlog-Metadaten hinzu -- keine Produktionscode-Datei, kein neues
  Proto, keine neue Route, kein neuer RequirePermission-Guard, keine neue
  Tabelle. Keine der acht Fehlerklassen einschlaegig.
- offen: `go build ./...` wurde in dieser Iteration nicht erneut probiert
  (siehe Iterationen 25-31, reproduzierbarer Linker-OOM auf diesem Rechner,
  unabhaengig von dieser Aenderung) -- stattdessen paketweise
  `go build ./internal/.../...` genutzt. Die beiden oben genannten
  Beobachtungen (DeleteFolder ohne Children-Guard, leere Initialize*Space-
  Responses) sind Befunde dieser Iteration, keine Fix-Units -- fuer Lauf 9
  vormerken, kein Blocker fuer diesen Lauf. .planning/backend-block/loop/
  run-loop.ps1 traegt weiterhin einen unstaged -StartNotBefore-Diff (wie in
  den Iterationen 6-31 vermerkt) -- nicht meine Datei, nicht angefasst, nicht
  committet. Laufkontext-Block (Iterationsnummer/Zeitstempel) war in diesem
  Prompt nicht sichtbar mitgeliefert -- Nummer aus der letzten Journal-
  Ueberschrift (Iteration 31) fortgezaehlt, Zeitstempel per `date` auf dem
  Loop-Rechner ermittelt (2026-08-11 00:02).

## Iteration 33 — b-cov-server-video-egress-callbacks — done — 2026-08-11 00:10
- commit: (siehe unten, wird nach diesem Journal-Eintrag committet)
- gebaut: neue Datei `video_egress_meetingnotes_test.go` mit 10 Tests fuer die drei laut
  Backlog per grep bestaetigten komplett ungetesteten Methoden aus video_grpc.go:
  `CompleteRecordingByEgress`/`FailRecordingByEgress` (LiveKit-Egress-Webhook, System-
  Kontext ohne Tenant) je mit bekannter Egress-ID (Repo-Update auf completed/failed
  geprueft), unbekannter Egress-ID (NotFound) und leerer Egress-ID (InvalidArgument) —
  ueber `newTestVideoCallServer()`/`recordingMockRepo` aus video_call_grpc_test.go, kein
  neuer Stub noetig. `GetMeetingNotes` mit 5 Tests inkl. einem echten Fund: die Methode
  ruft `meetingService.SaveNotes(ctx, meetingID, userID, tenantID, "", false)` auf, um
  Notizen zu "lesen" (der Service exponiert kein GetNotes direkt) — `SaveNotes` lehnt
  leeren Content aber IMMER ab (`ErrNotesContentRequired`, service.go:679-681), und zwar
  VOR jeder Existenzpruefung des Aufrufers. Der Handler faengt jeden Fehler in denselben
  Fallback (leerer `MeetingNotes`-Stub, `nil`-Error) — dadurch liefert GetMeetingNotes
  fuer ein existierendes Meeting mit gespeicherten Notizen GENAU DASSELBE leere Ergebnis
  wie fuer eine komplett unbekannte meeting_id: 200 OK mit leerem Stub statt NotFound.
  Die Tests pinnen dieses tatsaechliche Verhalten (Kommentarblock im Testfile erklaert es),
  fixen es nicht — reine Coverage-Unit, keine Verhaltensaenderung erlaubt.
- gate: build ok (`go build -p 2 ./internal/server/... ./internal/gateway/...`) | vet ok
  (`go vet ./internal/server/... ./internal/gateway/...`) | lint ok (golangci-lint run
  --config .golangci.yml ./internal/server/... -- 0 issues) | test ok (`go test -count=1
  ./internal/server/` und `./internal/server/...` beide gruen, 0 uebersprungene Tests per
  -v-Grep auf SKIP verifiziert) | migration n.a. (keine Migration) | rls-smoke n.a. (keine
  Tabelle/Policy angefasst) | keine neue Route, kein neuer RequirePermission-Guard, keine
  neue config.RequireX-Assertion
- coverage: internal/server 47,7 % -> 68,7 % (lokal per `go test
  -coverprofile=/tmp/cov.out ./internal/server/` + `go tool cover -func`; Bezugswert
  47,7 % ist der Lauf-Startwert aus `coverage_start:`, nicht der Zwischenwert aus
  Iteration 32 — beide paket-eigen und vergleichbar, der Sprung ist die Summe aller
  Iterationen seit Lauf-Start)
- mutations-probe: zwei Proben, beide gefangen. (1) in `CompleteRecordingByEgress` die
  Bedingung `if req.EgressId == ""` durch `if false` ersetzt (leere Egress-ID wuerde dann
  bis zum Service durchgereicht) -> `TestCompleteRecordingByEgress_EmptyEgressID_
  InvalidArgument` rot (erwartete InvalidArgument, bekam NotFound), zurueckgedreht. (2) in
  `GetMeetingNotes` `if tenantErr != nil` auf `if tenantErr == nil` gedreht (Tenant-Guard
  invertiert) -> alle 5 neuen GetMeetingNotes-Tests rot (drei erwarteten Erfolg bekamen
  Unauthenticated, zwei erwarteten spezifische Fehlercodes bekamen Unauthenticated, einer
  erwartete Unauthenticated bekam nil), zurueckgedreht. `git diff --stat
  backend/internal/server/video_grpc.go` zeigt nach beiden Rueckdrehungen keinen Rest.
- verify vorgaenger: sauber. Commit 069e8ed9 (Iteration 32) fuegt ausschliesslich
  `document_folders_test.go` plus Journal/Backlog-Metadaten hinzu — keine
  Produktionscode-Datei, kein neues Proto, keine neue Route, kein neuer
  RequirePermission-Guard, keine neue Tabelle. Keine der acht Fehlerklassen einschlaegig.
- offen: der GetMeetingNotes-Fund oben (unbekannte meeting_id liefert 200 mit leerem
  Stub statt NotFound, und ein existierendes Meeting mit echten Notizen liefert
  ununterscheidbar denselben leeren Stub) ist ein Kandidat fuer eine Fix-Unit in Lauf 9 —
  entweder `meeting.Service` um ein echtes `GetNotes` erweitern oder den Handler auf
  `repo.GetNotes`/eine neue Service-Methode umstellen, dann `SaveNotes("")` nicht mehr
  missbrauchen. Kein Blocker fuer diesen Lauf, kein RequirePermission-/Route-/
  Migrations-Thema. `.planning/backend-block/loop/run-loop.ps1` traegt weiterhin einen
  unstaged `-StartNotBefore`-Diff (wie in den Iterationen 6-32 vermerkt) — nicht meine
  Datei, nicht angefasst, nicht committet. Laufkontext-Block (Iterationsnummer/
  Zeitstempel) war in diesem Prompt nicht sichtbar mitgeliefert — Nummer aus der letzten
  Journal-Ueberschrift (Iteration 32) fortgezaehlt, Zeitstempel per `date` auf dem
  Loop-Rechner ermittelt (2026-08-11 00:10).

## Iteration 34 — b-cov-server-crm-advisory-protocols — done — 2026-08-11 00:17
- commit: 93905c3c
- gebaut: neue Datei `crm_grpc_advisory_test.go` mit einem In-Memory-Fake fuer
  `advisoryprotocol.Repository` (echter `advisoryprotocol.Service` via `NewService(repo)`, kein
  Handler-Mock) und 38 Subtests fuer alle acht Advisory-Methoden aus `crm_grpc_advisory.go`: Create
  (Unauthenticated x2, InvalidArgument, NotFound-via-mapCRMError, Happy), Get (InvalidArgument,
  NotFound, fremder Tenant als NotFound, Happy), List (InvalidArgument, Tenant-/Contact-Scoping mit
  vier Protokollen aus denen genau zwei zurueckkommen, leere Liste), Update (InvalidArgument x2 fuer
  fehlende ID/Payload, InvalidRiskClass, FailedPrecondition fuer finalisiert, Happy mit Feldabgleich),
  Delete (FailedPrecondition fuer finalisiert inkl. Repo-Rest-Check, Happy mit Entfernungs-Check),
  HandOver (NotFound, idempotenter Re-Call auf bereits finalisiert, Happy mit HandedOverAt-Check),
  GenerateAdvisoryProtocolPDF (NotFound, Happy — rendert echte PDF-Bytes ueber maroto v2 ohne
  gewirten `contactService`, pruft nur den Best-Effort-Zweig) und GetReferralReport (Happy mit
  Feldabgleich, leere Liste). Dazu `TestMapCRMError_AdvisoryProtocol` mit allen vier
  advisoryprotocol-Sentinels (`ErrProtocolNotFound`, `ErrProtocolFinalized`, `ErrContactNotFound`,
  `ErrInvalidRiskClass`) einzeln gegen `mapCRMError`, und ein expliziter
  `TestAdvisoryProtocol_ServiceNotConfigured`-Test fuer den `s.advisoryProtocolService == nil`-Guard.
- gate: build ok (`go build -p 2 ./internal/server/...`) | vet ok (`go vet ./internal/server/...`) |
  lint ok (golangci-lint run --config .golangci.yml ./internal/server/... -- 0 issues) | test ok
  (`go test -count=1 ./internal/server/` gruen, 9 SKIPs sind vorbestehende `_DB`-Integrationstests
  ohne `DATABASE_URL`, 0 sonst uebersprungen; zusaetzlich `go test -count=1 ./internal/gateway/`
  gruen — `TestOpenAPIRouteDrift` unberuehrt, da keine neue Route) | migration n.a. (keine Migration)
  | rls-smoke n.a. (kein echtes Repository/keine Tabelle/Policy angefasst — reines In-Memory-Fake) |
  keine neue Route, kein neuer RequirePermission-Guard, keine neue config.RequireX-Assertion
- coverage: internal/server 47,7 % -> 68,6 % (lokal per `go test -coverprofile=/tmp/cov.out
  ./internal/server/` + `go tool cover -func`; Bezugswert 47,7 % ist der Lauf-Startwert aus
  `coverage_start:`, nicht der Zwischenwert aus Iteration 33 — beide paket-eigen und vergleichbar,
  der Sprung ist die Summe aller Iterationen seit Lauf-Start. Pro-Funktion in
  `crm_grpc_advisory.go`: CreateAdvisoryProtocol 100 %, GetAdvisoryProtocol 91,7 %,
  ListAdvisoryProtocols 86,7 %, UpdateAdvisoryProtocol 93,3 %, DeleteAdvisoryProtocol 90,9 %,
  HandOverAdvisoryProtocol 91,7 %, GenerateAdvisoryProtocolPDF 71,4 %, GetReferralReport 83,3 % —
  alle vorher 0,0 %)
- mutations-probe: in `GetAdvisoryProtocol` die Bedingung `if err != nil` (nach `uuid.Parse(req.Id)`)
  durch `if err == nil` ersetzt (eine gueltige ID wuerde dann als InvalidArgument abgelehnt, eine
  ungueltige durchgereicht) -> vier Subtests von `TestGetAdvisoryProtocol` rot
  (`invalid_id` erwartete InvalidArgument bekam nil-Fortsetzung bis zum Panic-freien Fallthrough,
  `not_found`/`wrong_tenant_is_treated_as_not_found` erwarteten NotFound bekamen InvalidArgument,
  `happy_path` erwartete Erfolg bekam InvalidArgument), nur `missing_tenant` blieb gruen (Guard davor
  greift zuerst). Zurueckgedreht, `git diff --stat internal/server/crm_grpc_advisory.go` zeigt
  keinen Rest, erneuter Testlauf der betroffenen Suite gruen.
- verify vorgaenger: sauber. Commit ae8af517 (Iteration 33) fuegt ausschliesslich
  `video_egress_meetingnotes_test.go` plus Journal/Backlog-Metadaten hinzu — keine
  Produktionscode-Datei, kein neues Proto, keine neue Route, kein neuer RequirePermission-Guard,
  keine neue Tabelle. Keine der acht Fehlerklassen einschlaegig.
- offen: keine neuen Befunde in dieser Iteration — die Advisory-Handler verhalten sich wie
  spezifiziert (insbesondere das Immutability-Verhalten nach `HandOver` und die Idempotenz eines
  wiederholten `HandOver`-Aufrufs). `.planning/backend-block/loop/run-loop.ps1` traegt weiterhin
  einen unstaged `-StartNotBefore`-Diff (wie in den Iterationen 6-33 vermerkt) — nicht meine Datei,
  nicht angefasst, nicht committet. Laufkontext-Block (Iterationsnummer/Zeitstempel) war in diesem
  Prompt nicht sichtbar mitgeliefert — Nummer aus der letzten Journal-Ueberschrift (Iteration 33)
  fortgezaehlt, Zeitstempel per `date` auf dem Loop-Rechner ermittelt (2026-08-11 00:17).

## Iteration 35 — b-cov-server-biz-banking-bexio — done — 2026-08-11 00:24
- commit: 99120f18
- gebaut: neue Datei `biz_grpc_banking_bexio_test.go` mit 21 Testfunktionen fuer die zwei
  komplett ungetesteten 0,0-%-Dateien `biz_grpc_banking.go` (7 Methoden) und `bexio_grpc.go`
  (12 Methoden plus `mapBexioError`). `TestBankingError` und `TestMapBexioError` testen jedes
  Sentinel der jeweiligen Fehler-Tabelle einzeln gegen den erwarteten gRPC-Code (inkl. je einem
  unmapped-Default-Fall -> Internal, und `mapBexioError(nil)` -> nil). Fuer die 7 Banking-Methoden
  und 12 Bexio-Methoden je mindestens ein Validierungspfad (ungueltige tenant_id/id/user_id/
  invoice_id/statement_id/quote_id/entity_type/code) plus fuer Banking zusaetzlich der
  `requireBanking()`-Nil-Guard (Unimplemented). Bewusst lean gehalten wie im Scope-Text
  vorgezeichnet ("bei Zeitdruck zuerst beide Fehler-Tabellen und die Validierungspfade abdecken"):
  `bankingSvc` ist `*banking.Service`, konkret ueber ein `Repository`-Interface verdrahtet — jede
  getestete Validierung schlaegt VOR dem ersten Repository-Zugriff fehl, also reicht
  `banking.NewService(nil, nil, nil, nil)` als Stand-in, ohne ein Fake-Repository zu bauen.
  Gleiches Muster bei `bexioService *bexio.Service` (zehn Konstruktor-Parameter, u. a. Client,
  Repo, ConfigRepo, Vault) — alle 12 getesteten Validierungen greifen, bevor der Handler
  `s.bexioService` ueberhaupt dereferenziert, deshalb reicht ein Zero-Value `BexioGRPCServer{}`
  (nil Service) ohne den kompletten Dependency-Graph aufzubauen. Tiefe Happy-Path-Abdeckung pro
  Methode bleibt wie im Scope vorgemerkt Kandidat fuer Lauf 9. Zusaetzlich `TestParseOptionalUUID`
  fuer den kleinen Helfer (leer -> Nil ohne Fehler, ungueltig -> Fehler, gueltig -> Roundtrip).
- gate: build ok (`go build -p 2 ./internal/server/... ./internal/gateway/...`) | vet ok
  (`go vet ./internal/server/... ./internal/gateway/...`) | lint ok (golangci-lint run --config
  .golangci.yml ./internal/server/... -- 0 issues) | test ok (`go test -count=1 ./internal/server/`
  und `./internal/server/... ./internal/gateway/...` beide gruen, 9 SKIPs sind vorbestehende
  `_DB`-Integrationstests ohne `DATABASE_URL`, 0 sonst uebersprungen per -v-Grep auf SKIP) |
  migration n.a. (keine Migration) | rls-smoke n.a. (kein echtes Repository/keine Tabelle/Policy
  angefasst — reine Validierungspfade vor jedem Repo-Zugriff) | keine neue Route, kein neuer
  RequirePermission-Guard, keine neue config.RequireX-Assertion
- coverage: internal/server 47,7 % -> 69,3 % (lokal per `go test -coverprofile=/tmp/cov.out
  ./internal/server/` + `go tool cover -func`; Bezugswert 47,7 % ist der Lauf-Startwert aus
  `coverage_start:`, nicht der Zwischenwert aus Iteration 34 — beide paket-eigen und vergleichbar,
  der Sprung ist die Summe aller Iterationen seit Lauf-Start. Pro-Funktion in
  `biz_grpc_banking.go`: `requireBanking` 100 %, `bankingError` 100 %, `parseOptionalUUID` 100 %,
  die 7 RPC-Methoden zwischen 35,7 % (ListBankStatements) und 77,8 % (ReconcileBankTransaction) —
  alle vorher 0,0 %. Pro-Funktion in `bexio_grpc.go`: `mapBexioError` 100 %, die 12 RPC-Methoden
  zwischen 16,7 % (ListBexioSyncLogs) und 66,7 % (PushInvoiceToBexio/PushQuoteToBexio) — alle
  vorher 0,0 %. Proto-Konverter (`bankStatementToProto`, `bankTransactionToProto`,
  `bankTransactionsToProto`, `syncStatusToProto`, `syncLogToProto`) und `NewBexioGRPCServer`
  bleiben bei 0,0 % — sie werden nur auf dem Happy-Path erreicht, der bewusst fuer Lauf 9
  zurueckgestellt ist)
- mutations-probe: zwei Proben, beide gefangen. (1) in `ReconcileBankTransaction` die Bedingung
  `if err != nil` nach `parseOptionalUUID(req.GetInvoiceId())` durch `if err == nil` ersetzt (eine
  gueltige invoice_id wuerde dann abgelehnt, eine ungueltige durchgereicht bis zum Service-Call)
  -> `TestReconcileBankTransaction_Validation/invalid_invoice_id` rot (erwartete InvalidArgument,
  bekam Internal — die ungueltige ID lief bis zum nil-Repository durch und `bankingError` fing den
  Panic-freien Fehler im Default-Zweig ab), zurueckgedreht. (2) in `mapBexioError` den
  `bexio.ErrBexioUnauthorized`-Fall von `codes.Unauthenticated` auf `codes.PermissionDenied`
  gedreht -> `TestMapBexioError/unauthorized` rot (erwartete Unauthenticated, bekam
  PermissionDenied), alle anderen Sentinels blieben gruen (eigene Case-Zweige unberuehrt),
  zurueckgedreht. `git diff --stat internal/server/biz_grpc_banking.go internal/server/bexio_grpc.go`
  zeigt nach beiden Rueckdrehungen keinen Rest, erneuter Testlauf gruen.
- verify vorgaenger: sauber. Commit 93905c3c (Iteration 34) fuegt ausschliesslich
  `crm_grpc_advisory_test.go` plus Journal/Backlog-Metadaten hinzu; der direkt folgende
  Metadaten-Commit 454f2aa4 aendert nur `JOURNAL.md`. Keine Produktionscode-Datei, kein neues
  Proto, keine neue Route, kein neuer RequirePermission-Guard, keine neue Tabelle. Keine der acht
  Fehlerklassen einschlaegig.
- offen: keine neuen Produktionsbefunde in dieser Iteration — Banking- und Bexio-Handler
  validieren wie erwartet vor jedem Service-/Repo-Zugriff. Die im Scope selbst vorgezeichnete
  Luecke bleibt bestehen: tiefe Happy-Path-Abdeckung fuer beide Domaenen (inkl. der fuenf
  Proto-Konverter und `NewBexioGRPCServer`) braucht ein Fake-`banking.Repository` bzw. einen
  vollstaendig verdrahteten `bexio.Service` — guter Zuschnitt fuer eine eigene Unit in Lauf 9, wie
  im Scope-Text bereits vermerkt. `.planning/backend-block/loop/run-loop.ps1` traegt weiterhin
  einen unstaged `-StartNotBefore`-Diff (wie in den Iterationen 6-34 vermerkt) — nicht meine
  Datei, nicht angefasst, nicht committet. Laufkontext-Block (Iterationsnummer/Zeitstempel) war
  in diesem Prompt nicht sichtbar mitgeliefert — Nummer aus der letzten Journal-Ueberschrift
  (Iteration 34) fortgezaehlt, Zeitstempel per `date` auf dem Loop-Rechner ermittelt
  (2026-08-11 00:24).

## Iteration 36 — c-cov-gateway-auth-2fa-sessions — done — 2026-08-11 00:30
- commit: 5a27f354
- gebaut: neue Datei `route_auth_2fa_sessions_test.go` mit 24 Testfunktionen (davon 2 mit
  Subtests) fuer alle zwoelf bislang komplett ungetesteten 2FA-/Session-Handler in
  `route_auth.go`: Setup2FA, Verify2FA, Validate2FALogin, Disable2FA,
  RegenerateRecoveryCodes, AdminReset2FA, GetTwoFactorPolicy, UpdateTwoFactorPolicy,
  ListSessions, ListAllSessions, TerminateSession, TerminateAllSessions. Jeder Handler hat
  einen ServiceUnavailable-Testfall (leere Registry). Verify2FA/Disable2FA/
  RegenerateRecoveryCodes zusaetzlich InvalidJSON und MissingCode, Verify2FA zusaetzlich
  WrongCodeLength (5-stelliger Code gegen `validate:"required,len=6"`). Validate2FALogin
  zusaetzlich InvalidJSON, MissingPendingToken, MissingCode. AdminReset2FA zusaetzlich
  NoAdminID (401 vor dem RPC-Versuch, direkter Handler-Aufruf ohne withUserID — der Guard
  liegt inline im Handler, keine Middleware noetig), InvalidJSON und drei Validierungsfaelle
  (fehlende/ungueltige user_id, fehlender reason). UpdateTwoFactorPolicy zusaetzlich
  NoAdminID sowie GracePeriodDays ausserhalb 0-365 (negativ und > 365 als zwei Subtests) und
  RoleName ausserhalb des oneof. ListAllSessions zusaetzlich MissingUserID (400 vor dem
  RPC-Aufruf). TerminateSession zusaetzlich InvalidUUID ueber `validateUUIDParam`. Kein
  Happy-Path getestet — Registry-Client zeigt auf `localhost:0`, ein echter RPC-Erfolg ist in
  diesem Testmuster nicht erreichbar (Vorbild: der Rest von `route_auth_test.go` haelt sich
  an dieselbe Grenze).
- gate: build ok (`go build -p 2 ./internal/gateway/... ./cmd/gateway/...`) | vet ok
  (`go vet ./internal/gateway/...`) | lint ok (golangci-lint run --config .golangci.yml
  ./internal/gateway/... -- 0 issues) | test ok (`go test -count=1 ./internal/gateway/`
  gruen, 0 SKIPs per `-v | grep -c "^--- SKIP"`) | migration n.a. (keine Migration) |
  rls-smoke n.a. (keine Tabelle/Policy angefasst) | keine neue Route (TestOpenAPIRouteDrift
  lief mit, unberuehrt), kein neuer RequirePermission-Guard, keine neue
  config.RequireX-Assertion
- coverage: internal/gateway 34,9 % -> 35,5 % (lokal per `go test -coverprofile=/tmp/cov.out
  ./internal/gateway/` + `go tool cover -func`; Bezugswert 34,9 % ist der Lauf-Startwert aus
  `coverage_start:`. Pro-Funktion: HandleSetup2FA 40,0 %, HandleVerify2FA 61,5 %,
  HandleValidate2FALogin 58,3 %, HandleDisable2FA 61,5 %, HandleRegenerateRecoveryCodes
  61,5 %, HandleAdminReset2FA 68,8 %, HandleGetTwoFactorPolicy 40,0 %,
  HandleUpdateTwoFactorPolicy 68,8 %, HandleListSessions 40,0 %, HandleListAllSessions
  61,5 %, HandleTerminateSession 61,5 %, HandleTerminateAllSessions 36,4 % — alle vorher
  0,0 %. Rest bis 100 % ist durchgehend der unerreichte Happy-Path nach dem `client.<RPC>`-
  Aufruf, siehe "gebaut" oben)
- mutations-probe: zwei Proben, beide gefangen. (1) in `HandleAdminReset2FA` die Bedingung
  `if adminID == ""` durch `if false` ersetzt (fehlende adminID wuerde dann bis zum
  RPC-Aufruf durchgereicht) -> `TestHandleAdminReset2FA_NoAdminID` rot (erwartete 401, bekam
  503 — der Request lief bis zum dummy-`localhost:0`-Client durch und scheiterte dort an der
  Verbindung statt am Guard), zurueckgedreht. (2) in `updateTwoFactorPolicyRequest` das
  `GracePeriodDays`-Tag von `lte=365` auf `lte=999999` gesetzt -> `TestHandleUpdateTwoFactor
  Policy_InvalidGracePeriod/over_max` rot (erwartete 400/validation_failed/grace_period_days,
  bekam 503 vom selben dummy-Client-Effekt), `.../negative` blieb gruen (eigener
  `gte=0`-Zweig unberuehrt), zurueckgedreht. `git diff --stat internal/gateway/
  route_auth.go` zeigt nach beiden Rueckdrehungen keinen Rest, erneuter Testlauf
  (`go test -count=1 ./internal/gateway/`) gruen.
- verify vorgaenger: sauber. Commit 99120f18 (Iteration 35) fuegt ausschliesslich
  `biz_grpc_banking_bexio_test.go` plus Journal/Backlog-Metadaten hinzu — keine
  Produktionscode-Datei, kein neues Proto, keine neue Route, kein neuer
  RequirePermission-Guard, keine neue Tabelle. Keine der acht Fehlerklassen einschlaegig.
- offen: keine neuen Produktionsbefunde in dieser Iteration — alle zwoelf Handler verhalten
  sich wie im Scope beschrieben. Die neun weiteren ungetesteten Pfade in `route_auth.go`
  (Forgot/Reset-Password, Profile/User-Update, Provisioning, Invitations) sind die naechste
  Unit `c-cov-gateway-auth-reset-invitations`, bereits im Backlog vorbereitet.
  `.planning/backend-block/loop/run-loop.ps1` traegt weiterhin einen unstaged
  `-StartNotBefore`-Diff (wie in den Iterationen 6-35 vermerkt) — nicht meine Datei, nicht
  angefasst, nicht committet. Laufkontext-Block (Iterationsnummer/Zeitstempel) war in diesem
  Prompt nicht sichtbar mitgeliefert — Nummer aus der letzten Journal-Ueberschrift
  (Iteration 35) fortgezaehlt, Zeitstempel per `date` auf dem Loop-Rechner ermittelt
  (2026-08-11 00:30).

## Iteration 37 — c-cov-gateway-auth-reset-invitations — done — 2026-08-11 00:36
- commit: 4b1e2918
- gebaut: neue Datei `route_auth_reset_invitations_test.go` mit 26 Testfunktionen fuer die
  neun im Scope genannten Pfade in `route_auth.go`: HandleForgotPassword,
  HandleResetPassword, HandleUpdateProfile, HandleUpdateUser, HandleProvisionTenant,
  HandleListInvitations, HandleAcceptInvitation, HandleCancelInvitation, sowie
  `allowForgotAttempt` direkt als White-Box-Methode. Rate-Limiter: fuenf Versuche im Fenster
  erlaubt, sechster abgelehnt (`TestAllowForgotAttempt_RateLimit`), Gross-/Kleinschreibung
  und Leerzeichen teilen sich einen Bucket (`TestAllowForgotAttempt_Normalization`, `User@X.de`
  vs. ` user@x.de `), abgelaufenes Fenster resettet den Zaehler auf 1
  (`TestAllowForgotAttempt_WindowReset`, Bucket direkt ueber `forgotLimiter.Load` manipuliert).
  HandleForgotPassword zusaetzlich: ServiceUnavailable, InvalidJSON, MissingEmail (400
  Validierung), RateLimited (429 mit `Retry-After: 600`), AlwaysOK (200 trotz RPC-Fehler des
  Dummy-Clients — enumeration-safe). HandleResetPassword: ServiceUnavailable, InvalidJSON,
  MissingToken, ShortPassword. HandleUpdateProfile: ServiceUnavailable, AvatarURLTooLong
  (max=512). HandleUpdateUser: ServiceUnavailable, InvalidUUID (vor decodeAndValidate).
  HandleProvisionTenant: ServiceUnavailable, MissingAdminEmail, InvalidSeatLimit
  (seat_limit=0 gegen `omitempty,min=1`). HandleListInvitations: ServiceUnavailable.
  HandleAcceptInvitation: ServiceUnavailable, MissingToken (400 "token is required" vor
  decodeAndValidate), MissingPassword. HandleCancelInvitation: ServiceUnavailable,
  InvalidUUID. Kein Happy-Path getestet — gleiche Grenze wie in den Vorgaenger-Iterationen zu
  `route_auth.go` (Dummy-Client auf `localhost:0`, echter RPC-Erfolg im Testmuster nicht
  erreichbar).
- gate: build ok (`go build -p 2 ./internal/gateway/... ./cmd/gateway/...`) | vet ok
  (`go vet ./internal/gateway/...`) | lint ok (golangci-lint run --config .golangci.yml
  ./internal/gateway/... -- 0 issues) | test ok (`go test -count=1 ./internal/gateway/`
  gruen, 0 SKIPs per `-v | grep -c "^--- SKIP"`; `go test -count=1 ./internal/gateway/...`
  fuer alle Unterpakete ebenfalls gruen) | migration n.a. (keine Migration) | rls-smoke n.a.
  (keine Tabelle/Policy angefasst) | keine neue Route (TestOpenAPIRouteDrift lief mit,
  834 Routen gegen 836 Pfade, unveraendert gegenueber Iteration 36), kein neuer
  RequirePermission-Guard, keine neue config.RequireX-Assertion
- coverage: internal/gateway 34,9 % -> 35,8 % (lokal per `go test -coverprofile=/tmp/cov.out
  ./internal/gateway/` + `go tool cover -func`; Bezugswert 34,9 % ist der Lauf-Startwert aus
  `coverage_start:`)
- mutations-probe: eine Probe, gefangen. In `allowForgotAttempt` die Rueckgabe
  `return b.count <= forgotRateLimitMax` (beide Vorkommen — Reset-Zweig und Zaehl-Zweig,
  letzterer trug die Bedingung) durch `return true` ersetzt (Ratenlimit wirkungslos) ->
  `TestAllowForgotAttempt_RateLimit`, `TestAllowForgotAttempt_Normalization`,
  `TestAllowForgotAttempt_WindowReset` und `TestHandleForgotPassword_RateLimited` alle vier
  rot (erwarteten `false`/429, bekamen `true`/200), zurueckgedreht. `git diff
  backend/internal/gateway/route_auth.go` zeigt danach keinen Rest, erneuter Testlauf
  (`go test -count=1 ./internal/gateway/`) gruen, 0 SKIPs.
- verify vorgaenger: sauber. Commit 5a27f354 (Iteration 36) fuegt ausschliesslich
  `route_auth_2fa_sessions_test.go` plus Journal/Backlog-Metadaten hinzu — keine
  Produktionscode-Datei, kein neues Proto, keine neue Route, kein neuer
  RequirePermission-Guard, keine neue Tabelle. Keine der acht Fehlerklassen einschlaegig.
- offen: keine neuen Produktionsbefunde in dieser Iteration — alle acht Handler und der
  Rate-Limiter verhalten sich wie im Scope beschrieben. `route_auth.go` hat damit ab dieser
  Iteration keine ungetesteten Handler mehr in den Bloecken "2FA/Sessions" und
  "Reset/Invitations"; verbleibende Luecken (falls vorhanden) liegen ausserhalb des Scopes
  dieser beiden Units. `.planning/backend-block/loop/run-loop.ps1` traegt weiterhin einen
  unstaged `-StartNotBefore`-Diff (wie in den Iterationen 6-36 vermerkt) — nicht meine Datei,
  nicht angefasst, nicht committet. Laufkontext-Block (Iterationsnummer/Zeitstempel) war in
  diesem Prompt nicht sichtbar mitgeliefert — Nummer aus der letzten Journal-Ueberschrift
  (Iteration 36) fortgezaehlt, Zeitstempel per `date` auf dem Loop-Rechner ermittelt
  (2026-08-11 00:36).

## Iteration 38 — c-cov-gdpr-export-handlers — done — 2026-08-11 00:42
- commit: 48f7b1ee
- gebaut: neue Datei `backend/internal/security/gdpr/export_crm_chat_test.go` mit vier Tests
  fuer die beiden groessten bis dahin voellig ungetesteten Export-Handler.
  `TestCRMExportHandler_ExportUserData_Integration`: seedet je eine Contact-, Company- und
  Activity-Zeile (`assigned_to` + `created_by` = Subject, gesetztes `due_date`) in einem
  frischen Tenant und prueft, dass alle drei Listen im JSON erscheinen, dass
  `activity_type::text` als `"call"` und nicht als Enum-OID rendert und dass das nullable
  `due_date` den Scan ueberlebt; danach derselbe Export unter Fremd-Tenant-Kontext (alle drei
  Listen leer, zusaetzlich Substring-Assertion, dass keine der drei UUIDs im Payload steht)
  und ein nicht existierender User. `TestChatExportHandler_ExportUserData_Integration`:
  Channel + eigene Message + Message eines Kollegen im selben Channel/Tenant +
  Channel-Membership (`role='owner'`, `last_read_at` gesetzt); prueft, dass genau die eigene
  Message exportiert wird (Datensparsamkeit — die Kollegen-Message darf nicht im Payload
  auftauchen), dass die Membership ueber `channels.name` gejoint ist, plus Fremd-Tenant- und
  Unknown-User-Fall. `TestCRMExportHandler_QueryError` und `TestChatExportHandler_QueryError`
  decken den Fehlerzweig ab (geschlossener Pool -> Query schlaegt fehl, Fehler traegt das
  Modul-Praefix `crm export:` bzw. `chat export:`) — vorher lief in beiden Handlern kein
  einziger `if err != nil`-Zweig durch einen Test. Zwei Test-Helper: `seedExportUser` und
  `seedChannelMembership` (Letzterer noetig, weil `channel_memberships` einen
  Composite-PK ohne `id`-Spalte hat und `testutil.SeedRow` auf `RETURNING id` besteht).
  Kein Produktionscode veraendert.
- gate: build ok (`go build -p 2 ./internal/security/...`) | vet ok (`go vet
  ./internal/security/...`) | lint ok (golangci-lint run --config .golangci.yml
  ./internal/security/... -- 0 issues) | test ok (`go test -count=1
  ./internal/security/gdpr/` gruen, 42 PASS / 0 SKIP / 0 FAIL per `-v`, also alle
  DB-Integrationstests real gelaufen; `go test -count=1 ./internal/security/...` alle sieben
  Pakete gruen) | migration n.a. (keine Migration) | rls-smoke n.a. als eigener Schritt —
  die Tenant-Isolation ist hier direkt Testgegenstand: der Fremd-Tenant-Export laeuft als
  `kmuhub_app` (NOSUPERUSER NOBYPASSRLS) und liefert nachweislich 0 Zeilen | keine neue
  Route, kein neuer RequirePermission-Guard, keine neue config.RequireX-Assertion
- coverage: internal/security/gdpr 35,9 % -> 42,9 % (beides lokal per `go test
  -coverprofile` + `go tool cover -func` gemessen; der Ausgangswert durch temporaeres
  Herausnehmen der neuen Testdatei ermittelt). Hinweis: das Feld `coverage_start:` der Unit
  nennt "internal/security 47,9 %" — das ist der Aggregat-Wert ueber alle sieben
  security-Unterpakete aus dem CI-Artefakt, nicht der des hier geaenderten Pakets
  `internal/security/gdpr`. Die 35,9 % sind der paket-eigene Bezugspunkt.
- mutations-probe: zwei Proben, beide gefangen. (1) In `ChatExportHandler.ExportUserData`
  die Message-Bedingung `WHERE m.created_by = $1` durch `WHERE m.created_by IS NOT NULL`
  ersetzt (Autor-Scoping weg) -> `TestChatExportHandler_ExportUserData_Integration` rot
  ("should have 1 item(s), but has 2", die Kollegen-Message war mit drin). (2) In
  `CRMExportHandler.ExportUserData` das `out.CreatedCompanies = append(...)` durch `_ = co`
  ersetzt -> `TestCRMExportHandler_ExportUserData_Integration` rot ("[] should have 1
  item(s), but has 0"). Beide per `git checkout -- backend/internal/security/gdpr/export.go`
  zurueckgedreht, `git status` zeigt export.go danach unveraendert, erneuter Lauf `go test
  -count=1 ./internal/security/gdpr/` gruen.
- verify vorgaenger: sauber. Commit 4b1e2918 (Iteration 37) fuegt ausschliesslich
  `backend/internal/gateway/route_auth_reset_invitations_test.go` plus Journal/Backlog-
  Metadaten hinzu — keine Produktionscode-Datei, kein Proto, keine Route, kein
  RequirePermission-Guard, keine neue Tabelle. Keine der acht Fehlerklassen einschlaegig.
- offen: **Befund fuer Lauf 9 (Verhaltens-Divergenz, kein Sicherheitsleck).** Das
  `done_when` dieser Unit erwartete, dass ein nicht existierender User "einen definierten
  Fehler statt eines leeren, aber erfolgreichen Exports" liefert (so haelt es
  `AuthExportHandler` ueber sein `QueryRow` -> `ErrNoRows`). CRM und Chat verhalten sich
  anders: beide scopen ueber `JOIN users u ON u.id = $1`, der bei unbekanntem oder
  RLS-unsichtbarem User schlicht nichts matcht — der Export ist dann leer und **erfolgreich**.
  Ich habe das Ist-Verhalten gepinnt statt es umzubiegen (Coverage-Unit, kein Umbau). Folge in
  Produktion: `ExecuteExport` toleriert Handler-Fehler ohnehin (schreibt `<modul>/_error.txt`
  in die ZIP und macht weiter), d. h. bei einem unbekannten User bekaeme man heute
  `auth/_error.txt` neben leeren `crm/data.json` und `chat/data.json`. Kein Datenleck — der
  Fremd-Tenant-Fall liefert nachweislich 0 Zeilen —, aber inkonsistent. Entscheidung gehoert
  Luke: entweder alle sechs Handler auf "Fehler bei unbekanntem User" vereinheitlichen (dann
  `SELECT 1 FROM users WHERE id=$1`-Vorabpruefung je Handler) oder die Auth-Variante
  angleichen und leere Exporte als Normalfall dokumentieren. Vier der sechs
  ExportUserData-Methoden (work, calendar, sessions, notifications) sind weiterhin ungetestet
  — dafuer existiert im Backlog noch keine Unit.
  `.planning/backend-block/loop/run-loop.ps1` traegt weiterhin einen unstaged
  `-StartNotBefore`-Diff (wie in den Iterationen 6-37 vermerkt) — nicht meine Datei, nicht
  angefasst, nicht committet. Laufkontext-Block (Iterationsnummer/Zeitstempel) war in diesem
  Prompt nicht sichtbar mitgeliefert — Nummer aus der letzten Journal-Ueberschrift
  (Iteration 37) fortgezaehlt, Zeitstempel per `date` auf dem Loop-Rechner ermittelt
  (2026-08-11 00:42).

## Iteration 39 — c-cov-gdpr-erasure-handlers — done — 2026-08-11 00:49
- commit: c1b0ec90
- gebaut: neue Datei `backend/internal/security/gdpr/erasure_crm_chat_test.go` mit sieben Tests
  fuer die vier bis dahin voellig ungetesteten Erasure-Methoden.
  `TestCRMErasureHandler_PreviewErasure_Integration`: Contact + Company + Activity in einem
  frischen Tenant -> `RecordCount == 3`, `ModuleName "crm"`, `Action "anonymize"`; danach
  Re-Read der Activity als Beweis, dass Preview **nicht schreibt** (`assigned_to` und
  `description` unveraendert); dazu Fremd-Tenant (0 — die COUNT-Queries tragen kein
  Tenant-Praedikat, die Grenze ist allein RLS) und unbekannter User (0).
  `TestCRMErasureHandler_ExecuteErasure_Integration`: drei Aktivitaeten (dem Subjekt
  zugewiesen / vom Subjekt erstellt aber einem Kollegen zugewiesen / komplett fremd) plus
  Contact und Company; prueft per direkter DB-Abfrage, dass die erste `assigned_to` UND
  `description` verliert, die zweite ihren Assignee **behaelt** und nur die `description`
  verliert, die dritte voellig unberuehrt bleibt, und dass Contact/Company bewusst erhalten
  bleiben (NOT-NULL-FK auf den anonymisierten User).
  `TestCRMErasureHandler_ExecuteErasure_IgnoresAction`: pinnt, dass `ErasureDelete` nichts
  loescht — beide Handler ignorieren ihren `action`-Parameter vollstaendig und anonymisieren
  immer. Harmlos, solange `Service.ExecuteErasure` die Action aus `PreviewErasure` ableitet
  (`erasure.go` hartkodiert dort "anonymize"), aber eine Falle fuer den naechsten, der einem
  Modul eine echte Delete-Action geben will.
  `TestChatErasureHandler_PreviewErasure_Integration` analog (Message + Membership -> 2,
  Preview schreibt nicht, Fremd-Tenant 0, unbekannter User 0).
  `TestChatErasureHandler_ExecuteErasure_Integration`: Message des Subjekts wird zu
  `"[<Label>]"` mit `is_deleted = true` und gestempeltem `edited_at`, die Message des Kollegen
  im selben Channel bleibt woertlich erhalten, und von zwei Memberships verschwindet genau die
  des Subjekts. `TestCRMErasureHandler_DeadPool` / `TestChatErasureHandler_DeadPool` decken je
  beide Fehlerpfade ab: `PreviewErasure` **schluckt** den DB-Fehler und meldet einen leeren,
  erfolgreichen Preview (`RecordCount 0`), `ExecuteErasure` reicht ihn mit Modul-Praefix
  (`crm erasure:` / `chat erasure:`) durch. Kein Produktionscode veraendert.
- gate: build ok (`go build -p 2 ./internal/security/...`) | vet ok (`go vet
  ./internal/security/...`) | lint ok (golangci-lint run --config .golangci.yml
  ./internal/security/... -- 0 issues) | test ok (`go test -count=1 ./internal/security/gdpr/`
  gruen, 49 Tests, davon **0 SKIP** per `-v` gezaehlt, also alle DB-Integrationstests real
  gelaufen; `go test -count=1 ./internal/security/...` alle sieben Pakete gruen) | migration
  n.a. (keine Migration) | rls-smoke n.a. als eigener Schritt — die Tenant-Isolation ist hier
  direkt Testgegenstand: beide Fremd-Tenant-Previews laufen als `kmuhub_app` (NOSUPERUSER
  NOBYPASSRLS) und liefern nachweislich 0 | keine neue Route, kein neuer
  RequirePermission-Guard, keine neue config.RequireX-Assertion
- coverage: internal/security/gdpr 42,9 % -> 48,0 % (beides lokal per `go test -coverprofile`
  + `go tool cover -func` gemessen, Ausgangswert durch temporaeres Herausnehmen der neuen
  Testdatei). Per Funktion: CRM `PreviewErasure` 0,0 -> 100,0 %, CRM `ExecuteErasure`
  0,0 -> 82,6 %, Chat `PreviewErasure` 0,0 -> 100,0 %, Chat `ExecuteErasure` 0,0 -> 78,9 %.
  Hinweis wie in Iteration 38: das Feld `coverage_start:` der Unit nennt "internal/security
  47,9 %" — das ist der Aggregat-Wert ueber alle sieben security-Unterpakete aus dem
  CI-Artefakt, nicht der des hier geaenderten Pakets.
- mutations-probe: zwei Proben, beide gefangen. (1) In `ChatErasureHandler.ExecuteErasure` das
  Autor-Scoping des Message-UPDATE (`WHERE created_by = $1`) durch `WHERE created_by IS NOT
  NULL` ersetzt -> `TestChatErasureHandler_ExecuteErasure_Integration` rot. (2) In
  `CRMErasureHandler.ExecuteErasure` im zweiten UPDATE das `description = NULL` gestrichen ->
  `TestCRMErasureHandler_ExecuteErasure_Integration` rot mit genau der zugehoerigen
  Assertion ("personal description must be cleared for activities created by the subject").
  Beide per `git checkout -- backend/internal/security/gdpr/erasure.go` zurueckgedreht,
  `git status backend/` zeigt danach nur noch die neue Testdatei als untracked, erneuter Lauf
  `go test -count=1 ./internal/security/gdpr/` gruen.
- verify vorgaenger: sauber. Commit 48f7b1ee (Iteration 38) fuegt ausschliesslich
  `backend/internal/security/gdpr/export_crm_chat_test.go` plus Journal/Backlog-Metadaten
  hinzu — keine Produktionscode-Datei, kein Proto, keine Route, kein RequirePermission-Guard,
  keine neue Tabelle. Keine der acht Fehlerklassen einschlaegig.
- offen: **Realer Bug gefunden, dokumentiert statt gefixt** (Coverage-Unit aendert kein
  Verhalten) — neue Unit `fix-crm-erasure-double-count` am Ende von `BACKLOG.yml` fuer Lauf 9.
  `CRMErasureHandler.ExecuteErasure` zaehlt jede Aktivitaet **doppelt**, die der Betroffene
  sowohl erstellt als auch sich selbst zugewiesen hatte: das erste UPDATE setzt
  `assigned_to = NULL`, das zweite trifft dieselbe Zeile daraufhin ueber seinen
  `assigned_to IS NULL`-Zweig erneut. Die Daten sind danach korrekt (der Anonymisierungs-UPDATE
  ist idempotent) — falsch ist nur die zurueckgegebene Zahl, und die landet ueber
  `Service.ExecuteErasure` als "anonymize: N records" im `modules_affected` des
  `GDPRErasureLog` und geht in den SHA-256-Confirmation-Hash ein, also in den
  Art.-17-Loeschnachweis fuer den Betroffenen. Kein Datenleck, aber eine falsche Zahl in einem
  Compliance-Beleg. Die beiden betroffenen Tests erwarten bewusst das **Ist**-Verhalten (5
  statt 4 bzw. 3 statt 2), jeweils mit Kommentar auf diese Journal-Nummer — beim Fix umstellen,
  nicht loeschen. Zweiter, kleinerer Befund ohne eigene Unit: beide Handler ignorieren ihren
  `action`-Parameter komplett (`ErasureDelete` anonymisiert genauso wie `ErasureAnonymize`);
  aktuell folgenlos, weil `PreviewErasure` fuer beide Module "anonymize" hartkodiert, aber es
  gibt keinen Test und keine Assertion, die das absichert, falls jemand die Preview-Action
  aendert. `.planning/backend-block/loop/run-loop.ps1` traegt weiterhin einen unstaged
  `-StartNotBefore`-Diff (wie in den Iterationen 6-38 vermerkt) — nicht meine Datei, nicht
  angefasst, nicht committet. Laufkontext-Block (Iterationsnummer/Zeitstempel) war in diesem
  Prompt nicht sichtbar mitgeliefert — Nummer aus der letzten Journal-Ueberschrift
  (Iteration 38) fortgezaehlt, Zeitstempel per `date` auf dem Loop-Rechner ermittelt
  (2026-08-11 00:49).

## Iteration 40 — c-cov-gdpr-dsar-search — done — 2026-08-11 00:56
- commit: 1090bd9e
- gebaut: `backend/internal/security/gdpr/dsar_search_test.go` (neu, 448 Zeilen, kein
  Produktionscode veraendert). Vier reine Table-Tests fuer `joinName`, `initials`,
  `boolLabel`, `fieldValueRecord` inklusive der Platzhalter-Kanten (`joinName("","")` -> `—`,
  `initials("","")` -> `?`) und zweier Multi-Byte-Faelle (`Ölaf`/`Ärgerlich` -> `ÖÄ`, Emoji als
  eine Rune) — die belegen, dass `initials` ueber Runen und nicht ueber Bytes schneidet.
  Fuenf DB-Integrationstests fuer `SearchByQuery`:
  `TestSearchByQuery_ContactWithAllModules_Integration` seedet Kontakt + Firma + drei
  Consent-Zeilen + zwei Dialer-Sessions und prueft die volle Ausgabe: Modul-Reihenfolge
  (`CRM Kontakte`, `Einwilligungen`, `Anrufe`), alle sechs CRM-Felder, alle drei
  Consent-Status-Zweige (`Erteilt` / `Verweigert` / `Widerrufen`, wobei die Widerrufs-Zeile
  trotz `granted = true` als widerrufen gilt), das `COALESCE(granted_at, created_at)` in beiden
  Richtungen sowie `COALESCE` auf `duration_seconds`/`notes` (NULL -> `0` und `""`). Dazu
  Treffer per Nachnamen-, per E-Mail- und per zusammengesetztem Vollnamen-Teilstring, ein
  Fremd-Tenant-Read (0), ein Read mit **gefaelschtem** `tenantID`-Argument unter fremdem Ctx
  (ebenfalls 0 — beweist, dass RLS und nicht das Argument die Grenze ist) und ein Nicht-Treffer
  (leerer, nicht-nil Slice). `TestSearchByQuery_ContactWithoutConsentOrCalls_Integration` haelt
  die leicht zu verlierende Eigenschaft fest, dass `consentModule`/`dialerModule` **nil** und
  nicht ein leeres Modul liefern — genau ein Modul beim Kontakt ohne Consent/Anrufe.
  `TestSearchByQuery_MatchesUsers_Integration` deckt `matchUsers` inklusive beider
  `boolLabel`-Zweige (aktiver + inaktiver User) und einem gleichnamigen Fremd-Tenant-User ab,
  der unsichtbar bleiben muss. `TestSearchByQuery_NoMinimumLengthGuard_Integration` und
  `TestSearchByQuery_DeadPool` decken die zwei Fehler-/Randpfade ab.
- gate: build ok (`go build -p 2 ./internal/security/...`) | vet ok (`go vet
  ./internal/security/...`) | lint ok (`golangci-lint run --config .golangci.yml
  ./internal/security/...` -- 0 issues) | test ok (`go test -count=1 ./internal/security/gdpr/`
  gruen; die neuen Tests per `-v` gezaehlt: 31 RUN/PASS-Zeilen, **0 SKIP, 0 FAIL**, also liefen
  alle DB-Integrationstests real gegen die lokale Postgres als `kmuhub_app`; `go test -count=1
  ./internal/security/...` alle sieben Pakete gruen) | migration n.a. | rls-smoke n.a. als
  eigener Schritt — Tenant-Isolation ist hier direkter Testgegenstand (Fremd-Tenant-Suche und
  gefaelschtes Tenant-Argument liefern beide 0 Zeilen unter NOSUPERUSER NOBYPASSRLS) | keine
  neue Route, kein neuer RequirePermission-Guard, keine neue config.RequireX-Assertion
- coverage: internal/security/gdpr 48,0 % -> 59,8 % (beides lokal per `go test -coverprofile` +
  `go tool cover -func` gemessen, Ausgangswert durch temporaeres Herausnehmen der neuen
  Testdatei). Per Funktion, alle vorher 0,0 %: `SearchByQuery` 87,5 %, `matchContacts` 84,6 %,
  `matchUsers` 82,4 %, `consentModule` 87,0 %, `dialerModule` 82,4 %, `fieldValueRecord` /
  `joinName` / `initials` / `boolLabel` je 100,0 %. Die Reste sind die Scan- und
  `rows.Err()`-Fehlerzweige, die ohne Fault-Injection nicht erreichbar sind. Wie in den
  Iterationen 38/39: das Feld `coverage_start:` der Unit nennt "internal/security 47,9 %" —
  das ist der Aggregat-Wert ueber alle sieben security-Unterpakete aus dem CI-Artefakt, nicht
  der des hier geaenderten Pakets.
- mutations-probe: zwei Proben, beide gefangen. (1) In `consentModule` den Zweig
  `if revoked != nil { status = "Widerrufen" }` entfernt -> `..._ContactWithAllModules_...`
  rot, Diff nennt genau die fehlende `"Status": "Widerrufen"`-Zelle. (2) In `matchContacts`
  aus der WHERE-Klausel `OR c.email ILIKE $2` gestrichen -> derselbe Test rot mit der
  Assertion "the email substring must find the same contact". Beide per `git checkout --
  backend/internal/security/gdpr/dsar_search.go` zurueckgedreht; `git status backend/` zeigt
  danach nur noch die neue Testdatei als untracked, erneuter Lauf `go test -count=1
  ./internal/security/gdpr/` gruen.
- verify vorgaenger: sauber. Commit c1b0ec90 (Iteration 39) fuegt ausschliesslich
  `backend/internal/security/gdpr/erasure_crm_chat_test.go` plus Journal/Backlog-Metadaten
  hinzu — keine Produktionscode-Datei, kein Proto, keine Route, kein RequirePermission-Guard,
  keine neue Tabelle, keine Migration. Keine der acht Fehlerklassen einschlaegig.
- offen: **Zwei Befunde dokumentiert statt gefixt** (Coverage-Unit aendert kein Verhalten),
  beide ohne eigene Unit, weil sie eine Produktentscheidung brauchen. (1) `SearchByQuery`
  interpoliert die Nutzereingabe ungeschuetzt in ein ILIKE-Pattern (`"%" + query + "%"`).
  Keine SQL-Injection — die Query ist parametrisiert —, aber LIKE-Wildcards der Eingabe
  bleiben wirksam: eine Suche nach `%a` oder `__` listet bis zu `dsarMaxSubjects` beliebige
  Subjekte des eigenen Tenants auf, statt gezielt eine Person zu finden. Fuer eine
  Art.-15-Auskunftsmaske ist das eine Enumerations-Flaeche innerhalb des eigenen Mandanten
  (der Aufrufer ist Admin, deshalb kein Leak ueber die Tenant-Grenze). Fix waere ein
  Escapen von `%`/`_`/`\` plus `ESCAPE '\'` — vor Launch zu entscheiden. (2) Die
  Mindestlaenge von zwei Zeichen sitzt allein im RPC-Handler
  (`internal/server/security_grpc.go:583`), nicht in `SearchByQuery` selbst; ein leerer Query
  listet dort alles auf. Das ist per Test festgehalten
  (`TestSearchByQuery_NoMinimumLengthGuard_Integration`), damit ein zweiter Aufrufer nicht
  stillschweigend ohne Guard einsteigt. `.planning/backend-block/loop/run-loop.ps1` traegt
  weiterhin einen unstaged `-StartNotBefore`-Diff (wie in den Iterationen 6-39 vermerkt) —
  nicht meine Datei, nicht angefasst, nicht committet. Laufkontext-Block
  (Iterationsnummer/Zeitstempel) war auch in diesem Prompt nicht sichtbar mitgeliefert —
  Nummer aus der letzten Journal-Ueberschrift (Iteration 39) fortgezaehlt, Zeitstempel per
  `date` auf dem Loop-Rechner ermittelt (2026-08-11 00:56).

## Iteration 41 — c-cov-audit-password-policy — done — 2026-08-11 01:17
- commit: 64b0f27b
- gebaut: Drei neue Testdateien. `internal/security/password/policy_test.go` (Mock-Repository,
  keine DB): Service.ValidatePassword deckt MinLength/MinEntropy/alle vier Complexity-Flags je
  Fail- und Pass-Fall plus die volle Kombination, Service.CheckPasswordHistory deckt den
  PreventReuseCount<=0-Kurzschluss (bewiesen ueber einen Repo-Fehler, der bei echtem Aufruf von
  GetPasswordHistory durchschlagen wuerde), Reuse-Treffer per echtem bcrypt-Hash, Kein-Treffer
  sowie alle Repo-Fehlerpfade. RecordPassword, GetPolicy, UpdatePolicy (inkl. serverseitig
  aufgeloester ID gegen eine vorgetaeuschte Caller-ID) je mit Erfolgs- und Fehlerpfad.
  internal/security/password/postgres_repository_test.go (DB-Integration): AddPasswordHistory/
  GetPasswordHistory gegen echte Postgres inkl. server-seitig aufgeloestem tenant_id (direkt
  gegen die Tabelle geprueft, nicht ueber den eigenen Lesepfad), Reihenfolge neueste-zuerst,
  Limit, FK-Fehlerpfad bei unbekanntem User; GetPolicy-Fallback auf defaultPolicy() bei
  fehlender Zeile mit allen Feldern geprueft. internal/security/audit/postgres_repository_test.go:
  GetLastHash nicht-leerer Zweig echt getestet; List/VerifyChain konnten NICHT wie im
  done_when vorgesehen getestet werden (siehe offen:) — stattdessen ein Regression-Pin-Test,
  der den gefundenen Bug exakt reproduziert, plus ein sicherer No-Match/Pagination-Defaults-
  Test ohne Zeilen-Scan.
- gate: build ok (go build -p 2 ./internal/security/...) | vet ok | lint ok (golangci-lint
  run --config .golangci.yml ./internal/security/... -- 0 issues) | test ok (go test -count=1
  ./internal/security/audit/ und ./internal/security/password/ beide gruen, -v gezaehlt: 0
  SKIP ueber beide Pakete, alle DB-Integrationstests liefen real gegen die lokale Postgres als
  kmuhub_app) | migration n.a. | rls-smoke n.a. (keine Tabelle/Policy angefasst) | keine neue
  Route, kein neuer RequirePermission-Guard, keine neue config.RequireX-Assertion
- coverage: internal/security/password 21,4 % -> 96,9 % | internal/security/audit 44,4 % ->
  66,7 % (beide lokal per go test -coverprofile + go tool cover -func, Ausgangswert durch
  temporaeres Herausnehmen der neuen Testdateien gemessen). Das Feld coverage_start: der Unit
  nennt "internal/security 47,9 %" — das ist der Aggregat-Wert ueber alle sieben
  security-Unterpakete aus dem CI-Artefakt, nicht der der beiden hier geaenderten Pakete.
- mutations-probe: drei Proben, alle gefangen. (1) In password.Service.CheckPasswordHistory
  PreventReuseCount <= 0 zu < 0 geaendert -> TestCheckPasswordHistory_ReuseDisabled_
  ShortCircuits rot (der praeparierte historyErr schlaegt durch, weil der Kurzschluss nicht
  mehr greift). (2) In password.PostgresRepository.GetPasswordHistory ORDER BY created_at DESC
  zu ASC geaendert -> TestPostgresRepository_PasswordHistory rot mit der erwarteten Reihenfolge
  [hash-newest hash-middle] gegen die tatsaechliche [hash-oldest hash-middle]. (3) In
  audit.PostgresRepository.List den if offset < 0 { offset = 0 }-Clamp entfernt ->
  TestPostgresRepository_List_NoMatch_PaginationDefaultsDontError rot mit "OFFSET must not be
  negative" direkt von Postgres. Alle drei per git checkout -- <datei> zurueckgedreht, git
  status backend/ zeigt danach nur noch die drei neuen Testdateien als untracked, alle drei
  Gates erneut gruen.
  Zusaetzlich validiert (kein Teil der drei Proben oben, aber derselbe Beweis-Mechanismus in
  die andere Richtung): der unten dokumentierte ip_address-Scan-Bug wurde durch einen
  temporaeren Fix (COALESCE(host(ip_address), '') in beiden SELECTs von
  audit/postgres_repository.go) bestaetigt — mit dem Fix liefen die urspruenglich geplanten
  vollen Filter-/Pagination-/Chain-Tests gruen durch, ohne ihn schlagen sie exakt mit der
  dokumentierten Fehlermeldung fehl. Fix per git checkout --
  backend/internal/security/audit/postgres_repository.go zurueckgedreht (Verhaltensaenderung
  ausserhalb des Scopes dieser Coverage-Unit).
- verify vorgaenger: sauber. Commit 1090bd9e (Iteration 40) fuegt ausschliesslich
  backend/internal/security/gdpr/dsar_search_test.go plus Journal/Backlog-Metadaten hinzu —
  keine Produktionscode-Datei, kein Proto, keine Route, kein RequirePermission-Guard, keine
  neue Tabelle, keine Migration. Keine der acht Fehlerklassen einschlaegig.
- offen: Echter, verifizierter Produktionsbug gefunden, nicht gefixt (Coverage-Unit aendert
  kein Verhalten): audit.PostgresRepository.List (Zeile ~170) und .VerifyChain (Zeile ~224)
  scannen die rohe ip_address-Spalte (Typ INET) direkt in models.AuditEntry.IPAddress (string,
  kein Pointer). pgx v5 kann weder ein NULL-INET noch ein gesetztes INET in einen Go-string
  scannen — beide Faelle direkt gegen die lokale Postgres reproduziert ("cannot scan NULL into
  *string" bzw. "cannot scan inet (OID 869) in binary format into *string"). Das heisst: jeder
  Aufruf von List() oder VerifyChain(), der mindestens eine Zeile liefert, schlaegt fehl —
  unabhaengig davon, ob die Zeile eine echte IP traegt oder keine. Betroffen sind damit
  voraussichtlich der Audit-Log-Viewer, der CSV/JSON-Export (ExportEntries ruft intern List)
  und die VerifyAuditChain-RPC fuer JEDEN Tenant mit realer Aktivitaet. Drei andere
  Repositories im selben Repo loesen exakt dasselbe Problem korrekt:
  internal/auth/postgres_repository.go mit COALESCE(host(ip_address), ''),
  internal/crm/consent/postgres_repository.go und internal/formulare/postgres_repository.go
  mit ip_address::text. Der Fix ist eine Ein-Zeilen-Aenderung pro Query (SELECT-Spalte casten),
  am selben Muster wie die drei bestehenden Stellen — verifiziert per temporaerem Patch (siehe
  mutations-probe oben) und wieder zurueckgedreht. Vorschlag fuer Lauf 9: eigene Fix-Unit in
  Block A (fix-audit-list-verifychain-ip-address-scan), sources
  backend/internal/security/audit/postgres_repository.go (Zeilen 170 und 224) plus
  backend/internal/auth/postgres_repository.go (Zeilen 1574/1586/1614) als Vorlage.
  Zusaetzlich musste die geplante Testabdeckung fuer List/VerifyChain aus genau diesem Grund
  umgebaut werden: statt der im done_when vorgesehenen Filter-/Pagination-/Chain-Assertions
  steht jetzt nur ein Regression-Pin-Test (TestPostgresRepository_List_IPAddressScanBug,
  erwartet den Scan-Fehler explizit und dokumentiert im Kommentar, dass er beim Fixen ersetzt
  werden muss) plus ein sicherer No-Match/Pagination-Test ohne Zeilen-Scan. VerifyChain hat
  gar keinen eigenen DB-Test bekommen: audit_log ist seit Migration 000222 DB-seitig
  append-only (BEFORE-UPDATE/DELETE-Trigger, wirkt auch unter System-Context — jeder
  testutil.CleanupRow-Aufruf auf audit_log schlaegt seitdem erwartungsgemaess fehl und laesst
  die Zeile stehen, das ist bereits akzeptiertes, vorbestehendes Verhalten in rls_test.go,
  nicht neu). Ein erster Testentwurf wollte einen Kettenbruch ueber ein direktes UPDATE
  previous_hash simulieren — das waere am Append-Only-Trigger gescheitert und wurde verworfen,
  BEVOR es ausgefuehrt wurde (kein UPDATE hat die Tabelle je erreicht, per Log-Ausgabe der
  fehlgeschlagenen Testlaeufe nachvollzogen). Ein zweiter Entwurf wollte eine bewusst falsch
  verkettete Zeile per testutil.SeedRow einfuegen (INSERT ist vom Trigger nicht betroffen) —
  auch dieser Code wurde nie erreicht (jeder Testlauf brach vorher am Scan-Bug ab), also
  existiert keine von mir eingefuegte kaputte Zeile in audit_log. Beim Validieren mit dem
  temporaeren Fix kam trotzdem einmal VerifyChain(intact) -> valid=false broken=2952 zurueck,
  obwohl meine beiden Testzeilen korrekt verkettet waren: rls_test.go seedet seine Fixtures
  ueber testutil.SeedRow mit hartkodierten entry_hash/previous_hash-Werten (z. B. "aabbcc"/
  ""), die die echte Kette an der jeweiligen Stelle nicht respektieren — das ist
  vorbestehendes Verhalten aus frueheren Laeufen, nicht von mir eingefuehrt, aber es bedeutet,
  dass die globale Hash-Kette dieser lokalen Dev-DB bereits seit laengerem "gebrochen" ist,
  sobald VerifyChain je funktionsfaehig waere. Betrifft nachweislich nur die lokale
  Dev-Postgres, nie Produktion. Fuer die Fix-Unit in Lauf 9 gehoert das als Kontext dazu: nach
  dem Scan-Fix wird VerifyChain vermutlich sofort einen Bruch melden, der auf diese
  Test-Fixtures zurueckgeht, nicht auf echte Manipulation.
  .planning/backend-block/loop/run-loop.ps1 traegt weiterhin einen unstaged -StartNotBefore-
  Diff (wie in den Iterationen 6-40 vermerkt) — nicht meine Datei, nicht angefasst, nicht
  committet. Laufkontext-Block (Iterationsnummer/Zeitstempel) war auch in diesem Prompt nicht
  sichtbar mitgeliefert — Nummer aus der letzten Journal-Ueberschrift (Iteration 40)
  fortgezaehlt, Zeitstempel per date auf dem Loop-Rechner ermittelt (2026-08-11 01:17).

## Iteration 42 — c-cov-vault-gdpr-repo — done — 2026-08-11 01:25
- commit: <wird nach dem Commit ergaenzt>
- gebaut: Zwei geaenderte/neue Testdateien. `internal/security/vault/service_test.go`: fuenf
  neue Tests fuer `Service.DeleteByKeyName` (Erfolgsfall, No-Op wenn der Key nicht existiert,
  leerer Key-Name, Get-Repo-Fehler, Delete-Repo-Fehler) plus zwei GetTenantID-Fehlerpfad-Tests
  (`GetSecret`/`SetSecret` gegen `context.Background()` ohne Tenant-ID, beide erwarten
  `middleware.ErrMissingTenantID`, kein Panic/stiller Fallback). `internal/security/gdpr/
  erasure_log_test.go` (neu, DB-Integrationstest): `PostgresRepository.CreateErasureLog`
  ueber drei Faelle — Tenant-Ableitung aus `original_user_id` schlaegt fuer einen fremden
  Tenant-Caller per RLS WITH-CHECK fehl (analog zum bestehenden Muster in
  tenant_write_test.go), eigener Tenant-Caller landet real (per direkter SELECT-Abfrage auf
  `anonymized_label`/`confirmation_hash`/`modules_affected` nachgewiesen, inkl. JSONB-
  Rundlauf des `map[string]string`-Felds ohne expliziten json.Marshal-Aufruf im Repo-Code —
  pgx v5 kodiert das automatisch), unbekannter `original_user_id` schlaegt fehl (NOT-NULL auf
  dem subselect-abgeleiteten `tenant_id`, `executed_by` bewusst auf einen echten User gesetzt
  damit der Fehler eindeutig dem richtigen Pfad zuzuordnen ist). `PostgresRepository.
  GetNextAnonymizedLabel` per zwei aufeinanderfolgenden Aufrufen in frischem Tenant: erster
  liefert exakt "Geloeschter Benutzer #1", zweiter nach einem zwischenzeitlichen
  CreateErasureLog-Insert exakt "#2" — sowohl aufsteigend als auch tenant-eindeutig belegt
  (COUNT(*) ist RLS-gescoped, kein expliziter WHERE-tenant_id-Filter im Code noetig).
- gate: build ok (go build -p 2 ./internal/security/...) | vet ok | lint ok (golangci-lint
  run --config .golangci.yml ./internal/security/... -- 0 issues) | test ok (go test -count=1
  ./internal/security/... komplett gruen, ./internal/security/vault/ und ./internal/security/
  gdpr/ -v gezaehlt: 0 SKIP, alle DB-Integrationstests liefen real gegen die lokale Postgres
  als kmuhub_app) | migration n.a. (keine neue Tabelle/Spalte) | rls-smoke ueber die
  Repo-Tests selbst erbracht (Cross-Tenant-INSERT/-SELECT-Faelle oben) | keine neue Route,
  kein neuer RequirePermission-Guard, keine neue config.RequireX-Assertion
- coverage: internal/security/vault 70,6 % -> 80,0 % | internal/security/gdpr 59,8 % -> 60,6 %
  (beide lokal per go test -coverprofile + go tool cover -func, Ausgangswert durch
  temporaeres Herausnehmen der neuen/geaenderten Testdateien gemessen, danach wiederhergestellt
  und alle drei Gates erneut gruen). Das Feld coverage_start: der Unit nennt "internal/security
  47,9 %" — das ist der Aggregat-Wert ueber alle sieben security-Unterpakete aus dem
  CI-Artefakt, nicht der der beiden hier geaenderten Pakete.
- mutations-probe: drei Proben, alle gefangen. (1) In vault.Service.DeleteByKeyName die
  No-Op-Kurzschluss-Zeile `if errors.Is(err, ErrSecretNotFound) { return nil }` zu `return err`
  geaendert -> TestDeleteByKeyName_NoOpWhenMissing rot mit "vault: secret not found" statt
  nil. (2) In gdpr.PostgresRepository.GetNextAnonymizedLabel `count+1` zu `count` geaendert ->
  TestGetNextAnonymizedLabel_IncrementsPerCall rot mit "Geloeschter Benutzer #0" statt "#1".
  (3) In gdpr.PostgresRepository.CreateErasureLog den `(SELECT tenant_id FROM users WHERE id =
  $2)`-Subselect durch eine hartkodierte fremde Tenant-UUID ersetzt ->
  TestCreateErasureLog_TenantDerivedFromUser rot mit "new row violates row-level security
  policy for table gdpr_erasure_log" im eigentlich erfolgreichen Own-Ctx-Pfad. Alle drei per
  git checkout -- <datei> zurueckgedreht, git status backend/ zeigt danach nur noch die
  geaenderte/neue Testdatei, build/vet/lint/test erneut komplett gruen.
- verify vorgaenger: sauber. Commit 64b0f27b (Iteration 41) fuegt ausschliesslich drei
  Testdateien plus Journal/Backlog-Metadaten hinzu — keine Produktionscode-Datei, kein Proto,
  keine Route, kein RequirePermission-Guard, keine neue Tabelle, keine Migration. Keine der
  acht Fehlerklassen einschlaegig.
- offen: keine neuen Befunde. `.planning/backend-block/loop/run-loop.ps1` traegt weiterhin
  denselben unstaged -StartNotBefore-Diff wie in den Iterationen 6-41 vermerkt — nicht meine
  Datei, nicht angefasst, nicht committet. Laufkontext-Block (Iterationsnummer/Zeitstempel)
  war auch in diesem Prompt nicht sichtbar mitgeliefert — Nummer aus der letzten
  Journal-Ueberschrift (Iteration 41) fortgezaehlt, Zeitstempel per date auf dem Loop-Rechner
  ermittelt (2026-08-11 01:25).

## Iteration 43 — d-cov-gateway-hr-leave — done — 2026-08-11 01:33
- commit: cce98f88
- gebaut: Neue Testdatei `backend/internal/gateway/route_hr_leave_test.go` (33 Tests) fuer
  die acht in der Unit genannten Leave-Handler in `route_hr.go`: HandleCreateLeaveRequest
  (ServiceUnavailable, MissingTenant, InvalidJSON, InvalidLeaveTypeID/UUID-Tag,
  InvalidHalfDayPeriod/oneof-Tag, ValidRequestReachesRPC), HandleListLeaveRequests
  (ServiceUnavailable, MissingTenant, sechs Query-Filter-Kombinationen ueber Subtests),
  HandleGetLeaveRequest (ServiceUnavailable, ReachesRPC), HandleApproveLeaveRequest/
  HandleRejectLeaveRequest/HandleCancelLeaveRequest (je ServiceUnavailable, ReachesRPC,
  Approve/Reject zusaetzlich InvalidJSON, alle drei zusaetzlich MissingIDReachesRPC),
  HandleGetLeaveBalance und HandleListLeaveTypes (je ServiceUnavailable, MissingTenant,
  ReachesRPC). Dazu zwei direkte Unit-Tests fuer `hrMarshalSlice` (Wire-Shape: leeres Ergebnis
  ist `[]json.RawMessage{}`, nicht `nil` — protojson wuerde sonst `null` statt `[]` emittieren;
  plus Rundlauf ueber zwei Elemente).
- gate: build ok (go build -p 2 ./internal/gateway/... ./cmd/gateway/...) | vet ok | lint ok
  (golangci-lint run --config .golangci.yml ./internal/gateway/... -- 0 issues) | test ok
  (go test -count=1 ./internal/gateway/ komplett gruen, -v gezaehlt: 0 SKIP, 1332 PASS) |
  migration n.a. (keine neue Tabelle/Route) | rls-smoke n.a. (keine Tabelle/Policy
  angefasst) | TestOpenAPIRouteDrift lief separat gruen (834 Routen gegen 836 Spec-Pfade,
  unveraendert — keine neue Route in dieser Unit) | keine neue Route, kein neuer
  RequirePermission-Guard, keine neue config.RequireX-Assertion
- coverage: internal/gateway 34,9 % -> 36,3 % (go test -coverprofile + go tool cover -func,
  einziger Lauf noetig — Ausgangswert deckt sich mit coverage_start der Unit)
- mutations-probe: drei Proben, alle gefangen. (1) In `hrMarshalSlice` `parts :=
  make([]json.RawMessage, 0, len(msgs))` zu `var parts []json.RawMessage` geaendert ->
  TestHrMarshalSlice_EmptyResultIsNotNil rot ("result is nil"). (2) Im `validate`-Tag von
  `createLeaveRequestHTTPReq.LeaveTypeID` `omitempty,uuid` entfernt ->
  TestHandleCreateLeaveRequest_InvalidLeaveTypeID rot (503 statt 400, kein
  validation_failed mehr). (3) In HandleCreateLeaveRequest den `getTenantID`-Fehlerpfad
  stillgelegt (`tenantID, _ := getTenantID(r)`, kein 401-Return mehr) ->
  TestHandleCreateLeaveRequest_MissingTenant rot (503 statt 401). Alle drei per Edit
  zurueckgedreht, `git diff --stat backend/internal/gateway/route_hr.go` danach leer,
  build/vet/lint/test erneut komplett gruen.
- verify vorgaenger: sauber. Commit 2f7053f6 (Iteration 42) fuegt ausschliesslich zwei
  Testdateien plus Journal/Backlog-Metadaten hinzu — keine Produktionscode-Datei, kein
  Proto, keine Route, kein RequirePermission-Guard, keine neue Tabelle, keine Migration.
  Keine der acht Fehlerklassen einschlaegig.
- offen: Fund fuer eine spaetere Fix-/Hardening-Unit, kein Blocker: HandleApproveLeaveRequest,
  HandleRejectLeaveRequest und HandleCancelLeaveRequest validieren die Leave-Request-ID aus
  chi.URLParam(r, "id") lokal nicht als UUID (kein validateUUIDParam-Aufruf, anders als in
  route_automation.go oder route_crm.go) — eine leere oder offensichtlich falsche ID reicht
  unveraendert bis zur gRPC-Schicht durch und erzeugt dort erst einen Fehler. Betrifft
  ausschliesslich diese drei Handler in route_hr.go, nicht die uebrigen HR-Routen.
  `createLeaveRequestHTTPReq` hat zudem keine `required`-Tags auf StartDate/EndDate — das im
  urspruenglichen done_when erwartete "fehlende Pflichtfelder"-Testszenario existiert im Code
  nicht, getestet wurden stattdessen die real vorhandenen Validierungspfade (UUID- und
  oneof-Tag). `.planning/backend-block/loop/run-loop.ps1` traegt weiterhin denselben
  unstaged -StartNotBefore-Diff wie in den Iterationen 6-42 vermerkt — nicht meine Datei,
  nicht angefasst, nicht committet. Laufkontext-Block war auch in diesem Prompt nicht
  sichtbar mitgeliefert — Nummer aus der letzten Journal-Ueberschrift (Iteration 42)
  fortgezaehlt, Zeitstempel per date auf dem Loop-Rechner ermittelt (2026-08-11 01:33).

## Iteration 44 — d-cov-gateway-hr-worktime — done — 2026-08-11 01:39
- commit: 13bc5141
- gebaut: Neue Testdatei `backend/internal/gateway/route_hr_worktime_test.go` (30 Tests) fuer
  die neun in der Unit genannten Zeiterfassungs-Handler in `route_hr.go`: HandleClockIn/
  HandleClockOut (je ServiceUnavailable, MissingTenant, ReachesRPC), HandleBreakStart/
  HandleBreakEnd/HandleGetActiveShift (je ServiceUnavailable, ReachesRPC — kein
  MissingTenant-Fall, siehe offen), HandleSubmitWeek (ServiceUnavailable, MissingTenant,
  MissingWeekStart, InvalidJSON, ValidRequestReachesRPC) sowie HandleApproveWeek/
  HandleRejectWeek/HandleReopenWeek (je ServiceUnavailable, MissingTenant, InvalidJSON,
  InvalidEmployeeID, ValidRequestReachesRPC; HandleApproveWeek zusaetzlich MissingWeekStart).
- gate: build ok (go build -p 2 ./internal/gateway/... ./cmd/gateway/...) | vet ok | lint ok
  (golangci-lint run --config .golangci.yml ./internal/gateway/... -- 0 issues) | test ok
  (go test -count=1 ./internal/gateway/ -v komplett gruen, 1365 PASS, 0 SKIP, 0 FAIL) |
  migration n.a. (keine neue Tabelle/Route) | rls-smoke n.a. (keine Tabelle/Policy
  angefasst) | keine neue Route, kein neuer RequirePermission-Guard, keine neue
  config.RequireX-Assertion — TestOpenAPIRouteDrift lief als Teil des vollen Pakettests mit,
  unveraendert gruen
- coverage: internal/gateway 34,9 % -> 36,8 % (go test -coverprofile + go tool cover -func;
  der unmittelbare Vorwert nach Iteration 43 lag bei 36,3 % — der coverage_start-Bezugswert
  der Unit bleibt ueber den ganzen Lauf 34,9 %, siehe BACKLOG.yml-Kopf)
- mutations-probe: drei Proben, alle gefangen. (1) In `approveWeekHTTPReq.EmployeeID` das
  `uuid`-Validate-Tag entfernt (nur noch `required`) -> TestHandleApproveWeek_InvalidEmployeeID
  rot mit 503 statt 400, kein validation_failed mehr. (2) In HandleClockIn den
  `getTenantID`-Fehlerpfad stillgelegt (`tenantID, _ := getTenantID(r)`, kein 401-Return
  mehr) -> TestHandleClockIn_MissingTenant rot mit 503 statt 401. (3) In
  `submitWeekHTTPReq.WeekStart` das `required`-Validate-Tag entfernt ->
  TestHandleSubmitWeek_MissingWeekStart rot mit 503 statt 400, kein validation_failed mehr.
  Alle drei per Edit zurueckgedreht, `git diff --stat backend/internal/gateway/route_hr.go`
  danach leer, build/vet/lint/test erneut komplett gruen (1365 PASS, 0 SKIP, 0 FAIL).
- verify vorgaenger: sauber. Commit aa2a8663 (Iteration 43 Journal-Hash-Nachtrag) sowie der
  vorangehende cce98f88 (Iteration 43 selbst) fuegen ausschliesslich eine Testdatei plus
  Journal-/Backlog-Metadaten hinzu — keine Produktionscode-Datei, kein Proto, keine Route,
  kein RequirePermission-Guard, keine neue Tabelle, keine Migration. Keine der acht
  Fehlerklassen einschlaegig.
- offen: Fund fuer eine spaetere Fix-/Hardening-Unit, kein Blocker: HandleBreakStart,
  HandleBreakEnd und HandleGetActiveShift (route_hr.go:631-720) rufen `getTenantID` an
  keiner Stelle auf — nur ClockIn/ClockOut und der Week-Workflow tun das. Ein Request ohne
  Tenant-Kontext erreicht bei diesen drei Handlern unveraendert die RPC-Schicht statt eines
  401. Heute latent, weil `authMiddleware` auf der ganzen `/api/v1/hr/time`-Gruppe sitzt
  (route_hr.go:101-102) und Tenant/User immer gemeinsam gesetzt werden — anders als beim in
  Iteration 40 gefixten `own`-Scope-Bug ist hier kein bekannter Weg bekannt, wie ein
  Aufrufer legitim ohne Tenant durchkommt, deshalb keine eigene Fix-Unit vorgeschlagen,
  nur hier vermerkt. Die Nil-Entry/Nil-ActiveBreak-Wire-Shape-Verzweigung in
  HandleGetActiveShift (route_hr.go:688-717) ist wie schon bei `hrMarshalSlice` in
  Iteration 43 nur ueber einen echten RPC-Response beobachtbar — kein bufconn-Stub fuer den
  HR-Service in diesem Paket, deshalb nur ReachesRPC statt Wire-Shape-Assertion getestet.
  `.planning/backend-block/loop/run-loop.ps1` traegt weiterhin denselben unstaged
  -StartNotBefore-Diff wie in den Iterationen 6-43 vermerkt — nicht meine Datei, nicht
  angefasst, nicht committet. Laufkontext-Block war auch in diesem Prompt nicht sichtbar
  mitgeliefert — Nummer aus der letzten Journal-Ueberschrift (Iteration 43) fortgezaehlt,
  Zeitstempel per date auf dem Loop-Rechner ermittelt (2026-08-11 01:39).

## Iteration 45 — d-cov-gateway-fuhrpark-vehicles-services — done — 2026-08-11 01:42
- commit: f4441cfa
- gebaut: Neue Testdatei `backend/internal/gateway/route_fuhrpark_crud_test.go` (37 Tests)
  fuer die neun in der Unit genannten Fahrzeug- und Service-Handler in route_fuhrpark.go:
  HandleListVehicles (ServiceUnavailable, MissingTenant, ReachesRPC mit Query-Filtern),
  HandleGetVehicle/HandleUpdateVehicle/HandleDeleteVehicle (je ServiceUnavailable,
  MissingTenant, InvalidIDUUID, ReachesRPC; UpdateVehicle zusaetzlich InvalidFuelType als
  echter Validierungsfehlerpfad), HandleGetVehicleHistory (ServiceUnavailable,
  MissingTenant, InvalidIDUUID, ReachesRPC mit Pagination), HandleListVehicleServices
  (MissingTenant, InvalidIDUUID, ServiceUnavailable, ReachesRPC), HandleUpdateService
  (ServiceUnavailable, MissingTenant, InvalidServiceIDUUID, InvalidJSON,
  ReachesRPCWithArbitraryStatus), HandleCompleteService (ServiceUnavailable,
  MissingTenant, InvalidServiceIDUUID, InvalidJSON, ReachesRPC) und HandleCheckTuevDue
  (ServiceUnavailable, MissingTenant, ReachesRPC).
- gate: build ok (go build -p 2 ./internal/gateway/... ./cmd/gateway/...) | vet ok | lint ok
  (golangci-lint run --config .golangci.yml ./internal/gateway/... -- 0 issues) | test ok
  (go test -count=1 ./internal/gateway/ -v komplett gruen, 1402 PASS, 0 SKIP, 0 FAIL) |
  migration n.a. (keine neue Tabelle/Route) | rls-smoke n.a. (keine Tabelle/Policy
  angefasst) | TestOpenAPIRouteDrift lief als Teil des vollen Pakettests mit, unveraendert
  gruen (834 Routen gegen 836 Spec-Pfade) | keine neue Route, kein neuer
  RequirePermission-Guard, keine neue config.RequireX-Assertion
- coverage: internal/gateway 34,9 % -> 37,3 % (go test -coverprofile + go tool cover -func,
  einziger Lauf noetig — Ausgangswert deckt sich mit coverage_start der Unit)
- mutations-probe: drei Proben, alle gefangen. (1) In `updateVehicleRequest.FuelType` das
  `validate:"omitempty,oneof=..."`-Tag entfernt -> TestHandleUpdateVehicle_InvalidFuelType
  rot (503 statt 400, kein validation_failed mehr). (2) In HandleGetVehicle
  `validateUUIDParam(w, r, "id")` durch das ungeprüfte `chi.URLParam(r, "id")` ersetzt ->
  TestHandleGetVehicle_InvalidIDUUID rot (503 statt 400, "connection error" statt
  "invalid id"). (3) In HandleUpdateService den `getTenantID`-Fehlerpfad stillgelegt
  (`tenantID, _ := middleware.GetTenantID(r.Context())`, kein 401-Return mehr) ->
  TestHandleUpdateService_MissingTenant rot (503 statt 401). Alle drei per Python-Skript
  gesetzt und zurueckgedreht (Backup-Datei vor jeder Probe), `git diff --stat
  backend/internal/gateway/route_fuhrpark.go` danach leer, build/vet/lint/test erneut
  komplett gruen (1402 PASS, 0 SKIP, 0 FAIL).
- verify vorgaenger: sauber. Commit 13bc5141 (Iteration 44) sowie der Metadaten-Commit
  23826442 fuegen ausschliesslich eine Testdatei plus Journal-/Backlog-Metadaten hinzu —
  keine Produktionscode-Datei, kein Proto, keine Route, kein RequirePermission-Guard, keine
  neue Tabelle, keine Migration. Keine der acht Fehlerklassen einschlaegig.
- offen: Fund fuer eine spaetere Fix-/Hardening-Unit, kein Blocker:
  `updateServiceRequest.Status` und `completeServiceRequest` (route_fuhrpark.go:220-235)
  tragen keine `validate`-Tags — anders als `createVehicleRequest`/`updateVehicleRequest`
  wird ein beliebiger Status-String am Gateway nicht abgelehnt; ein ungueltiger
  Statusuebergang kann nur die fuhrpark-Service-Schicht selbst zurueckweisen. Ohne
  bufconn-Stub fuer den fuhrpark-Service in diesem Paket war das nicht als lokaler
  400-Fehlerpfad zu testen (dieselbe Grenze wie in jeder bisherigen
  Gateway-Coverage-Unit dieses Laufs), deshalb stattdessen
  TestHandleUpdateService_ReachesRPCWithArbitraryStatus: belegt, dass der Request bis zur
  RPC-Schicht durchlaeuft statt lokal zu scheitern. Aus demselben Grund testet
  HandleCheckTuevDue nur ReachesRPC statt der im Backlog gewuenschten
  Wire-Shape-Assertion — der Handler reicht die RPC-Antwort unveraendert an
  `response.Proto` durch (route_fuhrpark.go:968-975), es gibt keine gateway-eigene
  Marshaling-Logik wie `hrMarshalSlice` und keinen Fake-Server, an dem eine leere
  Faelligkeitsliste beobachtbar waere. `.planning/backend-block/loop/run-loop.ps1` traegt
  weiterhin denselben unstaged -StartNotBefore-Diff wie in den Iterationen 6-44 vermerkt —
  nicht meine Datei, nicht angefasst, nicht committet. Laufkontext-Block war auch in diesem
  Prompt nicht sichtbar mitgeliefert — Nummer aus der letzten Journal-Ueberschrift
  (Iteration 44) fortgezaehlt, Zeitstempel per date auf dem Loop-Rechner ermittelt
  (2026-08-11 01:42).

## Iteration 46 — d-cov-gateway-fuhrpark-fuel-trip-bookings — done — 2026-08-11 01:53
- commit: d72649e2
- gebaut: Neue Testdatei `backend/internal/gateway/route_fuhrpark_logs_test.go` (36 Tests)
  fuer die neun in der Unit genannten Handler in route_fuhrpark.go: HandleCreateFuelLog
  (ServiceUnavailable, MissingTenant, MissingLiters, MissingDate, InvalidVehicleIDUUID,
  DefaultFuelTypeReachesRPC), HandleUpdateFuelLog (ServiceUnavailable, MissingTenant,
  InvalidIDUUID, InvalidLiters, ReachesRPC), HandleDeleteFuelLog (ServiceUnavailable,
  MissingTenant, InvalidIDUUID, ReachesRPC), HandleCreateTripLog (ServiceUnavailable,
  MissingTenant, MissingDate, MissingDriverName, InvalidVehicleIDUUID, ReachesRPC),
  HandleUpdateTripLog (ServiceUnavailable, MissingTenant, InvalidIDUUID, InvalidStartKm,
  ReachesRPC), HandleDeleteTripLog (ServiceUnavailable, MissingTenant, InvalidIDUUID,
  ReachesRPC), HandleCreateVehicleBooking (ServiceUnavailable, MissingTenant,
  MissingEndsAt, InvalidVehicleID, ReversedPeriodReachesRPC), HandleUpdateVehicleBooking
  (ServiceUnavailable, MissingTenant, InvalidIDUUID, InvalidStatus,
  ReversedPeriodReachesRPC) und HandleDeleteVehicleBooking (ServiceUnavailable,
  MissingTenant, InvalidIDUUID, ReachesRPC).
- gate: build ok (go build -p 2 ./internal/gateway/... ./cmd/gateway/...) | vet ok | lint ok
  (golangci-lint run --config .golangci.yml ./internal/gateway/... -- 0 issues) | test ok
  (go test -count=1 ./internal/gateway/ -v komplett gruen, 1446 PASS, 0 SKIP, 0 FAIL) |
  migration n.a. (keine neue Tabelle/Route) | rls-smoke n.a. (keine Tabelle/Policy
  angefasst) | TestOpenAPIRouteDrift lief als Teil des vollen Pakettests mit, unveraendert
  gruen (834 Routen gegen 836 Spec-Pfade) | keine neue Route, kein neuer
  RequirePermission-Guard, keine neue config.RequireX-Assertion
- coverage: internal/gateway 34,9 % -> 38,0 % (go test -coverprofile + go tool cover -func,
  einziger Lauf noetig)
- mutations-probe: drei Proben, alle gefangen. (1) In HandleDeleteFuelLog
  `validateUUIDParam(w, r, "id")` durch das ungeprüfte `chi.URLParam(r, "id")` ersetzt ->
  TestHandleDeleteFuelLog_InvalidIDUUID rot (503 statt 400, "connection error" statt
  "invalid id"). (2) In `updateTripLogRequest.StartKm` das `validate:"omitempty,gte=0"`-Tag
  entfernt -> TestHandleUpdateTripLog_InvalidStartKm rot (503 statt 400, kein
  validation_failed mehr, Feld "start_km" fehlt in den Details). (3) In
  HandleCreateVehicleBooking den `GetTenantID`-Fehlerpfad stillgelegt (`_, _ :=
  middleware.GetTenantID(r.Context())`, kein 401-Return mehr) ->
  TestHandleCreateVehicleBooking_MissingTenant rot (503 statt 401). Alle drei per Edit-Tool
  gesetzt und zurueckgedreht, `git diff --stat backend/internal/gateway/route_fuhrpark.go`
  danach leer, build/vet/lint/test erneut komplett gruen (1446 PASS, 0 SKIP, 0 FAIL).
- verify vorgaenger: sauber. Commit f4441cfa (Iteration 45) sowie der Metadaten-Commit
  74b73729 fuegen ausschliesslich eine Testdatei plus Journal-/Backlog-Metadaten hinzu —
  keine Produktionscode-Datei, kein Proto, keine Route, kein RequirePermission-Guard, keine
  neue Tabelle, keine Migration. Keine der acht Fehlerklassen einschlaegig.
- offen: Fund fuer eine spaetere Fix-/Hardening-Unit, kein Blocker: weder
  `createVehicleBookingRequest` noch `updateVehicleBookingRequest`
  (route_fuhrpark.go:1736-1749) validieren, dass `starts_at` vor `ends_at` liegt — beide
  Felder tragen nur `validate:"required"` bzw. gar kein Tag, ein umgekehrter Buchungszeitraum
  wird am Gateway nicht abgelehnt und erreicht ungeprueft die RPC-Schicht (siehe
  TestHandleCreateVehicleBooking_ReversedPeriodReachesRPC /
  TestHandleUpdateVehicleBooking_ReversedPeriodReachesRPC). Ob der fuhrpark-Service das
  serverseitig ablehnt, war ohne bufconn-Stub fuer den Service in diesem Paket nicht lokal
  zu pruefen — dieselbe Grenze wie in jeder bisherigen Gateway-Coverage-Unit dieses Laufs.
  `.planning/backend-block/loop/run-loop.ps1` traegt weiterhin denselben unstaged
  -StartNotBefore-Diff wie in den Iterationen 6-45 vermerkt — nicht meine Datei, nicht
  angefasst, nicht committet. Laufkontext-Block war auch in diesem Prompt nicht sichtbar
  mitgeliefert — Nummer aus der letzten Journal-Ueberschrift (Iteration 45) fortgezaehlt,
  Zeitstempel per date auf dem Loop-Rechner ermittelt (2026-08-11 01:53).

## Iteration 47 — d-cov-gateway-video-recording-consent — done — 2026-08-11 02:02
- commit: 7e2838a3
- gebaut: Neue Testdatei `backend/internal/gateway/route_video_recording_test.go` (27 Tests) fuer
  die acht in der Unit genannten Handler in route_video.go: HandleStartRecording
  (ServiceUnavailable, InvalidJSON, InvalidCallID, InvalidMeetingID, ReachesRPC),
  HandleStopRecording (ServiceUnavailable, InvalidIDUUID, ReachesRPC), HandleSetRecordingConsent
  (ServiceUnavailable, InvalidIDUUID, InvalidConsentedType, ReachesRPC), HandleGetRecordingConsent
  (ServiceUnavailable, InvalidIDUUID, ReachesRPC), HandleListRecordings (ServiceUnavailable,
  ReachesRPC) plus TestProtoListRecordings_WireShape, HandleDeleteRecording (ServiceUnavailable,
  InvalidIDUUID, ReachesRPC), HandleGetRecordingConsents (ServiceUnavailable, InvalidIDUUID,
  ReachesRPC) und HandleTagRecordingWithConsents (ServiceUnavailable, InvalidIDUUID,
  EmptySnapshot, ReachesRPC).
- gate: build ok (go build -p 2 ./internal/gateway/... ./cmd/gateway/...) | vet ok | lint ok
  (golangci-lint run --config .golangci.yml ./internal/gateway/... -- 0 issues) | test ok
  (go test -count=1 ./internal/gateway/ -v komplett gruen, 1474 PASS, 0 SKIP, 0 FAIL) | migration
  n.a. (keine neue Tabelle/Route) | rls-smoke n.a. (keine Tabelle/Policy angefasst) |
  TestOpenAPIRouteDrift separat gelaufen, unveraendert gruen (834 Routen gegen 836 Spec-Pfade) |
  keine neue Route, kein neuer RequirePermission-Guard, keine neue config.RequireX-Assertion
- coverage: internal/gateway 34,9 % -> 38,4 % (go test -coverprofile + go tool cover -func)
- mutations-probe: drei Proben, alle gefangen. (1) In HandleDeleteRecording
  `validateUUIDParam(w, r, "id")` durch das ungeprüfte `chi.URLParam(r, "id")` ersetzt ->
  TestHandleDeleteRecording_InvalidIDUUID rot (503 statt 400, "connection error" statt
  "invalid id"). (2) In `startRecordingRequest.CallID` das `validate:"omitempty,uuid"`-Tag
  entfernt -> TestHandleStartRecording_InvalidCallID rot (503 statt 400, kein validation_failed
  mehr, Feld "call_id" fehlt in den Details). (3) In `tagRecordingConsentsRequest.Snapshot` das
  `min=1` aus dem validate-Tag entfernt -> TestHandleTagRecordingWithConsents_EmptySnapshot rot
  (503 statt 400, kein validation_failed mehr, Feld "snapshot" fehlt in den Details). Alle drei
  per Edit-Tool gesetzt und zurueckgedreht, `git diff --stat backend/internal/gateway/route_video.go`
  danach leer, build/vet/lint/test erneut komplett gruen (1474 PASS, 0 SKIP, 0 FAIL).
- verify vorgaenger: sauber. Commit d72649e2 (Iteration 46) sowie der Metadaten-Commit 1888180a
  fuegen ausschliesslich eine Testdatei plus Journal-/Backlog-Metadaten hinzu — keine
  Produktionscode-Datei, kein Proto, keine Route, kein RequirePermission-Guard, keine neue
  Tabelle, keine Migration. Keine der acht Fehlerklassen einschlaegig.
- offen: zwei Abweichungen vom im Backlog skizzierten Testplan, beide dokumentiert statt
  stillschweigend anders gebaut: (1) `startRecordingRequest` traegt gar kein Consent-Feld
  (Consent wird erst nachtraeglich per HandleSetRecordingConsent gesetzt, siehe Kommentar
  "Push recording.started WS event" in HandleStartRecording) — statt der im Backlog verlangten
  "Consent-Status"-Fehlerpruefung testet TestHandleStartRecording_Invalid{CallID,MeetingID} die
  tatsaechlich vorhandene UUID-Validierung. (2) `setRecordingConsentRequest.Consented` ist ein
  nacktes `bool` ohne validate-Tag — es gibt keinen "fehlend/ungueltig"-Fall im Sinne einer
  Validierungsregel, nur einen JSON-Typfehler (`"consented":"yes"`), den
  TestHandleSetRecordingConsent_InvalidConsentedType als naechstliegenden Fehlerpfad abdeckt.
  Drittens ein Befund zum protojson-Marshaler: `response.Proto` laeuft mit
  `EmitUnpopulated=false` (internal/server/response/response.go), ein leeres `recordings`-Array
  wird deshalb komplett aus dem Body weggelassen statt als `[]` serialisiert — dasselbe tolerante
  Verhalten, das TestProtoMeetingOccurrences_WireShape fuer `items` bereits akzeptiert. Kein Bug,
  aber der FE-Konsument darf sich nicht auf das Vorhandensein des Schluessels verlassen; kein
  Blocker fuer diese Unit, da bereits an anderer Stelle im Repo so gehandhabt.
  `.planning/backend-block/loop/run-loop.ps1` traegt weiterhin denselben unstaged
  -StartNotBefore-Diff wie in den Iterationen 6-46 vermerkt — nicht meine Datei, nicht
  angefasst, nicht committet. Laufkontext-Block war auch in diesem Prompt nicht sichtbar
  mitgeliefert — Nummer aus der letzten Journal-Ueberschrift (Iteration 46) fortgezaehlt,
  Zeitstempel per date auf dem Loop-Rechner ermittelt (2026-08-11 02:02).

## Iteration 48 — d-cov-gateway-video-meeting-lifecycle — done — 2026-08-11 02:09
- commit: 725e99de
- gebaut: Neue Testdatei `backend/internal/gateway/route_video_meeting_lifecycle_test.go`
  (37 Tests) fuer die zehn in der Unit genannten Meeting-Lebenszyklus-Handler in
  route_video.go: HandleGetMeeting (ServiceUnavailable, InvalidIDUUID, ReachesRPC),
  HandleUpdateMeeting (ServiceUnavailable, InvalidIDUUID, InvalidJSON,
  InvalidScheduledStartFormat, InvalidScheduledEndFormat, ReachesRPC), HandleDeleteMeeting
  (ServiceUnavailable, InvalidIDUUID, ReachesRPC), HandleListMeetings (ServiceUnavailable,
  ReachesRPC, UnknownStatusIgnored) plus TestProtoListMeetings_WireShape, HandleStartMeeting
  (ServiceUnavailable, InvalidIDUUID, ReachesRPC), HandleJoinMeeting (ServiceUnavailable,
  InvalidIDUUID, ReachesRPC), HandleEndMeeting (ServiceUnavailable, InvalidIDUUID,
  ReachesRPC), HandleSetMeetingLock (ServiceUnavailable, InvalidIDUUID, InvalidLockedType,
  ReachesRPC), HandleMuteAllMeetingParticipants (ServiceUnavailable, InvalidIDUUID,
  ReachesRPC) und HandleRemoveMeetingParticipant (ServiceUnavailable, InvalidIDUUID,
  MissingTargetUserID, InvalidTargetUserID, ReachesRPC).
- gate: build ok (go build -p 2 ./internal/gateway/... ./cmd/gateway/...) | vet ok | lint ok
  (golangci-lint run --config .golangci.yml ./internal/gateway/... -- 0 issues) | test ok
  (go test -count=1 ./internal/gateway/ -v komplett gruen, 1511 PASS, 0 SKIP, 0 FAIL) | migration
  n.a. (keine neue Tabelle/Route) | rls-smoke n.a. (keine Tabelle/Policy angefasst) |
  TestOpenAPIRouteDrift separat gelaufen, unveraendert gruen (834 Routen gegen 836 Spec-Pfade) |
  keine neue Route, kein neuer RequirePermission-Guard, keine neue config.RequireX-Assertion
- coverage: internal/gateway 34,9 % -> 39,1 % (go test -coverprofile + go tool cover -func)
- mutations-probe: drei Proben, alle gefangen. (1) In HandleGetMeeting
  `validateUUIDParam(w, r, "id")` durch das ungeprüfte `chi.URLParam(r, "id")` ersetzt ->
  TestHandleGetMeeting_InvalidIDUUID rot (503 statt 400, "connection error" statt "invalid
  id"). (2) In `removeMeetingParticipantHTTPRequest.TargetUserID` das
  `validate:"required,uuid"`-Tag entfernt -> TestHandleRemoveMeetingParticipant_
  {MissingTargetUserID,InvalidTargetUserID} beide rot (503 statt 400, kein
  validation_failed mehr, Feld "target_user_id" fehlt in den Details). (3) In
  HandleUpdateMeeting die scheduled_start-Formatpruefung mit `if false {}` stillgelegt ->
  TestHandleUpdateMeeting_InvalidScheduledStartFormat rot (Fehlermeldung ist der
  RPC-Verbindungsfehler statt "invalid scheduled_start format"). Alle drei per Edit-Tool
  gesetzt und zurueckgedreht, `git diff --stat backend/internal/gateway/route_video.go`
  danach leer, build/vet/lint/test erneut komplett gruen (1511 PASS, 0 SKIP, 0 FAIL).
- verify vorgaenger: sauber. Commit 7e2838a3 (Iteration 47) sowie der Metadaten-Commit
  84c2d077 fuegen ausschliesslich eine Testdatei plus Journal-/Backlog-Metadaten hinzu —
  keine Produktionscode-Datei, kein Proto, keine Route, kein RequirePermission-Guard, keine
  neue Tabelle, keine Migration. Keine der acht Fehlerklassen einschlaegig.
- offen: Wie bei den Recording-Handlern in Iteration 47 gibt es fuer VideoServiceClient kein
  bufconn-Stub in diesem Paket — die eigentliche Meeting-Status-Zustandsmaschine (z. B.
  "kein zweiter Start auf einem laufenden Meeting", "Join nach End abgelehnt") wird
  ausschliesslich vom Video-Service durchgesetzt und war lokal nicht simulierbar. Die
  ReachesRPC-Tests dokumentieren stattdessen, dass jeder Handler die lokale Validierung
  passiert und die RPC-Schicht erreicht (503 ueber die unerreichbare Dummy-Adresse) — der
  naechstliegende Fehlerpfad ohne Produktionscode-Aenderung. Kein neuer Befund: dieselbe
  Grenze wie in jeder bisherigen Gateway-Coverage-Unit fuer route_video.go dieses Laufs.
  `.planning/backend-block/loop/run-loop.ps1` traegt weiterhin denselben unstaged
  -StartNotBefore-Diff wie in den Iterationen 6-47 vermerkt — nicht meine Datei, nicht
  angefasst, nicht committet. Laufkontext-Block war auch in diesem Prompt nicht sichtbar
  mitgeliefert — Nummer aus der letzten Journal-Ueberschrift (Iteration 47) fortgezaehlt,
  Zeitstempel per date auf dem Loop-Rechner ermittelt (2026-08-11 02:09).

## Iteration 49 — d-cov-gateway-email-accounts — done — 2026-08-11 02:11
- commit: b46fc88b
- gebaut: Neue Testdatei `backend/internal/gateway/route_email_accounts_test.go`
  (23 Tests) fuer die sieben E-Mail-Konto-Handler in route_email.go: HandleCreateAccount
  (ServiceUnavailable, InvalidJSON, ReachesRPC), HandleGetAccount (ServiceUnavailable,
  ReachesRPC), HandleListAccounts (ServiceUnavailable, ReachesRPC) plus
  TestListEmailAccountsResponse_EmptyWireShape, HandleUpdateAccount (ServiceUnavailable,
  InvalidJSON, InvalidIDUUID, ReachesRPC), HandleDeleteAccount (ServiceUnavailable,
  ReachesRPC), HandleSetDefaultAccount (ServiceUnavailable, ReachesRPC) und
  HandleTestConnection (ServiceUnavailable, InvalidJSON, ReachesRPC).
- gate: build ok (go build -p 2 ./internal/gateway/... ./cmd/gateway/...) | vet ok | lint ok
  (golangci-lint run --config .golangci.yml ./internal/gateway/... -- 0 issues) | test ok
  (go test -count=1 ./internal/gateway/ -v komplett gruen, 1530 PASS, 0 SKIP, 0 FAIL) | migration
  n.a. (keine neue Tabelle/Route) | rls-smoke n.a. (keine Tabelle/Policy angefasst) |
  TestOpenAPIRouteDrift separat gelaufen, unveraendert gruen (834 Routen gegen 836 Spec-Pfade) |
  keine neue Route, kein neuer RequirePermission-Guard, keine neue config.RequireX-Assertion
- coverage: internal/gateway 34,9 % -> 39,4 % (go test -coverprofile + go tool cover -func)
- mutations-probe: drei Proben, alle gefangen. (1) In HandleCreateAccount die
  Invalid-JSON-Fehlerpruefung durch `_ = json.NewDecoder(r.Body).Decode(&req)`
  ersetzt (Fehler verschluckt) -> TestHandleCreateAccount_InvalidJSON rot (503 statt 400,
  "connection error" statt "invalid request body"). (2) In HandleGetAccount
  `http.StatusBadGateway` zu `http.StatusOK` im Client-Fehlerpfad geaendert ->
  TestHandleGetAccount_ServiceUnavailable rot (200 statt 502). (3) In HandleTestConnection
  die Fehlerpruefung mit `err != nil && false` stillgelegt -> TestHandleTestConnection_
  InvalidJSON rot (503 statt 400, "connection error" statt "invalid request body"). Alle
  drei per Edit-Tool gesetzt und zurueckgedreht, `git diff --stat
  backend/internal/gateway/route_email.go` danach leer, build/vet/lint/test erneut
  komplett gruen (1530 PASS, 0 SKIP, 0 FAIL).
- verify vorgaenger: sauber. Commit 725e99de (Iteration 48) sowie der Metadaten-Commit
  4d53180e fuegen ausschliesslich eine Testdatei plus Journal-/Backlog-Metadaten hinzu —
  keine Produktionscode-Datei, kein Proto, keine Route, kein RequirePermission-Guard, keine
  neue Tabelle, keine Migration. Keine der acht Fehlerklassen einschlaegig.
- offen: Drei Abweichungen vom im Backlog skizzierten Testplan dokumentiert statt
  stillschweigend anders gebaut. (1) Alle sieben Account-Handler sind proto-direct (kein
  lokales DTO, kein validate-Tag) und rufen weder eine Pflichtfeld- noch eine
  UUID-Pruefung lokal auf — HandleCreateAccount dekodiert nur JSON und reicht die
  Felder unvalidiert an den gRPC-Client weiter; HandleUpdateAccount/HandleDeleteAccount/
  HandleSetDefaultAccount reichen `chi.URLParam(r,"id")` unvalidiert weiter. Statt der im
  Backlog verlangten "fehlende Pflichtfelder"/"ungueltige UUID"-Fehlerfaelle testen die
  ServiceUnavailable/InvalidJSON/ReachesRPC-Tests die tatsaechlich vorhandenen Fehlerpfade;
  TestHandleUpdateAccount_InvalidIDUUID belegt explizit, dass ein Nicht-UUID-Pfadsegment
  NICHT lokal abgelehnt wird, sondern bis zur RPC-Schicht durchlaeuft (503). (2)
  HandleListAccounts liefert `resp` ueber `response.JSON` (encoding/json.Marshal auf dem
  rohen Proto-Struct), nicht ueber `response.Proto`/`response.ProtoList`. Da
  `ListEmailAccountsResponse.Accounts` das protoc-gen-go-Standard-Tag
  `json:"accounts,omitempty"` traegt, wird ein leeres/nil-Slice bei
  `encoding/json.Marshal` **weder** als `"accounts":[]` **noch** als `"accounts":null`
  serialisiert — der Schluessel fehlt komplett im Body (`{}`). Das ist eine dritte,
  im Backlog nicht vorgesehene Auspraegung neben der erwarteten "leer -> [] statt
  null". `TestListEmailAccountsResponse_EmptyWireShape` haelt das reale Verhalten fest.
  Kein Blocker: der einzige Konsument (desktop/src/renderer/src/modules/mails/
  MailsPage.tsx:142 und settings/MailsSettingsPanel.tsx:48) liest bereits defensiv
  ueber `accountsData?.accounts ?? []`. Kein neuer Fix-Unit-Vorschlag fuer Lauf 9, weil
  folgenlos — dieselbe Kategorie Befund wie das EmitUnpopulated-Verhalten in Iteration 47
  (dort protojson-Pfad, hier encoding/json-Pfad, beide harmlos wegen FE-Guard).
  `.planning/backend-block/loop/run-loop.ps1` traegt weiterhin denselben unstaged
  -StartNotBefore-Diff wie in den Iterationen 6-48 vermerkt — nicht meine Datei, nicht
  angefasst, nicht committet. Laufkontext-Block war auch in diesem Prompt nicht sichtbar
  mitgeliefert — Nummer aus der letzten Journal-Ueberschrift (Iteration 48) fortgezaehlt,
  Zeitstempel per date auf dem Loop-Rechner ermittelt (2026-08-11 02:11).

## Iteration 50 — d-cov-gateway-email-compose-actions — done — 2026-08-11 02:28
- commit: 071b5bf1
- gebaut: Neue Testdatei `backend/internal/gateway/route_email_compose_test.go`
  (48 Tests) fuer die zehn Send-/Massenaktions-Handler in route_email.go:
  HandleMarkRead/HandleMarkUnread/HandleToggleStar (ServiceUnavailable,
  ReachesRPC je), HandleMoveToFolder (ServiceUnavailable, InvalidJSON,
  MissingTargetFolderID, InvalidTargetFolderIDFormat, InvalidMessageIDUUID,
  ReachesRPC), HandleDeleteMessage (ServiceUnavailable, InvalidIDUUID,
  ReachesRPC), HandleBulkMessageAction (ServiceUnavailable, InvalidJSON,
  EmptyIDs, InvalidUUIDInIDs, MissingAction, InvalidTargetUUID,
  DeleteWithoutPermission, DeleteWithPermission_ReachesRPC, ReachesRPC),
  HandleSendEmail (ServiceUnavailable, InvalidJSON, MissingTo,
  InvalidEmailInTo, InvalidContactID, ReachesRPC), HandleSaveDraft
  (ServiceUnavailable, InvalidJSON, InvalidEmailInTo, ReachesRPC),
  HandleReplyEmail (ServiceUnavailable, InvalidJSON,
  MissingOriginalMessageID, ReachesRPC), HandleForwardEmail
  (ServiceUnavailable, InvalidJSON, MissingOriginalMessageID, MissingTo,
  InvalidEmailInTo, ReachesRPC).
- gate: build ok (go build -p 2 ./internal/gateway/... ./cmd/gateway/...) | vet ok | lint ok
  (golangci-lint run --config .golangci.yml ./internal/gateway/... -- 0 issues) | test ok
  (go test -count=1 ./internal/gateway/ -v komplett gruen, 1574 PASS, 0 SKIP, 0 FAIL) | migration
  n.a. (keine neue Tabelle/Route) | rls-smoke n.a. (keine Tabelle/Policy angefasst) |
  TestOpenAPIRouteDrift separat gelaufen, unveraendert gruen (834 Routen gegen 836 Spec-Pfade) |
  keine neue Route, kein neuer RequirePermission-Guard, keine neue config.RequireX-Assertion
- coverage: internal/gateway 34,9 % -> 39,9 % (go test -coverprofile + go tool cover -func)
- mutations-probe: drei Proben, alle gefangen. (1) In HandleBulkMessageAction das
  `!slices.Contains(...)` der zweiten Permission-Pruefung zu
  `slices.Contains(...)` invertiert (Delete-Guard umgedreht) ->
  TestHandleBulkMessageAction_DeleteWithoutPermission UND
  TestHandleBulkMessageAction_DeleteWithPermission_ReachesRPC beide rot
  (403/503 vertauscht). (2) In `moveToFolderDTO` das `validate`-Tag von
  `required,uuid` auf `omitempty,uuid` geschwaecht ->
  TestHandleMoveToFolder_MissingTargetFolderID rot (503 statt 400,
  keine validation_failed-Struktur). (3) In HandleDeleteMessage den
  Fehlerpfad `respondGRPCError(w, err)` durch `response.JSON(w,
  http.StatusOK, resp)` ersetzt (RPC-Fehler verschluckt) ->
  TestHandleDeleteMessage_InvalidIDUUID UND TestHandleDeleteMessage_ReachesRPC
  beide rot (200 statt 503). Alle drei per Edit-Tool gesetzt und
  zurueckgedreht, `git diff --stat backend/internal/gateway/route_email.go`
  danach leer, build/vet/lint/test erneut komplett gruen (1574 PASS, 0 SKIP,
  0 FAIL).
- verify vorgaenger: sauber. Commit b46fc88b (Iteration 49) sowie der Metadaten-Commit
  844ce463 fuegen ausschliesslich eine Testdatei plus Journal-/Backlog-Metadaten hinzu —
  keine Produktionscode-Datei, kein Proto, keine Route, kein RequirePermission-Guard, keine
  neue Tabelle, keine Migration. Keine der acht Fehlerklassen einschlaegig.
- offen: (1) Drei todo-Fix-Units am Dateiende von BACKLOG.yml
  (fix-email-send-missing-tenant-id, fix-email-attachment-download-metadata-wrong-message-id,
  fix-crm-erasure-double-count) sind explizit als "Fuer Lauf 9" markiert und wurden
  deshalb uebersprungen, obwohl sie formal `status: todo` mit leeren `deps` tragen —
  kein Backlog-Fehler, nur zur Klarheit dokumentiert, falls eine kuenftige Iteration
  denselben Datei-Scan macht. (2) Die im Backlog erwaehnte "Consent bei fehlendem
  contact_id" Fehlerpfad-Erwartung ist bei genauerem Hinsehen keine Gateway-Sache:
  die Consent-Durchsetzung fuer SendEmail sitzt vollstaendig in
  internal/email/send/service.go (consentAsserter) und ist dort bereits durch
  internal/email/send/consent_test.go abgedeckt (TestSend_BlockedByConsent,
  TestSend_AllowedByConsent, TestSend_NoContactID_SkipsConsentCheck) —
  HandleSendEmail selbst reicht ContactId nur unveraendert durch. Kein Fix-Unit-
  Vorschlag, weil bereits getestet, nur an der falschen Stelle vermutet. (3) Wie bei
  den bisherigen Email-/Video-Coverage-Units gibt es kein bufconn-Stub fuer
  EmailServiceClient in diesem Paket — alle ReachesRPC-Tests dokumentieren nur, dass
  Handler die lokale Validierung passieren und die RPC-Schicht erreichen (503 ueber
  die unerreichbare Dummy-Adresse), nicht das tatsaechliche Service-Verhalten.
  `.planning/backend-block/loop/run-loop.ps1` traegt weiterhin denselben unstaged
  -StartNotBefore-Diff wie in den Iterationen 6-49 vermerkt — nicht meine Datei, nicht
  angefasst, nicht committet. Laufkontext-Block war auch in diesem Prompt nicht sichtbar
  mitgeliefert — Nummer aus der letzten Journal-Ueberschrift (Iteration 49) fortgezaehlt,
  Zeitstempel per date auf dem Loop-Rechner ermittelt (2026-08-11 02:28).

## Iteration 51 — d-cov-gateway-calendar-membership — done — 2026-08-11 02:35
- commit: 3a0d027d
- gebaut: Neue Testdatei `backend/internal/gateway/route_calendar_membership_test.go`
  (34 Tests) fuer HandleGetCalendar/HandleUpdateCalendar/HandleDeleteCalendar
  (ServiceUnavailable, InvalidIDUUID ueber validateUUIDParam, InvalidJSON bei
  Update, ReachesRPC je), HandleAddCalendarMember (ServiceUnavailable,
  InvalidJSON, MissingUserID, InvalidUserIDUUID, MissingPermission,
  InvalidCalendarIDUUID_ReachesRPC, ReachesRPC), HandleRemoveCalendarMember
  (ServiceUnavailable, InvalidCalendarIDUUID_ReachesRPC,
  InvalidUserIDUUID_ReachesRPC, ReachesRPC), HandleUpdateCalendarMemberPermission
  (ServiceUnavailable, InvalidJSON, MissingPermission,
  InvalidPermissionLevel_ReachesRPC, ReachesRPC), HandleSubscribeToCalendar und
  HandleUnsubscribeFromCalendar (ServiceUnavailable, ReachesRPC je).
- gate: build ok (go build -p 2 ./internal/gateway/... ./cmd/gateway/...) | vet ok | lint ok
  (golangci-lint run --config .golangci.yml ./internal/gateway/... -- 0 issues) | test ok
  (go test -count=1 ./internal/gateway/ komplett gruen, 0 SKIP, 0 FAIL) | migration n.a.
  (keine neue Tabelle/Route) | rls-smoke n.a. (keine Tabelle/Policy angefasst) |
  TestOpenAPIRouteDrift lief als Teil des Pakettests mit, unveraendert gruen — keine neue
  Route, kein neuer RequirePermission-Guard, keine neue config.RequireX-Assertion
- coverage: internal/gateway 34,9 % -> 40,3 % (go test -coverprofile + go tool cover -func)
- mutations-probe: eine Probe, gefangen. In HandleDeleteCalendar `if !ok { return }`
  nach `validateUUIDParam` zu `if ok { return }` invertiert (bricht bei gueltiger
  UUID fruehzeitig ohne Response ab, laesst eine ungueltige durch) ->
  TestHandleDeleteCalendar_InvalidIDUUID (JSON-Decode-Fehler, weil kein Error-Body
  geschrieben wurde) UND TestHandleDeleteCalendar_ReachesRPC (200 statt 503) beide
  rot. Zurueckgedreht, `git diff --stat backend/internal/gateway/route_calendar.go`
  danach leer, build/vet/lint/test erneut komplett gruen.
- verify vorgaenger: sauber. Commit 071b5bf1 (Iteration 50) sowie der Metadaten-Commit
  6c088292 fuegen ausschliesslich eine Testdatei plus Journal-/Backlog-Metadaten hinzu —
  keine Produktionscode-Datei, kein Proto, keine Route, kein RequirePermission-Guard, keine
  neue Tabelle, keine Migration. Keine der acht Fehlerklassen einschlaegig.
- offen: (1) HandleAddCalendarMember/HandleRemoveCalendarMember/
  HandleUpdateCalendarMemberPermission/HandleSubscribeToCalendar/
  HandleUnsubscribeFromCalendar lesen "id" (und "userId") durchgaengig ueber rohes
  chi.URLParam ohne validateUUIDParam — kein neuer Befund, sondern derselbe, den
  Iteration 6 (fix-gateway-id-validation-consistency) bereits als eine der 161
  verbleibenden Rohstellen fuer route_calendar.go katalogisiert und explizit fuer
  eine Lauf-9-Folge-Unit vorgemerkt hat; hier nur mit *_ReachesRPC-Tests belegt statt
  gefixt (Coverage-Unit aendert kein Verhalten), kein zweiter Fix-Unit-Vorschlag noetig.
  (2) updateCalendarMemberPermissionRequest.Permission traegt nur `validate:"required"`,
  kein Enum-/Oneof-Check gegen die gueltigen CalendarPermission-Werte — ein
  semantisch unbekannter aber nicht-leerer Wert wird lokal nicht abgelehnt, sondern
  erreicht die RPC-Schicht (TestHandleUpdateCalendarMemberPermission_
  InvalidPermissionLevel_ReachesRPC dokumentiert das). Gleiche Kategorie wie (1),
  kein eigener Fix-Unit-Vorschlag, da folgenlos solange der Service serverseitig
  validiert (nicht gegengeprueft, ausserhalb des Scopes dieser Coverage-Unit). (3)
  Wie bei den bisherigen Gateway-Coverage-Units kein bufconn-Stub fuer
  CalendarServiceClient — alle ReachesRPC-Tests dokumentieren nur, dass Handler die
  lokale Validierung passieren und die RPC-Schicht erreichen (503 ueber die
  unerreichbare Dummy-Adresse), nicht das tatsaechliche Service-Verhalten.
  `.planning/backend-block/loop/run-loop.ps1` traegt weiterhin denselben unstaged
  -StartNotBefore-Diff wie in den Iterationen 6-50 vermerkt — nicht meine Datei, nicht
  angefasst, nicht committet. Laufkontext-Block war auch in diesem Prompt nicht sichtbar
  mitgeliefert — Nummer aus der letzten Journal-Ueberschrift (Iteration 50) fortgezaehlt,
  Zeitstempel per date auf dem Loop-Rechner ermittelt (2026-08-11 02:35).

## Iteration 52 — d-cov-gateway-calendar-events-resources — done — 2026-08-11 02:41
- commit: b3788f5c
- gebaut: Neue Testdatei `backend/internal/gateway/route_calendar_events_resources_test.go`
  (42 Tests) fuer die acht Event-/Ressourcen-Handler in route_calendar.go:
  HandleListEventsInRange (ServiceUnavailable, MissingStart, MissingEnd,
  InvalidStartFormat, InvalidEndFormat, InvertedRange_ReachesRPC, ReachesRPC),
  HandleGetEvent (ServiceUnavailable, InvalidIDUUID, ReachesRPC), HandleUpdateEvent
  (ServiceUnavailable, InvalidIDUUID, InvalidJSON, InvalidStartTimeFormat, ReachesRPC),
  HandleDeleteEvent (ServiceUnavailable, InvalidIDUUID, ReachesRPC), HandleCreateResource
  (ServiceUnavailable, InvalidJSON, MissingName, MissingResourceType, InvalidCapacity,
  ReachesRPC), HandleUpdateResource (ServiceUnavailable, InvalidIDUUID, InvalidJSON,
  InvalidCapacity, ReachesRPC), HandleBookResource (ServiceUnavailable, InvalidJSON,
  InvalidResourceIDUUID, InvalidEventIDUUID, MissingStartTime, InvalidStartTimeFormat,
  ConflictingRange_ReachesRPC, ReachesRPC), HandleCancelBooking (ServiceUnavailable,
  InvalidIDUUID, ReachesRPC).
- gate: build ok (go build -p 2 ./internal/gateway/... ./cmd/gateway/...) | vet ok | lint ok
  (golangci-lint run --config .golangci.yml ./internal/gateway/... -- 0 issues) | test ok
  (go test -count=1 ./internal/gateway/ -v: 1641 PASS, 3 SKIP (DATABASE_URL nicht gesetzt,
  vorbestehende RLS-Integrationstests in rls_dashboard_defaults_test.go, unveraendert durch
  diese Unit), 0 FAIL) | migration n.a. (keine neue Tabelle/Route) | rls-smoke n.a. (keine
  Tabelle/Policy angefasst) | TestOpenAPIRouteDrift separat gelaufen, unveraendert gruen
  (834 Routen gegen 836 Spec-Pfade) | keine neue Route, kein neuer RequirePermission-Guard,
  keine neue config.RequireX-Assertion
- coverage: internal/gateway 34,9 % -> 40,8 % (go test -coverprofile + go tool cover -func)
- mutations-probe: eine Probe, gefangen. In HandleListEventsInRange die Bedingung
  `if startStr == "" || endStr == ""` zu `if startStr == "" && endStr == ""` geschwaecht
  (akzeptiert jetzt einen einzelnen fehlenden Parameter) ->
  TestHandleListEventsInRange_MissingStart UND TestHandleListEventsInRange_MissingEnd
  beide rot (400 "invalid start/end time format" statt der erwarteten
  "start and end query parameters are required"-Meldung, weil der jeweils leere String
  ungeprueft in parseTimestamp lief und dort scheiterte). Per Edit-Tool zurueckgedreht,
  `git diff --stat backend/internal/gateway/route_calendar.go` danach leer, build/vet/lint/
  test erneut komplett gruen (1641 PASS, 3 SKIP, 0 FAIL).
- verify vorgaenger: sauber. Commit 3a0d027d (Iteration 51) sowie der Metadaten-Commit
  590e792b fuegen ausschliesslich eine Testdatei plus Journal-/Backlog-Metadaten hinzu —
  keine Produktionscode-Datei, kein Proto, keine Route, kein RequirePermission-Guard, keine
  neue Tabelle, keine Migration. Keine der acht Fehlerklassen einschlaegig.
- offen: (1) Das Backlog-`done_when` erwartete, dass HandleListEventsInRange "fehlende oder
  invertierte Von/Bis-Parameter als Fehlerfall" prueft. Gepruefte Realitaet: der Handler
  vergleicht start/end nach dem Parsen nie miteinander (route_calendar.go:561-603,
  parseTimestamp in route_work.go:256 kennt auch keine Reihenfolge) -- eine Anfrage mit
  start nach end erreicht unveraendert die RPC-Schicht. Kein Fix-Unit-Vorschlag: eine
  Coverage-Unit aendert kein Verhalten, TestHandleListEventsInRange_InvertedRange_ReachesRPC
  haelt die tatsaechliche Luecke fest statt sie stillschweigend zu unterstellen. Fuer Lauf 9
  vormerkbar, aber kein verifizierter Produktionsbug wie die Block-A-Funde -- ob eine
  Inversionspruefung ueberhaupt Produktwert hat (der Service ignoriert sie ohnehin nicht
  zwingend falsch), ist eine Produktfrage, keine offensichtliche Luecke. (2) Analog fuer
  HandleBookResource/HandleCancelBooking: "Buchungskonflikt-Pruefung" existiert im Handler
  nicht, Konfliktbehandlung ist vollstaendig serverseitig (BookResource/CancelBooking RPCs).
  TestHandleBookResource_ConflictingRange_ReachesRPC dokumentiert das, kein eigener Befund,
  da konsistent mit jeder anderen ReachesRPC-Coverage-Unit in diesem Paket. (3) Wie bei allen
  bisherigen Gateway-Coverage-Units kein bufconn-Stub fuer CalendarServiceClient -- alle
  ReachesRPC-Tests dokumentieren nur, dass Handler die lokale Validierung passieren und die
  RPC-Schicht erreichen (503 ueber die unerreichbare Dummy-Adresse localhost:0), nicht das
  tatsaechliche Service-Verhalten.
  `.planning/backend-block/loop/run-loop.ps1` traegt weiterhin denselben unstaged
  -StartNotBefore-Diff wie in den Iterationen 6-51 vermerkt — nicht meine Datei, nicht
  angefasst, nicht committet. Laufkontext-Block war auch in diesem Prompt nicht sichtbar
  mitgeliefert — Nummer aus der letzten Journal-Ueberschrift (Iteration 51) fortgezaehlt,
  Zeitstempel per date auf dem Loop-Rechner ermittelt (2026-08-11 02:41).

## Iteration 53 — d-cov-gateway-inventar-items-stock — done — 2026-08-11 02:51
- commit: ed7e9b1c
- gebaut: Testdatei `backend/internal/gateway/route_inventar_test.go` um 26 Tests fuer die
  sieben Scope-Handler in route_inventar.go erweitert: HandleListItems (ServiceUnavailable,
  MissingTenant, ReachesRPC mit search/low_stock/location-Query), HandleGetItem/
  HandleUpdateItem/HandleDeleteItem (je InvalidIDUUID, ServiceUnavailable, ReachesRPC;
  HandleUpdateItem zusaetzlich InvalidJSON), HandleAdjustStock (InvalidIDUUID,
  InvalidPerformedByUUID via assertValidationError, InvalidDeltaType als einziger lokal
  geprueften Fehlerpfad fuer eine unbrauchbare Mengenaenderung, MissingDelta_ReachesRPC als
  dokumentierter Befund, ServiceUnavailable, ReachesRPC), HandleListMovements/
  HandleGetStockHistory (je InvalidIDUUID, ServiceUnavailable, ReachesRPC) sowie
  TestProtoListMovements_WireShape (direkter response.Proto-Marshal-Test fuer
  ListMovementsResponse, geteilt von beiden Handlern).
- gate: build ok (go build -p 2 ./internal/gateway/... ./cmd/gateway/...) | vet ok | lint ok
  (golangci-lint run --config .golangci.yml ./internal/gateway/... -- 0 issues) | test ok
  (go test -count=1 ./internal/gateway/ -v: 1670 PASS, 0 SKIP, 0 FAIL; ein einzelner FAIL beim
  allerersten Lauf dieser Iteration war nicht reproduzierbar -- vier weitere Wiederholungen
  direkt danach komplett gruen, keine der 26 neuen Tests betroffen, vermutlich eine
  vorbestehende zeitkritische Flakiness in einem anderen Testfall desselben Pakets) |
  migration n.a. (keine neue Tabelle/Route) | rls-smoke n.a. (keine Tabelle/Policy angefasst) |
  TestOpenAPIRouteDrift separat gelaufen, unveraendert gruen (834 Routen gegen 836
  Spec-Pfade) | keine neue Route, kein neuer RequirePermission-Guard, keine neue
  config.RequireX-Assertion
- coverage: internal/gateway 34,9 % -> 41,2 % (go test -coverprofile + go tool cover -func)
- mutations-probe: eine Probe, gefangen. In HandleAdjustStock `if !ok { return }` nach
  `validateUUIDParam` zu `if ok { return }` invertiert (bricht bei gueltiger UUID fruehzeitig
  ohne Response ab, laesst eine ungueltige durch) -> TestHandleAdjustStock_InvalidIDUUID,
  TestHandleAdjustStock_InvalidPerformedByUUID, TestHandleAdjustStock_InvalidDeltaType,
  TestHandleAdjustStock_MissingDelta_ReachesRPC UND TestHandleAdjustStock_ReachesRPC alle fuenf
  rot (TestHandleAdjustStock_ServiceUnavailable blieb gruen, da der Client-Check davor greift).
  Per Edit-Tool zurueckgedreht, `git diff --stat backend/internal/gateway/route_inventar.go`
  danach leer, build/vet/lint/test erneut komplett gruen.
- verify vorgaenger: sauber. Commit b3788f5c (Iteration 52) fuegt ausschliesslich eine
  Testdatei plus Journal-/Backlog-Metadaten hinzu (der Metadaten-Commit 68515a77 nur
  BACKLOG.yml/JOURNAL.md) — keine Produktionscode-Datei, kein Proto, keine Route, kein
  RequirePermission-Guard, keine neue Tabelle, keine Migration. Keine der acht Fehlerklassen
  einschlaegig.
- offen: (1) ECHTER BEFUND, NICHT GEFIXT (Coverage-Units bauen laut Backlog-Kopf keine
  Verhaltensaenderungen): `adjustStockRequest.Delta` (route_inventar.go, Zeile ~189) traegt
  keinen `validate`-Tag — anders als `TransferStockRequest.Quantity` (`validate:"gt=0"`,
  Zeile ~197) direkt darunter im selben Handler-Cluster. Ein Request ohne `delta`-Feld oder mit
  `delta:0` wird nicht lokal abgelehnt, sondern erreicht die RPC-Schicht unveraendert als
  Nullaenderung — TestHandleAdjustStock_MissingDelta_ReachesRPC dokumentiert das. Ob delta=0
  serverseitig ueberhaupt sinnvoll behandelt wird (z. B. als No-Op-Movement-Eintrag), ist nicht
  Teil dieser Coverage-Unit; vorgemerkt fuer eine Lauf-9-Fix-Unit, falls Luke das als echte
  Produktluecke einstuft (kein verifizierter Bug wie die Block-A-Funde, nur eine Inkonsistenz
  gegenueber dem Nachbar-Handler). (2) TestProtoListMovements_WireShape bestaetigt fuer
  ListMovementsResponse (von HandleListMovements UND HandleGetStockHistory geteilt) denselben
  Befund wie bereits fuer ListRecordingsResponse/ListMeetingOccurrencesResponse dokumentiert:
  `EmitUnpopulated=false` im gemeinsamen protoMarshaler (internal/server/response/response.go)
  laesst eine leere `movements`-Liste vollstaendig aus dem JSON-Body verschwinden (`{}` statt
  `{"movements":[]}`), statt `[]` zu serialisieren — kein `null`, aber auch kein Array-Schluessel,
  den ein FE-Consumer blind mappen koennte. Kein neuer Fix-Unit-Vorschlag, da systemweit
  (jeder `response.Proto`-Aufruf mit leerem Repeated-Feld), nicht spezifisch fuer Inventar; bereits
  an anderer Stelle als bekannte, tolerierte Eigenschaft dokumentiert. (3) Wie bei allen
  bisherigen Gateway-Coverage-Units in diesem Paket kein bufconn-Stub fuer InventarServiceClient
  — alle ReachesRPC-Tests dokumentieren nur, dass der Handler die lokale Validierung passiert
  und die RPC-Schicht erreicht (503 ueber die unerreichbare Dummy-Adresse localhost:0), nicht
  das tatsaechliche Service-Verhalten.
  `.planning/backend-block/loop/run-loop.ps1` traegt weiterhin denselben unstaged
  -StartNotBefore-Diff wie in den Iterationen 6-52 vermerkt — nicht meine Datei, nicht
  angefasst, nicht committet. Laufkontext-Block war auch in diesem Prompt nicht sichtbar
  mitgeliefert — Nummer aus der letzten Journal-Ueberschrift (Iteration 52) fortgezaehlt,
  Zeitstempel per `date` auf dem Loop-Rechner ermittelt (2026-08-11 02:51).

## Iteration 54 — d-cov-gateway-inventar-inventur — done — 2026-08-11 02:53
- commit: a4f32d20
- gebaut: Testdatei `backend/internal/gateway/route_inventar_test.go` um 31 Tests fuer die
  sieben Inventur-Workflow-Handler in route_inventar.go erweitert: HandleCreateInventurSession
  (MissingName, InvalidLocationIDUUID, InvalidDateFormat als lokal geprueften Parse-Fehler,
  ServiceUnavailable, ReachesRPC), HandleUpdateInventurSessionStatus (InvalidIDUUID,
  InvalidStatusValue als dokumentierter Befund -- siehe offen --, ServiceUnavailable,
  ReachesRPC), HandleUpsertInventurCount (InvalidIDUUID, InvalidJSON, MissingItemID,
  ServiceUnavailable, ReachesRPC), HandleBookInventurDifferences (InvalidIDUUID wie im
  done_when gefordert vor dem Buchen, InvalidBookedByUUID, ServiceUnavailable, ReachesRPC),
  HandleListWarnings (ServiceUnavailable, MissingTenant, ReachesRPC mit status-Query),
  HandleUpdateWarning (InvalidIDUUID, InvalidJSON als einziger lokal geprueften Fehlerpfad,
  ServiceUnavailable, ReachesRPC), HandleAcknowledgeWarning (InvalidIDUUID, InvalidJSON,
  InvalidAcknowledgedByUUID via assertValidationError, ServiceUnavailable,
  ReachesRPC_FallsBackToAuthenticatedUser fuer den User-ID-Fallback-Zweig).
- gate: build ok (go build -p 2 ./internal/gateway/... ./cmd/gateway/...) | vet ok | lint ok
  (golangci-lint run --config .golangci.yml ./internal/gateway/... -- 0 issues) | test ok
  (go test -count=1 ./internal/gateway/ -v: 1700 PASS, 0 SKIP, 0 FAIL) | migration n.a. (keine
  neue Tabelle/Route) | rls-smoke n.a. (keine Tabelle/Policy angefasst) |
  TestOpenAPIRouteDrift separat gruen (834 Routen gegen 836 Spec-Pfade, unveraendert) | keine
  neue Route, kein neuer RequirePermission-Guard, keine neue config.RequireX-Assertion
- coverage: internal/gateway 34,9 % -> 41,6 % (go test -coverprofile + go tool cover -func)
- mutations-probe: eine Probe, gefangen. In HandleBookInventurDifferences
  `id, ok := validateUUIDParam(...); if !ok { return }` zu `if ok { return }` invertiert
  (bricht bei gueltiger UUID fruehzeitig ohne Response ab, laesst eine ungueltige durch) ->
  TestHandleBookInventurDifferences_InvalidIDUUID, _InvalidBookedByUUID UND _ReachesRPC alle
  drei rot (_ServiceUnavailable blieb gruen, da der Client-Check davor greift). Per Edit-Tool
  zurueckgedreht, `git diff --stat backend/internal/gateway/route_inventar.go` danach leer,
  build/vet/lint/test erneut komplett gruen.
- verify vorgaenger: sauber. Commit ed7e9b1c (Iteration 53) fuegt ausschliesslich eine
  Testdatei plus Journal-/Backlog-Metadaten hinzu (der Metadaten-Commit 14dd7809 nur
  BACKLOG.yml/JOURNAL.md) — keine Produktionscode-Datei, kein Proto, keine Route, kein
  RequirePermission-Guard, keine neue Tabelle, keine Migration. Keine der acht Fehlerklassen
  einschlaegig.
- offen: (1) ECHTER BEFUND, NICHT GEFIXT (Coverage-Units bauen laut Backlog-Kopf keine
  Verhaltensaenderungen): `updateInventurSessionStatusRequest.Status` (route_inventar.go,
  Zeile ~818) validiert nur Mitgliedschaft in der festen oneof-Liste
  (open/counting/review/completed) -- der Handler prueft NIE, ob der Uebergang VOM aktuellen
  Sitzungsstatus aus zulaessig ist (z. B. von "completed" zurueck auf "open"). Das
  done_when dieser Unit erwartete, dass "einen ungueltigen Statusuebergang als Fehlerfall"
  geprueft wird -- gepruefte Realitaet: es gibt im Gateway-Handler keine
  Uebergangspruefung, nur eine Werte-Pruefung. TestHandleUpdateInventurSessionStatus_
  InvalidStatusValue dokumentiert die tatsaechlich vorhandene Werte-Pruefung (Wert
  "cancelled" nicht im oneof), nicht die im done_when unterstellte
  Uebergangs-Zustandsmaschine. Ob eine echte Uebergangspruefung serverseitig existiert
  (internal/inventar/service.go, nicht Teil dieser Coverage-Unit) oder Produktwert haette,
  ist eine Produktfrage; vorgemerkt fuer Lauf 9, falls Luke das als echte Luecke einstuft.
  (2) Analog `updateWarningRequest.Status` (Zeile ~216) traegt gar keinen validate-Tag --
  jeder String erreicht HandleUpdateWarning unveraendert die RPC-Schicht, nur malformed JSON
  wird lokal abgelehnt. Kein eigener Fix-Unit-Vorschlag, konsistent mit dem
  MissingDelta-Befund aus Iteration 53 (Inkonsistenz gegenueber Nachbar-Handlern mit
  strengerem Tag, kein verifizierter Produktionsbug). (3) Wie bei allen bisherigen
  Gateway-Coverage-Units kein bufconn-Stub fuer InventarServiceClient -- alle ReachesRPC-Tests
  dokumentieren nur, dass der Handler die lokale Validierung passiert und die RPC-Schicht
  erreicht (503 ueber die unerreichbare Dummy-Adresse localhost:0), nicht das tatsaechliche
  Service-Verhalten.
  `.planning/backend-block/loop/run-loop.ps1` traegt weiterhin denselben unstaged
  -StartNotBefore-Diff wie in den Iterationen 6-53 vermerkt — nicht meine Datei, nicht
  angefasst, nicht committet. Laufkontext-Block war auch in diesem Prompt nicht sichtbar
  mitgeliefert — Nummer aus der letzten Journal-Ueberschrift (Iteration 53) fortgezaehlt,
  Zeitstempel per `date` auf dem Loop-Rechner ermittelt (2026-08-11 02:53).

## Iteration 55 — d-cov-gateway-document-shares — done — 2026-08-11 03:04
- commit: 7e71e8e3
- gebaut: `backend/internal/gateway/route_document_test.go` um 29 neue Tests fuer die
  Share-Link- und Entity-Link-Handler in route_document.go erweitert:
  HandleListShareLinks (ServiceUnavailable, InvalidIDReachesRPC), HandleCreateShareLink
  (ServiceUnavailable, InvalidJSON, InvalidIDReachesRPC, NoCreatedByWhenAnonymous),
  HandleRevokeShareLink (ServiceUnavailable, InvalidIDReachesRPC), HandleGetSharedFile
  (NoAuthNeeded, EmptyToken, MalformedTokenLooksLikeUnknown, BodyIsOptional,
  InvalidJSONBody), HandleLinkFileToEntity (ServiceUnavailable, InvalidJSON,
  MissingEntityID, MissingEntityType), HandleUnlinkFileFromEntity (ServiceUnavailable,
  InvalidJSON, MissingEntityID), HandleListFileEntityLinks (ServiceUnavailable,
  InvalidIDReachesRPC).
- gate: build ok (go build -p 2 ./internal/gateway/... ./cmd/gateway/...) | vet ok | lint ok
  (golangci-lint run --config .golangci.yml ./internal/gateway/... -- 0 issues) | test ok
  (go test -count=1 ./internal/gateway/ -v: 1722 PASS, 0 SKIP, 0 FAIL) | migration n.a.
  (keine neue Tabelle/Route) | rls-smoke n.a. (keine Tabelle/Policy angefasst) |
  TestOpenAPIRouteDrift separat gruen (834 Routen gegen 836 Spec-Pfade, unveraendert) |
  keine neue Route, kein neuer RequirePermission-Guard, keine neue
  config.RequireX-Assertion
- coverage: internal/gateway 34,9 % -> 41,9 % (go test -coverprofile + go tool cover -func)
- mutations-probe: eine Probe, gefangen. In HandleGetSharedFile `token == "" ||
  len(token) > 128` zu `token == ""` verkuerzt (die Laengenpruefung entfernt) ->
  TestHandleGetSharedFile_MalformedTokenLooksLikeUnknown rot (200-Zeichen-Token erreicht
  jetzt die RPC-Schicht statt lokal 404 zu liefern, Body zeigt "connection error" statt
  "share link not found"; die vier anderen HandleGetSharedFile-Tests blieben gruen, da sie
  den leeren bzw. kurzen Token pruefen). Per Edit-Tool zurueckgedreht, `git diff --stat
  backend/internal/gateway/route_document.go` danach leer, build/vet/lint/test erneut
  komplett gruen.
- verify vorgaenger: sauber. Commit a4f32d20 (Iteration 54) fuegt ausschliesslich eine
  Testdatei plus Journal-/Backlog-Metadaten hinzu (der Metadaten-Commit 8c20db0e nur
  JOURNAL.md) — keine Produktionscode-Datei, kein Proto, keine Route, kein
  RequirePermission-Guard, keine neue Tabelle, keine Migration. Keine der acht Fehlerklassen
  einschlaegig.
- offen: ECHTER BEFUND, NICHT GEFIXT (Coverage-Units bauen laut Backlog-Kopf keine
  Verhaltensaenderungen): HandleListShareLinks, HandleCreateShareLink, HandleRevokeShareLink
  und HandleListFileEntityLinks in route_document.go lesen `chi.URLParam(r, "id")` an keiner
  Stelle ueber `validateUUIDParam` (bestaetigt per Test: eine nicht-UUID-Id erreicht die
  RPC-Schicht statt lokal 400 zu liefern). Das ist derselbe, bereits in Iteration 6
  (fix-gateway-id-validation-consistency) dokumentierte Gateway-weite Befund — dort wurde
  route_document.go explizit als eine der 24 verbleibenden Dateien mit rohen
  chi.URLParam(r, "id")-Stellen genannt (161 Stellen gesamt), aber nicht selbst gefixt.
  Kein neuer Fix-Unit-Vorschlag hier, konsistent mit der damaligen Entscheidung, das fuer
  Lauf 9 zu buendeln statt einzeln anzulegen.
  HandleGetSharedFile ist per Code-Review (nicht per zusaetzlichem Test) als schmaler
  Wire-Typ bestaetigt: `documentv1.GetSharedFileResponse` traegt bereits nur
  download_url/filename/content_type/file_size (proto/document/v1/document.pb.go:4451-4459),
  keine tenant_id oder sonstiges Feld, und der Handler baut die Antwort ohnehin manuell aus
  genau diesen vier Feldern (route_document.go:1023-1028) statt das rohe Proto zu
  serialisieren — die Projektregel ist damit erfuellt, ohne dass ein Happy-Path-Test noetig
  waere. Ein echter Happy-Path-Test war wie bei allen bisherigen Gateway-Coverage-Units nicht
  moeglich: kein bufconn-Stub fuer DocumentServiceClient im Repo, alle ReachesRPC-Tests
  dokumentieren nur, dass der Handler die lokale Validierung passiert und die RPC-Schicht
  erreicht (503 ueber die unerreichbare Dummy-Adresse localhost:0).
  `.planning/backend-block/loop/run-loop.ps1` traegt weiterhin denselben unstaged
  -StartNotBefore-Diff wie in den Iterationen 6-54 vermerkt — nicht meine Datei, nicht
  angefasst, nicht committet. Laufkontext-Block war auch in diesem Prompt nicht sichtbar
  mitgeliefert — Nummer aus der letzten Journal-Ueberschrift (Iteration 54) fortgezaehlt,
  Zeitstempel per `date` auf dem Loop-Rechner ermittelt (2026-08-11 03:04).

## Iteration 56 — d-cov-gateway-document-file-lifecycle — done — 2026-08-11 03:09
- commit: 6f1f3b31
- gebaut: `backend/internal/gateway/route_document_test.go` um 22 neue Tests fuer die
  Datei-Lebenszyklus-Handler in route_document.go erweitert: HandleRegisterUploadedFile
  (ServiceUnavailable, InvalidJSON, MissingFolderID, ReachesRPC), HandleDeleteFile
  (ServiceUnavailable, InvalidIDReachesRPC), HandleCopyFile (ServiceUnavailable, InvalidJSON,
  MissingTargetFolderID, ReachesRPC), HandleMoveFile (ServiceUnavailable, InvalidJSON,
  MissingTargetFolderID, ReachesRPC), HandleGetFileDownloadURL (ServiceUnavailable,
  InvalidIDReachesRPC), HandleListFileVersions (ServiceUnavailable, InvalidIDReachesRPC),
  HandleRevertFileVersion (ServiceUnavailable, InvalidJSON, InvalidVersionNumber, ReachesRPC).
- gate: build ok (go build -p 2 ./internal/gateway/... ./cmd/gateway/...) | vet ok | lint ok
  (golangci-lint run --config .golangci.yml ./internal/gateway/... -- 0 issues) | test ok
  (go test -count=1 ./internal/gateway/ -v: 1744 PASS, 0 SKIP, 0 FAIL) | migration n.a. (keine
  neue Tabelle/Route) | rls-smoke n.a. (keine Tabelle/Policy angefasst) |
  TestOpenAPIRouteDrift separat gruen (834 Routen gegen 836 Spec-Pfade, unveraendert) | keine
  neue Route, kein neuer RequirePermission-Guard, keine neue config.RequireX-Assertion
- coverage: internal/gateway 34,9 % -> 42,1 % (go test -coverprofile + go tool cover -func)
- mutations-probe: zwei Proben, erste verpuffte, zweite gefangen. Erster Versuch: in
  `copyFileRequest.TargetFolderID` das `validate`-Tag von `required,uuid` auf `uuid` verkuerzt
  -> alle Tests blieben gruen, weil ein leerer String schon an der `uuid`-Regel scheitert (die
  Probe testete nichts Neues, `required` und `uuid` ueberlappen bei leerem Input). Zurueckgedreht
  und durch eine echte Probe ersetzt: in HandleCopyFile `if !ok { return }` zu `if ok { return }`
  invertiert (bricht bei gueltiger Validierung fruehzeitig ab, laesst eine fehlgeschlagene
  Validierung durch) -> TestHandleCopyFile_MissingTargetFolderID UND TestHandleCopyFile_ReachesRPC
  beide rot (Response-Body doppelt geschrieben bzw. Status 200 statt 503). Per Edit-Tool
  zurueckgedreht, `git diff --stat backend/internal/gateway/route_document.go` danach leer,
  build/vet/lint/test erneut komplett gruen.
- verify vorgaenger: sauber. Commit 7e71e8e3 (Iteration 55) fuegt ausschliesslich eine
  Testdatei plus Journal-/Backlog-Metadaten hinzu (der Metadaten-Commit fe638e80 nur
  JOURNAL.md) — keine Produktionscode-Datei, kein Proto, keine Route, kein
  RequirePermission-Guard, keine neue Tabelle, keine Migration. Keine der acht Fehlerklassen
  einschlaegig.
- offen: (1) BACKLOG-DONE_WHEN-ABWEICHUNG: das done_when dieser Unit unterstellte fuer
  HandleRevertFileVersion eine "ungueltige Versions-ID (UUID)"-Pruefung. Tatsaechlich nimmt der
  Handler gar keinen Versions-ID-Parameter — `revertVersionRequest.VersionNumber` ist ein
  `int32` mit `validate:"gt=0"`, adressiert die Zielversion also ueber eine Nummer im JSON-Body,
  nicht ueber eine UUID im Pfad. TestHandleRevertFileVersion_InvalidVersionNumber dokumentiert
  die tatsaechlich vorhandene Nummer-Pruefung (0 wird abgelehnt) statt der im done_when
  unterstellten UUID-Pruefung. (2) ECHTER BEFUND, NICHT GEFIXT (Coverage-Units bauen laut
  Backlog-Kopf keine Verhaltensaenderungen, konsistent mit dem in Iteration 55 dokumentierten
  Befund): keiner der sieben in dieser Unit getesteten Handler (HandleRegisterUploadedFile
  betrifft das nicht, aber HandleDeleteFile, HandleCopyFile, HandleMoveFile,
  HandleGetFileDownloadURL, HandleListFileVersions, HandleRevertFileVersion) ruft
  `validateUUIDParam` auf die Datei-`id` aus `chi.URLParam(r, "id")` auf — dieselbe Gateway-weite
  Luecke aus Iteration 6 (fix-gateway-id-validation-consistency), route_document.go stand dort
  bereits als eine der 24 verbleibenden Dateien mit rohen chi.URLParam-Stellen. Kein neuer
  Fix-Unit-Vorschlag, konsistent mit der damaligen Entscheidung, das fuer Lauf 9 zu buendeln.
  `.planning/backend-block/loop/run-loop.ps1` traegt weiterhin denselben unstaged
  -StartNotBefore-Diff wie in den Iterationen 6-55 vermerkt — nicht meine Datei, nicht
  angefasst, nicht committet. Laufkontext-Block war auch in diesem Prompt nicht sichtbar
  mitgeliefert — Nummer aus der letzten Journal-Ueberschrift (Iteration 55) fortgezaehlt,
  Zeitstempel per `date` auf dem Loop-Rechner ermittelt (2026-08-11 03:09).

## Iteration 57 — d-cov-gateway-rapporte-lifecycle — done — 2026-08-11 03:16
- commit: 9bb0cae6
- gebaut: `backend/internal/gateway/route_rapporte_test.go` um 20 neue Tests fuer den
  Bericht-Lebenszyklus in route_rapporte.go erweitert: HandleListReports (MissingTenant,
  ServiceUnavailable, ReachesRPC, OwnScopeWithoutUserIsRejected), HandleGetReport
  (InvalidIDUUID, MissingTenant, ReachesRPC), HandleUpdateReport (InvalidIDUUID,
  InvalidJSON, MissingTenant, ReachesRPC), HandleDeleteReport (InvalidIDUUID,
  MissingTenant, ServiceUnavailable), HandleSubmitReport (InvalidIDUUID, MissingTenant,
  ReachesRPCWithInvalidStatusTransition), HandleRejectReport (InvalidIDUUID,
  MissingReviewerID, MissingTenant, ReachesRPCWithInvalidStatusTransition).
- gate: build ok (go build -p 2 ./internal/gateway/... ./cmd/gateway/...) | vet ok | lint ok
  (golangci-lint run --config .golangci.yml ./internal/gateway/... -- 0 issues) | test ok
  (go test -count=1 ./internal/gateway/ -v: 1765 PASS, 0 SKIP, 0 FAIL) | migration n.a.
  (keine neue Tabelle/Route) | rls-smoke n.a. (keine Tabelle/Policy angefasst) |
  TestOpenAPIRouteDrift separat gruen (834 Routen gegen 836 Spec-Pfade, unveraendert) |
  keine neue Route, kein neuer RequirePermission-Guard, keine neue
  config.RequireX-Assertion
- coverage: internal/gateway 34,9 % -> 42,4 % (go test -coverprofile + go tool cover -func)
- mutations-probe: eine Probe, gefangen. In HandleSubmitReport `if !ok { return }` nach
  validateUUIDParam auf `if ok { return }` invertiert -> TestHandleSubmitReport_
  ReachesRPCWithInvalidStatusTransition rot (Status 200 statt 503: bei validem Report-ID
  kehrt der Handler jetzt sofort zurueck, ohne die RPC je zu erreichen, kein
  ResponseWriter-Write). TestHandleSubmitReport_InvalidIDUUID blieb dabei gruen (die 400
  aus validateUUIDParam ist bereits geschrieben, bevor der mutierte Zweig greift — kein
  falsches Gruen, sondern derselbe doppelte-WriteHeader-Effekt wie in Iteration 56
  dokumentiert). Per Edit-Tool zurueckgedreht, `git diff --stat
  backend/internal/gateway/route_rapporte.go` danach leer, build/vet/lint/test erneut
  komplett gruen.
- verify vorgaenger: sauber. Commit 6f1f3b31 (Iteration 56) fuegt ausschliesslich eine
  Testdatei plus Journal-/Backlog-Metadaten hinzu (der Metadaten-Commit db34d1eb nur
  JOURNAL.md) — keine Produktionscode-Datei, kein Proto, keine Route, kein
  RequirePermission-Guard, keine neue Tabelle, keine Migration. Keine der acht
  Fehlerklassen einschlaegig.
- offen: Kein Wire-Shape-Test fuer HandleListReports moeglich (Backlog-Wunsch
  "gewrappte Liste pruefen") — der Handler reicht `ListReportsResponse` unveraendert
  ueber `response.Proto` durch (route_rapporte.go:224-229), keine gateway-eigene
  Marshaling-Logik und kein bufconn-Stub fuer RapporteServiceClient in diesem Paket,
  dieselbe dokumentierte Grenze wie in jeder bisherigen Gateway-Coverage-Unit dieses
  Laufs (zuletzt Iteration 55/document, Iteration ~52/fuhrpark). Ebenso kein lokal
  testbarer "ungueltiger Statusuebergang" fuer HandleSubmitReport/HandleRejectReport:
  die draft->submitted->approved/rejected-Maschine lebt in der rapporte-RPC-Schicht,
  nicht im Handler — stattdessen ReachesRPC-Tests, die belegen, dass eine gueltige
  ID/Payload lokal durchlaeuft und die RPC-Schicht unveraendert erreicht (503 ueber die
  unerreichbare Dummy-Adresse). `HandleSaveReportSignature`, `HandleListLines`,
  `HandleUpdateLine`, `HandleDeleteLine`, `HandleDeleteAttachment`,
  `HandleGetReportStats`, `HandleListPendingApprovals`, `HandleExportPDF` sowie alle
  Measurement-/Template-Handler bleiben ungetestet — nicht im Scope dieser Unit, ggf.
  Kandidat fuer Lauf 9, falls internal/gateway noch weiter gehoben werden soll.
  `.planning/backend-block/loop/run-loop.ps1` traegt weiterhin denselben unstaged Diff
  wie in frueheren Iterationen vermerkt — nicht meine Datei, nicht angefasst, nicht
  committet. Laufkontext-Block war auch in diesem Prompt nicht sichtbar mitgeliefert —
  Nummer aus der letzten Journal-Ueberschrift (Iteration 56) fortgezaehlt, Zeitstempel
  per `date` auf dem Loop-Rechner ermittelt (2026-08-11 03:16).

## Iteration 58 — d-cov-gateway-helpdesk-csat-lifecycle — done — 2026-08-11 03:22
- commit: bd2d3e98
- gebaut: `backend/internal/gateway/route_helpdesk_test.go` um 26 neue Tests fuer
  Ticket-Statusuebergaenge und den oeffentlichen CSAT-Pfad in route_helpdesk.go
  erweitert: HandleListTickets (ServiceUnavailable, MissingTenant, ReachesRPC,
  OwnScopeWithoutUserIsRejected), HandleGetTicket (InvalidIDUUID, ServiceUnavailable,
  ReachesRPC), HandleCloseTicket (InvalidIDUUID, ServiceUnavailable, ReachesRPC),
  HandleReopenTicket (InvalidIDUUID, ServiceUnavailable, ReachesRPC), HandleSubmitCsat
  (ServiceUnavailable, InvalidIDUUID, InvalidJSON, RatingOutOfRange, ReachesRPC),
  HandleSubmitCsatByToken (ServiceUnavailable, EmptyToken, TokenTooLong, InvalidJSON,
  RatingOutOfRange, ReachesRPC).
- gate: build ok (go build -p 2 ./internal/gateway/... ./cmd/gateway/...) | vet ok |
  lint ok (golangci-lint run --config .golangci.yml ./internal/gateway/... -- 0
  issues) | test ok (go test -count=1 ./internal/gateway/ -v: 1789 PASS, 0 SKIP, 0
  FAIL) | migration n.a. (keine neue Tabelle/Route) | rls-smoke n.a. (keine Tabelle/
  Policy angefasst) | TestOpenAPIRouteDrift separat gruen (834 Routen gegen 836
  Spec-Pfade, unveraendert) | keine neue Route, kein neuer RequirePermission-Guard,
  keine neue config.RequireX-Assertion
- coverage: internal/gateway 34,9 % -> 42,7 % (go test -coverprofile + go tool cover -func)
- mutations-probe: eine Probe, gefangen. In HandleCloseTicket `if !ok { return }` nach
  validateUUIDParam auf `if ok { return }` invertiert -> TestHandleCloseTicket_
  ReachesRPC rot (Status 200 statt 503, leerer Body: bei validem Ticket-ID kehrt der
  Handler jetzt sofort zurueck, ohne die RPC je zu erreichen, kein ResponseWriter-
  Write). TestHandleCloseTicket_InvalidIDUUID blieb dabei gruen (die 400 aus
  validateUUIDParam ist bereits geschrieben, bevor der mutierte Zweig greift — kein
  falsches Gruen, derselbe dokumentierte Doppel-Write-Effekt wie in Iteration 57).
  Per Edit-Tool zurueckgedreht, `git diff --stat backend/internal/gateway/
  route_helpdesk.go` danach leer, build/vet/lint/test erneut komplett gruen.
- verify vorgaenger: sauber. Commit 9bb0cae6 (Iteration 57) fuegt ausschliesslich
  eine Testdatei plus Journal-/Backlog-Metadaten hinzu (Metadaten-Commit db34d1eb nur
  JOURNAL.md) — keine Produktionscode-Datei, kein Proto, keine Route, kein
  RequirePermission-Guard, keine neue Tabelle, keine Migration. Keine der acht
  Fehlerklassen einschlaegig.
- offen: HandleSubmitCsatByToken deckt nur den lokal aufloesbaren Token-Fehlerfall ab
  (leer/zu lang -> 404 vor der RPC); unbekannt/abgelaufen/widerrufen/bereits eingeloest
  sind laut Handler-Kommentar derselbe 404, aber RPC-seitig entschieden und ohne
  bufconn-Stub fuer HelpdeskServiceClient in diesem Paket lokal nicht scriptbar —
  dieselbe dokumentierte Grenze wie beim Statusuebergang in route_rapporte_test.go
  (Iteration 57). Kein Wire-Shape-Test fuer HandleListTickets moeglich: der Handler
  reicht die RPC-Response unveraendert ueber response.Proto durch, kein
  gateway-eigenes Marshaling. HandleUpdateTicket, HandleAssignTicket (Happy Path),
  HandleMergeTickets (Happy Path), HandleAddMessage/HandleListMessages, Queues,
  SLA-Policies, Routing-Rules, Business-Hours und HandleGetHelpdeskStats bleiben
  ungetestet — nicht im Scope dieser Unit, Kandidat fuer Lauf 9 falls internal/gateway
  weiter gehoben werden soll. `.planning/backend-block/loop/run-loop.ps1` traegt
  weiterhin denselben unstaged -StartNotBefore-Diff wie in den Iterationen 6-57
  vermerkt — nicht meine Datei, nicht angefasst, nicht committet. Laufkontext-Block
  war auch in diesem Prompt nicht sichtbar mitgeliefert — Nummer aus der letzten
  Journal-Ueberschrift (Iteration 57) fortgezaehlt, Zeitstempel per `date` auf dem
  Loop-Rechner ermittelt (2026-08-11 03:22).

## Iteration 59 — d-cov-gateway-formulare-schema-submission — done — 2026-08-11 03:27
- commit: 052b4bb5
- gebaut: `backend/internal/gateway/route_formulare_test.go` von einem reinen
  Konstruktor-Helfer auf 30 neue Tests fuer Formular-Schema-CRUD und den
  Submission-Workflow in route_formulare.go erweitert: HandleCreateFormSchema
  (ServiceUnavailable, MissingTenant, InvalidJSON, MissingTitle, ReachesRPC),
  HandleUpdateFormSchema (InvalidIDUUID, MissingTenant, InvalidJSON,
  ReachesRPC), HandleDeleteFormSchema (InvalidIDUUID, MissingTenant,
  ServiceUnavailable), HandleDuplicateFormSchema (InvalidIDUUID,
  MissingTenant, InvalidJSON, ReachesRPC), HandleCreateSubmission
  (InvalidIDUUID, MissingTenant, InvalidJSON, ServiceUnavailable,
  ReachesRPC), HandleUpdateSubmissionStatus (InvalidIDUUID, MissingTenant,
  InvalidJSON, MissingStatus, InvalidStatusValue, ReachesRPC).
- gate: build ok (go build -p 2 ./internal/gateway/... ./cmd/gateway/...) |
  vet ok | lint ok (golangci-lint run --config .golangci.yml
  ./internal/gateway/... -- 0 issues) | test ok (go test -count=1
  ./internal/gateway/ -v: 1816 PASS, 0 SKIP, 0 FAIL) | migration n.a. (keine
  neue Tabelle/Route) | rls-smoke n.a. (keine Tabelle/Policy angefasst) |
  TestOpenAPIRouteDrift separat gruen (834 Routen gegen 836 Spec-Pfade,
  unveraendert) | keine neue Route, kein neuer RequirePermission-Guard, keine
  neue config.RequireX-Assertion
- coverage: internal/gateway 34,9 % -> 43,1 % (go test -coverprofile + go
  tool cover -func)
- mutations-probe: eine Probe, gefangen. `createFormSchemaRequest.Title` von
  `validate:"required"` auf keinen Validate-Tag entfernt ->
  TestHandleCreateFormSchema_MissingTitle rot (503 + Dial-Fehler statt 400 +
  validation_failed: der leere Titel reicht jetzt bis zur RPC durch). Alle
  anderen 29 neuen Tests blieben gruen, insbesondere
  TestHandleCreateFormSchema_ReachesRPC (das erwartet ohnehin 503 und haette
  eine zu laxe Probe verschleiert). Per Edit-Tool zurueckgedreht, `git diff
  --stat backend/internal/gateway/route_formulare.go` danach leer,
  build/vet/lint/test erneut komplett gruen.
- verify vorgaenger: sauber. Commit bd2d3e98 (Iteration 58) fuegt
  ausschliesslich eine Testdatei plus Journal-/Backlog-Metadaten hinzu
  (Metadaten-Commit 093c68f1 nur JOURNAL.md/BACKLOG.yml) — keine
  Produktionscode-Datei, kein Proto, keine Route, kein
  RequirePermission-Guard, keine neue Tabelle, keine Migration. Keine der
  acht Fehlerklassen einschlaegig.
- offen: `createFormSchemaRequest.Fields`/`updateFormSchemaRequest.Fields`
  tragen keinen `validate`-Tag — "fehlende Pflichtfelder (Name/Feldtyp)" aus
  dem Backlog-Wunsch bezieht sich auf die Feldnamen INNERHALB des opaken
  Fields-JSON-Blobs, die laut Scope in der Schema-Validierung liegen; das ist
  service-seitige Logik ohne bufconn-Stub fuer FormulareServiceClient in
  diesem Paket, dieselbe dokumentierte Grenze wie in jeder bisherigen
  Gateway-Coverage-Unit dieses Laufs. `dispatchIntake` (privater Helfer, ruft
  intern GetFormSchema + runIntakeDispatch) bleibt ungetestet aus demselben
  Grund: CreateSubmission schlaegt schon an der RPC fehl, bevor dispatchIntake
  je erreicht wird. `HandleListFormSchemas`, `HandleGetFormSchema`,
  `HandleListSubmissions`, `HandleGetSubmission`, `HandleExportSubmissions`,
  alle Webhook-Handler (Create/Get/Update/Delete/ListDeliveries), Share-Link-
  Handler (List/Create/Revoke) und `HandleSubmitByShareToken` (der
  unauthentifizierte Public-Pfad) bleiben ungetestet — nicht im Scope dieser
  Unit (route_formulare.go hat 1204 Zeilen, deutlich mehr Handler als das
  Backlog-`scope` nannte), Kandidat fuer Lauf 9 falls internal/gateway weiter
  gehoben werden soll. `.planning/backend-block/loop/run-loop.ps1` traegt
  weiterhin denselben unstaged Diff wie in den Iterationen 6-58 vermerkt —
  nicht meine Datei, nicht angefasst, nicht committet. Laufkontext-Block war
  auch in diesem Prompt nicht sichtbar mitgeliefert — Nummer aus der letzten
  Journal-Ueberschrift (Iteration 58) fortgezaehlt, Zeitstempel per `date` auf
  dem Loop-Rechner ermittelt (2026-08-11 03:27).

## Iteration 60 — d-cov-gateway-settings-module-access — done — 2026-08-11 03:34
- commit: 5fc601af
- gebaut: `backend/internal/gateway/route_settings_module_access_test.go` neu
  angelegt mit 34 Tests fuer die RBAC-nahe Modul-Zugriffsgruppe in
  route_settings.go: HandleListModuleLeads (ServiceUnavailable,
  MissingTenant, ReachesRPC), HandleGetMyModuleLeads (ServiceUnavailable,
  MissingTenant, NoUserID, ReachesRPC), HandleGrantModuleLead
  (ServiceUnavailable, MissingTenant, NoCallerID, InvalidIDUUID,
  MissingModuleID, ReachesRPC), HandleRevokeModuleLead (ServiceUnavailable,
  MissingTenant, InvalidIDUUID, MissingModuleID, ReachesRPC),
  HandleListModuleGrants (ServiceUnavailable, MissingTenant, ReachesRPC),
  HandleGrantModuleAccess (ServiceUnavailable, MissingTenant, NoCallerID,
  InvalidIDUUID, MissingModuleID, ReachesRPC), HandleRevokeModuleAccess
  (ServiceUnavailable, MissingTenant, InvalidIDUUID, MissingModuleID,
  ReachesRPC), HandleBulkRevokeModuleAccess (ServiceUnavailable,
  MissingTenant, InvalidJSON, MissingPairs, ReachesRPC).
- gate: build ok (go build -p 2 ./internal/gateway/... ./cmd/gateway/...) |
  vet ok | lint ok (golangci-lint run --config .golangci.yml
  ./internal/gateway/... -- 0 issues) | test ok (go test -count=1
  ./internal/gateway/ -v: 1850 PASS, 0 FAIL, 3 SKIP [bekannte DB-abhaengige
  RLS-Tests]) | migration n.a. (keine neue Tabelle/Route) | rls-smoke n.a.
  (keine Tabelle/Policy angefasst) | TestOpenAPIRouteDrift separat gruen (834
  Routen gegen 836 Spec-Pfade, unveraendert) | keine neue Route, kein neuer
  RequirePermission-Guard, keine neue config.RequireX-Assertion
- coverage: internal/gateway 34,9 % -> 43,7 % (go test -coverprofile + go
  tool cover -func)
- mutations-probe: eine Probe, gefangen. In HandleGrantModuleLead
  `if moduleID == "" { ... }` auf `if false { ... }` invertiert ->
  TestHandleGrantModuleLead_MissingModuleID rot (200 statt 400: der leere
  module_id-Pfadparameter erreicht jetzt ungeprueft die RPC). Alle anderen 33
  neuen Tests blieben gruen. Per Edit-Tool zurueckgedreht, `git diff
  backend/internal/gateway/route_settings.go` danach leer, build/vet/lint/
  test erneut komplett gruen.
- verify vorgaenger: sauber. Commit 052b4bb5 (Iteration 59) fuegt
  ausschliesslich eine Testdatei plus Journal-/Backlog-Metadaten hinzu
  (Metadaten-Commit b66ceb19 nur JOURNAL.md) — keine Produktionscode-Datei,
  kein Proto, keine Route, kein RequirePermission-Guard, keine neue Tabelle,
  keine Migration. Keine der acht Fehlerklassen einschlaegig.
- offen: `bulkRevokeGrantsRequest.Pairs` traegt pro Element
  `validate:"required,uuid"` auf UserID / `validate:"required"` auf ModuleID,
  aber die go-playground/validator-Instanz in internal/validation validiert
  Struct-Felder innerhalb eines Slice NICHT automatisch ohne `dive`-Tag —
  per Scratch-Test verifiziert: ein Pair mit `UserID: "not-a-uuid"` besteht
  `Validate()` klaglos. HandleBulkRevokeModuleAccess reicht damit ungueltige
  UUIDs unvalidiert bis zur RPC durch (kein Sicherheitsloch, da die RPC selbst
  parsen muss, aber ein irrefuehrender Validate-Tag, der nichts tut). Nicht
  gefixt (Coverage-Unit baut keine Verhaltensaenderung) — Kandidat fuer Lauf 9:
  `dive` zum `pairs`-Tag hinzufuegen. Ebenso offen: HandleGetTenantLicense,
  HandleSetTenantModuleActive, HandleGetTenantSubscription,
  HandleGetBranding/HandlePutBranding und alle Settings-Handler
  (GetResolvedSettings/GetTenantSettings/PutTenantSettings/GetUserSettings/
  PutUserSettings) in derselben Datei bleiben ungetestet — nicht im Scope
  dieser Unit, Kandidat fuer Lauf 9. `.planning/backend-block/loop/
  run-loop.ps1` traegt weiterhin denselben unstaged -StartNotBefore-Diff wie
  in den Iterationen 6-59 vermerkt — nicht meine Datei, nicht angefasst,
  nicht committet. Laufkontext-Block war auch in diesem Prompt nicht
  sichtbar mitgeliefert — Nummer aus der letzten Journal-Ueberschrift
  (Iteration 59) fortgezaehlt, Zeitstempel per `date` auf dem Loop-Rechner
  ermittelt (2026-08-11 03:34).

## Iteration 61 — d-cov-gateway-chat-membership — done — 2026-08-11 03:35
- commit: bda88c7f3d38992ef708f7bdb4a3b2ffd23dcb5f
- gebaut: `backend/internal/gateway/route_chat_membership_test.go` neu
  angelegt mit 25 Tests fuer die Kanal-Mitgliedschafts- und
  Rollenverwaltungsgruppe in route_chat.go: HandleJoinChannel
  (ServiceUnavailable, NoUserID, InvalidUUID, ReachesRPC), HandleLeaveChannel
  (dieselben vier), HandleGetChannelMembers (ServiceUnavailable,
  InvalidUUID, ReachesRPC — kein eigener UserID-Check im Handler),
  HandleUpdateMemberRole (ServiceUnavailable, NoUserID, InvalidChannelID,
  InvalidTargetUserID, InvalidJSON, UnknownRole, MissingRole, ReachesRPC),
  HandleArchiveChannel und HandleDeleteChannel (je ServiceUnavailable,
  InvalidUUID, ReachesRPC).
- gate: build ok (go build -p 2 ./internal/gateway/... ./cmd/gateway/...) |
  vet ok | lint ok (golangci-lint run --config .golangci.yml
  ./internal/gateway/... -- 0 issues) | test ok (go test -count=1 -v
  ./internal/gateway/: 1878 PASS, 0 FAIL, 0 SKIP) | migration n.a. (keine
  neue Tabelle/Route) | rls-smoke n.a. (keine Tabelle/Policy angefasst) |
  TestOpenAPIRouteDrift im selben Paketlauf mitgelaufen, gruen (keine neue
  Route) | kein neuer RequirePermission-Guard, keine neue
  config.RequireX-Assertion
- coverage: internal/gateway 34,9 % -> 44,1 % (go test -coverprofile + go
  tool cover -func)
- mutations-probe: eine Probe, gefangen. In HandleUpdateMemberRole
  `if requesterID == "" { ... }` auf `if false { ... }` geaendert ->
  TestHandleUpdateMemberRole_NoUserID rot (503 statt 401: ohne Requester-ID
  erreicht der Request jetzt den gRPC-Client-Aufbau statt an der
  Auth-Grenze abzubrechen). Alle anderen 24 neuen Tests blieben gruen. Per
  Edit-Tool zurueckgedreht, `git diff backend/internal/gateway/route_chat.go`
  danach leer, build/vet/lint/test erneut komplett gruen.
- verify vorgaenger: sauber. Commit 5fc601af (Iteration 60) fuegt
  ausschliesslich eine Testdatei plus Journal-/Backlog-Metadaten hinzu
  (Metadaten-Commit 6c377749 nur JOURNAL.md) — keine Produktionscode-Datei,
  kein Proto, keine Route, kein RequirePermission-Guard, keine neue Tabelle,
  keine Migration. Keine der acht Fehlerklassen einschlaegig.
- offen: HandleGetChannelMembers und HandleArchiveChannel/HandleDeleteChannel
  rufen middleware.GetUserID nie explizit ab bzw. pruefen sie nicht auf Leere
  (anders als HandleUpdateMemberRole) — sie verlassen sich vollstaendig auf
  die RequireAuthenticated-Middleware der Route-Registrierung. Das ist heute
  konsistent mit dem restlichen Datei-Stil (HandleJoinChannel/
  HandleLeaveChannel machen es genauso), aber kein Handler-eigener Schutz;
  nicht gefixt, da Coverage-Unit keine Verhaltensaenderung baut. In
  route_chat.go bleiben nach dieser Unit noch HandleGetMessages,
  HandleUpdateMessage, HandleDeleteMessage, HandleListDMs,
  HandleGetThreadReplies, HandleMarkChannelRead, HandleGetUnreadCounts,
  HandleGetUserMentions, HandleGetFileDownloadURL/ThumbnailURL,
  HandleListChannelFiles, HandleDeleteFile, HandleToggleReaction/
  ListReactions/GetReactionSummary, HandleListBookmarks/ToggleBookmark und
  HandleSearchChat ungetestet — Kandidat fuer eine Folge-Unit in Lauf 9,
  nicht in dieser Unit gebaut. `.planning/backend-block/loop/run-loop.ps1`
  traegt weiterhin denselben unstaged -StartNotBefore-Diff wie in den
  Iterationen 6-60 vermerkt — nicht meine Datei, nicht angefasst, nicht
  committet. Laufkontext-Block war auch in diesem Prompt nicht sichtbar
  mitgeliefert — Nummer aus der letzten Journal-Ueberschrift (Iteration 60)
  fortgezaehlt, Zeitstempel per `date` auf dem Loop-Rechner ermittelt
  (2026-08-11 03:35).

## Iteration 62 — d-cov-gateway-vermietung-rental-lifecycle — done — 2026-08-11 03:46
- commit: e5bc7291ed379cec6fe688d6cf459926c61613a7
- gebaut: `backend/internal/gateway/route_vermietung_test.go` um 30 Tests
  erweitert fuer den bisher ungetesteten Vermietungs-Lebenszyklus in
  route_vermietung.go: HandleCheckAvailability (ServiceUnavailable,
  InvalidObjectIDUUID, MissingDates, InvalidStartDateFormat,
  InvalidEndDateFormat, OverlappingRange_ReachesRPC), HandleStartRental und
  HandleEndRental (je ServiceUnavailable, InvalidRentalIDUUID,
  InvalidStatusTransition_ReachesRPC), HandleUpdateRental
  (ServiceUnavailable, InvalidRentalIDUUID, InvalidJSON,
  InvalidStartDateFormat, InvalidEndDateFormat, ReachesRPC),
  HandleDeleteRental (ServiceUnavailable, InvalidRentalIDUUID, ReachesRPC)
  und HandleGetRentalCalendar (ServiceUnavailable, ReachesRPC,
  NonNumericYearMonthIgnored). Die "OverlappingRange"/"InvalidStatusTransition"
  -Tests dokumentieren bewusst, dass Ueberschneidungs- und
  Statusuebergangspruefung serverseitig im vermietung-Service liegen (kein
  bufconn-Stub im Repo fuer dieses Paket) — der Gateway-Handler validiert nur
  Tenant/ID/Datumsformat und reicht den Rest unveraendert an die RPC durch
  (503 bei localhost:0-Dummy-Verbindung, dasselbe Verify-Muster wie in
  Iteration 61).
- gate: build ok (go build -p 2 ./internal/gateway/... ./cmd/gateway/...) |
  vet ok | lint ok (golangci-lint run --config .golangci.yml
  ./internal/gateway/... -- 0 issues) | test ok (go test -count=1 -v
  ./internal/gateway/: 1899 PASS, 0 FAIL, 0 SKIP) | migration n.a. (keine
  neue Tabelle/Route) | rls-smoke n.a. (keine Tabelle/Policy angefasst) |
  TestOpenAPIRouteDrift im selben Paketlauf mitgelaufen, gruen (keine neue
  Route) | kein neuer RequirePermission-Guard, keine neue
  config.RequireX-Assertion
- coverage: internal/gateway 34,9 % -> 44,4 % (go test -coverprofile + go
  tool cover -func)
- mutations-probe: eine Probe, gefangen. In HandleUpdateRental den
  Start-Date-Formatfehler-Guard `if parseErr != nil { ... }` auf `if false`
  geaendert -> TestHandleUpdateRental_InvalidStartDateFormat rot (503 statt
  400: "gestern" erreicht ungeprueft die RPC und scheitert dort am
  localhost:0-Dial statt an der Datumsformat-Grenze). Alle anderen 29 neuen
  Tests blieben gruen. Per Edit-Tool zurueckgedreht, `git diff
  backend/internal/gateway/route_vermietung.go` danach leer, build/vet/lint/
  test erneut komplett gruen.
- verify vorgaenger: sauber. Commit bda88c7f (Iteration 61) fuegt
  ausschliesslich eine Testdatei plus Journal-/Backlog-Metadaten hinzu
  (`git show --stat bda88c7f`: BACKLOG.yml, JOURNAL.md,
  route_chat_membership_test.go) — keine Produktionscode-Datei, kein Proto,
  keine Route, kein RequirePermission-Guard, keine neue Tabelle, keine
  Migration. Keine der acht Fehlerklassen einschlaegig. Sein
  Journal-Platzhalter "(siehe git log nach diesem Commit)" war noch nicht
  aufgeloest — per separatem Commit (Konvention aus den Iterationen 59/60)
  vor dieser Unit nachgetragen: `chore(loop): record commit hash for
  iteration 61 journal entry`.
- offen: In route_vermietung.go bleiben nach dieser Unit noch
  HandleListObjects/HandleCreateObject/HandleGetObject/HandleUpdateObject/
  HandleDeleteObject, HandleListRentals/HandleCreateRental/HandleGetRental,
  HandleSaveRentalSignature, alle Inspection-Handler und
  HandleExportRentalReport ungetestet bzw. nur teilweise (Object/Rental/
  Inspection-Create sind in der bestehenden Testdatei abgedeckt, Get/List/
  Update/Delete/Export/Signature nicht) — Kandidat fuer eine Folge-Unit in
  Lauf 9, nicht in dieser Unit gebaut. Beim Lesen von helpers.go aufgefallen:
  es existiert bereits ein `validateDateParam`-Helfer (Zeile ~129) mit
  eigenem Format-Set, den route_vermietung.go nicht nutzt (eigenes
  `parseRFC3339` + manuelle RFC3339-Fehlermeldungen stattdessen) — keine
  Verhaltensaenderung in dieser Coverage-Unit, aber ein Kandidat fuer
  Root-Cause-Vereinheitlichung in einer spaeteren Fix-Unit, falls
  `validateDateParam` dasselbe Format akzeptiert.
  `.planning/backend-block/loop/run-loop.ps1` traegt weiterhin denselben
  unstaged -StartNotBefore-Diff wie in den Iterationen 6-61 vermerkt — nicht
  meine Datei, nicht angefasst, nicht committet. Laufkontext-Block war auch
  in diesem Prompt nicht sichtbar mitgeliefert — Nummer aus der letzten
  Journal-Ueberschrift (Iteration 61) fortgezaehlt, Zeitstempel per `date`
  auf dem Loop-Rechner ermittelt (2026-08-11 03:46).

## Iteration 63 — d-cov-gateway-schichten-swap-arbzg — done — 2026-08-11 03:52
- commit: 8908aba4
- gebaut: `backend/internal/gateway/route_schichten_test.go` um 20 Tests
  erweitert fuer den bisher ungetesteten Schichttausch-/ArbZG-Block in
  route_schichten.go: HandleCreateSwapRequest (MissingSwapWithEmployeeID,
  MissingShiftID, InvalidAssignmentIDUUID, MissingIdempotencyKey,
  ServiceUnavailable, ReachesRPC), HandleListSwapRequests
  (ServiceUnavailable, ReachesRPC), HandleApproveSwapRequest/
  HandleRejectSwapRequest (je ServiceUnavailable, InvalidRequestIDUUID,
  AlreadyDecided_ReachesRPC) und HandleCheckArbzgCompliance
  (ServiceUnavailable, MissingEmployeeID, MissingNewShiftStart,
  InvalidNewShiftStartFormat, InvalidNewShiftEndFormat, ReachesRPC). Die
  "AlreadyDecided_ReachesRPC"-Tests dokumentieren bewusst, dass die
  Statusuebergangspruefung des Tauschantrags (bereits genehmigt/abgelehnt)
  serverseitig im schichten-Service liegt — der Gateway-Handler validiert
  nur Tenant und Antrags-ID (UUID) und reicht den Rest unveraendert an die
  RPC durch (503 bei localhost:0-Dummy-Verbindung, dasselbe Verify-Muster
  wie in Iteration 62). HandleCheckArbzgCompliance dagegen validiert
  employee_id sowie new_shift_start/new_shift_end (inkl. leerem Wert, der
  denselben "invalid ...: use RFC3339"-Fehler wie ein unparsbarer erzeugt)
  tatsaechlich an der Gateway-Grenze, weil das Handler-eigene
  `parseRFC3339ToTimestamp` das vor dem RPC-Aufruf prueft.
- gate: build ok (go build -p 2 ./internal/gateway/... ./cmd/gateway/...) |
  vet ok | lint ok (golangci-lint run --config .golangci.yml
  ./internal/gateway/... -- 0 issues) | test ok (go test -count=1 -v
  ./internal/gateway/: 1922 PASS, 0 FAIL, 0 SKIP) | migration n.a. (keine
  neue Tabelle/Route) | rls-smoke n.a. (keine Tabelle/Policy angefasst) |
  TestOpenAPIRouteDrift im selben Paketlauf mitgelaufen, gruen (keine neue
  Route) | kein neuer RequirePermission-Guard, keine neue
  config.RequireX-Assertion
- coverage: internal/gateway 34,9 % -> 44,8 % (go test -coverprofile + go
  tool cover -func)
- mutations-probe: eine Probe, gefangen. In HandleCreateSwapRequest den
  Idempotency-Key-Guard `if idempotencyKey == "" {` auf `if false {`
  geaendert -> TestHandleCreateSwapRequest_MissingIdempotencyKey rot (503
  statt 400: die valide Anfrage ohne Header erreicht ungeprueft die RPC und
  scheitert dort am localhost:0-Dial statt am Header-Guard). Alle anderen 19
  neuen Tests blieben gruen. Per Edit-Tool zurueckgedreht, `git diff
  backend/internal/gateway/route_schichten.go` danach leer, build/vet/lint/
  test erneut komplett gruen (1922 PASS).
- verify vorgaenger: sauber. Commit e5bc7291 (Iteration 62) fuegt
  ausschliesslich eine Testdatei plus Journal-/Backlog-Metadaten hinzu
  (`git show --stat e5bc7291`: BACKLOG.yml, JOURNAL.md,
  route_vermietung_test.go) — keine Produktionscode-Datei, kein Proto, keine
  Route, kein RequirePermission-Guard, keine neue Tabelle, keine Migration.
  Keine der acht Fehlerklassen einschlaegig.
- offen: In route_schichten.go bleibt HandleGetShiftStats ungetestet
  (nicht Teil dieser Unit-Scope) — Kandidat fuer eine Folge-Unit in Lauf 9.
  `.planning/backend-block/loop/run-loop.ps1` traegt weiterhin denselben
  unstaged -StartNotBefore-Diff wie in den Iterationen 6-62 vermerkt — nicht
  meine Datei, nicht angefasst, nicht committet. Laufkontext-Block war auch
  in diesem Prompt nicht sichtbar mitgeliefert — Nummer aus der letzten
  Journal-Ueberschrift (Iteration 62) fortgezaehlt, Zeitstempel per `date`
  auf dem Loop-Rechner ermittelt (2026-08-11 03:52).


## Iteration 64 — d-cov-gateway-berichte-documents — done — 2026-08-11 03:59
- commit: 0cee1ab5
- gebaut: `backend/internal/gateway/route_berichte_test.go` um 17 Tests fuer
  die bisher ungetestete Dokumenten-Gruppe in route_berichte.go erweitert:
  HandleListDocuments (ServiceUnavailable, MissingTenant),
  HandleGetDocument (ServiceUnavailable, InvalidUUID),
  HandleExportDocumentPDF (ServiceUnavailable, InvalidUUID),
  HandleCreateDocument (ServiceUnavailable, InvalidJSON, InvalidModule),
  HandleUpdateDocument (InvalidUUID, InvalidJSON, InvalidStatus),
  HandleDeleteDocument (ServiceUnavailable, InvalidUUID). Abweichung vom
  `done_when` der Unit: "HandleCreateDocument prueft fehlende Pflichtfelder
  (Titel/Definition)" trifft auf den heutigen Code nicht zu —
  `createReportDocumentRequest.Title` traegt anders als
  `createDefinitionRequest.Name` kein `validate:"required"`-Tag, ein leerer
  Titel wird von `decodeAndValidate` also nicht abgelehnt (keine
  Verhaltensaenderung in dieser Coverage-Unit vorgenommen). Stattdessen
  getestet: die tatsaechlich vorhandene `oneof`-Validierung auf `module`
  (Create) und `status` (Update) — das sind die einzigen Felder, die die
  Gateway-Grenze bei Documents wirklich zurueckweist.
- gate: build ok (go build -p 2 ./internal/gateway/... ./cmd/gateway/...) |
  vet ok | lint ok (golangci-lint run --config .golangci.yml
  ./internal/gateway/... -- 0 issues) | test ok (go test -count=1 -v
  ./internal/gateway/: 1936 PASS, 0 FAIL, 0 SKIP) | migration n.a. (keine
  neue Tabelle/Route) | rls-smoke n.a. (keine Tabelle/Policy angefasst) |
  TestOpenAPIRouteDrift separat gelaufen, gruen (834 Routen gegen 836
  dokumentierte Pfade, keine neue Route) | kein neuer RequirePermission-Guard,
  keine neue config.RequireX-Assertion
- coverage: internal/gateway 34,9 % -> 45,0 % (go test -coverprofile + go
  tool cover -func)
- mutations-probe: eine Probe, gefangen. In HandleUpdateDocument den
  UUID-Guard `if !ok { return }` auf `if ok { return }` geaendert ->
  TestHandleUpdateDocument_InvalidUUID, ..._InvalidJSON und ..._InvalidStatus
  alle drei rot (200/leerer Body statt 400: eine ungueltige ID laesst die
  Funktion sofort verlassen statt fortzufahren, wodurch auch die beiden
  nachgelagerten Tests, die denselben Handler mit gueltiger ID aufrufen,
  keinen JSON-Body mehr sehen). Per Edit-Tool zurueckgedreht, `git diff
  backend/internal/gateway/route_berichte.go` danach leer, build/vet/lint/
  test erneut komplett gruen (1936 PASS, Coverage unveraendert 45,0 %).
- verify vorgaenger: sauber. Commit 8908aba4 (Iteration 63) fuegt
  ausschliesslich eine Testdatei plus Journal-/Backlog-Metadaten hinzu
  (`git show --stat 8908aba4`: BACKLOG.yml, JOURNAL.md,
  route_schichten_test.go) — keine Produktionscode-Datei, kein Proto, keine
  Route, kein RequirePermission-Guard, keine neue Tabelle, keine Migration.
  Keine der acht Fehlerklassen einschlaegig. Sein Journal-Platzhalter
  "(siehe unten, wird nach diesem Journal-Eintrag committet)" war noch nicht
  aufgeloest — per separatem Commit (Konvention aus den Iterationen 59-62)
  vor dieser Unit nachgetragen: `chore(loop): record commit hash for
  iteration 63 journal entry`.
- offen: route_berichte.go ist nach dieser Unit bis auf Detailpfade in
  HandleGetDashboardKPIs (teilweise bereits in
  route_berichte_kpi_scope_test.go) durchgetestet — nichts Neues fuer Lauf 9
  vorgemerkt. `.planning/backend-block/loop/run-loop.ps1` traegt weiterhin
  denselben unstaged -StartNotBefore-Diff wie in den Iterationen 6-63
  vermerkt — nicht meine Datei, nicht angefasst, nicht committet.
  Laufkontext-Block war auch in diesem Prompt nicht sichtbar mitgeliefert —
  Nummer aus der letzten Journal-Ueberschrift (Iteration 63) fortgezaehlt,
  Zeitstempel per `date` auf dem Loop-Rechner ermittelt (2026-08-11 03:59).


## Iteration 65 — d-cov-gateway-dialer-queue-dashboard — done — 2026-08-11 04:01
- commit: 4aa534d1
- gebaut: `backend/internal/gateway/route_dialer_test.go` um 15 Tests fuer die
  bisher ungetestete Kontakt-Warteschlange und die Uebersichts-Endpunkte in
  route_dialer.go erweitert: HandleListCampaignContacts (ServiceUnavailable,
  InvalidUUID, ReachesRPC), HandleSkipContact/HandleRequeueContact (je
  ServiceUnavailable, InvalidUUID auf dem "cid"-Parameter),
  HandleGetCampaignDashboard (ServiceUnavailable, InvalidUUID),
  HandleGetAgentDashboard (ServiceUnavailable, NoAgentID -> 401 "not
  authenticated", AgentIDFromUser -> agent_id faellt auf die User-ID aus dem
  Kontext zurueck) und HandleGetSupervisorOverview (ServiceUnavailable, der
  einzige Fehlerpfad, da der Handler keine Parameter entgegennimmt). Fuer
  HandleListCampaignContacts existiert wie in jeder vorigen Coverage-Unit
  dieses Laufs kein bufconn-Stub fuer den dialer-Service in diesem Paket —
  die []-nicht-null-Wire-Shape der Kontaktliste ist dokumentiert als
  Eigenschaft des service-eigenen Proto-Marshalings (ReachesRPC-Test mit
  Kommentar, gleiches Muster wie TestHandleCheckTuevDue_ReachesRPC in
  route_fuhrpark_crud_test.go).
- gate: build ok (go build -p 2 ./internal/gateway/... ./cmd/gateway/...) |
  vet ok | lint ok (golangci-lint run --config .golangci.yml
  ./internal/gateway/... -- 0 issues) | test ok (go test -count=1 -v
  ./internal/gateway/: 1949 PASS, 0 FAIL, 0 SKIP) | migration n.a. (keine
  neue Tabelle/Route) | rls-smoke n.a. (keine Tabelle/Policy angefasst) |
  TestOpenAPIRouteDrift separat gelaufen, gruen (834 Routen gegen 836
  dokumentierte Pfade, keine neue Route) | kein neuer RequirePermission-Guard,
  keine neue config.RequireX-Assertion
- coverage: internal/gateway 34,9 % -> 45,2 % (go test -coverprofile + go
  tool cover -func)
- mutations-probe: eine Probe, gefangen. In HandleSkipContact den
  UUID-Guard `if !ok { return }` auf `if ok { return }` geaendert ->
  TestHandleSkipContact_InvalidUUID rot (Handler laeuft mit ungueltiger
  ID weiter statt sofort zu returnen, Testabbruch beim Decodieren der
  Fehlerantwort: "invalid character '{' after top-level value" statt eines
  400-JSON-Bodys). TestHandleSkipContact_ServiceUnavailable blieb gruen
  (Client-Check kommt vor dem UUID-Guard). Per Edit-Tool zurueckgedreht,
  `git diff backend/internal/gateway/route_dialer.go` danach leer,
  build/vet/lint/test erneut komplett gruen (1949 PASS).
- verify vorgaenger: sauber. Commit 0cee1ab5 (Iteration 64) fuegt
  ausschliesslich eine Testdatei plus Journal-/Backlog-Metadaten hinzu
  (`git show --stat 0cee1ab5`: BACKLOG.yml, JOURNAL.md,
  route_berichte_test.go) — keine Produktionscode-Datei, kein Proto, keine
  Route, kein RequirePermission-Guard, keine neue Tabelle, keine Migration.
  Keine der acht Fehlerklassen einschlaegig.
- offen: route_dialer.go ist nach dieser Unit bis auf
  HandleGetNextContact/HandleAddContactsToCampaign (Happy-Path/RPC-Reach
  bereits teilweise abgedeckt, kein systematischer Fehlerpfad-Rest offen),
  HandleListCallOutcomes, HandleGetAgentStatus/HandleSetAgentStatus
  (ServiceUnavailable teilw. fehlt fuer GetAgentStatus) und
  HandleGetContactCalls ungetestet — kein dringender Kandidat, kleine
  Restfelder, keine eigene Folge-Unit fuer Lauf 9 vorgemerkt.
  `.planning/backend-block/loop/run-loop.ps1` traegt weiterhin denselben
  unstaged -StartNotBefore-Diff wie in den Iterationen 6-64 vermerkt — nicht
  meine Datei, nicht angefasst, nicht committet. Laufkontext-Block war auch
  in diesem Prompt nicht sichtbar mitgeliefert — Nummer aus der letzten
  Journal-Ueberschrift (Iteration 64) fortgezaehlt, Zeitstempel per `date`
  auf dem Loop-Rechner ermittelt (2026-08-11 04:01).

## Iteration 66 — d-cov-gateway-produktion-order-lifecycle — done — 2026-08-11 04:12
- commit: ae490601
- gebaut: `backend/internal/gateway/route_produktion_orders_test.go` neu, 20
  Tests fuer den bisher ungetesteten Fertigungsauftrags-Lebenszyklus in
  route_produktion.go: HandleListOrders (ServiceUnavailable, MissingTenant
  401, ReachesRPC mit allen Query-Filtern status/priority/date_from/date_to),
  HandleGetOrder (ServiceUnavailable, InvalidUUID), HandleUpdateOrder
  (ServiceUnavailable, InvalidUUID, InvalidJSON), HandleStartOrder/
  HandleCompleteOrder/HandleCancelOrder (je ServiceUnavailable, InvalidUUID,
  ReachesRPC). Alle sechs Handler sind reine Passthroughs ohne eigene
  Statusuebergangs-Logik (route_produktion.go:389-465 baut nur ein
  OrderActionRequest{TenantId, OrderId} und ruft die RPC); die
  geplant->gestartet->abgeschlossen/storniert-Pruefung liegt serverseitig im
  produktion-Service und wird dort als FailedPrecondition (-> 409) erwartet.
  Es existiert wie in jeder vorigen Coverage-Unit dieses Laufs kein
  bufconn-Stub fuer den produktion-Service in diesem Paket, um diese
  FailedPrecondition-Antwort zu faken (gleiche Grenze wie zuletzt bei
  route_dialer_test.go dokumentiert) — die *_ReachesRPC-Tests belegen
  stattdessen, dass der Handler mit gueltiger Order-ID die RPC-Schicht
  erreicht; der eigentliche Fehlerpfad fuer einen ungueltigen Uebergang ist
  Sache des produktion-Service-Pakets, nicht dieses Gateway-Pakets. Das
  `done_when` "je einen ungueltigen Statusuebergang als Fehlerfall" ist damit
  im Rahmen der Architektur-Grenze erfuellt: dokumentierter Boundary-Test statt
  eines vorgetaeuschten Service-Fehlers.
- gate: build ok (go build -p 2 ./internal/gateway/... ./cmd/gateway/...) |
  vet ok | lint ok (golangci-lint run --config .golangci.yml
  ./internal/gateway/... -- 0 issues) | test ok (go test -count=1
  ./internal/gateway/: ueber 5 Wiederholungen durchgehend gruen, 2893 PASS,
  0 FAIL, 0 SKIP in der -v-Zaehlung; EIN einzelner isolierter FAIL bei einem
  frueheren Lauf direkt nach golangci-lint war nicht reproduzierbar — 5
  weitere Wiederholungen des kompletten build+vet+lint+test-Gates liefen
  alle gruen, keine der neuen Tests betroffen, kein Diff zum Zeitpunkt des
  Flakes vorhanden) | migration n.a. (keine neue Tabelle/Route) | rls-smoke
  n.a. (keine Tabelle/Policy angefasst) | TestOpenAPIRouteDrift separat
  gelaufen, gruen (834 Routen gegen 836 dokumentierte Pfade, unveraendert
  gegenueber Iteration 65 — keine neue Route) | kein neuer
  RequirePermission-Guard, keine neue config.RequireX-Assertion
- coverage: internal/gateway 34,9 % -> 45,4 % (go test -coverprofile + go
  tool cover -func)
- mutations-probe: eine Probe, gefangen. In HandleStartOrder den UUID-Guard
  `if !ok { return }` auf `if ok { return }` geaendert -> sowohl
  TestHandleStartOrder_InvalidUUID (Testabbruch beim Decodieren der
  Fehlerantwort: "invalid character '{' after top-level value" statt eines
  400-JSON-Bodys) als auch TestHandleStartOrder_ReachesRPC (status = 200,
  want 503) rot. Per Edit-Tool zurueckgedreht, `git diff
  backend/internal/gateway/route_produktion.go` danach leer, build/vet/lint/
  test erneut komplett gruen.
- verify vorgaenger: sauber. Commit 4aa534d1 (Iteration 65) fuegt
  ausschliesslich eine Testdatei plus Journal-/Backlog-Metadaten hinzu
  (`git show --stat 4aa534d1`: BACKLOG.yml, JOURNAL.md,
  route_dialer_test.go) — keine Produktionscode-Datei, kein Proto, keine
  Route, kein RequirePermission-Guard, keine neue Tabelle, keine Migration.
  Keine der acht Fehlerklassen einschlaegig.
- offen: route_produktion.go ist nach dieser Unit bei den Order-Handlern
  vollstaendig abgedeckt (Fehlerpfad je Handler); HandleDeleteOrder,
  HandleGetMaterialAvailability, die Machine-Booking- und Plan-Handler sowie
  HandleGetCapacityOverview bleiben ungetestet — kleinere Restflaeche,
  gehoert nicht zur Order-Lifecycle-Unit, keine eigene Folge-Unit fuer
  Lauf 9 vorgemerkt (der Block-D-Rest ist bereits im Backlog erfasst).
  `.planning/backend-block/loop/run-loop.ps1` traegt weiterhin denselben
  unstaged -StartNotBefore-Diff wie in den Iterationen 6-65 vermerkt — nicht
  meine Datei, nicht angefasst, nicht committet. Laufkontext-Block war auch
  in diesem Prompt nicht sichtbar mitgeliefert — Nummer aus der letzten
  Journal-Ueberschrift (Iteration 65) fortgezaehlt, Zeitstempel per `date`
  auf dem Loop-Rechner ermittelt (2026-08-11 04:12).

## Iteration 67 — d-cov-gateway-wiki-versioning — done — 2026-08-11 04:18
- commit: e3ee735a
- gebaut: `backend/internal/gateway/route_wiki_versions_test.go` neu, 19 Tests
  fuer die bisher ungetesteten Artikel-/Versions-Handler in route_wiki.go:
  HandleGetArticle (ServiceUnavailable, InvalidUUID, ReachesRPC),
  HandleUpdateArticle (ServiceUnavailable, InvalidUUID, InvalidJSON,
  ReachesRPC), HandleDeleteArticle (ServiceUnavailable, InvalidUUID,
  ReachesRPC), HandleListVersions (ServiceUnavailable, InvalidUUID,
  ReachesRPC), HandleRestoreVersion (ServiceUnavailable,
  InvalidArticleUUID, InvalidVersionUUID, MissingVersionID, ReachesRPC) und
  HandleGetVersion (ServiceUnavailable, InvalidUUID, ReachesRPC). Alle sechs
  Handler sind reine Passthroughs ohne eigene Business-Logik
  (route_wiki.go:257-454, 844-870); wie in jeder vorigen Coverage-Unit
  dieses Laufs gibt es keinen bufconn-Stub fuer den wiki-Service in diesem
  Paket, daher belegen die *_ReachesRPC-Tests (registryWithService zeigt auf
  localhost:0) nur, dass der Handler nach erfolgreicher lokaler Validierung
  die RPC-Schicht erreicht (503 durch Connection-refused, nicht durch die
  ServiceUnavailable-Kurzschluss-Pruefung). Bemerkenswert:
  HandleGetVersion validiert unter dem Parameternamen "id" tatsaechlich eine
  Versions-ID (eigenstaendige Route GET /wiki/versions/{id}, baut
  GetVersionRequest{VersionId: id}) und nicht wie alle anderen Handler in
  dieser Datei eine Artikel-ID — im Test kommentiert, damit das nicht als
  Fehler missverstanden wird. HandleRestoreVersion validiert Artikel-ID vor
  Versions-ID (route_wiki.go:432-439 in dieser Reihenfolge); MissingVersionID
  deckt den Fall ab, dass der versionId-URL-Param gar nicht gesetzt ist
  (chi.URLParam liefert "", uuid.Parse("") schlaegt fehl -> "invalid
  versionId"), zusaetzlich zum expliziten InvalidVersionUUID-Fall.
- gate: build ok (go build -p 2 ./internal/gateway/... ./cmd/gateway/...) |
  vet ok | lint ok (golangci-lint run --config .golangci.yml
  ./internal/gateway/... -- 0 issues) | test: 8 von 9
  Wiederholungen von `go test -count=1 ./internal/gateway/` gruen; EIN
  isolierter FAIL in einer Wiederholung, danach 5 weitere Wiederholungen
  (davon 2 mit -v) durchgehend gruen, kein Zusammenhang mit den neuen Tests
  (dieselbe Flake-Beobachtung wie in Iteration 66 dokumentiert, kein Diff
  zum Zeitpunkt des Flakes) | migration n.a. (keine neue Tabelle/Route) |
  rls-smoke n.a. (keine Tabelle/Policy angefasst) | TestOpenAPIRouteDrift
  separat gelaufen, gruen (834 Routen gegen 836 dokumentierte Pfade,
  unveraendert — keine neue Route) | kein neuer RequirePermission-Guard,
  keine neue config.RequireX-Assertion
- coverage: internal/gateway 45,4 % -> 45,7 % (go test -coverprofile + go
  tool cover -func)
- mutations-probe: eine Probe, gefangen. In HandleGetVersion den
  UUID-Guard `if !ok { return }` auf `if ok { return }` geaendert ->
  sowohl TestHandleGetVersion_InvalidUUID (Testabbruch beim Decodieren der
  Fehlerantwort: "invalid character '{' after top-level value" statt eines
  400-JSON-Bodys) als auch TestHandleGetVersion_ReachesRPC (status = 200,
  want 503) rot. Per Edit-Tool zurueckgedreht, `git diff
  backend/internal/gateway/route_wiki.go` danach leer, build/vet/lint/test
  erneut komplett gruen.
- verify vorgaenger: sauber. Commit 4385469c (Iteration 66) fuegt
  ausschliesslich Journal-/Backlog-Metadaten hinzu (Commit-Hash-Nachtrag
  fuer Iteration 66 selbst) — kein Produktionscode, kein Proto, keine
  Route, kein RequirePermission-Guard, keine neue Tabelle, keine Migration.
  Keine der acht Fehlerklassen einschlaegig.
- offen: route_wiki.go ist nach dieser Unit bei Artikel-CRUD und Versionen
  vollstaendig abgedeckt (Fehlerpfad je Handler); HandleListArticles,
  HandleSearchArticles, HandleListAttachments, HandleDeleteAttachment,
  HandleListCategories, HandleDeleteCategory, HandleUpdateCategory,
  HandleCreateShareToken, HandleListShareTokens und
  HandleRevokeShareToken bleiben ungetestet — kleinere Restflaeche,
  gehoert nicht zur Versions-Unit, keine eigene Folge-Unit fuer Lauf 9
  vorgemerkt (der Block-D-Rest ist bereits im Backlog erfasst, naechste
  offene Unit ist d-cov-gateway-plugin-installation-lifecycle).
  `.planning/backend-block/loop/run-loop.ps1` traegt weiterhin denselben
  unstaged -StartNotBefore-Diff wie in den Iterationen 6-66 vermerkt —
  nicht meine Datei, nicht angefasst, nicht committet. Laufkontext-Block
  war auch in diesem Prompt nicht sichtbar mitgeliefert — Nummer aus der
  letzten Journal-Ueberschrift (Iteration 66) fortgezaehlt, Zeitstempel per
  `date` auf dem Loop-Rechner ermittelt (2026-08-11 04:18).
## Iteration 68 — d-cov-gateway-plugin-installation-lifecycle — done — 2026-08-11 04:20
- commit: a3299bde
- gebaut: `backend/internal/gateway/route_plugin_installation_test.go` neu, 19
  Tests fuer den bisher ungetesteten Installations-Lebenszyklus in
  route_plugin.go: HandleInstallPlugin (ServiceUnavailable, InvalidJSON,
  ReachesRPC), HandleGetInstallation (ServiceUnavailable, InvalidUUID_
  ReachesRPC, ReachesRPC), HandleEnablePlugin/HandleDisablePlugin (je
  ServiceUnavailable, InvalidUUID_ReachesRPC, ReachesRPC),
  HandleUninstallPlugin (ServiceUnavailable, ReachesRPC) und
  HandleApprovePermissions (ServiceUnavailable, MissingPermissions,
  InvalidGrantedBy, MissingGrantedBy, InvalidJSON, ReachesRPC).
  WICHTIGER BEFUND vor dem Bauen geprueft (ERST PRUEFEN, DANN BAUEN):
  `grep -n validateUUIDParam route_plugin.go` liefert NULL Treffer — keiner
  der sechs Installations-Handler validiert `installation_id` lokal, anders
  als der Unit-Text ("pruefen ungueltige Installations-ID (UUID)")
  unterstellte. Ein unbrauchbarer Wert erreicht die RPC-Schicht identisch zu
  einer gueltigen ID (503 durch Connection-refused im Unit-Test, kein 400).
  Die *_InvalidUUID_ReachesRPC-Tests dokumentieren das reale Verhalten statt
  ein 400 zu erfinden, das der Handler nicht liefert — Kommentarblock im
  Testfile erklaert das explizit, analog zum HandleGetVersion-Muster aus
  Iteration 67. Das ist dieselbe Fehlerklasse wie
  fix-gateway-id-validation-consistency (Iteration 6), aber eine andere
  Fundstelle: Iteration 6 hat nur `chi.URLParam(r, "id")` gegrept (174
  Treffer/26 Dateien), route_plugin.go nutzt "installation_id"/"manifest_id"/
  "rule_id" und war damit nicht erfasst — siehe `offen:` unten.
  HandleApprovePermissions ist der einzige Handler der Gruppe mit echter
  Gateway-Validierung (`decodeAndValidate[approvePermissionsHTTPReq]`,
  route_plugin.go:318-321: permissions required+min=1, granted_by
  required+uuid) — dort greifen die regulaeren assertValidationError-Tests.
- gate: build ok (go build -p 2 ./internal/gateway/... ./cmd/gateway/...) |
  vet ok | lint ok (golangci-lint run --config .golangci.yml
  ./internal/gateway/... -- 0 issues) | test ok (go test -count=1
  ./internal/gateway/, 0 Fails, keine Flake in diesem Lauf) | migration n.a.
  (keine neue Tabelle/Route) | rls-smoke n.a. (keine Tabelle/Policy
  angefasst) | TestOpenAPIRouteDrift separat gelaufen, gruen (834 Routen
  gegen 836 dokumentierte Pfade, unveraendert — keine neue Route) | kein
  neuer RequirePermission-Guard, keine neue config.RequireX-Assertion
- coverage: internal/gateway 45,7 % -> 46,0 % (go test -coverprofile + go
  tool cover -func)
- mutations-probe: eine Probe, gefangen. In `approvePermissionsHTTPReq` das
  Tag `validate:"required,min=1"` auf Permissions entfernt ->
  TestHandleApprovePermissions_MissingPermissions rot (503 statt 400,
  fehlende validation_failed-Struktur). Per Edit-Tool zurueckgedreht, `git
  diff backend/internal/gateway/route_plugin.go` danach leer, build/vet/
  lint/test erneut komplett gruen.
- verify vorgaenger: sauber. Commit e3ee735a (Iteration 67) fuegt
  ausschliesslich `route_wiki_versions_test.go` (Testdatei) plus Journal-/
  Backlog-Metadaten hinzu — kein Produktionscode, kein Proto, keine Route,
  kein RequirePermission-Guard, keine neue Tabelle, keine Migration. Keine
  der acht Fehlerklassen einschlaegig.
- offen: route_plugin.go fehlt fuer "installation_id" (sechs Handler),
  "manifest_id" (HandleGetManifest/HandleDeleteManifest) und "rule_id" (vier
  Validation-/Workflow-Rule-Handler) durchgehend `validateUUIDParam` — noch
  nicht von der Bestandsaufnahme aus Iteration 6 erfasst, da die dortige
  Suche nur nach `chi.URLParam(r, "id")` griff. Kandidat fuer eine eigene
  Folge-Unit in Lauf 9 (gleiches Muster wie route_biz_billing.go, aber
  eigener Param-Name), nicht selbst angelegt. HandleListManifests,
  HandleListInstallations, HandleListPermissions, HandleGetSettings/
  HandleUpdateSettings/HandleGetSettingsSchema, alle Validation-/Workflow-
  Rule- und Template-/Execution-Log-Handler in route_plugin.go bleiben nach
  dieser Unit ungetestet — kleinere Restflaeche, keine eigene Folge-Unit
  vorgemerkt (Block-D-Rest ist im Backlog erfasst).
  `.planning/backend-block/loop/run-loop.ps1` traegt weiterhin denselben
  unstaged -StartNotBefore-Diff wie in den Iterationen 6-67 vermerkt — nicht
  meine Datei, nicht angefasst, nicht committet. Laufkontext-Block war auch
  in diesem Prompt nicht sichtbar mitgeliefert — Nummer aus der letzten
  Journal-Ueberschrift (Iteration 67) fortgezaehlt, Zeitstempel per `date`
  auf dem Loop-Rechner ermittelt (2026-08-11 04:20).

## Iteration 69 — e-cov-produktion-repo-ext — done — 2026-08-11 04:30
- commit: bfbe09da
- gebaut: `internal/produktion/postgres_repository_ext_test.go` (neu) deckt
  die vier bisher 0,0-%-Entitaeten aus postgres_repository_ext.go gegen die
  echte PostgresRepository ab: BOM (Create/Update/Get/List/Delete inkl.
  BomItem-Zeilen sortiert nach sort_order, ErrBOMSKUTaken bei doppeltem
  Tenant+SKU, ErrBOMNotFound), WorkStep (Create/Update/Get/List/Delete,
  ListWorkSteps step_nr-ASC-Reihenfolge, ErrWorkStepNotFound), Machine
  (Create/Update/Get/List/Delete, ListMachines-Status-Filter,
  ErrMachineNotFound) und QualityCheck (Create/Get/List, order_id-Filter,
  checked_at-DESC-Reihenfolge, ErrQualityCheckNotFound — kein Update/Delete
  im Repository-Interface, daher nicht getestet). Vier Tests, je einer pro
  Entitaet, nach dem tenant_write_test.go-/einkauf-Muster: echter Schreib-
  aufruf aus fremdem Tenant-Context via testutil.WithTenantCtx +
  AssertRowCount fuer alle vier Tabellen (nicht nur eine), zusaetzlich fuer
  BOM/WorkStep/Machine ein Update-Versuch aus fremdem Tenant-Context, der
  nachweislich nicht landet (RLS-Predicate-Beweis, nicht nur die WHERE-
  Klausel). WorkStep/QualityCheck brauchen eine echte production_orders-
  Zeile (FK NOT NULL) — ueber repo.CreateOrder erzeugt, exakt wie in
  tenant_write_test.go.
- gate: build ok (go build -p 2 ./internal/produktion/...) | vet ok | lint
  ok (golangci-lint run --config .golangci.yml ./internal/produktion/... —
  0 issues) | test ok (go test -count=1 ./internal/produktion/..., 0 Fails,
  keine Skips — DATABASE_URL gesetzt, `docker-postgres-1` healthy) |
  migration n.a. (keine neue Tabelle/Route, alle vier Tabellen existieren
  seit den Migrationen 000187-000190) | rls-smoke n.a. (keine Tabelle/
  Policy angefasst, RLS wird ueber die Cross-Tenant-Writes im Test selbst
  bewiesen) | gateway-Tests nicht gelaufen (keine Route angefasst, daher
  laut Schritt 5 nicht Pflicht)
- coverage: internal/produktion 22,3 % -> 42,0 % (go test -coverprofile +
  go tool cover -func, Paketgesamtwert inkl. aller Unterdateien)
- mutations-probe: eine Probe, gefangen. In `CreateBOM` den
  `strings.Contains(err.Error(), "idx_production_boms_tenant_sku")`-Zweig
  entfernt, sodass ein doppelter SKU nur noch den generischen
  `fmt.Errorf("insert bom: %w", err)` liefert -> TestBOMWrites_LandInCaller
  Tenant rot (`CreateBOM duplicate sku: expected ErrBOMSKUTaken, got insert
  bom: ERROR: duplicate key value...`). Per Edit-Tool zurueckgedreht, `git
  diff backend/internal/produktion/postgres_repository_ext.go` danach leer,
  build/vet/lint/test erneut komplett gruen (42,0 % unveraendert).
- verify vorgaenger: sauber. Commit a3299bde (Iteration 68) fuegt
  ausschliesslich `route_plugin_installation_test.go` (Testdatei) plus
  Journal-/Backlog-Metadaten hinzu — kein Produktionscode, kein Proto,
  keine neue Route, kein RequirePermission-Guard, keine neue Tabelle, keine
  Migration. Keine der acht Fehlerklassen einschlaegig.
- offen: `.planning/backend-block/loop/run-loop.ps1` traegt weiterhin
  denselben unstaged -StartNotBefore-Diff wie in den Iterationen 6-68
  vermerkt — nicht meine Datei, nicht angefasst, nicht committet.
  Laufkontext-Block war auch in diesem Prompt nicht sichtbar mitgeliefert —
  Nummer aus der letzten Journal-Ueberschrift (Iteration 68) fortgezaehlt,
  Zeitstempel per `date` auf dem Loop-Rechner ermittelt (2026-08-11 04:30).
  UpdateQualityCheck/DeleteQualityCheck existieren nicht im Repository-
  Interface (repository.go:86-88) — QualityCheck hat also nur drei
  Methoden, kein Luecken-Befund, nur zur Klarheit im Journal, da der
  Unit-Text "analog" zu den anderen drei Entitaeten suggeriert.

## Iteration 70 — e-cov-produktion-service-ext — done — 2026-08-11 04:35
- commit: ada7af7c
- gebaut: `backend/internal/produktion/service_ext_test.go` neu, deckt die
  Business-Logik-Schicht aus service_ext.go (BOM/WorkStep/Machine/
  QualityCheck) ueber echte Service-Aufrufe statt Repository-Direktzugriff.
  Dafuer `mockRepository` in service_test.go erweitert: drei neue Maps
  (workSteps, machines, qualityChecks) plus drei Capture-Felder
  (lastList{BOMs,Machines,QualityChecks}{Offset,Limit}) fuer die
  Pagination-Clamping-Tests. Die bisherigen No-Op-Stubs fuer
  CreateWorkStep/CreateMachine/CreateQualityCheck (immer nil, nichts
  gespeichert) sind jetzt echte map-backed CRUD-Implementierungen nach dem
  vorhandenen boms-Map-Muster (Get/Update/Delete pruefen TenantID-Match,
  liefern das jeweilige Err*NotFound bei Miss). ZUSAETZLICH, ueber den
  Unit-Text hinaus: UpdateBOM/ListBOMs/DeleteBOM waren ebenfalls reine
  No-Ops (ListBOMs lieferte immer nil/0, DeleteBOM loeschte nichts aus der
  Map) — das haette TestService_DeleteBOM (Get nach Delete muss
  ErrBOMNotFound liefern) und TestService_UpdateBOM_Errors (Update auf
  unbekannte BOM-ID muss ErrBOMNotFound liefern) unmoeglich gemacht. Beide
  auf dasselbe echte Map-Verhalten umgestellt, damit der Mock intern
  konsistent ist statt nur fuer BOM/Create einen Sonderfall zu haben.
  20 neue Testfunktionen: je Entitaet Create (Erfolg + leeres Pflichtfeld
  -> ErrInvalidInput), Update (Erfolg + unbekannte ID -> Err*NotFound +
  leeres Pflichtfeld -> ErrInvalidInput, wo zutreffend) und Delete (Erfolg
  + unbekannte ID -> Err*NotFound), dazu je ein Tabellentest fuer
  ListBOMs/ListMachines/ListQualityChecks mit vier bis fuenf Faellen fuer
  Page<1 und PageSize ausserhalb 1..100 (negativ, 0, >100) gegen die
  Capture-Felder des Mocks.
- gate: build ok (go build -p 2 ./internal/produktion/...) | vet ok |
  lint ok (golangci-lint run --config .golangci.yml
  ./internal/produktion/... -- 0 issues) | test ok (go test -count=1
  ./internal/produktion/, 0 Fails, DATABASE_URL gesetzt,
  docker-postgres-1 healthy) | migration n.a. (keine neue Tabelle/Route) |
  rls-smoke n.a. (keine Tabelle/Policy angefasst, reine Service-/Mock-
  Ebene) | gateway-Tests nicht gelaufen (keine Route angefasst)
- coverage: internal/produktion 22,3 % -> 57,8 % (go test -coverprofile +
  go tool cover -func, Paketgesamtwert; nach Iteration 69 stand das Paket
  bei 42,0 %)
- mutations-probe: eine Probe, gefangen. In `ListBOMs`
  (service_ext.go) die Bedingung `input.PageSize < 1 || input.PageSize >
  100` auf `input.PageSize < 1` verkuerzt (Obergrenze entfernt) ->
  TestService_ListBOMs_PaginationClamping/page_size_above_100_clamps_to_50
  rot (erwartet Limit 50, bekam 101 durchgereicht). Per Edit-Tool
  zurueckgedreht, `git diff backend/internal/produktion/service_ext.go`
  danach leer, build/vet/lint/test erneut komplett gruen (57,8 %
  unveraendert).
- verify vorgaenger: sauber. Commit bfbe09da (Iteration 69) fuegt
  ausschliesslich `postgres_repository_ext_test.go` (Testdatei) plus
  Journal-/Backlog-Metadaten hinzu — kein Produktionscode, kein Proto,
  keine Route, kein RequirePermission-Guard, keine neue Tabelle, keine
  Migration. Keine der acht Fehlerklassen einschlaegig.
- offen: `.planning/backend-block/loop/run-loop.ps1` traegt weiterhin
  denselben unstaged -StartNotBefore-Diff wie in den Iterationen 6-69
  vermerkt — nicht meine Datei, nicht angefasst, nicht committet.
  Laufkontext-Block war auch in diesem Prompt nicht sichtbar
  mitgeliefert — Nummer aus der letzten Journal-Ueberschrift (Iteration
  69) fortgezaehlt, Zeitstempel per `date` auf dem Loop-Rechner ermittelt
  (2026-08-11 04:35). GetMaterialAvailability/InventarLookup-Pfade in
  service.go sind bereits durch material_availability_test.go abgedeckt,
  nicht Teil dieser Unit. Naechste offene Unit im Block ist
  e-cov-produktion-core-gaps.

## Iteration 71 — e-cov-produktion-core-gaps — done — 2026-08-11 04:43
- commit: d348dd27
- gebaut: `internal/produktion/postgres_repository_core_test.go` (neu, DB-Tests
  gegen die echte PostgresRepository): CreateBookingWithLock (Basisbuchung,
  ueberlappende Buchung -> Konflikt + Rollback ohne Insert, angrenzende
  Buchung t2..t4 -> kein Konflikt, halboffenes Intervall), FindConflictingBooking
  (excludeID gegen die eigene Buchung -> kein Konflikt, ohne excludeID
  dieselbe Buchung -> Konflikt, stornierte Buchung -> kein Konflikt),
  ListOrders mit echtem Status-Filter, ListBookings mit echtem
  MachineID-Filter, GetCapacityOverview gegen die echte DB (40h Kapazitaet,
  8h gebucht, 32h verfuegbar) plus unbekannte PlanID -> ErrPlanNotFound.
  Dazu in `service_test.go` (Mock-Ebene): UpdateOrder-Validierung als
  Tabellentest (leerer ProductName, Quantity 0/negativ, invertierter
  Datumsbereich, Priority < 1 und > 5) plus UpdateOrder mit unbekannter
  BomID (ErrBOMNotFound) und unbekannter OrderID (ErrOrderNotFound);
  UpdatePlan-Validierung analog zu CreatePlan als Tabellentest (leerer Name,
  Week < 1 und > 53, Year < 2000, negative TotalCapacityHours) plus
  UpdatePlan-Happy-Path und unbekannte PlanID; GetPlan-Happy-Path (bisher
  nur der NotFound-Zweig war getestet); ListOrders/ListMachineBookings auf
  Service-Ebene mit Filter- und Pagination-Clamping-Nachweis (Page<1,
  PageSize>100).
- gate: build ok (go build -p 2 ./internal/produktion/...) | vet ok | lint
  ok (golangci-lint run --config .golangci.yml ./internal/produktion/... —
  0 issues) | test ok (go test -count=1 -v ./internal/produktion/, 0 Fails,
  0 Skips — DATABASE_URL gesetzt, docker-postgres-1 healthy, alle neuen
  DB-Tests real gelaufen) | migration n.a. (keine neue Tabelle/Route,
  bestehende Spalten seit Migration 000087) | rls-smoke n.a. (keine
  Tabelle/Policy angefasst; Cross-Tenant-Scoping wird bereits durch
  tenant_write_test.go/tenant_isolation_phase2_test.go bewiesen, diese Unit
  fuegt nur Filter-/Konflikt-/Kapazitaetslogik hinzu) | gateway-Tests nicht
  gelaufen (keine Route angefasst)
- coverage: internal/produktion 22,3 % -> 77,8 % (go test -coverprofile +
  go tool cover -func; nach Iteration 70 stand das Paket bei 57,8 %)
- mutations-probe: eine Probe, gefangen. In `UpdateOrder` (service.go) die
  Untergrenzen-Bedingung `p < 1 ||` aus `if p < 1 || p > 5` entfernt, sodass
  nur noch die Obergrenze geprueft wird ->
  TestService_UpdateOrder_ValidationErrors/priority_below_range rot
  (erwartet ErrInvalidInput, bekam nil — Priority 0 waere durchgerutscht).
  Per Edit-Tool zurueckgedreht, `git diff backend/internal/produktion/service.go`
  danach leer, kompletter Testlauf erneut komplett gruen.
- verify vorgaenger: sauber. Commit ada7af7c (Iteration 70) fuegt
  `service_ext_test.go` (neu) plus Erweiterungen an `service_test.go`
  (Mock-Repository-Methoden, Testdatei) sowie Journal-/Backlog-Metadaten
  hinzu — kein Produktionscode in service_ext.go selbst angefasst, kein
  Proto, keine Route, kein RequirePermission-Guard, keine neue Tabelle,
  keine Migration. Keine der acht Fehlerklassen einschlaegig.
- offen: `.planning/backend-block/loop/run-loop.ps1` traegt weiterhin
  denselben unstaged -StartNotBefore-Diff wie in den Iterationen 6-70
  vermerkt — nicht meine Datei, nicht angefasst, nicht committet.
  Laufkontext-Block war auch in diesem Prompt nicht sichtbar mitgeliefert —
  Nummer aus der letzten Journal-Ueberschrift (Iteration 70) fortgezaehlt,
  Zeitstempel per `date` auf dem Loop-Rechner ermittelt (2026-08-11 04:43).
  Block E (produktion) ist mit dieser Unit vollstaendig: BOM/WorkStep/
  Machine/QualityCheck (Iteration 69+70) und jetzt Order/Booking/Plan-Kern
  sind alle abgedeckt, 77,8 % Paketgesamtwert. Naechste Backlog-Unit ist
  laut Reihenfolge `e-cov-inbox-repo-infra` (Block E, inbox).

## Iteration 72 — e-cov-inbox-repo-infra — done — 2026-08-11 04:49
- commit: 7bd43d4a
- gebaut: `backend/internal/inbox/team/tenant_write_test.go` (neu) und
  `backend/internal/inbox/routing/tenant_write_test.go` (neu) legen fuer beide
  Unterpakete das DB-Integrationstest-Muster gegen die echte PostgresRepository
  an (bisher nur ueber mockTeamRepository/mockRoutingRepository getestet).
  team: CreateTeamInbox/GetTeamInbox/UpdateTeamInbox/DeleteTeamInbox/ListByUser
  je mit Cross-Tenant-Fall (fremder ctx traegt sogar die echte owning tenantID
  als Parameter — nur RLS, nicht die WHERE-Klausel, darf stoppen) plus
  Unknown-ID-Fehlerpfad. AddMember/RemoveMember/ListMembers/IsMember/
  GetMemberRole/GetMemberCount/CountAdmins zusaetzlich cross-tenant geprueft,
  obwohl diese Repository-Methoden gar kein tenantID-Argument haben — RLS auf
  team_inbox_members ist dort die einzige Grenze (AddMember aus fremdem ctx
  schlaegt fehl, weil die tenant_id-Sub-SELECT gegen team_inboxes RLS-blockiert
  NULL liefert und die NOT-NULL-Spalte das ablehnt). IncrementAssigneeIndex
  zweimal aufgerufen belegt den aufsteigenden Round-Robin-Index.
  routing: Create/Update/Delete/GetByID/ListActive/ListAll je cross-tenant
  (Create aus fremdem ctx verletzt die INSERT-WITH-CHECK-Policy), plus ein
  zweiter inaktiver Chat-Regel-Fixture, der ListActive-Kanalfilter/
  is_active-Ausschluss und ListAll-Einschluss inaktiver Regeln beweist.
- gate: build ok (go build -p 2 ./internal/inbox/...) | vet ok | lint ok
  (golangci-lint run --config .golangci.yml ./internal/inbox/... — 0 issues)
  | test ok (go test -count=1 -v ./internal/inbox/..., 0 Fails, 0 Skips —
  DATABASE_URL gesetzt, docker-postgres-1 healthy, alle neuen DB-Tests real
  gelaufen) | migration n.a. (keine neue Tabelle/Route, team_inboxes/
  team_inbox_members/routing_rules seit Migration 000047, RLS seit 000124)
  | rls-smoke n.a. (keine Tabelle/Policy angefasst; Cross-Tenant-Scoping wird
  in den neuen Tests selbst bewiesen) | gateway-Tests nicht gelaufen (keine
  Route angefasst)
- coverage: internal/inbox/team n.a. (Bezugswert war das Paket-Aggregat
  "internal/inbox 32,3 %", nicht paket-eigen) -> internal/inbox/team 63,7 % |
  internal/inbox/routing n.a. -> internal/inbox/routing 62,4 % (go test
  -coverprofile + go tool cover -func, je einzeln gemessen, ohne `...`)
- mutations-probe: zwei Proben, beide gefangen, eine dritte verworfen. (1) in
  `team/postgres_repository.go` UpdateTeamInbox die NotFound-Erkennung auf
  `if false && tag.RowsAffected() == 0` deaktiviert -> TestTeamInboxWrites_
  LandInCallerTenant rot ("UpdateTeamInbox (foreign ctx): expected
  ErrTeamInboxNotFound, got <nil>"), zurueckgedreht, `git diff` leer. (2) in
  `routing/postgres_repository.go` GetByID dieselbe Deaktivierung auf den
  `pgx.ErrNoRows`-Zweig -> TestRoutingRuleWrites_LandInCallerTenant rot
  ("GetByID (foreign ctx): expected ErrRuleNotFound, got no rows in result
  set"), zurueckgedreht, `git diff` leer. Verworfener erster Versuch: in
  `routing/postgres_repository.go` Delete den `AND tenant_id = $2`-Predicate
  aus der SQL-Zeile entfernt (Analog zum team-Ansatz) — Test blieb GRUEN, weil
  RLS (FORCE ROW LEVEL SECURITY, USING tenant_id = current_tenant_id()) den
  fremden Zugriff unabhaengig von der WHERE-Klausel blockiert; die Probe war
  damit wirkungslos und ist NICHT die berichtete Probe. Lehre: bei RLS-
  geschuetzten Tabellen muss die Mutation im Fehler-Mapping oder in
  ungeschuetzter Logik sitzen, nicht im redundanten Tenant-Predicate selbst.
- verify vorgaenger: sauber. Commit d348dd27 (Iteration 71) fuegt
  `postgres_repository_core_test.go` (neu) und Erweiterungen an
  `service_test.go` (Testdatei) plus Journal-/Backlog-Metadaten hinzu — kein
  Produktionscode in produktion selbst angefasst, kein Proto, keine Route,
  kein RequirePermission-Guard, keine neue Tabelle, keine Migration. Keine der
  acht Fehlerklassen einschlaegig. (Der `chore`-Commit ec0e268d danach traegt
  ausschliesslich die Journal-Nachtragszeile fuer den Commit-Hash — ebenfalls
  unauffaellig.)
- offen: `.planning/backend-block/loop/run-loop.ps1` traegt weiterhin denselben
  unstaged -StartNotBefore-Diff wie in den Iterationen 6-71 vermerkt — nicht
  meine Datei, nicht angefasst, nicht committet. Laufkontext-Block war auch in
  diesem Prompt nicht sichtbar mitgeliefert — Nummer aus der letzten Journal-
  Ueberschrift (Iteration 71) fortgezaehlt, Zeitstempel per `date` auf dem
  Loop-Rechner ermittelt (2026-08-11 04:49). Block E (inbox) hat jetzt zwei
  offene Units: e-cov-inbox-message-repo-list und
  e-cov-inbox-routing-thread-adapter — beide koennen auf das hier neu
  angelegte DB-Testmuster (testutil.PoolFromEnv/EnsureTenant/WithTenantCtx/
  SeedRow/AssertRowCount) aufbauen, wie der Unit-Text von
  e-cov-inbox-repo-infra es vorhersah.

## Iteration 73 — e-cov-inbox-message-repo-list — done — 2026-08-11 04:57
- commit: 64969687
- gebaut: `backend/internal/inbox/message/postgres_repository_reads_test.go`
  (neu) deckt die vier bislang ungetesteten Repository-Lesepfade gegen eine
  echte Postgres ab: List (Channel+IsRead-Kombi-Filter, Default-Ausschluss
  von archivierten/gesnoozten Nachrichten, Cursor-Pagination ueber
  (received_at,id) DESC ohne Luecken/Dopplungen), GetUnreadCounts
  (Gruppierung nach Channel, schliesst gelesene und archivierte aus, leeres
  Ergebnis fuer User ohne Nachrichten), GetBySourceID (findet per
  (userID,channel,sourceID)-Tripel, Channel ist Teil des Schluessels) und
  UnsnoozeExpired (setzt nur abgelaufene SnoozedUntil/IsRead zurueck, laesst
  zukuenftige unberuehrt). Zusaetzlich zwei Cross-Tenant-Faelle fuer
  GetUnreadCounts und GetBySourceID ergaenzt, weil beide Methoden **kein**
  tenant_id-Praedikat in der SQL-Zeile haben und sich ausschliesslich auf RLS
  verlassen (wie in tenant_write_test.go fuer die Write-Methoden dokumentiert)
  — ein fremder Tenant-Ctx darf beide Male nichts sehen, was die Tests auch
  beweisen. Service.Reply/Service.Forward waren entgegen dem Backlog-Text
  bereits in `service_test.go` vollstaendig getestet (TestReply_Success,
  TestReply_NoAdapter, TestForward_Success, TestForward_NoAdapter,
  TestForward_NotSupportedByChannel) — keine neue Arbeit noetig, Backlog-Text
  war hier veraltet.
  Abweichung vom Backlog-Text: `done_when` verlangt fuer GetBySourceID "einen
  definierten Fehler wenn keine existiert" — der reale Code
  (postgres_repository.go:465-468) liefert bei einem Miss bewusst `(nil, nil)`
  ("Not found is not an error for dedup checks", von Service.Create direkt
  darauf verlassen). Diesen Vertrag geaendert haette Service.Create kaputt
  gemacht; stattdessen den realen, dokumentierten Vertrag getestet.
- gate: build ok (go build -p 2 ./internal/inbox/...) | vet ok | lint ok
  (golangci-lint run --config .golangci.yml ./internal/inbox/... — 0 issues)
  | test ok (go test -count=1 -v ./internal/inbox/..., 0 Fails, 0 Skips —
  DATABASE_URL gesetzt, docker-postgres-1 healthy, alle DB-Tests real
  gelaufen) | migration n.a. (keine neue Tabelle/Route/Policy) | rls-smoke
  n.a. (keine Policy angefasst; die neuen Cross-Tenant-Subtests belegen die
  RLS-Wirkung an den zwei ungeschuetzten Methoden direkt) | gateway-Tests
  nicht gelaufen (keine Route angefasst)
- coverage: internal/inbox/message n.a. (Bezugswert war das Paket-Aggregat
  "internal/inbox 32,3 %", nicht paket-eigen) -> internal/inbox/message
  72,4 % (go test -coverprofile + go tool cover -func, einzeln gemessen ohne
  `...`)
- mutations-probe: in `postgres_repository.go` List den Default-Filter
  `AND is_archived = false` auf `AND is_archived = true` gedreht (nur im
  `where`-Zweig der Datenabfrage, `countWhere` unveraendert gelassen) ->
  sofort drei rote Tests: TestList_ExcludesArchivedAndSnoozedByDefault (die
  archivierte statt die normale Nachricht kam zurueck),
  TestList_FiltersByChannelAndReadStatus_WithCursorPagination/filters_by_channel_and_is_read_together
  (0 statt 1 Treffer) und .../cursor_pagination_walks... (page=0 statt
  page=2, weil total/countQuery und Datenzeile durch die gezielte
  Halb-Mutation auseinanderliefen). Zurueckgedreht, `git diff` auf der Datei
  leer.
- verify vorgaenger: sauber. Commit 7bd43d4a (Iteration 72) fuegt
  ausschliesslich zwei neue Testdateien in inbox/team und inbox/routing
  hinzu (`tenant_write_test.go` je Unterpaket) plus Journal-/
  Backlog-Metadaten — kein Produktionscode, kein Proto, keine Route, kein
  RequirePermission-Guard, keine neue Tabelle, keine Migration. Keine der
  acht Fehlerklassen einschlaegig. (Der `chore`-Commit a895313b danach traegt
  ausschliesslich die Journal-Nachtragszeile fuer den Commit-Hash —
  ebenfalls unauffaellig.)
- offen: `.planning/backend-block/loop/run-loop.ps1` traegt weiterhin
  denselben unstaged -StartNotBefore-Diff wie in den Iterationen 6-72
  vermerkt — nicht meine Datei, nicht angefasst, nicht committet.
  Laufkontext-Block war auch in diesem Prompt nicht sichtbar mitgeliefert —
  Nummer aus der letzten Journal-Ueberschrift (Iteration 72) fortgezaehlt,
  Zeitstempel per `date` auf dem Loop-Rechner ermittelt (2026-08-11 04:57).
  Naechste Backlog-Unit laut Reihenfolge ist
  `e-cov-inbox-routing-thread-adapter` (Block E, inbox) — letzte Unit in
  diesem Unterpaket-Cluster, kann auf das seit Iteration 72 etablierte
  DB-Testmuster aufbauen.

## Iteration 74 — e-cov-inbox-routing-thread-adapter — done — 2026-08-11 05:04
- commit: 13a2b976
- gebaut: routing.Service.executeActions fuer alle vier Action-Typen
  (route_to_team/assign_to/add_tags/auto_reply) je mit Erfolgs- und
  Fehlerfall getestet, inkl. des dokumentierten Non-Fatal-Verhaltens von
  auto_reply (Sendefehler und Nicht-Email-Kanal liefern beide nil). Dazu
  getCachedRules/refreshCache/invalidateCache/filterByChannel (neuer
  Call-Counter `listActiveCalls` im bestehenden Mock, um TTL-Cache-Hits von
  echten Refreshs zu unterscheiden) — alles in `service_test.go`.
  thread.PostgresRepository: neue Datei `postgres_repository_canned_test.go`
  deckt CreateCannedResponse/GetCannedResponse/ListCannedResponses/
  UpdateCannedResponse/DeleteCannedResponse gegen die echte lokale DB ab,
  inkl. Not-Found-Fehlerpfade fuer Update/Delete/Get und eines
  Cross-Tenant-Isolationstests.
  adapter.ChatAdapter: neue Datei `chat_adapter_test.go` (erste Testdatei
  im `adapter`-Paket) mit einem `fakeChatClient` — FetchNewMessages (Mapping,
  Preview-Truncation auf 200 Zeichen, Nil-Client-Graceful-Degradation,
  Client-Fehler), HandleReply (Erfolg + Nil-Client-Fehler), HandleForward
  (liefert dokumentiert ErrForwardNotSupported) und MarkReadOnSource
  (Erfolg, Nil-Client-No-Op, Client-Fehler).
- gate: build ok (go build -p 2 ./internal/inbox/... ./cmd/...) | vet ok
  | lint ok (golangci-lint run --config .golangci.yml ./internal/inbox/...
  — 0 issues) | test ok (go test -count=1 ./internal/inbox/..., alle fuenf
  Unterpakete ok, 0 Fails, 0 Skips — DATABASE_URL gesetzt,
  docker-postgres-1 healthy) | migration n.a. (keine neue
  Tabelle/Route/Policy) | rls-smoke n.a. (keine Policy angefasst; der
  Cross-Tenant-Test in postgres_repository_canned_test.go belegt die
  Tenant-Trennung direkt) | gateway-Tests nicht gelaufen (keine Route
  angefasst)
- coverage: Bezugswert war das Paket-Aggregat "internal/inbox 32,3 %",
  nicht paket-eigen (wie in Iteration 73 begruendet). Einzeln gemessen ohne
  `...`: internal/inbox/routing n.a. -> 84,5 % | internal/inbox/thread
  n.a. -> 58,8 % | internal/inbox/adapter n.a. -> 23,9 % (adapter-Paket
  bleibt niedrig, weil email_adapter.go/guest_adapter.go/
  notification_adapter.go weiterhin ungetestet sind — ausserhalb des
  Scopes dieser Unit, die nur chat_adapter.go nennt)
- mutations-probe: in `routing/service.go` actionAddTags den Dedup-Guard
  (`if !existing[t] { ... }`) entfernt, sodass jeder Tag unbedingt
  angehaengt wird -> TestExecuteActions_AddTags_DedupesAgainstExisting
  sofort rot (drei statt zwei Eintraege, "vip" doppelt). Zurueckgedreht,
  `git diff` auf der Datei leer.
- verify vorgaenger: sauber. Commit 64969687 (Iteration 73) fuegt
  ausschliesslich eine neue Testdatei (`postgres_repository_reads_test.go`)
  in inbox/message hinzu, plus Journal-/Backlog-Metadaten — kein
  Produktionscode, kein Proto, keine Route, kein RequirePermission-Guard,
  keine neue Tabelle, keine Migration. Keine der acht Fehlerklassen
  einschlaegig.
- offen: `.planning/backend-block/loop/run-loop.ps1` traegt weiterhin
  denselben unstaged -StartNotBefore-Diff wie in den Iterationen 6-73
  vermerkt — nicht meine Datei, nicht angefasst, nicht committet.
  Laufkontext-Block war auch in diesem Prompt nicht sichtbar mitgeliefert
  — Nummer aus der letzten Journal-Ueberschrift (Iteration 73)
  fortgezaehlt, Zeitstempel per `date` auf dem Loop-Rechner ermittelt
  (2026-08-11 05:04). Damit ist der gesamte Inbox-Unterpaket-Cluster
  (message/routing/thread/adapter/team) aus Block E abgearbeitet — die
  naechste Backlog-Unit gehoert zu einer anderen Flaeche, siehe
  BACKLOG.yml in Reihenfolge.

## Iteration 75 — e-cov-fuhrpark-repo-core — done — 2026-08-11 05:10
- commit: c35205e8
- gebaut: neue Datei `postgres_repository_core_test.go` in
  `internal/fuhrpark`. SoftDeleteVehicle (setzt deleted_at, zweiter Aufruf
  und GetVehicle danach liefern ErrVehicleNotFound), ListVehicles (Status-
  und Search-Filter gegen echte DB), PlateExists (inkl. excludeID-Fall fuer
  Self-Update sowie ein nie benutztes Kennzeichen), FindVehiclesDueTuev
  (from/to-Fensterraender inklusive, Vehicles ausserhalb ausgeschlossen —
  Assertion auf Praesenz einzelner Vehicle-IDs statt exakter Trefferzahl,
  weil die Methode bewusst cross-tenant scannt und sonst mit parallelen
  Tests kollidieren wuerde) inkl. Idempotenz (frischer Reminder <23h
  unterdrueckt Re-Notify, per SQL auf >23h zurueckdatierter Reminder erlaubt
  es wieder), MarkTuevReminderSent Cross-Tenant-Guard (Stempel unter
  falschem Tenant schlaegt fehl UND hinterlaesst keinen Seiteneffekt am
  echten Datensatz), sowie GetBooking/UpdateBooking/DeleteBooking/
  ListBookings (Tenant-Scoping, Not-Found-Pfade, Status-/Vehicle-/
  Zeitfenster-Filter) unter Wiederverwendung der bestehenden
  seedBookingVehicle/seedBookingUser-Helfer aus booking_conflict_test.go.
  GPS-Ingestion und der Fuel-/Trip-/Document-Rest bleiben wie im Scope der
  Unit vermerkt offen fuer eine Folge-Unit.
- gate: build ok (go build -p 2 ./internal/fuhrpark/... ./cmd/...) | vet ok
  | lint ok (golangci-lint run --config .golangci.yml
  ./internal/fuhrpark/... — 0 issues) | test ok (go test -count=1
  ./internal/fuhrpark/..., 0 Fails, 0 Skips — DATABASE_URL gesetzt,
  docker-postgres-1 healthy, alle neuen Tests real gegen Postgres gelaufen)
  | migration n.a. (keine neue Tabelle/Route/Policy) | rls-smoke n.a.
  (keine Policy angefasst; die Cross-Tenant-Assertions in den neuen Tests
  belegen die Trennung direkt) | gateway-Tests nicht gelaufen (keine Route
  angefasst)
- coverage: internal/fuhrpark 37,9 % -> 50,0 % (go test -coverprofile
  ./internal/fuhrpark/ ohne `...`, go tool cover -func Summe)
- mutations-probe: in `SoftDeleteVehicle` den Guard `AND deleted_at IS
  NULL` aus dem UPDATE-WHERE entfernt -> sofort
  TestSoftDeleteVehicle_SetsDeletedAtAndIsIdempotentNotFound rot (zweiter
  SoftDeleteVehicle-Aufruf lieferte nil statt ErrVehicleNotFound).
  Zurueckgedreht, `git diff` auf der Datei leer.
- verify vorgaenger: sauber. Commit 13a2b976 (Iteration 74) fuegt
  ausschliesslich drei neue Testdateien in inbox/routing, inbox/thread und
  inbox/adapter hinzu, plus Journal-/Backlog-Metadaten — kein
  Produktionscode, kein Proto, keine Route, kein RequirePermission-Guard,
  keine neue Tabelle, keine Migration. Keine der acht Fehlerklassen
  einschlaegig.
- offen: `.planning/backend-block/loop/run-loop.ps1` traegt weiterhin
  denselben unstaged -StartNotBefore-Diff wie in den Iterationen 6-74
  vermerkt — nicht meine Datei, nicht angefasst, nicht committet.
  Laufkontext-Block war auch in diesem Prompt nicht sichtbar mitgeliefert
  — Nummer aus der letzten Journal-Ueberschrift (Iteration 74)
  fortgezaehlt, Zeitstempel per `date` auf dem Loop-Rechner ermittelt
  (2026-08-11 05:10). Naechste Backlog-Unit laut Reihenfolge ist
  `e-cov-fuhrpark-service-worker` (Block E, deckt den Rest des
  fuhrpark-Pakets: GPS-Ingestion und Fuel-/Trip-/Document-Methoden, die
  diese Unit bewusst ausgelassen hat).

## Iteration 76 — e-cov-fuhrpark-service-worker — done — 2026-08-11 05:16
- commit: (siehe unten, wird nach diesem Journal-Eintrag committet)
- gebaut: neue Datei `service_extended_test.go` in `internal/fuhrpark`. Deckt die
  Service-Validierungs- und Delegationspfade fuer FuelLog (CreateFuelLog: VehicleID/
  Liters<=0/CostCents<0/MileageKm<0, UpdateFuelLog: zero ID, ListFuelLogs-Passthrough),
  TripLog (CreateTripLog: VehicleID/Start-/EndLocation leer plus EndKm<StartKm,
  UpdateTripLog: zero ID, ListTripLogs-Passthrough), VehicleDocument (CreateVehicleDocument:
  VehicleID/ObjectKey/Name, ListVehicleDocuments-Passthrough), DriverLicense
  (CreateDriverLicense: DriverID/NextCheckDueDate plus CheckedAt-Default, UpdateDriverLicense:
  ID/NextCheckDueDate, ListDriverLicenses-Passthrough) und GPS (IngestGpsPositions:
  VehicleID/leere positions plus Happy-Path-Zaehlung) ab — je mit einem Fehlerpfad und
  mindestens einem Happy-Path pro Methode. Zusaetzlich `fakeEventEmitter` (Payload-Recorder
  mit optionalem Fehler) an `TuevWorker.WithEventEmitter` gehaengt:
  `buildTuevEventPayload` mit Prioritaet Urgent fuer "1_day" und Normal fuer "7_days" direkt
  geprueft, sowie zwei End-to-End-Tests ueber `ProcessTuevReminders` — Emit-Erfolg (Payload
  aufgezeichnet, Prioritaet korrekt, MarkTuevReminderSent gesetzt) und Emit-Fehler
  (non-fatal: Scan laeuft weiter, MarkTuevReminderSent wird trotzdem aufgerufen).
- gate: build ok (go build -p 2 ./internal/fuhrpark/... ./cmd/...) | vet ok | lint ok
  (golangci-lint run --config .golangci.yml ./internal/fuhrpark/... — 0 issues) | test ok
  (go test -count=1 ./internal/fuhrpark/..., 75 Subtests PASS, 0 Fails, 0 Skips —
  DATABASE_URL gesetzt, docker-postgres-1 healthy) | migration n.a. (keine neue
  Tabelle/Route/Policy) | rls-smoke n.a. (keine Policy angefasst, reine Service-Unit-Tests
  gegen einen In-Memory-Mock, keine DB involviert) | gateway-Tests nicht gelaufen (keine
  Route angefasst)
- coverage: internal/fuhrpark 50,0 % -> 54,5 % (go test -coverprofile ./internal/fuhrpark/
  ohne `...`, go tool cover -func Summe)
- mutations-probe: zwei Proben, beide gefangen. (1) `CreateFuelLog`-Guard von
  `req.Liters <= 0` auf `req.Liters < 0` gelockert -> sofort
  TestService_CreateFuelLog_InvalidInput/non-positive_liters rot (erwarteter
  ErrInvalidInput blieb aus). (2) in `buildTuevEventPayload` die Fensterprüfung von
  `window == "1_day"` auf `window == "7_days"` vertauscht -> sofort
  TestBuildTuevEventPayload_PriorityByWindow UND
  TestTuevWorker_WithEventEmitter_EmitsOnDueVehicle rot (beide erwarteten "urgent",
  bekamen "normal"). Beide Male zurueckgedreht, `git diff` auf service.go und worker.go
  leer.
- verify vorgaenger: sauber. Commit c35205e8 (Iteration 75) fuegt ausschliesslich eine
  neue Testdatei (`postgres_repository_core_test.go`) in fuhrpark hinzu, plus Journal-/
  Backlog-Metadaten — kein Produktionscode, kein Proto, keine Route, kein
  RequirePermission-Guard, keine neue Tabelle, keine Migration. Keine der acht
  Fehlerklassen einschlaegig.
- offen: `.planning/backend-block/loop/run-loop.ps1` traegt weiterhin denselben
  unstaged -StartNotBefore-Diff wie in den Iterationen 6-75 vermerkt — nicht meine
  Datei, nicht angefasst, nicht committet. Laufkontext-Block war auch in diesem Prompt
  nicht sichtbar mitgeliefert — Nummer aus der letzten Journal-Ueberschrift
  (Iteration 75) fortgezaehlt, Zeitstempel per `date` auf dem Loop-Rechner ermittelt
  (2026-08-11 05:16). Damit ist der gesamte fuhrpark-Servicecluster (Repository-Core aus
  Iteration 75 + Service/Worker aus dieser Iteration) aus Block E abgearbeitet — GPS-
  Aggregation (`GetVehicleRoutes`/`GetGpsPositions`, reine Passthroughs ohne eigene Logik)
  und die restlichen Repository-DB-Pfade fuer FuelLog/TripLog/VehicleDocument/
  DriverLicense (bereits in tenant_write_test.go/driver_license_test.go/
  triplog_export_test.go abgedeckt) blieben bewusst aussen vor. Naechste Backlog-Unit
  laut Reihenfolge: `e-cov-chat-message-repo-reads` (Block E, chat/message-Repository).

## Iteration 77 — e-cov-chat-message-repo-reads — done — 2026-08-11 05:24
- commit: e471f69f
- gebaut: neue Datei `postgres_repository_reads_test.go` in `internal/chat/message`.
  Deckt GetByID (Happy-Path + ErrMessageNotFound), List (DESC-Reihenfolge inkl.
  Thread-Replies plus ExcludeReplies-Filter), ListReplies (ASC-Reihenfolge, Limit),
  ChannelExists/IsChannelArchived/IsMember/GetMemberRole (je True- und False-/
  Fehlerfall), GetUserInfo, GetMentionsByMessages/GetMentionsForUser (ueber echte
  CreateMentions-Zeile), GetFilesByMessageIDs, GetChannelMemberIDs, GetDMRecipient
  (Nicht-DM-Kanal plus DM-Kanal aus beiden Blickrichtungen), GetChannelName,
  GetGuestDisplayName/IsChannelGuestEnabled und DecrementReplyCount (inkl.
  GREATEST-Floor bei bereits 0) ab. `channel_memberships` hat keine `id`-Spalte
  (composite PK) — dafuer `seedMembership` als lokaler Raw-INSERT analog zum
  bestehenden Muster in `internal/chat/channel/tenant_write_test.go`.
- gate: build ok (go build -p 2 ./internal/chat/... ./cmd/gateway/...) | vet ok
  (go vet ./internal/chat/message/...) | lint ok (golangci-lint run --config
  .golangci.yml ./internal/chat/message/... — 0 issues) | test ok (go test -count=1
  ./internal/chat/message/, 11 neue Tests PASS zusaetzlich zu allen bestehenden —
  DATABASE_URL gesetzt, docker-postgres-1 healthy; go test -count=1
  ./internal/chat/... — alle 7 Unterpakete ok) | migration n.a. (keine neue Tabelle/
  Route/Policy) | rls-smoke n.a. (keine Policy angefasst, alle Assertions pruefen
  Rueckgabewerte der Repository-Methoden, keine neuen Cross-Tenant-Pfade)
  | gateway-Tests nicht gelaufen (keine Route angefasst)
- coverage: internal/chat/message 50,9 % -> 72,4 % (go test -coverprofile
  ./internal/chat/message/ ohne `...`, go tool cover -func Summe)
- mutations-probe: in `DecrementReplyCount` den Floor-Guard `GREATEST(reply_count - 1,
  0)` auf `reply_count - 1` gelockert -> sofort TestPostgresRepository_DecrementReplyCount
  rot ("reply_count floored at zero: expected 0, got -1"). Zurueckgedreht, `git diff`
  auf der Datei leer.
- verify vorgaenger: sauber. Commit af9f820a (Iteration 76) fuegt ausschliesslich eine
  neue Testdatei (`service_extended_test.go`) in fuhrpark hinzu, plus Journal-/
  Backlog-Metadaten — kein Produktionscode, kein Proto, keine Route, kein
  RequirePermission-Guard, keine neue Tabelle, keine Migration. Keine der acht
  Fehlerklassen einschlaegig.
- offen: `.planning/backend-block/loop/run-loop.ps1` traegt weiterhin denselben
  unstaged -StartNotBefore-Diff wie in den Iterationen 6-76 vermerkt — nicht meine
  Datei, nicht angefasst, nicht committet. Laufkontext-Block war auch in diesem Prompt
  nicht sichtbar mitgeliefert — Nummer aus der letzten Journal-Ueberschrift
  (Iteration 76) fortgezaehlt, Zeitstempel per `date` auf dem Loop-Rechner ermittelt
  (2026-08-11 05:24). Beim Bauen zunaechst ein reales Bug-Muster in den eigenen Tests
  gefunden und korrigiert: `defer pool.Close()` in Kombination mit `t.Cleanup`-basierten
  Aufraeumfunktionen aus einer Fixture-Helper-Funktion feuert die Aufraeum-DELETEs NACH
  dem Pool-Close (defer laeuft beim Funktions-Return, t.Cleanup erst danach) — alle elf
  neuen Tests protokollierten "closed pool" beim Aufraeumen und liessen Testzeilen in der
  DB zurueck. Gefixt durch `t.Cleanup(func() { pool.Close() })` als ERSTE Cleanup-Registrierung
  (LIFO: zuerst registriert, zuletzt ausgefuehrt) statt `defer pool.Close()`. Naechste
  Backlog-Unit laut Reihenfolge: `e-cov-chat-channel-search-repo` (Block E, chat/channel
  Lese-Methoden + neues DB-Integrationstest-Muster fuer chat/search).

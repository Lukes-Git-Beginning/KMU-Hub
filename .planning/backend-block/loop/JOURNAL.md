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
- commit: <wird nach dem Commit nachgetragen>
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

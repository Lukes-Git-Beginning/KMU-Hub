# Backend-Nachtloop — Journal Lauf 11

Append-only. Ein Eintrag je Iteration, **ans Dateiende**, nie einsortieren. Form und Pflichtzeilen
stehen in `ITERATION.md` Schritt 6.

Frühere Läufe liegen vollständig im Archiv:
`archive/lauf-1-2/` (58 Units) · `archive/lauf-3/` (61) · `archive/lauf-4/` (54) ·
`archive/lauf-5/` (41) · `archive/lauf-6/` (46) · `archive/lauf-7/` (71) ·
`archive/lauf-8/` (94) · `archive/lauf-9/` (37) · `archive/lauf-10/` (93, inkl. `logs/`).

---

## Laufkontext

- **Ausgangspunkt:** Lauf 10 gemergt als `f87ffdcf` und deployt. `main` = `backend-loop` =
  `832eee78`, beide Branches deckungsgleich (Fast-Forward, **nicht** rebased). CI grün auf
  `a859974c` (Rerun 32571951649, alle fünf Jobs inkl. E2E), CD grün (32572607117), Produktion
  antwortet mit `commit: a859974c`.
- **Migrationen:** Repo-Kopf = lokaler DB-Kopf = Produktionskopf = **322**, `schema_migrations`
  clean. Nächste freie Nummer wäre 323 — aber immer zur Laufzeit ermitteln:
  `ls backend/migrations | grep -E "^[0-9]{6}" | sort | tail -1`.
  **Keine** Unit dieses Laufs bringt eine Migration mit. Entsteht im Lauf trotzdem eine
  (etwa aus einem Scan-Fund), gilt: `tenant_id UUID NOT NULL` + RLS-Policy oder ein Eintrag in
  der System-Global-Liste, kein stiller dritter Weg.
- **Lokale DB:** Startbedingung. `run-loop.ps1` prüft im Vorflug Port 5432, die Anmeldung als
  `kmuhub_app` und den Migrationskopf und bricht ab, wenn eins davon fehlt. Grund:
  `testutil.SkipIfNoDB` (`backend/internal/testutil/rls.go:24`) fragt nur, **ob**
  `DATABASE_URL` gesetzt ist — ohne die Variable meldet ein Paket `ok` für Tests, die gar nicht
  gelaufen sind.
- **Rolle:** `kmuhub_app` (NOSUPERUSER NOBYPASSRLS), niemals `kmuhub` — der Superuser hat
  BYPASSRLS und würde jede RLS-Lücke durchwinken.
- **Coverage-Ausgangslage** (CI-Run 32570176303): gesamt **62,7 %** bei Gate 15 %.
  Schwächste Kernpakete nach ungedeckten Zeilen: `internal/gateway` **54,1 %** (10.448),
  `internal/server` 70,5 (6.023), `internal/biz` 63,7 (3.190), `internal/work` 61,2 (1.876).
  Geldpakete: creditnote 28,2 · quote 33,3 · invoice 34,8 · payment 46,4 · dunning 61,8.
  Vollständige Liste im Kopf von `BACKLOG.yml`.
- **Umfang:** 64 vorab geschriebene Units — Block A (16, G2-Substanz), Block B (22,
  Geld-Repositories gegen echtes SQL), Block C (16, Gateway- und Server-Geldflächen),
  Block D (10, Muster-Scans). Block D legt weitere Units zur Laufzeit an; in Lauf 10 haben
  9 Scans 45 Zusatz-Units erzeugt. Fenster 23:00 → 14:00, `-MaxIterations 120`
  (Default ist 100; Lauf 10 endete bei 92 und wäre sonst gedeckelt worden).

## Der rote Faden

**Geld- und Compliance-Pfade vor dem ersten zahlenden Kunden (G2).** Entscheidung Luke,
2026-08-22. Etappe 3 / G1 ist backend-seitig abgeräumt; was davon übrig ist, braucht Frontend,
`deploy/` oder Ops und kann der Loop nicht anfassen.

## Was in diesem Lauf gilt

- **Vier Prämissen des Entwurfs haben die Gegenprüfung nicht überstanden** und stehen als
  Befunde 1 bis 6 im Kopf von `BACKLOG.yml`. Vor dem Bauen lesen — zwei geplante Units sind
  daran ersatzlos gestrichen worden. Wichtigste davon: der Lexware-Doppelzustellungs-Schaden
  existiert nicht (`noopEmitter{}`, `SetEventEmitter` nie aufgerufen), und die Geld-Repositories
  haben sehr wohl Tests — sie tragen nur `//go:build integration` und laufen deshalb weder im
  lokalen Gate noch im Coverage-Job.
- **Neue DB-Tests ungetagt schreiben.** Kein `//go:build integration`. Bausteine:
  `testutil.SkipIfNoDB`, `PoolFromEnv`, `EnsureTenant`, `SeedRow`, `CleanupRow`,
  `WithTenantCtx`. Vorlage: `backend/internal/idempotency/postgres_repository_db_test.go`.
- **Ein DB-Test, der lokal grün ist, weil der Pool nur eine warme Verbindung hatte, beweist
  nichts.** Wer eine Ressource *pro Verbindung* prüft (Advisory Locks, Session-GUCs, temporäre
  Tabellen), hält vorher eine zweite Verbindung fest. Genau das hat in Lauf 10 den
  Advisory-Lock-Leak aufgedeckt, den `idempotency.CleanupWithLock` und der neue
  `gdpr`-Retention-Scheduler teilten — der Scheduler wäre nach dem ersten Tick dauerhaft
  eingeschlafen, ohne Fehlermeldung. Behoben in `778a2e44`.
- **Wer ein bestehendes Muster als Vorlage kopiert, kopiert seine Fehler mit.** Genau so kam
  derselbe Leak aus `idempotency` in den Retention-Scheduler. Vorlage vorher prüfen, nicht nur
  nachbauen.
- **`-race` läuft auf dieser Maschine nicht** (kein `gcc` im PATH). Wo eine Unit `-race`
  verlangt, ist CI der einzige Beweis — das gehört in die `offen:`-Zeile.
- **Lokales Postgres-Verbindungslimit:** `go test` über mehrere DB-Pakete mit vollem
  Parallelismus reißt mit `53300 remaining connection slots are reserved` ab. `-p 1` verwenden
  oder das Paket-Set eingrenzen. Kein Code-Fehler.
- **Die lokale Dev-Postgres trägt 13,8k Müll-Tenants** aus alten Läufen (Produktion hat 1).
  Kein Test darf über `tenants` iterieren, und jeder Test räumt auf, was er seedet.
- **Zwei Nullen sind kein bestandener RLS-Smoke, sondern ein kaputter.** Bestanden heißt eigener
  Tenant grösser 0 UND fremder Tenant gleich 0.
- Root Cause statt Symptom: vor jedem Fix alle Caller greppen. Mutations-Probe ist Pflicht.
- Eine Prämisse aus dem Backlog, die sich am Code als falsch erweist, wird **nicht trotzdem
  gebaut** — sie wird hier widerlegt und die Unit auf `blocked` gesetzt.
- Scan-Units ändern kein Verhalten. `neue-units:` muss IDs nennen, die wirklich in
  `BACKLOG.yml` stehen. Ein abgebrochener Scan nennt in `offen:`, **was** tief geprüft, **was**
  nur gegrept und **was** gar nicht angesehen wurde — Lauf 10 Iteration 46 ist die Vorlage und
  hat den Nachfolge-Scan D2 überhaupt erst möglich gemacht.
- Gesperrt: Frontend/Desktop, CSAT und Public-Token-Routen, `internal/biz/bexio`,
  Dependency-Bumps, `deploy/`, Migrations-Umnummerierungen, Preise und Modul-Zuschnitt.
  `RETENTION_MODE` bleibt `dry_run`.

---

## Iteration 1 — fix-gobd-export-tax-grouping-rate-key-collision — done — 2026-08-22 23:00
- commit: 90d4b2ff
- gebaut: Die USt-Gruppierung des GoBD-Exports liegt nicht mehr im gRPC-Handler, sondern als
  `dunning.BuildGoBDRows` im Service-Paket (`internal/biz/dunning/gobd_rows.go`);
  `financeDocToGoBDRows` in `internal/server/biz_grpc.go` ist ersatzlos entfallen. Der
  Gruppenschlüssel ist jetzt der exakte Satz (`datev.RateKey`, "19"/"7"/"7.5") statt
  `TaxRate.IntPart()`; das Zeilennetto wird vor dem Summieren auf Cent gerundet. Die beiden
  Mapping-Funktionen `RevenueAccountForRateAndMode` und `BUSchluesselForRate` nehmen jetzt
  `decimal.Decimal` statt eines abgeschnittenen Strings und liefern für jeden nicht-deutschen
  Satz das generische Erlöskonto 8200 ohne BU-Schlüssel, statt ihn auf 8300/BU 2 zu buchen.
  Der zweite Caller `datev/exporter.go` (`truncateRate`, gleiche Fehlerklasse) ist mitgezogen
  und `truncateRate` entfernt. Der ungenutzte `RevenueAccountForRate(string)` ist entfallen —
  null Produktionsaufrufer, und seine String-Signatur war genau die Falle.
- gate: build ok | vet ok | lint ok (0 issues) | test ok | migration n.a. | rls-smoke n.a.
- coverage: internal/server 70,5 % -> 70,6 % · internal/biz/dunning 61,8 % -> 65,1 % ·
  internal/biz/datev 79,7 % -> 79,5 % (datev sinkt, weil zwei vollständig abgedeckte Funktionen
  gelöscht wurden — der Nenner schrumpft, ungedeckte Zeilen kamen keine dazu)
- mutations-probe: zwei Läufe, beide am finalen Tree.
  (a) `key := datev.RateKey(item.TaxRate)` zurück auf `fmt.Sprintf("%d", item.TaxRate.IntPart())`
  gedreht -> `TestBuildGoBDRows_SeparatesFractionalRateFromWholeRate` rot, und die Ausgabe zeigt
  den Produktionsschaden wörtlich: 7 % und 7,5 % fallen zu EINER Zeile Konto 8300 / BU 2 mit
  Netto 300,00 zusammen (DATEV rechnet darauf 21,00 statt der belegten 22,00 USt).
  (b) `.Round(2)` am Zeilennetto entfernt -> `TestBuildGoBDRows_RoundsLineNetBeforeSumming` rot
  (9,99 statt 10,00; brutto 11,89 statt 11,90). Beide zurückgedreht, Paket wieder grün, Diff sauber.
- verify vorgaenger: sauber — die drei Commits vor dieser Iteration (`69595874`, `832eee78`,
  `a859974c`) berühren ausschließlich `.planning/`; kein Go-Code, keine Migration, keine Route,
  also keine der acht Fehlerklassen anwendbar.
- neue-units: keine. Der Rundungs-Divergenz-Befund in `datev/exporter.go` (nur der Bruttobetrag
  wird gerundet) steht bereits als A2 im Backlog und wurde bewusst NICHT mitgefixt; ebenso die
  Schwesterfunktion `tax.Calculate` (A3).
- offen: (1) DB-Gate lief: `DATABASE_URL` als `kmuhub_app` gesetzt, **0 übersprungene Tests** in
  allen drei Paketen, 12 `SkipIfNoDB`-Tests real gelaufen (7 in `internal/server`, 3 in
  `internal/biz/datev`, keiner in `dunning`). `go test ./internal/gateway/` grün, obwohl keine
  Route angefasst wurde. (2) Fachliche Annahme, die Luke prüfen sollte: für einen Satz, den SKR03
  nicht kennt (7,5 %, historisch 16 %/5 %), buche ich auf **8200 "Erlöse" ohne BU-Schlüssel**
  statt wie bisher still auf 8300/BU 2. Das ist konservativ (kein automatischer Steuerabzug,
  der Steuerberater muss zuordnen) — falls die Buchhaltung ein anderes Sammelkonto will, ist es
  eine Konstante in `datev/mapping.go`. (3) Der CSV-Spaltenwert "Steuersatz" trägt jetzt den
  exakten Satz; für 19 und 7 ändert sich nichts, ein früher als "7" exportierter 7,5-%-Beleg
  erscheint künftig als "7.5".

---

## Iteration 2 — fix-tax-rounding-divergence-across-implementations — done — 2026-08-22 23:10
- commit: 90353058
- gebaut: Die vier Steuerarithmetiken runden weiterhin unterschiedlich — aber jetzt steht an
  jeder Stelle, welche Regel ihre Reihenfolge fordert, und die zwei Stellen, die gegen EN 16931
  verstießen, sind korrigiert. Im E-Rechnungs-Generator (`einvoice/generator_doc.go`) wird der
  Zeilenbetrag jetzt auf Cent gerundet, BEVOR er summiert wird (BR-CO-10 vergleicht BT-106 exakt
  gegen die Summe der geschriebenen Zeilenbeträge und kennt keine Toleranz — vorher wich BT-106
  bei Bruchmengen von den eigenen Zeilen ab, was jedes XRechnung-Portal ablehnt). Die Gruppensteuer
  BT-117 wird jetzt aus dem fertigen Gruppennetto BT-116 abgeleitet (BR-CO-17) statt aus
  aufsummierten Zeilensteuern. `totalsTolerance` ist von der Konstanten 0,01 zu einer Funktion der
  Zeilenzahl geworden (0,005 je Zeile, Boden 0,01), weil Schreibseite und Generator seither
  systematisch bis zu einen halben Cent je Zeile auseinanderliegen — ohne das hätte der Fix
  gewöhnliche fünfzeilige Rechnungen vom Export ausgesperrt. Dazu Rundungs-Notizen an allen vier
  Stellen (`tax/calculator.go`, `einvoice/generator_doc.go`, `einvoice/parser.go`,
  `datev/exporter.go`), jeweils mit der Regel, die sie fordert.
- gate: build ok | vet ok | lint ok (0 issues) | test ok | migration n.a. | rls-smoke n.a.
- coverage: internal/biz/einvoice 81,9 % -> 82,1 % (eigene Messung vor/nach der Änderung,
  `go tool cover -func`; deckt sich mit dem `coverage_start:` der Unit).
  internal/biz/tax und internal/biz/datev nur Kommentare, Coverage unverändert.
- mutations-probe: vier Läufe, jeweils am finalen Tree, jeweils zurückgedreht, Diff sauber.
  (a) `net = net.Round(2)` im Generator entfernt -> `TestRoundtrip_FractionalQuantities_
  HoldsEN16931SumRules` rot mit `BR-CO-10: sum of line nets 88.25 != BT-106 88.24`.
  (b) Gruppensteuer zurück auf Aufsummieren der Zeilensteuern -> derselbe Test rot mit
  `BR-CO-17: group 0 net 66.58 at 19% is 12.65, document says 12.66` plus `total tax: want 14.17,
  got 14.18`.
  (c) `totalsTolerance` zurück auf feste 0,01 -> `TestGenerateUBL_Rejections/write-path_rounding_
  order_stays_acceptable_on_many_lines` rot (ErrTotalsMismatch auf einer korrekten Rechnung).
  (d) Toleranz auf 0,05 je Zeile aufgerissen -> `.../stale_total_is_still_caught_on_many_lines`
  rot. (c) und (d) zusammen klemmen die Toleranz von beiden Seiten ein.
  Anmerkung zur Arbeitsweise: die erste Fassung des Fixtures bestand Probe (b) NICHT — bei 7,5
  und 19 % fielen beide Rundungsreihenfolgen zufällig auf dieselbe Zahl. Das Fixture wurde
  daraufhin auf fünf Zeilen umgebaut (zwei Materialpauschalen zu 8,29 EUR bei 19 %), bis beide
  Regeln einzeln brechen. Ein Fixture, das nur eine der beiden Regeln bricht, lässt die andere
  ungesehen zurückregressieren.
- verify vorgaenger: sauber — `90d4b2ff` verschiebt `financeDocToGoBDRows` aus
  `internal/server/biz_grpc.go` als `dunning.BuildGoBDRows` ins Service-Paket. Kein neuer
  Gateway-Handler (also keine gRPC-Umgehung), kein Stub oder TODO im neuen Pfad, kein `.proto`,
  keine Migration, kein `RequirePermission`, keine neue Tabelle, keine Route, keine
  Response-Form geändert. Der Reverse-Charge-/Kleinunternehmer-Zweig ist vollständig
  mitgezogen, der vorher verschluckte `json.Unmarshal`-Fehler wird jetzt geloggt.
  `d21d2f2d` berührt nur das JOURNAL.
- neue-units: `fix-write-path-line-total-unrounded-everywhere` (ans Backlog-Ende, deps auf diese
  Unit). Sieben Schreibstellen berechnen `LineTotal = Quantity.Mul(UnitPrice)` ohne `.Round(2)`:
  `invoice/service.go:151` und `:473`, `quote/service.go:121` und `:305`,
  `creditnote/service.go:100`, `recurring/service.go:482`, `pdf/templates.go:154`. Dadurch weicht
  das gespeicherte Netto (PDF, UI) bei Bruchmengen um Cent vom Netto der XRechnung ab — vier
  Zeilen zu 1,5 x 33,33 EUR ergeben gespeichert 199,98 und im XML 200,00. Der Hinweis steht auch
  in den `notes` von `fix-tax-calculator-line-total-unrounded` (A4), weil die beiden zusammen
  entschieden werden müssen.
- offen: (1) DB-Gate lief: `DATABASE_URL` als `kmuhub_app`, **0 übersprungene Tests** über
  einvoice/tax/datev/invoice (300 Tests), darunter die drei Roundtrip-Tests und die
  Tenant-Isolation. `go test ./internal/gateway/` und `./internal/server/` grün, obwohl keine
  Route angefasst wurde. (2) **Die Toleranz ist bewusst weicher geworden**, und das ist der
  Punkt, den Luke ansehen sollte: `totalsTolerance` erlaubt jetzt 0,005 je Zeile statt pauschal
  0,01. Bei einer 40-Zeilen-Rechnung sind das 0,20 EUR, die als "Rundung" durchgehen. Das ist die
  obere Schranke der Rundungsreihenfolge, also mathematisch begründet — aber die Schranke ist
  nur so lange nötig, wie die Schreibseite ungerundet summiert. Mit
  `fix-write-path-line-total-unrounded-everywhere` kann sie zurück auf einen festen Wert.
  (3) Fachliche Nebenwirkung, die in der Buchhaltung sichtbar wird: die XRechnung eines
  Bruchmengen-Belegs trägt ab jetzt bis zu wenige Cent mehr Netto als das PDF derselben
  Rechnung. Vorher trug sie dieselbe Zahl wie das PDF — war dafür aber BR-CO-10-widrig und
  hätte beim Empfänger abgelehnt werden können. Beide Zahlen decken sich erst wieder, wenn die
  Schreibseite nachzieht. (4) Nicht angefasst, weil eigene Units: `tax.Calculate` rundet die
  Zeilensteuer, aber nicht das Zeilennetto (A4), und sein Rate-Key trunkiert weiterhin (A3).

---

## Iteration 3 — fix-tax-calculator-rate-key-collision — done — 2026-08-22 23:25
- commit: 592a44c1
- gebaut: `tax.Calculate` (`internal/biz/tax/calculator.go`) hatte zwei Fehler in einer Zeile
  (`rateKey := item.TaxRate.Truncate(0).StringFixed(0)`). Erstens fiel ein Satz von 7,5 % unter
  denselben Schlüssel "7" wie 7 % — beide Sätze wurden zu einer Gruppe verschmolzen, mit dem
  falschen Steuerbetrag je Satz. Neue Funktion `rateGroupKey` formatiert stattdessen minimal
  (`"19"`, `"7"`, aber `"7.5"` bleibt `"7.5"`), per `StringFixed(2)` + `TrimRight` auf Nullen und
  Punkt — kein Float, wie vom Package-Kommentar gefordert. Zweitens erzeugte eine 0-%-Zeile im
  Standardmodus (z. B. eine Position ausserhalb des Anwendungsbereichs auf einer sonst
  steuerpflichtigen Rechnung) eine eigene Gruppe "0" mit 0,00 Steuer; diese Nullgruppe hätte in
  der Rechnung und der E-Rechnung als USt-Kategorie S mit Satz 0 auftauchen können, was ein
  EN-16931-Validator zurückweist (S verlangt einen positiven Satz). Die Zeile wird jetzt zwar
  weiter besteuert (0 EUR Steuer, korrekt) und geht ins Subtotal ein, erzeugt aber keinen
  TaxByRate-Eintrag mehr.
  Alle fünf Aufrufer geprüft (`invoice/service.go`, `quote/service.go`, `creditnote/service.go`,
  `recurring/service.go`, `server/biz_grpc.go:parseProtoTaxBreakdown`, `pdf/templates.go`): keiner
  parst den Rate-Key als Zahl, alle reichen ihn als String-Map-Key durch (JSONB-Storage bzw.
  `fmt.Sprintf("MwSt %s%%", rate)` im PDF) — kein Aufrufer musste mitgezogen werden.
- gate: build ok | vet ok | lint ok (0 issues) | test ok | migration n.a. | rls-smoke n.a.
- coverage: internal/biz/tax 100 % -> 100 % (eigene Messung vor/nach, `go tool cover -func`; das
  Paket war schon vor dem Fix voll gedeckt — der Fund war ein Korrektheitsbug, keine Lücke,
  Coverage sagt hier nichts aus, siehe Scope-Text der Unit).
- mutations-probe: zwei Läufe, jeweils per sed-Mutation gegen eine Kopie geprüft und aus der
  Kopie zurückgestellt (kein `git checkout`, siehe offen (1)), Diff am Ende sauber.
  (a) `rateGroupKey(item.TaxRate)` zurück auf `item.TaxRate.Truncate(0).StringFixed(0)` ->
  `TestCalculate_FractionalRate_DoesNotCollideWithWholeRate` rot: `map[7:14.5]` statt zwei
  getrennter Gruppen, "tax at 7.5% should be 7.50, got 0".
  (b) `if !item.TaxRate.IsZero()` zu `if true` -> `TestCalculate_ZeroRateLineInStandardMode_
  CreatesNoRateGroup` rot: `map[0:0 19:19]` statt einer Gruppe, "TaxByRate must not carry a \"0\"
  key".
- verify vorgaenger: sauber — `90353058` (Iteration 2) rundet Zeilennetto vor der Summierung im
  E-Rechnungs-Generator und leitet BT-117 aus dem fertigen Gruppennetto ab; kein neuer
  Gateway-Handler, kein Stub/TODO, kein `.proto`, keine Migration, kein `RequirePermission`,
  keine neue Tabelle, keine Route, keine Response-Form geändert. `e357c10c` trägt nur die
  fehlende SHA im Journal nach, kein Codewechsel.
- neue-units: keine
- offen: (1) Bei dieser Iteration hat `git checkout -- <datei>` nach der ersten Mutations-Probe
  den kompletten unversionierten Fix auf den letzten Commit zurückgesetzt statt nur die Probe
  rückgängig zu machen (der Fix war ja selbst noch nicht committet). Der Fix musste danach neu
  geschrieben werden — inhaltlich identisch (per Diff verglichen), aber ein vermeidbarer
  Zeitverlust. Für künftige Iterationen: Mutations-Proben gegen eine `cp`-Sicherungskopie fahren
  und aus der Kopie zurückschreiben, nicht `git checkout` auf einer Datei mit uncommiteten
  Änderungen. (2) DB-Gate lief mit `DATABASE_URL` als `kmuhub_app`, 0 übersprungene Tests über
  tax/invoice/quote/creditnote/recurring/pdf/server. `go test ./internal/gateway/ -run
  TestOpenAPIRouteDrift` grün, obwohl keine Route angefasst wurde. (3) `fix-tax-calculator-line-
  total-unrounded` (nächste Unit, deps auf diese) rundet das Zeilennetto — dort wird auch
  `assertTotalsMatch`/`totalsTolerance` aus A2 gegengeprüft, wie in deren Notes verlangt.

---

## Iteration 4 — fix-tax-calculator-line-total-unrounded — done — 2026-08-22 23:31
- commit: 0c99c0b3
- gebaut: `tax.Calculate` rundete die Zeilensteuer, aber nicht den Zeilenbetrag selbst
  (`lineTotal := item.Quantity.Mul(item.UnitPrice)` ging ungerundet in `subtotal`), obwohl der
  Funktionskommentar ausdrücklich das Gegenteil versprach. Jetzt wird `lineTotal` mit `.Round(2)`
  gerundet, bevor es sowohl in `subtotal` einläuft als auch als Basis für die Zeilensteuer dient
  (`lineTax := lineTotal.Mul(item.TaxRate)...`). Damit rundet die Funktion Netto UND Steuer auf
  derselben Zeilenebene, in derselben Reihenfolge wie der E-Rechnungs-Generator seit A2
  (BR-CO-10: Zeilennetto runden, dann summieren) — die beiden Schreibpfade zum Nettobetrag
  stimmen jetzt überein. Der Funktionskommentar ist entsprechend korrigiert: die alte Aussage
  "Rounding order ... is NOT the one the e-invoice generator uses" war nach A2 bereits falsch
  (A2 hat den Generator genau auf diese Reihenfolge umgestellt) und beschrieb jetzt nur noch den
  verbleibenden Unterschied — Zeilensteuer-Summe hier vs. Gruppensteuer-aus-Gruppennetto in
  BR-CO-17 der XML.
- gate: build ok | vet ok | lint ok (0 issues) | test ok | migration n.a. | rls-smoke n.a.
- coverage: internal/biz/tax 100 % -> 100 % (eigene Messung vor/nach, `go tool cover -func`; das
  Paket war schon vor dem Fix voll gedeckt — Korrektheitsbug, keine Lücke).
- mutations-probe: ein Lauf gegen eine `cp`-Sicherungskopie (nicht `git checkout`, siehe
  Iteration-3-Lehre), zurückgeschrieben, Diff sauber. `.Round(2)` von `lineTotal` entfernt ->
  `TestCalculate_FractionalQuantity_RoundsLineNetBeforeSumming` rot: "subtotal should be rounded
  to 50.00, got 49.9995" und "gross total should be 59.50, got 59.4995".
- verify vorgaenger: sauber — `db1210bb` (letzter Commit vor dieser Iteration) ändert nur eine
  Zeile im JOURNAL (SHA nachgetragen für Iteration 3), kein Codewechsel, keine der acht
  Fehlerklassen anwendbar.
- neue-units: keine
- offen: (1) DB-Gate lief: `DATABASE_URL` als `kmuhub_app`, **0 übersprungene Tests** über
  tax/einvoice/invoice/quote/creditnote/recurring. `go test ./internal/gateway/` grün, obwohl
  keine Route angefasst wurde. (2) Auswirkung auf `assertTotalsMatch`/`totalsTolerance`
  (`einvoice/generator_doc.go`), wie von der Unit verlangt: die **Netto-Seite** der Toleranz ist
  jetzt größtenteils entwertet, weil `tax.Calculate` denselben Rundungsschritt (Zeilennetto vor
  dem Summieren runden) wie der Generator macht — die Subtotal-Drift, die die Toleranz von
  0,005/Zeile ursprünglich auffangen sollte, entsteht auf diesem Weg nicht mehr. Die
  **Steuer-Seite** bleibt different: hier summiert die Zeilensteuer je Zeile, während BR-CO-17
  die Gruppensteuer aus dem fertigen Gruppennetto ableitet — dieser Rest-Unterschied ist es, für
  den `totalsTolerance` weiterhin gebraucht wird. `totalsTolerance` selbst wurde NICHT geändert
  (0,005/Zeile, Boden 0,01) — die Zahl ist nur nicht mehr auf beide Ursachen zurückzuführen,
  sondern nur noch auf die Steuer-Rundungsreihenfolge. Verkleinern wäre verfrüht: die
  Zeilenposten selbst (`invoice.LineItems[i].LineTotal = Quantity.Mul(UnitPrice)`, ungerundet)
  tragen weiterhin dieselbe Diskrepanz auf Anzeige-/PDF-Ebene, das ist die separate, bereits im
  Backlog stehende `fix-write-path-line-total-unrounded-everywhere` (deps auf diese Unit). (3)
  `assertTotalsMatch` selbst wurde nicht angefasst und ist mit den Bestandstests weiterhin grün.

## Iteration 5 — feat-einvoice-codelist-validation — done — 2026-08-22 23:36
- commit: 6d5816c3
- gebaut: `validateInvoiceDoc` (`internal/biz/einvoice/validation.go`) prüft jetzt zwei
  Codelisten, die bisher gar nicht geprüft wurden: BR-CL-04 (Rechnungswährung BT-5 muss
  ISO 4217 sein) und BR-CL-14 (Verkäufer-/Käuferland BT-40/BT-55 muss ISO 3166-1 alpha-2
  sein). `doc.Currency` ist reiner Freitext (`invoice.Currency`, ungeprüft bis hierher) —
  eine Rechnung mit "EURO" statt "EUR" wäre bisher unbeanstandet generiert worden.
  `isoCountryCode` (`generator_doc.go`) normalisiert bekannte Schreibweisen von DE/AT/CH,
  reicht aber jeden anderen exakt zweistelligen String unverändert durch (z. B. "FR", "XX")
  — genau diese Lücke schließt der neue Check. Whitelist bewusst DACH-scoped
  (EUR/CHF/USD, DE/AT/CH) mit `lean:`-Marker: CHF/USD sind bereits im DATEV-WKZ-Export
  vorgesehen (`datev/exporter.go:289` Kommentar "Hardcoding EUR mis-booked foreign-currency
  (CHF/USD) documents"), DE/AT/CH ist die Zielgruppe laut CLAUDE.md. Mengeneinheit (immer
  "C62"), Belegart (immer "380") und Zahlungsmittel (immer "58") werden NICHT geprüft —
  der Generator erzeugt sie fest verdrahtet, es gibt für keine der drei einen variablen
  Eingabepfad, den eine Prüfung abfangen könnte (per Grep in generator_cii.go/generator_ubl.go
  bestätigt: `unitCodePiece`/`invoiceTypeCodeCommercial`/`paymentMeansCodeSEPACredit` sind
  Konstanten, `models.LineItem` trägt kein Unit-Feld). USt-Kategorie (S/Z/E/AE) ebenso nicht
  geprüft — `taxCategoryFor` liefert ausschließlich diese vier Werte, kein Eingabepfad kann
  einen fünften erzeugen. Diese vier Streichungen stehen hier im Journal statt im Code, wie
  von der Unit verlangt.
  Rule-IDs BR-CL-04/BR-CL-14 per Websuche gegen die offizielle Peppol-BIS-3.0-Doku
  verifiziert (docs.peppol.eu/poacc/billing/3.0/rules/ubl-tc434/BR-CL-14/), nicht geraten.
- gate: build ok | vet ok | lint ok (0 issues) | test ok | migration n.a. | rls-smoke n.a.
- coverage: internal/biz/einvoice 81,9 % (CI-Stand) -> 82,3 % (eigene Messung, `go tool cover
  -func`, vor/nach identisch gemessen).
- mutations-probe: zwei separate Proben gegen eine `cp`-Sicherungskopie (nicht `git
  checkout`, siehe Iteration-3-Lehre). (1) `allowedCurrencies`-Check mit `if false && ...`
  stillgelegt -> `TestValidate_RejectsUnsupportedCurrency` rot ("expected a *ValidationError,
  got <nil>"). (2) beide `allowedCountries`-Checks (Seller + Buyer) mit `if false && ...`
  stillgelegt -> `TestValidate_RejectsUnsupportedCountry` rot, derselbe Fehler. Beide Male
  aus der Sicherungskopie zurückgeschrieben, `git diff --stat` danach identisch zum Stand vor
  der Probe (45 Zeilen, nur die beabsichtigte Änderung).
- verify vorgaenger: sauber — `aeb8725e` (letzter Commit vor dieser Iteration) ändert nur
  JOURNAL.md (44 Zeilen, Iteration-4-Eintrag), kein Codewechsel, keine der acht Fehlerklassen
  anwendbar.
- neue-units: keine
- offen: keine — Paket hat keine DB-Tests, DATABASE_URL-Gate daher nicht einschlägig für
  diese Unit. `go test ./internal/gateway/ -run TestOpenAPIRouteDrift` lief grün, obwohl keine
  Route angefasst wurde (Pflicht laut Prompt, unabhängig vom Anlass).

## Iteration 6 — feat-einvoice-vat-category-rules — done — 2026-08-22 23:59
- commit: 3917fe4d
- gebaut: zwei echte Funde in der bestehenden VAT-Kategorie-Prüfung, keiner davon
  im ursprünglich vermuteten Umfang ("Satz muss 0 sein + Befreiungsgrund Pflicht").
  (1) Der Buyer-VATID-Check für Reverse Charge trug die Regel-ID "BR-AE-03" —
  gegen die offizielle Peppol-BIS-3.0-Regeltextsammlung geprüft (zwei unabhängige
  Quellen, docs.peppol.eu und peppolvalidator.com) ist BR-AE-03 tatsächlich eine
  Regel für Document-Level-Allowances (BG-20), die dieses Produkt nie erzeugt.
  Die zutreffende Regel für eine Reverse-Charge-Rechnungszeile ist BR-AE-02 — sie
  bündelt Seller- UND Buyer-Identifikation in einer Anforderung, und
  `sellerTaxRuleFor` trägt für den Seller-Teil bereits korrekt "BR-AE-02". Der
  Buyer-Check war also nur der falsch beschriftete zweite Teil derselben Regel.
  Korrigiert in validation.go, im zugehörigen Test und in einem erläuternden
  Kommentar in `internal/biz/pdf/zugferd_test.go` (dort keine Assertion, nur
  Prosakommentar, aber derselbe Fehler). (2) Die Zero-Rated-Kategorie (Z) hatte
  KEINEN Test: `sellerTaxRuleFor` liefert seit jeher "BR-Z-02" für sie, aber die
  bestehende Tabelle `TestValidate_SellerTaxRuleFollowsTheCategory` erreicht nur
  Standard/Kleinunternehmer/Reverse-Charge über `inv.TaxMode` — Zero-Rated ist ein
  Zeilenmerkmal (Standardmodus, Zeile mit Satz 0), keine Dokumenteinstellung, und
  fällt durch die bestehende Testkonstruktion. Neuer eigener Test
  `TestValidate_ZeroRatedNeedsSellerIdentification` mit Positiv- und Negativfall.
  Die eigentlich befürchteten Regeln (BR-Z-05/09/10, BR-E-05/09/10, BR-AE-05/09/10:
  Satz muss 0 sein, Befreiungsgrund Pflicht bzw. verboten) sind NACH PRÜFUNG
  bereits vollständig strukturell garantiert: `taxCategoryFor`
  (`generator_doc.go:358`) ist die einzige Stelle, die Kategorie UND
  Befreiungsgrund setzt — beide Reverse-Charge/Kleinunternehmer-Zweige liefern
  immer einen nicht-leeren Grund, der Zero-Rated-Zweig immer einen leeren, und
  `omitempty` auf beiden XML-Feldern (CII wie UBL) sorgt dafür, dass ein leerer
  Grund nie gerendert wird. Der Satz ist für alle drei Kategorien vor dem Aufruf
  von `taxCategoryFor` bereits auf 0 erzwungen bzw. (bei Zero-Rated) die
  Bedingung, unter der die Kategorie überhaupt gewählt wird. Diese Garantien sind
  jetzt als Kommentar mit den zutreffenden Regel-IDs an `taxCategoryFor`
  dokumentiert, statt als tote Laufzeitprüfung gegen einen Zustand gebaut, den der
  Code selbst nie erzeugen kann — das wäre kein Fund, sondern nur Bestätigung.
  BR-G/BR-IC/BR-O (Ausfuhr, innergemeinschaftlich, außerhalb Anwendungsbereich)
  sind für dieses Produkt unerreichbar: `models.TaxMode` kennt nur
  standard/reverse_charge/kleinunternehmer (per Grep in `internal/models/finance.go`
  bestätigt) — lean-Marker mit Upgrade-Trigger an `taxCategoryFor`.
- gate: build ok | vet ok | lint ok (0 issues, beide Pakete) | test ok
  (internal/biz/einvoice, internal/biz/pdf) | migration n.a. | rls-smoke n.a.
- coverage: internal/biz/einvoice 82,3 % (Iteration-5-Messung, `go tool cover
  -func`) -> 82,5 % (eigene Messung, gleiches Kommando, vor/nach identisch).
- mutations-probe: zwei Proben gegen eine `cp`-Sicherungskopie (nicht `git
  checkout`). (1) `"BR-AE-02"` im Buyer-Check zu `"BR-AE-99"` verstümmelt ->
  `TestValidate_ReverseChargeNeedsBuyerVATID` rot ("does not contain BR-AE-02").
  (2) `sellerTaxRuleFor`s Zero-Rated-Zweig von `"BR-Z-02"` auf `"BR-S-02"`
  umgebogen -> `TestValidate_ZeroRatedNeedsSellerIdentification` rot ("does not
  contain BR-Z-02"). Beide Male aus der Sicherungskopie zurückgeschrieben,
  `diff` gegen die Kopie danach identisch (0 Zeilen Unterschied).
- verify vorgaenger: sauber — `6d5816c3` (feat-einvoice-codelist-validation)
  fügt nur eine begründete Whitelist plus Tests hinzu, keine der acht
  Fehlerklassen einschlägig (kein gRPC-Handler, kein Proto, keine Migration,
  kein neuer Guard, keine neue Tabelle, keine Route, kein Wire-Shape-Wechsel,
  kein ersetzter Guard-Key).
- neue-units: keine
- offen: (1) `internal/biz/pdf` hat keine DB-Tests, DATABASE_URL-Gate daher für
  diese Unit nicht einschlägig (wie schon Iteration 5 für einvoice). (2) BT-47
  (Buyer legal registration identifier), die per BR-AE-02 zulässige Alternative
  zur Buyer-VATID, ist im Datenmodell nicht abgebildet (`docParty` kennt nur
  `VATID`) — kein Fund im engeren Sinn, weil das Produkt nur die VATID-Variante
  je erzeugt, aber falls das je zur eigenen Unit wird: hier ansetzen.

## Iteration 7 — feat-einvoice-cardinality-rules — done — 2026-08-22 23:51
- commit: ca59a98f
- gebaut: alle 17 EN-16931-Kardinalitätsregeln BR-01 bis BR-17 sind jetzt
  entweder als Regel-ID im Fehlertext geprüft oder als "gilt durch Konstruktion"
  dokumentiert. Drei echte Funde ohne Regel-ID nachgerüstet: BR-02 (Rechnungs-
  nummer BT-1), BR-03 (Ausstellungsdatum BT-2), BR-16 (mindestens eine Position
  BG-25) — alle drei in `buildInvoiceDoc`s bestehenden Ablehnungen, jetzt in
  der Fehlermeldung enthalten und per Test belegt (`TestGenerateCII_Rejections`,
  `TestGenerateUBL_Rejections`). Die übrigen 14 Regeln waren KEIN Fund, sondern
  bereits garantiert — dokumentiert statt neu geprüft: BR-01 (Spezifikations-
  kennung) und BR-04 (Belegart "380") sind Konstanten, die beide Generatoren
  unbedingt schreiben (bereits durch `generator_ubl_test.go:93/99` und
  `generator_cii_test.go:36` bewiesen). BR-05 (Währung) ist durch den
  Default-Fallback in `buildInvoiceDoc` nie leer. BR-08/BR-10 (Postanschrift-
  Gruppen) folgen aus den bestehenden BR-09/BR-11-Ländercode-Prüfungen, weil
  Land das einzige von EN 16931 verlangte Element in beiden Gruppen ist.
  BR-12/13/14/15 (Summenfelder) folgen daraus, dass LineTotal/TaxTotal/
  GrossTotal unbedingte Structfelder sind, die beide Generatoren immer
  rendern. BR-17 (Zahlungsempfänger-Name) ist gegenstandslos: das Datenmodell
  kennt keinen vom Verkäufer abweichenden Payee — lean-Marker mit
  Upgrade-Trigger an `invoiceDoc`. BR-06/07/09/11 waren schon vor dieser
  Iteration mit Regel-ID versehen (keine Änderung nötig).
- gate: build ok | vet ok | lint ok (0 issues) | test ok
  (internal/biz/einvoice, TestOpenAPIRouteDrift trotz keiner Routenänderung
  pflichtgemäß gelaufen) | migration n.a. | rls-smoke n.a.
- coverage: internal/biz/einvoice 82,5 % (Iteration-6-Messung) -> 82,5 %
  (eigene Messung, `go tool cover -func`, vor/nach identisch) — unverändert,
  weil die Änderung nur Regel-IDs im bereits abgedeckten Fehlerpfad ergänzt
  und Kommentare, keine neue Verzweigung.
- mutations-probe: `BR-02` in der Rechnungsnummer-Fehlermeldung
  (`generator_doc.go`) zu `BR-99` verstümmelt (Kopie via `cp`, nicht
  `git checkout`) -> `TestGenerateCII_Rejections/missing_invoice_number` UND
  `TestGenerateUBL_Rejections/missing_invoice_number` beide rot ("does not
  contain BR-02"), alle übrigen Subtests weiterhin grün. Aus der Kopie
  zurückgeschrieben, `diff` gegen die Sicherungskopie danach identisch
  (0 Zeilen Unterschied).
- verify vorgaenger: sauber — `3917fe4d` (feat-einvoice-vat-category-rules)
  ändert nur `internal/biz/einvoice` (Regel-ID-Korrektur BR-AE-03->BR-AE-02,
  neuer Zero-Rated-Test, Kommentar in `taxCategoryFor`), keine der acht
  Fehlerklassen anwendbar: kein gRPC-Handler, kein Proto, keine Migration,
  kein neuer Guard, keine neue Tabelle, keine Route, kein Wire-Shape-Wechsel,
  kein ersetzter Guard-Key.
- neue-units: keine
- offen: `internal/biz/pdf` und `internal/biz/einvoice` haben keine DB-Tests,
  DATABASE_URL-Gate daher für diese Unit nicht einschlägig (wie schon
  Iterationen 5 und 6).

## Iteration 8 — feat-einvoice-allowance-charge-rules — done — 2026-08-22 23:56
- commit: e45d2d02
- gebaut: vierte und letzte EN-16931-Regelfamilie (A8). Erst geklärt, ob das
  Datenmodell Abschläge/Zuschläge kennt: Grep nach Discount/Allowance/Charge/
  Rabatt/Nachlass/Zuschlag über internal/models liefert 0 Treffer außer der
  unabhängigen Konstante TaxModeReverseCharge. Weder models.Invoice noch
  models.LineItem trägt ein solches Feld, buildInvoiceDoc hat also nichts zum
  Lesen und kein Generator emittiert eine AllowanceCharge-Gruppe. BR-31 bis
  BR-42 sind damit gegenstandslos — lean-Marker mit Upgrade-Trigger an
  validation.go statt einer Prüfung gegen nicht existierende Felder.
  BR-CO-9 (USt-ID-Präfix muss gültiger Ländercode sein) ist dagegen ein
  echter, unabhängiger Fund: weder Verkäufer- noch Käufer-VATID wurden bisher
  auf ihr Präfix geprüft. Neue Prüfung `hasAllowedVATCountryPrefix` in
  validateInvoiceDoc, für BT-31 (Seller.VATID) und BT-48 (Buyer.VATID) —
  reicht auf `allowedCountries` (DE/AT/CH) aus A5 zurück, keine zweite Liste
  angelegt (Kommentar hält fest, dass VAT-Präfixe nicht immer mit ISO 3166-1
  identisch sind — z. B. Griechenland "EL" — innerhalb der DACH-Whitelist
  aber schon).
- gate: build ok | vet ok | lint ok (0 issues) | test ok
  (internal/biz/einvoice, go test ./internal/gateway/ trotz keiner
  Routenänderung pflichtgemäß gelaufen) | migration n.a. | rls-smoke n.a.
- coverage: internal/biz/einvoice 82,5 % (eigene Messung vor der Änderung,
  go tool cover -func) -> 82,5 % (eigene Messung danach) — unverändert in
  der angezeigten Nachkommastelle: zwei neue Tests decken die zwei neuen
  if-Zweige exakt ab, keine Nettoveränderung im Prozentsatz des Pakets.
- mutations-probe: `hasAllowedVATCountryPrefix` in validation.go (Kopie via
  `cp`, nicht `git checkout`) auf `return true` verkürzt (Whitelist-Check
  entfernt) -> beide neuen Tests
  (TestValidate_RejectsSellerVATIdentifierWithBadCountryPrefix,
  TestValidate_RejectsBuyerVATIdentifierWithBadCountryPrefix) rot ("expected
  a *ValidationError, got nil"), alle übrigen Tests im Paket weiterhin grün.
  Aus der Kopie zurückgeschrieben, `diff` gegen die Sicherungskopie danach
  identisch (0 Zeilen Unterschied).
- verify vorgaenger: sauber — `ca59a98f` (feat-einvoice-cardinality-rules)
  fügt nur Regel-IDs zu bereits bestehenden Ablehnungen und Kommentare hinzu,
  keine der acht Fehlerklassen einschlägig (kein gRPC-Handler, kein Proto,
  keine Migration, kein neuer Guard, keine neue Tabelle, keine Route, kein
  Wire-Shape-Wechsel, kein ersetzter Guard-Key).
- neue-units: keine
- offen: Block A (A1-A8) ist damit vollständig abgeschlossen. Block B
  (Geld-Repositories gegen echtes SQL) ist die nächste offene Gruppe im
  Backlog. `internal/biz/einvoice` hat weiterhin keine DB-Tests,
  DATABASE_URL-Gate daher für diese Unit nicht einschlägig (wie schon
  Iterationen 5 bis 7).

## Iteration 9 — fix-booking-page-orphaned-after-owner-erasure — done — 2026-08-23 00:07
- commit: 4ae9605e
- gebaut: neuer `BookingPageErasureHandler` (`internal/security/gdpr/erasure.go`),
  registriert in `cmd/auth/main.go` direkt nach `CalendarErasureHandler`. Setzt
  `booking_pages.active = false` für jede noch aktive Buchungsseite, deren
  Kalender-Eigentümer (`calendars.owner_id`) der geloeschte Nutzer ist —
  unabhaengig von `calendar_type`, weil der Befund genau diesen Fall meint
  (ein gelöschter Mitarbeiter darf keine öffentliche Terminseite mehr
  bedienen). Fasst NUR `booking_pages.active` an, nie `calendars` oder
  `public_bookings` — deckungsgleich mit der bestehenden Entscheidung in
  `CalendarErasureHandler`, die genau diese Kalender bewusst nicht löscht.
  Neue Konstante `ErasureDeactivate ErasureAction = "deactivate"` fürs
  Preview/Execute-Actionfeld. `PreviewErasure` zählt dieselbe Menge, die
  `ExecuteErasure` anfasst (eigene, separate Zeile in der Modul-Liste des
  Service — `ModuleErasurePreview.ModuleName = "calendar_booking_pages"`),
  damit die Buchungsseite nicht mehr schweigend im Kalender-Zähler
  verschwindet: `CalendarErasureHandler` schließt sie ja per NOT-EXISTS aus
  der Zählung aus, zählt sie aber auch nirgendwo hin.
  `CalendarErasureHandler` selbst unveraendert — die NOT-EXISTS-Subquery, die
  den Ausschluss traegt, existierte schon.
- gate: build ok | vet ok | lint ok (0 issues) | test ok
  (internal/security/gdpr, internal/work/calendar, internal/server,
  internal/gateway — TestOpenAPIRouteDrift trotz keiner Routenänderung
  pflichtgemäß gelaufen) | migration n.a. (keine neue Tabelle/Spalte/Policy)
  | rls-smoke n.a. (keine neue Tabelle/Policy — `booking_pages` und
  `calendars` tragen ihre RLS-Policies seit Migration 000142 unverändert,
  der Handler läuft wie alle anderen Erasure-Handler über den
  tenant-gestempelten Pool-Kontext)
- coverage: internal/security/gdpr 70,9 % (Lauf-10-Referenzwert) -> eigene
  Vorher-Messung nicht separat gezogen (Paket hat sich seit Iteration 8
  nicht verändert) -> 72,1 % nach dieser Unit (go tool cover -func, ein
  Lauf, 159 Tests, 0 SKIP)
- mutations-probe: in `BookingPageErasureHandler.ExecuteErasure` das
  `active = false` aus dem SET entfernt (Kopie via `cp`, nicht
  `git checkout`) -> `TestBookingPageErasureHandler_Integration` rot an drei
  Assertions ("must be deactivated", "must find nothing left to deactivate"
  je zweimal) — `TestBookingPageErasureHandler_ExecuteErasure_DeadPool`
  weiterhin grün (Pool-Fehler unabhängig von der SET-Klausel). Aus der Kopie
  zurückgeschrieben, `diff` gegen die Sicherungskopie danach identisch
  (0 Zeilen Unterschied).
- verify vorgaenger: sauber — `e45d2d02` (feat-einvoice-allowance-charge-rules)
  ändert nur `internal/biz/einvoice/validation.go` und seinen Test, keine der
  acht Fehlerklassen einschlägig (kein gRPC-Handler, kein Proto, keine
  Migration, kein neuer Guard, keine neue Tabelle, keine Route, kein
  Wire-Shape-Wechsel, kein ersetzter Guard-Key).
- neue-units: keine
- offen: der Fall "mehrere Personenkalender, nur einer mit Buchungsseite"
  ist per Test belegt (`TestBookingPageErasureHandler_Integration`), ebenso
  eine bereits inaktive Buchungsseite (wird nicht doppelt gezählt) und ein
  zweiter Lauf (Idempotenz). Der bestehende
  `TestCalendarErasureHandler_ExecuteErasure_Integration` bleibt unverändert
  grün und deckt weiterhin den Kalender-Teil ab. Luke pruefen: ob eine
  Buchungsseite auf einem GETEILTEN Kalender (nicht `calendar_type =
  'personal'`), dessen Eigentuemer geloescht wird, tatsaechlich ebenfalls
  deaktiviert werden soll — diese Unit tut das (Filter ist nur `owner_id`,
  nicht `calendar_type`), weil `CalendarErasureHandler` einen solchen
  Kalender ohnehin nie loescht und die Buchungsseite sonst für immer aktiv
  bliebe.

## Iteration 10 — verify-biz-event-emitters-never-wired — done — 2026-08-23 00:35
- commit: 7ece4294
- gebaut: VERIFY-UNIT, verdrahtet nichts. Belegt per Grep + Read:
  `cmd/biz/main.go` ruft `SetEventEmitter` an KEINER Stelle auf (0 Treffer für
  `invoiceSvc`, `quoteSvc`, `lexwareSvc`) — alle drei laufen mit
  Konstruktor-Default (`invoice`/`quote`: `emitter == nil` -> `emitEvent` ist
  No-op; `lexware`: `noopEmitter{}`, `service.go:59-60`).
  Die UI-Frage ist eindeutig JA beantwortet: `GET /triggers`
  (`internal/gateway/route_automation.go:96` ->
  `AutomationGRPCServer.ListTriggerDefinitions` -> `TriggerRegistry.All()`,
  `internal/automation/trigger/registry.go:182-218`) liefert live zwei
  Finanz-Trigger mit deutschem UI-Label — "Rechnung versendet"
  (`biz.invoice.sent`) und "Angebot erstellt" (`biz.quote.created`) — beide
  hängen an `EmitBizEvent`, das wegen der fehlenden Verkabelung nie aufgerufen
  wird. Ein Kunde kann diese Trigger in einer Automation auswählen; sie feuern
  nie. Der dritte registrierte Finanz-Trigger, `biz.invoice.overdue`, ist NICHT
  betroffen — er ist `TimeBased: true` und wird per 5-Minuten-Poller direkt
  gegen `finance_invoices` aufgelöst (`due_postgres.go:41`), unabhängig vom
  Emitter.
  Nebenbefund aus den Notes bestätigt und behoben: die `HandleEvent`-Kommentare
  in `internal/biz/lexware/webhook_handler.go` (Zeile 154-166) behaupteten eine
  Emitter-Wirkung ("wrap in sysctx so any downstream emitter writes... pass
  WITH CHECK"), die es nicht gibt, weil `wh.emitter` in Produktion immer
  `noopEmitter{}` ist. Einzige Code-Änderung dieser Unit: ein `lean:`-Marker
  direkt über dem `Emit`-Aufruf, der das festhält und auf die neue
  BACKLOG-NEXT-Unit verweist. Kein Trigger in der Registry hört auf
  "lexware.*" — der Lexware-Fall bleibt folgenlos (deckt sich mit Befund 1 im
  Kopf von BACKLOG.yml).
- gate: build ok (`./internal/biz/lexware/...`, `./internal/automation/...`,
  `./internal/gateway/...`, `./cmd/biz/...`, `./cmd/gateway/...`) | vet ok
  | lint ok (0 issues, `internal/biz/lexware`) | test ok (93 PASS, 0 SKIP,
  `internal/biz/lexware`) | migration n.a. | rls-smoke n.a. (keine Tabelle/
  Policy berührt) | TestOpenAPIRouteDrift nicht gelaufen — keine Route
  angefasst, nur ein Kommentar in `internal/biz/lexware`
- coverage: internal/biz/lexware 74,4 % -> 74,4 % (unverändert, reiner
  Kommentar, kein Verhalten geändert) | internal/biz 63,7 % (Referenz aus
  coverage_start) -> n.a., diese Unit ist kein Coverage-Ziel
- mutations-probe: n.a. — VERIFY-UNIT ändert kein Verhalten, es gibt keine
  Fix-Logik zum Brechen
- verify vorgaenger: sauber — `4ae9605e` (fix-booking-page-orphaned-after-
  owner-erasure) ruft `h.pool` direkt auf wie alle anderen Erasure-Handler
  (kein Gateway-Handler, kein gRPC-Layer-Bypass einschlägig), keine Stubs,
  kein Proto berührt, kein neuer `RequirePermission`-Guard, keine neue Tabelle,
  keine neue Route, kein Wire-Shape-Wechsel, kein ersetzter Guard-Key. Eigene
  Journal-Notiz der Vor-Iteration bestätigt dieselbe Prüfung. Docs-Commit
  `364383c3` ändert nur eine Zeile in JOURNAL.md, unauffällig.
- neue-units: keine in BACKLOG.yml. Eine Folge-Unit
  `wire-biz-event-emitters-for-finance-triggers` mit `status: blocked` und
  `blocked_reason` in `BACKLOG-NEXT.yml` angelegt (Produktions-Verhaltens-
  änderung, braucht Lukes Entscheidung (a) verdrahten oder (b) Trigger aus der
  Registry entfernen) — der bereits vorbereitete Platzhalter-Abschnitt "EVENT-
  EMITTER IN cmd/biz/main.go" in derselben Datei wurde dabei nicht entfernt,
  weil er den Hintergrund erklärt; die neue Unit ist der ausführbare Nachfolger.
- offen: Luke muss `wire-biz-event-emitters-for-finance-triggers` entscheiden
  (a) oder (b), siehe BACKLOG-NEXT.yml. Bis dahin bleiben "Rechnung versendet"
  und "Angebot erstellt" wählbare, aber tote Automations-Trigger in der
  Produktions-UI — das ist ein bestehender, nicht durch diese Unit
  verschlechterter Zustand.

## Iteration 11 — fix-rls-smoke-hr-work-time-entries-with-data — done — 2026-08-23 00:14
- commit: -
- gebaut: NICHTS NEU GEBAUT — Prämisse widerlegt, done_when war bereits vollständig
  erfüllt. `TestTenantIsolation_HR_Standard/hr_work_time_entries`
  (`backend/internal/biz/hr/tenant_isolation_phase2_test.go:81-101`) existiert
  bereits seit `2863fbb3` (2026-05-11, welle3) bzw. `0b30a62c` (2026-06-05,
  Reparatur für den ersten echten DB-Lauf) — also lange vor dem Lauf-10-Fund
  UND vor dieser Lauf-11-Staging. Der Test seedet `hr_work_time_entries` selbst
  (eigener User als `employee_id`, echter `clock_in`-Wert) über
  `testutil.SeedRow` mit System-Kontext, liest danach als `kmuhub_app` einmal
  unter TenantA-Kontext (erwartet 1 Zeile) und einmal unter TenantB-Kontext
  (erwartet 0 Zeilen) — exakt die im Backlog geforderte Form, keine
  Zwei-Nullen-Messung. `TenantA`/`TenantB` sind laut `testutil/rls.go:137-143`
  bewusst stabile, nie gelöschte Fixture-UUIDs (Konvention über den ganzen
  Testbaum) — "Tenants wieder aufräumen" heißt hier bestehend zu Recht nur
  Zeilen (per `defer testutil.CleanupRow`), nicht die Fixture-Tenants selbst.
  Der Lauf-10-Befund ("beide Zählungen 0") bezog sich erkennbar auf einen
  Ad-hoc-psql-Smoke gegen die damals leere Tabelle, nicht auf diesen
  seedenden Go-Test — der lief zu keinem Zeitpunkt gegen leere Daten.
  Mutations-Probe durchgeführt: `ALTER TABLE hr_work_time_entries DISABLE ROW
  LEVEL SECURITY` per `docker exec docker-postgres-1 psql -U kmuhub -d kmuhub`
  → `go test -run TestTenantIsolation_HR_Standard` wird rot exakt am
  `hr_work_time_entries`-Subtest ("expected 0 row(s), got 1"), alle anderen
  Subtests bleiben grün. Danach `ENABLE ROW LEVEL SECURITY` zurückgedreht,
  Test wieder grün. Kein Diff im Repo — die Probe lief ausschließlich gegen
  den laufenden DB-Zustand, keine Migration angefasst.
- gate: build n.a. (kein Code geändert) | vet n.a. | lint n.a. | test ok
  (`go test -count=1 ./internal/biz/hr/ -run TestTenantIsolation_HR_Standard`,
  0 SKIP, DATABASE_URL gegen `kmuhub_app`) | migration n.a. | rls-smoke ok
  (Mutations-Probe siehe oben, das IST der Smoke)
- coverage: internal/biz/hr n.a. — kein Verhalten und keine Zeile geändert,
  reine Verifikation eines bestehenden Tests
- mutations-probe: RLS testweise deaktiviert (siehe oben) → Subtest
  `hr_work_time_entries` rot ("expected 0 row(s), got 1") → RLS wieder
  aktiviert → grün. Kein Diff übrig.
- verify vorgaenger: sauber — `7ece4294` (verify-biz-event-emitters-never-
  wired) ändert nur einen Kommentar in `webhook_handler.go` (7 Zeilen,
  `lean:`-Marker), keine der acht Fehlerklassen einschlägig: kein neuer
  Gateway-Handler, kein Stub, kein `.proto` angefasst, kein neuer Guard, keine
  neue Tabelle, kein Wire-Shape, keine neue Route, kein ersetzter Guard-Key.
- neue-units: keine
- offen: keine. Diese Unit ist inhaltlich erledigt, ohne dass Code geändert
  wurde — der Commit dieser Iteration ist ein reiner Doku-Commit
  (BACKLOG.yml-Status + dieser Journal-Eintrag).

## Iteration 12 — verify-dunning-calculate-interest-unreachable — done — 2026-08-23 00:17
- commit: 7941c1b7
- gebaut: VERIFY-UNIT, keine Verzugszins-Verdrahtung. Aufrufgraph von
  `dunning.CalculateInterest` (`service.go:415`) über `internal/`, `cmd/` und
  `proto/biz/v1/biz.proto` geprüft: null Aufrufer außer den fünf Tests in
  `service_test.go`. Die Backlog-Prämisse war aber nur zur Hälfte richtig:
  `models.DunningConfig` trägt tatsächlich keinen Basiszinssatz — ABER
  `models.CompanySettings.Basiszinssatz` (proto Feld 19, `biz.proto:374`)
  existiert bereits und ist voll verdrahtet: `route_biz.go:353/396`
  (`HandleUpdateCompanySettings`) -> `biz_grpc.go:229-234` -> persistiert via
  `quote.PostgresCompanySettingsRepo` (round-trip-getestet in
  `postgres_repository_db_test.go:351-418`, inkl. negativer Werte). Der
  Parameter KANN also befüllt werden — die Lücke ist ausschließlich, dass ihn
  niemand liest. Zweiter Fund: der Kommentar an der Record-Erstellung
  (`service.go:257`, vor dieser Iteration) behauptete "Interest is calculated
  separately at send time or display time" — das ist falsch, weder
  `sendAndNotify`/`SendDunningNotice` (Zeilen 380-403) noch die
  gRPC-Antwort (`biz_grpc.go:1640`, gibt nur den gespeicherten Wert zurück)
  berechnen je etwas. Dritter Fund: kein B2B/B2C-Flag auf `models.Invoice`
  oder Customer — die Wire-Entscheidung braucht also zusätzlich noch dieses
  Datenmodell-Stück, nicht nur den Basiszinssatz.
  ENTSCHEIDUNG: lean-Marker statt Entfernung. Fünf bestehende Tests
  (B2C/B2B/NotYetOverdue/ZeroDays/LargeAmount) beweisen korrekte Arithmetik;
  der fehlende Teil ist reine Verkabelung plus ein Datenmodell-Feld, kein
  räumbarer Ballast. Zwei Code-Änderungen, beide reine Kommentare:
  (1) `service.go:257` — irreführenden Kommentar durch `lean:`-Marker mit
  korrektem Sachstand ersetzt (Basiszinssatz verfügbar, aber nicht gelesen).
  (2) `service.go:415-431` (Funktionskopf `CalculateInterest`) — `lean:`-Marker
  mit Upgrade-Trigger (B2B/B2C-Flag + Basiszinssatz lesen) und die beiden
  fachlichen Mängel aus dem Backlog als Kommentar direkt an der Stelle
  festgehalten: fixer 365-Divisor (Schaltjahr) und die halbjährliche
  Basiszinssatz-Änderung nach § 247 BGB (Zeitraum müsste beim Satzwechsel
  gesplittet werden, nicht mit einem Satz über die ganze Spanne gerechnet).
  Kein Verzugszins-Pfad wurde verdrahtet.
- gate: build ok (`./internal/biz/dunning/...`) | vet ok | lint ok (0 issues)
  | test ok (`go test -count=1 ./internal/biz/dunning/`, alle PASS, 0 SKIP)
  | migration n.a. (keine Tabelle/Spalte angefasst) | rls-smoke n.a. (keine
  Policy berührt) | TestOpenAPIRouteDrift nicht gelaufen — keine Route
  angefasst
- coverage: internal/biz/dunning 65,1 % -> 65,1 % (unverändert; reine
  Kommentaränderung, keine ausführbare Zeile geändert). Weicht vom
  `coverage_start` (61,8 %, CI-Stand 32570176303) ab — vermutlich hat eine
  frühere Iteration dieses Laufs das Paket bereits angefasst; eigene Messung
  vor und nach dem Diff ist identisch (65,1 % beidseitig), also kein
  Widerspruch zu "kein Verhalten geändert"
- mutations-probe: n.a. — VERIFY-UNIT, reine Kommentaränderung, keine
  Fix-Logik zum Brechen
- verify vorgaenger: sauber — `22dba25c` (docs, Iteration 11) ändert nur
  `BACKLOG.yml` und `JOURNAL.md`, keine der acht Fehlerklassen einschlägig
- neue-units: keine. Kein Wiring, weil (a) Produktions-Verhaltensänderung
  außerhalb einer VERIFY-Unit und (b) zwei fehlende Voraussetzungen (B2B/B2C-
  Flag im Datenmodell, tatsächliches Lesen von Basiszinssatz in
  DetectAndCreateDunnings) — das ist eine Produktentscheidung, keine
  Coverage- oder Fix-Unit. Falls gewünscht, gehört sie mit `blocked_reason`
  in BACKLOG-NEXT.yml wie schon `wire-biz-event-emitters-for-finance-triggers`
  aus Iteration 10 — hier bewusst nicht angelegt, weil die Entscheidung
  "Verzugszinsen anbieten oder nicht" eine reine Produktfrage ohne
  verifizierten Schaden ist (anders als die toten Automations-Trigger, die
  in der UI bereits sichtbar sind)
- offen: Luke kann entscheiden, ob Verzugszinsen als Feature überhaupt
  gewünscht sind (Wettbewerbsdifferenzierung vs. Komplexität); die
  Voraussetzungen dafür stehen jetzt präzise im Code-Kommentar

## Iteration 13 — fix-invoice-pdf-ustg14-mandatory-fields — done — 2026-08-23 00:20
- commit: e4f78e4b
- gebaut: `templates.go`/`generator.go` gegen § 14 Abs. 4 UStG abgeglichen (Nr. 1-8, die
  vom Backlog vorgegebene Liste). Sieben von acht Punkten waren bereits vollständig
  abgedeckt (Name/Anschrift beider Parteien, Steuernummer/USt-IdNr., Ausstellungsdatum,
  Rechnungsnummer, Menge/Art der Leistung, Entgelt je Steuersatz, Steuersatz+Steuerbetrag
  bzw. Befreiungshinweis). DER FUND war Nr. 6 (Leistungszeitpunkt), wie im Backlog
  vorausgesagt: `GenerateInvoicePDF` zeigte die Zeile "Lieferdatum: ..." nur, wenn
  `invoice.DeliveryDate != nil` — bei nil (ein regulärer, häufiger Fall laut
  `internal/biz/invoice/service.go:434`, das Feld ist rein optional gesetzt) fehlte die
  Pflichtangabe komplett, statt auf das Rechnungsdatum zurückzufallen. Genau diese
  Fallback-Konvention (BT-72) existiert bereits in `einvoice/generator_doc.go:163` und
  ist dort per Test belegt (`generator_ubl_test.go:365`) — auf das PDF übertragen.
  FIX: `deliveryDate := invoice.InvoiceDate; if invoice.DeliveryDate != nil && !IsZero()
  { deliveryDate = *invoice.DeliveryDate }`, Zeile wird jetzt IMMER gerendert
  (`generator.go:180-194`). Begründung im Code: Abschn. 14.5 Abs. 17 UStAE erlaubt das
  Rechnungsdatum als Leistungsdatum, wenn beide zusammenfallen — keine separate
  Kennzeichnung nötig, das ist konsistent mit der bestehenden `einvoice`-Konvention.
  ZWEITENS: `GenerateInvoicePDF` in Build (`buildInvoiceDoc`, gibt `core.Maroto` statt
  `[]byte` zurück) und Generate (`generate(m)`) aufgeteilt — reiner Strukturschnitt, kein
  Verhaltensunterschied —, damit Tests die maroto-Dokumentstruktur über `m.GetStructure()`
  prüfen können, statt PDF-Bytes zu vergleichen (explizit verboten laut `done_when`).
  Neue Testdatei `internal/biz/pdf/invoice_ustg14_test.go`: ein Walker
  (`renderedTexts`) läuft den `node.Node[core.Structure]`-Baum ab und sammelt alle
  `Type == "text"`-Werte — das ist die "extrahierte Textinhalt"-Ebene aus der
  Backlog-Notiz, kein Golden-Vergleich. Sechs Tests: alle acht Pflichtangaben an einer
  vollständigen Rechnung, der Lieferdatum-Fallback (Regressionstest für den Fix), ein
  explizit gesetztes Lieferdatum (beweist, dass der Fallback ein echtes Datum nicht
  überschreibt), der Kleinunternehmer-Befreiungshinweis (Nr. 8, Befreiungsfall), und die
  Ablehnung bei unvollständigen Company-Settings (Nr. 1/2 vor dem Rendern erzwungen).
  NICHT geprüft (bewusst außerhalb der § 14 Abs. 4-Liste des Backlogs): die USt-IdNr.
  des Käufers wird im PDF nirgends gedruckt, auch nicht bei Reverse-Charge-Rechnungen
  (nur der Text "Steuerschuldnerschaft ... Abschnitt 13b UStG" erscheint). Das beträfe
  eher § 14a UStG (nicht § 14 Abs. 4, die Grundlage dieser Unit) und ist NICHT verifiziert
  als Verstoß — kein Fix, kein Fund, nur eine offene Beobachtung für einen künftigen Scan.
  `CreditNote` (Gutschrift) hat gar kein `DeliveryDate`-Feld im Datenmodell; nicht
  angefasst, da außerhalb des Titels dieser Unit ("invoice-pdf") und ein Datenmodell-Feld
  bräuchte eine Migration — siehe offen:.
- gate: build ok (`./internal/biz/pdf/... ./internal/biz/invoice/... ./internal/biz/einvoice/...`)
  | vet ok | lint ok (0 issues) | test ok (`go test -count=1 ./internal/biz/pdf/`, alle
  PASS, 0 SKIP) | migration n.a. (kein Schema angefasst) | rls-smoke n.a. (keine
  Policy/Tabelle) | TestOpenAPIRouteDrift nicht gelaufen — keine Route angefasst
- coverage: internal/biz/pdf 48,4 % -> 51,6 % (eigene Messung vor/nach dem Diff via
  `git stash` auf genau `generator.go` + `go.mod`, nicht die restlichen Iterationen
  dieses Laufs — `coverage_start` im Backlog war ein Platzhalter "im CI-Log nachschlagen",
  daher hier direkt gemessen statt übernommen)
- mutations-probe: Fallback-Logik testweise auf den alten Zustand zurückgesetzt
  (`if invoice.DeliveryDate != nil { ... }` ohne Fallback) -> `TestInvoicePDF_
  DeliveryDateFallsBackToInvoiceDate` rot ("expected ... to contain Lieferdatum:
  01.08.2026, got [...ohne Lieferdatum-Zeile...]") -> zurückgedreht (`cp` aus Backup
  vor der Probe) -> `go test ./internal/biz/pdf/` wieder grün, `git diff --stat` zeigt
  nur den beabsichtigten Diff (24 insertions, 7 deletions in generator.go)
- verify vorgaenger: sauber — `7941c1b7` (Iteration 12) ändert nur Kommentare in
  `service.go` (siehe `git show`), keine der acht Fehlerklassen einschlägig
- neue-units: keine
- offen: (a) Buyer-USt-IdNr. fehlt im gedruckten PDF bei Reverse-Charge — nicht
  verifiziert, ob § 14a UStG das für § 13b-Fälle verlangt; als Beobachtung für einen
  künftigen Scan, nicht als Fund angelegt. (b) `CreditNote` trägt kein `DeliveryDate`-Feld
  — falls ein künftiger Fund das für Gutschriften braucht, ist das eine Migration, kein
  reiner Code-Fix. (c) `go-tree` wurde durch den neuen Test zur direkten Dependency in
  `go.mod` (`go mod tidy` mitcommittet, keine neue externe Dependency — das Modul war
  bereits transitiv über maroto vorhanden).

## Iteration 14 — cov-invoice-pdf-generator-and-templates — done — 2026-08-23 00:28
- commit: c0d36487
- gebaut: neue Testdatei `internal/biz/pdf/generator_coverage_test.go`, sechs Tests,
  kein Produktionscode geändert (reine Coverage-Unit, kein Fund erzwang einen Fix).
  (1) `ValidateCompanySettingsForPDF` hatte laut `zugferd_test.go` nur "alles fehlt" und
  "nur city fehlt" als Fehlerpfade — `TestValidateCompanySettingsForPDF_
  MissingIndividualFields` deckt jetzt fehlenden Namen, fehlende Straße, fehlende PLZ und
  einen nur-Leerzeichen-Namen (TrimSpace-Pfad) einzeln ab.
  `TestValidateCompanySettingsForPDF_AllMissingFieldsAreListed` belegt, dass die
  Fehlermeldung bei leeren Settings alle fünf fehlenden Felder nennt, nicht nur das erste.
  (2) `TestInvoicePDF_UmlautsSurviveRenderPath` belegt, dass Umlaute in Firmenname,
  Straße, Stadt, Kundenname und -anschrift ("Müller & Söhne GmbH", "Weiß & Groß KG", "Köln")
  unverändert bis zur extrahierten Textebene durchlaufen. Vorab per Grep geprüft: kein
  `charmap`/`iconv`/`x/text/encoding` im Paket — der Renderpfad wandelt vor `Generate()`
  keine Zeichenkodierung um, der Test friert diese Garantie auf Struktur-Ebene ein
  (Font-Glyph-Rendering selbst bleibt außerhalb der Testgrenze, wie vom `done_when`
  verboten: kein binärer PDF-Vergleich). (3) Reverse-Charge- und Kleinunternehmer-Hinweis
  GEPRÜFT, KEIN FUND: `buildTotalsSection`s Switch vergleicht `taxMode` (String) gegen
  `models.TaxModeReverseCharge`/`models.TaxModeKleinunternehmer` — beide Konstanten tragen
  denselben Stringwert wie `tax.ModeReverseCharge`/`tax.ModeKleinunternehmer`
  ("reverse_charge"/"kleinunternehmer", per Grep in `internal/models/finance.go` und
  `internal/biz/tax/calculator.go` verglichen), und `tax.RequiresReverseChargeNote`/
  `RequiresKleinunternehmerNote` verlangen exakt diese beiden Hinweise. Kleinunternehmer
  war bereits von Iteration 13 getestet (`invoice_ustg14_test.go`); neu ist
  `TestInvoicePDF_ReverseChargeHintPresent` (Positivfall) und
  `TestInvoicePDF_StandardModeShowsNoExemptionHint` (Negativfall — Gegenprobe, dass eine
  reguläre Rechnung keinen der beiden Hinweise zeigt).
- gate: build ok (`./internal/biz/pdf/...`) | vet ok | lint ok (0 issues) | test ok
  (`go test -count=1 ./internal/biz/pdf/`, alle PASS, 0 SKIP) | migration n.a. (kein
  Schema angefasst) | rls-smoke n.a. (keine Tabelle/Policy) |
  `go test ./internal/gateway/ -run TestOpenAPIRouteDrift` grün, obwohl keine Route
  angefasst wurde
- coverage: internal/biz/pdf 51,6 % -> 52,0 % (eigene Messung, `go tool cover -func`;
  Vorher-Wert per `mv` der neuen Testdatei aus dem Paket und erneutem Lauf reproduziert,
  identisch zur Iteration-13-Messung — kein Widerspruch zum `coverage_start:` der Unit,
  der nur ein Platzhalter "im CI-Log nachschlagen" war)
- mutations-probe: zwei Läufe, beide gegen eine `cp`-Sicherungskopie (nicht
  `git checkout`, siehe Iteration-3-Lehre), beide zurückgeschrieben, `git diff --stat`
  gegen `templates.go`/`generator.go` danach leer (0 Zeilen Unterschied zu HEAD).
  (a) `if strings.TrimSpace(s.Name) == ""` in `ValidateCompanySettingsForPDF` mit
  `if false && ...` stillgelegt -> `TestValidateCompanySettingsForPDF_
  MissingIndividualFields/missing_name` UND `/whitespace-only_name` beide rot ("expected
  an error ..., got nil"). (b) `case models.TaxModeReverseCharge:` in `buildTotalsSection`
  auf einen unerreichbaren String-Wert umgebogen ->
  `TestInvoicePDF_ReverseChargeHintPresent` rot ("expected rendered invoice to contain
  \"Steuerschuldnerschaft\"").
- verify vorgaenger: sauber — `e4f78e4b` (Iteration 13) ändert nur
  `internal/biz/pdf/generator.go` (Lieferdatum-Fallback, Split in `buildInvoiceDoc`/
  `generate`), `go.mod` (go-tree von indirect zu direct verschoben, keine neue
  Dependency) und die neue Testdatei `invoice_ustg14_test.go`. Keine der acht
  Fehlerklassen einschlägig: kein gRPC-Handler, kein Stub/TODO, kein `.proto`, keine
  Migration, kein neuer Guard, keine neue Tabelle, keine Route, kein Wire-Shape-Wechsel,
  kein ersetzter Guard-Key. `93982b11` ändert nur JOURNAL.md/BACKLOG.yml.
- neue-units: keine
- offen: (a) `internal/biz/pdf` hat keine DB-Tests, DATABASE_URL-Gate daher für diese
  Unit nicht einschlägig (wie schon Iterationen 5-8, 13). (b) Mehrseitigkeit wurde NICHT
  getestet — laut Notes nur, wenn der Generator sie kennt; `newMaroto`/`buildInvoiceDoc`
  haben keine explizite Seitenumbruch-Logik zu prüfen (maroto paginiert intern), ein
  Test dafür wäre gegen Bibliotheksverhalten, nicht gegen eigenen Code. (c) "Rechnung
  ohne Positionen" (leere `LineItems`) wurde NICHT separat getestet — `parseLineItems`
  gibt bei leerem Raw `nil, nil` zurück, `buildInvoiceDoc` rendert dann nur den
  Tabellenkopf ohne Zeilen; kein Absturz, aber auch kein Fund, da das Backlog dafür kein
  eigenes done_when nennt und die Frage, ob eine Rechnung ohne Positionen überhaupt
  fachlich zulässig sein soll, eine Produktentscheidung ist, keine Coverage-Frage.

## Iteration 15 — fix-idempotency-inflight-409-not-in-openapi — done — 2026-08-23 00:34
- commit: (wird nach diesem Eintrag erstellt)
- gebaut: `api/openapi.yaml` erweitert, kein Produktionscode angefasst (reine Doku-Unit,
  Middleware-Verhalten unveraendert). (1) Zwei neue wiederverwendbare Response-Components
  `IdempotencyInFlight` (409, mit Retry-After-Header-Doku) und `IdempotencyKeyRequired` (400,
  nur bei IDEMPOTENCY_MODE=hard) unter `components/responses`. (2) Globaler Vertrag in
  `info.description`: was die Middleware tut (409+Retry-After bei In-Flight-Retry, Replay via
  gecachter Antwort, 422 bei Payload-Aenderung, 400 nur HardMode) UND die vollstaendig
  verifizierte Ausschlussliste (was NICHT geschuetzt ist: `/api/v1/public/*`, `/api/v1/guest/*`,
  `/api/v1/caldav/*` + `/api/v1/admin/caldav/*` — `route_caldav.go` ist die einzige Route-Datei,
  die den uebergebenen Middleware-Parameter komplett ignoriert und eine eigene nackte
  `authMiddleware` verwendet —, WOPI/CalDAV/CardDAV-Protokoll, `/reset-password`,
  `/auth/login|refresh|2fa`). (3) 409 konkret dokumentiert auf sechs Finanz-/Dialer-/
  Schichten-Buchungsrouten (die im Scope explizit genannte Prioritaet): POST
  /api/v1/finance/invoices, POST /api/v1/finance/invoices/{id}/payments (beide zusaetzlich mit
  400 IdempotencyKeyRequired, da vorher GAR KEIN Error-Response dokumentiert war), POST
  /api/v1/dialer/calls/{id}/outcome, PUT /api/v1/dialer/calls/{id}/notes, POST
  /api/v1/dialer/calls/{id}/complete, POST /api/v1/schichten/swap-requests (deklariert bereits
  `Idempotency-Key` als required Header-Parameter, hatte aber kein 409).
- gate: build ok (`./internal/gateway/... ./cmd/gateway/...`) | vet ok | lint ok (0 issues) |
  test ok (`go test -count=1 ./internal/gateway/`, 0 SKIP) |
  `go test -run 'TestOpenAPIRouteDrift|TestOpenAPISpecDrift' -v`: 836 registrierte gegen 838
  dokumentierte Pfade, beide gruen | `npx @apidevtools/swagger-cli validate api/openapi.yaml`:
  "api/openapi.yaml is valid" | migration n.a. | rls-smoke n.a.
- coverage: n.a. (Doku-Unit, keine Coverage-relevante Code-Aenderung)
- mutations-probe: n.a. — reine OpenAPI-Spec-Aenderung, keine Produktionslogik zum Brechen.
  TestOpenAPIRouteDrift/-SpecDrift liefen vor UND nach der Aenderung identisch gruen (838 statt
  837 dokumentierte Pfade — die Differenz ist ausschliesslich die neue Beschreibung, keine neue
  Pfad-Zeile), das belegt strukturell, dass nichts an der Routenmenge selbst verschoben wurde.
- verify vorgaenger: sauber — `c0d36487` (Iteration 14) fuegt ausschliesslich eine neue Testdatei
  `internal/biz/pdf/generator_coverage_test.go` hinzu (siehe `git show --stat`), keine der acht
  Fehlerklassen einschlaegig: kein gRPC-Handler, kein Stub/TODO, kein `.proto`, keine Migration,
  kein neuer Guard, keine neue Tabelle, keine Route, kein Wire-Shape-Wechsel, kein ersetzter
  Guard-Key. `377ad9ef` aendert nur JOURNAL.md/BACKLOG.yml.
- neue-units: fix-idempotency-409-rollout-remaining-routes (Fortsetzung — 739 mutierende
  Operationen insgesamt, diese Iteration deckt 6 explizit im Scope genannte ab, der Rest ist zu
  gross fuer einen Commit und braucht eine sichere Skript-Strategie plus Konflikt-Behandlung mit
  ~20 bereits bestehenden 409-Eintraegen anderer Bedeutung), fix-hr-time-entries-manual-post-not-
  in-openapi (echter Fund: POST /api/v1/hr/time/entries ist komplett undokumentiert,
  TestOpenAPIRouteDrift prueft nachweislich nur Pfad- nicht Methodenebene und faengt das nicht)
- offen: (a) Die Ausschlussliste in `info.description` wurde durch Code-Lesen verifiziert
  (`cmd/gateway/main.go` Registrar-Liste + jede `RegisterRoutes`-Signatur in
  `internal/gateway/route_*.go` auf Middleware-Parameter-Nutzung geprueft), nicht durch einen
  automatisierten Test — bei einer zukuenftigen Aenderung an der Registrar-Verdrahtung kann die
  Doku veralten, ohne dass CI das merkt. (b) Die zwei neuen Response-Components sind bislang nur
  auf 6 von 739 mutierenden Operationen angewendet — vollstaendige Rollout-Unit ist angelegt.
  (c) `route_caldav.go` registriert `/api/v1/caldav/*` und `/api/v1/admin/caldav/*` mit einer
  eigenen `authMiddleware` OHNE Idempotency-Schutz, obwohl es POST/PUT/DELETE-Operationen sind
  (App-Passwoerter erstellen/widerrufen) — das ist eine bewusste Entscheidung im bestehenden Code
  (nicht neu durch diese Iteration), aber nicht als eigener Fund geprueft, ob das gewollt ist;
  nur als Beobachtung im Journal, keine Unit angelegt, da unklar ob das ein Fehler oder Absicht
  ist und das Backlog explizit vor unbelegten Funden warnt.

## Iteration 16 — verify-invoice-number-gap-detection-gobd — done — 2026-08-23 00:45
- commit: 2c602e67
- gebaut: VERIFY-UNIT, ein Test ergaenzt. (1) Lueckenerkennung existiert bereits vollstaendig:
  `Service.GetJournalSummary` (`service_gobd.go:137`) vergleicht `seq.CurrentNumber`
  (`numberSeqRepo.GetSequenceInfo`, jahresgescopte Zeile in
  `quote.PostgresNumberSequenceRepo`) gegen `repo.CountByFiscalYear` (nicht-Entwuerfe mit
  gesetzter Nummer im Jahr) und liefert `GapsDetected = max(CurrentNumber - Count, 0)` — das ist
  der bestehende, per gRPC (`biz_grpc.go:2291`) exponierte Bericht, den ein Pruefer sehen kann.
  Keine neue Tabelle, kein neuer Endpunkt noetig. (2) Der Fall "versendete Rechnung storniert"
  ist bereits korrekt behandelt UND bereits getestet: `Service.Cancel` (`service.go:669-733`)
  loescht eine versendete/ueberfaellige Rechnung NIE physisch (kein `Delete` existiert im
  gesamten `invoice`-Paket — Grep bestaetigt 0 Treffer), sondern kehrt sie ueber eine
  Stornorechnung (`stornoCreator.StornoInvoice`) um und setzt nur `status = cancelled`; die
  `invoice_number`-Spalte bleibt unangetastet. `CountByFiscalYear` filtert `status != 'draft'`,
  zaehlt eine stornierte Rechnung also weiter mit — belegt durch den bereits bestehenden Test
  `TestService_GetJournalSummary_CountsCancelledInvoices`. (3) Die einzige echte Luecke im
  Beleg war die Fiskaljahresgrenze aus `done_when`: kein Test hatte bislang zwei Jahre
  gleichzeitig im Repo. Neuer Test `TestService_GetJournalSummary_FiscalYearBoundaryNoFalseGap`
  (`service_test.go`) seedet 5 Rechnungen in 2025 (eigene, voll ausgeschoepfte Sequenz) und 3 in
  2026, setzt die Mock-Sequenz auf `CurrentNumber=3` fuer 2026 und belegt
  `TotalInvoicesIssued=3`, `GapsDetected=0` fuer 2026 — die 5 Rechnungen aus dem Vorjahr duerfen
  weder mitgezaehlt werden noch eine Phantom-Luecke erzeugen.
- gate: build ok (`./internal/biz/invoice/...`) | vet ok | lint ok (0 issues) |
  test ok (`go test -count=1 -v ./internal/biz/invoice/...`, 0 SKIP, 0 FAIL) | migration n.a. |
  rls-smoke n.a. (keine Tabellen-/Policy-Aenderung)
- coverage: internal/biz/invoice 34,8 % -> 34,8 % (Vorgabe `coverage_start` bestaetigt; die
  neue Zeile deckt denselben bereits vollstaendig getesteten `GetJournalSummary`-Pfad ab wie
  der bestehende Cancelled-Test, keine neuen Zeilen erreicht — erwartet fuer eine Verify-Unit,
  die eine Regressionsluecke schliesst statt neuen Code zu bauen)
- mutations-probe: Jahresfilter `inv.InvoiceDate.Year() == year` im `MockRepository.
  CountByFiscalYear` (Testdatei, service_test.go) entfernt -> neuer Test wird rot
  (TotalInvoicesIssued erwartet 3, tatsaechlich 8 — die 5 Vorjahresrechnungen leaken durch),
  bestehender Cancelled-Test bleibt gruen (kein zweites Jahr im Setup). Zurueckgedreht, Diff
  sauber (`git diff --stat` zeigt nur die Netto-Testergaenzung, 52 Zeilen).
- verify vorgaenger: sauber — `ff353639` (Iteration 15) aendert laut `git show --stat` und Diff
  ausschliesslich `backend/api/openapi.yaml` (neue Response-Components + Beschreibungstext +
  409/400 auf sechs benannten Pfaden) sowie JOURNAL.md/BACKLOG.yml. Keine der acht
  Fehlerklassen einschlaegig: kein gRPC-Handler beruehrt, kein Stub/TODO, kein `.proto`, keine
  Migration, kein neuer Guard, keine neue Tabelle, keine neue Route (nur Response-Doku auf
  bestehenden Pfaden), kein Wire-Shape-Wechsel, kein ersetzter Guard-Key.
- neue-units: keine — die Unit war Verify-first, die einzige gefundene Luecke (Fiskaljahresgrenze)
  war klein genug fuer dieselbe Iteration (ein Testfall, keine neue Produktionslogik), damit
  entfaellt "als eigene Unit anlegen" laut Scope-Vorgabe.
- offen: keine DB-seitige Verifikation von `CountByFiscalYear`/`GetSequenceInfo` gegen echtes
  SQL in dieser Unit — der Test bleibt wie sein Vorbild `TestService_GetJournalSummary_
  CountsCancelledInvoices` auf Service-Ebene mit Mock-Repos. Ein SQL-Test fuer
  `invoice.PostgresRepository.CountByFiscalYear` ist als `cov-invoice-repository-number-and-
  fiscal-year-real-sql` bereits eigenstaendig im Backlog (Block B) vorgesehen und wuerde die
  Jahresgrenze dort zusaetzlich gegen echtes Postgres pruefen.

## Iteration 17 — cov-payment-repository-core-real-sql — done — 2026-08-23 00:49
- commit: 482434ef
- gebaut: `internal/biz/payment` hatte keinen einzigen DB-Test. Neue Datei
  `postgres_repository_db_test.go` (ungetaggt, package `payment`) deckt den Schreib-/Lesekern
  gegen echtes Postgres ab: `Create`+`GetByID` mit exaktem Dezimal-Roundtrip (Betrag 1029.33,
  zusaetzlich per `amount::text` roh gegen die NUMERIC-Spalte verglichen — kein Float64 im Pfad),
  `GetByID` cross-tenant (fremder Tenant bekommt `ErrNotFound`, nicht nur eine leere Struktur),
  `List` tenant- und invoice-gescoped inkl. `ORDER BY payment_date DESC, created_at DESC`,
  `Delete` cross-tenant als Bestaetigung eines wirkungslosen No-ops vs. eigener Tenant, sowie
  `CreateInTx`/`DeleteInTx` je einmal mit Commit und einmal mit Rollback (vier Tests), um zu
  beweisen, dass beide Varianten wirklich auf der aufrufereigenen Transaktion laufen und nicht
  auf einer zweiten Pool-Connection committen. `SumByInvoiceID`/`GetByIdempotencyKey` bewusst
  NICHT angefasst — das ist die Folge-Unit `cov-payment-repository-sums-and-idempotency-real-sql`.
  Seed-Helfer `seedPaymentInvoice` legt eine minimale `finance_invoices`-Zeile fuer die FK an
  (Vorlage: `hr/timetracking/postgres_invoice_reservation_test.go`), NICHT das
  `//go:build integration`-Muster aus `creditnote` (Befund 2: getaggte Tests bewegen weder Gate
  noch Coverage-Zahl).
- gate: build ok (`./internal/biz/payment/...`) | vet ok | lint ok (0 issues) |
  test ok (`go test -count=1 -v ./internal/biz/payment/`, 30 Tests gesamt davon 8 neu, 0 SKIP,
  0 FAIL) | migration n.a. (keine Schema-Aenderung) | rls-smoke n.a. formal (keine Tabellen-/
  Policy-Aenderung in dieser Unit), inhaltlich aber durch die neuen Tests selbst erbracht:
  `TestPostgresRepository_GetByID_CrossTenant_ReturnsNotFound` und
  `TestPostgresRepository_Delete_TenantScoped` belegen die Tenant-Isolation auf `finance_payments`
  als kmuhub_app gegen echtes Postgres.
- coverage: internal/biz/payment 46,4 % -> 71,5 % (eigens gemessen: Testdatei kurz beiseite
  verschoben, `go test -coverprofile` lief 46,4 % — bestaetigt `coverage_start` exakt — dann
  zurueckgelegt und erneut gemessen: 71,5 %)
- mutations-probe: erster Versuch an `Delete`s WHERE-Klausel (tenant_id-Filter entfernt) blieb
  GRUEN, weil RLS (`enable_tenant_rls('finance_payments')`, Migration 000122) den fehlenden
  App-Filter bereits allein abfaengt — das ist ein echter, dokumentierter Befund (Defense-in-Depth
  funktioniert), aber keine gueltige Probe fuer DIESEN Test. Zweiter Versuch an `List`s
  `ORDER BY payment_date DESC, created_at DESC` -> `ASC, ASC`: `TestPostgresRepository_
  List_TenantScopedAndOrdered` wurde sofort rot (erwartete vs. tatsaechliche UUID-Reihenfolge
  vertauscht). Zurueckgedreht, `git diff --stat` auf `postgres_repository.go` zeigt danach keine
  Aenderung (leerer Diff).
- verify vorgaenger: sauber — `2c602e67` (Iteration 16) aendert laut `git show --stat` und Diff
  ausschliesslich `backend/internal/biz/invoice/service_test.go` (neuer Testfall) sowie
  JOURNAL.md/BACKLOG.yml. Keine der acht Fehlerklassen einschlaegig: kein gRPC-Handler beruehrt,
  kein Stub/TODO, kein `.proto`, keine Migration, kein neuer Guard, keine neue Tabelle, keine
  Route, kein Wire-Shape-Wechsel, kein ersetzter Guard-Key.
- neue-units: keine
- offen: (a) Der erste, wirkungslose Mutations-Versuch an `Delete` ist ein Befund wert, aber kein
  Bug: RLS ist die tatsaechlich wirksame Verteidigungsschicht auf `finance_payments`, der
  App-seitige `tenant_id`-Filter im SQL ist redundant (aber richtig, als zweite Schicht). Keine
  Aktion noetig. (b) `SumByInvoiceID`/`SumByInvoiceIDInTx`/`GetByIdempotencyKey` bleiben laut Scope
  fuer `cov-payment-repository-sums-and-idempotency-real-sql` (deps auf diese Unit, naechste in
  der Block-B-Reihe) offen.

## Iteration 18 — cov-payment-repository-sums-and-idempotency-real-sql — done — 2026-08-23 01:05
- commit: 76b41d02
- gebaut: Zweite Haelfte zu Iteration 17. `postgres_repository_db_test.go` (weiter ungetaggt,
  package `payment`) bekommt neun weitere Tests fuer `SumByInvoiceID`, `SumByInvoiceIDInTx` und
  `GetByIdempotencyKey` gegen echtes Postgres: keine Zahlung (COALESCE liefert 0, nicht NULL/Fehler),
  drei Teilzahlungen mit Bruchcent die in Summe eine Ueberzahlung ergeben (exakter Dezimalvergleich
  100.10+100.10+100.10=300.30), eine Zahlung eines FREMDEN Tenants auf dieselbe invoice_id (die FK
  `fk_finance_payments_invoice` referenziert nur `finance_invoices(id)`, nicht `(tenant_id,id)` — ein
  Tenant kann eine Payment-Zeile mit fremder invoice_id anlegen, die Summe darf sie trotzdem nicht
  aufnehmen), eine geloeschte/stornierte Zahlung (dieses Paket kennt keinen Status, Stornierung =
  Delete), sowie der zentrale Unterschied `SumByInvoiceIDInTx` sieht die eigene noch nicht committete
  Zahlung, ein Pool-Read ausserhalb der Transaktion sieht sie erst nach Commit (READ COMMITTED).
  `GetByIdempotencyKey`: derselbe Key in zwei Tenants aufgeloest — der partielle Unique-Index aus
  Migration 000215 ist `(tenant_id, idempotency_key)`, also bewusst PRO TENANT eindeutig, kein globaler
  Konflikt (kein Fund, Bestaetigung). Zusaetzlich unbekannter Key -> (nil, nil), keine Fehlerbehandlung
  auf jeder Call-Site noetig.
- gate: build ok (`./internal/biz/payment/...`) | vet ok | lint ok (0 issues) |
  test ok (`go test -count=1 -v ./internal/biz/payment/`, 38 Tests gesamt davon 9 neu, 0 SKIP,
  0 FAIL) | migration n.a. (keine Schema-Aenderung) | rls-smoke n.a. formal (keine Tabellen-/
  Policy-Aenderung in dieser Unit), inhaltlich durch
  `TestSumByInvoiceID_ExcludesForeignTenantPaymentOnSameInvoiceID` erbracht
- coverage: internal/biz/payment 71,5 % -> 84,8 % (eigens gemessen mit
  `go test -coverprofile=/tmp/cov_payment_final.out ./internal/biz/payment/` nach Iteration-17-Stand,
  bestaetigt `coverage_start`-Bezugswert 46,4 % ueber beide Iterationen der Unit-Reihe hinweg als
  konsistent)
- mutations-probe: erster Versuch (WHERE-Klausel von `sumByInvoiceID` auf reines `invoice_id = $2`
  gekuerzt, `$1` unreferenziert) fuehrte zu einem HAENGENDEN `go test`-Prozess (pgx extended-protocol
  Bind mit ueberzaehligem, unreferenziertem Parameter — kein Timeout, musste per TaskStop beendet
  werden; kein Produktionscode-Fund, nur eine untaugliche Mutation). Zweiter Versuch mit
  `WHERE ($1::uuid IS NOT NULL OR true) AND invoice_id = $2` (syntaktisch gueltig, Tenant-Filter
  faktisch deaktiviert) blieb GRUEN — derselbe Befund wie in Iteration 17 bei `Delete`: RLS
  (`enable_tenant_rls('finance_payments')`) faengt den fehlenden App-Filter bereits ab, bevor die
  Zeile die Query verlaesst. Dritter, gueltiger Versuch traf den tatsaechlichen Witz der Unit:
  `SumByInvoiceIDInTx` auf `r.pool` statt `tx` umgestellt (Tx-Parameter ignoriert) ->
  `TestSumByInvoiceIDInTx_SeesUncommittedPayment_PoolDoesNotUntilCommit` sofort ROT ("got 0" statt
  "70.00"). Zurueckgedreht, `git diff --stat` auf `postgres_repository.go` zeigt danach keine
  Aenderung (leerer Diff).
- verify vorgaenger: sauber — `482434ef` (Iteration 17) aendert laut `git show --stat` und Diff
  ausschliesslich `backend/internal/biz/payment/postgres_repository_db_test.go` (neue Testdatei) sowie
  JOURNAL.md/BACKLOG.yml. Keine der acht Fehlerklassen einschlaegig: kein gRPC-Handler beruehrt,
  kein Stub/TODO, kein `.proto`, keine Migration, kein neuer Guard, keine neue Tabelle, keine Route,
  kein Wire-Shape-Wechsel, kein ersetzter Guard-Key.
- neue-units: keine
- offen: Der wirkungslose erste Mutations-Versuch bestaetigt erneut (wie schon in Iteration 17 bei
  `Delete`), dass der App-seitige `tenant_id`-Filter auf `finance_payments`-Lesepfaden gegenueber RLS
  redundant, aber als zweite Schicht korrekt ist — kein Bug, keine Aktion noetig. Fuer kuenftige
  Mutations-Proben auf demselben Muster: ein WHERE-Praedikat mit unreferenziertem `$N`-Platzhalter
  kann `go test` unter pgx haengen lassen statt sauber zu fehlern — lieber das Praedikat
  bedingungslos wahr machen (`$N::uuid IS NOT NULL OR true`) als den Platzhalter ganz zu entfernen.

## Iteration 19 — cov-dunning-repository-core-real-sql — done — 2026-08-23 01:05
- commit: 54e6b37a
- gebaut: `internal/biz/dunning` hatte wie `payment` (vor Lauf 11) keinen einzigen DB-Test. Neue
  ungetaggte `postgres_repository_db_test.go` deckt Create/GetByID (Decimal-Roundtrip fuer fee und
  interest ADR-0007, Fremdtenant -> ErrDunningNotFound), List mit Filter+Paginierung (Status-Filter,
  Level-Filter, InvoiceID-Filter, jeweils mit Limit kleiner als Treffermenge — Gesamtzaehler bleibt
  vom LIMIT unberuehrt, tenant-scope) und UpdateStatus. Beim Bauen der UpdateStatus-Tests einen
  echten Produktionsbug gefunden und in derselben Unit behoben (Block B ist laut BACKLOG.yml-Kopf
  ausdruecklich Bug-Suche, kein reines Coverage-Schneiden, und der Fix beruehrt keine der
  Unverhandelbaren Grenzen — kein Guard, keine Route, keine Migration): `UpdateStatus` setzte
  `sent_at = $2` bedingungslos. `service_gobd.go:UpdateDunningStatus` (Admin-Override, z. B.
  Ruecknahme von "sent" auf "draft") ruft `repo.UpdateStatus(..., sentAt)` mit `sentAt = nil` fuer
  jeden Uebergang ausser dem tatsaechlichen Versand — genau der im Scope beschriebene Fall
  "sentAt = nil muss die Spalte unveraendert lassen". Vorher wurde bei jeder Korrektur (z. B.
  faelschlich auf "sent" gesetzt, dann zurueckgenommen) der historische Versandzeitstempel
  stillschweigend auf NULL gesetzt — der GoBD-relevante Beleg, wann eine Mahnung wirklich verschickt
  wurde, ging verloren. Fix: `sent_at = COALESCE($2, sent_at)`. Beide Aufrufer (service.go:343 mit
  immer gesetztem `&now`, service_gobd.go:50 mit dem beschriebenen nil-Fall) bleiben unveraendert
  kompatibel.
- gate: build ok (`./internal/biz/dunning/...`) | vet ok | lint ok (0 issues) |
  test ok (`go test -count=1 ./internal/biz/dunning/...`, alle Subpakete gruen, 0 SKIP) |
  migration n.a. (keine Schema-Aenderung) | rls-smoke n.a. formal (keine neue Tabelle/Policy),
  inhaltlich durch `TestPostgresRepository_GetByID_CrossTenant_ReturnsNotFound`,
  `TestPostgresRepository_List_TenantScoped` und
  `TestPostgresRepository_UpdateStatus_CrossTenant_ZeroRowsAffected` erbracht
- coverage: internal/biz/dunning 65,1 % -> 81,0 % (eigens gemessen: Vorher-Wert per
  `git stash push -u -- postgres_repository.go postgres_repository_db_test.go` +
  `go test -coverprofile` auf dem dadurch wiederhergestellten Vor-Unit-Stand, danach
  `git stash pop`; coverage_start aus dem Backlog nennt 61,8 % CI-Stand 32570176303 — die Differenz
  zum selbst gemessenen 65,1 % ist die kumulierte Bewegung durch die uebrigen Block-A/B-Units seit
  diesem CI-Lauf, siehe Regel aus Lauf 8 zu coverage_start als reine Plausibilitaetskontrolle)
- mutations-probe: `COALESCE($2, sent_at)` zurueck auf reines `sent_at = $2` gesetzt ->
  `TestPostgresRepository_UpdateStatus_SentAtNil_LeavesColumnUnchanged` sofort ROT ("Expected value
  not to be nil"). Zurueckgedreht, `git diff --stat` auf `postgres_repository.go` zeigt danach nur
  die beabsichtigte Fix-Zeile (Kommentar + COALESCE), keinen Rest der Mutation.
- verify vorgaenger: sauber — `76b41d02` (Iteration 18) aendert laut `git show --stat` ausschliesslich
  eine neue Testdatei (`internal/biz/payment/postgres_repository_db_test.go`, 226 Zeilen) sowie
  JOURNAL.md/BACKLOG.yml. Keine Produktionslogik beruehrt, keine der acht Fehlerklassen einschlaegig.
- neue-units: keine
- offen: Der gefundene Bug (`sent_at`-Verlust bei Admin-Override) ist in dieser Unit selbst behoben,
  nicht nur dokumentiert — siehe Begruendung oben (Block-B-Mandat, keine Unverhandelbare Grenze
  beruehrt). `GetByInvoiceID`/`GetByInvoiceIDs`/`GetHighestLevelByInvoiceID` bleiben laut Scope fuer
  `cov-dunning-repository-invoice-lookups-real-sql` (naechste in der Block-B-Reihe, deps auf diese
  Unit) offen — inklusive der dort explizit genannten Gleichstand-Frage bei
  `GetHighestLevelByInvoiceID`.

## Iteration 20 — cov-dunning-repository-invoice-lookups-real-sql — done — 2026-08-23 01:20
- commit: a584aeda
- gebaut: Die drei Lesewege von Rechnung zu Mahnung in `internal/biz/dunning/postgres_repository.go`
  gegen echtes Postgres getestet: `GetByInvoiceID` (Level-aufsteigende Sortierung, Fremdtenant mit
  korrekter Invoice-ID liefert leer), `GetByInvoiceIDs` (leere ID-Liste -> leere Map ohne Query,
  unbekannte ID und Rechnung ohne Mahnung stoeren den bekannten Treffer nicht, keine `nil`-Panics
  beim Ranging), `GetHighestLevelByInvoiceID` (kein Datensatz -> `(nil, nil)`, Fremdtenant -> `nil`).
  Beim Bauen den in den Notes vorhergesagten Fund bestaetigt und in derselben Zeile behoben (Block-B-
  Mandat, keine Unverhandelbare Grenze beruehrt): `ORDER BY level DESC` ohne Tiebreaker ist bei zwei
  Datensaetzen derselben (hoechsten) Stufe nicht deterministisch — `finance_dunning_records` hat
  keinen Unique-Constraint auf `(invoice_id, level)` (siehe `000045_create_finance_tables.up.sql`),
  Gleichstand ist also schema-legal. Fix: `ORDER BY level DESC, created_at DESC` — der zuletzt
  angelegte Datensatz auf der hoechsten Stufe gewinnt. `GetHighestLevelByInvoiceID` und
  `GetByInvoiceID` sind derzeit ausserhalb des `dunning`-Pakets nirgends aufgerufen (nur
  `GetByInvoiceIDs` via `service.go:184`) — geprueft mit gezieltem Grep auf `backend/cmd` und
  `backend/internal`, kein toter Code im Sinne von "entfernen", da Teil des `Repository`-Interfaces
  und offensichtlich fuer kuenftige Aufrufer (z. B. Detail-Ansicht einer Rechnung) vorgesehen.
- gate: build ok (`./internal/biz/dunning/... ./internal/gateway/... ./cmd/biz/... ./cmd/gateway/...`)
  | vet ok | lint ok (0 issues) | test ok (`go test -count=1 ./internal/biz/dunning/`, 10
  Postgres-Tests real gelaufen, 0 SKIP, `-v`-Log geprueft) | test restliche Unterpakete ok
  (`go test -count=1 ./internal/biz/dunning/...`) | migration n.a. (keine Schema-Aenderung) |
  rls-smoke n.a. formal (keine neue Tabelle/Policy), inhaltlich durch
  `TestPostgresRepository_GetByInvoiceID_OrderedByLevel_CrossTenantEmpty` und
  `TestPostgresRepository_GetHighestLevelByInvoiceID_NoRecords_ReturnsNilNil` (Fremdtenant-Fall)
  erbracht | gateway-Test nicht gelaufen, da keine Route/kein Proto beruehrt (nicht Pflicht laut
  Schritt 5)
- coverage: internal/biz/dunning 81,0 % -> 88,7 % (eigens gemessen: Vorher-Wert per
  `git stash push -u -- postgres_repository.go postgres_repository_db_test.go` +
  `go test -coverprofile` auf dem dadurch wiederhergestellten Vor-Unit-Stand, danach
  `git stash pop`; coverage_start aus dem Backlog nennt 61,8 % CI-Stand 32570176303 — die Differenz
  zum selbst gemessenen 81,0 % ist die kumulierte Bewegung durch Iteration 19
  (`cov-dunning-repository-core-real-sql`) seit diesem CI-Lauf)
- mutations-probe: `ORDER BY level DESC, created_at DESC` zurueck auf `ORDER BY level DESC` gesetzt ->
  `TestPostgresRepository_GetHighestLevelByInvoiceID_TieBreaksOnMostRecent` sofort ROT
  (erwartete `tieNewer.ID`, bekam die aeltere Gleichstand-ID zurueck). Zurueckgedreht,
  `git diff` auf `postgres_repository.go` zeigt danach nur die beabsichtigte Fix-Zeile plus
  den erklaerenden Kommentar, keinen Rest der Mutation.
- verify vorgaenger: sauber — `54e6b37a` (Iteration 19) aendert laut `git show --stat` und Diff
  ausschliesslich `backend/internal/biz/dunning/postgres_repository.go` (die `COALESCE`-Fix-Zeile,
  Caller `service.go:343` und `service_gobd.go:50` gegengeprueft — beide bleiben kompatibel) sowie
  eine neue Testdatei (`postgres_repository_db_test.go`) und JOURNAL.md/BACKLOG.yml. Keine der acht
  Fehlerklassen einschlaegig: kein gRPC-Handler beruehrt, kein Stub/TODO, kein `.proto`, keine
  Migration, kein neuer Guard, keine neue Tabelle, keine Route, kein Wire-Shape-Wechsel, kein
  ersetzter Guard-Key.
- neue-units: keine
- offen: `internal/biz/dunning/postgres_repository.go` hat jetzt DB-Tests fuer alle sieben
  Methoden von `PostgresRepository`; `PostgresConfigRepository.Get`/`Upsert` (Zeile 274/310) sind
  weiterhin ungetestet — genau das Ziel der naechsten Unit in der Block-B-Reihe
  (`cov-dunning-config-repository-real-sql`, deps: `[]`, sofort ziehbar).

## Iteration 21 — cov-dunning-config-repository-real-sql — done — 2026-08-23 01:15
- commit: 5bca2b78
- gebaut: Vier DB-Tests fuer `PostgresConfigRepository` (Get/Upsert der Mahnkonfiguration je Mandant)
  gegen echtes Postgres: `Get` ohne existierende Konfiguration liefert `(nil, nil)` (belegt, dass
  `Service.GetConfig` sich zurecht darauf verlaesst, um den Default anzulegen); `Upsert` Insert-Zweig
  roundtriped alle drei Gebuehren exakt als Dezimal (12.34/0.07/199.99, `.Equal()`-Vergleich statt
  `.String()` — pgx liefert NUMERIC(10,2) ohne trailing zero, "2.50" kommt als "2.5" zurueck, kein Bug,
  nur Darstellungsform); Update-Zweig (`ON CONFLICT (tenant_id) DO UPDATE`) ueberschreibt die
  bestehende Zeile in-place, per `count(*)`-Kontrolle als System-Ctx belegt, dass kein Duplikat
  entsteht; zwei Tenants upserten unabhaengig, jeder behaelt seine eigene Gebuehr, und ein
  Cross-Tenant-`Get` (ctx Tenant B, Parameter Tenant A) liefert `nil` — RLS blockt trotz explizitem
  `WHERE tenant_id = $1` im SQL, weil die USING-Klausel zusaetzlich `current_tenant_id()` verlangt.
  Kein Produktionsfehler gefunden: `ON CONFLICT (tenant_id)` matched den echten Unique-Constraint
  (`uq_finance_dunning_config_tenant`, Migration 000045), RLS-Policy aktiv seit Migration 000122.
- gate: build ok (`./internal/biz/dunning/... ./internal/gateway/...`) | vet ok | lint ok (0 issues) |
  test ok (`go test -count=1 -v ./internal/biz/dunning/`, alle 60 Tests gruen inkl. der 4 neuen,
  0 SKIP) | test restliche Unterpakete ok (`go test -count=1 -p 1 ./internal/biz/dunning/...`) |
  migration n.a. (keine Schema-Aenderung) | rls-smoke n.a. formal (keine neue Tabelle/Policy),
  inhaltlich durch `TestPostgresConfigRepository_Upsert_TwoTenants_Independent` (Cross-Tenant-Read
  liefert nil) erbracht | gateway-Test nicht gelaufen, da keine Route/kein Proto beruehrt (nicht
  Pflicht laut Schritt 5)
- coverage: internal/biz/dunning 88,7 % -> 92,2 % (eigens gemessen: Vorher-Wert per
  `git stash push -u -- internal/biz/dunning/postgres_repository_db_test.go` + `go test
  -coverprofile` auf dem dadurch wiederhergestellten Vor-Unit-Stand, danach `git stash pop`;
  coverage_start aus dem Backlog nennt 61,8 % CI-Stand 32570176303 — die Differenz zum selbst
  gemessenen 88,7 % ist die kumulierte Bewegung durch die Block-B-Dunning-Units der Iterationen
  19+20 seit diesem CI-Lauf)
- mutations-probe: `level1_fee = EXCLUDED.level1_fee,` aus dem `ON CONFLICT DO UPDATE`-Set-Teil
  entfernt -> `TestPostgresConfigRepository_Upsert_UpdateBranch_OverwritesInPlace` sofort ROT
  ("Should be true", got 0 statt der erwarteten 2.50). Zurueckgedreht, `git diff --stat` auf
  `postgres_repository.go` zeigt keine Aenderung (nur CRLF-Hinweis von Git, kein Diff-Inhalt).
- verify vorgaenger: sauber — `a584aeda` (Iteration 20) aendert laut `git show --stat` und Diff
  ausschliesslich `backend/internal/biz/dunning/postgres_repository.go` (der `ORDER BY level DESC,
  created_at DESC`-Tiebreaker in `GetHighestLevelByInvoiceID`, Caller gegengeprueft — nirgends
  ausserhalb des Pakets aufgerufen ausser `GetByInvoiceIDs` via `service.go:184`, unveraendert
  kompatibel) sowie eine neue Testdatei (`postgres_repository_db_test.go`, 158 Zeilen) und
  JOURNAL.md/BACKLOG.yml. Keine der acht Fehlerklassen einschlaegig: kein gRPC-Handler beruehrt,
  kein Stub/TODO, kein `.proto`, keine Migration, kein neuer Guard, keine neue Tabelle, keine Route,
  kein Wire-Shape-Wechsel, kein ersetzter Guard-Key.
- neue-units: keine
- offen: `internal/biz/dunning/postgres_repository.go` und `postgres_repository_config...` haben
  jetzt vollstaendige DB-Test-Abdeckung fuer beide Repository-Typen. Naechste Unit in der
  Block-B-Dunning-Reihe laut Backlog ist `cov-dunning-service-gobd-real-sql`
  (deps: `[cov-dunning-repository-core-real-sql]`, bereits `done` seit Iteration 19 — damit sofort
  ziehbar).

## Iteration 22 — cov-dunning-service-gobd-real-sql — done — 2026-08-23 01:20
- commit: ff1f3cd3
- gebaut: `service_gobd.go` (UpdateDunningStatus, SendDunningNotice, GenerateGoBDExport/buildGoBDCSV)
  war bereits fast vollstaendig ueber Mocks getestet (94,4-100 % je Funktion); die vier
  `done_when`-Punkte waren zum Teil schon erfuellt (Formel-Injektions-Schutz existiert seit
  `csvutil.NeutralizeFormulaCell`, Idempotenz von `SendDunningNotice` ueber
  `ErrDunningNotDraft`). Echter Fund beim Gegenpruefen der CSV-Struktur gegen die Dezimal-
  konvention: `buildGoBDCSV` schrieb TaxRate/NetAmount/TaxAmount/GrossTotal mit Punkt-Dezimal
  (`"38.00"`, `"7.5"`), waehrend der bestehende DATEV-Export (`datev/exporter.go:317`
  `formatDecimalForDATEV`) fuer denselben Empfaengerkreis (DATEV/Lexware/Excel-DE) explizit auf
  Komma umstellt — genau die Regel, die der eigene Kommentar an `buildGoBDCSV`
  ("DATEV / Lexware / Excel render Umlaute ... when opened locally") schon fuer die
  Kundennamen-Spalte einhaelt, aber fuer die Zahlenspalten nie zog. Ohne Komma importiert
  deutsches Excel/DATEV "38.00" als Text statt als Zahl — die GoBD/IDEA-Journalspalten waeren
  fuer die Maschine, an die sie adressiert sind, nicht auswertbar. Fix: neue Funktion
  `germanDecimal` (gleiche Ersetzung wie `formatDecimalForDATEV`, `strings.Replace(s, ".", ",", 1)`)
  auf TaxRate/NetAmount/TaxAmount/GrossTotal in `buildGoBDCSV` angewandt. Kein interner Go-Parser
  liest die CSV zurueck (per Grep bestaetigt: einzige Verwender sind der gRPC-Handler
  `biz_grpc.go:2512` und der Download-Response, keine Rueckparsung) — Aenderung risikofrei.
  Bestehenden Test `TestService_GenerateGoBDExport_PostingColumns` auf "200,00"/"38,00"
  angepasst, neuen Test `TestService_GenerateGoBDExport_GermanDecimalSeparator` ergaenzt
  (prueft alle vier Spalten inkl. Bruchsatz "7.5" -> "7,5" und dass keine Zelle mehr einen
  bloßen Punkt traegt).
  Fuer den Block-B-Auftrag "gegen echtes SQL" vier neue Tests in
  `service_gobd_db_test.go` (neue Datei, ungetaggt): `Service.UpdateDunningStatus` und
  `Service.SendDunningNotice` ueber die echte `PostgresRepository` statt `MockRepository`
  getrieben — die Mock-Tests in `service_test.go` belegen nur die Verzweigungslogik, diese
  hier beweisen dieselbe Logik nach einem echten SQL-Roundtrip. Abgedeckt: Draft->Sent setzt
  `sent_at` persistent (nicht nur am In-Memory-Record); der in den Notes verlangte
  Uebergangs-Check "Sprung von versendet zurueck auf offen: erlaubt und begruendet oder Fund" —
  Ergebnis: erlaubt und begruendet (Doc-Kommentar an `UpdateDunningStatus` nennt es explizit
  "intentionally permissive" fuer Admin-Korrekturen), per Test belegt, dass der Ruecksprung den
  bereits gesetzten `sent_at` NICHT loescht (haengt an der `COALESCE(sent_at)`-Fix aus
  Iteration 19); Cross-Tenant-Update auf Service-Ebene liefert `ErrDunningNotFound`; die in den
  Notes verlangte Idempotenz-Frage bei doppeltem `SendDunningNotice` — zweiter Aufruf ueber
  echtes SQL liefert `ErrDunningNotDraft`, `sent_at` bleibt exakt der Wert des ersten Sendevorgangs
  (kein zweiter Versand, kein ueberschriebener Zeitstempel).
- gate: build ok (`./internal/biz/dunning/... ./internal/gateway/... ./cmd/biz/... ./cmd/gateway/...`)
  | vet ok | lint ok (0 issues) | test ok (`go test -count=1 -v ./internal/biz/dunning/`, alle Tests
  gruen inkl. 4 neuer Real-SQL-Tests, 18 Postgres-Tests im Paket insgesamt, 0 SKIP) | test restliche
  Unterpakete ok (`go test -count=1 -p 1 ./internal/biz/dunning/...`) | `go test -count=1
  ./internal/gateway/` gruen (TestOpenAPIRouteDrift, obwohl keine Route beruehrt, pflichtgemaess
  gelaufen) | migration n.a. (keine Schema-Aenderung) | rls-smoke n.a. formal (keine neue
  Tabelle/Policy), inhaltlich durch `TestService_UpdateDunningStatus_RealSQL_CrossTenant_NotFound`
  erbracht
- coverage: internal/biz/dunning 92,2 % -> 92,2 % (eigens gemessen, `go tool cover -func`, vor/nach
  identisch gemessen). Unveraendert, weil `service_gobd.go` bereits vor dieser Iteration ueber
  Mock-Tests zu 94,4-100 % pro Funktion abgedeckt war (`UpdateDunningStatus` 94,4 %,
  `SendDunningNotice`/`GenerateGoBDExport`/`buildGoBDCSV` je 100 %) — der Auftrag dieser Unit war
  laut Scope Bug-Suche und Fundamentabsicherung gegen echtes SQL, nicht Zeilenabdeckung; die
  Coverage-Zahl sagt hier bewusst nichts aus (`germanDecimal` selbst liegt bei 100 %, die eine
  verbleibende Luecke in `UpdateDunningStatus` ist der Repo-Fehlerpfad bei `UpdateStatus`, gegen
  Mock bereits geprueft, siehe TestService_UpdateDunningStatus_NotFound-Nachbarn in service_test.go).
- mutations-probe: zwei Laeufe, beide gegen `cp`-Sicherungskopien (nicht `git checkout`),
  zurueckgeschrieben, `git diff --stat` danach je nur die beabsichtigte Aenderung.
  (a) `germanDecimal` auf `strings.Replace(s, "X", ",", 1)` verstuemmelt (Identitaet auf "."
  waere wegen ungenutztem Import nicht kompiliert) -> `TestService_GenerateGoBDExport_
  GermanDecimalSeparator` rot an allen vier Spalten-Assertions ("100.00 should not contain .").
  (b) `if record.Status != models.DunningStatusDraft` in `sendAndNotify` (service.go) auf
  `if false` gesetzt -> `TestService_SendDunningNotice_RealSQL_DoubleCall_SecondCallRejected` rot:
  zweiter Aufruf liefert keinen Fehler mehr UND `sent_at` bewegt sich beim zweiten (unnoetigen)
  Send-Vorgang weiter (Beweis, dass der Guard tatsaechlich sowohl den Fehler als auch den
  Zeitstempel-Schutz traegt, nicht nur einen der beiden). Beide Male zurueckgedreht, Diff sauber.
  Anmerkung zur Arbeitsweise: die ersten beiden eigenen Real-SQL-Tests verglichen anfangs den
  In-Memory-`time.Now()`-Rueckgabewert des Service direkt gegen den aus Postgres zurueckgelesenen
  `sent_at` — TIMESTAMPTZ rundet auf Mikrosekunden, das schlug bei exaktem `.Equal()` fehl. Kein
  Produktionsfehler, sondern ein Testfehler: korrigiert, indem der Vergleichswert per zweitem
  `GetByID` aus der DB gelesen wird statt aus dem Rueckgabewert.
- verify vorgaenger: sauber — `5bca2b78` (Iteration 21) aendert laut `git show --stat` und Diff
  ausschliesslich eine neue Testdatei (`postgres_repository_db_test.go`, 153 Zeilen fuer
  `PostgresConfigRepository`) sowie JOURNAL.md/BACKLOG.yml. Keine der acht Fehlerklassen
  einschlaegig: kein gRPC-Handler beruehrt, kein Stub/TODO, kein `.proto`, keine Migration, kein
  neuer Guard, keine neue Tabelle, keine Route, kein Wire-Shape-Wechsel, kein ersetzter Guard-Key.
- neue-units: keine
- offen: Block-B-Dunning-Reihe ist damit vollstaendig (Repository-Core, Invoice-Lookups,
  Config-Repository, Service-GoBD). Naechste Units laut Backlog-Reihenfolge sind die
  `cov-invoice-repository-*`-Serie (ab `cov-invoice-repository-list-filter-real-sql`, deps: `[]`,
  sofort ziehbar). Luke pruefen: die geaenderte CSV-Ausgabe (Komma statt Punkt bei
  Steuersatz/Netto/MwSt/Brutto) ist eine sichtbare Formatanaenderung fuer jeden, der den GoBD-Export
  bereits einmal heruntergeladen und in einem NICHT-deutschen Tool geoeffnet hat (dort waere Punkt
  korrekt gewesen) — fuer DATEV/Lexware/deutsches Excel ist Komma dagegen die Voraussetzung, damit
  die Zahl ueberhaupt als Zahl importiert wird, siehe `datev/exporter.go` als bestehende Referenz
  fuer denselben Empfaengerkreis.

## Iteration 23 — cov-invoice-repository-list-filter-real-sql — done — 2026-08-23 01:29
- commit: c4564870
- gebaut: neue ungetaggte Testdatei `postgres_repository_list_db_test.go` fuer
  `PostgresRepository.List` (postgres_repository.go:195-306), gegen echtes Postgres statt Mock:
  Statusfilter (samt Total-Zaehler unter Filter), Datumsgrenzen (beide inklusiv, per Test belegt und
  jetzt als Kommentar in `postgres_repository.go:211-221` festgehalten), Overdue (nur sent+ueberfaellig,
  nicht draft+ueberfaellig, nicht sent+nicht-faellig), ContactID- und RecurringID-Filter (inkl.
  Ausschluss NULL-Zeilen und Fremd-ID), Total-Zaehler unabhaengig vom Limit (5 passende + 2
  nicht-passende Zeilen, Limit 2 -> total 5, Seite 2), Fremdtenant-Isolation ueber die echte
  RLS-Verbindung. Freitextsuche entfaellt — `ListFilter` hat kein Suchfeld, siehe `repository.go:90-102`.
  Helfer fuer contacts (users->contacts FK-Kette) und finance_recurring_invoices neu angelegt, da
  bislang niemand in diesem Paket contact_id/recurring_id ungetaggt geseedet hat.
- gate: build ok (`./internal/biz/invoice/... ./internal/gateway/... ./cmd/biz/... ./cmd/gateway/...`)
  | vet ok | lint ok (0 issues) | test ok (`go test -count=1 -v ./internal/biz/invoice/`, 81 PASS,
  0 SKIP, DATABASE_URL gegen kmuhub_app) | restliche Unterpakete ok (keine Unterpakete unter
  internal/biz/invoice) | migration n.a. (keine Schema-Aenderung) | rls-smoke erbracht durch
  `TestPostgresRepository_List_CrossTenantIsolation` (eigener Tenant total=1, fremder Tenant taucht
  weder in Liste noch Zaehler auf) | gateway-Gate n.a. formal (keine Route beruehrt, deshalb nicht
  pflichtgemaess gelaufen)
- coverage: internal/biz/invoice 34,8 % (CI-Stand, coverage_start) -> 46,0 % (eigens gemessen,
  `go tool cover -func`, lokaler Lauf nach der Aenderung; kein Vorher-Lauf mangels Testdatei zum
  Vergleich noetig, da vorher exakt 0 Tests fuer List existierten)
- mutations-probe: `invoice_date <= $N` (DateTo-Bedingung, postgres_repository.go:218) auf
  `invoice_date < $N` verstuemmelt -> `TestPostgresRepository_List_DateRangeInclusiveBoundaries` rot
  (total 2 statt 1 erwartet — Test selbst hatte den falschen erwarteten Wert im ersten Anlauf, siehe
  unten — nach Korrektur: total 1 statt 2, "invoice dated exactly on DateTo must be included" false).
  Zurueckgedreht, `git diff --stat` auf postgres_repository.go danach leer.
  Nebenbefund beim Bauen (kein Produktionsfehler, Testfehler): `defer testutil.CleanupRow(...,
  "tenants", tenantID)` in der Contact-ID-Testfunktion lief vor den `t.Cleanup`-registrierten
  User/Contact-Loeschungen (defer laeuft vor t.Cleanup, nicht danach) und riss dadurch eine
  FK-Verletzung beim Testende (non-fatal, nur t.Logf). Behoben: Tenant-Cleanup selbst als
  `t.Cleanup` registriert, direkt nach `EnsureTenant`, damit LIFO die Reihenfolge korrekt umkehrt.
- verify vorgaenger: sauber — `ff1f3cd3` (Iteration 22) aendert laut `git show --stat` und Diff
  `service_gobd.go` (germanDecimal-Helper, reine String-Transformation), zwei neue Testdateien/-faelle
  und BACKLOG.yml/JOURNAL.md. Keine der acht Fehlerklassen einschlaegig: kein gRPC-Handler beruehrt,
  kein Stub/TODO, kein `.proto`, keine Migration, kein neuer Guard, keine neue Tabelle, keine Route,
  kein Wire-Shape-Wechsel gegen das FE (CSV-Exportformat ist kein FE-Vertrag), kein ersetzter
  Guard-Key.
- neue-units: keine
- offen: naechste Unit laut Backlog-Reihenfolge ist `cov-invoice-repository-datev-export-keyset-real-sql`
  (deps: [cov-invoice-repository-list-filter-real-sql], jetzt ziehbar). Luke pruefen: die
  Datumsgrenzen-Entscheidung (beide inklusiv) ist reines Bestandsverhalten, keine Aenderung — nur
  jetzt erstmals per Test belegt und im Code kommentiert.

## Iteration 24 — cov-invoice-repository-datev-export-keyset-real-sql — done — 2026-08-23 01:45
- commit: 1a8ecf64
- gebaut: neue ungetaggte Testdatei `postgres_repository_datev_export_db_test.go` fuer
  `PostgresRepository.ListForDATEVExport` (postgres_repository.go:626-682), gegen echtes Postgres:
  Paging-Vollstaendigkeit (Baseline-Call mit grossem Limit vs. Seiten-fuer-Seite-Traversal mit
  Limit 2, identische ID-Menge, keine Dubletten, ueber Mix aus passenden/nicht-passenden Zeilen
  — falscher Status im Zeitraum, richtiger Status ausserhalb), Gleichdatum-Tiebreak (drei Rechnungen
  mit identischem invoice_date ueber eine Seitengrenze hinweg, Reihenfolge nach id ASC bewiesen ueber
  sortierte UUID-Strings — Postgres vergleicht uuid byteweise, was der lexikographischen Ordnung der
  kanonischen Hex-Form entspricht), Datumsgrenzen inklusive (genau auf fromDate/toDate rein, einen Tag
  davor/danach raus), Statusfilter (draft im Zeitraum nie im Export, sent schon), Fremdtenant-Isolation.
  Erste Seite (afterDate/afterID nil) und Folgeseite sind in der Tiebreak- und der Paging-Vollstaendigkeit-
  Probe beide abgedeckt.
- gate: build ok (`./internal/biz/invoice/... ./internal/gateway/... ./cmd/biz/... ./cmd/gateway/...`)
  | vet ok | lint ok (0 issues) | test ok (`go test -count=1 -v ./internal/biz/invoice/`, alle gruen,
  0 SKIP, DATABASE_URL gegen kmuhub_app) | restliche Unterpakete ok (keine Unterpakete unter
  internal/biz/invoice) | migration n.a. (keine Schema-Aenderung) | rls-smoke erbracht durch
  `TestPostgresRepository_ListForDATEVExport_CrossTenantIsolation` (Tenant A sieht nur eigene
  Rechnung trotz gleichem Zeitraum/Status bei Tenant B) | gateway-Gate n.a. formal (keine Route
  beruehrt, deshalb nicht pflichtgemaess gelaufen)
- coverage: internal/biz/invoice 34,8 % (CI-Stand, coverage_start) -> 49,1 % (eigens gemessen,
  `go tool cover -func`, lokaler Lauf nach der Aenderung; Vorher-Wert 46,0 % aus Iteration 23 ist der
  naehere Vergleichspunkt, da diese Unit direkt auf der vorigen aufsetzt — Beitrag dieser Iteration
  46,0 % -> 49,1 %)
- mutations-probe: Cursor-Vergleich `(invoice_date, id) > (...)` auf `>=` verstuemmelt (Zeile 630) ->
  zwei Tests rot: `TestPostgresRepository_ListForDATEVExport_PagingCoversFullSet` ("duplicate invoice
  ... returned across pages") und `TestPostgresRepository_ListForDATEVExport_SameDateCursorTiebreak`
  (Folgeseite hatte 2 statt 1 Eintrag — die Grenzzeile der Vorseite kam ein zweites Mal). Zurueckgedreht,
  `git diff --stat` auf postgres_repository.go danach leer.
- verify vorgaenger: sauber — `c4564870` (Iteration 23) aendert laut `git show --stat` und Diff nur
  einen Kommentar in `postgres_repository.go:211-212` (keine Logikaenderung) sowie eine neue
  Testdatei (`postgres_repository_list_db_test.go`, 363 Zeilen) und BACKLOG.yml/JOURNAL.md. Keine der
  acht Fehlerklassen einschlaegig: kein gRPC-Handler beruehrt, kein Stub/TODO, kein `.proto`, keine
  Migration, kein neuer Guard, keine neue Tabelle, keine Route, kein Wire-Shape-Wechsel, kein
  ersetzter Guard-Key.
- neue-units: keine
- offen: naechste Unit laut Backlog-Reihenfolge ist `cov-invoice-repository-gobd-export-real-sql`
  (deps: [cov-invoice-repository-datev-export-keyset-real-sql], jetzt ziehbar). Luke pruefen: keine
  besonderen Punkte — reine Coverage-Unit ohne Verhaltensaenderung.

## Iteration 25 — cov-invoice-repository-gobd-export-real-sql — done — 2026-08-23 01:52
- commit: 0b56911b
- gebaut: neue ungetaggte Testdatei `postgres_repository_gobd_export_db_test.go` fuer
  `PostgresRepository.ListForGoBDExport` (postgres_repository.go:574-620), gegen echtes Postgres:
  Statusfilter (alle vier Nicht-Entwurfsstatus inkl. cancelled enthalten, draft ausgeschlossen —
  mit ausfuehrlichem Testkommentar zur bewussten Divergenz gegenueber ListForDATEVExport, das
  cancelled zusaetzlich ausschliesst), leere Rechnungsnummer ausgeschlossen (invoice_number != ''
  greift auch bei status=sent), Datumsgrenzen inklusive, Sortierung nach invoice_number statt
  invoice_date/Einfuegereihenfolge (Beweis fuer die "gap-free journal"-Zusage im Doc-Kommentar),
  Zeilenpositionen werden mitgeliefert, Fremdtenant-Isolation.
  NEBENFUND (root cause fuer die Divergenz-Frage aus den Notes der Unit): `Service.ListForGoBDExport`
  hat NULL Aufrufer im Produktionscode — der tatsaechliche GoBD-CSV-Export
  (`biz_grpc.go:2451 GenerateGoBDExport`) baut seine Zeilen ueber `invoiceService.ListForDATEVExport`
  (schliesst cancelled aus) plus `creditNoteService.ListForDATEVExport` als negative Stornozeilen,
  NICHT ueber `ListForGoBDExport`. Die abweichende Statuslogik der getesteten Methode wird also nie
  ausgefuehrt — sie ist entweder abgeloester Alt-Code oder eine nie verdrahtete zweite Export-Variante.
  Root-Cause-Entscheidung (Entfernen vs. verdrahten) gehoert Luke, siehe neue-units.
- gate: build ok (`./internal/biz/invoice/... ./internal/gateway/... ./cmd/biz/... ./cmd/gateway/...`)
  | vet ok | lint ok (0 issues) | test ok (`go test -count=1 -v ./internal/biz/invoice/`, alle gruen,
  0 SKIP, DATABASE_URL gegen kmuhub_app) | restliche Unterpakete ok (keine Unterpakete unter
  internal/biz/invoice) | migration n.a. (keine Schema-Aenderung) | rls-smoke erbracht durch
  `TestPostgresRepository_ListForGoBDExport_CrossTenantIsolation` | gateway-Gate zusaetzlich
  gelaufen (`go test ./internal/gateway/ -run TestOpenAPIRouteDrift`, gruen) obwohl keine Route
  beruehrt wurde
- coverage: internal/biz/invoice 34,8 % (CI-Stand, coverage_start) -> 51,4 % (eigens gemessen,
  `go tool cover -func`, lokaler Lauf nach der Aenderung; Vorher-Wert 49,1 % aus Iteration 24 ist
  der naehere Vergleichspunkt, da diese Unit direkt auf der vorigen aufsetzt — Beitrag dieser
  Iteration 49,1 % -> 51,4 %)
- mutations-probe: `status != 'draft'`-Filter entfernt UND `ORDER BY invoice_number ASC` auf
  `ORDER BY invoice_date ASC` verstuemmelt (eine Codezeile, zwei Bedingungen gleichzeitig
  betroffen) -> zwei Tests rot: `TestPostgresRepository_ListForGoBDExport_StatusFilter` (draft
  erscheint im Export, 5 statt 4 Eintraege) und `TestPostgresRepository_ListForGoBDExport_
  OrderedByInvoiceNumber` (RE-2026-0001 und RE-2026-0003 vertauscht). Zurueckgedreht,
  `git diff --stat` auf postgres_repository.go danach leer.
- verify vorgaenger: sauber — `1a8ecf64` (Iteration 24) aendert laut `git show --stat` und Diff nur
  eine neue Testdatei (`postgres_repository_datev_export_db_test.go`, 254 Zeilen) sowie
  BACKLOG.yml/JOURNAL.md. Keine der acht Fehlerklassen einschlaegig: kein gRPC-Handler beruehrt,
  kein Stub/TODO, kein `.proto`, keine Migration, kein neuer Guard, keine neue Tabelle, keine
  Route, kein Wire-Shape-Wechsel, kein ersetzter Guard-Key.
- neue-units: `verify-invoice-list-for-gobd-export-unreachable` (ans Backlog-Ende angehaengt) —
  klaert den oben genannten Nebenfund: Service.ListForGoBDExport hat keinen Aufrufer, der echte
  Export laeuft ueber ListForDATEVExport + CreditNote-Stornos. Entscheidung Entfernen vs.
  Verdrahten liegt bei Luke.
- offen: naechste Unit laut Backlog-Reihenfolge ist `cov-invoice-repository-number-and-fiscal-year-
  real-sql` (deps: [], sofort ziehbar). Luke pruefen: die neue Verify-Unit
  `verify-invoice-list-for-gobd-export-unreachable` vor einer moeglichen Aenderung an
  `GenerateGoBDExport` lesen — dort steckt die fachliche Entscheidung, ob cancelled-Rechnungen
  in den GoBD-Export gehoeren.

## Iteration 26 — cov-invoice-repository-number-and-fiscal-year-real-sql — done — 2026-08-23 01:45
- commit: ea99721d
- gebaut: neue ungetaggte Testdatei `postgres_repository_number_fiscal_year_db_test.go` fuer
  `InvoiceNumberExists` (postgres_repository.go:494-507) und `CountByFiscalYear` (509-526), gegen
  echtes Postgres: Tenant-Scoping bei InvoiceNumberExists (dieselbe Nummer in zwei Mandanten),
  Negativfall fuer nie vergebene Nummer, Jahresgrenze 31.12./01.01. bei CountByFiscalYear,
  Ausschluss von Entwuerfen und nummernlosen Rechnungen, Fremdtenant-Isolation bei
  CountByFiscalYear.
  NEBENFUND (RLS als Verteidigungslinie bestaetigt): die Mutations-Probe auf den expliziten
  `tenant_id = $1`-Filter in `InvoiceNumberExists` (Filter entfernt, Vergleich durch eine
  triviale, immer wahre `$1::uuid = $1::uuid`-Bedingung ersetzt, um den Parameter syntaktisch
  weiter zu nutzen) blieb GRUEN — kein Datenleck, weil die RLS-Policy auf `finance_invoices`
  bereits vor der WHERE-Klausel der Anwendung greift und die Session auf `app.tenant_id`
  beschraenkt. Der App-seitige Filter ist damit defense-in-depth, nicht die einzige Schranke.
  Kein Fund, kein Fix noetig — aber dokumentiert, weil es die naechste Mutations-Probe auf einer
  RLS-geschuetzten Tabelle spart, blind denselben Fehler zu erwarten.
  Die Fiskaljahr=Kalenderjahr-Annahme aus den Notes ist am Code belegt: `GetJournalSummary`
  (service_gobd.go:137-170) uebergibt denselben `year`-Parameter sowohl an
  `numberSeqRepo.GetSequenceInfo` (Sequenznummer im Format RE-YYYY-NNNN) als auch an
  `CountByFiscalYear`, und es existiert keine separate Konfiguration fuer ein abweichendes
  Geschaeftsjahr (Grep nach "FiscalYear"/"fiscal_year" ueber internal/ liefert keinen Treffer
  ausserhalb dieser beiden Aufrufer und der DATEV/E-Rechnungs-Exporte, die denselben
  Kalenderjahr-Parameter weiterreichen). Kein Fund.
- gate: build ok (`./internal/biz/invoice/... ./internal/gateway/... ./cmd/biz/... ./cmd/gateway/...`)
  | vet ok | lint ok (0 issues) | test ok (`go test -count=1 -v ./internal/biz/invoice/`, alle
  gruen, 0 SKIP, DATABASE_URL gegen kmuhub_app) | gateway-Gate zusaetzlich gelaufen
  (`go test ./internal/gateway/ -run TestOpenAPIRouteDrift`, gruen) obwohl keine Route beruehrt
  wurde | migration n.a. (keine Schema-Aenderung) | rls-smoke erbracht durch
  `TestPostgresRepository_InvoiceNumberExists_IsTenantScoped` und
  `TestPostgresRepository_CountByFiscalYear_IsTenantScoped`
- coverage: internal/biz/invoice 51,4 % (eigens gemessen Iteration 25, naeherer Vergleichspunkt
  als der CI-Stand 34,8 % aus coverage_start) -> 52,3 % (eigens gemessen, `go tool cover -func`,
  lokaler Lauf nach der Aenderung)
- mutations-probe: `status != 'draft'` in `CountByFiscalYear` auf `status != 'nonexistent-status'`
  verstuemmelt (schliesst dann keine Entwuerfe mehr aus) -> `TestPostgresRepository_
  CountByFiscalYear_ExcludesDraftAndUnnumbered` rot (erwartet 1, erhalten 2). Zurueckgedreht,
  `git diff --stat` auf postgres_repository.go danach leer. Eine zweite Probe auf den
  `tenant_id`-Filter in `InvoiceNumberExists` blieb gruen (siehe RLS-Nebenfund oben) und wurde
  deshalb durch die obige Probe auf CountByFiscalYear ersetzt, um tatsaechlich eine rote
  Assertion zu belegen.
- verify vorgaenger: sauber — `0b56911b` (Iteration 25) aendert laut `git show --stat` und Diff
  nur eine neue Testdatei (`postgres_repository_gobd_export_db_test.go`, 239 Zeilen, untagged,
  6 Testfunktionen, kein Skip/TODO/Unimplemented) sowie BACKLOG.yml/JOURNAL.md. Keine der acht
  Fehlerklassen einschlaegig: kein gRPC-Handler beruehrt, kein Stub/TODO, kein `.proto`, keine
  Migration, kein neuer Guard, keine neue Tabelle, keine Route, kein Wire-Shape-Wechsel, kein
  ersetzter Guard-Key.
- neue-units: keine
- offen: naechste Unit laut Backlog-Reihenfolge ist `cov-invoice-repository-payment-stats-real-sql`
  (deps: [cov-invoice-repository-number-and-fiscal-year-real-sql], jetzt ziehbar). Luke pruefen:
  keine besonderen Punkte — reine Coverage-Unit ohne Verhaltensaenderung. Der RLS-Nebenfund oben
  ist informativ, keine Aktion noetig.

## Iteration 27 — cov-invoice-repository-payment-stats-real-sql — done — 2026-08-23 02:20
- commit: 5bae68ed
- gebaut: neue ungetaggte Testdatei `postgres_repository_payment_stats_db_test.go` für
  `AggregatePaymentStats` (postgres_repository.go:530-570), gegen echtes Postgres: leerer
  Zeitraum liefert Nullwerte, Status-Klassifizierung (paid/sent/overdue/cancelled) inkl.
  unabhängig gerechneter Summen, arithmetisches Mittel von avg_days_to_pay über zwei bezahlte
  Rechnungen mit deterministischem `updated_at` (Session-Timezone vorher gegen
  `docker-postgres-1` verifiziert: UTC), Tenant-Isolation gegen einen zweiten Tenant mit sehr
  hohen Beträgen.
  ECHTER FUND (dokumentiert, nicht gefixt — Coverage-Unit ändert kein Verhalten):
  `AggregatePaymentStats` joint nie gegen `finance_payments` und summiert stattdessen
  `gross_total` nach `status`. Zwei Konsequenzen, je mit eigenem Test belegt: (a) eine
  Rechnung mit bereits verbuchter Teilzahlung (400 von 1000) zeigt weiterhin den vollen Betrag
  als offen — `TestPostgresRepository_AggregatePaymentStats_PartialPaymentNotNettedFromOutstanding`;
  (b) eine als 'paid' markierte, überbezahlte Rechnung (450 gezahlt auf 400 Rechnungsbetrag)
  zeigt nur die 400 als vereinnahmt — `_PaidAmountIgnoresOverpayment`. Genau dieselbe
  Fehlerklasse wurde für die Offene-Posten-Liste bereits einmal behoben: der Kopfkommentar von
  `postgres_open_items.go` beschreibt wörtlich denselben Fehler ("used the invoice gross
  amount, so an invoice with a partial payment was reported as fully outstanding") als bereits
  gefixt für `openItemsBase` — `AggregatePaymentStats` ist eine zweite, unabhängige
  Implementierung derselben Kennzahl-Familie ohne diesen Fix. Als Fix-Unit
  `fix-payment-stats-outstanding-ignores-recorded-payments` ans Backlog-Ende gehängt.
- gate: build ok (`./internal/biz/invoice/... ./internal/biz/payment/... ./internal/gateway/...
  ./cmd/biz/... ./cmd/gateway/...`) | vet ok | lint ok (0 issues) | test ok
  (`go test -count=1 -v ./internal/biz/invoice/`, 103 PASS, 0 SKIP, DATABASE_URL gegen
  kmuhub_app) | gateway-Gate zusätzlich gelaufen (`TestOpenAPIRouteDrift`, grün) obwohl keine
  Route berührt wurde | migration n.a. | rls-smoke erbracht durch
  `TestPostgresRepository_AggregatePaymentStats_IsTenantScoped`
- coverage: internal/biz/invoice 52,3 % (eigens gemessen Iteration 26, näherer
  Vergleichspunkt als der CI-Stand 34,8 % aus coverage_start) -> 53,2 % (eigens gemessen,
  `go tool cover -func`, lokaler Lauf nach der Änderung)
- mutations-probe: `status NOT IN ('paid', 'cancelled')` im `total_outstanding_amount`-Filter
  auf `status != 'paid'` verstümmelt (cancelled-Rechnung würde dann wieder mitgezählt) ->
  `TestPostgresRepository_AggregatePaymentStats_ClassifiesByStatusAndSumsGross` rot (erwartet
  500, erhalten 9500 — die 9000 der cancelled-Rechnung schlugen durch). Zurückgedreht,
  `git diff --stat` auf postgres_repository.go danach leer.
- verify vorgaenger: sauber — `ea99721d` (Iteration 26) ändert laut `git show --stat` und Diff
  nur eine neue Testdatei (`postgres_repository_number_fiscal_year_db_test.go`, 172 Zeilen,
  untagged, kein Skip/TODO/Unimplemented) sowie BACKLOG.yml/JOURNAL.md. Keine der acht
  Fehlerklassen einschlägig: kein gRPC-Handler berührt, kein Stub/TODO, kein `.proto`, keine
  Migration, kein neuer Guard, keine neue Tabelle, keine Route, kein Wire-Shape-Wechsel, kein
  ersetzter Guard-Key.
- neue-units: fix-payment-stats-outstanding-ignores-recorded-payments
- offen: nächste Unit laut Backlog-Reihenfolge ist `cov-invoice-postgres-transactions-real-sql`
  (deps: [], sofort ziehbar). Luke prüfen: die neue Fix-Unit
  `fix-payment-stats-outstanding-ignores-recorded-payments` enthält auch die offene Frage, ob
  TotalPaidAmount künftig Umsatz oder tatsächlichen Cashflow zeigen soll — Produktentscheidung,
  UI-Beschriftung ist gesperrt, aber lesbar.

## Iteration 28 — cov-invoice-postgres-transactions-real-sql — done — 2026-08-23 02:35
- commit: (siehe naechster docs-Commit)
- gebaut: neue ungetaggte Testdatei `postgres_transactions_bracket_db_test.go` (5 Tests, real
  Postgres) fuer die Transaktionsklammer aus `Create` und `UpdateInTx`
  (`postgres_repository.go:48` bzw. `:328`), die auch `Service.Send` fuer die gekoppelte
  Nummernvergabe+Status-Aenderung verwendet: (1) Fehler beim Anlegen der Rechnung
  (`chk_finance_invoices_status` verletzt) — Header UND Positionen bleiben leer; (2) Fehler beim
  Anlegen der Positionen (`chk_invoice_lines_quantity` verletzt auf der zweiten von zwei Zeilen)
  — Header UND die erste, gueltige Zeile werden ebenfalls zurueckgerollt; (3) Fehler beim
  Header-UPDATE innerhalb von `UpdateInTx` — die urspruengliche Zeile bleibt komplett unberuehrt,
  weil das DELETE nie laeuft; (4) Fehler beim Loeschen-und-Neueinfuegen der Zeilen — Header UND
  die zwischenzeitlich geloeschte Zeile werden wiederhergestellt (isoliert die Garantie, die
  `send_atomic_integration_test.go` sonst nur ueber den vollen Send-Flow zeigt); (5) Fehler beim
  Commit — ein gueltiges `UpdateInTx` in einer Transaktion, die durch eine andere Anweisung
  (`SELECT 1/0`) vorher abgebrochen wird: `tx.Commit()` liefert tatsaechlich einen Fehler
  ("commit unexpectedly resulted in rollback", empirisch mit `-v` verifiziert, nicht geraten),
  und das eigentlich gueltige Update persistiert nicht.
  ABWEICHUNG vom Backlog-Scope: die Unit-Beschreibung passt nicht zur Datei
  `postgres_transactions.go` — die enthaelt seit Commit `120b32a4` ("add consolidated
  transaction ledger") ausschliesslich `ListTransactions`/`incomeTransactions`/
  `expenseTransactions`, eine reine Read-Union ueber `finance_payments`+`finance_expenses` ohne
  jede Transaktionsklammer. Die im Scope beschriebene "Transaktionsklammer... unter der Rechnung,
  Positionen und Nummernvergabe gemeinsam geschrieben werden" existiert tatsaechlich in
  `service.go:Send` (Zeile 519) + `postgres_repository.go:Create/UpdateInTx` — beide standen
  bereits in `sources`. Dorthin gebaut statt gegen den falschen Dateinamen.
  Ausserdem: der urspruenglich geplante Test fuer "Fehler bei der Nummernvergabe selbst"
  (`NextNumberInTx`) liess sich mit echtem SQL nicht deterministisch erzwingen, ohne den
  Unique-Constraint der Sequenztabelle aufzubrechen (wie es der getaggte Test fuer einen anderen
  Zweck tut) — `pgx.QueryRow.Scan` verwirft bei mehreren Treffern stillschweigend alle bis auf
  die erste Zeile, es entsteht kein Fehler. Stattdessen ueber den Header-UPDATE- und
  Commit-Fehler abgedeckt, die real reproduzierbar sind.
- gate: build ok (`./internal/biz/invoice/... ./internal/gateway/... ./cmd/biz/... ./cmd/gateway/...`)
  | vet ok | lint ok (0 issues) | test ok (`go test -count=1 -v ./internal/biz/invoice/`, 108
  PASS, 0 SKIP, DATABASE_URL gegen kmuhub_app) | gateway-Gate zusaetzlich gelaufen
  (`TestOpenAPIRouteDrift`, 836 Routen gegen 838 Pfade, gruen) obwohl keine Route beruehrt wurde
  | migration n.a. | rls-smoke n.a. (keine Tabelle/Policy angefasst, nur Transaktionsverhalten)
- coverage: internal/biz/invoice 53,2 % (eigens gemessen Iteration 27, gleiche Messmethode) ->
  56,3 % (eigens gemessen, `go tool cover -func`, lokaler Lauf mit und ohne die neue Datei
  verglichen)
- mutations-probe: `UpdateInTx` am Ende auf `_ = r.insertInvoiceLines(ctx, tx, inv); return nil`
  verstuemmelt (Fehler beim Zeilen-Reinsert wird verschluckt) ->
  `TestPostgresRepository_UpdateInTx_RollbackOnLineReinsertRestoresDeletedLine` rot ("An error is
  expected but got nil"). Zurueckgedreht, `git diff --stat` auf postgres_repository.go danach
  leer. WICHTIGER NEBENFUND beim ersten Versuch dieser Probe: mit einem manuellen
  `require.NoError(t, tx.Rollback(ctx))` als letzter Handlungsschritt (statt `defer` direkt nach
  `Begin`) haengt der Testlauf, wenn eine fruehere Assertion per `require.Error` fehlschlaegt —
  `t.FailNow()` (`runtime.Goexit`) ueberspringt den manuellen Rollback-Aufruf, die Transaktion
  haelt eine Connection aus dem Pool, und `pool.Close()` im `t.Cleanup` blockiert, bis diese
  Connection zurueckkommt (120 s Tool-Timeout, Task musste per TaskStop beendet werden). Fix: in
  allen drei `UpdateInTx`-Tests sofort nach `pool.Begin` ein `defer tx.Rollback(ctx)` als
  Sicherheitsnetz ergaenzt (Goexit fuehrt Defers weiterhin aus). Damit lief dieselbe Mutation beim
  zweiten Versuch sauber in unter einer Sekunde rot statt zu haengen.
- verify vorgaenger: sauber — `5bae68ed` (Iteration 27) aendert laut `git show --stat` und Diff
  nur eine neue Testdatei (`postgres_repository_payment_stats_db_test.go`, 280 Zeilen, untagged,
  kein Skip/TODO/Unimplemented) sowie BACKLOG.yml/JOURNAL.md. Keine der acht Fehlerklassen
  einschlaegig: kein gRPC-Handler beruehrt, kein Stub/TODO, kein `.proto`, keine Migration, kein
  neuer Guard, keine neue Tabelle, keine Route, kein Wire-Shape-Wechsel, kein ersetzter
  Guard-Key.
- neue-units: keine
- offen: naechste Unit laut Backlog-Reihenfolge ist `cov-invoice-service-gobd-lock-real-sql`
  (deps: [cov-invoice-postgres-transactions-real-sql], jetzt ziehbar). Luke pruefen: (1) die oben
  dokumentierte Scope/Sources-Abweichung bei dieser Unit — falls in `BACKLOG.yml` noch weitere
  Units auf denselben irrefuehrenden Dateinamen `postgres_transactions.go` verweisen, lohnt ein
  kurzer Grep vor dem naechsten Lauf; (2) der Mutations-Probe-Nebenfund zu haengenden
  Test-Pools ist keine Aktion fuer Luke, aber falls in aelteren `_db_test.go`-Dateien dasselbe
  Pattern (manuelles `tx.Rollback` ohne vorgeschalteten `defer`) vorkommt, waere das ein
  latentes Risiko fuer kuenftige rote Testlaeufe, die dann als Haenger statt als klarer Fail
  erscheinen — kein Fund in dieser Iteration ausserhalb der neuen Datei geprueft.

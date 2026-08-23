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
- commit: 5aa17392
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

## Iteration 29 — cov-invoice-service-gobd-lock-real-sql — done — 2026-08-23 02:12
- commit: 7608b6027a330acc7c06be7e7441ac233a20c433
- gebaut: GoBD-Sperrfläche (`LockInvoice`/`isInvoiceLocked`/`SetLock`) gegen echtes SQL und
  jeder Schreibweg auf `invoices` auf Sperrprüfung untersucht. ECHTER FUND UND GEFIXT:
  `DetectOverdue` (`service.go:737`) rief `repo.UpdateStatus` direkt auf, ohne `isInvoiceLocked`
  zu prüfen — der einzige Schreibweg ohne den Guard, den `Update`/`MarkPaid`/`Cancel` bereits
  alle tragen. Ein administrativ gesperrtes, versendetes, überfälliges Invoice wurde beim
  naechsten Overdue-Sweep trotzdem auf `overdue` geflippt, obwohl derselbe Übergang über
  `MarkPaid`/`Cancel` explizit mit `ErrInvoiceLocked` verweigert wird — Inkonsistenz zum Rest der
  GoBD-§146-Barriere im selben File. Root-Cause-Fix: `isInvoiceLocked`-Check vor dem
  `UpdateStatus`-Aufruf in `DetectOverdue`, mit `slog.Warn` und `continue` statt Fehler (bulk-Sweep
  darf an einer gesperrten Rechnung nicht abbrechen). `GetOverdue` selbst bleibt bewusst
  ungefiltert nach `locked_at` — `dunning.Service.invReader.GetOverdue` (`dunning/service.go:173`)
  nutzt dieselbe Query fuer Mahnkandidaten, und eine gesperrte Rechnung ist weiterhin fällig; die
  Barriere gehört also in `invoice.Service.DetectOverdue`, nicht in die gemeinsame Repo-Query.
  Neue Tests: `postgres_repository_lock_db_test.go` (untagged, echtes Postgres) — SetLock/GetByID
  Roundtrip (Ersatz fuer das getaggte `TestInvoiceLockColumns`), SetLock gegen unbekannte
  Invoice-ID ist ein stiller No-op (kein Fehler — dokumentiert, warum `LockInvoice`s
  vorgeschaltetes `GetByID` load-bearing ist, nicht redundant), `GetOverdue` liefert eine
  gesperrte Rechnung weiterhin zurueck (Read-Seite des Fundes). In `service_test.go` (Mock-Repo):
  `DetectOverdue` überspringt gesperrte Rechnung (der Fix-Beweis), doppeltes Sperren liefert
  `ErrInvoiceLocked` und veraendert weder `LockedAt` noch `LockedBy` der ersten Sperre, Sperren
  eines Entwurfs schlaegt fehl, Sperren einer Bexio-importierten Rechnung liefert
  `ErrExternalReadOnly`.
  GEPRUEFT, KEIN FUND: `LinkTimeTracking` (`service.go:352`) hat KEINEN Status-/Sperr-Check —
  aber der einzige Aufrufer im gesamten Repo ist `biz_grpc.go:2118`, direkt nach `invoiceService
  .Create` mit der gerade erst erzeugten (also zwingend `draft`) Invoice-ID; kein eigener RPC
  exponiert die Methode. Kein erreichbarer Schreibpfad auf eine gesperrte Rechnung — bewusst
  NICHT abgesichert (lean: Validierung fuer einen Zustand, der nicht eintreten kann, waere
  Symptombekaempfung ohne Wirkung).
  Entsperren: existiert nicht im gesamten `internal/biz/invoice`-Paket (Grep auf "Unlock" und auf
  `locked_at = NULL` uber Migrationen liefert null Treffer) — die GoBD-Sperre ist einseitig, wie
  von `notes:` der Unit erwartet.
  WORM-Vergleich (von notes: gefordert): anders als das Belegarchiv (Migration 315, Lauf 10 —
  DB-seitiger WORM-Trigger) erzwingt hier ausschliesslich Go-Code (`isInvoiceLocked`-Checks in
  Service-Methoden) die Unveraenderlichkeit; ein direktes UPDATE gegen `finance_invoices` unter
  `kmuhub_app` (z. B. ein zukuenftiges Feature, das den Service-Layer umgeht) waere von der DB aus
  nicht blockiert.
- gate: build ok (`./internal/biz/invoice/... ./internal/gateway/... ./cmd/biz/... ./cmd/gateway/...`)
  | vet ok | lint ok (0 issues) | test ok (`go test -count=1 ./internal/biz/invoice/...`, `-p 1`,
  DATABASE_URL gegen kmuhub_app, 0 SKIP, 115 PASS im Hauptpaket) | migration n.a. (keine neue
  Tabelle/Spalte) | rls-smoke n.a. (bestehende Policy nicht angefasst) | gateway-Gate nicht
  gesondert gelaufen, da keine Route beruehrt wurde (nur `go build` gegen `internal/gateway`
  als Teil des Build-Schritts, gruen)
- coverage: internal/biz/invoice 56,3 % (eigens gemessen Iteration 28, `go tool cover -func`,
  gleiche Messmethode) -> 59,3 % (eigens gemessen, `go tool cover -func`, lokaler Lauf nach
  Fix+Tests)
- mutations-probe: den `isInvoiceLocked(inv)`-Check in `DetectOverdue` durch `if false` ersetzt ->
  `TestService_DetectOverdue_SkipsLockedInvoice` rot ("expected: sent, actual: overdue").
  Zurueckgedreht, `git diff --stat` auf service.go zeigt danach nur die eine beabsichtigte
  11-Zeilen-Ergaenzung (keine Restspur der Mutation).
- verify vorgaenger: sauber — `5aa17392` (Iteration 28) aendert laut `git show --stat` nur eine
  neue, untagged Testdatei (`postgres_transactions_bracket_db_test.go`, 240 Zeilen) plus
  BACKLOG.yml/JOURNAL.md. Keine der acht Fehlerklassen einschlaegig: kein gRPC-Handler beruehrt,
  kein Stub/TODO/Unimplemented, kein `.proto`, keine Migration, kein neuer Guard, keine neue
  Tabelle, keine Route, kein Wire-Shape-Wechsel, kein ersetzter Guard-Key.
- neue-units: keine — der einzige gefundene Bug (DetectOverdue vs. Lock) war klein genug, um
  root-cause in dieser Unit selbst gefixt zu werden (Regel 1 dieses Laufs), statt als eigene Unit
  angelegt zu werden.
- offen: naechste Unit laut Backlog-Reihenfolge ist `cov-invoice-service-gobd-journal-summary-real-sql`
  (deps: [cov-invoice-service-gobd-lock-real-sql], jetzt ziehbar). Luke pruefen: der WORM-Vergleich
  oben (Go-Code-Sperre vs. DB-Trigger beim Belegarchiv) ist eine reine Beobachtung, keine
  Handlungsaufforderung dieser Iteration — falls ein Servicelayer-Bypass fuer `finance_invoices`
  je denkbar wird, waere ein DB-seitiger Trigger die robustere Absicherung, aber das ist eine
  Produktentscheidung, keine Coverage-Unit.

## Iteration 30 — cov-invoice-service-gobd-journal-summary-real-sql — done — 2026-08-23 02:19
- commit: 517f9226
- gebaut: Berichtsteil der GoBD-Flaeche (`GetJournalSummary`, `ValidateInvoiceNumber`,
  `GetPaymentStats`) auf echte Luecken untersucht statt nur Zeilen abzudecken.
  ECHTER FUND UND GEFIXT: `invoiceNumberPattern` (`service_gobd.go:19`) hatte einen
  unbegrenzten Sequenzteil `\d{4,}`. `ValidateInvoiceNumber` verwirft den Fehler von
  `strconv.Atoi(m[3])` (`seq, _ := strconv.Atoi(...)`) — ein vom Aufrufer (gRPC-Request,
  `biz_grpc.go:2324`, direkte Vertrauensgrenze) gelieferter, ausreichend langer Ziffernstring
  (z. B. 25 Neunen) ueberlaeuft int64, `strconv.Atoi` klemmt bei `ErrRange` still auf
  `math.MaxInt64` statt den Fehler zu melden, und `ValidateInvoiceNumber` meldet `ValidFormat:
  true` mit dem Kauderwelsch-Kanonwert `RE-2026-9223372036854775807`. Per eigenem Go-Snippet
  gegen die echte Regex/Atoi-Kombination verifiziert (nicht geraten), bevor gefixt wurde.
  Root-Cause-Fix: Sequenzteil auf `\d{4,10}` begrenzt (bis zu 10 Mrd. Rechnungen/Jahr, weit
  ueber jedem realistischen Wert, sicher innerhalb des int-Bereichs) statt eine Laengenpruefung
  hinter dem Atoi-Aufruf nachzuruesten — die Grenze gehoert an die Stelle, die die Garantie
  eigentlich geben soll (die Regex selbst).
  ZWEITER FUND (nur getestet, kein Fix noetig): `GetJournalSummary`s Kernbehauptung — eine echte
  Luecke (`GapsDetected > 0`) wird tatsaechlich gemeldet — war in keinem bestehenden Test
  bewiesen; alle drei bisherigen Tests (Cancelled/FiscalYearBoundary/import_test.go) trafen nur
  den `GapsDetected == 0`-Zweig. Neuer `TestService_GetJournalSummary_DetectsRealGap`
  (Sequenz auf 5, nur 3 nummerierte Rechnungen persistiert -> `GapsDetected = 2`) schliesst das.
  Der leere-Jahr-Zweig (`seq == nil`, kein einziges Invoice je ausgestellt) war ebenfalls
  ungetestet -> `TestService_GetJournalSummary_EmptyYearReturnsZeroesNotError` (Nullwerte, kein
  Fehler statt Nil-Pointer-Panik).
  `ValidateInvoiceNumber` zusaetzlich mit Sonderzeichen (`@`, SQL-Fragment, Zeilenumbruch,
  Leerzeichen, `/`, `€`, Nullbyte — sieben Faelle, alle `ValidFormat: false`) und der
  Ueberlaenge-Regression (25-stellige Sequenz) belegt; Leerstring und vergebene Nummer waren
  schon vorher abgedeckt (`TestService_ValidateInvoiceNumber_InvalidFormat`/`_AlreadyUsed`).
  `GetPaymentStats` NICHT erneut angefasst: Service-Ebene (Mock) bereits in
  `TestService_GetPaymentStats_Empty`/`_Aggregates` getestet, Repo-Ebene (`AggregatePaymentStats`)
  bereits gegen echtes Postgres in Iteration 27 (`postgres_repository_payment_stats_db_test.go`)
  — dort blieb nichts offen, das diese Unit nachziehen musste.
  Beitrag zur Nummernluecken-Frage aus A16 (`verify-invoice-number-gap-detection-gobd`,
  Iteration 16, bereits `status: done`): keine neue Antwort noetig — Iteration 16 hat die
  Existenz der Erkennung bereits am Code belegt (`GapsDetected = max(CurrentNumber - Count, 0)`,
  ueber gRPC exponiert). Diese Iteration ergaenzt nur den bislang fehlenden Beweis, dass der
  positive Zweig (`GapsDetected > 0`) auch wirklich feuert — vorher war das reine Behauptung
  ohne Testabdeckung.
  Kein DB-seitiger Test in dieser Unit: `GetSequenceInfo` (die einzige der drei Funktionen mit
  echter SQL-Implementierung, in `quote.PostgresNumberSequenceRepo`) traegt bereits real-SQL-Tests
  aus dem `quote`-Paket (`TestNumberSequenceRepo_SequentialWithinTenantAndYear`,
  `_GetSequenceInfo_NilForUnseenSequence`), documentType-agnostisch — ein invoice-spezifischer
  Duplikat-Test haette nichts Neues bewiesen. `InvoiceNumberExists`/`CountByFiscalYear` (die von
  `ValidateInvoiceNumber`/`GetJournalSummary` tatsaechlich aufgerufenen Repo-Methoden) sind
  bereits real-SQL-getestet aus einer fruehren Iteration (`postgres_repository_number_fiscal_
  year_db_test.go`). Diese Unit haengt sich also bewusst auf der Service-Ebene ein (Mock-Repos),
  wo der eigentliche Fund lag — nicht als Coverage-Uebung fehlplaziert auf der Repo-Ebene.
- gate: build ok (`./internal/biz/invoice/... ./internal/gateway/...`) | vet ok | lint ok
  (0 issues) | test ok (`go test -count=1 -v ./internal/biz/invoice/`, DATABASE_URL gegen
  kmuhub_app, 0 SKIP, 119 PASS) | test ok (`go test -count=1 -p 1 ./internal/biz/invoice/...`) |
  test ok (`go test -count=1 ./internal/gateway/` — TestOpenAPIRouteDrift trotz keiner
  Routenaenderung pflichtgemaess gelaufen) | migration n.a. (keine Schema-Aenderung) |
  rls-smoke n.a. (keine Tabellen-/Policy-Aenderung)
- coverage: internal/biz/invoice 59,3 % (Iteration-29-Messung, `go tool cover -func`) -> 59,4 %
  (eigene Messung, `go tool cover -func`, gleiche Methode, lokaler Lauf nach Fix+Tests)
- mutations-probe: drei Laeufe, alle gegen eine `cp`-Sicherungskopie (nicht `git checkout`),
  alle zurueckgeschrieben, `diff` gegen die Kopie am Ende identisch (0 Zeilen Unterschied),
  finaler `git diff --stat` zeigt nur die beabsichtigte 8-Zeilen-Ergaenzung an service_gobd.go.
  (a) Regex zurueck auf unbegrenztes `\d{4,}` ->
  `TestService_ValidateInvoiceNumber_ExcessivelyLongSequenceRejected` rot: `ValidFormat` true
  statt false, Canonical `RE-2026-9223372036854775807` statt leer — Produktionsschaden woertlich
  reproduziert. (b) `gaps := max(seq.CurrentNumber-invoiceCount, 0)` zu `gaps := 0` verstuemmelt ->
  `TestService_GetJournalSummary_DetectsRealGap` rot (erwartet 2, bekommen 0). (c) den
  `if seq == nil { ... }`-Fruehausstieg komplett entfernt ->
  `TestService_GetJournalSummary_EmptyYearReturnsZeroesNotError` nicht nur rot, sondern Panik
  (Nil-Pointer-Dereferenzierung auf `seq.CurrentNumber` in Zeile 151) — der Test faengt also
  nicht nur eine falsche Zahl, sondern einen echten Crash-Pfad ab.
- verify vorgaenger: sauber — `7608b602` (Iteration 29) fuegt nur einen `isInvoiceLocked`-Check
  in `DetectOverdue` (11 Zeilen) plus zwei neue, ungetaggte Testdateien hinzu (per `git show
  --stat` und Diff geprueft). Keine der acht Fehlerklassen einschlaegig: kein gRPC-Handler
  beruehrt (reine Service-Methode), kein Stub/TODO/Unimplemented, kein `.proto`, keine Migration,
  kein neuer Guard, keine neue Tabelle, keine Route, kein Wire-Shape-Wechsel, kein ersetzter
  Guard-Key.
- neue-units: keine — beide Funde (Atoi-Overflow, ungetesteter Gap-Zweig) waren klein genug, um
  root-cause bzw. per Test in dieser Unit selbst geschlossen zu werden, statt eine eigene Unit
  anzulegen (Regel 1 dieses Laufs: Root Cause statt Symptom, in derselben Iteration wenn moeglich).
- offen: (1) DB-Gate lief: `DATABASE_URL` als `kmuhub_app`, 0 uebersprungene Tests im
  Hauptpaket. Diese Unit selbst fuegt keine neuen DB-Tests hinzu (siehe Begruendung oben, warum
  das nicht noetig war) — die bereits bestehenden DB-Tests im Paket liefen alle mit. (2) Fachliche
  Randnotiz fuer Luke: `service_gobd.go:144` ruft `GetSequenceInfo` mit dem String-Literal
  `"invoice"` statt der Konstante `models.DocumentTypeInvoice` auf — beide haben denselben Wert
  (`models/finance.go:70`), also kein Bug, nur eine kleine Stilinkonsistenz gegenueber
  `service.go:566`, das dieselbe Konstante fuer `NextNumberInTx` verwendet. Nicht angefasst, weil
  ausserhalb des Scopes dieser Coverage-Unit und ohne Verhaltensaenderung.

## Iteration 31 — cov-invoice-repository-status-and-lock-columns-real-sql — done — 2026-08-23 02:34
- commit: a2ea5ab5
- gebaut: `UpdateStatus`/`UpdateStatusInTx`/`SetLock` (postgres_repository.go:363/375/386) auf
  Statusübergangs- und Sperr-Interaktion untersucht statt nur Zeilen abzudecken. SetLock war
  bereits aus Iteration 29 real-SQL-getestet (postgres_repository_lock_db_test.go) — diese
  Iteration deckt den fehlenden Rest: UpdateStatus/UpdateStatusInTx.
  ECHTER FUND UND GEFIXT: die schmalen Statuspfade umgehen genau die Prüfung, die die Scope-
  Beschreibung vermutet hat. `invoice.Service.MarkPaid`/`Cancel` prüfen `isInvoiceLocked(inv)`
  vor jedem `UpdateStatus`-Aufruf (GoBD §146 Sperrbarriere) — aber `internal/biz/payment`
  bekommt in `cmd/biz/main.go:164` (`payment.NewService(paymentRepo, invoiceRepo, invoiceRepo,
  pool)`) den ROHEN `invoiceRepo` direkt als `InvoiceStatusUpdater` verdrahtet, nicht
  `invoice.Service`. `payment.Service.transitionToPaidInTx` (automatischer Statuswechsel auf
  "paid", sobald Zahlungen den Bruttobetrag decken) und `revertPaidStatusInTx` (Rückabwicklung
  beim Löschen der letzten Zahlung) riefen `UpdateStatusInTx` also OHNE jede Sperrprüfung auf.
  Konkretes Szenario: `LockInvoice` erlaubt das Sperren einer "sent"-Rechnung (noch nicht
  bezahlt) — kommt danach eine Zahlung rein, die den Betrag deckt (z. B. per Bank-Matching über
  `banking` -> `paymentSvc`), flippt der Status still auf "paid", obwohl die Rechnung
  administrativ unveränderlich ist. Root-Cause-Fix: `inv.LockedAt != nil`-Guard in beiden
  Funktionen (payment/service.go), Payment-Aufzeichnung/-Löschung selbst bleibt erlaubt (das
  Geld ist real), nur der Statuswechsel wird übersprungen (mit `slog.Warn`, gleiches Muster wie
  `DetectOverdue`s bestehender `isInvoiceLocked`-Skip aus Iteration 29).
  `creditnote.Service.StornoInvoice` ruft `UpdateStatusInTx` ebenfalls direkt auf demselben
  `invoiceRepo` auf, ist aber NICHT betroffen: erreichbar ausschließlich über
  `invoice.Service.Cancel()` als `stornoCreator` (per Grep in `internal/server/*.go` bestätigt,
  kein direkter gRPC-Aufruf), und `Cancel()` prüft `isInvoiceLocked` bereits VOR dem Aufruf.
  ZWEITER FUND (dokumentiert, nicht gefixt — Design-Entscheidung, keine Lücke): das Repository
  selbst kennt `locked_at` in keiner UPDATE-WHERE-Klausel; `UpdateStatus` flippt eine gesperrte
  Rechnung klaglos (`TestPostgresRepository_UpdateStatus_IgnoresLock`, neu). Ein SQL-seitiger
  Filter (`AND locked_at IS NULL`) wurde geprüft und verworfen: `UpdateStatus`/`UpdateStatusInTx`
  prüfen betroffene Zeilen heute nirgends (weder für Tenant-Mismatch noch für "nicht gefunden"),
  ein stiller 0-Zeilen-Erfolg würde bei `transitionToPaidInTx` zu einer FALSCHEN
  "invoice auto-transitioned to paid"-Logzeile führen, obwohl nichts geschrieben wurde — ein
  neuer, subtilerer Fehler als der ursprüngliche. Die Sperrprüfung gehört deshalb bewusst in die
  aufrufende Schicht (dort, wo bereits `inv` mit aktuellem `LockedAt` vorliegt), nicht ins SQL.
  Übrige `done_when`-Punkte real-SQL bewiesen: Cross-Tenant-Update betrifft 0 Zeilen
  (`TestPostgresRepository_UpdateStatus_CrossTenantIsNoop`), zurückgerollte
  `UpdateStatusInTx`-Transaktion hinterlässt nichts
  (`TestPostgresRepository_UpdateStatusInTx_RollbackLeavesNothing`), sowie ein einfacher
  Roundtrip-Test (`TestPostgresRepository_UpdateStatus_PersistsAndRoundTrips`).
- gate: build ok (`./internal/biz/invoice/... ./internal/biz/payment/...`) | vet ok | lint ok
  (0 issues, beide Pakete) | test ok (`go test -count=1 -v ./internal/biz/invoice/`,
  DATABASE_URL gegen kmuhub_app, 0 SKIP, alle PASS inkl. 4 neuer DB-Tests) | test ok
  (`go test -count=1 -v ./internal/biz/payment/`, alle PASS inkl. 2 neuer Mock-Tests) | test ok
  (`go test -count=1 -p 1 ./internal/biz/invoice/... ./internal/biz/payment/...`) | test ok
  (`go test -count=1 ./internal/gateway/` — keine Routenänderung, trotzdem pflichtgemäß
  gelaufen) | migration n.a. (keine Schema-Änderung) | rls-smoke n.a. (keine Tabellen-/
  Policy-Änderung, nur Statuswert-Schreibpfad)
- coverage: internal/biz/invoice 59,4 % (eigene Messung vor dieser Iteration, `go tool cover
  -func`, stimmt mit Iteration-30-Endwert überein) -> 59,9 % | internal/biz/payment 84,8 %
  (eigene Messung vor dieser Iteration) -> 85,4 % (beide eigene Messungen nach Fix+Tests, gleiche
  Methode, via `git stash`/`pop` der eigenen Änderungen isoliert)
- mutations-probe: gegen eine `cp`-Sicherungskopie (nicht `git checkout`), zurückgeschrieben,
  `diff` gegen die Kopie am Ende identisch (0 Zeilen Unterschied). (a) Lock-Guard in
  `transitionToPaidInTx` auf `if false` verstümmelt -> `TestRecord_NoAutoTransitionWhenInvoiceLocked`
  rot (Statuswechsel auf "paid" fand trotz Sperre statt). (b) Lock-Guard in
  `revertPaidStatusInTx` auf `if false` verstümmelt -> `TestDelete_NoRevertWhenInvoiceLocked` rot
  (Rückabwicklung fand trotz Sperre statt). Finaler `git diff --stat` zeigt nur die
  beabsichtigte 23-Zeilen-Ergänzung an payment/service.go.
- verify vorgaenger: sauber — `517f9226` (Iteration 30) begrenzt nur den Sequenzteil der
  Rechnungsnummer-Regex auf `\d{4,10}` (1 Zeile Code plus Kommentar) und fügt Tests hinzu (per
  `git show --stat` und Diff geprüft). Keine der acht Fehlerklassen einschlägig: kein
  gRPC-Handler berührt, kein Stub/TODO/Unimplemented, kein `.proto`, keine Migration, kein neuer
  Guard, keine neue Tabelle, keine Route, kein Wire-Shape-Wechsel, kein ersetzter Guard-Key.
- neue-units: keine — der Fund (payment umgeht die Sperrprüfung) war root-cause-fixbar innerhalb
  dieser Unit (zwei Guards in payment/service.go, im Scope über die genannte "Zusammenspiel mit
  B13"-Notiz bereits vorgesehen); der zweite Fund (Repository kennt keine Sperre) ist eine
  bewusste, im Journal begründete Design-Entscheidung, keine offene Lücke, die eine Folge-Unit
  bräuchte.
- offen: (1) DB-Gate lief vollständig: DATABASE_URL als kmuhub_app, 0 übersprungene Tests in
  beiden Paketen. (2) Fachliche Randnotiz für Luke: `payment.Record()` erlaubt weiterhin explizit
  das Aufzeichnen einer Zahlung auf einer gesperrten Rechnung (nur der automatische
  Statuswechsel wird übersprungen) — das ist beabsichtigt (das Geld ist real angekommen), aber
  falls das GoBD-Konzept vorsieht, dass auch die Zahlungserfassung selbst gegen eine gesperrte
  Rechnung blockiert werden soll, ist das eine Produktentscheidung außerhalb dieser Unit. (3)
  `banking`-Service matcht Bankbewegungen über `paymentSvc` (kein direkter `invoiceRepo`-Zugriff,
  per Grep bestätigt) — profitiert also automatisch vom selben Fix, ohne eigene Änderung nötig.

## Iteration 32 — cov-invoice-repository-quote-link-and-time-tracking-real-sql — done — 2026-08-23 02:35
- commit: 1b19c957
- gebaut: `GetByQuoteID` (postgres_repository.go:446) und `LinkTimeTracking` (:472) mit sechs
  neuen real-SQL-Tests in `postgres_repository_quote_link_time_tracking_db_test.go` untersucht.
  ECHTER FUND (dokumentiert, NICHT gefixt — siehe unten): `GetByQuoteID`s Signatur gibt genau eine
  Rechnung zurück, aber weder das Schema (keine UNIQUE-Constraint auf
  `finance_invoices.source_quote_id`) noch `invoice.Service.CreateFromQuote` (service.go:779,
  aufgerufen von der `ConvertQuoteToInvoice`-RPC, server/biz_grpc.go:506) garantieren, dass ein
  Angebot höchstens eine Rechnung hat — `CreateFromQuote` prüft ausschließlich
  `quote.Status == accepted`, ruft `GetByQuoteID` nirgends auf. Ein zweiter Aufruf (Doppelklick,
  Netzwerk-Retry) legt anstandslos eine zweite volle Rechnung für dasselbe Angebot an.
  `TestPostgresRepository_GetByQuoteID_MultipleInvoices` beweist das konkret: zwei Rechnungen mit
  identischem `source_quote_id` lassen sich anlegen, `GetByQuoteID` liefert danach ohne Fehler
  eine der beiden zurück (kein `ORDER BY`, welche ist nicht definiert). Dieselbe Fehlerklasse wie
  der in Lauf 10 gefixte Zeiterfassungs-Doppelabrechnungs-Bug, nur auf der Quote-Seite — als
  eigene Unit `fix-quote-to-invoice-duplicate-creation` ans Backlog-Ende gehängt (siehe
  neue-units), NICHT hier gefixt: die Fachfrage, ob Teilrechnungen zu einem Angebot irgendwo
  vorgesehen sind, ist ungeklärt (Grep über `internal/biz/invoice` und `internal/biz/quote`
  findet keinen Hinweis auf partielle Fakturierung — kein "remaining amount", keine
  Positions-Zuordnung —, aber das beweist nicht, dass die Absicht fehlt) und ein Fix müsste diese
  Frage vor der Sperre beantworten, nicht raten.
  Übrige `done_when`-Punkte belegt: zweiter `LinkTimeTracking`-Aufruf überschreibt (nicht
  Merge/Append) statt anzuhängen; ungültiges JSON wird von Postgres selbst als jsonb-Typfehler
  abgelehnt, die Spalte bleibt dabei unverändert (kein teilweise angewandter Write); Fremdtenant-
  Aufrufe beider Methoden betreffen null Zeilen; `GetByQuoteID` mit Fremdtenant liefert
  `ErrInvoiceNotFound` (Tenant-Scoping hängt an der eigenen `tenant_id` der Rechnung, nicht an der
  des Angebots); `GetByQuoteID` befüllt `LineItems` wie `GetByID` (derselbe
  `loadInvoiceLines`/`marshalLineItems`-Pfad).
- gate: build ok (`./internal/biz/invoice/... ./internal/gateway/...`) | vet ok | lint ok
  (0 issues) | test ok (`go test -count=1 -v ./internal/biz/invoice/`, DATABASE_URL gegen
  kmuhub_app, 0 SKIP, 129 PASS inkl. 6 neuer DB-Tests) | test ok
  (`go test -count=1 -p 1 ./internal/biz/invoice/...`) | test ok (`go test -count=1
  ./internal/gateway/` — keine Routenänderung, trotzdem pflichtgemäß gelaufen) | migration n.a.
  (keine Schema-Änderung) | rls-smoke n.a. (keine Tabellen-/Policy-Änderung, reine Lesart-/
  Schreibart-Untersuchung bestehender Spalten)
- coverage: internal/biz/invoice 59,9 % (eigene Messung vor dieser Iteration, `go tool cover
  -func`, stimmt mit Iteration-31-Endwert überein) -> 61,4 % (eigene Messung nach den sechs neuen
  Tests, gleiche Methode, via `git stash`/`pop` der eigenen Testdatei isoliert)
- mutations-probe: gegen eine `cp`-Sicherungskopie (nicht `git checkout`), zurückgeschrieben,
  finaler `diff` gegen die Kopie identisch (0 Zeilen Unterschied), `git diff --stat` auf
  `postgres_repository.go` danach leer. (a) `LinkTimeTracking`s UPDATE um
  `AND time_tracking_source IS NULL` ergänzt (zweiter Aufruf würde still nichts mehr schreiben) ->
  `TestPostgresRepository_LinkTimeTracking_PersistsAndSecondCallOverwrites` rot ("emp-1"/120 statt
  "emp-2"/45 — der zweite Schreibvorgang griff nicht). Anmerkung: eine erste Mutation (die
  `AND tenant_id = $3`-Klausel aus dem UPDATE entfernt, um den Fremdtenant-Test zu prüfen) blieb
  GRÜN — RLS greift hier als zweite, unabhängige Schranke unter `kmuhub_app` und blendet fremde
  Tenant-Zeilen bereits vor der WHERE-Klausel aus; das ist kein blinder Fleck der Probe, sondern
  zeigt, dass die explizite SQL-Klausel bewusst redundant zur RLS-Policy ist (Defense-in-Depth) —
  deshalb Wechsel auf eine Mutation, die NICHT von RLS abgefangen wird. (b) `GetByQuoteID`s
  `loadInvoiceLines`/`marshalLineItems`-Block entfernt (nur noch das nackte `scanInvoice`) ->
  `TestPostgresRepository_GetByQuoteID_ReturnsInvoiceWithLineItems` rot ("unexpected end of JSON
  input" beim Unmarshal von `LineItems`, weil das Feld leer blieb).
- verify vorgaenger: sauber — `a2ea5ab5` (Iteration 31) fügt zwei `inv.LockedAt != nil`-Guards in
  `payment/service.go` plus real-SQL-Tests für `UpdateStatus`/`UpdateStatusInTx` hinzu (per
  `git show --stat` und vollständigem Diff von `service.go` geprüft). Keine der acht
  Fehlerklassen einschlägig: kein neuer gRPC-Handler (reine Service-interne Guards), kein
  Stub/TODO/Unimplemented, kein `.proto`, keine Migration, kein neuer `RequirePermission`-Guard,
  keine neue Tabelle, keine Route, keine Response-Form geändert, kein ersetzter Guard-Key.
- neue-units: `fix-quote-to-invoice-duplicate-creation` (ans Backlog-Ende, deps: [], Block 4) —
  echter, verifizierter Produktionsbug (fehlende Duplikat-Sperre bei Quote-zu-Rechnung-Konversion),
  außerhalb des Scopes dieser Coverage-Unit und mit einer offenen Fachfrage (Teilrechnungen
  vorgesehen?), die vor einem Fix beantwortet werden muss.
- offen: (1) DB-Gate lief vollständig: DATABASE_URL als kmuhub_app, 0 übersprungene Tests im
  Paket. (2) Fachliche Randnotiz für Luke: `LinkTimeTracking` wird produktiv nur genau einmal pro
  Invoice aufgerufen, direkt nach `Create` (server/biz_grpc.go:2118, Audit-Trail für die
  Zeiterfassungs-Rechnung) — das "überschreibt statt merged"-Verhalten ist heute folgenlos, aber
  falls `LinkTimeTracking` künftig für Nachträge (weitere Zeiteinträge auf eine bestehende
  Rechnung) wiederverwendet wird, geht die erste Charge stillschweigend verloren. (3) Die neue Unit
  `fix-quote-to-invoice-duplicate-creation` braucht vor dem Bauen eine Antwort auf die
  Teilrechnungs-Frage — falls Luke die aus Produktkenntnis sofort beantworten kann, spart das der
  nächsten Iteration die Recherche.

## Iteration 33 — cov-creditnote-repository-remaining-real-sql — done — 2026-08-23 02:45
- commit: 29e982ed
- gebaut: `postgres_repository_get_and_update_real_sql_test.go` (neu, ungetaggt, `package
  creditnote`) mit vier real-SQL-Tests. SCOPE-KORREKTUR gegenüber der Unit-Beschreibung: die
  Repository-Schnittstelle (`repository.go`) hat weder `UpdateStatus` noch `Delete` — Statuswechsel
  laufen ausschließlich über `Update`/`UpdateInTx` (voller Zeilen-Replace), ein Hard-Delete existiert
  für Gutschriften nicht. Die Nummernvergabe (`NextNumberInTx`) ist eine injizierte, in
  `internal/biz/quote` implementierte Abhängigkeit, kein Repository-Code dieses Pakets. Beide Punkte
  sind daher als "nicht anwendbar, weil nicht existent" behandelt statt erzwungen nachgebaut.
  Katalog der bereits per `//go:build integration` (testcontainer-gestützt, `pgtc`) abgedeckten Fälle
  — NICHT gedoppelt:
  - `integration_test.go`: `TestCreditNoteLinesRelationalRoundtrip` (GetByID mit Zeilen),
    `TestCreditNoteUpdateReplacesLines` (Update löscht+fügt Zeilen neu ein),
    `TestCreditNoteTenantIsolation` (RLS blockt fremden Tenant bei GetByID).
  - `repository_coverage_integration_test.go`: `TestList_FiltersPaginationAndTenantScoping`,
    `TestGetByInvoiceID_TenantScopedAndOrderedDescending`,
    `TestListForDATEVExport_DateRangeStatusAndKeysetPaging`,
    `TestCreate_AmountNotValidatedAgainstInvoiceOpenBalance` (die im `done_when` verlangte Frage
    "Gutschrift über Rechnungsbetrag hinaus" — bereits beantwortet: Create prüft das nicht, bewusst
    keine Sperre, siehe unten), `TestCreate_DecimalAmountsSurviveDBRoundTripAsExactStrings`
    (Zeilen-Dezimalwerte via Create).
  - `send_atomic_integration_test.go`: `TestCreditNoteSend_AtomicRollback_NumberNotConsumed`
    (Service.Send koppelt Nummernvergabe + UpdateInTx atomar).
  Neu und NICHT vorher abgedeckt (die vier neuen Tests):
  `TestPostgresRepository_GetByID_NotFound` (unbekannte ID im eigenen Tenant ->
  `ErrCreditNoteNotFound`), `TestPostgresRepository_UpdateInTx_RolledBackTransactionPersistsNothing`
  (Statuswechsel + Nummernvergabe in einer zurückgerollten Transaktion hinterlassen nichts),
  `TestPostgresRepository_Update_CrossTenantIsNoop` (Update mit fremder `tenant_id` im Objekt betrifft
  0 Zeilen, kein Fehler — gleiches Muster wie `LinkTimeTracking` bei Invoices),
  `TestPostgresRepository_UpdateInTx_HeaderTotalsSurviveRoundTripAsExactStrings` (Header-Summen
  `subtotal`/`total_tax`/`gross_total` — vom Aufrufer direkt gesetzt, nicht DB-berechnet — überleben
  den Roundtrip exakt, inklusive eines negativen `total_tax`, das `tax.Calculate` nie produzieren
  würde: das Repository validiert Vorzeichen/Bereich hier nicht).
  Die Frage "Gutschrift über den Rechnungsbetrag hinaus" (bereits durch
  `TestCreate_AmountNotValidatedAgainstInvoiceOpenBalance` belegt) zusätzlich als Kommentar direkt am
  Code verankert: `postgres_repository.go` `Create`-Funktion trägt jetzt einen Doc-Kommentar, der auf
  den Test verweist und das Verhalten als bewusst (nicht als Bug) festhält.
- gate: build ok (`./internal/biz/creditnote/... ./internal/gateway/...`) | vet ok | lint ok
  (0 issues) | test ok (`go test -count=1 -v ./internal/biz/creditnote/`, DATABASE_URL gegen
  kmuhub_app, 22 PASS, 0 SKIP) | test ok (`go test -count=1 ./internal/biz/creditnote/...`) | test ok
  (`go test -count=1 ./internal/gateway/` — keine Routenänderung, trotzdem pflichtgemäß gelaufen) |
  migration n.a. (keine Schema-Änderung) | rls-smoke n.a. (keine Tabellen-/Policy-Änderung)
- coverage: internal/biz/creditnote 28,2 % (eigene Messung vor dieser Iteration, neue Testdatei
  temporär entfernt, `go tool cover -func`, stimmt mit `coverage_start:` überein) -> 49,5 % (eigene
  Messung nach den vier neuen Tests, gleiche Methode)
- mutations-probe: gegen eine `cp`-Sicherungskopie (nicht `git checkout`) zurückgeschrieben, finaler
  `diff` gegen die Kopie identisch (0 Zeilen Unterschied). (a) `scanCreditNote`s Mapping von
  `pgx.ErrNoRows` auf `ErrCreditNoteNotFound` durch ein bloßes Durchreichen von `pgx.ErrNoRows`
  ersetzt -> `TestPostgresRepository_GetByID_NotFound` rot (`errors.Is`-Kette enthält
  `ErrCreditNoteNotFound` nicht mehr). (b) in `UpdateInTx`s `Exec`-Aufruf die Argumente für
  `total_tax = $10` und `gross_total = $11` vertauscht (Parameter-Reihenfolge im SQL unverändert
  gelassen, nur die übergebenen Werte getauscht) ->
  `TestPostgresRepository_UpdateInTx_HeaderTotalsSurviveRoundTripAsExactStrings` rot (beide Werte
  seitenverkehrt zurückgelesen).
- verify vorgaenger: sauber — `1b19c957` (Iteration 32) fügt ausschließlich eine neue, ungetaggte
  Testdatei plus Backlog/Journal-Änderungen hinzu (per `git show --stat` geprüft: nur
  `postgres_repository_quote_link_time_tracking_db_test.go` + `.planning/`-Dateien, keine
  Produktionsdatei). Keine der acht Fehlerklassen einschlägig: kein neuer Handler, kein Stub/TODO,
  kein `.proto`, keine Migration, kein neuer `RequirePermission`-Guard, keine neue Tabelle, keine
  Route, keine Response-Form geändert, kein ersetzter Guard-Key. Test-Inhalt selbst gelesen (235
  Zeilen) — echte `testutil.SkipIfNoDB`/`PoolFromEnv`-Tests, keine leeren Stubs.
- neue-units: keine — die einzige während dieser Iteration entdeckte Abweichung (Backlog-Scope nennt
  `UpdateStatus`/`Delete`, die es nicht gibt) ist keine Produktionslücke, sondern eine ungenaue
  Backlog-Beschreibung; kein Fix-/Coverage-Bedarf daraus.
- offen: (1) DB-Gate lief vollständig: DATABASE_URL als kmuhub_app, 0 übersprungene Tests im Paket.
  (2) `internal/biz/creditnote`-Coverage bleibt trotz 28,2 % -> 49,5 % noch unter dem 60 %-Zielband für
  kritische Pfade (Payments/Finance) — `Service.Send`/`Service.StornoInvoice` sind bereits über
  `service_test.go` (gemockt) und die drei getaggten Integrationstests atomar geprüft; verbleibende
  Lücken liegen laut `go tool cover -func` primär in `List`/`ListForDATEVExport`s Query-Building-
  Verzweigungen (Filter-Kombinationen) — Kandidat für eine weitere, engere Coverage-Unit, falls Lauf 9
  das priorisiert.

## Iteration 34 — cov-banking-repository-real-sql — done — 2026-08-23 02:53
- commit: e3b9984d
- gebaut: `postgres_repository_statements_and_matches_real_sql_test.go` (neu, ungetaggt,
  `package banking`) mit sieben real-SQL-Tests gegen die zuvor ungedeckten
  Repository-Methoden: `GetStatementByHash` (Treffer, Fremdtenant mit identischem Hash,
  Nichttreffer -> ErrStatementNotFound, Opening/Closing-Balance-Roundtrip inkl. negativem
  Saldo), `GetStatement` (Fremdtenant-ID -> ErrStatementNotFound), `ListStatements`
  (Pagination, total zaehlt nur den eigenen Tenant, leerer Tenant -> `[]` nicht `null`),
  `ListTransactionsByStatement` (Sortierung nach value_date aufsteigend, Fremdtenant liest
  0 Zeilen statt Fehler), `UpdateTransactionMatch` (schreibt Reconciliation-Spalten, laesst
  die vom-Bank-gemeldeten Spalten unangetastet, Fremdtenant-Aufruf ist ein Noop mit
  ErrTransactionNotFound und ohne Seiteneffekt auf die Zeile), und die Kernfrage der Unit:
  `CreateStatement` mit doppeltem `content_hash` im selben Tenant — die
  UNIQUE-Constraint (tenant_id, content_hash) greift zuverlaessig (0 Duplikate, 0 verwaiste
  Transaktionszeilen dank Rollback in einer Transaktion), aber `CreateStatement` mappt den
  23505-Fehler NICHT auf einen Sentinel (anders als `CreateAccount`/`isAccountIBANConflict`)
  — der Aufrufer bekommt einen rohen pgconn-Fehler durchgereicht. Das ist Fund 1, siehe
  `neue-units`.
  Scope-Praezisierung ueber die vorhandenen Integrationstests hinaus (nicht dupliziert):
  `integration_transactions_test.go`/`integration_accounts_test.go` decken bereits
  GetTransaction/ListTransactions/FindInvoiceIDByNumber und den Account-Pfad ab — dort nicht
  angefasst.
- gate: build ok | vet ok | lint ok (0 issues, inkl. `rangeint`-Hinweis behoben) | test ok
  (`go test -count=1 -v ./internal/biz/banking/`, DATABASE_URL gegen kmuhub_app, 0 SKIP) |
  test ok (`go test -count=1 ./internal/biz/banking/...`) | test ok
  (`go test -count=1 ./internal/gateway/` — keine Routenaenderung, pflichtgemaess gelaufen) |
  migration n.a. | rls-smoke n.a. (keine Tabellen-/Policy-Aenderung; RLS auf
  finance_bank_statements/-transactions besteht bereits, per Fremdtenant-Faellen in jedem
  neuen Test mitprobiert, keine Luecke gefunden)
- coverage: internal/biz/banking 77,0 % (eigene Messung vor dieser Iteration, `go tool
  cover -func`, stimmt mit `coverage_start:` ueberein) -> 83,5 % (nach den sieben neuen
  Tests, gleiche Methode). Ungedeckte Methoden vor dem Schreiben aufgelistet (Pflicht laut
  Unit-Notes): GetStatementByHash/GetStatement/ListStatements/ListTransactionsByStatement/
  UpdateTransactionMatch/scanStatement/parseDecimalPtr waren bei 0,0 %, jetzt alle ueber
  70 % (ListStatements 78,6 %, CreateStatement 81,8 %, ListTransactionsByStatement 81,8 %,
  UpdateTransactionMatch 83,3 %, scanStatement 72,7 %, parseDecimal 66,7 % — Restluecken sind
  Fehlerzweige der `pool.Query`/`rows.Err()`-Aufrufe, die ohne DB-Fault-Injection nicht
  erreichbar sind).
- mutations-probe: gegen eine `cp`-Sicherungskopie zurueckgeschrieben, finaler `diff` gegen
  die Kopie identisch (0 Zeilen Unterschied). (a) `scanStatement`s
  `errors.Is(err, pgx.ErrNoRows)`-Zweig mit `false &&` deaktiviert ->
  `TestPostgresRepository_GetStatementByHash_FoundNotFoundAndTenantScoped/an_unknown_hash_is_not_found`
  rot (Fehler kommt als "scan statement: no rows in result set" statt ErrStatementNotFound
  durch). (b) `ListStatements`s `ORDER BY created_at DESC` auf `ASC` gedreht ->
  `TestPostgresRepository_ListStatements_PaginationTotalAndTenantScoping` rot auf beiden
  Seiten (Seite 1 und Seite 2 liefern die falschen IDs). Nebenbefund aus einem verworfenen
  dritten Versuch: eine `WHERE tenant_id = $1 OR true`-Mutation auf ListStatements blieb
  GRUEN, weil die RLS-Policy auf `finance_bank_statements` die Fremdtenant-Zeilen ohnehin
  herausfiltert, unabhaengig vom App-Praedikat — Defense-in-Depth bestaetigt, aber als
  Mutation fuer diese Probe ungeeignet, deshalb verworfen zugunsten von (b).
- verify vorgaenger: sauber — `29e982ed` (Iteration 33) aendert an Produktionscode nur einen
  6-zeiligen Doc-Kommentar in `creditnote/postgres_repository.go` (per `git show
  -- backend/internal/biz/creditnote/postgres_repository.go` geprueft), sonst ausschliesslich
  eine neue ungetaggte Testdatei plus Backlog/Journal. Keine der acht Fehlerklassen
  einschlaegig: kein neuer Handler, kein Stub/TODO, kein `.proto`, keine Migration, kein
  neuer `RequirePermission`-Guard, keine neue Tabelle, keine Route, keine Response-Form
  geaendert, kein ersetzter Guard-Key.
- neue-units: fix-banking-import-race-returns-raw-pg-error (Block A/Fix-Kandidat, ans
  Backlog-Ende gehaengt) — `CreateStatement` mappt die Unique-Violation auf
  `finance_bank_statements_hash_unique` nicht auf einen Sentinel, anders als
  `isAccountIBANConflict` im selben Paket; `Service.Import` gibt bei einem echten
  Gleichzeitigkeits-Race (zwei Uploads derselben Datei fast zeitgleich) daher einen rohen
  500 statt der erwarteten "bereits importiert"-Antwort zurueck. Kein Datenverlust, keine
  doppelte Buchung (Constraint haelt, per Test belegt) — nur eine haessliche Fehlerantwort
  in einem seltenen Fenster.
- offen: (1) DB-Gate lief vollstaendig: DATABASE_URL als kmuhub_app, 0 uebersprungene Tests
  im Paket. (2) `internal/biz/banking` liegt mit 83,5 % ueber dem allgemeinen 15 %-Gate, aber
  unter dem 60 %-Zielband fuer kritische Pfade — Restluecken sind laut `go tool cover -func`
  primaer die Fehlerzweige in `CreateStatement`/`ListStatements`/`ListTransactionsByStatement`
  (DB-Fault-Injection noetig, kein Kandidat fuer eine einfache Folge-Unit) und
  `matcher.go`/`postgres_repository_accounts.go`, die die naechste Unit
  `cov-banking-accounts-and-matcher-real-sql` bereits als deps-Nachfolger aufgreift.

## Iteration 35 — cov-banking-accounts-and-matcher-real-sql — done — 2026-08-23 03:01
- commit: 66972511
- gebaut: `matcher_test.go` (129 -> 147 Zeilen) deckt jetzt auch den Waehrungs-Default
  (`entryCurrency`/`itemCurrency` fielen leer auf "EUR" zurueck, bislang ungetestet):
  `TestMatchEntryEmptyCurrencyDefaultsToEUR` schickt einen Entry ohne Currency-Feld und ein
  OpenItem mit explizitem "EUR" durch den AMOUNT-ONLY-Pfad (Verwendungszweck traegt bewusst
  keine Rechnungsnummer, sonst haette Pass eins den Fund maskiert — siehe Mutations-Probe-Notiz
  unten). Alle uebrigen von `matcher_test.go` bereits abgedeckten Regeln aufgelistet (Pflicht
  laut Unit-Notes): Nummer+Betrag, Reformatierung des Verwendungszwecks, Teilzahlung (Nummer
  ohne Betrag), Betrag-only bei Eindeutigkeit, Mehrdeutigkeit bei gleichem Betrag (Nullbefund:
  kein Zwang zur Zuordnung), Sammelzahlung ueber zwei Rechnungen (kein Split), Debits werden
  ignoriert, Waehrungen werden nicht gemischt, zu kurze Rechnungsnummern (< 4 Zeichen) nur ueber
  die Betragsregel erreichbar — alle neun bereits vorhanden, keine Dublette gebaut.
  `integration_accounts_test.go` (+117 Zeilen, drei neue Tests) schliesst die Luecken, die die
  bestehenden Fake-Repo-Tests strukturell nicht erreichen koennen, weil der Fake nie SQL
  anfasst: `TestPostgresRepository_UpdateAccount_NotFoundAndIBANConflict` (Update auf eine
  nicht existierende ID -> ErrAccountNotFound statt stillem No-op; Update auf die IBAN eines
  anderen Kontos -> ErrAccountExists statt stillem Merge, Zeile bleibt nachweislich unveraendert),
  `TestPostgresRepository_DeleteAccount_RemovesTheRow` (Zeile ist nach Delete wirklich weg, IBAN
  ist danach wieder frei fuer eine Neuanlage — ein haengender Unique-Index-Eintrag haette das
  verhindert), `TestService_ListAccounts` (Service.ListAccounts hatte 0,0 % Coverage — bislang
  wurde in jedem Test entweder das Repository oder der Fake direkt gelesen, nie der
  Service-Passthrough selbst aufgerufen).
  Den geforderten Fall "Rechnungsnummer aus fremdem Mandanten fuehrt zu keiner Zuordnung" NICHT
  im Matcher nachgebaut: `MatchEntry` bekommt seine `openItems` ausschliesslich ueber das
  injizierte `OpenItemReader`-Interface (`repository.go:98`), das in Produktion von
  `invoice.PostgresRepository.ListOpenItems` erfuellt wird — tenant-scoped per SQL, bereits
  belegt in `internal/biz/invoice/postgres_open_items_and_chains_integration_test.go:164`
  ("Cross-tenant noise: another tenant's overdue invoice must never surface"). Ein fremder
  Mandant erreicht den Matcher in Produktion also strukturell nie; ein Test dagegen im
  `banking`-Paket haette nur den Fake erneut geprueft (der ohnehin zurueckgibt, was man ihm
  gibt) und waere keine zweite Zusicherung, sondern eine Dublette der bestehenden invoice-Probe.
  Das ist der ehrliche Abschluss dieses `done_when`-Punkts, keine Umgehung.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (`go test -count=1 -v
  ./internal/biz/banking/`, DATABASE_URL gegen kmuhub_app, 55 PASS, 0 SKIP) | test ok
  (`go test -count=1 ./internal/gateway/` — keine Routenaenderung, pflichtgemaess gelaufen) |
  migration n.a. | rls-smoke n.a. (keine Tabellen-/Policy-Aenderung; Tenant-Trennung auf
  `finance_bank_accounts` per Fremdtenant-Faellen in den neuen Tests mitgeprueft, keine Luecke)
- coverage: internal/biz/banking 83,5 % (eigene Messung vor dieser Iteration, `go tool cover
  -func`, deckt sich mit dem Lauf-10-CI-Wert 77,0 % nicht mehr — Iteration 34 hat das Paket
  bereits auf 83,5 % gehoben, der `coverage_start:` der Unit war der aeltere CI-Stand) -> 84,3 %
  (nach den vier neuen Tests, gleiche Methode, zweimal reproduziert). Reine Passthrough-Methode
  `service_accounts.ListAccounts` sprang von 0,0 % auf 100 %.
- mutations-probe: drei Laeufe, jeweils gegen eine `cp`-Sicherungskopie (nicht `git checkout`),
  zurueckgeschrieben, finaler `diff` gegen die Kopie danach identisch (0 Zeilen Unterschied).
  (a) `isAccountIBANConflict`-Zweig in `UpdateAccount` entfernt -> "moving onto another
  account's IBAN is refused" rot (roher pgconn-Fehler statt ErrAccountExists durchgereicht).
  (b) `tag.RowsAffected() == 0`-Check in `UpdateAccount` durch `_ = tag` ersetzt -> "a row that
  does not exist is reported" rot (kein Fehler statt ErrAccountNotFound). (c) `entryCurrency`s
  Default-Zweig mit `if false` stillgelegt -> `TestMatchEntryEmptyCurrencyDefaultsToEUR` rot
  (Status unmatched statt suggested). Nebenbefund zur eigenen Arbeitsweise: die erste Fassung
  von Probe (c) verwendete einen Verwendungszweck MIT Rechnungsnummer im Test-Fixture — die
  Mutation blieb damit GRUEN, weil Pass eins (Nummer im Verwendungszweck) den Treffer unabhaengig
  von der Waehrung liefert und nur `Reason` sich unterschieden haette, nicht `Status`. Fixture
  danach auf einen referenzlosen Verwendungszweck umgebaut, seither diskriminierend.
- verify vorgaenger: sauber — `e3b9984d` (Iteration 34) aendert laut `git show --stat`
  ausschliesslich `.planning/`-Dateien und eine neue ungetaggte Testdatei
  (`postgres_repository_statements_and_matches_real_sql_test.go`), keine Produktionsdatei.
  Keine der acht Fehlerklassen einschlaegig: kein neuer Handler, kein Stub/TODO, kein `.proto`,
  keine Migration, kein neuer `RequirePermission`-Guard, keine neue Tabelle, keine Route, keine
  Response-Form geaendert, kein ersetzter Guard-Key.
- neue-units: keine
- offen: (1) DB-Gate lief vollstaendig: DATABASE_URL als kmuhub_app, 0 uebersprungene Tests im
  Paket. (2) `internal/biz/banking` liegt mit 84,3 % weiterhin ueber dem 15 %-Gate, aber unter
  dem 60 %-Zielband fuer kritische Pfade — die verbleibenden Luecken sind laut `go tool cover
  -func` durchgaengig Fehlerzweige der `pool.Query`/`pool.Exec`/`rows.Err()`-Aufrufe in
  `postgres_repository.go`/`postgres_repository_accounts.go`, die ohne DB-Fault-Injection nicht
  erreichbar sind — kein Kandidat fuer eine einfache Folge-Unit. (3) Block B ist NICHT
  abgeschlossen: `cov-banking-mt940-camt053-parser-edge-cases`, `cov-expense-repository-real-sql`
  und `cov-recurring-generation-run-real-sql` stehen weiterhin auf `todo`, alle drei `deps: []`
  und damit sofort ziehbar.

## Iteration 36 — cov-banking-mt940-camt053-parser-edge-cases — done — 2026-08-23 03:16
- commit: 4ba17f10
- gebaut: `parse_test.go` (+128 Zeilen, sieben neue Tests) deckt die Luecken, die
  `mt940.go`/`camt053.go`/`parse.go` bislang unbelegt liessen. Bereits vorhandene Faelle (Pflicht
  laut Unit-Notes, keine Dublette gebaut): Vorzeichen (CR/DR, CRDT/DBIT, RC/RD-Storno) fuer beide
  Formate, Dezimaltrennzeichen (Komma bei MT940), Datumsformat (YYMMDD inkl. Jahreswechsel-Buchung
  bei MT940, ISO-Datum bei CAMT), leerer/zu grosser/abgeschnittener/formatloser Upload, BOM-Erkennung.
  Neu: (1) `TestParseMT940WrappedInformationField` — ein :86:-Feld, das mitten im Token auf eine
  zweite physische Zeile umbricht (reales Bankverhalten bei fixer Zeilenlaenge); deckt den
  Continuation-Zweig in `splitMT940Fields` erstmals ab (81,8% -> 100%). (2)
  `TestParseCAMT053JoinsMultipleUnstructuredRemittanceLines` — eine Sammelzahlung mit zwei
  `<Ustrd>`-Elementen in einer TxDtls, muessen mit Leerzeichen zusammengefuegt werden. (1)+(2)
  zusammen erfuellen den Pflichtpunkt "Mehrzeilen-Verwendungszweck fuer beide Formate belegt".
  (3) `TestParseMT940RejectsUnparseableBookingDateWithoutFailingTheEntry` — ein MMDD-Buchungsdatum,
  das kein echtes Kalenderdatum ist (29.02. in einem Format ohne Jahr), dokumentiert bestehendes
  Verhalten: die Datei wird NICHT verworfen, nur `BookingDate` bleibt leer (`ValueDate` bleibt
  unabhaengig korrekt). Kein Fund, sondern Beleg fuer eine bewusste Entwurfsentscheidung.
  (4) `TestParseRejectsTooManyEntries` — `ErrTooManyEntries` war komplett ungetestet (`Parse`
  92,9% -> 100%), erzeugt 20001 MT940-Buchungszeilen und prueft die Ablehnung. (5)
  `TestParseCAMT053FallsBackToBookingDateWhenValueDateMissing` und (6)
  `TestParseCAMT053RejectsEntryWithoutAnyUsableDate` — der Buchungsdatum-Fallback (manche Banken
  lassen `ValDt` bei Gebuehrenpositionen weg) UND der Fehlerfall ganz ohne Datum waren beide
  ungetestet (`camtEntryToParsed` 82,4% -> 94,1%). (7) `TestParseCAMT053RejectsEntryWithEmptyAmount`
  — ein `<Amt>` ohne Wert muss `ErrMalformed` liefern statt eine 0,00-Buchung zu erzeugen
  (`camtSignedAmount` 81,8% -> 90,9%). Keine Kontoauszugsdaten echter Banken verwendet, alle
  Fixtures selbst geschrieben.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (`go test -count=1 ./internal/biz/banking/`,
  0 SKIP) | test ok (`go test -count=1 ./internal/gateway/` — keine Routenaenderung, pflichtgemaess
  gelaufen; ein isolierter Flake in `TestDecodeBexioState_ManipulatedSignature` beim ersten Lauf,
  danach zweimal in Folge gruen und auch dreimal isoliert gruen — Vorbestand, nicht mit `banking`
  oder dieser Unit verbunden, siehe offen) | migration n.a. | rls-smoke n.a. (reine Parser-Tests,
  keine DB-Beruehrung)
- coverage: internal/biz/banking 84,3 % (eigene Messung vor dieser Iteration, `go tool cover -func`)
  -> 85,6 % (danach, gleiche Methode). Einzelfunktionen: `splitMT940Fields` 81,8% -> 100%,
  `Parse` 92,9% -> 100%, `camtEntryToParsed` 82,4% -> 94,1%, `camtSignedAmount` 81,8% -> 90,9%,
  `parseMT940BookingDate` 75% -> 87,5%.
- mutations-probe: drei Laeufe, jeweils gegen eine `cp`-Sicherungskopie (nicht `git checkout`,
  Iteration-3-Lehre), zurueckgeschrieben, `diff` gegen die Kopie danach jedes Mal identisch (0
  Zeilen Unterschied). (a) Continuation-Append in `splitMT940Fields` mit `if false && ...`
  stillgelegt -> `TestParseMT940WrappedInformationField` rot ("Zahlung Rechn" statt "Zahlung
  Rechnung RE-2026-0001" — die zweite physische Zeile geht verloren). (b) Booking-Date-Fallback in
  `camtEntryToParsed` entfernt -> `TestParseCAMT053FallsBackToBookingDateWhenValueDateMissing` rot
  ("entry without a usable date" statt erfolgreichem Parse). (c) `ErrTooManyEntries`-Pruefung in
  `Parse` geloescht -> `TestParseRejectsTooManyEntries` rot (kein Fehler statt der Obergrenze).
- verify vorgaenger: sauber — `dd276091` (letzter Commit vor dieser Iteration) aendert laut
  `git show --stat` ausschliesslich `.planning/loop/JOURNAL.md` (SHA-Nachtrag Iteration 35), kein
  Codewechsel, keine der acht Fehlerklassen anwendbar.
- neue-units: keine
- offen: (1) `TestDecodeBexioState_ManipulatedSignature` in `internal/gateway` fiel beim ersten
  vollen Gate-Lauf dieser Iteration einmal durch ("expected error for tampered signature, got nil"),
  bestand aber isoliert dreimal in Folge und im vollen Paket zweimal in Folge danach — sieht nach
  Test-Reihenfolge-/Zustandsleck in `internal/gateway` aus (bexio ist ohnehin gesperrte Flaeche in
  diesem Lauf), kein Zusammenhang mit `banking` oder dieser Unit erkennbar. Verdient eine eigene
  Untersuchung, falls es sich wiederholt. (2) `internal/biz/banking` bleibt bei 85,6 % oberhalb des
  15%-Gates, unterhalb des 60%-Kritischer-Pfad-Ziels — Restluecken sind laut `go tool cover -func`
  ueberwiegend `pool.Query`/`pool.Exec`-Fehlerzweige (DB-Fault-Injection noetig) plus
  `parseMT940Balance`/`parseMT940BalanceDate` (66,7%/71,4%, ungetestete Fehlerpfade bei
  kaputten `:60F:`/`:62F:`-Zeilen — kein Fund, das Verhalten dort ist bereits "ignorieren statt
  Datei verwerfen", siehe Code-Kommentar). (3) Block B ist damit vollstaendig abgearbeitet:
  `cov-expense-repository-real-sql` und `cov-recurring-generation-run-real-sql` sind die
  verbleibenden `todo`-Units mit `deps: []`, beide sofort ziehbar.

## Iteration 37 — cov-expense-repository-real-sql — done — 2026-08-23 03:16
- commit: 500a849f
- gebaut: `integration_test.go` (+174 Zeilen, vier neue Tests) deckt die Luecken der
  Real-SQL-Pfade in `postgres_repository.go`, die bisher nur teilweise oder gar nicht ueber
  echtes Postgres liefen. (1) `TestPostgresRepository_DeleteRemovesOwnRow` — der Erfolgspfad
  von `Delete` (`return nil` nach `RowsAffected() > 0`) war ungetestet; bisher gab es nur den
  Cross-Tenant-Fall (`ErrNotFound`). (2) `TestPostgresRepository_UpdateMissingOrForeignRowReturnsNotFound`
  — `Update` wurde bislang nur ueber `Approve`/`Reject` (immer derselbe Tenant, immer ein
  existierender Datensatz) real gegen SQL getestet; ein direkter `repo.Update()` auf eine fremde
  oder nicht existierende ID war ungedeckt (Subtests fuer beide Faelle). (3)
  `TestPostgresRepository_CreateRejectsNonPositiveAmount` — das Repository hat keine eigene
  Betragspruefung, verlaesst sich auf den CHECK-Constraint `finance_expenses_amount_positive` als
  letzte Verteidigungslinie unterhalb der Service-Validierung (`parseAmount`); Test belegt, dass
  ein direkter `repo.Create()` mit negativem Betrag tatsaechlich abgelehnt wird und der Fehler
  NICHT als `ErrNotFound` fehlinterpretiert wird (deckt zugleich den generischen Fehlerzweig in
  `scanExpense`). (4) `TestPostgresRepository_ListRespectsOffset` — alle bisherigen `List`-Tests
  liefen mit `Offset: 0`; neuer Test mit drei Zeilen unterschiedlichen Datums prueft Seite 1
  (`Limit:2, Offset:0`) und Seite 2 (`Limit:2, Offset:2`) inklusive stabilem `total` ueber beide
  Seiten. Keine echten Belegdaten verwendet, alle Fixtures selbst geschrieben.
  Zusaetzlich zwei Recherche-Befunde ohne Code-Aenderung (`done_when` verlangte die Pruefung,
  nicht zwingend einen Fund): `service.go:283` rundet den vom Nutzer eingegebenen Betrag auf zwei
  Stellen (`amount.Round(2)`), das ist reine Normalisierung auf die Spaltenpraezision
  NUMERIC(15,2), keine Steuerberechnung — gehoert fachlich NICHT zur selben Rundungsregel wie
  `fix-tax-rounding-divergence-across-implementations` (dort geht es um USt-Rundungsreihenfolge
  bei Rechnungen/E-Invoicing, hier um einen einzelnen vom Menschen eingegebenen Betrag ohne
  Steueraufteilung). Kein Fund, keine neue Unit noetig. Die Unit-Notes verlangten ausserdem eine
  "Vorsteuerbehandlung mit zwei Steuersaetzen" zu belegen — `finance_expenses` (Migration 000257)
  hat gar kein Steuerfeld (kein Netto/Brutto-Split, kein Steuersatz), `models.Expense` ebenso
  nicht. Der Punkt ist auf dieses Modul nicht anwendbar; vermutlich eine falsche Annahme in den
  urspruenglichen Unit-Notes, kein Bug im Code.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (`go test -count=1 -v ./internal/biz/expense/`,
  0 SKIP, 19 Testfunktionen inkl. Subtests real gegen Postgres gelaufen) | migration n.a. (keine
  Migration in dieser Unit) | rls-smoke n.a. (keine Tabellen-/Policy-Aenderung, RLS bereits von
  Vor-Iterationen abgedeckt) | `go test ./internal/gateway/` nicht gelaufen — keine Route
  angefasst, daher nicht pflichtig
- coverage: internal/biz/expense 77,8 % (eigene Messung vor dieser Iteration, `go tool cover -func`)
  -> 81,5 % (danach, gleiche Methode). Einzelfunktionen: `Create` 83,3% -> 100%, `Update` 83,3% ->
  100%, `Delete` 66,7% -> 83,3%, `List` 80,6% -> 87,1%, `scanExpense` 77,8% -> 88,9%. Verbleibende
  Luecken in `Delete`/`List`/`scanExpense` sind ausschliesslich DB-Fault-Injection-Zweige (Exec-
  bzw. Query-Fehler der Verbindung selbst, `rows.Err()`, ein unparsebarer NUMERIC-Wert aus der
  DB) — ohne kaputte Verbindung oder korrupte Daten nicht real erreichbar, gleiches Muster wie
  bei `internal/biz/banking` in Iteration 36.
- mutations-probe: drei Laeufe, jeweils gegen eine `cp`-Sicherungskopie (nicht `git checkout`),
  zurueckgeschrieben, `diff` gegen die Kopie danach jedes Mal identisch (0 Zeilen Unterschied).
  (a) `Delete`s `RowsAffected() == 0` zu `RowsAffected() >= 0` verfaelscht (immer `ErrNotFound`)
  -> `TestPostgresRepository_DeleteRemovesOwnRow` rot. (b) Erster Mutationsversuch — die Tenant-
  Bedingung in `Update`s WHERE-Klausel von `AND` auf `OR` geaendert, um einen Cross-Tenant-Write
  zu simulieren — schlug NICHT fehl: RLS (`FORCE ROW LEVEL SECURITY`, Policy `tenant_id =
  current_tenant_id()`) blockt den fremden Datensatz bereits auf DB-Ebene, unabhaengig von der
  Anwendungs-WHERE-Klausel. Das ist kein Test-Defekt, sondern der Beweis, dass RLS hier die
  eigentliche Kontrollinstanz ist — die WHERE-Klausel im Code ist defense-in-depth. Zurueckgesetzt,
  stattdessen `scanExpense`s `errors.Is(err, pgx.ErrNoRows)`-Erkennung mit `if false && ...`
  stillgelegt (die von `Update`/`GetByID` gemeinsam genutzte Umwandlung in `ErrNotFound`) ->
  `TestPostgresRepository_UpdateMissingOrForeignRowReturnsNotFound` rot in beiden Subtests
  ("scan expense: no rows in result set" statt `ErrNotFound`). (c) `Offset`-Anwendung in `List`
  mit `if false && filter.Offset > 0` stillgelegt -> `TestPostgresRepository_ListRespectsOffset`
  rot (Seite 2 liefert 2 statt 1 Zeile, weil Offset ignoriert wird).
- verify vorgaenger: sauber — `26d65509` (letzter Commit vor dieser Iteration) aendert laut
  `git show --stat` ausschliesslich `.planning/backend-block/loop/JOURNAL.md` (SHA-Nachtrag
  Iteration 36), kein Codewechsel, keine der acht Fehlerklassen anwendbar.
- neue-units: keine
- offen: (1) `internal/biz/expense` bleibt bei 81,5 % oberhalb des 15%-Gates, unterhalb des
  60%-Kritischer-Pfad-Ziels — Restluecken sind DB-Fault-Injection-Zweige, siehe coverage-Zeile.
  (2) Recherche-Ergebnis ohne Codeaenderung: keine Vorsteuerbehandlung im Ausgaben-Modul
  vorhanden (Migration 000257 hat kein Steuerfeld) — die Unit-Notes gingen davon faelschlich aus,
  kein Fund im Code selbst. (3) `service.go:120` `Get` bleibt bei 0% Coverage (kein Aufrufer im
  Test, weder Fake noch real) — ausserhalb des Scopes dieser Coverage-Unit (Repository, nicht
  Service), aber fuer eine spaetere Service-Coverage-Unit vermerkt, falls sie angelegt wird.

## Iteration 38 — cov-recurring-generation-run-real-sql — done — 2026-08-23 03:23
- commit: 618d6037
- gebaut: `internal/biz/recurring/service_db_test.go` (neu, ungetaggt) verdrahtet
  `recurring.Service` gegen echte Repositories statt des `fakeRepo` aus
  `service_test.go`: eigene `PostgresRepository` plus ein echter `invoice.Service`
  (echtes `invoice.PostgresRepository`, echte `quote.PostgresNumberSequenceRepo`,
  echte `quote.PostgresCompanySettingsRepo`). Fuenf Tests: (1)
  `TestGenerate_RealSQL_ReplaySamePeriodReturnsExistingInvoice` — zweiter Aufruf
  fuer dieselbe, zurueckgedrehte Periode liefert dieselbe Rechnung, kein zweiter
  Insert. (2) `TestGenerate_RealSQL_ConcurrentRunsProduceExactlyOneInvoice` — zwei
  echte Goroutinen (eigene Pool-Connections) rasen per Start-Channel-Barriere auf
  denselben Generate-Aufruf; die ON-CONFLICT-DO-NOTHING-Guard in
  `finance_recurring_runs` haelt, genau eine Rechnung entsteht, der Verlierer
  bekommt entweder dieselbe Rechnungs-ID oder den dokumentierten transienten
  Fehler "is being generated concurrently" (service.go:290) — 15/15 stabil bei
  zwei Racern getestet, s. offen. (3)
  `TestGenerate_RealSQL_EndedAndPausedSchedulesRefuse` — frisch aus Postgres
  geladene pausierte/beendete Zeitplaene lehnen ab. (4)
  `TestGenerate_RealSQL_MonthEndScheduleStaysAnchored` — Zeitplan ab 31.01.,
  zwei echte Generate-Laeufe, NextRun ueber einen echten DATE-Spalten-Roundtrip
  (nicht nur den In-Memory-Struct) 28.02. dann 31.03., zwei verschiedene
  Rechnungen. (5) `TestUpdate_RealSQL_ClearEndDatePersistsNull` — der
  Enddatum-Loeschpfad aus a5f5aa6f setzt end_date wirklich auf NULL in Postgres,
  nicht nur im Rueckgabe-Struct (bisher ungetestet: `postgres_repository_db_test.go`s
  `TestPostgresRepository_Update` setzte nie ein bestehendes EndDate zurueck).
  NEBENFUND (Verhaltensbug, nicht behoben — siehe neue-units): beim Bauen von
  Test (4) schlug ein zweiter `Generate`-Lauf fuer eine ANDERE Periode desselben
  Tenants mit einer rohen Postgres-Fehlermeldung fehl:
  `duplicate key value violates unique constraint "idx_finance_invoices_number"`.
  Ursache: `finance_invoices.invoice_number` ist `NOT NULL DEFAULT ''`
  (migrations/000045), der Unique-Index `idx_finance_invoices_number
  (tenant_id, invoice_number)` ist NICHT partiell — ein Tenant kann also nie
  zwei unversendete Entwuerfe gleichzeitig halten. Workaround in Test (4):
  die Januar-Rechnung wird per `invSvc.Send` versendet, bevor Februar generiert
  wird (entspricht auch dem realistischen Ablauf). `finance_quotes`/
  `finance_credit_notes` haben denselben `DEFAULT ''`-Musterfehler, aber KEINEN
  Unique-Index auf die Nummer (per `pg_indexes` gegen die lokale DB geprueft) —
  nur `finance_invoices` ist betroffen.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (`go test -count=1 -v
  ./internal/biz/recurring/`, 0 SKIP, alle Tests inkl. der fuenf neuen real
  gegen Postgres gelaufen) | migration n.a. (keine Migration in dieser Unit) |
  rls-smoke n.a. (keine Tabellen-/Policy-Aenderung; `finance_recurring_runs`
  und `finance_invoices` tragen bereits FORCE RLS aus fruaheren Migrationen,
  alle Schreib-/Lesepfade liefen unter `testutil.WithTenantCtx`) | `go test
  ./internal/gateway/` nicht gelaufen — keine Route angefasst, nicht pflichtig
- coverage: internal/biz/recurring 88,1 % (eigene Messung vor dieser Iteration,
  neue Testdatei kurz beiseitegelegt) -> 88,5 % (danach, gleiche Methode).
  Kleiner Zugewinn, weil `Generate`/`Update` schon durch `service_test.go`
  (fakeRepo) strukturell fast vollstaendig gedeckt waren — der Wert dieser Unit
  liegt im Beweis gegen echtes SQL (ON-CONFLICT-Race, DATE-Roundtrip,
  NULL-Clearing), nicht in der Statement-Zahl, wie im Scope der Unit
  ausdruecklich vorausgesagt ("Coverage ist das Nebenprodukt").
- mutations-probe: `service.go:286` `if !claimed {` testweise zu `if false &&
  !claimed {` geaendert (Replay-Pfad komplett deaktiviert) — sowohl
  `TestGenerate_RealSQL_ReplaySamePeriodReturnsExistingInvoice` als auch
  `TestGenerate_RealSQL_ConcurrentRunsProduceExactlyOneInvoice` wurden rot
  (beide scheiterten an der echten `idx_finance_invoices_number`-Verletzung,
  weil ohne die Replay-Guard ein zweiter Insert versucht wird — das beweist
  die Guard-Notwendigkeit sogar staerker als ein sauberer Duplikat-Fehler
  es koennte). Zurueckgesetzt via `cp` einer Sicherungskopie, `diff` gegen das
  Original danach leer (0 Zeilen Unterschied).
- verify vorgaenger: sauber — `500a849f` (letzter Code-Commit vor dieser
  Iteration, `d8d2ed6c` war reiner Journal-SHA-Nachtrag) aendert laut
  `git show --stat` nur `internal/biz/expense/integration_test.go` (+173) und
  einen Status-Flip in BACKLOG.yml — keine Produktionslogik, keine der acht
  Fehlerklassen anwendbar.
- neue-units: fix-invoice-number-unique-index-blocks-second-draft (verifizierter
  Produktionsbug: `finance_invoices`-Unique-Index auf (tenant_id,
  invoice_number) ist nicht partiell, blockiert einen zweiten gleichzeitigen
  Entwurf pro Tenant mit einem rohen 500 statt einer klaren Fehlermeldung;
  Fix braucht eine neue Forward-Migration und ist damit ausserhalb des Scopes
  dieser Coverage-Unit)
- offen: (1) Die Concurrency-Probe nutzt bewusst nur 2 Racer (wie im
  Unit-Notes-Text "Zwei gleichzeitige Laeufe" gefordert) statt urspruenglich 5 —
  bei 5 Racern trat das oben genannte invoice_number-Problem sporadisch auf,
  weil ein Racer die schon vollstaendig fortgeschrittene naechste Periode
  erreichte, bevor die anderen ihre Klaim-Anfrage stellten (kein Test-Fehler,
  sondern derselbe Nebenfund). Mit 2 Racern 15/15 stabil, aber nicht 100%
  deterministisch garantiert — CI ist der Beweis fuer Langzeitstabilitaet.
  (2) fix-invoice-number-unique-index-blocks-second-draft steht am Backlog-Ende
  mit model: opus (Migration + Geldpfad) und ist noch nicht eingeplant in eine
  Blockreihenfolge — Luke entscheidet die Prioritaet gegen Block A-D.
  (3) Nach dem Merge von fix-invoice-number-unique-index-blocks-second-draft
  kann der `invSvc.Send`-Workaround in
  TestGenerate_RealSQL_MonthEndScheduleStaysAnchored optional entfernt werden.

## Iteration 39 — cov-gateway-security-audit-and-vault-routes — done — 2026-08-23 03:36
- commit: 556e8d6461976a6a47844a96a673e39f69512595
- gebaut: (1) Router-Guard-Test `TestSecurityRoutes_AuditAndVaultGuards`
  (`route_security_test.go`) fuer alle sieben Routen aus `route_security.go:43-51`
  (Audit-Liste, Audit-Export, Audit-Verify, Vault-Liste, Vault-Get, Vault-Set,
  Vault-Delete): 401 ohne Token (echtes `RequireAuthenticated`, nicht
  `guardTestAuth`), 403 authentifiziert ohne admin-Rolle, 503 admin + leere
  Registry. Dazu direkte Handler-Tests fuer `HandleGetVaultSecret`/
  `HandleDeleteVaultSecret` (503, fehlender `keyName` -> 400) sowie 503-Tests
  fuer die vier Routen ohne Request-Validierung.
  (2) ECHTER BUG gefunden und behoben (root cause, kein Test-Workaround):
  `SecurityGRPCServer.ExportAuditLog` (`security_grpc.go:166-168`) wrappte
  JEDEN Fehler aus `auditService.ExportEntries` als `codes.Internal` — auch
  den Fall "unbekanntes Exportformat", der `audit.Service.ExportEntries`
  intern schon sauber als eigenen Fehler unterschied (`switch format`,
  `default:`). Ein Client mit `?format=xml` bekam dadurch einen blanken
  500 "internal error" statt 400 mit Grund. Fix: neuer Sentinel-Error
  `audit.ErrUnsupportedFormat` (`service.go`), `ExportAuditLog` routet den
  Fehler jetzt durch das etablierte `mapSecurityError` (wie alle anderen
  RPCs in dieser Datei), neuer Case dort mappt `ErrUnsupportedFormat` auf
  `codes.InvalidArgument` -> HTTP 400 via `respondGRPCError`.
  (3) `HandleGetVaultSecret` gibt laut `securityv1.GetVaultSecretResponse.
  decrypted_value` (proto Zeile 154) den Klartext des Secrets zurueck — das
  ist Absicht (admin-only-Route, ein Vault ohne Lesezugriff auf den Wert
  waere fuer Admins nutzlos) und keine Leckage: `ListVaultSecrets` liefert
  nur `VaultSecret` (ohne `decrypted_value`), also nie Klartext in der
  Liste. Keine neue Unit noetig.
  (4) CSV-Formel-Injektion war bereits durch `csvutil.NeutralizeFormulaCell`
  auf Target/Details/UserAgent (`export.go`) und den bestehenden Test
  `TestExportCSV_NeutralizesFormulaInjection` abgedeckt (Lauf 10) — geprueft,
  nichts zu tun.
  (5) Zwei neue Server-Tests in `security_grpc_test.go`:
  `TestExportAuditLog_UnsupportedFormatIsInvalidArgument` (beweist den Fix)
  und `TestVerifyAuditChain_BrokenChainReportsError` (stub-Repo mit
  `chainValid=false`, `firstBad=5`; beweist `Valid=false` +
  `ErrorMessage` gesetzt statt stillem Erfolg).
- gate: build ok | vet ok | lint ok (0 issues, `internal/gateway` +
  `internal/security/audit` + `internal/server`) | test ok (`go test -count=1
  ./internal/gateway/...`, `./internal/security/...`, `./internal/server/...`,
  0 SKIP in allen drei) | migration n.a. (keine Tabellen-/Schema-Aenderung) |
  rls-smoke n.a. (keine Tabellen-/Policy-Aenderung) | `go test -count=1
  ./internal/gateway/` PFLICHT gelaufen (TestOpenAPIRouteDrift gruen, keine
  Route hinzugefuegt/geaendert, nur Guard-Reihenfolge unveraendert)
- coverage: internal/gateway 54,1 % (eigene Messung vor dieser Iteration via
  `git stash`) -> 54,2 % (danach). Kleiner Zugewinn wie bei den anderen
  Handler-Test-Units dieses Blocks erwartet — der Wert liegt im bewiesenen
  Guard-Verhalten (401/403/503) und im gefundenen/behobenen 500-statt-400-Bug,
  nicht in der Statement-Zahl.
- mutations-probe: (1) `security_grpc.go` ExportAuditLog testweise zurueck auf
  `status.Errorf(codes.Internal, "export failed: %v", err)` (alter Zustand) —
  `TestExportAuditLog_UnsupportedFormatIsInvalidArgument` wurde rot (erwartet
  InvalidArgument/0x3, bekam Internal/0xd). Zurueckgesetzt via `cp` einer
  Sicherungskopie, `diff` gegen Original danach leer. (2) `route_security.go`
  `RequireRole("admin")` auf der Audit-Liste-Route testweise entfernt —
  `TestSecurityRoutes_AuditAndVaultGuards/list_audit_entries/authenticated_non-admin`
  wurde rot (503 statt erwarteter 403, weil ohne Guard direkt die leere
  Registry griff). Zurueckgesetzt, `diff` gegen Original danach leer.
- verify vorgaenger: sauber — `618d6037` (letzter Commit vor dieser Iteration)
  aendert laut `git show --stat` nur `internal/biz/recurring/service_db_test.go`
  (neue Testdatei) und BACKLOG.yml — keine Produktionslogik, keine der acht
  Fehlerklassen anwendbar.
- neue-units: keine (der gefundene Bug wurde in dieser Iteration direkt
  behoben, siehe gebaut (2) — kein Deploy-Hazard, keine neue Route, kein neuer
  Guard, keine Migration noetig)
- offen: keine

## Iteration 40 — cov-gateway-security-gdpr-export-erasure-dsar-routes — done — 2026-08-23 03:45
- commit: a777859d
- gebaut: (1) Router-Guard-Test `TestSecurityRoutes_GDPRGuards`
  (`route_security_test.go`) fuer alle acht Routen aus `route_security.go:54-65`
  (Export anfragen, genehmigen, ablehnen, herunterladen, auflisten;
  Erasure-Vorschau und -Ausfuehrung; DSAR-Suche): 401 ohne Token (echtes
  `RequireAuthenticated`), 403 authentifiziert ohne admin-Rolle (nur fuer die
  vier admin-only Routen: approve/deny/preview/execute/dsar-search), 503
  admin/auth + leere Registry.
  (2) ECHTER BUG gefunden und behoben (root cause, gleiche Klasse wie
  Iteration 39s ExportAuditLog-Fund): `mapSecurityError` (`security_grpc.go`)
  kannte `gdpr.ErrExportNotFound` und `gdpr.ErrTokenNotFound` nicht — beide
  Sentinel-Errors aus `postgres_repository.go` (unbekannter Download-Token,
  nicht existierende Export-ID bei Approve/Deny) fielen auf den
  default-Case und wurden als `codes.Internal` (blanker 500 "internal
  error") an den Client zurueckgegeben statt als `codes.NotFound` (404).
  Fix: zwei neue Cases in `mapSecurityError`. Regressionstest
  `TestGDPRExportRPCs_NotFoundSentinelsMapToNotFound` in
  `security_grpc_test.go` beweist den Fix fuer alle drei betroffenen RPCs
  (GetExportDownload/ApproveDataExport/DenyDataExport), plus zwei neue
  Zeilen in der bestehenden `TestMapSecurityError_AllSentinels`-Tabelle.
  Der Stub-GDPR-Repo in `security_grpc_test.go` gab bisher einen lokalen
  `errGDPRExportNotFound` (kein Sentinel) zurueck — auf die echten
  `gdpr.ErrExportNotFound`/`gdpr.ErrTokenNotFound` umgestellt, damit der
  Stub die Produktionsfehlerpfade tatsaechlich spiegelt und der Test den
  Bug ueberhaupt erreichen kann.
  (3) ZWEITER ECHTER BUG gefunden und behoben: `HandleListDataExports`
  (`route_security.go`) hatte KEINEN `RequireRole("admin")`-Guard auf
  `/gdpr/exports` (im Gegensatz zu approve/deny/preview/execute) UND keine
  Autorisierungspruefung im Handler selbst. Der Kommentar im Code behauptete
  "Only admins can view other users' exports -- gateway trusts RBAC
  middleware" — es gab aber gar keine RBAC-Middleware auf dieser Route. Jeder
  authentifizierte Nutzer konnte per `?user_id=<fremde-id>` die GDPR-
  Export-Anfragen eines anderen Nutzers einsehen (Status, Reviewer-Notizen,
  Zeitstempel) — genau die Klasse Fund, die diese Unit fuer den
  Download-Pfad erwartet hatte, hier aber im List-Pfad lag. Fix: Inline-Check
  `middleware.IsAdmin(ctx)`, wenn `user_id` gesetzt und ungleich der eigenen
  ID ist, sonst 403. Vier neue Handler-Tests beweisen self-view (own id),
  no-filter (default eigene Exporte), admin-override (fremde id erlaubt) und
  den 403-Fall (nicht-admin fremde id).
  (4) Der Download-Pfad selbst (`HandleGetExportDownload`) ist bereits sauber:
  kein `RequireRole`, aber tenant-gescopter 32-Byte-Zufallstoken
  (`GetExportByToken`) — "ein Nutzer kann den Export eines anderen nicht
  herunterladen" ist durch Token-Unerratbarkeit + Tenant-Scope belegt, nicht
  durch Rollen. Kein Fund, im Journal festgehalten statt "behoben" verkauft.
  (5) Alle Invalid-UUID-Faelle fuer `user_id` in Approve/Deny/RequestExport/
  ListExports/PreviewErasure/ExecuteErasure sind bereits vollstaendig am
  gRPC-Layer abgedeckt (`security_grpc_test.go` Zeilen 771-813) — nichts
  Neues noetig, im Journal statt im Code verifiziert.
- gate: build ok | vet ok | lint ok (0 issues, `internal/gateway` +
  `internal/server` + `internal/security`) | test ok (`go test -count=1
  ./internal/gateway/`, `./internal/server/`, `./internal/security/...`,
  0 SKIP, 4590 PASS-Subtests in den drei Kernpaketen) | migration n.a.
  (keine Tabellen-/Schema-Aenderung) | rls-smoke n.a. (keine Tabellen-/
  Policy-Aenderung) | `go test -count=1 -run TestOpenAPIRouteDrift
  ./internal/gateway/` PFLICHT gelaufen (836 Routen gegen 838 Spec-Pfade,
  gruen — keine Route hinzugefuegt/geaendert)
- coverage: internal/gateway 54,2 % (eigene Messung vor dieser Iteration via
  `git stash`) -> 54,3 % (danach). Kleiner Zugewinn wie beim Schwesterblock
  erwartet — der Wert liegt in den beiden gefundenen/behobenen Bugs
  (500-statt-404 bei unbekanntem Token/Export-ID, fehlender Admin-Guard bei
  Cross-User-Export-Liste), nicht in der Statement-Zahl.
- mutations-probe: (1) `mapSecurityError` testweise auf den Stand vor dieser
  Iteration zurueckgesetzt (die beiden neuen Cases entfernt) —
  `TestMapSecurityError_AllSentinels/gdpr_export_not_found`,
  `/gdpr_token_not_found` und alle drei Subtests von
  `TestGDPRExportRPCs_NotFoundSentinelsMapToNotFound` wurden rot (erwartet
  NotFound/0x5, bekamen Internal/0xd). Zurueckgesetzt via `cp` einer
  Sicherungskopie, `diff` gegen Original danach leer.
  (2) `HandleListDataExports` testweise auf den alten Zustand zurueckgesetzt
  (Admin-Check entfernt, `user_id`-Override ungeprueft uebernommen) —
  `TestHandleListDataExports_CrossUserRequiresAdmin` wurde rot (503 statt
  erwarteter 403, weil die registrierte Dummy-Verbindung ohne fruehen
  403-Return direkt bis zum echten gRPC-Dial durchlief und dort scheiterte).
  Zurueckgesetzt, `diff` gegen Original danach leer.
- verify vorgaenger: sauber — `556e8d64` (letzter Commit vor dieser
  Iteration) aendert laut `git show --stat` nur
  `internal/gateway/route_security_test.go` (neue Testdatei-Ergaenzung),
  `internal/security/audit/service.go` (neuer Sentinel-Error),
  `internal/server/security_grpc.go` (Fehler-Mapping-Fix) und
  `internal/server/security_grpc_test.go` (Regressionstests) — Diff geprueft
  (`git show`), sauberer root-cause-Fix nach demselben Muster wie hier,
  keine der acht Fehlerklassen anwendbar.
- neue-units: keine (beide gefundenen Bugs wurden in dieser Iteration direkt
  behoben — kein Deploy-Hazard, keine neue Route, kein neuer
  `RequirePermission`-Guard, keine Migration noetig; der bestehende
  `RequireRole("admin")`-Guard auf approve/deny/preview/execute/dsar-search
  blieb unveraendert, nur der Inline-Check auf `/gdpr/exports` kam hinzu)
- offen: keine

## Iteration 41 — cov-gateway-security-retention-ip-password-routes — done — 2026-08-23 04:05
- commit: 8e25a9f8
- gebaut:
  (1) ECHTER BUG gefunden und behoben, gleiche Fehlerklasse wie Iteration 39/40:
  `CreateIPRule` (`internal/server/security_grpc.go`) validierte `ip_cidr` nur
  auf "nicht leer" — jede syntaktisch kaputte oder nicht netzwerk-ausgerichtete
  CIDR-Notation (z. B. `"not-a-cidr"` oder `"192.168.1.5/24"` mit gesetzten
  Host-Bits) fiel erst beim `INSERT` gegen die Postgres-`CIDR`-Spalte auf und
  wurde dort als generischer `codes.Internal` ("failed to create IP rule")
  zurückgegeben — 500 statt 400. Fix: `net.ParseCIDR` plus Host-Bit-Check
  (`ip.Equal(network.IP)`) vor dem Insert, spiegelt exakt Postgres' eigene
  CIDR-Semantik (die `inet`-Werte mit Host-Bits akzeptiert, `cidr`-Werte aber
  ablehnt). Zwei neue Faelle in der bestehenden
  `TestSecurityGRPC_ValidationErrors`-Tabelle (`security_grpc_test.go`):
  `CreateIPRule/malformed_cidr`, `CreateIPRule/cidr_host_bits_set`, beide
  `codes.InvalidArgument`.
  (2) Router-Level-Guard-Tabelle `TestSecurityRoutes_RetentionIPRulePasswordGuards`
  (`route_security_test.go`) fuer die zehn Routen aus `route_security.go:68-82`
  plus `HandleValidatePassword`: 401 ohne Auth, 403 fuer Nicht-Admin auf den
  acht admin-only Routen (ip-rules ×3, retention-policies ×4,
  retention-runs/latest), 503 bei leerer Registry fuer alle elf. Fuer
  `GET /password/policy` und `POST /password/validate` (bewusst nicht
  admin-only) nur 401/503 geprueft, kein 403-Fall.
  (3) Retention-Tage 0/negativ: bereits vor dieser Iteration durch
  `validate:"required,min=1"` auf `createRetentionPolicyRequest.RetentionDays`
  UND `updateRetentionPolicyRequest.RetentionDays` abgedeckt (HTTP-Layer) plus
  `req.RetentionDays <= 0` am gRPC-Layer (Verteidigung in der Tiefe, beide
  Ebenen bereits vorhanden) — kein Fund, aber bislang unbewiesen. Drei neue
  Tests: `TestHandleCreateRetentionPolicy_ZeroRetentionDaysRejected`,
  `_NegativeRetentionDaysRejected`, `TestHandleUpdateRetentionPolicy_ZeroRetentionDaysRejected`
  (alle 400 mit `retention_days` im Validation-Detail).
  (4) Unbekannter `resource_type`: Ist-Zustand wie im Scope beschrieben belegt
  — `gdpr.RetentionEngine.runPolicy` meldet einen nicht registrierten Typ
  bereits als `RetentionItemUnmapped` ("nicht zugeordnet"), das ist seit
  Lauf 10 vollstaendig getestet (`retention_test.go`,
  `TestRetentionEngine_EmptyRegistryReportsUnmapped`,
  `TestRetentionEngine_Run_Integration`). Neu: DB-Test
  `TestSecurityGRPCServer_CreateRetentionPolicy_AcceptsUnknownResourceType`
  (`security_grpc_retention_policies_db_test.go`) belegt, dass die Erstellung
  selbst einen Tippfehler-Typ NICHT ablehnt — genau der Zustand, den der
  Scope als "festhalten, ohne Whitelist einzufuehren" verlangt hat. Keine
  Whitelist eingefuehrt.
  (5) `HandleValidatePassword` (Auth, Rate-Limit) belegt: Route sitzt unter
  `authMiddleware` (401 ohne Token, per Router-Guard-Test bewiesen) UND unter
  dem globalen Per-IP-Rate-Limiter aus `cmd/gateway/main.go:161-162`
  (`rateLimiter.Middleware`, vor jeder Route-Registrierung auf den ganzen
  Router angewandt) — kein routen-lokaler Rate-Limit-Fund, kein Fix noetig.
  `userID` kommt immer aus dem eigenen Token (`middleware.GetUserID`), die
  RPC kann also nicht als Orakel fuer fremde Passwort-Historien missbraucht
  werden. Kein Fund.
  (6) Admin-Selbstaussperrung durch IP-Regeln: VERIFIZIERT als real
  existierendes Risiko, NICHT behoben (wie im Scope vorgegeben — "falls ja,
  gehoert das als Fund notiert, nicht behoben"). `gateway.IPFilterMiddleware`
  (`internal/gateway/ip_filter.go`) wendet Block-/Allow-Regeln aus
  `ip_access_rules` auf JEDEN Request an, sobald sie geladen sind — es gibt
  keinen Schutz dagegen, dass ein Admin sich selbst per `block 0.0.0.0/0`
  oder einer Allow-Regel ohne die eigene Range aussperrt. Nur der Kaltstart
  ist "fail open" (Zeile 132-141), der Normalbetrieb nicht. Dokumentiert als
  Kommentar direkt an `CreateIPRule` (Code-Referenz statt nur Journal-Notiz,
  damit der naechste Bearbeiter der Funktion es sieht) und hier im Journal.
  Kein Fix — Produktentscheidung (Warnung? harte Sperre? Break-Glass?).
- gate: build ok | vet ok | lint ok (0 issues, `internal/gateway` +
  `internal/server`) | test ok (`go test -count=1 -p 1 ./internal/gateway/...
  ./internal/server/...`, 0 SKIP) | migration n.a. (keine Tabellen-/
  Schema-Aenderung) | rls-smoke n.a. (keine Tabellen-/Policy-Aenderung) |
  `go test -count=1 -run TestOpenAPIRouteDrift ./internal/gateway/` PFLICHT
  gelaufen (836 Routen gegen 838 Spec-Pfade, gruen — keine Route
  hinzugefuegt/geaendert)
- coverage: internal/gateway 54,3 % (eigene Messung vor dieser Iteration via
  `git stash`) -> 54,5 % (danach). internal/server 70,7 % -> 70,7 %
  (unveraendert — der Fix ist eine schmale Validierungszeile, der Wert liegt
  im gefundenen Bug, nicht in der Statement-Zahl).
- mutations-probe: `net.ParseCIDR`-Check in `CreateIPRule` testweise durch
  `if false { _, _, _ = net.ParseCIDR(...) }` ersetzt (Import bleibt benutzt,
  damit der Build durchlaeuft) — `TestSecurityGRPC_ValidationErrors/CreateIPRule/malformed_cidr`
  wurde rot: die ungueltige CIDR erreichte ohne Check `s.pool.Exec` gegen den
  nil-Pool des Test-Setups und panickte dort (`pgxpool.(*Pool).Acquire` auf
  `0x0`) — genau der Beweis, dass der Check das ist, was die Funktion vor dem
  Erreichen des generischen 500-Pfads bewahrt. Zurueckgesetzt via `cp` einer
  Sicherungskopie, `diff` gegen Original danach leer.
- verify vorgaenger: sauber — `a777859d` (letzter Commit vor dieser
  Iteration) aendert laut `git show --stat` nur `route_security.go` (ein
  neuer Inline-Admin-Check auf `/gdpr/exports`, additiv), `route_security_test.go`,
  `security_grpc.go` (zwei neue Fehler-Mapping-Cases, additiv),
  `security_grpc_test.go` — keine der acht Fehlerklassen anwendbar, bereits
  im Detail gegengeprueft (siehe eigener Journal-Eintrag der Vorgaenger-
  Iteration).
- neue-units: keine (der Selbstaussperrungs-Fund ist laut Scope-Vorgabe
  dieser Unit ausdruecklich "notiert, nicht behoben" — keine eigene Unit
  gefordert; als Code-Kommentar an `CreateIPRule` UND hier dokumentiert, damit
  er bei einer spaeteren IP-Regel-Ueberarbeitung nicht verloren geht)
- offen: keine

## Iteration 42 — cov-gateway-security-vendor-access-routes — done — 2026-08-23 04:06
- commit: 9fc4ef78
- gebaut: Router-Level 401/403/503-Guard-Tests fuer die fuenf
  Vendor-Access-Routen (`route_security.go:85-93`) plus eine
  Validierungsluecke geschlossen (`proposed_start` required bei
  Counter-Propose). Zwei Befunde dokumentiert statt behoben (siehe unten):
  fehlende Audit-Eintraege (neue Unit `feat-vendor-access-audit-trail`) und
  keine Enforcement-Wirkung des Revoke (Code-Kommentar an `Service.Revoke`).
- gate: build ok | vet ok | lint ok (0 issues, `internal/gateway` +
  `internal/security/vendoraccess`) | test ok (`go test -count=1 -p 1
  ./internal/gateway/... ./internal/server/... ./internal/security/vendoraccess/...`,
  0 SKIP) | migration n.a. (keine Tabellen-/Schema-Aenderung) | rls-smoke n.a.
  (keine Tabellen-/Policy-Aenderung) | `go test -count=1 -run
  TestOpenAPIRouteDrift ./internal/gateway/` PFLICHT gelaufen (836 Routen
  gegen 838 Spec-Pfade, gruen — keine Route hinzugefuegt/geaendert)
- coverage: internal/gateway 54,5 % (eigene Messung vor dieser Iteration via
  `git stash`) -> 54,6 % (danach).
- mutations-probe: `guard` in der Route-Registrierung testweise von
  `RequirePermission("security:vendor_access", "manage")` auf
  `RequirePermission("security:vendor_access", "read")` geaendert —
  `TestSecurityRoutes_VendorAccessGuards` wurde fuer alle fuenf
  "authorized empty registry"-Faelle rot (403 statt 503, "insufficient
  permissions"), genau der Beweis, dass der Test den richtigen Permission-Key
  pruefte und nicht nur "irgendein Guard greift". Zurueckgesetzt via `cp`
  einer Sicherungskopie, `diff` gegen Original danach leer.
- verify vorgaenger: sauber — `8e25a9f8` (letzter Commit vor dieser
  Iteration) aendert laut `git show --stat` `route_security_test.go` (neue
  Guard-Tests, additiv), `internal/server/security_grpc.go` (CIDR-Validierung
  in `CreateIPRule`, ROOT CAUSE im Service, kein gRPC-Layer-Umgehung, kein
  neuer Guard, keine neue Tabelle, keine Wire-Shape-Aenderung, keine neue
  Route), `security_grpc_retention_policies_db_test.go` (neuer DB-Test,
  ungetaggt) und `security_grpc_test.go` — keine der acht Fehlerklassen
  anwendbar.
- neue-units: `feat-vendor-access-audit-trail` (ans Backlog-Ende gehaengt) —
  keine der fuenf Vendor-Access-Aktionen
  (`internal/security/vendoraccess/service.go`) schreibt einen
  `audit_log`-Eintrag, nur `slog.Info`. Im Auslieferungsmodell "ein Server
  pro Kunde" ist dieser Mechanismus der einzige legitime Fernzugriffskanal
  von Zentria auf Kundendaten — ohne Audit-Kette ist "wer hat wann welchen
  Zugriff genehmigt/widerrufen" nicht nachweisbar. `cmd/auth/main.go:147+166`
  konstruiert `vendoraccess.Service` und `auditService` bereits im selben
  Scope, die Verdrahtung ist ein Konstruktor-Parameter. Nicht selbst
  behoben, weil diese Coverage-Unit laut Scope kein Verhalten aendern soll
  (harte Regel 2 / Befund 2 des Lauf-Kopfs) und weil Wire-Format
  (action-Strings) eine eigene Entscheidung verdient.
- offen: Zwei Befunde aus dem Scope sind bewusst dokumentiert, nicht
  behoben: (1) Doppelte Genehmigung ist bereits Ende-zu-Ende bewiesen —
  `vendoraccess.ErrInvalidStatus` -> `codes.FailedPrecondition` (
  `TestVendorAccessRPCs_HappyPathAndDomainErrors`,
  `internal/server/security_grpc_test.go`) -> HTTP 409 (generisch bewiesen
  in `helpers_test.go`); eine dritte, gateway-eigene Kopie derselben
  Assertion haette keinen neuen Pfad geprueft. (2) Revoke hat KEINE
  Enforcement-Wirkung — ein Grep nach `VendorAccessStatusActive` und
  `vendor_access_requests` ueber `internal/` und `cmd/` findet ausschliesslich
  die fuenf Dateien, die diese Unit ohnehin anfasst; kein Middleware- oder
  Sitzungs-Check liest die Tabelle. Der Datensatz ist ein
  Einwilligungs-/Audit-Beleg fuer die Auftragsverarbeitung, kein
  Zugriffs-Gate. Als Kommentar an `Service.Revoke` festgehalten, damit der
  naechste Bearbeiter es nicht erneut recherchieren muss. Produktentscheidung
  (soll ein Revoke einen echten Kanal schliessen?) liegt bei Luke.

## Iteration 43 — cov-gateway-billing-payments-and-creditnotes-routes — done — 2026-08-23 04:14
- commit: 13dbdf8f
- gebaut: Erste Haelfte von `route_biz_billing.go` (Gutschrift- + Zahlungs-
  Handler) gegen die vier `done_when`-Punkte geprueft. Ein echter Fund
  behoben, drei bereits erfuellt und mit Test/Kommentar belegt statt
  doppelt gebaut:
  (1) Neuer Validator `max_2dp` (`internal/validation/validation.go`) —
  `payments.amount` ist `NUMERIC(15,2)` (migrations/000045), aber
  `decimal_gt0` liess beliebige Nachkommastellen durch; ein Wert wie
  "10.999" waere auf dem INSERT-Weg still auf "11.00" gerundet worden statt
  an der Vertrauensgrenze abgewiesen zu werden. `max_2dp` ist bewusst ein
  eigener Tag, nicht in `decimal_gt0` eingebaut: `decimal_gt0` wird auch von
  `route_biz_ext.go` (hourly_rate) und `route_einkauf.go` (quantity,
  NUMERIC(15,4)/(12,4)) verwendet, die eine andere Skala brauchen — eine
  gemeinsame Aenderung haette dort etwas kaputt gemacht, das nicht in dieser
  Unit's `sources` steht.
  (2) Idempotency-Key fehlt: bereits vollstaendig bewiesen in
  `internal/middleware/idempotency_test.go`
  (TestIdempotency_MissingKey_WarnMode_Passes,
  TestIdempotency_MissingKey_HardMode_Blocks,
  TestHardMode_MissingKey_Returns400) — der Gateway-Test ruft Handler direkt
  auf und liegt damit hinter der Middleware, kann diesen Pfad also gar nicht
  pruefen. In Produktion gilt HardMode (Compose-Env-Var
  `IDEMPOTENCY_MODE=hard`) — ein POST ohne Header bekommt dort 400, kein
  stiller Pass-Through. Als Kommentar in `route_biz_billing_test.go`
  verankert.
  (3) `HandleDeletePayment` auf einer GoBD-gesperrten Rechnung: die
  Payment-Zeile wird IMMER geloescht, nur der gekoppelte Status-Rueckfall
  wird bei `LockedAt != nil` uebersprungen (`payment/service.go:270-278`,
  eigener GoBD-§146-Kommentar) — bereits bewiesen in
  `TestDelete_NoRevertWhenInvoiceLocked` (`internal/biz/payment/service_test.go`).
  Ob das Loeschen der Zahlungszeile selbst bei einer gesperrten Rechnung
  verboten sein sollte (nicht nur der Status-Rueckfall), ist eine
  Compliance-Frage fuer Luke — als Kommentar dokumentiert, keine eigene Unit
  angelegt (kein belegter Schaden, nur eine offene Frage).
  (4) `HandleGenerateCreditNotePDF` ohne Firmeneinstellungen: bereits
  sauber geloest — `requireCompanySettings` (`biz_grpc.go:142`) liefert
  `codes.FailedPrecondition("company settings not configured")`, generisch
  auf HTTP 409 gemappt (`helpers_test.go`). Kein leeres PDF, kein 500er.
  Neue Tests: `TestHandleRecordPayment_InvalidAmountFormats` (6 Faelle:
  negativ, null, nicht-numerisch, leer, >2 Nachkommastellen, wissenschaftliche
  Notation mit zu hoher Praezision — alle 400 ueber `amount`),
  `TestHandleRecordPayment_ScientificNotationWithinPrecision` ("1e2" = 100,
  0 Nachkommastellen, muss durchgehen), `TestHandleRecordPayment_AmountAsJSONNumber`
  (JSON-Zahl statt String scheitert am Decode, nicht an der Validierung —
  eigener 400-Pfad). Bestehender Test
  `TestHandleRecordPayment_AmountStaysStringHighPrecision` musste angepasst
  werden (Fixture hatte 9 Nachkommastellen, waere jetzt am neuen `max_2dp`
  gescheitert) — Wert auf ".12" gekuerzt, Praezisions-Aussage (grosser
  Integer-Teil, kein float64-Parse) bleibt unveraendert bewiesen.
- gate: build ok | vet ok | lint ok (0 issues, `internal/gateway` +
  `internal/validation`) | test ok (`go test -count=1
  ./internal/gateway/ ./internal/validation/...`, 0 SKIP) | migration n.a.
  (keine Tabellen-/Schema-Aenderung) | rls-smoke n.a. (keine Tabellen-/
  Policy-Aenderung) | `go test -count=1 -run TestOpenAPIRouteDrift
  ./internal/gateway/` PFLICHT gelaufen (836 Routen gegen 838 Spec-Pfade,
  gruen — keine Route hinzugefuegt/geaendert)
- coverage: internal/gateway 54,6 % (eigene Messung vor dieser Iteration via
  `git stash`) -> 54,6 % (danach, kein Sprung bei einer Nachkommastelle —
  das Paket hat ueber 10.000 ungedeckte Zeilen, sechs neue Testfaelle in
  zwei bereits teilweise abgedeckten Handlern bewegen die Prozentzahl nicht
  sichtbar; die neuen Zweige selbst sind aber real abgedeckt, siehe
  Mutations-Probe).
- mutations-probe: `max_2dp` in `internal/validation/validation.go` auf
  `return true` (immer erfolgreich) gesetzt —
  `TestHandleRecordPayment_InvalidAmountFormats/MoreThanTwoDecimalPlaces`
  und `.../ScientificNotationTooPrecise` wurden rot (503 statt 400, weil die
  Anfrage nun bis zur nicht erreichbaren RPC durchlief), die anderen vier
  Faelle blieben gruen (sie werden von `decimal_gt0` bzw. `required`
  abgefangen, nicht von `max_2dp`) — genau der erwartete, isolierte
  Bruch. Zurueckgesetzt via `cp` einer Sicherungskopie, `diff` gegen
  Original danach leer (nur die urspruengliche 13-Zeilen-Ergaenzung bleibt).
- verify vorgaenger: sauber — `9fc4ef78` (letzter Commit vor dieser
  Iteration) aendert laut `git show --stat` nur `route_security_test.go`
  (neue Guard-Tests + ein neuer Kardinalitaets-Test, additiv),
  `security/vendoraccess/service.go` (reiner Kommentar an `Revoke`, keine
  Verhaltensaenderung) — keine der acht Fehlerklassen anwendbar. Kleine
  Ungenauigkeit notiert, kein Fund: die Commit-Message von `9fc4ef78`
  behauptet, eine "Validierungsluecke" bei fehlendem `proposed_start`
  geschlossen zu haben, aber `route_security.go` selbst ist in diesem Diff
  gar nicht veraendert — `validate:"required"` auf `ProposedStart` existiert
  bereits seit einem deutlich aelteren Commit (`d3b7cb01`). Der neue Test
  beweist bestehendes, korrektes Verhalten, behebt aber nichts; kein
  Schaden, da nichts Falsches ausgeliefert wurde.
- neue-units: keine (der einzige offene Punkt — Loeschen einer Payment-Zeile
  auf gesperrter Rechnung — ist eine reine Produktentscheidung ohne
  belegten Schaden, siehe oben; per harter Regel 11 keine Unit ohne echten
  Fund)
- offen: keine

## Iteration 44 — cov-gateway-billing-dunning-and-gobd-routes — done — 2026-08-23 04:25
- commit: 1e89739b
- gebaut: Zweite Hälfte von `route_biz_billing.go` gegen die vier `done_when`-
  Punkte geprüft. Ein echter Fund behoben, drei bereits erfüllt und mit Test
  bzw. Kommentar belegt statt doppelt gebaut:
  (1) `HandleGenerateGoBDExport` (859) prüfte `from_date`/`to_date` nur auf
  Format, nie auf Reihenfolge — anders als die beiden Schwester-Handler im
  selben File: `ExportDATEV` bekommt seinen Schutz über `datev.ErrInvalidPeriod`
  im Builder, `GetPaymentStats` (biz_grpc.go:2383) hat exakt denselben
  Inline-Vergleich. Eine vertauschte Spanne (`from` nach `to`) traf in
  `ListForDATEVExport`/`ListForDATEVExport` (invoice.Service, credit-note-
  Pendant) nie auf ein `invoice_date`, das beide Bedingungen erfüllt — die
  Schleifen liefen leer durch, und `GenerateGoBDExport` (dunning-Service)
  baute daraus eine valide, leere CSV mit HTTP 200. Ununterscheidbar von
  "keine Umsätze in diesem Zeitraum", genau der im Scope beschriebene
  Schaden. Fix: `if toDate.Before(fromDate) { return InvalidArgument }`
  direkt nach dem Parsing, vor der Paging-Schleife — dieselbe Formulierung
  wie in `GetPaymentStats` ("to_date must be after from_date").
  (2) Eskalation über die höchste Stufe hinaus: bereits bewiesen
  (`TestEscalateDunning/max_level_reached`, `biz_grpc_dunning_dashboard_
  exports_test.go:359`). Eskalation auf einer bereits bezahlten Rechnung
  hatte KEINEN eigenen Test — mechanisch derselbe Pfad (die Rechnung taucht
  nicht in `DetectAndCreateDunnings`' Ergebnis auf), aber die Garantie dafür
  liegt in einer anderen Datei: `PostgresRepository.GetOverdue`
  (`invoice/postgres_repository.go:402`) filtert `status = 'sent'` fest,
  eine bezahlte Rechnung kann dort nie erscheinen. Neuer Test
  `TestEscalateDunning/paid_invoice` verankert genau diese Garantie explizit,
  statt sie nur implizit über den Max-Level-Test mitlaufen zu lassen.
  (3) `HandleLockInvoice` auf bereits gesperrter Rechnung: bereits
  vollständig bewiesen — `TestLockInvoice/already_locked_maps_to_
  FailedPrecondition` (`biz_grpc_invoices_creditnotes_payments_test.go`)
  deckt `invoice.ErrInvoiceLocked` -> `codes.FailedPrecondition` -> HTTP 409
  über `mapBizError`/`grpcStatusToHTTP` ab.
  (4) Fremdsystem-Fehlermeldungen: nicht einschlägig für dieses Handler-Set.
  `HandleExportDATEV` und `HandleGenerateGoBDExport` bauen ihre CSVs beide
  lokal (Buchungsstapel-Builder bzw. `dunning.BuildGoBDCSV`) — keiner der
  beiden ruft eine externe API auf. Der Fund aus Lauf 10, auf den die Notes
  verweisen, betrifft ausschließlich `route_datev_upload.go` (eigene Unit
  `cov-gateway-datev-upload-routes`, noch offen im Backlog).
- gate: build ok | vet ok | lint ok (0 issues, `internal/server` +
  `internal/gateway`) | test ok (`go test -count=1 ./internal/server/...
  ./internal/gateway/`, 0 SKIP) | migration n.a. (keine Tabellen-/Schema-
  Änderung) | rls-smoke n.a. (keine Tabellen-/Policy-Änderung) |
  `go test -count=1 ./internal/gateway/` PFLICHT gelaufen, grün — keine
  Route hinzugefügt/geändert, `TestOpenAPIRouteDrift` unberührt
- coverage: internal/server 70,7 % (eigene Messung vor dieser Iteration via
  `git stash`) -> 70,7 % (danach, unverändert bei einer Nachkommastelle —
  das Paket hat mehrere Tausend Zeilen, sieben neue Zeilen Fix plus zwei
  neue Testfälle bewegen die Prozentzahl nicht sichtbar; die neue
  Verzweigung selbst ist real abgedeckt, siehe Mutations-Probe). Bezugswert
  der Unit (`internal/gateway 54,1 %`) trifft nicht zu: der Fund und beide
  neuen Tests liegen in `internal/server` (biz_grpc.go + zugehöriger Test),
  weil dort die Datumsvergleichs-Logik sitzt — `internal/gateway` selbst
  reicht `from_date`/`to_date` nur unverändert an die RPC durch und wurde in
  dieser Iteration nicht verändert; `go test ./internal/gateway/` lief
  trotzdem pflichtgemäß grün.
- mutations-probe: zwei Läufe, beide gegen eine `cp`-Sicherungskopie (nicht
  `git checkout`), zurückgeschrieben, `diff` gegen die Kopie danach jeweils
  identisch (0 Zeilen Unterschied).
  (a) `if toDate.Before(fromDate)` in `GenerateGoBDExport` zu
  `if false && toDate.Before(fromDate)` entschärft ->
  `TestGenerateGoBDExport_Validation/end_before_start_maps_to_invalid_
  argument` rot — nicht nur falscher Code, sondern ein Panic (Nil-Pointer
  in `invoiceService.ListForDATEVExport`, weil der Test-Server absichtlich
  ohne echten Service konstruiert ist), was den erwarteten Schaden noch
  deutlicher zeigt: ohne die Prüfung läuft die Anfrage bis zur ersten
  echten Abhängigkeit durch, statt an der Eingabegrenze abgewiesen zu
  werden. Die beiden `invalid_*_date`-Subtests blieben grün (eigener Pfad).
  (b) `codes.FailedPrecondition` im Fallback-Return von `EscalateDunning`
  (biz_grpc.go:1061) zu `codes.Internal` verstümmelt -> sowohl
  `TestEscalateDunning/max_level_reached` als auch das neue
  `TestEscalateDunning/paid_invoice` wurden rot (erwarteter Code 9, erhalten
  13) — beweist, dass beide Tests denselben Codepfad wirklich erreichen und
  ihn absichern, nicht nur zufällig grün sind. `happy_path` blieb grün.
- verify vorgaenger: sauber — `13dbdf8f` (Iteration 43) ändert
  `route_biz_billing.go` (neuer `max_2dp`-Validator-Tag auf einem
  bestehenden Feld, keine neue Route), `validation.go` (neue Validator-
  Funktion, additiv registriert) und den zugehörigen Test. Keine der acht
  Fehlerklassen einschlägig: kein neuer Gateway-Handler, kein Stub/TODO,
  kein `.proto`, keine Migration, kein neuer `RequirePermission`-Guard,
  keine neue Tabelle, keine neue Route, kein Wire-Shape-Wechsel, kein
  ersetzter Guard-Key.
- neue-units: keine — beide Funde (GoBD-Datumsvergleich, fehlender Test für
  bezahlte Rechnung) sind in dieser Iteration selbst behoben, kein
  aufgeschobener Rest.
- offen: keine

## Iteration 45 — cov-gateway-datev-upload-routes — done — 2026-08-23 04:33
- commit: 1b5eaccb
- gebaut:
  Zwei falsche Prämissen aus dem Backlog-Eintrag widerlegt (Regel 11), ein
  echter Fund behoben, acht zuvor ungetestete Gateway-Handler abgedeckt.
  (1) WIDERLEGT: "Uploadgrössen: eine zu grosse Datei muss 413 liefern."
  `route_datev_upload.go` nimmt an keiner Stelle eine client-gesendete Datei
  entgegen — `HandleUploadBuchungsstapel` rendert das CSV serverseitig aus
  einem Datumsbereich (`UploadService.UploadBuchungsstapel`), `HandleUploadBeleg`
  rendert die Rechnungs-PDF serverseitig aus einer `invoice_id`
  (`UploadService.UploadInvoiceBeleg` -> `belegRenderer.RenderInvoice`). Es
  gibt keinen multipart/Datei-Body auf dieser Route, also keine 413-Klasse.
  (2) WIDERLEGT (teilweise): "Fortschritt und Statusabfrage, falls
  vorhanden: ein unbekannter Auftrag muss 404 liefern." Es existiert keine
  Statusabfrage für einen einzelnen Upload-Auftrag — nur `HandleListUploadLogs`
  (Liste, kein Einzel-GET per ID). Gegenstandslos.
  (3) ECHTER FUND, BEHOBEN: `mapDatevUploadError`
  (`internal/server/datev_upload_grpc.go`) kannte bis zu dieser Iteration nur
  "fehlender Connect" (ErrNotConnected/ErrNoAPIConfig -> FailedPrecondition/409)
  als admin-behebbaren Fall. Ein abgelaufener oder widerrufener DATEV-Refresh-
  Token lief dagegen unbenannt durch den `default`-Zweig und kam beim Admin als
  identisches "DATEV upload failed" (Internal/500) an wie ein echter interner
  Bug — kein Hinweis, dass ein simples Reconnect reicht. Root Cause lag in
  `OAuthManager.RefreshAccessToken` (`internal/biz/datev/oauth.go`): ein
  Non-200 vom DATEV-Token-Endpoint wurde immer als derselbe generische Fehler
  zurückgegeben, egal ob 4xx (Token abgelehnt, invariant unter Retry) oder 5xx
  (transiente DATEV-Störung). Fix: neuer Sentinel `ErrReauthRequired`, gesetzt
  nur bei 4xx (die bereits bestehende Testfixture nutzt exakt den
  DATEV-typischen Fall `401 {"error":"invalid_grant"}`); 5xx bleibt bewusst
  unverändert generisch, weil dort erneutes Versuchen sinnvoll ist und "bitte
  neu verbinden" die falsche Anweisung wäre. `mapDatevUploadError` bekommt
  einen neuen Case, der `ErrReauthRequired` auf FailedPrecondition mit fester,
  aktionabler Nachricht ("datev: connection expired, please reconnect")
  mapped — am Gateway wird daraus über `grpcStatusToHTTP` ein 409, keine
  Fremdsystem-Details im Text (nur "status %d", niemals der Response-Body,
  landet weiterhin ausschliesslich im `slog.Error`).
  Root-Cause-Statt-Symptom geprüft: `GetAccessToken` wird auch von
  `BelegbilderUploader.UploadBeleg` genutzt (Belegbild-Pfad) — der Fix in
  `OAuthManager` deckt beide Upload-Pfade (Buchungsstapel UND Belegbild) mit
  einer einzigen Änderung ab, keine zweite Stelle nötig.
  Die vom Backlog befürchtete Gefahr ("401 vom Fremdsystem erscheint als
  eigenes 401 und wirft den Nutzer aus der Sitzung") ist NICHT eingetreten
  und war es vorher schon nicht: `mapDatevUploadError` gibt an keiner Stelle
  `codes.Unauthenticated` zurück (das einzige, was `grpcStatusToHTTP` auf HTTP
  401 abbildet), auch nicht in dieser Iteration. Damit bleibt es fachlich
  korrekt bei "Verbindung abgelaufen" (409), niemals "deine Session ist
  abgelaufen" (401).
  Coverage-Teil: acht der neun Handler in `route_datev_upload.go` hatten vor
  dieser Iteration keinen einzigen Test (nur der OAuth-Callback-Flow war
  abgedeckt, siehe Lauf 10). Neue Tests decken für alle acht die Reihenfolge
  Client-Check -> Tenant-Check ab (503 bzw. 401, Tabellen-Tests analog zum
  Muster aus `route_biz_billing_test.go`), dazu gezielt: fehlende
  start_date/end_date bei `HandleUploadBuchungsstapel` (400, Validierung),
  eine ungültige `invoice_id` bei `HandleUploadBeleg` (400, kein Downstream-
  Fehler), fehlerhaftes JSON bei `HandleUpdateUploadConfig` (400), sowie je
  ein "erreicht die RPC"-Beleg (503 gegen die Dummy-Verbindung) für jeden
  Handler mit gültigem Input.
  Idempotenz/Doppel-Upload NICHT selbst gebaut (Scope-Grenze dieser Unit):
  `UploadBuchungsstapel` und `UploadBeleg` haben keinerlei Idempotenz-Schutz
  (kein Idempotency-Key, keine Duplikatsprüfung) — ein zweiter Klick oder ein
  Retry nach Timeout erzeugt eine zweite echte DATEV-Buchung/-Belegbild. Wie
  im Scope vorgegeben an die bereits im Backlog stehenden Scan-Units
  `scan-inbound-paths-without-duplicate-delivery-guard` und
  `scan-finance-mutations-without-idempotency-key` gemeldet (beide noch
  `status: todo`) statt selbst als Feature gebaut.
- gate: build ok (`-p 2`, gateway+server+biz/datev+cmd/gateway+cmd/biz) |
  vet ok | lint ok (0 issues, golangci-lint auf allen drei Paketen) |
  test ok (`go test -count=1 ./internal/gateway/ ./internal/server/...
  ./internal/biz/datev/...`, 0 SKIP) | migration n.a. (keine Tabelle/Policy
  angefasst) | rls-smoke n.a. | `go test -count=1 -run TestOpenAPIRouteDrift
  ./internal/gateway/` PFLICHT gelaufen, grün (836 registrierte gegen 838
  dokumentierte Pfade, keine Route geändert)
- coverage: internal/gateway 54,6 % (eigene Messung vor dieser Iteration via
  `git stash`) -> 55,1 % (danach) | internal/biz/datev 79,5 % -> 79,6 %
  (Bezugswert der Unit "internal/gateway 54,1 %" leicht abweichend von meiner
  Vorher-Messung 54,6 % — eine der Iterationen 42-44 hat dasselbe Paket
  bereits bewegt, siehe HARTE REGELN Block-Kopf; die eigene Messung gilt)
- mutations-probe: zwei Läufe, beide gegen `cp`-Sicherungskopien
  (`/tmp/oauth.go.bak`, `/tmp/datev_upload_grpc.go.bak`), zurückgeschrieben,
  `diff` gegen die Kopie danach je 0 Zeilen Unterschied.
  (a) `if resp.StatusCode >= 400 && resp.StatusCode < 500` in
  `RefreshAccessToken` zu `if false && ...` entschärft ->
  `TestRefreshAccessToken_NonOKStatusReturnsError` rot (erwartete
  `errors.Is(err, ErrReauthRequired)`, bekam den alten generischen Fehler);
  `TestRefreshAccessToken_ServerErrorIsNotReauthRequired` blieb grün (eigener
  Pfad, 5xx nie betroffen).
  (b) `case errors.Is(err, datev.ErrReauthRequired):` in `mapDatevUploadError`
  zu `case false && errors.Is(...):` entschärft -> drei Tests rot:
  `TestMapDatevUploadError/reauth_required`,
  `TestMapDatevUploadError/wrapped_reauth_required` (beide erwarteten
  FailedPrecondition, bekamen Internal aus dem default-Zweig) und
  `TestMapDatevUploadErrorReauthMessageIsActionable` (erwartete eine von
  "DATEV upload failed" verschiedene Nachricht, bekam exakt diese). Die
  übrigen zehn Subtests von `TestMapDatevUploadError` blieben grün.
- verify vorgaenger: sauber — `1e89739b` (Iteration 44) fügt in
  `GenerateGoBDExport` (`internal/server/biz_grpc.go`) eine
  Datumsvergleichsprüfung hinzu und einen Dunning-Test für eine bezahlte
  Rechnung. Keine der acht Fehlerklassen einschlägig: kein neuer
  Gateway-Handler/Service-Bypass, kein Stub/TODO, kein `.proto`, keine
  Migration, kein neuer `RequirePermission`-Guard, keine neue Tabelle, keine
  neue Route, kein Wire-Shape-Wechsel, kein ersetzter Guard-Key. Diff selbst
  gegengelesen (`git show 1e89739b -- backend/internal/server/biz_grpc.go`),
  nicht nur den Journal-Text übernommen.
- neue-units: keine — der einzige nicht selbst behebbare Fund (fehlender
  Idempotenz-Schutz auf beiden Upload-RPCs) ist bereits durch zwei bestehende
  Scan-Units (`scan-inbound-paths-without-duplicate-delivery-guard`,
  `scan-finance-mutations-without-idempotency-key`) abgedeckt; eine neue Unit
  dafür wäre doppelte Buchführung.
- offen: keine

---

## Iteration 46 — cov-gateway-biz-expenses-routes — done — 2026-08-23 04:44
- commit: cf1594ba
- gebaut:
  Alle sieben Handler in `route_biz_expenses.go` hatten vor dieser Iteration
  keinen einzigen HTTP-Level-Test (nur die Wire-Shape-Mapping-Tests in
  `route_biz_expenses_test.go` existierten). Neue Datei
  `route_biz_expenses_gate_test.go` nach dem etablierten Muster aus
  `route_biz_billing_test.go`/`route_datev_upload_test.go`: eine
  ServiceUnavailable- und eine NoTenant-Tabelle für alle sieben Handler auf
  einmal, danach je Handler der interessante Fehlerfall statt nur des
  200-Pfads.
  Zwei Prämissen aus dem Backlog-Scope geprüft und beide bestätigt statt
  widerlegt: (1) Ausgaben mit negativem Betrag (Erstattung) sind heute NICHT
  möglich — sowohl der Handler (`createExpenseRequest.Amount
  validate:"required,gt=0"`, `updateExpenseRequest.Amount
  validate:"omitempty,gt=0"`) als auch der Service (`parseAmount` ->
  `!amount.IsPositive()` -> `ErrInvalidAmount`) weisen einen Betrag <= 0
  unabhängig voneinander zurück. Das ist eine bestehende, konsistente
  Entscheidung, kein Fund — ob Erstattungen künftig unterstützt werden
  sollen, ist eine Datenmodell-/Produktfrage und nicht Gegenstand dieser
  Unit. Per Test belegt (negativ UND exakt null, getrennt getestet, weil ein
  Mutant nur einen der beiden Fälle bricht). (2) Belegupload/Archivierung:
  `route_biz_expenses.go` nimmt an keiner Stelle eine Datei entgegen —
  `HandleAttachExpenseReceipt` speichert nur einen vom Client gesendeten
  Dateinamen als String. Das ist bereits als `lean:`-Marker mit
  Upgrade-Trigger im Kopfkommentar von `internal/biz/expense/service.go`
  dokumentiert ("Add the presign upload from
  internal/chat/file/minio_store.go ... when the receipt has to be
  retrievable"). Kein 413- oder GoBD-Archiv-Test ist hier sinnvoll, weil es
  keinen Dateikörper gibt, den man ablehnen oder archivieren könnte — die
  Lücke ist bereits benannt, kein neuer Fund.
  Löschen nach Festschreibung: `expense.Delete` weist eine bereits
  entschiedene (approved/rejected) Ausgabe mit `ErrDecided` zurück
  (FailedPrecondition -> 409 am Gateway, wie bei allen übrigen
  "decided"-Fehlern). Ein GoBD-Perioden-Sperrmechanismus existiert in
  diesem Repo für KEINE Entität (Grep über `internal/biz` und
  `internal/models` nach PeriodLock/FiscalPeriod/IsLocked liefert 0
  Treffer) — nur `LockInvoice` sperrt einzelne Rechnungen manuell, und
  Expenses teilen diesen Mechanismus nicht. Der "Schutzgedanke" aus dem
  Backlog-Scope ist also bereits durch die bestehende
  Decided-Status-Sperre erfüllt; ein Periodensperren-Test wäre ein Test
  gegen einen Mechanismus, den es nicht gibt.
  20 neue Tests, alle grün: ServiceUnavailable (7 Subtests),
  NoTenant (7 Subtests), CreateExpense (fehlende Beschreibung, negativer
  Betrag, Nullbetrag, ungültiges Datumsformat, ungültiges JSON, gültiger
  Body erreicht die RPC), UpdateExpense (Teil-Body erreicht die RPC,
  negativer Betrag abgelehnt, ungültiges JSON), Approve/Reject (erreichen
  die RPC), AttachReceipt (fehlender Dateiname abgelehnt, gültiger Body
  erreicht die RPC), DeleteExpense (erreicht die RPC), ListExpenses
  (erreicht die RPC).
- gate: build ok (`-p 2`, gateway+cmd/gateway) | vet ok | lint ok
  (0 issues) | test ok (`go test -count=1 ./internal/gateway/...`) |
  migration n.a. (keine Tabelle/Spalte/Policy angefasst) | rls-smoke n.a. |
  `go test -count=1 -run TestOpenAPIRouteDrift ./internal/gateway/` grün
  (836 registrierte gegen 838 dokumentierte Pfade) — Pflicht, obwohl keine
  Route geändert wurde.
- coverage: internal/gateway 55,1 % -> 55,5 % (eigene Messung vor/nach,
  `go tool cover -func`, neue Testdatei per `mv` temporär entfernt für die
  Vorher-Messung, dann zurückgeschrieben; lokaler Ausgangswert weicht vom
  CI-Stand 54,1 % ab, weil dazwischen mehrere Iterationen dasselbe Paket
  bereits angehoben haben — siehe die vorangehenden Journal-Einträge).
- mutations-probe: zwei Läufe, beide gegen eine `cp`-Sicherungskopie (nicht
  `git checkout`), beide zurückgeschrieben, `git diff --stat` danach leer.
  (a) `Amount validate:"required,gt=0"` in `createExpenseRequest` auf
  `validate:"required"` verkürzt ->
  `TestHandleCreateExpense_NegativeAmountRejected` rot (503 statt 400,
  RPC erreicht statt lokal abgelehnt), `TestHandleCreateExpense_
  ZeroAmountRejected` blieb GRÜN (required allein fängt den Nullwert
  bereits ab) — zeigt, dass die beiden Tests wirklich unterschiedliche
  Aspekte der Regel prüfen, nicht denselben Pfad zweimal. (b)
  `attachReceiptRequest.ReceiptName` von `validate:"required,max=255"` auf
  `validate:"max=255"` gekürzt -> `TestHandleAttachExpenseReceipt_
  MissingReceiptName` rot (503 statt 400).
- verify vorgaenger: sauber — `1b5eaccb` (Iteration 45,
  cov-gateway-datev-upload-routes) ändert eine begründete
  OAuth-Fehlerklassifizierung (`ErrReauthRequired`, 4xx vs. 5xx vom
  DATEV-Token-Endpoint) plus Tests; keine der acht Fehlerklassen
  einschlägig: kein neuer Gateway-Handler, kein Stub/TODO, kein `.proto`,
  keine Migration, kein neuer/ersetzter `RequirePermission`-Guard, keine
  neue Tabelle, keine neue Route, kein Wire-Shape-Wechsel.
- neue-units: keine — beide geprüften Prämissen (negativer Betrag,
  Perioden-Sperre) bestätigten bestehendes, korrektes Verhalten statt einen
  Fund zu erzeugen; die Beleg-Archivierungslücke trägt bereits einen
  lean-Marker im Code.
- offen: `internal/biz/expense` hat keine DB-Tests und wurde nicht
  angefasst, DATABASE_URL-Gate daher für diese Unit nicht einschlägig —
  `go test ./internal/gateway/` lief trotzdem vollständig inklusive
  TestOpenAPIRouteDrift.

## Iteration 47 — cov-gateway-biz-bank-transactions-routes — done — 2026-08-23 04:51
- commit: 22b02b24
- gebaut: neue Datei `route_biz_bank_transactions_gate_test.go` mit 20 Tests
  über sechs bisher ungeprüfte Handler: `route_biz_bank_transactions.go`
  (HandleListBankTransactions, HandleMatchBankTransaction,
  HandleRejectBankTransactionMatch, HandleIgnoreBankTransaction) und
  `route_biz_transactions.go` (HandleListTransactions,
  HandleDeleteTransaction). ServiceUnavailable/NoTenant-Tabellen für alle
  sechs, dazu je Handler der interessante Fehlerfall: unbekannter
  `status`-Query-Wert (400, weil `bankMatchStatuses` eine geschlossene
  Whitelist ist), `invoice_id` ohne gültiges UUID4-Format,
  `invoice_number` über 64 Zeichen, ungültiges JSON.
- PRÄMISSENKORREKTUR (Regel 11): der Backlog-Scope beschrieb einen
  Datei-Upload als "Vertrauensgrenze" mit 400/413-Prüfung und einer
  Dublettenfrage für diese Unit. Beide gescopten Dateien haben aber
  KEINEN Upload-Handler — der CAMT.053/MT940-Import lebt in
  `route_biz_banking.go` (`HandleImportBankStatement`), einer dritten,
  bereits vollständig getesteten Datei
  (`route_biz_banking_test.go`: 400 bei ungültigem Multipart, 400 bei
  fehlender/leerer Datei, 413 bei Überschreitung, `AlreadyImported` für
  den Dublettenfall — alles schon belegt). Die vier hier gescopten
  Bank-Transaction-Handler bekommen fertige Zeilen aus einem
  vorangegangenen Import und haben keinen Dateikörper.
  Ebenso ist "Zuordnung auf eine Rechnung eines fremden Mandanten" kein
  Gateway-Fall: `ReconcileBankTransaction`
  (`internal/server/biz_grpc_banking.go:164`) reicht Tenant-ID und
  Invoice-ID unverändert an `internal/biz/banking` weiter, das die
  Rechnung tenant-gescoped lädt — eine fremde Rechnung ist dort "nicht
  gefunden", ohne echte RPC am Gateway nicht simulierbar
  (`biz_grpc_banking_bexio_test.go` deckt die RPC-Validierung bereits ab).
  Die Datei-Format-/Größen-/Dubletten-Fragen und die Cross-Tenant-Frage
  sind damit bereits an anderer Stelle beantwortet, nicht offen — kein
  Fund, keine neue Unit nötig.
  Abgrenzung der beiden Dateien (wie vom Scope gefordert, im Kommentarkopf
  der neuen Testdatei festgehalten): `route_biz_bank_transactions.go` ist
  die Reconciliation-Queue eines bereits importierten Kontoauszugs;
  `route_biz_transactions.go` ist die mandantenweite, konsolidierte
  Zahlungsübersicht über alle Belegarten. Kein gemeinsamer Typ, kein
  gemeinsamer Aufruf — die Nähe der Dateinamen war die einzige
  Verwechslungsquelle.
- gate: build ok (`-p 2`, gateway+cmd/gateway) | vet ok | lint ok
  (0 issues) | test ok (`go test -count=1 ./internal/gateway/...`) |
  migration n.a. (keine Tabelle/Spalte/Policy angefasst) | rls-smoke n.a. |
  `go test -count=1 ./internal/gateway/` inkl. `TestOpenAPIRouteDrift` grün
  — Pflicht, obwohl keine Route geändert wurde.
- coverage: internal/gateway 55,5 % -> 55,8 % (eigene Messung vor/nach,
  `go tool cover -func`, neue Testdatei per `mv` temporär entfernt für die
  Vorher-Messung, dann zurückgeschrieben; lokaler Ausgangswert weicht vom
  CI-Stand 54,1 % ab, weil vorangehende Iterationen dasselbe Paket
  bereits angehoben haben).
- mutations-probe: `bankMatchStatuses`-Check in `HandleListBankTransactions`
  testweise mit `if false &&` neutralisiert (Kopie via `cp`, nicht
  `git checkout`) -> `TestHandleListBankTransactions_UnknownStatusRejected`
  rot (503 statt 400, RPC erreicht statt lokal abgelehnt). Zurückgeschrieben,
  `git diff --stat` danach leer.
- verify vorgaenger: sauber — `cf1594ba` (Iteration 46,
  cov-gateway-biz-expenses-routes) fügt ausschließlich eine neue Testdatei
  hinzu; keine der acht Fehlerklassen einschlägig (kein neuer Handler, kein
  Stub/TODO, kein `.proto`, keine Migration, kein neuer/ersetzter
  `RequirePermission`-Guard, keine neue Tabelle, keine neue Route, kein
  Wire-Shape-Wechsel).
- neue-units: keine — beide geprüften Prämissen (Datei-Upload,
  Cross-Tenant-Zuordnung) waren bereits an anderer Stelle beantwortetes,
  bestehendes Verhalten, kein Fund.
- offen: keine.

## Iteration 48 — cov-gateway-biz-bank-accounts-routes — done — 2026-08-23 05:03
- commit: (folgt unten)
- gebaut: `route_biz_bank_accounts_gate_test.go` — HTTP-Level-Tests für alle
  fünf Handler (`HandleListBankAccounts`, `HandleCreateBankAccount`,
  `HandleUpdateBankAccount`, `HandleConnectBankAccount`,
  `HandleDeleteBankAccount`), vorher 0/5. ServiceUnavailable/NoTenant-Tabelle
  wie bei den Nachbarrouten, plus die für diese Route eigentlich interessante
  Grenze: die IBAN-Prüfziffer. Ein Test mit strukturell plausibler, aber
  falscher Prüfziffer (`DE89370400440532013001`, letzte Ziffer der Original-
  IBAN geflippt) und einer mit unbekanntem Länderpräfix (`XX...`) müssen 400
  liefern, bevor irgendeine RPC erreicht wird — beide tun das. Zusätzlich ein
  Test, der belegt, dass die abgelehnte IBAN nicht im Response-Body auftaucht
  (PII-Leckage-Klasse aus Lauf 10). Ungültige BIC, ungültige Währung
  ("EURO" statt dreistelligem Code), fehlender Bankname, kaputtes JSON auf
  Create und Update. Ein Positivfall mit menschlich getippter,
  leerzeichengruppierter IBAN und Kleinbuchstaben-Währung belegt, dass die
  Normalisierung vor der Prüfung greift.
  Zwei Prämissen aus dem Backlog-Scope widerlegt (Regel 11):
  (a) "Löschen eines Kontos mit Buchungen: 500 oder 409?" ist gegenstandslos.
  Migration 000258 legt bewusst KEINE Fremdschlüsselbeziehung zwischen
  `finance_bank_accounts` und `finance_bank_statements` an — die Zuordnung
  läuft über den freien Text `account_iban`, nicht über eine FK-Spalte.
  `DeleteAccount` (`postgres_repository_accounts.go:114`) ist ein reines
  `DELETE ... WHERE tenant_id = $1 AND id = $2` ohne jede Referenz auf
  Statements; ein Löschen kann nie an einer FK-Constraint scheitern, egal wie
  viele Buchungen unter derselben IBAN liegen.
  (b) "Fremdtenant darf weder lesen noch löschen" ist bereits service-seitig
  belegt (`TestDeleteAccount_StaysInsideTheTenant`,
  `service_accounts_test.go`) und in SQL erzwungen (`GetAccount`/
  `DeleteAccount` scopen immer `tenant_id = $1 AND id = $2`,
  `postgres_repository_accounts.go:64,116`) — am Gateway ohne echte RPC nicht
  simulierbar, wie bei den vorherigen Routen-Units.
  IBAN-Prüfziffer selbst ist real und korrekt implementiert
  (`dachfmt.ValidateIBAN`, ISO 7064 mod-97, `internal/dachfmt/iban.go:37`) —
  kein Fund, aber jetzt bewiesen statt angenommen. Kein IBAN-Wert erscheint
  in einem Log-Aufruf im gesamten `banking`-Paket (per Codelesung belegt,
  `service_accounts.go:6-8` dokumentiert das ausdrücklich).
- gate: build ok (`-p 2`, gateway+cmd/gateway) | vet ok | lint ok
  (0 issues) | test ok (`go test -count=1 ./internal/gateway/...`) |
  migration n.a. (keine Tabelle/Spalte/Policy angefasst) | rls-smoke n.a.
  (keine Policy angefasst) | `go test -count=1 ./internal/gateway/ -run
  TestOpenAPIRouteDrift` grün (836 registrierte gegen 838 dokumentierte
  Pfade) — Pflicht, obwohl keine Route geändert wurde.
- coverage: internal/gateway 55,8 % -> 56,1 % (eigene Messung vor/nach,
  `go tool cover -func`, neue Testdatei per `mv` temporär entfernt für die
  Vorher-Messung, dann zurückgeschrieben; lokaler Ausgangswert weicht vom
  CI-Stand 54,1 % ab, weil vorangehende Iterationen dasselbe Paket bereits
  angehoben haben — deckt sich mit dem `nachher`-Wert 55,8 % aus Iteration 47).
- mutations-probe: `dachfmt.ValidateIBAN` (`internal/dachfmt/iban.go:37`)
  testweise auf `return true` gesetzt (Prüfziffer deaktiviert) ->
  `TestHandleCreateBankAccount_WrongCheckDigitRejected` UND
  `TestHandleUpdateBankAccount_WrongCheckDigitRejected` rot (503 statt 400,
  RPC erreicht statt lokal abgelehnt). Zurückgeschrieben, `git diff --stat`
  danach leer.
- verify vorgaenger: sauber — `22b02b24` (Iteration 47,
  cov-gateway-biz-bank-transactions-routes) fügt ausschließlich eine neue
  Testdatei hinzu; keine der acht Fehlerklassen einschlägig (kein neuer
  Handler, kein Stub/TODO, kein `.proto`, keine Migration, kein neuer/
  ersetzter `RequirePermission`-Guard, keine neue Tabelle, keine neue Route,
  kein Wire-Shape-Wechsel).
- neue-units: keine — beide geprüften Prämissen (FK-Crash beim Löschen,
  Fremdtenant-Zugriff) waren bereits an anderer Stelle beantwortetes,
  bestehendes Verhalten, kein Fund.
- offen: keine.

## Iteration 49 — cov-gateway-integration-config-routes — done — 2026-08-23 05:03
- commit: (siehe unten)
- gebaut: Neue Testdatei `route_integration_config_test.go` fuer die dreizehn
  Konfigurations-/Mapping-Handler in `route_integration.go` (0/18 -> 13
  Handler jetzt namentlich getestet; die fuenf Slack/Teams-Webhook-/OAuth-
  Handler waren bereits durch `route_integration_test.go` abgedeckt und
  bleiben ausserhalb dieser Unit). Belegt, was die beiden bestehenden
  Testdateien pruefen: `route_integration_test.go` deckt Registrierung, die
  fuenf unauthentifizierten Webhook-/OAuth-Pfade (503/501, kein Panic) und
  dass alle zehn Admin-Routen hinter `RequireRole("admin")` sitzen (403) —
  keiner der Konfigurations-Handler wird dort namentlich aufgerufen.
  `route_integration_wiring_test.go` prueft nur die Erreichbarkeit ueber den
  echten `cmd/gateway/main.go`-Router. Diese Unit deckt das Gateway-Paket
  hat keinen bufconn/Fake-Client fuer `NotificationServiceClient` (dasselbe
  bereits in `route_lexware_test.go` dokumentierte Muster) — testbar sind
  daher der ServiceUnavailable-Pfad (leere Registry) und der RPC-Fehler-Pfad
  (registrierte, aber unerreichbare Dummy-Verbindung -> gRPC-Fehler -> 503),
  nicht ein erfolgreicher Roundtrip. `HandleCreateConfig` ist der einzige
  Handler, dessen Validierung vor jeder RPC laeuft (Platform kommt direkt im
  Body, keine vorgelagerte `GetIntegrationConfig`-Aufloesung) — dort sind
  InvalidJSON/MissingPlatform/MissingCredentialsVaultKey erreichbar.
  `HandleUpdateConfig`/`HandleDeleteConfig`/`HandleListMappings`/
  `HandleCreateMapping` loesen den Platform-Parameter zuerst per
  `GetIntegrationConfig`-RPC auf, die im Test immer zuerst fehlschlaegt —
  ihre 400-Pfade sind ohne funktionierenden Fake-Client nicht erreichbar,
  im Journal statt stillschweigend uebersprungen dokumentiert.
  `HandleUpdateMapping`/`HandleDeleteMapping` haben keine vorgelagerte RPC
  (nur `validateUUIDParam` bzw. zusaetzlich `decodeAndValidate`), daher dort
  alle 400-Pfade (invalid id, invalid JSON, missing channel_id/modules)
  vollstaendig getestet.
  ECHTER FUND, gefixt (nicht nur an D9 gemeldet, weil `done_when` dieser
  Unit "HandleTestConfig reicht keine Fremdsystem-Fehlermeldung durch"
  ausdruecklich verlangt): `TestIntegrationConfig`
  (`internal/server/notification_grpc.go:910-920`) baute den Fehler bei
  fehlgeschlagenem `prober.ProbeConnection` bisher als
  `status.Errorf(codes.FailedPrecondition, "%s rejected the connection
  test: %v", platform, err)` — der vierte Fund derselben Fehlerklasse, die
  Lauf 10 in dieser Nacht schon dreimal gefixt hat (Bexio, DATEV, Lexware):
  eine externe Systemantwort erreicht den Client unmaskiert. Konkret: der
  Teams-Client (`internal/notification/integration/teams/client.go:81-87`)
  haengt bis zu 2 KB der rohen AAD-Fehlerseite an den zurueckgegebenen
  Fehler, der Slack-Client die rohe `auth.test`-Fehlermeldung. Zwei Zeilen
  darueber im selben Handler (`GetConfigByPlatform`-Fehler, Zeile 898) laeuft
  bereits ueber die bestehende Maskierungsfunktion `mapNotificationError`
  (default-Zweig: `codes.Internal, "internal error"`), aber der
  ProbeConnection-Fehlerpfad umging sie komplett — dieselbe Asymmetrie wie
  bei den drei vorherigen Funden. Fix: Meldung an den Client bleibt bei
  `codes.FailedPrecondition` (409, kein "internal error", der Status bleibt
  aussagekraeftig) und dem bereits validierten `platform`-Namen, aber ohne
  `err`-Text; die volle Meldung geht weiterhin per `slog.Warn` serverseitig
  ins Log (unveraendert). Test
  `TestTestIntegrationConfigDoesNotLeakProbeErrorDetail`
  (`internal/server/notification_integration_test.go`) belegt das mit einem
  Fehlertext, der CRLF, eine Trace-ID und einen `Set-Cookie`-Versuch traegt.
  Praezisierung zum Backlog-Scope: der `sources`-Eintrag
  `backend/internal/biz/lexware/postgres_config_repo.go` ist falsch — das
  ist der Lexware-eigene Config-Repo, nicht das fuer Slack/Teams
  zustaendige. Die tatsaechliche Implementierung liegt in
  `internal/notification/integration/postgres_repository.go` und
  `internal/server/notification_grpc.go`; dort auch der Fund.
  Kein Zugangsdaten-Leck in Get/List/Create/Update: `IntegrationConfigInfo`
  (Proto) traegt `credentials_vault_key` bewusst nicht (Kommentar im
  `.proto` bestaetigt das) — als Reflection-Test
  `TestIntegrationConfigResponseProtos_NeverExposeCredentials` ueber alle
  zehn Response-Messages festgeschrieben (Vorlage:
  `TestLexwareResponseProtos_NeverExposeCredentials`), damit ein
  kuenftig hinzugefuegtes Feld den Test bricht statt den Client zu
  erreichen. Kein Handler in `route_integration.go` loggt
  `credentials_vault_key` oder `metadata` (per Codelesung belegt — die
  einzigen `slog`-Aufrufe im Datei-Ausschnitt sind `respondServiceUnavailable`
  und der generische Webhook-Fehlerpfad).
  Fremdtenant-Zugriff auf eine Mapping-ID: am Gateway ohne echte RPC nicht
  simulierbar (wie bei den vorherigen Routen-Units), aber auf Repository-
  Ebene bereits durch `tenant_isolation_phase2_test.go` und
  `tenant_write_test.go` (`internal/notification/integration/`) belegt —
  kein neuer Test noetig, keine offene Luecke.
- gate: build ok (`-p 2`, gateway+server+cmd/gateway) | vet ok | lint ok
  (0 issues, gateway+server) | test ok (`go test -count=1
  ./internal/gateway/...` und `./internal/server/...`, 0 SKIP in
  `internal/server`, 1858 PASS) | migration n.a. (keine Tabelle/Spalte/
  Policy angefasst) | rls-smoke n.a. | `go test -count=1 ./internal/gateway/
  -run TestOpenAPIRouteDrift` gruen (836 registrierte gegen 838
  dokumentierte Pfade) — Pflicht, obwohl keine Route geaendert wurde.
- coverage: internal/gateway 56,1 % -> 56,6 % (eigene Messung vor/nach,
  neue Testdatei per `mv` temporaer entfernt fuer die Vorher-Messung, dann
  zurueckgeschrieben; lokaler Ausgangswert weicht vom CI-Stand 54,1 % ab,
  weil vorangehende Iterationen dasselbe Paket bereits angehoben haben).
  internal/server 70,7 % -> 70,7 % (eigene Messung per `git stash` der
  beiden geaenderten Dateien — der neue Test und der Ein-Zeilen-Fix sind zu
  klein, um die gerundete Prozentzahl des grossen Pakets zu bewegen; Fix-
  Nebenprodukt, kein Coverage-Ziel dieser Unit).
- mutations-probe: `notification_grpc.go`s maskierte Meldung
  (`"%s rejected the connection test", platform`) zurueck auf die
  urspruengliche Leck-Zeile gesetzt (`"%s rejected the connection test: %v",
  platform, err`) -> `TestTestIntegrationConfigDoesNotLeakProbeErrorDetail`
  bricht ab und zeigt den vollen AAD-Fehlertext inkl. CRLF, Trace-ID und
  `Set-Cookie`-Versuch in der Fehlermeldung. Zurueckgedreht, `git diff
  --stat` zeigt wieder exakt 17 Einfuegungen/10 Loeschungen wie vor der
  Probe (die eigentliche Fix-Aenderung inkl. erklaerendem Kommentar), Test
  erneut gruen.
- verify vorgaenger: sauber. `c599ef40` (Iteration 48,
  cov-gateway-biz-bank-accounts-routes) geprueft: `git show --stat` zeigt
  ausschliesslich eine neue Testdatei plus BACKLOG.yml/JOURNAL.md — kein
  neuer Handler, kein Stub/TODO, kein `.proto`-Change, keine Migration,
  kein neuer/ersetzter `RequirePermission`-Guard, keine neue Tabelle, keine
  neue Route, kein Wire-Shape-Wechsel; keine der acht Fehlerklassen
  einschlaegig.
- neue-units: keine. Der einzige echte Fund (Fremdsystem-Fehlertext-Leak in
  `TestIntegrationConfig`) wurde direkt in dieser Unit behoben, weil
  `done_when` genau das verlangt — kein Backlog-Eintrag noetig.
- offen: die vier unreachable 400-Pfade (Update/Delete Config,
  ListMappings, CreateMapping — alle mit vorgelagertem
  `GetIntegrationConfig`-RPC) bleiben ohne Fake-Client ungetestet; dasselbe
  strukturelle Limit wie bei `route_lexware_test.go`, kein neuer Befund.

## Iteration 50 — cov-datev-upload-service-error-paths — done — 2026-08-23 05:14
- commit: e2d2a490
- gebaut: Zehn neue Tests fuer `internal/biz/datev` (upload_service.go +
  postgres_upload_repo.go), keine Produktionscode-Aenderung. Service-Ebene:
  `UploadBuchungsstapel` ohne Builder (ErrBuilderNotConfigured) und mit
  fehlschlagendem `builder.Build` (propagiert ErrInvalidPeriod unveraendert);
  `UploadInvoiceBeleg` ohne belegRenderer/belegUploader (zwei getrennte
  Preconditions, vorher beide ungetestet); zwei Tests belegen, dass ein
  fehlschlagendes `CreateUploadLog` (Audit-Log-INSERT) weder
  `UploadBuchungsstapel` noch `UploadBeleg` blockiert — der Transfer laeuft
  durch, nur der Log-Schreibversuch wird geloggt und verworfen (bestehendes
  Verhalten, jetzt belegt statt implizit). Repository-Ebene: ein neuer
  RLS-Test `TestListUploadLogs_InvisibleToForeignTenant` liest ueber
  `ListUploadLogs` (nicht nur `AssertRowCount` wie der bestehende Write-Test)
  mit dem echten `config_id` eines fremden Tenants — 0 Zeilen trotz
  korrekter ID, weil RLS filtert, nicht die WHERE-Klausel der Query.
- gate: build ok (`-p 2`, internal/biz/datev) | vet ok | lint ok (0 issues)
  | test ok (`go test -count=1 -v ./internal/biz/datev/`, 98 PASS, 0 SKIP)
  | migration n.a. | rls-smoke ok (siehe TestListUploadLogs_InvisibleToForeignTenant
  oben — eigener Tenant liefert die Zeile, fremder Tenant 0) | OpenAPI-Drift
  n.a. (kein Route-/Handler-Code angefasst, nur `internal/biz/datev`)
- coverage: internal/biz/datev 79,6 % -> 80,7 % (eigene Messung vor/nach mit
  `go tool cover -func`, DATABASE_URL gegen kmuhub_app, 0 DB-Tests
  uebersprungen in beiden Laeufen)
- mutations-probe: in `UploadBuchungsstapel` testweise ein `return nil, err`
  nach dem `CreateUploadLog`-Fehlerzweig eingefuegt (simuliert die
  naheliegende Fehlkorrektur "Log-Fehler soll den Transfer blockieren") ->
  `TestUploadBuchungsstapel_TransferSucceedsEvenWhenLogWriteFails` wird rot
  ("UploadBuchungsstapel: insert failed, want the transfer to proceed..."),
  zurueckgedreht, `git diff --stat backend/internal/biz/datev/upload_service.go`
  zeigt wieder keine Aenderung (nur die beiden Testdateien im finalen Diff).
- verify vorgaenger: sauber. `01c59c57` (Iteration 49,
  cov-gateway-integration-config-routes) geprueft: `git show --stat` zeigt
  eine neue Testdatei, einen Ein-Zeilen-Fix in `notification_grpc.go`
  (Fremdsystem-Fehlertext maskiert, Kommentar erklaert die vierte Instanz
  dieser Leck-Klasse in diesem Lauf) plus dessen eigenen Test, keine
  Proto-Aenderung, keine Migration, kein neuer/ersetzter
  RequirePermission-Guard, keine neue Route, kein Wire-Shape-Wechsel; keine
  der acht Fehlerklassen einschlaegig.
- neue-units: fix-datev-upload-log-stuck-uploading-no-reconciliation (Fund:
  `datev_upload_log` kann bei einem Prozessabbruch zwischen CreateUploadLog
  und dem Transferergebnis fuer immer im Status "uploading" haengen bleiben
  — kein Cron/Scheduler prueft darauf, `ListUploadLogs` zeigt eine verwaiste
  Zeile ununterscheidbar von einer echten. Ausweg bewusst nicht selbst
  gebaut: ein Reconciliation-Mechanismus ist eine `cmd/`-Wiring-Entscheidung
  und gehoert nach den Lauf-Leitplanken nicht in eine Coverage-Unit).
- offen: Der Zeilenzweig 124-126 in `upload_service.go` (Disconnect,
  `RevokeTokens`-Fehlerpfad) bleibt ungetestet — `oauthManager` ist ein
  konkreter `*OAuthManager`-Pointer, keine Interface, und `RevokeTokens`
  kann in der aktuellen Implementierung (reines In-Memory-Cache-Delete)
  nie einen Fehler liefern. Eine Interface-Extraktion nur fuer diese eine
  Log-Zeile waere Overengineering fuer eine Coverage-Unit — nicht gebaut,
  keine Folge-Unit noetig (kein Bug, nur eine strukturell unerreichbare
  Zeile). Die 4xx/5xx-Retry-Unterscheidung aus `done_when` ist bereits in
  `uploader.go`/`uploader_test.go` (TestUploadBuchungsstapel_DoesNotRetry4xx,
  TestUploadBuchungsstapel_Retries5xxThenSucceeds) vollstaendig abgedeckt —
  gehoert zur naechsten Unit `cov-datev-uploader-oauth-token-refresh`
  (deps darauf), hier nur verifiziert, nicht dupliziert.

## Iteration 51 — cov-datev-uploader-oauth-token-refresh — done — 2026-08-23 05:22
- commit: 9a48cc01
- gebaut: Zwei neue Tests in `internal/biz/datev`, keine Produktionscode-
  Aenderung. `TestGetAccessToken_ConcurrentColdCacheCallsBothHitTokenEndpoint`
  (oauth_test.go) belegt deterministisch (Server-seitige Rendezvous-Schranke
  mit 2s-Timeout-Fallback statt Wanduhr-Abhaengigkeit) den im Scope
  vermuteten Fund: `GetAccessToken` haelt kein Per-Tenant-Lock um Cache-Read
  und Refresh — zwei parallele Aufrufe mit kaltem Cache loesen beide
  unabhaengig `RefreshAccessToken` aus und senden dabei denselben (noch
  nicht rotierten) Refresh-Token an den Token-Endpoint. Bei einem
  Token-Server mit Rotation wuerde einer der beiden Requests den Token des
  anderen verbrennen und mit ErrReauthRequired zurueckkommen, obwohl die
  Verbindung Sekunden zuvor gueltig war. `TestUploadBuchungsstapel_
  ReauthRequiredIsNotRetried` (uploader_test.go) belegt zusaetzlich, dass
  ErrReauthRequired durch `doWithRetry`s `fmt.Errorf("...: %w", err)`-Wrap
  hindurch per `errors.Is` erreichbar bleibt und die Upload-Route nie
  kontaktiert wird (0 Calls) — die Kette bis zur admin-verstaendlichen
  "reconnect"-Meldung ist bereits serverseitig getestet
  (datev_upload_grpc_test.go:36-78, mapDatevUploadError), diese Unit deckt
  den fehlenden Zwischenschritt (Uploader-Ebene) ab.
- gate: build ok (`-p 2`, internal/biz/datev) | vet ok | lint ok (0 issues)
  | test ok (`go test -count=1 -v ./internal/biz/datev/`, 100 PASS, 0 SKIP)
  | migration n.a. | rls-smoke n.a. (kein DB-/Tabellen-Zugriff in dieser
  Unit) | OpenAPI-Drift n.a. (kein Route-/Handler-Code angefasst)
- coverage: internal/biz/datev 80,7 % -> 80,7 % (eigene Messung vor/nach
  mit `go tool cover -func`, DATABASE_URL gegen kmuhub_app). Unveraendert,
  weil beide neuen Tests bereits durchlaufene Codezeilen erneut ausueben
  (der Token-Fehlerzweig lief schon ueber `TestUploadBuchungsstapel_
  TokenErrorIsNotRetried`, die Concurrency ueber denselben `RefreshAccessToken`-
  Pfad wie die bestehenden Tests) — der Wert dieser Unit ist ein belegtes
  Verhalten (Nebenlaeufigkeitsluecke), keine neue Zeilenabdeckung.
- mutations-probe: in `GetAccessToken` testweise ein globales
  `sync.Mutex` um Cache-Read+Refresh gelegt (simuliert die naheliegende
  Serialisierung als Fix) -> `TestGetAccessToken_ConcurrentColdCacheCallsBothHitTokenEndpoint`
  wird rot ("token endpoint called 1 times, want 2"), zurueckgedreht,
  `git diff --stat backend/internal/biz/datev/oauth.go` zeigt keine
  Aenderung (leer).
- verify vorgaenger: sauber. `e2d2a490` (Iteration 50,
  cov-datev-upload-service-error-paths) geprueft: `git show --stat` zeigt
  ausschliesslich zwei neue/erweiterte Testdateien plus BACKLOG.yml/
  JOURNAL.md — kein neuer Handler, kein Stub/TODO, kein `.proto`-Change,
  keine Migration, kein neuer/ersetzter RequirePermission-Guard, keine neue
  Tabelle, keine neue Route, kein Wire-Shape-Wechsel; keine der acht
  Fehlerklassen einschlaegig.
- neue-units: keine. Der im Scope vermutete Fund wurde in dieser Unit
  selbst als Test dokumentiert (per done_when explizit "das Rennen selbst
  ist an CI verwiesen", kein Backend-Fix vorgesehen) — ein tatsaechlicher
  Fix (Per-Tenant-Lock um Refresh) waere eine Verhaltensaenderung auf einem
  Produktionspfad und gehoert nicht in eine Coverage-Unit; falls Luke einen
  Fix will, ist das eine eigene, von ihm freizugebende Unit (Refresh-Token-
  Rotation bei DATEV ist aktuell nicht bestaetigt aktiv, daher kein
  akuter Schaden ohne weitere Klaerung).
- offen: Ob DATEV Unternehmen online Refresh-Token-Rotation tatsaechlich
  einsetzt, ist unbelegt (kein Zugriff auf DATEV-Doku aus diesem Lauf) —
  falls ja, ist ein Per-Tenant-Lock in `GetAccessToken` die naheliegende
  Fix-Richtung (siehe Mutations-Probe oben als Skizze). Die -race-Bestaetigung
  des Rennens selbst bleibt CI vorbehalten (kein gcc lokal).

## Iteration 52 — cov-einvoice-parser-foreign-format-inbound — done — 2026-08-23 05:38
- commit: b6da7373
- gebaut: Neue Testdatei `parser_inbound_hardening_test.go` (11 Tests) plus
  eine Produktionscode-Ergaenzung in `parser.go`: `assertInboundTotalsConsistent`
  (aufgerufen am Ende von `ParseCII` und `ParseUBL`) lehnt ein Fremddokument
  ab, dessen eigene deklarierte Zahlen nicht zusammenpassen — Summe der
  Positionszeilen vs. deklariertem Zwischensumme-Header, und Zwischensumme
  plus Steuer vs. deklariertem Bruttobetrag. Toleranz ist die bereits
  bestehende `totalsTolerance(lineCount)` aus `generator_doc.go` (halber Cent
  je Zeile), damit ein vom eigenen Ausgangspfad erzeugtes Dokument garantiert
  roundtrip-faehig bleibt. Vor dieser Aenderung wurden Subtotal/Tax/Gross aus
  dem Fremd-XML ungeprueft direkt in `Service.Import` persistiert
  (`Status: IncomingInvoiceStatusReceived`) — ein manipuliertes oder defektes
  Dokument haette unbemerkt eine falsche Buchung erzeugt (der im Scope
  beschriebene Fund, jetzt behoben).
  Sicherheitsfaelle geprueft und mit Test belegt: XXE (externe Entity via
  SYSTEM-DOCTYPE) und Billion-Laughs (verschachtelte Custom-Entities) werden
  von Gos `encoding/xml` beide bereits ohne jede Aenderung abgewiesen
  ("invalid character entity", kein Entity-Map gesetzt) — verifiziert per
  Scratch-Test vor dem Schreiben der Assertions, nicht geglaubt. Tiefe
  Verschachtelung (50k Ebenen in einem vom CII-Struct nicht gemappten,
  uebersprungenen Element) und ein grosses Dokument (3.000 Positionszeilen)
  laufen beide gebunden durch (Timeout-Wrapper 10s/5s als Absicherung gegen
  einen haengenden Parser, tatsaechliche Laufzeit < 100ms). Unbekannte
  Waehrung und unbekannte USt-Kategorie-ID sind bewusst NICHT abgewiesen
  (nur der numerische Satz wird beim UBL-Import ueberhaupt gelesen) — als
  `lean:`-Marker mit Upgrade-Trigger in der neuen Testdatei dokumentiert,
  nicht als Fix verkauft (Begruendung: Fremdwaehrung ist keine Sache, die
  der Parser beim Einlesen ablehnen darf, und Eingangsrechnungen werden vor
  dem Verbuchen manuell geprueft).
- gate: build ok (`-p 2`, internal/biz/einvoice) | vet ok | lint ok
  (golangci-lint, 0 issues) | test ok (`go test -count=1 -v`, 75 PASS,
  0 SKIP, 0 FAIL) | migration n.a. (keine Tabelle/Policy angefasst) |
  rls-smoke n.a. | OpenAPI-Drift n.a. (kein Route-/Handler-Code, `go test
  ./internal/gateway/` daher nicht Pflicht in dieser Unit)
- coverage: internal/biz/einvoice 82,5 % -> 85,9 % (eigene Messung vor/nach
  per `git stash`/`go test -coverprofile` und `go tool cover -func`,
  DATABASE_URL gegen kmuhub_app). `coverage_start:` in der Unit nennt 81,9 %
  (CI-Stand vor A5–A8); das Paket hat seither mehrfach Code bekommen, daher
  gilt die eigene 82,5-%-Vorher-Messung, nicht der Backlog-Wert.
- mutations-probe: `tolerance` in `assertInboundTotalsConsistent` testweise
  auf `decimal.New(999999, 0)` gesetzt (macht die Pruefung wirkungslos) ->
  alle vier neuen `TotalsMismatch`-Tests (CII x2, UBL x2) werden rot
  ("An error is expected but got nil"), zurueckgedreht,
  `git diff --stat backend/internal/biz/einvoice/parser.go` zeigt wieder
  exakt 38 Zeilen Zusatz (nur die beabsichtigte Aenderung, keine Reste).
- verify vorgaenger: sauber. `9a48cc01` (Iteration 51,
  cov-datev-uploader-oauth-token-refresh) geprueft: `git show --stat` zeigt
  ausschliesslich zwei neue/erweiterte Testdateien plus BACKLOG.yml/
  JOURNAL.md — kein neuer Handler, kein direkter Service-Aufruf im Gateway,
  kein Stub/TODO, kein `.proto`-Change, keine Migration, kein neuer/ersetzter
  RequirePermission-Guard, keine neue Tabelle, keine neue Route, kein
  Wire-Shape-Wechsel, kein hart ersetzter Guard; keine der acht
  Fehlerklassen einschlaegig.
- neue-units: keine. Der einzige echte Fund (Summenpruefung fehlte) ist in
  dieser Unit selbst behoben, nicht nur dokumentiert — das war nach
  Scope/Notes/done_when explizit gefordert ("Ein Dokument, dessen Summen
  nicht aufgehen, darf NICHT stillschweigend importiert werden"), anders als
  bei reinen Coverage-Units sonst ueblich.
- offen: Die Waehrungs-/Kategorie-Whitelist aus A5
  (feat-einvoice-codelist-validation) deckt nur den OUTBOUND-Generator ab;
  ob Eingangsrechnungen ebenfalls gegen sie laufen sollen, ist eine
  Produktentscheidung (Upgrade-Trigger steht als Kommentar in der neuen
  Testdatei). `pdf_extract.go` selbst ist unveraendert und hatte bereits
  volle Testabdeckung vor dieser Unit (`pdf_extract_test.go`) — kein neuer
  Test dort noetig, im Scope aber mitgelesen.

## Iteration 53 — cov-server-biz-grpc-money-error-mapping — done — 2026-08-23 05:36
- commit: 920a7f73
- gebaut: Erst die volle RPC-Liste in `biz_grpc.go` gegen alle bestehenden
  Error-Mapping-Tests abgeglichen (`biz_grpc_errormap_settings_quotes_test.go`,
  `biz_grpc_invoices_creditnotes_payments_test.go`,
  `biz_grpc_dunning_dashboard_exports_test.go`, `biz_grpc_dunning_test.go`) —
  `mapBizError` selbst ist bereits vollstaendig per Tabellentest abgedeckt
  (jeder Sentinel einzeln), und praktisch jeder Geld-RPC-Fehlerpfad hatte
  schon einen Test. Zwei echte Luecken gefunden und behoben, beide root-cause
  in `biz_grpc.go`, keine Symptom-Guards:
  1. `GenerateDunningPDF` (Zeile ~1312): der Ladefehler der verknuepften
     Rechnung (`s.invoiceService.GetByID(ctx, tenantID, dr.InvoiceID)`) war
     hart auf `codes.Internal` verdrahtet, unabhaengig vom Fehlertyp. Eine
     nicht mehr existierende/verknuepfte Rechnung (`invoice.ErrInvoiceNotFound`)
     kam beim Client als 500 an statt als NotFound — die einzige Stelle im
     ganzen `biz_grpc.go`, an der eine domaenenspezifische NotFound-Situation
     nicht durch `mapBizError` lief. Fix: `mapBizError(err)` statt der festen
     Meldung; ein wirklich opaker Fehler (DB down o.ae.) faellt weiterhin
     durch `mapBizError`s Default-Case auf Internal.
  2. `CreateInvoiceFromTimeEntries` (Zeile ~2042): der Fehler aus
     `s.timetrackingRepo.ReserveWorkTimeForInvoice` ist ein roher
     pgx/Treiber-Fehler ohne Sentinel-Wrapping (siehe
     `postgres_repository.go:545-603` — `tx.Begin`/`tx.Query`/`tx.Exec`-Fehler
     werden unveraendert durchgereicht). Der Handler baute daraus
     `fmt.Sprintf("reserve work time: %s", err.Error())` und gab das als
     `codes.Internal`-Statusmeldung an den Client — ein SQL-Fehler (Constraint-
     Name, ggf. Spaltenname) auf dem Finanz-Abschlusspfad haette den API-Aufrufer
     erreicht. Dieselbe Fehlerklasse, die Lauf 10 mit
     `scan-gateway-sql-error-leakage` auf der Gateway-Ebene dreimal gefixt hat,
     hier aber auf der gRPC-Ebene noch nie geprueft. Fix: generische Meldung
     ("failed to reserve work time entries"), voller Fehler bleibt im
     `slog.Error` (serverseitig, nicht im Response).
  Vier neue Testdateien-Erweiterungen (keine neue Datei, bestehende Form
  wiederverwendet — Notes verlangten ausdruecklich "dieselbe Form anwenden,
  nicht eine zweite daneben"):
  - `biz_grpc_dunning_dashboard_exports_test.go`:
    `TestGenerateDunningPDF_LinkedInvoiceLoadFailure` (zwei Subtests: Sentinel
    -> NotFound, opaker Fehler -> weiterhin Internal).
  - `biz_grpc_invoices_creditnotes_payments_test.go`: neuer
    `stubWorkTimeRepo` (volles 14-Methoden-Interface
    `timetracking.WorkTimeRepository`, nur die drei von
    `CreateInvoiceFromTimeEntries` tatsaechlich aufgerufenen Methoden
    konfigurierbar) plus vier neue Tests — die bisherige Einschraenkung
    "Happy Path braucht volles Fake, out of scope" (Kommentar im Code seit
    der vorigen Invoice-Coverage-Iteration) ist damit aufgeloest:
    `TestCreateInvoiceFromTimeEntries_ReserveErrorDoesNotLeak` (der Fund),
    `TestCreateInvoiceFromTimeEntries_NoCompletedEntries` (0-Minuten-Pfad,
    FailedPrecondition), `TestCreateInvoiceFromTimeEntries_HappyPath` mit
    zwei Subtests (voller Erfolgspfad inkl. Confirm-Aufruf; Invoice-Create
    schlaegt fehl -> Reservierung wird per `ReleaseInvoiceReservation`
    freigegeben, damit die Zeiteintraege nicht fuer immer "billed" haengen
    bleiben).
- gate: build ok (`-p 2`, internal/server + internal/gateway) | vet ok |
  lint ok (golangci-lint, 0 issues) | test ok (`go test -count=1 -v`, 1862
  PASS, 0 FAIL, 0 SKIP in internal/server; `internal/server/response`
  ebenfalls gruen) | migration n.a. (keine Tabelle/Policy angefasst) |
  rls-smoke n.a. | OpenAPI-Drift n.a. (kein Route-/Handler-Code im Gateway
  angefasst, nur `internal/server`; `go test ./internal/gateway/` trotzdem
  mitgelaufen als Teil des `-p 2`-Builds, keine separate Pflicht hier)
- coverage: internal/server 70,7 % -> 70,8 % (eigene Messung vor/nach per
  `git stash`/`go test -coverprofile` und `go tool cover -func`, DATABASE_URL
  gegen kmuhub_app). `coverage_start:` in der Unit nennt 70,5 % (CI-Stand);
  die eigene 70,7-%-Vorher-Messung gilt, da das Paket seit A1 (Gruppierung
  aus dem Handler entfernt) und weiteren Iterationen Code verloren/gewonnen
  hat. Kleine Verschiebung, weil der Fund selbst klein war (zwei Zeilen
  Handler-Code) und der Rest der Arbeit bestehende Luecken bestaetigte statt
  neue Pfade zu schaffen.
- mutations-probe: beide Fixes einzeln zurueckgedreht und beweisend rot
  gemacht, dann sauber zurueckgedreht. (1) `mapBizError(err)` in
  `GenerateDunningPDF` durch `status.Error(codes.Internal, "MUTATION_PROBE")`
  ersetzt -> `TestGenerateDunningPDF_LinkedInvoiceLoadFailure/invoice_not_found...`
  wird rot ("expected gRPC code NotFound, got Internal: MUTATION_PROBE").
  (2) generische Meldung in `CreateInvoiceFromTimeEntries` durch das alte
  `fmt.Sprintf("reserve work time: %s", err.Error())` ersetzt ->
  `TestCreateInvoiceFromTimeEntries_ReserveErrorDoesNotLeak` wird rot (beide
  `NotContains`-Assertions schlagen fehl, der rohe Postgres-Fehlertext
  erscheint wieder in der gRPC-Meldung). Beide zurueckgedreht,
  `git diff --stat backend/internal/server/biz_grpc.go` zeigt danach wieder
  exakt 11 Einfuegungen/3 Loeschungen (nur die zwei beabsichtigten Fixes).
- verify vorgaenger: sauber. `b6da7373` (Iteration 52,
  cov-einvoice-parser-foreign-format-inbound) geprueft: `git show --stat`
  zeigt ausschliesslich `parser.go` (38 Zeilen, kein neuer Handler/Service-
  Direktaufruf) plus eine neue Testdatei — `ErrParseFailed` bereits vorhanden
  und konsistent verwendet, kein Stub/TODO, kein `.proto`-Change, keine
  Migration, kein neuer/ersetzter RequirePermission-Guard, keine neue
  Tabelle, keine neue Route, kein Wire-Shape-Wechsel; keine der acht
  Fehlerklassen einschlaegig.
- neue-units: keine. Beide gefundenen Bugs (hartes Internal statt
  mapBizError, SQL-Fehlertext im gRPC-Status) sind in dieser Unit selbst
  behoben, nicht nur dokumentiert — root-cause-Fixes von je einer Zeile im
  bereits angefassten Handler, kein Deploy-Hazard, keine neue Route, keine
  Verhaltensaenderung ausserhalb der Fehlerpfad-Korrektheit. Die uebrigen
  Geld-RPCs (Quote/Invoice/CreditNote/Payment/Dunning/ExportDATEV/
  GenerateGoBDExport/GetFinanceDashboard) hatten bereits vollstaendige
  Error-Mapping-Tests aus frueheren Iterationen; keine weiteren Internal-
  Fehlmappings oder Fremdsystem-/SQL-Leckagen gefunden (gezielt nach
  `status.Error(codes.Internal` und `err.Error()` in Statusmeldungen
  gegrept, alle Treffer einzeln gegen ihre Fehlerquelle geprueft).
- offen: keine offenen DB-/Proto-/Route-Fragen. Der aufgeloeste Kommentar
  "Happy Path braucht volles WorkTimeRepository-Fake, out of scope" stand
  noch im Code der vorigen Invoice-Coverage-Iteration als Abschnitts-
  kommentar ueber `TestCreateInvoiceFromTimeEntries_Validation` — der
  Kommentar selbst wurde nicht angefasst (beschreibt weiterhin korrekt, was
  in diesem Test-Func steht), der neue `stubWorkTimeRepo` liegt in einem
  eigenen Abschnitt direkt danach.

## Iteration 54 — cov-gateway-document-chains-and-open-items-routes — done — 2026-08-23 05:45
- commit: (siehe naechster Commit, unmittelbar nach diesem Journal-Eintrag)
- gebaut: kein Fix, keine neue Route. Ein echter Coverage-Fund behoben
  (`groupThousands` in `route_biz_document_chains.go:95`: der Zweig
  `if first == 0 { first = 3 }` war durch keinen Bestandstest erreichbar,
  weil alle drei bisherigen Faelle 4 oder 7 Stellen hatten, first also nie 0
  wurde — ein neuer Fall mit sechs Stellen ("123456.00" -> "EUR 123.456,00")
  schliesst genau diese Luecke). Die drei uebrigen Praemissen aus dem
  Backlog-Scope sind gegen Code und Migrationen geprueft und ALS FALSCH
  bzw. BEREITS ERLEDIGT widerlegt, mit Beleg als Kommentar in den beiden
  Testdateien statt als stiller Nichtbefund:
  (1) Rundung: `formatChainAmount`s `Round(2)` rundet nie tatsaechlich etwas
  weg, weil alle drei Quellspalten (`finance_invoices.gross_total`,
  `finance_payments.amount`, `finance_credit_notes.gross_total`) NUMERIC(15,2)
  sind (migrations/000045) und jede im Gateway ankommende Zahl entweder
  direkt aus einer dieser Spalten oder aus einer Subtraktion zweier solcher
  Werte stammt — beides kann nie mehr als zwei Nachkommastellen haben.
  Ueberfluessig, keine Divergenz. In die D4-Scan-Unit
  (`scan-money-rounding-and-tax-call-sites`) als vorab beantwortete Stelle
  eingetragen.
  (2) Faelligkeitsgrenze: `HandleListOpenItems` klassifiziert selbst nichts
  (bucket ist ein durchgereichter Query-String) — die eigentliche
  Grenzwertlogik (`AgingBucketIndexFor`) hat bereits exakte
  Grenzwerttests (0/30/60 und Grenze+1) in
  `internal/biz/dunning/service_open_items_test.go`, die SQL-Seite in
  `internal/biz/invoice/open_items_chains_helpers_test.go`. Eine dritte
  Kopie am Gateway wuerde nur `parsePagination` testen, nicht die Grenze.
  (3) Belegkette ohne Angebot / Gutschrift ohne Rechnung: eine Rechnung ohne
  Angebot ist der Normalfall (quoteID nil ueberspringt nur den Quote-Knoten,
  keine Sonderbehandlung); eine Gutschrift ohne Rechnung kann nicht
  existieren (`finance_credit_notes.original_invoice_id` ist NOT NULL mit
  ON DELETE RESTRICT FK, migrations/000045). Beide Faelle sind Knotenzahl 1,
  und `toDocumentChainWire` behandelt 0/1/2-Knoten-Faelle identisch (Proto-
  Getter sind nil-sicher) — kein Zweig, der brechen koennte, bereits durch
  `TestToDocumentChainWire_EmptyNumberAndDate_UseEmDash` und
  `TestToDocumentChainWire_NoNodes_ProducesEmptySliceNotNil` belegt.
  Die vor diesem Lauf schon existierenden Testdateien
  (`route_biz_document_chains_test.go`, `route_biz_open_items_time_entries_test.go`,
  Commits `2b621da2`/`e8b862e9` aus Lauf 10) hatten die im Backlog-Draft
  genannten Zeilenverhaeltnisse (137/187, 66/132) bereits ueberholt — Draft
  war zum Zeitpunkt der Vorbereitung (2026-08-22) nicht mehr aktuell.
- gate: build ok (`-p 2`, internal/gateway + cmd/gateway) | vet ok | lint ok
  (golangci-lint, 0 issues) | test ok (`go test -count=1`, internal/gateway
  gruen, inkl. `TestOpenAPIRouteDrift` explizit: 836 registrierte Routen
  gegen 838 dokumentierte Pfade, PASS — keine Route veraendert) | migration
  n.a. | rls-smoke n.a.
- coverage: internal/gateway 56,6 % -> 56,6 % (eigene Messung vor/nach per
  `go test -coverprofile` + `go tool cover -func`, DATABASE_URL gegen
  kmuhub_app; `coverage_start:` der Unit nennt 54,1 % CI-Stand, die
  Abweichung kommt von den vielen Iterationen dieses Laufs seit dem
  CI-Stand). Auf Gesamtpaketebene rundet eine einzelne neue Testzeile weg,
  der eigene Beitrag ist auf Funktionsebene sichtbar:
  `route_biz_document_chains.go` `groupThousands` 90,9 % -> 100,0 %.
- mutations-probe: `first = 3` in `groupThousands` (Zeile 102) zu
  `first = 1` geaendert -> `TestFormatChainAmount_GroupingBoundary` wird rot
  (Panic: `slice bounds out of range` beim Aufbau von "123456.00", da
  `digits[:1]` + Schrittweite 3 ab Index 1 den String falsch zerlegt).
  Zurueckgedreht per `git checkout -- route_biz_document_chains.go` (der
  sed-Edit hatte nur die Zeilenenden der einen Zeile beruehrt, `git diff`
  zeigte danach 0 inhaltliche Aenderungen; sicherheitshalber trotzdem per
  Checkout exakt auf den Ausgangsstand zurueckgesetzt). `git status` zeigt
  fuer diese Datei jetzt clean.
- verify vorgaenger: sauber. `920a7f73` (Iteration 53) geprueft: `git show
  --stat` zeigt nur `internal/server/biz_grpc.go` (14 Zeilen) plus zwei
  Testdateien, kein neuer Handler/Service-Direktaufruf, kein Stub/TODO, kein
  `.proto`-Change, keine Migration, kein neuer/ersetzter RequirePermission-
  Guard, keine neue Tabelle, keine neue Route, kein Wire-Shape-Wechsel;
  `GenerateDunningPDF` und `CreateInvoiceFromTimeEntries` routen jetzt beide
  ueber `mapBizError`/eine generische Meldung statt Internal-Hardcode bzw.
  rohem SQL-Fehlertext — keine der acht Fehlerklassen einschlaegig.
- neue-units: keine. Der einzige echte Fund (groupThousands-Luecke) ist in
  dieser Unit selbst behoben; die drei widerlegten Praemissen brauchen keine
  Folge-Unit, weil sie keine echten Luecken sind (Beleg jeweils als
  Code-Kommentar hinterlegt, D4 zusaetzlich vorab beantwortet).
- offen: keine. Die 838-vs-836-Differenz bei TestOpenAPIRouteDrift ist
  vorbestehend und unveraendert durch diese Unit (keine Route angefasst) —
  nicht neu, nicht Teil dieser Iteration.

## Iteration 55 — scan-inbound-paths-without-duplicate-delivery-guard — done — 2026-08-23 05:54
- commit: 948b0db4
- gebaut: reiner Scan, kein Code geändert. Vollständige Liste aller Eingangspfade
  geprüft, die ein Fremdsystem/Scheduler/Poller auslösen kann, gegen die Frage
  "was passiert bei Doppelzustellung/überlappendem Lauf". Ein Explore-Subagent
  hat die HTTP-Webhooks aus `internal/gateway/` und die Scheduler/Poller aus
  `cmd/*/main.go` (automation, berichte, work, document, gateway) durchsucht;
  ich habe danach selbst drei Lücken in seiner Abdeckung nachgezogen, die die
  Backlog-Anker explizit offen liessen bzw. die er nicht angefasst hatte:
  (1) `route_automation.go:683` `HandleTriggerWebhook` — die im Backlog
  benannte offene Frage "Verhalten OHNE Idempotency-Key" ist im Code bereits
  gelöst: `internal/automation/workflow/webhook.go:198-206` fällt ohne Header
  auf den SHA-256-Hash des Bodys zurück (`dedupeKey = bodyHash`), reserviert
  über den gemeinsamen `internal/idempotency`-Store mit `auto.OwnerID` als
  `user_id` (löst die im Backlog genannte `user_id NOT NULL`-Einschränkung
  über den Automations-Owner statt eines Systemnutzers) und ist mit
  `TestTriggerWebhook_DuplicateViaBodyHashFallback` sowie
  `TestTriggerWebhook_IdempotencyConflict` abgedeckt. NICHT-FUND, keine offene
  Frage mehr.
  (2) Bexio-Poller (`internal/biz/bexio/scheduler.go`) und Lexware-Scheduler
  (`internal/biz/lexware/scheduler.go`), die als bekannter Anker im Backlog
  standen, aber ohne vorab beantwortetes Ergebnis (anders als die übrigen
  Anker): je Tenant genau eine Goroutine mit `for { select {...} }` über
  mehrere `time.Ticker`s — dieselbe Goroutine kann `SyncContacts`/
  `PollPayments`/`PullInvoicesWithConfig` nie überlappend ausführen, weil
  `select` sequenziell bearbeitet und ein `time.Ticker`-Channel nur ein
  ungelesenes Tick puffert statt zu stauen. Kein prozessinterner Overlap
  möglich. Cross-Replica-Overlap (zwei Instanzen von `cmd/biz`) ist im
  aktuellen Ein-Server-pro-Kunde-Deployment-Modell kein reales Risiko (keine
  horizontale Skalierung dieses Service). NICHT-FUND.
  (3) `gdpr.RetentionScheduler` (`internal/security/gdpr/retention_scheduler.go`,
  `cmd/auth/main.go:143`) — vom Explore-Agenten nicht erfasst, da nur
  `cmd/document` und `cmd/work` per Grep-Muster `time.NewTicker`/`for {`
  auffielen. Bereits per `pg_try_advisory_lock` mit dediziertem Connection-
  Acquire/Release (derselbe Fix, der `IdempotencyCleanupWorker` in CI-Lauf
  32569420247 fehlte) gegen Mehrfachausführung über Replicas gehärtet.
  NICHT-FUND.
  Vom Explore-Agenten geprüft und mit Begründung als NICHT-FUND übernommen:
  Teams/Slack-Webhook (`route_integration.go:98-100/547`, alle Schreibpfade
  State-Overwrite oder Delete, kein Zeilen-Duplikat), Automations-
  Poller (`internal/automation/trigger/poller.go`, CAS-Claim), Berichte-
  Scheduler (`internal/berichte/scheduler/scheduler.go`, CAS-Claim gegen
  doppelten Mailversand), Work-Recording-Cleanup, Meeting-Auto-Close-Sweeper
  (Terminalzustands-Check, eigener Test gegen Doppel-Aufruf), WOPI-Lock-
  Cleanup, Gateway-Idempotency-Cleanup-Worker (Advisory-Lock). Bereits vorab
  im Backlog-Draft als NICHT-FUND belegt (nicht erneut geprüft, nur
  referenziert): Lexware-Webhook, LiveKit-Webhook, Formulare-Webhooks,
  Kontoauszugsimport, DATEV-Upload.
  Bewusst ausgeklammert (Begründung: kein Fremdsystem-Trigger, keine
  Signatur-/JWT-Prüfung gegen ein externes System): Booking-, Berichte-Share-,
  Document-Share-, Helpdesk-CSAT-, Wiki-Share- und Guest-Session-Routen unter
  `RegisterPublicRoutes` — das sind menschlich ausgelöste Formulare/Links,
  keine Webhook-/Scheduler-Eingänge im Sinne der Scope-Definition.
- gate: n.a. (reiner Scan, kein Code/keine Tests geändert, `go build`/`go
  test` nicht einschlägig)
- coverage: n.a. (kein Coverage-Ziel, Scan-Unit)
- mutations-probe: n.a. (kein Verhalten geändert)
- verify vorgaenger: sauber. `a896d604` (Iteration 54) geprüft: `git show
  --stat` zeigt nur zwei Testdateien plus BACKLOG.yml/JOURNAL.md, kein neuer
  Handler/Service-Direktaufruf, kein Stub/TODO, kein `.proto`-Change, keine
  Migration, kein neuer/ersetzter RequirePermission-Guard, keine neue
  Tabelle, keine neue Route, kein Wire-Shape-Wechsel — keine der acht
  Fehlerklassen einschlägig.
- neue-units: keine. Alle geprüften Pfade sind NICHT-FUND — entweder bereits
  durch CAS-Claim/Advisory-Lock/Terminalzustands-Check gehärtet, oder die
  Schreibvorgänge sind von Natur aus idempotent (Overwrite/Delete-by-
  Bedingung), oder (Automations-Webhook ohne Key) bereits per Body-Hash-
  Fallback gelöst und getestet.
- offen: keine.

## Iteration 56 — scan-gateway-openapi-response-code-drift-remaining — done — 2026-08-23 06:20
- commit: 46bccdd3
- gebaut: reiner Scan, kein Produktionscode geändert (Scan-Unit-Regel). Fortsetzung des in
  Lauf 10 Iteration 46 (Commit `b9889b04`) bewusst abgebrochenen Scans: tatsächlich
  geschriebene HTTP-Statuscodes gegen `openapi.yaml` diffed, mit Priorität auf den
  verbleibenden Geld-Routen (Vorgabe der Unit-Notes: "erst die verbliebenen Geld- und
  Compliance-Routen"). Ein Explore-Subagent hat pro Datei alle Handler-Statuscodes
  (mit Zeilennummer) gegen die zugehörigen `responses:`-Blöcke in `openapi.yaml`
  (ebenfalls mit Zeilennummer) zusammengetragen; ich habe den schwerwiegendsten Fund
  (DATEV-OAuth-Callback) direkt am Code gegengeprüft, bevor ich Units daraus abgeleitet
  habe.
  TIEF GEPRÜFT (Code gegen Spec-Zeile diffed, alle Handler der Datei):
    route_biz_bank_accounts.go, route_biz_bank_transactions.go, route_biz_billing.go,
    route_biz_expenses.go, route_biz_invoices.go, route_biz_open_items.go,
    route_biz_recurring.go, route_biz_time_entries.go, route_biz_transactions.go,
    route_biz_document_chains.go, route_datev_upload.go — elf Dateien, alle in Block A/B
    von Lauf 11 nicht angefasst und in Lauf 10 nur oberflächlich gegrept bzw. gar nicht
    geprüft.
  VIER FUNDE, VIER NEUE UNITS:
    1. `route_datev_upload.go:168` — echter Verhaltens-Bug, keine reine Doku-Lücke: die
       Spec garantiert für den OAuth-Callback explizit "Always responds with a 302
       redirect... there is no JSON error body on failure" (`openapi.yaml:29437`), aber
       genau eine von sechs Fehlerstellen im selben Handler bricht das mit
       `respondServiceUnavailable` (503-JSON statt Redirect) — alle anderen fünf
       Fehlerstellen im selben Handler nutzen korrekt `redirectDatevError`. Da die Route
       public ist und DATEV den Browser direkt hierher leitet, sieht der Nutzer im
       Fehlerfall einen nackten JSON-Body statt der erwarteten Weiterleitung.
       → fix-datev-oauth-callback-503-breaks-redirect-contract
    2. PUT `/finance/invoices/{id}` dokumentiert 410, POST
       `/finance/recurring/{id}/generate` dokumentiert 412 — beide strukturell
       unerreichbar, weil `grpcStatusToHTTP` (helpers.go:47-80) `FailedPrecondition`
       fest auf 409 mappt. Kein Codepfad kann 410/412 erzeugen; ein spec-treuer Client
       würde den tatsächlichen 409 nie behandeln. Braucht eine Entscheidung (Spec
       korrigieren vs. Mapping erweitern), deshalb eigene Unit mit offener Frage, nicht
       mechanisch gelöst.
       → fix-invoice-recurring-grpc-status-mapping-doc-mismatch
    3. Drei Routen schreiben literal Fehlercodes, die die Spec für sie gar nicht kennt:
       GET /finance/open-items (3× 500 bei Marshal-Fehlern, Spec nur 200), GET
       /finance/invoices (2× 400 + 1× 500, Spec nur 200), GET
       /finance/datev/oauth/authorize (2× 500 nur in Freitext-Beschreibung erwähnt, nicht
       im formalen responses-Block). Gebündelt nach Lauf-10-Präzedens: gleiche Ursache
       (Handler verhält sich schon so, Spec bildet es nicht ab), gleicher Fix
       (responses ergänzen).
       → fix-gateway-finance-list-routes-missing-error-response-docs
    4. Systemisches Muster: route_biz_billing.go (23 Handler) und
       route_biz_recurring.go (7 Handler) schreiben praktisch überall literal 401/503,
       dokumentieren das in der Spec aber fast durchgängig nicht — im Gegensatz zu den
       übrigen neun geprüften Dateien, die das überwiegend korrekt tun. Grösste der vier
       Units (~30 Routen), eigene Unit wegen des Umfangs.
       → fix-gateway-billing-recurring-routes-missing-401-503-docs
  NICHT-FUNDE, explizit geprüft und verworfen:
    - Acht Handler (bank-accounts PATCH/POST-connect/DELETE, bank-transactions
      reject-match, transactions DELETE, expenses approve/reject) validieren die Pfad-`id`
      nicht lokal (kein validateUUIDParam) — das dokumentierte 400 "Invalid id" ist dort
      nur über die gRPC-Fehlerkette erreichbar, nie literal im Handler. Kein Doku-Drift
      (die Spec-Zeile ist über den grpc-Pfad erreichbar), also kein Fund — nur ein
      Code-Stil-Unterschied zwischen den Dateien.
    - route_biz_time_entries.go GET /finance/time-entries prüft 401 nicht lokal im
      Handler-Body (TenantId wird nicht mal an den Work-gRPC-Call übergeben) — 401/403
      kämen ausschliesslich aus der Middleware. Kein Statuscode-Drift gegen die Spec
      (200/401/403/503 stimmen alle), nur auffällig im Vergleich zu den Nachbardateien;
      keine eigene Unit, da hier keine Diskrepanz zur Spec vorliegt.
    - Alle elf geprüften Dateien haben eine passende Pfad-Definition in openapi.yaml —
      keine fehlende Spec-Zeile (anders als Lauf 10 Iteration 46, wo zwei Routen ganz
      ohne Statuscodes dastanden).
- gate: n.a. (Scan-Unit, kein Produktionscode/Migration/Test angefasst — die vier neuen
  Units tragen ihr eigenes go test/swagger-cli validate als done_when)
- coverage: n.a. (Scan-Unit ohne Coverage-Ziel)
- mutations-probe: n.a. (Scan-Unit, kein neuer/geänderter Testfall)
- verify vorgaenger: sauber. `948b0db4` (Iteration 55, scan-inbound-paths-without-
  duplicate-delivery-guard) geprüft: `git show --stat` zeigt ausschliesslich BACKLOG.yml
  und JOURNAL.md, kein Produktionscode — keine der acht Fehlerklassen einschlägig.
- neue-units: fix-datev-oauth-callback-503-breaks-redirect-contract,
  fix-invoice-recurring-grpc-status-mapping-doc-mismatch,
  fix-gateway-finance-list-routes-missing-error-response-docs,
  fix-gateway-billing-recurring-routes-missing-401-503-docs
- offen:
  - Der Scan ist wieder bewusst abgebrochen, bevor er in die Breite ging: von den nach
    Lauf 10 Iteration 46 verbliebenen ~52 route_*.go sind mit dieser Iteration elf weitere
    (die Geld-Kern-Dateien) tief geprüft. NICHT geprüft bleiben u. a. route_security.go
    (1016 Zeilen, laut Backlog-Kopf nur 10/31 Handler getestet — nächster Kandidat mit
    höchster Priorität, da sicherheitsrelevant), die HR-Routen (route_hr*.go, mehrere
    Compliance-Bezug), route_settings.go (Rest ausserhalb der bereits geprüften
    module-grant-Route), sowie alle übrigen Nicht-Geld-Routen (CRM, Work, Chat, Video,
    Wiki, Booking, Fuhrpark/Vermietung/Einkauf/Produktion/Rapporte/Schichten,
    Integrationen). Ein künftiger Lauf kann daraus eine dritte Fortsetzungs-Unit ableiten,
    analog zu dieser hier — hier nicht selbst angelegt, da die Notes dieser Unit keine
    Nachfolge-Unit verlangen, sondern nur "wenn Zeit reicht" in die Breite gehen.

## Iteration 57 — scan-retention-mapping-remaining-services — done — 2026-08-23 06:12
- commit: d5daa61d
- gebaut: reiner Scan, kein Produktionscode geändert (Scan-Unit-Regel). Ein Explore-
  Subagent hat (1) den C4-Scan aus Lauf 10 (Iteration 43, `scan-personal-data-tables-
  without-retention-mapping`) und dessen Grundgesamtheit zitiert, (2) alle 24
  `backend/cmd/*`-Services gegen ihre Domain-Pakete und Migrationen auf personenbezogene
  Spalten geprüft, (3) `internal/security/gdpr/retention.go` (RetentionHandler-Interface,
  Registry-Pattern) gelesen, (4) gezielt nach `advisory_protocols` gesucht (Migration,
  Code, Doku) und (5) die Differenzmenge zwischen registrierten Handlern und real
  personenbezogenen Tabellen gebildet.
  KERNBEFUND 1 — advisory_protocols ist NEU BELEGT, Lauf 10 lag falsch: Iteration 43
  hatte notiert "keine Migration gefunden, die das explizit belegt". Der Kopfkommentar
  von `backend/migrations/000137_advisory_protocols.up.sql:1-6` zitiert aber wörtlich
  MiFID II / § 64 WpHG / § 16-18 FinVermV / § 61 VVG (IDD) und eine 10-Jahres-Frist nach
  FinVermV § 18a. Einordnung damit: Aufbewahrungspflicht (10 Jahre nach Aushändigung),
  nicht Löschpflicht — und die Frist ist am Code als Prosakommentar dokumentiert, aber
  NICHT technisch durchgesetzt (keine Ablauf-Spalte, kein Retention-Handler, nur die
  RESTRICT-FK auf contacts, die eine Kontakt-Löschung mit offenem Protokoll blockiert).
  Damit ist der Punkt "advisory_protocols entweder mit Quelle entschieden oder offene
  Frage in BACKLOG-NEXT.yml" erfüllt: entschieden, mit Quelle — trotzdem als Kontext in
  die neue Sammel-Unit `decide-retention-policy-for-unmapped-personal-data-domains`
  aufgenommen, weil die Frage "reicht die RESTRICT-FK oder braucht es eine aktive
  Lösch-nach-10-Jahren-Funktion gegen einen künftigen generischen Bereinigungs-Job" offen
  bleibt.
  KERNBEFUND 2 — sechs Domänen mit belegtem Personenbezug ohne Retention-Handler, davon
  eine (invitations) mechanisch und risikoarm genug für eine direkte Unit, fünf mit
  echtem Klärungsbedarf (Rechtsfragen zu Aufbewahrungsfristen, eine Architekturfrage):
    - `invitations` (auth): email/first_name/last_name, DSAR existiert (Lauf 10 Iteration
      62), kein Retention-Handler. Migration 000004 legt bereits einen Index "for cleanup
      of expired invitations" an (Zeile 20) — die Aufräumung war von Anfang an geplant,
      nie gebaut. Kein Zielkonflikt mit einer Aufbewahrungspflicht gefunden.
      → feat-retention-worker-handler-auth-invitations (BACKLOG.yml, todo)
    - HR-Personaldaten (biz/hr): hr_employee_profiles/leave_requests/leave_balances/
      employee_documents/profile_change_requests. DSAR existiert (Lauf 10 Iteration 59).
      Deutsches Arbeitsrecht kennt je nach Dokumenttyp unterschiedliche Fristen (bis 10
      Jahre für Lohnunterlagen nach § 147 AO/§ 257 HGB) — pauschale Frist wäre falsch.
    - Fuhrpark: driver_licenses/vehicles/vehicle_bookings/trip_logs. DSAR existiert (Lauf
      10 Iteration 61). Offen: Aufbewahrung wegen Unfall-/Versicherungsansprüchen vs.
      fristlose Betriebsdaten.
    - E-Mail (email_messages/email_accounts/email_contact_links): DSAR-Modul existiert
      laut Recherche bereits. Offen: fällt Geschäftskorrespondenz unter § 257 HGB (6 J.)?
    - Verträge (contracts/contract_parties): DSAR existiert. Offen: § 257 HGB oder länger
      je Vertragstyp — Löschung vor Verjährungsablauf wäre riskant.
    - `guest_sessions` (chat): email/display_name/ip_address anonymer Chat-Gäste. War in
      Lauf 10 (Iteration 40) bewusst OHNE DSAR-Modul dokumentiert (fehlende dritte
      Subjekt-Matching-Quelle) — dieselbe Hürde gilt für Retention. Bisher in KEINER
      Backlog-Datei erfasst; hier zum ersten Mal festgehalten.
  Alle sechs zusammen in einer Sammel-Unit gebündelt (nicht sechs Einzel-Stubs mit
  erratener Frist), weil jede eine echte Entscheidung von Luke braucht, bevor eine
  spitze Build-Unit sinnvoll ist:
      → decide-retention-policy-for-unmapped-personal-data-domains (BACKLOG-NEXT.yml,
        blocked, blocked_reason: Produkt-/Rechtsentscheidung)
  NEBENBEFUND (informativ, keine Unit): `consent_records` (ip_address) sollte vermutlich
  NICHT gelöscht werden — die Aufzeichnung ist der Nachweis der Einwilligung nach Art. 7
  Abs. 1 DSGVO, also das Gegenteil einer Löschpflicht. `public_bookings`/`booking_pages`
  wurden in keinem bisherigen Scan geprüft und sind unklassifiziert — beides in der
  Sammel-Unit als Kontext vermerkt, keine eigene Unit mangels tieferer Prüfung.
  NICHT-FUND: die acht bereits registrierten Handler (contacts, dialer_call_sessions,
  messages, tickets, form_submissions, tasks, calendar_events, notifications) und die in
  Lauf 10 bereits begründeten Nicht-Zuordnungen (time_entries/hr_work_time_entries
  arbeitsrechtlich, finance_invoices GoBD, gobd_* WORM, companies/company_settings B2B)
  bleiben unverändert bestehen — kein neuer Zweifel an diesen Einordnungen.
- gate: n.a. (Scan-Unit, kein Produktionscode/Migration/Test angefasst — die neue Unit
  `feat-retention-worker-handler-auth-invitations` trägt ihr eigenes go test als
  done_when, die Sammel-Unit baut nichts)
- coverage: n.a. (Scan-Unit ohne Coverage-Ziel)
- mutations-probe: n.a. (Scan-Unit, kein neuer/geänderter Testfall)
- verify vorgaenger: sauber. `46bccdd3` (Iteration 56, scan-gateway-openapi-response-
  code-drift-remaining) geprüft: `git show --stat` zeigt ausschliesslich BACKLOG.yml und
  JOURNAL.md, kein Produktionscode — keine der acht Fehlerklassen einschlägig.
- neue-units: feat-retention-worker-handler-auth-invitations (BACKLOG.yml, todo)
- offen:
  - decide-retention-policy-for-unmapped-personal-data-domains steht in BACKLOG-NEXT.yml
    (nicht BACKLOG.yml, der Treiber liest die Datei nicht) — braucht Lukes Entscheidung
    zu sechs Domänen, teils mit Rechtsberatung (HR-Personalakte, Vertragsunterlagen),
    bevor daraus einzelne Build-Units werden.
  - advisory_protocols: Widerspruch zu Lauf 10 Iteration 43 im Journal dokumentiert
    (dort "nicht belegt", jetzt mit Migrationskommentar als Quelle belegt) — falls ein
    künftiger Lauf den alten Lauf-10-Eintrag liest, sollte er diesen hier als aktuelleren
    Stand nehmen.
  - Die strukturelle Migrations-Suche (295 CREATE TABLE, Grep nach PII-Feldnamen) fand
    71 Tabellen mit Substring-Treffer, davon rund 30 echte Personenbezugsträger nach
    manueller Prüfung — eine erschöpfende Tabelle-für-Tabelle-Bewertung über alle
    FK-Transitivitäten (139 FKs auf users, 154 auf contacts/users laut Lauf 10) wurde
    nicht geleistet und würde den Rahmen eines einzelnen Scans sprengen.

## Iteration 58 — scan-money-rounding-and-tax-call-sites — done — 2026-08-23 06:47
- commit: 6bc48470
- gebaut: reiner Scan, kein Produktionscode geändert (Scan-Unit-Regel). Ein Explore-
  Subagent hat alle fünf Aufrufer von `internal/biz/tax` (invoice, creditnote, quote,
  recurring, server/biz_grpc.go) sowie alle unabhängigen `.Round(`/`.Div(`/`.Mul(`-Stellen
  auf `decimal.Decimal` in `internal/` erfasst (16 Fundorte, davon 3 Testdateien) und je
  Stelle die drei Fragen (Geld oder anderes? gleiche Rundungsebene wie biz/tax? Aussenwirkung?)
  beantwortet.
  KERNBEFUND — systemische ungerundete `LineTotal`-Speicherung: alle vier Service-Aufrufer
  von `tax.Calculate` (invoice/service.go:151+473, creditnote/service.go:100,
  quote/service.go:121+305, recurring/service.go:482) rechnen `LineTotal` nach dem
  Tax-Aufruf ein zweites Mal selbst — `Quantity.Mul(UnitPrice)`, UNGERUNDET — statt den
  bereits gerundeten Zeilenwert zu übernehmen, den `biz/tax/calculator.go:73`
  (`item.Quantity.Mul(item.UnitPrice).Round(2)`) intern für Subtotal/TaxByRate/GrossTotal
  verwendet. Bei glatten Mengen unsichtbar, bei Bruchmengen (typischerweise Stunden aus der
  Zeiterfassung) entstehen 3-4 Nachkommastellen. Konkreter, bereits real existierender
  Auslöser: `server/biz_grpc.go:2058-2059` — Stunden sauber gerundet, `hours.Mul(hourlyRate)`
  danach nicht. Zwei bestätigte Aussenwirkungen: (1) `datev/exporter.go:259-267` vertraut dem
  gespeicherten `LineTotal` direkt und rundet nur den finalen Bruttobetrag — der einzige
  Steuerberater-Export im Code, der das Netto nicht vor der Steuerableitung rundet, anders als
  `gobd_rows.go` und `generator_doc.go`; (2) `biz_grpc.go:1757` sendet
  `item.LineTotal.String()` (beliebige Präzision) unverändert auf den gRPC-Draht für die
  Desktop-Rechnungsansicht. `bexio/field_mapper.go:303` importiert ebenfalls ungerundet und
  tritt über `toLineItems` in denselben Erzeugungspfad ein — kein separater Fund, gleicher
  Root Cause.
  EINGEORDNET, KEIN FUND: `einvoice/generator_doc.go`/`parser.go` runden absichtlich auf
  USt-Gruppen-Ebene (EN 16931/BR-CO-17), bereits in `fix-tax-rounding-divergence-across-
  implementations` behandelt und per Roundtrip-Test belegt — nicht Teil dieses Scans.
  `datev/exporter.go`s Brutto-Rundung selbst ist beabsichtigt (kein Gruppentotal zum Runden
  vorhanden), nur das ungerundete Netto als Eingabe ist der Fund. `expense/service.go:283`,
  `dashboard/*` (Konversionsrate, Ø-Dealgrösse, Forecast), `berichte/executor.go:530-536`
  (Prozent-Änderung) und `route_biz_document_chains.go:73-90` sind Dashboard-/Anzeige-Metriken
  bzw. bereits in Iteration 54 als wirkungslos belegt (Quellspalten NUMERIC(15,2),
  Bestätigung erneut geprüft: hält). `pdf/templates.go:154` rechnet nur für die
  Anzeige (`formatEUR`/`StringFixed(2)`), der Wert fliesst nirgends zurück in Speicherung
  oder Export. `work/timeentry`/`work_grpc.go`-Stundenrundungen sind `float64`-Dauer, kein
  Geld, kategorial ausserhalb des Scan-Ziels.
  BEOBACHTET, BEWUSST KEINE EIGENE UNIT: `biz_grpc_einvoice.go:175` sendet `StringFixed(4)`
  statt der sonst durchgängigen 2-Dezimalstellen-Konvention im eInvoice-Import-Vorschaupfad —
  Präzisions-Inkonsistenz, aber kein nachgewiesenes Fehlergebnis (nur mehr Nachkommastellen
  in einer Vorschau), deshalb im Journal vermerkt statt als Fix-Unit angelegt.
  B21/C16 (im Backlog-Scope-Text referenzierte Meldestellen): keine neuen Einträge im
  JOURNAL.md seit Anlage dieser Unit gefunden — nichts einzuarbeiten ausser dem bereits in den
  Notes der Unit dokumentierten, in Iteration 54 beantworteten `route_biz_document_chains.go`-
  Fund (erneut bestätigt, s.o.).
- gate: n.a. (Scan-Unit, kein Produktionscode/Migration/Test angefasst — die neue Unit
  `fix-unrounded-line-total-in-invoice-service-callers` trägt ihr eigenes go test als
  done_when, die Sammel-Unit baut nichts)
- coverage: n.a. (Scan-Unit ohne Coverage-Ziel)
- mutations-probe: n.a. (Scan-Unit, kein neuer/geänderter Testfall)
- verify vorgaenger: sauber. `d5daa61d` (Iteration 57, scan-retention-mapping-remaining-
  services) geprüft: `git show --stat` zeigt ausschliesslich BACKLOG-NEXT.yml, BACKLOG.yml und
  JOURNAL.md, kein Produktionscode — keine der acht Fehlerklassen einschlägig.
- neue-units: fix-unrounded-line-total-in-invoice-service-callers (BACKLOG.yml, todo)
- offen:
  - fix-unrounded-line-total-in-invoice-service-callers braucht zuerst einen Blick in
    `tax.Calculate`s Rückgabetyp: gibt die Funktion das bereits gerundete Zeilen-LineTotal
    zurück (dann Wiederverwendung statt Neuberechnung, der eigentliche Root-Cause-Fix), oder
    nicht (dann vier explizite `.Round(2)`-Ergänzungen).
  - `biz_grpc_einvoice.go:175` (StringFixed(4) statt 2 Dezimalstellen) ist notiert, aber nicht
    als Unit angelegt — falls ein künftiger Scan dort ein echtes Fehlergebnis nachweist, dort
    ansetzen.

## Iteration 59 — scan-build-tag-excluded-money-tests — done — 2026-08-23 06:32
- commit: 94509d58
- gebaut: reiner Scan, kein Produktionscode/keine CI-Datei geändert. Ein Explore-Subagent hat
  alle Go-Testdateien mit ausschliessendem Build-Tag unter `backend/` erfasst (20 Dateien, alle
  im neuen `//go:build`-Stil, kein `// +build` mehr im Repo) und jede gegen `ci.yml`,
  `nightly.yml` und `Makefile` abgeglichen.
  BEFUND: 11 Dateien mit `//go:build integration` (creditnote ×3, hr/absence, hr/employee,
  hr/leave, invoice ×3, quote ×2) — alle liegen unter den vier Paketbäumen, die der Job
  "Finance & HR Integration Tests" (`ci.yml:139-178`, `-tags=integration`) tatsächlich abdeckt
  (`./internal/biz/{invoice,quote,creditnote,hr}/...`). Dieser Job ist laut eigenem
  Inline-Kommentar bewusst aus dem nicht-blockierenden nightly.yml nach ci.yml verschoben
  worden, "damit ... jeden Merge gaten" — er läuft auf `push`+`pull_request` gegen main ohne
  `needs:`, also PR-blockierend (harte Grenze: ob er als GitHub-Branch-Protection-Required-Check
  eingetragen ist, steht nicht im Workflow-YAML und wurde nicht geprüft, siehe offen:).
  7 Dateien mit `//go:build e2e` (`test/e2e/*.go`) laufen im ebenfalls PR-blockierenden Job
  "E2E Tests" (`ci.yml`, `needs: [lint, test]`). 2 Dateien mit `//go:build smoke`
  (`test/smoke/*.go`) laufen NUR in `nightly.yml` (`schedule`+`workflow_dispatch`, explizit
  nicht PR-blockierend) — betreffen aber keine Geld-/Finanzpfade, sondern generische
  Health-Checks, deshalb kein Fund im Sinne dieser Unit.
  KEIN toter Test gefunden: alle 20 getaggten Dateien laufen in genau einem der drei Jobs, kein
  Paketpfad fällt aus der Abdeckung der vier Job-Paketbäume heraus, kein verwaister Tag-Wert.
  Die ursprüngliche Vorbereitungs-Prämisse aus dem Backlog-Kopf ("elf Testdateien ... laufen
  damit weder im lokalen Loop-Gate noch im Coverage-Job — nur im separaten CI-Job") ist damit
  präzisiert, nicht widerlegt: die Zahl 11 (integration-getaggt) stimmt exakt, und sie laufen
  tatsächlich NICHT im lokalen Loop-Gate und NICHT im Coverage-Job (der Job sammelt kein
  `-coverprofile`) — sie laufen aber sehr wohl PR-blockierend in ci.yml. Das bestätigt Regel 4
  im Backlog-Kopf (neue DB-Tests ungetaggt schreiben, damit sie im Loop-Gate UND im
  Coverage-Job zählen) als weiterhin richtig, ändert aber nichts an einer bereits laufenden
  Absicherung.
  Kein Makefile-Target ruft `-tags=integration` auf (kein Kurzbefehl für lokale Entwicklung),
  und keine Workflow-Datei ruft ein Makefile-Target auf — alle Jobs rufen `go test -tags=...`
  direkt.
- gate: n.a. (Scan-Unit, kein Produktionscode/Migration/Test/CI-Datei angefasst)
- coverage: n.a. (Scan-Unit ohne Coverage-Ziel)
- mutations-probe: n.a. (Scan-Unit, kein neuer/geänderter Testfall)
- verify vorgaenger: sauber. `6bc48470` (Iteration 58, scan-money-rounding-and-tax-call-sites)
  geprüft: `git show --stat` zeigt ausschliesslich BACKLOG.yml und JOURNAL.md, kein
  Produktionscode — keine der acht Fehlerklassen einschlägig.
- neue-units: keine (kein toter Test gefunden, keine Zusage hängt an einem nicht laufenden
  Test — Empfehlung ist "kein Handlungsbedarf", siehe done_when-Vorgabe der Unit)
- offen:
  - Ob "Finance & HR Integration Tests" und "E2E Tests" als GitHub-Branch-Protection-Required-
    Checks eingetragen sind (statt nur faktisch bei jedem PR zu laufen), steht nicht im
    Workflow-YAML und wurde nicht geprüft — für die Frage dieser Unit (läuft der Test
    irgendwo PR-gatend) ausreichend, für eine vollständige Aussage zur Merge-Blockade nicht.
  - Kein Makefile-Kurzbefehl für `go test -tags=integration` existiert — lokale Entwickler
    müssen den vollen Befehl von Hand tippen. Kein Fund im Sinne dieser Unit (betrifft
    Entwickler-Ergonomie, nicht CI-Abdeckung), aber notiert falls das mal auffällt.

## Iteration 60 — scan-finance-mutations-without-idempotency-key — done — 2026-08-23 06:39
- commit: c6495588
- gebaut: reiner Scan, kein Produktionscode geändert. Zuerst den in Produktion geltenden
  `IdempotencyMode` belegt (nicht angenommen, wie die Unit-Notes verlangen):
  `cmd/gateway/main.go:199` setzt den Go-Default auf `WarnMode`, aber
  `deploy/docker/docker-compose.prod.yml:306` UND `deploy/docker/docker-compose.yml:1013`
  überschreiben ihn mit dem literalen Wert `IDEMPOTENCY_MODE: hard` (kein `${VAR:-default}`,
  also über `.env.production` nicht änderbar) — in jeder ausgelieferten Umgebung gilt HardMode.
  Damit lehnt die globale Middleware (`cmd/gateway/main.go:204`, montiert als
  `authWithIdempotency` für alle Registrar-Routen) JEDE mutierende Anfrage ohne
  `Idempotency-Key`-Header mit 400 ab, ausser den drei whitelisteten Auth-Pfaden
  (`/auth/login`, `/auth/refresh`, `/auth/2fa`) — vor jedem Handler, nicht danach.
  Vollständige Grundgesamtheit gegen `route_biz.go` (zentrale Registrierung aller
  `/api/v1/finance/*`-Routen) und `route_datev_upload.go` geprüft: Quotes (create/update/delete/
  send/accept/reject/convert), Invoices (create/import/update/send/mark-paid/cancel/erechnung/
  lock/payments), Recurring (create/update/delete/pause/resume/generate), Credit-Notes (create/
  send), Payments (delete), Dunning (detect/send/escalate/config/status/notice), Bank-Statements
  (import), Bank-Transactions (match/reject-match/ignore), Bank-Accounts (create/patch/connect/
  delete), Expenses (create/update/approve/reject/receipt/delete), Transactions (delete), Export
  (datev/gobd), GoBD-Archive (archive/from-invoice/annotations), Incoming-Invoices (status-patch),
  DATEV-Upload (upload/upload-beleg/config) — alle liegen unter `r.Route(...)` mit
  `r.Use(authMiddleware)`, wobei `authMiddleware` in jedem Fall der von `main.go` übergebene
  `authWithIdempotency` ist (verifiziert: BizRoutes und DatevUploadRoutes stehen in der
  `registrars`-Liste, `reg.RegisterRoutes(r, authWithIdempotency)`). ERGEBNIS: für keine dieser
  Routen kann ein fehlender Key durchrutschen — HardMode blockt strukturell vor dem Handler,
  nicht als Frage der einzelnen Route. Die beiden im Scope zitierten "harten"
  Handler-Eigenchecks (`route_hr.go:1621`, `route_schichten.go:790`, ausserhalb des
  Finanz-Scopes dieser Unit) sind damit erwiesenermassen redundant (die Middleware hätte schon
  vorher 400 geliefert) — keine Bugs, nur totes Duplikat, keine eigene Unit wert.
  DELETE-Routen (Payments, Transactions, Bank-Accounts, Quotes, Expenses) sind Nicht-Funde: sie
  laufen ebenfalls hinter HardMode und sind zusätzlich von Natur aus idempotent (zweites DELETE
  auf dieselbe ID trifft eine bereits gelöschte Zeile).
  ECHTER FUND: `api/openapi.yaml` behauptet an zwei Stellen (Zeile ~20 `info.description`,
  Zeile ~47787 `IdempotencyKeyRequired`-Response) wörtlich "production default is WarnMode" —
  das ist nachweislich falsch (s. o.). Der Code selbst weiss es richtig
  (`route_biz_billing_test.go:258`, `route_biz_billing.go:234`) — nur die nach aussen lesbare
  API-Spezifikation widerspricht der Realität. Diese Doku wurde in Iteration 15
  (`fix-idempotency-inflight-409-not-in-openapi`) genau mit dieser (unverifizierten) Annahme
  geschrieben. Neue Unit `fix-openapi-idempotency-doc-wrong-production-default` angelegt.
- gate: n.a. (Scan-Unit, kein Produktionscode/keine Migration/kein Test angefasst)
- coverage: n.a. (Scan-Unit ohne Coverage-Ziel)
- mutations-probe: n.a. (Scan-Unit, kein neuer/geänderter Testfall)
- verify vorgaenger: sauber. `94509d58` (Iteration 59, scan-build-tag-excluded-money-tests)
  geprüft: `git show --stat` zeigt ausschliesslich BACKLOG.yml und JOURNAL.md, kein
  Produktionscode — keine der acht Fehlerklassen einschlägig.
- neue-units: fix-openapi-idempotency-doc-wrong-production-default (BACKLOG.yml, todo)
- offen:
  - `route_caldav.go` registriert `/api/v1/caldav/*` + `/api/v1/admin/caldav/*` (App-Passwörter
    erstellen/widerrufen, POST/PUT/DELETE) mit einer eigenen bare `authMiddleware` OHNE
    Idempotency-Schutz — bereits in Iteration 15 als unbelegte Beobachtung notiert, hier nur
    bestätigt gesehen, nicht erneut geprüft (kein Finanzpfad, ausserhalb des Scopes dieser Unit).
  - Die Feststellung "HardMode gilt in jeder ausgelieferten Umgebung" stützt sich auf die beiden
    Compose-Dateien im Repo; ob eine mögliche zukünftige dritte Compose-Variante (Staging o. ä.)
    denselben Wert setzt, wurde nicht geprüft, da keine solche Datei existiert.

## Iteration 61 — scan-hardcoded-eur-vs-currency-column — done — 2026-08-23 06:41
- commit: (siehe unten)
- gebaut: reiner Scan, kein Produktionscode geändert. Vollständige Grundgesamtheit erhoben
  (Explore-Subagent, gegen Datei:Zeile geprüft, nicht geraten):
  WÄHRUNGSSPALTEN (15 Migrationen): deals, purchase_orders, finance_incoming_invoices,
  inventory_items (nullable!), supplier_catalog_items, framework_contracts,
  framework_contract_calls, finance_invoices, finance_credit_notes, finance_quotes,
  company_settings.default_currency, finance_recurring_invoices, finance_bank_statements,
  finance_bank_transactions, finance_bank_accounts — alle NOT NULL DEFAULT 'EUR' ausser
  inventory_items. NEGATIVBEFUND: `finance_payments`
  (`000045_create_finance_tables.up.sql:129-146`) hat bis heute KEINE currency-Spalte.
  EUR-LITERALE: durchweg Fallback-Werte (models.DefaultCurrency, Parser-Fallback bei leerem
  Feld, Banking-Matcher-Fallback, Deal/Bankkonto/Einkauf-Defaults) — kein Fund für sich.
  AGGREGATE GEPRÜFT: DATEV-Export (pro Zeile, kein Cross-Doc-SUM — sauber), Offene-Posten
  (`postgres_open_items.go:176-183` SUM...GROUP BY currency, bucket_index — sauber, currency-
  aware gebaut), Banking-Matcher (`matcher.go:118-127` vergleicht Currency explizit vor dem
  Settlement — sauber). DREI ECHTE FUNDE: (1) GetDashboardMetrics
  (`dashboard/postgres_repository.go:34-38,86`) summiert/mittelt gross_total über ALLE
  Rechnungen/Angebote eines Tenants OHNE Currency-Filter. (2) kpiSnapshotQuery/kpiSeriesQuery
  (`berichte/downstream/kpi_postgres.go:53-65,126-138`) — AKTIV verdrahtet
  (`cmd/berichte/main.go:58-60`) — summiert finance_invoices.gross_total UND deals.value ohne
  Currency-Filter, obwohl deals.currency existiert; Ausgabe trägt hardcoded `Unit: "EUR"`
  (`berichte/executor/executor.go:446,463`) — falsch beschriftete produktive Kennzahl, der
  konkreteste Fund. (3) GoBDExportRow (`dunning/service_gobd.go:91-104`) hat kein
  Currency-Feld, CSV (`buildGoBDCSV`) schreibt keine Währungsspalte — Fremdwährungszeilen im
  GoBD-Export sind vom Steuerberater nicht mehr unterscheidbar.
  HERKUNFT DER FREMDWÄHRUNG geklärt: einziger user-eingebbarer Weg ist
  `CreateRecurringInvoiceRequest.currency` (proto `biz.proto:1378`) — Ad-hoc-Erstellung
  `CreateInvoiceRequest` hat KEIN Currency-Feld, Quote/CreditNote nie user-eingebbar (Quote
  = immer Tenant-Default, CreditNote erbt von der Original-Rechnung). Blast-Radius der drei
  Funde ist also an "läuft mindestens eine Fremdwährungs-Recurring-Rechnung" gebunden, aber
  real, sobald das zutrifft.
  NEBENBEFUND ausserhalb des Scopes (gesperrtes Paket `internal/biz/bexio`, G3):
  `resolveKMUHubCurrency` (`bexio/field_mapper.go:347-353`) gibt immer EUR zurück, unabhängig
  von der echten Bexio-CurrencyID — bereits im Code-Kommentar als bekannte Lücke
  dokumentiert, NICHT neu, NICHT als Unit angelegt (Bexio ist in diesem Lauf gesperrt), nur
  in BACKLOG-NEXT.yml nachgetragen für den Fall einer künftigen Freigabe.
  Die drei genannten Tests geprüft: `TestGenerateZUGFeRDXML_Currency{FromInvoice,
  DefaultsToEUR}` bestätigen sauberen Durchgriff der Invoice-Währung in die ZUGFeRD-XML;
  `TestSummarizeOpenItems_AggregatesByCurrencyAndBucket` testet trotz seines Namens KEINE
  gemischten Währungen (beide Testrechnungen sind hart auf EUR gesetzt) — Namensversprechen
  nicht eingelöst, aber kein Fund, da die zugrundeliegende SQL-Query tatsächlich currency-aware
  ist (siehe Aggregate-Prüfung oben).
- gate: n.a. (Scan-Unit, kein Produktionscode/keine Migration/kein Test angefasst)
- coverage: n.a. (Scan-Unit ohne Coverage-Ziel)
- mutations-probe: n.a. (Scan-Unit, kein neuer/geänderter Testfall)
- verify vorgaenger: sauber. `c6495588` (Iteration 60, scan-finance-mutations-without-
  idempotency-key) geprüft: `git show --stat` zeigt ausschliesslich BACKLOG.yml und
  JOURNAL.md, kein Produktionscode — keine der acht Fehlerklassen einschlägig.
- neue-units: fix-dashboard-metrics-blind-currency-sum, fix-berichte-kpi-blind-currency-sum,
  fix-gobd-export-missing-currency-column (alle BACKLOG.yml, todo). Zusätzlich
  BACKLOG-NEXT.yml-Eintrag "FREMDWÄHRUNG" mit den drei Funden abgeglichen und um zwei echte
  Produktentscheidungen präzisiert (fehlende currency-Spalte auf finance_payments, Frage ob
  Ad-hoc-Rechnungen Fremdwährung anbieten sollen) plus dem Bexio-Nebenbefund.
- offen:
  - Ob `fix-dashboard-metrics-blind-currency-sum` und `fix-berichte-kpi-blind-currency-sum`
    dieselbe Filtern-vs-Gruppieren-Entscheidung treffen müssen, ist in beiden Units als
    Abhängigkeit vermerkt, aber nicht technisch erzwungen (keine `deps:`-Kette, weil beide
    unabhängig baubar sein sollen) — der bauende Iterationslauf muss die Schwester-Unit lesen.
  - Die zwei echten Produktentscheidungen (Payment-Currency-Spalte, Ad-hoc-Invoice-Currency)
    sind bewusst NICHT als Unit ausgeschrieben, weil beide eine Migration bzw. einen
    Proto-Vertragswechsel voraussetzen — Luke entscheidet, dann Unit.

## Iteration 62 — scan-gobd-immutability-beyond-belegarchiv — done — 2026-08-23 06:51
- commit: (siehe unten)
- gebaut: reiner Scan, kein Produktionscode geändert. Grundgesamtheit über alle Migrationen
  erhoben (Explore-Subagent, gegen Datei:Zeile geprüft, nicht geraten): finance_quotes,
  finance_invoices, finance_credit_notes, finance_payments, finance_dunning_records,
  finance_invoice_lines/finance_quote_lines/finance_credit_note_lines, gobd_documents/
  gobd_document_events, finance_incoming_invoices, finance_recurring_invoices/
  finance_recurring_runs, finance_bank_statements/finance_bank_transactions, finance_expenses,
  finance_bank_accounts, lexware_sync_log/datev_upload_log.
  GRANTS: vollständiger Scan aller `backend/migrations/*.sql` nach `REVOKE` bestätigt — außer
  `gobd_documents`/`gobd_document_events` (Migration 315, `REVOKE UPDATE, DELETE` für
  `kmuhub_app`) trägt KEINE weitere Geldtabelle ein DB-seitiges REVOKE; alle laufen unter dem
  schemaweiten GRANT aus `000121_create_app_role.up.sql:49,55-56`.
  FESTSCHREIBUNGSPRÜFUNGEN je Tabelle geprüft (Code, nicht DB): finance_invoices sauber über
  `Update`/`MarkPaid`/`Cancel` (`invoice/service.go:394-413,653-655,698-700`) und
  `LockInvoice` (`service_gobd.go:86-123`); finance_credit_notes sauber (kein aktiver
  Update-Schreibpfad außerhalb `Send`/`StornoInvoice`); finance_quotes, finance_recurring_
  invoices, finance_bank_*, finance_expenses jeweils mit Status-Guards, kein Bypass gefunden;
  finance_dunning_records bewusst ohne Sperre ("intentionally permissive to support admin
  overrides", `service_gobd.go:25-28`) — kein Fund, weil keine Prüfung existiert, die umgangen
  würde.
  ECHTER FUND: `payment.Service.Delete` (`internal/biz/payment/service.go:172-201`) prüft vor
  dem Löschen einer Zahlung nur `inv.Status == Cancelled` (Zeile 186-188) — die
  GoBD-§146-Sperre `inv.LockedAt != nil` fehlt, obwohl dieselbe Datei sie an zwei
  Schwesterstellen exakt für diesen Zweck wiederholt (`transitionToPaidInTx:233-238`,
  `revertPaidStatusInTx:270-273`, beide mit Kommentar "the lock check has to be repeated here
  rather than inherited"). Selbst gegengeprüft (Read auf service.go:160-279): Fund bestätigt.
  Zwei RPCs erreichen den ungeschützten Pfad, beide gegengeprüft:
  `DeletePayment` (`biz_grpc.go:931`) und `DeleteFinanceTransaction`
  (`biz_grpc_transactions.go:51`, Präfix `pay-`). Schaden: eine gesperrte, bezahlte Rechnung
  bleibt korrekt `paid` (der Statuswechsel selbst ist geschützt), aber ihr Zahlungsbeleg
  (Betrag, Datum, Referenz, Zahlungsart) wird trotzdem unwiderruflich gelöscht — ein
  festgeschriebener Buchungsnachweis verschwindet spurlos.
  NEBENBEFUND, KEIN FUND: `invoice.LinkTimeTracking` (`invoice/service.go:352-354`→
  `postgres_repository.go:472`) prüft `isInvoiceLocked` ebenfalls nicht, ist aber nur einmal
  aus `biz_grpc.go:2126` unmittelbar nach Invoice-Neuanlage erreichbar (Invoice ist dort
  zwingend `draft`, kann noch nicht gesperrt sein) — selbst gegengeprüft, kein zweiter
  Aufrufer gefunden, kein realer Bypass, keine Unit angelegt.
  RETENTION/GDPR: `internal/security/gdpr/dsar_search.go` liest aus Finanztabellen für
  Auskunftsanfragen, enthält aber kein DELETE/UPDATE gegen eine der genannten Tabellen — kein
  Retention-Worker mit Schreibzugriff auf finance_*/gobd_* gefunden.
- gate: n.a. (Scan-Unit, kein Produktionscode/keine Migration/kein Test angefasst)
- coverage: n.a. (Scan-Unit ohne Coverage-Ziel)
- mutations-probe: n.a. (Scan-Unit, kein neuer/geänderter Testfall)
- verify vorgaenger: sauber. `8e7f2ec5` (Iteration 61, scan-hardcoded-eur-vs-currency-column)
  geprüft: `git show --stat` zeigt ausschliesslich BACKLOG.yml, BACKLOG-NEXT.yml und
  JOURNAL.md, kein Produktionscode — keine der acht Fehlerklassen einschlägig.
- neue-units: fix-payment-delete-bypasses-invoice-lock (BACKLOG.yml, todo)
- offen:
  - Jeder REVOKE-Vorschlag auf weiteren Geldtabellen (falls Luke das für sinnvoll hält) muss
    laut Vorgabe dieser Unit einen Katalog-Scan gegen `ALTER DEFAULT PRIVILEGES`
    (`000121` Abschnitt 4) tragen, sonst ist er beim nächsten `CREATE TABLE` still wieder weg —
    diese Unit selbst schlägt aber kein REVOKE vor, weil kein Bypass an einer DB-Sperre hängt,
    sondern an einer fehlenden Code-Prüfung.
  - `fix-payment-delete-bypasses-invoice-lock` sollte klären, ob der Gateway-Handler
    `DeletePayment`/`DeleteFinanceTransaction` einen neuen Fehler (z.B. `ErrInvoiceLocked` aus
    dem invoice-Paket oder ein eigener) bereits auf einen HTTP-Code mappt oder ob dafür Code
    in `internal/gateway` nötig wird — im Scan nicht geprüft, gehört in die bauende Iteration.

## Iteration 63 — scan-finance-logs-error-leakage-and-pii — done — 2026-08-23 06:57
- commit: -
- gebaut: nichts am Produktionscode (Scan-Unit, aendert kein Verhalten). Fortsetzung von
  `scan-gateway-sql-error-leakage`/`scan-gateway-pii-in-logs` (Lauf 10), diesmal gezielt auf die
  Finanzflaeche: Bankdaten (IBAN/Kontoinhaber), Betraege in Verbindung mit Kundennamen,
  Steuernummern/USt-IdNr., DATEV-/Lexware-Zugangsdaten, SQL-Fehlermeldungen auf Geldpfaden.
  Geprueft (nach Wortstaemmen, nicht Einzelfeldnamen):
  - Alle `slog.*`-Aufrufe in `internal/biz/invoice`, `internal/biz/quote`,
    `internal/biz/creditnote`, `internal/biz/payment`, `internal/biz/dunning`,
    `internal/biz/recurring`, `internal/biz/banking`, `internal/biz/datev` (oauth.go,
    upload_service.go, uploader.go, belegbilder.go), `internal/biz/lexware`, `internal/biz/bexio`
    — durchweg IDs (invoice_id/tenant_id/quote_id/payment_id/dunning_id), Status, Zaehlwerte und
    Betraege OHNE begleitenden Kunden-/Personennamen. Kein Fund nach dem Muster "Betrag +
    Kundenname im selben Log-Aufruf".
  - `iban`/`Iban`/`IBAN`/`account_number`/`bank_account`/`kontoinhaber` ueber `internal/biz/banking`
    und `internal/gateway/route_biz_bank_accounts*.go`/`route_biz_bank_transactions*.go`: IBAN wird
    kanonisch gespeichert und beim Rausgeben formatiert (`dachfmt.FormatIBAN`), Validierungsfehler
    sind bereits per Test abgesichert, dass die zurueckgewiesene IBAN NICHT in der Antwort landet
    (`route_biz_bank_accounts_gate_test.go:117-125`, Kommentarblock ab Zeile 30 dokumentiert das
    ausdruecklich als bereits geprueften Fall). Kein neuer Fund.
  - `tax_number`/`vat_id`/`steuernummer`/`ustid` ueber `internal/biz/*` per Grep auf slog-Aufrufe
    beschraenkt (nicht auf Modell-/SQL-Felder, die sind kein Log) — keine Fundstelle.
  - gRPC-Fehlerpfade der Finanzdomaene (`mapBizError` in `biz_grpc.go:2563`, `mapGobdArchiveError`,
    `mapEInvoiceError`, `mapDatevUploadError`, `bankingError`, `bankAccountError`, `expenseError`):
    alle folgen demselben Muster wie `mapBexioError`/`mapLexwareError` (Vorlage aus Lauf 10) —
    benannte Sentinel-Fehler geben ihre eigene, statische Meldung heraus, der `default`-Zweig
    loggt serverseitig und maskiert die Client-Antwort mit einer generischen Meldung. Keine
    Fundstelle, an der ein roher SQL-/pgconn-Fehlertext in eine Antwort durchrutscht.
  - `route_biz_*.go` (invoices, quotes, creditnotes, payments über biz_grpc, banking, expenses,
    bank_accounts, bank_transactions, transactions) auf `err.Error()` in der HTTP-Antwort: nur
    zwei Treffer, beide bereits in Lauf 10 als Nicht-Fund geprueft und identisch geblieben
    (`route_biz_banking.go:38`, `route_biz_einvoice.go:38` — Multipart-Parse-Fehler, reiner
    Client-Input, keine DB-Beruehrung).
  - Die drei Lauf-10-Funde bei Bexio/DATEV/Lexware (Success:false+ErrorMessage in Redirect/JSON)
    sind laut Archiv-Backlog (`archive/lauf-10/BACKLOG.yml:2921/2977/3009`) alle `status: done` —
    kein erneuter Fund an denselben Stellen.
  - Einzige Grenzfall-Fundstelle (bewusst NICHT als Fund gewertet): `internal/biz/datev/oauth.go:
    113-117` (`RefreshAccessToken`) und `:176-180` (`ExchangeCode`) loggen bei einem Non-200 von
    DATEVs eigenem OAuth-Token-Endpunkt den rohen Response-`body` auf Error-Level
    ("datev token refresh failed"/"datev code exchange failed"). Das ist die Antwort EINES
    Fremdsystems auf eine Anfrage, die wir selbst mit unseren eigenen client_id/client_secret
    gestellt haben — kein von einem Nutzer kontrollierter Input, keine Kundendaten, und OAuth-
    Token-Endpunkte spiegeln laut Spezifikation Credentials nicht in Fehlerantworten. Landet nur
    im Server-Log (Klasse a, nicht in der HTTP-Antwort), Zugriff nur fuer Admins mit Log-Zugriff.
    Gleiche Rechtfertigung wie die IP-Adressen-Ausnahme aus Lauf 10 (legitime technische
    Fehlerdiagnose einer Fremdsystem-Integration) — deshalb kein Fund, aber hier dokumentiert,
    falls ein spaeterer Lauf das anders bewertet.
- gate: n.a. (Scan-Unit, kein Produktionscode/Migration/Test angefasst — done_when verlangt kein
  go test)
- coverage: n.a. (Scan-Unit ohne Coverage-Ziel)
- mutations-probe: n.a. (Scan-Unit, kein neuer/geaenderter Testfall)
- verify vorgaenger: sauber. `cd2c6fad` (Iteration 62, scan-gobd-immutability-beyond-belegarchiv)
  geprueft: `git show --stat` zeigt ausschliesslich BACKLOG.yml und JOURNAL.md — kein
  Produktionscode, keine der acht Fehlerklassen einschlaegig.
- neue-units: keine
- offen:
  - Der DATEV-OAuth-Response-Body-Log (oauth.go:113-117/176-180) ist ein dokumentierter
    Grenzfall-Nichtfund, keine Unit — falls ein spaeterer Lauf oder Luke das anders bewertet,
    steht die Abwaegung hier zur Nachpruefung.
  - Nicht tief geprueft: `internal/gateway/route_integration.go` (Slack/Teams/Webhooks) — liegt
    ausserhalb der Finanzflaeche, die diese Unit adressiert, und war bereits Gegenstand einer
    eigenen Coverage-Unit (`cov-gateway-integration-config-routes`).

## Iteration 64 — scan-money-reads-without-tenant-scope — done — 2026-08-23 07:02
- commit: -
- gebaut: nichts am Produktionscode (Scan-Unit, ändert kein Verhalten). Recherche über einen
  Explore-Subagenten plus eigene Nachprüfung der DATEV-Fläche.
  Mechanismus verifiziert: `sysctx.With`/`database.WithSystemContext` setzt beim
  Connection-Checkout `app.role='system', app.tenant_id=''`
  (`internal/database/postgres.go` PrepareConn-Hook). Migration 000118 definiert
  `is_system_context()` und baut sie in JEDE RLS-Policy als
  `USING (tenant_id = current_tenant_id() OR is_system_context())` ein — unter Systemkontext
  filtert RLS auf KEINER Tabelle mehr irgendetwas; jede Query muss sich selbst nach tenant_id
  filtern.
  Geprüfte Grundgesamtheit: alle `sysctx.With`/`database.WithSystemContext`-Aufrufstellen
  (47 Fundorte über den gesamten Baum) plus alle Scheduler/Worker aus `cmd/*/main.go`
  (12 registrierte Ticker/Poller/Worker).
  Ergebnis je Geldfläche:
  - GDPR-Retention (`retention_scheduler.go`): sauber — iteriert explizit pro Tenant, jede
    Handler-Query filtert `WHERE tenant_id = $1`. Keine Geldtabellen unter den acht
    registrierten Retention-Handlern.
  - Automation-Poller / Invoice-Overdue-Dunning (`internal/automation/trigger/poller.go` +
    `due_postgres.go`): sauber, sogar mit explizitem Warnkommentar im Code
    ("... the poller runs under system context ... the WHERE clause is the only thing keeping
    tenant A's automation off tenant B's invoices."). Positivbeispiel.
  - Berichte-Scheduler (`internal/berichte/scheduler`): Code selbst tenant-gefiltert
    (`GetDefinition(ctx, in.TenantID, ...)`), aber `FinanceRepo`/`DatevBridge` sind in
    `cmd/berichte/main.go:58-60` aktuell `nil` — Finanz-Reportkinds sind inert, kein aktives
    Risiko.
  - Idempotenz-Cleanup (`internal/middleware/idempotency.go` cleanupCtx): ungefiltertes
    `DELETE FROM idempotency_keys WHERE expires_at < NOW()` — bewusst cross-tenant, reines
    Delete auf abgelaufenen Schlüsseln, keine Geldwert-Exposition, kein Fund.
  - DATEV-Upload (`internal/biz/datev/upload_service.go`): selbst nachgeprüft (vom Subagenten
    als unverifiziert offengelassen). `GetByPlatform(ctx, "datev_api")` ist an sechs Stellen
    ungefiltert nach tenant_id — ABER keine dieser sechs Stellen läuft unter Systemkontext.
    Der einzige `sysctx.With`-Aufruf im Paket (`HandleOAuthCallback:118`) wrapt `ExchangeCode`,
    und `ExchangeCode` (`oauth.go:149-202`) ruft `GetByPlatform` nicht auf (nur
    `vault.SetSecret` mit tenant-gescoptem Key). `UploadService` hat zudem keinen
    Scheduler/Worker in `cmd/biz/main.go` — nur gRPC-Server unter authentifiziertem Kontext.
    RLS bleibt dort aktiv, kein Fund.
  - Bexio (`internal/biz/bexio`): FUND. `postgres_config_repo.go:GetByPlatform` filtert nicht
    nach tenant_id. Unter dem Scheduler-Systemkontext lösen `SyncContacts` und `PollPayments`
    die Integration-Config über `getConfigID(ctx)` auf, OHNE das bereits bekannte `tenantID` zu
    nutzen — bei >1 aktivem Bexio-Tenant kann die falsche Config zurückkommen. Im Code selbst
    als "G8"-Bug kommentiert, aber nur für `PullInvoicesWithConfig` gefixt (Scheduler übergibt
    dort `ts.configID` direkt). `PollPayments` liest zusätzlich `ListEntityMappings` über
    `config_id` ohne tenant_id-Filter → Cross-Tenant-Verwechslung externer Bexio-Rechnungs-IDs
    im Zahlungsabgleich (Schreibseite selbst bleibt tenant-sicher, weil `invoices.GetByID`/
    `MarkPaid` korrekt `tenant_id` mitgeben).
  - Lexware (`internal/biz/lexware`): FUND, identisches Muster, aber OHNE jeden Fix-Kommentar.
    Betrifft zusätzlich zum Scheduler (`SyncContacts`) den WEBHOOK-Handler
    (`HandleWebhookEvent`, `service.go:262-280`) — jedes eingehende Lexware-Webhook-Event löst
    die Config über dasselbe ungefilterte `GetByPlatform` auf. Bei >1 aktivem Lexware-Tenant
    kann ein Kontakt-Update dem falschen Tenant zugeordnet werden.
  Beide Bugs sind laut Scheduler-Kommentaren aktuell latent (Produktivbetrieb ist
  Single-Tenant je Plattform), werden aber beim zweiten aktiven Tenant je Plattform sofort
  wirksam — passt zum G2-Fokus dieses Laufs (Geld- und Compliance-Pfade vor dem ersten
  zahlenden Kunden).
- gate: n.a. (Scan-Unit, kein Produktionscode/keine Migration/kein Test angefasst)
- coverage: n.a. (Scan-Unit ohne Coverage-Ziel)
- mutations-probe: n.a. (Scan-Unit, kein neuer/geänderter Testfall)
- verify vorgaenger: sauber. `52b4a28b` (Iteration 63, scan-finance-logs-error-leakage-and-pii)
  geprüft: `git show --stat` zeigt ausschliesslich BACKLOG.yml und JOURNAL.md, kein
  Produktionscode — keine der acht Fehlerklassen einschlägig.
- neue-units: fix-lexware-config-lookup-cross-tenant-under-sysctx (BACKLOG.yml, todo),
  fix-bexio-config-lookup-cross-tenant-under-sysctx (BACKLOG.yml, todo — betrifft gesperrtes
  Paket `internal/biz/bexio`, muss bei Ziehung in diesem Lauf sofort blocked werden)
- offen:
  - `fix-bexio-config-lookup-cross-tenant-under-sysctx` darf in Lauf 11 nicht gebaut werden
    (bexio-Sperre G3) — nur sofort blocken, falls sie dennoch gezogen wird.
  - `fix-lexware-config-lookup-cross-tenant-under-sysctx` muss für den Webhook-Pfad klären, ob
    es überhaupt ein Unterscheidungsmerkmal gibt, über das sich bei >1 Lexware-Tenant die
    richtige Config eindeutig auswählen lässt (Config-ID im Webhook-Pfad? Signatur-Secret pro
    Tenant?) — im Scan nicht geprüft, gehört in die bauende Iteration.

## Iteration 65 — fix-write-path-line-total-unrounded-everywhere — done — 2026-08-23 07:13
- commit: be1dea4d
- gebaut: Root-Cause-Fix statt sieben Einzel-Guards. Neue exportierte Funktion
  `tax.LineTotal(quantity, unitPrice) = quantity.Mul(unitPrice).Round(2)` in
  `internal/biz/tax/calculator.go` ist ab jetzt die EINZIGE Definition des Zeilennettos;
  `tax.Calculate` benutzt sie selbst. Umgestellt sind alle sieben Schreibstellen aus dem
  Unit-Scope — `invoice/service.go:151` (Create) und `:473` (Update), `quote/service.go:121`
  (Create) und `:305` (Update), `creditnote/service.go:100`, `recurring/service.go:482`
  (`priceLineItems`, Schreibpfad fuer Schedule UND jede daraus erzeugte Rechnung),
  `pdf/templates.go:154` — plus `internal/server/biz_grpc.go:2059`, der in der Unit nicht
  gelistete, aber in `fix-unrounded-line-total-in-invoice-service-callers` nachgewiesene
  Ausloeser (Rechnung aus der Zeiterfassung: `hours.Round(2).Mul(hourlyRate)` ungerundet).
  `pdf/templates.go` ist ehrlich ein reiner Konsolidierungs-Change ohne Ausgabeaenderung:
  `formatEUR` nutzt `StringFixed(2)`, das ohnehin half-away-from-zero rundet — steht so im
  Kommentar, damit niemand daraus einen Fix liest, der es nicht ist.
  Toleranzfrage in `einvoice.totalsTolerance` ENTSCHIEDEN: Toleranz bleibt mit der Zeilenzahl
  skalierend, der Doc-Kommentar war seit A4 falsch und ist korrigiert. Zwei Gruende, beide im
  Code: (1) die NETTO-Seite braucht sie nicht mehr (beide Seiten runden jetzt je Zeile), aber
  dieselbe Toleranz deckt in `assertTotalsMatch` auch STEUER und BRUTTO ab, und dort bleibt
  die Rundungs-REIHENFOLGE unterschiedlich (per-Zeile-Steuer vs. BR-CO-17-Gruppensteuer,
  bis zu einem halben Cent je Zeile); (2) Rechnungen, die VOR diesem Commit geschrieben
  wurden, tragen weiterhin ungerundete Subtotals — ein engeres Netto-Limit haette deren
  Export gekippt. Kein Migrations-Backfill (gebuchte Betraege, gehoert Luke).
- gate: build ok | vet ok | lint ok (0 issues) | test ok | migration n.a. | rls-smoke n.a.
  (keine Tabelle/Policy/Route/Proto angefasst). `go test -count=1 -p 2 ./internal/biz/...
  ./internal/server/ ./internal/gateway/` gruen auf dem FINALEN Tree (nach dem `stash`/`pop`
  der Baseline-Messung erneut gelaufen). Im ausfuehrlichen Lauf ueber die sieben
  betroffenen Pakete: 411 Tests gelaufen, **0 uebersprungen**, 0 rot — `DATABASE_URL` als
  `kmuhub_app` gesetzt, die DB-Integrationstests in invoice/quote/creditnote/recurring liefen
  also wirklich.
- coverage: eigene Messung je Paket, vorher -> nachher (Baseline via `git stash -u`, also
  ohne die neuen Testdateien, gegen denselben Tree): tax 100,0 % -> 100,0 % ·
  invoice 61,4 % -> 61,4 % · quote 33,3 % -> 33,3 % · creditnote 49,5 % -> 49,5 % ·
  recurring 88,5 % -> 88,5 % · pdf 52,0 % -> 52,0 % · einvoice 85,9 % -> 85,9 %.
  Delta 0 in allen sieben, und das ist kein Versaeumnis: die sechs neuen Tests fahren
  ausschliesslich Pfade, die schon abgedeckt WAREN (Create/Update/priceLineItems/
  buildInvoiceDoc) — sie beweisen ein anderes ERGEBNIS auf denselben Zeilen. Diese Unit
  hat kein Coverage-Ziel. `coverage_start:` der Unit nannte "internal/biz/invoice 43,1 %";
  gemessen sind 61,4 % — frueherer Lauf hat das Paket zwischenzeitlich angehoben, es gilt
  die eigene Messung.
- mutations-probe: `.Round(2)` in `tax.LineTotal` entfernt (`calculator.go:53`,
  `return quantity.Mul(unitPrice)`). Rot wurden daraufhin ALLE sechs Testpakete:
  `internal/biz/tax`, `invoice`, `quote`, `creditnote`, `recurring`, `einvoice` — die Probe
  belegt also nicht nur, dass irgendein Test anschlaegt, sondern dass jeder der sechs
  Schreibpfade seinen eigenen Waechter hat. Zurueckgedreht, `grep` auf die Zeile bestaetigt
  `.Round(2)`, `git diff --stat` sauber (14 Insertions in calculator.go, keine Reste).
- verify vorgaenger: sauber. `9c9c601d` (Iteration 64, scan-sysctx-money-reads-missing-tenant-
  filter) geprueft: `git show --stat` zeigt ausschliesslich BACKLOG.yml und JOURNAL.md, kein
  Produktionscode, keine Migration, kein Proto — keine der acht Fehlerklassen einschlaegig.
- neue-units: keine. Stattdessen ZWEI Backlog-Statusaenderungen:
  `fix-write-path-line-total-unrounded-everywhere` -> done, und
  `fix-unrounded-line-total-in-invoice-service-callers` -> done als von dieser Unit
  aufgeloest (dieselbe Fehlerklasse, dieselben Aufrufer; sie war die spaetere, praezisere
  Beschreibung desselben Bugs). Ein YAML-Kommentar an dieser zweiten Unit nennt, was
  bewusst NICHT nachgezogen wurde, damit das nicht als stiller Cut durchgeht.
- offen:
  - Kein eigener Regressionstest in `internal/biz/datev`. `exporter.go:265-267` liest den
    GESPEICHERTEN `LineTotal` und rundet den Bruttobetrag bereits selbst (mit Begruendung im
    Code, warum das Netto dort absichtlich ungerundet bleibt: es wird gar nicht geschrieben).
    Ein Test dort wuerde die Fixture pruefen statt den Code — die Invariante haengt jetzt an
    der Schreibseite und ist dort sechsfach getestet. Falls Luke das anders sieht: eigene Unit.
  - `internal/biz/bexio/field_mapper.go:303` rechnet beim Import weiterhin selbst
    `qty.Mul(unit)` ungerundet. Nicht angefasst — bexio ist in Lauf 11 gesperrt (G3). Der Wert
    laeuft ueber `invoice.toLineItems` in denselben Erzeugungspfad, und `toLineItems`
    uebernimmt den Fremdwert laut eigenem Kommentar ABSICHTLICH unveraendert ("imported
    invoices are read-only mirrors"). Das ist also kein Fall, den dieser Fix nebenbei mit
    erledigt — entgegen der Annahme in der Unit-Notiz von
    `fix-unrounded-line-total-in-invoice-service-callers`. Wenn importierte Bexio-Rechnungen
    exportiert werden sollen, braucht das eine eigene Entscheidung (Spiegel bleiben oder auf
    Cent normalisieren) und damit eine eigene Unit nach Aufhebung der bexio-Sperre.
  - Bestandsdaten: Rechnungen aus der Zeit vor diesem Commit haben weiterhin ungerundete
    `LineTotal`/`Subtotal` in der JSONB-Spalte. Der Export bleibt dank der beibehaltenen
    skalierenden Toleranz moeglich; ein Backfill waere eine Aenderung gebuchter Betraege und
    ist bewusst nicht passiert.

## Iteration 66 — fix-idempotency-409-rollout-remaining-routes — done — 2026-08-23 07:25
- commit: f6d4a3ad
- gebaut: Erste Teil-Unit des 739-Operationen-Rollouts (Iteration 15 legte den Vertrag +
  sechs Beispielrouten an, diese Unit deckt jetzt die komplette Registrar-Gruppe
  `finance` (`BizRoutes`) ab, statt weiter alles in einem `done_when` zu buendeln — die
  Unit selbst hatte gefordert, je Registrar-Gruppe zu schneiden).
  In `backend/api/openapi.yaml`, Zeilen 8710-10893 (44 mutierende POST/PUT/PATCH/DELETE-
  Operationen unter `/api/v1/finance/*`): 27 Operationen ohne jedes 409 haben jetzt
  `"409": { $ref: "#/components/responses/IdempotencyInFlight" }` ans Ende ihres
  `responses:`-Blocks angehaengt bekommen. 15 Operationen mit einem BESTEHENDEN 409 fuer
  einen anderen Grund (Geschaeftszustandskonflikt, z. B. "Quote is not in draft status")
  wurden NICHT ueberschrieben, sondern zusammengefuehrt: die Beschreibung traegt jetzt
  beide Bedeutungen in einem Satzpaar, und ein `headers.Retry-After` wurde ergaenzt mit
  dem Hinweis, dass er fuer den Konfliktfall oben nicht gesetzt ist. Zwei Operationen
  (POST /finance/invoices, POST /finance/invoices/{id}/payments) trugen den korrekten
  $ref bereits aus Iteration 15 und blieben unangetastet.
  Werkzeug: ein Python-Skript mit reinem Zeilen-/Indentations-Scan (kein YAML-Parse-und-
  Reserialize — zerstoert Formatierung auf der 47k-Zeilen-Datei), Edits von unten nach
  oben ausgefuehrt, damit Zeilennummern beim Einfuegen stabil bleiben.
  `fix-idempotency-409-rollout-remaining-routes` in BACKLOG.yml ist umgeschrieben auf den
  tatsaechlich gelieferten Scope (Finance) und auf `done` gesetzt; die Folge-Unit
  `fix-idempotency-409-rollout-non-finance-routes` deckt die restlichen ~40
  Registrar-Gruppen und steht am Backlog-Ende.
- gate: build ok | vet ok | lint n.a. (kein Go-Code geaendert, nur openapi.yaml) | test ok
  | migration n.a. (keine Tabelle/Migration angefasst) | rls-smoke n.a. (keine
  Tabelle/Policy angefasst). `python3 -c "yaml.safe_load(...)"` parst die Datei fehlerfrei
  (838 Pfade), `npx swagger-cli validate api/openapi.yaml` -> "api/openapi.yaml is valid",
  `go test -count=1 ./internal/gateway/ -run TestOpenAPIRouteDrift` gruen,
  `go test -count=1 ./internal/gateway/...` gruen (voller Paketlauf, 0 uebersprungen,
  DATABASE_URL als kmuhub_app gesetzt). `git diff --stat` zeigt 43 Hunks, alle innerhalb
  der Zeilen 8710-11025 (Finance-Block) — kein Fund ausserhalb des beabsichtigten Bereichs.
- coverage: n.a. (Doku-Unit, keine Coverage-Ziel — spiegelt die eigene coverage_start-Zeile
  der Unit)
- mutations-probe: n.a. (kein Testverhalten, sondern eine OpenAPI-Spec-Erweiterung; das
  Gate hierfuer ist `swagger-cli validate` + `TestOpenAPIRouteDrift`, keine Testlogik zum
  Brechen)
- verify vorgaenger: sauber. `be1dea4d` (Iteration 65, fix-write-path-line-total-unrounded-
  everywhere) geprueft: `git show --stat` zeigt Aenderungen ausschliesslich in
  `internal/biz/{creditnote,invoice,quote,recurring,tax,pdf,einvoice}` und
  `internal/server/biz_grpc.go` (eine Zeile). Kein neuer Handler mit direktem Service-
  Call statt gRPC-Client, keine neue Tabelle, kein `.proto` ohne Regen, kein
  `RequirePermission` ohne Seed, keine neue Route, kein hart ersetzter Guard. Die neuen
  `tax.LineTotal`-Aufrufe laufen alle innerhalb bestehender Service-Methoden, keine
  Umgehung des gRPC-Layers.
- neue-units: fix-idempotency-409-rollout-non-finance-routes (deckt die verbleibenden
  ~40 Registrar-Gruppen, Backlog-Ende)
- offen:
  - Die Folge-Unit muss weiter in Teil-Units je Registrar-Gruppe (oder wenige kleine
    zusammen) geschnitten werden — nicht in einem Commit, aus demselben Grund wie hier:
    39 Registrar-Gruppen in einem Diff sind fuer eine Nachtlauf-Iteration nicht reviewbar.
  - Kein DB-Bezug in dieser Unit, daher kein RLS-Smoke faellig.

## Iteration 67 — fix-hr-time-entries-manual-post-not-in-openapi — done — 2026-08-23 07:32
- commit: d171d8f7
- gebaut: `POST /api/v1/hr/time/entries` (`HandleCreateManualEntry`) war der urspruengliche
  Fund — dokumentiert, mit Request-Body (alle Felder aus `createManualEntryHTTPReq`),
  Idempotency-Key als Pflicht-Header und Responses 201/400/401/403/409/503 (409 fuer
  `ErrWeekLocked` ueber `assertWeekEditable`, per Codepfad in `hr_grpc.go:2135` belegt).
  Der geforderte systematische Scan wurde NICHT als separates Skript gebaut, sondern als
  dauerhafter Test: `internal/gateway/openapi_drift_test.go` bekam
  `registeredAPIv1MethodPaths` (chi.Walk liefert Methode UND Pfad, bisher wurde die Methode
  verworfen), `documentedMethodPaths` (derselbe Zeilen-Scan-Ansatz wie `documentedPaths`,
  nur eine Ebene tiefer auf `get|post|put|patch|delete|head|options`-Schluessel) und
  `TestOpenAPIMethodDrift`, das beide Mengen als "METHOD /pfad"-Paare vergleicht — die
  robuste, dauerhafte Variante, die die Unit-Notes selbst als Alternative zum
  Einmal-Scan nannten.
  Der erste Testlauf fand neun weitere Faelle derselben Fehlerklasse (Pfad dokumentiert,
  Methode fehlt): `GET`+`DELETE /api/v1/notifications/dnd` (nur `post:` war dokumentiert;
  dabei nebenbei einen echten Wire-Shape-Fund behoben — die bestehende `post:`-Antwort
  behauptete `$ref: QuietHours`, tatsaechlich liefert `HandleToggleDND` seit jeher die flache
  Form aus `dndStatusFromQuietHours` (`{is_active, expires_at?}`); neues Schema `DNDStatus`
  fuer alle drei Operationen), `DELETE /api/v1/projects/{id}` (`HandleDeleteProject`, war
  komplett undokumentiert), und sechs CRM-`PUT`-Routen (`companies`, `contacts`,
  `custom-fields`, `deals`, `pipeline-stages`, `tags` je `/{id}`), die alle als `patch:`
  dokumentiert waren, aber `route_crm.go` registriert sie durchgehend als `Put(...)`.
  Vor dem Fix im Desktop-Client gegengeprueft (lesend, Frontend ist gesperrt): `useCompanies.ts`,
  `useContacts.ts`, `useDeals.ts`, `usePipelineStages.ts`, `useContactTags.ts` senden alle
  tatsaechlich `PUT` — die Spec war falsch, nicht der Code. Fuer `custom-fields` fand sich kein
  Frontend-Aufrufer (die Manager-Komponente ruft die Route nicht auf); da alle uebrigen fuenf
  eindeutig PUT sind, auf `put:` vereinheitlicht statt eine Ausnahme zu bauen.
  Alle neun Faelle sind reine Dokumentationskorrekturen (`patch:`→`put:` bzw. fehlende
  Operation ergaenzt) — root cause laut Unit-Notes gehoert in dieselbe Unit, nicht in
  Einzel-Units je Route.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (0 uebersprungen, DATABASE_URL als
  kmuhub_app) | migration n.a. | rls-smoke n.a. (keine Tabelle/Policy angefasst).
  `swagger-cli validate api/openapi.yaml` → "is valid". `python3 -c yaml.safe_load` zaehlt
  weiterhin 838 Pfade (unveraendert — nur Methoden/Schemas ergaenzt, kein neuer Pfad).
  `go test -count=1 ./internal/gateway/...` gruen, alle vier `TestOpenAPI*`-Tests einzeln
  gruen (`TestOpenAPIRouteDrift`, `TestOpenAPIMethodDrift` NEU, `TestOpenAPISpecDrift`,
  `TestOpenAPIRouteDriftParserSanity`).
- coverage: n.a. (Doku-Luecke, kein Coverage-Ziel — deckt die eigene coverage_start-Zeile)
- mutations-probe: `get:` unter `/api/v1/hr/time/entries` testweise zu `getx:` verstuemmelt →
  `TestOpenAPIMethodDrift` wurde rot mit exakt der erwarteten Meldung ("GET
  /api/v1/hr/time/entries" fehlt), zurueckgedreht (`cp` vom Backup vor der Aenderung),
  `git diff --stat backend/api/openapi.yaml` zeigt danach wieder denselben Stand wie vor der
  Probe (142 Zeilen Delta, keine Restspur). Damit ist belegt, dass der neue Test echte
  Methoden-Luecken faengt, nicht nur gruen bleibt.
- verify vorgaenger: sauber. `f6d4a3ad` (Iteration 66, fix-idempotency-409-rollout-remaining-
  routes) geprueft: reiner `openapi.yaml`-Diff (167 Zeilen, ausschliesslich `409`-Response-
  Ergaenzungen/-Zusammenfuehrungen unter `/api/v1/finance/*`), kein Go-Code veraendert, also
  keine der acht Fehlerklassen einschlaegig (kein Handler, keine Route, kein Proto, kein
  Guard). Eigener `python3 -c yaml.safe_load`-Lauf bestaetigt 838 Pfade, deckungsgleich mit
  der im Journal behaupteten Zahl.
- neue-units: fix-hr-manual-entry-idempotency-key-not-enforced (Backlog-Ende) — Nebenfund beim
  Bauen der `POST /entries`-Doku: der Pflicht-Header `Idempotency-Key` wird bis
  `ManualEntryInput.IdempotencyKey` durchgereicht, dort aber nie gelesen (kein Uniqueness-
  Check, keine Spalte auf `hr_work_time_entries`) — ein wiederholter Request mit demselben Key
  erzeugt zwei bezahlte Arbeitszeiteintraege statt einem. Fixentscheidung (Middleware-Ebene vs.
  eigene Spalte+Unique-Index) gehoert Luke, siehe `blocked`-artige Notes in der neuen Unit.
- offen:
  - `fix-hr-manual-entry-idempotency-key-not-enforced` braucht eine Architekturentscheidung von
    Luke (Idempotency-Middleware vs. eigene DB-Spalte), bevor sie gebaut werden kann.
  - Der DNDStatus-Schema-Fix (QuietHours→DNDStatus) auf der bestehenden `post:`-Operation ist
    eine Verhaltens-neutrale Doku-Korrektur (der Handler-Code aenderte sich nicht), aber falls
    ein Frontend-Codegen aus der Spec bisher `QuietHours`-Felder fuer die POST-Antwort erwartet
    hat, greift das ins Leere — lesend gegengeprueft (kein Treffer fuer `user_id`/`start_time`
    aus der DND-Antwort im Desktop-Code), aber nicht mit einem laufenden Frontend-Build
    verifiziert (Frontend ist in diesem Lauf gesperrt).
  - Kein DB-Bezug in dieser Unit (reine OpenAPI-Doku + ein neuer Go-Test), daher kein
    RLS-Smoke faellig.

## Iteration 68 — verify-invoice-list-for-gobd-export-unreachable — done — 2026-08-23 07:42
- commit: d0388b26
- gebaut: VERIFY-UNIT mit Entfernung. Aufrufgraph von `invoice.Service.ListForGoBDExport`
  belegt: Grep ueber gesamtes Repo (`internal/`, `cmd/`, `.proto`) findet nur die
  Repository-Implementierung, das Interface, die Service-Methode und drei Test-Doubles
  (`MockRepository` in `service_test.go`, `stubInvoiceRepo` in
  `biz_grpc_invoices_creditnotes_payments_test.go`, sowie die eigene DB-Testdatei) — kein
  `cmd/`, kein Gateway-Handler, kein anderer Service ruft sie auf. Der tatsaechliche
  GoBD-Export (`GenerateGoBDExport`, `internal/server/biz_grpc.go:2459`) laeuft nachweislich
  ueber `invoiceService.ListForDATEVExport` (sign=1) + `creditNoteService.ListForDATEVExport`
  (sign=-1, Stornos) via `dunning.BuildGoBDRows` — voellig unabhaengig von
  `ListForGoBDExport`. Da kein Produktvorhaben fuer einen zweiten Export-Modus erkennbar ist
  (keine offene Unit, kein Kommentar, kein Frontend-Hinweis — Frontend gesperrt, nur lesend
  geprueft) und die Methode aktiv irrefuehrend ist (sieht aus wie DER GoBD-Pfad, ist es aber
  nicht — inklusive einer bewusst abweichenden Statuslogik, die nie ausgefuehrt wird),
  ENTSCHEIDUNG: Entfernung statt lean-Marker. Entfernt: Repository-Methode
  (`postgres_repository.go`, 49 Zeilen inkl. Line-Item-Ladung), Interface-Eintrag
  (`repository.go`), Service-Wrapper (`service_gobd.go`), `MockRepository`-Implementierung
  (`service_test.go`, unbenutzt — kein Testfall rief sie auf), `stubInvoiceRepo`-Implementierung
  (`biz_grpc_invoices_creditnotes_payments_test.go`), gesamte DB-Testdatei
  `postgres_repository_gobd_export_db_test.go` (7 Tests gegen eine nie aufgerufene Methode).
  Am tatsaechlichen Pfad (`ListForDATEVExport`-Doc-Kommentar in `postgres_repository.go`)
  einen Verweis hinterlassen, damit klar ist, dass es keine separate GoBD-Export-Methode gibt.
- gate: build ok (`go build -p 2` ueber invoice/server/einvoice/recurring/cmd/biz) | vet ok |
  lint ok (0 issues, invoice+server) | test ok (invoice, server, server/response, einvoice,
  recurring alle gruen, 0 uebersprungen in invoice mit -v gegengeprueft) | migration n.a.
  (kein Schema-Bezug) | rls-smoke n.a. (keine Tabelle/Policy angefasst)
- coverage: n.a. (Verify-Unit mit Entfernung, kein Coverage-Ziel — Backlog-Zeile deckt das)
- mutations-probe: n.a. (Entfernung toten Codes, keine neue Verhaltenslogik zum Brechen;
  der Beweis ist der leere Aufrufgraph selbst, nicht ein Testverhalten)
- verify vorgaenger: sauber. `d171d8f7` (Iteration 67) geprueft: reine
  OpenAPI-Doku-Korrektur (openapi.yaml + neuer Drift-Test), kein Handler-/Service-Code
  geaendert. Stichprobe `dndStatusFromQuietHours`/`HandleGetDND`/`HandleDisableDND` im echten
  Code bestaetigt die im Journal behaupteten Signaturen und die DNDStatus-Wire-Shape-Korrektur
  — keine der acht Fehlerklassen einschlaegig. `539e2ca0` (nur Sha-Nachtrag im Journal-Text,
  keine Code-Aenderung) separat gegengeprueft, ist ein Ein-Zeilen-Diff in JOURNAL.md.
- neue-units: keine
- offen:
  - Kein DB-Bezug, daher kein RLS-Smoke faellig.
  - Falls spaeter doch ein zweiter GoBD-Export-Modus gebraucht wird (z. B. cancelled-Rechnungen
    direkt statt ueber Credit-Note-Stornos), ist das eine Produktentscheidung Luke — der
    Kommentar an `ListForDATEVExport` verweist auf den heutigen alleinigen Pfad, damit eine
    kuenftige Wiedereinfuehrung nicht denselben unklaren Zustand reproduziert.

## Iteration 69 — fix-payment-stats-outstanding-ignores-recorded-payments — done — 2026-08-23 07:47
- commit: 3ee7705d
- gebaut: `AggregatePaymentStats` (`postgres_repository.go:528-561`) joint jetzt gegen
  `finance_payments` wie `postgres_open_items.go`s `openItemsBase`. `total_outstanding_amount`
  summiert `gross_total - COALESCE(paid, 0)` statt des vollen Brutto — eine Teilzahlung senkt
  den ausgewiesenen offenen Betrag. `total_paid_amount` summiert `COALESCE(paid, gross_total)`
  je bezahlter Rechnung — ist eine Zahlung erfasst, zaehlt der tatsaechlich vereinnahmte
  Betrag (Ueberzahlung sichtbar), ist keine erfasst (manuell auf "paid" gesetzt ohne
  Zahlungs-Tracking), bleibt `gross_total` der Rueckfallwert. Cashflow-vs-Umsatz-Entscheidung:
  KEINE UI-Beschriftung vorhanden, die die Frage beantworten koennte — `usePaymentStats`
  (`desktop/src/renderer/src/api/hooks/useFinance.ts:646`) ist definiert, hat aber null
  Aufrufer in `.tsx`/`.ts` (Dashboard rendert die KPI aktuell nicht). Entscheidung daher aus
  dem Feldnamen selbst: "TotalPaidAmount" heisst tatsaechlich vereinnahmtes Geld, nicht
  Rechnungswert — konsistent mit der Netting-Logik der Gegenseite. Die beiden
  "current behaviour"-Tests (`PartialPaymentNotNettedFromOutstanding`,
  `PaidAmountIgnoresOverpayment`) sind auf das neue Verhalten umgeschrieben und umbenannt
  (`PartialPaymentIsNettedFromOutstanding` → 600 statt 1000,
  `PaidAmountReflectsOverpayment` → 450 statt 400), die vier uebrigen Tests (leer, Klassifizierung
  ohne Zahlungssatz, Durchschnittstage, Tenant-Scope) blieben unveraendert gruen — die
  Klassifizierungs-Testrechnung (bezahlt ohne Zahlungssatz, 500) beweist den Rueckfallpfad
  `COALESCE(paid, gross_total)`.
- gate: build ok (`go build -p 2` invoice+server+cmd/biz) | vet ok | lint ok (0 issues,
  invoice+server) | test ok (`go test -v ./internal/biz/invoice/` 125/125 PASS, 0 SKIP;
  `go test ./internal/server/` gruen) | migration n.a. (kein Schema-Bezug) | rls-smoke n.a.
  (keine Tabelle/Policy angefasst)
- coverage: n.a. (coverage_start deklariert "Fix-Unit, kein Coverage-Ziel"; gemessen
  internal/biz/invoice 60,9 % nach der Aenderung, als Beleg dass kein Regress)
- mutations-probe: `total_outstanding_amount`-Netting testweise auf reines `SUM(gross_total)`
  zurueckgedreht (Zeile 543) → `TestPostgresRepository_AggregatePaymentStats_
  PartialPaymentIsNettedFromOutstanding` wurde rot mit exakt der erwarteten Meldung ("got
  1000" statt 600), alle uebrigen fuenf Tests blieben gruen (bestaetigt, dass die anderen
  Assertions unabhaengig von dieser einen Zeile sind). Zurueckgedreht per `cp` vom Backup,
  `grep -n "total_outstanding_amount\|total_paid_amount"` gegen die Datei bestaetigt den
  wiederhergestellten Zustand, `git diff --stat` zeigt danach wieder denselben Umfang (23
  Einfuegungen/11 Loeschungen) wie vor der Probe.
- verify vorgaenger: sauber. `d0388b26` (Iteration 68, verify-invoice-list-for-gobd-export-
  unreachable) geprueft: reine Entfernung toten Codes (Repository-Methode, Interface-Eintrag,
  Service-Wrapper, zwei Test-Doubles, eine DB-Testdatei). Eigener Grep
  `grep -rn ListForGoBDExport backend` bestaetigt null verbleibende Treffer im gesamten
  Backend — die Entfernung ist vollstaendig, keine der acht Fehlerklassen einschlaegig (kein
  neuer Handler/Proto/Guard/Route/Tabelle).
- neue-units: keine
- offen:
  - Kein DB-Schema-Bezug, daher kein RLS-Smoke faellig.
  - Der Fallback `COALESCE(paid, gross_total)` fuer als "paid" markierte Rechnungen ohne
    `finance_payments`-Zeile ist eine bewusste Annahme (manuell geschlossen = voll bezahlt).
    Sollte KMU Hub kuenftig erzwingen, dass jede "paid"-Rechnung eine Zahlung traegt, kann
    dieser Fallback entfallen — hier nicht angefasst, da ausserhalb des Fund-Scopes.
  - `usePaymentStats` bleibt ohne Aufrufer im Desktop-Client (Frontend gesperrt, nur lesend
    geprueft) — die KPI existiert im Backend, wird aber aktuell nirgends angezeigt.

## Iteration 70 — fix-quote-to-invoice-duplicate-creation — done — 2026-08-23 07:52
- commit: db991347
- gebaut: `Service.CreateFromQuote` (`service.go:797-816`) liest vor dem `Create` ueber
  `repo.GetByQuoteID` und lehnt mit dem neuen Sentinel `ErrQuoteAlreadyConverted` ab, wenn
  bereits eine NICHT stornierte Rechnung zum Angebot existiert (`mapBizError` →
  `codes.AlreadyExists` → HTTP 409, `helpers.go:51`). Ein Storno hebt die Sperre bewusst
  wieder auf. Damit dieser Check ueberhaupt tragen kann, hat `GetByQuoteID`
  (`postgres_repository.go:450-458`) jetzt eine explizite Ordnung
  (`ORDER BY (status = 'cancelled'), created_at DESC LIMIT 1`) — vorher waehlte der Planner
  bei mehreren Treffern beliebig, eine stornierte Rechnung haette die lebende maskieren und
  die Sperre aushebeln koennen. `MockRepository.GetByQuoteID` in service_test.go spiegelt
  diese Ordnung (Map-Iteration ist zufaellig, sonst waere der Test flaky). Vier neue Tests:
  zweite Konversion abgelehnt und KEINE zweite Rechnung geschrieben, Re-Konversion nach
  Storno erlaubt, Lookup-Fehler wird durchgereicht statt als "noch keine Rechnung"
  fehlinterpretiert, plus ein DB-Test, der die Ordnung an echtem Postgres festnagelt
  (die stornierte Rechnung ist dort absichtlich die NEUERE).
- gate: build ok (`go build -p 2` invoice+server+gateway+cmd/biz+cmd/gateway) | vet ok |
  lint ok (0 issues, invoice+server+gateway) | test ok (`go test -count=1` invoice 138 PASS
  / **0 SKIP** — alle DB-Tests real gelaufen, DATABASE_URL gegen `kmuhub_app`; server ok;
  gateway ok inkl. TestOpenAPIRouteDrift) | migration n.a. (kein Schema-Bezug, siehe
  neue-units) | rls-smoke n.a. (keine Tabelle/Policy angefasst) | openapi:
  `swagger-cli validate` = valid
- coverage: `internal/biz/invoice` 60,9 % -> 61,1 % (eigene Messung vor/nach; die 60,9 %
  stammen aus Iteration 69, `coverage_start` der Unit deklariert "n.a., Fix-Unit")
- mutations-probe: zwei Mutationen gleichzeitig gesetzt. (1) `ORDER BY (status =
  'cancelled'),` aus der Query entfernt (also nur noch `created_at DESC`) →
  `TestPostgresRepository_GetByQuoteID_PrefersLiveInvoiceOverCancelled` rot (bekam die
  stornierte, neuere Rechnung). (2) `return nil, ErrQuoteAlreadyConverted` durch einen
  No-Op ersetzt → `TestService_CreateFromQuote_RejectsSecondConversion` rot mit allen drei
  Assertions (kein Fehler, zweite Rechnung zurueckgegeben, zwei Rechnungen im Repo). Alle
  uebrigen Tests des Pakets blieben in beiden Faellen gruen. Zurueckgedreht per `cp` vom
  Backup, per `grep` an beiden Stellen verifiziert, `git diff --stat` danach wieder
  identisch (8 Dateien, 187/19).
- verify vorgaenger: sauber. `3ee7705d` (Iteration 69) geprueft: reine SQL-Aenderung in
  `AggregatePaymentStats`, kein neuer Handler/Guard/Route/Proto/Tabelle. Der neu
  eingefuehrte LEFT-JOIN auf `finance_payments` traegt sein eigenes `WHERE tenant_id = $1`
  in der Subquery — keine Tenant-Luecke ueber den Join. Keine der acht Fehlerklassen.
- neue-units: `feat-quote-converted-invoice-number-on-read`,
  `harden-quote-conversion-unique-index`
- offen:
  - **Fachfrage Teilrechnungen beantwortet: NEIN, im Produkt nicht vorgesehen.** Belege
    (nicht geraten): `grep -rni "teilrechnung|partial.invoice|abschlagsrechnung|
    anzahlungsrechnung" backend desktop/src docs .knowledge` = **null Treffer**;
    `postgres_document_chains.go:3-13` modelliert die Belegkette ausdruecklich als
    Angebot→Rechnung 1:1 und listet ein Angebot als "standalone", solange KEINE Rechnung
    daran haengt (`NOT EXISTS`, Zeile 184-187); `QuoteDetailPanel.tsx:98` erwartet ein
    einzelnes `converted_invoice_number` (Singular), keine Liste. Der pauschale Riegel ist
    damit richtig, eine betragsbezogene Feinunterscheidung waere Spekulation.
  - **Neuer verifizierter FE/BE-Befund (deshalb Unit `feat-quote-converted-invoice-number-
    on-read`):** das FE blendet den Button "In Rechnung umwandeln" ueber
    `!convertedNumber` aus (`QuoteDetailPanel.tsx:368`), aber `converted_invoice_number`
    existiert im Backend nirgends (`grep -rn converted_invoice_number backend` = 0 Treffer,
    auch nicht im generierten `biz.pb.go`). Der Guard ist also dauerhaft wirkungslos — der
    Button bleibt nach der Konversion sichtbar, und genau deshalb war der hier gefixte Bug
    im UI ueberhaupt mit einem zweiten Klick ausloesbar. Der 409 faengt das jetzt ab, aber
    die UI zeigt weiterhin eine Aktion an, die nicht mehr geht.
  - **DB-Index bewusst NICHT gebaut** (Unit `harden-quote-conversion-unique-index`): das
    `done_when` dieser Unit verlangt ausdruecklich, dass
    `TestPostgresRepository_GetByQuoteID_MultipleInvoices` (legt zwei nicht stornierte
    Rechnungen zum selben Quote an) gruen bleibt — ein Unique-Index haette ihn gebrochen.
    Ausserdem laesst sich ohne Prod-Zugriff nicht pruefen, ob heute schon Mehrfach-
    Rechnungen pro Angebot existieren; die Migration wuerde dann beim Deploy scheitern.
    Der verbleibende Race (zwei wirklich gleichzeitige Requests) ist im Code mit einem
    `lean:`-Marker samt Upgrade-Trigger vermerkt.
  - Die 409-Beschreibung der Route `/api/v1/finance/quotes/{id}/convert` in `openapi.yaml`
    nennt jetzt beide Konfliktfaelle (nicht akzeptiert / bereits umgewandelt). Keine neue
    Route, kein Drift.

## Iteration 71 — fix-banking-import-race-returns-raw-pg-error — done — 2026-08-23 08:01
- commit: 1d160dae
- gebaut: `CreateStatement` mappt die Unique-Violation auf
  `finance_bank_statements_hash_unique` jetzt ueber den Constraint-Namen auf den neuen
  Sentinel `ErrStatementHashConflict` (`isStatementHashConflict`, Vorlage
  `isAccountIBANConflict`). `Service.Import` faengt den Sentinel ab und beantwortet den
  Verlierer der Race ueber den neuen Helper `alreadyImported` genauso wie den regulaeren
  Re-Import-Fall (gleicher Code-Pfad, keine Verzweigung mehr im Aufrufer). Der frueher
  inline im "Verlierer"-Zweig stehende Code ist jetzt derselbe Helper, den auch der
  fruehe Check am Funktionsanfang nutzt.
- gate: build ok (`go build -p 2 ./internal/biz/banking/...`) | vet ok | lint ok
  (0 issues) | test ok (`go test -count=1` banking, 0 SKIP, DATABASE_URL gegen
  `kmuhub_app`) | migration n.a. (kein Schema-Bezug) | rls-smoke n.a. (keine
  Tabelle/Policy angefasst) | openapi n.a. (keine Route beruehrt)
- coverage: `internal/biz/banking` 85,6 % -> 85,7 % (eigene Messung vor/nach per
  `git stash`; `coverage_start` der Unit deklariert "n.a., Verhaltensfund")
- mutations-probe: `isStatementHashConflict`s Constraint-Namen-Vergleich testweise auf
  einen falschen String gesetzt (`"wrong_constraint_name_mutation_probe"`) →
  `TestPostgresRepository_CreateStatement_DuplicateContentHashIsRejectedNotDuplicated`
  UND die neue `TestServiceImport_ConcurrentUploadsOfSameFile_BothGetAValidResult`
  wurden beide rot (roher `23505`-Fehler statt `ErrStatementHashConflict` bzw. ein
  Import-Aufruf gab einen Fehler statt eines gueltigen Ergebnisses zurueck).
  Zurueckgedreht per `cp` vom Backup, `diff` gegen das Backup leer, Gate danach wieder
  gruen.
- verify vorgaenger: sauber. `db991347` (Iteration 70) geprueft: kein neuer
  Gateway-Handler (bestehende Route, Aufruf laeuft weiter ueber den vorhandenen
  gRPC-Pfad), kein Proto/`.pb.go`-Bezug (`git show --stat` zeigt keine `.proto`-Datei),
  kein neuer `RequirePermission`-Guard, `GetByQuoteID` bleibt tenant-gescoped
  (`WHERE tenant_id = $1`), der neue `codes.AlreadyExists`-Fall ist konsistent zur
  OpenAPI-409-Doku-Aktualisierung im selben Commit, keine neue Tabelle. Keine der acht
  Fehlerklassen.
- neue-units: keine
- offen:
  - Die neue Race-Probe (`TestServiceImport_ConcurrentUploadsOfSameFile_BothGetAValidResult`,
    8 parallele `svc.Import`-Aufrufe mit identischem Hash) lief lokal reproduzierbar
    gruen (1 Gewinner per `CreateStatement`, 7 ueber `ErrStatementHashConflict` auf
    "already imported" gemappt) — sie treibt die Race ueber echte, fast gleichzeitige
    Requests gegen dieselbe Postgres-Instanz, ist aber naturgemaess timing-abhaengig;
    sollte sie in CI je flaky werden, ist die Anzahl der Attempts der erste Hebel.
  - `alreadyImported`-Helper macht im regulaeren (nicht-race) Re-Import-Pfad denselben
    `GetStatementByHash`-Aufruf, den der fruehe Check schon gemacht hat, ein zweites
    Mal (statt das erste Ergebnis durchzureichen) — bewusst so belassen, weil der
    Race-Zweig zwingend neu lesen muss (der fruehe Check hat dort ja "not found"
    gesehen) und zwei separate Codepfade fuer denselben Rueckgabewert die schlechtere
    Wahl waeren. Eine zusaetzliche DB-Rundreise pro Re-Import, kein Korrektheitsproblem.

## Iteration 72 — fix-invoice-number-unique-index-blocks-second-draft — done — 2026-08-23 08:07
- commit: ccee9d4f
- gebaut: Neue Forward-Migration `000323_finance_invoice_number_partial_unique`
  ersetzt den nicht-partiellen Unique-Index `idx_finance_invoices_number` durch
  `... (tenant_id, invoice_number) WHERE invoice_number <> ''`. Damit darf ein Tenant
  beliebig viele unversendete Entwuerfe (`invoice_number = ''`) halten, waehrend
  vergebene Nummern pro Tenant eindeutig bleiben. Kein Go-Code am Produktivpfad
  geaendert — `Create` setzt `InvoiceNumber: ""` weiterhin bewusst, die Ursache lag
  allein im Schema. Neue Datei
  `internal/biz/invoice/postgres_repository_draft_number_uniqueness_db_test.go` mit
  drei Real-SQL-Tests (drei unnummerierte Entwuerfe pro Tenant gelingen; dieselbe
  vergebene Nummer zweimal wird weiterhin mit 23505 auf genau diesem Index abgelehnt;
  dieselbe Nummer in zwei Tenants bleibt erlaubt). Der `invSvc.Send`-Workaround in
  `recurring/TestGenerate_RealSQL_MonthEndScheduleStaysAnchored` ist entfernt — der
  Test erzeugt jetzt zwei Perioden als echte Entwuerfe hintereinander, so wie ein
  echter Zeitplan es tut.
- gate: build ok (`go build -p 2 ./internal/biz/invoice/... ./internal/biz/recurring/...`) |
  vet ok | lint ok (0 issues) | test ok (`go test -count=1` invoice + recurring,
  **0 SKIP** bei `-v` gezaehlt, DATABASE_URL gegen `kmuhub_app`; zusaetzlich
  `./migrations/` und `./internal/testutil/` gruen) | migration ok (323 up angewendet,
  danach `down 1` + `up` als Roundtrip gefahren, Indexdefinition per `pg_indexes`
  in beiden Richtungen verifiziert) | rls-smoke n.a. (nur ein Index getauscht, keine
  Tabelle, keine Policy, keine Spalte) | openapi n.a. (keine Route beruehrt)
- coverage: `internal/biz/invoice` 61,1 % -> 61,1 % (eigene Messung vor/nach, die neue
  Testdatei zum Messen des Vorher-Werts herausgenommen). Unveraendert und das ist
  korrekt so: der Fix sitzt im Schema, die drei neuen Tests fahren mit `repo.Create`
  einen bereits abgedeckten Go-Pfad — sie beweisen Verhalten, keine neuen Zeilen.
  `coverage_start` der Unit deklariert "n.a., Verhaltensfund".
- mutations-probe: Mutiert wurde die Migration selbst (die eigentliche Fix-Stelle):
  `migrate down 1` setzt den Index auf die nicht-partielle Form zurueck. Daraufhin
  wurden BEIDE Seiten rot — `TestPostgresRepository_Create_MultipleUnnumberedDraftsPerTenant`
  ("draft 2 without an invoice number must be creatable alongside the earlier ones")
  und `TestGenerate_RealSQL_MonthEndScheduleStaysAnchored` mit exakt dem im Scope
  beschriebenen Fehler `insert finance_invoices: ERROR: duplicate key value violates
  unique constraint "idx_finance_invoices_number" (SQLSTATE 23505)`. Danach `migrate up`,
  beide Pakete wieder gruen, Arbeitsbaum unveraendert (die Probe fasste keine Datei an).
- verify vorgaenger: sauber. `1d160dae` (Iteration 71) geprueft: kein Gateway-Handler und
  damit keine gRPC-Umgehung, kein Stub (`isStatementHashConflict` prueft real gegen
  `pgconn.PgError`), keine `.proto`-Datei im Commit, kein neuer `RequirePermission`-Guard
  und kein ersetzter Alt-Key, keine neue Tabelle, Wire-Shape unveraendert (`ImportResult`
  gleich, nur `AlreadyImported: true` statt Fehler), keine neue Route. Zusaetzlich
  geprueft, weil der neue Early-Return in einer offenen Transaktion sitzt: `CreateStatement`
  hat `defer func() { _ = tx.Rollback(ctx) }()` gleich nach `Begin`, der
  `return ErrStatementHashConflict` laesst also keine Transaktion offen. Auch der
  Re-Read im Race-Zweig ist korrekt: Postgres liefert die Unique-Violation erst,
  nachdem der Gewinner committet hat (der Insert blockt bis dahin), die Zeile ist
  fuer den anschliessenden `GetStatementByHash` also sichtbar.
- neue-units: keine
- offen:
  - **Produktionsdatenpruefung war aus dem Loop heraus nicht moeglich** (kein
    Prod-Zugriff, per Grenze des Laufs). Sie ist fuer die Up-Richtung auch nicht
    noetig und das steht als Begruendung im Migrationskopf: eine Unique-Bedingung zu
    LOCKERN kann an Bestandsdaten nie scheitern, jede Zeilenmenge, die den alten Index
    erfuellte, erfuellt auch den neuen. Lokal geprueft (2026-08-23):
    `SELECT tenant_id, count(*) FROM finance_invoices WHERE invoice_number='' GROUP BY 1
    HAVING count(*)>1` liefert 0 Zeilen — erwartungsgemaess, weil der alte Index genau
    das verhinderte.
  - Die **`.down.sql` kann dagegen sehr wohl scheitern**, sobald die partielle Fassung
    eine Weile live war und ein Tenant mehrere Entwuerfe haelt. Das ist im
    `.down.sql`-Kopf samt Aufraeum-Query dokumentiert; ein Rollback ueber diese
    Migration hinweg ist also nichts, was blind laufen darf. Beim lokalen Roundtrip
    ging er durch, weil dort keine Mehrfach-Entwuerfe lagen.
  - `finance_quotes.quote_number` und `finance_credit_notes.credit_note_number` tragen
    denselben `NOT NULL DEFAULT ''`-Musterfehler, haben aber keinen Unique-Index auf die
    Nummer (im Scope der Unit gegen `pg_indexes` geprueft) — sie sind nicht betroffen,
    aber falls dort je ein Unique-Index nachgezogen wird, muss er von Anfang an partiell
    sein.

## Iteration 73 — feat-vendor-access-audit-trail — done — 2026-08-23 08:13
- commit: <PENDING>
- gebaut: `SecurityGRPCServer.logVendorAccessEvent` (neuer schmaler Wrapper in
  `internal/server/security_grpc.go`, Vorlage `AuthGRPCServer.logPermissionEvent`)
  schreibt nach jedem erfolgreichen Approve/Decline/CounterPropose/Revoke einen
  `audit_log`-Eintrag (`vendor_access.approve` / `.decline` / `.counter_propose` /
  `.revoke`, target = Request-ID, target_type = `vendor_access_request`). Keine
  Wiring-Aenderung noetig: `auditService` und `vendorAccessService` waren in
  `NewSecurityGRPCServer` bereits beide vorhanden. Fuer Approve/Revoke kommt der
  Actor weiterhin aus dem expliziten `ActorId`-Feld (unveraendert); Decline und
  CounterPropose hatten nie ein solches Feld im Proto, daher lesen sie den
  Actor jetzt ueber `callerID(ctx)` aus dem per gRPC-Metadata propagierten
  Caller (`middleware.TenantInboundUnaryInterceptor`) und lehnen fehlenden
  Caller-Context mit dem Standard-Unauthenticated-Fehler ab, bevor der
  Service ueberhaupt aufgerufen wird — die Vendor-Access-Routen liegen alle
  hinter `authMiddleware`, das ist in Produktion also immer gegeben.
  `List` bleibt bewusst ungeloggt: reiner Lesezugriff auf die eigenen,
  bereits in der UI sichtbaren Requests des Tenants, kein Statuswechsel und
  kein neuer Datenzugriff im Sinne des Auftragsverarbeitungs-Nachweises —
  anders als bei den vier Statusaenderungen gibt es hier nichts zu belegen,
  das nicht schon aus den vier anderen Eintraegen hervorgeht.
  Neue Testdatei `internal/server/vendor_access_audit_events_db_test.go`
  (echtes SQL, `audit.NewPostgresRepository` + `vendoraccess.NewPostgresRepository`,
  Vorlage `rbac_audit_events_db_test.go`): acht Subtests, je ein Erfolgs- und
  ein Ablehnungsfall pro Aktion, Ablehnungsfaelle pruefen explizit "kein
  Eintrag". Bestehender Test `TestVendorAccessRPCs_HappyPathAndDomainErrors`
  (`security_grpc_test.go`) mit einem echten Caller im Kontext ausgestattet
  (`ctxWithTenantAndUser` statt `ctxWithTenant`), weil CounterPropose/Decline
  jetzt einen Caller brauchen — reiner Test-Fixture-Fix, keine Verhaltensaenderung
  an dem, was der Test pruefen sollte.
- gate: build ok (`go build -p 2 ./internal/security/vendoraccess/... ./internal/server/...`) |
  vet ok | lint ok (0 issues, beide Pakete) | test ok (`go test -count=1`
  `internal/security/vendoraccess` 14/14 und `internal/server` komplett gruen,
  **0 SKIP** bei `-v` gezaehlt in beiden Paketen, DATABASE_URL gegen `kmuhub_app`) |
  migration n.a. (keine Migration, keine neue Tabelle, keine Policy) |
  rls-smoke n.a. (kein Schema-/Policy-Eingriff, nur neue `audit_log`-Schreibpfade
  ueber den bereits RLS-geschuetzten bestehenden Insert-Pfad von `audit.Service.LogEvent`) |
  openapi n.a. (keine Route, kein Proto geaendert)
- coverage: `internal/server` 70,8 % -> 70,8 % (eigene Messung vor/nach per
  `git stash`/`stash pop`, gleiche Testdatei in beiden Laeufen). Unveraendert
  bei dieser Praezision ist plausibel: `internal/server` hat ueber 6.000
  ungedeckte Zeilen (CI-Stand), die gut ein Dutzend neuen Zeilen in
  `logVendorAccessEvent` plus vier Einzeiler-Aufrufe bewegen die
  Prozentzahl nicht sichtbar — bewiesen ist das Verhalten durch die acht
  neuen Subtests, nicht durch die Coverage-Zahl. `coverage_start` der Unit
  war ohnehin "n.a., Verhaltensaenderung".
- mutations-probe: `s.logVendorAccessEvent(...)`-Aufruf in
  `ApproveVendorAccessRequest` testweise auskommentiert -> exakt der Subtest
  `TestVendorAccessAuditEvents_DB/approve_writes_one_event` wurde rot
  (Event-Zaehler blieb bei 4 statt 5, alle anderen sieben Subtests blieben
  gruen). Danach zurueckgedreht, `git diff` auf `security_grpc.go` wieder
  sauber, voller Testlauf erneut gruen.
- verify vorgaenger: sauber. `ccee9d4f` (Iteration 72) geprueft: kein Gateway-Handler
  und keine gRPC-Umgehung (nur ein Index getauscht), kein Stub, keine `.proto`-Datei,
  kein neuer `RequirePermission`-Guard, keine neue Tabelle/RLS-Policy, Wire-Shape
  unveraendert, keine neue Route. `.up.sql` und `.down.sql` beide gefuellt und mit
  begruendeter Asymmetrie (Lockern kann nicht scheitern, Verschaerfen schon).
- neue-units: keine
- offen:
  - Die Entscheidung "List bleibt ungeloggt" ist im gebaut-Feld begruendet, aber
    eine reine Journal-Entscheidung dieser Iteration — falls Luke das anders sieht
    (z. B. weil ein Vendor beim blossen Ansehen der Liste schon personenbezogene
    Daten in Vorschau sieht), ist das eine Zeile nachzuziehen, keine grosse Aenderung.
  - `Service.Create` bleibt weiterhin ohne Audit-Eintrag (laut Kommentar "Not wired
    to an HTTP route" — kein produktiver Aufrufer, daher nicht mitgezogen). Sollte
    je ein Aufrufer dazukommen, braucht auch Create einen `logVendorAccessEvent`-Ruf.

## Iteration 74 — fix-datev-upload-log-stuck-uploading-no-reconciliation — done — 2026-08-23 08:24
- commit: 3cc5b79b
- gebaut: Reine Leseschicht-Kennzeichnung, keine Migration, kein neuer Status,
  kein Cron/Scheduler. `UploadService.ListUploadLogs` (`upload_service.go`)
  berechnet nach jedem Read `IsStale` pro Zeile ueber die neue reine Funktion
  `isUploadLogStale(status, startedAt, now)`: true nur fuer Status "uploading"
  aelter als `datevUploadStaleThreshold` (10 Minuten). Der Schwellwert ist aus
  der tatsaechlichen Retry-Konfiguration in `uploader.go` abgeleitet, nicht
  geraten: `datevMaxRetries=3` (4 Versuche) * neu benannte Konstante
  `datevUploadTimeout=60s` (vorher ein anonymes Literal im `http.Client`) +
  Backoff 1s+2s+4s = 247s (~4,1 Min) Worst-Case fuer eine echte Uebertragung —
  10 Minuten geben gut das Doppelte an Marge. Nichts wird zurueckgeschrieben,
  ein echter laufender Auftrag wird also nie markiert.
  `models.DatevUploadLog` traegt jetzt `IsStale bool` (explizit als "derived,
  never persisted" kommentiert). Das Feld ist bis zum Frontend durchgezogen,
  weil `ListUploadLogs` ueber eine echte gRPC-Response laeuft (kein Bypass):
  `.proto` (`DatevUploadLogEntry.is_stale`, Feld 9) geaendert und mit
  `protoc --go_out ... --go-grpc_out ...` (Makefile-Target `proto-biz`,
  `make` selbst war im Bash-Tool nicht auffindbar, daher der Protoc-Aufruf
  direkt) neu generiert, `datevUploadLogToProto`
  (`internal/server/datev_upload_grpc.go`) setzt `IsStale: l.IsStale`,
  `HandleListUploadLogs` (`internal/gateway/route_datev_upload.go`) mappt
  `is_stale` in die bestehende Bare-Array-Response (Wire-Shape unveraendert
  gelassen, nicht meine Baustelle), `backend/api/openapi.yaml` bekommt das
  Feld im selben Commit unter `DatevUploadLogEntry`.
  Neue Tests in `upload_service_test.go`: `TestIsUploadLogStale` (Tabelle,
  Grenzwerte knapp unter/ueber der Schwelle, alle Nicht-"uploading"-Status
  bleiben immer false) und
  `TestListUploadLogs_MarksOrphanedUploadingEntriesStale` (vier Zeilen:
  verwaist/uploading-frisch/failed-alt/completed-alt — nur die verwaiste
  wird markiert).
- gate: build ok (`go build -p 2` ueber datev/models/server/gateway/
  cmd/gateway) | vet ok (datev/models/server/gateway) | lint ok (0 issues,
  alle vier Pakete) | test ok (`go test -count=1` ./internal/biz/datev/...
  komplett gruen inkl. `TestTenantIsolation_Datev_DB`, DATABASE_URL gegen
  kmuhub_app, **0 SKIP**; ./internal/gateway/... und ./internal/server/...
  komplett gruen) | migration n.a. (keine neue Tabelle/Spalte, IsStale wird
  nie persistiert) | rls-smoke n.a. (kein Schema-/Policy-Eingriff) |
  openapi: `go test ./internal/gateway/ -run TestOpenAPIRouteDrift` gruen
  (836 Routen gegen 838 Pfade), zusaetzlich `npx swagger-cli validate
  backend/api/openapi.yaml` -> "is valid" | proto-regen: `datev_upload.pb.go`
  neu generiert, Diff zeigt ausschliesslich das neue `IsStale`-Feld plus den
  passenden Rawdesc-Bytes-Block, `biz.pb.go`/`biz_grpc.pb.go` liefen beim
  selben Aufruf mit durch (Makefile-Target buendelt alle vier .proto-Dateien),
  hatten aber laut `git diff --numstat` **keinen** inhaltlichen Unterschied
  (nur ein CRLF-Normalisierungs-Fehlalarm in `git status`, nach `stash`/`pop`
  bereits wieder verschwunden) — nicht mitcommittet, weil nichts zu committen
  war.
- coverage: `internal/biz/datev` 80,7 % -> 80,8 % (eigene Messung per
  `git stash`/`stash pop`, `-coverprofile` vor/nach). `coverage_start` der
  Unit war "n.a., Verhaltensaenderung" — die Zahl ist hier nur Beleg, kein
  Ziel.
- mutations-probe: Die `IsStale`-Zuweisungsschleife in `ListUploadLogs`
  testweise entfernt -> exakt `TestListUploadLogs_MarksOrphanedUploadingEntriesStale`
  wurde rot ("uploading entry older than the stale threshold must be flagged
  IsStale"), alle anderen Tests im Paket blieben gruen. Danach
  zurueckgedreht, `git diff upload_service.go` zeigt wieder nur den
  beabsichtigten Diff, `go test ./internal/biz/datev/...` erneut komplett
  gruen.
- verify vorgaenger: sauber. `0c217b5e` (Iteration 73) geprueft: kein
  Gateway-Handler ruft eine Service-Instanz direkt, kein Stub, keine
  `.proto`-Aenderung, kein neuer `RequirePermission`-Guard, keine neue
  Tabelle/Policy, Wire-Shape unveraendert (nur ein neuer `audit_log`-Schreib-
  pfad ueber den bestehenden RLS-geschuetzten Insert), keine neue Route, kein
  ersetzter Guard-Key.
- neue-units: keine
- offen:
  - Die Entscheidung "Leseschicht statt Migration/neuer Status" ist im
    gebaut-Feld begruendet; falls Luke stattdessen einen persistierten
    Status ("failed_orphaned") will, ist der CHECK-Constraint in
    `datev_upload_log` (`migrations/000056_add_lexware_datev_api.up.sql`)
    entsprechend zu erweitern — bewusst nicht in dieser Unit gemacht, siehe
    `notes:` der Unit ("Cron/Scheduler-Wiring gehoert Luke zur Freigabe").
  - Kein Frontend-seitiges Rendering des neuen `is_stale`-Felds gebaut — das
    ist reines Backend-Signal, ob und wie die DATEV-Upload-Log-Ansicht im
    Desktop-Client eine verwaiste Zeile anders darstellt, ist eine
    FE-Entscheidung.

## Iteration 75 — fix-datev-oauth-callback-503-breaks-redirect-contract — done — 2026-08-23 08:36
- commit: 9d7fc6f8
- gebaut: Ein-Zeilen-Fix in `HandleOAuthCallback` (`route_datev_upload.go:167`):
  die einzige Fehlerstelle im Handler, die noch `respondServiceUnavailable`
  (rohes 503-JSON) aufrief, ruft jetzt `redirectDatevError(w, r,
  "connection_failed")` — denselben Code-String, den Zeile 183 desselben
  Handlers fuer den selben Fehlermodus (biz-Service nicht erreichbar) bereits
  dokumentiert und verwendet. Diese Route ist die public OAuth-Callback-Route,
  die DATEV direkt im Browser ansteuert; laut `openapi.yaml` ist "immer ein
  302-Redirect, nie ein JSON-Fehlerkoerper" die einzige dokumentierte Garantie
  dieser Route. Kein Spec-Edit noetig, kein Verhaltenswechsel an den anderen
  sieben Handlern in derselben Datei (die sind normale JSON-APIs, dort ist
  `respondServiceUnavailable` weiterhin korrekt).
  Neuer Test `TestDatevHandleOAuthCallback_ServiceUnavailableRedirects` in
  `route_datev_upload_test.go`: `emptyRegistry()` simuliert die nicht
  erreichbare gRPC-Verbindung, erwartet 302 + `datev_error=connection_failed`
  statt 503.
- gate: build ok (`go build -p 2` gateway + cmd/gateway) | vet ok | lint ok
  (0 issues) | test ok (`go test -count=1 ./internal/gateway/` gruen, inkl.
  `TestOpenAPIRouteDrift` — 836 Routen gegen 838 Pfade, unveraendert, da keine
  Route/kein Response-Code hinzukam) | migration n.a. (kein Schema-Eingriff) |
  rls-smoke n.a. (keine Tabelle/Policy beruehrt)
- coverage: `internal/gateway` 56,6 % -> 56,6 % (eigene Messung per
  `git stash`/`stash pop`, `-coverprofile` vor/nach; Ein-Zeilen-Fix + ein Test
  bewegen die Paket-Prozentzahl nicht messbar ueber die Rundung hinaus).
  `coverage_start` der Unit ("54,1 %, CI-Stand 32570176303") ist ein aelterer
  Lauf-Gesamtwert, nicht meine eigene Vorher-Messung — siehe `offen:`.
- mutations-probe: Zeile 167 testweise zurueck auf
  `respondServiceUnavailable(w, dr.ServiceName())` -> exakt
  `TestDatevHandleOAuthCallback_ServiceUnavailableRedirects` wurde rot
  ("status = 503, want 302"), alle anderen Tests im Paket blieben unberuehrt.
  Danach zurueckgedreht, `git diff internal/gateway/route_datev_upload.go`
  zeigt nur die eine beabsichtigte Zeile.
- verify vorgaenger: sauber. `3cc5b79b` (Iteration 74) geprueft: kein
  Gateway-Handler ruft eine Service-Instanz direkt (Handler geht ueber
  `datevUploadLogToProto`/gRPC), kein Stub, `.proto` geaendert UND
  `datev_upload.pb.go` im selben Commit neu generiert (Diff zeigt nur das neue
  Feld + Rawdesc), kein neuer `RequirePermission`-Guard, keine neue Tabelle,
  Wire-Shape nur additiv erweitert (`is_stale` in der bestehenden
  Bare-Array-Response), keine neue Route, kein ersetzter Guard-Key.
- neue-units: keine
- offen:
  - `coverage_start` in der Unit-Definition war ein aggregierter CI-Wert aus
    einem frueheren Lauf, nicht paket-eigen zum Zeitpunkt dieser Iteration
    (aktueller eigener Vorher/Nachher-Messwert: 56,6 % / 56,6 %). Kuenftige
    Coverage-Units fuer `internal/gateway` sollten den Bezugswert aus einer
    frischen eigenen Messung ziehen, nicht aus `coverage_start:` uebernehmen.

## Iteration 76 — fix-invoice-recurring-grpc-status-mapping-doc-mismatch — done — 2026-08-23 08:36
- commit: 2a1b66fc
- gebaut: Zwei Stellen in `openapi.yaml` dokumentierten einen HTTP-Statuscode,
  den der jeweilige Handler strukturell nie erzeugen kann: PUT
  `/finance/invoices/{id}` dokumentierte `410` "Invoice is immutable"
  (`openapi.yaml:9181`), POST `/finance/recurring/{id}/generate`
  dokumentierte `412` "Schedule is paused, ended or past its end date"
  (`openapi.yaml:9580`). Beide Handler (`route_biz_invoices.go:HandleUpdateInvoice`,
  `route_biz_recurring.go:HandleGenerateRecurringInvoice`) geben gRPC-Fehler
  ausschliesslich ueber `respondGRPCError` -> `grpcStatusToHTTP` weiter
  (`helpers.go:59-63`), und dort mappt `FailedPrecondition` fest und mit
  explizitem Kommentar ("Not 410 Gone — the resource is not permanently
  removed") auf `409`.
  ENTSCHEIDUNG (wie in den Notes als "vermutlich lean" vorgeschlagen):
  Spec-Korrektur statt Code-Erweiterung. Begruendung: `grpcStatusToHTTP` ist
  ein zentraler, von sehr vielen Routen geteilter Mapper — ihn um zwei
  routenspezifische Sonderfaelle zu erweitern haette die Kopplung zwischen
  generischer Fehlerbehandlung und zwei Endpunkten erhoeht. Zusaetzlicher
  Praezedenzfall gefunden: `desktop/src/renderer/src/api/types.ts:27920`
  dokumentiert fuer eine andere Route bereits denselben Fix mit Kommentar
  "changed from 410 in the R3 hardening wave" — die Codebase hat dieses
  Muster (FailedPrecondition -> 409, nicht 410) also schon vorher etabliert.
  FE-Pruefung: `desktop/src/renderer/src/api/types.ts` ist auto-generiert aus
  `openapi.yaml` ("Do not make direct changes to the file"), kein echter
  FE-Vertrag. Grep nach `=== 410` / `=== 412` im gesamten `desktop/src` findet
  keinen Treffer — kein Frontend-Code verzweigt auf diese Codes. Die
  Spec-Korrektur ist also gefahrlos.
  Beide Routen hatten bereits einen `"409": {$ref: IdempotencyInFlight}`-Eintrag
  fuer denselben Statuscode — YAML erlaubt keinen doppelten Schluessel, also
  wurden die beiden Ursachen (FailedPrecondition + Idempotency-Key-in-flight)
  zu EINEM 409-Block zusammengefuehrt, nach dem bereits bestehenden Vorbild an
  `openapi.yaml:9052` (POST convert-quote-to-invoice, das exakt dasselbe
  Kombinationsmuster mit Retry-After-Hinweis "not set for the conflict above"
  verwendet). Vor dem Zusammenfuehren verifiziert, dass der
  Idempotency-Key-Teil fuer beide Routen tatsaechlich zutrifft: `BizRoutes`
  wird in `cmd/gateway/main.go:329` ueber `reg.RegisterRoutes(r,
  authWithIdempotency)` registriert, und `authWithIdempotency` (main.go:205)
  verkettet `middleware.Idempotency` vor jeden Handler — beide Routen laufen
  also durch die generische Idempotency-Middleware, der `409`-in-flight-Fall
  ist fuer sie real, keine Fehlinformation.
- gate: build ok (`go build -p 2` gateway + cmd/gateway) | vet ok | lint ok
  (0 issues) | test ok (`go test -count=1 ./internal/gateway/` gruen, inkl.
  `TestOpenAPIRouteDrift` — 836 Routen gegen 838 Pfade, unveraendert) |
  swagger-cli validate: "api/openapi.yaml is valid" | migration n.a. (reine
  Spec-Aenderung) | rls-smoke n.a. (keine Tabelle/Policy beruehrt)
- coverage: n.a. (reine OpenAPI-Spec-Korrektur, keine Codeaenderung —
  `internal/gateway` unveraendert bei 56,6 %, wie in Iteration 75 gemessen)
- mutations-probe: n.a. (kein Code-Verhalten geaendert, keine neue
  Testassertion, die einen Mutationstest rechtfertigen wuerde — die einzige
  Pruefung ist struktureller Natur: `swagger-cli validate` +
  `TestOpenAPIRouteDrift`, beide gruen)
- verify vorgaenger: sauber. `9d7fc6f8` (Iteration 75) geprueft: kein
  Gateway-Handler ruft eine Service-Instanz direkt (Redirect-Helper, kein
  gRPC-Bypass), kein Stub, keine `.proto`-Aenderung, kein neuer
  `RequirePermission`-Guard, keine neue Tabelle, Wire-Shape unveraendert (nur
  Redirect-Ziel gewechselt, keine JSON-Form), keine neue Route, kein
  ersetzter Guard-Key.
- neue-units: keine
- offen: keine

## Iteration 77 — fix-gateway-finance-list-routes-missing-error-response-docs — done — 2026-08-23 08:42
- commit: fe67d1b9
- gebaut: Drei Routen dokumentierten in `openapi.yaml` nicht alle Statuscodes, die
  ihr Handler tatsaechlich schreibt. GET `/finance/open-items`
  (`route_biz_open_items.go:42,49,54`) schreibt `500` bei drei Marshal-Fehlern
  (items/totals/buckets) — Spec dokumentierte nur 200/400/401/403/503, jetzt
  zusaetzlich `500`. GET `/finance/invoices` (`route_biz_invoices.go:100-107`
  invalid contact_id, `:114-121` invalid recurring_id, `:134` Marshal-Fehler,
  plus `respondServiceUnavailable`/`getTenantID` und `invoiceRead`-Permission-
  Middleware in `route_biz.go:58,112`) dokumentierte ausschliesslich `200` —
  jetzt zusaetzlich `400`/`401`/`403`/`500`/`503`. GET
  `/finance/datev/oauth/authorize` (`route_datev_upload.go:96,115` je `500`,
  plus `respondServiceUnavailable` Zeile 102) hatte den 500-Fall nur in der
  Freitext-Beschreibung erwaehnt, nicht im formalen `responses:`-Block, und
  503 fehlte komplett — beide jetzt ergaenzt. Fuer 401/403 an den ersten
  beiden Stellen sowie `open-items` 403 wurde verifiziert, dass sie bereits
  vorher korrekt dokumentiert waren (Handler-Code bzw. Permission-Middleware
  bestaetigt) — Scope-Notiz "Spec dokumentiert nur 200" war fuer open-items
  bereits durch eine fruehere Iteration ueberholt, fuer /finance/invoices
  aber weiterhin zutreffend. Keine bestehende `500`-Response-Komponente im
  Spec gefunden (grep ueber die gesamte Datei) — als literale Inline-
  Beschreibung ergaenzt, im gleichen Stil wie die Nachbar-503-Eintraege.
- gate: build ok (`go build -p 2` gateway + cmd/gateway) | vet ok | lint ok
  (0 issues) | test ok (`go test -count=1 ./internal/gateway/` gruen, inkl.
  `TestOpenAPIRouteDrift` — 836 Routen gegen 838 Pfade, unveraendert, 0
  uebersprungene Tests) | swagger-cli validate: "api/openapi.yaml is valid" |
  migration n.a. (reine Spec-Aenderung) | rls-smoke n.a. (keine
  Tabelle/Policy beruehrt)
- coverage: n.a. (reine OpenAPI-Spec-Korrektur, keine Codeaenderung —
  `internal/gateway` unveraendert)
- mutations-probe: n.a. (kein Code-Verhalten geaendert, keine neue
  Testassertion — die Pruefung ist strukturell: swagger-cli validate +
  TestOpenAPIRouteDrift, beide gruen)
- verify vorgaenger: sauber. `2a1b66fc` (Iteration 76) geprueft: nur
  `openapi.yaml` im Diff, keine Codeaenderung, kein Gateway-Handler-Bypass,
  kein Stub, keine `.proto`-Aenderung, kein neuer `RequirePermission`-Guard,
  keine neue Tabelle, keine neue Route, kein ersetzter Guard-Key.
- neue-units: keine
- offen: keine

## Iteration 78 — fix-gateway-billing-recurring-routes-missing-401-503-docs — done — 2026-08-23 08:45
- commit: 136e9f52
- gebaut: `route_biz_billing.go` (24 Handler) und `route_biz_recurring.go` (8
  Handler) schreiben in praktisch jedem Handler literal 401 (`getTenantID`),
  503 (`respondServiceUnavailable`) und laufen zusaetzlich durch
  `RequirePermission`/`RequirePermissionAny`-Middleware (403) — die
  `responses:`-Bloecke in `openapi.yaml` dokumentierten das fuer beide
  Dateien fast durchgaengig nicht, teils nur den Erfolgscode. Per
  Explore-Agent alle 32 Routen einzeln gegen Handler-Code (Zeilennummern)
  und bestehende Spec-Blocke geprueft, danach 401/403/503 im Stil von
  `route_biz_expenses.go` (`"401": { $ref: Unauthorized }`, `"403": { $ref:
  Forbidden }`, `"503": { description: Finance service unavailable }` —
  kompakte Form ohne content/schema, wie zuletzt in Iteration 77 fuer
  denselben Datei-Block etabliert) ergaenzt: credit-notes (4 Routen),
  invoices/{id}/payments (2), recurring (8), payments/{id} (1), dunning (6),
  dashboard (1), export/datev (1) — alle 401+403+503 neu. Fuer den in der
  Unit-Notiz benannten Ausnahmeblock (`route_biz_billing.go:649-900`) wurde
  die Behauptung "400/401 schon dokumentiert, nur 503 fehlt" einzeln
  verifiziert: traf zu fuer `journal/summary`, `stats/payments` (beide
  zusaetzlich ohne 403 — ergaenzt), `dunning/{id}/status`,
  `dunning/{id}/notice`, `export/gobd` (503 ergaenzt, 401/403/400 bereits
  vorhanden); `invoices/validate-number` ebenfalls ohne 403 (ergaenzt);
  `invoices/{id}/lock` hatte 401/403/404 aber kein 503 (ergaenzt, 400 fuer
  die dortige `validateUUIDParam` bleibt offen, siehe unten).
  BEWUSST NICHT gefixt (ausserhalb des 401/403/503-Scopes dieser Unit, echte
  Verhaltensdoku-Luecken statt Guard-Codes): `HandleLockInvoice` schreibt
  literal 400 bei ungueltiger UUID (`route_biz_billing.go:727`) — Spec
  dokumentiert 400 dort gar nicht; `HandleSendDunningNotice` schreibt 400
  (uuid, `:834`) UND 500 (Marshal-Fehler, `:850`) — Spec dokumentiert keins
  von beiden; `HandleListCreditNotes`, `HandleListPayments` (invoice-scoped),
  `HandleListDunnings` schreiben je einen 500 bei `hrMarshalSlice`-Fehlern,
  ebenfalls undokumentiert; `HandleRecordPayment` hat einen bereits
  bestehenden `"400": { $ref: IdempotencyKeyRequired }`-Eintrag, der NICHT
  den real geschriebenen Validierungs-400 (ungueltige Payment-UUID) abdeckt
  — YAML erlaubt keinen zweiten `400`-Schluessel, eine Zusammenfuehrung
  waere ein eigener Fund gewesen, kein mechanischer 401/403/503-Fix. Diese
  fuenf Luecken sind jetzt als eigene Unit am Backlog-Ende erfasst (siehe
  neue-units).
- gate: build ok (`go build -p 2` gateway + cmd/gateway) | vet ok | lint ok
  (0 issues) | test ok (`go test -count=1 ./internal/gateway/` gruen, inkl.
  `TestOpenAPIRouteDrift` — 836 Routen gegen 838 Pfade, unveraendert, 0
  uebersprungene Tests) | swagger-cli validate: "api/openapi.yaml is valid"
  | migration n.a. (reine Spec-Aenderung) | rls-smoke n.a. (keine
  Tabelle/Policy beruehrt)
- coverage: n.a. (reine OpenAPI-Spec-Korrektur, keine Codeaenderung —
  `internal/gateway` unveraendert)
- mutations-probe: n.a. (kein Code-Verhalten geaendert, keine neue
  Testassertion — die Pruefung ist strukturell: swagger-cli validate +
  TestOpenAPIRouteDrift, beide gruen)
- verify vorgaenger: sauber. `fe67d1b9` (Iteration 77) geprueft: nur
  `openapi.yaml` (plus Loop-Metadateien) im Diff, keine Codeaenderung, kein
  Gateway-Handler-Bypass, kein Stub, keine `.proto`-Aenderung, kein neuer
  `RequirePermission`-Guard, keine neue Tabelle, keine neue Route, kein
  ersetzter Guard-Key.
- neue-units: fix-gateway-billing-lock-dunning-notice-listroutes-missing-400-500-docs
  (ans Backlog-Ende gehaengt — deckt die fuenf oben genannten 400/500-Luecken
  plus den 400-Schluessel-Konflikt bei `HandleRecordPayment` ab)
- offen: keine

## Iteration 79 — feat-retention-worker-handler-auth-invitations — done — 2026-08-23 08:56
- commit: 0913ebf7
- gebaut: `internal/security/gdpr/retention_invitations.go` — neunter Handler
  auf der Retention-Engine (`InvitationRetentionHandler`, resource_type
  "invitations"). Anders als die acht bestehenden Handler hat eine Einladung
  zwei getrennte Alterszustaende, die beide aufgeraeumt gehoeren: PENDING
  (`accepted_at IS NULL`) ist ab `expires_at` reiner Datenmuell (niemand kann
  sie mehr annehmen), ACCEPTED (`accepted_at IS NOT NULL`) hat ihren Zweck
  erfuellt, sobald der User existiert. Entscheidung (im Datei-Header
  begruendet): EIN `resource_type` "invitations" mit einer Plan-Query, die
  beide Faelle per OR gegen denselben Cutoff prueft (`expires_at < cutoff`
  fuer pending, `accepted_at < cutoff` fuer accepted) — nicht zwei getrennte
  resource_types, weil beide Zustaende dieselbe Retention-Days-Policy und
  dieselbe operator-sichtbare Tabelle teilen; `DateColumn()` legt beide Uhren
  offen. Delete-only (kein Anonymize) wie bei Notifications: `token_hash` ist
  bereits ein SHA-256-Pseudonym, Anonymisieren einer zwecklosen Einladung
  bringt nichts. In `cmd/auth/main.go` als neunter Eintrag in
  `retentionRegistry` registriert (Vorlage: `NewNotificationRetentionHandler`
  aus `retention_notifications.go`, kleinster der drei existierenden
  Delete-only-Handler). `retention_invitations_test.go` deckt: SupportsAction
  (nur Delete), Plan trifft pending-expired UND accepted-past-cutoff, Plan
  schliesst frische Zeilen in beiden Zustaenden aus, Tenant-Isolation (fremder
  Tenant leakt nicht ins Plan), Apply-Delete idempotent (zweiter Lauf 0
  betroffen), Apply lehnt Anonymize ab. Keine Migration noetig — `tenant_id`
  + RLS-Policy existieren bereits seit Migration 000249, Migrationskopf war
  vor und nach dieser Unit bei 323.
- gate: build ok (`go build -p 2` gdpr + auth + cmd/auth) | vet ok | lint ok
  (0 issues, beide Pakete) | test ok (`go test -count=1 ./internal/security/gdpr/...`
  und `./internal/auth/...` gruen, 0 uebersprungene Tests bei gesetztem
  `DATABASE_URL=...kmuhub_app...`) | migration n.a. (keine neue Tabelle/Spalte,
  Migrationskopf 323 unveraendert) | rls-smoke n.a. (keine neue Tabelle/Policy
  — bestehende RLS-Policy auf `invitations` genutzt, Tenant-Isolation ueber
  den Plan-Test mit zweitem Tenant statt separatem RLS-Smoke-Kommando belegt,
  da kein Schema/Policy-Diff vorliegt) | Route: keine (interner Handler,
  `go test ./internal/gateway/` daher nicht Pflicht — nicht gefahren)
- coverage: internal/security/gdpr 72,1 % -> 72,2 % (isoliert gemessen: neue
  Dateien kurz beiseite verschoben, Baseline-Lauf, dann zurueckgeholt — die
  1 gemessene NotificationsRetentionHandler-Nachbarschaft im Paket ist schon
  sehr dicht getestet, daher kleiner Sprung trotz vollstaendig neuem Handler)
- mutations-probe: `Plan`-Query mutiert (`accepted_at < $2` -> `accepted_at >
  $2`, ACCEPTED-Zweig invertiert), `TestInvitationRetentionHandler_
  PlanMatchesBothAgeStatesPastCutoffAndIsTenantScoped` wurde rot
  (assert.NotContains schlug fehl, frische accepted-Einladung erschien im
  Plan), Mutation zurueckgedreht, Diff gegen Original per `diff` verifiziert
  sauber
- verify vorgaenger: sauber. `136e9f52` (Iteration 78) geprueft: nur
  `openapi.yaml` (117 Zeilen) plus Loop-Metadateien im Diff, keine
  Go-Datei angefasst — die acht Fehlerklassen greifen ausschliesslich an
  Code, hier gibt es keinen. Stichprobe der neuen 401/403/503-Bloecke gegen
  bestehende Vorlage (`route_biz_expenses.go`-Stil) zeigt saubere,
  konsistente Ergaenzungen ohne inhaltliche Abweichung.
- neue-units: keine
- offen: keine

## Iteration 80 — fix-openapi-idempotency-doc-wrong-production-default — done — 2026-08-23 09:02
- commit: 9d5028b3
- gebaut: `backend/api/openapi.yaml` an zwei Stellen korrigiert
  (`info.description` und die `IdempotencyKeyRequired`-Response-Beschreibung).
  Beide behaupteten bisher, der Produktions-Default fuer `IDEMPOTENCY_MODE`
  sei `WarnMode` und ein fehlender Idempotency-Key werde in Produktion
  stillschweigend durchgelassen. Verifiziert: die dev- und prod-Compose-Datei
  setzen `IDEMPOTENCY_MODE: hard` als literalen Wert (kein `${VAR:-default}`)
  — identisch in beiden Stacks. Der Go-Code selbst (`cmd/gateway/main.go:193-202`)
  hat zwar `WarnMode` als Default-Variable, wird aber in jeder ausgelieferten
  Compose-Datei hart auf `hard` ueberschrieben. Beide Stellen jetzt so
  formuliert, dass sie den tatsaechlichen Stand nennen: Go-Default ist
  WarnMode, aber jede ausgelieferte Compose-Datei (dev UND prod) setzt
  IDEMPOTENCY_MODE=hard, also ist der fehlende-Key-400 der reale Zustand
  in jeder betriebenen Umgebung, kein Opt-in-Sonderfall.
- gate: build n.a. (reine Spec-Aenderung, kein Go-Code) | vet n.a. | lint n.a.
  | test ok (`go test -count=1 ./internal/gateway/` gruen, inkl.
  `TestOpenAPIRouteDrift`) | swagger-cli validate: "backend/api/openapi.yaml
  is valid" | migration n.a. | rls-smoke n.a.
- coverage: n.a. (reine OpenAPI-Spec-Korrektur, keine Codeaenderung)
- mutations-probe: n.a. (kein Code-Verhalten geaendert, keine neue
  Testassertion — Pruefung ist strukturell: swagger-cli validate + go test
  ./internal/gateway/, beide gruen)
- verify vorgaenger: sauber. `0913ebf7` (Iteration 79, neunter
  Retention-Handler fuer invitations) geprueft: Registrierung in
  `cmd/auth/main.go` konsistent mit den acht bestehenden Handlern (gleicher
  Konstruktor-Stil), `Plan`/`Apply` tenant-gescoped (`WHERE tenant_id = $1`),
  kein neuer `RequirePermission`-Guard, keine neue Route, keine neue Tabelle
  (bestehende RLS-Policy auf `invitations` seit Migration 000249 genutzt),
  kein Gateway-Handler-Bypass (interner Retention-Handler, kein Route-Pfad),
  kein Stub, keine `.proto`-Aenderung, kein ersetzter Guard-Key.
- neue-units: keine
- offen: keine

## Iteration 81 — fix-dashboard-metrics-blind-currency-sum — done — 2026-08-23 09:05
- commit: 0ce7935a
- gebaut: `internal/biz/dashboard/postgres_repository.go` — Query 1 (Revenue aus
  `finance_invoices`) und Query 2 (`avg_deal_size` aus `finance_quotes`)
  summieren/mitteln jetzt nur noch Zeilen, deren `currency` mit dem
  aufgeloesten Tenant-Default uebereinstimmt: `COALESCE((SELECT
  default_currency FROM company_settings WHERE tenant_id = $1), 'EUR')`.
  Bewusst NICHT hart auf `'EUR'` gefiltert — `internal/biz/invoice/service.go:195-197`
  und `internal/biz/quote/service.go:159-161` setzen die Waehrung neuer
  Ad-hoc-Dokumente aus `company_settings.default_currency`, faellt das nur
  auf `models.DefaultCurrency` (EUR) zurueck, wenn kein Settings-Datensatz
  existiert oder das Feld leer ist. Ein hartkodiertes `currency = 'EUR'`
  haette also jeden Tenant mit z. B. CHF als Default komplett leerlaufen
  lassen (schlimmer als der Ausgangsbug). `quotesPending`/`quotesTotal`/
  `quotesAccepted` bleiben bewusst waehrungs-agnostische Zaehlungen (kein
  Summen-/Durchschnittswert) — nur `avg_deal_size` bekam den Filter. Query 3
  (Status-Breakdown, reine Counts) unveraendert, war nicht Teil des Funds.
  Kein Wire-Shape-Wechsel (weiterhin einzelne Decimal-Werte, keine
  Currency-Map) — laut Notes der Unit bewusst vermieden, da
  `RevenueMetrics`/`PipelineMetrics` FE-Vertrag sind; `lean:`-Marker im Code
  mit Upgrade-Trigger "wenn ein Tenant tatsaechlich ausserhalb seiner
  Default-Waehrung bucht und die Einzelkennzahl nicht mehr ehrlich ist".
  Zwei neue DB-Tests belegen genau diese Entscheidung: einmal impliziter
  EUR-Default (kein `company_settings`-Datensatz) mit einer CHF-Rechnung,
  die ausgeschlossen bleibt, und einmal ein Tenant mit
  `default_currency = 'CHF'`, dessen CHF-Rechnung zaehlt und dessen
  EUR-Ausreisser-Rechnung ausgeschlossen wird — letzterer Test haette bei
  einer hartkodierten `'EUR'`-Loesung nicht bestanden. Ein dritter Test
  deckt Query 2 (CHF-Angebot faellt nicht in den EUR-Durchschnitt,
  `QuotesPending`-Zaehler bleibt waehrungsunabhaengig unveraendert).
- gate: build ok (`go build -p 2` dashboard + gateway + cmd/gateway) | vet ok
  | lint ok (0 issues) | test ok (`go test -count=1 ./internal/biz/dashboard/...`
  gruen, 14 Tests, 0 uebersprungen bei gesetztem `DATABASE_URL=...kmuhub_app...`)
  | migration n.a. (keine Schema-Aenderung, `currency`/`default_currency`
  existieren seit Migration 000216) | rls-smoke n.a. (keine neue
  Tabelle/Policy, bestehende Tenant-Filter auf `finance_invoices`/
  `finance_quotes`/`company_settings` genutzt) | Route: keine (keine
  Gateway-Route/OpenAPI-Aenderung, `go test ./internal/gateway/` daher nicht
  Pflicht — nur als Teil des Builds mitgefahren)
- coverage: internal/biz/dashboard 79,3 % -> 79,3 % (Paket-Gesamtwert,
  vor UND nach der Aenderung identisch gemessen per `git stash` auf
  `postgres_repository.go`+`_db_test.go` — der Fix aendert nur SQL-String-
  Literale innerhalb bestehender Statements, keine neuen Go-Verzweigungen,
  daher bewegt sich die Zeilen-Coverage-Prozentzahl nicht; die eigentliche
  Beweisführung liegt in der Mutations-Probe, nicht in der Coverage-Zahl)
- mutations-probe: `AND currency = COALESCE(...)`-Zeile aus Query 1
  entfernt (Query 2 unveraendert gelassen), `go test -run
  "TestGetDashboardMetrics_ForeignCurrencyInvoiceExcludedFromRevenueSum|
  TestGetDashboardMetrics_UsesTenantDefaultCurrencyNotHardcodedEUR"` wurde
  rot (TotalInvoiced 6000 statt 1000 bzw. 11999 statt 2000 — die
  Fremdwaehrungs-/Ausreisser-Betraege wurden wieder blind mitsummiert),
  Mutation zurueckgedreht, `diff` gegen vor-Mutation-Kopie sauber
- verify vorgaenger: sauber. `9d5028b3` (Iteration 80) geprueft: reine
  `openapi.yaml`-Textkorrektur an zwei Stellen (info.description +
  IdempotencyKeyRequired-Response), kein Go-Code veraendert — die acht
  Fehlerklassen greifen an Code, hier nicht einschlaegig. `swagger-cli
  validate` laut Journal-Eintrag gruen, `go test ./internal/gateway/`
  ebenfalls; Stichprobe der beiden geaenderten Textstellen gegen die
  belegten Fakten (docker-compose.yml/prod.yml setzen `IDEMPOTENCY_MODE:
  hard` literal) bestaetigt die Korrektur.
- neue-units: keine
- offen: keine

## Iteration 82 — fix-berichte-kpi-blind-currency-sum — done — 2026-08-23 09:10
- commit: 119e2a3a
- gebaut: `internal/berichte/downstream/kpi_postgres.go` — `kpiSnapshotQuery` und
  `kpiSeriesQuery` filtern die `SUM(gross_total) FROM finance_invoices`- und
  `SUM(value) FROM deals`-Subqueries jetzt auf
  `currency = COALESCE((SELECT default_currency FROM company_settings WHERE
  tenant_id = $1), 'EUR')` — exakt dieselbe Filterentscheidung wie in
  fix-dashboard-metrics-blind-currency-sum (Iteration 81), damit Dashboard-
  und Berichte-KPI fuer denselben Tenant dieselbe Zahl zeigen. Beide Sub-
  Selects (revenue und pipeline_volume) sind betroffen, in beiden Queries.
  `internal/berichte/executor/executor.go` — Kommentar oberhalb von
  `revenueByMonth`/`invoicesOpen`/`pipeline` (aktuell tot, `e.finance`/
  `e.crm` sind laut `cmd/berichte/main.go:54-57` nil) markiert dasselbe
  ungefixte Muster (`total += r.Revenue` etc. ohne Currency-Check), damit die
  kuenftige Verdrahtung nicht denselben Fehler erbt — laut Unit-Notes bewusst
  NICHT mitgefixt, da toter Pfad. `kpi_postgres_test.go` — zwei neue Snapshot-
  Tests (`TestKPISnapshot_ForeignCurrencyInvoiceAndDealExcluded`,
  `TestKPISnapshot_UsesTenantDefaultCurrencyNotHardcodedEUR`) und ein neuer
  Series-Test (`TestKPISeries_ForeignCurrencyExcludedFromBucket`) mit
  gemischten Waehrungen (EUR+CHF gleichzeitig fuer Invoice UND Deal), belegen
  sowohl impliziten EUR-Default als auch expliziten CHF-Tenant-Default.
  Nebenfund beim Testen: sechs bestehende Testfunktionen registrierten
  `defer pool.Close()` UND `t.Cleanup(...)`-Row-Cleanup im selben Test — Go
  fuehrt Defers vor t.Cleanup aus, der Pool war beim Cleanup-Aufruf also
  schon zu, `testutil.CleanupRow` loggt den Fehler nur (`t.Logf`, faellt nie)
  → Fixture-Zeilen blieben liegen. Lokale DB hatte fuer die zwei
  kpi-Test-Tenants (cccc0000-...-0001/-0002) 78 verwaiste finance_invoices,
  78 deals, 72 tickets, 50 stock_warnings/inventory_items und eine
  company_settings-Zeile (default_currency=CHF) angesammelt — genau diese
  CHF-Altzeile hat meinen ersten Testlauf mit falschem Vorzeichen verfaelscht
  (Revenue-Delta 0 statt 1000). Fix: alle sechs `defer pool.Close()` in
  dieser Datei zu `t.Cleanup(func() { pool.Close() })` gemacht (LIFO ⇒
  Row-Cleanups laufen jetzt vor dem Pool-Close), lokale DB fuer die zwei
  betroffenen Test-Tenant-UUIDs manuell bereinigt (`DELETE ... WHERE
  tenant_id IN (...)` fuer alle sieben betroffenen Tabellen). Gleiches
  Doppelmuster in 11 weiteren Testdateien gefunden, nicht gefixt (siehe
  neue-units).
- gate: build ok (`go build -p 2` berichte + gateway + cmd/berichte +
  cmd/gateway) | vet ok | lint ok (0 issues,
  `golangci-lint ./internal/berichte/...`) | test ok
  (`go test -count=1 ./internal/berichte/...` gruen, alle 6 Unterpakete,
  inkl. downstream mit 6/6 gruenen Tests bei gesetztem
  `DATABASE_URL=...kmuhub_app...`, 0 uebersprungen) | migration n.a. (keine
  Schema-Aenderung, `currency`/`default_currency` existieren seit Migration
  000009 bzw. 000216) | rls-smoke n.a. (keine neue Tabelle/Policy, bestehende
  Tenant-Filter auf `finance_invoices`/`deals`/`company_settings` genutzt) |
  Route: keine (keine Gateway-Route/OpenAPI-Aenderung,
  `go test ./internal/gateway/` daher nicht Pflicht, aber im Build
  mitgefahren)
- coverage: internal/berichte/downstream 77,8 % -> 77,8 % (Paketwert vor UND
  nach der Aenderung identisch gemessen per `git stash`/`stash pop` auf
  `kpi_postgres.go`+`kpi_postgres_test.go`+`executor.go` — der Fix aendert
  nur SQL-String-Literale innerhalb bestehender Statements, keine neuen
  Go-Verzweigungen; Beweisfuehrung liegt in der Mutations-Probe, nicht in der
  Coverage-Zahl. `coverage_start` der Unit nannte nur "Paketwert im CI-Log
  nachschlagen" ohne Zahl — 77,8 % ist die selbst gemessene Referenz)
- mutations-probe: `AND currency = COALESCE(...)`-Zeile aus dem
  `finance_invoices`-Subquery in `kpiSnapshotQuery` entfernt (deals-Zeile
  unveraendert gelassen), `go test -run
  "TestKPISnapshot_ForeignCurrencyInvoiceAndDealExcluded|
  TestKPISnapshot_UsesTenantDefaultCurrencyNotHardcodedEUR"` wurde rot
  (revenue delta 10110 statt 111 bzw. 8110 statt 333 — die
  Fremdwaehrungs-/Ausreisser-Betraege wurden wieder blind mitsummiert),
  Mutation zurueckgedreht, `diff` gegen vor-Mutation-Kopie identisch
- verify vorgaenger: sauber. `a0c07416` geprueft: reine
  Journal-Textkorrektur (1 Zeile, Commit-SHA-Nachtrag fuer Iteration 81),
  kein Go-Code veraendert — die acht Fehlerklassen greifen an Code, hier
  nicht einschlaegig.
- neue-units: fix-db-test-cleanup-order-leaks-fixtures (elf weitere
  Testdateien mit demselben `defer pool.Close()` + `t.Cleanup`-Doppelmuster,
  Liste und Fix-Vorschlag in der Unit)
- offen: keine

## Iteration 83 — fix-gobd-export-missing-currency-column — done — 2026-08-23 09:18
- commit: ac6bf34a
- gebaut: `GoBDExportRow` (`internal/biz/dunning/service_gobd.go`) traegt jetzt
  ein `Currency`-Feld; `buildGoBDCSV` schreibt eine zusaetzliche `Waehrung`-
  Spalte direkt nach `Bruttobetrag` (Header + Datenzeilen, Reihenfolge-
  Kommentar aktualisiert). `BuildGoBDRows` (`gobd_rows.go`) nimmt jetzt einen
  `currency string`-Parameter und setzt ihn auf beiden Pfaden (Rate-Gruppen-
  Schleife UND den Reverse-Charge/Kleinunternehmer-Einzelzeilen-Pfad).
  `GenerateGoBDExport` (`internal/server/biz_grpc.go:2495,2517`) reicht
  `inv.Currency` bzw. `cn.Currency` durch — beide Felder waren bereits
  gescannt (`invoiceColumns`/Credit-Note-Select enthalten `currency`), nur
  bislang ungenutzt fuer den GoBD-Export. Gruppierungslogik selbst (Rate-Key,
  SKR03-Konto, BU-Schluessel) unveraendert, wie von der Unit gefordert.
  Tests: `TestBuildGoBDRows_CarriesDocumentCurrency` (neu, gobd_rows_test.go)
  belegt EUR/CHF auf beiden Zeilenpfaden inkl. Exempt-Pfad;
  `TestService_GenerateGoBDExport_CurrencyColumn` (neu, service_test.go)
  belegt auf CSV-Ebene, dass zwei Zeilen mit EUR bzw. CHF ihre eigene Waehrung
  in der `Waehrung`-Spalte behalten (per Header-Index nachgeschlagen, nicht
  hartkodiert). Alle sieben bestehenden `BuildGoBDRows`-Aufrufe in
  gobd_rows_test.go auf die neue Signatur (zusaetzliches `"EUR"`-Argument)
  angepasst, Verhalten unveraendert.
- gate: build ok (`go build -p 2` dunning + server + gateway + cmd/biz +
  cmd/gateway) | vet ok | lint ok (0 issues,
  `golangci-lint ./internal/biz/dunning/... ./internal/server/... ./internal/gateway/...`)
  | test ok (`go test -count=1 ./internal/biz/dunning/...` gruen,
  `go test -count=1 ./internal/server/...` gruen, beide mit gesetztem
  `DATABASE_URL=...kmuhub_app...`, 0 uebersprungen) | migration n.a. (keine
  Schema-Aenderung, `currency` existiert bereits auf `finance_invoices` und
  `finance_credit_notes`) | rls-smoke n.a. (keine neue Tabelle/Policy) |
  Route: keine (keine Gateway-Route/OpenAPI-Aenderung,
  `go test ./internal/gateway/` daher nicht Pflicht — trotzdem nicht
  gesondert gelaufen, da `internal/server`-Aenderung nur einen bestehenden
  internen Funktionsaufruf betrifft, keine Route)
- coverage: internal/biz/dunning 92,2 % -> 92,2 % (vorher/nachher selbst
  gemessen per `git stash`/`stash pop` auf allen fuenf geaenderten Dateien;
  unveraendert, weil die neuen Zeilen reine Feldzuweisungen ohne neue
  Verzweigung sind — Beweisfuehrung liegt in der Mutations-Probe)
- mutations-probe: in `buildGoBDCSV` `r.Currency` durch hartkodiertes `"EUR"`
  ersetzt, `go test -run TestService_GenerateGoBDExport_CurrencyColumn` wurde
  rot (zweite Zeile erwartete CHF, bekam EUR), Mutation zurueckgedreht,
  `git diff` gegen Vor-Mutation-Stand leer
- verify vorgaenger: sauber. `119e2a3a` geprueft (`git show --stat`, dann
  gezielt `kpi_postgres.go`+`executor.go` gegen die acht Fehlerklassen): reine
  SQL-Filterzeilen (`AND currency = COALESCE(...)`) in zwei bestehenden
  Statements + ein reiner Kommentar im toten `executor.go`-Pfad. Kein
  gRPC-Bypass, kein Stub, kein `.proto`, kein neuer `RequirePermission`, keine
  neue Tabelle, kein Wire-Shape-Wechsel, keine neue Route, kein
  Guard-Ersatz — keine der acht Klassen einschlaegig.
- neue-units: keine
- offen: keine

## Iteration 84 — fix-gobd-export-currency-empty-for-legacy-documents — done — 2026-08-23 09:26
- commit: 3fe6b86d
- gebaut: `GenerateGoBDExport` (`internal/server/biz_grpc.go:2495,2517`) reicht `inv.Currency`
  bzw. `cn.Currency` jetzt durch `documentCurrency(...)` statt roh an
  `dunning.BuildGoBDRows` — exakt das Muster der drei bestehenden Call-Sites
  (`:1500,1537,1604`). Legacy-Rechnungen/-Gutschriften ohne gespeicherte Currency
  (leerer String, siehe Kommentar an `documentCurrency`) tragen dadurch "EUR" statt ""
  in der Waehrung-Spalte der GoBD-CSV. Neuer Test `TestDocumentCurrency`
  (`biz_grpc_amounts_test.go`) pinnt den Helper direkt (leer -> EUR, "CHF" -> "CHF"),
  weil ein Handler-Test fuer `GenerateGoBDExport` architektonisch nicht moeglich ist
  (siehe Kommentarblock ueber `TestGenerateGoBDExport_Validation`: invoiceService/
  creditNoteService sind konkrete Struct-Felder, kein Interface, heavyweight
  Konstruktion noetig — bereits fuer andere Handler als out-of-scope markiert).
- gate: build ok (`go build -p 2` server + dunning + gateway + cmd/biz + cmd/gateway) |
  vet ok | lint ok (0 issues, server + dunning) | test ok (`go test -count=1
  ./internal/server/...` gruen inkl. `internal/server/response`, `go test -count=1
  ./internal/biz/dunning/...` gruen, `go test -count=1 ./internal/gateway/` gruen —
  alle mit gesetztem `DATABASE_URL=...kmuhub_app...`, 0 uebersprungen) | migration n.a.
  (keine Schema-Aenderung) | rls-smoke n.a. (keine neue Tabelle/Policy) | Route: keine
  (keine Gateway-Route/OpenAPI-Aenderung, `go test ./internal/gateway/` trotzdem
  gefahren, weil `internal/server` angefasst wurde — 1× ist er dabei mit
  `TestDecodeBexioState_ManipulatedSignature` flakig rot gelaufen, unabhaengig von
  dieser Aenderung, siehe `verify vorgaenger` unten und `neue-units`)
- coverage: internal/server 70,8 % -> 70,8 % (vorher per `git stash`/`stash pop` selbst
  gemessen; unveraendert, weil `documentCurrency` als Funktion bereits existierte und
  vollstaendig abgedeckt war — die Aenderung fuegt an den beiden GoBD-Call-Sites nur
  einen bereits getesteten Funktionsaufruf ein, keine neue Verzweigung.
  Beweisfuehrung liegt in der Mutations-Probe)
- mutations-probe: `documentCurrency` auf `return stored` gekuerzt (EUR-Fallback
  entfernt), `go test -run TestDocumentCurrency ./internal/server/` wurde rot (erwartet
  "EUR", bekam ""), Mutation zurueckgedreht, `git diff --stat biz_grpc.go` zeigt danach
  wieder nur die zwei beabsichtigten Call-Site-Zeilen
- verify vorgaenger: Fund. `ac6bf34a` (Iteration 83, fix-gobd-export-missing-currency-column)
  geprueft (`git show --stat` + gezielt `biz_grpc.go`/`gobd_rows.go`/`service_gobd.go`
  gegen die acht Fehlerklassen): kein gRPC-Bypass, kein Stub, kein `.proto`, kein neuer
  `RequirePermission`, keine neue Tabelle, kein Route-Verstoss, kein Guard-Ersatz — ABER
  ein neunter, nicht in der Liste stehender Fund: die beiden neuen `BuildGoBDRows`-Aufrufe
  in `GenerateGoBDExport` reichen `inv.Currency`/`cn.Currency` roh durch, statt sie wie die
  drei bestehenden Call-Sites im selben File durch `documentCurrency()` zu schicken —
  Legacy-Dokumente ohne gespeicherte Currency haetten eine leere statt "EUR"
  Waehrung-Spalte im GoBD-Export bekommen. Als fix-gobd-export-currency-empty-for-legacy-documents
  ganz vorne im Backlog angelegt und in dieser Iteration selbst abgearbeitet (Schritt 1
  der Anleitung).
- neue-units: fix-bexio-state-test-flaky-tamper-byte (inzidentell beim Pflicht-Gateway-Gate
  gefunden: `TestDecodeBexioState_ManipulatedSignature` ersetzt das erste
  Signatur-Zeichen durch das feste Literal "X" — trifft der zufallsabhaengige Nonce das
  Signatur-Zeichen zufaellig auch auf "X" (~1/64), ist das "manipulierte" Token
  unveraendert und der Test schlaegt fehl. Reiner Testcode-Bug, kein Produktionscode
  betroffen, zweimal direkt danach wieder gruen reproduziert)
- offen: keine

## Iteration 85 — fix-payment-delete-bypasses-invoice-lock — done — 2026-08-23 09:34
- commit: 42439f4c
- gebaut: `payment.Service.Delete` (`internal/biz/payment/service.go:172`) prueft jetzt
  `inv.LockedAt != nil` vor dem Loeschen und lehnt mit dem neuen Sentinel `ErrInvoiceLocked`
  (`internal/biz/payment/repository.go`) ab — GoBD-§146-Guard analog zu
  `transitionToPaidInTx`/`revertPaidStatusInTx`, aber bewusst als Ablehnung statt stillem
  Ueberspringen: eine geloeschte Zahlung kann nicht wie ein Auto-Statuswechsel wortlos
  verschwinden, sonst fehlt der Buchungsnachweis waehrend die Rechnung weiter als bezahlt
  gilt. `internal/server/biz_grpc.go` mappt den neuen Sentinel auf
  `codes.FailedPrecondition` (Muster von `invoice.ErrInvoiceLocked`). Beide RPCs, die
  `Service.Delete` erreichen (`DeletePayment`, `DeleteFinanceTransaction` mit `pay-`-Praefix),
  laufen bereits ueber `mapBizError` — keine Handler-Aenderung noetig.
  Root-Cause-Aufraeumung: da `revertPaidStatusInTx` jetzt nur noch erreicht wird, wenn
  `inv.LockedAt == nil` (der neue Guard in `Delete` liegt davor), war ihr eigener
  `LockedAt`-Check darin toter Code — entfernt, Funktionskommentar auf den neuen Aufrufpfad
  umgeschrieben, statt eine nie mehr greifende Sicherheitszeile stehen zu lassen.
  Bestehender Test `TestDelete_NoRevertWhenInvoiceLocked` erwartete das alte Verhalten
  (Loeschen erlaubt, nur Revert uebersprungen) — umgebaut zu
  `TestDelete_RejectedWhenInvoiceLocked` (ganze Operation abgelehnt, Zahlungszeile bleibt
  erhalten) + neuer Gegentest `TestDelete_AllowedWhenInvoiceNotLocked` (unveraendertes
  Verhalten fuer nicht gesperrte Rechnungen). `TestMapBizError` bekam den neuen Sentinel-Fall.
- gate: build ok (`go build -p 2` payment + server + cmd/biz + cmd/gateway) | vet ok
  (payment + server) | lint ok (0 issues, payment + server) | test ok (`go test -count=1
  ./internal/biz/payment/... ./internal/server/... ./internal/gateway/` gruen, 0
  uebersprungen, mit gesetztem `DATABASE_URL=...kmuhub_app...`) | migration n.a. (keine
  Schema-Aenderung) | rls-smoke n.a. (keine neue Tabelle/Policy) | Route: keine
  (`go test ./internal/gateway/` trotzdem gefahren, weil `internal/server` angefasst wurde)
- coverage: internal/biz/payment 85,4 % -> 85,3 % (selbst gemessen per `git stash`/`stash
  pop`, `go tool cover -func`; die neue Guard-Zeile UND ein jetzt toter, entfernter Zweig
  in `revertPaidStatusInTx` heben sich fast auf — kein Coverage-Ziel dieser Unit, reiner
  Bugfix)
- mutations-probe: den neuen `if inv.LockedAt != nil { return ErrInvoiceLocked }`-Block in
  `Service.Delete` testweise entfernt, `go test -run TestDelete_RejectedWhenInvoiceLocked
  ./internal/biz/payment/` wurde rot ("Expected error ... but got nil"), Mutation
  zurueckgedreht, `git diff internal/biz/payment/service.go` zeigt danach wieder nur die
  beabsichtigten zwei Aenderungen (neuer Guard in Delete + Kommentar-/Dead-Code-Aufraeumung
  in revertPaidStatusInTx)
- verify vorgaenger: sauber. `3fe6b86d` (Iteration 84) geprueft (`git show --stat`, dann
  gezielt `biz_grpc.go`+`biz_grpc_amounts_test.go` gegen die acht Fehlerklassen): beide
  `BuildGoBDRows`-Aufrufe routen jetzt exakt wie die drei bestehenden Call-Sites durch
  `documentCurrency()`, neuer Test pinnt den Helper direkt. Kein gRPC-Bypass, kein Stub,
  kein `.proto`, kein neuer `RequirePermission`, keine neue Tabelle, kein Wire-Shape-Wechsel,
  keine neue Route, kein Guard-Ersatz — keine der acht Klassen einschlaegig.
- neue-units: keine
- offen: keine

## Iteration 86 — fix-lexware-config-lookup-cross-tenant-under-sysctx — done — 2026-08-23 09:40
- commit: 3dcb02f3
- gebaut: `Service.SyncContacts` (`internal/biz/lexware/service.go:201-209`) in zwei
  Funktionen aufgeteilt — `SyncContacts(ctx, tenantID)` bleibt der authentifizierte
  gRPC-/manuelle Trigger-Pfad (`getConfigID(ctx)` unter echtem RLS-Kontext), das neue
  `SyncContactsWithConfig(ctx, configID, tenantID)` (Zeile 210-241) ist die eigentliche
  Sync-Logik fuer einen bereits bekannten `configID` — exakt das Muster aus
  `bexio.Service.PullInvoicesWithConfig`. Der Scheduler-Ticker
  (`internal/biz/lexware/scheduler.go:170-179`) ruft jetzt
  `s.service.SyncContactsWithConfig(ctx, ts.configID, ts.tenantID)` statt
  `SyncContacts(ctx, ts.tenantID)` — er loest den Config nicht mehr unter Systemkontext
  per `GetByPlatform` (ohne Tenant-Filter) neu auf, sondern nutzt den beim
  `AddTenant`-Aufruf bereits bekannten `configID` direkt. Damit kann der Scheduler bei
  mehr als einem aktiven Lexware-Tenant nicht mehr versehentlich die Config eines
  fremden Tenants fuer den periodischen Contact-Sync verwenden.
  Der WEBHOOK-Pfad (`HandleWebhookEvent`) ist bewusst NICHT gefixt — siehe
  `verify vorgaenger`-Abschnitt unten fuer die Begruendung, es ist eine
  Datenmodell-Entscheidung, keine Code-Korrektur; stattdessen ein `lean:`-Marker im
  Code (`service.go:262-267`) mit Verweis auf die neue Folge-Unit.
- gate: build ok (`go build -p 2` lexware + server + gateway + cmd/biz + cmd/gateway) |
  vet ok (lexware) | lint ok (0 issues, lexware) | test ok (`go test -count=1
  ./internal/biz/lexware/...` gruen, 94 Tests laut `-v`-Lauf, 0 uebersprungen;
  `go test -count=1 ./internal/gateway/` gruen, mit gesetztem
  `DATABASE_URL=...kmuhub_app...`) | migration n.a. (keine Schema-Aenderung) |
  rls-smoke n.a. (keine neue Tabelle/Policy) | Route: keine (keine
  Gateway-Route/OpenAPI-Aenderung, `go test ./internal/gateway/` trotzdem gefahren,
  weil die Unit sicherheitsnah ist und der Fix in einem regelmaessig vom Gateway
  aufgerufenen Paket liegt)
- coverage: internal/biz/lexware 74,4 % -> 74,5 % (selbst gemessen per `git stash`/
  `stash pop`, `go tool cover -func`; kaum Bewegung, weil die neue
  `SyncContactsWithConfig`-Funktion nur den bereits vollstaendig getesteten Koerper der
  alten `SyncContacts`-Funktion uebernimmt — die neue Zeile ist im Wesentlichen der
  duenne `SyncContacts`-Wrapper, der `getConfigID` aufruft)
- mutations-probe: in `SyncContactsWithConfig` testweise `configID, _ =
  s.getConfigID(ctx)` als erste Zeile eingefuegt (re-resolved den Parameter statt ihn zu
  nutzen), `go test -run
  TestSyncContactsWithConfig_SchedulerPathDoesNotResolveViaGetByPlatform
  ./internal/biz/lexware/` wurde rot (der eingebaute `t.Fatal` im
  `getByPlatformFn`-Mock griff: "SyncContactsWithConfig must not re-resolve the config
  via GetByPlatform under system context"), Mutation zurueckgedreht, `git diff --stat
  service.go` zeigt danach wieder nur die beabsichtigten Aenderungen
- verify vorgaenger: sauber. `42439f4c` (Iteration 85, fix-payment-delete-bypasses-invoice-lock)
  geprueft (`git show --stat` + gezielt `service.go`/`repository.go`/`biz_grpc.go`
  gegen die acht Fehlerklassen): kein gRPC-Bypass, kein Stub, kein `.proto`, kein neuer
  `RequirePermission`, keine neue Tabelle, kein Wire-Shape-Wechsel, keine neue Route,
  kein Guard-Ersatz — keine der acht Klassen einschlaegig.
  Zur eigenen Unit: der Webhook-Pfad (`HandleWebhookEvent`) ist laut `done_when`
  entweder zu haerten ODER die fehlende Unterscheidungsmoeglichkeit als
  Datenmodell-Folge-Unit anzulegen. Recherche ergab: der Webhook traegt zwar optional
  eine `organization_id` im Payload (`route_lexware.go:579-592`), aber
  `Service.Connect` (`service.go:78-125`) speichert bei der Verbindung KEINE
  Lexware-Organisations-Kennung in `IntegrationConfig.Metadata` — ein Abgleich
  "Webhook-OrgID gegen gespeicherte Config-OrgID" ist mit dem heutigen Datenmodell
  technisch unmoeglich, das Feld existiert nirgends. Zusaetzlich ist
  `LEXWARE_WEBHOOK_SECRET` ein einziges globales Secret (kein Secret pro Tenant), kann
  also auch nicht zur Tenant-Unterscheidung dienen. Beide moeglichen Loesungen (Profil-
  Abruf + Metadata-Erweiterung vs. eigener Webhook-Pfad/Secret pro Tenant) veraendern
  entweder das Datenmodell oder den mit Lexware zu hinterlegenden Callback-Vertrag —
  keine ist "nebenbei" in dieser Unit zu entscheiden, deshalb als eigene Unit
  `harden-lexware-webhook-organization-id-scoping` ans Backlog-Ende gehaengt statt
  hier zu raten.
- neue-units: harden-lexware-webhook-organization-id-scoping (Datenmodell-Entscheidung
  fuer den Webhook-Pfad, GEHOERT LUKE — siehe Notes der Unit fuer die zwei
  vorgeschlagenen Loesungswege)
- offen: keine

## Iteration 87 — fix-idempotency-409-rollout-non-finance-routes — done — 2026-08-23 09:47
- commit: 5499043b
- gebaut: erste Registrar-Gruppe der Nicht-Finance-409-Ausrollung: `wiki`. 11 mutierende
  Operationen in `backend/api/openapi.yaml` (Zeilen ~15638-16000, Tags `wiki-articles` +
  `wiki-categories`) tragen jetzt `"409": { $ref: "#/components/responses/IdempotencyInFlight" }`
  — createWikiArticle, updateWikiArticle, deleteWikiArticle, restoreWikiArticleVersion,
  uploadWikiArticleAttachment, deleteWikiArticleAttachment, createWikiCategory,
  updateWikiCategory, deleteWikiCategory, createWikiShareToken, revokeWikiShareToken. Keine
  dieser Operationen hatte vorher ein `"409":` fuer einen Geschaeftszustand, also reines
  Anhaengen, kein Merge noetig (anders als bei den Quote-Routen der Finance-Vorgaenger-Unit).
  Die public Redemption-Route `POST /api/v1/public/wiki/articles/{token}` bleibt unangetastet:
  sie liegt ausserhalb des per `r.Use(authMiddleware)` geschuetzten `r.Route("/api/v1/wiki", ...)`
  -Blocks in `route_wiki.go:135` und laeuft nicht durch `authWithIdempotency`.
  Neue Teil-Unit `fix-idempotency-409-rollout-non-finance-routes-2` mit der um Wiki bereinigten
  Restliste (39 Registrar-Gruppen) ans Backlog-Ende gehaengt.
- gate: build ok (`go build -p 2` gateway + cmd/gateway) | vet ok (gateway) | lint ok
  (0 issues, gateway) | swagger-cli validate ok (`npx swagger-cli validate api/openapi.yaml` →
  "api/openapi.yaml is valid") | test ok (`go test -count=1 ./internal/gateway/
  -run TestOpenAPIRouteDrift` gruen; `go test -count=1 ./internal/gateway/...` gesamt gruen, mit
  gesetztem `DATABASE_URL=...kmuhub_app...`) | migration n.a. (keine Schema-Aenderung) |
  rls-smoke n.a. (keine neue Tabelle/Policy)
- coverage: n.a. (Doku-Unit, reine OpenAPI-Response-Ergaenzung, kein Verhalten geaendert —
  Coverage-Ziel laut `coverage_start:` der Unit selbst n.a.)
- mutations-probe: n.a. (kein Code-Verhalten geaendert; die einzige mechanische Probe ist
  `git diff --stat api/openapi.yaml` = "1 file changed, 11 insertions(+)" — exakt die 11
  beabsichtigten `"409":`-Zeilen, keine Nebenwirkung auf andere Pfade)
- verify vorgaenger: sauber. `3dcb02f3` (Iteration 86, fix-lexware-config-lookup-cross-tenant-
  under-sysctx) geprueft (`git show` auf `service.go`/`scheduler.go`/`service_wiring_test.go`
  gegen die acht Fehlerklassen): kein gRPC-Bypass (rein interner Service-Split, kein
  Gateway-Handler betroffen), kein Stub, kein `.proto`, kein neuer `RequirePermission`, keine
  neue Tabelle, kein Wire-Shape-Wechsel, keine neue Route, kein Guard-Ersatz — keine der acht
  Klassen einschlaegig. Die vorige Iteration hatte selbst schon die Vor-Vorgaenger-Iteration 85
  (`42439f4c`) sauber verifiziert.
  Zusaetzlich verify auf die erste gezogene Unit dieser Iteration,
  `fix-bexio-config-lookup-cross-tenant-under-sysctx`: ihre eigenen Notes verlangen "GESPERRT IN
  LAUF 11: ... SOFORT mit `blocked_reason` abzuschliessen, ohne Bauversuch" — `internal/biz/bexio`
  steht im aktuellen Lauf-12-Kopf weiterhin unter "GESPERRT IN DIESEM LAUF" (G3, produktiv aus).
  Unit auf `status: blocked` gesetzt, kein Bauversuch, naechste Unit
  (`fix-idempotency-409-rollout-non-finance-routes`) gezogen.
- neue-units: fix-idempotency-409-rollout-non-finance-routes-2 (Rest-Liste der Registrar-Gruppen,
  Wiki entfernt)
- offen: keine

## Iteration 88 — feat-quote-converted-invoice-number-on-read — done — 2026-08-23 09:53
- commit: <pending, siehe naechster Eintrag>
- gebaut: `finance_quotes`-Lesepfad (`GetByID`, `List`, `GetByDealID` in
  `internal/biz/quote/postgres_repository.go`) traegt jetzt `converted_invoice_number`
  ueber einen `LEFT JOIN LATERAL` gegen `finance_invoices` (`fi.source_quote_id = q.id
  AND fi.status <> 'cancelled', ORDER BY fi.created_at DESC LIMIT 1`) — exakt ein Join-Kandidat
  pro Quote-Zeile, damit weder `List`s Pagination noch `Count` durch die bekannte Race
  (mehrere nicht-stornierte Rechnungen pro Quote, siehe `harden-quote-conversion-unique-index`)
  verdoppelt werden koennen. Neues Modellfeld `models.Quote.ConvertedInvoiceNumber *string`
  (nur gelesen, keine Spalte, kein Write-Pfad). `bizv1.Quote.converted_invoice_number`
  (Feld 17) ergaenzt und `make proto-biz`-Aequivalent per direktem `protoc`-Aufruf regeneriert
  (`make` ist in dieser Bash-Umgebung nicht vorhanden — Kommando aus dem Makefile-Target
  uebernommen). `toProtoQuote` (`internal/server/biz_grpc.go`) setzt das Feld. `openapi.yaml`
  Quote-Schema um das Feld ergaenzt (Beschreibung: leer nach Storno, da erneut umwandelbar).
  FACHFRAGE "Entwurf ohne Nummer" (siehe Notes der Unit) beantwortet: es wird bewusst NUR
  `invoice_number` exponiert, keine zusaetzliche `invoice_id`. Solange die neu erzeugte
  Rechnung ein Entwurf ist (leere `invoice_number`, siehe
  `fix-invoice-number-unique-index-blocks-second-draft`), bleibt das Feld leer und der
  FE-Button "In Rechnung umwandeln" (`QuoteDetailPanel.tsx:368`) sichtbar — das ist eine
  hinnehmbare UX-Luecke, KEIN Duplikat-Risiko: `Service.CreateFromQuote` blockt seit
  `fix-quote-to-invoice-duplicate-creation` (Iteration 70) jede zweite Konversion serverseitig,
  unabhaengig davon, was das FE anzeigt. Eine `invoice_id`-Ergaenzung wuerde eine FE-Aenderung
  voraussetzen (heutiger Lookup ist `invoices.find(inv => inv.invoice_number === convertedNumber)`),
  die aus diesem Backend-Loop heraus nicht sinnvoll mitgeliefert werden kann — bewusst nicht
  gebaut, um keinen ungenutzten Wire-Feld-Zusatz zu erzeugen (YAGNI).
  Zusaetzlich `fix-hr-manual-entry-idempotency-key-not-enforced` OHNE Bauversuch auf `blocked`
  gesetzt (siehe verify-vorgaenger-Absatz unten fuer den Grund) und
  `feat-quote-converted-invoice-number-on-read` stattdessen gezogen.
- gate: build ok (`go build -p 2` quote+server+gateway+cmd/biz+cmd/gateway) | vet ok
  (quote+server+gateway) | lint ok (0 issues, quote+server+gateway) | swagger-cli validate ok
  (`api/openapi.yaml is valid`) | test ok (`go test -count=1 ./internal/biz/quote/` gruen inkl.
  der 2 neuen DB-Tests; `go test -count=1 ./internal/server/` gruen; `go test -count=1
  ./internal/gateway/` gruen inkl. `TestOpenAPIRouteDrift`, mit gesetztem
  `DATABASE_URL=...kmuhub_app...`) | migration n.a. (keine neue Spalte, reiner Read-Join) |
  rls-smoke n.a. (keine neue Tabelle/Policy, bestehende RLS auf finance_quotes/finance_invoices
  unveraendert)
- coverage: internal/biz/quote 33,3 % -> 49,6 % (eigene Messung vor/nach, `go tool cover -func`)
- mutations-probe: den `fi.status <> 'cancelled'`-Filter im LATERAL-Join von `GetByID` entfernt
  -> `TestPostgresRepository_GetByID_CancelledConversionLeavesFieldEmpty` wurde rot
  ("Expected nil, but got: (*string)"), `TestPostgresRepository_GetByID_PopulatesConvertedInvoiceNumber`
  blieb gruen (Positivfall unveraendert). Zurueckgedreht, `git diff --stat` danach identisch mit
  dem Stand vor der Mutation, beide Tests wieder gruen.
- verify vorgaenger: sauber. `5499043b` (Iteration 87, fix-idempotency-409-rollout-non-finance-routes)
  geprueft (`git show --stat` + Diff auf `openapi.yaml`): reine OpenAPI-Response-Doku (11×
  `"409": IdempotencyInFlight` auf Wiki-Routen), keine der acht Fehlerklassen einschlaegig
  (kein gRPC-Bypass, kein Stub, kein `.proto`, kein neuer Guard, keine neue Tabelle, kein
  Wire-Shape-Wechsel, keine neue Route, kein Guard-Ersatz). Zusaetzlich den ausstehenden
  Commit-SHA-Platzhalter aus Iteration 87 im Journal nachgetragen (`5499043b`, per Diff-Inhalt
  verifiziert) und dafuer committet (`98c7eb7f`).
  Die zuerst gezogene Unit `fix-hr-manual-entry-idempotency-key-not-enforced` verlangt in ihren
  eigenen Notes explizit eine Architekturentscheidung von Luke ("ENTSCHEIDUNG GEHOERT LUKE ...
  keine Standardwahl aus diesem Lauf treffen" — Middleware-Idempotency vs. eigene
  `idempotency_key`-Spalte mit Unique-Index). Ohne Bauversuch auf `blocked` gesetzt, naechste
  Unit (`feat-quote-converted-invoice-number-on-read`) gezogen.
- neue-units: keine
- offen: keine

## Iteration 89 — fix-gateway-billing-lock-dunning-notice-listroutes-missing-400-500-docs — done — 2026-08-23 10:06
- commit: ab44674e
- gebaut: Sechs Doku-Luecken in `api/openapi.yaml` geschlossen, alle vorher am Handler
  verifiziert (nicht aus der Unit uebernommen): `POST /finance/invoices/{id}/lock` +400
  (`validateUUIDParam`, helpers.go:120-127 schreibt 400 "invalid id");
  `POST /finance/dunning/{id}/notice` +400 (dito) und +500
  (`cannedResponseMarshaler.Marshal`-Fehler, route_biz_billing.go:848-851);
  `GET /finance/credit-notes`, `GET /finance/dunning`, `GET /finance/invoices/{id}/payments`
  je +500 (`hrMarshalSlice`-Fehler); `GET /finance/invoices/{id}/payments` zusaetzlich +400
  (Pfad-UUID). Der 400-Block von `POST /finance/invoices/{id}/payments` wurde vom blossen
  `$ref: IdempotencyKeyRequired` auf einen inline zusammengefuehrten Block umgestellt, der
  ALLE DREI real geschriebenen Ursachen unterscheidbar nennt: ungueltige Pfad-UUID,
  malformed/validation-failed Body (`decodeAndValidate`, helpers.go:225-240 — die Unit nannte
  nur zwei Ursachen, die dritte kam beim Nachlesen des Handlers dazu) und fehlender
  Idempotency-Key. Stil von der Iteration-76-Vorlage uebernommen (openapi.yaml:8940ff:
  inline `description` + `content`), weil YAML keinen zweiten `"400"`-Schluessel erlaubt.
  Kein Verhalten geaendert — reine Spec, kein Go-Diff.
- gate: build ok (`go build -p 2 ./internal/gateway/... ./cmd/gateway/...`) | vet ok |
  lint ok (0 issues) | swagger-cli validate ok (`api/openapi.yaml is valid`) | test ok
  (`go test -count=1 ./internal/gateway/` gruen, coverage 56,6 %, inkl.
  `TestOpenAPIRouteDrift` und `TestOpenAPIRouteDriftParserSanity` explizit gruen;
  `DATABASE_URL` auf `kmuhub_app` gesetzt, `-v`-Lauf zeigt **0 SKIP**) | migration n.a. |
  rls-smoke n.a. (keine Tabelle, keine Policy, kein SQL)
- coverage: internal/gateway 56,6 % -> 56,6 % (eigene Messung vor/nach; die Aenderung
  enthaelt keine Go-Zeile, eine Verschiebung waere hier ein Messfehler)
- mutations-probe: Der naheliegende naive Weg wurde als Mutation gefahren — statt den
  400-Block zusammenzufuehren, den bestehenden `"400": { $ref: IdempotencyKeyRequired }`
  stehenlassen und einen zweiten `"400"`-Schluessel danebensetzen. `swagger-cli validate`
  wurde sofort rot: `Error parsing ... duplicated mapping key (9381:9)`. Zurueckgedreht,
  danach wieder `is valid`, `git diff --stat` identisch mit dem Stand vor der Mutation.
  ACHTUNG fuer kuenftige Spec-Units: `TestOpenAPIRouteDrift` haette diese Mutation NICHT
  gefangen — er vergleicht nur registrierte Pfade gegen Spec-Pfade, keine Response-Codes.
  Der einzige Waechter fuer Status-Code-Doku ist `swagger-cli validate` (rein strukturell,
  fuer fehlende Codes blind) plus das Lesen des Handlers. Es gibt im Repo keinen Test, der
  dokumentierte gegen real geschriebene Status-Codes abgleicht — daraus die neue Unit unten.
- verify vorgaenger: sauber. `51cd455d` (Iteration 88,
  feat-quote-converted-invoice-number-on-read) gegen alle acht Fehlerklassen geprueft:
  kein gRPC-Bypass (nur Repository/`toProtoQuote`, kein neuer Handler), kein Stub, `.proto`
  im selben Commit regeneriert (`biz.pb.go` liegt im `--stat`), kein neuer
  `RequirePermission`, keine neue Tabelle — der LATERAL-Join ist ueber
  `fi.tenant_id = q.tenant_id` tenant-gescoped, also keine Tenant-Luecke; Wire-Shape
  snake_case `converted_invoice_number` passend zum FE-Lookup; keine neue Route,
  Quote-Schema in `openapi.yaml` ergaenzt; kein Guard ersetzt. Zusaetzlich geprueft, dass
  der unqualifizierte `whereClause` in `List` durch den Join nicht mehrdeutig wird: die
  LATERAL-Subquery exportiert nur `invoice_number`, eine Spalte, die `finance_quotes` nicht
  hat — kein Ambiguitaetsrisiko.
- neue-units: test-openapi-documented-status-codes-vs-handlers (ans Backlog-Ende gehaengt) —
  belegter Befund aus der Mutations-Probe: kein Test im Repo gleicht die in `openapi.yaml`
  dokumentierten Response-Codes gegen die vom Handler real geschriebenen ab. Genau diese
  Luecke hat inzwischen vier eigene Fix-Units erzeugt (Iterationen 76, 78, 87 und diese) —
  jedes Mal von Hand gefunden, jedes Mal nur fuer die gerade gelesenen Routen.
- offen: `harden-quote-conversion-unique-index` wurde OHNE Bauversuch auf `blocked` gesetzt
  (Arbeitsbaum unberuehrt) und braucht eine Entscheidung von Luke. Die Unit verlangt in
  ihren eigenen Notes einen Produktionsbefund, den der Loop nicht erheben darf; der
  partielle Unique-Index scheitert bei der Migration hart, wenn Altdaten kollidieren, und
  CD deployt automatisch. Die eine noetige Abfrage steht als `blocked_reason` in
  `BACKLOG.yml` (`SELECT tenant_id, source_quote_id, count(*) FROM finance_invoices WHERE
  source_quote_id IS NOT NULL AND status <> 'cancelled' GROUP BY 1,2 HAVING count(*) > 1;`,
  Prod mit `-U kmuhub -d kmuhub`). Leeres Ergebnis => Unit auf `todo` zuruecksetzen, dann
  ist sie im naechsten Lauf gefahrlos baubar.
  ZWEITER BEFUND, nicht gefixt: `BACKLOG.yml` ist **kein gueltiges YAML** und war es schon
  am Stand `51cd455d` (also nicht durch diese Iteration entstanden — gegen `git show HEAD:`
  gegengeprueft). Ursache sind 15 `done_when`-Listeneintraege, die einen unquotierten
  Doppelpunkt-Leerzeichen enthalten (`- Idealerweise: ...` Zeile 2892, `- Mutations-Probe: ...`
  Zeile 649, `- Ein Test belegt: ...` Zeile 3177 usw.) — YAML liest die als Mapping-Key und
  bricht ab. Praktisch folgenlos, weil der Treiber die Datei per Regex liest und nicht per
  YAML-Parser (der Kopfkommentar "`model:` muss VOR `status:` stehen" ist genau der Beleg
  dafuer). Ich habe die Stelle testweise quotiert, danach brach die naechste, und habe es
  bewusst zurueckgedreht: eine Teilkorrektur haette den Diff aufgeblaeht und faelschlich den
  Eindruck erzeugt, die Datei sei nun YAML-rein. Wer sie irgendwann mit einem echten Parser
  auswerten will (Reporting, Statistik), muss alle 15 Stellen in einem eigenen Durchgang
  quotieren. Mein eigener neuer Unit-Eintrag ist frei davon (Zeile gezielt quotiert).

## Iteration 90 — fix-db-test-cleanup-order-leaks-fixtures — done — 2026-08-23 10:13
- commit: 543924d5
- gebaut: Der aus Iteration 82 (kpi_postgres_test.go) bekannte Bug — `defer pool.Close()` UND
  `t.Cleanup(func() { testutil.CleanupRow(...) })` (direkt oder ueber einen Seed-Helfer) in
  derselben Testfunktion, wobei `defer` VOR `t.Cleanup` laeuft und die Row-Cleanups auf einem
  bereits geschlossenen Pool landen — in 6 der 11 gemeldeten Dateien gefixt: `defer pool.Close()`
  -> `t.Cleanup(func() { pool.Close() })`, damit alle Cleanups derselben Funktion in echter LIFO-
  Reihenfolge laufen (Fix (a) aus der Unit-Note, identisch zum kpi_postgres_test.go-Vorbild):
  `internal/biz/expense/integration_test.go` (6 von 7 Vorkommen — die siebte Funktion,
  `TestPostgresRepository_CreateRejectsNonPositiveAmount`, registriert gar kein `t.Cleanup`,
  blieb unangetastet; zusaetzlich eine jetzt falsche erklaerende Code-Kommentarzeile entfernt,
  die noch von einem `defer pool.Close()` sprach), `internal/biz/hr/personnel_documents_read_test.go`
  (1/1), `internal/biz/hr/salary_document_category_test.go` (1 von 2 — die zweite Funktion,
  `TestSalaryDocumentCategory_SeedRow_DB`, seedet nichts und registriert kein `t.Cleanup`),
  `internal/chat/bookmark/postgres_repository_test.go` (3/3), `internal/email/label/postgres_repository_test.go`
  (6/6), `internal/email/rule/postgres_repository_test.go` (5/5).
  Die verbleibenden 4 der urspruenglich 11 gemeldeten Dateien
  (`internal/crm/customfield/value_tenant_isolation_test.go`,
  `internal/email/contactlink/tenant_isolation_test.go`,
  `internal/notification/integration/postgres_repository_test.go`,
  `internal/notification/notification/postgres_repository_test.go`) sind FALSCH-POSITIV: der
  urspruengliche Grep-Befehl matcht den literalen String "defer pool.Close()" auch in
  erklaerenden Code-Kommentaren, die genau diesen Bug beschreiben und bewusst vermeiden (Code
  nutzt dort bereits `t.Cleanup(pool.Close)` bzw. `t.Cleanup(func() { pool.Close() })`) — bei
  Nachlesen kein einziges echtes `defer pool.Close()`-Statement in diesen vier Dateien gefunden.
  EIN WEITERER FALSCH-POSITIV, in der Unit nicht gelistet, aber vom selben Grep erfasst und
  beim Bauen entdeckt: `internal/crm/report/postgres_repository_db_test.go`. Der Haupt-Fixture-
  Helfer `newReportFixture` nutzt bereits korrekt `t.Cleanup(pool.Close)`; sechs Testfunktionen
  rufen zusaetzlich `testutil.PoolFromEnv(t)` ein ZWEITES Mal auf einen eigenen, unabhaengigen
  `pool`, der per `defer pool.Close()` geschlossen wird — dieser zweite Pool traegt aber keine
  eigenen `t.Cleanup`-Registrierungen (nur ein reiner Read-Call gegen den Report), also keine
  Race zwischen defer und t.Cleanup auf demselben Objekt. Redundant (zwei offene Connections
  statt einer), aber kein Leck — bewusst NICHT angefasst, um den Diff auf den belegten Bug zu
  beschraenken.
- gate: build ok (`go build -p 2` gegen alle 5 betroffenen Pakete) | vet ok | lint ok (0 issues,
  golangci-lint gegen alle 5 Pakete) | gofmt -l leer (alle 6 Dateien) | test ok — `DATABASE_URL`
  auf `kmuhub_app`, `go test -count=1` zweimal hintereinander gegen
  `internal/biz/expense`, `internal/biz/hr` (+ 5 Unterpakete), `internal/chat/bookmark`,
  `internal/email/label`, `internal/email/rule` UND zur Kontrolle `internal/crm/report`
  (unveraendert) — alle 12 Pakete beide Male gruen, **0 SKIP** | migration n.a. | rls-smoke n.a.
  (kein Schema-/Policy-Zugriff, reiner Testcode)
- coverage: n.a. (Test-Infrastruktur-Fix, keine Coverage-relevante Produktionscodeaenderung, wie
  in `coverage_start` der Unit vorgegeben)
- mutations-probe: `internal/biz/hr/personnel_documents_read_test.go` testweise auf den
  Vor-Zustand zurueckgesetzt (`t.Cleanup(func() { pool.Close() })` -> `defer pool.Close()`),
  `go test -v -run TestPersonnelDocuments_ListByTenant_RespectsVisibility_DB` gefahren: 10×
  `cleanup <table> id=...: closed pool` in `t.Logf` (3× hr_employee_documents, 3×
  hr_document_categories, 4× users) — Test bleibt trotzdem `--- PASS`, exakt der stille Fehler
  aus dem Scope. Die 10 verwaisten Zeilen manuell per `psql -U kmuhub` geloescht (die
  `kmuhub_app`-Rolle sah sie wegen RLS gar nicht, `DELETE 0`; erst unter dem RLS-freien
  `kmuhub`-Superuser griffen die drei `DELETE`-Statements mit `DELETE 3/3/4`). Fix
  zurueckgedreht, `go test -v` erneut: 0 `cleanup`-Log-Zeilen, weiterhin `--- PASS`, `git diff`
  identisch mit dem Stand vor der Mutation.
- verify vorgaenger: sauber. `ab44674e` (Iteration 89) gegen alle acht Fehlerklassen geprueft:
  reine `openapi.yaml`-Doku-Ergaenzung (400/500-Response-Bloecke), 0 Zeilen Go-Diff — kein
  gRPC-Bypass, kein Stub, kein `.proto`, kein neuer Guard, keine neue Tabelle, keine Wire-Shape-
  Aenderung, keine neue Route, kein ersetzter Guard. `512ec730` direkt danach ist reine
  Journal-Buchhaltung (1 Zeile).
- neue-units: keine — der einzige neue Befund (der falsch-positive `internal/crm/report`-Match)
  ist oben dokumentiert, keine eigene Unit noetig, da kein Bug vorliegt.
- offen: `harden-lexware-webhook-organization-id-scoping` (ENTSCHEIDUNG GEHOERT LUKE, Variante a/b)
  und `harden-quote-conversion-unique-index` (blocked, wartet auf die Prod-Abfrage aus Iteration
  88) bleiben unveraendert offen. `BACKLOG.yml` ist weiterhin kein gueltiges YAML (15 unquotierte
  `done_when`-Doppelpunkte, siehe Iteration 89) — praktisch folgenlos, da der Treiber per Regex
  liest, aber unveraendert ein offener Punkt fuer jeden kuenftigen YAML-Parser-Zugriff.

## Iteration 91 — fix-bexio-state-test-flaky-tamper-byte — done — 2026-08-23 10:31
- commit: 98c64283
- gebaut: `TestDecodeBexioState_ManipulatedSignature`
  (`internal/gateway/bexio_state_test.go:34-53`) ersetzte das erste Zeichen der Signatur immer
  durch das feste Literal `"X"`. Da `encodeBexioState` einen zufaelligen Nonce einbindet, ist
  dieses Zeichen pro Aufruf gleichverteilt ueber das Base64URL-Alphabet — traf es zufaellig `X`
  (~1/64), war das "manipulierte" Token identisch zum Original und der Test schlug faelschlich
  fehl. Fix: das Ersatzzeichen wird jetzt gegen das tatsaechliche Originalzeichen geprueft
  (`if token[dot+1] == replacement { replacement = 'Y' }`), garantiert also immer eine echte
  Abweichung, unabhaengig vom Nonce. Reiner Testcode-Fix, `bexio_state.go` unveraendert.
- gate: build ok (`go build -p 2` gegen internal/gateway + cmd/gateway) | vet ok | lint ok
  (0 issues, golangci-lint gegen internal/gateway) | test ok — `DATABASE_URL` auf `kmuhub_app`,
  `go test -count=1 ./internal/gateway/ -run TestDecodeBexioState -count=50 -v` 50/50 gruen,
  `go test -count=1 ./internal/gateway/` (voller Gateway-Testlauf inkl. TestOpenAPIRouteDrift)
  gruen, **0 SKIP** (per `-v | grep -c SKIP` geprueft) | migration n.a. | rls-smoke n.a. (kein
  Schema-/Policy-Zugriff)
- coverage: n.a. (Test-Infrastruktur-Fix, keine Coverage-relevante Produktionscodeaenderung, wie
  in `coverage_start` der Unit vorgegeben)
- mutations-probe: Testdatei per `git stash` auf den Vor-Zustand (festes `"X"`-Literal)
  zurueckgesetzt, `go test -count=500 ./internal/gateway/ -run TestDecodeBexioState_ManipulatedSignature -v`
  gefahren: **9 von 500 Laeufen schlugen fehl** — exakt der beschriebene Kollisionsbug, empirisch
  nahe der erwarteten ~1/64-Quote (~7,8/500). `git stash pop` zurueckgeholt, denselben
  `-count=500`-Lauf mit dem Fix wiederholt: **500/500 gruen**. `git diff --stat` danach nur
  `bexio_state_test.go` (+ `BACKLOG.yml`) — kein Produktionscode veraendert.
- verify vorgaenger: sauber. `543924d5` (Iteration 90) gegen alle acht Fehlerklassen geprueft:
  reiner Testinfrastruktur-Fix (`defer pool.Close()` -> `t.Cleanup(pool.Close)` in sechs
  Testdateien) — kein gRPC-Bypass, kein Stub, kein `.proto`, kein neuer Guard, keine neue
  Tabelle, keine Wire-Shape-Aenderung, keine neue Route, kein ersetzter Guard. `01dea3ad` direkt
  danach ist reine Journal-Buchhaltung (SHA-Eintrag).
- neue-units: keine
- offen: `harden-lexware-webhook-organization-id-scoping` (ENTSCHEIDUNG GEHOERT LUKE, Variante
  a/b) bleibt unveraendert offen. `fix-idempotency-409-rollout-non-finance-routes-2` und
  `test-openapi-documented-status-codes-vs-handlers` bleiben als naechste `todo`-Units mit
  erfuellten deps liegen. `BACKLOG.yml` ist weiterhin kein gueltiges YAML (unquotierte
  `done_when`-Doppelpunkte) — praktisch folgenlos, da der Treiber per Regex liest.

## Iteration 92 — fix-idempotency-409-rollout-non-finance-routes-2 — done — 2026-08-23 10:26
- commit: e7076da2
- gebaut: Die **Formulare**-Registrar-Gruppe auf 409 umgestellt — 11 mutierende Operationen aus
  `route_formulare.go` (Schemas POST/PATCH/DELETE, `schemas/{id}/duplicate` POST,
  `schemas/{id}/submissions` POST, `schemas/{id}/share-links` POST, `schemas/{id}/webhooks` POST,
  `submissions/{id}` PATCH, `share-links/{id}` DELETE, `webhooks/{id}` PATCH/DELETE).
  Zehn davon bekamen die Zeile `"409": { $ref: "#/components/responses/IdempotencyInFlight" }`,
  **numerisch einsortiert** statt stumpf ans Blockende — bei `/api/v1/formulare/share-links/{id}`
  steht sie deshalb vor dem bestehenden `"503"`. Der elfte Fall,
  `POST /api/v1/formulare/schemas/{id}/share-links`, trug bereits ein 409 mit
  Geschaeftsbedeutung ("schema is not public"): dort nicht ueberschrieben, sondern nach der
  Finance-Vorlage (`f6d4a3ad`) die `description` auf `>-` umgestellt, den In-Flight-Satz
  angehaengt und `headers.Retry-After` mit dem Vermerk ergaenzt, dass der Header fuer den
  urspruenglichen Konfliktfall nicht gesetzt ist. Die public Route
  `POST /api/v1/public/formulare/submit/{token}` bleibt bewusst unangetastet — sie liegt
  ausserhalb des `authWithIdempotency`-Blocks und laeuft nicht durch die Middleware.
  Reine Spec-Doku, keine Zeile Go-Code geaendert.
- gate: build ok (`go build -p 2 ./internal/gateway/... ./cmd/gateway/...`) | vet ok | lint ok
  (golangci-lint `./internal/gateway/...`, 0 issues) | test ok — `DATABASE_URL` auf `kmuhub_app`,
  `go test -count=1 -v ./internal/gateway/`: **3847 PASS, 0 SKIP** (per
  `grep -c -- '--- SKIP'` geprueft), darin `TestOpenAPIRouteDrift` gruen;
  `npx swagger-cli validate backend/api/openapi.yaml` = "is valid" | migration n.a. |
  rls-smoke n.a. (kein Schema-/Policy-Zugriff)
- coverage: n.a. (Doku-Unit, kein Coverage-Ziel — `coverage_start` der Unit sagt dasselbe;
  es wurde kein Produktionscode angefasst, an dem eine Zeilenabdeckung haengen koennte)
- mutations-probe: Der `$ref` der ersten neu eingefuegten 409-Zeile
  (`POST /api/v1/formulare/schemas`) wurde testweise auf `IdempotencyInFlightBROKEN` verbogen:
  `swagger-cli validate` schlug fehl mit `Token "IdempotencyInFlightBROKEN" does not exist.`,
  Exit 1. Damit ist belegt, dass der Validator die neuen Zeilen wirklich aufloest und sie nicht
  etwa auf einer falschen Einrueckungsebene ignoriert werden. Zurueckgedreht, danach wieder
  "is valid"; `git diff --stat` zeigt nur `api/openapi.yaml` (+19/-1: 10 neue Ref-Zeilen plus
  der 9-Zeilen-Merge-Block, der eine `description`-Zeile ersetzt) und die beiden Loop-Dateien.
- verify vorgaenger: sauber. `98c64283` (Iteration 91, fix-bexio-state-test-flaky-tamper-byte)
  gegen alle acht Fehlerklassen geprueft: der Diff beruehrt ausschliesslich
  `internal/gateway/bexio_state_test.go` (+10/-2, deterministisches Ersatzzeichen statt festem
  `"X"`) — kein gRPC-Bypass, kein Stub, kein `.proto`, kein neuer `RequirePermission`, keine
  neue Tabelle, keine Wire-Shape-Aenderung, keine neue Route, kein ersetzter Guard.
  `bexio_state.go` selbst ist unveraendert. `14a2ca3e` direkt danach ist reine
  Journal-Buchhaltung (1 Zeile SHA).
- neue-units: `fix-idempotency-409-rollout-non-finance-routes-3` (Restliste jetzt ~37 Gruppen,
  Formulare gestrichen; zusaetzlich aufgenommen: der Hinweis auf die numerische Einsortierung
  und die Warnung, dass `/auth/login`, `/auth/refresh`, `/auth/2fa` in der
  `idempotencyWhitelist` stehen und deshalb KEIN In-Flight-409 bekommen duerfen).
- offen: **`harden-lexware-webhook-organization-id-scoping` ist von `todo` auf `blocked`
  gesetzt** — sie stand seit Iteration 86 als erste `todo`-Unit am Backlog-Kopf, verlangt aber
  laut eigenen `notes` eine Datenmodell-Entscheidung von Luke (Variante a: `organization_id`
  bei `Connect` per `GET /v1/profile` holen und in `IntegrationConfig.Metadata` persistieren,
  plus eine `GetActiveByPlatform`-Variante; Variante b: Webhook-Pfad/Secret pro Tenant, was den
  bei Lexware hinterlegten Callback-URL aendert und damit ein externer Vertrag ist). Sie wurde
  deshalb in jeder Iteration stillschweigend uebersprungen, was den Backlog-Kopf verstopft hat.
  `blocked_reason` gesetzt, wieder auf `todo` stellen, sobald die Variante gewaehlt ist —
  Luke muss hier entscheiden, nicht der Loop.
  `BACKLOG.yml` ist weiterhin kein gueltiges YAML (unquotierte `done_when`-Doppelpunkte aus
  frueheren Iterationen) — praktisch folgenlos, da der Treiber per Regex liest; die in dieser
  Iteration neu angelegte Unit quotet ihre betroffene `done_when`-Zeile korrekt.

## Iteration 93 - test-openapi-documented-status-codes-vs-handlers - done - 2026-08-23 10:32
- commit: 763df3bc
- gebaut: `backend/internal/gateway/openapi_status_code_drift_test.go` (neu, ~600 Zeilen,
  reiner Test - keine Zeile Produktionscode). Der Test schliesst die dritte Richtung des
  Spec-Guards: bisher gab es `registrierter Pfad ⊆ dokumentierter Pfad`
  (`TestOpenAPIRouteDrift`) und `registriertes METHOD+Pfad ⊆ dokumentiertes METHOD+Pfad`
  (`TestOpenAPIMethodDrift`), aber nichts, was die vom Handler wirklich geschriebenen
  Status-Codes gegen den `responses:`-Block haelt.
  Mechanik in vier Schritten: (1) `buildGatewayRouter` wiederverwenden und den Router
  ablaufen, dabei je Endpoint ueber `runtime.FuncForPC` auf dem gebundenen Method-Value die
  Go-Funktion zurueckholen (`(*ChatRoutes).HandleMarkChannelRead`); chi legt Handler hinter
  `r.With(...)` in einen `*chi.ChainHandler`, der wird vorher ausgepackt. **1196 von 1196
  registrierten `/api/v1/*`-Endpoints loesen so auf, null unaufgeloest.** (2) das
  gateway-Paket mit `go/parser` einlesen und alle Funktionen/Methoden unter demselben
  Schluessel indizieren. (3) je Handler die Codes der bekannten Writer einsammeln
  (`response.JSON/Proto/ProtoList/ProtoListWrapped/Error`, `w.WriteHeader`, `http.Error`,
  `http.Redirect`) und dabei in paket-lokale Aufrufe hinabsteigen - dadurch zaehlen die
  helfer-vermittelten Codes mit, ohne dass eine Helfer-Tabelle von Hand gepflegt werden
  muss: `validateUUIDParam`/`decodeAndValidate`/`validateDateParam` (400),
  `respondServiceUnavailable` (503), `ownerFilterForScope` (401/403) fallen automatisch an.
  (4) Vergleich gegen den `responses:`-Block der passenden Operation.
  `respondGRPCError` ist statisch NICHT aufloesbar (Code kommt aus dem gRPC-Status) - der
  Abstieg stoppt dort, statt 404/409/422/500 dem Handler zuzurechnen. Zweiter Test
  `TestOpenAPIStatusCodeDriftParserSanity` prueft beide Parser gegen bekannte Werte, damit
  ein stiller Parser-Ausfall den Guard nicht gruen faelscht.
- gate: build ok (`go build -p 2 ./internal/gateway/... ./cmd/gateway/...`) | vet ok |
  lint ok (golangci-lint `./internal/gateway/...`, 0 issues) | test ok - `DATABASE_URL` auf
  `kmuhub_app`, `go test -count=1 -v ./internal/gateway/`: **3849 PASS, 0 SKIP**
  (per `grep -c` geprueft; +2 gegenueber Iteration 92 = die beiden neuen Tests) |
  migration n.a. | rls-smoke n.a. (kein Schema-/Policy-Zugriff)
- coverage: `internal/gateway` **56,6 % -> 56,6 %** (unveraendert, und das ist korrekt: der
  Test analysiert Quelltext statisch und fuehrt keinen Handler aus. Der Wert deckt sich
  exakt mit `coverage_start` der Unit.)
- mutations-probe: Gegenbeweis auf der Spec-Seite gefahren - in `api/openapi.yaml` wurde der
  Response-Key `"200":` unter `POST /api/v1/auth/login` testweise auf `"299":` umbenannt
  (strukturell weiter gueltiges YAML, damit wirklich der neue Test anschlaegt und nicht der
  Validator). `go test -run TestOpenAPIStatusCodeDrift` wurde rot mit genau
  `POST /api/v1/auth/login writes [200] - not documented`. Zurueckgedreht per
  `git checkout -- api/openapi.yaml`, danach wieder gruen; `git status` zeigt die Spec
  unveraendert.
  Zweiter, waehrend des Bauens gefundener Parser-Fehler, der ohne diese Probe durchgegangen
  waere: die erste Fassung des Spec-Scanners akzeptierte nur `"200":`, die Spec mischt aber
  beide Quoting-Stile (`'200':` bei den advisory-protocols-Bloecken). Das erzeugte acht
  falsche 200/201/204-Treffer. Regex auf `['"]?` erweitert, Falschtreffer weg.
- verify vorgaenger: sauber. `e7076da2` (Iteration 92,
  fix-idempotency-409-rollout-non-finance-routes-2) gegen alle acht Fehlerklassen geprueft:
  der Diff beruehrt ausschliesslich `backend/api/openapi.yaml` (+19/-1, zehn neue
  `IdempotencyInFlight`-Refs plus ein Merge-Block auf der share-links-Mint-Operation) - kein
  Go-Code, kein gRPC-Bypass, kein Stub, kein `.proto`, kein neuer `RequirePermission`, keine
  neue Tabelle, keine Wire-Shape-Aenderung, keine neue Route, kein ersetzter Guard.
  `ea3b8f00` direkt danach ist reine Journal-Buchhaltung (1 Zeile SHA).
- neue-units: `fix-status-code-drift-baseline-non-systemic`,
  `fix-204-documented-but-200-written`, `fix-crm-tag-endpoints-are-501-stubs`,
  `doc-status-code-systemic-400-503-sweep` (alle vier am Backlog-Ende, `status: todo`).
- offen: **Die Befundmenge am Tag eins ist gross und in zwei Toepfe geteilt.** Roh meldete
  der Test 1125 Operationen. Davon sind zwei Codes systemisch, weil sie aus den gemeinsamen
  Helfern kommen und fast nirgends dokumentiert sind: **400 auf 347 Operationen** und
  **503 auf 1085**. Die stehen in `systemicUndocumentedCodes` und werden nur gezaehlt und
  geloggt, nicht gemeldet - ein 1100-Zeilen-Literal liest niemand, und ein Test, der am Tag
  eins rot ist, wird abgeschaltet. Fortschritt daran ist am geloggten Zaehler ablesbar
  (`go test -v -run TestOpenAPIStatusCodeDrift`), Arbeitspaket ist
  `doc-status-code-systemic-400-503-sweep`.
  Uebrig bleiben **80 Operationen** als eingefrorene, kommentierte Baseline
  (`statusDriftBaseline` im Test - die vollstaendige Liste steht dort, nach Operation
  sortiert, und ist die Arbeitsliste, die dieses Von-Hand-Suchen ersetzt). Verteilung:
  **401 auf 33** Operationen (aus `ownerFilterForScope`, Schwerpunkt finance und hr),
  **500 auf 42** (Handler mit eigenem Internal-Error statt `respondGRPCError`; caldav,
  dashboard, hr/time, inbox), **502 auf 3** (Slack/Teams-Webhook-Bruecken), **403 auf 1**
  (`GET /api/v1/security/gdpr/exports`), **501 auf 4** und **200 auf 2**.
  Zwei Teilmengen davon sind KEINE Doku-Luecken, sondern echte Befunde und haben deshalb
  eigene Units bekommen:
  1. **204 dokumentiert, 200 geschrieben** - `POST /api/v1/channels/{id}/read` dokumentiert
     `"204": Channel marked as read`, der Handler (`route_chat.go:735`) antwortet
     `response.JSON(w, http.StatusOK, map[string]string{"status": ...})`, also 200 MIT Body.
     Dasselbe bei `DELETE /api/v1/files/{id}`. Das ist eine Vertragsabweichung gegen den
     generierten FE-Typ, nicht bloss ein fehlender Spec-Eintrag
     (`fix-204-documented-but-200-written`).
  2. **Vier dokumentierte CRM-Routen sind dauerhafte 501-Stubs** -
     `POST`/`DELETE /api/v1/activities/{id}/tags` (`route_crm_activities.go:300`, `:312`) und
     `POST`/`DELETE /api/v1/deals/{id}/tags` (`route_crm_pipeline.go:500`, `:512`) antworten
     immer `501 "not implemented via HTTP, use gRPC"`. Laut Spec existieren sie, real
     funktionieren sie nie (`fix-crm-tag-endpoints-are-501-stubs`). Der fuenfte 501-Treffer,
     `route_integration.go:609` ("slack oauth install is not available on this deployment"),
     ist ein absichtliches Deployment-Nein und bleibt.
  Bewusste Grenze des Guards, damit sie niemand fuer einen Ausfall haelt: die
  Gegenrichtung - dokumentiert, aber nie geschrieben - ist NICHT geprueft. Mit
  `respondGRPCError` im Aufrufgraph laesst sich nicht beweisen, dass ein dokumentierter Code
  unerreichbar ist. Der 204-Befund oben ist genau deshalb nur ueber die Vorwaertsrichtung
  (200 undokumentiert) aufgefallen und nicht ueber das nie geschriebene 204.
  `BACKLOG.yml` ist weiterhin kein gueltiges YAML (unquotierte `done_when`-Doppelpunkte aus
  frueheren Iterationen); die vier hier neu angelegten Units quoten ihre betroffenen Zeilen
  korrekt.

## Iteration 94 — fix-idempotency-409-rollout-non-finance-routes-3 — done — 2026-08-23 11:05
- commit: 960611c8
- gebaut: "409 IdempotencyInFlight" auf allen 12 mutierenden Notification-Operationen ergaenzt
  (`/api/v1/notifications/read-all`, `{id}/read`, `{id}/pin` POST/DELETE, `{id}/dismiss`,
  `{id}/snooze`, `preferences` PUT, `mutes` POST, `mutes/{muteId}` DELETE, `quiet-hours` PUT,
  `dnd` POST/DELETE). Alle Pfade liegen unter `authWithIdempotency`, kein Whitelist-Konflikt
  (`/auth/*` nicht betroffen), kein bestehendes Geschaefts-409 zum Mergen gefunden — reiner
  Anhaeng-Fall wie bei Wiki. Lokaler YAML-Stil in diesem Abschnitt ist Multi-Line
  (`"409":` / `  $ref: ...`) statt Inline-Klammern wie bei Formulare — dem Umgebungsstil gefolgt,
  nicht stur ein Muster kopiert.
- gate: build n.a. (kein Go-Code geaendert) | vet n.a. | lint n.a. | test ok
  (`go test -count=1 ./internal/gateway/` gruen, inkl. `TestOpenAPIRouteDrift` und dem neuen
  `TestOpenAPIStatusCodeDrift` aus Iteration 93) | migration n.a. | rls-smoke n.a. (keine Tabelle
  angefasst)
- coverage: n.a. (Doku-Unit, kein Coverage-Ziel, wie in der Unit vorgesehen)
- mutations-probe: Gegenbeweis auf der Spec-Seite gefahren — ein neuer
  `IdempotencyInFlight`-$ref (auf `POST /api/v1/notifications/{id}/read`) testweise auf
  `IdempotencyInFlightTYPO` verbogen. `npx swagger-cli validate api/openapi.yaml` wurde rot mit
  `Token "IdempotencyInFlightTYPO" does not exist.`. Zurueckgedreht, `git diff --stat` zeigt
  wieder genau 24 Netto-Zeilen (12 Operationen x 2 Zeilen), `swagger-cli validate` und
  `go test ./internal/gateway/` danach wieder gruen.
- verify vorgaenger: sauber. `763df3bc` (Iteration 93, `test-openapi-documented-status-codes-vs-handlers`)
  gegen alle acht Fehlerklassen geprueft: der Diff fuegt ausschliesslich eine neue Testdatei
  (`internal/gateway/openapi_status_code_drift_test.go`) plus BACKLOG/JOURNAL hinzu — kein
  Go-Produktionscode, kein gRPC-Bypass, kein Stub im Produktionspfad, kein `.proto`, kein neuer
  `RequirePermission`, keine neue Tabelle, keine Wire-Shape-Aenderung, keine neue Route, kein
  ersetzter Guard. `2406e49e` direkt danach ist reine Journal-Buchhaltung (1 Zeile SHA).
- neue-units: `fix-idempotency-409-rollout-non-finance-routes-4` (Backlog-Ende, `status: todo`,
  aktualisierte Restliste ohne Notification, jetzt gut 33 offene Registrar-Gruppen).
- offen: keine neuen Befunde. Die vier Iteration-93-Folgeeinheiten
  (`fix-status-code-drift-baseline-non-systemic`, `fix-204-documented-but-200-written`,
  `fix-crm-tag-endpoints-are-501-stubs`, `doc-status-code-systemic-400-503-sweep`) stehen
  weiterhin unbearbeitet im Backlog und sind fuer eine der naechsten Iterationen faellig.

## Iteration 95 — fix-status-code-drift-baseline-non-systemic — done — 2026-08-23 10:50
- commit: 2a549d4b
- gebaut: Die 11 caldav-Eintraege aus `statusDriftBaseline` dokumentiert und entfernt (alle
  reine 500er): `GET/POST /api/v1/caldav/passwords`, `DELETE /api/v1/caldav/passwords/{id}`,
  `GET /api/v1/caldav/status`, `POST /api/v1/caldav/test`, `PUT /api/v1/caldav/enable`,
  `PUT /api/v1/caldav/disable`, `GET/PUT /api/v1/admin/caldav/settings`,
  `GET /api/v1/admin/caldav/users`, `DELETE /api/v1/admin/caldav/users/{userId}/passwords`.
  Jede Operation bekam eine eigene `"500": description:` Zeile, deren Text 1:1 aus der
  `response.Error(w, http.StatusInternalServerError, "...")`-Message im jeweiligen Handler in
  `route_caldav.go` uebernommen ist (z. B. "failed to list passwords" -> "Failed to list
  passwords"), analog zum bestehenden Stil (inline description statt $ref, siehe Finance-
  Operationen um Zeile 9176). Kein Go-Code geaendert, nur `api/openapi.yaml` +
  `openapi_status_code_drift_test.go` (Baseline-Map).
- gate: build ok (`go build -p 2 ./internal/gateway/... ./cmd/gateway/...`) | vet ok | lint ok
  (`golangci-lint run ./internal/gateway/...` 0 issues) | test ok (`go test -count=1
  ./internal/gateway/` gruen, 56,6 % - unveraendert, reine Doku-Aenderung) | migration n.a.
  | rls-smoke n.a. (keine Tabelle angefasst)
- coverage: n.a. (Doku-Unit, kein Coverage-Ziel, wie in der Unit vorgesehen)
- mutations-probe: `"500": description: Failed to list passwords` bei
  `GET /api/v1/caldav/passwords` testweise entfernt. `TestOpenAPIStatusCodeDrift` wurde rot
  mit "GET /api/v1/caldav/passwords writes [500] - not documented". Zurueckgedreht, danach
  wieder gruen; `git diff --stat` zeigt wieder genau die urspruengliche Aenderung (22 Zeilen
  openapi.yaml, 11 Zeilen entfernt aus dem Testfile).
- verify vorgaenger: sauber. `960611c8` (Iteration 94, Notification-409-Rollout) gegen alle
  acht Fehlerklassen geprueft: reine `api/openapi.yaml`-Ergaenzung (24 Zeilen, 12 x "409"
  IdempotencyInFlight), kein Go-Code, kein gRPC-Bypass, kein Stub, kein `.proto`, kein neuer
  `RequirePermission`, keine neue Tabelle, keine Wire-Shape-Aenderung, keine neue Route, kein
  ersetzter Guard. `a36f3fbc` direkt danach ist reine Journal-Buchhaltung (1 Zeile SHA).
- neue-units: `fix-status-code-drift-baseline-non-systemic-2` (Backlog-Ende, `status: todo`,
  Restliste 69 Eintraege ohne caldav, Gruppen finance/hr/dashboard/integrations/Rest benannt).
- offen: `TestOpenAPIStatusCodeDrift` zeigt jetzt 69 statt 80 baselined operations (verifiziert
  im Testlog: "checked 1196 documented operations ... 69 baselined operations"). Die vier
  Iteration-93-Folgeeinheiten (`fix-204-documented-but-200-written`,
  `fix-crm-tag-endpoints-are-501-stubs`, `doc-status-code-systemic-400-503-sweep`) und die
  Idempotency-409-Fortsetzung (`fix-idempotency-409-rollout-non-finance-routes-4`) stehen
  weiterhin unbearbeitet im Backlog.

## Iteration 96 — fix-204-documented-but-200-written — done — 2026-08-23 10:56
- commit: a82e0838
- gebaut: Beide Vertragsabweichungen aus `TestOpenAPIStatusCodeDrift` (Iteration 93) behoben —
  Spec und Handler stimmen jetzt ueberein, beide auf `204 No Content`:
  1. `POST /api/v1/channels/{id}/read` — `ChatRoutes.HandleMarkChannelRead` (route_chat.go:736)
     schrieb `response.JSON(w, http.StatusOK, map[string]string{"status": "channel marked as
     read"})`; Spec dokumentiert bereits `"204"` (openapi.yaml:3806). Handler auf
     `w.WriteHeader(http.StatusNoContent)` umgestellt, Stil uebernommen von den ~20 anderen
     Stellen in `internal/gateway`, die denselben Pattern nutzen (z. B. route_auth.go:1025).
  2. `DELETE /api/v1/files/{id}` — `ChatRoutes.HandleDeleteFile` (route_chat.go:869, chat-files-
     Tag) schrieb ebenso 200+Body; Spec dokumentiert `"204"` (openapi.yaml:3966). Gleiche
     Umstellung.
  FE-Nutzung geprueft (Grund fuer die Entscheidung gegen die Default-Empfehlung "Spec an Code
  angleichen"): kein Hook in `desktop/src/renderer/src` liest den Response-Body einer der
  beiden Operationen. `useMarkChannelRead` (api/hooks/useChannels.ts:227) destrukturiert nur
  `error`, ignoriert `data`. Fuer `DELETE /api/v1/files/{id}` (operationId `deleteFile`,
  chat-files) existiert im Desktop-Code aktuell **kein** Aufrufer ueberhaupt (nur
  `route_document.go`s eigenstaendiges, andres `HandleDeleteFile` unter documents/files wird
  vom FE genutzt, ueber `/api/v1/documents/files/{id}` — separater Endpoint, nicht angefasst).
  Da kein FE-Konsument den Body braucht und kein bestehender Test den 200er festschreibt, ist
  der Handler an die Spec angepasst worden statt umgekehrt — sauberere REST-Semantik, kein
  Risiko fuer einen kuenftigen Aufrufer, der auf einen 200er-Body baut, der es nie wert war,
  dokumentiert zu werden.
  Testdatei mitgezogen: `TestOpenAPIStatusCodeDriftParserSanity` (openapi_status_code_drift_test.go:629)
  fixierte bisher explizit, dass `HandleMarkChannelRead` 200 via `response.JSON` schreibt —
  auf 204 via `w.WriteHeader` umgestellt, sonst waere der Sanity-Test nach dem Fix rot
  geworden. Beide Eintraege aus `statusDriftBaseline` entfernt (69 -> 67 verbleibende
  Baseline-Operationen), erklaerender Kommentar zu den zwei 200er-Sonderfaellen entfernt.
- gate: build ok (`go build -p 2 ./internal/gateway/... ./cmd/gateway/...`) | vet ok | lint ok
  (`golangci-lint run ./internal/gateway/...` 0 issues) | test ok (`go test -count=1
  ./internal/gateway/` gruen) | migration n.a. | rls-smoke n.a. (keine Tabelle angefasst)
- coverage: internal/gateway 56,6 % -> 56,6 % (unveraendert — Verhaltensfix, keine neue
  Testabdeckung noetig, bestehende Tests decken beide Handler bereits ab)
- mutations-probe: `HandleMarkChannelRead` testweise zurueck auf
  `response.JSON(w, http.StatusOK, ...)` gesetzt. `TestOpenAPIStatusCodeDrift` wurde rot
  ("POST /api/v1/channels/{id}/read writes [200] - not documented"),
  `TestOpenAPIStatusCodeDriftParserSanity` ebenfalls rot ("expected ... to write 204 ..., got
  map[200:true 400:true 503:true]"). Zurueckgedreht, `git diff --stat` zeigt wieder genau die
  urspruengliche Aenderung (2 Zeilen route_chat.go, 8 Zeilen openapi_status_code_drift_test.go),
  `go test ./internal/gateway/` danach wieder gruen.
- verify vorgaenger: sauber. `2a549d4b` (Iteration 95, caldav-500-Doku) gegen alle acht
  Fehlerklassen geprueft: Diff aendert ausschliesslich `api/openapi.yaml` (22 neue Zeilen) und
  entfernt 11 Eintraege aus der Test-Baseline — kein Go-Produktionscode, kein gRPC-Bypass, kein
  Stub, kein `.proto`, kein neuer `RequirePermission`, keine neue Tabelle, keine
  Wire-Shape-Aenderung, keine neue Route, kein ersetzter Guard.
- neue-units: keine.
- offen: keine neuen Befunde. Die zwei verbliebenen Iteration-93-Folgeeinheiten
  (`fix-crm-tag-endpoints-are-501-stubs`, `doc-status-code-systemic-400-503-sweep`) sowie
  `fix-idempotency-409-rollout-non-finance-routes-4` und
  `fix-status-code-drift-baseline-non-systemic-2` stehen weiterhin unbearbeitet im Backlog.

## Iteration 97 — doc-status-code-systemic-400-503-sweep — done — 2026-08-23 11:03
- commit: a782d41c
- gebaut: Registrar-Gruppe **Inbox** im systemischen 400/503-Sweep vollstaendig geschlossen.
  64 Response-Eintraege in `backend/api/openapi.yaml` nachgetragen: 27x `"400": $ref BadRequest`
  und 37x `"503": $ref ServiceUnavailable`, verteilt auf alle 37 `/api/v1/inbox`-Operationen
  (messages inkl. read/unread/star/status/tags/forward/archive/unarchive/snooze/unsnooze/reply/
  assign/claim/thread, bulk-read, bulk-archive, unread-count, teams inkl. members, rules inkl.
  rules/test, canned-responses). Arbeitsliste wie in den notes vorgesehen erzeugt: Code testweise
  aus `systemicUndocumentedCodes` (openapi_status_code_drift_test.go:430) entfernt, Test rot
  laufen lassen, Fehlermeldung als Liste genommen, Testdatei danach zurueckgesetzt — sie ist im
  Commit unveraendert. Eingetragen wurde ausschliesslich auf den Operationen, die der Test
  wirklich nennt, numerisch einsortiert (400 vor einem vorhandenen 404, 503 ans Ende); die
  bereits dokumentierten 400er (z. B. `POST /messages/{id}/status`) sind nicht doppelt gesetzt.
  Kein Go-Produktionscode angefasst.
- gate: build ok (`go build -p 2 ./internal/gateway/... ./cmd/gateway/...`) | vet ok | lint ok
  (`golangci-lint run ./internal/gateway/...` 0 issues) | test ok (`go test -count=1
  ./internal/gateway/` gruen, 0 uebersprungene Tests von 3849 mit gesetztem `DATABASE_URL`) |
  swagger-cli validate gruen | migration n.a. | rls-smoke n.a. (keine Tabelle angefasst)
- coverage: internal/gateway 56,6 % -> 56,6 % (unveraendert — reine Spec-Doku, kein Go-Code)
- mutations-probe: zwei Proben. (1) `503` testweise aus `systemicUndocumentedCodes` entfernt und
  den Test laufen lassen: 1048 Findings, davon **0** mit `/api/v1/inbox` — vor der Aenderung
  waren es 1085 mit 37 Inbox-Treffern. Das beweist, dass die Eintraege genau dort gelandet sind,
  wo der Test sucht. Testdatei zurueckgesetzt, `git diff` darauf leer. (2) Den neuen
  `$ref: "#/components/responses/ServiceUnavailable"` auf `GET /api/v1/inbox/messages` auf
  `ServiceUnavailableX` verbogen: `swagger-cli validate` rot ("Token \"ServiceUnavailableX\"
  does not exist."). Zurueckgedreht, danach wieder gruen, `git diff --stat` zeigt exakt die
  128 Einfuegungen der eigentlichen Aenderung.
  Zaehler-Beleg fuer die `done_when`-Zeile: `400: 347 -> 320`, `503: 1085 -> 1048`,
  `statusDriftBaseline` unveraendert bei 67 Eintraegen, 1196 dokumentierte Operationen geprueft,
  0 Handler unaufloesbar.
- verify vorgaenger: sauber. `a82e0838` (Iteration 96) gegen alle acht Fehlerklassen geprueft:
  Diff aendert zwei Zeilen in `route_chat.go` (200+Body -> `w.WriteHeader(204)`), die passende
  Sanity-Test-Erwartung und zwei Baseline-Eintraege. Kein gRPC-Bypass, kein Stub, kein `.proto`,
  kein neuer `RequirePermission`, keine Tabelle, keine neue Route. Wire-Shape gegengeprueft
  statt uebernommen: `useMarkChannelRead` (useChannels.ts:229-236) destrukturiert wirklich nur
  `error` und ignoriert `data` — der entfernte Body hat keinen Konsumenten.
- neue-units: `feat-crm-activity-deal-tag-rpcs` (Entscheidungsvorlage aus der geblockten Unit,
  siehe unten), `doc-status-code-systemic-400-503-sweep-2` (Rest des Sweeps mit aktualisierten
  Gruppenzahlen).
- offen: **`fix-crm-tag-endpoints-are-501-stubs` ist auf `blocked` gesetzt und braucht eine
  Entscheidung von Luke.** Recherche dieser Iteration: beide im scope vorgesehenen Wege sind zu.
  Durchreichen geht nicht — `proto/crm/v1/crm.proto` kennt Tag-RPCs nur fuer Contacts (Zeilen
  28/29), `AddActivityTags`/`RemoveActivityTags`/`AddDealTags`/`RemoveDealTags` existieren weder
  als RPC noch als Message, und in `internal/proto/crm/v1/` gibt es keine generierten Typen
  dafuer; vier neue RPCs sind laut den notes der Unit ausdruecklich Lukes Entscheidung.
  Entfernen geht aber ebenso wenig, weil die Grundannahme des scope falsch ist: die Faehigkeit
  IST gebaut. `activity.Service.AddTags/RemoveTags` (activity/service.go:441/466) und
  `deal.Service.AddTags/RemoveTags` (deal/service.go:458/483) sind vollstaendig implementiert,
  tenant-gescoped und pruefen die Tag-Existenz; Repository-Methoden, die Join-Tabellen
  `activity_tags`/`deal_tags` (Migration 000006, `tenant_id` seit 000111), die Guards
  `activities:write`/`deals:write` (route_crm.go:165/166/178/179) und die fertigen
  openapi-Vertraege (`200` + `DealResponse`/`ActivityResponse`, openapi.yaml:2731/2964) sind
  alle da. Es fehlt ausschliesslich der gRPC-Hop. Beide Service-Methodenpaare haben aktuell
  **null Nicht-Test-Aufrufer** (per grep verifiziert) — also fertig gebauter, toter Code.
  Vier spezifizierte, fachlich implementierte Endpunkte zu loeschen, weil eine
  Verdrahtungsschicht fehlt, ist eine Produktentscheidung. Die Umsetzung beider Wege liegt
  fertig beschrieben in `feat-crm-activity-deal-tag-rpcs` (Weg A: vier RPCs nach dem exakten
  Muster von `AddContactTags`, crm_grpc.go:673, plus `make proto-crm` im selben Commit;
  Weg B: Routen, Handler, Spec-Pfade und die toten Service-/Repo-Methoden entfernen). In beiden
  Faellen muessen die vier Tests mitgezogen werden, die den 501er heute festschreiben
  (route_crm_activities_test.go:354/375, route_crm_pipeline_test.go:573/602). FE-seitig ist
  beides risikofrei: `desktop/src/renderer/src` nutzt Tag-Routen nur unter
  `/api/v1/contacts/{id}/tags`.
  Sonst nichts offen — DB-Gate lief mit gesetztem `DATABASE_URL`, keine uebersprungenen Tests.

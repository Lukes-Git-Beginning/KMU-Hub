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

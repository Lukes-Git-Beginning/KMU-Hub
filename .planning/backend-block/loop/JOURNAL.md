# Backend-Nachtloop — Journal Lauf 12

Append-only. Ein Eintrag je Iteration, **ans Dateiende**, nie einsortieren. Form und Pflichtzeilen
stehen in `ITERATION.md` Schritt 6.

Frühere Läufe liegen vollständig im Archiv:
`archive/lauf-1-2/` (58 Units) · `archive/lauf-3/` (61) · `archive/lauf-4/` (54) ·
`archive/lauf-5/` (41) · `archive/lauf-6/` (46) · `archive/lauf-7/` (71) ·
`archive/lauf-8/` (94) · `archive/lauf-9/` (37) · `archive/lauf-10/` (93) ·
`archive/lauf-11/` (131 Einträge, davon 121 done — inkl. Laufbilanz am Dateiende).

---

## Laufkontext

- **Ausgangspunkt:** Lauf 11 gemergt als `acc48aee` und deployt. `main` = `backend-loop`,
  Fast-Forward, **nicht** rebased. CI grün auf `acc48aee` (Run 32735558575, alle fünf Jobs,
  57 Steps), CD grün, Produktion antwortet mit `commit: acc48aee`.
- **Migrationen:** Repo-Kopf = lokaler DB-Kopf = Produktionskopf = **323**, `schema_migrations`
  clean. Nächste freie Nummer wäre 324 — aber immer zur Laufzeit ermitteln:
  `ls backend/migrations | grep -E "^[0-9]{6}" | sort | tail -1`.
  Zwei Units dieses Laufs bringen eine Migration mit (`harden-quote-conversion-unique-index`,
  ggf. `feat-retention-handler-guest-sessions`). Entsteht im Lauf eine weitere, gilt:
  `tenant_id UUID NOT NULL` + RLS-Policy oder ein Eintrag in der System-Global-Liste,
  kein stiller dritter Weg.
- **Lokale DB:** Startbedingung. `run-loop.ps1` prüft im Vorflug Port 5432, die Anmeldung als
  `kmuhub_app` und den Migrationskopf und bricht ab, wenn eins davon fehlt. Grund:
  `testutil.SkipIfNoDB` (`backend/internal/testutil/rls.go:24`) fragt nur, **ob**
  `DATABASE_URL` gesetzt ist — ohne die Variable meldet ein Paket `ok` für Tests, die gar nicht
  gelaufen sind.
- **Rolle:** `kmuhub_app` (NOSUPERUSER NOBYPASSRLS), niemals `kmuhub` — der Superuser hat
  BYPASSRLS und würde jede RLS-Lücke durchwinken.
- **Coverage-Ausgangslage** (CI-Run 32735558575, Artefakt 9523355904): gesamt **64,1 %** bei
  Gate 15 %. Schwächste Kernpakete nach ungedeckten Zeilen: `internal/gateway` **56,6 %**
  (9.882), `internal/server` 70,8 (5.941), `internal/security/gdpr` 72,2 (626),
  `internal/auth` 67,9 (515), `internal/caldav` 54,9 (494), `internal/fuhrpark` 54,5 (464),
  `internal/formulare` 53,6 (381). Vollständige Liste im Kopf von `BACKLOG.yml`.
  **`coverage_start` in einer Unit ist ein Bezugswert, keine Messung** — jede Coverage-Unit
  misst ihr Paket vorher selbst (Lehre aus Lauf 11 Iteration 75).
- **Umfang:** 70 Units vorab, 9 davon auf `opus`. Block A (10 entschiedene) · Block A2 (5, G1-Rest und Funde) ·
  Block B (38, Nicht-Geld-Module end-to-end) · Block C (7, Sicherheits- und Kernflächen) ·
  Block D (10, Muster-Scans). Block D legt weitere Units zur Laufzeit an; in Lauf 10 haben
  9 Scans 45 Zusatz-Units erzeugt, in Lauf 11 waren es 10 Scans.
  Fenster bis 07:30, `-MaxIterations 130`.

## Der rote Faden

**NICHT-GELD-MODULE END-TO-END.** Entscheidung Luke, 2026-08-24. Lauf 11 hat die Geldpfade
abgearbeitet (payment 46,4 → 85,3 · dunning 61,8 → 92,2 · invoice 34,8 → 61,1); übrig geblieben
sind genau die Module, die dabei nicht drankamen — und die sind jetzt die schwächsten Flächen
im Backend. Je Unit **eine Domäne durch alle Schichten** (Route → gRPC → Service → Repo), als
**Bug-Suche** geschnitten, nicht als Coverage-Übung.

Warum nicht G1 oder G2 wie im Entwurf vorgesehen: beide Gates sind backend-seitig fast
abgeräumt. Von 16 G1-Punkten sind 7 erledigt, vom Rest kann der Loop genau einen bauen — alles
Übrige ist Frontend, `deploy/`, Ops oder Legal. Die Gegenprüfung steht als Befunde 1 bis 10 im
Kopf von `BACKLOG.yml`.

## Was in diesem Lauf gilt

- **Zehn Prämissen des Entwurfs haben die Gegenprüfung nicht überstanden** und stehen als
  Befunde 1 bis 10 im Kopf von `BACKLOG.yml`. Vor dem Bauen lesen — fünf davon sind Dinge, die
  bereits fertig im Code stehen (`/health` mit Postgres-Checker, Retention-Worker mit neun
  Handlern, DSAR über 39 Tabellen, GoBD-WORM per `REVOKE`, Passwort-Reset-URL). Wer sie
  trotzdem baut, baut doppelt und merkt es nicht, weil „gebaut" und „grün" dann beide stimmen.
- **Block B ist Bug-Suche, nicht Coverage.** Gesucht wird: Tenant-Scoping auf der READ-Seite,
  fehlende Fehlerpfade, Business-Logik im Handler, blinde Summierungen über Währungen,
  fehlende Indizes auf Join-Spalten. Coverage ist das Nebenprodukt und gehört in die
  `coverage:`-Zeile. Vorbild sind die beiden `idempotency`-Units aus Lauf 10 (`6507e475`,
  `254120eb`) — die erste hat sofort einen Produktionsfehler gefunden.
- **Neue DB-Tests ungetagt schreiben.** Kein `//go:build integration`. Bausteine:
  `testutil.SkipIfNoDB`, `PoolFromEnv`, `EnsureTenant`, `SeedRow`, `CleanupRow`,
  `WithTenantCtx`. Vorlage: `backend/internal/idempotency/postgres_repository_db_test.go`.
  Getaggte Tests laufen weder im lokalen Gate noch im Coverage-Job.
- **Ein DB-Test, der lokal grün ist, weil der Pool nur eine warme Verbindung hatte, beweist
  nichts.** Wer eine Ressource *pro Verbindung* prüft (Advisory Locks, Session-GUCs, temporäre
  Tabellen), hält vorher eine zweite Verbindung fest.
- **Wer ein bestehendes Muster als Vorlage kopiert, kopiert seine Fehler mit.** Vorlage vorher
  prüfen, nicht nur nachbauen.
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
  gebaut** — sie wird hier widerlegt und die Unit auf `blocked` gesetzt, **im selben Commit**,
  mit `blocked_reason`. Kommentarlos überspringen ist ein Fehlschlag der Iteration.
- Scan-Units ändern kein Verhalten. `neue-units:` muss IDs nennen, die wirklich in
  `BACKLOG.yml` stehen. Ein abgebrochener Scan nennt in `offen:`, **was** tief geprüft, **was**
  nur gegrept und **was** gar nicht angesehen wurde.
- Gesperrt: Frontend/Desktop, CSAT und Public-Token-Routen, `internal/biz/bexio`,
  Dependency-Bumps, Migrations-Umnummerierungen, Preise und Modul-Zuschnitt.
  `RETENTION_MODE` bleibt `dry_run`. `deploy/` ist gesperrt **bis auf zwei namentlich
  freigegebene Ausnahmen** — die Backup-Regel in `alerts.yml` und die SMTP-Vorlagen-Angleichung,
  beide als eigene Unit in Block C, beide ohne Compose-Änderung und ohne neues Env.

---

# Lauf 15 vorbereiten

> **Auftrag dieser Sitzung:** den Backlog für Lauf 15 ausschreiben. Drei Dinge, in dieser
> Reihenfolge: erst die **mechanischen Pflichten** (Branch-Stand, Coverage-Basis, DB-Gate), dann
> den **roten Faden entscheiden** (Luke), dann die **Units schreiben** — jede gegen den
> verifizierten Code, nicht gegen Notizen.
>
> **Was diese Sitzung nicht macht:** Units bauen. Kein Merge, kein Deploy. Am Ende steht ein
> Backlog, den der Treiber ziehen kann, und ein Startkommando.

---

## Ausgangslage (2026-10-23, am Code und an der Produktion gemessen)

- `main` = `backend-loop` = **`ea5a39a9`**. Produktion: `ea5a39a9` (CD 2026-10-23 13:10 UTC, `/health` bestätigt).
- **Migrationskopf 332**, Repo = lokale DB = Produktion, `dirty = f`. Nächste freie Nummer
  **zur Laufzeit ermitteln**: `ls backend/migrations | grep -E "^[0-9]{6}" | sort | tail -1`
- `BACKLOG.yml`: **leer** (`units: []`, Kopf für Lauf 15 steht schon). Lauf 14 ist archiviert
  (`loop/archive/lauf-14/`), `JOURNAL.md` ist auf den Lauf-15-Kopf zurückgesetzt.
- `BACKLOG-NEXT.yml`: 26 Units, davon **6 `todo`** (unten), 20 `blocked` (Entscheidungen Luke,
  Legal, FE). `BACKLOG-PARKED.yml`: 15.
- Kontext außerhalb des Codes: 1 Tenant, Registrierung zu, seit 22.09. keine echten Logins,
  kein Pilot in Sicht. Empfehlung der Gesamtbewertung 2026-10-20: 90-Tage-Test (fokussierter
  Hybrid) bis Ende Jan 2027. **Ein Lauf muss sich deshalb rechtfertigen** — mehr Härtung ohne
  Nutzer ist kein Selbstzweck.

### Die sechs entschiedenen bzw. baubaren Units in `BACKLOG-NEXT.yml`

| Unit | Modell | Herkunft |
|---|---|---|
| `feat-gobd-verify-integrity-on-export` | sonnet | Entscheidungsrunde 2026-10-23 |
| `fix-team-inbox-manage-permission-bypass` | sonnet | Entscheidungsrunde 2026-10-23 |
| `chore-dialer-stale-contact-default-240` | sonnet | Entscheidungsrunde 2026-10-23 — ⚠ `deploy/`-Ausnahme, eine Zeile |
| `fix-dsar-third-party-matching` | opus | Entscheidungsrunde 2026-10-23 |
| `decide-produktion-capacity-week-bounds-tenant-timezone` | sonnet | trägt Empfehlung (b), Status `todo` — **prüfen, ob die Entscheidung dokumentiert ist**, sonst bleibt sie draußen |
| `wire-biz-event-emitters-for-finance-triggers` | sonnet | Entscheidung 2026-08-24 — laut NEXT-Kopf „nach BACKLOG.yml gezogen", steht aber noch in NEXT. **Verify-first**: ist sie in Lauf 12–14 schon gebaut worden? |

---

## Was Lauf 14 und seine Nachbearbeitung gelehrt haben (gilt für diesen Backlog)

1. **Der Loop prüft gegen Backend-Aufrufer, nie gegen das Desktop-FE.** Die Nachbearbeitung fand
   eine Wiki-Validierung, die den FE-Marker `public` mit 400 ablehnte, und einen Gateway-Endpunkt
   (`GET /inbox/rules`), dessen Antwortform den Regel-Editor mit einem TypeError abstürzen ließ —
   seit Monaten, weil nur die MSW-Mocks die richtige Form lieferten.
2. **Tests haben einen Totalausfall festgeschrieben.** `calendar_grpc.go` reichte an acht Stellen
   `uuid.Nil` als Akteur durch („Gateway handles auth"); Kalender ändern/löschen, Mitglieder,
   Termin löschen, Erinnerungen und Kategorie löschen gaben **jedem** Nutzer 403. Ein früherer
   Lauf hatte das gefunden — und als `FUND (nicht Teil dieser Unit, nur dokumentiert)` in einen
   Testkommentar geschrieben, die Happy Paths mit `OwnerID: uuid.Nil` grün gemacht und **keine
   Unit angelegt**. Behoben erst in `ea5a39a9`, entdeckt nur, weil der Smoke-Aufräum-DELETE 403
   bekam. → Ein Fund, der einen Nutzerpfad für alle bricht, ist **nie** „nur dokumentiert".
3. **`-race` sieht nur die CI.** Lauf 13 baute zwei Data Races, Lauf 14 zwei flaky Tests
   (bcrypt-Kosten unter `-race`, Goroutine-Baseline im Teardown). Lokal grün heißt nichts.
4. **Smoke-Rot rollt Prod zurück.** Der erste Deploy heute (`81a8ba24`) fiel an einem
   Skript-Fehler (Join-Token ohne `Idempotency-Key`, Prod läuft `IDEMPOTENCY_MODE=hard`).

---

## Teil 0 — Mechanische Pflichten, bevor irgendeine Unit geschrieben wird

### 0.1 · Branch-Stand
`backend-loop` vor dem Lauf auf `origin/main` **mergen, nicht rebasen**. Heute sind beide gleich;
prüfen, ob seitdem etwas auf `main` kam:
```bash
git fetch origin && git log --oneline backend-loop..origin/main
```

### 0.2 · `coverage_start` gegen den CI-Lauf des aktuellen `main`-Kopfs
Nicht aus Lauf 14 kopieren.
```bash
gh run list --branch main --workflow ci.yml --limit 3 --json databaseId,headSha
# Coverage-Zahlen aus dem Test-Job-Log dieses Laufs
```

### 0.3 · Roten Faden und Blockfolge in den Kopf von `BACKLOG.yml`
Der Kopf ist die **einzige** Stelle, an der Faden, Blockfolge und Sperren des Laufs stehen
(`ITERATION.md` verweist dorthin). Er trägt heute nur „STAND 2026-10-23: LEER".

---

## Teil 1 — Der rote Faden (Entscheidung Luke)

### Empfehlung: **„Funktioniert es für einen echten Nutzer?" — Totalausfälle statt Härtung**

Begründung: 14 Läufe haben das Backend weit über den Bedarf ohne Nutzer hinaus gehärtet. Was
heute noch gefunden wurde, war keine Härtungslücke, sondern **Funktionen, die für jeden
scheitern** — und genau die würde der erste Pilotkunde in der ersten Stunde treffen. Diese
Klasse ist für den Loop gut baubar (Backend, testbar, kein Produktionszugriff) und zahlt direkt
auf den 90-Tage-Test ein. Kleiner Lauf, klarer Fokus.

**Drei Scans als Kern, jeder mit Verify-first:**

1. **`uuid.Nil`-Akteur in gRPC-Servern.** Heute verifiziert (Stand `ea5a39a9`): noch rund 20 Stellen,
   v. a. `biz_grpc.go` (Quote/Invoice/CreditNote/Dunning `Send`, `Cancel`, `CreateFromQuote`) und
   `work_grpc.go` (Projekte: `Get/List/Update/Archive/Delete/AddMember/RemoveMember/...`).
   **Nicht automatisch Bugs:** `work` übergibt `isAdmin=true` (Prüfung umgangen, aber der Nutzer
   fehlt in Audit/Aktivität), bei `biz` kann der Akteur nur Attribution sein (`sent_by`). Je Stelle
   klassifizieren: (a) blockiert jeden → Fix wie `ea5a39a9` mit `callerID(ctx)`,
   (b) nur Attribution fehlt → Fix, wenn der Wert persistiert wird, (c) bewusst → `lean:`-Marker.
   ```bash
   grep -rnE "\(ctx, [^)]*uuid\.Nil" backend/internal/server/*_grpc.go | grep -v _test
   ```
2. **FE↔Gateway-Antwortform.** Je Desktop-API-Client (`desktop/src/renderer/src/api/*-client.ts`)
   den typisierten Rückgabetyp gegen das vergleichen, was der Gateway-Handler wirklich schreibt
   (`response.Proto(w, …, resp)` mit Wrapper-Message vs. `ProtoList`/`resp.GetX()`). Muster des
   Fehlers: `3c13fd66`. Die MSW-Mocks sind **kein** Beleg — sie zeigen, was das FE erwartet, nicht
   was das Backend liefert. Fix sitzt am Gateway (Backend-Territorium); FE-Änderungen sind tabu.
3. **Validierungs-Verschärfungen gegen FE-Aufrufer.** Für jede in Lauf 12–14 verschärfte
   Eingabevalidierung die Desktop-Aufrufer greppen, ob sie noch passende Werte senden
   (Muster: `40374f53`, Wiki-Marker `public`).

Plus die entschiedenen Units aus der Tabelle oben.

### Alternativen
- **Kein Lauf.** Legitim, solange es keinen Pilotkunden gibt — die Scans oben ließen sich auch
  in einer gezielten Tagessitzung erledigen.
- **Gateway-Coverage / G2-Rest** wie in früheren Läufen — eher nicht, siehe Begründung oben.

---

## Teil 2 — Regeln, gegen die der Backlog geschrieben werden MUSS

### Neu aus Lauf 14 (in `ITERATION.md` bzw. im Backlog-Kopf verankern)
- **Kein „nur dokumentiert" für Totalausfälle.** Findet eine Iteration einen Pfad, der für jeden
  echten Nutzer scheitert, legt sie im selben Commit eine Unit **vor** die erste `todo`-Unit an
  (wie seit Lauf 13 für Tenant-/Auth-Befunde). Ein Test darf einen solchen Fehler nicht mit
  Workaround-Daten (`uuid.Nil`-Owner o. ä.) grün machen.
- **Jede Validierungs- oder Antwortform-Änderung am Gateway greppt die Desktop-Aufrufer**
  (`desktop/src/renderer/src/api/`) und nennt das Ergebnis im Journal.

### Unverändert (der Vorflug prüft es selbst)
```bash
python .planning/backend-block/loop/hooks/backlog-check.py --preflight
python .planning/backend-block/loop/hooks/backlog-check.py --state
```
- Kein `blocked` / `blocked_reason` in `BACKLOG.yml`; Entscheidungsbedürftiges bleibt in NEXT.
- **Dateireihenfolge ist die einzige Priorität**, und `MODEL` kommt von der ersten baubaren Unit —
  `opus`-Units so platzieren, dass sie ihren Anteil bekommen, nicht vorne aus Versehen.
- Kein Push, kein PR, kein Produktionszugriff (Guard). Keine neue `config.RequireX`-Assertion.
- Neue Tabelle: `tenant_id NOT NULL` + RLS + tenant-gescopte INSERTs **und** SELECTs; jeder neue
  `RequirePermission`-Guard braucht seine Seed-Migration.
- `DATABASE_URL` als `kmuhub_app`, sonst ist `go test` kein Gate.
- `deploy/` ist gesperrt — **einzige Ausnahme** in diesem Lauf: die eine Compose-Zeile von
  `chore-dialer-stale-contact-default-240`.

---

## Teil 3 — Zuschnitt (Entscheidung Luke)

1. **Umfang.** Lauf 14: 108 Units in 459 min (~4,2 min/Iteration). Für den empfohlenen Faden
   reichen vermutlich **20–40 Units** und ein kürzeres Fenster — die Scans erzeugen ihre
   Folge-Units selbst.
2. **opus/sonnet.** Die Scans sind Lese- und Klassifikationsarbeit (sonnet genügt meist);
   `fix-dsar-third-party-matching` bleibt opus.
3. **Nachbearbeitung einplanen.** Jeder Lauf brauchte bisher einen halben Tag Review, PR, CI-Fixes
   und Deploy. Den Lauf nur starten, wenn dieser halbe Tag danach auch da ist.

---

## Verifikation am Ende dieser Sitzung

```bash
# 0 · lokale Postgres an und auf dem Repo-Kopf (Docker Desktop muss laufen)
docker start docker-postgres-1
docker exec docker-postgres-1 psql -U kmuhub -d kmuhub -tAc "select version, dirty from schema_migrations"

# 1 · Backlog lädt und erfüllt die Vorflug-Regeln
python .planning/backend-block/loop/hooks/backlog-check.py --preflight
python .planning/backend-block/loop/hooks/backlog-check.py --state   # NEXT/MODEL plausibel?

# 2 · Guard
bash .planning/backend-block/loop/hooks/test-loop-guard.sh

# 3 · Treiber trocken
powershell -ExecutionPolicy Bypass -File .planning\backend-block\loop\run-loop.ps1 -DryRun -MaxIterations 1

# 4 · Ein Commit (docs(loop): stage run 15), Push
```

## Startkommando für die Nacht (Werte an den Zuschnitt anpassen)

```powershell
powershell -ExecutionPolicy Bypass -File .planning\backend-block\loop\run-loop.ps1 `
  -MaxIterations 60 -UntilTime "07:30" -BudgetUsd 12 -Effort high
```

Der Treiber pusht am Ende einmal auf `backend-loop`. Den PR macht Luke bewusst — vorher beide
Review-Workflows deaktivieren (`gh workflow disable "Claude PR Review"` und `"Security Review"`;
der Auto-Mode-Klassifikator blockt das für Claude, also per `!` selbst ausführen), nach dem
Merge wieder `enable`.

---

## Belege

- `.planning/backend-block/loop/BACKLOG-NEXT.yml` — Block „NACHBEARBEITUNG LAUF 14" mit der
  Entscheidungsrunde 2026-10-23
- `.planning/backend-block/loop/archive/lauf-14/` — Journal und Backlog von Lauf 14
- Commits `076585f0` (Reset-Tx), `3c13fd66` (Routing-Regeln-Form), `336eb7b8` (Idempotency 5xx),
  `a50ddfd1` (DSAR-Kürzungshinweise), `24a3cb7e` (Smoke-Key), `ea5a39a9` (Kalender-Akteur)
- `.planning/backend-block/PROMPT-VORBEREITUNG-LAUF-12.md` — Vorlage dieser Datei
- `.planning/launch-lagebild-2026-08-12.md` — Lage und Gates (SSoT)

# Lauf 12 vorbereiten

> **Auftrag dieser Sitzung:** den Backlog für Lauf 12 ausschreiben. Drei Dinge, in dieser
> Reihenfolge: erst die **mechanischen Pflichten** (Archiv, Journal-Reset, Coverage-Basis), dann
> den **roten Faden entscheiden**, dann die **Units schreiben** — jede gegen den verifizierten
> Code, nicht gegen Notizen.
>
> **Was diese Sitzung nicht macht:** Units bauen. Kein Merge, kein Deploy. Am Ende steht ein
> Backlog, den der Treiber ziehen kann, und ein Startkommando.

---

## Ausgangslage (2026-08-24, alles am Code und an der Produktion gemessen)

- `main` = `backend-loop` = **`acc48aee`**, per Fast-Forward gemergt und deployt.
  `/health` meldet `commit: acc48aee`, `status: healthy`, Redis und Postgres gesund,
  23 Services registriert. CI grün (5 Jobs, 57 Steps), CD grün.
- **Migrationskopf 323**, Repo = lokale DB = Produktion, `schema_migrations` clean.
  Nächste freie Nummer wäre 324 — aber **immer zur Laufzeit ermitteln**:
  `ls backend/migrations | grep -E "^[0-9]{6}" | sort | tail -1`
- `BACKLOG.yml`: **121 done, 10 todo, 0 blocked** · `BACKLOG-NEXT.yml`: 1 blocked (Legal,
  Etappe 4) · `BACKLOG-PARKED.yml`: 15.

**Die zehn `todo`-Units sind bereits entschieden** (Entscheidungsrunde 2026-08-24, jede trägt
`decided:` + `decision:` mit Datum und Begründung). Sie sind der **Kern von Lauf 12**, kein
Rohmaterial:

| # | Unit | Modell |
|---|---|---|
| 1 | `fix-hr-manual-entry-idempotency-key-not-enforced` | sonnet |
| 2 | `harden-quote-conversion-unique-index` | opus |
| 3 | `feat-crm-activity-deal-tag-rpcs` | opus |
| 4 | `feat-lexware-store-organization-id-on-connect` | sonnet |
| 5 | `fix-event-payload-missing-tenant-id` | opus |
| 6 | `wire-biz-event-emitters-for-finance-triggers` (deps: 5) | sonnet |
| 7 | `feat-retention-handler-fuhrpark-operational` | sonnet |
| 8 | `harden-advisory-protocols-retention-guard` | sonnet |
| 9 | `feat-retention-handler-guest-sessions` | sonnet |
| 10 | `fix-409-double-meaning-on-grpc-conflict-routes` | sonnet |

Nummer 5 ist ein **Produktionsfehler auf den heute verdrahteten Pfaden**, keine Vorarbeit:
20 von 25 `models.EventPayload`-Literalen setzen `TenantID` nicht, `bus.dispatch`
(`notification/event/bus.go:133`) kann den Tenant deshalb nicht stempeln, und der
RLS-geschützte Handler scheitert still. Das betrifft `crm/deal`, `work/task`, `chat/message`
heute schon.

---

## Teil 0 — Mechanische Pflichten, bevor irgendeine Unit geschrieben wird

### 0.1 · Lauf 11 archivieren und das Journal leeren

`archive/lauf-11/` **existiert noch nicht**. Konvention aus `archive/lauf-10/`:
`{BACKLOG.yml, JOURNAL.md, logs}`.

```bash
cd .planning/backend-block/loop
mkdir -p archive/lauf-11
cp BACKLOG.yml JOURNAL.md archive/lauf-11/
cp -r logs archive/lauf-11/logs
```

**Das Journal-Leeren ist nicht Kosmetik, sondern mechanisch nötig.** `run-loop.ps1` liest die
höchste `## Iteration N`-Überschrift als Fortschrittsanzeige und vergleicht sie mit seinem
eigenen Zähler. Bleiben Lauf 11s 120 Einträge stehen, meldet der Treiber ab Iteration 1 in
**jeder** Iteration `DRIFT: hoechste Journal-Nummer ist 120, Treiber-Iteration ist 1` — und eine
Warnung, die immer feuert, liest niemand mehr. Genau dieser Fehlermodus ist in Lauf 8 schon
einmal aufgetreten (Minuten-Drift, 90 von 94 Iterationen).

`JOURNAL.md` also auf den Kopf-Block zurücksetzen (Laufkontext für Lauf 12), Einträge raus.

### 0.2 · Bilanz für Lauf 11 nachziehen

Der Treiber schreibt sie ab Lauf 12 selbst. Für Lauf 11 einmal von Hand, damit die archivierte
Fassung eine Zahl trägt statt einer Rekonstruktion:

```bash
python .planning/backend-block/loop/hooks/run-summary.py \
  --base-sha f87ffdcf --from 1 --to 120 --minutes 885 --dry-run
```
Verifiziertes Ergebnis (deckt sich mit der Handzählung der Merge-Sitzung):
`fix 49 · cov 39 · scan 10 · doc 10 · feat 7 · verify 4 · test 1`, 7,4 min/Iteration,
`37 von 70 offen:-Zeilen mit Entscheidungsbedarf`.

### 0.3 · Roten Faden in den Kopf von `BACKLOG.yml` schreiben

`ITERATION.md` hatte bis `acc48aee` einen ausgeschriebenen laufspezifischen Block — **eingefroren
auf Lauf 8, Stand 2026-08-10**. Lauf 11 lief elf Wochen später mit einer völlig anderen
Blockstruktur gegen diesen Prompt. Der Block ist raus und zeigt jetzt auf den Kopf von
`BACKLOG.yml`. **Damit ist der Backlog-Kopf die einzige Stelle, an der Roter Faden, Blockfolge
und Sperren dieses Laufs stehen** — er muss gepflegt werden, sonst steht dort nichts.

### 0.4 · `coverage_start` gegen den CI-Lauf von `acc48aee` nachziehen

Nicht aus Lauf 11 kopieren. Die Paketwerte haben sich durch 121 Units bewegt.

```bash
gh run list --branch main --workflow ci.yml --limit 3 --json databaseId,headSha
# Coverage-Zahlen aus dem Test-Job-Log des Laufs auf acc48aee
```

---

## Teil 1 — Der rote Faden (Entscheidung Luke)

### Empfehlung: **G1 — „vor dem ersten echten personenbezogenen Datensatz"**

Begründung: Lagebild §4 nennt G1 als das, was zwischen dem heutigen Stand und dem ersten echten
Kundendatensatz steht. Etappe 2 ist gebaut, Gate 2 fehlt nur noch die Abnahme; danach kommt
Etappe 3/G1. Der Nachtloop kann von dieser Liste **den größeren Teil selbst erledigen** — es ist
Backend, es ist verifizierbar, und es braucht weder Produktionszugriff noch einen Anwalt.

**⚠ Verify-first ist hier keine Floskel.** Ich habe vier der sechzehn G1-Punkte stichprobenartig
gegen den Code geprüft — **drei davon sind längst erledigt**:

| G1-Punkt laut Lagebild | Tatsächlicher Stand |
|---|---|
| „`/health` prüft nur Redis" | **Erledigt.** `cmd/gateway/main.go:136-137` registriert `NewRedisChecker` **und** `NewPostgresChecker`. Die Live-Antwort zeigt beide. |
| „Aufbewahrungs-Policy ist eine Eingabemaske ohne Wirkung, kein Worker" | **Erledigt.** `security/gdpr/retention.go` ist die Engine, `retention_scheduler.go:133` der Worker, neun Handler sind registriert. |
| „Art.-15-Auskunft deckt 3 von ≥6 Datentöpfen ab" | **Weitgehend erledigt.** `dsar_search.go` fragt inzwischen über 20 Tabellen ab (contacts, deals, activities, email_messages, finance_invoices, hr_*, contracts, driver_licenses, form_submissions, dialer_call_sessions …). |
| „`AnonymizeContact` löscht nur 2 von ≥6 Tabellen" | **Teilweise.** `crm/consent/postgres_repository.go` trägt jetzt 5 UPDATE/DELETE-Anweisungen statt 2 — die genaue Restlücke muss gezählt werden. |

**Konsequenz für diese Sitzung: jeden G1-Punkt einzeln am Code gegenprüfen, bevor er eine Unit
wird.** Die drei erledigten gehören mit Beleg ins Lagebild zurückgeschrieben, nicht in den
Backlog. Sonst baut Lauf 12 dreimal etwas, das schon dasteht — und niemand merkt es, weil
„gebaut" und „grün" dann beide stimmen.

**Was der Loop von G1 kann** (nach Verifikation): `AnonymizeContact`-Restlücke, GoBD-Archiv
wirklich unveränderbar machen (Trigger/REVOKE statt Anwendungsdisziplin — Migration + RLS,
also `opus`), `restore.sh` und `rollback.sh` reparieren, Backup-Alert in `alerts.yml`,
SMTP-/Captcha-Defaults auf EU, Passwort-Reset-URL an der Wurzel (`docker-compose.yml:123`
überschreibt den korrekten Go-Default aus `config.go:204`).

**Was er nicht kann:** alles mit Frontend-Anteil (Consent-Plumbing über `contact_id`,
Kontakt-Löschung-UI), Backups auf fremden Speicher, RUNBOOK, Secret-Rotation.

⚠ **Vorsicht beim Passwort-Reset-Punkt:** `docker-compose.yml` anzufassen ist ein
Deploy-Hazard. Wenn, dann als eigene Unit mit ausdrücklicher Notiz, dass Compose-Passthrough und
Werte **vor** dem Merge geprüft werden.

### Alternativen, falls G1 nicht der Faden sein soll

- **Gateway-Coverage.** `internal/gateway` ist mit ~56,6 % weiter das schwächste Kernpaket.
  Als Bug-Suche schneiden, nicht als Coverage-Übung — das Muster aus Lauf 11 Block B hat
  funktioniert.
- **G2-Rest** (Lagebild §4): DATEV-EXTF schreibt UTF-8 statt Windows-1252 (Mojibake beim
  Steuerberater), EN-16931 ohne Schematron, Route-Dateien ohne Test in Geldpfaden.
- **Frontend-/Vertrags-Parität.** Braucht aber Desktop-Arbeit, die der Loop nicht macht.

---

## Teil 2 — Woher die Units kommen (vier Quellen)

1. **Die zehn entschiedenen Units.** Stehen schon drin. Nur Reihenfolge und `coverage_start`
   nachziehen.
2. **Die G1-Liste** aus `.planning/launch-lagebild-2026-08-12.md` §4 — jeden Punkt verifizieren
   (siehe oben).
3. **Die `offen:`-Zeilen aus Lauf 11.** 70 nicht-leer, 37 mit Entscheidungsbezug.
   ```bash
   grep -n "^- offen:" .planning/backend-block/loop/JOURNAL.md | grep -viE "^\s*$|keine"
   ```
   ⚠ **Vorher deduplizieren.** Die Zahl ist aufgebläht: `harden-lexware-webhook-...` allein steht
   dutzendfach drin, weil die Unit in jeder Iteration erneut übersprungen wurde. Sie ist
   inzwischen entschieden und geparkt. Erst dedupen, dann prüfen, ob der Punkt noch offen ist.
4. **Coverage-Schwachstellen** aus dem CI-Lauf von `acc48aee`.

Für jeden Fund gilt die Lehre aus Lauf 11: **ein Journal-Eintrag ist kein Backlog-Eintrag.**
Was nicht als Unit dasteht, ist mit dem nächsten Lauf weg.

---

## Teil 3 — Regeln, gegen die der Backlog geschrieben werden MUSS

Neu seit `acc48aee`. Wer sie verletzt, merkt es beim Start — der Vorflug bricht ab, nicht der
Lauf mittendrin.

### Der Vorflug prüft es selbst
```bash
python .planning/backend-block/loop/hooks/backlog-check.py --preflight
python .planning/backend-block/loop/hooks/backlog-check.py --state
```
`--preflight` bricht ab (Exit 1) bei:
- nicht ladbarem YAML in **einer der drei** Dateien (nennt Datei, Zeile, Spalte),
- einer Unit mit `status: blocked` **oder** einem `blocked_reason`-Feld in `BACKLOG.yml`,
- einem `done_when[0]`, das eine Entscheidung von Luke verlangt (`Luke hat`,
  `ENTSCHEIDUNG GEHOERT LUKE`),
- `deps` auf nicht existierende IDs oder Zyklen.

**Entscheidungsbedürftiges gehört deshalb gar nicht erst in `BACKLOG.yml`** — sondern nach
`BACKLOG-NEXT.yml` (späterer Lauf) oder `BACKLOG-PARKED.yml` (verworfen/geparkt, mit `decided`,
`decision`, `target`).

### Die Blockreihenfolge steuert jetzt die Modellkosten
`--state` liefert dem Treiber `OPEN=`, `NEXT=` und `MODEL=` — und `MODEL` kommt von der **ersten
baubaren** Unit (`status: todo` **und** alle `deps` auf `done`). Wo eine `opus`-Unit im Backlog
steht, entscheidet also, wie viele Iterationen auf opus laufen. In Lauf 11 hat genau das neun
Iterationen Opus für YAML-Einrückung gekostet.

Der Treiber warnt seit `acc48aee` selbst: weicht die Unit aus der Journal-Überschrift von der
geplanten ab, steht `MODELL-DRIFT` bzw. `UNIT-DRIFT` im `run.log`. **Morgens danach greppen** —
das ist die Messung, die vorher gefehlt hat.

### Der Agent darf nicht mehr überspringen
`ITERATION.md` Schritt 2: Kann er die erste baubare Unit nicht bauen, setzt er sie **im selben
Commit** auf `blocked` mit `blocked_reason`. Kommentarlos weiterziehen ist ein Fehlschlag der
Iteration. Der Kopf heilt sich damit nach einer Iteration statt nach 35.

### Die Doku-Ketten sind Geschichte
`doc-status-code-systemic-400-503-sweep-*` und `fix-idempotency-409-rollout-non-finance-routes-*`
sind geparkt und durch `hooks/openapi-status-fill.py` ersetzt (1649 Spec-Einträge in zwei Läufen
statt zwanzig Nachtstunden). **Nicht wieder als Kette anlegen.** Bleibt Restdrift, wird sie
**eine** spitze Unit.

### Unverändert gültig
- Kein Push, kein PR, kein Produktionszugriff — der Guard blockt hart (`test-loop-guard.sh`
  muss grün sein, der Treiber prüft es).
- Keine neue `config.RequireX`-Assertion, kein Scharfschalten neuer `modules.*`-Flags.
- Neue Tabelle: `tenant_id NOT NULL` + RLS-Policy + tenant-gescopte INSERTs **und SELECTs**.
- Jeder neue `RequirePermission`-Guard braucht seine Seed-Migration.
- `DATABASE_URL` als `kmuhub_app` ist Pflicht, sonst ist `go test` kein Gate.

---

## Teil 4 — Zuschnitt (Entscheidung Luke)

1. **Umfang.** Lauf 11 waren 121 Units in 885 Minuten (7,4 min/Iteration). Wie lang soll die
   Nacht sein, und wie viele Units werden ausgeschrieben? Faustregel aus den letzten Läufen:
   Backlog etwa 60–90 Units, der Rest fällt im Lauf als Folge-Unit an.
2. **opus/sonnet-Mischung.** Erstmals messbar. Wie viel Opus-Anteil ist gewollt — und stehen die
   `opus`-Units so im Backlog, dass sie ihn auch bekommen?
3. **Coverage-Gate.** Steht bei 15 % (CI-erzwungen), gemessen sind ~62,7 %. Anheben, damit ein
   Rückschritt auffällt?
4. **Scan-Anteil.** Lauf 11 hatte 10 Scans, die den Großteil der Folge-Units erzeugt haben.
   Wieder so?

---

## Verifikation am Ende dieser Sitzung

```bash
# 1 · Backlog lädt und erfüllt die Vorflug-Regeln
python .planning/backend-block/loop/hooks/backlog-check.py --preflight
python .planning/backend-block/loop/hooks/backlog-check.py --state   # NEXT/MODEL plausibel?

# 2 · Guard
bash .planning/backend-block/loop/hooks/test-loop-guard.sh

# 3 · Treiber trocken — läuft Vorflug, DB-Gate und Backlog-Gate durch?
powershell -ExecutionPolicy Bypass -File .planning\backend-block\loop\run-loop.ps1 -DryRun -MaxIterations 1

# 4 · Ein Commit, Push
```

Vorher muss die lokale Postgres laufen und auf dem Repo-Migrationskopf stehen, sonst bricht das
DB-Gate ab (korrekt):
```bash
docker compose -f deploy/docker/docker-compose.yml up -d postgres
docker exec docker-postgres-1 psql -U kmuhub -d kmuhub -tAc "select version, dirty from schema_migrations"
```

## Startkommando für die Nacht

```powershell
powershell -ExecutionPolicy Bypass -File .planning\backend-block\loop\run-loop.ps1 `
  -MaxIterations 130 -UntilTime "07:30" -BudgetUsd 12 -Effort high
```

Der Treiber pusht am Ende **einmal** auf `backend-loop` und schreibt vorher die Laufbilanz ins
Journal. Ein PR wird nicht angelegt — den macht Luke bewusst (Reihenfolge: erst beide
Review-Workflows `gh workflow disable`, dann PR öffnen, nach grünem CI wieder enablen).

---

## Belege

- `.planning/launch-lagebild-2026-08-12.md` §4 — G0/G1/G2, Quelle für den roten Faden
- `.planning/backend-block/loop/BACKLOG.yml` — Kopf trägt die Entscheidungsrunde 2026-08-24
- `.planning/backend-block/loop/hooks/backlog-check.py` — der Vorflug, gegen den geschrieben wird
- `.planning/backend-block/loop/run-loop.ps1` — `Get-BacklogState`, Modell-/Unit-Drift, Bilanz
- `.planning/backend-block/PROMPT-LOOP-VERBESSERUNG-UND-ENTSCHEIDUNGEN.md` — die Sitzung davor
- Commits `fada9af4`…`acc48aee` — Entscheidungen, Treiber-Fixes, Spec-Skript

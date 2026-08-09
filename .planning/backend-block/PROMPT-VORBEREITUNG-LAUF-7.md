# Prompt — Loop-Mechanik reparieren (K5–K8) und Lauf 7 planen

> Für eine frische Session im Repo-Root `C:\Users\Luke\Documents\KMU Hub`.
> Erstellt 2026-08-09 nach dem Review von Lauf 6.

---

Wir haben heute Nachtlauf 6 reviewt, zwei Korrekturen nachgezogen, gemergt und deployt.
Production ist verifiziert: `main` auf `1e68b7dc`, Migrationskopf **308 clean**, Server-Git
nicht detached, 30/35 Container healthy, `/health` meldet `commit: 1e68b7dc`.

Der vollständige Review liegt in `~/.claude/plans/sodele-nachtlauf-ist-durch-effervescent-breeze.md`
— **lies den zuerst**, besonders Abschnitt 4 (P1–P6) und Abschnitt 5 (K5–K9). Alles unten
Beschriebene ist dort ausführlich belegt.

Deine zwei Aufgaben: **(A)** die Loop-Mechanik reparieren, **(B)** Lauf 7 für heute Abend
vorbereiten. Nutze für die Bauarbeit primär Subagenten (max. 3 parallel, Sonnet), damit das
Hauptkontextfenster schlank bleibt. Subagenten können nicht nachfragen — gib ihnen alles mit.

---

## A · Loop-Mechanik (K5–K8)

Alles unter `.planning/backend-block/loop/`. Das muss von Hand passieren, **nicht** als
Backlog-Unit: der Loop darf seinen eigenen Treiber nicht umbauen.

### K5a — CI-Erkennung in `run-loop.ps1` (~Zeile 377)

Nach dem Push von Lauf 6 hat der Treiber gemeldet „Nach 5 min kein CI-Lauf für diese SHA —
vermutlich paths-Filter". **Das war falsch.** Der Lauf existierte 14 Sekunden nach dem Push
(Run `31282724353`, `head_sha = fc7c6e5c`) und wurde grün.

Aktueller Code:

```powershell
$runId = Invoke-Native { & gh run list --branch backend-loop --workflow=ci.yml --limit 10 `
    --json databaseId,headSha --jq "[.[] | select(.headSha==\"$sha\")][0].databaseId" 2>$null }
```

**Hypothese, nicht bewiesen:** PS 5.1 reicht die eingebetteten `\"` nicht sauber über die
Native-Grenze, jq bekommt `select(.headSha==fc7c…)` ohne Quotes und wirft. Fix: kein jq über
die Grenze, in PowerShell filtern:

```powershell
$runs = Invoke-Native { & gh run list --branch backend-loop --workflow ci.yml --limit 10 --json databaseId,headSha } | ConvertFrom-Json
$runId = ($runs | Where-Object { $_.headSha -eq $sha } | Select-Object -First 1).databaseId
```

**Verifizier die Ursache, statt sie zu glauben** — ruf beide Varianten einmal von Hand gegen
eine bekannte SHA auf und vergleich die Ausgabe. Wenn die Ursache eine andere ist, sag das und
fix die echte.

Folgeschaden: der `## CI nach Lauf`-Journaleintrag wurde nie geschrieben. Siehe K6.

### K5b — Iterationsnummer und Zeitstempel vom Treiber setzen (P3/P4)

Das Modell rät beides. Ab Treiber-Iteration 27 schrieb es „## Iteration 28" — seitdem läuft
alles um eins vor, „Iteration 32" existiert zweimal, „27" gar nicht. Zeitstempel sind
ebenfalls frei erfunden (Treiber: Iteration 16 um 18:22, Journal: „19:35").

Der Loop-README sagt selbst, dass der Treiber die Fortschrittsanzeige aus der höchsten
Iterationsnummer im Journal ableitet — in Lauf 3 hat genau diese Klasse zwei Iterationen lang
denselben Stand gemeldet.

Fix: der Treiber kennt `$i` und die Uhrzeit. Beides in den Prompt einsetzen oder die
Kopfzeile gleich vom Treiber schreiben lassen und das Modell nur den Rumpf anhängen.

### K5c — `error_max_budget_usd` sichtbar machen (P5)

Iteration 18 lief in den 12-USD-Deckel (`subtype: error_max_budget_usd` in `iter-018.json`,
mit 12,06 USD die teuerste des Laufs). Der Treiber loggte korrekt „endete mit Exit 1", aber
nicht *warum*. Diesmal war die Arbeit zufällig schon committet; wäre der Deckel mitten in der
Arbeit gefallen, läge die Unit halbfertig auf `in_progress`.

Fix: `subtype` aus dem JSON in die Logzeile ziehen, und bei `error_max_budget_usd` die
`in_progress`-Unit explizit auf `todo` zurücksetzen.

### K5d — `ITERATION.md` auf Lauf 7 (P2)

Die Datei steht inhaltlich noch auf Lauf 5 (mtime 2026-08-06): „**Freigegeben in diesem Lauf**
(Lauf 5, Stand 2026-08-05)…" und „**Öffentliche Routen sind in diesem Lauf ein Schwerpunkt**
(drei Stück: CSAT-Antwort, Formular-Einreichung, Wiki-Einlösung)". Alle 47 Iterationen von
Lauf 6 liefen mit diesem Kontext.

Umschreiben auf Lauf 7. Schwerpunkt dort ist **reine Coverage** (Block B: gateway, dann
server) plus eine Fix-Unit. Nebenbei: die Überschrift sagt „sechs Fehlerklassen", die Liste
hat acht.

### K6 — Abschlussblock für Lauf 6 ins `JOURNAL.md`

Fehlt wegen K5a. Ohne ihn startet Lauf 7 blind. Echte Werte:

- CI-Run `31282724353`, SHA `fc7c6e5c`, `success`, 7 940 PASS / 0 SKIP / 0 FAIL
- danach zwei Review-Korrekturen (`d902f3a1`, `0f5a8bc2`) und die Backlog-Umstellung (`79651386`)
- Run `31306374635` auf `0f5a8bc2`: 7 944 PASS / 0 SKIP / 0 FAIL, neuer Integration-Job
  344 Tests / 0 FAIL, Coverage 36,3 %
- PR #19 gemergt → `main` `1e68b7dc`, CD `31307275850` success, Prod-Kopf **308 clean**

### K7 — Journal-Nummerierung 27–47 geradeziehen

Einträge ab „## Iteration 28" um eins zurück; die Dublette „Iteration 32"
(`c-cov-crm-contact-repo`) ist in Wahrheit 27. **Nur Kopfzeilen korrigieren, keine Einträge
umsortieren** — die Append-only-Regel bleibt.

### K8 — Forward-only auf Kommentare ausweiten

In Lauf 6 wurde `backend/migrations/000139_gobd_belegarchiv.up.sql` kosmetisch geändert (nur
ein SQL-Kommentar, Retention-Jahr +8 → +10; der Code rechnet nachweislich +10). Wirkungslos,
aber die Regel soll auch für Kommentare gelten. In `ITERATION.md` explizit machen: an einer
ausgerollten Migration wird gar nichts mehr angefasst, Korrekturen gehören in eine neue
Migration oder an den Code.

---

## B · Lauf 7 vorbereiten

### B1 — Entscheidung, die du dem User vorlegen musst: der Backlog ist zu klein

Das Fenster ist **22:00 bis 09:00**, also 11 Stunden ≈ **50 Iterationen** beim Median von
13 min aus Lauf 6. Im Backlog stehen aber nur **27 todo**:

1. `fix-inventar-picking-partial-book`
2. `c-cov-biz-lexware`
3. `c-cov-biz-recurring`
4.–15. zwölf `b-cov-gateway-*`
16.–27. zwölf `b-cov-server-*`

Das sind ≈ 5 h 51 — der Loop wäre gegen **04:00** durch, schriebe `ALLE UNITS ABGEARBEITET`,
legte `STOP` an und beendete sich. Rund **23 Units fehlen**, um das Fenster zu füllen.

Zum Kostenrahmen: Lauf 6 kostete **316 USD für 47 Iterationen** (Ø 6,73). 50 Iterationen
liegen bei grob 340 USD.

Leg dem User per `AskUserQuestion` vor: Backlog auffüllen (und womit) oder den frühen Stopp
akzeptieren. Kandidaten für neue Units, mit gemessenen Zahlen aus dem CI-Coverage-Artefakt
von Lauf 6:

| Paket | Coverage | ungedeckte Statements |
|---|---|---|
| `internal/caldav` | 7,2 % | 1 015 |
| `internal/plugin` | 23,8 % | 854 |
| `internal/einkauf` | 33,2 % | 602 |
| `internal/document` | 40,3 % | 787 |
| `internal/dialer` | 36,7 % | 596 |
| `internal/notification` | 36,5 % | 896 |
| `internal/email` | 50,2 % | 1 028 |
| `internal/work` | 47,9 % | 2 518 |

**Wichtig bei neuen Coverage-Units:** prüf vorher mit
`grep -rl "^//go:build" --include=*_test.go`, ob im Zielpaket Tests hinter einem Build-Tag
liegen. Solche Tests laufen weder im PR-Gate noch zählen sie in `coverage.out` — genau das ist
in Lauf 6 dreimal passiert (invoice/quote/creditnote).

### B2 — Vorflug

- `git checkout backend-loop && git merge origin/main --no-edit` — **Merge, nicht Rebase**.
  Aktuell ist `backend-loop` = `79651386`, `main` = `1e68b7dc` (ein Merge-Commit voraus), das
  wird ein Fast-Forward.
- Lokale DB: Container `docker-postgres-1` muss laufen, Migrationskopf auf **308**, Rolle
  `kmuhub_app` mit Passwort `app_dev`. Genaue Kommandos in
  `.planning/backend-block/loop/GATE-COMMANDS.md`.
- **Draft-PR anlegen — vorher die Review-Workflows abschalten.** `Claude PR Review` und
  `Security Review` sind beide `active` und haben **kein Draft-Gate**: `pr create` zündet sie
  sofort und kostet Geld. Also erst `gh workflow disable "Claude PR Review"` und
  `gh workflow disable "Security Review"`, dann den Draft-PR gegen `main` anlegen. Der Treiber
  legt selbst keinen an — er pusht nur und wartet auf CI eines bereits offenen PRs.
- `-DryRun` fahren, um Prompt-Pfade und Modellwahl zu prüfen, ohne eine Iteration zu brennen.

### B3 — Start

```
powershell -ExecutionPolicy Bypass -File .planning\backend-block\loop\run-loop.ps1 -UntilTime "09:00"
```

`-UntilTime` meint das nächste Auftreten von `HH:mm`, ein Start um 22:00 trifft also den
Folgemorgen. Die Schlafsperre setzt das Skript seit `e3b1afca` selbst; Display darf ausgehen.

Frag den User, ob der Start um 22:00 **manuell** erfolgen soll (Kommando bereitlegen) oder ob
du eine Windows-Aufgabe einrichten sollst. Bei der Aufgabenplanung ist die Auth der
`claude`-CLI der Haken: „Ausführen, auch wenn Benutzer nicht angemeldet ist" kann sie brechen.
Wenn Aufgabenplanung, dann unter dem angemeldeten Konto mit „Nur ausführen, wenn Benutzer
angemeldet ist" — und **vorher testweise auslösen**, nicht auf gut Glück um 22:00.

---

## Randbedingungen

- Deutsch für Kommunikation und Journal/Notizen, Englisch für Code, Identifier und Commits.
- Conventional Commits, imperativ, **keine AI-Attribution**.
- Gate-Kommandos **nie durch eine Pipe** — der Exit-Code wäre der der Pipe und immer 0.
- PS 5.1: kein `&&`, kein Ternary, kein `??`. Native stderr terminiert unter
  `$ErrorActionPreference = "Stop"` — deshalb existiert `Invoke-Native` im Treiber; neue
  native Aufrufe gehören da hinein.
- `run-loop.ps1` und `ITERATION.md` nicht mit `Set-Content` schreiben (BOM + Mojibake) —
  Edit-Tool nutzen.
- Änderungen am Treiber **vor** dem Start mit `-DryRun` gegenprüfen.

## Rückgabe

Am Ende: was geändert wurde, was der `-DryRun` gesagt hat, ob der Draft-PR steht, wie der Lauf
gestartet wird, und die Entscheidung aus B1 mit ihrer Begründung.

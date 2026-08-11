# Backend-Nachtloop — Journal Lauf 9

Append-only. Ein Eintrag je Iteration, **ans Dateiende**, nie einsortieren. Form und Pflichtzeilen
stehen in `ITERATION.md` Schritt 6.

Frühere Läufe liegen vollständig im Archiv:
`archive/lauf-1-2/` (58 Units) · `archive/lauf-3/` (61) · `archive/lauf-4/` (54) ·
`archive/lauf-5/` (41) · `archive/lauf-6/` (46) · `archive/lauf-7/` (71) ·
`archive/lauf-8/` (94, inkl. `logs/`).

---

## Laufkontext

- **Ausgangspunkt:** `backend-loop` auf `origin/main` gemergt (nicht rebased), Fast-Forward auf
  `10a1a26e`. `main` = `10a1a26e`. Produktion: Migrationskopf **310 clean**, 36 Container laufend /
  30 healthy / 0 unhealthy, `/health` mit 23 Services.
- **Migrationen:** Repo-Kopf = lokaler Kopf = Produktionskopf = **310**. Nächste freie **311** —
  aber immer zur Laufzeit ermitteln.
- **Lokale DB:** vor dem Start prüfen. Rolle muss `kmuhub_app` sein — `kmuhub` hat BYPASSRLS und
  würde jede RLS-Lücke durchwinken. `go test` ohne `DATABASE_URL` ist **kein** Gate.
- **Backlog:** `BACKLOG.yml`, Block A (10 Fix-Units) + Block B (11 Scan-Units). Null `blocked`- und
  null `done`-Units zum Laufbeginn — die Datei ist für diesen Lauf frisch aufgebaut, Lauf 8 liegt
  vollständig in `archive/lauf-8/`. `BACKLOG-NEXT.yml` ist leer: die zehn Fix-Units sind
  **verschoben**, nicht kopiert.
- **Fenster:** ein Lauf ab 16:00 (`-StartNotBefore "16:00"`), Deadline `-UntilTime "09:00"` als
  Sicherheitsnetz. Kein Pausenfenster. Ein Prozess, ein Push, ein CI-Lauf.
- **Workflow-Zustand beim Start:** `Claude PR Review` `disabled_manually`, `Security Review` vor dem
  Anlegen des Draft-PRs disabled (beide haben kein Draft-Gate und würden beim `opened`-Event zünden).

### Was dieser Lauf ist — und was nicht

Lauf 9 ist ein reiner **Fix- und Scan-Lauf**. **Keine Coverage-Units.**

Der Coverage-Engpass ist zu: Lauf 8 hat 47,7 → **60,0 %** gehoben, bei einem Gate von 15 %. Wichtiger
ist, was dabei sichtbar wurde: die vier Pakete mit den **schlimmsten** Bugs haben die **höchste**
Coverage — `notification/preference` 87,2 % (`UpsertQuietHours` schlägt bei jedem Aufruf fehl),
`document/virtual` 83,1 % (vier Queries auf eine gelöschte Spalte), `schichten` 79,7 % (Schichttausch
hat keinen funktionierenden Pfad), `biz/datev` 79,3 % (Upload seit zwei Monaten totalausgefallen).
Coverage misst keine Korrektheit. Mehr Prozente auf `gateway` (46,0 %, schwächstes Kernpaket) würden
dieselbe Bug-Klasse *erzeugen*, nicht finden — deshalb steht `gateway` in diesem Lauf nicht auf der
Liste.

**Block A** sind die zehn verifizierten Produktionsbugs aus Lauf 8, nach Schwere sortiert.
**Block B** sind elf Muster-Scans: die zehn Bugs teilen vier mechanisch auffindbare Muster
(Typ-Scan-Mismatch, `ON CONFLICT` gegen einen nicht existierenden Index, SQL auf gelöschte Spalten,
INSERT ohne `tenant_id`, `nil`-Slice als JSON `null`). Jede Scan-Unit legt ihre Funde als neue
Fix-Units am Backlog-Ende an — **Block C** entsteht damit zur Laufzeit und füllt den Rest des
Fensters.

### Neu in diesem Lauf

- **`neue-units:` ist Pflichtzeile** (seit `d6d80fcc`). Ein Fund ohne angelegte Unit ist kein Fund.
  In früheren Läufen sind verifizierte Bugs verlorengegangen, weil sie nur im Journal standen —
  drei der zehn Block-A-Units mussten bei der Nachbereitung von Lauf 8 nachgetragen werden.
- **`coverage:` ist ein Delta auf dem Paket der Unit**, nicht gegen den Laufstart. `coverage_start`
  nennt jetzt bei jeder Unit **exakt** das Paket, das sie anfasst — in der Lauf-8-Fassung standen
  bei fünf Units Elternpakete oder veraltete Werte (`internal/security 76,9 %` statt
  `internal/security/audit 66,1 %`), womit die Iteration nur `n.a.` schreiben konnte.
- **Der Drift-Check vergleicht nur noch den Kalendertag** (`d6d80fcc`). Vorher feuerte er
  minutengenau und damit in 90 von 94 Iterationen, obwohl die Nummer 94/94 stimmte — eine Warnung,
  die fast immer feuert, wird nicht gelesen.
- **Pin-Tests werden UMGEDREHT, nicht gelöscht.** Jede Block-A-Unit bringt einen Test mit, der heute
  das *kaputte* Verhalten assertiert. Nach dem Fix muss er das *korrekte* assertieren. Ein
  gelöschter Pin-Test ist eine verlorene Regression.
- **Die Schichttausch-Semantik ist vorab entschieden** (Luke, 2026-08-11): ein Tausch ist nur
  zwischen zwei bereits zugeordneten Mitarbeitern gültig, Fall 2 wird ein Validierungsfehler. An
  genau dieser offenen Frage hat Iteration 94 in Lauf 8 gehalten — sie ist beantwortet und nicht
  erneut aufzumachen.
- **Neu auf `main` seit Lauf 8:** eine öffentliche Route `GET/POST /reset-password` im Gateway
  (`10a1a26e`, embedded HTML-Seite für den Passwort-Reset-Mail-Link). Beim Anfassen von
  `cmd/gateway/main.go` nicht versehentlich ausbauen.

---

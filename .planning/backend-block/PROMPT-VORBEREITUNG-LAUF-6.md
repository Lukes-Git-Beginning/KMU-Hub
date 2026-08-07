# Prompt — Nachtlauf 6 vorbereiten (Nacht 2026-08-07 → 08-08)

> Zum Kopieren in ein frisches Fenster. Self-contained: setzt keinen Kontext aus einer Vorsession
> voraus. Erwarteter Aufwand 2–3 h. Modell: **Opus** (Rechercheurteil, kein Volumen).
> Muss **vor 21:00** fertig sein — dann startet der Lauf.

---

## Der Prompt

Ich bereite den **sechsten Backend-Nachtlauf** vor. Fenster: **heute 21:00 bis morgen früh
ca. 09:00**, also rund **12 Stunden**. Du sollst in dieser Session **nichts von der offenen
Arbeit selbst umsetzen** — deine einzige Aufgabe ist, den Lauf so vorzubereiten, dass er um 21:00
startklar ist und zwölf Stunden durchhält.

### Ausgangslage (verifiziert am 2026-08-06/07, nicht nachrecherchieren)

- Läufe 1–5 sind durch, gemergt und deployt. **Repo-Kopf = Prod-Kopf 297 clean.**
- `backend-loop` steht auf `2e09c706`, identisch mit `main`. **Kein offener PR.**
- **Der Loop-Backlog ist leer:** `BACKLOG.yml` hat 34 `done`, 7 `blocked`, **0 `todo`**.
  Ein 12-Stunden-Lauf braucht ihn damit **komplett neu**.
- Die 7 `blocked` sind **keine Ausfälle**, sondern Entscheidungsvorlagen für mich
  (Payroll-Datenmodell, Admin-Billing, Projekt-Meilensteine, öffentliche CSAT-Oberfläche,
  Landing-Pages für die sieben Public-Token-Routen, `DisallowUnknownFields`) plus ein reiner
  Frontend-Fix. **Lass sie blockiert.** Zieh sie nicht in den neuen Backlog.

**Lies zuerst** `.planning/backend-block/loop/README.md` (Mechanik, Sicherheitsmodell, Grenzen) und
`.planning/status-overview.md` (gemessener Ist-Stand vom 06.08.). Der alte Prompt
`PROMPT-BACKLOG-NACHLEGEN.md` beschreibt Lauf 4 — als Muster für Format und Tonfall brauchbar, die
Zahlen darin sind überholt (er nennt Migrationskopf 268, aktuell ist **297**).

### Zielgröße

Lauf 4 schaffte **54 Units in 48 Iterationen**. Plane für zwölf Stunden **45–55 Units**. Lieber
etwas mehr als zu wenig: überzählige Units bleiben `todo` und starten Lauf 7. Ein Loop, der um
02:00 leerläuft, hat sechs Stunden verschenkt.

### Woher die Units kommen — in dieser Reihenfolge

**1. Test-Coverage (Schwerpunkt, ~25–30 Units).** Das ist der gemessene Engpass und mechanisch
genug für einen Nachtlauf. Stand aus dem CI-Lauf vom 06.08. (Gesamt 30,2 %, Gate 15 %):

| Paket | Cov | Funktionen |
|---|---|---|
| `internal/server` | **8,1 %** | 1.687 |
| `internal/gateway` | **27,2 %** | 1.538 |
| `internal/biz` (Zahlungen, kritischer Pfad) | **48,4 %** | 911 |
| `internal/crm` (Kundendaten, kritischer Pfad) | **51,4 %** | 318 |
| `produktion` / `caldav` | 19,6 / 13,1 % | |
| `sysctx`, `models`, `metrics`, `idempotency`, `health`, `clientctx` | 0 % | Infrastruktur-Glue |

Ziel für die kritischen Pfade ist 60 %. `server` und `gateway` sind zusammen 3.225 Funktionen —
dort liegt der größte Hebel, auch wenn per Architekturregel „thin handlers" wenig Logik drinsteckt.

**Harte Qualitätsregel für jede Coverage-Unit** — sonst produziert der Loop Zeilen-Abdeckung ohne
Wert. Jede Unit muss in `done_when` beides tragen:
- mindestens ein **Fehlerpfad** pro getesteter Funktion (nicht nur Happy-Path): falsche UUID,
  fehlender Tenant, Downstream-Fehler, Validierungsbruch;
- eine **Mutations-Probe**: eine Zeile im getesteten Code absichtlich brechen, prüfen dass der neue
  Test rot wird, zurückdrehen. Wird er nicht rot, testet er nichts — dann `blocked`.

Schneide die Units entlang existierender Dateien (`route_<modul>_test.go`,
`<modul>_grpc_test.go`), nicht entlang von Prozentzielen. Eine Unit = eine Datei = ein Commit.

**2. `.planning/backend-gaps.md` (~10–15 Units).** 108 KB, 28 Modul-Abschnitte, **an vielen Stellen
überholt** — die Läufe 1–5 haben still vieles geschlossen. Die Kernarbeit ist deshalb
**Verifikation, nicht Abschreiben**. Für jede Behauptung, bevor sie eine Unit wird:

- Tabelle/Spalte da? → `docker exec -i docker-postgres-1 psql -U kmuhub -d kmuhub -tA -c "..."`
  (Migrationskopf **297**; als App-Rolle `kmuhub_app` testen, nicht `kmuhub` — der hat BYPASSRLS)
- Route da? → `grep -rn "<pfad>" backend/internal/gateway/`
- Service-Methode echt oder Stub? → **die Funktion lesen**, nicht den Namen
- Erwartet das Frontend etwas? → `desktop/src/renderer/src/api/<modul>-client.ts` plus der
  MSW-Handler in `mocks/handlers/` sind der bindende Vertrag

Was sich als längst gebaut erweist: in `backend-gaps.md` als erledigt markieren. Das entstaubt die
Datei für Lauf 7 gleich mit und ist mir genauso viel wert wie eine neue Unit.

**3. Rest (~5 Units).** Was dir beim Lesen auffällt und belastbar ist. Kein Füllmaterial.

### Was NICHT hineingehört

- **Frontend.** Der Loop baut Backend. Die drei Stores mit hartkodierten Demo-Daten
  (`helpdesk`, `vertraege`, `timetracking`) und die 34 fehlenden fr/it-i18n-Schlüssel gehören in
  eine Frontend-Session.
- **RLS-Scans.** `knownRLSGaps` ist seit Migration 286 leer, ein Standing-Guard hält es so.
- Gesperrt bleibt: Phase 4 (Branchen-Backend), neue `config.RequireX`-Assertionen, Scharfschalten
  neuer `modules.*`-Flags, Merge nach main, Deploy.
- **CSAT nicht anfassen.** Der SMTP-Passthrough an `helpdesk` und die öffentliche Seite sind eine
  Deploy-Entscheidung, keine Nachtlauf-Iteration.

### Format der Units

Neuer Block in `.planning/backend-block/loop/BACKLOG.yml`, Struktur exakt wie die bestehenden:

```yaml
  - id: <kebab-case>
    phase: 3
    service: <modul>
    model: sonnet          # opus nur fuer Proto, Security, Migration+RLS
    deps: []
    status: todo
    scope: >
      Was gebaut wird, in ganzen Saetzen. Mit der verifizierten Fundstelle
      (Datei:Zeile oder Tabellenname), nicht mit "laut backend-gaps".
    sources:
      - <pfade, die der Agent zuerst lesen soll>
    notes: >
      Was schiefgehen kann, welches bestehende Muster wiederzuverwenden ist,
      und — falls die Praemisse wackelt — der Auftrag, sie zuerst zu pruefen
      und sonst `blocked` zu setzen.
    done_when:
      - nachpruefbare Kriterien, kein "funktioniert"
```

**Zwei harte Parsing-Regeln des Treibers** (`run-loop.ps1`), sonst zählt er die Units falsch:
`model:` muss **vor** `status:` stehen, und hinter `status: todo` darf **kein** Kommentar stehen.

### Qualitätsmaßstab

- Eine Unit, deren Prämisse du nicht selbst im Code gesehen hast, kommt nicht rein. Lieber
  40 belastbare Units als 55, von denen nachts zehn als `blocked` zurückkommen.
- Jede neue Tabelle braucht `tenant_id UUID NOT NULL` + `CALL enable_tenant_rls(...)`.
- Schwerpunkt: Sicherheit vor Features, echte Lücken vor Kosmetik.

### Neben dem Backlog: den Start scharf machen

Diese vier Punkte gehören zur Vorbereitung, nicht zum Lauf:

1. **Guard-Test grün** — `bash .planning/backend-block/loop/hooks/test-loop-guard.sh`.
   Der Treiber bricht sonst vor der ersten Iteration ab.
2. **Branch aktuell** — `backend-loop` auf `origin/main` **mergen, nicht rebasen** (Rebase ist im
   Guard gesperrt und macht den nächsten Push force-pflichtig). Steht aktuell auf demselben Commit
   wie main, sollte also ein No-op sein — trotzdem prüfen.
3. **PR für das CI-Signal anlegen.** `ci.yml` triggert nur auf `push: [main]` und
   `pull_request: [main]` — ohne offenen PR läuft nach dem Nacht-Push **kein** CI. PR #18 ist
   gemergt, es gibt aktuell keinen. Reihenfolge zwingend, weil beide Review-Workflows auf
   `opened` feuern und **kein Draft-Gate** haben:
   ```bash
   gh workflow disable "Claude PR Review"; gh workflow disable "Security Review"
   gh pr create --base main --head backend-loop --draft --title "Backend-Nachtlauf 6"
   gh workflow enable "Claude PR Review"; gh workflow enable "Security Review"
   ```
   Das geht erst, wenn `backend-loop` mindestens einen Commit vor `main` hat — also **nach** dem
   Backlog-Commit.
4. **Trockenlauf** — `powershell -ExecutionPolicy Bypass -File .planning\backend-block\loop\run-loop.ps1 -DryRun`
   muss die neue Unit-Zahl zeigen. Zeigt er weniger als du geschrieben hast, ist das YAML oder die
   Feldreihenfolge kaputt.

### Abschluss

- Ein Commit auf **`backend-loop`** (dort arbeitet der Loop), Conventional Commits,
  **keine AI-Attribution**. Der Backlog muss auf dem Branch liegen, sonst sieht der Loop ihn nicht.
- Sag mir am Ende in dieser Reihenfolge:
  1. **wie viele Units** drin sind und wie viele Stunden das nach Lauf-4-Tempo trägt,
  2. welche Punkte aus `backend-gaps.md` sich als **bereits erledigt** erwiesen haben,
  3. **das exakte Startkommando** für 21:00, mit gesetzter Deadline,
  4. was du **nicht** verifizieren konntest.
- Starte den Lauf **nicht selbst**. Das mache ich um 21:00.

### Startkommando (zur Kontrolle — du gibst es mir am Ende nochmal aus)

```powershell
powershell -ExecutionPolicy Bypass -File .planning\backend-block\loop\run-loop.ps1 -UntilTime "08:45"
```

Vorher einmal unter Aufsicht: `-MaxIterations 2`. Die Schlafsperre setzt `run-loop.ps1` seit
`e3b1afca` selbst — Standby würde sonst die API-Verbindung reißen.

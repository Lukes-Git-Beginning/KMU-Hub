# Zwei-Terminal-Workflow „Arbeiten mit Nico"

> **Aktiv, sobald Darien sinngemäß sagt „ich arbeite mit Nico" / „arbeite mit Nico".** Dann läuft er mit zwei Terminals, jedes mit eigenem Claude. Dieses Dokument definiert beide Rollen + die Koordination, damit sich die parallelen Ströme nicht ins Gehege kommen.

## Warum überhaupt getrennt
Zwei Claude-Sessions auf **derselben** Arbeitskopie teilen einen Working Tree und ein `.git` → Git-Races, kollidierende uncommittete Änderungen, hakelige Merges (genau das ist beim ersten Nico-Merge passiert). Lösung: **zwei getrennte Klone**. Jeder Klon hat eigenes `main`, eigenes `node_modules`, eigenen Dev-Server-Port. Synchronisiert wird ganz normal über `origin`.

## Die zwei Rollen

| | **Terminal A — Build** | **Terminal B — Review & Koordination** |
|---|---|---|
| Ordner | `…/KMU Hub` (Hauptklon) | `…/KMU-Hub-review` (zweiter Klon) |
| Dev-Port | 5173 | **5174** (kollidiert nicht mit A) |
| Aufgabe | unsere Feature-Phasen bauen (direct-to-main) | Nicos PRs reviewen, mergen, Backlog/Koordination, Memory |
| Schreibt nach `main`? | ja (unsere Arbeit) | ja (nur gemergte Nico-Branches) |

### Wie Claude seine Rolle erkennt
Beim Trigger „ich arbeite mit Nico" prüft Claude sein Arbeitsverzeichnis:
- Pfad enthält **`KMU-Hub-review`** → **Rolle B (Review/Koordination)**.
- sonst (`KMU Hub`) → **Rolle A (Build)**.

Wenn der Review-Klon noch nicht existiert und Claude in einer Session ist, die als B agieren soll → erst das Setup unten ausführen.

## Setup (einmalig — zweiter Klon für Terminal B)
```bash
cd C:/Users/darie/Documents
git clone https://github.com/Lukes-Git-Beginning/KMU-Hub.git KMU-Hub-review
cd KMU-Hub-review/desktop
npm install
npm install -D playwright && npx playwright install chromium
# Dev-Server in B immer auf eigenem Port starten:
npm run dev -- --port 5174
```
> Die QA-Scripts in B müssen gegen `http://localhost:5174` laufen (nicht 5173). Beim Kopieren einer `qa-*.mjs`-Vorlage in B die `BASE`-Konstante auf 5174 setzen.

## Koordinations-Regeln (verbindlich für beide)
1. **Getrennte Klone** = keine Datei-Kollision. Niemals beide Terminals im selben Ordner.
2. **`git pull` vor jeder Arbeit und vor jedem Push.** Beide schreiben nach `main`, also immer erst neuesten Stand holen.
3. **Nie `--force`, nie `reset --hard` auf gemeinsame History.** Bei Divergenz: `git pull --rebase` (lokale, noch ungepushte Commits sauber oben drauf).
4. **Atomare Pushes:** Erst committen, dann `git pull --rebase`, dann `git push`. So bleibt `main` linear.
5. **Wer mittendrin ist, committet bevor der andere merged** — wenn B einen Nico-Merge pushen will und A gerade uncommittete Arbeit hat, ist das dank getrennter Klone egal (A pullt vor seinem nächsten Push).

## Terminal B — Nico-Review→Merge-Ablauf (Schritt für Schritt)
Wenn Nico „fertig" meldet (Branch + Commit-SHA + Verify-Checkliste + Screenshots):
1. `git fetch origin`
2. **Diff-Review (read-only):** `git show <sha> --stat` + `git show <sha> -- <dateien>` — Code gegen die Definition-of-Done der Spec prüfen.
3. **Live-Review (gründlich):** Nicos Branch auschecken + im echten UI testen:
   ```bash
   git checkout <nico-branch>
   cd desktop && npm install && npm run dev -- --port 5174
   # QA-Script gegen :5174 laufen + Screenshots mit Read ansehen
   ```
4. **Urteil:** grün → weiter; sonst konkrete Nachbesser-Punkte an Nico, zurück zu ihr.
5. **Merge (grün):**
   ```bash
   git checkout main
   git pull
   git merge --no-ff <nico-branch> -m "merge: <modul> phase-XX (Nico)"
   git push
   ```
   `--no-ff` hält Nicos Arbeit als erkennbaren Block in der History.
6. **Aufräumen:** gemergten Branch optional löschen (`git push origin --delete <nico-branch>`), Backlog/Beobachtungen festhalten, Nico grünes Licht + nächste Phase geben.
7. **Terminal A informieren:** A muss vor seinem nächsten Push `git pull` machen (B hat `main` bewegt).

## Terminal A — Build-Pflichten
- Auf `main`, direct-to-main, unsere Feature-Phasen.
- **Vor jedem Push `git pull`** (B könnte Nico-Merges eingespielt haben).
- Reviews macht A NICHT — das ist B's Rolle. A bleibt im Bau-Fluss.

## Eskalation / Sonderfälle
- **Merge-Konflikt** beim Nico-Merge (selten, da Nico isolierte Blöcke baut): in B auflösen, Nico ggf. um Rebase auf aktuellen `main` bitten.
- **Beide brauchen dieselbe Datei:** sollte durch klare Modul-Zuteilung (Nico = Content-Block, Haupt-Team = Rest) kaum vorkommen. Wenn doch → kurz absprechen, einer nach dem anderen.
- **Backlog-Pflege** (z.B. von Nico gemeldete Out-of-Scope-Bugs) macht B in `.planning/backend-gaps.md` bzw. einem Review-Log.

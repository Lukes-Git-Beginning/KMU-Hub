# Multi-Stream-Workflow — 3 Ströme parallel (Marathon-Modus)

> Erweitert `two-terminal-nico-workflow.md` von 2 auf **3 parallele Bau-Ströme**. Gilt ab dem „Alle-Phasen-Durchlauf" (2026-06-09).
> **Begleitdokumente:** Kollisions-Karte → `collision-map.md` · Review-System → `reviews/_TEMPLATE.md` · Phasen → `module-phase-plans.md` · Delegation → `nico-block/` + `luke-block/`.

## Die drei Ströme

| | **N — Nico** | **D — Dein-PC** | **L — Luke** |
|---|---|---|---|
| Maschine | Nicos PC | Dariens PC (remote) | Lukes PC |
| Treiber | Nico | Nico od. Luke (remote, semi-autonom) | Luke |
| Lane | Content-Block | Daily-Use-Module | vorm. Backend-P0 / nachm. FE-Block |
| Klon | eigener | Hauptklon | eigener FE-Klon |
| Schreibt nach `main` | ja | ja | ja |

Alle drei schreiben nach `main` → **Kollisions-Karte ist Pflicht** (getrennte Klone, `pull --rebase` vor Push, Hot Files nur additiv).

## Tagesablauf pro Strom (immer gleich)
1. **Session-Start:** `git pull --rebase` · KICKOFF des Stroms lesen · Lane + nächste Phase aus `module-phase-plans.md` bestimmen.
2. **Bauen:** eine Phase nach `nico-block/WORKFLOW.md` (gilt für alle): bauen → i18n ×4 → Demo-Handler falls nötig → gescopter Typecheck (nur geänderte Dateien) → Playwright-Screenshot-QA → **Screenshots wirklich ansehen** → iterieren bis grün.
3. **Review-Faden notieren:** nach jeder fertigen Phase einen Eintrag in `reviews/<modul>.md` (Pfad zum Hinklicken · worauf achten · Screenshots · offene Punkte). Das ist die Vorlage für den gemeinsamen Feinschliff-Review, wenn Darien zurück ist.
4. **Commit + Push:** `git add` → `commit` (Conventional, keine AI-Attribution) → `git pull --rebase origin main` → `push`. Ein Commit pro Phase.
5. **Backend-Bedarf** → `backend-handover-luke.md` (nicht selbst Backend bauen außer Strom L).

## Cadence (Darien-Entscheidung)
- **Erst alle bauen, Reviews danach.** Phasen laufen durch; pro Phase entsteht ein Review-Faden. Der Feinschliff-Review (Darien navigiert/wir schauen — wie zuletzt beim Profil-Fenster) passiert **gebündelt**, wenn Darien zurück ist, und wird **unter dem Team aufgeteilt**.
- **Kein Live-Review-Gate während des Laufs** — die Build-+-Verify-Schleife (scoped tsc + Screenshot-QA) ist die Selbstkontrolle. Qualität vor Tempo: lieber 3 Phasen sauber als 6 halb.

## Konflikt-/Eskalationsregeln
- **Rebase-Konflikt** (fast immer in einer Hot File, additiv): beide Blöcke behalten → `git rebase --continue`. Siehe `collision-map.md §2`.
- **CI rot** nach Push: `git revert <sha>` (nie `reset --hard`), Ursache fixen, neu pushen.
- **Zwei Ströme brauchen dieselbe geteilte Datei:** abstimmen, einer nach dem anderen (`collision-map.md §5`).
- **Unklarheit über Scope/Domäne:** im Review-Faden als „offene Frage" markieren, weiterbauen, Darien klärt beim Review.

## Rollen-Erkennung (für jeden Claude-Bot beim Start)
- Arbeitsverzeichnis / KICKOFF bestimmt die Lane. Jeder Strom hat seinen eigenen KICKOFF (`nico-block/KICKOFF.md`, `luke-block/KICKOFF.md`, `dein-pc-KICKOFF.md`), der Lane + Kollisions-Regeln nennt.
- **Erste Aktion jeder Session:** `git pull --rebase` + `collision-map.md` lesen.

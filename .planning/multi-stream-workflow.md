# Multi-Stream-Workflow — 3 Ströme parallel (Marathon-Modus, Branch-Modell)

> Erweitert `two-terminal-nico-workflow.md` von 2 auf **3 parallele Bau-Ströme** für den unbeaufsichtigten Tag (Darien weg). Gilt ab 2026-06-09.
> **Begleitdokumente:** Kollisions-/Branch-Regeln → `collision-map.md` · Review-System → `reviews/_TEMPLATE.md` · Phasen + Lanes → `module-phase-plans.md` · Delegations-Pakete → `nico-block/` · `luke-block/` · Dein-PC → `dein-pc-KICKOFF.md` · Copy-Paste-Starttexte → `HANDOFF-TEXTS.md`.

## Die drei Ströme
| | **N — Nico** | **D — Dein-PC** | **L — Luke** |
|---|---|---|---|
| Maschine | Nicos PC | Dariens PC (remote) | Lukes PC |
| Treiber/Überwachung | Nico | Nico od. Luke (remote, semi-autonom) | Luke |
| Branch | `marathon/nico` | `marathon/dein-pc` | `marathon/luke-fe` (+ vorm. Backend-Repo) |
| Lane | wiki·formulare·berichte·notifications | calendar·dokumente·zeiterfassung | vorm. Backend-P0 / nachm. vertraege·dashboard·profil |
| KICKOFF | `nico-block/KICKOFF.md` | `dein-pc-KICKOFF.md` | `luke-block/KICKOFF.md` |

`main` bleibt heute **eingefroren** — alle bauen auf ihrem Branch ab demselben Stand. Darien merged + reviewt, wenn er zurück ist (`collision-map.md §5`).

## Tagesablauf pro Strom (immer gleich — gilt für ALLE Claudes)
1. **Session-Start (einmal):** `git checkout main && git pull && git checkout -B marathon/<strom>`. Dann lesen: `collision-map.md` (PFLICHT), eigener KICKOFF, `multi-stream-workflow.md` (dieses Doc).
2. **Nächste Phase bestimmen:** in `module-phase-plans.md` die nächste offene Phase **deiner Lane** (Module sind mit „→ Strom X" markiert).
3. **Bauen — die immer gleiche 6-Schritte-Schleife** (Detail: `nico-block/WORKFLOW.md`, gilt für alle):
   bauen → **i18n ×4** (`{var}` nicht `{{var}}`, Plural als ICU) → Demo-Handler falls nötig → **gescopter Typecheck** (nur geänderte Dateien, `tsconfig.<phase>check.json`) → **Playwright-Screenshot-QA** (`scripts/qa-<modul>-*.mjs`) → **Screenshots wirklich mit Read ansehen** (Roh-Keys? Emojis? Layout? leere Zustände?) → iterieren bis grün.
4. **Review-Faden notieren:** nach jeder fertigen Phase einen Eintrag in `reviews/<modul>.md` (Hinklick-Pfad · worauf achten · Screenshots · offene Punkte). Das ist die Vorlage für Dariens Feinschliff-Review.
5. **Commit + Branch-Push:** ein Commit pro Phase → `git push -u origin marathon/<strom>`. **Nie nach `main`.**
6. **Backend-Bedarf** → `backend-handover-luke.md` ergänzen (nicht selbst Backend bauen, außer Luke vormittags).

## Harte Regeln (für alle, nicht verhandelbar)
- **i18n ×4**, einfache Klammern, ICU-Plural · **keine Emojis im UI** · **keine ASCII-Umlaute** (ä/ö/ü/ß echt) · **wiederverwendbar in `shared/` bauen** · keine sichtbaren Scrollbars · Zurück-Buttons in Detail-Views.
- **Motion:** nur `transform`/`opacity` (GPU), Tokens aus `lib/motion.ts`.
- **Nur deine Lane**, nur dein Branch, Hot Files nur additiv (`collision-map.md §2`).
- **„Kompiliert ja" reicht nicht** — die Screenshots müssen angeschaut werden.

## Cadence (Darien-Entscheidung)
**Erst alle bauen, Reviews danach + aufgeteilt.** Kein Live-Review-Gate während des Laufs — die Build-+-Verify-Schleife ist die Selbstkontrolle. Qualität vor Tempo: lieber 3 Phasen sauber als 6 halb. Realistisch **3-5 Phasen/Strom/Tag**.

## Eskalation
- **Build/QA kriegst du nicht grün:** Phase im Review-Faden als „blockiert + warum" markieren, zur nächsten Phase deiner Lane springen (nicht hängenbleiben).
- **Domänen-Unklarheit:** im Review-Faden als „offene Frage" notieren, sinnvollen Default bauen, weiter. Darien klärt beim Review.
- **Du müsstest etwas außerhalb deiner Lane / an `main` anfassen:** NICHT tun → notieren, Darien macht's beim Merge.

# KICKOFF — Strom D (Dein-PC, remote-gefahren). Copy-Paste an den Anfang jeder Session.

> Wer auch immer Dariens PC remote fährt (Nico oder Luke): Claude Code im Repo-Root öffnen (`claude`), diesen Text zu Sessionbeginn geben. Semi-autonom — du baust eigenständig, der Treiber schaut periodisch drüber.

---

## Rolle & Auftrag (für Claude)
Du baust am Cosmi-CRM (Electron + React 19 + TypeScript Desktop-CRM) als **Strom D** in einem 3-Strom-Marathon (Nico · Dein-PC · Luke). Du arbeitest auf Dariens Hauptklon, aber **auf einem eigenen Branch** — `main` bleibt unangetastet. Du übernimmst die technische Führung und verifizierst selbst, bevor etwas „fertig" ist. Gilt für jede Session.

## Pflicht-Lektüre (zuerst)
1. `.planning/collision-map.md` — **Branch-/Kollisions-Regeln** (Branch `marathon/dein-pc`, nur deine Lane, Hot Files additiv).
2. `.planning/multi-stream-workflow.md` — Tagesablauf + harte Regeln.
3. `.planning/nico-block/RUNBOOK.md` + `.planning/nico-block/WORKFLOW.md` — der **exakte Build-+-Verify-Prozess** (gilt für alle Ströme: gescopter tsc + Playwright-Screenshot-QA + Screenshots wirklich ansehen + Mandatory-Verify-Checkliste).
4. Die aktuelle Phasen-Spec deiner Lane (siehe unten / `module-phase-plans.md`).
5. `CLAUDE.md` (Repo-Root) — Projektkontext + Architektur + UI/UX-Direktiven.

## Branch-Regeln (Marathon)
- Einmal am Anfang: `git checkout main && git pull && git checkout -B marathon/dein-pc`.
- Pro Phase ein Commit (Conventional, Englisch, **keine** AI-Attribution) → `git push -u origin marathon/dein-pc`.
- **Nie nach `main` pushen. Nie `main` auschecken/mergen. `main` heute nicht mitten am Tag pullen.** Darien merged + reviewt später.

## Lane & Phasen (nur diese Module)
**Fertige Pilot-Specs:** `.planning/dein-pc-block/` (README → `calendar-p1-views.md` START → `zeiterfassung-p1-standalone.md` → `dokumente-settings.md`). Danach Folgephasen aus dem Plan.
In `module-phase-plans.md` mit „→ Strom D" markiert:
- **calendar** (nur Layout-Shell, Views = Platzhalter): **P1 Views** (Tag/Woche/Monat + Mini-Cal + Kalender-Liste) — reines FE, guter Start · P2 Events-CRUD + RRULE + DnD + Erinnerungen (mock-first) · P3 mehrere/geteilte Kalender (FE) · …
- **dokumente** (sehr vollständig): P1 „Coming-soon" beseitigen (Move/Copy, granulare Rechte, Datei-Kommentare — mock-first) · P2 Metadaten/Tags · …
- **zeiterfassung** (dünner Wrapper um Profil-Tab): P1 Standalone-Modul (Timer + Projekt/Kunde-Zuordnung, Stundenkonten-Saldo mock-first) · …
Backend-schwere Teile → `backend-handover-luke.md`, FE mock-first weiterbauen. Du baust **nur** calendar/dokumente/zeiterfassung.

## Bau-Schleife (pro Phase, immer gleich)
bauen → **i18n ×4** (`{var}`, ICU-Plural) → Demo-Handler falls nötig → **gescopter Typecheck** (`tsconfig.<phase>check.json`, nur geänderte Dateien) → **Playwright-QA** (`scripts/qa-<modul>-*.mjs`) → **Screenshots mit Read ansehen** (Roh-Keys/Emojis/Layout/leere Zustände) → iterieren bis grün → **Review-Faden** in `reviews/<modul>.md` → commit → `git push origin marathon/dein-pc`.

## Skills
`frontend-design` (auto), `/run`, `/verify`, `/code-review medium`, optional `/polish` `/critique`. Screenshot-Ansehen = Prozess (WORKFLOW.md), kein Skill.

## Harte Regeln (Kurzform)
i18n ×4 · keine Emojis · keine ASCII-Umlaute (ä/ö/ü/ß echt) · `shared/` wiederverwenden · nur `transform`/`opacity` animieren · keine sichtbaren Scrollbars · Zurück-Buttons in Detail-Views · „kompiliert" reicht nicht — Screenshots ansehen · nur deine Lane + dein Branch.

## Eskalation
Build/QA nicht grün → Phase im Review-Faden als „blockiert + warum", nächste Phase deiner Lane. Domänen-Unklarheit → „offene Frage" notieren, sinnvollen Default bauen, weiter. Etwas außerhalb der Lane/an `main` nötig → NICHT tun, notieren.

# KICKOFF — Strom L (Luke). Copy-Paste an den Anfang jeder Session.

> Luke: Öffne Claude Code im Repo-Root (`claude`) und gib ihm zu Sessionbeginn diesen Text (oder den Block aus `../HANDOFF-TEXTS.md`). Gilt für JEDE Session, nicht nur die erste.

---

## Rolle & Tagesstruktur (für Claude)
Du arbeitest mit Luke am Cosmi-CRM (Electron + React 19 + TypeScript Desktop, Go-Microservices-Backend). Heute läuft ein **3-Strom-Marathon** (Nico, Dein-PC, Luke) parallel; du bist **Strom L**.

- **Vormittag — Backend-P0 (Lukes Domäne):** die Launch-Blocker aus `.planning/backend-handover-luke.md`, **P0 zuerst** (E-Rechnung/GoBD/DATEV/Bexio, Online-Terminbuchung, Dialer-Consent, DSGVO). Luke führt fachlich; du unterstützt beim Go-/gRPC-/Migrations-Code nach den Architektur-Regeln in `CLAUDE.md` (Thick Services / Thin Handlers, golang-migrate, slog, tenant_id).
- **Nachmittag — FE-Lane (wie Nico):** Frontend-Module **vertraege → dashboard → profil**, eine Phase nach der anderen, mit der vollen Build-+-Verify-Schleife.

## Pflicht-Lektüre (zuerst lesen)
1. `.planning/collision-map.md` — **Branch-/Kollisions-Regeln** (du baust FE auf `marathon/luke-fe`, nur deine Lane, Hot Files additiv).
2. `.planning/multi-stream-workflow.md` — der Tagesablauf + harte Regeln.
3. `.planning/nico-block/RUNBOOK.md` + `.planning/nico-block/WORKFLOW.md` — der **exakte FE-Build-+-Verify-Prozess** (gilt für alle Ströme: gescopter Typecheck + Playwright-Screenshot-QA + Screenshots wirklich ansehen + Mandatory-Verify-Checkliste).
4. `.planning/luke-block/RUNBOOK.md` — Setup + AM/PM-Spielregeln.
5. Vormittag: `.planning/backend-handover-luke.md`. Nachmittag: die aktuelle FE-Phasen-Spec (`phase-01-vertraege-settings.md`, dann `BACKLOG.md`).
6. `CLAUDE.md` (Repo-Root) — Projektkontext + Architektur-Regeln.

## Branch-Regeln (Marathon, wichtig)
- FE: **`git checkout main && git pull && git checkout -B marathon/luke-fe`** einmal am Anfang. Pro Phase ein Commit, dann `git push -u origin marathon/luke-fe`. **Nie nach `main`, nie `main` mergen.** (Abweichung vom Nico-Pilot-Pro-Phase-Branch — wir nutzen einen Strom-Branch.)
- Backend: auf dem üblichen Backend-Weg (eigenes Repo/Branch), nicht im FE-Strom-Branch.

## Lane & Phasen
Deine FE-Module + Phasen stehen in `.planning/module-phase-plans.md` (mit „→ Strom L" markiert):
- **vertraege** (FE-only auf Zustand-Store): P-Settings (Pilot) · P1 Backend-Anbindung-UI/Audit-Log/Erinnerungen · P2 Dokumente echt · P3 E-Signatur · P4 CRM/Finanzen-Verknüpfung.
- **dashboard** · **profil** danach (Phasen im Plan).
Du baust **nur** diese Module. Nichts außerhalb der Lane.

## Review-Pflicht (nach JEDER FE-Phase)
Eintrag in `.planning/reviews/<modul>.md` (aus `_TEMPLATE.md`): Hinklick-Pfad · worauf achten · Screenshots · offene Punkte. Das ist Dariens Vorlage für den Feinschliff-Review.

## Skills
`frontend-design` (auto), `/run`, `/verify`, `/code-review medium`, optional `/polish` `/critique`. Screenshot-Ansehen ist Prozess, kein Skill (siehe WORKFLOW.md).

## Harte Regeln (Kurzform)
i18n ×4 (`{var}`, ICU-Plural) · keine Emojis · keine ASCII-Umlaute · `shared/` wiederverwenden · nur `transform`/`opacity` animieren · „kompiliert" reicht nicht — Screenshots ansehen · nur deine Lane + dein Branch.

# Luke-Block — Delegations-Paket „Backend-P0 (vorm.) + FE-Lane vertraege·dashboard·profil (nachm.)"

Komplettes Paket, um Lukes Marathon-Tag zu fahren: **vormittags** die Launch-Blocker aus `backend-handover-luke.md`, **nachmittags** ein FE-Modul-Block wie Nico.

## Was Luke an seinen Claude gibt
- Zu Sessionbeginn den Starttext aus **`../HANDOFF-TEXTS.md` › „Strom L — Luke"** (oder: „lies `.planning/luke-block/KICKOFF.md`").

## Reihenfolge für Luke
1. **`KICKOFF.md`** — Rolle, Tagesstruktur (AM Backend / PM FE), Pflicht-Lektüre, Lane, Branch.
2. **`RUNBOOK.md`** — Setup + die AM/PM-Spielregeln; verweist für den FE-Build-+-Verify-Prozess auf `../nico-block/RUNBOOK.md` + `../nico-block/WORKFLOW.md` (identisch für alle FE-Ströme).
3. **Vormittag:** `backend-handover-luke.md` (P0 zuerst) — Lukes Domäne, er kennt den Stack.
4. **Nachmittag (fertige Pilot-Specs, der Reihe nach):** `phase-01-vertraege-settings.md` → `dashboard-settings.md` → `profil-p1-presence.md` → dann `BACKLOG.md` (Folgephasen).

## Branch & Kollision
- FE-Arbeit auf **`marathon/luke-fe`** (Branch-Modell, `../collision-map.md`). Backend auf dem Backend-Repo/seinem üblichen Branch.
- Nur die Lane-Module anfassen (vertraege · dashboard · profil). Hot Files nur additiv.

## Dateien
| Datei | Zweck |
|---|---|
| `README.md` | dieser Index |
| `KICKOFF.md` | Session-Start-Kontext (vollständiges Betriebshandbuch) |
| `RUNBOOK.md` | Setup + AM/PM-Regeln + Verweis auf gemeinsamen FE-Prozess |
| `phase-01-vertraege-settings.md` | FE-Pilot (Modul-Einstellungen, gemustert) |
| `BACKLOG.md` | restliche FE-Phasen der Lane |

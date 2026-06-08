# Starttexte für die 3 Claude-Ströme (Copy-Paste)

> **Darien gibt morgen früh jedem Strom seinen Block** (in Claude Code im Repo-Root einfügen). Jeder Block ist selbsttragend: der Claude liest die Pflicht-Docs, kennt seine Lane, seinen Branch und die Regeln. **Erst `git pull` auf `main`, dann den Block geben.**
> Reihenfolge des Tages: alle drei starten parallel, bauen ihre Lane, pushen ihren Strom-Branch. Darien merged + reviewt später (Branch-Modell, `collision-map.md`).

---

## ▶ Strom N — Nico (Nicos PC)

```
Wir arbeiten heute im 3-Strom-Marathon am Cosmi-CRM. Du bist Strom N (Content-Block).

Lies ZUERST in dieser Reihenfolge und befolge sie strikt:
1. .planning/collision-map.md  (Branch-/Kollisions-Regeln — PFLICHT)
2. .planning/multi-stream-workflow.md  (Tagesablauf + harte Regeln)
3. .planning/nico-block/KICKOFF.md + .planning/nico-block/RUNBOOK.md + .planning/nico-block/WORKFLOW.md  (Rolle + Build-+-Verify-Prozess)

Deine Lane (NUR diese Module, in dieser Reihenfolge): wiki → formulare → berichte → notifications.
Die Phasen je Modul stehen in .planning/module-phase-plans.md (Module mit „→ Strom N" markiert).

Branch-Modell (Marathon): einmal `git checkout main && git pull && git checkout -B marathon/nico`.
Pro Phase: bauen → i18n ×4 → gescopter tsc → Playwright-Screenshot-QA → Screenshots WIRKLICH mit Read ansehen → Review-Faden in .planning/reviews/<modul>.md → ein Commit → `git push -u origin marathon/nico`.
NIE nach main pushen, NIE main mergen. Hot Files (i18n/App.tsx/registry) nur additiv.

Beginne mit der nächsten offenen Phase von wiki. Bei Domänen-Unklarheit: sinnvollen Default bauen, im Review-Faden als „offene Frage" notieren, weiter. Qualität vor Tempo.
```

---

## ▶ Strom D — Dein-PC (Dariens PC, remote gefahren)

```
Wir arbeiten heute im 3-Strom-Marathon am Cosmi-CRM. Du bist Strom D, auf Dariens Hauptklon, aber auf EIGENEM Branch — main bleibt unangetastet.

Lies ZUERST in dieser Reihenfolge und befolge sie strikt:
1. .planning/collision-map.md  (Branch-/Kollisions-Regeln — PFLICHT)
2. .planning/multi-stream-workflow.md  (Tagesablauf + harte Regeln)
3. .planning/dein-pc-KICKOFF.md  (deine Rolle + Lane)
4. .planning/nico-block/RUNBOOK.md + .planning/nico-block/WORKFLOW.md  (Build-+-Verify-Prozess — gilt für alle Ströme)

Deine Lane (NUR diese Module): calendar → dokumente → zeiterfassung.
Phasen je Modul in .planning/module-phase-plans.md (Module mit „→ Strom D" markiert). Guter Start: calendar P1 (Views — reines FE).

Branch-Modell: einmal `git checkout main && git pull && git checkout -B marathon/dein-pc`.
Pro Phase: bauen → i18n ×4 → gescopter tsc → Playwright-Screenshot-QA → Screenshots WIRKLICH ansehen → Review-Faden in .planning/reviews/<modul>.md → Commit → `git push -u origin marathon/dein-pc`.
NIE nach main pushen, NIE main mergen, main heute nicht mitten am Tag pullen. Hot Files nur additiv. Nur deine Lane.

Beginne mit calendar P1. Backend-schwere Teile → .planning/backend-handover-luke.md, FE mock-first weiterbauen.
```

---

## ▶ Strom L — Luke (Lukes PC)

```
Wir arbeiten heute im 3-Strom-Marathon am Cosmi-CRM. Du bist Strom L: VORMITTAGS Backend-P0, NACHMITTAGS FE-Lane.

Lies ZUERST:
1. .planning/collision-map.md  (Branch-/Kollisions-Regeln — PFLICHT)
2. .planning/multi-stream-workflow.md  (Tagesablauf)
3. .planning/luke-block/KICKOFF.md + .planning/luke-block/RUNBOOK.md  (deine AM/PM-Struktur)
4. Für den FE-Build: .planning/nico-block/RUNBOOK.md + .planning/nico-block/WORKFLOW.md (identischer Prozess für alle Ströme)

VORMITTAG — Backend-P0: .planning/backend-handover-luke.md, P0 zuerst (E-Rechnung/GoBD/DATEV/Bexio, Online-Terminbuchung, Dialer-Consent, DSGVO). Deine Domäne — Architektur nach CLAUDE.md (Thick Services/Thin Handlers, golang-migrate, slog, tenant_id). Backend-Commits auf deinem üblichen Repo/Branch.

NACHMITTAG — FE-Lane (NUR diese Module): vertraege → dashboard → profil. Pilot: .planning/luke-block/phase-01-vertraege-settings.md, dann BACKLOG.md. Phasen in .planning/module-phase-plans.md (Module mit „→ Strom L" markiert).
FE-Branch: einmal `git checkout main && git pull && git checkout -B marathon/luke-fe`. Pro Phase: bauen → i18n ×4 → gescopter tsc → QA → Screenshots ansehen → Review-Faden → Commit → `git push -u origin marathon/luke-fe`. NIE nach main. Hot Files additiv. Backend- und FE-Arbeit getrennt halten.
```

---

## Für Darien — beim Review (wenn zurück)
Pro Strom-Branch: Review-Fäden in `.planning/reviews/<modul>.md` durchgehen (Pfad nachklicken, Feinschliff), dann `git checkout main && git merge --no-ff marathon/<strom>` → Hot-File-Konflikte additiv lösen (beide Blöcke behalten) → Smoke + scoped tsc → push. Ein Strom nach dem anderen. Reviews unter dem Team aufteilbar (jeder reviewt einen fremden Strom).

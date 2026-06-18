# Starttexte für die 3 Claude-Ströme (Copy-Paste)

> **Darien gibt jedem Strom seinen Block** (in Claude Code im Repo-Root einfügen). Jeder Block ist selbsttragend: der Claude liest die Pflicht-Docs, kennt seine Lane, seinen Branch und die Regeln.

---

## ⚠ TAG 2 (2026-06-10) — FORTSETZUNG: zwei Anpassungen gegenüber den Blöcken unten

**Die Blöcke unten sind der Tag-1-Stand. Für Tag 2 gilt zusätzlich (ZUERST `RESUME-2026-06-10.md` lesen):**

1. **Branch FORTSETZEN, nicht neu erzeugen.** In den Blöcken unten steht `git checkout -B marathon/<strom>` — das ist NUR für Tag 1. **Tag 2 stattdessen:**
   ```bash
   git fetch origin && git checkout marathon/<strom> && git pull origin marathon/<strom>
   ```
   `checkout -B` würde den Branch von main überschreiben und die gestrige Arbeit löschen.
2. **Semi-autonom: ein Treiber ist da.** Bei Design-/Klärungsfragen **den anwesenden Treiber fragen** (statt still zu defaulten). Nur für Dariens Spezialwissen → „offene Frage für Darien" notieren. Details: `RESUME-2026-06-10.md`.
3. **Vor dem Bauen den echten Code-Stand prüfen** (Specs sind teils veraltet — gestern war calendar/kontakte längst weiter als der Plan sagte).

**Was gestern fertig wurde + nächste Phase pro Strom:** siehe Tabelle in `RESUME-2026-06-10.md`. Kurz: D → calendar fertig, weiter mit **dokumente** · N → berichte-Pilots fertig, weiter in der Lane · L → vorm. Backend-P0, nachm. **vertraege**.

---

# ▶▶ TAG-2-STARTTEXTE (2026-06-10) — diese hier benutzen (paste-and-go)

> Selbsttragend für Tag 2: bestehenden Branch fortsetzen, semi-autonom (Treiber fragen), echten Code-Stand prüfen. Die Tag-1-Blöcke ganz unten sind nur noch Referenz.

## ▶ Strom N — Nico (Tag 2)

```
Wir arbeiten weiter am Cosmi-CRM, 3-Strom-Marathon, Tag 2. Du bist Strom N (Content-Block, Nicos PC). SEMI-autonom: Nico ist da und beantwortet Design-/Klärungsfragen — frag ihn, statt still zu defaulten.

Session-Start (einmal):
  git fetch origin
  git checkout main && git pull              # Handoff-Docs lesen
  git checkout marathon/nico && git pull     # bestehenden Branch FORTSETZEN — NICHT checkout -B (löscht gestrige Arbeit)

Lies ZUERST: .planning/RESUME-2026-06-10.md, dann .planning/collision-map.md, .planning/multi-stream-workflow.md, .planning/DESIGN-DECISIONS.md, .planning/nico-block/KICKOFF.md + RUNBOOK.md + WORKFLOW.md.

Deine Lane: wiki · formulare · berichte · notifications (Branch marathon/nico). Gestern fertig: berichte KPI-Sparklines + Pilots 01/02. Nächste offene Phase in .planning/module-phase-plans.md (Module „→ Strom N") + Specs in .planning/nico-block/.

WICHTIG: Vor dem Bauen den ECHTEN Code-Stand prüfen (Specs teils veraltet — gestern war mehrfach schon mehr gebaut als der Plan sagte). Bei Abweichung: echten Gap bauen, im Review-Faden notieren. Bei Design-/Domänenunklarheit: Nico fragen.

Pro Phase: bauen → i18n ×4 ({var}, ICU-Plural) → gescopter tsc → Playwright-Screenshot-QA → Screenshots WIRKLICH mit Read ansehen → Review-Faden reviews/<modul>.md → ein Commit → git push origin marathon/nico. NIE nach main pushen/mergen, main nicht mitten am Tag pullen. Hot Files (i18n/App.tsx/registry) nur additiv. Qualität vor Tempo.
```

## ▶ Strom L — Luke (Tag 2)

```
Wir arbeiten weiter am Cosmi-CRM, 3-Strom-Marathon, Tag 2. Du bist Strom L (Lukes PC): VORMITTAGS Backend-P0, NACHMITTAGS FE-Lane. SEMI-autonom: Luke ist da und entscheidet Design-/Klärungsfragen.

Session-Start (einmal):
  git fetch origin
  git checkout main && git pull              # Handoff-Docs lesen

Lies ZUERST: .planning/RESUME-2026-06-10.md, dann .planning/collision-map.md, .planning/multi-stream-workflow.md, .planning/DESIGN-DECISIONS.md, .planning/luke-block/KICKOFF.md + RUNBOOK.md, für FE zusätzlich .planning/nico-block/WORKFLOW.md.

VORMITTAG — Backend-P0: .planning/backend-gaps.md (P0 zuerst: E-Rechnung/GoBD/DATEV, Online-Terminbuchung, Dialer-Consent, DSGVO; neu: kontakte-Backend — revisionssichere Beratungsprotokoll-Ablage + tenant-Settings). Architektur nach CLAUDE.md (Thick Services/Thin Handlers, golang-migrate, slog, tenant_id). Backend-Commits auf deinem üblichen Repo/Branch.

NACHMITTAG — FE-Lane (vertraege → dashboard → profil). FE-Branch (gestern noch nicht gepusht → frisch von main):
  git checkout marathon/luke-fe 2>/dev/null || git checkout -B marathon/luke-fe
Specs: .planning/luke-block/phase-01-vertraege-settings.md (START), dann dashboard-settings.md, profil-p1-presence.md. Phasen in module-phase-plans.md „→ Strom L".

WICHTIG: Vor dem Bauen echten Code-Stand prüfen (Specs teils veraltet). Pro FE-Phase: bauen → i18n ×4 → gescopter tsc → Screenshot-QA → Screenshots ansehen → Review-Faden → Commit → git push origin marathon/luke-fe. NIE nach main. Hot Files additiv. Backend- und FE-Arbeit getrennt halten.
```

## ▶ Strom D — Dein-PC / hier (Tag 2)

```
Wir arbeiten weiter am Cosmi-CRM, 3-Strom-Marathon, Tag 2. Du bist Strom D, auf Dariens Hauptklon, auf EIGENEM Branch — main bleibt unangetastet. SEMI-autonom: der Treiber (Nico oder Luke, remote) beantwortet Design-/Klärungsfragen.

Session-Start (einmal):
  git fetch origin
  git checkout main && git pull                  # Handoff-Docs lesen
  git checkout marathon/dein-pc && git pull      # bestehenden Branch FORTSETZEN — NICHT checkout -B (gestrige calendar-Arbeit!)

Lies ZUERST: .planning/RESUME-2026-06-10.md, dann .planning/collision-map.md, .planning/multi-stream-workflow.md, .planning/DESIGN-DECISIONS.md, .planning/dein-pc-KICKOFF.md, .planning/nico-block/WORKFLOW.md.

Deine Lane: calendar · dokumente · zeiterfassung (Branch marathon/dein-pc). Gestern fertig: calendar (Correctness + Serientermine + RSVP + Räume) — gilt als rund. NÄCHSTES MODUL: dokumente — aber ZUERST Spec gegen echten Code prüfen (modules/dokumente/ lesen), dann den echten Gap bauen. Danach zeiterfassung. Folgephasen in module-phase-plans.md „→ Strom D", Specs in .planning/dein-pc-block/.

WICHTIG: Specs sind teils veraltet — gestern war calendar längst weiter als der Plan sagte (fast Doppelarbeit). Immer echten Code-Stand prüfen. Bei Design-/Domänenunklarheit: Treiber fragen.

Pro Phase: bauen → i18n ×4 → gescopter tsc → Playwright-Screenshot-QA → Screenshots WIRKLICH ansehen → Review-Faden reviews/<modul>.md → Commit → git push origin marathon/dein-pc. NIE nach main, NIE main mergen, main nicht mitten am Tag pullen. Hot Files additiv. Backend-schweres → backend-gaps.md notieren, FE mock-first weiterbauen.
```

---

# (Tag-1-Referenz — superseded, nur zum Nachschlagen)

## ▶ Strom N — Nico (Nicos PC)

```
Wir arbeiten heute im 3-Strom-Marathon am Cosmi-CRM. Du bist Strom N (Content-Block).

Lies ZUERST in dieser Reihenfolge und befolge sie strikt:
1. .planning/collision-map.md  (Branch-/Kollisions-Regeln — PFLICHT)
2. .planning/multi-stream-workflow.md  (Tagesablauf + harte Regeln)
3. .planning/DESIGN-DECISIONS.md  (wie du Design-/Produktentscheidungen triffst wenn niemand danebensitzt — PFLICHT)
4. .planning/nico-block/KICKOFF.md + .planning/nico-block/RUNBOOK.md + .planning/nico-block/WORKFLOW.md  (Rolle + Build-+-Verify-Prozess)

Deine Lane (NUR diese Module, in dieser Reihenfolge): wiki → formulare → berichte → notifications.
Fertige Pilot-Specs: .planning/nico-block/wiki-settings.md, .planning/nico-block/formulare-settings.md (+ die bestehenden phase-01/02 für notifications/berichte). Folgephasen je Modul in .planning/module-phase-plans.md (Module mit „→ Strom N" markiert).

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
3. .planning/DESIGN-DECISIONS.md  (wie du Design-/Produktentscheidungen triffst wenn niemand danebensitzt — PFLICHT)
4. .planning/dein-pc-KICKOFF.md  (deine Rolle + Lane)
5. .planning/nico-block/RUNBOOK.md + .planning/nico-block/WORKFLOW.md  (Build-+-Verify-Prozess — gilt für alle Ströme)

Deine Lane (NUR diese Module): calendar → dokumente → zeiterfassung.
Fertige Pilot-Specs in .planning/dein-pc-block/: calendar-p1-views.md (START HIER), zeiterfassung-p1-standalone.md, dokumente-settings.md. Folgephasen in .planning/module-phase-plans.md (Module mit „→ Strom D" markiert).

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
3. .planning/DESIGN-DECISIONS.md  (wie du Design-/Produktentscheidungen triffst wenn niemand danebensitzt — PFLICHT, für die FE-Lane)
4. .planning/luke-block/KICKOFF.md + .planning/luke-block/RUNBOOK.md  (deine AM/PM-Struktur)
5. Für den FE-Build: .planning/nico-block/RUNBOOK.md + .planning/nico-block/WORKFLOW.md (identischer Prozess für alle Ströme)

VORMITTAG — Backend-P0: .planning/backend-handover-luke.md, P0 zuerst (E-Rechnung/GoBD/DATEV/Bexio, Online-Terminbuchung, Dialer-Consent, DSGVO). Deine Domäne — Architektur nach CLAUDE.md (Thick Services/Thin Handlers, golang-migrate, slog, tenant_id). Backend-Commits auf deinem üblichen Repo/Branch.

NACHMITTAG — FE-Lane (NUR diese Module): vertraege → dashboard → profil. Fertige Pilot-Specs: .planning/luke-block/phase-01-vertraege-settings.md (START HIER), dann dashboard-settings.md, profil-p1-presence.md; danach BACKLOG.md. Phasen in .planning/module-phase-plans.md (Module mit „→ Strom L" markiert).
FE-Branch: einmal `git checkout main && git pull && git checkout -B marathon/luke-fe`. Pro Phase: bauen → i18n ×4 → gescopter tsc → QA → Screenshots ansehen → Review-Faden → Commit → `git push -u origin marathon/luke-fe`. NIE nach main. Hot Files additiv. Backend- und FE-Arbeit getrennt halten.
```

---

## Für Darien — beim Review (wenn zurück)
Pro Strom-Branch: Review-Fäden in `.planning/reviews/<modul>.md` durchgehen (Pfad nachklicken, Feinschliff), dann `git checkout main && git merge --no-ff marathon/<strom>` → Hot-File-Konflikte additiv lösen (beide Blöcke behalten) → Smoke + scoped tsc → push. Ein Strom nach dem anderen. Reviews unter dem Team aufteilbar (jeder reviewt einen fremden Strom).

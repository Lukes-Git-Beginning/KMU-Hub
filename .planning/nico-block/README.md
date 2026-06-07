# Nico-Block — Delegations-Paket „Content & Self-Service"

Dieses Verzeichnis ist das komplette Paket, um den Block **wiki · formulare · berichte · notifications** an Nico zu delegieren.

## Was Darien an Nico gibt
1. Zugang zum Repo (Branch `main`, direct-to-main).
2. Den Hinweis: **„Installier Claude Code, dann lies und befolge `.planning/nico-block/README.md`."**

## Reihenfolge für Nico
1. **`RUNBOOK.md`** — einmal komplett lesen (Setup, harte Regeln, Workflow, Verify-Checkliste, Fallen, Review-Gate).
2. **`KICKOFF.md`** — das gibst du deinem Claude zu **Sessionbeginn** (Rolle, Skills, Repo-Landkarte, Konventionen).
3. **`phase-01-notifications-quiet-hours.md`** — die erste Pilot-Phase. Komplett abarbeiten, verifizieren, „fertig" melden, Review abwarten.
4. **`phase-02-berichte-sparkline.md`** — die zweite Pilot-Phase.
5. Erst nach grünem Review beider Piloten: **`BACKLOG.md`** — der Rest des Blocks (Specs kommen dann nach und nach).

## Für Darien / Haupt-Team (Claude)
- Der Pilot (Phase 01 + 02) ist der **Test**, ob die Delegation für Nico trägt. Review streng gegen die Definition-of-Done jeder Spec.
- Läuft der Pilot sauber → Block freigeben, Backlog-Specs schreiben, skalieren.
- Hohe Fehlerrate → enger gefassten Scope wählen (z.B. nur das ModuleSettingsShell-Muster) oder Block zurückziehen.
- **Realismus:** 3–5 Phasen/Tag/Person bei Qualität, nicht 10. Engpass ist das Review-Gate, nicht das Bauen.

## Dateien
| Datei | Zweck |
|---|---|
| `README.md` | dieser Index |
| `RUNBOOK.md` | Regeln, Workflow, Verify-Checkliste, Review-Gate |
| `KICKOFF.md` | Session-Start-Kontext für Nicos Claude (Skills + Repo-Map) |
| `phase-01-notifications-quiet-hours.md` | Pilot 1 (niedriges Risiko) |
| `phase-02-berichte-sparkline.md` | Pilot 2 (visuell, gemustert) |
| `BACKLOG.md` | restliche Block-Phasen, nach Risiko sortiert |

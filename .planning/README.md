# .planning — Arbeits- und Planungs-Ablage

Scratch-Space für Session-Planung, Tracker und Handoffs. **Nicht** die Single Source of Truth —
verbindliches Wissen liegt in [`docs/`](../docs/) und [`.knowledge/`](../.knowledge/).

> **Aufgeräumt am 2026-08-06:** Der Root war auf 80 Einträge gewachsen, überwiegend tote
> RESUME-/batch-/VISION-Dateien aus Mai/Juni. 53 davon liegen jetzt in `archiv/`. Was hier steht,
> beschreibt offene Arbeit — wenn eine Datei das nicht mehr tut, gehört sie ins Archiv.

## Aktiv

### Steuerung

| Datei | Zweck |
|---|---|
| `MASTER-PLAN.md` | **DER Master-Plan (Cosmi 1.0, Frontend).** Phasen, Batch-Queue, Review-Pipeline. Löst `MASTER-TRACKER.md` ab. |
| `RESUME-NEXT.md` | **Einstiegspunkt der nächsten Session** (Frontend-Track, Darien) |
| `SESSION-RUNBOOK.md` | Wiederholbarer Bau-Zyklus: laden → planen → recherchieren → fragen → bauen → QA → speichern |
| `AUTONOMOUS-RUN.md` | Protokoll für „mach autonom weiter" |
| `status-overview.md` | **Live-Status-Snapshot** — gemessene Kennzahlen, Modul-Reifegrad, offene Posten. Stand 2026-08-06 |
| `status-overview.prompt.md` | Generator-Prompt, mit dem der Snapshot neu erzeugt wird |

### Offene Arbeitspakete

| Datei | Zweck |
|---|---|
| `backend-gaps.md` | Backend-Gap-Tracker (was das Frontend zum Andocken braucht) |
| `wertelisten-und-fokus-naechste-runde.md` | Wertelisten + Fokus-Verhalten, nächste Runde |
| `editor-spalten-naechste-runde.md` | Editor-Spalten, nächste Runde |
| `intake-pilot-feedback.md` · `intake-pilot-review-checklist.md` | Helpdesk-Intake-Pilot: Feedback + Review-Checkliste |
| `security-echtschaltung-luke.md` | security/DSGVO-Echtschaltung — FE-Teil ✅, **Backend-Teil offen (Luke)** |
| `bexio-review-paket.md` | Bexio-Invoice-Pull, Review-Paket für Darien |
| `hetzner-review-checklist.md` | Team-Review-Agenda auf der laufenden Cosmi-exe (Darien + Luke + Nico) |

### Blöcke (Verzeichnisse)

| Verzeichnis | Zustand |
|---|---|
| `backend-block/` | **aktiv** — Nachtloop (`loop/`: Backlog, Journal, Gate-Kommandos), Übergaben, Wellen-Briefing |
| `customization-block/` | **aktiv** (08-06) |
| `helpdesk-intake-block/` | **aktiv** (08-06) |
| `rbac-block/` | **aktiv** (07-26) — RBAC-Phasen-Briefings |
| `nico-block/` | **Build-+-Verify-Workflow-Standard**, von `CLAUDE.md` referenziert |
| `branchen-block/` · `coworking-block/` | Konzepte, ruhend (07-21) |
| `legal/` · `meeting-parity/` · `parallel-batch/` · `reviews/` | ruhend (Juni), noch nicht archiviert |

## Archiv

`archiv/` (80 Einträge) enthält abgeschlossene Sprint-/Wellen-Pläne, Handoffs, alte
RESUME-Snapshots, VISION-Dokumente und Delegations-Blöcke. Nichts davon ist gelöscht — die Dateien
sind per `git mv` verschoben und über die Git-History nachvollziehbar.

Darunter auch `MASTER-TRACKER.md` (Stand 19.06., von `MASTER-PLAN.md` abgelöst) und
`BACKEND-PLAN.md` (vom Nachtloop in `backend-block/` abgelöst).

## Hygiene

- Build-Artefakte (`pdf-build/`, `*.pdf`) und QA-Output (`desktop/scripts/screenshots/`) sind
  `.gitignore`d und regenerierbar (`node desktop/scripts/build-status-pdf.mjs`).
- Eine Datei mit `RESUME-`, `-DONE` oder abgeschlossenem `VISION`-Inhalt gehört nach dem Abschluss
  direkt nach `archiv/` — nicht liegen lassen.

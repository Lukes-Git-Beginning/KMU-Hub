# .planning — Arbeits- und Planungs-Ablage

Scratch-Space für Session-Planung, Tracker und Handoffs. **Nicht** die Single Source of Truth —
verbindliches Wissen liegt in [`docs/`](../docs/) und [`.knowledge/`](../.knowledge/).

## Aktiv (laufend gepflegt)

| Datei | Zweck |
|---|---|
| `MASTER-PLAN.md` | **DER Master-Plan (Cosmi 1.0, Frontend).** Single Source zum Abarbeiten — Phasen, Batch-Queue, Review-Pipeline. Stand verifiziert 23.06. |
| `BACKEND-PLAN.md` | **Backend-Plan (Luke-Track), parallel zu MASTER-PLAN.** Backend-Lücken nach Priorität + FE↔BE-Warte-Mapping (was wartet worauf). |
| `SESSION-RUNBOOK.md` | **Wiederholbarer Bau-Zyklus.** Trigger „mach an den Phasen weiter" → laden→planen→recherchieren→fragen→bauen→QA→speichern. 2-Terminal-Modell. |
| `MASTER-TRACKER.md` | ⛔ Abgelöst (Historie) — siehe `MASTER-PLAN.md` |
| `status-overview.md` · `status-overview.prompt.md` | Live-Status-Snapshot (Modul-Reifegrad, Blocker-Burndown) + Generator-Prompt |
| `RESUME-NEXT-SESSION.md` | Resume-Pointer für die nächste Session |
| `fe-wiring-welle-NEXT-SESSION.md` | FE↔Backend-Wiring **Welle 3** (offen: fuhrpark/einkauf/produktion) |
| `aufraeum-welle-5-NEXT.md` | Aufräum-Welle-5 — nächste Schritte |
| `backend-gaps.md` | Backend-Gap-Tracker |
| `work-tiefe-pass.md` | Work-Modul Tiefen-Pass |
| `bexio-scope-check.md` | Bexio-Integrations-Scope (G12 = bidirektionaler Invoice-Pull noch offen) |
| `doku-konsolidierung.prompt.md` | Prompt der Doku-/Knowledge-Konsolidierungs-Session (2026-06-18) |
| `nico-block/` | **Build-+-Verify-Workflow-Standard** (von `CLAUDE.md` + `CONTRIBUTING.md` referenziert) |

## Archiv

`archiv/` enthält abgeschlossene Sprint-/Wellen-Pläne, Handoffs, alte RESUME-Snapshots und
Delegations-Blöcke (Jun 2026). Inhalte bleiben über Git-History und im Knowledge-Vault/Memory
nachvollziehbar — hier nur aus dem aktiven Arbeitsset herausgenommen.

## Hygiene

Build-Artefakte (`pdf-build/`, `*.pdf`) und QA-Output (`desktop/scripts/screenshots/`) sind
`.gitignore`d und regenerierbar (`node desktop/scripts/build-status-pdf.mjs`).

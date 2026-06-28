# Prompt — Projekt-Status-Snapshot (Diagramm-Übersicht)

> Wiederverwendbarer Prompt für eine **rein deskriptive** Standortbestimmung des Cosmi/Zentria-CRM.
> Zweck: „Wo stehen wir gerade?" als Diagramme — **keine** Priorisierung, **keine** Empfehlung was als
> Nächstes dran ist, **keine** Rückfrage zur Auswahl. Einfach: Ist-Stand, visualisiert.
>
> Aufruf: Inhalt dieses Files als Prompt an Claude Code in diesem Repo geben (oder als `/status-snapshot`-Command
> hinterlegen). Optional mit `ultracode`/Workflow für parallele Quellen-Reader.

---

## Rolle & Ziel
Du bist Projekt-Analyst für das KMU-Hub-/Cosmi-CRM (Marke: Software „Cosmi", Firma „Zentria"). Erzeuge einen
**visuellen Status-Snapshot** des Gesamtprojekts: Phase, Modul-Reife, Roadmap-Lage, Architektur, Launch-Reife,
Multi-Tenancy-/RLS-Stand, Qualität/CI. Ergebnis ist ein Markdown-Dokument mit Mermaid-Diagrammen und knappen
Bildunterschriften.

## Quellen — in dieser Reihenfolge lesen, nichts annehmen
1. `README.md` — Projektüberblick, Tech-Stack, Setup
2. `docs/ROADMAP.md` — **Single Source of Truth**, 6-Sprint-Plan bis Launch **2026-09-01**
3. `docs/MODULES_SCOPE_MATRIX.md` — 14 Module × Tabellen/RPCs/Hooks/Flag-Keys/Pilot-Prio/Sprint
4. `docs/ARCHITECTURE.md` — ADRs / Architektur-Entscheidungen
5. `.knowledge/_index.md` + `.knowledge/milestones.md` — Master-Index, Meilensteine, Rigorosum-Runden 1+2
6. `CLAUDE.md` — Architektur-Regeln, Feature-Flags, aktueller Projekt-Kontext
7. Der Auto-Memory-Index `MEMORY.md` (wird je Session geladen) — letzter Sprint-/Wellen-Stand, Launch-Blocker,
   Prod-Stand (Migrationskopf, Smoke)
8. Live-Repo-Signale (nicht raten — messen):
   - `git log --oneline -30` (zuletzt gelieferte Features)
   - höchste Migration: `ls backend/migrations/*.up.sql | tail -1`
   - Service-Zahl: `backend/cmd/*` zählen
   - Feature-Flag-Registry: `backend/internal/featureflag/`
   - Test-/CI-Lage: `.github/workflows/` (ci/nightly/scans-Split)

Markiere jeden Wert, den du nicht belegen kannst, explizit als „(geschätzt)". Datiere den Snapshot mit dem
heutigen Datum.

## Zu erzeugende Diagramme (Mermaid, GitHub-rendert) — mindestens diese
1. **Modul-Reifegrad-Matrix** — die ~14 Fachmodule (crm, helpdesk, wiki, schichten, fuhrpark, vermietung,
   rapporte, produktion, einkauf, inventar, hr/zeiterfassung, finanzen, dialer, …). Markdown-Tabelle mit
   Ampeln (✅ voll · 🟡 teilweise · ⬜ Stub/offen) je Dimension **Backend-RPCs · FE-Wiring · Live-Flag ·
   Pilot-Prio**, **plus** ein Mermaid-`pie` der Status-Verteilung.
2. **Roadmap-Gantt** — `gantt` der 6 Sprints bis 2026-09-01: abgeschlossene als `done`, laufender als `active`,
   Rest Default; Meilensteine UG-Gründung 01.06 + Launch 01.09 als `milestone`.
3. **Architektur-Überblick** — `graph LR`: Desktop (Electron/React) + PWA → API-Gateway (Go) → gRPC-Service-
   Cluster (24) → PostgreSQL 16 / Redis / LiveKit; Feature-Flag- und Consent-Layer andeuten; WASM-Plugin =
   OFF kennzeichnen.
4. **Launch-Reife / Blocker-Burndown** — Rigorosum Runde 1+2: P0/P1/P2/P3 erledigt vs. offen (Mermaid `pie`
   oder Balken-Surrogat); aktuelle kombinierte Launch-Note nennen.
5. **Multi-Tenancy / RLS-Retrofit** — Option-B-Rollout (tenant_id + RLS über ~50 Tabellen): Fortschritts-`pie`
   (retrofittet vs. offen) mit aktuellem Migrationskopf + Datum des Scharfschaltens (`COSMI_ENV=production`).
6. **Qualitäts-/CI-Status** — CI-Pipeline-Split (per-push `ci.yml`: lint/test/e2e/openapi · `nightly.yml`: smoke ·
   `scans.yml`: gosec/trivy/npm-audit), Test-Coverage-Lage, letzter grüner Prod-Deploy/Smoke — kompakte Tabelle
   oder `graph`.

Weitere sinnvolle Diagramme erlaubt, z. B.: **Integrationen-Landkarte** (Bexio/Lexware/DATEV/LiveKit/OnlyOffice),
**Pilot-Timeline** (Dienstleister Juli→Okt, Handwerk ab Nov), **Modul-Abhängigkeits-Graph**.

## Output-Format
- Ein Markdown-Dokument: **„Projekt-Status-Snapshot — Cosmi/Zentria CRM (Stand: <heutiges Datum>)"**.
- Oben: 3–5 Sätze Executive Summary (Phase, Launch-Datum, grobe Gesamtreife/Note, was zuletzt live ging).
- Danach die Diagramme, je mit 1–2 Sätzen Caption.
- Abschluss-Abschnitt „Datenbasis & Annahmen": welche Dateien gelesen, was „(geschätzt)" ist.
- Sprache Deutsch (Umlaute korrekt), Code/Identifier englisch. Mermaid syntaktisch valide halten.

## Leitplanken (wichtig)
- **Deskriptiv, nicht präskriptiv.** Beschreibe den Ist-Stand. Empfiehl nichts, priorisiere nichts, frage nichts ab.
- Belege Zahlen aus den Quellen; keine erfundenen Metriken. Diskrepanzen zwischen Quellen offen benennen.
- Optional als Multi-Agent-Workflow (parallele Reader je Quelle → Synthese), wenn der User „ultracode"/Workflow
  wünscht — sonst inline.

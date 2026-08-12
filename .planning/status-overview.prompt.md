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
1. `.planning/launch-lagebild-2026-08-12.md` — **Single Source of Truth für Lage und Sequenz.** Ziel ist
   **Produkt 1.0.0** nach Reifegrad-Gates, **kein Kalenderdatum**. §3 = prüfbare 1.0.0-Definition,
   §4 = offene Befunde G0/G1/G2, §6 = Etappen 0–4 mit Eintritts-/Austrittskriterien, §7 = Streichliste.
2. `README.md` — Projektüberblick, Tech-Stack, Setup
3. `docs/ROADMAP.md` — ⚠ **Kalendermodell überholt** (Launch 2026-09-01, Sprint S0–S5, ZFA-Pilot sind
   entwertet). Weiter brauchbar: gemessene Kennzahlen und Modul-Realisierungs-Matrix. **Keine Termine
   von dort übernehmen.**
4. `docs/MODULES_SCOPE_MATRIX.md` — 14 Module × Tabellen/RPCs/Hooks/Flag-Keys/Pilot-Prio/Sprint
5. `docs/ARCHITECTURE.md` — ADRs / Architektur-Entscheidungen
6. `.knowledge/_index.md` + `.knowledge/milestones.md` — Master-Index, Meilensteine, Rigorosum-Runden 1+2
7. `CLAUDE.md` — Architektur-Regeln, Feature-Flags, aktueller Projekt-Kontext
8. Der Auto-Memory-Index `MEMORY.md` (wird je Session geladen) — letzter Lauf-/Arbeitsstand, offene
   Befunde, Prod-Stand (Migrationskopf, Smoke)
9. Live-Repo-Signale (nicht raten — messen):
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
2. **Etappen-/Gate-Flowchart** — `flowchart LR` der Etappen 0–4 aus Lagebild §6, jeweils gefolgt von ihrem
   Gate: `Etappe 0 → Gate 0 → Etappe 1 → Gate 1 → …`. **Ohne Kalenderdaten, ohne `gantt`** — die Sequenz
   ist durch Eintritts-/Austrittskriterien getaktet, nicht durch Termine. Etappe N+1 beginnt, wenn N belegt
   abgeschlossen ist. Erreichte Gates markieren, das aktuell laufende hervorheben. Kein Launch-Datum und
   kein ZFA-Meilenstein — beides ist seit 2026-08-12 entwertet.
3. **Architektur-Überblick** — `graph LR`: Desktop (Electron/React) + PWA → API-Gateway (Go) → gRPC-Service-
   Cluster (24) → PostgreSQL 16 / Redis / LiveKit; Feature-Flag- und Consent-Layer andeuten; WASM-Plugin =
   OFF kennzeichnen.
4. **Befund-Burndown gegen 1.0.0** — die G0-/G1-/G2-Befunde aus Lagebild §4: geschlossen vs. offen, mit
   Personentagen (Mermaid `pie` oder Balken-Surrogat). Dazu die fünf 1.0.0-Kriterien aus §3 mit
   erfüllt/nicht erfüllt. Die Rigorosum-P0–P3-Zählung ist Historie und gehört, wenn überhaupt, in eine
   Nebenbemerkung — nicht in die Hauptaussage.
5. **Multi-Tenancy / RLS-Retrofit** — Option-B-Rollout (tenant_id + RLS über ~50 Tabellen): Fortschritts-`pie`
   (retrofittet vs. offen) mit aktuellem Migrationskopf + Datum des Scharfschaltens (`COSMI_ENV=production`).
6. **Qualitäts-/CI-Status** — CI-Pipeline-Split (per-push `ci.yml`: lint/test/e2e/openapi · `nightly.yml`: smoke ·
   `scans.yml`: gosec/trivy/npm-audit), Test-Coverage-Lage, letzter grüner Prod-Deploy/Smoke — kompakte Tabelle
   oder `graph`.

Weitere sinnvolle Diagramme erlaubt, z. B.: **Integrationen-Landkarte** (Bexio/Lexware/DATEV/LiveKit/OnlyOffice),
**Modul-Abhängigkeits-Graph**. **Keine Pilot-Timeline** — es gibt keinen Piloten und keine Segment-Staffel.

## Output-Format
- Ein Markdown-Dokument: **„Projekt-Status-Snapshot — Cosmi/Zentria CRM (Stand: <heutiges Datum>)"**.
- Oben: 3–5 Sätze Executive Summary (aktuelle Etappe, erfüllte und offene 1.0.0-Kriterien, grobe
  Gesamtreife, was zuletzt live ging). **Kein Launch-Datum und kein Countdown** — wenn eine Quelle eines
  nennt, ist die Quelle veraltet, nicht die Lage.
- Danach die Diagramme, je mit 1–2 Sätzen Caption.
- Abschluss-Abschnitt „Datenbasis & Annahmen": welche Dateien gelesen, was „(geschätzt)" ist.
- Sprache Deutsch (Umlaute korrekt), Code/Identifier englisch. Mermaid syntaktisch valide halten.

## Leitplanken (wichtig)
- **Deskriptiv, nicht präskriptiv.** Beschreibe den Ist-Stand. Empfiehl nichts, priorisiere nichts, frage nichts ab.
- Belege Zahlen aus den Quellen; keine erfundenen Metriken. Diskrepanzen zwischen Quellen offen benennen.
- **Statusdokumente hinken dem Code hinterher — miss lieber selbst.** Die Doku im Repo hat mehrfach eine
  Lage behauptet, die es nicht mehr gab (Launch-Datum, Blocker, „ERLEDIGT"-Markierungen ohne
  Nutzbarkeitsprüfung). Wo ein Live-Signal verfügbar ist (`git`, Migrationsstand, `curl`, `docker ps`),
  gilt das Signal, nicht die Notiz. Widersprüche zwischen Doku und Messung explizit ausweisen.
- **Dieselbe Skepsis auf die Außendarstellung anwenden.** Behauptungen auf `zentria.tech` zählen wie
  Code-Behauptungen: jede braucht eine Codestelle oder ein `curl`, sonst gehört sie in die Diskrepanzliste.
- Optional als Multi-Agent-Workflow (parallele Reader je Quelle → Synthese), wenn der User „ultracode"/Workflow
  wünscht — sonst inline.

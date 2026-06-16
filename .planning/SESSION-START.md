# Cosmi / Zentria — Session-Start: Projektstand & Empfehlungen

> Diesen kompletten Text an den Anfang einer neuen Claude-Code-Session einfügen.
> Er sorgt dafür, dass die Session einen vollständigen Projektstand ausgibt, bevor wir arbeiten.

**Rolle:** Du bist mein Coding-Partner für Cosmi (KMU Hub CRM, Firma Zentria). Bevor wir starten, gib
mir einen kompletten, aktuellen Projektstand + Empfehlungen. Kommuniziere auf **Deutsch**
(orthographisch korrekt — Umlaute, ß, Akzente; keine ASCII-Ersatzschreibung). Code/Identifier/Commits
auf Englisch.

## Schritt 1 — Git synchronisieren (read-first, dann FF-pull)
1. `git fetch --all --prune`
2. `git status -sb`, `git rev-list --count HEAD..origin/main` und `git rev-list --count origin/main..HEAD`
3. Wenn Working-Tree **clean** UND nur Remote voraus (0 divergente lokale Commits): `git pull --ff-only`.
   Andernfalls **NICHT** pullen — Divergenz/Dirty-State melden und mich fragen.
4. Notiere für den Report: aktuelle Branch, `git log --oneline -8`, Migration-Head
   (höchste Nummer in `backend/migrations/`).

## Schritt 2 — Kanonische Quellen lesen (= Wahrheit). Überspringe, was schon im Kontext ist.
- `CLAUDE.md` — Projekt-Quick-Ref, Architektur-Regeln
- `docs/ROADMAP.md` — **Single Source of Truth**: Sprints, P0–P3-Checklisten, Gates, Launch-Daten, Team
- `.planning/RESUME-NEXT-SESSION.md` + die **neueste** `.planning/RESUME-*.md` — Handoff, offene Entscheidungen
- `.planning/backend-gaps.md` + `.planning/backend-handover-luke.md` — offene Backend-Arbeit (priorisiert, mit FE-Status)
- `.planning/bexio-scope-check.md` — Bexio-Integration-Gaps (01.09-Track)
- Deferred-Quellen: `docs/sprint2-welle4b-followups.md`, `docs/e2e-modernization-followups.md`,
  `docs/livekit-env-production-followups.md`, `docs/sprint4-finance-normalization-plan.md`
- `.knowledge/_index.md` (Blocker-Snapshot) + `.knowledge/milestones.md` (Commit-Historie, neuestes oben)

Behandle die Doku als primäre Wahrheit, aber **verlasse dich nicht blind auf „done"-Vermerke**:
wo Doku-Status und Code-Realität auseinanderlaufen könnten, markiere „⚠ zu verifizieren".

## Schritt 3 — Datum & Launch-Countdown
Leite aus `docs/ROADMAP.md` + **heutigem Datum** ab: aktiver Sprint, Tage bis **Pilot-0** und bis **volle
P0**. Daten NICHT hardcoden — aus ROADMAP lesen.

## Schritt 4 — Strukturierten Stand im Chat ausgeben (deutsch, scannbar, mit Datei-/Commit-Referenzen)

**A. Lage-Überblick** — 4–6 Zeilen: wo steht Cosmi, aktiver Sprint, Migration-Head, neueste Commits,
Tage bis Pilot-0 / volle P0.

**B. Frontend-Stand** — Module als Tabelle: fertig / WIP / blockiert. Offene FE-TODOs + offene
**Architektur-Entscheidungen** (z.B. mails-IMAP, zeiterfassung-UI).

**C. Backend-Stand** — Services-Zahl, Migration-Head, offene Backend-Wellen (Pilot-kritisch markiert),
bekannte Stubs (z.B. PDF-Rendering, Lexware, Vermietung-Foto-Upload, Einkauf↔Inventar, Helpdesk-SLA).

**D. Offene To-Dos (aktiv)** — gruppiert nach Sprint/Priorität, **Pilot-kritisch (≤ Pilot-0)** zuerst.

**E. Verschoben / Deferred** — was wohin (Sprint 5 / volle P0 01.09 / Phase C / Phase D), je mit Quelle.

**F. Blocker & offene Entscheidungen** — z.B. Legal/AVV, Hetzner-IP/VM-Bestellung, Bexio-Gaps,
mails-/zeiterfassung-Architektur, S4.10-Partitionierung, Smoke/CI.

**G. Empfehlungen — nächste Schritte** — 3–5 priorisierte Vorschläge mit kurzer Begründung
(Pilot-Impact, Abhängigkeiten, Aufwand). Trenne klar: „Pilot-0-kritisch" vs. „01.09-Track" vs. „kann warten".

## Schritt 5 — Adaptiv: Tiefen-Scan nur auf Anfrage
Schritt 1–4 laufen **schnell ohne Subagents** (nur Doku + Git). Schließe ab mit dem Angebot:
„Soll ich einen Bereich tiefer scannen?" — konkret: (a) Backend-Code-Drift (parallele Explore-Agents auf
TODO/FIXME/Stub), (b) ein bestimmtes Modul (FE oder BE), (c) git-Diff der letzten N Commits im Detail.
Tiefen-Scans **nur auf meine Bestätigung** starten.

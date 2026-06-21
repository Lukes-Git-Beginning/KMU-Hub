# RESUME — nächster Einstieg (Stand 2026-06-21 Session-Ende)

> **Direkt-Wiedereinstieg für morgen.** main = `0623a32b`, working tree clean, alles gepusht.
> SCHRITT 0 (beide Terminals): `git pull --rebase origin main`

## Was fertig ist
- **berichte R-3/R-4 (Batch B5):** Print/PDF (B5-1) + Scheduling (B5-2…B5-5) — gepusht, von Darien reviewt („geht in die richtige Richtung").
- **berichte R-5 Integration (Batch B6):** an Aufgabe/Kontakt anhängen, PDF→Dokumente, externer Share-Link + Verwaltung (B6-1…B6-5) + Kontakt-Verweis-Fix (`c940d762`) — gepusht, reviewt.
  - Damit ist **berichte Report-Authoring R-0…R-5 komplett**. Nur **R-6 (Datenquellen)** fehlt für volle Review-Reife.
- **wiki Batch 1 Tiefe (WT-1…WT-5, Sub-Terminal):** rekursive Kategorien + Verschieben, Tags/Titel-Rename, Versions-Diff, Anhänge/Share-Verwaltung, Suche-Fix — gepusht, von Darien reviewt.
  - **Dariens Hauptbefund:** „das Erstellen sieht kein bisschen so aus wie beim berichte-Modul" → wiki Batch 2 priorisiert.

## Nächster Batch (morgen) — 2 Terminals parallel (Vorschlag)
- **Sub-Terminal: wiki Batch 2 (Editor auf berichte-Niveau, WP-1…WP-5) — ⭐ PRIORITÄT (Dariens Hauptanliegen).**
  - Copy-paste-Paket: **`.planning/wiki-batch2-editor-SUB.md`** (fertig).
  - Detail-Plan: `.planning/wiki-knowledge-base-VISION.md` (Batch 2, geschärft).
- **Haupt-Terminal: berichte R-6 (Datenquellen-Ausbau).**
  - Plan/Sub-Block (jetzt fürs Haupt nutzbar): `.planning/RESUME-batch-4.md` (R-6-Abschnitt) + `.planning/berichte-report-authoring-VISION.md` (R-6).
  - Inhalt: 6 neue Quellen (HR, Zeiterfassung, Verträge, Einkauf, Fuhrpark, Rapporte) + 5 bestehende vertiefen; Merge-Punkt nur `registry.ts`.

**OFFENE ENTSCHEIDUNG (morgen früh, 1 Satz):** wiki Batch 2 ins Sub (parallel zu R-6 im Haupt) — ODER wiki Batch 2 ins Haupt (volle Aufmerksamkeit), R-6 danach. Darien entscheidet die Terminal-Zuordnung; Pakete liegen für beide Varianten bereit.

## Danach in der wiki-Pipeline
- Batch 3 (Vertrauen & Vernetzung: Lebenszyklus, Page-Owner/Review, Kommentare, Backlinks)
- Batch 4 (CRM-Integration + Lese-KI: Zusammenfassen, „Frag das Wiki")
- Alle in `.planning/wiki-knowledge-base-VISION.md`.

## Build-+-Verify-Standard (beide Terminals, pro Phase)
bauen → i18n ×4 (`{var}`, ICU-Plural) → gescopter Typecheck (foreground, echter Exit, NIE `| tail`) → Playwright-Screenshot-QA + PNGs WIRKLICH ansehen → ein Commit (explizite Pfade) → push (pull --rebase über parallele Pushes).
Latenz-Hinweis: vorbestehende tsc-Fehler in `useTasks.ts`/`crm.ts:396`/`email-client.ts` ignorieren (nicht unsere Dateien) — nur eigene Dateien müssen sauber sein.

## Review-Findings-Ablage
`.planning/berichte-r3r4-review-findings.md` (berichte) — falls morgen noch Punkte kommen.

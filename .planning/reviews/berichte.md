# Review — berichte

> **Status:** review-reif (B-1…B-5, Main-Terminal, direct-to-main).
> **Lane:** FE/UX-Review (mock-first). Echter No-Code-Query-Builder + echtes DATEV-/BI-Backend = Lukes Lane, **nicht** Teil dieses Reviews.
> **Screens:** `/#/berichte` — Tabs Dashboard · Erstellen · Geplant · DATEV.
> **Kontext:** Vor dem Batch waren **3 von 4 Tabs im Demo tot** (MSW-Handler fehlten → leere Dropdowns/Tabellen). Jetzt alle vier lebendig.

## Was gebaut wurde (Definition of Done)
- [x] **B-1 — MSW-Vollausbau**: `mocks/handlers/berichte.ts` ausgebaut — 6 System-Definitionen, `run` (Zeitreihen + Aggregate), `export` (echte PDF/CSV-Blobs via `mini-pdf`), `schedules`-CRUD + Toggle, **stateful**. Erstellen-/Geplant-/DATEV-Tab leben.
- [x] **B-2 — Dashboard-Tiefe**: Hero-Charts laden automatisch (kein manueller Klick), Ticket-Chart nutzt eigene Definition. KPI-Klick → **`shared/DetailModal`** (Mini-Zeitreihe + Kennzahlen-Tabelle, ganze Card klickbar, sticky Close). DATEV BWA↔SuSa-Toggle repariert (war ohne `onClick`).
- [x] **B-3 — Schedules + Alerts**: Toggle/Löschen/Anlegen wirken stateful. **„Nächster Lauf"-Spalte** aus Cron berechnet (z. B. nächster Montag / Monatserster / „Pausiert"). Alert-Schwellwert-Feld im Anlegen-Dialog + Bell-Indikator in der Liste. (Bonus: Epoch-Fallback „01.01.1970" → „Noch nicht gelaufen".)
- [x] **B-4 — Sortierung + Einstellungen**: `shared/SortMenu` in der Schedule-Liste (Name/Zeitplan/Status, Richtung). Modul-Einstellungs-Eintrag „Berichte" mit **personal** (Standard-Format + Zeitraum — Format wird im Builder real vorausgewählt) + **tenant** (erlaubte Export-Formate + zulässige E-Mail-Domains).
- [x] **B-5 — i18n + Schlusscheck**: alle defaultValue-Keys ×4 migriert (DE/EN/FR/IT), Hardcode-Placeholder i18n'd. EN-Sweep über alle Tabs sauber.

## Worauf besonders achten
- Alle 4 Tabs durchklicken — wirklich gefüllt (kein EmptyState)?
- Erstellen → Bericht wählen → „Bericht generieren" → lädt eine Datei (PDF/CSV)? Format-Toggle.
- KPI-Card klicken → Drilldown-Modal zentriert mit Chart + Kennzahlen, Close sticky.
- Geplant: Anlegen (mit Alert-Schwelle → Bell), Toggle, Löschen, „Nächster Lauf" plausibel; Sortierung Feld + Richtung.
- DATEV: BWA ↔ Summen & Salden umschalten → Tabelle wechselt.
- **EN umschalten** über alle Tabs + Einstellungs-Fenster — keine deutschen UI-Reste, keine Raw-Keys. (DACH-Demo-Daten wie Schedule-Namen/DATEV-Positionen bleiben bewusst deutsch.)

## Out of scope (kein Mangel)
- Echter No-Code-Query-Builder (P2 🔒), echtes DATEV-/BI-Backend (P4 🔒) — DATEV-Tabelle zeigt MSW-Demo-Daten.
- Tenant-Settings (Formate/Domains) sind demo-stateful, keine echte Backend-Durchsetzung.
- Verschiebbare Dashboard-Kacheln (DnD) — nicht in diesem Batch.

## Befunde
> Format pro Zeile: **Schweregrad** (P0 / P1 / P2) · **was** · **wo** (Datei/Screen) · **Repro**.

_(noch keine — von Nico zu füllen)_

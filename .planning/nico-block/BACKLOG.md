# Block-Backlog — Content & Self-Service (für Nico)

> Reihenfolge nach **Risiko aufsteigend** (erst sichere, gut gemusterte Aufgaben). Detailliert ausgearbeitet werden Specs **erst nach erfolgreichem Pilot** (Phase 01 + 02). Diese Liste ist die Landkarte, nicht die Endausbaustufe.
> Quelle: Ist-Stand-Analyse 2026-06-08. Viele Plan-Phasen sind schon **teilweise** gebaut — hier steht der echte Rest-Bedarf, nicht der nominale Plan.

## Risiko-Matrix (aus Ist-Stand)
| Modul | Risiko | Warum |
|---|---|---|
| **notifications** | 🟢 niedrig | Einziges mit vollem OpenAPI + MSW-Handler + Backend. Klar getrennte Komponenten. |
| **wiki** | 🟢🟡 niedrig–mittel | FE-komplett auf Zustand-Store. Solange Store-seitig bleiben → sicher. (Zwei Datenwege Store vs. Hooks — nicht mischen.) |
| **berichte** | 🟡 mittel | Klar komponentisiert, `useChartTheme`/`useReducedMotion` als Utilities da. Aber **kein MSW** → Daten ggf. lokal mocken. |
| **formulare** | 🟡 mittel | 2458-Zeilen-Monolith, Dual-State (Zustand-Draft + TanStack). Kein MSW. Klar gegliedert, aber groß. |

## Pilot (zuerst)
- [x] **Phase 01** — notifications: Ruhezeiten & DND-UI → `phase-01-notifications-quiet-hours.md` (Commit `279dee2`, Demo-Handler-Fix; UI war bereits da)
- [x] **Phase 02** — berichte: Sparkline in KPI-Karten → `phase-02-berichte-sparkline.md` (Commit `f3a30e2` + ASCII-Fix `7dcbb4b`; inkl. neuem berichte-MSW-Handler, da bisher keiner existierte)

→ **Review-Gate (Stand 2026-06-09):** Beide Piloten von Nico gebaut + selbst-verifiziert (gescopter tsc grün, QA-Script grün, Screenshots geprüft, 0 Raw-Keys/pageErrors). **Branch `marathon/nico` (von HEAD, trägt Phase-01-Fix), gepusht.** ⏳ **Wartet auf Darien-Review.** Grün → wiki-Block freigeben + nächste Specs schreiben.

## Nach dem Pilot — Kandidaten (grob, je 1 Spec wenn dran)

### notifications (🟢)
- Modul-Icon-Mapping erweitern (kennt nur chat/crm/hr/dialer → Rest fällt auf Megaphone-Fallback).
- `is_read`/`unread` Query-Mismatch fixen (Hook schickt `is_read`, Handler erwartet `unread`).
- Modul-Gruppierung der Liste im Center (nach Modul gebündelt + Zähler).

### wiki (🟢🟡) — Store-seitig bleiben
- Editor-Modus visuell sauber abgrenzen (Read↔Edit), Speichern/Abbrechen.
- Interne Verlinkung `[[Artikel]]` zwischen Wiki-Artikeln.
- @Mention im Editor (Muster aus Chat `MentionAutocomplete` wiederverwendbar).
- Anhänge-Liste am Artikel (UI, Store-seitig).

### berichte (🟡) — Daten ggf. lokal mocken
- Drilldown-Slot füllen (KPI-Klick → Detailansicht statt Platzhaltertext).
- Zusätzlicher Chart-Typ (Pie/Donut) mit eigenem Datensatz (zweites Chart nutzt aktuell dieselben Daten).
- Zeitraum-Filter auf das Dashboard (nicht nur im ReportBuilder).

### formulare (🟡) — größtes Modul, vorsichtig
- **DnD-Reordering** der Felder im Editor (`@dnd-kit` einbinden; `reorderFields` ist im Store vorbereitet).
- Export-Button-UI verdrahten (`useExportSubmissions` ist da, nur kein Button).
- DSGVO-Einwilligungs-Feldtyp ergänzen.

## Wichtige modulübergreifende Notizen für Nico
- **wiki, formulare, berichte fehlen im OpenAPI-Spec + haben keinen MSW-Handler.** Wenn ein Feature echte Server-Daten braucht und nichts kommt: entweder Store-seitig/lokal mocken **oder** in `.planning/backend-gaps.md` für Luke notieren. **Nicht** raten oder echte Endpunkte erfinden.
- **notifications** ist der sichere Hafen — dort ist alles end-to-end vorhanden.
- Backend-Bedarf, der dir auffällt, immer in `.planning/backend-gaps.md` sammeln (für Luke).

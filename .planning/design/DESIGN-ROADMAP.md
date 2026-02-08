# Design Roadmap: KMU Hub Desktop

## Overview

Design-Phasen fuer das KMU Hub Desktop Frontend. Getrennt von Lukes GSD-Phasen
(`.planning/ROADMAP.md`), um Merge-Konflikte zu vermeiden. Fertige Design-Arbeit
wird per Branch-Review an Luke uebergeben, der sie in offizielle GSD-Phasen integriert.

**Branch:** `design/brainstorm`
**Designer:** Darien
**Uebergabe an:** Luke (main branch, GSD-Phasen)

---

## Design-Phasen

- [x] **D1: Desk Foundation** — Schreibtisch-Metapher, Room-Scene Layout, Theme-System, Uhr-Deko, Maximize-Modus
- [ ] **D2: Visual Polish** — Texturen, Schatten, Tiefenwirkung, Farbverfeinerung, Dark-Mode-Feinschliff
- [ ] **D3: Theme Picker** — UI zum Wechseln von Desk-Themes, Theme-Vorschau, evtl. Custom-Theme-Editor
- [ ] **D4: Decorations System** — Pflanzen-SVGs, Bilderrahmen, Schreibtisch-Items, Drag-to-Place UI
- [ ] **D5: Module Styling** — Dashboard-Widgets, CRM-Listen/Details, Chat-UI, Notification-Center visuell aufwerten

---

## Abhaengigkeiten

```
D1 (done) → D2 (polish das Fundament)
                → D3 (braucht poliertes Theme-System)
                → D4 (braucht polierte Slots)
           → D5 (unabhaengig, kann parallel zu D2-D4)
```

## Uebergabe-Workflow

1. Darien plant + implementiert auf `design/brainstorm`
2. Phase abgeschlossen → SUMMARY.md schreiben
3. Branch pushen, Luke Bescheid geben
4. Luke reviewed, erstellt offizielle GSD-Phase auf `main`
5. Luke merged/adaptiert die Aenderungen

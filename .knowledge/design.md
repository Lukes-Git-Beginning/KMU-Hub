---
tags: [frontend, design, erledigt]
updated: 2026-03-04
---
# Design System

## Status: Integration erledigt (2026-02-27)
- D1-D8 cherry-picked from design/brainstorm to main (full design system)
- 5-layer desk theme system (5 themes: cozy, dreamy, raumstation, clean, minimal)
- SVG empty state illustrations + search dropdown conversion
- Darien's note: `.planning/for-darien-2026-02-27.md`

## D9 (Visual Polish + Accessibility)
- Geplant fuer Phase B (April 2026)

## Key Lessons
- Design files use `actionLabel`/`onAction` aber main's EmptyState uses `action:{label,onClick}`
- `git checkout origin/design/brainstorm -- <path>` safe fuer additive changes
- 12 industry-specific modules from design = mock only for v1

## Frontend Wiring Progress
- KontaktePage: DONE (2026-02-27, React Query)
- Next: FirmenPage + DealsPage (stores/crm.ts Mock entfernen)
- Phase A remaining: WorkPage, KalenderPage, FinanzenPage, Dashboard-Widgets
- 11 industry stores stay mock until v2 (Plugin roadmap)

## Verwandte Notes
- [[architektur]] — Frontend Stack
- [[milestones]] — Wiring Progress

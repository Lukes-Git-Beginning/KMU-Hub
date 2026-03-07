---
tags: [frontend, design, erledigt]
updated: 2026-03-07
---
# Design System

## Status: D9 gemerged (2026-03-07)
- D1-D8 cherry-picked from design/brainstorm to main (full design system)
- D9 Waves 15-20 gemerged (2026-03-07) — Visual Polish + UI system integration
- 5-layer desk theme system (5 themes: cozy, dreamy, raumstation, clean, minimal)
- SVG empty state illustrations + search dropdown conversion
- Darien's note: `.planning/for-darien-2026-02-27.md`
- **Lint-Status:** 0 ESLint-Probleme (347 gefixt am 2026-03-07)

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

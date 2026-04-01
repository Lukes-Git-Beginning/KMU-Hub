---
tags: [frontend, design, tools, skills, audit]
updated: 2026-03-26
---
# Design System

## Status: D9 gemerged + Design-Tooling + UI Hardening (2026-03-26)
- D1-D8 cherry-picked from design/brainstorm to main (full design system)
- D9 Waves 15-20 gemerged (2026-03-07) — Visual Polish + UI system integration
- 5-layer desk theme system (5 themes: cozy, dreamy, raumstation, clean, minimal)
- SVG empty state illustrations + search dropdown conversion
- **Lint-Status:** 0 ESLint-Probleme (347 gefixt am 2026-03-07)

## AI Design Skills (installiert 2026-03-26)

### frontend-design (Anthropic Official)
- Auto-loaded Design-Directives (~400 Tokens)
- Bannt generische Fonts (Inter, Roboto, Arial)
- Erzwingt BOLD aesthetic direction vor Code-Generierung

### impeccable (pbakaus, 13.8k Stars)
- 21 Skills installiert in `.agents/skills/`
- Key Commands: `/audit`, `/critique`, `/polish`, `/animate`, `/normalize`, `/bolder`, `/overdrive`, `/typeset`, `/colorize`, `/harden`, `/adapt`, `/distill`, `/extract`, `/delight`, `/clarify`, `/arrange`, `/optimize`, `/quieter`, `/onboard`, `/teach-impeccable`
- `/audit` — Qualitaets-Check mit P0-P3 Severity Levels
- `/critique` — UX-Review gegen Nielsen's 10 Heuristiken + Persona Testing
- `/polish` — Final Pass für pixelperfekte Perfektion

### Design-Workflow
Generate -> `/audit` -> `/critique` -> Fix -> `/polish` -> Ship

## Component Libraries (installiert 2026-03-26)

### Motion (ex Framer Motion)
- Package: `motion` — deklarative Animationen für React
- Import: `import { motion } from 'motion/react'`
- Ergaenzt unser CSS-Keyframe-System für komplexere Animationen (Spring Physics, Gestures, Layout Animations)

### Magic UI Komponenten (via shadcn Registry)
Installiert in `src/renderer/src/components/ui/`:

| Komponente | Datei | Use Case |
|-----------|-------|----------|
| **NumberTicker** | `number-ticker.tsx` | KPI-Cards mit animiertem Count-Up |
| **AnimatedList** | `animated-list.tsx` | Live Activity Feeds |
| **BentoGrid** | `bento-grid.tsx` | Dashboard Widget-Layouts |
| **Marquee** | `marquee.tsx` | Ticker-Bands, Partner-Logos |
| **AnimatedBeam** | `animated-beam.tsx` | Datenfluss-Visualisierung |
| **Dock** | `dock.tsx` | macOS-Style Quick-Access Navigation |
| **ShimmerButton** | `shimmer-button.tsx` | Premium CTAs mit Schimmer-Effekt |
| **AnimatedCircularProgressBar** | `animated-circular-progress-bar.tsx` | Pipeline-Completion, Quotas |
| **BorderBeam** | `border-beam.tsx` | Highlight aktiver/wichtiger Cards |
| **BlurFade** | `blur-fade.tsx` | Sanfte Content-Reveals |

Install-Pattern: `npx shadcn@latest add "https://magicui.design/r/<name>" --yes`

## Design-Directives (in CLAUDE.md dokumentiert)
- Aesthetic Direction: "Premium SaaS mit Editorial Touch"
- Font-Bans: Inter, Roboto, Arial, Space Grotesk, Helvetica, Open Sans
- Anti-Patterns: Card-in-Card, AI Slop, symmetrische Layouts, Über-Animation
- Farb-Hierarchie: Dominante Primaerfarbe + scharfe Akzente
- Motion: Orchestrierte Page-Loads > verstreute Micro-Interactions

## UI Hardening Pass (B8, 2026-03-26)

### Audit Score: 12/20 → 17.5/20 (6 Commits, ~210 Dateien)

**Accessibility:**
- 26 Custom-Modale auf Radix Dialog migriert (Focus-Trap, ESC, ARIA)
- div-onClick → semantic button konvertiert
- aria-labels auf allen Icon-Buttons
- Semantic HTML: aside, nav, main korrekt
- Skip-to-content Link in AppShell

**Performance:**
- MessageList virtualisiert (@tanstack/react-virtual)
- TipTap lazy-loaded (React.lazy, 7 Consumer)
- Query-Key-Stabilitaet (Primitives statt Objects)
- Images lazy-loaded (loading="lazy" decoding="async")
- Intl.NumberFormat gehoisted
- PresenceProvider userIds in useMemo

**Theming:**
- ~500 Hard-Coded Tailwind-Colors → Design-Tokens migriert
- 3 neue Token-Familien: success-foreground, warning-foreground, info-foreground (in allen 16 Theme-Bloecken)
- Roboto aus Font-Fallback entfernt → system-ui
- Verbleibend: ~59 Dateien in Settings/Integrations (niedrige Prioritaet)

**Anti-Patterns bereinigt:**
- Keine banned Fonts, kein Gradient-Text, kein Card-Nesting
- Glass selektiv (26 Dateien, Layout-fokussiert)
- ModulesGrid Hero-Card spannt 2 Spalten (Card-Monotonie gebrochen)
- sm: Breakpoint zu Grids hinzugefuegt

## Key Lessons
- Design files use `actionLabel`/`onAction` aber main's EmptyState uses `action:{label,onClick}`
- `git checkout origin/design/brainstorm -- <path>` safe für additive changes
- 12 industry-specific modules from design = mock only for v1

## Frontend Wiring Progress
- KontaktePage: DONE (2026-02-27, React Query)
- Next: FirmenPage + DealsPage (stores/crm.ts Mock entfernen)
- Phase A remaining: WorkPage, KalenderPage, FinanzenPage, Dashboard-Widgets
- 11 industry stores stay mock until v2 (Plugin roadmap)

## Weitere Design-Ressourcen (nicht installiert, bei Bedarf)
- **Aceternity UI** (ui.aceternity.com) — 200+ Wow-Factor Components (3D, Parallax, Spotlight). Copy-Paste, $199 Lifetime Premium.
- **UI UX Pro Max** — AI Design System Generator (67 Styles, 161 Paletten)
- **Figma MCP Server** — Bidirektionale Figma <-> Code Pipeline
- **Magic MCP (21st.dev)** — Component Generation im IDE
- **v0 by Vercel** — Prototyping mit shadcn Output

## Verwandte Notes
- [[architektur]] — Frontend Stack
- [[milestones]] — Wiring Progress

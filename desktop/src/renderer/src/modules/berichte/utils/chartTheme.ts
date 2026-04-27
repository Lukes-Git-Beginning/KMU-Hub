/**
 * ChartTheme — bridges Cosmi CSS theme variables into Recharts primitives.
 *
 * Rationale: the UI directive in CLAUDE.md forbids Recharts' rainbow defaults.
 * We resolve `--primary`, `--accent-1`, `--accent-2`, `--muted-foreground`,
 * and `--border` at render time so charts inherit the active theme (light,
 * dark, colour-blind) without a component-level theme prop.
 */
import { useMemo } from 'react'

export interface ChartTheme {
  primary: string
  accent1: string
  accent2: string
  muted: string
  grid: string
  gradientStops: { start: number; end: number }
}

const DEFAULT_THEME: ChartTheme = {
  primary: '#1e7e74',
  accent1: '#ec4899',
  accent2: '#14b8a6',
  muted: '#64748b',
  grid: 'rgba(100,116,139,.2)',
  gradientStops: { start: 0.8, end: 0.05 },
}

function readCssVar(root: HTMLElement, name: string, fallback: string): string {
  const raw = root.style.getPropertyValue(name) || getComputedStyle(root).getPropertyValue(name)
  const trimmed = raw.trim()
  return trimmed.length > 0 ? trimmed : fallback
}

export function useChartTheme(): ChartTheme {
  return useMemo<ChartTheme>(() => {
    if (typeof document === 'undefined') return DEFAULT_THEME
    const root = document.documentElement
    return {
      primary: readCssVar(root, '--primary', DEFAULT_THEME.primary),
      accent1: readCssVar(root, '--accent-1', DEFAULT_THEME.accent1),
      accent2: readCssVar(root, '--accent-2', DEFAULT_THEME.accent2),
      muted: readCssVar(root, '--muted-foreground', DEFAULT_THEME.muted),
      grid: readCssVar(root, '--border', DEFAULT_THEME.grid),
      gradientStops: DEFAULT_THEME.gradientStops,
    }
  }, [])
}

/** Palette for categorical series — primary dominant, accents secondary. */
export function categoricalPalette(theme: ChartTheme): string[] {
  return [theme.primary, theme.accent1, theme.accent2, theme.muted]
}

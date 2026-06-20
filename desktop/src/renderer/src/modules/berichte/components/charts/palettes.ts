import type { ChartTheme } from '../../utils/chartTheme'
import { categoricalPalette } from '../../utils/chartTheme'

/** Named per-report palettes. 'default' follows the active Cosmi theme. */
export type PaletteId = 'default' | 'ocean' | 'sunset' | 'forest' | 'mono'

export const PALETTE_OPTIONS: { id: PaletteId; label: string }[] = [
  { id: 'default', label: 'Standard' },
  { id: 'ocean', label: 'Ozean' },
  { id: 'sunset', label: 'Sonnenuntergang' },
  { id: 'forest', label: 'Wald' },
  { id: 'mono', label: 'Einfarbig' },
]

const NAMED: Record<Exclude<PaletteId, 'default' | 'mono'>, string[]> = {
  ocean: ['#0e7490', '#0ea5e9', '#6366f1', '#22d3ee', '#1e40af'],
  sunset: ['#ea580c', '#f59e0b', '#e11d48', '#fb7185', '#b45309'],
  forest: ['#15803d', '#65a30d', '#0d9488', '#84cc16', '#166534'],
}

/** Resolve a palette to a colour array, cycling to cover `n` series.
 *  Unknown/legacy ids (e.g. older docs saved with a theme name) fall back to the
 *  active theme palette so a stale value never breaks rendering. */
export function resolvePalette(theme: ChartTheme, id: PaletteId | undefined, n: number): string[] {
  let base: string[]
  if (id === 'mono') {
    base = [theme.primary, `${theme.primary}cc`, `${theme.primary}99`, `${theme.primary}66`, `${theme.primary}40`]
  } else if (id && id in NAMED) {
    base = NAMED[id as keyof typeof NAMED]
  } else {
    base = categoricalPalette(theme)
  }
  return Array.from({ length: Math.max(n, 1) }, (_, i) => base[i % base.length])
}

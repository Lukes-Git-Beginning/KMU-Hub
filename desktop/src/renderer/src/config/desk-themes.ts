/**
 * Desk theme registry.
 *
 * Adding a new theme is purely data-driven: define a DeskTheme
 * object and add it to the DESK_THEMES record. No component
 * changes needed.
 */
import type { DeskTheme } from '@/types/desk-theme'

/** Classic office theme — warm oak desk with subtle wood grain. */
const classicOffice: DeskTheme = {
  id: 'classic-office',
  name: 'Klassisches Buero',
  description: 'Warmer Eichenholz-Schreibtisch mit klassischem Ambiente',
  frame: {
    top: 24,
    right: 20,
    bottom: 28,
    left: 0, // Sidebar acts as the left edge
    windowRadius: 8,
  },
  cssVariables: {
    'desk-frame-bg': 'hsl(30 25% 78%)',
    'desk-frame-texture':
      'repeating-linear-gradient(105deg, transparent, transparent 20px, hsla(30,15%,70%,0.08) 20px, hsla(30,15%,70%,0.08) 21px)',
    'desk-frame-border': 'hsl(30 20% 70%)',
    'desk-frame-shadow': 'inset 0 0 20px rgba(0,0,0,0.06)',
    'desk-window-radius': '8px',
    'desk-window-shadow': '0 2px 20px rgba(0,0,0,0.12)',
    'desk-sidebar-bg': 'hsl(30 15% 96%)',
    'desk-sidebar-border': 'hsl(30 15% 88%)',
  },
  cssVariablesDark: {
    'desk-frame-bg': 'hsl(25 20% 22%)',
    'desk-frame-texture':
      'repeating-linear-gradient(105deg, transparent, transparent 20px, hsla(25,15%,18%,0.12) 20px, hsla(25,15%,18%,0.12) 21px)',
    'desk-frame-border': 'hsl(25 15% 30%)',
    'desk-frame-shadow': 'inset 0 0 20px rgba(0,0,0,0.15)',
    'desk-window-shadow': '0 2px 20px rgba(0,0,0,0.3)',
    'desk-sidebar-bg': 'hsl(222.2 84% 6%)',
    'desk-sidebar-border': 'hsl(217.2 32.6% 17.5%)',
  },
  sidebar: {
    background: 'solid',
    integratedWithFrame: true,
  },
  decorationSlots: [
    {
      id: 'top-right-clock',
      edge: 'top',
      position: { x: '88%', y: '50%' },
      maxSize: { width: 20, height: 20 },
      acceptTypes: ['clock'],
    },
    {
      id: 'bottom-left-plant',
      edge: 'bottom',
      position: { x: '5%', y: '30%' },
      maxSize: { width: 40, height: 24 },
      acceptTypes: ['plant'],
    },
    {
      id: 'right-top-photo',
      edge: 'right',
      position: { x: '50%', y: '15%' },
      maxSize: { width: 16, height: 20 },
      acceptTypes: ['photo'],
    },
  ],
}

/** Registry of all available desk themes, keyed by theme ID. */
export const DESK_THEMES: Record<string, DeskTheme> = {
  'classic-office': classicOffice,
}

export const DEFAULT_DESK_THEME_ID = 'classic-office'

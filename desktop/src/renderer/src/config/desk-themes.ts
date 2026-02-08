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
    top: 16,
    right: 180,     // Right decoration wall
    bottom: 120,    // Desk surface
    left: 180,      // Left decoration wall
    windowRadius: 12,
  },
  cssVariables: {
    // Room wall (background behind everything)
    'desk-wall-bg': 'hsl(35 20% 88%)',
    'desk-wall-texture': 'none',
    // Desk surface (bottom strip)
    'desk-surface-bg': 'hsl(30 25% 65%)',
    'desk-surface-texture':
      'repeating-linear-gradient(90deg, transparent, transparent 40px, hsla(30,15%,55%,0.1) 40px, hsla(30,15%,55%,0.1) 41px)',
    'desk-surface-border': 'hsl(30 20% 55%)',
    // Decoration panels (left/right walls)
    'desk-deco-bg': 'hsl(35 18% 85%)',
    // Window (work area)
    'desk-window-radius': '12px',
    'desk-window-shadow': '0 4px 30px rgba(0,0,0,0.18)',
    'desk-window-border': 'hsl(210 10% 70%)',
    // Sidebar
    'desk-sidebar-bg': 'hsl(30 15% 96%)',
    'desk-sidebar-border': 'hsl(30 15% 88%)',
    // Transition
    'desk-transition-duration': '300ms',
    'desk-transition-easing': 'cubic-bezier(0.4, 0, 0.2, 1)',
  },
  cssVariablesDark: {
    'desk-wall-bg': 'hsl(220 15% 12%)',
    'desk-surface-bg': 'hsl(25 20% 18%)',
    'desk-surface-texture':
      'repeating-linear-gradient(90deg, transparent, transparent 40px, hsla(25,15%,14%,0.15) 40px, hsla(25,15%,14%,0.15) 41px)',
    'desk-surface-border': 'hsl(25 15% 25%)',
    'desk-deco-bg': 'hsl(220 12% 14%)',
    'desk-window-shadow': '0 4px 30px rgba(0,0,0,0.4)',
    'desk-window-border': 'hsl(220 10% 25%)',
    'desk-sidebar-bg': 'hsl(222.2 84% 6%)',
    'desk-sidebar-border': 'hsl(217.2 32.6% 17.5%)',
  },
  sidebar: {
    background: 'solid',
    integratedWithFrame: true,
  },
  decorationSlots: [
    {
      id: 'left-wall-clock',
      edge: 'left',
      position: { x: '50%', y: '15%' },
      maxSize: { width: 80, height: 80 },
      acceptTypes: ['clock'],
    },
    {
      id: 'left-wall-photo',
      edge: 'left',
      position: { x: '50%', y: '50%' },
      maxSize: { width: 120, height: 100 },
      acceptTypes: ['photo'],
    },
    {
      id: 'right-wall-plant',
      edge: 'right',
      position: { x: '50%', y: '70%' },
      maxSize: { width: 120, height: 120 },
      acceptTypes: ['plant'],
    },
    {
      id: 'right-wall-photo',
      edge: 'right',
      position: { x: '50%', y: '25%' },
      maxSize: { width: 120, height: 100 },
      acceptTypes: ['photo'],
    },
    {
      id: 'desk-surface-stationery',
      edge: 'bottom',
      position: { x: '30%', y: '50%' },
      maxSize: { width: 150, height: 80 },
      acceptTypes: ['stationery'],
    },
  ],
}

/** Registry of all available desk themes, keyed by theme ID. */
export const DESK_THEMES: Record<string, DeskTheme> = {
  'classic-office': classicOffice,
}

export const DEFAULT_DESK_THEME_ID = 'classic-office'

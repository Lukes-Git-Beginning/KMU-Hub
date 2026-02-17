/**
 * Desk theme registry.
 *
 * 5 themes: Cozy (default), Dreamy, Raumstation, Clean, Minimal.
 * Adding a new theme is purely data-driven: define a DeskTheme
 * object and add it to the DESK_THEMES record.
 *
 * Each theme defines all 5 layers of the desk system:
 *   L1 - Room Scene (undefined for now — CSS gradient fallback)
 *   L2 - Furniture Overlays (empty for now)
 *   L3+L4 - Mount Points (where decorations can be placed)
 *   L5 - UI Skin (CSS variable overrides for the app chrome)
 */
import type { DeskTheme, MountPoint } from '@/types/desk-theme'

// ── Room Scene Imports ──────────────────────────────────────────────────────
import cozyRoomLight from '@/../assets/desk/cozy/room-scene-light-v7.png'
import dreamyRoomLight from '@/../assets/desk/dreamy/room-scene-light.png'
import raumstationRoomLight from '@/../assets/desk/raumstation/b1939468-3d5d-4503-a128-f9a8f7e60e12.png'
import cleanRoomLight from '@/../assets/desk/clean/room-scene-light.png'

// ── SHARED MOUNT POINTS ────────────────────────────────────────────────────
//
// Standard 9-point layout used by most themes.
// Left wall (3), right wall (3), desk surface (3).
// Positions are percentage-based relative to the full room container
// and must NOT overlap with the window area.

const STANDARD_MOUNT_POINTS: MountPoint[] = [
  // Left wall — x: 3-10%, outside the window's left edge
  {
    id: 'left-wall-clock',
    surface: 'left-wall',
    position: { x: '5%', y: '12%' },
    maxSize: { width: 80, height: 80 },
    acceptTypes: ['clock'],
  },
  {
    id: 'left-wall-calendar',
    surface: 'left-wall',
    position: { x: '6%', y: '32%' },
    maxSize: { width: 60, height: 80 },
    acceptTypes: ['calendar'],
  },
  {
    id: 'left-wall-deco',
    surface: 'left-wall',
    position: { x: '5%', y: '55%' },
    maxSize: { width: 100, height: 100 },
    acceptTypes: ['image', 'photo', 'plant'],
  },

  // Right wall — x: 90-97%, outside the window's right edge
  {
    id: 'right-wall-deco1',
    surface: 'right-wall',
    position: { x: '92%', y: '15%' },
    maxSize: { width: 100, height: 100 },
    acceptTypes: ['image', 'photo'],
  },
  {
    id: 'right-wall-deco2',
    surface: 'right-wall',
    position: { x: '93%', y: '42%' },
    maxSize: { width: 120, height: 120 },
    acceptTypes: ['image', 'plant'],
  },
  {
    id: 'right-wall-deco3',
    surface: 'right-wall',
    position: { x: '91%', y: '68%' },
    maxSize: { width: 100, height: 100 },
    acceptTypes: ['image', 'stationery'],
  },

  // Desk surface — y: 82-95%, below the window
  {
    id: 'desk-surface-left',
    surface: 'desk-surface',
    position: { x: '18%', y: '85%' },
    maxSize: { width: 90, height: 80 },
    acceptTypes: ['image', 'stationery', 'plant'],
  },
  {
    id: 'desk-surface-center',
    surface: 'desk-surface',
    position: { x: '50%', y: '88%' },
    maxSize: { width: 80, height: 70 },
    acceptTypes: ['stationery', 'custom'],
  },
  {
    id: 'desk-surface-right',
    surface: 'desk-surface',
    position: { x: '82%', y: '85%' },
    maxSize: { width: 90, height: 80 },
    acceptTypes: ['image', 'stationery', 'plant'],
  },
]

// ── Shared transition values ───────────────────────────────────────────────

const TRANSITION_DURATION = '300ms'
const TRANSITION_EASING = 'cubic-bezier(0.4, 0, 0.2, 1)'

// ══════════════════════════════════════════════════════════════════════════════
// THEME DEFINITIONS
// ══════════════════════════════════════════════════════════════════════════════

// ── 1. COZY (Default) ──────────────────────────────────────────────────────

const cozy: DeskTheme = {
  id: 'cozy',
  name: 'Gemuetlich',
  description: 'Warmes Home-Office mit Eichenholz und Sonnenlicht',
  isMinimal: false,

  window: {
    top: '10%',
    right: '13%',
    bottom: '27%',
    left: '13%',
    borderRadius: '12px',
    innerPadding: '12px',
  },

  roomScene: { light: cozyRoomLight, dark: cozyRoomLight },
  furniture: [],
  mountPoints: STANDARD_MOUNT_POINTS,

  roomVariables: {
    'desk-room-bg':
      'linear-gradient(180deg, hsl(35 25% 82%) 0%, hsl(35 20% 75%) 70%, hsl(30 30% 62%) 70%, hsl(30 28% 58%) 100%)',
    'desk-window-radius': '12px',
    'desk-window-shadow': '0 4px 30px rgba(0,0,0,0.18)',
    'desk-window-border': 'hsl(30 15% 75%)',
    'desk-transition-duration': TRANSITION_DURATION,
    'desk-transition-easing': TRANSITION_EASING,
  },
  roomVariablesDark: {
    'desk-room-bg':
      'linear-gradient(180deg, hsl(25 15% 12%) 0%, hsl(25 12% 10%) 70%, hsl(25 20% 18%) 70%, hsl(25 18% 15%) 100%)',
    'desk-window-shadow': '0 4px 30px rgba(0,0,0,0.4)',
    'desk-window-border': 'hsl(25 10% 25%)',
  },

  uiSkin: {
    variables: {
      'skin-card-bg': 'hsl(35 30% 97%)',
      'skin-card-border': 'hsl(35 20% 88%)',
      'skin-card-shadow': '0 1px 3px rgba(0,0,0,0.06)',
      'skin-radius': '10px',
      'skin-sidebar-bg': 'hsl(30 15% 96%)',
      'skin-sidebar-border': 'hsl(30 15% 88%)',
    },
    variablesDark: {
      'skin-card-bg': 'hsl(25 12% 14%)',
      'skin-card-border': 'hsl(25 10% 22%)',
      'skin-card-shadow': '0 1px 3px rgba(0,0,0,0.2)',
      'skin-sidebar-bg': 'hsl(222.2 84% 6%)',
      'skin-sidebar-border': 'hsl(217.2 32.6% 17.5%)',
    },
  },

  sidebar: { background: 'solid', integratedWithFrame: true },
  viewDescription: 'Sonniger Garten mit Blumen und sanften Huegeln',
}

// ── 2. DREAMY ──────────────────────────────────────────────────────────────

const dreamy: DeskTheme = {
  id: 'dreamy',
  name: 'Verträumt',
  description: 'Magische Atmosphäre mit Kristallen und Lichtpartikeln',
  isMinimal: false,

  window: {
    top: '10%',
    right: '13%',
    bottom: '27%',
    left: '13%',
    borderRadius: '16px',
    innerPadding: '12px',
  },

  roomScene: { light: dreamyRoomLight, dark: dreamyRoomLight },
  furniture: [],
  mountPoints: STANDARD_MOUNT_POINTS,

  roomVariables: {
    'desk-room-bg':
      'linear-gradient(180deg, hsl(270 20% 88%) 0%, hsl(270 18% 80%) 70%, hsl(270 15% 72%) 70%, hsl(270 14% 68%) 100%)',
    'desk-window-radius': '16px',
    'desk-window-shadow': '0 4px 30px rgba(120,80,200,0.15)',
    'desk-window-border': 'hsl(270 15% 78%)',
    'desk-transition-duration': TRANSITION_DURATION,
    'desk-transition-easing': TRANSITION_EASING,
  },
  roomVariablesDark: {
    'desk-room-bg':
      'linear-gradient(180deg, hsl(270 20% 10%) 0%, hsl(270 18% 8%) 70%, hsl(270 15% 15%) 70%, hsl(270 14% 12%) 100%)',
    'desk-window-shadow': '0 4px 30px rgba(120,80,200,0.25)',
    'desk-window-border': 'hsl(270 15% 22%)',
  },

  uiSkin: {
    variables: {
      'skin-card-bg': 'hsl(270 25% 97%)',
      'skin-card-border': 'hsl(270 15% 88%)',
      'skin-card-shadow': '0 1px 4px rgba(120,80,200,0.08)',
      'skin-radius': '14px',
      'skin-focus-ring': '0 0 0 3px rgba(160,120,220,0.3)',
      'skin-sidebar-bg': 'hsla(270 20% 96% / 0.85)',
      'skin-sidebar-border': 'hsl(270 15% 88%)',
    },
    variablesDark: {
      'skin-card-bg': 'hsl(270 18% 12%)',
      'skin-card-border': 'hsl(270 14% 20%)',
      'skin-card-shadow': '0 1px 4px rgba(120,80,200,0.15)',
      'skin-focus-ring': '0 0 0 3px rgba(160,120,220,0.25)',
      'skin-sidebar-bg': 'hsla(270 30% 8% / 0.9)',
      'skin-sidebar-border': 'hsl(270 20% 18%)',
    },
  },

  sidebar: { background: 'solid', integratedWithFrame: true },
  viewDescription: 'Magische Landschaft mit schwebenden Inseln und Kristallen',
}

// ── 3. RAUMSTATION ─────────────────────────────────────────────────────────

/** Sci-fi station: fewer mount points for a cleaner, futuristic feel. */
const RAUMSTATION_MOUNT_POINTS: MountPoint[] = [
  // Left wall (2)
  {
    id: 'left-wall-panel1',
    surface: 'left-wall',
    position: { x: '4%', y: '18%' },
    maxSize: { width: 80, height: 80 },
    acceptTypes: ['clock', 'image'],
  },
  {
    id: 'left-wall-panel2',
    surface: 'left-wall',
    position: { x: '5%', y: '48%' },
    maxSize: { width: 90, height: 90 },
    acceptTypes: ['image', 'custom'],
  },

  // Right wall (2)
  {
    id: 'right-wall-panel1',
    surface: 'right-wall',
    position: { x: '93%', y: '20%' },
    maxSize: { width: 80, height: 80 },
    acceptTypes: ['image', 'photo'],
  },
  {
    id: 'right-wall-panel2',
    surface: 'right-wall',
    position: { x: '92%', y: '52%' },
    maxSize: { width: 90, height: 90 },
    acceptTypes: ['image', 'custom'],
  },

  // Desk surface (3)
  {
    id: 'desk-surface-left',
    surface: 'desk-surface',
    position: { x: '20%', y: '84%' },
    maxSize: { width: 80, height: 70 },
    acceptTypes: ['stationery', 'custom'],
  },
  {
    id: 'desk-surface-center',
    surface: 'desk-surface',
    position: { x: '50%', y: '86%' },
    maxSize: { width: 70, height: 60 },
    acceptTypes: ['stationery', 'custom'],
  },
  {
    id: 'desk-surface-right',
    surface: 'desk-surface',
    position: { x: '80%', y: '84%' },
    maxSize: { width: 80, height: 70 },
    acceptTypes: ['stationery', 'custom'],
  },
]

const raumstation: DeskTheme = {
  id: 'raumstation',
  name: 'Raumstation',
  description: 'Futuristische Station mit Panorama-Aussicht ins All',
  isMinimal: false,

  window: {
    top: '14.9%',
    right: '13%',
    bottom: '25.6%',
    left: '13%',
    borderRadius: '8px',
    innerPadding: '12px',
  },

  roomScene: { light: raumstationRoomLight, dark: raumstationRoomLight },
  furniture: [],
  mountPoints: RAUMSTATION_MOUNT_POINTS,

  roomVariables: {
    'desk-room-bg':
      'linear-gradient(180deg, hsl(220 15% 22%) 0%, hsl(220 12% 18%) 72%, hsl(220 10% 25%) 72%, hsl(220 8% 22%) 100%)',
    'desk-window-radius': '8px',
    'desk-window-shadow': '0 0 24px rgba(80,140,255,0.12), 0 4px 30px rgba(0,0,0,0.2)',
    'desk-window-border': 'hsl(220 20% 40%)',
    'desk-transition-duration': TRANSITION_DURATION,
    'desk-transition-easing': TRANSITION_EASING,
  },
  roomVariablesDark: {
    'desk-room-bg':
      'linear-gradient(180deg, hsl(220 15% 6%) 0%, hsl(220 12% 4%) 72%, hsl(220 10% 10%) 72%, hsl(220 8% 8%) 100%)',
    'desk-window-shadow': '0 0 24px rgba(80,140,255,0.08), 0 4px 30px rgba(0,0,0,0.5)',
    'desk-window-border': 'hsl(220 15% 22%)',
  },

  uiSkin: {
    variables: {
      'skin-card-bg': 'hsl(220 12% 96%)',
      'skin-card-border': 'hsl(220 15% 85%)',
      'skin-card-shadow': '0 1px 3px rgba(0,0,0,0.06)',
      'skin-radius': '4px',
      'skin-border-accent': 'hsl(220 60% 60%)',
      'skin-sidebar-bg': 'hsl(220 10% 96%)',
      'skin-sidebar-border': 'hsl(220 12% 85%)',
    },
    variablesDark: {
      'skin-card-bg': 'hsl(220 12% 10%)',
      'skin-card-border': 'hsl(220 10% 18%)',
      'skin-card-shadow': '0 1px 3px rgba(0,0,0,0.3)',
      'skin-border-accent': 'hsl(220 50% 45%)',
      'skin-sidebar-bg': 'hsl(220 12% 6%)',
      'skin-sidebar-border': 'hsl(220 10% 15%)',
    },
  },

  sidebar: { background: 'solid', integratedWithFrame: true },
  viewDescription: 'Weltraum-Panorama mit Sternen, Nebeln und fernen Planeten',
}

// ── 4. CLEAN (Neutral base) ────────────────────────────────────────────────

const clean: DeskTheme = {
  id: 'clean',
  name: 'Neutral',
  description: 'Neutrale Clean-Base — weisse Waende, Holz-Desk, Landschaftsblick',
  isMinimal: false,

  window: {
    top: '10%',
    right: '13%',
    bottom: '27%',
    left: '13%',
    borderRadius: '10px',
    innerPadding: '12px',
  },

  roomScene: { light: cleanRoomLight, dark: cleanRoomLight },
  furniture: [],
  mountPoints: STANDARD_MOUNT_POINTS,

  roomVariables: {
    'desk-room-bg':
      'linear-gradient(180deg, hsl(0 0% 90%) 0%, hsl(0 0% 85%) 70%, hsl(30 15% 65%) 70%, hsl(30 12% 60%) 100%)',
    'desk-window-radius': '10px',
    'desk-window-shadow': '0 4px 30px rgba(0,0,0,0.12)',
    'desk-window-border': 'hsl(0 0% 80%)',
    'desk-transition-duration': TRANSITION_DURATION,
    'desk-transition-easing': TRANSITION_EASING,
  },
  roomVariablesDark: {
    'desk-room-bg':
      'linear-gradient(180deg, hsl(0 0% 10%) 0%, hsl(0 0% 8%) 70%, hsl(30 10% 14%) 70%, hsl(30 8% 12%) 100%)',
    'desk-window-shadow': '0 4px 30px rgba(0,0,0,0.35)',
    'desk-window-border': 'hsl(0 0% 22%)',
  },

  uiSkin: {
    variables: {
      'skin-card-bg': 'hsl(0 0% 98%)',
      'skin-card-border': 'hsl(0 0% 88%)',
      'skin-card-shadow': '0 1px 3px rgba(0,0,0,0.05)',
      'skin-radius': '10px',
      'skin-sidebar-bg': 'hsl(0 0% 96%)',
      'skin-sidebar-border': 'hsl(0 0% 88%)',
    },
    variablesDark: {
      'skin-card-bg': 'hsl(0 0% 12%)',
      'skin-card-border': 'hsl(0 0% 20%)',
      'skin-card-shadow': '0 1px 3px rgba(0,0,0,0.2)',
      'skin-sidebar-bg': 'hsl(0 0% 8%)',
      'skin-sidebar-border': 'hsl(0 0% 16%)',
    },
  },

  sidebar: { background: 'solid', integratedWithFrame: true },
  viewDescription: 'Huegel-Landschaft mit See und blauem Himmel',
}

// ── 5. MINIMAL ─────────────────────────────────────────────────────────────

const minimal: DeskTheme = {
  id: 'minimal',
  name: 'Minimalistisch',
  description: 'Ablenkungsfreier Arbeitsplatz ohne Dekoration',
  isMinimal: true,

  window: {
    top: '0',
    right: '0',
    bottom: '0',
    left: '0',
    borderRadius: '0',
    innerPadding: '8px',
  },

  roomScene: undefined,
  furniture: [],
  mountPoints: [],

  roomVariables: {
    'desk-room-bg': 'hsl(0 0% 93%)',
    'desk-window-radius': '0',
    'desk-window-shadow': 'none',
    'desk-window-border': 'transparent',
    'desk-transition-duration': TRANSITION_DURATION,
    'desk-transition-easing': TRANSITION_EASING,
  },
  roomVariablesDark: {
    'desk-room-bg': 'hsl(0 0% 8%)',
    'desk-window-shadow': 'none',
    'desk-window-border': 'transparent',
  },

  uiSkin: {
    variables: {},
    variablesDark: {},
  },

  sidebar: { background: 'solid', integratedWithFrame: false },
  viewDescription: '',
}

// ══════════════════════════════════════════════════════════════════════════════
// REGISTRY & EXPORTS
// ══════════════════════════════════════════════════════════════════════════════

export const DESK_THEMES: Record<string, DeskTheme> = {
  cozy,
  dreamy,
  raumstation,
  clean,
  minimal,
}

/** Display order in the theme picker UI. */
export const DESK_THEME_ORDER = [
  'cozy',
  'dreamy',
  'raumstation',
  'clean',
  'minimal',
] as const

export const DEFAULT_DESK_THEME_ID = 'cozy'

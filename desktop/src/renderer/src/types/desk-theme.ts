/**
 * Type definitions for the desk theme / workspace layer system.
 *
 * 5-Layer architecture:
 *   L1 – Room Scene (painted room with window + view)
 *   L2 – Furniture Overlays (desk, shelves — transparent PNGs)
 *   L3 – Placeable Decorations (freigestellt objects on mount points)
 *   L4 – Mount Points (defined positions — data only, not rendered)
 *   L5 – UI Skin (theme-dependent CSS variable overrides for the app)
 */

// ---------------------------------------------------------------------------
// Layer 1: Room Scene
// ---------------------------------------------------------------------------

/** A single cohesive room background image (light + dark variants). */
export interface RoomScene {
  light: string
  dark: string
}

// ---------------------------------------------------------------------------
// Layer 2: Furniture Overlays
// ---------------------------------------------------------------------------

/** A transparent PNG overlay positioned absolutely within the room. */
export interface FurnitureOverlay {
  id: string
  image: { light: string; dark: string }
  /** CSS positioning relative to the room container */
  position: {
    top?: string
    right?: string
    bottom?: string
    left?: string
    width?: string
    height?: string
  }
  /** Stacking order within the furniture layer */
  zIndex: number
}

// ---------------------------------------------------------------------------
// Layer 3 + 4: Decorations & Mount Points
// ---------------------------------------------------------------------------

/** Types of decorative items that can be placed. */
export type DecorationItemType =
  | 'clock'
  | 'calendar'
  | 'plant'
  | 'photo'
  | 'stationery'
  | 'custom'
  | 'image'

/** Which surface a mount point sits on. */
export type MountSurface =
  | 'left-wall'
  | 'right-wall'
  | 'desk-surface'
  | 'window-sill'
  | 'shelf'

/** A defined position where a decoration can be placed. */
export interface MountPoint {
  id: string
  /** Which physical surface this point belongs to */
  surface: MountSurface
  /** CSS position (percentage-based, relative to room container) */
  position: { x: string; y: string }
  /** Maximum dimensions for items at this point */
  maxSize: { width: number; height: number }
  /** What types of decorations are allowed here */
  acceptTypes: DecorationItemType[]
}

/** A decoration item placed by the user at a specific mount point. */
export interface DecorationPlacement {
  /** Mount point ID this decoration occupies */
  slotId: string
  type: DecorationItemType
  /** Specific variant (e.g., 'succulent' for plants, 'analog' for clock) */
  variant: string
  /** Image URL for image-type decorations */
  imageUrl?: string
  /** CSS animation class for image decorations */
  animationClass?: string
  /** Optional data (photo URL, custom settings, etc.) */
  data?: Record<string, string>
}

// ---------------------------------------------------------------------------
// Layer 5: UI Skin
// ---------------------------------------------------------------------------

/** Theme-dependent visual overrides for the functional app UI. */
export interface UISkin {
  /** CSS variable overrides applied to the work area (light mode) */
  variables: Record<string, string>
  /** CSS variable overrides for dark mode */
  variablesDark?: Record<string, string>
  /** Optional CSS class added to the work area root */
  className?: string
}

// ---------------------------------------------------------------------------
// Theme Definition
// ---------------------------------------------------------------------------

/** Complete definition of a desk theme with all 5 layers. */
export interface DeskTheme {
  id: string
  /** Display name shown in theme picker */
  name: string
  /** Short description */
  description: string

  /** If true, skip room scene entirely (no frame, no decorations) */
  isMinimal: boolean

  /** How the UI "window" is positioned within the room scene (percentages) */
  window: {
    top: string
    right: string
    bottom: string
    left: string
    borderRadius: string
    /** Gap between window frame edge and UI edge */
    innerPadding: string
  }

  /** L1: Room scene images (undefined = CSS gradient fallback) */
  roomScene?: RoomScene

  /** L2: Furniture overlay images */
  furniture: FurnitureOverlay[]

  /** L4: Where decorations can be placed */
  mountPoints: MountPoint[]

  /** L5: UI skin CSS variable overrides */
  uiSkin: UISkin

  /** Room-level CSS variables (fallback colors, transitions, etc.) */
  roomVariables: Record<string, string>
  roomVariablesDark?: Record<string, string>

  /** Sidebar visual config */
  sidebar: {
    background: 'solid' | 'transparent'
    integratedWithFrame: boolean
  }

  /** Description of what's visible through the window (for Art Director) */
  viewDescription: string
}

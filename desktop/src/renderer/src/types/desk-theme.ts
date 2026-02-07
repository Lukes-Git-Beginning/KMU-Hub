/**
 * Type definitions for the desk theme system.
 *
 * Desk themes define the visual "room" environment that wraps
 * around the main work area, including frame dimensions, colors,
 * and decoration placement slots.
 */

/** Defines the visual properties of a single desk theme. */
export interface DeskTheme {
  /** Unique identifier (kebab-case) */
  id: string
  /** Display name shown in theme picker */
  name: string
  /** Short description */
  description: string

  /** Frame edge dimensions in pixels */
  frame: {
    top: number
    right: number
    bottom: number
    /** Left edge width (0 when sidebar is integrated with frame) */
    left: number
    /** Border radius on the work area "window" */
    windowRadius: number
  }

  /** CSS custom property overrides applied when this theme is active.
   *  Keys are variable names WITHOUT the -- prefix. */
  cssVariables: Record<string, string>

  /** Optional dark mode CSS variable overrides */
  cssVariablesDark?: Record<string, string>

  /** Sidebar visual integration */
  sidebar: {
    background: 'solid' | 'transparent'
    /** Whether sidebar visually merges with the left desk frame edge */
    integratedWithFrame: boolean
  }

  /** Available decoration slots for this theme */
  decorationSlots: DecorationSlot[]
}

/** A named position in the desk frame where a decoration can be placed. */
export interface DecorationSlot {
  id: string
  /** Which frame edge this slot belongs to */
  edge: 'top' | 'right' | 'bottom' | 'left'
  /** CSS positioning within that edge (percentage-based) */
  position: { x: string; y: string }
  /** Maximum dimensions for items in this slot */
  maxSize: { width: number; height: number }
  /** What types of decorations can go here */
  acceptTypes: DecorationItemType[]
}

/** Types of decorative items that can be placed on the desk. */
export type DecorationItemType = 'clock' | 'plant' | 'photo' | 'stationery' | 'custom'

/** A decoration item placed by the user in a specific slot. */
export interface DecorationPlacement {
  slotId: string
  type: DecorationItemType
  /** Specific variant (e.g., 'succulent' for plants, 'analog' for clock) */
  variant: string
  /** Optional data (photo URL, custom settings, etc.) */
  data?: Record<string, string>
}

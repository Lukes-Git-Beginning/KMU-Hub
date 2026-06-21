/**
 * Block registry — the extension point of the shared document engine.
 *
 * A module declares one BlockTypeDef per block type (icon, label, default
 * factory, edit component, read-view component) and the engine drives the rest:
 * the "+Block" insert menu, drag-reorder, inline editing and the reader. Adding a
 * block type to a module = add one definition here; improving a block = edit its
 * components once and every surface that registered it benefits.
 */
import type { ComponentType } from 'react'
import type { LucideIcon } from 'lucide-react'
import type { DocBlockBase } from './types'

export interface BlockEditProps<B extends DocBlockBase> {
  block: B
  onPatch: (patch: Partial<B>) => void
}

export interface BlockViewProps<B extends DocBlockBase> {
  block: B
  /** Accent colour from document settings (optional). */
  accent?: string
  /** Chart palette id from document settings (optional). */
  palette?: string
}

/** Insert-menu grouping. */
export type BlockGroup = 'content' | 'data' | 'layout'

export interface BlockTypeDef<B extends DocBlockBase = DocBlockBase> {
  type: B['type']
  icon: LucideIcon
  /** i18n key for the menu label. */
  labelKey: string
  /** Insert-menu group; defaults to 'content'. */
  group?: BlockGroup
  /** Offered in the "+Block" insert menu. Default true. */
  insertable?: boolean
  /** Atomic blocks never break across a page in print mode (chart/table/kpi/image). */
  atomic?: boolean
  /** Factory for a fresh block of this type. */
  makeDefault: () => B
  Edit: ComponentType<BlockEditProps<B>>
  View: ComponentType<BlockViewProps<B>>
}

/** A module's block catalogue, keyed by block type. */
export type BlockRegistry = Record<string, BlockTypeDef>

/**
 * Author a typed block definition, erased to the registry's base type. Keeps full
 * type-safety at the call site (Edit/View are checked against the concrete block)
 * while letting the engine treat all entries uniformly.
 */
export function defineBlock<B extends DocBlockBase>(def: BlockTypeDef<B>): BlockTypeDef {
  return def as unknown as BlockTypeDef
}

/** Build a registry (keyed map) from an ordered list of definitions. */
export function buildRegistry(defs: BlockTypeDef[]): BlockRegistry {
  const reg: BlockRegistry = {}
  for (const def of defs) reg[def.type] = def
  return reg
}

/** Resolve a block's definition, or undefined for an unknown type. */
export function resolveBlockDef(registry: BlockRegistry, type: string): BlockTypeDef | undefined {
  return registry[type]
}

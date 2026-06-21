/**
 * Generic block-document model — the shared substrate behind every "author a
 * document" surface in Cosmi (berichte reports, wiki articles, and whatever
 * comes next). A document is rows → columns → blocks. One full-width column per
 * row is the default top-down reading flow; a row may split into 2–3 columns for
 * KPI rows and "text beside chart" layouts.
 *
 * The model is type-agnostic: each module supplies its own block union (each
 * member extending DocBlockBase) plus a BlockRegistry that maps every block type
 * to its icon, default factory, edit component and read-view component. New block
 * types — and improvements to existing ones — live in the registry, never in the
 * engine. That is the whole point: build a block once, reuse it everywhere, keep
 * extending it.
 */

/** Every block carries a stable id and a discriminating type tag. */
export interface DocBlockBase {
  id: string
  type: string
}

export interface DocColumn<B extends DocBlockBase = DocBlockBase> {
  id: string
  /** Relative width weight (1/1 = 50/50, 3/2 = 60/40). Default 1. */
  width?: number
  blocks: B[]
}

export interface DocRow<B extends DocBlockBase = DocBlockBase> {
  id: string
  /** 1–3 columns. One column = full width. */
  columns: DocColumn<B>[]
}

/** Stable unique block/row/column id helper. */
export function docUid(prefix: string): string {
  return `${prefix}-${crypto.randomUUID().slice(0, 8)}`
}

/** A fresh, empty full-width column. */
export function emptyDocColumn(): DocColumn {
  return { id: docUid('col'), blocks: [], width: 1 }
}

/** True when no column in any row holds a block. */
export function isDocEmpty(rows: DocRow[]): boolean {
  return rows.every((row) => row.columns.every((col) => col.blocks.length === 0))
}

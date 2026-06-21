/**
 * Read-view helpers — reading time, the table-of-contents outline (derived from
 * the block document's heading blocks), and the category breadcrumb path.
 */
import type { DocRow } from '@/components/shared/document'
import type { WikiCategory } from '@/types/wiki'

export interface TocHeading {
  id: string
  text: string
  level: number
}

/** Stable anchor id a wiki heading block renders, shared with the TOC. */
export function headingAnchorId(blockId: string): string {
  return `wh-${blockId}`
}

/** Rough reading time in minutes from the article HTML (~200 wpm). */
export function readingTimeMinutes(html: string): number {
  const text = html.replace(/<[^>]*>/g, ' ')
  const words = text.trim().split(/\s+/).filter(Boolean).length
  return Math.max(1, Math.round(words / 200))
}

/**
 * Build the table-of-contents outline from the block document's heading blocks.
 * Each heading block renders a stable anchor (see {@link headingAnchorId}), so
 * the TOC links + scroll-spy target the rendered sections without parsing HTML.
 */
export function tocFromRows(rows: DocRow[]): TocHeading[] {
  const headings: TocHeading[] = []
  for (const row of rows) {
    for (const col of row.columns) {
      for (const block of col.blocks) {
        if (block.type !== 'heading') continue
        const b = block as unknown as { id: string; text?: string; level?: number }
        const text = (b.text ?? '').trim()
        if (!text) continue
        headings.push({ id: headingAnchorId(b.id), text, level: b.level === 1 ? 1 : 2 })
      }
    }
  }
  return headings
}

/** Root → … → article category, for the breadcrumb trail (parentId chain). */
export function categoryPath(
  categoryId: string | undefined,
  categories: WikiCategory[],
): WikiCategory[] {
  if (!categoryId) return []
  const byId = new Map(categories.map((c) => [c.id, c]))
  const path: WikiCategory[] = []
  let cur = byId.get(categoryId)
  let guard = 0
  while (cur && guard++ < 12) {
    path.unshift(cur)
    cur = cur.parentId ? byId.get(cur.parentId) : undefined
  }
  return path
}

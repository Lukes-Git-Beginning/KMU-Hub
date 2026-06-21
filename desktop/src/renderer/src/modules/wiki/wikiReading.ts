/**
 * Read-view helpers (WP-4) — reading time, heading anchors for the TOC, and the
 * category breadcrumb path.
 */
import type { WikiCategory } from '@/types/wiki'

export interface TocHeading {
  id: string
  text: string
  level: number
}

/** Rough reading time in minutes from the article HTML (~200 wpm). */
export function readingTimeMinutes(html: string): number {
  const text = html.replace(/<[^>]*>/g, ' ')
  const words = text.trim().split(/\s+/).filter(Boolean).length
  return Math.max(1, Math.round(words / 200))
}

function slugify(s: string): string {
  const base = s
    .toLowerCase()
    .trim()
    .replace(/[^\w\sÀ-ɏ-]/g, '')
    .replace(/\s+/g, '-')
    .slice(0, 60)
  return base || 'section'
}

/**
 * Add stable anchor ids to H1–H3 in the article HTML and return the heading
 * outline for the table of contents. Ids are slugged from the heading text and
 * de-duplicated, so TOC links and scroll-spy can target each section.
 */
export function addHeadingIds(html: string): { html: string; headings: TocHeading[] } {
  if (!html) return { html: '', headings: [] }
  const doc = new DOMParser().parseFromString(html, 'text/html')
  const headings: TocHeading[] = []
  const seen = new Set<string>()
  doc.querySelectorAll('h1, h2, h3').forEach((h, i) => {
    const text = (h.textContent ?? '').trim()
    if (!text) return
    let id = slugify(text)
    if (seen.has(id)) id = `${id}-${i}`
    seen.add(id)
    h.setAttribute('id', id)
    headings.push({ id, text, level: Number(h.tagName[1]) })
  })
  return { html: doc.body.innerHTML, headings }
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

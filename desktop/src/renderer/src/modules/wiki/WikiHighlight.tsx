import { useMemo } from 'react'

// ---------------------------------------------------------------------------
// Highlight helper — wraps query matches in a <mark> for the search result list
// ---------------------------------------------------------------------------

function escapeRegExp(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

interface WikiHighlightProps {
  text: string
  query: string
}

/** Renders `text` with every (case-insensitive) occurrence of `query` highlighted. */
export function WikiHighlight({ text, query }: WikiHighlightProps) {
  const q = query.trim()
  const parts = useMemo(() => {
    if (!q) return [text]
    return text.split(new RegExp(`(${escapeRegExp(q)})`, 'ig'))
  }, [text, q])

  const lower = q.toLowerCase()
  if (!lower) return <>{text}</>

  return (
    <>
      {parts.map((part, i) =>
        part.toLowerCase() === lower ? (
          <mark key={i} className="rounded-sm bg-warning-light px-0.5 text-warning">
            {part}
          </mark>
        ) : (
          <span key={i}>{part}</span>
        ),
      )}
    </>
  )
}

/**
 * Returns a short content excerpt centred on the first query match, or null if
 * the query is empty or only appears in the title. Strips HTML to plain text.
 */
export function searchSnippet(html: string, query: string, radius = 42): string | null {
  const q = query.trim().toLowerCase()
  if (!q) return null
  const text = html.replace(/<[^>]*>/g, ' ').replace(/\s+/g, ' ').trim()
  const idx = text.toLowerCase().indexOf(q)
  if (idx === -1) return null
  const start = Math.max(0, idx - radius)
  const end = Math.min(text.length, idx + q.length + radius)
  return (start > 0 ? '… ' : '') + text.slice(start, end).trim() + (end < text.length ? ' …' : '')
}

/**
 * Lightweight text diff for the wiki version preview.
 *
 * Reduces two HTML snapshots to plain text lines (block-level segmentation),
 * then runs a classic LCS line diff so the version panel can show what was
 * added/removed before a restore. Intentionally simple — no word-level diff.
 */

export interface DiffRow {
  type: 'same' | 'added' | 'removed'
  text: string
}

/** Convert an HTML fragment into trimmed, block-segmented text lines. */
export function htmlToLines(html: string): string[] {
  if (!html) return []
  const withBreaks = html
    .replace(/<\/(p|div|h[1-6]|li|blockquote|tr|pre)>/gi, '\n')
    .replace(/<br\s*\/?>/gi, '\n')
  const text = withBreaks.replace(/<[^>]*>/g, '')
  const decoded = text
    .replace(/&nbsp;/g, ' ')
    .replace(/&amp;/g, '&')
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'")
  return decoded
    .split('\n')
    .map((l) => l.replace(/\s+/g, ' ').trim())
    .filter(Boolean)
}

/** LCS line diff between two text-line arrays (old → new). */
export function diffLines(oldLines: string[], newLines: string[]): DiffRow[] {
  const n = oldLines.length
  const m = newLines.length
  const dp: number[][] = Array.from({ length: n + 1 }, () => new Array<number>(m + 1).fill(0))
  for (let i = n - 1; i >= 0; i--) {
    for (let j = m - 1; j >= 0; j--) {
      dp[i][j] =
        oldLines[i] === newLines[j]
          ? dp[i + 1][j + 1] + 1
          : Math.max(dp[i + 1][j], dp[i][j + 1])
    }
  }
  const rows: DiffRow[] = []
  let i = 0
  let j = 0
  while (i < n && j < m) {
    if (oldLines[i] === newLines[j]) {
      rows.push({ type: 'same', text: oldLines[i] })
      i++
      j++
    } else if (dp[i + 1][j] >= dp[i][j + 1]) {
      rows.push({ type: 'removed', text: oldLines[i] })
      i++
    } else {
      rows.push({ type: 'added', text: newLines[j] })
      j++
    }
  }
  while (i < n) rows.push({ type: 'removed', text: oldLines[i++] })
  while (j < m) rows.push({ type: 'added', text: newLines[j++] })
  return rows
}

/** Convenience: diff two HTML snapshots directly. */
export function diffHtml(oldHtml: string, newHtml: string): DiffRow[] {
  return diffLines(htmlToLines(oldHtml), htmlToLines(newHtml))
}

/**
 * Text bridge for block conversion. Converting between prose blocks (text ↔
 * heading ↔ quote ↔ callout ↔ bullet) carries the content across by reducing it
 * to plain text and re-wrapping it for the target shape.
 */

/** Strip HTML to its plain text (DOMParser when available, regex fallback). */
export function htmlToText(html: string): string {
  if (!html) return ''
  if (typeof window !== 'undefined' && 'DOMParser' in window) {
    const doc = new DOMParser().parseFromString(html, 'text/html')
    return (doc.body.textContent ?? '').trim()
  }
  return html
    .replace(/<[^>]+>/g, '')
    .replace(/\s+/g, ' ')
    .trim()
}

/** Wrap plain text as a single rich-text paragraph (escaped, newlines → <br>). */
export function textToHtml(text: string): string {
  const trimmed = text.trim()
  if (!trimmed) return ''
  const escaped = trimmed
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/\n/g, '<br>')
  return `<p>${escaped}</p>`
}

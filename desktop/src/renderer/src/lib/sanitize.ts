import DOMPurify from 'dompurify'

/**
 * URI regexp that allows only http, https, mailto, and tel.
 * Blocks javascript: and data: URIs.
 */
const SAFE_URI_REGEXP =
  /^(?:(?:https?|mailto|tel):|[^a-z]|[a-z+.-]+(?:[^a-z+.:-]|$))/i

/**
 * Creates a fresh DOMPurify instance and configures it.
 * We use a factory pattern to avoid polluting the global DOMPurify state
 * with hooks (hooks are global per instance in DOMPurify v3).
 */
function createPurify(): DOMPurify.DOMPurifyI {
  const purify = DOMPurify(window)

  purify.addHook('afterSanitizeAttributes', (node) => {
    if (node.tagName === 'A') {
      node.setAttribute('target', '_blank')
      node.setAttribute('rel', 'noopener noreferrer')
    }
  })

  return purify
}

const purify = createPurify()

const DEFAULT_ALLOWED_TAGS = [
  'p', 'br', 'strong', 'em', 'b', 'i', 'u', 'a',
  'ul', 'ol', 'li',
  'h1', 'h2', 'h3', 'h4', 'h5', 'h6',
  'code', 'pre', 'blockquote', 'hr',
  'table', 'thead', 'tbody', 'tr', 'th', 'td',
  'img', 'span', 'div',
]

const DEFAULT_ALLOWED_ATTR = [
  'href', 'target', 'rel', 'src', 'alt', 'title', 'class',
]

const DEFAULT_FORBID_TAGS = ['script', 'iframe', 'object', 'embed', 'form', 'style']

const DEFAULT_FORBID_ATTR = [
  'onerror', 'onload', 'onclick', 'onmouseover', 'onfocus', 'onblur',
]

/**
 * Sanitize arbitrary HTML for use in dangerouslySetInnerHTML.
 * Suitable for: Mail body, Wiki content, Email template preview, IT-Admin HTML preview.
 *
 * - Allows a broad set of structural tags and attributes.
 * - Blocks script, iframe, form, style and all on* event handlers.
 * - Restricts URIs to https, http, mailto, tel (blocks javascript: and data:).
 * - Forces target="_blank" rel="noopener noreferrer" on all <a> elements.
 */
export function sanitizeHtml(raw: string, config?: DOMPurify.Config): string {
  return purify.sanitize(raw, {
    ALLOWED_TAGS: DEFAULT_ALLOWED_TAGS,
    ALLOWED_ATTR: DEFAULT_ALLOWED_ATTR,
    FORBID_TAGS: DEFAULT_FORBID_TAGS,
    FORBID_ATTR: DEFAULT_FORBID_ATTR,
    ALLOWED_URI_REGEXP: SAFE_URI_REGEXP,
    ...config,
  })
}

const STRICT_ALLOWED_TAGS = ['p', 'br', 'strong', 'em', 'b', 'i', 'u', 'span']

/**
 * Strict sanitizer for text-only formatting (no links, no images, no tables).
 * Suitable for: Signature preview.
 */
export function sanitizeHtmlStrict(raw: string): string {
  return purify.sanitize(raw, {
    ALLOWED_TAGS: STRICT_ALLOWED_TAGS,
    ALLOWED_ATTR: [],
    FORBID_TAGS: DEFAULT_FORBID_TAGS,
    FORBID_ATTR: DEFAULT_FORBID_ATTR,
    ALLOWED_URI_REGEXP: SAFE_URI_REGEXP,
  })
}

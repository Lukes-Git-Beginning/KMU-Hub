import { describe, it, expect } from 'vitest'
import { sanitizeHtml, sanitizeHtmlStrict } from '../sanitize'

describe('sanitizeHtml', () => {
  it('removes <script> tags and alert() calls', () => {
    const result = sanitizeHtml('<script>alert(1)</script>foo')
    expect(result).not.toContain('<script>')
    expect(result).not.toContain('alert')
    expect(result).toContain('foo')
  })

  it('removes <iframe> tags', () => {
    const result = sanitizeHtml('<iframe src="evil"></iframe>')
    expect(result).not.toContain('<iframe')
  })

  it('removes onclick from <a> but keeps the element', () => {
    const result = sanitizeHtml('<a href="https://x" onclick="evil()">link</a>')
    expect(result).not.toContain('onclick')
    expect(result).toContain('<a')
    expect(result).toContain('link')
  })

  it('strips javascript: href from <a>', () => {
    const result = sanitizeHtml('<a href="javascript:alert(1)">x</a>')
    expect(result).not.toContain('javascript:')
  })

  it('removes onerror from <img>', () => {
    const result = sanitizeHtml('<img src="x" onerror="alert(1)">')
    expect(result).not.toContain('onerror')
  })

  it('preserves allowed tags like <b>', () => {
    const result = sanitizeHtml('<b>bold</b>')
    expect(result).toContain('<b>bold</b>')
  })

  it('adds target=_blank and rel=noopener noreferrer to safe links', () => {
    const result = sanitizeHtml('<a href="https://example.com">link</a>')
    expect(result).toContain('target="_blank"')
    expect(result).toContain('rel="noopener noreferrer"')
  })

  it('handles empty string input', () => {
    expect(sanitizeHtml('')).toBe('')
  })
})

describe('sanitizeHtmlStrict', () => {
  it('removes <a> tags (links not allowed in strict mode)', () => {
    const result = sanitizeHtmlStrict('<a href="https://x">link</a>')
    expect(result).not.toContain('<a')
  })

  it('preserves <b> tags (text formatting allowed in strict mode)', () => {
    const result = sanitizeHtmlStrict('<b>bold</b>')
    expect(result).toContain('<b>bold</b>')
  })
})

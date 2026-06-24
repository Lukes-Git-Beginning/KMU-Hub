import { describe, it, expect } from 'vitest'
import { decodeWikiContent, encodeWikiContent } from '../wiki-content'

describe('wiki content codec', () => {
  it('round-trips a TipTap object through base64 (incl. umlauts/ß)', () => {
    const doc = {
      type: 'doc',
      content: [{ type: 'paragraph', content: [{ type: 'text', text: 'Größe & Übergänge — fünf Straßen' }] }],
    }
    const encoded = encodeWikiContent(doc)
    expect(typeof encoded).toBe('string')
    expect(decodeWikiContent(encoded)).toEqual(doc)
  })

  it('decodes a base64 string produced by the Go backend (UTF-8 JSON bytes)', () => {
    // base64 of `{"type":"doc","content":[]}` (what response.JSON emits for bytes)
    const wire = Buffer.from('{"type":"doc","content":[]}', 'utf-8').toString('base64')
    expect(decodeWikiContent(wire)).toEqual({ type: 'doc', content: [] })
  })

  it('tolerates an already-decoded object on read (MSW mock path)', () => {
    const obj = { type: 'doc', content: [] }
    expect(decodeWikiContent(obj)).toBe(obj)
  })

  it('tolerates an already-encoded string on write (idempotent encode)', () => {
    const wire = encodeWikiContent({ type: 'doc' })
    expect(encodeWikiContent(wire)).toBe(wire)
  })

  it('falls back to an empty doc for null/malformed content', () => {
    expect(decodeWikiContent(null)).toEqual({})
    expect(decodeWikiContent(undefined)).toEqual({})
    expect(decodeWikiContent('not-base64-json!!')).toEqual({})
  })
})

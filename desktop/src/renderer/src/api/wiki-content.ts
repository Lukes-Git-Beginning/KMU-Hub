/**
 * Wiki article/version `content` (a TipTap JSONB document) is serialised by the
 * gRPC service as protobuf `bytes`, i.e. a base64-encoded string on the wire.
 * The frontend and the MSW mock model it as a TipTap object
 * (Record<string, unknown>). These helpers translate at the client/mock
 * boundary so components always work with the object form while the wire stays
 * byte-accurate against the real backend.
 *
 * UTF-8 safe: article text routinely contains umlauts/ß, so plain btoa/atob
 * (Latin1-only) would corrupt content — we round-trip through
 * TextEncoder/TextDecoder, which also mirrors the Go side's []byte JSON bytes.
 */
import type { TipTapContent } from './wiki-types'

function utf8ToBase64(s: string): string {
  const bytes = new TextEncoder().encode(s)
  let bin = ''
  for (let i = 0; i < bytes.length; i++) bin += String.fromCharCode(bytes[i])
  return btoa(bin)
}

function base64ToUtf8(b64: string): string {
  const bin = atob(b64)
  const bytes = new Uint8Array(bin.length)
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i)
  return new TextDecoder().decode(bytes)
}

/**
 * Wire (base64 string) → TipTap object. Tolerates an already-decoded object
 * (MSW mock path) and falls back to an empty doc on malformed input.
 */
export function decodeWikiContent(content: unknown): TipTapContent {
  if (content == null) return {}
  if (typeof content !== 'string') return content as TipTapContent
  try {
    return JSON.parse(base64ToUtf8(content)) as TipTapContent
  } catch {
    return {}
  }
}

/**
 * TipTap object → wire (base64 string). Tolerates an already-encoded string so
 * it is safe to apply more than once.
 */
export function encodeWikiContent(content: unknown): string {
  if (typeof content === 'string') return content
  return utf8ToBase64(JSON.stringify(content ?? {}))
}

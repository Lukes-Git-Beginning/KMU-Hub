/**
 * Session-Cache für hochgeladene Belege (finanzen P2).
 *
 * Da es noch keinen Upload-Endpoint gibt (swap-ready, siehe backend-gaps.md),
 * persistiert der Mock nur den Dateinamen. Damit ein gerade hochgeladener Beleg
 * trotzdem in der Vorschau erscheint, halten wir die Datei als Blob-URL im
 * Speicher — keyed by Expense-ID. CSP erlaubt blob: für img/frame (NICHT data:).
 * Seed-Belege haben keinen Blob → die Vorschau zeigt einen Platzhalter.
 *
 * Beim Anbinden des echten Backends entfällt dieser Cache; die Vorschau lädt
 * dann die Beleg-URL vom Server.
 */
interface CachedReceipt {
  url: string
  name: string
  mimeType: string
}

const cache = new Map<string, CachedReceipt>()

export function putReceipt(id: string, file: File): void {
  const existing = cache.get(id)
  if (existing) URL.revokeObjectURL(existing.url)
  cache.set(id, { url: URL.createObjectURL(file), name: file.name, mimeType: file.type })
}

export function getReceipt(id: string): CachedReceipt | undefined {
  return cache.get(id)
}

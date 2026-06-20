/**
 * Deterministic helpers for generating demo rows in report sources.
 *
 * All randomness is seeded so preview charts and QA screenshots stay stable
 * across reloads. Date anchors use the current date so demo data looks recent,
 * but the *shape* (values, distribution) is fully deterministic per row index.
 */

/** Mulberry32 — tiny deterministic PRNG. Returns a function yielding [0,1). */
export function seededRng(seed: number): () => number {
  let a = seed >>> 0
  return () => {
    a |= 0
    a = (a + 0x6d2b79f5) | 0
    let t = Math.imul(a ^ (a >>> 15), 1 | a)
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }
}

/** Pick an element deterministically. */
export function pick<T>(arr: readonly T[], r: number): T {
  return arr[Math.floor(r * arr.length) % arr.length]
}

/** Integer in [min, max]. */
export function intBetween(r: number, min: number, max: number): number {
  return Math.floor(min + r * (max - min + 1))
}

/** Rounded float in [min, max] with given decimals. */
export function floatBetween(r: number, min: number, max: number, decimals = 2): number {
  const v = min + r * (max - min)
  const p = Math.pow(10, decimals)
  return Math.round(v * p) / p
}

/** ISO date (yyyy-mm-dd) `daysAgo` days before today. */
export function isoDaysAgo(daysAgo: number): string {
  const d = new Date()
  d.setHours(0, 0, 0, 0)
  d.setDate(d.getDate() - daysAgo)
  return d.toISOString().slice(0, 10)
}

/**
 * Generate `count` rows by calling `make(rng, index)` for each. The rng is
 * re-seeded per row from the base seed + index so a row's values never depend
 * on iteration order.
 */
export function generateRows(
  seed: number,
  count: number,
  make: (r: () => number, index: number) => Record<string, unknown>,
): Record<string, unknown>[] {
  const rows: Record<string, unknown>[] = []
  for (let i = 0; i < count; i++) {
    rows.push(make(seededRng(seed + i * 2654435761), i))
  }
  return rows
}

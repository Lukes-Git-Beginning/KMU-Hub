/**
 * Mandanten-Segmente A/B/C — rule-based client segmentation by revenue potential.
 *
 * Potential = sum of a contact's deal values. Thresholds are tenant-configurable
 * (CRM settings, "Für alle"). A = high value, B = medium, C = low.
 */

export type MandantSegment = 'A' | 'B' | 'C'

export interface SegmentThresholds {
  /** Minimum revenue potential (€) for segment A. */
  a: number
  /** Minimum revenue potential (€) for segment B (below a). */
  b: number
}

export const DEFAULT_SEGMENT_THRESHOLDS: SegmentThresholds = { a: 50000, b: 10000 }

export function computeSegment(revenuePotential: number, th: SegmentThresholds): MandantSegment {
  if (revenuePotential >= th.a) return 'A'
  if (revenuePotential >= th.b) return 'B'
  return 'C'
}

export const SEGMENT_META: Record<MandantSegment, { color: string; labelKey: string }> = {
  A: { color: '#16a34a', labelKey: 'crm.segment.a' },
  B: { color: '#ca8a04', labelKey: 'crm.segment.b' },
  C: { color: '#94a3b8', labelKey: 'crm.segment.c' },
}

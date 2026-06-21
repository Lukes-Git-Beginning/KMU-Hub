/**
 * Report-sources registry — the single shared merge point.
 *
 * Each finished module contributes one `<module>.source.ts`. To add a new
 * module's reporting, import it below and append it to REPORT_SOURCES. Parallel
 * work stays conflict-free: only this one line is shared.
 *
 * Mirrors the flat-array pattern of `modules/settings/module-settings-registry`.
 */
import type { ReportSource } from './types'
import { finanzenSource } from './finanzen.source'
import { kontakteSource } from './kontakte.source'
import { workSource } from './work.source'
import { helpdeskSource } from './helpdesk.source'
import { kommunikationSource } from './kommunikation.source'
import { hrSource } from './hr.source'
import { zeiterfassungSource } from './zeiterfassung.source'
import { vertraegeSource } from './vertraege.source'
import { einkaufSource } from './einkauf.source'

export const REPORT_SOURCES: ReportSource[] = [
  finanzenSource,
  kontakteSource,
  workSource,
  helpdeskSource,
  kommunikationSource,
  hrSource,
  zeiterfassungSource,
  vertraegeSource,
  einkaufSource,
]

/** Resolve a source by its registry id. */
export function resolveSource(id: string): ReportSource | undefined {
  return REPORT_SOURCES.find((s) => s.id === id)
}

export * from './types'

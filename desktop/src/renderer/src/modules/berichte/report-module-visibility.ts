/**
 * ReportModule → RBAC level-1 visibility (R-3 batch 5).
 *
 * Dashboard KPIs, hero charts and pinned builder reports carry a source
 * module; a role that cannot see a module must not see its numbers surfaced
 * through berichte either (the batch-2 mini-chart lesson: revenue leaked to
 * extern through a "module-free" widget). `cross` is deliberately visible to
 * anyone holding `berichte:reports:read`; UNKNOWN module ids fail closed so a
 * new source without a mapping surfaces in QA instead of leaking.
 */
import { moduleViewKey, type ModuleKey } from '@/config/capabilities'

const REPORT_MODULE_KEY: Record<string, ModuleKey | null> = {
  finanzen: 'finance',
  crm: 'crm',
  helpdesk: 'helpdesk',
  inventar: 'inventar',
  produktion: 'produktion',
  work: 'work',
  kommunikation: 'kommunikation',
  hr: 'team',
  zeiterfassung: 'zeiterfassung',
  vertraege: 'vertraege',
  einkauf: 'einkauf',
  fuhrpark: 'fuhrpark',
  rapporte: 'rapporte',
  cross: null,
}

/** While permissions load (`ready === false`) everything stays visible to
 *  avoid flicker — the surrounding tab gating already denies pessimistically. */
export function isReportModuleVisible(
  has: (key: string) => boolean,
  ready: boolean,
  module: string,
): boolean {
  if (!ready) return true
  const key = REPORT_MODULE_KEY[module]
  if (key === null) return true
  if (key === undefined) return false
  return has(moduleViewKey(key))
}

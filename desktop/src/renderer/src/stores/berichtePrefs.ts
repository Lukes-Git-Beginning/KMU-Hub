import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { ReportFormat } from '@/api/berichte-types'

/**
 * Berichte (Reports/BI) settings.
 * - personal: default export format + period for new reports (applies for real
 *   in the ReportBuilder + schedule dialog).
 * - tenant (demo, persisted locally — no real backend): allowed export formats
 *   and permitted recipient e-mail domains for scheduled reports.
 */
export type BerichtePeriod = 'last30' | 'thisMonth' | 'thisQuarter' | 'thisYear'

interface BerichtePrefsState {
  defaultFormat: ReportFormat
  defaultPeriod: BerichtePeriod
  /** Default colour palette for new builder reports (palette id). */
  defaultPalette: string
  allowedFormats: ReportFormat[]
  scheduleDomains: string[]
  setDefaultFormat: (f: ReportFormat) => void
  setDefaultPeriod: (p: BerichtePeriod) => void
  setDefaultPalette: (p: string) => void
  toggleAllowedFormat: (f: ReportFormat) => void
  addScheduleDomain: (d: string) => void
  removeScheduleDomain: (d: string) => void
}

export const useBerichtePrefsStore = create<BerichtePrefsState>()(
  persist(
    (set, get) => ({
      defaultFormat: 'pdf',
      defaultPeriod: 'thisMonth',
      defaultPalette: 'default',
      allowedFormats: ['pdf', 'xlsx', 'csv'],
      scheduleDomains: ['zentria.tech'],
      setDefaultFormat: (defaultFormat) => set({ defaultFormat }),
      setDefaultPeriod: (defaultPeriod) => set({ defaultPeriod }),
      setDefaultPalette: (defaultPalette) => set({ defaultPalette }),
      toggleAllowedFormat: (f) => {
        const cur = get().allowedFormats
        // keep at least one format enabled
        const next = cur.includes(f) ? cur.filter((x) => x !== f) : [...cur, f]
        if (next.length === 0) return
        set({ allowedFormats: next })
      },
      addScheduleDomain: (d) => {
        const dom = d.trim().toLowerCase().replace(/^@/, '')
        if (!dom || get().scheduleDomains.includes(dom)) return
        set({ scheduleDomains: [...get().scheduleDomains, dom] })
      },
      removeScheduleDomain: (d) =>
        set({ scheduleDomains: get().scheduleDomains.filter((x) => x !== d) }),
    }),
    { name: 'cosmi-berichte-prefs' },
  ),
)

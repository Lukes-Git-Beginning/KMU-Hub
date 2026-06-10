import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { FileSortField, SortDirection } from '@/api/types/document-types'

/**
 * Personal dokumente preferences — per-user comfort settings that adapt the
 * module to one's own workflow (personal scope, see ModuleSettingsShell).
 * Persisted locally; no tenant impact.
 *
 * Note: the per-folder grid/list choice (localStorage `cosmi-view-pref-*`)
 * overrides `defaultView`; this store only provides the fallback for folders
 * without an explicit choice.
 */
export type DokumenteView = 'grid' | 'list'
export type DokumenteDensity = 'comfortable' | 'compact'

interface DokumentePrefsState {
  /** Fallback view for folders without a per-folder choice. */
  defaultView: DokumenteView
  /** Default file sorting (field + direction) — bound to the toolbar SortMenu. */
  sortField: FileSortField
  sortDir: SortDirection
  /** Row/card density. */
  density: DokumenteDensity
  /** Show document-page previews on grid tiles (instead of plain type icons). */
  showPreviews: boolean
  setDefaultView: (v: DokumenteView) => void
  setSort: (field: FileSortField, dir: SortDirection) => void
  setDensity: (d: DokumenteDensity) => void
  setShowPreviews: (show: boolean) => void
}

export const useDokumentePrefsStore = create<DokumentePrefsState>()(
  persist(
    (set) => ({
      defaultView: 'grid',
      sortField: 'date',
      sortDir: 'desc',
      density: 'comfortable',
      showPreviews: true,
      setDefaultView: (defaultView) => set({ defaultView }),
      setSort: (sortField, sortDir) => set({ sortField, sortDir }),
      setDensity: (density) => set({ density }),
      setShowPreviews: (showPreviews) => set({ showPreviews }),
    }),
    { name: 'cosmi-dokumente-prefs' },
  ),
)

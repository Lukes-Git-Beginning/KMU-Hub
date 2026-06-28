import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { FileSortField, SortDirection } from '@/api/types/document-types'
import { loadModuleSettings, saveModuleSettings } from '@/api/settings-persist'

/**
 * Personal dokumente preferences (personal/user scope, see ModuleSettingsShell).
 * Server-synced via /settings/dokumente/user (X-4); hydrated by the central
 * settings hydrator on app shell mount, write-through on each setter.
 * (Tenant-wide dokumente settings live separately in dokumenteSettings.ts.)
 *
 * Note: the per-folder grid/list choice (localStorage `cosmi-view-pref-*`)
 * overrides `defaultView`; this store only provides the fallback for folders
 * without an explicit choice.
 */
const MODULE_ID = 'dokumente'

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
  serverInitialized: boolean
  setDefaultView: (v: DokumenteView) => void
  setSort: (field: FileSortField, dir: SortDirection) => void
  setDensity: (d: DokumenteDensity) => void
  setShowPreviews: (show: boolean) => void
  initFromServer: () => Promise<void>
}

function userPayload(s: DokumentePrefsState): Record<string, unknown> {
  return {
    defaultView: s.defaultView,
    sortField: s.sortField,
    sortDir: s.sortDir,
    density: s.density,
    showPreviews: s.showPreviews,
  }
}

export const useDokumentePrefsStore = create<DokumentePrefsState>()(
  persist(
    (set, get) => ({
      defaultView: 'grid',
      sortField: 'date',
      sortDir: 'desc',
      density: 'comfortable',
      showPreviews: true,
      serverInitialized: false,
      setDefaultView: (defaultView) => {
        set({ defaultView })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      setSort: (sortField, sortDir) => {
        set({ sortField, sortDir })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      setDensity: (density) => {
        set({ density })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      setShowPreviews: (showPreviews) => {
        set({ showPreviews })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      initFromServer: async () => {
        if (get().serverInitialized) return
        const map = await loadModuleSettings(MODULE_ID, 'user')
        set((s) => ({
          defaultView: map.defaultView === 'grid' || map.defaultView === 'list' ? map.defaultView : s.defaultView,
          sortField: typeof map.sortField === 'string' ? (map.sortField as FileSortField) : s.sortField,
          sortDir: map.sortDir === 'asc' || map.sortDir === 'desc' ? map.sortDir : s.sortDir,
          density: map.density === 'comfortable' || map.density === 'compact' ? map.density : s.density,
          showPreviews: typeof map.showPreviews === 'boolean' ? map.showPreviews : s.showPreviews,
          serverInitialized: true,
        }))
      },
    }),
    { name: 'cosmi-dokumente-prefs' },
  ),
)

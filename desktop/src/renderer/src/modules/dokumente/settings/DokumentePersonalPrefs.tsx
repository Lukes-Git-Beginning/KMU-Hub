/**
 * DokumentePersonalPrefs — personal-scope settings for the dokumente module
 * (everyone can adapt these to their own workflow). Embedded as the
 * "Persönlich" section of the dokumente settings panel. Backed by
 * stores/dokumentePrefs.
 */
import { useTranslation } from 'react-i18next'
import { LayoutGrid, List, Rows3, Rows4, ArrowUp, ArrowDown } from 'lucide-react'
import { Switch } from '@/components/ui/switch'
import {
  useDokumentePrefsStore,
  type DokumenteView,
  type DokumenteDensity,
} from '@/stores/dokumentePrefs'
import type { FileSortField, SortDirection } from '@/api/types/document-types'

export function DokumentePersonalPrefs() {
  const { t } = useTranslation()
  const defaultView = useDokumentePrefsStore((s) => s.defaultView)
  const sortField = useDokumentePrefsStore((s) => s.sortField)
  const sortDir = useDokumentePrefsStore((s) => s.sortDir)
  const density = useDokumentePrefsStore((s) => s.density)
  const showPreviews = useDokumentePrefsStore((s) => s.showPreviews)
  const setDefaultView = useDokumentePrefsStore((s) => s.setDefaultView)
  const setSort = useDokumentePrefsStore((s) => s.setSort)
  const setDensity = useDokumentePrefsStore((s) => s.setDensity)
  const setShowPreviews = useDokumentePrefsStore((s) => s.setShowPreviews)

  // `as const` keeps the labelKeys as literal types for the typed t().
  const viewOptions = [
    { id: 'grid', labelKey: 'dokumente.settings.personal.viewGrid', icon: LayoutGrid },
    { id: 'list', labelKey: 'dokumente.settings.personal.viewList', icon: List },
  ] as const satisfies readonly { id: DokumenteView; labelKey: string; icon: typeof List }[]
  const sortFieldOptions = [
    { id: 'name', labelKey: 'dokumente.list.name' },
    { id: 'size', labelKey: 'dokumente.list.size' },
    { id: 'type', labelKey: 'dokumente.list.type' },
    { id: 'date', labelKey: 'dokumente.list.date' },
  ] as const satisfies readonly { id: FileSortField; labelKey: string }[]
  const sortDirOptions = [
    { id: 'asc', labelKey: 'common.sort.ascending', icon: ArrowUp },
    { id: 'desc', labelKey: 'common.sort.descending', icon: ArrowDown },
  ] as const satisfies readonly { id: SortDirection; labelKey: string; icon: typeof ArrowUp }[]
  const densityOptions = [
    { id: 'comfortable', labelKey: 'dokumente.settings.personal.densityComfortable', icon: Rows3 },
    { id: 'compact', labelKey: 'dokumente.settings.personal.densityCompact', icon: Rows4 },
  ] as const satisfies readonly { id: DokumenteDensity; labelKey: string; icon: typeof Rows3 }[]

  const segBtn = (active: boolean) =>
    `flex flex-1 items-center justify-center gap-2 rounded-lg border py-2 text-sm transition-colors ${
      active
        ? 'border-primary bg-primary/5 font-medium text-primary'
        : 'border-border text-foreground hover:bg-secondary'
    }`

  return (
    <div className="space-y-5">
      {/* Default view (fallback — per-folder choices override it) */}
      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">
          {t('dokumente.settings.personal.defaultView')}
        </label>
        <div className="flex gap-2">
          {viewOptions.map((opt) => {
            const Icon = opt.icon
            return (
              <button key={opt.id} onClick={() => setDefaultView(opt.id)} className={segBtn(defaultView === opt.id)}>
                <Icon className="h-4 w-4" />
                {t(opt.labelKey)}
              </button>
            )
          })}
        </div>
        <p className="text-xs text-muted-foreground">{t('dokumente.settings.personal.defaultViewHint')}</p>
      </div>

      {/* Default sorting: field + direction */}
      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">
          {t('dokumente.settings.personal.defaultSort')}
        </label>
        <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
          {sortFieldOptions.map((opt) => (
            <button key={opt.id} onClick={() => setSort(opt.id, sortDir)} className={segBtn(sortField === opt.id)}>
              {t(opt.labelKey)}
            </button>
          ))}
        </div>
        <div className="flex gap-2">
          {sortDirOptions.map((opt) => {
            const Icon = opt.icon
            return (
              <button key={opt.id} onClick={() => setSort(sortField, opt.id)} className={segBtn(sortDir === opt.id)}>
                <Icon className="h-4 w-4" />
                {t(opt.labelKey)}
              </button>
            )
          })}
        </div>
      </div>

      {/* Density */}
      <div className="space-y-1.5">
        <label className="text-sm font-medium text-foreground">
          {t('dokumente.settings.personal.density')}
        </label>
        <div className="flex gap-2">
          {densityOptions.map((opt) => {
            const Icon = opt.icon
            return (
              <button key={opt.id} onClick={() => setDensity(opt.id)} className={segBtn(density === opt.id)}>
                <Icon className="h-4 w-4" />
                {t(opt.labelKey)}
              </button>
            )
          })}
        </div>
      </div>

      {/* Tile previews */}
      <div className="flex items-center justify-between gap-3 rounded-lg border border-border px-3 py-2.5">
        <span>
          <span className="block text-sm text-foreground">
            {t('dokumente.settings.personal.showPreviews')}
          </span>
          <span className="block text-xs text-muted-foreground">
            {t('dokumente.settings.personal.showPreviewsHint')}
          </span>
        </span>
        <Switch checked={showPreviews} onCheckedChange={setShowPreviews} />
      </div>
    </div>
  )
}

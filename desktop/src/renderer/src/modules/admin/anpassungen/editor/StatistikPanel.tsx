/**
 * StatistikPanel (Modul-Editor) — the stats-view widget catalog. Toggles which
 * cards/charts the module's Statistik tab shows, writing to the DRAFT layer via
 * setDraftModuleArea under a `stat:` prefix (reusing the areas draft/resolve/
 * deploy/undo machinery). Locked widgets (e.g. CSAT) have no real data source yet
 * → shown greyed with a hint and cannot be enabled until the feature ships.
 */
import { useTranslation } from 'react-i18next'
import { Eye, EyeOff, Lock, BarChart3 } from 'lucide-react'
import { resolveModuleAreas } from '@/mocks/data/customization'
import { useCustomFields } from '@/api/hooks/useCustomFields'
import { getEditorModule } from './editorModules'
import { useDraftConfig } from './DraftConfigProvider'

/** Prefix that separates stat-widget toggles from real tab areas in moduleAreas. */
export const STAT_AREA_PREFIX = 'stat:'

export function StatistikPanel({ moduleKey }: { moduleKey: string }): React.ReactElement {
  const { t } = useTranslation()
  const { moduleAreas: draftAreas, setDraftModuleArea, customFields: draftFields } = useDraftConfig()
  const module = getEditorModule(moduleKey)
  const entity = module?.fieldEntities?.[0]
  const { data: liveFields = [] } = useCustomFields(entity ?? 'crm_contact')

  // Breakdown widgets the module derives at runtime — one per select field, so a
  // value list added in THIS session already has a switch here (Darien 2026-08-04:
  // "muss auch gehen im Editor wenn's noch nicht übernommen wurde"). Reads the
  // draft snapshot ⊕ live definitions, so it never waits for a deploy.
  const derived = (entity ? (draftFields[entity] ?? liveFields) : [])
    .filter((f) => f.visible && f.type === 'select' && (f.valueSetId || f.options.length > 0))
    .map((f) => ({ key: `field:${f.key}`, label: t('helpdesk.stats.byField', { field: f.label }), locked: false }))

  const widgets = [
    ...(module?.statWidgets ?? []).map((w) => ({ key: w.key, label: t(w.labelKey), locked: w.locked })),
    ...derived,
  ]

  if (widgets.length === 0) {
    return (
      <div className="flex flex-1 flex-col items-center justify-center gap-2 px-6 text-center">
        <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-muted text-muted-foreground">
          <BarChart3 className="h-5 w-5" aria-hidden="true" />
        </div>
        <p className="text-sm text-muted-foreground">{t('customization.editor.statistik.empty')}</p>
      </div>
    )
  }

  // Effective state = tenant ⊕ draft; a missing key means visible.
  const resolved = resolveModuleAreas(moduleKey, false, draftAreas)

  return (
    <div className="flex flex-1 flex-col gap-2 overflow-y-auto px-4 py-3">
      <p className="px-0.5 pb-1 text-xs leading-relaxed text-muted-foreground">
        {t('customization.editor.statistik.hint')}
      </p>
      {widgets.map((w) => {
        const areaKey = `${STAT_AREA_PREFIX}${w.key}`

        // Locked widget: no data source yet → greyed, non-toggleable, with a hint.
        if (w.locked) {
          return (
            <div key={w.key} className="flex items-center gap-2 rounded-lg border border-dashed px-3 py-2.5 opacity-80">
              <Lock className="h-3.5 w-3.5 shrink-0 text-muted-foreground" aria-hidden="true" />
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm text-muted-foreground">{w.label}</p>
                <p className="truncate text-[11px] text-muted-foreground">{t('customization.editor.statistik.csatLocked')}</p>
              </div>
            </div>
          )
        }

        const enabled = resolved[areaKey] !== false
        return (
          <div
            key={w.key}
            className={`flex items-center gap-2 rounded-lg border px-3 py-2.5 ${enabled ? 'bg-card' : 'bg-muted/40'}`}
          >
            <span className={`min-w-0 flex-1 truncate text-sm ${enabled ? 'text-foreground' : 'text-muted-foreground line-through'}`}>
              {w.label}
            </span>
            <button
              type="button"
              role="switch"
              aria-checked={enabled}
              onClick={() => setDraftModuleArea(moduleKey, areaKey, !enabled)}
              aria-label={t(enabled ? 'customization.editor.statistik.hide' : 'customization.editor.statistik.show', { widget: w.label })}
              title={t(enabled ? 'customization.editor.statistik.visible' : 'customization.editor.statistik.hidden')}
              className={`flex h-7 items-center gap-1.5 rounded-md px-2 text-xs font-medium transition-colors ${
                enabled
                  ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 hover:bg-emerald-500/20'
                  : 'bg-secondary text-muted-foreground hover:bg-secondary/80'
              }`}
            >
              {enabled ? <Eye className="h-3.5 w-3.5" aria-hidden="true" /> : <EyeOff className="h-3.5 w-3.5" aria-hidden="true" />}
              {t(enabled ? 'customization.editor.statistik.visible' : 'customization.editor.statistik.hidden')}
            </button>
          </div>
        )
      })}
    </div>
  )
}

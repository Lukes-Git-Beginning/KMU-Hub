/**
 * BereichePanel (Modul-Editor, R4) — the module-area visibility editor inside the
 * properties panel. Toggles a module's sub-areas (tabs/sections) on or off for the
 * tenant; writes to the DRAFT layer via setDraftModuleArea, so the change previews
 * live in the canvas (the module reads it via useModuleAreas) and only goes live on
 * "Übernehmen".
 *
 * Sparse + soft: hiding an area removes it from the UI only — data is retained, and
 * a future update can ship new areas that default to on. Modules with no toggleable
 * areas (e.g. router-based CRM tabs) show an empty state.
 */
import { useTranslation } from 'react-i18next'
import { Eye, EyeOff, Layers } from 'lucide-react'
import { resolveModuleAreas } from '@/mocks/data/customization'
import { getEditorModule } from './editorModules'
import { useDraftConfig } from './DraftConfigProvider'

export function BereichePanel({ moduleKey }: { moduleKey: string }): React.ReactElement {
  const { t } = useTranslation()
  const { moduleAreas: draftAreas, setDraftModuleArea } = useDraftConfig()
  const module = getEditorModule(moduleKey)
  const areas = module?.areas ?? []

  if (areas.length === 0) {
    return (
      <div className="flex flex-1 flex-col items-center justify-center gap-2 px-6 text-center">
        <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-muted text-muted-foreground">
          <Layers className="h-5 w-5" aria-hidden="true" />
        </div>
        <p className="text-sm text-muted-foreground">{t('customization.editor.bereiche.empty')}</p>
      </div>
    )
  }

  // Effective state = tenant ⊕ draft; a missing key means enabled.
  const resolved = resolveModuleAreas(moduleKey, false, draftAreas)

  return (
    <div className="flex flex-1 flex-col gap-2 overflow-y-auto px-4 py-3">
      <p className="px-0.5 pb-1 text-xs leading-relaxed text-muted-foreground">
        {t('customization.editor.bereiche.hint')}
      </p>
      {areas.map((area) => {
        const enabled = resolved[area.key] !== false
        return (
          <div
            key={area.key}
            className={`flex items-center gap-2 rounded-lg border px-3 py-2.5 ${enabled ? 'bg-card' : 'bg-muted/40'}`}
          >
            <span className={`min-w-0 flex-1 truncate text-sm ${enabled ? 'text-foreground' : 'text-muted-foreground line-through'}`}>
              {t(area.labelKey)}
            </span>
            <button
              type="button"
              role="switch"
              aria-checked={enabled}
              onClick={() => setDraftModuleArea(moduleKey, area.key, !enabled)}
              aria-label={t(enabled ? 'customization.editor.bereiche.hide' : 'customization.editor.bereiche.show', { area: t(area.labelKey) })}
              title={t(enabled ? 'customization.editor.bereiche.visible' : 'customization.editor.bereiche.hidden')}
              className={`flex h-7 items-center gap-1.5 rounded-md px-2 text-xs font-medium transition-colors ${
                enabled
                  ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 hover:bg-emerald-500/20'
                  : 'bg-secondary text-muted-foreground hover:bg-secondary/80'
              }`}
            >
              {enabled ? <Eye className="h-3.5 w-3.5" aria-hidden="true" /> : <EyeOff className="h-3.5 w-3.5" aria-hidden="true" />}
              {t(enabled ? 'customization.editor.bereiche.visible' : 'customization.editor.bereiche.hidden')}
            </button>
          </div>
        )
      })}
    </div>
  )
}

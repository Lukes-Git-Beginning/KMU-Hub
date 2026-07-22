/**
 * BegriffePanel (Modul-Editor v1, E-3) — the module-scoped term editor inside
 * the properties panel. Lists this module's whitelisted labels; edits write to
 * the DRAFT layer (not the live tenant layer), so they show instantly in the
 * sandbox preview and only go live on "Übernehmen".
 *
 * Reuses the ICU-Live-Fix: the DraftConfigProvider re-applies the merged overlay
 * to the global i18n bundle on every change, so a module rendering t(labelKey)
 * updates live. Editing the current app locale only (multi-locale = later).
 */
import { useTranslation } from 'react-i18next'
import { RotateCcw } from 'lucide-react'
import { i18n } from '@/i18n/i18n'
import { getLabelDefault } from '@/i18n/useLabelOverlay'
import { resolveLabelOverrides } from '@/mocks/data/customization'
import { getEditorModule } from './editorModules'
import { useDraftConfig } from './DraftConfigProvider'

const PROVENANCE_STYLE: Record<string, string> = {
  vendor: 'bg-blue-500/10 text-blue-600 dark:text-blue-400',
  tenant: 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400',
  draft: 'bg-amber-500/10 text-amber-600 dark:text-amber-400',
}

export function BegriffePanel({ moduleKey }: { moduleKey: string }): React.ReactElement {
  const { t } = useTranslation()
  const { labels: draftLabels, setDraftLabel, resetDraftLabel } = useDraftConfig()
  const locale = i18n.language
  const module = getEditorModule(moduleKey)
  const keys = module?.labelKeys ?? []

  const resolved = resolveLabelOverrides(locale, false, draftLabels)

  if (keys.length === 0) {
    return (
      <div className="flex flex-1 flex-col items-center justify-center gap-1.5 px-6 py-8 text-center">
        <p className="text-sm text-muted-foreground">{t('customization.editor.begriffe.noneForModule')}</p>
      </div>
    )
  }

  return (
    <div className="flex flex-1 flex-col gap-2 overflow-y-auto px-4 py-3">
      {keys.map((key) => {
        const entry = resolved[key]
        const provenance = entry?.provenance ?? 'default'
        const defaultTerm = getLabelDefault(locale, key)
        const value = provenance !== 'default' && entry?.value ? entry.value : defaultTerm
        const isDraft = provenance === 'draft'

        return (
          <div key={key} className="rounded-lg border bg-card px-3 py-2.5">
            <div className="flex items-center gap-2">
              <input
                value={value}
                onChange={(e) => setDraftLabel(locale, key, e.target.value)}
                placeholder={t('customization.labels.editPlaceholder')}
                className="h-8 min-w-0 flex-1 rounded-md border border-border bg-background px-2.5 text-sm outline-none focus:border-primary"
              />
              {isDraft && (
                <button
                  type="button"
                  onClick={() => resetDraftLabel(locale, key)}
                  aria-label={t('customization.editor.begriffe.reset')}
                  title={t('customization.editor.begriffe.reset')}
                  className="flex h-8 w-8 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
                >
                  <RotateCcw className="h-3.5 w-3.5" aria-hidden="true" />
                </button>
              )}
            </div>
            <div className="mt-1.5 flex items-center gap-2">
              {provenance !== 'default' && (
                <span
                  className={`inline-flex items-center rounded px-1.5 py-0.5 text-[10px] font-medium ${PROVENANCE_STYLE[provenance] ?? ''}`}
                >
                  {t(`customization.labels.provenance.${provenance}`)}
                </span>
              )}
              {provenance !== 'default' && (
                <span className="truncate text-[11px] text-muted-foreground/70">
                  {t('customization.editor.begriffe.standardHint', { value: defaultTerm })}
                </span>
              )}
              <span className="ml-auto shrink-0 font-mono text-[10px] text-muted-foreground/40" aria-hidden="true">
                {key}
              </span>
            </div>
          </div>
        )
      })}
    </div>
  )
}

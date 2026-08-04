/**
 * SpaltenPanel (Modul-Editor, Darien 2026-08-04) — which columns the module's list
 * view shows. Priority/status are readable from the list without opening a record;
 * a value list added in the editor had no way to get that treatment, and the
 * built-in columns had no way to be dropped. Both are toggles here.
 *
 * Writes to the DRAFT layer via setDraftModuleArea under a `col:` prefix, reusing
 * the areas draft/resolve/deploy/undo machinery (same trick as StatistikPanel's
 * `stat:`). Built-in columns default to ON, custom-field columns to OFF — a new
 * field must not silently widen everyone's table.
 *
 * The custom-field columns are derived from the live definitions ⊕ the draft
 * snapshot, so a field created in this very session is already offered here.
 */
import { useTranslation } from 'react-i18next'
import { Eye, EyeOff, Columns3 } from 'lucide-react'
import { resolveModuleAreas } from '@/mocks/data/customization'
import { useCustomFields } from '@/api/hooks/useCustomFields'
import { getEditorModule } from './editorModules'
import { useDraftConfig } from './DraftConfigProvider'

/** Prefix separating list-column toggles from real tab areas in moduleAreas. */
export const COLUMN_AREA_PREFIX = 'col:'

export function SpaltenPanel({ moduleKey }: { moduleKey: string }): React.ReactElement {
  const { t } = useTranslation()
  const { moduleAreas: draftAreas, setDraftModuleArea, customFields: draftFields } = useDraftConfig()
  const module = getEditorModule(moduleKey)
  const entity = module?.fieldEntities?.[0]
  const { data: liveFields = [] } = useCustomFields(entity ?? 'crm_contact')
  const builtIns = module?.listColumns ?? []

  if (builtIns.length === 0) {
    return (
      <div className="flex flex-1 flex-col items-center justify-center gap-2 px-6 text-center">
        <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-muted text-muted-foreground">
          <Columns3 className="h-5 w-5" aria-hidden="true" />
        </div>
        <p className="text-sm text-muted-foreground">{t('customization.editor.spalten.empty')}</p>
      </div>
    )
  }

  const fields = (entity ? (draftFields[entity] ?? liveFields) : []).filter((f) => f.visible)
  const resolved = resolveModuleAreas(moduleKey, false, draftAreas)

  const rows: { key: string; label: string; optIn: boolean }[] = [
    ...builtIns.map((c) => ({ key: c.key, label: t(c.labelKey), optIn: false })),
    ...fields.map((f) => ({ key: `field:${f.key}`, label: f.label, optIn: true })),
  ]

  return (
    <div className="flex flex-1 flex-col gap-2 overflow-y-auto px-4 py-3">
      <p className="px-0.5 pb-1 text-xs leading-relaxed text-muted-foreground">
        {t('customization.editor.spalten.hint')}
      </p>
      {rows.map(({ key, label, optIn }) => {
        const areaKey = `${COLUMN_AREA_PREFIX}${key}`
        const enabled = optIn ? resolved[areaKey] === true : resolved[areaKey] !== false
        return (
          <div
            key={key}
            className={`flex items-center gap-2 rounded-lg border px-3 py-2.5 ${enabled ? 'bg-card' : 'bg-muted/40'}`}
          >
            <div className="min-w-0 flex-1">
              <p className={`truncate text-sm ${enabled ? 'text-foreground' : 'text-muted-foreground line-through'}`}>
                {label}
              </p>
              {optIn && (
                <p className="truncate text-[11px] text-muted-foreground">
                  {t('customization.editor.spalten.fromField')}
                </p>
              )}
            </div>
            <button
              type="button"
              role="switch"
              aria-checked={enabled}
              onClick={() => setDraftModuleArea(moduleKey, areaKey, !enabled)}
              aria-label={t(enabled ? 'customization.editor.spalten.hide' : 'customization.editor.spalten.show', { column: label })}
              className={`flex h-7 items-center gap-1.5 rounded-md px-2 text-xs font-medium transition-colors ${
                enabled
                  ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 hover:bg-emerald-500/20'
                  : 'bg-secondary text-muted-foreground hover:bg-secondary/80'
              }`}
            >
              {enabled ? <Eye className="h-3.5 w-3.5" aria-hidden="true" /> : <EyeOff className="h-3.5 w-3.5" aria-hidden="true" />}
              {t(enabled ? 'customization.editor.spalten.visible' : 'customization.editor.spalten.hidden')}
            </button>
          </div>
        )
      })}
    </div>
  )
}

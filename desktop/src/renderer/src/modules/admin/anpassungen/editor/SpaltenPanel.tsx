/**
 * SpaltenPanel (Modul-Editor, Darien 2026-08-04/05) — which columns the module's
 * list view shows, and what they contain.
 *
 * Priority/status are readable from the list without opening a record; a value
 * list added in the editor had no way to get that treatment, and the built-in
 * columns had no way to be dropped. Visibility rides the moduleAreas machinery
 * under a `col:` prefix (draft/resolve/deploy/undo come for free); built-ins
 * default to ON, custom-field columns to OFF so a new field never silently widens
 * everyone's table.
 *
 * A column is more than a switch (Darien 2026-08-05: "muss man eig auch gut
 * bearbeiten können — neue anlegen, andere löschen, verschiedene Optionen an
 * Auswahl wählen"), so custom-field columns are created, edited and deleted right
 * here through the shared field draft — same snapshot the Felder panel writes.
 *
 * Value lists that no field points at yet are offered as "Als Spalte hinzufügen":
 * one click creates the bound select field behind it and switches the column on.
 * That closes the gap where a freshly created list appeared nowhere at all — a
 * list needs a column to live in, and this is how it gets one.
 */
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Eye, EyeOff, Columns3, Plus, ListChecks, Pencil } from 'lucide-react'
import { resolveModuleAreas, resolveValueSet } from '@/mocks/data/customization'
import type { CustomFieldDefinition, CustomFieldEntity } from '@/mocks/data/custom-fields'
import { FieldEditorModal, type ValueSetChoice } from '../FieldEditorModal'
import { getEditorModule } from './editorModules'
import { useDraftConfig } from './DraftConfigProvider'
import { useEntityFieldDraft } from './useEntityFieldDraft'

/** Prefix separating list-column toggles from real tab areas in moduleAreas. */
export const COLUMN_AREA_PREFIX = 'col:'

export function SpaltenPanel({ moduleKey }: { moduleKey: string }): React.ReactElement {
  const { t } = useTranslation()
  const module = getEditorModule(moduleKey)
  const entity = module?.fieldEntities?.[0]
  const builtIns = module?.listColumns ?? []

  if (builtIns.length === 0 || !entity) {
    return (
      <div className="flex flex-1 flex-col items-center justify-center gap-2 px-6 text-center">
        <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-muted text-muted-foreground">
          <Columns3 className="h-5 w-5" aria-hidden="true" />
        </div>
        <p className="text-sm text-muted-foreground">{t('customization.editor.spalten.empty')}</p>
      </div>
    )
  }

  return <ColumnsEditor moduleKey={moduleKey} entity={entity} builtIns={builtIns} />
}

function ColumnsEditor({
  moduleKey,
  entity,
  builtIns,
}: {
  moduleKey: string
  entity: CustomFieldEntity
  builtIns: { key: string; labelKey: string; valueSetId?: string }[]
}): React.ReactElement {
  const { t } = useTranslation()
  const { moduleAreas: draftAreas, setDraftModuleArea, valueSets: draftSets } = useDraftConfig()
  const module = getEditorModule(moduleKey)
  const { effective, create, update, remove } = useEntityFieldDraft(entity)
  const [createOpen, setCreateOpen] = useState(false)
  const [editTarget, setEditTarget] = useState<CustomFieldDefinition | null>(null)

  const resolved = resolveModuleAreas(moduleKey, false, draftAreas)
  const fields = effective.filter((f) => f.visible)

  // Value lists selectable in the field dialog: the module's predefined ones plus
  // anything created in the Wertelisten panel this session.
  const valueSetChoices = useMemo<ValueSetChoice[]>(() => {
    const predefined = module?.valueSetIds ?? []
    const ids = [...predefined, ...Object.keys(draftSets).filter((id) => !predefined.includes(id))]
    return ids.flatMap((id) => {
      const set = resolveValueSet(id, false, draftSets)
      if (!set) return []
      return [{
        id,
        name: set.name,
        options: set.options.filter((o) => o.active).map((o) => ({ id: o.id, label: o.label, color: o.color })),
      }]
    })
  }, [module, draftSets])

  // Lists nothing shows yet — one click turns each into a real column. A list a
  // built-in column already renders (priority/status) is NOT unused, even though
  // no custom field points at it.
  const unusedSets = valueSetChoices.filter(
    (s) => !fields.some((f) => f.valueSetId === s.id) && !builtIns.some((c) => c.valueSetId === s.id),
  )

  const setVisible = (key: string, visible: boolean): void =>
    setDraftModuleArea(moduleKey, `${COLUMN_AREA_PREFIX}${key}`, visible)

  /** Turn a value list into a column: create the bound field, switch the column on. */
  const addSetAsColumn = (set: ValueSetChoice): void => {
    const field = create({ entity, label: set.name, type: 'select', valueSetId: set.id, visible: true })
    setVisible(`field:${field.key}`, true)
  }

  const deleteColumn = (field: CustomFieldDefinition): void => {
    remove(field.id)
    setVisible(`field:${field.key}`, false)
    setEditTarget(null)
  }

  const switchFor = (key: string, label: string, enabled: boolean): React.ReactElement => (
    <button
      type="button"
      role="switch"
      aria-checked={enabled}
      onClick={() => setVisible(key, !enabled)}
      aria-label={t(enabled ? 'customization.editor.spalten.hide' : 'customization.editor.spalten.show', { column: label })}
      className={`flex h-7 shrink-0 items-center gap-1.5 rounded-md px-2 text-xs font-medium transition-colors ${
        enabled
          ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 hover:bg-emerald-500/20'
          : 'bg-secondary text-muted-foreground hover:bg-secondary/80'
      }`}
    >
      {enabled ? <Eye className="h-3.5 w-3.5" aria-hidden="true" /> : <EyeOff className="h-3.5 w-3.5" aria-hidden="true" />}
      {t(enabled ? 'customization.editor.spalten.visible' : 'customization.editor.spalten.hidden')}
    </button>
  )

  return (
    <div className="flex flex-1 flex-col gap-2 overflow-y-auto px-4 py-3">
      <p className="px-0.5 pb-1 text-xs leading-relaxed text-muted-foreground">
        {t('customization.editor.spalten.hint')}
      </p>

      {/* Value lists with no column yet — the missing link between "I made a list"
          and "I see it in the overview". Sits at the TOP: this is where there is
          something to do, and buried at the bottom it kept getting missed. Gone
          entirely once every list has a column, so it never becomes noise. */}
      {unusedSets.length > 0 && (
        <div className="rounded-lg border border-dashed border-[var(--accent-1)]/40 bg-[var(--accent-1)]/5 px-3 py-2.5">
          <p className="mb-2 text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
            {t('customization.editor.spalten.availableSets')}
          </p>
          <div className="flex flex-col gap-1.5">
            {unusedSets.map((s) => (
              <button
                key={s.id}
                type="button"
                onClick={() => addSetAsColumn(s)}
                className="flex items-center gap-2 rounded-md border border-border bg-card px-2.5 py-2 text-left transition-colors hover:border-primary/40 hover:bg-accent/40"
              >
                <ListChecks className="h-3.5 w-3.5 shrink-0 text-muted-foreground" aria-hidden="true" />
                <span className="min-w-0 flex-1">
                  <span className="block truncate text-sm text-foreground">{s.name}</span>
                  <span className="block truncate text-[11px] text-muted-foreground">
                    {s.options.length > 0
                      ? s.options.map((o) => o.label).join(', ')
                      : t('customization.editor.spalten.setEmpty')}
                  </span>
                </span>
                <Plus className="h-3.5 w-3.5 shrink-0 text-primary" aria-hidden="true" />
              </button>
            ))}
          </div>
          <p className="mt-2 text-[11px] leading-relaxed text-muted-foreground">
            {t('customization.editor.spalten.availableSetsHint')}
          </p>
        </div>
      )}

      {/* Built-in columns — switchable, but not deletable: they render record data
          the module owns, not a tenant-defined field. */}
      {builtIns.map(({ key, labelKey }) => {
        const label = t(labelKey)
        const enabled = resolved[`${COLUMN_AREA_PREFIX}${key}`] !== false
        return (
          <div key={key} className={`flex items-center gap-2 rounded-lg border px-3 py-2.5 ${enabled ? 'bg-card' : 'bg-muted/40'}`}>
            <span className={`min-w-0 flex-1 truncate text-sm ${enabled ? 'text-foreground' : 'text-muted-foreground line-through'}`}>
              {label}
            </span>
            {switchFor(key, label, enabled)}
          </div>
        )
      })}

      {/* Custom-field columns — full editing: click to change name, type, options or
          the bound value list; delete removes the column and its field. */}
      {fields.map((f) => {
        const colKey = `field:${f.key}`
        const enabled = resolved[`${COLUMN_AREA_PREFIX}${colKey}`] === true
        const boundSet = f.valueSetId ? valueSetChoices.find((s) => s.id === f.valueSetId) : undefined
        return (
          <div key={f.id} className={`flex items-center gap-2 rounded-lg border px-3 py-2.5 ${enabled ? 'bg-card' : 'bg-muted/40'}`}>
            <button
              type="button"
              onClick={() => setEditTarget(f)}
              className="group flex min-w-0 flex-1 items-center gap-1.5 text-left"
              // aria-label, nicht title: der Textinhalt (Name + Herkunft) würde sonst
              // den zugänglichen Namen bilden und die Aktion verschlucken.
              aria-label={t('customization.editor.spalten.editColumn', { column: f.label })}
              title={t('customization.editor.spalten.editColumn', { column: f.label })}
            >
              <span className="min-w-0 flex-1">
                <span className={`block truncate text-sm ${enabled ? 'text-foreground' : 'text-muted-foreground line-through'}`}>
                  {f.label}
                </span>
                <span className="block truncate text-[11px] text-muted-foreground">
                  {boundSet
                    ? t('customization.fields.boundToSet', { set: boundSet.name })
                    : t('customization.editor.spalten.fromField')}
                </span>
              </span>
              <Pencil className="h-3 w-3 shrink-0 text-muted-foreground/0 transition-colors group-hover:text-muted-foreground" aria-hidden="true" />
            </button>
            {switchFor(colKey, f.label, enabled)}
          </div>
        )
      })}

      <button
        type="button"
        onClick={() => setCreateOpen(true)}
        className="mt-1 flex items-center justify-center gap-1.5 rounded-lg border border-dashed border-border px-3 py-2.5 text-xs font-medium text-muted-foreground transition-colors hover:border-primary/40 hover:bg-secondary hover:text-foreground"
      >
        <Plus className="h-3.5 w-3.5" aria-hidden="true" />
        {t('customization.editor.spalten.newColumn')}
      </button>

      {createOpen && (
        <FieldEditorModal
          open
          mode="create"
          entity={entity}
          valueSetChoices={valueSetChoices}
          onClose={() => setCreateOpen(false)}
          onCreate={(input) => {
            const field = create(input)
            setVisible(`field:${field.key}`, true)
            setCreateOpen(false)
          }}
        />
      )}

      {editTarget && (
        <FieldEditorModal
          open
          mode="edit"
          entity={entity}
          field={editTarget}
          valueSetChoices={valueSetChoices}
          onClose={() => setEditTarget(null)}
          onUpdate={(input) => { update(editTarget.id, input); setEditTarget(null) }}
          onDeleteRequest={deleteColumn}
        />
      )}
    </div>
  )
}

/**
 * SpaltenPanel (Modul-Editor, Darien 2026-08-04/05) — which columns the module's
 * list view shows, what they contain, in which order and how wide.
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
 *
 * ★ Round 4 (Darien 2026-08-05): built-in and custom columns sit in ONE sortable
 * list, because order runs across both. Built-ins are renamable here (writing the
 * same label override as clicking the heading in the preview — one name, one
 * store) but stay undeletable, with the reason on screen instead of a missing
 * button: they render data the module owns, so dropping them would gut the list.
 * Width is dragged on the column edge in the preview and only mirrored here.
 */
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Eye,
  EyeOff,
  Columns3,
  Plus,
  ListChecks,
  Pencil,
  GripVertical,
  RotateCcw,
  Lock,
} from 'lucide-react'
import {
  DndContext,
  closestCenter,
  PointerSensor,
  KeyboardSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
} from '@dnd-kit/core'
import {
  SortableContext,
  sortableKeyboardCoordinates,
  verticalListSortingStrategy,
  arrayMove,
  useSortable,
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { i18n } from '@/i18n/i18n'
import { getLabelDefault } from '@/i18n/useLabelOverlay'
import {
  resolveModuleAreas,
  resolveModuleAreaLayout,
  resolveValueSet,
  resolveLabelOverrides,
} from '@/mocks/data/customization'
import { COLUMN_AREA_PREFIX } from '@/components/customization/EditorSurface'
import type { CustomFieldDefinition, CustomFieldEntity } from '@/mocks/data/custom-fields'
import { FieldEditorModal, type ValueSetChoice } from '../FieldEditorModal'
import { getEditorModule } from './editorModules'
import { useDraftConfig } from './DraftConfigProvider'
import { useEntityFieldDraft } from './useEntityFieldDraft'

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

/** One row of the panel: a built-in column or one backed by a custom field. */
interface ColumnRow {
  /** moduleAreas key AND drag id — unique across both kinds. */
  areaKey: string
  /** Column key as the module's list knows it. */
  key: string
  label: string
  visible: boolean
  width?: number
  /** Renders module-owned data → renamable and hideable, never deletable. */
  builtIn: boolean
  /** i18n key carrying the heading (built-ins). */
  labelKey?: string
  /** Whether the heading currently carries a draft rename. */
  renamed?: boolean
  /** The definition behind a custom-field column. */
  field?: CustomFieldDefinition
  /** Name of the value list this column renders, when bound. */
  boundSetName?: string
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
  const {
    moduleAreas: draftAreas,
    setDraftModuleArea,
    setDraftModuleAreaOrder,
    patchDraftModuleArea,
    valueSets: draftSets,
    labels: draftLabels,
    setDraftLabel,
    resetDraftLabel,
  } = useDraftConfig()
  const module = getEditorModule(moduleKey)
  const { effective, create, update, remove } = useEntityFieldDraft(entity)
  const [createOpen, setCreateOpen] = useState(false)
  const [editTarget, setEditTarget] = useState<CustomFieldDefinition | null>(null)
  const [renaming, setRenaming] = useState<string | null>(null)
  const locale = i18n.language

  const resolved = resolveModuleAreas(moduleKey, false, draftAreas)
  const layout = resolveModuleAreaLayout(moduleKey, false, draftAreas)
  const resolvedLabels = resolveLabelOverrides(locale, false, draftLabels)
  const fields = effective.filter((f) => f.visible)

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  )

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

  // Built-ins first, custom columns after — the code order, used whenever a column
  // carries no explicit position yet. One drag gives every column one.
  const rows: ColumnRow[] = [
    ...builtIns.map(({ key, labelKey }) => {
      const entry = resolvedLabels[labelKey]
      const renamed = entry?.provenance === 'draft'
      return {
        areaKey: `${COLUMN_AREA_PREFIX}${key}`,
        key,
        // Same value the preview heading shows: draft ⊕ tenant ⊕ code default.
        label: entry?.value || getLabelDefault(locale, labelKey) || t(labelKey),
        visible: resolved[`${COLUMN_AREA_PREFIX}${key}`] !== false,
        builtIn: true,
        labelKey,
        renamed,
      }
    }),
    ...fields.map((f) => ({
      areaKey: `${COLUMN_AREA_PREFIX}field:${f.key}`,
      key: `field:${f.key}`,
      label: f.label,
      visible: resolved[`${COLUMN_AREA_PREFIX}field:${f.key}`] === true,
      builtIn: false,
      field: f,
      boundSetName: f.valueSetId ? valueSetChoices.find((s) => s.id === f.valueSetId)?.name : undefined,
    })),
  ]
    .map((row, index) => ({ row, index, order: layout[row.areaKey]?.order }))
    .sort((a, b) => (a.order ?? a.index) - (b.order ?? b.index) || a.index - b.index)
    .map(({ row }) => ({ ...row, width: layout[row.areaKey]?.width }))

  const setVisible = (key: string, visible: boolean): void =>
    setDraftModuleArea(moduleKey, `${COLUMN_AREA_PREFIX}${key}`, visible)

  const handleDragEnd = (event: DragEndEvent): void => {
    const { active, over } = event
    if (!over || active.id === over.id) return
    const from = rows.findIndex((r) => r.areaKey === active.id)
    const to = rows.findIndex((r) => r.areaKey === over.id)
    if (from === -1 || to === -1) return
    // One dispatch for the whole list: every column gets its new position, and the
    // move stays a single undo step.
    setDraftModuleAreaOrder(moduleKey, arrayMove(rows, from, to).map((r) => r.areaKey))
  }

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

      {/* One list for both kinds — order runs across built-in and custom columns. */}
      <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
        <SortableContext items={rows.map((r) => r.areaKey)} strategy={verticalListSortingStrategy}>
          <div className="flex flex-col gap-2">
            {rows.map((row) => (
              <SortableColumnRow
                key={row.areaKey}
                row={row}
                renaming={renaming === row.areaKey}
                onStartRename={() => setRenaming(row.areaKey)}
                onCancelRename={() => setRenaming(null)}
                onRename={(value) => {
                  if (row.labelKey) setDraftLabel(locale, row.labelKey, value)
                  setRenaming(null)
                }}
                onResetName={() => row.labelKey && resetDraftLabel(locale, row.labelKey)}
                onToggle={() => setVisible(row.key, !row.visible)}
                onEditField={() => row.field && setEditTarget(row.field)}
                onResetWidth={() => patchDraftModuleArea(moduleKey, row.areaKey, { width: undefined })}
              />
            ))}
          </div>
        </SortableContext>
      </DndContext>

      {/* The two rules that have no button of their own: how width is set, and why
          built-in columns have no delete. Once, at the foot of the list — repeating
          them per row only truncated the row label. */}
      <p className="px-0.5 pt-0.5 text-[11px] leading-relaxed text-muted-foreground">
        {t('customization.editor.spalten.widthHint')}
      </p>
      <p className="px-0.5 text-[11px] leading-relaxed text-muted-foreground">
        {t('customization.editor.spalten.builtInHint')}
      </p>

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

function SortableColumnRow({
  row,
  renaming,
  onStartRename,
  onCancelRename,
  onRename,
  onResetName,
  onToggle,
  onEditField,
  onResetWidth,
}: {
  row: ColumnRow
  renaming: boolean
  onStartRename: () => void
  onCancelRename: () => void
  onRename: (value: string) => void
  onResetName: () => void
  onToggle: () => void
  onEditField: () => void
  onResetWidth: () => void
}): React.ReactElement {
  const { t } = useTranslation()
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: row.areaKey,
  })

  return (
    <div
      ref={setNodeRef}
      style={{ transform: CSS.Transform.toString(transform), transition }}
      className={`flex items-center gap-2 rounded-lg border px-2 py-2.5 ${row.visible ? 'bg-card' : 'bg-muted/40'} ${
        isDragging ? 'opacity-50' : ''
      }`}
    >
      <button
        type="button"
        {...attributes}
        {...listeners}
        aria-label={t('customization.editor.spalten.reorder', { column: row.label })}
        title={t('customization.editor.spalten.reorder', { column: row.label })}
        className="shrink-0 cursor-grab rounded-md p-0.5 text-muted-foreground/50 transition-colors hover:text-muted-foreground active:cursor-grabbing"
      >
        <GripVertical className="h-4 w-4" aria-hidden="true" />
      </button>

      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
        {renaming ? (
          <input
            autoFocus
            defaultValue={row.label}
            onKeyDown={(e) => {
              if (e.key === 'Enter') onRename((e.target as HTMLInputElement).value)
              else if (e.key === 'Escape') onCancelRename()
            }}
            onBlur={(e) => onRename(e.target.value)}
            aria-label={t('customization.editor.spalten.rename', { column: row.label })}
            className="h-7 w-full min-w-0 rounded-md border border-border bg-background px-2 text-sm outline-none focus:border-primary"
          />
        ) : (
          <button
            type="button"
            // Built-ins rename in place (same override the preview heading writes);
            // custom columns open the full field dialog, where the bound value list
            // and the options live too.
            onClick={row.builtIn ? onStartRename : onEditField}
            aria-label={t(
              row.builtIn ? 'customization.editor.spalten.rename' : 'customization.editor.spalten.editColumn',
              { column: row.label },
            )}
            title={t(
              row.builtIn ? 'customization.editor.spalten.rename' : 'customization.editor.spalten.editColumn',
              { column: row.label },
            )}
            className="group flex min-w-0 items-center gap-1.5 text-left"
          >
            <span
              className={`min-w-0 truncate text-sm ${row.visible ? 'text-foreground' : 'text-muted-foreground line-through'}`}
            >
              {row.label}
            </span>
            <Pencil
              className="h-3 w-3 shrink-0 text-muted-foreground/0 transition-colors group-hover:text-muted-foreground"
              aria-hidden="true"
            />
          </button>
        )}

        {/* Wraps instead of truncating: with a width set, origin + width + reset do
            not fit on one line in the panel's column. */}
        <span className="flex min-w-0 flex-wrap items-center gap-x-1.5 text-[11px] text-muted-foreground">
          {row.builtIn ? (
            <span className="flex items-center gap-1">
              <Lock className="h-2.5 w-2.5 shrink-0" aria-hidden="true" />
              {t('customization.editor.spalten.builtIn')}
            </span>
          ) : (
            <span className="min-w-0 truncate">
              {row.boundSetName
                ? t('customization.fields.boundToSet', { set: row.boundSetName })
                : t('customization.editor.spalten.fromField')}
            </span>
          )}
          {row.width !== undefined && (
            <>
              {/* Own line, no separator: origin + width never fit side by side in
                  the panel's width, and a dangling "·" at a wrap looks like debris.
                  The number is a READING (Darien: "da stehen nur zahlen … man checkt
                  das nicht"), the reset is its own labelled control next to it. */}
              <span className="w-full" aria-hidden="true" />
              <span className="shrink-0">
                {t('customization.editor.spalten.width', { percent: Math.round(row.width * 100) })}
              </span>
              <button
                type="button"
                onClick={onResetWidth}
                aria-label={t('customization.editor.spalten.widthReset', { column: row.label })}
                title={t('customization.editor.spalten.widthReset', { column: row.label })}
                className="flex shrink-0 items-center gap-1 rounded px-1 py-0.5 transition-colors hover:bg-secondary hover:text-foreground"
              >
                <RotateCcw className="h-2.5 w-2.5" aria-hidden="true" />
                {t('customization.editor.spalten.widthResetShort')}
              </button>
            </>
          )}
        </span>
      </div>

      {row.builtIn && row.renamed && (
        <button
          type="button"
          onClick={onResetName}
          aria-label={t('customization.editor.spalten.nameReset', { column: row.label })}
          title={t('customization.editor.spalten.nameReset', { column: row.label })}
          className="flex h-7 w-7 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
        >
          <RotateCcw className="h-3.5 w-3.5" aria-hidden="true" />
        </button>
      )}

      <button
        type="button"
        role="switch"
        aria-checked={row.visible}
        onClick={onToggle}
        aria-label={t(
          row.visible ? 'customization.editor.spalten.hide' : 'customization.editor.spalten.show',
          { column: row.label },
        )}
        className={`flex h-7 shrink-0 items-center gap-1.5 rounded-md px-2 text-xs font-medium transition-colors ${
          row.visible
            ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 hover:bg-emerald-500/20'
            : 'bg-secondary text-muted-foreground hover:bg-secondary/80'
        }`}
      >
        {row.visible ? (
          <Eye className="h-3.5 w-3.5" aria-hidden="true" />
        ) : (
          <EyeOff className="h-3.5 w-3.5" aria-hidden="true" />
        )}
        {t(row.visible ? 'customization.editor.spalten.visible' : 'customization.editor.spalten.hidden')}
      </button>
    </div>
  )
}

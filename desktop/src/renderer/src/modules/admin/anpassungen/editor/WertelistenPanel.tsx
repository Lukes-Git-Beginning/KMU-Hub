/**
 * WertelistenPanel (Modul-Editor v1, E-3b) — the module-scoped value-set editor
 * inside the properties panel. Edits (set name, option labels, colors, soft-
 * delete) write to the DRAFT layer via setDraftValueSet, so they show in the
 * in-panel chip preview immediately and only go live on "Übernehmen".
 *
 * Modules that consume the resolver (via useModuleValueSet) now preview these
 * edits live in the canvas itself (Helpdesk pilot, P2); the in-panel chip preview
 * below stays as a compact overview and for modules not yet wired.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { RotateCcw, EyeOff, Plus, Trash2, GripVertical } from 'lucide-react'
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
import { resolveValueSet, listTenantValueSetsForModule } from '@/mocks/data/customization'
import type { ResolvedValueSet, ValueSetOption } from '@/api/customization-types'
import { getEditorModule } from './editorModules'
import { useDraftConfig } from './DraftConfigProvider'

/**
 * Cosmi status palette (HSL) — the swatches offered for option colors. Each
 * carries a name: six identically-labelled circles are indistinguishable to a
 * screenreader (and to a test), so the name goes into the button's label.
 */
const SWATCHES: { value: string; nameKey: string }[] = [
  { value: 'hsl(215 16% 47%)', nameKey: 'customization.editor.wertelisten.color.gray' },
  { value: 'hsl(217 91% 60%)', nameKey: 'customization.editor.wertelisten.color.blue' },
  { value: 'hsl(38 92% 50%)', nameKey: 'customization.editor.wertelisten.color.amber' },
  { value: 'hsl(25 95% 53%)', nameKey: 'customization.editor.wertelisten.color.orange' },
  { value: 'hsl(142 71% 45%)', nameKey: 'customization.editor.wertelisten.color.green' },
  { value: 'hsl(0 72% 51%)', nameKey: 'customization.editor.wertelisten.color.red' },
]

const PROVENANCE_STYLE: Record<string, string> = {
  vendor: 'bg-blue-500/10 text-blue-600 dark:text-blue-400',
  tenant: 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400',
  draft: 'bg-amber-500/10 text-amber-600 dark:text-amber-400',
}

/** Strip resolver-only provenance to the editable ValueSet shape. */
function toEditable(resolved: ResolvedValueSet): {
  id: string
  name: string
  options: ValueSetOption[]
  moduleKey?: string
} {
  return {
    id: resolved.id,
    name: resolved.name,
    // Carried through every edit: dropping it here would orphan a self-created
    // list on the next keystroke.
    moduleKey: resolved.moduleKey,
    options: resolved.options.map(({ id, label, color, order, active }) => ({ id, label, color, order, active })),
  }
}

/**
 * One draggable option row. The handle is its own button so the label input stays
 * clickable — dragging the whole row would fight with typing in it.
 */
function SortableOption({
  id,
  label,
  active,
  children,
}: {
  id: string
  label: string
  active: boolean
  children: React.ReactNode
}): React.ReactElement {
  const { t } = useTranslation()
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({ id })
  return (
    <div
      ref={setNodeRef}
      style={{ transform: CSS.Transform.toString(transform), transition }}
      className={`flex items-start gap-1.5 rounded-md border px-2.5 py-2 ${active ? 'bg-background' : 'bg-muted/40'} ${
        isDragging ? 'opacity-50' : ''
      }`}
    >
      <button
        type="button"
        {...attributes}
        {...listeners}
        aria-label={t('customization.editor.wertelisten.reorder', { option: label })}
        title={t('customization.editor.wertelisten.reorder', { option: label })}
        className="mt-1 shrink-0 cursor-grab rounded-md p-0.5 text-muted-foreground/50 transition-colors hover:text-muted-foreground active:cursor-grabbing"
      >
        <GripVertical className="h-4 w-4" aria-hidden="true" />
      </button>
      <div className="min-w-0 flex-1">{children}</div>
    </div>
  )
}

function ValueSetEditor({ id, predefined }: { id: string; predefined: boolean }): React.ReactElement | null {
  const { t } = useTranslation()
  const {
    valueSets: draftSets,
    setDraftValueSet,
    resetDraftValueSet,
    valueSetMigrations,
    setDraftValueSetMigration,
    clearDraftValueSetMigration,
  } = useDraftConfig()
  const setMig = valueSetMigrations[id] ?? {}
  // Options that exist in the persisted layers (pre-draft) — records may use them,
  // so deleting one requires reassigning those records. Draft-added options aren't
  // here → they can be deleted straight away (nothing references them yet).
  const baseIds = new Set((resolveValueSet(id, false)?.options ?? []).map((o) => o.id))
  const [reassignFrom, setReassignFrom] = useState<string | null>(null)
  const [reassignTo, setReassignTo] = useState<string>('')
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  )

  const resolved = resolveValueSet(id, false, draftSets)
  if (!resolved) return null

  const editable = toEditable(resolved)
  const isDraft = resolved.provenance === 'draft'
  const commit = (next: typeof editable): void => setDraftValueSet(id, next)

  const patchOption = (optId: string, patch: Partial<ValueSetOption>): void => {
    commit({ ...editable, options: editable.options.map((o) => (o.id === optId ? { ...o, ...patch } : o)) })
  }

  const removeOption = (optId: string): void => {
    commit({ ...editable, options: editable.options.filter((o) => o.id !== optId) })
  }

  const requestDelete = (optId: string): void => {
    if (baseIds.has(optId)) {
      // In use (possibly) → force a reassignment target before removing.
      const firstOther = editable.options.find((o) => o.id !== optId && o.active)
      setReassignFrom(optId)
      setReassignTo(firstOther?.id ?? '')
    } else {
      removeOption(optId)
    }
  }

  const confirmReassign = (): void => {
    if (!reassignFrom || !reassignTo) return
    // Base options can't be dropped from the overlay (the code default would resurface),
    // so soft-remove (active:false) — that hides it from every picker/table/stat — and
    // record where its records go. Shown in the "Entfernt" section, restorable.
    patchOption(reassignFrom, { active: false })
    setDraftValueSetMigration(id, reassignFrom, reassignTo)
    setReassignFrom(null)
  }

  const restoreRemoved = (optId: string): void => {
    patchOption(optId, { active: true })
    clearDraftValueSetMigration(id, optId)
  }

  const labelOf = (optId: string): string => editable.options.find((o) => o.id === optId)?.label ?? optId

  /** The rows on screen — removed-with-migration options live in their own section. */
  const visibleOptions = editable.options.filter((opt) => !setMig[opt.id])

  const handleDragEnd = (event: DragEndEvent): void => {
    const { active: dragged, over } = event
    if (!over || dragged.id === over.id) return
    const from = visibleOptions.findIndex((o) => o.id === dragged.id)
    const to = visibleOptions.findIndex((o) => o.id === over.id)
    if (from < 0 || to < 0) return
    // Renumber across the WHOLE set: the removed ones keep a slot at the end, so
    // restoring one later does not drop it into the middle of the new order.
    const reordered = [
      ...arrayMove(visibleOptions, from, to),
      ...editable.options.filter((opt) => setMig[opt.id]),
    ]
    commit({ ...editable, options: reordered.map((opt, index) => ({ ...opt, order: index })) })
  }

  const addOption = (): void => {
    const order = editable.options.reduce((max, o) => Math.max(max, o.order), -1) + 1
    const id = `opt-${Date.now().toString(36)}`
    commit({
      ...editable,
      options: [
        ...editable.options,
        { id, label: t('customization.editor.wertelisten.newOption'), color: SWATCHES[1].value, order, active: true },
      ],
    })
  }

  return (
    <div className="rounded-lg border bg-card">
      {/* Set header — every set renames here, predefined ones included (Darien
          2026-08-04). The old split (static title + "rename it in the module")
          conflated two different things: the SET's name is metadata of the list
          itself, while the module's column heading is a label edited in place.
          Renaming one never renamed the other, so the hint was misleading. */}
      <div className="flex items-center gap-2 border-b px-3 py-2.5">
        <input
          value={editable.name}
          onChange={(e) => commit({ ...editable, name: e.target.value })}
          aria-label={t('customization.editor.wertelisten.setNameLabel')}
          placeholder={t('customization.editor.wertelisten.newSetName')}
          className="h-8 min-w-0 flex-1 rounded-md border border-border bg-background px-2.5 text-sm font-medium outline-none focus:border-primary"
        />
        {predefined && resolved.provenance !== 'default' && (
          <span className={`shrink-0 rounded px-1.5 py-0.5 text-[10px] font-medium ${PROVENANCE_STYLE[resolved.provenance] ?? ''}`}>
            {t(`customization.labels.provenance.${resolved.provenance}`)}
          </span>
        )}
        {predefined && isDraft && (
          <button
            type="button"
            onClick={() => resetDraftValueSet(id)}
            aria-label={t('customization.editor.begriffe.reset')}
            title={t('customization.editor.begriffe.reset')}
            className="flex h-7 w-7 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
          >
            <RotateCcw className="h-3.5 w-3.5" aria-hidden="true" />
          </button>
        )}
        {!predefined && (
          <button
            type="button"
            onClick={() => resetDraftValueSet(id)}
            aria-label={t('customization.editor.wertelisten.deleteSet')}
            title={t('customization.editor.wertelisten.deleteSet')}
            className="flex h-7 w-7 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-error-light hover:text-error"
          >
            <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
          </button>
        )}
      </div>

      {/* Reassignment prompt — removing an in-use option needs a target for records */}
      {reassignFrom && (
        <div className="border-b border-amber-500/30 bg-amber-500/5 px-3 py-3">
          <p className="text-xs font-medium text-foreground">
            {t('customization.editor.wertelisten.reassignTitle', { option: labelOf(reassignFrom) })}
          </p>
          <p className="mt-0.5 text-[11px] leading-relaxed text-muted-foreground">
            {t('customization.editor.wertelisten.reassignBody')}
          </p>
          <select
            value={reassignTo}
            onChange={(e) => setReassignTo(e.target.value)}
            className="mt-2 h-8 w-full rounded-md border border-border bg-background px-2 text-sm outline-none focus:border-primary"
          >
            {editable.options
              .filter((o) => o.id !== reassignFrom && o.active)
              .map((o) => (
                <option key={o.id} value={o.id}>{o.label}</option>
              ))}
          </select>
          <div className="mt-2.5 flex items-center justify-end gap-2">
            <button
              type="button"
              onClick={() => setReassignFrom(null)}
              className="rounded-md px-2.5 py-1.5 text-xs font-medium text-muted-foreground transition-colors hover:bg-secondary"
            >
              {t('common.cancel')}
            </button>
            <button
              type="button"
              onClick={confirmReassign}
              disabled={!reassignTo}
              className="rounded-md bg-error px-2.5 py-1.5 text-xs font-medium text-white transition-opacity hover:opacity-90 disabled:opacity-40"
            >
              {t('customization.editor.wertelisten.reassignConfirm')}
            </button>
          </div>
        </div>
      )}

      {/* Options (removed-with-migration ones move to the "Entfernt" section below).
          Order is draggable: the resolver has always sorted by `order`, but nothing
          could change it — so an option added later was stuck at the bottom of every
          picker and every statistics breakdown (Darien 2026-08-06). Same handle and
          keyboard behaviour as the Spalten panel. */}
      <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
        <SortableContext items={visibleOptions.map((o) => o.id)} strategy={verticalListSortingStrategy}>
      <div className="flex flex-col gap-2 px-3 py-2.5">
        {visibleOptions.map((opt) => (
          <SortableOption key={opt.id} id={opt.id} active={opt.active} label={opt.label}>
            <div className="flex items-center gap-2">
              <input
                value={opt.label}
                onChange={(e) => patchOption(opt.id, { label: e.target.value })}
                // Ohne Namen sind sechs Eingabefelder in einer Liste nicht
                // unterscheidbar — weder für einen Screenreader noch für die QA.
                aria-label={t('customization.editor.wertelisten.optionNameLabel', { option: opt.label })}
                className={`h-7 min-w-0 flex-1 rounded border border-transparent bg-transparent px-1.5 text-sm outline-none focus:border-border focus:bg-background ${opt.active ? '' : 'text-muted-foreground line-through'}`}
              />
              <button
                type="button"
                onClick={() => patchOption(opt.id, { active: !opt.active })}
                aria-pressed={!opt.active}
                aria-label={t('customization.editor.wertelisten.toggleHiddenFor', { option: opt.label })}
                title={t(opt.active ? 'customization.editor.wertelisten.optionActive' : 'customization.editor.wertelisten.optionHidden')}
                className={`flex h-7 w-7 shrink-0 items-center justify-center rounded transition-colors hover:bg-secondary ${opt.active ? 'text-muted-foreground/50' : 'text-amber-600 dark:text-amber-400'}`}
              >
                <EyeOff className="h-3.5 w-3.5" aria-hidden="true" />
              </button>
              <button
                type="button"
                onClick={() => requestDelete(opt.id)}
                aria-label={t('customization.editor.wertelisten.deleteOption', { option: opt.label })}
                title={t('customization.editor.wertelisten.deleteOption', { option: opt.label })}
                className="flex h-7 w-7 shrink-0 items-center justify-center rounded text-muted-foreground/50 transition-colors hover:bg-error-light hover:text-error"
              >
                <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
              </button>
            </div>
            {/* Colour swatches */}
            <div className="mt-1.5 flex items-center gap-1 pl-5">
              {SWATCHES.map((c) => (
                <button
                  key={c.value}
                  type="button"
                  onClick={() => patchOption(opt.id, { color: c.value })}
                  aria-label={t('customization.editor.wertelisten.colorLabelFor', {
                    color: t(c.nameKey),
                    option: opt.label,
                  })}
                  aria-pressed={opt.color === c.value}
                  className={`h-4 w-4 rounded-full border transition-transform hover:scale-110 ${opt.color === c.value ? 'ring-2 ring-foreground/40 ring-offset-1 ring-offset-background' : ''}`}
                  style={{ backgroundColor: c.value }}
                />
              ))}
            </div>
          </SortableOption>
        ))}
        <button
          type="button"
          onClick={addOption}
          className="flex items-center justify-center gap-1.5 rounded-md border border-dashed border-border px-2.5 py-2 text-xs font-medium text-muted-foreground transition-colors hover:border-primary/40 hover:bg-secondary hover:text-foreground"
        >
          <Plus className="h-3.5 w-3.5" aria-hidden="true" />
          {t('customization.editor.wertelisten.addOption')}
        </button>
      </div>
        </SortableContext>
      </DndContext>

      {/* Entfernt — deleted options with a staged record migration; restorable */}
      {Object.keys(setMig).length > 0 && (
        <div className="mx-3 mb-2.5 rounded-lg border border-dashed border-amber-500/30 bg-amber-500/5 px-3 py-2.5">
          <p className="mb-1.5 text-[11px] font-medium uppercase tracking-wide text-amber-600 dark:text-amber-400">
            {t('customization.editor.wertelisten.removedTitle')}
          </p>
          <div className="flex flex-col gap-1.5">
            {editable.options
              .filter((o) => setMig[o.id])
              .map((o) => (
                <div key={o.id} className="flex items-center gap-2">
                  <span className="min-w-0 flex-1 truncate text-xs text-muted-foreground">
                    <span className="line-through">{o.label}</span>
                    <span className="mx-1">→</span>
                    <span className="text-foreground">{labelOf(setMig[o.id])}</span>
                  </span>
                  <button
                    type="button"
                    onClick={() => restoreRemoved(o.id)}
                    className="shrink-0 rounded px-1.5 py-0.5 text-[11px] font-medium text-primary transition-colors hover:bg-primary/10"
                  >
                    {t('customization.editor.felder.restore')}
                  </button>
                </div>
              ))}
          </div>
        </div>
      )}

      {/* In-panel preview (chips) — the module doesn't render the resolver yet. */}
      <div className="border-t px-3 py-2.5">
        <p className="mb-1.5 text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
          {t('customization.editor.wertelisten.preview')}
        </p>
        <div className="flex flex-wrap gap-1.5">
          {editable.options
            .filter((o) => o.active)
            .map((o) => (
              <span
                key={o.id}
                className="inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-[11px]"
              >
                <span className="h-2 w-2 rounded-full" style={{ backgroundColor: o.color ?? 'transparent' }} aria-hidden="true" />
                {o.label}
              </span>
            ))}
        </div>
      </div>
    </div>
  )
}

export function WertelistenPanel({ moduleKey }: { moduleKey: string }): React.ReactElement {
  const { t } = useTranslation()
  const module = getEditorModule(moduleKey)
  const predefinedIds = module?.valueSetIds ?? []
  const { valueSets: draftSets, setDraftValueSet } = useDraftConfig()
  // Lists the user created for this module: the ones staged in the current draft
  // plus the ones already deployed. Without the second half a self-created list
  // vanished from the panel the moment it went live — it was still in use in the
  // module, but only reachable through the field that bound it.
  const newIds = [
    ...new Set([...Object.keys(draftSets), ...listTenantValueSetsForModule(moduleKey)]),
  ].filter((id) => !predefinedIds.includes(id))

  const createValueSet = (): void => {
    const stamp = Date.now().toString(36)
    const rand = Math.random().toString(36).slice(2, 6)
    const setId = `vs-${stamp}-${rand}`
    setDraftValueSet(setId, {
      id: setId,
      name: t('customization.editor.wertelisten.newSetName'),
      moduleKey,
      options: [
        { id: `opt-${stamp}`, label: t('customization.editor.wertelisten.newOption'), color: SWATCHES[1].value, order: 0, active: true },
      ],
    })
  }

  return (
    <div className="flex flex-1 flex-col gap-3 overflow-y-auto px-4 py-3">
      {predefinedIds.map((id) => (
        <ValueSetEditor key={id} id={id} predefined />
      ))}
      {newIds.map((id) => (
        <ValueSetEditor key={id} id={id} predefined={false} />
      ))}
      <button
        type="button"
        onClick={createValueSet}
        className="flex items-center justify-center gap-1.5 rounded-lg border border-dashed border-border px-3 py-2.5 text-xs font-medium text-muted-foreground transition-colors hover:border-primary/40 hover:bg-secondary hover:text-foreground"
      >
        <Plus className="h-3.5 w-3.5" aria-hidden="true" />
        {t('customization.editor.wertelisten.addSet')}
      </button>
    </div>
  )
}

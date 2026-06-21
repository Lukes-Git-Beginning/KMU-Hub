/**
 * DocumentBlockEditor — the shared block-authoring engine. Controlled: it takes
 * `rows` + `onChange` and a `registry` of block types. Everything module-specific
 * (which blocks exist, how they edit/render) lives in the registry; the engine
 * owns the document mechanics: insert menu, drag-reorder, columns, delete.
 *
 * Extracted from the berichte block editor so berichte, wiki and future document
 * surfaces share one engine.
 */
import { useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Columns2,
  Columns3,
  GripVertical,
  MoreVertical,
  Plus,
  Square,
  Trash2,
  X,
  type LucideIcon,
} from 'lucide-react'
import {
  DndContext,
  PointerSensor,
  closestCenter,
  useSensor,
  useSensors,
  type DragEndEvent,
} from '@dnd-kit/core'
import {
  SortableContext,
  arrayMove,
  useSortable,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { docUid, emptyDocColumn, type DocBlockBase, type DocColumn, type DocRow } from './types'
import type { BlockRegistry } from './block-registry'

/** A module-supplied insert that adds a whole pre-shaped row (e.g. a KPI row). */
export interface QuickInsert {
  id: string
  labelKey: string
  icon: LucideIcon
  makeRow: () => DocRow
}

/** Width presets per column count (flex weights, carried into the read mode). */
const WIDTH_PRESETS: Record<number, { label: string; widths: number[] }[]> = {
  2: [
    { label: '50 / 50', widths: [1, 1] },
    { label: '60 / 40', widths: [3, 2] },
    { label: '40 / 60', widths: [2, 3] },
  ],
  3: [{ label: '33 / 33 / 33', widths: [1, 1, 1] }],
}

export interface DocumentBlockEditorProps {
  rows: DocRow[]
  onChange: (rows: DocRow[]) => void
  registry: BlockRegistry
  /** Extra row inserts shown in the layout section (e.g. berichte KPI row). */
  quickInserts?: QuickInsert[]
  /** Show the multi-column layout options. Default true. */
  enableColumns?: boolean
  /** Constrain the editor width. Default 'max-w-3xl'. */
  widthClass?: string
}

export function DocumentBlockEditor({
  rows,
  onChange,
  registry,
  quickInserts = [],
  enableColumns = true,
  widthClass = 'max-w-3xl',
}: DocumentBlockEditorProps) {
  const { t } = useTranslation()
  const [pickerOpen, setPickerOpen] = useState(false)
  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 5 } }))
  const insertable = Object.values(registry).filter((d) => d.insertable !== false)

  function makeBlock(type: string): DocBlockBase {
    return registry[type].makeDefault()
  }

  function patchBlock(blockId: string, patch: Partial<DocBlockBase>) {
    onChange(
      rows.map((row) => ({
        ...row,
        columns: row.columns.map((col) => ({
          ...col,
          blocks: col.blocks.map((b) => (b.id === blockId ? { ...b, ...patch } : b)),
        })),
      })),
    )
  }

  function deleteBlock(blockId: string) {
    onChange(
      rows
        .map((row) => ({
          ...row,
          columns: row.columns.map((col) => ({
            ...col,
            blocks: col.blocks.filter((b) => b.id !== blockId),
          })),
        }))
        .filter((row) => row.columns.some((col) => col.blocks.length > 0)),
    )
  }

  function addBlock(type: string) {
    onChange([...rows, { id: docUid('row'), columns: [{ ...emptyDocColumn(), blocks: [makeBlock(type)] }] }])
    setPickerOpen(false)
  }

  function addColumnRow(count: number) {
    onChange([...rows, { id: docUid('row'), columns: Array.from({ length: count }, emptyDocColumn) }])
    setPickerOpen(false)
  }

  function addQuickRow(qi: QuickInsert) {
    onChange([...rows, qi.makeRow()])
    setPickerOpen(false)
  }

  function addBlockToColumn(rowId: string, colId: string, type: string) {
    onChange(
      rows.map((row) =>
        row.id !== rowId
          ? row
          : {
              ...row,
              columns: row.columns.map((col) =>
                col.id !== colId ? col : { ...col, blocks: [...col.blocks, makeBlock(type)] },
              ),
            },
      ),
    )
  }

  function setColumnCount(rowId: string, count: number) {
    onChange(
      rows.map((row) => {
        if (row.id !== rowId || row.columns.length === count) return row
        const next: DocColumn[] = []
        for (let i = 0; i < count; i += 1) next.push(row.columns[i] ?? emptyDocColumn())
        if (row.columns.length > count) {
          const merged = row.columns.slice(count).flatMap((c) => c.blocks)
          next[count - 1] = { ...next[count - 1], blocks: [...next[count - 1].blocks, ...merged] }
        }
        return { ...row, columns: next.map((c) => ({ ...c, width: 1 })) }
      }),
    )
  }

  function setRowWidths(rowId: string, widths: number[]) {
    onChange(
      rows.map((row) =>
        row.id !== rowId
          ? row
          : { ...row, columns: row.columns.map((c, i) => ({ ...c, width: widths[i] ?? 1 })) },
      ),
    )
  }

  function deleteRow(rowId: string) {
    onChange(rows.filter((row) => row.id !== rowId))
  }

  function handleDragEnd(event: DragEndEvent) {
    const { active, over } = event
    if (!over || active.id === over.id) return
    const from = rows.findIndex((r) => r.id === active.id)
    const to = rows.findIndex((r) => r.id === over.id)
    if (from === -1 || to === -1) return
    onChange(arrayMove(rows, from, to))
  }

  const hasLayoutSection = enableColumns || quickInserts.length > 0

  return (
    <div className={`mx-auto ${widthClass} space-y-4 pl-6`}>
      <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
        <SortableContext items={rows.map((r) => r.id)} strategy={verticalListSortingStrategy}>
          {rows.map((row) => (
            <SortableRow
              key={row.id}
              id={row.id}
              menu={
                <RowMenu
                  columnCount={row.columns.length}
                  widths={row.columns.map((c) => c.width ?? 1)}
                  enableColumns={enableColumns}
                  onSetColumns={(n) => setColumnCount(row.id, n)}
                  onSetWidths={(w) => setRowWidths(row.id, w)}
                  onDelete={() => deleteRow(row.id)}
                />
              }
            >
              <div className={row.columns.length > 1 ? 'flex gap-3' : undefined}>
                {row.columns.map((col) => (
                  <div key={col.id} className="min-w-0 space-y-4" style={{ flex: col.width ?? 1 }}>
                    {col.blocks.map((block) => {
                      const def = registry[block.type]
                      if (!def) return null
                      const Edit = def.Edit
                      return (
                        <div key={block.id} className="group/block relative">
                          <Edit block={block} onPatch={(p) => patchBlock(block.id, p)} />
                          <button
                            type="button"
                            onClick={() => deleteBlock(block.id)}
                            className="absolute -right-2 -top-2 z-20 flex h-6 w-6 items-center justify-center rounded-full border border-border bg-card text-muted-foreground opacity-0 shadow-sm transition-opacity hover:text-destructive group-hover/block:opacity-100"
                            aria-label={t('document.editor.deleteBlock', { defaultValue: 'Block löschen' })}
                          >
                            <Trash2 className="h-3 w-3" />
                          </button>
                        </div>
                      )
                    })}
                    {row.columns.length > 1 && (
                      <ColumnInsert
                        defs={insertable.map((d) => ({ type: d.type, icon: d.icon, labelKey: d.labelKey }))}
                        onAdd={(type) => addBlockToColumn(row.id, col.id, type)}
                        empty={col.blocks.length === 0}
                      />
                    )}
                  </div>
                ))}
              </div>
            </SortableRow>
          ))}
        </SortableContext>
      </DndContext>

      {pickerOpen ? (
        <div className="space-y-3 rounded-xl border border-border bg-card p-3">
          <div className="flex items-center justify-between">
            <span className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
              {t('document.editor.blockLabel', { defaultValue: 'Block' })}
            </span>
            <button
              type="button"
              onClick={() => setPickerOpen(false)}
              className="flex h-7 w-7 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-secondary"
              aria-label={t('document.editor.cancelInsert', { defaultValue: 'Abbrechen' })}
            >
              <X className="h-4 w-4" />
            </button>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            {insertable.map((def) => {
              const Icon = def.icon
              return (
                <button
                  key={def.type}
                  type="button"
                  onClick={() => addBlock(def.type)}
                  className="flex items-center gap-1.5 rounded-lg border border-border px-2.5 py-1.5 text-xs text-foreground transition-colors hover:border-primary/40 hover:bg-secondary"
                >
                  <Icon className="h-3.5 w-3.5 text-muted-foreground" />
                  {t(def.labelKey)}
                </button>
              )
            })}
          </div>
          {hasLayoutSection && (
            <div className="border-t border-border-muted pt-3">
              <span className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
                {t('document.editor.layout', { defaultValue: 'Layout' })}
              </span>
              <div className="mt-2 flex flex-wrap items-center gap-2">
                {enableColumns && (
                  <>
                    <button
                      type="button"
                      onClick={() => addColumnRow(2)}
                      className="flex items-center gap-1.5 rounded-lg border border-border px-2.5 py-1.5 text-xs text-foreground transition-colors hover:border-primary/40 hover:bg-secondary"
                    >
                      <Columns2 className="h-3.5 w-3.5 text-muted-foreground" />
                      {t('document.editor.columnCount', { defaultValue: '{count} Spalten', count: 2 })}
                    </button>
                    <button
                      type="button"
                      onClick={() => addColumnRow(3)}
                      className="flex items-center gap-1.5 rounded-lg border border-border px-2.5 py-1.5 text-xs text-foreground transition-colors hover:border-primary/40 hover:bg-secondary"
                    >
                      <Columns3 className="h-3.5 w-3.5 text-muted-foreground" />
                      {t('document.editor.columnCount', { defaultValue: '{count} Spalten', count: 3 })}
                    </button>
                  </>
                )}
                {quickInserts.map((qi) => {
                  const Icon = qi.icon
                  return (
                    <button
                      key={qi.id}
                      type="button"
                      onClick={() => addQuickRow(qi)}
                      className="flex items-center gap-1.5 rounded-lg border border-border px-2.5 py-1.5 text-xs text-foreground transition-colors hover:border-primary/40 hover:bg-secondary"
                    >
                      <Icon className="h-3.5 w-3.5 text-muted-foreground" />
                      {t(qi.labelKey)}
                    </button>
                  )
                })}
              </div>
            </div>
          )}
        </div>
      ) : (
        <button
          type="button"
          onClick={() => setPickerOpen(true)}
          className="flex w-full items-center justify-center gap-2 rounded-xl border border-dashed border-border py-2.5 text-sm text-muted-foreground transition-colors hover:border-primary/40 hover:text-foreground"
        >
          <Plus className="h-4 w-4" />
          {t('document.editor.addBlock', { defaultValue: 'Block einfügen' })}
        </button>
      )}
    </div>
  )
}

/** Row wrapper with a hover-only drag handle (left) and options menu (right). */
function SortableRow({ id, menu, children }: { id: string; menu: ReactNode; children: ReactNode }) {
  const { t } = useTranslation()
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({ id })
  return (
    <div
      ref={setNodeRef}
      style={{ transform: CSS.Transform.toString(transform), transition, opacity: isDragging ? 0.4 : 1 }}
      className="group/row relative"
    >
      <button
        type="button"
        {...attributes}
        {...listeners}
        className="absolute -left-7 top-1.5 flex h-6 w-6 cursor-grab items-center justify-center rounded text-muted-foreground opacity-0 transition-opacity hover:bg-secondary active:cursor-grabbing group-hover/row:opacity-100"
        aria-label={t('document.editor.dragRow', { defaultValue: 'Zeile verschieben' })}
      >
        <GripVertical className="h-4 w-4" />
      </button>
      <div className="absolute -right-9 top-1.5 opacity-0 transition-opacity group-hover/row:opacity-100">
        {menu}
      </div>
      {children}
    </div>
  )
}

/** Row options popover: column count, width presets, delete. */
function RowMenu({
  columnCount,
  widths,
  enableColumns,
  onSetColumns,
  onSetWidths,
  onDelete,
}: {
  columnCount: number
  widths: number[]
  enableColumns: boolean
  onSetColumns: (count: number) => void
  onSetWidths: (widths: number[]) => void
  onDelete: () => void
}) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const presets = WIDTH_PRESETS[columnCount] ?? []
  const currentKey = JSON.stringify(widths)

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <button
          type="button"
          className="flex h-6 w-6 items-center justify-center rounded text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
          aria-label={t('document.editor.rowOptions', { defaultValue: 'Zeilenoptionen' })}
        >
          <MoreVertical className="h-4 w-4" />
        </button>
      </PopoverTrigger>
      <PopoverContent className="w-52 p-2" align="end" sideOffset={4}>
        {enableColumns && (
          <>
            <p className="px-1 pb-1.5 text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
              {t('document.editor.columnsLabel', { defaultValue: 'Spalten' })}
            </p>
            <div className="flex gap-1.5">
              {([1, 2, 3] as const).map((n) => {
                const Icon = n === 1 ? Square : n === 2 ? Columns2 : Columns3
                return (
                  <button
                    key={n}
                    type="button"
                    onClick={() => onSetColumns(n)}
                    className={`flex flex-1 items-center justify-center gap-1 rounded-lg border py-1.5 text-xs transition-colors ${
                      columnCount === n
                        ? 'border-primary/40 bg-primary-light text-primary'
                        : 'border-border text-muted-foreground hover:bg-secondary'
                    }`}
                    aria-label={t('document.editor.columnCount', { defaultValue: '{count} Spalten', count: n })}
                  >
                    <Icon className="h-3.5 w-3.5" />
                    {n}
                  </button>
                )
              })}
            </div>
            {presets.length > 0 && (
              <>
                <p className="px-1 pb-1.5 pt-3 text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
                  {t('document.editor.widthLabel', { defaultValue: 'Breite' })}
                </p>
                <div className="space-y-1">
                  {presets.map((preset) => {
                    const active = JSON.stringify(preset.widths) === currentKey
                    return (
                      <button
                        key={preset.label}
                        type="button"
                        onClick={() => onSetWidths(preset.widths)}
                        className={`flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-xs transition-colors ${
                          active ? 'bg-primary-light text-primary' : 'text-foreground hover:bg-secondary'
                        }`}
                      >
                        <span className="flex h-3.5 w-6 gap-0.5">
                          {preset.widths.map((w, i) => (
                            <span
                              key={i}
                              className={`rounded-sm ${active ? 'bg-primary/60' : 'bg-muted-foreground/40'}`}
                              style={{ flex: w }}
                            />
                          ))}
                        </span>
                        {preset.label}
                      </button>
                    )
                  })}
                </div>
              </>
            )}
          </>
        )}

        <div className={`${enableColumns ? 'mt-2 border-t border-border-muted pt-2' : ''}`}>
          <button
            type="button"
            onClick={() => {
              setOpen(false)
              onDelete()
            }}
            className="flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-xs text-destructive transition-colors hover:bg-destructive/10"
          >
            <Trash2 className="h-3.5 w-3.5" />
            {t('document.editor.deleteRow', { defaultValue: 'Zeile löschen' })}
          </button>
        </div>
      </PopoverContent>
    </Popover>
  )
}

/** Compact per-column block inserter for multi-column rows. */
function ColumnInsert({
  defs,
  onAdd,
  empty,
}: {
  defs: { type: string; icon: LucideIcon; labelKey: string }[]
  onAdd: (type: string) => void
  empty: boolean
}) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <button
          type="button"
          className={`flex w-full items-center justify-center gap-1.5 rounded-lg border border-dashed border-border text-xs text-muted-foreground transition-colors hover:border-primary/40 hover:text-foreground ${
            empty ? 'py-3' : 'py-1.5'
          }`}
        >
          <Plus className="h-3.5 w-3.5" />
          {t('document.editor.blockLabel', { defaultValue: 'Block' })}
        </button>
      </PopoverTrigger>
      <PopoverContent className="w-44 p-1.5" align="start" sideOffset={4}>
        <div className="flex flex-col gap-0.5">
          {defs.map((def) => {
            const Icon = def.icon
            return (
              <button
                key={def.type}
                type="button"
                onClick={() => {
                  setOpen(false)
                  onAdd(def.type)
                }}
                className="flex items-center gap-2 rounded-lg px-2 py-1.5 text-xs text-foreground transition-colors hover:bg-secondary"
              >
                <Icon className="h-3.5 w-3.5 text-muted-foreground" />
                {t(def.labelKey)}
              </button>
            )
          })}
        </div>
      </PopoverContent>
    </Popover>
  )
}

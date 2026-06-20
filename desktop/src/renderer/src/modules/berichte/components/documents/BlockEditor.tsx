import { useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Columns2,
  Columns3,
  GripVertical,
  Heading1,
  Heading2,
  MoreVertical,
  Plus,
  Square,
  Trash2,
  X,
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
import { RichTextEditor } from '@/components/shared/RichTextEditor'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import type {
  ReportBlock,
  ReportBlockType,
  ReportDocColumn,
  ReportRow,
} from '@/api/berichte-types'
import { BLOCK_META } from './doc-utils'

/** Block id helper — unique per inserted block. */
const uid = (p: string): string => `${p}-${crypto.randomUUID().slice(0, 8)}`

/** Simple block types insertable in the core editor (chart/table/kpi = R-1b). */
const INSERTABLE: ReportBlockType[] = ['heading', 'text', 'bullet', 'divider', 'pagebreak']

/** Width presets per column count (flex weights, carried into the read mode). */
const WIDTH_PRESETS: Record<number, { label: string; widths: number[] }[]> = {
  2: [
    { label: '50 / 50', widths: [1, 1] },
    { label: '60 / 40', widths: [3, 2] },
    { label: '40 / 60', widths: [2, 3] },
  ],
  3: [{ label: '33 / 33 / 33', widths: [1, 1, 1] }],
}

function makeBlock(type: ReportBlockType): ReportBlock {
  switch (type) {
    case 'heading':
      return { id: uid('b'), type: 'heading', level: 1, text: '' }
    case 'text':
      return { id: uid('b'), type: 'text', html: '' }
    case 'bullet':
      return { id: uid('b'), type: 'bullet', items: [''] }
    case 'divider':
      return { id: uid('b'), type: 'divider' }
    case 'pagebreak':
      return { id: uid('b'), type: 'pagebreak' }
    default:
      return { id: uid('b'), type: 'text', html: '' }
  }
}

function emptyColumn(): ReportDocColumn {
  return { id: uid('col'), blocks: [], width: 1 }
}

interface BlockEditorProps {
  rows: ReportRow[]
  onChange: (rows: ReportRow[]) => void
}

/** Editable block document: rows -> columns -> blocks, drag-reorder + insert. */
export function BlockEditor({ rows, onChange }: BlockEditorProps) {
  const { t } = useTranslation()
  const [pickerOpen, setPickerOpen] = useState(false)
  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 5 } }))

  function patchBlock(blockId: string, patch: Partial<ReportBlock>) {
    onChange(
      rows.map((row) => ({
        ...row,
        columns: row.columns.map((col) => ({
          ...col,
          blocks: col.blocks.map((b) =>
            b.id === blockId ? ({ ...b, ...patch } as ReportBlock) : b,
          ),
        })),
      })),
    )
  }

  function deleteBlock(blockId: string) {
    // Keep empty columns inside multi-column rows so the layout persists; drop a
    // row only once every column is empty.
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

  /** Append a new single-column row with one block. */
  function addBlock(type: ReportBlockType) {
    onChange([
      ...rows,
      { id: uid('row'), columns: [{ ...emptyColumn(), blocks: [makeBlock(type)] }] },
    ])
    setPickerOpen(false)
  }

  /** Append an empty multi-column row to fill column by column. */
  function addColumnRow(count: number) {
    onChange([
      ...rows,
      { id: uid('row'), columns: Array.from({ length: count }, emptyColumn) },
    ])
    setPickerOpen(false)
  }

  /** Insert a block into a specific column of a specific row. */
  function addBlockToColumn(rowId: string, colId: string, type: ReportBlockType) {
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

  /** Change a row's column count, preserving content (excess columns merge left). */
  function setColumnCount(rowId: string, count: number) {
    onChange(
      rows.map((row) => {
        if (row.id !== rowId || row.columns.length === count) return row
        const next: ReportDocColumn[] = []
        for (let i = 0; i < count; i += 1) next.push(row.columns[i] ?? emptyColumn())
        if (row.columns.length > count) {
          const merged = row.columns.slice(count).flatMap((c) => c.blocks)
          next[count - 1] = { ...next[count - 1], blocks: [...next[count - 1].blocks, ...merged] }
        }
        // Reset to even widths; the user then picks a preset from the menu.
        return { ...row, columns: next.map((c) => ({ ...c, width: 1 })) }
      }),
    )
  }

  /** Apply a width preset to a row (flex weights per column). */
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

  return (
    <div className="mx-auto max-w-3xl space-y-4 pl-6">
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
                  onSetColumns={(n) => setColumnCount(row.id, n)}
                  onSetWidths={(w) => setRowWidths(row.id, w)}
                  onDelete={() => deleteRow(row.id)}
                />
              }
            >
              <div className={row.columns.length > 1 ? 'flex gap-3' : undefined}>
                {row.columns.map((col) => (
                  <div
                    key={col.id}
                    className="min-w-0 space-y-4"
                    style={{ flex: col.width ?? 1 }}
                  >
                    {col.blocks.map((block) => (
                      <div key={block.id} className="group/block relative">
                        <BlockEdit block={block} onPatch={(p) => patchBlock(block.id, p)} />
                        <button
                          type="button"
                          onClick={() => deleteBlock(block.id)}
                          className="absolute -right-2 -top-2 flex h-6 w-6 items-center justify-center rounded-full border border-border bg-card text-muted-foreground opacity-0 shadow-sm transition-opacity hover:text-destructive group-hover/block:opacity-100"
                          aria-label={t('berichte.docs.deleteBlock')}
                        >
                          <Trash2 className="h-3 w-3" />
                        </button>
                      </div>
                    ))}
                    {/* Per-column insert (only shown for multi-column rows; single
                        column rows use the document-level insert below). */}
                    {row.columns.length > 1 && (
                      <ColumnInsert
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

      {/* Insert a new row */}
      {pickerOpen ? (
        <div className="space-y-3 rounded-xl border border-border bg-card p-3">
          <div className="flex items-center justify-between">
            <span className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
              {t('berichte.docs.blockLabel')}
            </span>
            <button
              type="button"
              onClick={() => setPickerOpen(false)}
              className="flex h-7 w-7 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-secondary"
              aria-label={t('berichte.docs.cancelInsert')}
            >
              <X className="h-4 w-4" />
            </button>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            {INSERTABLE.map((type) => {
              const meta = BLOCK_META[type]
              const Icon = meta.icon
              return (
                <button
                  key={type}
                  type="button"
                  onClick={() => addBlock(type)}
                  className="flex items-center gap-1.5 rounded-lg border border-border px-2.5 py-1.5 text-xs text-foreground transition-colors hover:border-primary/40 hover:bg-secondary"
                >
                  <Icon className="h-3.5 w-3.5 text-muted-foreground" />
                  {t(meta.labelKey)}
                </button>
              )
            })}
          </div>
          <div className="border-t border-border-muted pt-3">
            <span className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
              {t('berichte.docs.layout')}
            </span>
            <div className="mt-2 flex flex-wrap items-center gap-2">
              <button
                type="button"
                onClick={() => addColumnRow(2)}
                className="flex items-center gap-1.5 rounded-lg border border-border px-2.5 py-1.5 text-xs text-foreground transition-colors hover:border-primary/40 hover:bg-secondary"
              >
                <Columns2 className="h-3.5 w-3.5 text-muted-foreground" />
                {t('berichte.docs.columnCount', { count: 2 })}
              </button>
              <button
                type="button"
                onClick={() => addColumnRow(3)}
                className="flex items-center gap-1.5 rounded-lg border border-border px-2.5 py-1.5 text-xs text-foreground transition-colors hover:border-primary/40 hover:bg-secondary"
              >
                <Columns3 className="h-3.5 w-3.5 text-muted-foreground" />
                {t('berichte.docs.columnCount', { count: 3 })}
              </button>
            </div>
          </div>
        </div>
      ) : (
        <button
          type="button"
          onClick={() => setPickerOpen(true)}
          className="flex w-full items-center justify-center gap-2 rounded-xl border border-dashed border-border py-2.5 text-sm text-muted-foreground transition-colors hover:border-primary/40 hover:text-foreground"
        >
          <Plus className="h-4 w-4" />
          {t('berichte.docs.addBlock')}
        </button>
      )}
    </div>
  )
}

/** Row wrapper with a hover-only drag handle (left) and options menu (right). */
function SortableRow({
  id,
  menu,
  children,
}: {
  id: string
  menu: ReactNode
  children: ReactNode
}) {
  const { t } = useTranslation()
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id,
  })
  return (
    <div
      ref={setNodeRef}
      style={{
        transform: CSS.Transform.toString(transform),
        transition,
        opacity: isDragging ? 0.4 : 1,
      }}
      className="group/row relative"
    >
      <button
        type="button"
        {...attributes}
        {...listeners}
        className="absolute -left-7 top-1.5 flex h-6 w-6 cursor-grab items-center justify-center rounded text-muted-foreground opacity-0 transition-opacity hover:bg-secondary active:cursor-grabbing group-hover/row:opacity-100"
        aria-label={t('berichte.docs.dragRow')}
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
  onSetColumns,
  onSetWidths,
  onDelete,
}: {
  columnCount: number
  widths: number[]
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
          aria-label={t('berichte.docs.rowOptions')}
        >
          <MoreVertical className="h-4 w-4" />
        </button>
      </PopoverTrigger>
      <PopoverContent className="w-52 p-2" align="end" sideOffset={4}>
        <p className="px-1 pb-1.5 text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
          {t('berichte.docs.columnsLabel')}
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
                aria-label={t('berichte.docs.columnCount', { count: n })}
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
              {t('berichte.docs.widthLabel')}
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
                      active
                        ? 'bg-primary-light text-primary'
                        : 'text-foreground hover:bg-secondary'
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

        <div className="mt-2 border-t border-border-muted pt-2">
          <button
            type="button"
            onClick={() => {
              setOpen(false)
              onDelete()
            }}
            className="flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-xs text-destructive transition-colors hover:bg-destructive/10"
          >
            <Trash2 className="h-3.5 w-3.5" />
            {t('berichte.docs.deleteRow')}
          </button>
        </div>
      </PopoverContent>
    </Popover>
  )
}

/** Compact per-column block inserter for multi-column rows. */
function ColumnInsert({
  onAdd,
  empty,
}: {
  onAdd: (type: ReportBlockType) => void
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
          {t('berichte.docs.blockLabel')}
        </button>
      </PopoverTrigger>
      <PopoverContent className="w-44 p-1.5" align="start" sideOffset={4}>
        <div className="flex flex-col gap-0.5">
          {INSERTABLE.map((type) => {
            const meta = BLOCK_META[type]
            const Icon = meta.icon
            return (
              <button
                key={type}
                type="button"
                onClick={() => {
                  setOpen(false)
                  onAdd(type)
                }}
                className="flex items-center gap-2 rounded-lg px-2 py-1.5 text-xs text-foreground transition-colors hover:bg-secondary"
              >
                <Icon className="h-3.5 w-3.5 text-muted-foreground" />
                {t(meta.labelKey)}
              </button>
            )
          })}
        </div>
      </PopoverContent>
    </Popover>
  )
}

/* ---------------------------- per-block editors --------------------------- */

function BlockEdit({
  block,
  onPatch,
}: {
  block: ReportBlock
  onPatch: (patch: Partial<ReportBlock>) => void
}) {
  const { t } = useTranslation()
  const inputCls =
    'w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring'

  switch (block.type) {
    case 'cover':
      return (
        <div className="space-y-2 rounded-xl border border-border bg-card p-4">
          <input
            value={block.title}
            onChange={(e) => onPatch({ title: e.target.value })}
            placeholder={t('berichte.docs.ph.coverTitle')}
            className={`${inputCls} text-base font-semibold`}
          />
          <input
            value={block.subtitle ?? ''}
            onChange={(e) => onPatch({ subtitle: e.target.value })}
            placeholder={t('berichte.docs.ph.coverSubtitle')}
            className={inputCls}
          />
          <input
            value={block.author ?? ''}
            onChange={(e) => onPatch({ author: e.target.value })}
            placeholder={t('berichte.docs.ph.author')}
            className={inputCls}
          />
        </div>
      )

    case 'heading':
      return (
        <div className="flex items-center gap-2">
          <div className="flex overflow-hidden rounded-lg border border-border">
            {([1, 2] as const).map((lvl) => {
              const Icon = lvl === 1 ? Heading1 : Heading2
              return (
                <button
                  key={lvl}
                  type="button"
                  onClick={() => onPatch({ level: lvl })}
                  className={`p-2 ${block.level === lvl ? 'bg-primary-light text-primary' : 'text-muted-foreground hover:bg-secondary'}`}
                  aria-label={`H${lvl}`}
                >
                  <Icon className="h-4 w-4" />
                </button>
              )
            })}
          </div>
          <input
            value={block.text}
            onChange={(e) => onPatch({ text: e.target.value })}
            placeholder={t('berichte.docs.ph.heading')}
            className={`${inputCls} ${block.level === 1 ? 'text-lg font-semibold' : 'text-base font-medium'}`}
          />
        </div>
      )

    case 'text':
      return (
        <RichTextEditor
          content={block.html}
          onChange={(html) => onPatch({ html })}
          compact
          placeholder={t('berichte.docs.ph.text')}
        />
      )

    case 'bullet':
      return (
        <div className="space-y-1.5 rounded-xl border border-border bg-card p-3">
          {block.items.map((item, i) => (
            <div key={i} className="flex items-center gap-2">
              <span className="text-muted-foreground">•</span>
              <input
                value={item}
                onChange={(e) => {
                  const items = [...block.items]
                  items[i] = e.target.value
                  onPatch({ items })
                }}
                placeholder={t('berichte.docs.ph.bullet')}
                className="flex-1 rounded-md border border-transparent bg-transparent px-2 py-1 text-sm text-foreground hover:border-border focus:border-border focus:outline-none"
              />
              <button
                type="button"
                onClick={() => onPatch({ items: block.items.filter((_, j) => j !== i) })}
                className="flex h-7 w-7 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-secondary"
                aria-label={t('berichte.docs.removeItem')}
              >
                <Trash2 className="h-3.5 w-3.5" />
              </button>
            </div>
          ))}
          <button
            type="button"
            onClick={() => onPatch({ items: [...block.items, ''] })}
            className="flex items-center gap-1.5 px-2 text-xs text-muted-foreground transition-colors hover:text-foreground"
          >
            <Plus className="h-3.5 w-3.5" />
            {t('berichte.docs.addItem')}
          </button>
        </div>
      )

    case 'divider':
      return <hr className="border-border" />

    case 'pagebreak':
      return (
        <div className="flex items-center gap-2 text-[10px] uppercase tracking-wider text-muted-foreground/60">
          <span className="h-px flex-1 bg-border" />
          {t('berichte.docs.blockType.pagebreak')}
          <span className="h-px flex-1 bg-border" />
        </div>
      )

    // chart / table / kpi / image — placeholder until R-1b (source picker).
    default: {
      const meta = BLOCK_META[block.type]
      const Icon = meta.icon
      const caption =
        'caption' in block ? block.caption : 'label' in block ? block.label : undefined
      return (
        <div className="flex items-center gap-2.5 rounded-xl border border-dashed border-border bg-secondary/20 p-3">
          <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-secondary text-muted-foreground">
            <Icon className="h-4 w-4" />
          </div>
          <div className="min-w-0">
            <p className="text-xs font-medium text-foreground">{t(meta.labelKey)}</p>
            <p className="truncate text-[11px] text-muted-foreground">
              {caption || t('berichte.docs.blockSoon')}
            </p>
          </div>
        </div>
      )
    }
  }
}

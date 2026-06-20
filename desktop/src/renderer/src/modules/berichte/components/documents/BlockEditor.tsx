import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Heading1, Heading2, Plus, Trash2, X } from 'lucide-react'
import { RichTextEditor } from '@/components/shared/RichTextEditor'
import type { ReportBlock, ReportBlockType, ReportRow } from '@/api/berichte-types'
import { BLOCK_META } from './doc-utils'

/** Block id helper — unique per inserted block. */
const uid = (p: string): string => `${p}-${crypto.randomUUID().slice(0, 8)}`

/** Simple block types insertable in the core editor (chart/table/kpi = R-1b). */
const INSERTABLE: ReportBlockType[] = ['heading', 'text', 'bullet', 'divider', 'pagebreak']

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

interface BlockEditorProps {
  rows: ReportRow[]
  onChange: (rows: ReportRow[]) => void
}

/** Editable block document: renders rows -> columns -> editable blocks + insert. */
export function BlockEditor({ rows, onChange }: BlockEditorProps) {
  const { t } = useTranslation()
  const [pickerOpen, setPickerOpen] = useState(false)

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

  function addBlock(type: ReportBlockType) {
    onChange([
      ...rows,
      { id: uid('row'), columns: [{ id: uid('col'), blocks: [makeBlock(type)] }] },
    ])
    setPickerOpen(false)
  }

  return (
    <div className="mx-auto max-w-3xl space-y-4">
      {rows.map((row) => (
        <div key={row.id} className={row.columns.length > 1 ? 'flex gap-3' : undefined}>
          {row.columns.map((col) => (
            <div key={col.id} className="space-y-4" style={{ flex: col.width ?? 1 }}>
              {col.blocks.map((block) => (
                <BlockEdit
                  key={block.id}
                  block={block}
                  onPatch={(patch) => patchBlock(block.id, patch)}
                />
              ))}
            </div>
          ))}
        </div>
      ))}

      {/* Insert a new block */}
      {pickerOpen ? (
        <div className="flex flex-wrap items-center gap-2 rounded-xl border border-border bg-card p-3">
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
          <button
            type="button"
            onClick={() => setPickerOpen(false)}
            className="ml-auto flex h-7 w-7 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-secondary"
            aria-label={t('berichte.docs.cancelInsert')}
          >
            <X className="h-4 w-4" />
          </button>
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

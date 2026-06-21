/**
 * Core block implementations (edit + read view) shared across document surfaces.
 * Built once here, registered by any module via createCoreBlockDefs(). Styling
 * mirrors the berichte block editor/reader so existing documents look identical.
 */
import { useTranslation } from 'react-i18next'
import {
  AlertTriangle,
  AlignLeft,
  CheckCircle2,
  Heading,
  Heading1,
  Heading2,
  Image as ImageIcon,
  Info,
  Lightbulb,
  List,
  Minus,
  Plus,
  Trash2,
  type LucideIcon,
} from 'lucide-react'
import DOMPurify from 'dompurify'
import { RichTextEditor } from '@/components/shared/RichTextEditor'
import { docUid } from '../types'
import { defineBlock, type BlockEditProps, type BlockTypeDef, type BlockViewProps } from '../block-registry'
import type {
  BulletBlock,
  CalloutBlock,
  CalloutVariant,
  DividerBlock,
  HeadingBlock,
  ImageBlock,
  TextBlock,
} from './core-types'

/* ----------------------------- callout styling ---------------------------- */

const CALLOUT_VARIANTS: { variant: CalloutVariant; icon: LucideIcon; active: string }[] = [
  { variant: 'info', icon: Info, active: 'border-info/40 bg-info-light text-info' },
  { variant: 'success', icon: CheckCircle2, active: 'border-success/40 bg-success-light text-success' },
  { variant: 'warning', icon: AlertTriangle, active: 'border-warning/40 bg-warning-light text-warning' },
  { variant: 'recommendation', icon: Lightbulb, active: 'border-primary/40 bg-primary-light text-primary' },
]
const CALLOUT_BG: Record<CalloutVariant, string> = {
  info: 'border-info/30 bg-info-light',
  success: 'border-success/30 bg-success-light',
  warning: 'border-warning/30 bg-warning-light',
  recommendation: 'border-primary/30 bg-primary-light',
}
const CALLOUT_STYLE: Record<CalloutVariant, string> = {
  info: 'border-info/25 bg-info-light',
  success: 'border-success/25 bg-success-light',
  warning: 'border-warning/25 bg-warning-light',
  recommendation: 'border-primary/25 bg-primary-light',
}
const CALLOUT_ICON: Record<CalloutVariant, { icon: LucideIcon; tone: string }> = {
  info: { icon: Info, tone: 'text-info' },
  success: { icon: CheckCircle2, tone: 'text-success' },
  warning: { icon: AlertTriangle, tone: 'text-warning' },
  recommendation: { icon: Lightbulb, tone: 'text-primary' },
}

const inputCls =
  'w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring'

/* -------------------------------- edit views ------------------------------ */

function HeadingEdit({ block, onPatch }: BlockEditProps<HeadingBlock>) {
  const { t } = useTranslation()
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
        placeholder={t('document.block.heading.placeholder', { defaultValue: 'Überschrift' })}
        className={`${inputCls} ${block.level === 1 ? 'text-lg font-semibold' : 'text-base font-medium'}`}
      />
    </div>
  )
}

function TextEdit({ block, onPatch }: BlockEditProps<TextBlock>) {
  const { t } = useTranslation()
  return (
    <RichTextEditor
      content={block.html}
      onChange={(html) => onPatch({ html })}
      compact
      placeholder={t('document.block.text.placeholder', { defaultValue: 'Text schreiben…' })}
    />
  )
}

function BulletEdit({ block, onPatch }: BlockEditProps<BulletBlock>) {
  const { t } = useTranslation()
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
            placeholder={t('document.block.bullet.placeholder', { defaultValue: 'Listenpunkt' })}
            className="flex-1 rounded-md border border-transparent bg-transparent px-2 py-1 text-sm text-foreground hover:border-border focus:border-border focus:outline-none"
          />
          <button
            type="button"
            onClick={() => onPatch({ items: block.items.filter((_, j) => j !== i) })}
            className="flex h-7 w-7 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-secondary"
            aria-label={t('document.block.bullet.removeItem', { defaultValue: 'Punkt entfernen' })}
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
        {t('document.block.bullet.addItem', { defaultValue: 'Punkt hinzufügen' })}
      </button>
    </div>
  )
}

function CalloutEdit({ block, onPatch }: BlockEditProps<CalloutBlock>) {
  const { t } = useTranslation()
  return (
    <div className={`space-y-2 rounded-xl border p-3 ${CALLOUT_BG[block.variant]}`}>
      <div className="flex items-center gap-1.5">
        {CALLOUT_VARIANTS.map(({ variant, icon: Icon, active }) => (
          <button
            key={variant}
            type="button"
            onClick={() => onPatch({ variant })}
            className={`flex h-7 w-7 items-center justify-center rounded-lg border transition-colors ${
              block.variant === variant ? active : 'border-transparent text-muted-foreground hover:bg-card/60'
            }`}
            aria-label={t(`document.block.callout.variant.${variant}`, { defaultValue: variant })}
          >
            <Icon className="h-4 w-4" />
          </button>
        ))}
      </div>
      <input
        value={block.title ?? ''}
        onChange={(e) => onPatch({ title: e.target.value })}
        placeholder={t('document.block.callout.titlePlaceholder', { defaultValue: 'Titel (optional)' })}
        className="w-full rounded-md border border-transparent bg-card/50 px-2 py-1 text-sm font-semibold text-foreground hover:border-border focus:border-border focus:outline-none"
      />
      <RichTextEditor
        content={block.html}
        onChange={(html) => onPatch({ html })}
        compact
        placeholder={t('document.block.callout.textPlaceholder', { defaultValue: 'Hinweistext…' })}
      />
    </div>
  )
}

function DividerEdit(_props: BlockEditProps<DividerBlock>) {
  return <hr className="border-border" />
}

function ImageEdit({ block, onPatch }: BlockEditProps<ImageBlock>) {
  const { t } = useTranslation()
  const cls =
    'w-full rounded-md border border-border bg-card px-2 py-1.5 text-xs text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring'
  return (
    <div className="space-y-2 rounded-xl border border-border bg-card p-3">
      {block.url ? (
        <img src={block.url} alt={block.alt ?? ''} className="max-h-56 w-full rounded-lg object-contain" />
      ) : (
        <div className="flex h-28 flex-col items-center justify-center gap-1.5 rounded-lg bg-secondary/40 text-muted-foreground">
          <ImageIcon className="h-6 w-6 text-muted-foreground/50" />
          <span className="text-[11px]">
            {t('document.block.image.hint', { defaultValue: 'Bild-URL einfügen' })}
          </span>
        </div>
      )}
      <input
        value={block.url}
        onChange={(e) => onPatch({ url: e.target.value })}
        placeholder={t('document.block.image.urlPlaceholder', { defaultValue: 'Bild-URL' })}
        className={cls}
      />
      <div className="flex gap-2">
        <input
          value={block.alt ?? ''}
          onChange={(e) => onPatch({ alt: e.target.value })}
          placeholder={t('document.block.image.altPlaceholder', { defaultValue: 'Alternativtext' })}
          className={cls}
        />
        <input
          value={block.caption ?? ''}
          onChange={(e) => onPatch({ caption: e.target.value })}
          placeholder={t('document.block.image.captionPlaceholder', { defaultValue: 'Bildunterschrift' })}
          className={cls}
        />
      </div>
    </div>
  )
}

/* -------------------------------- read views ------------------------------ */

function HeadingView({ block }: BlockViewProps<HeadingBlock>) {
  return block.level === 1 ? (
    <h2 className="report-serif mt-2 text-[2rem] font-semibold leading-tight tracking-tight text-foreground">
      {block.text}
    </h2>
  ) : (
    <h3 className="text-lg font-semibold tracking-tight text-foreground">{block.text}</h3>
  )
}

function TextView({ block }: BlockViewProps<TextBlock>) {
  return (
    <div
      className="tiptap-content prose prose-sm max-w-[65ch] leading-relaxed text-foreground dark:prose-invert"
      dangerouslySetInnerHTML={{ __html: DOMPurify.sanitize(block.html) }}
    />
  )
}

function BulletView({ block }: BlockViewProps<BulletBlock>) {
  const items = block.items.filter((item) => item.trim().length > 0)
  const cls = 'max-w-[65ch] space-y-1.5 pl-5 text-sm leading-relaxed text-foreground'
  return block.ordered ? (
    <ol className={`list-decimal marker:text-muted-foreground ${cls}`}>
      {items.map((item, i) => (
        <li key={i}>{item}</li>
      ))}
    </ol>
  ) : (
    <ul className={`list-disc marker:text-muted-foreground/60 ${cls}`}>
      {items.map((item, i) => (
        <li key={i}>{item}</li>
      ))}
    </ul>
  )
}

function CalloutView({ block }: BlockViewProps<CalloutBlock>) {
  const { icon: Icon, tone } = CALLOUT_ICON[block.variant]
  return (
    <div className={`report-keep flex gap-3 rounded-xl border p-4 ${CALLOUT_STYLE[block.variant]}`}>
      <Icon className={`mt-0.5 h-4 w-4 shrink-0 ${tone}`} aria-hidden="true" />
      <div className="min-w-0">
        {block.title && <p className="mb-1 text-sm font-semibold text-foreground">{block.title}</p>}
        <div
          className="prose prose-sm max-w-[60ch] text-foreground dark:prose-invert"
          dangerouslySetInnerHTML={{ __html: DOMPurify.sanitize(block.html) }}
        />
      </div>
    </div>
  )
}

function DividerView(_props: BlockViewProps<DividerBlock>) {
  return <hr className="my-2 border-border-muted" />
}

function ImageView({ block }: BlockViewProps<ImageBlock>) {
  return (
    <figure className="report-keep space-y-2">
      <img src={block.url} alt={block.alt ?? ''} className="w-full rounded-lg border border-border-muted" />
      {block.caption && (
        <figcaption className="text-center text-xs italic text-muted-foreground">{block.caption}</figcaption>
      )}
    </figure>
  )
}

/* ------------------------------ registry defs ----------------------------- */

/**
 * The core block definitions, ready to register. Modules spread these into their
 * own registry and append module-specific blocks. `omit` drops core blocks a
 * surface doesn't want (e.g. wiki may not need every default).
 */
export function createCoreBlockDefs(options?: { omit?: string[] }): BlockTypeDef[] {
  const omit = new Set(options?.omit ?? [])
  const defs: BlockTypeDef[] = [
    defineBlock<HeadingBlock>({
      type: 'heading',
      icon: Heading,
      labelKey: 'document.block.heading.label',
      group: 'content',
      makeDefault: () => ({ id: docUid('b'), type: 'heading', level: 1, text: '' }),
      Edit: HeadingEdit,
      View: HeadingView,
    }),
    defineBlock<TextBlock>({
      type: 'text',
      icon: AlignLeft,
      labelKey: 'document.block.text.label',
      group: 'content',
      makeDefault: () => ({ id: docUid('b'), type: 'text', html: '' }),
      Edit: TextEdit,
      View: TextView,
    }),
    defineBlock<BulletBlock>({
      type: 'bullet',
      icon: List,
      labelKey: 'document.block.bullet.label',
      group: 'content',
      makeDefault: () => ({ id: docUid('b'), type: 'bullet', items: [''] }),
      Edit: BulletEdit,
      View: BulletView,
    }),
    defineBlock<CalloutBlock>({
      type: 'callout',
      icon: Lightbulb,
      labelKey: 'document.block.callout.label',
      group: 'content',
      makeDefault: () => ({ id: docUid('b'), type: 'callout', variant: 'info', html: '' }),
      Edit: CalloutEdit,
      View: CalloutView,
    }),
    defineBlock<ImageBlock>({
      type: 'image',
      icon: ImageIcon,
      labelKey: 'document.block.image.label',
      group: 'content',
      atomic: true,
      makeDefault: () => ({ id: docUid('b'), type: 'image', url: '' }),
      Edit: ImageEdit,
      View: ImageView,
    }),
    defineBlock<DividerBlock>({
      type: 'divider',
      icon: Minus,
      labelKey: 'document.block.divider.label',
      group: 'layout',
      makeDefault: () => ({ id: docUid('b'), type: 'divider' }),
      Edit: DividerEdit,
      View: DividerView,
    }),
  ]
  return defs.filter((d) => !omit.has(d.type))
}

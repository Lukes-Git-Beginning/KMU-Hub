/**
 * wiki block registry — wires the shared document engine into the wiki module.
 *
 * Phase B: the core blocks, but the text + heading blocks are overridden with
 * frameless, page-like editors so a wiki article reads as flowing long-form
 * prose rather than a stack of boxed cards (PB-2). The special elements
 * (callout, image, divider) keep their subtle frames from the core registry.
 *
 * Wiki-specific blocks (toggle, code, simple table, attachment) land in the next
 * batch as additional definitions appended here — the engine and every other
 * surface stay untouched.
 */
import { useRef, useState, type ChangeEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { Heading1, Heading2, Image as ImageIcon } from 'lucide-react'
import { RichTextEditor } from '@/components/shared/RichTextEditor'
import {
  buildRegistry,
  createCoreBlockDefs,
  type BlockEditProps,
  type BlockRegistry,
  type BlockTypeDef,
  type BlockViewProps,
  type HeadingBlock,
  type ImageBlock,
  type TextBlock,
} from '@/components/shared/document'
import { headingAnchorId } from './wikiReading'

/* ------------------------- frameless long-form text ----------------------- */

function WikiTextEdit({ block, onPatch }: BlockEditProps<TextBlock>) {
  const { t } = useTranslation()
  return (
    <RichTextEditor
      frameless
      content={block.html}
      onChange={(html) => onPatch({ html })}
      minHeight="1.75rem"
      placeholder={t('document.block.text.placeholder', { defaultValue: 'Text schreiben…' })}
    />
  )
}

/* ---------------------------- frameless heading --------------------------- */

function WikiHeadingEdit({ block, onPatch }: BlockEditProps<HeadingBlock>) {
  const { t } = useTranslation()
  const isH1 = block.level === 1
  return (
    <div className="group/wh relative">
      <input
        value={block.text}
        onChange={(e) => onPatch({ text: e.target.value })}
        placeholder={t('document.block.heading.placeholder', { defaultValue: 'Überschrift' })}
        className={`w-full border-0 bg-transparent px-0 text-foreground placeholder:text-muted-foreground/40 focus:outline-none ${
          isH1
            ? 'report-serif text-[1.9rem] font-semibold leading-tight tracking-tight'
            : 'text-xl font-semibold tracking-tight'
        }`}
      />
      {/* Level toggle — hidden until the heading is hovered or focused. */}
      <div className="absolute -top-2 right-0 hidden items-center overflow-hidden rounded-md border border-border bg-card shadow-sm group-hover/wh:flex group-focus-within/wh:flex">
        {([1, 2] as const).map((lvl) => {
          const Icon = lvl === 1 ? Heading1 : Heading2
          return (
            <button
              key={lvl}
              type="button"
              onClick={() => onPatch({ level: lvl })}
              className={`p-1 ${block.level === lvl ? 'bg-primary-light text-primary' : 'text-muted-foreground hover:bg-secondary'}`}
              aria-label={`H${lvl}`}
            >
              <Icon className="h-3.5 w-3.5" />
            </button>
          )
        })}
      </div>
    </div>
  )
}

/* ---------------------------- image (file-first) -------------------------- */
// Most wiki images are local files, not web URLs — so lead with a file picker
// (reads to a data URL) and keep the URL field as a secondary, opt-in option.

function WikiImageEdit({ block, onPatch }: BlockEditProps<ImageBlock>) {
  const { t } = useTranslation()
  const fileRef = useRef<HTMLInputElement>(null)
  const [showUrl, setShowUrl] = useState(false)
  const inputCls =
    'w-full rounded-md border border-border bg-card px-2 py-1.5 text-xs text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring'

  const pick = () => fileRef.current?.click()
  const onFile = (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    const reader = new FileReader()
    reader.onload = () => onPatch({ url: String(reader.result), alt: block.alt || file.name })
    reader.readAsDataURL(file)
    e.target.value = ''
  }

  return (
    <div className="space-y-2 rounded-xl border border-border bg-card p-3">
      {block.url ? (
        <img src={block.url} alt={block.alt ?? ''} className="max-h-56 w-full rounded-lg object-contain" />
      ) : (
        <button
          type="button"
          onClick={pick}
          className="flex h-28 w-full flex-col items-center justify-center gap-1.5 rounded-lg border border-dashed border-border bg-secondary/30 text-muted-foreground transition-colors hover:border-primary/40 hover:bg-secondary/50"
        >
          <ImageIcon className="h-6 w-6 text-muted-foreground/60" />
          <span className="text-xs font-medium">{t('wiki.image.pick', { defaultValue: 'Bild auswählen' })}</span>
          <span className="text-[11px] text-muted-foreground/70">
            {t('wiki.image.pickHint', { defaultValue: 'Aus deinen Dateien' })}
          </span>
        </button>
      )}
      <input ref={fileRef} type="file" accept="image/*" className="hidden" onChange={onFile} />
      <div className="flex flex-wrap items-center gap-2">
        <button
          type="button"
          onClick={pick}
          className="rounded-md border border-border px-2 py-1 text-xs text-foreground transition-colors hover:bg-secondary"
        >
          {block.url
            ? t('wiki.image.replace', { defaultValue: 'Bild ersetzen' })
            : t('wiki.image.pick', { defaultValue: 'Bild auswählen' })}
        </button>
        <button
          type="button"
          onClick={() => setShowUrl((v) => !v)}
          className="text-xs text-muted-foreground underline-offset-2 transition-colors hover:text-foreground hover:underline"
        >
          {t('wiki.image.useUrl', { defaultValue: 'oder Bild-Adresse einfügen' })}
        </button>
      </div>
      {showUrl && (
        <input
          value={block.url}
          onChange={(e) => onPatch({ url: e.target.value })}
          placeholder={t('wiki.image.urlPlaceholder', { defaultValue: 'https://… Bild-Adresse' })}
          className={inputCls}
        />
      )}
      <div className="flex gap-2">
        <input
          value={block.alt ?? ''}
          onChange={(e) => onPatch({ alt: e.target.value })}
          placeholder={t('document.block.image.altPlaceholder', { defaultValue: 'Alternativtext' })}
          className={inputCls}
        />
        <input
          value={block.caption ?? ''}
          onChange={(e) => onPatch({ caption: e.target.value })}
          placeholder={t('document.block.image.captionPlaceholder', { defaultValue: 'Bildunterschrift' })}
          className={inputCls}
        />
      </div>
    </div>
  )
}

/** Read view — mirrors the core heading, plus a stable anchor id for the TOC. */
function WikiHeadingView({ block }: BlockViewProps<HeadingBlock>) {
  return block.level === 1 ? (
    <h2
      id={headingAnchorId(block.id)}
      className="report-serif mt-2 scroll-mt-4 text-[2rem] font-semibold leading-tight tracking-tight text-foreground"
    >
      {block.text}
    </h2>
  ) : (
    <h3
      id={headingAnchorId(block.id)}
      className="scroll-mt-4 text-lg font-semibold tracking-tight text-foreground"
    >
      {block.text}
    </h3>
  )
}

/* -------------------------------- registry -------------------------------- */

// Core blocks, keyed so we can override text + heading and order the menu.
const core = Object.fromEntries(createCoreBlockDefs().map((d) => [d.type, d])) as Record<
  string,
  BlockTypeDef
>

// Keep each core block's icon/label/factory; swap in the frameless edit. The
// heading also gets a read-view with a stable anchor id so the TOC can target it.
const wikiText: BlockTypeDef = { ...core.text, Edit: WikiTextEdit as BlockTypeDef['Edit'] }
const wikiHeading: BlockTypeDef = {
  ...core.heading,
  Edit: WikiHeadingEdit as BlockTypeDef['Edit'],
  View: WikiHeadingView as BlockTypeDef['View'],
}
// File-first image picker; core read-view unchanged.
const wikiImage: BlockTypeDef = { ...core.image, Edit: WikiImageEdit as BlockTypeDef['Edit'] }

/**
 * Insert-menu order: prose first (the writer's default), then structure and the
 * special inline elements. Columns/layout come from the engine itself.
 */
export const wikiBlockRegistry: BlockRegistry = buildRegistry([
  wikiText,
  wikiHeading,
  core.bullet,
  core.callout,
  wikiImage,
  core.divider,
])

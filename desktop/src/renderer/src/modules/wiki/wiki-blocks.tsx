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
import { AtSign, FileText, Heading1, Heading2, Image as ImageIcon } from 'lucide-react'
import type { Editor } from '@tiptap/react'
import DOMPurify from 'dompurify'
import { RichTextEditor } from '@/components/shared/RichTextEditor'
import { LinkPreviewPopover } from '@/components/shared/LinkPreviewPopover'
import {
  buildRegistry,
  createCoreBlockDefs,
  createSpecialBlockDefs,
  type BlockEditProps,
  type BlockRegistry,
  type BlockTypeDef,
  type BlockViewProps,
  type HeadingBlock,
  type ImageBlock,
  type TextBlock,
} from '@/components/shared/document'
import { useArticles } from '@/api/hooks/useWiki'
import { useWikiStore } from '@/stores/wiki'
import { headingAnchorId } from './wikiReading'
import { useWikiUsers } from './useWikiUsers'
import { WikiSuggest } from './WikiSuggest'
import { WikiLink } from './extensions/WikiLinkExtension'
import { WikiMention } from './extensions/WikiMentionExtension'

// Stable extension list so the editor isn't rebuilt on every render. Preserves
// existing [[link]] / @mention nodes when a text block is edited (insertion UX
// follows in PB-4b).
const WIKI_TEXT_EXTENSIONS = [WikiLink, WikiMention]

/* ------------------------- frameless long-form text ----------------------- */

function WikiTextEdit({ block, onPatch }: BlockEditProps<TextBlock>) {
  const { t } = useTranslation()
  const [editor, setEditor] = useState<Editor | null>(null)
  return (
    <>
      <RichTextEditor
        frameless
        content={block.html}
        onChange={(html) => onPatch({ html })}
        minHeight="1.75rem"
        placeholder={t('document.block.text.placeholder', { defaultValue: 'Text schreiben…' })}
        extraExtensions={WIKI_TEXT_EXTENSIONS}
        onEditorReady={setEditor}
      />
      <WikiSuggest editor={editor} />
    </>
  )
}

type LinkPreview = {
  rect: DOMRect
  kind: 'article' | 'user'
  targetId: string | null
  name: string
  subtitle: string
}

/**
 * Read view — renders the long-form HTML and turns the formerly dead inline
 * cross-references into live ones: clicking a [[wiki link]] or @mention opens a
 * shared preview popover (who/what is the target) with a jump to it.
 */
function WikiTextView({ block }: BlockViewProps<TextBlock>) {
  const { t } = useTranslation()
  const setSelectedArticle = useWikiStore((s) => s.setSelectedArticle)
  const setEditing = useWikiStore((s) => s.setEditing)
  const resolveUser = useWikiUsers()
  const { data: articlesData } = useArticles()
  const [preview, setPreview] = useState<LinkPreview | null>(null)

  const onClick = (e: { target: EventTarget | null; preventDefault: () => void }) => {
    const el = e.target as HTMLElement
    const link = el.closest('[data-wiki-link]')
    if (link) {
      e.preventDefault()
      const id = link.getAttribute('data-article-id')
      const art = (articlesData?.articles ?? []).find((a) => a.id === id)
      setPreview({
        rect: link.getBoundingClientRect(),
        kind: 'article',
        targetId: id,
        name: (link.textContent ?? '').trim() || art?.title || t('wiki.link.articleType', { defaultValue: 'Wiki-Artikel' }),
        subtitle: t('wiki.link.articleType', { defaultValue: 'Wiki-Artikel' }),
      })
      return
    }
    const mention = el.closest('[data-mention-id]')
    if (mention) {
      e.preventDefault()
      const id = mention.getAttribute('data-mention-id')
      setPreview({
        rect: mention.getBoundingClientRect(),
        kind: 'user',
        targetId: id,
        name: resolveUser(id) || (mention.textContent ?? '').replace(/^@/, '').trim(),
        subtitle: t('wiki.link.teamMember', { defaultValue: 'Teammitglied' }),
      })
    }
  }

  const onAction = () => {
    if (!preview) return
    if (preview.kind === 'article' && preview.targetId) {
      setEditing(false)
      setSelectedArticle(preview.targetId)
    } else {
      window.location.hash = '#/team'
    }
    setPreview(null)
  }

  return (
    <>
      <div
        className="tiptap-content prose prose-sm max-w-[65ch] leading-relaxed text-foreground dark:prose-invert"
        onClick={onClick}
        dangerouslySetInnerHTML={{ __html: DOMPurify.sanitize(block.html) }}
      />
      {preview && (
        <LinkPreviewPopover
          anchor={preview.rect}
          icon={preview.kind === 'article' ? <FileText className="h-4 w-4" /> : <AtSign className="h-4 w-4" />}
          name={preview.name}
          subtitle={preview.subtitle}
          actionLabel={
            preview.kind === 'article'
              ? t('wiki.link.openArticle', { defaultValue: 'Artikel öffnen' })
              : t('wiki.link.openProfile', { defaultValue: 'Im Team öffnen' })
          }
          onAction={onAction}
          onClose={() => setPreview(null)}
          closeLabel={t('common.close', { defaultValue: 'Schließen' })}
        />
      )}
    </>
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
      {/* Heading-size switch — labelled so it reads as "Überschrift 1 / 2", not a
          cryptic badge. Floats above the heading on hover/focus, clear of the
          block's delete button at the top-right. */}
      <div className="absolute -top-7 left-0 z-10 hidden items-center gap-0.5 rounded-lg border border-border bg-card p-0.5 shadow-sm group-hover/wh:flex group-focus-within/wh:flex">
        <span className="px-1 text-[10px] font-medium uppercase tracking-wide text-muted-foreground/70">
          {t('wiki.heading.label', { defaultValue: 'Überschrift' })}
        </span>
        {([1, 2] as const).map((lvl) => {
          const Icon = lvl === 1 ? Heading1 : Heading2
          const label = `${t('wiki.heading.label', { defaultValue: 'Überschrift' })} ${lvl}`
          return (
            <button
              key={lvl}
              type="button"
              onClick={() => onPatch({ level: lvl })}
              title={label}
              aria-label={label}
              className={`flex items-center gap-1 rounded-md px-1.5 py-0.5 text-[11px] font-medium transition-colors ${
                block.level === lvl ? 'bg-primary-light text-primary' : 'text-muted-foreground hover:bg-secondary'
              }`}
            >
              <Icon className="h-3 w-3" />
              {lvl}
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
const wikiText: BlockTypeDef = {
  ...core.text,
  Edit: WikiTextEdit as BlockTypeDef['Edit'],
  View: WikiTextView as BlockTypeDef['View'],
}
const wikiHeading: BlockTypeDef = {
  ...core.heading,
  Edit: WikiHeadingEdit as BlockTypeDef['Edit'],
  View: WikiHeadingView as BlockTypeDef['View'],
}
// File-first image picker; core read-view unchanged.
const wikiImage: BlockTypeDef = { ...core.image, Edit: WikiImageEdit as BlockTypeDef['Edit'] }

// Wiki-specific special blocks, in insert-menu order. A knowledge article wants
// the structural elements (collapsible sections, code, tables, attachments,
// quotes, bookmarks); each lands here as it's built — the engine and every other
// surface stay untouched.
const wikiSpecial = createSpecialBlockDefs({ only: ['toggle', 'code', 'simpletable', 'attachment'] })

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
  ...wikiSpecial,
])

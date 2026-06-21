import { useState, useRef, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { useWikiPrefsStore } from '@/stores/wikiPrefs'
import { DocumentBlockEditor, type DocRow } from '@/components/shared/document'
import { wikiBlockRegistry } from './wiki-blocks'
import { WikiIdentityBar } from './WikiIdentityBar'
import { WikiAttachments } from './WikiAttachments'

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface WikiEditorProps {
  /** The article being edited — needed for inline attachment management. */
  articleId: string
  /** Editorial title heading — editable inline, saved together with the body. */
  title: string
  onTitleChange: (title: string) => void
  /** Block document (Phase B) — rows edited via the shared document engine. */
  rows: DocRow[]
  onChange: (rows: DocRow[]) => void
  /** Identity (WP-3) — icon + cover, edited in the head, saved with the rest. */
  icon?: string
  coverUrl?: string
  onIconChange: (icon: string | undefined) => void
  onCoverChange: (cover: string | undefined) => void
  /** Save the article; the optional change note annotates the new version. */
  onSave: (changeNote?: string) => void
  onCancel: () => void
  saving?: boolean
}

/**
 * Article editor — a focused, frameless writing canvas (WP-1) running on the
 * shared block-document engine (Phase B).
 *
 * The page reads as one quiet surface: a large Playfair title heading, the
 * block editor below (long-form text plus the special elements between it), and
 * a dezent footer with the change-note + save. Ctrl/Cmd+S saves, Escape cancels.
 */
export function WikiEditor({
  articleId,
  title,
  onTitleChange,
  rows,
  onChange,
  icon,
  coverUrl,
  onIconChange,
  onCoverChange,
  onSave,
  onCancel,
  saving,
}: WikiEditorProps) {
  const { t } = useTranslation()
  const readingWidth = useWikiPrefsStore((s) => s.readingWidth)
  const widthClass = readingWidth === 'wide' ? 'max-w-none' : 'max-w-[65ch] mx-auto'
  const [changeNote, setChangeNote] = useState('')

  // Auto-grow the title textarea so long titles wrap as an editorial heading
  // rather than scrolling inside an input box.
  const titleRef = useRef<HTMLTextAreaElement>(null)
  useEffect(() => {
    const el = titleRef.current
    if (!el) return
    el.style.height = 'auto'
    el.style.height = `${el.scrollHeight}px`
  }, [title])

  // Editor-shell shortcuts: Ctrl/Cmd+S saves, Escape cancels. Kept off inputs so
  // they don't fight inline text editing inside a block.
  const handleKeyDown = (e: React.KeyboardEvent) => {
    if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 's') {
      e.preventDefault()
      onSave(changeNote)
    }
  }

  return (
    <div className="flex flex-1 flex-col overflow-hidden" onKeyDown={handleKeyDown}>
      {/* Frameless canvas */}
      <div className="flex-1 overflow-y-auto px-6 py-6">
        {/* Identity — cover (full width) + icon (aligned to the measure) */}
        <WikiIdentityBar
          title={title}
          icon={icon}
          coverUrl={coverUrl}
          onIconChange={onIconChange}
          onCoverChange={onCoverChange}
          widthClass={widthClass}
        />
        <div className={widthClass}>
          {/* Editorial title heading (no input chrome) */}
          <textarea
            ref={titleRef}
            value={title}
            onChange={(e) => onTitleChange(e.target.value)}
            placeholder={t('wiki.editor.titlePlaceholder')}
            rows={1}
            spellCheck={false}
            aria-label={t('wiki.article.titleLabel')}
            className="wiki-title-input mb-6"
            onKeyDown={(e) => {
              // Enter in the title should not insert a newline — it commits and
              // lets the writer move into the body.
              if (e.key === 'Enter') e.preventDefault()
            }}
          />

          <DocumentBlockEditor
            rows={rows}
            onChange={onChange}
            registry={wikiBlockRegistry}
            widthClass="max-w-none"
          />

          {/* Attachments — manageable while writing, not only when reading. */}
          <div className="mt-8">
            <WikiAttachments articleId={articleId} />
          </div>
        </div>
      </div>

      {/* Footer actions */}
      <div className="flex items-center gap-3 border-t border-border bg-background/60 px-6 py-2.5">
        <div className="flex min-w-0 flex-1 items-center gap-2">
          <label htmlFor="wiki-change-note" className="shrink-0 text-[11px] text-muted-foreground">
            {t('wiki.editor.changeNote')}
          </label>
          <input
            id="wiki-change-note"
            value={changeNote}
            onChange={(e) => setChangeNote(e.target.value)}
            placeholder={t('wiki.editor.changeNotePlaceholder')}
            className="h-7 min-w-0 flex-1 rounded-md border border-border bg-transparent px-2 text-xs outline-none transition-colors placeholder:text-muted-foreground/70 focus:border-primary"
          />
        </div>
        <span className="hidden shrink-0 text-[10px] text-muted-foreground lg:inline">
          {t('wiki.editor.shortcutHint')}
        </span>
        <div className="flex shrink-0 items-center gap-2">
          <button
            onClick={onCancel}
            className="h-8 rounded-md border border-border px-3 text-xs text-foreground hover:bg-accent transition-colors"
          >
            {t('common.cancel')}
          </button>
          <button
            onClick={() => onSave(changeNote)}
            disabled={saving}
            className="h-8 rounded-md bg-primary px-3 text-xs font-medium text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50"
          >
            {saving ? t('wiki.editor.saving') : t('common.save')}
          </button>
        </div>
      </div>
    </div>
  )
}

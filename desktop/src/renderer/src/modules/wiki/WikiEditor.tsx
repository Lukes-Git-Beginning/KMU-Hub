import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useWikiPrefsStore } from '@/stores/wikiPrefs'
import { WikiRichEditor } from './WikiRichEditor'

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface WikiEditorProps {
  content: string
  onChange: (content: string) => void
  /** Save the article; the optional change note annotates the new version. */
  onSave: (changeNote?: string) => void
  onCancel: () => void
  saving?: boolean
}

/**
 * Article editor — wraps the wiki-specific TipTap editor (WikiRichEditor).
 *
 * The editor speaks HTML (onChange returns serialised HTML); the article store
 * persists it as `{ html }`. Ctrl/Cmd+S saves, Escape cancels — both handled
 * inside the editor so they cooperate with the @mention / [[link]] autocomplete.
 */
export function WikiEditor({ content, onChange, onSave, onCancel, saving }: WikiEditorProps) {
  const { t } = useTranslation()
  const readingWidth = useWikiPrefsStore((s) => s.readingWidth)
  const widthClass = readingWidth === 'wide' ? 'max-w-none' : 'max-w-3xl mx-auto'
  const [changeNote, setChangeNote] = useState('')

  return (
    <div className="flex flex-1 flex-col overflow-hidden">
      {/* Editor */}
      <div className="flex-1 overflow-y-auto px-4 py-3">
        <div className={widthClass}>
          <WikiRichEditor
            content={content}
            onChange={onChange}
            placeholder={t('wiki.editor.placeholder')}
            autofocus
            onSave={() => onSave(changeNote)}
            onCancel={onCancel}
          />
        </div>
      </div>

      {/* Footer actions */}
      <div className="flex items-center gap-3 border-t border-border px-4 py-2">
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

import { useTranslation } from 'react-i18next'
import { WikiRichEditor } from './WikiRichEditor'

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface WikiEditorProps {
  content: string
  onChange: (content: string) => void
  onSave: () => void
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

  return (
    <div className="flex flex-1 flex-col overflow-hidden">
      {/* Editor */}
      <div className="flex-1 overflow-y-auto px-4 py-3">
        <WikiRichEditor
          content={content}
          onChange={onChange}
          placeholder={t('wiki.editor.placeholder')}
          autofocus
          onSave={onSave}
          onCancel={onCancel}
        />
      </div>

      {/* Footer actions */}
      <div className="flex items-center justify-between gap-2 border-t border-border px-4 py-2">
        <span className="text-[11px] text-muted-foreground">
          {t('wiki.editor.shortcutHint')}
        </span>
        <div className="flex items-center gap-2">
          <button
            onClick={onCancel}
            className="h-8 rounded-md border border-border px-3 text-xs text-foreground hover:bg-accent transition-colors"
          >
            {t('common.cancel')}
          </button>
          <button
            onClick={onSave}
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

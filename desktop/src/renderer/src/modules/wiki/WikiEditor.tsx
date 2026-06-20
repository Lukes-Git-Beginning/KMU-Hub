import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { RichTextEditor } from '@/components/shared'

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
 * Article editor — wraps the shared TipTap RichTextEditor.
 *
 * The editor speaks HTML (onChange returns serialised HTML); the article store
 * persists it as `{ html }`. Ctrl/Cmd+S saves, Escape cancels — both captured
 * at the wrapper so they fire regardless of editor focus.
 */
export function WikiEditor({ content, onChange, onSave, onCancel, saving }: WikiEditorProps) {
  const { t } = useTranslation()
  const [html, setHtml] = useState(content)

  useEffect(() => {
    setHtml(content)
  }, [content])

  const handleChange = (value: string) => {
    setHtml(value)
    onChange(value)
  }

  const handleKeyDown = (e: React.KeyboardEvent<HTMLDivElement>) => {
    if (e.key === 's' && (e.ctrlKey || e.metaKey)) {
      e.preventDefault()
      onSave()
    }
    if (e.key === 'Escape') {
      e.preventDefault()
      onCancel()
    }
  }

  return (
    <div
      className="flex flex-1 flex-col overflow-hidden"
      onKeyDownCapture={handleKeyDown}
    >
      {/* Editor */}
      <div className="flex-1 overflow-y-auto px-4 py-3">
        <RichTextEditor
          content={content}
          onChange={handleChange}
          placeholder={t('wiki.editor.placeholder')}
          showFooter={false}
          autofocus
          minHeight="320px"
          maxHeight="none"
          className="border-border"
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

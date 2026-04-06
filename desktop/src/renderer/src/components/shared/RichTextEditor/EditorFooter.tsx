import type { Editor } from '@tiptap/react'
import { useTranslation } from 'react-i18next'

interface EditorFooterProps {
  editor: Editor
}

export function EditorFooter({ editor }: EditorFooterProps) {
  const { t } = useTranslation()
  const text = editor.state.doc.textContent
  const wordCount = text.trim() ? text.trim().split(/\s+/).length : 0
  const charCount = text.length

  return (
    <div className="flex items-center justify-end border-t border-border px-3 py-1.5 text-xs text-muted-foreground">
      <span>
        {t('shared.editor.wordCount', { count: wordCount })} · {t('shared.editor.charCount', { count: charCount })}
      </span>
    </div>
  )
}

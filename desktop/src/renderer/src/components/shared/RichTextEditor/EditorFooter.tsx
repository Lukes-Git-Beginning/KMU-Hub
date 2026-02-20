import type { Editor } from '@tiptap/react'

interface EditorFooterProps {
  editor: Editor
}

export function EditorFooter({ editor }: EditorFooterProps) {
  const text = editor.state.doc.textContent
  const wordCount = text.trim() ? text.trim().split(/\s+/).length : 0
  const charCount = text.length

  return (
    <div className="flex items-center justify-end border-t border-border px-3 py-1.5 text-xs text-muted-foreground">
      <span>
        {wordCount} {wordCount === 1 ? 'Wort' : 'Wörter'} · {charCount} Zeichen
      </span>
    </div>
  )
}

import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { StickyNote, Send } from 'lucide-react'

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface InternalNoteComposerProps {
  onSend: (content: string) => void
}

export function InternalNoteComposer({ onSend }: InternalNoteComposerProps) {
  const { t } = useTranslation()
  const [expanded, setExpanded] = useState(false)
  const [text, setText] = useState('')

  const handleSend = () => {
    if (!text.trim()) return
    onSend(text.trim())
    setText('')
    setExpanded(false)
  }

  if (!expanded) {
    return (
      <button
        onClick={() => setExpanded(true)}
        className="flex items-center gap-1.5 rounded-md px-2.5 py-1.5 text-xs text-warning hover:bg-warning/10 transition-colors"
      >
        <StickyNote className="h-3.5 w-3.5" />
        {t('kommunikation.note.internalNote')}
      </button>
    )
  }

  return (
    <div className="rounded-lg border border-warning/30 bg-warning/5 p-2.5">
      <div className="flex items-center gap-1.5 mb-2">
        <StickyNote className="h-3.5 w-3.5 text-warning" />
        <span className="text-[11px] font-medium text-warning">{t('kommunikation.note.internalNote')}</span>
        <span className="text-[10px] text-muted-foreground">{t('kommunikation.note.teamOnly')}</span>
      </div>
      <textarea
        value={text}
        onChange={(e) => setText(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
            e.preventDefault()
            handleSend()
          }
          if (e.key === 'Escape') {
            setExpanded(false)
            setText('')
          }
        }}
        placeholder={t('kommunikation.note.placeholder')}
        rows={2}
        className="w-full rounded-md border border-warning/20 bg-transparent px-2.5 py-1.5 text-sm text-foreground outline-none placeholder:text-muted-foreground focus:border-warning/40 resize-none"
        autoFocus
      />
      <div className="flex items-center justify-between mt-2">
        <span className="text-[10px] text-muted-foreground">{t('kommunikation.note.keyboardHint')}</span>
        <div className="flex gap-1.5">
          <button
            onClick={() => { setExpanded(false); setText('') }}
            className="rounded-md px-2.5 py-1 text-xs text-muted-foreground hover:bg-accent transition-colors"
          >
            {t('common.cancel')}
          </button>
          <button
            onClick={handleSend}
            disabled={!text.trim()}
            className="flex items-center gap-1 rounded-md bg-warning px-2.5 py-1 text-xs font-medium text-white hover:bg-warning/90 transition-colors disabled:opacity-50"
          >
            <Send className="h-3 w-3" />
            {t('kommunikation.note.send')}
          </button>
        </div>
      </div>
    </div>
  )
}

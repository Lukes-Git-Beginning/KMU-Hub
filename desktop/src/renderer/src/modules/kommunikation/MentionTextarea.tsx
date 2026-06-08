/**
 * Textarea with @mention autocomplete, reused across the Posteingang
 * composers (reply + internal note). Wraps the chat module's
 * MentionAutocomplete and handles the `@` trigger + caret tracking.
 *
 * Keyboard handling: while the mention dropdown is open, ArrowUp/Down/Enter/
 * Escape are owned by MentionAutocomplete (document listener) — this component
 * swallows them so the host's onSubmit/onEscape do not also fire. When the
 * dropdown is closed, Ctrl/Cmd+Enter triggers onSubmit and Escape onEscape.
 */
import { useRef, useState } from 'react'
import { MentionAutocomplete } from '@/modules/chat/messages/MentionAutocomplete'

interface MentionTextareaProps {
  value: string
  onChange: (value: string) => void
  onSubmit?: () => void
  onEscape?: () => void
  placeholder?: string
  rows?: number
  className?: string
  autoFocus?: boolean
}

export function MentionTextarea({
  value,
  onChange,
  onSubmit,
  onEscape,
  placeholder,
  rows = 3,
  className = '',
  autoFocus,
}: MentionTextareaProps) {
  const ref = useRef<HTMLTextAreaElement>(null)
  const [mention, setMention] = useState<{ query: string; start: number } | null>(null)

  const detect = (text: string, caret: number) => {
    const upto = text.slice(0, caret)
    const m = upto.match(/(?:^|\s)@(\w*)$/)
    if (m) {
      setMention({ query: m[1], start: caret - m[1].length - 1 })
    } else {
      setMention(null)
    }
  }

  const handleChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    onChange(e.target.value)
    detect(e.target.value, e.target.selectionStart ?? e.target.value.length)
  }

  const insertMention = (name: string) => {
    if (!mention) return
    const before = value.slice(0, mention.start)
    const after = value.slice(mention.start + 1 + mention.query.length)
    onChange(`${before}@${name} ${after}`)
    setMention(null)
    ref.current?.focus()
  }

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (mention && ['ArrowDown', 'ArrowUp', 'Enter', 'Escape'].includes(e.key)) {
      return // owned by MentionAutocomplete
    }
    if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
      e.preventDefault()
      onSubmit?.()
    } else if (e.key === 'Escape') {
      onEscape?.()
    }
  }

  return (
    <div className="relative">
      {mention && (
        <MentionAutocomplete
          query={mention.query}
          onSelect={insertMention}
          onClose={() => setMention(null)}
        />
      )}
      <textarea
        ref={ref}
        value={value}
        onChange={handleChange}
        onKeyDown={handleKeyDown}
        placeholder={placeholder}
        rows={rows}
        className={className}
        autoFocus={autoFocus}
      />
    </div>
  )
}

import { type ChangeEvent, type KeyboardEvent, useCallback, useRef, useState } from 'react'
import type { GuestChannelConfig } from '../api/types'
import { FileUpload } from './FileUpload'

interface MessageInputProps {
  onSend: (content: string) => void
  isSending: boolean
  disabled: boolean
  channelId: string
  config: GuestChannelConfig
}

const MAX_CHARS = 2000

export function MessageInput({ onSend, isSending, disabled, channelId, config }: MessageInputProps) {
  const [text, setText] = useState('')
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  const handleChange = useCallback((e: ChangeEvent<HTMLTextAreaElement>) => {
    const value = e.target.value
    if (value.length <= MAX_CHARS) {
      setText(value)
    }

    // Auto-resize textarea
    const ta = e.target
    ta.style.height = 'auto'
    ta.style.height = Math.min(ta.scrollHeight, 120) + 'px'
  }, [])

  const handleSend = useCallback(() => {
    const content = text.trim()
    if (!content || isSending || disabled) return
    onSend(content)
    setText('')
    if (textareaRef.current) {
      textareaRef.current.style.height = 'auto'
    }
  }, [text, isSending, disabled, onSend])

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault()
        handleSend()
      }
    },
    [handleSend],
  )

  const showCharCount = text.length > MAX_CHARS * 0.8

  return (
    <>
      {showCharCount && (
        <div
          className={`char-count ${text.length > MAX_CHARS * 0.95 ? 'error' : text.length > MAX_CHARS * 0.8 ? 'warn' : ''}`}
        >
          {text.length}/{MAX_CHARS}
        </div>
      )}
      <div className="message-input-area">
        <FileUpload channelId={channelId} config={config} disabled={disabled} />
        <textarea
          ref={textareaRef}
          className="message-textarea"
          placeholder={disabled ? 'Verbindung unterbrochen...' : 'Nachricht eingeben...'}
          value={text}
          onChange={handleChange}
          onKeyDown={handleKeyDown}
          disabled={disabled}
          rows={1}
        />
        <button
          className="icon-btn send"
          onClick={handleSend}
          disabled={!text.trim() || isSending || disabled}
          title="Senden"
        >
          {/* Paper plane SVG */}
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <line x1="22" y1="2" x2="11" y2="13" />
            <polygon points="22 2 15 22 11 13 2 9 22 2" />
          </svg>
        </button>
      </div>
    </>
  )
}

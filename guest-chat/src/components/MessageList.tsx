import { useEffect, useRef } from 'react'
import type { ChatMessage } from '../api/types'
import { MessageBubble } from './MessageBubble'

interface MessageListProps {
  messages: ChatMessage[]
  isLoading: boolean
  hasMore: boolean
  onLoadMore: () => void
  sessionId: string
  welcomeMessage: string
}

function formatDateSeparator(dateStr: string): string {
  const date = new Date(dateStr)
  const today = new Date()
  const yesterday = new Date()
  yesterday.setDate(yesterday.getDate() - 1)

  if (date.toDateString() === today.toDateString()) return 'Heute'
  if (date.toDateString() === yesterday.toDateString()) return 'Gestern'
  return date.toLocaleDateString('de-DE', { day: 'numeric', month: 'long', year: 'numeric' })
}

function getDateKey(dateStr: string): string {
  return new Date(dateStr).toDateString()
}

export function MessageList({
  messages,
  isLoading,
  hasMore,
  onLoadMore,
  sessionId,
  welcomeMessage,
}: MessageListProps) {
  const bottomRef = useRef<HTMLDivElement>(null)
  const listRef = useRef<HTMLDivElement>(null)
  const prevLengthRef = useRef(messages.length)

  // Auto-scroll to bottom on new messages (not on load more)
  useEffect(() => {
    if (messages.length > prevLengthRef.current) {
      bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
    }
    prevLengthRef.current = messages.length
  }, [messages.length])

  // Scroll to bottom on initial load
  useEffect(() => {
    if (!isLoading && messages.length > 0) {
      bottomRef.current?.scrollIntoView()
    }
  }, [isLoading]) // eslint-disable-line react-hooks/exhaustive-deps

  if (isLoading) {
    return (
      <div className="message-list">
        <div className="message-list-empty">
          <div className="spinner" />
        </div>
      </div>
    )
  }

  if (messages.length === 0) {
    return (
      <div className="message-list">
        <div className="message-list-empty">{welcomeMessage}</div>
      </div>
    )
  }

  let lastDateKey = ''

  return (
    <div className="message-list" ref={listRef}>
      {hasMore && (
        <button className="load-more-btn" onClick={onLoadMore}>
          Aeltere Nachrichten laden
        </button>
      )}
      {messages.map((msg) => {
        const dateKey = getDateKey(msg.created_at)
        const showSeparator = dateKey !== lastDateKey
        lastDateKey = dateKey

        return (
          <div key={msg.id}>
            {showSeparator && (
              <div className="date-separator">
                <span>{formatDateSeparator(msg.created_at)}</span>
              </div>
            )}
            <MessageBubble message={msg} isSelf={msg.guest_session_id === sessionId} />
          </div>
        )
      })}
      <div ref={bottomRef} />
    </div>
  )
}

import { useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import type { ConversationMessage } from '@/types/communication'
import { MessageItem } from './MessageItem'

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function isSameDay(a: string, b: string): boolean {
  try {
    return new Date(a).toDateString() === new Date(b).toDateString()
  } catch {
    return false
  }
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface MessageTimelineProps {
  messages: ConversationMessage[]
}

export function MessageTimeline({ messages }: MessageTimelineProps) {
  const bottomRef = useRef<HTMLDivElement>(null)

  // Auto-scroll to bottom on new messages
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages.length])

  return (
    <div className="flex-1 overflow-y-auto px-4 py-4 space-y-3">
      {messages.map((msg, idx) => {
        const prevMsg = idx > 0 ? messages[idx - 1] : null
        const showDate = !prevMsg || !isSameDay(prevMsg.timestamp, msg.timestamp)

        return (
          <MessageItem
            key={msg.id}
            message={msg}
            showDate={showDate}
          />
        )
      })}
      <div ref={bottomRef} />
    </div>
  )
}

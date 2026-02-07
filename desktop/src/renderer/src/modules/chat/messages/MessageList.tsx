/**
 * Scrollable message history for a chat channel.
 *
 * Displays messages reverse-chronologically (newest at bottom) with
 * infinite scroll for older messages, date separators, and real-time
 * updates via WebSocket.
 */
import { useEffect, useRef, useCallback } from 'react'
import { format, isToday, isYesterday } from 'date-fns'
import { de } from 'date-fns/locale'
import { Loader2 } from 'lucide-react'
import { useMessages, useMessageWebSocket, useEditMessage, useDeleteMessage, type MessageInfo } from '@/api/hooks/useMessages'
import { useAuthStore } from '@/stores/auth'
import { ScrollArea } from '@/components/ui/scroll-area'
import { MessageBubble } from './MessageBubble'

interface MessageListProps {
  channelId: string
  onOpenThread: (messageId: string) => void
}

export function MessageList({ channelId, onOpenThread }: MessageListProps) {
  const currentUserId = useAuthStore((s) => s.user?.id)
  const scrollRef = useRef<HTMLDivElement>(null)
  const bottomRef = useRef<HTMLDivElement>(null)
  const wasAtBottomRef = useRef(true)

  const {
    allMessages: messages,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
    isLoading,
  } = useMessages(channelId)

  // Subscribe to real-time message updates
  useMessageWebSocket(channelId)

  const editMessage = useEditMessage()
  const deleteMessage = useDeleteMessage()

  // Auto-scroll to bottom on new messages (only if user was already at bottom)
  useEffect(() => {
    if (wasAtBottomRef.current && bottomRef.current) {
      bottomRef.current.scrollIntoView({ behavior: 'smooth' })
    }
  }, [messages.length])

  // Scroll to bottom on channel change
  useEffect(() => {
    if (bottomRef.current) {
      bottomRef.current.scrollIntoView()
    }
    wasAtBottomRef.current = true
  }, [channelId])

  // Track scroll position to know if user is at bottom
  const handleScroll = useCallback((e: React.UIEvent<HTMLDivElement>) => {
    const target = e.currentTarget
    const atBottom = target.scrollHeight - target.scrollTop - target.clientHeight < 50
    wasAtBottomRef.current = atBottom

    // Load older messages when scrolled to top
    if (target.scrollTop < 100 && hasNextPage && !isFetchingNextPage) {
      fetchNextPage()
    }
  }, [hasNextPage, isFetchingNextPage, fetchNextPage])

  const handleEdit = useCallback((messageId: string, content: string) => {
    const newContent = window.prompt('Nachricht bearbeiten:', content)
    if (newContent !== null && newContent.trim() !== content) {
      editMessage.mutate({ messageId, content: newContent.trim() })
    }
  }, [editMessage])

  const handleDelete = useCallback((messageId: string) => {
    if (window.confirm('Nachricht wirklich loeschen?')) {
      deleteMessage.mutate(messageId)
    }
  }, [deleteMessage])

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="text-center">
          <div className="mx-auto h-8 w-8 animate-spin rounded-full border-4 border-muted border-t-primary" />
          <p className="mt-4 text-sm text-muted-foreground">Nachrichten laden...</p>
        </div>
      </div>
    )
  }

  if (messages.length === 0) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="text-center">
          <p className="text-sm text-muted-foreground">
            Noch keine Nachrichten. Schreibe die erste!
          </p>
        </div>
      </div>
    )
  }

  // Group messages by date
  const groupedMessages = groupByDate(messages)

  return (
    <div
      ref={scrollRef}
      className="flex-1 overflow-y-auto"
      onScroll={handleScroll}
    >
      {/* Load more indicator */}
      {isFetchingNextPage && (
        <div className="flex justify-center py-2">
          <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
        </div>
      )}

      {hasNextPage && !isFetchingNextPage && (
        <div className="flex justify-center py-2">
          <button
            className="text-xs text-muted-foreground hover:text-foreground"
            onClick={() => fetchNextPage()}
          >
            Aeltere Nachrichten laden
          </button>
        </div>
      )}

      {/* Messages grouped by date */}
      {groupedMessages.map(({ date, messages: dateMessages }) => (
        <div key={date}>
          <DateSeparator date={date} />
          {dateMessages.map((message) => (
            <MessageBubble
              key={message.id}
              message={message}
              isOwn={message.created_by === currentUserId}
              onOpenThread={onOpenThread}
              onEdit={handleEdit}
              onDelete={handleDelete}
            />
          ))}
        </div>
      ))}

      {/* Scroll anchor */}
      <div ref={bottomRef} />
    </div>
  )
}

function DateSeparator({ date }: { date: string }) {
  return (
    <div className="flex items-center gap-4 px-4 py-3">
      <div className="flex-1 border-t border-border" />
      <span className="text-xs font-medium text-muted-foreground">{date}</span>
      <div className="flex-1 border-t border-border" />
    </div>
  )
}

interface MessageGroup {
  date: string
  messages: MessageInfo[]
}

function groupByDate(messages: MessageInfo[]): MessageGroup[] {
  const groups = new Map<string, MessageInfo[]>()

  for (const msg of messages) {
    const dateStr = msg.created_at
      ? formatDateLabel(new Date(msg.created_at))
      : 'Unbekannt'

    const existing = groups.get(dateStr)
    if (existing) {
      existing.push(msg)
    } else {
      groups.set(dateStr, [msg])
    }
  }

  return Array.from(groups.entries()).map(([date, msgs]) => ({
    date,
    messages: msgs,
  }))
}

function formatDateLabel(date: Date): string {
  if (isToday(date)) return 'Heute'
  if (isYesterday(date)) return 'Gestern'
  return format(date, 'd. MMMM yyyy', { locale: de })
}

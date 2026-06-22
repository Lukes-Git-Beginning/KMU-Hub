/**
 * Virtualized scrollable message history for a chat channel.
 *
 * Uses @tanstack/react-virtual for windowed rendering so only visible
 * messages are in the DOM. Supports infinite scroll for older messages,
 * date separators, and real-time updates via WebSocket.
 */
import { useEffect, useRef, useCallback, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/cn'
import { format, isToday, isYesterday } from 'date-fns'
import { de } from 'date-fns/locale'
import { Loader2 } from 'lucide-react'
import { useVirtualizer } from '@tanstack/react-virtual'
import { useMessages, useMessageWebSocket, useEditMessage, useDeleteMessage, type MessageInfo } from '@/api/hooks/useMessages'
import { useMarkChannelRead } from '@/api/hooks/useChannels'
import { useReactionSummary } from '@/api/hooks/useReactions'
import { useAuthStore } from '@/stores/auth'
import { MessageBubble } from './MessageBubble'
import type { ReactionSummary } from '@/api/video-types'

interface MessageListProps {
  channelId: string
  onOpenThread: (messageId: string) => void
  /** When set, scroll to and briefly highlight this message (jump-to-search-result). */
  highlightMessageId?: string | null
}

type FlatItem =
  | { type: 'date'; date: string }
  | { type: 'message'; message: MessageInfo }

export function MessageList({ channelId, onOpenThread, highlightMessageId }: MessageListProps) {
  const { t } = useTranslation()
  const currentUserId = useAuthStore((s) => s.user?.id)
  const scrollRef = useRef<HTMLDivElement>(null)
  const wasAtBottomRef = useRef(true)
  const [flashId, setFlashId] = useState<string | null>(null)

  const {
    allMessages: messages,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
    isLoading,
  } = useMessages(channelId)

  useMessageWebSocket(channelId)

  const editMessage = useEditMessage()
  const deleteMessage = useDeleteMessage()
  const markRead = useMarkChannelRead()

  // Batch-fetch reaction summaries for all loaded messages (one request, not N).
  // Re-runs whenever the message list changes (new messages added / channel switch).
  const messageIds = useMemo(() => messages.map((m) => m.id ?? '').filter(Boolean), [messages])
  const { data: reactionSummaryData } = useReactionSummary(messageIds)

  // Build a lookup map: messageId → ReactionSummary[] for O(1) access per bubble.
  const reactionsByMessage = useMemo<Record<string, ReactionSummary[]>>(() => {
    const map: Record<string, ReactionSummary[]> = {}
    for (const s of reactionSummaryData?.summaries ?? []) {
      if (!map[s.message_id]) map[s.message_id] = []
      map[s.message_id].push(s)
    }
    return map
  }, [reactionSummaryData])

  // Mark the channel read once its newest message is known (and whenever the
  // channel changes or a new message arrives). Clears the unread badge.
  const latestMessageId = messages.length > 0 ? messages[messages.length - 1]?.id : undefined
  useEffect(() => {
    if (channelId && latestMessageId) {
      markRead.mutate({ channelId, messageId: latestMessageId })
    }
  }, [channelId, latestMessageId]) // eslint-disable-line react-hooks/exhaustive-deps

  // Flatten grouped messages into a single array for virtualizer
  const flatItems = useMemo<FlatItem[]>(() => {
    const groups = groupByDate(messages, t)
    const items: FlatItem[] = []
    for (const group of groups) {
      items.push({ type: 'date', date: group.date })
      for (const msg of group.messages) {
        items.push({ type: 'message', message: msg })
      }
    }
    return items
  }, [messages, t])

  const virtualizer = useVirtualizer({
    count: flatItems.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: (index) => flatItems[index].type === 'date' ? 40 : 72,
    overscan: 10,
  })

  // Auto-scroll to bottom on new messages (only if user was already at bottom)
  useEffect(() => {
    if (wasAtBottomRef.current && flatItems.length > 0) {
      virtualizer.scrollToIndex(flatItems.length - 1, { align: 'end', behavior: 'smooth' })
    }
  }, [flatItems.length, virtualizer])

  // Scroll to bottom on channel change
  useEffect(() => {
    if (flatItems.length > 0) {
      virtualizer.scrollToIndex(flatItems.length - 1, { align: 'end' })
    }
    wasAtBottomRef.current = true
  }, [channelId]) // eslint-disable-line react-hooks/exhaustive-deps

  // Jump-to-search-result: scroll to the message and flash-highlight it briefly.
  useEffect(() => {
    if (!highlightMessageId || flatItems.length === 0) return
    const index = flatItems.findIndex(
      (it) => it.type === 'message' && it.message.id === highlightMessageId,
    )
    if (index < 0) return
    virtualizer.scrollToIndex(index, { align: 'center' })
    setFlashId(highlightMessageId)
    const timer = setTimeout(() => setFlashId(null), 2200)
    return () => clearTimeout(timer)
  }, [highlightMessageId, flatItems, virtualizer])

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

  // Inline edit: MessageBubble owns the editing UI and passes the new content.
  const handleEdit = useCallback((messageId: string, newContent: string) => {
    const trimmed = newContent.trim()
    if (trimmed) {
      editMessage.mutate({ messageId, content: trimmed })
    }
  }, [editMessage])

  const handleDelete = useCallback((messageId: string) => {
    if (window.confirm(t('chat.messages.deleteConfirm'))) {
      deleteMessage.mutate(messageId)
    }
  }, [deleteMessage, t])

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="text-center">
          <div className="mx-auto h-8 w-8 animate-spin rounded-full border-4 border-muted border-t-primary" />
          <p className="mt-4 text-sm text-muted-foreground">{t('chat.messages.loading')}</p>
        </div>
      </div>
    )
  }

  if (messages.length === 0) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="text-center">
          <p className="text-sm text-muted-foreground">
            {t('chat.messages.empty')}
          </p>
        </div>
      </div>
    )
  }

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
            {t('chat.messages.loadOlder')}
          </button>
        </div>
      )}

      {/* Virtualized message list */}
      <div
        style={{
          height: `${virtualizer.getTotalSize()}px`,
          width: '100%',
          position: 'relative',
        }}
      >
        {virtualizer.getVirtualItems().map((virtualItem) => {
          const item = flatItems[virtualItem.index]
          return (
            <div
              key={virtualItem.key}
              data-index={virtualItem.index}
              ref={virtualizer.measureElement}
              className={cn(
                'transition-colors duration-700',
                item.type === 'message' && flashId === item.message.id && 'bg-warning/15',
              )}
              style={{
                position: 'absolute',
                top: 0,
                left: 0,
                width: '100%',
                transform: `translateY(${virtualItem.start}px)`,
              }}
            >
              {item.type === 'date' ? (
                <DateSeparator date={item.date} />
              ) : (
                <MessageBubble
                  message={item.message}
                  isOwn={item.message.created_by === currentUserId}
                  currentUserId={currentUserId}
                  reactionSummaries={reactionsByMessage[item.message.id ?? ''] ?? []}
                  onOpenThread={onOpenThread}
                  onEdit={handleEdit}
                  onDelete={handleDelete}
                />
              )}
            </div>
          )
        })}
      </div>
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

function groupByDate(messages: MessageInfo[], t: (key: string) => string): MessageGroup[] {
  const groups = new Map<string, MessageInfo[]>()

  for (const msg of messages) {
    const dateStr = msg.created_at
      ? formatDateLabel(new Date(msg.created_at), t)
      : t('chat.unknown')

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

function formatDateLabel(date: Date, t: (key: string) => string): string {
  if (isToday(date)) return t('chat.messages.today')
  if (isYesterday(date)) return t('chat.messages.yesterday')
  return format(date, 'd. MMMM yyyy', { locale: de })
}

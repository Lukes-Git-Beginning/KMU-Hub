import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Search, Plus, MessageSquareText, Loader2 } from 'lucide-react'
import { moduleHsl } from '@/components/layout/sidebar/nav-items'
import { useKommunikationStore } from '@/stores/kommunikation'
import type { CommunicationChannel } from '@/types/communication'
import type { InboxMessage } from '@/api/inbox-types'
import { useUnreadCount } from '@/api/hooks/useInbox'
import { ChannelTabs } from './ChannelTabs'
import { ConversationListFilters, type ConversationSort } from './ConversationListFilters'
import { ConversationListItem } from './ConversationListItem'
import { EmptyState } from '@/components/shared'

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface ConversationListProps {
  messages: InboxMessage[]
  isLoading: boolean
  onNewConversation: () => void
}

export function ConversationList({ messages, isLoading, onNewConversation }: ConversationListProps) {
  const { t } = useTranslation()
  const activeChannel = useKommunikationStore((s) => s.activeChannel)
  const setActiveChannel = useKommunikationStore((s) => s.setActiveChannel)
  const searchQuery = useKommunikationStore((s) => s.searchQuery)
  const setSearchQuery = useKommunikationStore((s) => s.setSearchQuery)
  const selectedConversationId = useKommunikationStore((s) => s.selectedConversationId)
  const setSelectedConversation = useKommunikationStore((s) => s.setSelectedConversation)

  // TODO: statusFilter not supported by InboxMessage — keep UI but filter client-side is no longer possible
  const [sort, setSort] = useState<ConversationSort>('newest')

  // Unread counts per channel from API
  const { data: unreadData } = useUnreadCount()
  const unreadCounts = useMemo(() => {
    const counts: Record<string, number> = {}
    if (unreadData?.counts) {
      for (const c of unreadData.counts) {
        counts[c.channel] = c.count
      }
    }
    return counts
  }, [unreadData])

  // Sort messages client-side (filtering is done via API filter in KommunikationPage)
  const sorted = useMemo(() => {
    const result = [...messages]

    switch (sort) {
      case 'newest':
        result.sort((a, b) => b.received_at.localeCompare(a.received_at))
        break
      case 'oldest':
        result.sort((a, b) => a.received_at.localeCompare(b.received_at))
        break
      case 'priority':
        // TODO: InboxMessage has no priority field — keep default sort
        break
      case 'unread':
        result.sort((a, b) => (a.is_read === b.is_read ? 0 : a.is_read ? 1 : -1))
        break
    }

    return result
  }, [messages, sort])

  return (
    <div className="flex w-80 shrink-0 flex-col border-r border-border bg-card/50">
      {/* Header */}
      <div className="flex items-center justify-between px-3 py-2.5 border-b border-border">
        <div className="flex items-center gap-2">
          <div
            className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg"
            style={{ backgroundColor: moduleHsl('kommunikation') + '15', color: moduleHsl('kommunikation') }}
          >
            <MessageSquareText className="h-3.5 w-3.5" />
          </div>
          <h2 className="text-sm font-semibold" style={{ color: moduleHsl('kommunikation') }}>{t('kommunikation.title')}</h2>
        </div>
        <button
          onClick={onNewConversation}
          className="rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
          title={t('kommunikation.conversation.new')}
        >
          <Plus className="h-4 w-4" />
        </button>
      </div>

      {/* Search */}
      <div className="px-3 py-2 border-b border-border">
        <div className="relative">
          <Search className="absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder={t('kommunikation.conversation.searchPlaceholder')}
            className="h-8 w-full rounded-md border border-border bg-transparent pl-8 pr-3 text-sm outline-none transition-colors placeholder:text-muted-foreground focus:border-primary"
          />
        </div>
      </div>

      {/* Channel tabs */}
      <ChannelTabs
        active={activeChannel}
        onChange={(ch) => setActiveChannel(ch as CommunicationChannel | 'all')}
        unreadCounts={unreadCounts}
      />

      {/* Status filters + sort */}
      <ConversationListFilters
        statusFilter="all"
        onStatusChange={() => {/* TODO: Status filter not available in InboxMessage API */}}
        sort={sort}
        onSortChange={setSort}
        totalCount={sorted.length}
      />

      {/* Conversation list */}
      <div className="flex-1 overflow-y-auto">
        {isLoading ? (
          <div className="flex items-center justify-center py-12">
            <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
          </div>
        ) : sorted.length === 0 ? (
          <EmptyState
            icon={MessageSquareText}
            title={t('kommunikation.conversation.noConversations')}
            description={
              searchQuery
                ? t('kommunikation.conversation.noSearchResults')
                : t('kommunikation.conversation.noConversationsInChannel')
            }
          />
        ) : (
          sorted.map((msg) => (
            <ConversationListItem
              key={msg.id}
              message={msg}
              isSelected={selectedConversationId === msg.id}
              onSelect={setSelectedConversation}
            />
          ))
        )}
      </div>
    </div>
  )
}

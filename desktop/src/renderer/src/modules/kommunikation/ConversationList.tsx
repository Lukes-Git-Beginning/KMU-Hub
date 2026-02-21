import { useState, useMemo } from 'react'
import { Search, Plus, MessageSquareText } from 'lucide-react'
import { useKommunikationStore } from '@/stores/kommunikation'
import type { CommunicationChannel, ConversationStatus } from '@/types/communication'
import { ChannelTabs } from './ChannelTabs'
import { ConversationListFilters, type ConversationSort } from './ConversationListFilters'
import { ConversationListItem } from './ConversationListItem'
import { EmptyState } from '@/components/shared'

// ---------------------------------------------------------------------------
// Priority weight for sorting
// ---------------------------------------------------------------------------

const priorityWeight: Record<string, number> = {
  urgent: 0,
  high: 1,
  normal: 2,
  low: 3,
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface ConversationListProps {
  onNewConversation: () => void
}

export function ConversationList({ onNewConversation }: ConversationListProps) {
  const conversations = useKommunikationStore((s) => s.conversations)
  const activeChannel = useKommunikationStore((s) => s.activeChannel)
  const setActiveChannel = useKommunikationStore((s) => s.setActiveChannel)
  const searchQuery = useKommunikationStore((s) => s.searchQuery)
  const setSearchQuery = useKommunikationStore((s) => s.setSearchQuery)
  const selectedConversationId = useKommunikationStore((s) => s.selectedConversationId)
  const setSelectedConversation = useKommunikationStore((s) => s.setSelectedConversation)

  const [statusFilter, setStatusFilter] = useState<ConversationStatus | 'all'>('all')
  const [sort, setSort] = useState<ConversationSort>('newest')

  // Unread counts per channel
  const unreadCounts = useMemo(() => {
    const counts: Record<string, number> = {}
    for (const c of conversations) {
      if (c.unreadCount > 0) {
        counts[c.channel] = (counts[c.channel] ?? 0) + c.unreadCount
      }
    }
    return counts
  }, [conversations])

  // Filter + sort conversations
  const filtered = useMemo(() => {
    let result = conversations

    // Channel filter
    if (activeChannel !== 'all') {
      result = result.filter((c) => c.channel === activeChannel)
    }

    // Status filter
    if (statusFilter !== 'all') {
      result = result.filter((c) => c.status === statusFilter)
    }

    // Search
    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase()
      result = result.filter(
        (c) =>
          c.subject.toLowerCase().includes(q) ||
          c.contactName.toLowerCase().includes(q) ||
          c.contactCompany?.toLowerCase().includes(q) ||
          c.lastMessage.toLowerCase().includes(q) ||
          c.tags.some((t) => t.toLowerCase().includes(q)),
      )
    }

    // Sort
    switch (sort) {
      case 'newest':
        result = [...result].sort((a, b) => b.lastMessageAt.localeCompare(a.lastMessageAt))
        break
      case 'oldest':
        result = [...result].sort((a, b) => a.lastMessageAt.localeCompare(b.lastMessageAt))
        break
      case 'priority':
        result = [...result].sort(
          (a, b) => (priorityWeight[a.priority] ?? 2) - (priorityWeight[b.priority] ?? 2),
        )
        break
      case 'unread':
        result = [...result].sort((a, b) => b.unreadCount - a.unreadCount)
        break
    }

    return result
  }, [conversations, activeChannel, statusFilter, searchQuery, sort])

  return (
    <div className="flex w-80 shrink-0 flex-col border-r border-border bg-card/50">
      {/* Header */}
      <div className="flex items-center justify-between px-3 py-2.5 border-b border-border">
        <h2 className="text-sm font-semibold text-foreground">Kommunikation</h2>
        <button
          onClick={onNewConversation}
          className="rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
          title="Neue Konversation"
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
            placeholder="Konversationen suchen..."
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
        statusFilter={statusFilter}
        onStatusChange={setStatusFilter}
        sort={sort}
        onSortChange={setSort}
        totalCount={filtered.length}
      />

      {/* Conversation list */}
      <div className="flex-1 overflow-y-auto">
        {filtered.length === 0 ? (
          <EmptyState
            icon={MessageSquareText}
            title="Keine Konversationen"
            description={
              searchQuery
                ? 'Keine Treffer fuer diese Suche.'
                : 'Noch keine Konversationen in diesem Kanal.'
            }
          />
        ) : (
          filtered.map((conv) => (
            <ConversationListItem
              key={conv.id}
              conversation={conv}
              isSelected={selectedConversationId === conv.id}
              onSelect={setSelectedConversation}
            />
          ))
        )}
      </div>
    </div>
  )
}

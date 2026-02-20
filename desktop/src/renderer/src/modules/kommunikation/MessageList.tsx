/**
 * Center column: scrollable list of inbox messages.
 *
 * Each item: channel badge + sender + timestamp, subject, preview.
 * Quick actions on hover: reply, snooze, archive, assign, star.
 * Channel badge colors: blue (email), green (chat), orange (notification).
 */
import {
  Mail,
  MessageSquare,
  Bell,
  Reply,
  Clock,
  Archive,
  UserPlus,
  Star,
  Inbox,
} from 'lucide-react'
import { toast } from 'sonner'
import type { InboxMessage, InboxChannel } from '@/api/inbox-types'
import {
  useMarkRead,
  useToggleStar,
  useArchiveMessage,
  useSnoozeMessage,
} from '@/api/hooks/useInbox'
import { SnoozePopover } from './SnoozePopover'
import { EmptyState } from '@/components/shared'

// ---------------------------------------------------------------------------
// Channel badge config
// ---------------------------------------------------------------------------

const channelConfig: Record<
  InboxChannel,
  { icon: typeof Mail; color: string; bg: string; label: string }
> = {
  email: {
    icon: Mail,
    color: 'text-blue-500',
    bg: 'bg-blue-50 dark:bg-blue-950/30',
    label: 'E-Mail',
  },
  chat: {
    icon: MessageSquare,
    color: 'text-green-500',
    bg: 'bg-green-50 dark:bg-green-950/30',
    label: 'Chat',
  },
  notification: {
    icon: Bell,
    color: 'text-orange-500',
    bg: 'bg-orange-50 dark:bg-orange-950/30',
    label: 'Benachrichtigung',
  },
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function formatRelativeTime(dateStr: string): string {
  try {
    const date = new Date(dateStr)
    const now = new Date()
    const diffMs = now.getTime() - date.getTime()
    const diffMin = Math.floor(diffMs / 60000)
    const diffH = Math.floor(diffMin / 60)
    const diffD = Math.floor(diffH / 24)

    if (diffMin < 1) return 'Gerade eben'
    if (diffMin < 60) return `${diffMin} Min.`
    if (diffH < 24) return `${diffH} Std.`
    if (diffD < 7) return `${diffD} T.`
    return date.toLocaleDateString('de-DE', {
      day: '2-digit',
      month: '2-digit',
    })
  } catch {
    return dateStr
  }
}

// ---------------------------------------------------------------------------
// Skeleton loader
// ---------------------------------------------------------------------------

function MessageSkeleton() {
  return (
    <div className="flex items-start gap-3 px-4 py-3 animate-pulse">
      <div className="h-5 w-5 rounded bg-secondary" />
      <div className="flex-1 space-y-2">
        <div className="flex items-center justify-between">
          <div className="h-3.5 w-24 rounded bg-secondary" />
          <div className="h-3 w-12 rounded bg-secondary" />
        </div>
        <div className="h-3.5 w-48 rounded bg-secondary" />
        <div className="h-3 w-64 rounded bg-secondary" />
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface MessageListProps {
  messages: InboxMessage[]
  isLoading: boolean
  selectedId: string | null
  onSelect: (id: string) => void
  onReply: (msg: InboxMessage) => void
  onAssign: (msg: InboxMessage) => void
}

export function MessageList({
  messages,
  isLoading,
  selectedId,
  onSelect,
  onReply,
  onAssign,
}: MessageListProps) {
  const markRead = useMarkRead()
  const toggleStar = useToggleStar()
  const archiveMsg = useArchiveMessage()
  const snoozeMsg = useSnoozeMessage()

  const handleSelect = (msg: InboxMessage) => {
    onSelect(msg.id)
    if (!msg.is_read) {
      markRead.mutate(msg.id)
    }
  }

  const handleSnooze = (id: string, isoTimestamp: string) => {
    snoozeMsg.mutate({ id, snoozeUntil: isoTimestamp })
    const displayDate = new Date(isoTimestamp).toLocaleDateString('de-DE', {
      weekday: 'short',
      day: '2-digit',
      month: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
    })
    toast.success(`Verschoben bis ${displayDate}`)
  }

  // Loading state
  if (isLoading) {
    return (
      <div className="flex-1 overflow-y-auto">
        <MessageSkeleton />
        <MessageSkeleton />
        <MessageSkeleton />
        <MessageSkeleton />
      </div>
    )
  }

  // Empty state
  if (messages.length === 0) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <EmptyState
          icon={Inbox}
          title="Posteingang leer"
          description="Keine Nachrichten in dieser Ansicht"
        />
      </div>
    )
  }

  return (
    <div className="flex-1 overflow-y-auto">
      {messages.map((msg) => {
        const ch = channelConfig[msg.channel]
        const ChannelIcon = ch.icon
        const isSelected = selectedId === msg.id
        const isUnread = !msg.is_read

        return (
          <div
            key={msg.id}
            role="button"
            tabIndex={0}
            onClick={() => handleSelect(msg)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' || e.key === ' ') handleSelect(msg)
            }}
            className={`group flex w-full items-start gap-3 border-b border-border-muted px-4 py-3 text-left transition-colors hover:bg-secondary/50 cursor-pointer ${
              isSelected ? 'bg-accent/50' : ''
            }`}
          >
            {/* Channel badge */}
            <div
              className={`mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded ${ch.bg}`}
            >
              <ChannelIcon className={`h-3.5 w-3.5 ${ch.color}`} />
            </div>

            {/* Content */}
            <div className="min-w-0 flex-1">
              {/* Line 1: Sender + timestamp */}
              <div className="flex items-center justify-between gap-2">
                <span
                  className={`text-sm truncate ${
                    isUnread
                      ? 'font-semibold text-foreground'
                      : 'text-text-body'
                  }`}
                >
                  {msg.sender_name}
                </span>
                <span className="text-[10px] text-muted-foreground whitespace-nowrap shrink-0">
                  {formatRelativeTime(msg.received_at)}
                </span>
              </div>
              {/* Line 2: Subject */}
              <p
                className={`text-sm truncate ${
                  isUnread
                    ? 'font-semibold text-foreground'
                    : 'text-text-body'
                }`}
              >
                {msg.subject}
              </p>
              {/* Line 3: Preview */}
              <p className="text-xs text-muted-foreground truncate mt-0.5">
                {msg.preview}
              </p>
              {/* Indicators */}
              <div className="flex items-center gap-2 mt-1">
                {msg.is_starred && (
                  <Star className="h-3 w-3 text-warning fill-warning" />
                )}
                {msg.tags.length > 0 && (
                  <span className="text-[10px] text-muted-foreground">
                    {msg.tags[0]}
                    {msg.tags.length > 1 && ` +${msg.tags.length - 1}`}
                  </span>
                )}
              </div>
            </div>

            {/* Quick actions on hover */}
            <div
              className="flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity shrink-0 self-center"
              onClick={(e) => e.stopPropagation()}
              onKeyDown={(e) => e.stopPropagation()}
            >
              <button
                onClick={() => onReply(msg)}
                className="rounded p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
                title="Antworten"
              >
                <Reply className="h-3.5 w-3.5" />
              </button>
              <SnoozePopover
                onSnooze={(ts) => handleSnooze(msg.id, ts)}
              >
                <button
                  className="rounded p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
                  title="Spaeter erinnern"
                >
                  <Clock className="h-3.5 w-3.5" />
                </button>
              </SnoozePopover>
              <button
                onClick={() => archiveMsg.mutate(msg.id)}
                className="rounded p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
                title="Archivieren"
              >
                <Archive className="h-3.5 w-3.5" />
              </button>
              <button
                onClick={() => onAssign(msg)}
                className="rounded p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
                title="Zuweisen"
              >
                <UserPlus className="h-3.5 w-3.5" />
              </button>
              <button
                onClick={() => toggleStar.mutate(msg.id)}
                className="rounded p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
                title={msg.is_starred ? 'Stern entfernen' : 'Stern vergeben'}
              >
                <Star
                  className={`h-3.5 w-3.5 ${
                    msg.is_starred ? 'fill-warning text-warning' : ''
                  }`}
                />
              </button>
            </div>
          </div>
        )
      })}
    </div>
  )
}

import {
  Mail,
  MessageCircle,
  Bell,
  Star,
} from 'lucide-react'
import type { InboxMessage, InboxChannel } from '@/api/inbox-types'

// ---------------------------------------------------------------------------
// Channel config
// ---------------------------------------------------------------------------

const channelIcon: Record<InboxChannel, { icon: typeof Mail; color: string }> = {
  email: { icon: Mail, color: 'text-blue-500' },
  chat: { icon: MessageCircle, color: 'text-green-500' },
  notification: { icon: Bell, color: 'text-orange-500' },
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

    if (diffMin < 1) return 'Jetzt'
    if (diffMin < 60) return `${diffMin}m`
    if (diffH < 24) return `${diffH}h`
    if (diffD < 7) return `${diffD}d`
    return date.toLocaleDateString('de-CH', { day: '2-digit', month: '2-digit' })
  } catch {
    return dateStr
  }
}

function getInitials(name: string): string {
  return name
    .split(' ')
    .map((n) => n[0])
    .join('')
    .toUpperCase()
    .slice(0, 2)
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface ConversationListItemProps {
  message: InboxMessage
  isSelected: boolean
  onSelect: (id: string) => void
}

export function ConversationListItem({
  message: msg,
  isSelected,
  onSelect,
}: ConversationListItemProps) {
  const ch = channelIcon[msg.channel]
  const ChannelIcon = ch.icon
  const isUnread = !msg.is_read

  return (
    <button
      onClick={() => onSelect(msg.id)}
      className={`flex w-full items-start gap-3 px-3 py-3 text-left transition-colors border-b border-border/50 ${
        isSelected
          ? 'bg-primary/5'
          : isUnread
            ? 'bg-accent/30 hover:bg-accent/50'
            : 'hover:bg-accent/50'
      }`}
    >
      {/* Avatar */}
      <div className="relative shrink-0">
        <div className="flex h-9 w-9 items-center justify-center rounded-full bg-primary-light text-xs font-medium text-primary">
          {getInitials(msg.sender_name)}
        </div>
        {/* Channel badge */}
        <div className="absolute -bottom-0.5 -right-0.5 flex h-4 w-4 items-center justify-center rounded-full bg-background border border-border">
          <ChannelIcon className={`h-2.5 w-2.5 ${ch.color}`} />
        </div>
      </div>

      {/* Content */}
      <div className="min-w-0 flex-1">
        {/* Row 1: Name + time */}
        <div className="flex items-center justify-between gap-2">
          <span className={`text-sm truncate ${isUnread ? 'font-semibold text-foreground' : 'text-foreground/90'}`}>
            {msg.sender_name}
          </span>
          <span className="text-[10px] text-muted-foreground whitespace-nowrap shrink-0">
            {formatRelativeTime(msg.received_at)}
          </span>
        </div>

        {/* Row 2: Subject */}
        <p className={`text-[13px] truncate ${isUnread ? 'font-medium text-foreground' : 'text-foreground/80'}`}>
          {msg.subject}
        </p>

        {/* Row 3: Preview */}
        <p className="text-xs text-muted-foreground truncate mt-0.5">
          {msg.preview}
        </p>

        {/* Row 4: Indicators */}
        <div className="flex items-center gap-1.5 mt-1.5">
          {/* Star indicator */}
          {msg.is_starred && (
            <Star className="h-3 w-3 shrink-0 text-warning fill-warning" />
          )}

          {/* Assigned */}
          {msg.assigned_to && (
            <span className="rounded bg-secondary px-1.5 py-0.5 text-[10px] text-muted-foreground truncate max-w-[80px]">
              {msg.assigned_to}
            </span>
          )}

          {/* Tags (first 2) */}
          {msg.tags.slice(0, 2).map((tag) => (
            <span key={tag} className="rounded bg-secondary/70 px-1.5 py-0.5 text-[10px] text-muted-foreground">
              {tag}
            </span>
          ))}

          {/* Unread indicator */}
          {isUnread && (
            <span className="ml-auto flex h-4 min-w-4 items-center justify-center rounded-full bg-primary px-1 text-[10px] font-semibold text-primary-foreground">
              !
            </span>
          )}
        </div>
      </div>
    </button>
  )
}

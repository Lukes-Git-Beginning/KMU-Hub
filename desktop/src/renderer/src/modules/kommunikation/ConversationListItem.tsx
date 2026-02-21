import {
  Mail,
  MessageCircle,
  Globe,
  Headphones,
  Users,
  Paperclip,
  AlertTriangle,
  ArrowUp,
} from 'lucide-react'
import type { Conversation, CommunicationChannel, ConversationPriority } from '@/types/communication'

// ---------------------------------------------------------------------------
// Channel config
// ---------------------------------------------------------------------------

const channelIcon: Record<CommunicationChannel, { icon: typeof Mail; color: string }> = {
  email: { icon: Mail, color: 'text-blue-500' },
  teams: { icon: Users, color: 'text-violet-500' },
  whatsapp: { icon: MessageCircle, color: 'text-green-500' },
  widget: { icon: Globe, color: 'text-orange-500' },
  portal: { icon: Headphones, color: 'text-teal-500' },
}

const priorityConfig: Record<ConversationPriority, { icon?: typeof ArrowUp; color: string; label: string }> = {
  low: { color: '', label: '' },
  normal: { color: '', label: '' },
  high: { icon: ArrowUp, color: 'text-warning', label: 'Hoch' },
  urgent: { icon: AlertTriangle, color: 'text-destructive', label: 'Dringend' },
}

const statusDot: Record<string, string> = {
  open: 'bg-success',
  pending: 'bg-warning',
  resolved: 'bg-blue-500',
  closed: 'bg-muted-foreground',
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
  conversation: Conversation
  isSelected: boolean
  onSelect: (id: string) => void
}

export function ConversationListItem({
  conversation: conv,
  isSelected,
  onSelect,
}: ConversationListItemProps) {
  const ch = channelIcon[conv.channel]
  const ChannelIcon = ch.icon
  const prio = priorityConfig[conv.priority]
  const PrioIcon = prio.icon
  const hasAttachments = conv.messages.some((m) => m.attachments.length > 0)
  const isUnread = conv.unreadCount > 0

  return (
    <button
      onClick={() => onSelect(conv.id)}
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
          {getInitials(conv.contactName)}
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
            {conv.contactName}
          </span>
          <span className="text-[10px] text-muted-foreground whitespace-nowrap shrink-0">
            {formatRelativeTime(conv.lastMessageAt)}
          </span>
        </div>

        {/* Row 2: Subject */}
        <p className={`text-[13px] truncate ${isUnread ? 'font-medium text-foreground' : 'text-foreground/80'}`}>
          {conv.subject}
        </p>

        {/* Row 3: Preview */}
        <p className="text-xs text-muted-foreground truncate mt-0.5">
          {conv.lastMessage}
        </p>

        {/* Row 4: Indicators */}
        <div className="flex items-center gap-1.5 mt-1.5">
          {/* Status dot */}
          <span className={`h-1.5 w-1.5 rounded-full shrink-0 ${statusDot[conv.status] ?? ''}`} />

          {/* Priority */}
          {PrioIcon && (
            <PrioIcon className={`h-3 w-3 shrink-0 ${prio.color}`} />
          )}

          {/* Assigned */}
          {conv.assignedTo && (
            <span className="rounded bg-secondary px-1.5 py-0.5 text-[10px] text-muted-foreground truncate max-w-[80px]">
              {conv.assignedTo}
            </span>
          )}

          {/* Tags (first 2) */}
          {conv.tags.slice(0, 2).map((tag) => (
            <span key={tag} className="rounded bg-secondary/70 px-1.5 py-0.5 text-[10px] text-muted-foreground">
              {tag}
            </span>
          ))}

          {/* Attachment indicator */}
          {hasAttachments && (
            <Paperclip className="h-3 w-3 shrink-0 text-muted-foreground" />
          )}

          {/* Unread badge */}
          {isUnread && (
            <span className="ml-auto flex h-4 min-w-4 items-center justify-center rounded-full bg-primary px-1 text-[10px] font-semibold text-primary-foreground">
              {conv.unreadCount}
            </span>
          )}

          {/* Company */}
          {conv.contactCompany && (
            <span className="ml-auto text-[10px] text-muted-foreground truncate max-w-[100px]">
              {conv.contactCompany}
            </span>
          )}
        </div>
      </div>
    </button>
  )
}

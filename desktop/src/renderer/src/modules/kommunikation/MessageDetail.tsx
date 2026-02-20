/**
 * Right column: full detail view of the selected inbox message.
 *
 * Adapts per channel type:
 * - Email: text area reply box
 * - Chat: single-line input
 * - Notification: action buttons or "Keine Antwort moeglich"
 *
 * Shows deep link to source module, tags, team assignment info.
 */
import { useState } from 'react'
import {
  X,
  Star,
  Archive,
  UserPlus,
  ExternalLink,
  Mail,
  MessageSquare,
  Bell,
  Send,
} from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import type { InboxMessage, InboxChannel } from '@/api/inbox-types'
import {
  useToggleStar,
  useArchiveMessage,
  useReplyToMessage,
  useClaimMessage,
} from '@/api/hooks/useInbox'
import { useAuthStore } from '@/stores/auth'
import { SnoozePopover } from './SnoozePopover'
import { useSnoozeMessage } from '@/api/hooks/useInbox'

// ---------------------------------------------------------------------------
// Channel config
// ---------------------------------------------------------------------------

const channelBadge: Record<
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

function formatFullTimestamp(dateStr: string): string {
  try {
    const d = new Date(dateStr)
    return d.toLocaleDateString('de-DE', {
      weekday: 'long',
      day: '2-digit',
      month: 'long',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  } catch {
    return dateStr
  }
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface MessageDetailProps {
  message: InboxMessage
  onClose: () => void
}

export function MessageDetail({ message, onClose }: MessageDetailProps) {
  const navigate = useNavigate()
  const user = useAuthStore((s) => s.user)
  const [replyText, setReplyText] = useState('')

  const toggleStar = useToggleStar()
  const archiveMsg = useArchiveMessage()
  const replyMutation = useReplyToMessage()
  const claimMsg = useClaimMessage()
  const snoozeMsg = useSnoozeMessage()

  const ch = channelBadge[message.channel]
  const ChannelIcon = ch.icon

  const handleReply = () => {
    if (!replyText.trim()) return
    replyMutation.mutate(
      { id: message.id, body: replyText.trim() },
      {
        onSuccess: () => {
          setReplyText('')
        },
      },
    )
  }

  const handleSnooze = (isoTimestamp: string) => {
    snoozeMsg.mutate({ id: message.id, snoozeUntil: isoTimestamp })
    const displayDate = new Date(isoTimestamp).toLocaleDateString('de-DE', {
      weekday: 'short',
      day: '2-digit',
      month: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
    })
    toast.success(`Verschoben bis ${displayDate}`)
  }

  const handleOpenSource = () => {
    if (message.deep_link) {
      navigate(message.deep_link)
    }
  }

  const isAssignedToMe = message.assigned_to === user?.id
  const isInTeam = !!message.team_inbox_id

  return (
    <div className="flex flex-col h-full w-[480px] shrink-0 border-l border-border bg-card">
      {/* Header */}
      <div className="flex items-center gap-3 border-b border-border px-4 py-3">
        {/* Channel badge */}
        <div
          className={`flex h-7 w-7 shrink-0 items-center justify-center rounded ${ch.bg}`}
        >
          <ChannelIcon className={`h-4 w-4 ${ch.color}`} />
        </div>
        <span className="text-xs text-muted-foreground">{ch.label}</span>

        <div className="flex-1" />

        {/* Action buttons */}
        <button
          onClick={() => toggleStar.mutate(message.id)}
          className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
          title={message.is_starred ? 'Stern entfernen' : 'Stern vergeben'}
        >
          <Star
            className={`h-4 w-4 ${
              message.is_starred ? 'fill-warning text-warning' : ''
            }`}
          />
        </button>
        <button
          onClick={() => archiveMsg.mutate(message.id)}
          className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
          title="Archivieren"
        >
          <Archive className="h-4 w-4" />
        </button>
        <SnoozePopover onSnooze={handleSnooze}>
          <button
            className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
            title="Spaeter erinnern"
          >
            <svg
              className="h-4 w-4"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth={2}
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <circle cx="12" cy="12" r="10" />
              <polyline points="12 6 12 12 16 14" />
            </svg>
          </button>
        </SnoozePopover>
        <button
          onClick={onClose}
          className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
          title="Schliessen"
        >
          <X className="h-4 w-4" />
        </button>
      </div>

      {/* Body */}
      <div className="flex-1 overflow-y-auto px-4 py-4 space-y-4">
        {/* Sender info */}
        <div>
          <div className="flex items-center gap-2">
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-primary-light text-xs font-medium text-primary">
              {message.sender_name
                .split(' ')
                .map((n) => n[0])
                .join('')
                .toUpperCase()
                .slice(0, 2)}
            </div>
            <div className="min-w-0 flex-1">
              <p className="text-sm font-medium text-foreground truncate">
                {message.sender_name}
              </p>
              {message.sender_email && (
                <p className="text-xs text-muted-foreground truncate">
                  {message.sender_email}
                </p>
              )}
            </div>
          </div>
          <p className="text-xs text-muted-foreground mt-1">
            {formatFullTimestamp(message.received_at)}
          </p>
        </div>

        {/* Subject */}
        <h2 className="text-base font-medium text-foreground">
          {message.subject}
        </h2>

        {/* Full preview text */}
        <div className="text-sm text-text-body leading-relaxed whitespace-pre-line">
          {message.preview}
        </div>

        {/* Tags */}
        {message.tags.length > 0 && (
          <div className="flex flex-wrap gap-1.5">
            {message.tags.map((tag) => (
              <span
                key={tag}
                className="inline-flex items-center rounded-full bg-secondary px-2 py-0.5 text-[10px] font-medium text-secondary-foreground"
              >
                {tag}
              </span>
            ))}
          </div>
        )}

        {/* Team inbox assignment info */}
        {isInTeam && (
          <div className="rounded-lg border border-border px-3 py-2">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <UserPlus className="h-4 w-4 text-muted-foreground" />
                <span className="text-xs text-muted-foreground">
                  {message.assigned_to
                    ? isAssignedToMe
                      ? 'Dir zugewiesen'
                      : `Zugewiesen an: ${message.assigned_to}`
                    : 'Nicht zugewiesen'}
                </span>
              </div>
              {!isAssignedToMe && (
                <button
                  onClick={() => claimMsg.mutate(message.id)}
                  className="rounded-md px-2 py-1 text-xs font-medium text-primary hover:bg-primary/10 transition-colors"
                >
                  Uebernehmen
                </button>
              )}
            </div>
          </div>
        )}

        {/* Deep link */}
        {message.deep_link && (
          <button
            onClick={handleOpenSource}
            className="flex items-center gap-2 rounded-lg border border-border px-3 py-2 text-sm text-foreground hover:bg-secondary transition-colors"
          >
            <ExternalLink className="h-4 w-4 text-muted-foreground" />
            Im Modul oeffnen
          </button>
        )}
      </div>

      {/* Inline reply (bottom) */}
      <div className="border-t border-border px-4 py-3">
        {message.channel === 'email' && (
          <div className="space-y-2">
            <textarea
              value={replyText}
              onChange={(e) => setReplyText(e.target.value)}
              placeholder="Antwort schreiben..."
              rows={3}
              className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring resize-none"
            />
            <div className="flex justify-end">
              <button
                onClick={handleReply}
                disabled={!replyText.trim() || replyMutation.isPending}
                className="flex items-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50"
              >
                <Send className="h-3.5 w-3.5" />
                Antworten
              </button>
            </div>
          </div>
        )}

        {message.channel === 'chat' && (
          <div className="flex gap-2">
            <input
              type="text"
              value={replyText}
              onChange={(e) => setReplyText(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && !e.shiftKey) {
                  e.preventDefault()
                  handleReply()
                }
              }}
              placeholder="Nachricht senden..."
              className="flex-1 rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
            <button
              onClick={handleReply}
              disabled={!replyText.trim() || replyMutation.isPending}
              className="rounded-md bg-primary p-2 text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50"
            >
              <Send className="h-4 w-4" />
            </button>
          </div>
        )}

        {message.channel === 'notification' && (
          <p className="text-xs text-muted-foreground text-center py-1">
            Keine Antwort moeglich
          </p>
        )}
      </div>
    </div>
  )
}

/**
 * Team Chat widget — recent messages from team channels.
 *
 * Wired to useChannels() which includes last_message and unread_count
 * per channel, giving us a cross-channel "recent messages" view
 * without needing a dedicated endpoint.
 */
import { memo, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Hash, ArrowRight, Loader2 } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { useChannels } from '@/api/hooks/useChannels'
import type { WidgetProps } from '@/components/widgets/WidgetRegistry'

/** Build initials from first + last name. */
function getInitials(firstName?: string, lastName?: string): string {
  const f = firstName?.[0] ?? ''
  const l = lastName?.[0] ?? ''
  return (f + l).toUpperCase() || '?'
}

/** Format ISO timestamp to relative short time (e.g. "14:30"). */
function formatTime(iso?: string): string {
  if (!iso) return ''
  try {
    return new Date(iso).toLocaleTimeString('de-DE', { hour: '2-digit', minute: '2-digit' })
  } catch {
    return ''
  }
}

const MAX_MESSAGES = 7

function TeamChat(_props: WidgetProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { data: channelData, isLoading } = useChannels()
  const channels = useMemo(() => channelData?.channels ?? [], [channelData?.channels])

  const recentMessages = useMemo(() => {
    return channels
      .filter((ch) => ch.last_message?.content)
      .map((ch) => ({
        id: ch.last_message!.id ?? ch.id ?? '',
        senderName: [ch.last_message!.sender_first_name, ch.last_message!.sender_last_name]
          .filter(Boolean)
          .join(' ') || t('dashboard.teamChat.unknown'),
        senderInitials: getInitials(
          ch.last_message!.sender_first_name,
          ch.last_message!.sender_last_name,
        ),
        channel: ch.name ?? 'DM',
        text: ch.last_message!.content ?? '',
        time: formatTime(ch.last_message!.created_at),
        unread: (ch.unread_count ?? 0) > 0,
        sortKey: ch.last_message!.created_at ?? '',
      }))
      .sort((a, b) => b.sortKey.localeCompare(a.sortKey))
      .slice(0, MAX_MESSAGES)
  }, [channels, t])

  const unreadCount = recentMessages.filter((m) => m.unread).length

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (recentMessages.length === 0) {
    return (
      <div className="flex h-full flex-col">
        <div className="flex items-center justify-between px-4 pt-4 pb-2">
          <span className="text-xs text-muted-foreground">{t('dashboard.teamChat.recentMessages')}</span>
          <button
            onClick={() => navigate('/chat')}
            className="flex items-center gap-0.5 text-[10px] text-primary hover:underline"
          >
            {t('dashboard.notifications.showAll')} <ArrowRight className="h-2.5 w-2.5" />
          </button>
        </div>
        <div className="flex flex-1 items-center justify-center px-4">
          <p className="text-xs text-muted-foreground">{t('dashboard.teamChat.noMessages')}</p>
        </div>
      </div>
    )
  }

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="flex items-center justify-between px-4 pt-4 pb-2">
        <span className="text-xs text-muted-foreground">
          {unreadCount > 0 && (
            <span className="inline-flex h-4 min-w-[16px] items-center justify-center rounded-full bg-destructive px-1 text-[10px] font-bold text-white mr-1.5">
              {unreadCount}
            </span>
          )}
          {t('dashboard.teamChat.recentMessages')}
        </span>
        <button
          onClick={() => navigate('/chat')}
          className="flex items-center gap-0.5 text-[10px] text-primary hover:underline"
        >
          Alle anzeigen <ArrowRight className="h-2.5 w-2.5" />
        </button>
      </div>

      {/* Messages */}
      <div className="flex-1 overflow-auto divide-y divide-border">
        {recentMessages.map((msg) => (
          <div
            key={msg.id}
            className={`flex items-start gap-3 px-4 py-2.5 cursor-pointer transition-colors hover:bg-accent/50 ${
              msg.unread ? 'bg-primary/5' : ''
            }`}
            onClick={() => navigate('/chat')}
          >
            <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-primary/10 text-[10px] font-semibold text-primary mt-0.5">
              {msg.senderInitials}
            </div>
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-1.5">
                <span className="text-xs font-semibold text-foreground">{msg.senderName}</span>
                <span className="flex items-center gap-0.5 text-[10px] text-muted-foreground">
                  <Hash className="h-2.5 w-2.5" />{msg.channel}
                </span>
                <span className="text-[10px] text-muted-foreground ml-auto shrink-0">{msg.time}</span>
              </div>
              <p className="text-xs text-muted-foreground truncate mt-0.5">{msg.text}</p>
            </div>
            {msg.unread && (
              <span className="mt-2 h-2 w-2 shrink-0 rounded-full bg-primary" />
            )}
          </div>
        ))}
      </div>
    </div>
  )
}

export default memo(TeamChat)

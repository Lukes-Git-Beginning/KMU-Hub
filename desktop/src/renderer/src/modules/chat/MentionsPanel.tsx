/**
 * Mentions inbox panel (right slot).
 *
 * Lists every message where the current user was mentioned across all
 * channels via useUserMentions. Each item jumps to its channel on click.
 */
import { useTranslation } from 'react-i18next'
import { AtSign, X, Loader2 } from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'
import { de } from 'date-fns/locale'
import { useUserMentions, type MentionDetail } from '@/api/hooks/useMentions'
import { useChannels } from '@/api/hooks/useChannels'

interface MentionsPanelProps {
  onClose: () => void
  onSelectChannel: (channelId: string) => void
}

export function MentionsPanel({ onClose, onSelectChannel }: MentionsPanelProps) {
  const { t } = useTranslation()
  const { data, isFetching } = useUserMentions()
  const { data: channelsData } = useChannels()

  const mentions = data?.mentions ?? []
  const channelName = (id?: string) =>
    channelsData?.channels?.find((c) => c.id === id)?.name ?? ''

  return (
    <div className="flex h-full flex-col border-l border-border bg-card">
      <div className="flex items-center justify-between border-b border-border px-4 py-3">
        <div className="flex items-center gap-2">
          <AtSign className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-semibold text-foreground">{t('chat.mentions.title')}</h3>
        </div>
        <button onClick={onClose} className="text-muted-foreground hover:text-foreground" aria-label={t('common.close')}>
          <X className="h-4 w-4" />
        </button>
      </div>

      <div className="flex-1 overflow-y-auto px-2 py-2">
        {isFetching && mentions.length === 0 ? (
          <div className="flex justify-center py-6">
            <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
          </div>
        ) : mentions.length === 0 ? (
          <div className="px-3 py-10 text-center">
            <AtSign className="mx-auto h-8 w-8 text-muted-foreground/40" />
            <p className="mt-3 text-xs text-muted-foreground">{t('chat.mentions.empty')}</p>
          </div>
        ) : (
          <div className="space-y-1">
            <p className="px-2 pb-1 text-[10px] uppercase tracking-wider text-muted-foreground">
              {t('chat.mentions.count', { count: mentions.length })}
            </p>
            {mentions.map((m: MentionDetail, i) => (
              <button
                key={`${m.message_id}-${i}`}
                className="w-full rounded-lg px-2.5 py-2 text-left hover:bg-secondary/50"
                onClick={() => m.channel_id && onSelectChannel(m.channel_id)}
              >
                <div className="flex items-baseline justify-between gap-2">
                  <span className="truncate text-xs font-medium text-foreground">
                    {[m.sender_first_name, m.sender_last_name].filter(Boolean).join(' ') || t('chat.unknown')}
                  </span>
                  <span className="shrink-0 text-[10px] text-muted-foreground">
                    {m.created_at ? formatDistanceToNow(new Date(m.created_at), { addSuffix: true, locale: de }) : ''}
                  </span>
                </div>
                <div className="mt-0.5 flex items-center gap-1.5">
                  {channelName(m.channel_id) && (
                    <span className="shrink-0 text-[10px] text-primary">#{channelName(m.channel_id)}</span>
                  )}
                  {(m.mention_type === 'everyone' || m.mention_type === 'channel') && (
                    <span className="shrink-0 rounded-sm bg-warning/20 px-1 text-[9px] font-medium text-foreground">
                      {t('chat.mentions.broadcast')}
                    </span>
                  )}
                </div>
                <p className="mt-0.5 text-xs text-muted-foreground line-clamp-2">{m.content}</p>
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

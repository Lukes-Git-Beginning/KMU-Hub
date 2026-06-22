/**
 * Unified unread inbox panel — right slot (KO-7).
 *
 * Lists every channel and DM with unread messages so the user can triage the
 * whole workspace from one place. Click an entry to open the channel (which
 * marks it read); "mark all read" clears everything.
 */
import { useTranslation } from 'react-i18next'
import { Inbox, Hash, AtSign, X, Loader2, CheckCheck } from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'
import { de } from 'date-fns/locale'
import { useUnreadInbox, useMarkAllRead, type UnreadInboxEntry } from '@/api/hooks/useUnreadInbox'

interface UnreadInboxPanelProps {
  onClose: () => void
  onSelectChannel: (channelId: string) => void
}

export function UnreadInboxPanel({ onClose, onSelectChannel }: UnreadInboxPanelProps) {
  const { t } = useTranslation()
  const { data, isFetching } = useUnreadInbox()
  const markAllRead = useMarkAllRead()

  const entries = data?.entries ?? []
  const total = entries.reduce((sum, e) => sum + e.unread_count, 0)

  return (
    <div className="flex h-full flex-col border-l border-border bg-card">
      <div className="flex items-center justify-between border-b border-border px-4 py-3">
        <div className="flex items-center gap-2">
          <Inbox className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-semibold text-foreground">{t('chat.unread.title')}</h3>
        </div>
        <div className="flex items-center gap-1">
          {entries.length > 0 && (
            <button
              onClick={() => markAllRead.mutate()}
              className="flex items-center gap-1 rounded-md px-1.5 py-1 text-[11px] text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
            >
              <CheckCheck className="h-3.5 w-3.5" />
              {t('chat.unread.markAllRead')}
            </button>
          )}
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground" aria-label={t('common.close')}>
            <X className="h-4 w-4" />
          </button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto px-2 py-2">
        {isFetching && entries.length === 0 ? (
          <div className="flex justify-center py-6">
            <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
          </div>
        ) : entries.length === 0 ? (
          <div className="px-3 py-10 text-center">
            <CheckCheck className="mx-auto h-8 w-8 text-success/50" />
            <p className="mt-3 text-xs text-muted-foreground">{t('chat.unread.empty')}</p>
          </div>
        ) : (
          <div className="space-y-1">
            <p className="px-2 pb-1 text-[10px] uppercase tracking-wider text-muted-foreground">
              {t('chat.unread.count', { count: total })}
            </p>
            {entries.map((entry: UnreadInboxEntry) => {
              const last = entry.messages[entry.messages.length - 1]
              return (
                <button
                  key={entry.channel_id}
                  className="w-full rounded-lg px-2.5 py-2 text-left transition-colors hover:bg-secondary/50"
                  onClick={() => onSelectChannel(entry.channel_id)}
                >
                  <div className="flex items-center justify-between gap-2">
                    <span className="flex min-w-0 items-center gap-1.5">
                      {entry.is_dm ? (
                        <AtSign className="h-3 w-3 shrink-0 text-muted-foreground" />
                      ) : (
                        <Hash className="h-3 w-3 shrink-0 text-muted-foreground" />
                      )}
                      <span className="truncate text-xs font-medium text-foreground">{entry.channel_name}</span>
                    </span>
                    <span className="flex h-4 min-w-4 shrink-0 items-center justify-center rounded-full bg-destructive px-1 text-[10px] font-medium text-destructive-foreground">
                      {entry.unread_count}
                    </span>
                  </div>
                  {last && (
                    <div className="mt-0.5">
                      <span className="text-[10px] text-muted-foreground">
                        {[last.sender_first_name, last.sender_last_name].filter(Boolean).join(' ') || t('chat.unknown')}
                        {last.created_at ? ` · ${formatDistanceToNow(new Date(last.created_at), { addSuffix: true, locale: de })}` : ''}
                      </span>
                      <p className="text-xs text-muted-foreground line-clamp-1">{last.content}</p>
                    </div>
                  )}
                </button>
              )
            })}
          </div>
        )}
      </div>
    </div>
  )
}

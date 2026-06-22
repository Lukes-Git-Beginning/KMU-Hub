/**
 * Bookmarks (saved messages) panel — right slot.
 *
 * Lists every message the current user bookmarked across all channels.
 * Click jumps to the message's channel; the bookmark icon removes it.
 */
import { useTranslation } from 'react-i18next'
import { Bookmark, X, Loader2 } from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'
import { de } from 'date-fns/locale'
import { useBookmarks, useToggleBookmark } from '@/api/hooks/useBookmarks'
import { useChannels } from '@/api/hooks/useChannels'
import type { components } from '@/api/types'

type MessageInfo = components['schemas']['MessageInfo']

interface BookmarksPanelProps {
  onClose: () => void
  onSelectChannel: (channelId: string) => void
}

export function BookmarksPanel({ onClose, onSelectChannel }: BookmarksPanelProps) {
  const { t } = useTranslation()
  const { data, isFetching } = useBookmarks()
  const { data: channelsData } = useChannels()
  const toggleBookmark = useToggleBookmark()

  const bookmarks = data?.messages ?? []
  const channelName = (id?: string) =>
    channelsData?.channels?.find((c) => c.id === id)?.name ?? ''

  return (
    <div className="flex h-full flex-col border-l border-border bg-card">
      <div className="flex items-center justify-between border-b border-border px-4 py-3">
        <div className="flex items-center gap-2">
          <Bookmark className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-semibold text-foreground">{t('chat.bookmarks.title')}</h3>
        </div>
        <button onClick={onClose} className="text-muted-foreground hover:text-foreground" aria-label={t('common.close')}>
          <X className="h-4 w-4" />
        </button>
      </div>

      <div className="flex-1 overflow-y-auto px-2 py-2">
        {isFetching && bookmarks.length === 0 ? (
          <div className="flex justify-center py-6">
            <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
          </div>
        ) : bookmarks.length === 0 ? (
          <div className="px-3 py-10 text-center">
            <Bookmark className="mx-auto h-8 w-8 text-muted-foreground/40" />
            <p className="mt-3 text-xs text-muted-foreground">{t('chat.bookmarks.empty')}</p>
          </div>
        ) : (
          <div className="space-y-1">
            <p className="px-2 pb-1 text-[10px] uppercase tracking-wider text-muted-foreground">
              {t('chat.bookmarks.count', { count: bookmarks.length })}
            </p>
            {bookmarks.map((m: MessageInfo, i) => (
              <div
                key={`${m.id}-${i}`}
                role="button"
                tabIndex={0}
                className="group w-full cursor-pointer rounded-lg px-2.5 py-2 text-left hover:bg-secondary/50"
                onClick={() => m.channel_id && onSelectChannel(m.channel_id)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && m.channel_id) onSelectChannel(m.channel_id)
                }}
              >
                <div className="flex items-baseline justify-between gap-2">
                  <span className="truncate text-xs font-medium text-foreground">
                    {[m.sender_first_name, m.sender_last_name].filter(Boolean).join(' ') || t('chat.unknown')}
                  </span>
                  <div className="flex shrink-0 items-center gap-1.5">
                    <span className="text-[10px] text-muted-foreground">
                      {m.created_at ? formatDistanceToNow(new Date(m.created_at), { addSuffix: true, locale: de }) : ''}
                    </span>
                    <button
                      className="text-primary opacity-0 transition-opacity group-hover:opacity-100"
                      onClick={(e) => {
                        e.stopPropagation()
                        if (m.id) toggleBookmark.mutate(m.id)
                      }}
                      aria-label={t('chat.bookmarks.remove')}
                    >
                      <Bookmark className="h-3.5 w-3.5 fill-current" />
                    </button>
                  </div>
                </div>
                {channelName(m.channel_id) && (
                  <span className="mt-0.5 block text-[10px] text-primary">#{channelName(m.channel_id)}</span>
                )}
                <p className="mt-0.5 text-xs text-muted-foreground line-clamp-2">{m.content}</p>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

/**
 * Channel message search panel (right slot).
 *
 * Full-text search across messages via useChatSearch. Scoped to the current
 * channel by default; a toggle widens it to all channels. Highlights matches
 * from the server's <mark> snippet.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Search, X, Loader2 } from 'lucide-react'
import { useChatSearch } from '@/api/hooks/useChatSearch'
import { Input } from '@/components/ui/input'

interface ChannelSearchPanelProps {
  channelId: string
  onClose: () => void
  onJumpToMessage?: (channelId: string, messageId: string) => void
}

export function ChannelSearchPanel({ channelId, onClose, onJumpToMessage }: ChannelSearchPanelProps) {
  const { t } = useTranslation()
  const [query, setQuery] = useState('')
  const [allChannels, setAllChannels] = useState(false)

  const { data, isFetching } = useChatSearch(query, allChannels ? undefined : channelId)
  const results = data?.results ?? []
  const hasQuery = query.trim().length >= 2

  return (
    <div className="flex h-full flex-col border-l border-border bg-card">
      <div className="flex items-center justify-between border-b border-border px-4 py-3">
        <h3 className="text-sm font-semibold text-foreground">{t('chat.search.title')}</h3>
        <button onClick={onClose} className="text-muted-foreground hover:text-foreground" aria-label={t('common.close')}>
          <X className="h-4 w-4" />
        </button>
      </div>

      <div className="space-y-2 border-b border-border px-3 py-2.5">
        <div className="relative">
          <Search className="absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
          <Input
            autoFocus
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={t('chat.search.placeholder')}
            className="h-8 pl-8 text-xs"
          />
        </div>
        <label className="flex items-center gap-1.5 text-[11px] text-muted-foreground">
          <input
            type="checkbox"
            checked={allChannels}
            onChange={(e) => setAllChannels(e.target.checked)}
            className="h-3 w-3 accent-primary"
          />
          {t('chat.search.allChannels')}
        </label>
      </div>

      <div className="flex-1 overflow-y-auto px-2 py-2">
        {!hasQuery ? (
          <p className="px-2 py-6 text-center text-xs text-muted-foreground">{t('chat.search.hint')}</p>
        ) : isFetching ? (
          <div className="flex justify-center py-6">
            <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
          </div>
        ) : results.length === 0 ? (
          <p className="px-2 py-6 text-center text-xs text-muted-foreground">{t('chat.search.noResults', { query })}</p>
        ) : (
          <div className="space-y-1">
            <p className="px-2 pb-1 text-[10px] uppercase tracking-wider text-muted-foreground">
              {t('chat.search.resultCount', { count: results.length })}
            </p>
            {results.map((r) => (
              <button
                key={`${r.id}-${r.channel_id}`}
                type="button"
                className="w-full rounded-lg px-2.5 py-2 text-left transition-colors hover:bg-secondary/50"
                onClick={() => r.channel_id && r.id && onJumpToMessage?.(r.channel_id, r.id)}
              >
                <div className="flex items-baseline justify-between gap-2">
                  <span className="truncate text-xs font-medium text-foreground">
                    {[r.first_name, r.last_name].filter(Boolean).join(' ') || t('chat.unknown')}
                  </span>
                  {allChannels && r.channel_name && (
                    <span className="shrink-0 text-[10px] text-muted-foreground">#{r.channel_name}</span>
                  )}
                </div>
                <p className="mt-0.5 text-xs text-muted-foreground line-clamp-2">{renderSnippet(r.snippet ?? '')}</p>
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

/** Render a server snippet that wraps matches in <mark> — safely, without dangerouslySetInnerHTML. */
function renderSnippet(snippet: string): React.ReactNode {
  const parts = snippet.split(/(<mark>.*?<\/mark>)/g)
  return parts.map((part, i) => {
    const m = part.match(/^<mark>(.*?)<\/mark>$/)
    if (m) {
      return (
        <mark key={i} className="rounded-sm bg-warning/30 px-0.5 text-foreground">
          {m[1]}
        </mark>
      )
    }
    return <span key={i}>{part}</span>
  })
}

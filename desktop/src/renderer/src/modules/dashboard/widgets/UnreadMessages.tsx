/**
 * UnreadMessages widget -- lists channels and DMs with unread messages.
 *
 * Each row shows channel name, unread count badge, last message preview,
 * and timestamp. Click navigates to the chat module with the channel selected.
 */
import { memo } from 'react'
import { useNavigate } from 'react-router-dom'
import { MessageSquare } from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'
import { de } from 'date-fns/locale'
import { useChannels } from '@/api/hooks/useChannels'
import { Skeleton } from '@/components/ui/skeleton'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import type { WidgetProps } from '@/components/widgets/WidgetRegistry'

function UnreadMessages(_props: WidgetProps) {
  const navigate = useNavigate()
  const { data, isLoading, error, refetch } = useChannels()

  // Filter to channels with unread messages and take top 5
  const channels = (data?.channels ?? [])
    .filter((ch) => (ch.unread_count ?? 0) > 0)
    .slice(0, 5)

  if (error) {
    return (
      <div className="flex h-full items-center justify-center p-4">
        <div className="text-center">
          <p className="text-sm text-muted-foreground">Fehler beim Laden</p>
          <Button variant="ghost" size="sm" className="mt-2" onClick={() => refetch()}>
            Erneut versuchen
          </Button>
        </div>
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="space-y-3 p-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i} className="flex items-center gap-3">
            <Skeleton className="h-4 w-4 rounded" />
            <div className="flex-1 space-y-1">
              <Skeleton className="h-3.5 w-1/2" />
              <Skeleton className="h-3 w-3/4" />
            </div>
            <Skeleton className="h-5 w-5 rounded-full" />
          </div>
        ))}
      </div>
    )
  }

  if (channels.length === 0) {
    return (
      <div className="flex h-full items-center justify-center p-4">
        <div className="text-center">
          <MessageSquare className="mx-auto h-8 w-8 text-muted-foreground/50" />
          <p className="mt-2 text-sm text-muted-foreground">Alle Nachrichten gelesen</p>
        </div>
      </div>
    )
  }

  return (
    <div className="divide-y">
      {channels.map((channel) => {
        const lastMsg = channel.last_message
        const preview = lastMsg?.content
          ? lastMsg.content.length > 60
            ? lastMsg.content.slice(0, 60) + '...'
            : lastMsg.content
          : ''
        const timeAgo = lastMsg?.created_at
          ? formatDistanceToNow(new Date(lastMsg.created_at), { addSuffix: true, locale: de })
          : ''

        const displayName = channel.is_dm
          ? (lastMsg?.sender_first_name
              ? `${lastMsg.sender_first_name} ${lastMsg.sender_last_name ?? ''}`
              : channel.name ?? 'DM')
          : `# ${channel.name}`

        return (
          <div
            key={channel.id}
            className="flex cursor-pointer items-start gap-3 px-4 py-2.5 transition-colors hover:bg-accent/50"
            onClick={() => navigate(`/chat?channel=${channel.id}`)}
          >
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2">
                <p className="truncate text-sm font-medium">{displayName}</p>
                <Badge variant="destructive" className="h-5 min-w-5 shrink-0 px-1.5 text-xs">
                  {channel.unread_count}
                </Badge>
              </div>
              {preview && (
                <p className="mt-0.5 truncate text-xs text-muted-foreground">
                  {preview}
                </p>
              )}
            </div>
            <span className="shrink-0 text-xs text-muted-foreground">
              {timeAgo}
            </span>
          </div>
        )
      })}
    </div>
  )
}

export default memo(UnreadMessages)

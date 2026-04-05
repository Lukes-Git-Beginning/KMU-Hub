/**
 * Chat channel sidebar listing channels and DM conversations.
 *
 * Sections: "Channels" with create button, "Direct Messages" with new DM button.
 * Active channel is highlighted. Unread badges shown for channels with new messages.
 */
import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Hash, Plus, Search } from 'lucide-react'
import { cn } from '@/lib/cn'
import { useChannels, useDMs, useUnreadCounts, type ChannelInfo } from '@/api/hooks/useChannels'
import { usePresenceStore } from '@/stores/presence'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Separator } from '@/components/ui/separator'
import { CreateChannelDialog } from './CreateChannelDialog'

const PRESENCE_DOT_COLORS: Record<string, string> = {
  online: 'bg-success',
  away: 'bg-warning',
  dnd: 'bg-destructive',
  offline: 'bg-gray-400',
}

interface ChannelListProps {
  selectedChannelId: string | null
  onSelectChannel: (id: string) => void
}

export function ChannelList({ selectedChannelId, onSelectChannel }: ChannelListProps) {
  const { t } = useTranslation()
  const [filter, setFilter] = useState('')
  const [showCreateDialog, setShowCreateDialog] = useState(false)

  const { data: channelsData } = useChannels()
  const { data: dmsData } = useDMs()
  const { data: unreadData } = useUnreadCounts()

  const channels = useMemo(() => {
    const list = channelsData?.channels ?? []
    if (!filter) return list.filter((c) => !c.is_dm)
    const lowerFilter = filter.toLowerCase()
    return list.filter(
      (c) => !c.is_dm && c.name?.toLowerCase().includes(lowerFilter)
    )
  }, [channelsData, filter])

  const dms = useMemo(() => {
    const list = dmsData?.channels ?? []
    if (!filter) return list
    const lowerFilter = filter.toLowerCase()
    return list.filter((c) => c.name?.toLowerCase().includes(lowerFilter))
  }, [dmsData, filter])

  const unreadCounts = unreadData?.unread_counts ?? {}

  return (
    <div className="flex h-full flex-col border-r border-border bg-card">
      {/* Search input */}
      <div className="p-3">
        <div className="relative">
          <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder={t('chat.channels.searchPlaceholder')}
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            className="pl-9 h-9"
          />
        </div>
      </div>

      <ScrollArea className="flex-1">
        {/* Channels section */}
        <div className="px-3 pb-2">
          <div className="flex items-center justify-between py-2">
            <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
              {t('chat.channels.title')}
            </span>
            <Button
              variant="ghost"
              size="icon"
              className="h-6 w-6"
              onClick={() => setShowCreateDialog(true)}
            >
              <Plus className="h-4 w-4" />
            </Button>
          </div>

          <div className="space-y-0.5">
            {channels.map((channel) => (
              <ChannelItem
                key={channel.id}
                channel={channel}
                isActive={selectedChannelId === channel.id}
                unreadCount={channel.id ? unreadCounts[channel.id] ?? 0 : 0}
                onClick={() => channel.id && onSelectChannel(channel.id)}
              />
            ))}
            {channels.length === 0 && (
              <p className="py-2 text-center text-xs text-muted-foreground">
                {t('chat.channels.noResults')}
              </p>
            )}
          </div>
        </div>

        <Separator className="mx-3" />

        {/* DMs section */}
        <div className="px-3 pt-2 pb-4">
          <div className="flex items-center justify-between py-2">
            <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
              {t('chat.dms.title')}
            </span>
          </div>

          <div className="space-y-0.5">
            {dms.map((dm) => (
              <DMItem
                key={dm.id}
                channel={dm}
                isActive={selectedChannelId === dm.id}
                unreadCount={dm.id ? unreadCounts[dm.id] ?? 0 : 0}
                onClick={() => dm.id && onSelectChannel(dm.id)}
              />
            ))}
            {dms.length === 0 && (
              <p className="py-2 text-center text-xs text-muted-foreground">
                {t('chat.dms.noResults')}
              </p>
            )}
          </div>
        </div>
      </ScrollArea>

      <CreateChannelDialog
        open={showCreateDialog}
        onOpenChange={setShowCreateDialog}
        onCreated={(channelId) => {
          setShowCreateDialog(false)
          onSelectChannel(channelId)
        }}
      />
    </div>
  )
}

function ChannelItem({
  channel,
  isActive,
  unreadCount,
  onClick,
}: {
  channel: ChannelInfo
  isActive: boolean
  unreadCount: number
  onClick: () => void
}) {
  return (
    <button
      className={cn(
        'flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors',
        isActive
          ? 'bg-secondary text-secondary-foreground'
          : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
      )}
      onClick={onClick}
    >
      <Hash className="h-4 w-4 shrink-0" />
      <span className="flex-1 truncate text-left">{channel.name}</span>
      {unreadCount > 0 && (
        <Badge variant="destructive" className="h-5 min-w-5 px-1.5 text-xs">
          {unreadCount > 99 ? '99+' : unreadCount}
        </Badge>
      )}
    </button>
  )
}

function DMItem({
  channel,
  isActive,
  unreadCount,
  onClick,
}: {
  channel: ChannelInfo
  isActive: boolean
  unreadCount: number
  onClick: () => void
}) {
  const { t } = useTranslation()
  const presenceMap = usePresenceStore((s) => s.presenceMap)
  const name = channel.name || t('chat.unknown')
  const initials = name
    .split(' ')
    .slice(0, 2)
    .map((w) => w.charAt(0))
    .join('')
    .toUpperCase()

  // Use channel member ID for presence lookup (fallback to 'offline')
  const memberId = (channel as Record<string, unknown>).other_user_id as string | undefined
  const presence = memberId ? presenceMap[memberId] ?? 'offline' : 'offline'

  return (
    <button
      className={cn(
        'flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors',
        isActive
          ? 'bg-secondary text-secondary-foreground'
          : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
      )}
      onClick={onClick}
    >
      <div className="relative shrink-0">
        <Avatar className="h-6 w-6">
          <AvatarFallback className="text-[10px]">{initials}</AvatarFallback>
        </Avatar>
        <span
          className={`absolute -bottom-0.5 -right-0.5 h-2 w-2 rounded-full border-[1.5px] border-card ${PRESENCE_DOT_COLORS[presence] ?? 'bg-gray-400'}`}
        />
      </div>
      <span className="flex-1 truncate text-left">{name}</span>
      {unreadCount > 0 && (
        <Badge variant="destructive" className="h-5 min-w-5 px-1.5 text-xs">
          {unreadCount > 99 ? '99+' : unreadCount}
        </Badge>
      )}
    </button>
  )
}

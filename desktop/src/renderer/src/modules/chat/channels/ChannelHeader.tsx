/**
 * Header bar for the selected chat channel.
 *
 * Shows channel name, member count, description, typing indicators,
 * and a call-from-chat button (VID-06) for starting video/voice calls.
 */
import { Hash, Users, Search, PhoneCall } from 'lucide-react'
import { useChannel, useChannelMembers, type ChannelInfo } from '@/api/hooks/useChannels'
import { useTypingIndicator } from '@/api/hooks/useMessages'
import { useCreateCall } from '@/api/hooks/useVideo'
import { useVideoStore } from '@/stores/video'
import { useAuthStore } from '@/stores/auth'
import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

interface ChannelHeaderProps {
  channelId: string
}

export function ChannelHeader({ channelId }: ChannelHeaderProps) {
  const { data: channelData } = useChannel(channelId)
  const { data: membersData } = useChannelMembers(channelId)
  const { typingUsers } = useTypingIndicator(channelId)
  const createCall = useCreateCall()
  const setActiveCall = useVideoStore((s) => s.setActiveCall)
  const currentUserId = useAuthStore((s) => s.user?.id)

  const channel = channelData?.channel
  const memberCount = membersData?.total ?? channel?.member_count ?? 0
  const isDM = channel?.is_dm ?? false
  const members = membersData?.members ?? []

  const typingText = getTypingText(typingUsers)

  const handleStartCall = () => {
    // Collect target user IDs (all members except the current user)
    const targetUserIds = members
      .map((m) => m.user_id)
      .filter((id): id is string => !!id && id !== currentUserId)

    if (targetUserIds.length === 0) return

    createCall.mutate(
      {
        call_type: isDM ? 'one_to_one' : 'group',
        target_user_ids: targetUserIds,
        channel_id: channelId,
      },
      {
        onSuccess: (callSession) => {
          // Call created -- the join flow is handled by the video module
          // For now, store the call session so the floating bar appears
          if (callSession) {
            setActiveCall(callSession, '', '')
          }
        },
      },
    )
  }

  return (
    <div className="flex flex-col border-b border-border bg-card px-4 py-2">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 min-w-0">
          {!isDM && <Hash className="h-4 w-4 shrink-0 text-muted-foreground" />}
          <h2 className="text-sm font-semibold text-foreground truncate">
            {channel?.name ?? 'Channel'}
          </h2>
          {channel?.description && (
            <span className="hidden md:inline text-xs text-muted-foreground truncate">
              -- {channel.description}
            </span>
          )}
        </div>

        <div className="flex items-center gap-1 shrink-0">
          {/* Call from chat (VID-06) */}
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                className="h-8 w-8"
                onClick={handleStartCall}
                disabled={createCall.isPending}
              >
                <PhoneCall className="h-4 w-4" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>
              {isDM ? 'Anrufen' : 'Gruppenanruf starten'}
            </TooltipContent>
          </Tooltip>

          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="ghost" size="icon" className="h-8 w-8">
                <Users className="h-4 w-4" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>{memberCount} Mitglieder</TooltipContent>
          </Tooltip>

          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="ghost" size="icon" className="h-8 w-8">
                <Search className="h-4 w-4" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>Nachrichten suchen</TooltipContent>
          </Tooltip>
        </div>
      </div>

      {/* Typing indicator */}
      {typingText && (
        <div className="h-4 mt-0.5">
          <p className="text-xs text-muted-foreground animate-pulse truncate">
            {typingText}
          </p>
        </div>
      )}
    </div>
  )
}

function getTypingText(typingUsers: string[]): string | null {
  if (typingUsers.length === 0) return null
  if (typingUsers.length === 1) return `${typingUsers[0]} tippt...`
  if (typingUsers.length === 2) return `${typingUsers[0]} und ${typingUsers[1]} tippen...`
  return 'Mehrere Personen tippen...'
}

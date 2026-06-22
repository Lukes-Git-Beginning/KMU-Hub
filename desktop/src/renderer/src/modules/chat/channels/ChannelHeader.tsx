/**
 * Header bar for the selected chat channel.
 *
 * Shows channel name, member count, description, and typing indicators.
 * Actions: members list button, search messages button, channel overflow
 * menu (leave) for members, join button for non-members of public channels.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Hash, Users, Search, MoreVertical, LogOut, UserPlus, Pencil } from 'lucide-react'
import { useChannel, useChannelMembers, useJoinChannel, useLeaveChannel } from '@/api/hooks/useChannels'
import { EditChannelDialog } from './EditChannelDialog'
import { useTypingIndicator } from '@/api/hooks/useMessages'
import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
} from '@/components/ui/dropdown-menu'
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogFooter,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogAction,
  AlertDialogCancel,
} from '@/components/ui/alert-dialog'

interface ChannelHeaderProps {
  channelId: string
  membersActive?: boolean
  onToggleMembers?: () => void
  searchActive?: boolean
  onToggleSearch?: () => void
  onLeftChannel?: (channelId: string) => void
}

export function ChannelHeader({ channelId, membersActive, onToggleMembers, searchActive, onToggleSearch, onLeftChannel }: ChannelHeaderProps) {
  const { t } = useTranslation()
  const { data: channelData } = useChannel(channelId)
  const { data: membersData } = useChannelMembers(channelId)
  const { typingUsers } = useTypingIndicator(channelId)
  const joinChannel = useJoinChannel()
  const leaveChannel = useLeaveChannel()
  const [confirmLeave, setConfirmLeave] = useState(false)
  const [showEdit, setShowEdit] = useState(false)

  const channel = channelData?.channel
  const memberCount = membersData?.total ?? channel?.member_count ?? 0
  const isDM = channel?.is_dm ?? false
  const isMember = !!channel?.my_role
  const isOwner = channel?.my_role === 'owner'
  const canEdit = !isDM && (isOwner || channel?.my_role === 'admin')
  const canJoin = !isMember && !isDM && !channel?.is_private

  const typingText = getTypingText(typingUsers, t)

  const handleLeave = async () => {
    try {
      await leaveChannel.mutateAsync(channelId)
      setConfirmLeave(false)
      onLeftChannel?.(channelId)
    } catch {
      setConfirmLeave(false)
    }
  }

  return (
    <div className="flex flex-col border-b border-border bg-card px-4 py-2">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 min-w-0">
          {!isDM && <Hash className="h-4 w-4 shrink-0 text-muted-foreground" />}
          <h2 className="text-sm font-semibold text-foreground truncate">
            {channel?.name ?? t('chat.channels.title')}
          </h2>
          {channel?.description && (
            <span className="hidden md:inline text-xs text-muted-foreground truncate">
              -- {channel.description}
            </span>
          )}
        </div>

        <div className="flex items-center gap-1 shrink-0">
          {canJoin && (
            <Button
              variant="default"
              size="sm"
              className="h-8 gap-1.5"
              onClick={() => joinChannel.mutate(channelId)}
              disabled={joinChannel.isPending}
            >
              <UserPlus className="h-4 w-4" />
              {t('chat.header.join')}
            </Button>
          )}

          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                className={`h-8 w-8 ${membersActive ? 'bg-secondary text-foreground' : ''}`}
                onClick={onToggleMembers}
                aria-pressed={membersActive}
              >
                <Users className="h-4 w-4" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>{t('chat.header.members', { count: memberCount })}</TooltipContent>
          </Tooltip>

          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                className={`h-8 w-8 ${searchActive ? 'bg-secondary text-foreground' : ''}`}
                onClick={onToggleSearch}
                aria-pressed={searchActive}
              >
                <Search className="h-4 w-4" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>{t('chat.header.searchMessages')}</TooltipContent>
          </Tooltip>

          {isMember && !isDM && (
            <DropdownMenu>
              <Tooltip>
                <TooltipTrigger asChild>
                  <DropdownMenuTrigger asChild>
                    <Button variant="ghost" size="icon" className="h-8 w-8" aria-label={t('chat.header.menu')}>
                      <MoreVertical className="h-4 w-4" />
                    </Button>
                  </DropdownMenuTrigger>
                </TooltipTrigger>
                <TooltipContent>{t('chat.header.menu')}</TooltipContent>
              </Tooltip>
              <DropdownMenuContent align="end">
                {canEdit && (
                  <DropdownMenuItem onSelect={() => setShowEdit(true)}>
                    <Pencil className="mr-2 h-4 w-4" />
                    {t('chat.editChannel.menuItem')}
                  </DropdownMenuItem>
                )}
                <DropdownMenuItem
                  className="text-destructive focus:text-destructive"
                  disabled={isOwner}
                  onSelect={() => !isOwner && setConfirmLeave(true)}
                >
                  <LogOut className="mr-2 h-4 w-4" />
                  {isOwner ? t('chat.header.leaveOwnerBlocked') : t('chat.header.leave')}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          )}
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

      <AlertDialog open={confirmLeave} onOpenChange={setConfirmLeave}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('chat.leaveDialog.title')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('chat.leaveDialog.description', { channel: channel?.name ?? '' })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('common.cancel')}</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              onClick={(e) => {
                e.preventDefault()
                handleLeave()
              }}
              disabled={leaveChannel.isPending}
            >
              {t('chat.leaveDialog.confirm')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <EditChannelDialog open={showEdit} onOpenChange={setShowEdit} channel={channel} />
    </div>
  )
}

function getTypingText(typingUsers: string[], t: ReturnType<typeof useTranslation>['t']): string | null {
  if (typingUsers.length === 0) return null
  if (typingUsers.length === 1) return t('chat.header.typingOne', { user: typingUsers[0] })
  if (typingUsers.length === 2) return t('chat.header.typingTwo', { user1: typingUsers[0], user2: typingUsers[1] })
  return t('chat.header.typingMany')
}

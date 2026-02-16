/**
 * Channel member list panel with presence indicators.
 *
 * Shows all members of the current chat channel or DM with their
 * presence status. Per user decision, presence dots appear only in
 * chat participant lists and DMs (not in CRM or calendar).
 */
import { useChannelMembers } from '@/api/hooks/useChannels'
import { PresenceIndicator } from '@/features/presence'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { ScrollArea } from '@/components/ui/scroll-area'

interface ChannelMemberListProps {
  channelId: string
}

export function ChannelMemberList({ channelId }: ChannelMemberListProps) {
  const { data: membersData, isLoading } = useChannelMembers(channelId)

  const members = membersData?.members ?? []

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center p-4">
        <span className="text-sm text-muted-foreground">Laden...</span>
      </div>
    )
  }

  return (
    <div className="flex h-full flex-col border-l border-border bg-card">
      <div className="border-b border-border px-4 py-3">
        <h3 className="text-sm font-semibold text-foreground">
          Mitglieder ({members.length})
        </h3>
      </div>

      <ScrollArea className="flex-1">
        <div className="space-y-0.5 p-2">
          {members.map((member) => {
            const name = [member.first_name, member.last_name]
              .filter(Boolean)
              .join(' ') || 'Unbekannt'

            const initials = [member.first_name, member.last_name]
              .filter(Boolean)
              .map((n) => n!.charAt(0))
              .join('')
              .toUpperCase() || '??'

            return (
              <div
                key={member.user_id}
                className="flex items-center gap-3 rounded-md px-3 py-2 text-sm text-foreground hover:bg-accent/50"
              >
                <div className="relative shrink-0">
                  <Avatar className="h-8 w-8">
                    <AvatarFallback className="text-xs">{initials}</AvatarFallback>
                  </Avatar>
                  {member.user_id && (
                    <PresenceIndicator
                      userId={member.user_id}
                      size="sm"
                      className="absolute -bottom-0.5 -right-0.5"
                    />
                  )}
                </div>
                <span className="truncate">{name}</span>
              </div>
            )
          })}
        </div>
      </ScrollArea>
    </div>
  )
}

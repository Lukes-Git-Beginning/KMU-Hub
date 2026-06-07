/**
 * Chat module layout with three-panel design.
 *
 * Left: channel/DM list (280px). Center: message area (flex-1).
 * Right: thread panel (350px, conditional). Empty state shown when
 * no channel is selected.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { MessageSquare } from 'lucide-react'
import { ChannelList } from './channels/ChannelList'
import { ChannelHeader } from './channels/ChannelHeader'
import { MessageList } from './messages/MessageList'
import { MessageInput } from './messages/MessageInput'
import { ThreadPanel } from './threads/ThreadPanel'
import { ChannelMemberList } from '@/components/chat/ChannelMemberList'

export default function ChatLayout() {
  const { t } = useTranslation()
  const [selectedChannelId, setSelectedChannelId] = useState<string | null>(null)
  const [selectedThreadMessageId, setSelectedThreadMessageId] = useState<string | null>(null)
  const [showMembers, setShowMembers] = useState(false)

  const handleSelectChannel = (id: string) => {
    setSelectedChannelId(id)
    setSelectedThreadMessageId(null)
  }

  // Members panel and thread panel share the right slot — keep them exclusive.
  const handleOpenThread = (messageId: string) => {
    setSelectedThreadMessageId(messageId)
    setShowMembers(false)
  }

  const handleCloseThread = () => {
    setSelectedThreadMessageId(null)
  }

  const handleToggleMembers = () => {
    setShowMembers((v) => !v)
    setSelectedThreadMessageId(null)
  }

  return (
    <div className="flex h-full">
      {/* Channel sidebar */}
      <aside className="h-full w-[280px] shrink-0">
        <ChannelList
          selectedChannelId={selectedChannelId}
          onSelectChannel={handleSelectChannel}
        />
      </aside>

      {/* Message area */}
      <div className="flex flex-1 flex-col min-w-0">
        {selectedChannelId ? (
          <>
            <ChannelHeader
              channelId={selectedChannelId}
              membersActive={showMembers}
              onToggleMembers={handleToggleMembers}
            />
            <MessageList
              channelId={selectedChannelId}
              onOpenThread={handleOpenThread}
            />
            <MessageInput channelId={selectedChannelId} />
          </>
        ) : (
          <div className="flex h-full items-center justify-center">
            <div className="text-center">
              <MessageSquare className="mx-auto h-12 w-12 text-muted-foreground/50" />
              <h2 className="mt-4 text-lg font-semibold text-foreground">
                {t('chat.empty.title')}
              </h2>
              <p className="mt-1 text-sm text-muted-foreground">
                {t('chat.empty.description')}
              </p>
            </div>
          </div>
        )}
      </div>

      {/* Thread panel */}
      {selectedThreadMessageId && selectedChannelId && (
        <div className="w-[350px] shrink-0">
          <ThreadPanel
            messageId={selectedThreadMessageId}
            channelId={selectedChannelId}
            onClose={handleCloseThread}
          />
        </div>
      )}

      {/* Members panel (exclusive with thread panel) */}
      {showMembers && selectedChannelId && !selectedThreadMessageId && (
        <div className="w-[280px] shrink-0">
          <ChannelMemberList channelId={selectedChannelId} />
        </div>
      )}
    </div>
  )
}

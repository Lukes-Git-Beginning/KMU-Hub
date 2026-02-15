/**
 * Chat module layout with three-panel design.
 *
 * Left: channel/DM list (280px). Center: message area (flex-1).
 * Right: thread panel (350px, conditional). Empty state shown when
 * no channel is selected.
 */
import { useState } from 'react'
import { MessageSquare } from 'lucide-react'
import { ChannelList } from './channels/ChannelList'
import { ChannelHeader } from './channels/ChannelHeader'
import { MessageList } from './messages/MessageList'
import { MessageInput } from './messages/MessageInput'
import { ThreadPanel } from './threads/ThreadPanel'

export default function ChatLayout() {
  const [selectedChannelId, setSelectedChannelId] = useState<string | null>(null)
  const [selectedThreadMessageId, setSelectedThreadMessageId] = useState<string | null>(null)

  const handleSelectChannel = (id: string) => {
    setSelectedChannelId(id)
    setSelectedThreadMessageId(null)
  }

  const handleOpenThread = (messageId: string) => {
    setSelectedThreadMessageId(messageId)
  }

  const handleCloseThread = () => {
    setSelectedThreadMessageId(null)
  }

  return (
    <div className="flex h-full">
      {/* Channel sidebar */}
      <div className="w-[280px] shrink-0">
        <ChannelList
          selectedChannelId={selectedChannelId}
          onSelectChannel={handleSelectChannel}
        />
      </div>

      {/* Message area */}
      <div className="flex flex-1 flex-col min-w-0">
        {selectedChannelId ? (
          <>
            <ChannelHeader channelId={selectedChannelId} />
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
                Wähle einen Channel
              </h2>
              <p className="mt-1 text-sm text-muted-foreground">
                Wähle einen Channel oder eine Direktnachricht aus der Liste, um zu chatten.
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
    </div>
  )
}

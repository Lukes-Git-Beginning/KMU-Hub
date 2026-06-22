/**
 * Chat module layout with three-panel design.
 *
 * Left: channel/DM list (280px). Center: message area (flex-1).
 * Right: thread panel (350px, conditional). Empty state shown when
 * no channel is selected.
 */
import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { MessageSquare } from 'lucide-react'
import { ChannelList } from './channels/ChannelList'
import { ChannelHeader } from './channels/ChannelHeader'
import { MessageList } from './messages/MessageList'
import { MessageInput } from './messages/MessageInput'
import { ThreadPanel } from './threads/ThreadPanel'
import { ChannelSearchPanel } from './channels/ChannelSearchPanel'
import { MentionsPanel } from './MentionsPanel'
import { BookmarksPanel } from './BookmarksPanel'
import { ChannelMemberList } from '@/components/chat/ChannelMemberList'
import { useNavigationStore } from '@/stores/navigation'

export default function ChatLayout() {
  const { t } = useTranslation()
  const [selectedChannelId, setSelectedChannelId] = useState<string | null>(null)
  const intent = useNavigationStore((s) => s.intent)
  const consumeIntent = useNavigationStore((s) => s.consumeIntent)
  const [selectedThreadMessageId, setSelectedThreadMessageId] = useState<string | null>(null)
  const [showMembers, setShowMembers] = useState(false)
  const [showSearch, setShowSearch] = useState(false)
  const [showMentions, setShowMentions] = useState(false)
  const [showBookmarks, setShowBookmarks] = useState(false)
  const [highlightMessageId, setHighlightMessageId] = useState<string | null>(null)

  const handleSelectChannel = (id: string) => {
    setSelectedChannelId(id)
    setSelectedThreadMessageId(null)
    setShowMentions(false)
    setShowBookmarks(false)
    setHighlightMessageId(null)
  }

  // Jump from a search result: switch channel if needed and flash the message.
  const handleJumpToMessage = (channelId: string, messageId: string) => {
    setSelectedChannelId(channelId)
    setSelectedThreadMessageId(null)
    setHighlightMessageId(messageId)
  }

  const handleLeftChannel = () => {
    setSelectedChannelId(null)
    setSelectedThreadMessageId(null)
    setShowMembers(false)
    setShowSearch(false)
  }

  // Consume send-message intent: if channelId is provided (from UserProfileCard ping-to-chat),
  // select that DM channel directly. Reacts to the store value (not mount-only) because the
  // card can be opened from inside the chat (MessageBubble, ChannelMemberList) while ChatLayout
  // stays mounted. Only send-message intents with channelId are consumed — other intent types
  // are left for their own consumers. Name-matching fallback is intentionally NOT implemented:
  // existing callers (TeamPage, MemberProfileContent, KontaktePage) do not pass channelId.
  useEffect(() => {
    if (intent?.type === 'send-message' && intent.data.channelId) {
      consumeIntent()
      handleSelectChannel(intent.data.channelId)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [intent])

  // Thread, members, search, mentions and bookmarks share the right slot — keep them mutually exclusive.
  const handleOpenThread = (messageId: string) => {
    setSelectedThreadMessageId(messageId)
    setShowMembers(false)
    setShowSearch(false)
    setShowMentions(false)
    setShowBookmarks(false)
  }

  const handleCloseThread = () => {
    setSelectedThreadMessageId(null)
  }

  const handleToggleMembers = () => {
    setShowMembers((v) => !v)
    setSelectedThreadMessageId(null)
    setShowSearch(false)
    setShowMentions(false)
    setShowBookmarks(false)
  }

  const handleToggleSearch = () => {
    setShowSearch((v) => !v)
    setSelectedThreadMessageId(null)
    setShowMembers(false)
    setShowMentions(false)
    setShowBookmarks(false)
  }

  const handleToggleMentions = () => {
    setShowMentions((v) => !v)
    setSelectedThreadMessageId(null)
    setShowMembers(false)
    setShowSearch(false)
    setShowBookmarks(false)
  }

  const handleToggleBookmarks = () => {
    setShowBookmarks((v) => !v)
    setSelectedThreadMessageId(null)
    setShowMembers(false)
    setShowSearch(false)
    setShowMentions(false)
  }

  return (
    <div className="flex h-full">
      {/* Channel sidebar */}
      <aside className="h-full w-[280px] shrink-0">
        <ChannelList
          selectedChannelId={selectedChannelId}
          onSelectChannel={handleSelectChannel}
          mentionsActive={showMentions}
          onToggleMentions={handleToggleMentions}
          bookmarksActive={showBookmarks}
          onToggleBookmarks={handleToggleBookmarks}
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
              searchActive={showSearch}
              onToggleSearch={handleToggleSearch}
              onLeftChannel={handleLeftChannel}
            />
            <MessageList
              channelId={selectedChannelId}
              onOpenThread={handleOpenThread}
              highlightMessageId={highlightMessageId}
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

      {/* Members panel (exclusive with thread/search) */}
      {showMembers && selectedChannelId && !selectedThreadMessageId && (
        <div className="w-[280px] shrink-0">
          <ChannelMemberList channelId={selectedChannelId} />
        </div>
      )}

      {/* Search panel (exclusive with thread/members) */}
      {showSearch && selectedChannelId && !selectedThreadMessageId && !showMembers && (
        <div className="w-[320px] shrink-0">
          <ChannelSearchPanel channelId={selectedChannelId} onClose={() => setShowSearch(false)} onJumpToMessage={handleJumpToMessage} />
        </div>
      )}

      {/* Mentions panel (cross-channel, available without a selected channel) */}
      {showMentions && !selectedThreadMessageId && (
        <div className="w-[320px] shrink-0">
          <MentionsPanel
            onClose={() => setShowMentions(false)}
            onSelectChannel={handleSelectChannel}
          />
        </div>
      )}

      {/* Bookmarks panel (cross-channel saved messages) */}
      {showBookmarks && !selectedThreadMessageId && (
        <div className="w-[320px] shrink-0">
          <BookmarksPanel
            onClose={() => setShowBookmarks(false)}
            onSelectChannel={handleSelectChannel}
          />
        </div>
      )}
    </div>
  )
}

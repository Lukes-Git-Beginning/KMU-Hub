/**
 * Kommunikation — Unified External Inbox.
 *
 * Three-column layout:
 *   Left  (w-80):    ConversationList — channel tabs, filters, conversation items
 *   Center (flex-1):  ConversationThread — message timeline + reply composer
 *   Right  (w-72):    ContextPanel — CRM contact, deals, tickets, activity
 *
 * Data from useInboxMessages (TanStack Query). UI state from useKommunikationStore.
 *
 * Keyboard shortcuts: j/k nav, Escape deselect.
 */
import { useEffect, useCallback, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useKommunikationStore } from '@/stores/kommunikation'
import { useInboxMessages } from '@/api/hooks/useInbox'
import type { InboxListFilter, InboxChannel } from '@/api/inbox-types'
import { ConversationList } from './ConversationList'
import { ConversationThread } from './ConversationThread'
import { ContextPanel } from './ContextPanel'
import { NewConversationDialog } from './NewConversationDialog'
import { ChannelSettingsDialog } from './ChannelSettingsDialog'

// ---------------------------------------------------------------------------
// Main page
// ---------------------------------------------------------------------------

export default function KommunikationPage() {
  const activeChannel = useKommunikationStore((s) => s.activeChannel)
  const searchQuery = useKommunikationStore((s) => s.searchQuery)
  const selectedId = useKommunikationStore((s) => s.selectedConversationId)
  const setSelectedConversation = useKommunikationStore((s) => s.setSelectedConversation)
  const [showNewConversation, setShowNewConversation] = useState(false)
  const [showChannelSettings, setShowChannelSettings] = useState(false)

  // Build filter from UI state
  const filter = useMemo<InboxListFilter>(() => {
    const f: InboxListFilter = {}
    if (activeChannel !== 'all') {
      // Map old CommunicationChannel values to InboxChannel
      const channelMap: Record<string, InboxChannel> = {
        email: 'email',
        whatsapp: 'chat',
        teams: 'chat',
        widget: 'chat',
        portal: 'chat',
      }
      f.channel = channelMap[activeChannel] ?? (activeChannel as InboxChannel)
    }
    if (searchQuery.trim()) {
      f.search = searchQuery.trim()
    }
    return f
  }, [activeChannel, searchQuery])

  const { data, isLoading } = useInboxMessages(filter)
  const messages = useMemo(() => data?.messages ?? [], [data?.messages])

  // Keyboard shortcuts: j/k to navigate, Escape to deselect
  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      const target = e.target as HTMLElement
      if (
        target.tagName === 'INPUT' ||
        target.tagName === 'TEXTAREA' ||
        target.tagName === 'SELECT' ||
        target.isContentEditable
      ) {
        return
      }

      const currentIdx = messages.findIndex((m) => m.id === selectedId)

      switch (e.key) {
        case 'j': {
          const nextIdx = currentIdx < messages.length - 1 ? currentIdx + 1 : currentIdx
          if (messages[nextIdx]) {
            setSelectedConversation(messages[nextIdx].id)
          }
          break
        }
        case 'k': {
          const prevIdx = currentIdx > 0 ? currentIdx - 1 : 0
          if (messages[prevIdx]) {
            setSelectedConversation(messages[prevIdx].id)
          }
          break
        }
        case 'Escape': {
          setSelectedConversation(null)
          break
        }
      }
    },
    [messages, selectedId, setSelectedConversation],
  )

  useEffect(() => {
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [handleKeyDown])

  return (
    <>
      <div className="flex h-full overflow-hidden animate-fade-in">
        {/* Left: Conversation list */}
        <ConversationList
          messages={messages}
          isLoading={isLoading}
          onNewConversation={() => setShowNewConversation(true)}
        />

        {/* Center: Conversation thread */}
        <ConversationThread />

        {/* Right: CRM context panel */}
        <ContextPanel />
      </div>

      {/* Dialogs */}
      <NewConversationDialog
        open={showNewConversation}
        onOpenChange={setShowNewConversation}
      />
      <ChannelSettingsDialog
        open={showChannelSettings}
        onOpenChange={setShowChannelSettings}
      />
    </>
  )
}

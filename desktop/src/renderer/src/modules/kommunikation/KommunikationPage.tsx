/**
 * Kommunikation — Unified External Inbox.
 *
 * Three-column layout:
 *   Left  (w-80):    ConversationList — channel tabs, filters, conversation items
 *   Center (flex-1):  ConversationThread — message timeline + reply composer
 *   Right  (w-72):    ContextPanel — CRM contact, deals, tickets, activity
 *
 * All data from useKommunikationStore (mock). Backend swap: replace store
 * reads with TanStack Query hooks, keep components identical.
 *
 * Keyboard shortcuts: j/k nav, Escape deselect.
 */
import { useEffect, useCallback, useState } from 'react'
import { Settings } from 'lucide-react'
import { useKommunikationStore } from '@/stores/kommunikation'
import { ConversationList } from './ConversationList'
import { ConversationThread } from './ConversationThread'
import { ContextPanel } from './ContextPanel'
import { NewConversationDialog } from './NewConversationDialog'
import { ChannelSettingsDialog } from './ChannelSettingsDialog'

// ---------------------------------------------------------------------------
// Main page
// ---------------------------------------------------------------------------

export default function KommunikationPage() {
  const conversations = useKommunikationStore((s) => s.conversations)
  const selectedId = useKommunikationStore((s) => s.selectedConversationId)
  const setSelectedConversation = useKommunikationStore((s) => s.setSelectedConversation)
  const [showNewConversation, setShowNewConversation] = useState(false)
  const [showChannelSettings, setShowChannelSettings] = useState(false)

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

      const currentIdx = conversations.findIndex((c) => c.id === selectedId)

      switch (e.key) {
        case 'j': {
          const nextIdx = currentIdx < conversations.length - 1 ? currentIdx + 1 : currentIdx
          if (conversations[nextIdx]) {
            setSelectedConversation(conversations[nextIdx].id)
          }
          break
        }
        case 'k': {
          const prevIdx = currentIdx > 0 ? currentIdx - 1 : 0
          if (conversations[prevIdx]) {
            setSelectedConversation(conversations[prevIdx].id)
          }
          break
        }
        case 'Escape': {
          setSelectedConversation(null)
          break
        }
      }
    },
    [conversations, selectedId, setSelectedConversation],
  )

  useEffect(() => {
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [handleKeyDown])

  return (
    <>
      <div className="flex h-full overflow-hidden">
        {/* Left: Conversation list */}
        <ConversationList
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

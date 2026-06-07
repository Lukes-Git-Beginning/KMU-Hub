/**
 * Kommunikation — unified communication module.
 *
 * One module, two areas switched via a segmented control at the top:
 *   • Team        → internal team chat (channels, DMs, threads)  [ChatLayout]
 *   • Posteingang → omnichannel customer inbox (conversations)    [InboxView]
 *
 * The active area is reflected in the `?bereich=team|posteingang` query param
 * (so /chat can redirect into the Team area). Without a param the user's
 * personal default (kommunikationPrefs) decides.
 */
import { useEffect, useCallback, useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { MessageSquare, Inbox } from 'lucide-react'
import ChatLayout from '@/modules/chat/ChatLayout'
import { useKommunikationStore } from '@/stores/kommunikation'
import { useKommunikationPrefs, type KommunikationBereich } from '@/stores/kommunikationPrefs'
import { useInboxMessages, useUnreadCount } from '@/api/hooks/useInbox'
import type { InboxListFilter, InboxChannel } from '@/api/inbox-types'
import { ConversationList } from './ConversationList'
import { ConversationThread } from './ConversationThread'
import { ContextPanel } from './ContextPanel'
import { NewConversationDialog } from './NewConversationDialog'
import { ChannelSettingsDialog } from './ChannelSettingsDialog'

// ---------------------------------------------------------------------------
// Root: area switcher + the two areas
// ---------------------------------------------------------------------------

export default function KommunikationPage() {
  const { t } = useTranslation()
  const [searchParams, setSearchParams] = useSearchParams()
  const defaultBereich = useKommunikationPrefs((s) => s.defaultBereich)
  const { data: inboxUnread } = useUnreadCount()

  const paramBereich = searchParams.get('bereich') as KommunikationBereich | null
  const bereich: KommunikationBereich = paramBereich === 'team' || paramBereich === 'posteingang' ? paramBereich : defaultBereich

  const setBereich = (next: KommunikationBereich) => {
    const params = new URLSearchParams(searchParams)
    params.set('bereich', next)
    setSearchParams(params, { replace: true })
  }

  const posteingangUnread = inboxUnread?.total ?? 0

  return (
    <div className="flex h-full flex-col overflow-hidden animate-fade-in">
      {/* Area switcher */}
      <div className="flex shrink-0 items-center gap-1 border-b border-border px-3 py-2">
        <div className="inline-flex items-center gap-0.5 rounded-lg bg-secondary/60 p-0.5">
          <SwitchTab
            active={bereich === 'team'}
            icon={MessageSquare}
            label={t('kommunikation.bereich.team')}
            onClick={() => setBereich('team')}
          />
          <SwitchTab
            active={bereich === 'posteingang'}
            icon={Inbox}
            label={t('kommunikation.bereich.posteingang')}
            badge={posteingangUnread}
            onClick={() => setBereich('posteingang')}
          />
        </div>
      </div>

      {/* Active area */}
      <div className="min-h-0 flex-1">
        {bereich === 'team' ? <ChatLayout /> : <InboxView />}
      </div>
    </div>
  )
}

function SwitchTab({
  active,
  icon: Icon,
  label,
  badge,
  onClick,
}: {
  active: boolean
  icon: typeof MessageSquare
  label: string
  badge?: number
  onClick: () => void
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={active}
      className={`flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium transition-colors ${
        active ? 'bg-card text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'
      }`}
    >
      <Icon className="h-3.5 w-3.5" aria-hidden="true" />
      {label}
      {badge !== undefined && badge > 0 && (
        <span className="ml-0.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-primary px-1 text-[10px] font-semibold text-primary-foreground">
          {badge > 99 ? '99+' : badge}
        </span>
      )}
    </button>
  )
}

// ---------------------------------------------------------------------------
// Posteingang — omnichannel customer inbox (formerly the whole page)
// ---------------------------------------------------------------------------

function InboxView() {
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
      <div className="flex h-full overflow-hidden">
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

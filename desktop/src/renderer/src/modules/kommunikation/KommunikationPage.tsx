/**
 * Unified Inbox (Kommunikation) main page.
 *
 * Three-column layout:
 * - Left: InboxSidebar (w-64, collapsible)
 * - Center: MessageList (flex-1, min-w-80)
 * - Right: MessageDetail (w-[480px], shown when message selected)
 *
 * Keyboard shortcuts: j/k (next/prev), e (archive), s (star), r (reply).
 */
import { useState, useEffect, useCallback, useMemo } from 'react'
import { useKommunikationStore } from '@/stores/kommunikation'
import { useInboxMessages, useArchiveMessage, useToggleStar } from '@/api/hooks/useInbox'
import type { InboxListFilter, InboxMessage, TeamInbox, InboxChannel } from '@/api/inbox-types'
import { useAuthStore } from '@/stores/auth'
import { InboxSidebar } from './InboxSidebar'
import { MessageList } from './MessageList'
import { MessageDetail } from './MessageDetail'
import { TeamInboxSettings } from './TeamInboxSettings'
import { RoutingRulesEditor } from './RoutingRulesEditor'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { useCreateTeamInbox } from '@/api/hooks/useInbox'
import { Settings } from 'lucide-react'

// ---------------------------------------------------------------------------
// Filter derivation from active view
// ---------------------------------------------------------------------------

function deriveFilter(activeView: string, userId: string): InboxListFilter {
  const base: InboxListFilter = { is_archived: false, page_size: 50 }

  switch (activeView) {
    case 'all':
      return base
    case 'unread':
      return { ...base, is_read: false }
    case 'starred':
      return { ...base, is_starred: true }
    case 'assigned':
      return base // assigned filtering is done client-side via assigned_to === userId
    case 'email':
    case 'chat':
    case 'notification':
      return { ...base, channel: activeView as InboxChannel }
    default:
      // Assume it's a team_inbox_id
      return { ...base, team_inbox_id: activeView }
  }
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function KommunikationPage() {
  const user = useAuthStore((s) => s.user)
  const userId = user?.id ?? ''
  const {
    activeView,
    selectedMessageId,
    setSelectedMessage,
  } = useKommunikationStore()

  // Routing rules panel
  const [showRules, setShowRules] = useState(false)

  // Team inbox create dialog
  const [createTeamOpen, setCreateTeamOpen] = useState(false)
  const [newTeamName, setNewTeamName] = useState('')
  const createTeam = useCreateTeamInbox()

  // Team inbox settings
  const [settingsTeam, setSettingsTeam] = useState<TeamInbox | null>(null)
  const [settingsOpen, setSettingsOpen] = useState(false)

  // Derive filter from active view
  const filter = useMemo(
    () => deriveFilter(activeView, userId),
    [activeView, userId],
  )
  const { data: messagesData, isLoading } = useInboxMessages(filter)

  // Client-side filter for "assigned" view
  const messages = useMemo(() => {
    const list = messagesData?.messages ?? []
    if (activeView === 'assigned') {
      return list.filter((m) => m.assigned_to === userId)
    }
    return list
  }, [messagesData, activeView, userId])

  const selectedMessage = messages.find((m) => m.id === selectedMessageId) ?? null

  // Mutations for keyboard shortcuts
  const archiveMsg = useArchiveMessage()
  const toggleStar = useToggleStar()

  // Reply state (opens detail with focus on reply)
  const [replyTarget, setReplyTarget] = useState<InboxMessage | null>(null)

  // Keyboard shortcuts
  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      // Don't capture when typing in inputs
      const target = e.target as HTMLElement
      if (
        target.tagName === 'INPUT' ||
        target.tagName === 'TEXTAREA' ||
        target.tagName === 'SELECT' ||
        target.isContentEditable
      ) {
        return
      }

      const currentIdx = messages.findIndex(
        (m) => m.id === selectedMessageId,
      )

      switch (e.key) {
        case 'j': {
          // Next message
          const nextIdx = currentIdx < messages.length - 1 ? currentIdx + 1 : currentIdx
          if (messages[nextIdx]) {
            setSelectedMessage(messages[nextIdx].id)
          }
          break
        }
        case 'k': {
          // Previous message
          const prevIdx = currentIdx > 0 ? currentIdx - 1 : 0
          if (messages[prevIdx]) {
            setSelectedMessage(messages[prevIdx].id)
          }
          break
        }
        case 'e': {
          // Archive
          if (selectedMessageId) {
            archiveMsg.mutate(selectedMessageId)
            setSelectedMessage(null)
          }
          break
        }
        case 's': {
          // Star
          if (selectedMessageId) {
            toggleStar.mutate(selectedMessageId)
          }
          break
        }
        case 'r': {
          // Reply (set reply target to trigger focus)
          if (selectedMessage) {
            setReplyTarget(selectedMessage)
          }
          break
        }
        case 'Escape': {
          setSelectedMessage(null)
          break
        }
      }
    },
    [messages, selectedMessageId, selectedMessage, setSelectedMessage, archiveMsg, toggleStar],
  )

  useEffect(() => {
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [handleKeyDown])

  // Handle create team
  const handleCreateTeam = () => {
    if (!newTeamName.trim()) return
    createTeam.mutate(
      { name: newTeamName.trim() },
      {
        onSuccess: () => {
          setNewTeamName('')
          setCreateTeamOpen(false)
        },
      },
    )
  }

  const handleTeamSettings = (team: TeamInbox) => {
    setSettingsTeam(team)
    setSettingsOpen(true)
  }

  return (
    <div className="flex h-full overflow-hidden">
      {/* Left: Sidebar */}
      <InboxSidebar
        onCreateTeam={() => setCreateTeamOpen(true)}
        onTeamSettings={handleTeamSettings}
      />

      {/* Center: Message list */}
      <div className="flex flex-col flex-1 min-w-[320px] overflow-hidden">
        {/* Toolbar */}
        <div className="flex items-center justify-between border-b border-border px-4 py-2">
          <h2 className="text-sm font-medium text-foreground">
            Posteingang
          </h2>
          <button
            onClick={() => setShowRules(!showRules)}
            className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
            title="Routing-Regeln"
          >
            <Settings className="h-4 w-4" />
          </button>
        </div>

        {/* Routing rules panel (toggle) */}
        {showRules ? (
          <div className="flex-1 overflow-y-auto p-4">
            <RoutingRulesEditor />
          </div>
        ) : (
          <MessageList
            messages={messages}
            isLoading={isLoading}
            selectedId={selectedMessageId}
            onSelect={(id) => setSelectedMessage(id)}
            onReply={(msg) => {
              setSelectedMessage(msg.id)
              setReplyTarget(msg)
            }}
            onAssign={(msg) => {
              // Open detail and scroll to assign section
              setSelectedMessage(msg.id)
            }}
          />
        )}
      </div>

      {/* Right: Message detail */}
      {selectedMessage && (
        <MessageDetail
          message={selectedMessage}
          onClose={() => setSelectedMessage(null)}
        />
      )}

      {/* Create team inbox dialog */}
      <Dialog open={createTeamOpen} onOpenChange={setCreateTeamOpen}>
        <DialogContent className="max-w-sm">
          <DialogHeader>
            <DialogTitle>Neues Team-Postfach</DialogTitle>
            <DialogDescription>
              Erstelle ein gemeinsames Postfach fuer dein Team.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-3 pt-2">
            <div>
              <label className="text-xs font-medium text-foreground">
                Name
              </label>
              <input
                type="text"
                value={newTeamName}
                onChange={(e) => setNewTeamName(e.target.value)}
                placeholder="z.B. Support, Vertrieb, Buchhaltung"
                className="mt-1 w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
                onKeyDown={(e) => {
                  if (e.key === 'Enter') handleCreateTeam()
                }}
              />
            </div>
            <div className="flex justify-end gap-2">
              <button
                onClick={() => setCreateTeamOpen(false)}
                className="rounded-md border border-border px-3 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors"
              >
                Abbrechen
              </button>
              <button
                onClick={handleCreateTeam}
                disabled={!newTeamName.trim() || createTeam.isPending}
                className="rounded-md bg-primary px-3 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50"
              >
                Erstellen
              </button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      {/* Team inbox settings dialog */}
      <TeamInboxSettings
        team={settingsTeam}
        open={settingsOpen}
        onOpenChange={setSettingsOpen}
      />
    </div>
  )
}

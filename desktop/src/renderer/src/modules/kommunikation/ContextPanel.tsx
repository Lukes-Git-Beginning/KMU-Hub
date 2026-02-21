import { useMemo } from 'react'
import { X, PanelRightClose } from 'lucide-react'
import { useKommunikationStore } from '@/stores/kommunikation'
import { ContactCard } from './ContactCard'
import { OpenDeals } from './OpenDeals'
import { OpenTickets } from './OpenTickets'
import { ActivityTimeline } from './ActivityTimeline'

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function ContextPanel() {
  const selectedId = useKommunikationStore((s) => s.selectedConversationId)
  const conversations = useKommunikationStore((s) => s.conversations)
  const detailPaneOpen = useKommunikationStore((s) => s.detailPaneOpen)
  const toggleDetailPane = useKommunikationStore((s) => s.toggleDetailPane)

  const conv = useMemo(
    () => conversations.find((c) => c.id === selectedId) ?? null,
    [conversations, selectedId],
  )

  // No conversation selected → no panel
  if (!conv) return null

  // Panel collapsed → show toggle button
  if (!detailPaneOpen) {
    return (
      <div className="flex shrink-0 flex-col items-center border-l border-border bg-card/30 px-1.5 py-2">
        <button
          onClick={toggleDetailPane}
          className="rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
          title="Kontext-Panel oeffnen"
        >
          <PanelRightClose className="h-4 w-4" />
        </button>
      </div>
    )
  }

  return (
    <div className="flex w-72 shrink-0 flex-col border-l border-border bg-card/30 overflow-y-auto">
      {/* Header */}
      <div className="flex items-center justify-between px-3 py-2.5 border-b border-border">
        <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
          Kontext
        </h3>
        <button
          onClick={toggleDetailPane}
          className="rounded-md p-1 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
          title="Panel schliessen"
        >
          <X className="h-3.5 w-3.5" />
        </button>
      </div>

      {/* Contact card */}
      <ContactCard conversation={conv} />

      {/* Open deals */}
      <OpenDeals dealIds={conv.crmDealIds ?? []} />

      {/* Open tickets */}
      <OpenTickets ticketIds={conv.crmTicketIds ?? []} />

      {/* Activity timeline */}
      <ActivityTimeline contactName={conv.contactName} />

      {/* Participants */}
      <div className="p-3">
        <h4 className="text-[10px] font-semibold text-muted-foreground uppercase tracking-wider mb-2">
          Teilnehmer ({conv.participants.length})
        </h4>
        <div className="space-y-1.5">
          {conv.participants.map((p) => (
            <div key={p.id} className="flex items-center gap-2 text-xs">
              <div className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-secondary text-[9px] font-medium text-muted-foreground">
                {p.name
                  .split(' ')
                  .map((n) => n[0])
                  .join('')
                  .toUpperCase()
                  .slice(0, 2)}
              </div>
              <span className="text-foreground/90 truncate">{p.name}</span>
              {p.isInternal && (
                <span className="shrink-0 rounded bg-primary/10 px-1 py-0.5 text-[9px] text-primary">
                  Intern
                </span>
              )}
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

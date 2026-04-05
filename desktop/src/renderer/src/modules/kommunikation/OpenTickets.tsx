import { useTranslation } from 'react-i18next'
import { Headphones, ArrowRight } from 'lucide-react'

// ---------------------------------------------------------------------------
// Mock ticket data (linked from conversation.crmTicketIds)
// ---------------------------------------------------------------------------

interface MockTicket {
  id: string
  subject: string
  status: string
  priority: string
  createdAt: string
}

const mockTickets: Record<string, MockTicket> = {
  t15: { id: 't15', subject: 'Kalender-Sync Outlook', status: 'Offen', priority: 'Normal', createdAt: '2026-02-18' },
}

const statusColors: Record<string, string> = {
  Offen: 'text-success',
  'In Bearbeitung': 'text-warning',
  Wartend: 'text-orange-500',
  Geloest: 'text-blue-500',
  Geschlossen: 'text-muted-foreground',
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface OpenTicketsProps {
  ticketIds: string[]
}

export function OpenTickets({ ticketIds }: OpenTicketsProps) {
  const { t } = useTranslation()
  if (ticketIds.length === 0) return null

  const tickets = ticketIds
    .map((id) => mockTickets[id])
    .filter(Boolean)

  if (tickets.length === 0) return null

  return (
    <div className="p-3 border-b border-border">
      <div className="flex items-center gap-1.5 mb-2">
        <Headphones className="h-3 w-3 text-muted-foreground" />
        <h4 className="text-[10px] font-semibold text-muted-foreground uppercase tracking-wider">
          {t('kommunikation.context.helpdeskTickets')}
        </h4>
        <span className="ml-auto text-[10px] text-muted-foreground">{tickets.length}</span>
      </div>
      <div className="space-y-1.5">
        {tickets.map((ticket) => (
          <button
            key={ticket.id}
            className="flex w-full items-center gap-2 rounded-md bg-secondary/50 px-2.5 py-2 text-left hover:bg-secondary transition-colors group"
          >
            <div className="min-w-0 flex-1">
              <p className="text-xs font-medium text-foreground truncate">{ticket.subject}</p>
              <div className="flex items-center gap-1.5 text-[10px] text-muted-foreground">
                <span className={statusColors[ticket.status] ?? ''}>
                  {ticket.status}
                </span>
                <span>·</span>
                <span>{ticket.priority}</span>
                <span>·</span>
                <span>#{ticket.id}</span>
              </div>
            </div>
            <ArrowRight className="h-3 w-3 text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity shrink-0" />
          </button>
        ))}
      </div>
    </div>
  )
}

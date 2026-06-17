/**
 * Open Tickets widget — open/in-progress tickets with SLA status.
 *
 * Module: helpdesk
 * Data: useHelpdeskStore (Zustand, localStorage-persisted mock data).
 */
import { memo, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { TicketCheck, AlertCircle } from 'lucide-react'
import { useHelpdeskStore, slaLabel } from '@/stores/helpdesk'
import type { WidgetProps } from '@/components/widgets/WidgetRegistry'

const PRIORITY_BADGE: Record<string, string> = {
  critical: 'bg-destructive/15 text-destructive',
  high:     'bg-warning/15 text-warning',
  medium:   'bg-amber-500/15 text-amber-600',
  low:      'bg-muted text-muted-foreground',
}

function OpenTickets(_props: WidgetProps) {
  const { t } = useTranslation()
  const tickets = useHelpdeskStore((s) => s.tickets)

  const open = useMemo(
    () =>
      tickets
        .filter((tk) => tk.status === 'open' || tk.status === 'in_progress')
        .sort((a, b) => {
          // Sort: SLA-overdue first, then by priority
          if (a.slaOverdue !== b.slaOverdue) return a.slaOverdue ? -1 : 1
          const pOrder = { critical: 0, high: 1, medium: 2, low: 3 }
          return (pOrder[a.priority] ?? 4) - (pOrder[b.priority] ?? 4)
        })
        .slice(0, 8),
    [tickets],
  )

  const overdueCount = useMemo(
    () => open.filter((tk) => tk.slaOverdue).length,
    [open],
  )

  if (open.length === 0) {
    return (
      <div className="flex h-full items-center justify-center px-4">
        <p className="text-xs text-muted-foreground">{t('dashboard.openTickets.noTickets')}</p>
      </div>
    )
  }

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="flex items-center justify-between px-4 pt-4 pb-2">
        <div className="flex items-center gap-2">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-orange-500/10">
            <TicketCheck className="h-4 w-4 text-orange-600" />
          </div>
          <div>
            <p className="text-xs font-medium text-muted-foreground">
              {t('dashboard.openTickets.title')}
            </p>
            <p className="text-sm font-bold text-foreground tabular-nums">{open.length}</p>
          </div>
        </div>
        {overdueCount > 0 && (
          <div className="flex items-center gap-1 rounded-md bg-destructive/10 px-2 py-1">
            <AlertCircle className="h-3.5 w-3.5 text-destructive" />
            <span className="text-xs font-semibold text-destructive tabular-nums">
              {t('dashboard.openTickets.overdueSla', { count: overdueCount })}
            </span>
          </div>
        )}
      </div>

      {/* Ticket list */}
      <div className="flex-1 overflow-auto divide-y divide-border">
        {open.map((tk) => (
          <div
            key={tk.id}
            className="flex items-start gap-3 px-4 py-2 hover:bg-accent/50 transition-colors"
            data-testid="open-ticket-row"
          >
            <div className="min-w-0 flex-1">
              <p
                className="text-xs font-medium text-foreground truncate"
                data-testid="open-ticket-subject"
              >
                {tk.subject}
              </p>
              <div className="mt-0.5 flex items-center gap-2">
                <span className="text-[10px] text-muted-foreground">{tk.ticketNr}</span>
                {tk.slaOverdue && (
                  <span className="text-[10px] font-semibold text-destructive">
                    {slaLabel(t, tk)}
                  </span>
                )}
              </div>
            </div>
            <span
              className={`shrink-0 rounded-sm px-1.5 py-0.5 text-[10px] font-medium ${PRIORITY_BADGE[tk.priority] ?? PRIORITY_BADGE.low}`}
            >
              {t(`dashboard.openTickets.priority.${tk.priority}`)}
            </span>
          </div>
        ))}
      </div>
    </div>
  )
}

export default memo(OpenTickets)

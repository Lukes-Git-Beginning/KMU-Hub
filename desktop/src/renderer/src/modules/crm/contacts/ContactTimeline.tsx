/**
 * ContactTimeline — Chronological view of all interactions for a contact.
 *
 * Shows emails, calls, meetings, deals, tickets, notes in a vertical timeline.
 * Filter by type. Fetches data from GET /api/v1/crm/contacts/{id}/timeline.
 */
import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Clock,
  Filter,
  Loader2,
} from 'lucide-react'
import { TimelineItem, type TimelineEntry, type TimelineActivityType } from './TimelineItem'
import { useContactTimeline, type TimelineEvent } from '@/api/hooks/useTimeline'

// ---------------------------------------------------------------------------
// Filter config
// ---------------------------------------------------------------------------

const filterOptions: { value: TimelineActivityType | 'all'; labelKey: string }[] = [
  { value: 'all', labelKey: 'crm.timeline.filterAll' },
  { value: 'email', labelKey: 'crm.timeline.filterEmails' },
  { value: 'call', labelKey: 'crm.timeline.filterCalls' },
  { value: 'meeting', labelKey: 'crm.timeline.filterMeetings' },
  { value: 'deal', labelKey: 'crm.timeline.filterDeals' },
  { value: 'ticket', labelKey: 'crm.timeline.filterTickets' },
  { value: 'note', labelKey: 'crm.timeline.filterNotes' },
]

// ---------------------------------------------------------------------------
// Map backend TimelineEvent → frontend TimelineEntry
// ---------------------------------------------------------------------------

function mapActivityType(event: TimelineEvent): TimelineActivityType {
  if (event.event_type === 'deal_linked') return 'deal'

  // For activities, use the metadata.activity_type or title heuristics
  const actType = (event.metadata?.activity_type as string) ?? ''
  const typeMap: Record<string, TimelineActivityType> = {
    call: 'call',
    email: 'email',
    meeting: 'meeting',
    note: 'note',
    task: 'note',
  }
  return typeMap[actType] ?? 'note'
}

function toTimelineEntry(event: TimelineEvent): TimelineEntry {
  // Build metadata record for display, filtering out internal keys
  const displayMeta: Record<string, string> = {}
  if (event.metadata) {
    for (const [k, v] of Object.entries(event.metadata)) {
      if (k !== 'activity_type' && v != null) {
        displayMeta[k] = String(v)
      }
    }
  }

  return {
    id: event.id,
    type: mapActivityType(event),
    title: event.title,
    description: event.description,
    date: event.occurred_at,
    user: event.created_by_name,
    metadata: Object.keys(displayMeta).length > 0 ? displayMeta : undefined,
  }
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface ContactTimelineProps {
  contactId?: string
}

export function ContactTimeline({ contactId }: ContactTimelineProps) {
  const { t } = useTranslation()
  const [filter, setFilter] = useState<TimelineActivityType | 'all'>('all')
  const [visibleCount, setVisibleCount] = useState(8)

  const { data, isLoading, isError } = useContactTimeline(contactId, 1, 100)

  const entries = useMemo(() => {
    if (!data?.events) return []
    return data.events.map(toTimelineEntry)
  }, [data])

  const filtered = useMemo(() => {
    if (filter === 'all') return entries
    return entries.filter((e) => e.type === filter)
  }, [filter, entries])

  const visible = filtered.slice(0, visibleCount)
  const hasMore = visibleCount < filtered.length

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (isError) {
    return (
      <div className="py-8 text-center text-sm text-destructive">
        {t('crm.timeline.loadError')}
      </div>
    )
  }

  return (
    <div className="space-y-3">
      {/* Header + filters */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 text-sm font-medium text-foreground">
          <Clock className="h-4 w-4 text-muted-foreground" />
          {t('crm.timeline.title')}
          <span className="text-xs text-muted-foreground font-normal">({filtered.length})</span>
        </div>
        <div className="flex items-center gap-1">
          <Filter className="h-3 w-3 text-muted-foreground" />
          {filterOptions.map((opt) => (
            <button
              key={opt.value}
              onClick={() => { setFilter(opt.value); setVisibleCount(8) }}
              className={`rounded-md px-2 py-1 text-[10px] transition-colors ${
                filter === opt.value
                  ? 'bg-primary text-primary-foreground font-medium'
                  : 'text-muted-foreground hover:bg-accent hover:text-foreground'
              }`}
            >
              {t(opt.labelKey)}
            </button>
          ))}
        </div>
      </div>

      {/* Timeline */}
      {visible.length === 0 ? (
        <div className="py-8 text-center text-sm text-muted-foreground">
          {t('crm.timeline.noResults')}
        </div>
      ) : (
        <div>
          {visible.map((entry, i) => (
            <TimelineItem
              key={entry.id}
              entry={entry}
              isLast={i === visible.length - 1}
            />
          ))}
        </div>
      )}

      {/* Load more */}
      {hasMore && (
        <button
          onClick={() => setVisibleCount((c) => c + 8)}
          className="w-full rounded-md border border-border py-2 text-xs text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
        >
          {t('crm.timeline.loadMore', { count: filtered.length - visibleCount })}
        </button>
      )}
    </div>
  )
}

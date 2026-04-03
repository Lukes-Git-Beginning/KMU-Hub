/**
 * ContactTimeline — Chronological view of all interactions for a contact.
 *
 * Shows emails, calls, meetings, deals, tickets, notes in a vertical timeline.
 * Filter by type. Mock data for design. Backend swap: replace with API hook.
 */
import { useState, useMemo } from 'react'
import {
  Clock,
  Filter,
} from 'lucide-react'
import { TimelineItem, type TimelineEntry, type TimelineActivityType } from './TimelineItem'

// ---------------------------------------------------------------------------
// Filter config
// ---------------------------------------------------------------------------

const filterOptions: { value: TimelineActivityType | 'all'; label: string }[] = [
  { value: 'all', label: 'Alle' },
  { value: 'email', label: 'E-Mails' },
  { value: 'call', label: 'Anrufe' },
  { value: 'meeting', label: 'Meetings' },
  { value: 'deal', label: 'Deals' },
  { value: 'ticket', label: 'Tickets' },
  { value: 'note', label: 'Notizen' },
]

// ---------------------------------------------------------------------------
// Mock timeline data
// ---------------------------------------------------------------------------

const mockTimeline: TimelineEntry[] = [
  {
    id: 'tl1',
    type: 'meeting',
    title: 'Sprint Planning Q1',
    description: 'Besprechung der Roadmap-Prioritaeten für Q1 2026. Teilnehmer: Anna, Michael, Sarah.',
    date: '2026-02-18',
    user: 'Anna Müller',
    metadata: { Dauer: '60 min', Ort: 'Konferenzraum 1' },
  },
  {
    id: 'tl2',
    type: 'email',
    title: 'Projektupdate versendet',
    description: 'Statusbericht Woche 7 an alle Stakeholder versendet.',
    date: '2026-02-17',
    user: 'Anna Müller',
  },
  {
    id: 'tl3',
    type: 'deal',
    title: 'CRM-Lizenz Erweiterung',
    description: 'Deal auf Phase "Verhandlung" verschoben. Wert: CHF 45.000.',
    date: '2026-02-15',
    user: 'Thomas Weber',
    metadata: { Phase: 'Verhandlung', Wert: 'CHF 45.000' },
  },
  {
    id: 'tl4',
    type: 'call',
    title: 'Rückruf wegen Deadline',
    description: 'Telefonat zur Klärung der Lieferfrist für Phase 2.',
    date: '2026-02-14',
    user: 'Michael Berg',
    metadata: { Dauer: '15 min' },
  },
  {
    id: 'tl5',
    type: 'ticket',
    title: 'Login-Problem gemeldet',
    description: 'Kunde kann sich nicht mehr einloggen. Ticket #T-2026-089 erstellt.',
    date: '2026-02-13',
    user: 'Support Team',
    metadata: { Ticket: 'T-2026-089', Prioritaet: 'Hoch' },
  },
  {
    id: 'tl6',
    type: 'note',
    title: 'Interne Notiz',
    description: 'Kunde hat Interesse an Workshop-Paket für Mitarbeiterschulung geaeussert.',
    date: '2026-02-12',
    user: 'Lisa Schmidt',
  },
  {
    id: 'tl7',
    type: 'email',
    title: 'Angebot für Phase 2 versendet',
    description: 'Detailliertes Angebot über EUR 35.000 inkl. Schulung und Support.',
    date: '2026-02-10',
    user: 'Anna Müller',
    metadata: { Betrag: 'EUR 35.000' },
  },
  {
    id: 'tl8',
    type: 'meeting',
    title: 'Quartalsbesprechung',
    description: 'Review der Zusammenarbeit und Planung nächste Schritte.',
    date: '2026-02-05',
    user: 'Thomas Weber',
    metadata: { Dauer: '90 min', Ort: 'Beim Kunden' },
  },
  {
    id: 'tl9',
    type: 'document',
    title: 'Vertrag aktualisiert',
    description: 'SLA-Vertrag um 12 Monate verlaengert.',
    date: '2026-02-01',
    user: 'Eva Brunner',
    metadata: { Typ: 'SLA', Laufzeit: '12 Monate' },
  },
  {
    id: 'tl10',
    type: 'call',
    title: 'Feedback zum Prototyp',
    description: 'Positives Feedback zur neuen Kalender-Integration.',
    date: '2026-01-28',
    user: 'Thomas Weber',
    metadata: { Dauer: '25 min' },
  },
  {
    id: 'tl11',
    type: 'message',
    title: 'Chat-Nachricht',
    description: 'Kurze Rückfrage zur API-Dokumentation beantwortet.',
    date: '2026-01-25',
    user: 'Michael Berg',
  },
  {
    id: 'tl12',
    type: 'email',
    title: 'Infomaterial versendet',
    description: 'Produktbroschuere und Preisliste an neuen Ansprechpartner.',
    date: '2026-01-20',
    user: 'Lisa Schmidt',
  },
]

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface ContactTimelineProps {
  contactId?: string
}

export function ContactTimeline({ contactId: _contactId }: ContactTimelineProps) {
  const [filter, setFilter] = useState<TimelineActivityType | 'all'>('all')
  const [visibleCount, setVisibleCount] = useState(8)

  const filtered = useMemo(() => {
    if (filter === 'all') return mockTimeline
    return mockTimeline.filter((e) => e.type === filter)
  }, [filter])

  const visible = filtered.slice(0, visibleCount)
  const hasMore = visibleCount < filtered.length

  return (
    <div className="space-y-3">
      {/* Header + filters */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 text-sm font-medium text-foreground">
          <Clock className="h-4 w-4 text-muted-foreground" />
          Aktivitaeten-Timeline
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
              {opt.label}
            </button>
          ))}
        </div>
      </div>

      {/* Timeline */}
      {visible.length === 0 ? (
        <div className="py-8 text-center text-sm text-muted-foreground">
          Keine Aktivitaeten für diesen Filter.
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
          Weitere laden ({filtered.length - visibleCount} übrig)
        </button>
      )}
    </div>
  )
}

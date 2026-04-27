/**
 * TimelineItem — Single entry in the contact timeline.
 *
 * Renders with connecting line, type-colored icon, and content.
 */
import { useTranslation } from 'react-i18next'
import {
  Mail,
  Phone,
  Calendar,
  FileText,
  MessageSquare,
  Handshake,
  Ticket,
  TrendingUp,
} from 'lucide-react'

import { formatDate } from '@/lib/format'

// ---------------------------------------------------------------------------
// Activity type config
// ---------------------------------------------------------------------------

export type TimelineActivityType =
  | 'email'
  | 'call'
  | 'meeting'
  | 'note'
  | 'message'
  | 'deal'
  | 'ticket'
  | 'document'

const typeConfig: Record<
  TimelineActivityType,
  { icon: typeof Mail; bg: string; color: string; labelKey: string }
> = {
  email: { icon: Mail, bg: 'bg-blue-100 dark:bg-blue-900/30', color: 'text-blue-600 dark:text-blue-400', labelKey: 'crm.timeline.type.email' },
  call: { icon: Phone, bg: 'bg-green-100 dark:bg-green-900/30', color: 'text-green-600 dark:text-green-400', labelKey: 'crm.timeline.type.call' },
  meeting: { icon: Calendar, bg: 'bg-purple-100 dark:bg-purple-900/30', color: 'text-purple-600 dark:text-purple-400', labelKey: 'crm.timeline.type.meeting' },
  note: { icon: FileText, bg: 'bg-amber-100 dark:bg-amber-900/30', color: 'text-amber-600 dark:text-amber-400', labelKey: 'crm.timeline.type.note' },
  message: { icon: MessageSquare, bg: 'bg-cyan-100 dark:bg-cyan-900/30', color: 'text-cyan-600 dark:text-cyan-400', labelKey: 'crm.timeline.type.message' },
  deal: { icon: TrendingUp, bg: 'bg-emerald-100 dark:bg-emerald-900/30', color: 'text-emerald-600 dark:text-emerald-400', labelKey: 'crm.timeline.type.deal' },
  ticket: { icon: Ticket, bg: 'bg-rose-100 dark:bg-rose-900/30', color: 'text-rose-600 dark:text-rose-400', labelKey: 'crm.timeline.type.ticket' },
  document: { icon: Handshake, bg: 'bg-slate-100 dark:bg-slate-800/50', color: 'text-slate-600 dark:text-slate-400', labelKey: 'crm.timeline.type.document' },
}

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface TimelineEntry {
  id: string
  type: TimelineActivityType
  title: string
  description?: string
  date: string
  user?: string
  metadata?: Record<string, string>
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface TimelineItemProps {
  entry: TimelineEntry
  isLast: boolean
}

export function TimelineItem({ entry, isLast }: TimelineItemProps) {
  const { t } = useTranslation()
  const tc = typeConfig[entry.type]
  const Icon = tc.icon

  const formattedDate = (() => {
    try {
      const d = new Date(entry.date + (entry.date.includes('T') ? '' : 'T00:00:00'))
      const now = new Date()
      const diff = now.getTime() - d.getTime()
      const days = Math.floor(diff / 86400000)

      if (days === 0) return t('crm.timeline.today')
      if (days === 1) return t('crm.timeline.yesterday')
      if (days < 7) return t('crm.timeline.daysAgo', { days })
      return formatDate(d, { day: '2-digit', month: '2-digit', year: 'numeric' })
    } catch {
      return entry.date
    }
  })()

  return (
    <div className="flex gap-3">
      {/* Left: icon + connecting line */}
      <div className="flex flex-col items-center">
        <div className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-full ${tc.bg}`}>
          <Icon className={`h-4 w-4 ${tc.color}`} />
        </div>
        {!isLast && <div className="w-px flex-1 bg-border my-1" />}
      </div>

      {/* Right: content */}
      <div className={`flex-1 pb-4 ${isLast ? '' : ''}`}>
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium text-foreground">{entry.title}</span>
          <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${tc.bg} ${tc.color}`}>
            {t(tc.labelKey)}
          </span>
        </div>
        {entry.description && (
          <p className="mt-0.5 text-xs text-muted-foreground line-clamp-2">{entry.description}</p>
        )}
        <div className="mt-1 flex items-center gap-2 text-[10px] text-muted-foreground">
          <span>{formattedDate}</span>
          {entry.user && (
            <>
              <span>·</span>
              <span>{entry.user}</span>
            </>
          )}
        </div>
        {entry.metadata && Object.keys(entry.metadata).length > 0 && (
          <div className="mt-1.5 flex flex-wrap gap-1.5">
            {Object.entries(entry.metadata).map(([key, value]) => (
              <span
                key={key}
                className="rounded-md bg-secondary px-2 py-0.5 text-[10px] text-muted-foreground"
              >
                {key}: {value}
              </span>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

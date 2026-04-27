/**
 * Calendar Upcoming widget — today's events from real API.
 */
import { memo, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Clock, MapPin } from 'lucide-react'
import { useCalendars } from '@/api/hooks/useCalendars'
import { useEventsInRange } from '@/api/hooks/useEvents'
import { expandedEventToUI } from '@/modules/kalender/adapters'
import type { WidgetProps } from '@/components/widgets/WidgetRegistry'
import { formatDate } from '@/lib/format'

function CalendarUpcoming(_props: WidgetProps) {
  const { t } = useTranslation()
  const { todayStart, todayEnd, today, dd } = useMemo(() => {
    const today = new Date()
    const dd = String(today.getDate()).padStart(2, '0')
    const todayStr = `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, '0')}-${dd}`
    return { todayStart: `${todayStr}T00:00:00Z`, todayEnd: `${todayStr}T23:59:59Z`, today, dd }
  }, [])

  const { data: calData } = useCalendars()
  const calendarIds = useMemo(() => {
    const calendars = (calData as { calendars?: Array<{ id: string }> })?.calendars ?? []
    return calendars.map((c) => c.id)
  }, [calData])

  const { data: eventData, isLoading } = useEventsInRange(calendarIds, todayStart, todayEnd)

  const events = useMemo(() => {
    const rawEvents = (eventData as { events?: Array<Record<string, unknown>> })?.events ?? []
    return rawEvents
      .map((ev) => {
        const uiEvent = expandedEventToUI(ev as Parameters<typeof expandedEventToUI>[0])
        // Compute duration string
        const [sh, sm] = uiEvent.startTime.split(':').map(Number)
        const [eh, em] = uiEvent.endTime.split(':').map(Number)
        const diffMin = (eh * 60 + em) - (sh * 60 + sm)
        let duration = ''
        if (uiEvent.isAllDay) {
          duration = t('dashboard.calendar.allDay')
        } else if (diffMin >= 60) {
          const hours = Math.floor(diffMin / 60)
          const mins = diffMin % 60
          duration = mins > 0 ? `${hours} ${t('dashboard.calendar.hours')} ${mins} ${t('dashboard.calendar.minutes')}` : `${hours} ${t('dashboard.calendar.hours')}`
        } else {
          duration = `${diffMin} ${t('dashboard.calendar.minutes')}`
        }

        return {
          id: uiEvent.id,
          title: uiEvent.title,
          time: uiEvent.startTime,
          duration,
          location: uiEvent.location,
          color: uiEvent.color ?? '#3b82f6',
        }
      })
      .sort((a, b) => a.time.localeCompare(b.time))
  }, [eventData, t])

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center p-4">
        <div role="status" aria-label={t('common.loading')} className="h-5 w-5 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
    )
  }

  return (
    <div className="flex h-full flex-col">
      {/* Date header */}
      <div className="flex items-center gap-2 px-4 pt-4 pb-2">
        <div className="flex h-10 w-10 flex-col items-center justify-center rounded-lg bg-primary/10">
          <span className="text-[9px] font-bold text-primary uppercase leading-none">
            {formatDate(today, { weekday: 'short' })}
          </span>
          <span className="text-sm font-bold text-primary leading-tight">{dd}</span>
        </div>
        <div>
          <p className="text-sm font-medium text-foreground">
            {formatDate(today, { weekday: 'long', day: 'numeric', month: 'long' })}
          </p>
          <p className="text-xs text-muted-foreground">{t('dashboard.calendar.eventsToday', { count: events.length })}</p>
        </div>
      </div>

      {/* Events list */}
      {events.length > 0 ? (
        <div className="flex-1 overflow-auto divide-y divide-border">
          {events.map((event) => (
            <div
              key={event.id}
              role="button"
              tabIndex={0}
              className="flex items-start gap-3 px-4 py-2.5 hover:bg-accent/50 cursor-pointer transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50 focus-visible:ring-inset"
            >
              <div className="mt-1 h-2 w-2 shrink-0 rounded-full" style={{ backgroundColor: event.color }} />
              <div className="min-w-0 flex-1">
                <p className="text-sm font-medium text-foreground truncate">{event.title}</p>
                <div className="flex items-center gap-2 mt-0.5">
                  <Clock className="h-3 w-3 text-muted-foreground" />
                  <span className="text-xs text-muted-foreground">{event.time} · {event.duration}</span>
                  {event.location && (
                    <>
                      <MapPin className="h-3 w-3 text-muted-foreground ml-1" />
                      <span className="text-xs text-muted-foreground truncate">{event.location}</span>
                    </>
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>
      ) : (
        <div className="flex-1 flex items-center justify-center p-4">
          <p className="text-sm text-muted-foreground">{t('dashboard.calendar.noEventsToday')}</p>
        </div>
      )}
    </div>
  )
}

export default memo(CalendarUpcoming)

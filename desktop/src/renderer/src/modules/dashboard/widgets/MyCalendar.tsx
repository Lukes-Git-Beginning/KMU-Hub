/**
 * My Calendar widget — personal schedule for today.
 */
import { memo, useMemo } from 'react'
import { Video, MapPin, Users } from 'lucide-react'
import { useCalendars } from '@/api/hooks/useCalendars'
import { useEventsInRange } from '@/api/hooks/useEvents'
import { expandedEventToUI } from '@/modules/kalender/adapters'
import type { WidgetProps } from '@/components/widgets/WidgetRegistry'

const TYPE_STYLE: Record<string, { color: string; label: string }> = {
  meeting: { color: 'bg-blue-500', label: 'Meeting' },
  focus: { color: 'bg-emerald-500', label: 'Fokus' },
  call: { color: 'bg-violet-500', label: 'Anruf' },
  break: { color: 'bg-amber-500', label: 'Pause' },
  workshop: { color: 'bg-teal-500', label: 'Workshop' },
}

function MyCalendar(_props: WidgetProps) {
  const now = new Date()
  const todayStr = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`
  const todayStart = `${todayStr}T00:00:00Z`
  const todayEnd = `${todayStr}T23:59:59Z`

  const { data: calData } = useCalendars()
  const calendarIds = useMemo(() => {
    const calendars = (calData as { calendars?: Array<{ id: string }> })?.calendars ?? []
    return calendars.map((c) => c.id)
  }, [calData])

  const { data: eventData, isLoading } = useEventsInRange(calendarIds, todayStart, todayEnd)

  const schedule = useMemo(() => {
    const events = (eventData as { events?: Array<Record<string, unknown>> })?.events ?? []
    return events
      .map((ev) => {
        const uiEvent = expandedEventToUI(ev as Parameters<typeof expandedEventToUI>[0])
        return {
          id: uiEvent.id,
          title: uiEvent.title,
          time: uiEvent.startTime,
          endTime: uiEvent.endTime,
          type: 'meeting' as string,
          location: uiEvent.location,
          color: uiEvent.color,
          attendeeCount: uiEvent.participants?.length ?? 0,
        }
      })
      .sort((a, b) => a.time.localeCompare(b.time))
  }, [eventData])

  const currentHour = now.getHours()
  const currentMin = now.getMinutes()
  const currentTimeStr = `${String(currentHour).padStart(2, '0')}:${String(currentMin).padStart(2, '0')}`

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center p-4">
        <div className="h-5 w-5 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
    )
  }

  if (!schedule.length) {
    return (
      <div className="flex h-full flex-col">
        <div className="flex items-center justify-between px-4 pt-4 pb-2">
          <span className="text-xs font-medium text-muted-foreground">
            {now.toLocaleDateString('de-DE', { weekday: 'long', day: 'numeric', month: 'long' })}
          </span>
          <span className="text-xs font-mono text-primary font-semibold">{currentTimeStr}</span>
        </div>
        <div className="flex-1 flex items-center justify-center p-4">
          <p className="text-sm text-muted-foreground">Keine Termine heute</p>
        </div>
      </div>
    )
  }

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="flex items-center justify-between px-4 pt-4 pb-2">
        <span className="text-xs font-medium text-muted-foreground">
          {now.toLocaleDateString('de-DE', { weekday: 'long', day: 'numeric', month: 'long' })}
        </span>
        <span className="text-xs font-mono text-primary font-semibold">{currentTimeStr}</span>
      </div>

      {/* Timeline */}
      <div className="flex-1 overflow-auto">
        {schedule.map((slot) => {
          const isPast = slot.endTime < currentTimeStr
          const isCurrent = slot.time <= currentTimeStr && slot.endTime > currentTimeStr
          const style = TYPE_STYLE[slot.type] ?? TYPE_STYLE.meeting

          return (
            <div
              key={slot.id}
              className={`flex items-start gap-3 px-4 py-2 border-l-2 ml-4 transition-colors cursor-pointer hover:bg-accent/50 ${
                isCurrent ? 'border-l-primary bg-primary/5' : isPast ? 'border-l-border opacity-50' : 'border-l-border'
              }`}
            >
              {/* Time */}
              <div className="w-12 shrink-0 text-right">
                <p className={`text-xs font-mono ${isCurrent ? 'text-primary font-bold' : 'text-muted-foreground'}`}>
                  {slot.time}
                </p>
                <p className="text-[10px] text-muted-foreground">{slot.endTime}</p>
              </div>

              {/* Event */}
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-1.5">
                  <span className={`h-2 w-2 rounded-full shrink-0 ${style.color}`} />
                  <p className={`text-sm truncate ${isCurrent ? 'font-semibold text-foreground' : isPast ? 'text-muted-foreground' : 'font-medium text-foreground'}`}>
                    {slot.title}
                  </p>
                </div>
                <div className="flex items-center gap-2 mt-0.5">
                  {slot.location && (
                    <span className="flex items-center gap-0.5 text-[10px] text-muted-foreground">
                      {slot.location.includes('Video') ? <Video className="h-2.5 w-2.5" /> : <MapPin className="h-2.5 w-2.5" />}
                      {slot.location}
                    </span>
                  )}
                  {slot.attendeeCount > 0 && (
                    <span className="flex items-center gap-0.5 text-[10px] text-muted-foreground">
                      <Users className="h-2.5 w-2.5" />
                      {slot.attendeeCount}
                    </span>
                  )}
                </div>
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}

export default memo(MyCalendar)

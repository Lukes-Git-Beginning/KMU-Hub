/**
 * Calendar Upcoming widget — today's events from central mock-db.
 */
import { memo } from 'react'
import { Clock, MapPin } from 'lucide-react'
import { TODAY_EVENTS } from '@/mocks/mock-db'
import type { WidgetProps } from '@/components/widgets/WidgetRegistry'

const today = new Date()
const dd = String(today.getDate()).padStart(2, '0')

function CalendarUpcoming(_props: WidgetProps) {
  return (
    <div className="flex h-full flex-col">
      {/* Date header */}
      <div className="flex items-center gap-2 px-4 pt-4 pb-2">
        <div className="flex h-10 w-10 flex-col items-center justify-center rounded-lg bg-primary/10">
          <span className="text-[9px] font-bold text-primary uppercase leading-none">
            {today.toLocaleDateString('de-DE', { weekday: 'short' })}
          </span>
          <span className="text-sm font-bold text-primary leading-tight">{dd}</span>
        </div>
        <div>
          <p className="text-sm font-medium text-foreground">
            {today.toLocaleDateString('de-DE', { weekday: 'long', day: 'numeric', month: 'long' })}
          </p>
          <p className="text-xs text-muted-foreground">{TODAY_EVENTS.length} Termine heute</p>
        </div>
      </div>

      {/* Events list */}
      <div className="flex-1 overflow-auto divide-y divide-border">
        {TODAY_EVENTS.map((event) => (
          <div
            key={event.id}
            className="flex items-start gap-3 px-4 py-2.5 hover:bg-accent/50 cursor-pointer transition-colors"
          >
            <div className={`mt-1 h-2 w-2 shrink-0 rounded-full`} style={{ backgroundColor: event.color }} />
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
    </div>
  )
}

export default memo(CalendarUpcoming)

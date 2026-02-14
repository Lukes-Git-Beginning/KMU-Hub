/**
 * Scrollable timeline header with day/week/month columns.
 *
 * Renders a sticky dual-row header: top row shows month/year grouping,
 * bottom row shows individual date columns. Weekend columns have a
 * subtle tint. Today column is highlighted.
 */
import { cn } from '@/lib'
import type { TimelineConfig, TimelineColumn } from './gantt-utils'
import {
  generateTimelineColumns,
  generateGroupHeaders,
  HEADER_HEIGHT,
} from './gantt-utils'

interface GanttTimelineProps {
  config: TimelineConfig
}

export default function GanttTimeline({ config }: GanttTimelineProps) {
  const columns = generateTimelineColumns(config)
  const groups = generateGroupHeaders(columns, config)

  return (
    <div
      className="sticky top-0 z-10 border-b border-border bg-background"
      style={{ height: HEADER_HEIGHT }}
    >
      {/* Top row: grouped headers (months or years) */}
      <div className="flex" style={{ height: HEADER_HEIGHT / 2 }}>
        {groups.map((g, i) => (
          <div
            key={`${g.label}-${i}`}
            className="flex items-center justify-center border-r border-border text-xs font-medium text-muted-foreground truncate"
            style={{ width: g.span * config.columnWidth }}
          >
            {g.label}
          </div>
        ))}
      </div>

      {/* Bottom row: individual columns */}
      <div className="flex" style={{ height: HEADER_HEIGHT / 2 }}>
        {columns.map((col, i) => (
          <div
            key={i}
            className={cn(
              'flex items-center justify-center border-r border-border text-xs',
              col.isWeekend && 'bg-muted/30',
              col.isToday && 'bg-primary/10 font-semibold text-primary'
            )}
            style={{ width: config.columnWidth }}
          >
            {col.label}
          </div>
        ))}
      </div>
    </div>
  )
}

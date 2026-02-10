/**
 * Calendar module layout with collapsible sidebar and main view area.
 *
 * The sidebar shows mini-calendar, calendar list, and controls (implemented
 * in plan 07-08). The main area hosts the active calendar view (day/week/month)
 * implemented in plan 07-06. This shell provides the structural layout and
 * toolbar with navigation, view switcher, and today button.
 */
import { Routes, Route, Navigate } from 'react-router-dom'
import {
  ChevronLeft,
  ChevronRight,
  CalendarDays,
  PanelLeftClose,
  PanelLeftOpen,
} from 'lucide-react'
import { format } from 'date-fns'
import { de } from 'date-fns/locale'
import { cn } from '@/lib/cn'
import { Button } from '@/components/ui/button'
import { useCalendarStore, type CalendarView } from '@/stores/calendar'

// ---------------------------------------------------------------------------
// View switcher options
// ---------------------------------------------------------------------------

const viewOptions: { value: CalendarView; label: string }[] = [
  { value: 'day', label: 'Tag' },
  { value: 'week', label: 'Woche' },
  { value: 'month', label: 'Monat' },
]

// ---------------------------------------------------------------------------
// Layout
// ---------------------------------------------------------------------------

export default function CalendarLayout() {
  const currentView = useCalendarStore((s) => s.currentView)
  const currentDate = useCalendarStore((s) => s.currentDate)
  const sidebarOpen = useCalendarStore((s) => s.sidebarOpen)
  const setCurrentView = useCalendarStore((s) => s.setCurrentView)
  const goToToday = useCalendarStore((s) => s.goToToday)
  const navigateForward = useCalendarStore((s) => s.navigateForward)
  const navigateBackward = useCalendarStore((s) => s.navigateBackward)
  const toggleSidebar = useCalendarStore((s) => s.toggleSidebar)

  /** Format the header date label based on current view. */
  const dateLabel = (() => {
    switch (currentView) {
      case 'day':
        return format(currentDate, 'EEEE, d. MMMM yyyy', { locale: de })
      case 'week':
        return format(currentDate, 'MMMM yyyy', { locale: de })
      case 'month':
        return format(currentDate, 'MMMM yyyy', { locale: de })
    }
  })()

  return (
    <div className="flex h-full flex-col">
      {/* Toolbar */}
      <div className="flex items-center gap-2 border-b border-border bg-card px-4 py-2">
        {/* Sidebar toggle */}
        <Button
          variant="ghost"
          size="icon"
          className="h-8 w-8"
          onClick={toggleSidebar}
          title={sidebarOpen ? 'Seitenleiste einklappen' : 'Seitenleiste ausklappen'}
        >
          {sidebarOpen ? (
            <PanelLeftClose className="h-4 w-4" />
          ) : (
            <PanelLeftOpen className="h-4 w-4" />
          )}
        </Button>

        {/* Today button */}
        <Button variant="outline" size="sm" onClick={goToToday}>
          Heute
        </Button>

        {/* Navigation arrows */}
        <Button
          variant="ghost"
          size="icon"
          className="h-8 w-8"
          onClick={navigateBackward}
          title="Zurueck"
        >
          <ChevronLeft className="h-4 w-4" />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          className="h-8 w-8"
          onClick={navigateForward}
          title="Vorwaerts"
        >
          <ChevronRight className="h-4 w-4" />
        </Button>

        {/* Date label */}
        <h2 className="min-w-0 flex-1 truncate text-lg font-semibold text-foreground">
          {dateLabel}
        </h2>

        {/* View switcher */}
        <div className="flex items-center rounded-md border border-border bg-background p-0.5">
          {viewOptions.map((opt) => (
            <button
              key={opt.value}
              onClick={() => setCurrentView(opt.value)}
              className={cn(
                'rounded-sm px-3 py-1 text-sm font-medium transition-colors',
                currentView === opt.value
                  ? 'bg-secondary text-secondary-foreground'
                  : 'text-muted-foreground hover:text-foreground',
              )}
            >
              {opt.label}
            </button>
          ))}
        </div>
      </div>

      {/* Body: sidebar + main */}
      <div className="flex flex-1 overflow-hidden">
        {/* Sidebar */}
        {sidebarOpen && (
          <aside className="flex w-[280px] shrink-0 flex-col border-r border-border bg-card">
            {/* Placeholder -- sidebar components added in 07-08 */}
            <div className="flex flex-1 flex-col items-center justify-center p-4 text-muted-foreground">
              <CalendarDays className="mb-2 h-8 w-8" />
              <p className="text-sm">Kalender-Seitenleiste</p>
              <p className="text-xs">(Wird in Plan 07-08 implementiert)</p>
            </div>
          </aside>
        )}

        {/* Main content area -- calendar views added in 07-06 */}
        <div className="flex-1 overflow-auto">
          <Routes>
            <Route
              index
              element={<Navigate to="/calendar/view" replace />}
            />
            <Route
              path="view"
              element={
                <div className="flex h-full items-center justify-center text-muted-foreground">
                  <div className="text-center">
                    <CalendarDays className="mx-auto mb-3 h-12 w-12" />
                    <p className="text-lg font-medium">Kalender</p>
                    <p className="text-sm">
                      Ansicht: {currentView === 'day' ? 'Tag' : currentView === 'week' ? 'Woche' : 'Monat'}
                    </p>
                    <p className="text-sm">
                      {format(currentDate, 'd. MMMM yyyy', { locale: de })}
                    </p>
                    <p className="mt-2 text-xs">
                      (Kalender-Ansichten werden in Plan 07-06 implementiert)
                    </p>
                  </div>
                </div>
              }
            />
            <Route path="*" element={<Navigate to="/calendar/view" replace />} />
          </Routes>
        </div>
      </div>
    </div>
  )
}

import { useState, useMemo, useEffect } from 'react'
import {
  ChevronLeft,
  ChevronRight,
  Plus,
  Clock,
  MapPin,
  Users,
  Video,
  Calendar,
  X,
  Bell,
  Repeat,
  Check,
  CircleHelp,
  CircleX,
  Search,
  Settings2,
  Layers,
  DoorOpen,
  Loader2,
} from 'lucide-react'
import { RoomBookingView } from './RoomBookingView'
import { CategoryManagerDialog } from './CategoryManagerDialog'
import { CalendarBrowseDialog } from './CalendarBrowseDialog'
import {
  expandedEventToUI,
  calendarToUI,
  categoryToUI,
  holidayToUI,
  deadlineToUI,
  uiEventToCreateRequest,
  uiEventToUpdateRequest,
  HOLIDAY_CALENDAR,
  DEADLINE_CALENDAR,
  type ViewMode,
  type RSVPStatus,
  type CalendarEvent,
  type CalendarSource,
  type Participant,
  type UIEventCategory as EventCategory,
} from './adapters'
import { useCalendars, useEventCategories } from '@/api/hooks/useCalendars'
import { useEventsInRange, useCreateEvent, useUpdateEvent, useDeleteEvent, useTaskDeadlines } from '@/api/hooks/useEvents'
import { useHolidays } from '@/api/hooks/useHolidays'
import { useCalendarStore } from '@/stores/calendar'
import { useAuthStore } from '@/stores/auth'

// ============================================================
// Types (re-export UI types for sub-components)
// ============================================================

export type { CalendarEvent, CalendarSource, Participant, EventCategory, ViewMode, RSVPStatus }

interface EventLayout {
  event: CalendarEvent
  column: number
  totalColumns: number
}

interface QuickCreateState {
  date: string
  hour: number
  minute: number
  x: number
  y: number
}

// ============================================================
// Constants (fallback defaults for empty state)
// ============================================================

const DEFAULT_CATEGORIES: EventCategory[] = [
  { id: 'meeting', name: 'Meeting', color: '#3d5c7d' },
  { id: 'focus', name: 'Fokuszeit', color: '#4a7c6a' },
  { id: 'client', name: 'Kundentermin', color: '#c4873a' },
  { id: 'private', name: 'Privat', color: '#7c5a8a' },
  { id: 'travel', name: 'Reise', color: '#8a6b3d' },
]

const INITIAL_CALENDARS: CalendarSource[] = [
  { id: 'personal', name: 'Mein Kalender', group: 'mine', color: '#1e7e74', visible: true },
  { id: 'work', name: 'Arbeit', group: 'mine', color: '#3d5c7d', visible: true },
  { id: 'team', name: 'Team-Kalender', group: 'shared', color: '#c4873a', visible: true },
  { id: 'dev', name: 'Entwickler Team', group: 'shared', color: '#4a7c6a', visible: true },
  { id: 'holidays', name: 'Feiertage CH', group: 'other', color: '#9d8f85', visible: true },
  { id: 'deadlines', name: 'Task-Deadlines', group: 'other', color: '#a13f3f', visible: true },
]

const ROOMS = [
  { id: 'r1', name: 'Raum A — Besprechung', capacity: 8, tags: ['Beamer', 'Whiteboard'] },
  { id: 'r2', name: 'Raum B — Klein', capacity: 4, tags: ['Display'] },
  { id: 'r3', name: 'Telefonkabine 1', capacity: 1, tags: [] as string[] },
  { id: 'r4', name: 'Telefonkabine 2', capacity: 1, tags: [] as string[] },
]

const TEAM_MEMBERS = [
  { name: 'Anna Mueller', initials: 'AM', role: 'Project Manager' },
  { name: 'Max Berg', initials: 'MB', role: 'Senior Developer' },
  { name: 'Sarah Klein', initials: 'SK', role: 'Lead Developer' },
  { name: 'Jonas Diaz', initials: 'JD', role: 'Designer' },
  { name: 'Peter Keller', initials: 'PK', role: 'Sales Manager' },
  { name: 'Lisa Weber', initials: 'LW', role: 'HR Manager' },
  { name: 'Tom Brunner', initials: 'TB', role: 'Junior Developer' },
]

const DAYS_SHORT = ['Mo', 'Di', 'Mi', 'Do', 'Fr', 'Sa', 'So']
const MONTHS_DE = [
  'Januar', 'Februar', 'Maerz', 'April', 'Mai', 'Juni',
  'Juli', 'August', 'September', 'Oktober', 'November', 'Dezember',
]
const RECURRENCE_OPTIONS = ['Keine', 'Täglich', 'Wöchentlich', 'Monatlich', 'Jaehrlich', 'Benutzerdefiniert...']
const REMINDER_OPTIONS = ['Keine', '5 Minuten', '10 Minuten', '15 Minuten', '30 Minuten', '1 Stunde', '2 Stunden', '1 Tag']

const HOUR_HEIGHT = 60
const START_HOUR = 7
const END_HOUR = 20
const HOURS = Array.from({ length: END_HOUR - START_HOUR + 1 }, (_, i) => i + START_HOUR)

// ============================================================
// Helpers
// ============================================================

function timeToMinutes(time: string): number {
  const [h, m] = time.split(':').map(Number)
  return h * 60 + m
}

function formatDateKey(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

function isToday(d: Date): boolean {
  return formatDateKey(d) === formatDateKey(new Date())
}

function getWeekDays(date: Date, fiveDay: boolean): Date[] {
  const d = new Date(date)
  const day = d.getDay()
  const diff = d.getDate() - day + (day === 0 ? -6 : 1)
  const monday = new Date(d.setDate(diff))
  const count = fiveDay ? 5 : 7
  return Array.from({ length: count }, (_, i) => {
    const dd = new Date(monday)
    dd.setDate(monday.getDate() + i)
    return dd
  })
}

function getMonthDays(year: number, month: number) {
  const firstDay = new Date(year, month, 1)
  const lastDay = new Date(year, month + 1, 0)
  let startOffset = firstDay.getDay() - 1
  if (startOffset < 0) startOffset = 6

  const days: { date: Date; isCurrentMonth: boolean }[] = []
  for (let i = startOffset - 1; i >= 0; i--) {
    days.push({ date: new Date(year, month, -i), isCurrentMonth: false })
  }
  for (let i = 1; i <= lastDay.getDate(); i++) {
    days.push({ date: new Date(year, month, i), isCurrentMonth: true })
  }
  const remaining = 42 - days.length
  for (let i = 1; i <= remaining; i++) {
    days.push({ date: new Date(year, month + 1, i), isCurrentMonth: false })
  }
  return days
}

function getCategoryColor(event: CalendarEvent, calendars: CalendarSource[]): string {
  if (event.isHoliday) return '#9d8f85'
  if (event.isTaskDeadline) return '#a13f3f'
  const cal = calendars.find((c) => c.id === event.calendarId)
  if (cal) return cal.color
  const cat = DEFAULT_CATEGORIES.find((c) => c.id === event.categoryId)
  return cat?.color ?? '#6b6159'
}

function layoutOverlappingEvents(events: CalendarEvent[]): EventLayout[] {
  const timed = events
    .filter((e) => !e.isAllDay)
    .sort((a, b) => timeToMinutes(a.startTime) - timeToMinutes(b.startTime))

  if (timed.length === 0) return []

  const groups: CalendarEvent[][] = []
  let currentGroup: CalendarEvent[] = []
  let groupEnd = 0

  for (const event of timed) {
    const start = timeToMinutes(event.startTime)
    const end = timeToMinutes(event.endTime)
    if (currentGroup.length === 0 || start < groupEnd) {
      currentGroup.push(event)
      groupEnd = Math.max(groupEnd, end)
    } else {
      groups.push(currentGroup)
      currentGroup = [event]
      groupEnd = end
    }
  }
  if (currentGroup.length > 0) groups.push(currentGroup)

  const result: EventLayout[] = []
  for (const group of groups) {
    const columns: CalendarEvent[][] = []
    for (const event of group) {
      const start = timeToMinutes(event.startTime)
      let placed = false
      for (let col = 0; col < columns.length; col++) {
        const lastInCol = columns[col][columns[col].length - 1]
        if (timeToMinutes(lastInCol.endTime) <= start) {
          columns[col].push(event)
          placed = true
          break
        }
      }
      if (!placed) columns.push([event])
    }
    for (let col = 0; col < columns.length; col++) {
      for (const event of columns[col]) {
        result.push({ event, column: col, totalColumns: columns.length })
      }
    }
  }
  return result
}

// ============================================================
// Main Component
// ============================================================

export default function KalenderPage() {
  // ---- Zustand store (persisted view state) ----
  const view = useCalendarStore((s) => s.currentView) as ViewMode
  const setView = useCalendarStore((s) => s.setCurrentView)
  const currentDate = useCalendarStore((s) => s.currentDate)
  const setCurrentDate = useCalendarStore((s) => s.setCurrentDate)
  const visibleCalendarIds = useCalendarStore((s) => s.visibleCalendarIds)
  const toggleCalendarVisibility = useCalendarStore((s) => s.toggleCalendarVisibility)
  const showTaskDeadlines = useCalendarStore((s) => s.showTaskDeadlines)

  // ---- Auth ----
  const currentUserId = useAuthStore((s) => s.user?.id) || ''

  // ---- Local UI state ----
  const [selectedDate, setSelectedDate] = useState(new Date())
  const [selectedEvent, setSelectedEvent] = useState<CalendarEvent | null>(null)
  const [quickCreate, setQuickCreate] = useState<QuickCreateState | null>(null)
  const [showEventForm, setShowEventForm] = useState(false)
  const [eventFormDefaults, setEventFormDefaults] = useState<Partial<CalendarEvent>>({})
  const [showRoomBooking, setShowRoomBooking] = useState(false)
  const [showCategoryManager, setShowCategoryManager] = useState(false)
  const [showCalendarBrowse, setShowCalendarBrowse] = useState(false)

  // ---- API queries ----
  const { data: calendarData } = useCalendars()
  const { data: categoryData } = useEventCategories()
  const year = currentDate.getFullYear()
  const { data: holidayData } = useHolidays(year, 'CH')

  // Compute date range for event query (pad by 7 days for month view)
  const dateRange = useMemo(() => {
    const d = new Date(currentDate)
    const start = new Date(d)
    const end = new Date(d)
    start.setDate(start.getDate() - 7)
    end.setDate(end.getDate() + 38) // covers full month + overlap
    return {
      start: start.toISOString(),
      end: end.toISOString(),
    }
  }, [currentDate])

  const visibleCalendarIdsArray = useMemo(
    () => Array.from(visibleCalendarIds),
    [visibleCalendarIds],
  )

  const { data: eventData, isLoading: eventsLoading } = useEventsInRange(
    visibleCalendarIdsArray,
    dateRange.start,
    dateRange.end,
  )

  const { data: deadlineData } = useTaskDeadlines(dateRange.start, dateRange.end)

  // ---- Mutations ----
  const createEventMutation = useCreateEvent()
  const updateEventMutation = useUpdateEvent()
  const deleteEventMutation = useDeleteEvent()

  // ---- Transform backend data to UI types ----
  const categories = useMemo(() => {
    if (!categoryData?.categories?.length) return DEFAULT_CATEGORIES
    return categoryData.categories.map(categoryToUI)
  }, [categoryData])

  const calendars: CalendarSource[] = useMemo(() => {
    const apiCalendars = calendarData?.calendars?.map((c) =>
      calendarToUI(c, currentUserId, visibleCalendarIds),
    ) || []
    // Add synthetic calendars
    return [
      ...apiCalendars,
      { ...HOLIDAY_CALENDAR, visible: visibleCalendarIds.has('holidays') || !visibleCalendarIds.size },
      ...(showTaskDeadlines
        ? [{ ...DEADLINE_CALENDAR, visible: visibleCalendarIds.has('deadlines') || !visibleCalendarIds.size }]
        : []),
    ]
  }, [calendarData, currentUserId, visibleCalendarIds, showTaskDeadlines])

  // Auto-enable all calendars on first load
  useEffect(() => {
    if (calendarData?.calendars && visibleCalendarIds.size === 0) {
      const store = useCalendarStore.getState()
      for (const cal of calendarData.calendars) {
        store.setCalendarVisible(cal.id, true)
      }
      store.setCalendarVisible('holidays', true)
      store.setCalendarVisible('deadlines', true)
    }
  }, [calendarData, visibleCalendarIds.size])

  const visibleEvents: CalendarEvent[] = useMemo(() => {
    const apiEvents = eventData?.events?.map(expandedEventToUI) || []
    const holidays = holidayData?.holidays?.map(holidayToUI) || []
    const deadlines = deadlineData?.deadlines?.map(deadlineToUI) || []

    const allEvents = [...apiEvents, ...holidays, ...deadlines]
    return allEvents.filter((e) =>
      calendars.find((c) => c.id === e.calendarId)?.visible,
    )
  }, [eventData, holidayData, deadlineData, calendars])

  const getEventsForDate = (d: Date) =>
    visibleEvents.filter((e) => e.date === formatDateKey(d))

  const navigate = (dir: -1 | 1) => {
    const d = new Date(currentDate)
    if (view === 'month') d.setMonth(d.getMonth() + dir)
    else if (view === 'week') d.setDate(d.getDate() + dir * 7)
    else d.setDate(d.getDate() + dir)
    setCurrentDate(d)
  }

  const goToToday = () => {
    const today = new Date()
    setCurrentDate(today)
    setSelectedDate(today)
  }

  const toggleCalendar = (id: string) => {
    toggleCalendarVisibility(id)
  }

  const handleSlotClick = (date: string, hour: number, minute: number, e: React.MouseEvent) => {
    setQuickCreate({ date, hour, minute, x: e.clientX, y: e.clientY })
  }

  const handleOpenFullForm = (defaults?: Partial<CalendarEvent>) => {
    setEventFormDefaults(defaults ?? {})
    setQuickCreate(null)
    setShowEventForm(true)
  }

  const handleDateClick = (d: Date) => {
    setSelectedDate(d)
    if (view === 'month') {
      setCurrentDate(d)
      setView('day')
    }
  }

  return (
    <div className="flex h-full overflow-hidden">
      {/* Left sidebar */}
      <CalendarSidebar
        selectedDate={selectedDate}
        events={visibleEvents}
        calendars={calendars}
        onToggleCalendar={toggleCalendar}
        onSelectEvent={setSelectedEvent}
      />

      {/* Main area */}
      <div className="flex-1 flex flex-col overflow-hidden">
        <CalendarToolbar
          view={view}
          currentDate={currentDate}
          onViewChange={setView}
          onNavigate={navigate}
          onToday={goToToday}
          onNewEvent={() => handleOpenFullForm()}
          onOpenRooms={() => setShowRoomBooking(true)}
          onOpenCategories={() => setShowCategoryManager(true)}
          onOpenCalendarBrowse={() => setShowCalendarBrowse(true)}
        />

        <div className="flex-1 overflow-auto bg-card">
          {eventsLoading && (
            <div className="flex h-full items-center justify-center">
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            </div>
          )}
          {!eventsLoading && view === 'week' && (
            <WeekView
              currentDate={currentDate}
              getEventsForDate={getEventsForDate}
              calendars={calendars}
              onSelectEvent={setSelectedEvent}
              onSlotClick={handleSlotClick}
              onDateClick={handleDateClick}
            />
          )}
          {!eventsLoading && view === 'day' && (
            <DayView
              currentDate={currentDate}
              events={getEventsForDate(currentDate)}
              calendars={calendars}
              onSelectEvent={setSelectedEvent}
              onSlotClick={handleSlotClick}
            />
          )}
          {!eventsLoading && view === 'month' && (
            <MonthView
              currentDate={currentDate}
              getEventsForDate={getEventsForDate}
              calendars={calendars}
              onSelectEvent={setSelectedEvent}
              onDateClick={handleDateClick}
            />
          )}
        </div>
      </div>

      {/* Quick-create popover */}
      {quickCreate && (
        <QuickCreatePopover
          state={quickCreate}
          onClose={() => setQuickCreate(null)}
          onMoreOptions={() =>
            handleOpenFullForm({
              date: quickCreate.date,
              startTime: `${String(quickCreate.hour).padStart(2, '0')}:${String(quickCreate.minute).padStart(2, '0')}`,
              endTime: `${String(quickCreate.hour + 1).padStart(2, '0')}:${String(quickCreate.minute).padStart(2, '0')}`,
            })
          }
        />
      )}

      {/* Full event form */}
      {showEventForm && (
        <EventFormModal
          defaults={eventFormDefaults}
          onClose={() => setShowEventForm(false)}
        />
      )}

      {/* Event detail panel */}
      {selectedEvent && (
        <EventDetailPanel
          event={selectedEvent}
          calendars={calendars}
          onClose={() => setSelectedEvent(null)}
          onEdit={() => {
            setEventFormDefaults(selectedEvent)
            setSelectedEvent(null)
            setShowEventForm(true)
          }}
        />
      )}

      {/* Room booking view */}
      {showRoomBooking && (
        <RoomBookingView onClose={() => setShowRoomBooking(false)} />
      )}

      {/* Category manager */}
      <CategoryManagerDialog
        open={showCategoryManager}
        onOpenChange={setShowCategoryManager}
        categories={categories}
        onCategoriesChange={() => {/* categories managed via API */}}
      />

      {/* Calendar browse */}
      <CalendarBrowseDialog
        open={showCalendarBrowse}
        onOpenChange={setShowCalendarBrowse}
        calendars={calendars}
        onToggleCalendar={toggleCalendar}
      />
    </div>
  )
}

// ============================================================
// Calendar Sidebar
// ============================================================

function CalendarSidebar({
  selectedDate,
  events,
  calendars,
  onToggleCalendar,
  onSelectEvent,
}: {
  selectedDate: Date
  events: CalendarEvent[]
  calendars: CalendarSource[]
  onToggleCalendar: (id: string) => void
  onSelectEvent: (e: CalendarEvent) => void
}) {
  const dayEvents = events
    .filter((e) => e.date === formatDateKey(selectedDate) && !e.isAllDay)
    .sort((a, b) => timeToMinutes(a.startTime) - timeToMinutes(b.startTime))

  const allDayEvents = events.filter(
    (e) => e.date === formatDateKey(selectedDate) && e.isAllDay,
  )

  const groups = [
    { label: 'Meine Kalender', items: calendars.filter((c) => c.group === 'mine') },
    { label: 'Geteilte Kalender', items: calendars.filter((c) => c.group === 'shared') },
    { label: 'Andere', items: calendars.filter((c) => c.group === 'other') },
  ]

  const dayLabel = `${DAYS_SHORT[(selectedDate.getDay() + 6) % 7]}, ${selectedDate.getDate()}. ${MONTHS_DE[selectedDate.getMonth()]}`

  return (
    <aside className="hidden lg:flex w-72 shrink-0 flex-col border-r border-border bg-card overflow-y-auto">
      {/* Day agenda */}
      <div className="p-4 border-b border-border">
        <h3 className="text-sm font-medium text-foreground mb-1">Tages-Agenda</h3>
        <p className="text-xs text-muted-foreground mb-3">{dayLabel}</p>

        {allDayEvents.length > 0 && (
          <div className="mb-2">
            {allDayEvents.map((e) => (
              <button
                key={e.id}
                onClick={() => onSelectEvent(e)}
                className="w-full flex items-center gap-2 rounded-md px-2 py-1.5 text-xs hover:bg-secondary transition-colors"
              >
                <span className="h-2 w-2 shrink-0 rounded-sm" style={{ backgroundColor: getCategoryColor(e, calendars) }} />
                <span className="truncate text-foreground">Ganztaegig: {e.title}</span>
              </button>
            ))}
          </div>
        )}

        {dayEvents.length > 0 ? (
          <div className="space-y-0.5">
            {dayEvents.map((e) => (
              <button
                key={e.id}
                onClick={() => onSelectEvent(e)}
                className="w-full flex items-center gap-2 rounded-md px-2 py-1.5 text-left hover:bg-secondary transition-colors"
              >
                <span
                  className="h-2 w-2 shrink-0 rounded-sm"
                  style={{ backgroundColor: getCategoryColor(e, calendars) }}
                />
                <div className="min-w-0 flex-1">
                  <p className="text-xs font-medium text-foreground truncate">{e.title}</p>
                  <p className="text-[10px] text-muted-foreground">
                    {e.startTime} – {e.endTime}
                    {e.location && ` · ${e.location}`}
                  </p>
                </div>
              </button>
            ))}
          </div>
        ) : (
          <p className="text-xs text-muted-foreground italic">Keine Termine</p>
        )}
      </div>

      {/* Calendar list */}
      <div className="p-4 space-y-4">
        {groups.map((group) => (
          <div key={group.label}>
            <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2">
              {group.label}
            </h4>
            <div className="space-y-1">
              {group.items.map((cal) => (
                <label
                  key={cal.id}
                  className="flex items-center gap-2 rounded-md px-2 py-1 cursor-pointer hover:bg-secondary transition-colors"
                >
                  <span
                    className="flex h-4 w-4 shrink-0 items-center justify-center rounded border-2 transition-colors"
                    style={{
                      borderColor: cal.color,
                      backgroundColor: cal.visible ? cal.color : 'transparent',
                    }}
                  >
                    {cal.visible && <Check className="h-2.5 w-2.5 text-white" />}
                  </span>
                  <input
                    type="checkbox"
                    checked={cal.visible}
                    onChange={() => onToggleCalendar(cal.id)}
                    className="sr-only"
                  />
                  <span className="text-xs text-foreground truncate">{cal.name}</span>
                </label>
              ))}
            </div>
          </div>
        ))}
      </div>
    </aside>
  )
}

// ============================================================
// Toolbar
// ============================================================

function CalendarToolbar({
  view,
  currentDate,
  onViewChange,
  onNavigate,
  onToday,
  onNewEvent,
  onOpenRooms,
  onOpenCategories,
  onOpenCalendarBrowse,
}: {
  view: ViewMode
  currentDate: Date
  onViewChange: (v: ViewMode) => void
  onNavigate: (dir: -1 | 1) => void
  onToday: () => void
  onNewEvent: () => void
  onOpenRooms: () => void
  onOpenCategories: () => void
  onOpenCalendarBrowse: () => void
}) {
  const label =
    view === 'day'
      ? `${currentDate.getDate()}. ${MONTHS_DE[currentDate.getMonth()]} ${currentDate.getFullYear()}`
      : `${MONTHS_DE[currentDate.getMonth()]} ${currentDate.getFullYear()}`

  return (
    <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 border-b border-border bg-card px-4 py-2.5">
      <div className="flex items-center gap-2">
        <button onClick={() => onNavigate(-1)} className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary transition-colors">
          <ChevronLeft className="h-4 w-4" />
        </button>
        <button onClick={() => onNavigate(1)} className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary transition-colors">
          <ChevronRight className="h-4 w-4" />
        </button>
        <h2 className="text-sm font-medium text-foreground min-w-[140px]">{label}</h2>
        <button
          onClick={onToday}
          className="rounded-md border border-primary bg-primary-subtle px-3 py-1 text-xs font-medium text-primary hover:bg-primary-light transition-colors"
        >
          Heute
        </button>
      </div>
      <div className="flex items-center gap-2">
        <div className="flex rounded-lg border border-border bg-secondary/50 p-0.5">
          {(['day', 'week', 'month'] as ViewMode[]).map((v) => (
            <button
              key={v}
              onClick={() => onViewChange(v)}
              className={`rounded-md px-3 py-1 text-xs transition-colors ${
                view === v ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:text-foreground'
              }`}
            >
              {v === 'day' ? 'Tag' : v === 'week' ? 'Woche' : 'Monat'}
            </button>
          ))}
        </div>
        <span className="mx-0.5 h-5 w-px bg-border" />
        <button
          onClick={onOpenRooms}
          className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
          title="Raumplanung"
        >
          <DoorOpen className="h-4 w-4" />
        </button>
        <button
          onClick={onOpenCalendarBrowse}
          className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
          title="Kalender verwalten"
        >
          <Layers className="h-4 w-4" />
        </button>
        <button
          onClick={onOpenCategories}
          className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
          title="Kategorien"
        >
          <Settings2 className="h-4 w-4" />
        </button>
        <span className="mx-0.5 h-5 w-px bg-border" />
        <button
          onClick={onNewEvent}
          className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors"
        >
          <Plus className="h-3.5 w-3.5" />
          Neues Event
        </button>
      </div>
    </div>
  )
}

// ============================================================
// Week View
// ============================================================

function WeekView({
  currentDate,
  getEventsForDate,
  calendars,
  onSelectEvent,
  onSlotClick,
  onDateClick,
}: {
  currentDate: Date
  getEventsForDate: (d: Date) => CalendarEvent[]
  calendars: CalendarSource[]
  onSelectEvent: (e: CalendarEvent) => void
  onSlotClick: (date: string, hour: number, minute: number, e: React.MouseEvent) => void
  onDateClick: (d: Date) => void
}) {
  const weekDays = getWeekDays(currentDate, true) // Mo-Fr

  return (
    <div className="flex flex-col min-h-full">
      {/* Day headers */}
      <div className="grid grid-cols-[56px_repeat(5,1fr)] border-b border-border sticky top-0 bg-card z-10">
        <div />
        {weekDays.map((d, i) => {
          const today = isToday(d)
          return (
            <button
              key={i}
              onClick={() => onDateClick(d)}
              className="px-2 py-2 text-center border-l border-border-muted hover:bg-secondary/50 transition-colors"
            >
              <p className="text-[10px] uppercase tracking-wider text-muted-foreground">{DAYS_SHORT[i]}</p>
              <p
                className={`text-sm mt-0.5 inline-flex h-7 w-7 items-center justify-center rounded-full ${
                  today ? 'bg-primary text-primary-foreground font-medium' : 'text-foreground'
                }`}
              >
                {d.getDate()}
              </p>
            </button>
          )
        })}
      </div>

      {/* All-day event row */}
      {weekDays.some((d) => getEventsForDate(d).some((e) => e.isAllDay)) && (
        <div className="grid grid-cols-[56px_repeat(5,1fr)] border-b border-border bg-secondary/30">
          <div className="px-1 py-1 text-right">
            <span className="text-[9px] text-muted-foreground">Ganztag</span>
          </div>
          {weekDays.map((d, i) => {
            const allDay = getEventsForDate(d).filter((e) => e.isAllDay)
            return (
              <div key={i} className="border-l border-border-muted px-0.5 py-0.5">
                {allDay.map((e) => (
                  <button
                    key={e.id}
                    onClick={() => onSelectEvent(e)}
                    className="w-full rounded px-1.5 py-0.5 text-[10px] truncate text-left transition-colors hover:brightness-90"
                    style={{
                      backgroundColor: e.isHoliday ? 'var(--secondary)' : `${getCategoryColor(e, calendars)}20`,
                      color: e.isHoliday ? 'var(--muted-foreground)' : getCategoryColor(e, calendars),
                    }}
                  >
                    {e.title}
                  </button>
                ))}
              </div>
            )
          })}
        </div>
      )}

      {/* Time grid */}
      <div className="grid grid-cols-[56px_repeat(5,1fr)] flex-1">
        {/* Time labels */}
        <div>
          {HOURS.map((hour) => (
            <div key={hour} className="h-[60px] pr-2 pt-0 text-right">
              <span className="text-[10px] text-muted-foreground leading-none relative -top-1.5">{hour}:00</span>
            </div>
          ))}
        </div>

        {/* Day columns */}
        {weekDays.map((d, dayIdx) => {
          const dayEvents = getEventsForDate(d).filter((e) => !e.isAllDay)
          const layouts = layoutOverlappingEvents(dayEvents)

          return (
            <div
              key={dayIdx}
              className="border-l border-border-muted relative"
              style={{ height: HOURS.length * HOUR_HEIGHT }}
            >
              {/* Hour grid lines */}
              {HOURS.map((hour) => (
                <div
                  key={hour}
                  className="absolute w-full border-t border-border-muted cursor-pointer"
                  style={{ top: (hour - START_HOUR) * HOUR_HEIGHT, height: HOUR_HEIGHT }}
                  onClick={(e) => onSlotClick(formatDateKey(d), hour, 0, e)}
                />
              ))}
              {/* Half-hour lines */}
              {HOURS.map((hour) => (
                <div
                  key={`half-${hour}`}
                  className="absolute w-full border-t border-border-muted/40"
                  style={{ top: (hour - START_HOUR) * HOUR_HEIGHT + HOUR_HEIGHT / 2 }}
                />
              ))}

              {/* Current time indicator */}
              {isToday(d) && (
                <div
                  className="absolute left-0 right-0 z-10 flex items-center"
                  style={{ top: ((10 * 60 + 30 - START_HOUR * 60) / 60) * HOUR_HEIGHT }}
                >
                  <div className="h-2.5 w-2.5 -ml-1 rounded-full bg-error" />
                  <div className="flex-1 h-[2px] bg-error" />
                </div>
              )}

              {/* Events */}
              {layouts.map(({ event, column, totalColumns }) => {
                const startMin = timeToMinutes(event.startTime)
                const endMin = timeToMinutes(event.endTime)
                const top = ((startMin - START_HOUR * 60) / 60) * HOUR_HEIGHT
                const height = Math.max(((endMin - startMin) / 60) * HOUR_HEIGHT, 22)
                const color = getCategoryColor(event, calendars)
                const leftPct = (column / totalColumns) * 100
                const widthPct = (1 / totalColumns) * 100 - 1

                return (
                  <button
                    key={event.id}
                    onClick={(e) => {
                      e.stopPropagation()
                      onSelectEvent(event)
                    }}
                    className="absolute rounded-[4px] px-1.5 py-0.5 text-left overflow-hidden transition-all hover:brightness-95 hover:shadow-sm z-[5]"
                    style={{
                      top,
                      height,
                      left: `calc(${leftPct}% + 2px)`,
                      width: `calc(${widthPct}% - 2px)`,
                      backgroundColor: event.isTaskDeadline ? 'transparent' : `${color}18`,
                      borderLeft: event.isTaskDeadline ? 'none' : `3px solid ${color}`,
                      border: event.isTaskDeadline ? `1.5px dashed ${color}` : undefined,
                    }}
                  >
                    <p className="text-[10px] font-medium truncate" style={{ color }}>
                      {event.title}
                    </p>
                    {height > 30 && (
                      <p className="text-[9px] truncate" style={{ color, opacity: 0.7 }}>
                        {event.startTime} – {event.endTime}
                      </p>
                    )}
                    {height > 44 && event.videoCall && (
                      <Video className="h-2.5 w-2.5 mt-0.5" style={{ color, opacity: 0.6 }} />
                    )}
                  </button>
                )
              })}
            </div>
          )
        })}
      </div>
    </div>
  )
}

// ============================================================
// Day View
// ============================================================

function DayView({
  currentDate,
  events,
  calendars,
  onSelectEvent,
  onSlotClick,
}: {
  currentDate: Date
  events: CalendarEvent[]
  calendars: CalendarSource[]
  onSelectEvent: (e: CalendarEvent) => void
  onSlotClick: (date: string, hour: number, minute: number, e: React.MouseEvent) => void
}) {
  const allDay = events.filter((e) => e.isAllDay)
  const timed = events.filter((e) => !e.isAllDay)
  const layouts = layoutOverlappingEvents(timed)
  const dateKey = formatDateKey(currentDate)

  return (
    <div className="flex flex-col min-h-full">
      {/* All-day events */}
      {allDay.length > 0 && (
        <div className="border-b border-border px-4 py-2">
          <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1">Ganztaegig</p>
          <div className="flex flex-wrap gap-1">
            {allDay.map((e) => (
              <button
                key={e.id}
                onClick={() => onSelectEvent(e)}
                className="rounded-md px-2.5 py-1 text-xs transition-colors hover:brightness-90"
                style={{
                  backgroundColor: e.isHoliday ? 'var(--secondary)' : `${getCategoryColor(e, calendars)}20`,
                  color: e.isHoliday ? 'var(--muted-foreground)' : getCategoryColor(e, calendars),
                }}
              >
                {e.title}
              </button>
            ))}
          </div>
        </div>
      )}

      {/* Time grid */}
      <div className="flex flex-1">
        {/* Time labels */}
        <div className="w-14 shrink-0">
          {HOURS.map((hour) => (
            <div key={hour} className="h-[60px] pr-2 text-right">
              <span className="text-[10px] text-muted-foreground relative -top-1.5">{hour}:00</span>
            </div>
          ))}
        </div>

        {/* Event column */}
        <div className="flex-1 border-l border-border-muted relative" style={{ height: HOURS.length * HOUR_HEIGHT }}>
          {HOURS.map((hour) => (
            <div
              key={hour}
              className="absolute w-full border-t border-border-muted cursor-pointer"
              style={{ top: (hour - START_HOUR) * HOUR_HEIGHT, height: HOUR_HEIGHT }}
              onClick={(e) => onSlotClick(dateKey, hour, 0, e)}
            />
          ))}
          {HOURS.map((hour) => (
            <div
              key={`half-${hour}`}
              className="absolute w-full border-t border-border-muted/40"
              style={{ top: (hour - START_HOUR) * HOUR_HEIGHT + HOUR_HEIGHT / 2 }}
            />
          ))}

          {layouts.map(({ event, column, totalColumns }) => {
            const startMin = timeToMinutes(event.startTime)
            const endMin = timeToMinutes(event.endTime)
            const top = ((startMin - START_HOUR * 60) / 60) * HOUR_HEIGHT
            const height = Math.max(((endMin - startMin) / 60) * HOUR_HEIGHT, 24)
            const color = getCategoryColor(event, calendars)
            const leftPct = (column / totalColumns) * 100
            const widthPct = (1 / totalColumns) * 100 - 2

            return (
              <button
                key={event.id}
                onClick={(e) => {
                  e.stopPropagation()
                  onSelectEvent(event)
                }}
                className="absolute rounded-md px-3 py-1.5 text-left overflow-hidden transition-all hover:brightness-95 hover:shadow-sm z-[5]"
                style={{
                  top,
                  height,
                  left: `calc(${leftPct}% + 4px)`,
                  width: `calc(${widthPct}% - 4px)`,
                  backgroundColor: event.isTaskDeadline ? 'transparent' : `${color}18`,
                  borderLeft: event.isTaskDeadline ? 'none' : `4px solid ${color}`,
                  border: event.isTaskDeadline ? `2px dashed ${color}` : undefined,
                }}
              >
                <p className="text-xs font-medium truncate" style={{ color }}>{event.title}</p>
                <p className="text-[10px] mt-0.5" style={{ color, opacity: 0.7 }}>
                  {event.startTime} – {event.endTime}
                  {event.location && ` · ${event.location}`}
                </p>
                {height > 50 && event.participants && (
                  <div className="flex gap-0.5 mt-1">
                    {event.participants.slice(0, 3).map((p) => (
                      <span
                        key={p.initials}
                        className="inline-flex h-4 w-4 items-center justify-center rounded-full text-[7px] font-medium text-white"
                        style={{ backgroundColor: color }}
                      >
                        {p.initials}
                      </span>
                    ))}
                    {event.participants.length > 3 && (
                      <span className="text-[9px]" style={{ color, opacity: 0.6 }}>
                        +{event.participants.length - 3}
                      </span>
                    )}
                  </div>
                )}
              </button>
            )
          })}
        </div>
      </div>
    </div>
  )
}

// ============================================================
// Month View
// ============================================================

function MonthView({
  currentDate,
  getEventsForDate,
  calendars,
  onSelectEvent,
  onDateClick,
}: {
  currentDate: Date
  getEventsForDate: (d: Date) => CalendarEvent[]
  calendars: CalendarSource[]
  onSelectEvent: (e: CalendarEvent) => void
  onDateClick: (d: Date) => void
}) {
  const days = getMonthDays(currentDate.getFullYear(), currentDate.getMonth())

  return (
    <div className="h-full flex flex-col">
      <div className="grid grid-cols-7 border-b border-border">
        {DAYS_SHORT.map((day) => (
          <div key={day} className="px-2 py-2 text-center text-[10px] uppercase tracking-wider font-medium text-muted-foreground">
            {day}
          </div>
        ))}
      </div>
      <div className="grid grid-cols-7 flex-1">
        {days.map((d, i) => {
          const events = getEventsForDate(d.date)
          const today = isToday(d.date)
          return (
            <div
              key={i}
              className={`min-h-[100px] border-b border-r border-border-muted p-1 cursor-pointer hover:bg-secondary/30 transition-colors ${
                !d.isCurrentMonth ? 'bg-secondary/20' : ''
              }`}
              onClick={() => onDateClick(d.date)}
            >
              <div className="flex justify-end">
                <span
                  className={`flex h-6 w-6 items-center justify-center rounded-full text-xs ${
                    today
                      ? 'bg-primary text-primary-foreground font-medium'
                      : d.isCurrentMonth
                        ? 'text-foreground'
                        : 'text-text-disabled'
                  }`}
                >
                  {d.date.getDate()}
                </span>
              </div>
              <div className="space-y-0.5 mt-0.5">
                {events.slice(0, 3).map((event) => {
                  const color = getCategoryColor(event, calendars)
                  return (
                    <button
                      key={event.id}
                      onClick={(e) => {
                        e.stopPropagation()
                        onSelectEvent(event)
                      }}
                      className="flex w-full items-center gap-1 rounded px-1 py-0.5 text-[10px] hover:bg-secondary/80 transition-colors truncate text-left"
                    >
                      <span
                        className="h-1.5 w-1.5 shrink-0 rounded-full"
                        style={{ backgroundColor: color }}
                      />
                      <span className="truncate text-foreground">
                        {!event.isAllDay && `${event.startTime} `}
                        {event.title}
                      </span>
                    </button>
                  )
                })}
                {events.length > 3 && (
                  <p className="text-[10px] text-primary font-medium px-1 cursor-pointer hover:underline">
                    +{events.length - 3} weitere
                  </p>
                )}
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}

// ============================================================
// Quick-Create Popover
// ============================================================

function QuickCreatePopover({
  state,
  onClose,
  onMoreOptions,
}: {
  state: QuickCreateState
  onClose: () => void
  onMoreOptions: () => void
}) {
  const [title, setTitle] = useState('')
  const [categoryId, setCategoryId] = useState('meeting')
  const timeLabel = `${String(state.hour).padStart(2, '0')}:${String(state.minute).padStart(2, '0')} – ${String(state.hour + 1).padStart(2, '0')}:${String(state.minute).padStart(2, '0')}`

  // Position near click, clamped to viewport
  const top = Math.min(state.y, window.innerHeight - 280)
  const left = Math.min(state.x, window.innerWidth - 320)

  return (
    <>
      <div className="fixed inset-0 z-40" onClick={onClose} />
      <div
        className="fixed z-50 w-72 rounded-xl border border-border bg-card shadow-[var(--shadow-large)] overflow-hidden"
        style={{ top, left }}
      >
        <div className="p-3 space-y-3">
          <input
            autoFocus
            type="text"
            placeholder="Titel hinzufuegen..."
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            className="w-full bg-transparent text-sm text-foreground placeholder:text-muted-foreground outline-none border-b border-border-muted pb-2"
          />

          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <Clock className="h-3.5 w-3.5" />
            <span>{timeLabel}</span>
          </div>

          <div className="flex items-center gap-2">
            <span className="text-[10px] text-muted-foreground">Kategorie:</span>
            <div className="flex gap-1">
              {DEFAULT_CATEGORIES.map((cat) => (
                <button
                  key={cat.id}
                  onClick={() => setCategoryId(cat.id)}
                  className="h-5 w-5 rounded-full border-2 transition-all"
                  style={{
                    backgroundColor: cat.color,
                    borderColor: categoryId === cat.id ? 'var(--foreground)' : 'transparent',
                    transform: categoryId === cat.id ? 'scale(1.15)' : 'scale(1)',
                  }}
                  title={cat.name}
                />
              ))}
            </div>
          </div>

          <div className="flex items-center gap-2 pt-1">
            <button
              onClick={onClose}
              className="flex-1 rounded-lg bg-primary px-3 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors"
            >
              Speichern
            </button>
            <button
              onClick={onMoreOptions}
              className="flex-1 rounded-lg border border-border px-3 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors"
            >
              Mehr Optionen
            </button>
          </div>
        </div>
      </div>
    </>
  )
}

// ============================================================
// Event Form Modal
// ============================================================

function EventFormModal({
  defaults,
  onClose,
}: {
  defaults: Partial<CalendarEvent>
  onClose: () => void
}) {
  const [title, setTitle] = useState(defaults.title ?? '')
  const [date, setDate] = useState(defaults.date ?? '2026-02-09')
  const [startTime, setStartTime] = useState(defaults.startTime ?? '09:00')
  const [endTime, setEndTime] = useState(defaults.endTime ?? '10:00')
  const [isAllDay, setIsAllDay] = useState(defaults.isAllDay ?? false)
  const [categoryId, setCategoryId] = useState(defaults.categoryId ?? 'meeting')
  const [location, setLocation] = useState(defaults.location ?? '')
  const [room, setRoom] = useState(defaults.room ?? '')
  const [description, setDescription] = useState(defaults.description ?? '')
  const [recurrence, setRecurrence] = useState(defaults.recurrence ?? 'Keine')
  const [reminder, setReminder] = useState(defaults.reminder ?? '15 Minuten')
  const [calendarId, setCalendarId] = useState(defaults.calendarId ?? 'personal')
  const [videoCall, setVideoCall] = useState(defaults.videoCall ?? false)
  const [participantSearch, setParticipantSearch] = useState('')

  const filteredMembers = participantSearch
    ? TEAM_MEMBERS.filter((m) => m.name.toLowerCase().includes(participantSearch.toLowerCase()))
    : []

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center" onClick={(e) => { if (e.target === e.currentTarget) onClose() }}>
      <div className="absolute inset-0 bg-black/40" />
      <div className="relative w-full max-w-lg max-h-[85vh] rounded-xl border border-border bg-card shadow-[var(--shadow-large)] overflow-hidden flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-border px-5 py-3">
          <h3 className="text-sm font-medium text-foreground">
            {defaults.title ? 'Event bearbeiten' : 'Neues Event'}
          </h3>
          <button onClick={onClose} className="rounded-md p-1 text-muted-foreground hover:bg-secondary">
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Form */}
        <div className="overflow-y-auto flex-1 p-5 space-y-4">
          {/* Title */}
          <div>
            <input
              autoFocus
              type="text"
              placeholder="Titel"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              className="w-full rounded-lg border border-input-border bg-input-background px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder outline-none focus:border-primary"
            />
          </div>

          {/* Date & Time */}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1 block">Datum</label>
              <input
                type="date"
                value={date}
                onChange={(e) => setDate(e.target.value)}
                className="w-full rounded-lg border border-input-border bg-input-background px-3 py-1.5 text-xs text-foreground outline-none focus:border-primary"
              />
            </div>
            {!isAllDay && (
              <>
                <div className="grid grid-cols-2 gap-2">
                  <div>
                    <label className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1 block">Von</label>
                    <input
                      type="time"
                      value={startTime}
                      onChange={(e) => setStartTime(e.target.value)}
                      className="w-full rounded-lg border border-input-border bg-input-background px-2 py-1.5 text-xs text-foreground outline-none focus:border-primary"
                    />
                  </div>
                  <div>
                    <label className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1 block">Bis</label>
                    <input
                      type="time"
                      value={endTime}
                      onChange={(e) => setEndTime(e.target.value)}
                      className="w-full rounded-lg border border-input-border bg-input-background px-2 py-1.5 text-xs text-foreground outline-none focus:border-primary"
                    />
                  </div>
                </div>
              </>
            )}
          </div>

          {/* All-day toggle */}
          <label className="flex items-center gap-2 cursor-pointer">
            <div
              className="relative h-5 w-9 rounded-full transition-colors"
              style={{ backgroundColor: isAllDay ? 'var(--primary)' : 'var(--switch-background)' }}
              onClick={() => setIsAllDay(!isAllDay)}
            >
              <div
                className="absolute top-0.5 h-4 w-4 rounded-full bg-white shadow-sm transition-transform"
                style={{ left: isAllDay ? 18 : 2 }}
              />
            </div>
            <span className="text-xs text-foreground">Ganztaegig</span>
          </label>

          {/* Category */}
          <div>
            <label className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1.5 block">Kategorie</label>
            <div className="flex flex-wrap gap-1.5">
              {DEFAULT_CATEGORIES.map((cat) => (
                <button
                  key={cat.id}
                  onClick={() => setCategoryId(cat.id)}
                  className="flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs transition-all"
                  style={{
                    backgroundColor: `${cat.color}20`,
                    color: cat.color,
                    outlineColor: categoryId === cat.id ? cat.color : undefined,
                    outlineWidth: categoryId === cat.id ? 2 : 0,
                    outlineStyle: 'solid',
                    outlineOffset: 2,
                  }}
                >
                  <span className="h-2 w-2 rounded-full" style={{ backgroundColor: cat.color }} />
                  {cat.name}
                </button>
              ))}
            </div>
          </div>

          {/* Location */}
          <div>
            <label className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1 block">Ort</label>
            <div className="flex items-center gap-2 rounded-lg border border-input-border bg-input-background px-3 py-1.5">
              <MapPin className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
              <input
                type="text"
                placeholder="z.B. Buero Zuerich, externe Adresse..."
                value={location}
                onChange={(e) => setLocation(e.target.value)}
                className="flex-1 bg-transparent text-xs text-foreground placeholder:text-input-placeholder outline-none"
              />
            </div>
          </div>

          {/* Room */}
          <div>
            <label className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1 block">Raum</label>
            <select
              value={room}
              onChange={(e) => setRoom(e.target.value)}
              className="w-full rounded-lg border border-input-border bg-input-background px-3 py-1.5 text-xs text-foreground outline-none focus:border-primary"
            >
              <option value="">Kein Raum</option>
              {ROOMS.map((r) => (
                <option key={r.id} value={r.name}>
                  {r.name} ({r.capacity} Pl.)
                </option>
              ))}
            </select>
          </div>

          {/* Description */}
          <div>
            <label className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1 block">Beschreibung</label>
            <textarea
              rows={3}
              placeholder="Beschreibung..."
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              className="w-full rounded-lg border border-input-border bg-input-background px-3 py-2 text-xs text-foreground placeholder:text-input-placeholder outline-none resize-none focus:border-primary"
            />
          </div>

          {/* Recurrence & Reminder */}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1 block">
                <Repeat className="h-3 w-3 inline mr-1" />
                Wiederholung
              </label>
              <select
                value={recurrence}
                onChange={(e) => setRecurrence(e.target.value)}
                className="w-full rounded-lg border border-input-border bg-input-background px-3 py-1.5 text-xs text-foreground outline-none focus:border-primary"
              >
                {RECURRENCE_OPTIONS.map((opt) => (
                  <option key={opt} value={opt}>{opt}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1 block">
                <Bell className="h-3 w-3 inline mr-1" />
                Erinnerung
              </label>
              <select
                value={reminder}
                onChange={(e) => setReminder(e.target.value)}
                className="w-full rounded-lg border border-input-border bg-input-background px-3 py-1.5 text-xs text-foreground outline-none focus:border-primary"
              >
                {REMINDER_OPTIONS.map((opt) => (
                  <option key={opt} value={opt}>{opt}</option>
                ))}
              </select>
            </div>
          </div>

          {/* Calendar */}
          <div>
            <label className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1 block">
              <Calendar className="h-3 w-3 inline mr-1" />
              Kalender
            </label>
            <select
              value={calendarId}
              onChange={(e) => setCalendarId(e.target.value)}
              className="w-full rounded-lg border border-input-border bg-input-background px-3 py-1.5 text-xs text-foreground outline-none focus:border-primary"
            >
              {INITIAL_CALENDARS.filter((c) => c.group !== 'other').map((c) => (
                <option key={c.id} value={c.id}>{c.name}</option>
              ))}
            </select>
          </div>

          {/* Participants */}
          <div>
            <label className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1 block">
              <Users className="h-3 w-3 inline mr-1" />
              Teilnehmer einladen
            </label>
            <div className="relative">
              <div className="flex items-center gap-2 rounded-lg border border-input-border bg-input-background px-3 py-1.5">
                <Search className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                <input
                  type="text"
                  placeholder="Name eingeben..."
                  value={participantSearch}
                  onChange={(e) => setParticipantSearch(e.target.value)}
                  className="flex-1 bg-transparent text-xs text-foreground placeholder:text-input-placeholder outline-none"
                />
              </div>
              {filteredMembers.length > 0 && (
                <div className="absolute left-0 right-0 top-full mt-1 rounded-lg border border-border bg-card shadow-[var(--shadow-medium)] z-10 overflow-hidden">
                  {filteredMembers.map((m) => (
                    <button
                      key={m.initials}
                      onClick={() => setParticipantSearch('')}
                      className="w-full flex items-center gap-2 px-3 py-2 text-left hover:bg-secondary transition-colors"
                    >
                      <span className="flex h-6 w-6 items-center justify-center rounded-full bg-primary text-[9px] font-medium text-primary-foreground">
                        {m.initials}
                      </span>
                      <div>
                        <p className="text-xs text-foreground">{m.name}</p>
                        <p className="text-[10px] text-muted-foreground">{m.role}</p>
                      </div>
                    </button>
                  ))}
                </div>
              )}
            </div>
          </div>

          {/* Video call toggle */}
          <label className="flex items-center gap-2 cursor-pointer">
            <div
              className="relative h-5 w-9 rounded-full transition-colors"
              style={{ backgroundColor: videoCall ? 'var(--primary)' : 'var(--switch-background)' }}
              onClick={() => setVideoCall(!videoCall)}
            >
              <div
                className="absolute top-0.5 h-4 w-4 rounded-full bg-white shadow-sm transition-transform"
                style={{ left: videoCall ? 18 : 2 }}
              />
            </div>
            <Video className="h-3.5 w-3.5 text-muted-foreground" />
            <span className="text-xs text-foreground">Video-Call erstellen</span>
          </label>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-2 border-t border-border px-5 py-3">
          <button
            onClick={onClose}
            className="rounded-lg border border-border px-4 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors"
          >
            Abbrechen
          </button>
          <button
            onClick={onClose}
            className="rounded-lg bg-primary px-4 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors"
          >
            Speichern
          </button>
        </div>
      </div>
    </div>
  )
}

// ============================================================
// Event Detail Panel
// ============================================================

function EventDetailPanel({
  event,
  calendars,
  onClose,
  onEdit,
}: {
  event: CalendarEvent
  calendars: CalendarSource[]
  onClose: () => void
  onEdit: () => void
}) {
  const color = getCategoryColor(event, calendars)
  const category = DEFAULT_CATEGORIES.find((c) => c.id === event.categoryId)
  const calendar = calendars.find((c) => c.id === event.calendarId)

  const rsvpIcon = (status: RSVPStatus) => {
    switch (status) {
      case 'accepted': return <Check className="h-3 w-3 text-success" />
      case 'declined': return <CircleX className="h-3 w-3 text-error" />
      case 'maybe': return <CircleHelp className="h-3 w-3 text-warning" />
      default: return <Clock className="h-3 w-3 text-muted-foreground" />
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center" onClick={(e) => { if (e.target === e.currentTarget) onClose() }}>
      <div className="absolute inset-0 bg-black/30" />
      <div className="relative w-full max-w-sm rounded-xl border border-border bg-card shadow-[var(--shadow-large)] overflow-hidden">
        {/* Color bar */}
        <div className="h-1.5" style={{ backgroundColor: color }} />

        <div className="p-4 space-y-3">
          {/* Header */}
          <div className="flex items-start justify-between gap-2">
            <div>
              <h3 className="text-sm font-medium text-foreground">{event.title}</h3>
              {event.recurrence && (
                <span className="inline-flex items-center gap-1 mt-0.5 text-[10px] text-muted-foreground">
                  <Repeat className="h-2.5 w-2.5" /> {event.recurrence}
                </span>
              )}
            </div>
            <div className="flex items-center gap-1">
              <button onClick={onEdit} className="rounded-md p-1 text-muted-foreground hover:bg-secondary text-xs">
                Bearbeiten
              </button>
              <button onClick={onClose} className="rounded-md p-1 text-muted-foreground hover:bg-secondary">
                <X className="h-4 w-4" />
              </button>
            </div>
          </div>

          {/* Details */}
          <div className="space-y-2.5 text-xs">
            {/* Date & time */}
            <div className="flex items-center gap-2 text-muted-foreground">
              <Clock className="h-3.5 w-3.5 shrink-0" />
              {event.isAllDay ? (
                <span>Ganztaegig</span>
              ) : (
                <span>{event.startTime} – {event.endTime}</span>
              )}
            </div>

            {/* Category & calendar */}
            <div className="flex items-center gap-2 text-muted-foreground">
              <span className="h-3 w-3 rounded-full shrink-0" style={{ backgroundColor: color }} />
              <span>{category?.name ?? 'Event'} · {calendar?.name}</span>
            </div>

            {/* Location */}
            {event.location && (
              <div className="flex items-center gap-2 text-muted-foreground">
                <MapPin className="h-3.5 w-3.5 shrink-0" />
                <span>{event.location}</span>
              </div>
            )}

            {/* Room */}
            {event.room && (
              <div className="flex items-center gap-2 text-muted-foreground">
                <Calendar className="h-3.5 w-3.5 shrink-0" />
                <span>{event.room}</span>
              </div>
            )}

            {/* Video call */}
            {event.videoCall && (
              <div className="flex items-center gap-2 text-muted-foreground">
                <Video className="h-3.5 w-3.5 shrink-0" />
                <button className="text-primary hover:underline">Video-Call beitreten</button>
              </div>
            )}

            {/* Description */}
            {event.description && (
              <p className="text-muted-foreground leading-relaxed pl-5">{event.description}</p>
            )}

            {/* Task deadline hint */}
            {event.isTaskDeadline && (
              <div className="flex items-center gap-2 rounded-md bg-error-light px-2 py-1.5 text-error">
                <Clock className="h-3.5 w-3.5" />
                <span className="text-[10px] font-medium">Task-Deadline</span>
              </div>
            )}

            {/* Participants & RSVP */}
            {event.participants && event.participants.length > 0 && (
              <div>
                <div className="flex items-center gap-2 text-muted-foreground mb-2">
                  <Users className="h-3.5 w-3.5 shrink-0" />
                  <span>{event.participants.length} Teilnehmer</span>
                </div>
                <div className="space-y-1 pl-5">
                  {event.participants.map((p) => (
                    <div key={p.initials} className="flex items-center gap-2">
                      <span
                        className="flex h-5 w-5 items-center justify-center rounded-full text-[8px] font-medium text-white"
                        style={{ backgroundColor: color }}
                      >
                        {p.initials}
                      </span>
                      <span className="text-foreground flex-1">{p.name}</span>
                      {rsvpIcon(p.rsvp)}
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

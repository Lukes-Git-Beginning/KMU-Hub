import { useState, useMemo, useEffect, useRef, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { useCalendars, useEventCategories, useCreateEventCategory, useDeleteEventCategory } from '@/api/hooks/useCalendars'
import { useEventsInRange, useCreateEvent, useUpdateEvent, useDeleteEvent, useTaskDeadlines } from '@/api/hooks/useEvents'
import { useHolidays } from '@/api/hooks/useHolidays'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'
import {
  expandedEventToUI, calendarToUI, categoryToUI, deadlineToUI, holidayToUI,
  uiEventToCreateRequest, uiEventToUpdateRequest,
  getHolidayCalendar, getDeadlineCalendar,
  type CalendarEvent as AdapterCalendarEvent,
  type CalendarSource as AdapterCalendarSource,
  type UIEventCategory,
} from './adapters'
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
  CalendarCheck,
  Scissors,
  Euro,
  User,
  FileText,
  Copy,
  ExternalLink,
  GripVertical,
  Mail,
  Phone,
} from 'lucide-react'
import { toast } from 'sonner'
import { moduleHsl } from '@/components/layout/sidebar/nav-items'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { RoomBookingView } from './RoomBookingView'
import { CategoryManagerDialog } from './CategoryManagerDialog'
import { CalendarBrowseDialog } from './CalendarBrowseDialog'
import { CalendarSettingsTab } from './tabs/CalendarSettingsTab'

// ============================================================
// Types (re-export from adapters)
// ============================================================

type TopTab = 'kalender' | 'terminbuchung'

// Re-use adapter types
type CalendarEvent = AdapterCalendarEvent
type CalendarSource = AdapterCalendarSource
type EventCategory = UIEventCategory
type ViewMode = 'week' | 'day' | 'month'
type RSVPStatus = 'accepted' | 'declined' | 'maybe' | 'pending'

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

interface DragState {
  eventId: string
  startY: number
  startHour: number
  startMinute: number
  currentY: number
  mode: 'move' | 'resize'
  dayDate: string
  originalStartTime: string
  originalEndTime: string
}

// ============================================================
// Constants
// ============================================================

// (Calendar and category constants removed — now fetched from API)

const ROOMS = [
  { id: 'r1', name: 'Raum Alpen — Besprechung', capacity: 10, tags: ['Beamer', 'Whiteboard'] },
  { id: 'r2', name: 'Raum Isar — Klein', capacity: 4, tags: ['Display'] },
  { id: 'r3', name: 'Fokusraum 1', capacity: 1, tags: [] },
  { id: 'r4', name: 'Fokusraum 2', capacity: 1, tags: [] },
]

const TEAM_MEMBERS = [
  { name: 'Stefan Vogel', initials: 'SV', role: 'Geschaeftsfuehrer' },
  { name: 'Markus Weber', initials: 'MW', role: 'CTO' },
  { name: 'Thomas Meier', initials: 'TM', role: 'Vertriebsleiter' },
  { name: 'Laura Neumann', initials: 'LN', role: 'Senior Developerin' },
  { name: 'Nina Richter', initials: 'NR', role: 'Creative Director' },
  { name: 'Sarah Beck', initials: 'SB', role: 'Projektleiterin' },
  { name: 'Felix Krause', initials: 'FK', role: 'Backend Developer' },
]

const DAYS_SHORT = ['Mo', 'Di', 'Mi', 'Do', 'Fr', 'Sa', 'So']
const MONTHS_DE = [
  'Januar', 'Februar', 'März', 'April', 'Mai', 'Juni',
  'Juli', 'August', 'September', 'Oktober', 'November', 'Dezember',
]
const RECURRENCE_OPTIONS = ['Keine', 'Täglich', 'Wöchentlich', 'Monatlich', 'Jaehrlich', 'Benutzerdefiniert...']
const REMINDER_OPTIONS = ['Keine', '5 Minuten', '10 Minuten', '15 Minuten', '30 Minuten', '1 Stunde', '2 Stunden', '1 Tag']

const HOUR_HEIGHT = 60
// Work-hour grid bounds are read per-view from the calendar settings store
// (useWorkHours); no longer module-level constants.

// (Mock events removed — now fetched from API)

// ============================================================
// Terminbuchung Types & Mock Data
// ============================================================

interface BookingService {
  id: string
  name: string
  dauer: number  // minutes
  preis: number  // CHF
  color: string
  personal: string[]
}

interface BookingAppointment {
  id: string
  serviceId: string
  kunde: string
  datum: string
  startTime: string
  endTime: string
  personal: string
  notizen?: string
  status: 'bestätigt' | 'ausstehend' | 'abgesagt'
}

const BOOKING_SERVICES: BookingService[] = [
  { id: 'bs1', name: 'Haarschnitt Herren', dauer: 30, preis: 45, color: '#3d7cc9', personal: ['Lena Huber', 'Marco Roth'] },
  { id: 'bs2', name: 'Haarschnitt Damen', dauer: 45, preis: 65, color: '#d4619b', personal: ['Lena Huber', 'Nina Frei'] },
  { id: 'bs3', name: 'Faerben komplett', dauer: 90, preis: 120, color: '#8b5fc7', personal: ['Nina Frei'] },
  { id: 'bs4', name: 'Bartpflege', dauer: 20, preis: 25, color: '#3da356', personal: ['Marco Roth'] },
  { id: 'bs5', name: 'Beratungsgespräch', dauer: 60, preis: 80, color: '#d48c3d', personal: ['Lena Huber', 'Marco Roth', 'Nina Frei'] },
  { id: 'bs6', name: 'Massage 30min', dauer: 30, preis: 55, color: '#1e7e74', personal: ['Sandra Wyss'] },
  { id: 'bs7', name: 'Massage 60min', dauer: 60, preis: 95, color: '#1e7e74', personal: ['Sandra Wyss'] },
  { id: 'bs8', name: 'Manikuere', dauer: 45, preis: 50, color: '#c75a8b', personal: ['Nina Frei', 'Sandra Wyss'] },
]

const BOOKING_STAFF = ['Lena Huber', 'Marco Roth', 'Nina Frei', 'Sandra Wyss']

const MOCK_EXTERNAL_SERVICES = [
  { id: 'ext1', name: 'Beratungsgespräch 30 Min', duration: 30, price: 0 },
  { id: 'ext2', name: 'Erstgespräch 60 Min', duration: 60, price: 0 },
  { id: 'ext3', name: 'Technischer Support 45 Min', duration: 45, price: 0 },
]

const EXTERNAL_TIME_SLOTS = [
  '09:00', '09:30', '10:00', '10:30', '11:00', '11:30',
  '13:00', '13:30', '14:00', '14:30', '15:00', '15:30',
  '16:00', '16:30', '17:00',
]

const MOCK_BOOKINGS: BookingAppointment[] = [
  // Today (2026-02-09 as mock "today")
  { id: 'bk1', serviceId: 'bs1', kunde: 'Anna Weber', datum: '2026-02-09', startTime: '09:00', endTime: '09:30', personal: 'Marco Roth', status: 'bestätigt' },
  { id: 'bk2', serviceId: 'bs2', kunde: 'Markus Steiner', datum: '2026-02-09', startTime: '09:30', endTime: '10:15', personal: 'Lena Huber', status: 'bestätigt' },
  { id: 'bk3', serviceId: 'bs6', kunde: 'Sarah Keller', datum: '2026-02-09', startTime: '10:00', endTime: '10:30', personal: 'Sandra Wyss', status: 'bestätigt' },
  { id: 'bk4', serviceId: 'bs3', kunde: 'Julia Meier', datum: '2026-02-09', startTime: '11:00', endTime: '12:30', personal: 'Nina Frei', status: 'ausstehend' },
  { id: 'bk5', serviceId: 'bs5', kunde: 'Thomas Brunner', datum: '2026-02-09', startTime: '13:00', endTime: '14:00', personal: 'Lena Huber', status: 'bestätigt' },
  { id: 'bk6', serviceId: 'bs7', kunde: 'Elena Fischer', datum: '2026-02-09', startTime: '14:00', endTime: '15:00', personal: 'Sandra Wyss', status: 'bestätigt' },
  { id: 'bk7', serviceId: 'bs4', kunde: 'Peter Zimmermann', datum: '2026-02-09', startTime: '15:00', endTime: '15:20', personal: 'Marco Roth', status: 'bestätigt' },
  { id: 'bk8', serviceId: 'bs8', kunde: 'Claudia Berger', datum: '2026-02-09', startTime: '15:30', endTime: '16:15', personal: 'Nina Frei', status: 'ausstehend' },
  // Past days
  { id: 'bk9', serviceId: 'bs1', kunde: 'David Müller', datum: '2026-02-07', startTime: '10:00', endTime: '10:30', personal: 'Lena Huber', status: 'bestätigt' },
  { id: 'bk10', serviceId: 'bs2', kunde: 'Monika Schwarz', datum: '2026-02-07', startTime: '11:00', endTime: '11:45', personal: 'Nina Frei', status: 'bestätigt' },
  { id: 'bk11', serviceId: 'bs6', kunde: 'Hans Kaufmann', datum: '2026-02-08', startTime: '09:00', endTime: '09:30', personal: 'Sandra Wyss', status: 'bestätigt' },
  { id: 'bk12', serviceId: 'bs3', kunde: 'Ursula Schmid', datum: '2026-02-08', startTime: '13:00', endTime: '14:30', personal: 'Nina Frei', status: 'abgesagt' },
]

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
  if (event.color) return event.color
  if (event.isHoliday) return '#9d8f85'
  if (event.isTaskDeadline) return '#a13f3f'
  const cal = calendars.find((c) => c.id === event.calendarId)
  return cal?.color ?? '#6b6159'
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

function minutesToTime(totalMinutes: number): string {
  const h = Math.floor(totalMinutes / 60)
  const m = totalMinutes % 60
  return `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}`
}

/** Current time as minutes-from-midnight, re-evaluated every minute. */
function useNowMinutes(): number {
  const [minutes, setMinutes] = useState(() => {
    const n = new Date()
    return n.getHours() * 60 + n.getMinutes()
  })
  useEffect(() => {
    const id = setInterval(() => {
      const n = new Date()
      setMinutes(n.getHours() * 60 + n.getMinutes())
    }, 60000)
    return () => clearInterval(id)
  }, [])
  return minutes
}

/** Work-hour grid bounds + hour ticks from the calendar settings store. */
function useWorkHours(): { startHour: number; endHour: number; hours: number[] } {
  const { workStartHour, workEndHour } = useSettingsStore((s) => s.calendar)
  // Guard against inverted/empty ranges from bad settings.
  const startHour = Math.min(workStartHour, workEndHour)
  const endHour = Math.max(workStartHour, workEndHour)
  const hours = Array.from({ length: endHour - startHour + 1 }, (_, i) => i + startHour)
  return { startHour, endHour, hours }
}

// ============================================================
// Main Component
// ============================================================

export default function KalenderPage() {
  const { t } = useTranslation()
  const [topTab, setTopTab] = useState<TopTab>('kalender')
  const calSettings = useSettingsStore((s) => s.calendar)
  const [view, setView] = useState<ViewMode>(calSettings.defaultView)
  const [currentDate, setCurrentDate] = useState(new Date())
  const [selectedDate, setSelectedDate] = useState(new Date())
  const [selectedEvent, setSelectedEvent] = useState<CalendarEvent | null>(null)
  const [quickCreate, setQuickCreate] = useState<QuickCreateState | null>(null)
  const [showEventForm, setShowEventForm] = useState(false)
  const [eventFormDefaults, setEventFormDefaults] = useState<Partial<CalendarEvent>>({})
  const [showRoomBooking, setShowRoomBooking] = useState(false)
  const [showCategoryManager, setShowCategoryManager] = useState(false)
  const [showCalendarBrowse, setShowCalendarBrowse] = useState(false)
  const [showCalendarSettings, setShowCalendarSettings] = useState(false)

  // Auth
  const currentUserId = useAuthStore((s) => s.user?.id ?? '')

  // Calendar visibility (local UI state)
  const [visibleCalendarIds, setVisibleCalendarIds] = useState<Set<string>>(new Set())
  const [visibilityInitialized, setVisibilityInitialized] = useState(false)

  // ---- API: Calendars ----
  const { data: calendarsData } = useCalendars()
  const calendars = useMemo<CalendarSource[]>(() => {
    const apiCals = (calendarsData?.calendars ?? []).map((c) =>
      calendarToUI(c, currentUserId, visibleCalendarIds),
    )
    const holidayCal: CalendarSource = { ...getHolidayCalendar(), visible: visibleCalendarIds.has('holidays') }
    const deadlineCal: CalendarSource = { ...getDeadlineCalendar(), visible: visibleCalendarIds.has('deadlines') }
    return [...apiCals, holidayCal, deadlineCal]
  }, [calendarsData, currentUserId, visibleCalendarIds])

  // Initialize visibility once calendars load
  useEffect(() => {
    if (visibilityInitialized || !calendarsData?.calendars?.length) return
    const ids = new Set<string>(calendarsData.calendars.map((c) => c.id))
    ids.add('holidays')
    ids.add('deadlines')
    // eslint-disable-next-line react-hooks/set-state-in-effect -- sync local editable state from prop
    setVisibleCalendarIds(ids)
    setVisibilityInitialized(true)
  }, [calendarsData, visibilityInitialized])

  // ---- API: Categories ----
  const { data: categoriesData } = useEventCategories()
  const categories = useMemo<EventCategory[]>(
    () => (categoriesData?.categories ?? []).map(categoryToUI),
    [categoriesData],
  )

  // ---- API: Events ----
  const { rangeStart, rangeEnd } = useMemo(() => {
    const d = new Date(currentDate)
    let start: Date
    let end: Date
    if (view === 'month') {
      start = new Date(d.getFullYear(), d.getMonth(), 1)
      end = new Date(d.getFullYear(), d.getMonth() + 1, 0)
      // Extend to full calendar grid (prev/next month padding)
      start.setDate(start.getDate() - 7)
      end.setDate(end.getDate() + 7)
    } else if (view === 'week') {
      const day = d.getDay()
      const diff = d.getDate() - day + (day === 0 ? -6 : 1)
      start = new Date(d.getFullYear(), d.getMonth(), diff)
      end = new Date(start)
      end.setDate(start.getDate() + 6)
    } else {
      start = new Date(d)
      end = new Date(d)
    }
    return {
      rangeStart: formatDateKey(start),
      rangeEnd: formatDateKey(end),
    }
  }, [currentDate, view])

  const apiCalendarIds = useMemo(
    () => calendars.filter((c) => c.visible && c.id !== 'holidays' && c.id !== 'deadlines').map((c) => c.id),
    [calendars],
  )

  const { data: eventsData, isLoading: eventsLoading } = useEventsInRange(apiCalendarIds, rangeStart, rangeEnd)
  const deadlinesVisible = calendars.find((c) => c.id === 'deadlines')?.visible ?? false
  const { data: deadlinesData } = useTaskDeadlines(rangeStart, rangeEnd)

  // ---- API: Holidays ----
  const holidaysVisible = calendars.find((c) => c.id === 'holidays')?.visible ?? false
  const [holidayCountry, holidaySubdivision] = (calSettings.holidayRegion || 'DE-BY').split('-')
  const { data: holidaysData } = useHolidays(
    currentDate.getFullYear(),
    holidayCountry,
    holidaySubdivision,
  )

  const events = useMemo<CalendarEvent[]>(() => {
    const apiEvents = (eventsData?.events ?? []).map(expandedEventToUI)
    const deadlines = deadlinesVisible ? (deadlinesData?.deadlines ?? []).map(deadlineToUI) : []
    const holidays = holidaysVisible ? (holidaysData?.holidays ?? []).map(holidayToUI) : []
    return [...apiEvents, ...deadlines, ...holidays]
  }, [eventsData, deadlinesData, deadlinesVisible, holidaysData, holidaysVisible])

  // ---- Mutations ----
  const createEventMutation = useCreateEvent()
  const updateEventMutation = useUpdateEvent()
  const deleteEventMutation = useDeleteEvent()
  const createCategoryMutation = useCreateEventCategory()
  const deleteCategoryMutation = useDeleteEventCategory()

  // 10.13: Push-Erinnerungen — track which events already notified
  const notifiedEventsRef = useRef<Set<string>>(new Set())

  useEffect(() => {
    const checkReminders = () => {
      const now = new Date()
      const todayKey = formatDateKey(now)
      const nowMinutes = now.getHours() * 60 + now.getMinutes()

      events
        .filter((e) => e.date === todayKey && !e.isAllDay && e.reminder && e.reminder !== 'Keine')
        .forEach((e) => {
          if (notifiedEventsRef.current.has(e.id)) return
          const eventMinutes = timeToMinutes(e.startTime)
          const diff = eventMinutes - nowMinutes
          if (diff > 0 && diff <= 15) {
            notifiedEventsRef.current.add(e.id)
            toast(t('kalender.reminder.upcoming', { minutes: diff, title: e.title }), {
              description: `${e.startTime} – ${e.endTime}`,
              action: {
                label: t('kalender.reminder.open'),
                onClick: () => setSelectedEvent(e),
              },
              duration: 10000,
            })
          }
        })
    }

    checkReminders()
    const interval = setInterval(checkReminders, 60000)
    return () => clearInterval(interval)
  }, [events])

  // Event update handler for drag-and-drop
  const handleUpdateEvent = useCallback((eventId: string, updates: Partial<CalendarEvent>) => {
    // Skip synthetic events (deadlines, holidays)
    if (eventId.startsWith('deadline-') || eventId.startsWith('holiday-')) return
    updateEventMutation.mutate({
      id: eventId,
      ...uiEventToUpdateRequest(updates),
    })
  }, [updateEventMutation])

  // Quick create handler
  const handleQuickCreate = useCallback((eventData: Partial<CalendarEvent>) => {
    const calId = eventData.calendarId || calendars.find((c) => c.group === 'mine')?.id || ''
    if (!calId) return
    createEventMutation.mutate(
      uiEventToCreateRequest(eventData, calId),
      {
        onSuccess: () => toast.success(t('kalender.event.created')),
        onError: () => toast.error(t('kalender.event.createError')),
      },
    )
  }, [createEventMutation, calendars, t])

  // Save event (create or update)
  const handleSaveEvent = useCallback((eventData: Partial<CalendarEvent>) => {
    if (eventData.id) {
      updateEventMutation.mutate(
        { id: eventData.id, ...uiEventToUpdateRequest(eventData) },
        {
          onSuccess: () => toast.success(t('kalender.event.updated')),
          onError: () => toast.error(t('kalender.event.updateError')),
        },
      )
    } else {
      const calId = eventData.calendarId || calendars.find((c) => c.group === 'mine')?.id || ''
      if (!calId) return
      createEventMutation.mutate(
        uiEventToCreateRequest(eventData, calId),
        {
          onSuccess: () => toast.success(t('kalender.event.created')),
          onError: () => toast.error(t('kalender.event.createError')),
        },
      )
    }
  }, [createEventMutation, updateEventMutation, calendars])

  // Delete event handler
  const handleDeleteEvent = useCallback((eventId: string) => {
    if (eventId.startsWith('deadline-') || eventId.startsWith('holiday-')) return
    deleteEventMutation.mutate(eventId, {
      onSuccess: () => {
        toast.success(t('kalender.event.deleted'))
        setSelectedEvent(null)
      },
      onError: () => toast.error(t('kalender.event.deleteError')),
    })
  }, [deleteEventMutation])

  const visibleEvents = useMemo(
    () => events.filter((e) => calendars.find((c) => c.id === e.calendarId)?.visible),
    [calendars, events],
  )

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
    setVisibleCalendarIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
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
    <div className="flex h-full flex-col overflow-hidden animate-fade-up">
      {/* Top-level tabs */}
      <div className="flex items-center gap-6 border-b border-border bg-card px-6 pt-3">
        <button
          onClick={() => setTopTab('kalender')}
          className={`border-b-2 px-1 pb-2 text-sm transition-colors ${topTab === 'kalender' ? 'font-medium' : 'border-transparent text-muted-foreground hover:text-foreground'}`}
          style={topTab === 'kalender' ? { borderColor: moduleHsl('calendar'), color: moduleHsl('calendar') } : undefined}
        >
          <Calendar className="mr-1.5 inline h-4 w-4" />
          {t('kalender.tabs.calendar')}
        </button>
        <button
          onClick={() => setTopTab('terminbuchung')}
          className={`border-b-2 px-1 pb-2 text-sm transition-colors ${topTab === 'terminbuchung' ? 'font-medium' : 'border-transparent text-muted-foreground hover:text-foreground'}`}
          style={topTab === 'terminbuchung' ? { borderColor: moduleHsl('calendar'), color: moduleHsl('calendar') } : undefined}
        >
          <CalendarCheck className="mr-1.5 inline h-4 w-4" />
          {t('kalender.tabs.booking')}
        </button>
      </div>

      {/* Tab content */}
      {topTab === 'kalender' ? (
        <div className="flex flex-1 overflow-hidden">
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
              onOpenSettings={() => setShowCalendarSettings(true)}
            />

            <div className="flex-1 overflow-auto bg-card">
              {eventsLoading ? (
                <div className="flex items-center justify-center h-full">
                  <div className="flex flex-col items-center gap-2 text-muted-foreground">
                    <div className="h-6 w-6 animate-spin rounded-full border-2 border-primary border-t-transparent" />
                    <span className="text-xs">{t('kalender.loading')}</span>
                  </div>
                </div>
              ) : (
                <>
                  {view === 'week' && (
                    <WeekView
                      currentDate={currentDate}
                      getEventsForDate={getEventsForDate}
                      calendars={calendars}
                      onSelectEvent={setSelectedEvent}
                      onSlotClick={handleSlotClick}
                      onDateClick={handleDateClick}
                      onUpdateEvent={handleUpdateEvent}
                    />
                  )}
                  {view === 'day' && (
                    <DayView
                      currentDate={currentDate}
                      events={getEventsForDate(currentDate)}
                      calendars={calendars}
                      onSelectEvent={setSelectedEvent}
                      onSlotClick={handleSlotClick}
                      onUpdateEvent={handleUpdateEvent}
                    />
                  )}
                  {view === 'month' && (
                    <MonthView
                      currentDate={currentDate}
                      getEventsForDate={getEventsForDate}
                      calendars={calendars}
                      onSelectEvent={setSelectedEvent}
                      onDateClick={handleDateClick}
                    />
                  )}
                </>
              )}
            </div>
          </div>

          {/* Quick-create popover */}
          {quickCreate && (
            <QuickCreatePopover
              state={quickCreate}
              categories={categories}
              onClose={() => setQuickCreate(null)}
              onSave={(eventData) => {
                handleQuickCreate(eventData)
                setQuickCreate(null)
              }}
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
              categories={categories}
              calendars={calendars}
              isSaving={createEventMutation.isPending || updateEventMutation.isPending}
              onSave={(eventData) => {
                handleSaveEvent(eventData)
                setShowEventForm(false)
              }}
              onClose={() => setShowEventForm(false)}
            />
          )}

          {/* Event detail panel */}
          {selectedEvent && (
            <EventDetailPanel
              event={selectedEvent}
              calendars={calendars}
              categories={categories}
              onClose={() => setSelectedEvent(null)}
              onEdit={() => {
                setEventFormDefaults(selectedEvent)
                setSelectedEvent(null)
                setShowEventForm(true)
              }}
              onDelete={() => handleDeleteEvent(selectedEvent.id)}
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
            onCreateCategory={(name, color) => createCategoryMutation.mutate({ name, color })}
            onDeleteCategory={(id) => deleteCategoryMutation.mutate(id)}
          />

          {/* Calendar browse */}
          <CalendarBrowseDialog
            open={showCalendarBrowse}
            onOpenChange={setShowCalendarBrowse}
            calendars={calendars}
            onToggleCalendar={toggleCalendar}
          />

          {/* Calendar Settings Modal */}
          <Dialog open={showCalendarSettings} onOpenChange={setShowCalendarSettings}>
            <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
              <DialogHeader>
                <DialogTitle>{t('kalender.einstellungen.title', { defaultValue: 'Kalender-Einstellungen' })}</DialogTitle>
              </DialogHeader>
              <CalendarSettingsTab />
            </DialogContent>
          </Dialog>
        </div>
      ) : (
        <TerminbuchungTab />
      )}
    </div>
  )
}

// ============================================================
// Terminbuchung Tab
// ============================================================

function TerminbuchungTab() {
  const { t } = useTranslation()
  const [showNewBooking, setShowNewBooking] = useState(false)
  const [bookingDate, setBookingDate] = useState('2026-02-09')
  const [bookings, setBookings] = useState(MOCK_BOOKINGS)
  const [buchungSubTab, setBuchungSubTab] = useState<'übersicht' | 'vorschau'>('übersicht')

  const bookingsForDate = useMemo(
    () => bookings
      .filter((b) => b.datum === bookingDate)
      .sort((a, b) => timeToMinutes(a.startTime) - timeToMinutes(b.startTime)),
    [bookings, bookingDate],
  )

  const getServiceById = (id: string) => BOOKING_SERVICES.find((s) => s.id === id)

  const statusLabel = (status: BookingAppointment['status']) => {
    switch (status) {
      case 'bestätigt': return { text: t('kalender.booking.confirmed'), cls: 'bg-success/15 text-success' }
      case 'ausstehend': return { text: t('kalender.booking.pending'), cls: 'bg-warning/15 text-warning' }
      case 'abgesagt': return { text: t('kalender.booking.cancelled'), cls: 'bg-error/15 text-error' }
    }
  }

  const handleCreateBooking = (newBooking: Omit<BookingAppointment, 'id'>) => {
    const id = `bk${Date.now()}`
    setBookings((prev) => [...prev, { ...newBooking, id }])
    setShowNewBooking(false)
    toast.success(t('kalender.booking.createdSuccess'))
  }

  const todayBookingCount = bookings.filter((b) => b.datum === '2026-02-09' && b.status !== 'abgesagt').length
  const todayRevenue = bookings
    .filter((b) => b.datum === '2026-02-09' && b.status !== 'abgesagt')
    .reduce((sum, b) => sum + (getServiceById(b.serviceId)?.preis ?? 0), 0)

  return (
    <div className="flex-1 overflow-auto bg-card">
      <div className="mx-auto max-w-6xl p-6 space-y-6">
        {/* Sub-tabs: Übersicht / Vorschau */}
        <div className="flex items-center gap-4 border-b border-border pb-0">
          <button
            onClick={() => setBuchungSubTab('übersicht')}
            className={`border-b-2 px-1 pb-2 text-xs transition-colors ${buchungSubTab === 'übersicht' ? 'border-primary text-primary font-medium tab-accent-active' : 'border-transparent text-muted-foreground hover:text-foreground'}`}
          >
            {t('kalender.booking.overview')}
          </button>
          <button
            onClick={() => setBuchungSubTab('vorschau')}
            className={`border-b-2 px-1 pb-2 text-xs transition-colors ${buchungSubTab === 'vorschau' ? 'border-primary text-primary font-medium tab-accent-active' : 'border-transparent text-muted-foreground hover:text-foreground'}`}
          >
            <ExternalLink className="mr-1 inline h-3 w-3" />
            {t('kalender.booking.linkPreview')}
          </button>
        </div>

        {buchungSubTab === 'vorschau' ? (
          <ExternalBookingPreview />
        ) : (
        <>
        {/* Header row with stats and action */}
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold text-foreground">{t('kalender.tabs.booking')}</h2>
            <p className="text-xs text-muted-foreground mt-0.5">
              {t('kalender.booking.todayStats', { count: todayBookingCount, revenue: todayRevenue.toFixed(0) })}
            </p>
          </div>
          <button
            onClick={() => setShowNewBooking(true)}
            className="flex items-center gap-1.5 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors"
          >
            <Plus className="h-4 w-4" />
            {t('kalender.booking.newAppointment')}
          </button>
        </div>

        {/* Service Catalog */}
        <div>
          <h3 className="text-sm font-medium text-foreground mb-3">{t('kalender.booking.serviceCatalog')}</h3>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
            {BOOKING_SERVICES.map((service) => (
              <div key={service.id} className="rounded-lg border border-border bg-card p-4 hover:shadow-sm transition-shadow">
                <div className="flex items-start justify-between mb-2">
                  <div className="flex items-center gap-2">
                    <span className="h-3 w-3 rounded-full shrink-0" style={{ backgroundColor: service.color }} />
                    <h4 className="text-sm font-medium text-foreground">{service.name}</h4>
                  </div>
                </div>
                <div className="space-y-1.5">
                  <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                    <Clock className="h-3 w-3" />
                    <span>{service.dauer} Min.</span>
                  </div>
                  <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                    <Euro className="h-3 w-3" />
                    <span className="font-medium text-foreground">CHF {service.preis.toFixed(0)}</span>
                  </div>
                  <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                    <User className="h-3 w-3" />
                    <span>{service.personal.join(', ')}</span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Day Overview (Timeline) */}
        <div>
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-sm font-medium text-foreground">{t('kalender.booking.dayOverview')}</h3>
            <div className="flex items-center gap-2">
              <button
                onClick={() => {
                  const d = new Date(bookingDate)
                  d.setDate(d.getDate() - 1)
                  setBookingDate(formatDateKey(d))
                }}
                className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary transition-colors"
              >
                <ChevronLeft className="h-4 w-4" />
              </button>
              <input
                type="date"
                value={bookingDate}
                onChange={(e) => setBookingDate(e.target.value)}
                className="rounded-lg border border-input-border bg-input-background px-3 py-1.5 text-xs text-foreground outline-none focus:border-primary"
              />
              <button
                onClick={() => {
                  const d = new Date(bookingDate)
                  d.setDate(d.getDate() + 1)
                  setBookingDate(formatDateKey(d))
                }}
                className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary transition-colors"
              >
                <ChevronRight className="h-4 w-4" />
              </button>
            </div>
          </div>

          {bookingsForDate.length === 0 ? (
            <div className="rounded-lg border border-border bg-card p-8 text-center">
              <CalendarCheck className="mx-auto h-8 w-8 text-muted-foreground mb-2" />
              <p className="text-sm text-muted-foreground">{t('kalender.booking.noAppointments')}</p>
            </div>
          ) : (
            <div className="rounded-lg border border-border bg-card overflow-hidden">
              {/* Timeline header */}
              <div className="grid grid-cols-[80px_1fr_120px_120px_100px] gap-3 border-b border-border bg-secondary/30 px-4 py-2">
                <span className="text-[10px] uppercase tracking-wider font-semibold text-muted-foreground">{t('kalender.booking.time')}</span>
                <span className="text-[10px] uppercase tracking-wider font-semibold text-muted-foreground">{t('kalender.booking.appointment')}</span>
                <span className="text-[10px] uppercase tracking-wider font-semibold text-muted-foreground">{t('kalender.booking.staff')}</span>
                <span className="text-[10px] uppercase tracking-wider font-semibold text-muted-foreground">{t('kalender.booking.client')}</span>
                <span className="text-[10px] uppercase tracking-wider font-semibold text-muted-foreground">{t('common.status')}</span>
              </div>

              {/* Timeline rows */}
              {bookingsForDate.map((booking) => {
                const service = getServiceById(booking.serviceId)
                const status = statusLabel(booking.status)
                return (
                  <div
                    key={booking.id}
                    className="grid grid-cols-[80px_1fr_120px_120px_100px] gap-3 border-b border-border-muted px-4 py-3 hover:bg-secondary/20 transition-colors items-center"
                  >
                    {/* Time */}
                    <div className="text-xs font-medium text-foreground">
                      {booking.startTime} – {booking.endTime}
                    </div>

                    {/* Service */}
                    <div className="flex items-center gap-2 min-w-0">
                      <span
                        className="h-2.5 w-2.5 rounded-full shrink-0"
                        style={{ backgroundColor: service?.color ?? '#888' }}
                      />
                      <div className="min-w-0">
                        <p className="text-xs font-medium text-foreground truncate">{service?.name}</p>
                        <p className="text-[10px] text-muted-foreground">{service?.dauer} Min. &middot; CHF {service?.preis}</p>
                      </div>
                    </div>

                    {/* Staff */}
                    <span className="text-xs text-muted-foreground truncate">{booking.personal}</span>

                    {/* Client */}
                    <span className="text-xs text-foreground truncate">{booking.kunde}</span>

                    {/* Status */}
                    <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium text-center ${status.cls}`}>
                      {status.text}
                    </span>
                  </div>
                )
              })}
            </div>
          )}
        </div>

        {/* Visual timeline blocks */}
        {bookingsForDate.length > 0 && (
          <div>
            <h3 className="text-sm font-medium text-foreground mb-3">{t('kalender.booking.timeline')}</h3>
            <div className="rounded-lg border border-border bg-card p-4">
              {/* Staff rows */}
              {BOOKING_STAFF.map((staff) => {
                const staffBookings = bookingsForDate.filter((b) => b.personal === staff)
                return (
                  <div key={staff} className="flex items-center gap-3 mb-3 last:mb-0">
                    <div className="w-28 shrink-0">
                      <p className="text-xs font-medium text-foreground truncate">{staff}</p>
                    </div>
                    <div className="flex-1 relative h-8 rounded bg-secondary/40">
                      {/* Hour markers */}
                      {Array.from({ length: 10 }, (_, i) => i + 8).map((h) => (
                        <div
                          key={h}
                          className="absolute top-0 bottom-0 border-l border-border-muted/60"
                          style={{ left: `${((h - 8) / 10) * 100}%` }}
                        >
                          <span className="text-[8px] text-muted-foreground ml-0.5">{h}:00</span>
                        </div>
                      ))}
                      {/* Booking blocks */}
                      {staffBookings.map((b) => {
                        const service = getServiceById(b.serviceId)
                        const startMin = timeToMinutes(b.startTime) - 8 * 60 // start from 08:00
                        const endMin = timeToMinutes(b.endTime) - 8 * 60
                        const totalMin = 10 * 60 // 08:00-18:00
                        const leftPct = Math.max(0, (startMin / totalMin) * 100)
                        const widthPct = Math.max(2, ((endMin - startMin) / totalMin) * 100)
                        return (
                          <div
                            key={b.id}
                            className="absolute top-0.5 bottom-0.5 rounded-[3px] flex items-center px-1.5 overflow-hidden"
                            style={{
                              left: `${leftPct}%`,
                              width: `${widthPct}%`,
                              backgroundColor: `${service?.color ?? '#888'}30`,
                              borderLeft: `3px solid ${service?.color ?? '#888'}`,
                            }}
                            title={`${b.startTime}-${b.endTime} ${b.kunde} (${service?.name})`}
                          >
                            <span className="text-[9px] font-medium truncate" style={{ color: service?.color }}>
                              {b.kunde}
                            </span>
                          </div>
                        )
                      })}
                    </div>
                  </div>
                )
              })}
            </div>
          </div>
        )}

      {/* New Booking Dialog */}
      {showNewBooking && (
        <NewBookingDialog
          onClose={() => setShowNewBooking(false)}
          onSave={handleCreateBooking}
        />
      )}
        </>
        )}
      </div>
    </div>
  )
}

// ============================================================
// External Booking Preview (10.14)
// ============================================================

function ExternalBookingPreview() {
  const { t } = useTranslation()
  const [selectedService, setSelectedService] = useState<string | null>(null)
  const [selectedDate, setSelectedDate] = useState<string | null>(null)
  const [selectedTime, setSelectedTime] = useState<string | null>(null)
  const [bookingForm, setBookingForm] = useState({ name: '', email: '', phone: '', notes: '' })
  const [step, setStep] = useState(1)

  // Generate dates for current + next week (14 days starting from mock today)
  const availableDates = useMemo(() => {
    const dates: { key: string; label: string; dayShort: string; dayNum: number; available: boolean }[] = []
    const baseDate = new Date(2026, 1, 9) // mock today
    for (let i = 0; i < 14; i++) {
      const d = new Date(baseDate)
      d.setDate(baseDate.getDate() + i)
      const dayOfWeek = d.getDay()
      // Mon-Fri available, weekends not
      dates.push({
        key: formatDateKey(d),
        label: `${d.getDate()}. ${MONTHS_DE[d.getMonth()].slice(0, 3)}`,
        dayShort: DAYS_SHORT[(dayOfWeek + 6) % 7],
        dayNum: d.getDate(),
        available: dayOfWeek >= 1 && dayOfWeek <= 5,
      })
    }
    return dates
  }, [])

  // Simulate some slots being taken
  const availableSlots = useMemo(() => {
    if (!selectedDate) return []
    const takenSlots = new Set<string>()
    // Mock: some slots are taken on certain days
    if (selectedDate === '2026-02-10') { takenSlots.add('09:00'); takenSlots.add('10:30'); takenSlots.add('14:00') }
    if (selectedDate === '2026-02-11') { takenSlots.add('11:00'); takenSlots.add('13:30') }
    if (selectedDate === '2026-02-12') { takenSlots.add('09:30'); takenSlots.add('15:00') }
    return EXTERNAL_TIME_SLOTS.filter((s) => !takenSlots.has(s))
  }, [selectedDate])

  const handleBook = () => {
    if (!bookingForm.name.trim() || !bookingForm.email.trim()) {
      toast.error(t('kalender.external.fillRequired'))
      return
    }
    toast.success(t('kalender.external.booked'))
    setStep(1)
    setSelectedService(null)
    setSelectedDate(null)
    setSelectedTime(null)
    setBookingForm({ name: '', email: '', phone: '', notes: '' })
  }

  const selectedServiceObj = MOCK_EXTERNAL_SERVICES.find((s) => s.id === selectedService)

  return (
    <div className="space-y-4">
      {/* Info banner */}
      <div className="flex items-center justify-between rounded-lg border border-primary/30 bg-primary-subtle px-4 py-3">
        <div className="flex items-center gap-2">
          <ExternalLink className="h-4 w-4 text-primary" />
          <p className="text-xs text-primary font-medium">{t('kalender.external.previewBanner')}</p>
        </div>
        <button
          onClick={() => {
            navigator.clipboard?.writeText('https://booking.zentria.tech/firma/cosmi-demo')
            toast.success(t('kalender.external.linkCopied'))
          }}
          className="flex items-center gap-1.5 rounded-lg border border-border bg-card px-3 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors"
        >
          <Copy className="h-3 w-3" />
          {t('kalender.external.copyLink')}
        </button>
      </div>

      {/* Preview panel — simulated customer view */}
      <div className="mx-auto max-w-xl rounded-xl border border-border bg-card shadow-[var(--shadow-large)] overflow-hidden">
        {/* Company header */}
        <div className="border-b border-border bg-primary px-6 py-5 text-center">
          <div className="inline-flex h-12 w-12 items-center justify-center rounded-full bg-white/20 mb-2">
            <Calendar className="h-6 w-6 text-white" />
          </div>
          <h3 className="text-base font-semibold text-white">Cosmi GmbH</h3>
          <p className="text-xs text-white/70 mt-0.5">{t('kalender.external.onlineBooking')}</p>
        </div>

        <div className="p-6 space-y-5">
          {/* Step indicator */}
          <div className="flex items-center gap-2 justify-center">
            {[1, 2, 3, 4].map((s) => (
              <div key={s} className="flex items-center gap-2">
                <div
                  className={`flex h-6 w-6 items-center justify-center rounded-full text-[10px] font-medium transition-colors ${
                    step >= s ? 'bg-primary text-primary-foreground' : 'bg-secondary text-muted-foreground'
                  }`}
                >
                  {s}
                </div>
                {s < 4 && <div className={`h-0.5 w-6 rounded-full ${step > s ? 'bg-primary' : 'bg-secondary'}`} />}
              </div>
            ))}
          </div>

          {/* Step 1: Service selection */}
          {step === 1 && (
            <div className="space-y-3">
              <h4 className="text-sm font-medium text-foreground text-center">{t('kalender.external.selectService')}</h4>
              <div className="space-y-2">
                {MOCK_EXTERNAL_SERVICES.map((svc) => (
                  <button
                    key={svc.id}
                    onClick={() => {
                      setSelectedService(svc.id)
                      setStep(2)
                    }}
                    className={`w-full flex items-center justify-between rounded-lg border px-4 py-3 transition-colors ${
                      selectedService === svc.id
                        ? 'border-primary bg-primary-subtle'
                        : 'border-border hover:border-primary/50 hover:bg-secondary/30'
                    }`}
                  >
                    <div className="text-left">
                      <p className="text-sm font-medium text-foreground">{svc.name}</p>
                      <div className="flex items-center gap-2 mt-0.5">
                        <span className="flex items-center gap-1 text-[10px] text-muted-foreground">
                          <Clock className="h-3 w-3" /> {svc.duration} Min.
                        </span>
                        {svc.price > 0 && (
                          <span className="text-[10px] text-muted-foreground">EUR {svc.price}</span>
                        )}
                        {svc.price === 0 && (
                          <span className="text-[10px] text-success font-medium">{t('kalender.external.free')}</span>
                        )}
                      </div>
                    </div>
                    <ChevronRight className="h-4 w-4 text-muted-foreground" />
                  </button>
                ))}
              </div>
            </div>
          )}

          {/* Step 2: Date selection */}
          {step === 2 && (
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <button onClick={() => setStep(1)} className="text-xs text-primary hover:underline">{t('common.back')}</button>
                <h4 className="text-sm font-medium text-foreground">{t('kalender.external.selectDate')}</h4>
                <div className="w-10" />
              </div>
              {selectedServiceObj && (
                <p className="text-center text-xs text-muted-foreground">
                  {selectedServiceObj.name} ({selectedServiceObj.duration} Min.)
                </p>
              )}
              <div className="grid grid-cols-7 gap-1.5">
                {availableDates.map((d) => (
                  <button
                    key={d.key}
                    disabled={!d.available}
                    onClick={() => {
                      if (d.available) {
                        setSelectedDate(d.key)
                        setStep(3)
                      }
                    }}
                    className={`flex flex-col items-center rounded-lg py-2 text-xs transition-colors ${
                      !d.available
                        ? 'text-text-disabled cursor-not-allowed opacity-40'
                        : selectedDate === d.key
                          ? 'bg-primary text-primary-foreground'
                          : 'hover:bg-secondary text-foreground border border-border'
                    }`}
                  >
                    <span className="text-[9px] uppercase">{d.dayShort}</span>
                    <span className="text-sm font-medium mt-0.5">{d.dayNum}</span>
                  </button>
                ))}
              </div>
            </div>
          )}

          {/* Step 3: Time slot selection */}
          {step === 3 && (
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <button onClick={() => setStep(2)} className="text-xs text-primary hover:underline">{t('common.back')}</button>
                <h4 className="text-sm font-medium text-foreground">{t('kalender.external.selectTime')}</h4>
                <div className="w-10" />
              </div>
              <p className="text-center text-xs text-muted-foreground">
                {selectedDate && (() => {
                  const d = new Date(selectedDate)
                  return `${DAYS_SHORT[(d.getDay() + 6) % 7]}, ${d.getDate()}. ${MONTHS_DE[d.getMonth()]}`
                })()}
              </p>
              {availableSlots.length === 0 ? (
                <p className="text-center text-sm text-muted-foreground py-4">{t('kalender.external.noSlots')}</p>
              ) : (
                <div className="grid grid-cols-3 gap-2">
                  {availableSlots.map((slot) => (
                    <button
                      key={slot}
                      onClick={() => {
                        setSelectedTime(slot)
                        setStep(4)
                      }}
                      className={`rounded-lg border py-2.5 text-sm font-medium transition-colors ${
                        selectedTime === slot
                          ? 'border-primary bg-primary text-primary-foreground'
                          : 'border-border hover:border-primary text-foreground hover:bg-primary-subtle'
                      }`}
                    >
                      {slot}
                    </button>
                  ))}
                </div>
              )}
            </div>
          )}

          {/* Step 4: Contact form */}
          {step === 4 && (
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <button onClick={() => setStep(3)} className="text-xs text-primary hover:underline">{t('common.back')}</button>
                <h4 className="text-sm font-medium text-foreground">{t('kalender.external.yourData')}</h4>
                <div className="w-10" />
              </div>

              {/* Summary */}
              <div className="rounded-lg bg-secondary/50 p-3 space-y-1 text-xs">
                <div className="flex items-center gap-2 text-muted-foreground">
                  <Calendar className="h-3 w-3" />
                  <span>{selectedServiceObj?.name}</span>
                </div>
                <div className="flex items-center gap-2 text-muted-foreground">
                  <Clock className="h-3 w-3" />
                  <span>
                    {selectedDate && (() => {
                      const d = new Date(selectedDate)
                      return `${d.getDate()}. ${MONTHS_DE[d.getMonth()]} ${d.getFullYear()}`
                    })()} {t('kalender.external.atTime', { time: selectedTime })}
                  </span>
                </div>
              </div>

              {/* Name */}
              <div>
                <label className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1 block">
                  <User className="h-3 w-3 inline mr-1" />
                  Name *
                </label>
                <input
                  type="text"
                  placeholder={t('kalender.external.namePlaceholder')}
                  value={bookingForm.name}
                  onChange={(e) => setBookingForm({ ...bookingForm, name: e.target.value })}
                  className="w-full rounded-lg border border-input-border bg-input-background px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder outline-none focus:border-primary"
                />
              </div>

              {/* Email */}
              <div>
                <label className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1 block">
                  <Mail className="h-3 w-3 inline mr-1" />
                  E-Mail *
                </label>
                <input
                  type="email"
                  placeholder="ihre@email.de"
                  value={bookingForm.email}
                  onChange={(e) => setBookingForm({ ...bookingForm, email: e.target.value })}
                  className="w-full rounded-lg border border-input-border bg-input-background px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder outline-none focus:border-primary"
                />
              </div>

              {/* Phone */}
              <div>
                <label className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1 block">
                  <Phone className="h-3 w-3 inline mr-1" />
                  Telefon
                </label>
                <input
                  type="tel"
                  placeholder="+49 123 456789"
                  value={bookingForm.phone}
                  onChange={(e) => setBookingForm({ ...bookingForm, phone: e.target.value })}
                  className="w-full rounded-lg border border-input-border bg-input-background px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder outline-none focus:border-primary"
                />
              </div>

              {/* Notes */}
              <div>
                <label className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1 block">
                  <FileText className="h-3 w-3 inline mr-1" />
                  {t('kalender.external.notes')}
                </label>
                <textarea
                  rows={2}
                  placeholder={t('kalender.external.notesPlaceholder')}
                  value={bookingForm.notes}
                  onChange={(e) => setBookingForm({ ...bookingForm, notes: e.target.value })}
                  className="w-full rounded-lg border border-input-border bg-input-background px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder outline-none resize-none focus:border-primary"
                />
              </div>

              {/* Book button */}
              <button
                onClick={handleBook}
                className="w-full rounded-lg bg-primary py-2.5 text-sm font-medium text-primary-foreground hover:bg-button-primary-hover transition-colors"
              >
                {t('kalender.external.bookAppointment')}
              </button>

              <p className="text-center text-[10px] text-muted-foreground">
                {t('kalender.external.termsNote')}
              </p>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="border-t border-border px-6 py-3 text-center">
          <p className="text-[10px] text-muted-foreground">
            Powered by <span className="font-medium text-foreground">Cosmi</span>
          </p>
        </div>
      </div>
    </div>
  )
}

// ============================================================
// New Booking Dialog
// ============================================================

function NewBookingDialog({
  onClose,
  onSave,
}: {
  onClose: () => void
  onSave: (booking: Omit<BookingAppointment, 'id'>) => void
}) {
  const { t } = useTranslation()
  const [serviceId, setServiceId] = useState(BOOKING_SERVICES[0].id)
  const [kunde, setKunde] = useState('')
  const [datum, setDatum] = useState('2026-02-09')
  const [startTime, setStartTime] = useState('09:00')
  const [personal, setPersonal] = useState(BOOKING_STAFF[0])
  const [notizen, setNotizen] = useState('')

  const selectedService = BOOKING_SERVICES.find((s) => s.id === serviceId)!

  // Compute end time from service duration
  const endTime = useMemo(() => {
    const startMin = timeToMinutes(startTime)
    const endMin = startMin + selectedService.dauer
    const h = Math.floor(endMin / 60)
    const m = endMin % 60
    return `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}`
  }, [startTime, selectedService.dauer])

  const availableStaff = selectedService.personal

  const handleSubmit = () => {
    if (!kunde.trim()) {
      toast.error(t('kalender.newBooking.clientRequired'))
      return
    }
    onSave({
      serviceId,
      kunde: kunde.trim(),
      datum,
      startTime,
      endTime,
      personal,
      notizen: notizen.trim() || undefined,
      status: 'bestätigt',
    })
  }

  return (
    <Dialog open onOpenChange={(open) => { if (!open) onClose() }}>
      <DialogContent className="gap-0 p-0 max-w-md max-h-[85vh] overflow-hidden flex flex-col">
        <DialogHeader className="border-b border-border px-5 py-3">
          <DialogTitle className="text-sm font-medium text-foreground">{t('kalender.booking.newAppointment')}</DialogTitle>
        </DialogHeader>

        {/* Form */}
        <div className="overflow-y-auto flex-1 p-5 space-y-4">
          {/* Service */}
          <div>
            <label className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1 block">
              <Scissors className="h-3 w-3 inline mr-1" />
              Service
            </label>
            <select
              value={serviceId}
              onChange={(e) => {
                setServiceId(e.target.value)
                const svc = BOOKING_SERVICES.find((s) => s.id === e.target.value)
                if (svc && !svc.personal.includes(personal)) {
                  setPersonal(svc.personal[0])
                }
              }}
              className="w-full rounded-lg border border-input-border bg-input-background px-3 py-2 text-sm text-foreground outline-none focus:border-primary"
            >
              {BOOKING_SERVICES.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name} — {s.dauer} Min. — CHF {s.preis}
                </option>
              ))}
            </select>
          </div>

          {/* Kunde */}
          <div>
            <label className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1 block">
              <User className="h-3 w-3 inline mr-1" />
              Kunde
            </label>
            <input
              type="text"
              placeholder={t('kalender.newBooking.clientPlaceholder')}
              value={kunde}
              onChange={(e) => setKunde(e.target.value)}
              className="w-full rounded-lg border border-input-border bg-input-background px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder outline-none focus:border-primary"
            />
          </div>

          {/* Date & Time */}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1 block">{t('kalender.form.date')}</label>
              <input
                type="date"
                value={datum}
                onChange={(e) => setDatum(e.target.value)}
                className="w-full rounded-lg border border-input-border bg-input-background px-3 py-1.5 text-xs text-foreground outline-none focus:border-primary"
              />
            </div>
            <div>
              <label className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1 block">{t('kalender.newBooking.timeSlot')}</label>
              <div className="flex items-center gap-2">
                <input
                  type="time"
                  value={startTime}
                  onChange={(e) => setStartTime(e.target.value)}
                  className="flex-1 rounded-lg border border-input-border bg-input-background px-2 py-1.5 text-xs text-foreground outline-none focus:border-primary"
                />
                <span className="text-xs text-muted-foreground">{t('kalender.room.until')}</span>
                <span className="text-xs text-foreground font-medium">{endTime}</span>
              </div>
              <p className="text-[10px] text-muted-foreground mt-0.5">{t('kalender.newBooking.duration', { minutes: selectedService.dauer })}</p>
            </div>
          </div>

          {/* Personal */}
          <div>
            <label className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1 block">
              <Users className="h-3 w-3 inline mr-1" />
              Personal
            </label>
            <select
              value={personal}
              onChange={(e) => setPersonal(e.target.value)}
              className="w-full rounded-lg border border-input-border bg-input-background px-3 py-1.5 text-xs text-foreground outline-none focus:border-primary"
            >
              {availableStaff.map((name) => (
                <option key={name} value={name}>{name}</option>
              ))}
            </select>
          </div>

          {/* Notizen */}
          <div>
            <label className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1 block">
              <FileText className="h-3 w-3 inline mr-1" />
              Notizen
            </label>
            <textarea
              rows={3}
              placeholder={t('kalender.newBooking.notesPlaceholder')}
              value={notizen}
              onChange={(e) => setNotizen(e.target.value)}
              className="w-full rounded-lg border border-input-border bg-input-background px-3 py-2 text-xs text-foreground placeholder:text-input-placeholder outline-none resize-none focus:border-primary"
            />
          </div>

          {/* Price preview */}
          <div className="rounded-lg bg-secondary/50 px-4 py-3 flex items-center justify-between">
            <span className="text-xs text-muted-foreground">{t('kalender.newBooking.price')}</span>
            <span className="text-sm font-semibold text-foreground">CHF {selectedService.preis.toFixed(2)}</span>
          </div>
        </div>

        {/* Footer */}
        <DialogFooter className="border-t border-border px-5 py-3">
          <button
            onClick={onClose}
            className="rounded-lg border border-border px-4 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors"
          >
            {t('common.cancel')}
          </button>
          <button
            onClick={handleSubmit}
            className="rounded-lg bg-primary px-4 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors"
          >
            {t('kalender.newBooking.create')}
          </button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
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
  const { t } = useTranslation()
  const dayEvents = events
    .filter((e) => e.date === formatDateKey(selectedDate) && !e.isAllDay)
    .sort((a, b) => timeToMinutes(a.startTime) - timeToMinutes(b.startTime))

  const allDayEvents = events.filter(
    (e) => e.date === formatDateKey(selectedDate) && e.isAllDay,
  )

  const groups = [
    { label: t('kalender.sidebar.myCalendars'), items: calendars.filter((c) => c.group === 'mine') },
    { label: t('kalender.sidebar.sharedCalendars'), items: calendars.filter((c) => c.group === 'shared') },
    { label: t('kalender.sidebar.other'), items: calendars.filter((c) => c.group === 'other') },
  ]

  const dayLabel = `${DAYS_SHORT[(selectedDate.getDay() + 6) % 7]}, ${selectedDate.getDate()}. ${MONTHS_DE[selectedDate.getMonth()]}`

  return (
    <aside className="hidden lg:flex w-72 shrink-0 flex-col border-r border-border bg-card overflow-y-auto">
      {/* Day agenda */}
      <div className="p-4 border-b border-border">
        <h3 className="text-sm font-medium text-foreground mb-1">{t('kalender.sidebar.dayAgenda')}</h3>
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
                <span className="truncate text-foreground">{t('kalender.sidebar.allDay')}: {e.title}</span>
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
          <p className="text-xs text-muted-foreground italic">{t('kalender.sidebar.noEvents')}</p>
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
  onOpenSettings,
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
  onOpenSettings: () => void
}) {
  const { t } = useTranslation()
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
          {t('kalender.toolbar.today')}
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
              {v === 'day' ? t('kalender.toolbar.day') : v === 'week' ? t('kalender.toolbar.week') : t('kalender.toolbar.month')}
            </button>
          ))}
        </div>
        <span className="mx-0.5 h-5 w-px bg-border" />
        <button
          onClick={onOpenRooms}
          className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
          title={t('kalender.toolbar.roomPlanning')}
        >
          <DoorOpen className="h-4 w-4" />
        </button>
        <button
          onClick={onOpenCalendarBrowse}
          className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
          title={t('kalender.toolbar.manageCalendars')}
        >
          <Layers className="h-4 w-4" />
        </button>
        <button
          onClick={onOpenCategories}
          className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
          title={t('kalender.toolbar.categories')}
        >
          <Settings2 className="h-4 w-4" />
        </button>
        <button
          onClick={onOpenSettings}
          className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
          title={t('kalender.einstellungen.title', { defaultValue: 'Einstellungen' })}
        >
          <Settings2 className="h-4 w-4 opacity-60" />
        </button>
        <span className="mx-0.5 h-5 w-px bg-border" />
        <button
          onClick={onNewEvent}
          className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors"
        >
          <Plus className="h-3.5 w-3.5" />
          {t('kalender.toolbar.newEvent')}
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
  onUpdateEvent,
}: {
  currentDate: Date
  getEventsForDate: (d: Date) => CalendarEvent[]
  calendars: CalendarSource[]
  onSelectEvent: (e: CalendarEvent) => void
  onSlotClick: (date: string, hour: number, minute: number, e: React.MouseEvent) => void
  onDateClick: (d: Date) => void
  onUpdateEvent: (eventId: string, updates: Partial<CalendarEvent>) => void
}) {
  const { t } = useTranslation()
  const { startHour: START_HOUR, endHour: END_HOUR, hours: HOURS } = useWorkHours()
  const nowMinutes = useNowMinutes()
  const weekDays = getWeekDays(currentDate, true) // Mo-Fr
  const [dragState, setDragState] = useState<DragState | null>(null)
  const gridRef = useRef<HTMLDivElement>(null)

  // 10.11: Drag-and-drop mouse handlers
  const handleEventMouseDown = useCallback((
    e: React.MouseEvent,
    event: CalendarEvent,
    dayDate: string,
    mode: 'move' | 'resize',
  ) => {
    e.preventDefault()
    e.stopPropagation()
    const startMin = timeToMinutes(event.startTime)
    setDragState({
      eventId: event.id,
      startY: e.clientY,
      startHour: Math.floor(startMin / 60),
      startMinute: startMin % 60,
      currentY: e.clientY,
      mode,
      dayDate,
      originalStartTime: event.startTime,
      originalEndTime: event.endTime,
    })
  }, [])

  useEffect(() => {
    if (!dragState) return

    const handleMouseMove = (e: MouseEvent) => {
      setDragState((prev) => prev ? { ...prev, currentY: e.clientY } : null)
    }

    const handleMouseUp = () => {
      if (!dragState) return
      const deltaY = dragState.currentY - dragState.startY
      const origStartMin = timeToMinutes(dragState.originalStartTime)
      const origEndMin = timeToMinutes(dragState.originalEndTime)
      const duration = origEndMin - origStartMin

      if (dragState.mode === 'move') {
        const deltaMinutes = Math.round((deltaY / HOUR_HEIGHT) * 60)
        const snappedDelta = Math.round(deltaMinutes / 15) * 15
        let newStartMin = origStartMin + snappedDelta
        // Clamp to grid bounds
        newStartMin = Math.max(START_HOUR * 60, Math.min(END_HOUR * 60 - duration, newStartMin))
        const newEndMin = newStartMin + duration
        const newStartTime = minutesToTime(newStartMin)
        const newEndTime = minutesToTime(newEndMin)
        onUpdateEvent(dragState.eventId, { startTime: newStartTime, endTime: newEndTime })
        toast.success(t('kalender.event.moved', { time: newStartTime }))
      } else {
        // resize — change end time
        const deltaMinutes = Math.round((deltaY / HOUR_HEIGHT) * 60)
        const snappedDelta = Math.round(deltaMinutes / 15) * 15
        let newEndMin = origEndMin + snappedDelta
        newEndMin = Math.max(origStartMin + 15, Math.min(END_HOUR * 60, newEndMin))
        const newEndTime = minutesToTime(newEndMin)
        onUpdateEvent(dragState.eventId, { endTime: newEndTime })
        toast.success(t('kalender.event.resized', { time: newEndTime }))
      }
      setDragState(null)
    }

    window.addEventListener('mousemove', handleMouseMove)
    window.addEventListener('mouseup', handleMouseUp)
    return () => {
      window.removeEventListener('mousemove', handleMouseMove)
      window.removeEventListener('mouseup', handleMouseUp)
    }
  }, [dragState, onUpdateEvent])

  // Compute ghost position during drag
  const getGhostPosition = useCallback((event: CalendarEvent) => {
    if (!dragState || dragState.eventId !== event.id) return null
    const deltaY = dragState.currentY - dragState.startY
    const origStartMin = timeToMinutes(dragState.originalStartTime)
    const origEndMin = timeToMinutes(dragState.originalEndTime)
    const duration = origEndMin - origStartMin

    if (dragState.mode === 'move') {
      const deltaMinutes = Math.round((deltaY / HOUR_HEIGHT) * 60)
      const snappedDelta = Math.round(deltaMinutes / 15) * 15
      let newStartMin = origStartMin + snappedDelta
      newStartMin = Math.max(START_HOUR * 60, Math.min(END_HOUR * 60 - duration, newStartMin))
      const newEndMin = newStartMin + duration
      const top = ((newStartMin - START_HOUR * 60) / 60) * HOUR_HEIGHT
      const height = Math.max(((newEndMin - newStartMin) / 60) * HOUR_HEIGHT, 22)
      return { top, height, startTime: minutesToTime(newStartMin), endTime: minutesToTime(newEndMin) }
    } else {
      const deltaMinutes = Math.round((deltaY / HOUR_HEIGHT) * 60)
      const snappedDelta = Math.round(deltaMinutes / 15) * 15
      let newEndMin = origEndMin + snappedDelta
      newEndMin = Math.max(origStartMin + 15, Math.min(END_HOUR * 60, newEndMin))
      const top = ((origStartMin - START_HOUR * 60) / 60) * HOUR_HEIGHT
      const height = Math.max(((newEndMin - origStartMin) / 60) * HOUR_HEIGHT, 22)
      return { top, height, startTime: dragState.originalStartTime, endTime: minutesToTime(newEndMin) }
    }
  }, [dragState])

  return (
    <div className={`flex flex-col min-h-full ${dragState ? 'select-none' : ''}`}>
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
              <p className="text-xs font-medium uppercase tracking-wider text-muted-foreground">{DAYS_SHORT[i]}</p>
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
            <span className="text-[9px] text-muted-foreground">{t('kalender.view.allDay')}</span>
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
      <div ref={gridRef} className="grid grid-cols-[56px_repeat(5,1fr)] flex-1">
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
          const dateKey = formatDateKey(d)

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
              {isToday(d) && nowMinutes >= START_HOUR * 60 && nowMinutes <= END_HOUR * 60 && (
                <div
                  className="absolute left-0 right-0 z-10 flex items-center"
                  style={{ top: ((nowMinutes - START_HOUR * 60) / 60) * HOUR_HEIGHT }}
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
                const isDragging = dragState?.eventId === event.id
                const ghost = getGhostPosition(event)

                return (
                  <div key={event.id}>
                    {/* Original position (faded during drag) */}
                    <div
                      onMouseDown={(e) => handleEventMouseDown(e, event, dateKey, 'move')}
                      onClick={(e) => {
                        if (isDragging) return
                        e.stopPropagation()
                        onSelectEvent(event)
                      }}
                      className={`absolute rounded-[4px] px-1.5 py-0.5 text-left overflow-hidden transition-all z-[5] ${
                        isDragging ? 'opacity-30 pointer-events-none' : 'hover:brightness-95 hover:shadow-sm cursor-grab'
                      }`}
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
                      {/* 10.12: Video call badge */}
                      {event.videoCall && (
                        <span className="absolute top-0.5 right-0.5 flex h-3.5 w-3.5 items-center justify-center rounded-full" style={{ backgroundColor: color }}>
                          <Video className="h-2 w-2 text-white" />
                        </span>
                      )}
                      {/* 10.13: Reminder bell badge */}
                      {event.reminder && event.reminder !== 'Keine' && (
                        <span className="absolute top-0.5 right-[18px] flex h-3.5 w-3.5 items-center justify-center rounded-full" style={{ backgroundColor: `${color}40` }}>
                          <Bell className="h-2 w-2" style={{ color }} />
                        </span>
                      )}
                      <p className="text-[10px] font-medium truncate pr-5" style={{ color }}>
                        {event.title}
                      </p>
                      {height > 30 && (
                        <p className="text-[9px] truncate" style={{ color, opacity: 0.7 }}>
                          {event.startTime} – {event.endTime}
                        </p>
                      )}
                      {/* Resize handle */}
                      {!event.isTaskDeadline && height > 24 && (
                        <div
                          onMouseDown={(e) => {
                            e.stopPropagation()
                            handleEventMouseDown(e, event, dateKey, 'resize')
                          }}
                          className="absolute bottom-0 left-0 right-0 h-2 cursor-s-resize flex justify-center items-center group"
                        >
                          <GripVertical className="h-2 w-2 text-transparent group-hover:text-current opacity-50" style={{ color }} />
                        </div>
                      )}
                    </div>

                    {/* Ghost event at new position during drag */}
                    {isDragging && ghost && (
                      <div
                        className="absolute rounded-[4px] px-1.5 py-0.5 text-left overflow-hidden z-[15] pointer-events-none shadow-md cursor-grabbing"
                        style={{
                          top: ghost.top,
                          height: ghost.height,
                          left: `calc(${leftPct}% + 2px)`,
                          width: `calc(${widthPct}% - 2px)`,
                          backgroundColor: `${color}30`,
                          borderLeft: `3px solid ${color}`,
                        }}
                      >
                        <p className="text-[10px] font-medium truncate" style={{ color }}>
                          {event.title}
                        </p>
                        {ghost.height > 30 && (
                          <p className="text-[9px] truncate" style={{ color, opacity: 0.7 }}>
                            {ghost.startTime} – {ghost.endTime}
                          </p>
                        )}
                      </div>
                    )}
                  </div>
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
  onUpdateEvent,
}: {
  currentDate: Date
  events: CalendarEvent[]
  calendars: CalendarSource[]
  onSelectEvent: (e: CalendarEvent) => void
  onSlotClick: (date: string, hour: number, minute: number, e: React.MouseEvent) => void
  onUpdateEvent: (eventId: string, updates: Partial<CalendarEvent>) => void
}) {
  const { t } = useTranslation()
  const { startHour: START_HOUR, endHour: END_HOUR, hours: HOURS } = useWorkHours()
  const nowMinutes = useNowMinutes()
  const allDay = events.filter((e) => e.isAllDay)
  const timed = events.filter((e) => !e.isAllDay)
  const layouts = layoutOverlappingEvents(timed)
  const dateKey = formatDateKey(currentDate)
  const [dragState, setDragState] = useState<DragState | null>(null)

  // 10.11: Drag-and-drop handlers (Day view)
  const handleEventMouseDown = useCallback((
    e: React.MouseEvent,
    event: CalendarEvent,
    mode: 'move' | 'resize',
  ) => {
    e.preventDefault()
    e.stopPropagation()
    const startMin = timeToMinutes(event.startTime)
    setDragState({
      eventId: event.id,
      startY: e.clientY,
      startHour: Math.floor(startMin / 60),
      startMinute: startMin % 60,
      currentY: e.clientY,
      mode,
      dayDate: dateKey,
      originalStartTime: event.startTime,
      originalEndTime: event.endTime,
    })
  }, [dateKey])

  useEffect(() => {
    if (!dragState) return
    const handleMouseMove = (e: MouseEvent) => {
      setDragState((prev) => prev ? { ...prev, currentY: e.clientY } : null)
    }
    const handleMouseUp = () => {
      if (!dragState) return
      const deltaY = dragState.currentY - dragState.startY
      const origStartMin = timeToMinutes(dragState.originalStartTime)
      const origEndMin = timeToMinutes(dragState.originalEndTime)
      const duration = origEndMin - origStartMin

      if (dragState.mode === 'move') {
        const deltaMinutes = Math.round((deltaY / HOUR_HEIGHT) * 60)
        const snappedDelta = Math.round(deltaMinutes / 15) * 15
        let newStartMin = origStartMin + snappedDelta
        newStartMin = Math.max(START_HOUR * 60, Math.min(END_HOUR * 60 - duration, newStartMin))
        const newEndMin = newStartMin + duration
        const newStartTime = minutesToTime(newStartMin)
        const newEndTime = minutesToTime(newEndMin)
        onUpdateEvent(dragState.eventId, { startTime: newStartTime, endTime: newEndTime })
        toast.success(t('kalender.event.moved', { time: newStartTime }))
      } else {
        const deltaMinutes = Math.round((deltaY / HOUR_HEIGHT) * 60)
        const snappedDelta = Math.round(deltaMinutes / 15) * 15
        let newEndMin = origEndMin + snappedDelta
        newEndMin = Math.max(origStartMin + 15, Math.min(END_HOUR * 60, newEndMin))
        const newEndTime = minutesToTime(newEndMin)
        onUpdateEvent(dragState.eventId, { endTime: newEndTime })
        toast.success(t('kalender.event.resized', { time: newEndTime }))
      }
      setDragState(null)
    }
    window.addEventListener('mousemove', handleMouseMove)
    window.addEventListener('mouseup', handleMouseUp)
    return () => {
      window.removeEventListener('mousemove', handleMouseMove)
      window.removeEventListener('mouseup', handleMouseUp)
    }
  }, [dragState, onUpdateEvent])

  const getGhostPosition = useCallback((event: CalendarEvent) => {
    if (!dragState || dragState.eventId !== event.id) return null
    const deltaY = dragState.currentY - dragState.startY
    const origStartMin = timeToMinutes(dragState.originalStartTime)
    const origEndMin = timeToMinutes(dragState.originalEndTime)
    const duration = origEndMin - origStartMin

    if (dragState.mode === 'move') {
      const deltaMinutes = Math.round((deltaY / HOUR_HEIGHT) * 60)
      const snappedDelta = Math.round(deltaMinutes / 15) * 15
      let newStartMin = origStartMin + snappedDelta
      newStartMin = Math.max(START_HOUR * 60, Math.min(END_HOUR * 60 - duration, newStartMin))
      const newEndMin = newStartMin + duration
      const top = ((newStartMin - START_HOUR * 60) / 60) * HOUR_HEIGHT
      const height = Math.max(((newEndMin - newStartMin) / 60) * HOUR_HEIGHT, 24)
      return { top, height, startTime: minutesToTime(newStartMin), endTime: minutesToTime(newEndMin) }
    } else {
      const deltaMinutes = Math.round((deltaY / HOUR_HEIGHT) * 60)
      const snappedDelta = Math.round(deltaMinutes / 15) * 15
      let newEndMin = origEndMin + snappedDelta
      newEndMin = Math.max(origStartMin + 15, Math.min(END_HOUR * 60, newEndMin))
      const top = ((origStartMin - START_HOUR * 60) / 60) * HOUR_HEIGHT
      const height = Math.max(((newEndMin - origStartMin) / 60) * HOUR_HEIGHT, 24)
      return { top, height, startTime: dragState.originalStartTime, endTime: minutesToTime(newEndMin) }
    }
  }, [dragState])

  return (
    <div className={`flex flex-col min-h-full ${dragState ? 'select-none' : ''}`}>
      {/* All-day events */}
      {allDay.length > 0 && (
        <div className="border-b border-border px-4 py-2">
          <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1">{t('kalender.view.allDay')}</p>
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

          {/* Current time indicator */}
          {isToday(currentDate) && nowMinutes >= START_HOUR * 60 && nowMinutes <= END_HOUR * 60 && (
            <div
              className="absolute left-0 right-0 z-10 flex items-center"
              style={{ top: ((nowMinutes - START_HOUR * 60) / 60) * HOUR_HEIGHT }}
            >
              <div className="h-2.5 w-2.5 -ml-1 rounded-full bg-error" />
              <div className="flex-1 h-[2px] bg-error" />
            </div>
          )}

          {layouts.map(({ event, column, totalColumns }) => {
            const startMin = timeToMinutes(event.startTime)
            const endMin = timeToMinutes(event.endTime)
            const top = ((startMin - START_HOUR * 60) / 60) * HOUR_HEIGHT
            const height = Math.max(((endMin - startMin) / 60) * HOUR_HEIGHT, 24)
            const color = getCategoryColor(event, calendars)
            const leftPct = (column / totalColumns) * 100
            const widthPct = (1 / totalColumns) * 100 - 2
            const isDragging = dragState?.eventId === event.id
            const ghost = getGhostPosition(event)

            return (
              <div key={event.id}>
                {/* Original (faded during drag) */}
                <div
                  onMouseDown={(e) => handleEventMouseDown(e, event, 'move')}
                  onClick={(e) => {
                    if (isDragging) return
                    e.stopPropagation()
                    onSelectEvent(event)
                  }}
                  className={`absolute rounded-md px-3 py-1.5 text-left overflow-hidden transition-all z-[5] ${
                    isDragging ? 'opacity-30 pointer-events-none' : 'hover:brightness-95 hover:shadow-sm cursor-grab'
                  }`}
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
                  {/* 10.12: Video call badge */}
                  {event.videoCall && (
                    <span className="absolute top-1 right-1 flex h-4 w-4 items-center justify-center rounded-full" style={{ backgroundColor: color }}>
                      <Video className="h-2.5 w-2.5 text-white" />
                    </span>
                  )}
                  {/* 10.13: Reminder bell badge */}
                  {event.reminder && event.reminder !== 'Keine' && (
                    <span className="absolute top-1 right-6 flex h-4 w-4 items-center justify-center rounded-full" style={{ backgroundColor: `${color}40` }}>
                      <Bell className="h-2.5 w-2.5" style={{ color }} />
                    </span>
                  )}
                  <p className="text-xs font-medium truncate pr-7" style={{ color }}>{event.title}</p>
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
                  {/* Resize handle */}
                  {!event.isTaskDeadline && height > 28 && (
                    <div
                      onMouseDown={(e) => {
                        e.stopPropagation()
                        handleEventMouseDown(e, event, 'resize')
                      }}
                      className="absolute bottom-0 left-0 right-0 h-2.5 cursor-s-resize flex justify-center items-center group"
                    >
                      <GripVertical className="h-2.5 w-2.5 text-transparent group-hover:text-current opacity-50" style={{ color }} />
                    </div>
                  )}
                </div>

                {/* Ghost event during drag */}
                {isDragging && ghost && (
                  <div
                    className="absolute rounded-md px-3 py-1.5 text-left overflow-hidden z-[15] pointer-events-none shadow-md"
                    style={{
                      top: ghost.top,
                      height: ghost.height,
                      left: `calc(${leftPct}% + 4px)`,
                      width: `calc(${widthPct}% - 4px)`,
                      backgroundColor: `${color}30`,
                      borderLeft: `4px solid ${color}`,
                    }}
                  >
                    <p className="text-xs font-medium truncate" style={{ color }}>{event.title}</p>
                    <p className="text-[10px] mt-0.5" style={{ color, opacity: 0.7 }}>
                      {ghost.startTime} – {ghost.endTime}
                    </p>
                  </div>
                )}
              </div>
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
  const { t } = useTranslation()
  const days = getMonthDays(currentDate.getFullYear(), currentDate.getMonth())

  return (
    <div className="h-full flex flex-col">
      <div className="grid grid-cols-7 border-b border-border">
        {DAYS_SHORT.map((day) => (
          <div key={day} className="px-2 py-2 text-center text-xs uppercase tracking-wider font-medium text-muted-foreground">
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
                    {t('kalender.view.moreEvents', { count: events.length - 3 })}
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
  categories,
  onClose,
  onSave,
  onMoreOptions,
}: {
  state: QuickCreateState
  categories: EventCategory[]
  onClose: () => void
  onSave: (event: Partial<CalendarEvent>) => void
  onMoreOptions: () => void
}) {
  const { t } = useTranslation()
  const [title, setTitle] = useState('')
  const [categoryId, setCategoryId] = useState(categories[0]?.id ?? '')
  const startTime = `${String(state.hour).padStart(2, '0')}:${String(state.minute).padStart(2, '0')}`
  const endTime = `${String(state.hour + 1).padStart(2, '0')}:${String(state.minute).padStart(2, '0')}`
  const timeLabel = `${startTime} – ${endTime}`

  // Position near click, clamped to viewport
  const top = Math.min(state.y, window.innerHeight - 280)
  const left = Math.min(state.x, window.innerWidth - 320)

  const handleSave = () => {
    if (!title.trim()) return
    onSave({
      title: title.trim(),
      date: state.date,
      startTime,
      endTime,
      categoryId,
      isAllDay: false,
    })
  }

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
            placeholder={t('kalender.quickCreate.titlePlaceholder')}
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            onKeyDown={(e) => { if (e.key === 'Enter') handleSave() }}
            className="w-full bg-transparent text-sm text-foreground placeholder:text-muted-foreground outline-none border-b border-border-muted pb-2"
          />

          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <Clock className="h-3.5 w-3.5" />
            <span>{timeLabel}</span>
          </div>

          <div className="flex items-center gap-2">
            <span className="text-[10px] text-muted-foreground">{t('kalender.quickCreate.category')}</span>
            <div className="flex gap-1">
              {categories.map((cat) => (
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
              onClick={handleSave}
              disabled={!title.trim()}
              className="flex-1 rounded-lg bg-primary px-3 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50"
            >
              {t('kalender.quickCreate.save')}
            </button>
            <button
              onClick={onMoreOptions}
              className="flex-1 rounded-lg border border-border px-3 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors"
            >
              {t('kalender.quickCreate.moreOptions')}
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
  categories,
  calendars,
  isSaving,
  onSave,
  onClose,
}: {
  defaults: Partial<CalendarEvent>
  categories: EventCategory[]
  calendars: CalendarSource[]
  isSaving: boolean
  onSave: (event: Partial<CalendarEvent>) => void
  onClose: () => void
}) {
  const { t } = useTranslation()
  const defaultCalendar = defaults.calendarId || calendars.find((c) => c.group === 'mine')?.id || ''
  const [title, setTitle] = useState(defaults.title ?? '')
  const [date, setDate] = useState(defaults.date ?? formatDateKey(new Date()))
  const [startTime, setStartTime] = useState(defaults.startTime ?? '09:00')
  const [endTime, setEndTime] = useState(defaults.endTime ?? '10:00')
  const [isAllDay, setIsAllDay] = useState(defaults.isAllDay ?? false)
  const [categoryId, setCategoryId] = useState(defaults.categoryId ?? categories[0]?.id ?? '')
  const [location, setLocation] = useState(defaults.location ?? '')
  const [room, setRoom] = useState(defaults.room ?? '')
  const [description, setDescription] = useState(defaults.description ?? '')
  const [recurrence, setRecurrence] = useState(defaults.recurrence ?? 'Keine')
  const [reminder, setReminder] = useState(defaults.reminder ?? '15 Minuten')
  const [calendarId, setCalendarId] = useState(defaultCalendar)
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
            {defaults.title ? t('kalender.form.editEvent') : t('kalender.form.newEvent')}
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
              placeholder={t('kalender.form.titlePlaceholder')}
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              className="w-full rounded-lg border border-input-border bg-input-background px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder outline-none focus:border-primary"
            />
          </div>

          {/* Date & Time */}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1 block">{t('kalender.form.date')}</label>
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
                    <label className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1 block">{t('kalender.form.from')}</label>
                    <input
                      type="time"
                      value={startTime}
                      onChange={(e) => setStartTime(e.target.value)}
                      className="w-full rounded-lg border border-input-border bg-input-background px-2 py-1.5 text-xs text-foreground outline-none focus:border-primary"
                    />
                  </div>
                  <div>
                    <label className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1 block">{t('kalender.form.to')}</label>
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
            <span className="text-xs text-foreground">{t('kalender.form.allDay')}</span>
          </label>

          {/* Category */}
          <div>
            <label className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1.5 block">{t('kalender.form.category')}</label>
            <div className="flex flex-wrap gap-1.5">
              {categories.map((cat) => (
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
            <label className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1 block">{t('kalender.form.location')}</label>
            <div className="flex items-center gap-2 rounded-lg border border-input-border bg-input-background px-3 py-1.5">
              <MapPin className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
              <input
                type="text"
                placeholder={t('kalender.form.locationPlaceholder')}
                value={location}
                onChange={(e) => setLocation(e.target.value)}
                className="flex-1 bg-transparent text-xs text-foreground placeholder:text-input-placeholder outline-none"
              />
            </div>
          </div>

          {/* Room */}
          <div>
            <label className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1 block">{t('kalender.form.room')}</label>
            <select
              value={room}
              onChange={(e) => setRoom(e.target.value)}
              className="w-full rounded-lg border border-input-border bg-input-background px-3 py-1.5 text-xs text-foreground outline-none focus:border-primary"
            >
              <option value="">{t('kalender.form.noRoom')}</option>
              {ROOMS.map((r) => (
                <option key={r.id} value={r.name}>
                  {r.name} ({r.capacity} Pl.)
                </option>
              ))}
            </select>
          </div>

          {/* Description */}
          <div>
            <label className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1 block">{t('kalender.form.description')}</label>
            <textarea
              rows={3}
              placeholder={t('kalender.form.descriptionPlaceholder')}
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
                {t('kalender.form.recurrence')}
              </label>
              <select
                value={recurrence}
                onChange={(e) => setRecurrence(e.target.value)}
                className="w-full rounded-lg border border-input-border bg-input-background px-3 py-1.5 text-xs text-foreground outline-none focus:border-primary"
              >
                {RECURRENCE_OPTIONS.map((opt) => (
                  <option key={opt} value={opt}>{t(`kalender.form.recurrence.${opt}`)}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1 block">
                <Bell className="h-3 w-3 inline mr-1" />
                {t('kalender.form.reminderLabel')}
              </label>
              <select
                value={reminder}
                onChange={(e) => setReminder(e.target.value)}
                className="w-full rounded-lg border border-input-border bg-input-background px-3 py-1.5 text-xs text-foreground outline-none focus:border-primary"
              >
                {REMINDER_OPTIONS.map((opt) => (
                  <option key={opt} value={opt}>{t(`kalender.form.reminder.${opt}`)}</option>
                ))}
              </select>
            </div>
          </div>

          {/* Calendar */}
          <div>
            <label className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1 block">
              <Calendar className="h-3 w-3 inline mr-1" />
              {t('kalender.form.calendarLabel')}
            </label>
            <select
              value={calendarId}
              onChange={(e) => setCalendarId(e.target.value)}
              className="w-full rounded-lg border border-input-border bg-input-background px-3 py-1.5 text-xs text-foreground outline-none focus:border-primary"
            >
              {calendars.filter((c) => c.group !== 'other').map((c) => (
                <option key={c.id} value={c.id}>{c.name}</option>
              ))}
            </select>
          </div>

          {/* Participants */}
          <div>
            <label className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1 block">
              <Users className="h-3 w-3 inline mr-1" />
              {t('kalender.form.inviteParticipants')}
            </label>
            <div className="relative">
              <div className="flex items-center gap-2 rounded-lg border border-input-border bg-input-background px-3 py-1.5">
                <Search className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                <input
                  type="text"
                  placeholder={t('kalender.form.participantPlaceholder')}
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
            <span className="text-xs text-foreground">{t('kalender.form.videoCall')}</span>
          </label>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-2 border-t border-border px-5 py-3">
          <button
            onClick={onClose}
            className="rounded-lg border border-border px-4 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors"
          >
            {t('common.cancel')}
          </button>
          <button
            disabled={isSaving || !title.trim()}
            onClick={() => onSave({
              ...(defaults.id ? { id: defaults.id } : {}),
              title,
              date,
              startTime,
              endTime,
              isAllDay,
              categoryId,
              calendarId,
              location: location || undefined,
              room: room || undefined,
              description: description || undefined,
              recurrence,
              reminder,
              videoCall,
            })}
            className="rounded-lg bg-primary px-4 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50"
          >
            {isSaving ? t('kalender.form.saving') : t('common.save')}
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
  categories,
  onClose,
  onEdit,
  onDelete,
}: {
  event: CalendarEvent
  calendars: CalendarSource[]
  categories: EventCategory[]
  onClose: () => void
  onEdit: () => void
  onDelete: () => void
}) {
  const { t } = useTranslation()
  const color = getCategoryColor(event, calendars)
  const category = categories.find((c) => c.id === event.categoryId)
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
              {!event.isHoliday && !event.isTaskDeadline && (
                <>
                  <button onClick={onEdit} className="rounded-md p-1 text-muted-foreground hover:bg-secondary text-xs">
                    {t('kalender.detail.edit')}
                  </button>
                  <button onClick={onDelete} className="rounded-md p-1 text-muted-foreground hover:bg-secondary hover:text-error text-xs">
                    {t('common.delete')}
                  </button>
                </>
              )}
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
                <span>{t('kalender.detail.allDay')}</span>
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
              <div className="space-y-2">
                <div className="flex items-center gap-2 text-muted-foreground">
                  <Video className="h-3.5 w-3.5 shrink-0" />
                  <button className="text-primary hover:underline">{t('kalender.detail.joinVideoCall')}</button>
                </div>
                <button
                  onClick={() => toast.success(t('kalender.detail.startMeetingToast'))}
                  className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-xs font-medium text-primary-foreground hover:bg-button-primary-hover transition-colors w-full justify-center"
                >
                  <Video className="h-3.5 w-3.5" />
                  {t('kalender.detail.startMeeting')}
                </button>
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
                <span className="text-[10px] font-medium">{t('kalender.detail.taskDeadline')}</span>
              </div>
            )}

            {/* Participants & RSVP */}
            {event.participants && event.participants.length > 0 && (
              <div>
                <div className="flex items-center gap-2 text-muted-foreground mb-2">
                  <Users className="h-3.5 w-3.5 shrink-0" />
                  <span>{t('kalender.detail.participants', { count: event.participants.length })}</span>
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

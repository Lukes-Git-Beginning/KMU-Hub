/**
 * Adapter layer for transforming backend API types to Darien's UI types.
 *
 * The calendar UI uses local interface shapes (CalendarEvent, CalendarSource, etc.)
 * that differ from the backend API types (ExpandedEvent, Calendar, etc.).
 * These adapters handle the transformation in both directions.
 */
import type {
  ExpandedEvent,
  CalendarWithMemberInfo,
  EventCategory as BackendEventCategory,
  EventAttendee,
  Resource,
  ResourceBooking,
  PublicHoliday,
  TaskDeadlineStub,
  CreateEventRequest,
  UpdateEventRequest,
} from '@/api/calendar-types'

// ---------------------------------------------------------------------------
// UI types (matching Darien's KalenderPage interfaces)
// ---------------------------------------------------------------------------

export type ViewMode = 'week' | 'day' | 'month'
export type RSVPStatus = 'accepted' | 'declined' | 'maybe' | 'pending'

export interface UIEventCategory {
  id: string
  name: string
  color: string
}

export interface CalendarSource {
  id: string
  name: string
  group: 'mine' | 'shared' | 'other'
  color: string
  visible: boolean
}

export interface Participant {
  name: string
  initials: string
  rsvp: RSVPStatus
}

export interface CalendarEvent {
  id: string
  title: string
  date: string
  startTime: string
  endTime: string
  isAllDay: boolean
  categoryId: string
  calendarId: string
  location?: string
  room?: string
  description?: string
  recurrence?: string
  reminder?: string
  videoCall?: boolean
  participants?: Participant[]
  isTaskDeadline?: boolean
  isHoliday?: boolean
}

export interface Room {
  id: string
  name: string
  capacity: number
  tags: string[]
}

export interface Booking {
  id: string
  title: string
  room: string
  date: string
  startTime: string
  endTime: string
  organizer: string
  participants: number
}

// ---------------------------------------------------------------------------
// Backend → UI transformations
// ---------------------------------------------------------------------------

/** Convert ISO datetime string to HH:mm. */
function isoToTime(iso: string): string {
  const d = new Date(iso)
  return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
}

/** Convert ISO datetime string to YYYY-MM-DD. */
function isoToDate(iso: string): string {
  const d = new Date(iso)
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

/** Map backend RSVP status to UI status. */
function mapRSVPStatus(status: string): RSVPStatus {
  switch (status) {
    case 'accepted': return 'accepted'
    case 'declined': return 'declined'
    case 'tentative': return 'maybe'
    default: return 'pending'
  }
}

/** Get initials from first + last name. */
function getInitials(firstName: string, lastName: string): string {
  return `${firstName.charAt(0)}${lastName.charAt(0)}`.toUpperCase()
}

/** Map RRULE string to German display label. */
function rruleToDisplay(rrule: string | null): string | undefined {
  if (!rrule) return undefined
  const upper = rrule.toUpperCase()
  if (upper.includes('FREQ=DAILY')) return 'Täglich'
  if (upper.includes('FREQ=WEEKLY')) return 'Wöchentlich'
  if (upper.includes('FREQ=MONTHLY')) return 'Monatlich'
  if (upper.includes('FREQ=YEARLY')) return 'Jaehrlich'
  return 'Benutzerdefiniert...'
}

/**
 * Transform a backend ExpandedEvent to Darien's CalendarEvent UI type.
 */
export function expandedEventToUI(event: ExpandedEvent): CalendarEvent {
  return {
    id: event.id,
    title: event.title,
    date: isoToDate(event.start_time),
    startTime: event.is_all_day ? '00:00' : isoToTime(event.start_time),
    endTime: event.is_all_day ? '23:59' : isoToTime(event.end_time),
    isAllDay: event.is_all_day,
    categoryId: event.category_id || '',
    calendarId: event.calendar_id,
    location: event.location || undefined,
    room: event.resource_name || undefined,
    description: event.description || undefined,
    recurrence: rruleToDisplay(event.rrule),
    videoCall: event.has_video_call,
  }
}

/**
 * Transform a CalendarWithMemberInfo to a CalendarSource for the sidebar.
 */
export function calendarToUI(
  cal: CalendarWithMemberInfo,
  currentUserId: string,
  visibleCalendarIds: Set<string>,
): CalendarSource {
  const group: CalendarSource['group'] =
    cal.owner_id === currentUserId ? 'mine' : 'shared'

  return {
    id: cal.id,
    name: cal.name,
    group,
    color: cal.color_override || cal.color,
    visible: visibleCalendarIds.has(cal.id),
  }
}

/**
 * Transform backend EventCategory to UI EventCategory.
 */
export function categoryToUI(cat: BackendEventCategory): UIEventCategory {
  return {
    id: cat.id,
    name: cat.name,
    color: cat.color,
  }
}

/**
 * Transform a backend EventAttendee to a Participant.
 */
export function attendeeToUI(attendee: EventAttendee): Participant {
  return {
    name: `${attendee.user_first_name} ${attendee.user_last_name}`,
    initials: getInitials(attendee.user_first_name, attendee.user_last_name),
    rsvp: mapRSVPStatus(attendee.rsvp_status),
  }
}

/**
 * Transform a backend Resource to a Room.
 */
export function resourceToUI(resource: Resource): Room {
  return {
    id: resource.id,
    name: resource.name,
    capacity: resource.capacity || 0,
    tags: resource.tags || [],
  }
}

/**
 * Transform a backend ResourceBooking to a Booking.
 */
export function bookingToUI(booking: ResourceBooking): Booking {
  return {
    id: booking.id,
    title: booking.event_title,
    room: booking.resource_name,
    date: isoToDate(booking.start_time),
    startTime: isoToTime(booking.start_time),
    endTime: isoToTime(booking.end_time),
    organizer: booking.booked_by,
    participants: 0,
  }
}

/**
 * Transform a PublicHoliday into a CalendarEvent for display.
 */
export function holidayToUI(holiday: PublicHoliday): CalendarEvent {
  return {
    id: `holiday-${holiday.id}`,
    title: holiday.local_name,
    date: holiday.date,
    startTime: '00:00',
    endTime: '23:59',
    isAllDay: true,
    categoryId: '',
    calendarId: 'holidays',
    isHoliday: true,
  }
}

/**
 * Transform a TaskDeadlineStub into a CalendarEvent for display.
 */
export function deadlineToUI(deadline: TaskDeadlineStub): CalendarEvent {
  return {
    id: `deadline-${deadline.task_id}`,
    title: deadline.project_key
      ? `${deadline.project_key}: ${deadline.title}`
      : deadline.title,
    date: deadline.due_date,
    startTime: '18:00',
    endTime: '18:30',
    isAllDay: false,
    categoryId: '',
    calendarId: 'deadlines',
    isTaskDeadline: true,
  }
}

// ---------------------------------------------------------------------------
// UI → Backend transformations
// ---------------------------------------------------------------------------

/**
 * Convert German recurrence label back to RRULE.
 */
function displayToRrule(display: string | undefined): string | undefined {
  if (!display || display === 'Keine') return undefined
  switch (display) {
    case 'Täglich': return 'FREQ=DAILY'
    case 'Wöchentlich': return 'FREQ=WEEKLY'
    case 'Monatlich': return 'FREQ=MONTHLY'
    case 'Jaehrlich': return 'FREQ=YEARLY'
    default: return undefined
  }
}

/**
 * Convert UI CalendarEvent data to a backend CreateEventRequest.
 */
export function uiEventToCreateRequest(
  event: Partial<CalendarEvent>,
  calendarId: string,
  timezone: string = 'Europe/Zurich',
): CreateEventRequest {
  const startIso = `${event.date}T${event.startTime || '09:00'}:00`
  const endIso = `${event.date}T${event.endTime || '10:00'}:00`

  return {
    calendar_id: calendarId,
    title: event.title || 'Neuer Termin',
    description: event.description,
    location: event.location,
    start_time: new Date(startIso).toISOString(),
    end_time: new Date(endIso).toISOString(),
    is_all_day: event.isAllDay,
    timezone,
    rrule: displayToRrule(event.recurrence),
    has_video_call: event.videoCall,
    category_id: event.categoryId || undefined,
  }
}

/**
 * Convert UI CalendarEvent data to a backend UpdateEventRequest.
 */
export function uiEventToUpdateRequest(
  event: Partial<CalendarEvent>,
): UpdateEventRequest {
  const result: UpdateEventRequest = {}

  if (event.title !== undefined) result.title = event.title
  if (event.description !== undefined) result.description = event.description
  if (event.location !== undefined) result.location = event.location
  if (event.isAllDay !== undefined) result.is_all_day = event.isAllDay
  if (event.videoCall !== undefined) result.has_video_call = event.videoCall
  if (event.categoryId !== undefined) result.category_id = event.categoryId || null
  if (event.recurrence !== undefined) result.rrule = displayToRrule(event.recurrence) || null

  if (event.date && event.startTime) {
    result.start_time = new Date(`${event.date}T${event.startTime}:00`).toISOString()
  }
  if (event.date && event.endTime) {
    result.end_time = new Date(`${event.date}T${event.endTime}:00`).toISOString()
  }

  return result
}

// ---------------------------------------------------------------------------
// Synthetic calendar sources for holidays and task deadlines
// ---------------------------------------------------------------------------

export const HOLIDAY_CALENDAR: CalendarSource = {
  id: 'holidays',
  name: 'Feiertage CH',
  group: 'other',
  color: '#9d8f85',
  visible: true,
}

export const DEADLINE_CALENDAR: CalendarSource = {
  id: 'deadlines',
  name: 'Task-Deadlines',
  group: 'other',
  color: '#a13f3f',
  visible: true,
}

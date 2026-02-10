/**
 * Calendar module TypeScript types.
 *
 * Manually defined types for the Calendar API entities. These types mirror
 * the backend calendar service models and will be replaced by auto-generated
 * types once the calendar OpenAPI spec is added to openapi.yaml.
 */

// ---------------------------------------------------------------------------
// Enums / union types
// ---------------------------------------------------------------------------

export type CalendarType = 'personal' | 'shared' | 'resource'
export type CalendarPermission = 'view' | 'edit' | 'admin'
export type RSVPStatus = 'pending' | 'accepted' | 'declined' | 'tentative'
export type RecurringEditScope = 'this' | 'this_and_future' | 'all'
export type ResourceType = 'room' | 'equipment' | 'vehicle'

// ---------------------------------------------------------------------------
// Calendar entities
// ---------------------------------------------------------------------------

export interface Calendar {
  id: string
  name: string
  description: string
  calendar_type: CalendarType
  color: string
  owner_id: string
  is_default: boolean
  timezone: string
  created_at: string
  updated_at: string
}

export interface CalendarWithMemberInfo extends Calendar {
  permission: CalendarPermission
  color_override: string | null
  is_visible: boolean
}

export interface CalendarMember {
  calendar_id: string
  user_id: string
  permission: CalendarPermission
  color_override: string | null
  is_visible: boolean
  created_at: string
  user_first_name: string
  user_last_name: string
}

// ---------------------------------------------------------------------------
// Event entities
// ---------------------------------------------------------------------------

export interface CalendarEvent {
  id: string
  calendar_id: string
  title: string
  description: string
  location: string
  resource_id: string | null
  start_time: string
  end_time: string
  is_all_day: boolean
  timezone: string
  rrule: string | null
  recurrence_end: string | null
  has_video_call: boolean
  livekit_room_name: string | null
  category_id: string | null
  created_by: string
  created_at: string
  updated_at: string
}

export interface ExpandedEvent extends CalendarEvent {
  original_event_id: string | null
  original_date: string | null
  calendar_name: string
  calendar_color: string
  display_color: string
  category_name: string | null
  category_color: string | null
  resource_name: string | null
  attendee_count: number
  my_rsvp: RSVPStatus | null
}

export interface EventAttendee {
  event_id: string
  user_id: string
  rsvp_status: RSVPStatus
  responded_at: string | null
  created_at: string
  user_first_name: string
  user_last_name: string
}

export interface EventException {
  id: string
  event_id: string
  original_date: string
  is_cancelled: boolean
  title: string | null
  description: string | null
  location: string | null
  start_time: string | null
  end_time: string | null
  resource_id: string | null
  created_at: string
  updated_at: string
}

export interface EventReminder {
  id: string
  event_id: string
  minutes_before: number
  created_at: string
}

export interface EventCategory {
  id: string
  user_id: string
  name: string
  color: string
  sort_order: number
  created_at: string
}

// ---------------------------------------------------------------------------
// Resource entities
// ---------------------------------------------------------------------------

export interface Resource {
  id: string
  name: string
  resource_type: ResourceType
  capacity: number | null
  floor: string | null
  location: string | null
  description: string | null
  is_active: boolean
  tags: string[]
  created_by: string
  created_at: string
  updated_at: string
}

export interface ResourceBooking {
  id: string
  resource_id: string
  event_id: string
  booked_by: string
  start_time: string
  end_time: string
  cancelled_at: string | null
  created_at: string
  resource_name: string
  event_title: string
}

// ---------------------------------------------------------------------------
// Holiday entities
// ---------------------------------------------------------------------------

export interface PublicHoliday {
  id: string
  date: string
  name: string
  local_name: string
  country_code: string
  is_global: boolean
  subdivision_codes: string[]
  holiday_type: string
  year: number
}

// ---------------------------------------------------------------------------
// Preferences & overlay
// ---------------------------------------------------------------------------

export interface CalendarPreferences {
  user_id: string
  default_view: 'day' | 'week' | 'month'
  week_days: 5 | 7
  default_reminder_minutes: number
  default_allday_reminder_minutes: number
  subdivision_code: string | null
  show_task_deadlines: boolean
}

export interface TaskDeadlineStub {
  task_id: string
  title: string
  due_date: string
  project_key: string | null
  priority: string
}

// ---------------------------------------------------------------------------
// Request types
// ---------------------------------------------------------------------------

export interface CreateCalendarRequest {
  name: string
  description?: string
  calendar_type: CalendarType
  color?: string
  timezone?: string
}

export interface UpdateCalendarRequest {
  name?: string
  description?: string
  color?: string
  timezone?: string
}

export interface CreateEventRequest {
  calendar_id: string
  title: string
  description?: string
  location?: string
  resource_id?: string
  start_time: string
  end_time: string
  is_all_day?: boolean
  timezone?: string
  rrule?: string
  has_video_call?: boolean
  category_id?: string
  attendee_ids?: string[]
  reminder_minutes?: number[]
}

export interface UpdateEventRequest {
  title?: string
  description?: string
  location?: string
  resource_id?: string | null
  start_time?: string
  end_time?: string
  is_all_day?: boolean
  rrule?: string | null
  has_video_call?: boolean
  category_id?: string | null
}

export interface UpdateRecurringEventRequest extends UpdateEventRequest {
  scope: RecurringEditScope
  original_date: string
}

export interface CreateResourceRequest {
  name: string
  resource_type: ResourceType
  capacity?: number
  floor?: string
  location?: string
  description?: string
  tags?: string[]
}

export interface UpdateResourceRequest {
  name?: string
  resource_type?: ResourceType
  capacity?: number | null
  floor?: string | null
  location?: string | null
  description?: string | null
  is_active?: boolean
  tags?: string[]
}

export interface BookResourceRequest {
  resource_id: string
  event_id: string
  start_time: string
  end_time: string
}

// ---------------------------------------------------------------------------
// API response wrappers (match backend JSON patterns)
// ---------------------------------------------------------------------------

export interface CalendarListResponse {
  calendars: CalendarWithMemberInfo[]
}

export interface CalendarResponse {
  calendar: Calendar
}

export interface CalendarMembersResponse {
  members: CalendarMember[]
}

export interface EventListResponse {
  events: ExpandedEvent[]
}

export interface EventResponse {
  event: CalendarEvent
}

export interface EventAttendeesResponse {
  attendees: EventAttendee[]
}

export interface ResourceListResponse {
  resources: Resource[]
}

export interface ResourceResponse {
  resource: Resource
}

export interface ResourceBookingsResponse {
  bookings: ResourceBooking[]
}

export interface HolidayListResponse {
  holidays: PublicHoliday[]
}

export interface CalendarPreferencesResponse {
  preferences: CalendarPreferences
}

export interface EventCategoriesResponse {
  categories: EventCategory[]
}

export interface TaskDeadlinesResponse {
  deadlines: TaskDeadlineStub[]
}

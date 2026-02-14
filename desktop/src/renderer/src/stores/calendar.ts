/**
 * Calendar module UI state store (Zustand with localStorage persistence).
 *
 * Manages current view, date navigation, sidebar visibility, calendar
 * toggle state, and task deadline layer. Persists view preferences
 * to localStorage; transient state (currentDate, selectedDate) always
 * starts at today on app load.
 */
import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { addDays, addWeeks, addMonths, subDays, subWeeks, subMonths } from 'date-fns'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type CalendarView = 'day' | 'week' | 'month'

interface CalendarState {
  // View state
  currentView: CalendarView
  currentDate: Date
  selectedDate: Date

  // Sidebar
  sidebarOpen: boolean

  // Calendar visibility (which calendars are toggled on)
  visibleCalendarIds: Set<string>

  // Task deadline layer
  showTaskDeadlines: boolean

  // Actions
  setCurrentView: (view: CalendarView) => void
  setCurrentDate: (date: Date) => void
  setSelectedDate: (date: Date) => void
  goToToday: () => void
  navigateForward: () => void
  navigateBackward: () => void
  toggleSidebar: () => void
  toggleCalendarVisibility: (calendarId: string) => void
  setCalendarVisible: (calendarId: string, visible: boolean) => void
  toggleTaskDeadlines: () => void
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

export const useCalendarStore = create<CalendarState>()(
  persist(
    (set, get) => ({
      // Defaults -- currentDate/selectedDate not persisted (always today)
      currentView: 'week',
      currentDate: new Date(),
      selectedDate: new Date(),
      sidebarOpen: true,
      visibleCalendarIds: new Set<string>(),
      showTaskDeadlines: true,

      setCurrentView: (view: CalendarView) => set({ currentView: view }),

      setCurrentDate: (date: Date) => set({ currentDate: date }),

      setSelectedDate: (date: Date) => set({ selectedDate: date }),

      goToToday: () => {
        const today = new Date()
        set({ currentDate: today, selectedDate: today })
      },

      navigateForward: () => {
        const { currentView, currentDate } = get()
        let next: Date
        switch (currentView) {
          case 'day':
            next = addDays(currentDate, 1)
            break
          case 'week':
            next = addWeeks(currentDate, 1)
            break
          case 'month':
            next = addMonths(currentDate, 1)
            break
        }
        set({ currentDate: next })
      },

      navigateBackward: () => {
        const { currentView, currentDate } = get()
        let prev: Date
        switch (currentView) {
          case 'day':
            prev = subDays(currentDate, 1)
            break
          case 'week':
            prev = subWeeks(currentDate, 1)
            break
          case 'month':
            prev = subMonths(currentDate, 1)
            break
        }
        set({ currentDate: prev })
      },

      toggleSidebar: () =>
        set((s) => ({ sidebarOpen: !s.sidebarOpen })),

      toggleCalendarVisibility: (calendarId: string) =>
        set((s) => {
          const next = new Set(s.visibleCalendarIds)
          if (next.has(calendarId)) {
            next.delete(calendarId)
          } else {
            next.add(calendarId)
          }
          return { visibleCalendarIds: next }
        }),

      setCalendarVisible: (calendarId: string, visible: boolean) =>
        set((s) => {
          const next = new Set(s.visibleCalendarIds)
          if (visible) {
            next.add(calendarId)
          } else {
            next.delete(calendarId)
          }
          return { visibleCalendarIds: next }
        }),

      toggleTaskDeadlines: () =>
        set((s) => ({ showTaskDeadlines: !s.showTaskDeadlines })),
    }),
    {
      name: 'kmuhub-calendar',
      // Only persist view preferences, not transient date state.
      // Serialize Set as array for JSON compatibility.
      partialize: (state) => ({
        currentView: state.currentView,
        sidebarOpen: state.sidebarOpen,
        visibleCalendarIds: state.visibleCalendarIds,
        showTaskDeadlines: state.showTaskDeadlines,
      }),
      storage: {
        getItem: (name) => {
          const raw = localStorage.getItem(name)
          if (!raw) return null
          const parsed = JSON.parse(raw)
          // Deserialize visibleCalendarIds from array back to Set
          if (parsed.state?.visibleCalendarIds) {
            parsed.state.visibleCalendarIds = new Set(
              parsed.state.visibleCalendarIds,
            )
          }
          return parsed
        },
        setItem: (name, value) => {
          // Serialize Set to array before JSON.stringify
          const serializable = {
            ...value,
            state: {
              ...value.state,
              visibleCalendarIds: value.state.visibleCalendarIds
                ? Array.from(value.state.visibleCalendarIds as Set<string>)
                : [],
            },
          }
          localStorage.setItem(name, JSON.stringify(serializable))
        },
        removeItem: (name) => localStorage.removeItem(name),
      },
    },
  ),
)

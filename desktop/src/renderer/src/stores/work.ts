/**
 * Work module UI state store (Zustand with localStorage persistence).
 *
 * Manages task panel visibility, active task selection, and running timer state
 * for the slide-over task detail panel and timer persistence across navigation.
 */
import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface WorkState {
  /** Currently selected task ID for the detail panel. */
  activeTaskId: string | null
  /** Whether the task detail side panel is open. */
  taskPanelOpen: boolean

  /** Task ID of the currently running timer (persists across navigation). */
  activeTimerTaskId: string | null
  /** ISO timestamp when the active timer was started. */
  activeTimerStartedAt: string | null
  /** Time entry ID for the active timer. */
  activeTimerEntryId: string | null

  /** Open the task detail panel for a specific task. */
  openTaskPanel: (taskId: string) => void
  /** Close the task detail panel. */
  closeTaskPanel: () => void

  /** Set the active timer state (called after successful API start). */
  setActiveTimer: (taskId: string, startedAt: string, entryId: string) => void
  /** Clear the active timer state (called after successful API stop). */
  clearActiveTimer: () => void
}

export const useWorkStore = create<WorkState>()(
  persist(
    (set) => ({
      activeTaskId: null,
      taskPanelOpen: false,

      activeTimerTaskId: null,
      activeTimerStartedAt: null,
      activeTimerEntryId: null,

      openTaskPanel: (taskId: string) =>
        set({ activeTaskId: taskId, taskPanelOpen: true }),

      closeTaskPanel: () =>
        set({ taskPanelOpen: false, activeTaskId: null }),

      setActiveTimer: (taskId: string, startedAt: string, entryId: string) =>
        set({
          activeTimerTaskId: taskId,
          activeTimerStartedAt: startedAt,
          activeTimerEntryId: entryId,
        }),

      clearActiveTimer: () =>
        set({
          activeTimerTaskId: null,
          activeTimerStartedAt: null,
          activeTimerEntryId: null,
        }),
    }),
    {
      name: 'kmuhub-work',
    }
  )
)

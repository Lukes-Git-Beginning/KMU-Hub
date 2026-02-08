/**
 * Work module UI state store (Zustand with localStorage persistence).
 *
 * Manages task panel visibility and active task selection for the
 * slide-over task detail panel used across project and my-tasks views.
 */
import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface WorkState {
  /** Currently selected task ID for the detail panel. */
  activeTaskId: string | null
  /** Whether the task detail side panel is open. */
  taskPanelOpen: boolean

  /** Open the task detail panel for a specific task. */
  openTaskPanel: (taskId: string) => void
  /** Close the task detail panel. */
  closeTaskPanel: () => void
}

export const useWorkStore = create<WorkState>()(
  persist(
    (set) => ({
      activeTaskId: null,
      taskPanelOpen: false,

      openTaskPanel: (taskId: string) =>
        set({ activeTaskId: taskId, taskPanelOpen: true }),

      closeTaskPanel: () =>
        set({ taskPanelOpen: false, activeTaskId: null }),
    }),
    {
      name: 'kmuhub-work',
    }
  )
)

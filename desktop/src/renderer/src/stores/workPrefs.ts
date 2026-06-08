import { create } from 'zustand'
import { persist } from 'zustand/middleware'

/**
 * Personal work (Projekte/Aufgaben) preferences — per-user comfort settings
 * that adapt the module to one's own workflow (personal scope, see
 * ModuleSettingsShell). Persisted locally; no tenant impact.
 */
export type WorkView = 'kanban' | 'list' | 'gantt'
export type MyTasksGroupBy = 'project' | 'priority' | 'dueDate' | 'status'
export type WorkDensity = 'comfortable' | 'compact'

interface WorkPrefsState {
  /** Default board view when opening a project. */
  defaultView: WorkView
  /** Default grouping in "Meine Aufgaben". */
  myTasksGroupBy: MyTasksGroupBy
  /** Row/card density. */
  density: WorkDensity
  /** Project pre-selected when entering the module (null = projects list). */
  defaultProjectId: string | null
  setDefaultView: (v: WorkView) => void
  setMyTasksGroupBy: (g: MyTasksGroupBy) => void
  setDensity: (d: WorkDensity) => void
  setDefaultProjectId: (id: string | null) => void
}

export const useWorkPrefsStore = create<WorkPrefsState>()(
  persist(
    (set) => ({
      defaultView: 'kanban',
      myTasksGroupBy: 'project',
      density: 'comfortable',
      defaultProjectId: null,
      setDefaultView: (defaultView) => set({ defaultView }),
      setMyTasksGroupBy: (myTasksGroupBy) => set({ myTasksGroupBy }),
      setDensity: (density) => set({ density }),
      setDefaultProjectId: (defaultProjectId) => set({ defaultProjectId }),
    }),
    { name: 'cosmi-work-prefs' },
  ),
)

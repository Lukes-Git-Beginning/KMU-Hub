import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { loadModuleSettings, saveModuleSettings } from '@/api/settings-persist'

/**
 * Personal work (Projekte/Aufgaben) preferences — per-user comfort settings
 * that adapt the module to one's own workflow (personal/user scope, see
 * ModuleSettingsShell).
 *
 * Server sync (settings foundation, Migr. 138, GET/PUT /settings/work/user):
 *   - `initFromServer()` once per session hydrates from the backend (fired by
 *     useHydrateModuleSettings on app-shell mount); falls back to local defaults.
 *   - Each setter writes through to the user-scope endpoint. localStorage stays
 *     as the optimistic cache so the UI is instant and survives offline.
 */
const MODULE_ID = 'work'

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
  /** Whether the store has hydrated from the backend at least once this session. */
  serverInitialized: boolean
  setDefaultView: (v: WorkView) => void
  setMyTasksGroupBy: (g: MyTasksGroupBy) => void
  setDensity: (d: WorkDensity) => void
  setDefaultProjectId: (id: string | null) => void
  /** Hydrate from GET /settings/work/user (once per session). */
  initFromServer: () => Promise<void>
}

/** The user-persisted keys, extracted as the PUT payload. */
function userPayload(s: WorkPrefsState): Record<string, unknown> {
  return {
    defaultView: s.defaultView,
    myTasksGroupBy: s.myTasksGroupBy,
    density: s.density,
    defaultProjectId: s.defaultProjectId,
  }
}

const WORK_VIEWS: WorkView[] = ['kanban', 'list', 'gantt']
const GROUP_BYS: MyTasksGroupBy[] = ['project', 'priority', 'dueDate', 'status']

export const useWorkPrefsStore = create<WorkPrefsState>()(
  persist(
    (set, get) => ({
      defaultView: 'kanban',
      myTasksGroupBy: 'project',
      density: 'comfortable',
      defaultProjectId: null,
      serverInitialized: false,
      setDefaultView: (defaultView) => {
        set({ defaultView })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      setMyTasksGroupBy: (myTasksGroupBy) => {
        set({ myTasksGroupBy })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      setDensity: (density) => {
        set({ density })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      setDefaultProjectId: (defaultProjectId) => {
        set({ defaultProjectId })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      initFromServer: async () => {
        if (get().serverInitialized) return
        const map = await loadModuleSettings(MODULE_ID, 'user')
        set((s) => ({
          defaultView: WORK_VIEWS.includes(map.defaultView as WorkView)
            ? (map.defaultView as WorkView)
            : s.defaultView,
          myTasksGroupBy: GROUP_BYS.includes(map.myTasksGroupBy as MyTasksGroupBy)
            ? (map.myTasksGroupBy as MyTasksGroupBy)
            : s.myTasksGroupBy,
          density:
            map.density === 'comfortable' || map.density === 'compact'
              ? map.density
              : s.density,
          defaultProjectId:
            typeof map.defaultProjectId === 'string' || map.defaultProjectId === null
              ? (map.defaultProjectId as string | null)
              : s.defaultProjectId,
          serverInitialized: true,
        }))
      },
    }),
    { name: 'cosmi-work-prefs' },
  ),
)

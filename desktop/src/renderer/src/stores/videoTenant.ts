import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { loadModuleSettings, saveModuleSettings } from '@/api/settings-persist'

/**
 * Tenant-wide video/call defaults ("Für alle"-Scope, see ModuleSettingsShell).
 * Editable only by a Meetings-Modul-Leiter/Admin (the PUT is dropped server-side
 * for other roles; the panel gates the UI). Personal device prefs live in
 * stores/videoPrefs.ts.
 *
 * Server sync (settings foundation, GET/PUT /settings/meetings/tenant):
 *   - `initFromServer()` hydrates once per session (central hydrator).
 *   - Setters write through (tenant scope); localStorage is the optimistic cache.
 */
const MODULE_ID = 'meetings'

/** consent = record only with the participant's consent; always = auto; off = never. */
export type RecordingPolicy = 'consent' | 'always' | 'off'
const POLICIES: RecordingPolicy[] = ['consent', 'always', 'off']

interface VideoTenantState {
  /** Default recording behaviour for new calls/meetings. */
  recordingPolicy: RecordingPolicy
  /** Default room label pre-filled in new meetings (empty = none). */
  defaultRoom: string
  /** Whether the store has hydrated from the backend at least once this session. */
  serverInitialized: boolean
  setRecordingPolicy: (v: RecordingPolicy) => void
  setDefaultRoom: (v: string) => void
  /** Hydrate from GET /settings/meetings/tenant (once per session). */
  initFromServer: () => Promise<void>
}

function tenantPayload(s: VideoTenantState): Record<string, unknown> {
  return {
    recordingPolicy: s.recordingPolicy,
    defaultRoom: s.defaultRoom,
  }
}

export const useVideoTenantStore = create<VideoTenantState>()(
  persist(
    (set, get) => ({
      recordingPolicy: 'consent',
      defaultRoom: '',
      serverInitialized: false,
      setRecordingPolicy: (recordingPolicy) => {
        set({ recordingPolicy })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      setDefaultRoom: (defaultRoom) => {
        set({ defaultRoom })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      initFromServer: async () => {
        if (get().serverInitialized) return
        const map = await loadModuleSettings(MODULE_ID, 'tenant')
        set((s) => ({
          recordingPolicy: POLICIES.includes(map.recordingPolicy as RecordingPolicy)
            ? (map.recordingPolicy as RecordingPolicy)
            : s.recordingPolicy,
          defaultRoom: typeof map.defaultRoom === 'string' ? map.defaultRoom : s.defaultRoom,
          serverInitialized: true,
        }))
      },
    }),
    { name: 'cosmi-video-tenant' },
  ),
)

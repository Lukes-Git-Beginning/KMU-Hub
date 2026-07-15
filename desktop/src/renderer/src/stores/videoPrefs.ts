import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { loadModuleSettings, saveModuleSettings } from '@/api/settings-persist'

/**
 * Personal video/call preferences — per-user device defaults for the Video &
 * Anrufe module (personal/user scope, see ModuleSettingsShell). Backed by the
 * settings foundation (GET/PUT /settings/meetings/user): localStorage is the
 * optimistic cache, `initFromServer()` hydrates once per session (central
 * hydrator) and each setter writes through.
 */
const MODULE_ID = 'meetings'

export type VideoBackground = 'none' | 'blur' | 'office' | 'nature'
const BACKGROUNDS: VideoBackground[] = ['none', 'blur', 'office', 'nature']

interface VideoPrefsState {
  /** Preferred microphone (device option key). */
  audioInput: string
  /** Preferred speaker/output. */
  audioOutput: string
  /** Preferred camera. */
  videoInput: string
  /** Virtual-background choice. */
  background: VideoBackground
  /** Whether the store has hydrated from the backend at least once this session. */
  serverInitialized: boolean
  setAudioInput: (v: string) => void
  setAudioOutput: (v: string) => void
  setVideoInput: (v: string) => void
  setBackground: (v: VideoBackground) => void
  /** Hydrate from GET /settings/meetings/user (once per session). */
  initFromServer: () => Promise<void>
}

function userPayload(s: VideoPrefsState): Record<string, unknown> {
  return {
    audioInput: s.audioInput,
    audioOutput: s.audioOutput,
    videoInput: s.videoInput,
    background: s.background,
  }
}

export const useVideoPrefsStore = create<VideoPrefsState>()(
  persist(
    (set, get) => ({
      audioInput: 'default',
      audioOutput: 'default',
      videoInput: 'default',
      background: 'none',
      serverInitialized: false,
      setAudioInput: (audioInput) => {
        set({ audioInput })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      setAudioOutput: (audioOutput) => {
        set({ audioOutput })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      setVideoInput: (videoInput) => {
        set({ videoInput })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      setBackground: (background) => {
        set({ background })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      initFromServer: async () => {
        if (get().serverInitialized) return
        const map = await loadModuleSettings(MODULE_ID, 'user')
        set((s) => ({
          audioInput: typeof map.audioInput === 'string' ? map.audioInput : s.audioInput,
          audioOutput: typeof map.audioOutput === 'string' ? map.audioOutput : s.audioOutput,
          videoInput: typeof map.videoInput === 'string' ? map.videoInput : s.videoInput,
          background: BACKGROUNDS.includes(map.background as VideoBackground)
            ? (map.background as VideoBackground)
            : s.background,
          serverInitialized: true,
        }))
      },
    }),
    { name: 'cosmi-video-prefs' },
  ),
)

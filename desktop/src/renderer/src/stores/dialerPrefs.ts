import { create } from 'zustand'
import { persist } from 'zustand/middleware'

/**
 * Dialer settings store (D-5). Holds both the personal comfort prefs (wrap-up
 * time, auto-advance) and the tenant-wide defaults (max concurrent calls,
 * recording-consent default, default outcome). In demo mode everything persists
 * locally; in production the tenant block would resolve server-side and only a
 * Modul-Leiter could change it (gated by ModuleSettingsShell).
 */
interface DialerPrefsState {
  // ── Personal ──
  /** Seconds the wrap-up timer counts before auto-advance kicks in. */
  defaultWrapUpSeconds: number
  /** After completing wrap-up, load the next queued contact automatically. */
  autoAdvance: boolean

  // ── Tenant ──
  /** Max simultaneous live calls allowed per agent. */
  maxConcurrentCalls: number
  /** Whether the recording-consent toggle defaults to on for new calls. */
  recordingConsentDefault: boolean
  /** Outcome pre-selected in wrap-up (null = none). */
  defaultOutcomeId: string | null

  setDefaultWrapUpSeconds: (s: number) => void
  setAutoAdvance: (v: boolean) => void
  setMaxConcurrentCalls: (n: number) => void
  setRecordingConsentDefault: (v: boolean) => void
  setDefaultOutcomeId: (id: string | null) => void
}

export const useDialerPrefsStore = create<DialerPrefsState>()(
  persist(
    (set) => ({
      defaultWrapUpSeconds: 30,
      autoAdvance: false,
      maxConcurrentCalls: 1,
      recordingConsentDefault: true,
      defaultOutcomeId: null,
      setDefaultWrapUpSeconds: (defaultWrapUpSeconds) => set({ defaultWrapUpSeconds }),
      setAutoAdvance: (autoAdvance) => set({ autoAdvance }),
      setMaxConcurrentCalls: (maxConcurrentCalls) => set({ maxConcurrentCalls }),
      setRecordingConsentDefault: (recordingConsentDefault) => set({ recordingConsentDefault }),
      setDefaultOutcomeId: (defaultOutcomeId) => set({ defaultOutcomeId }),
    }),
    { name: 'cosmi-dialer-prefs' },
  ),
)

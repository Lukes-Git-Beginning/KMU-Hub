import { create } from 'zustand'
import { persist } from 'zustand/middleware'

/**
 * Tenant-wide zeiterfassung settings (tenant scope — only Modul-Leiter/admin).
 * Mock-first; backend persistence is a backend-gap (tenant_settings).
 */
export type HolidayRegion = 'DE' | 'AT' | 'CH'
export type RoundingMode = 'none' | '5' | '15'

export const HOLIDAY_REGIONS: HolidayRegion[] = ['DE', 'AT', 'CH']
export const ROUNDING_MODES: RoundingMode[] = ['none', '5', '15']

interface ZeiterfassungSettingsState {
  /** Weekly target hours (full-time baseline). */
  weeklyTargetHours: number
  /** ArbZG: after this many hours a break becomes mandatory. */
  autoBreakAfterHours: number
  /** Auto-deducted break minutes once the threshold is crossed. */
  autoBreakMinutes: number
  /** Rounding of clock entries (minutes). */
  rounding: RoundingMode
  /** Public-holiday region for working-day calculations. */
  holidayRegion: HolidayRegion
  setWeeklyTargetHours: (h: number) => void
  setAutoBreakAfterHours: (h: number) => void
  setAutoBreakMinutes: (m: number) => void
  setRounding: (r: RoundingMode) => void
  setHolidayRegion: (r: HolidayRegion) => void
}

export const useZeiterfassungSettingsStore = create<ZeiterfassungSettingsState>()(
  persist(
    (set) => ({
      weeklyTargetHours: 40,
      autoBreakAfterHours: 6,
      autoBreakMinutes: 30,
      rounding: 'none',
      holidayRegion: 'DE',
      setWeeklyTargetHours: (weeklyTargetHours) => set({ weeklyTargetHours }),
      setAutoBreakAfterHours: (autoBreakAfterHours) => set({ autoBreakAfterHours }),
      setAutoBreakMinutes: (autoBreakMinutes) => set({ autoBreakMinutes }),
      setRounding: (rounding) => set({ rounding }),
      setHolidayRegion: (holidayRegion) => set({ holidayRegion }),
    }),
    { name: 'cosmi-zeiterfassung-settings' },
  ),
)

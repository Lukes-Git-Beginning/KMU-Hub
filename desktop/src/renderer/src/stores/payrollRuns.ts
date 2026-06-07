import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { PayrollTarget } from '@/stores/payrollSettings'

/**
 * Payroll runs (Lohnläufe) — history + period lock for the Lohnvorbereitung.
 *
 * MOCK-FIRST. A run is the monthly handover of prepared master + movement data
 * to DATEV. Locking a period freezes it for the handover (immutable). Backend:
 * payroll_runs (period, group, status, exported_at) + actual file generation —
 * see backend-gaps.md + team-datev-lohn-spec.md.
 */
export interface PayrollRun {
  id: string
  /** YYYY-MM */
  period: string
  group: string
  employeeCount: number
  target: PayrollTarget
  exportedAt: string
}

/** Lock key: `${period}|${groupId}`. */
function lockKey(period: string, groupId: string): string {
  return `${period}|${groupId}`
}

interface PayrollRunsState {
  runs: PayrollRun[]
  locked: string[]
  isLocked: (period: string, groupId: string) => boolean
  lock: (period: string, groupId: string) => void
  unlock: (period: string, groupId: string) => void
  recordExport: (run: Omit<PayrollRun, 'id' | 'exportedAt'>) => void
}

export const usePayrollRunsStore = create<PayrollRunsState>()(
  persist(
    (set, get) => ({
      runs: [],
      locked: [],
      isLocked: (period, groupId) => get().locked.includes(lockKey(period, groupId)),
      lock: (period, groupId) =>
        set((s) => {
          const k = lockKey(period, groupId)
          return s.locked.includes(k) ? s : { locked: [...s.locked, k] }
        }),
      unlock: (period, groupId) =>
        set((s) => ({ locked: s.locked.filter((k) => k !== lockKey(period, groupId)) })),
      recordExport: (run) =>
        set((s) => ({
          runs: [
            { ...run, id: crypto.randomUUID(), exportedAt: new Date().toISOString() },
            ...s.runs,
          ],
        })),
    }),
    { name: 'cosmi-payroll-runs' },
  ),
)

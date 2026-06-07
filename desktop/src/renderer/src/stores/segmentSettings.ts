import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { DEFAULT_SEGMENT_THRESHOLDS, type SegmentThresholds } from '@/lib/segments'

/**
 * Tenant-wide Mandanten-Segment thresholds (CRM settings, "Für alle").
 * MOCK-FIRST. Backend: tenant_settings (module_id='crm', key='segment.*') —
 * see .planning/backend-gaps.md (P8).
 */
interface SegmentSettingsState {
  thresholds: SegmentThresholds
  setThreshold: (key: keyof SegmentThresholds, value: number) => void
}

export const useSegmentSettingsStore = create<SegmentSettingsState>()(
  persist(
    (set) => ({
      thresholds: DEFAULT_SEGMENT_THRESHOLDS,
      setThreshold: (key, value) =>
        set((s) => ({ thresholds: { ...s.thresholds, [key]: value } })),
    }),
    { name: 'cosmi-crm-segment-thresholds' },
  ),
)

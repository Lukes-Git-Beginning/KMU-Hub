import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { MandantSegment } from '@/lib/segments'

/**
 * Manual A/B/C segment overrides per contact (frontend mock-first).
 *
 * The base segment is rule-based (revenue potential, see lib/segments). An
 * advisor can override it for a specific contact (exceptions the rule can't
 * capture). The effective segment = override ?? computed.
 *
 * Backend target: contacts.segment_override — see .planning/backend-gaps.md.
 */
interface SegmentOverrideState {
  overrides: Record<string, MandantSegment>
  get: (contactId: string) => MandantSegment | undefined
  set: (contactId: string, segment: MandantSegment | null) => void
}

export const useSegmentOverrideStore = create<SegmentOverrideState>()(
  persist(
    (set, get) => ({
      overrides: {},
      get: (contactId) => get().overrides[contactId],
      set: (contactId, segment) =>
        set((s) => {
          const next = { ...s.overrides }
          if (segment === null) delete next[contactId]
          else next[contactId] = segment
          return { overrides: next }
        }),
    }),
    { name: 'cosmi-crm-segment-overrides' },
  ),
)

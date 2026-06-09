import { create } from 'zustand'
import { persist } from 'zustand/middleware'

/**
 * Configurable lead-scoring rules (frontend mock-first).
 *
 * Replaces the previously hardcoded scoring in api/hooks/useLeads.ts so a CRM
 * lead can set source/field point weights and the hot/warm thresholds — the
 * market-standard pattern (HubSpot/Pipedrive let users configure score rules).
 * Defaults reproduce the former hardcoded behaviour exactly.
 *
 * Backend target: tenant_settings (module_id='crm', key='leadScoring.*') —
 * see .planning/backend-gaps.md.
 */
export interface LeadScoringConfig {
  /** Base points per lead source. */
  sourceBase: { dialer: number; manual: number; csv: number }
  /** Points added when the respective field is filled. */
  fieldPoints: { email: number; phone: number; company: number; notes: number }
  /** Score thresholds: >= hot → hot, >= warm → warm, else cold. */
  thresholds: { hot: number; warm: number }
}

export const DEFAULT_LEAD_SCORING: LeadScoringConfig = {
  sourceBase: { dialer: 35, manual: 25, csv: 10 },
  fieldPoints: { email: 20, phone: 15, company: 20, notes: 10 },
  thresholds: { hot: 66, warm: 33 },
}

interface LeadScoringState {
  config: LeadScoringConfig
  setConfig: (patch: Partial<LeadScoringConfig>) => void
  reset: () => void
}

export const useLeadScoringStore = create<LeadScoringState>()(
  persist(
    (set) => ({
      config: DEFAULT_LEAD_SCORING,
      setConfig: (patch) => set((s) => ({ config: { ...s.config, ...patch } })),
      reset: () => set({ config: DEFAULT_LEAD_SCORING }),
    }),
    { name: 'cosmi-crm-lead-scoring' },
  ),
)
